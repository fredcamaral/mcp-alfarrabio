package errors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStandardError_Creation(t *testing.T) {
	tests := []struct {
		name            string
		createError     func() *StandardError
		expectedCode    ErrorCode
		expectedMessage string
	}{
		{
			name: "validation error",
			createError: func() *StandardError {
				return NewValidationError("repository", "must be a valid URL", "invalid-repo")
			},
			expectedCode:    ErrorCodeValidationError,
			expectedMessage: "Validation failed for field 'repository': must be a valid URL",
		},
		{
			name: "required field error",
			createError: func() *StandardError {
				return NewRequiredFieldError("session_id")
			},
			expectedCode:    ErrorCodeRequiredField,
			expectedMessage: "Required field 'session_id' is missing",
		},
		{
			name: "rate limit error",
			createError: func() *StandardError {
				return NewRateLimitError(100, "1m", 60*time.Second, 0)
			},
			expectedCode:    ErrorCodeRateLimited,
			expectedMessage: "Rate limit exceeded: 100 requests per 1m",
		},
		{
			name: "unauthorized error",
			createError: func() *StandardError {
				return NewUnauthorizedError("missing_api_key")
			},
			expectedCode:    ErrorCodeUnauthorized,
			expectedMessage: "Authentication required",
		},
		{
			name: "internal error",
			createError: func() *StandardError {
				return NewInternalError("Database connection failed", assert.AnError)
			},
			expectedCode:    ErrorCodeInternalError,
			expectedMessage: "Database connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createError()

			assert.Equal(t, tt.expectedCode, err.ErrorInfo.Code)
			assert.Equal(t, tt.expectedMessage, err.ErrorInfo.Message)
			assert.NotNil(t, err.ErrorInfo.Details)
		})
	}
}

func TestStandardError_WithMethods(t *testing.T) {
	baseError := NewValidationError("test", "test reason", "test value")

	// Test WithTraceID
	errorWithTrace := baseError.WithTraceID("trace-123")
	assert.Equal(t, "trace-123", errorWithTrace.ErrorInfo.TraceID)

	// Test WithProtocol
	errorWithProtocol := baseError.WithProtocol("http")
	assert.Equal(t, "http", errorWithProtocol.ErrorInfo.Protocol)

	// Test chaining
	chainedError := baseError.WithTraceID("trace-456").WithProtocol("json-rpc")
	assert.Equal(t, "trace-456", chainedError.ErrorInfo.TraceID)
	assert.Equal(t, "json-rpc", chainedError.ErrorInfo.Protocol)
}

func TestStandardError_ToJSONRPCError(t *testing.T) {
	tests := []struct {
		name         string
		error        *StandardError
		expectedCode int
		id           interface{}
	}{
		{
			name:         "validation error maps to invalid params",
			error:        NewValidationError("test", "test reason", "test value"),
			expectedCode: -32602,
			id:           "test-id",
		},
		{
			name:         "unauthorized error maps to server error",
			error:        NewUnauthorizedError("test reason"),
			expectedCode: -32000,
			id:           123,
		},
		{
			name:         "internal error maps to internal error",
			error:        NewInternalError("test message", nil),
			expectedCode: -32603,
			id:           "internal-test",
		},
		{
			name:         "rate limit error maps to server error",
			error:        NewRateLimitError(100, "1m", 60*time.Second, 0),
			expectedCode: -32001,
			id:           "rate-limit-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonRPCError := tt.error.ToJSONRPCError(tt.id)

			assert.Equal(t, "2.0", jsonRPCError.JSONRPC)
			assert.Equal(t, tt.id, jsonRPCError.ID)
			assert.NotNil(t, jsonRPCError.Error)
			assert.Equal(t, tt.expectedCode, jsonRPCError.Error.Code)
			assert.Equal(t, tt.error.ErrorInfo.Message, jsonRPCError.Error.Message)
			assert.Equal(t, tt.error, jsonRPCError.Error.Data)
		})
	}
}

