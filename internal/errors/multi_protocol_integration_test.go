package errors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fredcamaral/gomcp-sdk/protocol"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiProtocolErrorHandling tests error handling across all supported protocols
func TestMultiProtocolErrorHandling(t *testing.T) {
	testCases := []struct {
		name         string
		error        *StandardError
		traceID      string
		expectedCode string
	}{
		{
			name:         "validation error across protocols",
			error:        NewValidationError("repository", "invalid format", "bad-repo"),
			traceID:      "trace-validation-123",
			expectedCode: string(ErrorCodeValidationError),
		},
		{
			name:         "authentication error across protocols",
			error:        NewUnauthorizedError("invalid_api_key"),
			traceID:      "trace-auth-456",
			expectedCode: string(ErrorCodeUnauthorized),
		},
		{
			name:         "rate limit error across protocols",
			error:        NewRateLimitError(100, "1m", 60*time.Second, 5),
			traceID:      "trace-rate-789",
			expectedCode: string(ErrorCodeRateLimited),
		},
		{
			name:         "internal server error across protocols",
			error:        NewInternalError("database connection failed", nil),
			traceID:      "trace-internal-012",
			expectedCode: string(ErrorCodeInternalError),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test each protocol
			t.Run("HTTP", func(t *testing.T) {
				testHTTPErrorHandling(t, tc.error, tc.traceID, tc.expectedCode)
			})

			t.Run("JSON-RPC", func(t *testing.T) {
				testJSONRPCErrorHandling(t, tc.error, tc.traceID, tc.expectedCode)
			})

			t.Run("GraphQL", func(t *testing.T) {
				testGraphQLErrorHandling(t, tc.error, tc.traceID, tc.expectedCode)
			})

			t.Run("WebSocket", func(t *testing.T) {
				testWebSocketErrorHandling(t, tc.error, tc.traceID, tc.expectedCode)
			})
		})
	}
}

func testHTTPErrorHandling(t *testing.T, stdErr *StandardError, traceID, expectedCode string) {
	// Add trace ID to error
	errorWithTrace := stdErr.WithTraceID(traceID).WithProtocol("http")

	// Create HTTP response recorder
	recorder := httptest.NewRecorder()

	// Write error to HTTP response
	errorWithTrace.WriteHTTPError(recorder)

	// Verify HTTP status code
	expectedStatus := errorWithTrace.ToHTTPStatus()
	assert.Equal(t, expectedStatus, recorder.Code)

	// Verify Content-Type header
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	// Verify trace ID in response header
	if traceID != "" {
		assert.Equal(t, traceID, recorder.Header().Get("X-Trace-ID"))
	}

	// Verify JSON response body
	var response StandardError
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedCode, string(response.ErrorInfo.Code))
	assert.Equal(t, traceID, response.ErrorInfo.TraceID)
	assert.Equal(t, "http", response.ErrorInfo.Protocol)

	// Verify rate limit headers for rate limit errors
	if stdErr.ErrorInfo.Code == ErrorCodeRateLimited {
		assert.NotEmpty(t, recorder.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, recorder.Header().Get("Retry-After"))
	}
}

func testJSONRPCErrorHandling(t *testing.T, stdErr *StandardError, traceID, expectedCode string) {
	// Verify expected code matches the error
	assert.Equal(t, expectedCode, string(stdErr.ErrorInfo.Code))
	// Add trace ID to error
	errorWithTrace := stdErr.WithTraceID(traceID).WithProtocol("json-rpc")

	// Convert to JSON-RPC error
	jsonRPCError := errorWithTrace.ToJSONRPCError("test-request-id")

	// Verify JSON-RPC structure
	assert.Equal(t, "2.0", jsonRPCError.JSONRPC)
	assert.Equal(t, "test-request-id", jsonRPCError.ID)
	assert.NotNil(t, jsonRPCError.Error)

	// Verify error code mapping
	expectedJSONRPCCode := mapToJSONRPCCode(stdErr.ErrorInfo.Code)
	assert.Equal(t, expectedJSONRPCCode, jsonRPCError.Error.Code)

	// Verify error message
	assert.Equal(t, stdErr.ErrorInfo.Message, jsonRPCError.Error.Message)

	// Verify error data contains trace ID
	if data, ok := jsonRPCError.Error.Data.(*StandardError); ok {
		assert.Equal(t, traceID, data.ErrorInfo.TraceID)
		assert.Equal(t, "json-rpc", data.ErrorInfo.Protocol)
	}

	// Test serialization
	jsonBytes, err := json.Marshal(jsonRPCError)
	require.NoError(t, err)

	// Test deserialization
	var parsedResponse protocol.JSONRPCResponse
	err = json.Unmarshal(jsonBytes, &parsedResponse)
	require.NoError(t, err)

	assert.Equal(t, jsonRPCError.JSONRPC, parsedResponse.JSONRPC)
	assert.Equal(t, jsonRPCError.ID, parsedResponse.ID)
}

