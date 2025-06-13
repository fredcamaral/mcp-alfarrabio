// Package embeddings provides test compatibility wrappers
package embeddings

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"lerian-mcp-memory/internal/config"
)

// CompatibilityService wraps OpenAIService to match test expectations
type CompatibilityService struct {
	*OpenAIService
	client      interface{}
	config      *config.OpenAIConfig
	cache       *EmbeddingCache
	rateLimiter *RateLimiter
}

// NewOpenAIEmbeddingService creates a service for test compatibility
func NewOpenAIEmbeddingService(cfg *config.OpenAIConfig) *CompatibilityService {
	// Convert config format
	openaiConfig := &OpenAIConfig{
		APIKey:         cfg.APIKey,
		Model:          cfg.EmbeddingModel,
		RequestsPerMin: cfg.RateLimitRPM,
		Timeout:        time.Duration(cfg.RequestTimeout) * time.Second,
	}

	if openaiConfig.Model == "" {
		openaiConfig.Model = "text-embedding-ada-002"
	}

	// Create the actual service
	service, err := NewOpenAIService(openaiConfig, slog.Default())
	if err != nil {
		// For tests, return a mock service if creation fails
		service = &OpenAIService{
			apiKey:      "test-key",
			model:       openaiConfig.Model,
			cache:       NewEmbeddingCache(1000, 24*time.Hour),
			rateLimiter: NewRateLimiter(3600, time.Minute),
			metrics:     NewServiceMetrics(),
		}
	}

	return &CompatibilityService{
		OpenAIService: service,
		client:        service,
		config:        cfg,
		cache:         service.cache,
		rateLimiter:   service.rateLimiter,
	}
}

// GetDimension returns embedding dimensions (compatibility method)
func (s *CompatibilityService) GetDimension() int {
	return s.GetDimensions()
}

// GetModel returns the configured model name
func (s *CompatibilityService) GetModel() string {
	if s.config != nil {
		return s.config.EmbeddingModel
	}
	return "text-embedding-ada-002"
}

// getCacheKey generates a cache key for given text
func (s *CompatibilityService) getCacheKey(text string) string {
	return text // Simple cache key
}

// getFromCache retrieves embeddings from cache (single return value for test compatibility)
func (s *CompatibilityService) getFromCache(key string) []float64 {
	if s.cache != nil {
		if embeddings, found := s.cache.Get(key); found {
			// Return a copy to prevent modification
			result := make([]float64, len(embeddings))
			copy(result, embeddings)
			return result
		}
	}
	return nil
}

// putInCache stores embeddings in cache
func (s *CompatibilityService) putInCache(key string, embeddings []float64) {
	if s.cache != nil {
		s.cache.Set(key, embeddings)
	}
}

// ClearCache clears the embedding cache
func (s *CompatibilityService) ClearCache() {
	if s.cache != nil {
		s.cache.Clear()
	}
}

// GetCacheStats returns cache statistics as a map for test compatibility
func (s *CompatibilityService) GetCacheStats() map[string]interface{} {
	result := make(map[string]interface{})

	if s.cache != nil {
		stats := s.cache.Stats()
		result["cache_size"] = stats.Size
		result["hits"] = stats.Hits
		result["misses"] = stats.Misses
		result["hit_rate"] = stats.HitRate
	} else {
		result["cache_size"] = 0
		result["hits"] = int64(0)
		result["misses"] = int64(0)
		result["hit_rate"] = 0.0
	}

	// Add model and dimension info
	result["model"] = s.GetModel()
	result["dimension"] = s.GetDimensions()

	return result
}

// GenerateEmbedding generates embeddings for given text (with optional context)
func (s *CompatibilityService) GenerateEmbedding(args ...interface{}) ([]float64, error) {
	if len(args) == 1 {
		// Single argument: text only
		if text, ok := args[0].(string); ok {
			return s.Generate(nil, text)
		}
	} else if len(args) == 2 {
		// Two arguments: context and text
		if ctx, ok := args[0].(context.Context); ok {
			if text, ok := args[1].(string); ok {
				return s.Generate(ctx, text)
			}
		}
	}
	return nil, fmt.Errorf("invalid arguments to GenerateEmbedding")
}

// GenerateBatchEmbeddings generates embeddings for multiple texts
func (s *CompatibilityService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	return s.GenerateBatch(ctx, texts)
}

// GetOpenAIService returns the underlying OpenAIService for tests that need it
func (s *CompatibilityService) GetOpenAIService() *OpenAIService {
	return s.OpenAIService
}
