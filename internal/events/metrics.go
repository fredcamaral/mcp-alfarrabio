// Package events provides event metrics and monitoring capabilities
package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// MetricsCollector collects and aggregates event metrics
type MetricsCollector struct {
	metrics         *EventMetrics
	config          *MetricsConfig
	eventStats      map[EventType]*EventTypeStats
	sourceStats     map[string]*SourceStats
	subscriberStats map[string]*SubscriberStats
	timeWindows     map[string]*TimeWindowStats
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	running         bool
	wg              sync.WaitGroup
}

// EventMetrics represents comprehensive event system metrics
type EventMetrics struct {
	// Overall statistics
	TotalEvents     int64         `json:"total_events"`
	EventsPerSecond float64       `json:"events_per_second"`
	AverageLatency  time.Duration `json:"average_latency"`
	P95Latency      time.Duration `json:"p95_latency"`
	P99Latency      time.Duration `json:"p99_latency"`
	ErrorRate       float64       `json:"error_rate"`

	// Bus metrics
	ActiveSubscribers    int   `json:"active_subscribers"`
	SuccessfulDeliveries int64 `json:"successful_deliveries"`
	FailedDeliveries     int64 `json:"failed_deliveries"`
	DroppedEvents        int64 `json:"dropped_events"`
	DuplicateEvents      int64 `json:"duplicate_events"`

	// Performance metrics
	MemoryUsage       int64   `json:"memory_usage_bytes"`
	CPUUsage          float64 `json:"cpu_usage_percent"`
	QueueDepth        int     `json:"queue_depth"`
	BufferUtilization float64 `json:"buffer_utilization_percent"`

	// Time-based metrics
	LastEventTime time.Time `json:"last_event_time"`
	UptimeSeconds int64     `json:"uptime_seconds"`
	StartTime     time.Time `json:"start_time"`

	mu sync.RWMutex
}

// EventTypeStats tracks metrics for specific event types
type EventTypeStats struct {
	EventType      EventType     `json:"event_type"`
	Count          int64         `json:"count"`
	AverageSize    int           `json:"average_size_bytes"`
	AverageLatency time.Duration `json:"average_latency"`
	ErrorCount     int64         `json:"error_count"`
	LastSeen       time.Time     `json:"last_seen"`
	PeakRate       float64       `json:"peak_rate_per_second"`
	mu             sync.RWMutex
}

// SourceStats tracks metrics for event sources
type SourceStats struct {
	Source         string        `json:"source"`
	EventCount     int64         `json:"event_count"`
	AverageLatency time.Duration `json:"average_latency"`
	ErrorCount     int64         `json:"error_count"`
	LastActivity   time.Time     `json:"last_activity"`
	Reliability    float64       `json:"reliability_percent"`
	mu             sync.RWMutex
}

// SubscriberStats tracks metrics for subscribers
type SubscriberStats struct {
	SubscriberID          string        `json:"subscriber_id"`
	EventsReceived        int64         `json:"events_received"`
	EventsProcessed       int64         `json:"events_processed"`
	EventsFailed          int64         `json:"events_failed"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	LastActivity          time.Time     `json:"last_activity"`
	HealthScore           float64       `json:"health_score"`
	mu                    sync.RWMutex
}

// TimeWindowStats tracks metrics within time windows
type TimeWindowStats struct {
	Window      string              `json:"window"`
	StartTime   time.Time           `json:"start_time"`
	EndTime     time.Time           `json:"end_time"`
	EventCount  int64               `json:"event_count"`
	PeakRate    float64             `json:"peak_rate"`
	AverageRate float64             `json:"average_rate"`
	ErrorCount  int64               `json:"error_count"`
	EventTypes  map[EventType]int64 `json:"event_types"`
	mu          sync.RWMutex
}

// MetricsConfig configures the metrics collector
type MetricsConfig struct {
	CollectionInterval       time.Duration    `json:"collection_interval"`
	RetentionPeriod          time.Duration    `json:"retention_period"`
	TimeWindowSizes          []time.Duration  `json:"time_window_sizes"`
	EnableDetailedStats      bool             `json:"enable_detailed_stats"`
	EnablePerformanceMetrics bool             `json:"enable_performance_metrics"`
	MaxEventTypes            int              `json:"max_event_types"`
	MaxSources               int              `json:"max_sources"`
	MaxSubscribers           int              `json:"max_subscribers"`
	EnableAlerting           bool             `json:"enable_alerting"`
	AlertThresholds          *AlertThresholds `json:"alert_thresholds"`
}

// AlertThresholds defines thresholds for alerting
type AlertThresholds struct {
	MaxErrorRate   float64       `json:"max_error_rate"`
	MaxLatency     time.Duration `json:"max_latency"`
	MinThroughput  float64       `json:"min_throughput"`
	MaxMemoryUsage int64         `json:"max_memory_usage_bytes"`
	MaxCPUUsage    float64       `json:"max_cpu_usage_percent"`
	MaxQueueDepth  int           `json:"max_queue_depth"`
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	Timestamp   time.Time                     `json:"timestamp"`
	Overall     *EventMetrics                 `json:"overall"`
	EventTypes  map[EventType]*EventTypeStats `json:"event_types"`
	Sources     map[string]*SourceStats       `json:"sources"`
	Subscribers map[string]*SubscriberStats   `json:"subscribers"`
	TimeWindows map[string]*TimeWindowStats   `json:"time_windows"`
}

// Alert represents a metrics alert
type Alert struct {
	ID          string        `json:"id"`
	Type        AlertType     `json:"type"`
	Severity    AlertSeverity `json:"severity"`
	Message     string        `json:"message"`
	Threshold   interface{}   `json:"threshold"`
	ActualValue interface{}   `json:"actual_value"`
	Timestamp   time.Time     `json:"timestamp"`
	Resolved    bool          `json:"resolved"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
}

