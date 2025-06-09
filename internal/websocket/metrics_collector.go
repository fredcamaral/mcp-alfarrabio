// Package websocket provides metrics collection for WebSocket connections
package websocket

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects and aggregates WebSocket connection metrics
type MetricsCollector struct {
	mu            sync.RWMutex
	config        *MetricsConfig
	connections   map[string]*ConnectionMetrics
	systemMetrics *SystemMetrics
	timeSeries    *TimeSeriesMetrics
	done          chan struct{}
}

// MetricsConfig configures metrics collection behavior
type MetricsConfig struct {
	CollectionInterval    time.Duration `json:"collection_interval" yaml:"collection_interval"`
	RetentionPeriod       time.Duration `json:"retention_period" yaml:"retention_period"`
	TimeSeriesResolution  time.Duration `json:"time_series_resolution" yaml:"time_series_resolution"`
	MaxDataPoints         int           `json:"max_data_points" yaml:"max_data_points"`
	EnableDetailedMetrics bool          `json:"enable_detailed_metrics" yaml:"enable_detailed_metrics"`
	TrackBandwidth        bool          `json:"track_bandwidth" yaml:"track_bandwidth"`
	TrackMessageTypes     bool          `json:"track_message_types" yaml:"track_message_types"`
}

// ConnectionMetrics tracks metrics for a single connection
type ConnectionMetrics struct {
	mu           sync.RWMutex
	ID           string
	StartTime    time.Time
	LastActivity time.Time

	// Message metrics
	MessagesReceived int64            `json:"messages_received"`
	MessagesSent     int64            `json:"messages_sent"`
	BytesReceived    int64            `json:"bytes_received"`
	BytesSent        int64            `json:"bytes_sent"`
	MessageTypes     map[string]int64 `json:"message_types"`

	// Performance metrics
	AverageLatency time.Duration   `json:"average_latency"`
	MinLatency     time.Duration   `json:"min_latency"`
	MaxLatency     time.Duration   `json:"max_latency"`
	LatencySamples []time.Duration `json:"-"`

	// Bandwidth metrics
	InboundBandwidth      float64 `json:"inbound_bandwidth_bps"`
	OutboundBandwidth     float64 `json:"outbound_bandwidth_bps"`
	PeakInboundBandwidth  float64 `json:"peak_inbound_bandwidth_bps"`
	PeakOutboundBandwidth float64 `json:"peak_outbound_bandwidth_bps"`

	// Error metrics
	Errors        int64     `json:"errors"`
	Reconnections int64     `json:"reconnections"`
	LastError     string    `json:"last_error,omitempty"`
	LastErrorTime time.Time `json:"last_error_time,omitempty"`

	// Connection quality
	QualityScore float64 `json:"quality_score"`
	Stability    float64 `json:"stability"`

	// Time series data
	hourlyStats    [24]*HourlyStats
	minutelyStats  [60]*MinutelyStats
	lastStatUpdate time.Time
}

