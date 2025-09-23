package sync

import (
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"seanime/internal/events"
	"seanime/internal/global_mapping"
	"seanime/internal/library/anime"
	"seanime/internal/library/scanner"
	"sync"

	"github.com/rs/zerolog"
)

// SyncManager coordinates all sync-related functionality
type SyncManager struct {
	db                *db.Database
	logger            *zerolog.Logger

	// Core components
	enhancedWS        *events.EnhancedWSEventManager
	localFileManager  *LocalFileManager
	progressManager   *ProgressManager
	enhancedWatcher   *scanner.EnhancedWatcher
	globalMappingSvc  *global_mapping.GlobalMappingService

	// Configuration
	libraryPaths      []string

	// State
	isRunning         bool
	mu                sync.RWMutex
}

// SyncManagerOptions contains options for creating a sync manager
type SyncManagerOptions struct {
	Database           *db.Database
	Logger             *zerolog.Logger
	LibraryPaths       []string
	GlobalMappingService *global_mapping.GlobalMappingService
}

// NewSyncManager creates a new sync manager that coordinates all real-time sync functionality
func NewSyncManager(opts *SyncManagerOptions) (*SyncManager, error) {
	sm := &SyncManager{
		db:                opts.Database,
		logger:            opts.Logger,
		libraryPaths:      opts.LibraryPaths,
		globalMappingSvc:  opts.GlobalMappingService,
	}
	
	// Initialize components in order
	err := sm.initializeComponents()
	if err != nil {
		return nil, err
	}
	
	sm.logger.Info().Msg("sync-manager: Initialized successfully")
	
	return sm, nil
}

// initializeComponents sets up all sync system components
func (sm *SyncManager) initializeComponents() error {
	// 1. Enhanced WebSocket Manager
	sm.enhancedWS = events.NewEnhancedWSEventManager(sm.logger, sm.db)
	
	// 2. LocalFile Manager
	sm.localFileManager = NewLocalFileManager(sm.db, sm.enhancedWS, sm.logger, sm.globalMappingSvc)
	
	// 3. Progress Manager
	sm.progressManager = NewProgressManager(sm.db, sm.enhancedWS, sm.logger)
	
	// 4. Enhanced Watcher
	var err error
	sm.enhancedWatcher, err = scanner.NewEnhancedWatcher(&scanner.EnhancedWatcherOptions{
		Logger:            sm.logger,
		EnhancedWS:        sm.enhancedWS,
		LocalFileProcessor: sm.localFileManager,
		Database:          sm.db,
		LibraryPaths:      sm.libraryPaths,
		ProcessingWorkers: 3,
	})
	if err != nil {
		return err
	}
	
	return nil
}

// Start begins all sync operations
func (sm *SyncManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if sm.isRunning {
		return nil
	}
	
	// Initialize file watcher
	err := sm.enhancedWatcher.InitLibraryFileWatcher()
	if err != nil {
		return err
	}
	
	// Start file watcher
	sm.enhancedWatcher.StartWatching()
	
	sm.isRunning = true
	sm.logger.Info().Msg("sync-manager: Started all sync operations")
	
	return nil
}

// Stop gracefully shuts down all sync operations
func (sm *SyncManager) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if !sm.isRunning {
		return
	}
	
	// Stop file watcher
	sm.enhancedWatcher.StopWatching()
	
	// Stop progress manager
	sm.progressManager.Close()
	
	sm.isRunning = false
	sm.logger.Info().Msg("sync-manager: Stopped all sync operations")
}

// RegisterUserSession registers a WebSocket session with a user
func (sm *SyncManager) RegisterUserSession(userID uint, sessionID string) {
	sm.enhancedWS.RegisterUserSession(userID, sessionID)
}

// UnregisterUserSession unregisters a WebSocket session
func (sm *SyncManager) UnregisterUserSession(sessionID string) {
	sm.enhancedWS.UnregisterUserSession(sessionID)
}

// SyncUserLibrary synchronizes a user's library subscriptions
func (sm *SyncManager) SyncUserLibrary(userID uint, mediaIDs []int) error {
	return sm.localFileManager.SyncUserLibrarySubscriptions(userID, mediaIDs)
}

// HandleUserLibraryChanged updates subscriptions when user's library changes
func (sm *SyncManager) HandleUserLibraryChanged(userID uint, addedMediaIDs []int, removedMediaIDs []int) error {
	return sm.localFileManager.HandleUserLibraryChanged(userID, addedMediaIDs, removedMediaIDs)
}

// StartWatching starts progress tracking for a user
func (sm *SyncManager) StartWatching(userID uint, sessionID string, state *UserPlaybackState) error {
	return sm.progressManager.StartWatching(userID, sessionID, state)
}

// UpdateProgress updates playback progress for a user
func (sm *SyncManager) UpdateProgress(sessionID string, state *UserPlaybackState) error {
	return sm.progressManager.UpdateProgress(sessionID, state)
}

// PauseWatching pauses progress tracking
func (sm *SyncManager) PauseWatching(sessionID string, state *UserPlaybackState) error {
	return sm.progressManager.PauseWatching(sessionID, state)
}

// StopWatching stops progress tracking
func (sm *SyncManager) StopWatching(sessionID string, state *UserPlaybackState) error {
	return sm.progressManager.StopWatching(sessionID, state)
}

// GetResumePoint gets resume point for an episode
func (sm *SyncManager) GetResumePoint(userID uint, mediaID int, episodeNumber int) (*ResumePoint, error) {
	return sm.progressManager.GetResumePoint(userID, mediaID, episodeNumber)
}

// GetUserProgress gets all progress for a user's media
func (sm *SyncManager) GetUserProgress(userID uint, mediaID int) ([]*models.UserEpisodeProgress, error) {
	return sm.progressManager.GetUserProgress(userID, mediaID)
}

// ProcessBulkLocalFiles processes bulk LocalFile updates (e.g., from scan)
func (sm *SyncManager) ProcessBulkLocalFiles(localFiles []*anime.LocalFile) error {
	return sm.localFileManager.ProcessBulkLocalFiles(localFiles)
}

// GetEnhancedWSEventManager returns the enhanced WebSocket manager for direct access
func (sm *SyncManager) GetEnhancedWSEventManager() *events.EnhancedWSEventManager {
	return sm.enhancedWS
}

// GetLocalFileManager returns the LocalFile manager for direct access
func (sm *SyncManager) GetLocalFileManager() *LocalFileManager {
	return sm.localFileManager
}

// GetProgressManager returns the progress manager for direct access
func (sm *SyncManager) GetProgressManager() *ProgressManager {
	return sm.progressManager
}

// GetStats returns comprehensive statistics about the sync system
func (sm *SyncManager) GetStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	stats := map[string]interface{}{
		"isRunning":     sm.isRunning,
		"libraryPaths":  sm.libraryPaths,
	}
	
	if sm.enhancedWatcher != nil {
		stats["watcher"] = sm.enhancedWatcher.GetStats()
	}
	
	// Add component-specific stats here as needed
	
	return stats
}