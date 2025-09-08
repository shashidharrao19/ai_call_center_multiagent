"""
MCP (Model Context Protocol) client for function calling
Integrates with existing MCP server for data access
"""

import asyncio
import json
import logging
from typing import Dict, Any, Optional, List
import aiohttp

logger = logging.getLogger(__name__)

class MCPClient:
    """MCP client for function calling and data access"""
    
    def __init__(self, mcp_server_url: str):
        self.mcp_server_url = mcp_server_url
        self.session = None
        self.is_connected = False
        self.available_functions: List[Dict[str, Any]] = []
    
    async def connect(self):
        """Connect to the MCP server"""
        try:
            self.session = aiohttp.ClientSession()
            
            # Get available functions from MCP server
            await self._get_available_functions()
            
            self.is_connected = True
            logger.info(f"Connected to MCP server at {self.mcp_server_url}")
            
        except Exception as e:
            logger.error(f"Failed to connect to MCP server: {str(e)}")
            raise
    
    async def _get_available_functions(self):
        """Get list of available functions from MCP server"""
        try:
            async with self.session.get(f"{self.mcp_server_url}/functions") as response:
                if response.status == 200:
                    data = await response.json()
                    self.available_functions = data.get("functions", [])
                    logger.info(f"Loaded {len(self.available_functions)} MCP functions")
                else:
                    logger.warning(f"MCP server returned status {response.status}")
                    
        except Exception as e:
            logger.error(f"Failed to get available functions: {str(e)}")
            # Continue without functions - MCP might not be available
    
    async def call_function(self, function_name: str, parameters: Dict[str, Any]) -> Dict[str, Any]:
        """Call a function on the MCP server"""
        if not self.is_connected:
            raise ConnectionError("Not connected to MCP server")
        
        try:
            payload = {
                "function": function_name,
                "parameters": parameters
            }
            
            async with self.session.post(
                f"{self.mcp_server_url}/call",
                json=payload,
                timeout=aiohttp.ClientTimeout(total=30)
            ) as response:
                
                if response.status == 200:
                    result = await response.json()
                    logger.debug(f"MCP function {function_name} called successfully")
                    return result
                else:
                    error_text = await response.text()
                    logger.error(f"MCP function call failed: {response.status} - {error_text}")
                    return {"error": f"Function call failed: {error_text}"}
                    
        except asyncio.TimeoutError:
            logger.error(f"MCP function {function_name} timed out")
            return {"error": "Function call timed out"}
        except Exception as e:
            logger.error(f"Failed to call MCP function {function_name}: {str(e)}")
            return {"error": str(e)}
    
    async def get_customer_data(self, customer_id: str) -> Dict[str, Any]:
        """Get customer data from MCP server"""
        return await self.call_function("get_customer_data", {"customer_id": customer_id})
    
    async def update_customer_data(self, customer_id: str, data: Dict[str, Any]) -> Dict[str, Any]:
        """Update customer data via MCP server"""
        return await self.call_function("update_customer_data", {
            "customer_id": customer_id,
            "data": data
        })
    
    async def search_knowledge_base(self, query: str) -> Dict[str, Any]:
        """Search knowledge base via MCP server"""
        return await self.call_function("search_knowledge_base", {"query": query})
    
    async def create_ticket(self, ticket_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create a support ticket via MCP server"""
        return await self.call_function("create_ticket", ticket_data)
    
    async def get_available_functions(self) -> List[Dict[str, Any]]:
        """Get list of available MCP functions"""
        return self.available_functions
    
    def get_function_schema(self, function_name: str) -> Optional[Dict[str, Any]]:
        """Get schema for a specific function"""
        for func in self.available_functions:
            if func.get("name") == function_name:
                return func
        return None
    
    async def disconnect(self):
        """Disconnect from MCP server"""
        if self.session:
            await self.session.close()
            self.is_connected = False
            logger.info("Disconnected from MCP server")
