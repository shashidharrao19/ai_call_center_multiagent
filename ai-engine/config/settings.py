"""
Configuration settings for the AI Engine
"""

import os
from typing import Optional
from pydantic import BaseSettings, Field

class Settings(BaseSettings):
    """Application settings"""
    
    # Gemini API Configuration
    google_api_key: str = Field(..., env="GOOGLE_API_KEY")
    gemini_model: str = Field(default="models/gemini-2.0-flash-live-001", env="GEMINI_MODEL")
    
    # Server Configuration
    python_port: str = Field(default="8000", env="PYTHON_PORT")
    host: str = Field(default="0.0.0.0", env="HOST")
    
    # MCP Configuration
    mcp_server_url: str = Field(default="http://localhost:8001", env="MCP_SERVER_URL")
    
    # Audio Configuration
    audio_sample_rate: int = Field(default=24000, env="AUDIO_SAMPLE_RATE")
    audio_channels: int = Field(default=1, env="AUDIO_CHANNELS")
    audio_format: str = Field(default="PCM", env="AUDIO_FORMAT")
    
    # Logging Configuration
    log_level: str = Field(default="info", env="LOG_LEVEL")
    log_format: str = Field(default="json", env="LOG_FORMAT")
    
    # Session Configuration
    session_timeout: int = Field(default=3600, env="SESSION_TIMEOUT")  # 1 hour
    max_sessions: int = Field(default=1000, env="MAX_SESSIONS")
    
    # Performance Configuration
    max_concurrent_requests: int = Field(default=100, env="MAX_CONCURRENT_REQUESTS")
    request_timeout: int = Field(default=30, env="REQUEST_TIMEOUT")
    
    class Config:
        env_file = ".env"
        case_sensitive = False

# Create global settings instance
settings = Settings()
