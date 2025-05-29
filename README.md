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
   docker-compose up -d
   
   # OR with auto-updates (pulls latest from registry every hour)
   docker-compose --profile auto-update up -d
   ```

2. **Configure your AI client** (e.g., Claude Desktop):
   
   Add this to your `claude_desktop_config.json`:
   ```json
   {
     "mcpServers": {
       "memory": {
         "type": "stdio",
         "command": "docker",
         "args": ["exec", "-i", "mcp-memory-server", "/app/mcp-memory"]
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
    image: ghcr.io/fredcamaral/mcp-memory:latest
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

### Option 3: Local Development

1. **Prerequisites:**
   - Go 1.21+
   - Docker (for Chroma database)
   - OpenAI API key

2. **Setup:**
   ```bash
   git clone https://github.com/fredcamaral/mcp-memory.git
   cd mcp-memory
   cp .env.example .env
   # Edit .env and add your OPENAI_API_KEY
   ```

3. **Start the database:**
   ```bash
   docker run -d -p 8000:8000 --name chroma chromadb/chroma:latest
   ```

4. **Run the MCP server:**
   ```bash
   go run cmd/server/main.go
   ```

5. **Configure your AI client:**
   ```json
   {
     "mcpServers": {
       "memory": {
         "type": "stdio",
         "command": "/path/to/mcp-memory/mcp-memory"
       }
     }
   }
   ```

## 🎯 What Does This Do?

**MCP Memory** transforms your AI assistant into a smart companion that:

- **📚 Remembers Everything**: Stores all your conversations and contexts across sessions
- **🔍 Smart Search**: Finds relevant past conversations using AI-powered similarity search  
- **🧠 Pattern Learning**: Recognizes your preferences, coding patterns, and decision-making
- **💡 Proactive Suggestions**: Automatically suggests relevant context from your history
- **🔄 Cross-Project Intelligence**: Learns patterns across all your repositories and projects

## 🛠️ Configuration Files

### Claude Desktop Configuration

**Location:**
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

**Configuration:**
```json
{
  "mcpServers": {
    "memory": {
      "type": "stdio",
      "command": "docker",
      "args": ["exec", "-i", "mcp-memory-server", "/app/mcp-memory"]
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
      "type": "stdio",
      "command": "docker",
      "args": ["exec", "-i", "mcp-memory-server", "/app/mcp-memory"]
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

### Checking if Everything Works

1. **Test the server directly:**
   ```bash
   curl http://localhost:8081/health
   ```

2. **Browse the web interface:**
   - Open http://localhost:8082 in your browser
   - You should see the memory management dashboard

3. **Test with your AI client:**
   - Ask it to remember something: *"Please store that I work on the mcp-memory project"*
   - Ask it to recall: *"What do you remember about my current projects?"*

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
- **🌐 [Web Interface](http://localhost:8082)** - Browse and manage memories
- **📊 [GraphQL API](http://localhost:8082/graphql)** - Playground for advanced queries
- **🔍 [Health Monitoring](http://localhost:8081/health)** - System status and metrics

## 🤝 Contributing

We welcome contributions! See [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

**🚀 Ready to give your AI assistant a perfect memory?** Follow the Quick Start above and you'll be up and running in minutes!

**Questions?** [Open an issue](https://github.com/fredcamaral/mcp-memory/issues) or check our [documentation](docs/README.md).