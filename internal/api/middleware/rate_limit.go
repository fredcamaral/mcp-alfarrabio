// Package middleware provides HTTP middleware for rate limiting and API protection.
package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"lerian-mcp-memory/internal/api/response"
)

// RateLimiter provides rate limiting functionality using sliding window algorithm
type RateLimiter struct {
	rules   map[string]*RateRule
	windows map[string]*SlidingWindow
	mu      sync.RWMutex
	config  RateLimitConfig
}

// RateRule defines rate limiting configuration for endpoints
type RateRule struct {
	Endpoint   string        `json:"endpoint"`
	Limit      int           `json:"limit"`       // Requests per window
	Window     time.Duration `json:"window"`      // Time window duration
	PerClient  bool          `json:"per_client"`  // Apply limit per client IP
	PerUser    bool          `json:"per_user"`    // Apply limit per authenticated user
	BurstLimit int           `json:"burst_limit"` // Maximum burst requests
	Priority   RulePriority  `json:"priority"`    // Rule priority
	Exemptions []string      `json:"exemptions"`  // Exempt IPs or user IDs
}

// RulePriority defines rule evaluation priority
type RulePriority int

const (
	PriorityLow    RulePriority = 1
	PriorityMedium RulePriority = 2
	PriorityHigh   RulePriority = 3
	PriorityMax    RulePriority = 4
)

// SlidingWindow implements sliding window rate limiting
type SlidingWindow struct {
	requests   []time.Time
	limit      int
	window     time.Duration
	burstLimit int
	mu         sync.Mutex
}

