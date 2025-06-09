// Package ratelimit provides Redis-backed rate limiting capabilities
package ratelimit

import (
	"fmt"
	"time"
)

// Config represents the rate limiting configuration
type Config struct {
	// Redis connection settings
	RedisAddr     string `json:"redis_addr" yaml:"redis_addr"`
	RedisPassword string `json:"redis_password" yaml:"redis_password"`
	RedisDB       int    `json:"redis_db" yaml:"redis_db"`

	// Connection pool settings
	MaxRetries      int           `json:"max_retries" yaml:"max_retries"`
	MinRetryBackoff time.Duration `json:"min_retry_backoff" yaml:"min_retry_backoff"`
	MaxRetryBackoff time.Duration `json:"max_retry_backoff" yaml:"max_retry_backoff"`
	DialTimeout     time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	ReadTimeout     time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout" yaml:"write_timeout"`
	PoolSize        int           `json:"pool_size" yaml:"pool_size"`
	MinIdleConns    int           `json:"min_idle_conns" yaml:"min_idle_conns"`
	MaxIdleConns    int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`

	// Global rate limiting settings
	DefaultLimit    int           `json:"default_limit" yaml:"default_limit"`
	DefaultWindow   time.Duration `json:"default_window" yaml:"default_window"`
	DefaultBurst    int           `json:"default_burst" yaml:"default_burst"`
	KeyPrefix       string        `json:"key_prefix" yaml:"key_prefix"`
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval"`

	// Endpoint-specific rate limits
	EndpointLimits map[string]*EndpointLimit `json:"endpoint_limits" yaml:"endpoint_limits"`

	// Global bypass settings
	BypassIPs        []string `json:"bypass_ips" yaml:"bypass_ips"`
	BypassUserAgents []string `json:"bypass_user_agents" yaml:"bypass_user_agents"`
	InternalBypass   bool     `json:"internal_bypass" yaml:"internal_bypass"`

	// Monitoring and alerting
	EnableMetrics  bool    `json:"enable_metrics" yaml:"enable_metrics"`
	EnableAlerting bool    `json:"enable_alerting" yaml:"enable_alerting"`
	AlertThreshold float64 `json:"alert_threshold" yaml:"alert_threshold"`

	// Fallback settings
	EnableFallback bool          `json:"enable_fallback" yaml:"enable_fallback"`
	FallbackLimit  int           `json:"fallback_limit" yaml:"fallback_limit"`
	FallbackWindow time.Duration `json:"fallback_window" yaml:"fallback_window"`
}

// EndpointLimit represents rate limiting configuration for a specific endpoint
type EndpointLimit struct {
	// Basic rate limiting
	Limit  int           `json:"limit" yaml:"limit"`
	Window time.Duration `json:"window" yaml:"window"`
	Burst  int           `json:"burst" yaml:"burst"`

	// Advanced settings
	Algorithm Algorithm `json:"algorithm" yaml:"algorithm"`
	Scope     Scope     `json:"scope" yaml:"scope"`

	// Per-client limits
	PerIPLimit   int `json:"per_ip_limit" yaml:"per_ip_limit"`
	PerUserLimit int `json:"per_user_limit" yaml:"per_user_limit"`

	// Headers and responses
	IncludeHeaders bool   `json:"include_headers" yaml:"include_headers"`
	ResponseCode   int    `json:"response_code" yaml:"response_code"`
	ResponseBody   string `json:"response_body" yaml:"response_body"`

	// Skip conditions
	SkipSuccessfulRequests bool     `json:"skip_successful_requests" yaml:"skip_successful_requests"`
	SkipFailedRequests     bool     `json:"skip_failed_requests" yaml:"skip_failed_requests"`
	SkipPaths              []string `json:"skip_paths" yaml:"skip_paths"`
	SkipMethods            []string `json:"skip_methods" yaml:"skip_methods"`

	// Custom settings
	CustomKey string                 `json:"custom_key" yaml:"custom_key"`
	Metadata  map[string]interface{} `json:"metadata" yaml:"metadata"`
}

// Algorithm represents the rate limiting algorithm
type Algorithm string

const (
	AlgorithmSlidingWindow Algorithm = "sliding_window"
	AlgorithmTokenBucket   Algorithm = "token_bucket"
	AlgorithmFixedWindow   Algorithm = "fixed_window"
	AlgorithmLeakyBucket   Algorithm = "leaky_bucket"
)

// Scope represents the scope of rate limiting
type Scope string

const (
	ScopeGlobal     Scope = "global"
	ScopePerIP      Scope = "per_ip"
	ScopePerUser    Scope = "per_user"
	ScopePerSession Scope = "per_session"
	ScopePerClient  Scope = "per_client"
	ScopeCustom     Scope = "custom"
)

