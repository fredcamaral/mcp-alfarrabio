// Package middleware provides security headers middleware for production security
package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// SecurityHeadersMiddleware provides comprehensive security headers
type SecurityHeadersMiddleware struct {
	config SecurityHeadersConfig
}

// SecurityHeadersConfig configures security headers
type SecurityHeadersConfig struct {
	// Content Security Policy
	ContentSecurityPolicy string `json:"content_security_policy"`

	// HTTP Strict Transport Security
	StrictTransportSecurity string `json:"strict_transport_security"`

	// X-Frame-Options
	FrameOptions string `json:"frame_options"`

	// X-Content-Type-Options
	ContentTypeOptions bool `json:"content_type_options"`

	// X-XSS-Protection
	XSSProtection string `json:"xss_protection"`

	// Referrer-Policy
	ReferrerPolicy string `json:"referrer_policy"`

	// Permissions-Policy
	PermissionsPolicy string `json:"permissions_policy"`

	// Custom security headers
	CustomHeaders map[string]string `json:"custom_headers"`

	// Environment-specific settings
	Environment string `json:"environment"`

	// Server identification
	HideServerVersion bool `json:"hide_server_version"`
}

// DefaultSecurityHeadersConfig returns secure defaults for production
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy:   "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' ws: wss:; frame-ancestors 'none';",
		StrictTransportSecurity: "max-age=31536000; includeSubDomains; preload",
		FrameOptions:            "DENY",
		ContentTypeOptions:      true,
		XSSProtection:           "1; mode=block",
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		PermissionsPolicy:       "geolocation=(), microphone=(), camera=(), fullscreen=(self), payment=()",
		Environment:             "production",
		HideServerVersion:       true,
		CustomHeaders: map[string]string{
			"X-Robots-Tag": "noindex, nofollow, noarchive, nosnippet",
		},
	}
}

// DevelopmentSecurityHeadersConfig returns relaxed settings for development
func DevelopmentSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy:   "default-src 'self' 'unsafe-inline' 'unsafe-eval'; connect-src 'self' ws: wss: http://localhost:* http://127.0.0.1:*;",
		StrictTransportSecurity: "", // Disabled for HTTP development
		FrameOptions:            "SAMEORIGIN",
		ContentTypeOptions:      true,
		XSSProtection:           "1; mode=block",
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		PermissionsPolicy:       "geolocation=(), microphone=(), camera=()",
		Environment:             "development",
		HideServerVersion:       false,
		CustomHeaders:           make(map[string]string),
	}
}

// NewSecurityHeadersMiddleware creates a new security headers middleware
func NewSecurityHeadersMiddleware(config *SecurityHeadersConfig) *SecurityHeadersMiddleware {
	return &SecurityHeadersMiddleware{
		config: *config,
	}
}

// NewDefaultSecurityHeadersMiddleware creates middleware with production defaults
func NewDefaultSecurityHeadersMiddleware() *SecurityHeadersMiddleware {
	defaultConfig := DefaultSecurityHeadersConfig()
	return NewSecurityHeadersMiddleware(&defaultConfig)
}

// NewDevelopmentSecurityHeadersMiddleware creates middleware for development
func NewDevelopmentSecurityHeadersMiddleware() *SecurityHeadersMiddleware {
	devConfig := DevelopmentSecurityHeadersConfig()
	return NewSecurityHeadersMiddleware(&devConfig)
}

// Handler returns the security headers middleware handler
func (s *SecurityHeadersMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set security headers before processing request
			s.setSecurityHeaders(w, r)

			next.ServeHTTP(w, r)
		})
	}
}

// setSecurityHeaders applies all configured security headers
func (s *SecurityHeadersMiddleware) setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	// Content Security Policy
	if s.config.ContentSecurityPolicy != "" {
		w.Header().Set("Content-Security-Policy", s.config.ContentSecurityPolicy)
	}

	// HTTP Strict Transport Security (only for HTTPS)
	if s.config.StrictTransportSecurity != "" && s.isHTTPS(r) {
		w.Header().Set("Strict-Transport-Security", s.config.StrictTransportSecurity)
	}

	// X-Frame-Options
	if s.config.FrameOptions != "" {
		w.Header().Set("X-Frame-Options", s.config.FrameOptions)
	}

	// X-Content-Type-Options
	if s.config.ContentTypeOptions {
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}

	// X-XSS-Protection
	if s.config.XSSProtection != "" {
		w.Header().Set("X-XSS-Protection", s.config.XSSProtection)
	}

	// Referrer-Policy
	if s.config.ReferrerPolicy != "" {
		w.Header().Set("Referrer-Policy", s.config.ReferrerPolicy)
	}

	// Permissions-Policy
	if s.config.PermissionsPolicy != "" {
		w.Header().Set("Permissions-Policy", s.config.PermissionsPolicy)
	}

	// Hide server version information
	if s.config.HideServerVersion {
		w.Header().Set("Server", "")
		w.Header().Del("Server")
	}

	// Custom headers
	for key, value := range s.config.CustomHeaders {
		w.Header().Set(key, value)
	}

	// API-specific security headers
	s.setAPISecurityHeaders(w, r)
}

