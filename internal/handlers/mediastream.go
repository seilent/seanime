package handlers

import (
    "fmt"
    "net/http"
    "seanime/internal/database/models"
    "seanime/internal/mediastream"

    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
)

// (Unified mediastream settings endpoint removed in favor of split endpoints.)


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
			DirectPlayOnly: false,
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
	default:
		err = fmt.Errorf("stream type %s not implemented, only direct streaming is available", b.StreamType)
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
	case mediastream.StreamTypeDirect:
		err = h.App.MediastreamRepository.RequestPreloadDirectPlay(b.Path)
	default:
		err = fmt.Errorf("stream type %s not implemented, only direct streaming is available", b.StreamType)
	}
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

func (h *Handler) HandleMediastreamGetSubtitles(c echo.Context) error {
	client := h.getClientId(c)
	return h.App.MediastreamRepository.ServeEchoExtractedSubtitles(c, client)
}

func (h *Handler) HandleMediastreamGetAttachments(c echo.Context) error {
	client := h.getClientId(c)
	return h.App.MediastreamRepository.ServeEchoExtractedAttachments(c, client)
}

//
// Direct
//

func (h *Handler) HandleMediastreamDirectPlay(c echo.Context) error {
	client := h.getClientId(c)
	return h.App.MediastreamRepository.ServeEchoDirectPlay(c, client)
}


//
// Serve file
//

func (h *Handler) HandleMediastreamFile(c echo.Context) error {
	client := h.getClientId(c)
	fp := c.QueryParam("path")
	libraryPaths := h.App.GlobalSettings.GetLibrary().GetLibraryPaths()
	return h.App.MediastreamRepository.ServeEchoFile(c, fp, client, libraryPaths)
}

// getClientId retrieves the client ID from the request context or creates a new one
func (h *Handler) getClientId(c echo.Context) string {
	if clientId := c.Get("Seanime-Client-Id"); clientId != nil {
		if id, ok := clientId.(string); ok && id != "" {
			return id
		}
	}

	// Fallback: generate a new client ID if none exists
	newId := uuid.New().String()
	c.Set("Seanime-Client-Id", newId)
	c.SetCookie(&http.Cookie{
		Name:     "Seanime-Client-Id",
		Value:    newId,
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return newId
}
