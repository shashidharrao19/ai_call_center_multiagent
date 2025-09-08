package optimization

import (
	"testing"
	"time"
)

func TestAudioBufferPool(t *testing.T) {
	pool := NewAudioBufferPool(1024)
	
	// Test getting and putting buffers
	buffer1 := pool.Get()
	if len(buffer1) != 1024 {
		t.Errorf("Expected buffer size 1024, got %d", len(buffer1))
	}
	
	// Test reusing buffers
	pool.Put(buffer1)
	buffer2 := pool.Get()
	if len(buffer2) != 1024 {
		t.Errorf("Expected buffer size 1024, got %d", len(buffer2))
	}
	
	// Test metrics
	metrics := pool.GetMetrics()
	if metrics.Created == 0 {
		t.Error("Expected at least one buffer to be created")
	}
	
	t.Logf("Buffer pool metrics: %+v", metrics)
}

func TestLockFreeRingBuffer(t *testing.T) {
	rb := NewLockFreeRingBuffer(8) // Power of 2
	
	// Test enqueue/dequeue
	chunk := AudioChunk{
		Data:      []byte("test data"),
		Timestamp: time.Now().UnixNano(),
		SessionID: "test-session",
		Format:    "PCM",
	}
	
	// Enqueue
	if !rb.Enqueue(chunk) {
		t.Error("Failed to enqueue chunk")
	}
	
	// Dequeue
	retrievedChunk, ok := rb.Dequeue()
	if !ok {
		t.Error("Failed to dequeue chunk")
	}
	
	if retrievedChunk.SessionID != chunk.SessionID {
		t.Errorf("Expected session ID %s, got %s", chunk.SessionID, retrievedChunk.SessionID)
	}
	
	// Test buffer full
	for i := 0; i < 8; i++ {
		rb.Enqueue(chunk)
	}
	
	// Should fail to enqueue when full
	if rb.Enqueue(chunk) {
		t.Error("Expected enqueue to fail when buffer is full")
	}
	
	// Test metrics
	metrics := rb.GetMetrics()
	if metrics.Usage == 0 {
		t.Error("Expected buffer usage to be > 0")
	}
	
	t.Logf("Ring buffer metrics: %+v", metrics)
}

func TestBackpressureHandler(t *testing.T) {
	handler := NewBackpressureHandler(5, "drop")
	
	// Test normal message handling
	err := handler.HandleMessage("test message")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Test backpressure (fill queue)
	for i := 0; i < 5; i++ {
		handler.HandleMessage("test message")
	}
	
	// Should trigger backpressure
	err = handler.HandleMessage("test message")
	if err == nil {
		t.Error("Expected backpressure error")
	}
	
	// Test metrics
	metrics := handler.GetMetrics()
	if metrics.DroppedCount == 0 {
		t.Error("Expected dropped count to be > 0")
	}
	
	t.Logf("Backpressure metrics: %+v", metrics)
}

func TestAudioProcessor(t *testing.T) {
	pool := NewAudioBufferPool(1024)
	processor := NewAudioProcessor(0, pool)
	
	// Test processor stats
	stats := processor.GetStats()
	if stats["cpu_id"] != 0 {
		t.Errorf("Expected CPU ID 0, got %v", stats["cpu_id"])
	}
	
	if stats["running"] != false {
		t.Error("Expected processor to not be running initially")
	}
	
	// Test starting processor
	processor.Start()
	time.Sleep(100 * time.Millisecond) // Let it start
	
	// Test stopping processor
	processor.Stop()
	time.Sleep(100 * time.Millisecond) // Let it stop
	
	t.Logf("Audio processor stats: %+v", stats)
}

func TestOptimizationManager(t *testing.T) {
	// Create a mock config
	config := &struct {
		AudioBufferPoolSize int
		RingBufferSize      int
		AudioCPUStart       int
		AudioCPUCount       int
		MaxQueueSize        int
		BackpressureStrategy string
	}{
		AudioBufferPoolSize: 1024,
		RingBufferSize:      8,
		AudioCPUStart:       0,
		AudioCPUCount:       1,
		MaxQueueSize:        5,
		BackpressureStrategy: "drop",
	}
	
	// Note: This test would need the actual config package
	// For now, we'll test individual components
	t.Log("Optimization manager test would require config package integration")
}

func BenchmarkAudioBufferPool(b *testing.B) {
	pool := NewAudioBufferPool(1024)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer := pool.Get()
			pool.Put(buffer)
		}
	})
}

func BenchmarkLockFreeRingBuffer(b *testing.B) {
	rb := NewLockFreeRingBuffer(1024)
	chunk := AudioChunk{
		Data:      make([]byte, 1024),
		Timestamp: time.Now().UnixNano(),
		SessionID: "bench-session",
		Format:    "PCM",
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rb.Enqueue(chunk)
			rb.Dequeue()
		}
	})
}

func BenchmarkBackpressureHandler(b *testing.B) {
	handler := NewBackpressureHandler(1000, "drop")
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			handler.HandleMessage("test message")
		}
	})
}
