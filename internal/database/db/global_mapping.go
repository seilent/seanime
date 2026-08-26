package db

import (
	"seanime/internal/database/models"
	"time"
)

func (db *Database) GetCompletedEpisodeNumbers(userID uint, mediaID int) ([]int, error) {
	var eps []int
	err := db.gormdb.Model(&models.UserEpisodeProgress{}).
		Where("user_id = ? AND media_id = ? AND is_completed = ?", userID, mediaID, true).
		Pluck("episode_number", &eps).Error
	return eps, err
}

func (db *Database) IsEpisodeCompletedByActiveWatchers(mediaID int, episode int) (bool, error) {
	var activeWatcherCount int64
	if err := db.gormdb.Raw(`
		SELECT COUNT(DISTINCT uas.user_id) FROM user_anime_subscriptions uas
		INNER JOIN user_episode_progresses uep
			ON uep.user_id = uas.user_id AND uep.media_id = uas.anilist_id
		WHERE uas.anilist_id = ?
	`, mediaID).Scan(&activeWatcherCount).Error; err != nil {
		return false, err
	}
	if activeWatcherCount == 0 {
		return false, nil
	}

	var completedCount int64
	if err := db.gormdb.Raw(`
		SELECT COUNT(DISTINCT uas.user_id) FROM user_anime_subscriptions uas
		INNER JOIN user_episode_progresses uep
			ON uep.user_id = uas.user_id AND uep.media_id = uas.anilist_id
		WHERE uas.anilist_id = ? AND uep.episode_number = ? AND uep.is_completed = 1
	`, mediaID, episode).Scan(&completedCount).Error; err != nil {
		return false, err
	}

	return completedCount >= activeWatcherCount, nil
}

func (db *Database) GetEpisodeWatchTimes(mediaID int) (map[int]time.Time, error) {
	var rows []models.UserEpisodeProgress
	err := db.gormdb.
		Where("media_id = ? AND is_completed = ?", mediaID, true).
		Order("last_watched_at ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int]time.Time)
	for _, r := range rows {
		if r.LastWatchedAt.After(result[r.EpisodeNumber]) {
			result[r.EpisodeNumber] = r.LastWatchedAt
		}
	}
	return result, nil
}

func (db *Database) GetEpisodeLastWatched(mediaID int, episode int) (time.Time, error) {
	var lastWatched time.Time
	err := db.gormdb.Raw(`
		SELECT MAX(last_watched_at) FROM user_episode_progresses
		WHERE media_id = ? AND episode_number = ?
	`, mediaID, episode).Scan(&lastWatched).Error
	return lastWatched, err
}

func (db *Database) GetMediaActivityTimes() (map[int]time.Time, error) {
	var rows []models.UserEpisodeProgress
	err := db.gormdb.Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[int]time.Time)
	for _, r := range rows {
		if r.LastWatchedAt.After(result[r.MediaID]) {
			result[r.MediaID] = r.LastWatchedAt
		}
	}
	return result, nil
}

func (db *Database) GetGlobalMapping(filePath string) (*models.GlobalAnimeFileMapping, error) {
	var mapping models.GlobalAnimeFileMapping
	if err := db.gormdb.Where("local_file_path = ?", filePath).First(&mapping).Error; err != nil {
		return nil, err
	}
	return &mapping, nil
}