// DefaultConfig returns a default rate limiting configuration
func DefaultConfig() *Config {
	return &Config{
		// Redis settings
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       0,

		// Connection pool
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        10,
		MinIdleConns:    5,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,

		// Global settings
		DefaultLimit:    100,
		DefaultWindow:   time.Minute,
		DefaultBurst:    10,
		KeyPrefix:       "rl:",
		CleanupInterval: 5 * time.Minute,

		// Monitoring
		EnableMetrics:  true,
		EnableAlerting: true,
		AlertThreshold: 0.8, // Alert when 80% of limit is reached

		// Fallback
		EnableFallback: true,
		FallbackLimit:  10,
		FallbackWindow: time.Minute,

		// Default endpoint limits
		EndpointLimits: make(map[string]*EndpointLimit),
		BypassIPs:      []string{"127.0.0.1", "::1"},
		InternalBypass: true,
	}
}

// ProductionConfig returns a production-optimized rate limiting configuration
func ProductionConfig() *Config {
	config := DefaultConfig()

	// Production Redis settings
	config.PoolSize = 20
	config.MinIdleConns = 10
	config.MaxIdleConns = 20
	config.ConnMaxLifetime = 2 * time.Hour

	// Stricter limits
	config.DefaultLimit = 60
	config.DefaultBurst = 5
	config.AlertThreshold = 0.75

	// Fallback settings
	config.FallbackLimit = 5

	// Production endpoint limits
	config.EndpointLimits = map[string]*EndpointLimit{
		// Authentication endpoints (stricter limits)
		"/api/v1/auth/login": {
			Limit:        10,
			Window:       time.Minute,
			Burst:        2,
			Algorithm:    AlgorithmSlidingWindow,
			Scope:        ScopePerIP,
			PerIPLimit:   5,
			ResponseCode: 429,
			ResponseBody: `{"error":"Too many authentication attempts","retry_after":60}`,
		},
		"/api/v1/auth/register": {
			Limit:        5,
			Window:       time.Minute,
			Burst:        1,
			Algorithm:    AlgorithmSlidingWindow,
			Scope:        ScopePerIP,
			PerIPLimit:   3,
			ResponseCode: 429,
			ResponseBody: `{"error":"Too many registration attempts","retry_after":60}`,
		},

		// API endpoints (moderate limits)
		"/api/v1/memory/*": {
			Limit:          100,
			Window:         time.Minute,
			Burst:          10,
			Algorithm:      AlgorithmSlidingWindow,
			Scope:          ScopePerUser,
			IncludeHeaders: true,
		},
		"/api/v1/tasks/*": {
			Limit:          50,
			Window:         time.Minute,
			Burst:          5,
			Algorithm:      AlgorithmSlidingWindow,
			Scope:          ScopePerUser,
			IncludeHeaders: true,
		},

		// WebSocket endpoints (higher limits)
		"/api/v1/ws": {
			Limit:                  1000,
			Window:                 time.Hour,
			Burst:                  50,
			Algorithm:              AlgorithmTokenBucket,
			Scope:                  ScopePerIP,
			SkipSuccessfulRequests: true,
		},

		// Health and monitoring (relaxed limits)
		"/health": {
			Limit:     1000,
			Window:    time.Minute,
			Burst:     100,
			Algorithm: AlgorithmFixedWindow,
			Scope:     ScopeGlobal,
			SkipPaths: []string{"/health", "/metrics", "/status"},
		},

		// CLI registration (moderate limits)
		"/api/v1/cli/register": {
			Limit:      20,
			Window:     time.Minute,
			Burst:      3,
			Algorithm:  AlgorithmSlidingWindow,
			Scope:      ScopePerIP,
			PerIPLimit: 10,
		},
	}

	return config
}

// DevelopmentConfig returns a development-friendly rate limiting configuration
func DevelopmentConfig() *Config {
	config := DefaultConfig()

	// Relaxed limits for development
	config.DefaultLimit = 1000
	config.DefaultBurst = 100
	config.AlertThreshold = 0.9

	// Development endpoint limits (very relaxed)
	config.EndpointLimits = map[string]*EndpointLimit{
		"/api/v1/auth/*": {
			Limit:     100,
			Window:    time.Minute,
			Burst:     20,
			Algorithm: AlgorithmFixedWindow,
			Scope:     ScopePerIP,
		},
		"/api/v1/*": {
			Limit:          1000,
			Window:         time.Minute,
			Burst:          200,
			Algorithm:      AlgorithmTokenBucket,
			Scope:          ScopePerIP,
			IncludeHeaders: true,
		},
	}

	// Bypass for local development
	config.BypassIPs = []string{"127.0.0.1", "::1", "localhost"}
	config.InternalBypass = true

	return config
}

