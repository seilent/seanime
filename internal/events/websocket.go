package events

import (
	"seanime/internal/util"
	"seanime/internal/util/result"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

type WSEventManagerInterface interface {
	SendEvent(t string, payload interface{})
	SendEventTo(clientId string, t string, payload interface{}, noLog ...bool)
	SendEventToUser(userID uint, t string, payload interface{}) // New method for user-specific events
	SubscribeToClientEvents(id string) *ClientEventSubscriber
	SubscribeToClientNativePlayerEvents(id string) *ClientEventSubscriber
	UnsubscribeFromClientEvents(id string)
}

type GlobalWSEventManagerWrapper struct {
	WSEventManager WSEventManagerInterface
}

var GlobalWSEventManager *GlobalWSEventManagerWrapper

func (w *GlobalWSEventManagerWrapper) SendEvent(t string, payload interface{}) {
	if w.WSEventManager == nil {
		return
	}
	w.WSEventManager.SendEvent(t, payload)
}

func (w *GlobalWSEventManagerWrapper) SendEventTo(clientId string, t string, payload interface{}, noLog ...bool) {
	if w.WSEventManager == nil {
		return
	}
	w.WSEventManager.SendEventTo(clientId, t, payload, noLog...)
}

func (w *GlobalWSEventManagerWrapper) SendEventToUser(userID uint, t string, payload interface{}) {
	if w.WSEventManager == nil {
		return
	}
	w.WSEventManager.SendEventToUser(userID, t, payload)
}

type (
	// WSEventManager holds the websocket connection instance.
	// It is attached to the App instance, so it is available to other handlers.
	WSEventManager struct {
		Conns                              []*WSConn
		Logger                             *zerolog.Logger
		hasHadConnection                   bool
		mu                                 sync.Mutex
		eventMu                            sync.RWMutex
		clientEventSubscribers             *result.Map[string, *ClientEventSubscriber]
		clientNativePlayerEventSubscribers *result.Map[string, *ClientEventSubscriber]
	}

	ClientEventSubscriber struct {
		Channel chan *WebsocketClientEvent
		mu      sync.RWMutex
		closed  bool
	}

	WSConn struct {
		ID     string
		UserID uint // User ID associated with this WebSocket connection
		Conn   *websocket.Conn
	}

	WSEvent struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}
)

// NewWSEventManager creates a new WSEventManager instance for App.
func NewWSEventManager(logger *zerolog.Logger) *WSEventManager {
	ret := &WSEventManager{
		Logger:                             logger,
		Conns:                              make([]*WSConn, 0),
		clientEventSubscribers:             result.NewResultMap[string, *ClientEventSubscriber](),
		clientNativePlayerEventSubscribers: result.NewResultMap[string, *ClientEventSubscriber](),
	}
	GlobalWSEventManager = &GlobalWSEventManagerWrapper{
		WSEventManager: ret,
	}
	return ret
}

func (m *WSEventManager) AddConn(id string, userID uint, conn *websocket.Conn) {
	m.hasHadConnection = true
	m.Conns = append(m.Conns, &WSConn{
		ID:     id,
		UserID: userID,
		Conn:   conn,
	})
}

func (m *WSEventManager) RemoveConn(id string) {
	for i, conn := range m.Conns {
		if conn.ID == id {
			m.Conns = append(m.Conns[:i], m.Conns[i+1:]...)
			break
		}
	}
}

// SendEvent sends a websocket event to the client.
func (m *WSEventManager) SendEvent(t string, payload interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// If there's no connection, do nothing
	//if m.Conn == nil {
	//	return
	//}

	if t != PlaybackManagerProgressPlaybackState && payload == nil {
		m.Logger.Trace().Str("type", t).Msg("ws: Sending message")
	}

	for _, conn := range m.Conns {
		err := conn.Conn.WriteJSON(WSEvent{
			Type:    t,
			Payload: payload,
		})
		if err != nil {
			// Note: NaN error coming from [progress_tracking.go]
			//m.Logger.Err(err).Msg("ws: Failed to send message")
		}
		//m.Logger.Trace().Str("type", t).Msg("ws: Sent message")
	}

	//err := m.Conn.WriteJSON(WSEvent{
	//	Type:    t,
	//	Payload: payload,
	//})
	//if err != nil {
	//	m.Logger.Err(err).Msg("ws: Failed to send message")
	//}
	//m.Logger.Trace().Str("type", t).Msg("ws: Sent message")
}

