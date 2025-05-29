# MCP Memory Server

> **Smart memory for AI assistants** - A Model Context Protocol (MCP) server that remembers your conversations, learns from patterns, and provides intelligent context suggestions.

Perfect for **Claude Desktop**, **VS Code**, **Continue**, **Cursor**, and any MCP-compatible AI client.

## 🚀 Quick Start (5 minutes)

### Option 1: Single Docker Commands (Quick Test)

```bash
# 1. Start Chroma vector database
docker run -d --name mcp-chroma \
  -p 8000:8000 \
  -v chroma_data:/chroma/chroma \
  chromadb/chroma:latest

# 2. Start MCP Memory Server
docker run -d --name mcp-memory \
  -p 9080:9080 \
  -e OPENAI_API_KEY="your-api-key-here" \
  -e MCP_MEMORY_CHROMA_ENDPOINT="http://host.docker.internal:8000" \
  -v mcp_data:/app/data \
  --link mcp-chroma:chroma \
  ghcr.io/fredcamaral/mcp-memory:latest

# 3. Optional: Auto-updater (watches for new releases)
docker run -d --name mcp-auto-updater \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_CLEANUP=true \
  -e WATCHTOWER_POLL_INTERVAL=3600 \
  -e WATCHTOWER_SCOPE=mcp-memory \
  containrrr/watchtower:latest
```

### Option 2: Docker Compose (Recommended - Full Setup)

1. **Clone and start everything:**
   ```bash
   git clone https://github.com/fredcamaral/mcp-memory.git
   cd mcp-memory
   cp .env.example .env
   # Edit .env and add your OPENAI_API_KEY
   
   # Local development (builds from source)
   docker-compose up -d
   
   # Production with auto-updates (uses registry + Watchtower)
   docker-compose -f docker-compose.yml -f docker-compose.prod.yml --profile auto-update up -d
   ```

2. **Configure your AI client** (e.g., Claude Desktop, Claude Code, Windsurf, Cursor, etc):

   The MCP server works through a stdio <> HTTP proxy bridge written in Node.js that runs inside the container.
   
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
   
   **Alternative: SSE protocol** (Server-Sent Events for real-time communication):
   ```json
   {
     "mcpServers": {
       "memory": {
         "type": "sse",
         "url": "http://localhost:9080/sse"
       }
     }
   }
   ```

3. **Test it!** 🎉
   - Open your AI client (Claude Desktop, etc.)
   - Ask it to store a memory: *"Please remember that I prefer TypeScript over JavaScript"*
   - Later ask: *"What do you remember about my coding preferences?"*

<details>
<summary><b>📋 Full docker-compose.yml Reference</b></summary>

```yaml
services:
  # Chroma Vector Database
  chroma:
    image: chromadb/chroma:latest
    container_name: mcp-chroma
    restart: unless-stopped
    ports:
      - "8000:8000"
    volumes:
      - chroma_data:/chroma/chroma
    networks:
      - mcp_network

  # Main MCP Memory Server
  mcp-memory-server:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: mcp-memory-server
    restart: unless-stopped
    depends_on:
      - chroma
    ports:
      - "9080:9080"
      - "9081:8081"
      - "9082:8082"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - MCP_MEMORY_CHROMA_ENDPOINT=http://chroma:8000
      - MCP_MEMORY_DB_TYPE=sqlite
      - MCP_MEMORY_DB_PATH=/app/data/memory.db
    volumes:
      - mcp_data:/app/data
      - mcp_logs:/app/logs
      - mcp_backups:/app/backups
    networks:
      - mcp_network

  # Auto-updater sidecar (optional - use --profile auto-update)
  watchtower:
    image: containrrr/watchtower:latest
    container_name: mcp-auto-updater
    restart: unless-stopped
    environment:
      - WATCHTOWER_CLEANUP=true
      - WATCHTOWER_POLL_INTERVAL=3600  # Check hourly
      - WATCHTOWER_SCOPE=mcp-memory-server
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    networks:
      - mcp_network
    profiles:
      - auto-update

networks:
  mcp_network:
    driver: bridge

volumes:
  chroma_data:
    name: mcp_memory_chroma_vector_db_NEVER_DELETE
  mcp_data:
    name: mcp_memory_app_data_NEVER_DELETE
  mcp_logs:
    name: mcp_memory_logs_NEVER_DELETE
  mcp_backups:
    name: mcp_memory_backups_NEVER_DELETE
```
</details>


## 🎯 What Does This Do?

**MCP Memory** transforms your AI assistant into a smart companion that:

