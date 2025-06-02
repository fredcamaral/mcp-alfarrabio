package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Logger interface for structured logging with trace support
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
	
	// Context-aware logging with trace IDs
	InfoContext(ctx context.Context, msg string, fields ...interface{})
	WarnContext(ctx context.Context, msg string, fields ...interface{})
	ErrorContext(ctx context.Context, msg string, fields ...interface{})
	DebugContext(ctx context.Context, msg string, fields ...interface{})
	
	// Trace ID management
	WithTraceID(traceID string) Logger
	WithComponent(component string) Logger
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	TraceID   string                 `json:"trace_id,omitempty"`
	Component string                 `json:"component,omitempty"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// ContextKey represents keys used in context for trace IDs
type ContextKey string

const (
	TraceIDKey ContextKey = "trace_id"
)

// StructuredLogger implements structured logging with JSON output
type StructuredLogger struct {
	level     LogLevel
	traceID   string
	component string
	useJSON   bool
}

// LogLevel represents logging levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// NewLogger creates a new structured logger
func NewLogger(level LogLevel) Logger {
	return &StructuredLogger{
		level:   level,
		useJSON: getEnvBool("LOG_JSON", true),
	}
}

// NewLoggerWithTrace creates a logger with a trace ID
func NewLoggerWithTrace(level LogLevel, traceID string) Logger {
	return &StructuredLogger{
		level:   level,
		traceID: traceID,
		useJSON: getEnvBool("LOG_JSON", true),
	}
}

// getEnvBool gets a boolean environment variable with default
func getEnvBool(key string, defaultValue bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val == "true" || val == "1"
}

// WithTraceID creates a new logger with a trace ID
func (l *StructuredLogger) WithTraceID(traceID string) Logger {
	return &StructuredLogger{
		level:     l.level,
		traceID:   traceID,
		component: l.component,
		useJSON:   l.useJSON,
	}
}

// WithComponent creates a new logger with a component name
func (l *StructuredLogger) WithComponent(component string) Logger {
	return &StructuredLogger{
		level:     l.level,
		traceID:   l.traceID,
		component: component,
		useJSON:   l.useJSON,
	}
}

// Info logs an info message
func (l *StructuredLogger) Info(msg string, fields ...interface{}) {
	if l.level <= INFO {
		l.logEntry("INFO", msg, "", fields...)
	}
}

// InfoContext logs an info message with context
func (l *StructuredLogger) InfoContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.level <= INFO {
		traceID := l.extractTraceID(ctx)
		l.logEntry("INFO", msg, traceID, fields...)
	}
}

// Warn logs a warning message
func (l *StructuredLogger) Warn(msg string, fields ...interface{}) {
	if l.level <= WARN {
		l.logEntry("WARN", msg, "", fields...)
	}
}

// WarnContext logs a warning message with context
func (l *StructuredLogger) WarnContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.level <= WARN {
		traceID := l.extractTraceID(ctx)
		l.logEntry("WARN", msg, traceID, fields...)
	}
}

// Error logs an error message
func (l *StructuredLogger) Error(msg string, fields ...interface{}) {
	if l.level <= ERROR {
		l.logEntry("ERROR", msg, "", fields...)
	}
}

// ErrorContext logs an error message with context
func (l *StructuredLogger) ErrorContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.level <= ERROR {
		traceID := l.extractTraceID(ctx)
		l.logEntry("ERROR", msg, traceID, fields...)
	}
}

// Debug logs a debug message
func (l *StructuredLogger) Debug(msg string, fields ...interface{}) {
	if l.level <= DEBUG {
		l.logEntry("DEBUG", msg, "", fields...)
	}
}

// DebugContext logs a debug message with context
func (l *StructuredLogger) DebugContext(ctx context.Context, msg string, fields ...interface{}) {
	if l.level <= DEBUG {
		traceID := l.extractTraceID(ctx)
		l.logEntry("DEBUG", msg, traceID, fields...)
	}
}

// Fatal logs a fatal message and exits
func (l *StructuredLogger) Fatal(msg string, fields ...interface{}) {
	l.logEntry("FATAL", msg, "", fields...)
	os.Exit(1)
}

// logEntry creates and outputs a structured log entry
func (l *StructuredLogger) logEntry(level, msg, contextTraceID string, fields ...interface{}) {
	// Determine trace ID (context takes precedence)
	traceID := l.traceID
	if contextTraceID != "" {
		traceID = contextTraceID
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		file = "unknown"
		line = 0
	} else {
		// Extract just the filename
		parts := strings.Split(file, "/")
		file = parts[len(parts)-1]
	}

	// Parse fields into key-value pairs
	fieldMap := make(map[string]interface{})
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fmt.Sprintf("%v", fields[i])
			fieldMap[key] = fields[i+1]
		} else {
			fieldMap[fmt.Sprintf("field_%d", i)] = fields[i]
		}
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Level:     level,
		Message:   msg,
		TraceID:   traceID,
		Component: l.component,
		File:      file,
		Line:      line,
		Fields:    fieldMap,
	}

	if l.useJSON {
		l.outputJSON(entry)
	} else {
		l.outputText(entry)
	}
}

// outputJSON outputs the log entry as JSON
func (l *StructuredLogger) outputJSON(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// outputText outputs the log entry as human-readable text
func (l *StructuredLogger) outputText(entry LogEntry) {
	var parts []string
	
	parts = append(parts, entry.Timestamp)
	parts = append(parts, fmt.Sprintf("[%s]", entry.Level))
	
	if entry.TraceID != "" {
		parts = append(parts, fmt.Sprintf("trace:%s", entry.TraceID[:8]))
	}
	
	if entry.Component != "" {
		parts = append(parts, fmt.Sprintf("component:%s", entry.Component))
	}
	
	parts = append(parts, entry.Message)
	
	if len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	
	if entry.File != "" && entry.Line > 0 {
		parts = append(parts, fmt.Sprintf("(%s:%d)", entry.File, entry.Line))
	}
	
	fmt.Println(strings.Join(parts, " "))
}

// extractTraceID extracts trace ID from context
func (l *StructuredLogger) extractTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	
	return ""
}

// Default logger instance
var defaultLogger = NewLogger(INFO)

// Package-level functions for convenience
func Info(msg string, fields ...interface{}) {
	defaultLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...interface{}) {
	defaultLogger.Warn(msg, fields...)
}

func Error(msg string, fields ...interface{}) {
	defaultLogger.Error(msg, fields...)
}

func Debug(msg string, fields ...interface{}) {
	defaultLogger.Debug(msg, fields...)
}

func Fatal(msg string, fields ...interface{}) {
	defaultLogger.Fatal(msg, fields...)
}

// Context-aware package functions
func InfoContext(ctx context.Context, msg string, fields ...interface{}) {
	defaultLogger.InfoContext(ctx, msg, fields...)
}

func WarnContext(ctx context.Context, msg string, fields ...interface{}) {
	defaultLogger.WarnContext(ctx, msg, fields...)
}

func ErrorContext(ctx context.Context, msg string, fields ...interface{}) {
	defaultLogger.ErrorContext(ctx, msg, fields...)
}

func DebugContext(ctx context.Context, msg string, fields ...interface{}) {
	defaultLogger.DebugContext(ctx, msg, fields...)
}

// Trace ID utilities
func GenerateTraceID() string {
	return uuid.New().String()
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		traceID = GenerateTraceID()
	}
	return context.WithValue(ctx, TraceIDKey, traceID)
}

func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// Component logger creation
func WithComponent(component string) Logger {
	return defaultLogger.WithComponent(component)
}

// Level parsing from string
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// SetDefaultLogger sets the default logger instance
func SetDefaultLogger(logger Logger) {
	defaultLogger = logger
}
