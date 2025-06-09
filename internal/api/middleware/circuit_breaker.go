// Package middleware provides HTTP middleware for circuit breaker protection.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"lerian-mcp-memory/internal/api/response"
)

// CircuitBreakerManager manages multiple circuit breakers for different services
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	config   CircuitBreakerConfig
	mu       sync.RWMutex
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name            string
	state           CircuitState
	failureCount    int64
	successCount    int64
	lastFailTime    time.Time
	lastSuccessTime time.Time
	config          BreakerConfig
	metrics         *BreakerMetrics
	mu              sync.RWMutex
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreakerConfig represents global circuit breaker configuration
type CircuitBreakerConfig struct {
	Enabled         bool                     `json:"enabled"`
	DefaultSettings BreakerConfig            `json:"default_settings"`
	ServiceConfigs  map[string]BreakerConfig `json:"service_configs"`
	MonitorInterval time.Duration            `json:"monitor_interval"`
	EnableMetrics   bool                     `json:"enable_metrics"`
}

// BreakerConfig represents individual circuit breaker configuration
type BreakerConfig struct {
	FailureThreshold  int             `json:"failure_threshold"`
	SuccessThreshold  int             `json:"success_threshold"`
	Timeout           time.Duration   `json:"timeout"`
	MaxRequests       int64           `json:"max_requests"`
	ResetTimeout      time.Duration   `json:"reset_timeout"`
	BackoffStrategy   BackoffStrategy `json:"backoff_strategy"`
	BackoffMultiplier float64         `json:"backoff_multiplier"`
	MaxBackoffTime    time.Duration   `json:"max_backoff_time"`
}

// BackoffStrategy defines the backoff strategy for circuit breaker
type BackoffStrategy string

const (
	BackoffConstant    BackoffStrategy = "constant"
	BackoffLinear      BackoffStrategy = "linear"
	BackoffExponential BackoffStrategy = "exponential"
)

