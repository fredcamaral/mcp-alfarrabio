// Package websocket provides comprehensive metrics collection for WebSocket connections
package websocket

import (
	"context"
	"sync"
	"time"
)

// Metrics collects and aggregates WebSocket server metrics
type Metrics struct {
	connectionMetrics *ConnectionMetrics
	messageMetrics    *MessageMetrics
	errorMetrics      *ErrorMetrics
	performanceMetrics *PerformanceMetrics
	mu                sync.RWMutex
	startTime         time.Time
}

// ConnectionMetrics tracks connection-related metrics
type ConnectionMetrics struct {
	TotalConnections      int64         `json:"total_connections"`
	ActiveConnections     int64         `json:"active_connections"`
	ConnectionsAccepted   int64         `json:"connections_accepted"`
	ConnectionsRejected   int64         `json:"connections_rejected"`
	ConnectionsClosed     int64         `json:"connections_closed"`
	AverageConnectionTime time.Duration `json:"average_connection_time"`
	MaxConcurrentConnections int64      `json:"max_concurrent_connections"`
	RejectionReasons      map[string]int64 `json:"rejection_reasons"`
	mu                    sync.RWMutex
}

// MessageMetrics tracks message-related metrics
type MessageMetrics struct {
	TotalMessagesSent     int64   `json:"total_messages_sent"`
	TotalMessagesReceived int64   `json:"total_messages_received"`
	TotalBytesSent        int64   `json:"total_bytes_sent"`
	TotalBytesReceived    int64   `json:"total_bytes_received"`
	AverageMessageSize    float64 `json:"average_message_size"`
	MessagesPerSecond     float64 `json:"messages_per_second"`
	MessageTypes          map[string]int64 `json:"message_types"`
	mu                    sync.RWMutex
}

// ErrorMetrics tracks error-related metrics
type ErrorMetrics struct {
	TotalErrors           int64            `json:"total_errors"`
	ConnectionErrors      int64            `json:"connection_errors"`
	MessageErrors         int64            `json:"message_errors"`
	TimeoutErrors         int64            `json:"timeout_errors"`
	AuthenticationErrors  int64            `json:"authentication_errors"`
	ErrorsByType          map[string]int64 `json:"errors_by_type"`
	LastError             string           `json:"last_error"`
	LastErrorTime         time.Time        `json:"last_error_time"`
	mu                    sync.RWMutex
}

// PerformanceMetrics tracks performance-related metrics
type PerformanceMetrics struct {
	AverageLatency        time.Duration `json:"average_latency"`
	P95Latency            time.Duration `json:"p95_latency"`
	P99Latency            time.Duration `json:"p99_latency"`
	MaxLatency            time.Duration `json:"max_latency"`
	MinLatency            time.Duration `json:"min_latency"`
	ThroughputMBPS        float64       `json:"throughput_mbps"`
	MemoryUsage           int64         `json:"memory_usage_bytes"`
	GoroutineCount        int           `json:"goroutine_count"`
	LatencyHistogram      []int64       `json:"latency_histogram"`
	mu                    sync.RWMutex
}

// MetricsSummary provides a consolidated view of all metrics
type MetricsSummary struct {
	Connections *ConnectionMetrics  `json:"connections"`
	Messages    *MessageMetrics     `json:"messages"`
	Errors      *ErrorMetrics       `json:"errors"`
	Performance *PerformanceMetrics `json:"performance"`
	Uptime      time.Duration       `json:"uptime"`
	Timestamp   time.Time           `json:"timestamp"`
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		connectionMetrics: &ConnectionMetrics{
			RejectionReasons: make(map[string]int64),
		},
		messageMetrics: &MessageMetrics{
			MessageTypes: make(map[string]int64),
		},
		errorMetrics: &ErrorMetrics{
			ErrorsByType: make(map[string]int64),
		},
		performanceMetrics: &PerformanceMetrics{
			MinLatency:       time.Hour, // Initialize with high value
			LatencyHistogram: make([]int64, 10), // 10 buckets for latency distribution
		},
		startTime: time.Now(),
	}
}

