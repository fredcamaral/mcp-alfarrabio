package ai

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory/internal/config"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const userIDKey contextKey = "user_id"

func TestCache_BasicOperations(t *testing.T) {
	cfg := &config.Config{}
	cache, err := NewCache(cfg)
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Create test request
	req := &Request{
		ID:    "test-1",
		Model: string(ModelClaude),
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	// Test cache miss
	resp, found := cache.Get(req)
	assert.False(t, found)
	assert.Nil(t, resp)

	// Store response
	testResp := &Response{
		ID:      "resp-1",
		Model:   string(ModelClaude),
		Content: "Hello response",
		TokensUsed: &TokenUsage{
			PromptTokens:     5,
			CompletionTokens: 10,
			TotalTokens:      15,
			Total:            15,
		},
	}
	cache.Set(req, testResp)

	// Test cache hit
	resp, found = cache.Get(req)
	assert.True(t, found)
	assert.NotNil(t, resp)
	assert.Equal(t, testResp.Content, resp.Content)
	assert.Equal(t, testResp.TokensUsed, resp.TokensUsed)
}

func TestCache_KeyGeneration(t *testing.T) {
	cfg := &config.Config{}
	cache, err := NewCache(cfg)
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Same content should generate same key
	req1 := &Request{
		Model: string(ModelClaude),
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}
	req2 := &Request{
		Model: string(ModelClaude),
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}

	key1 := cache.generateKey(req1)
	key2 := cache.generateKey(req2)
	assert.Equal(t, key1, key2)

	// Different content should generate different keys
	req3 := &Request{
		Model: string(ModelClaude),
		Messages: []Message{
			{Role: "user", Content: "Goodbye"},
		},
	}
	key3 := cache.generateKey(req3)
	assert.NotEqual(t, key1, key3)

	// Different model should generate different keys
	req4 := &Request{
		Model: string(ModelOpenAI),
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	}
	key4 := cache.generateKey(req4)
	assert.NotEqual(t, key1, key4)
}

func TestCache_TTL(t *testing.T) {
	cfg := &config.Config{}
	cache, err := NewCache(cfg)
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Set short TTL for testing
	cache.SetTTL(100 * time.Millisecond)

	req := &Request{
		Model: string(ModelClaude),
		Messages: []Message{
			{Role: "user", Content: "Test TTL"},
		},
	}
	resp := &Response{
		Content: "TTL response",
	}

	// Store and verify
	cache.Set(req, resp)
	found, _ := cache.Get(req)
	assert.NotNil(t, found)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	found, exists := cache.Get(req)
	assert.False(t, exists)
	assert.Nil(t, found)
}

func TestCache_SizeLimit(t *testing.T) {
	cfg := &config.Config{}
	cache, err := NewCache(cfg)
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Set small size for testing
	cache.SetMaxSize(3)

	// Add entries
	for i := 0; i < 5; i++ {
		req := &Request{
			Model: string(ModelClaude),
			Messages: []Message{
				{Role: "user", Content: string(rune('A' + i))},
			},
		}
		resp := &Response{
			Content: "Response " + string(rune('A'+i)),
		}
		cache.Set(req, resp)
	}

	// Cache should only have 3 entries
	assert.Equal(t, 3, cache.Size())

	// Oldest entries should be evicted
	req1 := &Request{
		Model: string(ModelClaude),
		Messages: []Message{
			{Role: "user", Content: "A"},
		},
	}
	_, found := cache.Get(req1)
	assert.False(t, found) // Should be evicted
}

func TestCache_Concurrency(t *testing.T) {
	cfg := &config.Config{}
	cache, err := NewCache(cfg)
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Run concurrent operations
	for i := 0; i < numGoroutines; i++ {
		go func(_ int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Mix of get and set operations
				req := &Request{
					Model: string(ModelClaude),
					Messages: []Message{
						{Role: "user", Content: string(rune('A' + (j % 26)))},
					},
				}

				if j%2 == 0 {
					resp := &Response{
						Content: "Response",
					}
					cache.Set(req, resp)
				} else {
					cache.Get(req)
				}
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race conditions occur
}

func TestCache_Statistics(t *testing.T) {
	cfg := &config.Config{}
	cache, err := NewCache(cfg)
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	req := &Request{
		Model: string(ModelClaude),
		Messages: []Message{
			{Role: "user", Content: "Stats test"},
		},
	}
	resp := &Response{
		Content: "Stats response",
	}

	// Initial stats
	stats := cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)

	// Miss
	cache.Get(req)
	stats = cache.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)

	// Store
	cache.Set(req, resp)

	// Hit
	cache.Get(req)
	stats = cache.GetStats()
	assert.Equal(t, int64(1), stats.Hits)
	assert.Equal(t, int64(1), stats.Misses)
	assert.Equal(t, 0.5, stats.HitRate)
}

func TestCache_Clear(t *testing.T) {
	cfg := &config.Config{}
	cache, err := NewCache(cfg)
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Add some entries
	for i := 0; i < 5; i++ {
		req := &Request{
			Model: string(ModelClaude),
			Messages: []Message{
				{Role: "user", Content: string(rune('A' + i))},
			},
		}
		resp := &Response{
			Content: "Response",
		}
		cache.Set(req, resp)
	}

	assert.Equal(t, 5, cache.Size())

	// Clear cache
	cache.Clear()
	assert.Equal(t, 0, cache.Size())

	// Verify entries are gone
	req := &Request{
		Model: string(ModelClaude),
		Messages: []Message{
			{Role: "user", Content: "A"},
		},
	}
	_, found := cache.Get(req)
	assert.False(t, found)
}

func TestCache_WithContext(t *testing.T) {
	cfg := &config.Config{}
	cache, err := NewCache(cfg)
	require.NoError(t, err)
	defer func() { _ = cache.Close() }()

	// Test basic context-aware caching
	ctx := context.WithValue(context.Background(), userIDKey, "user123")

	req := &Request{
		Model: string(ModelClaude),
		Messages: []Message{
			{Role: "user", Content: "Context test"},
		},
	}

	resp := &Response{
		Content: "Context response",
	}

	cache.SetWithContext(ctx, req, resp)

	// Should find with same context
	found, exists := cache.GetWithContext(ctx, req)
	assert.True(t, exists)
	assert.Equal(t, resp.Content, found.Content)

	// Test with different context values to ensure cache differentiation
	ctx2 := context.WithValue(context.Background(), userIDKey, "user456")
	resp2 := &Response{
		Content: "Different user response",
	}
	cache.SetWithContext(ctx2, req, resp2)

	// Should find different responses for different contexts
	found2, exists2 := cache.GetWithContext(ctx2, req)
	assert.True(t, exists2)
	assert.Equal(t, resp2.Content, found2.Content)
	assert.NotEqual(t, found.Content, found2.Content)
}
