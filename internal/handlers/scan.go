package handlers

import (
	"errors"
	"seanime/internal/api/anilist"
	"seanime/internal/database/db_bridge"
	"seanime/internal/library/anime"
	"seanime/internal/library/scanner"
	"seanime/internal/library/summary"
	"seanime/internal/platforms/anilist_platform"

	"github.com/labstack/echo/v4"
)

// HandleScanLocalFiles
//
//	@summary scans the user's library.
//	@desc This will scan the user's library.
//	@desc The response is ignored, the client should re-fetch the library after this.
//	@route /api/v1/library/scan [POST]
//	@returns []anime.LocalFile
func (h *Handler) HandleScanLocalFiles(c echo.Context) error {

	type body struct {
		Enhanced         bool `json:"enhanced"`
		SkipLockedFiles  bool `json:"skipLockedFiles"`
		SkipIgnoredFiles bool `json:"skipIgnoredFiles"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get current user for user-specific operations
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	// Retrieve the user's library path
	libraryPath, err := h.App.Database.GetLibraryPathFromSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}
	additionalLibraryPaths, err := h.App.Database.GetAdditionalLibraryPathsFromSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Get the latest local files for this user
	existingLfs, _, err := db_bridge.GetLocalFilesForUser(h.App.Database, user.ID)
	if err != nil {
		// If no local files exist for this user, start with empty slice
		existingLfs = []*anime.LocalFile{}
	}

	// +---------------------+
	// |       Scanner       |
	// +---------------------+

	// Create scan summary logger
	scanSummaryLogger := summary.NewScanSummaryLogger()

	// Create a new scan logger
	scanLogger, err := scanner.NewScanLogger(h.App.Config.Logs.Dir)
	if err != nil {
		return h.RespondWithError(c, err)
	}
	defer scanLogger.Done()

	// Create user-specific AniList token and platform
	token := h.App.GetUserAnilistTokenForUser(user)
	if token == "" {
		return h.RespondWithError(c, errors.New("user has no AniList connection"))
	}

	// Get user's AniList account to get the AniList username
	account, err := h.App.Database.GetAccountForUser(user.ID)
	if err != nil || account == nil || account.Username == "" {
		return h.RespondWithError(c, errors.New("user AniList account not found"))
	}

	client := anilist.NewAnilistClient(token)
	userPlatform := anilist_platform.NewAnilistPlatform(client, h.App.Logger)
	userPlatform.SetUsername(account.Username) // Use the stored AniList username

	// Create a new scanner
	sc := scanner.Scanner{
		DirPath:            libraryPath,
		OtherDirPaths:      additionalLibraryPaths,
		Enhanced:           b.Enhanced,
		Platform:           userPlatform,
		Logger:             h.App.Logger,
		WSEventManager:     h.App.WSEventManager,
		ExistingLocalFiles: existingLfs,
		SkipLockedFiles:    b.SkipLockedFiles,
		SkipIgnoredFiles:   b.SkipIgnoredFiles,
		ScanSummaryLogger:  scanSummaryLogger,
		ScanLogger:         scanLogger,
		MetadataProvider:   h.App.MetadataProvider,
		MatchingAlgorithm:  h.App.Settings.GetLibrary().ScannerMatchingAlgorithm,
		MatchingThreshold:  h.App.Settings.GetLibrary().ScannerMatchingThreshold,
	}

	// Scan the library
	allLfs, err := sc.Scan(c.Request().Context())
	if err != nil {
		if errors.Is(err, scanner.ErrNoLocalFiles) {
			return h.RespondWithData(c, []interface{}{})
		} else {
			return h.RespondWithError(c, err)
		}
	}

	// Insert the local files for this user
	lfs, err := db_bridge.InsertLocalFilesForUser(h.App.Database, allLfs, user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Save the scan summary
	_ = db_bridge.InsertScanSummary(h.App.Database, scanSummaryLogger.GenerateSummary())

	go h.App.AutoDownloader.CleanUpDownloadedItems()

	return h.RespondWithData(c, lfs)

}
