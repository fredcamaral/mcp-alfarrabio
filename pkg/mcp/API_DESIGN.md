# MCP-Go API Design

## Design Principles

1. **Simplicity First**: Easy to use for common cases, powerful for advanced use
2. **Type Safety**: Leverage Go's type system to prevent errors at compile time
3. **Zero Dependencies**: No external dependencies for core functionality
4. **Extensibility**: Plugin architecture for custom transports, middleware, and handlers
5. **Performance**: Optimized for production workloads with minimal allocations

## Core API Structure

### Package Organization

```
github.com/yourusername/mcp-go/
├── mcp.go              # Main convenience functions
├── version.go          # Version information
├── api_stability.go    # API stability definitions
├── protocol/           # Protocol types and interfaces
│   ├── types.go        # Core MCP types
│   ├── errors.go       # Error definitions
│   └── validation.go   # Request/response validation
├── server/             # Server implementation
│   ├── server.go       # Server type and methods
│   ├── handlers.go     # Handler interfaces
│   └── middleware.go   # Middleware support
├── client/             # Client implementation (future)
│   └── client.go       # MCP client
├── transport/          # Transport implementations
│   ├── transport.go    # Transport interface
│   ├── stdio.go        # STDIO transport
│   ├── http.go         # HTTP transport (Phase 3)
│   └── websocket.go    # WebSocket transport (Phase 3)
└── middleware/         # Middleware implementations (Phase 3)
    ├── auth.go         # Authentication
    ├── ratelimit.go    # Rate limiting
    └── logging.go      # Request logging
```

### Core Interfaces

#### Transport Interface
```go
type Transport interface {
    // Start begins listening for messages
    Start(ctx context.Context) error
    
    // SetHandler sets the request handler
    SetHandler(handler RequestHandler)
    
    // Send sends a message
    Send(ctx context.Context, message []byte) error
    
    // Close gracefully shuts down the transport
    Close() error
}
```

#### Handler Interfaces
```go
// ToolHandler handles tool execution requests
type ToolHandler interface {
    Handle(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// ResourceHandler handles resource read requests
type ResourceHandler interface {
    Read(ctx context.Context, uri string) (*protocol.Resource, error)
    List(ctx context.Context) ([]protocol.Resource, error)
}

// PromptHandler handles prompt requests
type PromptHandler interface {
    GetPrompt(ctx context.Context, args map[string]string) (*protocol.PromptMessage, error)
}
```

#### Middleware Interface (Phase 3)
```go
type Middleware interface {
    // Wrap wraps a handler with middleware logic
    Wrap(next HandlerFunc) HandlerFunc
}

type HandlerFunc func(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error)
```

### Usage Examples

#### Basic Server
```go
package main

import (
    "context"
    "log"
    "github.com/yourusername/mcp-go"
    "github.com/yourusername/mcp-go/protocol"
)

func main() {
    // Create server
    server := mcp.NewServer("my-server", "1.0.0")
    
    // Add a tool
    tool := mcp.NewTool(
        "calculate", 
        "Performs calculations",
        mcp.ObjectSchema(map[string]mcp.SchemaProperty{
            "operation": mcp.StringParam("The operation to perform", true),
            "a": mcp.NumberParam("First number", true),
            "b": mcp.NumberParam("Second number", true),
        }),
    )
    
    server.AddTool(tool, mcp.ToolHandlerFunc(calculateHandler))
    
    // Start with STDIO transport
    transport := mcp.NewStdioTransport()
    server.SetTransport(transport)
    
    if err := server.Start(context.Background()); err != nil {
        log.Fatal(err)
    }
}

func calculateHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Implementation
    return nil, nil
}
```

#### With Middleware (Phase 3)
```go
import "github.com/yourusername/mcp-go/middleware"

// Create server with middleware
server := mcp.NewServer("my-server", "1.0.0",
    mcp.WithMiddleware(
        middleware.NewRateLimiter(100), // 100 requests per minute
        middleware.NewLogger(log.Default()),
        middleware.NewAuthenticator(authFunc),
    ),
)
```

#### HTTP Transport (Phase 3)
```go
import "github.com/yourusername/mcp-go/transport"

// Create HTTP transport
httpTransport := transport.NewHTTPTransport(
    transport.WithPort(8080),
    transport.WithTLS(certFile, keyFile),
)

server.SetTransport(httpTransport)
```

### Error Handling

All errors follow Go conventions:
- Errors are returned as the last return value
- Sentinel errors for common cases
- Error wrapping for context

```go
var (
    ErrToolNotFound = errors.New("tool not found")
    ErrInvalidParams = errors.New("invalid parameters")
    ErrUnauthorized = errors.New("unauthorized")
)

// Error wrapping example
if err := handler.Handle(ctx, params); err != nil {
    return nil, fmt.Errorf("tool execution failed: %w", err)
}
```

### Context Usage

All handler methods accept `context.Context` as the first parameter:
- Request cancellation
- Timeouts
- Request-scoped values (request ID, user info, etc.)

```go
// Extract request metadata from context
if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
    log.Printf("Processing request %s", reqID)
}
```

### Logging Interface

Pluggable logging interface:

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
}

// Set custom logger
mcp.SetLogger(myLogger)
```

### Metrics Interface (Phase 3)

```go
type MetricsCollector interface {
    RecordRequest(method string, duration time.Duration, err error)
    RecordToolExecution(tool string, duration time.Duration, err error)
}

// Set metrics collector
mcp.SetMetrics(prometheusCollector)
```

## Migration Path

### From v0.1 to v0.2
- No breaking changes for stable APIs
- New features are additive

### Future v1.0
- Stable API guarantee
- Long-term support commitment
- Performance optimizations