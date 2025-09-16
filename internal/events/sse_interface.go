package events

// SSEEventManagerInterface provides WebSocket-compatible interface for SSE
type SSEEventManagerInterface interface {
	// Core event methods
	SendEvent(t string, payload interface{})
	SendEventTo(clientId string, t string, payload interface{}, noLog ...bool)
	SendEventToUser(userID uint, t string, payload interface{})

	// Enhanced methods for media-aware broadcasting
	SendEventToUsersWithMedia(mediaID int, eventType string, payload interface{})

	// Subscription methods for plugin compatibility
	SubscribeToClientEvents(id string) *SSESubscriber
	SubscribeToClientNativePlayerEvents(id string) *SSESubscriber
	UnsubscribeFromClientEvents(id string)
}

// SSEEventManagerAdapter adapts SSEManager to WSEventManagerInterface
type SSEEventManagerAdapter struct {
	sseManager *SSEManager
}

// NewSSEEventManagerAdapter creates a new adapter
func NewSSEEventManagerAdapter(sseManager *SSEManager) *SSEEventManagerAdapter {
	return &SSEEventManagerAdapter{
		sseManager: sseManager,
	}
}

// SendEvent broadcasts to all connections
func (a *SSEEventManagerAdapter) SendEvent(t string, payload interface{}) {
	if a.sseManager != nil {
		a.sseManager.BroadcastEvent(t, payload)
	}
}

// SendEventTo sends to specific connection (by connection ID)
func (a *SSEEventManagerAdapter) SendEventTo(clientId string, t string, payload interface{}, noLog ...bool) {
	if a.sseManager != nil {
		a.sseManager.SendEventToConnection(clientId, t, payload)
	}
}

// SendEventToUser sends to specific user
func (a *SSEEventManagerAdapter) SendEventToUser(userID uint, t string, payload interface{}) {
	if a.sseManager != nil {
		a.sseManager.SendEventToUser(userID, t, payload)
	}
}

// SubscribeToClientEvents creates a subscription for client events
func (a *SSEEventManagerAdapter) SubscribeToClientEvents(id string) *ClientEventSubscriber {
	if a.sseManager != nil {
		_ = a.sseManager.SubscribeToEvents(id, "client-events")
		// Convert SSESubscriber to ClientEventSubscriber if needed
		// For now, return nil as this requires bidirectional communication
		// which SSE doesn't naturally support
		return nil
	}
	return nil
}

// SubscribeToClientNativePlayerEvents creates a subscription for native player events
func (a *SSEEventManagerAdapter) SubscribeToClientNativePlayerEvents(id string) *ClientEventSubscriber {
	if a.sseManager != nil {
		_ = a.sseManager.SubscribeToEvents(id, "native-player-events")
		// Convert SSESubscriber to ClientEventSubscriber if needed
		// For now, return nil as this requires bidirectional communication
		return nil
	}
	return nil
}

// UnsubscribeFromClientEvents removes a subscription
func (a *SSEEventManagerAdapter) UnsubscribeFromClientEvents(id string) {
	if a.sseManager != nil {
		a.sseManager.UnsubscribeFromEvents(id)
	}
}

// Additional methods for enhanced functionality

// SendEventToUsersWithMedia broadcasts to users with specific media
func (a *SSEEventManagerAdapter) SendEventToUsersWithMedia(mediaID int, eventType string, payload interface{}) {
	if a.sseManager != nil {
		a.sseManager.SendEventToUsersWithMedia(mediaID, eventType, payload)
	}
}

// GetConnectionByUserID finds connection ID for a user
func (a *SSEEventManagerAdapter) GetConnectionByUserID(userID uint) (string, bool) {
	if a.sseManager != nil {
		return a.sseManager.GetConnectionByUserID(userID)
	}
	return "", false
}