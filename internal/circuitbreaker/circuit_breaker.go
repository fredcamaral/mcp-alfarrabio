// Package circuitbreaker provides circuit breaker pattern implementation
package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

// State represents the circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
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

// Config holds circuit breaker configuration
type Config struct {
	// FailureThreshold is the number of failures before opening the circuit
	FailureThreshold int
	// SuccessThreshold is the number of successes in half-open state before closing
	SuccessThreshold int
	// Timeout is the duration the circuit stays open before switching to half-open
	Timeout time.Duration
	// MaxConcurrentRequests in half-open state
	MaxConcurrentRequests int
	// OnStateChange is called when the circuit state changes
	OnStateChange func(from, to State)
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		FailureThreshold:      5,
		SuccessThreshold:      2,
		Timeout:               30 * time.Second,
		MaxConcurrentRequests: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config *Config
	
	state           int32 // atomic State
	lastFailureTime int64 // atomic time.Time as unix nano
	
	consecutiveFailures int32
	consecutiveSuccesses int32
	halfOpenRequests    int32
	
	totalRequests   int64
	totalFailures   int64
	totalSuccesses  int64
	totalRejections int64
}

// New creates a new circuit breaker
func New(config *Config) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &CircuitBreaker{
		config: config,
		state:  int32(StateClosed),
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	return cb.ExecuteWithFallback(ctx, fn, nil)
}

// ExecuteWithFallback runs the function with circuit breaker protection and fallback
func (cb *CircuitBreaker) ExecuteWithFallback(ctx context.Context, fn func(context.Context) error, fallback func(context.Context, error) error) error {
	cbErr := cb.canExecute()
	if cbErr != nil {
		atomic.AddInt64(&cb.totalRejections, 1)
		if fallback != nil {
			return fallback(ctx, cbErr)
		}
		return cbErr
	}
	
	atomic.AddInt64(&cb.totalRequests, 1)
	
	// Execute the function
	err := fn(ctx)
	
	// Record the result
	cb.recordResult(err)
	
	if err != nil && fallback != nil {
		return fallback(ctx, err)
	}
	
	return err
}

// canExecute checks if a request can be executed
func (cb *CircuitBreaker) canExecute() error {
	state := cb.getState()
	
	switch state {
	case StateClosed:
		return nil
		
	case StateOpen:
		// Check if we should transition to half-open
		if cb.shouldTransitionToHalfOpen() {
			cb.transitionTo(StateHalfOpen)
			return nil
		}
		return ErrCircuitOpen
		
	case StateHalfOpen:
		// Limit concurrent requests in half-open state
		current := atomic.AddInt32(&cb.halfOpenRequests, 1)
		if current > int32(cb.config.MaxConcurrentRequests) {
			atomic.AddInt32(&cb.halfOpenRequests, -1)
			return ErrTooManyConcurrentRequests
		}
		return nil
		
	default:
		return fmt.Errorf("unknown circuit breaker state: %v", state)
	}
}

// recordResult records the result of a request
func (cb *CircuitBreaker) recordResult(err error) {
	state := cb.getState()
	
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	
	// Decrement half-open counter if needed
	if state == StateHalfOpen {
		atomic.AddInt32(&cb.halfOpenRequests, -1)
	}
}

// recordSuccess records a successful request
func (cb *CircuitBreaker) recordSuccess() {
	atomic.AddInt64(&cb.totalSuccesses, 1)
	
	state := cb.getState()
	switch state {
	case StateClosed:
		// Reset consecutive failures
		atomic.StoreInt32(&cb.consecutiveFailures, 0)
		
	case StateHalfOpen:
		successes := atomic.AddInt32(&cb.consecutiveSuccesses, 1)
		if successes >= int32(cb.config.SuccessThreshold) {
			cb.transitionTo(StateClosed)
		}
	case StateOpen:
		// In open state, successes don't affect state transitions
		// The state will transition after timeout period
	}
}

