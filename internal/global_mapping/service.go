package global_mapping

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"seanime/internal/events"
	"seanime/internal/library/anime"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type GlobalMappingService struct {
	db             *db.Database
	logger         *zerolog.Logger
	wsEventManager events.WSEventManagerInterface
	cache          *GlobalMappingCache
	mu             sync.RWMutex
}

type GlobalMappingCache struct {
	sync.RWMutex
	AniListToFiles    map[int][]string          // AniList ID → file paths
	FileToAniList     map[string]int            // file path → AniList ID
	UserSubscriptions map[uint]map[int]bool     // user ID → subscribed anime IDs
}

func NewGlobalMappingService(database *db.Database, logger *zerolog.Logger, wsEventManager events.WSEventManagerInterface) *GlobalMappingService {
	gms := &GlobalMappingService{
		db:             database,
		logger:         logger,
		wsEventManager: wsEventManager,
		cache: &GlobalMappingCache{
			AniListToFiles:    make(map[int][]string),
			FileToAniList:     make(map[string]int),
			UserSubscriptions: make(map[uint]map[int]bool),
		},
	}

	// Load existing mappings into cache
	gms.loadCacheFromDB()

	return gms
}

// loadCacheFromDB loads all existing mappings into memory cache
func (gms *GlobalMappingService) loadCacheFromDB() {
	gms.logger.Debug().Msg("global_mapping: Loading cache from database")

	// Load global mappings
	var mappings []models.GlobalAnimeFileMapping
	if err := gms.db.Gorm().Find(&mappings).Error; err != nil {
		gms.logger.Error().Err(err).Msg("global_mapping: Failed to load mappings from database")
		return
	}

	gms.cache.Lock()
	for _, mapping := range mappings {
		// AniList ID → files
		if _, exists := gms.cache.AniListToFiles[mapping.AniListID]; !exists {
			gms.cache.AniListToFiles[mapping.AniListID] = make([]string, 0)
		}
		gms.cache.AniListToFiles[mapping.AniListID] = append(gms.cache.AniListToFiles[mapping.AniListID], mapping.LocalFilePath)

		// File → AniList ID
		gms.cache.FileToAniList[mapping.LocalFilePath] = mapping.AniListID
	}
	gms.cache.Unlock()

	// Load user subscriptions
	var subscriptions []models.UserLibrarySubscription
	if err := gms.db.Gorm().Find(&subscriptions).Error; err != nil {
		gms.logger.Error().Err(err).Msg("global_mapping: Failed to load subscriptions from database")
		return
	}

	gms.cache.Lock()
	for _, sub := range subscriptions {
		if _, exists := gms.cache.UserSubscriptions[sub.UserID]; !exists {
			gms.cache.UserSubscriptions[sub.UserID] = make(map[int]bool)
		}
		gms.cache.UserSubscriptions[sub.UserID][sub.AniListID] = true
	}
	gms.cache.Unlock()

	gms.logger.Info().
		Int("mappings", len(mappings)).
		Int("subscriptions", len(subscriptions)).
		Msg("global_mapping: Cache loaded from database")
}

// ProcessNewFile attempts to match a new file to an AniList entry
func (gms *GlobalMappingService) ProcessNewFile(filePath string) error {
	gms.logger.Debug().Str("path", filePath).Msg("global_mapping: Processing new file")

	// Check if file already exists in mapping
	if _, exists := gms.getFileMapping(filePath); exists {
		gms.logger.Debug().Str("path", filePath).Msg("global_mapping: File already mapped, skipping")
		return nil
	}

	// Extract file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Parse anime info from filename
	localFile := &anime.LocalFile{
		Path: filePath,
		Name: filepath.Base(filePath),
	}

	// Extract metadata from filename
	metadata := localFile.GetMetadata()
	if metadata == nil {
		// Could not extract metadata, add to unmapped files
		return gms.addToUnmappedFiles(filePath, fileInfo.Size())
	}

	// TODO: Implement actual AniList matching logic here
	// For now, we'll add it to unmapped files for manual processing
	return gms.addToUnmappedFiles(filePath, fileInfo.Size())
}

