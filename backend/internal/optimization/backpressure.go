package optimization

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// BackpressureHandler handles backpressure to prevent system overload
type BackpressureHandler struct {
	maxQueueSize    int
	queue           chan interface{}
	dropped         int64
	throttled       int64
	rejected        int64
	strategy        string
	circuitBreaker  *CircuitBreaker
	metrics         *BackpressureMetrics
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	maxFailures    int
	failures       int64
	lastFailure    time.Time
	timeout        time.Duration
	state          CircuitState
}

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// BackpressureMetrics tracks backpressure performance
type BackpressureMetrics struct {
	DroppedCount    int64     `json:"dropped_count"`
	ThrottledCount  int64     `json:"throttled_count"`
	RejectedCount   int64     `json:"rejected_count"`
	QueueSize       int       `json:"queue_size"`
	CircuitState    string    `json:"circuit_state"`
	LastUpdate      time.Time `json:"last_update"`
}

// NewBackpressureHandler creates a new backpressure handler
func NewBackpressureHandler(maxQueueSize int, strategy string) *BackpressureHandler {
	bh := &BackpressureHandler{
		maxQueueSize: maxQueueSize,
		queue:        make(chan interface{}, maxQueueSize),
		strategy:     strategy,
		circuitBreaker: &CircuitBreaker{
			maxFailures: 10,
			timeout:     30 * time.Second,
			state:       StateClosed,
		},
		metrics: &BackpressureMetrics{
			LastUpdate: time.Now(),
		},
	}
	
	logrus.Infof("Created backpressure handler with strategy: %s, max queue size: %d", strategy, maxQueueSize)
	return bh
}

// HandleMessage handles a message with backpressure
func (bh *BackpressureHandler) HandleMessage(msg interface{}) error {
	// Check circuit breaker
	if !bh.circuitBreaker.CanExecute() {
		atomic.AddInt64(&bh.rejected, 1)
		return errors.New("circuit breaker open")
	}
	
	select {
	case bh.queue <- msg:
		// Message queued successfully
		return nil
	default:
		// Queue full - implement backpressure
		return bh.handleBackpressure(msg)
	}
}

// handleBackpressure implements backpressure strategies
func (bh *BackpressureHandler) handleBackpressure(msg interface{}) error {
	// Record failure for circuit breaker
	bh.circuitBreaker.RecordFailure()
	
	switch bh.strategy {
	case "drop":
		return bh.dropMessage(msg)
	case "throttle":
		return bh.throttleMessage(msg)
	case "reject":
		return bh.rejectMessage(msg)
	default:
		return errors.New("unknown backpressure strategy")
	}
}

// dropMessage drops the message
func (bh *BackpressureHandler) dropMessage(msg interface{}) error {
	atomic.AddInt64(&bh.dropped, 1)
	logrus.Debugf("Dropped message due to backpressure")
	return errors.New("message dropped due to backpressure")
}

// throttleMessage implements exponential backoff
func (bh *BackpressureHandler) throttleMessage(msg interface{}) error {
	atomic.AddInt64(&bh.throttled, 1)
	
	// Implement exponential backoff
	backoff := time.Duration(bh.getDroppedCount()) * 100 * time.Millisecond
	if backoff > 5*time.Second {
		backoff = 5 * time.Second
	}
	
	logrus.Debugf("Throttling message with backoff: %v", backoff)
	time.Sleep(backoff)
	
	// Retry with backoff
	select {
	case bh.queue <- msg:
		return nil
	default:
		atomic.AddInt64(&bh.dropped, 1)
		return errors.New("message dropped after backoff")
	}
}

// rejectMessage rejects the message
func (bh *BackpressureHandler) rejectMessage(msg interface{}) error {
	atomic.AddInt64(&bh.rejected, 1)
	logrus.Debugf("Rejected message due to backpressure")
	return errors.New("system overloaded, rejecting message")
}

// CanExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = StateHalfOpen
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordFailure records a failure for the circuit breaker
func (cb *CircuitBreaker) RecordFailure() {
	atomic.AddInt64(&cb.failures, 1)
	cb.lastFailure = time.Now()
	
	if atomic.LoadInt64(&cb.failures) >= int64(cb.maxFailures) {
		cb.state = StateOpen
		logrus.Warnf("Circuit breaker opened due to %d failures", cb.maxFailures)
	}
}

// RecordSuccess records a success for the circuit breaker
func (cb *CircuitBreaker) RecordSuccess() {
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		atomic.StoreInt64(&cb.failures, 0)
		logrus.Info("Circuit breaker closed after successful operation")
	}
}

// GetDroppedCount returns the number of dropped messages
func (bh *BackpressureHandler) GetDroppedCount() int64 {
	return atomic.LoadInt64(&bh.dropped)
}

// GetThrottledCount returns the number of throttled messages
func (bh *BackpressureHandler) GetThrottledCount() int64 {
	return atomic.LoadInt64(&bh.throttled)
}

// GetRejectedCount returns the number of rejected messages
func (bh *BackpressureHandler) GetRejectedCount() int64 {
	return atomic.LoadInt64(&bh.rejected)
}

// GetQueueSize returns the current queue size
func (bh *BackpressureHandler) GetQueueSize() int {
	return len(bh.queue)
}

// GetMetrics returns current backpressure metrics
func (bh *BackpressureHandler) GetMetrics() *BackpressureMetrics {
	state := "closed"
	switch bh.circuitBreaker.state {
	case StateOpen:
		state = "open"
	case StateHalfOpen:
		state = "half-open"
	}
	
	return &BackpressureMetrics{
		DroppedCount:   atomic.LoadInt64(&bh.dropped),
		ThrottledCount: atomic.LoadInt64(&bh.throttled),
		RejectedCount:  atomic.LoadInt64(&bh.rejected),
		QueueSize:      len(bh.queue),
		CircuitState:   state,
		LastUpdate:     time.Now(),
	}
}

// Reset resets the backpressure handler
func (bh *BackpressureHandler) Reset() {
	atomic.StoreInt64(&bh.dropped, 0)
	atomic.StoreInt64(&bh.throttled, 0)
	atomic.StoreInt64(&bh.rejected, 0)
	
	// Clear queue
	for len(bh.queue) > 0 {
		select {
		case <-bh.queue:
		default:
			break
		}
	}
	
	// Reset circuit breaker
	bh.circuitBreaker.failures = 0
	bh.circuitBreaker.state = StateClosed
	
	bh.metrics.LastUpdate = time.Now()
	logrus.Info("Backpressure handler reset")
}

// String returns a string representation of the backpressure handler
func (bh *BackpressureHandler) String() string {
	metrics := bh.GetMetrics()
	return fmt.Sprintf("BackpressureHandler(strategy=%s, dropped=%d, throttled=%d, rejected=%d, queue_size=%d, circuit=%s)",
		bh.strategy, metrics.DroppedCount, metrics.ThrottledCount, metrics.RejectedCount, metrics.QueueSize, metrics.CircuitState)
}
