package storage

import (
	"context"
	"fmt"
	"lerian-mcp-memory/internal/retry"
	"lerian-mcp-memory/pkg/types"
	"time"
)

// RetryableVectorStore wraps a VectorStore with retry logic
type RetryableVectorStore struct {
	store   VectorStore
	retrier *retry.Retrier
}

// NewRetryableVectorStore creates a new retryable vector store
func NewRetryableVectorStore(store VectorStore, config *retry.Config) VectorStore {
	if config == nil {
		config = defaultRetryConfig()
	}
	return &RetryableVectorStore{
		store:   store,
		retrier: retry.New(config),
	}
}

// defaultRetryConfig returns the default retry configuration for storage operations
func defaultRetryConfig() *retry.Config {
	return &retry.Config{
		MaxAttempts:     3,
		InitialDelay:    200 * time.Millisecond,
		MaxDelay:        5 * time.Second,
		Multiplier:      2.0,
		RandomizeFactor: 0.1,
		RetryIf:         isRetryableStorageError,
	}
}

// isRetryableStorageError determines if a storage error should be retried
func isRetryableStorageError(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error messages that indicate transient issues
	errStr := err.Error()
	transientPatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"too many requests",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
	}

	for _, pattern := range transientPatterns {
		if containsIgnoreCase(errStr, pattern) {
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

	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			containsIgnoreCaseImpl(s, substr))
}

func containsIgnoreCaseImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalsFoldRange(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalsFoldRange(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if toLower(s[i]) != toLower(t[i]) {
			return false
		}
	}
	return true
}

func toLower(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// Initialize initializes the vector store with retries
func (r *RetryableVectorStore) Initialize(ctx context.Context) error {
	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		return r.store.Initialize(ctx)
	})
	if result.Err != nil {
		return fmt.Errorf("failed to initialize after %d attempts: %w", result.Attempts, result.Err)
	}
	return nil
}

// Store stores a chunk with retries
func (r *RetryableVectorStore) Store(ctx context.Context, chunk *types.ConversationChunk) error {
	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		return r.store.Store(ctx, chunk)
	})
	if result.Err != nil {
		return fmt.Errorf("failed to store chunk after %d attempts: %w", result.Attempts, result.Err)
	}
	return nil
}

// Search performs search with retries
func (r *RetryableVectorStore) Search(ctx context.Context, query *types.MemoryQuery, embeddings []float64) (*types.SearchResults, error) {
	var results *types.SearchResults

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		results, err = r.store.Search(ctx, query, embeddings)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("search failed after %d attempts: %w", result.Attempts, result.Err)
	}
	return results, nil
}

// GetByID gets a chunk by ID with retries
func (r *RetryableVectorStore) GetByID(ctx context.Context, id string) (*types.ConversationChunk, error) {
	var chunk *types.ConversationChunk

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		chunk, err = r.store.GetByID(ctx, id)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("failed to get chunk by ID after %d attempts: %w", result.Attempts, result.Err)
	}
	return chunk, nil
}

// ListByRepository lists chunks with retries
func (r *RetryableVectorStore) ListByRepository(ctx context.Context, repository string, limit, offset int) ([]types.ConversationChunk, error) {
	var chunks []types.ConversationChunk

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		chunks, err = r.store.ListByRepository(ctx, repository, limit, offset)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("failed to list chunks after %d attempts: %w", result.Attempts, result.Err)
	}
	return chunks, nil
}

// ListBySession lists chunks by session ID with retries
func (r *RetryableVectorStore) ListBySession(ctx context.Context, sessionID string) ([]types.ConversationChunk, error) {
	var chunks []types.ConversationChunk

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		chunks, err = r.store.ListBySession(ctx, sessionID)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("failed to list chunks by session after %d attempts: %w", result.Attempts, result.Err)
	}
	return chunks, nil
}

// Delete deletes a chunk with retries
func (r *RetryableVectorStore) Delete(ctx context.Context, id string) error {
	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		return r.store.Delete(ctx, id)
	})
	if result.Err != nil {
		return fmt.Errorf("failed to delete chunk after %d attempts: %w", result.Attempts, result.Err)
	}
	return nil
}