// MapFileToAniList manually maps a file to an AniList ID
func (gms *GlobalMappingService) MapFileToAniList(filePath string, aniListID int, userID uint, title string, year int, episodeNumber int) error {
	gms.logger.Info().
		Str("path", filePath).
		Int("anilist_id", aniListID).
		Uint("user_id", userID).
		Msg("global_mapping: Mapping file to AniList ID")

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Create or update mapping in database
	mapping := models.GlobalAnimeFileMapping{
		AniListID:      aniListID,
		LocalFilePath:  filePath,
		Title:         title,
		Year:          year,
		EpisodeNumber: episodeNumber,
		FileSize:      fileInfo.Size(),
		LastScanned:   time.Now(),
	}

	if err := gms.db.Gorm().Save(&mapping).Error; err != nil {
		return fmt.Errorf("failed to save mapping: %w", err)
	}

	// Update cache
	gms.cache.Lock()
	if _, exists := gms.cache.AniListToFiles[aniListID]; !exists {
		gms.cache.AniListToFiles[aniListID] = make([]string, 0)
	}
	// Remove from old mapping if exists
	if oldAniListID, exists := gms.cache.FileToAniList[filePath]; exists && oldAniListID != aniListID {
		gms.removeFileFromCache(filePath, oldAniListID)
	}
	// Add to new mapping
	gms.cache.AniListToFiles[aniListID] = append(gms.cache.AniListToFiles[aniListID], filePath)
	gms.cache.FileToAniList[filePath] = aniListID
	gms.cache.Unlock()

	// Remove from unmapped files if it was there
	if err := gms.db.Gorm().Model(&models.UnmappedFile{}).
		Where("local_file_path = ?", filePath).
		Update("status", "MAPPED").Error; err != nil {
		gms.logger.Error().Err(err).Msg("global_mapping: Failed to update unmapped file status")
	}

	// Notify subscribed users
	gms.notifySubscribedUsers(aniListID, "file_mapped", map[string]interface{}{
		"anilist_id":      aniListID,
		"file_path":       filePath,
		"episode_number":  episodeNumber,
		"title":           title,
		"mapped_by_user":  userID,
	})

	return nil
}

// IgnoreFile marks a file as ignored
func (gms *GlobalMappingService) IgnoreFile(filePath string, userID uint) error {
	gms.logger.Info().
		Str("path", filePath).
		Uint("user_id", userID).
		Msg("global_mapping: Ignoring file")

	now := time.Now()
	if err := gms.db.Gorm().Model(&models.UnmappedFile{}).
		Where("local_file_path = ?", filePath).
		Updates(map[string]interface{}{
			"status":            "IGNORED",
			"ignored_by_user_id": userID,
			"ignored_at":        &now,
		}).Error; err != nil {
		return fmt.Errorf("failed to ignore file: %w", err)
	}

	return nil
}

// GetUnmappedFiles returns all unmapped files
func (gms *GlobalMappingService) GetUnmappedFiles() ([]*models.UnmappedFile, error) {
	var unmappedFiles []*models.UnmappedFile
	if err := gms.db.Gorm().Where("status = ?", "UNMAPPED").Find(&unmappedFiles).Error; err != nil {
		return nil, err
	}
	return unmappedFiles, nil
}

// GetIgnoredFiles returns all ignored files
func (gms *GlobalMappingService) GetIgnoredFiles() ([]*models.UnmappedFile, error) {
	var ignoredFiles []*models.UnmappedFile
	if err := gms.db.Gorm().Where("status = ?", "IGNORED").Find(&ignoredFiles).Error; err != nil {
		return nil, err
	}
	return ignoredFiles, nil
}

