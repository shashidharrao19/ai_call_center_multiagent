package optimization

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// AudioBufferPool provides a memory pool for audio buffers to reduce GC pressure
type AudioBufferPool struct {
	pool      sync.Pool
	size      int
	created   int64
	reused    int64
	allocated int64
	metrics   *PoolMetrics
}

// PoolMetrics tracks memory pool performance
type PoolMetrics struct {
	Created    int64 `json:"created"`
	Reused     int64 `json:"reused"`
	Allocated  int64 `json:"allocated"`
	HitRate    float64 `json:"hit_rate"`
	LastUpdate time.Time `json:"last_update"`
}

// NewAudioBufferPool creates a new audio buffer pool
func NewAudioBufferPool(bufferSize int) *AudioBufferPool {
	pool := &AudioBufferPool{
		size: bufferSize,
		metrics: &PoolMetrics{
			LastUpdate: time.Now(),
		},
	}
	
	pool.pool = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.created, 1)
			atomic.AddInt64(&pool.allocated, 1)
			return make([]byte, bufferSize)
		},
	}
	
	logrus.Infof("Created audio buffer pool with size: %d bytes", bufferSize)
	return pool
}

// Get retrieves a buffer from the pool
func (p *AudioBufferPool) Get() []byte {
	buffer := p.pool.Get().([]byte)
	
	// Check if we're reusing a buffer
	if len(buffer) == p.size {
		atomic.AddInt64(&p.reused, 1)
	} else {
		// This shouldn't happen, but handle it gracefully
		logrus.Warnf("Retrieved buffer with unexpected size: %d, expected: %d", len(buffer), p.size)
		atomic.AddInt64(&p.allocated, 1)
	}
	
	return buffer
}

// Put returns a buffer to the pool
func (p *AudioBufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}
	
	// Only put back buffers of the correct size
	if len(buf) == p.size {
		// Reset the slice to full capacity
		buf = buf[:p.size]
		p.pool.Put(buf)
	} else {
		// Buffer size mismatch, don't put it back
		logrus.Warnf("Attempted to put buffer with size %d into pool expecting size %d", len(buf), p.size)
	}
}

// GetMetrics returns current pool metrics
func (p *AudioBufferPool) GetMetrics() *PoolMetrics {
	created := atomic.LoadInt64(&p.created)
	reused := atomic.LoadInt64(&p.reused)
	allocated := atomic.LoadInt64(&p.allocated)
	
	hitRate := float64(0)
	if allocated > 0 {
		hitRate = float64(reused) / float64(allocated) * 100
	}
	
	return &PoolMetrics{
		Created:    created,
		Reused:     reused,
		Allocated:  allocated,
		HitRate:    hitRate,
		LastUpdate: time.Now(),
	}
}

// Reset resets the pool metrics
func (p *AudioBufferPool) Reset() {
	atomic.StoreInt64(&p.created, 0)
	atomic.StoreInt64(&p.reused, 0)
	atomic.StoreInt64(&p.allocated, 0)
	p.metrics.LastUpdate = time.Now()
	logrus.Info("Audio buffer pool metrics reset")
}

// Size returns the buffer size for this pool
func (p *AudioBufferPool) Size() int {
	return p.size
}

// String returns a string representation of the pool
func (p *AudioBufferPool) String() string {
	metrics := p.GetMetrics()
	return fmt.Sprintf("AudioBufferPool(size=%d, created=%d, reused=%d, hit_rate=%.2f%%)",
		p.size, metrics.Created, metrics.Reused, metrics.HitRate)
}
