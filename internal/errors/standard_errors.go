// Package errors provides standardized error handling across all API protocols
package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fredcamaral/gomcp-sdk/protocol"
)

// ErrorCode represents semantic error codes for consistent error handling
type ErrorCode string

const (
	// Authentication and authorization errors
	ErrorCodeUnauthorized  ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden     ErrorCode = "FORBIDDEN"
	ErrorCodeInvalidAPIKey ErrorCode = "INVALID_API_KEY" //nolint:gosec // This is an error code, not credentials

	// Validation errors
	ErrorCodeValidationError ErrorCode = "VALIDATION_ERROR"
	ErrorCodeRequiredField   ErrorCode = "REQUIRED_FIELD"
	ErrorCodeInvalidFormat   ErrorCode = "INVALID_FORMAT"
	ErrorCodeInvalidValue    ErrorCode = "INVALID_VALUE"

	// Resource errors
	ErrorCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrorCodeAlreadyExists ErrorCode = "ALREADY_EXISTS"
	ErrorCodeConflict      ErrorCode = "CONFLICT"

	// Rate limiting and quota errors
	ErrorCodeRateLimited   ErrorCode = "RATE_LIMITED"
	ErrorCodeQuotaExceeded ErrorCode = "QUOTA_EXCEEDED"

	// System and processing errors
	ErrorCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrorCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorCodeTimeout            ErrorCode = "TIMEOUT"
	ErrorCodeDatabaseError      ErrorCode = "DATABASE_ERROR"
	ErrorCodeEmbeddingError     ErrorCode = "EMBEDDING_ERROR"

	// Repository and session errors
	ErrorCodeInvalidRepository  ErrorCode = "INVALID_REPOSITORY"
	ErrorCodeInvalidSession     ErrorCode = "INVALID_SESSION"
	ErrorCodeRepositoryNotFound ErrorCode = "REPOSITORY_NOT_FOUND"
)

// StandardError represents the unified error structure across all protocols
type StandardError struct {
	ErrorInfo ErrorDetails `json:"error"`
}

// Error implements the Go error interface
func (e *StandardError) Error() string {
	return e.ErrorInfo.Message
}

// ErrorDetails contains the detailed error information
type ErrorDetails struct {
	Code     ErrorCode   `json:"code"`
	Message  string      `json:"message"`
	Details  interface{} `json:"details,omitempty"`
	Protocol string      `json:"protocol,omitempty"`
	TraceID  string      `json:"trace_id,omitempty"`
}

// ValidationDetail provides specific validation error information
type ValidationDetail struct {
	Field  string      `json:"field"`
	Reason string      `json:"reason"`
	Value  interface{} `json:"value,omitempty"`
}

// RateLimitDetail provides rate limiting error information
type RateLimitDetail struct {
	Limit      int           `json:"limit"`
	Window     string        `json:"window"`
	RetryAfter time.Duration `json:"retry_after"`
	Remaining  int           `json:"remaining"`
}

