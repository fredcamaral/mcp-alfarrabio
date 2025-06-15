# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lerian MCP Memory Server is a high-performance Go-based Model Context Protocol (MCP) server that gives AI assistants persistent memory capabilities. It stores conversations, learns patterns, and provides intelligent context suggestions across sessions and projects. The server supports multiple transport protocols (stdio, WebSocket, SSE, HTTP) and uses Qdrant for vector storage with OpenAI embeddings.

**Key Focus**: This is a Go-only project focused on the MCP server engine. The web-ui has been removed to maintain simplicity and performance.

## Core Architecture

### Technology Stack
- **Language**: Go 1.23+ with gomcp-sdk v1.0.3 (using LerianStudio fork)
- **Vector Database**: Qdrant for embeddings and similarity search  
- **Metadata Storage**: PostgreSQL for structured data (SQLite support deprecated)
- **AI Service**: OpenAI for text embeddings
- **Transport**: Multiple protocols (stdio, WebSocket, SSE, HTTP)

### Key Components
- `internal/mcp/` - MCP protocol handlers and memory server implementation
- `internal/storage/` - Qdrant vector database and PostgreSQL adapters
- `internal/intelligence/` - Pattern recognition, learning engines, conflict detection
- `internal/embeddings/` - OpenAI integration with circuit breakers and retries
- `internal/chunking/` - Smart content chunking with context awareness
- `internal/tasks/` - Task management service and CRUD operations
- `internal/api/` - HTTP REST API handlers and router
- `pkg/types/` - Core data structures and type definitions
- `cli/` - Command-line interface (separate Go module)

### Entry Points
- `cmd/server/main.go` - Main server with stdio/HTTP mode support
- `cmd/migrate/main.go` - Database migration utility
- `cmd/openapi/main.go` - OpenAPI specification generator
- `cli/cmd/lerian-mcp-memory-cli/main.go` - CLI application

## Common Development Commands

### Environment Setup
```bash
# Initial setup with environment file
make setup

# Production mode - uses Docker Compose
make docker-up

# Development mode with hot reload
make dev-docker-up

# Regular development mode (stdio)
make dev

# HTTP mode for testing
make dev-http
```

### Building and Testing
```bash
# Build both server and CLI
make build

# Build server only
make build-server

# Build CLI only  
make build-cli

# Install CLI to PATH
make install

# Run all tests
make test

# Run tests with coverage (70% threshold)
make test-coverage

# Run integration tests
make test-integration

# Run with race detector
make test-race

# Run benchmarks
make benchmark
```

### Code Quality
```bash
# Format code and run imports
make fmt

# Run linter (golangci-lint)
make lint

# Run go vet
make vet

# Security scanning (gosec + govulncheck)
make security-scan

# Complete CI pipeline
make ci
```

### Docker Operations
```bash
# Start production services
make docker-up

# Stop all services
make docker-down

# Development mode with hot reload
make docker-dev

# View service logs
make docker-logs

# Clean Docker images
make clean
```

### Database Operations
```bash
# Build migration tool
go build -o bin/migrate cmd/migrate/main.go

# Plan migrations (dry run)
./bin/migrate -command plan

# Execute migrations
./bin/migrate -command migrate -force

# Check migration status
./bin/migrate -command status
```

### CLI Operations
```bash
# Build CLI
make build-cli

# Install CLI to PATH
make install

# Run CLI interactively
make dev-cli

# CLI usage examples
lmmc create "Task title" -t implementation -p high
lmmc list
lmmc complete <task-id>
lmmc sync
```

## Testing Individual Components
```bash
# Test specific package
go test -v ./internal/mcp/

# Test with integration tag
go test -tags=integration -v ./internal/storage/

# Test specific function
go test -v -run TestMemorySearch ./internal/mcp/

# Run comprehensive integration tests
go test -tags=integration -v ./internal/testing/

# Test with race detection (important for concurrent code)
go test -race -v ./internal/websocket/

# Run benchmarks for performance-critical components
go test -bench=. -v ./internal/embeddings/
go test -bench=. -v ./internal/storage/
```

## Environment Configuration

