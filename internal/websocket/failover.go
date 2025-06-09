// Package websocket provides failover mechanisms for WebSocket connections
package websocket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// FailoverManager manages failover mechanisms for WebSocket connections
type FailoverManager struct {
	mu                 sync.RWMutex
	config             *FailoverConfig
	connections        map[string]*FailoverConnection
	fallbackHandlers   map[string]FallbackHandler
	failoverStrategies map[string]FailoverStrategy
	metrics            *FailoverMetrics
	done               chan struct{}
}

// FailoverConfig configures failover behavior
type FailoverConfig struct {
	EnableFailover            bool             `json:"enable_failover" yaml:"enable_failover"`
	FailureThreshold          int              `json:"failure_threshold" yaml:"failure_threshold"`
	HealthCheckInterval       time.Duration    `json:"health_check_interval" yaml:"health_check_interval"`
	FailoverTimeout           time.Duration    `json:"failover_timeout" yaml:"failover_timeout"`
	FallbackMethods           []FallbackMethod `json:"fallback_methods" yaml:"fallback_methods"`
	MaxFailoverAttempts       int              `json:"max_failover_attempts" yaml:"max_failover_attempts"`
	EnableGracefulDegradation bool             `json:"enable_graceful_degradation" yaml:"enable_graceful_degradation"`
	MonitoringEnabled         bool             `json:"monitoring_enabled" yaml:"monitoring_enabled"`
	AutoRecovery              bool             `json:"auto_recovery" yaml:"auto_recovery"`
	RecoveryCheckInterval     time.Duration    `json:"recovery_check_interval" yaml:"recovery_check_interval"`
}

// FailoverConnection represents a connection with failover capabilities
type FailoverConnection struct {
	mu                 sync.RWMutex
	ID                 string
	PrimaryEndpoint    string
	FallbackEndpoints  []string
	CurrentEndpoint    string
	State              FailoverState
	FailureCount       int
	LastFailure        time.Time
	FailoverAttempts   int
	ActiveFallback     FallbackMethod
	FallbackData       interface{}
	HealthScore        float64
	LastHealthCheck    time.Time
	RecoveryInProgress bool
	Metadata           map[string]interface{}

	// Connection tracking
	WSConnection    interface{} // WebSocket connection
	HTTPFallback    *HTTPFallback
	PollingFallback *PollingFallback

	// Callbacks
	OnFailover          func(from, to string) error
	OnRecovery          func() error
	OnFallbackActivated func(method FallbackMethod) error
}

// FailoverMetrics tracks failover performance
type FailoverMetrics struct {
	TotalFailovers       int64         `json:"total_failovers"`
	SuccessfulFailovers  int64         `json:"successful_failovers"`
	FailedFailovers      int64         `json:"failed_failovers"`
	ActiveFallbacks      int64         `json:"active_fallbacks"`
	AverageFailoverTime  time.Duration `json:"average_failover_time"`
	MaxFailoverTime      time.Duration `json:"max_failover_time"`
	MinFailoverTime      time.Duration `json:"min_failover_time"`
	RecoveryAttempts     int64         `json:"recovery_attempts"`
	SuccessfulRecoveries int64         `json:"successful_recoveries"`
	LastUpdated          time.Time     `json:"last_updated"`
}

// HTTPFallback provides HTTP polling fallback
type HTTPFallback struct {
	Client       *http.Client
	PollURL      string
	PollInterval time.Duration
	MessageQueue chan []byte
	Running      bool
	LastPoll     time.Time
	PollCount    int64
	ErrorCount   int64
	ctx          context.Context
	cancel       context.CancelFunc
}

// PollingFallback provides server-sent events fallback
type PollingFallback struct {
	EventSource  string
	PollInterval time.Duration
	MessageQueue chan []byte
	Running      bool
	LastEvent    time.Time
	EventCount   int64
	ErrorCount   int64
	ctx          context.Context
	cancel       context.CancelFunc
}

