// Package embeddings provides caching and rate limiting for embeddings operations
package embeddings

import (
	"container/list"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// EmbeddingCache provides LRU caching for embeddings with TTL support
type EmbeddingCache struct {
	mu        sync.RWMutex
	cache     map[string]*cacheEntry
	lruList   *list.List
	maxSize   int
	ttl       time.Duration
	hits      int64
	misses    int64
	evictions int64
}

type cacheEntry struct {
	key        string
	value      []float64
	element    *list.Element
	createdAt  time.Time
	accessedAt time.Time
}

// NewEmbeddingCache creates a new LRU cache with TTL
func NewEmbeddingCache(maxSize int, ttl time.Duration) *EmbeddingCache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	return &EmbeddingCache{
		cache:   make(map[string]*cacheEntry),
		lruList: list.New(),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves embeddings from cache
func (c *EmbeddingCache) Get(text string) ([]float64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.hashKey(text)
	entry, exists := c.cache[key]
	if !exists {
		c.misses++
		return nil, false
	}

	// Check if entry has expired
	if time.Since(entry.createdAt) > c.ttl {
		c.removeEntry(entry)
		c.misses++
		return nil, false
	}

	// Move to front (most recently used)
	c.lruList.MoveToFront(entry.element)
	entry.accessedAt = time.Now()
	c.hits++

	// Return a copy to prevent modification
	result := make([]float64, len(entry.value))
	copy(result, entry.value)
	return result, true
}

// Set stores embeddings in cache
func (c *EmbeddingCache) Set(text string, embeddings []float64) {
	if len(embeddings) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.hashKey(text)
	now := time.Now()

	// Check if entry already exists
	if entry, exists := c.cache[key]; exists {
		// Update existing entry
		entry.value = make([]float64, len(embeddings))
		copy(entry.value, embeddings)
		entry.createdAt = now
		entry.accessedAt = now
		c.lruList.MoveToFront(entry.element)
		return
	}

	// Create new entry
	entry := &cacheEntry{
		key:        key,
		value:      make([]float64, len(embeddings)),
		createdAt:  now,
		accessedAt: now,
	}
	copy(entry.value, embeddings)

	// Add to front of LRU list
	entry.element = c.lruList.PushFront(entry)
	c.cache[key] = entry

	// Evict oldest entries if cache is full
	for c.lruList.Len() > c.maxSize {
		oldest := c.lruList.Back()
		if oldest != nil {
			c.removeEntry(oldest.Value.(*cacheEntry))
			c.evictions++
		}
	}
}

// Clear removes all entries from cache
func (c *EmbeddingCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*cacheEntry)
	c.lruList = list.New()
}

// Stats returns cache statistics
func (c *EmbeddingCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalRequests := c.hits + c.misses
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(c.hits) / float64(totalRequests)
	}

	return CacheStats{
		Size:      c.lruList.Len(),
		MaxSize:   c.maxSize,
		Hits:      c.hits,
		Misses:    c.misses,
		Evictions: c.evictions,
		HitRate:   hitRate,
		TTL:       c.ttl,
	}
}

// CleanExpired removes expired entries from cache
func (c *EmbeddingCache) CleanExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	cleaned := 0
	current := c.lruList.Back()

	for current != nil {
		entry := current.Value.(*cacheEntry)
		if time.Since(entry.createdAt) > c.ttl {
			next := current.Prev()
			c.removeEntry(entry)
			cleaned++
			current = next
		} else {
			// Since we're going from oldest to newest,
			// if this entry isn't expired, no newer ones will be
			break
		}
	}

	return cleaned
}

// Private methods

func (c *EmbeddingCache) hashKey(text string) string {
	hash := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", hash)
}

func (c *EmbeddingCache) removeEntry(entry *cacheEntry) {
	delete(c.cache, entry.key)
	c.lruList.Remove(entry.element)
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	Size      int           `json:"size"`
	MaxSize   int           `json:"max_size"`
	Hits      int64         `json:"hits"`
	Misses    int64         `json:"misses"`
	Evictions int64         `json:"evictions"`
	HitRate   float64       `json:"hit_rate"`
	TTL       time.Duration `json:"ttl"`
}
