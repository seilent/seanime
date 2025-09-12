package handlers

import (
	"errors"
	"os"
	"path/filepath"
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

	// Get global/system settings for shared resources like library path
	globalSettings, err := h.App.Database.GetSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Merge settings: use global library settings, user settings for personal preferences
	if globalSettings != nil && globalSettings.Library != nil {
		userSettings.Library = globalSettings.Library
	}

	return h.RespondWithData(c, userSettings)
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
		Library                models.LibrarySettings      `json:"library"`
		MediaPlayer            models.MediaPlayerSettings  `json:"mediaPlayer"`
		Torrent                models.TorrentSettings      `json:"torrent"`
		Anilist                models.AnilistSettings      `json:"anilist"`
		Discord                models.DiscordSettings      `json:"discord"`
		Manga                  models.MangaSettings        `json:"manga"`
		Notifications          models.NotificationSettings `json:"notifications"`
		Nakama                 models.NakamaSettings       `json:"nakama"`
		EnableTranscode        bool                        `json:"enableTranscode"`
		EnableTorrentStreaming bool                        `json:"enableTorrentStreaming"`
		DebridProvider         string                      `json:"debridProvider"`
		DebridApiKey           string                      `json:"debridApiKey"`
		// Admin user creation fields
		AdminUsername    string `json:"adminUsername,omitempty"`
		AdminPassword    string `json:"adminPassword,omitempty"`
		AdminDisplayName string `json:"adminDisplayName,omitempty"`
	}
	var b body

	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Check if admin user creation is needed during first-time setup
	if b.AdminUsername != "" && b.AdminPassword != "" {
		// Create the first admin user
		displayName := b.AdminDisplayName
		if displayName == "" {
			displayName = b.AdminUsername
		}
		
		_, err := h.App.Database.CreateFirstTimeSetup(b.AdminUsername, b.AdminPassword, displayName)
		if err != nil {
			// Log the error but don't fail the setup if users already exist
			h.App.Logger.Warn().Err(err).Str("username", b.AdminUsername).Msg("Could not create admin user during setup")
		} else {
			h.App.Logger.Info().Str("username", b.AdminUsername).Msg("Created first-time admin user during setup")
		}
	}

	// Check settings
	if b.Library.LibraryPaths == nil {
		b.Library.LibraryPaths = []string{}
	}
	b.Library.LibraryPath = filepath.ToSlash(b.Library.LibraryPath)

	settings, err := h.App.Database.UpsertSettings(&models.Settings{
		BaseModel: models.BaseModel{
			ID:        1,
			UpdatedAt: time.Now(),
		},
		Library:       &b.Library,
		MediaPlayer:   &b.MediaPlayer,
		Torrent:       &b.Torrent,
		Anilist:       &b.Anilist,
		Discord:       &b.Discord,
		Manga:         &b.Manga,
		Notifications: &b.Notifications,
		Nakama:        &b.Nakama,
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

	if b.EnableTorrentStreaming {
		go func() {
			defer util.HandlePanicThen(func() {})
			prev, found := h.App.Database.GetTorrentstreamSettings()
			if found {
				prev.Enabled = true
				//prev.IncludeInLibrary = true
				_, _ = h.App.Database.UpsertTorrentstreamSettings(prev)
			}
		}()
	}

	if b.EnableTranscode {
		go func() {
			defer util.HandlePanicThen(func() {})
			prev, found := h.App.Database.GetMediastreamSettings()
			if found {
				prev.TranscodeEnabled = true
				_, _ = h.App.Database.UpsertMediastreamSettings(prev)
			}
		}()
	}

	if b.DebridProvider != "" && b.DebridProvider != "none" {
		go func() {
			defer util.HandlePanicThen(func() {})
			prev, found := h.App.Database.GetDebridSettings()
			if found {
				prev.Enabled = true
				prev.Provider = b.DebridProvider
				prev.ApiKey = b.DebridApiKey
				//prev.IncludeDebridStreamInLibrary = true
				_, _ = h.App.Database.UpsertDebridSettings(prev)
			}
		}()
	}

	h.App.WSEventManager.SendEvent("settings", settings)

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
		Library       models.LibrarySettings      `json:"library"`
		MediaPlayer   models.MediaPlayerSettings  `json:"mediaPlayer"`
		Torrent       models.TorrentSettings      `json:"torrent"`
		Anilist       models.AnilistSettings      `json:"anilist"`
		Discord       models.DiscordSettings      `json:"discord"`
		Manga         models.MangaSettings        `json:"manga"`
		Notifications models.NotificationSettings `json:"notifications"`
		Nakama        models.NakamaSettings       `json:"nakama"`
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
		globalSettings, err := h.App.Database.GetSettings()
		if err != nil {
			return h.RespondWithError(c, err)
		}
		
		if globalSettings == nil {
			globalSettings = &models.Settings{
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
		_, err = h.App.Database.UpsertSettings(globalSettings)
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
		UserID:        user.ID,
		MediaPlayer:   &b.MediaPlayer,
		Anilist:       &b.Anilist,
		Discord:       &b.Discord,
		Manga:         &b.Manga,
		Notifications: &b.Notifications,
		Nakama:        &b.Nakama,
	}

	// Non-admins cannot modify library/torrent settings, so keep existing values
	if !isAdmin && prevSettings != nil {
		userSettings.Library = prevSettings.Library
		userSettings.Torrent = prevSettings.Torrent  
		userSettings.AutoDownloader = prevSettings.AutoDownloader
	}

	err = h.App.Database.SaveSettingsForUser(user.ID, userSettings)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	h.App.WSEventManager.SendEvent("settings", userSettings)

	status := h.NewStatus(c)

	// Refresh modules that depend on the settings
	h.App.InitOrRefreshModules()

	return h.RespondWithData(c, status)
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

	currSettings, err := h.App.Database.GetSettings()
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

	_, err = h.App.Database.UpsertSettings(currSettings)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Update Auto Downloader - This runs in a goroutine
	h.App.AutoDownloader.SetSettings(autoDownloaderSettings, currSettings.Library.TorrentProvider)

	return h.RespondWithData(c, true)
}