// AlertType defines types of alerts
type AlertType string

const (
	AlertTypeErrorRate        AlertType = "error_rate"
	AlertTypeLatency          AlertType = "latency"
	AlertTypeThroughput       AlertType = "throughput"
	AlertTypeMemoryUsage      AlertType = "memory_usage"
	AlertTypeCPUUsage         AlertType = "cpu_usage"
	AlertTypeQueueDepth       AlertType = "queue_depth"
	AlertTypeSubscriberHealth AlertType = "subscriber_health"
)

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		CollectionInterval:       10 * time.Second,
		RetentionPeriod:          24 * time.Hour,
		TimeWindowSizes:          []time.Duration{time.Minute, 5 * time.Minute, time.Hour},
		EnableDetailedStats:      true,
		EnablePerformanceMetrics: true,
		MaxEventTypes:            100,
		MaxSources:               50,
		MaxSubscribers:           200,
		EnableAlerting:           true,
		AlertThresholds: &AlertThresholds{
			MaxErrorRate:   5.0, // 5%
			MaxLatency:     500 * time.Millisecond,
			MinThroughput:  10.0,               // 10 events/sec
			MaxMemoryUsage: 1024 * 1024 * 1024, // 1GB
			MaxCPUUsage:    80.0,               // 80%
			MaxQueueDepth:  10000,
		},
	}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config *MetricsConfig) *MetricsCollector {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &MetricsCollector{
		metrics:         &EventMetrics{StartTime: time.Now()},
		config:          config,
		eventStats:      make(map[EventType]*EventTypeStats),
		sourceStats:     make(map[string]*SourceStats),
		subscriberStats: make(map[string]*SubscriberStats),
		timeWindows:     make(map[string]*TimeWindowStats),
		ctx:             ctx,
		cancel:          cancel,
		running:         false,
	}
}

// Start starts the metrics collector
func (mc *MetricsCollector) Start() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.running {
		return errors.New("metrics collector already running")
	}

	log.Println("Starting event metrics collector...")

	// Initialize time windows
	mc.initializeTimeWindows()

	// Start collection routine
	mc.wg.Add(1)
	go mc.collectionRoutine()

	// Start cleanup routine
	mc.wg.Add(1)
	go mc.cleanupRoutine()

	mc.running = true
	log.Printf("Metrics collector started with collection interval: %v", mc.config.CollectionInterval)

	return nil
}

// Stop stops the metrics collector
func (mc *MetricsCollector) Stop() error {
	mc.mu.Lock()
	if !mc.running {
		mc.mu.Unlock()
		return errors.New("metrics collector not running")
	}
	mc.running = false
	mc.mu.Unlock()

	log.Println("Stopping event metrics collector...")

	// Cancel context to signal routines to stop
	mc.cancel()

	// Wait for all routines to finish
	mc.wg.Wait()

	log.Println("Event metrics collector stopped")
	return nil
}

// IsRunning returns whether the metrics collector is running
func (mc *MetricsCollector) IsRunning() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.running
}

