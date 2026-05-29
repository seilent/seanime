package handlers

import (
	"errors"
	"seanime/internal/library/playbackmanager"

	"github.com/labstack/echo/v4"
)

// HandlePlaybackSyncCurrentProgress
//
//	@summary updates the AniList progress of the currently playing media.
//	@desc This is called after 'Update progress' is clicked when watching a media.
//	@desc This route returns the media ID of the currently playing media, so the client can refetch the media entry data.
//	@route /api/v1/playback-manager/sync-current-progress [POST]
//	@returns int
func (h *Handler) HandlePlaybackSyncCurrentProgress(c echo.Context) error {

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	err := h.App.PlaybackManager.SyncCurrentProgress()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	mId, _ := h.App.PlaybackManager.GetCurrentMediaID()

	return h.RespondWithData(c, mId)
}

// HandlePlaybackGetNextEpisode
//
//	@summary gets the next episode of the currently playing media.
//	@desc This is used by the client's autoplay feature
//	@route /api/v1/playback-manager/next-episode [GET]
//	@returns *anime.LocalFile
func (h *Handler) HandlePlaybackGetNextEpisode(c echo.Context) error {

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	lf := h.App.PlaybackManager.GetNextEpisode()
	return h.RespondWithData(c, lf)
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// HandlePlaybackStartManualTracking
//
//	@summary starts manual tracking of a media.
//	@desc Used for tracking progress of media that is not played through any integrated media player.
//	@desc This should only be used for trackable episodes (episodes that count towards progress).
//	@desc This returns 'true' if the tracking was successfully started.
//	@route /api/v1/playback-manager/manual-tracking/start [POST]
//	@returns bool
func (h *Handler) HandlePlaybackStartManualTracking(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	type body struct {
		MediaId       int    `json:"mediaId"`
		EpisodeNumber int    `json:"episodeNumber"`
		ClientId      string `json:"clientId"`
	}
	b := new(body)
	if err := c.Bind(b); err != nil {
		return h.RespondWithError(c, err)
	}

	err := h.App.PlaybackManager.StartManualProgressTracking(&playbackmanager.StartManualProgressTrackingOptions{
		ClientId:      b.ClientId,
		MediaId:       b.MediaId,
		EpisodeNumber: b.EpisodeNumber,
	})
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

// HandlePlaybackCancelManualTracking
//
//	@summary cancels manual tracking of a media.
//	@desc This will stop the server from expecting progress updates for the media.
//	@route /api/v1/playback-manager/manual-tracking/cancel [POST]
//	@returns bool
func (h *Handler) HandlePlaybackCancelManualTracking(c echo.Context) error {

	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	h.App.PlaybackManager.CancelManualProgressTracking()

	return h.RespondWithData(c, true)
}
