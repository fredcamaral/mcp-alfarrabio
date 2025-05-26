# MCP Memory Server

A high-performance Model Context Protocol (MCP) server that provides intelligent memory and context management for ANY AI application or client. Built with Go, it features advanced vector storage, pattern recognition, and contextual learning capabilities using Chroma vector database. Works seamlessly with Claude, VS Code Copilot, Continue, Cursor, and any MCP-compatible client.

## 📢 Important: MCP SDK Now Standalone

The MCP Go implementation has been moved to a separate open-source project:
- **GoMCP SDK**: [github.com/fredcamaral/gomcp-sdk](https://github.com/fredcamaral/gomcp-sdk)

This allows the community to use the MCP SDK independently for building any MCP-compatible application.

## 🚀 Features

### Core Memory System
- **Persistent Conversation Memory**: Store and retrieve conversation history across sessions
- **Vector Similarity Search**: Find semantically similar conversations and contexts
- **Hierarchical Memory Organization**: Project-based memory isolation and organization
- **Intelligent Context Suggestions**: Proactive recommendations based on conversation patterns
- **Web UI & GraphQL API**: Modern web interface for browsing memories with GraphQL API
- **Memory Tracing**: Trace sessions and find related memories with visual timelines

### Advanced Intelligence Layer
- **Pattern Recognition**: Automatically detect conversation patterns and user preferences
- **Knowledge Graph Construction**: Build semantic relationships between entities and concepts
- **Learning & Adaptation**: Continuously improve suggestions based on user feedback
- **Multi-Repository Intelligence**: Cross-project pattern detection and insights

### Production-Ready Features
- **Multi-Level Caching**: LRU/LFU/FIFO caching strategies for optimal performance
- **Data Backup & Restore**: Automated backup with tar.gz compression and encryption
- **Security & Access Control**: Repository-level permissions and AES-GCM encryption
- **Health Monitoring**: Comprehensive health checks with Prometheus metrics
- **Docker Containerization**: Production-ready containerization with multi-stage builds

## 📋 Requirements

- Go 1.21 or higher
- Chroma vector database (required)
- PostgreSQL 13+ (optional, for metadata storage)
- Docker & Docker Compose (for containerized deployment)
- Redis (optional, for distributed caching)
- OpenAI API key (for embeddings generation)

## 🛠️ Quick Start

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/fredcamaral/mcp-memory.git
   cd mcp-memory
   ```

2. **Set up environment**
   ```bash
   cp .env.example .env
   # Edit .env to add your OPENAI_API_KEY and other configurations
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Start Chroma database**
   ```bash
   docker run -p 9000:8000 chromadb/chroma:latest run --path /data --host 0.0.0.0
   ```

5. **Run the MCP server (if using MCP tools)**
   ```bash
   go run cmd/server/main.go
   ```

6. **Run the GraphQL server and Web UI**
   ```bash
   go run cmd/graphql/main.go
   # Or use the binary:
   # ./graphql
   ```

7. **Access the Web UI**
   - Open http://localhost:8082/ in your browser
   - GraphQL playground: http://localhost:8082/graphql
   - Health check: `curl http://localhost:8081/health`

### Docker Deployment

1. **Using Docker Compose (Recommended)**
   ```bash
   cp .env.example .env
   # Edit .env to configure your environment
   docker-compose up -d
   ```

2. **Using Docker directly**
   ```bash
   docker build -t mcp-memory .
   docker run -p 8080:8080 -p 8081:8081 -p 8082:8082 \
     -e OPENAI_API_KEY=your-api-key \
     -e CHROMA_URL=http://chroma:8000 \
     mcp-memory
   ```

3. **Check deployment**
   ```bash
   curl http://localhost:8081/health
   curl http://localhost:8082/metrics
   ```

## 📚 Documentation

All documentation is organized in the `docs/` directory. See the [Documentation Index](docs/README.md) for a complete overview.

- [Development Setup](docs/DEV-HOT-RELOAD.md) - Hot reload development environment
- [Deployment Guide](docs/DEPLOYMENT.md) - Production deployment instructions
- [API Reference](docs/website/api-reference.md) - Complete API documentation
- [Development Roadmap](docs/ROADMAP.md) - Current priorities and future plans
- [Monitoring Setup](docs/MONITORING.md) - Observability and metrics configuration

## 🔧 Configuration

### Environment Variables

See `.env.example` for a complete list of configuration options. Key variables include:

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENAI_API_KEY` | (required) | OpenAI API key for embeddings |
| `CHROMA_URL` | `http://localhost:8000` | Chroma database URL |
| `MCP_MEMORY_DATA_DIR` | `./data` | Data storage directory |
| `MCP_MEMORY_LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `MCP_MEMORY_HTTP_PORT` | `8080` | Main MCP API port |
| `MCP_MEMORY_HEALTH_PORT` | `8081` | Health check port |
| `MCP_MEMORY_GRAPHQL_PORT` | `8082` | GraphQL API & Web UI port |
| `MCP_MEMORY_METRICS_PORT` | `9090` | Prometheus metrics port |
| `MCP_MEMORY_VECTOR_DIM` | `1536` | Vector dimension (OpenAI ada-002) |
| `MCP_MEMORY_ENCRYPTION_ENABLED` | `false` | Enable data encryption |
| `MCP_MEMORY_ACCESS_CONTROL_ENABLED` | `false` | Enable access control |
| `MCP_MEMORY_CACHE_ENABLED` | `true` | Enable performance caching |

### Configuration Files

- **Development**: `configs/dev/config.yaml`
- **Staging**: `configs/staging/config.yaml`
- **Production**: `configs/production/config.yaml`
- **Docker**: `configs/docker/config.yaml`

## 📊 Monitoring & Observability

### Health Checks
- **Endpoint**: `http://localhost:8081/health`
- **Liveness Probe**: Kubernetes-compatible health check
- **Readiness Probe**: Service availability check

### Metrics
- **Endpoint**: `http://localhost:8082/metrics`
- **Format**: Prometheus format
- **Dashboards**: Pre-configured Grafana dashboards included

### Logging
- **Structured Logging**: JSON format for production
- **Log Levels**: Debug, Info, Warn, Error
- **Correlation IDs**: Request tracing support

## 🔒 Security

### Encryption
- **Algorithm**: AES-GCM 256-bit encryption
- **Key Derivation**: PBKDF2 with 100,000 iterations
- **Scope**: Sensitive fields (API keys, passwords, tokens)

### Access Control
- **Repository-Level**: Isolated access per repository
- **User Authentication**: Token-based authentication
- **Permission System**: Read/Write/Admin permissions

### Rate Limiting
- **Default**: 60 requests per minute per user
- **Burst**: 10 requests burst capacity
- **Distributed**: Redis-backed rate limiting

## 🌐 GraphQL API & Web UI

### Web Interface
Access the modern web UI at `http://localhost:8082/` to:
- Browse and search memories
- View memory details and metadata
- Trace sessions with timeline visualization
- Explore related memories with relationship graphs
- Filter by repository, type, and time period

### GraphQL API
The GraphQL endpoint is available at `http://localhost:8082/graphql` with a built-in GraphiQL playground.

#### Key Queries
```graphql
# Search memories
query SearchMemories($input: MemoryQueryInput!) {
  search(input: $input) {
    chunks {
      chunk { id content summary type timestamp }
      score
    }
  }
}

# Trace a session
query TraceSession($sessionId: String!) {
  traceSession(sessionId: $sessionId) {
    id content type timestamp
  }
}

# Find related memories
query TraceRelated($chunkId: String!, $depth: Int) {
  traceRelated(chunkId: $chunkId, depth: $depth) {
    id content type timestamp
  }
}
```

#### Key Mutations
```graphql
# Store a memory
mutation StoreChunk($input: StoreChunkInput!) {
  storeChunk(input: $input) {
    id summary
  }
}
```

## 🚀 MCP Tools Reference (Legacy)

**Note**: The MCP tools are still available but the GraphQL API is now the recommended interface for most use cases.

The server implements the following MCP tools with the standardized naming convention:

### Core Memory Tools

#### `mcp__memory__store`
Store a conversation or context in memory.
```json
{
  "content": "User asked about implementing authentication",
  "metadata": {
    "type": "conversation",
    "tags": ["auth", "security"],
    "project": "my-app"
  }
}
```

#### `mcp__memory__search`
Search for similar conversations or contexts using vector similarity.
```json
{
  "query": "authentication implementation",
  "limit": 10,
  "threshold": 0.7,
  "project": "my-app"
}
```

#### `mcp__memory__list`
List all stored memories with optional filtering.
```json
{
  "project": "my-app",
  "limit": 20,
  "offset": 0
}
```

#### `mcp__memory__delete`
Delete specific memories by ID.
```json
{
  "id": "memory-id-123"
}
```

### Intelligence Tools

#### `mcp__memory__suggest_related`
Get AI-powered context suggestions based on current context.
```json
{
  "current_context": "implementing user login",
  "project": "my-app"
}
```

#### `mcp__memory__analyze_patterns`
Analyze conversation patterns and trends.
```json
{
  "project": "my-app",
  "time_range": "7d"
}
```

### Advanced Tools

#### `mcp__memory__export_project`
Export all memory for a project.
```json
{
  "project": "my-app",
  "format": "json",
  "include_vectors": false
}
```

#### `mcp__memory__import_context`
Import conversation context from external sources.
```json
{
  "source": "file",
  "data": "...",
  "project": "my-app"
}
```

#### `mcp__memory__get_stats`
Get memory usage statistics.
```json
{
  "project": "my-app"
}
```

#### `mcp__memory__update_metadata`
Update metadata for existing memories.
```json
{
  "id": "memory-id-123",
  "metadata": {
    "tags": ["updated", "important"]
  }
}
```

## 🏗️ Development

### Building from Source
```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run linting
golangci-lint run

# Build binary
go build -o bin/mcp-memory cmd/server/main.go

# Run with race detector
go run -race cmd/server/main.go
```

### Development Commands
```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Generate mocks (if using mockgen)
go generate ./...

# Run specific tests
go test -run TestFunctionName ./internal/...

# Benchmark tests
go test -bench=. ./...
```

## 🐳 Docker

### Multi-Stage Build
- **Builder Stage**: Go compilation with optimizations
- **Runtime Stage**: Alpine Linux minimal image
- **Security**: Non-root user, minimal attack surface
- **Size**: <50MB final image

### Docker Compose Services
- **mcp-memory**: Main MCP server application
- **chroma**: Vector database for embeddings storage
- **postgres**: Metadata database (optional)
- **redis**: Distributed cache (optional)  
- **prometheus**: Metrics collection
- **grafana**: Metrics visualization with pre-built dashboards
- **traefik**: Reverse proxy with automatic SSL

## 📈 Performance

### Benchmarks
- **Memory Operations**: >10,000 ops/sec
- **Vector Search**: <100ms p95 latency
- **Concurrent Users**: 1,000+ simultaneous connections
- **Memory Usage**: <500MB typical workload

### Optimization Features
- **Multi-Level Caching**: Memory, Query, and Vector caches
- **Connection Pooling**: Database connection management
- **Batch Processing**: Efficient bulk operations
- **Graceful Degradation**: Fallback strategies

## 🔄 Migration & Backup

### Automatic Backups
- **Schedule**: Configurable interval (default: 24h)
- **Retention**: Configurable retention period (default: 30 days)
- **Compression**: gzip compression to reduce storage
- **Encryption**: Optional backup encryption

### Manual Operations
```bash
# Create backup
curl -X POST http://localhost:8080/api/backup

# List backups
curl http://localhost:8080/api/backups

# Restore backup
curl -X POST http://localhost:8080/api/restore \
  -H "Content-Type: application/json" \
  -d '{"backup_id": "backup-20241201-120000"}'
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Commit Convention
- `feat:` New features
- `fix:` Bug fixes  
- `docs:` Documentation changes
- `style:` Code style changes
- `refactor:` Code refactoring
- `test:` Test changes
- `build:` Build system changes

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Anthropic](https://www.anthropic.com/) for the Model Context Protocol specification
- [Chroma](https://www.trychroma.com/) for the high-performance vector database
- [OpenAI](https://openai.com/) for embedding model APIs
- [Prometheus](https://prometheus.io/) & [Grafana](https://grafana.com/) communities for monitoring tools
- Go community for excellent libraries and tooling

## 📞 Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/fredcamaral/mcp-memory/issues)
- **Documentation**: [Full documentation](https://github.com/fredcamaral/mcp-memory/wiki)
- **Discord**: [Community support](https://discord.gg/mcp-memory)

## 🔗 Related Projects

- [MCP Specification](https://modelcontextprotocol.io/) - Official Model Context Protocol documentation
- [chroma-go](https://github.com/amikos-tech/chroma-go) - Go client for Chroma vector database
- [Claude Desktop](https://claude.ai/) - Desktop application with MCP support

---

**Made with ❤️ for the MCP ecosystem**