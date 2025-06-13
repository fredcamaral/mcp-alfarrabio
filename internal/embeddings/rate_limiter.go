// Package embeddings provides rate limiting for API calls
package embeddings

import (
	"context"
	"sync"
	"time"
)

// RateLimiter provides token bucket rate limiting for API calls
type RateLimiter struct {
	maxTokens  int
	tokens     int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the specified max tokens and refill rate
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	if maxTokens <= 0 {
		maxTokens = 60 // Default to 60 tokens
	}
	if refillRate == 0 {
		refillRate = time.Minute
	}

	return &RateLimiter{
		maxTokens:  maxTokens,
		tokens:     maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow returns true if a token is available, false otherwise
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.refill()

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// Wait blocks until a token is available or context is cancelled
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(rl.refillRate / time.Duration(rl.maxTokens)):
			// Continue to next iteration
		}
	}
}

// refill adds tokens based on elapsed time since last refill
func (rl *RateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// Calculate how many tokens to add based on elapsed time
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		rl.tokens += tokensToAdd
		if rl.tokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		}
		rl.lastRefill = now
	}
}
