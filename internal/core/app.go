package core

import (
	"os"
	"runtime"
	"seanime/internal/api/anilist"
	"seanime/internal/api/metadata"
	"seanime/internal/constants"
	"seanime/internal/continuity"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"seanime/internal/directstream"
	discordrpc_presence "seanime/internal/discordrpc/presence"
	"seanime/internal/doh"
	"seanime/internal/events"
	"seanime/internal/extension_playground"
	"seanime/internal/extension_repo"
	"seanime/internal/global_mapping"
	"seanime/internal/hook"
	"seanime/internal/library/autodownloader"
	"seanime/internal/library/autoscanner"
	"seanime/internal/library/fillermanager"
	"seanime/internal/library/playbackmanager"
	"seanime/internal/library/scanner"
	libsync "seanime/internal/library/sync"
	"seanime/internal/util/limiter"
	"seanime/internal/local"
	"seanime/internal/manga"
	"seanime/internal/mediastream"
	"seanime/internal/mediaplayers/iina"
	"seanime/internal/mediaplayers/mediaplayer"
	"seanime/internal/mediaplayers/mpchc"
	"seanime/internal/mediaplayers/mpv"
	"seanime/internal/mediaplayers/vlc"
	"seanime/internal/nativeplayer"
	"seanime/internal/platforms/anilist_platform"
	"seanime/internal/platforms/platform"
	"seanime/internal/plugin"
	"seanime/internal/report"
	"seanime/internal/torrent_clients/torrent_client"
	"seanime/internal/torrents/torrent"
	"seanime/internal/updater"
	"seanime/internal/util"
	"seanime/internal/util/filecache"
	"seanime/internal/util/result"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type (
	App struct {
	Config                        *Config
	Database                      *db.Database
	Logger                        *zerolog.Logger
	TorrentClientRepository       *torrent_client.Repository
	TorrentRepository             *torrent.Repository
	Watcher                       *scanner.Watcher
	AnilistClient                 anilist.AnilistClient
	AnilistPlatform               platform.Platform
	LocalManager                  local.Manager
	FillerManager                 *fillermanager.FillerManager
	WSEventManager                *events.WSEventManager
	SSEManager                    *events.SSEManager
	AutoDownloader                *autodownloader.AutoDownloader
	ExtensionRepository           *extension_repo.Repository
	ExtensionPlaygroundRepository *extension_playground.PlaygroundRepository
	DirectStreamManager           *directstream.Manager
	NativePlayer                  *nativeplayer.NativePlayer
	MediaPlayer                   struct {
		VLC   *vlc.VLC
		MpcHc *mpchc.MpcHc
		Mpv   *mpv.Mpv
		Iina  *iina.Iina
	}
	MediaPlayerRepository           *mediaplayer.Repository
	Version                         string
	Updater                         *updater.Updater
	AutoScanner                     *autoscanner.AutoScanner
	PlaybackManager                 *playbackmanager.PlaybackManager
	FileCacher                      *filecache.Cacher
	MangaRepository                 *manga.Repository
	MediastreamRepository           *mediastream.Repository
	MetadataProvider                metadata.Provider
	DiscordPresence                 *discordrpc_presence.Presence
	MangaDownloader                 *manga.Downloader
	ContinuityManager               *continuity.Manager
	Cleanups                        []func()
	OnRefreshAnilistCollectionFuncs *result.Map[string, func()]
	OnFlushLogs                     func()
	FeatureFlags                    FeatureFlags
	Settings                        *models.Settings
	GlobalSettings                  *models.GlobalSettings
        SecondarySettings               struct {
            Torrentstream *models.TorrentstreamSettings
        } // Struct for other settings sent to client
	SelfUpdater        *updater.SelfUpdater
	ReportRepository   *report.Repository
	TotalLibrarySize   uint64 // Initialized in modules.go
	LibraryDir         string
	IsDesktopSidecar   bool
	// Global collections and user removed - now pure multiuser system
	previousVersion    string
	moduleMu           sync.Mutex
	HookManager        hook.Manager
	ServerReady        bool // Whether the Anilist data from the first request has been fetched
	SyncManager        *libsync.SyncManager // Real-time sync system for LocalFiles and progress
	// Global mapping system
	GlobalMappingService    *global_mapping.GlobalMappingService
	UserSubscriptionService *global_mapping.UserSubscriptionService
	ProgressSyncService     *global_mapping.ProgressSyncService
	FileWatcherService      *global_mapping.FileWatcherService
	SystemScanService       *SystemScanService
}
)

// NewApp creates a new server instance
func NewApp(configOpts *ConfigOptions, selfupdater *updater.SelfUpdater) *App {

	// Initialize logger with predefined format
	logger := util.NewLogger()

	// Log application version, OS, architecture and system info
	logger.Info().Msgf("app: Seanime %s-%s", constants.Version, constants.VersionName)
	logger.Info().Msgf("app: OS: %s", runtime.GOOS)
	logger.Info().Msgf("app: Arch: %s", runtime.GOARCH)
	logger.Info().Msgf("app: Processor count: %d", runtime.NumCPU())

	// Initialize hook manager for plugin event system
	hookManager := hook.NewHookManager(hook.NewHookManagerOptions{Logger: logger})
	hook.SetGlobalHookManager(hookManager)
	plugin.GlobalAppContext.SetLogger(logger)

	// Store current version to detect version changes
	previousVersion := constants.Version

	// Add callback to track version changes
	configOpts.OnVersionChange = append(configOpts.OnVersionChange, func(oldVersion string, newVersion string) {
		logger.Info().Str("prev", oldVersion).Str("current", newVersion).Msg("app: Version change detected")
		previousVersion = oldVersion
	})

	// Initialize configuration with provided options
	// Creates config directory if it doesn't exist
	cfg, err := NewConfig(configOpts, logger)
	if err != nil {
		logger.Fatal().Err(err).Msgf("app: Failed to initialize config")
	}


	// Create logs directory if it doesn't exist
	_ = os.MkdirAll(cfg.Logs.Dir, 0755)

	// Start background process to trim log files
	go TrimLogEntries(cfg.Logs.Dir, logger)

	logger.Info().Msgf("app: Data directory: %s", cfg.Data.AppDataDir)
	logger.Info().Msgf("app: Working directory: %s", cfg.Data.WorkingDir)

	// Log if running in desktop sidecar mode
	if configOpts.IsDesktopSidecar {
		logger.Info().Msg("app: Desktop sidecar mode enabled")
	}

	// Initialize database connection
	database, err := db.NewDatabase(cfg.Data.AppDataDir, cfg.Database.Name, logger)
	if err != nil {
		logger.Fatal().Err(err).Msgf("app: Failed to initialize database")
	}

	HandleNewDatabaseEntries(database, logger)

	// Clean up old database entries in background goroutines
	database.TrimLocalFileEntries()   // Remove old local file entries
	database.TrimScanSummaryEntries() // Remove old scan summaries

	// Get anime library paths for plugin context
	animeLibraryPaths, _ := database.GetAllLibraryPathsFromSettings()
	plugin.GlobalAppContext.SetModulesPartial(plugin.AppContextModules{
		Database:          database,
		AnimeLibraryPaths: &animeLibraryPaths,
	})

	// Get Anilist token from database if available
	anilistToken := database.GetAnilistToken()

	// Initialize Anilist API client with the token
	// If the token is empty, the client will not be authenticated
	anilistCW := anilist.NewAnilistClient(anilistToken)

	// Initialize WebSocket event manager for real-time communication
	wsEventManager := events.NewWSEventManager(logger)

	// Initialize SSE manager for Server-Sent Events
	sseManager := events.NewSSEManager(logger)

	// Exit if no WebSocket connections in desktop sidecar mode
	if configOpts.IsDesktopSidecar {
		wsEventManager.ExitIfNoConnsAsDesktopSidecar()
	}

	// Initialize sync manager for real-time LocalFile and progress tracking
	syncManager, err := libsync.NewSyncManager(&libsync.SyncManagerOptions{
		Database:     database,
		Logger:       logger,
		LibraryPaths: animeLibraryPaths,
	})
	if err != nil {
		logger.Fatal().Err(err).Msgf("app: Failed to initialize sync manager")
	}

	// Initialize global mapping services (progressSyncService will be initialized later)
	globalMappingService := global_mapping.NewGlobalMappingService(database, logger, wsEventManager)
	userSubscriptionService := global_mapping.NewUserSubscriptionService(database, logger, globalMappingService)
	fileWatcherService := global_mapping.NewFileWatcherService(logger, globalMappingService)

	// Initialize DNS-over-HTTPS service in background
	go doh.HandleDoH(cfg.Server.DoHUrl, logger)

	// Initialize file cache system for media and metadata
	fileCacher, err := filecache.NewCacher(cfg.Cache.Dir)
	if err != nil {
		logger.Fatal().Err(err).Msgf("app: Failed to initialize file cacher")
	}

	// Initialize extension repository
	extensionRepository := extension_repo.NewRepository(&extension_repo.NewRepositoryOptions{
		Logger:         logger,
		ExtensionDir:   cfg.Extensions.Dir,
		WSEventManager: wsEventManager,
		FileCacher:     fileCacher,
		HookManager:    hookManager,
	})
	// Load extensions in background
	go LoadExtensions(extensionRepository, logger, cfg)

	// Initialize metadata provider for media information
	metadataProvider := metadata.NewProvider(&metadata.NewProviderImplOptions{
		Logger:     logger,
		FileCacher: fileCacher,
	})

	// Use metadata provider directly
	activeMetadataProvider := metadataProvider

	// Initialize manga repository
	mangaRepository := manga.NewRepository(&manga.NewRepositoryOptions{
		Logger:         logger,
		FileCacher:     fileCacher,
		CacheDir:       cfg.Cache.Dir,
		ServerURI:      cfg.GetServerURI(),
		WsEventManager: wsEventManager,
		DownloadDir:    cfg.Manga.DownloadDir,
		Database:       database,
	})

	// Initialize Anilist platform
	anilistPlatform := anilist_platform.NewAnilistPlatform(anilistCW, logger)

	// Update plugin context with new modules
	plugin.GlobalAppContext.SetModulesPartial(plugin.AppContextModules{
		AnilistPlatform:  anilistPlatform,
		WSEventManager:   wsEventManager,
		MetadataProvider: metadataProvider,
	})

	// Initialize sync manager for offline/online synchronization
	localManager, err := local.NewManager(&local.NewManagerOptions{
		LocalDir:         cfg.Offline.Dir,
		AssetDir:         cfg.Offline.AssetDir,
		Logger:           logger,
		MetadataProvider: metadataProvider,
		MangaRepository:  mangaRepository,
		Database:         database,
		WSEventManager:   wsEventManager,
		IsOffline:        false,
		AnilistPlatform:  anilistPlatform,
	})
	if err != nil {
		logger.Fatal().Err(err).Msgf("app: Failed to initialize sync manager")
	}

	// Use AniList platform directly - multi-user system handles authentication per user
	activePlatform := anilistPlatform
	if !anilistCW.IsAuthenticated() {
		logger.Warn().Msg("app: AniList client is not authenticated - multi-token system will handle API calls when needed")
		// No fallback platform needed - multi-token system can use other authenticated users' tokens
	}

	// Initialize ProgressSyncService now that we have the active platform
	progressSyncService := global_mapping.NewProgressSyncService(database, logger, activePlatform)

	// Initialize SystemScanService for debounced file change notifications
	systemScanService := NewSystemScanService(&SystemScanServiceOptions{
		Logger:           logger,
		Database:         database,
		Platform:         activePlatform,
		MetadataProvider: activeMetadataProvider,
		WSEventManager:   wsEventManager,
		SSEManager:       sseManager,
		RateLimiter:      limiter.NewLimiter(time.Second, 10), // Rate limit for AniList API
		AnimeCache:       anilist.NewCompleteAnimeCache(),
		DebounceDelay:    5 * time.Second,  // 5 second debounce delay
		CooldownDuration: 1 * time.Minute, // 1 minute cooldown after scan
	})


	// Initialize extension playground for testing extensions
	extensionPlaygroundRepository := extension_playground.NewPlaygroundRepository(logger, activePlatform, activeMetadataProvider)


	// Create the main app instance with initialized components
	app := &App{
		Config:                        cfg,
		Database:                      database,
		AnilistClient:                 anilistCW,
		AnilistPlatform:               activePlatform,
		LocalManager:                  localManager,
		WSEventManager:                wsEventManager,
		SSEManager:                    sseManager,
		Logger:                        logger,
		Version:                       constants.Version,
		Updater:                       updater.New(constants.Version, logger, wsEventManager),
		FileCacher:                    fileCacher,
		MetadataProvider:              activeMetadataProvider,
		MangaRepository:               mangaRepository,
		ExtensionRepository:           extensionRepository,
		ExtensionPlaygroundRepository: extensionPlaygroundRepository,
		ReportRepository:              report.NewRepository(logger),
		TorrentRepository:             nil, // Initialized in App.initModulesOnce
		FillerManager:                 nil, // Initialized in App.initModulesOnce
		MangaDownloader:               nil, // Initialized in App.initModulesOnce
		PlaybackManager:               nil, // Initialized in App.initModulesOnce
		AutoDownloader:                nil, // Initialized in App.initModulesOnce
		AutoScanner:                   nil, // Initialized in App.initModulesOnce
		ContinuityManager:             nil, // Initialized in App.initModulesOnce
		DirectStreamManager:           nil, // Initialized in App.initModulesOnce
		NativePlayer:                  nil, // Initialized in App.initModulesOnce
		TorrentClientRepository:       nil, // Initialized in App.InitOrRefreshModules
		MediaPlayerRepository:         nil, // Initialized in App.InitOrRefreshModules
		DiscordPresence:               nil, // Initialized in App.InitOrRefreshModules
		previousVersion:               previousVersion,
		FeatureFlags:                  NewFeatureFlags(cfg, logger),
		IsDesktopSidecar:              configOpts.IsDesktopSidecar,
        SecondarySettings: struct {
            Torrentstream *models.TorrentstreamSettings
        }{Torrentstream: nil},
		SelfUpdater:                     selfupdater,
		moduleMu:                        sync.Mutex{},
		OnRefreshAnilistCollectionFuncs: result.NewResultMap[string, func()](),
		HookManager:                     hookManager,
		SyncManager:                     syncManager,
		// Global mapping system
		GlobalMappingService:    globalMappingService,
		UserSubscriptionService: userSubscriptionService,
		ProgressSyncService:     progressSyncService,
		FileWatcherService:      fileWatcherService,
		SystemScanService:       systemScanService,
	}

	// Database tables are created via GORM AutoMigrate during NewDatabase() - no migrations needed

	// Initialize modules that only need to be initialized once
	app.initModulesOnce()

    plugin.GlobalAppContext.SetModulesPartial(plugin.AppContextModules{
        IsOffline:               util.NewBool(false),
        ContinuityManager:       app.ContinuityManager,
        AutoScanner:             app.AutoScanner,
        AutoDownloader:          app.AutoDownloader,
        FileCacher:              app.FileCacher,
    })

    // Initialize all modules that depend on settings
    app.InitOrRefreshModules()

    // Fetch announcements after modules are initialized so updater respects settings
    go app.Updater.FetchAnnouncements()

	// Start sync manager for real-time LocalFile and progress tracking
	if app.SyncManager != nil {
		err = app.SyncManager.Start()
		if err != nil {
			logger.Error().Err(err).Msg("app: Failed to start sync manager")
		} else {
			logger.Info().Msg("app: Sync manager started successfully")
		}
	}

	// Load built-in extensions into extension consumers
	app.AddExtensionBankToConsumers()

	// Initialize Anilist data
	app.InitOrRefreshAnilistData()

	
	// Register sync manager cleanup
	if app.SyncManager != nil {
		app.AddCleanupFunction(app.SyncManager.Stop)
	}

	// Run one-time initialization actions
	app.performActionsOnce()

	return app
}


func (a *App) AddCleanupFunction(f func()) {
	a.Cleanups = append(a.Cleanups, f)
}
func (a *App) AddOnRefreshAnilistCollectionFunc(key string, f func()) {
	if key == "" {
		return
	}
	a.OnRefreshAnilistCollectionFuncs.Set(key, f)
}

func (a *App) Cleanup() {
	for _, f := range a.Cleanups {
		f()
	}
}