Copy `.env.example` to `.env` and configure. The new .env.example contains only variables actually used in the codebase, organized by functionality.

### Required Variables
- `OPENAI_API_KEY` - Your OpenAI API key for embeddings and AI features
- `AI_PROVIDER` - AI provider selection (auto-detects if not set): openai, claude, perplexity, mock

### Core Configuration Categories

**Server & Networking:**
- `MCP_MEMORY_HOST` - Internal server host (default: localhost)
- `MCP_MEMORY_PORT` - Internal server port (default: 9080)
- `MCP_HOST_PORT` - Docker host port mapping (default: 9080)

**Database & Storage:**
- `DATABASE_URL` - Full PostgreSQL connection string (takes precedence)
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` - Individual DB settings
- `MCP_MEMORY_QDRANT_HOST` - Qdrant vector database host
- `MCP_MEMORY_QDRANT_PORT` - Qdrant port (default: 6333)

**AI & Processing:**
- `OPENAI_MODEL` - OpenAI model to use (default: gpt-4o)
- `MCP_MEMORY_CHUNKING_STRATEGY` - Content chunking strategy (default: semantic)
- `MCP_MEMORY_CHUNKING_MAX_LENGTH` - Maximum chunk size (default: 8000)

**Logging & Monitoring:**
- `MCP_MEMORY_LOG_LEVEL` - Logging level (debug, info, warn, error)
- `MCP_MEMORY_LOG_FORMAT` - Log format (json, text)
- `MCP_MEMORY_LOG_FILE` - Log file path

## Code Conventions

### Error Handling
- Always return explicit errors, never panic in production code
- Use context.Context as first parameter for all operations
- Implement circuit breakers for external service calls (OpenAI, Qdrant)
- Standard error types defined in `internal/errors/standard_errors.go`

### Testing Patterns
- Use testify/assert for assertions
- Mock external dependencies (OpenAI API, Qdrant)
- Integration tests tagged with `integration`
- Benchmark tests for performance-critical code
- Comprehensive test framework in `internal/testing/test_framework.go`

### Key Implementation Patterns

**Dependency Injection**
- All services configured in `internal/di/container.go`
- Interface-based design for testability and modularity
- Circuit breaker wrappers for external dependencies

**Event-Driven Architecture**
- Memory operations emit events via `internal/events/bus.go`
- Event filtering and distribution for loose coupling
- Audit logging triggered by events

**Storage Abstraction**
- `internal/storage/interfaces.go` defines storage contracts
- Adapter pattern for different storage backends
- Retry and circuit breaker wrappers for reliability

**AI Provider Abstraction**
- Factory pattern in `pkg/ai/factory.go` with auto-detection
- Unified interface for multiple AI providers (OpenAI, Claude, Perplexity)
- Graceful fallback and circuit breaker protection

### Memory Management
- 41 MCP tools for memory operations (search, store, analyze, etc.)
- Smart chunking with configurable strategies
- Vector similarity search with confidence scoring
- Cross-repository pattern learning

### Package Structure
```
internal/
├── mcp/           # MCP protocol and memory server
├── storage/       # Database and vector storage
├── intelligence/  # Learning engines and pattern detection  
├── embeddings/    # OpenAI integration with reliability
├── chunking/      # Content processing and chunking
├── api/           # HTTP REST API handlers
├── tasks/         # Task management service
└── config/        # Configuration management

