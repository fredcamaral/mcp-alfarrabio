package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for MCP server
type Metrics struct {
	// Request metrics
	RequestDuration *prometheus.HistogramVec
	RequestCount    *prometheus.CounterVec
	ActiveRequests  *prometheus.GaugeVec

	// Error metrics
	ErrorCount *prometheus.CounterVec

	// Tool metrics
	ToolExecutionDuration *prometheus.HistogramVec
	ToolExecutionCount    *prometheus.CounterVec

	// Resource metrics
	ResourceOperationDuration *prometheus.HistogramVec
	ResourceOperationCount    *prometheus.CounterVec

	// Prompt metrics
	PromptOperationDuration *prometheus.HistogramVec
	PromptOperationCount    *prometheus.CounterVec

	// WebSocket metrics
	WebSocketConnections   prometheus.Gauge
	WebSocketMessagesSent  prometheus.Counter
	WebSocketMessagesRecvd prometheus.Counter

	// System metrics
	ServerUptime prometheus.Counter
}

// NewMetrics creates a new Metrics instance with all metrics registered
func NewMetrics(namespace, subsystem string) *Metrics {
	return &Metrics{
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "Duration of MCP requests in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "status"},
		),
		RequestCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_total",
				Help:      "Total number of MCP requests",
			},
			[]string{"method", "status"},
		),
		ActiveRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "active_requests",
				Help:      "Number of active MCP requests",
			},
			[]string{"method"},
		),
		ErrorCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "errors_total",
				Help:      "Total number of errors",
			},
			[]string{"method", "error_type"},
		),
		ToolExecutionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "tool_execution_duration_seconds",
				Help:      "Duration of tool executions in seconds",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
			},
			[]string{"tool_name"},
		),
		ToolExecutionCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "tool_execution_total",
				Help:      "Total number of tool executions",
			},
			[]string{"tool_name", "status"},
		),
		ResourceOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "resource_operation_duration_seconds",
				Help:      "Duration of resource operations in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation", "resource_type"},
		),
		ResourceOperationCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "resource_operation_total",
				Help:      "Total number of resource operations",
			},
			[]string{"operation", "resource_type", "status"},
		),
		PromptOperationDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "prompt_operation_duration_seconds",
				Help:      "Duration of prompt operations in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation"},
		),
		PromptOperationCount: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "prompt_operation_total",
				Help:      "Total number of prompt operations",
			},
			[]string{"operation", "status"},
		),
		WebSocketConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "websocket_connections",
				Help:      "Current number of WebSocket connections",
			},
		),
		WebSocketMessagesSent: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "websocket_messages_sent_total",
				Help:      "Total number of WebSocket messages sent",
			},
		),
		WebSocketMessagesRecvd: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "websocket_messages_received_total",
				Help:      "Total number of WebSocket messages received",
			},
		),
		ServerUptime: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "server_uptime_seconds_total",
				Help:      "Total server uptime in seconds",
			},
		),
	}
}

// Middleware provides a middleware for tracking HTTP metrics
type Middleware struct {
	metrics *Metrics
}

// NewMiddleware creates a new metrics middleware
func NewMiddleware(metrics *Metrics) *Middleware {
	return &Middleware{metrics: metrics}
}

// TrackRequest tracks metrics for an MCP request
func (m *Middleware) TrackRequest(method string, fn func() error) error {
	start := time.Now()
	
	// Increment active requests
	m.metrics.ActiveRequests.WithLabelValues(method).Inc()
	defer m.metrics.ActiveRequests.WithLabelValues(method).Dec()

	// Execute the function
	err := fn()

	// Record metrics
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
		m.metrics.ErrorCount.WithLabelValues(method, "request_error").Inc()
	}

	m.metrics.RequestDuration.WithLabelValues(method, status).Observe(duration)
	m.metrics.RequestCount.WithLabelValues(method, status).Inc()

	return err
}

// TrackToolExecution tracks metrics for tool execution
func (m *Middleware) TrackToolExecution(toolName string, fn func() error) error {
	start := time.Now()

	// Execute the function
	err := fn()

	// Record metrics
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
		m.metrics.ErrorCount.WithLabelValues(toolName, "tool_error").Inc()
	}

	m.metrics.ToolExecutionDuration.WithLabelValues(toolName).Observe(duration)
	m.metrics.ToolExecutionCount.WithLabelValues(toolName, status).Inc()

	return err
}

// TrackResourceOperation tracks metrics for resource operations
func (m *Middleware) TrackResourceOperation(operation, resourceType string, fn func() error) error {
	start := time.Now()

	// Execute the function
	err := fn()

	// Record metrics
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
		m.metrics.ErrorCount.WithLabelValues(operation, "resource_error").Inc()
	}

	m.metrics.ResourceOperationDuration.WithLabelValues(operation, resourceType).Observe(duration)
	m.metrics.ResourceOperationCount.WithLabelValues(operation, resourceType, status).Inc()

	return err
}

// TrackPromptOperation tracks metrics for prompt operations
func (m *Middleware) TrackPromptOperation(operation string, fn func() error) error {
	start := time.Now()

	// Execute the function
	err := fn()

	// Record metrics
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
		m.metrics.ErrorCount.WithLabelValues(operation, "prompt_error").Inc()
	}

	m.metrics.PromptOperationDuration.WithLabelValues(operation).Observe(duration)
	m.metrics.PromptOperationCount.WithLabelValues(operation, status).Inc()

	return err
}

// IncrementWebSocketConnection increments the WebSocket connection gauge
func (m *Middleware) IncrementWebSocketConnection() {
	m.metrics.WebSocketConnections.Inc()
}

// DecrementWebSocketConnection decrements the WebSocket connection gauge
func (m *Middleware) DecrementWebSocketConnection() {
	m.metrics.WebSocketConnections.Dec()
}

// IncrementWebSocketMessagesSent increments the sent messages counter
func (m *Middleware) IncrementWebSocketMessagesSent() {
	m.metrics.WebSocketMessagesSent.Inc()
}

// IncrementWebSocketMessagesReceived increments the received messages counter
func (m *Middleware) IncrementWebSocketMessagesReceived() {
	m.metrics.WebSocketMessagesRecvd.Inc()
}

// StartUptimeCounter starts a goroutine that increments uptime counter
func (m *Middleware) StartUptimeCounter(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.metrics.ServerUptime.Inc()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Handler provides HTTP handler for Prometheus metrics
func Handler() http.Handler {
	return promhttp.Handler()
}