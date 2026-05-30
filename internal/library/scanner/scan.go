package scanner

import (
	"context"
	"errors"
	"os"
	"seanime/internal/api/anilist"
	"seanime/internal/api/metadata"
	"seanime/internal/database/models"
	"seanime/internal/events"
	"seanime/internal/hook"
	"seanime/internal/library/anime"
	"seanime/internal/library/filesystem"
	"seanime/internal/library/summary"
	"seanime/internal/platforms/platform"
	"seanime/internal/util"
	"seanime/internal/util/limiter"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
)

type Scanner struct {
	DirPath            string
	OtherDirPaths      []string
	Platform           platform.Platform
	Logger             *zerolog.Logger
	WSEventManager     events.WSEventManagerInterface
	ExistingLocalFiles []*anime.LocalFile
	SkipIgnoredFiles   bool
	ScanSummaryLogger  *summary.ScanSummaryLogger
	ScanLogger         *ScanLogger
	MetadataProvider   metadata.Provider
	MatchingThreshold  float64
	MatchingAlgorithm  string
	Database           DatabaseInterface // For global media pool
	UserID             uint              // For user subscription tracking
}

// Scan will scan the directory and return a list of anime.LocalFile.
func (scn *Scanner) Scan(ctx context.Context) (lfs []*anime.LocalFile, err error) {
	defer util.HandlePanicWithError(&err)

	go func() {
		anime.EpisodeCollectionFromLocalFilesCache.Clear()
	}()

	scn.WSEventManager.SendEvent(events.EventScanProgress, 0)
	scn.WSEventManager.SendEvent(events.EventScanStatus, "Retrieving local files...")

	completeAnimeCache := anilist.NewCompleteAnimeCache()

	// Create a new Anilist rate limiter
	anilistRateLimiter := limiter.NewAnilistLimiter()

	if scn.ScanSummaryLogger == nil {
		scn.ScanSummaryLogger = summary.NewScanSummaryLogger()
	}

	scn.Logger.Debug().Msg("scanner: Starting scan")
	scn.WSEventManager.SendEvent(events.EventScanProgress, 10)
	scn.WSEventManager.SendEvent(events.EventScanStatus, "Retrieving local files...")

	startTime := time.Now()

	// Invoke ScanStarted hook
	event := &ScanStartedEvent{
		LibraryPath:       scn.DirPath,
		OtherLibraryPaths: scn.OtherDirPaths,
		SkipIgnored:       scn.SkipIgnoredFiles,
		LocalFiles:        scn.ExistingLocalFiles,
	}
	_ = hook.GlobalHookManager.OnScanStarted().Trigger(event)
	scn.DirPath = event.LibraryPath
	scn.OtherDirPaths = event.OtherLibraryPaths
	scn.SkipIgnoredFiles = event.SkipIgnored

	// Default prevented, return the local files
	if event.DefaultPrevented {
		// Invoke ScanCompleted hook
		completedEvent := &ScanCompletedEvent{
			LocalFiles: event.LocalFiles,
			Duration:   int(time.Since(startTime).Milliseconds()),
		}
		hook.GlobalHookManager.OnScanCompleted().Trigger(completedEvent)

		return completedEvent.LocalFiles, nil
	}

	// +---------------------+
	// |     File paths      |
	// +---------------------+

	libraryPaths := append([]string{scn.DirPath}, scn.OtherDirPaths...)

	// Create a map of local file paths used to avoid duplicates
	retrievedPathMap := make(map[string]struct{})

	paths := make([]string, 0)
	mu := sync.Mutex{}
	logMu := sync.Mutex{}
	wg := sync.WaitGroup{}

	wg.Add(len(libraryPaths))

	// Get local files from all directories
	for i, dirPath := range libraryPaths {
		go func(dirPath string, i int) {
			defer wg.Done()
			retrievedPaths, err := filesystem.GetMediaFilePathsFromDirS(dirPath)
			if err != nil {
				scn.Logger.Error().Msgf("scanner: An error occurred while retrieving local files from directory: %s", err)
				return
			}

			if scn.ScanLogger != nil {
				logMu.Lock()
				if i == 0 {
					scn.ScanLogger.logger.Info().
						Any("count", len(paths)).
						Msgf("Retrieved file paths from main directory: %s", dirPath)
				} else {
					scn.ScanLogger.logger.Info().
						Any("count", len(retrievedPaths)).
						Msgf("Retrieved file paths from other directory: %s", dirPath)
				}
				logMu.Unlock()
			}

			for _, path := range retrievedPaths {
				if _, ok := retrievedPathMap[util.NormalizePath(path)]; !ok {
					mu.Lock()
					paths = append(paths, path)
					mu.Unlock()
				}
			}
		}(dirPath, i)
	}

	wg.Wait()

	if scn.ScanLogger != nil {
		scn.ScanLogger.logger.Info().
			Any("count", len(paths)).
			Msg("Retrieved file paths from all directories")
	}

	// Invoke ScanFilePathsRetrieved hook
	fpEvent := &ScanFilePathsRetrievedEvent{
		FilePaths: paths,
	}
	_ = hook.GlobalHookManager.OnScanFilePathsRetrieved().Trigger(fpEvent)
	paths = fpEvent.FilePaths

	// +---------------------+
	// |    Local files      |
	// +---------------------+

	localFiles := make([]*anime.LocalFile, 0)

	// Get skipped files depending on options
	skippedLfs := make(map[string]*anime.LocalFile)
	if scn.ExistingLocalFiles != nil {
		// Skip files that already have a global mapping (MediaId != 0) or are ignored
		for _, lf := range scn.ExistingLocalFiles {
			if lf.MediaId != 0 {
				skippedLfs[lf.GetNormalizedPath()] = lf
			} else if scn.SkipIgnoredFiles && lf.IsIgnored() {
				skippedLfs[lf.GetNormalizedPath()] = lf
			}
		}
	}

	// Create local files from paths (skipping skipped files)
	localFiles = lop.Map(paths, func(path string, _ int) *anime.LocalFile {
		if _, ok := skippedLfs[util.NormalizePath(path)]; !ok {
			// Create a new local file
			return anime.NewLocalFileS(path, libraryPaths)
		} else {
			return nil
		}
	})

	// Remove nil values
	localFiles = lo.Filter(localFiles, func(lf *anime.LocalFile, _ int) bool {
		return lf != nil
	})

	// Invoke ScanLocalFilesParsed hook
	parsedEvent := &ScanLocalFilesParsedEvent{
		LocalFiles: localFiles,
	}
	_ = hook.GlobalHookManager.OnScanLocalFilesParsed().Trigger(parsedEvent)
	localFiles = parsedEvent.LocalFiles

	if scn.ScanLogger != nil {
		scn.ScanLogger.logger.Debug().
			Any("count", len(localFiles)).
			Msg("Local files to be scanned")
		scn.ScanLogger.logger.Debug().
			Any("count", len(skippedLfs)).
			Msg("Skipped files")

		scn.ScanLogger.logger.Debug().
			Msg("===========================================================================================================")
	}

	for _, lf := range localFiles {
		if scn.ScanLogger != nil {
			scn.ScanLogger.logger.Trace().
				Str("path", lf.Path).
				Str("filename", lf.Name).
				Interface("parsedData", lf.ParsedData).
				Interface("parsedFolderData", lf.ParsedFolderData).
				Msg("Parsed local file")
		}
	}

	if scn.ScanLogger != nil {
		scn.ScanLogger.logger.Debug().
			Msg("===========================================================================================================")
	}

	// DEVNOTE: Removed library path checking because it causes some issues with symlinks

	// +---------------------+
	// |  No files to scan   |
	// +---------------------+

	// If there are no local files to scan (all files are skipped, or a file was deleted)
	if len(localFiles) == 0 {
		scn.WSEventManager.SendEvent(events.EventScanProgress, 90)
		scn.WSEventManager.SendEvent(events.EventScanStatus, "Verifying file integrity...")
		// Add skipped files
		if len(skippedLfs) > 0 {
			for _, sf := range skippedLfs {
				if filesystem.FileExists(sf.Path) { // Verify that the file still exists
					localFiles = append(localFiles, sf)
				}
			}
		}
		scn.Logger.Debug().Msg("scanner: Scan completed")
		scn.WSEventManager.SendEvent(events.EventScanProgress, 100)
		scn.WSEventManager.SendEvent(events.EventScanStatus, "Scan completed")

		// Invoke ScanCompleted hook
		completedEvent := &ScanCompletedEvent{
			LocalFiles: localFiles,
			Duration:   int(time.Since(startTime).Milliseconds()),
		}
		hook.GlobalHookManager.OnScanCompleted().Trigger(completedEvent)
		localFiles = completedEvent.LocalFiles

		return localFiles, nil
	}

	scn.WSEventManager.SendEvent(events.EventScanProgress, 20)
	scn.WSEventManager.SendEvent(events.EventScanStatus, "Fetching media...")

	// +---------------------+
	// |    MediaFetcher     |
	// +---------------------+

	// Fetch media needed for matching
	mf, err := NewMediaFetcher(ctx, &MediaFetcherOptions{
		Platform:               scn.Platform,
		MetadataProvider:       scn.MetadataProvider,
		LocalFiles:             localFiles,
		CompleteAnimeCache:     completeAnimeCache,
		Logger:                 scn.Logger,
		AnilistRateLimiter:     anilistRateLimiter,
		DisableAnimeCollection: false,
		ScanLogger:             scn.ScanLogger,
		Database:               scn.Database, // Pass database for global media pool
		UserID:                 scn.UserID,   // For user subscription tracking
	})
	if err != nil {
		return nil, err
	}

	scn.WSEventManager.SendEvent(events.EventScanProgress, 40)
	scn.WSEventManager.SendEvent(events.EventScanStatus, "Matching local files...")

	// +---------------------+
	// |   MediaContainer    |
	// +---------------------+

	// Create a new container for media
	mc := NewMediaContainer(&MediaContainerOptions{
		AllMedia:   mf.AllMedia,
		ScanLogger: scn.ScanLogger,
	})

	scn.Logger.Debug().
		Any("count", len(mc.NormalizedMedia)).
		Msg("media container: Media container created")

	// +---------------------+
	// |      Matcher        |
	// +---------------------+

	// Create a new matcher
	matcher := &Matcher{
		LocalFiles:         localFiles,
		MediaContainer:     mc,
		CompleteAnimeCache: completeAnimeCache,
		Logger:             scn.Logger,
		ScanLogger:         scn.ScanLogger,
		ScanSummaryLogger:  scn.ScanSummaryLogger,
		Algorithm:          scn.MatchingAlgorithm,
		Threshold:          scn.MatchingThreshold,
	}

	scn.WSEventManager.SendEvent(events.EventScanProgress, 60)

	err = matcher.MatchLocalFilesWithMedia()
	if err != nil {
		// If the matcher received no local files, return an error
		if errors.Is(err, ErrNoLocalFiles) {
			scn.Logger.Debug().Msg("scanner: Scan completed")
			scn.WSEventManager.SendEvent(events.EventScanProgress, 100)
			scn.WSEventManager.SendEvent(events.EventScanStatus, "Scan completed")
		}
		return nil, err
	}

	scn.WSEventManager.SendEvent(events.EventScanProgress, 70)
	scn.WSEventManager.SendEvent(events.EventScanStatus, "Hydrating metadata...")

	// +---------------------+
	// |    FileHydrator     |
	// +---------------------+

	// Create a new hydrator
	hydrator := &FileHydrator{
		AllMedia:           mc.NormalizedMedia,
		LocalFiles:         localFiles,
		MetadataProvider:   scn.MetadataProvider,
		Platform:           scn.Platform,
		CompleteAnimeCache: completeAnimeCache,
		AnilistRateLimiter: anilistRateLimiter,
		Logger:             scn.Logger,
		ScanLogger:         scn.ScanLogger,
		ScanSummaryLogger:  scn.ScanSummaryLogger,
	}
	hydrator.HydrateMetadata()

	scn.WSEventManager.SendEvent(events.EventScanProgress, 80)

	// +---------------------+
	// |  Add missing media  |
	// +---------------------+

	// Add non-added media entries to AniList collection
	// Max of 4 to avoid rate limit issues
	if len(mf.UnknownMediaIds) < 5 {
		scn.WSEventManager.SendEvent(events.EventScanStatus, "Adding missing media to AniList...")

		if err = scn.Platform.AddMediaToCollection(ctx, mf.UnknownMediaIds); err != nil {
			scn.Logger.Warn().Msg("scanner: An error occurred while adding media to planning list: " + err.Error())
		}
	}

	scn.WSEventManager.SendEvent(events.EventScanProgress, 90)
	scn.WSEventManager.SendEvent(events.EventScanStatus, "Verifying file integrity...")

	// Hydrate the summary logger before merging files
	scn.ScanSummaryLogger.HydrateData(localFiles, mc.NormalizedMedia, mf.AnimeCollectionWithRelations)

	// +---------------------+
	// |    Merge files      |
	// +---------------------+

	// Merge skipped files with scanned files
	// Only files that exist (this removes deleted/moved files)
	if len(skippedLfs) > 0 {
		wg := sync.WaitGroup{}
		mu := sync.Mutex{}
		wg.Add(len(skippedLfs))
		for _, skippedLf := range skippedLfs {
			go func(skippedLf *anime.LocalFile) {
				defer wg.Done()
				if filesystem.FileExists(skippedLf.Path) {
					mu.Lock()
					localFiles = append(localFiles, skippedLf)
					mu.Unlock()
				}
			}(skippedLf)
		}
		wg.Wait()
	}

	// +---------------------+
	// |  Save to Global DB  |
	// +---------------------+

	// Save successful matches to global database for server-wide reuse
	if scn.Database != nil {
		err = scn.saveMatchesToGlobalDatabase(localFiles)
		if err != nil {
			scn.Logger.Error().Err(err).Msg("scanner: Failed to save matches to global database")
		}
	}

	scn.Logger.Info().Msg("scanner: Scan completed")
	scn.WSEventManager.SendEvent(events.EventScanProgress, 100)
	scn.WSEventManager.SendEvent(events.EventScanStatus, "Scan completed")

	if scn.ScanLogger != nil {
		scn.ScanLogger.logger.Info().
			Int("count", len(localFiles)).
			Int("unknownMediaCount", len(mf.UnknownMediaIds)).
			Msg("Scan completed")
	}

	// Invoke ScanCompleted hook
	completedEvent := &ScanCompletedEvent{
		LocalFiles: localFiles,
		Duration:   int(time.Since(startTime).Milliseconds()),
	}
	hook.GlobalHookManager.OnScanCompleted().Trigger(completedEvent)
	localFiles = completedEvent.LocalFiles

	return localFiles, nil
}

