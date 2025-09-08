"""
Python AI Engine Profiling and Performance Monitoring
"""

import asyncio
import time
import psutil
import threading
from typing import Dict, List, Optional
from dataclasses import dataclass, field
from collections import deque
import statistics
import json
import logging
from datetime import datetime

logger = logging.getLogger(__name__)

@dataclass
class LatencyMeasurement:
    """Single latency measurement"""
    duration: float
    timestamp: float = field(default_factory=time.time)

@dataclass
class LatencyHistogram:
    """Latency histogram for percentile calculations"""
    measurements: deque = field(default_factory=lambda: deque(maxlen=10000))
    
    def record(self, duration: float):
        """Record a latency measurement"""
        self.measurements.append(LatencyMeasurement(duration))
    
    def get_percentiles(self) -> Dict[str, float]:
        """Get p50, p95, p99 percentiles"""
        if not self.measurements:
            return {"p50": 0.0, "p95": 0.0, "p99": 0.0}
        
        durations = [m.duration for m in self.measurements]
        durations.sort()
        
        n = len(durations)
        return {
            "p50": durations[int(n * 0.50)],
            "p95": durations[int(n * 0.95)],
            "p99": durations[int(n * 0.99)]
        }

@dataclass
class PerformanceMetrics:
    """Performance metrics collector"""
    # Gemini API metrics
    gemini_requests: int = 0
    gemini_errors: int = 0
    gemini_latency: LatencyHistogram = field(default_factory=LatencyHistogram)
    
    # Audio processing metrics
    audio_chunks_processed: int = 0
    audio_processing_latency: LatencyHistogram = field(default_factory=LatencyHistogram)
    
    # MCP metrics
    mcp_calls: int = 0
    mcp_errors: int = 0
    mcp_latency: LatencyHistogram = field(default_factory=LatencyHistogram)
    
    # System metrics
    cpu_usage: float = 0.0
    memory_usage: float = 0.0
    active_tasks: int = 0
    
    # Event loop metrics
    event_loop_latency: LatencyHistogram = field(default_factory=LatencyHistogram)
    blocking_calls: int = 0

class PythonProfiler:
    """Python AI Engine Profiler"""
    
    def __init__(self, enabled: bool = True):
        self.enabled = enabled
        self.metrics = PerformanceMetrics()
        self.start_time = time.time()
        self._lock = threading.Lock()
        self._running = False
        
    def start(self):
        """Start the profiler"""
        if not self.enabled:
            logger.info("Python profiler disabled")
            return
        
        self._running = True
        logger.info("Starting Python profiler")
        
        # Start system metrics collection
        asyncio.create_task(self._collect_system_metrics())
        
        # Start event loop monitoring
        asyncio.create_task(self._monitor_event_loop())
    
    def stop(self):
        """Stop the profiler"""
        self._running = False
        logger.info("Python profiler stopped")
    
    async def _collect_system_metrics(self):
        """Collect system metrics periodically"""
        while self._running:
            try:
                # Get CPU and memory usage
                self.metrics.cpu_usage = psutil.cpu_percent()
                self.metrics.memory_usage = psutil.virtual_memory().percent
                
                # Get active task count
                try:
                    loop = asyncio.get_running_loop()
                    self.metrics.active_tasks = len([t for t in asyncio.all_tasks(loop) if not t.done()])
                except:
                    self.metrics.active_tasks = 0
                
                await asyncio.sleep(5)  # Collect every 5 seconds
                
            except Exception as e:
                logger.error(f"Error collecting system metrics: {e}")
                await asyncio.sleep(5)
    
    async def _monitor_event_loop(self):
        """Monitor event loop performance"""
        while self._running:
            try:
                start_time = time.time()
                await asyncio.sleep(0.001)  # Small delay to measure loop latency
                loop_latency = (time.time() - start_time) * 1000  # Convert to ms
                
                self.metrics.event_loop_latency.record(loop_latency)
                
                await asyncio.sleep(1)  # Check every second
                
            except Exception as e:
                logger.error(f"Error monitoring event loop: {e}")
                await asyncio.sleep(1)
    
    def record_gemini_request(self, duration: float, success: bool = True):
        """Record Gemini API request"""
        with self._lock:
            self.metrics.gemini_requests += 1
            if not success:
                self.metrics.gemini_errors += 1
            self.metrics.gemini_latency.record(duration)
    
    def record_audio_processing(self, duration: float):
        """Record audio processing time"""
        with self._lock:
            self.metrics.audio_chunks_processed += 1
            self.metrics.audio_processing_latency.record(duration)
    
    def record_mcp_call(self, duration: float, success: bool = True):
        """Record MCP function call"""
        with self._lock:
            self.metrics.mcp_calls += 1
            if not success:
                self.metrics.mcp_errors += 1
            self.metrics.mcp_latency.record(duration)
    
    def record_blocking_call(self):
        """Record a blocking call detection"""
        with self._lock:
            self.metrics.blocking_calls += 1
    
    def get_performance_report(self) -> Dict:
        """Get comprehensive performance report"""
        with self._lock:
            uptime = time.time() - self.start_time
            
            return {
                "timestamp": datetime.now().isoformat(),
                "uptime_seconds": uptime,
                "gemini": {
                    "requests": self.metrics.gemini_requests,
                    "errors": self.metrics.gemini_errors,
                    "error_rate": self.metrics.gemini_errors / max(self.metrics.gemini_requests, 1),
                    "latency": self.metrics.gemini_latency.get_percentiles()
                },
                "audio_processing": {
                    "chunks_processed": self.metrics.audio_chunks_processed,
                    "latency": self.metrics.audio_processing_latency.get_percentiles()
                },
                "mcp": {
                    "calls": self.metrics.mcp_calls,
                    "errors": self.metrics.mcp_errors,
                    "error_rate": self.metrics.mcp_errors / max(self.metrics.mcp_calls, 1),
                    "latency": self.metrics.mcp_latency.get_percentiles()
                },
                "system": {
                    "cpu_usage": self.metrics.cpu_usage,
                    "memory_usage": self.metrics.memory_usage,
                    "active_tasks": self.metrics.active_tasks,
                    "blocking_calls": self.metrics.blocking_calls
                },
                "event_loop": {
                    "latency": self.metrics.event_loop_latency.get_percentiles()
                }
            }

