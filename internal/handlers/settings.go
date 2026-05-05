package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"seanime/internal/api/anilist"
	"seanime/internal/database/models"
	"seanime/internal/torrents/torrent"
	"seanime/internal/util"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
)

// HandleGetSettings
//
//	@summary returns the app settings.
//	@route /api/v1/settings [GET]
//	@returns models.Settings
func (h *Handler) HandleGetSettings(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return c.JSON(401, map[string]string{"error": "Authentication required"})
	}

	// Get user-specific settings
	userSettings, err := h.App.Database.GetSettingsForUser(user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Get global settings for library, torrent, auto-downloader
	globalSettings, err := h.App.Database.GetGlobalSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Populate virtual fields from GlobalSettings for frontend compatibility
	if globalSettings != nil {
		userSettings.Library = globalSettings.Library
		userSettings.Torrent = globalSettings.Torrent
		userSettings.AutoDownloader = globalSettings.AutoDownloader
	}

	return h.RespondWithData(c, userSettings)
}

// HandleGetGlobalSettings
//
//	@summary returns the global server settings (admin-only).
//	@desc Returns server-wide settings like library paths, torrent settings, etc.
//	@route /api/v1/settings/global [GET]
//	@returns models.GlobalSettings
func (h *Handler) HandleGetGlobalSettings(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.JSON(403, map[string]string{"error": "Admin access required"})
	}

	globalSettings, err := h.App.Database.GetGlobalSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, globalSettings)
}

// HandleUpdateGlobalSettings
//
//	@summary updates global server settings (admin-only).
//	@desc Updates server-wide settings like library paths, torrent settings, etc.
//	@route /api/v1/settings/global [PUT]
//	@returns models.GlobalSettings
func (h *Handler) HandleUpdateGlobalSettings(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.JSON(403, map[string]string{"error": "Admin access required"})
	}

	var settings models.GlobalSettings
	if err := c.Bind(&settings); err != nil {
		return h.RespondWithError(c, err)
	}

	updatedSettings, err := h.App.Database.UpsertGlobalSettings(&settings)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, updatedSettings)
}

// HandleGetUserSettings
//
//	@summary returns the current user's personal settings.
//	@desc Returns user-specific settings like media player preferences, display settings, etc.
//	@route /api/v1/settings/user [GET]
//	@returns models.Settings
func (h *Handler) HandleGetUserSettings(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return c.JSON(401, map[string]string{"error": "Authentication required"})
	}

	userSettings, err := h.App.Database.GetSettingsForUser(user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, userSettings)
}

// HandleUpdateUserSettings
//
//	@summary updates the current user's personal settings.
//	@desc Updates user-specific settings like media player preferences, display settings, etc.
//	@route /api/v1/settings/user [PUT]
//	@returns models.Settings
func (h *Handler) HandleUpdateUserSettings(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return c.JSON(401, map[string]string{"error": "Authentication required"})
	}

	var settings models.Settings
	if err := c.Bind(&settings); err != nil {
		return h.RespondWithError(c, err)
	}

	// Ensure the settings are for the current user
	settings.UserID = user.ID

	if err := h.App.Database.SaveSettingsForUser(user.ID, &settings); err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, settings)
}

