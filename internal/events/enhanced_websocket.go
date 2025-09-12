package events

import (
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// EnhancedWSEventManager extends the basic WebSocket manager with user-specific and media-specific broadcasting
type EnhancedWSEventManager struct {
	*WSEventManager // Embed the original manager
	db              *db.Database
	
	// User and media subscription tracking
	userSubscriptions map[uint]map[int]bool // UserID -> MediaID -> subscribed
	mediaSubscribers  map[int][]uint        // MediaID -> []UserID
	userSessions      map[uint][]string     // UserID -> []SessionID
	sessionUsers      map[string]uint       // SessionID -> UserID
	
	mu                sync.RWMutex
}

// LocalFileUpdateEvent represents a LocalFile update event for selective broadcasting
type LocalFileUpdateEvent struct {
	Type       string      `json:"type"`      // "added", "removed", "modified"
	MediaID    int         `json:"mediaId"`
	LocalFiles interface{} `json:"localFiles"` // Could be single LocalFile or array
	Timestamp  time.Time   `json:"timestamp"`
}

// UserProgressUpdateEvent represents a progress update event
type UserProgressUpdateEvent struct {
	UserID            uint    `json:"userId"`
	MediaID           int     `json:"mediaId"`
	EpisodeNumber     int     `json:"episodeNumber"`
	WatchTimeSeconds  int     `json:"watchTimeSeconds"`
	DurationSeconds   int     `json:"durationSeconds"`
	CompletionPercent float64 `json:"completionPercent"`
	IsCompleted       bool    `json:"isCompleted"`
	SessionID         string  `json:"sessionId"`
	Timestamp         time.Time `json:"timestamp"`
}

// NewEnhancedWSEventManager creates a new enhanced WebSocket event manager
func NewEnhancedWSEventManager(logger *zerolog.Logger, database *db.Database) *EnhancedWSEventManager {
	baseManager := NewWSEventManager(logger)
	
	enhanced := &EnhancedWSEventManager{
		WSEventManager:    baseManager,
		db:               database,
		userSubscriptions: make(map[uint]map[int]bool),
		mediaSubscribers:  make(map[int][]uint),
		userSessions:      make(map[uint][]string),
		sessionUsers:      make(map[string]uint),
	}
	
	// Initialize user library subscriptions from database
	enhanced.initializeUserSubscriptions()
	
	return enhanced
}

// initializeUserSubscriptions loads existing user library entries and sets up subscriptions
func (e *EnhancedWSEventManager) initializeUserSubscriptions() {
	var userLibraryEntries []models.UserLibraryEntry
	err := e.db.Gorm().Where("is_active = ?", true).Find(&userLibraryEntries).Error
	if err != nil {
		e.Logger.Error().Err(err).Msg("enhanced-ws: Failed to load user library entries")
		return
	}
	
	e.mu.Lock()
	defer e.mu.Unlock()
	
	for _, entry := range userLibraryEntries {
		// Initialize user subscription map if needed
		if e.userSubscriptions[entry.UserID] == nil {
			e.userSubscriptions[entry.UserID] = make(map[int]bool)
		}
		
		// Add subscription
		e.userSubscriptions[entry.UserID][entry.MediaID] = true
		
		// Add to media subscribers
		if !e.containsUser(e.mediaSubscribers[entry.MediaID], entry.UserID) {
			e.mediaSubscribers[entry.MediaID] = append(e.mediaSubscribers[entry.MediaID], entry.UserID)
		}
	}
	
	e.Logger.Info().
		Int("users", len(e.userSubscriptions)).
		Int("media", len(e.mediaSubscribers)).
		Msg("enhanced-ws: Initialized user subscriptions")
}

// RegisterUserSession associates a WebSocket session with a user
func (e *EnhancedWSEventManager) RegisterUserSession(userID uint, sessionID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Add session to user
	if e.userSessions[userID] == nil {
		e.userSessions[userID] = make([]string, 0)
	}
	e.userSessions[userID] = append(e.userSessions[userID], sessionID)
	
	// Map session to user
	e.sessionUsers[sessionID] = userID
	
	e.Logger.Debug().
		Uint("userId", userID).
		Str("sessionId", sessionID).
		Msg("enhanced-ws: Registered user session")
}

// UnregisterUserSession removes a WebSocket session
func (e *EnhancedWSEventManager) UnregisterUserSession(sessionID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	userID, exists := e.sessionUsers[sessionID]
	if !exists {
		return
	}
	
	// Remove session from user
	sessions := e.userSessions[userID]
	for i, session := range sessions {
		if session == sessionID {
			e.userSessions[userID] = append(sessions[:i], sessions[i+1:]...)
			break
		}
	}
	
	// Remove session mapping
	delete(e.sessionUsers, sessionID)
	
	e.Logger.Debug().
		Uint("userId", userID).
		Str("sessionId", sessionID).
		Msg("enhanced-ws: Unregistered user session")
}

// SubscribeUserToMedia adds a user's subscription to a specific media
func (e *EnhancedWSEventManager) SubscribeUserToMedia(userID uint, mediaID int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Initialize user subscription map if needed
	if e.userSubscriptions[userID] == nil {
		e.userSubscriptions[userID] = make(map[int]bool)
	}
	
	// Add subscription
	e.userSubscriptions[userID][mediaID] = true
	
	// Add to media subscribers
	if !e.containsUser(e.mediaSubscribers[mediaID], userID) {
		e.mediaSubscribers[mediaID] = append(e.mediaSubscribers[mediaID], userID)
	}
	
	// Create or update database entry using upsert
	subscription := &models.UserMediaSubscription{
		UserID:   userID,
		MediaID:  mediaID,
		IsActive: true,
	}
	
	// Use ON CONFLICT/upsert to handle existing records
	err := e.db.Gorm().Where("user_id = ? AND media_id = ?", userID, mediaID).
		Assign(models.UserMediaSubscription{IsActive: true}).
		FirstOrCreate(subscription).Error
		
	if err != nil {
		e.Logger.Error().Err(err).
			Uint("userId", userID).
			Int("mediaId", mediaID).
			Msg("enhanced-ws: Failed to create/update media subscription")
		return err
	}
	
	e.Logger.Debug().
		Uint("userId", userID).
		Int("mediaId", mediaID).
		Msg("enhanced-ws: User subscribed to media")
	
	return nil
}

// UnsubscribeUserFromMedia removes a user's subscription to a specific media
func (e *EnhancedWSEventManager) UnsubscribeUserFromMedia(userID uint, mediaID int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Remove from user subscriptions
	if e.userSubscriptions[userID] != nil {
		delete(e.userSubscriptions[userID], mediaID)
	}
	
	// Remove from media subscribers
	subscribers := e.mediaSubscribers[mediaID]
	for i, subscriber := range subscribers {
		if subscriber == userID {
			e.mediaSubscribers[mediaID] = append(subscribers[:i], subscribers[i+1:]...)
			break
		}
	}
	
	// Update database entry
	err := e.db.Gorm().Model(&models.UserMediaSubscription{}).
		Where("user_id = ? AND media_id = ?", userID, mediaID).
		Update("is_active", false).Error
	
	if err != nil {
		e.Logger.Error().Err(err).
			Uint("userId", userID).
			Int("mediaId", mediaID).
			Msg("enhanced-ws: Failed to update media subscription")
		return err
	}
	
	e.Logger.Debug().
		Uint("userId", userID).
		Int("mediaId", mediaID).
		Msg("enhanced-ws: User unsubscribed from media")
	
	return nil
}

// SendEventToUser sends an event to all sessions of a specific user
func (e *EnhancedWSEventManager) SendEventToUser(userID uint, eventType string, payload interface{}) {
	e.mu.RLock()
	sessions := e.userSessions[userID]
	e.mu.RUnlock()
	
	if len(sessions) == 0 {
		return
	}
	
	for _, sessionID := range sessions {
		e.SendEventTo(sessionID, eventType, payload, true) // noLog = true to avoid spam
	}
}

// SendEventToUsersWithMedia sends an event to all users who have a specific media in their library
func (e *EnhancedWSEventManager) SendEventToUsersWithMedia(mediaID int, eventType string, payload interface{}) {
	e.mu.RLock()
	subscribedUsers := e.mediaSubscribers[mediaID]
	e.mu.RUnlock()
	
	if len(subscribedUsers) == 0 {
		e.Logger.Trace().
			Int("mediaId", mediaID).
			Str("eventType", eventType).
			Msg("enhanced-ws: No users subscribed to media")
		return
	}
	
	e.Logger.Debug().
		Int("mediaId", mediaID).
		Str("eventType", eventType).
		Int("userCount", len(subscribedUsers)).
		Msg("enhanced-ws: Broadcasting to users with media")
	
	for _, userID := range subscribedUsers {
		e.SendEventToUser(userID, eventType, payload)
	}
}

// BroadcastLocalFileUpdate sends LocalFile updates to interested users only
func (e *EnhancedWSEventManager) BroadcastLocalFileUpdate(mediaID int, updateType string, localFile interface{}) {
	event := LocalFileUpdateEvent{
		Type:       updateType,
		MediaID:    mediaID,
		LocalFiles: localFile,
		Timestamp:  time.Now(),
	}
	
	var eventType string
	switch updateType {
	case "added":
		eventType = EventLocalFileAddedForMedia
	case "removed":
		eventType = EventLocalFileRemovedForMedia
	case "modified":
		eventType = EventLocalFileUpdatedForMedia
	default:
		eventType = EventLocalFilesBatchUpdate
	}
	
	e.SendEventToUsersWithMedia(mediaID, eventType, event)
}

// BroadcastProgressUpdate sends progress updates to a specific user
func (e *EnhancedWSEventManager) BroadcastProgressUpdate(userID uint, progress *models.UserEpisodeProgress) {
	event := UserProgressUpdateEvent{
		UserID:            userID,
		MediaID:           progress.MediaID,
		EpisodeNumber:     progress.EpisodeNumber,
		WatchTimeSeconds:  progress.WatchTimeSeconds,
		DurationSeconds:   progress.DurationSeconds,
		CompletionPercent: progress.CompletionPercent,
		IsCompleted:       progress.IsCompleted,
		Timestamp:         time.Now(),
	}
	
	e.SendEventToUser(userID, EventProgressUpdated, event)
}

// GetUserSubscriptions returns the media subscriptions for a user
func (e *EnhancedWSEventManager) GetUserSubscriptions(userID uint) map[int]bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if e.userSubscriptions[userID] == nil {
		return make(map[int]bool)
	}
	
	// Return a copy to prevent external modification
	result := make(map[int]bool)
	for mediaID, subscribed := range e.userSubscriptions[userID] {
		result[mediaID] = subscribed
	}
	
	return result
}

// GetMediaSubscribers returns the users subscribed to a specific media
func (e *EnhancedWSEventManager) GetMediaSubscribers(mediaID int) []uint {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	subscribers := e.mediaSubscribers[mediaID]
	if subscribers == nil {
		return make([]uint, 0)
	}
	
	// Return a copy to prevent external modification
	result := make([]uint, len(subscribers))
	copy(result, subscribers)
	
	return result
}

// Helper function to check if a user is in a slice
func (e *EnhancedWSEventManager) containsUser(users []uint, userID uint) bool {
	for _, user := range users {
		if user == userID {
			return true
		}
	}
	return false
}