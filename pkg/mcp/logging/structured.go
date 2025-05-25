package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// LogLevel represents logging levels
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Config holds logger configuration
type Config struct {
	Level      LogLevel
	Format     string // "json" or "text"
	AddSource  bool
	TimeFormat string
}

// Logger wraps slog.Logger with MCP-specific methods
type Logger struct {
	*slog.Logger
	config Config
}

// NewLogger creates a new structured logger
func NewLogger(config Config) *Logger {
	var handler slog.Handler
	
	opts := &slog.HandlerOptions{
		Level:     parseLevel(config.Level),
		AddSource: config.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Custom time format
			if a.Key == slog.TimeKey && config.TimeFormat != "" {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(config.TimeFormat))
				}
			}
			return a
		},
	}

	switch config.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	return &Logger{
		Logger: logger,
		config: config,
	}
}

// WithContext returns a logger with context values
func (l *Logger) WithContext(ctx context.Context) *Logger {
	logger := l.Logger

	// Add trace information if available
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		logger = logger.With(
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}

	return &Logger{
		Logger: logger,
		config: l.config,
	}
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	
	return &Logger{
		Logger: l.Logger.With(attrs...),
		config: l.config,
	}
}

// WithError returns a logger with error field
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.Logger.With(slog.String("error", err.Error())),
		config: l.config,
	}
}

// Request logs MCP request information
func (l *Logger) Request(ctx context.Context, method string, id interface{}, params interface{}) {
	logger := l.WithContext(ctx)
	logger.Info("MCP request received",
		slog.String("method", method),
		slog.Any("id", id),
		slog.Any("params", params),
		slog.String("component", "mcp.server"),
	)
}

// Response logs MCP response information
func (l *Logger) Response(ctx context.Context, method string, id interface{}, result interface{}, duration time.Duration) {
	logger := l.WithContext(ctx)
	logger.Info("MCP response sent",
		slog.String("method", method),
		slog.Any("id", id),
		slog.Duration("duration", duration),
		slog.String("component", "mcp.server"),
	)
}

// ResponseError logs MCP error response
func (l *Logger) ResponseError(ctx context.Context, method string, id interface{}, err error, duration time.Duration) {
	logger := l.WithContext(ctx)
	logger.Error("MCP error response",
		slog.String("method", method),
		slog.Any("id", id),
		slog.String("error", err.Error()),
		slog.Duration("duration", duration),
		slog.String("component", "mcp.server"),
	)
}

// ToolExecution logs tool execution
func (l *Logger) ToolExecution(ctx context.Context, toolName string, args interface{}) {
	logger := l.WithContext(ctx)
	logger.Info("Tool execution started",
		slog.String("tool", toolName),
		slog.Any("args", args),
		slog.String("component", "mcp.tools"),
	)
}

// ToolResult logs tool execution result
func (l *Logger) ToolResult(ctx context.Context, toolName string, result interface{}, duration time.Duration) {
	logger := l.WithContext(ctx)
	logger.Info("Tool execution completed",
		slog.String("tool", toolName),
		slog.Duration("duration", duration),
		slog.String("component", "mcp.tools"),
	)
}

// ToolError logs tool execution error
func (l *Logger) ToolError(ctx context.Context, toolName string, err error, duration time.Duration) {
	logger := l.WithContext(ctx)
	logger.Error("Tool execution failed",
		slog.String("tool", toolName),
		slog.String("error", err.Error()),
		slog.Duration("duration", duration),
		slog.String("component", "mcp.tools"),
	)
}

// ResourceOperation logs resource operations
func (l *Logger) ResourceOperation(ctx context.Context, operation, resourceType, uri string) {
	logger := l.WithContext(ctx)
	logger.Info("Resource operation",
		slog.String("operation", operation),
		slog.String("resource_type", resourceType),
		slog.String("uri", uri),
		slog.String("component", "mcp.resources"),
	)
}

// PromptOperation logs prompt operations
func (l *Logger) PromptOperation(ctx context.Context, operation, promptName string) {
	logger := l.WithContext(ctx)
	logger.Info("Prompt operation",
		slog.String("operation", operation),
		slog.String("prompt", promptName),
		slog.String("component", "mcp.prompts"),
	)
}

// WebSocketConnection logs WebSocket connection events
func (l *Logger) WebSocketConnection(event string, remoteAddr string) {
	l.Info("WebSocket connection event",
		slog.String("event", event),
		slog.String("remote_addr", remoteAddr),
		slog.String("component", "mcp.transport.websocket"),
	)
}

// parseLevel converts LogLevel to slog.Level
func parseLevel(level LogLevel) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Middleware provides logging middleware functionality
type Middleware struct {
	logger *Logger
}

// NewMiddleware creates a new logging middleware
func NewMiddleware(logger *Logger) *Middleware {
	return &Middleware{logger: logger}
}

// LogRequest logs incoming requests with recovery
func (m *Middleware) LogRequest(ctx context.Context, method string, id interface{}, fn func() (interface{}, error)) (result interface{}, err error) {
	start := time.Now()
	
	// Log request
	m.logger.Request(ctx, method, id, nil)
	
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
			
			// Log stack trace
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			m.logger.WithContext(ctx).Error("Panic in request handler",
				slog.String("method", method),
				slog.Any("panic", r),
				slog.String("stack", string(buf[:n])),
			)
		}
	}()
	
	// Execute function
	result, err = fn()
	
	// Log response
	duration := time.Since(start)
	if err != nil {
		m.logger.ResponseError(ctx, method, id, err, duration)
	} else {
		m.logger.Response(ctx, method, id, result, duration)
	}
	
	return result, err
}

// LogTool logs tool execution with recovery
func (m *Middleware) LogTool(ctx context.Context, toolName string, args interface{}, fn func() (interface{}, error)) (result interface{}, err error) {
	start := time.Now()
	
	// Log execution start
	m.logger.ToolExecution(ctx, toolName, args)
	
	// Recover from panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
			
			// Log stack trace
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			m.logger.WithContext(ctx).Error("Panic in tool execution",
				slog.String("tool", toolName),
				slog.Any("panic", r),
				slog.String("stack", string(buf[:n])),
			)
		}
	}()
	
	// Execute function
	result, err = fn()
	
	// Log result
	duration := time.Since(start)
	if err != nil {
		m.logger.ToolError(ctx, toolName, err, duration)
	} else {
		m.logger.ToolResult(ctx, toolName, result, duration)
	}
	
	return result, err
}

// Global logger instance
var defaultLogger = NewLogger(Config{
	Level:  LevelInfo,
	Format: "json",
})

// SetDefault sets the default logger
func SetDefault(logger *Logger) {
	defaultLogger = logger
}

// Default returns the default logger
func Default() *Logger {
	return defaultLogger
}