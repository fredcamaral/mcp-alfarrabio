# Lerian MCP Memory Server

> **Smart memory for AI assistants** - A high-performance Go-based Model Context Protocol (MCP) server that remembers your conversations, learns from patterns, and provides intelligent context suggestions.

Perfect for **Claude Desktop**, **VS Code**, **Continue**, **Cursor**, and any MCP-compatible AI client.

## 🚀 Quick Start (5 minutes)

### Step 1: Start the Server

```bash
git clone https://github.com/LerianStudio/lerian-mcp-memory.git
cd lerian-mcp-memory
cp .env.example .env
# Edit .env if you need to change OPENAI_API_KEY (defaults to your global env)

# Start everything (uses pre-built image from GitHub Container Registry)
docker-compose up -d
```

> **Note**: The default `docker-compose.yml` uses the pre-built image from `ghcr.io/lerianstudio/lerian-mcp-memory:latest` and includes Watchtower for automatic updates. For development with hot reload, use `make dev-docker-up` instead.

### Step 2: Choose Your Connection Method

The MCP Memory Server supports **multiple transport protocols** for maximum compatibility:

| Protocol | Use Case |
|----------|----------|
| **📡 stdio + proxy** | MCP clients (Claude Desktop, VS Code, Cursor) |
| **🔌 WebSocket** | Real-time bidirectional communication |
| **📤 SSE (Server-Sent Events)** | Event streaming + direct HTTP |
| **🌐 Direct HTTP** | Simple request/response |

---

## 🔌 MCP Protocol Options

### Option 1: stdio + proxy (Recommended for Most Clients)

**Best for:** Claude Desktop, Claude Code CLI, VS Code with Continue, Cursor

The server includes a Node.js proxy that bridges stdio ↔ HTTP for MCP client compatibility.

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

Use the same configuration as Claude Desktop above. For clients that support SSE or WebSocket:

**SSE Configuration:**
```json
{
  "mcpServers": {
    "memory": {
      "transport": "sse",
      "url": "http://localhost:9080/sse"
    }
  }
}
```

**WebSocket Configuration:**
```json
{
  "mcpServers": {
    "memory": {
      "transport": "websocket",
      "url": "ws://localhost:9080/ws"
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

Your AI assistant gets 9 powerful memory tools:

- `memory_create` - Store conversations and decisions
- `memory_read` - Search and retrieve context  
- `memory_update` - Update existing memories
- `memory_delete` - Remove outdated information
- `memory_intelligence` - Get AI-powered insights
- `memory_transfer` - Export/import contexts
- `memory_tasks` - Track workflows and todos
- `memory_analyze` - Analyze patterns across projects
- `memory_system` - System health and status

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

**🔴 Disable automatic updates**
- Set `WATCHTOWER_LABEL_ENABLE=false` in `.env`
- Or stop Watchtower: `docker-compose stop watchtower`

---

## 🎛️ Advanced Configuration

### Environment Variables

```bash
# Required (defaults to your global OPENAI_API_KEY if set)
OPENAI_API_KEY=${OPENAI_API_KEY:-your-key-here}

# Common optional settings
MCP_MEMORY_LOG_LEVEL=info          # debug, info, warn, error
MCP_HOST_PORT=9080                 # MCP API port
QDRANT_HOST_PORT=6333              # Qdrant port
MCP_MEMORY_BACKUP_ENABLED=true     # Enable automatic backups
MCP_MEMORY_BACKUP_INTERVAL_HOURS=24 # Backup frequency
```

See `.env.example` for all available configuration options.

The `.env` file is the single source of truth - all settings are automatically passed to containers.

### Development Mode

```bash
# Start development environment with hot reload
make dev-docker-up

# View logs
make dev-docker-logs

# Rebuild after major changes
make dev-docker-rebuild

# Access container shell
make dev-docker-shell

# Stop development environment
make dev-docker-down
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
- **stdio + proxy**: Works with MCP clients
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
