package profiling

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// Profiler manages profiling and performance monitoring
type Profiler struct {
	server   *http.Server
	enabled  bool
	port     string
	metrics  *MetricsCollector
}

// MetricsCollector collects performance metrics
type MetricsCollector struct {
	// WebSocket metrics
	WSConnections     int64
	WSMessagesReceived int64
	WSMessagesSent    int64
	WSLatency         *LatencyHistogram
	
	// Go handler metrics
	HandlerLatency    *LatencyHistogram
	HandlerErrors     int64
	
	// RPC metrics
	RPCCalls          int64
	RPCLatency        *LatencyHistogram
	RPCErrors         int64
	
	// Audio processing metrics
	AudioChunksProcessed int64
	AudioProcessingLatency *LatencyHistogram
	DroppedAudioChunks int64
	
	// AI Engine metrics
	GeminiRequests int64
	GeminiErrors   int64
	GeminiLatency  *LatencyHistogram
	
	// MCP metrics
	MCPCalls  int64
	MCPErrors int64
	MCPLatency *LatencyHistogram
	
	// System metrics
	CPUUsage          float64
	MemoryUsage       uint64
	GCPauseTime       time.Duration
}

// LatencyHistogram tracks latency percentiles
type LatencyHistogram struct {
	measurements []time.Duration
	mutex        sync.RWMutex
}

func NewLatencyHistogram() *LatencyHistogram {
	return &LatencyHistogram{
		measurements: make([]time.Duration, 0, 10000),
	}
}

func (lh *LatencyHistogram) Record(duration time.Duration) {
	lh.mutex.Lock()
	defer lh.mutex.Unlock()
	
	if len(lh.measurements) >= 10000 {
		// Keep only recent measurements
		lh.measurements = lh.measurements[5000:]
	}
	lh.measurements = append(lh.measurements, duration)
}

func (lh *LatencyHistogram) GetPercentiles() (p50, p95, p99 time.Duration) {
	lh.mutex.RLock()
	defer lh.mutex.RUnlock()
	
	if len(lh.measurements) == 0 {
		return 0, 0, 0
	}
	
	// Sort measurements
	sorted := make([]time.Duration, len(lh.measurements))
	copy(sorted, lh.measurements)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	
	n := len(sorted)
	p50 = sorted[int(float64(n)*0.50)]
	p95 = sorted[int(float64(n)*0.95)]
	p99 = sorted[int(float64(n)*0.99)]
	
	return p50, p95, p99
}

// NewProfiler creates a new profiler instance
func NewProfiler(port string, enabled bool) *Profiler {
	return &Profiler{
		enabled: enabled,
		port:    port,
		metrics: &MetricsCollector{
			WSLatency:             NewLatencyHistogram(),
			HandlerLatency:        NewLatencyHistogram(),
			RPCLatency:            NewLatencyHistogram(),
			AudioProcessingLatency: NewLatencyHistogram(),
			GeminiLatency:         NewLatencyHistogram(),
			MCPLatency:            NewLatencyHistogram(),
		},
	}
}

// Start starts the profiling server
func (p *Profiler) Start() error {
	if !p.enabled {
		logrus.Info("Profiling disabled")
		return nil
	}
	
	mux := http.NewServeMux()
	
	// Add pprof endpoints
	mux.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)
	
	// Add custom metrics endpoint
	mux.HandleFunc("/metrics", p.handleMetrics)
	
	// Add performance report endpoint
	mux.HandleFunc("/performance", p.handlePerformanceReport)
	
	p.server = &http.Server{
		Addr:    ":" + p.port,
		Handler: mux,
	}
	
	go func() {
		logrus.Infof("Starting profiler on port %s", p.port)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("Profiler server error: %v", err)
		}
	}()
	
	// Start metrics collection
	go p.collectSystemMetrics()
	
	return nil
}

// Stop stops the profiling server
func (p *Profiler) Stop(ctx context.Context) error {
	if p.server == nil {
		return nil
	}
	return p.server.Shutdown(ctx)
}

