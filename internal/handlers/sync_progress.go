package handlers

import (
	"errors"
	"net/http"
	"seanime/internal/library/sync"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// HandleGetUserProgress
//
//	@summary Get user's episode progress for a specific media
//	@desc Returns all episode progress for a user's media including resume points
//	@route /api/v1/sync/progress/:mediaId [GET]
//	@param mediaId path int true "Media ID"
//	@returns []*models.UserEpisodeProgress
func (h *Handler) HandleGetUserProgress(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	mediaID, err := strconv.Atoi(c.Param("mediaId"))
	if err != nil {
		return h.RespondWithError(c, err)
	}

	if h.App.SyncManager == nil {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusServiceUnavailable, "Sync system not available"))
	}

	progress, err := h.App.SyncManager.GetUserProgress(user.ID, mediaID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, progress)
}

// HandleGetResumePoint
//
//	@summary Get resume point for a specific episode
//	@desc Returns resume point if available for the specified episode
//	@route /api/v1/sync/progress/:mediaId/:episode/resume [GET]
//	@param mediaId path int true "Media ID"
//	@param episode path int true "Episode number"
//	@returns sync.ResumePoint
func (h *Handler) HandleGetResumePoint(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	mediaID, err := strconv.Atoi(c.Param("mediaId"))
	if err != nil {
		return h.RespondWithError(c, err)
	}

	episodeNumber, err := strconv.Atoi(c.Param("episode"))
	if err != nil {
		return h.RespondWithError(c, err)
	}

	if h.App.SyncManager == nil {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusServiceUnavailable, "Sync system not available"))
	}

	resumePoint, err := h.App.SyncManager.GetResumePoint(user.ID, mediaID, episodeNumber)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	if resumePoint == nil {
		return h.RespondWithData(c, nil)
	}

	return h.RespondWithData(c, resumePoint)
}

// ProgressUpdateRequest represents a progress update request
type ProgressUpdateRequest struct {
	MediaID              int     `json:"mediaId" validate:"required"`
	EpisodeNumber        int     `json:"episodeNumber" validate:"required"`
	AniDBEpisode         string  `json:"aniDbEpisode"`
	CurrentTimeSeconds   int     `json:"currentTimeSeconds" validate:"min=0"`
	DurationSeconds      int     `json:"durationSeconds" validate:"min=1"`
	IsPlaying           bool    `json:"isPlaying"`
	PlaybackSpeed       float64 `json:"playbackSpeed"`
	LocalFilePath       string  `json:"localFilePath"`
	MediaPlayerUsed     string  `json:"mediaPlayerUsed"`
	MediaTitle          string  `json:"mediaTitle"`
	MediaCoverImage     string  `json:"mediaCoverImage"`
	MediaTotalEpisodes  int     `json:"mediaTotalEpisodes"`
}

// HandleStartWatching
//
//	@summary Start progress tracking for playback
//	@desc Initiates progress tracking for a user's playback session
//	@route /api/v1/sync/progress/start [POST]
//	@body ProgressUpdateRequest
//	@returns bool
func (h *Handler) HandleStartWatching(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	var body ProgressUpdateRequest
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	// Basic validation will be handled within the sync manager

	if h.App.SyncManager == nil {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusServiceUnavailable, "Sync system not available"))
	}

	// Generate session ID from WebSocket connection or create one
	sessionID := c.Request().Header.Get("X-Session-ID")
	if sessionID == "" {
		sessionID = h.GenerateSessionID()
	}

	// Create playback state
	completionPercentage := 0.0
	if body.DurationSeconds > 0 {
		completionPercentage = (float64(body.CurrentTimeSeconds) / float64(body.DurationSeconds)) * 100.0
	}

	state := &sync.UserPlaybackState{
		UserID:               user.ID,
		SessionID:            sessionID,
		EpisodeNumber:        body.EpisodeNumber,
		AniDbEpisode:         body.AniDBEpisode,
		MediaTitle:           body.MediaTitle,
		MediaCoverImage:      body.MediaCoverImage,
		MediaTotalEpisodes:   body.MediaTotalEpisodes,
		MediaId:              body.MediaID,
		CurrentTimeSeconds:   body.CurrentTimeSeconds,
		DurationSeconds:      body.DurationSeconds,
		CompletionPercentage: completionPercentage,
		IsPlaying:           body.IsPlaying,
		PlaybackSpeed:       body.PlaybackSpeed,
		LocalFilePath:       body.LocalFilePath,
		MediaPlayerUsed:     body.MediaPlayerUsed,
	}

	err := h.App.SyncManager.StartWatching(user.ID, sessionID, state)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, map[string]interface{}{
		"success":   true,
		"sessionId": sessionID,
	})
}

