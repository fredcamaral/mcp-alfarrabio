// Package ai provides response caching for AI service optimization.
package ai

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"lerian-mcp-memory/internal/config"
)

// CacheEntry represents a cached AI response
type CacheEntry struct {
	Response  *Response `json:"response"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	AccessCount int     `json:"access_count"`
	LastAccess  time.Time `json:"last_access"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled    bool          `json:"enabled"`
	TTL        time.Duration `json:"ttl"`
	MaxSize    int           `json:"max_size"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// Cache provides in-memory caching for AI responses
type Cache struct {
	entries map[string]*CacheEntry
	mutex   sync.RWMutex
	config  CacheConfig
	stats   CacheStats
	cleanup *time.Ticker
	done    chan bool
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	Evictions   int64     `json:"evictions"`
	TotalSize   int       `json:"total_size"`
	LastCleanup time.Time `json:"last_cleanup"`
}

// NewCache creates a new AI response cache
func NewCache(cfg *config.Config) (*Cache, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	cacheConfig := CacheConfig{
		Enabled:         cfg.AI.Cache.Enabled,
		TTL:             cfg.AI.Cache.TTL,
		MaxSize:         cfg.AI.Cache.MaxSize,
		CleanupInterval: cfg.AI.Cache.CleanupInterval,
	}

	// Set defaults if not configured
	if cacheConfig.TTL == 0 {
		cacheConfig.TTL = 30 * time.Minute
	}
	if cacheConfig.MaxSize == 0 {
		cacheConfig.MaxSize = 1000
	}
	if cacheConfig.CleanupInterval == 0 {
		cacheConfig.CleanupInterval = 5 * time.Minute
	}

	cache := &Cache{
		entries: make(map[string]*CacheEntry),
		config:  cacheConfig,
		done:    make(chan bool),
	}

	// Start cleanup routine if enabled
	if cacheConfig.Enabled {
		cache.startCleanup()
	}

	return cache, nil
}

// generateKey creates a cache key from request
func (c *Cache) generateKey(req *Request) string {
	// Create a consistent hash from request content
	data := struct {
		Model    Model     `json:"model"`
		Messages []Message `json:"messages"`
		Context  map[string]string `json:"context"`
	}{
		Model:    req.Model,
		Messages: req.Messages,
		Context:  req.Context,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

// Get retrieves a cached response
func (c *Cache) Get(req *Request) (*Response, bool) {
	if !c.config.Enabled {
		return nil, false
	}

	key := c.generateKey(req)
	
	c.mutex.RLock()
	entry, exists := c.entries[key]
	c.mutex.RUnlock()

	if !exists {
		c.stats.Misses++
		return nil, false
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		c.mutex.Lock()
		delete(c.entries, key)
		c.mutex.Unlock()
		c.stats.Misses++
		return nil, false
	}

	// Update access information
	c.mutex.Lock()
	entry.AccessCount++
	entry.LastAccess = time.Now()
	c.mutex.Unlock()

	c.stats.Hits++
	
	// Return a copy of the response
	responseCopy := *entry.Response
	return &responseCopy, true
}

// Set stores a response in cache
func (c *Cache) Set(req *Request, response *Response) {
	if !c.config.Enabled || response == nil {
		return
	}

	key := c.generateKey(req)
	now := time.Now()

	entry := &CacheEntry{
		Response:    response,
		CreatedAt:   now,
		ExpiresAt:   now.Add(c.config.TTL),
		AccessCount: 1,
		LastAccess:  now,
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if we need to evict entries to make space
	if len(c.entries) >= c.config.MaxSize {
		c.evictOldest()
	}

	c.entries[key] = entry
	c.stats.TotalSize = len(c.entries)
}

// evictOldest removes the oldest entry (LRU eviction)
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.stats.Evictions++
	}
}

// startCleanup begins the periodic cleanup routine
func (c *Cache) startCleanup() {
	c.cleanup = time.NewTicker(c.config.CleanupInterval)
	
	go func() {
		for {
			select {
			case <-c.cleanup.C:
				c.cleanupExpired()
			case <-c.done:
				return
			}
		}
	}()
}

// cleanupExpired removes expired entries
func (c *Cache) cleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			expired = append(expired, key)
		}
	}

	for _, key := range expired {
		delete(c.entries, key)
		c.stats.Evictions++
	}

	c.stats.TotalSize = len(c.entries)
	c.stats.LastCleanup = now
}

// GetStats returns current cache statistics
func (c *Cache) GetStats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	statsCopy := c.stats
	statsCopy.TotalSize = len(c.entries)
	return statsCopy
}

// GetHitRate calculates cache hit rate
func (c *Cache) GetHitRate() float64 {
	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0.0
	}
	return float64(c.stats.Hits) / float64(total)
}

// Clear removes all cached entries
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.entries = make(map[string]*CacheEntry)
	c.stats.TotalSize = 0
}

// Size returns the current number of cached entries
func (c *Cache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.entries)
}

// Close stops the cache and cleanup routines
func (c *Cache) Close() error {
	if c.cleanup != nil {
		c.cleanup.Stop()
	}
	
	select {
	case c.done <- true:
	default:
	}
	
	c.Clear()
	return nil
}