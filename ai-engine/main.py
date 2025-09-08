"""
AI Engine - Main entry point for the AI processing service
Handles Gemini Live API integration and MCP function calling
"""

import asyncio
import logging
import os
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import uvicorn

from src.gemini.live_api import GeminiLiveAPI
from src.mcp.client import MCPClient
from src.conversation.manager import ConversationManager
from src.profiling.profiler import get_profiler, measure_time
from src.observability.tracing import get_tracer, trace_function
from config.settings import Settings

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Load settings
settings = Settings()

# Global instances
gemini_api = None
mcp_client = None
conversation_manager = None
profiler = get_profiler()
tracer = get_tracer()

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager"""
    global gemini_api, mcp_client, conversation_manager
    
    logger.info("Starting AI Engine...")
    
    # Initialize profiler and tracer
    profiler.start()
    tracer.initialize()
    
    # Initialize Gemini Live API
    gemini_api = GeminiLiveAPI(settings.google_api_key, settings.gemini_model)
    await gemini_api.initialize()
    
    # Initialize MCP client
    mcp_client = MCPClient(settings.mcp_server_url)
    await mcp_client.connect()
    
    # Initialize conversation manager
    conversation_manager = ConversationManager(gemini_api, mcp_client)
    
    logger.info("AI Engine started successfully")
    
    yield
    
    # Cleanup
    logger.info("Shutting down AI Engine...")
    profiler.stop()
    if mcp_client:
        await mcp_client.disconnect()
    if gemini_api:
        await gemini_api.close()
    logger.info("AI Engine shutdown complete")

# Create FastAPI app
app = FastAPI(
    title="AI Call Center Engine",
    description="Real-time AI processing with Gemini Live API and MCP integration",
    version="1.0.0",
    lifespan=lifespan
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Request/Response models
class ProcessRequest(BaseModel):
    session_id: str
    message_type: str
    data: dict
    audio_config: dict

class ProcessResponse(BaseModel):
    session_id: str
    message_type: str
    data: dict
    timestamp: str

@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy", "service": "ai-engine"}

@app.post("/process", response_model=ProcessResponse)
async def process_message(request: ProcessRequest):
    """Process incoming messages from the Go backend"""
    try:
        logger.info(f"Processing message for session {request.session_id}")
        
        # Process the message through conversation manager
        response = await conversation_manager.process_message(
            session_id=request.session_id,
            message_type=request.message_type,
            data=request.data,
            audio_config=request.audio_config
        )
        
        return ProcessResponse(
            session_id=request.session_id,
            message_type=response["message_type"],
            data=response["data"],
            timestamp=response["timestamp"]
        )
        
    except Exception as e:
        logger.error(f"Error processing message: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/sessions")
async def get_active_sessions():
    """Get active conversation sessions"""
    try:
        sessions = await conversation_manager.get_active_sessions()
        return {"sessions": sessions}
    except Exception as e:
        logger.error(f"Error getting sessions: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/performance")
async def get_performance_report():
    """Get performance metrics report"""
    try:
        report = profiler.get_performance_report()
        return report
    except Exception as e:
        logger.error(f"Error getting performance report: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/metrics")
async def get_metrics():
    """Get metrics in Prometheus format"""
    try:
        report = profiler.get_performance_report()
        
        # Convert to Prometheus format
        metrics = []
        metrics.append(f"# HELP gemini_requests_total Total Gemini API requests")
        metrics.append(f"# TYPE gemini_requests_total counter")
        metrics.append(f"gemini_requests_total {report['gemini']['requests']}")
        
        metrics.append(f"# HELP gemini_errors_total Total Gemini API errors")
        metrics.append(f"# TYPE gemini_errors_total counter")
        metrics.append(f"gemini_errors_total {report['gemini']['errors']}")
        
        metrics.append(f"# HELP audio_chunks_processed_total Total audio chunks processed")
        metrics.append(f"# TYPE audio_chunks_processed_total counter")
        metrics.append(f"audio_chunks_processed_total {report['audio_processing']['chunks_processed']}")
        
        metrics.append(f"# HELP mcp_calls_total Total MCP function calls")
        metrics.append(f"# TYPE mcp_calls_total counter")
        metrics.append(f"mcp_calls_total {report['mcp']['calls']}")
        
        metrics.append(f"# HELP cpu_usage_percent Current CPU usage percentage")
        metrics.append(f"# TYPE cpu_usage_percent gauge")
        metrics.append(f"cpu_usage_percent {report['system']['cpu_usage']}")
        
        metrics.append(f"# HELP memory_usage_percent Current memory usage percentage")
        metrics.append(f"# TYPE memory_usage_percent gauge")
        metrics.append(f"memory_usage_percent {report['system']['memory_usage']}")
        
        return "\n".join(metrics)
    except Exception as e:
        logger.error(f"Error getting metrics: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

@app.delete("/sessions/{session_id}")
async def end_session(session_id: str):
    """End a conversation session"""
    try:
        await conversation_manager.end_session(session_id)
        return {"message": f"Session {session_id} ended successfully"}
    except Exception as e:
        logger.error(f"Error ending session: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=int(settings.python_port),
        reload=True,
        log_level="info"
    )
