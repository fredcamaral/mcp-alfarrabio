# Advanced MCP-Go Guide

This guide covers advanced topics and best practices for building production-ready MCP servers with MCP-Go.

## Table of Contents

1. [Performance Optimization](#performance-optimization)
2. [Concurrency Patterns](#concurrency-patterns)
3. [Security Best Practices](#security-best-practices)
4. [Custom Transport Implementations](#custom-transport-implementations)
5. [Middleware Architecture](#middleware-architecture)
6. [State Management](#state-management)
7. [Production Deployment](#production-deployment)
8. [Monitoring and Observability](#monitoring-and-observability)
9. [Testing Strategies](#testing-strategies)
10. [Common Patterns and Anti-patterns](#common-patterns-and-anti-patterns)

## Performance Optimization

### Memory Management

Minimize allocations in hot paths:

```go
// Bad: Creates new slice on every call
func processData(data []byte) []string {
    results := []string{} // Allocation
    // Process...
    return results
}

// Good: Reuse buffers with sync.Pool
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 1024)
    },
}

func processDataOptimized(data []byte) []string {
    buf := bufferPool.Get().([]byte)
    defer func() {
        buf = buf[:0] // Reset slice
        bufferPool.Put(buf)
    }()
    
    // Use buf for processing...
    return results
}
```

### JSON Processing

Optimize JSON encoding/decoding:

```go
// Use json.RawMessage for deferred parsing
type ToolRequest struct {
    Name   string          `json:"name"`
    Params json.RawMessage `json:"params"`
}

func (s *Server) handleToolCall(req *ToolRequest) (interface{}, error) {
    tool, ok := s.tools[req.Name]
    if !ok {
        return nil, ErrToolNotFound
    }
    
    // Parse params only when needed
    params := make(map[string]interface{})
    if err := json.Unmarshal(req.Params, &params); err != nil {
        return nil, err
    }
    
    return tool.Handler.Handle(context.Background(), params)
}
```

### Concurrent Request Handling

Implement request pipelining:

```go
type RequestPipeline struct {
    workers   int
    taskQueue chan *Task
    results   chan *Result
}

func NewRequestPipeline(workers int) *RequestPipeline {
    p := &RequestPipeline{
        workers:   workers,
        taskQueue: make(chan *Task, workers*2),
        results:   make(chan *Result, workers*2),
    }
    
    // Start worker pool
    for i := 0; i < workers; i++ {
        go p.worker()
    }
    
    return p
}

func (p *RequestPipeline) worker() {
    for task := range p.taskQueue {
        result := &Result{
            ID: task.ID,
        }
        
        // Process task
        response, err := task.Handler(task.Context, task.Params)
        if err != nil {
            result.Error = err
        } else {
            result.Response = response
        }
        
        p.results <- result
    }
}
```

## Concurrency Patterns

### Tool Execution with Timeout

Implement timeouts for tool execution:

```go
func (s *Server) executeToolWithTimeout(ctx context.Context, tool *Tool, params map[string]interface{}, timeout time.Duration) (interface{}, error) {
    // Create timeout context
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    // Channel for result
    type result struct {
        data interface{}
        err  error
    }
    resultChan := make(chan result, 1)
    
    // Execute in goroutine
    go func() {
        data, err := tool.Handler.Handle(ctx, params)
        resultChan <- result{data, err}
    }()
    
    // Wait for result or timeout
    select {
    case r := <-resultChan:
        return r.data, r.err
    case <-ctx.Done():
        return nil, fmt.Errorf("tool execution timeout after %v", timeout)
    }
}
```

### Parallel Tool Execution

Execute multiple tools in parallel:

```go
func (s *Server) executeToolsParallel(ctx context.Context, requests []ToolRequest) []ToolResult {
    results := make([]ToolResult, len(requests))
    var wg sync.WaitGroup
    
    for i, req := range requests {
        wg.Add(1)
        go func(index int, request ToolRequest) {
            defer wg.Done()
            
            tool, ok := s.tools[request.Name]
            if !ok {
                results[index] = ToolResult{
                    Error: fmt.Errorf("tool not found: %s", request.Name),
                }
                return
            }
            
            data, err := tool.Handler.Handle(ctx, request.Params)
            results[index] = ToolResult{
                Data:  data,
                Error: err,
            }
        }(i, req)
    }
    
    wg.Wait()
    return results
}
```

### Rate Limiting

Implement per-client rate limiting:

```go
type RateLimiter struct {
    clients map[string]*rate.Limiter
    mu      sync.RWMutex
    limit   rate.Limit
    burst   int
}

func NewRateLimiter(rps int, burst int) *RateLimiter {
    return &RateLimiter{
        clients: make(map[string]*rate.Limiter),
        limit:   rate.Limit(rps),
        burst:   burst,
    }
}

func (rl *RateLimiter) Allow(clientID string) bool {
    rl.mu.RLock()
    limiter, exists := rl.clients[clientID]
    rl.mu.RUnlock()
    
    if !exists {
        rl.mu.Lock()
        limiter = rate.NewLimiter(rl.limit, rl.burst)
        rl.clients[clientID] = limiter
        rl.mu.Unlock()
    }
    
    return limiter.Allow()
}

// Middleware implementation
func RateLimitMiddleware(rl *RateLimiter) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, method string, params interface{}) (interface{}, error) {
            clientID := getClientID(ctx)
            if !rl.Allow(clientID) {
                return nil, mcp.NewError(
                    mcp.ErrorCodeRateLimitExceeded,
                    "rate limit exceeded",
                    map[string]interface{}{
                        "retry_after": "60s",
                    },
                )
            }
            return next.Handle(ctx, method, params)
        })
    }
}
```

## Security Best Practices

### Input Validation

Implement comprehensive input validation:

```go
type Validator struct {
    rules map[string]ValidationRule
}

type ValidationRule func(value interface{}) error

func (v *Validator) Validate(params map[string]interface{}) error {
    for key, rule := range v.rules {
        value, exists := params[key]
        if !exists {
            continue
        }
        if err := rule(value); err != nil {
            return fmt.Errorf("validation failed for %s: %w", key, err)
        }
    }
    return nil
}

// Path traversal protection
func ValidatePath(basePath string) ValidationRule {
    return func(value interface{}) error {
        path, ok := value.(string)
        if !ok {
            return fmt.Errorf("expected string, got %T", value)
        }
        
        // Clean and resolve the path
        cleanPath := filepath.Clean(path)
        absPath, err := filepath.Abs(filepath.Join(basePath, cleanPath))
        if err != nil {
            return err
        }
        
        // Ensure path is within base directory
        if !strings.HasPrefix(absPath, basePath) {
            return fmt.Errorf("path traversal detected")
        }
        
        return nil
    }
}
```

### Authentication and Authorization

Implement token-based authentication:

```go
type AuthHandler struct {
    tokenValidator TokenValidator
    permissions    map[string][]string // tool -> required permissions
}

func (a *AuthHandler) Authenticate(ctx context.Context, token string) (*User, error) {
    claims, err := a.tokenValidator.Validate(token)
    if err != nil {
        return nil, err
    }
    
    return &User{
        ID:          claims.Subject,
        Permissions: claims.Permissions,
    }, nil
}

func (a *AuthHandler) Authorize(user *User, tool string) error {
    requiredPerms, ok := a.permissions[tool]
    if !ok {
        return nil // No permissions required
    }
    
    for _, required := range requiredPerms {
        if !user.HasPermission(required) {
            return fmt.Errorf("missing permission: %s", required)
        }
    }
    
    return nil
}

// Middleware implementation
func AuthMiddleware(auth *AuthHandler) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, method string, params interface{}) (interface{}, error) {
            token := getToken(ctx)
            if token == "" {
                return nil, mcp.NewError(
                    mcp.ErrorCodeUnauthorized,
                    "authentication required",
                    nil,
                )
            }
            
            user, err := auth.Authenticate(ctx, token)
            if err != nil {
                return nil, mcp.NewError(
                    mcp.ErrorCodeUnauthorized,
                    "invalid token",
                    nil,
                )
            }
            
            // Store user in context
            ctx = context.WithValue(ctx, userKey, user)
            
            return next.Handle(ctx, method, params)
        })
    }
}
```

### Secure File Operations

Implement secure file access:

```go
type SecureFileHandler struct {
    basePath       string
    maxFileSize    int64
    allowedExts    []string
    forbiddenPaths []string
}

func (h *SecureFileHandler) ReadFile(path string) ([]byte, error) {
    // Validate path
    if err := h.validatePath(path); err != nil {
        return nil, err
    }
    
    // Check file extension
    if !h.isExtensionAllowed(path) {
        return nil, fmt.Errorf("file type not allowed")
    }
    
    // Check file size
    info, err := os.Stat(path)
    if err != nil {
        return nil, err
    }
    
    if info.Size() > h.maxFileSize {
        return nil, fmt.Errorf("file too large: %d bytes (max: %d)", info.Size(), h.maxFileSize)
    }
    
    // Read with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    return h.readWithContext(ctx, path)
}

func (h *SecureFileHandler) validatePath(path string) error {
    // Resolve to absolute path
    absPath, err := filepath.Abs(filepath.Join(h.basePath, path))
    if err != nil {
        return err
    }
    
    // Check if path is within allowed base
    if !strings.HasPrefix(absPath, h.basePath) {
        return fmt.Errorf("access denied: path outside allowed directory")
    }
    
    // Check forbidden paths
    for _, forbidden := range h.forbiddenPaths {
        if strings.Contains(absPath, forbidden) {
            return fmt.Errorf("access denied: forbidden path")
        }
    }
    
    return nil
}
```

## Custom Transport Implementations

### HTTP Transport with WebSockets

Implement a custom HTTP transport:

```go
type HTTPTransport struct {
    server     *http.Server
    upgrader   websocket.Upgrader
    handler    MessageHandler
    sessions   map[string]*Session
    sessionsMu sync.RWMutex
}

func NewHTTPTransport(addr string) *HTTPTransport {
    t := &HTTPTransport{
        upgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool {
                // Implement origin validation
                return true
            },
        },
        sessions: make(map[string]*Session),
    }
    
    mux := http.NewServeMux()
    mux.HandleFunc("/mcp", t.handleHTTP)
    mux.HandleFunc("/mcp/ws", t.handleWebSocket)
    
    t.server = &http.Server{
        Addr:    addr,
        Handler: mux,
    }
    
    return t
}

func (t *HTTPTransport) handleHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Parse JSON-RPC request
    var req jsonrpc.Request
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Process request
    response := t.handler.Handle(r.Context(), &req)
    
    // Send response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (t *HTTPTransport) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := t.upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    
    session := &Session{
        ID:   generateSessionID(),
        Conn: conn,
    }
    
    t.registerSession(session)
    defer t.unregisterSession(session.ID)
    
    // Handle WebSocket messages
    for {
        var req jsonrpc.Request
        if err := conn.ReadJSON(&req); err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket error: %v", err)
            }
            break
        }
        
        // Process request asynchronously
        go func() {
            response := t.handler.Handle(r.Context(), &req)
            if err := conn.WriteJSON(response); err != nil {
                log.Printf("Failed to send response: %v", err)
            }
        }()
    }
}
```

### gRPC Transport

Implement a gRPC transport:

```go
type GRPCTransport struct {
    server  *grpc.Server
    handler MessageHandler
}

func NewGRPCTransport() *GRPCTransport {
    return &GRPCTransport{
        server: grpc.NewServer(
            grpc.UnaryInterceptor(grpcMiddleware),
            grpc.StreamInterceptor(grpcStreamMiddleware),
        ),
    }
}

// Implement the gRPC service
type mcpService struct {
    handler MessageHandler
}

func (s *mcpService) Execute(ctx context.Context, req *pb.MCPRequest) (*pb.MCPResponse, error) {
    // Convert protobuf to JSON-RPC
    jsonReq := &jsonrpc.Request{
        ID:      req.Id,
        Method:  req.Method,
        Params:  req.Params,
        JSONRPC: "2.0",
    }
    
    // Handle request
    response := s.handler.Handle(ctx, jsonReq)
    
    // Convert response to protobuf
    return &pb.MCPResponse{
        Id:     response.ID,
        Result: response.Result,
        Error:  convertError(response.Error),
    }, nil
}
```

## Middleware Architecture

### Building a Middleware Chain

Create composable middleware:

```go
type MiddlewareChain struct {
    middlewares []Middleware
}

func (c *MiddlewareChain) Then(handler Handler) Handler {
    // Build chain in reverse order
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        handler = c.middlewares[i](handler)
    }
    return handler
}

// Usage
chain := &MiddlewareChain{
    middlewares: []Middleware{
        LoggingMiddleware(),
        MetricsMiddleware(),
        AuthMiddleware(authHandler),
        RateLimitMiddleware(limiter),
        ValidationMiddleware(validator),
    },
}

finalHandler := chain.Then(toolHandler)
```

### Context-Aware Middleware

Pass data through middleware:

```go
type contextKey string

const (
    requestIDKey contextKey = "requestID"
    userKey      contextKey = "user"
    startTimeKey contextKey = "startTime"
)

func RequestIDMiddleware() Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, method string, params interface{}) (interface{}, error) {
            requestID := generateRequestID()
            ctx = context.WithValue(ctx, requestIDKey, requestID)
            
            // Add to response headers
            if transport, ok := TransportFromContext(ctx); ok {
                transport.SetHeader("X-Request-ID", requestID)
            }
            
            return next.Handle(ctx, method, params)
        })
    }
}

func TimingMiddleware() Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, method string, params interface{}) (interface{}, error) {
            start := time.Now()
            ctx = context.WithValue(ctx, startTimeKey, start)
            
            result, err := next.Handle(ctx, method, params)
            
            duration := time.Since(start)
            if logger, ok := LoggerFromContext(ctx); ok {
                logger.Info("Request completed",
                    "method", method,
                    "duration", duration,
                    "error", err != nil,
                )
            }
            
            return result, err
        })
    }
}
```

## State Management

### Session Management

Implement stateful sessions:

```go
type SessionManager struct {
    sessions map[string]*Session
    mu       sync.RWMutex
    ttl      time.Duration
}

type Session struct {
    ID        string
    UserID    string
    Data      map[string]interface{}
    ExpiresAt time.Time
    mu        sync.RWMutex
}

func (sm *SessionManager) Create(userID string) *Session {
    session := &Session{
        ID:        generateSessionID(),
        UserID:    userID,
        Data:      make(map[string]interface{}),
        ExpiresAt: time.Now().Add(sm.ttl),
    }
    
    sm.mu.Lock()
    sm.sessions[session.ID] = session
    sm.mu.Unlock()
    
    return session
}

func (sm *SessionManager) Get(id string) (*Session, error) {
    sm.mu.RLock()
    session, exists := sm.sessions[id]
    sm.mu.RUnlock()
    
    if !exists {
        return nil, ErrSessionNotFound
    }
    
    if time.Now().After(session.ExpiresAt) {
        sm.Delete(id)
        return nil, ErrSessionExpired
    }
    
    return session, nil
}

// Session-aware tool handler
func (s *Session) Get(key string) (interface{}, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    value, ok := s.Data[key]
    return value, ok
}

func (s *Session) Set(key string, value interface{}) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.Data[key] = value
}
```

### Distributed State

Implement distributed state with Redis:

```go
type RedisStateStore struct {
    client *redis.Client
    prefix string
}

func (r *RedisStateStore) Get(ctx context.Context, key string) (interface{}, error) {
    data, err := r.client.Get(ctx, r.prefix+key).Bytes()
    if err == redis.Nil {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, err
    }
    
    var value interface{}
    if err := json.Unmarshal(data, &value); err != nil {
        return nil, err
    }
    
    return value, nil
}

func (r *RedisStateStore) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    
    return r.client.Set(ctx, r.prefix+key, data, ttl).Err()
}

// Distributed lock for coordination
func (r *RedisStateStore) Lock(ctx context.Context, key string, ttl time.Duration) (*redislock.Lock, error) {
    return redislock.Obtain(ctx, r.client, r.prefix+"lock:"+key, ttl, &redislock.Options{
        RetryStrategy: redislock.LinearBackoff(100 * time.Millisecond),
    })
}
```

## Production Deployment

### Configuration Management

Implement environment-based configuration:

```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Security SecurityConfig `yaml:"security"`
    Limits   LimitsConfig   `yaml:"limits"`
    Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
    Address         string        `yaml:"address" env:"MCP_SERVER_ADDRESS" default:":8080"`
    ReadTimeout     time.Duration `yaml:"read_timeout" env:"MCP_READ_TIMEOUT" default:"30s"`
    WriteTimeout    time.Duration `yaml:"write_timeout" env:"MCP_WRITE_TIMEOUT" default:"30s"`
    ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"MCP_SHUTDOWN_TIMEOUT" default:"10s"`
}

func LoadConfig(path string) (*Config, error) {
    config := &Config{}
    
    // Load from file if provided
    if path != "" {
        data, err := os.ReadFile(path)
        if err != nil {
            return nil, err
        }
        if err := yaml.Unmarshal(data, config); err != nil {
            return nil, err
        }
    }
    
    // Override with environment variables
    if err := env.Parse(config); err != nil {
        return nil, err
    }
    
    // Validate configuration
    if err := config.Validate(); err != nil {
        return nil, err
    }
    
    return config, nil
}
```

### Graceful Shutdown

Implement graceful shutdown:

```go
type Server struct {
    // ... other fields
    shutdown chan struct{}
    wg       sync.WaitGroup
}

func (s *Server) Start(ctx context.Context) error {
    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    // Start server components
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        s.runTransport(ctx)
    }()
    
    // Wait for shutdown signal
    select {
    case <-ctx.Done():
        log.Println("Context cancelled, shutting down...")
    case sig := <-sigChan:
        log.Printf("Received signal %v, shutting down...", sig)
    }
    
    // Initiate graceful shutdown
    return s.Shutdown(context.Background())
}

func (s *Server) Shutdown(ctx context.Context) error {
    log.Println("Starting graceful shutdown...")
    
    // Stop accepting new requests
    close(s.shutdown)
    
    // Create shutdown context with timeout
    shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
    defer cancel()
    
    // Wait for ongoing requests to complete
    done := make(chan struct{})
    go func() {
        s.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        log.Println("Graceful shutdown completed")
        return nil
    case <-shutdownCtx.Done():
        log.Println("Shutdown timeout exceeded, forcing shutdown")
        return shutdownCtx.Err()
    }
}
```

### Health Checks

Implement comprehensive health checks:

```go
type HealthChecker struct {
    checks map[string]HealthCheck
    mu     sync.RWMutex
}

type HealthCheck func(ctx context.Context) error

type HealthStatus struct {
    Status    string                 `json:"status"`
    Checks    map[string]CheckResult `json:"checks"`
    Timestamp time.Time              `json:"timestamp"`
}

type CheckResult struct {
    Status string `json:"status"`
    Error  string `json:"error,omitempty"`
}

func (h *HealthChecker) Check(ctx context.Context) *HealthStatus {
    status := &HealthStatus{
        Checks:    make(map[string]CheckResult),
        Timestamp: time.Now(),
    }
    
    h.mu.RLock()
    checks := make(map[string]HealthCheck, len(h.checks))
    for name, check := range h.checks {
        checks[name] = check
    }
    h.mu.RUnlock()
    
    // Run checks in parallel
    var wg sync.WaitGroup
    var mu sync.Mutex
    allHealthy := true
    
    for name, check := range checks {
        wg.Add(1)
        go func(n string, c HealthCheck) {
            defer wg.Done()
            
            checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
            defer cancel()
            
            err := c(checkCtx)
            
            mu.Lock()
            if err != nil {
                status.Checks[n] = CheckResult{
                    Status: "unhealthy",
                    Error:  err.Error(),
                }
                allHealthy = false
            } else {
                status.Checks[n] = CheckResult{
                    Status: "healthy",
                }
            }
            mu.Unlock()
        }(name, check)
    }
    
    wg.Wait()
    
    if allHealthy {
        status.Status = "healthy"
    } else {
        status.Status = "unhealthy"
    }
    
    return status
}

// Example checks
func DatabaseHealthCheck(db *sql.DB) HealthCheck {
    return func(ctx context.Context) error {
        return db.PingContext(ctx)
    }
}

func RedisHealthCheck(client *redis.Client) HealthCheck {
    return func(ctx context.Context) error {
        return client.Ping(ctx).Err()
    }
}
```

## Monitoring and Observability

### Metrics Collection

Implement Prometheus metrics:

```go
type Metrics struct {
    RequestsTotal   *prometheus.CounterVec
    RequestDuration *prometheus.HistogramVec
    ActiveRequests  prometheus.Gauge
    ErrorsTotal     *prometheus.CounterVec
}

func NewMetrics() *Metrics {
    return &Metrics{
        RequestsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "mcp_requests_total",
                Help: "Total number of MCP requests",
            },
            []string{"method", "status"},
        ),
        RequestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "mcp_request_duration_seconds",
                Help:    "Request duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method"},
        ),
        ActiveRequests: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "mcp_active_requests",
                Help: "Number of active requests",
            },
        ),
        ErrorsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "mcp_errors_total",
                Help: "Total number of errors",
            },
            []string{"method", "error_type"},
        ),
    }
}

