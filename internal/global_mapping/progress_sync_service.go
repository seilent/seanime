package global_mapping

import (
	"context"
	"seanime/internal/api/anilist"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"seanime/internal/platforms/anilist_platform"
	"seanime/internal/platforms/platform"
	"time"

	"github.com/rs/zerolog"
)

type ProgressSyncService struct {
	db       *db.Database
	logger   *zerolog.Logger
	platform platform.Platform
	ticker   *time.Ticker
	stopCh   chan struct{}
}

func NewProgressSyncService(database *db.Database, logger *zerolog.Logger, platform platform.Platform) *ProgressSyncService {
	return &ProgressSyncService{
		db:       database,
		logger:   logger,
		platform: platform,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the periodic sync process
func (pss *ProgressSyncService) Start(interval time.Duration) {
	pss.logger.Info().
		Dur("interval", interval).
		Msg("progress_sync: Starting progress sync service")

	pss.ticker = time.NewTicker(interval)

	go func() {
		defer pss.ticker.Stop()
		for {
			select {
			case <-pss.ticker.C:
				if err := pss.processSyncQueue(); err != nil {
					pss.logger.Error().Err(err).Msg("progress_sync: Failed to process sync queue")
				}
			case <-pss.stopCh:
				pss.logger.Info().Msg("progress_sync: Progress sync service stopped")
				return
			}
		}
	}()
}

// Stop stops the periodic sync process
func (pss *ProgressSyncService) Stop() {
	pss.logger.Info().Msg("progress_sync: Stopping progress sync service")
	close(pss.stopCh)
}

// QueueProgressUpdate adds a progress update to the sync queue
func (pss *ProgressSyncService) QueueProgressUpdate(userID uint, aniListID int, episodeNumber int, status string, progress int, score int, isCompleted bool) error {
	pss.logger.Debug().
		Uint("user_id", userID).
		Int("anilist_id", aniListID).
		Int("episode", episodeNumber).
		Int("progress", progress).
		Msg("progress_sync: Queueing progress update")

	syncItem := models.UserProgressSyncItem{
		UserID:         userID,
		AniListID:      aniListID,
		EpisodeNumber:  episodeNumber,
		Status:         status,
		Score:          score,
		Progress:       progress,
		IsCompleted:    isCompleted,
		LastUpdated:    time.Now(),
		SyncStatus:     "PENDING",
		RetryCount:     0,
	}

	// Check if there's already a pending sync for this user/anime combination
	var existingItem models.UserProgressSyncItem
	err := pss.db.Gorm().Where("user_id = ? AND anilist_id = ? AND sync_status = ?",
		userID, aniListID, "PENDING").First(&existingItem).Error

	if err == nil {
		// Update existing item with latest data
		existingItem.EpisodeNumber = episodeNumber
		existingItem.Status = status
		existingItem.Score = score
		existingItem.Progress = progress
		existingItem.IsCompleted = isCompleted
		existingItem.LastUpdated = time.Now()
		return pss.db.Gorm().Save(&existingItem).Error
	} else {
		// Create new sync item
		return pss.db.Gorm().Create(&syncItem).Error
	}
}

// processSyncQueue processes all pending sync items
func (pss *ProgressSyncService) processSyncQueue() error {
	pss.logger.Debug().Msg("progress_sync: Processing sync queue")

	// Get all pending items
	var pendingItems []models.UserProgressSyncItem
	if err := pss.db.Gorm().Where("sync_status = ?", "PENDING").Find(&pendingItems).Error; err != nil {
		return err
	}

	if len(pendingItems) == 0 {
		pss.logger.Debug().Msg("progress_sync: No pending items to sync")
		return nil
	}

	pss.logger.Info().Int("count", len(pendingItems)).Msg("progress_sync: Processing pending sync items")

	// Group items by user
	userItems := make(map[uint][]models.UserProgressSyncItem)
	for _, item := range pendingItems {
		userItems[item.UserID] = append(userItems[item.UserID], item)
	}

	// Process each user's items
	for userID, items := range userItems {
		if err := pss.syncUserProgress(userID, items); err != nil {
			pss.logger.Error().Err(err).
				Uint("user_id", userID).
				Msg("progress_sync: Failed to sync user progress")

			// Mark failed items for retry
			pss.markItemsForRetry(items)
		}
	}

	return nil
}

// syncUserProgress syncs progress for a specific user
func (pss *ProgressSyncService) syncUserProgress(userID uint, items []models.UserProgressSyncItem) error {
	pss.logger.Debug().
		Uint("user_id", userID).
		Int("items", len(items)).
		Msg("progress_sync: Syncing user progress")

	// Get user's account info
	account, err := pss.db.GetAccountForUser(userID)
	if err != nil {
		return err
	}

	if account.Token == "" {
		pss.logger.Debug().Uint("user_id", userID).Msg("progress_sync: User has no AniList token, skipping")
		return nil
	}

	// Create user-specific AniList platform (reusing existing pattern)
	client := anilist.NewAnilistClient(account.Token)
	userPlatform := anilist_platform.NewAnilistPlatform(client, pss.logger)
	userPlatform.SetUsername(account.Username)

	// Process each sync item
	for _, item := range items {
		if err := pss.syncSingleItem(userPlatform, item); err != nil {
			pss.logger.Error().Err(err).
				Uint("user_id", userID).
				Int("anilist_id", item.AniListID).
				Msg("progress_sync: Failed to sync single item")

			// Mark this specific item for retry
			pss.markSingleItemForRetry(item)
			continue
		}

		// Mark as synced
		if err := pss.markItemAsSynced(item); err != nil {
			pss.logger.Error().Err(err).Msg("progress_sync: Failed to mark item as synced")
		}
	}

	return nil
}

// syncSingleItem syncs a single progress item to AniList
func (pss *ProgressSyncService) syncSingleItem(userPlatform platform.Platform, item models.UserProgressSyncItem) error {
	ctx := context.Background()

	// Prepare the update data using existing AniList types
	var status *anilist.MediaListStatus
	if item.Status != "" {
		statusValue := anilist.MediaListStatus(item.Status)
		status = &statusValue
	}

	var score *int
	if item.Score > 0 {
		score = &item.Score
	}

	// Use the existing UpdateEntry method from platform interface
	err := userPlatform.UpdateEntry(ctx, item.AniListID, status, score, &item.Progress, nil, nil)

	if err != nil {
		return err
	}

	pss.logger.Debug().
		Int("anilist_id", item.AniListID).
		Int("progress", item.Progress).
		Msg("progress_sync: Successfully synced item to AniList")

	return nil
}

// markItemAsSynced marks a sync item as successfully synced
func (pss *ProgressSyncService) markItemAsSynced(item models.UserProgressSyncItem) error {
	return pss.db.Gorm().Model(&item).Updates(map[string]interface{}{
		"sync_status": "SYNCED",
		"last_updated": time.Now(),
	}).Error
}

// markSingleItemForRetry marks a single item for retry with backoff
func (pss *ProgressSyncService) markSingleItemForRetry(item models.UserProgressSyncItem) {
	retryCount := item.RetryCount + 1

	// Exponential backoff: if retry count is too high, mark as failed
	if retryCount > 5 {
		pss.db.Gorm().Model(&item).Updates(map[string]interface{}{
			"sync_status": "FAILED",
			"retry_count": retryCount,
			"last_updated": time.Now(),
		})
		pss.logger.Error().
			Uint("user_id", item.UserID).
			Int("anilist_id", item.AniListID).
			Int("retries", retryCount).
			Msg("progress_sync: Item marked as failed after max retries")
		return
	}

	pss.db.Gorm().Model(&item).Updates(map[string]interface{}{
		"retry_count": retryCount,
		"last_updated": time.Now(),
	})

	pss.logger.Debug().
		Uint("user_id", item.UserID).
		Int("anilist_id", item.AniListID).
		Int("retries", retryCount).
		Msg("progress_sync: Item marked for retry")
}

// markItemsForRetry marks multiple items for retry
func (pss *ProgressSyncService) markItemsForRetry(items []models.UserProgressSyncItem) {
	for _, item := range items {
		pss.markSingleItemForRetry(item)
	}
}

// GetPendingItemsCount returns the count of pending sync items
func (pss *ProgressSyncService) GetPendingItemsCount() (int64, error) {
	var count int64
	if err := pss.db.Gorm().Model(&models.UserProgressSyncItem{}).
		Where("sync_status = ?", "PENDING").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetFailedItemsCount returns the count of failed sync items
func (pss *ProgressSyncService) GetFailedItemsCount() (int64, error) {
	var count int64
	if err := pss.db.Gorm().Model(&models.UserProgressSyncItem{}).
		Where("sync_status = ?", "FAILED").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// RetryFailedItems resets failed items to pending status for retry
func (pss *ProgressSyncService) RetryFailedItems() error {
	pss.logger.Info().Msg("progress_sync: Retrying failed items")

	return pss.db.Gorm().Model(&models.UserProgressSyncItem{}).
		Where("sync_status = ?", "FAILED").
		Updates(map[string]interface{}{
			"sync_status": "PENDING",
			"retry_count": 0,
			"last_updated": time.Now(),
		}).Error
}