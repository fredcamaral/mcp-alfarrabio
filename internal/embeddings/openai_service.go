// Package embeddings provides OpenAI embeddings integration with retry logic and caching
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultEmbeddingModel is the default OpenAI embedding model
	DefaultEmbeddingModel = "text-embedding-ada-002"
)

// OpenAIService implements embeddings generation using OpenAI API
type OpenAIService struct {
	apiKey      string
	baseURL     string
	model       string
	httpClient  *http.Client
	logger      *slog.Logger
	cache       *EmbeddingCache
	metrics     *ServiceMetrics
	rateLimiter *RateLimiter
}

// OpenAIConfig contains configuration for OpenAI embeddings service
type OpenAIConfig struct {
	APIKey         string        `json:"api_key"`
	BaseURL        string        `json:"base_url"`
	Model          string        `json:"model"`
	Timeout        time.Duration `json:"timeout"`
	MaxRetries     int           `json:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay"`
	CacheSize      int           `json:"cache_size"`
	CacheTTL       time.Duration `json:"cache_ttl"`
	RequestsPerMin int           `json:"requests_per_min"`
}

// DefaultOpenAIConfig returns sensible defaults for OpenAI embeddings
func DefaultOpenAIConfig() *OpenAIConfig {
	return &OpenAIConfig{
		BaseURL:        "https://api.openai.com/v1",
		Model:          DefaultEmbeddingModel,
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		RetryDelay:     1 * time.Second,
		CacheSize:      1000,
		CacheTTL:       24 * time.Hour,
		RequestsPerMin: 3000, // OpenAI default tier limit
	}
}

// NewOpenAIService creates a new OpenAI embeddings service
func NewOpenAIService(config *OpenAIConfig, logger *slog.Logger) (*OpenAIService, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	if logger == nil {
		logger = slog.Default()
	}

	if config.BaseURL == "" {
		config.BaseURL = DefaultOpenAIConfig().BaseURL
	}
	if config.Model == "" {
		config.Model = DefaultOpenAIConfig().Model
	}

	service := &OpenAIService{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		model:   config.Model,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger:      logger,
		cache:       NewEmbeddingCache(config.CacheSize, config.CacheTTL),
		metrics:     NewServiceMetrics(),
		rateLimiter: NewRateLimiter(config.RequestsPerMin, time.Minute),
	}

	return service, nil
}

