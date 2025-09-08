"""
Gemini Live API integration for real-time bidirectional audio streaming
Based on the notebook implementation but adapted for production use
"""

import asyncio
import base64
import json
import logging
import websockets
from typing import Dict, Any, Optional, Callable
from dataclasses import dataclass

import google.generativeai as genai

logger = logging.getLogger(__name__)

@dataclass
class AudioConfig:
    """Audio configuration for the Live API"""
    sample_rate: int = 24000
    channels: int = 1
    format: str = "PCM"

class GeminiLiveAPI:
    """Gemini Live API client for real-time audio streaming"""
    
    def __init__(self, api_key: str, model: str = "models/gemini-2.0-flash-live-001"):
        self.api_key = api_key
        self.model = model
        self.websocket = None
        self.audio_config = AudioConfig()
        self.is_connected = False
        self.message_handlers: Dict[str, Callable] = {}
        
        # Configure Gemini API
        genai.configure(api_key=api_key)
    
    async def initialize(self):
        """Initialize the Gemini Live API connection"""
        try:
            # Connect to Gemini Live API WebSocket
            ws_url = f"wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent?key={self.api_key}"
            
            self.websocket = await websockets.connect(ws_url)
            self.is_connected = True
            
            # Send initial setup
            await self._send_setup()
            
            logger.info("Gemini Live API initialized successfully")
            
        except Exception as e:
            logger.error(f"Failed to initialize Gemini Live API: {str(e)}")
            raise
    
    async def _send_setup(self):
        """Send initial setup message to Gemini"""
        setup_message = {
            "setup": {
                "model": self.model,
            }
        }
        await self._send_message(setup_message)
    
    async def _send_message(self, message: Dict[str, Any]):
        """Send a message to the Gemini Live API"""
        if not self.websocket or not self.is_connected:
            raise ConnectionError("Not connected to Gemini Live API")
        
        try:
            await self.websocket.send(json.dumps(message))
        except Exception as e:
            logger.error(f"Failed to send message: {str(e)}")
            raise
    
    async def send_audio_input(self, audio_data: bytes, session_id: str):
        """Send audio input to Gemini"""
        try:
            # Encode audio data to base64
            audio_b64 = base64.b64encode(audio_data).decode('utf-8')
            
            message = {
                "realtimeInput": {
                    "mediaChunks": [{
                        "mimeType": f"audio/pcm;rate={self.audio_config.sample_rate}",
                        "data": audio_b64,
                    }],
                },
            }
            
            await self._send_message(message)
            logger.debug(f"Audio input sent for session {session_id}")
            
        except Exception as e:
            logger.error(f"Failed to send audio input: {str(e)}")
            raise
    
    async def send_text_input(self, text: str, session_id: str):
        """Send text input to Gemini"""
        try:
            message = {
                "clientContent": {
                    "turns": [{
                        "role": "USER",
                        "parts": [{"text": text}],
                    }],
                    "turnComplete": True,
                },
            }
            
            await self._send_message(message)
            logger.debug(f"Text input sent for session {session_id}: {text}")
            
        except Exception as e:
            logger.error(f"Failed to send text input: {str(e)}")
            raise
    
    async def listen_for_responses(self, response_handler: Callable):
        """Listen for responses from Gemini and call the handler"""
        if not self.websocket or not self.is_connected:
            raise ConnectionError("Not connected to Gemini Live API")
        
        try:
            async for message in self.websocket:
                try:
                    data = json.loads(message)
                    await response_handler(data)
                except json.JSONDecodeError as e:
                    logger.error(f"Failed to decode message: {str(e)}")
                except Exception as e:
                    logger.error(f"Error processing response: {str(e)}")
                    
        except websockets.exceptions.ConnectionClosed:
            logger.warning("Gemini Live API connection closed")
            self.is_connected = False
        except Exception as e:
            logger.error(f"Error listening for responses: {str(e)}")
            raise
    
    def decode_audio_output(self, message: Dict[str, Any]) -> Optional[bytes]:
        """Decode audio output from Gemini response"""
        try:
            content_input = message.get('serverContent', {})
            content = content_input.get('modelTurn', {})
            
            audio_data = b''
            for part in content.get('parts', []):
                data = part.get('inlineData', {}).get('data', '')
                if data:
                    audio_data += base64.b64decode(data)
            
            return audio_data if audio_data else None
            
        except Exception as e:
            logger.error(f"Failed to decode audio output: {str(e)}")
            return None
    
    def decode_text_output(self, message: Dict[str, Any]) -> Optional[str]:
        """Decode text output from Gemini response"""
        try:
            content_input = message.get('serverContent', {})
            content = content_input.get('modelTurn', {})
            
            text_parts = []
            for part in content.get('parts', []):
                if 'text' in part:
                    text_parts.append(part['text'])
            
            return ' '.join(text_parts) if text_parts else None
            
        except Exception as e:
            logger.error(f"Failed to decode text output: {str(e)}")
            return None
    
    def is_interrupted(self, message: Dict[str, Any]) -> bool:
        """Check if the message indicates user interruption"""
        return 'interrupted' in message.get('serverContent', {})
    
    def is_turn_complete(self, message: Dict[str, Any]) -> bool:
        """Check if the message indicates turn completion"""
        return 'turnComplete' in message.get('serverContent', {})
    
    async def close(self):
        """Close the Gemini Live API connection"""
        if self.websocket:
            await self.websocket.close()
            self.is_connected = False
            logger.info("Gemini Live API connection closed")
    
    def set_audio_config(self, sample_rate: int, channels: int = 1, format: str = "PCM"):
        """Set audio configuration"""
        self.audio_config = AudioConfig(
            sample_rate=sample_rate,
            channels=channels,
            format=format
        )
        logger.info(f"Audio config updated: {self.audio_config}")
