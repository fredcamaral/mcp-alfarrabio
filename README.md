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
The server works with multiple AI providers and auto-detects your configuration:

```bash
# If not already done in quick start:
cp .env.example .env
# Edit .env to set your API keys
```

### AI Provider Support
The server **automatically detects** which AI provider to use based on your API keys:
- **OpenAI** - For embeddings and AI features (most common)
- **Claude/Anthropic** - Advanced reasoning and analysis
- **Perplexity** - Web search and research capabilities
- **Mock** - Testing and development (no API key needed)

**Detection Priority**: Claude → OpenAI → Perplexity → Mock

### Key Settings in `.env`:
- `OPENAI_API_KEY` - **Required** for embeddings (defaults to your global env variable)
- `AI_PROVIDER` - Optional: `openai`, `claude`, `perplexity`, or `mock` (auto-detects if not set)
- `CLAUDE_API_KEY` - For Claude/Anthropic features
- `PERPLEXITY_API_KEY` - For Perplexity features
- `MCP_MEMORY_LOG_LEVEL` - Set to `debug` for troubleshooting
- `MCP_HOST_PORT` - Change from default 9080 if needed

### Environment Validation
```bash
# Validate your configuration
./scripts/validate-env.sh

# This checks:
# ✅ Required API keys are set
# ✅ Database configuration is valid
# ✅ Docker setup is working
# ✅ No unused variables
```

## 💡 Memory Tools Available

Your AI assistant gets **41 powerful memory tools** organized in categories:

| Category | Key Tools | Example Use |
|----------|-----------|-------------|
| **Core Memory** | `memory_store_chunk`, `memory_search`, `memory_get_context` | Store conversations, find past solutions, get relevant context |
| **Intelligence** | `memory_analyze_patterns`, `memory_detect_conflicts`, `memory_suggest_related` | Find patterns, detect contradictions, get smart suggestions |
| **Tasks & Workflow** | `memory_create_tasks`, `memory_track_progress`, `memory_suggest_actions` | Manage todos, track work, get action suggestions |
| **Cross-Repository** | `memory_search_multi_repo`, `memory_analyze_cross_repo_patterns` | Share knowledge across projects |
| **System & Health** | `memory_system_status`, `memory_export_project`, `memory_backup_create` | Monitor health, backup data, export/import |

<details>
<summary>📚 View All Memory Categories & Tools</summary>

**🧠 Core Memory Operations (8 tools)**
- `memory_store_chunk` - Store conversations and decisions
- `memory_search` - Search with smart similarity matching
- `memory_get_context` - Get relevant context for current work
- `memory_store_decision` - Save important decisions with rationale
- `memory_find_similar` - Find related past discussions
- `memory_update_chunk` - Update existing memories
- `memory_delete_chunk` - Remove outdated information
- `memory_get_chunk` - Retrieve specific memory by ID

**🔍 Intelligence & Analysis (12 tools)**
- `memory_analyze_patterns` - Discover patterns across projects
- `memory_detect_conflicts` - Find contradictory information
- `memory_suggest_related` - Get smart content suggestions
- `memory_learning_insights` - AI-powered insights from your data
- `memory_get_patterns` - Retrieve discovered patterns
- `memory_create_thread` - Start conversation threads
- `memory_get_threads` - List active conversation threads
- `memory_detect_threads` - Auto-detect conversation flows
- `memory_conflicts` - Get conflict reports
- `memory_continuity` - Maintain context across sessions
- `memory_quality_check` - Validate memory quality
- `memory_pattern_evolution` - Track how patterns change over time

**📋 Tasks & Workflow (8 tools)**
- `memory_create_tasks` - Generate tasks from conversations
- `memory_track_progress` - Monitor task completion
- `memory_suggest_actions` - Get action recommendations
- `memory_workflow_analyze` - Analyze workflow patterns
- `memory_todo_extract` - Extract todos from text
- `memory_task_dependencies` - Understand task relationships
- `memory_progress_report` - Generate progress summaries
- `memory_workflow_optimize` - Optimize workflow efficiency

**🌐 Multi-Repository (5 tools)**
- `memory_search_multi_repo` - Search across all repositories
- `memory_analyze_cross_repo_patterns` - Find patterns across projects
- `memory_repo_compare` - Compare repository patterns
- `memory_knowledge_transfer` - Transfer knowledge between repos
- `memory_repo_insights` - Get repository-specific insights

**⚙️ System & Management (8 tools)**
- `memory_system_status` - Check system health and metrics
- `memory_export_project` - Export project memory
- `memory_import_context` - Import memory from other projects
- `memory_backup_create` - Create memory backups
- `memory_backup_restore` - Restore from backups
- `memory_cleanup_old` - Clean up old memories
- `memory_optimize_storage` - Optimize storage performance
- `memory_diagnostic_report` - Get detailed system diagnostics
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

# Validate your environment
./scripts/validate-env.sh

# Test the MCP proxy directly
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | \
  docker exec -i lerian-mcp-memory-server node /app/mcp-proxy.js

# View detailed logs
docker compose logs -f lerian-mcp-memory-server

# Check which AI provider is being used
docker exec lerian-mcp-memory-server env | grep -E "(AI_PROVIDER|OPENAI_API_KEY|CLAUDE_API_KEY)"
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
```

### Key Features
- **AI Provider Auto-Detection** - Works with OpenAI, Claude, or Perplexity
- **Smart Chunking** - Context-aware content processing
- **Pattern Recognition** - Learns from your conversations
- **Conflict Detection** - Identifies contradictory information
- **Cross-Repository Learning** - Shares knowledge across projects
- **Circuit Breakers** - Fault tolerance for external services
- **WebSocket Support** - Real-time bidirectional communication
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

# CLI provides direct access to memory operations
lmmc search "authentication patterns"
lmmc store "Database migration strategy"
lmmc analyze patterns
```

### Environment Configuration Guide
The `.env.example` file has been completely cleaned and now contains **only variables actually used** in the codebase:

```bash
# Check your environment configuration
./scripts/validate-env.sh

# See what variables are available
head -50 .env.example

# Get migration guide for environment changes
cat docs/ENV-MIGRATION-GUIDE.md
```

## 📄 License

Apache 2.0 - see [LICENSE](LICENSE)

---

## 🎯 What's New

### Recent Improvements
- ✅ **AI Provider Auto-Detection** - Automatically detects OpenAI, Claude, or Perplexity API keys
- ✅ **Cleaned Environment Configuration** - Removed 45+ unused variables, added 26 missing ones
- ✅ **Enhanced Memory Tools** - 41 powerful tools across 5 categories
- ✅ **Environment Validation** - `./scripts/validate-env.sh` checks your configuration
- ✅ **Improved Documentation** - Comprehensive guides in `docs/` folder
- ✅ **Better CLI Integration** - Enhanced `lmmc` tool with direct memory access

### For Developers
- **No Breaking Changes** - All existing configurations continue to work
- **100% Variable Accuracy** - All documented environment variables are actually used
- **Enhanced Testing** - Comprehensive validation and debugging tools
- **Better Architecture** - Clean separation of concerns and fault tolerance

**Need help?** [Open an issue](https://github.com/LerianStudio/lerian-mcp-memory/issues) or check our [detailed documentation](docs/)