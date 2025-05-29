// Package pool provides connection pooling for storage backends
package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrPoolClosed    = errors.New("pool is closed")
	ErrPoolExhausted = errors.New("pool is exhausted")
	ErrInvalidConn   = errors.New("invalid connection")
)

// Connection represents a pooled connection
type Connection interface {
	// IsAlive checks if the connection is still valid
	IsAlive() bool
	// Close closes the underlying connection
	Close() error
	// Reset resets the connection state
	Reset() error
}

// Factory creates new connections
type Factory func(ctx context.Context) (Connection, error)

// PoolConfig holds pool configuration
type PoolConfig struct {
	MaxSize             int           // Maximum number of connections
	MinSize             int           // Minimum number of connections to maintain
	MaxIdleTime         time.Duration // Maximum time a connection can be idle
	MaxLifetime         time.Duration // Maximum lifetime of a connection
	HealthCheckInterval time.Duration // How often to check connection health
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxSize:             10,
		MinSize:             2,
		MaxIdleTime:         30 * time.Minute,
		MaxLifetime:         2 * time.Hour,
		HealthCheckInterval: 1 * time.Minute,
	}
}

// pooledConn wraps a connection with metadata
type pooledConn struct {
	conn       Connection
	createdAt  time.Time
	lastUsedAt time.Time
	usageCount int64
	mu         sync.Mutex
}

func (pc *pooledConn) isExpired(maxLifetime, maxIdleTime time.Duration) bool {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	now := time.Now()
	if maxLifetime > 0 && now.Sub(pc.createdAt) > maxLifetime {
		return true
	}
	if maxIdleTime > 0 && now.Sub(pc.lastUsedAt) > maxIdleTime {
		return true
	}
	return false
}

func (pc *pooledConn) markUsed() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.lastUsedAt = time.Now()
	pc.usageCount++
}

// ConnectionPool manages a pool of connections
type ConnectionPool struct {
	config      *PoolConfig
	factory     Factory
	connections chan *pooledConn
	mu          sync.RWMutex
	closed      int32
	activeCount int32
	waitCount   int32

	// Metrics
	totalCreated   int64
	totalDestroyed int64
	totalErrors    int64

	// Health check
	healthTicker *time.Ticker
	healthDone   chan struct{}
	healthWg     sync.WaitGroup
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *PoolConfig, factory Factory) (*ConnectionPool, error) {
	if config == nil {
		config = DefaultPoolConfig()
	}

	if config.MaxSize <= 0 {
		return nil, errors.New("max size must be positive")
	}
	if config.MinSize < 0 || config.MinSize > config.MaxSize {
		return nil, errors.New("invalid min size")
	}

	pool := &ConnectionPool{
		config:      config,
		factory:     factory,
		connections: make(chan *pooledConn, config.MaxSize),
		healthDone:  make(chan struct{}),
	}

	// Pre-create minimum connections
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < config.MinSize; i++ {
		if err := pool.createConnection(ctx); err != nil {
			// Clean up any created connections
			_ = pool.Close()
			return nil, fmt.Errorf("failed to create initial connections: %w", err)
		}
	}

	// Start health check routine
	if config.HealthCheckInterval > 0 {
		pool.healthTicker = time.NewTicker(config.HealthCheckInterval)
		pool.healthWg.Add(1)
		go pool.healthCheckLoop()
	}

	return pool, nil
}

