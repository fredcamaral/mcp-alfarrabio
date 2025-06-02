// Package examples demonstrates how to integrate the standardized error handling
// system into the MCP Memory Server
package examples

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	
	"mcp-memory/internal/errors"
	
	"github.com/fredcamaral/gomcp-sdk/protocol"
)

// ExampleMCPHandler demonstrates how to integrate standardized error handling
// into MCP tool handlers
func ExampleMCPHandler() {
	// Initialize error handler
	errorHandler := errors.NewMCPErrorHandler()
	
	// Example MCP tool handler with proper error handling
	memoryCreateHandler := func(_ context.Context, req *protocol.JSONRPCRequest) *protocol.JSONRPCResponse {
		// Parse tool parameters (simplified example)
		// In real implementation, you would parse req.Params properly
		var params map[string]interface{}
		
		// Example validation
		if params == nil {
			err := errors.NewValidationError("options", "must be an object", nil)
			return errorHandler.HandleJSONRPCError(err, req.ID)
		}
		
		// Validate operation parameter (example)
		operation := "store_chunk" // In real code, extract from req.Params
		if operation == "" {
			err := errors.NewRequiredFieldError("operation")
			return errorHandler.HandleJSONRPCError(err, req.ID)
		}
		
		// Use helper validation functions
		if err := errorHandler.ValidateMemoryCreateParams(operation, params); err != nil {
			return errorHandler.HandleJSONRPCError(err, req.ID)
		}
		
		// Simulate successful operation
		result := map[string]interface{}{
			"chunk_id":  "550e8400-e29b-41d4-a716-446655440000",
			"type":      "chunk",
			"summary":   "Example chunk stored successfully",
			"stored_at": "2024-01-15T10:30:00Z",
		}
		
		return &protocol.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}
	}
	
	// Register the handler (this is just an example - actual registration would be in server setup)
	_ = memoryCreateHandler
}

// ExampleHTTPHandler demonstrates error handling for HTTP endpoints
func ExampleHTTPHandler() {
	errorHandler := errors.NewMCPErrorHandler()
	
	// Example HTTP endpoint with standardized error handling
	httpHandler := func(w http.ResponseWriter, r *http.Request) {
		// Validate authentication (when implemented)
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			err := errors.NewUnauthorizedError("missing_api_key")
			errorHandler.HandleHTTPError(w, err)
			return
		}
		
		// Validate API key (placeholder)
		if !isValidAPIKey(apiKey) {
			err := errors.ErrInvalidAPIKey
			errorHandler.HandleHTTPError(w, err)
			return
		}
		
		// Example successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "success", "message": "Request processed successfully"}`))
	}
	
	_ = httpHandler
}

// ExampleValidationChain demonstrates comprehensive parameter validation
func ExampleValidationChain() {
	errorHandler := errors.NewMCPErrorHandler()
	
	validateMemorySearchRequest := func(params map[string]interface{}) *errors.StandardError {
		// Chain multiple validations
		if err := errorHandler.ValidateRepositoryParam(params); err != nil {
			return err
		}
		
		// Validate query parameter
		query, exists := params["query"]
		if !exists || query == "" {
			return errors.NewRequiredFieldError("query")
		}
		
		queryStr, ok := query.(string)
		if !ok {
			return errors.NewValidationError("query", "must be a string", query)
		}
		
		// Validate query length
		if len(queryStr) > 1000 {
			return errors.NewValidationError("query", "must be less than 1000 characters", len(queryStr))
		}
		
		// Validate optional limit parameter
		if limit, exists := params["limit"]; exists {
			limitFloat, ok := limit.(float64)
			if !ok {
				return errors.NewValidationError("limit", "must be a number", limit)
			}
			
			limitInt := int(limitFloat)
			if limitInt < 1 || limitInt > 50 {
				return errors.NewValidationError("limit", "must be between 1 and 50", limitInt)
			}
		}
		
		// All validations passed
		return nil
	}
	
	_ = validateMemorySearchRequest
}

// ExampleErrorWrapping demonstrates wrapping internal errors
func ExampleErrorWrapping() {
	errorHandler := errors.NewMCPErrorHandler()
	
	processSearchRequest := func(query string) *errors.StandardError {
		// Simulate database operation
		if err := performDatabaseSearch(query); err != nil {
			return errorHandler.WrapDatabaseError("search operation", err)
		}
		
		// Simulate embedding generation
		if err := generateEmbeddings(query); err != nil {
			return errorHandler.WrapEmbeddingError("generate embeddings", err)
		}
		
		return nil
	}
	
	_ = processSearchRequest
}