// FallbackHandler interface for different fallback mechanisms
type FallbackHandler interface {
	Initialize(connectionID string, config interface{}) error
	Start(ctx context.Context) error
	Stop() error
	SendMessage(data []byte) error
	ReceiveMessage() ([]byte, error)
	IsHealthy() bool
	GetMetrics() map[string]interface{}
	GetType() FallbackMethod
}

// FailoverStrategy interface for different failover strategies
type FailoverStrategy interface {
	ShouldFailover(connection *FailoverConnection) bool
	SelectFallback(connection *FailoverConnection) (FallbackMethod, error)
	ExecuteFailover(connection *FailoverConnection, method FallbackMethod) error
	GetName() string
}

// Enums and constants

type FailoverState string

const (
	FailoverStateNormal     FailoverState = "normal"
	FailoverStateFailedOver FailoverState = "failed_over"
	FailoverStateRecovering FailoverState = "recovering"
	FailoverStateFailed     FailoverState = "failed"
	FailoverStateDegraded   FailoverState = "degraded"
)

type FallbackMethod string

const (
	FallbackHTTPPolling FallbackMethod = "http_polling"
	FallbackSSE         FallbackMethod = "server_sent_events"
	FallbackWebhook     FallbackMethod = "webhook"
	FallbackTCP         FallbackMethod = "tcp_socket"
	FallbackUDP         FallbackMethod = "udp_socket"
)

// NewFailoverManager creates a new failover manager
func NewFailoverManager(config *FailoverConfig) *FailoverManager {
	if config == nil {
		config = DefaultFailoverConfig()
	}

	fm := &FailoverManager{
		config:             config,
		connections:        make(map[string]*FailoverConnection),
		fallbackHandlers:   make(map[string]FallbackHandler),
		failoverStrategies: make(map[string]FailoverStrategy),
		metrics:            &FailoverMetrics{},
		done:               make(chan struct{}),
	}

	// Register default strategies
	fm.registerDefaultStrategies()

	// Start monitoring if enabled
	if config.MonitoringEnabled {
		go fm.monitoringRoutine()
	}

	// Start recovery routine if auto-recovery is enabled
	if config.AutoRecovery {
		go fm.recoveryRoutine()
	}

	return fm
}

// DefaultFailoverConfig returns default failover configuration
func DefaultFailoverConfig() *FailoverConfig {
	return &FailoverConfig{
		EnableFailover:            true,
		FailureThreshold:          3,
		HealthCheckInterval:       30 * time.Second,
		FailoverTimeout:           10 * time.Second,
		FallbackMethods:           []FallbackMethod{FallbackHTTPPolling, FallbackSSE},
		MaxFailoverAttempts:       3,
		EnableGracefulDegradation: true,
		MonitoringEnabled:         true,
		AutoRecovery:              true,
		RecoveryCheckInterval:     2 * time.Minute,
	}
}

// registerDefaultStrategies registers default failover strategies
func (fm *FailoverManager) registerDefaultStrategies() {
	fm.failoverStrategies["threshold"] = &ThresholdStrategy{
		FailureThreshold: fm.config.FailureThreshold,
	}

	fm.failoverStrategies["health_based"] = &HealthBasedStrategy{
		MinHealthScore: 0.5,
	}

	fm.failoverStrategies["latency_based"] = &LatencyBasedStrategy{
		MaxLatency: 2 * time.Second,
	}
}

