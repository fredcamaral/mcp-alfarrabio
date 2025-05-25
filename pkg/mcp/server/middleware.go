package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"mcp-memory/pkg/mcp/protocol"
)

// Middleware defines the interface for MCP middleware components
type Middleware interface {
	// Process handles the request/response cycle
	Process(ctx context.Context, request interface{}, next Handler) (interface{}, error)
}

// Handler represents a function that processes MCP requests
type Handler func(ctx context.Context, request interface{}) (interface{}, error)

// MiddlewareFunc is an adapter to allow ordinary functions to be used as middleware
type MiddlewareFunc func(ctx context.Context, request interface{}, next Handler) (interface{}, error)

// Process implements the Middleware interface for MiddlewareFunc
func (f MiddlewareFunc) Process(ctx context.Context, request interface{}, next Handler) (interface{}, error) {
	return f(ctx, request, next)
}

// Pipeline represents a chain of middleware
type Pipeline struct {
	middlewares []Middleware
	logger      *slog.Logger
}

// NewPipeline creates a new middleware pipeline
func NewPipeline(logger *slog.Logger) *Pipeline {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pipeline{
		middlewares: make([]Middleware, 0),
		logger:      logger,
	}
}

// Use adds middleware to the pipeline
func (p *Pipeline) Use(middleware ...Middleware) *Pipeline {
	p.middlewares = append(p.middlewares, middleware...)
	return p
}

// Execute runs the pipeline with the given handler
func (p *Pipeline) Execute(ctx context.Context, request interface{}, finalHandler Handler) (interface{}, error) {
	// Build the chain in reverse order
	handler := finalHandler
	for i := len(p.middlewares) - 1; i >= 0; i-- {
		middleware := p.middlewares[i]
		// Capture the current handler in the closure
		currentHandler := handler
		handler = func(ctx context.Context, req interface{}) (interface{}, error) {
			return middleware.Process(ctx, req, currentHandler)
		}
	}
	
	// Execute the chain
	return handler(ctx, request)
}

// Built-in middleware implementations

// LoggingMiddleware logs requests and responses
type LoggingMiddleware struct {
	logger *slog.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger *slog.Logger) *LoggingMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &LoggingMiddleware{logger: logger}
}

// Process implements the Middleware interface
func (m *LoggingMiddleware) Process(ctx context.Context, request interface{}, next Handler) (interface{}, error) {
	start := time.Now()
	
	// Log the request
	m.logger.InfoContext(ctx, "processing request",
		"type", fmt.Sprintf("%T", request),
		"request", request)
	
	// Call the next handler
	response, err := next(ctx, request)
	
	// Log the response
	duration := time.Since(start)
	if err != nil {
		m.logger.ErrorContext(ctx, "request failed",
			"type", fmt.Sprintf("%T", request),
			"duration", duration,
			"error", err)
	} else {
		m.logger.InfoContext(ctx, "request completed",
			"type", fmt.Sprintf("%T", request),
			"duration", duration,
			"response_type", fmt.Sprintf("%T", response))
	}
	
	return response, err
}

// RecoveryMiddleware handles panics and converts them to errors
type RecoveryMiddleware struct {
	logger *slog.Logger
}

// NewRecoveryMiddleware creates a new recovery middleware
func NewRecoveryMiddleware(logger *slog.Logger) *RecoveryMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &RecoveryMiddleware{logger: logger}
}

// Process implements the Middleware interface
func (m *RecoveryMiddleware) Process(ctx context.Context, request interface{}, next Handler) (response interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			m.logger.ErrorContext(ctx, "panic recovered",
				"panic", r,
				"request", request)
			
			// Convert panic to error
			switch v := r.(type) {
			case error:
				err = fmt.Errorf("panic: %w", v)
			default:
				err = fmt.Errorf("panic: %v", v)
			}
			
			// Return error response
			if req, ok := request.(*protocol.Request); ok {
				response = &protocol.Response{
					ID: req.ID,
					Error: &protocol.Error{
						Code:    protocol.InternalError,
						Message: "Internal server error",
					},
				}
			}
		}
	}()
	
	return next(ctx, request)
}

