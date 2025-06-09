// Package middleware provides input sanitization and validation middleware
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode"
)

// Removed unused constant

// sanitizationContextKey is a custom type for context values to avoid collisions
type sanitizationContextKey string

const (
	// contextKeySecurityThreats is the context key for security threats
	contextKeySecurityThreats sanitizationContextKey = "security_threats"
)

// SanitizationMiddleware provides input sanitization and validation
type SanitizationMiddleware struct {
	config SanitizationConfig
}

// SanitizationConfig configures input sanitization rules
type SanitizationConfig struct {
	// Enable/disable different types of sanitization
	EnableXSSProtection      bool `json:"enable_xss_protection"`
	EnableSQLInjectionCheck  bool `json:"enable_sql_injection_check"`
	EnableHTMLSanitization   bool `json:"enable_html_sanitization"`
	EnableScriptTagRemoval   bool `json:"enable_script_tag_removal"`
	EnablePathTraversalCheck bool `json:"enable_path_traversal_check"`

	// Size limits
	MaxRequestSize int64 `json:"max_request_size"` // bytes
	MaxFieldLength int   `json:"max_field_length"` // characters
	MaxArrayLength int   `json:"max_array_length"` // array elements
	MaxObjectDepth int   `json:"max_object_depth"` // JSON nesting depth

	// Content validation
	AllowedContentTypes []string `json:"allowed_content_types"`
	RequiredHeaders     []string `json:"required_headers"`

	// Custom validation patterns
	DeniedPatterns  []string `json:"denied_patterns"`
	AllowedPatterns []string `json:"allowed_patterns"`

	// Logging
	LogSuspiciousRequests bool   `json:"log_suspicious_requests"`
	LogLevel              string `json:"log_level"` // "debug", "info", "warn", "error"
}

// DefaultSanitizationConfig returns secure defaults
func DefaultSanitizationConfig() SanitizationConfig {
	return SanitizationConfig{
		EnableXSSProtection:      true,
		EnableSQLInjectionCheck:  true,
		EnableHTMLSanitization:   true,
		EnableScriptTagRemoval:   true,
		EnablePathTraversalCheck: true,
		MaxRequestSize:           10 * 1024 * 1024, // 10MB
		MaxFieldLength:           50000,            // 50k characters
		MaxArrayLength:           1000,             // 1000 elements
		MaxObjectDepth:           10,               // 10 levels deep
		AllowedContentTypes: []string{
			"application/json",
			"application/x-www-form-urlencoded",
			"multipart/form-data",
			"text/plain",
		},
		RequiredHeaders: []string{
			"Content-Type",
		},
		DeniedPatterns: []string{
			// SQL injection patterns
			`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute)\s+`,
			`(?i)'.*or.*'`,
			`(?i)'.*and.*'`,
			`(?i);.*--`,

			// XSS patterns
			`<script[^>]*>.*?</script>`,
			`javascript:`,
			`on\w+\s*=`,
			`<iframe[^>]*>.*?</iframe>`,
			`<object[^>]*>.*?</object>`,
			`<embed[^>]*>`,

			// Path traversal patterns
			`\.\.\/`,
			`\.\.\\`,
			`%2e%2e%2f`,
			`%2e%2e%5c`,
		},
		LogSuspiciousRequests: true,
		LogLevel:              "warn",
	}
}

// ThreatType represents the type of security threat detected
type ThreatType string

const (
	ThreatXSS           ThreatType = "xss"
	ThreatSQLInjection  ThreatType = "sql_injection"
	ThreatPathTraversal ThreatType = "path_traversal"
	ThreatOversized     ThreatType = "oversized_request"
	ThreatInvalidFormat ThreatType = "invalid_format"
	ThreatDeniedPattern ThreatType = "denied_pattern"
)

// ThreatDetection represents a detected security threat
type ThreatDetection struct {
	Type        ThreatType `json:"type"`
	Description string     `json:"description"`
	Field       string     `json:"field,omitempty"`
	Value       string     `json:"value,omitempty"`
	Pattern     string     `json:"pattern,omitempty"`
	Severity    string     `json:"severity"`
}

// NewSanitizationMiddleware creates a new sanitization middleware
func NewSanitizationMiddleware(config *SanitizationConfig) *SanitizationMiddleware {
	return &SanitizationMiddleware{
		config: *config,
	}
}

