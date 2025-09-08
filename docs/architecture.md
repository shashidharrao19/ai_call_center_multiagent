# AI Call Center Architecture

## Overview

The AI Call Center is a real-time, high-performance system built with a multi-language architecture to handle customer service calls using Google's Gemini 2.0 Live API. The system is designed for low latency, high concurrency, and seamless integration with existing MCP (Model Context Protocol) systems.

## Architecture Components

### 1. Go Backend (WebSocket Server)
**Location**: `backend/`
**Purpose**: High-performance WebSocket handling and session management

**Key Features**:
- Concurrent WebSocket connections (1000+)
- Session management with automatic cleanup
- Audio streaming and processing
- Load balancing and routing
- Real-time message handling

**Components**:
- `main.go`: Application entry point
- `internal/gateway/`: WebSocket connection handling
- `internal/websocket/`: WebSocket hub for message broadcasting
- `internal/session/`: Session lifecycle management
- `internal/audio/`: Audio processing utilities
- `internal/profiling/`: Performance profiling and metrics collection
- `pkg/models/`: Data structures and models

### 2. Python AI Engine
**Location**: `ai-engine/`
**Purpose**: AI processing with Gemini Live API and MCP integration

**Key Features**:
- Gemini 2.0 Live API integration
- Real-time bidirectional audio streaming
- MCP function calling for data access
- Conversation management
- Audio format conversion

**Components**:
- `main.py`: FastAPI server entry point
- `src/gemini/`: Gemini Live API client
- `src/mcp/`: MCP client for function calling
- `src/conversation/`: Conversation flow management
- `src/profiling/`: Performance profiling and metrics collection
- `src/observability/`: OpenTelemetry tracing and monitoring
- `config/`: Configuration management

### 3. JavaScript Frontend
**Location**: `frontend/`
**Purpose**: Browser-based audio interface

**Key Features**:
- Real-time audio capture and playback
- WebSocket client for backend communication
- Modern, responsive UI
- Audio visualization
- Session management

**Components**:
- `index.html`: Main application interface
- `src/main.js`: Application entry point
- `src/websocket/`: WebSocket client
- `src/audio/`: Audio capture and playback
- `src/ui/`: User interface management

## Data Flow

```
┌─────────────┐    WebSocket    ┌─────────────┐    HTTP/REST    ┌─────────────┐
│   Browser   │◄──────────────►│ Go Backend  │◄──────────────►│Python AI    │
│ (JavaScript)│                 │ (WebSocket) │                 │Engine       │
└─────────────┘                 └─────────────┘                 └─────────────┘
       │                               │                               │
       │ Audio Data                    │ Session Mgmt                  │ MCP Calls
       │ Text Messages                 │ Message Routing               │ Function Calls
       │ Control Commands              │ Audio Processing              │
       ▼                               ▼                               ▼
┌─────────────┐                 ┌─────────────┐                 ┌─────────────┐
│Audio Capture│                 │Session Store│                 │MCP Server   │
│Audio Playback│                │Message Queue│                 │(External)   │
└─────────────┘                 └─────────────┘                 └─────────────┘
```

## Communication Protocols

### WebSocket Messages
All communication between frontend and backend uses WebSocket with JSON messages:

