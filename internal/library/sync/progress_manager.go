package sync

import (
	"context"
	"seanime/internal/database/db"
	"seanime/internal/database/models"
	"seanime/internal/events"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ProgressManager handles per-user episode watch progress and resume functionality
type ProgressManager struct {
	db         *db.Database
	enhancedWS *events.EnhancedWSEventManager
	logger     *zerolog.Logger
	
	// Active playback sessions
	activeSessions map[string]*models.UserActivePlayback // SessionID -> ActivePlayback
	sessionsMu     sync.RWMutex
	
	// Auto-save intervals
	autoSaveInterval    time.Duration
	sessionCleanupInterval time.Duration
	
	// Context for cleanup routines
	ctx    context.Context
	cancel context.CancelFunc
}

// UserPlaybackState represents real-time playback state
type UserPlaybackState struct {
	UserID               uint      `json:"userId"`
	SessionID            string    `json:"sessionId"`
	
	// Current episode info
	EpisodeNumber        int       `json:"episodeNumber"`
	AniDbEpisode         string    `json:"aniDbEpisode"`
	MediaTitle           string    `json:"mediaTitle"`
	MediaCoverImage      string    `json:"mediaCoverImage"`
	MediaTotalEpisodes   int       `json:"mediaTotalEpisodes"`
	MediaId              int       `json:"mediaId"`
	
	// Progress info
	CurrentTimeSeconds   int       `json:"currentTimeSeconds"`
	DurationSeconds      int       `json:"durationSeconds"`
	CompletionPercentage float64   `json:"completionPercentage"`
	
	// Resume functionality
	HasResumePoint       bool      `json:"hasResumePoint"`
	ResumeTimeSeconds    int       `json:"resumeTimeSeconds"`
	
	// Playback state
	IsPlaying           bool      `json:"isPlaying"`
	PlaybackSpeed       float64   `json:"playbackSpeed"`
	LastUpdatedAt       time.Time `json:"lastUpdatedAt"`
	
	// File info
	LocalFilePath       string    `json:"localFilePath"`
	MediaPlayerUsed     string    `json:"mediaPlayerUsed"`
}

// ResumePoint represents an available resume point for an episode
type ResumePoint struct {
	MediaID           int       `json:"mediaId"`
	EpisodeNumber     int       `json:"episodeNumber"`
	ResumeTimeSeconds int       `json:"resumeTimeSeconds"`
	DurationSeconds   int       `json:"durationSeconds"`
	CompletionPercent float64   `json:"completionPercent"`
	LastWatchedAt     time.Time `json:"lastWatchedAt"`
	LocalFilePath     string    `json:"localFilePath"`
}

// NewProgressManager creates a new progress tracking manager
func NewProgressManager(database *db.Database, enhancedWS *events.EnhancedWSEventManager, logger *zerolog.Logger) *ProgressManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	pm := &ProgressManager{
		db:                     database,
		enhancedWS:            enhancedWS,
		logger:                logger,
		activeSessions:        make(map[string]*models.UserActivePlayback),
		autoSaveInterval:      30 * time.Second,  // Save progress every 30 seconds
		sessionCleanupInterval: 5 * time.Minute, // Clean up inactive sessions every 5 minutes
		ctx:                   ctx,
		cancel:                cancel,
	}
	
	// Start background routines
	pm.startAutoSave()
	pm.startSessionCleanup()
	
	return pm
}

// StartWatching initiates progress tracking for a user's playback session
func (pm *ProgressManager) StartWatching(userID uint, sessionID string, state *UserPlaybackState) error {
	pm.sessionsMu.Lock()
	defer pm.sessionsMu.Unlock()
	
	// Create active playback record
	activePlayback := &models.UserActivePlayback{
		UserID:            userID,
		MediaID:           state.MediaId,
		EpisodeNumber:     state.EpisodeNumber,
		CurrentTimeSeconds: state.CurrentTimeSeconds,
		PlaybackState:     "playing",
		LastUpdateAt:      time.Now(),
		SessionID:         sessionID,
	}
	
	// Save to database
	err := pm.db.Gorm().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		UpdateAll: true,
	}).Create(activePlayback).Error
	
	if err != nil {
		pm.logger.Error().Err(err).
			Uint("userId", userID).
			Str("sessionId", sessionID).
			Msg("progress-manager: Failed to create active playback")
		return err
	}
	
	// Add to active sessions
	pm.activeSessions[sessionID] = activePlayback
	
	// Check for existing resume point
	resumePoint, err := pm.GetResumePoint(userID, state.MediaId, state.EpisodeNumber)
	if err != nil && err != gorm.ErrRecordNotFound {
		pm.logger.Error().Err(err).Msg("progress-manager: Failed to check resume point")
	}
	
	// Broadcast start event
	startEvent := events.UserProgressUpdateEvent{
		UserID:            userID,
		MediaID:           state.MediaId,
		EpisodeNumber:     state.EpisodeNumber,
		WatchTimeSeconds:  state.CurrentTimeSeconds,
		DurationSeconds:   state.DurationSeconds,
		CompletionPercent: state.CompletionPercentage,
		IsCompleted:       false,
		SessionID:         sessionID,
		Timestamp:         time.Now(),
	}
	
	pm.enhancedWS.SendEventToUser(userID, events.EventUserStartedWatching, startEvent)
	
	// Send resume point if available
	if resumePoint != nil {
		pm.enhancedWS.SendEventToUser(userID, events.EventResumePointAvailable, resumePoint)
	}
	
	pm.logger.Debug().
		Uint("userId", userID).
		Str("sessionId", sessionID).
		Int("mediaId", state.MediaId).
		Int("episode", state.EpisodeNumber).
		Msg("progress-manager: Started watching")
	
	return nil
}

