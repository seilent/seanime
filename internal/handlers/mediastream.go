package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"seanime/internal/database/models"
	"seanime/internal/mediastream"

	"github.com/labstack/echo/v4"
)

// HandleGetTranscodingSettings
//
//	@summary get server transcoding settings (admin only).
//	@desc This returns the server-side transcoding settings from GlobalSettings.
//	@returns models.ServerTranscodingSettings
//	@route /api/v1/transcoding/settings [GET]
func (h *Handler) HandleGetTranscodingSettings(c echo.Context) error {
	globalSettings, err := h.App.Database.GetGlobalSettings()
	if err != nil {
		return h.RespondWithError(c, errors.New("global settings not found"))
	}

	if globalSettings.Transcoding == nil {
		// Return default settings if not set
		defaultSettings := &models.ServerTranscodingSettings{
			TranscodeEnabled: false,
			TranscodeHwAccel: "cpu",
			TranscodePreset:  "fast",
			TranscodeThreads: 4,
		}
		return h.RespondWithData(c, defaultSettings)
	}

	return h.RespondWithData(c, globalSettings.Transcoding)
}

// HandleSaveTranscodingSettings
//
//	@summary save server transcoding settings (admin only).
//	@desc This saves the server-side transcoding settings to GlobalSettings. Admin access required.
//	@returns models.ServerTranscodingSettings
//	@route /api/v1/transcoding/settings [PATCH]
func (h *Handler) HandleSaveTranscodingSettings(c echo.Context) error {
	// Check admin permissions
	user := h.getCurrentUser(c)
	if user == nil || !user.IsAdmin() {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Admin access required",
		})
	}

	type body struct {
		Settings models.ServerTranscodingSettings `json:"settings"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get current global settings
	globalSettings, err := h.App.Database.GetGlobalSettings()
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Update transcoding settings
	globalSettings.Transcoding = &b.Settings

	// Save global settings
	_, err = h.App.Database.UpsertGlobalSettings(globalSettings)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Refresh mediastream modules with new settings
	h.App.InitOrRefreshMediastreamSettings()

	return h.RespondWithData(c, globalSettings.Transcoding)
}

// HandleGetClientMediaSettings
//
//	@summary get user client media settings.
//	@desc This returns the client-side media playback settings for the current user.
//	@returns models.ClientMediaSettings
//	@route /api/v1/client-media/settings [GET]
func (h *Handler) HandleGetClientMediaSettings(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	userSettings, err := h.App.Database.GetSettingsForUser(user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	if userSettings.ClientMedia == nil {
		// Return default settings if not set
		defaultSettings := &models.ClientMediaSettings{
			DisableAutoSwitchToDirectPlay: false,
			DirectPlayOnly:                false,
		}
		return h.RespondWithData(c, defaultSettings)
	}

	return h.RespondWithData(c, userSettings.ClientMedia)
}

// HandleSaveClientMediaSettings
//
//	@summary save user client media settings.
//	@desc This saves the client-side media playback settings for the current user.
//	@returns models.ClientMediaSettings
//	@route /api/v1/client-media/settings [PATCH]
func (h *Handler) HandleSaveClientMediaSettings(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	type body struct {
		Settings models.ClientMediaSettings `json:"settings"`
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get current user settings
	userSettings, err := h.App.Database.GetSettingsForUser(user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	// Update client media settings
	userSettings.ClientMedia = &b.Settings

	// Save user settings
	err = h.App.Database.SaveSettingsForUser(user.ID, userSettings)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, userSettings.ClientMedia)
}

// HandleRequestMediastreamMediaContainer
//
//	@summary request media stream.
//	@desc This requests a media stream and returns the media container to start the playback.
//	@returns mediastream.MediaContainer
//	@route /api/v1/mediastream/request [POST]
func (h *Handler) HandleRequestMediastreamMediaContainer(c echo.Context) error {

	type body struct {
		Path             string                 `json:"path"`             // The path of the file.
		StreamType       mediastream.StreamType `json:"streamType"`       // The type of stream to request.
		AudioStreamIndex int                    `json:"audioStreamIndex"` // The audio stream index to use. (unused)
		ClientId         string                 `json:"clientId"`         // The session id
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	var mediaContainer *mediastream.MediaContainer
	var err error

	switch b.StreamType {
	case mediastream.StreamTypeDirect:
		mediaContainer, err = h.App.MediastreamRepository.RequestDirectPlay(b.Path, b.ClientId)
	case mediastream.StreamTypeTranscode:
		mediaContainer, err = h.App.MediastreamRepository.RequestTranscodeStream(b.Path, b.ClientId)
	case mediastream.StreamTypeOptimized:
		err = fmt.Errorf("stream type %s not implemented", b.StreamType)
		//mediaContainer, err = h.App.MediastreamRepository.RequestOptimizedStream(b.Path)
	default:
		err = fmt.Errorf("stream type %s not implemented", b.StreamType)
	}
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, mediaContainer)
}

// HandlePreloadMediastreamMediaContainer
//
//	@summary preloads media stream for playback.
//	@desc This preloads a media stream by extracting the media information and attachments.
//	@returns bool
//	@route /api/v1/mediastream/preload [POST]
func (h *Handler) HandlePreloadMediastreamMediaContainer(c echo.Context) error {

	type body struct {
		Path             string                 `json:"path"`             // The path of the file.
		StreamType       mediastream.StreamType `json:"streamType"`       // The type of stream to request.
		AudioStreamIndex int                    `json:"audioStreamIndex"` // The audio stream index to use.
	}

	var b body
	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	var err error

	switch b.StreamType {
	case mediastream.StreamTypeTranscode:
		err = h.App.MediastreamRepository.RequestPreloadTranscodeStream(b.Path)
	case mediastream.StreamTypeDirect:
		err = h.App.MediastreamRepository.RequestPreloadDirectPlay(b.Path)
	default:
		err = fmt.Errorf("stream type %s not implemented", b.StreamType)
	}
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

func (h *Handler) HandleMediastreamGetSubtitles(c echo.Context) error {
	return h.App.MediastreamRepository.ServeEchoExtractedSubtitles(c)
}

func (h *Handler) HandleMediastreamGetAttachments(c echo.Context) error {
	return h.App.MediastreamRepository.ServeEchoExtractedAttachments(c)
}

//
// Direct
//

func (h *Handler) HandleMediastreamDirectPlay(c echo.Context) error {
	client := "1"
	return h.App.MediastreamRepository.ServeEchoDirectPlay(c, client)
}

//
// Transcode
//

func (h *Handler) HandleMediastreamTranscode(c echo.Context) error {
	client := "1"
	return h.App.MediastreamRepository.ServeEchoTranscodeStream(c, client)
}

// HandleMediastreamShutdownTranscodeStream
//
//	@summary shuts down the transcode stream
//	@desc This requests the transcoder to shut down. It should be called when unmounting the player (playback is no longer needed).
//	@desc This will also send an events.MediastreamShutdownStream event.
//	@desc It will not return any error and is safe to call multiple times.
//	@returns bool
//	@route /api/v1/mediastream/shutdown-transcode [POST]
func (h *Handler) HandleMediastreamShutdownTranscodeStream(c echo.Context) error {
	client := "1"
	h.App.MediastreamRepository.ShutdownTranscodeStream(client)
	return h.RespondWithData(c, true)
}

//
// Serve file
//

func (h *Handler) HandleMediastreamFile(c echo.Context) error {
	client := "1"
	fp := c.QueryParam("path")
	libraryPaths := h.App.GlobalSettings.GetLibrary().GetLibraryPaths()
	return h.App.MediastreamRepository.ServeEchoFile(c, fp, client, libraryPaths)
}