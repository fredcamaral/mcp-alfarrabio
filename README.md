# Lerian MCP Memory Server

> **Smart memory for AI assistants** - A high-performance Go-based Model Context Protocol (MCP) server that remembers your conversations, learns from patterns, and provides intelligent context suggestions.

Perfect for **Claude Desktop**, **VS Code**, **Continue**, **Cursor**, and any MCP-compatible AI client.

## 🚀 Quick Start (5 minutes)

### Step 1: Start the Server

```bash
git clone https://github.com/LerianStudio/lerian-mcp-memory.git
cd lerian-mcp-memory
cp .env.example .env
# Edit .env and add your OPENAI_API_KEY

# Start everything
docker-compose up -d
```

### Step 2: Choose Your Connection Method

The MCP Memory Server supports **multiple transport protocols** for maximum compatibility:

| Protocol | Use Case | Configuration Complexity |
|----------|----------|-------------------------|
| **📡 stdio + proxy** | Legacy MCP clients (Claude Desktop, VS Code) | Easy |
| **🔌 WebSocket** | Real-time bidirectional communication | Easy |
| **📤 SSE (Server-Sent Events)** | Event streaming + direct HTTP | Medium |
| **🌐 Direct HTTP** | Simple request/response | Easy |

---

## 🔌 MCP Protocol Options

### Option 1: stdio + proxy (Recommended for Most Clients)

**Best for:** Claude Desktop, Claude Code CLI, VS Code with Continue, Cursor

The server includes a Node.js proxy that bridges stdio ↔ HTTP for full compatibility with existing MCP clients.

```json
{
  "mcpServers": {
    "memory": {
      "command": "docker",
      "args": ["exec", "-i", "lerian-mcp-memory-server", "node", "/app/mcp-proxy.js"],
      "env": {
        "MCP_SERVER_HOST": "localhost",
        "MCP_SERVER_PORT": "9080",
        "MCP_SERVER_PATH": "/mcp"
      }
    }
  }
}
```

### Option 2: WebSocket (Real-time Bidirectional)

**Best for:** Custom applications, real-time use cases

```javascript
const ws = new WebSocket('ws://localhost:9080/ws');
ws.send(JSON.stringify({
  jsonrpc: "2.0",
  method: "initialize",
  params: { protocolVersion: "2024-11-05", capabilities: {} },
  id: 1
}));
```

### Option 3: Server-Sent Events (Event Streaming)

**Best for:** Web applications, Claude/Cursor with SSE support, real-time updates

#### For MCP Clients with SSE Support:
```json
{
  "mcpServers": {
    "memory": {
      "transport": "sse",
      "url": "http://localhost:9080/sse",
      "env": {
        "MCP_SERVER_HOST": "localhost",
        "MCP_SERVER_PORT": "9080"
      }
    }
  }
}
```

#### For Custom Applications:
```javascript
// Stream connection for real-time events
const eventSource = new EventSource('http://localhost:9080/sse');
eventSource.onmessage = (event) => {
  const response = JSON.parse(event.data);
  console.log('Received:', response);
};

// Direct JSON-RPC requests
fetch('http://localhost:9080/sse', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    jsonrpc: "2.0",
    method: "tools/list",
    id: 1
  })
}).then(res => res.json());
```

#### For Python Applications:
```python
import requests
import sseclient

# SSE streaming
response = requests.get('http://localhost:9080/sse', stream=True)
client = sseclient.SSEClient(response)
for event in client.events():
    print(f"Received: {event.data}")

# Direct JSON-RPC
response = requests.post('http://localhost:9080/sse', json={
    "jsonrpc": "2.0",
    "method": "memory_search",
    "params": {"query": "example"},
    "id": 1
})
```

### Option 4: Direct HTTP (Simple REST-like)

**Best for:** Testing, simple integrations

```bash
curl -X POST http://localhost:9080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

---

## 🛠️ Client-Specific Configurations

<details>
<summary><b>🖥️ Claude Desktop</b></summary>

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (Mac) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "memory": {
      "command": "docker",
      "args": ["exec", "-i", "lerian-mcp-memory-server", "node", "/app/mcp-proxy.js"],
      "env": {
        "MCP_SERVER_HOST": "localhost",
        "MCP_SERVER_PORT": "9080",
        "MCP_SERVER_PATH": "/mcp"
      }
    }
  }
}
```
</details>

