// Package reliability provides circuit breaker and failover mechanisms
// for external service calls to ensure system stability
package reliability

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	StateClosed   CircuitState = iota // Normal operation
	StateHalfOpen                     // Testing if service recovered
	StateOpen                         // Service is down, requests are failing fast
)

// String returns string representation of circuit state
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	Name                   string        // Name for monitoring/logging
	MaxFailures            int           // Max failures before opening circuit
	ResetTimeout           time.Duration // Time to wait before trying half-open
	SuccessThreshold       int           // Consecutive successes needed to close circuit
	RequestVolumeThreshold int           // Minimum requests needed before evaluating failure rate
	FailureRateThreshold   float64       // Failure rate (0.0-1.0) that triggers opening
	Timeout                time.Duration // Timeout for individual requests
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig(name string) *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		Name:                   name,
		MaxFailures:            5,
		ResetTimeout:           30 * time.Second,
		SuccessThreshold:       3,
		RequestVolumeThreshold: 10,
		FailureRateThreshold:   0.5, // 50% failure rate
		Timeout:                10 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern for protecting external service calls
type CircuitBreaker struct {
	config *CircuitBreakerConfig
	mutex  sync.RWMutex

	state           CircuitState
	failures        int
	successes       int
	lastFailureTime time.Time
	lastSuccessTime time.Time

	// Request tracking for failure rate calculation
	requests      []requestResult
	requestWindow time.Duration
}

// requestResult tracks individual request outcomes
type requestResult struct {
	timestamp time.Time
	success   bool
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig("default")
	}

	return &CircuitBreaker{
		config:        config,
		state:         StateClosed,
		requests:      make([]requestResult, 0),
		requestWindow: 60 * time.Second, // Track requests over 1 minute window
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Check if circuit is open
	if cb.shouldRejectRequest() {
		return &CircuitBreakerError{
			State:   cb.GetState(),
			Message: fmt.Sprintf("circuit breaker '%s' is %s", cb.config.Name, cb.state),
		}
	}

	// Create timeout context if configured
	execCtx := ctx
	if cb.config.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, cb.config.Timeout)
		defer cancel()
	}

	// Execute the function
	err := fn(execCtx)

	// Record the result
	cb.recordResult(err == nil)

	return err
}

// shouldRejectRequest determines if the request should be rejected based on circuit state
func (cb *CircuitBreaker) shouldRejectRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		return false

	case StateOpen:
		// Check if we should transition to half-open
		if now.Sub(cb.lastFailureTime) >= cb.config.ResetTimeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			return false
		}
		return true

	case StateHalfOpen:
		return false

	default:
		return false
	}
}

// recordResult records the outcome of a request and updates circuit state
func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	// Add to request tracking
	cb.requests = append(cb.requests, requestResult{
		timestamp: now,
		success:   success,
	})

	// Clean old requests outside the window
	cb.cleanOldRequests(now)

	if success {
		cb.handleSuccess(now)
	} else {
		cb.handleFailure(now)
	}

	// Update state based on current conditions
	cb.updateState(now)
}

// handleSuccess processes a successful request
func (cb *CircuitBreaker) handleSuccess(now time.Time) {
	cb.lastSuccessTime = now
	cb.successes++

	// Reset failure count on success in closed state
	if cb.state == StateClosed {
		cb.failures = 0
	}
}

// handleFailure processes a failed request
func (cb *CircuitBreaker) handleFailure(now time.Time) {
	cb.lastFailureTime = now
	cb.failures++
	cb.successes = 0 // Reset success count on any failure
}

// updateState updates the circuit breaker state based on current metrics
func (cb *CircuitBreaker) updateState(now time.Time) {
	switch cb.state {
	case StateClosed:
		// Check if we should open the circuit
		if cb.shouldOpenCircuit() {
			cb.state = StateOpen
			cb.lastFailureTime = now
		}

	case StateHalfOpen:
		// Check if we should close the circuit (enough consecutive successes)
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
		} else if cb.failures > 0 {
			// Any failure in half-open state opens the circuit again
			cb.state = StateOpen
			cb.lastFailureTime = now
		}

	case StateOpen:
		// State transition handled in shouldRejectRequest
		break
	}
}

