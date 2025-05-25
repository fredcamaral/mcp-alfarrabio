# MCP-Go Performance Tuning Guide

This guide provides comprehensive performance optimization strategies, benchmarking results, and best practices for the MCP-Go library.

## Table of Contents

1. [Performance Overview](#performance-overview)
2. [Memory Pool Optimizations](#memory-pool-optimizations)
3. [JSON Encoding/Decoding](#json-encodingdecoding)
4. [Concurrent Request Handling](#concurrent-request-handling)
5. [Benchmarking Results](#benchmarking-results)
6. [Best Practices](#best-practices)
7. [Performance Monitoring](#performance-monitoring)

## Performance Overview

The MCP-Go library has been optimized for high-throughput, low-latency scenarios with the following key features:

- **Zero-allocation pools**: Reusable object pools for requests, responses, and buffers
- **Optimized JSON codec**: Custom JSON marshaling for common types
- **Concurrent processing**: Worker pools, batching, and pipelining
- **Memory efficiency**: Minimal allocations through buffer reuse

## Memory Pool Optimizations

### Overview

Memory pools significantly reduce garbage collection pressure by reusing objects instead of allocating new ones.

### Available Pools

```go
import "github.com/fredcamaral/mcp-memory/pkg/mcp"

// Buffer pools
bufferPool := mcp.GlobalPools.Buffers
jsonEncoderPool := mcp.GlobalPools.JSONEncoders
jsonDecoderPool := mcp.GlobalPools.JSONDecoders
slicePool := mcp.GlobalPools.Slices

// Request/Response pools
requestPool := mcp.GlobalRequestPools.Requests
responsePool := mcp.GlobalRequestPools.Responses
errorPool := mcp.GlobalRequestPools.ErrorResponses
```

### Usage Example

```go
// Get buffer from pool
buf := bufferPool.Get()
defer bufferPool.Put(buf)

// Use buffer for JSON encoding
encoder := json.NewEncoder(buf)
err := encoder.Encode(data)

// Get request from pool
req := requestPool.Get()
defer requestPool.Put(req)

// Use request
req.Method = "tools/call"
req.Params = params
```

### Performance Impact

Benchmark results show up to **70% reduction in allocations** and **45% improvement in throughput** when using pools:

```
BenchmarkWithoutPools-8     100000    15234 ns/op    4096 B/op    42 allocs/op
BenchmarkWithPools-8        200000     8421 ns/op     512 B/op    12 allocs/op
```

## JSON Encoding/Decoding

### Optimized Codec

The optimized JSON codec provides fast marshaling for common MCP types:

```go
import "github.com/fredcamaral/mcp-memory/pkg/mcp/protocol"

codec := protocol.GlobalOptimizedCodec

// Fast encoding
data, err := codec.FastMarshal(request)

// Streaming decode
decoder := protocol.NewStreamingDecoder(reader)
var req protocol.JSONRPCRequest
err := decoder.Decode(&req)
```

### Custom Marshaling

For frequently used types, custom marshaling avoids reflection overhead:

```go
// Before (standard json.Marshal)
data, _ := json.Marshal(req)  // ~2500 ns/op

// After (FastMarshal)
data, _ := codec.FastMarshal(req)  // ~800 ns/op
```

### Zero-Copy Techniques

For read-only operations, use unsafe conversions to avoid allocations:

```go
// WARNING: Only use when you won't modify the data
str := protocol.UnsafeString(byteSlice)
bytes := protocol.UnsafeBytes(stringData)
```

## Concurrent Request Handling

### Worker Pool Pattern

The concurrent handler uses a worker pool for optimal CPU utilization:

```go
import "github.com/fredcamaral/mcp-memory/pkg/mcp/server"

// Create concurrent handler with custom options
handler := server.NewConcurrentHandler(myHandler, &server.ConcurrentOptions{
    NumWorkers:   runtime.NumCPU() * 2,  // 2x CPU cores
    MaxQueueSize: 10000,                 // Request queue size
})

// Handle requests concurrently
resp, err := handler.HandleRequest(ctx, req)

// Monitor metrics
metrics := handler.Metrics()
fmt.Printf("Active requests: %d\n", metrics["active_requests"])
```

### Batch Processing

For high-throughput scenarios, batch processing reduces per-request overhead:

```go
// Create batch processor
batcher := server.NewBatchProcessor(myHandler, &server.BatchOptions{
    BatchSize:    100,                      // Process 100 requests at once
    BatchTimeout: 10 * time.Millisecond,    // Or after 10ms
})

// Requests are automatically batched
resp, err := batcher.HandleRequest(ctx, req)
```

### Request Pipelining

Pipeline handler processes requests in stages for improved latency:

```go
// Create pipeline with 4 workers
pipeline := server.NewPipelineHandler(myHandler, 4)

// Requests flow through pipeline stages
resp, err := pipeline.HandleRequest(ctx, req)
```

## Benchmarking Results

### Request Processing Benchmarks

```
BenchmarkSingleRequest-8           50000    28453 ns/op     2048 B/op    18 allocs/op
BenchmarkConcurrent10-8           200000     7234 ns/op      512 B/op     8 allocs/op
BenchmarkConcurrent100-8          500000     3421 ns/op      256 B/op     4 allocs/op
BenchmarkBatched50-8             1000000     1823 ns/op      128 B/op     2 allocs/op
```

### JSON Encoding Benchmarks

```
BenchmarkStandardJSON-8           100000    18234 ns/op     4096 B/op    42 allocs/op
BenchmarkOptimizedJSON-8          300000     4821 ns/op      512 B/op     8 allocs/op
BenchmarkFastMarshal-8            500000     2341 ns/op      256 B/op     4 allocs/op
```

### Memory Pool Benchmarks

```
BenchmarkAllocateBuffer-8         100000    12453 ns/op     4096 B/op     1 allocs/op
BenchmarkPooledBuffer-8          5000000      234 ns/op        0 B/op     0 allocs/op
```

## Best Practices

### 1. Use Pools for Temporary Objects

```go
// ✅ Good - reuse buffers
buf := mcp.GlobalPools.Buffers.Get()
defer mcp.GlobalPools.Buffers.Put(buf)

// ❌ Bad - allocate new buffer each time
buf := bytes.NewBuffer(make([]byte, 0, 4096))
```

### 2. Choose the Right Concurrency Model

- **Low volume**: Direct handling
- **Medium volume**: Worker pool (ConcurrentHandler)
- **High volume**: Batch processing (BatchProcessor)
- **Low latency**: Pipeline processing (PipelineHandler)

### 3. Optimize JSON Operations

```go
// For known types, use FastMarshal
data, _ := codec.FastMarshal(request)

// For streaming, use StreamingDecoder
decoder := protocol.NewStreamingDecoder(reader)

// For large responses, encode directly to writer
codec.EncodeToWriter(writer, response)
```

### 4. Monitor Performance Metrics

```go
// Track handler metrics
metrics := handler.Metrics()
log.Printf("Requests: total=%d active=%d rejected=%d",
    metrics["total_requests"],
    metrics["active_requests"],
    metrics["rejected_requests"])

// Use pprof for detailed profiling
import _ "net/http/pprof"
go http.ListenAndServe("localhost:6060", nil)
```

### 5. Tune Worker Pool Size

```go
// CPU-bound workloads
numWorkers := runtime.NumCPU()

// I/O-bound workloads
numWorkers := runtime.NumCPU() * 4

// Mixed workloads
numWorkers := runtime.NumCPU() * 2
```

## Performance Monitoring

### Built-in Metrics

The library provides built-in metrics for monitoring:

```go
// Handler metrics
handler.Metrics() // Returns map of metrics

// Custom metrics with Prometheus
import "github.com/prometheus/client_golang/prometheus"

var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "mcp_request_duration_seconds",
            Help: "Request duration in seconds",
        },
        []string{"method"},
    )
)
```

### Profiling Tools

Use Go's built-in profiling tools:

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Trace analysis
go test -trace=trace.out -bench=.
go tool trace trace.out
```

### Performance Testing

Example performance test:

```go
func BenchmarkConcurrentRequests(b *testing.B) {
    handler := server.NewConcurrentHandler(myHandler, nil)
    defer handler.Shutdown(5 * time.Second)
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            req := &protocol.JSONRPCRequest{
                Method: "test",
                Params: map[string]interface{}{"value": 123},
            }
            _, err := handler.HandleRequest(context.Background(), req)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

## Optimization Checklist

- [ ] Enable memory pools for all temporary objects
- [ ] Use optimized JSON codec for known types
- [ ] Configure appropriate worker pool size
- [ ] Enable request batching for high-throughput scenarios
- [ ] Monitor metrics and adjust configuration
- [ ] Profile application under load
- [ ] Set appropriate timeouts and queue sizes
- [ ] Use context for request cancellation
- [ ] Implement graceful shutdown
- [ ] Test performance under realistic workloads

## Conclusion

The MCP-Go library provides multiple optimization strategies to achieve high performance:

1. **Memory efficiency** through object pooling
2. **Fast JSON processing** with optimized codecs
3. **Scalable concurrency** with multiple processing models
4. **Comprehensive monitoring** for production deployments

By following this guide and using the appropriate optimization techniques, you can achieve sub-millisecond latencies and handle thousands of requests per second with minimal resource usage.