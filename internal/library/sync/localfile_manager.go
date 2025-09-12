package sync

import (
	"crypto/sha256"
	"fmt"
	"seanime/internal/database/db"
	"seanime/internal/database/db_bridge"
	"seanime/internal/database/models"
	"seanime/internal/events"
	"seanime/internal/library/anime"
	"seanime/internal/library/filesystem"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// LocalFileManager handles real-time LocalFile updates and selective broadcasting
type LocalFileManager struct {
	db                *db.Database
	enhancedWS        *events.EnhancedWSEventManager
	logger            *zerolog.Logger
	
	// Cache for performance
	mediaLibraryCache map[int][]uint // MediaID -> []UserID (users who have this media)
	cacheMu           sync.RWMutex
	
	// Batch processing
	batchUpdates      map[int][]*anime.LocalFile // MediaID -> []LocalFile updates
	batchMu           sync.Mutex
	batchTimer        *time.Timer
	batchInterval     time.Duration
}

// LocalFileUpdate represents a change to a LocalFile
type LocalFileUpdate struct {
	Type      string              `json:"type"`      // "added", "removed", "modified"
	LocalFile *anime.LocalFile    `json:"localFile"`
	MediaID   int                 `json:"mediaId"`
	Timestamp time.Time           `json:"timestamp"`
}

// BatchLocalFileUpdate represents multiple LocalFile changes for a single media
type BatchLocalFileUpdate struct {
	MediaID     int                   `json:"mediaId"`
	AddedFiles  []*anime.LocalFile    `json:"addedFiles"`
	RemovedFiles []string             `json:"removedFiles"`  // File paths
	UpdatedFiles []*anime.LocalFile   `json:"updatedFiles"`
	Timestamp   time.Time             `json:"timestamp"`
}

// NewLocalFileManager creates a new LocalFile manager
func NewLocalFileManager(database *db.Database, enhancedWS *events.EnhancedWSEventManager, logger *zerolog.Logger) *LocalFileManager {
	lfm := &LocalFileManager{
		db:                database,
		enhancedWS:        enhancedWS,
		logger:            logger,
		mediaLibraryCache: make(map[int][]uint),
		batchUpdates:      make(map[int][]*anime.LocalFile),
		batchInterval:     2 * time.Second, // Batch updates for 2 seconds
	}
	
	// Initialize media library cache
	lfm.refreshMediaLibraryCache()
	
	// Start batch processing
	lfm.startBatchProcessor()
	
	return lfm
}

// refreshMediaLibraryCache updates the cache of which users have which media in their library
func (lfm *LocalFileManager) refreshMediaLibraryCache() {
	var userLibraryEntries []models.UserLibraryEntry
	err := lfm.db.Gorm().Where("is_active = ?", true).Find(&userLibraryEntries).Error
	if err != nil {
		lfm.logger.Error().Err(err).Msg("localfile-manager: Failed to load user library entries")
		return
	}
	
	lfm.cacheMu.Lock()
	defer lfm.cacheMu.Unlock()
	
	// Clear and rebuild cache
	lfm.mediaLibraryCache = make(map[int][]uint)
	
	for _, entry := range userLibraryEntries {
		if !lfm.containsUser(lfm.mediaLibraryCache[entry.MediaID], entry.UserID) {
			lfm.mediaLibraryCache[entry.MediaID] = append(lfm.mediaLibraryCache[entry.MediaID], entry.UserID)
		}
	}
	
	lfm.logger.Info().
		Int("mediaCount", len(lfm.mediaLibraryCache)).
		Msg("localfile-manager: Refreshed media library cache")
}

// ProcessNewLocalFile handles a newly discovered LocalFile
func (lfm *LocalFileManager) ProcessNewLocalFile(lf *anime.LocalFile) error {
	if lf.MediaId == 0 {
		// File not yet matched to media, skip broadcasting
		return nil
	}
	
	// Add to batch updates
	lfm.addToBatch(lf.MediaId, lf, "added")
	
	// Send immediate notification about new media availability
	// This is the key enhancement for server-wide mapping
	go lfm.NotifyNewMediaAvailability(lf.MediaId, []*anime.LocalFile{lf})
	
	// Log the addition
	lfm.logger.Debug().
		Str("path", lf.Path).
		Int("mediaId", lf.MediaId).
		Msg("localfile-manager: New LocalFile added")
	
	return nil
}

// ProcessRemovedLocalFile handles a removed LocalFile
func (lfm *LocalFileManager) ProcessRemovedLocalFile(filePath string, mediaID int) error {
	if mediaID == 0 {
		return nil
	}
	
	// Get affected users
	users := lfm.getUsersWithMedia(mediaID)
	if len(users) == 0 {
		return nil
	}
	
	// Create removal event
	update := LocalFileUpdate{
		Type:      "removed",
		LocalFile: &anime.LocalFile{Path: filePath, MediaId: mediaID},
		MediaID:   mediaID,
		Timestamp: time.Now(),
	}
	
	// Broadcast immediately for removals (more urgent)
	lfm.enhancedWS.SendEventToUsersWithMedia(mediaID, events.EventLocalFileRemovedForMedia, update)
	
	lfm.logger.Debug().
		Str("path", filePath).
		Int("mediaId", mediaID).
		Int("userCount", len(users)).
		Msg("localfile-manager: LocalFile removed")
	
	return nil
}

// ProcessModifiedLocalFile handles a modified LocalFile  
func (lfm *LocalFileManager) ProcessModifiedLocalFile(lf *anime.LocalFile) error {
	if lf.MediaId == 0 {
		return nil
	}
	
	// Add to batch updates
	lfm.addToBatch(lf.MediaId, lf, "modified")
	
	lfm.logger.Debug().
		Str("path", lf.Path).
		Int("mediaId", lf.MediaId).
		Msg("localfile-manager: LocalFile modified")
	
	return nil
}

// ProcessBulkLocalFiles handles bulk LocalFile updates (e.g., from scan)
func (lfm *LocalFileManager) ProcessBulkLocalFiles(localFiles []*anime.LocalFile) error {
	// Group by media ID
	mediaGroups := make(map[int][]*anime.LocalFile)
	for _, lf := range localFiles {
		if lf.MediaId != 0 {
			mediaGroups[lf.MediaId] = append(mediaGroups[lf.MediaId], lf)
		}
	}
	
	// Process each media group
	for mediaID, files := range mediaGroups {
		users := lfm.getUsersWithMedia(mediaID)
		if len(users) == 0 {
			continue
		}
		
		// Send server-wide availability notification for bulk additions
		go lfm.NotifyNewMediaAvailability(mediaID, files)
		
		// Create batch update
		batchUpdate := BatchLocalFileUpdate{
			MediaID:      mediaID,
			AddedFiles:   files,
			RemovedFiles: []string{},
			UpdatedFiles: []*anime.LocalFile{},
			Timestamp:    time.Now(),
		}
		
		// Broadcast to interested users
		lfm.enhancedWS.SendEventToUsersWithMedia(mediaID, events.EventLocalFilesBatchUpdate, batchUpdate)
		
		lfm.logger.Debug().
			Int("mediaId", mediaID).
			Int("fileCount", len(files)).
			Int("userCount", len(users)).
			Msg("localfile-manager: Bulk LocalFiles processed")
	}
	
	return nil
}

// SyncUserLibrarySubscriptions ensures a user is subscribed to their library media
func (lfm *LocalFileManager) SyncUserLibrarySubscriptions(userID uint, mediaIDs []int) error {
	for _, mediaID := range mediaIDs {
		err := lfm.enhancedWS.SubscribeUserToMedia(userID, mediaID)
		if err != nil {
			lfm.logger.Error().Err(err).
				Uint("userId", userID).
				Int("mediaId", mediaID).
				Msg("localfile-manager: Failed to subscribe user to media")
			continue
		}
		
		// Update cache
		lfm.cacheMu.Lock()
		if !lfm.containsUser(lfm.mediaLibraryCache[mediaID], userID) {
			lfm.mediaLibraryCache[mediaID] = append(lfm.mediaLibraryCache[mediaID], userID)
		}
		lfm.cacheMu.Unlock()
	}
	
	lfm.logger.Debug().
		Uint("userId", userID).
		Int("mediaCount", len(mediaIDs)).
		Msg("localfile-manager: Synced user library subscriptions")
	
	return nil
}

// HandleUserLibraryChanged updates subscriptions when user's library changes
func (lfm *LocalFileManager) HandleUserLibraryChanged(userID uint, addedMediaIDs []int, removedMediaIDs []int) error {
	// Subscribe to new media
	for _, mediaID := range addedMediaIDs {
		err := lfm.enhancedWS.SubscribeUserToMedia(userID, mediaID)
		if err != nil {
			lfm.logger.Error().Err(err).
				Uint("userId", userID).
				Int("mediaId", mediaID).
				Msg("localfile-manager: Failed to subscribe user to new media")
			continue
		}
		
		// Update cache
		lfm.cacheMu.Lock()
		if !lfm.containsUser(lfm.mediaLibraryCache[mediaID], userID) {
			lfm.mediaLibraryCache[mediaID] = append(lfm.mediaLibraryCache[mediaID], userID)
		}
		lfm.cacheMu.Unlock()
		
		// Send existing LocalFiles for this media to the user
		go lfm.sendExistingLocalFilesToUser(userID, mediaID)
	}
	
	// Unsubscribe from removed media
	for _, mediaID := range removedMediaIDs {
		err := lfm.enhancedWS.UnsubscribeUserFromMedia(userID, mediaID)
		if err != nil {
			lfm.logger.Error().Err(err).
				Uint("userId", userID).
				Int("mediaId", mediaID).
				Msg("localfile-manager: Failed to unsubscribe user from media")
			continue
		}
		
		// Update cache
		lfm.cacheMu.Lock()
		users := lfm.mediaLibraryCache[mediaID]
		for i, user := range users {
			if user == userID {
				lfm.mediaLibraryCache[mediaID] = append(users[:i], users[i+1:]...)
				break
			}
		}
		lfm.cacheMu.Unlock()
	}
	
	lfm.logger.Info().
		Uint("userId", userID).
		Int("added", len(addedMediaIDs)).
		Int("removed", len(removedMediaIDs)).
		Msg("localfile-manager: User library changed")
	
	return nil
}

// sendExistingLocalFilesToUser sends all existing LocalFiles for a media to a user
func (lfm *LocalFileManager) sendExistingLocalFilesToUser(userID uint, mediaID int) {
	// Get existing LocalFiles for this media from global database
	localFiles, _, err := db_bridge.GetLocalFiles(lfm.db)
	if err != nil {
		lfm.logger.Error().Err(err).
			Uint("userId", userID).
			Int("mediaId", mediaID).
			Msg("localfile-manager: Failed to get existing local files")
		return
	}
	
	// Filter files for this specific media
	var mediaFiles []*anime.LocalFile
	for _, lf := range localFiles {
		if lf.MediaId == mediaID {
			mediaFiles = append(mediaFiles, lf)
		}
	}
	
	if len(mediaFiles) == 0 {
		lfm.logger.Debug().
			Uint("userId", userID).
			Int("mediaId", mediaID).
			Msg("localfile-manager: No existing files found for media")
		return
	}
	
	// Create media availability event
	availabilityEvent := map[string]interface{}{
		"type":        "media-available",
		"mediaId":     mediaID,
		"userId":      userID,
		"fileCount":   len(mediaFiles),
		"localFiles":  mediaFiles,
		"message":     fmt.Sprintf("Found %d episodes available for playback", len(mediaFiles)),
		"timestamp":   time.Now(),
	}
	
	lfm.enhancedWS.SendEventToUser(userID, events.EventLocalFilesBatchUpdate, availabilityEvent)
	
	lfm.logger.Info().
		Uint("userId", userID).
		Int("mediaId", mediaID).
		Int("fileCount", len(mediaFiles)).
		Msg("localfile-manager: Sent existing media files to user")
}

// addToBatch adds a LocalFile to the batch processing queue
func (lfm *LocalFileManager) addToBatch(mediaID int, lf *anime.LocalFile, updateType string) {
	lfm.batchMu.Lock()
	defer lfm.batchMu.Unlock()
	
	lfm.batchUpdates[mediaID] = append(lfm.batchUpdates[mediaID], lf)
	
	// Reset/start batch timer
	if lfm.batchTimer != nil {
		lfm.batchTimer.Stop()
	}
	
	lfm.batchTimer = time.AfterFunc(lfm.batchInterval, lfm.processBatchUpdates)
}

// startBatchProcessor starts the batch processing system
func (lfm *LocalFileManager) startBatchProcessor() {
	// The actual processing is triggered by the timer in addToBatch
	lfm.logger.Info().
		Dur("interval", lfm.batchInterval).
		Msg("localfile-manager: Batch processor started")
}

// processBatchUpdates processes all batched updates
func (lfm *LocalFileManager) processBatchUpdates() {
	lfm.batchMu.Lock()
	updates := lfm.batchUpdates
	lfm.batchUpdates = make(map[int][]*anime.LocalFile)
	lfm.batchMu.Unlock()
	
	if len(updates) == 0 {
		return
	}
	
	for mediaID, files := range updates {
		users := lfm.getUsersWithMedia(mediaID)
		if len(users) == 0 {
			continue
		}
		
		batchUpdate := BatchLocalFileUpdate{
			MediaID:      mediaID,
			AddedFiles:   files,
			RemovedFiles: []string{},
			UpdatedFiles: []*anime.LocalFile{},
			Timestamp:    time.Now(),
		}
		
		lfm.enhancedWS.SendEventToUsersWithMedia(mediaID, events.EventLocalFilesBatchUpdate, batchUpdate)
		
		lfm.logger.Debug().
			Int("mediaId", mediaID).
			Int("fileCount", len(files)).
			Int("userCount", len(users)).
			Msg("localfile-manager: Batch update processed")
	}
}

// getUsersWithMedia returns users who have a specific media in their library
func (lfm *LocalFileManager) getUsersWithMedia(mediaID int) []uint {
	lfm.cacheMu.RLock()
	defer lfm.cacheMu.RUnlock()
	
	users := lfm.mediaLibraryCache[mediaID]
	if users == nil {
		return []uint{}
	}
	
	// Return copy to prevent external modification
	result := make([]uint, len(users))
	copy(result, users)
	
	return result
}

// containsUser checks if a user is in a slice
func (lfm *LocalFileManager) containsUser(users []uint, userID uint) bool {
	for _, user := range users {
		if user == userID {
			return true
		}
	}
	return false
}

// GenerateFileHash creates a hash for file integrity checking
func (lfm *LocalFileManager) GenerateFileHash(filePath string) string {
	// Use file path for hash
	if !filesystem.FileExists(filePath) {
		return ""
	}
	
	hashInput := fmt.Sprintf("%s", filePath)
	hash := sha256.Sum256([]byte(hashInput))
	return fmt.Sprintf("%x", hash)[:16] // Use first 16 chars
}

// NotifyNewMediaAvailability sends notifications to all users who have a media in their library
// when new episodes become available (server-wide discovery)
func (lfm *LocalFileManager) NotifyNewMediaAvailability(mediaID int, newFiles []*anime.LocalFile) error {
	// Get all users who have this media in their library
	lfm.cacheMu.RLock()
	subscribedUsers := make([]uint, len(lfm.mediaLibraryCache[mediaID]))
	copy(subscribedUsers, lfm.mediaLibraryCache[mediaID])
	lfm.cacheMu.RUnlock()
	
	if len(subscribedUsers) == 0 {
		lfm.logger.Debug().
			Int("mediaId", mediaID).
			Msg("localfile-manager: No users subscribed to media for new file notification")
		return nil
	}
	
	// Create availability notification event
	availabilityEvent := map[string]interface{}{
		"type":        "new-episodes-available",
		"mediaId":     mediaID,
		"fileCount":   len(newFiles),
		"newFiles":    newFiles,
		"message":     fmt.Sprintf("%d new episodes are now available to watch!", len(newFiles)),
		"timestamp":   time.Now(),
	}
	
	// Notify all subscribed users
	for _, userID := range subscribedUsers {
		lfm.enhancedWS.SendEventToUser(userID, events.EventLocalFileAddedForMedia, availabilityEvent)
	}
	
	lfm.logger.Info().
		Int("mediaId", mediaID).
		Int("fileCount", len(newFiles)).
		Int("notifiedUsers", len(subscribedUsers)).
		Msg("localfile-manager: Notified users about new media availability")
	
	return nil
}

// GetMediaAvailabilityForUser returns all available episodes for a media that a user can access
func (lfm *LocalFileManager) GetMediaAvailabilityForUser(userID uint, mediaID int) ([]*anime.LocalFile, error) {
	// Get all LocalFiles from global database  
	localFiles, _, err := db_bridge.GetLocalFiles(lfm.db)
	if err != nil {
		return nil, fmt.Errorf("failed to get local files: %w", err)
	}
	
	// Filter files for this specific media
	var mediaFiles []*anime.LocalFile
	for _, lf := range localFiles {
		if lf.MediaId == mediaID {
			mediaFiles = append(mediaFiles, lf)
		}
	}
	
	return mediaFiles, nil
}