<details>
<summary><b>💻 Claude Code CLI</b></summary>

Add to `.claude/mcp.json` in your project root:

```json
{
  "mcpServers": {
    "memory": {
      "command": "docker",
      "args": ["exec", "-i", "lerian-mcp-memory-server", "node", "/app/mcp-proxy.js"],
      "env": {
        "MCP_SERVER_HOST": "localhost",
        "MCP_SERVER_PORT": "9080",
        "MCP_SERVER_PATH": "/mcp"
      }
    }
  }
}
```
</details>

<details>
<summary><b>📝 VS Code with Continue</b></summary>

Add to your Continue configuration:

```json
{
  "models": [...],
  "mcpServers": {
    "memory": {
      "command": "docker",
      "args": ["exec", "-i", "lerian-mcp-memory-server", "node", "/app/mcp-proxy.js"],
      "env": {
        "MCP_SERVER_HOST": "localhost",
        "MCP_SERVER_PORT": "9080",
        "MCP_SERVER_PATH": "/mcp"
      }
    }
  }
}
```
</details>

<details>
<summary><b>🖱️ Cursor, Windsurf, Other MCP Clients</b></summary>

#### Option A: stdio + proxy (Recommended)
```json
{
  "mcpServers": {
    "memory": {
      "command": "docker",
      "args": ["exec", "-i", "lerian-mcp-memory-server", "node", "/app/mcp-proxy.js"],
      "env": {
        "MCP_SERVER_HOST": "localhost",
        "MCP_SERVER_PORT": "9080", 
        "MCP_SERVER_PATH": "/mcp"
      }
    }
  }
}
```

#### Option B: Direct SSE (If client supports it)
```json
{
  "mcpServers": {
    "memory": {
      "transport": "sse",
      "url": "http://localhost:9080/sse",
      "timeout": 30000,
      "env": {
        "MCP_SERVER_HOST": "localhost",
        "MCP_SERVER_PORT": "9080"
      }
    }
  }
}
```

#### Option C: WebSocket (If client supports it)
```json
{
  "mcpServers": {
    "memory": {
      "transport": "websocket",
      "url": "ws://localhost:9080/ws",
      "timeout": 30000,
      "reconnect": true
    }
  }
}
```
</details>

---

## 🎯 What Does This Do?

**Lerian MCP Memory** transforms your AI assistant into a smart companion that:

- **📚 Remembers Everything**: Stores conversations and contexts across sessions
- **🔍 Smart Search**: AI-powered similarity search through your history  
- **🧠 Pattern Learning**: Recognizes your preferences and coding patterns
- **💡 Proactive Suggestions**: Suggests relevant context automatically
- **🔄 Cross-Project Intelligence**: Learns across all your repositories

### 🛠️ Available Memory Tools

Once configured, your AI gets 9 powerful consolidated memory tools:

- `memory_create` - Store conversations, decisions, and create memory structures
- `memory_read` - Search, find patterns, and retrieve context across projects  
- `memory_update` - Refresh memories and manage relationships
- `memory_delete` - Remove outdated or bulk delete memories
- `memory_intelligence` - AI-powered insights and pattern suggestions
- `memory_transfer` - Export/import contexts across projects
- `memory_tasks` - Track workflows and todo management
- `memory_analyze` - Cross-repo patterns and health monitoring
- `memory_system` - Health checks and system documentation

---

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   MCP Clients   │    │   Go MCP Server  │    │    Storage      │
│                 │    │                  │    │                 │
│ Claude Desktop  │◄──►│ stdio + proxy    │    │    Qdrant       │
│ Claude Code CLI │    │ WebSocket        │◄──►│   (Vectors)     │
│ VS Code/Continue│    │ SSE              │    │                 │
│ Cursor/Windsurf │    │ Direct HTTP      │    │   SQLite        │
│ Custom Apps     │    │                  │    │   (Metadata)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

**Available Endpoints:**
- `http://localhost:9080/mcp` - Direct HTTP JSON-RPC
- `http://localhost:9080/sse` - Server-Sent Events + HTTP
- `ws://localhost:9080/ws` - WebSocket bidirectional
- `http://localhost:8081/health` - Health check
- `http://localhost:8082` - Metrics (optional)

---

## 🔧 Troubleshooting

### Quick Diagnostics