// NewDefaultSanitizationMiddleware creates middleware with secure defaults
func NewDefaultSanitizationMiddleware() *SanitizationMiddleware {
	defaultConfig := DefaultSanitizationConfig()
	return NewSanitizationMiddleware(&defaultConfig)
}

// Handler returns the sanitization middleware handler
func (s *SanitizationMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check request size
			if s.config.MaxRequestSize > 0 && r.ContentLength > s.config.MaxRequestSize {
				s.logThreat(r, &ThreatDetection{
					Type:        ThreatOversized,
					Description: fmt.Sprintf("Request size %d exceeds limit %d", r.ContentLength, s.config.MaxRequestSize),
					Severity:    "high",
				})
				http.Error(w, "Request entity too large", http.StatusRequestEntityTooLarge)
				return
			}

			// Validate content type
			if !s.isContentTypeAllowed(r) {
				s.logThreat(r, &ThreatDetection{
					Type:        ThreatInvalidFormat,
					Description: fmt.Sprintf("Content-Type %s not allowed", r.Header.Get("Content-Type")),
					Severity:    "medium",
				})
				http.Error(w, "Unsupported content type", http.StatusUnsupportedMediaType)
				return
			}

			// Check required headers
			for _, header := range s.config.RequiredHeaders {
				if r.Header.Get(header) == "" {
					http.Error(w, fmt.Sprintf("Missing required header: %s", header), http.StatusBadRequest)
					return
				}
			}

			// Sanitize request
			sanitizedRequest, threats := s.sanitizeRequest(r)
			if len(threats) > 0 {
				// Log threats
				for _, threat := range threats {
					s.logThreat(r, &threat)
				}

				// Check if any severityCritical threats were found
				if s.hasCriticalThreats(threats) {
					http.Error(w, "Request contains malicious content", http.StatusBadRequest)
					return
				}
			}

			// Add threat information to context for audit logging
			ctx := context.WithValue(r.Context(), contextKeySecurityThreats, threats)
			sanitizedRequest = sanitizedRequest.WithContext(ctx)

			next.ServeHTTP(w, sanitizedRequest)
		})
	}
}

// isContentTypeAllowed checks if the request content type is allowed
func (s *SanitizationMiddleware) isContentTypeAllowed(r *http.Request) bool {
	if len(s.config.AllowedContentTypes) == 0 {
		return true // No restrictions
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" && r.Method == "GET" {
		return true // GET requests without body are okay
	}

	// Extract the main content type (ignore charset, boundary, etc.)
	mainType := strings.Split(contentType, ";")[0]
	mainType = strings.TrimSpace(mainType)

	for _, allowed := range s.config.AllowedContentTypes {
		if strings.EqualFold(mainType, allowed) {
			return true
		}
	}

	return false
}

// sanitizeRequest sanitizes the entire request and returns threats found
func (s *SanitizationMiddleware) sanitizeRequest(r *http.Request) (*http.Request, []ThreatDetection) {
	var threats []ThreatDetection

	// Sanitize URL and query parameters
	urlThreats := s.sanitizeURL(r)
	threats = append(threats, urlThreats...)

	// Sanitize headers
	headerThreats := s.sanitizeHeaders(r)
	threats = append(threats, headerThreats...)

	// Sanitize body (for POST, PUT, PATCH requests)
	if r.Body != nil && r.ContentLength > 0 {
		sanitizedBody, bodyThreats := s.sanitizeBody(r)
		threats = append(threats, bodyThreats...)

		// Replace the request body with sanitized version
		r.Body = io.NopCloser(bytes.NewReader(sanitizedBody))
		r.ContentLength = int64(len(sanitizedBody))
	}

	return r, threats
}