// SystemMetrics tracks overall system metrics
type SystemMetrics struct {
	mu                sync.RWMutex
	TotalConnections  int64         `json:"total_connections"`
	ActiveConnections int64         `json:"active_connections"`
	TotalMessages     int64         `json:"total_messages"`
	TotalBytes        int64         `json:"total_bytes"`
	TotalErrors       int64         `json:"total_errors"`
	AverageLatency    time.Duration `json:"average_latency"`
	SystemThroughput  float64       `json:"system_throughput_msg_per_sec"`
	SystemBandwidth   float64       `json:"system_bandwidth_bps"`
	UptimeSeconds     int64         `json:"uptime_seconds"`
	StartTime         time.Time     `json:"start_time"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// TimeSeriesMetrics stores time-series data
type TimeSeriesMetrics struct {
	mu             sync.RWMutex
	dataPoints     []*TimeSeriesPoint
	maxDataPoints  int
	resolution     time.Duration
	lastCollection time.Time
}

// TimeSeriesPoint represents a single time series data point
type TimeSeriesPoint struct {
	Timestamp         time.Time     `json:"timestamp"`
	ActiveConnections int           `json:"active_connections"`
	MessagesPerSecond float64       `json:"messages_per_second"`
	BytesPerSecond    float64       `json:"bytes_per_second"`
	AverageLatency    time.Duration `json:"average_latency"`
	ErrorRate         float64       `json:"error_rate"`
}

// HourlyStats tracks hourly statistics
type HourlyStats struct {
	Hour           int           `json:"hour"`
	Messages       int64         `json:"messages"`
	Bytes          int64         `json:"bytes"`
	Errors         int64         `json:"errors"`
	AverageLatency time.Duration `json:"average_latency"`
	Timestamp      time.Time     `json:"timestamp"`
}

// MinutelyStats tracks minutely statistics
type MinutelyStats struct {
	Minute         int           `json:"minute"`
	Messages       int64         `json:"messages"`
	Bytes          int64         `json:"bytes"`
	Errors         int64         `json:"errors"`
	AverageLatency time.Duration `json:"average_latency"`
	Timestamp      time.Time     `json:"timestamp"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config *MetricsConfig) *MetricsCollector {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	mc := &MetricsCollector{
		config:      config,
		connections: make(map[string]*ConnectionMetrics),
		systemMetrics: &SystemMetrics{
			StartTime: time.Now(),
		},
		timeSeries: &TimeSeriesMetrics{
			dataPoints:    make([]*TimeSeriesPoint, 0, config.MaxDataPoints),
			maxDataPoints: config.MaxDataPoints,
			resolution:    config.TimeSeriesResolution,
		},
		done: make(chan struct{}),
	}

	// Start collection routines
	go mc.collectionRoutine()
	go mc.timeSeriesRoutine()

	return mc
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		CollectionInterval:    30 * time.Second,
		RetentionPeriod:       24 * time.Hour,
		TimeSeriesResolution:  time.Minute,
		MaxDataPoints:         1440, // 24 hours at 1-minute resolution
		EnableDetailedMetrics: true,
		TrackBandwidth:        true,
		TrackMessageTypes:     true,
	}
}

