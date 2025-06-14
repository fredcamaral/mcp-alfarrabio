// Package websocket provides connection recovery mechanisms for WebSocket clients
package websocket

import (
	"context"
	"errors"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// RecoveryManager handles automatic connection recovery with exponential backoff
type RecoveryManager struct {
	mu                 sync.RWMutex
	config             *RecoveryConfig
	clients            map[string]*RecoverableClient
	metrics            *RecoveryMetrics
	done               chan struct{}
	recoveryQueue      chan *RecoveryRequest
	maxConcurrentRetry int
	activeRetries      int64
}

// RecoveryConfig configures connection recovery behavior
type RecoveryConfig struct {
	MaxRetries        int           `json:"max_retries" yaml:"max_retries"`
	InitialBackoff    time.Duration `json:"initial_backoff" yaml:"initial_backoff"`
	MaxBackoff        time.Duration `json:"max_backoff" yaml:"max_backoff"`
	BackoffMultiplier float64       `json:"backoff_multiplier" yaml:"backoff_multiplier"`
	Jitter            bool          `json:"jitter" yaml:"jitter"`
	HealthCheckPeriod time.Duration `json:"health_check_period" yaml:"health_check_period"`
	RecoveryTimeout   time.Duration `json:"recovery_timeout" yaml:"recovery_timeout"`
	EnabledByDefault  bool          `json:"enabled_by_default" yaml:"enabled_by_default"`
}

// RecoverableClient represents a WebSocket client that can be recovered
type RecoverableClient struct {
	ID                string
	Connection        *websocket.Conn
	URL               string
	Headers           map[string]string
	LastSeen          time.Time
	RetryCount        int
	State             RecoveryClientState
	HealthScore       float64
	RecoveryEnabled   bool
	Metadata          map[string]interface{}
	mu                sync.RWMutex
	recoveryCtx       context.Context
	recoveryCancel    context.CancelFunc
	backoffDuration   time.Duration
	lastRecoveryStart time.Time
}

// RecoveryRequest represents a request to recover a connection
type RecoveryRequest struct {
	ClientID    string
	Priority    RecoveryPriority
	RequestedAt time.Time
	Context     context.Context
	Callback    func(success bool, err error)
}

// RecoveryMetrics tracks recovery performance
type RecoveryMetrics struct {
	TotalRecoveries      int64         `json:"total_recoveries"`
	SuccessfulRecoveries int64         `json:"successful_recoveries"`
	FailedRecoveries     int64         `json:"failed_recoveries"`
	AverageRecoveryTime  time.Duration `json:"average_recovery_time"`
	MaxRecoveryTime      time.Duration `json:"max_recovery_time"`
	MinRecoveryTime      time.Duration `json:"min_recovery_time"`
	ActiveRecoveries     int64         `json:"active_recoveries"`
	QueueLength          int           `json:"queue_length"`
}

// RecoveryClientState represents the state of a recoverable client
type RecoveryClientState int

const (
	RecoveryStateConnected RecoveryClientState = iota
	RecoveryStateDisconnected
	RecoveryStateReconnecting
	RecoveryStateFailed
	RecoveryStateDraining
)

// RecoveryPriority defines recovery priority levels
type RecoveryPriority int

const (
	PriorityLow RecoveryPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// NewRecoveryManager creates a new connection recovery manager
func NewRecoveryManager(config *RecoveryConfig) *RecoveryManager {
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	rm := &RecoveryManager{
		config:             config,
		clients:            make(map[string]*RecoverableClient),
		metrics:            &RecoveryMetrics{},
		done:               make(chan struct{}),
		recoveryQueue:      make(chan *RecoveryRequest, 1000),
		maxConcurrentRetry: 10,
	}

	// Start recovery workers
	for i := 0; i < rm.maxConcurrentRetry; i++ {
		go rm.recoveryWorker()
	}

	// Start health checker
	go rm.healthChecker()

	return rm
}

// DefaultRecoveryConfig returns default recovery configuration
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		MaxRetries:        5,
		InitialBackoff:    time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            true,
		HealthCheckPeriod: 30 * time.Second,
		RecoveryTimeout:   5 * time.Minute,
		EnabledByDefault:  true,
	}
}

