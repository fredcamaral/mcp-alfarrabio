// Package storage provides vector database and storage abstractions.
// It includes Qdrant integration, circuit breakers, retry logic, and storage interfaces.
package storage

import (
	"context"
	"lerian-mcp-memory/internal/workflow"
	"lerian-mcp-memory/pkg/types"
)

// VectorStorageAdapter adapts VectorStore to workflow.VectorStorage interface
type VectorStorageAdapter struct {
	store VectorStore
}

// NewVectorStorageAdapter creates a new vector storage adapter
func NewVectorStorageAdapter(store VectorStore) workflow.VectorStorage {
	return &VectorStorageAdapter{
		store: store,
	}
}

// Search performs a simplified search (without embeddings)
func (v *VectorStorageAdapter) Search(ctx context.Context, query string, filters map[string]interface{}, limit int) ([]types.ConversationChunk, error) {
	// Convert the simplified search to our MemoryQuery format
	memoryQuery := types.MemoryQuery{
		Query:   query,
		Limit:   limit,
		Recency: types.RecencyAllTime,
	}

	// Apply filters if provided
	if repo, ok := filters["repository"].(string); ok && repo != "" {
		memoryQuery.Repository = &repo
	}

	// For now, we can't perform vector search without embeddings
	// So we'll use ListByRepository as a fallback
	if memoryQuery.Repository != nil {
		return v.store.ListByRepository(ctx, *memoryQuery.Repository, limit, 0)
	}

	// If no specific repository, return empty for now
	// In a full implementation, this would generate embeddings for the query
	return []types.ConversationChunk{}, nil
}

// FindSimilar finds similar chunks based on content
func (v *VectorStorageAdapter) FindSimilar(ctx context.Context, content string, chunkType *types.ChunkType, limit int) ([]types.ConversationChunk, error) {
	return v.store.FindSimilar(ctx, content, chunkType, limit)
}
