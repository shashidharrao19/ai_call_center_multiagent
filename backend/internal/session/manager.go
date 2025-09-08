package session

import (
	"ai-call-center/backend/internal/config"
	"ai-call-center/backend/pkg/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Manager handles session management and AI engine communication
type Manager struct {
	config      *config.Config
	sessions    map[string]*models.Session
	mutex       sync.RWMutex
	aiClient    *http.Client
}

// NewManager creates a new session manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:   cfg,
		sessions: make(map[string]*models.Session),
		aiClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterSession registers a new session
func (m *Manager) RegisterSession(session *models.Session) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.sessions[session.ID] = session
	logrus.Infof("Session registered: %s", session.ID)
}

// UnregisterSession unregisters a session
func (m *Manager) UnregisterSession(sessionID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if session, exists := m.sessions[sessionID]; exists {
		session.Close()
		delete(m.sessions, sessionID)
		logrus.Infof("Session unregistered: %s", sessionID)
	}
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(sessionID string) (*models.Session, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	session, exists := m.sessions[sessionID]
	return session, exists
}

// ForwardToAI forwards a message to the AI engine
func (m *Manager) ForwardToAI(sessionID string, message models.Message) error {
	m.mutex.RLock()
	session, exists := m.sessions[sessionID]
	m.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Prepare AI request
	aiRequest := AIRequest{
		SessionID: sessionID,
		Message:   message,
		AudioConfig: session.AudioConfig,
	}

	// Send to AI engine
	response, err := m.sendToAI(aiRequest)
	if err != nil {
		return fmt.Errorf("failed to send to AI: %w", err)
	}

	// Handle AI response
	return m.handleAIResponse(session, response)
}

// sendToAI sends a request to the AI engine
func (m *Manager) sendToAI(request AIRequest) (*AIResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/process", m.config.AIEngineURL)
	resp, err := m.aiClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI engine returned status %d", resp.StatusCode)
	}

	var aiResponse AIResponse
	if err := json.NewDecoder(resp.Body).Decode(&aiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &aiResponse, nil
}

// handleAIResponse handles the response from the AI engine
func (m *Manager) handleAIResponse(session *models.Session, response *AIResponse) error {
	// Send response back to client
	message := models.Message{
		Type:      response.MessageType,
		SessionID: session.ID,
		Data:      response.Data,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := session.Conn.WriteMessage(1, data); err != nil { // 1 = TextMessage
		return fmt.Errorf("failed to send response: %w", err)
	}

	return nil
}

// GetActiveSessions returns all active sessions
func (m *Manager) GetActiveSessions() []*models.Session {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sessions := make([]*models.Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		if session.IsActive() {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// CleanupInactiveSessions removes sessions that haven't been active
func (m *Manager) CleanupInactiveSessions(timeout time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	for sessionID, session := range m.sessions {
		if now.Sub(session.LastActivity) > timeout {
			logrus.Infof("Cleaning up inactive session: %s", sessionID)
			session.Close()
			delete(m.sessions, sessionID)
		}
	}
}

// AIRequest represents a request to the AI engine
type AIRequest struct {
	SessionID   string           `json:"session_id"`
	Message     models.Message   `json:"message"`
	AudioConfig models.AudioConfig `json:"audio_config"`
}

// AIResponse represents a response from the AI engine
type AIResponse struct {
	SessionID   string      `json:"session_id"`
	MessageType models.MessageType `json:"message_type"`
	Data        interface{} `json:"data"`
	Timestamp   time.Time   `json:"timestamp"`
}
