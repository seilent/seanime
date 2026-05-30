package db

import (
	"errors"
	"seanime/internal/database/models"
	"time"

	"gorm.io/gorm"
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

// GetCachedMediaByID retrieves a single cached media row by AniList ID.
func (db *Database) GetCachedMediaByID(id int) (*models.CachedMedia, bool, error) {
	var row models.CachedMedia
	err := db.gormdb.Where("anilist_id = ?", id).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &row, true, nil
}

// GetReleasingAnimeIDs returns AniList IDs of all cached anime with status RELEASING.
func (db *Database) GetReleasingAnimeIDs() ([]int, error) {
	var ids []int
	err := db.gormdb.Model(&models.CachedMedia{}).
		Where("status = ? AND type = ?", "RELEASING", "anime").
		Pluck("anilist_id", &ids).Error
	return ids, err
}

// GetReleasingAnimeIDsInLibrary returns AniList IDs of releasing anime that have local files.
func (db *Database) GetReleasingAnimeIDsInLibrary() ([]int, error) {
	var ids []int
	err := db.gormdb.Raw(`
		SELECT DISTINCT cm.anilist_id FROM cached_media cm
		INNER JOIN global_anime_file_mappings gm ON cm.anilist_id = gm.anilist_id
		WHERE cm.status = ? AND cm.type = ?
	`, "RELEASING", "anime").Pluck("anilist_id", &ids).Error
	return ids, err
}

// SetSubsPleaseSid stores the SubsPlease sid for a cached media entry.
func (db *Database) SetSubsPleaseSid(anilistID int, sid string) error {
	return db.gormdb.Model(&models.CachedMedia{}).
		Where("anilist_id = ?", anilistID).
		Update("subsplease_sid", sid).Error
}

// SetSubspleaseEpisodeCount updates the cached SubsPlease episode count.
func (db *Database) SetSubspleaseEpisodeCount(anilistID int, count int) error {
	return db.gormdb.Model(&models.CachedMedia{}).
		Where("anilist_id = ?", anilistID).
		Update("subsplease_episode_count", count).Error
}

// GetSubspleaseInfo returns the cached SubsPlease sid and episode count for an anime.
func (db *Database) GetSubspleaseInfo(anilistID int) (sid string, episodeCount int, err error) {
	var row models.CachedMedia
	err = db.gormdb.Select("subsplease_sid, subsplease_episode_count").
		Where("anilist_id = ?", anilistID).First(&row).Error
	return row.SubsPleaseSid, row.SubspleaseEpisodeCount, err
}

// HasSubsPleaseSid checks if an anime has a cached SubsPlease sid.
func (db *Database) HasSubsPleaseSid(anilistID int) bool {
	var sid string
	db.gormdb.Model(&models.CachedMedia{}).
		Where("anilist_id = ?", anilistID).
		Pluck("subsplease_sid", &sid)
	return sid != ""
}

// UpsertCachedMediaDetail upserts only the detail-page blob columns on a CachedMedia row.
// The column set is disjoint from UpsertCachedMediaBatch to avoid clobbering collection data.
func (db *Database) UpsertCachedMediaDetail(id int, mediaType string, detail []byte) error {
	row := models.CachedMedia{
		AnilistID:       id,
		Type:            mediaType,
		DetailData:      detail,
		DetailUpdatedAt: time.Now(),
	}
	return db.gormdb.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "anilist_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "detail_data", "detail_updated_at"}),
	}).Create(&row).Error
}
