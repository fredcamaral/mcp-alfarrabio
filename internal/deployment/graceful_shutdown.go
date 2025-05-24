package deployment

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fredcamaral/mcp-memory/internal/logging"
)

// ShutdownManager handles graceful shutdown of the application
type ShutdownManager struct {
	logger       logging.Logger
	timeout      time.Duration
	shutdownFuncs []ShutdownFunc
	mutex        sync.RWMutex
	shutdownChan chan os.Signal
	done         chan struct{}
}

// ShutdownFunc represents a function to be called during shutdown
type ShutdownFunc struct {
	Name     string
	Function func(ctx context.Context) error
	Priority int // Lower numbers shutdown first
}

// NewShutdownManager creates a new graceful shutdown manager
func NewShutdownManager(logger logging.Logger, timeout time.Duration) *ShutdownManager {
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	return &ShutdownManager{
		logger:       logger,
		timeout:      timeout,
		shutdownFuncs: make([]ShutdownFunc, 0),
		shutdownChan: shutdownChan,
		done:         make(chan struct{}),
	}
}

// RegisterShutdownFunc registers a function to be called during shutdown
func (sm *ShutdownManager) RegisterShutdownFunc(name string, priority int, fn func(ctx context.Context) error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	shutdownFunc := ShutdownFunc{
		Name:     name,
		Function: fn,
		Priority: priority,
	}

	// Insert in priority order (lower numbers first)
	inserted := false
	for i, existing := range sm.shutdownFuncs {
		if priority < existing.Priority {
			sm.shutdownFuncs = append(sm.shutdownFuncs[:i], append([]ShutdownFunc{shutdownFunc}, sm.shutdownFuncs[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		sm.shutdownFuncs = append(sm.shutdownFuncs, shutdownFunc)
	}

	sm.logger.Info("Registered shutdown function", "name", name, "priority", priority)
}

// WaitForShutdown blocks until a shutdown signal is received
func (sm *ShutdownManager) WaitForShutdown() {
	select {
	case sig := <-sm.shutdownChan:
		sm.logger.Info("Received shutdown signal", "signal", sig.String())
		sm.performShutdown()
	case <-sm.done:
		sm.logger.Info("Shutdown completed")
	}
}

// Shutdown initiates the shutdown process programmatically
func (sm *ShutdownManager) Shutdown() {
	close(sm.done)
	sm.performShutdown()
}

// performShutdown executes all registered shutdown functions
func (sm *ShutdownManager) performShutdown() {
	sm.logger.Info("Starting graceful shutdown", "timeout", sm.timeout.String())

	ctx, cancel := context.WithTimeout(context.Background(), sm.timeout)
	defer cancel()

	sm.mutex.RLock()
	functions := make([]ShutdownFunc, len(sm.shutdownFuncs))
	copy(functions, sm.shutdownFuncs)
	sm.mutex.RUnlock()

	// Execute shutdown functions in priority order
	for _, fn := range functions {
		select {
		case <-ctx.Done():
			sm.logger.Error("Shutdown timeout reached", "remaining_function", fn.Name)
			return
		default:
			sm.logger.Info("Executing shutdown function", "name", fn.Name)
			start := time.Now()

			if err := fn.Function(ctx); err != nil {
				sm.logger.Error("Shutdown function failed", "name", fn.Name, "error", err)
			} else {
				sm.logger.Info("Shutdown function completed", "name", fn.Name, "duration", time.Since(start))
			}
		}
	}

	sm.logger.Info("Graceful shutdown completed")
}

// ShutdownHook provides common shutdown functions for different components
type ShutdownHook struct {
	logger logging.Logger
}

// NewShutdownHook creates a new shutdown hook helper
func NewShutdownHook(logger logging.Logger) *ShutdownHook {
	return &ShutdownHook{logger: logger}
}

// CreateHTTPServerShutdown creates a shutdown function for HTTP servers
func (sh *ShutdownHook) CreateHTTPServerShutdown(name string, server interface{}) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if srv, ok := server.(interface{ Shutdown(context.Context) error }); ok {
			sh.logger.Info("Shutting down HTTP server", "name", name)
			return srv.Shutdown(ctx)
		}
		return fmt.Errorf("server does not implement Shutdown method")
	}
}

// CreateDatabaseShutdown creates a shutdown function for database connections
func (sh *ShutdownHook) CreateDatabaseShutdown(name string, db interface{}) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if closer, ok := db.(interface{ Close() error }); ok {
			sh.logger.Info("Closing database connection", "name", name)
			return closer.Close()
		}
		return fmt.Errorf("database does not implement Close method")
	}
}

// CreateResourceCleanup creates a shutdown function for general resource cleanup
func (sh *ShutdownHook) CreateResourceCleanup(name string, cleanup func() error) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		sh.logger.Info("Cleaning up resource", "name", name)
		return cleanup()
	}
}

// CreateAsyncWorkerShutdown creates a shutdown function for async workers
func (sh *ShutdownHook) CreateAsyncWorkerShutdown(name string, stopChan chan<- struct{}, doneChan <-chan struct{}) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		sh.logger.Info("Stopping async worker", "name", name)
		
		// Signal worker to stop
		close(stopChan)
		
		// Wait for worker to finish or timeout
		select {
		case <-doneChan:
			sh.logger.Info("Async worker stopped gracefully", "name", name)
			return nil
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for worker %s to stop", name)
		}
	}
}

// CreateConnectionPoolShutdown creates a shutdown function for connection pools
func (sh *ShutdownHook) CreateConnectionPoolShutdown(name string, pool interface{}) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		sh.logger.Info("Draining connection pool", "name", name)
		
		// Try common connection pool interfaces
		if drainer, ok := pool.(interface{ Drain() error }); ok {
			return drainer.Drain()
		}
		
		if closer, ok := pool.(interface{ Close() error }); ok {
			return closer.Close()
		}
		
		return fmt.Errorf("connection pool does not implement Drain or Close method")
	}
}