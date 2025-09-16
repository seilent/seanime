package core

import (
	"runtime"
	"seanime/internal/continuity"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"seanime/internal/directstream"
	discordrpc_presence "seanime/internal/discordrpc/presence"
	"seanime/internal/events"
	"seanime/internal/library/autodownloader"
	"seanime/internal/library/autoscanner"
	"seanime/internal/library/fillermanager"
	"seanime/internal/library/playbackmanager"
	"seanime/internal/manga"
	"seanime/internal/mediastream"
	"seanime/internal/mediaplayers/iina"
	"seanime/internal/mediaplayers/mediaplayer"
	"seanime/internal/mediaplayers/mpchc"
	"seanime/internal/mediaplayers/mpv"
	"seanime/internal/mediaplayers/vlc"
	"seanime/internal/nativeplayer"
	"seanime/internal/notifier"
	"seanime/internal/plugin"
	"seanime/internal/torrent_clients/qbittorrent"
	"seanime/internal/torrent_clients/torrent_client"
	"seanime/internal/torrent_clients/transmission"
	"seanime/internal/torrents/torrent"
	"seanime/internal/util"

	"github.com/cli/browser"
	"github.com/rs/zerolog"
)

// initModulesOnce will initialize modules that need to persist.
// This function is called once after the App instance is created.
// The settings of these modules will be set/refreshed in InitOrRefreshModules.
func (a *App) initModulesOnce() {

	// Skip setting global collection refresh callbacks in multiuser mode
	// Collections are now fetched per-user via handlers
	a.Logger.Debug().Msg("app: Skipping global LocalManager collection refresh callbacks in multiuser mode")

	// Skip setting global plugin refresh callbacks in multiuser mode
	// Collections are now fetched per-user via handlers
	plugin.GlobalAppContext.SetModulesPartial(plugin.AppContextModules{
		// OnRefreshAnilistAnimeCollection and OnRefreshAnilistMangaCollection
		// are intentionally omitted in multiuser mode
	})

	// +---------------------+
	// |     Discord RPC     |
	// +---------------------+

	a.DiscordPresence = discordrpc_presence.New(nil, a.Logger)
	a.AddCleanupFunction(func() {
		a.DiscordPresence.Close()
	})

	plugin.GlobalAppContext.SetModulesPartial(plugin.AppContextModules{
		DiscordPresence: a.DiscordPresence,
	})

	// +---------------------+
	// |       Filler        |
	// +---------------------+

	a.FillerManager = fillermanager.New(&fillermanager.NewFillerManagerOptions{
		DB:     a.Database,
		Logger: a.Logger,
	})

	plugin.GlobalAppContext.SetModulesPartial(plugin.AppContextModules{
		FillerManager: a.FillerManager,
	})

	// +---------------------+
	// |     Continuity      |
	// +---------------------+

	a.ContinuityManager = continuity.NewManager(&continuity.NewManagerOptions{
		FileCacher:  a.FileCacher,
		Logger:      a.Logger,
		Database:    a.Database,
		SyncManager: a.SyncManager,
	})

	// +---------------------+
	// |   Playback Manager  |
	// +---------------------+

	// Playback Manager with SSE adapter
	sseAdapter := events.NewSSEEventManagerAdapter(a.SSEManager)
	a.PlaybackManager = playbackmanager.New(&playbackmanager.NewPlaybackManagerOptions{
		Logger:            a.Logger,
		WSEventManager:    a.WSEventManager,    // Keep for backwards compatibility
		SSEEventManager:   sseAdapter,          // Preferred SSE implementation
		Platform:          a.AnilistPlatform,
		MetadataProvider:  a.MetadataProvider,
		Database:          a.Database,
		DiscordPresence:   a.DiscordPresence,
		IsOffline:         util.NewBool(false),
		ContinuityManager: a.ContinuityManager,
		RefreshAnimeCollectionFunc: nil, // Disabled in multiuser mode
	})

	// +---------------------+
	// |  Torrent Repository |
	// +---------------------+

	a.TorrentRepository = torrent.NewRepository(&torrent.NewRepositoryOptions{
		Logger:           a.Logger,
		MetadataProvider: a.MetadataProvider,
	})

	// +---------------------+
	// |  Manga Downloader   |
	// +---------------------+

	a.MangaDownloader = manga.NewDownloader(&manga.NewDownloaderOptions{
		Database:       a.Database,
		Logger:         a.Logger,
		WSEventManager: sseAdapter,  // Use SSE adapter
		DownloadDir:    a.Config.Manga.DownloadDir,
		Repository:     a.MangaRepository,
		IsOffline:      util.NewBool(false),
	})

	a.MangaDownloader.Start()


	// +---------------------+
	// |    Native Player    |
	// +---------------------+

	a.NativePlayer = nativeplayer.New(nativeplayer.NewNativePlayerOptions{
		WsEventManager: a.WSEventManager,
		Logger:         a.Logger,
	})

	// +---------------------+
	// |   Direct Stream     |
	// +---------------------+

	a.DirectStreamManager = directstream.NewManager(directstream.NewManagerOptions{
		Logger:            a.Logger,
		WSEventManager:    sseAdapter,  // Use SSE adapter
		ContinuityManager: a.ContinuityManager,
		MetadataProvider:  a.MetadataProvider,
		DiscordPresence:   a.DiscordPresence,
		Platform:          a.AnilistPlatform,
		RefreshAnimeCollectionFunc: nil, // Disabled in multiuser mode
		IsOffline:    util.NewBool(false),
		NativePlayer: a.NativePlayer,
	})

	// +---------------------+
	// |    Media Stream     |
	// +---------------------+

	a.MediastreamRepository = mediastream.NewRepository(&mediastream.NewRepositoryOptions{
		Logger:         a.Logger,
		WSEventManager: sseAdapter,  // Use SSE adapter
		FileCacher:     a.FileCacher,
	})

	a.AddCleanupFunction(func() {
		a.MediastreamRepository.OnCleanup()
	})

	plugin.GlobalAppContext.SetModulesPartial(plugin.AppContextModules{
		PlaybackManager: a.PlaybackManager,
		MangaRepository: a.MangaRepository,
	})

	// +---------------------+
	// |   Auto Downloader   |
	// +---------------------+

	a.AutoDownloader = autodownloader.New(&autodownloader.NewAutoDownloaderOptions{
		Logger:                  a.Logger,
		TorrentClientRepository: a.TorrentClientRepository,
		TorrentRepository:       a.TorrentRepository,
		Database:                a.Database,
		WSEventManager:          sseAdapter,  // Use SSE adapter
		MetadataProvider:        a.MetadataProvider,
		IsOffline:               util.NewBool(false),
	})

	// This is run in a goroutine
	a.AutoDownloader.Start()

	// +---------------------+
	// |   Auto Scanner      |
	// +---------------------+

	a.AutoScanner = autoscanner.New(&autoscanner.NewAutoScannerOptions{
		Database:         a.Database,
		Platform:         a.AnilistPlatform,
		Logger:           a.Logger,
		WSEventManager:   sseAdapter,  // Use SSE adapter
		Enabled:          false, // Will be set in InitOrRefreshModules
		AutoDownloader:   a.AutoDownloader,
		MetadataProvider: a.MetadataProvider,
		LogsDir:          a.Config.Logs.Dir,
	})

	// This is run in a goroutine
	a.AutoScanner.Start()


}

