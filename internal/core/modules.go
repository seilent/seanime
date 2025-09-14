package core

import (
	"runtime"
	"seanime/internal/continuity"
	"seanime/internal/database/db"
	"seanime/internal/database/db_bridge"
	"seanime/internal/directstream"
	discordrpc_presence "seanime/internal/discordrpc/presence"
	"seanime/internal/events"
	"seanime/internal/library/anime"
	"seanime/internal/library/autodownloader"
	"seanime/internal/library/autoscanner"
	"seanime/internal/library/fillermanager"
	"seanime/internal/library/playbackmanager"
	"seanime/internal/manga"
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
		FileCacher: a.FileCacher,
		Logger:     a.Logger,
		Database:   a.Database,
	})

	// +---------------------+
	// |   Playback Manager  |
	// +---------------------+

	// Playback Manager
	a.PlaybackManager = playbackmanager.New(&playbackmanager.NewPlaybackManagerOptions{
		Logger:            a.Logger,
		WSEventManager:    a.WSEventManager,
		Platform:          a.AnilistPlatform,
		MetadataProvider:  a.MetadataProvider,
		Database:          a.Database,
		DiscordPresence:   a.DiscordPresence,
		IsOffline:         a.IsOffline(),
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
		WSEventManager: a.WSEventManager,
		DownloadDir:    a.Config.Manga.DownloadDir,
		Repository:     a.MangaRepository,
		IsOffline:      a.IsOffline(),
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
		WSEventManager:    a.WSEventManager,
		ContinuityManager: a.ContinuityManager,
		MetadataProvider:  a.MetadataProvider,
		DiscordPresence:   a.DiscordPresence,
		Platform:          a.AnilistPlatform,
		RefreshAnimeCollectionFunc: nil, // Disabled in multiuser mode
		IsOffline:    a.IsOffline(),
		NativePlayer: a.NativePlayer,
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
		WSEventManager:          a.WSEventManager,
		MetadataProvider:        a.MetadataProvider,
		IsOffline:               a.IsOffline(),
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
		WSEventManager:   a.WSEventManager,
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

	// Create initial empty local files collection if none exists
	if _, _, err := db_bridge.GetLocalFiles(database); err != nil {
		_, err := db_bridge.InsertLocalFiles(database, make([]*anime.LocalFile, 0))
		if err != nil {
			logger.Fatal().Err(err).Msgf("app: Failed to initialize local files in the database")
		}
	}

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

	// Get settings from database
	settings, err := a.Database.GetSettings()
	if err != nil || settings == nil {
		a.Logger.Warn().Msg("app: Did not initialize modules, no settings found")
		return
	}

	a.Settings = settings // Store settings instance in app
	if settings.Library != nil {
		a.LibraryDir = settings.GetLibrary().LibraryPath
	}

	// +---------------------+
	// |   Module settings   |
	// +---------------------+
	// Refresh settings of modules that were initialized in initModulesOnce

	notifier.GlobalNotifier.SetSettings(a.Config.Data.AppDataDir, a.Settings.GetNotifications(), a.Logger)

	// Refresh updater settings
	if settings.Library != nil {
		plugin.GlobalAppContext.SetModulesPartial(plugin.AppContextModules{
			AnimeLibraryPaths: a.Database.AllLibraryPathsFromSettings(settings),
		})

		if a.Updater != nil {
			a.Updater.SetEnabled(!settings.Library.DisableUpdateCheck)
		}

		// Refresh auto scanner settings
		if a.AutoScanner != nil {
			a.AutoScanner.SetSettings(*settings.Library)
		}

		// Torrent Repository
		a.TorrentRepository.SetSettings(&torrent.RepositorySettings{
			DefaultAnimeProvider: settings.Library.TorrentProvider,
		})
	}

	if settings.MediaPlayer != nil {
		a.MediaPlayer.VLC = &vlc.VLC{
			Host:     settings.MediaPlayer.Host,
			Port:     settings.MediaPlayer.VlcPort,
			Password: settings.MediaPlayer.VlcPassword,
			Path:     settings.MediaPlayer.VlcPath,
			Logger:   a.Logger,
		}
		a.MediaPlayer.MpcHc = &mpchc.MpcHc{
			Host:   settings.MediaPlayer.Host,
			Port:   settings.MediaPlayer.MpcPort,
			Path:   settings.MediaPlayer.MpcPath,
			Logger: a.Logger,
		}
		a.MediaPlayer.Mpv = mpv.New(a.Logger, settings.MediaPlayer.MpvSocket, settings.MediaPlayer.MpvPath, settings.MediaPlayer.MpvArgs)
		a.MediaPlayer.Iina = iina.New(a.Logger, settings.MediaPlayer.IinaSocket, settings.MediaPlayer.IinaPath, settings.MediaPlayer.IinaArgs)

		// Set media player repository
		a.MediaPlayerRepository = mediaplayer.NewRepository(&mediaplayer.NewRepositoryOptions{
			Logger:            a.Logger,
			Default:           settings.MediaPlayer.Default,
			VLC:               a.MediaPlayer.VLC,
			MpcHc:             a.MediaPlayer.MpcHc,
			Mpv:               a.MediaPlayer.Mpv, // Socket
			Iina:              a.MediaPlayer.Iina,
			WSEventManager:    a.WSEventManager,
			ContinuityManager: a.ContinuityManager,
		})

		a.PlaybackManager.SetMediaPlayerRepository(a.MediaPlayerRepository)
		a.PlaybackManager.SetSettings(&playbackmanager.Settings{
			AutoPlayNextEpisode: a.Settings.GetLibrary().AutoPlayNextEpisode,
		})

		a.DirectStreamManager.SetSettings(&directstream.Settings{
			AutoPlayNextEpisode: a.Settings.GetLibrary().AutoPlayNextEpisode,
			AutoUpdateProgress:  a.Settings.GetLibrary().AutoUpdateProgress,
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

	if settings.Torrent != nil {
		// Init qBittorrent
		qbit := qbittorrent.NewClient(&qbittorrent.NewClientOptions{
			Logger:   a.Logger,
			Username: settings.Torrent.QBittorrentUsername,
			Password: settings.Torrent.QBittorrentPassword,
			Port:     settings.Torrent.QBittorrentPort,
			Host:     settings.Torrent.QBittorrentHost,
			Path:     settings.Torrent.QBittorrentPath,
			Tags:     settings.Torrent.QBittorrentTags,
		})
		// Login to qBittorrent
		go func() {
			if settings.Torrent.Default == "qbittorrent" {
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
			Username: settings.Torrent.TransmissionUsername,
			Password: settings.Torrent.TransmissionPassword,
			Port:     settings.Torrent.TransmissionPort,
			Host:     settings.Torrent.TransmissionHost,
			Path:     settings.Torrent.TransmissionPath,
		})
		if err != nil && settings.Torrent.TransmissionUsername != "" && settings.Torrent.TransmissionPassword != "" { // Only log error if username and password are set
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
			Provider:          settings.Torrent.Default,
			MetadataProvider:  a.MetadataProvider,
		})

		a.TorrentClientRepository.InitActiveTorrentCount(settings.Torrent.ShowActiveTorrentCount, a.WSEventManager)

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
	if settings.AutoDownloader != nil {
		a.AutoDownloader.SetSettings(settings.AutoDownloader, settings.Library.TorrentProvider)
	}

	// +---------------------+
	// |   Library Watcher   |
	// +---------------------+

	// Initialize library watcher
	if settings.Library != nil && len(settings.Library.LibraryPath) > 0 {
		go func() {
			a.initLibraryWatcher(settings.Library.GetLibraryPaths())
		}()
	}

	// +---------------------+
	// |       Discord       |
	// +---------------------+

	if settings.Discord != nil && a.DiscordPresence != nil {
		a.DiscordPresence.SetSettings(settings.Discord)
	}

	// +---------------------+
	// |     Continuity      |
	// +---------------------+

	if settings.Library != nil {
		a.ContinuityManager.SetSettings(&continuity.Settings{
			WatchContinuityEnabled: settings.Library.EnableWatchContinuity,
		})
	}

	if settings.Manga != nil {
		a.MangaRepository.SetSettings(settings)
	}


	runtime.GC()

	a.Logger.Info().Msg("app: Refreshed modules")

}




// InitOrRefreshAnilistData is now simplified for pure multiuser mode.
// No global user or collections - everything is per-user via handlers.
func (a *App) InitOrRefreshAnilistData() {
	a.Logger.Debug().Msg("app: Multiuser AniList initialization - no global user/collections")

	// No global user setup - everything is handled per-user in handlers
	a.ServerReady = true
	a.WSEventManager.SendEvent(events.ServerReady, nil)

	a.Logger.Info().Msg("app: Multiuser AniList initialization complete")
}

func (a *App) performActionsOnce() {

	go func() {
		if a.Settings == nil || a.Settings.Library == nil {
			return
		}

		if a.Settings.GetLibrary().OpenWebURLOnStart {
			// Open the web URL
			err := browser.OpenURL(a.Config.GetServerURI("127.0.0.1"))
			if err != nil {
				a.Logger.Warn().Err(err).Msg("app: Failed to open web URL, please open it manually in your browser")
			} else {
				a.Logger.Info().Msg("app: Opened web URL")
			}
		}

		if a.Settings.GetLibrary().RefreshLibraryOnStart {
			go func() {
				a.Logger.Debug().Msg("app: Refreshing library")
				a.AutoScanner.RunNow()
				a.Logger.Info().Msg("app: Refreshed library")
			}()
		}

		if a.Settings.GetLibrary().OpenTorrentClientOnStart && a.TorrentClientRepository != nil {
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
