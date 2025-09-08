package gemini

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	// Test with empty API key
	_, err := NewClient("", "test-model")
	if err == nil {
		t.Error("Expected error for empty API key")
	}

	// Test with valid API key (this will fail to connect without real key)
	client, err := NewClient("test-key", "test-model")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if client == nil {
		t.Error("Expected client to be created")
	}

	if client.apiKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", client.apiKey)
	}

	if client.model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", client.model)
	}
}

func TestAudioConfig(t *testing.T) {
	client := &Client{
		audioConfig: AudioConfig{
			SampleRate: 24000,
			Channels:   1,
			Format:     "PCM",
		},
	}

	// Test setting audio config
	client.SetAudioConfig(48000, 2, "WAV")

	if client.audioConfig.SampleRate != 48000 {
		t.Errorf("Expected sample rate 48000, got %d", client.audioConfig.SampleRate)
	}

	if client.audioConfig.Channels != 2 {
		t.Errorf("Expected channels 2, got %d", client.audioConfig.Channels)
	}

	if client.audioConfig.Format != "WAV" {
		t.Errorf("Expected format 'WAV', got '%s'", client.audioConfig.Format)
	}
}

func TestIsConnected(t *testing.T) {
	client := &Client{
		isConnected: false,
	}

	if client.IsConnected() {
		t.Error("Expected client to not be connected")
	}

	client.isConnected = true
	if !client.IsConnected() {
		t.Error("Expected client to be connected")
	}
}
