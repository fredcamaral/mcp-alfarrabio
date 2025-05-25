package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Common errors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// RateLimitConfig contains configuration for rate limiting
type RateLimitConfig struct {
	// Token bucket configuration
	Rate           float64      // Tokens per second
	Burst          int          // Maximum burst size
	
	// Limiter configuration
	PerUser        bool         // Apply rate limit per user
	PerIP          bool         // Apply rate limit per IP
	Global         bool         // Apply global rate limit
	
	// Cleanup configuration
	CleanupInterval time.Duration // How often to clean up expired limiters
	TTL             time.Duration // Time to live for inactive limiters
	
	// General configuration
	Logger         *slog.Logger
}

// DefaultRateLimitConfig returns a default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Rate:            10.0,  // 10 requests per second
		Burst:           20,    // Allow burst of 20
		PerUser:         true,
		PerIP:           false,
		Global:          false,
		CleanupInterval: 5 * time.Minute,
		TTL:             30 * time.Minute,
		Logger:          slog.Default(),
	}
}

// TokenBucket implements the token bucket algorithm
type TokenBucket struct {
	rate       float64       // Tokens per second
	burst      int           // Maximum tokens
	tokens     float64       // Current tokens
	lastUpdate time.Time     // Last update time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(rate float64, burst int) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst), // Start with full bucket
		lastUpdate: time.Now(),
	}
}

// Allow checks if n tokens can be consumed
func (tb *TokenBucket) Allow(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	return tb.allowN(time.Now(), n)
}

// AllowN checks if n tokens can be consumed at a specific time
func (tb *TokenBucket) AllowN(now time.Time, n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	return tb.allowN(now, n)
}

// allowN is the internal implementation (must be called with lock held)
func (tb *TokenBucket) allowN(now time.Time, n int) bool {
	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.tokens += elapsed * tb.rate
	
	// Cap tokens at burst limit
	if tb.tokens > float64(tb.burst) {
		tb.tokens = float64(tb.burst)
	}
	
	// Update last update time
	tb.lastUpdate = now
	
	// Check if we have enough tokens
	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	
	return false
}

