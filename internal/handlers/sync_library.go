package handlers

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

// LibrarySyncRequest represents a library synchronization request
type LibrarySyncRequest struct {
	MediaIDs []int `json:"mediaIds" validate:"required"`
}

// LibraryChangedRequest represents changes to user's library
type LibraryChangedRequest struct {
	AddedMediaIDs   []int `json:"addedMediaIds"`
	RemovedMediaIDs []int `json:"removedMediaIds"`
}

// HandleSyncUserLibrary
//
//	@summary Synchronize user's library subscriptions
//	@desc Updates user's media subscriptions for real-time LocalFile updates
//	@route /api/v1/sync/library/sync [POST]
//	@body LibrarySyncRequest
//	@returns bool
func (h *Handler) HandleSyncUserLibrary(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	var body LibrarySyncRequest
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	// Basic validation - ensure we have media IDs
	if len(body.MediaIDs) == 0 {
		return h.RespondWithError(c, errors.New("mediaIds is required"))
	}

	if h.App.SyncManager == nil {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusServiceUnavailable, "Sync system not available"))
	}

	err := h.App.SyncManager.SyncUserLibrary(user.ID, body.MediaIDs)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

// HandleLibraryChanged
//
//	@summary Handle user library changes
//	@desc Updates subscriptions when user's library changes (adds/removes anime)
//	@route /api/v1/sync/library/changed [POST]
//	@body LibraryChangedRequest
//	@returns bool
func (h *Handler) HandleLibraryChanged(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	var body LibraryChangedRequest
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	if h.App.SyncManager == nil {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusServiceUnavailable, "Sync system not available"))
	}

	err := h.App.SyncManager.HandleUserLibraryChanged(user.ID, body.AddedMediaIDs, body.RemovedMediaIDs)
	if err != nil {
		return h.RespondWithError(c, err)
	}

	return h.RespondWithData(c, true)
}

// HandleGetSyncStats
//
//	@summary Get sync system statistics
//	@desc Returns statistics about the sync system for debugging/monitoring
//	@route /api/v1/sync/stats [GET]
//	@returns map[string]interface{}
func (h *Handler) HandleGetSyncStats(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	if h.App.SyncManager == nil {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusServiceUnavailable, "Sync system not available"))
	}

	stats := h.App.SyncManager.GetStats()
	
	// Add user-specific stats
	enhancedWS := h.App.SyncManager.GetEnhancedWSEventManager()
	if enhancedWS != nil {
		userSubscriptions := enhancedWS.GetUserSubscriptions(user.ID)
		stats["userSubscriptions"] = len(userSubscriptions)
		stats["subscribedMedia"] = userSubscriptions
	}

	return h.RespondWithData(c, stats)
}

// WebSocketSessionRequest represents a WebSocket session registration
type WebSocketSessionRequest struct {
	SessionID string `json:"sessionId" validate:"required"`
}

// HandleRegisterWebSocketSession
//
//	@summary Register WebSocket session with user
//	@desc Associates a WebSocket session with the current user for targeted events
//	@route /api/v1/sync/websocket/register [POST]
//	@body WebSocketSessionRequest
//	@returns bool
func (h *Handler) HandleRegisterWebSocketSession(c echo.Context) error {
	user := h.getCurrentUser(c)
	if user == nil {
		return h.RespondWithError(c, errors.New("authentication required"))
	}

	var body WebSocketSessionRequest
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	// Basic validation
	if body.SessionID == "" {
		return h.RespondWithError(c, errors.New("sessionId is required"))
	}

	if h.App.SyncManager == nil {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusServiceUnavailable, "Sync system not available"))
	}

	h.App.SyncManager.RegisterUserSession(user.ID, body.SessionID)

	return h.RespondWithData(c, true)
}

// HandleUnregisterWebSocketSession
//
//	@summary Unregister WebSocket session
//	@desc Removes a WebSocket session from user association
//	@route /api/v1/sync/websocket/unregister [POST]
//	@body WebSocketSessionRequest
//	@returns bool
func (h *Handler) HandleUnregisterWebSocketSession(c echo.Context) error {
	var body WebSocketSessionRequest
	if err := c.Bind(&body); err != nil {
		return h.RespondWithError(c, err)
	}

	// Basic validation
	if body.SessionID == "" {
		return h.RespondWithError(c, errors.New("sessionId is required"))
	}

	if h.App.SyncManager == nil {
		return h.RespondWithError(c, echo.NewHTTPError(http.StatusServiceUnavailable, "Sync system not available"))
	}

	h.App.SyncManager.UnregisterUserSession(body.SessionID)

	return h.RespondWithData(c, true)
}