// ExampleRateLimitingError demonstrates rate limiting error handling
func ExampleRateLimitingError() {
	// Simulate rate limiting check
	checkRateLimit := func(clientIP string) *errors.StandardError {
		// Check if rate limit exceeded (placeholder logic)
		if isRateLimited(clientIP) {
			return errors.NewRateLimitError(
				100,        // limit: 100 requests
				"1m",       // window: per minute
				60,         // retry_after: 60 seconds
				0,          // remaining: 0 requests left
			)
		}
		return nil
	}
	
	_ = checkRateLimit
}

// ExampleGraphQLErrorHandling demonstrates GraphQL error conversion
func ExampleGraphQLErrorHandling() {
	_ = errors.NewMCPErrorHandler() // Example handler for demonstration
	
	graphqlResolver := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		// Validate GraphQL arguments
		if repository, ok := args["repository"].(string); !ok || repository == "" {
			err := errors.NewRequiredFieldError("repository")
			// GraphQL resolvers return Go errors, which get converted by the GraphQL framework
			return nil, err
		}
		
		// For custom GraphQL error handling, you can also use:
		// graphqlError := errorHandler.HandleGraphQLError(err)
		// return nil, graphqlError
		
		return map[string]interface{}{"status": "success"}, nil
	}
	
	_ = graphqlResolver
}

// ExampleMultiProtocolErrorHandling shows how to handle errors across different protocols
func ExampleMultiProtocolErrorHandling() {
	_ = errors.NewMCPErrorHandler() // Example handler for demonstration
	
	handleError := func(err error, protocol string, w http.ResponseWriter, id interface{}) {
		if err == nil {
			return
		}
		
		// Convert to StandardError if needed
		var stdErr *errors.StandardError
		if stderrors.As(err, &stdErr) {
			// err is already a StandardError
		} else {
			stdErr = errors.NewInternalError("Operation failed", err)
		}
		
		// Handle based on protocol
		switch protocol {
		case "json-rpc":
			response := stdErr.WithProtocol("json-rpc").ToJSONRPCError(id)
			// Send JSON-RPC response
			_ = response
			
		case "http":
			stdErr.WithProtocol("http").WriteHTTPError(w)
			
		case "graphql":
			graphqlError := stdErr.WithProtocol("graphql").ToGraphQLError()
			// Return GraphQL error
			_ = graphqlError
			
		case "websocket":
			// For WebSocket, you might send JSON-RPC format
			response := stdErr.WithProtocol("websocket").ToJSONRPCError(id)
			// Send over WebSocket connection
			_ = response
		}
	}
	
	_ = handleError
}

// Helper functions (placeholders for demonstration)

func isValidAPIKey(apiKey string) bool {
	// Placeholder API key validation
	return apiKey != "" && apiKey != "invalid"
}

func isRateLimited(clientIP string) bool {
	// Placeholder rate limiting check
	return clientIP == "127.0.0.1" // Simulate rate limit for localhost
}

func performDatabaseSearch(query string) error {
	// Placeholder database operation
	if query == "error" {
		return fmt.Errorf("database connection timeout")
	}
	return nil
}

func generateEmbeddings(query string) error {
	// Placeholder embedding generation
	if query == "embedding_error" {
		return fmt.Errorf("OpenAI API quota exceeded")
	}
	return nil
}

// ExampleErrorLogging demonstrates structured error logging
func ExampleErrorLogging() {
	errorHandler := errors.NewMCPErrorHandler()
	
	logAndHandleError := func(ctx context.Context, operation string, err error, id interface{}) *protocol.JSONRPCResponse {
		if err == nil {
			return nil
		}
		
		// Convert to StandardError
		var stdErr *errors.StandardError
		if stderrors.As(err, &stdErr) {
			// err is already a StandardError
		} else {
			stdErr = errors.NewInternalError(fmt.Sprintf("Failed to %s", operation), err)
		}
		
		// Log the error with structured logging
		errorHandler.LogError(ctx, operation, stdErr)
		
		// Return JSON-RPC error response
		return errorHandler.HandleJSONRPCError(stdErr, id)
	}
	
	_ = logAndHandleError
}

// ExampleCustomErrorTypes demonstrates creating custom error types
func ExampleCustomErrorTypes() {
	// Create custom error for business logic
	createCustomBusinessError := func(reason string, details interface{}) *errors.StandardError {
		return &errors.StandardError{
			ErrorInfo: errors.ErrorDetails{
				Code:    "BUSINESS_LOGIC_ERROR",
				Message: fmt.Sprintf("Business rule violation: %s", reason),
				Details: details,
			},
		}
	}
	
	// Usage
	if insufficientPermissions() {
		err := createCustomBusinessError("insufficient permissions", map[string]interface{}{
			"required_permission": "write_access",
			"user_permissions":    []string{"read_access"},
		})
		_ = err
	}
}

func insufficientPermissions() bool {
	return true // Placeholder
}