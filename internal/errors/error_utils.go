package errors

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// ErrorCategory classifies errors for handling strategies
type ErrorCategory string

const (
	ErrorCategoryRetryable  ErrorCategory = "retryable"
	ErrorCategoryPermanent  ErrorCategory = "permanent"
	ErrorCategoryResource   ErrorCategory = "resource"
	ErrorCategoryTimeout    ErrorCategory = "timeout"
	ErrorCategoryRateLimit  ErrorCategory = "rate_limit"
	ErrorCategoryValidation ErrorCategory = "validation"
)

// ErrorContext provides additional context for debugging
type ErrorContext struct {
	Operation  string                 `json:"operation"`
	Component  string                 `json:"component"`
	TraceID    string                 `json:"trace_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Category   ErrorCategory          `json:"category"`
	Retryable  bool                   `json:"retryable"`
}

// EnhancedError wraps errors with production-ready context
type EnhancedError struct {
	Err     error        `json:"error"`
	Context ErrorContext `json:"context"`
}

func (e *EnhancedError) Error() string {
	return fmt.Sprintf("[%s:%s] %s", e.Context.Component, e.Context.Operation, e.Err.Error())
}

func (e *EnhancedError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if error can be retried
func (e *EnhancedError) IsRetryable() bool {
	return e.Context.Retryable
}

// GetCategory returns error category
func (e *EnhancedError) GetCategory() ErrorCategory {
	return e.Context.Category
}

// NewEnhancedError creates a new enhanced error with context
func NewEnhancedError(err error, component, operation string, category ErrorCategory) *EnhancedError {
	return &EnhancedError{
		Err: err,
		Context: ErrorContext{
			Operation:  operation,
			Component:  component,
			Category:   category,
			Retryable:  category == ErrorCategoryRetryable || category == ErrorCategoryTimeout || category == ErrorCategoryRateLimit,
			Timestamp:  time.Now(),
			StackTrace: getStackTrace(),
		},
	}
}

// WithContext adds context information to error
func (e *EnhancedError) WithContext(ctx context.Context) *EnhancedError {
	if traceID := getTraceID(ctx); traceID != "" {
		e.Context.TraceID = traceID
	}
	if requestID := getRequestID(ctx); requestID != "" {
		e.Context.RequestID = requestID
	}
	if userID := getUserID(ctx); userID != "" {
		e.Context.UserID = userID
	}
	return e
}

// WithMetadata adds metadata to error
func (e *EnhancedError) WithMetadata(key string, value interface{}) *EnhancedError {
	if e.Context.Metadata == nil {
		e.Context.Metadata = make(map[string]interface{})
	}
	e.Context.Metadata[key] = value
	return e
}

// Standard error wrapping functions for common components

// WrapDatabaseError wraps database operation errors
func WrapDatabaseError(err error, operation string) error {
	if err == nil {
		return nil
	}

	category := ErrorCategoryPermanent
	if isTemporaryError(err) {
		category = ErrorCategoryRetryable
	}

	return NewEnhancedError(err, "database", operation, category)
}

// WrapAIServiceError wraps AI service errors
func WrapAIServiceError(err error, model, operation string) error {
	if err == nil {
		return nil
	}

	category := ErrorCategoryPermanent
	if isRateLimitError(err) {
		category = ErrorCategoryRateLimit
	} else if isTemporaryError(err) {
		category = ErrorCategoryRetryable
	}

	enhanced := NewEnhancedError(err, "ai_service", operation, category)
	enhanced.WithMetadata("model", model)
	return enhanced
}

// WrapStorageError wraps vector storage errors
func WrapStorageError(err error, operation string) error {
	if err == nil {
		return nil
	}

	category := ErrorCategoryPermanent
	if isTemporaryError(err) {
		category = ErrorCategoryRetryable
	}

	return NewEnhancedError(err, "storage", operation, category)
}

// WrapValidationError wraps validation errors
func WrapValidationError(err error, field string) error {
	if err == nil {
		return nil
	}

	enhanced := NewEnhancedError(err, "validation", "field_validation", ErrorCategoryValidation)
	enhanced.WithMetadata("field", field)
	return enhanced
}

// WrapTimeoutError wraps timeout errors
func WrapTimeoutError(err error, operation string, timeout time.Duration) error {
	if err == nil {
		return nil
	}

	enhanced := NewEnhancedError(err, "timeout", operation, ErrorCategoryTimeout)
	enhanced.WithMetadata("timeout_duration", timeout.String())
	return enhanced
}

// WrapMCPError wraps MCP protocol errors
func WrapMCPError(err error, method string) error {
	if err == nil {
		return nil
	}

	category := ErrorCategoryPermanent
	if isTemporaryError(err) {
		category = ErrorCategoryRetryable
	}

	enhanced := NewEnhancedError(err, "mcp", method, category)
	return enhanced
}

// Helper functions

// getStackTrace captures current stack trace
func getStackTrace() string {
	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// Context extraction helpers
func getTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return traceID
	}
	return ""
}

func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

func getUserID(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return ""
}

// Error classification helpers
func isTemporaryError(err error) bool {
	// Check for common temporary error patterns
	msg := err.Error()
	temporaryPatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"service unavailable",
		"too many requests",
		"deadline exceeded",
		"context deadline exceeded",
	}

	for _, pattern := range temporaryPatterns {
		if contains(msg, pattern) {
			return true
		}
	}

	return false
}

func isRateLimitError(err error) bool {
	msg := err.Error()
	rateLimitPatterns := []string{
		"rate limit",
		"quota exceeded",
		"too many requests",
		"429",
	}

	for _, pattern := range rateLimitPatterns {
		if contains(msg, pattern) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOfSubstring(s, substr) >= 0))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
