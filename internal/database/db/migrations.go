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

// IsMultiUserEnabled checks if multi-user mode is enabled (i.e., any users exist)
func (db *Database) IsMultiUserEnabled() bool {
	var userCount int64
	if err := db.gormdb.Model(&models.User{}).Count(&userCount).Error; err != nil {
		return false
	}
	return userCount > 0
}