// sanitizeURL sanitizes URL path and query parameters
func (s *SanitizationMiddleware) sanitizeURL(r *http.Request) []ThreatDetection {
	var threats []ThreatDetection

	// Check path traversal in URL path
	if s.config.EnablePathTraversalCheck {
		if pathThreats := s.detectPathTraversal("url_path", r.URL.Path); len(pathThreats) > 0 {
			threats = append(threats, pathThreats...)
		}
	}

	// Sanitize query parameters
	for key, values := range r.URL.Query() {
		for i, value := range values {
			// Check for XSS
			if s.config.EnableXSSProtection {
				if xssThreats := s.detectXSS(fmt.Sprintf("query_%s", key), value); len(xssThreats) > 0 {
					threats = append(threats, xssThreats...)
					// Sanitize the value
					values[i] = s.sanitizeXSS(value)
				}
			}

			// Check for SQL injection
			if s.config.EnableSQLInjectionCheck {
				if sqlThreats := s.detectSQLInjection(fmt.Sprintf("query_%s", key), value); len(sqlThreats) > 0 {
					threats = append(threats, sqlThreats...)
				}
			}

			// Check denied patterns
			if patternThreats := s.checkDeniedPatterns(fmt.Sprintf("query_%s", key), value); len(patternThreats) > 0 {
				threats = append(threats, patternThreats...)
			}
		}
	}

	return threats
}

// sanitizeHeaders sanitizes HTTP headers
func (s *SanitizationMiddleware) sanitizeHeaders(r *http.Request) []ThreatDetection {
	var threats []ThreatDetection

	for name, values := range r.Header {
		for _, value := range values {
			// Check for XSS in headers
			if s.config.EnableXSSProtection {
				if xssThreats := s.detectXSS(fmt.Sprintf("header_%s", name), value); len(xssThreats) > 0 {
					threats = append(threats, xssThreats...)
				}
			}

			// Check denied patterns in headers
			if patternThreats := s.checkDeniedPatterns(fmt.Sprintf("header_%s", name), value); len(patternThreats) > 0 {
				threats = append(threats, patternThreats...)
			}
		}
	}

	return threats
}

// sanitizeBody sanitizes request body
func (s *SanitizationMiddleware) sanitizeBody(r *http.Request) ([]byte, []ThreatDetection) {
	var threats []ThreatDetection

	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return body, threats
	}

	// Get content type
	contentType := r.Header.Get("Content-Type")
	mainType := strings.Split(contentType, ";")[0]
	mainType = strings.TrimSpace(mainType)

	// Handle JSON content
	if strings.EqualFold(mainType, "application/json") {
		sanitizedBody, jsonThreats := s.sanitizeJSON(body)
		threats = append(threats, jsonThreats...)
		return sanitizedBody, threats
	}

	// Handle other content types as plain text
	bodyStr := string(body)
	bodyThreats := s.sanitizeString("request_body", bodyStr)
	threats = append(threats, bodyThreats...)

	// Apply sanitization
	sanitizedStr := s.sanitizeXSS(bodyStr)
	return []byte(sanitizedStr), threats
}

// sanitizeJSON sanitizes JSON content
func (s *SanitizationMiddleware) sanitizeJSON(data []byte) ([]byte, []ThreatDetection) {
	var threats []ThreatDetection
	var obj interface{}

	// Parse JSON
	if err := json.Unmarshal(data, &obj); err != nil {
		// Invalid JSON, return as-is but log threat
		threats = append(threats, ThreatDetection{
			Type:        ThreatInvalidFormat,
			Description: "Invalid JSON format",
			Severity:    "medium",
		})
		return data, threats
	}

	// Sanitize the object recursively
	sanitizedObj, objThreats := s.sanitizeJSONObject(obj, 0)
	threats = append(threats, objThreats...)

	// Marshal back to JSON
	sanitizedData, err := json.Marshal(sanitizedObj)
	if err != nil {
		// If we can't marshal, return original
		return data, threats
	}

	return sanitizedData, threats
}

