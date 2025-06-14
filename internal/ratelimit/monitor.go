// Package ratelimit provides rate limit monitoring and alerting capabilities
package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Monitor provides comprehensive rate limiting monitoring and alerting
type Monitor struct {
	mu      sync.RWMutex
	config  *Config
	metrics *Metrics
	alerts  chan *Alert
	limiter RateLimiter
	done    chan struct{}

	// Monitoring state
	lastCheck      time.Time
	alertsSent     map[string]time.Time
	performanceLog []PerformanceMetric
}

// RateLimiter interface for monitoring different limiter implementations
type RateLimiter interface {
	IsHealthy(ctx context.Context) error
	GetInfo(ctx context.Context) (map[string]interface{}, error)
	GetStats(ctx context.Context, key string) (map[string]interface{}, error)
}

// Metrics holds real-time rate limiting metrics
type Metrics struct {
	mu sync.RWMutex

	// Request metrics
	TotalRequests   int64 `json:"total_requests"`
	AllowedRequests int64 `json:"allowed_requests"`
	BlockedRequests int64 `json:"blocked_requests"`

	// Performance metrics
	AverageCheckTime time.Duration `json:"average_check_time"`
	MaxCheckTime     time.Duration `json:"max_check_time"`
	MinCheckTime     time.Duration `json:"min_check_time"`

	// Error metrics
	TotalErrors   int64 `json:"total_errors"`
	RedisErrors   int64 `json:"redis_errors"`
	FallbackUsage int64 `json:"fallback_usage"`

	// Rate metrics by endpoint
	EndpointMetrics map[string]*EndpointMetrics `json:"endpoint_metrics"`

	// Time series data (last hour)
	RequestsPerMinute []int64 `json:"requests_per_minute"`
	BlockedPerMinute  []int64 `json:"blocked_per_minute"`
	ErrorsPerMinute   []int64 `json:"errors_per_minute"`

	// Current state
	ActiveConnections int       `json:"active_connections"`
	HealthyLimiters   int       `json:"healthy_limiters"`
	LastUpdated       time.Time `json:"last_updated"`
}

// EndpointMetrics holds metrics for a specific endpoint
type EndpointMetrics struct {
	mu sync.RWMutex

	Endpoint        string        `json:"endpoint"`
	TotalRequests   int64         `json:"total_requests"`
	AllowedRequests int64         `json:"allowed_requests"`
	BlockedRequests int64         `json:"blocked_requests"`
	AverageLatency  time.Duration `json:"average_latency"`
	LastRequest     time.Time     `json:"last_request"`

	// Recent activity (sliding window)
	RecentRequests  []RequestRecord `json:"recent_requests"`
	UtilizationRate float64         `json:"utilization_rate"`
}

// RequestRecord represents a single rate limit check
type RequestRecord struct {
	Timestamp time.Time     `json:"timestamp"`
	Key       string        `json:"key"`
	Allowed   bool          `json:"allowed"`
	Latency   time.Duration `json:"latency"`
	Count     int           `json:"count"`
	Limit     int           `json:"limit"`
}

// Alert represents a rate limiting alert
type Alert struct {
	ID         string                 `json:"id"`
	Type       AlertType              `json:"type"`
	Severity   Severity               `json:"severity"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Endpoint   string                 `json:"endpoint,omitempty"`
	Key        string                 `json:"key,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// AlertType represents different types of alerts
type AlertType string

const (
	AlertTypeHighUsage    AlertType = "high_usage"
	AlertTypeRateLimited  AlertType = "rate_limited"
	AlertTypeRedisDown    AlertType = "redis_down"
	AlertTypeHighLatency  AlertType = "high_latency"
	AlertTypeConfigError  AlertType = "config_error"
	AlertTypeFallbackUsed AlertType = "fallback_used"
)

// Severity represents alert severity levels
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// PerformanceMetric tracks performance over time
type PerformanceMetric struct {
	Timestamp     time.Time     `json:"timestamp"`
	CheckDuration time.Duration `json:"check_duration"`
	RequestCount  int64         `json:"request_count"`
	ErrorCount    int64         `json:"error_count"`
	MemoryUsage   int64         `json:"memory_usage"`
	RedisLatency  time.Duration `json:"redis_latency"`
}

// NewMonitor creates a new rate limit monitor
func NewMonitor(config *Config, limiter RateLimiter) *Monitor {
	if config == nil {
		config = DefaultConfig()
	}

	monitor := &Monitor{
		config:         config,
		limiter:        limiter,
		metrics:        NewMetrics(),
		alerts:         make(chan *Alert, 100),
		done:           make(chan struct{}),
		lastCheck:      time.Now(),
		alertsSent:     make(map[string]time.Time),
		performanceLog: make([]PerformanceMetric, 0, 1000),
	}

	// Start monitoring routines if alerting is enabled
	if config.EnableAlerting {
		go monitor.alertingRoutine()
		go monitor.metricsCollectionRoutine()
		go monitor.performanceMonitoringRoutine()
	}

	return monitor
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		EndpointMetrics:   make(map[string]*EndpointMetrics),
		RequestsPerMinute: make([]int64, 60), // Last hour
		BlockedPerMinute:  make([]int64, 60),
		ErrorsPerMinute:   make([]int64, 60),
		LastUpdated:       time.Now(),
	}
}

