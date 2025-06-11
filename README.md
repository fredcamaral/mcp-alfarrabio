# Lerian MCP Memory Server

> **Give your AI assistant a perfect memory** - Works with Claude, Cursor, VS Code, and any MCP-compatible client

## 🚀 Quick Start (2 minutes)

### Prerequisites
- Docker installed and running
- OpenAI API key (for embeddings) - [Get one here](https://platform.openai.com/api-keys)

```bash
# 1. Clone and start the server
git clone https://github.com/LerianStudio/lerian-mcp-memory.git
cd lerian-mcp-memory

# If you don't have OPENAI_API_KEY in your environment:
cp .env.example .env
# Edit .env and add your OpenAI API key

make docker-up

# 2. Add to your AI client config:
```

```json
{
  "mcpServers": {
    "memory": {
      "command": "docker",
      "args": ["exec", "-i", "lerian-mcp-memory-server", "node", "/app/mcp-proxy.js"]
    }
  }
}
```

**That's it!** Your AI assistant now has persistent memory.

### Where to add the config:
- **Claude Desktop**: `~/Library/Application Support/Claude/claude_desktop_config.json` (Mac) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows)
- **VS Code/Continue**: Your Continue configuration file
- **Cursor**: Settings → MCP Servers
- **Claude Code CLI**: `.claude/mcp.json` in your project

## 🎯 What Can It Do?

Your AI assistant can now:
- **Remember conversations** across sessions
- **Search past discussions** with smart similarity matching
- **Learn your patterns** and coding preferences
- **Suggest relevant context** from previous work
- **Track decisions** and their rationale

### Example Usage:
- "Remember this solution for handling authentication"
- "What did we discuss about the database schema?"
- "Find similar problems we've solved before"
- "Save this as our coding standard for error handling"

## 🛠️ Essential Commands

```bash
# Check server health
curl http://localhost:8081/health

# View logs
make docker-logs

# Restart server
make docker-restart

# Stop server
make docker-down

# Development mode (with hot reload)
make dev-docker-up
```

## 🔧 Configuration

### Basic Setup
The server requires an OpenAI API key for embeddings:

```bash
# If not already done in quick start:
cp .env.example .env
# Edit .env to set your OPENAI_API_KEY
```

### Key Settings in `.env`:
- `OPENAI_API_KEY` - **Required** for embeddings (defaults to your global `OPENAI_API_KEY` env variable)
- `MCP_MEMORY_LOG_LEVEL` - Set to `debug` for troubleshooting
- `MCP_HOST_PORT` - Change from default 9080 if needed

## 💡 Memory Tools Available

Your AI assistant gets these memory commands:

| Tool | Purpose | Example Use |
|------|---------|-------------|
| `memory_create` | Store new information | Save code snippets, decisions, conversations |
| `memory_read` | Search and retrieve | Find past solutions, recall discussions |
| `memory_update` | Modify existing memories | Update outdated information |
| `memory_intelligence` | Get AI insights | Find patterns, get suggestions |
| `memory_tasks` | Track todos | Manage work across sessions |

<details>
<summary>📚 View All 9 Memory Tools</summary>

- `memory_create` - Store conversations and decisions
- `memory_read` - Search and retrieve context  
- `memory_update` - Update existing memories
- `memory_delete` - Remove outdated information
- `memory_analyze` - Analyze patterns across projects
- `memory_intelligence` - Get AI-powered insights
- `memory_transfer` - Export/import contexts
- `memory_tasks` - Track workflows and todos
- `memory_system` - Check system health and status
</details>

## 🚨 Troubleshooting

### Common Issues

**"Connection refused" or "MCP server not responding"**
```bash
make docker-restart
```

**"OpenAI API error"**
1. Check your API key: `grep OPENAI_API_KEY .env`
2. Ensure you have credits in your OpenAI account
3. If needed, set key explicitly in `.env`

**"Node.js not found" error**
```bash
make docker-build && make docker-restart
```

### Debug Commands
```bash
# Check what's running
docker compose ps

# Test the MCP proxy directly
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | \
  docker exec -i lerian-mcp-memory-server node /app/mcp-proxy.js

# View detailed logs
docker compose logs -f lerian-mcp-memory-server
```

## 🏗️ Architecture

<details>
<summary>View Technical Details</summary>

### Stack
- **Language**: Go 1.23+ 
- **Vector DB**: Qdrant (similarity search)
- **Metadata**: SQLite (relationships)
- **Embeddings**: OpenAI text-embedding-ada-002

### Available Endpoints
- `http://localhost:9080/mcp` - MCP JSON-RPC endpoint
- `ws://localhost:9080/ws` - WebSocket connection
- `http://localhost:9080/sse` - Server-sent events
- `http://localhost:8081/health` - Health check

### Docker Profiles
```bash
make docker-up       # Production (pre-built image)
make dev-docker-up   # Development (hot reload)
make monitoring-up   # Prometheus + Grafana
```
</details>

## 🛡️ Data & Security

- **Your data stays local** - All storage is in Docker volumes on your machine
- **Automatic backups** - Configurable via `MCP_MEMORY_BACKUP_ENABLED`
- **Encryption available** - Set `MCP_MEMORY_ENCRYPTION_ENABLED=true`
- **No telemetry** - We don't collect any usage data

### Important Volumes
```
mcp_memory_qdrant_vector_db_NEVER_DELETE  # Your embeddings
mcp_memory_app_data_NEVER_DELETE         # Your conversations
```

## 🤝 Contributing

We love contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Quick Dev Setup
```bash
make dev-docker-up    # Start dev environment
make test            # Run tests
make lint            # Check code quality
```

## 📦 Additional Tools

### CLI Tool (lmmc)
```bash
make cli-build       # Build the CLI
make cli-install     # Install to PATH
lmmc --help         # Use the CLI
```

## 📄 License

Apache 2.0 - see [LICENSE](LICENSE)

---

**Need help?** [Open an issue](https://github.com/LerianStudio/lerian-mcp-memory/issues) or check our [detailed documentation](docs/)