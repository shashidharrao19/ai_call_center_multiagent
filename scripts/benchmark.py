#!/usr/bin/env python3
"""
AI Call Center Performance Benchmarking Suite
"""

import asyncio
import time
import json
import statistics
import websockets
import base64
import numpy as np
import argparse
import logging
from typing import List, Dict, Any
from dataclasses import dataclass
import aiohttp
import concurrent.futures

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

@dataclass
class BenchmarkResult:
    """Benchmark result for a single measurement"""
    operation: str
    duration: float
    success: bool
    error: str = None
    metadata: Dict[str, Any] = None

@dataclass
class LatencyStats:
    """Latency statistics"""
    p50: float
    p95: float
    p99: float
    mean: float
    min: float
    max: float
    count: int

class PerformanceBenchmark:
    """Performance benchmarking suite"""
    
    def __init__(self, backend_url: str = "ws://localhost:8080/ws", 
                 ai_engine_url: str = "http://localhost:8000"):
        self.backend_url = backend_url
        self.ai_engine_url = ai_engine_url
        self.results: List[BenchmarkResult] = []
        
    def add_result(self, result: BenchmarkResult):
        """Add a benchmark result"""
        self.results.append(result)
    
    def get_latency_stats(self, operation: str) -> LatencyStats:
        """Get latency statistics for an operation"""
        durations = [r.duration for r in self.results if r.operation == operation and r.success]
        
        if not durations:
            return LatencyStats(0, 0, 0, 0, 0, 0, 0)
        
        durations.sort()
        n = len(durations)
        
        return LatencyStats(
            p50=durations[int(n * 0.50)],
            p95=durations[int(n * 0.95)],
            p99=durations[int(n * 0.99)],
            mean=statistics.mean(durations),
            min=min(durations),
            max=max(durations),
            count=n
        )
    
    def generate_report(self) -> Dict[str, Any]:
        """Generate comprehensive performance report"""
        report = {
            "timestamp": time.time(),
            "summary": {
                "total_operations": len(self.results),
                "successful_operations": len([r for r in self.results if r.success]),
                "failed_operations": len([r for r in self.results if not r.success]),
                "success_rate": len([r for r in self.results if r.success]) / max(len(self.results), 1)
            },
            "latency_metrics": {}
        }
        
        # Get unique operations
        operations = set(r.operation for r in self.results)
        
        for operation in operations:
            stats = self.get_latency_stats(operation)
            report["latency_metrics"][operation] = {
                "p50_ms": stats.p50 * 1000,
                "p95_ms": stats.p95 * 1000,
                "p99_ms": stats.p99 * 1000,
                "mean_ms": stats.mean * 1000,
                "min_ms": stats.min * 1000,
                "max_ms": stats.max * 1000,
                "count": stats.count
            }
        
        return report
    
    async def benchmark_websocket_connection(self, num_connections: int = 10) -> List[BenchmarkResult]:
        """Benchmark WebSocket connection establishment"""
        results = []
        
        async def connect_and_measure():
            start_time = time.time()
            try:
                async with websockets.connect(self.backend_url) as websocket:
                    duration = time.time() - start_time
                    results.append(BenchmarkResult(
                        operation="websocket_connect",
                        duration=duration,
                        success=True
                    ))
                    
                    # Wait a bit then close
                    await asyncio.sleep(0.1)
                    
            except Exception as e:
                duration = time.time() - start_time
                results.append(BenchmarkResult(
                    operation="websocket_connect",
                    duration=duration,
                    success=False,
                    error=str(e)
                ))
        
        # Create multiple connections concurrently
        tasks = [connect_and_measure() for _ in range(num_connections)]
        await asyncio.gather(*tasks, return_exceptions=True)
        
        return results
    
    async def benchmark_websocket_message_roundtrip(self, num_messages: int = 100) -> List[BenchmarkResult]:
        """Benchmark WebSocket message roundtrip latency"""
        results = []
        
        async def send_message_and_measure():
            try:
                async with websockets.connect(self.backend_url) as websocket:
                    # Wait for session message
                    await websocket.recv()
                    
                    for _ in range(num_messages):
                        start_time = time.time()
                        
                        # Send test message
                        message = {
                            "type": "text",
                            "data": {"text": "Hello, this is a test message"}
                        }
                        await websocket.send(json.dumps(message))
                        
                        # Wait for response
                        response = await websocket.recv()
                        
                        duration = time.time() - start_time
                        results.append(BenchmarkResult(
                            operation="websocket_roundtrip",
                            duration=duration,
                            success=True,
                            metadata={"message_size": len(json.dumps(message))}
                        ))
                        
            except Exception as e:
                results.append(BenchmarkResult(
                    operation="websocket_roundtrip",
                    duration=0,
                    success=False,
                    error=str(e)
                ))
        
        await send_message_and_measure()
        return results
    
    async def benchmark_audio_processing(self, num_chunks: int = 50) -> List[BenchmarkResult]:
        """Benchmark audio processing latency"""
        results = []
        
        # Generate test audio data (1 second of 24kHz mono audio)
        sample_rate = 24000
        duration = 1.0
        samples = int(sample_rate * duration)
        audio_data = np.random.randint(-32768, 32767, samples, dtype=np.int16)
        audio_base64 = base64.b64encode(audio_data.tobytes()).decode('utf-8')
        
        async def process_audio_chunk():
            try:
                async with websockets.connect(self.backend_url) as websocket:
                    # Wait for session message
                    await websocket.recv()
                    
                    for _ in range(num_chunks):
                        start_time = time.time()
                        
                        # Send audio message
                        message = {
                            "type": "audio",
                            "data": {
                                "data": audio_base64,
                                "format": "PCM",
                                "timestamp": int(time.time() * 1000)
                            }
                        }
                        await websocket.send(json.dumps(message))
                        
                        # Wait for response
                        response = await websocket.recv()
                        
                        duration = time.time() - start_time
                        results.append(BenchmarkResult(
                            operation="audio_processing",
                            duration=duration,
                            success=True,
                            metadata={"audio_size": len(audio_base64)}
                        ))
                        
            except Exception as e:
                results.append(BenchmarkResult(
                    operation="audio_processing",
                    duration=0,
                    success=False,
                    error=str(e)
                ))
        
        await process_audio_chunk()
        return results
    
    async def benchmark_ai_engine_rpc(self, num_requests: int = 100) -> List[BenchmarkResult]:
        """Benchmark AI engine RPC calls"""
        results = []
        
        async def make_rpc_request():
            try:
                async with aiohttp.ClientSession() as session:
                    for _ in range(num_requests):
                        start_time = time.time()
                        
                        # Prepare request
                        request_data = {
                            "session_id": f"benchmark_{int(time.time())}",
                            "message_type": "text",
                            "data": {"text": "Hello, this is a benchmark test"},
                            "audio_config": {
                                "sample_rate": 24000,
                                "channels": 1,
                                "format": "PCM"
                            }
                        }
                        
                        # Make request
                        async with session.post(
                            f"{self.ai_engine_url}/process",
                            json=request_data
                        ) as response:
                            await response.text()
                        
                        duration = time.time() - start_time
                        results.append(BenchmarkResult(
                            operation="ai_engine_rpc",
                            duration=duration,
                            success=response.status == 200,
                            error=None if response.status == 200 else f"HTTP {response.status}"
                        ))
                        
            except Exception as e:
                results.append(BenchmarkResult(
                    operation="ai_engine_rpc",
                    duration=0,
                    success=False,
                    error=str(e)
                ))
        
        await make_rpc_request()
        return results
    
    async def benchmark_gemini_api_simulation(self, num_requests: int = 50) -> List[BenchmarkResult]:
        """Benchmark Gemini API simulation (since we can't call real API in benchmark)"""
        results = []
        
        async def simulate_gemini_request():
            for _ in range(num_requests):
                start_time = time.time()
                
                # Simulate Gemini API call latency (50-200ms)
                await asyncio.sleep(0.05 + np.random.random() * 0.15)
                
                duration = time.time() - start_time
                results.append(BenchmarkResult(
                    operation="gemini_api_simulation",
                    duration=duration,
                    success=True,
                    metadata={"simulated": True}
                ))
        
        await simulate_gemini_request()
        return results
    
    async def benchmark_concurrent_load(self, num_concurrent: int = 50, 
                                      duration_seconds: int = 30) -> List[BenchmarkResult]:
        """Benchmark system under concurrent load"""
        results = []
        end_time = time.time() + duration_seconds
        
        async def concurrent_worker(worker_id: int):
            worker_results = []
            try:
                async with websockets.connect(self.backend_url) as websocket:
                    # Wait for session message
                    await websocket.recv()
                    
                    while time.time() < end_time:
                        start_time = time.time()
                        
                        # Send mixed message types
                        if np.random.random() < 0.7:  # 70% audio, 30% text
                            # Audio message
                            audio_data = np.random.randint(-32768, 32767, 1024, dtype=np.int16)
                            audio_base64 = base64.b64encode(audio_data.tobytes()).decode('utf-8')
                            message = {
                                "type": "audio",
                                "data": {
                                    "data": audio_base64,
                                    "format": "PCM",
                                    "timestamp": int(time.time() * 1000)
                                }
                            }
                        else:
                            # Text message
                            message = {
                                "type": "text",
                                "data": {"text": f"Worker {worker_id} message"}
                            }
                        
                        await websocket.send(json.dumps(message))
                        
                        # Wait for response
                        response = await websocket.recv()
                        
                        duration = time.time() - start_time
                        worker_results.append(BenchmarkResult(
                            operation="concurrent_load",
                            duration=duration,
                            success=True,
                            metadata={"worker_id": worker_id}
                        ))
                        
                        # Small delay between messages
                        await asyncio.sleep(0.1)
                        
            except Exception as e:
                worker_results.append(BenchmarkResult(
                    operation="concurrent_load",
                    duration=0,
                    success=False,
                    error=str(e),
                    metadata={"worker_id": worker_id}
                ))
            
            return worker_results
        
        # Start concurrent workers
        tasks = [concurrent_worker(i) for i in range(num_concurrent)]
        worker_results = await asyncio.gather(*tasks, return_exceptions=True)
        
        # Flatten results
        for worker_result in worker_results:
            if isinstance(worker_result, list):
                results.extend(worker_result)
        
        return results
    
    async def run_full_benchmark(self) -> Dict[str, Any]:
        """Run complete benchmark suite"""
        logger.info("Starting full benchmark suite...")
        
        # Clear previous results
        self.results = []
        
        # Run individual benchmarks
        logger.info("Benchmarking WebSocket connections...")
        connection_results = await self.benchmark_websocket_connection(20)
        self.results.extend(connection_results)
        
        logger.info("Benchmarking WebSocket message roundtrip...")
        roundtrip_results = await self.benchmark_websocket_message_roundtrip(100)
        self.results.extend(roundtrip_results)
        
        logger.info("Benchmarking audio processing...")
        audio_results = await self.benchmark_audio_processing(50)
        self.results.extend(audio_results)
        
        logger.info("Benchmarking AI engine RPC...")
        rpc_results = await self.benchmark_ai_engine_rpc(100)
        self.results.extend(rpc_results)
        
        logger.info("Benchmarking Gemini API simulation...")
        gemini_results = await self.benchmark_gemini_api_simulation(50)
        self.results.extend(gemini_results)
        
        logger.info("Benchmarking concurrent load...")
        load_results = await self.benchmark_concurrent_load(30, 20)
        self.results.extend(load_results)
        
        # Generate report
        report = self.generate_report()
        
        logger.info("Benchmark completed!")
        return report

