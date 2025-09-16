package events

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog"
)

// SSEConnection represents an active SSE connection
type SSEConnection struct {
	ID       string
	UserID   uint // User ID associated with this connection
	Writer   http.ResponseWriter
	Flusher  http.Flusher
	Request  *http.Request
	Done     chan bool
	Logger   *zerolog.Logger
}

// SSESubscriber represents a subscription to events
type SSESubscriber struct {
	ConnectionID string
	EventFilter  string
	UserID       uint
}

// SSEManager manages active SSE connections
type SSEManager struct {
	connections map[string]*SSEConnection
	mutex       sync.RWMutex
	logger      *zerolog.Logger

	// Enhanced functionality for WebSocket compatibility
	subscribers map[string]*SSESubscriber  // ConnectionID -> Subscriber info
	db          interface{}               // Database for media subscriptions (optional)
}

// NewSSEManager creates a new SSE connection manager
func NewSSEManager(logger *zerolog.Logger) *SSEManager {
	return &SSEManager{
		connections: make(map[string]*SSEConnection),
		subscribers: make(map[string]*SSESubscriber),
		logger:      logger,
	}
}

// AddConnection adds a new SSE connection
func (m *SSEManager) AddConnection(id string, conn *SSEConnection) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Close existing connection with same ID if it exists
	if existingConn, exists := m.connections[id]; exists {
		close(existingConn.Done)
	}

	m.connections[id] = conn
	m.logger.Debug().Str("connection_id", id).Int("total_connections", len(m.connections)).Msg("sse: Connection added")
}

// RemoveConnection removes an SSE connection
func (m *SSEManager) RemoveConnection(id string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if conn, exists := m.connections[id]; exists {
		close(conn.Done)
		delete(m.connections, id)

		// Clean up any subscriptions for this connection
		if _, hasSubscription := m.subscribers[id]; hasSubscription {
			delete(m.subscribers, id)
		}

		m.logger.Debug().Str("connection_id", id).Int("total_connections", len(m.connections)).Msg("sse: Connection removed")
	}
}

// BroadcastEvent sends an event to all connected clients
func (m *SSEManager) BroadcastEvent(eventType string, data interface{}) {
	m.mutex.RLock()
	connections := make([]*SSEConnection, 0, len(m.connections))
	for _, conn := range m.connections {
		connections = append(connections, conn)
	}
	m.mutex.RUnlock()

	for _, conn := range connections {
		select {
		case <-conn.Done:
			// Connection is closed, skip
			continue
		default:
			m.sendEventToConnection(conn, eventType, data)
		}
	}

	m.logger.Debug().Str("event_type", eventType).Int("connections", len(connections)).Msg("sse: Event broadcasted")
}

// sendEventToConnection sends an event to a specific connection
func (m *SSEManager) sendEventToConnection(conn *SSEConnection, eventType string, data interface{}) {
	defer func() {
		if r := recover(); r != nil {
			conn.Logger.Error().Interface("panic", r).Msg("sse: Recovered from panic while sending event")
		}
	}()

	// Serialize data to JSON
	jsonData, err := json.Marshal(map[string]interface{}{
		"type":    eventType,
		"payload": data,
	})
	if err != nil {
		conn.Logger.Error().Err(err).Str("connection_id", conn.ID).Msg("sse: Failed to serialize event data")
		return
	}

	// Format SSE event
	eventData := fmt.Sprintf("data: %s\n\n", string(jsonData))

	// Write to connection
	if _, err := fmt.Fprint(conn.Writer, eventData); err != nil {
		conn.Logger.Error().Err(err).Str("connection_id", conn.ID).Msg("sse: Failed to write event")
		return
	}

	// Flush the response
	if conn.Flusher != nil {
		conn.Flusher.Flush()
	}
}

// GetConnectionCount returns the number of active connections
func (m *SSEManager) GetConnectionCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.connections)
}

// BroadcastEventToUser sends an event to connections from a specific user
func (m *SSEManager) BroadcastEventToUser(userID uint, eventType string, data interface{}) {
	m.mutex.RLock()
	connections := make([]*SSEConnection, 0)
	for _, conn := range m.connections {
		if conn.UserID == userID {
			connections = append(connections, conn)
		}
	}
	m.mutex.RUnlock()

	for _, conn := range connections {
		select {
		case <-conn.Done:
			// Connection is closed, skip
			continue
		default:
			m.sendEventToConnection(conn, eventType, data)
		}
	}

	m.logger.Debug().Str("event_type", eventType).Uint("user_id", userID).Int("connections", len(connections)).Msg("sse: Event broadcasted to user")
}