// Get acquires a connection from the pool
func (p *ConnectionPool) Get(ctx context.Context) (Connection, error) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return nil, ErrPoolClosed
	}

	atomic.AddInt32(&p.waitCount, 1)
	defer atomic.AddInt32(&p.waitCount, -1)

	// Try to get an existing connection
	select {
	case pc := <-p.connections:
		if pc == nil {
			return nil, ErrInvalidConn
		}

		// Check if connection is still valid
		if pc.isExpired(p.config.MaxLifetime, p.config.MaxIdleTime) || !pc.conn.IsAlive() {
			p.destroyConnection(pc)
			// Try to create a new one
			if err := p.createConnection(ctx); err != nil {
				return nil, err
			}
			return p.Get(ctx)
		}

		pc.markUsed()
		atomic.AddInt32(&p.activeCount, 1)
		return &WrappedConn{pc: pc, pool: p}, nil

	default:
		// No connection available, try to create one
		currentSize := len(p.connections) + int(atomic.LoadInt32(&p.activeCount))
		if currentSize < p.config.MaxSize {
			if err := p.createConnection(ctx); err != nil {
				return nil, err
			}
			return p.Get(ctx)
		}

		// Pool is at max size, wait for a connection
		select {
		case pc := <-p.connections:
			if pc == nil {
				return nil, ErrInvalidConn
			}

			if pc.isExpired(p.config.MaxLifetime, p.config.MaxIdleTime) || !pc.conn.IsAlive() {
				p.destroyConnection(pc)
				return nil, ErrPoolExhausted
			}

			pc.markUsed()
			atomic.AddInt32(&p.activeCount, 1)
			return &WrappedConn{pc: pc, pool: p}, nil

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Put returns a connection to the pool
func (p *ConnectionPool) Put(conn Connection) error {
	// Unwrap the connection first to check if it's already closed
	wc, ok := conn.(*WrappedConn)
	if !ok {
		return ErrInvalidConn
	}

	// If pool is closed, close the underlying connection directly
	if atomic.LoadInt32(&p.closed) == 1 {
		// Don't call wc.Close() to avoid infinite recursion
		if wc.pc != nil && wc.pc.conn != nil {
			_ = wc.pc.conn.Close() // Ignore close error
		}
		return ErrPoolClosed
	}

	atomic.AddInt32(&p.activeCount, -1)

	// Reset the connection
	if err := wc.pc.conn.Reset(); err != nil {
		p.destroyConnection(wc.pc)
		return nil
	}

	// Check if we should keep this connection
	currentSize := len(p.connections) + int(atomic.LoadInt32(&p.activeCount))
	if currentSize > p.config.MaxSize {
		p.destroyConnection(wc.pc)
		return nil
	}

	// Return to pool
	select {
	case p.connections <- wc.pc:
		return nil
	default:
		// Pool is full, destroy the connection
		p.destroyConnection(wc.pc)
		return nil
	}
}

// Close closes the pool and all connections
func (p *ConnectionPool) Close() error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil // Already closed
	}

	// Stop health check and wait for it to complete
	if p.healthTicker != nil {
		p.healthTicker.Stop()
		close(p.healthDone)
		// Wait for health check goroutine to exit
		p.healthWg.Wait()
	}

	// Close all connections
	close(p.connections)
	var lastErr error
	for pc := range p.connections {
		if err := pc.conn.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Stats returns pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PoolStats{
		MaxSize:        p.config.MaxSize,
		CurrentSize:    len(p.connections) + int(atomic.LoadInt32(&p.activeCount)),
		IdleCount:      len(p.connections),
		ActiveCount:    int(atomic.LoadInt32(&p.activeCount)),
		WaitCount:      int(atomic.LoadInt32(&p.waitCount)),
		TotalCreated:   atomic.LoadInt64(&p.totalCreated),
		TotalDestroyed: atomic.LoadInt64(&p.totalDestroyed),
		TotalErrors:    atomic.LoadInt64(&p.totalErrors),
	}
}

// createConnection creates a new connection and adds it to the pool
func (p *ConnectionPool) createConnection(ctx context.Context) error {
	conn, err := p.factory(ctx)
	if err != nil {
		atomic.AddInt64(&p.totalErrors, 1)
		return fmt.Errorf("failed to create connection: %w", err)
	}

	pc := &pooledConn{
		conn:       conn,
		createdAt:  time.Now(),
		lastUsedAt: time.Now(),
	}

	select {
	case p.connections <- pc:
		atomic.AddInt64(&p.totalCreated, 1)
		return nil
	default:
		// Pool is full, destroy the connection
		_ = conn.Close()
		return ErrPoolExhausted
	}
}

// destroyConnection destroys a connection
func (p *ConnectionPool) destroyConnection(pc *pooledConn) {
	if pc == nil || pc.conn == nil {
		return
	}

	if err := pc.conn.Close(); err != nil {
		atomic.AddInt64(&p.totalErrors, 1)
	}
	atomic.AddInt64(&p.totalDestroyed, 1)
}

// healthCheckLoop performs periodic health checks
func (p *ConnectionPool) healthCheckLoop() {
	defer p.healthWg.Done()
	for {
		select {
		case <-p.healthTicker.C:
			p.performHealthCheck()
		case <-p.healthDone:
			return
		}
	}
}

// performHealthCheck checks and maintains pool health
func (p *ConnectionPool) performHealthCheck() {
	// Remove expired connections
	var toCheck []*pooledConn
	for {
		select {
		case pc := <-p.connections:
			if pc == nil {
				continue
			}
			toCheck = append(toCheck, pc)
		default:
			goto checkConnections
		}
	}

checkConnections:
	for _, pc := range toCheck {
		// Check if connection should be destroyed
		shouldDestroy := pc.isExpired(p.config.MaxLifetime, p.config.MaxIdleTime) ||
			!pc.conn.IsAlive() ||
			atomic.LoadInt32(&p.closed) == 1

		if shouldDestroy {
			p.destroyConnection(pc)
			continue
		}

		// Try to put back healthy connection
		select {
		case p.connections <- pc:
			// Successfully returned to pool
		default:
			// Pool is full, destroy connection
			p.destroyConnection(pc)
		}
	}

	// Ensure minimum connections (only if pool is not closed)
	if atomic.LoadInt32(&p.closed) == 0 {
		currentSize := len(p.connections) + int(atomic.LoadInt32(&p.activeCount))
		for currentSize < p.config.MinSize {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := p.createConnection(ctx); err != nil {
				cancel()
				break
			}
			cancel()
			currentSize++
		}
	}
}

// WrappedConn wraps a pooled connection for safe return to pool
type WrappedConn struct {
	pc     *pooledConn
	pool   *ConnectionPool
	mu     sync.Mutex
	closed bool
}

func (wc *WrappedConn) IsAlive() bool {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	if wc.closed {
		return false
	}
	return wc.pc.conn.IsAlive()
}

func (wc *WrappedConn) Close() error {
	wc.mu.Lock()
	if wc.closed {
		wc.mu.Unlock()
		return nil
	}
	wc.closed = true
	wc.mu.Unlock()

	// Call Put without holding the lock to avoid deadlock
	return wc.pool.Put(wc)
}

func (wc *WrappedConn) Reset() error {
	return wc.pc.conn.Reset()
}

// Unwrap returns the underlying connection
func (wc *WrappedConn) Unwrap() Connection {
	return wc.pc.conn
}

// PoolStats contains pool statistics
type PoolStats struct {
	MaxSize        int
	CurrentSize    int
	IdleCount      int
	ActiveCount    int
	WaitCount      int
	TotalCreated   int64
	TotalDestroyed int64
	TotalErrors    int64
}
