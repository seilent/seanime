package global_mapping

import (
	"seanime/internal/api/anilist"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"time"

	"github.com/rs/zerolog"
)

type UserSubscriptionService struct {
	db     *db.Database
	logger *zerolog.Logger
	gms    *GlobalMappingService
}

func NewUserSubscriptionService(database *db.Database, logger *zerolog.Logger, gms *GlobalMappingService) *UserSubscriptionService {
	return &UserSubscriptionService{
		db:     database,
		logger: logger,
		gms:    gms,
	}
}

// SyncUserSubscriptions synchronizes user subscriptions based on their AniList collection
func (uss *UserSubscriptionService) SyncUserSubscriptions(userID uint, collection *anilist.AnimeCollection) error {
	uss.logger.Debug().Uint("user_id", userID).Msg("subscription: Syncing user subscriptions")

	if collection == nil || collection.MediaListCollection == nil {
		uss.logger.Debug().Uint("user_id", userID).Msg("subscription: No collection provided")
		return nil
	}

	// Get current user subscriptions
	currentSubs, err := uss.getUserSubscriptions(userID)
	if err != nil {
		return err
	}

	// Create map of current subscriptions for quick lookup
	currentSubsMap := make(map[int]bool)
	for _, sub := range currentSubs {
		currentSubsMap[sub.AniListID] = true
	}

	// Process AniList collection
	newSubscriptions := make(map[int]bool)
	for _, list := range collection.MediaListCollection.Lists {
		if list.Entries == nil {
			continue
		}

		for _, entry := range list.Entries {
			if entry.Media == nil {
				continue
			}

			aniListID := entry.Media.ID
			newSubscriptions[aniListID] = true

			// Check if this anime has global file mappings
			files := uss.gms.GetFilesForAniList(aniListID)
			if len(files) > 0 {
				// User should be subscribed to this anime
				if !currentSubsMap[aniListID] {
					// Create new subscription
					if err := uss.createSubscription(userID, aniListID); err != nil {
						uss.logger.Error().Err(err).
							Uint("user_id", userID).
							Int("anilist_id", aniListID).
							Msg("subscription: Failed to create subscription")
					}
				}
			}
		}
	}

	// Remove subscriptions for anime no longer in user's collection
	for _, sub := range currentSubs {
		if !newSubscriptions[sub.AniListID] {
			if err := uss.removeSubscription(userID, sub.AniListID); err != nil {
				uss.logger.Error().Err(err).
					Uint("user_id", userID).
					Int("anilist_id", sub.AniListID).
					Msg("subscription: Failed to remove subscription")
			}
		}
	}

	// Update cache
	uss.updateSubscriptionCache(userID, newSubscriptions)

	uss.logger.Info().
		Uint("user_id", userID).
		Int("subscriptions", len(newSubscriptions)).
		Msg("subscription: User subscriptions synced")

	return nil
}

// SubscribeUserToAnime manually subscribes a user to an anime
func (uss *UserSubscriptionService) SubscribeUserToAnime(userID uint, aniListID int) error {
	uss.logger.Info().
		Uint("user_id", userID).
		Int("anilist_id", aniListID).
		Msg("subscription: Manual user subscription")

	if err := uss.createSubscription(userID, aniListID); err != nil {
		return err
	}

	// Update cache
	uss.gms.cache.Lock()
	if _, exists := uss.gms.cache.UserSubscriptions[userID]; !exists {
		uss.gms.cache.UserSubscriptions[userID] = make(map[int]bool)
	}
	uss.gms.cache.UserSubscriptions[userID][aniListID] = true
	uss.gms.cache.Unlock()

	return nil
}

// UnsubscribeUserFromAnime manually unsubscribes a user from an anime
func (uss *UserSubscriptionService) UnsubscribeUserFromAnime(userID uint, aniListID int) error {
	uss.logger.Info().
		Uint("user_id", userID).
		Int("anilist_id", aniListID).
		Msg("subscription: Manual user unsubscription")

	if err := uss.removeSubscription(userID, aniListID); err != nil {
		return err
	}

	// Update cache
	uss.gms.cache.Lock()
	if userSubs, exists := uss.gms.cache.UserSubscriptions[userID]; exists {
		delete(userSubs, aniListID)
	}
	uss.gms.cache.Unlock()

	return nil
}

// GetUserSubscriptions returns all subscriptions for a user
func (uss *UserSubscriptionService) GetUserSubscriptions(userID uint) ([]*models.UserLibrarySubscription, error) {
	return uss.getUserSubscriptions(userID)
}

// IsUserSubscribed checks if a user is subscribed to an anime
func (uss *UserSubscriptionService) IsUserSubscribed(userID uint, aniListID int) bool {
	uss.gms.cache.RLock()
	defer uss.gms.cache.RUnlock()

	if userSubs, exists := uss.gms.cache.UserSubscriptions[userID]; exists {
		return userSubs[aniListID]
	}
	return false
}

// GetSubscribedUsers returns all users subscribed to an anime
func (uss *UserSubscriptionService) GetSubscribedUsers(aniListID int) []uint {
	uss.gms.cache.RLock()
	defer uss.gms.cache.RUnlock()

	subscribedUsers := make([]uint, 0)
	for userID, subscriptions := range uss.gms.cache.UserSubscriptions {
		if subscriptions[aniListID] {
			subscribedUsers = append(subscribedUsers, userID)
		}
	}

	return subscribedUsers
}

// createSubscription creates a new subscription in the database
func (uss *UserSubscriptionService) createSubscription(userID uint, aniListID int) error {
	subscription := models.UserLibrarySubscription{
		UserID:    userID,
		AniListID: aniListID,
		AddedAt:   time.Now(),
	}

	return uss.db.Gorm().Save(&subscription).Error
}

// removeSubscription removes a subscription from the database
func (uss *UserSubscriptionService) removeSubscription(userID uint, aniListID int) error {
	return uss.db.Gorm().
		Where("user_id = ? AND anilist_id = ?", userID, aniListID).
		Delete(&models.UserLibrarySubscription{}).Error
}

// getUserSubscriptions retrieves all subscriptions for a user from database
func (uss *UserSubscriptionService) getUserSubscriptions(userID uint) ([]*models.UserLibrarySubscription, error) {
	var subscriptions []*models.UserLibrarySubscription
	if err := uss.db.Gorm().Where("user_id = ?", userID).Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

// updateSubscriptionCache updates the subscription cache for a user
func (uss *UserSubscriptionService) updateSubscriptionCache(userID uint, subscriptions map[int]bool) {
	uss.gms.cache.Lock()
	uss.gms.cache.UserSubscriptions[userID] = subscriptions
	uss.gms.cache.Unlock()
}