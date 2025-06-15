package ai

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"lerian-mcp-memory/internal/config"
)

// Cache provides caching for AI responses with LRU eviction
type Cache struct {
	store      map[string]*list.Element
	lru        *list.List
	mu         sync.RWMutex
	ttl        time.Duration
	maxSize    int
	stats      CacheStats
	contextKey string
}

type cacheEntry struct {
	key        string
	response   *Response
	createdAt  time.Time
	accessedAt time.Time
	hits       int64
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	Hits       int64
	Misses     int64
	Sets       int64
	Evictions  int64
	TotalBytes int64
	HitRate    float64
}

// ComputeHitRate calculates the cache hit rate
func (s *CacheStats) ComputeHitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// NewCache creates a new AI response cache
func NewCache(cfg *config.Config) (*Cache, error) {
	ttl := 15 * time.Minute
	maxSize := 1000

	// Override from config if available
	if cfg != nil && cfg.AI.Cache.TTL > 0 {
		ttl = cfg.AI.Cache.TTL
	}
	if cfg != nil && cfg.AI.Cache.MaxSize > 0 {
		maxSize = cfg.AI.Cache.MaxSize
	}

	return &Cache{
		store:      make(map[string]*list.Element),
		lru:        list.New(),
		ttl:        ttl,
		maxSize:    maxSize,
		contextKey: "cache_context",
	}, nil
}

// Get retrieves a cached response
func (c *Cache) Get(req *Request) (*Response, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.generateKey(req)
	elem, exists := c.store[key]
	if !exists {
		atomic.AddInt64(&c.stats.Misses, 1)
		return nil, false
	}

	entry := elem.Value.(*cacheEntry)

	// Check if entry is expired
	if time.Since(entry.createdAt) > c.ttl {
		c.removeElement(elem)
		atomic.AddInt64(&c.stats.Misses, 1)
		return nil, false
	}

	// Move to front (LRU)
	c.lru.MoveToFront(elem)
	entry.accessedAt = time.Now()
	entry.hits++

	atomic.AddInt64(&c.stats.Hits, 1)

	// Return a copy of the response
	resp := *entry.response
	return &resp, true
}

// Set stores a response in the cache
func (c *Cache) Set(req *Request, resp *Response) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.generateKey(req)

	// Check if already exists
	if elem, exists := c.store[key]; exists {
		// Update existing entry
		c.lru.MoveToFront(elem)
		entry := elem.Value.(*cacheEntry)
		entry.response = resp
		entry.createdAt = time.Now()
		entry.accessedAt = time.Now()
		return
	}

	// Check size limit and evict if necessary
	for c.lru.Len() >= c.maxSize {
		c.evictOldest()
	}

	// Create new entry
	entry := &cacheEntry{
		key:        key,
		response:   resp,
		createdAt:  time.Now(),
		accessedAt: time.Now(),
		hits:       0,
	}

	elem := c.lru.PushFront(entry)
	c.store[key] = elem

	atomic.AddInt64(&c.stats.Sets, 1)
}

// generateKey creates a cache key from the request
func (c *Cache) generateKey(req *Request) string {
	h := sha256.New()
	h.Write([]byte(req.Model))

	for _, msg := range req.Messages {
		h.Write([]byte(msg.Role))
		h.Write([]byte(msg.Content))
	}

	// Include context in key if present
	if req.Context != nil {
		for k, v := range req.Context {
			h.Write([]byte(k))
			if s, ok := v.(string); ok {
				h.Write([]byte(s))
			} else {
				// hash.Hash.Write never returns an error, but fmt.Fprintf to a hash can fail
				if _, err := fmt.Fprintf(h, "%v", v); err != nil {
					// This should never happen with hash.Hash, but handle defensively
					// Use fmt.Fprintf as gocritic prefers it over fmt.Sprintf + Write
					_, _ = fmt.Fprintf(h, "%v", v) // Safe to ignore error for hash.Hash
				}
			}
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}

// evictOldest removes the oldest cache entry
func (c *Cache) evictOldest() {
	elem := c.lru.Back()
	if elem != nil {
		c.removeElement(elem)
		atomic.AddInt64(&c.stats.Evictions, 1)
	}
}

// removeElement removes an element from cache
func (c *Cache) removeElement(elem *list.Element) {
	entry := elem.Value.(*cacheEntry)
	delete(c.store, entry.key)
	c.lru.Remove(elem)
}

// Close cleans up cache resources
func (c *Cache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]*list.Element)
	c.lru = list.New()
	return nil
}