// NewStandardError creates a new standardized error
func NewStandardError(code ErrorCode, message string, details interface{}) *StandardError {
	return &StandardError{
		ErrorInfo: ErrorDetails{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// NewValidationError creates a validation error with field details
func NewValidationError(field, reason string, value interface{}) *StandardError {
	return &StandardError{
		ErrorInfo: ErrorDetails{
			Code:    ErrorCodeValidationError,
			Message: fmt.Sprintf("Validation failed for field '%s': %s", field, reason),
			Details: ValidationDetail{
				Field:  field,
				Reason: reason,
				Value:  value,
			},
		},
	}
}

// NewRequiredFieldError creates an error for missing required fields
func NewRequiredFieldError(field string) *StandardError {
	return &StandardError{
		ErrorInfo: ErrorDetails{
			Code:    ErrorCodeRequiredField,
			Message: fmt.Sprintf("Required field '%s' is missing", field),
			Details: ValidationDetail{
				Field:  field,
				Reason: "missing_required_field",
			},
		},
	}
}

// NewRateLimitError creates a rate limiting error
func NewRateLimitError(limit int, window string, retryAfter time.Duration, remaining int) *StandardError {
	return &StandardError{
		ErrorInfo: ErrorDetails{
			Code:    ErrorCodeRateLimited,
			Message: fmt.Sprintf("Rate limit exceeded: %d requests per %s", limit, window),
			Details: RateLimitDetail{
				Limit:      limit,
				Window:     window,
				RetryAfter: retryAfter,
				Remaining:  remaining,
			},
		},
	}
}

// NewUnauthorizedError creates an unauthorized access error
func NewUnauthorizedError(reason string) *StandardError {
	return &StandardError{
		ErrorInfo: ErrorDetails{
			Code:    ErrorCodeUnauthorized,
			Message: "Authentication required",
			Details: map[string]interface{}{
				"reason": reason,
			},
		},
	}
}

// NewInternalError creates an internal server error
func NewInternalError(message string, originalError error) *StandardError {
	details := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	if originalError != nil {
		details["original_error"] = originalError.Error()
	}

	return &StandardError{
		ErrorInfo: ErrorDetails{
			Code:    ErrorCodeInternalError,
			Message: message,
			Details: details,
		},
	}
}

// WithTraceID adds a trace ID to the error for debugging
func (e *StandardError) WithTraceID(traceID string) *StandardError {
	e.ErrorInfo.TraceID = traceID
	return e
}

// WithProtocol adds protocol information to the error
func (e *StandardError) WithProtocol(protocolName string) *StandardError {
	e.ErrorInfo.Protocol = protocolName
	return e
}

// ToJSONRPCError converts StandardError to JSON-RPC error format
func (e *StandardError) ToJSONRPCError(id interface{}) *protocol.JSONRPCResponse {
	// Map semantic error codes to JSON-RPC error codes
	var rpcCode int
	switch e.ErrorInfo.Code {
	case ErrorCodeValidationError, ErrorCodeRequiredField, ErrorCodeInvalidFormat, ErrorCodeInvalidValue:
		rpcCode = -32602 // Invalid params
	case ErrorCodeNotFound, ErrorCodeRepositoryNotFound:
		rpcCode = -32601 // Method not found (closest equivalent)
	case ErrorCodeInternalError, ErrorCodeDatabaseError, ErrorCodeEmbeddingError:
		rpcCode = -32603 // Internal error
	case ErrorCodeUnauthorized, ErrorCodeForbidden, ErrorCodeInvalidAPIKey:
		rpcCode = -32000 // Server error (custom range)
	case ErrorCodeRateLimited, ErrorCodeQuotaExceeded:
		rpcCode = -32001 // Server error (custom range)
	case ErrorCodeAlreadyExists, ErrorCodeConflict:
		rpcCode = -32000 // Server error (custom range)
	case ErrorCodeServiceUnavailable, ErrorCodeTimeout:
		rpcCode = -32002 // Server error (custom range)
	case ErrorCodeInvalidRepository, ErrorCodeInvalidSession:
		rpcCode = -32602 // Invalid params
	default:
		rpcCode = -32603 // Internal error (fallback)
	}

	return &protocol.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &protocol.JSONRPCError{
			Code:    rpcCode,
			Message: e.ErrorInfo.Message,
			Data:    e,
		},
	}
}

// ToHTTPStatus maps StandardError to appropriate HTTP status code
func (e *StandardError) ToHTTPStatus() int {
	switch e.ErrorInfo.Code {
	case ErrorCodeUnauthorized, ErrorCodeInvalidAPIKey:
		return http.StatusUnauthorized
	case ErrorCodeForbidden:
		return http.StatusForbidden
	case ErrorCodeValidationError, ErrorCodeRequiredField, ErrorCodeInvalidFormat, ErrorCodeInvalidValue:
		return http.StatusBadRequest
	case ErrorCodeNotFound, ErrorCodeRepositoryNotFound:
		return http.StatusNotFound
	case ErrorCodeAlreadyExists, ErrorCodeConflict:
		return http.StatusConflict
	case ErrorCodeRateLimited, ErrorCodeQuotaExceeded:
		return http.StatusTooManyRequests
	case ErrorCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrorCodeTimeout:
		return http.StatusRequestTimeout
	case ErrorCodeInternalError, ErrorCodeDatabaseError, ErrorCodeEmbeddingError:
		return http.StatusInternalServerError
	case ErrorCodeInvalidRepository, ErrorCodeInvalidSession:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// ToGraphQLError converts StandardError to GraphQL error format
func (e *StandardError) ToGraphQLError() map[string]interface{} {
	return map[string]interface{}{
		"message": e.ErrorInfo.Message,
		"extensions": map[string]interface{}{
			"code":     string(e.ErrorInfo.Code),
			"details":  e.ErrorInfo.Details,
			"protocol": "graphql",
			"trace_id": e.ErrorInfo.TraceID,
		},
	}
}

// ToJSON converts StandardError to JSON bytes
func (e *StandardError) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// WriteHTTPError writes StandardError as HTTP response
func (e *StandardError) WriteHTTPError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")

	// Add trace ID header if present
	if e.ErrorInfo.TraceID != "" {
		w.Header().Set("X-Trace-ID", e.ErrorInfo.TraceID)
	}

	// Add rate limiting headers if applicable
	if e.ErrorInfo.Code == ErrorCodeRateLimited {
		if rateLimitDetail, ok := e.ErrorInfo.Details.(RateLimitDetail); ok {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", rateLimitDetail.RetryAfter.Seconds()))
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rateLimitDetail.Limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", rateLimitDetail.Remaining))
		}
	}

	w.WriteHeader(e.ToHTTPStatus())

	jsonBytes, _ := e.ToJSON()
	_, _ = w.Write(jsonBytes)
}

// Predefined common errors for convenience
var (
	ErrRepositoryRequired = NewRequiredFieldError("repository")
	ErrSessionIDRequired  = NewRequiredFieldError("session_id")
	ErrQueryRequired      = NewRequiredFieldError("query")
	ErrContentRequired    = NewRequiredFieldError("content")
	ErrDecisionRequired   = NewRequiredFieldError("decision")
	ErrRationaleRequired  = NewRequiredFieldError("rationale")

	ErrUnauthorizedAccess = NewUnauthorizedError("authentication_required")
	ErrInvalidAPIKey      = NewStandardError(ErrorCodeInvalidAPIKey, "Invalid API key provided", nil)

	ErrInternalServer     = NewInternalError("Internal server error occurred", nil)
	ErrServiceUnavailable = NewStandardError(ErrorCodeServiceUnavailable, "Service temporarily unavailable", nil)
)

// IsValidationError checks if the error is a validation-related error
func IsValidationError(err *StandardError) bool {
	return err.ErrorInfo.Code == ErrorCodeValidationError ||
		err.ErrorInfo.Code == ErrorCodeRequiredField ||
		err.ErrorInfo.Code == ErrorCodeInvalidFormat ||
		err.ErrorInfo.Code == ErrorCodeInvalidValue
}

func IsAuthenticationError(err *StandardError) bool {
	return err.ErrorInfo.Code == ErrorCodeUnauthorized ||
		err.ErrorInfo.Code == ErrorCodeForbidden ||
		err.ErrorInfo.Code == ErrorCodeInvalidAPIKey
}

func IsSystemError(err *StandardError) bool {
	return err.ErrorInfo.Code == ErrorCodeInternalError ||
		err.ErrorInfo.Code == ErrorCodeServiceUnavailable ||
		err.ErrorInfo.Code == ErrorCodeTimeout ||
		err.ErrorInfo.Code == ErrorCodeDatabaseError ||
		err.ErrorInfo.Code == ErrorCodeEmbeddingError
}
