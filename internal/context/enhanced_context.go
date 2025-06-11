package context

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Context keys for request metadata
type contextKey string

const (
	TraceIDKey   contextKey = "trace_id"
	RequestIDKey contextKey = "request_id"
	UserIDKey    contextKey = "user_id"
	ComponentKey contextKey = "component"
	OperationKey contextKey = "operation"
	StartTimeKey contextKey = "start_time"
)

// ContextBuilder provides fluent interface for building enhanced contexts
type ContextBuilder struct {
	ctx context.Context
}

// NewBuilder creates a new context builder
func NewBuilder(parent context.Context) *ContextBuilder {
	if parent == nil {
		parent = context.Background()
	}
	return &ContextBuilder{ctx: parent}
}

// WithTraceID adds a trace ID to context
func (b *ContextBuilder) WithTraceID(traceID string) *ContextBuilder {
	if traceID == "" {
		traceID = generateTraceID()
	}
	b.ctx = context.WithValue(b.ctx, TraceIDKey, traceID)
	return b
}

// WithNewTraceID generates and adds a new trace ID
func (b *ContextBuilder) WithNewTraceID() *ContextBuilder {
	return b.WithTraceID(generateTraceID())
}

// WithRequestID adds a request ID to context
func (b *ContextBuilder) WithRequestID(requestID string) *ContextBuilder {
	if requestID == "" {
		requestID = generateRequestID()
	}
	b.ctx = context.WithValue(b.ctx, RequestIDKey, requestID)
	return b
}

// WithNewRequestID generates and adds a new request ID
func (b *ContextBuilder) WithNewRequestID() *ContextBuilder {
	return b.WithRequestID(generateRequestID())
}

// WithUserID adds a user ID to context
func (b *ContextBuilder) WithUserID(userID string) *ContextBuilder {
	b.ctx = context.WithValue(b.ctx, UserIDKey, userID)
	return b
}

// WithComponent adds component information to context
func (b *ContextBuilder) WithComponent(component string) *ContextBuilder {
	b.ctx = context.WithValue(b.ctx, ComponentKey, component)
	return b
}

// WithOperation adds operation information to context
func (b *ContextBuilder) WithOperation(operation string) *ContextBuilder {
	b.ctx = context.WithValue(b.ctx, OperationKey, operation)
	return b
}

// WithStartTime adds start time to context for duration tracking
func (b *ContextBuilder) WithStartTime(startTime time.Time) *ContextBuilder {
	b.ctx = context.WithValue(b.ctx, StartTimeKey, startTime)
	return b
}

// WithCurrentTime adds current time as start time
func (b *ContextBuilder) WithCurrentTime() *ContextBuilder {
	return b.WithStartTime(time.Now())
}

// WithTimeout adds timeout to context
func (b *ContextBuilder) WithTimeout(timeout time.Duration) (*ContextBuilder, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(b.ctx, timeout)
	return &ContextBuilder{ctx: ctx}, cancel
}

// WithDeadline adds deadline to context
func (b *ContextBuilder) WithDeadline(deadline time.Time) (*ContextBuilder, context.CancelFunc) {
	ctx, cancel := context.WithDeadline(b.ctx, deadline)
	return &ContextBuilder{ctx: ctx}, cancel
}

// WithCancel adds cancellation to context
func (b *ContextBuilder) WithCancel() (*ContextBuilder, context.CancelFunc) {
	ctx, cancel := context.WithCancel(b.ctx)
	return &ContextBuilder{ctx: ctx}, cancel
}

// Build returns the built context
func (b *ContextBuilder) Build() context.Context {
	return b.ctx
}

// Convenience functions for common context operations

// NewRequestContext creates a new request context with trace and request IDs
func NewRequestContext(parent context.Context) context.Context {
	return NewBuilder(parent).
		WithNewTraceID().
		WithNewRequestID().
		WithCurrentTime().
		Build()
}

// NewOperationContext creates a context for a specific operation
func NewOperationContext(parent context.Context, component, operation string, timeout time.Duration) (context.Context, context.CancelFunc) {
	builder, cancel := NewBuilder(parent).
		WithComponent(component).
		WithOperation(operation).
		WithCurrentTime().
		WithTimeout(timeout)

	return builder.Build(), cancel
}

// NewMCPContext creates a context for MCP operations
func NewMCPContext(parent context.Context, method string) context.Context {
	return NewBuilder(parent).
		WithComponent("mcp").
		WithOperation(method).
		WithCurrentTime().
		Build()
}