// RegisterConnection registers a connection for failover management
func (fm *FailoverManager) RegisterConnection(id, primaryEndpoint string, fallbackEndpoints []string) error {
	if id == "" {
		return fmt.Errorf("connection ID cannot be empty")
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	connection := &FailoverConnection{
		ID:                id,
		PrimaryEndpoint:   primaryEndpoint,
		FallbackEndpoints: fallbackEndpoints,
		CurrentEndpoint:   primaryEndpoint,
		State:             FailoverStateNormal,
		HealthScore:       1.0,
		Metadata:          make(map[string]interface{}),
	}

	fm.connections[id] = connection

	log.Printf("Registered connection %s for failover management", id)
	return nil
}

// UnregisterConnection removes a connection from failover management
func (fm *FailoverManager) UnregisterConnection(id string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	if connection, exists := fm.connections[id]; exists {
		// Stop any active fallbacks
		fm.stopFallbacks(connection)
		delete(fm.connections, id)
		log.Printf("Unregistered connection %s from failover management", id)
	}
}

// HandleConnectionFailure handles a connection failure
func (fm *FailoverManager) HandleConnectionFailure(connectionID string, err error) error {
	if !fm.config.EnableFailover {
		return fmt.Errorf("failover is disabled")
	}

	fm.mu.RLock()
	connection, exists := fm.connections[connectionID]
	fm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}

	connection.mu.Lock()
	connection.FailureCount++
	connection.LastFailure = time.Now()
	connection.mu.Unlock()

	log.Printf("Connection failure detected for %s: %v (count: %d)",
		connectionID, err, connection.FailureCount)

	// Check if failover should be triggered
	strategy := fm.failoverStrategies["threshold"] // Default strategy
	if strategy.ShouldFailover(connection) {
		return fm.executeFailover(connection, strategy)
	}

	return nil
}

// executeFailover executes failover for a connection
func (fm *FailoverManager) executeFailover(connection *FailoverConnection, strategy FailoverStrategy) error {
	start := time.Now()
	atomic.AddInt64(&fm.metrics.TotalFailovers, 1)

	connection.mu.Lock()
	if connection.FailoverAttempts >= fm.config.MaxFailoverAttempts {
		connection.State = FailoverStateFailed
		connection.mu.Unlock()
		atomic.AddInt64(&fm.metrics.FailedFailovers, 1)
		return fmt.Errorf("maximum failover attempts reached for connection %s", connection.ID)
	}

	connection.FailoverAttempts++
	connection.State = FailoverStateFailedOver
	oldEndpoint := connection.CurrentEndpoint
	connection.mu.Unlock()

	log.Printf("Executing failover for connection %s (attempt %d)",
		connection.ID, connection.FailoverAttempts)

	// Select fallback method
	fallbackMethod, err := strategy.SelectFallback(connection)
	if err != nil {
		atomic.AddInt64(&fm.metrics.FailedFailovers, 1)
		return fmt.Errorf("failed to select fallback method: %w", err)
	}

	// Execute the failover
	ctx, cancel := context.WithTimeout(context.Background(), fm.config.FailoverTimeout)
	defer cancel()

	err = fm.activateFallback(ctx, connection, fallbackMethod)
	if err != nil {
		atomic.AddInt64(&fm.metrics.FailedFailovers, 1)
		return fmt.Errorf("failed to activate fallback: %w", err)
	}

	// Update metrics
	duration := time.Since(start)
	fm.updateFailoverMetrics(duration)
	atomic.AddInt64(&fm.metrics.SuccessfulFailovers, 1)
	atomic.AddInt64(&fm.metrics.ActiveFallbacks, 1)

	// Call failover callback
	connection.mu.RLock()
	onFailover := connection.OnFailover
	newEndpoint := connection.CurrentEndpoint
	connection.mu.RUnlock()

	if onFailover != nil {
		go func() {
			if err := onFailover(oldEndpoint, newEndpoint); err != nil {
				log.Printf("Failover callback failed for connection %s: %v", connection.ID, err)
			}
		}()
	}

	log.Printf("Failover completed for connection %s in %v", connection.ID, duration)
	return nil
}

// activateFallback activates a fallback method
func (fm *FailoverManager) activateFallback(ctx context.Context, connection *FailoverConnection, method FallbackMethod) error {
	switch method {
	case FallbackHTTPPolling:
		return fm.activateHTTPPolling(ctx, connection)
	case FallbackSSE:
		return fm.activateSSE(ctx, connection)
	default:
		return fmt.Errorf("unsupported fallback method: %s", method)
	}
}

