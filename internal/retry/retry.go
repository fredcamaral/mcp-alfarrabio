// Package retry provides retry mechanisms with exponential backoff
package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Config holds retry configuration
type Config struct {
	MaxAttempts     int           // Maximum number of attempts (0 = unlimited)
	InitialDelay    time.Duration // Initial delay between retries
	MaxDelay        time.Duration // Maximum delay between retries
	Multiplier      float64       // Backoff multiplier
	RandomizeFactor float64       // Jitter factor (0-1)
	RetryIf         func(error) bool // Function to determine if error is retryable
}

// DefaultConfig returns a default retry configuration
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:     3,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        30 * time.Second,
		Multiplier:      2.0,
		RandomizeFactor: 0.1,
		RetryIf:         DefaultRetryIf,
	}
}

// Operation represents a retryable operation
type Operation func(ctx context.Context) error

// Result contains the result of a retry operation
type Result struct {
	Attempts int           // Number of attempts made
	Duration time.Duration // Total duration of all attempts
	Err      error         // Final error (nil if successful)
}

// Retrier provides retry functionality
type Retrier struct {
	config *Config
}

// New creates a new retrier with the given configuration
func New(config *Config) *Retrier {
	if config == nil {
		config = DefaultConfig()
	}
	if config.Multiplier < 1 {
		config.Multiplier = 1
	}
	if config.RandomizeFactor < 0 {
		config.RandomizeFactor = 0
	} else if config.RandomizeFactor > 1 {
		config.RandomizeFactor = 1
	}
	if config.RetryIf == nil {
		config.RetryIf = DefaultRetryIf
	}
	return &Retrier{config: config}
}

// Do executes the operation with retries
func (r *Retrier) Do(ctx context.Context, op Operation) *Result {
	return r.DoWithData(ctx, func(ctx context.Context, _ interface{}) error {
		return op(ctx)
	}, nil)
}

// DoWithData executes the operation with retries and passes data through attempts
func (r *Retrier) DoWithData(ctx context.Context, op func(context.Context, interface{}) error, data interface{}) *Result {
	start := time.Now()
	result := &Result{Attempts: 0}

	var lastErr error
	delay := r.config.InitialDelay

retryLoop:
	for attempt := 1; r.config.MaxAttempts == 0 || attempt <= r.config.MaxAttempts; attempt++ {
		result.Attempts = attempt

		// Check context cancellation
		if err := ctx.Err(); err != nil {
			lastErr = fmt.Errorf("context cancelled: %w", err)
			break
		}

		// Execute operation
		err := op(ctx, data)
		if err == nil {
			// Success
			result.Duration = time.Since(start)
			return result
		}

		lastErr = err

		// Check if we should retry
		if !r.config.RetryIf(err) {
			break
		}

		// Check if this was the last attempt
		if r.config.MaxAttempts > 0 && attempt >= r.config.MaxAttempts {
			break
		}

		// Calculate next delay with jitter
		nextDelay := r.calculateDelay(delay)
		
		// Wait for the delay or context cancellation
		select {
		case <-time.After(nextDelay):
			delay = r.nextDelay(delay)
		case <-ctx.Done():
			lastErr = fmt.Errorf("context cancelled during retry delay: %w", ctx.Err())
			break retryLoop
		}
	}

	result.Duration = time.Since(start)
	result.Err = lastErr
	return result
}

// calculateDelay adds jitter to the delay
func (r *Retrier) calculateDelay(delay time.Duration) time.Duration {
	if r.config.RandomizeFactor == 0 {
		return delay
	}

	// Add randomization
	delta := float64(delay) * r.config.RandomizeFactor
	minDelay := float64(delay) - delta
	maxDelay := float64(delay) + delta

	// Generate random delay between min and max
	randomDelay := minDelay + rand.Float64()*(maxDelay-minDelay)
	return time.Duration(randomDelay)
}

// nextDelay calculates the next delay with exponential backoff
func (r *Retrier) nextDelay(currentDelay time.Duration) time.Duration {
	nextDelay := time.Duration(float64(currentDelay) * r.config.Multiplier)
	if nextDelay > r.config.MaxDelay {
		return r.config.MaxDelay
	}
	return nextDelay
}

// Common error types

// TemporaryError represents a temporary error that should be retried
type TemporaryError struct {
	Err error
}

func (e *TemporaryError) Error() string {
	return fmt.Sprintf("temporary error: %v", e.Err)
}

func (e *TemporaryError) Unwrap() error {
	return e.Err
}

