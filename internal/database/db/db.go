package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"seanime/internal/database/models"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/rs/zerolog"
	"github.com/samber/mo"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Database struct {
	gormdb           *gorm.DB
	Logger           *zerolog.Logger
	CurrMediaFillers mo.Option[map[int]*MediaFillerItem]
}

func (db *Database) Gorm() *gorm.DB {
	return db.gormdb
}

func NewDatabase(appDataDir, dbName string, logger *zerolog.Logger) (*Database, error) {

	// Set the SQLite database path
	var sqlitePath string
	if os.Getenv("TEST_ENV") == "true" {
		sqlitePath = ":memory:"
	} else {
		sqlitePath = filepath.Join(appDataDir, dbName+".db")
	}

	// Connect to the SQLite database
	db, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{
		Logger: gormlogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			gormlogger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  gormlogger.Error,
				IgnoreRecordNotFoundError: true,
				ParameterizedQueries:      false,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		return nil, err
	}

	// Migrate tables
	err = migrateTables(db)
	if err != nil {
		logger.Fatal().Err(err).Msg("db: Failed to perform auto migration")
		return nil, err
	}

	// One-time data migration: remap removed 'animetosho' provider to 'nyaa'
	migrateAnimetoshoToNyaa(db, logger)

	database := &Database{
		gormdb:           db,
		Logger:           logger,
		CurrMediaFillers: mo.None[map[int]*MediaFillerItem](),
	}

	// Initialize multi-user tables
	err = database.InitializeMultiUserTables()
	if err != nil {
		logger.Fatal().Err(err).Msg("db: Failed to initialize multi-user tables")
		return nil, err
	}

	logger.Info().Str("name", fmt.Sprintf("%s.db", dbName)).Msg("db: Database instantiated")

	return database, nil
}

// InitializeMultiUserTables ensures multi-user tables exist
func (db *Database) InitializeMultiUserTables() error {
	db.Logger.Info().Msg("database: Initializing multi-user tables")

	// Create multi-user tables
	if err := db.gormdb.AutoMigrate(&models.User{}, &models.UserSession{}); err != nil {
		return err
	}

	// Start session cleanup
	go db.startSessionCleanup()

	db.Logger.Info().Msg("database: Multi-user tables initialized successfully")
	return nil
}

// startSessionCleanup starts a goroutine to periodically clean up expired sessions
func (db *Database) startSessionCleanup() {
	ticker := time.NewTicker(1 * time.Hour) // Clean up every hour
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			if err := db.CleanupExpiredSessions(); err != nil {
				db.Logger.Error().Err(err).Msg("database: Failed to cleanup expired sessions")
			}
		}
	}()
}

// MigrateTables performs auto migration on the database
func migrateTables(db *gorm.DB) error {
	err := db.AutoMigrate(
		&models.GlobalSettings{},
		&models.Settings{},
		&models.Account{},
		&models.Mal{},
		&models.ScanSummary{},
		&models.AutoDownloaderRule{},
		&models.AutoDownloaderItem{},
		&models.SilencedMediaEntry{},
		&models.Theme{},
		&models.ChapterDownloadQueueItem{},
		&models.TorrentstreamSettings{},
		&models.TorrentstreamHistory{},
		&models.MediaFiller{},
		&models.MangaMapping{},
		&models.OnlinestreamMapping{},
		&models.PluginData{},
		// User-specific models for multi-user support
		&models.UserLibraryEntry{},
		&models.UserEpisodeProgress{},
		&models.UserActivePlayback{},
		&models.UserMediaSubscription{},
		&models.UserPreference{},
		// Global mapping models
		&models.GlobalAnimeFileMapping{},
		&models.UnmappedFile{},
		&models.UserAnimeSubscription{},
		&models.UserLibrarySubscription{},
		&models.UserProgressSyncItem{},
		&models.PendingDownloadIntent{},
		//&models.MangaChapterContainer{},
		// Collection SWR cache
		&models.CachedMedia{},
		&models.CachedUserMediaList{},
	)
	if err != nil {

		return err
	}

	return nil
}

// migrateAnimetoshoToNyaa remaps the removed 'animetosho' torrent provider to 'nyaa'.
// Idempotent: the WHERE clause ensures no-op if already migrated.
func migrateAnimetoshoToNyaa(db *gorm.DB, logger *zerolog.Logger) {
	r1 := db.Exec(`UPDATE global_settings SET torrent_provider = 'nyaa' WHERE torrent_provider = 'animetosho'`)
	r2 := db.Exec(`UPDATE global_settings SET auto_downloader_provider = 'nyaa' WHERE auto_downloader_provider = 'animetosho'`)

	affected := r1.RowsAffected + r2.RowsAffected
	if affected > 0 {
		logger.Info().Int64("rows", affected).Msg("db: Migrated animetosho -> nyaa provider")
	}
}