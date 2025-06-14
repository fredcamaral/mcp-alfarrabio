// Package performance provides high-performance architecture components
// including connection pooling, resource management, and optimization utilities.
package performance

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// ConnectionPool manages database connections with intelligent pooling
type ConnectionPool struct {
	config      *PoolConfig
	pool        chan *sql.DB
	activeConns map[*sql.DB]time.Time
	mutex       sync.RWMutex
	healthCheck chan struct{}
	closed      bool
}

// PoolConfig defines connection pool configuration
type PoolConfig struct {
	// Connection settings
	MaxConnections    int           `json:"max_connections"`
	MinConnections    int           `json:"min_connections"`
	MaxIdleTime       time.Duration `json:"max_idle_time"`
	MaxLifetime       time.Duration `json:"max_lifetime"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`

	// Health check settings
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MaxRetries          int           `json:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay"`

	// Database settings
	DatabaseURL    string        `json:"database_url"`
	ConnectTimeout time.Duration `json:"connect_timeout"`
	QueryTimeout   time.Duration `json:"query_timeout"`
}

// DefaultPoolConfig returns optimized default configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConnections:      25, // Optimized for typical workloads
		MinConnections:      5,  // Keep minimum connections warm
		MaxIdleTime:         30 * time.Minute,
		MaxLifetime:         1 * time.Hour,
		ConnectionTimeout:   10 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		MaxRetries:          3,
		RetryDelay:          1 * time.Second,
		ConnectTimeout:      10 * time.Second,
		QueryTimeout:        30 * time.Second,
	}
}

// NewConnectionPool creates a new high-performance connection pool
func NewConnectionPool(config *PoolConfig) (*ConnectionPool, error) {
	if config == nil {
		config = DefaultPoolConfig()
	}

	pool := &ConnectionPool{
		config:      config,
		pool:        make(chan *sql.DB, config.MaxConnections),
		activeConns: make(map[*sql.DB]time.Time),
		healthCheck: make(chan struct{}, 1),
	}

	// Initialize minimum connections
	if err := pool.initializeConnections(); err != nil {
		return nil, fmt.Errorf("failed to initialize connection pool: %w", err)
	}

	// Start health check routine
	go pool.healthCheckRoutine()

	return pool, nil
}

// Get acquires a connection from the pool
func (p *ConnectionPool) Get(ctx context.Context) (*sql.DB, error) {
	if p.closed {
		return nil, fmt.Errorf("connection pool is closed")
	}

	// Try to get connection with timeout
	ctx, cancel := context.WithTimeout(ctx, p.config.ConnectionTimeout)
	defer cancel()

	select {
	case conn := <-p.pool:
		// Check if connection is still healthy
		if err := p.pingConnection(ctx, conn); err != nil {
			// Connection is unhealthy, create new one
			if newConn, err := p.createConnection(ctx); err == nil {
				p.trackConnection(newConn)
				return newConn, nil
			}
			// If we can't create new connection, return error
			return nil, fmt.Errorf("failed to get healthy connection: %w", err)
		}

		p.trackConnection(conn)
		return conn, nil

	case <-ctx.Done():
		return nil, fmt.Errorf("connection timeout: %w", ctx.Err())

	default:
		// Pool is empty, try to create new connection
		if p.canCreateConnection() {
			conn, err := p.createConnection(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to create connection: %w", err)
			}
			p.trackConnection(conn)
			return conn, nil
		}

		// Wait for available connection
		select {
		case conn := <-p.pool:
			if err := p.pingConnection(ctx, conn); err != nil {
				return nil, fmt.Errorf("connection unhealthy: %w", err)
			}
			p.trackConnection(conn)
			return conn, nil
		case <-ctx.Done():
			return nil, fmt.Errorf("connection timeout: %w", ctx.Err())
		}
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(ctx context.Context, conn *sql.DB) {
	if p.closed || conn == nil {
		if conn != nil {
			_ = conn.Close()
		}
		return
	}

	p.mutex.Lock()
	delete(p.activeConns, conn)
	p.mutex.Unlock()

	// Check if connection is still healthy
	if err := p.pingConnection(ctx, conn); err != nil {
		_ = conn.Close()
		return
	}

	// Check if connection has exceeded lifetime
	if p.isConnectionExpired(conn) {
		_ = conn.Close()
		return
	}

	// Return to pool (non-blocking)
	select {
	case p.pool <- conn:
	default:
		// Pool is full, close connection
		_ = conn.Close()
	}
}

// Close gracefully shuts down the connection pool
func (p *ConnectionPool) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	close(p.healthCheck)

	// Close all pooled connections
	close(p.pool)
	for conn := range p.pool {
		_ = conn.Close()
	}

	// Close all active connections
	for conn := range p.activeConns {
		_ = conn.Close()
	}

	return nil
}

// Stats returns pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return PoolStats{
		TotalConnections:  len(p.activeConns) + len(p.pool),
		ActiveConnections: len(p.activeConns),
		IdleConnections:   len(p.pool),
		MaxConnections:    p.config.MaxConnections,
		HealthCheckErrors: 0, // Would track in real implementation
	}
}

