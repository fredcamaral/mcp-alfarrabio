// Package ratelimit provides sliding window rate limiting algorithm
package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"
)

// SlidingWindow implements an in-memory sliding window rate limiter
// This serves as a fallback when Redis is unavailable
type SlidingWindow struct {
	mu      sync.RWMutex
	windows map[string]*Window
	config  *Config
	cleanup chan struct{}
	done    chan struct{}
}

// Window represents a sliding window for rate limiting
type Window struct {
	mu        sync.RWMutex
	requests  []time.Time
	limit     int
	window    time.Duration
	burst     int
	lastClean time.Time
}

// WindowStats represents statistics for a sliding window
type WindowStats struct {
	Key           string        `json:"key"`
	RequestCount  int           `json:"request_count"`
	Limit         int           `json:"limit"`
	Window        time.Duration `json:"window"`
	Burst         int           `json:"burst"`
	OldestRequest *time.Time    `json:"oldest_request,omitempty"`
	NewestRequest *time.Time    `json:"newest_request,omitempty"`
	LastCleanup   time.Time     `json:"last_cleanup"`
	WindowStart   time.Time     `json:"window_start"`
	WindowEnd     time.Time     `json:"window_end"`
	RequestRate   float64       `json:"request_rate"` // requests per second
}

// NewSlidingWindow creates a new sliding window rate limiter
func NewSlidingWindow(config *Config) *SlidingWindow {
	if config == nil {
		config = DefaultConfig()
	}

	sw := &SlidingWindow{
		windows: make(map[string]*Window),
		config:  config,
		cleanup: make(chan struct{}, 1),
		done:    make(chan struct{}),
	}

	// Start cleanup routine
	go sw.cleanupRoutine()

	return sw
}

// Check performs a rate limit check using sliding window algorithm
func (sw *SlidingWindow) Check(ctx context.Context, key string, limit *EndpointLimit) (*LimitResult, error) {
	if limit == nil {
		return nil, errors.New("endpoint limit configuration is required")
	}

	now := time.Now()

	sw.mu.Lock()
	window, exists := sw.windows[key]
	if !exists {
		window = &Window{
			requests:  make([]time.Time, 0),
			limit:     limit.Limit,
			window:    limit.Window,
			burst:     limit.Burst,
			lastClean: now,
		}
		sw.windows[key] = window
	}
	sw.mu.Unlock()

	return sw.checkWindow(window, key, now, limit)
}

// CheckMultiple performs rate limit checks for multiple keys
func (sw *SlidingWindow) CheckMultiple(ctx context.Context, keys []string, limits []*EndpointLimit) ([]*LimitResult, error) {
	if len(keys) != len(limits) {
		return nil, errors.New("keys and limits slices must have the same length")
	}

	results := make([]*LimitResult, len(keys))
	now := time.Now()

	sw.mu.Lock()
	// Ensure all windows exist
	for i, key := range keys {
		limit := limits[i]
		if limit == nil {
			results[i] = &LimitResult{
				Allowed: false,
				Key:     key,
			}
			continue
		}

		if _, exists := sw.windows[key]; !exists {
			sw.windows[key] = &Window{
				requests:  make([]time.Time, 0),
				limit:     limit.Limit,
				window:    limit.Window,
				burst:     limit.Burst,
				lastClean: now,
			}
		}
	}
	sw.mu.Unlock()

	// Process each window
	for i, key := range keys {
		limit := limits[i]
		if limit == nil {
			continue
		}

		sw.mu.RLock()
		window := sw.windows[key]
		sw.mu.RUnlock()

		result, err := sw.checkWindow(window, key, now, limit)
		if err != nil {
			result = &LimitResult{
				Allowed: false,
				Key:     key,
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			}
		}
		results[i] = result
	}

	return results, nil
}

