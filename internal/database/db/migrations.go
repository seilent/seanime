package db

import (
	"seanime/internal/database/models"
	"time"
)

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


// MigrateSharedDataTables removes UserID from tables that should be shared across users
func (db *Database) MigrateSharedDataTables() error {
	db.Logger.Info().Msg("database: Migrating shared data tables (removing UserID columns)")

	// Drop UserID columns from shared tables
	// Note: GORM AutoMigrate doesn't drop columns, so we need to do it manually
	
	// Check if columns exist before trying to drop them
	if db.gormdb.Migrator().HasColumn(&models.LocalFiles{}, "user_id") {
		if err := db.gormdb.Migrator().DropColumn(&models.LocalFiles{}, "user_id"); err != nil {
			db.Logger.Error().Err(err).Msg("database: Failed to drop user_id from local_files")
			return err
		}
		db.Logger.Info().Msg("database: Dropped user_id column from local_files table")
	}

	if db.gormdb.Migrator().HasColumn(&models.ScanSummary{}, "user_id") {
		if err := db.gormdb.Migrator().DropColumn(&models.ScanSummary{}, "user_id"); err != nil {
			db.Logger.Error().Err(err).Msg("database: Failed to drop user_id from scan_summaries")
			return err
		}
		db.Logger.Info().Msg("database: Dropped user_id column from scan_summaries table")
	}

	if db.gormdb.Migrator().HasColumn(&models.AutoDownloaderRule{}, "user_id") {
		if err := db.gormdb.Migrator().DropColumn(&models.AutoDownloaderRule{}, "user_id"); err != nil {
			db.Logger.Error().Err(err).Msg("database: Failed to drop user_id from auto_downloader_rules")
			return err
		}
		db.Logger.Info().Msg("database: Dropped user_id column from auto_downloader_rules table")
	}

	db.Logger.Info().Msg("database: Shared data tables migration completed successfully")
	return nil
}
