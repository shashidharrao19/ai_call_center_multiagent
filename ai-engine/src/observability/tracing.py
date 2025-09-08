"""
OpenTelemetry tracing for AI Call Center
"""

import asyncio
import time
from typing import Optional, Dict, Any
from opentelemetry import trace
from opentelemetry.exporter.jaeger.thrift import JaegerExporter
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.resources import Resource
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.httpx import HTTPXClientInstrumentor
from opentelemetry.instrumentation.websockets import WebSocketClientInstrumentor
from opentelemetry.trace import Status, StatusCode
import logging

logger = logging.getLogger(__name__)

class AICallCenterTracer:
    """OpenTelemetry tracer for AI Call Center"""
    
    def __init__(self, service_name: str = "ai-call-center", jaeger_endpoint: Optional[str] = None):
        self.service_name = service_name
        self.jaeger_endpoint = jaeger_endpoint
        self.tracer = None
        self._initialized = False
    
    def initialize(self):
        """Initialize OpenTelemetry tracing"""
        try:
            # Create resource
            resource = Resource.create({
                "service.name": self.service_name,
                "service.version": "1.0.0",
            })
            
            # Create tracer provider
            trace.set_tracer_provider(TracerProvider(resource=resource))
            
            # Create tracer
            self.tracer = trace.get_tracer(__name__)
            
            # Setup Jaeger exporter if endpoint provided
            if self.jaeger_endpoint:
                jaeger_exporter = JaegerExporter(
                    agent_host_name="localhost",
                    agent_port=14268,
                )
                
                span_processor = BatchSpanProcessor(jaeger_exporter)
                trace.get_tracer_provider().add_span_processor(span_processor)
            
            # Instrument FastAPI
            FastAPIInstrumentor.instrument()
            
            # Instrument HTTP client
            HTTPXClientInstrumentor().instrument()
            
            # Instrument WebSocket client
            WebSocketClientInstrumentor().instrument()
            
            self._initialized = True
            logger.info("OpenTelemetry tracing initialized")
            
        except Exception as e:
            logger.error(f"Failed to initialize OpenTelemetry tracing: {e}")
            self._initialized = False
    
    def create_span(self, name: str, attributes: Optional[Dict[str, Any]] = None) -> trace.Span:
        """Create a new span"""
        if not self._initialized or not self.tracer:
            return trace.NoOpTracer().start_span(name)
        
        span = self.tracer.start_span(name)
        
        if attributes:
            for key, value in attributes.items():
                span.set_attribute(key, value)
        
        return span
    
    def trace_gemini_request(self, session_id: str, request_type: str):
        """Trace Gemini API request"""
        return self.create_span(
            "gemini.request",
            {
                "session.id": session_id,
                "request.type": request_type,
                "service": "gemini"
            }
        )
    
    def trace_audio_processing(self, session_id: str, audio_size: int):
        """Trace audio processing"""
        return self.create_span(
            "audio.processing",
            {
                "session.id": session_id,
                "audio.size": audio_size,
                "service": "audio"
            }
        )
    
    def trace_mcp_call(self, session_id: str, function_name: str):
        """Trace MCP function call"""
        return self.create_span(
            "mcp.call",
            {
                "session.id": session_id,
                "function.name": function_name,
                "service": "mcp"
            }
        )
    
    def trace_conversation_flow(self, session_id: str, step: str):
        """Trace conversation flow"""
        return self.create_span(
            "conversation.flow",
            {
                "session.id": session_id,
                "conversation.step": step,
                "service": "conversation"
            }
        )

# Global tracer instance
tracer = AICallCenterTracer()

def get_tracer() -> AICallCenterTracer:
    """Get the global tracer instance"""
    return tracer

# Context manager for tracing
class TraceContext:
    """Context manager for tracing code blocks"""
    
    def __init__(self, span_name: str, attributes: Optional[Dict[str, Any]] = None):
        self.span_name = span_name
        self.attributes = attributes or {}
        self.span = None
    
    def __enter__(self):
        self.span = tracer.create_span(self.span_name, self.attributes)
        return self.span
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        if self.span:
            if exc_type is not None:
                self.span.set_status(Status(StatusCode.ERROR, str(exc_val)))
                self.span.set_attribute("error", True)
                self.span.set_attribute("error.message", str(exc_val))
            else:
                self.span.set_status(Status(StatusCode.OK))
            
            self.span.end()

# Decorator for tracing functions
def trace_function(span_name: str, attributes: Optional[Dict[str, Any]] = None):
    """Decorator to trace function execution"""
    def decorator(func):
        async def async_wrapper(*args, **kwargs):
            with TraceContext(span_name, attributes):
                return await func(*args, **kwargs)
        
        def sync_wrapper(*args, **kwargs):
            with TraceContext(span_name, attributes):
                return func(*args, **kwargs)
        
        if asyncio.iscoroutinefunction(func):
            return async_wrapper
        else:
            return sync_wrapper
    
    return decorator

# Specific tracing decorators
def trace_gemini_request(session_id: str, request_type: str):
    """Trace Gemini API request"""
    return trace_function(
        "gemini.request",
        {
            "session.id": session_id,
            "request.type": request_type,
            "service": "gemini"
        }
    )

def trace_audio_processing(session_id: str, audio_size: int):
    """Trace audio processing"""
    return trace_function(
        "audio.processing",
        {
            "session.id": session_id,
            "audio.size": audio_size,
            "service": "audio"
        }
    )

def trace_mcp_call(session_id: str, function_name: str):
    """Trace MCP function call"""
    return trace_function(
        "mcp.call",
        {
            "session.id": session_id,
            "function.name": function_name,
            "service": "mcp"
        }
    )

def trace_conversation_flow(session_id: str, step: str):
    """Trace conversation flow"""
    return trace_function(
        "conversation.flow",
        {
            "session.id": session_id,
            "conversation.step": step,
            "service": "conversation"
        }
    )