// PoolStats provides connection pool statistics
type PoolStats struct {
	TotalConnections  int `json:"total_connections"`
	ActiveConnections int `json:"active_connections"`
	IdleConnections   int `json:"idle_connections"`
	MaxConnections    int `json:"max_connections"`
	HealthCheckErrors int `json:"health_check_errors"`
}

// Private methods

func (p *ConnectionPool) initializeConnections() error {
	ctx := context.Background()
	for i := 0; i < p.config.MinConnections; i++ {
		conn, err := p.createConnection(ctx)
		if err != nil {
			return fmt.Errorf("failed to create initial connection %d: %w", i+1, err)
		}

		p.pool <- conn
	}

	return nil
}

func (p *ConnectionPool) createConnection(ctx context.Context) (*sql.DB, error) {
	conn, err := sql.Open("postgres", p.config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection settings
	conn.SetMaxOpenConns(1) // Each connection manages itself
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(p.config.MaxLifetime)
	conn.SetConnMaxIdleTime(p.config.MaxIdleTime)

	// Test connection
	pingCtx, cancel := context.WithTimeout(ctx, p.config.ConnectTimeout)
	defer cancel()

	if err := conn.PingContext(pingCtx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping new connection: %w", err)
	}

	return conn, nil
}

func (p *ConnectionPool) pingConnection(ctx context.Context, conn *sql.DB) error {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return conn.PingContext(pingCtx)
}

func (p *ConnectionPool) trackConnection(conn *sql.DB) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.activeConns[conn] = time.Now()
}

func (p *ConnectionPool) canCreateConnection() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	total := len(p.activeConns) + len(p.pool)
	return total < p.config.MaxConnections
}

func (p *ConnectionPool) isConnectionExpired(conn *sql.DB) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if createdAt, exists := p.activeConns[conn]; exists {
		return time.Since(createdAt) > p.config.MaxLifetime
	}

	return false
}

func (p *ConnectionPool) healthCheckRoutine() {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.performHealthCheck()
		case <-p.healthCheck:
			return
		}
	}
}

func (p *ConnectionPool) performHealthCheck() {
	// Check idle connections in pool
	var healthyConns []*sql.DB

	// Drain pool temporarily
	for {
		select {
		case conn := <-p.pool:
			if err := p.pingConnection(context.Background(), conn); err == nil {
				healthyConns = append(healthyConns, conn)
			} else {
				_ = conn.Close()
			}
		default:
			// Pool is empty
			goto refillPool
		}
	}

refillPool:
	// Return healthy connections to pool
	for _, conn := range healthyConns {
		select {
		case p.pool <- conn:
		default:
			// Pool is full, close excess connections
			_ = conn.Close()
		}
	}

	// Ensure minimum connections
	for len(p.pool) < p.config.MinConnections && p.canCreateConnection() {
		if conn, err := p.createConnection(context.Background()); err == nil {
			p.pool <- conn
		} else {
			break // Stop trying if we can't create connections
		}
	}
}

// WithConnection executes a function with a pooled connection
func (p *ConnectionPool) WithConnection(ctx context.Context, fn func(*sql.DB) error) error {
	conn, err := p.Get(ctx)
	if err != nil {
		return err
	}
	defer p.Put(ctx, conn)

	return fn(conn)
}

// WithTransaction executes a function within a database transaction
func (p *ConnectionPool) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	return p.WithConnection(ctx, func(conn *sql.DB) error {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		defer func() {
			if r := recover(); r != nil {
				_ = tx.Rollback()
				panic(r)
			}
		}()

		if err := fn(tx); err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				return fmt.Errorf("transaction error: %w, rollback error: %w", err, rbErr)
			}
			return err
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	})
}