// RecordWebSocketLatency records WebSocket message latency
func (p *Profiler) RecordWebSocketLatency(duration time.Duration) {
	p.metrics.WSLatency.Record(duration)
}

// RecordHandlerLatency records Go handler processing latency
func (p *Profiler) RecordHandlerLatency(duration time.Duration) {
	p.metrics.HandlerLatency.Record(duration)
}

// RecordRPCLatency records RPC call latency
func (p *Profiler) RecordRPCLatency(duration time.Duration) {
	p.metrics.RPCLatency.Record(duration)
}

// RecordAudioProcessingLatency records audio processing latency
func (p *Profiler) RecordAudioProcessingLatency(duration time.Duration) {
	p.metrics.AudioProcessingLatency.Record(duration)
}

// IncrementConnections increments connection count
func (p *Profiler) IncrementConnections() {
	atomic.AddInt64(&p.metrics.WSConnections, 1)
}

// DecrementConnections decrements connection count
func (p *Profiler) DecrementConnections() {
	atomic.AddInt64(&p.metrics.WSConnections, -1)
}

// IncrementMessagesReceived increments received message count
func (p *Profiler) IncrementMessagesReceived() {
	atomic.AddInt64(&p.metrics.WSMessagesReceived, 1)
}

// IncrementMessagesSent increments sent message count
func (p *Profiler) IncrementMessagesSent() {
	atomic.AddInt64(&p.metrics.WSMessagesSent, 1)
}

// IncrementRPCCalls increments RPC call count
func (p *Profiler) IncrementRPCCalls() {
	atomic.AddInt64(&p.metrics.RPCCalls, 1)
}

// IncrementRPCErrors increments RPC error count
func (p *Profiler) IncrementRPCErrors() {
	atomic.AddInt64(&p.metrics.RPCErrors, 1)
}

// IncrementAudioChunksProcessed increments processed audio chunks
func (p *Profiler) IncrementAudioChunksProcessed() {
	atomic.AddInt64(&p.metrics.AudioChunksProcessed, 1)
}

// IncrementDroppedAudioChunks increments dropped audio chunks
func (p *Profiler) IncrementDroppedAudioChunks() {
	atomic.AddInt64(&p.metrics.DroppedAudioChunks, 1)
}

// RecordGeminiRequest records a Gemini API request
func (p *Profiler) RecordGeminiRequest(duration time.Duration, success bool) {
	atomic.AddInt64(&p.metrics.GeminiRequests, 1)
	if !success {
		atomic.AddInt64(&p.metrics.GeminiErrors, 1)
	}
	p.metrics.GeminiLatency.Record(duration)
}

// RecordMCPCall records an MCP function call
func (p *Profiler) RecordMCPCall(duration time.Duration, success bool) {
	atomic.AddInt64(&p.metrics.MCPCalls, 1)
	if !success {
		atomic.AddInt64(&p.metrics.MCPErrors, 1)
	}
	p.metrics.MCPLatency.Record(duration)
}

// collectSystemMetrics collects system-level metrics
func (p *Profiler) collectSystemMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		
		p.metrics.MemoryUsage = m.Alloc
		p.metrics.GCPauseTime = time.Duration(m.PauseTotalNs)
		
		// Get CPU usage (simplified)
		p.metrics.CPUUsage = getCPUUsage()
	}
}

// getCPUUsage returns current CPU usage percentage
func getCPUUsage() float64 {
	// Simplified CPU usage calculation
	// In production, use more sophisticated methods
	return 0.0 // Placeholder
}