// GetGlobalMappingsByAniListID retrieves all global mappings for an AniList ID
func (db *Database) GetGlobalMappingsByAniListID(aniListID int) ([]*models.GlobalAnimeFileMapping, error) {
	var mappings []*models.GlobalAnimeFileMapping
	if err := db.gormdb.Where("anilist_id = ?", aniListID).Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

// GetAllGlobalMappings retrieves all global mappings
func (db *Database) GetAllGlobalMappings() ([]*models.GlobalAnimeFileMapping, error) {
	var mappings []*models.GlobalAnimeFileMapping
	if err := db.gormdb.Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

// CreateGlobalMapping creates a new global mapping
func (db *Database) CreateGlobalMapping(mapping *models.GlobalAnimeFileMapping) error {
	return db.gormdb.Create(mapping).Error
}

// UpdateGlobalMapping updates an existing global mapping
func (db *Database) UpdateGlobalMapping(mapping *models.GlobalAnimeFileMapping) error {
	return db.gormdb.Save(mapping).Error
}

// DeleteGlobalMapping deletes a global mapping by file path
func (db *Database) DeleteGlobalMapping(filePath string) error {
	return db.gormdb.Where("local_file_path = ?", filePath).Delete(&models.GlobalAnimeFileMapping{}).Error
}

// ClearAllGlobalMappings deletes all global mappings
func (db *Database) ClearAllGlobalMappings() error {
	return db.gormdb.Where("1 = 1").Delete(&models.GlobalAnimeFileMapping{}).Error
}

// SetGlobalMappingIgnored sets the ignored flag for a mapping by path
func (db *Database) SetGlobalMappingIgnored(filePath string, ignored bool) error {
	return db.gormdb.Model(&models.GlobalAnimeFileMapping{}).
		Where("local_file_path = ?", filePath).
		Updates(map[string]interface{}{"ignored": ignored, "anilist_id": 0}).Error
}

// UpdateGlobalMappingMediaId updates the anilist_id for a mapping by path
func (db *Database) UpdateGlobalMappingMediaId(filePath string, mediaId int) error {
	return db.gormdb.Model(&models.GlobalAnimeFileMapping{}).
		Where("local_file_path = ?", filePath).
		Update("anilist_id", mediaId).Error
}

// DeleteGlobalMappingsByAniListID deletes all mappings for a given AniList media ID
func (db *Database) DeleteGlobalMappingsByAniListID(aniListID int) error {
	return db.gormdb.Where("anilist_id = ?", aniListID).Delete(&models.GlobalAnimeFileMapping{}).Error
}

// DeleteGlobalMappingsByPaths deletes mappings for the given paths
func (db *Database) DeleteGlobalMappingsByPaths(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	return db.gormdb.Where("local_file_path IN ?", paths).Delete(&models.GlobalAnimeFileMapping{}).Error
}

// UpsertGlobalMapping creates or updates a mapping by path
func (db *Database) UpsertGlobalMapping(mapping *models.GlobalAnimeFileMapping) error {
	var existing models.GlobalAnimeFileMapping
	err := db.gormdb.Where("local_file_path = ?", mapping.LocalFilePath).First(&existing).Error
	if err == nil {
		mapping.ID = existing.ID
		return db.gormdb.Save(mapping).Error
	}
	return db.gormdb.Create(mapping).Error
}

// Unmapped Files Operations

// CreateUnmappedFile creates a new unmapped file record
func (db *Database) CreateUnmappedFile(unmappedFile *models.UnmappedFile) error {
	return db.gormdb.Create(unmappedFile).Error
}

// GetUnmappedFiles retrieves all unmapped files with a specific status
func (db *Database) GetUnmappedFiles(status string) ([]*models.UnmappedFile, error) {
	var unmappedFiles []*models.UnmappedFile
	query := db.gormdb
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&unmappedFiles).Error; err != nil {
		return nil, err
	}
	return unmappedFiles, nil
}

// UpdateUnmappedFile updates an unmapped file record
func (db *Database) UpdateUnmappedFile(unmappedFile *models.UnmappedFile) error {
	return db.gormdb.Save(unmappedFile).Error
}

// DeleteUnmappedFile deletes an unmapped file record
func (db *Database) DeleteUnmappedFile(filePath string) error {
	return db.gormdb.Where("local_file_path = ?", filePath).Delete(&models.UnmappedFile{}).Error
}

// User Anime Subscription Operations (New Multi-Token System)

// CreateUserAnimeSubscription creates a new user anime subscription
func (db *Database) CreateUserAnimeSubscription(subscription *models.UserAnimeSubscription) error {
	return db.gormdb.Create(subscription).Error
}

// UpsertUserAnimeSubscription creates or updates a user anime subscription
func (db *Database) UpsertUserAnimeSubscription(userID uint, aniListID int, tokenStatus string) error {
	subscription := &models.UserAnimeSubscription{
		UserID:       userID,
		AniListID:    aniListID,
		LastVerified: time.Now(),
		TokenStatus:  tokenStatus,
		AddedAt:      time.Now(),
	}

	// Use GORM's FirstOrCreate for upsert functionality
	result := db.gormdb.Where("user_id = ? AND anilist_id = ?", userID, aniListID).
		Assign(map[string]interface{}{
			"last_verified": time.Now(),
			"token_status":  tokenStatus,
		}).
		FirstOrCreate(subscription)

	return result.Error
}

// GetUsersWithAnime retrieves all users who have a specific anime (for token rotation)
func (db *Database) GetUsersWithAnime(aniListID int) ([]*models.UserAnimeSubscription, error) {
	var subscriptions []*models.UserAnimeSubscription
	if err := db.gormdb.Where("anilist_id = ? AND token_status = ?", aniListID, "active").
		Order("last_verified ASC").Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// GetUserAnimeSubscriptions retrieves all anime subscriptions for a user
func (db *Database) GetUserAnimeSubscriptions(userID uint) ([]*models.UserAnimeSubscription, error) {
	var subscriptions []*models.UserAnimeSubscription
	if err := db.gormdb.Where("user_id = ?", userID).Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// UpdateTokenStatus updates the token status for a user-anime subscription
func (db *Database) UpdateTokenStatus(userID uint, aniListID int, status string) error {
	return db.gormdb.Model(&models.UserAnimeSubscription{}).
		Where("user_id = ? AND anilist_id = ?", userID, aniListID).
		Update("token_status", status).
		Update("last_verified", time.Now()).Error
}

// DeleteUserAnimeSubscription deletes a user anime subscription
func (db *Database) DeleteUserAnimeSubscription(userID uint, aniListID int) error {
	return db.gormdb.Where("user_id = ? AND anilist_id = ?", userID, aniListID).Delete(&models.UserAnimeSubscription{}).Error
}

// User Library Subscription Operations (Legacy)

// CreateUserLibrarySubscription creates a new user library subscription
func (db *Database) CreateUserLibrarySubscription(subscription *models.UserLibrarySubscription) error {
	return db.gormdb.Create(subscription).Error
}

// GetUserLibrarySubscriptions retrieves all subscriptions for a user
func (db *Database) GetUserLibrarySubscriptions(userID uint) ([]*models.UserLibrarySubscription, error) {
	var subscriptions []*models.UserLibrarySubscription
	if err := db.gormdb.Where("user_id = ?", userID).Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// GetUsersSubscribedToAnime retrieves all users subscribed to a specific anime
func (db *Database) GetUsersSubscribedToAnime(aniListID int) ([]*models.UserLibrarySubscription, error) {
	var subscriptions []*models.UserLibrarySubscription
	if err := db.gormdb.Where("anilist_id = ?", aniListID).Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// DeleteUserLibrarySubscription deletes a user library subscription
func (db *Database) DeleteUserLibrarySubscription(userID uint, aniListID int) error {
	return db.gormdb.Where("user_id = ? AND anilist_id = ?", userID, aniListID).Delete(&models.UserLibrarySubscription{}).Error
}

// IsUserSubscribedToAnime checks if a user is subscribed to an anime
func (db *Database) IsUserSubscribedToAnime(userID uint, aniListID int) (bool, error) {
	var count int64
	if err := db.gormdb.Model(&models.UserLibrarySubscription{}).
		Where("user_id = ? AND anilist_id = ?", userID, aniListID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// User Progress Sync Operations

// CreateUserProgressSyncItem creates a new progress sync item
func (db *Database) CreateUserProgressSyncItem(syncItem *models.UserProgressSyncItem) error {
	return db.gormdb.Create(syncItem).Error
}

// GetPendingUserProgressSyncItems retrieves all pending sync items
func (db *Database) GetPendingUserProgressSyncItems() ([]*models.UserProgressSyncItem, error) {
	var syncItems []*models.UserProgressSyncItem
	if err := db.gormdb.Where("sync_status = ?", "PENDING").Find(&syncItems).Error; err != nil {
		return nil, err
	}
	return syncItems, nil
}

// GetUserProgressSyncItems retrieves all sync items for a user
func (db *Database) GetUserProgressSyncItems(userID uint, syncStatus string) ([]*models.UserProgressSyncItem, error) {
	var syncItems []*models.UserProgressSyncItem
	query := db.gormdb.Where("user_id = ?", userID)
	if syncStatus != "" {
		query = query.Where("sync_status = ?", syncStatus)
	}
	if err := query.Find(&syncItems).Error; err != nil {
		return nil, err
	}
	return syncItems, nil
}

// UpdateUserProgressSyncItem updates a progress sync item
func (db *Database) UpdateUserProgressSyncItem(syncItem *models.UserProgressSyncItem) error {
	return db.gormdb.Save(syncItem).Error
}

// DeleteUserProgressSyncItem deletes a progress sync item
func (db *Database) DeleteUserProgressSyncItem(id uint) error {
	return db.gormdb.Delete(&models.UserProgressSyncItem{}, id).Error
}

// UpsertUserProgressSyncItem creates or updates a progress sync item
func (db *Database) UpsertUserProgressSyncItem(userID uint, aniListID int, updates map[string]interface{}) error {
	var existingItem models.UserProgressSyncItem
	err := db.gormdb.Where("user_id = ? AND anilist_id = ? AND sync_status = ?",
		userID, aniListID, "PENDING").First(&existingItem).Error

	if err != nil {
		// Create new item
		syncItem := models.UserProgressSyncItem{
			UserID:      userID,
			AniListID:   aniListID,
			SyncStatus:  "PENDING",
			LastUpdated: time.Now(),
		}

		// Apply updates
		for key, value := range updates {
			switch key {
			case "episode_number":
				if v, ok := value.(int); ok {
					syncItem.EpisodeNumber = v
				}
			case "status":
				if v, ok := value.(string); ok {
					syncItem.Status = v
				}
			case "progress":
				if v, ok := value.(int); ok {
					syncItem.Progress = v
				}
			case "score":
				if v, ok := value.(int); ok {
					syncItem.Score = v
				}
			case "is_completed":
				if v, ok := value.(bool); ok {
					syncItem.IsCompleted = v
				}
			}
		}

		return db.gormdb.Create(&syncItem).Error
	} else {
		// Update existing item
		updates["last_updated"] = time.Now()
		return db.gormdb.Model(&existingItem).Updates(updates).Error
	}
}

// GetProgressSyncStats returns statistics about sync queue
func (db *Database) GetProgressSyncStats() (map[string]int64, error) {
	stats := make(map[string]int64)

	var pendingCount, syncedCount, failedCount int64

	// Count pending items
	if err := db.gormdb.Model(&models.UserProgressSyncItem{}).
		Where("sync_status = ?", "PENDING").
		Count(&pendingCount).Error; err != nil {
		return nil, err
	}
	stats["pending"] = pendingCount

	// Count synced items
	if err := db.gormdb.Model(&models.UserProgressSyncItem{}).
		Where("sync_status = ?", "SYNCED").
		Count(&syncedCount).Error; err != nil {
		return nil, err
	}
	stats["synced"] = syncedCount

	// Count failed items
	if err := db.gormdb.Model(&models.UserProgressSyncItem{}).
		Where("sync_status = ?", "FAILED").
		Count(&failedCount).Error; err != nil {
		return nil, err
	}
	stats["failed"] = failedCount

	return stats, nil
}