// HandleGettingStarted
//
//	@summary updates the app settings.
//	@desc This will update the app settings.
//	@desc The client should re-fetch the server status after this.
//	@route /api/v1/start [POST]
//	@returns handlers.Status
func (h *Handler) HandleGettingStarted(c echo.Context) error {

	type body struct {
		Library         models.LibrarySettings      `json:"library"`
		Torrent         models.TorrentSettings      `json:"torrent"`
		Anilist         models.AnilistSettings      `json:"anilist"`
		Manga           models.MangaSettings        `json:"manga"`
		Notifications   models.NotificationSettings `json:"notifications"`
		EnableTranscode bool                        `json:"enableTranscode"`
		// Admin AniList token for authentication
		AdminAnilistToken string `json:"adminAnilistToken,omitempty"`
	}
	var b body

	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Security check: Only allow getting started during first-time setup
	setupCompleted, _ := h.App.Database.GetGlobalSettingsSetupStatus()
	if setupCompleted {
		return h.RespondWithError(c, errors.New("setup has already been completed"))
	}

	// Set up AniList whitelist and admin user during first-time setup
	// Note: If no admin token is provided, preserve any existing whitelist instead of overwriting it.
	var anilistWhitelist models.StringSlice
	var adminUsername string
	var dbUser *models.User
	if b.AdminAnilistToken != "" {
		// Validate the AniList token and get user info
		authenticatedClient := anilist.NewAnilistClient(b.AdminAnilistToken)
		getViewer, err := authenticatedClient.GetViewer(context.Background())
		if err != nil {
			return h.RespondWithError(c, fmt.Errorf("invalid AniList token: %w", err))
		}

		// If admin token is not provided, try to preserve an existing whitelist
		if b.AdminAnilistToken == "" {
			if existing, err := h.App.Database.GetGlobalSettings(); err == nil && existing != nil {
				if len(existing.AnilistWhitelist) > 0 {
					anilistWhitelist = existing.AnilistWhitelist
				}
			}
		}

		adminUsername = getViewer.Viewer.Name
		anilistWhitelist = models.StringSlice{adminUsername}
		h.App.Logger.Info().Str("adminUsername", adminUsername).Msg("Set admin AniList username in whitelist during setup")

		// Find or create admin user account and authenticate them
		existingUser, getErr := h.App.Database.GetUserByUsername(adminUsername)
		if getErr == nil && existingUser != nil {
			dbUser = existingUser
		} else {
			dbUser = &models.User{
				Username:    adminUsername,
				DisplayName: getViewer.Viewer.Name,
				Role:        "admin",
				IsActive:    true,
			}
			dbUser, err = h.App.Database.CreateUser(dbUser)
			if err != nil {
				h.App.Logger.Error().Err(err).Msg("Failed to create admin user")
			}
		}

		if dbUser != nil && err == nil {
			// Create AniList account entry
			viewerBytes, _ := json.Marshal(getViewer.Viewer)
			_, err = h.App.Database.UpsertAccount(&models.Account{
				UserID:   dbUser.ID,
				Username: getViewer.Viewer.Name,
				Token:    b.AdminAnilistToken,
				Viewer:   viewerBytes,
			})
			if err != nil {
				h.App.Logger.Error().Err(err).Msg("Failed to create admin AniList account")
			}

			// Create user session and set cookie (persists until manually revoked)
			session, err := h.App.Database.CreateUserSession(dbUser.ID, 0)
			if err != nil {
				h.App.Logger.Error().Err(err).Msg("Failed to create admin user session")
			} else {
				// Set session cookie
				cookie := &http.Cookie{
					Name:     "seanime-session",
					Value:    session.Token,
					Path:     "/",
					HttpOnly: true,
					Secure:   false,
					SameSite: http.SameSiteLaxMode,
				}
				if !session.ExpiresAt.IsZero() {
					cookie.Expires = session.ExpiresAt
					if remaining := time.Until(session.ExpiresAt); remaining > 0 {
						cookie.MaxAge = int(remaining.Seconds())
					}
				}
				c.SetCookie(cookie)
				h.App.Logger.Info().Str("username", adminUsername).Msg("Admin user logged in during setup")
			}
		}
	}

	// Check settings
	if b.Library.LibraryPaths == nil {
		b.Library.LibraryPaths = []string{}
	}
	b.Library.LibraryPath = filepath.ToSlash(b.Library.LibraryPath)

	// Create global settings (server-wide)
	globalSettings, err := h.App.Database.UpsertGlobalSettings(&models.GlobalSettings{
		BaseModel: models.BaseModel{
			ID:        1,
			UpdatedAt: time.Now(),
		},
		SetupCompleted:   true,
		AnilistWhitelist: anilistWhitelist,
		Library:          &b.Library,
		Torrent:          &b.Torrent,
		AutoDownloader: &models.AutoDownloaderSettings{
			Provider:              b.Library.TorrentProvider,
			Interval:              20,
			Enabled:               false,
			DownloadAutomatically: true,
			EnableEnhancedQueries: true,
		},
	})

	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Create user settings for the admin
	if dbUser != nil {
		_, err = h.App.Database.UpsertSettings(&models.Settings{
			BaseModel: models.BaseModel{
				UpdatedAt: time.Now(),
			},
			UserID:              dbUser.ID,
			AutoPlayNextEpisode: false,
			AutoUpdateProgress:  true,
			Anilist:             &b.Anilist,
			Manga:               &b.Manga,
			Notifications:       &b.Notifications,
		})

		if err != nil {
			return h.RespondWithError(c, err)
		}
	}

	// Auto-bootstrap existing library discovery for first admin
	if dbUser != nil && b.AdminAnilistToken != "" {
		go func() {
			defer util.HandlePanicThen(func() {})
			h.App.Logger.Info().Str("username", adminUsername).Msg("Starting auto-discovery of existing library files")

			// 1. Fetch AniList collection to populate local database
			_, err := h.App.RefreshAnimeCollectionForUser(dbUser)
			if err != nil {
				h.App.Logger.Error().Err(err).Msg("Failed to fetch AniList collection during setup")
				return
			}

			h.App.Logger.Info().Msg("AniList collection fetched, triggering initial library scan")

			// 2. Trigger system scan to discover existing files
			if h.App.SystemScanService != nil {
				h.App.SystemScanService.NotifyFileChange()
			}
		}()
	}

	settings := globalSettings

	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Enable transcoding by default during setup
	go func() {
		defer util.HandlePanicThen(func() {})
		globalSettings, err := h.App.Database.GetGlobalSettings()
		if err != nil {
			return
		}

		_, _ = h.App.Database.UpsertGlobalSettings(globalSettings)
	}()

	// Broadcast settings update to all users via SSE
	h.App.SSEManager.BroadcastEvent("settings", settings)

	status := h.NewStatus(c)

	// Refresh modules that depend on the settings
	h.App.InitOrRefreshModules()

	return h.RespondWithData(c, status)
}

