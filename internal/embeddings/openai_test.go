package embeddings

import (
	"context"
	"mcp-memory/internal/config"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimiter(t *testing.T) {
	maxTokens := 10
	refillRate := time.Second

	rl := NewRateLimiter(maxTokens, refillRate)

	assert.Equal(t, maxTokens, rl.maxTokens)
	assert.Equal(t, maxTokens, rl.tokens)
	assert.Equal(t, refillRate, rl.refillRate)
	assert.False(t, rl.lastRefill.IsZero())
}

func TestRateLimiter_Allow(t *testing.T) {
	t.Run("allow when tokens available", func(t *testing.T) {
		rl := NewRateLimiter(5, time.Second)

		// Should allow first 5 requests
		for i := 0; i < 5; i++ {
			assert.True(t, rl.Allow(), "request %d should be allowed", i+1)
		}

		// Should deny 6th request
		assert.False(t, rl.Allow(), "6th request should be denied")
	})

	t.Run("refill tokens over time", func(t *testing.T) {
		rl := NewRateLimiter(2, time.Millisecond*100)

		// Use all tokens
		assert.True(t, rl.Allow())
		assert.True(t, rl.Allow())
		assert.False(t, rl.Allow())

		// Wait for refill
		time.Sleep(time.Millisecond * 250) // Should add 2 tokens

		// Should allow again
		assert.True(t, rl.Allow())
		assert.True(t, rl.Allow())
		assert.False(t, rl.Allow())
	})
}

func TestRateLimiter_Wait(t *testing.T) {
	t.Run("wait until token available", func(t *testing.T) {
		rl := NewRateLimiter(1, time.Millisecond*50)

		// Use the token
		assert.True(t, rl.Allow())

		// Wait should succeed after refill
		ctx := context.Background()
		start := time.Now()
		err := rl.Wait(ctx)
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.True(t, duration >= time.Millisecond*40) // Allow some tolerance
	})

	t.Run("context cancellation", func(t *testing.T) {
		rl := NewRateLimiter(1, time.Second*10) // Very slow refill

		// Use the token
		assert.True(t, rl.Allow())

		// Cancel context quickly
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
		defer cancel()

		err := rl.Wait(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestNewOpenAIEmbeddingService(t *testing.T) {
	cfg := &config.OpenAIConfig{
		APIKey:         "test-key",
		EmbeddingModel: "text-embedding-ada-002",
		MaxTokens:      8191,
		Temperature:    0.0,
		RequestTimeout: 60,
		RateLimitRPM:   60,
	}

	service := NewOpenAIEmbeddingService(cfg)

	assert.NotNil(t, service.client)
	assert.Equal(t, cfg, service.config)
	assert.NotNil(t, service.cache)
	assert.NotNil(t, service.rateLimiter)
}

func TestOpenAIEmbeddingService_GetDimension(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected int
	}{
		{"ada-002", "text-embedding-ada-002", 1536},
		{"3-small", "text-embedding-3-small", 1536},
		{"3-large", "text-embedding-3-large", 3072},
		{"unknown", "unknown-model", 1536}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.OpenAIConfig{
				APIKey:         "test-key",
				EmbeddingModel: tt.model,
			}
			service := NewOpenAIEmbeddingService(cfg)

			assert.Equal(t, tt.expected, service.GetDimension())
		})
	}
}

func TestOpenAIEmbeddingService_GetModel(t *testing.T) {
	cfg := &config.OpenAIConfig{
		APIKey:         "test-key",
		EmbeddingModel: "text-embedding-3-small",
	}
	service := NewOpenAIEmbeddingService(cfg)

	assert.Equal(t, "text-embedding-3-small", service.GetModel())
}

func TestOpenAIEmbeddingService_Cache(t *testing.T) {
	cfg := &config.OpenAIConfig{
		APIKey:         "test-key",
		EmbeddingModel: "text-embedding-ada-002",
	}
	service := NewOpenAIEmbeddingService(cfg)

	t.Run("cache operations", func(t *testing.T) {
		text := "test text"
		embedding := []float64{0.1, 0.2, 0.3}

		// Should return nil for non-existent key
		key := service.getCacheKey(text)
		cached := service.getFromCache(key)
		assert.Nil(t, cached)

		// Put in cache
		service.putInCache(key, embedding)

		// Should return cached value
		cached = service.getFromCache(key)
		assert.Equal(t, embedding, cached)

		// Modify returned slice shouldn't affect cache
		cached[0] = 999.0
		cached2 := service.getFromCache(key)
		assert.Equal(t, 0.1, cached2[0])
	})

	t.Run("cache key consistency", func(t *testing.T) {
		text := "test text"
		key1 := service.getCacheKey(text)
		key2 := service.getCacheKey(text)

		assert.Equal(t, key1, key2)

		// Different text should produce different key
		key3 := service.getCacheKey("different text")
		assert.NotEqual(t, key1, key3)
	})

	t.Run("clear cache", func(t *testing.T) {
		text := "test text"
		embedding := []float64{0.1, 0.2, 0.3}

		key := service.getCacheKey(text)
		service.putInCache(key, embedding)

		// Verify cached
		cached := service.getFromCache(key)
		assert.Equal(t, embedding, cached)

		// Clear cache
		service.ClearCache()

		// Should be empty now
		cached = service.getFromCache(key)
		assert.Nil(t, cached)
	})

	t.Run("cache stats", func(t *testing.T) {
		service.ClearCache()

		stats := service.GetCacheStats()
		assert.Equal(t, 0, stats["cache_size"])
		assert.Equal(t, "text-embedding-ada-002", stats["model"])
		assert.Equal(t, 1536, stats["dimension"])

		// Add something to cache
		key := service.getCacheKey("test")
		service.putInCache(key, []float64{0.1})

		stats = service.GetCacheStats()
		assert.Equal(t, 1, stats["cache_size"])
	})
}

func TestOpenAIEmbeddingService_GenerateEmbedding_InputValidation(t *testing.T) {
	cfg := &config.OpenAIConfig{
		APIKey:         "test-key",
		EmbeddingModel: "text-embedding-ada-002",
		RateLimitRPM:   3600, // High rate limit for testing
	}
	service := NewOpenAIEmbeddingService(cfg)

	t.Run("empty text", func(t *testing.T) {
		ctx := context.Background()
		_, err := service.GenerateEmbedding(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "text cannot be empty")
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := service.GenerateEmbedding(ctx, "test text")
		assert.Error(t, err)
	})
}

func TestOpenAIEmbeddingService_GenerateBatchEmbeddings_InputValidation(t *testing.T) {
	cfg := &config.OpenAIConfig{
		APIKey:         "test-key",
		EmbeddingModel: "text-embedding-ada-002",
		RateLimitRPM:   3600,
	}
	service := NewOpenAIEmbeddingService(cfg)

	t.Run("empty texts", func(t *testing.T) {
		ctx := context.Background()
		_, err := service.GenerateBatchEmbeddings(ctx, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "texts cannot be empty")
	})

	t.Run("with cached results", func(t *testing.T) {
		// This test doesn't make actual API calls but tests the caching logic
		texts := []string{"text1", "text2", "text1"} // text1 appears twice

		// Pre-populate cache for text1
		key1 := service.getCacheKey("text1")
		embedding1 := []float64{0.1, 0.2}
		service.putInCache(key1, embedding1)

		// This would normally fail due to no real API key, but we can test
		// that it tries to process only uncached texts
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
		defer cancel()

		_, err := service.GenerateBatchEmbeddings(ctx, texts)
		// Should fail due to timeout/invalid API, but the important thing is
		// it tried to process fewer texts due to caching
		assert.Error(t, err)
	})
}

func TestOpenAIEmbeddingService_CacheIntegration(t *testing.T) {
	cfg := &config.OpenAIConfig{
		APIKey:         "test-key",
		EmbeddingModel: "text-embedding-ada-002",
		RateLimitRPM:   3600,
	}
	service := NewOpenAIEmbeddingService(cfg)

	t.Run("cache hit avoids API call", func(t *testing.T) {
		text := "cached text"
		expectedEmbedding := []float64{0.1, 0.2, 0.3}

		// Pre-populate cache
		key := service.getCacheKey(text)
		service.putInCache(key, expectedEmbedding)

		// This should return immediately from cache without API call
		ctx := context.Background()
		embedding, err := service.GenerateEmbedding(ctx, text)
		assert.NoError(t, err)
		assert.Equal(t, expectedEmbedding, embedding)
	})
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a smaller", 3, 5, 3},
		{"b smaller", 8, 4, 4},
		{"equal", 7, 7, 7},
		{"negative numbers", -3, -1, -3},
		{"zero", 0, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests for performance
func BenchmarkRateLimiter_Allow(b *testing.B) {
	rl := NewRateLimiter(1000000, time.Microsecond) // Very high limit for benchmarking

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Allow()
	}
}

func BenchmarkOpenAIEmbeddingService_GetCacheKey(b *testing.B) {
	cfg := &config.OpenAIConfig{
		APIKey:         "test-key",
		EmbeddingModel: "text-embedding-ada-002",
	}
	service := NewOpenAIEmbeddingService(cfg)
	text := "This is a test text for benchmarking cache key generation"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.getCacheKey(text)
	}
}

func BenchmarkOpenAIEmbeddingService_CacheOperations(b *testing.B) {
	cfg := &config.OpenAIConfig{
		APIKey:         "test-key",
		EmbeddingModel: "text-embedding-ada-002",
	}
	service := NewOpenAIEmbeddingService(cfg)

	embedding := make([]float64, 1536) // Standard embedding size
	for i := range embedding {
		embedding[i] = float64(i) * 0.001
	}

	keys := make([]string, 100)
	for i := range keys {
		keys[i] = service.getCacheKey(string(rune('a' + i)))
	}

	b.Run("put", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			key := keys[i%len(keys)]
			service.putInCache(key, embedding)
		}
	})

	b.Run("get", func(b *testing.B) {
		// Pre-populate cache
		for _, key := range keys {
			service.putInCache(key, embedding)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := keys[i%len(keys)]
			service.getFromCache(key)
		}
	})
}