// checkWindow performs the actual sliding window check
func (sw *SlidingWindow) checkWindow(window *Window, key string, now time.Time, _ *EndpointLimit) (*LimitResult, error) {
	window.mu.Lock()
	defer window.mu.Unlock()

	// Clean expired requests if needed
	if now.Sub(window.lastClean) > time.Minute {
		sw.cleanExpiredRequests(window, now)
		window.lastClean = now
	}

	// Count current requests in window
	windowStart := now.Add(-window.window)
	currentCount := 0

	for _, reqTime := range window.requests {
		if reqTime.After(windowStart) {
			currentCount++
		}
	}

	// Check if request is allowed
	allowed := currentCount < window.limit
	actualLimit := window.limit + window.burst

	// Allow burst if within burst limit
	if !allowed && currentCount < actualLimit {
		allowed = true
	}

	var retryAfter time.Duration
	var resetTime time.Time

	if allowed {
		// Add current request
		window.requests = append(window.requests, now)
		currentCount++
		resetTime = sw.calculateResetTimeForAllowed(window, windowStart, now)
	} else {
		resetTime, retryAfter = sw.calculateResetTimeForDenied(window, windowStart, now)
	}

	remaining := window.limit - currentCount
	if remaining < 0 {
		remaining = 0
	}

	return &LimitResult{
		Allowed:        allowed,
		Count:          currentCount,
		Limit:          window.limit,
		Remaining:      remaining,
		RetryAfter:     retryAfter,
		ResetTime:      resetTime,
		Algorithm:      AlgorithmSlidingWindow,
		Key:            key,
		Window:         window.window,
		Burst:          window.burst,
		IsFirstRequest: len(window.requests) == 1,
		Metadata:       make(map[string]interface{}),
	}, nil
}

// calculateResetTimeForAllowed calculates reset time when request is allowed
func (sw *SlidingWindow) calculateResetTimeForAllowed(window *Window, windowStart, now time.Time) time.Time {
	if len(window.requests) == 0 {
		return now.Add(window.window)
	}

	oldestInWindow := window.requests[0]
	for _, reqTime := range window.requests {
		if reqTime.After(windowStart) {
			oldestInWindow = reqTime
			break
		}
	}
	return oldestInWindow.Add(window.window)
}

// calculateResetTimeForDenied calculates reset time and retry after when request is denied
func (sw *SlidingWindow) calculateResetTimeForDenied(window *Window, windowStart, now time.Time) (time.Time, time.Duration) {
	oldestInWindow := now
	for _, reqTime := range window.requests {
		if reqTime.After(windowStart) {
			oldestInWindow = reqTime
			break
		}
	}

	resetTime := oldestInWindow.Add(window.window)
	retryAfter := time.Until(resetTime)
	if retryAfter < 0 {
		retryAfter = 0
	}

	return resetTime, retryAfter
}

// Reset resets the sliding window for a given key
func (sw *SlidingWindow) Reset(ctx context.Context, key string) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if window, exists := sw.windows[key]; exists {
		window.mu.Lock()
		window.requests = window.requests[:0] // Clear slice but keep capacity
		window.mu.Unlock()
	}

	return nil
}

// ResetMultiple resets the sliding windows for multiple keys
func (sw *SlidingWindow) ResetMultiple(ctx context.Context, keys []string) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	for _, key := range keys {
		if window, exists := sw.windows[key]; exists {
			window.mu.Lock()
			window.requests = window.requests[:0]
			window.mu.Unlock()
		}
	}

	return nil
}

// GetStats returns current statistics for a key
func (sw *SlidingWindow) GetStats(ctx context.Context, key string) (map[string]interface{}, error) {
	sw.mu.RLock()
	window, exists := sw.windows[key]
	sw.mu.RUnlock()

	if !exists {
		return map[string]interface{}{
			"key":           key,
			"exists":        false,
			"request_count": 0,
		}, nil
	}

	return sw.getWindowStats(window, key), nil
}

// GetAllStats returns statistics for all windows
func (sw *SlidingWindow) GetAllStats(ctx context.Context) (map[string]interface{}, error) {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_windows"] = len(sw.windows)
	stats["windows"] = make(map[string]interface{})

	for key, window := range sw.windows {
		stats["windows"].(map[string]interface{})[key] = sw.getWindowStats(window, key)
	}

	return stats, nil
}

// getWindowStats returns detailed statistics for a window
func (sw *SlidingWindow) getWindowStats(window *Window, key string) map[string]interface{} {
	window.mu.RLock()
	defer window.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-window.window)

	// Count current requests and find oldest/newest
	currentCount := 0
	var oldest, newest *time.Time

	for _, reqTime := range window.requests {
		if reqTime.After(windowStart) {
			currentCount++
			if oldest == nil || reqTime.Before(*oldest) {
				oldest = &reqTime
			}
			if newest == nil || reqTime.After(*newest) {
				newest = &reqTime
			}
		}
	}

	// Calculate request rate (requests per second)
	var requestRate float64
	if window.window > 0 {
		requestRate = float64(currentCount) / window.window.Seconds()
	}

	stats := map[string]interface{}{
		"key":            key,
		"request_count":  currentCount,
		"total_requests": len(window.requests),
		"limit":          window.limit,
		"window":         window.window.String(),
		"burst":          window.burst,
		"last_cleanup":   window.lastClean,
		"window_start":   windowStart,
		"window_end":     now,
		"request_rate":   requestRate,
	}

	if oldest != nil {
		stats["oldest_request"] = *oldest
	}
	if newest != nil {
		stats["newest_request"] = *newest
	}

	return stats
}