// HandleSaveSettings
//
//	@summary updates the app settings.
//	@desc This will update the app settings.
//	@desc The client should re-fetch the server status after this.
//	@route /api/v1/settings [PATCH]
//	@returns handlers.Status
func (h *Handler) HandleSaveSettings(c echo.Context) error {

	type body struct {
		Library             models.LibrarySettings      `json:"library"`
		MediaPlayer         models.MediaPlayerSettings  `json:"mediaPlayer"`
		Torrent             models.TorrentSettings      `json:"torrent"`
		Anilist             models.AnilistSettings      `json:"anilist"`
		Discord             models.DiscordSettings      `json:"discord"`
		Manga               models.MangaSettings        `json:"manga"`
		Notifications       models.NotificationSettings `json:"notifications"`
		AutoUpdateProgress  bool                        `json:"autoUpdateProgress"`
		AutoPlayNextEpisode bool                        `json:"autoPlayNextEpisode"`
	}
	var b body

	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	if b.Library.LibraryPath != "" {
		b.Library.LibraryPath = filepath.ToSlash(filepath.Clean(b.Library.LibraryPath))
	}

	if b.Library.LibraryPaths == nil || b.Library.LibraryPath == "" {
		b.Library.LibraryPaths = []string{}
	}

	for i, path := range b.Library.LibraryPaths {
		b.Library.LibraryPaths[i] = filepath.ToSlash(filepath.Clean(path))
	}

	b.Library.LibraryPaths = lo.Filter(b.Library.LibraryPaths, func(s string, _ int) bool {
		if s == "" || util.IsSameDir(s, b.Library.LibraryPath) {
			return false
		}
		info, err := os.Stat(s)
		if err != nil {
			return false
		}
		return info.IsDir()
	})

	// Check that any library paths are not subdirectories of each other
	for i, path1 := range b.Library.LibraryPaths {
		if util.IsSubdirectory(b.Library.LibraryPath, path1) || util.IsSubdirectory(path1, b.Library.LibraryPath) {
			return h.RespondWithError(c, errors.New("library paths cannot be subdirectories of each other"))
		}
		for j, path2 := range b.Library.LibraryPaths {
			if i != j && util.IsSubdirectory(path1, path2) {
				return h.RespondWithError(c, errors.New("library paths cannot be subdirectories of each other"))
			}
		}
	}

	user := h.getCurrentUser(c)
	if user == nil {
		return c.JSON(401, map[string]string{"error": "Authentication required"})
	}

	// Check if user is admin to determine what settings they can modify
	isAdmin := user.IsAdmin()

	// Handle global settings (admin-only)
	if isAdmin {
		// Admin can modify global settings (library, torrent, auto-downloader)
		globalSettings, err := h.App.Database.GetGlobalSettings()
		if err != nil {
			return h.RespondWithError(c, err)
		}

		if globalSettings == nil {
			globalSettings = &models.GlobalSettings{
				BaseModel: models.BaseModel{ID: 1, UpdatedAt: time.Now()},
			}
		}

		// Update global settings with admin changes
		globalSettings.Library = &b.Library
		globalSettings.Torrent = &b.Torrent

		// Handle auto-downloader settings
		autoDownloaderSettings := models.AutoDownloaderSettings{}
		if globalSettings.AutoDownloader != nil {
			autoDownloaderSettings = *globalSettings.AutoDownloader
		}
		if b.Library.TorrentProvider == torrent.ProviderNone && autoDownloaderSettings.Enabled {
			h.App.Logger.Debug().Msg("app: Disabling auto-downloader because the torrent provider is set to none")
			autoDownloaderSettings.Enabled = false
		}
		globalSettings.AutoDownloader = &autoDownloaderSettings

		// Save global settings
		_, err = h.App.Database.UpsertGlobalSettings(globalSettings)
		if err != nil {
			return h.RespondWithError(c, err)
		}
	}

	// Handle user-specific settings (all users)
	prevSettings, err := h.App.Database.GetSettingsForUser(user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Create user settings with only personal preferences
	userSettings := &models.Settings{
		BaseModel: models.BaseModel{
			ID:        prevSettings.ID,
			UpdatedAt: time.Now(),
		},
		UserID:              user.ID,
		AutoUpdateProgress:  b.AutoUpdateProgress,
		AutoPlayNextEpisode: b.AutoPlayNextEpisode,
		MediaPlayer:         &b.MediaPlayer,
		Anilist:             &b.Anilist,
		Discord:             &b.Discord,
		Manga:               &b.Manga,
		Notifications:       &b.Notifications,
	}

	// Note: Library/Torrent/AutoDownloader settings are now global-only and not part of user settings

	err = h.App.Database.SaveSettingsForUser(user.ID, userSettings)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Broadcast user settings update to all users via SSE
	h.App.SSEManager.BroadcastEvent("settings", userSettings)

	status := h.NewStatus(c)

	// Refresh modules that depend on the settings
	h.App.InitOrRefreshModules()

	return h.RespondWithData(c, status)
}

// HandleTriggerLibraryScan
//
//	@summary triggers a manual library scan.
//	@desc This endpoint triggers a manual library scan to discover new files and update the library.
//	@desc It uses the same scan logic as the automatic file watcher but can be triggered on-demand.
//	@desc Admin-only operation as it affects system-wide file scanning for all users.
//	@route /api/v1/admin/system-scan/trigger [POST]
//	@returns bool
func (h *Handler) HandleTriggerLibraryScan(c echo.Context) error {

	// Get current user and verify admin privileges
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	// Verify admin privileges
	if !user.IsAdmin() {
		return h.RespondWithError(c, errors.New("admin privileges required"))
	}

	// Trigger the system scan service if available
	if h.App.SystemScanService != nil {
		h.App.SystemScanService.NotifyFileChange()
		h.App.Logger.Info().Str("username", user.Username).Msg("handlers: Manual system scan triggered by admin")
		return h.RespondWithData(c, true)
	}

	return h.RespondWithError(c, errors.New("system scan service not available"))
}

// HandleSaveAutoDownloaderSettings
//
//	@summary updates the auto-downloader settings.
//	@route /api/v1/settings/auto-downloader [PATCH]
//	@returns bool
func (h *Handler) HandleSaveAutoDownloaderSettings(c echo.Context) error {

	type body struct {
		Interval              int  `json:"interval"`
		Enabled               bool `json:"enabled"`
		DownloadAutomatically bool `json:"downloadAutomatically"`
		EnableEnhancedQueries bool `json:"enableEnhancedQueries"`
		EnableSeasonCheck     bool `json:"enableSeasonCheck"`
		UseDebrid             bool `json:"useDebrid"`
	}

	var b body

	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	currSettings, err := h.App.Database.GetGlobalSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Validation
	if b.Interval < 15 {
		return h.RespondWithError(c, errors.New("interval must be at least 15 minutes"))
	}

	autoDownloaderSettings := &models.AutoDownloaderSettings{
		Provider:              currSettings.Library.TorrentProvider,
		Interval:              b.Interval,
		Enabled:               b.Enabled,
		DownloadAutomatically: b.DownloadAutomatically,
		EnableEnhancedQueries: b.EnableEnhancedQueries,
		EnableSeasonCheck:     b.EnableSeasonCheck,
		UseDebrid:             b.UseDebrid,
	}

	currSettings.AutoDownloader = autoDownloaderSettings
	currSettings.BaseModel = models.BaseModel{
		ID:        1,
		UpdatedAt: time.Now(),
	}

	_, err = h.App.Database.UpsertGlobalSettings(currSettings)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Update Auto Downloader - This runs in a goroutine
	h.App.AutoDownloader.SetSettings(autoDownloaderSettings, currSettings.Library.TorrentProvider)

	return h.RespondWithData(c, true)
}