// handleMetrics returns current metrics in Prometheus format
func (p *Profiler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	
	fmt.Fprintf(w, "# HELP websocket_connections Current WebSocket connections\n")
	fmt.Fprintf(w, "# TYPE websocket_connections gauge\n")
	fmt.Fprintf(w, "websocket_connections %d\n", atomic.LoadInt64(&p.metrics.WSConnections))
	
	fmt.Fprintf(w, "# HELP websocket_messages_received Total WebSocket messages received\n")
	fmt.Fprintf(w, "# TYPE websocket_messages_received counter\n")
	fmt.Fprintf(w, "websocket_messages_received %d\n", atomic.LoadInt64(&p.metrics.WSMessagesReceived))
	
	fmt.Fprintf(w, "# HELP websocket_messages_sent Total WebSocket messages sent\n")
	fmt.Fprintf(w, "# TYPE websocket_messages_sent counter\n")
	fmt.Fprintf(w, "websocket_messages_sent %d\n", atomic.LoadInt64(&p.metrics.WSMessagesSent))
	
	fmt.Fprintf(w, "# HELP rpc_calls Total RPC calls\n")
	fmt.Fprintf(w, "# TYPE rpc_calls counter\n")
	fmt.Fprintf(w, "rpc_calls %d\n", atomic.LoadInt64(&p.metrics.RPCCalls))
	
	fmt.Fprintf(w, "# HELP rpc_errors Total RPC errors\n")
	fmt.Fprintf(w, "# TYPE rpc_errors counter\n")
	fmt.Fprintf(w, "rpc_errors %d\n", atomic.LoadInt64(&p.metrics.RPCErrors))
	
	fmt.Fprintf(w, "# HELP audio_chunks_processed Total audio chunks processed\n")
	fmt.Fprintf(w, "# TYPE audio_chunks_processed counter\n")
	fmt.Fprintf(w, "audio_chunks_processed %d\n", atomic.LoadInt64(&p.metrics.AudioChunksProcessed))
	
	fmt.Fprintf(w, "# HELP dropped_audio_chunks Total dropped audio chunks\n")
	fmt.Fprintf(w, "# TYPE dropped_audio_chunks counter\n")
	fmt.Fprintf(w, "dropped_audio_chunks %d\n", atomic.LoadInt64(&p.metrics.DroppedAudioChunks))
	
	fmt.Fprintf(w, "# HELP memory_usage_bytes Current memory usage in bytes\n")
	fmt.Fprintf(w, "# TYPE memory_usage_bytes gauge\n")
	fmt.Fprintf(w, "memory_usage_bytes %d\n", p.metrics.MemoryUsage)
	
	fmt.Fprintf(w, "# HELP gc_pause_time_ns Total GC pause time in nanoseconds\n")
	fmt.Fprintf(w, "# TYPE gc_pause_time_ns gauge\n")
	fmt.Fprintf(w, "gc_pause_time_ns %d\n", p.metrics.GCPauseTime.Nanoseconds())
	
	// AI Engine metrics
	fmt.Fprintf(w, "# HELP gemini_requests_total Total Gemini API requests\n")
	fmt.Fprintf(w, "# TYPE gemini_requests_total counter\n")
	fmt.Fprintf(w, "gemini_requests_total %d\n", atomic.LoadInt64(&p.metrics.GeminiRequests))
	
	fmt.Fprintf(w, "# HELP gemini_errors_total Total Gemini API errors\n")
	fmt.Fprintf(w, "# TYPE gemini_errors_total counter\n")
	fmt.Fprintf(w, "gemini_errors_total %d\n", atomic.LoadInt64(&p.metrics.GeminiErrors))
	
	fmt.Fprintf(w, "# HELP mcp_calls_total Total MCP function calls\n")
	fmt.Fprintf(w, "# TYPE mcp_calls_total counter\n")
	fmt.Fprintf(w, "mcp_calls_total %d\n", atomic.LoadInt64(&p.metrics.MCPCalls))
	
	fmt.Fprintf(w, "# HELP mcp_errors_total Total MCP function errors\n")
	fmt.Fprintf(w, "# TYPE mcp_errors_total counter\n")
	fmt.Fprintf(w, "mcp_errors_total %d\n", atomic.LoadInt64(&p.metrics.MCPErrors))
}

