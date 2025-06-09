// Package monitoring provides comprehensive API metrics collection and monitoring.
package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// MetricsCollector collects and aggregates API metrics
type MetricsCollector struct {
	httpMetrics     *HTTPMetrics
	endpointMetrics map[string]*EndpointMetrics
	systemMetrics   *SystemMetrics
	config          MetricsConfig
	mu              sync.RWMutex
	startTime       time.Time
}

// HTTPMetrics tracks overall HTTP metrics
type HTTPMetrics struct {
	TotalRequests       int64           `json:"total_requests"`
	RequestsPerSecond   float64         `json:"requests_per_second"`
	AverageResponseTime time.Duration   `json:"average_response_time"`
	P95ResponseTime     time.Duration   `json:"p95_response_time"`
	P99ResponseTime     time.Duration   `json:"p99_response_time"`
	StatusCodes         map[int]int64   `json:"status_codes"`
	ErrorRate           float64         `json:"error_rate"`
	ThroughputMBPS      float64         `json:"throughput_mbps"`
	ActiveConnections   int64           `json:"active_connections"`
	ResponseTimes       []time.Duration `json:"-"` // For percentile calculation
	mu                  sync.RWMutex
}

// EndpointMetrics tracks per-endpoint metrics
type EndpointMetrics struct {
	Endpoint            string          `json:"endpoint"`
	Method              string          `json:"method"`
	TotalRequests       int64           `json:"total_requests"`
	SuccessRequests     int64           `json:"success_requests"`
	ErrorRequests       int64           `json:"error_requests"`
	AverageResponseTime time.Duration   `json:"average_response_time"`
	MinResponseTime     time.Duration   `json:"min_response_time"`
	MaxResponseTime     time.Duration   `json:"max_response_time"`
	StatusCodes         map[int]int64   `json:"status_codes"`
	ErrorRate           float64         `json:"error_rate"`
	RequestSize         int64           `json:"average_request_size"`
	ResponseSize        int64           `json:"average_response_size"`
	LastActivity        time.Time       `json:"last_activity"`
	ResponseTimes       []time.Duration `json:"-"`
	mu                  sync.RWMutex
}

// SystemMetrics tracks system-level metrics
type SystemMetrics struct {
	Uptime              time.Duration `json:"uptime"`
	MemoryUsage         int64         `json:"memory_usage_bytes"`
	CPUUsage            float64       `json:"cpu_usage_percent"`
	GoroutineCount      int           `json:"goroutine_count"`
	HeapSize            int64         `json:"heap_size_bytes"`
	GCPauses            time.Duration `json:"gc_pause_total"`
	ConnectionsActive   int64         `json:"connections_active"`
	ConnectionsIdle     int64         `json:"connections_idle"`
	DatabaseConnections int           `json:"database_connections"`
	mu                  sync.RWMutex
}

// MetricsConfig represents metrics collection configuration
type MetricsConfig struct {
	Enabled               bool          `json:"enabled"`
	CollectionInterval    time.Duration `json:"collection_interval"`
	RetentionPeriod       time.Duration `json:"retention_period"`
	MaxEndpoints          int           `json:"max_endpoints"`
	PercentileWindow      int           `json:"percentile_window"`
	EnableDetailedMetrics bool          `json:"enable_detailed_metrics"`
	ExportInterval        time.Duration `json:"export_interval"`
	ExportFormat          string        `json:"export_format"`
}

// RequestMetrics represents metrics for a single request
type RequestMetrics struct {
	Method       string        `json:"method"`
	Endpoint     string        `json:"endpoint"`
	StatusCode   int           `json:"status_code"`
	ResponseTime time.Duration `json:"response_time"`
	RequestSize  int64         `json:"request_size"`
	ResponseSize int64         `json:"response_size"`
	UserAgent    string        `json:"user_agent"`
	ClientIP     string        `json:"client_ip"`
	Timestamp    time.Time     `json:"timestamp"`
	Error        string        `json:"error,omitempty"`
}

