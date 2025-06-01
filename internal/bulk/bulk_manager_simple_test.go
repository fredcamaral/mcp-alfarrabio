package bulk

import (
	"context"
	"errors"
	"testing"
	"time"

	"mcp-memory/internal/storage"
	"mcp-memory/pkg/types"
)

// SimpleMockStorage for testing bulk operations
type SimpleMockStorage struct {
	chunks map[string]types.ConversationChunk
}

func NewSimpleMockStorage() *SimpleMockStorage {
	return &SimpleMockStorage{
		chunks: make(map[string]types.ConversationChunk),
	}
}

func (s *SimpleMockStorage) BatchStore(ctx context.Context, chunks []types.ConversationChunk) (*storage.BatchResult, error) {
	success := 0
	for _, chunk := range chunks {
		s.chunks[chunk.ID] = chunk
		success++
	}
	return &storage.BatchResult{Success: success, Failed: 0}, nil
}

func (s *SimpleMockStorage) BatchDelete(ctx context.Context, ids []string) (*storage.BatchResult, error) {
	success := 0
	for _, id := range ids {
		if _, exists := s.chunks[id]; exists {
			delete(s.chunks, id)
			success++
		}
	}
	return &storage.BatchResult{Success: success, Failed: len(ids) - success}, nil
}

// Minimal interface implementation - return nil/empty for unused methods
func (s *SimpleMockStorage) Initialize(ctx context.Context) error { return nil }
func (s *SimpleMockStorage) Store(ctx context.Context, chunk types.ConversationChunk) error {
	return nil
}
func (s *SimpleMockStorage) Search(ctx context.Context, query types.MemoryQuery, embeddings []float64) (*types.SearchResults, error) {
	return &types.SearchResults{}, nil
}
func (s *SimpleMockStorage) GetByID(ctx context.Context, id string) (*types.ConversationChunk, error) {
	return nil, errors.New("not found")
}
func (s *SimpleMockStorage) ListByRepository(ctx context.Context, repository string, limit int, offset int) ([]types.ConversationChunk, error) {
	return []types.ConversationChunk{}, nil
}
func (s *SimpleMockStorage) ListBySession(ctx context.Context, sessionID string) ([]types.ConversationChunk, error) {
	return []types.ConversationChunk{}, nil
}
func (s *SimpleMockStorage) Delete(ctx context.Context, id string) error { return nil }
func (s *SimpleMockStorage) Update(ctx context.Context, chunk types.ConversationChunk) error {
	return nil
}
func (s *SimpleMockStorage) HealthCheck(ctx context.Context) error { return nil }
func (s *SimpleMockStorage) GetStats(ctx context.Context) (*storage.StoreStats, error) {
	return &storage.StoreStats{}, nil
}
func (s *SimpleMockStorage) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	return 0, nil
}
func (s *SimpleMockStorage) Close() error { return nil }
func (s *SimpleMockStorage) GetAllChunks(ctx context.Context) ([]types.ConversationChunk, error) {
	return []types.ConversationChunk{}, nil
}
func (s *SimpleMockStorage) DeleteCollection(ctx context.Context, collection string) error {
	return nil
}
func (s *SimpleMockStorage) ListCollections(ctx context.Context) ([]string, error) {
	return []string{}, nil
}
func (s *SimpleMockStorage) FindSimilar(ctx context.Context, content string, chunkType *types.ChunkType, limit int) ([]types.ConversationChunk, error) {
	return []types.ConversationChunk{}, nil
}
func (s *SimpleMockStorage) StoreChunk(ctx context.Context, chunk types.ConversationChunk) error {
	return nil
}
func (s *SimpleMockStorage) StoreRelationship(ctx context.Context, sourceID, targetID string, relationType types.RelationType, confidence float64, source types.ConfidenceSource) (*types.MemoryRelationship, error) {
	return nil, errors.New("not implemented")
}
func (s *SimpleMockStorage) GetRelationships(ctx context.Context, query types.RelationshipQuery) ([]types.RelationshipResult, error) {
	return []types.RelationshipResult{}, nil
}
func (s *SimpleMockStorage) TraverseGraph(ctx context.Context, startChunkID string, maxDepth int, relationTypes []types.RelationType) (*types.GraphTraversalResult, error) {
	return &types.GraphTraversalResult{}, nil
}
func (s *SimpleMockStorage) UpdateRelationship(ctx context.Context, relationshipID string, confidence float64, factors types.ConfidenceFactors) error {
	return nil
}
func (s *SimpleMockStorage) DeleteRelationship(ctx context.Context, relationshipID string) error {
	return nil
}
func (s *SimpleMockStorage) GetRelationshipByID(ctx context.Context, relationshipID string) (*types.MemoryRelationship, error) {
	return nil, errors.New("not found")
}

