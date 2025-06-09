// Package response provides standardized HTTP response structures and utilities
// for the MCP Memory Server API layer.
package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// ErrorCode represents standardized error codes for the API
type ErrorCode string

const (
	// Client error codes (4xx)
	ErrorCodeBadRequest          ErrorCode = "BAD_REQUEST"
	ErrorCodeUnauthorized        ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden           ErrorCode = "FORBIDDEN"
	ErrorCodeNotFound            ErrorCode = "NOT_FOUND"
	ErrorCodeMethodNotAllowed    ErrorCode = "METHOD_NOT_ALLOWED"
	ErrorCodeValidationFailed    ErrorCode = "VALIDATION_FAILED"
	ErrorCodeVersionMismatch     ErrorCode = "VERSION_MISMATCH"
	ErrorCodeRateLimited         ErrorCode = "RATE_LIMITED"

	// Server error codes (5xx)
	ErrorCodeInternalError       ErrorCode = "INTERNAL_ERROR"
	ErrorCodeServiceUnavailable  ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorCodeTimeout             ErrorCode = "TIMEOUT"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error     ErrorDetails `json:"error"`
	Timestamp string       `json:"timestamp"`
	RequestID string       `json:"request_id,omitempty"`
}

// ErrorDetails contains detailed error information
type ErrorDetails struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
}

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Data      interface{} `json:"data"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// WriteError writes a standardized error response
func WriteError(w http.ResponseWriter, statusCode int, code ErrorCode, message string, details ...string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorDetails := ErrorDetails{
		Code:    code,
		Message: message,
	}

	if len(details) > 0 {
		errorDetails.Details = details[0]
	}

	response := ErrorResponse{
		Error:     errorDetails,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: getRequestID(w),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback to simple error if JSON encoding fails
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// WriteSuccess writes a standardized success response
func WriteSuccess(w http.ResponseWriter, data interface{}, message ...string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := SuccessResponse{
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if len(message) > 0 {
		response.Message = message[0]
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		WriteError(w, http.StatusInternalServerError, ErrorCodeInternalError, "Failed to encode response")
	}
}

// WriteBadRequest writes a 400 Bad Request error
func WriteBadRequest(w http.ResponseWriter, message string, details ...string) {
	WriteError(w, http.StatusBadRequest, ErrorCodeBadRequest, message, details...)
}

// WriteUnauthorized writes a 401 Unauthorized error
func WriteUnauthorized(w http.ResponseWriter, message string, details ...string) {
	WriteError(w, http.StatusUnauthorized, ErrorCodeUnauthorized, message, details...)
}

// WriteForbidden writes a 403 Forbidden error
func WriteForbidden(w http.ResponseWriter, message string, details ...string) {
	WriteError(w, http.StatusForbidden, ErrorCodeForbidden, message, details...)
}

// WriteNotFound writes a 404 Not Found error
func WriteNotFound(w http.ResponseWriter, message string, details ...string) {
	WriteError(w, http.StatusNotFound, ErrorCodeNotFound, message, details...)
}

// WriteMethodNotAllowed writes a 405 Method Not Allowed error
func WriteMethodNotAllowed(w http.ResponseWriter, message string, details ...string) {
	WriteError(w, http.StatusMethodNotAllowed, ErrorCodeMethodNotAllowed, message, details...)
}

// WriteValidationError writes a 422 Validation Failed error
func WriteValidationError(w http.ResponseWriter, message string, details ...string) {
	WriteError(w, http.StatusUnprocessableEntity, ErrorCodeValidationFailed, message, details...)
}

// WriteVersionMismatch writes a 400 Version Mismatch error
func WriteVersionMismatch(w http.ResponseWriter, message string, details ...string) {
	WriteError(w, http.StatusBadRequest, ErrorCodeVersionMismatch, message, details...)
}

// WriteRateLimited writes a 429 Rate Limited error
func WriteRateLimited(w http.ResponseWriter, message string, details ...string) {
	WriteError(w, http.StatusTooManyRequests, ErrorCodeRateLimited, message, details...)
}

// WriteInternalError writes a 500 Internal Server Error
func WriteInternalError(w http.ResponseWriter, message string, details ...string) {
	WriteError(w, http.StatusInternalServerError, ErrorCodeInternalError, message, details...)
}

// WriteServiceUnavailable writes a 503 Service Unavailable error
func WriteServiceUnavailable(w http.ResponseWriter, message string, details ...string) {
	WriteError(w, http.StatusServiceUnavailable, ErrorCodeServiceUnavailable, message, details...)
}

// WriteTimeout writes a 504 Gateway Timeout error
func WriteTimeout(w http.ResponseWriter, message string, details ...string) {
	WriteError(w, http.StatusGatewayTimeout, ErrorCodeTimeout, message, details...)
}

// getRequestID extracts request ID from response writer or request context
func getRequestID(w http.ResponseWriter) string {
	// Try to get request ID from headers set by middleware
	if reqID := w.Header().Get("X-Request-ID"); reqID != "" {
		return reqID
	}
	return ""
}