// MetricsSummary provides a summary of all metrics
type MetricsSummary struct {
	HTTP      *HTTPMetrics                `json:"http"`
	Endpoints map[string]*EndpointMetrics `json:"endpoints"`
	System    *SystemMetrics              `json:"system"`
	Timestamp time.Time                   `json:"timestamp"`
	Uptime    time.Duration               `json:"uptime"`
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled:               true,
		CollectionInterval:    time.Second,
		RetentionPeriod:       24 * time.Hour,
		MaxEndpoints:          1000,
		PercentileWindow:      1000,
		EnableDetailedMetrics: true,
		ExportInterval:        5 * time.Minute,
		ExportFormat:          "json",
	}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config MetricsConfig) *MetricsCollector {
	mc := &MetricsCollector{
		httpMetrics:     NewHTTPMetrics(),
		endpointMetrics: make(map[string]*EndpointMetrics),
		systemMetrics:   NewSystemMetrics(),
		config:          config,
		startTime:       time.Now(),
	}

	// Start collection routines
	if config.Enabled {
		go mc.collectSystemMetrics()
		go mc.calculatePercentiles()
		go mc.cleanupOldMetrics()
	}

	return mc
}

// RecordRequest records metrics for an HTTP request
func (mc *MetricsCollector) RecordRequest(metrics *RequestMetrics) {
	if !mc.config.Enabled {
		return
	}

	// Record HTTP metrics
	mc.recordHTTPMetrics(metrics)

	// Record endpoint metrics
	mc.recordEndpointMetrics(metrics)
}

// Middleware returns HTTP middleware for metrics collection
func (mc *MetricsCollector) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !mc.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			startTime := time.Now()

			// Wrap response writer to capture metrics
			wrapper := NewResponseWriterWrapper(w)

			// Record active connection
			mc.httpMetrics.mu.Lock()
			mc.httpMetrics.ActiveConnections++
			mc.httpMetrics.mu.Unlock()

			// Execute handler
			next.ServeHTTP(wrapper, r)

			// Record metrics
			responseTime := time.Since(startTime)

			requestMetrics := &RequestMetrics{
				Method:       r.Method,
				Endpoint:     mc.normalizeEndpoint(r.URL.Path),
				StatusCode:   wrapper.StatusCode(),
				ResponseTime: responseTime,
				RequestSize:  r.ContentLength,
				ResponseSize: int64(wrapper.BytesWritten()),
				UserAgent:    r.UserAgent(),
				ClientIP:     mc.getClientIP(r),
				Timestamp:    startTime,
			}

			mc.RecordRequest(requestMetrics)

			// Record connection completion
			mc.httpMetrics.mu.Lock()
			mc.httpMetrics.ActiveConnections--
			mc.httpMetrics.mu.Unlock()
		})
	}
}

// GetMetrics returns current metrics summary
func (mc *MetricsCollector) GetMetrics() *MetricsSummary {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Create copies to avoid race conditions
	httpMetrics := mc.copyHTTPMetrics()
	endpointMetrics := mc.copyEndpointMetrics()
	systemMetrics := mc.copySystemMetrics()

	return &MetricsSummary{
		HTTP:      httpMetrics,
		Endpoints: endpointMetrics,
		System:    systemMetrics,
		Timestamp: time.Now(),
		Uptime:    time.Since(mc.startTime),
	}
}

// GetHTTPMetrics returns HTTP metrics
func (mc *MetricsCollector) GetHTTPMetrics() *HTTPMetrics {
	return mc.copyHTTPMetrics()
}

// GetEndpointMetrics returns metrics for a specific endpoint
func (mc *MetricsCollector) GetEndpointMetrics(endpoint string) *EndpointMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if metrics, exists := mc.endpointMetrics[endpoint]; exists {
		return mc.copyEndpointMetric(metrics)
	}

	return nil
}

// GetSystemMetrics returns system metrics
func (mc *MetricsCollector) GetSystemMetrics() *SystemMetrics {
	return mc.copySystemMetrics()
}

