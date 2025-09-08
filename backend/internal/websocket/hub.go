package websocket

import (
	"ai-call-center/backend/pkg/models"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*models.Session]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *models.Session

	// Unregister requests from clients
	unregister chan *models.Session

	// Mutex for thread-safe operations
	mutex sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*models.Session]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *models.Session),
		unregister: make(chan *models.Session),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	ticker := time.NewTicker(30 * time.Second) // Heartbeat every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case session := <-h.register:
			h.mutex.Lock()
			h.clients[session] = true
			h.mutex.Unlock()
			logrus.Infof("Client registered: %s", session.ID)

		case session := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[session]; ok {
				delete(h.clients, session)
				close(session.Conn.CloseHandler())
				logrus.Infof("Client unregistered: %s", session.ID)
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for session := range h.clients {
				select {
				case session.Conn.WriteMessage(websocket.TextMessage, message):
				default:
					close(session.Conn.CloseHandler())
					delete(h.clients, session)
				}
			}
			h.mutex.RUnlock()

		case <-ticker.C:
			h.sendHeartbeat()
		}
	}
}

// RegisterClient registers a new client
func (h *Hub) RegisterClient(session *models.Session) {
	h.register <- session
}

// UnregisterClient unregisters a client
func (h *Hub) UnregisterClient(session *models.Session) {
	h.unregister <- session
}

// BroadcastMessage broadcasts a message to all clients
func (h *Hub) BroadcastMessage(message []byte) {
	h.broadcast <- message
}

// SendToSession sends a message to a specific session
func (h *Hub) SendToSession(sessionID string, message interface{}) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for session := range h.clients {
		if session.ID == sessionID {
			data, err := json.Marshal(message)
			if err != nil {
				return err
			}
			return session.Conn.WriteMessage(websocket.TextMessage, data)
		}
	}
	return nil
}

// GetSessionCount returns the number of active sessions
func (h *Hub) GetSessionCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// GetSessions returns all active sessions
func (h *Hub) GetSessions() []*models.Session {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	
	sessions := make([]*models.Session, 0, len(h.clients))
	for session := range h.clients {
		sessions = append(sessions, session)
	}
	return sessions
}

// sendHeartbeat sends heartbeat messages to all clients
func (h *Hub) sendHeartbeat() {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	heartbeat := models.Message{
		Type:      models.MessageTypeHeartbeat,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(heartbeat)
	if err != nil {
		logrus.Error("Failed to marshal heartbeat:", err)
		return
	}

	for session := range h.clients {
		select {
		case session.Conn.WriteMessage(websocket.TextMessage, data):
		default:
			// Connection is closed, will be cleaned up in next iteration
		}
	}
}

// CleanupInactiveSessions removes sessions that haven't been active
func (h *Hub) CleanupInactiveSessions(timeout time.Duration) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	now := time.Now()
	for session := range h.clients {
		if now.Sub(session.LastActivity) > timeout {
			logrus.Infof("Cleaning up inactive session: %s", session.ID)
			session.Close()
			delete(h.clients, session)
		}
	}
}