func MetricsMiddleware(metrics *Metrics) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, method string, params interface{}) (interface{}, error) {
            start := time.Now()
            metrics.ActiveRequests.Inc()
            defer metrics.ActiveRequests.Dec()
            
            result, err := next.Handle(ctx, method, params)
            
            duration := time.Since(start)
            status := "success"
            if err != nil {
                status = "error"
                errorType := "unknown"
                if mcpErr, ok := err.(*mcp.Error); ok {
                    errorType = fmt.Sprintf("%d", mcpErr.Code)
                }
                metrics.ErrorsTotal.WithLabelValues(method, errorType).Inc()
            }
            
            metrics.RequestsTotal.WithLabelValues(method, status).Inc()
            metrics.RequestDuration.WithLabelValues(method).Observe(duration.Seconds())
            
            return result, err
        })
    }
}
```

### Distributed Tracing

Implement OpenTelemetry tracing:

```go
func TracingMiddleware(tracer trace.Tracer) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, method string, params interface{}) (interface{}, error) {
            // Start span
            ctx, span := tracer.Start(ctx, fmt.Sprintf("mcp.%s", method),
                trace.WithAttributes(
                    attribute.String("mcp.method", method),
                    attribute.String("mcp.version", "2.0"),
                ),
            )
            defer span.End()
            
            // Add request ID to span
            if requestID := ctx.Value(requestIDKey); requestID != nil {
                span.SetAttributes(attribute.String("request.id", requestID.(string)))
            }
            
            // Execute handler
            result, err := next.Handle(ctx, method, params)
            
            // Record error if any
            if err != nil {
                span.RecordError(err)
                span.SetStatus(codes.Error, err.Error())
            } else {
                span.SetStatus(codes.Ok, "")
            }
            
            return result, err
        })
    }
}

