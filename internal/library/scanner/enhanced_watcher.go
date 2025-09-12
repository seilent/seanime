package scanner

import (
	"context"
	"os"
	"path/filepath"
	"seanime/internal/database/db"
	"seanime/internal/events"
	"seanime/internal/library"
	"seanime/internal/library/anime"
	"seanime/internal/library/filesystem"
	"seanime/internal/util"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
)

// EnhancedWatcher extends the basic file watcher with real-time LocalFile processing
type EnhancedWatcher struct {
	watcher           *fsnotify.Watcher
	logger            *zerolog.Logger
	enhancedWS        *events.EnhancedWSEventManager
	localFileProcessor library.LocalFileProcessor
	database          *db.Database
	
	// Library paths for context
	libraryPaths      []string
	
	// File processing
	fileQueue         chan FileEvent
	processingWorkers int
	
	// Debouncing for rapid file changes
	debounceMap       map[string]*time.Timer
	debounceMu        sync.Mutex
	debounceInterval  time.Duration
	
	// Context for shutdown
	ctx               context.Context
	cancel            context.CancelFunc
}

// FileEvent represents a file system event to be processed
type FileEvent struct {
	Path      string
	Operation string // "created", "removed", "modified"
	Timestamp time.Time
}

// EnhancedWatcherOptions contains options for creating an enhanced watcher
type EnhancedWatcherOptions struct {
	Logger            *zerolog.Logger
	EnhancedWS        *events.EnhancedWSEventManager
	LocalFileProcessor library.LocalFileProcessor
	Database          *db.Database
	LibraryPaths      []string
	ProcessingWorkers int // Number of workers to process file events
}

// NewEnhancedWatcher creates a new enhanced file watcher
func NewEnhancedWatcher(opts *EnhancedWatcherOptions) (*EnhancedWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	workers := opts.ProcessingWorkers
	if workers == 0 {
		workers = 3 // Default to 3 workers
	}
	
	ew := &EnhancedWatcher{
		watcher:           watcher,
		logger:            opts.Logger,
		enhancedWS:        opts.EnhancedWS,
		localFileProcessor: opts.LocalFileProcessor,
		database:          opts.Database,
		libraryPaths:      opts.LibraryPaths,
		fileQueue:         make(chan FileEvent, 100),
		processingWorkers: workers,
		debounceMap:       make(map[string]*time.Timer),
		debounceInterval:  1 * time.Second, // Debounce rapid changes for 1 second
		ctx:               ctx,
		cancel:            cancel,
	}
	
	return ew, nil
}

// InitLibraryFileWatcher starts watching the specified directories
func (ew *EnhancedWatcher) InitLibraryFileWatcher() error {
	// Watch all library paths and their subdirectories
	for _, libraryPath := range ew.libraryPaths {
		err := ew.watchDirectory(libraryPath)
		if err != nil {
			return err
		}
	}
	
	ew.logger.Info().
		Strs("paths", ew.libraryPaths).
		Msg("enhanced-watcher: Watching library directories")
	
	return nil
}

// watchDirectory recursively adds a directory and its subdirectories to the watcher
func (ew *EnhancedWatcher) watchDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip paths that can't be accessed
		}
		
		if info.IsDir() {
			return ew.watcher.Add(path)
		}
		
		return nil
	})
}

// StartWatching begins monitoring file system events and processing them
func (ew *EnhancedWatcher) StartWatching() {
	// Start file event processors
	for i := 0; i < ew.processingWorkers; i++ {
		go ew.fileEventProcessor(i)
	}
	
	// Start the main event loop
	go ew.eventLoop()
	
	ew.logger.Info().
		Int("workers", ew.processingWorkers).
		Msg("enhanced-watcher: Started watching with workers")
}

// StopWatching stops the enhanced watcher
func (ew *EnhancedWatcher) StopWatching() {
	ew.cancel()
	
	err := ew.watcher.Close()
	if err != nil {
		ew.logger.Error().Err(err).Msg("enhanced-watcher: Error closing watcher")
	}
	
	close(ew.fileQueue)
	
	ew.logger.Info().Msg("enhanced-watcher: Stopped watching")
}

