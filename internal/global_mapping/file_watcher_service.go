package global_mapping

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
)

type FileWatcherService struct {
	watcher    *fsnotify.Watcher
	logger     *zerolog.Logger
	gms        *GlobalMappingService
	watchPaths []string
	stopCh     chan struct{}
}

func NewFileWatcherService(logger *zerolog.Logger, gms *GlobalMappingService) *FileWatcherService {
	return &FileWatcherService{
		logger:     logger,
		gms:        gms,
		watchPaths: make([]string, 0),
		stopCh:     make(chan struct{}),
	}
}

// Start initializes the file watcher and begins monitoring
func (fws *FileWatcherService) Start(watchPaths []string) error {
	fws.logger.Info().
		Strs("paths", watchPaths).
		Msg("file_watcher: Starting file watcher service")

	var err error
	fws.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	fws.watchPaths = watchPaths

	// Add paths to watcher
	for _, path := range watchPaths {
		if err := fws.addPathRecursively(path); err != nil {
			fws.logger.Error().Err(err).Str("path", path).Msg("file_watcher: Failed to add watch path")
			continue
		}
		fws.logger.Info().Str("path", path).Msg("file_watcher: Added watch path")
	}

	// Start watching in a goroutine
	go fws.watchLoop()

	return nil
}

// Stop stops the file watcher service
func (fws *FileWatcherService) Stop() {
	fws.logger.Info().Msg("file_watcher: Stopping file watcher service")

	close(fws.stopCh)

	if fws.watcher != nil {
		fws.watcher.Close()
	}
}

// addPathRecursively adds a path and all its subdirectories to the watcher
func (fws *FileWatcherService) addPathRecursively(rootPath string) error {
	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only add directories to the watcher
		if info.IsDir() {
			err = fws.watcher.Add(path)
			if err != nil {
				fws.logger.Error().Err(err).Str("path", path).Msg("file_watcher: Failed to add directory")
				return err
			}
		}

		return nil
	})
}

// watchLoop is the main event loop for file watching
func (fws *FileWatcherService) watchLoop() {
	debounceTimer := make(map[string]*time.Timer)

	for {
		select {
		case event, ok := <-fws.watcher.Events:
			if !ok {
				return
			}

			// Filter for video files and CREATE/WRITE events
			if fws.isVideoFile(event.Name) && (event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write) {
				fws.logger.Debug().
					Str("file", event.Name).
					Str("op", event.Op.String()).
					Msg("file_watcher: File event detected")

				// Debounce file events (files might be written in chunks)
				fws.debounceFileEvent(event.Name, debounceTimer)
			}

			// Handle directory creation to add new directories to watcher
			if event.Op&fsnotify.Create == fsnotify.Create {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					fws.logger.Debug().Str("dir", event.Name).Msg("file_watcher: New directory created, adding to watcher")
					if err := fws.watcher.Add(event.Name); err != nil {
						fws.logger.Error().Err(err).Str("dir", event.Name).Msg("file_watcher: Failed to add new directory")
					}
				}
			}

		case err, ok := <-fws.watcher.Errors:
			if !ok {
				return
			}
			fws.logger.Error().Err(err).Msg("file_watcher: Watcher error")

		case <-fws.stopCh:
			fws.logger.Info().Msg("file_watcher: Watch loop stopped")
			return
		}
	}
}

// debounceFileEvent debounces file events to avoid processing incomplete files
func (fws *FileWatcherService) debounceFileEvent(filePath string, debounceTimer map[string]*time.Timer) {
	// Cancel existing timer for this file
	if timer, exists := debounceTimer[filePath]; exists {
		timer.Stop()
		delete(debounceTimer, filePath)
	}

	// Create new timer
	debounceTimer[filePath] = time.AfterFunc(2*time.Second, func() {
		delete(debounceTimer, filePath)
		fws.processNewFile(filePath)
	})
}

// processNewFile processes a newly detected file
func (fws *FileWatcherService) processNewFile(filePath string) {
	fws.logger.Info().Str("file", filePath).Msg("file_watcher: Processing new file")

	// Verify file still exists and is accessible
	if _, err := os.Stat(filePath); err != nil {
		fws.logger.Debug().Err(err).Str("file", filePath).Msg("file_watcher: File no longer accessible, skipping")
		return
	}

	// Process the file through GlobalMappingService
	if err := fws.gms.ProcessNewFile(filePath); err != nil {
		fws.logger.Error().Err(err).Str("file", filePath).Msg("file_watcher: Failed to process new file")
	}
}

// isVideoFile checks if a file has a video extension
func (fws *FileWatcherService) isVideoFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	videoExtensions := []string{
		".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm",
		".m4v", ".3gp", ".3g2", ".f4v", ".asf", ".rm", ".rmvb",
		".vob", ".ogv", ".drc", ".mng", ".qt", ".yuv", ".mpg",
		".mpeg", ".m2v", ".m4p", ".mp2", ".mpe", ".mpv", ".m2ts",
		".mts", ".ts", ".divx", ".xvid",
	}

	for _, videoExt := range videoExtensions {
		if ext == videoExt {
			return true
		}
	}
	return false
}

// AddWatchPath adds a new path to watch
func (fws *FileWatcherService) AddWatchPath(path string) error {
	fws.logger.Info().Str("path", path).Msg("file_watcher: Adding new watch path")

	// Add to internal list
	fws.watchPaths = append(fws.watchPaths, path)

	// Add to watcher if it's running
	if fws.watcher != nil {
		return fws.addPathRecursively(path)
	}

	return nil
}

// RemoveWatchPath removes a path from watching
func (fws *FileWatcherService) RemoveWatchPath(path string) error {
	fws.logger.Info().Str("path", path).Msg("file_watcher: Removing watch path")

	// Remove from internal list
	for i, watchPath := range fws.watchPaths {
		if watchPath == path {
			fws.watchPaths = append(fws.watchPaths[:i], fws.watchPaths[i+1:]...)
			break
		}
	}

	// Remove from watcher if it's running
	if fws.watcher != nil {
		return fws.watcher.Remove(path)
	}

	return nil
}

// GetWatchPaths returns the currently watched paths
func (fws *FileWatcherService) GetWatchPaths() []string {
	return append([]string(nil), fws.watchPaths...) // Return copy
}