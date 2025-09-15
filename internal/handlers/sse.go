package handlers

import (
	"fmt"
	"net/http"

	"seanime/internal/database/db_bridge"
	"seanime/internal/events"

	"github.com/labstack/echo/v4"
)

// HandleSSEEvents handles SSE connection requests
func (h *Handler) HandleSSEEvents(c echo.Context) error {
	// Authenticate user for SSE connection
	user := h.getCurrentUser(c)
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Authentication required for SSE connection")
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

	// Send sync-check event to verify data freshness
	dbTimestamp, err := db_bridge.GetLocalFilesTimestamp(h.App.Database)
	cacheTimestamp := db_bridge.GetCacheTimestamp(h.App.Database)

	syncCheckPayload := map[string]interface{}{
		"connection_id": connectionID,
	}

	if err == nil && dbTimestamp != nil {
		syncCheckPayload["db_timestamp"] = dbTimestamp.Unix()
		syncCheckPayload["cache_timestamp"] = cacheTimestamp.Unix()
		syncCheckPayload["cache_stale"] = dbTimestamp.After(*cacheTimestamp)
	} else {
		// If error getting timestamp, assume cache might be stale
		syncCheckPayload["db_timestamp"] = 0
		syncCheckPayload["cache_timestamp"] = 0
		syncCheckPayload["cache_stale"] = true
	}

	h.App.SSEManager.SendEventToUser(user.ID, "sync-check", syncCheckPayload)

	h.App.Logger.Debug().
		Str("connection_id", connectionID).
		Uint("user_id", user.ID).
		Bool("cache_stale", syncCheckPayload["cache_stale"].(bool)).
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