// SendEventToUser sends a websocket event to connections from a specific user.
func (m *WSEventManager) SendEventToUser(userID uint, t string, payload interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t != PlaybackManagerProgressPlaybackState && payload == nil {
		m.Logger.Trace().Str("type", t).Uint("user_id", userID).Msg("ws: Sending user-specific message")
	}

	userConnCount := 0
	for _, conn := range m.Conns {
		if conn.UserID == userID {
			userConnCount++
			err := conn.Conn.WriteJSON(WSEvent{
				Type:    t,
				Payload: payload,
			})
			if err != nil {
				// Log the actual error to help with debugging
				m.Logger.Warn().Err(err).Str("type", t).Uint("user_id", userID).Msg("ws: Failed to send message")
			}
		}
	}

	m.Logger.Debug().Str("type", t).Uint("user_id", userID).Int("connections", userConnCount).Msg("ws: Event sent to user")
}

// SendEventTo sends a websocket event to the specified client.
func (m *WSEventManager) SendEventTo(clientId string, t string, payload interface{}, noLog ...bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, conn := range m.Conns {
		if conn.ID == clientId {
			if t != "pong" {
				if len(noLog) == 0 || !noLog[0] {
					truncated := spew.Sprint(payload)
					if len(truncated) > 500 {
						truncated = truncated[:500] + "..."
					}
					m.Logger.Trace().Str("to", clientId).Str("type", t).Str("payload", truncated).Msg("ws: Sending message")
				}
			}
			_ = conn.Conn.WriteJSON(WSEvent{
				Type:    t,
				Payload: payload,
			})
		}
	}
}

func (m *WSEventManager) SendStringTo(clientId string, s string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, conn := range m.Conns {
		if conn.ID == clientId {
			_ = conn.Conn.WriteMessage(websocket.TextMessage, []byte(s))
		}
	}
}

func (m *WSEventManager) OnClientEvent(event *WebsocketClientEvent) {
	m.eventMu.RLock()
	defer m.eventMu.RUnlock()

	onEvent := func(key string, subscriber *ClientEventSubscriber) bool {
		go func() {
			defer util.HandlePanicInModuleThen("events/OnClientEvent/clientNativePlayerEventSubscribers", func() {})
			subscriber.mu.RLock()
			defer subscriber.mu.RUnlock()
			if !subscriber.closed {
				select {
				case subscriber.Channel <- event:
				default:
					// Channel is blocked, skip sending
					m.Logger.Warn().Msg("ws: Client event channel is blocked, event dropped")
				}
			}
		}()
		return true
	}

	switch event.Type {
	case NativePlayerEventType:
		m.clientNativePlayerEventSubscribers.Range(onEvent)
	default:
		m.clientEventSubscribers.Range(onEvent)
	}
}

func (m *WSEventManager) SubscribeToClientEvents(id string) *ClientEventSubscriber {
	subscriber := &ClientEventSubscriber{
		Channel: make(chan *WebsocketClientEvent, 900),
	}
	m.clientEventSubscribers.Set(id, subscriber)
	return subscriber
}

func (m *WSEventManager) SubscribeToClientNativePlayerEvents(id string) *ClientEventSubscriber {
	subscriber := &ClientEventSubscriber{
		Channel: make(chan *WebsocketClientEvent, 100),
	}
	m.clientNativePlayerEventSubscribers.Set(id, subscriber)
	return subscriber
}

func (m *WSEventManager) UnsubscribeFromClientEvents(id string) {
	m.eventMu.Lock()
	defer m.eventMu.Unlock()
	defer func() {
		if r := recover(); r != nil {
			m.Logger.Warn().Msg("ws: Failed to unsubscribe from client events")
		}
	}()
	subscriber, ok := m.clientEventSubscribers.Get(id)
	if !ok {
		subscriber, ok = m.clientNativePlayerEventSubscribers.Get(id)
	}
	if ok {
		subscriber.mu.Lock()
		defer subscriber.mu.Unlock()
		subscriber.closed = true
		m.clientEventSubscribers.Delete(id)
		close(subscriber.Channel)
	}
}