// Tool-specific tracing
func (s *Server) executeToolTraced(ctx context.Context, name string, params map[string]interface{}) (interface{}, error) {
    ctx, span := s.tracer.Start(ctx, fmt.Sprintf("tool.%s", name),
        trace.WithAttributes(
            attribute.String("tool.name", name),
        ),
    )
    defer span.End()
    
    // Add parameter info (be careful with sensitive data)
    span.SetAttributes(attribute.Int("tool.params.count", len(params)))
    
    return s.executeTool(ctx, name, params)
}
```

### Structured Logging

Implement context-aware structured logging:

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    With(fields ...Field) Logger
}

type Field struct {
    Key   string
    Value interface{}
}

func LoggingMiddleware(logger Logger) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, method string, params interface{}) (interface{}, error) {
            // Create request-scoped logger
            requestLogger := logger.With(
                Field{"method", method},
                Field{"request_id", ctx.Value(requestIDKey)},
            )
            
            // Add logger to context
            ctx = context.WithValue(ctx, loggerKey, requestLogger)
            
            requestLogger.Info("Request started")
            
            result, err := next.Handle(ctx, method, params)
            
            if err != nil {
                requestLogger.Error("Request failed",
                    Field{"error", err.Error()},
                )
            } else {
                requestLogger.Info("Request completed")
            }
            
            return result, err
        })
    }
}

// Use logger from context in handlers
func someToolHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    logger := LoggerFromContext(ctx)
    logger.Debug("Processing tool request", Field{"params", params})
    
    // ... tool logic ...
    
    return result, nil
}
```