func (e *TemporaryError) Temporary() bool {
	return true
}

// PermanentError represents a permanent error that should not be retried
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return fmt.Sprintf("permanent error: %v", e.Err)
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

// DefaultRetryIf is the default retry predicate
func DefaultRetryIf(err error) bool {
	if err == nil {
		return false
	}

	// Check for temporary error interface
	type temporary interface {
		Temporary() bool
	}
	if te, ok := err.(temporary); ok {
		return te.Temporary()
	}

	// Check for wrapped temporary error
	var tempErr *TemporaryError
	if errors.As(err, &tempErr) {
		return true
	}

	// Check for permanent error
	var permErr *PermanentError
	if errors.As(err, &permErr) {
		return false
	}

	// By default, retry on any error
	return true
}

// Retry helper functions

// Retry executes the operation with default configuration
func Retry(ctx context.Context, op Operation) error {
	r := New(DefaultConfig())
	result := r.Do(ctx, op)
	return result.Err
}

// RetryWithConfig executes the operation with custom configuration
func RetryWithConfig(ctx context.Context, config *Config, op Operation) error {
	r := New(config)
	result := r.Do(ctx, op)
	return result.Err
}

// ExponentialBackoff creates a config with exponential backoff
func ExponentialBackoff(maxAttempts int) *Config {
	return &Config{
		MaxAttempts:     maxAttempts,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        1 * time.Minute,
		Multiplier:      2.0,
		RandomizeFactor: 0.1,
		RetryIf:         DefaultRetryIf,
	}
}

// LinearBackoff creates a config with linear backoff
func LinearBackoff(maxAttempts int, delay time.Duration) *Config {
	return &Config{
		MaxAttempts:     maxAttempts,
		InitialDelay:    delay,
		MaxDelay:        delay * 10,
		Multiplier:      1.0,
		RandomizeFactor: 0,
		RetryIf:         DefaultRetryIf,
	}
}

// FibonacciBackoff creates a config with Fibonacci backoff
func FibonacciBackoff(maxAttempts int) *Config {
	return &Config{
		MaxAttempts:     maxAttempts,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        1 * time.Minute,
		Multiplier:      1.618, // Golden ratio
		RandomizeFactor: 0.1,
		RetryIf:         DefaultRetryIf,
	}
}

// WithMaxAttempts returns a function that modifies max attempts
func WithMaxAttempts(n int) func(*Config) {
	return func(c *Config) {
		c.MaxAttempts = n
	}
}

// WithDelay returns a function that modifies initial delay
func WithDelay(d time.Duration) func(*Config) {
	return func(c *Config) {
		c.InitialDelay = d
	}
}

// WithMaxDelay returns a function that modifies max delay
func WithMaxDelay(d time.Duration) func(*Config) {
	return func(c *Config) {
		c.MaxDelay = d
	}
}

// WithMultiplier returns a function that modifies the multiplier
func WithMultiplier(m float64) func(*Config) {
	return func(c *Config) {
		c.Multiplier = m
	}
}

// WithJitter returns a function that modifies the jitter factor
func WithJitter(j float64) func(*Config) {
	return func(c *Config) {
		c.RandomizeFactor = j
	}
}

// WithRetryIf returns a function that modifies the retry predicate
func WithRetryIf(f func(error) bool) func(*Config) {
	return func(c *Config) {
		c.RetryIf = f
	}
}

// NewConfigWithOptions creates a config with options
func NewConfigWithOptions(opts ...func(*Config)) *Config {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// Backoff strategies

// ConstantBackoff implements constant delay between retries
type ConstantBackoff struct {
	Delay time.Duration
}

func (b *ConstantBackoff) Next(current time.Duration) time.Duration {
	return b.Delay
}

// ExponentialBackoffStrategy implements exponential backoff
type ExponentialBackoffStrategy struct {
	Multiplier float64
	Max        time.Duration
}

func (b *ExponentialBackoffStrategy) Next(current time.Duration) time.Duration {
	next := time.Duration(float64(current) * b.Multiplier)
	if next > b.Max {
		return b.Max
	}
	return next
}

// PolynomialBackoff implements polynomial backoff
type PolynomialBackoff struct {
	Exponent float64
	Max      time.Duration
}

func (b *PolynomialBackoff) Next(current time.Duration) time.Duration {
	next := time.Duration(math.Pow(float64(current), b.Exponent))
	if next > b.Max {
		return b.Max
	}
	return next
}