# Global profiler instance
profiler = PythonProfiler()

def get_profiler() -> PythonProfiler:
    """Get the global profiler instance"""
    return profiler

# Decorator for measuring function execution time
def measure_time(metric_name: str):
    """Decorator to measure function execution time"""
    def decorator(func):
        async def async_wrapper(*args, **kwargs):
            start_time = time.time()
            try:
                result = await func(*args, **kwargs)
                duration = time.time() - start_time
                
                if metric_name == "gemini":
                    profiler.record_gemini_request(duration, True)
                elif metric_name == "audio":
                    profiler.record_audio_processing(duration)
                elif metric_name == "mcp":
                    profiler.record_mcp_call(duration, True)
                
                return result
            except Exception as e:
                duration = time.time() - start_time
                
                if metric_name == "gemini":
                    profiler.record_gemini_request(duration, False)
                elif metric_name == "mcp":
                    profiler.record_mcp_call(duration, False)
                
                raise e
        
        def sync_wrapper(*args, **kwargs):
            start_time = time.time()
            try:
                result = func(*args, **kwargs)
                duration = time.time() - start_time
                
                if metric_name == "gemini":
                    profiler.record_gemini_request(duration, True)
                elif metric_name == "audio":
                    profiler.record_audio_processing(duration)
                elif metric_name == "mcp":
                    profiler.record_mcp_call(duration, True)
                
                return result
            except Exception as e:
                duration = time.time() - start_time
                
                if metric_name == "gemini":
                    profiler.record_gemini_request(duration, False)
                elif metric_name == "mcp":
                    profiler.record_mcp_call(duration, False)
                
                raise e
        
        if asyncio.iscoroutinefunction(func):
            return async_wrapper
        else:
            return sync_wrapper
    
    return decorator

# Context manager for measuring code blocks
class MeasureTime:
    """Context manager for measuring execution time"""
    
    def __init__(self, metric_name: str, success_callback=None, error_callback=None):
        self.metric_name = metric_name
        self.success_callback = success_callback
        self.error_callback = error_callback
        self.start_time = None
    
    def __enter__(self):
        self.start_time = time.time()
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        duration = time.time() - self.start_time
        
        if exc_type is None:
            # Success
            if self.success_callback:
                self.success_callback(duration)
            else:
                if self.metric_name == "gemini":
                    profiler.record_gemini_request(duration, True)
                elif self.metric_name == "audio":
                    profiler.record_audio_processing(duration)
                elif self.metric_name == "mcp":
                    profiler.record_mcp_call(duration, True)
        else:
            # Error
            if self.error_callback:
                self.error_callback(duration)
            else:
                if self.metric_name == "gemini":
                    profiler.record_gemini_request(duration, False)
                elif self.metric_name == "mcp":
                    profiler.record_mcp_call(duration, False)
        
        return False  # Don't suppress exceptions