// RecordRequest records a rate limiting request
func (m *Monitor) RecordRequest(endpoint, key string, result *LimitResult, duration time.Duration) {
	if !m.config.EnableMetrics {
		return
	}

	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	// Update global metrics
	m.metrics.TotalRequests++
	if result.Allowed {
		m.metrics.AllowedRequests++
	} else {
		m.metrics.BlockedRequests++
	}

	// Update timing metrics
	if duration > 0 {
		if m.metrics.AverageCheckTime == 0 {
			m.metrics.AverageCheckTime = duration
		} else {
			m.metrics.AverageCheckTime = (m.metrics.AverageCheckTime + duration) / 2
		}

		if duration > m.metrics.MaxCheckTime {
			m.metrics.MaxCheckTime = duration
		}

		if m.metrics.MinCheckTime == 0 || duration < m.metrics.MinCheckTime {
			m.metrics.MinCheckTime = duration
		}
	}

	// Update endpoint metrics
	endpointMetrics := m.getOrCreateEndpointMetrics(endpoint)
	endpointMetrics.mu.Lock()
	endpointMetrics.TotalRequests++
	if result.Allowed {
		endpointMetrics.AllowedRequests++
	} else {
		endpointMetrics.BlockedRequests++
	}
	endpointMetrics.LastRequest = time.Now()

	// Update average latency
	if duration > 0 {
		if endpointMetrics.AverageLatency == 0 {
			endpointMetrics.AverageLatency = duration
		} else {
			endpointMetrics.AverageLatency = (endpointMetrics.AverageLatency + duration) / 2
		}
	}

	// Add to recent requests (keep last 100)
	record := RequestRecord{
		Timestamp: time.Now(),
		Key:       key,
		Allowed:   result.Allowed,
		Latency:   duration,
		Count:     result.Count,
		Limit:     result.Limit,
	}

	endpointMetrics.RecentRequests = append(endpointMetrics.RecentRequests, record)
	if len(endpointMetrics.RecentRequests) > 100 {
		endpointMetrics.RecentRequests = endpointMetrics.RecentRequests[1:]
	}

	// Calculate utilization rate
	if result.Limit > 0 {
		endpointMetrics.UtilizationRate = float64(result.Count) / float64(result.Limit)
	}
	endpointMetrics.mu.Unlock()

	// Update time series data
	m.updateTimeSeriesMetrics(result.Allowed)
	m.metrics.LastUpdated = time.Now()

	// Check for alerts
	if m.config.EnableAlerting {
		m.checkForAlerts(endpoint, key, result, endpointMetrics)
	}
}

// RecordError records an error in rate limiting
func (m *Monitor) RecordError(errorType string, err error) {
	if !m.config.EnableMetrics {
		return
	}

	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.TotalErrors++

	switch errorType {
	case "redis":
		m.metrics.RedisErrors++
	case "fallback":
		m.metrics.FallbackUsage++
	}

	// Update time series
	minute := time.Now().Minute()
	m.metrics.ErrorsPerMinute[minute]++

	// Send alert for critical errors
	if m.config.EnableAlerting && err != nil {
		alert := &Alert{
			ID:        fmt.Sprintf("error_%d", time.Now().Unix()),
			Type:      AlertTypeConfigError,
			Severity:  SeverityError,
			Message:   fmt.Sprintf("Rate limiting error: %s - %v", errorType, err),
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"error_type": errorType,
				"error":      err.Error(),
			},
		}

		select {
		case m.alerts <- alert:
		default:
			log.Printf("Alert channel full, dropping error alert")
		}
	}
}

