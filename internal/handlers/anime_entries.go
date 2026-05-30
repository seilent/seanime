package handlers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"seanime/internal/api/anilist"
	"seanime/internal/database/db_bridge"
	"seanime/internal/database/models"
	"seanime/internal/hook"
	"seanime/internal/library/anime"
	"seanime/internal/library/filesystem"
	"seanime/internal/library/scanner"
	"seanime/internal/library/summary"
	"seanime/internal/platforms/platform"
	"seanime/internal/util"
	"seanime/internal/util/limiter"
	"seanime/internal/util/result"
	"slices"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
	"gorm.io/gorm"
)

// HandleGetAnimeEntry
//
//	@summary return a media entry for the given AniList anime media id.
//	@desc This is used by the anime media entry pages to get all the data about the anime.
//	@desc This includes episodes and metadata (if any), AniList list data, download info...
//	@route /api/v1/library/anime-entry/{id} [GET]
//	@param id - int - true - "AniList anime media ID"
//	@returns anime.Entry
func (h *Handler) HandleGetAnimeEntry(c echo.Context) error {

	// Get the current authenticated user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not authenticated"))
	}

	mId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Get all the local files from global mappings table
	lfs, _, err := db_bridge.GetLocalFilesFromGlobalMappings(h.App.Database)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Get the user's anilist collection
	animeCollection, err := h.App.GetAnimeCollectionForUser(user, false)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	if animeCollection == nil {
		return h.RespondWithError(c, errors.New("anime collection not found"))
	}

	// Get user-specific platform
	userPlatform, err := h.GetUserPlatform(c)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Create a new media entry
	entry, err := anime.NewEntry(c.Request().Context(), &anime.NewEntryOptions{
		MediaId:          mId,
		LocalFiles:       lfs,
		AnimeCollection:  animeCollection,
		Platform:         userPlatform,
		MetadataProvider: h.App.MetadataProvider,
		IsSimulated:      false, // Pure multiuser system - no simulated users
	})
	if err != nil {
		return h.RespondWithError(c, err)
	}

	fillerEvent := new(anime.AnimeEntryFillerHydrationEvent)
	fillerEvent.Entry = entry
	err = hook.GlobalHookManager.OnAnimeEntryFillerHydration().Trigger(fillerEvent)
	if err != nil {
		return h.RespondWithError(c, err)
	}
	entry = fillerEvent.Entry

	if !fillerEvent.DefaultPrevented {
		h.App.FillerManager.HydrateFillerData(fillerEvent.Entry)
	}

	return h.RespondWithData(c, entry)
}

//----------------------------------------------------------------------------------------------------------------------

// HandleAnimeEntryBulkAction
//
//	@summary perform given action on all the local files for the given media id.
//	@desc This is used to unmatch or toggle the lock status of all the local files for a specific media entry
//	@desc The response is not used in the frontend. The client should just refetch the entire media entry data.
//	@route /api/v1/library/anime-entry/bulk-action [PATCH]
//	@returns []anime.LocalFile
func (h *Handler) HandleAnimeEntryBulkAction(c echo.Context) error {

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	type body struct {
		MediaId int    `json:"mediaId"`
		Action  string `json:"action"` // "unmatch" or "toggle-lock"
	}

	p := new(body)
	if err := c.Bind(p); err != nil {
		return h.RespondWithError(c, err)
	}

	switch p.Action {
	case "unmatch":
		// Set anilist_id=0 for all mappings with this media
		h.App.Database.Gorm().Model(&models.GlobalAnimeFileMapping{}).
			Where("anilist_id = ?", p.MediaId).
			Updates(map[string]interface{}{"anilist_id": 0, "ignored": false})
	case "toggle-lock":
	}

	// Return current local files
	lfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, lfs)

}

//----------------------------------------------------------------------------------------------------------------------

var (
	entriesSuggestionsCache = result.NewCache[string, []*anilist.BaseAnime]()
)