// activateHTTPPolling activates HTTP polling fallback
func (fm *FailoverManager) activateHTTPPolling(ctx context.Context, connection *FailoverConnection) error {
	connection.mu.Lock()
	defer connection.mu.Unlock()

	// Create HTTP fallback
	fallbackCtx, cancel := context.WithCancel(context.Background())

	httpFallback := &HTTPFallback{
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		PollURL:      fmt.Sprintf("http://localhost:9080/api/v1/poll/%s", connection.ID),
		PollInterval: 5 * time.Second,
		MessageQueue: make(chan []byte, 100),
		ctx:          fallbackCtx,
		cancel:       cancel,
	}

	connection.HTTPFallback = httpFallback
	connection.ActiveFallback = FallbackHTTPPolling
	connection.CurrentEndpoint = httpFallback.PollURL

	// Start polling
	go fm.runHTTPPolling(httpFallback)

	// Call fallback activated callback
	if connection.OnFallbackActivated != nil {
		go func() {
			if err := connection.OnFallbackActivated(FallbackHTTPPolling); err != nil {
				log.Printf("Fallback activated callback failed: %v", err)
			}
		}()
	}

	log.Printf("Activated HTTP polling fallback for connection %s", connection.ID)
	return nil
}

// activateSSE activates Server-Sent Events fallback
func (fm *FailoverManager) activateSSE(ctx context.Context, connection *FailoverConnection) error {
	connection.mu.Lock()
	defer connection.mu.Unlock()

	// Create SSE fallback
	fallbackCtx, cancel := context.WithCancel(context.Background())

	pollingFallback := &PollingFallback{
		EventSource:  fmt.Sprintf("http://localhost:9080/api/v1/sse/%s", connection.ID),
		PollInterval: 1 * time.Second,
		MessageQueue: make(chan []byte, 100),
		ctx:          fallbackCtx,
		cancel:       cancel,
	}

	connection.PollingFallback = pollingFallback
	connection.ActiveFallback = FallbackSSE
	connection.CurrentEndpoint = pollingFallback.EventSource

	// Start SSE
	go fm.runSSE(pollingFallback)

	// Call fallback activated callback
	if connection.OnFallbackActivated != nil {
		go func() {
			if err := connection.OnFallbackActivated(FallbackSSE); err != nil {
				log.Printf("Fallback activated callback failed: %v", err)
			}
		}()
	}

	log.Printf("Activated SSE fallback for connection %s", connection.ID)
	return nil
}

// runHTTPPolling runs HTTP polling fallback
func (fm *FailoverManager) runHTTPPolling(fallback *HTTPFallback) {
	fallback.Running = true
	defer func() {
		fallback.Running = false
	}()

	ticker := time.NewTicker(fallback.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fm.performHTTPPoll(fallback)
		case <-fallback.ctx.Done():
			return
		}
	}
}

// performHTTPPoll performs a single HTTP poll
func (fm *FailoverManager) performHTTPPoll(fallback *HTTPFallback) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fallback.PollURL, nil)
	if err != nil {
		atomic.AddInt64(&fallback.ErrorCount, 1)
		return
	}

	resp, err := fallback.Client.Do(req)
	if err != nil {
		atomic.AddInt64(&fallback.ErrorCount, 1)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Read response body and queue message
		// Implementation would depend on the API format
		atomic.AddInt64(&fallback.PollCount, 1)
		fallback.LastPoll = time.Now()
	} else {
		atomic.AddInt64(&fallback.ErrorCount, 1)
	}
}

// runSSE runs Server-Sent Events fallback
func (fm *FailoverManager) runSSE(fallback *PollingFallback) {
	fallback.Running = true
	defer func() {
		fallback.Running = false
	}()

	// Implementation would create SSE connection and listen for events
	// For simplicity, we'll use a polling approach here
	ticker := time.NewTicker(fallback.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Simulate SSE event processing
			atomic.AddInt64(&fallback.EventCount, 1)
			fallback.LastEvent = time.Now()
		case <-fallback.ctx.Done():
			return
		}
	}
}