```bash
# Check if everything is running
docker-compose ps

# Test the server
curl http://localhost:8081/health

# Test MCP proxy
echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}},"id":1}' | docker exec -i lerian-mcp-memory-server node /app/mcp-proxy.js

# View logs
docker logs lerian-mcp-memory-server
```

### Common Issues

**🔴 Connection refused**
```bash
docker-compose restart
```

**🔴 OpenAI API errors**
- Check your API key in `.env`
- Verify account credits

**🔴 Node.js not found**
```bash
docker-compose build --no-cache
```

### Testing Individual Protocols

```bash
# Test HTTP endpoint
curl -X POST http://localhost:9080/mcp -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'

# Test SSE endpoint  
curl -X POST http://localhost:9080/sse -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}},"id":1}'

# Test WebSocket (requires ws tool)
echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}},"id":1}' | websocat ws://localhost:9080/ws
```

---

## 🎛️ Advanced Configuration

### Environment Variables

```bash
# Required
OPENAI_API_KEY=your-key-here

# Optional server configuration  
MCP_MEMORY_LOG_LEVEL=info
QDRANT_HOST_PORT=6333
QDRANT_GRPC_PORT=6334
MCP_HOST_PORT=9080
MCP_HEALTH_PORT=8081
MCP_METRICS_PORT=8082

# Storage configuration
MCP_MEMORY_DB_TYPE=sqlite
SQLITE_DB_PATH=/app/data/memory.db
QDRANT_COLLECTION=claude_memory
MCP_MEMORY_EMBEDDING_DIMENSION=1536

# Security settings
MCP_MEMORY_ENCRYPTION_ENABLED=true
MCP_MEMORY_BACKUP_ENABLED=true
MCP_MEMORY_BACKUP_INTERVAL_HOURS=24
MCP_MEMORY_ACCESS_CONTROL_ENABLED=true

# CORS configuration
MCP_MEMORY_CORS_ENABLED=true
MCP_MEMORY_CORS_ORIGINS=http://localhost:*,https://localhost:*

# Optional proxy configuration (for stdio clients)
MCP_SERVER_HOST=localhost
MCP_SERVER_PORT=9080
MCP_SERVER_PATH=/mcp
MCP_PROXY_DEBUG=false
```

### Development Mode

For local development:

```bash
# Run natively (requires Go 1.23+)
go run ./cmd/server -mode=stdio

# Or in HTTP mode for testing
go run ./cmd/server -mode=http -addr=:9080

# Build binary
make build
./bin/lerian-mcp-memory-server -mode=http
```

### Production Deployment

For production use:
- Set proper environment variables in `.env`
- Use Docker Compose with volume mounts for data persistence
- Configure proper backup intervals
- Monitor health endpoint: `http://localhost:8081/health`
- Use `docker-compose logs -f` for monitoring

---

## 📚 Key Features

### 🧠 Intelligence Engine
- **Pattern Recognition**: Learns from your coding patterns and preferences
- **Conflict Detection**: Identifies contradictory decisions or approaches
- **Context Suggestions**: Proactively suggests relevant historical context
- **Cross-Repository Learning**: Shares insights across different projects

### 🔄 Multi-Protocol Support
- **stdio compatibility**: Works with existing MCP clients
- **WebSocket**: Real-time bidirectional communication
- **Server-Sent Events**: Event streaming with HTTP fallback
- **Direct HTTP**: Simple JSON-RPC over HTTP

### 🏪 Storage & Performance
- **Qdrant Vector Database**: High-performance similarity search
- **SQLite Metadata**: Fast local storage for relationships and metadata
- **Intelligent Chunking**: Optimizes content for vector embeddings
- **Circuit Breakers**: Reliable external service integration

### 🔒 Security & Reliability
- **Access Control**: Configurable authentication and authorization
- **Data Encryption**: Optional encryption for sensitive data
- **Automatic Backups**: Scheduled data protection
- **Health Monitoring**: Built-in health checks and metrics

---

## 🤝 Contributing

We welcome contributions! See [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

Apache 2.0 License - see [LICENSE](LICENSE) file for details.

---

**🚀 Ready to give your AI assistant a perfect memory?** 

Start with the [Quick Start](#-quick-start-5-minutes) above and choose your preferred [protocol option](#-mcp-protocol-options).

**Questions?** [Open an issue](https://github.com/LerianStudio/lerian-mcp-memory/issues) or check the troubleshooting section above.