// UpdateProgress updates the current playback progress
func (pm *ProgressManager) UpdateProgress(sessionID string, state *UserPlaybackState) error {
	pm.sessionsMu.Lock()
	activeSession, exists := pm.activeSessions[sessionID]
	pm.sessionsMu.Unlock()
	
	if !exists {
		pm.logger.Warn().Str("sessionId", sessionID).Msg("progress-manager: Session not found for progress update")
		return nil
	}
	
	// Update active session
	pm.sessionsMu.Lock()
	activeSession.CurrentTimeSeconds = state.CurrentTimeSeconds
	activeSession.PlaybackState = func() string {
		if state.IsPlaying {
			return "playing"
		}
		return "paused"
	}()
	activeSession.LastUpdateAt = time.Now()
	pm.sessionsMu.Unlock()
	
	// Calculate completion percentage
	completionPercent := 0.0
	if state.DurationSeconds > 0 {
		completionPercent = (float64(state.CurrentTimeSeconds) / float64(state.DurationSeconds)) * 100.0
	}
	
	// Broadcast progress update
	progressEvent := events.UserProgressUpdateEvent{
		UserID:            state.UserID,
		MediaID:           state.MediaId,
		EpisodeNumber:     state.EpisodeNumber,
		WatchTimeSeconds:  state.CurrentTimeSeconds,
		DurationSeconds:   state.DurationSeconds,
		CompletionPercent: completionPercent,
		IsCompleted:       completionPercent >= 90.0, // Consider 90%+ as completed
		SessionID:         sessionID,
		Timestamp:         time.Now(),
	}
	
	pm.enhancedWS.SendEventToUser(state.UserID, events.EventProgressUpdated, progressEvent)
	
	// Check if episode should be marked as completed
	if completionPercent >= 90.0 {
		err := pm.markEpisodeCompleted(state.UserID, state.MediaId, state.EpisodeNumber, state)
		if err != nil {
			pm.logger.Error().Err(err).Msg("progress-manager: Failed to mark episode as completed")
		}
	}
	
	return nil
}

// PauseWatching pauses the current playback
func (pm *ProgressManager) PauseWatching(sessionID string, state *UserPlaybackState) error {
	pm.sessionsMu.Lock()
	activeSession, exists := pm.activeSessions[sessionID]
	if exists {
		activeSession.PlaybackState = "paused"
		activeSession.LastUpdateAt = time.Now()
	}
	pm.sessionsMu.Unlock()
	
	// Save current progress
	err := pm.SaveProgress(state.UserID, state)
	if err != nil {
		return err
	}
	
	// Broadcast pause event
	pauseEvent := events.UserProgressUpdateEvent{
		UserID:            state.UserID,
		MediaID:           state.MediaId,
		EpisodeNumber:     state.EpisodeNumber,
		WatchTimeSeconds:  state.CurrentTimeSeconds,
		DurationSeconds:   state.DurationSeconds,
		CompletionPercent: state.CompletionPercentage,
		IsCompleted:       false,
		SessionID:         sessionID,
		Timestamp:         time.Now(),
	}
	
	pm.enhancedWS.SendEventToUser(state.UserID, events.EventUserPausedWatching, pauseEvent)
	
	pm.logger.Debug().
		Uint("userId", state.UserID).
		Str("sessionId", sessionID).
		Msg("progress-manager: Paused watching")
	
	return nil
}