// Update updates a chunk with retries
func (r *RetryableVectorStore) Update(ctx context.Context, chunk *types.ConversationChunk) error {
	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		return r.store.Update(ctx, chunk)
	})
	if result.Err != nil {
		return fmt.Errorf("failed to update chunk after %d attempts: %w", result.Attempts, result.Err)
	}
	return nil
}

// HealthCheck performs health check with retries
func (r *RetryableVectorStore) HealthCheck(ctx context.Context) error {
	// Health check uses a more aggressive retry strategy
	healthConfig := &retry.Config{
		MaxAttempts:     5,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        2 * time.Second,
		Multiplier:      1.5,
		RandomizeFactor: 0.1,
		RetryIf:         isRetryableStorageError,
	}

	healthRetrier := retry.New(healthConfig)
	result := healthRetrier.Do(ctx, func(ctx context.Context) error {
		return r.store.HealthCheck(ctx)
	})

	if result.Err != nil {
		return fmt.Errorf("health check failed after %d attempts: %w", result.Attempts, result.Err)
	}
	return nil
}

// GetStats gets statistics with retries
func (r *RetryableVectorStore) GetStats(ctx context.Context) (*StoreStats, error) {
	var stats *StoreStats

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		stats, err = r.store.GetStats(ctx)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("failed to get stats after %d attempts: %w", result.Attempts, result.Err)
	}
	return stats, nil
}

// Cleanup performs cleanup with retries
func (r *RetryableVectorStore) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	var count int

	// Cleanup might take longer, use different config
	cleanupConfig := &retry.Config{
		MaxAttempts:     3,
		InitialDelay:    500 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		Multiplier:      2.0,
		RandomizeFactor: 0.1,
		RetryIf:         isRetryableStorageError,
	}

	cleanupRetrier := retry.New(cleanupConfig)
	result := cleanupRetrier.Do(ctx, func(ctx context.Context) error {
		var err error
		count, err = r.store.Cleanup(ctx, retentionDays)
		return err
	})

	if result.Err != nil {
		return 0, fmt.Errorf("cleanup failed after %d attempts: %w", result.Attempts, result.Err)
	}
	return count, nil
}

// Close closes the connection (no retry needed)
func (r *RetryableVectorStore) Close() error {
	return r.store.Close()
}

// Additional methods for service compatibility (with retries)

// GetAllChunks gets all chunks with retries
func (r *RetryableVectorStore) GetAllChunks(ctx context.Context) ([]types.ConversationChunk, error) {
	var chunks []types.ConversationChunk

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		chunks, err = r.store.GetAllChunks(ctx)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("failed to get all chunks after %d attempts: %w", result.Attempts, result.Err)
	}
	return chunks, nil
}

// DeleteCollection deletes collection with retries
func (r *RetryableVectorStore) DeleteCollection(ctx context.Context, collection string) error {
	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		return r.store.DeleteCollection(ctx, collection)
	})
	if result.Err != nil {
		return fmt.Errorf("failed to delete collection after %d attempts: %w", result.Attempts, result.Err)
	}
	return nil
}

// ListCollections lists collections with retries
func (r *RetryableVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	var collections []string

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		collections, err = r.store.ListCollections(ctx)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("failed to list collections after %d attempts: %w", result.Attempts, result.Err)
	}
	return collections, nil
}

// FindSimilar finds similar chunks with retries
func (r *RetryableVectorStore) FindSimilar(ctx context.Context, content string, chunkType *types.ChunkType, limit int) ([]types.ConversationChunk, error) {
	var chunks []types.ConversationChunk

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		chunks, err = r.store.FindSimilar(ctx, content, chunkType, limit)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("failed to find similar chunks after %d attempts: %w", result.Attempts, result.Err)
	}
	return chunks, nil
}

