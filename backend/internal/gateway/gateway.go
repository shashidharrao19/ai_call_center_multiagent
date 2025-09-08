package gateway

import (
	"ai-call-center/backend/internal/config"
	"ai-call-center/backend/internal/optimization"
	"ai-call-center/backend/internal/session"
	"ai-call-center/backend/internal/websocket"
	"ai-call-center/backend/pkg/models"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Gateway handles WebSocket connections and routing
type Gateway struct {
	hub        *websocket.Hub
	config     *config.Config
	sessionMgr *session.Manager
	profiler   *profiling.Profiler
	optimizer  *optimization.OptimizationManager
}

// NewGateway creates a new gateway instance
func NewGateway(hub *websocket.Hub, cfg *config.Config, profiler *profiling.Profiler) *Gateway {
	return &Gateway{
		hub:        hub,
		config:     cfg,
		sessionMgr: session.NewManager(cfg),
		profiler:   profiler,
		optimizer:  optimization.NewOptimizationManager(cfg),
	}
}

// HandleWebSocket handles WebSocket connections
func (g *Gateway) HandleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := g.upgradeConnection(c)
	if err != nil {
		logrus.Errorf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Create new session
	sess := models.NewSession(conn)
	logrus.Infof("New WebSocket connection established: %s", sess.ID)

	// Create optimization components for session
	g.optimizer.CreateSessionOptimizations(sess.ID)
	defer g.optimizer.RemoveSessionOptimizations(sess.ID)

	// Assign session to audio processor
	audioProcessor := g.optimizer.AssignSessionToProcessor(sess.ID)
	if audioProcessor != nil {
		ringBuffer, _ := g.optimizer.GetSessionOptimizations(sess.ID)
		audioProcessor.SetRingBuffer(ringBuffer)
	}

	// Register session with hub
	g.hub.RegisterClient(sess)
	defer g.hub.UnregisterClient(sess)

	// Register session with session manager
	g.sessionMgr.RegisterSession(sess)
	defer g.sessionMgr.UnregisterSession(sess.ID)

	// Send session info to client
	g.sendSessionInfo(sess)

	// Handle messages
	g.handleMessages(sess)
}

// upgradeConnection upgrades HTTP connection to WebSocket
func (g *Gateway) upgradeConnection(c *gin.Context) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  g.config.ReadBufferSize,
		WriteBufferSize: g.config.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			// Allow connections from any origin in development
			// In production, implement proper origin checking
			return true
		},
	}

	return upgrader.Upgrade(c.Writer, c.Request, nil)
}

// sendSessionInfo sends session information to the client
func (g *Gateway) sendSessionInfo(sess *models.Session) {
	message := models.Message{
		Type:      models.MessageTypeSession,
		SessionID: sess.ID,
		Data: map[string]interface{}{
			"session_id":    sess.ID,
			"audio_config":  sess.AudioConfig,
			"status":        sess.Status,
			"created_at":    sess.CreatedAt,
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		logrus.Errorf("Failed to marshal session info: %v", err)
		return
	}

	if err := sess.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logrus.Errorf("Failed to send session info: %v", err)
	}
}

// handleMessages handles incoming WebSocket messages
func (g *Gateway) handleMessages(sess *models.Session) {
	for {
		messageType, data, err := sess.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Errorf("WebSocket error: %v", err)
			}
			break
		}

		// Update session activity
		sess.UpdateActivity()

		// Handle different message types
		switch messageType {
		case websocket.TextMessage:
			g.handleTextMessage(sess, data)
		case websocket.BinaryMessage:
			g.handleBinaryMessage(sess, data)
		case websocket.PingMessage:
			sess.Conn.WriteMessage(websocket.PongMessage, nil)
		case websocket.CloseMessage:
			logrus.Infof("Client %s closed connection", sess.ID)
			return
		}
	}
}

// handleTextMessage handles text messages from client
func (g *Gateway) handleTextMessage(sess *models.Session, data []byte) {
	var message models.Message
	if err := json.Unmarshal(data, &message); err != nil {
		logrus.Errorf("Failed to unmarshal message: %v", err)
		g.sendError(sess, "Invalid message format")
		return
	}

	// Set session ID if not provided
	message.SessionID = sess.ID
	message.Timestamp = time.Now()

	logrus.Debugf("Received message from %s: %s", sess.ID, message.Type)

	// Route message based on type
	switch message.Type {
	case models.MessageTypeAudio:
		g.handleAudioMessage(sess, message)
	case models.MessageTypeText:
		g.handleTextMessage(sess, message)
	case models.MessageTypeControl:
		g.handleControlMessage(sess, message)
	default:
		logrus.Warnf("Unknown message type: %s", message.Type)
	}
}

