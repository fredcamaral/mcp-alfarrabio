// Package errors provides MCP-specific error handling utilities
package errors

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/fredcamaral/gomcp-sdk/protocol"
)

// MCPErrorHandler provides MCP-specific error handling functionality
type MCPErrorHandler struct {
	traceIDGenerator func() string
}

// NewMCPErrorHandler creates a new MCP error handler
func NewMCPErrorHandler() *MCPErrorHandler {
	return &MCPErrorHandler{
		traceIDGenerator: generateTraceID,
	}
}

// HandleJSONRPCError processes errors for JSON-RPC responses (MCP protocol)
func (h *MCPErrorHandler) HandleJSONRPCError(err error, id interface{}) *protocol.JSONRPCResponse {
	if err == nil {
		return nil
	}

	// Check if it's already a StandardError
	var stdErr *StandardError
	if errors.As(err, &stdErr) {
		return stdErr.WithTraceID(h.traceIDGenerator()).WithProtocol("json-rpc").ToJSONRPCError(id)
	}

	// Convert regular error to StandardError
	stdErr = NewInternalError("Request processing failed", err)
	return stdErr.WithTraceID(h.traceIDGenerator()).WithProtocol("json-rpc").ToJSONRPCError(id)
}

// HandleHTTPError processes errors for HTTP responses
func (h *MCPErrorHandler) HandleHTTPError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	// Check if it's already a StandardError
	var stdErr *StandardError
	if errors.As(err, &stdErr) {
		stdErr.WithTraceID(h.traceIDGenerator()).WithProtocol("http").WriteHTTPError(w)
		return
	}

	// Convert regular error to StandardError
	stdErr = NewInternalError("HTTP request processing failed", err)
	stdErr.WithTraceID(h.traceIDGenerator()).WithProtocol("http").WriteHTTPError(w)
}

// HandleGraphQLError processes errors for GraphQL responses
func (h *MCPErrorHandler) HandleGraphQLError(err error) map[string]interface{} {
	if err == nil {
		return nil
	}

	// Check if it's already a StandardError
	var stdErr *StandardError
	if errors.As(err, &stdErr) {
		return stdErr.WithTraceID(h.traceIDGenerator()).WithProtocol("graphql").ToGraphQLError()
	}

	// Convert regular error to StandardError
	stdErr = NewInternalError("GraphQL request processing failed", err)
	return stdErr.WithTraceID(h.traceIDGenerator()).WithProtocol("graphql").ToGraphQLError()
}

// ValidateRequiredParams validates required parameters for MCP tools
func (h *MCPErrorHandler) ValidateRequiredParams(params map[string]interface{}, required []string) *StandardError {
	for _, field := range required {
		if value, exists := params[field]; !exists || value == nil || value == "" {
			return NewRequiredFieldError(field)
		}
	}
	return nil
}

// ValidateRepositoryParam specifically validates the repository parameter
func (h *MCPErrorHandler) ValidateRepositoryParam(params map[string]interface{}) *StandardError {
	repository, exists := params["repository"]
	if !exists {
		return NewRequiredFieldError("repository")
	}

	repoStr, ok := repository.(string)
	if !ok || repoStr == "" {
		return NewValidationError("repository", "must be a non-empty string", repository)
	}

	// Additional repository format validation
	if repoStr != "global" && !isValidRepositoryURL(repoStr) {
		return NewValidationError("repository", 
			"must be 'global' or a valid repository URL (e.g., 'github.com/user/repo')", 
			repository)
	}

	return nil
}

// ValidateSessionIDParam validates session_id parameter
func (h *MCPErrorHandler) ValidateSessionIDParam(params map[string]interface{}) *StandardError {
	sessionID, exists := params["session_id"]
	if !exists {
		return NewRequiredFieldError("session_id")
	}

	sessionStr, ok := sessionID.(string)
	if !ok || sessionStr == "" {
		return NewValidationError("session_id", "must be a non-empty string", sessionID)
	}

	return nil
}

// WrapInternalError wraps an internal error with context
func (h *MCPErrorHandler) WrapInternalError(operation string, err error) *StandardError {
	if err == nil {
		return nil
	}

	message := fmt.Sprintf("Failed to %s", operation)
	return NewInternalError(message, err).WithTraceID(h.traceIDGenerator())
}

// WrapDatabaseError wraps database-related errors
func (h *MCPErrorHandler) WrapDatabaseError(operation string, err error) *StandardError {
	if err == nil {
		return nil
	}

	details := map[string]interface{}{
		"operation": operation,
		"error":     err.Error(),
	}

	return &StandardError{
		ErrorInfo: ErrorDetails{
			Code:    ErrorCodeDatabaseError,
			Message: fmt.Sprintf("Database operation failed: %s", operation),
			Details: details,
			TraceID: h.traceIDGenerator(),
		},
	}
}