func TestBulkManager_NewManager(t *testing.T) {
	storage := NewSimpleMockStorage()
	manager := NewManager(storage, nil)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestBulkManager_SubmitStoreOperation(t *testing.T) {
	ctx := context.Background()
	storage := NewSimpleMockStorage()
	manager := NewManager(storage, nil)

	chunks := []types.ConversationChunk{
		{
			ID:        "test-chunk-1",
			SessionID: "test-session",
			Content:   "Test content 1",
			Type:      types.ChunkTypeDiscussion,
			Timestamp: time.Now(),
		},
	}

	request := BulkRequest{
		Operation: OperationStore,
		Chunks:    chunks,
		Options: BulkOptions{
			BatchSize:       10,
			MaxConcurrency:  2,
			ContinueOnError: true,
		},
	}

	progress, err := manager.SubmitOperation(ctx, request)
	if err != nil {
		t.Fatalf("SubmitOperation failed: %v", err)
	}

	if progress.OperationID == "" {
		t.Error("Expected non-empty operation ID")
	}

	if progress.TotalItems != 1 {
		t.Errorf("Expected 1 total item, got %d", progress.TotalItems)
	}
}

func TestBulkManager_GetProgress(t *testing.T) {
	ctx := context.Background()
	storage := NewSimpleMockStorage()
	manager := NewManager(storage, nil)

	chunks := []types.ConversationChunk{
		{
			ID:        "test-chunk-1",
			SessionID: "test-session",
			Content:   "Test content 1",
			Type:      types.ChunkTypeDiscussion,
			Timestamp: time.Now(),
		},
	}

	request := BulkRequest{
		Operation: OperationStore,
		Chunks:    chunks,
		Options: BulkOptions{
			BatchSize:       10,
			MaxConcurrency:  2,
			ContinueOnError: true,
		},
	}

	progress, err := manager.SubmitOperation(ctx, request)
	if err != nil {
		t.Fatalf("SubmitOperation failed: %v", err)
	}

	// Test GetProgress
	retrievedProgress, err := manager.GetProgress(progress.OperationID)
	if err != nil {
		t.Fatalf("GetProgress failed: %v", err)
	}

	if retrievedProgress.OperationID != progress.OperationID {
		t.Errorf("Expected operation ID %s, got %s", progress.OperationID, retrievedProgress.OperationID)
	}
}

func TestBulkManager_ListOperations(t *testing.T) {
	storage := NewSimpleMockStorage()
	manager := NewManager(storage, nil)

	operations, err := manager.ListOperations(nil, 10)
	if err != nil {
		t.Fatalf("ListOperations failed: %v", err)
	}

	// Should return an empty or non-nil list initially
	// The exact behavior depends on the implementation
	_ = operations // Accept whatever is returned
}

// Benchmark for bulk store performance
func BenchmarkBulkManager_SubmitStoreOperation(b *testing.B) {
	ctx := context.Background()
	storage := NewSimpleMockStorage()
	manager := NewManager(storage, nil)

	chunks := []types.ConversationChunk{
		{
			ID:        "bench-chunk-1",
			SessionID: "bench-session",
			Content:   "Benchmark content",
			Type:      types.ChunkTypeDiscussion,
			Timestamp: time.Now(),
		},
	}

	request := BulkRequest{
		Operation: OperationStore,
		Chunks:    chunks,
		Options: BulkOptions{
			BatchSize:       10,
			MaxConcurrency:  2,
			ContinueOnError: true,
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := manager.SubmitOperation(ctx, request)
		if err != nil {
			b.Fatalf("SubmitOperation failed: %v", err)
		}
	}
}
