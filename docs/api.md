# AI Call Center API Documentation

## Overview

The AI Call Center provides REST APIs for the Python AI Engine and WebSocket APIs for real-time communication. This document describes all available endpoints and message formats.

## Python AI Engine REST API

Base URL: `http://localhost:8000`

### Health Check

**GET** `/health`

Returns the health status of the AI engine.

**Response:**
```json
{
  "status": "healthy",
  "service": "ai-engine"
}
```

### Performance Report

**GET** `/performance`

Returns detailed performance metrics and latency statistics.

**Response:**
```json
{
  "timestamp": "2024-01-01T00:00:00Z",
  "uptime_seconds": 3600,
  "gemini": {
    "requests": 1500,
    "errors": 5,
    "error_rate": 0.003,
    "latency": {
      "p50": 0.125,
      "p95": 0.245,
      "p99": 0.389
    }
  },
  "audio_processing": {
    "chunks_processed": 5000,
    "latency": {
      "p50": 0.008,
      "p95": 0.025,
      "p99": 0.048
    }
  },
  "mcp": {
    "calls": 200,
    "errors": 2,
    "error_rate": 0.01,
    "latency": {
      "p50": 0.045,
      "p95": 0.089,
      "p99": 0.156
    }
  },
  "system": {
    "cpu_usage": 45.2,
    "memory_usage": 67.8,
    "active_tasks": 12,
    "blocking_calls": 0
  },
  "event_loop": {
    "latency": {
      "p50": 0.001,
      "p95": 0.003,
      "p99": 0.008
    }
  }
}
```

### Metrics (Prometheus Format)

**GET** `/metrics`

Returns metrics in Prometheus format for monitoring systems.

**Response:**
```
# HELP gemini_requests_total Total Gemini API requests
# TYPE gemini_requests_total counter
gemini_requests_total 1500

# HELP gemini_errors_total Total Gemini API errors
# TYPE gemini_errors_total counter
gemini_errors_total 5

# HELP audio_chunks_processed_total Total audio chunks processed
# TYPE audio_chunks_processed_total counter
audio_chunks_processed_total 5000

# HELP mcp_calls_total Total MCP function calls
# TYPE mcp_calls_total counter
mcp_calls_total 200

# HELP cpu_usage_percent Current CPU usage percentage
# TYPE cpu_usage_percent gauge
cpu_usage_percent 45.2

# HELP memory_usage_percent Current memory usage percentage
# TYPE memory_usage_percent gauge
memory_usage_percent 67.8
```

### Process Message

**POST** `/process`

Processes incoming messages from the Go backend.

**Request Body:**
```json
{
  "session_id": "uuid",
  "message_type": "audio|text",
  "data": {
    "data": "base64_audio_data|text_content",
    "format": "PCM",
    "timestamp": 1640995200000
  },
  "audio_config": {
    "sample_rate": 24000,
    "channels": 1,
    "format": "PCM"
  }
}
```

**Response:**
```json
{
  "session_id": "uuid",
  "message_type": "audio|text|error",
  "data": {
    "data": "base64_audio_data|text_content",
    "format": "PCM",
    "function_called": "get_customer_data",
    "function_result": {...}
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### Get Active Sessions

**GET** `/sessions`

Returns all active conversation sessions.

**Response:**
```json
{
  "sessions": [
    {
      "session_id": "uuid",
      "created_at": "2024-01-01T00:00:00Z",
      "last_activity": "2024-01-01T00:05:00Z",
      "status": "active",
      "customer_id": "12345"
    }
  ]
}
```

### End Session

**DELETE** `/sessions/{session_id}`

Ends a specific conversation session.

**Response:**
```json
{
  "message": "Session {session_id} ended successfully"
}
```

## Go Backend WebSocket API

WebSocket URL: `ws://localhost:8080/ws`

### Performance Profiling

**GET** `/debug/pprof/`

Access Go's built-in profiling endpoints for performance analysis.

**Available Endpoints:**
- `/debug/pprof/profile` - CPU profiling
- `/debug/pprof/heap` - Memory heap profiling
- `/debug/pprof/goroutine` - Goroutine profiling
- `/debug/pprof/block` - Block profiling
- `/debug/pprof/mutex` - Mutex profiling