// saveMatchesToGlobalDatabase saves successful LocalFile to AniList matches to the global database
// This populates the server-wide media pool for use by all users
func (scn *Scanner) saveMatchesToGlobalDatabase(localFiles []*anime.LocalFile) error {
	if scn.Database == nil {
		return nil
	}

	successfulMatches := 0
	skippedDuplicates := 0

	for _, lf := range localFiles {
		// Only save files that have been successfully matched to AniList media
		if lf.MediaId == 0 {
			continue
		}

		// Check for duplicate by looking for the same file path through database adapter
		isDuplicate := false
		if adapter, ok := scn.Database.(*DatabaseAdapter); ok {
			// Check if this file path already exists in global mappings
			_, checkErr := adapter.db.GetGlobalMapping(lf.Path)
			if checkErr == nil {
				// Mapping already exists
				isDuplicate = true
			}
		}

		if isDuplicate {
			skippedDuplicates++
			continue
		}

		// Save to database using the adapter
		saveErr := scn.saveGlobalMapping(lf)
		if saveErr != nil {
			scn.Logger.Warn().Err(saveErr).
				Str("filePath", lf.Path).
				Int("aniListID", lf.MediaId).
				Msg("scanner: Failed to save global mapping")
			continue
		}

		successfulMatches++
	}

	if scn.ScanLogger != nil {
		scn.ScanLogger.logger.Info().
			Int("saved", successfulMatches).
			Int("skipped", skippedDuplicates).
			Msg("Saved successful matches to global database")
	}

	return nil
}

