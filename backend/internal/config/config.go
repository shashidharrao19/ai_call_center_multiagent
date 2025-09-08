package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Server configuration
	Port            string
	WebSocketPath   string
	MaxConnections  int
	ReadBufferSize  int
	WriteBufferSize int

	// Logging configuration
	LogLevel  string
	LogFormat string

	// AI Engine configuration
	AIEngineURL string
	PythonPort  string
	
	// Gemini API configuration
	GoogleAPIKey string
	GeminiModel  string
	
	// MCP configuration
	MCPServerURL string
	
	// Session configuration
	SessionTimeout int
	MaxSessions    int
	
	// Performance configuration
	MaxConcurrentRequests int
	RequestTimeout        int

	// Audio configuration
	AudioSampleRate int
	AudioChannels   int
	AudioFormat     string

	// Phase 2 optimization configuration
	AudioBufferPoolSize    int
	AudioBufferPoolCount   int
	RingBufferSize         int
	AudioCPUStart          int
	AudioCPUCount          int
	MaxQueueSize           int
	BackpressureStrategy   string
	CircuitBreakerTimeout  int
}

func Load() *Config {
	return &Config{
		Port:            getEnv("GO_PORT", "8080"),
		WebSocketPath:   getEnv("GO_WEBSOCKET_PATH", "/ws"),
		MaxConnections:  getEnvAsInt("GO_MAX_CONNECTIONS", 1000),
		ReadBufferSize:  getEnvAsInt("GO_READ_BUFFER_SIZE", 4096),
		WriteBufferSize: getEnvAsInt("GO_WRITE_BUFFER_SIZE", 4096),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		LogFormat:       getEnv("LOG_FORMAT", "json"),
		AIEngineURL:     getEnv("AI_ENGINE_URL", "http://localhost:8000"),
		PythonPort:      getEnv("PYTHON_PORT", "8000"),
		
		// Gemini API configuration
		GoogleAPIKey: getEnv("GOOGLE_API_KEY", ""),
		GeminiModel:  getEnv("GEMINI_MODEL", "models/gemini-2.0-flash-live-001"),
		
		// MCP configuration
		MCPServerURL: getEnv("MCP_SERVER_URL", "http://localhost:8001"),
		
		// Session configuration
		SessionTimeout: getEnvAsInt("SESSION_TIMEOUT", 3600),
		MaxSessions:    getEnvAsInt("MAX_SESSIONS", 1000),
		
		// Performance configuration
		MaxConcurrentRequests: getEnvAsInt("MAX_CONCURRENT_REQUESTS", 100),
		RequestTimeout:        getEnvAsInt("REQUEST_TIMEOUT", 30),
		AudioSampleRate: getEnvAsInt("AUDIO_SAMPLE_RATE", 24000),
		AudioChannels:   getEnvAsInt("AUDIO_CHANNELS", 1),
		AudioFormat:     getEnv("AUDIO_FORMAT", "PCM"),

		// Phase 2 optimization configuration
		AudioBufferPoolSize:    getEnvAsInt("AUDIO_BUFFER_POOL_SIZE", 4096),
		AudioBufferPoolCount:   getEnvAsInt("AUDIO_BUFFER_POOL_COUNT", 100),
		RingBufferSize:         getEnvAsInt("RING_BUFFER_SIZE", 1024),
		AudioCPUStart:          getEnvAsInt("AUDIO_CPU_START", 0),
		AudioCPUCount:          getEnvAsInt("AUDIO_CPU_COUNT", 2),
		MaxQueueSize:           getEnvAsInt("MAX_QUEUE_SIZE", 1000),
		BackpressureStrategy:   getEnv("BACKPRESSURE_STRATEGY", "throttle"),
		CircuitBreakerTimeout:  getEnvAsInt("CIRCUIT_BREAKER_TIMEOUT", 30),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
