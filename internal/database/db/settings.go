package db

import (
	"seanime/internal/database/models"

	"gorm.io/gorm/clause"
)

var CurrSettings *models.Settings

func (db *Database) UpsertSettings(settings *models.Settings) (*models.Settings, error) {

	err := db.gormdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(settings).Error

	if err != nil {
		db.Logger.Error().Err(err).Msg("db: Failed to save settings in the database")
		return nil, err
	}

	CurrSettings = settings

	db.Logger.Debug().Msg("db: Settings saved")
	return settings, nil

}

func (db *Database) GetSettings() (*models.Settings, error) {

	if CurrSettings != nil {
		return CurrSettings, nil
	}

	var settings models.Settings
	err := db.gormdb.Where("id = ?", 1).Find(&settings).Error

	if err != nil {
		return nil, err
	}
	return &settings, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (db *Database) GetLibraryPathFromSettings() (string, error) {
	// Library path should be global - try to get from system settings first (user_id = 0)
	var settings models.Settings
	err := db.gormdb.Where("user_id = ?", 0).First(&settings).Error
	if err == nil && settings.Library.LibraryPath != "" {
		return settings.Library.LibraryPath, nil
	}
	
	// Fallback to the first non-empty library path from any user
	err = db.gormdb.Where("library_path != '' AND library_path IS NOT NULL").First(&settings).Error
	if err != nil {
		return "", err
	}
	
	return settings.Library.LibraryPath, nil
}

func (db *Database) GetAdditionalLibraryPathsFromSettings() ([]string, error) {
	// Library paths should be global - try to get from system settings first (user_id = 0)
	var settings models.Settings
	err := db.gormdb.Where("user_id = ?", 0).First(&settings).Error
	if err == nil && len(settings.Library.LibraryPaths) > 0 {
		return settings.Library.LibraryPaths, nil
	}
	
	// Fallback to the first non-empty library paths from any user
	err = db.gormdb.Where("library_paths != '' AND library_paths IS NOT NULL").First(&settings).Error
	if err != nil {
		return []string{}, nil // Return empty slice instead of error
	}
	
	return settings.Library.LibraryPaths, nil
}

func (db *Database) GetAllLibraryPathsFromSettings() ([]string, error) {
	settings, err := db.GetSettings()
	if err != nil {
		return []string{}, err
	}
	if settings.Library == nil {
		return []string{}, nil
	}
	return append([]string{settings.Library.LibraryPath}, settings.Library.LibraryPaths...), nil
}

func (db *Database) AllLibraryPathsFromSettings(settings *models.Settings) *[]string {
	if settings.Library == nil {
		return &[]string{}
	}
	r := append([]string{settings.Library.LibraryPath}, settings.Library.LibraryPaths...)
	return &r
}

func (db *Database) AutoUpdateProgressIsEnabled() (bool, error) {
	settings, err := db.GetSettings()
	if err != nil {
		return false, err
	}
	return settings.Library.AutoUpdateProgress, nil
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var CurrMediastreamSettings *models.MediastreamSettings

func (db *Database) UpsertMediastreamSettings(settings *models.MediastreamSettings) (*models.MediastreamSettings, error) {

	err := db.gormdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(settings).Error

	if err != nil {
		db.Logger.Error().Err(err).Msg("db: Failed to save media streaming settings in the database")
		return nil, err
	}

	CurrMediastreamSettings = settings

	db.Logger.Debug().Msg("db: Media streaming settings saved")
	return settings, nil

}

func (db *Database) GetMediastreamSettings() (*models.MediastreamSettings, bool) {

	if CurrMediastreamSettings != nil {
		return CurrMediastreamSettings, true
	}

	var settings models.MediastreamSettings
	err := db.gormdb.Where("id = ?", 1).First(&settings).Error

	if err != nil {
		return nil, false
	}
	return &settings, true
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

var CurrentDebridSettings *models.DebridSettings

func (db *Database) UpsertDebridSettings(settings *models.DebridSettings) (*models.DebridSettings, error) {
	err := db.gormdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(settings).Error

	if err != nil {
		db.Logger.Error().Err(err).Msg("db: Failed to save debrid settings in the database")
		return nil, err
	}

	CurrentDebridSettings = settings

	db.Logger.Debug().Msg("db: Debrid settings saved")
	return settings, nil
}

func (db *Database) GetDebridSettings() (*models.DebridSettings, bool) {

	if CurrentDebridSettings != nil {
		return CurrentDebridSettings, true
	}

	var settings models.DebridSettings
	err := db.gormdb.Where("id = ?", 1).First(&settings).Error
	if err != nil {
		return nil, false
	}
	return &settings, true
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