// saveGlobalMapping saves a single mapping to the global database
func (scn *Scanner) saveGlobalMapping(lf *anime.LocalFile) error {
	// Helper functions to safely extract data from LocalFile
	getTitle := func() string {
		if lf.ParsedData != nil && lf.ParsedData.Title != "" {
			return lf.ParsedData.Title
		}
		return lf.Name
	}

	getYear := func() int {
		if lf.ParsedData != nil && lf.ParsedData.Year != "" {
			if year, err := strconv.Atoi(lf.ParsedData.Year); err == nil {
				return year
			}
		}
		return 0
	}

	getEpisodeNumber := func() int {
		if lf.Metadata != nil && lf.Metadata.Episode > 0 {
			return lf.Metadata.Episode
		}
		if lf.ParsedData != nil && lf.ParsedData.Episode != "" {
			if ep, err := strconv.Atoi(lf.ParsedData.Episode); err == nil {
				return ep
			}
		}
		return 0
	}

	getFileSize := func() int64 {
		if info, err := os.Stat(lf.Path); err == nil {
			return info.Size()
		}
		return 0
	}

	// Create a models.GlobalAnimeFileMapping from our data
	dbMapping := &models.GlobalAnimeFileMapping{
		AniListID:     lf.MediaId,
		LocalFilePath: lf.Path,
		Title:         getTitle(),
		Year:          getYear(),
		EpisodeNumber: getEpisodeNumber(),
		FileSize:      getFileSize(),
		LastScanned:   time.Now(),
		// Note: MappedByUserID could be added if we have access to current user context
	}

	// Save through database adapter
	if adapter, ok := scn.Database.(*DatabaseAdapter); ok {
		return adapter.db.CreateGlobalMapping(dbMapping)
	}

	return errors.New("database adapter not properly configured")
}