cli/
├── internal/      # CLI-specific internal packages
├── cmd/           # CLI entry point
└── pkg/           # Shared CLI packages
```

### Database Schema
- PostgreSQL for metadata (tasks, prds, sessions)
- Qdrant for vector embeddings and similarity search
- Automatic backup and persistence management
- Migration system with rollback capabilities

## Architecture Deep Dive

### MCP Memory Architecture Flow
1. **Input Processing**: Content arrives via MCP tools → `internal/mcp/server.go` → consolidated tools handlers
2. **Content Analysis**: Smart chunking (`internal/chunking/`) → embedding generation (`internal/embeddings/`)
3. **Storage Layer**: Vector data (Qdrant) + metadata (PostgreSQL) via `internal/storage/` adapters
4. **Intelligence Layer**: Pattern detection, conflict resolution, learning engines in `internal/intelligence/`
5. **Retrieval**: Context-aware search with confidence scoring and relevance ranking

### AI Provider Integration
The system uses a factory pattern in `pkg/ai/` with automatic provider detection:
- **Auto-detection Priority**: Claude → OpenAI → Perplexity → Mock
- **Fallback Strategy**: Multiple API keys can be configured; system gracefully degrades
- **Circuit Breaker Protection**: All AI providers wrapped with fault tolerance (`internal/circuitbreaker/`)

### Data Flow and State Management
- **Dependency Injection**: `internal/di/container.go` manages all service dependencies
- **Event-Driven Architecture**: Memory operations trigger events via `internal/events/`
- **Audit Trail**: All operations logged to `internal/audit_logs/` with JSONL format
- **Session Management**: Multi-session context via `internal/session/manager.go`

### MCP Transport Protocols
The server is protocol-agnostic with multiple transport options:
- **stdio + proxy** - For legacy MCP clients (Claude Desktop, VS Code) via `mcp-proxy.js`
- **WebSocket** - Real-time bidirectional communication (`internal/websocket/`)
- **SSE** - Server-sent events with HTTP fallback
- **Direct HTTP** - Simple JSON-RPC over HTTP (`internal/api/router.go`)

### Vector Operations & Search
- **Qdrant Integration**: High-performance vector search with connection pooling
- **Embedding Strategy**: OpenAI text-embedding-ada-002 (1536 dimensions) with caching
- **Chunking Intelligence**: Context-aware chunking with configurable strategies
- **Search Optimization**: Multi-stage retrieval with confidence scoring and re-ranking

### Memory Intelligence Systems
- **Pattern Recognition**: Cross-conversation and cross-repository pattern detection
- **Conflict Detection**: Automated identification of contradictory information
- **Learning Engine**: Continuous improvement of suggestions based on usage patterns
- **Knowledge Graph**: Relationship mapping between concepts and decisions
- **Freshness Management**: Time-based relevance scoring and content decay

### Performance & Reliability
- **Connection Pooling**: Database and vector store connections managed via pools
- **Circuit Breakers**: Fault tolerance for external services (OpenAI, Qdrant)
- **Caching Strategy**: Multi-layer caching for embeddings, queries, and patterns
- **Graceful Degradation**: System continues functioning with reduced capabilities during outages
- **Rate Limiting**: Configurable rate limiting for API calls and resource usage

## Deployment Endpoints

When running via docker-compose:
- `http://localhost:9080/mcp` - Main MCP JSON-RPC endpoint
- `http://localhost:9080/health` - Health check
- `ws://localhost:9080/ws` - WebSocket endpoint
- `http://localhost:9080/sse` - Server-sent events endpoint
- `http://localhost:6333` - Qdrant vector database UI
- `http://localhost:9080/api/v1/tasks` - REST API for tasks

## Troubleshooting

### Common Issues
- **Connection refused**: Run `make docker-restart`
- **OpenAI API errors**: Check API key and account credits in `.env`
- **Vector search failures**: Verify Qdrant is running (`docker logs mcp-qdrant`)
- **Memory not persisting**: Check volume mounts and permissions
- **Migration failures**: Ensure database is accessible and migrations are idempotent

### Debugging
```bash
# View logs
docker logs mcp-memory-server
make docker-logs

# Health check
curl http://localhost:9080/health

# Test MCP protocol
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | docker exec -i mcp-memory-server node /app/mcp-proxy.js

# Test REST API
curl -X GET http://localhost:9080/api/v1/tasks?repository=test
```

## Recent Project Changes

### Simplified Migration System
- Reduced from 15+ migrations to 3 essential ones
- All migrations are now idempotent and handle existing objects gracefully
- Core migrations: 000 (migration tracking), 001 (core tables), 002 (sessions)

### CLI HTTP Integration
- CLI now properly syncs with server via HTTP REST API
- Fixed type mismatches between CLI and server entities
- Improved error handling and offline mode support

### Database Changes
- Switched from SQLite to PostgreSQL as primary metadata store
- Simplified table structures for easier maintenance
- Added proper indexes for common query patterns