// RegisterClient registers a client for recovery management
func (rm *RecoveryManager) RegisterClient(id, url string, conn *websocket.Conn, headers map[string]string) error {
	if id == "" {
		return errors.New("client ID cannot be empty")
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	client := &RecoverableClient{
		ID:              id,
		Connection:      conn,
		URL:             url,
		Headers:         headers,
		LastSeen:        time.Now(),
		State:           RecoveryStateConnected,
		HealthScore:     1.0,
		RecoveryEnabled: rm.config.EnabledByDefault,
		Metadata:        make(map[string]interface{}),
		recoveryCtx:     ctx,
		recoveryCancel:  cancel,
		backoffDuration: rm.config.InitialBackoff,
	}

	rm.clients[id] = client
	log.Printf("Registered client %s for recovery management", id)

	return nil
}

// UnregisterClient removes a client from recovery management
func (rm *RecoveryManager) UnregisterClient(id string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if client, exists := rm.clients[id]; exists {
		client.mu.Lock()
		client.State = RecoveryStateDraining
		if client.recoveryCancel != nil {
			client.recoveryCancel()
		}
		client.mu.Unlock()

		delete(rm.clients, id)
		log.Printf("Unregistered client %s from recovery management", id)
	}
}

// HandleDisconnection handles a client disconnection and initiates recovery if enabled
func (rm *RecoveryManager) HandleDisconnection(clientID string, err error) {
	rm.mu.RLock()
	client, exists := rm.clients[clientID]
	rm.mu.RUnlock()

	if !exists {
		return
	}

	client.mu.Lock()
	if !client.RecoveryEnabled || client.State == RecoveryStateDraining {
		client.mu.Unlock()
		return
	}

	client.State = RecoveryStateDisconnected
	client.Connection = nil
	client.mu.Unlock()

	log.Printf("Client %s disconnected: %v, initiating recovery", clientID, err)

	// Queue recovery request
	request := &RecoveryRequest{
		ClientID:    clientID,
		Priority:    PriorityNormal,
		RequestedAt: time.Now(),
		Context:     client.recoveryCtx,
		Callback: func(success bool, recoveryErr error) {
			if success {
				log.Printf("Client %s recovered successfully", clientID)
			} else {
				log.Printf("Client %s recovery failed: %v", clientID, recoveryErr)
			}
		},
	}

	select {
	case rm.recoveryQueue <- request:
	default:
		log.Printf("Recovery queue full, dropping recovery request for client %s", clientID)
	}
}

// recoveryWorker processes recovery requests
func (rm *RecoveryManager) recoveryWorker() {
	for {
		select {
		case request := <-rm.recoveryQueue:
			rm.processRecoveryRequest(request)
		case <-rm.done:
			return
		}
	}
}

// processRecoveryRequest processes a single recovery request
func (rm *RecoveryManager) processRecoveryRequest(request *RecoveryRequest) {
	atomic.AddInt64(&rm.activeRetries, 1)
	defer atomic.AddInt64(&rm.activeRetries, -1)

	start := time.Now()
	success := rm.attemptRecovery(request)

	// Update metrics
	atomic.AddInt64(&rm.metrics.TotalRecoveries, 1)
	if success {
		atomic.AddInt64(&rm.metrics.SuccessfulRecoveries, 1)
	} else {
		atomic.AddInt64(&rm.metrics.FailedRecoveries, 1)
	}

	duration := time.Since(start)
	rm.updateRecoveryTimeMetrics(duration)

	if request.Callback != nil {
		var err error
		if !success {
			err = errors.New("recovery failed after all retries")
		}
		request.Callback(success, err)
	}
}

// attemptRecovery attempts to recover a client connection
func (rm *RecoveryManager) attemptRecovery(request *RecoveryRequest) bool {
	rm.mu.RLock()
	client, exists := rm.clients[request.ClientID]
	rm.mu.RUnlock()

	if !exists {
		return false
	}

	client.mu.Lock()
	if client.State == RecoveryStateDraining {
		client.mu.Unlock()
		return false
	}

	client.State = RecoveryStateReconnecting
	client.lastRecoveryStart = time.Now()
	client.mu.Unlock()

	// Attempt recovery with exponential backoff
	for attempt := 0; attempt < rm.config.MaxRetries; attempt++ {
		select {
		case <-request.Context.Done():
			rm.markRecoveryFailed(client)
			return false
		default:
		}

		if attempt > 0 {
			backoff := rm.calculateBackoff(client, attempt)
			log.Printf("Client %s recovery attempt %d, waiting %v", client.ID, attempt+1, backoff)

			timer := time.NewTimer(backoff)
			select {
			case <-timer.C:
			case <-request.Context.Done():
				timer.Stop()
				rm.markRecoveryFailed(client)
				return false
			}
		}

		if rm.tryReconnect(client) {
			rm.markRecoverySuccessful(client)
			return true
		}
	}

	rm.markRecoveryFailed(client)
	return false
}

// tryReconnect attempts to reconnect a single client
func (rm *RecoveryManager) tryReconnect(client *RecoverableClient) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.DialContext(ctx, client.URL, nil)
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		log.Printf("Reconnection failed for client %s: %v", client.ID, err)
		return false
	}

	client.mu.Lock()
	client.Connection = conn
	client.LastSeen = time.Now()
	client.RetryCount = 0
	client.backoffDuration = rm.config.InitialBackoff
	client.mu.Unlock()

	log.Printf("Client %s reconnected successfully", client.ID)
	return true
}