// Generate creates embeddings for the given text
func (s *OpenAIService) Generate(ctx context.Context, text string) ([]float64, error) {
	start := time.Now()
	defer s.updateMetrics("generate", start)

	if strings.TrimSpace(text) == "" {
		s.incrementErrorCount("generate")
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Check cache first
	if cached, found := s.cache.Get(text); found {
		s.incrementCacheHit()
		return cached, nil
	}
	s.incrementCacheMiss()

	// Apply rate limiting
	if err := s.rateLimiter.Wait(ctx); err != nil {
		s.incrementErrorCount("generate")
		return nil, fmt.Errorf("rate limiting error: %w", err)
	}

	// Generate embeddings with retry logic
	embeddings, err := s.generateWithRetry(ctx, text)
	if err != nil {
		s.incrementErrorCount("generate")
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Cache the result
	s.cache.Set(text, embeddings)

	s.logger.Debug("embeddings generated successfully",
		slog.Int("dimensions", len(embeddings)),
		slog.Int("text_length", len(text)))

	return embeddings, nil
}

// GenerateBatch creates embeddings for multiple texts efficiently
func (s *OpenAIService) GenerateBatch(ctx context.Context, texts []string) ([][]float64, error) {
	start := time.Now()
	defer s.updateMetrics("generate_batch", start)

	if len(texts) == 0 {
		return [][]float64{}, nil
	}

	// Filter out cached embeddings and collect uncached texts
	var uncachedTexts []string
	var uncachedIndices []int
	results := make([][]float64, len(texts))

	for i, text := range texts {
		if strings.TrimSpace(text) == "" {
			s.incrementErrorCount("generate_batch")
			return nil, fmt.Errorf("text at index %d cannot be empty", i)
		}

		if cached, found := s.cache.Get(text); found {
			results[i] = cached
			s.incrementCacheHit()
		} else {
			uncachedTexts = append(uncachedTexts, text)
			uncachedIndices = append(uncachedIndices, i)
			s.incrementCacheMiss()
		}
	}

	// If all were cached, return early
	if len(uncachedTexts) == 0 {
		return results, nil
	}

	// Apply rate limiting for batch request
	if err := s.rateLimiter.Wait(ctx); err != nil {
		s.incrementErrorCount("generate_batch")
		return nil, fmt.Errorf("rate limiting error: %w", err)
	}

	// Generate embeddings for uncached texts
	embeddings, err := s.generateBatchWithRetry(ctx, uncachedTexts)
	if err != nil {
		s.incrementErrorCount("generate_batch")
		return nil, fmt.Errorf("failed to generate batch embeddings: %w", err)
	}

	// Fill in results and cache new embeddings
	for i, embedding := range embeddings {
		originalIndex := uncachedIndices[i]
		results[originalIndex] = embedding
		s.cache.Set(uncachedTexts[i], embedding)
	}

	s.logger.Debug("batch embeddings generated successfully",
		slog.Int("total_texts", len(texts)),
		slog.Int("cached", len(texts)-len(uncachedTexts)),
		slog.Int("generated", len(uncachedTexts)))

	return results, nil
}

// GetDimensions returns the embedding dimensions for the configured model
func (s *OpenAIService) GetDimensions() int {
	switch s.model {
	case DefaultEmbeddingModel:
		return 1536
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-3-large":
		return 3072
	default:
		return 1536 // Default to ada-002 dimensions
	}
}

// HealthCheck verifies the service is working properly
func (s *OpenAIService) HealthCheck(ctx context.Context) error {
	testText := "health check test"
	_, err := s.Generate(ctx, testText)
	return err
}

// GetMetrics returns current service metrics
func (s *OpenAIService) GetMetrics() *ServiceMetrics {
	return s.metrics
}

// Private methods

func (s *OpenAIService) generateWithRetry(ctx context.Context, text string) ([]float64, error) {
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		embeddings, err := s.callOpenAIAPI(ctx, []string{text})
		if err == nil && len(embeddings) > 0 {
			return embeddings[0], nil
		}

		lastErr = err
		s.logger.Warn("embedding generation attempt failed",
			slog.Int("attempt", attempt+1),
			slog.String("error", err.Error()))
	}

	return nil, fmt.Errorf("all retry attempts failed, last error: %w", lastErr)
}

func (s *OpenAIService) generateBatchWithRetry(ctx context.Context, texts []string) ([][]float64, error) {
	var lastErr error

	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		embeddings, err := s.callOpenAIAPI(ctx, texts)
		if err == nil {
			return embeddings, nil
		}

		lastErr = err
		s.logger.Warn("batch embedding generation attempt failed",
			slog.Int("attempt", attempt+1),
			slog.Int("texts_count", len(texts)),
			slog.String("error", err.Error()))
	}

	return nil, fmt.Errorf("all batch retry attempts failed, last error: %w", lastErr)
}

func (s *OpenAIService) callOpenAIAPI(ctx context.Context, texts []string) ([][]float64, error) {
	// Prepare request body
	requestBody := map[string]interface{}{
		"input": texts,
		"model": s.model,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	// Make the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle non-200 responses
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response OpenAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract embeddings
	embeddings := make([][]float64, len(response.Data))
	for i, item := range response.Data {
		embeddings[i] = item.Embedding
	}

	return embeddings, nil
}

func (s *OpenAIService) updateMetrics(operation string, start time.Time) {
	duration := time.Since(start)
	s.metrics.OperationCounts[operation]++

	// Update average latency
	current := s.metrics.AverageLatency[operation]
	count := s.metrics.OperationCounts[operation]
	s.metrics.AverageLatency[operation] = (current*float64(count-1) + duration.Seconds()) / float64(count)
}

func (s *OpenAIService) incrementErrorCount(operation string) {
	s.metrics.ErrorCounts[operation]++
}

func (s *OpenAIService) incrementCacheHit() {
	s.metrics.CacheHits++
}

func (s *OpenAIService) incrementCacheMiss() {
	s.metrics.CacheMisses++
}

// OpenAIResponse represents the response structure from OpenAI API.
type OpenAIResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// ServiceMetrics tracks embeddings service performance
type ServiceMetrics struct {
	OperationCounts map[string]int64   `json:"operation_counts"`
	AverageLatency  map[string]float64 `json:"average_latency"`
	ErrorCounts     map[string]int64   `json:"error_counts"`
	CacheHits       int64              `json:"cache_hits"`
	CacheMisses     int64              `json:"cache_misses"`
	LastUpdated     time.Time          `json:"last_updated"`
}

// NewServiceMetrics creates new service metrics
func NewServiceMetrics() *ServiceMetrics {
	return &ServiceMetrics{
		OperationCounts: make(map[string]int64),
		AverageLatency:  make(map[string]float64),
		ErrorCounts:     make(map[string]int64),
		LastUpdated:     time.Now(),
	}
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