// Start begins metrics collection
func (m *Metrics) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.calculateDerivedMetrics()
		case <-ctx.Done():
			return
		}
	}
}

// RecordConnectionAccepted records a successful connection
func (m *Metrics) RecordConnectionAccepted() {
	m.connectionMetrics.mu.Lock()
	defer m.connectionMetrics.mu.Unlock()

	m.connectionMetrics.TotalConnections++
	m.connectionMetrics.ConnectionsAccepted++
	m.connectionMetrics.ActiveConnections++

	if m.connectionMetrics.ActiveConnections > m.connectionMetrics.MaxConcurrentConnections {
		m.connectionMetrics.MaxConcurrentConnections = m.connectionMetrics.ActiveConnections
	}
}

// RecordConnectionRejected records a rejected connection
func (m *Metrics) RecordConnectionRejected(reason string) {
	m.connectionMetrics.mu.Lock()
	defer m.connectionMetrics.mu.Unlock()

	m.connectionMetrics.ConnectionsRejected++
	m.connectionMetrics.RejectionReasons[reason]++
}

// RecordConnectionClosed records a closed connection
func (m *Metrics) RecordConnectionClosed(connectionTime time.Duration) {
	m.connectionMetrics.mu.Lock()
	defer m.connectionMetrics.mu.Unlock()

	m.connectionMetrics.ConnectionsClosed++
	m.connectionMetrics.ActiveConnections--

	// Update average connection time
	if m.connectionMetrics.AverageConnectionTime == 0 {
		m.connectionMetrics.AverageConnectionTime = connectionTime
	} else {
		// Exponential moving average
		m.connectionMetrics.AverageConnectionTime = time.Duration(
			int64(m.connectionMetrics.AverageConnectionTime)*9/10 + int64(connectionTime)/10,
		)
	}
}

// RecordMessageSent records a sent message
func (m *Metrics) RecordMessageSent(messageType string, size int64) {
	m.messageMetrics.mu.Lock()
	defer m.messageMetrics.mu.Unlock()

	m.messageMetrics.TotalMessagesSent++
	m.messageMetrics.TotalBytesSent += size
	m.messageMetrics.MessageTypes[messageType]++

	// Update average message size
	if m.messageMetrics.TotalMessagesSent > 0 {
		m.messageMetrics.AverageMessageSize = float64(m.messageMetrics.TotalBytesSent) / float64(m.messageMetrics.TotalMessagesSent)
	}
}

// RecordMessageReceived records a received message
func (m *Metrics) RecordMessageReceived(messageType string, size int64) {
	m.messageMetrics.mu.Lock()
	defer m.messageMetrics.mu.Unlock()

	m.messageMetrics.TotalMessagesReceived++
	m.messageMetrics.TotalBytesReceived += size
	m.messageMetrics.MessageTypes[messageType]++
}

// RecordLatency records message latency
func (m *Metrics) RecordLatency(latency time.Duration) {
	m.performanceMetrics.mu.Lock()
	defer m.performanceMetrics.mu.Unlock()

	// Update latency statistics
	if latency > m.performanceMetrics.MaxLatency {
		m.performanceMetrics.MaxLatency = latency
	}
	if latency < m.performanceMetrics.MinLatency {
		m.performanceMetrics.MinLatency = latency
	}

	// Update average latency
	if m.performanceMetrics.AverageLatency == 0 {
		m.performanceMetrics.AverageLatency = latency
	} else {
		m.performanceMetrics.AverageLatency = time.Duration(
			int64(m.performanceMetrics.AverageLatency)*9/10 + int64(latency)/10,
		)
	}

	// Update latency histogram
	bucket := m.getLatencyBucket(latency)
	if bucket < len(m.performanceMetrics.LatencyHistogram) {
		m.performanceMetrics.LatencyHistogram[bucket]++
	}
}

