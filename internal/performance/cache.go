// Package performance provides caching layer for high-performance operations
package performance

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Cache provides a high-performance, thread-safe caching layer
type Cache struct {
	store   map[string]*CacheEntry
	mutex   sync.RWMutex
	config  *CacheConfig
	janitor *janitor
	stats   *CacheStats
	closed  bool
}

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Key         string        `json:"key"`
	Value       interface{}   `json:"value"`
	CreatedAt   time.Time     `json:"created_at"`
	AccessedAt  time.Time     `json:"accessed_at"`
	ExpiresAt   time.Time     `json:"expires_at"`
	AccessCount int64         `json:"access_count"`
	Size        int64         `json:"size"`
	TTL         time.Duration `json:"ttl"`
}

// CacheConfig defines cache configuration
type CacheConfig struct {
	// Size limits
	MaxSize  int64 `json:"max_size"`  // Max cache size in bytes
	MaxItems int   `json:"max_items"` // Max number of items

	// TTL settings
	DefaultTTL time.Duration `json:"default_ttl"` // Default time-to-live
	MaxTTL     time.Duration `json:"max_ttl"`     // Maximum allowed TTL

	// Cleanup settings
	CleanupInterval time.Duration `json:"cleanup_interval"` // How often to clean expired items
	CleanupBatch    int           `json:"cleanup_batch"`    // Max items to clean per batch

	// Performance settings
	PreComputeSize bool `json:"precompute_size"` // Pre-compute item sizes
	TrackStats     bool `json:"track_stats"`     // Track detailed statistics

	// Eviction settings
	EvictionPolicy string `json:"eviction_policy"` // "lru", "lfu", "ttl"
}

// DefaultCacheConfig returns optimized default configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		MaxSize:         100 * 1024 * 1024, // 100MB
		MaxItems:        10000,
		DefaultTTL:      15 * time.Minute,
		MaxTTL:          2 * time.Hour,
		CleanupInterval: 5 * time.Minute,
		CleanupBatch:    100,
		PreComputeSize:  true,
		TrackStats:      true,
		EvictionPolicy:  "lru",
	}
}

// CacheStatsData provides cache performance statistics
type CacheStatsData struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	Sets        int64     `json:"sets"`
	Deletes     int64     `json:"deletes"`
	Evictions   int64     `json:"evictions"`
	Size        int64     `json:"size"`
	Items       int       `json:"items"`
	HitRate     float64   `json:"hit_rate"`
	LastCleanup time.Time `json:"last_cleanup"`
}

// CacheStats holds internal cache statistics with synchronization
type CacheStats struct {
	mutex sync.RWMutex
	data  CacheStatsData
}

// janitor handles background cleanup operations
type janitor struct {
	cache    *Cache
	interval time.Duration
	stop     chan struct{}
}

// NewCache creates a new high-performance cache
func NewCache(config *CacheConfig) *Cache {
	if config == nil {
		config = DefaultCacheConfig()
	}

	cache := &Cache{
		store:  make(map[string]*CacheEntry),
		config: config,
		stats:  &CacheStats{data: CacheStatsData{}},
	}

	// Start janitor for cleanup
	if config.CleanupInterval > 0 {
		cache.janitor = &janitor{
			cache:    cache,
			interval: config.CleanupInterval,
			stop:     make(chan struct{}),
		}
		go cache.janitor.run()
	}

	return cache
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.closed {
		return nil, false
	}

	entry, exists := c.store[key]
	if !exists {
		c.incrementMisses()
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.store, key)
		c.mutex.Unlock()
		c.mutex.RLock()
		c.incrementMisses()
		return nil, false
	}

	// Update access metadata
	entry.AccessedAt = time.Now()
	entry.AccessCount++

	c.incrementHits()
	return entry.Value, true
}

// Set stores an item in the cache with default TTL
func (c *Cache) Set(key string, value interface{}) error {
	return c.SetWithTTL(key, value, c.config.DefaultTTL)
}