// shouldOpenCircuit determines if the circuit should be opened based on failure metrics
func (cb *CircuitBreaker) shouldOpenCircuit() bool {
	// Check simple failure count threshold
	if cb.failures >= cb.config.MaxFailures {
		return true
	}

	// Check failure rate if we have enough requests
	if len(cb.requests) >= cb.config.RequestVolumeThreshold {
		failureRate := cb.calculateFailureRate()
		if failureRate >= cb.config.FailureRateThreshold {
			return true
		}
	}

	return false
}

// calculateFailureRate calculates the current failure rate within the request window
func (cb *CircuitBreaker) calculateFailureRate() float64 {
	if len(cb.requests) == 0 {
		return 0.0
	}

	failures := 0
	for _, req := range cb.requests {
		if !req.success {
			failures++
		}
	}

	return float64(failures) / float64(len(cb.requests))
}

// cleanOldRequests removes requests outside the tracking window
func (cb *CircuitBreaker) cleanOldRequests(now time.Time) {
	cutoff := now.Add(-cb.requestWindow)

	// Find first request within window
	start := 0
	for i, req := range cb.requests {
		if req.timestamp.After(cutoff) {
			start = i
			break
		}
		start = len(cb.requests) // All requests are old
	}

	// Keep only recent requests
	if start > 0 {
		if start >= len(cb.requests) {
			cb.requests = cb.requests[:0] // Clear all
		} else {
			cb.requests = cb.requests[start:]
		}
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetMetrics returns current circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() *CircuitBreakerMetrics {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	now := time.Now()

	return &CircuitBreakerMetrics{
		Name:             cb.config.Name,
		State:            cb.state,
		Failures:         cb.failures,
		Successes:        cb.successes,
		RequestsInWindow: len(cb.requests),
		FailureRate:      cb.calculateFailureRate(),
		LastFailureTime:  cb.lastFailureTime,
		LastSuccessTime:  cb.lastSuccessTime,
		TimeSinceLastFailure: func() time.Duration {
			if cb.lastFailureTime.IsZero() {
				return 0
			}
			return now.Sub(cb.lastFailureTime)
		}(),
		IsHealthy: cb.state == StateClosed,
	}
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.requests = cb.requests[:0]
}

// CircuitBreakerMetrics contains metrics about circuit breaker performance
type CircuitBreakerMetrics struct {
	Name                 string        `json:"name"`
	State                CircuitState  `json:"state"`
	Failures             int           `json:"failures"`
	Successes            int           `json:"successes"`
	RequestsInWindow     int           `json:"requests_in_window"`
	FailureRate          float64       `json:"failure_rate"`
	LastFailureTime      time.Time     `json:"last_failure_time"`
	LastSuccessTime      time.Time     `json:"last_success_time"`
	TimeSinceLastFailure time.Duration `json:"time_since_last_failure"`
	IsHealthy            bool          `json:"is_healthy"`
}

// CircuitBreakerError represents an error when circuit breaker rejects a request
type CircuitBreakerError struct {
	State   CircuitState
	Message string
}

func (e *CircuitBreakerError) Error() string {
	return e.Message
}

// IsCircuitBreakerError checks if an error is a circuit breaker error
func IsCircuitBreakerError(err error) bool {
	var cbErr *CircuitBreakerError
	return errors.As(err, &cbErr)
}

// CircuitBreakerManager manages multiple circuit breakers for different services
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	if cb, exists := cbm.breakers[name]; exists {
		return cb
	}

	if config == nil {
		config = DefaultConfig(name)
	}

	cb := NewCircuitBreaker(config)
	cbm.breakers[name] = cb
	return cb
}

// GetMetrics returns metrics for all circuit breakers
func (cbm *CircuitBreakerManager) GetMetrics() map[string]*CircuitBreakerMetrics {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	metrics := make(map[string]*CircuitBreakerMetrics)
	for name, cb := range cbm.breakers {
		metrics[name] = cb.GetMetrics()
	}

	return metrics
}

// ResetAll resets all circuit breakers
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	for _, cb := range cbm.breakers {
		cb.Reset()
	}
}
