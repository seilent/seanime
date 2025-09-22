package handlers

import (
	"fmt"
	"github.com/labstack/echo/v4"
)

// HandleStartDefaultMediaPlayer
//
//	@summary launches the default media player (vlc or mpc-hc).
//	@route /api/v1/media-player/start [POST]
//	@returns bool
func (h *Handler) HandleStartDefaultMediaPlayer(c echo.Context) error {

	// Get current user
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, fmt.Errorf("user authentication required"))
	}

	// Retrieve user settings
	settings, err := h.App.Database.GetSettingsForUser(user.ID)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	switch settings.MediaPlayer.Default {
	case "vlc":
		err = h.App.MediaPlayer.VLC.Start()
		if err != nil {
			return h.RespondWithError(c, err)
		}
	case "mpc-hc":
		err = h.App.MediaPlayer.MpcHc.Start()
		if err != nil {
			return h.RespondWithError(c, err)
		}
	}

	return h.RespondWithData(c, true)
}