// setAPISecurityHeaders sets API-specific security headers
func (s *SecurityHeadersMiddleware) setAPISecurityHeaders(w http.ResponseWriter, r *http.Request) {
	// Cache control for API responses
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
	}

	// Set Content-Type for API responses if not already set
	if strings.HasPrefix(r.URL.Path, "/api/") && w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	// Rate limiting information headers (if available)
	s.setRateLimitHeaders(w, r)

	// Security information headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Download-Options", "noopen")
	w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
}

// setRateLimitHeaders sets rate limiting information headers
func (s *SecurityHeadersMiddleware) setRateLimitHeaders(w http.ResponseWriter, r *http.Request) {
	// These headers can be set by rate limiting middleware
	// We ensure they're present in the response if rate limiting is active

	// Get rate limit information from context if available
	if rateLimitInfo := getRateLimitFromContext(r.Context()); rateLimitInfo != nil {
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rateLimitInfo.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", rateLimitInfo.Remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", rateLimitInfo.Reset))

		if rateLimitInfo.RetryAfter > 0 {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", rateLimitInfo.RetryAfter))
		}
	}
}

// RateLimitInfo represents rate limiting information
type RateLimitInfo struct {
	Limit      int   `json:"limit"`
	Remaining  int   `json:"remaining"`
	Reset      int64 `json:"reset"`
	RetryAfter int   `json:"retry_after"`
}

// getRateLimitFromContext extracts rate limit info from request context
func getRateLimitFromContext(ctx interface{}) *RateLimitInfo {
	// This would be implemented based on how rate limiting middleware
	// stores information in the request context
	// For now, return nil as it's optional
	return nil
}

// isHTTPS determines if the request is using HTTPS
func (s *SecurityHeadersMiddleware) isHTTPS(r *http.Request) bool {
	// Check the URL scheme
	if r.URL.Scheme == "https" {
		return true
	}

	// Check for proxy headers indicating HTTPS
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}

	if r.Header.Get("X-Forwarded-Ssl") == "on" {
		return true
	}

	// Check TLS connection
	return r.TLS != nil
}

// GetSecurityHeaders returns the current security headers configuration
func (s *SecurityHeadersMiddleware) GetSecurityHeaders() map[string]string {
	headers := make(map[string]string)

	if s.config.ContentSecurityPolicy != "" {
		headers["Content-Security-Policy"] = s.config.ContentSecurityPolicy
	}

	if s.config.StrictTransportSecurity != "" {
		headers["Strict-Transport-Security"] = s.config.StrictTransportSecurity
	}

	if s.config.FrameOptions != "" {
		headers["X-Frame-Options"] = s.config.FrameOptions
	}

	if s.config.ContentTypeOptions {
		headers["X-Content-Type-Options"] = "nosniff"
	}

	if s.config.XSSProtection != "" {
		headers["X-XSS-Protection"] = s.config.XSSProtection
	}

	if s.config.ReferrerPolicy != "" {
		headers["Referrer-Policy"] = s.config.ReferrerPolicy
	}

	if s.config.PermissionsPolicy != "" {
		headers["Permissions-Policy"] = s.config.PermissionsPolicy
	}

	// Add custom headers
	for key, value := range s.config.CustomHeaders {
		headers[key] = value
	}

	return headers
}

// ValidateConfig validates the security headers configuration
func (s *SecurityHeadersMiddleware) ValidateConfig() error {
	if s.config.Environment != "development" && s.config.Environment != "production" {
		return fmt.Errorf("invalid environment: %s (must be 'development' or 'production')", s.config.Environment)
	}

	// Validate CSP policy format (basic check)
	if s.config.ContentSecurityPolicy != "" {
		if !strings.Contains(s.config.ContentSecurityPolicy, "default-src") {
			return fmt.Errorf("content Security Policy must include default-src directive")
		}
	}

	// Validate HSTS format
	if s.config.StrictTransportSecurity != "" {
		if !strings.Contains(s.config.StrictTransportSecurity, "max-age=") {
			return fmt.Errorf("Strict-Transport-Security must include max-age directive")
		}
	}

	// Validate frame options
	validFrameOptions := []string{"DENY", "SAMEORIGIN", "ALLOW-FROM"}
	if s.config.FrameOptions != "" {
		valid := false
		for _, option := range validFrameOptions {
			if strings.HasPrefix(s.config.FrameOptions, option) {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid X-Frame-Options: %s", s.config.FrameOptions)
		}
	}

	return nil
}

// UpdateConfig updates the security headers configuration
func (s *SecurityHeadersMiddleware) UpdateConfig(config *SecurityHeadersConfig) error {
	// Create temporary middleware to validate config
	temp := &SecurityHeadersMiddleware{config: *config}
	if err := temp.ValidateConfig(); err != nil {
		return err
	}

	s.config = *config
	return nil
}
