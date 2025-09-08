package main

import (
	"ai-call-center/backend/internal/config"
	"ai-call-center/backend/internal/gateway"
	"ai-call-center/backend/internal/websocket"
	"ai-call-center/backend/internal/profiling"
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

	// Initialize Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	// Initialize gateway with profiler
	gatewayHandler := gateway.NewGateway(hub, cfg, profiler)
	
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