// SetWithTTL stores an item in the cache with specific TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	if c.closed {
		return fmt.Errorf("cache is closed")
	}

	// Validate TTL
	if ttl > c.config.MaxTTL {
		ttl = c.config.MaxTTL
	}

	// Calculate size if enabled
	var size int64
	if c.config.PreComputeSize {
		size = c.calculateSize(value)
	}

	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		ExpiresAt:   time.Now().Add(ttl),
		AccessCount: 1,
		Size:        size,
		TTL:         ttl,
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if we need to evict items
	if err := c.ensureCapacity(entry); err != nil {
		return fmt.Errorf("failed to ensure capacity: %w", err)
	}

	// Remove existing entry if it exists
	if existing, exists := c.store[key]; exists {
		c.updateStats(-existing.Size, false)
	}

	// Store the entry
	c.store[key] = entry
	c.updateStats(entry.Size, true)
	c.incrementSets()

	return nil
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return false
	}

	entry, exists := c.store[key]
	if !exists {
		return false
	}

	delete(c.store, key)
	c.updateStats(-entry.Size, false)
	c.incrementDeletes()

	return true
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.store = make(map[string]*CacheEntry)
	c.stats.data.Size = 0
	c.stats.data.Items = 0
}

// Keys returns all cache keys
func (c *Cache) Keys() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	keys := make([]string, 0, len(c.store))
	for key := range c.store {
		keys = append(keys, key)
	}

	return keys
}

// Size returns the number of items in the cache
func (c *Cache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.store)
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStatsData {
	c.stats.mutex.RLock()
	defer c.stats.mutex.RUnlock()

	// Return a copy of the data without the mutex
	stats := c.stats.data

	// Calculate hit rate
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}

	return stats
}

// Close gracefully shuts down the cache
func (c *Cache) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	if c.janitor != nil {
		close(c.janitor.stop)
	}

	c.store = nil

	return nil
}

// Cleanup removes expired items
func (c *Cache) Cleanup() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return 0
	}

	now := time.Now()
	removed := 0
	batch := 0

	for key, entry := range c.store {
		if !now.After(entry.ExpiresAt) {
			continue
		}
		delete(c.store, key)
		c.updateStats(-entry.Size, false)
		removed++
		batch++

		// Limit batch size to avoid blocking
		if batch >= c.config.CleanupBatch {
			break
		}
	}

	c.stats.mutex.Lock()
	c.stats.data.Evictions += int64(removed)
	c.stats.data.LastCleanup = now
	c.stats.mutex.Unlock()

	return removed
}

// GetOrCompute gets a value or computes it if not present
func (c *Cache) GetOrCompute(key string, compute func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	if value, found := c.Get(key); found {
		return value, nil
	}

	// Compute the value
	value, err := compute()
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := c.Set(key, value); err != nil {
		// Log error but don't fail the operation
		// The cache is optional, so we don't want to fail the operation
		// just because we can't cache the result
		log.Printf("failed to cache value for key %s: %v", key, err)
	}

	return value, nil
}

// GetOrComputeWithTTL gets a value or computes it with specific TTL
func (c *Cache) GetOrComputeWithTTL(key string, ttl time.Duration, compute func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	if value, found := c.Get(key); found {
		return value, nil
	}

	// Compute the value
	value, err := compute()
	if err != nil {
		return nil, err
	}

	// Store in cache with specific TTL
	if err := c.SetWithTTL(key, value, ttl); err != nil {
		// Log error but don't fail the operation
		log.Printf("failed to cache value with TTL for key %s: %v", key, err)
	}

	return value, nil
}

// Private methods

func (c *Cache) ensureCapacity(newEntry *CacheEntry) error {
	// Check item count limit
	if len(c.store) >= c.config.MaxItems {
		if err := c.evictItems(1); err != nil {
			return fmt.Errorf("failed to evict items for count limit: %w", err)
		}
	}

	// Check size limit
	if c.config.PreComputeSize && c.stats.data.Size+newEntry.Size > c.config.MaxSize {
		// Calculate how much space we need
		needed := (c.stats.data.Size + newEntry.Size) - c.config.MaxSize
		if err := c.evictSize(needed); err != nil {
			return fmt.Errorf("failed to evict items for size limit: %w", err)
		}
	}

	return nil
}

func (c *Cache) evictItems(count int) error {
	if count <= 0 {
		return nil
	}

	switch c.config.EvictionPolicy {
	case "lru":
		return c.evictLRU(count)
	case "lfu":
		return c.evictLFU(count)
	case "ttl":
		return c.evictTTL(count)
	default:
		return c.evictLRU(count) // Default to LRU
	}
}

func (c *Cache) evictSize(targetSize int64) error {
	var evicted int64
	var count int

	for evicted < targetSize && len(c.store) > 0 {
		if err := c.evictItems(1); err != nil {
			return err
		}
		count++

		// Safety check to avoid infinite loops
		if count > len(c.store) {
			break
		}
	}

	return nil
}

