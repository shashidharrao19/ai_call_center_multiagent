# AI Call Center - Real-time Customer Service

A high-performance, real-time AI call center built with Go, Python, and JavaScript, leveraging Google's Gemini 2.0 Live API for human-like customer conversations.

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Web Client    │    │   Go Gateway     │    │  Go Audio Hub   │
│   (JavaScript)  │◄──►│   (Load Balancer)│◄──►│  (WebSocket)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │  Go Session Mgr  │    │  Go Audio Proc  │
                       │  (Concurrency)   │    │  (Streaming)    │
                       └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │  Python AI Core  │    │  Python MCP     │
                       │  (Gemini Live)   │◄──►│  (Functions)    │
                       └──────────────────┘    └─────────────────┘
```

## 🚀 Key Features

- **Real-time bidirectional audio streaming** via WebSocket
- **Barge-in support** - users can interrupt AI responses
- **Inline MCP function calling** during conversations
- **High-performance Go backend** for WebSocket management
- **Python AI processing** with official Gemini SDK
- **24kHz PCM mono audio** with automatic normalization
- **Per-call session management** (no persistent context)
- **Advanced profiling & observability** with pprof, OpenTelemetry, and performance metrics
- **Comprehensive benchmarking suite** for performance testing
- **Real-time performance monitoring** with latency tracking (p50/p95/p99)
- **Production-ready monitoring** with Prometheus metrics and health checks

## 🛠️ Technology Stack

### Backend (Go)
- **WebSocket handling**: High-performance concurrent connections
- **Session management**: Per-call state management
- **Audio processing**: PCM normalization and streaming
- **Load balancing**: Request distribution and routing
- **Performance profiling**: pprof integration with custom metrics
- **Real-time monitoring**: Latency histograms and system metrics

### AI Engine (Python)
- **Gemini 2.0 Live API**: Official Google SDK integration
- **MCP integration**: Function calling for data access
- **Audio processing**: Base64 encoding/decoding
- **Conversation management**: Turn-based dialogue handling
- **Performance profiling**: Custom metrics collection and system monitoring
- **Distributed tracing**: OpenTelemetry integration with Jaeger

### Frontend (JavaScript)
- **Audio capture**: Browser microphone access
- **Audio playback**: Real-time audio streaming
- **WebSocket client**: Connection to Go backend
- **UI components**: Call interface and controls

## 📁 Project Structure

```
ai_call_center/
├── README.md
├── docker-compose.yml
├── .env.example
├── go.mod
├── requirements.txt
├── package.json
│
├── backend/                    # Go WebSocket server
│   ├── main.go
│   ├── internal/
│   │   ├── gateway/           # Load balancer & routing
│   │   ├── websocket/         # WebSocket connection handling
│   │   ├── session/           # Session management
│   │   ├── audio/             # Audio processing
│   │   ├── profiling/         # Performance profiling & metrics
│   │   └── config/            # Configuration
│   ├── pkg/
│   │   ├── models/            # Data structures
│   │   └── utils/             # Utilities
│   └── cmd/
│       └── server/            # Server entry point
│
├── ai-engine/                 # Python AI processing
│   ├── main.py
│   ├── requirements.txt
│   ├── src/
│   │   ├── gemini/            # Gemini Live API integration
│   │   ├── mcp/               # MCP function calling
│   │   ├── audio/             # Audio processing
│   │   ├── conversation/      # Conversation management
│   │   ├── profiling/         # Performance profiling & metrics
│   │   └── observability/     # OpenTelemetry tracing
│   ├── config/
│   │   └── settings.py        # Configuration
│   └── tests/
│
├── frontend/                  # JavaScript client
│   ├── index.html
│   ├── package.json
│   ├── src/
│   │   ├── audio/             # Audio capture/playback
│   │   ├── websocket/         # WebSocket client
│   │   ├── ui/                # User interface
│   │   └── utils/             # Utilities
│   └── dist/                  # Built assets
│
├── shared/                    # Shared configurations
│   ├── proto/                 # Protocol buffers (if needed)
│   └── schemas/               # JSON schemas
│
├── scripts/                   # Deployment & utility scripts
│   ├── setup.sh
│   ├── start.sh
│   ├── stop.sh
│   ├── test.sh
│   └── benchmark.py           # Performance benchmarking suite
│
└── docs/                      # Documentation
    ├── api.md
    ├── deployment.md
    ├── architecture.md
    └── performance_optimization_plan.md
