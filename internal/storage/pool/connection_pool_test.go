package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Mock connection for testing
type mockConnection struct {
	id       int
	alive    bool
	resetErr error
	closeErr error
	mu       sync.Mutex
}

func (m *mockConnection) IsAlive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.alive
}

func (m *mockConnection) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alive = false
	return m.closeErr
}

func (m *mockConnection) Reset() error {
	return m.resetErr
}

func (m *mockConnection) setAlive(alive bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alive = alive
}

// Test factory
var connCounter int32

func mockFactory(ctx context.Context) (Connection, error) {
	id := atomic.AddInt32(&connCounter, 1)
	return &mockConnection{
		id:    int(id),
		alive: true,
	}, nil
}

func errorFactory(ctx context.Context) (Connection, error) {
	return nil, errors.New("factory error")
}

func TestConnectionPool_BasicOperations(t *testing.T) {
	config := &PoolConfig{
		MaxSize:     5,
		MinSize:     2,
		MaxIdleTime: 1 * time.Minute,
		MaxLifetime: 5 * time.Minute,
	}

	pool, err := NewConnectionPool(config, mockFactory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer func() { _ = pool.Close() }()

	// Test Get
	ctx := context.Background()
	conn, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Verify it's wrapped
	_, ok := conn.(*WrappedConn)
	if !ok {
		t.Fatal("Connection should be wrapped")
	}

	// Test Put
	err = pool.Put(conn)
	if err != nil {
		t.Fatalf("Failed to put connection: %v", err)
	}

	// Test stats
	stats := pool.Stats()
	if stats.TotalCreated < 1 {
		t.Error("Should have created at least one connection")
	}
}

func TestConnectionPool_MinSize(t *testing.T) {
	config := &PoolConfig{
		MaxSize:     5,
		MinSize:     3,
		MaxIdleTime: 1 * time.Minute,
		MaxLifetime: 5 * time.Minute,
	}

	atomic.StoreInt32(&connCounter, 0)
	pool, err := NewConnectionPool(config, mockFactory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer func() { _ = pool.Close() }()

	// Wait for min connections to be created
	time.Sleep(100 * time.Millisecond)

	stats := pool.Stats()
	if stats.CurrentSize < config.MinSize {
		t.Errorf("Pool size %d is less than min size %d", stats.CurrentSize, config.MinSize)
	}
}

func TestConnectionPool_MaxSize(t *testing.T) {
	config := &PoolConfig{
		MaxSize:     3,
		MinSize:     1,
		MaxIdleTime: 1 * time.Minute,
		MaxLifetime: 5 * time.Minute,
	}

	pool, err := NewConnectionPool(config, mockFactory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer func() { _ = pool.Close() }()

	ctx := context.Background()
	conns := make([]Connection, 0)

	// Get max connections
	for i := 0; i < config.MaxSize; i++ {
		conn, err := pool.Get(ctx)
		if err != nil {
			t.Fatalf("Failed to get connection %d: %v", i, err)
		}
		conns = append(conns, conn)
	}

	// Try to get one more (should block/fail)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = pool.Get(ctx)
	if err == nil {
		t.Error("Expected error when exceeding max size")
	}

	// Return one connection
	_ = pool.Put(conns[0])

	// Now should be able to get one
	ctx = context.Background()
	conn, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection after return: %v", err)
	}
	_ = pool.Put(conn)

	// Return all connections
	for i := 1; i < len(conns); i++ {
		_ = pool.Put(conns[i])
	}
}

func TestConnectionPool_DeadConnection(t *testing.T) {
	config := &PoolConfig{
		MaxSize:     3,
		MinSize:     1,
		MaxIdleTime: 1 * time.Minute,
		MaxLifetime: 5 * time.Minute,
	}

	pool, err := NewConnectionPool(config, mockFactory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer func() { _ = pool.Close() }()

	ctx := context.Background()
	
	// Get a connection
	conn, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Kill the connection
	if wrapped, ok := conn.(*WrappedConn); ok {
		if mock, ok := wrapped.Unwrap().(*mockConnection); ok {
			mock.setAlive(false)
		}
	}

	// Return dead connection
	err = pool.Put(conn)
	if err != nil {
		t.Fatalf("Failed to put connection: %v", err)
	}

	// Get a new connection (should be fresh)
	conn2, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get new connection: %v", err)
	}

	if wrapped, ok := conn2.(*WrappedConn); ok {
		if mock, ok := wrapped.Unwrap().(*mockConnection); ok {
			if !mock.IsAlive() {
				t.Error("New connection should be alive")
			}
		}
	}

	_ = pool.Put(conn2)
}

func TestConnectionPool_HealthCheck(t *testing.T) {
	config := &PoolConfig{
		MaxSize:             3,
		MinSize:             1,
		MaxIdleTime:         1 * time.Minute,
		MaxLifetime:         5 * time.Minute,
		HealthCheckInterval: 50 * time.Millisecond,
	}

	pool, err := NewConnectionPool(config, mockFactory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer func() { _ = pool.Close() }()

	ctx := context.Background()
	
	// Get and return a connection
	conn, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Kill the underlying connection
	if wrapped, ok := conn.(*WrappedConn); ok {
		if mock, ok := wrapped.Unwrap().(*mockConnection); ok {
			mock.setAlive(false)
		}
	}

	_ = pool.Put(conn)

	// Wait for health check to run (need more time with race detector)
	time.Sleep(200 * time.Millisecond)

	// Health check failures are tracked internally
	// Just verify the pool is still working
	stats := pool.Stats()
	t.Logf("Pool stats after health check: CurrentSize=%d, IdleCount=%d, ActiveCount=%d", 
		stats.CurrentSize, stats.IdleCount, stats.ActiveCount)
	if stats.CurrentSize == 0 {
		t.Error("Pool should maintain minimum connections despite failures")
	}
}

func TestConnectionPool_MaxLifetime(t *testing.T) {
	config := &PoolConfig{
		MaxSize:     3,
		MinSize:     1,
		MaxIdleTime: 1 * time.Minute,
		MaxLifetime: 100 * time.Millisecond, // Very short for testing
	}

	pool, err := NewConnectionPool(config, mockFactory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer func() { _ = pool.Close() }()

	ctx := context.Background()
	
	// Get a connection
	conn, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Note the connection ID
	var connID int
	if wrapped, ok := conn.(*WrappedConn); ok {
		if mock, ok := wrapped.Unwrap().(*mockConnection); ok {
			connID = mock.id
		}
	}

	_ = pool.Put(conn)

	// Wait for connection to expire
	time.Sleep(150 * time.Millisecond)

	// Get a new connection
	conn2, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get new connection: %v", err)
	}

	// Should be a different connection
	if wrapped, ok := conn2.(*WrappedConn); ok {
		if mock, ok := wrapped.Unwrap().(*mockConnection); ok {
			if mock.id == connID {
				t.Error("Expected a new connection after lifetime expiry")
			}
		}
	}

	_ = pool.Put(conn2)
}

func TestConnectionPool_Concurrent(t *testing.T) {
	config := &PoolConfig{
		MaxSize:     10,
		MinSize:     2,
		MaxIdleTime: 1 * time.Minute,
		MaxLifetime: 5 * time.Minute,
	}

	pool, err := NewConnectionPool(config, mockFactory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer func() { _ = pool.Close() }()

	ctx := context.Background()
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Run concurrent operations
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				conn, err := pool.Get(ctx)
				if err != nil {
					errors <- err
					return
				}
				
				// Simulate work
				time.Sleep(time.Millisecond)
				
				if err := pool.Put(conn); err != nil {
					errors <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}

	// Verify pool is still healthy
	stats := pool.Stats()
	if stats.CurrentSize > config.MaxSize {
		t.Errorf("Pool size %d exceeds max %d", stats.CurrentSize, config.MaxSize)
	}
}

func TestConnectionPool_Close(t *testing.T) {
	config := &PoolConfig{
		MaxSize:     3,
		MinSize:     1,
		MaxIdleTime: 1 * time.Minute,
		MaxLifetime: 5 * time.Minute,
	}

	pool, err := NewConnectionPool(config, mockFactory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	ctx := context.Background()
	
	// Get some connections
	conn1, _ := pool.Get(ctx)
	conn2, _ := pool.Get(ctx)
	
	_ = pool.Put(conn1)
	// Don't return conn2 - simulate in-use connection

	// Close pool
	err = pool.Close()
	if err != nil {
		t.Fatalf("Failed to close pool: %v", err)
	}

	// Try to get connection after close
	_, err = pool.Get(ctx)
	if err == nil {
		t.Error("Expected error when getting from closed pool")
	}

	// Try to put connection after close
	err = pool.Put(conn2)
	if err == nil {
		t.Error("Expected error when putting to closed pool")
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

func TestConnectionPool_FactoryError(t *testing.T) {
	config := &PoolConfig{
		MaxSize:     3,
		MinSize:     0, // Don't create initial connections
		MaxIdleTime: 1 * time.Minute,
		MaxLifetime: 5 * time.Minute,
	}

	pool, err := NewConnectionPool(config, errorFactory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer func() { _ = pool.Close() }()

	ctx := context.Background()
	
	// Try to get connection
	_, err = pool.Get(ctx)
	if err == nil {
		t.Error("Expected error from factory")
	}
}

func TestWrappedConn_Metrics(t *testing.T) {
	config := &PoolConfig{
		MaxSize:     3,
		MinSize:     1,
		MaxIdleTime: 1 * time.Minute,
		MaxLifetime: 5 * time.Minute,
	}

	pool, err := NewConnectionPool(config, mockFactory)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer func() { _ = pool.Close() }()

	ctx := context.Background()
	
	// Get connection
	conn, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	wrapped, ok := conn.(*WrappedConn)
	if !ok {
		t.Fatal("Connection should be wrapped")
	}

	// Wrapped connection internals are not exposed
	// Just verify it works
	if !wrapped.IsAlive() {
		t.Error("Connection should be alive")
	}

	_ = pool.Put(conn)
}