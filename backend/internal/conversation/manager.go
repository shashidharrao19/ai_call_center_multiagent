package conversation

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"ai-call-center/backend/internal/gemini"
	"ai-call-center/backend/internal/mcp"
	"ai-call-center/backend/internal/profiling"
)

// ConversationSession represents an active conversation session
type ConversationSession struct {
	SessionID    string                 `json:"session_id"`
	CreatedAt    time.Time              `json:"created_at"`
	LastActivity time.Time              `json:"last_activity"`
	Status       string                 `json:"status"`
	Context      map[string]interface{} `json:"context"`
	CustomerID   *string                `json:"customer_id,omitempty"`
}

// ProcessRequest represents a request to process a message
type ProcessRequest struct {
	SessionID   string                 `json:"session_id"`
	MessageType string                 `json:"message_type"`
	Data        map[string]interface{} `json:"data"`
	AudioConfig map[string]interface{} `json:"audio_config"`
}

// ProcessResponse represents a response from processing a message
type ProcessResponse struct {
	SessionID   string                 `json:"session_id"`
	MessageType string                 `json:"message_type"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   string                 `json:"timestamp"`
}

// Manager manages conversation sessions and coordinates between Gemini and MCP
type Manager struct {
	geminiClient        *gemini.Client
	mcpClient          *mcp.Client
	profiler           *profiling.Profiler
	sessions           map[string]*ConversationSession
	responseHandlers   map[string]chan ProcessResponse
	mu                 sync.RWMutex
	mcpKeywords        []string
}

// NewManager creates a new conversation manager
func NewManager(geminiClient *gemini.Client, mcpClient *mcp.Client, profiler *profiling.Profiler) *Manager {
	manager := &Manager{
		geminiClient:      geminiClient,
		mcpClient:        mcpClient,
		profiler:         profiler,
		sessions:         make(map[string]*ConversationSession),
		responseHandlers: make(map[string]chan ProcessResponse),
		mcpKeywords: []string{
			"customer data", "look up customer", "get customer info",
			"search knowledge", "find information", "create ticket",
			"update customer", "customer history",
		},
	}

	// Start listening for Gemini responses
	go manager.listenForGeminiResponses()

	return manager
}

// ProcessMessage processes incoming message from Go backend
func (m *Manager) ProcessMessage(req ProcessRequest) (*ProcessResponse, error) {
	// Get or create session
	session := m.getOrCreateSession(req.SessionID)
	
	// Update session activity
	session.LastActivity = time.Now()
	
	// Set audio config if provided
	if req.AudioConfig != nil {
		sampleRate := 24000
		channels := 1
		format := "PCM"
		
		if sr, ok := req.AudioConfig["sample_rate"].(float64); ok {
			sampleRate = int(sr)
		}
		if ch, ok := req.AudioConfig["channels"].(float64); ok {
			channels = int(ch)
		}
		if f, ok := req.AudioConfig["format"].(string); ok {
			format = f
		}
		
		m.geminiClient.SetAudioConfig(sampleRate, channels, format)
	}
	
	// Process based on message type
	switch req.MessageType {
	case "audio":
		return m.processAudioMessage(session, req.Data)
	case "text":
		return m.processTextMessage(session, req.Data)
	default:
		log.Printf("Unknown message type: %s", req.MessageType)
		return m.createErrorResponse("Unknown message type"), nil
	}
}

// getOrCreateSession gets existing session or creates new one
func (m *Manager) getOrCreateSession(sessionID string) *ConversationSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if session, exists := m.sessions[sessionID]; exists {
		return session
	}
	
	session := &ConversationSession{
		SessionID:    sessionID,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		Status:       "active",
		Context:      make(map[string]interface{}),
	}
	
	m.sessions[sessionID] = session
	m.responseHandlers[sessionID] = make(chan ProcessResponse, 10)
	
	log.Printf("Created new conversation session: %s", sessionID)
	return session
}

// processAudioMessage processes audio message
func (m *Manager) processAudioMessage(session *ConversationSession, data map[string]interface{}) (*ProcessResponse, error) {
	audioData, ok := data["data"]
	if !ok {
		return m.createErrorResponse("No audio data provided"), nil
	}
	
	var audioBytes []byte
	var err error
	
	// Convert base64 audio data to bytes if needed
	if audioStr, ok := audioData.(string); ok {
		audioBytes, err = base64.StdEncoding.DecodeString(audioStr)
		if err != nil {
			return m.createErrorResponse("Invalid base64 audio data"), nil
		}
	} else if audioBytes, ok = audioData.([]byte); !ok {
		return m.createErrorResponse("Invalid audio data format"), nil
	}
	
	// Send audio to Gemini
	if err := m.geminiClient.SendAudioInput(audioBytes, session.SessionID); err != nil {
		return m.createErrorResponse(fmt.Sprintf("Failed to send audio: %v", err)), nil
	}
	
	// Wait for response (with timeout)
	select {
	case response := <-m.responseHandlers[session.SessionID]:
		return &response, nil
	case <-time.After(30 * time.Second):
		return m.createErrorResponse("Response timeout"), nil
	}
}

// processTextMessage processes text message
func (m *Manager) processTextMessage(session *ConversationSession, data map[string]interface{}) (*ProcessResponse, error) {
	text, ok := data["text"].(string)
	if !ok || text == "" {
		return m.createErrorResponse("No text provided"), nil
	}
	
	// Check if text contains function call requests
	if m.shouldCallMCPFunction(text) {
		return m.handleMCPFunctionCall(text, session)
	}
	
	// Send text to Gemini
	if err := m.geminiClient.SendTextInput(text, session.SessionID); err != nil {
		return m.createErrorResponse(fmt.Sprintf("Failed to send text: %v", err)), nil
	}
	
	// Wait for response
	select {
	case response := <-m.responseHandlers[session.SessionID]:
		return &response, nil
	case <-time.After(30 * time.Second):
		return m.createErrorResponse("Response timeout"), nil
	}
}

// shouldCallMCPFunction determines if text should trigger MCP function call
func (m *Manager) shouldCallMCPFunction(text string) bool {
	textLower := fmt.Sprintf("%s", text)
	for _, keyword := range m.mcpKeywords {
		if regexp.MustCompile(fmt.Sprintf(`(?i)%s`, keyword)).MatchString(textLower) {
			return true
		}
	}
	return false
}

// handleMCPFunctionCall handles MCP function calling
func (m *Manager) handleMCPFunctionCall(text string, session *ConversationSession) (*ProcessResponse, error) {
	// Determine which function to call based on text
	functionName, parameters := m.parseFunctionCall(text, session)
	
	if functionName == "" {
		// No function call needed, send to Gemini
		if err := m.geminiClient.SendTextInput(text, session.SessionID); err != nil {
			return m.createErrorResponse(fmt.Sprintf("Failed to send text: %v", err)), nil
		}
		
		// Wait for response
		select {
		case response := <-m.responseHandlers[session.SessionID]:
			return &response, nil
		case <-time.After(30 * time.Second):
			return m.createErrorResponse("Response timeout"), nil
		}
	}
	
	// Call MCP function
	result, err := m.mcpClient.CallFunction(functionName, parameters)
	if err != nil {
		return m.createErrorResponse(fmt.Sprintf("MCP function call failed: %v", err)), nil
	}
	
	// Create response with function result
	responseText := m.formatFunctionResponse(functionName, result)
	
	return &ProcessResponse{
		SessionID:   session.SessionID,
		MessageType: "text",
		Data: map[string]interface{}{
			"text":           responseText,
			"function_called": functionName,
			"function_result": result,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// parseFunctionCall parses text to determine function call and parameters
func (m *Manager) parseFunctionCall(text string, session *ConversationSession) (string, map[string]interface{}) {
	textLower := fmt.Sprintf("%s", text)
	
	// Simple parsing - in production, use more sophisticated NLP
	if regexp.MustCompile(`(?i)(customer data|look up customer)`).MatchString(textLower) {
		// Extract customer ID from text
		customerID := m.extractCustomerID(text)
		if customerID != "" {
			session.CustomerID = &customerID
			return "get_customer_data", map[string]interface{}{"customer_id": customerID}
		}
	} else if regexp.MustCompile(`(?i)(search.*knowledge|search.*information)`).MatchString(textLower) {
		return "search_knowledge_base", map[string]interface{}{"query": text}
	} else if regexp.MustCompile(`(?i)(create ticket|support ticket)`).MatchString(textLower) {
		return "create_ticket", map[string]interface{}{
			"description": text,
			"customer_id": session.CustomerID,
		}
	}
	
	return "", nil
}

// extractCustomerID extracts customer ID from text
func (m *Manager) extractCustomerID(text string) string {
	patterns := []string{
		`(?i)customer\s+(\d+)`,
		`(?i)id:\s*(\d+)`,
		`(?i)customer\s+id:\s*(\d+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

// formatFunctionResponse formats MCP function result into natural language response
func (m *Manager) formatFunctionResponse(functionName string, result map[string]interface{}) string {
	if errorMsg, hasError := result["error"]; hasError {
		return fmt.Sprintf("I encountered an error while %s: %v", functionName, errorMsg)
	}
	
	// Format based on function type
	switch functionName {
	case "get_customer_data":
		if customerData, ok := result["data"].(map[string]interface{}); ok {
			jsonData, _ := json.MarshalIndent(customerData, "", "  ")
			return fmt.Sprintf("Here's the customer information: %s", string(jsonData))
		}
		return "Customer data retrieved successfully."
		
	case "search_knowledge_base":
		if searchResults, ok := result["results"].([]interface{}); ok {
			if len(searchResults) > 0 {
				if firstResult, ok := searchResults[0].(map[string]interface{}); ok {
					if title, ok := firstResult["title"].(string); ok {
						return fmt.Sprintf("I found %d relevant articles. Here's the most relevant one: %s", len(searchResults), title)
					}
				}
			}
			return "I couldn't find any relevant information in the knowledge base."
		}
		return "Search completed successfully."
		
	case "create_ticket":
		if ticketID, ok := result["ticket_id"].(string); ok {
			return fmt.Sprintf("I've created a support ticket for you. Ticket ID: %s", ticketID)
		}
		return "Support ticket created successfully."
	}
	
	return fmt.Sprintf("Function %s completed successfully.", functionName)
}

// listenForGeminiResponses listens for responses from Gemini and routes to appropriate sessions
func (m *Manager) listenForGeminiResponses() {
	// Set up response handler
	handler := func(message map[string]interface{}) error {
		return m.handleGeminiResponse(message)
	}
	
	// Start listening
	if err := m.geminiClient.ListenForResponses(handler); err != nil {
		log.Printf("Error listening for Gemini responses: %v", err)
	}
}

// handleGeminiResponse handles response from Gemini Live API
func (m *Manager) handleGeminiResponse(message map[string]interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if len(m.sessions) == 0 {
		return nil
	}
	
	// Get the most recent session
	var latestSession *ConversationSession
	for _, session := range m.sessions {
		if latestSession == nil || session.LastActivity.After(latestSession.LastActivity) {
			latestSession = session
		}
	}
	
	if latestSession == nil {
		return nil
	}
	
	// Check for audio output
	if audioData, err := m.geminiClient.DecodeAudioOutput(message); err == nil && len(audioData) > 0 {
		response := ProcessResponse{
			SessionID:   latestSession.SessionID,
			MessageType: "audio",
			Data: map[string]interface{}{
				"data":   audioData,
				"format": "PCM",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		
		select {
		case m.responseHandlers[latestSession.SessionID] <- response:
		default:
			log.Printf("Response channel full for session %s", latestSession.SessionID)
		}
		return nil
	}
	
	// Check for text output
	if textData, err := m.geminiClient.DecodeTextOutput(message); err == nil && textData != "" {
		response := ProcessResponse{
			SessionID:   latestSession.SessionID,
			MessageType: "text",
			Data: map[string]interface{}{
				"text": textData,
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		
		select {
		case m.responseHandlers[latestSession.SessionID] <- response:
		default:
			log.Printf("Response channel full for session %s", latestSession.SessionID)
		}
		return nil
	}
	
	// Check for interruption
	if m.geminiClient.IsInterrupted(message) {
		log.Printf("User interrupted response for session %s", latestSession.SessionID)
		// Clear any pending responses
		select {
		case <-m.responseHandlers[latestSession.SessionID]:
		default:
		}
	}
	
	return nil
}

// createErrorResponse creates error response
func (m *Manager) createErrorResponse(errorMessage string) *ProcessResponse {
	return &ProcessResponse{
		SessionID:   "",
		MessageType: "error",
		Data: map[string]interface{}{
			"error": errorMessage,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// GetActiveSessions returns list of active sessions
func (m *Manager) GetActiveSessions() []map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var activeSessions []map[string]interface{}
	for _, session := range m.sessions {
		if session.Status == "active" {
			sessionData := map[string]interface{}{
				"session_id":    session.SessionID,
				"created_at":    session.CreatedAt.Format(time.RFC3339),
				"last_activity": session.LastActivity.Format(time.RFC3339),
				"status":        session.Status,
			}
			if session.CustomerID != nil {
				sessionData["customer_id"] = *session.CustomerID
			}
			activeSessions = append(activeSessions, sessionData)
		}
	}
	
	return activeSessions
}

// EndSession ends a conversation session
func (m *Manager) EndSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if session, exists := m.sessions[sessionID]; exists {
		session.Status = "ended"
		// Clean up response channel
		if ch, exists := m.responseHandlers[sessionID]; exists {
			close(ch)
			delete(m.responseHandlers, sessionID)
		}
		log.Printf("Session %s ended", sessionID)
	}
	
	return nil
}