// RecordEvent records metrics for an event
func (mc *MetricsCollector) RecordEvent(event *Event, latency time.Duration, success bool) {
	if !mc.IsRunning() {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Update overall metrics
	mc.metrics.mu.Lock()
	mc.metrics.TotalEvents++
	mc.metrics.LastEventTime = time.Now()

	// Update latency (simple moving average)
	if mc.metrics.AverageLatency == 0 {
		mc.metrics.AverageLatency = latency
	} else {
		mc.metrics.AverageLatency = time.Duration(
			int64(mc.metrics.AverageLatency)*9/10 + int64(latency)/10,
		)
	}

	if !success {
		mc.metrics.FailedDeliveries++
	} else {
		mc.metrics.SuccessfulDeliveries++
	}

	mc.metrics.mu.Unlock()

	// Update event type stats
	if mc.config.EnableDetailedStats {
		mc.updateEventTypeStats(event, latency, success)
		mc.updateSourceStats(event, latency, success)
		mc.updateTimeWindowStats(event)
	}
}

// RecordSubscriberActivity records subscriber activity metrics
func (mc *MetricsCollector) RecordSubscriberActivity(subscriberID string, eventsReceived, eventsProcessed, eventsFailed int64, processingTime time.Duration) {
	if !mc.IsRunning() {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	stats, exists := mc.subscriberStats[subscriberID]
	if !exists {
		if len(mc.subscriberStats) >= mc.config.MaxSubscribers {
			return // Don't track more subscribers
		}
		stats = &SubscriberStats{
			SubscriberID: subscriberID,
		}
		mc.subscriberStats[subscriberID] = stats
	}

	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.EventsReceived += eventsReceived
	stats.EventsProcessed += eventsProcessed
	stats.EventsFailed += eventsFailed
	stats.LastActivity = time.Now()

	// Update average processing time
	if stats.AverageProcessingTime == 0 {
		stats.AverageProcessingTime = processingTime
	} else {
		stats.AverageProcessingTime = time.Duration(
			int64(stats.AverageProcessingTime)*8/10 + int64(processingTime)*2/10,
		)
	}

	// Calculate health score
	if stats.EventsReceived > 0 {
		successRate := float64(stats.EventsProcessed) / float64(stats.EventsReceived)
		stats.HealthScore = successRate * 100
	}
}

// GetMetrics returns current metrics snapshot
func (mc *MetricsCollector) GetMetrics() *EventMetrics {
	mc.metrics.mu.RLock()
	defer mc.metrics.mu.RUnlock()

	// Calculate uptime
	uptime := time.Since(mc.metrics.StartTime)

	return &EventMetrics{
		TotalEvents:          mc.metrics.TotalEvents,
		EventsPerSecond:      mc.metrics.EventsPerSecond,
		AverageLatency:       mc.metrics.AverageLatency,
		P95Latency:           mc.metrics.P95Latency,
		P99Latency:           mc.metrics.P99Latency,
		ErrorRate:            mc.metrics.ErrorRate,
		ActiveSubscribers:    mc.metrics.ActiveSubscribers,
		SuccessfulDeliveries: mc.metrics.SuccessfulDeliveries,
		FailedDeliveries:     mc.metrics.FailedDeliveries,
		DroppedEvents:        mc.metrics.DroppedEvents,
		DuplicateEvents:      mc.metrics.DuplicateEvents,
		MemoryUsage:          mc.metrics.MemoryUsage,
		CPUUsage:             mc.metrics.CPUUsage,
		QueueDepth:           mc.metrics.QueueDepth,
		BufferUtilization:    mc.metrics.BufferUtilization,
		LastEventTime:        mc.metrics.LastEventTime,
		UptimeSeconds:        int64(uptime.Seconds()),
		StartTime:            mc.metrics.StartTime,
	}
}

// GetEventTypeStats returns event type statistics
func (mc *MetricsCollector) GetEventTypeStats() map[EventType]*EventTypeStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy
	result := make(map[EventType]*EventTypeStats)
	for eventType, stats := range mc.eventStats {
		stats.mu.RLock()
		result[eventType] = &EventTypeStats{
			EventType:      stats.EventType,
			Count:          stats.Count,
			AverageSize:    stats.AverageSize,
			AverageLatency: stats.AverageLatency,
			ErrorCount:     stats.ErrorCount,
			LastSeen:       stats.LastSeen,
			PeakRate:       stats.PeakRate,
		}
		stats.mu.RUnlock()
	}

	return result
}

// GetSourceStats returns source statistics
func (mc *MetricsCollector) GetSourceStats() map[string]*SourceStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy
	result := make(map[string]*SourceStats)
	for source, stats := range mc.sourceStats {
		stats.mu.RLock()
		result[source] = &SourceStats{
			Source:         stats.Source,
			EventCount:     stats.EventCount,
			AverageLatency: stats.AverageLatency,
			ErrorCount:     stats.ErrorCount,
			LastActivity:   stats.LastActivity,
			Reliability:    stats.Reliability,
		}
		stats.mu.RUnlock()
	}

	return result
}