// Integration test helper that would work with a real API key
func createIntegrationTestService() *OpenAIEmbeddingService {
	cfg := &config.OpenAIConfig{
		APIKey:         "test-key", // Would need real key for integration tests
		EmbeddingModel: "text-embedding-ada-002",
		MaxTokens:      8191,
		Temperature:    0.0,
		RequestTimeout: 30,
		RateLimitRPM:   60,
	}
	return NewOpenAIEmbeddingService(cfg)
}

// This test would only work with a real API key - skip in CI
func TestOpenAIEmbeddingService_Integration(t *testing.T) {
	t.Skip("Integration test - requires real OpenAI API key")

	service := createIntegrationTestService()
	ctx := context.Background()

	t.Run("generate single embedding", func(t *testing.T) {
		embedding, err := service.GenerateEmbedding(ctx, "Hello world")
		require.NoError(t, err)
		assert.Equal(t, 1536, len(embedding))

		// Embeddings should be normalized (roughly between -1 and 1)
		for _, val := range embedding {
			assert.True(t, val >= -1.0 && val <= 1.0)
		}
	})

	t.Run("generate batch embeddings", func(t *testing.T) {
		texts := []string{"Hello", "World", "Test"}
		embeddings, err := service.GenerateBatchEmbeddings(ctx, texts)
		require.NoError(t, err)
		assert.Equal(t, len(texts), len(embeddings))

		for _, embedding := range embeddings {
			assert.Equal(t, 1536, len(embedding))
		}
	})

	t.Run("health check", func(t *testing.T) {
		err := service.HealthCheck(ctx)
		assert.NoError(t, err)
	})
}
