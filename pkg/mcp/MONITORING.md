# MCP-Go Monitoring & Observability Guide

This guide covers the monitoring and observability features available in the MCP-Go library, including metrics, tracing, logging, and health checks.

## Table of Contents

- [Overview](#overview)
- [Metrics with Prometheus](#metrics-with-prometheus)
- [Distributed Tracing with OpenTelemetry](#distributed-tracing-with-opentelemetry)
- [Structured Logging](#structured-logging)
- [Health Checks](#health-checks)
- [Complete Example](#complete-example)
- [Best Practices](#best-practices)

## Overview

The MCP-Go library provides comprehensive monitoring and observability features:

- **Metrics**: Prometheus-compatible metrics for request latency, error rates, throughput, and resource usage
- **Tracing**: OpenTelemetry integration for distributed tracing across the full request lifecycle
- **Logging**: Structured logging with trace correlation
- **Health Checks**: Liveness, readiness, and detailed health endpoints

## Metrics with Prometheus

### Setup

```go
import (
    "github.com/fredcamaral/mcp-memory/pkg/mcp/metrics"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// Create metrics instance
mcpMetrics := metrics.NewMetrics("mcp", "server")

// Create metrics middleware
metricsMiddleware := metrics.NewMiddleware(mcpMetrics)

// Start uptime counter
metricsMiddleware.StartUptimeCounter(context.Background())

// Expose metrics endpoint
http.Handle("/metrics", promhttp.Handler())
```

### Available Metrics

#### Request Metrics
- `mcp_server_request_duration_seconds`: Histogram of request durations
- `mcp_server_request_total`: Counter of total requests
- `mcp_server_active_requests`: Gauge of currently active requests

#### Error Metrics
- `mcp_server_errors_total`: Counter of errors by type

#### Tool Execution Metrics
- `mcp_server_tool_execution_duration_seconds`: Histogram of tool execution times
- `mcp_server_tool_execution_total`: Counter of tool executions

#### Resource Operation Metrics
- `mcp_server_resource_operation_duration_seconds`: Histogram of resource operation times
- `mcp_server_resource_operation_total`: Counter of resource operations

#### WebSocket Metrics
- `mcp_server_websocket_connections`: Gauge of active WebSocket connections
- `mcp_server_websocket_messages_sent_total`: Counter of sent messages
- `mcp_server_websocket_messages_received_total`: Counter of received messages

### Using Metrics Middleware

```go
// Track MCP request
err := metricsMiddleware.TrackRequest("tools/call", func() error {
    // Your request handling logic
    return nil
})

// Track tool execution
err := metricsMiddleware.TrackToolExecution("memory_search", func() error {
    // Tool execution logic
    return nil
})

// Track resource operation
err := metricsMiddleware.TrackResourceOperation("read", "file", func() error {
    // Resource operation logic
    return nil
})

// Track WebSocket connections
metricsMiddleware.IncrementWebSocketConnection()
defer metricsMiddleware.DecrementWebSocketConnection()
```

### Prometheus Configuration

Example `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'mcp-server'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

## Distributed Tracing with OpenTelemetry

### Setup

```go
import (
    "github.com/fredcamaral/mcp-memory/pkg/mcp/tracing"
)

// Configure tracing
config := tracing.Config{
    ServiceName:    "mcp-server",
    ServiceVersion: "1.0.0",
    Environment:    "production",
    Endpoint:       "localhost:4317",
    UseHTTP:        false, // Use gRPC
    Insecure:       true,  // For development
}

// Initialize tracer
tracer, shutdown, err := tracing.NewTracer(context.Background(), config)
if err != nil {
    log.Fatal(err)
}
defer shutdown(context.Background())
```

### Creating Spans

```go
// Trace an MCP request
ctx, span := tracer.TraceRequest(ctx, "tools/call",
    attribute.String("tool.name", "memory_search"),
    attribute.String("client.id", clientID),
)
defer span.End()

// Trace tool execution
ctx, span := tracer.TraceToolExecution(ctx, "memory_search",
    attribute.String("query", query),
)
defer span.End()

// Trace with automatic error handling
err := tracer.WithRequestSpan(ctx, "tools/call", func(ctx context.Context) error {
    // Your logic here
    return nil
})
```

### Adding Context to Spans

```go
// Add events
tracing.AddEvent(ctx, "Processing started",
    attribute.Int("batch.size", 100),
)

// Set attributes
tracing.SetAttributes(ctx,
    attribute.String("user.id", userID),
    attribute.Bool("cache.hit", true),
)

// Record errors
if err != nil {
    tracing.RecordError(ctx, err)
    tracing.SetStatus(ctx, codes.Error, err.Error())
}
```

### Trace Propagation

```go
// Extract trace context from headers
carrier := propagation.HeaderCarrier(r.Header)
ctx = tracing.Extract(ctx, carrier)

// Inject trace context into outgoing requests
carrier := propagation.HeaderCarrier(req.Header)
tracing.Inject(ctx, carrier)
```

## Structured Logging

### Setup

```go
import (
    "github.com/fredcamaral/mcp-memory/pkg/mcp/logging"
)

// Configure logger
config := logging.Config{
    Level:      logging.LevelInfo,
    Format:     "json",
    AddSource:  true,
    TimeFormat: time.RFC3339,
}

// Create logger
logger := logging.NewLogger(config)

// Set as default
logging.SetDefault(logger)
```

### Logging with Context

```go
// Create logger with trace context
logger := logger.WithContext(ctx)

// Log with additional fields
logger.WithFields(map[string]interface{}{
    "user_id": userID,
    "tool":    toolName,
}).Info("Processing request")

// Log with error
logger.WithError(err).Error("Failed to process request")
```

### MCP-Specific Logging

```go
// Log MCP request
logger.Request(ctx, "tools/call", requestID, params)

// Log MCP response
logger.Response(ctx, "tools/call", requestID, result, duration)

// Log tool execution
logger.ToolExecution(ctx, "memory_search", args)
logger.ToolResult(ctx, "memory_search", result, duration)
logger.ToolError(ctx, "memory_search", err, duration)

// Log resource operations
logger.ResourceOperation(ctx, "read", "file", "/path/to/file")

// Log WebSocket events
logger.WebSocketConnection("connected", remoteAddr)
```

### Using Logging Middleware

```go
loggingMiddleware := logging.NewMiddleware(logger)

// Log requests with panic recovery
result, err := loggingMiddleware.LogRequest(ctx, "tools/call", requestID, func() (interface{}, error) {
    // Your logic here
    return result, nil
})

// Log tool execution with panic recovery
result, err := loggingMiddleware.LogTool(ctx, "memory_search", args, func() (interface{}, error) {
    // Tool logic here
    return result, nil
})
```

## Health Checks

### Setup

```go
import (
    "github.com/fredcamaral/mcp-memory/pkg/mcp/health"
)

// Create health checker
healthChecker := health.NewHealthChecker(10 * time.Second)

// Register health checks
healthChecker.RegisterCheck("database", health.CheckDatabase(db))
healthChecker.RegisterCheck("disk_space", health.CheckDiskSpace("/", 1<<30)) // 1GB minimum
healthChecker.RegisterCheck("memory", health.CheckMemory(90.0)) // 90% max usage
healthChecker.RegisterCheck("external_api", health.CheckHTTPEndpoint("API", "https://api.example.com/health", 5*time.Second))
```

### Custom Health Checks

```go
// Register custom health check
healthChecker.RegisterCheck("vector_store", func(ctx context.Context) *health.Result {
    // Check vector store connectivity
    err := vectorStore.Ping(ctx)
    if err != nil {
        return &health.Result{
            Status:  health.StatusUnhealthy,
            Message: "Vector store unreachable",
            Details: map[string]interface{}{
                "error": err.Error(),
            },
        }
    }
    
    return &health.Result{
        Status:  health.StatusHealthy,
        Message: "Vector store is healthy",
    }
})
```

### HTTP Endpoints

```go
// Register health endpoints
http.HandleFunc("/health/live", healthChecker.HTTPHandlerLive())
http.HandleFunc("/health/ready", healthChecker.HTTPHandlerReady())
http.HandleFunc("/health", healthChecker.HTTPHandlerHealth())
```

### Health Check Responses

#### Liveness Check (`/health/live`)
```json
{
    "status": "alive",
    "timestamp": "2024-01-15T10:30:00Z"
}
```

#### Readiness Check (`/health/ready`)
```json
{
    "status": "healthy",
    "timestamp": "2024-01-15T10:30:00Z",
    "checks": {
        "database": {
            "status": "healthy",
            "message": "Database is responsive",
            "details": {
                "duration": 15
            },
            "last_checked": "2024-01-15T10:30:00Z",
            "duration_ms": 15
        }
    }
}
```

#### Detailed Health Check (`/health`)
```json
{
    "status": "healthy",
    "timestamp": "2024-01-15T10:30:00Z",
    "system": {
        "version": "1.0.0",
        "go_version": "1.21",
        "uptime": "24h30m15s",
        "hostname": "mcp-server-1"
    },
    "checks": {
        "database": {
            "status": "healthy",
            "message": "Database is responsive",
            "details": {
                "duration": 15
            },
            "last_checked": "2024-01-15T10:30:00Z",
            "duration_ms": 15
        },
        "disk_space": {
            "status": "healthy",
            "message": "Disk space adequate",
            "details": {
                "path": "/",
                "total_bytes": 1000000000000,
                "used_bytes": 600000000000,
                "free_bytes": 400000000000,
                "free_percent": "40.00%"
            },
            "last_checked": "2024-01-15T10:30:00Z",
            "duration_ms": 5
        }
    }
}
```

## Complete Example

Here's a complete example integrating all monitoring features:

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"
    
    "github.com/fredcamaral/mcp-memory/pkg/mcp"
    "github.com/fredcamaral/mcp-memory/pkg/mcp/health"
    "github.com/fredcamaral/mcp-memory/pkg/mcp/logging"
    "github.com/fredcamaral/mcp-memory/pkg/mcp/metrics"
    "github.com/fredcamaral/mcp-memory/pkg/mcp/server"
    "github.com/fredcamaral/mcp-memory/pkg/mcp/tracing"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    ctx := context.Background()
    
    // Initialize logging
    logger := logging.NewLogger(logging.Config{
        Level:  logging.LevelInfo,
        Format: "json",
    })
    logging.SetDefault(logger)
    
    // Initialize metrics
    mcpMetrics := metrics.NewMetrics("mcp", "server")
    metricsMiddleware := metrics.NewMiddleware(mcpMetrics)
    metricsMiddleware.StartUptimeCounter(ctx)
    
    // Initialize tracing
    tracer, shutdown, err := tracing.NewTracer(ctx, tracing.Config{
        ServiceName:    "mcp-server",
        ServiceVersion: "1.0.0",
        Environment:    "production",
        Endpoint:       "localhost:4317",
        Insecure:       true,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer shutdown(ctx)
    
    // Initialize health checks
    healthChecker := health.NewHealthChecker(10 * time.Second)
    
    // Create MCP server with monitoring
    mcpServer := server.New(
        server.WithName("example-server"),
        server.WithVersion("1.0.0"),
        server.WithMiddleware(func(next server.Handler) server.Handler {
            return func(ctx context.Context, req *mcp.Request) (*mcp.Response, error) {
                // Add tracing
                ctx, span := tracer.TraceRequest(ctx, req.Method)
                defer span.End()
                
                // Add metrics
                var resp *mcp.Response
                err := metricsMiddleware.TrackRequest(req.Method, func() error {
                    var err error
                    resp, err = next(ctx, req)
                    return err
                })
                
                return resp, err
            }
        }),
    )
    
    // Register health check for MCP server
    healthChecker.RegisterCheck("mcp_server", func(ctx context.Context) *health.Result {
        // Check if server is accepting requests
        return &health.Result{
            Status:  health.StatusHealthy,
            Message: "MCP server is operational",
        }
    })
    
    // Setup HTTP server for monitoring endpoints
    mux := http.NewServeMux()
    mux.Handle("/metrics", promhttp.Handler())
    mux.HandleFunc("/health/live", healthChecker.HTTPHandlerLive())
    mux.HandleFunc("/health/ready", healthChecker.HTTPHandlerReady())
    mux.HandleFunc("/health", healthChecker.HTTPHandlerHealth())
    
    // Start monitoring server
    go func() {
        logger.Info("Starting monitoring server on :8080")
        if err := http.ListenAndServe(":8080", mux); err != nil {
            logger.WithError(err).Error("Monitoring server failed")
        }
    }()
    
    // Start MCP server
    logger.Info("Starting MCP server")
    if err := mcpServer.Start(ctx); err != nil {
        logger.WithError(err).Fatal("Failed to start MCP server")
    }
}
```

## Best Practices

### Metrics
1. **Use appropriate buckets**: Configure histogram buckets based on your SLOs
2. **Label cardinality**: Keep label cardinality low to avoid memory issues
3. **Metric naming**: Follow Prometheus naming conventions
4. **Alert on SLIs**: Focus alerts on Service Level Indicators

### Tracing
1. **Sampling**: Use sampling in production to reduce overhead
2. **Context propagation**: Always propagate trace context through your application
3. **Span attributes**: Add meaningful attributes but avoid sensitive data
4. **Error recording**: Always record errors in spans

### Logging
1. **Log levels**: Use appropriate log levels (debug, info, warn, error)
2. **Structured fields**: Use consistent field names across your application
3. **Correlation**: Always include trace IDs in logs when available
4. **Performance**: Be mindful of logging frequency in hot paths

### Health Checks
1. **Dependency checks**: Include all critical dependencies
2. **Timeouts**: Set appropriate timeouts for health checks
3. **Graceful degradation**: Use "degraded" status for non-critical issues
4. **Caching**: Cache health check results if checks are expensive

### Dashboard Examples

#### Grafana Dashboard Queries

Request rate:
```promql
rate(mcp_server_request_total[5m])
```

Error rate:
```promql
rate(mcp_server_errors_total[5m]) / rate(mcp_server_request_total[5m])
```

P95 latency:
```promql
histogram_quantile(0.95, rate(mcp_server_request_duration_seconds_bucket[5m]))
```

Active connections:
```promql
mcp_server_websocket_connections
```

Tool execution time by tool:
```promql
histogram_quantile(0.95, rate(mcp_server_tool_execution_duration_seconds_bucket[5m])) by (tool_name)
```

### Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: mcp_server
    rules:
      - alert: HighErrorRate
        expr: rate(mcp_server_errors_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors per second"
      
      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(mcp_server_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High request latency"
          description: "P95 latency is {{ $value }} seconds"
      
      - alert: ServerUnhealthy
        expr: up{job="mcp-server"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "MCP server is down"
          description: "MCP server has been down for more than 1 minute"
```

## Conclusion

The MCP-Go library provides comprehensive monitoring and observability features that enable you to:

- Track performance metrics and SLIs
- Trace requests across distributed systems
- Correlate logs with traces
- Monitor service health
- Set up effective alerting

By following this guide and best practices, you can ensure your MCP server is production-ready with full observability.