// handlePerformanceReport returns detailed performance report
func (p *Profiler) handlePerformanceReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	report := PerformanceReport{
		Timestamp: time.Now(),
		WebSocket: WebSocketMetrics{
			Connections: atomic.LoadInt64(&p.metrics.WSConnections),
			MessagesReceived: atomic.LoadInt64(&p.metrics.WSMessagesReceived),
			MessagesSent: atomic.LoadInt64(&p.metrics.WSMessagesSent),
			Latency: p.metrics.WSLatency.GetPercentiles(),
		},
		Handler: HandlerMetrics{
			Latency: p.metrics.HandlerLatency.GetPercentiles(),
			Errors:  atomic.LoadInt64(&p.metrics.HandlerErrors),
		},
		RPC: RPCMetrics{
			Calls:   atomic.LoadInt64(&p.metrics.RPCCalls),
			Errors:  atomic.LoadInt64(&p.metrics.RPCErrors),
			Latency: p.metrics.RPCLatency.GetPercentiles(),
		},
		Audio: AudioMetrics{
			ChunksProcessed: atomic.LoadInt64(&p.metrics.AudioChunksProcessed),
			DroppedChunks:   atomic.LoadInt64(&p.metrics.DroppedAudioChunks),
			ProcessingLatency: p.metrics.AudioProcessingLatency.GetPercentiles(),
		},
		AI: AIMetrics{
			GeminiRequests: atomic.LoadInt64(&p.metrics.GeminiRequests),
			GeminiErrors:   atomic.LoadInt64(&p.metrics.GeminiErrors),
			GeminiLatency:  p.metrics.GeminiLatency.GetPercentiles(),
			MCPCalls:       atomic.LoadInt64(&p.metrics.MCPCalls),
			MCPErrors:      atomic.LoadInt64(&p.metrics.MCPErrors),
			MCPLatency:     p.metrics.MCPLatency.GetPercentiles(),
		},
		System: SystemMetrics{
			CPUUsage:    p.metrics.CPUUsage,
			MemoryUsage: p.metrics.MemoryUsage,
			GCPauseTime: p.metrics.GCPauseTime,
		},
	}
	
	json.NewEncoder(w).Encode(report)
}

// PerformanceReport represents the complete performance report
type PerformanceReport struct {
	Timestamp time.Time       `json:"timestamp"`
	WebSocket WebSocketMetrics `json:"websocket"`
	Handler   HandlerMetrics   `json:"handler"`
	RPC       RPCMetrics       `json:"rpc"`
	Audio     AudioMetrics     `json:"audio"`
	AI        AIMetrics        `json:"ai"`
	System    SystemMetrics    `json:"system"`
}

type WebSocketMetrics struct {
	Connections      int64                `json:"connections"`
	MessagesReceived int64                `json:"messages_received"`
	MessagesSent     int64                `json:"messages_sent"`
	Latency          LatencyPercentiles   `json:"latency"`
}

type HandlerMetrics struct {
	Latency LatencyPercentiles `json:"latency"`
	Errors  int64              `json:"errors"`
}

type RPCMetrics struct {
	Calls   int64              `json:"calls"`
	Errors  int64              `json:"errors"`
	Latency LatencyPercentiles `json:"latency"`
}

type AudioMetrics struct {
	ChunksProcessed   int64              `json:"chunks_processed"`
	DroppedChunks     int64              `json:"dropped_chunks"`
	ProcessingLatency LatencyPercentiles `json:"processing_latency"`
}

type AIMetrics struct {
	GeminiRequests int64              `json:"gemini_requests"`
	GeminiErrors   int64              `json:"gemini_errors"`
	GeminiLatency  LatencyPercentiles `json:"gemini_latency"`
	MCPCalls       int64              `json:"mcp_calls"`
	MCPErrors      int64              `json:"mcp_errors"`
	MCPLatency     LatencyPercentiles `json:"mcp_latency"`
}

type SystemMetrics struct {
	CPUUsage    float64       `json:"cpu_usage"`
	MemoryUsage uint64        `json:"memory_usage"`
	GCPauseTime time.Duration `json:"gc_pause_time"`
}

type LatencyPercentiles struct {
	P50 time.Duration `json:"p50"`
	P95 time.Duration `json:"p95"`
	P99 time.Duration `json:"p99"`
}
