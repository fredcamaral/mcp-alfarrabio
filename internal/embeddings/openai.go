// Package embeddings provides OpenAI integration for generating and managing
// text embeddings with circuit breaker and retry capabilities.
package embeddings

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"lerian-mcp-memory/internal/config"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
)

// EmbeddingService defines the interface for generating embeddings
type EmbeddingService interface {
	// Generate embeddings for a single text
	GenerateEmbedding(ctx context.Context, text string) ([]float64, error)

	// Generate embeddings for multiple texts in batch
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error)

	// Get the dimension of embeddings produced by this service
	GetDimension() int

	// Get the model name
	GetModel() string

	// Health check
	HealthCheck(ctx context.Context) error
}

// OpenAIEmbeddingService implements EmbeddingService using OpenAI's API
type OpenAIEmbeddingService struct {
	client      *openai.Client
	config      *config.OpenAIConfig
	cache       map[string][]float64
	cacheMu     sync.RWMutex
	rateLimiter *RateLimiter
}

// RateLimiter implements a simple rate limiter for API calls
type RateLimiter struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request can proceed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	// Refill tokens based on elapsed time
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		rl.tokens = minInt(rl.maxTokens, rl.tokens+tokensToAdd)
		rl.lastRefill = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// Wait blocks until a request can proceed
func (rl *RateLimiter) Wait(ctx context.Context) error {
	for {
		if rl.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}
}

// NewOpenAIEmbeddingService creates a new OpenAI embedding service
func NewOpenAIEmbeddingService(cfg *config.OpenAIConfig) *OpenAIEmbeddingService {
	client := openai.NewClient(cfg.APIKey)

	// Create rate limiter: allow 1 request per minute / max_rpm
	// Ensure RateLimitRPM is at least 1 to avoid divide by zero
	rpm := cfg.RateLimitRPM
	if rpm <= 0 {
		rpm = getEnvInt("MCP_MEMORY_OPENAI_DEFAULT_RPM", 60) // Default to configured RPM or 60
	}
	refillRate := time.Minute / time.Duration(rpm)
	rateLimiter := NewRateLimiter(rpm, refillRate)

	return &OpenAIEmbeddingService{
		client:      client,
		config:      cfg,
		cache:       make(map[string][]float64),
		rateLimiter: rateLimiter,
	}
}

// GenerateEmbedding generates an embedding for a single text
func (oes *OpenAIEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	if text == "" {
		return nil, errors.New("text cannot be empty")
	}

	// Check cache first
	cacheKey := oes.getCacheKey(text)
	if cached := oes.getFromCache(cacheKey); cached != nil {
		return cached, nil
	}

	// Wait for rate limiter
	if err := oes.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Create embedding request
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(oes.config.EmbeddingModel),
	}

	// Add timeout to context
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(oes.config.RequestTimeout)*time.Second)
	defer cancel()

	// Make API call
	resp, err := oes.client.CreateEmbeddings(timeoutCtx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, errors.New("no embeddings returned")
	}

	embedding := resp.Data[0].Embedding

	// Convert []float32 to []float64
	embeddingFloat64 := make([]float64, len(embedding))
	for i, v := range embedding {
		embeddingFloat64[i] = float64(v)
	}

	// Cache the result
	oes.putInCache(cacheKey, embeddingFloat64)

	return embeddingFloat64, nil
}