// GetMetrics returns current metrics
func (m *Monitor) GetMetrics(ctx context.Context) (*Metrics, error) {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	// Create a deep copy to avoid race conditions
	metrics := &Metrics{
		TotalRequests:     m.metrics.TotalRequests,
		AllowedRequests:   m.metrics.AllowedRequests,
		BlockedRequests:   m.metrics.BlockedRequests,
		AverageCheckTime:  m.metrics.AverageCheckTime,
		MaxCheckTime:      m.metrics.MaxCheckTime,
		MinCheckTime:      m.metrics.MinCheckTime,
		TotalErrors:       m.metrics.TotalErrors,
		RedisErrors:       m.metrics.RedisErrors,
		FallbackUsage:     m.metrics.FallbackUsage,
		ActiveConnections: m.metrics.ActiveConnections,
		HealthyLimiters:   m.metrics.HealthyLimiters,
		LastUpdated:       m.metrics.LastUpdated,
		EndpointMetrics:   make(map[string]*EndpointMetrics),
		RequestsPerMinute: make([]int64, len(m.metrics.RequestsPerMinute)),
		BlockedPerMinute:  make([]int64, len(m.metrics.BlockedPerMinute)),
		ErrorsPerMinute:   make([]int64, len(m.metrics.ErrorsPerMinute)),
	}

	// Copy time series data
	copy(metrics.RequestsPerMinute, m.metrics.RequestsPerMinute)
	copy(metrics.BlockedPerMinute, m.metrics.BlockedPerMinute)
	copy(metrics.ErrorsPerMinute, m.metrics.ErrorsPerMinute)

	// Copy endpoint metrics
	for endpoint, endpointMetrics := range m.metrics.EndpointMetrics {
		endpointMetrics.mu.RLock()
		copied := &EndpointMetrics{
			Endpoint:        endpointMetrics.Endpoint,
			TotalRequests:   endpointMetrics.TotalRequests,
			AllowedRequests: endpointMetrics.AllowedRequests,
			BlockedRequests: endpointMetrics.BlockedRequests,
			AverageLatency:  endpointMetrics.AverageLatency,
			LastRequest:     endpointMetrics.LastRequest,
			UtilizationRate: endpointMetrics.UtilizationRate,
			RecentRequests:  make([]RequestRecord, len(endpointMetrics.RecentRequests)),
		}
		copy(copied.RecentRequests, endpointMetrics.RecentRequests)
		metrics.EndpointMetrics[endpoint] = copied
		endpointMetrics.mu.RUnlock()
	}

	return metrics, nil
}

// GetAlerts returns current alerts
func (m *Monitor) GetAlerts(ctx context.Context) ([]*Alert, error) {
	alerts := make([]*Alert, 0)

	// Drain alerts channel without blocking
	for {
		select {
		case alert := <-m.alerts:
			alerts = append(alerts, alert)
		default:
			return alerts, nil
		}
	}
}

// GetHealthStatus returns the health status of rate limiting
func (m *Monitor) GetHealthStatus(ctx context.Context) (map[string]interface{}, error) {
	status := make(map[string]interface{})

	// Check limiter health
	var limiterHealthy bool
	if m.limiter != nil {
		err := m.limiter.IsHealthy(ctx)
		limiterHealthy = err == nil
		if err != nil {
			status["limiter_error"] = err.Error()
		}
	}

	status["limiter_healthy"] = limiterHealthy
	status["monitoring_enabled"] = m.config.EnableMetrics
	status["alerting_enabled"] = m.config.EnableAlerting
	status["last_check"] = m.lastCheck
	status["alerts_in_queue"] = len(m.alerts)

	// Get current metrics summary
	m.metrics.mu.RLock()
	status["total_requests"] = m.metrics.TotalRequests
	status["success_rate"] = float64(m.metrics.AllowedRequests) / float64(m.metrics.TotalRequests)
	status["error_rate"] = float64(m.metrics.TotalErrors) / float64(m.metrics.TotalRequests)
	status["active_endpoints"] = len(m.metrics.EndpointMetrics)
	m.metrics.mu.RUnlock()

	return status, nil
}

// getOrCreateEndpointMetrics returns existing or creates new endpoint metrics
func (m *Monitor) getOrCreateEndpointMetrics(endpoint string) *EndpointMetrics {
	if existing, exists := m.metrics.EndpointMetrics[endpoint]; exists {
		return existing
	}

	endpointMetrics := &EndpointMetrics{
		Endpoint:       endpoint,
		RecentRequests: make([]RequestRecord, 0, 100),
	}
	m.metrics.EndpointMetrics[endpoint] = endpointMetrics
	return endpointMetrics
}

// updateTimeSeriesMetrics updates minute-by-minute metrics
func (m *Monitor) updateTimeSeriesMetrics(allowed bool) {
	minute := time.Now().Minute()

	m.metrics.RequestsPerMinute[minute]++
	if !allowed {
		m.metrics.BlockedPerMinute[minute]++
	}
}