func TestStandardError_ToHTTPStatus(t *testing.T) {
	tests := []struct {
		name           string
		error          *StandardError
		expectedStatus int
	}{
		{
			name:           "validation error returns bad request",
			error:          NewValidationError("test", "test reason", "test value"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "required field error returns bad request",
			error:          NewRequiredFieldError("test"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized error returns unauthorized",
			error:          NewUnauthorizedError("test reason"),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "rate limit error returns too many requests",
			error:          NewRateLimitError(100, "1m", 60*time.Second, 0),
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:           "internal error returns internal server error",
			error:          NewInternalError("test message", nil),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "unknown error code returns internal server error",
			error:          &StandardError{ErrorInfo: ErrorDetails{Code: "UNKNOWN_ERROR"}},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.error.ToHTTPStatus()
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestStandardError_WriteHTTPError(t *testing.T) {
	tests := []struct {
		name           string
		error          *StandardError
		expectedStatus int
		checkHeaders   func(t *testing.T, headers http.Header)
	}{
		{
			name:           "validation error response",
			error:          NewValidationError("repository", "invalid format", "bad-repo"),
			expectedStatus: http.StatusBadRequest,
			checkHeaders: func(t *testing.T, headers http.Header) {
				assert.Equal(t, "application/json", headers.Get("Content-Type"))
			},
		},
		{
			name:           "rate limit error with headers",
			error:          NewRateLimitError(100, "1m", 60*time.Second, 5),
			expectedStatus: http.StatusTooManyRequests,
			checkHeaders: func(t *testing.T, headers http.Header) {
				assert.Equal(t, "application/json", headers.Get("Content-Type"))
				assert.Equal(t, "60", headers.Get("Retry-After"))
				assert.Equal(t, "100", headers.Get("X-RateLimit-Limit"))
				assert.Equal(t, "5", headers.Get("X-RateLimit-Remaining"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			tt.error.WriteHTTPError(recorder)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
			tt.checkHeaders(t, recorder.Header())

			// Verify JSON response body
			var response StandardError
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.Equal(t, tt.error.ErrorInfo.Code, response.ErrorInfo.Code)
			assert.Equal(t, tt.error.ErrorInfo.Message, response.ErrorInfo.Message)
		})
	}
}

func TestStandardError_ToGraphQLError(t *testing.T) {
	stdErr := NewValidationError("test", "test reason", "test value").
		WithTraceID("trace-123").
		WithProtocol("graphql")

	graphqlError := stdErr.ToGraphQLError()

	// Check structure
	assert.Equal(t, stdErr.ErrorInfo.Message, graphqlError["message"])

	extensions, ok := graphqlError["extensions"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, string(stdErr.ErrorInfo.Code), extensions["code"])
	assert.Equal(t, stdErr.ErrorInfo.Details, extensions["details"])
	assert.Equal(t, "graphql", extensions["protocol"])
	assert.Equal(t, "trace-123", extensions["trace_id"])
}

func TestStandardError_ToJSON(t *testing.T) {
	stdErr := NewValidationError("repository", "invalid format", "bad-repo").
		WithTraceID("trace-123").
		WithProtocol("http")

	jsonBytes, err := stdErr.ToJSON()
	require.NoError(t, err)

	// Verify JSON structure
	var parsed StandardError
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, stdErr.ErrorInfo.Code, parsed.ErrorInfo.Code)
	assert.Equal(t, stdErr.ErrorInfo.Message, parsed.ErrorInfo.Message)
	assert.Equal(t, stdErr.ErrorInfo.TraceID, parsed.ErrorInfo.TraceID)
	assert.Equal(t, stdErr.ErrorInfo.Protocol, parsed.ErrorInfo.Protocol)
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		error    *StandardError
		expected ErrorCode
	}{
		{
			name:     "repository required",
			error:    ErrRepositoryRequired,
			expected: ErrorCodeRequiredField,
		},
		{
			name:     "session ID required",
			error:    ErrSessionIDRequired,
			expected: ErrorCodeRequiredField,
		},
		{
			name:     "query required",
			error:    ErrQueryRequired,
			expected: ErrorCodeRequiredField,
		},
		{
			name:     "content required",
			error:    ErrContentRequired,
			expected: ErrorCodeRequiredField,
		},
		{
			name:     "unauthorized access",
			error:    ErrUnauthorizedAccess,
			expected: ErrorCodeUnauthorized,
		},
		{
			name:     "invalid API key",
			error:    ErrInvalidAPIKey,
			expected: ErrorCodeInvalidAPIKey,
		},
		{
			name:     "internal server error",
			error:    ErrInternalServer,
			expected: ErrorCodeInternalError,
		},
		{
			name:     "service unavailable",
			error:    ErrServiceUnavailable,
			expected: ErrorCodeServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.error.ErrorInfo.Code)
			assert.NotEmpty(t, tt.error.ErrorInfo.Message)
		})
	}
}

func TestErrorClassifiers(t *testing.T) {
	tests := []struct {
		name         string
		error        *StandardError
		isValidation bool
		isAuth       bool
		isSystem     bool
	}{
		{
			name:         "validation error",
			error:        NewValidationError("test", "test", "test"),
			isValidation: true,
			isAuth:       false,
			isSystem:     false,
		},
		{
			name:         "required field error",
			error:        NewRequiredFieldError("test"),
			isValidation: true,
			isAuth:       false,
			isSystem:     false,
		},
		{
			name:         "unauthorized error",
			error:        NewUnauthorizedError("test"),
			isValidation: false,
			isAuth:       true,
			isSystem:     false,
		},
		{
			name:         "invalid API key error",
			error:        ErrInvalidAPIKey,
			isValidation: false,
			isAuth:       true,
			isSystem:     false,
		},
		{
			name:         "internal error",
			error:        NewInternalError("test", nil),
			isValidation: false,
			isAuth:       false,
			isSystem:     true,
		},
		{
			name:         "database error",
			error:        &StandardError{ErrorInfo: ErrorDetails{Code: ErrorCodeDatabaseError}},
			isValidation: false,
			isAuth:       false,
			isSystem:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isValidation, IsValidationError(tt.error))
			assert.Equal(t, tt.isAuth, IsAuthenticationError(tt.error))
			assert.Equal(t, tt.isSystem, IsSystemError(tt.error))
		})
	}
}

func TestErrorDetails_Serialization(t *testing.T) {
	// Test complex error details serialization
	err := &StandardError{
		ErrorInfo: ErrorDetails{
			Code:    ErrorCodeValidationError,
			Message: "Complex validation error",
			Details: ValidationDetail{
				Field:  "repository",
				Reason: "invalid_format",
				Value:  "bad-repo",
			},
			Protocol: "http",
			TraceID:  "trace-123",
		},
	}

	// Serialize to JSON
	jsonBytes, serErr := json.Marshal(err)
	require.NoError(t, serErr)

	// Deserialize from JSON
	var parsed StandardError
	deserErr := json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, deserErr)

	// Verify all fields
	assert.Equal(t, err.ErrorInfo.Code, parsed.ErrorInfo.Code)
	assert.Equal(t, err.ErrorInfo.Message, parsed.ErrorInfo.Message)
	assert.Equal(t, err.ErrorInfo.Protocol, parsed.ErrorInfo.Protocol)
	assert.Equal(t, err.ErrorInfo.TraceID, parsed.ErrorInfo.TraceID)

	// Note: Details will be map[string]interface{} after JSON roundtrip
	assert.NotNil(t, parsed.ErrorInfo.Details)
}

// Benchmark tests
func BenchmarkStandardError_Creation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewValidationError("repository", "invalid format", "bad-repo")
	}
}

func BenchmarkStandardError_ToJSONRPCError(b *testing.B) {
	err := NewValidationError("repository", "invalid format", "bad-repo")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.ToJSONRPCError("test-id")
	}
}

func BenchmarkStandardError_ToJSON(b *testing.B) {
	err := NewValidationError("repository", "invalid format", "bad-repo")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = err.ToJSON()
	}
}