// StopWatching stops the current playback and saves final progress
func (pm *ProgressManager) StopWatching(sessionID string, state *UserPlaybackState) error {
	pm.sessionsMu.Lock()
	activeSession, exists := pm.activeSessions[sessionID]
	if exists {
		delete(pm.activeSessions, sessionID)
	}
	pm.sessionsMu.Unlock()
	
	// Save final progress
	err := pm.SaveProgress(state.UserID, state)
	if err != nil {
		return err
	}
	
	// Remove from database
	if exists {
		err = pm.db.Gorm().Delete(activeSession).Error
		if err != nil {
			pm.logger.Error().Err(err).Msg("progress-manager: Failed to delete active session")
		}
	}
	
	// Broadcast stop event
	stopEvent := events.UserProgressUpdateEvent{
		UserID:            state.UserID,
		MediaID:           state.MediaId,
		EpisodeNumber:     state.EpisodeNumber,
		WatchTimeSeconds:  state.CurrentTimeSeconds,
		DurationSeconds:   state.DurationSeconds,
		CompletionPercent: state.CompletionPercentage,
		IsCompleted:       state.CompletionPercentage >= 90.0,
		SessionID:         sessionID,
		Timestamp:         time.Now(),
	}
	
	pm.enhancedWS.SendEventToUser(state.UserID, events.EventUserStoppedWatching, stopEvent)
	
	pm.logger.Debug().
		Uint("userId", state.UserID).
		Str("sessionId", sessionID).
		Msg("progress-manager: Stopped watching")
	
	return nil
}

// SaveProgress saves the current progress to database
func (pm *ProgressManager) SaveProgress(userID uint, state *UserPlaybackState) error {
	completionPercent := 0.0
	if state.DurationSeconds > 0 {
		completionPercent = (float64(state.CurrentTimeSeconds) / float64(state.DurationSeconds)) * 100.0
	}
	
	progress := &models.UserEpisodeProgress{
		UserID:           userID,
		MediaID:          state.MediaId,
		EpisodeNumber:    state.EpisodeNumber,
		AniDBEpisode:     state.AniDbEpisode,
		WatchTimeSeconds: state.CurrentTimeSeconds,
		DurationSeconds:  state.DurationSeconds,
		CompletionPercent: completionPercent,
		IsCompleted:      completionPercent >= 90.0,
		LastWatchedAt:    time.Now(),
		LocalFilePath:    state.LocalFilePath,
		LocalFileHash:    "", // TODO: Generate from LocalFile manager
		MediaPlayerUsed:  state.MediaPlayerUsed,
		DeviceInfo:       state.SessionID, // Use session ID as device identifier for now
	}
	
	// Upsert progress
	err := pm.db.Gorm().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "media_id"}, {Name: "episode_number"}},
		UpdateAll: true,
	}).Create(progress).Error
	
	if err != nil {
		pm.logger.Error().Err(err).
			Uint("userId", userID).
			Int("mediaId", state.MediaId).
			Int("episode", state.EpisodeNumber).
			Msg("progress-manager: Failed to save progress")
		return err
	}
	
	// Broadcast progress saved event
	pm.enhancedWS.BroadcastProgressUpdate(userID, progress)
	
	return nil
}

// GetResumePoint retrieves the resume point for an episode
func (pm *ProgressManager) GetResumePoint(userID uint, mediaID int, episodeNumber int) (*ResumePoint, error) {
	var progress models.UserEpisodeProgress
	err := pm.db.Gorm().Where("user_id = ? AND media_id = ? AND episode_number = ?", 
		userID, mediaID, episodeNumber).First(&progress).Error
	
	if err != nil {
		return nil, err
	}
	
	// Only provide resume point if watched less than 90% and more than 30 seconds
	if progress.CompletionPercent < 90.0 && progress.WatchTimeSeconds > 30 {
		return &ResumePoint{
			MediaID:           mediaID,
			EpisodeNumber:     episodeNumber,
			ResumeTimeSeconds: progress.WatchTimeSeconds,
			DurationSeconds:   progress.DurationSeconds,
			CompletionPercent: progress.CompletionPercent,
			LastWatchedAt:     progress.LastWatchedAt,
			LocalFilePath:     progress.LocalFilePath,
		}, nil
	}
	
	return nil, nil // No resume point needed
}

// GetUserProgress retrieves all progress for a user's media
func (pm *ProgressManager) GetUserProgress(userID uint, mediaID int) ([]*models.UserEpisodeProgress, error) {
	var progress []*models.UserEpisodeProgress
	err := pm.db.Gorm().Where("user_id = ? AND media_id = ?", userID, mediaID).
		Order("episode_number ASC").Find(&progress).Error
	
	if err != nil {
		return nil, err
	}
	
	return progress, nil
}

