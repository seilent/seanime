package db

import (
	"seanime/internal/database/models"
	"time"

	"gorm.io/gorm/clause"
)

// UpsertCachedMediaBatch upserts a batch of cached media records.
func (db *Database) UpsertCachedMediaBatch(items []*models.CachedMedia) error {
	if len(items) == 0 {
		return nil
	}
	return db.gormdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "anilist_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "format", "status", "season", "season_year", "title_romaji", "data", "updated_at"}),
	}).Create(&items).Error
}

// GetCachedMediaByIDs returns cached media keyed by AniList ID.
func (db *Database) GetCachedMediaByIDs(ids []int) (map[int]*models.CachedMedia, error) {
	if len(ids) == 0 {
		return make(map[int]*models.CachedMedia), nil
	}
	var results []*models.CachedMedia
	err := db.gormdb.Where("anilist_id IN ?", ids).Find(&results).Error
	if err != nil {
		return nil, err
	}
	m := make(map[int]*models.CachedMedia, len(results))
	for _, r := range results {
		m[r.AnilistID] = r
	}
	return m, nil
}

// GetCachedUserMediaList retrieves a cached collection skeleton for a user+variant.
func (db *Database) GetCachedUserMediaList(userID uint, variant string) (*models.CachedUserMediaList, bool, error) {
	var row models.CachedUserMediaList
	err := db.gormdb.Where("user_id = ? AND variant = ?", userID, variant).First(&row).Error
	if err != nil {
		if err.Error() == "record not found" {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &row, true, nil
}

// UpsertCachedUserMediaList upserts the collection skeleton for a user+variant.
func (db *Database) UpsertCachedUserMediaList(userID uint, variant string, data []byte) error {
	now := time.Now()
	row := models.CachedUserMediaList{
		UserID:  userID,
		Variant: variant,
		Data:    data,
	}
	row.UpdatedAt = now
	return db.gormdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "variant"}},
		DoUpdates: clause.AssignmentColumns([]string{"data", "updated_at"}),
	}).Create(&row).Error
}