// BreakerMetrics tracks circuit breaker performance
type BreakerMetrics struct {
	TotalRequests       int64         `json:"total_requests"`
	SuccessfulRequests  int64         `json:"successful_requests"`
	FailedRequests      int64         `json:"failed_requests"`
	RejectedRequests    int64         `json:"rejected_requests"`
	Timeouts            int64         `json:"timeouts"`
	StateChanges        int64         `json:"state_changes"`
	LastStateChange     time.Time     `json:"last_state_change"`
	Uptime              time.Duration `json:"uptime"`
	Downtime            time.Duration `json:"downtime"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	mu                  sync.RWMutex
}

// BreakerResult represents the result of a circuit breaker call
type BreakerResult struct {
	Success      bool          `json:"success"`
	State        CircuitState  `json:"state"`
	Error        error         `json:"error,omitempty"`
	ResponseTime time.Duration `json:"response_time"`
	Rejected     bool          `json:"rejected"`
}

// DefaultCircuitBreakerConfig returns default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Enabled:         true,
		MonitorInterval: 30 * time.Second,
		EnableMetrics:   true,
		DefaultSettings: BreakerConfig{
			FailureThreshold:  5,
			SuccessThreshold:  3,
			Timeout:           30 * time.Second,
			MaxRequests:       100,
			ResetTimeout:      60 * time.Second,
			BackoffStrategy:   BackoffExponential,
			BackoffMultiplier: 2.0,
			MaxBackoffTime:    5 * time.Minute,
		},
		ServiceConfigs: map[string]BreakerConfig{
			"openai": {
				FailureThreshold:  3,
				SuccessThreshold:  2,
				Timeout:           15 * time.Second,
				MaxRequests:       50,
				ResetTimeout:      30 * time.Second,
				BackoffStrategy:   BackoffExponential,
				BackoffMultiplier: 1.5,
				MaxBackoffTime:    2 * time.Minute,
			},
			"claude": {
				FailureThreshold:  3,
				SuccessThreshold:  2,
				Timeout:           20 * time.Second,
				MaxRequests:       30,
				ResetTimeout:      45 * time.Second,
				BackoffStrategy:   BackoffExponential,
				BackoffMultiplier: 1.8,
				MaxBackoffTime:    3 * time.Minute,
			},
			"database": {
				FailureThreshold:  10,
				SuccessThreshold:  5,
				Timeout:           5 * time.Second,
				MaxRequests:       200,
				ResetTimeout:      15 * time.Second,
				BackoffStrategy:   BackoffLinear,
				BackoffMultiplier: 1.2,
				MaxBackoffTime:    1 * time.Minute,
			},
			"qdrant": {
				FailureThreshold:  5,
				SuccessThreshold:  3,
				Timeout:           10 * time.Second,
				MaxRequests:       100,
				ResetTimeout:      30 * time.Second,
				BackoffStrategy:   BackoffExponential,
				BackoffMultiplier: 2.0,
				MaxBackoffTime:    2 * time.Minute,
			},
		},
	}
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(config CircuitBreakerConfig) *CircuitBreakerManager {
	manager := &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		config:   config,
	}

	// Initialize circuit breakers for configured services
	for service, serviceConfig := range config.ServiceConfigs {
		manager.breakers[service] = NewCircuitBreaker(service, serviceConfig)
	}

	// Start monitoring routine
	if config.EnableMetrics {
		go manager.monitorBreakers()
	}

	return manager
}

// GetBreaker returns a circuit breaker for a service, creating it if necessary
func (m *CircuitBreakerManager) GetBreaker(service string) *CircuitBreaker {
	m.mu.RLock()
	breaker, exists := m.breakers[service]
	m.mu.RUnlock()

	if exists {
		return breaker
	}

	// Create new breaker with default config
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check pattern
	if breaker, exists := m.breakers[service]; exists {
		return breaker
	}

	config := m.config.DefaultSettings
	if serviceConfig, hasConfig := m.config.ServiceConfigs[service]; hasConfig {
		config = serviceConfig
	}

	breaker = NewCircuitBreaker(service, config)
	m.breakers[service] = breaker
	return breaker
}

// Middleware returns HTTP middleware for circuit breaker protection
func (m *CircuitBreakerManager) Middleware(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !m.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			breaker := m.GetBreaker(service)

			// Execute with circuit breaker protection
			result := breaker.Execute(func() error {
				next.ServeHTTP(w, r)
				return nil
			})

			if result.Rejected {
				// Circuit breaker is open, return error
				response.WriteError(w, http.StatusServiceUnavailable,
					"Service temporarily unavailable",
					fmt.Sprintf("Circuit breaker is open for service %s. Please try again later.", service))
				return
			}

			if !result.Success && result.Error != nil {
				// Request failed
				response.WriteError(w, http.StatusInternalServerError,
					"Service error", result.Error.Error())
				return
			}
		})
	}
}

// GetAllMetrics returns metrics for all circuit breakers
func (m *CircuitBreakerManager) GetAllMetrics() map[string]*BreakerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := make(map[string]*BreakerMetrics)
	for name, breaker := range m.breakers {
		metrics[name] = breaker.GetMetrics()
	}

	return metrics
}

// Helper methods

func (m *CircuitBreakerManager) monitorBreakers() {
	ticker := time.NewTicker(m.config.MonitorInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.RLock()
		for _, breaker := range m.breakers {
			breaker.updateMetrics()
		}
		m.mu.RUnlock()
	}
}

// CircuitBreaker implementation

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config BreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:    name,
		state:   StateClosed,
		config:  config,
		metrics: NewBreakerMetrics(),
	}
}

// Execute runs a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) *BreakerResult {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Check if request should be allowed
	if !cb.allowRequest() {
		cb.metrics.RejectedRequests++
		return &BreakerResult{
			Success:  false,
			State:    cb.state,
			Rejected: true,
			Error:    fmt.Errorf("circuit breaker is open"),
		}
	}

	// Execute the function
	startTime := time.Now()
	cb.metrics.TotalRequests++

	// Create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), cb.config.Timeout)
	defer cancel()

	// Execute with timeout
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	var err error
	select {
	case err = <-done:
		// Function completed
	case <-ctx.Done():
		// Timeout occurred
		err = fmt.Errorf("operation timed out after %v", cb.config.Timeout)
		cb.metrics.Timeouts++
	}

	responseTime := time.Since(startTime)
	cb.updateAverageResponseTime(responseTime)

	// Record result
	if err != nil {
		cb.onFailure()
		return &BreakerResult{
			Success:      false,
			State:        cb.state,
			Error:        err,
			ResponseTime: responseTime,
		}
	} else {
		cb.onSuccess()
		return &BreakerResult{
			Success:      true,
			State:        cb.state,
			ResponseTime: responseTime,
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() *BreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &BreakerMetrics{
		TotalRequests:       cb.metrics.TotalRequests,
		SuccessfulRequests:  cb.metrics.SuccessfulRequests,
		FailedRequests:      cb.metrics.FailedRequests,
		RejectedRequests:    cb.metrics.RejectedRequests,
		Timeouts:            cb.metrics.Timeouts,
		StateChanges:        cb.metrics.StateChanges,
		LastStateChange:     cb.metrics.LastStateChange,
		Uptime:              cb.metrics.Uptime,
		Downtime:            cb.metrics.Downtime,
		AverageResponseTime: cb.metrics.AverageResponseTime,
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.metrics.StateChanges++
	cb.metrics.LastStateChange = time.Now()
}

// Helper methods

func (cb *CircuitBreaker) allowRequest() bool {
	now := time.Now()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if reset timeout has passed
		if now.Sub(cb.lastFailTime) > cb.getResetTimeout() {
			cb.setState(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		// Allow limited number of requests
		return cb.metrics.TotalRequests < cb.config.MaxRequests
	default:
		return false
	}
}

func (cb *CircuitBreaker) onSuccess() {
	cb.successCount++
	cb.lastSuccessTime = time.Now()
	cb.metrics.SuccessfulRequests++

	switch cb.state {
	case StateHalfOpen:
		if cb.successCount >= int64(cb.config.SuccessThreshold) {
			cb.setState(StateClosed)
			cb.failureCount = 0
		}
	case StateClosed:
		// Reset failure count on success
		cb.failureCount = 0
	}
}

func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailTime = time.Now()
	cb.metrics.FailedRequests++

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= int64(cb.config.FailureThreshold) {
			cb.setState(StateOpen)
		}
	case StateHalfOpen:
		cb.setState(StateOpen)
	}
}

func (cb *CircuitBreaker) setState(newState CircuitState) {
	if cb.state != newState {
		cb.state = newState
		cb.metrics.StateChanges++
		cb.metrics.LastStateChange = time.Now()

		// Reset counters on state change
		if newState == StateClosed {
			cb.failureCount = 0
			cb.successCount = 0
		}
	}
}

func (cb *CircuitBreaker) getResetTimeout() time.Duration {
	baseTimeout := cb.config.ResetTimeout

	switch cb.config.BackoffStrategy {
	case BackoffConstant:
		return baseTimeout
	case BackoffLinear:
		multiplier := float64(cb.failureCount) * cb.config.BackoffMultiplier
		timeout := time.Duration(float64(baseTimeout) * multiplier)
		if timeout > cb.config.MaxBackoffTime {
			timeout = cb.config.MaxBackoffTime
		}
		return timeout
	case BackoffExponential:
		multiplier := 1.0
		for i := int64(0); i < cb.failureCount && multiplier < 1000; i++ {
			multiplier *= cb.config.BackoffMultiplier
		}
		timeout := time.Duration(float64(baseTimeout) * multiplier)
		if timeout > cb.config.MaxBackoffTime {
			timeout = cb.config.MaxBackoffTime
		}
		return timeout
	default:
		return baseTimeout
	}
}

func (cb *CircuitBreaker) updateAverageResponseTime(responseTime time.Duration) {
	// Simple moving average - can be enhanced with more sophisticated calculation
	if cb.metrics.AverageResponseTime == 0 {
		cb.metrics.AverageResponseTime = responseTime
	} else {
		cb.metrics.AverageResponseTime = (cb.metrics.AverageResponseTime + responseTime) / 2
	}
}

func (cb *CircuitBreaker) updateMetrics() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	// Update uptime/downtime
	if cb.state == StateClosed {
		if !cb.metrics.LastStateChange.IsZero() {
			cb.metrics.Uptime += now.Sub(cb.metrics.LastStateChange)
		}
	} else {
		if !cb.metrics.LastStateChange.IsZero() {
			cb.metrics.Downtime += now.Sub(cb.metrics.LastStateChange)
		}
	}
}

// BreakerMetrics constructor

func NewBreakerMetrics() *BreakerMetrics {
	return &BreakerMetrics{
		LastStateChange: time.Now(),
	}
}

// CircuitState string methods

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// BackoffStrategy string methods

func (bs BackoffStrategy) String() string {
	return string(bs)
}