// calculateBackoff calculates backoff duration with exponential backoff and jitter
func (rm *RecoveryManager) calculateBackoff(client *RecoverableClient, attempt int) time.Duration {
	client.mu.Lock()
	defer client.mu.Unlock()

	// Exponential backoff
	backoff := float64(rm.config.InitialBackoff) * math.Pow(rm.config.BackoffMultiplier, float64(attempt))

	// Cap at max backoff
	if backoff > float64(rm.config.MaxBackoff) {
		backoff = float64(rm.config.MaxBackoff)
	}

	duration := time.Duration(backoff)

	// Add jitter if enabled
	if rm.config.Jitter {
		jitter := time.Duration(float64(duration) * 0.1 * (2.0*float64(time.Now().UnixNano()%1000)/1000.0 - 1.0))
		duration += jitter
	}

	client.backoffDuration = duration
	return duration
}

// markRecoverySuccessful marks a client recovery as successful
func (rm *RecoveryManager) markRecoverySuccessful(client *RecoverableClient) {
	client.mu.Lock()
	defer client.mu.Unlock()

	client.State = RecoveryStateConnected
	client.HealthScore = 1.0
	client.RetryCount = 0
	client.backoffDuration = rm.config.InitialBackoff
}

// markRecoveryFailed marks a client recovery as failed
func (rm *RecoveryManager) markRecoveryFailed(client *RecoverableClient) {
	client.mu.Lock()
	defer client.mu.Unlock()

	client.State = RecoveryStateFailed
	client.HealthScore = 0.0
	client.RetryCount++

	log.Printf("Client %s recovery failed after %d attempts", client.ID, client.RetryCount)
}

// healthChecker periodically checks client health
func (rm *RecoveryManager) healthChecker() {
	ticker := time.NewTicker(rm.config.HealthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.performHealthChecks()
		case <-rm.done:
			return
		}
	}
}

// performHealthChecks performs health checks on all clients
func (rm *RecoveryManager) performHealthChecks() {
	rm.mu.RLock()
	clients := make([]*RecoverableClient, 0, len(rm.clients))
	for _, client := range rm.clients {
		clients = append(clients, client)
	}
	rm.mu.RUnlock()

	for _, client := range clients {
		go rm.checkClientHealth(client)
	}
}

// checkClientHealth checks the health of a single client
func (rm *RecoveryManager) checkClientHealth(client *RecoverableClient) {
	client.mu.RLock()
	conn := client.Connection
	state := client.State
	client.mu.RUnlock()

	if conn == nil || state != RecoveryStateConnected {
		return
	}

	// Send ping to check connection health
	start := time.Now()
	err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second))
	if err != nil {
		log.Printf("Health check ping failed for client %s: %v", client.ID, err)
		rm.HandleDisconnection(client.ID, err)
		return
	}

	// Update health metrics
	latency := time.Since(start)
	rm.updateClientHealth(client, latency, nil)
}

