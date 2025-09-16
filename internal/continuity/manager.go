package continuity

import (
	"github.com/rs/zerolog"
	"github.com/samber/mo"
	"seanime/internal/database/db"
	libsync "seanime/internal/library/sync"
	"seanime/internal/util/filecache"
	"sync"
	"time"
)

const (
	OnlinestreamKind   Kind = "onlinestream"
	MediastreamKind    Kind = "mediastream"
	ExternalPlayerKind Kind = "external_player"
)

type (
	// Manager is used to manage the user's viewing history across different media types.
	Manager struct {
		fileCacher                  *filecache.Cacher
		db                          *db.Database
		watchHistoryFileCacheBucket *filecache.Bucket
		syncManager                 *libsync.SyncManager

		externalPlayerEpisodeDetails mo.Option[*ExternalPlayerEpisodeDetails]

		logger   *zerolog.Logger
		settings *Settings
		mu       sync.RWMutex
	}

	// ExternalPlayerEpisodeDetails is used to store the episode details when using an external player.
	// Since the media player module only cares about the filepath, the PlaybackManager will store the episode number and media id here when playback starts.
	ExternalPlayerEpisodeDetails struct {
		EpisodeNumber int    `json:"episodeNumber"`
		MediaId       int    `json:"mediaId"`
		Filepath      string `json:"filepath"`
	}

	Settings struct {
		WatchContinuityEnabled bool
	}

	Kind string
)

type (
	NewManagerOptions struct {
		FileCacher  *filecache.Cacher
		Logger      *zerolog.Logger
		Database    *db.Database
		SyncManager *libsync.SyncManager
	}
)

// NewManager creates a new Manager, it should be initialized once.
func NewManager(opts *NewManagerOptions) *Manager {
	// File cache is no longer used for watch history - keeping bucket for backwards compatibility
	watchHistoryFileCacheBucket := filecache.NewBucket("watch_history", time.Hour*24*99999)

	ret := &Manager{
		fileCacher:                  opts.FileCacher,
		logger:                      opts.Logger,
		db:                          opts.Database,
		syncManager:                 opts.SyncManager,
		watchHistoryFileCacheBucket: &watchHistoryFileCacheBucket,
		settings: &Settings{
			WatchContinuityEnabled: true,
		},
		externalPlayerEpisodeDetails: mo.None[*ExternalPlayerEpisodeDetails](),
	}

	ret.logger.Info().Msg("continuity: Initialized manager")

	return ret
}

// SetSettings should be called after initializing the Manager.
func (m *Manager) SetSettings(settings *Settings) {
	if m == nil || settings == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings = settings
}

// GetSettings returns the current settings.
func (m *Manager) GetSettings() *Settings {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.settings
}