## Testing Strategies

### Table-Driven Tests

Write comprehensive table-driven tests:

```go
func TestToolExecution(t *testing.T) {
    tests := []struct {
        name      string
        tool      string
        params    map[string]interface{}
        setup     func(*Server)
        want      interface{}
        wantErr   bool
        errorCode int
    }{
        {
            name: "successful execution",
            tool: "echo",
            params: map[string]interface{}{
                "message": "hello",
            },
            want: map[string]interface{}{
                "message": "hello",
            },
        },
        {
            name: "missing required parameter",
            tool: "echo",
            params: map[string]interface{}{},
            wantErr: true,
            errorCode: mcp.ErrorCodeInvalidParams,
        },
        {
            name: "tool not found",
            tool: "nonexistent",
            params: map[string]interface{}{},
            wantErr: true,
            errorCode: mcp.ErrorCodeMethodNotFound,
        },
        {
            name: "tool execution error",
            tool: "failing_tool",
            params: map[string]interface{}{},
            setup: func(s *Server) {
                s.AddTool(mcp.NewTool("failing_tool", "", nil),
                    mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
                        return nil, errors.New("tool failed")
                    }),
                )
            },
            wantErr: true,
            errorCode: mcp.ErrorCodeInternalError,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            server := NewTestServer()
            if tt.setup != nil {
                tt.setup(server)
            }
            
            got, err := server.ExecuteTool(context.Background(), tt.tool, tt.params)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("ExecuteTool() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if tt.wantErr && err != nil {
                mcpErr, ok := err.(*mcp.Error)
                if !ok {
                    t.Errorf("Expected mcp.Error, got %T", err)
                    return
                }
                if mcpErr.Code != tt.errorCode {
                    t.Errorf("Error code = %d, want %d", mcpErr.Code, tt.errorCode)
                }
                return
            }
            
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ExecuteTool() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Testing

Test full MCP flows:

```go
func TestMCPIntegration(t *testing.T) {
    // Create test server
    server := NewTestServer()
    
    // Create test transport
    transport := NewTestTransport()
    server.SetTransport(transport)
    
    // Start server
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go server.Start(ctx)
    
    // Test initialize
    t.Run("initialize", func(t *testing.T) {
        req := &jsonrpc.Request{
            ID:      "1",
            Method:  "initialize",
            Params:  map[string]interface{}{
                "protocolVersion": "2024-11-05",
                "capabilities": map[string]interface{}{},
            },
            JSONRPC: "2.0",
        }
        
        resp := transport.SendRequest(req)
        assert.NotNil(t, resp.Result)
        assert.Nil(t, resp.Error)
        
        result := resp.Result.(map[string]interface{})
        assert.Equal(t, "2024-11-05", result["protocolVersion"])
    })
    
    // Test tool listing
    t.Run("tools/list", func(t *testing.T) {
        req := &jsonrpc.Request{
            ID:      "2",
            Method:  "tools/list",
            JSONRPC: "2.0",
        }
        
        resp := transport.SendRequest(req)
        assert.NotNil(t, resp.Result)
        
        result := resp.Result.(map[string]interface{})
        tools := result["tools"].([]interface{})
        assert.Greater(t, len(tools), 0)
    })
    
    // Test tool execution
    t.Run("tools/call", func(t *testing.T) {
        req := &jsonrpc.Request{
            ID:     "3",
            Method: "tools/call",
            Params: map[string]interface{}{
                "name": "echo",
                "arguments": map[string]interface{}{
                    "message": "test",
                },
            },
            JSONRPC: "2.0",
        }
        
        resp := transport.SendRequest(req)
        assert.NotNil(t, resp.Result)
        
        result := resp.Result.(map[string]interface{})
        content := result["content"].([]interface{})
        assert.Greater(t, len(content), 0)
    })
}
```

### Benchmarking

Benchmark critical paths:

```go
func BenchmarkToolExecution(b *testing.B) {
    server := NewTestServer()
    ctx := context.Background()
    params := map[string]interface{}{
        "message": "benchmark test",
    }
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := server.ExecuteTool(ctx, "echo", params)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

func BenchmarkJSONParsing(b *testing.B) {
    data := []byte(`{"jsonrpc":"2.0","id":"123","method":"tools/call","params":{"name":"test","arguments":{"key":"value"}}}`)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        var req jsonrpc.Request
        if err := json.Unmarshal(data, &req); err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkConcurrentRequests(b *testing.B) {
    server := NewTestServer()
    transport := NewTestTransport()
    server.SetTransport(transport)
    
    ctx := context.Background()
    go server.Start(ctx)
    
    requests := make([]*jsonrpc.Request, 100)
    for i := range requests {
        requests[i] = &jsonrpc.Request{
            ID:     fmt.Sprintf("%d", i),
            Method: "tools/call",
            Params: map[string]interface{}{
                "name": "echo",
                "arguments": map[string]interface{}{
                    "message": "test",
                },
            },
            JSONRPC: "2.0",
        }
    }
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        i := 0
        for pb.Next() {
            req := requests[i%len(requests)]
            resp := transport.SendRequest(req)
            if resp.Error != nil {
                b.Fatal(resp.Error)
            }
            i++
        }
    })
}
```

## Common Patterns and Anti-patterns

### Patterns

#### Resource Pooling

```go
// Good: Reuse expensive resources
type ResourcePool struct {
    pool chan *Resource
    new  func() (*Resource, error)
}