// GenerateBatchEmbeddings generates embeddings for multiple texts
func (oes *OpenAIEmbeddingService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, errors.New("texts cannot be empty")
	}

	// Filter out cached embeddings and prepare uncached texts
	uncachedTexts := []string{}
	uncachedIndices := []int{}
	results := make([][]float64, len(texts))

	for i, text := range texts {
		if text == "" {
			continue
		}

		cacheKey := oes.getCacheKey(text)
		if cached := oes.getFromCache(cacheKey); cached != nil {
			results[i] = cached
		} else {
			uncachedTexts = append(uncachedTexts, text)
			uncachedIndices = append(uncachedIndices, i)
		}
	}

	// If all were cached, return immediately
	if len(uncachedTexts) == 0 {
		return results, nil
	}

	// Wait for rate limiter
	if err := oes.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	// Create batch embedding request
	req := openai.EmbeddingRequest{
		Input: uncachedTexts,
		Model: openai.EmbeddingModel(oes.config.EmbeddingModel),
	}

	// Add timeout to context
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(oes.config.RequestTimeout)*time.Second)
	defer cancel()

	// Make API call
	resp, err := oes.client.CreateEmbeddings(timeoutCtx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch embeddings: %w", err)
	}

	if len(resp.Data) != len(uncachedTexts) {
		return nil, fmt.Errorf("mismatch between input texts (%d) and embeddings (%d)", len(uncachedTexts), len(resp.Data))
	}

	// Place embeddings in correct positions and cache them
	for i, embeddingData := range resp.Data {
		embedding := embeddingData.Embedding

		// Convert []float32 to []float64
		embeddingFloat64 := make([]float64, len(embedding))
		for j, v := range embedding {
			embeddingFloat64[j] = float64(v)
		}

		resultIndex := uncachedIndices[i]
		results[resultIndex] = embeddingFloat64

		// Cache the result
		cacheKey := oes.getCacheKey(uncachedTexts[i])
		oes.putInCache(cacheKey, embeddingFloat64)
	}

	return results, nil
}

// GetDimension returns the dimension of embeddings produced by this service
func (oes *OpenAIEmbeddingService) GetDimension() int {
	// text-embedding-ada-002 produces 1536-dimensional embeddings
	switch oes.config.EmbeddingModel {
	case "text-embedding-ada-002":
		return 1536
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-3-large":
		return 3072
	default:
		return 1536 // Default assumption
	}
}

// GetModel returns the model name
func (oes *OpenAIEmbeddingService) GetModel() string {
	return oes.config.EmbeddingModel
}

// HealthCheck verifies the service is working
func (oes *OpenAIEmbeddingService) HealthCheck(ctx context.Context) error {
	// Try to generate a simple embedding
	_, err := oes.GenerateEmbedding(ctx, "health check")
	return err
}

// Cache management methods

func (oes *OpenAIEmbeddingService) getCacheKey(text string) string {
	// Create a hash of the text for consistent caching
	hash := sha256.Sum256([]byte(oes.config.EmbeddingModel + "|" + text))
	return fmt.Sprintf("%x", hash)
}

func (oes *OpenAIEmbeddingService) getFromCache(key string) []float64 {
	oes.cacheMu.RLock()
	defer oes.cacheMu.RUnlock()

	if embedding, exists := oes.cache[key]; exists {
		// Return a copy to prevent modification
		result := make([]float64, len(embedding))
		copy(result, embedding)
		return result
	}

	return nil
}

func (oes *OpenAIEmbeddingService) putInCache(key string, embedding []float64) {
	oes.cacheMu.Lock()
	defer oes.cacheMu.Unlock()

	// Store a copy to prevent external modification
	cached := make([]float64, len(embedding))
	copy(cached, embedding)
	oes.cache[key] = cached

	// Simple cache size management - remove oldest entries if cache gets too large
	maxCacheSize := getEnvInt("MCP_MEMORY_CACHE_MAX_SIZE", 1000)
	if len(oes.cache) > maxCacheSize {
		// Remove random entries (in practice, you'd want LRU or similar)
		count := 0
		cleanupBatch := getEnvInt("MCP_MEMORY_CACHE_CLEANUP_BATCH", 100)
		for k := range oes.cache {
			delete(oes.cache, k)
			count++
			if count >= cleanupBatch {
				break
			}
		}
		log.Printf("Cache cleanup: removed %d entries", count)
	}
}

// ClearCache clears the embedding cache
func (oes *OpenAIEmbeddingService) ClearCache() {
	oes.cacheMu.Lock()
	defer oes.cacheMu.Unlock()
	oes.cache = make(map[string][]float64)
}

// GetCacheStats returns cache statistics
func (oes *OpenAIEmbeddingService) GetCacheStats() map[string]interface{} {
	oes.cacheMu.RLock()
	defer oes.cacheMu.RUnlock()

	return map[string]interface{}{
		"cache_size": len(oes.cache),
		"model":      oes.config.EmbeddingModel,
		"dimension":  oes.GetDimension(),
	}
}

// Helper function for Go 1.24.3 (older version doesn't have min built-in)
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getEnvInt gets an integer from environment variable with a default
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}