// stopFallbacks stops all active fallbacks for a connection
func (fm *FailoverManager) stopFallbacks(connection *FailoverConnection) {
	connection.mu.Lock()
	defer connection.mu.Unlock()

	if connection.HTTPFallback != nil {
		connection.HTTPFallback.cancel()
		connection.HTTPFallback = nil
	}

	if connection.PollingFallback != nil {
		connection.PollingFallback.cancel()
		connection.PollingFallback = nil
	}

	if connection.ActiveFallback != "" {
		atomic.AddInt64(&fm.metrics.ActiveFallbacks, -1)
		connection.ActiveFallback = ""
	}
}

// AttemptRecovery attempts to recover a failed-over connection
func (fm *FailoverManager) AttemptRecovery(connectionID string) error {
	fm.mu.RLock()
	connection, exists := fm.connections[connectionID]
	fm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("connection %s not found", connectionID)
	}

	connection.mu.Lock()
	if connection.State != FailoverStateFailedOver {
		connection.mu.Unlock()
		return fmt.Errorf("connection %s is not in failed-over state", connectionID)
	}

	if connection.RecoveryInProgress {
		connection.mu.Unlock()
		return fmt.Errorf("recovery already in progress for connection %s", connectionID)
	}

	connection.RecoveryInProgress = true
	connection.State = FailoverStateRecovering
	connection.mu.Unlock()

	atomic.AddInt64(&fm.metrics.RecoveryAttempts, 1)

	log.Printf("Attempting recovery for connection %s", connectionID)

	// Attempt to reconnect to primary endpoint
	success := fm.testPrimaryConnection(connection)

	connection.mu.Lock()
	connection.RecoveryInProgress = false

	if success {
		// Stop fallbacks
		fm.stopFallbacks(connection)

		// Reset connection state
		connection.State = FailoverStateNormal
		connection.CurrentEndpoint = connection.PrimaryEndpoint
		connection.FailureCount = 0
		connection.FailoverAttempts = 0
		connection.HealthScore = 1.0

		atomic.AddInt64(&fm.metrics.SuccessfulRecoveries, 1)

		// Call recovery callback
		onRecovery := connection.OnRecovery
		connection.mu.Unlock()

		if onRecovery != nil {
			go func() {
				if err := onRecovery(); err != nil {
					log.Printf("Recovery callback failed for connection %s: %v", connectionID, err)
				}
			}()
		}

		log.Printf("Successfully recovered connection %s", connectionID)
		return nil
	} else {
		connection.State = FailoverStateFailedOver
		connection.mu.Unlock()
		return fmt.Errorf("recovery failed for connection %s", connectionID)
	}
}

// testPrimaryConnection tests if the primary connection is available
func (fm *FailoverManager) testPrimaryConnection(connection *FailoverConnection) bool {
	// Implementation would test the primary WebSocket endpoint
	// For now, we'll simulate this with a simple check

	connection.mu.RLock()
	endpoint := connection.PrimaryEndpoint
	connection.mu.RUnlock()

	// Simulate connection test
	time.Sleep(100 * time.Millisecond)

	// In a real implementation, you would:
	// 1. Attempt to establish a WebSocket connection
	// 2. Send a ping message
	// 3. Wait for pong response
	// 4. Return true if successful, false otherwise

	log.Printf("Testing primary connection to %s", endpoint)
	return true // Simplified for this implementation
}

// monitoringRoutine monitors connection health and triggers failover
func (fm *FailoverManager) monitoringRoutine() {
	ticker := time.NewTicker(fm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fm.performHealthChecks()
		case <-fm.done:
			return
		}
	}
}

// performHealthChecks performs health checks on all connections
func (fm *FailoverManager) performHealthChecks() {
	fm.mu.RLock()
	connections := make([]*FailoverConnection, 0, len(fm.connections))
	for _, conn := range fm.connections {
		connections = append(connections, conn)
	}
	fm.mu.RUnlock()

	for _, connection := range connections {
		go fm.checkConnectionHealth(connection)
	}
}

