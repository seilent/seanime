package db

import (
	"seanime/internal/database/models"

	"gorm.io/gorm/clause"
)

// Global Settings (server-wide)
var CurrGlobalSettings *models.GlobalSettings

func (db *Database) UpsertGlobalSettings(settings *models.GlobalSettings) (*models.GlobalSettings, error) {
	err := db.gormdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(settings).Error

	if err != nil {
		db.Logger.Error().Err(err).Msg("db: Failed to save global settings in the database")
		return nil, err
	}

	CurrGlobalSettings = settings
	db.Logger.Debug().Msg("db: Global settings saved")
	return settings, nil
}

func (db *Database) GetGlobalSettings() (*models.GlobalSettings, error) {
	if CurrGlobalSettings != nil {
		return CurrGlobalSettings, nil
	}

	var settings models.GlobalSettings
	err := db.gormdb.Where("id = ?", 1).First(&settings).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// User Settings (per-user)
func (db *Database) UpsertSettings(settings *models.Settings) (*models.Settings, error) {
	err := db.gormdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(settings).Error

	if err != nil {
		db.Logger.Error().Err(err).Msg("db: Failed to save user settings in the database")
		return nil, err
	}

	db.Logger.Debug().Msg("db: User settings saved")
	return settings, nil
}

// DEPRECATED: Use GetGlobalSettings() for server-wide settings or GetSettingsForUser(userID) for user-specific settings
// This method is kept for legacy compatibility but should not be used in new code
func (db *Database) GetSettings() (*models.Settings, error) {
	db.Logger.Warn().Msg("db: GetSettings() is deprecated. Use GetGlobalSettings() or GetSettingsForUser(userID)")

	var settings models.Settings
	err := db.gormdb.Where("id = ?", 1).First(&settings).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// Legacy compatibility methods
func (db *Database) GetGlobalSettingsSetupStatus() (bool, error) {
	settings, err := db.GetGlobalSettings()
	if err != nil {
		return false, err
	}
	return settings.SetupCompleted, nil
}

func (db *Database) SetGlobalSettingsSetupCompleted(completed bool) error {
	return db.gormdb.Table("global_settings").Where("id = ?", 1).Update("setup_completed", completed).Error
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (db *Database) GetLibraryPathFromSettings() (string, error) {
	globalSettings, err := db.GetGlobalSettings()
	if err != nil {
		return "", err
	}

	if globalSettings.Library != nil && globalSettings.Library.LibraryPath != "" {
		return globalSettings.Library.LibraryPath, nil
	}

	return "", nil
}

func (db *Database) GetAdditionalLibraryPathsFromSettings() ([]string, error) {
	globalSettings, err := db.GetGlobalSettings()
	if err != nil {
		return []string{}, nil
	}

	if globalSettings.Library != nil && len(globalSettings.Library.LibraryPaths) > 0 {
		return globalSettings.Library.LibraryPaths, nil
	}

	return []string{}, nil
}

func (db *Database) GetAllLibraryPathsFromSettings() ([]string, error) {
	globalSettings, err := db.GetGlobalSettings()
	if err != nil {
		return []string{}, err
	}
	if globalSettings.Library == nil {
		return []string{}, nil
	}
	return append([]string{globalSettings.Library.LibraryPath}, globalSettings.Library.LibraryPaths...), nil
}

func (db *Database) AllLibraryPathsFromSettings(globalSettings *models.GlobalSettings) *[]string {
	if globalSettings == nil || globalSettings.Library == nil {
		return &[]string{}
	}
	r := append([]string{globalSettings.Library.LibraryPath}, globalSettings.Library.LibraryPaths...)
	return &r
}

func (db *Database) AutoUpdateProgressIsEnabled() (bool, error) {
	return true, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var CurrTorrentstreamSettings *models.TorrentstreamSettings

func (db *Database) UpsertTorrentstreamSettings(settings *models.TorrentstreamSettings) (*models.TorrentstreamSettings, error) {

	err := db.gormdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(settings).Error

	if err != nil {
		db.Logger.Error().Err(err).Msg("db: Failed to save torrent streaming settings in the database")
		return nil, err
	}

	CurrTorrentstreamSettings = settings

	db.Logger.Debug().Msg("db: Torrent streaming settings saved")
	return settings, nil
}

func (db *Database) GetTorrentstreamSettings() (*models.TorrentstreamSettings, bool) {

	if CurrTorrentstreamSettings != nil {
		return CurrTorrentstreamSettings, true
	}

	var settings models.TorrentstreamSettings
	err := db.gormdb.Where("id = ?", 1).First(&settings).Error

	if err != nil {
		return nil, false
	}
	return &settings, true
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Debrid feature removed: CurrentDebridSettings removed

// Debrid feature removed: UpsertDebridSettings deleted

// Debrid feature removed: GetDebridSettings deleted

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
