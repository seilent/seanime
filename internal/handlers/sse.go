package handlers

import (
	"fmt"
	"net/http"

	"seanime/internal/events"

	"github.com/labstack/echo/v4"
)

// HandleSSEEvents handles SSE connection requests
func (h *Handler) HandleSSEEvents(c echo.Context) error {
	// Authenticate user for SSE connection
	user := h.getCurrentUser(c)
	if user == nil {
		// Fallback: validate session token directly (route may be registered before auth middleware)
		if token := h.getSessionToken(c); token != "" {
			if u, err := h.validateUserSession(token); err == nil && u != nil {
				c.Set("user", u)
				user = u
			}
		}
		if user == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required for SSE connection")
		}
	}

	// Check if client supports SSE
	w := c.Response().Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "SSE not supported")
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Get connection ID from query parameter or generate one
	connectionID := c.QueryParam("id")
	if connectionID == "" {
		connectionID = fmt.Sprintf("sse_user_%d_%d", user.ID, h.App.SSEManager.GetConnectionCount()+1)
	}

	// Create connection with user ID
	conn := &events.SSEConnection{
		ID:      connectionID,
		UserID:  user.ID,
		Writer:  w,
		Flusher: flusher,
		Request: c.Request(),
		Done:    make(chan bool),
		Logger:  h.App.Logger,
	}

	// Add connection to manager
	h.App.SSEManager.AddConnection(connectionID, conn)
	defer h.App.SSEManager.RemoveConnection(connectionID)

	h.App.Logger.Debug().Str("connection_id", connectionID).Uint("user_id", user.ID).Msg("sse: Client connected")

	// Send initial connection event to the user only
	h.App.SSEManager.SendEventToUser(user.ID, "connected", map[string]interface{}{
		"connection_id": connectionID,
		"user_id":       user.ID,
		"message":       "SSE connection established",
	})

	// Send sync-check event (no longer uses blob timestamps)
	syncCheckPayload := map[string]interface{}{
		"connection_id":   connectionID,
		"db_timestamp":    0,
		"cache_timestamp": 0,
		"cache_stale":     false,
	}

	h.App.SSEManager.SendEventToUser(user.ID, "sync-check", syncCheckPayload)

	h.App.Logger.Debug().
		Str("connection_id", connectionID).
		Uint("user_id", user.ID).
		Bool("cache_stale", false).
		Msg("sse: Sync check event sent")

	// Keep connection alive until client disconnects or context is done
	select {
	case <-c.Request().Context().Done():
		h.App.Logger.Debug().Str("connection_id", connectionID).Uint("user_id", user.ID).Msg("sse: Client disconnected (context done)")
	case <-conn.Done:
		h.App.Logger.Debug().Str("connection_id", connectionID).Uint("user_id", user.ID).Msg("sse: Client disconnected (connection closed)")
	}

	return nil
}