func NewResourcePool(size int, new func() (*Resource, error)) *ResourcePool {
    pool := &ResourcePool{
        pool: make(chan *Resource, size),
        new:  new,
    }
    
    // Pre-populate pool
    for i := 0; i < size; i++ {
        resource, err := new()
        if err != nil {
            continue
        }
        pool.pool <- resource
    }
    
    return pool
}

func (p *ResourcePool) Get(ctx context.Context) (*Resource, error) {
    select {
    case resource := <-p.pool:
        return resource, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        return p.new()
    }
}

func (p *ResourcePool) Put(resource *Resource) {
    select {
    case p.pool <- resource:
    default:
        // Pool is full, discard resource
        resource.Close()
    }
}
```

#### Circuit Breaker

```go
// Good: Protect against cascading failures
type CircuitBreaker struct {
    maxFailures  int
    resetTimeout time.Duration
    failures     int
    lastFailTime time.Time
    mu           sync.Mutex
    state        int // 0=closed, 1=open, 2=half-open
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    
    // Check if circuit should be reset
    if cb.state == 1 && time.Since(cb.lastFailTime) > cb.resetTimeout {
        cb.state = 2 // half-open
        cb.failures = 0
    }
    
    if cb.state == 1 {
        cb.mu.Unlock()
        return ErrCircuitOpen
    }
    
    cb.mu.Unlock()
    
    err := fn()
    
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    if err != nil {
        cb.failures++
        cb.lastFailTime = time.Now()
        
        if cb.failures >= cb.maxFailures {
            cb.state = 1 // open
        }
        return err
    }
    
    // Success
    if cb.state == 2 {
        cb.state = 0 // closed
    }
    cb.failures = 0
    return nil
}
```

### Anti-patterns

#### Blocking in Handlers

```go
// Bad: Blocking operations in handler
func badHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // This blocks the entire handler
    time.Sleep(5 * time.Second)
    
    // Synchronous HTTP call without timeout
    resp, err := http.Get("https://slow-api.example.com")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    return processResponse(resp)
}

