package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig represents CORS configuration options
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// CORSMiddleware provides Cross-Origin Resource Sharing (CORS) support
type CORSMiddleware struct {
	config CORSConfig
}

// NewCORSMiddleware creates a new CORS middleware with configuration
func NewCORSMiddleware(config *CORSConfig) *CORSMiddleware {
	// Set defaults if not provided
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}

	if len(config.AllowedHeaders) == 0 {
		config.AllowedHeaders = []string{
			"Accept",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
			"X-Client-Version",
			"X-CLI-Version",
			"X-Request-ID",
		}
	}

	if len(config.ExposedHeaders) == 0 {
		config.ExposedHeaders = []string{
			"X-Request-ID",
			"X-Server-Version",
			"X-Compatible-Versions",
		}
	}

	if config.MaxAge == 0 {
		config.MaxAge = 86400 // 24 hours
	}

	return &CORSMiddleware{config: *config}
}

// NewDefaultCORSMiddleware creates CORS middleware with sensible defaults for development
func NewDefaultCORSMiddleware() *CORSMiddleware {
	return NewCORSMiddleware(&CORSConfig{
		AllowedOrigins: []string{
			"http://localhost:2001",
			"http://localhost:3000",
			"http://localhost:8080",
			"http://127.0.0.1:2001",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:8080",
		},
		AllowCredentials: true,
	})
}

// NewProductionCORSMiddleware creates CORS middleware for production with specific origins
func NewProductionCORSMiddleware(allowedOrigins []string) *CORSMiddleware {
	return NewCORSMiddleware(&CORSConfig{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
	})
}

// Handler returns the CORS middleware handler
func (c *CORSMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Set CORS headers if origin is allowed or if no origin (same-origin requests)
			if origin == "" || c.isOriginAllowed(origin) {
				c.setCORSHeaders(w, origin)
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				c.handlePreflight(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if the origin is in the allowed list
func (c *CORSMiddleware) isOriginAllowed(origin string) bool {
	// Allow all origins if "*" is in the list
	for _, allowedOrigin := range c.config.AllowedOrigins {
		if allowedOrigin == "*" {
			return true
		}
		if allowedOrigin == origin {
			return true
		}
		// Support wildcard subdomains (e.g., *.example.com)
		if c.matchesWildcard(allowedOrigin, origin) {
			return true
		}
	}
	return false
}

// matchesWildcard checks if origin matches a wildcard pattern
func (c *CORSMiddleware) matchesWildcard(pattern, origin string) bool {
	if !strings.Contains(pattern, "*") {
		return false
	}

	// Simple wildcard matching for subdomains
	if strings.HasPrefix(pattern, "*.") {
		domain := pattern[2:] // Remove "*."
		return strings.HasSuffix(origin, domain)
	}

	return false
}

// setCORSHeaders sets the appropriate CORS headers
func (c *CORSMiddleware) setCORSHeaders(w http.ResponseWriter, origin string) {
	// Set Access-Control-Allow-Origin
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	} else if len(c.config.AllowedOrigins) == 1 && c.config.AllowedOrigins[0] == "*" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	// Set Access-Control-Allow-Credentials
	if c.config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// Set Access-Control-Expose-Headers
	if len(c.config.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(c.config.ExposedHeaders, ", "))
	}
}

// handlePreflight handles CORS preflight OPTIONS requests
func (c *CORSMiddleware) handlePreflight(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	// Check if origin is allowed
	if origin == "" || !c.isOriginAllowed(origin) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Set CORS headers
	c.setCORSHeaders(w, origin)

	// Set Access-Control-Allow-Methods
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.config.AllowedMethods, ", "))

	// Set Access-Control-Allow-Headers
	requestedHeaders := r.Header.Get("Access-Control-Request-Headers")
	if requestedHeaders != "" {
		// Validate requested headers against allowed headers
		if c.areHeadersAllowed(requestedHeaders) {
			w.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
		} else {
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.config.AllowedHeaders, ", "))
		}
	} else {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.config.AllowedHeaders, ", "))
	}

	// Set Access-Control-Max-Age
	w.Header().Set("Access-Control-Max-Age", strconv.Itoa(c.config.MaxAge))

	w.WriteHeader(http.StatusOK)
}

// areHeadersAllowed checks if all requested headers are in the allowed list
func (c *CORSMiddleware) areHeadersAllowed(requestedHeaders string) bool {
	headers := strings.Split(requestedHeaders, ",")
	allowedMap := make(map[string]bool)

	for _, header := range c.config.AllowedHeaders {
		allowedMap[strings.ToLower(strings.TrimSpace(header))] = true
	}

	for _, header := range headers {
		headerName := strings.ToLower(strings.TrimSpace(header))
		if !allowedMap[headerName] {
			return false
		}
	}

	return true
}
