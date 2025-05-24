# Claude Vector Memory MCP Server

An intelligent conversation memory server for Claude MCP with advanced vector storage, pattern recognition, and contextual learning capabilities.

## 🚀 Features

### Core Memory System
- **Persistent Conversation Memory**: Store and retrieve conversation history across sessions
- **Vector Similarity Search**: Find semantically similar conversations and contexts
- **Hierarchical Memory Organization**: Project-based memory isolation and organization
- **Intelligent Context Suggestions**: Proactive recommendations based on conversation patterns

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
- SQLite 3.35+ (default) or PostgreSQL 13+
- Docker & Docker Compose (for containerized deployment)
- Redis (optional, for distributed caching)

## 🛠️ Quick Start

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/fredcamaral/mcp-memory.git
   cd mcp-memory
   ```

2. **Install dependencies**
   ```bash
   make deps
   ```

3. **Run in development mode**
   ```bash
   make dev
   ```

4. **Test the installation**
   ```bash
   curl http://localhost:8081/health
   ```

### Docker Deployment

1. **Using Docker Compose (Recommended)**
   ```bash
   docker-compose up -d
   ```

2. **Using Docker directly**
   ```bash
   make docker-build
   make docker-run
   ```

3. **Check deployment**
   ```bash
   curl http://localhost:8081/health
   curl http://localhost:8082/metrics
   ```

## 🔧 Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_MEMORY_DATA_DIR` | `./data` | Data storage directory |
| `MCP_MEMORY_CONFIG_DIR` | `./configs` | Configuration directory |
| `MCP_MEMORY_LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) |
| `MCP_MEMORY_HTTP_PORT` | `8080` | Main API port |
| `MCP_MEMORY_HEALTH_PORT` | `8081` | Health check port |
| `MCP_MEMORY_METRICS_PORT` | `8082` | Metrics port |
| `MCP_MEMORY_VECTOR_DIM` | `1536` | Vector dimension (must match embedding model) |
| `MCP_MEMORY_ENCRYPTION_ENABLED` | `false` | Enable data encryption |
| `MCP_MEMORY_ACCESS_CONTROL_ENABLED` | `false` | Enable access control |

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

## 🚀 API Reference

### Core MCP Tools

#### `memory_store`
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

#### `memory_search`
Search for similar conversations or contexts.
```json
{
  "query": "authentication implementation",
  "limit": 10,
  "threshold": 0.7,
  "project": "my-app"
}
```

#### `memory_suggest_related`
Get AI-powered context suggestions.
```json
{
  "current_context": "implementing user login",
  "project": "my-app"
}
```

### Advanced Tools

#### `memory_export_project`
Export all memory for a project.
```json
{
  "project": "my-app",
  "format": "json",
  "include_vectors": false
}
```

#### `memory_import_context`
Import conversation context from external source.
```json
{
  "source": "file",
  "data": "...",
  "project": "my-app"
}
```

## 🏗️ Development

### Building from Source
```bash
# Install dependencies
make deps

# Run tests
make test

# Run linting
make lint

# Build binary
make build

# Cross-compile for all platforms
make cross-compile

# Create release package
make release
```

### Testing
```bash
# Unit tests
make test-unit

# Integration tests  
make test-integration

# End-to-end tests
make test-e2e

# Coverage report
make test-coverage

# Benchmarks
make benchmark
```

### Database Operations
```bash
# Run migrations
make migrate

# Create backup
make backup

# Restore from backup
make restore FILE=backup.tar.gz

# Health check
make health-check
```

## 🐳 Docker

### Multi-Stage Build
- **Builder Stage**: Go compilation with optimizations
- **Runtime Stage**: Alpine Linux minimal image
- **Security**: Non-root user, minimal attack surface
- **Size**: <50MB final image

### Docker Compose Services
- **mcp-memory-server**: Main application
- **postgres**: Database (optional)
- **redis**: Cache (optional)  
- **prometheus**: Metrics collection
- **grafana**: Metrics visualization
- **traefik**: Reverse proxy

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

- HashiCorp for containerization best practices
- OpenAI for embedding model compatibility
- Prometheus & Grafana communities for monitoring tools
- FAISS for high-performance vector search

## 📞 Support

- **GitHub Issues**: [Report bugs and request features](https://github.com/fredcamaral/mcp-memory/issues)
- **Documentation**: [Full documentation](https://github.com/fredcamaral/mcp-memory/wiki)
- **Discord**: [Community support](https://discord.gg/mcp-memory)

---

**Made with ❤️ for the Claude MCP ecosystem**