// recordFailure records a failed request
func (cb *CircuitBreaker) recordFailure() {
	atomic.AddInt64(&cb.totalFailures, 1)
	atomic.StoreInt64(&cb.lastFailureTime, time.Now().UnixNano())
	
	state := cb.getState()
	switch state {
	case StateClosed:
		failures := atomic.AddInt32(&cb.consecutiveFailures, 1)
		if failures >= int32(cb.config.FailureThreshold) {
			cb.transitionTo(StateOpen)
		}
	case StateOpen:
		// Already open, no action needed
	
		
	case StateHalfOpen:
		// Any failure in half-open state reopens the circuit
		cb.transitionTo(StateOpen)
	}
}

// shouldTransitionToHalfOpen checks if the circuit should transition from open to half-open
func (cb *CircuitBreaker) shouldTransitionToHalfOpen() bool {
	lastFailure := atomic.LoadInt64(&cb.lastFailureTime)
	if lastFailure == 0 {
		return true
	}
	
	elapsed := time.Since(time.Unix(0, lastFailure))
	return elapsed >= cb.config.Timeout
}

// transitionTo transitions to a new state
func (cb *CircuitBreaker) transitionTo(newState State) {
	oldState := State(atomic.SwapInt32(&cb.state, int32(newState)))
	
	if oldState == newState {
		return
	}
	
	// Reset counters based on transition
	switch newState {
	case StateClosed:
		atomic.StoreInt32(&cb.consecutiveFailures, 0)
		atomic.StoreInt32(&cb.consecutiveSuccesses, 0)
		
	case StateOpen:
		atomic.StoreInt32(&cb.consecutiveSuccesses, 0)
		
	case StateHalfOpen:
		atomic.StoreInt32(&cb.consecutiveSuccesses, 0)
		atomic.StoreInt32(&cb.halfOpenRequests, 0)
	}
	
	// Notify state change
	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(oldState, newState)
	}
}

// getState returns the current state
func (cb *CircuitBreaker) getState() State {
	return State(atomic.LoadInt32(&cb.state))
}

// GetState returns the current state (public)
func (cb *CircuitBreaker) GetState() State {
	return cb.getState()
}

// Stats holds circuit breaker statistics
type Stats struct {
	State             State
	TotalRequests     int64
	TotalFailures     int64
	TotalSuccesses    int64
	TotalRejections   int64
	FailureRate       float64
	LastFailureTime   time.Time
	ConsecutiveErrors int32
}

// GetStats returns current statistics
func (cb *CircuitBreaker) GetStats() Stats {
	requests := atomic.LoadInt64(&cb.totalRequests)
	failures := atomic.LoadInt64(&cb.totalFailures)
	
	var failureRate float64
	if requests > 0 {
		failureRate = float64(failures) / float64(requests)
	}
	
	lastFailureNano := atomic.LoadInt64(&cb.lastFailureTime)
	var lastFailureTime time.Time
	if lastFailureNano > 0 {
		lastFailureTime = time.Unix(0, lastFailureNano)
	}
	
	return Stats{
		State:             cb.getState(),
		TotalRequests:     requests,
		TotalFailures:     failures,
		TotalSuccesses:    atomic.LoadInt64(&cb.totalSuccesses),
		TotalRejections:   atomic.LoadInt64(&cb.totalRejections),
		FailureRate:       failureRate,
		LastFailureTime:   lastFailureTime,
		ConsecutiveErrors: atomic.LoadInt32(&cb.consecutiveFailures),
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	atomic.StoreInt32(&cb.state, int32(StateClosed))
	atomic.StoreInt32(&cb.consecutiveFailures, 0)
	atomic.StoreInt32(&cb.consecutiveSuccesses, 0)
	atomic.StoreInt32(&cb.halfOpenRequests, 0)
	atomic.StoreInt64(&cb.lastFailureTime, 0)
}

// Errors
var (
	ErrCircuitOpen               = errors.New("circuit breaker is open")
	ErrTooManyConcurrentRequests = errors.New("too many concurrent requests in half-open state")
)