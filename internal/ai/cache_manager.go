// Package ai provides cache management with automatic cleanup
package ai

import (
	"context"
	"sync"
	"time"

	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/logging"
)

// CacheManager manages cache lifecycle with automatic cleanup
type CacheManager struct {
	cache  *Cache
	config *config.CacheClientConfig
	logger logging.Logger
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewCacheManager creates a new cache manager with automatic cleanup
func NewCacheManager(cfg *config.Config, logger logging.Logger) (*CacheManager, error) {
	cache, err := NewCache(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &CacheManager{
		cache:  cache,
		config: &cfg.AI.Cache,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}

	// Start cleanup goroutine if enabled
	if cfg.AI.Cache.Enabled && cfg.AI.Cache.CleanupInterval > 0 {
		manager.startCleanupRoutine()
	}

	return manager, nil
}

// startCleanupRoutine starts the background cleanup process
func (m *CacheManager) startCleanupRoutine() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ticker := time.NewTicker(m.config.CleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.performCleanup()
			}
		}
	}()
}

// performCleanup removes expired entries and logs metrics
func (m *CacheManager) performCleanup() {
	start := time.Now()
	expired := m.cache.CleanupExpired()

	if expired > 0 {
		m.logger.Info("Cache cleanup completed",
			"expired_entries", expired,
			"duration", time.Since(start),
			"cache_size", m.cache.Size(),
		)
	}

	// Log cache statistics periodically
	stats := m.cache.GetStats()
	m.logger.Debug("Cache statistics",
		"hits", stats.Hits,
		"misses", stats.Misses,
		"hit_rate", stats.HitRate,
		"sets", stats.Sets,
		"evictions", stats.Evictions,
		"size", m.cache.Size(),
	)
}

// Get retrieves a cached response
func (m *CacheManager) Get(req *Request) (*Response, bool) {
	return m.cache.Get(req)
}

// Set stores a response in the cache
func (m *CacheManager) Set(req *Request, resp *Response) {
	m.cache.Set(req, resp)
}

// GetWithContext retrieves a cached response with context
func (m *CacheManager) GetWithContext(ctx context.Context, req *Request) (*Response, bool) {
	return m.cache.GetWithContext(ctx, req)
}

// SetWithContext stores a response with context
func (m *CacheManager) SetWithContext(ctx context.Context, req *Request, resp *Response) {
	m.cache.SetWithContext(ctx, req, resp)
}

// GetStats returns cache statistics
func (m *CacheManager) GetStats() CacheStats {
	return m.cache.GetStats()
}

// Clear removes all entries from the cache
func (m *CacheManager) Clear() {
	m.cache.Clear()
}

// Close stops the cleanup routine and closes the cache
func (m *CacheManager) Close() error {
	// Stop cleanup routine
	m.cancel()
	m.wg.Wait()

	// Close the cache
	return m.cache.Close()
}

// Size returns the current number of cached entries
func (m *CacheManager) Size() int {
	return m.cache.Size()
}

// SetTTL updates the cache TTL
func (m *CacheManager) SetTTL(ttl time.Duration) {
	m.cache.SetTTL(ttl)
}

// SetMaxSize updates the maximum cache size
func (m *CacheManager) SetMaxSize(size int) {
	m.cache.SetMaxSize(size)
}