// HandleFetchAnimeEntrySuggestions
//
//	@summary returns a list of media suggestions for files in the given directory.
//	@desc This is used by the "Resolve unmatched media" feature to suggest media entries for the local files in the given directory.
//	@desc If some matches files are found in the directory, it will ignore them and base the suggestions on the remaining files.
//	@route /api/v1/library/anime-entry/suggestions [POST]
//	@returns []anilist.BaseAnime
func (h *Handler) HandleFetchAnimeEntrySuggestions(c echo.Context) error {

	// Get the current authenticated user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not authenticated"))
	}

	type body struct {
		Dir string `json:"dir"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	b.Dir = strings.ToLower(b.Dir)

	suggestions, found := entriesSuggestionsCache.Get(b.Dir)
	if found {
		return h.RespondWithData(c, suggestions)
	}

	// Retrieve the user's local files
	lfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Group local files by dir
	groupedLfs := lop.GroupBy(lfs, func(item *anime.LocalFile) string {
		return filepath.Dir(item.GetNormalizedPath())
	})

	selectedLfs, found := groupedLfs[b.Dir]
	if !found {
		return h.RespondWithError(c, errors.New("no local files found for selected directory"))
	}

	// Filter out local files that are already matched
	selectedLfs = lo.Filter(selectedLfs, func(item *anime.LocalFile, _ int) bool {
		return item.MediaId == 0
	})

	title := selectedLfs[0].GetParsedTitle()

	h.App.Logger.Info().Str("title", title).Msg("handlers: Fetching anime suggestions")

	res, err := anilist.ListAnimeM(
		lo.ToPtr(1),
		&title,
		lo.ToPtr(8),
		nil,
		[]*anilist.MediaStatus{lo.ToPtr(anilist.MediaStatusFinished), lo.ToPtr(anilist.MediaStatusReleasing), lo.ToPtr(anilist.MediaStatusCancelled), lo.ToPtr(anilist.MediaStatusHiatus)},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		h.App.Logger,
		h.App.GetUserAnilistTokenForUser(user),
	)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Cache the results
	entriesSuggestionsCache.Set(b.Dir, res.GetPage().GetMedia())

	return h.RespondWithData(c, res.GetPage().GetMedia())

}

//----------------------------------------------------------------------------------------------------------------------

// HandleAnimeEntryManualMatch
//
//	@summary matches un-matched local files in the given directory to the given media.
//	@desc It is used by the "Resolve unmatched media" feature to manually match local files to a specific media entry.
//	@desc Matching involves the use of scanner.FileHydrator. It will also lock the files.
//	@desc The response is not used in the frontend. The client should just refetch the entire library collection.
//	@route /api/v1/library/anime-entry/manual-match [POST]
//	@returns []anime.LocalFile
func (h *Handler) HandleAnimeEntryManualMatch(c echo.Context) error {

	type body struct {
		Paths   []string `json:"paths"`
		MediaId int      `json:"mediaId"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get current user and user-specific AniList platform
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	var userPlatform platform.Platform
	var err error
	userPlatform, err = h.GetUserPlatform(c)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	var animeCollectionWithRelations *anilist.AnimeCollectionWithRelations
	animeCollectionWithRelations, err = userPlatform.GetAnimeCollectionWithRelations(c.Request().Context())
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Retrieve the user's local files
	var lfs []*anime.LocalFile
	lfs, _, err = db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	compPaths := make(map[string]struct{})
	for _, p := range b.Paths {
		compPaths[util.NormalizePath(p)] = struct{}{}
	}

	selectedLfs := lo.Filter(lfs, func(item *anime.LocalFile, _ int) bool {
		_, found := compPaths[item.GetNormalizedPath()]
		return found && item.MediaId == 0
	})

	// Add the media id to the selected local files
	selectedLfs = lop.Map(selectedLfs, func(item *anime.LocalFile, _ int) *anime.LocalFile {
		item.MediaId = b.MediaId
		item.Ignored = false
		return item
	})

	// Get user-specific AniList platform (second instance)
	userPlatform, err = h.GetUserPlatform(c)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Get the media
	var media *anilist.BaseAnime
	media, err = userPlatform.GetAnime(c.Request().Context(), b.MediaId)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Create a slice of normalized media
	normalizedMedia := []*anime.NormalizedMedia{
		anime.NewNormalizedMedia(media),
	}

	scanLogger, err := scanner.NewScanLogger(h.App.Config.Logs.Dir)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Create scan summary logger
	scanSummaryLogger := summary.NewScanSummaryLogger()

	fh := scanner.FileHydrator{
		LocalFiles:         selectedLfs,
		CompleteAnimeCache: anilist.NewCompleteAnimeCache(),
		Platform:           userPlatform,
		MetadataProvider:   h.App.MetadataProvider,
		AnilistRateLimiter: limiter.NewAnilistLimiter(),
		Logger:             h.App.Logger,
		ScanLogger:         scanLogger,
		ScanSummaryLogger:  scanSummaryLogger,
		AllMedia:           normalizedMedia,
		ForceMediaId:       media.GetID(),
	}

	fh.HydrateMetadata()

	// Hydrate the summary logger before merging files
	fh.ScanSummaryLogger.HydrateData(selectedLfs, normalizedMedia, animeCollectionWithRelations)

	// Save the scan summary
	go func() {
		err = db_bridge.InsertScanSummary(h.App.Database, scanSummaryLogger.GenerateSummary())
	}()

	// Remove select local files from the database slice, we will add them (hydrated) later
	selectedPaths := lop.Map(selectedLfs, func(item *anime.LocalFile, _ int) string { return item.GetNormalizedPath() })
	lfs = lo.Filter(lfs, func(item *anime.LocalFile, _ int) bool {
		if slices.Contains(selectedPaths, item.GetNormalizedPath()) {
			return false
		}
		return true
	})

	// Event
	event := new(anime.AnimeEntryManualMatchBeforeSaveEvent)
	event.MediaId = b.MediaId
	event.Paths = b.Paths
	event.MatchedLocalFiles = selectedLfs
	err = hook.GlobalHookManager.OnAnimeEntryManualMatchBeforeSave().Trigger(event)
	if err != nil {
		return h.RespondWithError(c, fmt.Errorf("OnAnimeEntryManualMatchBeforeSave: %w", err))
	}

	// Default prevented, do not save the local files
	if event.DefaultPrevented {
		return h.RespondWithData(c, lfs)
	}

	// Upsert global mappings for matched files
	for _, lf := range event.MatchedLocalFiles {
		episodeNumber := 0
		fileType := "main"
		if lf.Metadata != nil {
			episodeNumber = lf.Metadata.Episode
			if lf.Metadata.Type != "" {
				fileType = string(lf.Metadata.Type)
			}
		}
		romajiTitle := ""
		if v := media.GetTitle().GetRomaji(); v != nil {
			romajiTitle = *v
		}
		englishTitle := ""
		if v := media.GetTitle().GetEnglish(); v != nil {
			englishTitle = *v
		}
		_ = h.App.Database.UpsertGlobalMapping(&models.GlobalAnimeFileMapping{
			AniListID:     b.MediaId,
			LocalFilePath: lf.Path,
			Title:         media.GetTitleSafe(),
			RomajiTitle:   romajiTitle,
			EnglishTitle:  englishTitle,
			EpisodeNumber: episodeNumber,
			FileType:      fileType,
		})
	}

	// Return all local files
	retLfs, _, _ := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	return h.RespondWithData(c, retLfs)
}

