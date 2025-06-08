# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lerian MCP Memory Server is a high-performance Go-based Model Context Protocol (MCP) server that gives AI assistants persistent memory capabilities. It stores conversations, learns patterns, and provides intelligent context suggestions across sessions and projects. The server supports multiple transport protocols (stdio, WebSocket, SSE, HTTP) and uses Qdrant for vector storage with OpenAI embeddings.

**Key Focus**: This is a Go-only project focused on the MCP server engine. The web-ui has been removed to maintain simplicity and performance.

## Core Architecture

### Technology Stack
- **Language**: Go 1.23+ with gomcp-sdk v1.0.3
- **Vector Database**: Qdrant for embeddings and similarity search  
- **Metadata Storage**: SQLite for structured data
- **AI Service**: OpenAI for text embeddings
- **Transport**: Multiple protocols (stdio, WebSocket, SSE, HTTP)

### Key Components
- `internal/mcp/` - MCP protocol handlers and memory server implementation
- `internal/storage/` - Qdrant vector database and SQLite adapters
- `internal/intelligence/` - Pattern recognition, learning engines, conflict detection
- `internal/embeddings/` - OpenAI integration with circuit breakers and retries
- `internal/chunking/` - Smart content chunking with context awareness
- `pkg/types/` - Core data structures and type definitions

### Entry Points
- `cmd/server/main.go` - Main server with stdio/HTTP mode support
- `cmd/migrate/main.go` - Database migration utility
- `cmd/openapi/main.go` - OpenAPI specification generator

## Common Development Commands

### Environment Setup
```bash
# Initial setup with environment file
make setup-env

# Production mode - uses pre-built image from GHCR
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
# Build binary
make build

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
# Start with docker-compose
make docker-compose-up

# Stop services
make docker-compose-down

# Development mode with hot reload
make dev-up
make dev-logs
make dev-restart
```

### Testing Individual Components
```bash
# Test specific package
go test -v ./internal/mcp/

# Test with integration tag
go test -tags=integration -v ./internal/storage/

# Test specific function
go test -v -run TestMemorySearch ./internal/mcp/
```

## Environment Configuration

Copy `.env.example` to `.env` and configure:

### Required
- `OPENAI_API_KEY` - Your OpenAI API key for embeddings

### Key Environment Variables
- `MCP_HOST_PORT` - Main MCP server port (default: 9080)
- `QDRANT_HOST_PORT` - Qdrant vector DB port (default: 6333)
- `MCP_MEMORY_LOG_LEVEL` - Logging level (debug, info, warn, error)
- `MCP_MEMORY_VECTOR_DIM` - Embedding dimension (default: 1536 for ada-002)

## Code Conventions

### Error Handling
- Always return explicit errors, never panic in production code
- Use context.Context as first parameter for all operations
- Implement circuit breakers for external service calls (OpenAI, Qdrant)

### Testing Patterns
- Use testify/assert for assertions
- Mock external dependencies (OpenAI API, Qdrant)
- Integration tests tagged with `integration`
- Benchmark tests for performance-critical code

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
└── config/        # Configuration management
```

### Database Schema
- SQLite for metadata (chunks, relationships, sessions)
- Qdrant for vector embeddings and similarity search
- Automatic backup and persistence management

## Important Implementation Notes

### MCP Transport Support
The server supports multiple transport protocols:
- **stdio + proxy** - For legacy MCP clients (Claude Desktop, VS Code)
- **WebSocket** - Real-time bidirectional communication
- **SSE** - Server-sent events with HTTP fallback  
- **Direct HTTP** - Simple JSON-RPC over HTTP

### Vector Operations
- Uses Qdrant for high-performance vector search
- OpenAI text-embedding-ada-002 by default (1536 dimensions)
- Circuit breakers and retries for reliability
- Intelligent chunking to maximize embedding effectiveness

### Memory Intelligence
- Pattern recognition across conversations and repositories
- Conflict detection for contradictory decisions
- Learning engine that improves suggestions over time
- Cross-repository knowledge sharing

### Performance Considerations
- Connection pooling for Qdrant
- Embedding caching to reduce OpenAI API calls
- Query optimization with performance monitoring
- Graceful degradation under high load

## Deployment Endpoints

When running via docker-compose:
- `http://localhost:9080/mcp` - Main MCP JSON-RPC endpoint
- `http://localhost:9080/health` - Health check
- `ws://localhost:9080/ws` - WebSocket endpoint
- `http://localhost:9080/sse` - Server-sent events endpoint
- `http://localhost:6333` - Qdrant vector database UI

## Troubleshooting

### Common Issues
- **Connection refused**: Run `make docker-compose-restart`
- **OpenAI API errors**: Check API key and account credits in `.env`
- **Vector search failures**: Verify Qdrant is running (`docker logs mcp-qdrant`)
- **Memory not persisting**: Check volume mounts and permissions

### Debugging
```bash
# View logs
docker logs mcp-memory-server
make dev-logs

# Health check
curl http://localhost:9080/health

# Test MCP protocol
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | docker exec -i mcp-memory-server node /app/mcp-proxy.js
```