func (c *Cache) evictLRU(count int) error {
	// Find oldest accessed items
	type entryAge struct {
		key        string
		entry      *CacheEntry
		accessTime time.Time
	}

	candidates := make([]entryAge, 0, len(c.store))
	for key, entry := range c.store {
		candidates = append(candidates, entryAge{
			key:        key,
			entry:      entry,
			accessTime: entry.AccessedAt,
		})
	}

	// Sort by access time (oldest first)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].accessTime.After(candidates[j].accessTime) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Evict oldest items
	evicted := 0
	for i := 0; i < len(candidates) && evicted < count; i++ {
		delete(c.store, candidates[i].key)
		c.updateStats(-candidates[i].entry.Size, false)
		evicted++
	}

	c.stats.mutex.Lock()
	c.stats.data.Evictions += int64(evicted)
	c.stats.mutex.Unlock()

	return nil
}

func (c *Cache) evictLFU(count int) error {
	// Find least frequently used items
	type entryFreq struct {
		key   string
		entry *CacheEntry
		freq  int64
	}

	candidates := make([]entryFreq, 0, len(c.store))
	for key, entry := range c.store {
		candidates = append(candidates, entryFreq{
			key:   key,
			entry: entry,
			freq:  entry.AccessCount,
		})
	}

	// Sort by frequency (lowest first)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].freq > candidates[j].freq {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Evict least frequent items
	evicted := 0
	for i := 0; i < len(candidates) && evicted < count; i++ {
		delete(c.store, candidates[i].key)
		c.updateStats(-candidates[i].entry.Size, false)
		evicted++
	}

	c.stats.mutex.Lock()
	c.stats.data.Evictions += int64(evicted)
	c.stats.mutex.Unlock()

	return nil
}

func (c *Cache) evictTTL(count int) error {
	// Find items with shortest remaining TTL
	type entryTTL struct {
		key       string
		entry     *CacheEntry
		remaining time.Duration
	}

	now := time.Now()
	candidates := make([]entryTTL, 0, len(c.store))
	for key, entry := range c.store {
		remaining := entry.ExpiresAt.Sub(now)
		candidates = append(candidates, entryTTL{
			key:       key,
			entry:     entry,
			remaining: remaining,
		})
	}

	// Sort by remaining TTL (shortest first)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].remaining > candidates[j].remaining {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Evict items with shortest TTL
	evicted := 0
	for i := 0; i < len(candidates) && evicted < count; i++ {
		delete(c.store, candidates[i].key)
		c.updateStats(-candidates[i].entry.Size, false)
		evicted++
	}

	c.stats.mutex.Lock()
	c.stats.data.Evictions += int64(evicted)
	c.stats.mutex.Unlock()

	return nil
}

func (c *Cache) calculateSize(value interface{}) int64 {
	// Simple size calculation - in production would be more sophisticated
	if data, err := json.Marshal(value); err == nil {
		return int64(len(data))
	}
	return 100 // Default size estimate
}

func (c *Cache) updateStats(sizeChange int64, itemAdded bool) {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()

	c.stats.data.Size += sizeChange
	if itemAdded {
		c.stats.data.Items++
	} else {
		c.stats.data.Items--
	}
}

func (c *Cache) incrementHits() {
	if !c.config.TrackStats {
		return
	}

	c.stats.mutex.Lock()
	c.stats.data.Hits++
	c.stats.mutex.Unlock()
}

func (c *Cache) incrementMisses() {
	if !c.config.TrackStats {
		return
	}

	c.stats.mutex.Lock()
	c.stats.data.Misses++
	c.stats.mutex.Unlock()
}

func (c *Cache) incrementSets() {
	if !c.config.TrackStats {
		return
	}

	c.stats.mutex.Lock()
	c.stats.data.Sets++
	c.stats.mutex.Unlock()
}

func (c *Cache) incrementDeletes() {
	if !c.config.TrackStats {
		return
	}

	c.stats.mutex.Lock()
	c.stats.data.Deletes++
	c.stats.mutex.Unlock()
}

// GetStats returns a copy of cache statistics
func (c *Cache) GetStats() *CacheStatsData {
	c.stats.mutex.RLock()
	defer c.stats.mutex.RUnlock()

	// Return a copy of the data to avoid race conditions
	statsCopy := c.stats.data
	return &statsCopy
}

// janitor background cleanup routine
func (j *janitor) run() {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			j.cache.Cleanup()
		case <-j.stop:
			return
		}
	}
}