// GetSubscriberStats returns subscriber statistics
func (mc *MetricsCollector) GetSubscriberStats() map[string]*SubscriberStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy
	result := make(map[string]*SubscriberStats)
	for subscriberID, stats := range mc.subscriberStats {
		stats.mu.RLock()
		result[subscriberID] = &SubscriberStats{
			SubscriberID:          stats.SubscriberID,
			EventsReceived:        stats.EventsReceived,
			EventsProcessed:       stats.EventsProcessed,
			EventsFailed:          stats.EventsFailed,
			AverageProcessingTime: stats.AverageProcessingTime,
			LastActivity:          stats.LastActivity,
			HealthScore:           stats.HealthScore,
		}
		stats.mu.RUnlock()
	}

	return result
}

// GetTimeWindowStats returns time window statistics
func (mc *MetricsCollector) GetTimeWindowStats() map[string]*TimeWindowStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy
	result := make(map[string]*TimeWindowStats)
	for window, stats := range mc.timeWindows {
		stats.mu.RLock()
		eventTypesCopy := make(map[EventType]int64)
		for eventType, count := range stats.EventTypes {
			eventTypesCopy[eventType] = count
		}
		result[window] = &TimeWindowStats{
			Window:      stats.Window,
			StartTime:   stats.StartTime,
			EndTime:     stats.EndTime,
			EventCount:  stats.EventCount,
			PeakRate:    stats.PeakRate,
			AverageRate: stats.AverageRate,
			ErrorCount:  stats.ErrorCount,
			EventTypes:  eventTypesCopy,
		}
		stats.mu.RUnlock()
	}

	return result
}

// GetSnapshot returns a complete metrics snapshot
func (mc *MetricsCollector) GetSnapshot() *MetricsSnapshot {
	return &MetricsSnapshot{
		Timestamp:   time.Now(),
		Overall:     mc.GetMetrics(),
		EventTypes:  mc.GetEventTypeStats(),
		Sources:     mc.GetSourceStats(),
		Subscribers: mc.GetSubscriberStats(),
		TimeWindows: mc.GetTimeWindowStats(),
	}
}

// ExportMetrics exports metrics in JSON format
func (mc *MetricsCollector) ExportMetrics() ([]byte, error) {
	snapshot := mc.GetSnapshot()
	return json.MarshalIndent(snapshot, "", "  ")
}

// updateEventTypeStats updates statistics for a specific event type
func (mc *MetricsCollector) updateEventTypeStats(event *Event, latency time.Duration, success bool) {
	stats, exists := mc.eventStats[event.Type]
	if !exists {
		if len(mc.eventStats) >= mc.config.MaxEventTypes {
			return // Don't track more event types
		}
		stats = &EventTypeStats{
			EventType: event.Type,
		}
		mc.eventStats[event.Type] = stats
	}

	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.Count++
	stats.LastSeen = time.Now()

	// Update average latency
	if stats.AverageLatency == 0 {
		stats.AverageLatency = latency
	} else {
		stats.AverageLatency = time.Duration(
			int64(stats.AverageLatency)*9/10 + int64(latency)/10,
		)
	}

	// Update average size (rough estimation)
	eventSize := len(event.ID) + len(event.Action) + len(event.Source)
	if stats.AverageSize == 0 {
		stats.AverageSize = eventSize
	} else {
		stats.AverageSize = (stats.AverageSize*9 + eventSize) / 10
	}

	if !success {
		stats.ErrorCount++
	}
}

