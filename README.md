# MCP Memory Server

> **Smart memory for AI assistants** - A Model Context Protocol (MCP) server that remembers your conversations, learns from patterns, and provides intelligent context suggestions.

Perfect for **Claude Desktop**, **VS Code**, **Continue**, **Cursor**, and any MCP-compatible AI client.

## 🚀 Quick Start (5 minutes)

### Step 1: Start the Server

```bash
git clone https://github.com/LerianStudio/mcp-memory.git
cd mcp-memory
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
      "args": ["exec", "-i", "mcp-memory-server", "node", "/app/mcp-proxy.js"],
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
      "args": ["exec", "-i", "mcp-memory-server", "node", "/app/mcp-proxy.js"],
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
      "args": ["exec", "-i", "mcp-memory-server", "node", "/app/mcp-proxy.js"],
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
      "args": ["exec", "-i", "mcp-memory-server", "node", "/app/mcp-proxy.js"],
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
      "args": ["exec", "-i", "mcp-memory-server", "node", "/app/mcp-proxy.js"],
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

**MCP Memory** transforms your AI assistant into a smart companion that:

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

## 🔧 Troubleshooting

### Quick Diagnostics

```bash
# Check if everything is running
docker-compose ps

# Test the server
curl http://localhost:9081/health

# Test MCP proxy
echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{}},"id":1}' | docker exec -i mcp-memory-server node /app/mcp-proxy.js

# View logs
docker logs mcp-memory-server
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

## 🏗️ Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   MCP Clients   │    │   MCP Server     │    │    Storage      │
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
- `http://localhost:9081/health` - Health check
- `http://localhost:9082` - Metrics (optional)

---

## 🎛️ Advanced Configuration

### Environment Variables

```bash
# Required
OPENAI_API_KEY=your-key-here

# Optional server configuration  
MCP_MEMORY_LOG_LEVEL=info
QDRANT_HOST=localhost
QDRANT_PORT=6334
MCP_MEMORY_DB_TYPE=sqlite
MCP_MEMORY_BACKUP_ENABLED=true

# Optional proxy configuration
MCP_SERVER_HOST=localhost
MCP_SERVER_PORT=9080
MCP_SERVER_PATH=/mcp
MCP_PROXY_DEBUG=false
```

### Production Deployment

For production use:
- [Docker Deployment Guide](docs/DEPLOYMENT.md)
- [Monitoring Setup](docs/MONITORING.md)
- [Production Config](configs/production/config.yaml)

---

## 📚 More Information

- **📖 [Full Documentation](docs/README.md)** - Complete guides and API reference  
- **🔍 [Health Monitoring](http://localhost:9081/health)** - System status
- **📊 [Metrics Dashboard](http://localhost:9082)** - Performance metrics
- **🐳 [Container Logs](./docs/DEPLOYMENT.md)** - Docker guides

## 🤝 Contributing

We welcome contributions! See [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

**🚀 Ready to give your AI assistant a perfect memory?** 

Start with the [Quick Start](#-quick-start-5-minutes) above and choose your preferred [protocol option](#-mcp-protocol-options).

**Questions?** [Open an issue](https://github.com/LerianStudio/mcp-memory/issues) or check our [documentation](docs/README.md).