// GetSyncManager returns the sync manager for progress operations.
func (m *Manager) GetSyncManager() *libsync.SyncManager {
	if m == nil {
		return nil
	}
	return m.syncManager
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (m *Manager) SetExternalPlayerEpisodeDetails(details *ExternalPlayerEpisodeDetails) {
	if m == nil || details == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.externalPlayerEpisodeDetails = mo.Some(details)
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// User-specific methods that bridge to the ProgressManager
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// UpdateWatchHistoryItemForUser updates a watch history item for a specific user using the ProgressManager
func (m *Manager) UpdateWatchHistoryItemForUser(userID uint, opts *UpdateWatchHistoryItemOptions) error {
	if m == nil || m.syncManager == nil || opts == nil {
		return nil
	}

	// Convert continuity options to progress manager format
	progressManager := m.syncManager.GetProgressManager()
	if progressManager == nil {
		return nil
	}

	// Create user playback state from continuity options
	state := &libsync.UserPlaybackState{
		UserID:               userID,
		EpisodeNumber:        opts.EpisodeNumber,
		MediaId:              opts.MediaId,
		CurrentTimeSeconds:   int(opts.CurrentTime),
		DurationSeconds:      int(opts.Duration),
		CompletionPercentage: (opts.CurrentTime / opts.Duration) * 100,
		LocalFilePath:        opts.Filepath,
		MediaPlayerUsed:      string(opts.Kind),
		LastUpdatedAt:        time.Now(),
	}

	// Save progress using the progress manager
	return progressManager.SaveProgress(userID, state)
}

// GetWatchHistoryItemForUser retrieves a watch history item for a specific user using the ProgressManager
func (m *Manager) GetWatchHistoryItemForUser(userID uint, mediaId int) *WatchHistoryItemResponse {
	if m == nil || m.syncManager == nil {
		return &WatchHistoryItemResponse{Item: nil, Found: false}
	}

	progressManager := m.syncManager.GetProgressManager()
	if progressManager == nil {
		return &WatchHistoryItemResponse{Item: nil, Found: false}
	}

	// Get user progress for the media
	progressList, err := progressManager.GetUserProgress(userID, mediaId)
	if err != nil || len(progressList) == 0 {
		return &WatchHistoryItemResponse{Item: nil, Found: false}
	}

	// Find the most recent episode progress
	var latestProgress *libsync.ResumePoint
	var latestEpisode int
	for _, progress := range progressList {
		if progress.EpisodeNumber > latestEpisode && !progress.IsCompleted {
			resumePoint, _ := progressManager.GetResumePoint(userID, mediaId, progress.EpisodeNumber)
			if resumePoint != nil {
				latestProgress = resumePoint
				latestEpisode = progress.EpisodeNumber
			}
		}
	}

	if latestProgress == nil {
		return &WatchHistoryItemResponse{Item: nil, Found: false}
	}

	// Convert to continuity format
	item := &WatchHistoryItem{
		Kind:          ExternalPlayerKind,
		Filepath:      latestProgress.LocalFilePath,
		MediaId:       mediaId,
		EpisodeNumber: latestProgress.EpisodeNumber,
		CurrentTime:   float64(latestProgress.ResumeTimeSeconds),
		Duration:      float64(latestProgress.DurationSeconds),
		TimeAdded:     latestProgress.LastWatchedAt,
		TimeUpdated:   latestProgress.LastWatchedAt,
	}

	return &WatchHistoryItemResponse{Item: item, Found: true}
}

// GetWatchHistoryForUser retrieves all watch history for a specific user using the ProgressManager
func (m *Manager) GetWatchHistoryForUser(userID uint) WatchHistory {
	if m == nil || m.syncManager == nil {
		return make(map[int]*WatchHistoryItem)
	}

	progressManager := m.syncManager.GetProgressManager()
	if progressManager == nil {
		return make(map[int]*WatchHistoryItem)
	}

	// For now, return empty map as we'd need to query all media for this user
	// This is a complex operation that might not be needed
	// The individual item lookup is more important
	return make(map[int]*WatchHistoryItem)
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// User-specific methods for external player tracking
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// UpdateExternalPlayerEpisodeWatchHistoryItemForUser updates watch history for external player with user context
func (m *Manager) UpdateExternalPlayerEpisodeWatchHistoryItemForUser(userID uint, currentTime, duration float64) {
	if !m.settings.WatchContinuityEnabled {
		return
	}

	if m.externalPlayerEpisodeDetails.IsAbsent() {
		return
	}

	opts, ok := m.externalPlayerEpisodeDetails.Get()
	if !ok {
		return
	}

	// Use user-specific method
	_ = m.UpdateWatchHistoryItemForUser(userID, &UpdateWatchHistoryItemOptions{
		CurrentTime:   currentTime,
		Duration:      duration,
		MediaId:       opts.MediaId,
		EpisodeNumber: opts.EpisodeNumber,
		Filepath:      opts.Filepath,
		Kind:          ExternalPlayerKind,
	})
}

// DeleteWatchHistoryItemForUser removes progress for a specific user
func (m *Manager) DeleteWatchHistoryItemForUser(userID uint, mediaId int) error {
	if m == nil || m.syncManager == nil {
		return nil
	}

	progressManager := m.syncManager.GetProgressManager()
	if progressManager == nil {
		return nil
	}

	// Get all progress for this media and user, then delete each episode
	progressList, err := progressManager.GetUserProgress(userID, mediaId)
	if err != nil {
		return err
	}

	// Delete each episode progress entry
	for _, progress := range progressList {
		err = m.db.Gorm().Delete(progress).Error
		if err != nil {
			m.logger.Error().Err(err).Msgf("continuity: Failed to delete progress for episode %d", progress.EpisodeNumber)
		}
	}

	return nil
}