// eventLoop handles incoming file system events
func (ew *EnhancedWatcher) eventLoop() {
	for {
		select {
		case <-ew.ctx.Done():
			return
			
		case event, ok := <-ew.watcher.Events:
			if !ok {
				return
			}
			
			ew.handleFileSystemEvent(event)
			
		case err, ok := <-ew.watcher.Errors:
			if !ok {
				return
			}
			ew.logger.Error().Err(err).Msg("enhanced-watcher: File system watcher error")
		}
	}
}

// handleFileSystemEvent processes a raw file system event
func (ew *EnhancedWatcher) handleFileSystemEvent(event fsnotify.Event) {
	// Skip temporary files and partial downloads
	if strings.Contains(event.Name, ".part") || 
	   strings.Contains(event.Name, ".tmp") || 
	   strings.Contains(event.Name, ".crdownload") {
		return
	}
	
	// Only process video files
	if !util.IsValidMediaFile(event.Name) {
		return
	}
	
	// Determine operation type
	var operation string
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		operation = "created"
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		operation = "removed"
	case event.Op&fsnotify.Write == fsnotify.Write:
		operation = "modified"
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		operation = "removed" // Treat rename as removal of old path
	default:
		return // Skip other operations
	}
	
	// Debounce rapid changes to the same file
	ew.debounceFileEvent(event.Name, operation)
}

// debounceFileEvent prevents processing rapid successive events for the same file
func (ew *EnhancedWatcher) debounceFileEvent(filePath, operation string) {
	ew.debounceMu.Lock()
	defer ew.debounceMu.Unlock()
	
	// Cancel existing timer if present
	if existingTimer, exists := ew.debounceMap[filePath]; exists {
		existingTimer.Stop()
	}
	
	// Create new timer
	ew.debounceMap[filePath] = time.AfterFunc(ew.debounceInterval, func() {
		// Queue the file event for processing
		fileEvent := FileEvent{
			Path:      filePath,
			Operation: operation,
			Timestamp: time.Now(),
		}
		
		select {
		case ew.fileQueue <- fileEvent:
		case <-ew.ctx.Done():
			return
		default:
			ew.logger.Warn().
				Str("path", filePath).
				Str("operation", operation).
				Msg("enhanced-watcher: File event queue full, dropping event")
		}
		
		// Clean up debounce map
		ew.debounceMu.Lock()
		delete(ew.debounceMap, filePath)
		ew.debounceMu.Unlock()
	})
}

// fileEventProcessor processes queued file events
func (ew *EnhancedWatcher) fileEventProcessor(workerID int) {
	ew.logger.Debug().Int("workerId", workerID).Msg("enhanced-watcher: File event processor started")
	
	for {
		select {
		case <-ew.ctx.Done():
			return
			
		case fileEvent, ok := <-ew.fileQueue:
			if !ok {
				return
			}
			
			ew.processFileEvent(fileEvent, workerID)
		}
	}
}

// processFileEvent handles an individual file event
func (ew *EnhancedWatcher) processFileEvent(fileEvent FileEvent, workerID int) {
	defer func() {
		if r := recover(); r != nil {
			ew.logger.Error().
				Interface("panic", r).
				Str("path", fileEvent.Path).
				Int("workerId", workerID).
				Msg("enhanced-watcher: Panic in file event processor")
		}
	}()
	
	switch fileEvent.Operation {
	case "created":
		ew.handleFileCreated(fileEvent.Path, workerID)
	case "removed":
		ew.handleFileRemoved(fileEvent.Path, workerID)
	case "modified":
		ew.handleFileModified(fileEvent.Path, workerID)
	}
}

// handleFileCreated processes a newly created file
func (ew *EnhancedWatcher) handleFileCreated(filePath string, workerID int) {
	// Verify file still exists (might have been moved/deleted quickly)
	if !filesystem.FileExists(filePath) {
		return
	}
	
	// Create LocalFile object
	localFile := anime.NewLocalFileS(filePath, ew.libraryPaths)
	if localFile == nil {
		ew.logger.Warn().
			Str("path", filePath).
			Int("workerId", workerID).
			Msg("enhanced-watcher: Failed to create LocalFile")
		return
	}
	
	// Try to match with existing media (basic matching)
	mediaID := ew.attemptMediaMatching(localFile)
	localFile.MediaId = mediaID
	
	// Send to LocalFile processor for processing
	err := ew.localFileProcessor.ProcessNewLocalFile(localFile)
	if err != nil {
		ew.logger.Error().Err(err).
			Str("path", filePath).
			Int("workerId", workerID).
			Msg("enhanced-watcher: Failed to process new LocalFile")
		return
	}
	
	// Send basic WebSocket event for immediate UI feedback
	ew.enhancedWS.SendEvent(events.LibraryWatcherFileAdded, map[string]interface{}{
		"path":    filePath,
		"mediaId": mediaID,
	})
	
	ew.logger.Debug().
		Str("path", filePath).
		Int("mediaId", mediaID).
		Int("workerId", workerID).
		Msg("enhanced-watcher: File created and processed")
}

