// Package middleware provides enhanced rate limiting middleware with Redis backend
package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lerian-mcp-memory/internal/ratelimit"
)

// EnhancedRateLimit provides comprehensive rate limiting middleware
type EnhancedRateLimit struct {
	config          *ratelimit.Config
	redisLimiter    *ratelimit.RedisLimiter
	fallbackLimiter *ratelimit.SlidingWindow
	monitor         *ratelimit.Monitor
}

// RateLimitContext holds rate limiting context for a request
type RateLimitContext struct {
	Key        string
	IP         string
	UserAgent  string
	Endpoint   string
	Method     string
	IsInternal bool
	UserID     string
	SessionID  string
	ClientID   string
}

// RateLimitResponse represents the response when rate limited
type RateLimitResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	RetryAfter int    `json:"retry_after"`
	Limit      int    `json:"limit"`
	Remaining  int    `json:"remaining"`
	ResetTime  int64  `json:"reset_time"`
	RequestID  string `json:"request_id,omitempty"`
}

// NewEnhancedRateLimit creates a new enhanced rate limiting middleware
func NewEnhancedRateLimit(config *ratelimit.Config) (*EnhancedRateLimit, error) {
	if config == nil {
		config = ratelimit.DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid rate limit config: %w", err)
	}

	// Create Redis limiter
	redisLimiter, err := ratelimit.NewRedisLimiter(config)
	if err != nil {
		// If Redis is unavailable, log warning but continue with fallback
		fmt.Printf("Warning: Redis unavailable for rate limiting, using fallback: %v\n", err)
	}

	// Create fallback limiter
	fallbackLimiter := ratelimit.NewSlidingWindow(config)

	// Create monitor
	var limiter ratelimit.RateLimiter = fallbackLimiter
	if redisLimiter != nil {
		limiter = redisLimiter
	}
	monitor := ratelimit.NewMonitor(config, limiter)

	return &EnhancedRateLimit{
		config:          config,
		redisLimiter:    redisLimiter,
		fallbackLimiter: fallbackLimiter,
		monitor:         monitor,
	}, nil
}

// Middleware returns the rate limiting HTTP middleware
func (erl *EnhancedRateLimit) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Build rate limiting context
			ctx := erl.buildContext(r)

			// Check if request should bypass rate limiting
			if erl.config.ShouldBypass(ctx.IP, ctx.UserAgent, ctx.IsInternal) {
				next.ServeHTTP(w, r)
				return
			}

			// Get endpoint configuration
			endpointLimit := erl.config.GetEndpointLimit(ctx.Endpoint)

			// Check if this path should be skipped
			if erl.shouldSkipPath(r.URL.Path, r.Method, endpointLimit) {
				next.ServeHTTP(w, r)
				return
			}

			// Build rate limiting key
			key := erl.buildKey(ctx, endpointLimit)

			// Perform rate limiting check
			result, err := erl.checkRateLimit(r.Context(), key, endpointLimit)
			if err != nil {
				// Log error but allow request to proceed
				erl.monitor.RecordError("check_failed", err)
				next.ServeHTTP(w, r)
				return
			}

			// Record metrics
			duration := time.Since(start)
			erl.monitor.RecordRequest(ctx.Endpoint, key, result, duration)

			// Check if rate limited
			if !result.Allowed {
				erl.handleRateLimited(w, r, result, endpointLimit)
				return
			}

			// Add rate limit headers if enabled
			if endpointLimit.IncludeHeaders {
				erl.addRateLimitHeaders(w, result)
			}

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// buildContext builds rate limiting context from HTTP request
func (erl *EnhancedRateLimit) buildContext(r *http.Request) *RateLimitContext {
	ctx := &RateLimitContext{
		Endpoint:   erl.normalizeEndpoint(r.URL.Path),
		Method:     r.Method,
		UserAgent:  r.Header.Get("User-Agent"),
		IsInternal: erl.isInternalRequest(r),
	}

	// Extract IP address
	ctx.IP = erl.extractIP(r)

	// Extract user/session/client IDs from headers or context
	ctx.UserID = r.Header.Get("X-User-ID")
	ctx.SessionID = r.Header.Get("X-Session-ID")
	ctx.ClientID = r.Header.Get("X-Client-ID")

	return ctx
}