// updateClientHealth updates client health score based on latency and errors
func (rm *RecoveryManager) updateClientHealth(client *RecoverableClient, latency time.Duration, err error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	client.LastSeen = time.Now()

	if err != nil {
		client.HealthScore = math.Max(0.0, client.HealthScore-0.1)
		return
	}

	// Calculate health score based on latency
	var latencyScore float64
	switch {
	case latency < 50*time.Millisecond:
		latencyScore = 1.0
	case latency < 100*time.Millisecond:
		latencyScore = 0.9
	case latency < 200*time.Millisecond:
		latencyScore = 0.8
	case latency < 500*time.Millisecond:
		latencyScore = 0.6
	case latency < time.Second:
		latencyScore = 0.4
	default:
		latencyScore = 0.2
	}

	// Smooth health score changes
	client.HealthScore = 0.8*client.HealthScore + 0.2*latencyScore
}

// updateRecoveryTimeMetrics updates recovery time metrics
func (rm *RecoveryManager) updateRecoveryTimeMetrics(duration time.Duration) {
	// Update average (simple moving average)
	if rm.metrics.AverageRecoveryTime == 0 {
		rm.metrics.AverageRecoveryTime = duration
	} else {
		rm.metrics.AverageRecoveryTime = (rm.metrics.AverageRecoveryTime + duration) / 2
	}

	// Update min/max
	if rm.metrics.MinRecoveryTime == 0 || duration < rm.metrics.MinRecoveryTime {
		rm.metrics.MinRecoveryTime = duration
	}
	if duration > rm.metrics.MaxRecoveryTime {
		rm.metrics.MaxRecoveryTime = duration
	}
}

// GetMetrics returns current recovery metrics
func (rm *RecoveryManager) GetMetrics() *RecoveryMetrics {
	metrics := &RecoveryMetrics{
		TotalRecoveries:      atomic.LoadInt64(&rm.metrics.TotalRecoveries),
		SuccessfulRecoveries: atomic.LoadInt64(&rm.metrics.SuccessfulRecoveries),
		FailedRecoveries:     atomic.LoadInt64(&rm.metrics.FailedRecoveries),
		AverageRecoveryTime:  rm.metrics.AverageRecoveryTime,
		MaxRecoveryTime:      rm.metrics.MaxRecoveryTime,
		MinRecoveryTime:      rm.metrics.MinRecoveryTime,
		ActiveRecoveries:     atomic.LoadInt64(&rm.activeRetries),
		QueueLength:          len(rm.recoveryQueue),
	}

	return metrics
}

// GetClientStatus returns status for all registered clients
func (rm *RecoveryManager) GetClientStatus() map[string]ClientInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	status := make(map[string]ClientInfo)
	for id, client := range rm.clients {
		client.mu.RLock()
		status[id] = ClientInfo{
			ID:              client.ID,
			State:           client.State,
			HealthScore:     client.HealthScore,
			LastSeen:        client.LastSeen,
			RetryCount:      client.RetryCount,
			RecoveryEnabled: client.RecoveryEnabled,
			BackoffDuration: client.backoffDuration,
		}
		client.mu.RUnlock()
	}

	return status
}

// ClientInfo provides client status information
type ClientInfo struct {
	ID              string              `json:"id"`
	State           RecoveryClientState `json:"state"`
	HealthScore     float64             `json:"health_score"`
	LastSeen        time.Time           `json:"last_seen"`
	RetryCount      int                 `json:"retry_count"`
	RecoveryEnabled bool                `json:"recovery_enabled"`
	BackoffDuration time.Duration       `json:"backoff_duration"`
}

// String returns string representation of RecoveryClientState
func (cs RecoveryClientState) String() string {
	switch cs {
	case RecoveryStateConnected:
		return "connected"
	case RecoveryStateDisconnected:
		return "disconnected"
	case RecoveryStateReconnecting:
		return "reconnecting"
	case RecoveryStateFailed:
		return "failed"
	case RecoveryStateDraining:
		return "draining"
	default:
		return "unknown"
	}
}

// Close stops the recovery manager
func (rm *RecoveryManager) Close() error {
	close(rm.done)

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Cancel all recovery contexts
	for _, client := range rm.clients {
		client.mu.Lock()
		if client.recoveryCancel != nil {
			client.recoveryCancel()
		}
		client.mu.Unlock()
	}

	return nil
}
