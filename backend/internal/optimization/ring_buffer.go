package optimization

import (
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"
)

// AudioChunk represents a chunk of audio data
type AudioChunk struct {
	Data      []byte
	Timestamp int64
	SessionID string
	Format    string
	Size      int
}

// LockFreeRingBuffer provides a lock-free ring buffer for audio streaming
type LockFreeRingBuffer struct {
	buffer    []AudioChunk
	head      uint64  // Write position
	tail      uint64  // Read position
	mask      uint64  // Size mask for modulo operation
	size      uint64  // Buffer size (must be power of 2)
	overflow  int64   // Number of overflows
	underflow int64   // Number of underflows
	metrics   *RingBufferMetrics
}

// RingBufferMetrics tracks ring buffer performance
type RingBufferMetrics struct {
	Size      uint64  `json:"size"`
	Head      uint64  `json:"head"`
	Tail      uint64  `json:"tail"`
	Usage     float64 `json:"usage_percent"`
	Overflow  int64   `json:"overflow_count"`
	Underflow int64   `json:"underflow_count"`
	LastUpdate time.Time `json:"last_update"`
}

// NewLockFreeRingBuffer creates a new lock-free ring buffer
func NewLockFreeRingBuffer(size int) *LockFreeRingBuffer {
	// Ensure size is power of 2 for efficient modulo
	if size&(size-1) != 0 {
		size = nextPowerOfTwo(size)
		logrus.Infof("Adjusted ring buffer size to power of 2: %d", size)
	}
	
	rb := &LockFreeRingBuffer{
		buffer:  make([]AudioChunk, size),
		mask:    uint64(size - 1),
		size:    uint64(size),
		metrics: &RingBufferMetrics{
			Size: uint64(size),
			LastUpdate: time.Now(),
		},
	}
	
	logrus.Infof("Created lock-free ring buffer with size: %d", size)
	return rb
}

// Enqueue adds an audio chunk to the buffer
func (rb *LockFreeRingBuffer) Enqueue(chunk AudioChunk) bool {
	head := atomic.LoadUint64(&rb.head)
	tail := atomic.LoadUint64(&rb.tail)
	
	// Check if buffer is full
	if head-tail >= rb.size {
		atomic.AddInt64(&rb.overflow, 1)
		return false // Buffer full
	}
	
	// Set chunk size
	chunk.Size = len(chunk.Data)
	
	// Write to buffer
	rb.buffer[head&rb.mask] = chunk
	
	// Update head position
	atomic.StoreUint64(&rb.head, head+1)
	
	return true
}

// Dequeue removes an audio chunk from the buffer
func (rb *LockFreeRingBuffer) Dequeue() (AudioChunk, bool) {
	tail := atomic.LoadUint64(&rb.tail)
	head := atomic.LoadUint64(&rb.head)
	
	// Check if buffer is empty
	if tail >= head {
		atomic.AddInt64(&rb.underflow, 1)
		return AudioChunk{}, false // Buffer empty
	}
	
	// Read from buffer
	chunk := rb.buffer[tail&rb.mask]
	
	// Update tail position
	atomic.StoreUint64(&rb.tail, tail+1)
	
	return chunk, true
}

// TryEnqueue attempts to enqueue without blocking
func (rb *LockFreeRingBuffer) TryEnqueue(chunk AudioChunk) bool {
	return rb.Enqueue(chunk)
}

// TryDequeue attempts to dequeue without blocking
func (rb *LockFreeRingBuffer) TryDequeue() (AudioChunk, bool) {
	return rb.Dequeue()
}

// IsEmpty checks if the buffer is empty
func (rb *LockFreeRingBuffer) IsEmpty() bool {
	tail := atomic.LoadUint64(&rb.tail)
	head := atomic.LoadUint64(&rb.head)
	return tail >= head
}

// IsFull checks if the buffer is full
func (rb *LockFreeRingBuffer) IsFull() bool {
	tail := atomic.LoadUint64(&rb.tail)
	head := atomic.LoadUint64(&rb.head)
	return head-tail >= rb.size
}

// Size returns the current number of items in the buffer
func (rb *LockFreeRingBuffer) Size() uint64 {
	tail := atomic.LoadUint64(&rb.tail)
	head := atomic.LoadUint64(&rb.head)
	if head >= tail {
		return head - tail
	}
	return 0
}

// Capacity returns the maximum capacity of the buffer
func (rb *LockFreeRingBuffer) Capacity() uint64 {
	return rb.size
}

// Usage returns the buffer usage as a percentage
func (rb *LockFreeRingBuffer) Usage() float64 {
	size := rb.Size()
	return float64(size) / float64(rb.size) * 100
}

// GetMetrics returns current buffer metrics
func (rb *LockFreeRingBuffer) GetMetrics() *RingBufferMetrics {
	head := atomic.LoadUint64(&rb.head)
	tail := atomic.LoadUint64(&rb.tail)
	overflow := atomic.LoadInt64(&rb.overflow)
	underflow := atomic.LoadInt64(&rb.underflow)
	
	usage := float64(0)
	if head >= tail {
		usage = float64(head-tail) / float64(rb.size) * 100
	}
	
	return &RingBufferMetrics{
		Size:      rb.size,
		Head:      head,
		Tail:      tail,
		Usage:     usage,
		Overflow:  overflow,
		Underflow: underflow,
		LastUpdate: time.Now(),
	}
}

// Reset resets the buffer and metrics
func (rb *LockFreeRingBuffer) Reset() {
	atomic.StoreUint64(&rb.head, 0)
	atomic.StoreUint64(&rb.tail, 0)
	atomic.StoreInt64(&rb.overflow, 0)
	atomic.StoreInt64(&rb.underflow, 0)
	rb.metrics.LastUpdate = time.Now()
	logrus.Info("Lock-free ring buffer reset")
}

// String returns a string representation of the buffer
func (rb *LockFreeRingBuffer) String() string {
	metrics := rb.GetMetrics()
	return fmt.Sprintf("LockFreeRingBuffer(size=%d, usage=%.2f%%, overflow=%d, underflow=%d)",
		rb.size, metrics.Usage, metrics.Overflow, metrics.Underflow)
}

// nextPowerOfTwo returns the next power of 2 greater than or equal to n
func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}
	
	// Find the next power of 2
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	
	return n
}

// MemoryBarrier ensures memory ordering for lock-free operations
func MemoryBarrier() {
	// Use a volatile read to ensure memory ordering
	_ = *(*int)(unsafe.Pointer(&struct{}{}))
}