```

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Python 3.11+
- Node.js 18+
- Google Cloud API key with Gemini access

### Setup

1. **Clone and setup environment**:
   ```bash
   git clone <repository>
   cd ai_call_center
   cp .env.example .env
   # Edit .env with your API keys
   ```

2. **Start backend services**:
   ```bash
   # Start Go backend
   cd backend && go run main.go
   
   # Start Python AI engine
   cd ai-engine && pip install -r requirements.txt && python main.py
   
   # Start frontend
   cd frontend && npm install && npm start
   ```

3. **Access the application**:
   - Open `http://localhost:3000` in your browser
   - Allow microphone access
   - Start a conversation!

4. **Access profiling endpoints**:
   - Go Backend: `http://localhost:8080/debug/pprof/`
   - Go Performance: `http://localhost:8080/performance`
   - Python Performance: `http://localhost:8000/performance`
   - Prometheus Metrics: `http://localhost:8000/metrics`

## 🔧 Configuration

### Environment Variables
```bash
# Gemini API
GOOGLE_API_KEY=your_gemini_api_key
GEMINI_MODEL=models/gemini-2.0-flash-live-001

# Go Backend
GO_PORT=8080
GO_WEBSOCKET_PATH=/ws
GO_MAX_CONNECTIONS=1000

# Python AI Engine
PYTHON_PORT=8000
MCP_SERVER_URL=http://localhost:8001
AUDIO_SAMPLE_RATE=24000

# Frontend
FRONTEND_PORT=3000
WEBSOCKET_URL=ws://localhost:8080/ws
```

## 📊 Performance Targets

- **Latency**: <50ms end-to-end (p95), <200ms response time
- **Concurrency**: 1000+ concurrent calls
- **Audio Quality**: 24kHz PCM, <50ms audio delay
- **Uptime**: 99.9% availability
- **Profiling**: Real-time p50/p95/p99 latency tracking
- **Monitoring**: Comprehensive metrics and distributed tracing

## 🔒 Security

- API key management via environment variables
- WebSocket connection authentication
- Rate limiting and DDoS protection
- Audio data encryption in transit

## 📈 Monitoring & Observability

### Performance Profiling
- **Go Backend**: pprof integration (`/debug/pprof/`)
- **Python AI Engine**: Custom performance metrics
- **Real-time Monitoring**: Latency histograms and system metrics
- **Benchmarking**: Comprehensive performance testing suite

### Observability
- **Distributed Tracing**: OpenTelemetry with Jaeger integration
- **Metrics Collection**: Prometheus-compatible metrics
- **Health Checks**: `/health` endpoints for all services
- **Performance Reports**: Detailed latency analysis (p50/p95/p99)

### Key Metrics Tracked
- WebSocket connection count and latency
- Audio processing performance
- AI engine RPC call metrics
- Gemini API response times
- System resource usage (CPU, memory)
- Error rates and success rates

## 🧪 Performance Testing

### Run Benchmarks
```bash
# Run comprehensive benchmark suite
python scripts/benchmark.py

# Run specific tests
python scripts/benchmark.py --test websocket
python scripts/benchmark.py --test audio
python scripts/benchmark.py --test rpc
```

### Access Profiling Data
```bash
# Go pprof profiling
go tool pprof http://localhost:8080/debug/pprof/profile

# View performance reports
curl http://localhost:8080/performance | jq
curl http://localhost:8000/performance | jq

# Prometheus metrics
curl http://localhost:8000/metrics
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details

## 🆘 Support

- Documentation: `/docs`
- Issues: GitHub Issues
- Discussions: GitHub Discussions