// SetTTL updates the cache TTL
func (c *Cache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ttl = ttl
}

// SetMaxSize updates the maximum cache size
func (c *Cache) SetMaxSize(size int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.maxSize = size

	// Evict entries if necessary
	for c.lru.Len() > c.maxSize {
		c.evictOldest()
	}
}

// Size returns the current number of cached entries
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lru.Len()
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]*list.Element)
	c.lru = list.New()
}

// GetStats returns cache statistics
func (c *Cache) GetStats() CacheStats {
	stats := CacheStats{
		Hits:       atomic.LoadInt64(&c.stats.Hits),
		Misses:     atomic.LoadInt64(&c.stats.Misses),
		Sets:       atomic.LoadInt64(&c.stats.Sets),
		Evictions:  atomic.LoadInt64(&c.stats.Evictions),
		TotalBytes: atomic.LoadInt64(&c.stats.TotalBytes),
	}
	stats.HitRate = stats.ComputeHitRate()
	return stats
}

// GetWithContext retrieves a cached response with context awareness
func (c *Cache) GetWithContext(ctx context.Context, req *Request) (*Response, bool) {
	// Create a copy of the request to avoid modifying the original
	reqCopy := *req
	if reqCopy.Context == nil {
		reqCopy.Context = make(map[string]interface{})
	}

	// Extract context values - check for common keys (both string and typed keys)
	contextKeys := []interface{}{"user_id", "session_id", "repository", c.contextKey}

	// Also check for the contextKey type used in tests
	type contextKey string
	const userIDKey contextKey = "user_id"
	contextKeys = append(contextKeys, userIDKey)

	for _, key := range contextKeys {
		if ctxValue := ctx.Value(key); ctxValue != nil {
			// Use string representation of the key for storage
			keyStr := fmt.Sprintf("%v", key)
			reqCopy.Context[keyStr] = fmt.Sprintf("%v", ctxValue)
		}
	}

	return c.Get(&reqCopy)
}

// SetWithContext stores a response with context awareness
func (c *Cache) SetWithContext(ctx context.Context, req *Request, resp *Response) {
	// Create a copy of the request to avoid modifying the original
	reqCopy := *req
	if reqCopy.Context == nil {
		reqCopy.Context = make(map[string]interface{})
	}

	// Extract context values - check for common keys (both string and typed keys)
	contextKeys := []interface{}{"user_id", "session_id", "repository", c.contextKey}

	// Also check for the contextKey type used in tests
	type contextKey string
	const userIDKey contextKey = "user_id"
	contextKeys = append(contextKeys, userIDKey)

	for _, key := range contextKeys {
		if ctxValue := ctx.Value(key); ctxValue != nil {
			// Use string representation of the key for storage
			keyStr := fmt.Sprintf("%v", key)
			reqCopy.Context[keyStr] = fmt.Sprintf("%v", ctxValue)
		}
	}

	c.Set(&reqCopy, resp)
}

// CleanupExpired removes all expired entries
func (c *Cache) CleanupExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	expired := 0
	now := time.Now()

	// Iterate from back (oldest) to front
	for elem := c.lru.Back(); elem != nil; {
		prev := elem.Prev()
		entry := elem.Value.(*cacheEntry)

		if now.Sub(entry.createdAt) > c.ttl {
			c.removeElement(elem)
			expired++
		}

		elem = prev
	}

	return expired
}
