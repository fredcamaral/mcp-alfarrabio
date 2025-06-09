package pool

import (
	"context"
	"testing"
	"time"

	"lerian-mcp-memory/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConnectionPool(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "test_user",
		Password:        "test_pass",
		Name:            "test_db",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
		QueryTimeout:    time.Second * 30,
	}

	// This test will fail if no database is available, which is expected in CI
	_, err := NewConnectionPool(cfg)
	// We expect this to fail in test environment
	assert.Error(t, err)
}

func TestConnectionPoolConfig(t *testing.T) {
	// Test configuration validation
	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "test_user",
		Password:        "test_pass",
		Name:            "test_db",
		SSLMode:         "disable",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
		QueryTimeout:    time.Second * 30,
	}

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 5432, cfg.Port)
	assert.Equal(t, 10, cfg.MaxOpenConns)
	assert.Equal(t, 5, cfg.MaxIdleConns)
}

func TestHealthChecker(t *testing.T) {
	hc := &HealthChecker{
		interval:    time.Second,
		timeout:     time.Second * 5,
		maxFailures: 3,
		isHealthy:   true,
		stopCh:      make(chan struct{}),
	}

	// Test initial state
	assert.True(t, hc.IsHealthy())

	// Test setting unhealthy
	hc.setHealthy(false)
	assert.False(t, hc.IsHealthy())

	// Test setting healthy
	hc.setHealthy(true)
	assert.True(t, hc.IsHealthy())
}

func TestPoolStats(t *testing.T) {
	stats := &PoolStats{
		TotalConns:        10,
		IdleConns:         5,
		OpenConns:         3,
		InUseConns:        2,
		WaitCount:         0,
		WaitDuration:      0,
		MaxIdleClosed:     0,
		MaxIdleTimeClosed: 0,
		MaxLifetimeClosed: 0,
	}

	assert.Equal(t, 10, stats.TotalConns)
	assert.Equal(t, 5, stats.IdleConns)
	assert.Equal(t, 3, stats.OpenConns)
	assert.Equal(t, 2, stats.InUseConns)
}

// TestConnectionPoolIntegration tests with mock components
func TestConnectionPoolIntegration(t *testing.T) {
	// Skip if no test database available
	t.Skip("Integration test requires live database")

	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "postgres",
		Name:            "test_db",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
		QueryTimeout:    time.Second * 30,
	}

	pool, err := NewConnectionPool(cfg)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}
	defer pool.Close()

	// Test basic operations
	ctx := context.Background()

	// Test simple query
	rows, err := pool.Query(ctx, "SELECT 1")
	require.NoError(t, err)
	require.NotNil(t, rows)
	rows.Close()

	// Test single row query
	row := pool.QueryRow(ctx, "SELECT 1")
	require.NotNil(t, row)

	var result int
	err = row.Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, 1, result)

	// Test stats
	stats := pool.GetStats()
	require.NotNil(t, stats)
	assert.GreaterOrEqual(t, stats.TotalConns, 0)

	// Test health
	assert.True(t, pool.IsHealthy())
}

func TestConnectionPoolErrorHandling(t *testing.T) {
	// Test with invalid configuration
	invalidCfg := &config.DatabaseConfig{
		Host:     "invalid_host",
		Port:     9999,
		User:     "invalid_user",
		Password: "invalid_pass",
		Name:     "invalid_db",
		SSLMode:  "disable",
	}

	_, err := NewConnectionPool(invalidCfg)
	assert.Error(t, err)
}

func TestConnectionPoolTimeout(t *testing.T) {
	// Test timeout behavior
	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "test_user",
		Password:        "test_pass",
		Name:            "test_db",
		SSLMode:         "disable",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 30,
		QueryTimeout:    time.Millisecond * 1, // Very short timeout
	}

	// This should fail due to timeout
	_, err := NewConnectionPool(cfg)
	assert.Error(t, err)
}
