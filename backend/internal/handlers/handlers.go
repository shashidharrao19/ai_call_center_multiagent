package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"ai-call-center/backend/internal/conversation"
	"ai-call-center/backend/internal/profiling"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	conversationManager *conversation.Manager
	profiler           *profiling.Profiler
}

// NewHandlers creates a new handlers instance
func NewHandlers(conversationManager *conversation.Manager, profiler *profiling.Profiler) *Handlers {
	return &Handlers{
		conversationManager: conversationManager,
		profiler:           profiler,
	}
}

// HealthCheck handles health check requests
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "ai-engine-go",
	})
}

// ProcessMessage handles message processing requests
func (h *Handlers) ProcessMessage(c *gin.Context) {
	var req conversation.ProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Process the message through conversation manager
	response, err := h.conversationManager.ProcessMessage(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetActiveSessions handles getting active sessions
func (h *Handlers) GetActiveSessions(c *gin.Context) {
	sessions := h.conversationManager.GetActiveSessions()
	c.JSON(http.StatusOK, gin.H{
		"sessions": sessions,
	})
}

// GetPerformanceReport handles getting performance report
func (h *Handlers) GetPerformanceReport(c *gin.Context) {
	report := h.profiler.GetPerformanceReport()
	c.JSON(http.StatusOK, report)
}

// GetMetrics handles getting metrics in Prometheus format
func (h *Handlers) GetMetrics(c *gin.Context) {
	report := h.profiler.GetPerformanceReport()

	// Convert to Prometheus format
	var metrics []string

	// Gemini metrics
	if gemini, ok := report["gemini"].(map[string]interface{}); ok {
		if requests, ok := gemini["requests"].(int64); ok {
			metrics = append(metrics, "# HELP gemini_requests_total Total Gemini API requests")
			metrics = append(metrics, "# TYPE gemini_requests_total counter")
			metrics = append(metrics, "gemini_requests_total "+strconv.FormatInt(requests, 10))
		}

		if errors, ok := gemini["errors"].(int64); ok {
			metrics = append(metrics, "# HELP gemini_errors_total Total Gemini API errors")
			metrics = append(metrics, "# TYPE gemini_errors_total counter")
			metrics = append(metrics, "gemini_errors_total "+strconv.FormatInt(errors, 10))
		}

		if latency, ok := gemini["latency"].(map[string]interface{}); ok {
			if p95, ok := latency["p95"].(float64); ok {
				metrics = append(metrics, "# HELP gemini_latency_p95_ms Gemini API latency p95 in milliseconds")
				metrics = append(metrics, "# TYPE gemini_latency_p95_ms gauge")
				metrics = append(metrics, "gemini_latency_p95_ms "+strconv.FormatFloat(p95, 'f', 2, 64))
			}
		}
	}

	// Audio processing metrics
	if audioProcessing, ok := report["audio_processing"].(map[string]interface{}); ok {
		if chunksProcessed, ok := audioProcessing["chunks_processed"].(int64); ok {
			metrics = append(metrics, "# HELP audio_chunks_processed_total Total audio chunks processed")
			metrics = append(metrics, "# TYPE audio_chunks_processed_total counter")
			metrics = append(metrics, "audio_chunks_processed_total "+strconv.FormatInt(chunksProcessed, 10))
		}

		if latency, ok := audioProcessing["latency"].(map[string]interface{}); ok {
			if p95, ok := latency["p95"].(float64); ok {
				metrics = append(metrics, "# HELP audio_processing_latency_p95_ms Audio processing latency p95 in milliseconds")
				metrics = append(metrics, "# TYPE audio_processing_latency_p95_ms gauge")
				metrics = append(metrics, "audio_processing_latency_p95_ms "+strconv.FormatFloat(p95, 'f', 2, 64))
			}
		}
	}

	// MCP metrics
	if mcp, ok := report["mcp"].(map[string]interface{}); ok {
		if calls, ok := mcp["calls"].(int64); ok {
			metrics = append(metrics, "# HELP mcp_calls_total Total MCP function calls")
			metrics = append(metrics, "# TYPE mcp_calls_total counter")
			metrics = append(metrics, "mcp_calls_total "+strconv.FormatInt(calls, 10))
		}

		if errors, ok := mcp["errors"].(int64); ok {
			metrics = append(metrics, "# HELP mcp_errors_total Total MCP function errors")
			metrics = append(metrics, "# TYPE mcp_errors_total counter")
			metrics = append(metrics, "mcp_errors_total "+strconv.FormatInt(errors, 10))
		}
	}

	// System metrics
	if system, ok := report["system"].(map[string]interface{}); ok {
		if cpuUsage, ok := system["cpu_usage"].(float64); ok {
			metrics = append(metrics, "# HELP cpu_usage_percent Current CPU usage percentage")
			metrics = append(metrics, "# TYPE cpu_usage_percent gauge")
			metrics = append(metrics, "cpu_usage_percent "+strconv.FormatFloat(cpuUsage, 'f', 2, 64))
		}

		if memoryUsage, ok := system["memory_usage"].(float64); ok {
			metrics = append(metrics, "# HELP memory_usage_mb Current memory usage in MB")
			metrics = append(metrics, "# TYPE memory_usage_mb gauge")
			metrics = append(metrics, "memory_usage_mb "+strconv.FormatFloat(memoryUsage, 'f', 2, 64))
		}

		if activeTasks, ok := system["active_tasks"].(int64); ok {
			metrics = append(metrics, "# HELP active_tasks_total Current number of active goroutines")
			metrics = append(metrics, "# TYPE active_tasks_total gauge")
			metrics = append(metrics, "active_tasks_total "+strconv.FormatInt(activeTasks, 10))
		}
	}

	c.String(http.StatusOK, strings.Join(metrics, "\n")+"\n")
}

// EndSession handles ending a conversation session
func (h *Handlers) EndSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}

	if err := h.conversationManager.EndSession(sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Session " + sessionID + " ended successfully",
	})
}
