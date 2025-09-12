package db_bridge

import (
	"github.com/goccy/go-json"
	"github.com/samber/mo"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"seanime/internal/library/anime"
)

var CurrLocalFilesDbId uint
var CurrLocalFiles mo.Option[[]*anime.LocalFile]

// GetLocalFiles will return the latest local files and the id of the entry.
// This is now shared across all users (admin controlled).
func GetLocalFiles(db *db.Database) ([]*anime.LocalFile, uint, error) {

	if CurrLocalFiles.IsPresent() {
		return CurrLocalFiles.MustGet(), CurrLocalFilesDbId, nil
	}

	// Get the latest entry (shared across all users)
	var res models.LocalFiles
	err := db.Gorm().Last(&res).Error
	if err != nil {
		return nil, 0, err
	}

	// Unmarshal the local files
	lfsBytes := res.Value
	var lfs []*anime.LocalFile
	if err := json.Unmarshal(lfsBytes, &lfs); err != nil {
		return nil, 0, err
	}

	db.Logger.Debug().Msg("db: Shared local files retrieved")

	CurrLocalFiles = mo.Some(lfs)
	CurrLocalFilesDbId = res.ID

	return lfs, res.ID, nil
}

// GetLocalFilesForUser will return the latest local files for a specific user and the id of the entry.
func GetLocalFilesForUser(db *db.Database, userID uint) ([]*anime.LocalFile, uint, error) {

	// Get the latest entry for this user
	var res models.LocalFiles
	err := db.Gorm().Where("user_id = ?", userID).Last(&res).Error
	if err != nil {
		return nil, 0, err
	}

	// Unmarshal the local files
	lfsBytes := res.Value
	var lfs []*anime.LocalFile
	if err := json.Unmarshal(lfsBytes, &lfs); err != nil {
		return nil, 0, err
	}

	db.Logger.Debug().Uint("userId", userID).Msg("db: User-specific local files retrieved")

	return lfs, res.ID, nil
}

// SaveLocalFiles will save the local files in the database at the given id.
func SaveLocalFiles(db *db.Database, lfsId uint, lfs []*anime.LocalFile) ([]*anime.LocalFile, error) {
	// Marshal the local files
	marshaledLfs, err := json.Marshal(lfs)
	if err != nil {
		return nil, err
	}

	// Save the local files
	ret, err := db.UpsertLocalFiles(&models.LocalFiles{
		BaseModel: models.BaseModel{
			ID: lfsId,
		},
		Value: marshaledLfs,
	})
	if err != nil {
		return nil, err
	}

	// Unmarshal the saved local files
	var retLfs []*anime.LocalFile
	if err := json.Unmarshal(ret.Value, &retLfs); err != nil {
		return lfs, nil
	}

	CurrLocalFiles = mo.Some(retLfs)
	CurrLocalFilesDbId = ret.ID

	return retLfs, nil
}

// InsertLocalFiles will insert the local files in the database at a new entry.
func InsertLocalFiles(db *db.Database, lfs []*anime.LocalFile) ([]*anime.LocalFile, error) {

	// Marshal the local files
	bytes, err := json.Marshal(lfs)
	if err != nil {
		return nil, err
	}

	// Save the local files to the database
	ret, err := db.InsertLocalFiles(&models.LocalFiles{
		Value: bytes,
	})

	if err != nil {
		return nil, err
	}

	CurrLocalFiles = mo.Some(lfs)
	CurrLocalFilesDbId = ret.ID

	return lfs, nil

}

// InsertLocalFilesForUser will insert the local files in the database for a specific user at a new entry.
func InsertLocalFilesForUser(db *db.Database, lfs []*anime.LocalFile, userID uint) ([]*anime.LocalFile, error) {

	// Marshal the local files
	bytes, err := json.Marshal(lfs)
	if err != nil {
		return nil, err
	}

	// Save the local files to the database for this user
	_, err = db.InsertLocalFiles(&models.LocalFiles{
		UserID: userID,
		Value:  bytes,
	})

	if err != nil {
		return nil, err
	}

	// Don't update global cache since this is user-specific
	db.Logger.Debug().Uint("userId", userID).Msg("db: User-specific local files inserted")

	return lfs, nil

}
