package embeddings

import (
	"context"
	"fmt"
	"mcp-memory/internal/retry"
	"strings"
	"time"
)

// RetryableEmbeddingService wraps an EmbeddingService with retry logic
type RetryableEmbeddingService struct {
	service EmbeddingService
	retrier *retry.Retrier
}

// NewRetryableEmbeddingService creates a new retryable embedding service
func NewRetryableEmbeddingService(service EmbeddingService, config *retry.Config) EmbeddingService {
	if config == nil {
		config = defaultEmbeddingRetryConfig()
	}
	return &RetryableEmbeddingService{
		service: service,
		retrier: retry.New(config),
	}
}

// defaultEmbeddingRetryConfig returns the default retry configuration for embedding operations
func defaultEmbeddingRetryConfig() *retry.Config {
	return &retry.Config{
		MaxAttempts:     3,
		InitialDelay:    500 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		Multiplier:      2.0,
		RandomizeFactor: 0.2,
		RetryIf:         isRetryableEmbeddingError,
	}
}

// isRetryableEmbeddingError determines if an embedding error should be retried
func isRetryableEmbeddingError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// OpenAI specific error patterns
	retryablePatterns := []string{
		// Network errors
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"i/o timeout",
		"eof",
		
		// HTTP status codes (as strings in errors)
		"429", // Too Many Requests
		"500", // Internal Server Error
		"502", // Bad Gateway
		"503", // Service Unavailable
		"504", // Gateway Timeout
		
		// OpenAI specific
		"rate limit",
		"quota exceeded",
		"overloaded",
		"temporarily unavailable",
		"server_error",
	}

	// Non-retryable patterns
	nonRetryablePatterns := []string{
		"invalid api key",
		"unauthorized",
		"forbidden",
		"insufficient_quota",
		"invalid_request_error",
		"model not found",
		"context length exceeded",
	}

	// Check non-retryable patterns first
	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errStr, pattern) {
			return false
		}
	}

	// Check retryable patterns
	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Check if error implements temporary interface
	type temporary interface {
		Temporary() bool
	}
	if te, ok := err.(temporary); ok {
		return te.Temporary()
	}

	// Don't retry by default for embedding services
	// as they often have specific error types
	return false
}

// GenerateEmbedding generates embeddings with retry logic
func (r *RetryableEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	var embeddings []float64
	
	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		embeddings, err = r.service.GenerateEmbedding(ctx, text)
		return err
	})
	
	if result.Err != nil {
		return nil, fmt.Errorf("failed to generate embedding after %d attempts: %w", result.Attempts, result.Err)
	}
	return embeddings, nil
}

// GenerateBatchEmbeddings generates multiple embeddings with retry logic
func (r *RetryableEmbeddingService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	var embeddings [][]float64
	
	// For batch operations, we might want a different retry strategy
	batchConfig := &retry.Config{
		MaxAttempts:     3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		Multiplier:      2.0,
		RandomizeFactor: 0.3,
		RetryIf:         isRetryableEmbeddingError,
	}
	
	batchRetrier := retry.New(batchConfig)
	result := batchRetrier.Do(ctx, func(ctx context.Context) error {
		var err error
		embeddings, err = r.service.GenerateBatchEmbeddings(ctx, texts)
		return err
	})
	
	if result.Err != nil {
		return nil, fmt.Errorf("failed to generate batch embeddings after %d attempts: %w", result.Attempts, result.Err)
	}
	return embeddings, nil
}

// HealthCheck performs health check with retry logic
func (r *RetryableEmbeddingService) HealthCheck(ctx context.Context) error {
	// Health checks should be quick but might fail due to transient issues
	healthConfig := &retry.Config{
		MaxAttempts:     5,
		InitialDelay:    200 * time.Millisecond,
		MaxDelay:        2 * time.Second,
		Multiplier:      1.5,
		RandomizeFactor: 0.1,
		RetryIf:         isRetryableEmbeddingError,
	}
	
	healthRetrier := retry.New(healthConfig)
	result := healthRetrier.Do(ctx, func(ctx context.Context) error {
		return r.service.HealthCheck(ctx)
	})
	
	if result.Err != nil {
		return fmt.Errorf("health check failed after %d attempts: %w", result.Attempts, result.Err)
	}
	return nil
}

// GetModel returns the model name (no retry needed)
func (r *RetryableEmbeddingService) GetModel() string {
	return r.service.GetModel()
}

// GetDimension returns the embedding dimension (no retry needed)
func (r *RetryableEmbeddingService) GetDimension() int {
	return r.service.GetDimension()
}

// RateLimitAwareRetryConfig creates a retry config that respects rate limits
func RateLimitAwareRetryConfig() *retry.Config {
	return &retry.Config{
		MaxAttempts:     5,
		InitialDelay:    1 * time.Second,
		MaxDelay:        60 * time.Second,
		Multiplier:      2.0,
		RandomizeFactor: 0.5, // Higher jitter for rate limits
		RetryIf: func(err error) bool {
			if err == nil {
				return false
			}
			
			errStr := strings.ToLower(err.Error())
			// Only retry on rate limit errors
			return strings.Contains(errStr, "429") || 
				   strings.Contains(errStr, "rate limit") ||
				   strings.Contains(errStr, "quota exceeded")
		},
	}
}

// CircuitBreakerRetryConfig creates a retry config with circuit breaker behavior
func CircuitBreakerRetryConfig() *retry.Config {
	failureCount := 0
	lastFailure := time.Time{}
	
	return &retry.Config{
		MaxAttempts:     3,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        5 * time.Second,
		Multiplier:      2.0,
		RandomizeFactor: 0.1,
		RetryIf: func(err error) bool {
			if err == nil {
				failureCount = 0
				return false
			}
			
			now := time.Now()
			
			// Reset counter if last failure was more than 5 minutes ago
			if now.Sub(lastFailure) > 5*time.Minute {
				failureCount = 0
			}
			
			failureCount++
			lastFailure = now
			
			// Circuit breaker: stop retrying after 10 consecutive failures
			if failureCount > 10 {
				return false
			}
			
			return isRetryableEmbeddingError(err)
		},
	}
}