// ExportMetrics exports metrics in specified format
func (mc *MetricsCollector) ExportMetrics(format string) ([]byte, error) {
	metrics := mc.GetMetrics()

	switch format {
	case "json":
		return json.MarshalIndent(metrics, "", "  ")
	case "prometheus":
		return mc.exportPrometheusFormat(metrics)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// Helper methods

func (mc *MetricsCollector) recordHTTPMetrics(metrics *RequestMetrics) {
	mc.httpMetrics.mu.Lock()
	defer mc.httpMetrics.mu.Unlock()

	mc.httpMetrics.TotalRequests++

	// Update status codes
	if mc.httpMetrics.StatusCodes == nil {
		mc.httpMetrics.StatusCodes = make(map[int]int64)
	}
	mc.httpMetrics.StatusCodes[metrics.StatusCode]++

	// Update response times
	mc.httpMetrics.ResponseTimes = append(mc.httpMetrics.ResponseTimes, metrics.ResponseTime)

	// Keep only recent response times for percentile calculation
	if len(mc.httpMetrics.ResponseTimes) > mc.config.PercentileWindow {
		mc.httpMetrics.ResponseTimes = mc.httpMetrics.ResponseTimes[1:]
	}

	// Update average response time
	mc.updateAverageResponseTime(&mc.httpMetrics.AverageResponseTime, metrics.ResponseTime, mc.httpMetrics.TotalRequests)

	// Calculate error rate
	errorCount := int64(0)
	for code, count := range mc.httpMetrics.StatusCodes {
		if code >= 400 {
			errorCount += count
		}
	}
	mc.httpMetrics.ErrorRate = float64(errorCount) / float64(mc.httpMetrics.TotalRequests)
}

func (mc *MetricsCollector) recordEndpointMetrics(metrics *RequestMetrics) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := fmt.Sprintf("%s %s", metrics.Method, metrics.Endpoint)

	endpointMetric, exists := mc.endpointMetrics[key]
	if !exists {
		endpointMetric = NewEndpointMetrics(metrics.Endpoint, metrics.Method)
		mc.endpointMetrics[key] = endpointMetric
	}

	endpointMetric.mu.Lock()
	defer endpointMetric.mu.Unlock()

	endpointMetric.TotalRequests++
	endpointMetric.LastActivity = metrics.Timestamp

	// Update status codes
	if endpointMetric.StatusCodes == nil {
		endpointMetric.StatusCodes = make(map[int]int64)
	}
	endpointMetric.StatusCodes[metrics.StatusCode]++

	// Update success/error counts
	if metrics.StatusCode < 400 {
		endpointMetric.SuccessRequests++
	} else {
		endpointMetric.ErrorRequests++
	}

	// Update response times
	endpointMetric.ResponseTimes = append(endpointMetric.ResponseTimes, metrics.ResponseTime)
	if len(endpointMetric.ResponseTimes) > mc.config.PercentileWindow {
		endpointMetric.ResponseTimes = endpointMetric.ResponseTimes[1:]
	}

	// Update average response time
	mc.updateAverageResponseTime(&endpointMetric.AverageResponseTime, metrics.ResponseTime, endpointMetric.TotalRequests)

	// Update min/max response times
	if endpointMetric.MinResponseTime == 0 || metrics.ResponseTime < endpointMetric.MinResponseTime {
		endpointMetric.MinResponseTime = metrics.ResponseTime
	}
	if metrics.ResponseTime > endpointMetric.MaxResponseTime {
		endpointMetric.MaxResponseTime = metrics.ResponseTime
	}

	// Update error rate
	endpointMetric.ErrorRate = float64(endpointMetric.ErrorRequests) / float64(endpointMetric.TotalRequests)

	// Update average sizes
	if metrics.RequestSize > 0 {
		endpointMetric.RequestSize = (endpointMetric.RequestSize + metrics.RequestSize) / 2
	}
	if metrics.ResponseSize > 0 {
		endpointMetric.ResponseSize = (endpointMetric.ResponseSize + metrics.ResponseSize) / 2
	}
}

func (mc *MetricsCollector) updateAverageResponseTime(avg *time.Duration, newTime time.Duration, count int64) {
	if count == 1 {
		*avg = newTime
	} else {
		// Exponential moving average
		alpha := 0.1 // Smoothing factor
		*avg = time.Duration(alpha*float64(newTime) + (1-alpha)*float64(*avg))
	}
}

func (mc *MetricsCollector) normalizeEndpoint(path string) string {
	// Normalize paths to group similar endpoints
	// This is a simple implementation - can be enhanced with regex patterns

	// Remove trailing slash
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	// Replace UUIDs and numbers with placeholders
	// This is simplified - in practice you'd use more sophisticated pattern matching
	if mc.containsUUID(path) {
		return mc.replaceUUIDs(path)
	}

	return path
}

func (mc *MetricsCollector) containsUUID(path string) bool {
	// Simple UUID detection - can be enhanced
	return len(path) > 36 // Minimum length for UUID
}

func (mc *MetricsCollector) replaceUUIDs(path string) string {
	// Simple UUID replacement - can be enhanced with regex
	return path // For now, return as-is
}

func (mc *MetricsCollector) getClientIP(r *http.Request) string {
	// Extract client IP from request
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}

func (mc *MetricsCollector) collectSystemMetrics() {
	ticker := time.NewTicker(mc.config.CollectionInterval)
	defer ticker.Stop()

	for range ticker.C {
		mc.updateSystemMetrics()
	}
}

func (mc *MetricsCollector) updateSystemMetrics() {
	// This would integrate with system monitoring libraries
	// For now, providing placeholder implementation
	mc.systemMetrics.mu.Lock()
	defer mc.systemMetrics.mu.Unlock()

	mc.systemMetrics.Uptime = time.Since(mc.startTime)
	// Other metrics would be collected from runtime and system
}

func (mc *MetricsCollector) calculatePercentiles() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mc.updatePercentiles()
	}
}