// HandleNewDatabaseEntries initializes essential database collections.
// It creates an empty local files collection if one does not already exist.
func HandleNewDatabaseEntries(database *db.Database, logger *zerolog.Logger) {

	// NOTE: Local files are now managed via global_anime_file_mappings table
	// No initialization needed for the new system

}

// InitOrRefreshModules will initialize or refresh modules that depend on settings.
// This function is called:
//   - After the App instance is created
//   - After settings are updated.
func (a *App) InitOrRefreshModules() {
	a.moduleMu.Lock()
	defer a.moduleMu.Unlock()

	a.Logger.Debug().Msgf("app: Refreshing modules")

	// Stop watching if already watching
	if a.Watcher != nil {
		a.Watcher.StopWatching()
	}

	// If Discord presence is already initialized, close it
	if a.DiscordPresence != nil {
		a.DiscordPresence.Close()
	}

	// Get global settings from database
	globalSettings, err := a.Database.GetGlobalSettings()
	if err != nil || globalSettings == nil {
		a.Logger.Warn().Msg("app: Did not initialize modules, no global settings found")
		return
	}

	// Get user settings (for first user or admin)
	userSettings, err := a.Database.GetSettings()
	if err != nil {
		userSettings = nil // User settings are optional
	}

	a.Settings = userSettings // Store user settings instance in app
	a.GlobalSettings = globalSettings // Store global settings instance in app
	if globalSettings.Library != nil {
		a.LibraryDir = globalSettings.GetLibrary().LibraryPath
	}

	// +---------------------+
	// |   Module settings   |
	// +---------------------+
	// Refresh settings of modules that were initialized in initModulesOnce

	notifier.GlobalNotifier.SetSettings(a.Config.Data.AppDataDir, a.Settings.GetNotifications(), a.Logger)

	// Refresh updater settings
	if globalSettings.Library != nil {
		plugin.GlobalAppContext.SetModulesPartial(plugin.AppContextModules{
			AnimeLibraryPaths: a.Database.AllLibraryPathsFromSettings(globalSettings),
		})

		if a.Updater != nil {
			a.Updater.SetEnabled(!globalSettings.Library.DisableUpdateCheck)
		}

		// Refresh auto scanner settings
		if a.AutoScanner != nil {
			a.AutoScanner.SetSettings(*globalSettings.Library)
		}

		// Torrent Repository
		a.TorrentRepository.SetSettings(&torrent.RepositorySettings{
			DefaultAnimeProvider: globalSettings.Library.TorrentProvider,
		})
	}

	if userSettings != nil && userSettings.MediaPlayer != nil {
		a.MediaPlayer.VLC = &vlc.VLC{
			Host:     userSettings.MediaPlayer.Host,
			Port:     userSettings.MediaPlayer.VlcPort,
			Password: userSettings.MediaPlayer.VlcPassword,
			Path:     userSettings.MediaPlayer.VlcPath,
			Logger:   a.Logger,
		}
		a.MediaPlayer.MpcHc = &mpchc.MpcHc{
			Host:   userSettings.MediaPlayer.Host,
			Port:   userSettings.MediaPlayer.MpcPort,
			Path:   userSettings.MediaPlayer.MpcPath,
			Logger: a.Logger,
		}
		a.MediaPlayer.Mpv = mpv.New(a.Logger, userSettings.MediaPlayer.MpvSocket, userSettings.MediaPlayer.MpvPath, userSettings.MediaPlayer.MpvArgs)
		a.MediaPlayer.Iina = iina.New(a.Logger, userSettings.MediaPlayer.IinaSocket, userSettings.MediaPlayer.IinaPath, userSettings.MediaPlayer.IinaArgs)

		// Set media player repository
		a.MediaPlayerRepository = mediaplayer.NewRepository(&mediaplayer.NewRepositoryOptions{
			Logger:            a.Logger,
			Default:           userSettings.MediaPlayer.Default,
			VLC:               a.MediaPlayer.VLC,
			MpcHc:             a.MediaPlayer.MpcHc,
			Mpv:               a.MediaPlayer.Mpv, // Socket
			Iina:              a.MediaPlayer.Iina,
			WSEventManager:    events.NewSSEEventManagerAdapter(a.SSEManager),  // Use SSE adapter
			ContinuityManager: a.ContinuityManager,
		})

		a.PlaybackManager.SetMediaPlayerRepository(a.MediaPlayerRepository)
		// Use user-specific playback preferences or defaults
		autoPlayNext := false
		autoUpdateProgress := true
		if userSettings != nil {
			autoPlayNext = userSettings.AutoPlayNextEpisode
			autoUpdateProgress = userSettings.AutoUpdateProgress
		}

		a.PlaybackManager.SetSettings(&playbackmanager.Settings{
			AutoPlayNextEpisode: autoPlayNext,
		})

		a.DirectStreamManager.SetSettings(&directstream.Settings{
			AutoPlayNextEpisode: autoPlayNext,
			AutoUpdateProgress:  autoUpdateProgress,
		})


		plugin.GlobalAppContext.SetModulesPartial(plugin.AppContextModules{
			MediaPlayerRepository: a.MediaPlayerRepository,
		})
	} else {
		a.Logger.Warn().Msg("app: Did not initialize media player module, no settings found")
	}

	// +---------------------+
	// |       Torrents      |
	// +---------------------+

	if globalSettings.Torrent != nil {
		// Init qBittorrent
		qbit := qbittorrent.NewClient(&qbittorrent.NewClientOptions{
			Logger:   a.Logger,
			Username: globalSettings.Torrent.QBittorrentUsername,
			Password: globalSettings.Torrent.QBittorrentPassword,
			Port:     globalSettings.Torrent.QBittorrentPort,
			Host:     globalSettings.Torrent.QBittorrentHost,
			Path:     globalSettings.Torrent.QBittorrentPath,
			Tags:     globalSettings.Torrent.QBittorrentTags,
		})
		// Login to qBittorrent
		go func() {
			if globalSettings.Torrent.Default == "qbittorrent" {
				err = qbit.Login()
				if err != nil {
					a.Logger.Error().Err(err).Msg("app: Failed to login to qBittorrent")
				} else {
					a.Logger.Info().Msg("app: Logged in to qBittorrent")
				}
			}
		}()
		// Init Transmission
		trans, err := transmission.New(&transmission.NewTransmissionOptions{
			Logger:   a.Logger,
			Username: globalSettings.Torrent.TransmissionUsername,
			Password: globalSettings.Torrent.TransmissionPassword,
			Port:     globalSettings.Torrent.TransmissionPort,
			Host:     globalSettings.Torrent.TransmissionHost,
			Path:     globalSettings.Torrent.TransmissionPath,
		})
		if err != nil && globalSettings.Torrent.TransmissionUsername != "" && globalSettings.Torrent.TransmissionPassword != "" { // Only log error if username and password are set
			a.Logger.Error().Err(err).Msg("app: Failed to initialize transmission client")
		}

		// Shutdown torrent client first
		if a.TorrentClientRepository != nil {
			a.TorrentClientRepository.Shutdown()
		}

		// Torrent Client Repository
		a.TorrentClientRepository = torrent_client.NewRepository(&torrent_client.NewRepositoryOptions{
			Logger:            a.Logger,
			QbittorrentClient: qbit,
			Transmission:      trans,
			TorrentRepository: a.TorrentRepository,
			Provider:          globalSettings.Torrent.Default,
			MetadataProvider:  a.MetadataProvider,
		})

		a.TorrentClientRepository.InitActiveTorrentCount(globalSettings.Torrent.ShowActiveTorrentCount, events.NewSSEEventManagerAdapter(a.SSEManager))

		// Set AutoDownloader qBittorrent client
		a.AutoDownloader.SetTorrentClientRepository(a.TorrentClientRepository)

		plugin.GlobalAppContext.SetModulesPartial(plugin.AppContextModules{
			TorrentClientRepository: a.TorrentClientRepository,
			AutoDownloader:          a.AutoDownloader,
		})
	} else {
		a.Logger.Warn().Msg("app: Did not initialize torrent client module, no settings found")
	}

	// +---------------------+
	// |   AutoDownloader    |
	// +---------------------+

	// Update Auto Downloader - This runs in a goroutine
	if globalSettings.AutoDownloader != nil {
		a.AutoDownloader.SetSettings(globalSettings.AutoDownloader, globalSettings.Library.TorrentProvider)
	}

	// +---------------------+
	// |   Library Watcher   |
	// +---------------------+

	// Initialize library watcher
	if globalSettings.Library != nil && len(globalSettings.Library.LibraryPath) > 0 {
		go func() {
			a.initLibraryWatcher(globalSettings.Library.GetLibraryPaths())
		}()
	}

	// +---------------------+
	// |       Discord       |
	// +---------------------+

	if userSettings != nil && userSettings.Discord != nil && a.DiscordPresence != nil {
		a.DiscordPresence.SetSettings(userSettings.Discord)
	}

	// +---------------------+
	// |     Continuity      |
	// +---------------------+

	if globalSettings.Library != nil {
		a.ContinuityManager.SetSettings(&continuity.Settings{
			WatchContinuityEnabled: globalSettings.Library.EnableWatchContinuity,
		})
	}

	if userSettings != nil && userSettings.Manga != nil {
		a.MangaRepository.SetSettings(userSettings)
	}

	// +---------------------+
	// | Secondary Settings  |
	// +---------------------+
	// Load settings that are sent to the client via status endpoint

	// Convert transcoding settings from GlobalSettings for backwards compatibility
	var mediastreamSettings *models.MediastreamSettings
	if globalSettings != nil && globalSettings.Transcoding != nil {
		mediastreamSettings = &models.MediastreamSettings{
			BaseModel: models.BaseModel{ID: 1},
			TranscodeEnabled:              globalSettings.Transcoding.TranscodeEnabled,
			TranscodeHwAccel:              globalSettings.Transcoding.TranscodeHwAccel,
			TranscodeThreads:              globalSettings.Transcoding.TranscodeThreads,
			TranscodePreset:               globalSettings.Transcoding.TranscodePreset,
			PreTranscodeEnabled:           globalSettings.Transcoding.PreTranscodeEnabled,
			PreTranscodeLibraryDir:        globalSettings.Transcoding.PreTranscodeLibraryDir,
			FfmpegPath:                    globalSettings.Transcoding.FfmpegPath,
			FfprobePath:                   globalSettings.Transcoding.FfprobePath,
			TranscodeHwAccelCustomSettings: globalSettings.Transcoding.TranscodeHwAccelCustomSettings,
			// Client settings are no longer used in server-side operations
			DisableAutoSwitchToDirectPlay: false,
			DirectPlayOnly:                false,
		}
	}
	a.SecondarySettings.Mediastream = mediastreamSettings


	runtime.GC()

	a.Logger.Info().Msg("app: Refreshed modules")

}