// sanitizeJSONObject recursively sanitizes a JSON object
func (s *SanitizationMiddleware) sanitizeJSONObject(obj interface{}, depth int) (interface{}, []ThreatDetection) {
	var threats []ThreatDetection

	// Check nesting depth
	if depth > s.config.MaxObjectDepth {
		threats = append(threats, ThreatDetection{
			Type:        ThreatInvalidFormat,
			Description: fmt.Sprintf("JSON nesting depth %d exceeds limit %d", depth, s.config.MaxObjectDepth),
			Severity:    "high",
		})
		return nil, threats
	}

	switch v := obj.(type) {
	case string:
		stringThreats := s.sanitizeString(fmt.Sprintf("json_field_depth_%d", depth), v)
		threats = append(threats, stringThreats...)
		return s.sanitizeXSS(v), threats

	case map[string]interface{}:
		sanitizedMap := make(map[string]interface{})
		for key, value := range v {
			// Sanitize key
			keyThreats := s.sanitizeString(fmt.Sprintf("json_key_depth_%d", depth), key)
			threats = append(threats, keyThreats...)
			sanitizedKey := s.sanitizeXSS(key)

			// Sanitize value recursively
			sanitizedValue, valueThreats := s.sanitizeJSONObject(value, depth+1)
			threats = append(threats, valueThreats...)

			sanitizedMap[sanitizedKey] = sanitizedValue
		}
		return sanitizedMap, threats

	case []interface{}:
		// Check array length
		if len(v) > s.config.MaxArrayLength {
			threats = append(threats, ThreatDetection{
				Type:        ThreatInvalidFormat,
				Description: fmt.Sprintf("Array length %d exceeds limit %d", len(v), s.config.MaxArrayLength),
				Severity:    "medium",
			})
			// Truncate array
			v = v[:s.config.MaxArrayLength]
		}

		sanitizedArray := make([]interface{}, len(v))
		for i, item := range v {
			sanitizedItem, itemThreats := s.sanitizeJSONObject(item, depth+1)
			threats = append(threats, itemThreats...)
			sanitizedArray[i] = sanitizedItem
		}
		return sanitizedArray, threats

	default:
		// Numbers, booleans, null - return as-is
		return obj, threats
	}
}

// sanitizeString performs comprehensive string sanitization
func (s *SanitizationMiddleware) sanitizeString(field, value string) []ThreatDetection {
	var threats []ThreatDetection

	// Check length
	if len(value) > s.config.MaxFieldLength {
		threats = append(threats, ThreatDetection{
			Type:        ThreatInvalidFormat,
			Description: fmt.Sprintf("Field length %d exceeds limit %d", len(value), s.config.MaxFieldLength),
			Field:       field,
			Severity:    "medium",
		})
	}

	// Check for XSS
	if s.config.EnableXSSProtection {
		if xssThreats := s.detectXSS(field, value); len(xssThreats) > 0 {
			threats = append(threats, xssThreats...)
		}
	}

	// Check for SQL injection
	if s.config.EnableSQLInjectionCheck {
		if sqlThreats := s.detectSQLInjection(field, value); len(sqlThreats) > 0 {
			threats = append(threats, sqlThreats...)
		}
	}

	// Check for path traversal
	if s.config.EnablePathTraversalCheck {
		if pathThreats := s.detectPathTraversal(field, value); len(pathThreats) > 0 {
			threats = append(threats, pathThreats...)
		}
	}

	// Check denied patterns
	if patternThreats := s.checkDeniedPatterns(field, value); len(patternThreats) > 0 {
		threats = append(threats, patternThreats...)
	}

	return threats
}

// detectXSS detects XSS patterns in input
func (s *SanitizationMiddleware) detectXSS(field, value string) []ThreatDetection {
	var threats []ThreatDetection

	xssPatterns := []string{
		`<script[^>]*>.*?</script>`,
		`javascript:`,
		`on\w+\s*=`,
		`<iframe[^>]*>`,
		`<object[^>]*>`,
		`<embed[^>]*>`,
		`<link[^>]*>`,
		`<meta[^>]*>`,
		`<style[^>]*>`,
		`expression\s*\(`,
		`@import`,
		`vbscript:`,
	}

	for _, pattern := range xssPatterns {
		if matched, _ := regexp.MatchString(`(?i)`+pattern, value); matched {
			threats = append(threats, ThreatDetection{
				Type:        ThreatXSS,
				Description: "Potential XSS attack detected",
				Field:       field,
				Value:       value,
				Pattern:     pattern,
				Severity:    "high",
			})
		}
	}

	return threats
}