func (mc *MetricsCollector) updatePercentiles() {
	mc.httpMetrics.mu.Lock()
	defer mc.httpMetrics.mu.Unlock()

	if len(mc.httpMetrics.ResponseTimes) == 0 {
		return
	}

	// Calculate percentiles
	mc.httpMetrics.P95ResponseTime = mc.calculatePercentile(mc.httpMetrics.ResponseTimes, 95)
	mc.httpMetrics.P99ResponseTime = mc.calculatePercentile(mc.httpMetrics.ResponseTimes, 99)
}

func (mc *MetricsCollector) calculatePercentile(times []time.Duration, percentile int) time.Duration {
	if len(times) == 0 {
		return 0
	}

	// Create a copy and sort
	sorted := make([]time.Duration, len(times))
	copy(sorted, times)

	// Simple insertion sort for small arrays
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}

	// Calculate percentile index
	index := (percentile * len(sorted)) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

func (mc *MetricsCollector) cleanupOldMetrics() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		mc.performCleanup()
	}
}

func (mc *MetricsCollector) performCleanup() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoff := time.Now().Add(-mc.config.RetentionPeriod)

	// Clean up inactive endpoints
	for key, metrics := range mc.endpointMetrics {
		if metrics.LastActivity.Before(cutoff) {
			delete(mc.endpointMetrics, key)
		}
	}

	// Limit number of endpoints
	if len(mc.endpointMetrics) > mc.config.MaxEndpoints {
		// Remove least active endpoints
		// This is simplified - in practice you'd use a more sophisticated eviction policy
		count := 0
		for key := range mc.endpointMetrics {
			delete(mc.endpointMetrics, key)
			count++
			if count >= len(mc.endpointMetrics)/4 {
				break
			}
		}
	}
}