// StoreChunk stores chunk with retries
func (r *RetryableVectorStore) StoreChunk(ctx context.Context, chunk *types.ConversationChunk) error {
	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		return r.store.StoreChunk(ctx, chunk)
	})
	if result.Err != nil {
		return fmt.Errorf("failed to store chunk after %d attempts: %w", result.Attempts, result.Err)
	}
	return nil
}

// BatchStore stores chunks in batch with retries
func (r *RetryableVectorStore) BatchStore(ctx context.Context, chunks []*types.ConversationChunk) (*BatchResult, error) {
	var result *BatchResult

	retryResult := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		result, err = r.store.BatchStore(ctx, chunks)
		return err
	})

	if retryResult.Err != nil {
		return nil, fmt.Errorf("batch store failed after %d attempts: %w", retryResult.Attempts, retryResult.Err)
	}
	return result, nil
}

// BatchDelete deletes chunks in batch with retries
func (r *RetryableVectorStore) BatchDelete(ctx context.Context, ids []string) (*BatchResult, error) {
	var result *BatchResult

	retryResult := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		result, err = r.store.BatchDelete(ctx, ids)
		return err
	})

	if retryResult.Err != nil {
		return nil, fmt.Errorf("batch delete failed after %d attempts: %w", retryResult.Attempts, retryResult.Err)
	}
	return result, nil
}

// Relationship management methods

// StoreRelationship stores a relationship with retries
func (r *RetryableVectorStore) StoreRelationship(ctx context.Context, sourceID, targetID string, relationType types.RelationType, confidence float64, source types.ConfidenceSource) (*types.MemoryRelationship, error) {
	var relationship *types.MemoryRelationship

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		relationship, err = r.store.StoreRelationship(ctx, sourceID, targetID, relationType, confidence, source)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("failed to store relationship after %d attempts: %w", result.Attempts, result.Err)
	}
	return relationship, nil
}

// GetRelationships gets relationships with retries
func (r *RetryableVectorStore) GetRelationships(ctx context.Context, query *types.RelationshipQuery) ([]types.RelationshipResult, error) {
	var relationships []types.RelationshipResult

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		relationships, err = r.store.GetRelationships(ctx, query)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("failed to get relationships after %d attempts: %w", result.Attempts, result.Err)
	}
	return relationships, nil
}

// TraverseGraph traverses graph with retries
func (r *RetryableVectorStore) TraverseGraph(ctx context.Context, startChunkID string, maxDepth int, relationTypes []types.RelationType) (*types.GraphTraversalResult, error) {
	var traversalResult *types.GraphTraversalResult

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		traversalResult, err = r.store.TraverseGraph(ctx, startChunkID, maxDepth, relationTypes)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("failed to traverse graph after %d attempts: %w", result.Attempts, result.Err)
	}
	return traversalResult, nil
}

// UpdateRelationship updates a relationship with retries
func (r *RetryableVectorStore) UpdateRelationship(ctx context.Context, relationshipID string, confidence float64, factors types.ConfidenceFactors) error {
	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		return r.store.UpdateRelationship(ctx, relationshipID, confidence, factors)
	})

	if result.Err != nil {
		return fmt.Errorf("failed to update relationship after %d attempts: %w", result.Attempts, result.Err)
	}
	return nil
}

// DeleteRelationship deletes a relationship with retries
func (r *RetryableVectorStore) DeleteRelationship(ctx context.Context, relationshipID string) error {
	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		return r.store.DeleteRelationship(ctx, relationshipID)
	})

	if result.Err != nil {
		return fmt.Errorf("failed to delete relationship after %d attempts: %w", result.Attempts, result.Err)
	}
	return nil
}

// GetRelationshipByID gets a relationship by ID with retries
func (r *RetryableVectorStore) GetRelationshipByID(ctx context.Context, relationshipID string) (*types.MemoryRelationship, error) {
	var relationship *types.MemoryRelationship

	result := r.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		relationship, err = r.store.GetRelationshipByID(ctx, relationshipID)
		return err
	})

	if result.Err != nil {
		return nil, fmt.Errorf("failed to get relationship by ID after %d attempts: %w", result.Attempts, result.Err)
	}
	return relationship, nil
}
