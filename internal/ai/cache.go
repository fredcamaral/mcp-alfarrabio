package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"lerian-mcp-memory/internal/config"
)

// Cache provides caching for AI responses
type Cache struct {
	store   map[string]*cacheEntry
	mu      sync.RWMutex
	ttl     time.Duration
	maxSize int
}

type cacheEntry struct {
	response  *Response
	createdAt time.Time
}

// NewCache creates a new AI response cache
func NewCache(cfg *config.Config) (*Cache, error) {
	return &Cache{
		store:   make(map[string]*cacheEntry),
		ttl:     15 * time.Minute, // Default TTL
		maxSize: 1000,             // Default max entries
	}, nil
}

// Get retrieves a cached response
func (c *Cache) Get(req *Request) (*Response, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey(req)
	entry, exists := c.store[key]
	if !exists {
		return nil, false
	}

	// Check if entry is expired
	if time.Since(entry.createdAt) > c.ttl {
		return nil, false
	}

	// Return a copy of the response
	resp := *entry.response
	return &resp, true
}

// Set stores a response in the cache
func (c *Cache) Set(req *Request, resp *Response) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check size limit
	if len(c.store) >= c.maxSize {
		// Simple eviction: remove oldest entry
		c.evictOldest()
	}

	key := c.generateKey(req)
	c.store[key] = &cacheEntry{
		response:  resp,
		createdAt: time.Now(),
	}
}

// generateKey creates a cache key from the request
func (c *Cache) generateKey(req *Request) string {
	h := sha256.New()
	h.Write([]byte(req.Model))

	for _, msg := range req.Messages {
		h.Write([]byte(msg.Role))
		h.Write([]byte(msg.Content))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// evictOldest removes the oldest cache entry
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.store {
		if oldestKey == "" || entry.createdAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.createdAt
		}
	}

	if oldestKey != "" {
		delete(c.store, oldestKey)
	}
}

// Close cleans up cache resources
func (c *Cache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]*cacheEntry)
	return nil
}