// HandleUpdateProgress
//
//	@summary Update playback progress
//	@desc Updates the current playback progress for an active session
//	@route /api/v1/sync/progress/update [PUT]
//	@body ProgressUpdateRequest
//	@returns bool
func (h *Handler) HandleUpdateProgress(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	var body ProgressUpdateRequest
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	// Basic validation will be handled within the sync manager

	sessionID := c.Request().Header.Get("X-Session-ID")
	if sessionID == "" {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusBadRequest, "Session ID required"))
	}

	if h.App.SyncManager == nil {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusServiceUnavailable, "Sync system not available"))
	}

	// Create playback state
	completionPercentage := 0.0
	if body.DurationSeconds > 0 {
		completionPercentage = (float64(body.CurrentTimeSeconds) / float64(body.DurationSeconds)) * 100.0
	}

	state := &sync.UserPlaybackState{
		UserID:               user.ID,
		SessionID:            sessionID,
		EpisodeNumber:        body.EpisodeNumber,
		AniDbEpisode:         body.AniDBEpisode,
		MediaTitle:           body.MediaTitle,
		MediaCoverImage:      body.MediaCoverImage,
		MediaTotalEpisodes:   body.MediaTotalEpisodes,
		MediaId:              body.MediaID,
		CurrentTimeSeconds:   body.CurrentTimeSeconds,
		DurationSeconds:      body.DurationSeconds,
		CompletionPercentage: completionPercentage,
		IsPlaying:           body.IsPlaying,
		PlaybackSpeed:       body.PlaybackSpeed,
		LocalFilePath:       body.LocalFilePath,
		MediaPlayerUsed:     body.MediaPlayerUsed,
	}

	err := h.App.SyncManager.UpdateProgress(sessionID, state)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

// HandlePauseWatching
//
//	@summary Pause progress tracking
//	@desc Pauses the current playback and saves progress
//	@route /api/v1/sync/progress/pause [POST]
//	@body ProgressUpdateRequest
//	@returns bool
func (h *Handler) HandlePauseWatching(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	var body ProgressUpdateRequest
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	sessionID := c.Request().Header.Get("X-Session-ID")
	if sessionID == "" {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusBadRequest, "Session ID required"))
	}

	if h.App.SyncManager == nil {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusServiceUnavailable, "Sync system not available"))
	}

	completionPercentage := 0.0
	if body.DurationSeconds > 0 {
		completionPercentage = (float64(body.CurrentTimeSeconds) / float64(body.DurationSeconds)) * 100.0
	}

	state := &sync.UserPlaybackState{
		UserID:               user.ID,
		SessionID:            sessionID,
		EpisodeNumber:        body.EpisodeNumber,
		AniDbEpisode:         body.AniDBEpisode,
		MediaTitle:           body.MediaTitle,
		MediaCoverImage:      body.MediaCoverImage,
		MediaTotalEpisodes:   body.MediaTotalEpisodes,
		MediaId:              body.MediaID,
		CurrentTimeSeconds:   body.CurrentTimeSeconds,
		DurationSeconds:      body.DurationSeconds,
		CompletionPercentage: completionPercentage,
		IsPlaying:           false, // Always false for pause
		PlaybackSpeed:       body.PlaybackSpeed,
		LocalFilePath:       body.LocalFilePath,
		MediaPlayerUsed:     body.MediaPlayerUsed,
	}

	err := h.App.SyncManager.PauseWatching(sessionID, state)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

// HandleStopWatching
//
//	@summary Stop progress tracking
//	@desc Stops the current playback and saves final progress
//	@route /api/v1/sync/progress/stop [POST]
//	@body ProgressUpdateRequest
//	@returns bool
func (h *Handler) HandleStopWatching(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	var body ProgressUpdateRequest
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	sessionID := c.Request().Header.Get("X-Session-ID")
	if sessionID == "" {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusBadRequest, "Session ID required"))
	}

	if h.App.SyncManager == nil {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusServiceUnavailable, "Sync system not available"))
	}

	completionPercentage := 0.0
	if body.DurationSeconds > 0 {
		completionPercentage = (float64(body.CurrentTimeSeconds) / float64(body.DurationSeconds)) * 100.0
	}

	state := &sync.UserPlaybackState{
		UserID:               user.ID,
		SessionID:            sessionID,
		EpisodeNumber:        body.EpisodeNumber,
		AniDbEpisode:         body.AniDBEpisode,
		MediaTitle:           body.MediaTitle,
		MediaCoverImage:      body.MediaCoverImage,
		MediaTotalEpisodes:   body.MediaTotalEpisodes,
		MediaId:              body.MediaID,
		CurrentTimeSeconds:   body.CurrentTimeSeconds,
		DurationSeconds:      body.DurationSeconds,
		CompletionPercentage: completionPercentage,
		IsPlaying:           false,
		PlaybackSpeed:       body.PlaybackSpeed,
		LocalFilePath:       body.LocalFilePath,
		MediaPlayerUsed:     body.MediaPlayerUsed,
	}

	err := h.App.SyncManager.StopWatching(sessionID, state)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

// GenerateSessionID generates a unique session identifier
func (h *Handler) GenerateSessionID() string {
	// This is a placeholder - you might want to use a more sophisticated method
	return "session_" + strconv.FormatInt(h.getNowUnix(), 10)
}

func (h *Handler) getNowUnix() int64 {
	return time.Now().Unix()
}