- **📚 Remembers Everything**: Stores all your conversations and contexts across sessions
- **🔍 Smart Search**: Finds relevant past conversations using AI-powered similarity search  
- **🧠 Pattern Learning**: Recognizes your preferences, coding patterns, and decision-making
- **💡 Proactive Suggestions**: Automatically suggests relevant context from your history
- **🔄 Cross-Project Intelligence**: Learns patterns across all your repositories and projects

## 🛠️ Configuration Files

### Claude Desktop Configuration

**Configuration:**
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

### Claude Code CLI

Add to your `.claude/mcp.json` in your project root:
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

### VS Code with Continue

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

### Cursor, Windsurf, or Other MCP Clients

For any MCP-compatible client, use:
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

### Environment Variables (.env file)

```bash
# Required
OPENAI_API_KEY=your-openai-api-key-here

# Optional (defaults work for Docker setup)
CHROMA_URL=http://chroma:8000
MCP_MEMORY_DATA_DIR=./data
MCP_MEMORY_LOG_LEVEL=info
```

## 🌟 Key Features

### Memory Tools Available to Your AI

Once configured, your AI assistant automatically gets these powerful memory abilities:

- **Store important moments**: `memory_store_chunk` - Save conversations, decisions, solutions
- **Smart search**: `memory_search` - Find similar past conversations and contexts  
- **Get context**: `memory_get_context` - Retrieve project overview and recent activity
- **Find patterns**: `memory_get_patterns` - Identify recurring themes and solutions
- **Health monitoring**: `memory_health_dashboard` - Track memory system effectiveness
- **Intelligent decay**: `memory_decay_management` - Automatically summarize and archive old memories

### Advanced Intelligence

- **🧠 Conversation Flow Detection**: Recognizes when you're debugging, implementing, or planning
- **🔗 Relationship Mapping**: Automatically links related memories and contexts
- **📊 Pattern Recognition**: Learns your coding patterns, preferences, and decision-making
- **💡 Smart Suggestions**: Proactively suggests relevant memories based on current context
- **🗂️ Multi-Repository Support**: Works across all your projects with intelligent cross-referencing

## 🔧 Troubleshooting

### Common Issues

**🔴 "Connection refused" or "Server not responding"**
```bash
# Check if containers are running
docker-compose ps

# Check logs
docker-compose logs mcp-memory-server

# Restart services
docker-compose restart
```

**🔴 "OpenAI API errors"**
- Check your API key in `.env` file
- Verify you have credits in your OpenAI account
- Check network connectivity

**🔴 "Memory not persisting"**
```bash
# Check database connection
docker-compose logs chroma

# Verify data directory permissions
ls -la ./data/
```

**🔴 "Node.js not found in container"**
The container now uses Debian slim with Node.js pre-installed. If you're using an older image:
```bash
# Pull latest image
docker-compose pull

# Rebuild from source  
docker-compose build --no-cache
```

### Checking if Everything Works

1. **Test the server directly:**
   ```bash
   # Health check
   curl http://localhost:9081/health
   
   # Test MCP proxy inside container
   echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"roots":{"listChanged":true},"sampling":{}}},"id":1}' | docker exec -i mcp-memory-server node /app/mcp-proxy.js
   ```

2. **Check container logs:**
   ```bash
   # View server logs
   docker logs mcp-memory-server
   
   # Check if containers are running
   docker-compose ps
   ```

3. **Test with your AI client:**
   - Ask it to remember something: *"Please store that I work on the mcp-memory project"*
   - Ask it to recall: *"What do you remember about my current projects?"*
   - Try: *"Use memory_health to check the memory system status"*

## 🎛️ Advanced Configuration

### Production Deployment

For production use, see the detailed configurations:
- [Production Config](configs/production/config.yaml)
- [Docker Deployment Guide](docs/DEPLOYMENT.md)
- [Monitoring Setup](docs/MONITORING.md)

### Custom Configuration

```yaml
# configs/custom/config.yaml
storage:
  chroma:
    url: "http://your-chroma-instance:8000"
    
embeddings:
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "text-embedding-ada-002"
    
security:
  encryption:
    enabled: true
  access_control:
    enabled: true
```

## 📚 More Information

- **📖 [Full Documentation](docs/README.md)** - Complete guides and API reference  
- **🔍 [Health Monitoring](http://localhost:9081/health)** - System status and metrics
- **📊 [Metrics](http://localhost:9082)** - Performance and usage metrics
- **🐳 [Container Logs](./docs/DEPLOYMENT.md)** - Docker deployment guides

## 🤝 Contributing

We welcome contributions! See [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

**🚀 Ready to give your AI assistant a perfect memory?** Follow the Quick Start above and you'll be up and running in minutes!

**Questions?** [Open an issue](https://github.com/fredcamaral/mcp-memory/issues) or check our [documentation](docs/README.md).
