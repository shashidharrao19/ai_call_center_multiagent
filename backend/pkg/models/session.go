package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Session represents an active call session
type Session struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id,omitempty"`
	Conn        *websocket.Conn `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity"`
	Status      SessionStatus `json:"status"`
	AudioConfig AudioConfig   `json:"audio_config"`
	
	// Phase 2 optimization components
	BufferPool     interface{} `json:"-"` // *optimization.AudioBufferPool
	AudioBuffer    interface{} `json:"-"` // *optimization.LockFreeRingBuffer
	AudioProcessor interface{} `json:"-"` // *optimization.AudioProcessor
	BackpressureHandler interface{} `json:"-"` // *optimization.BackpressureHandler
}

// SessionStatus represents the current state of a session
type SessionStatus string

const (
	SessionStatusConnecting SessionStatus = "connecting"
	SessionStatusActive     SessionStatus = "active"
	SessionStatusPaused     SessionStatus = "paused"
	SessionStatusEnded      SessionStatus = "ended"
)

// AudioConfig represents audio configuration for a session
type AudioConfig struct {
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Format     string `json:"format"`
	ChunkSize  int    `json:"chunk_size"`
}

// Message represents a WebSocket message
type Message struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeAudio      MessageType = "audio"
	MessageTypeText       MessageType = "text"
	MessageTypeControl    MessageType = "control"
	MessageTypeError      MessageType = "error"
	MessageTypeHeartbeat  MessageType = "heartbeat"
	MessageTypeSession    MessageType = "session"
)

// AudioMessage represents audio data in a message
type AudioMessage struct {
	Data      []byte `json:"data"`
	Format    string `json:"format"`
	Timestamp int64  `json:"timestamp"`
}

// TextMessage represents text data in a message
type TextMessage struct {
	Text      string `json:"text"`
	Timestamp int64  `json:"timestamp"`
}

// ControlMessage represents control commands
type ControlMessage struct {
	Action string      `json:"action"`
	Params interface{} `json:"params,omitempty"`
}

// NewSession creates a new session with a unique ID
func NewSession(conn *websocket.Conn) *Session {
	return &Session{
		ID:        uuid.New().String(),
		Conn:      conn,
		CreatedAt: time.Now(),
		LastActivity: time.Now(),
		Status:    SessionStatusConnecting,
		AudioConfig: AudioConfig{
			SampleRate: 24000,
			Channels:   1,
			Format:     "PCM",
			ChunkSize:  1024,
		},
	}
}

// IsActive checks if the session is currently active
func (s *Session) IsActive() bool {
	return s.Status == SessionStatusActive
}

// UpdateActivity updates the last activity timestamp
func (s *Session) UpdateActivity() {
	s.LastActivity = time.Now()
}

// Close closes the WebSocket connection
func (s *Session) Close() error {
	if s.Conn != nil {
		return s.Conn.Close()
	}
	return nil
}

// MessageTypeFromString converts a string to MessageType
func MessageTypeFromString(s string) MessageType {
	switch s {
	case "audio":
		return MessageTypeAudio
	case "text":
		return MessageTypeText
	case "control":
		return MessageTypeControl
	case "error":
		return MessageTypeError
	case "heartbeat":
		return MessageTypeHeartbeat
	case "session":
		return MessageTypeSession
	default:
		return MessageTypeError
	}
}
