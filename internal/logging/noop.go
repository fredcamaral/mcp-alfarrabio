// Package logging provides structured logging capabilities
package logging

import "context"

// NoOpLogger is a logger that discards all logs (useful for testing)
type NoOpLogger struct{}

// NewNoOpLogger creates a new no-op logger
func NewNoOpLogger() Logger {
	return &NoOpLogger{}
}

// Info logs an info message (no-op)
func (n *NoOpLogger) Info(msg string, fields ...interface{}) {}

// Warn logs a warning message (no-op)
func (n *NoOpLogger) Warn(msg string, fields ...interface{}) {}

// Error logs an error message (no-op)
func (n *NoOpLogger) Error(msg string, fields ...interface{}) {}

// Debug logs a debug message (no-op)
func (n *NoOpLogger) Debug(msg string, fields ...interface{}) {}

// Fatal logs a fatal message (no-op, does not exit)
func (n *NoOpLogger) Fatal(msg string, fields ...interface{}) {}

// InfoContext logs an info message with context (no-op)
func (n *NoOpLogger) InfoContext(ctx context.Context, msg string, fields ...interface{}) {}

// WarnContext logs a warning message with context (no-op)
func (n *NoOpLogger) WarnContext(ctx context.Context, msg string, fields ...interface{}) {}

// ErrorContext logs an error message with context (no-op)
func (n *NoOpLogger) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {}

// DebugContext logs a debug message with context (no-op)
func (n *NoOpLogger) DebugContext(ctx context.Context, msg string, fields ...interface{}) {}

// WithTraceID creates a new logger with a trace ID (returns self)
func (n *NoOpLogger) WithTraceID(traceID string) Logger {
	return n
}

// WithComponent creates a new logger with a component name (returns self)
func (n *NoOpLogger) WithComponent(component string) Logger {
	return n
}
