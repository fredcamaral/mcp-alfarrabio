package storage

import (
	"context"
	"fmt"
	"mcp-memory/internal/circuitbreaker"
	"mcp-memory/pkg/types"
	"time"
)

// CircuitBreakerVectorStore wraps a VectorStore with circuit breaker protection
type CircuitBreakerVectorStore struct {
	store VectorStore
	cb    *circuitbreaker.CircuitBreaker
}

// NewCircuitBreakerVectorStore creates a new circuit breaker wrapped store
func NewCircuitBreakerVectorStore(store VectorStore, config *circuitbreaker.Config) *CircuitBreakerVectorStore {
	if config == nil {
		config = &circuitbreaker.Config{
			FailureThreshold:      5,
			SuccessThreshold:      2,
			Timeout:               30 * time.Second,
			MaxConcurrentRequests: 3,
			OnStateChange: func(from, to circuitbreaker.State) {
				// Log state changes
				fmt.Printf("VectorStore circuit breaker: %s -> %s\n", from, to)
			},
		}
	}

	return &CircuitBreakerVectorStore{
		store: store,
		cb:    circuitbreaker.New(config),
	}
}

// Initialize initializes the store
func (s *CircuitBreakerVectorStore) Initialize(ctx context.Context) error {
	return s.cb.Execute(ctx, func(ctx context.Context) error {
		return s.store.Initialize(ctx)
	})
}

// Store stores a chunk
func (s *CircuitBreakerVectorStore) Store(ctx context.Context, chunk types.ConversationChunk) error {
	return s.cb.Execute(ctx, func(ctx context.Context) error {
		return s.store.Store(ctx, chunk)
	})
}

// Search performs a search with fallback to empty results
func (s *CircuitBreakerVectorStore) Search(ctx context.Context, query types.MemoryQuery, embeddings []float64) (*types.SearchResults, error) {
	var result *types.SearchResults
	
	err := s.cb.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			var err error
			result, err = s.store.Search(ctx, query, embeddings)
			return err
		},
		func(ctx context.Context, cbErr error) error {
			// Return empty results on circuit breaker failure
			result = &types.SearchResults{
				Results: []types.SearchResult{},
				Total:   0,
			}
			return nil
		},
	)
	
	return result, err
}

// GetByID gets a chunk by ID
func (s *CircuitBreakerVectorStore) GetByID(ctx context.Context, id string) (*types.ConversationChunk, error) {
	var result *types.ConversationChunk
	
	err := s.cb.Execute(ctx, func(ctx context.Context) error {
		var err error
		result, err = s.store.GetByID(ctx, id)
		return err
	})
	
	return result, err
}

// ListByRepository lists chunks by repository
func (s *CircuitBreakerVectorStore) ListByRepository(ctx context.Context, repository string, limit int, offset int) ([]types.ConversationChunk, error) {
	var result []types.ConversationChunk
	
	err := s.cb.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			var err error
			result, err = s.store.ListByRepository(ctx, repository, limit, offset)
			return err
		},
		func(ctx context.Context, cbErr error) error {
			// Return empty list on circuit breaker failure
			result = []types.ConversationChunk{}
			return nil
		},
	)
	
	return result, err
}

// ListBySession lists chunks by session ID
func (s *CircuitBreakerVectorStore) ListBySession(ctx context.Context, sessionID string) ([]types.ConversationChunk, error) {
	var result []types.ConversationChunk
	
	err := s.cb.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			var err error
			result, err = s.store.ListBySession(ctx, sessionID)
			return err
		},
		func(ctx context.Context, cbErr error) error {
			// Return empty list on circuit breaker failure
			result = []types.ConversationChunk{}
			return nil
		},
	)
	
	return result, err
}

// Delete deletes a chunk
func (s *CircuitBreakerVectorStore) Delete(ctx context.Context, id string) error {
	return s.cb.Execute(ctx, func(ctx context.Context) error {
		return s.store.Delete(ctx, id)
	})
}

// Update updates a chunk
func (s *CircuitBreakerVectorStore) Update(ctx context.Context, chunk types.ConversationChunk) error {
	return s.cb.Execute(ctx, func(ctx context.Context) error {
		return s.store.Update(ctx, chunk)
	})
}

// HealthCheck performs a health check
func (s *CircuitBreakerVectorStore) HealthCheck(ctx context.Context) error {
	return s.cb.Execute(ctx, func(ctx context.Context) error {
		return s.store.HealthCheck(ctx)
	})
}

// GetStats gets store statistics with fallback
func (s *CircuitBreakerVectorStore) GetStats(ctx context.Context) (*StoreStats, error) {
	var result *StoreStats
	
	err := s.cb.ExecuteWithFallback(ctx,
		func(ctx context.Context) error {
			var err error
			result, err = s.store.GetStats(ctx)
			return err
		},
		func(ctx context.Context, cbErr error) error {
			// Return empty stats on circuit breaker failure
			result = &StoreStats{
				TotalChunks:  0,
				ChunksByType: make(map[string]int64),
				ChunksByRepo: make(map[string]int64),
			}
			return nil
		},
	)
	
	return result, err
}

// Cleanup performs cleanup
func (s *CircuitBreakerVectorStore) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	var result int
	
	err := s.cb.Execute(ctx, func(ctx context.Context) error {
		var err error
		result, err = s.store.Cleanup(ctx, retentionDays)
		return err
	})
	
	return result, err
}

// Close closes the store
func (s *CircuitBreakerVectorStore) Close() error {
	// Don't use circuit breaker for close operations
	return s.store.Close()
}

// GetCircuitBreakerStats returns circuit breaker statistics
func (s *CircuitBreakerVectorStore) GetCircuitBreakerStats() circuitbreaker.Stats {
	return s.cb.GetStats()
}