**Usage:**
```bash
# CPU profiling
go tool pprof http://localhost:8080/debug/pprof/profile

# Memory profiling
go tool pprof http://localhost:8080/debug/pprof/heap

# View in browser
go tool pprof -http=:8081 http://localhost:8080/debug/pprof/profile
```

### Performance Report

**GET** `/performance`

Returns comprehensive performance metrics for the Go backend.

**Response:**
```json
{
  "timestamp": "2024-01-01T00:00:00Z",
  "summary": {
    "total_operations": 10000,
    "successful_operations": 9950,
    "failed_operations": 50,
    "success_rate": 0.995
  },
  "latency_metrics": {
    "websocket_roundtrip": {
      "p50_ms": 12.5,
      "p95_ms": 45.2,
      "p99_ms": 89.7,
      "mean_ms": 18.3,
      "count": 5000
    },
    "audio_processing": {
      "p50_ms": 8.2,
      "p95_ms": 25.1,
      "p99_ms": 48.3,
      "mean_ms": 12.7,
      "count": 3000
    },
    "ai_engine_rpc": {
      "p50_ms": 15.8,
      "p95_ms": 35.4,
      "p99_ms": 67.2,
      "mean_ms": 19.1,
      "count": 2000
    }
  }
}
```

### Metrics (Prometheus Format)

**GET** `/metrics`

Returns metrics in Prometheus format for monitoring systems.

**Response:**
```
# HELP websocket_connections Current WebSocket connections
# TYPE websocket_connections gauge
websocket_connections 45

# HELP websocket_messages_received Total WebSocket messages received
# TYPE websocket_messages_received counter
websocket_messages_received 15000

# HELP websocket_messages_sent Total WebSocket messages sent
# TYPE websocket_messages_sent counter
websocket_messages_sent 14800

# HELP rpc_calls Total RPC calls
# TYPE rpc_calls counter
rpc_calls 2000

# HELP rpc_errors Total RPC errors
# TYPE rpc_errors counter
rpc_errors 25

# HELP audio_chunks_processed Total audio chunks processed
# TYPE audio_chunks_processed counter
audio_chunks_processed 3000

# HELP dropped_audio_chunks Total dropped audio chunks
# TYPE dropped_audio_chunks counter
dropped_audio_chunks 5

# HELP memory_usage_bytes Current memory usage in bytes
# TYPE memory_usage_bytes gauge
memory_usage_bytes 134217728

# HELP gc_pause_time_ns Total GC pause time in nanoseconds
# TYPE gc_pause_time_ns gauge
gc_pause_time_ns 5000000
```

### Connection

Connect to the WebSocket endpoint to establish a real-time connection.

**Connection Headers:**
```
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: <base64-encoded-key>
Sec-WebSocket-Version: 13
```

### Message Format

All WebSocket messages use the following JSON format:

