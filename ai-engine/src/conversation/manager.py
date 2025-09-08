"""
Conversation Manager - Handles conversation flow and MCP function calling
Integrates Gemini Live API with MCP for intelligent function calling
"""

import asyncio
import json
import logging
from datetime import datetime
from typing import Dict, Any, Optional, List
from dataclasses import dataclass, asdict

from ..gemini.live_api import GeminiLiveAPI
from ..mcp.client import MCPClient

logger = logging.getLogger(__name__)

@dataclass
class ConversationSession:
    """Represents an active conversation session"""
    session_id: str
    created_at: datetime
    last_activity: datetime
    status: str = "active"
    context: Dict[str, Any] = None
    customer_id: Optional[str] = None
    
    def __post_init__(self):
        if self.context is None:
            self.context = {}

class ConversationManager:
    """Manages conversation sessions and coordinates between Gemini and MCP"""
    
    def __init__(self, gemini_api: GeminiLiveAPI, mcp_client: MCPClient):
        self.gemini_api = gemini_api
        self.mcp_client = mcp_client
        self.sessions: Dict[str, ConversationSession] = {}
        self.response_handlers: Dict[str, asyncio.Queue] = {}
        
        # Start listening for Gemini responses
        asyncio.create_task(self._listen_for_gemini_responses())
    
    async def process_message(self, session_id: str, message_type: str, data: Dict[str, Any], audio_config: Dict[str, Any]) -> Dict[str, Any]:
        """Process incoming message from Go backend"""
        try:
            # Get or create session
            session = await self._get_or_create_session(session_id)
            
            # Update session activity
            session.last_activity = datetime.now()
            
            # Set audio config if provided
            if audio_config:
                self.gemini_api.set_audio_config(
                    sample_rate=audio_config.get("sample_rate", 24000),
                    channels=audio_config.get("channels", 1),
                    format=audio_config.get("format", "PCM")
                )
            
            # Process based on message type
            if message_type == "audio":
                return await self._process_audio_message(session, data)
            elif message_type == "text":
                return await self._process_text_message(session, data)
            else:
                logger.warning(f"Unknown message type: {message_type}")
                return self._create_error_response("Unknown message type")
                
        except Exception as e:
            logger.error(f"Error processing message: {str(e)}")
            return self._create_error_response(str(e))
    
    async def _get_or_create_session(self, session_id: str) -> ConversationSession:
        """Get existing session or create new one"""
        if session_id not in self.sessions:
            self.sessions[session_id] = ConversationSession(
                session_id=session_id,
                created_at=datetime.now(),
                last_activity=datetime.now()
            )
            # Create response queue for this session
            self.response_handlers[session_id] = asyncio.Queue()
            logger.info(f"Created new conversation session: {session_id}")
        
        return self.sessions[session_id]
    
    async def _process_audio_message(self, session: ConversationSession, data: Dict[str, Any]) -> Dict[str, Any]:
        """Process audio message"""
        try:
            audio_data = data.get("data")
            if not audio_data:
                return self._create_error_response("No audio data provided")
            
            # Convert base64 audio data to bytes if needed
            if isinstance(audio_data, str):
                import base64
                audio_bytes = base64.b64decode(audio_data)
            else:
                audio_bytes = audio_data
            
            # Send audio to Gemini
            await self.gemini_api.send_audio_input(audio_bytes, session.session_id)
            
            # Wait for response (with timeout)
            try:
                response = await asyncio.wait_for(
                    self.response_handlers[session.session_id].get(),
                    timeout=30.0
                )
                return response
            except asyncio.TimeoutError:
                return self._create_error_response("Response timeout")
                
        except Exception as e:
            logger.error(f"Error processing audio message: {str(e)}")
            return self._create_error_response(str(e))
    
    async def _process_text_message(self, session: ConversationSession, data: Dict[str, Any]) -> Dict[str, Any]:
        """Process text message"""
        try:
            text = data.get("text", "")
            if not text:
                return self._create_error_response("No text provided")
            
            # Check if text contains function call requests
            if await self._should_call_mcp_function(text, session):
                return await self._handle_mcp_function_call(text, session)
            
            # Send text to Gemini
            await self.gemini_api.send_text_input(text, session.session_id)
            
            # Wait for response
            try:
                response = await asyncio.wait_for(
                    self.response_handlers[session.session_id].get(),
                    timeout=30.0
                )
                return response
            except asyncio.TimeoutError:
                return self._create_error_response("Response timeout")
                
        except Exception as e:
            logger.error(f"Error processing text message: {str(e)}")
            return self._create_error_response(str(e))
    
    async def _should_call_mcp_function(self, text: str, session: ConversationSession) -> bool:
        """Determine if text should trigger MCP function call"""
        # Simple keyword-based detection - in production, use more sophisticated NLP
        mcp_keywords = [
            "customer data", "look up customer", "get customer info",
            "search knowledge", "find information", "create ticket",
            "update customer", "customer history"
        ]
        
        text_lower = text.lower()
        return any(keyword in text_lower for keyword in mcp_keywords)
    
    async def _handle_mcp_function_call(self, text: str, session: ConversationSession) -> Dict[str, Any]:
        """Handle MCP function calling"""
        try:
            # Determine which function to call based on text
            function_name, parameters = await self._parse_function_call(text, session)
            
            if not function_name:
                # No function call needed, send to Gemini
                await self.gemini_api.send_text_input(text, session.session_id)
                return await asyncio.wait_for(
                    self.response_handlers[session.session_id].get(),
                    timeout=30.0
                )
            
            # Call MCP function
            result = await self.mcp_client.call_function(function_name, parameters)
            
            # Create response with function result
            response_text = await self._format_function_response(function_name, result)
            
            return {
                "message_type": "text",
                "data": {
                    "text": response_text,
                    "function_called": function_name,
                    "function_result": result
                },
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error(f"Error handling MCP function call: {str(e)}")
            return self._create_error_response(str(e))
    
    async def _parse_function_call(self, text: str, session: ConversationSession) -> tuple[Optional[str], Dict[str, Any]]:
        """Parse text to determine function call and parameters"""
        text_lower = text.lower()
        
        # Simple parsing - in production, use more sophisticated NLP
        if "customer data" in text_lower or "look up customer" in text_lower:
            # Extract customer ID from text (simple regex or NLP)
            customer_id = self._extract_customer_id(text)
            if customer_id:
                session.customer_id = customer_id
                return "get_customer_data", {"customer_id": customer_id}
        
        elif "search" in text_lower and ("knowledge" in text_lower or "information" in text_lower):
            query = text  # Use full text as query
            return "search_knowledge_base", {"query": query}
        
        elif "create ticket" in text_lower or "support ticket" in text_lower:
            return "create_ticket", {"description": text, "customer_id": session.customer_id}
        
        return None, {}
    
    def _extract_customer_id(self, text: str) -> Optional[str]:
        """Extract customer ID from text (simple implementation)"""
        import re
        # Look for patterns like "customer 12345" or "ID: 12345"
        patterns = [
            r'customer\s+(\d+)',
            r'id:\s*(\d+)',
            r'customer\s+id:\s*(\d+)'
        ]
        
        for pattern in patterns:
            match = re.search(pattern, text.lower())
            if match:
                return match.group(1)
        
        return None
    
    async def _format_function_response(self, function_name: str, result: Dict[str, Any]) -> str:
        """Format MCP function result into natural language response"""
        if "error" in result:
            return f"I encountered an error while {function_name}: {result['error']}"
        
        # Format based on function type
        if function_name == "get_customer_data":
            customer_data = result.get("data", {})
            return f"Here's the customer information: {json.dumps(customer_data, indent=2)}"
        
        elif function_name == "search_knowledge_base":
            search_results = result.get("results", [])
            if search_results:
                return f"I found {len(search_results)} relevant articles. Here's the most relevant one: {search_results[0].get('title', 'No title')}"
            else:
                return "I couldn't find any relevant information in the knowledge base."
        
        elif function_name == "create_ticket":
            ticket_id = result.get("ticket_id")
            return f"I've created a support ticket for you. Ticket ID: {ticket_id}"
        
        return f"Function {function_name} completed successfully."
    
    async def _listen_for_gemini_responses(self):
        """Listen for responses from Gemini and route to appropriate sessions"""
        try:
            await self.gemini_api.listen_for_responses(self._handle_gemini_response)
        except Exception as e:
            logger.error(f"Error listening for Gemini responses: {str(e)}")
    
    async def _handle_gemini_response(self, message: Dict[str, Any]):
        """Handle response from Gemini Live API"""
        try:
            # Extract session ID from message (you might need to implement session tracking in Gemini)
            # For now, we'll use a simple approach - route to the most recent session
            if not self.sessions:
                return
            
            # Get the most recent session
            latest_session = max(self.sessions.values(), key=lambda s: s.last_activity)
            session_id = latest_session.session_id
            
            # Check for audio output
            audio_data = self.gemini_api.decode_audio_output(message)
            if audio_data:
                response = {
                    "message_type": "audio",
                    "data": {
                        "data": audio_data,
                        "format": "PCM"
                    },
                    "timestamp": datetime.now().isoformat()
                }
                await self.response_handlers[session_id].put(response)
                return
            
            # Check for text output
            text_data = self.gemini_api.decode_text_output(message)
            if text_data:
                response = {
                    "message_type": "text",
                    "data": {
                        "text": text_data
                    },
                    "timestamp": datetime.now().isoformat()
                }
                await self.response_handlers[session_id].put(response)
                return
            
            # Check for interruption
            if self.gemini_api.is_interrupted(message):
                logger.info(f"User interrupted response for session {session_id}")
                # Clear any pending responses
                while not self.response_handlers[session_id].empty():
                    try:
                        self.response_handlers[session_id].get_nowait()
                    except asyncio.QueueEmpty:
                        break
            
        except Exception as e:
            logger.error(f"Error handling Gemini response: {str(e)}")
    
    def _create_error_response(self, error_message: str) -> Dict[str, Any]:
        """Create error response"""
        return {
            "message_type": "error",
            "data": {
                "error": error_message
            },
            "timestamp": datetime.now().isoformat()
        }
    
    async def get_active_sessions(self) -> List[Dict[str, Any]]:
        """Get list of active sessions"""
        return [
            {
                "session_id": session.session_id,
                "created_at": session.created_at.isoformat(),
                "last_activity": session.last_activity.isoformat(),
                "status": session.status,
                "customer_id": session.customer_id
            }
            for session in self.sessions.values()
            if session.status == "active"
        ]
    
    async def end_session(self, session_id: str):
        """End a conversation session"""
        if session_id in self.sessions:
            self.sessions[session_id].status = "ended"
            # Clean up response queue
            if session_id in self.response_handlers:
                del self.response_handlers[session_id]
            logger.info(f"Session {session_id} ended")
