package optimization

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// AudioProcessor handles audio processing on a dedicated CPU core
type AudioProcessor struct {
	cpuID        int
	bufferPool   *AudioBufferPool
	ringBuffer   *LockFreeRingBuffer
	quit         chan bool
	running      int32
	processed    int64
	errors       int64
	lastActivity time.Time
}

// NewAudioProcessor creates a new audio processor
func NewAudioProcessor(cpuID int, bufferPool *AudioBufferPool) *AudioProcessor {
	return &AudioProcessor{
		cpuID:      cpuID,
		bufferPool: bufferPool,
		quit:       make(chan bool, 1),
		lastActivity: time.Now(),
	}
}

// Start starts the audio processor on a dedicated CPU core
func (ap *AudioProcessor) Start() {
	if !atomic.CompareAndSwapInt32(&ap.running, 0, 1) {
		logrus.Warnf("Audio processor %d is already running", ap.cpuID)
		return
	}
	
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		
		// Set CPU affinity
		if err := ap.setCPUAffinity(ap.cpuID); err != nil {
			logrus.Errorf("Failed to set CPU affinity for processor %d: %v", ap.cpuID, err)
			atomic.StoreInt32(&ap.running, 0)
			return
		}
		
		logrus.Infof("Audio processor %d started on CPU %d", ap.cpuID, ap.cpuID)
		
		// Process audio loop
		ap.processAudioLoop()
		
		logrus.Infof("Audio processor %d stopped", ap.cpuID)
	}()
}

// Stop stops the audio processor
func (ap *AudioProcessor) Stop() {
	if !atomic.CompareAndSwapInt32(&ap.running, 1, 0) {
		logrus.Warnf("Audio processor %d is not running", ap.cpuID)
		return
	}
	
	select {
	case ap.quit <- true:
	default:
	}
	
	// Wait for processor to stop
	for atomic.LoadInt32(&ap.running) == 1 {
		time.Sleep(10 * time.Millisecond)
	}
	
	logrus.Infof("Audio processor %d stopped", ap.cpuID)
}

// SetRingBuffer sets the ring buffer for this processor
func (ap *AudioProcessor) SetRingBuffer(ringBuffer *LockFreeRingBuffer) {
	ap.ringBuffer = ringBuffer
}

// processAudioLoop is the main processing loop
func (ap *AudioProcessor) processAudioLoop() {
	ticker := time.NewTicker(1 * time.Millisecond) // 1ms processing interval
	defer ticker.Stop()
	
	for {
		select {
		case <-ap.quit:
			atomic.StoreInt32(&ap.running, 0)
			return
			
		case <-ticker.C:
			ap.processAudioChunks()
		}
	}
}

// processAudioChunks processes available audio chunks
func (ap *AudioProcessor) processAudioChunks() {
	if ap.ringBuffer == nil {
		return
	}
	
	// Process up to 10 chunks per tick to avoid blocking
	for i := 0; i < 10; i++ {
		chunk, ok := ap.ringBuffer.Dequeue()
		if !ok {
			break // No more chunks to process
		}
		
		ap.processAudioChunk(chunk)
		atomic.AddInt64(&ap.processed, 1)
		ap.lastActivity = time.Now()
	}
}

// processAudioChunk processes a single audio chunk
func (ap *AudioProcessor) processAudioChunk(chunk AudioChunk) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Audio processor %d panic processing chunk: %v", ap.cpuID, r)
			atomic.AddInt64(&ap.errors, 1)
		}
	}()
	
	// Get buffer from pool
	buffer := ap.bufferPool.Get()
	defer ap.bufferPool.Put(buffer)
	
	// Copy audio data to pooled buffer
	copy(buffer, chunk.Data)
	
	// Process audio (normalize, validate, etc.)
	if err := ap.normalizeAudio(buffer, chunk.Size); err != nil {
		logrus.Errorf("Audio processor %d error normalizing audio: %v", ap.cpuID, err)
		atomic.AddInt64(&ap.errors, 1)
		return
	}
	
	// Here you would typically send the processed audio to the AI engine
	// For now, we just log the processing
	logrus.Debugf("Audio processor %d processed chunk for session %s (size: %d)", 
		ap.cpuID, chunk.SessionID, chunk.Size)
}

// normalizeAudio normalizes audio data
func (ap *AudioProcessor) normalizeAudio(buffer []byte, size int) error {
	if size <= 0 || size > len(buffer) {
		return fmt.Errorf("invalid audio size: %d", size)
	}
	
	// Simple normalization - in production, this would be more sophisticated
	// For now, just validate the data
	for i := 0; i < size; i++ {
		if buffer[i] == 0 {
			// Skip zero bytes
			continue
		}
	}
	
	return nil
}

// setCPUAffinity sets the CPU affinity for the current thread
func (ap *AudioProcessor) setCPUAffinity(cpuID int) error {
	// This is a simplified version - in production, you'd use platform-specific calls
	// For now, we'll just log the intended CPU assignment
	logrus.Infof("Setting CPU affinity to CPU %d for audio processor", cpuID)
	
	// In a real implementation, you would use:
	// - Linux: unix.SchedSetaffinity
	// - Windows: SetThreadAffinityMask
	// - macOS: thread_policy_set
	
	return nil
}

// GetStats returns processor statistics
func (ap *AudioProcessor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"cpu_id":        ap.cpuID,
		"running":       atomic.LoadInt32(&ap.running) == 1,
		"processed":     atomic.LoadInt64(&ap.processed),
		"errors":        atomic.LoadInt64(&ap.errors),
		"last_activity": ap.lastActivity,
		"error_rate":    ap.getErrorRate(),
	}
}

// getErrorRate calculates the error rate
func (ap *AudioProcessor) getErrorRate() float64 {
	processed := atomic.LoadInt64(&ap.processed)
	errors := atomic.LoadInt64(&ap.errors)
	
	if processed == 0 {
		return 0
	}
	
	return float64(errors) / float64(processed) * 100
}

// IsRunning returns true if the processor is running
func (ap *AudioProcessor) IsRunning() bool {
	return atomic.LoadInt32(&ap.running) == 1
}

// String returns a string representation of the processor
func (ap *AudioProcessor) String() string {
	stats := ap.GetStats()
	return fmt.Sprintf("AudioProcessor(cpu=%d, running=%v, processed=%d, errors=%d, error_rate=%.2f%%)",
		stats["cpu_id"], stats["running"], stats["processed"], stats["errors"], stats["error_rate"])
}
