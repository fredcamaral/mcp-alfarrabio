# MCP Client Integration Guide

This guide provides comprehensive instructions for building MCP clients that integrate with the Memory Server.

## Table of Contents
- [Overview](#overview)
- [Protocol Basics](#protocol-basics)
- [Client Libraries](#client-libraries)
- [Building a TypeScript Client](#building-a-typescript-client)
- [Building a Python Client](#building-a-python-client)
- [Building a Go Client](#building-a-go-client)
- [Advanced Features](#advanced-features)
- [Testing](#testing)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

MCP (Model Context Protocol) clients communicate with servers using JSON-RPC 2.0 over various transports (stdio, HTTP, WebSocket). This guide covers building robust clients for the Memory Server.

## Protocol Basics

### Message Format

All MCP messages follow JSON-RPC 2.0:

```json
// Request
{
  "jsonrpc": "2.0",
  "method": "tools/list",
  "params": {},
  "id": 1
}

// Response
{
  "jsonrpc": "2.0",
  "result": {
    "tools": [...]
  },
  "id": 1
}

// Error
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32600,
    "message": "Invalid Request"
  },
  "id": 1
}
```

### Available Methods

- `initialize` - Initialize connection
- `tools/list` - List available tools
- `tools/call` - Call a specific tool
- `completion/complete` - Get completions (if supported)

## Client Libraries

### TypeScript/JavaScript

```bash
npm install @modelcontextprotocol/sdk
```

### Python

```bash
pip install mcp-sdk
```

### Go

```bash
go get github.com/yourusername/mcp-memory/pkg/mcp
```

## Building a TypeScript Client

### Basic Client Implementation

```typescript
import { Client, StdioClientTransport } from '@modelcontextprotocol/sdk';
import { spawn } from 'child_process';

// Define tool interfaces
interface MemoryStoreChunkParams {
  content: string;
  session_id: string;
  repository?: string;
  tags?: string[];
  files_modified?: string[];
}

interface MemorySearchParams {
  query: string;
  limit?: number;
  min_relevance?: number;
  repository?: string;
}

class MCPMemoryClient {
  private client: Client;
  private transport: StdioClientTransport;
  private initialized: boolean = false;

  constructor(private serverPath: string, private configPath?: string) {}

  async connect(): Promise<void> {
    // Spawn the server process
    const args = this.configPath ? ['--config', this.configPath] : [];
    const serverProcess = spawn(this.serverPath, args, {
      stdio: ['pipe', 'pipe', 'pipe'],
      env: {
        ...process.env,
        MCP_MEMORY_ENV: 'production'
      }
    });

    // Create transport
    this.transport = new StdioClientTransport({
      stdin: serverProcess.stdin,
      stdout: serverProcess.stdout
    });

    // Create client
    this.client = new Client({
      name: 'mcp-memory-client',
      version: '1.0.0'
    }, {
      capabilities: {}
    });

    // Connect
    await this.client.connect(this.transport);

    // Initialize
    const initResult = await this.client.request({
      method: 'initialize',
      params: {
        protocolVersion: '1.0',
        capabilities: {},
        clientInfo: {
          name: 'mcp-memory-client',
          version: '1.0.0'
        }
      }
    });

    this.initialized = true;
    console.log('Connected to MCP Memory Server', initResult);
  }

  async disconnect(): Promise<void> {
    if (this.client) {
      await this.client.close();
    }
    if (this.transport) {
      await this.transport.close();
    }
  }

  async listTools(): Promise<any[]> {
    this.ensureInitialized();
    const response = await this.client.request({
      method: 'tools/list',
      params: {}
    });
    return response.tools;
  }

  async storeChunk(params: MemoryStoreChunkParams): Promise<any> {
    this.ensureInitialized();
    return await this.callTool('memory_store_chunk', params);
  }

  async search(params: MemorySearchParams): Promise<any> {
    this.ensureInitialized();
    return await this.callTool('memory_search', params);
  }

  async findSimilar(problem: string, repository?: string, limit?: number): Promise<any> {
    this.ensureInitialized();
    return await this.callTool('memory_find_similar', {
      problem,
      repository,
      limit
    });
  }

  async storeDecision(params: {
    decision: string;
    rationale: string;
    context?: string;
    repository?: string;
    session_id: string;
  }): Promise<any> {
    this.ensureInitialized();
    return await this.callTool('memory_store_decision', params);
  }

  async getPatterns(repository: string, timeframe?: string): Promise<any> {
    this.ensureInitialized();
    return await this.callTool('memory_get_patterns', {
      repository,
      timeframe
    });
  }

  async suggestRelated(params: {
    current_context: string;
    session_id: string;
    repository?: string;
    include_patterns?: boolean;
    max_suggestions?: number;
  }): Promise<any> {
    this.ensureInitialized();
    return await this.callTool('memory_suggest_related', params);
  }

  private async callTool(toolName: string, args: any): Promise<any> {
    const response = await this.client.request({
      method: 'tools/call',
      params: {
        name: toolName,
        arguments: args
      }
    });

    if (response.error) {
      throw new Error(`Tool call failed: ${response.error.message}`);
    }

    return response.content;
  }

  private ensureInitialized(): void {
    if (!this.initialized) {
      throw new Error('Client not initialized. Call connect() first.');
    }
  }
}

// Usage example
async function main() {
  const client = new MCPMemoryClient(
    '/path/to/mcp-memory/bin/server',
    '/path/to/config.yaml'
  );

  try {
    await client.connect();

    // List available tools
    const tools = await client.listTools();
    console.log('Available tools:', tools);

    // Store a chunk
    await client.storeChunk({
      content: 'Implemented user authentication with JWT',
      session_id: 'dev-session-123',
      repository: 'my-app',
      tags: ['auth', 'security'],
      files_modified: ['auth/jwt.go']
    });

    // Search for similar content
    const results = await client.search({
      query: 'authentication implementation',
      repository: 'my-app',
      limit: 5
    });

    console.log('Search results:', results);
  } finally {
    await client.disconnect();
  }
}

main().catch(console.error);
```

### HTTP Client Implementation

```typescript
import axios, { AxiosInstance } from 'axios';

class MCPMemoryHTTPClient {
  private client: AxiosInstance;
  private sessionId: string;

  constructor(
    baseURL: string = 'http://localhost:8080',
    private apiKey?: string
  ) {
    this.client = axios.create({
      baseURL,
      headers: {
        'Content-Type': 'application/json',
        ...(apiKey && { 'Authorization': `Bearer ${apiKey}` })
      },
      timeout: 30000
    });

    // Add request/response interceptors
    this.setupInterceptors();
  }

  private setupInterceptors(): void {
    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        console.log(`${config.method?.toUpperCase()} ${config.url}`);
        return config;
      },
      (error) => {
        console.error('Request error:', error);
        return Promise.reject(error);
      }
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => response,
      async (error) => {
        if (error.response?.status === 429) {
          // Rate limit handling
          const retryAfter = error.response.headers['retry-after'] || 5;
          console.log(`Rate limited. Retrying after ${retryAfter}s`);
          await new Promise(resolve => setTimeout(resolve, retryAfter * 1000));
          return this.client.request(error.config);
        }
        return Promise.reject(error);
      }
    );
  }

  async initialize(): Promise<void> {
    const response = await this.client.post('/rpc', {
      jsonrpc: '2.0',
      method: 'initialize',
      params: {
        protocolVersion: '1.0',
        capabilities: {},
        clientInfo: {
          name: 'mcp-memory-http-client',
          version: '1.0.0'
        }
      },
      id: 1
    });

    this.sessionId = response.data.result.sessionId;
  }

  async callTool(toolName: string, params: any): Promise<any> {
    const response = await this.client.post('/rpc', {
      jsonrpc: '2.0',
      method: 'tools/call',
      params: {
        name: toolName,
        arguments: params
      },
      id: Date.now()
    });

    if (response.data.error) {
      throw new Error(response.data.error.message);
    }

    return response.data.result;
  }

  // Convenience methods
  async storeChunk(params: any): Promise<any> {
    return this.callTool('memory_store_chunk', params);
  }

  async search(params: any): Promise<any> {
    return this.callTool('memory_search', params);
  }

  // Batch operations
  async batchCall(calls: Array<{tool: string, params: any}>): Promise<any[]> {
    const requests = calls.map((call, index) => ({
      jsonrpc: '2.0',
      method: 'tools/call',
      params: {
        name: call.tool,
        arguments: call.params
      },
      id: index
    }));

    const response = await this.client.post('/rpc/batch', requests);
    return response.data.map((res: any) => res.result || res.error);
  }
}
```

## Building a Python Client

### Basic Python Client

```python
import json
import asyncio
import subprocess
from typing import Dict, Any, Optional, List
from dataclasses import dataclass
import aiohttp

@dataclass
class MemoryChunk:
    content: str
    session_id: str
    repository: Optional[str] = None
    tags: Optional[List[str]] = None
    files_modified: Optional[List[str]] = None

class MCPMemoryClient:
    """Python client for MCP Memory Server"""
    
    def __init__(self, server_path: str, config_path: Optional[str] = None):
        self.server_path = server_path
        self.config_path = config_path
        self.process = None
        self.reader = None
        self.writer = None
        self.request_id = 0
        self.pending_requests = {}
        
    async def connect(self):
        """Connect to the MCP Memory Server"""
        args = [self.server_path]
        if self.config_path:
            args.extend(['--config', self.config_path])
        
        # Start the server process
        self.process = await asyncio.create_subprocess_exec(
            *args,
            stdin=asyncio.subprocess.PIPE,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        
        self.reader = self.process.stdout
        self.writer = self.process.stdin
        
        # Start message reader
        asyncio.create_task(self._read_messages())
        
        # Initialize connection
        await self._initialize()
    
    async def disconnect(self):
        """Disconnect from the server"""
        if self.process:
            self.process.terminate()
            await self.process.wait()
    
    async def _initialize(self):
        """Initialize the connection"""
        response = await self._request('initialize', {
            'protocolVersion': '1.0',
            'capabilities': {},
            'clientInfo': {
                'name': 'mcp-memory-python-client',
                'version': '1.0.0'
            }
        })
        print(f"Initialized: {response}")
    
    async def _request(self, method: str, params: Dict[str, Any]) -> Any:
        """Send a request and wait for response"""
        self.request_id += 1
        request_id = self.request_id
        
        # Create request
        request = {
            'jsonrpc': '2.0',
            'method': method,
            'params': params,
            'id': request_id
        }
        
        # Create future for response
        future = asyncio.Future()
        self.pending_requests[request_id] = future
        
        # Send request
        await self._send(request)
        
        # Wait for response
        return await future
    
    async def _send(self, message: Dict[str, Any]):
        """Send a message to the server"""
        data = json.dumps(message) + '\n'
        self.writer.write(data.encode())
        await self.writer.drain()
    
    async def _read_messages(self):
        """Read messages from the server"""
        while True:
            try:
                line = await self.reader.readline()
                if not line:
                    break
                
                message = json.loads(line.decode())
                
                # Handle response
                if 'id' in message and message['id'] in self.pending_requests:
                    future = self.pending_requests.pop(message['id'])
                    if 'error' in message:
                        future.set_exception(Exception(message['error']['message']))
                    else:
                        future.set_result(message.get('result'))
                
            except Exception as e:
                print(f"Error reading message: {e}")
    
    async def call_tool(self, tool_name: str, arguments: Dict[str, Any]) -> Any:
        """Call a tool on the server"""
        return await self._request('tools/call', {
            'name': tool_name,
            'arguments': arguments
        })
    
    async def store_chunk(self, chunk: MemoryChunk) -> Dict[str, Any]:
        """Store a memory chunk"""
        params = {
            'content': chunk.content,
            'session_id': chunk.session_id
        }
        if chunk.repository:
            params['repository'] = chunk.repository
        if chunk.tags:
            params['tags'] = chunk.tags
        if chunk.files_modified:
            params['files_modified'] = chunk.files_modified
        
        return await self.call_tool('memory_store_chunk', params)
    
    async def search(self, 
                    query: str, 
                    limit: int = 10,
                    min_relevance: float = 0.7,
                    repository: Optional[str] = None) -> List[Dict[str, Any]]:
        """Search for similar chunks"""
        params = {
            'query': query,
            'limit': limit,
            'min_relevance': min_relevance
        }
        if repository:
            params['repository'] = repository
        
        result = await self.call_tool('memory_search', params)
        return result.get('results', [])
    
    async def find_similar(self, 
                          problem: str,
                          repository: Optional[str] = None,
                          limit: int = 5) -> List[Dict[str, Any]]:
        """Find similar problems and solutions"""
        params = {'problem': problem, 'limit': limit}
        if repository:
            params['repository'] = repository
        
        return await self.call_tool('memory_find_similar', params)
    
    async def store_decision(self,
                           decision: str,
                           rationale: str,
                           session_id: str,
                           context: Optional[str] = None,
                           repository: Optional[str] = None) -> Dict[str, Any]:
        """Store an architectural decision"""
        params = {
            'decision': decision,
            'rationale': rationale,
            'session_id': session_id
        }
        if context:
            params['context'] = context
        if repository:
            params['repository'] = repository
        
        return await self.call_tool('memory_store_decision', params)

# Usage example
async def main():
    client = MCPMemoryClient(
        '/path/to/mcp-memory/bin/server',
        '/path/to/config.yaml'
    )
    
    try:
        await client.connect()
        
        # Store a chunk
        chunk = MemoryChunk(
            content="Implemented OAuth2 authentication flow",
            session_id="dev-session-001",
            repository="my-app",
            tags=["auth", "oauth2"],
            files_modified=["auth/oauth.py", "auth/providers.py"]
        )
        
        result = await client.store_chunk(chunk)
        print(f"Stored chunk: {result}")
        
        # Search for similar content
        results = await client.search(
            query="authentication implementation",
            repository="my-app",
            limit=5
        )
        
        for result in results:
            print(f"Found: {result['content'][:100]}...")
            print(f"Relevance: {result['relevance']}")
            print("---")
        
    finally:
        await client.disconnect()

if __name__ == "__main__":
    asyncio.run(main())
```

### HTTP Client with Retry Logic

```python
import aiohttp
import asyncio
from typing import Dict, Any, Optional
import backoff
import logging

class MCPMemoryHTTPClient:
    """HTTP client for MCP Memory Server with retry logic"""
    
    def __init__(self, base_url: str = "http://localhost:8080", api_key: Optional[str] = None):
        self.base_url = base_url
        self.api_key = api_key
        self.session = None
        self.logger = logging.getLogger(__name__)
    
    async def __aenter__(self):
        """Async context manager entry"""
        self.session = aiohttp.ClientSession(
            headers={
                'Content-Type': 'application/json',
                **({"Authorization": f"Bearer {self.api_key}"} if self.api_key else {})
            }
        )
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        if self.session:
            await self.session.close()
    
    @backoff.on_exception(
        backoff.expo,
        aiohttp.ClientError,
        max_tries=3,
        max_time=30
    )
    async def _request(self, method: str, endpoint: str, **kwargs) -> Dict[str, Any]:
        """Make HTTP request with retry logic"""
        url = f"{self.base_url}{endpoint}"
        
        async with self.session.request(method, url, **kwargs) as response:
            if response.status >= 400:
                error_text = await response.text()
                raise aiohttp.ClientResponseError(
                    request_info=response.request_info,
                    history=response.history,
                    status=response.status,
                    message=error_text
                )
            
            return await response.json()
    
    async def call_tool(self, tool_name: str, arguments: Dict[str, Any]) -> Any:
        """Call a tool via HTTP"""
        response = await self._request(
            'POST',
            '/rpc',
            json={
                'jsonrpc': '2.0',
                'method': 'tools/call',
                'params': {
                    'name': tool_name,
                    'arguments': arguments
                },
                'id': 1
            }
        )
        
        if 'error' in response:
            raise Exception(f"Tool call failed: {response['error']['message']}")
        
        return response['result']
    
    # Streaming support
    async def search_stream(self, query: str, **kwargs):
        """Stream search results as they arrive"""
        url = f"{self.base_url}/stream/search"
        
        async with self.session.post(
            url,
            json={'query': query, **kwargs}
        ) as response:
            async for line in response.content:
                if line:
                    yield json.loads(line)

# WebSocket client
class MCPMemoryWebSocketClient:
    """WebSocket client for real-time MCP Memory operations"""
    
    def __init__(self, ws_url: str = "ws://localhost:8080/ws"):
        self.ws_url = ws_url
        self.ws = None
        self.request_id = 0
        self.pending_requests = {}
    
    async def connect(self):
        """Connect to WebSocket server"""
        self.ws = await aiohttp.ClientSession().ws_connect(self.ws_url)
        
        # Start message handler
        asyncio.create_task(self._handle_messages())
    
    async def _handle_messages(self):
        """Handle incoming WebSocket messages"""
        async for msg in self.ws:
            if msg.type == aiohttp.WSMsgType.TEXT:
                data = json.loads(msg.data)
                
                # Handle response
                if 'id' in data and data['id'] in self.pending_requests:
                    future = self.pending_requests.pop(data['id'])
                    if 'error' in data:
                        future.set_exception(Exception(data['error']['message']))
                    else:
                        future.set_result(data.get('result'))
    
    async def call_tool(self, tool_name: str, arguments: Dict[str, Any]) -> Any:
        """Call tool via WebSocket"""
        self.request_id += 1
        request_id = self.request_id
        
        # Create future for response
        future = asyncio.Future()
        self.pending_requests[request_id] = future
        
        # Send request
        await self.ws.send_json({
            'jsonrpc': '2.0',
            'method': 'tools/call',
            'params': {
                'name': tool_name,
                'arguments': arguments
            },
            'id': request_id
        })
        
        # Wait for response
        return await future
```

## Building a Go Client

### Basic Go Client

```go
package mcpclient

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os/exec"
    "sync"
    "time"
)

// Client represents an MCP client
type Client struct {
    serverPath   string
    configPath   string
    cmd          *exec.Cmd
    stdin        io.WriteCloser
    stdout       io.ReadCloser
    requestID    int64
    pending      map[int64]chan *Response
    pendingMu    sync.Mutex
    initialized  bool
}

// Request represents a JSON-RPC request
type Request struct {
    JSONRPC string      `json:"jsonrpc"`
    Method  string      `json:"method"`
    Params  interface{} `json:"params"`
    ID      int64       `json:"id"`
}

// Response represents a JSON-RPC response
type Response struct {
    JSONRPC string          `json:"jsonrpc"`
    Result  json.RawMessage `json:"result,omitempty"`
    Error   *Error          `json:"error,omitempty"`
    ID      int64           `json:"id"`
}

// Error represents a JSON-RPC error
type Error struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

// NewClient creates a new MCP client
func NewClient(serverPath string, configPath string) *Client {
    return &Client{
        serverPath: serverPath,
        configPath: configPath,
        pending:    make(map[int64]chan *Response),
    }
}

// Connect establishes connection to the MCP server
func (c *Client) Connect(ctx context.Context) error {
    args := []string{}
    if c.configPath != "" {
        args = append(args, "--config", c.configPath)
    }
    
    c.cmd = exec.CommandContext(ctx, c.serverPath, args...)
    
    var err error
    c.stdin, err = c.cmd.StdinPipe()
    if err != nil {
        return fmt.Errorf("failed to get stdin pipe: %w", err)
    }
    
    c.stdout, err = c.cmd.StdoutPipe()
    if err != nil {
        return fmt.Errorf("failed to get stdout pipe: %w", err)
    }
    
    if err := c.cmd.Start(); err != nil {
        return fmt.Errorf("failed to start server: %w", err)
    }
    
    // Start message reader
    go c.readMessages()
    
    // Initialize connection
    return c.initialize(ctx)
}

// Disconnect closes the connection
func (c *Client) Disconnect() error {
    if c.stdin != nil {
        c.stdin.Close()
    }
    if c.cmd != nil {
        return c.cmd.Wait()
    }
    return nil
}

func (c *Client) initialize(ctx context.Context) error {
    params := map[string]interface{}{
        "protocolVersion": "1.0",
        "capabilities":    map[string]interface{}{},
        "clientInfo": map[string]string{
            "name":    "mcp-memory-go-client",
            "version": "1.0.0",
        },
    }
    
    var result map[string]interface{}
    if err := c.Request(ctx, "initialize", params, &result); err != nil {
        return fmt.Errorf("initialization failed: %w", err)
    }
    
    c.initialized = true
    return nil
}

// Request sends a request and waits for response
func (c *Client) Request(ctx context.Context, method string, params interface{}, result interface{}) error {
    c.pendingMu.Lock()
    c.requestID++
    id := c.requestID
    ch := make(chan *Response, 1)
    c.pending[id] = ch
    c.pendingMu.Unlock()
    
    defer func() {
        c.pendingMu.Lock()
        delete(c.pending, id)
        c.pendingMu.Unlock()
    }()
    
    // Send request
    req := Request{
        JSONRPC: "2.0",
        Method:  method,
        Params:  params,
        ID:      id,
    }
    
    if err := json.NewEncoder(c.stdin).Encode(req); err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    
    // Wait for response
    select {
    case resp := <-ch:
        if resp.Error != nil {
            return fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
        }
        
        if result != nil && len(resp.Result) > 0 {
            return json.Unmarshal(resp.Result, result)
        }
        return nil
        
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (c *Client) readMessages() {
    scanner := bufio.NewScanner(c.stdout)
    for scanner.Scan() {
        var resp Response
        if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
            continue
        }
        
        c.pendingMu.Lock()
        if ch, ok := c.pending[resp.ID]; ok {
            ch <- &resp
        }
        c.pendingMu.Unlock()
    }
}

// CallTool calls a specific tool
func (c *Client) CallTool(ctx context.Context, toolName string, arguments interface{}) (json.RawMessage, error) {
    if !c.initialized {
        return nil, fmt.Errorf("client not initialized")
    }
    
    params := map[string]interface{}{
        "name":      toolName,
        "arguments": arguments,
    }
    
    var result struct {
        Content json.RawMessage `json:"content"`
    }
    
    if err := c.Request(ctx, "tools/call", params, &result); err != nil {
        return nil, err
    }
    
    return result.Content, nil
}

// Memory-specific methods

// StoreChunk stores a memory chunk
func (c *Client) StoreChunk(ctx context.Context, params StoreChunkParams) error {
    _, err := c.CallTool(ctx, "memory_store_chunk", params)
    return err
}

// Search searches for similar chunks
func (c *Client) Search(ctx context.Context, params SearchParams) (*SearchResult, error) {
    result, err := c.CallTool(ctx, "memory_search", params)
    if err != nil {
        return nil, err
    }
    
    var searchResult SearchResult
    if err := json.Unmarshal(result, &searchResult); err != nil {
        return nil, err
    }
    
    return &searchResult, nil
}

// Types for memory operations
type StoreChunkParams struct {
    Content       string   `json:"content"`
    SessionID     string   `json:"session_id"`
    Repository    string   `json:"repository,omitempty"`
    Tags          []string `json:"tags,omitempty"`
    FilesModified []string `json:"files_modified,omitempty"`
}

type SearchParams struct {
    Query        string  `json:"query"`
    Limit        int     `json:"limit,omitempty"`
    MinRelevance float64 `json:"min_relevance,omitempty"`
    Repository   string  `json:"repository,omitempty"`
}

type SearchResult struct {
    Results []struct {
        Content   string    `json:"content"`
        Relevance float64   `json:"relevance"`
        Timestamp time.Time `json:"timestamp"`
        Metadata  map[string]interface{} `json:"metadata"`
    } `json:"results"`
}

// Example usage
func Example() {
    ctx := context.Background()
    client := NewClient("/path/to/server", "/path/to/config.yaml")
    
    // Connect
    if err := client.Connect(ctx); err != nil {
        panic(err)
    }
    defer client.Disconnect()
    
    // Store a chunk
    err := client.StoreChunk(ctx, StoreChunkParams{
        Content:    "Implemented user authentication",
        SessionID:  "session-123",
        Repository: "my-app",
        Tags:       []string{"auth", "security"},
    })
    if err != nil {
        panic(err)
    }
    
    // Search
    results, err := client.Search(ctx, SearchParams{
        Query:      "authentication",
        Repository: "my-app",
        Limit:      5,
    })
    if err != nil {
        panic(err)
    }
    
    for _, result := range results.Results {
        fmt.Printf("Found: %s (relevance: %.2f)\n", 
            result.Content[:50], result.Relevance)
    }
}
```

### HTTP Client with Connection Pooling

```go
package mcpclient

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// HTTPClient provides HTTP-based MCP client
type HTTPClient struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
}

// NewHTTPClient creates a new HTTP-based MCP client
func NewHTTPClient(baseURL string, apiKey string) *HTTPClient {
    return &HTTPClient{
        baseURL: baseURL,
        apiKey:  apiKey,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
            },
        },
    }
}

// CallTool calls a tool via HTTP
func (c *HTTPClient) CallTool(ctx context.Context, toolName string, arguments interface{}) (json.RawMessage, error) {
    request := map[string]interface{}{
        "jsonrpc": "2.0",
        "method":  "tools/call",
        "params": map[string]interface{}{
            "name":      toolName,
            "arguments": arguments,
        },
        "id": time.Now().UnixNano(),
    }
    
    body, err := json.Marshal(request)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    
    req, err := http.NewRequestWithContext(
        ctx,
        "POST",
        fmt.Sprintf("%s/rpc", c.baseURL),
        bytes.NewReader(body),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    if c.apiKey != "" {
        req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()
    
    var response struct {
        Result json.RawMessage `json:"result"`
        Error  *struct {
            Code    int    `json:"code"`
            Message string `json:"message"`
        } `json:"error"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    if response.Error != nil {
        return nil, fmt.Errorf("RPC error %d: %s", 
            response.Error.Code, response.Error.Message)
    }
    
    return response.Result, nil
}
```

## Advanced Features

### Connection Management

```typescript
class ConnectionManager {
  private clients: Map<string, MCPMemoryClient> = new Map();
  private maxClients: number = 10;

  async getClient(serverPath: string, configPath?: string): Promise<MCPMemoryClient> {
    const key = `${serverPath}:${configPath || 'default'}`;
    
    if (this.clients.has(key)) {
      return this.clients.get(key)!;
    }

    if (this.clients.size >= this.maxClients) {
      // Evict least recently used
      const lru = this.clients.keys().next().value;
      const client = this.clients.get(lru)!;
      await client.disconnect();
      this.clients.delete(lru);
    }

    const client = new MCPMemoryClient(serverPath, configPath);
    await client.connect();
    this.clients.set(key, client);
    
    return client;
  }

  async closeAll(): Promise<void> {
    const promises = Array.from(this.clients.values()).map(
      client => client.disconnect()
    );
    await Promise.all(promises);
    this.clients.clear();
  }
}
```

### Request Batching

```typescript
class BatchingMCPClient extends MCPMemoryClient {
  private batchQueue: Array<{
    method: string;
    params: any;
    resolve: (value: any) => void;
    reject: (error: any) => void;
  }> = [];
  private batchTimeout: NodeJS.Timeout | null = null;
  private batchSize: number = 10;
  private batchDelay: number = 100; // ms

  async callTool(toolName: string, args: any): Promise<any> {
    return new Promise((resolve, reject) => {
      this.batchQueue.push({
        method: toolName,
        params: args,
        resolve,
        reject
      });

      if (this.batchQueue.length >= this.batchSize) {
        this.flushBatch();
      } else if (!this.batchTimeout) {
        this.batchTimeout = setTimeout(() => this.flushBatch(), this.batchDelay);
      }
    });
  }

  private async flushBatch(): Promise<void> {
    if (this.batchTimeout) {
      clearTimeout(this.batchTimeout);
      this.batchTimeout = null;
    }

    const batch = this.batchQueue.splice(0, this.batchSize);
    if (batch.length === 0) return;

    const requests = batch.map((item, index) => ({
      jsonrpc: '2.0',
      method: 'tools/call',
      params: {
        name: item.method,
        arguments: item.params
      },
      id: index
    }));

    try {
      const responses = await this.client.request({
        method: 'batch',
        params: { requests }
      });

      batch.forEach((item, index) => {
        const response = responses[index];
        if (response.error) {
          item.reject(new Error(response.error.message));
        } else {
          item.resolve(response.result);
        }
      });
    } catch (error) {
      batch.forEach(item => item.reject(error));
    }
  }
}
```

### Event Streaming

```typescript
class StreamingMCPClient extends MCPMemoryClient {
  async *streamSearch(params: MemorySearchParams): AsyncGenerator<any> {
    const stream = await this.client.request({
      method: 'tools/stream',
      params: {
        name: 'memory_search',
        arguments: params,
        stream: true
      }
    });

    for await (const chunk of stream) {
      yield chunk;
    }
  }

  async subscribeToPatterns(repository: string, callback: (pattern: any) => void): Promise<() => void> {
    const subscription = await this.client.request({
      method: 'subscribe',
      params: {
        event: 'patterns',
        filter: { repository }
      }
    });

    this.client.on(`patterns:${subscription.id}`, callback);

    return () => {
      this.client.request({
        method: 'unsubscribe',
        params: { id: subscription.id }
      });
      this.client.off(`patterns:${subscription.id}`, callback);
    };
  }
}
```

## Testing

### Unit Testing

```typescript
import { jest } from '@jest/globals';

describe('MCPMemoryClient', () => {
  let client: MCPMemoryClient;
  let mockTransport: any;

  beforeEach(() => {
    mockTransport = {
      send: jest.fn(),
      on: jest.fn(),
      close: jest.fn()
    };

    client = new MCPMemoryClient('/fake/path');
    // Override transport
    (client as any).transport = mockTransport;
  });

  test('storeChunk sends correct request', async () => {
    const params = {
      content: 'Test content',
      session_id: 'test-session',
      tags: ['test']
    };

    mockTransport.send.mockResolvedValue({
      jsonrpc: '2.0',
      result: { success: true },
      id: 1
    });

    await client.storeChunk(params);

    expect(mockTransport.send).toHaveBeenCalledWith({
      jsonrpc: '2.0',
      method: 'tools/call',
      params: {
        name: 'memory_store_chunk',
        arguments: params
      },
      id: expect.any(Number)
    });
  });
});
```

### Integration Testing

```python
import pytest
import asyncio
from mcpclient import MCPMemoryClient

@pytest.fixture
async def client():
    """Create test client"""
    client = MCPMemoryClient(
        server_path='./test-server',
        config_path='./test-config.yaml'
    )
    await client.connect()
    yield client
    await client.disconnect()

@pytest.mark.asyncio
async def test_store_and_search(client):
    """Test storing and searching chunks"""
    # Store test data
    await client.store_chunk({
        'content': 'Test implementation of feature X',
        'session_id': 'test-session',
        'repository': 'test-repo',
        'tags': ['test', 'feature-x']
    })
    
    # Search for it
    results = await client.search({
        'query': 'feature X implementation',
        'repository': 'test-repo'
    })
    
    assert len(results) > 0
    assert 'feature X' in results[0]['content']
```

## Best Practices

### 1. Error Handling

Always implement comprehensive error handling:

```typescript
class RobustMCPClient extends MCPMemoryClient {
  async callToolWithRetry(toolName: string, args: any, maxRetries: number = 3): Promise<any> {
    let lastError: Error;
    
    for (let i = 0; i < maxRetries; i++) {
      try {
        return await this.callTool(toolName, args);
      } catch (error) {
        lastError = error as Error;
        
        // Don't retry on client errors
        if (error.message.includes('Invalid arguments')) {
          throw error;
        }
        
        // Exponential backoff
        const delay = Math.pow(2, i) * 1000;
        await new Promise(resolve => setTimeout(resolve, delay));
      }
    }
    
    throw lastError!;
  }
}
```

### 2. Resource Management

Properly manage client lifecycle:

```typescript
class ManagedMCPClient {
  private client: MCPMemoryClient;
  private heartbeatInterval: NodeJS.Timer;

  async initialize(): Promise<void> {
    this.client = new MCPMemoryClient('/path/to/server');
    await this.client.connect();
    
    // Start heartbeat
    this.heartbeatInterval = setInterval(() => {
      this.client.ping().catch(error => {
        console.error('Heartbeat failed:', error);
        this.reconnect();
      });
    }, 30000);
  }

  async shutdown(): Promise<void> {
    clearInterval(this.heartbeatInterval);
    await this.client.disconnect();
  }

  private async reconnect(): Promise<void> {
    try {
      await this.client.disconnect();
    } catch (error) {
      // Ignore disconnect errors
    }
    
    await this.client.connect();
  }
}
```

### 3. Logging and Monitoring

Implement comprehensive logging:

```typescript
import winston from 'winston';

class LoggingMCPClient extends MCPMemoryClient {
  private logger: winston.Logger;

  constructor(serverPath: string, configPath?: string) {
    super(serverPath, configPath);
    
    this.logger = winston.createLogger({
      level: 'info',
      format: winston.format.json(),
      transports: [
        new winston.transports.File({ filename: 'mcp-client.log' })
      ]
    });
  }

  async callTool(toolName: string, args: any): Promise<any> {
    const startTime = Date.now();
    
    try {
      this.logger.info('Tool call started', { toolName, args });
      const result = await super.callTool(toolName, args);
      
      this.logger.info('Tool call completed', {
        toolName,
        duration: Date.now() - startTime
      });
      
      return result;
    } catch (error) {
      this.logger.error('Tool call failed', {
        toolName,
        error: error.message,
        duration: Date.now() - startTime
      });
      throw error;
    }
  }
}
```

## Troubleshooting

### Common Issues

1. **Connection Failures**
   ```typescript
   // Check server is running
   const checkServer = async (serverPath: string): Promise<boolean> => {
     try {
       const process = spawn(serverPath, ['--version']);
       return new Promise(resolve => {
         process.on('exit', code => resolve(code === 0));
       });
     } catch {
       return false;
     }
   };
   ```

2. **Message Parsing Errors**
   ```typescript
   // Validate messages before parsing
   const parseMessage = (data: string): any => {
     try {
       const message = JSON.parse(data);
       if (!message.jsonrpc || message.jsonrpc !== '2.0') {
         throw new Error('Invalid JSON-RPC version');
       }
       return message;
     } catch (error) {
       console.error('Failed to parse message:', data);
       throw error;
     }
   };
   ```

3. **Memory Leaks**
   ```typescript
   // Clean up pending requests on disconnect
   class CleanMCPClient extends MCPMemoryClient {
     async disconnect(): Promise<void> {
       // Cancel all pending requests
       for (const [id, handler] of this.pendingRequests) {
         handler.reject(new Error('Client disconnecting'));
       }
       this.pendingRequests.clear();
       
       await super.disconnect();
     }
   }
   ```

## Conclusion

Building robust MCP clients requires careful attention to protocol details, error handling, and resource management. This guide provides patterns and examples for creating clients in multiple languages that can reliably integrate with the MCP Memory Server.