// checkForAlerts checks if any alerts should be triggered
func (m *Monitor) checkForAlerts(endpoint, key string, result *LimitResult, _ *EndpointMetrics) {
	// High usage alert
	if result.Limit > 0 {
		utilizationRate := float64(result.Count) / float64(result.Limit)
		if utilizationRate >= m.config.AlertThreshold {
			alertKey := "high_usage_" + endpoint
			if m.shouldSendAlert(alertKey) {
				alert := &Alert{
					ID:        fmt.Sprintf("usage_%s_%d", endpoint, time.Now().Unix()),
					Type:      AlertTypeHighUsage,
					Severity:  SeverityWarning,
					Message:   fmt.Sprintf("High usage detected for endpoint %s: %.1f%% of limit", endpoint, utilizationRate*100),
					Timestamp: time.Now(),
					Endpoint:  endpoint,
					Key:       key,
					Metadata: map[string]interface{}{
						"utilization_rate": utilizationRate,
						"current_count":    result.Count,
						"limit":            result.Limit,
					},
				}

				select {
				case m.alerts <- alert:
					m.alertsSent[alertKey] = time.Now()
				default:
					log.Printf("Alert channel full, dropping high usage alert")
				}
			}
		}
	}

	// Rate limited alert
	if !result.Allowed {
		alertKey := fmt.Sprintf("rate_limited_%s", endpoint)
		if m.shouldSendAlert(alertKey) {
			alert := &Alert{
				ID:        fmt.Sprintf("limited_%s_%d", endpoint, time.Now().Unix()),
				Type:      AlertTypeRateLimited,
				Severity:  SeverityInfo,
				Message:   fmt.Sprintf("Rate limit exceeded for endpoint %s", endpoint),
				Timestamp: time.Now(),
				Endpoint:  endpoint,
				Key:       key,
				Metadata: map[string]interface{}{
					"current_count": result.Count,
					"limit":         result.Limit,
					"retry_after":   result.RetryAfter.Seconds(),
				},
			}

			select {
			case m.alerts <- alert:
				m.alertsSent[alertKey] = time.Now()
			default:
				log.Printf("Alert channel full, dropping rate limited alert")
			}
		}
	}
}

// shouldSendAlert checks if enough time has passed since last alert
func (m *Monitor) shouldSendAlert(alertKey string) bool {
	if lastSent, exists := m.alertsSent[alertKey]; exists {
		// Only send alerts every 5 minutes for the same key
		return time.Since(lastSent) > 5*time.Minute
	}
	return true
}

// alertingRoutine processes alerts
func (m *Monitor) alertingRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case alert := <-m.alerts:
			m.processAlert(alert)
		case <-ticker.C:
			m.cleanupOldAlerts()
		case <-m.done:
			return
		}
	}
}

// processAlert processes a single alert
func (m *Monitor) processAlert(alert *Alert) {
	// Log the alert
	alertJSON, err := json.Marshal(alert)
	if err != nil {
		log.Printf("Rate limit alert (failed to marshal): %+v", alert)
		return
	}
	log.Printf("Rate limit alert: %s", alertJSON)

	// Here you could add integrations with:
	// - Slack webhooks
	// - Email notifications
	// - PagerDuty
	// - Custom webhook endpoints
}

// metricsCollectionRoutine collects metrics periodically
func (m *Monitor) metricsCollectionRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectSystemMetrics()
		case <-m.done:
			return
		}
	}
}

// performanceMonitoringRoutine monitors performance metrics
func (m *Monitor) performanceMonitoringRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectPerformanceMetrics()
		case <-m.done:
			return
		}
	}
}

// collectSystemMetrics collects system-level metrics
func (m *Monitor) collectSystemMetrics() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check limiter health
	if m.limiter != nil {
		if err := m.limiter.IsHealthy(ctx); err != nil {
			m.RecordError("health_check", err)
		}
	}

	m.lastCheck = time.Now()
}

// collectPerformanceMetrics collects performance metrics
func (m *Monitor) collectPerformanceMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	metric := PerformanceMetric{
		Timestamp:    time.Now(),
		RequestCount: m.metrics.TotalRequests,
		ErrorCount:   m.metrics.TotalErrors,
	}

	m.performanceLog = append(m.performanceLog, metric)

	// Keep only last 1000 entries
	if len(m.performanceLog) > 1000 {
		m.performanceLog = m.performanceLog[len(m.performanceLog)-1000:]
	}
}

// cleanupOldAlerts removes old alert tracking entries
func (m *Monitor) cleanupOldAlerts() {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-time.Hour)
	for key, timestamp := range m.alertsSent {
		if timestamp.Before(cutoff) {
			delete(m.alertsSent, key)
		}
	}
}

// Close stops the monitor
func (m *Monitor) Close() error {
	close(m.done)
	return nil
}
