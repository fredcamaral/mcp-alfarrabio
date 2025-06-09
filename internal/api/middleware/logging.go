package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// RequestIDKey is the context key for request ID
type contextKey string

const RequestIDKey contextKey = "request_id"

// LoggingMiddleware provides request/response logging capabilities
type LoggingMiddleware struct {
	logger *log.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: log.Default(),
	}
}

// Handler returns the logging middleware handler
func (lm *LoggingMiddleware) Handler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Generate request ID if not present
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Add request ID to context and response headers
			ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
			r = r.WithContext(ctx)
			w.Header().Set("X-Request-ID", requestID)

			// Create response writer wrapper to capture status code
			wrapper := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Log request
			lm.logRequest(r, requestID)

			// Process request
			next.ServeHTTP(wrapper, r)

			// Log response
			duration := time.Since(start)
			lm.logResponse(r, wrapper.statusCode, duration, requestID)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// logRequest logs incoming HTTP requests
func (lm *LoggingMiddleware) logRequest(r *http.Request, requestID string) {
	// Skip logging for health checks to reduce noise
	if r.URL.Path == "/health" || r.URL.Path == "/api/v1/health" {
		return
	}

	lm.logger.Printf("[%s] --> %s %s %s | User-Agent: %s | Remote: %s",
		requestID,
		r.Method,
		r.URL.Path,
		r.Proto,
		r.Header.Get("User-Agent"),
		r.RemoteAddr,
	)

	// Log important headers
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		lm.logger.Printf("[%s] Content-Type: %s", requestID, contentType)
	}

	if clientVersion := r.Header.Get("X-Client-Version"); clientVersion != "" {
		lm.logger.Printf("[%s] Client-Version: %s", requestID, clientVersion)
	}
}

// logResponse logs HTTP response information
func (lm *LoggingMiddleware) logResponse(r *http.Request, statusCode int, duration time.Duration, requestID string) {
	// Skip logging for health checks to reduce noise
	if r.URL.Path == "/health" || r.URL.Path == "/api/v1/health" {
		return
	}

	// Determine log level based on status code
	statusIcon := lm.getStatusIcon(statusCode)
	
	lm.logger.Printf("[%s] <-- %s %d %s | %v | %s %s",
		requestID,
		statusIcon,
		statusCode,
		http.StatusText(statusCode),
		duration,
		r.Method,
		r.URL.Path,
	)

	// Log slow requests
	if duration > 1*time.Second {
		lm.logger.Printf("[%s] SLOW REQUEST: %v for %s %s", 
			requestID, duration, r.Method, r.URL.Path)
	}

	// Log errors
	if statusCode >= 400 {
		lm.logger.Printf("[%s] ERROR: %d %s for %s %s", 
			requestID, statusCode, http.StatusText(statusCode), r.Method, r.URL.Path)
	}
}

// getStatusIcon returns an icon based on HTTP status code
func (lm *LoggingMiddleware) getStatusIcon(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "✅"
	case statusCode >= 300 && statusCode < 400:
		return "↩️"
	case statusCode >= 400 && statusCode < 500:
		return "❌"
	case statusCode >= 500:
		return "💥"
	default:
		return "❓"
	}
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}