// updateSourceStats updates statistics for a specific source
func (mc *MetricsCollector) updateSourceStats(event *Event, latency time.Duration, success bool) {
	stats, exists := mc.sourceStats[event.Source]
	if !exists {
		if len(mc.sourceStats) >= mc.config.MaxSources {
			return // Don't track more sources
		}
		stats = &SourceStats{
			Source: event.Source,
		}
		mc.sourceStats[event.Source] = stats
	}

	stats.mu.Lock()
	defer stats.mu.Unlock()

	stats.EventCount++
	stats.LastActivity = time.Now()

	// Update average latency
	if stats.AverageLatency == 0 {
		stats.AverageLatency = latency
	} else {
		stats.AverageLatency = time.Duration(
			int64(stats.AverageLatency)*9/10 + int64(latency)/10,
		)
	}

	if !success {
		stats.ErrorCount++
	}

	// Calculate reliability
	if stats.EventCount > 0 {
		successCount := stats.EventCount - stats.ErrorCount
		stats.Reliability = float64(successCount) / float64(stats.EventCount) * 100
	}
}

// updateTimeWindowStats updates time window statistics
func (mc *MetricsCollector) updateTimeWindowStats(event *Event) {
	now := time.Now()

	for _, windowSize := range mc.config.TimeWindowSizes {
		windowKey := windowSize.String()
		stats, exists := mc.timeWindows[windowKey]

		if !exists || now.Sub(stats.StartTime) >= windowSize {
			// Create new window
			stats = &TimeWindowStats{
				Window:     windowKey,
				StartTime:  now.Truncate(windowSize),
				EndTime:    now.Truncate(windowSize).Add(windowSize),
				EventTypes: make(map[EventType]int64),
			}
			mc.timeWindows[windowKey] = stats
		}

		stats.mu.Lock()
		stats.EventCount++
		stats.EventTypes[event.Type]++

		// Calculate rates
		elapsed := now.Sub(stats.StartTime).Seconds()
		if elapsed > 0 {
			currentRate := float64(stats.EventCount) / elapsed
			if currentRate > stats.PeakRate {
				stats.PeakRate = currentRate
			}
			stats.AverageRate = float64(stats.EventCount) / elapsed
		}
		stats.mu.Unlock()
	}
}

// initializeTimeWindows initializes time window tracking
func (mc *MetricsCollector) initializeTimeWindows() {
	now := time.Now()

	for _, windowSize := range mc.config.TimeWindowSizes {
		windowKey := windowSize.String()
		mc.timeWindows[windowKey] = &TimeWindowStats{
			Window:     windowKey,
			StartTime:  now.Truncate(windowSize),
			EndTime:    now.Truncate(windowSize).Add(windowSize),
			EventTypes: make(map[EventType]int64),
		}
	}
}

// collectionRoutine performs periodic metrics collection and calculation
func (mc *MetricsCollector) collectionRoutine() {
	defer mc.wg.Done()

	ticker := time.NewTicker(mc.config.CollectionInterval)
	defer ticker.Stop()

	lastEventCount := int64(0)
	lastCollectionTime := time.Now()

	for {
		select {
		case <-ticker.C:
			mc.performCollection(&lastEventCount, &lastCollectionTime)
		case <-mc.ctx.Done():
			return
		}
	}
}

// performCollection performs metrics collection and calculations
func (mc *MetricsCollector) performCollection(lastEventCount *int64, lastCollectionTime *time.Time) {
	now := time.Now()

	mc.metrics.mu.Lock()

	// Calculate events per second
	timeDiff := now.Sub(*lastCollectionTime).Seconds()
	eventDiff := mc.metrics.TotalEvents - *lastEventCount

	if timeDiff > 0 {
		mc.metrics.EventsPerSecond = float64(eventDiff) / timeDiff
	}

	// Calculate error rate
	totalDeliveries := mc.metrics.SuccessfulDeliveries + mc.metrics.FailedDeliveries
	if totalDeliveries > 0 {
		mc.metrics.ErrorRate = float64(mc.metrics.FailedDeliveries) / float64(totalDeliveries) * 100
	}

	// Update subscriber count
	mc.mu.RLock()
	mc.metrics.ActiveSubscribers = len(mc.subscriberStats)
	mc.mu.RUnlock()

	mc.metrics.mu.Unlock()

	// Check alerts if enabled
	if mc.config.EnableAlerting {
		mc.checkAlerts()
	}

	*lastEventCount = mc.metrics.TotalEvents
	*lastCollectionTime = now
}

// cleanupRoutine performs periodic cleanup of old metrics data
func (mc *MetricsCollector) cleanupRoutine() {
	defer mc.wg.Done()

	ticker := time.NewTicker(time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.performCleanup()
		case <-mc.ctx.Done():
			return
		}
	}
}