// extractIP extracts the real IP address from the request
func (erl *EnhancedRateLimit) extractIP(r *http.Request) string {
	// Check X-Forwarded-For header (most common)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Check CF-Connecting-IP (Cloudflare)
	if cfip := r.Header.Get("CF-Connecting-IP"); cfip != "" {
		return cfip
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// normalizeEndpoint normalizes the endpoint path for rate limiting
func (erl *EnhancedRateLimit) normalizeEndpoint(path string) string {
	// Remove query parameters
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}

	// Remove trailing slash
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	// Normalize API versioning
	if strings.HasPrefix(path, "/api/v") {
		parts := strings.Split(path, "/")
		if len(parts) >= 3 {
			// Keep /api/v1/resource format
			return "/" + strings.Join(parts[1:3], "/") + "/*"
		}
	}

	return path
}

// isInternalRequest checks if the request is internal
func (erl *EnhancedRateLimit) isInternalRequest(r *http.Request) bool {
	// Check for internal service headers
	if r.Header.Get("X-Internal-Service") != "" {
		return true
	}

	// Check for service mesh headers
	if r.Header.Get("X-Service-Mesh") != "" {
		return true
	}

	// Check for health check patterns
	if strings.Contains(r.URL.Path, "/health") ||
		strings.Contains(r.URL.Path, "/metrics") ||
		strings.Contains(r.URL.Path, "/status") {
		return true
	}

	return false
}

// buildKey builds the rate limiting key based on scope
func (erl *EnhancedRateLimit) buildKey(ctx *RateLimitContext, limit *ratelimit.EndpointLimit) string {
	var keyParts []string

	// Add endpoint
	keyParts = append(keyParts, ctx.Endpoint)

	// Add scope-specific identifiers
	switch limit.Scope {
	case ratelimit.ScopeGlobal:
		keyParts = append(keyParts, "global")
	case ratelimit.ScopePerIP:
		keyParts = append(keyParts, "ip", ctx.IP)
	case ratelimit.ScopePerUser:
		if ctx.UserID != "" {
			keyParts = append(keyParts, "user", ctx.UserID)
		} else {
			keyParts = append(keyParts, "ip", ctx.IP) // Fall back to IP
		}
	case ratelimit.ScopePerSession:
		if ctx.SessionID != "" {
			keyParts = append(keyParts, "session", ctx.SessionID)
		} else {
			keyParts = append(keyParts, "ip", ctx.IP) // Fall back to IP
		}
	case ratelimit.ScopePerClient:
		if ctx.ClientID != "" {
			keyParts = append(keyParts, "client", ctx.ClientID)
		} else {
			keyParts = append(keyParts, "ip", ctx.IP) // Fall back to IP
		}
	case ratelimit.ScopeCustom:
		if limit.CustomKey != "" {
			keyParts = append(keyParts, "custom", limit.CustomKey)
		} else {
			keyParts = append(keyParts, "ip", ctx.IP) // Fall back to IP
		}
	default:
		keyParts = append(keyParts, "ip", ctx.IP)
	}

	return strings.Join(keyParts, ":")
}

// shouldSkipPath checks if the path should skip rate limiting
func (erl *EnhancedRateLimit) shouldSkipPath(path, method string, limit *ratelimit.EndpointLimit) bool {
	// Check skip paths
	for _, skipPath := range limit.SkipPaths {
		if path == skipPath {
			return true
		}
	}

	// Check skip methods
	for _, skipMethod := range limit.SkipMethods {
		if method == skipMethod {
			return true
		}
	}

	return false
}

// checkRateLimit performs the rate limiting check
func (erl *EnhancedRateLimit) checkRateLimit(ctx context.Context, key string, limit *ratelimit.EndpointLimit) (*ratelimit.LimitResult, error) {
	// Try Redis limiter first
	if erl.redisLimiter != nil {
		result, err := erl.redisLimiter.Check(ctx, key, limit)
		if err == nil {
			return result, nil
		}

		// Record Redis error and fall back
		erl.monitor.RecordError("redis", err)
	}

	// Use fallback limiter
	if erl.fallbackLimiter != nil {
		erl.monitor.RecordError("fallback", nil)
		return erl.fallbackLimiter.Check(ctx, key, limit)
	}

	return nil, errors.New("no rate limiter available")
}

// handleRateLimited handles rate limited requests
func (erl *EnhancedRateLimit) handleRateLimited(w http.ResponseWriter, r *http.Request, result *ratelimit.LimitResult, limit *ratelimit.EndpointLimit) {
	_ = r // unused parameter, kept for potential future request analysis
	// Set response code
	responseCode := limit.ResponseCode
	if responseCode == 0 {
		responseCode = http.StatusTooManyRequests
	}

	// Add rate limit headers
	erl.addRateLimitHeaders(w, result)

	// Set content type
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(responseCode)

	// Create response
	response := RateLimitResponse{
		Error:      "rate_limit_exceeded",
		Message:    "Rate limit exceeded. Please try again later.",
		RetryAfter: int(result.RetryAfter.Seconds()),
		Limit:      result.Limit,
		Remaining:  result.Remaining,
		ResetTime:  result.ResetTime.Unix(),
	}

	// Use custom response body if provided
	if limit.ResponseBody != "" {
		_, _ = w.Write([]byte(limit.ResponseBody))
		return
	}

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fall back to simple text response
		_, _ = w.Write([]byte(`{"error":"rate_limit_exceeded","message":"Rate limit exceeded"}`))
	}
}