//----------------------------------------------------------------------------------------------------------------------

var missingEpisodesCache *anime.MissingEpisodes

// HandleGetMissingEpisodes
//
//	@summary returns a list of episodes missing from the user's library collection
//	@desc It detects missing episodes by comparing the user's AniList collection 'next airing' data with the local files.
//	@desc This route can be called multiple times, as it does not bypass the cache.
//	@route /api/v1/library/missing-episodes [GET]
//	@returns anime.MissingEpisodes
func (h *Handler) HandleGetMissingEpisodes(c echo.Context) error {
	// Get the current authenticated user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not authenticated"))
	}

	// Short-circuit when missing-episodes is disabled
	if anime.DisableMissingEpisodesCheck {
		return h.RespondWithData(c, &anime.MissingEpisodes{Episodes: []*anime.Episode{}, SilencedEpisodes: []*anime.Episode{}})
	}

	h.App.AddOnRefreshAnilistCollectionFunc("HandleGetMissingEpisodes", func() {
		missingEpisodesCache = nil
	})

	if missingEpisodesCache != nil {
		return h.RespondWithData(c, missingEpisodesCache)
	}

	// Get the user's anilist collection
	// Do not bypass the cache, since this handler might be called multiple times, and we don't want to spam the API
	// A cron job will refresh the cache every 10 minutes
	animeCollection, err := h.App.GetAnimeCollectionForUser(user, false)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	lfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Get the silenced media ids
	silencedMediaIds, _ := h.App.Database.GetSilencedMediaEntryIds()

	missingEps := anime.NewMissingEpisodes(&anime.NewMissingEpisodesOptions{
		AnimeCollection:  animeCollection,
		LocalFiles:       lfs,
		SilencedMediaIds: silencedMediaIds,
		MetadataProvider: h.App.MetadataProvider,
	})

	event := new(anime.MissingEpisodesEvent)
	event.MissingEpisodes = missingEps
	err = hook.GlobalHookManager.OnMissingEpisodes().Trigger(event)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	missingEpisodesCache = event.MissingEpisodes

	return h.RespondWithData(c, event.MissingEpisodes)
}