// performCleanup removes old metrics data
func (mc *MetricsCollector) performCleanup() {
	now := time.Now()
	cutoff := now.Add(-mc.config.RetentionPeriod)

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Clean up inactive subscribers
	for subscriberID, stats := range mc.subscriberStats {
		stats.mu.RLock()
		lastActivity := stats.LastActivity
		stats.mu.RUnlock()

		if lastActivity.Before(cutoff) {
			delete(mc.subscriberStats, subscriberID)
		}
	}

	// Clean up inactive sources
	for source, stats := range mc.sourceStats {
		stats.mu.RLock()
		lastActivity := stats.LastActivity
		stats.mu.RUnlock()

		if lastActivity.Before(cutoff) {
			delete(mc.sourceStats, source)
		}
	}

	// Clean up old event type stats
	for eventType, stats := range mc.eventStats {
		stats.mu.RLock()
		lastSeen := stats.LastSeen
		stats.mu.RUnlock()

		if lastSeen.Before(cutoff) {
			delete(mc.eventStats, eventType)
		}
	}

	log.Printf("Metrics cleanup completed")
}

// checkAlerts checks for alert conditions
func (mc *MetricsCollector) checkAlerts() {
	if mc.config.AlertThresholds == nil {
		return
	}

	metrics := mc.GetMetrics()
	thresholds := mc.config.AlertThresholds

	// Check error rate
	if metrics.ErrorRate > thresholds.MaxErrorRate {
		mc.triggerAlert(AlertTypeErrorRate, AlertSeverityError,
			fmt.Sprintf("Error rate %.2f%% exceeds threshold %.2f%%", metrics.ErrorRate, thresholds.MaxErrorRate),
			thresholds.MaxErrorRate, metrics.ErrorRate)
	}

	// Check latency
	if metrics.AverageLatency > thresholds.MaxLatency {
		mc.triggerAlert(AlertTypeLatency, AlertSeverityWarning,
			fmt.Sprintf("Average latency %v exceeds threshold %v", metrics.AverageLatency, thresholds.MaxLatency),
			thresholds.MaxLatency, metrics.AverageLatency)
	}

	// Check throughput
	if metrics.EventsPerSecond < thresholds.MinThroughput {
		mc.triggerAlert(AlertTypeThroughput, AlertSeverityWarning,
			fmt.Sprintf("Throughput %.2f events/sec below threshold %.2f", metrics.EventsPerSecond, thresholds.MinThroughput),
			thresholds.MinThroughput, metrics.EventsPerSecond)
	}

	// Check queue depth
	if metrics.QueueDepth > thresholds.MaxQueueDepth {
		mc.triggerAlert(AlertTypeQueueDepth, AlertSeverityCritical,
			fmt.Sprintf("Queue depth %d exceeds threshold %d", metrics.QueueDepth, thresholds.MaxQueueDepth),
			thresholds.MaxQueueDepth, metrics.QueueDepth)
	}
}

// triggerAlert triggers an alert
func (mc *MetricsCollector) triggerAlert(alertType AlertType, severity AlertSeverity, message string, threshold, actualValue interface{}) {
	alert := &Alert{
		ID:          fmt.Sprintf("alert_%d", time.Now().UnixNano()),
		Type:        alertType,
		Severity:    severity,
		Message:     message,
		Threshold:   threshold,
		ActualValue: actualValue,
		Timestamp:   time.Now(),
		Resolved:    false,
	}

	// In a real implementation, you would send this alert to an alerting system
	log.Printf("ALERT [%s/%s]: %s (threshold: %v, actual: %v)", severity, alertType, message, alert.Threshold, alert.ActualValue)
}

// GetHealthStatus returns the overall health status of the event system
func (mc *MetricsCollector) GetHealthStatus() map[string]interface{} {
	metrics := mc.GetMetrics()

	health := "healthy"
	if metrics.ErrorRate > 10 {
		health = "degraded"
	}
	if metrics.ErrorRate > 25 || metrics.QueueDepth > 10000 {
		health = "unhealthy"
	}

	return map[string]interface{}{
		"status":             health,
		"uptime_seconds":     metrics.UptimeSeconds,
		"total_events":       metrics.TotalEvents,
		"events_per_second":  metrics.EventsPerSecond,
		"error_rate":         metrics.ErrorRate,
		"average_latency":    metrics.AverageLatency.String(),
		"active_subscribers": metrics.ActiveSubscribers,
		"buffer_utilization": metrics.BufferUtilization,
		"last_event_time":    metrics.LastEventTime,
	}
}