// Validate validates the rate limiting configuration
func (c *Config) Validate() error {
	if c.RedisAddr == "" {
		return fmt.Errorf("redis address is required")
	}

	if c.DefaultLimit <= 0 {
		return fmt.Errorf("default limit must be positive")
	}

	if c.DefaultWindow <= 0 {
		return fmt.Errorf("default window must be positive")
	}

	if c.DefaultBurst < 0 {
		return fmt.Errorf("default burst cannot be negative")
	}

	if c.KeyPrefix == "" {
		c.KeyPrefix = "rl:"
	}

	if c.PoolSize <= 0 {
		c.PoolSize = 10
	}

	if c.AlertThreshold < 0 || c.AlertThreshold > 1 {
		return fmt.Errorf("alert threshold must be between 0 and 1")
	}

	// Validate endpoint limits
	for endpoint, limit := range c.EndpointLimits {
		if err := limit.Validate(); err != nil {
			return fmt.Errorf("invalid configuration for endpoint %s: %w", endpoint, err)
		}
	}

	return nil
}

// Validate validates an endpoint limit configuration
func (el *EndpointLimit) Validate() error {
	if el.Limit <= 0 {
		return fmt.Errorf("limit must be positive")
	}

	if el.Window <= 0 {
		return fmt.Errorf("window must be positive")
	}

	if el.Burst < 0 {
		return fmt.Errorf("burst cannot be negative")
	}

	if el.PerIPLimit < 0 {
		return fmt.Errorf("per-IP limit cannot be negative")
	}

	if el.PerUserLimit < 0 {
		return fmt.Errorf("per-user limit cannot be negative")
	}

	if el.ResponseCode != 0 && (el.ResponseCode < 400 || el.ResponseCode >= 600) {
		return fmt.Errorf("response code must be a valid HTTP error code (400-599)")
	}

	// Set defaults
	if el.Algorithm == "" {
		el.Algorithm = AlgorithmSlidingWindow
	}

	if el.Scope == "" {
		el.Scope = ScopePerIP
	}

	if el.ResponseCode == 0 {
		el.ResponseCode = 429
	}

	if el.ResponseBody == "" {
		el.ResponseBody = `{"error":"Rate limit exceeded","retry_after":60}`
	}

	return nil
}

// GetEndpointLimit returns the rate limit configuration for a specific endpoint
func (c *Config) GetEndpointLimit(endpoint string) *EndpointLimit {
	// Check for exact match first
	if limit, exists := c.EndpointLimits[endpoint]; exists {
		return limit
	}

	// Check for wildcard matches
	for pattern, limit := range c.EndpointLimits {
		if matchWildcard(pattern, endpoint) {
			return limit
		}
	}

	// Return default configuration
	return &EndpointLimit{
		Limit:          c.DefaultLimit,
		Window:         c.DefaultWindow,
		Burst:          c.DefaultBurst,
		Algorithm:      AlgorithmSlidingWindow,
		Scope:          ScopePerIP,
		IncludeHeaders: true,
		ResponseCode:   429,
		ResponseBody:   `{"error":"Rate limit exceeded","retry_after":60}`,
	}
}

// matchWildcard performs simple wildcard matching (* at the end of pattern)
func matchWildcard(pattern, endpoint string) bool {
	if !hasWildcard(pattern) {
		return pattern == endpoint
	}

	// Remove the * and check if endpoint starts with the pattern prefix
	prefix := pattern[:len(pattern)-1]
	return len(endpoint) >= len(prefix) && endpoint[:len(prefix)] == prefix
}

// hasWildcard checks if a pattern contains a wildcard
func hasWildcard(pattern string) bool {
	return len(pattern) > 0 && pattern[len(pattern)-1] == '*'
}

// ShouldBypass checks if a request should bypass rate limiting
func (c *Config) ShouldBypass(ip, userAgent string, isInternal bool) bool {
	// Check internal bypass
	if isInternal && c.InternalBypass {
		return true
	}

	// Check IP bypass
	for _, bypassIP := range c.BypassIPs {
		if ip == bypassIP {
			return true
		}
	}

	// Check user agent bypass
	for _, bypassUA := range c.BypassUserAgents {
		if userAgent == bypassUA {
			return true
		}
	}

	return false
}

// Clone creates a deep copy of the configuration
func (c *Config) Clone() *Config {
	clone := *c

	// Deep copy slices
	clone.BypassIPs = make([]string, len(c.BypassIPs))
	copy(clone.BypassIPs, c.BypassIPs)

	clone.BypassUserAgents = make([]string, len(c.BypassUserAgents))
	copy(clone.BypassUserAgents, c.BypassUserAgents)

	// Deep copy endpoint limits
	clone.EndpointLimits = make(map[string]*EndpointLimit)
	for k, v := range c.EndpointLimits {
		limitClone := *v
		if v.SkipPaths != nil {
			limitClone.SkipPaths = make([]string, len(v.SkipPaths))
			copy(limitClone.SkipPaths, v.SkipPaths)
		}
		if v.SkipMethods != nil {
			limitClone.SkipMethods = make([]string, len(v.SkipMethods))
			copy(limitClone.SkipMethods, v.SkipMethods)
		}
		if v.Metadata != nil {
			limitClone.Metadata = make(map[string]interface{})
			for mk, mv := range v.Metadata {
				limitClone.Metadata[mk] = mv
			}
		}
		clone.EndpointLimits[k] = &limitClone
	}

	return &clone
}
