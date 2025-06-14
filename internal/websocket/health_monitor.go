// Package websocket provides health monitoring for WebSocket connections
package websocket

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// HealthMonitor monitors the health of WebSocket connections
type HealthMonitor struct {
	mu              sync.RWMutex
	config          *HealthConfig
	connections     map[string]*ConnectionHealth
	aggregateHealth *AggregateHealth
	alerts          chan *HealthAlert
	done            chan struct{}
	lastUpdate      time.Time
}

// HealthConfig configures health monitoring behavior
type HealthConfig struct {
	CheckInterval      time.Duration `json:"check_interval" yaml:"check_interval"`
	PingTimeout        time.Duration `json:"ping_timeout" yaml:"ping_timeout"`
	HealthThreshold    float64       `json:"health_threshold" yaml:"health_threshold"`
	UnhealthyThreshold float64       `json:"unhealthy_threshold" yaml:"unhealthy_threshold"`
	MaxSampleSize      int           `json:"max_sample_size" yaml:"max_sample_size"`
	AlertCooldown      time.Duration `json:"alert_cooldown" yaml:"alert_cooldown"`
	EnableDetailedLogs bool          `json:"enable_detailed_logs" yaml:"enable_detailed_logs"`
	TrackPerformance   bool          `json:"track_performance" yaml:"track_performance"`
}

// ConnectionHealth tracks health metrics for a single connection
type ConnectionHealth struct {
	mu                  sync.RWMutex
	ID                  string
	Connection          *websocket.Conn
	HealthScore         float64
	LastPing            time.Time
	LastPong            time.Time
	PingLatency         time.Duration
	AverageLatency      time.Duration
	MinLatency          time.Duration
	MaxLatency          time.Duration
	LatencySamples      []time.Duration
	ErrorCount          int64
	TotalPings          int64
	SuccessfulPings     int64
	ConsecutiveFailures int
	State               HealthState
	CreatedAt           time.Time
	LastHealthCheck     time.Time
	Metadata            map[string]interface{}
}

// AggregateHealth tracks overall health across all connections
type AggregateHealth struct {
	mu                   sync.RWMutex
	TotalConnections     int           `json:"total_connections"`
	HealthyConnections   int           `json:"healthy_connections"`
	UnhealthyConnections int           `json:"unhealthy_connections"`
	AverageHealthScore   float64       `json:"average_health_score"`
	AverageLatency       time.Duration `json:"average_latency"`
	TotalErrors          int64         `json:"total_errors"`
	OverallHealthStatus  string        `json:"overall_health_status"`
	LastUpdated          time.Time     `json:"last_updated"`
}