// RateLimitConfig represents rate limiter configuration
type RateLimitConfig struct {
	Enabled         bool          `json:"enabled"`
	DefaultLimit    int           `json:"default_limit"`
	DefaultWindow   time.Duration `json:"default_window"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	MaxWindows      int           `json:"max_windows"`
	EnableMetrics   bool          `json:"enable_metrics"`
	HeaderPrefix    string        `json:"header_prefix"`
	SkipSuccessful  bool          `json:"skip_successful"`
	TrustedProxies  []string      `json:"trusted_proxies"`
}

// RateLimitResult represents the result of rate limit check
type RateLimitResult struct {
	Allowed     bool          `json:"allowed"`
	Limit       int           `json:"limit"`
	Remaining   int           `json:"remaining"`
	ResetTime   time.Time     `json:"reset_time"`
	RetryAfter  time.Duration `json:"retry_after"`
	Rule        *RateRule     `json:"rule"`
	WindowUsage float64       `json:"window_usage"`
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:         true,
		DefaultLimit:    100,
		DefaultWindow:   time.Hour,
		CleanupInterval: 15 * time.Minute,
		MaxWindows:      10000,
		EnableMetrics:   true,
		HeaderPrefix:    "X-RateLimit",
		SkipSuccessful:  false,
		TrustedProxies:  []string{"127.0.0.1", "::1"},
	}
}

// NewRateLimiter creates a new rate limiter with configuration
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		rules:   make(map[string]*RateRule),
		windows: make(map[string]*SlidingWindow),
		config:  *config,
	}

	// Add default rate limiting rules
	rl.addDefaultRules()

	// Start cleanup routine
	go rl.cleanupRoutine()

	return rl
}

// AddRule adds a rate limiting rule
func (rl *RateLimiter) AddRule(rule *RateRule) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	key := rl.generateRuleKey(rule)
	rl.rules[key] = rule
}

// CheckLimit checks if request is within rate limits
func (rl *RateLimiter) CheckLimit(r *http.Request) *RateLimitResult {
	if !rl.config.Enabled {
		return &RateLimitResult{
			Allowed:   true,
			Limit:     -1,
			Remaining: -1,
		}
	}

	// Find applicable rule
	rule := rl.findApplicableRule(r)
	if rule == nil {
		// Use default limits if no rule found
		rule = rl.getDefaultRule()
	}

	// Check if client is exempt
	if rl.isExempt(r, rule) {
		return &RateLimitResult{
			Allowed:   true,
			Limit:     rule.Limit,
			Remaining: rule.Limit,
			Rule:      rule,
		}
	}

	// Generate window key
	windowKey := rl.generateWindowKey(r, rule)

	// Get or create sliding window
	window := rl.getOrCreateWindow(windowKey, rule)

	// Check rate limit
	allowed, remaining, resetTime := window.Allow()

	result := &RateLimitResult{
		Allowed:     allowed,
		Limit:       rule.Limit,
		Remaining:   remaining,
		ResetTime:   resetTime,
		Rule:        rule,
		WindowUsage: window.Usage(),
	}

	if !allowed {
		result.RetryAfter = time.Until(resetTime)
	}

	return result
}

// Middleware returns HTTP middleware for rate limiting
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result := rl.CheckLimit(r)

			// Add rate limit headers
			rl.addHeaders(w, result)

			if !result.Allowed {
				// Rate limit exceeded
				retryAfter := int(result.RetryAfter.Seconds())
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))

				response.WriteError(w, http.StatusTooManyRequests, "Rate limit exceeded",
					fmt.Sprintf("Rate limit of %d requests per %v exceeded. Try again in %v.",
						result.Limit, result.Rule.Window, result.RetryAfter))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Helper methods

func (rl *RateLimiter) addDefaultRules() {
	// API-wide default rule
	rl.AddRule(&RateRule{
		Endpoint:   "/*",
		Limit:      rl.config.DefaultLimit,
		Window:     rl.config.DefaultWindow,
		PerClient:  true,
		BurstLimit: rl.config.DefaultLimit / 10,
		Priority:   PriorityLow,
	})

	// More restrictive rules for sensitive endpoints
	rl.AddRule(&RateRule{
		Endpoint:   "/api/v1/tasks",
		Limit:      50,
		Window:     time.Hour,
		PerClient:  true,
		BurstLimit: 10,
		Priority:   PriorityMedium,
	})

	rl.AddRule(&RateRule{
		Endpoint:   "/api/v1/tasks/search",
		Limit:      200,
		Window:     time.Hour,
		PerClient:  true,
		BurstLimit: 20,
		Priority:   PriorityMedium,
	})

	rl.AddRule(&RateRule{
		Endpoint:   "/api/v1/tasks/batch/*",
		Limit:      20,
		Window:     time.Hour,
		PerClient:  true,
		BurstLimit: 5,
		Priority:   PriorityHigh,
	})

	// Authentication endpoints
	rl.AddRule(&RateRule{
		Endpoint:   "/api/v1/auth/*",
		Limit:      10,
		Window:     time.Minute * 15,
		PerClient:  true,
		BurstLimit: 3,
		Priority:   PriorityMax,
	})
}

func (rl *RateLimiter) findApplicableRule(r *http.Request) *RateRule {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	var matchedRule *RateRule
	var highestPriority RulePriority = 0

	for _, rule := range rl.rules {
		if rl.matchesRule(r, rule) && rule.Priority > highestPriority {
			matchedRule = rule
			highestPriority = rule.Priority
		}
	}

	return matchedRule
}

func (rl *RateLimiter) matchesRule(r *http.Request, rule *RateRule) bool {
	path := r.URL.Path
	endpoint := rule.Endpoint

	// Exact match
	if endpoint == path {
		return true
	}

	// Wildcard match
	if strings.HasSuffix(endpoint, "/*") {
		prefix := strings.TrimSuffix(endpoint, "/*")
		return strings.HasPrefix(path, prefix)
	}

	// Glob pattern match
	if strings.Contains(endpoint, "*") {
		return rl.matchGlob(path, endpoint)
	}

	return false
}

func (rl *RateLimiter) matchGlob(path, pattern string) bool {
	// Simple glob matching - can be enhanced with more sophisticated pattern matching
	if pattern == "/*" {
		return true
	}

	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		return strings.HasPrefix(path, parts[0]) && strings.HasSuffix(path, parts[1])
	}

	return false
}

func (rl *RateLimiter) getDefaultRule() *RateRule {
	return &RateRule{
		Endpoint:   "default",
		Limit:      rl.config.DefaultLimit,
		Window:     rl.config.DefaultWindow,
		PerClient:  true,
		BurstLimit: rl.config.DefaultLimit / 10,
		Priority:   PriorityLow,
	}
}

func (rl *RateLimiter) isExempt(r *http.Request, rule *RateRule) bool {
	clientIP := rl.getClientIP(r)
	userID := rl.getUserID(r)

	for _, exemption := range rule.Exemptions {
		if exemption == clientIP || exemption == userID {
			return true
		}
	}

	return false
}

func (rl *RateLimiter) generateRuleKey(rule *RateRule) string {
	return fmt.Sprintf("%s:%v:%v", rule.Endpoint, rule.Limit, rule.Window)
}

func (rl *RateLimiter) generateWindowKey(r *http.Request, rule *RateRule) string {
	var keyParts []string

	keyParts = append(keyParts, rule.Endpoint)

	if rule.PerClient {
		keyParts = append(keyParts, "client:"+rl.getClientIP(r))
	}

	if rule.PerUser {
		userID := rl.getUserID(r)
		if userID != "" {
			keyParts = append(keyParts, "user:"+userID)
		}
	}

	return strings.Join(keyParts, "|")
}

func (rl *RateLimiter) getOrCreateWindow(key string, rule *RateRule) *SlidingWindow {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	window, exists := rl.windows[key]
	if !exists {
		window = NewSlidingWindow(rule.Limit, rule.Window, rule.BurstLimit)
		rl.windows[key] = window
	}

	return window
}

func (rl *RateLimiter) getClientIP(r *http.Request) string {
	// Check for trusted proxy headers
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Extract IP from RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}

func (rl *RateLimiter) getUserID(r *http.Request) string {
	// Extract user ID from various sources
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID
	}

	// Can be enhanced to extract from JWT tokens or session
	if ctx := r.Context(); ctx != nil {
		if userID, ok := ctx.Value("user_id").(string); ok {
			return userID
		}
	}

	return ""
}

func (rl *RateLimiter) addHeaders(w http.ResponseWriter, result *RateLimitResult) {
	prefix := rl.config.HeaderPrefix

	w.Header().Set(prefix+"-Limit", strconv.Itoa(result.Limit))
	w.Header().Set(prefix+"-Remaining", strconv.Itoa(result.Remaining))
	w.Header().Set(prefix+"-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))

	if result.WindowUsage > 0 {
		w.Header().Set(prefix+"-Usage", fmt.Sprintf("%.2f", result.WindowUsage))
	}
}

func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, window := range rl.windows {
		if window.IsExpired(now) {
			delete(rl.windows, key)
		}
	}

	// Limit total number of windows to prevent memory issues
	if len(rl.windows) > rl.config.MaxWindows {
		// Remove oldest windows - can be enhanced with LRU
		count := 0
		for key := range rl.windows {
			delete(rl.windows, key)
			count++
			if count >= len(rl.windows)/4 { // Remove 25% of windows
				break
			}
		}
	}
}

// SlidingWindow implementation

// NewSlidingWindow creates a new sliding window
func NewSlidingWindow(limit int, window time.Duration, burstLimit int) *SlidingWindow {
	return &SlidingWindow{
		requests:   make([]time.Time, 0),
		limit:      limit,
		window:     window,
		burstLimit: burstLimit,
	}
}

// Allow checks if request is allowed and records it if so
func (sw *SlidingWindow) Allow() (allowed bool, remaining int, resetTime time.Time) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-sw.window)

	// Remove expired requests
	sw.removeExpiredRequests(windowStart)

	// Check if within limits
	currentCount := len(sw.requests)
	if currentCount >= sw.limit {
		// Calculate reset time (when oldest request expires)
		if len(sw.requests) > 0 {
			resetTime = sw.requests[0].Add(sw.window)
		} else {
			resetTime = now.Add(sw.window)
		}
		return false, 0, resetTime
	}

	// Check burst limit
	if sw.burstLimit > 0 {
		recentRequests := sw.countRecentRequests(now.Add(-time.Minute))
		if recentRequests >= sw.burstLimit {
			resetTime = now.Add(time.Minute)
			return false, sw.limit - currentCount, resetTime
		}
	}

	// Add current request
	sw.requests = append(sw.requests, now)
	remaining = sw.limit - len(sw.requests)
	resetTime = now.Add(sw.window)

	return true, remaining, resetTime
}

// Usage returns current window usage as percentage
func (sw *SlidingWindow) Usage() float64 {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.limit == 0 {
		return 0
	}

	now := time.Now()
	windowStart := now.Add(-sw.window)
	sw.removeExpiredRequests(windowStart)

	return float64(len(sw.requests)) / float64(sw.limit)
}

// IsExpired checks if window can be cleaned up
func (sw *SlidingWindow) IsExpired(now time.Time) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	windowStart := now.Add(-sw.window)
	sw.removeExpiredRequests(windowStart)

	return len(sw.requests) == 0
}

func (sw *SlidingWindow) removeExpiredRequests(windowStart time.Time) {
	// Remove requests outside the window
	validFrom := 0
	for i, req := range sw.requests {
		if req.After(windowStart) {
			validFrom = i
			break
		}
		validFrom = len(sw.requests) // All expired
	}

	if validFrom > 0 {
		sw.requests = sw.requests[validFrom:]
	}
}

func (sw *SlidingWindow) countRecentRequests(since time.Time) int {
	count := 0
	for _, req := range sw.requests {
		if req.After(since) {
			count++
		}
	}
	return count
}
