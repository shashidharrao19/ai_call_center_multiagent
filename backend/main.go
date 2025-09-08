package main

import (
	"ai-call-center/backend/internal/config"
	"ai-call-center/backend/internal/gateway"
	"ai-call-center/backend/internal/websocket"
	"ai-call-center/backend/internal/profiling"
	"ai-call-center/backend/internal/gemini"
	"ai-call-center/backend/internal/conversation"
	"ai-call-center/backend/internal/mcp"
	"ai-call-center/backend/pkg/utils"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found, using system environment variables")
	}

	// Initialize configuration
	cfg := config.Load()

	// Setup logging
	utils.SetupLogging(cfg.LogLevel, cfg.LogFormat)

	// Initialize profiler
	profiler := profiling.NewProfiler("6060", true)
	if err := profiler.Start(); err != nil {
		logrus.Fatalf("Failed to start profiler: %v", err)
	}
	defer profiler.Stop(context.Background())

	logrus.Info("Starting AI Call Center Backend...")

	// Initialize Gemini Live API client
	geminiClient, err := gemini.NewClient(cfg.GoogleAPIKey, cfg.GeminiModel)
	if err != nil {
		logrus.Fatalf("Failed to initialize Gemini client: %v", err)
	}
	defer geminiClient.Close()

	// Initialize MCP client
	mcpClient := mcp.NewClient(cfg.MCPServerURL)
	if err := mcpClient.Connect(); err != nil {
		logrus.Warnf("Failed to connect to MCP server: %v", err)
	}
	defer mcpClient.Disconnect()

	// Initialize conversation manager
	conversationManager := conversation.NewManager(geminiClient, mcpClient, profiler)

	// Initialize Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize gateway with profiler and AI components
	gatewayHandler := gateway.NewGateway(hub, cfg, profiler, conversationManager)
	
	// Start audio processors
	gatewayHandler.StartAudioProcessors()
	defer gatewayHandler.StopAudioProcessors()

	// Setup routes
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	router.GET("/performance", func(c *gin.Context) {
		metrics := gatewayHandler.GetOptimizationMetrics()
		c.JSON(http.StatusOK, metrics)
	})

	// AI Engine endpoints
	router.POST("/process", func(c *gin.Context) {
		var req conversation.ProcessRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
			return
		}

		response, err := conversationManager.ProcessMessage(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, response)
	})

	router.GET("/sessions", func(c *gin.Context) {
		sessions := conversationManager.GetActiveSessions()
		c.JSON(http.StatusOK, gin.H{"sessions": sessions})
	})

	router.DELETE("/sessions/:session_id", func(c *gin.Context) {
		sessionID := c.Param("session_id")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
			return
		}

		if err := conversationManager.EndSession(sessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Session " + sessionID + " ended successfully"})
	})

	router.GET(cfg.WebSocketPath, gatewayHandler.HandleWebSocket)

	// Start server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logrus.Infof("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.Fatalf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server exited")
}