// InitOrRefreshAnilistData is now simplified for pure multiuser mode.
// No global user or collections - everything is per-user via handlers.
func (a *App) InitOrRefreshAnilistData() {
	a.Logger.Debug().Msg("app: Multiuser AniList initialization - no global user/collections")

	// No global user setup - everything is handled per-user in handlers
	a.ServerReady = true
	// Send server ready via SSE
	a.SSEManager.BroadcastEvent(events.ServerReady, nil)

	a.Logger.Info().Msg("app: Multiuser AniList initialization complete")
}

func (a *App) performActionsOnce() {

	go func() {
		if a.GlobalSettings == nil || a.GlobalSettings.Library == nil {
			return
		}

		if a.GlobalSettings.GetLibrary().OpenWebURLOnStart {
			// Open the web URL
			err := browser.OpenURL(a.Config.GetServerURI("127.0.0.1"))
			if err != nil {
				a.Logger.Warn().Err(err).Msg("app: Failed to open web URL, please open it manually in your browser")
			} else {
				a.Logger.Info().Msg("app: Opened web URL")
			}
		}

		if a.GlobalSettings.GetLibrary().RefreshLibraryOnStart {
			go func() {
				a.Logger.Debug().Msg("app: Refreshing library")
				a.AutoScanner.RunNow()
				a.Logger.Info().Msg("app: Refreshed library")
			}()
		}

		if a.GlobalSettings.GetLibrary().OpenTorrentClientOnStart && a.TorrentClientRepository != nil {
			// Start the torrent client
			ok := a.TorrentClientRepository.Start()
			if !ok {
				a.Logger.Warn().Msg("app: Failed to open torrent client")
			} else {
				a.Logger.Info().Msg("app: Started torrent client")
			}

		}
	}()

}

