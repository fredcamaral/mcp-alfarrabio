# LLM Integration Guide

This guide provides instructions for integrating the MCP Memory Server with various Large Language Models (LLMs) beyond Claude.

## Table of Contents
- [Overview](#overview)
- [Supported Integration Methods](#supported-integration-methods)
- [OpenAI Integration](#openai-integration)
- [Google Gemini Integration](#google-gemini-integration)
- [Local LLM Integration](#local-llm-integration)
- [Custom LLM Integration](#custom-llm-integration)
- [Best Practices](#best-practices)
- [Performance Considerations](#performance-considerations)
- [Troubleshooting](#troubleshooting)

## Overview

The MCP Memory Server can be integrated with any LLM that supports:
- Function calling / Tool use
- HTTP API interactions
- Custom plugin systems

## Supported Integration Methods

### 1. HTTP API Mode

The most universal integration method using REST APIs:

```yaml
# config.yaml
server:
  mode: http
  port: 8080
  cors:
    enabled: true
    origins: ["*"]
```

### 2. WebSocket Mode

For real-time bidirectional communication:

```yaml
server:
  mode: websocket
  port: 8080
  ping_interval: 30s
```

### 3. gRPC Mode (Coming Soon)

For high-performance integrations:

```yaml
server:
  mode: grpc
  port: 9090
```

## OpenAI Integration

### Using Function Calling

```python
import openai
import requests
import json

# Initialize OpenAI client
client = openai.OpenAI(api_key="your-api-key")

# Define MCP Memory functions
mcp_functions = [
    {
        "name": "memory_store_chunk",
        "description": "Store a conversation chunk in memory",
        "parameters": {
            "type": "object",
            "properties": {
                "content": {"type": "string"},
                "session_id": {"type": "string"},
                "repository": {"type": "string"},
                "tags": {"type": "array", "items": {"type": "string"}}
            },
            "required": ["content", "session_id"]
        }
    },
    {
        "name": "memory_search",
        "description": "Search for similar conversation chunks",
        "parameters": {
            "type": "object",
            "properties": {
                "query": {"type": "string"},
                "limit": {"type": "integer"},
                "repository": {"type": "string"}
            },
            "required": ["query"]
        }
    }
]

# MCP Memory API client
class MCPMemoryClient:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
    
    def call_function(self, function_name, arguments):
        """Call MCP Memory function via HTTP API"""
        endpoint = f"{self.base_url}/tools/{function_name}"
        response = requests.post(endpoint, json=arguments)
        return response.json()

# Example usage
memory_client = MCPMemoryClient()

# Create a conversation with function calling
messages = [
    {"role": "user", "content": "Store this conversation about implementing OAuth2"}
]

response = client.chat.completions.create(
    model="gpt-4",
    messages=messages,
    functions=mcp_functions,
    function_call="auto"
)

# Handle function calls
if response.choices[0].finish_reason == "function_call":
    function_call = response.choices[0].message.function_call
    function_name = function_call.name
    arguments = json.loads(function_call.arguments)
    
    # Call MCP Memory
    result = memory_client.call_function(function_name, arguments)
    
    # Continue conversation with function result
    messages.append({
        "role": "function",
        "name": function_name,
        "content": json.dumps(result)
    })
```

### Custom GPT Action

Create a custom GPT with MCP Memory actions:

```yaml
openapi: 3.0.0
info:
  title: MCP Memory API
  version: 1.0.0
servers:
  - url: https://your-mcp-server.com
paths:
  /tools/memory_store_chunk:
    post:
      summary: Store a conversation chunk
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                content:
                  type: string
                session_id:
                  type: string
                repository:
                  type: string
      responses:
        200:
          description: Success
  /tools/memory_search:
    post:
      summary: Search memory
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                query:
                  type: string
                limit:
                  type: integer
      responses:
        200:
          description: Search results
```

## Google Gemini Integration

### Using Function Declarations

```python
import google.generativeai as genai
import requests

# Configure Gemini
genai.configure(api_key="your-api-key")

# Define MCP Memory functions for Gemini
memory_store = genai.FunctionDeclaration(
    name="memory_store_chunk",
    description="Store a conversation chunk in memory",
    parameters={
        "type": "object",
        "properties": {
            "content": {"type": "string"},
            "session_id": {"type": "string"},
            "repository": {"type": "string"}
        },
        "required": ["content", "session_id"]
    }
)

memory_search = genai.FunctionDeclaration(
    name="memory_search",
    description="Search for similar conversations",
    parameters={
        "type": "object",
        "properties": {
            "query": {"type": "string"},
            "limit": {"type": "integer"}
        },
        "required": ["query"]
    }
)

# Create model with functions
model = genai.GenerativeModel(
    'gemini-pro',
    tools=[memory_store, memory_search]
)

# Function executor
def execute_mcp_function(function_call):
    function_name = function_call.name
    args = function_call.args
    
    # Call MCP Memory HTTP API
    response = requests.post(
        f"http://localhost:8080/tools/{function_name}",
        json=args
    )
    return response.json()

# Use in conversation
chat = model.start_chat()
response = chat.send_message("Find previous discussions about API design")

# Handle function calls
for part in response.parts:
    if hasattr(part, 'function_call'):
        result = execute_mcp_function(part.function_call)
        # Continue conversation with result
```

## Local LLM Integration

### Using LangChain

```python
from langchain.llms import LlamaCpp
from langchain.tools import Tool
from langchain.agents import initialize_agent, AgentType
import requests

# Initialize local LLM
llm = LlamaCpp(
    model_path="./models/llama-2-7b.gguf",
    n_ctx=2048,
    n_threads=8
)

# Define MCP Memory tools
def store_memory(content: str) -> str:
    """Store content in MCP Memory"""
    response = requests.post(
        "http://localhost:8080/tools/memory_store_chunk",
        json={
            "content": content,
            "session_id": "local-llm-session",
            "repository": "local-project"
        }
    )
    return response.json().get("message", "Stored successfully")

def search_memory(query: str) -> str:
    """Search MCP Memory"""
    response = requests.post(
        "http://localhost:8080/tools/memory_search",
        json={"query": query, "limit": 5}
    )
    results = response.json().get("results", [])
    return "\n".join([r["content"] for r in results])

# Create tools
tools = [
    Tool(
        name="StoreMemory",
        func=store_memory,
        description="Store information in long-term memory"
    ),
    Tool(
        name="SearchMemory",
        func=search_memory,
        description="Search for information in memory"
    )
]

# Create agent
agent = initialize_agent(
    tools,
    llm,
    agent=AgentType.ZERO_SHOT_REACT_DESCRIPTION,
    verbose=True
)

# Use the agent
result = agent.run("Search for previous authentication implementations")
```

### Using Ollama

```python
import ollama
import json
import requests

# Define function schemas
functions = [
    {
        "type": "function",
        "function": {
            "name": "memory_store",
            "description": "Store conversation in memory",
            "parameters": {
                "type": "object",
                "properties": {
                    "content": {"type": "string"},
                    "tags": {"type": "array", "items": {"type": "string"}}
                },
                "required": ["content"]
            }
        }
    }
]

# Create a function-calling prompt
def create_function_prompt(user_input, functions):
    return f"""You are an AI assistant with access to functions.
Functions available: {json.dumps(functions)}

User: {user_input}

If you need to use a function, respond with:
FUNCTION_CALL: function_name
ARGUMENTS: {{"arg": "value"}}

Otherwise, respond normally."""

# Process with Ollama
response = ollama.generate(
    model='llama2',
    prompt=create_function_prompt(
        "Store this conversation about database optimization",
        functions
    )
)

# Parse and execute function calls
if "FUNCTION_CALL:" in response['response']:
    # Extract function call and arguments
    lines = response['response'].split('\n')
    function_name = None
    arguments = {}
    
    for i, line in enumerate(lines):
        if line.startswith("FUNCTION_CALL:"):
            function_name = line.split(":")[1].strip()
        elif line.startswith("ARGUMENTS:"):
            arguments = json.loads(lines[i+1])
    
    # Execute MCP Memory call
    if function_name == "memory_store":
        result = requests.post(
            "http://localhost:8080/tools/memory_store_chunk",
            json={
                "content": arguments.get("content"),
                "session_id": "ollama-session",
                "tags": arguments.get("tags", [])
            }
        )
```

## Custom LLM Integration

### Building a Custom Integration

```python
from abc import ABC, abstractmethod
import aiohttp
import asyncio
from typing import Dict, Any, List

class MCPMemoryIntegration(ABC):
    """Base class for MCP Memory integrations"""
    
    def __init__(self, mcp_url: str = "http://localhost:8080"):
        self.mcp_url = mcp_url
        self.session = None
    
    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.session.close()
    
    @abstractmethod
    async def process_with_memory(self, prompt: str) -> str:
        """Process prompt with memory context"""
        pass
    
    async def store_chunk(self, content: str, **kwargs) -> Dict[str, Any]:
        """Store a conversation chunk"""
        async with self.session.post(
            f"{self.mcp_url}/tools/memory_store_chunk",
            json={
                "content": content,
                "session_id": kwargs.get("session_id", "default"),
                "repository": kwargs.get("repository", "default"),
                "tags": kwargs.get("tags", [])
            }
        ) as response:
            return await response.json()
    
    async def search_memory(self, query: str, **kwargs) -> List[Dict[str, Any]]:
        """Search memory for relevant context"""
        async with self.session.post(
            f"{self.mcp_url}/tools/memory_search",
            json={
                "query": query,
                "limit": kwargs.get("limit", 5),
                "repository": kwargs.get("repository")
            }
        ) as response:
            result = await response.json()
            return result.get("results", [])
    
    async def get_context(self, prompt: str) -> str:
        """Get relevant context for a prompt"""
        results = await self.search_memory(prompt, limit=3)
        if results:
            context = "\n\n".join([r["content"] for r in results])
            return f"Relevant context:\n{context}\n\nCurrent query: {prompt}"
        return prompt

# Example implementation for a custom LLM
class CustomLLMWithMemory(MCPMemoryIntegration):
    def __init__(self, llm_client, mcp_url: str = "http://localhost:8080"):
        super().__init__(mcp_url)
        self.llm = llm_client
    
    async def process_with_memory(self, prompt: str) -> str:
        # Get relevant context
        enhanced_prompt = await self.get_context(prompt)
        
        # Process with LLM
        response = await self.llm.generate(enhanced_prompt)
        
        # Store the interaction
        await self.store_chunk(
            content=f"User: {prompt}\nAssistant: {response}",
            tags=["conversation"]
        )
        
        return response

# Usage
async def main():
    # Initialize your custom LLM client
    llm_client = YourLLMClient()
    
    async with CustomLLMWithMemory(llm_client) as memory_llm:
        response = await memory_llm.process_with_memory(
            "How did we implement user authentication?"
        )
        print(response)

asyncio.run(main())
```

### WebSocket Integration

```python
import websockets
import json
import asyncio

class MCPWebSocketClient:
    def __init__(self, uri="ws://localhost:8080/ws"):
        self.uri = uri
    
    async def connect(self):
        self.websocket = await websockets.connect(self.uri)
    
    async def call_tool(self, tool_name: str, params: dict):
        message = {
            "jsonrpc": "2.0",
            "method": tool_name,
            "params": params,
            "id": 1
        }
        
        await self.websocket.send(json.dumps(message))
        response = await self.websocket.recv()
        return json.loads(response)
    
    async def close(self):
        await self.websocket.close()

# Use with any LLM
async def enhanced_llm_call(llm, prompt):
    mcp = MCPWebSocketClient()
    await mcp.connect()
    
    try:
        # Search for context
        context_results = await mcp.call_tool(
            "memory_search",
            {"query": prompt, "limit": 3}
        )
        
        # Enhance prompt with context
        if context_results.get("result", {}).get("results"):
            context = "\n".join([
                r["content"] for r in context_results["result"]["results"]
            ])
            enhanced_prompt = f"Context:\n{context}\n\nQuery: {prompt}"
        else:
            enhanced_prompt = prompt
        
        # Call your LLM
        response = await llm.generate(enhanced_prompt)
        
        # Store the interaction
        await mcp.call_tool(
            "memory_store_chunk",
            {
                "content": f"Q: {prompt}\nA: {response}",
                "session_id": "websocket-session"
            }
        )
        
        return response
    finally:
        await mcp.close()
```

## Best Practices

### 1. Session Management

- Generate unique session IDs for each conversation thread
- Include user identifiers in session IDs for multi-user systems
- Implement session cleanup for long-running applications

### 2. Error Handling

```python
import logging
from tenacity import retry, stop_after_attempt, wait_exponential

class RobustMCPClient:
    def __init__(self, base_url):
        self.base_url = base_url
        self.logger = logging.getLogger(__name__)
    
    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=4, max=10)
    )
    async def call_with_retry(self, endpoint, data):
        try:
            async with aiohttp.ClientSession() as session:
                async with session.post(
                    f"{self.base_url}{endpoint}",
                    json=data,
                    timeout=aiohttp.ClientTimeout(total=30)
                ) as response:
                    if response.status != 200:
                        raise Exception(f"API error: {response.status}")
                    return await response.json()
        except Exception as e:
            self.logger.error(f"MCP call failed: {e}")
            raise
```

### 3. Batch Processing

```python
async def batch_store_chunks(mcp_client, chunks):
    """Store multiple chunks efficiently"""
    tasks = []
    for chunk in chunks:
        task = mcp_client.store_chunk(
            content=chunk["content"],
            session_id=chunk["session_id"],
            tags=chunk.get("tags", [])
        )
        tasks.append(task)
    
    results = await asyncio.gather(*tasks, return_exceptions=True)
    
    # Handle any failures
    for i, result in enumerate(results):
        if isinstance(result, Exception):
            logging.error(f"Failed to store chunk {i}: {result}")
    
    return results
```

### 4. Context Window Management

```python
def manage_context_window(results, max_tokens=4000, tokenizer=None):
    """Manage context to fit within LLM token limits"""
    if not tokenizer:
        # Simple character-based approximation
        total_chars = sum(len(r["content"]) for r in results)
        if total_chars > max_tokens * 4:  # Rough estimate
            # Truncate oldest results
            truncated = []
            char_count = 0
            for result in reversed(results):
                result_len = len(result["content"])
                if char_count + result_len > max_tokens * 4:
                    break
                truncated.insert(0, result)
                char_count += result_len
            return truncated
    return results
```

## Performance Considerations

### 1. Caching Strategy

```python
from functools import lru_cache
import hashlib

class CachedMCPClient:
    def __init__(self, mcp_client, cache_size=100):
        self.mcp_client = mcp_client
        self.cache_size = cache_size
    
    @lru_cache(maxsize=100)
    def _cached_search(self, query_hash, repository):
        # This won't work directly with async, shown for concept
        pass
    
    async def search_with_cache(self, query, repository=None):
        # Create cache key
        cache_key = hashlib.md5(
            f"{query}:{repository}".encode()
        ).hexdigest()
        
        # Check cache first (implement proper async caching)
        # ... cache implementation ...
        
        # If not cached, search
        results = await self.mcp_client.search_memory(
            query=query,
            repository=repository
        )
        
        # Cache results
        # ... cache storage ...
        
        return results
```

### 2. Connection Pooling

```python
class PooledMCPClient:
    def __init__(self, base_url, pool_size=10):
        self.base_url = base_url
        self.connector = aiohttp.TCPConnector(
            limit=pool_size,
            limit_per_host=pool_size
        )
        self.session = None
    
    async def __aenter__(self):
        self.session = aiohttp.ClientSession(
            connector=self.connector
        )
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.session.close()
```

### 3. Streaming Responses

```python
async def stream_search_results(mcp_client, query):
    """Stream results as they arrive"""
    # For HTTP/2 or WebSocket connections
    async for chunk in mcp_client.stream_search(query):
        # Process each result as it arrives
        yield process_result(chunk)
```

## Troubleshooting

### Common Integration Issues

1. **Connection Timeouts**
   - Increase timeout settings
   - Check network connectivity
   - Verify MCP server is running

2. **Authentication Failures**
   - Verify API keys are correct
   - Check CORS settings for browser-based clients
   - Ensure proper headers are sent

3. **Response Format Issues**
   - Validate JSON responses
   - Handle different response formats
   - Implement proper error parsing

4. **Performance Problems**
   - Enable connection pooling
   - Implement caching
   - Use batch operations
   - Monitor memory usage

### Debug Logging

```python
import logging

# Enable debug logging
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

# Log all MCP interactions
class DebugMCPClient:
    def __init__(self, mcp_client):
        self.mcp_client = mcp_client
        self.logger = logging.getLogger("mcp.debug")
    
    async def call_tool(self, tool_name, params):
        self.logger.debug(f"Calling {tool_name} with params: {params}")
        try:
            result = await self.mcp_client.call_tool(tool_name, params)
            self.logger.debug(f"Result: {result}")
            return result
        except Exception as e:
            self.logger.error(f"Error calling {tool_name}: {e}")
            raise
```

## Conclusion

The MCP Memory Server can be integrated with any LLM through various methods. Choose the integration approach that best fits your LLM's capabilities and your application's requirements. The HTTP API provides the most universal compatibility, while WebSocket and custom integrations offer better performance for specific use cases.