// MetricsMiddleware collects metrics about requests
type MetricsMiddleware struct {
	requestCount   map[string]int64
	requestLatency map[string][]time.Duration
	logger         *slog.Logger
}

// NewMetricsMiddleware creates a new metrics middleware
func NewMetricsMiddleware(logger *slog.Logger) *MetricsMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &MetricsMiddleware{
		requestCount:   make(map[string]int64),
		requestLatency: make(map[string][]time.Duration),
		logger:         logger,
	}
}

// Process implements the Middleware interface
func (m *MetricsMiddleware) Process(ctx context.Context, request interface{}, next Handler) (interface{}, error) {
	start := time.Now()
	requestType := fmt.Sprintf("%T", request)
	
	// Call the next handler
	response, err := next(ctx, request)
	
	// Record metrics
	duration := time.Since(start)
	m.recordMetrics(requestType, duration, err)
	
	return response, err
}

// recordMetrics records request metrics
func (m *MetricsMiddleware) recordMetrics(requestType string, duration time.Duration, err error) {
	m.requestCount[requestType]++
	m.requestLatency[requestType] = append(m.requestLatency[requestType], duration)
	
	// Log metrics periodically (in production, you'd export to Prometheus/etc)
	if m.requestCount[requestType]%100 == 0 {
		m.logger.Info("request metrics",
			"type", requestType,
			"count", m.requestCount[requestType],
			"avg_latency", m.calculateAvgLatency(requestType))
	}
}

// calculateAvgLatency calculates average latency for a request type
func (m *MetricsMiddleware) calculateAvgLatency(requestType string) time.Duration {
	latencies := m.requestLatency[requestType]
	if len(latencies) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, d := range latencies {
		total += d
	}
	return total / time.Duration(len(latencies))
}

// ContextMiddleware adds values to the context
type ContextMiddleware struct {
	values map[interface{}]interface{}
}

// NewContextMiddleware creates a new context middleware
func NewContextMiddleware() *ContextMiddleware {
	return &ContextMiddleware{
		values: make(map[interface{}]interface{}),
	}
}

// WithValue adds a key-value pair to be added to the context
func (m *ContextMiddleware) WithValue(key, value interface{}) *ContextMiddleware {
	m.values[key] = value
	return m
}

// Process implements the Middleware interface
func (m *ContextMiddleware) Process(ctx context.Context, request interface{}, next Handler) (interface{}, error) {
	// Add all values to the context
	for key, value := range m.values {
		ctx = context.WithValue(ctx, key, value)
	}
	
	return next(ctx, request)
}

// TimeoutMiddleware adds timeout handling to requests
type TimeoutMiddleware struct {
	timeout time.Duration
	logger  *slog.Logger
}

// NewTimeoutMiddleware creates a new timeout middleware
func NewTimeoutMiddleware(timeout time.Duration, logger *slog.Logger) *TimeoutMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &TimeoutMiddleware{
		timeout: timeout,
		logger:  logger,
	}
}

// Process implements the Middleware interface
func (m *TimeoutMiddleware) Process(ctx context.Context, request interface{}, next Handler) (interface{}, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	
	// Channel to receive the result
	type result struct {
		response interface{}
		err      error
	}
	resultChan := make(chan result, 1)
	
	// Execute the handler in a goroutine
	go func() {
		response, err := next(ctx, request)
		resultChan <- result{response, err}
	}()
	
	// Wait for either the result or timeout
	select {
	case res := <-resultChan:
		return res.response, res.err
	case <-ctx.Done():
		m.logger.ErrorContext(ctx, "request timeout",
			"timeout", m.timeout,
			"request", request)
		
		// Return timeout error
		if req, ok := request.(*protocol.Request); ok {
			return &protocol.Response{
				ID: req.ID,
				Error: &protocol.Error{
					Code:    protocol.InternalError,
					Message: "Request timeout",
				},
			}, nil
		}
		return nil, ctx.Err()
	}
}