// Good: Non-blocking with timeouts
func goodHandler(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    // Use context for cancellation
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    // Async operation
    resultChan := make(chan interface{}, 1)
    errChan := make(chan error, 1)
    
    go func() {
        // HTTP call with timeout
        req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com", nil)
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
            errChan <- err
            return
        }
        defer resp.Body.Close()
        
        result, err := processResponse(resp)
        if err != nil {
            errChan <- err
        } else {
            resultChan <- result
        }
    }()
    
    select {
    case result := <-resultChan:
        return result, nil
    case err := <-errChan:
        return nil, err
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

#### Memory Leaks

```go
// Bad: Goroutine leak
func badServer() {
    for {
        conn, _ := listener.Accept()
        go handleConnection(conn) // Never terminates
    }
}

// Good: Proper goroutine lifecycle management
func goodServer(ctx context.Context) {
    var wg sync.WaitGroup
    
    for {
        select {
        case <-ctx.Done():
            wg.Wait() // Wait for all handlers to complete
            return
        default:
            conn, err := listener.Accept()
            if err != nil {
                continue
            }
            
            wg.Add(1)
            go func() {
                defer wg.Done()
                handleConnectionWithContext(ctx, conn)
            }()
        }
    }
}
```

## Conclusion

This advanced guide has covered the essential patterns and practices for building production-ready MCP servers with MCP-Go. Key takeaways:

1. **Performance**: Use pooling, minimize allocations, and optimize hot paths
2. **Concurrency**: Leverage Go's concurrency primitives safely and effectively
3. **Security**: Validate inputs, implement proper authentication, and follow security best practices
4. **Reliability**: Implement circuit breakers, health checks, and graceful shutdown
5. **Observability**: Add comprehensive logging, metrics, and tracing
6. **Testing**: Write thorough tests including unit, integration, and benchmarks

Remember that building robust MCP servers is an iterative process. Start with the basics, measure performance, and optimize based on real-world usage patterns.

For more examples and the latest updates, visit our [GitHub repository](https://github.com/yourusername/mcp-go).

Happy building! 🚀