// RecordError records an error
func (m *Metrics) RecordError(errorType, message string) {
	m.errorMetrics.mu.Lock()
	defer m.errorMetrics.mu.Unlock()

	m.errorMetrics.TotalErrors++
	m.errorMetrics.ErrorsByType[errorType]++
	m.errorMetrics.LastError = message
	m.errorMetrics.LastErrorTime = time.Now()

	switch errorType {
	case "connection":
		m.errorMetrics.ConnectionErrors++
	case "message":
		m.errorMetrics.MessageErrors++
	case "timeout":
		m.errorMetrics.TimeoutErrors++
	case "authentication":
		m.errorMetrics.AuthenticationErrors++
	}
}

// GetSummary returns a complete metrics summary
func (m *Metrics) GetSummary() *MetricsSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &MetricsSummary{
		Connections: m.copyConnectionMetrics(),
		Messages:    m.copyMessageMetrics(),
		Errors:      m.copyErrorMetrics(),
		Performance: m.copyPerformanceMetrics(),
		Uptime:      time.Since(m.startTime),
		Timestamp:   time.Now(),
	}
}

// GetConnectionMetrics returns connection metrics
func (m *Metrics) GetConnectionMetrics() *ConnectionMetrics {
	return m.copyConnectionMetrics()
}

// GetMessageMetrics returns message metrics
func (m *Metrics) GetMessageMetrics() *MessageMetrics {
	return m.copyMessageMetrics()
}

// GetErrorMetrics returns error metrics
func (m *Metrics) GetErrorMetrics() *ErrorMetrics {
	return m.copyErrorMetrics()
}

// GetPerformanceMetrics returns performance metrics
func (m *Metrics) GetPerformanceMetrics() *PerformanceMetrics {
	return m.copyPerformanceMetrics()
}

// calculateDerivedMetrics calculates metrics that depend on time
func (m *Metrics) calculateDerivedMetrics() {
	m.messageMetrics.mu.Lock()
	defer m.messageMetrics.mu.Unlock()

	uptime := time.Since(m.startTime)
	if uptime > 0 {
		// Calculate messages per second
		totalMessages := m.messageMetrics.TotalMessagesSent + m.messageMetrics.TotalMessagesReceived
		m.messageMetrics.MessagesPerSecond = float64(totalMessages) / uptime.Seconds()

		// Calculate throughput in MBPS
		totalBytes := m.messageMetrics.TotalBytesSent + m.messageMetrics.TotalBytesReceived
		m.performanceMetrics.mu.Lock()
		m.performanceMetrics.ThroughputMBPS = float64(totalBytes) / (1024 * 1024) / uptime.Seconds()
		m.performanceMetrics.mu.Unlock()
	}
}

// copyConnectionMetrics returns a copy of connection metrics
func (m *Metrics) copyConnectionMetrics() *ConnectionMetrics {
	m.connectionMetrics.mu.RLock()
	defer m.connectionMetrics.mu.RUnlock()

	rejectionReasons := make(map[string]int64)
	for k, v := range m.connectionMetrics.RejectionReasons {
		rejectionReasons[k] = v
	}

	return &ConnectionMetrics{
		TotalConnections:         m.connectionMetrics.TotalConnections,
		ActiveConnections:        m.connectionMetrics.ActiveConnections,
		ConnectionsAccepted:      m.connectionMetrics.ConnectionsAccepted,
		ConnectionsRejected:      m.connectionMetrics.ConnectionsRejected,
		ConnectionsClosed:        m.connectionMetrics.ConnectionsClosed,
		AverageConnectionTime:    m.connectionMetrics.AverageConnectionTime,
		MaxConcurrentConnections: m.connectionMetrics.MaxConcurrentConnections,
		RejectionReasons:         rejectionReasons,
	}
}