```json
{
  "type": "audio|text|control|error|heartbeat|session",
  "session_id": "uuid",
  "data": {...},
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### Message Types

#### 1. Session Message
Sent by server when a new session is created.

```json
{
  "type": "session",
  "session_id": "uuid",
  "data": {
    "session_id": "uuid",
    "audio_config": {
      "sample_rate": 24000,
      "channels": 1,
      "format": "PCM",
      "chunk_size": 1024
    },
    "status": "connecting|active|paused|ended",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### 2. Audio Message
Sent by client to transmit audio data or by server to send AI-generated audio.

**Client to Server:**
```json
{
  "type": "audio",
  "data": {
    "data": "base64_encoded_audio_data",
    "format": "PCM",
    "timestamp": 1640995200000
  }
}
```

**Server to Client:**
```json
{
  "type": "audio",
  "data": {
    "data": "base64_encoded_audio_data",
    "format": "PCM"
  }
}
```

#### 3. Text Message
Sent by client to transmit text or by server to send AI responses.

**Client to Server:**
```json
{
  "type": "text",
  "data": {
    "text": "Hello, I need help with my account"
  }
}
```

**Server to Client:**
```json
{
  "type": "text",
  "data": {
    "text": "I'd be happy to help you with your account. Can you provide your customer ID?",
    "function_called": "get_customer_data",
    "function_result": {
      "customer_id": "12345",
      "name": "John Doe",
      "email": "john@example.com"
    }
  }
}
```

#### 4. Control Message
Sent by client to control the call session.

```json
{
  "type": "control",
  "data": {
    "action": "start_call|end_call|pause_call"
  }
}
```

#### 5. Error Message
Sent by server when an error occurs.

```json
{
  "type": "error",
  "data": {
    "error": "Error description",
    "code": "ERROR_CODE"
  }
}
```

#### 6. Heartbeat Message
Sent by server periodically to maintain connection.

```json
{
  "type": "heartbeat",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## MCP Integration

The AI engine integrates with MCP servers for function calling. Available functions depend on the connected MCP server.

### Common MCP Functions

#### Get Customer Data
```json
{
  "function": "get_customer_data",
  "parameters": {
    "customer_id": "12345"
  }
}
```

#### Search Knowledge Base
```json
{
  "function": "search_knowledge_base",
  "parameters": {
    "query": "password reset"
  }
}
```

#### Create Support Ticket
```json
{
  "function": "create_ticket",
  "parameters": {
    "customer_id": "12345",
    "description": "Customer needs help with billing",
    "priority": "medium"
  }
}
```

## Error Handling

### HTTP Status Codes

- `200 OK`: Request successful
- `400 Bad Request`: Invalid request format
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

### WebSocket Error Codes

- `1000`: Normal closure
- `1001`: Going away
- `1002`: Protocol error
- `1003`: Unsupported data
- `1006`: Abnormal closure
- `1011`: Server error

### Error Response Format

```json
{
  "type": "error",
  "data": {
    "error": "Detailed error message",
    "code": "ERROR_CODE",
    "details": {
      "field": "Additional error details"
    }
  },
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## Rate Limiting

### WebSocket Connections
- Maximum concurrent connections: 1000
- Connection timeout: 30 seconds
- Message rate limit: 100 messages/second per connection

### REST API
- Rate limit: 100 requests/minute per IP
- Burst limit: 10 requests/second
- Timeout: 30 seconds

## Authentication

### API Keys
Set the `GOOGLE_API_KEY` environment variable for Gemini API access.

### WebSocket Authentication
Currently, WebSocket connections are open. In production, implement:
- JWT token validation
- Origin checking
- Rate limiting per user

## Performance Benchmarking

### Benchmark Suite

The system includes a comprehensive benchmarking suite (`scripts/benchmark.py`) for performance testing.

**Usage:**
```bash
# Run full benchmark suite
python scripts/benchmark.py

# Run specific tests
python scripts/benchmark.py --test websocket
python scripts/benchmark.py --test audio
python scripts/benchmark.py --test rpc

# Run with custom parameters
python scripts/benchmark.py --concurrent 100 --duration 300
```

**Benchmark Types:**
- **WebSocket Connection**: Connection establishment latency
- **Message Roundtrip**: WebSocket message latency
- **Audio Processing**: Audio chunk processing performance
- **AI Engine RPC**: HTTP RPC call performance
- **Gemini API Simulation**: Simulated API response times
- **Concurrent Load**: System performance under load

**Sample Output:**
```json
{
  "timestamp": 1640995200,
  "summary": {
    "total_operations": 1000,
    "successful_operations": 995,
    "failed_operations": 5,
    "success_rate": 0.995
  },
  "latency_metrics": {
    "websocket_roundtrip": {
      "p50_ms": 12.5,
      "p95_ms": 45.2,
      "p99_ms": 89.7,
      "mean_ms": 18.3,
      "count": 500
    },
    "audio_processing": {
      "p50_ms": 8.2,
      "p95_ms": 25.1,
      "p99_ms": 48.3,
      "mean_ms": 12.7,
      "count": 300
    }
  }
}
```

## Examples

### Complete Call Flow

1. **Connect to WebSocket**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
```

2. **Receive Session Info**
```javascript
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === 'session') {
    console.log('Session ID:', message.data.session_id);
  }
};
```

3. **Send Audio Data**
```javascript
const audioMessage = {
  type: 'audio',
  data: {
    data: base64AudioData,
    format: 'PCM',
    timestamp: Date.now()
  }
};
ws.send(JSON.stringify(audioMessage));
```

4. **Receive AI Response**
```javascript
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === 'audio') {
    // Play AI-generated audio
    playAudio(message.data.data);
  } else if (message.type === 'text') {
    // Display AI text response
    displayText(message.data.text);
  }
};
```

5. **End Call**
```javascript
const endCallMessage = {
  type: 'control',
  data: {
    action: 'end_call'
  }
};
ws.send(JSON.stringify(endCallMessage));
```