// handleFileRemoved processes a removed file
func (ew *EnhancedWatcher) handleFileRemoved(filePath string, workerID int) {
	// Try to determine which media this file belonged to
	// This could be enhanced by maintaining a cache of path -> mediaID mappings
	mediaID := ew.getMediaIDFromFilePath(filePath)
	
	// Send to LocalFile processor
	err := ew.localFileProcessor.ProcessRemovedLocalFile(filePath, mediaID)
	if err != nil {
		ew.logger.Error().Err(err).
			Str("path", filePath).
			Int("workerId", workerID).
			Msg("enhanced-watcher: Failed to process removed LocalFile")
		return
	}
	
	// Send basic WebSocket event
	ew.enhancedWS.SendEvent(events.LibraryWatcherFileRemoved, map[string]interface{}{
		"path":    filePath,
		"mediaId": mediaID,
	})
	
	ew.logger.Debug().
		Str("path", filePath).
		Int("mediaId", mediaID).
		Int("workerId", workerID).
		Msg("enhanced-watcher: File removed and processed")
}

// handleFileModified processes a modified file
func (ew *EnhancedWatcher) handleFileModified(filePath string, workerID int) {
	// Verify file still exists
	if !filesystem.FileExists(filePath) {
		return
	}
	
	// Create updated LocalFile object
	localFile := anime.NewLocalFileS(filePath, ew.libraryPaths)
	if localFile == nil {
		return
	}
	
	// Try to match with existing media
	mediaID := ew.attemptMediaMatching(localFile)
	localFile.MediaId = mediaID
	
	// Send to LocalFile processor
	err := ew.localFileProcessor.ProcessModifiedLocalFile(localFile)
	if err != nil {
		ew.logger.Error().Err(err).
			Str("path", filePath).
			Int("workerId", workerID).
			Msg("enhanced-watcher: Failed to process modified LocalFile")
		return
	}
	
	ew.logger.Debug().
		Str("path", filePath).
		Int("mediaId", mediaID).
		Int("workerId", workerID).
		Msg("enhanced-watcher: File modified and processed")
}

// attemptMediaMatching tries to match a LocalFile with existing media
// This is a simplified version - in practice, you'd want more sophisticated matching
func (ew *EnhancedWatcher) attemptMediaMatching(localFile *anime.LocalFile) int {
	if localFile.ParsedData == nil {
		return 0
	}
	
	// This is a placeholder - you would implement actual media matching logic here
	// For now, we'll return 0 (unmatched) and let the full scan process handle matching
	
	// In a full implementation, this might:
	// 1. Query database for media with similar titles
	// 2. Use fuzzy matching algorithms
	// 3. Check against known media in user libraries
	// 4. Use external APIs for verification
	
	return 0
}

// getMediaIDFromFilePath attempts to determine media ID from file path
// This could be enhanced with a cache or database lookup
func (ew *EnhancedWatcher) getMediaIDFromFilePath(filePath string) int {
	// Placeholder implementation
	// In practice, you might:
	// 1. Check a cache of path -> mediaID mappings
	// 2. Parse the file path to extract media information
	// 3. Query the database for matching LocalFiles
	
	return 0
}

// GetStats returns statistics about the watcher
func (ew *EnhancedWatcher) GetStats() map[string]interface{} {
	ew.debounceMu.Lock()
	pendingEvents := len(ew.debounceMap)
	ew.debounceMu.Unlock()
	
	return map[string]interface{}{
		"watchedPaths":     ew.libraryPaths,
		"processingWorkers": ew.processingWorkers,
		"queuedEvents":     len(ew.fileQueue),
		"pendingEvents":    pendingEvents,
		"debounceInterval": ew.debounceInterval.String(),
	}
}