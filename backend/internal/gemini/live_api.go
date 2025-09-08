package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// AudioConfig represents audio configuration for the Live API
type AudioConfig struct {
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
	Format     string `json:"format"`
}

// Client represents a Gemini Live API client
type Client struct {
	apiKey      string
	model       string
	client      *genai.Client
	live        *genai.Live
	session     *genai.Session
	audioConfig AudioConfig
	isConnected bool
	mu          sync.RWMutex
	responseHandler func(map[string]interface{}) error
}

// NewClient creates a new Gemini Live API client
func NewClient(apiKey, model string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Client{
		apiKey: apiKey,
		model:  model,
		client: client,
		audioConfig: AudioConfig{
			SampleRate: 24000,
			Channels:   1,
			Format:     "PCM",
		},
	}, nil
}

// Initialize initializes the Gemini Live API connection
func (c *Client) Initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx := context.Background()
	
	// Create Live API instance
	live := c.client.Live()
	
	// Configure the connection
	config := &genai.LiveConnectConfig{
		Model: c.model,
		// Add any additional configuration here
	}
	
	// Connect to the Live API
	session, err := live.Connect(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to connect to Gemini Live API: %w", err)
	}
	
	c.live = live
	c.session = session
	c.isConnected = true
	
	log.Println("Gemini Live API initialized successfully")
	return nil
}

// SendAudioInput sends audio input to Gemini
func (c *Client) SendAudioInput(audioData []byte, sessionID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected || c.session == nil {
		return fmt.Errorf("not connected to Gemini Live API")
	}

	// Encode audio data to base64
	audioB64 := base64.StdEncoding.EncodeToString(audioData)
	
	// Create media chunk
	mediaChunk := &genai.Blob{
		MimeType: fmt.Sprintf("audio/pcm;rate=%d", c.audioConfig.SampleRate),
		Data:     []byte(audioB64),
	}
	
	// Create realtime input
	input := &genai.LiveRealtimeInput{
		MediaChunks: []*genai.Blob{mediaChunk},
	}
	
	// Send to Gemini
	ctx := context.Background()
	if err := c.session.SendRealtimeInput(ctx, input); err != nil {
		return fmt.Errorf("failed to send audio input: %w", err)
	}
	
	log.Printf("Audio input sent for session %s", sessionID)
	return nil
}

// SendTextInput sends text input to Gemini
func (c *Client) SendTextInput(text, sessionID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.isConnected || c.session == nil {
		return fmt.Errorf("not connected to Gemini Live API")
	}

	// Create client content
	content := &genai.LiveClientContent{
		Turns: []*genai.LiveClientTurn{
			{
				Role: "USER",
				Parts: []*genai.Part{
					{Text: text},
				},
			},
		},
		TurnComplete: true,
	}
	
	// Send to Gemini
	ctx := context.Background()
	if err := c.session.SendRealtimeInput(ctx, content); err != nil {
		return fmt.Errorf("failed to send text input: %w", err)
	}
	
	log.Printf("Text input sent for session %s: %s", sessionID, text)
	return nil
}

// ListenForResponses listens for responses from Gemini and calls the handler
func (c *Client) ListenForResponses(handler func(map[string]interface{}) error) error {
	c.mu.Lock()
	c.responseHandler = handler
	c.mu.Unlock()

	if !c.isConnected || c.session == nil {
		return fmt.Errorf("not connected to Gemini Live API")
	}

	ctx := context.Background()
	
	// Start listening for responses
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Read response from session
				response, err := c.session.ReadResponse(ctx)
				if err != nil {
					log.Printf("Error reading response: %v", err)
					continue
				}
				
				// Convert response to map for handler
				responseMap, err := c.convertResponseToMap(response)
				if err != nil {
					log.Printf("Error converting response: %v", err)
					continue
				}
				
				// Call handler
				if c.responseHandler != nil {
					if err := c.responseHandler(responseMap); err != nil {
						log.Printf("Error in response handler: %v", err)
					}
				}
			}
		}
	}()
	
	return nil
}

// convertResponseToMap converts Gemini response to map
func (c *Client) convertResponseToMap(response *genai.LiveServerMessage) (map[string]interface{}, error) {
	// Convert to JSON first, then to map
	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, err
	}
	
	return result, nil
}

// DecodeAudioOutput decodes audio output from Gemini response
func (c *Client) DecodeAudioOutput(message map[string]interface{}) ([]byte, error) {
	// Navigate through the response structure
	serverContent, ok := message["serverContent"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	
	modelTurn, ok := serverContent["modelTurn"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	
	parts, ok := modelTurn["parts"].([]interface{})
	if !ok {
		return nil, nil
	}
	
	var audioData []byte
	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}
		
		inlineData, ok := partMap["inlineData"].(map[string]interface{})
		if !ok {
			continue
		}
		
		data, ok := inlineData["data"].(string)
		if !ok {
			continue
		}
		
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			continue
		}
		
		audioData = append(audioData, decoded...)
	}
	
	return audioData, nil
}

// DecodeTextOutput decodes text output from Gemini response
func (c *Client) DecodeTextOutput(message map[string]interface{}) (string, error) {
	// Navigate through the response structure
	serverContent, ok := message["serverContent"].(map[string]interface{})
	if !ok {
		return "", nil
	}
	
	modelTurn, ok := serverContent["modelTurn"].(map[string]interface{})
	if !ok {
		return "", nil
	}
	
	parts, ok := modelTurn["parts"].([]interface{})
	if !ok {
		return "", nil
	}
	
	var textParts []string
	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}
		
		text, ok := partMap["text"].(string)
		if !ok {
			continue
		}
		
		textParts = append(textParts, text)
	}
	
	if len(textParts) == 0 {
		return "", nil
	}
	
	// Join all text parts
	result := ""
	for i, part := range textParts {
		if i > 0 {
			result += " "
		}
		result += part
	}
	
	return result, nil
}

// IsInterrupted checks if the message indicates user interruption
func (c *Client) IsInterrupted(message map[string]interface{}) bool {
	serverContent, ok := message["serverContent"].(map[string]interface{})
	if !ok {
		return false
	}
	
	_, interrupted := serverContent["interrupted"]
	return interrupted
}

// IsTurnComplete checks if the message indicates turn completion
func (c *Client) IsTurnComplete(message map[string]interface{}) bool {
	serverContent, ok := message["serverContent"].(map[string]interface{})
	if !ok {
		return false
	}
	
	_, turnComplete := serverContent["turnComplete"]
	return turnComplete
}

// SetAudioConfig sets audio configuration
func (c *Client) SetAudioConfig(sampleRate, channels int, format string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.audioConfig = AudioConfig{
		SampleRate: sampleRate,
		Channels:   channels,
		Format:     format,
	}
	
	log.Printf("Audio config updated: %+v", c.audioConfig)
}

// Close closes the Gemini Live API connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.session != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := c.session.Close(ctx); err != nil {
			log.Printf("Error closing session: %v", err)
		}
		c.session = nil
	}
	
	if c.client != nil {
		if err := c.client.Close(); err != nil {
			log.Printf("Error closing client: %v", err)
		}
		c.client = nil
	}
	
	c.isConnected = false
	log.Println("Gemini Live API connection closed")
	return nil
}

// IsConnected returns the connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}