async def main():
    """Main benchmark function"""
    parser = argparse.ArgumentParser(description="AI Call Center Performance Benchmark")
    parser.add_argument("--backend-url", default="ws://localhost:8080/ws",
                       help="Backend WebSocket URL")
    parser.add_argument("--ai-engine-url", default="http://localhost:8000",
                       help="AI Engine HTTP URL")
    parser.add_argument("--output", default="benchmark_report.json",
                       help="Output file for benchmark report")
    parser.add_argument("--verbose", action="store_true",
                       help="Verbose output")
    
    args = parser.parse_args()
    
    if args.verbose:
        logging.getLogger().setLevel(logging.DEBUG)
    
    # Create benchmark instance
    benchmark = PerformanceBenchmark(args.backend_url, args.ai_engine_url)
    
    # Run benchmark
    report = await benchmark.run_full_benchmark()
    
    # Save report
    with open(args.output, 'w') as f:
        json.dump(report, f, indent=2)
    
    # Print summary
    print("\n" + "="*60)
    print("BENCHMARK SUMMARY")
    print("="*60)
    print(f"Total Operations: {report['summary']['total_operations']}")
    print(f"Success Rate: {report['summary']['success_rate']:.2%}")
    print(f"Failed Operations: {report['summary']['failed_operations']}")
    
    print("\nLATENCY METRICS (ms):")
    print("-" * 40)
    for operation, metrics in report['latency_metrics'].items():
        print(f"{operation}:")
        print(f"  P50: {metrics['p50_ms']:.2f}ms")
        print(f"  P95: {metrics['p95_ms']:.2f}ms")
        print(f"  P99: {metrics['p99_ms']:.2f}ms")
        print(f"  Mean: {metrics['mean_ms']:.2f}ms")
        print(f"  Count: {metrics['count']}")
        print()
    
    print(f"Detailed report saved to: {args.output}")

if __name__ == "__main__":
    asyncio.run(main())
