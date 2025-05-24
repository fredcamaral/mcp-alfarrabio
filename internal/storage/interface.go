package storage

import (
	"context"
	"mcp-memory/pkg/types"
)

// VectorStore defines the interface for vector database operations
type VectorStore interface {
	// Initialize the vector store (create collections, etc.)
	Initialize(ctx context.Context) error
	
	// Store a conversation chunk with embeddings
	Store(ctx context.Context, chunk types.ConversationChunk) error
	
	// Search for similar chunks based on query embeddings
	Search(ctx context.Context, query types.MemoryQuery, embeddings []float64) (*types.SearchResults, error)
	
	// Get a chunk by its ID
	GetByID(ctx context.Context, id string) (*types.ConversationChunk, error)
	
	// List chunks by repository with optional filters
	ListByRepository(ctx context.Context, repository string, limit int, offset int) ([]types.ConversationChunk, error)
	
	// Delete a chunk by ID
	Delete(ctx context.Context, id string) error
	
	// Update a chunk
	Update(ctx context.Context, chunk types.ConversationChunk) error
	
	// Health check for the vector store
	HealthCheck(ctx context.Context) error
	
	// Get statistics about the store
	GetStats(ctx context.Context) (*StoreStats, error)
	
	// Cleanup old chunks based on retention policy
	Cleanup(ctx context.Context, retentionDays int) (int, error)
	
	// Close the connection
	Close() error
}

// StoreStats represents statistics about the vector store
type StoreStats struct {
	TotalChunks      int64             `json:"total_chunks"`
	ChunksByType     map[string]int64  `json:"chunks_by_type"`
	ChunksByRepo     map[string]int64  `json:"chunks_by_repo"`
	OldestChunk      *string           `json:"oldest_chunk,omitempty"`
	NewestChunk      *string           `json:"newest_chunk,omitempty"`
	StorageSize      int64             `json:"storage_size_bytes"`
	AverageEmbedding float64           `json:"average_embedding_size"`
}

// SearchFilter represents additional filters for search operations
type SearchFilter struct {
	Repository    *string               `json:"repository,omitempty"`
	ChunkTypes    []types.ChunkType     `json:"chunk_types,omitempty"`
	TimeRange     *TimeRange            `json:"time_range,omitempty"`
	Tags          []string              `json:"tags,omitempty"`
	Outcomes      []types.Outcome       `json:"outcomes,omitempty"`
	Difficulties  []types.Difficulty    `json:"difficulties,omitempty"`
	FilePatterns  []string              `json:"file_patterns,omitempty"`
}

// TimeRange represents a time range filter
type TimeRange struct {
	Start *string `json:"start,omitempty"` // RFC3339 format
	End   *string `json:"end,omitempty"`   // RFC3339 format
}

// BatchOperation represents a batch operation for multiple chunks
type BatchOperation struct {
	Operation string                  `json:"operation"` // "store", "update", "delete"
	Chunks    []types.ConversationChunk `json:"chunks,omitempty"`
	IDs       []string                `json:"ids,omitempty"` // For delete operations
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	Success     int      `json:"success"`
	Failed      int      `json:"failed"`
	Errors      []string `json:"errors,omitempty"`
	ProcessedIDs []string `json:"processed_ids,omitempty"`
}

// StorageMetrics represents metrics for monitoring storage performance
type StorageMetrics struct {
	OperationCounts  map[string]int64 `json:"operation_counts"`
	AverageLatency   map[string]float64 `json:"average_latency_ms"`
	ErrorCounts      map[string]int64 `json:"error_counts"`
	LastOperation    *string          `json:"last_operation,omitempty"`
	ConnectionStatus string           `json:"connection_status"`
}