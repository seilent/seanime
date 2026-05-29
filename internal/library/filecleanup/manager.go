package filecleanup

import (
	"os"
	"path/filepath"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type (
	CleanupTask struct {
		FilePath string
		MediaID  int
		Mapping  *models.GlobalAnimeFileMapping
		Retries  int
	}

	Manager struct {
		logger     *zerolog.Logger
		database   *db.Database
		taskQueue  chan *CleanupTask
		workerWg   sync.WaitGroup
		stopChan   chan struct{}
		maxRetries int
		retryDelay time.Duration
		roots      []string
		rootsOnce  sync.Once
	}
)

func NewManager(logger *zerolog.Logger, database *db.Database) *Manager {
	m := &Manager{
		logger:     logger,
		database:   database,
		taskQueue:  make(chan *CleanupTask, 100),
		stopChan:   make(chan struct{}),
		maxRetries: 30,          // Retry 30 times
		retryDelay: time.Minute, // Retry every minute
	}

	// Start background workers
	for i := 0; i < 3; i++ {
		m.workerWg.Add(1)
		go m.worker()
	}

	return m
}

// CleanupFiles attempts to delete the provided file mappings immediately.
// Files that cannot be deleted (e.g., locked/in use) are added to a background cleanup queue.
func (m *Manager) CleanupFiles(mappings []*models.GlobalAnimeFileMapping) error {
	var lastErr error
	queuedCount := 0
	deletedCount := 0

	for _, mapping := range mappings {
		err := os.Remove(mapping.LocalFilePath)
		if err != nil {
			// File is locked or doesn't exist, add to queue
			m.taskQueue <- &CleanupTask{
				FilePath: mapping.LocalFilePath,
				MediaID:  mapping.AniListID,
				Mapping:  mapping,
				Retries:  0,
			}
			queuedCount++
			lastErr = err
			m.logger.Debug().Err(err).Str("file", mapping.LocalFilePath).
				Msg("File could not be deleted immediately, queued for background cleanup")
		} else {
			// Successfully deleted, remove from database
			err = m.database.DeleteGlobalMapping(mapping.LocalFilePath)
			if err != nil {
				m.logger.Error().Err(err).Str("filePath", mapping.LocalFilePath).
					Msg("Failed to delete global mapping from database after file deletion")
			}
			deletedCount++
			m.logger.Debug().Str("file", mapping.LocalFilePath).
				Msg("File deleted successfully")
			m.removeEmptyParents(mapping.LocalFilePath)
		}
	}

	if queuedCount > 0 {
		m.logger.Info().Int("queued", queuedCount).Int("deleted", deletedCount).
			Msg("Files processed for cleanup")
	}

	return lastErr
}

func (m *Manager) worker() {
	defer m.workerWg.Done()

	for {
		select {
		case task, ok := <-m.taskQueue:
			if !ok {
				// Channel closed
				return
			}

			if task.Retries >= m.maxRetries {
				m.logger.Warn().Str("file", task.FilePath).Int("retries", task.Retries).
					Msg("File cleanup failed after max retries, giving up")
				continue
			}

			// Wait before retry (except for first attempt which happens immediately)
			if task.Retries > 0 {
				select {
				case <-time.After(m.retryDelay):
				case <-m.stopChan:
					return
				}
			}

			err := os.Remove(task.FilePath)
			if err != nil {
				// Still locked, retry later
				task.Retries++
				m.taskQueue <- task
				m.logger.Debug().Err(err).Str("file", task.FilePath).Int("retry", task.Retries).
					Msg("File still locked, requeued for cleanup")
				continue
			}

			// Successfully deleted, remove from database
			err = m.database.DeleteGlobalMapping(task.FilePath)
			if err != nil {
				m.logger.Error().Err(err).Str("filePath", task.FilePath).
					Msg("Failed to delete global mapping from database after delayed file deletion")
			}

			m.logger.Info().Str("file", task.FilePath).Int("retries", task.Retries).
				Msg("File cleaned up successfully via background worker")
			m.removeEmptyParents(task.FilePath)

		case <-m.stopChan:
			return
		}
	}
}

// libraryRoots returns the configured library root directories, loaded once and cached.
func (m *Manager) libraryRoots() []string {
	m.rootsOnce.Do(func() {
		gs, err := m.database.GetGlobalSettings()
		if err != nil || gs.Library == nil {
			return
		}
		for _, p := range gs.Library.GetLibraryPaths() {
			if p != "" {
				m.roots = append(m.roots, filepath.Clean(p))
			}
		}
	})
	return m.roots
}

// removeEmptyParents removes now-empty parent directories of filePath, walking up until it
// reaches (but never removes) a library root or a non-empty directory. This cleans up the
// anime-title/release folders left behind after all of an entry's episodes are deleted.
func (m *Manager) removeEmptyParents(filePath string) {
	roots := m.libraryRoots()
	dir := filepath.Clean(filepath.Dir(filePath))
	for {
		// Only ever remove directories that live strictly inside a configured library root.
		inside := false
		for _, root := range roots {
			if dir != root && strings.HasPrefix(dir, root+string(os.PathSeparator)) {
				inside = true
				break
			}
		}
		if !inside {
			return
		}
		// os.Remove only succeeds on an empty directory; a non-empty dir stops the walk.
		if err := os.Remove(dir); err != nil {
			return
		}
		m.logger.Debug().Str("dir", dir).Msg("Removed empty parent directory after file cleanup")
		dir = filepath.Dir(dir)
	}
}

// Stop gracefully stops the cleanup workers
func (m *Manager) Stop() {
	close(m.stopChan)
	close(m.taskQueue)
	m.workerWg.Wait()
}