// copyMessageMetrics returns a copy of message metrics
func (m *Metrics) copyMessageMetrics() *MessageMetrics {
	m.messageMetrics.mu.RLock()
	defer m.messageMetrics.mu.RUnlock()

	messageTypes := make(map[string]int64)
	for k, v := range m.messageMetrics.MessageTypes {
		messageTypes[k] = v
	}

	return &MessageMetrics{
		TotalMessagesSent:     m.messageMetrics.TotalMessagesSent,
		TotalMessagesReceived: m.messageMetrics.TotalMessagesReceived,
		TotalBytesSent:        m.messageMetrics.TotalBytesSent,
		TotalBytesReceived:    m.messageMetrics.TotalBytesReceived,
		AverageMessageSize:    m.messageMetrics.AverageMessageSize,
		MessagesPerSecond:     m.messageMetrics.MessagesPerSecond,
		MessageTypes:          messageTypes,
	}
}

// copyErrorMetrics returns a copy of error metrics
func (m *Metrics) copyErrorMetrics() *ErrorMetrics {
	m.errorMetrics.mu.RLock()
	defer m.errorMetrics.mu.RUnlock()

	errorsByType := make(map[string]int64)
	for k, v := range m.errorMetrics.ErrorsByType {
		errorsByType[k] = v
	}

	return &ErrorMetrics{
		TotalErrors:          m.errorMetrics.TotalErrors,
		ConnectionErrors:     m.errorMetrics.ConnectionErrors,
		MessageErrors:        m.errorMetrics.MessageErrors,
		TimeoutErrors:        m.errorMetrics.TimeoutErrors,
		AuthenticationErrors: m.errorMetrics.AuthenticationErrors,
		ErrorsByType:         errorsByType,
		LastError:            m.errorMetrics.LastError,
		LastErrorTime:        m.errorMetrics.LastErrorTime,
	}
}

// copyPerformanceMetrics returns a copy of performance metrics
func (m *Metrics) copyPerformanceMetrics() *PerformanceMetrics {
	m.performanceMetrics.mu.RLock()
	defer m.performanceMetrics.mu.RUnlock()

	histogram := make([]int64, len(m.performanceMetrics.LatencyHistogram))
	copy(histogram, m.performanceMetrics.LatencyHistogram)

	return &PerformanceMetrics{
		AverageLatency:   m.performanceMetrics.AverageLatency,
		P95Latency:       m.performanceMetrics.P95Latency,
		P99Latency:       m.performanceMetrics.P99Latency,
		MaxLatency:       m.performanceMetrics.MaxLatency,
		MinLatency:       m.performanceMetrics.MinLatency,
		ThroughputMBPS:   m.performanceMetrics.ThroughputMBPS,
		MemoryUsage:      m.performanceMetrics.MemoryUsage,
		GoroutineCount:   m.performanceMetrics.GoroutineCount,
		LatencyHistogram: histogram,
	}
}

// getLatencyBucket returns the histogram bucket for a given latency
func (m *Metrics) getLatencyBucket(latency time.Duration) int {
	// Define latency buckets: 0-1ms, 1-5ms, 5-10ms, 10-50ms, 50-100ms, 100-500ms, 500ms-1s, 1-5s, 5-10s, 10s+
	buckets := []time.Duration{
		time.Millisecond,
		5 * time.Millisecond,
		10 * time.Millisecond,
		50 * time.Millisecond,
		100 * time.Millisecond,
		500 * time.Millisecond,
		time.Second,
		5 * time.Second,
		10 * time.Second,
	}

	for i, bucket := range buckets {
		if latency <= bucket {
			return i
		}
	}
	return len(buckets) // Last bucket for >10s
}

// Reset clears all metrics (useful for testing)
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connectionMetrics = &ConnectionMetrics{
		RejectionReasons: make(map[string]int64),
	}
	m.messageMetrics = &MessageMetrics{
		MessageTypes: make(map[string]int64),
	}
	m.errorMetrics = &ErrorMetrics{
		ErrorsByType: make(map[string]int64),
	}
	m.performanceMetrics = &PerformanceMetrics{
		MinLatency:       time.Hour,
		LatencyHistogram: make([]int64, 10),
	}
	m.startTime = time.Now()
}