// detectSQLInjection detects SQL injection patterns
func (s *SanitizationMiddleware) detectSQLInjection(field, value string) []ThreatDetection {
	var threats []ThreatDetection

	sqlPatterns := []string{
		`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute)\s+`,
		`(?i)'.*or.*'`,
		`(?i)'.*and.*'`,
		`(?i);.*--`,
		`(?i)/\*.*\*/`,
		`(?i)char\s*\(`,
		`(?i)convert\s*\(`,
		`(?i)cast\s*\(`,
	}

	for _, pattern := range sqlPatterns {
		if matched, _ := regexp.MatchString(pattern, value); matched {
			threats = append(threats, ThreatDetection{
				Type:        ThreatSQLInjection,
				Description: "Potential SQL injection detected",
				Field:       field,
				Value:       value,
				Pattern:     pattern,
				Severity:    "severityCritical",
			})
		}
	}

	return threats
}

// detectPathTraversal detects path traversal patterns
func (s *SanitizationMiddleware) detectPathTraversal(field, value string) []ThreatDetection {
	var threats []ThreatDetection

	pathPatterns := []string{
		`\.\.\/`,
		`\.\.\\`,
		`%2e%2e%2f`,
		`%2e%2e%5c`,
		`%252e%252e%252f`,
		`%c0%ae%c0%ae%c0%af`,
		`%c1%9c`,
	}

	for _, pattern := range pathPatterns {
		if matched, _ := regexp.MatchString(`(?i)`+pattern, value); matched {
			threats = append(threats, ThreatDetection{
				Type:        ThreatPathTraversal,
				Description: "Potential path traversal attack detected",
				Field:       field,
				Value:       value,
				Pattern:     pattern,
				Severity:    "high",
			})
		}
	}

	return threats
}

// checkDeniedPatterns checks against custom denied patterns
func (s *SanitizationMiddleware) checkDeniedPatterns(field, value string) []ThreatDetection {
	var threats []ThreatDetection

	for _, pattern := range s.config.DeniedPatterns {
		if matched, _ := regexp.MatchString(`(?i)`+pattern, value); matched {
			threats = append(threats, ThreatDetection{
				Type:        ThreatDeniedPattern,
				Description: "Matches denied pattern",
				Field:       field,
				Value:       value,
				Pattern:     pattern,
				Severity:    "medium",
			})
		}
	}

	return threats
}

// sanitizeXSS sanitizes XSS content
func (s *SanitizationMiddleware) sanitizeXSS(input string) string {
	if !s.config.EnableXSSProtection {
		return input
	}

	// HTML escape
	if s.config.EnableHTMLSanitization {
		input = html.EscapeString(input)
	}

	// Remove script tags
	if s.config.EnableScriptTagRemoval {
		scriptPattern := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
		input = scriptPattern.ReplaceAllString(input, "")

		// Remove dangerous attributes
		onEventPattern := regexp.MustCompile(`(?i)on\w+\s*=\s*["'][^"']*["']`)
		input = onEventPattern.ReplaceAllString(input, "")

		// Remove javascript: URLs
		jsPattern := regexp.MustCompile(`(?i)javascript:`)
		input = jsPattern.ReplaceAllString(input, "")
	}

	return input
}

// hasCriticalThreats checks if any severityCritical threats were detected
func (s *SanitizationMiddleware) hasCriticalThreats(threats []ThreatDetection) bool {
	for _, threat := range threats {
		if threat.Severity == "severityCritical" {
			return true
		}
	}
	return false
}

// logThreat logs a security threat
func (s *SanitizationMiddleware) logThreat(r *http.Request, threat *ThreatDetection) {
	if !s.config.LogSuspiciousRequests {
		return
	}

	// This would integrate with your logging system
	// For now, we'll use a simple approach
	logMessage := fmt.Sprintf("Security threat detected: %s - %s (Field: %s, IP: %s, User-Agent: %s)",
		threat.Type, threat.Description, threat.Field,
		r.RemoteAddr, r.Header.Get("User-Agent"))

	// In a real implementation, you'd use structured logging
	switch threat.Severity {
	case "severityCritical":
		// log.Error(logMessage)
		fmt.Printf("ERROR: %s\n", logMessage)
	case "high":
		// log.Warn(logMessage)
		fmt.Printf("WARN: %s\n", logMessage)
	default:
		// log.Info(logMessage)
		fmt.Printf("INFO: %s\n", logMessage)
	}
}

// IsValidUTF8 checks if the string is valid UTF-8
func IsValidUTF8(s string) bool {
	for _, r := range s {
		if r == unicode.ReplacementChar {
			return false
		}
	}
	return true
}
