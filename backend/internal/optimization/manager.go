package optimization

import (
	"ai-call-center/backend/internal/config"
	"ai-call-center/backend/pkg/models"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// OptimizationManager manages all optimization components
type OptimizationManager struct {
	config        *config.Config
	bufferPool    *AudioBufferPool
	ringBuffers   map[string]*LockFreeRingBuffer
	audioProcessors []*AudioProcessor
	backpressureHandlers map[string]*BackpressureHandler
	metrics       *OptimizationMetrics
	mutex         sync.RWMutex
}

// OptimizationMetrics tracks overall optimization performance
type OptimizationMetrics struct {
	BufferPoolMetrics    *PoolMetrics            `json:"buffer_pool"`
	RingBufferMetrics    map[string]*RingBufferMetrics `json:"ring_buffers"`
	AudioProcessorCount  int                     `json:"audio_processor_count"`
	BackpressureEvents   int64                   `json:"backpressure_events"`
	LastUpdate           time.Time               `json:"last_update"`
}

// NewOptimizationManager creates a new optimization manager
func NewOptimizationManager(cfg *config.Config) *OptimizationManager {
	om := &OptimizationManager{
		config:               cfg,
		ringBuffers:          make(map[string]*LockFreeRingBuffer),
		backpressureHandlers: make(map[string]*BackpressureHandler),
		metrics: &OptimizationMetrics{
			RingBufferMetrics: make(map[string]*RingBufferMetrics),
			LastUpdate:        time.Now(),
		},
	}
	
	// Initialize buffer pool
	om.bufferPool = NewAudioBufferPool(cfg.AudioBufferPoolSize)
	
	// Initialize audio processors
	om.initializeAudioProcessors()
	
	logrus.Info("Optimization manager initialized")
	return om
}

// initializeAudioProcessors sets up audio processors with CPU affinity
func (om *OptimizationManager) initializeAudioProcessors() {
	cpuCount := runtime.NumCPU()
	audioCPUCount := om.config.AudioCPUCount
	
	// Limit audio CPU count to available CPUs
	if audioCPUCount > cpuCount {
		audioCPUCount = cpuCount
		logrus.Warnf("Audio CPU count limited to available CPUs: %d", cpuCount)
	}
	
	// Create audio processors
	for i := 0; i < audioCPUCount; i++ {
		cpuID := om.config.AudioCPUStart + i
		if cpuID >= cpuCount {
			cpuID = i // Wrap around if needed
		}
		
		processor := NewAudioProcessor(cpuID, om.bufferPool)
		om.audioProcessors = append(om.audioProcessors, processor)
	}
	
	om.metrics.AudioProcessorCount = len(om.audioProcessors)
	logrus.Infof("Initialized %d audio processors", len(om.audioProcessors))
}

// CreateSessionOptimizations creates optimization components for a session
func (om *OptimizationManager) CreateSessionOptimizations(sessionID string) {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	
	// Create ring buffer for session
	ringBuffer := NewLockFreeRingBuffer(om.config.RingBufferSize)
	om.ringBuffers[sessionID] = ringBuffer
	
	// Create backpressure handler for session
	backpressureHandler := NewBackpressureHandler(om.config.MaxQueueSize, om.config.BackpressureStrategy)
	om.backpressureHandlers[sessionID] = backpressureHandler
	
	logrus.Debugf("Created optimization components for session: %s", sessionID)
}

// GetSessionOptimizations returns optimization components for a session
func (om *OptimizationManager) GetSessionOptimizations(sessionID string) (*LockFreeRingBuffer, *BackpressureHandler) {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	
	ringBuffer := om.ringBuffers[sessionID]
	backpressureHandler := om.backpressureHandlers[sessionID]
	
	return ringBuffer, backpressureHandler
}

// AssignSessionToProcessor assigns a session to an audio processor
func (om *OptimizationManager) AssignSessionToProcessor(sessionID string) *AudioProcessor {
	if len(om.audioProcessors) == 0 {
		return nil
	}
	
	// Simple round-robin assignment
	processorID := hashString(sessionID) % len(om.audioProcessors)
	return om.audioProcessors[processorID]
}

// RemoveSessionOptimizations removes optimization components for a session
func (om *OptimizationManager) RemoveSessionOptimizations(sessionID string) {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	
	// Remove ring buffer
	if ringBuffer, exists := om.ringBuffers[sessionID]; exists {
		ringBuffer.Reset()
		delete(om.ringBuffers, sessionID)
	}
	
	// Remove backpressure handler
	if backpressureHandler, exists := om.backpressureHandlers[sessionID]; exists {
		backpressureHandler.Reset()
		delete(om.backpressureHandlers, sessionID)
	}
	
	logrus.Debugf("Removed optimization components for session: %s", sessionID)
}

// GetBufferPool returns the shared buffer pool
func (om *OptimizationManager) GetBufferPool() *AudioBufferPool {
	return om.bufferPool
}

// GetMetrics returns current optimization metrics
func (om *OptimizationManager) GetMetrics() *OptimizationMetrics {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	
	// Update buffer pool metrics
	om.metrics.BufferPoolMetrics = om.bufferPool.GetMetrics()
	
	// Update ring buffer metrics
	om.metrics.RingBufferMetrics = make(map[string]*RingBufferMetrics)
	for sessionID, ringBuffer := range om.ringBuffers {
		om.metrics.RingBufferMetrics[sessionID] = ringBuffer.GetMetrics()
	}
	
	// Update backpressure events
	totalBackpressureEvents := int64(0)
	for _, handler := range om.backpressureHandlers {
		totalBackpressureEvents += handler.GetDroppedCount()
	}
	om.metrics.BackpressureEvents = totalBackpressureEvents
	
	om.metrics.LastUpdate = time.Now()
	return om.metrics
}

// StartAudioProcessors starts all audio processors
func (om *OptimizationManager) StartAudioProcessors() {
	for _, processor := range om.audioProcessors {
		processor.Start()
	}
	logrus.Info("All audio processors started")
}

// StopAudioProcessors stops all audio processors
func (om *OptimizationManager) StopAudioProcessors() {
	for _, processor := range om.audioProcessors {
		processor.Stop()
	}
	logrus.Info("All audio processors stopped")
}

// Reset resets all optimization components
func (om *OptimizationManager) Reset() {
	om.mutex.Lock()
	defer om.mutex.Unlock()
	
	// Reset buffer pool
	om.bufferPool.Reset()
	
	// Reset all ring buffers
	for _, ringBuffer := range om.ringBuffers {
		ringBuffer.Reset()
	}
	
	// Reset all backpressure handlers
	for _, handler := range om.backpressureHandlers {
		handler.Reset()
	}
	
	om.metrics.LastUpdate = time.Now()
	logrus.Info("All optimization components reset")
}

// String returns a string representation of the optimization manager
func (om *OptimizationManager) String() string {
	om.mutex.RLock()
	defer om.mutex.RUnlock()
	
	return fmt.Sprintf("OptimizationManager(processors=%d, sessions=%d, buffer_pool=%s)",
		len(om.audioProcessors), len(om.ringBuffers), om.bufferPool.String())
}

// hashString creates a simple hash for string distribution
func hashString(s string) int {
	hash := 0
	for _, c := range s {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}