func (mc *MetricsCollector) copyHTTPMetrics() *HTTPMetrics {
	mc.httpMetrics.mu.RLock()
	defer mc.httpMetrics.mu.RUnlock()

	metrics := &HTTPMetrics{
		TotalRequests:       mc.httpMetrics.TotalRequests,
		RequestsPerSecond:   mc.httpMetrics.RequestsPerSecond,
		AverageResponseTime: mc.httpMetrics.AverageResponseTime,
		P95ResponseTime:     mc.httpMetrics.P95ResponseTime,
		P99ResponseTime:     mc.httpMetrics.P99ResponseTime,
		ErrorRate:           mc.httpMetrics.ErrorRate,
		ThroughputMBPS:      mc.httpMetrics.ThroughputMBPS,
		ActiveConnections:   mc.httpMetrics.ActiveConnections,
		StatusCodes:         make(map[int]int64),
	}
	for k, v := range mc.httpMetrics.StatusCodes {
		metrics.StatusCodes[k] = v
	}

	return metrics
}

func (mc *MetricsCollector) copyEndpointMetrics() map[string]*EndpointMetrics {
	metrics := make(map[string]*EndpointMetrics)
	for k, v := range mc.endpointMetrics {
		metrics[k] = mc.copyEndpointMetric(v)
	}
	return metrics
}

func (mc *MetricsCollector) copyEndpointMetric(original *EndpointMetrics) *EndpointMetrics {
	original.mu.RLock()
	defer original.mu.RUnlock()

	metrics := &EndpointMetrics{
		Endpoint:            original.Endpoint,
		Method:              original.Method,
		TotalRequests:       original.TotalRequests,
		SuccessRequests:     original.SuccessRequests,
		ErrorRequests:       original.ErrorRequests,
		AverageResponseTime: original.AverageResponseTime,
		MinResponseTime:     original.MinResponseTime,
		MaxResponseTime:     original.MaxResponseTime,
		ErrorRate:           original.ErrorRate,
		RequestSize:         original.RequestSize,
		ResponseSize:        original.ResponseSize,
		LastActivity:        original.LastActivity,
		StatusCodes:         make(map[int]int64),
	}
	for k, v := range original.StatusCodes {
		metrics.StatusCodes[k] = v
	}

	return metrics
}

func (mc *MetricsCollector) copySystemMetrics() *SystemMetrics {
	mc.systemMetrics.mu.RLock()
	defer mc.systemMetrics.mu.RUnlock()

	return &SystemMetrics{
		Uptime:              mc.systemMetrics.Uptime,
		MemoryUsage:         mc.systemMetrics.MemoryUsage,
		CPUUsage:            mc.systemMetrics.CPUUsage,
		GoroutineCount:      mc.systemMetrics.GoroutineCount,
		HeapSize:            mc.systemMetrics.HeapSize,
		GCPauses:            mc.systemMetrics.GCPauses,
		ConnectionsActive:   mc.systemMetrics.ConnectionsActive,
		ConnectionsIdle:     mc.systemMetrics.ConnectionsIdle,
		DatabaseConnections: mc.systemMetrics.DatabaseConnections,
	}
}

func (mc *MetricsCollector) exportPrometheusFormat(metrics *MetricsSummary) ([]byte, error) {
	// This would implement Prometheus format export
	// For now, return JSON as placeholder
	return json.MarshalIndent(metrics, "", "  ")
}

// Constructor functions

func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		StatusCodes:   make(map[int]int64),
		ResponseTimes: make([]time.Duration, 0),
	}
}

func NewEndpointMetrics(endpoint, method string) *EndpointMetrics {
	return &EndpointMetrics{
		Endpoint:      endpoint,
		Method:        method,
		StatusCodes:   make(map[int]int64),
		ResponseTimes: make([]time.Duration, 0),
	}
}

func NewSystemMetrics() *SystemMetrics {
	return &SystemMetrics{}
}

// ResponseWriterWrapper wraps http.ResponseWriter to capture metrics
type ResponseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{
		ResponseWriter: w,
		statusCode:     200, // Default status code
	}
}

func (w *ResponseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *ResponseWriterWrapper) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += n
	return n, err
}

func (w *ResponseWriterWrapper) StatusCode() int {
	return w.statusCode
}

func (w *ResponseWriterWrapper) BytesWritten() int {
	return w.bytesWritten
}