// GetGlobalMappings returns all global mappings
func (gms *GlobalMappingService) GetGlobalMappings() ([]*models.GlobalAnimeFileMapping, error) {
	var mappings []*models.GlobalAnimeFileMapping
	if err := gms.db.Gorm().Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

// GetFilesForAniList returns all files mapped to a specific AniList ID
func (gms *GlobalMappingService) GetFilesForAniList(aniListID int) []string {
	gms.cache.RLock()
	defer gms.cache.RUnlock()

	if files, exists := gms.cache.AniListToFiles[aniListID]; exists {
		// Return copy to avoid race conditions
		result := make([]string, len(files))
		copy(result, files)
		return result
	}
	return []string{}
}

// GetAniListIDForFile returns the AniList ID for a given file path
func (gms *GlobalMappingService) GetAniListIDForFile(filePath string) (int, bool) {
	return gms.getFileMapping(filePath)
}

// addToUnmappedFiles adds a file to the unmapped files table
func (gms *GlobalMappingService) addToUnmappedFiles(filePath string, fileSize int64) error {
	// Extract title from filename for display
	detectedTitle := gms.extractTitleFromPath(filePath)

	unmappedFile := models.UnmappedFile{
		LocalFilePath: filePath,
		DetectedTitle: detectedTitle,
		FileSize:      fileSize,
		LastDetected:  time.Now(),
		Status:        "UNMAPPED",
	}

	if err := gms.db.Gorm().Save(&unmappedFile).Error; err != nil {
		return fmt.Errorf("failed to save unmapped file: %w", err)
	}

	gms.logger.Debug().Str("path", filePath).Str("title", detectedTitle).Msg("global_mapping: Added to unmapped files")
	return nil
}

// extractTitleFromPath extracts a readable title from the file path
func (gms *GlobalMappingService) extractTitleFromPath(filePath string) string {
	fileName := filepath.Base(filePath)

	// Remove extension
	ext := filepath.Ext(fileName)
	titlePart := strings.TrimSuffix(fileName, ext)

	// Basic cleanup - replace common separators with spaces
	titlePart = strings.ReplaceAll(titlePart, "_", " ")
	titlePart = strings.ReplaceAll(titlePart, ".", " ")
	titlePart = strings.ReplaceAll(titlePart, "-", " ")

	// Remove multiple spaces
	spaceRegex := regexp.MustCompile(`\s+`)
	titlePart = spaceRegex.ReplaceAllString(titlePart, " ")

	return strings.TrimSpace(titlePart)
}

// getFileMapping returns the AniList ID for a file path from cache
func (gms *GlobalMappingService) getFileMapping(filePath string) (int, bool) {
	gms.cache.RLock()
	defer gms.cache.RUnlock()

	id, exists := gms.cache.FileToAniList[filePath]
	return id, exists
}

// removeFileFromCache removes a file from the cache
func (gms *GlobalMappingService) removeFileFromCache(filePath string, aniListID int) {
	// Remove from AniListToFiles
	if files, exists := gms.cache.AniListToFiles[aniListID]; exists {
		for i, file := range files {
			if file == filePath {
				gms.cache.AniListToFiles[aniListID] = append(files[:i], files[i+1:]...)
				break
			}
		}
		// Remove the slice if it's empty
		if len(gms.cache.AniListToFiles[aniListID]) == 0 {
			delete(gms.cache.AniListToFiles, aniListID)
		}
	}
	// Remove from FileToAniList
	delete(gms.cache.FileToAniList, filePath)
}

// notifySubscribedUsers sends SSE notifications to users subscribed to an anime
func (gms *GlobalMappingService) notifySubscribedUsers(aniListID int, eventType string, data map[string]interface{}) {
	gms.cache.RLock()
	subscribedUsers := make([]uint, 0)
	for userID, subscriptions := range gms.cache.UserSubscriptions {
		if subscriptions[aniListID] {
			subscribedUsers = append(subscribedUsers, userID)
		}
	}
	gms.cache.RUnlock()

	if len(subscribedUsers) == 0 {
		gms.logger.Debug().Int("anilist_id", aniListID).Msg("global_mapping: No users subscribed to this anime")
		return
	}

	// Send SSE event to subscribed users
	for range subscribedUsers {
		// For now, send to all users - we'll need to implement user-specific targeting later
		gms.wsEventManager.SendEvent(eventType, data)
	}

	gms.logger.Debug().
		Int("anilist_id", aniListID).
		Int("users_notified", len(subscribedUsers)).
		Str("event", eventType).
		Msg("global_mapping: Notified subscribed users")
}

// RemoveFromCache removes a file mapping from the cache
func (gms *GlobalMappingService) RemoveFromCache(filePath string, aniListID int) {
	gms.cache.Lock()
	defer gms.cache.Unlock()

	// Remove from AniListToFiles
	if files, exists := gms.cache.AniListToFiles[aniListID]; exists {
		for i, file := range files {
			if file == filePath {
				gms.cache.AniListToFiles[aniListID] = append(files[:i], files[i+1:]...)
				break
			}
		}
		// Remove the slice if it's empty
		if len(gms.cache.AniListToFiles[aniListID]) == 0 {
			delete(gms.cache.AniListToFiles, aniListID)
		}
	}
	// Remove from FileToAniList
	delete(gms.cache.FileToAniList, filePath)
}