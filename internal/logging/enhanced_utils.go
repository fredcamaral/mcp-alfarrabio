package logging

import (
	"context"
	"time"

	mcperrors "lerian-mcp-memory/internal/errors"
)

// LogField provides a structured way to add fields to logs
type LogField struct {
	Key   string
	Value interface{}
}

// EnhancedLogger wraps the existing StructuredLogger with additional utilities
type EnhancedLogger struct {
	Logger
	component string
}

// NewEnhancedLogger creates an enhanced logger for a component
func NewEnhancedLogger(component string) *EnhancedLogger {
	baseLogger := NewLogger(INFO)
	return &EnhancedLogger{
		Logger:    baseLogger.WithComponent(component),
		component: component,
	}
}

// WithContext creates a logger with context information
func (l *EnhancedLogger) WithContext(ctx context.Context) *EnhancedLogger {
	traceID := getTraceIDFromContext(ctx)
	newLogger := l.Logger.WithTraceID(traceID)

	return &EnhancedLogger{
		Logger:    newLogger,
		component: l.component,
	}
}

// WithError logs an error with enhanced error information
func (l *EnhancedLogger) WithError(err error) *EnhancedLogger {
	if err == nil {
		return l
	}

	// If it's an enhanced error, extract additional information
	if enhancedErr, ok := err.(*mcperrors.EnhancedError); ok {
		// For now, log basic error information
		// This could be extended to use the enhanced error context
		l.Error("Enhanced error occurred",
			"error", err.Error(),
			"category", string(enhancedErr.GetCategory()),
			"retryable", enhancedErr.IsRetryable(),
			"component", enhancedErr.Context.Component,
			"operation", enhancedErr.Context.Operation,
		)
	} else {
		l.Error("Error occurred", "error", err.Error())
	}

	return l
}

// LogOperation logs the start and completion of an operation
func (l *EnhancedLogger) LogOperation(operation string, fn func() error) error {
	startTime := time.Now()
	l.Info("Starting operation", "operation", operation)

	err := fn()
	duration := time.Since(startTime)

	if err != nil {
		l.Error("Operation failed",
			"operation", operation,
			"duration_ms", duration.Milliseconds(),
			"error", err.Error(),
		)
		return err
	}

	l.Info("Operation completed",
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

// LogSlowOperation logs operations that exceed expected duration
func (l *EnhancedLogger) LogSlowOperation(operation string, duration, expected time.Duration) {
	l.Warn("Slow operation detected",
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
		"expected_ms", expected.Milliseconds(),
		"slowdown_factor", float64(duration)/float64(expected),
	)
}

// Migration helpers for standard log package

// LogMigrationHelper provides helpers for migrating from standard log
type LogMigrationHelper struct {
	*EnhancedLogger
}

// NewLogMigrationHelper creates a helper for migrating standard log calls
func NewLogMigrationHelper(component string) *LogMigrationHelper {
	return &LogMigrationHelper{
		EnhancedLogger: NewEnhancedLogger(component),
	}
}

// Printf mimics log.Printf but with structured logging
func (h *LogMigrationHelper) Printf(format string, args ...interface{}) {
	// Simple implementation - enhance as needed
	h.Info(format, args...)
}

// Print mimics log.Print
func (h *LogMigrationHelper) Print(args ...interface{}) {
	h.Info("log message", "args", args)
}

// Println mimics log.Println
func (h *LogMigrationHelper) Println(args ...interface{}) {
	h.Info("log message", "args", args)
}

// Fatal mimics log.Fatal
func (h *LogMigrationHelper) Fatal(args ...interface{}) {
	h.EnhancedLogger.Fatal("fatal error", "args", args)
}

// Fatalf mimics log.Fatalf
func (h *LogMigrationHelper) Fatalf(format string, args ...interface{}) {
	h.EnhancedLogger.Fatal(format, "args", args)
}

// Utility functions

// getTraceIDFromContext extracts trace ID from context
func getTraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		return traceID
	}
	// Try the logging package's trace ID key
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// Global logger instances for different components
var (
	ServerLogger   = NewEnhancedLogger("server")
	MCPLogger      = NewEnhancedLogger("mcp")
	DatabaseLogger = NewEnhancedLogger("database")
	AILogger       = NewEnhancedLogger("ai")
	StorageLogger  = NewEnhancedLogger("storage")
	CLILogger      = NewEnhancedLogger("cli")
)

// GetComponentLogger returns an enhanced logger for specific component
func GetComponentLogger(component string) *EnhancedLogger {
	return NewEnhancedLogger(component)
}