// HealthAlert represents a health-related alert
type HealthAlert struct {
	ID           string                 `json:"id"`
	Type         HealthAlertType        `json:"type"`
	Severity     HealthAlertSeverity    `json:"severity"`
	ConnectionID string                 `json:"connection_id,omitempty"`
	Message      string                 `json:"message"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// HealthState represents the health state of a connection
type HealthState int

const (
	HealthStateUnknown HealthState = iota
	HealthStateHealthy
	HealthStateWarning
	HealthStateUnhealthy
	HealthStateCritical
)

// HealthAlertType represents different types of health alerts
type HealthAlertType string

const (
	AlertTypeHighLatency      HealthAlertType = "high_latency"
	AlertTypeConnectionDown   HealthAlertType = "connection_down"
	AlertTypeHealthDegraded   HealthAlertType = "health_degraded"
	AlertTypeSystemUnhealthy  HealthAlertType = "system_unhealthy"
	AlertTypeRecoveryComplete HealthAlertType = "recovery_complete"
)

// HealthAlertSeverity represents alert severity levels
type HealthAlertSeverity string

const (
	HealthSeverityInfo     HealthAlertSeverity = "info"
	HealthSeverityWarning  HealthAlertSeverity = "warning"
	HealthSeverityError    HealthAlertSeverity = "error"
	HealthSeverityCritical HealthAlertSeverity = "critical"
)

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(config *HealthConfig) *HealthMonitor {
	if config == nil {
		config = DefaultHealthConfig()
	}

	hm := &HealthMonitor{
		config:          config,
		connections:     make(map[string]*ConnectionHealth),
		aggregateHealth: &AggregateHealth{},
		alerts:          make(chan *HealthAlert, 1000),
		done:            make(chan struct{}),
		lastUpdate:      time.Now(),
	}

	// Start monitoring routines
	go hm.healthCheckRoutine()
	go hm.aggregateHealthRoutine()

	return hm
}

// DefaultHealthConfig returns default health monitoring configuration
func DefaultHealthConfig() *HealthConfig {
	return &HealthConfig{
		CheckInterval:      30 * time.Second,
		PingTimeout:        5 * time.Second,
		HealthThreshold:    0.8,
		UnhealthyThreshold: 0.5,
		MaxSampleSize:      100,
		AlertCooldown:      5 * time.Minute,
		EnableDetailedLogs: true,
		TrackPerformance:   true,
	}
}

// RegisterConnection registers a connection for health monitoring
func (hm *HealthMonitor) RegisterConnection(id string, conn *websocket.Conn) error {
	if id == "" {
		return errors.New("connection ID cannot be empty")
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	health := &ConnectionHealth{
		ID:              id,
		Connection:      conn,
		HealthScore:     1.0,
		State:           HealthStateHealthy,
		CreatedAt:       time.Now(),
		LastHealthCheck: time.Now(),
		LatencySamples:  make([]time.Duration, 0, hm.config.MaxSampleSize),
		Metadata:        make(map[string]interface{}),
	}

	hm.connections[id] = health

	if hm.config.EnableDetailedLogs {
		log.Printf("Registered connection %s for health monitoring", id)
	}

	return nil
}

// UnregisterConnection removes a connection from health monitoring
func (hm *HealthMonitor) UnregisterConnection(id string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.connections[id]; exists {
		delete(hm.connections, id)

		if hm.config.EnableDetailedLogs {
			log.Printf("Unregistered connection %s from health monitoring", id)
		}
	}
}

// healthCheckRoutine performs periodic health checks
func (hm *HealthMonitor) healthCheckRoutine() {
	ticker := time.NewTicker(hm.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hm.performHealthChecks()
		case <-hm.done:
			return
		}
	}
}

// performHealthChecks checks health of all registered connections
func (hm *HealthMonitor) performHealthChecks() {
	hm.mu.RLock()
	connections := make([]*ConnectionHealth, 0, len(hm.connections))
	for _, conn := range hm.connections {
		connections = append(connections, conn)
	}
	hm.mu.RUnlock()

	for _, conn := range connections {
		go hm.checkConnectionHealth(conn)
	}
}

// checkConnectionHealth checks health of a single connection
func (hm *HealthMonitor) checkConnectionHealth(health *ConnectionHealth) {
	health.mu.Lock()
	conn := health.Connection
	if conn == nil {
		health.mu.Unlock()
		return
	}
	health.mu.Unlock()

	// Perform ping test
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), hm.config.PingTimeout)
	defer cancel()

	done := make(chan bool, 1)
	var pingErr error

	go func() {
		pingErr = conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(hm.config.PingTimeout))
		done <- true
	}()

	select {
	case <-done:
		latency := time.Since(start)
		hm.updateHealthMetrics(health, latency, pingErr)
	case <-ctx.Done():
		hm.updateHealthMetrics(health, hm.config.PingTimeout, errors.New("ping timeout"))
	}

	// Update health score and state
	hm.calculateHealthScore(health)
	hm.updateHealthState(health)
}

// updateHealthMetrics updates health metrics for a connection
func (hm *HealthMonitor) updateHealthMetrics(health *ConnectionHealth, latency time.Duration, err error) {
	health.mu.Lock()
	defer health.mu.Unlock()

	health.LastHealthCheck = time.Now()
	atomic.AddInt64(&health.TotalPings, 1)

	if err != nil {
		atomic.AddInt64(&health.ErrorCount, 1)
		health.ConsecutiveFailures++

		if hm.config.EnableDetailedLogs {
			log.Printf("Health check failed for connection %s: %v", health.ID, err)
		}

		// Send alert for consecutive failures
		if health.ConsecutiveFailures >= 3 {
			hm.sendAlert(&HealthAlert{
				ID:           fmt.Sprintf("conn_down_%s_%d", health.ID, time.Now().Unix()),
				Type:         AlertTypeConnectionDown,
				Severity:     HealthSeverityError,
				ConnectionID: health.ID,
				Message:      fmt.Sprintf("Connection %s has %d consecutive failures", health.ID, health.ConsecutiveFailures),
				Timestamp:    time.Now(),
				Metadata: map[string]interface{}{
					"consecutive_failures": health.ConsecutiveFailures,
					"error":                err.Error(),
				},
			})
		}
		return
	}

	// Successful ping
	atomic.AddInt64(&health.SuccessfulPings, 1)
	health.ConsecutiveFailures = 0
	health.LastPing = time.Now()
	health.PingLatency = latency

	// Update latency metrics
	hm.updateLatencyMetrics(health, latency)

	// Check for high latency alerts
	if latency > 500*time.Millisecond {
		hm.sendAlert(&HealthAlert{
			ID:           fmt.Sprintf("high_latency_%s_%d", health.ID, time.Now().Unix()),
			Type:         AlertTypeHighLatency,
			Severity:     HealthSeverityWarning,
			ConnectionID: health.ID,
			Message:      fmt.Sprintf("High latency detected for connection %s: %v", health.ID, latency),
			Timestamp:    time.Now(),
			Metadata: map[string]interface{}{
				"latency_ms": latency.Milliseconds(),
			},
		})
	}
}

// updateLatencyMetrics updates latency statistics
func (hm *HealthMonitor) updateLatencyMetrics(health *ConnectionHealth, latency time.Duration) {
	// Add to samples
	health.LatencySamples = append(health.LatencySamples, latency)
	if len(health.LatencySamples) > hm.config.MaxSampleSize {
		health.LatencySamples = health.LatencySamples[1:]
	}

	// Update min/max
	if health.MinLatency == 0 || latency < health.MinLatency {
		health.MinLatency = latency
	}
	if latency > health.MaxLatency {
		health.MaxLatency = latency
	}

	// Calculate average
	if health.AverageLatency == 0 {
		health.AverageLatency = latency
	} else {
		health.AverageLatency = (health.AverageLatency + latency) / 2
	}
}

// calculateHealthScore calculates health score based on various metrics
func (hm *HealthMonitor) calculateHealthScore(health *ConnectionHealth) {
	health.mu.Lock()
	defer health.mu.Unlock()

	if health.TotalPings == 0 {
		health.HealthScore = 1.0
		return
	}

	// Base score on success rate
	successRate := float64(health.SuccessfulPings) / float64(health.TotalPings)

	// Penalty for consecutive failures
	failurePenalty := math.Min(float64(health.ConsecutiveFailures)*0.1, 0.5)

	// Penalty for high latency
	var latencyPenalty float64
	if health.AverageLatency > 0 {
		switch {
		case health.AverageLatency > time.Second:
			latencyPenalty = 0.3
		case health.AverageLatency > 500*time.Millisecond:
			latencyPenalty = 0.2
		case health.AverageLatency > 200*time.Millisecond:
			latencyPenalty = 0.1
		}
	}

	// Calculate final score
	score := successRate - failurePenalty - latencyPenalty
	health.HealthScore = math.Max(0.0, math.Min(1.0, score))
}

// updateHealthState updates health state based on health score
func (hm *HealthMonitor) updateHealthState(health *ConnectionHealth) {
	health.mu.Lock()
	oldState := health.State

	switch {
	case health.HealthScore >= hm.config.HealthThreshold:
		health.State = HealthStateHealthy
	case health.HealthScore >= hm.config.UnhealthyThreshold:
		health.State = HealthStateWarning
	case health.HealthScore > 0:
		health.State = HealthStateUnhealthy
	default:
		health.State = HealthStateCritical
	}

	newState := health.State
	health.mu.Unlock()

	// Send alert on state change
	if oldState != newState && newState != HealthStateHealthy {
		severity := HealthSeverityWarning
		if newState == HealthStateCritical {
			severity = HealthSeverityCritical
		}

		hm.sendAlert(&HealthAlert{
			ID:           fmt.Sprintf("state_change_%s_%d", health.ID, time.Now().Unix()),
			Type:         AlertTypeHealthDegraded,
			Severity:     severity,
			ConnectionID: health.ID,
			Message:      fmt.Sprintf("Connection %s health degraded: %s -> %s", health.ID, oldState, newState),
			Timestamp:    time.Now(),
			Metadata: map[string]interface{}{
				"old_state":    oldState.String(),
				"new_state":    newState.String(),
				"health_score": health.HealthScore,
			},
		})
	}
}

// aggregateHealthRoutine calculates aggregate health metrics
func (hm *HealthMonitor) aggregateHealthRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hm.calculateAggregateHealth()
		case <-hm.done:
			return
		}
	}
}

// calculateAggregateHealth calculates overall system health
func (hm *HealthMonitor) calculateAggregateHealth() {
	hm.mu.RLock()
	connections := make([]*ConnectionHealth, 0, len(hm.connections))
	for _, conn := range hm.connections {
		connections = append(connections, conn)
	}
	hm.mu.RUnlock()

	if len(connections) == 0 {
		return
	}

	hm.aggregateHealth.mu.Lock()
	defer hm.aggregateHealth.mu.Unlock()

	var totalScore float64
	var totalLatency time.Duration
	var totalErrors int64
	healthy := 0
	unhealthy := 0

	for _, conn := range connections {
		conn.mu.RLock()
		totalScore += conn.HealthScore
		totalLatency += conn.AverageLatency
		totalErrors += conn.ErrorCount

		if conn.State == HealthStateHealthy {
			healthy++
		} else {
			unhealthy++
		}
		conn.mu.RUnlock()
	}

	hm.aggregateHealth.TotalConnections = len(connections)
	hm.aggregateHealth.HealthyConnections = healthy
	hm.aggregateHealth.UnhealthyConnections = unhealthy
	hm.aggregateHealth.AverageHealthScore = totalScore / float64(len(connections))
	hm.aggregateHealth.AverageLatency = totalLatency / time.Duration(len(connections))
	hm.aggregateHealth.TotalErrors = totalErrors
	hm.aggregateHealth.LastUpdated = time.Now()

	// Determine overall status
	healthyRatio := float64(healthy) / float64(len(connections))
	switch {
	case healthyRatio >= 0.9:
		hm.aggregateHealth.OverallHealthStatus = "healthy"
	case healthyRatio >= 0.7:
		hm.aggregateHealth.OverallHealthStatus = "warning"
	case healthyRatio >= 0.5:
		hm.aggregateHealth.OverallHealthStatus = "unhealthy"
	default:
		hm.aggregateHealth.OverallHealthStatus = "critical"
	}
}

// sendAlert sends a health alert
func (hm *HealthMonitor) sendAlert(alert *HealthAlert) {
	select {
	case hm.alerts <- alert:
		if hm.config.EnableDetailedLogs {
			log.Printf("Health alert: %s - %s", alert.Type, alert.Message)
		}
	default:
		log.Printf("Alert queue full, dropping health alert")
	}
}

// GetConnectionHealth returns health information for a specific connection
func (hm *HealthMonitor) GetConnectionHealth(id string) (*ConnectionHealth, error) {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	health, exists := hm.connections[id]
	if !exists {
		return nil, fmt.Errorf("connection %s not found", id)
	}

	// Return a copy to avoid race conditions
	health.mu.RLock()
	defer health.mu.RUnlock()

	return &ConnectionHealth{
		ID:                  health.ID,
		HealthScore:         health.HealthScore,
		LastPing:            health.LastPing,
		LastPong:            health.LastPong,
		PingLatency:         health.PingLatency,
		AverageLatency:      health.AverageLatency,
		MinLatency:          health.MinLatency,
		MaxLatency:          health.MaxLatency,
		ErrorCount:          health.ErrorCount,
		TotalPings:          health.TotalPings,
		SuccessfulPings:     health.SuccessfulPings,
		ConsecutiveFailures: health.ConsecutiveFailures,
		State:               health.State,
		CreatedAt:           health.CreatedAt,
		LastHealthCheck:     health.LastHealthCheck,
	}, nil
}

// GetAggregateHealth returns overall health metrics
func (hm *HealthMonitor) GetAggregateHealth() *AggregateHealth {
	hm.aggregateHealth.mu.RLock()
	defer hm.aggregateHealth.mu.RUnlock()

	return &AggregateHealth{
		TotalConnections:     hm.aggregateHealth.TotalConnections,
		HealthyConnections:   hm.aggregateHealth.HealthyConnections,
		UnhealthyConnections: hm.aggregateHealth.UnhealthyConnections,
		AverageHealthScore:   hm.aggregateHealth.AverageHealthScore,
		AverageLatency:       hm.aggregateHealth.AverageLatency,
		TotalErrors:          hm.aggregateHealth.TotalErrors,
		OverallHealthStatus:  hm.aggregateHealth.OverallHealthStatus,
		LastUpdated:          hm.aggregateHealth.LastUpdated,
	}
}

// GetAlerts returns pending health alerts
func (hm *HealthMonitor) GetAlerts() []*HealthAlert {
	var alerts []*HealthAlert

	// Drain alerts channel without blocking
	for {
		select {
		case alert := <-hm.alerts:
			alerts = append(alerts, alert)
		default:
			return alerts
		}
	}
}

// String returns string representation of HealthState
func (hs HealthState) String() string {
	switch hs {
	case HealthStateHealthy:
		return "healthy"
	case HealthStateWarning:
		return "warning"
	case HealthStateUnhealthy:
		return "unhealthy"
	case HealthStateCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Close stops the health monitor
func (hm *HealthMonitor) Close() error {
	close(hm.done)
	return nil
}
