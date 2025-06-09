// Package pool provides database connection pooling functionality with monitoring and health checks.
package pool

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"lerian-mcp-memory/internal/config"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// ConnectionPool manages database connections with health monitoring
type ConnectionPool struct {
	db            *sql.DB
	config        *config.DatabaseConfig
	stats         *PoolStats
	healthChecker *HealthChecker
	mu            sync.RWMutex
}

// PoolStats tracks connection pool statistics
type PoolStats struct {
	TotalConns        int           `json:"total_connections"`
	IdleConns         int           `json:"idle_connections"`
	OpenConns         int           `json:"open_connections"`
	InUseConns        int           `json:"in_use_connections"`
	WaitCount         int64         `json:"wait_count"`
	WaitDuration      time.Duration `json:"wait_duration"`
	MaxIdleClosed     int64         `json:"max_idle_closed"`
	MaxIdleTimeClosed int64         `json:"max_idle_time_closed"`
	MaxLifetimeClosed int64         `json:"max_lifetime_closed"`
}

// HealthChecker monitors connection pool health
type HealthChecker struct {
	interval     time.Duration
	timeout      time.Duration
	failureCount int
	maxFailures  int
	isHealthy    bool
	lastCheck    time.Time
	lastError    error
	mu           sync.RWMutex
	stopCh       chan struct{}
	isRunning    bool
}

// NewConnectionPool creates a new database connection pool
func NewConnectionPool(cfg *config.DatabaseConfig) (*ConnectionPool, error) {
	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.QueryTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create health checker
	healthChecker := &HealthChecker{
		interval:    time.Minute,
		timeout:     time.Second * 10,
		maxFailures: 3,
		isHealthy:   true,
		stopCh:      make(chan struct{}),
	}

	pool := &ConnectionPool{
		db:            db,
		config:        cfg,
		stats:         &PoolStats{},
		healthChecker: healthChecker,
	}

	// Start health monitoring
	go pool.startHealthMonitoring()

	return pool, nil
}

// GetDB returns the underlying database connection
func (cp *ConnectionPool) GetDB() *sql.DB {
	return cp.db
}

// Query executes a query with timeout and monitoring
func (cp *ConnectionPool) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()

	// Add query timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cp.config.QueryTimeout)
		defer cancel()
	}

	rows, err := cp.db.QueryContext(ctx, query, args...)

	duration := time.Since(start)
	cp.recordQueryMetrics(query, duration, err)

	return rows, err
}

// QueryRow executes a single-row query with timeout and monitoring
func (cp *ConnectionPool) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()

	// Add query timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cp.config.QueryTimeout)
		defer cancel()
	}

	row := cp.db.QueryRowContext(ctx, query, args...)

	duration := time.Since(start)
	cp.recordQueryMetrics(query, duration, nil)

	return row
}

// Exec executes a command with timeout and monitoring
func (cp *ConnectionPool) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()

	// Add query timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cp.config.QueryTimeout)
		defer cancel()
	}

	result, err := cp.db.ExecContext(ctx, query, args...)

	duration := time.Since(start)
	cp.recordQueryMetrics(query, duration, err)

	return result, err
}

// Begin starts a transaction with timeout
func (cp *ConnectionPool) Begin(ctx context.Context) (*sql.Tx, error) {
	// Add query timeout if not already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cp.config.QueryTimeout)
		defer cancel()
	}

	return cp.db.BeginTx(ctx, nil)
}

// GetStats returns current connection pool statistics
func (cp *ConnectionPool) GetStats() *PoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	stats := cp.db.Stats()

	cp.stats.TotalConns = stats.MaxOpenConnections
	cp.stats.OpenConns = stats.OpenConnections
	cp.stats.InUseConns = stats.InUse
	cp.stats.IdleConns = stats.Idle
	cp.stats.WaitCount = stats.WaitCount
	cp.stats.WaitDuration = stats.WaitDuration
	cp.stats.MaxIdleClosed = stats.MaxIdleClosed
	cp.stats.MaxIdleTimeClosed = stats.MaxIdleTimeClosed
	cp.stats.MaxLifetimeClosed = stats.MaxLifetimeClosed

	return cp.stats
}

// IsHealthy returns the current health status
func (cp *ConnectionPool) IsHealthy() bool {
	return cp.healthChecker.IsHealthy()
}

// GetHealthStatus returns detailed health information
func (cp *ConnectionPool) GetHealthStatus() map[string]interface{} {
	return cp.healthChecker.GetStatus()
}

// recordQueryMetrics records query execution metrics
func (cp *ConnectionPool) recordQueryMetrics(query string, duration time.Duration, err error) {
	// Log slow queries if enabled
	if cp.config.EnableQueryLogging && duration > cp.config.SlowQueryThreshold {
		fmt.Printf("SLOW QUERY [%v]: %s\n", duration, query)
	}

	// Record metrics if enabled
	if cp.config.EnableMetrics {
		// TODO: Integrate with metrics system (Prometheus, etc.)
		// For now, just log to stdout in debug mode
		if duration > cp.config.SlowQueryThreshold {
			fmt.Printf("METRICS: query_duration=%v query_error=%v\n", duration, err != nil)
		}
	}
}

// startHealthMonitoring starts the health checker goroutine
func (cp *ConnectionPool) startHealthMonitoring() {
	cp.healthChecker.Start(cp.db)
}

// Close closes the connection pool and stops health monitoring
func (cp *ConnectionPool) Close() error {
	cp.healthChecker.Stop()
	return cp.db.Close()
}

// IsHealthy returns whether the health checker considers the pool healthy
func (hc *HealthChecker) IsHealthy() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.isHealthy
}

// GetStatus returns detailed health status information
func (hc *HealthChecker) GetStatus() map[string]interface{} {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	status := map[string]interface{}{
		"healthy":       hc.isHealthy,
		"failure_count": hc.failureCount,
		"max_failures":  hc.maxFailures,
		"last_check":    hc.lastCheck,
	}

	if hc.lastError != nil {
		status["last_error"] = hc.lastError.Error()
	}

	return status
}

// Start begins health checking
func (hc *HealthChecker) Start(db *sql.DB) {
	hc.mu.Lock()
	if hc.isRunning {
		hc.mu.Unlock()
		return
	}
	hc.isRunning = true
	hc.mu.Unlock()

	go hc.healthCheckLoop(db)
}

// Stop stops health checking
func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if !hc.isRunning {
		return
	}

	close(hc.stopCh)
	hc.isRunning = false
}

// healthCheckLoop runs the health check periodically
func (hc *HealthChecker) healthCheckLoop(db *sql.DB) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.performHealthCheck(db)
		case <-hc.stopCh:
			return
		}
	}
}

// performHealthCheck executes a health check
func (hc *HealthChecker) performHealthCheck(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	err := db.PingContext(ctx)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.lastCheck = time.Now()

	if err != nil {
		hc.failureCount++
		hc.lastError = err

		if hc.failureCount >= hc.maxFailures {
			hc.isHealthy = false
		}
	} else {
		hc.failureCount = 0
		hc.lastError = nil
		hc.isHealthy = true
	}
}

// setHealthy sets the health status (used for testing)
func (hc *HealthChecker) setHealthy(healthy bool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.isHealthy = healthy
}
