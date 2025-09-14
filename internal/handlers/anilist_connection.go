package handlers

import (
	"context"
	"errors"
	"seanime/internal/database/models"
	"seanime/internal/util"
	"time"

	"github.com/goccy/go-json"
	"github.com/labstack/echo/v4"
)

// HandleAnilistConnect
//
//	@summary connects the user's personal AniList account by saving the JWT token in the database.
//	@desc This is called when the JWT token is obtained from AniList after logging in with redirection on the client.
//	@desc It also fetches the Viewer data from AniList and saves it in the database for the current user.
//	@desc It creates a new handlers.Status and refreshes App modules.
//	@route /api/v1/anilist/connect [POST]
//	@returns handlers.Status
func (h *Handler) HandleAnilistConnect(c echo.Context) error {

	type body struct {
		Token string `json:"token"`
	}

	var b body

	if err := c.Bind(&b); err != nil {
		return h.RespondWithError(c, err)
	}

	// Get current user from context
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not found in context"))
	}

	// Set a new AniList client by passing to JWT token
	h.App.UpdateAnilistClientToken(b.Token)

	// Get viewer data from AniList
	getViewer, err := h.App.AnilistClient.GetViewer(context.Background())
	if err != nil {
		h.App.Logger.Error().Msg("Could not authenticate to AniList")
		return h.RespondWithError(c, err)
	}

	if len(getViewer.Viewer.Name) == 0 {
		return h.RespondWithError(c, errors.New("could not find AniList user"))
	}

	// Marshal viewer data
	bytes, err := json.Marshal(getViewer.Viewer)
	if err != nil {
		h.App.Logger.Err(err).Msg("anilist: could not marshal viewer data")
	}

	// Save account data in database with proper user context
	_, err = h.App.Database.UpsertAccountForUser(user.ID, &models.Account{
		BaseModel: models.BaseModel{
			UpdatedAt: time.Now(),
		},
		Username: getViewer.Viewer.Name,
		Token:    b.Token,
		Viewer:   bytes,
	})

	if err != nil {
		return h.RespondWithError(c, err)
	}

	h.App.Logger.Info().Str("user", user.Username).Str("anilist_user", getViewer.Viewer.Name).Msg("app: Connected to AniList")

	// Note: No longer updating global platform - user-specific platforms are created per request

	// Create a new status
	status := h.NewStatus(c)

	h.App.InitOrRefreshAnilistData()

	h.App.InitOrRefreshModules()

	go func() {
		defer util.HandlePanicThen(func() {})
	}()

	// Return new status
	return h.RespondWithData(c, status)

}

// HandleAnilistDisconnect
//
//	@summary disconnects the user's personal AniList account by removing JWT token from the database.
//	@desc It removes JWT token and Viewer data from the database for the current user.
//	@desc It creates a new handlers.Status and refreshes App modules.
//	@route /api/v1/anilist/disconnect [POST]
//	@returns handlers.Status
func (h *Handler) HandleAnilistDisconnect(c echo.Context) error {

	// Get current user from context
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not found in context"))
	}

	// Note: In multi-user system, don't update global client/platform when individual user disconnects

	// Clear account data in database for the current user
	_, err := h.App.Database.UpsertAccountForUser(user.ID, &models.Account{
		BaseModel: models.BaseModel{
			UpdatedAt: time.Now(),
		},
		Username: "",
		Token:    "",
		Viewer:   nil,
	})

	if err != nil {
		return h.RespondWithError(c, err)
	}

	h.App.Logger.Info().Str("user", user.Username).Msg("Disconnected from AniList")

	status := h.NewStatus(c)

	h.App.InitOrRefreshModules()

	h.App.InitOrRefreshAnilistData()

	return h.RespondWithData(c, status)
}

// HandleAnilistConnectionStatus
//
//	@summary returns the AniList connection status for the current user.
//	@desc This checks if the current user has an active AniList connection.
//	@route /api/v1/anilist/status [GET]
//	@returns object with connection status and user info
func (h *Handler) HandleAnilistConnectionStatus(c echo.Context) error {

	// Get current user from context
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("user not found in context"))
	}

	// Get account data for the current user
	account, err := h.App.Database.GetAccountForUser(user.ID)
	if err != nil {
		// No account found - user is not connected to AniList
		return h.RespondWithData(c, map[string]interface{}{
			"connected":     false,
			"anilistUser":   nil,
			"connectionDate": nil,
		})
	}

	// Check if account has valid token
	connected := account.Token != ""
	var anilistUser interface{} = nil
	
	if connected && account.Viewer != nil {
		// Unmarshal viewer data
		var viewer interface{}
		if err := json.Unmarshal(account.Viewer, &viewer); err == nil {
			anilistUser = viewer
		}
	}

	return h.RespondWithData(c, map[string]interface{}{
		"connected":      connected,
		"anilistUser":    anilistUser,
		"anilistUsername": account.Username,
		"connectionDate": account.UpdatedAt,
	})
}