// addRateLimitHeaders adds standard rate limiting headers
func (erl *EnhancedRateLimit) addRateLimitHeaders(w http.ResponseWriter, result *ratelimit.LimitResult) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))

	if result.RetryAfter > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
	}

	// Add additional headers
	w.Header().Set("X-RateLimit-Algorithm", string(result.Algorithm))
	w.Header().Set("X-RateLimit-Window", result.Window.String())

	if result.Burst > 0 {
		w.Header().Set("X-RateLimit-Burst", strconv.Itoa(result.Burst))
	}
}

// GetMetrics returns current rate limiting metrics
func (erl *EnhancedRateLimit) GetMetrics(ctx context.Context) (*ratelimit.Metrics, error) {
	return erl.monitor.GetMetrics(ctx)
}

// GetHealthStatus returns health status
func (erl *EnhancedRateLimit) GetHealthStatus(ctx context.Context) (map[string]interface{}, error) {
	return erl.monitor.GetHealthStatus(ctx)
}

// Reset resets rate limits for a key
func (erl *EnhancedRateLimit) Reset(ctx context.Context, key string) error {
	var err error

	// Reset in Redis limiter
	if erl.redisLimiter != nil {
		if resetErr := erl.redisLimiter.Reset(ctx, key); resetErr != nil {
			err = resetErr
		}
	}

	// Reset in fallback limiter
	if erl.fallbackLimiter != nil {
		if resetErr := erl.fallbackLimiter.Reset(ctx, key); resetErr != nil {
			err = resetErr
		}
	}

	return err
}

// Cleanup performs cleanup operations
func (erl *EnhancedRateLimit) Cleanup(ctx context.Context) error {
	var err error

	// Cleanup Redis limiter
	if erl.redisLimiter != nil {
		if cleanupErr := erl.redisLimiter.Cleanup(ctx); cleanupErr != nil {
			err = cleanupErr
		}
	}

	// Cleanup fallback limiter
	if erl.fallbackLimiter != nil {
		if cleanupErr := erl.fallbackLimiter.Cleanup(ctx); cleanupErr != nil {
			err = cleanupErr
		}
	}

	return err
}

// Close closes the rate limiter and releases resources
func (erl *EnhancedRateLimit) Close() error {
	var err error

	// Close monitor
	if erl.monitor != nil {
		if closeErr := erl.monitor.Close(); closeErr != nil {
			err = closeErr
		}
	}

	// Close Redis limiter
	if erl.redisLimiter != nil {
		if closeErr := erl.redisLimiter.Close(); closeErr != nil {
			err = closeErr
		}
	}

	// Close fallback limiter
	if erl.fallbackLimiter != nil {
		if closeErr := erl.fallbackLimiter.Close(); closeErr != nil {
			err = closeErr
		}
	}

	return err
}