// Wait blocks until n tokens are available
func (tb *TokenBucket) Wait(ctx context.Context, n int) error {
	tb.mu.Lock()
	now := time.Now()
	
	// Check if we already have enough tokens
	if tb.allowN(now, n) {
		tb.mu.Unlock()
		return nil
	}
	
	// Calculate wait time
	tokensNeeded := float64(n) - tb.tokens
	waitDuration := time.Duration(tokensNeeded/tb.rate * float64(time.Second))
	tb.mu.Unlock()
	
	// Wait with context
	timer := time.NewTimer(waitDuration)
	defer timer.Stop()
	
	select {
	case <-timer.C:
		// Try again after waiting
		if tb.Allow(n) {
			return nil
		}
		return ErrRateLimitExceeded
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Reserve reserves n tokens and returns a Reservation
func (tb *TokenBucket) Reserve(n int) *Reservation {
	return tb.ReserveN(time.Now(), n)
}

// ReserveN reserves n tokens at a specific time and returns a Reservation
func (tb *TokenBucket) ReserveN(now time.Time, n int) *Reservation {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	r := &Reservation{
		ok:     true,
		bucket: tb,
		tokens: n,
		timeAt: now,
	}
	
	// If we can satisfy the request now, do it
	if tb.allowN(now, n) {
		return r
	}
	
	// Calculate when the reservation can be satisfied
	tokensNeeded := float64(n) - tb.tokens
	waitDuration := time.Duration(tokensNeeded/tb.rate * float64(time.Second))
	r.timeAt = now.Add(waitDuration)
	r.ok = true
	
	// Reserve the tokens
	tb.tokens -= float64(n)
	
	return r
}

// Reservation represents a token reservation
type Reservation struct {
	ok     bool
	bucket *TokenBucket
	tokens int
	timeAt time.Time
}

// OK returns whether the reservation is valid
func (r *Reservation) OK() bool {
	return r.ok
}

// Delay returns how long to wait before the reservation can be used
func (r *Reservation) Delay() time.Duration {
	if !r.ok {
		return 0
	}
	return r.timeAt.Sub(time.Now())
}

// Cancel cancels the reservation and returns tokens to the bucket
func (r *Reservation) Cancel() {
	if !r.ok {
		return
	}
	
	r.bucket.mu.Lock()
	defer r.bucket.mu.Unlock()
	
	r.bucket.tokens += float64(r.tokens)
	if r.bucket.tokens > float64(r.bucket.burst) {
		r.bucket.tokens = float64(r.bucket.burst)
	}
	
	r.ok = false
}

// limiterEntry represents a rate limiter with last access time
type limiterEntry struct {
	limiter    *TokenBucket
	lastAccess time.Time
}

// RateLimitMiddleware provides rate limiting functionality
type RateLimitMiddleware struct {
	config     *RateLimitConfig
	limiters   map[string]*limiterEntry
	mu         sync.RWMutex
	globalLimiter *TokenBucket
	stopCleanup   chan struct{}
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(config *RateLimitConfig) *RateLimitMiddleware {
	if config == nil {
		config = DefaultRateLimitConfig()
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	if config.TTL == 0 {
		config.TTL = 30 * time.Minute
	}
	
	m := &RateLimitMiddleware{
		config:      config,
		limiters:    make(map[string]*limiterEntry),
		stopCleanup: make(chan struct{}),
	}
	
	// Create global limiter if needed
	if config.Global {
		m.globalLimiter = NewTokenBucket(config.Rate, config.Burst)
	}
	
	// Start cleanup goroutine
	go m.cleanupRoutine()
	
	return m
}

// Process implements the Middleware interface
func (m *RateLimitMiddleware) Process(ctx context.Context, request interface{}, next func(context.Context, interface{}) (interface{}, error)) (interface{}, error) {
	// Apply global rate limit if configured
	if m.config.Global && m.globalLimiter != nil {
		if !m.globalLimiter.Allow(1) {
			m.config.Logger.WarnContext(ctx, "global rate limit exceeded")
			return nil, ErrRateLimitExceeded
		}
	}
	
	// Get identifier for per-user/per-IP limiting
	identifier := m.getIdentifier(ctx)
	if identifier != "" {
		limiter := m.getLimiter(identifier)
		if !limiter.Allow(1) {
			m.config.Logger.WarnContext(ctx, "rate limit exceeded",
				"identifier", identifier)
			return nil, ErrRateLimitExceeded
		}
	}
	
	// Call the next handler
	return next(ctx, request)
}

// getIdentifier extracts the rate limit identifier from context
func (m *RateLimitMiddleware) getIdentifier(ctx context.Context) string {
	var parts []string
	
	// Extract user ID if per-user limiting is enabled
	if m.config.PerUser {
		if user, ok := GetUser(ctx); ok && user.ID != "" {
			parts = append(parts, "user:"+user.ID)
		}
	}
	
	// Extract IP address if per-IP limiting is enabled
	if m.config.PerIP {
		if ip := ctx.Value("RemoteAddr"); ip != nil {
			if ipStr, ok := ip.(string); ok && ipStr != "" {
				parts = append(parts, "ip:"+ipStr)
			}
		}
	}
	
	if len(parts) == 0 {
		return ""
	}
	
	// Combine parts to create identifier
	identifier := ""
	for i, part := range parts {
		if i > 0 {
			identifier += ":"
		}
		identifier += part
	}
	
	return identifier
}

// getLimiter gets or creates a rate limiter for an identifier
func (m *RateLimitMiddleware) getLimiter(identifier string) *TokenBucket {
	// Try to get existing limiter with read lock
	m.mu.RLock()
	if entry, ok := m.limiters[identifier]; ok {
		entry.lastAccess = time.Now()
		limiter := entry.limiter
		m.mu.RUnlock()
		return limiter
	}
	m.mu.RUnlock()
	
	// Create new limiter with write lock
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Double-check in case another goroutine created it
	if entry, ok := m.limiters[identifier]; ok {
		entry.lastAccess = time.Now()
		return entry.limiter
	}
	
	// Create new limiter
	limiter := NewTokenBucket(m.config.Rate, m.config.Burst)
	m.limiters[identifier] = &limiterEntry{
		limiter:    limiter,
		lastAccess: time.Now(),
	}
	
	m.config.Logger.Debug("created new rate limiter",
		"identifier", identifier,
		"rate", m.config.Rate,
		"burst", m.config.Burst)
	
	return limiter
}

// cleanupRoutine periodically cleans up expired limiters
func (m *RateLimitMiddleware) cleanupRoutine() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopCleanup:
			return
		}
	}
}

// cleanup removes expired limiters
func (m *RateLimitMiddleware) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	expired := make([]string, 0)
	
	// Find expired entries
	for id, entry := range m.limiters {
		if now.Sub(entry.lastAccess) > m.config.TTL {
			expired = append(expired, id)
		}
	}
	
	// Remove expired entries
	for _, id := range expired {
		delete(m.limiters, id)
	}
	
	if len(expired) > 0 {
		m.config.Logger.Debug("cleaned up expired rate limiters",
			"count", len(expired))
	}
}

// Stop stops the rate limit middleware and cleans up resources
func (m *RateLimitMiddleware) Stop() {
	close(m.stopCleanup)
}

// WaitN waits until n tokens are available for the identifier
func (m *RateLimitMiddleware) WaitN(ctx context.Context, identifier string, n int) error {
	limiter := m.getLimiter(identifier)
	return limiter.Wait(ctx, n)
}

// ReserveN reserves n tokens for the identifier
func (m *RateLimitMiddleware) ReserveN(identifier string, n int) *Reservation {
	limiter := m.getLimiter(identifier)
	return limiter.Reserve(n)
}

// Stats returns statistics about the rate limiters
func (m *RateLimitMiddleware) Stats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := map[string]interface{}{
		"active_limiters": len(m.limiters),
		"config": map[string]interface{}{
			"rate":     m.config.Rate,
			"burst":    m.config.Burst,
			"per_user": m.config.PerUser,
			"per_ip":   m.config.PerIP,
			"global":   m.config.Global,
		},
	}
	
	// Add per-limiter stats
	limiters := make(map[string]interface{})
	for id, entry := range m.limiters {
		limiters[id] = map[string]interface{}{
			"tokens":      fmt.Sprintf("%.2f", entry.limiter.tokens),
			"last_access": entry.lastAccess.Format(time.RFC3339),
		}
	}
	stats["limiters"] = limiters
	
	return stats
}