// markEpisodeCompleted marks an episode as completed and handles related logic
func (pm *ProgressManager) markEpisodeCompleted(userID uint, mediaID int, episodeNumber int, state *UserPlaybackState) error {
	// Update progress as completed
	err := pm.db.Gorm().Model(&models.UserEpisodeProgress{}).
		Where("user_id = ? AND media_id = ? AND episode_number = ?", userID, mediaID, episodeNumber).
		Updates(map[string]interface{}{
			"is_completed":      true,
			"completion_percent": 100.0,
			"last_watched_at":   time.Now(),
		}).Error
	
	if err != nil {
		return err
	}
	
	// Broadcast completion event
	completionEvent := events.UserProgressUpdateEvent{
		UserID:            userID,
		MediaID:           mediaID,
		EpisodeNumber:     episodeNumber,
		WatchTimeSeconds:  state.CurrentTimeSeconds,
		DurationSeconds:   state.DurationSeconds,
		CompletionPercent: 100.0,
		IsCompleted:       true,
		SessionID:         state.SessionID,
		Timestamp:         time.Now(),
	}
	
	pm.enhancedWS.SendEventToUser(userID, events.EventUserCompletedEpisode, completionEvent)
	
	return nil
}

// startAutoSave starts the auto-save routine for active sessions
func (pm *ProgressManager) startAutoSave() {
	go func() {
		ticker := time.NewTicker(pm.autoSaveInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-pm.ctx.Done():
				return
			case <-ticker.C:
				pm.autoSaveActiveSessions()
			}
		}
	}()
	
	pm.logger.Info().
		Dur("interval", pm.autoSaveInterval).
		Msg("progress-manager: Auto-save started")
}

// startSessionCleanup starts the session cleanup routine
func (pm *ProgressManager) startSessionCleanup() {
	go func() {
		ticker := time.NewTicker(pm.sessionCleanupInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-pm.ctx.Done():
				return
			case <-ticker.C:
				pm.cleanupInactiveSessions()
			}
		}
	}()
	
	pm.logger.Info().
		Dur("interval", pm.sessionCleanupInterval).
		Msg("progress-manager: Session cleanup started")
}

// autoSaveActiveSessions saves progress for all active sessions
func (pm *ProgressManager) autoSaveActiveSessions() {
	pm.sessionsMu.RLock()
	sessions := make([]*models.UserActivePlayback, 0, len(pm.activeSessions))
	for _, session := range pm.activeSessions {
		sessions = append(sessions, session)
	}
	pm.sessionsMu.RUnlock()
	
	for _, session := range sessions {
		// Update database record
		err := pm.db.Gorm().Model(session).Updates(map[string]interface{}{
			"current_time_seconds": session.CurrentTimeSeconds,
			"playback_state":      session.PlaybackState,
			"last_update_at":      session.LastUpdateAt,
		}).Error
		
		if err != nil {
			pm.logger.Error().Err(err).
				Uint("userId", session.UserID).
				Str("sessionId", session.SessionID).
				Msg("progress-manager: Failed to auto-save session")
		}
	}
	
	if len(sessions) > 0 {
		pm.logger.Trace().
			Int("sessions", len(sessions)).
			Msg("progress-manager: Auto-saved active sessions")
	}
}

// cleanupInactiveSessions removes sessions that haven't been updated recently
func (pm *ProgressManager) cleanupInactiveSessions() {
	cutoff := time.Now().Add(-pm.sessionCleanupInterval * 2) // 2x cleanup interval
	
	pm.sessionsMu.Lock()
	var toRemove []string
	
	for sessionID, session := range pm.activeSessions {
		if session.LastUpdateAt.Before(cutoff) {
			toRemove = append(toRemove, sessionID)
		}
	}
	
	for _, sessionID := range toRemove {
		delete(pm.activeSessions, sessionID)
	}
	pm.sessionsMu.Unlock()
	
	// Remove from database
	if len(toRemove) > 0 {
		err := pm.db.Gorm().Where("last_update_at < ?", cutoff).Delete(&models.UserActivePlayback{}).Error
		if err != nil {
			pm.logger.Error().Err(err).Msg("progress-manager: Failed to cleanup inactive sessions from database")
		}
		
		pm.logger.Debug().
			Int("cleaned", len(toRemove)).
			Msg("progress-manager: Cleaned up inactive sessions")
	}
}

// Close shuts down the progress manager
func (pm *ProgressManager) Close() {
	pm.cancel()
	pm.logger.Info().Msg("progress-manager: Shutting down")
}