// Cleanup removes expired windows and requests
func (sw *SlidingWindow) Cleanup(ctx context.Context) error {
	select {
	case sw.cleanup <- struct{}{}:
		// Trigger cleanup
	default:
		// Cleanup already in progress
	}
	return nil
}

// cleanupRoutine runs periodic cleanup
func (sw *SlidingWindow) cleanupRoutine() {
	ticker := time.NewTicker(sw.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sw.performCleanup()
		case <-sw.cleanup:
			sw.performCleanup()
		case <-sw.done:
			return
		}
	}
}

// performCleanup removes expired requests and empty windows
func (sw *SlidingWindow) performCleanup() {
	now := time.Now()

	sw.mu.Lock()
	defer sw.mu.Unlock()

	keysToDelete := make([]string, 0)

	for key, window := range sw.windows {
		window.mu.Lock()

		// Clean expired requests
		sw.cleanExpiredRequests(window, now)

		// Remove window if no recent activity
		if len(window.requests) == 0 && now.Sub(window.lastClean) > window.window*2 {
			keysToDelete = append(keysToDelete, key)
		}

		window.mu.Unlock()
	}

	// Delete empty windows
	for _, key := range keysToDelete {
		delete(sw.windows, key)
	}
}

// cleanExpiredRequests removes requests outside the window
func (sw *SlidingWindow) cleanExpiredRequests(window *Window, now time.Time) {
	windowStart := now.Add(-window.window)

	// Find first request within window
	validStart := 0
	for i, reqTime := range window.requests {
		if reqTime.After(windowStart) {
			validStart = i
			break
		}
		if i == len(window.requests)-1 {
			// No valid requests found
			validStart = len(window.requests)
		}
	}

	// Keep only valid requests
	if validStart > 0 {
		copy(window.requests, window.requests[validStart:])
		window.requests = window.requests[:len(window.requests)-validStart]
	}
}

// Close stops the sliding window limiter
func (sw *SlidingWindow) Close() error {
	close(sw.done)

	sw.mu.Lock()
	defer sw.mu.Unlock()

	// Clear all windows
	sw.windows = make(map[string]*Window)

	return nil
}

// IsHealthy checks if the sliding window limiter is healthy
func (sw *SlidingWindow) IsHealthy(ctx context.Context) error {
	// Always healthy for in-memory implementation
	return nil
}

// GetInfo returns information about the sliding window limiter
func (sw *SlidingWindow) GetInfo(ctx context.Context) (map[string]interface{}, error) {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	info := map[string]interface{}{
		"type":         "sliding_window",
		"window_count": len(sw.windows),
		"config": map[string]interface{}{
			"default_limit":    sw.config.DefaultLimit,
			"default_window":   sw.config.DefaultWindow.String(),
			"cleanup_interval": sw.config.CleanupInterval.String(),
		},
	}

	// Add memory usage estimation
	totalRequests := 0
	for _, window := range sw.windows {
		window.mu.RLock()
		totalRequests += len(window.requests)
		window.mu.RUnlock()
	}

	// Rough memory estimation (time.Time is ~24 bytes)
	estimatedMemory := totalRequests * 24
	info["estimated_memory_bytes"] = estimatedMemory
	info["total_tracked_requests"] = totalRequests

	return info, nil
}

// WindowCount returns the number of active windows
func (sw *SlidingWindow) WindowCount() int {
	sw.mu.RLock()
	defer sw.mu.RUnlock()
	return len(sw.windows)
}

// TotalRequests returns the total number of tracked requests across all windows
func (sw *SlidingWindow) TotalRequests() int {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	total := 0
	for _, window := range sw.windows {
		window.mu.RLock()
		total += len(window.requests)
		window.mu.RUnlock()
	}
	return total
}