// checkConnectionHealth checks the health of a single connection
func (fm *FailoverManager) checkConnectionHealth(connection *FailoverConnection) {
	connection.mu.Lock()
	lastCheck := connection.LastHealthCheck
	connection.LastHealthCheck = time.Now()
	state := connection.State
	connection.mu.Unlock()

	// Skip if recently checked
	if time.Since(lastCheck) < fm.config.HealthCheckInterval/2 {
		return
	}

	// Only check connections in normal state
	if state != FailoverStateNormal {
		return
	}

	// Perform health check (simplified)
	healthy := fm.isConnectionHealthy(connection)

	connection.mu.Lock()
	if healthy {
		connection.HealthScore = 1.0
		connection.FailureCount = 0
	} else {
		connection.HealthScore *= 0.8 // Degrade health score
		connection.FailureCount++
	}
	connection.mu.Unlock()

	// Trigger failover if unhealthy
	if !healthy && fm.failoverStrategies["health_based"].ShouldFailover(connection) {
		fm.executeFailover(connection, fm.failoverStrategies["health_based"])
	}
}

// isConnectionHealthy checks if a connection is healthy
func (fm *FailoverManager) isConnectionHealthy(connection *FailoverConnection) bool {
	// Implementation would check actual connection health
	// For now, we'll simulate this
	return true
}

// recoveryRoutine periodically attempts recovery of failed-over connections
func (fm *FailoverManager) recoveryRoutine() {
	ticker := time.NewTicker(fm.config.RecoveryCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fm.performRecoveryChecks()
		case <-fm.done:
			return
		}
	}
}

// performRecoveryChecks checks for recovery opportunities
func (fm *FailoverManager) performRecoveryChecks() {
	fm.mu.RLock()
	connections := make([]*FailoverConnection, 0, len(fm.connections))
	for _, conn := range fm.connections {
		connections = append(connections, conn)
	}
	fm.mu.RUnlock()

	for _, connection := range connections {
		connection.mu.RLock()
		state := connection.State
		connection.mu.RUnlock()

		if state == FailoverStateFailedOver {
			go func(conn *FailoverConnection) {
				if err := fm.AttemptRecovery(conn.ID); err != nil {
					log.Printf("Recovery attempt failed for connection %s: %v", conn.ID, err)
				}
			}(connection)
		}
	}
}

// updateFailoverMetrics updates failover timing metrics
func (fm *FailoverManager) updateFailoverMetrics(duration time.Duration) {
	if fm.metrics.AverageFailoverTime == 0 {
		fm.metrics.AverageFailoverTime = duration
	} else {
		fm.metrics.AverageFailoverTime = (fm.metrics.AverageFailoverTime + duration) / 2
	}

	if fm.metrics.MinFailoverTime == 0 || duration < fm.metrics.MinFailoverTime {
		fm.metrics.MinFailoverTime = duration
	}

	if duration > fm.metrics.MaxFailoverTime {
		fm.metrics.MaxFailoverTime = duration
	}

	fm.metrics.LastUpdated = time.Now()
}

// GetFailoverStatus returns failover status for a connection
func (fm *FailoverManager) GetFailoverStatus(connectionID string) (*FailoverConnection, error) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	connection, exists := fm.connections[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection %s not found", connectionID)
	}

	// Return a copy to avoid race conditions
	connection.mu.RLock()
	defer connection.mu.RUnlock()

	return &FailoverConnection{
		ID:                 connection.ID,
		PrimaryEndpoint:    connection.PrimaryEndpoint,
		FallbackEndpoints:  connection.FallbackEndpoints,
		CurrentEndpoint:    connection.CurrentEndpoint,
		State:              connection.State,
		FailureCount:       connection.FailureCount,
		LastFailure:        connection.LastFailure,
		FailoverAttempts:   connection.FailoverAttempts,
		ActiveFallback:     connection.ActiveFallback,
		HealthScore:        connection.HealthScore,
		LastHealthCheck:    connection.LastHealthCheck,
		RecoveryInProgress: connection.RecoveryInProgress,
	}, nil
}

