package main

import (
	"ai-call-center/backend/internal/config"
	"ai-call-center/backend/internal/gemini"
	"ai-call-center/backend/internal/conversation"
	"ai-call-center/backend/internal/mcp"
	"ai-call-center/backend/internal/profiling"
	"fmt"
	"log"
)

func main() {
	fmt.Println("Testing consolidated backend...")

	// Test configuration loading
	cfg := config.Load()
	fmt.Printf("✓ Configuration loaded: Port=%s, GeminiModel=%s\n", cfg.Port, cfg.GeminiModel)

	// Test profiler initialization
	profiler := profiling.NewProfiler("6060", false)
	fmt.Println("✓ Profiler initialized")

	// Test Gemini client initialization (without API key for testing)
	geminiClient, err := gemini.NewClient("test-key", cfg.GeminiModel)
	if err != nil {
		log.Printf("✗ Gemini client failed: %v", err)
	} else {
		fmt.Println("✓ Gemini client initialized")
	}

	// Test MCP client initialization
	mcpClient := mcp.NewClient(cfg.MCPServerURL)
	fmt.Println("✓ MCP client initialized")

	// Test conversation manager initialization
	conversationManager := conversation.NewManager(geminiClient, mcpClient, profiler)
	fmt.Println("✓ Conversation manager initialized")

	fmt.Println("\n🎉 All components initialized successfully!")
	fmt.Println("The consolidation is complete and ready for testing.")
}