// RegisterConnection registers a connection for metrics collection
func (mc *MetricsCollector) RegisterConnection(id string) error {
	if id == "" {
		return fmt.Errorf("connection ID cannot be empty")
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics := &ConnectionMetrics{
		ID:            id,
		StartTime:     time.Now(),
		LastActivity:  time.Now(),
		MessageTypes:  make(map[string]int64),
		hourlyStats:   [24]*HourlyStats{},
		minutelyStats: [60]*MinutelyStats{},
	}

	// Initialize stats arrays
	for i := 0; i < 24; i++ {
		metrics.hourlyStats[i] = &HourlyStats{Hour: i}
	}
	for i := 0; i < 60; i++ {
		metrics.minutelyStats[i] = &MinutelyStats{Minute: i}
	}

	mc.connections[id] = metrics
	atomic.AddInt64(&mc.systemMetrics.TotalConnections, 1)
	atomic.AddInt64(&mc.systemMetrics.ActiveConnections, 1)

	log.Printf("Registered connection %s for metrics collection", id)
	return nil
}

// UnregisterConnection removes a connection from metrics collection
func (mc *MetricsCollector) UnregisterConnection(id string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.connections[id]; exists {
		delete(mc.connections, id)
		atomic.AddInt64(&mc.systemMetrics.ActiveConnections, -1)
		log.Printf("Unregistered connection %s from metrics collection", id)
	}
}

// RecordMessage records a message for metrics tracking
func (mc *MetricsCollector) RecordMessage(connectionID string, messageType string, size int, direction MessageDirection, latency time.Duration) {
	mc.mu.RLock()
	conn, exists := mc.connections[connectionID]
	mc.mu.RUnlock()

	if !exists {
		return
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.LastActivity = time.Now()

	// Update message counts
	if direction == DirectionInbound {
		atomic.AddInt64(&conn.MessagesReceived, 1)
		atomic.AddInt64(&conn.BytesReceived, int64(size))
	} else {
		atomic.AddInt64(&conn.MessagesSent, 1)
		atomic.AddInt64(&conn.BytesSent, int64(size))
	}

	// Track message types
	if mc.config.TrackMessageTypes && messageType != "" {
		conn.MessageTypes[messageType]++
	}

	// Update latency metrics
	if latency > 0 {
		mc.updateLatencyMetrics(conn, latency)
	}

	// Update system metrics
	atomic.AddInt64(&mc.systemMetrics.TotalMessages, 1)
	atomic.AddInt64(&mc.systemMetrics.TotalBytes, int64(size))

	// Update time-based stats
	mc.updateTimeBasedStats(conn, size, latency)
}

// RecordError records an error for metrics tracking
func (mc *MetricsCollector) RecordError(connectionID string, err error) {
	mc.mu.RLock()
	conn, exists := mc.connections[connectionID]
	mc.mu.RUnlock()

	if !exists {
		return
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	atomic.AddInt64(&conn.Errors, 1)
	atomic.AddInt64(&mc.systemMetrics.TotalErrors, 1)

	if err != nil {
		conn.LastError = err.Error()
		conn.LastErrorTime = time.Now()
	}

	// Update quality score based on errors
	mc.updateQualityScore(conn)
}

// RecordReconnection records a reconnection event
func (mc *MetricsCollector) RecordReconnection(connectionID string) {
	mc.mu.RLock()
	conn, exists := mc.connections[connectionID]
	mc.mu.RUnlock()

	if !exists {
		return
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	atomic.AddInt64(&conn.Reconnections, 1)
}

// MessageDirection represents message direction
type MessageDirection int

const (
	DirectionInbound MessageDirection = iota
	DirectionOutbound
)

// updateLatencyMetrics updates latency statistics
func (mc *MetricsCollector) updateLatencyMetrics(conn *ConnectionMetrics, latency time.Duration) {
	// Add to samples
	conn.LatencySamples = append(conn.LatencySamples, latency)
	if len(conn.LatencySamples) > 1000 {
		conn.LatencySamples = conn.LatencySamples[1:]
	}

	// Update min/max
	if conn.MinLatency == 0 || latency < conn.MinLatency {
		conn.MinLatency = latency
	}
	if latency > conn.MaxLatency {
		conn.MaxLatency = latency
	}

	// Calculate average
	if conn.AverageLatency == 0 {
		conn.AverageLatency = latency
	} else {
		conn.AverageLatency = (conn.AverageLatency + latency) / 2
	}
}

// updateTimeBasedStats updates hourly and minutely statistics
func (mc *MetricsCollector) updateTimeBasedStats(conn *ConnectionMetrics, size int, latency time.Duration) {
	now := time.Now()
	hour := now.Hour()
	minute := now.Minute()

	// Update hourly stats
	hourlyStats := conn.hourlyStats[hour]
	hourlyStats.Messages++
	hourlyStats.Bytes += int64(size)
	hourlyStats.Timestamp = now
	if latency > 0 {
		if hourlyStats.AverageLatency == 0 {
			hourlyStats.AverageLatency = latency
		} else {
			hourlyStats.AverageLatency = (hourlyStats.AverageLatency + latency) / 2
		}
	}

	// Update minutely stats
	minutelyStats := conn.minutelyStats[minute]
	minutelyStats.Messages++
	minutelyStats.Bytes += int64(size)
	minutelyStats.Timestamp = now
	if latency > 0 {
		if minutelyStats.AverageLatency == 0 {
			minutelyStats.AverageLatency = latency
		} else {
			minutelyStats.AverageLatency = (minutelyStats.AverageLatency + latency) / 2
		}
	}
}

// updateQualityScore updates connection quality score
func (mc *MetricsCollector) updateQualityScore(conn *ConnectionMetrics) {
	totalMessages := conn.MessagesReceived + conn.MessagesSent
	if totalMessages == 0 {
		conn.QualityScore = 1.0
		return
	}

	// Base score on error rate
	errorRate := float64(conn.Errors) / float64(totalMessages)
	baseScore := 1.0 - errorRate

	// Adjust for latency
	latencyPenalty := 0.0
	if conn.AverageLatency > 0 {
		switch {
		case conn.AverageLatency > time.Second:
			latencyPenalty = 0.3
		case conn.AverageLatency > 500*time.Millisecond:
			latencyPenalty = 0.2
		case conn.AverageLatency > 200*time.Millisecond:
			latencyPenalty = 0.1
		}
	}

	// Adjust for reconnections
	reconnectionPenalty := float64(conn.Reconnections) * 0.1

	conn.QualityScore = baseScore - latencyPenalty - reconnectionPenalty
	if conn.QualityScore < 0 {
		conn.QualityScore = 0
	}

	// Calculate stability (time connected / total time)
	uptime := time.Since(conn.StartTime)
	if uptime > 0 {
		conn.Stability = 1.0 - (float64(conn.Reconnections) / uptime.Hours())
		if conn.Stability < 0 {
			conn.Stability = 0
		}
	}
}

// collectionRoutine periodically collects and updates metrics
func (mc *MetricsCollector) collectionRoutine() {
	ticker := time.NewTicker(mc.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.updateSystemMetrics()
			mc.calculateBandwidth()
		case <-mc.done:
			return
		}
	}
}

// updateSystemMetrics updates system-wide metrics
func (mc *MetricsCollector) updateSystemMetrics() {
	mc.systemMetrics.mu.Lock()
	defer mc.systemMetrics.mu.Unlock()

	mc.systemMetrics.UptimeSeconds = int64(time.Since(mc.systemMetrics.StartTime).Seconds())
	mc.systemMetrics.LastUpdated = time.Now()

	// Calculate average latency across all connections
	mc.mu.RLock()
	var totalLatency time.Duration
	var connectionCount int

	for _, conn := range mc.connections {
		conn.mu.RLock()
		if conn.AverageLatency > 0 {
			totalLatency += conn.AverageLatency
			connectionCount++
		}
		conn.mu.RUnlock()
	}
	mc.mu.RUnlock()

	if connectionCount > 0 {
		mc.systemMetrics.AverageLatency = totalLatency / time.Duration(connectionCount)
	}

	// Calculate throughput (messages per second)
	if mc.systemMetrics.UptimeSeconds > 0 {
		mc.systemMetrics.SystemThroughput = float64(mc.systemMetrics.TotalMessages) / float64(mc.systemMetrics.UptimeSeconds)
		mc.systemMetrics.SystemBandwidth = float64(mc.systemMetrics.TotalBytes*8) / float64(mc.systemMetrics.UptimeSeconds) // bits per second
	}
}

// calculateBandwidth calculates bandwidth metrics for connections
func (mc *MetricsCollector) calculateBandwidth() {
	if !mc.config.TrackBandwidth {
		return
	}

	mc.mu.RLock()
	defer mc.mu.RUnlock()

	for _, conn := range mc.connections {
		conn.mu.Lock()
		uptime := time.Since(conn.StartTime).Seconds()
		if uptime > 0 {
			conn.InboundBandwidth = float64(conn.BytesReceived*8) / uptime
			conn.OutboundBandwidth = float64(conn.BytesSent*8) / uptime

			// Update peak bandwidth (simplified calculation)
			if conn.InboundBandwidth > conn.PeakInboundBandwidth {
				conn.PeakInboundBandwidth = conn.InboundBandwidth
			}
			if conn.OutboundBandwidth > conn.PeakOutboundBandwidth {
				conn.PeakOutboundBandwidth = conn.OutboundBandwidth
			}
		}
		conn.mu.Unlock()
	}
}

// timeSeriesRoutine collects time series data
func (mc *MetricsCollector) timeSeriesRoutine() {
	ticker := time.NewTicker(mc.timeSeries.resolution)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.collectTimeSeriesData()
		case <-mc.done:
			return
		}
	}
}

// collectTimeSeriesData collects a time series data point
func (mc *MetricsCollector) collectTimeSeriesData() {
	mc.timeSeries.mu.Lock()
	defer mc.timeSeries.mu.Unlock()

	now := time.Now()

	// Calculate current metrics
	activeConnections := int(atomic.LoadInt64(&mc.systemMetrics.ActiveConnections))

	var messagesPerSecond, bytesPerSecond, errorRate float64
	var avgLatency time.Duration

	if mc.timeSeries.lastCollection.IsZero() {
		mc.timeSeries.lastCollection = now
		return
	}

	timeDiff := now.Sub(mc.timeSeries.lastCollection).Seconds()
	if timeDiff > 0 {
		totalMessages := atomic.LoadInt64(&mc.systemMetrics.TotalMessages)
		totalBytes := atomic.LoadInt64(&mc.systemMetrics.TotalBytes)
		totalErrors := atomic.LoadInt64(&mc.systemMetrics.TotalErrors)

		// Calculate rates (simplified - in real implementation you'd track deltas)
		messagesPerSecond = float64(totalMessages) / timeDiff
		bytesPerSecond = float64(totalBytes) / timeDiff

		if totalMessages > 0 {
			errorRate = float64(totalErrors) / float64(totalMessages)
		}

		mc.systemMetrics.mu.RLock()
		avgLatency = mc.systemMetrics.AverageLatency
		mc.systemMetrics.mu.RUnlock()
	}

	// Create data point
	point := &TimeSeriesPoint{
		Timestamp:         now,
		ActiveConnections: activeConnections,
		MessagesPerSecond: messagesPerSecond,
		BytesPerSecond:    bytesPerSecond,
		AverageLatency:    avgLatency,
		ErrorRate:         errorRate,
	}

	// Add to time series
	mc.timeSeries.dataPoints = append(mc.timeSeries.dataPoints, point)

	// Trim to max data points
	if len(mc.timeSeries.dataPoints) > mc.timeSeries.maxDataPoints {
		mc.timeSeries.dataPoints = mc.timeSeries.dataPoints[1:]
	}

	mc.timeSeries.lastCollection = now
}

// GetConnectionMetrics returns metrics for a specific connection
func (mc *MetricsCollector) GetConnectionMetrics(id string) (*ConnectionMetrics, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	conn, exists := mc.connections[id]
	if !exists {
		return nil, fmt.Errorf("connection %s not found", id)
	}

	// Return a copy to avoid race conditions
	conn.mu.RLock()
	defer conn.mu.RUnlock()

	copy := &ConnectionMetrics{
		ID:                    conn.ID,
		StartTime:             conn.StartTime,
		LastActivity:          conn.LastActivity,
		MessagesReceived:      conn.MessagesReceived,
		MessagesSent:          conn.MessagesSent,
		BytesReceived:         conn.BytesReceived,
		BytesSent:             conn.BytesSent,
		AverageLatency:        conn.AverageLatency,
		MinLatency:            conn.MinLatency,
		MaxLatency:            conn.MaxLatency,
		InboundBandwidth:      conn.InboundBandwidth,
		OutboundBandwidth:     conn.OutboundBandwidth,
		PeakInboundBandwidth:  conn.PeakInboundBandwidth,
		PeakOutboundBandwidth: conn.PeakOutboundBandwidth,
		Errors:                conn.Errors,
		Reconnections:         conn.Reconnections,
		LastError:             conn.LastError,
		LastErrorTime:         conn.LastErrorTime,
		QualityScore:          conn.QualityScore,
		Stability:             conn.Stability,
		MessageTypes:          make(map[string]int64),
	}

	// Copy message types map
	for k, v := range conn.MessageTypes {
		copy.MessageTypes[k] = v
	}

	return copy, nil
}

// GetSystemMetrics returns system-wide metrics
func (mc *MetricsCollector) GetSystemMetrics() *SystemMetrics {
	mc.systemMetrics.mu.RLock()
	defer mc.systemMetrics.mu.RUnlock()

	return &SystemMetrics{
		TotalConnections:  atomic.LoadInt64(&mc.systemMetrics.TotalConnections),
		ActiveConnections: atomic.LoadInt64(&mc.systemMetrics.ActiveConnections),
		TotalMessages:     atomic.LoadInt64(&mc.systemMetrics.TotalMessages),
		TotalBytes:        atomic.LoadInt64(&mc.systemMetrics.TotalBytes),
		TotalErrors:       atomic.LoadInt64(&mc.systemMetrics.TotalErrors),
		AverageLatency:    mc.systemMetrics.AverageLatency,
		SystemThroughput:  mc.systemMetrics.SystemThroughput,
		SystemBandwidth:   mc.systemMetrics.SystemBandwidth,
		UptimeSeconds:     mc.systemMetrics.UptimeSeconds,
		StartTime:         mc.systemMetrics.StartTime,
		LastUpdated:       mc.systemMetrics.LastUpdated,
	}
}

// GetTimeSeriesData returns time series data
func (mc *MetricsCollector) GetTimeSeriesData(since time.Time) []*TimeSeriesPoint {
	mc.timeSeries.mu.RLock()
	defer mc.timeSeries.mu.RUnlock()

	var filteredPoints []*TimeSeriesPoint
	for _, point := range mc.timeSeries.dataPoints {
		if point.Timestamp.After(since) {
			filteredPoints = append(filteredPoints, point)
		}
	}

	return filteredPoints
}

// GetAllConnectionMetrics returns metrics for all connections
func (mc *MetricsCollector) GetAllConnectionMetrics() map[string]*ConnectionMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*ConnectionMetrics)
	for id := range mc.connections {
		if metrics, err := mc.GetConnectionMetrics(id); err == nil {
			result[id] = metrics
		}
	}

	return result
}

// Close stops the metrics collector
func (mc *MetricsCollector) Close() error {
	close(mc.done)
	return nil
}
