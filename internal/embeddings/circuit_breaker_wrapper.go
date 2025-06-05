package embeddings

import (
	"context"
	"fmt"
	"lerian-mcp-memory/internal/circuitbreaker"
	"time"
)

// CircuitBreakerEmbeddingService wraps an EmbeddingService with circuit breaker protection
type CircuitBreakerEmbeddingService struct {
	service EmbeddingService
	cb      *circuitbreaker.CircuitBreaker
}

// NewCircuitBreakerEmbeddingService creates a new circuit breaker wrapped service
func NewCircuitBreakerEmbeddingService(service EmbeddingService, config *circuitbreaker.Config) *CircuitBreakerEmbeddingService {
	if config == nil {
		config = &circuitbreaker.Config{
			FailureThreshold:      3, // Lower threshold for embedding service
			SuccessThreshold:      2,
			Timeout:               20 * time.Second,
			MaxConcurrentRequests: 5,
			OnStateChange: func(from, to circuitbreaker.State) {
				// Log state changes
				fmt.Printf("EmbeddingService circuit breaker: %s -> %s\n", from, to)
			},
		}
	}

	return &CircuitBreakerEmbeddingService{
		service: service,
		cb:      circuitbreaker.New(config),
	}
}

// GenerateEmbedding generates embeddings with circuit breaker protection
func (s *CircuitBreakerEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	var result []float64

	err := s.cb.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			var err error
			result, err = s.service.GenerateEmbedding(ctx, text)
			return err
		},
		func(ctx context.Context, cbErr error) error {
			// For embeddings, we can't provide a meaningful fallback
			// Return the circuit breaker error
			return fmt.Errorf("embedding service unavailable: %w", cbErr)
		},
	)

	return result, err
}

// GenerateBatchEmbeddings generates batch embeddings with circuit breaker protection
func (s *CircuitBreakerEmbeddingService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	var result [][]float64

	err := s.cb.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			var err error
			result, err = s.service.GenerateBatchEmbeddings(ctx, texts)
			return err
		},
		func(ctx context.Context, cbErr error) error {
			return fmt.Errorf("embedding service unavailable: %w", cbErr)
		},
	)

	return result, err
}

// HealthCheck performs a health check
func (s *CircuitBreakerEmbeddingService) HealthCheck(ctx context.Context) error {
	return s.cb.Execute(ctx, func(ctx context.Context) error {
		return s.service.HealthCheck(ctx)
	})
}

// GetDimension returns the embedding dimension
func (s *CircuitBreakerEmbeddingService) GetDimension() int {
	return s.service.GetDimension()
}

// GetModel returns the model name
func (s *CircuitBreakerEmbeddingService) GetModel() string {
	return s.service.GetModel()
}

// GetCircuitBreakerStats returns circuit breaker statistics
func (s *CircuitBreakerEmbeddingService) GetCircuitBreakerStats() circuitbreaker.Stats {
	return s.cb.GetStats()
}