// handleBinaryMessage handles binary messages (audio data)
func (g *Gateway) handleBinaryMessage(sess *models.Session, data []byte) {
	// Get optimization components for session
	ringBuffer, backpressureHandler := g.optimizer.GetSessionOptimizations(sess.ID)
	if ringBuffer == nil || backpressureHandler == nil {
		logrus.Errorf("Optimization components not found for session: %s", sess.ID)
		g.sendError(sess, "Session optimization error")
		return
	}

	// Create audio chunk
	chunk := optimization.AudioChunk{
		Data:      data,
		Timestamp: time.Now().UnixNano(),
		SessionID: sess.ID,
		Format:    sess.AudioConfig.Format,
	}

	// Handle backpressure
	if err := backpressureHandler.HandleMessage(chunk); err != nil {
		logrus.Warnf("Backpressure handling failed for session %s: %v", sess.ID, err)
		// Continue processing even if backpressure fails
	}

	// Enqueue audio chunk to ring buffer
	if !ringBuffer.Enqueue(chunk) {
		logrus.Warnf("Ring buffer full for session: %s", sess.ID)
		g.sendError(sess, "Audio buffer full")
		return
	}

	logrus.Debugf("Audio chunk enqueued for session: %s (size: %d)", sess.ID, len(data))
}

// StartAudioProcessors starts all audio processors
func (g *Gateway) StartAudioProcessors() {
	g.optimizer.StartAudioProcessors()
}

// StopAudioProcessors stops all audio processors
func (g *Gateway) StopAudioProcessors() {
	g.optimizer.StopAudioProcessors()
}

// GetOptimizationMetrics returns optimization metrics
func (g *Gateway) GetOptimizationMetrics() *optimization.OptimizationMetrics {
	return g.optimizer.GetMetrics()
}

// handleAudioMessage processes audio messages
func (g *Gateway) handleAudioMessage(sess *models.Session, message models.Message) {
	// Forward audio to AI engine
	if err := g.sessionMgr.ForwardToAI(sess.ID, message); err != nil {
		logrus.Errorf("Failed to forward audio to AI: %v", err)
		g.sendError(sess, "Failed to process audio")
		return
	}

	logrus.Debugf("Audio message forwarded to AI for session %s", sess.ID)
}

// handleTextMessage processes text messages
func (g *Gateway) handleTextMessage(sess *models.Session, message models.Message) {
	// Forward text to AI engine
	if err := g.sessionMgr.ForwardToAI(sess.ID, message); err != nil {
		logrus.Errorf("Failed to forward text to AI: %v", err)
		g.sendError(sess, "Failed to process text")
		return
	}

	logrus.Debugf("Text message forwarded to AI for session %s", sess.ID)
}

// handleControlMessage processes control messages
func (g *Gateway) handleControlMessage(sess *models.Session, message models.Message) {
	controlData, ok := message.Data.(map[string]interface{})
	if !ok {
		g.sendError(sess, "Invalid control message format")
		return
	}

	action, ok := controlData["action"].(string)
	if !ok {
		g.sendError(sess, "Missing action in control message")
		return
	}

	switch action {
	case "start_call":
		sess.Status = models.SessionStatusActive
		logrus.Infof("Call started for session %s", sess.ID)
	case "end_call":
		sess.Status = models.SessionStatusEnded
		logrus.Infof("Call ended for session %s", sess.ID)
	case "pause_call":
		sess.Status = models.SessionStatusPaused
		logrus.Infof("Call paused for session %s", sess.ID)
	default:
		logrus.Warnf("Unknown control action: %s", action)
	}
}

// sendError sends an error message to the client
func (g *Gateway) sendError(sess *models.Session, errorMsg string) {
	message := models.Message{
		Type:      models.MessageTypeError,
		SessionID: sess.ID,
		Data: map[string]interface{}{
			"error": errorMsg,
		},
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		logrus.Errorf("Failed to marshal error message: %v", err)
		return
	}

	if err := sess.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logrus.Errorf("Failed to send error message: %v", err)
	}
}