func testGraphQLErrorHandling(t *testing.T, stdErr *StandardError, traceID, expectedCode string) {
	// Add trace ID to error
	errorWithTrace := stdErr.WithTraceID(traceID).WithProtocol("graphql")

	// Convert to GraphQL error
	graphqlError := errorWithTrace.ToGraphQLError()

	// Verify GraphQL error structure
	assert.Equal(t, stdErr.ErrorInfo.Message, graphqlError["message"])

	// Verify extensions
	extensions, ok := graphqlError["extensions"].(map[string]interface{})
	require.True(t, ok)

	assert.Equal(t, expectedCode, extensions["code"])
	assert.Equal(t, traceID, extensions["trace_id"])
	assert.Equal(t, "graphql", extensions["protocol"])
	assert.Equal(t, stdErr.ErrorInfo.Details, extensions["details"])

	// Test JSON serialization of GraphQL error
	jsonBytes, err := json.Marshal(graphqlError)
	require.NoError(t, err)

	// Test deserialization
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, stdErr.ErrorInfo.Message, parsed["message"])
}

func testWebSocketErrorHandling(t *testing.T, stdErr *StandardError, traceID, expectedCode string) {
	// Verify expected code matches the error
	assert.Equal(t, expectedCode, string(stdErr.ErrorInfo.Code))
	// Create a test server for WebSocket
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
		}
		defer func() {
			if err := conn.Close(); err != nil {
				t.Logf("Failed to close connection: %v", err)
			}
		}()

		// Read message from client
		var msg map[string]interface{}
		err = conn.ReadJSON(&msg)
		if err != nil {
			t.Fatalf("Failed to read JSON: %v", err)
		}

		// Create error response
		errorWithTrace := stdErr.WithTraceID(traceID).WithProtocol("websocket")
		jsonRPCError := errorWithTrace.ToJSONRPCError(msg["id"])

		// Send error response
		err = conn.WriteJSON(jsonRPCError)
		if err != nil {
			t.Fatalf("Failed to write JSON: %v", err)
		}
	}))
	defer server.Close()

	// Connect to WebSocket server
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(url, nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		}()
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Logf("Failed to close connection: %v", err)
		}
	}()

	// Send test request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "test_method",
		"id":      "test-ws-id",
	}

	err = conn.WriteJSON(request)
	require.NoError(t, err)

	// Read error response
	var response protocol.JSONRPCResponse
	err = conn.ReadJSON(&response)
	require.NoError(t, err)

	// Verify error response
	assert.Equal(t, "2.0", response.JSONRPC)
	assert.Equal(t, "test-ws-id", response.ID)
	assert.NotNil(t, response.Error)

	// Verify error code and message
	expectedJSONRPCCode := mapToJSONRPCCode(stdErr.ErrorInfo.Code)
	assert.Equal(t, expectedJSONRPCCode, response.Error.Code)
	assert.Equal(t, stdErr.ErrorInfo.Message, response.Error.Message)
}

// TestErrorConsistencyAcrossProtocols ensures the same error produces consistent results across protocols
func TestErrorConsistencyAcrossProtocols(t *testing.T) {
	baseError := NewValidationError("repository", "invalid format", "bad-repo")
	traceID := "consistency-test-123"

	// Test consistency of error information
	httpError := baseError.WithTraceID(traceID).WithProtocol("http")
	jsonRPCError := baseError.WithTraceID(traceID).WithProtocol("json-rpc")
	graphqlError := baseError.WithTraceID(traceID).WithProtocol("graphql")
	wsError := baseError.WithTraceID(traceID).WithProtocol("websocket")

	// All should have the same core error information
	errors := []*StandardError{httpError, jsonRPCError, graphqlError, wsError}

	for i, err := range errors {
		t.Run(fmt.Sprintf("error_%d", i), func(t *testing.T) {
			assert.Equal(t, string(ErrorCodeValidationError), string(err.ErrorInfo.Code))
			assert.Equal(t, baseError.ErrorInfo.Message, err.ErrorInfo.Message)
			assert.Equal(t, traceID, err.ErrorInfo.TraceID)
			assert.Equal(t, baseError.ErrorInfo.Details, err.ErrorInfo.Details)
		})
	}
}

// contextKey is a type for context keys to avoid collisions
type contextKey string

const testTraceIDKey contextKey = "trace_id"