//----------------------------------------------------------------------------------------------------------------------

// HandleGetAnimeEntrySilenceStatus
//
//	@summary returns the silence status of a media entry.
//	@param id - int - true - "The ID of the media entry."
//	@route /api/v1/library/anime-entry/silence/{id} [GET]
//	@returns models.SilencedMediaEntry
func (h *Handler) HandleGetAnimeEntrySilenceStatus(c echo.Context) error {
	mId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return h.RespondWithError(c, errors.New("invalid id"))
	}

	animeEntry, err := h.App.Database.GetSilencedMediaEntry(uint(mId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return h.RespondWithData(c, false)
		} else {
			return h.RespondWithError(c, err)
		}
	}

	return h.RespondWithData(c, animeEntry)
}

// HandleToggleAnimeEntrySilenceStatus
//
//	@summary toggles the silence status of a media entry.
//	@desc The missing episodes should be re-fetched after this.
//	@route /api/v1/library/anime-entry/silence [POST]
//	@returns bool
func (h *Handler) HandleToggleAnimeEntrySilenceStatus(c echo.Context) error {

	type body struct {
		MediaId int `json:"mediaId"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	animeEntry, err := h.App.Database.GetSilencedMediaEntry(uint(b.MediaId))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = h.App.Database.InsertSilencedMediaEntry(uint(b.MediaId))
			if err != nil {
				return h.RespondWithError(c, err)
			}
			return h.RespondWithData(c, true)
		} else {
			return h.RespondWithError(c, err)
		}
	}

	err = h.App.Database.DeleteSilencedMediaEntry(animeEntry.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

//-----------------------------------------------------------------------------------------------------------------------------

// HandleUpdateAnimeEntryProgress
//
//	@summary update the progress of the given anime media entry.
//	@desc This is used to update the progress of the given anime media entry on AniList.
//	@desc The response is not used in the frontend, the client should just refetch the entire media entry data.
//	@desc NOTE: This is currently only used by the 'Online streaming' feature since anime progress updates are handled by the Playback Manager.
//	@route /api/v1/library/anime-entry/update-progress [POST]
//	@returns bool
func (h *Handler) HandleUpdateAnimeEntryProgress(c echo.Context) error {

	// Get the current authenticated user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not authenticated"))
	}

	type body struct {
		MediaId       int `json:"mediaId"`
		MalId         int `json:"malId,omitempty"`
		EpisodeNumber int `json:"episodeNumber"`
		TotalEpisodes int `json:"totalEpisodes"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get user-specific AniList platform
	userPlatform, err := h.GetUserPlatform(c)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Update the progress on AniList
	err = userPlatform.UpdateEntryProgress(
		c.Request().Context(),
		b.MediaId,
		b.EpisodeNumber,
		&b.TotalEpisodes,
	)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	_, _ = h.App.RefreshAnimeCollectionForUser(user) // Refresh the AniList collection

	return h.RespondWithData(c, true)
}

//-----------------------------------------------------------------------------------------------------------------------------

// HandleUpdateAnimeEntryRepeat
//
//	@summary update the repeat value of the given anime media entry.
//	@desc This is used to update the repeat value of the given anime media entry on AniList.
//	@desc The response is not used in the frontend, the client should just refetch the entire media entry data.
//	@route /api/v1/library/anime-entry/update-repeat [POST]
//	@returns bool
func (h *Handler) HandleUpdateAnimeEntryRepeat(c echo.Context) error {

	type body struct {
		MediaId int `json:"mediaId"`
		Repeat  int `json:"repeat"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get user-specific AniList platform
	userPlatform, err := h.GetUserPlatform(c)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	err = userPlatform.UpdateEntryRepeat(
		c.Request().Context(),
		b.MediaId,
		b.Repeat,
	)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	//_, _ = h.App.RefreshAnimeCollectionForUser(user) // Refresh the AniList collection

	return h.RespondWithData(c, true)
}

//-----------------------------------------------------------------------------------------------------------------------------

// HandleValidateAnimeEntryLocalFiles
//
//	@summary validates and removes non-existent local files, and scans for new files for a specific media.
//	@desc This checks if the local files associated with the given media ID actually exist on disk.
//	@desc It also scans the media's directory to find any new files that aren't in the database yet.
//	@desc This is called automatically when opening an anime entry page to ensure data consistency.
//	@route /api/v1/library/anime-entry/validate-local-files [POST]
//	returns bool
func (h *Handler) HandleValidateAnimeEntryLocalFiles(c echo.Context) error {

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	type body struct {
		MediaId int `json:"mediaId"`
	}

	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	if b.MediaId == 0 {
		return h.RespondWithError(c, errors.New("invalid media id"))
	}

	// Get existing global mappings for this media
	existingMappings, err := h.App.Database.GetGlobalMappingsByAniListID(b.MediaId)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Step 1: Validate existing mappings and remove non-existent files
	removedCount := 0
	existingPathsMap := make(map[string]bool)

	for _, mapping := range existingMappings {
		existingPathsMap[mapping.LocalFilePath] = true
		if _, err := os.Stat(mapping.LocalFilePath); err != nil {
			// File doesn't exist, remove from database
			h.App.Logger.Debug().
				Str("path", mapping.LocalFilePath).
				Int("mediaId", b.MediaId).
				Msg("anime-entry: Removing non-existent local file from global mappings")
			err := h.App.Database.DeleteGlobalMapping(mapping.LocalFilePath)
			if err != nil {
				h.App.Logger.Error().Err(err).Msg("anime-entry: Failed to delete global mapping")
			}
			removedCount++
		}
	}

	h.App.Logger.Debug().
		Int("mediaId", b.MediaId).
		Int("totalFiles", len(existingMappings)).
		Int("removedCount", removedCount).
		Msg("anime-entry: File validation summary")

	// Step 2: Scan directories to find new files
	libraryPaths, err := h.App.Database.GetAllLibraryPathsFromSettings()
	if err != nil {
		h.App.Logger.Warn().Err(err).Msg("anime-entry: Failed to get library paths for scanning")
	}

	addedCount := 0
	if len(libraryPaths) > 0 {
		// Find the directory containing the media files
		mediaDir := ""

		// First, try to find directory from existing valid mappings
		for _, mapping := range existingMappings {
			if _, err := os.Stat(mapping.LocalFilePath); err == nil {
				mediaDir = filepath.Dir(mapping.LocalFilePath)
				h.App.Logger.Debug().Str("dir", mediaDir).Msg("anime-entry: Found directory from existing files")
				break
			}
		}

		// If no valid files, construct expected directory path from media title
		if mediaDir == "" {
			if userPlatform, err := h.GetUserPlatform(c); err == nil {
				media, err := userPlatform.GetAnime(c.Request().Context(), b.MediaId)
				if err == nil && media != nil {
					title := media.GetTitleSafe()
					if title == "" {
						title = media.GetPreferredTitle()
					}

					if title != "" {
						sanitizedTitle := strings.ReplaceAll(title, ":", " ")
						sanitizedTitle = strings.TrimSpace(sanitizedTitle)

						for _, libPath := range libraryPaths {
							expectedDir := filepath.Join(libPath, sanitizedTitle)
							if info, err := os.Stat(expectedDir); err == nil && info.IsDir() {
								mediaDir = expectedDir
								h.App.Logger.Debug().
									Str("dir", mediaDir).
									Str("title", title).
									Msg("anime-entry: Found directory from media title")
								break
							}
						}
					}
				}
			}
		}

		if mediaDir != "" {
			h.App.Logger.Debug().
				Int("mediaId", b.MediaId).
				Str("scanDir", mediaDir).
				Msg("anime-entry: Scanning directory for new files")

			// Scan this directory for new files
			allFilePaths, err := filesystem.GetMediaFilePathsFromDir(mediaDir)
			if err != nil {
				h.App.Logger.Warn().Err(err).Str("dir", mediaDir).Msg("anime-entry: Failed to scan directory")
			} else {
				// Find new files that aren't in the database
				for _, filePath := range allFilePaths {
					if !existingPathsMap[filePath] {
						// Create new LocalFile to extract metadata
						newLf := anime.NewLocalFile(filePath, libraryPaths[0])
						if newLf != nil {
							newLf.MediaId = b.MediaId

							// Extract metadata using FileHydrator
							if userPlatform, err := h.GetUserPlatform(c); err == nil {
								media, err := userPlatform.GetAnime(c.Request().Context(), b.MediaId)
								if err == nil && media != nil {
									normalizedMedia := []*anime.NormalizedMedia{
										anime.NewNormalizedMedia(media),
									}

									fh := &scanner.FileHydrator{
										LocalFiles:         []*anime.LocalFile{newLf},
										CompleteAnimeCache: anilist.NewCompleteAnimeCache(),
										Platform:           userPlatform,
										MetadataProvider:   h.App.MetadataProvider,
										Logger:             h.App.Logger,
										AllMedia:           normalizedMedia,
										ForceMediaId:       b.MediaId,
									}
									fh.HydrateMetadata()

									// Create global mapping from LocalFile
									episodeNumber := 0
									if newLf.Metadata != nil && newLf.Metadata.Episode > 0 {
										episodeNumber = newLf.Metadata.Episode
									} else if newLf.ParsedData != nil && newLf.ParsedData.Episode != "" {
										if ep, err := strconv.Atoi(newLf.ParsedData.Episode); err == nil {
											episodeNumber = ep
										}
									}

									// Create global mapping
									romajiTitle := ""
									if v := media.GetTitle().GetRomaji(); v != nil {
										romajiTitle = *v
									}
									englishTitle := ""
									if v := media.GetTitle().GetEnglish(); v != nil {
										englishTitle = *v
									}

									newMapping := &models.GlobalAnimeFileMapping{
										AniListID:     b.MediaId,
										LocalFilePath: filePath,
										Title:         media.GetTitleSafe(),
										RomajiTitle:   romajiTitle,
										EnglishTitle:  englishTitle,
										EpisodeNumber: episodeNumber,
									}

									// Upsert (not Create): the file may already have a mapping under a
									// different or zero anilist_id (ignored, or matched to another media),
									// which would violate the local_file_path UNIQUE constraint on INSERT.
									err = h.App.Database.UpsertGlobalMapping(newMapping)
									if err != nil {
										h.App.Logger.Error().Err(err).Str("path", filePath).Msg("anime-entry: Failed to create global mapping")
									} else {
										addedCount++
										existingPathsMap[filePath] = true

										h.App.Logger.Debug().
											Str("path", filePath).
											Int("mediaId", b.MediaId).
											Int("episode", episodeNumber).
											Msg("anime-entry: Added new local file")
									}
								}
							}
						}
					}
				}
			}
		}
	}

	h.App.Logger.Info().
		Int("mediaId", b.MediaId).
		Int("removedCount", removedCount).
		Int("addedCount", addedCount).
		Msg("anime-entry: Validated local files and updated database")

	return h.RespondWithData(c, true)
}