// WrapEmbeddingError wraps embedding service errors
func (h *MCPErrorHandler) WrapEmbeddingError(operation string, err error) *StandardError {
	if err == nil {
		return nil
	}

	details := map[string]interface{}{
		"operation": operation,
		"error":     err.Error(),
		"service":   "openai_embeddings",
	}

	return &StandardError{
		ErrorInfo: ErrorDetails{
			Code:    ErrorCodeEmbeddingError,
			Message: fmt.Sprintf("Embedding service failed: %s", operation),
			Details: details,
			TraceID: h.traceIDGenerator(),
		},
	}
}

// LogError logs standardized errors with structured logging
func (h *MCPErrorHandler) LogError(ctx context.Context, operation string, stdErr *StandardError) {
	// Log the error with structured format
	detailsStr := ""
	if stdErr.ErrorInfo.Details != nil {
		detailsStr = fmt.Sprintf(", details: %+v", stdErr.ErrorInfo.Details)
	}

	log.Printf("[ERROR] %s: %s (code: %s, trace_id: %s, protocol: %s%s)", 
		operation, 
		stdErr.ErrorInfo.Message, 
		string(stdErr.ErrorInfo.Code),
		stdErr.ErrorInfo.TraceID,
		stdErr.ErrorInfo.Protocol,
		detailsStr)
}

// Helper functions

// generateTraceID generates a unique trace ID for error tracking
func generateTraceID() string {
	// In production, this would generate a proper distributed trace ID
	// For now, using a simple timestamp-based ID
	return fmt.Sprintf("trace_%d", getCurrentTimestamp())
}

// getCurrentTimestamp returns current timestamp in microseconds
func getCurrentTimestamp() int64 {
	return time.Now().UnixNano() / 1000
}

// isValidRepositoryURL validates repository URL format
func isValidRepositoryURL(repo string) bool {
	// Basic validation for repository URLs
	// Patterns: github.com/user/repo, gitlab.com/group/project, etc.
	patterns := []string{
		`^github\.com/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`,
		`^gitlab\.com/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`,
		`^bitbucket\.org/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`,
		`^[a-zA-Z0-9_.-]+\.[a-zA-Z]{2,}/[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`, // Generic git hosting
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, repo); matched {
			return true
		}
	}

	return false
}

// Common parameter validation helpers

// ValidateMemoryCreateParams validates parameters for memory_create operations
func (h *MCPErrorHandler) ValidateMemoryCreateParams(operation string, options map[string]interface{}) *StandardError {
	// Repository is always required
	if err := h.ValidateRepositoryParam(options); err != nil {
		return err
	}

	switch operation {
	case "store_chunk":
		return h.ValidateRequiredParams(options, []string{"session_id", "content"})
	case "store_decision":
		return h.ValidateRequiredParams(options, []string{"session_id", "decision", "rationale"})
	case "create_thread":
		return h.ValidateRequiredParams(options, []string{"name", "description", "chunk_ids"})
	case "create_relationship":
		return h.ValidateRequiredParams(options, []string{"source_chunk_id", "target_chunk_id", "relation_type"})
	case "import_context":
		return h.ValidateRequiredParams(options, []string{"session_id", "data"})
	default:
		return NewValidationError("operation", "unsupported operation", operation)
	}
}

// ValidateMemoryReadParams validates parameters for memory_read operations
func (h *MCPErrorHandler) ValidateMemoryReadParams(operation string, options map[string]interface{}) *StandardError {
	// Repository is always required
	if err := h.ValidateRepositoryParam(options); err != nil {
		return err
	}

	switch operation {
	case "search", "search_multi_repo":
		if err := h.ValidateRequiredParams(options, []string{"query"}); err != nil {
			return err
		}
		if operation == "search_multi_repo" {
			return h.ValidateRequiredParams(options, []string{"session_id"})
		}
		return nil
	case "find_similar":
		return h.ValidateRequiredParams(options, []string{"problem"})
	case "get_relationships":
		return h.ValidateRequiredParams(options, []string{"chunk_id"})
	case "traverse_graph":
		return h.ValidateRequiredParams(options, []string{"start_chunk_id"})
	case "resolve_alias":
		return h.ValidateRequiredParams(options, []string{"alias_name"})
	case "get_bulk_progress":
		return h.ValidateRequiredParams(options, []string{"operation_id"})
	case "get_context", "get_patterns", "get_threads", "search_explained", "list_aliases":
		return nil // Only repository required
	default:
		return NewValidationError("operation", "unsupported operation", operation)
	}
}