// TestErrorPropagationWithContext tests error propagation through context
func TestErrorPropagationWithContext(t *testing.T) {
	ctx := context.Background()
	traceID := "context-test-456"

	// Add trace ID to context
	ctx = context.WithValue(ctx, testTraceIDKey, traceID)

	// Create error and verify it can extract trace ID from context
	err := NewValidationError("test", "test reason", "test value")

	// Simulate context-aware error handling
	if contextTraceID, ok := ctx.Value(testTraceIDKey).(string); ok {
		err = err.WithTraceID(contextTraceID)
	}

	assert.Equal(t, traceID, err.ErrorInfo.TraceID)
}

// TestConcurrentErrorHandling tests error handling under concurrent load
func TestConcurrentErrorHandling(t *testing.T) {
	const numGoroutines = 100

	// Channel to collect errors
	errorChan := make(chan *StandardError, numGoroutines)

	// Start multiple goroutines creating and processing errors
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			traceID := fmt.Sprintf("concurrent-test-%d", id)
			err := NewValidationError("field", "reason", "value").
				WithTraceID(traceID).
				WithProtocol("http")

			// Test various conversions
			_ = err.ToHTTPStatus()
			_ = err.ToJSONRPCError(id)
			_ = err.ToGraphQLError()
			_, _ = err.ToJSON()

			errorChan <- err
		}(i)
	}

	// Collect all errors
	errors := make([]*StandardError, 0, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errorChan:
			errors = append(errors, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for error")
		}
	}

	// Verify all errors were processed correctly
	assert.Len(t, errors, numGoroutines)

	// Verify each error has unique trace ID
	traceIDs := make(map[string]bool)
	for _, err := range errors {
		assert.NotEmpty(t, err.ErrorInfo.TraceID)
		assert.False(t, traceIDs[err.ErrorInfo.TraceID], "Duplicate trace ID found")
		traceIDs[err.ErrorInfo.TraceID] = true
	}
}

// TestErrorMetadata tests that error metadata is preserved across transformations
func TestErrorMetadata(t *testing.T) {
	// Create error with complex metadata
	err := &StandardError{
		ErrorInfo: ErrorDetails{
			Code:    ErrorCodeValidationError,
			Message: "Complex validation error",
			Details: ValidationDetail{
				Field:  "repository",
				Reason: "invalid_format",
				Value:  "bad-repo",
			},
			TraceID:  "metadata-test-789",
			Protocol: "http",
		},
	}

	// Test HTTP transformation preserves metadata
	recorder := httptest.NewRecorder()
	err.WriteHTTPError(recorder)

	var httpResponse StandardError
	jsonErr := json.Unmarshal(recorder.Body.Bytes(), &httpResponse)
	require.NoError(t, jsonErr)

	assert.Equal(t, err.ErrorInfo.Code, httpResponse.ErrorInfo.Code)
	assert.Equal(t, err.ErrorInfo.TraceID, httpResponse.ErrorInfo.TraceID)
	assert.NotNil(t, httpResponse.ErrorInfo.Details)

	// Test JSON-RPC transformation preserves metadata
	jsonRPCErr := err.ToJSONRPCError("test-id")
	if data, ok := jsonRPCErr.Error.Data.(*StandardError); ok {
		assert.Equal(t, err.ErrorInfo.TraceID, data.ErrorInfo.TraceID)
		assert.Equal(t, err.ErrorInfo.Details, data.ErrorInfo.Details)
	}

	// Test GraphQL transformation preserves metadata
	graphqlErr := err.ToGraphQLError()
	extensions := graphqlErr["extensions"].(map[string]interface{})
	assert.Equal(t, err.ErrorInfo.TraceID, extensions["trace_id"])
	assert.Equal(t, err.ErrorInfo.Details, extensions["details"])
}

// Helper function to map error codes to JSON-RPC codes
func mapToJSONRPCCode(errorCode ErrorCode) int {
	switch errorCode {
	case ErrorCodeValidationError, ErrorCodeRequiredField, ErrorCodeInvalidFormat, ErrorCodeInvalidValue:
		return -32602 // Invalid params
	case ErrorCodeUnauthorized, ErrorCodeInvalidAPIKey, ErrorCodeForbidden:
		return -32000 // Server error
	case ErrorCodeRateLimited, ErrorCodeQuotaExceeded:
		return -32001 // Rate limited
	case ErrorCodeNotFound, ErrorCodeRepositoryNotFound:
		return -32000 // Server error
	case ErrorCodeAlreadyExists, ErrorCodeConflict:
		return -32000 // Server error
	case ErrorCodeTimeout:
		return -32000 // Server error
	case ErrorCodeInternalError, ErrorCodeDatabaseError, ErrorCodeEmbeddingError, ErrorCodeServiceUnavailable:
		return -32603 // Internal error
	case ErrorCodeInvalidRepository, ErrorCodeInvalidSession:
		return -32602 // Invalid params
	default:
		return -32603 // Internal error (fallback)
	}
}
