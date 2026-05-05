package filecleanup

import (
	"os"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
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
	}
)

func NewManager(logger *zerolog.Logger, database *db.Database) *Manager {
	m := &Manager{
		logger:     logger,
		database:   database,
		taskQueue:  make(chan *CleanupTask, 100),
		stopChan:   make(chan struct{}),
		maxRetries: 30,        // Retry 30 times
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

		case <-m.stopChan:
			return
		}
	}
}

// Stop gracefully stops the cleanup workers
func (m *Manager) Stop() {
	close(m.stopChan)
	close(m.taskQueue)
	m.workerWg.Wait()
}
