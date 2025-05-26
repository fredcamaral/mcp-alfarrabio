# MCP Memory Development Guide

## Overview

This guide covers development setup, architecture, and best practices for contributing to MCP Memory.

## Table of Contents

1. [Development Setup](#development-setup)
2. [Architecture Overview](#architecture-overview)
3. [Adding New Features](#adding-new-features)
4. [Testing Strategy](#testing-strategy)
5. [Performance Optimization](#performance-optimization)
6. [Debugging](#debugging)
7. [Contributing](#contributing)

## Development Setup

### Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Python 3.8+ (for ChromaDB)
- OpenAI API key

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/your-org/mcp-memory.git
   cd mcp-memory
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Start ChromaDB**
   ```bash
   docker run -p 8000:8000 chromadb/chroma
   ```

4. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your settings
   ```

5. **Run the server**
   ```bash
   go run cmd/server/main.go
   ```

### Hot Reload Setup

For development with hot reload:
```bash
# Install air
go install github.com/air-verse/air@latest

# Run with hot reload
air
```

### VS Code Setup

Recommended extensions:
- Go
- GraphQL
- REST Client
- GitLens

Launch configuration (`.vscode/launch.json`):
```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Server",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/server",
      "env": {
        "LOG_LEVEL": "debug"
      }
    }
  ]
}
```

## Architecture Overview

### Project Structure
```
mcp-memory/
├── cmd/                    # Entry points
│   ├── server/            # MCP server
│   ├── graphql/           # GraphQL server
│   └── openapi/           # OpenAPI server
├── internal/              # Private packages
│   ├── chunking/          # Text chunking
│   ├── circuitbreaker/    # Circuit breaker
│   ├── config/            # Configuration
│   ├── di/                # Dependency injection
│   ├── embeddings/        # Embedding service
│   ├── graphql/           # GraphQL schema
│   ├── intelligence/      # AI/ML components
│   ├── mcp/               # MCP protocol
│   ├── retry/             # Retry logic
│   ├── storage/           # Storage layer
│   └── workflow/          # Workflow components
├── pkg/                   # Public packages
│   ├── mcp/              # MCP client library
│   └── types/            # Shared types
├── docs/                  # Documentation
├── scripts/               # Utility scripts
└── tests/                 # Integration tests
```

### Dependency Injection

The DI container manages all dependencies:

```go
// internal/di/container.go
type Container struct {
    Config           *config.Config
    VectorStore      storage.VectorStore
    EmbeddingService embeddings.EmbeddingService
    ChunkingService  *chunking.ChunkingService
    // ... other services
}
```

### Interface Design

Key interfaces:

```go
// VectorStore interface
type VectorStore interface {
    Store(ctx context.Context, chunk types.ConversationChunk) error
    Search(ctx context.Context, query types.MemoryQuery, embeddings []float64) (*types.SearchResults, error)
    GetByID(ctx context.Context, id string) (*types.ConversationChunk, error)
    // ... other methods
}

// EmbeddingService interface
type EmbeddingService interface {
    GenerateEmbeddings(ctx context.Context, text string) ([]float64, error)
    HealthCheck(ctx context.Context) error
    GetModelInfo() ModelInfo
}
```

## Adding New Features

### 1. Define the Interface

Start with the interface definition:
```go
// internal/myfeature/interface.go
package myfeature

type MyFeature interface {
    DoSomething(ctx context.Context, input string) (string, error)
}
```

### 2. Implement the Feature

```go
// internal/myfeature/implementation.go
package myfeature

type myFeatureImpl struct {
    dependency SomeDependency
}

func New(dep SomeDependency) MyFeature {
    return &myFeatureImpl{
        dependency: dep,
    }
}

func (f *myFeatureImpl) DoSomething(ctx context.Context, input string) (string, error) {
    // Implementation
}
```

### 3. Add Tests

```go
// internal/myfeature/implementation_test.go
package myfeature

func TestDoSomething(t *testing.T) {
    // Test implementation
}
```

### 4. Wire into DI Container

```go
// internal/di/container.go
type Container struct {
    // ... existing fields
    MyFeature myfeature.MyFeature
}

// In initialization
container.MyFeature = myfeature.New(dependency)
```

### 5. Expose via API

For MCP tools:
```go
// internal/mcp/server.go
tools = append(tools, mcp.Tool{
    Name: "my_feature",
    Description: "Does something cool",
    InputSchema: myFeatureSchema,
})
```

## Testing Strategy

### Unit Tests

```go
func TestChunkingService_ProcessConversation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    []types.ConversationChunk
        wantErr bool
    }{
        {
            name:  "simple conversation",
            input: "Hello, how are you?",
            want:  []types.ConversationChunk{/* expected */},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

### Integration Tests

```go
// tests/integration/storage_test.go
func TestStorageIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup real ChromaDB
    // Test full flow
}
```

### Benchmarks

```go
func BenchmarkEmbeddingGeneration(b *testing.B) {
    service := embeddings.NewOpenAIEmbeddingService(config)
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := service.GenerateEmbeddings(ctx, "test text")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Test Coverage

```bash
# Run tests with coverage
go test -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out
```

## Performance Optimization

### 1. Connection Pooling

Enable for production:
```go
// Controlled by environment variable
CHROMA_USE_POOLING=true
CHROMA_POOL_MAX_SIZE=10
```

### 2. Batch Operations

```go
// Instead of individual operations
for _, chunk := range chunks {
    store.Store(ctx, chunk)
}

// Use batch operations
store.StoreBatch(ctx, chunks)
```

### 3. Caching Strategy

```go
// Implement caching layer
type CachedVectorStore struct {
    store storage.VectorStore
    cache cache.Cache
}

func (c *CachedVectorStore) Search(ctx context.Context, query types.MemoryQuery, embeddings []float64) (*types.SearchResults, error) {
    key := generateCacheKey(query, embeddings)
    if cached, ok := c.cache.Get(key); ok {
        return cached.(*types.SearchResults), nil
    }
    
    results, err := c.store.Search(ctx, query, embeddings)
    if err == nil {
        c.cache.Set(key, results, 5*time.Minute)
    }
    return results, err
}
```

### 4. Profiling

```go
import _ "net/http/pprof"

// In main.go
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

Profile CPU usage:
```bash
go tool pprof http://localhost:6060/debug/pprof/profile
```

## Debugging

### Enable Debug Logging

```go
// Set log level
os.Setenv("LOG_LEVEL", "debug")

// In code
log.Debug("Processing chunk", 
    "id", chunk.ID,
    "size", len(chunk.Content),
)
```

### Trace Requests

```go
// Add request tracing
func TraceMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        traceID := uuid.New().String()
        ctx := context.WithValue(r.Context(), "trace_id", traceID)
        
        log.Info("Request started",
            "trace_id", traceID,
            "method", r.Method,
            "path", r.URL.Path,
        )
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Debug ChromaDB

```python
# Check ChromaDB directly
import chromadb

client = chromadb.HttpClient(host="localhost", port=8000)
collection = client.get_collection("memories")
print(f"Collection count: {collection.count()}")

# Query directly
results = collection.query(
    query_texts=["test query"],
    n_results=10
)
```

### Common Issues

1. **"context deadline exceeded"**
   - Increase timeouts
   - Check network connectivity
   - Verify service health

2. **"rate limit exceeded"**
   - Implement exponential backoff
   - Use circuit breaker
   - Cache embeddings

3. **"vector dimension mismatch"**
   - Verify embedding model consistency
   - Check for data migration issues

## Contributing

### Code Style

Follow Go conventions:
```go
// Good
func (s *Service) ProcessChunk(ctx context.Context, chunk types.ConversationChunk) error {
    // Implementation
}

// Avoid
func (s *Service) process_chunk(ctx context.Context, c types.ConversationChunk) error {
    // Implementation
}
```

### Commit Messages

Use conventional commits:
```
feat(storage): add connection pooling support
fix(embeddings): handle rate limit errors gracefully
docs(api): update GraphQL examples
test(chunking): add edge case tests
refactor(di): simplify container initialization
```

### Pull Request Process

1. Fork the repository
2. Create feature branch: `git checkout -b feat/my-feature`
3. Make changes with tests
4. Run linters: `golangci-lint run`
5. Submit PR with description

### Code Review Checklist

- [ ] Tests pass
- [ ] Documentation updated
- [ ] No security vulnerabilities
- [ ] Performance impact considered
- [ ] Backward compatibility maintained

## Advanced Topics

### Custom Storage Backend

Implement the VectorStore interface:
```go
type MyCustomStore struct {
    // fields
}

func (s *MyCustomStore) Store(ctx context.Context, chunk types.ConversationChunk) error {
    // Custom implementation
}

// Implement all interface methods
```

### Custom Embedding Provider

```go
type MyEmbeddingService struct {
    model *MyModel
}

func (s *MyEmbeddingService) GenerateEmbeddings(ctx context.Context, text string) ([]float64, error) {
    // Use your model
    return s.model.Embed(text)
}
```

### Plugin System (Future)

```go
// Plugin interface
type Plugin interface {
    Name() string
    Initialize(container *di.Container) error
    RegisterTools() []mcp.Tool
}

// Load plugins
func LoadPlugins(dir string) ([]Plugin, error) {
    // Dynamic loading logic
}
```

## Resources

- [Go Documentation](https://golang.org/doc/)
- [MCP Protocol Spec](https://github.com/anthropics/mcp)
- [ChromaDB Documentation](https://docs.chromadb.com/)
- [OpenAI API Reference](https://platform.openai.com/docs)
- [GraphQL Best Practices](https://graphql.org/learn/best-practices/)