// NewDatabaseContext creates a context for database operations
func NewDatabaseContext(parent context.Context, operation string) (context.Context, context.CancelFunc) {
	return NewOperationContext(parent, "database", operation, 30*time.Second)
}

// NewAIContext creates a context for AI service operations
func NewAIContext(parent context.Context, model, operation string) (context.Context, context.CancelFunc) {
	builder, cancel := NewBuilder(parent).
		WithComponent("ai").
		WithOperation(operation).
		WithCurrentTime().
		WithTimeout(60 * time.Second) // AI operations can take longer

	// Add model as custom metadata (would need to extend context for this)
	ctx := builder.Build()

	return ctx, cancel
}

// NewStorageContext creates a context for storage operations
func NewStorageContext(parent context.Context, operation string) (context.Context, context.CancelFunc) {
	return NewOperationContext(parent, "storage", operation, 15*time.Second)
}

// Context extraction functions

// GetTraceID extracts trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetComponent extracts component from context
func GetComponent(ctx context.Context) string {
	if component, ok := ctx.Value(ComponentKey).(string); ok {
		return component
	}
	return ""
}

// GetOperation extracts operation from context
func GetOperation(ctx context.Context) string {
	if operation, ok := ctx.Value(OperationKey).(string); ok {
		return operation
	}
	return ""
}

// GetStartTime extracts start time from context
func GetStartTime(ctx context.Context) time.Time {
	if startTime, ok := ctx.Value(StartTimeKey).(time.Time); ok {
		return startTime
	}
	return time.Time{}
}

// GetElapsedTime calculates elapsed time since start time in context
func GetElapsedTime(ctx context.Context) time.Duration {
	startTime := GetStartTime(ctx)
	if startTime.IsZero() {
		return 0
	}
	return time.Since(startTime)
}

// Context validation and debugging

// ValidateContext checks if context has required fields
func ValidateContext(ctx context.Context, required ...contextKey) error {
	for _, key := range required {
		if ctx.Value(key) == nil {
			return fmt.Errorf("missing required context value: %s", key)
		}
	}
	return nil
}

// ContextSummary returns a summary of context for debugging
func ContextSummary(ctx context.Context) map[string]interface{} {
	summary := make(map[string]interface{})

	if traceID := GetTraceID(ctx); traceID != "" {
		summary["trace_id"] = traceID
	}
	if requestID := GetRequestID(ctx); requestID != "" {
		summary["request_id"] = requestID
	}
	if userID := GetUserID(ctx); userID != "" {
		summary["user_id"] = userID
	}
	if component := GetComponent(ctx); component != "" {
		summary["component"] = component
	}
	if operation := GetOperation(ctx); operation != "" {
		summary["operation"] = operation
	}
	if elapsed := GetElapsedTime(ctx); elapsed > 0 {
		summary["elapsed_ms"] = elapsed.Milliseconds()
	}

	// Add deadline information if present
	if deadline, ok := ctx.Deadline(); ok {
		summary["deadline"] = deadline.Format(time.RFC3339)
		summary["remaining_ms"] = time.Until(deadline).Milliseconds()
	}

	return summary
}

// Timeout helpers

// WithStandardTimeout applies standard timeout based on operation type
func WithStandardTimeout(ctx context.Context, operationType string) (context.Context, context.CancelFunc) {
	var timeout time.Duration

	switch operationType {
	case "database":
		timeout = 30 * time.Second
	case "ai_service":
		timeout = 60 * time.Second
	case "storage":
		timeout = 15 * time.Second
	case "mcp":
		timeout = 10 * time.Second
	case "http":
		timeout = 30 * time.Second
	default:
		timeout = 30 * time.Second
	}

	return context.WithTimeout(ctx, timeout)
}

// IsTimeoutError checks if error is due to context timeout
func IsTimeoutError(err error) bool {
	return err == context.DeadlineExceeded
}

// ID generation functions

// generateTraceID generates a new trace ID
func generateTraceID() string {
	// Use hex encoding of random bytes for trace ID
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to UUID if random fails
		return uuid.New().String()
	}
	return hex.EncodeToString(bytes)
}

// generateRequestID generates a new request ID
func generateRequestID() string {
	return uuid.New().String()
}

// Middleware helpers for HTTP handlers

// HTTPContextMiddleware adds request context to HTTP requests
func HTTPContextMiddleware(next func(context.Context)) func(context.Context) {
	return func(ctx context.Context) {
		// Add request context if not already present
		if GetTraceID(ctx) == "" {
			ctx = NewRequestContext(ctx)
		}
		next(ctx)
	}
}
