package logging

import (
	"log"
	"os"
)

// Logger interface for structured logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
}

// SimpleLogger implements a basic logger using standard library
type SimpleLogger struct {
	logger *log.Logger
	level  LogLevel
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

// NewLogger creates a new simple logger
func NewLogger(level LogLevel) Logger {
	return &SimpleLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile),
		level:  level,
	}
}

// Info logs an info message
func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	if l.level <= INFO {
		l.logger.Printf("[INFO] %s %v", msg, fields)
	}
}

// Warn logs a warning message
func (l *SimpleLogger) Warn(msg string, fields ...interface{}) {
	if l.level <= WARN {
		l.logger.Printf("[WARN] %s %v", msg, fields)
	}
}

// Error logs an error message
func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	if l.level <= ERROR {
		l.logger.Printf("[ERROR] %s %v", msg, fields)
	}
}

// Debug logs a debug message
func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	if l.level <= DEBUG {
		l.logger.Printf("[DEBUG] %s %v", msg, fields)
	}
}

// Fatal logs a fatal message and exits
func (l *SimpleLogger) Fatal(msg string, fields ...interface{}) {
	l.logger.Printf("[FATAL] %s %v", msg, fields)
	os.Exit(1)
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