```json
{
  "type": "audio|text|control|error|heartbeat|session",
  "session_id": "uuid",
  "data": {...},
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### Audio Format
- **Sample Rate**: 24kHz
- **Channels**: Mono (1 channel)
- **Format**: 16-bit PCM
- **Encoding**: Base64 for transmission
- **Chunk Size**: 1024 samples

### MCP Integration
The Python AI engine integrates with MCP servers for function calling:

```python
# Example MCP function call
result = await mcp_client.call_function("get_customer_data", {
    "customer_id": "12345"
})
```

## Performance Characteristics

### Latency Targets
- **Audio Processing**: <50ms
- **Response Time**: <200ms
- **WebSocket Connection**: <100ms

### Scalability
- **Concurrent Calls**: 1000+
- **Audio Quality**: 24kHz PCM
- **Memory Usage**: Optimized for low footprint
- **CPU Usage**: Efficient Go concurrency + Python AI

### Reliability
- **Uptime**: 99.9% target
- **Error Handling**: Graceful degradation
- **Session Recovery**: Automatic cleanup
- **Connection Resilience**: Auto-reconnection

## Security Considerations

### Authentication
- API key management via environment variables
- WebSocket connection validation
- Session-based access control

### Data Protection
- Audio data encryption in transit
- No persistent audio storage
- Secure MCP communication
- Input validation and sanitization

### Network Security
- CORS configuration
- Rate limiting
- DDoS protection
- Secure WebSocket connections (WSS in production)

## Deployment Architecture

### Development
```bash
# Start all services locally
./scripts/start.sh

# Individual services
cd backend && go run main.go
cd ai-engine && python main.py
cd frontend && npm run dev
```

### Production (Docker)
```bash
# Build and start all services
docker-compose up -d

# Scale services
docker-compose up -d --scale backend=3
```

### Cloud Deployment
- **Go Backend**: Deploy to GCP Cloud Run or AWS ECS
- **Python AI Engine**: Deploy to GCP Cloud Run or AWS Lambda
- **Frontend**: Deploy to GCP Cloud Storage or AWS S3
- **Load Balancer**: GCP Load Balancer or AWS ALB

## Monitoring and Observability

### Performance Profiling
- **Go Backend**: pprof integration with CPU, memory, and goroutine profiling
- **Python AI Engine**: Custom performance metrics with latency histograms
- **Real-time Monitoring**: Live performance tracking with p50/p95/p99 percentiles
- **Benchmarking Suite**: Comprehensive performance testing framework

### Distributed Tracing
- **OpenTelemetry Integration**: Cross-service request tracing
- **Jaeger Integration**: Distributed tracing visualization
- **Request Flow Tracking**: End-to-end request lifecycle monitoring
- **Performance Bottleneck Identification**: Automatic performance issue detection

### Metrics Collection
- **Prometheus Metrics**: Standardized metrics export
- **Custom Metrics**: Application-specific performance indicators
- **System Metrics**: CPU, memory, and network utilization
- **Business Metrics**: Connection count, audio quality, response times

### Key Performance Indicators
- **Latency Metrics**: WebSocket roundtrip, audio processing, AI engine RPC
- **Throughput Metrics**: Messages per second, connections per second
- **Quality Metrics**: Error rates, dropped audio chunks, success rates
- **Resource Metrics**: CPU usage, memory consumption, GC pause times

### Logging
- **Structured JSON Logging**: Machine-readable log format
- **Request/Response Tracing**: Complete request lifecycle logging
- **Audio Processing Metrics**: Detailed audio pipeline logging
- **Session Lifecycle Events**: Session creation, updates, and cleanup

### Health Checks
- **Service Health**: `/health` endpoints for all services
- **WebSocket Monitoring**: Connection status and performance
- **MCP Server Connectivity**: External service health monitoring
- **Audio System Status**: Audio pipeline health verification
- **Performance Alerts**: Automated performance degradation detection

## Future Enhancements

### Planned Features
- Multi-language support
- Voice cloning capabilities
- Advanced conversation analytics
- Integration with more MCP servers
- Mobile app support

### Performance Improvements
- **Memory Optimization**: Lock-free ring buffers and memory pools
- **CPU Optimization**: CPU affinity and SIMD audio processing
- **Audio Compression**: Opus codec integration for reduced bandwidth
- **Connection Pooling**: HTTP connection reuse and optimization
- **Caching Strategies**: Intelligent caching for frequently accessed data
- **Edge Deployment**: CDN and edge computing support
- **Real-time Profiling**: Continuous performance monitoring and optimization

### Scalability Enhancements
- Horizontal scaling
- Database integration
- Message queue systems
- Microservices architecture