// BroadcastEventToUsers sends an event to connections from multiple specific users
func (m *SSEManager) BroadcastEventToUsers(userIDs []uint, eventType string, data interface{}) {
	userIDSet := make(map[uint]bool)
	for _, userID := range userIDs {
		userIDSet[userID] = true
	}

	m.mutex.RLock()
	connections := make([]*SSEConnection, 0)
	for _, conn := range m.connections {
		if userIDSet[conn.UserID] {
			connections = append(connections, conn)
		}
	}
	m.mutex.RUnlock()

	for _, conn := range connections {
		select {
		case <-conn.Done:
			// Connection is closed, skip
			continue
		default:
			m.sendEventToConnection(conn, eventType, data)
		}
	}

	m.logger.Debug().Str("event_type", eventType).Interface("user_ids", userIDs).Int("connections", len(connections)).Msg("sse: Event broadcasted to users")
}

// SendEventToSSE is a method that can be called from anywhere to send events via SSE
func (m *SSEManager) SendEventToSSE(eventType string, data interface{}) {
	if m == nil {
		return
	}
	m.BroadcastEvent(eventType, data)
}

// SendEventToUser sends event to specific user (convenience method)
func (m *SSEManager) SendEventToUser(userID uint, eventType string, data interface{}) {
	if m == nil {
		return
	}
	m.BroadcastEventToUser(userID, eventType, data)
}

// SendEventToConnection sends an event to a specific connection by ID
// This provides WebSocket SendEventTo compatibility
func (m *SSEManager) SendEventToConnection(connectionID string, eventType string, data interface{}) {
	if m == nil {
		return
	}

	m.mutex.RLock()
	conn, exists := m.connections[connectionID]
	m.mutex.RUnlock()

	if exists {
		select {
		case <-conn.Done:
			// Connection is closed, skip
			return
		default:
			m.sendEventToConnection(conn, eventType, data)
		}
		m.logger.Debug().Str("event_type", eventType).Str("connection_id", connectionID).Msg("sse: Event sent to connection")
	} else {
		m.logger.Warn().Str("connection_id", connectionID).Msg("sse: Connection not found for SendEventToConnection")
	}
}

// SendEventToUsersWithMedia sends an event to all users who have a specific media in their library
// This provides Enhanced WebSocket compatibility
func (m *SSEManager) SendEventToUsersWithMedia(mediaID int, eventType string, data interface{}) {
	if m == nil {
		return
	}

	// For now, broadcast to all users - can be enhanced with database integration later
	m.logger.Debug().Str("event_type", eventType).Int("media_id", mediaID).Msg("sse: Broadcasting event to users with media (fallback to all users)")
	m.BroadcastEvent(eventType, data)
}

// SubscribeToEvents creates a subscription for filtered events
// This provides WebSocket subscription compatibility
func (m *SSEManager) SubscribeToEvents(connectionID string, eventFilter string) *SSESubscriber {
	if m == nil {
		return nil
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Get user ID from connection
	var userID uint
	if conn, exists := m.connections[connectionID]; exists {
		userID = conn.UserID
	}

	subscriber := &SSESubscriber{
		ConnectionID: connectionID,
		EventFilter:  eventFilter,
		UserID:       userID,
	}

	m.subscribers[connectionID] = subscriber
	m.logger.Debug().Str("connection_id", connectionID).Str("filter", eventFilter).Msg("sse: Event subscription created")

	return subscriber
}

// UnsubscribeFromEvents removes a subscription
// This provides WebSocket unsubscription compatibility
func (m *SSEManager) UnsubscribeFromEvents(connectionID string) {
	if m == nil {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.subscribers[connectionID]; exists {
		delete(m.subscribers, connectionID)
		m.logger.Debug().Str("connection_id", connectionID).Msg("sse: Event subscription removed")
	}
}

// GetConnectionByUserID finds the first connection for a given user ID
// Utility method for user-to-connection mapping
func (m *SSEManager) GetConnectionByUserID(userID uint) (string, bool) {
	if m == nil {
		return "", false
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for id, conn := range m.connections {
		if conn.UserID == userID {
			return id, true
		}
	}
	return "", false
}