// InitOrRefreshMediastreamSettings will initialize or refresh the mediastream settings.
// It is called after the App instance is created and after settings are updated.
func (a *App) InitOrRefreshMediastreamSettings() {

	// Get transcoding settings from GlobalSettings
	globalSettings, err := a.Database.GetGlobalSettings()
	if err != nil {
		a.Logger.Error().Err(err).Msg("app: Failed to get global settings")
		return
	}

	// Initialize default transcoding settings if not present
	if globalSettings.Transcoding == nil {
		globalSettings.Transcoding = &models.ServerTranscodingSettings{
			TranscodeEnabled:    false,
			TranscodeHwAccel:    "cpu",
			TranscodePreset:     "fast",
			TranscodeThreads:    4,
			PreTranscodeEnabled: false,
		}
		// Save the updated global settings
		_, err = a.Database.UpsertGlobalSettings(globalSettings)
		if err != nil {
			a.Logger.Error().Err(err).Msg("app: Failed to initialize transcoding settings")
			return
		}
	}

	// Convert ServerTranscodingSettings to MediastreamSettings for backwards compatibility
	// TODO: Update mediastream repository to use ServerTranscodingSettings directly
	settings := &models.MediastreamSettings{
		BaseModel: models.BaseModel{ID: 1},
		TranscodeEnabled:              globalSettings.Transcoding.TranscodeEnabled,
		TranscodeHwAccel:              globalSettings.Transcoding.TranscodeHwAccel,
		TranscodeThreads:              globalSettings.Transcoding.TranscodeThreads,
		TranscodePreset:               globalSettings.Transcoding.TranscodePreset,
		PreTranscodeEnabled:           globalSettings.Transcoding.PreTranscodeEnabled,
		PreTranscodeLibraryDir:        globalSettings.Transcoding.PreTranscodeLibraryDir,
		FfmpegPath:                    globalSettings.Transcoding.FfmpegPath,
		FfprobePath:                   globalSettings.Transcoding.FfprobePath,
		TranscodeHwAccelCustomSettings: globalSettings.Transcoding.TranscodeHwAccelCustomSettings,
		// Client settings are no longer used in server-side operations
		DisableAutoSwitchToDirectPlay: false,
		DirectPlayOnly:                false,
	}

	a.MediastreamRepository.InitializeModules(settings, a.Config.Cache.Dir, a.Config.Cache.TranscodeDir)

	// Cleanup cache
	go func() {
		if settings.TranscodeEnabled {
			// If transcoding is enabled, trim files
			_ = a.FileCacher.TrimMediastreamVideoFiles()
		} else {
			// If transcoding is disabled, clear all files
			_ = a.FileCacher.ClearMediastreamVideoFiles()
		}
	}()

	// Store the converted settings for SecondarySettings (backwards compatibility)
	a.SecondarySettings.Mediastream = settings
}