// GetFailoverMetrics returns failover metrics
func (fm *FailoverManager) GetFailoverMetrics() *FailoverMetrics {
	return &FailoverMetrics{
		TotalFailovers:       atomic.LoadInt64(&fm.metrics.TotalFailovers),
		SuccessfulFailovers:  atomic.LoadInt64(&fm.metrics.SuccessfulFailovers),
		FailedFailovers:      atomic.LoadInt64(&fm.metrics.FailedFailovers),
		ActiveFallbacks:      atomic.LoadInt64(&fm.metrics.ActiveFallbacks),
		AverageFailoverTime:  fm.metrics.AverageFailoverTime,
		MaxFailoverTime:      fm.metrics.MaxFailoverTime,
		MinFailoverTime:      fm.metrics.MinFailoverTime,
		RecoveryAttempts:     atomic.LoadInt64(&fm.metrics.RecoveryAttempts),
		SuccessfulRecoveries: atomic.LoadInt64(&fm.metrics.SuccessfulRecoveries),
		LastUpdated:          fm.metrics.LastUpdated,
	}
}

// Close stops the failover manager
func (fm *FailoverManager) Close() error {
	close(fm.done)

	// Stop all active fallbacks
	fm.mu.Lock()
	defer fm.mu.Unlock()

	for _, connection := range fm.connections {
		fm.stopFallbacks(connection)
	}

	return nil
}

// Failover strategy implementations

// ThresholdStrategy implements threshold-based failover
type ThresholdStrategy struct {
	FailureThreshold int
}

func (ts *ThresholdStrategy) ShouldFailover(connection *FailoverConnection) bool {
	connection.mu.RLock()
	defer connection.mu.RUnlock()
	return connection.FailureCount >= ts.FailureThreshold
}

func (ts *ThresholdStrategy) SelectFallback(connection *FailoverConnection) (FallbackMethod, error) {
	// Simple selection: prefer HTTP polling
	return FallbackHTTPPolling, nil
}

func (ts *ThresholdStrategy) ExecuteFailover(connection *FailoverConnection, method FallbackMethod) error {
	// Strategy-specific failover logic would go here
	return nil
}

func (ts *ThresholdStrategy) GetName() string {
	return "threshold"
}

// HealthBasedStrategy implements health-based failover
type HealthBasedStrategy struct {
	MinHealthScore float64
}

func (hbs *HealthBasedStrategy) ShouldFailover(connection *FailoverConnection) bool {
	connection.mu.RLock()
	defer connection.mu.RUnlock()
	return connection.HealthScore < hbs.MinHealthScore
}

func (hbs *HealthBasedStrategy) SelectFallback(connection *FailoverConnection) (FallbackMethod, error) {
	// Health-based selection logic
	connection.mu.RLock()
	defer connection.mu.RUnlock()

	if connection.HealthScore > 0.3 {
		return FallbackSSE, nil
	}
	return FallbackHTTPPolling, nil
}

func (hbs *HealthBasedStrategy) ExecuteFailover(connection *FailoverConnection, method FallbackMethod) error {
	return nil
}

func (hbs *HealthBasedStrategy) GetName() string {
	return "health_based"
}

// LatencyBasedStrategy implements latency-based failover
type LatencyBasedStrategy struct {
	MaxLatency time.Duration
}

func (lbs *LatencyBasedStrategy) ShouldFailover(connection *FailoverConnection) bool {
	// Implementation would check actual latency
	return false
}

func (lbs *LatencyBasedStrategy) SelectFallback(connection *FailoverConnection) (FallbackMethod, error) {
	return FallbackHTTPPolling, nil
}

func (lbs *LatencyBasedStrategy) ExecuteFailover(connection *FailoverConnection, method FallbackMethod) error {
	return nil
}

func (lbs *LatencyBasedStrategy) GetName() string {
	return "latency_based"
}
