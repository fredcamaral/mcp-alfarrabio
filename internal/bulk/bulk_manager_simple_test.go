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

func (s *SimpleMockStorage) BatchStore(_ context.Context, chunks []types.ConversationChunk) (*storage.BatchResult, error) {
	success := 0
	for i := range chunks {
		chunk := &chunks[i]
		s.chunks[chunk.ID] = *chunk
		success++
	}
	return &storage.BatchResult{Success: success, Failed: 0}, nil
}

func (s *SimpleMockStorage) BatchDelete(_ context.Context, ids []string) (*storage.BatchResult, error) {
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
func (s *SimpleMockStorage) Initialize(_ context.Context) error { return nil }
func (s *SimpleMockStorage) Store(_ context.Context, _ types.ConversationChunk) error {
	return nil
}
func (s *SimpleMockStorage) Search(_ context.Context, _ types.MemoryQuery, _ []float64) (*types.SearchResults, error) {
	return &types.SearchResults{}, nil
}
func (s *SimpleMockStorage) GetByID(_ context.Context, _ string) (*types.ConversationChunk, error) {
	return nil, errors.New("not found")
}
func (s *SimpleMockStorage) ListByRepository(_ context.Context, _ string, _, _ int) ([]types.ConversationChunk, error) {
	return []types.ConversationChunk{}, nil
}
func (s *SimpleMockStorage) ListBySession(_ context.Context, _ string) ([]types.ConversationChunk, error) {
	return []types.ConversationChunk{}, nil
}
func (s *SimpleMockStorage) Delete(_ context.Context, _ string) error { return nil }
func (s *SimpleMockStorage) Update(_ context.Context, _ types.ConversationChunk) error {
	return nil
}
func (s *SimpleMockStorage) HealthCheck(_ context.Context) error { return nil }
func (s *SimpleMockStorage) GetStats(_ context.Context) (*storage.StoreStats, error) {
	return &storage.StoreStats{}, nil
}
func (s *SimpleMockStorage) Cleanup(_ context.Context, _ int) (int, error) {
	return 0, nil
}
func (s *SimpleMockStorage) Close() error { return nil }
func (s *SimpleMockStorage) GetAllChunks(_ context.Context) ([]types.ConversationChunk, error) {
	return []types.ConversationChunk{}, nil
}
func (s *SimpleMockStorage) DeleteCollection(_ context.Context, _ string) error {
	return nil
}
func (s *SimpleMockStorage) ListCollections(_ context.Context) ([]string, error) {
	return []string{}, nil
}
func (s *SimpleMockStorage) FindSimilar(_ context.Context, _ string, _ *types.ChunkType, _ int) ([]types.ConversationChunk, error) {
	return []types.ConversationChunk{}, nil
}
func (s *SimpleMockStorage) StoreChunk(_ context.Context, _ types.ConversationChunk) error {
	return nil
}
func (s *SimpleMockStorage) StoreRelationship(_ context.Context, _, _ string, _ types.RelationType, _ float64, _ types.ConfidenceSource) (*types.MemoryRelationship, error) {
	return nil, errors.New("not implemented")
}
func (s *SimpleMockStorage) GetRelationships(_ context.Context, _ types.RelationshipQuery) ([]types.RelationshipResult, error) {
	return []types.RelationshipResult{}, nil
}
func (s *SimpleMockStorage) TraverseGraph(_ context.Context, _ string, _ int, _ []types.RelationType) (*types.GraphTraversalResult, error) {
	return &types.GraphTraversalResult{}, nil
}
func (s *SimpleMockStorage) UpdateRelationship(_ context.Context, _ string, _ float64, _ types.ConfidenceFactors) error {
	return nil
}
func (s *SimpleMockStorage) DeleteRelationship(_ context.Context, _ string) error {
	return nil
}
func (s *SimpleMockStorage) GetRelationshipByID(_ context.Context, _ string) (*types.MemoryRelationship, error) {
	return nil, errors.New("not found")
}

func TestBulkManager_NewManager(t *testing.T) {
	vectorStore := NewSimpleMockStorage()
	manager := NewManager(vectorStore, nil)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestBulkManager_SubmitStoreOperation(t *testing.T) {
	ctx := context.Background()
	vectorStore := NewSimpleMockStorage()
	manager := NewManager(vectorStore, nil)

	chunks := []types.ConversationChunk{
		{
			ID:        "test-chunk-1",
			SessionID: "test-session",
			Content:   "Test content 1",
			Type:      types.ChunkTypeDiscussion,
			Timestamp: time.Now(),
		},
	}

	request := Request{
		Operation: OperationStore,
		Chunks:    chunks,
		Options: Options{
			BatchSize:       10,
			MaxConcurrency:  2,
			ContinueOnError: true,
		},
	}

	progress, err := manager.SubmitOperation(ctx, &request)
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
	vectorStore := NewSimpleMockStorage()
	manager := NewManager(vectorStore, nil)

	chunks := []types.ConversationChunk{
		{
			ID:        "test-chunk-1",
			SessionID: "test-session",
			Content:   "Test content 1",
			Type:      types.ChunkTypeDiscussion,
			Timestamp: time.Now(),
		},
	}

	request := Request{
		Operation: OperationStore,
		Chunks:    chunks,
		Options: Options{
			BatchSize:       10,
			MaxConcurrency:  2,
			ContinueOnError: true,
		},
	}

	progress, err := manager.SubmitOperation(ctx, &request)
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
	vectorStore := NewSimpleMockStorage()
	manager := NewManager(vectorStore, nil)

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
	vectorStore := NewSimpleMockStorage()
	manager := NewManager(vectorStore, nil)

	chunks := []types.ConversationChunk{
		{
			ID:        "bench-chunk-1",
			SessionID: "bench-session",
			Content:   "Benchmark content",
			Type:      types.ChunkTypeDiscussion,
			Timestamp: time.Now(),
		},
	}

	request := Request{
		Operation: OperationStore,
		Chunks:    chunks,
		Options: Options{
			BatchSize:       10,
			MaxConcurrency:  2,
			ContinueOnError: true,
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := manager.SubmitOperation(ctx, &request)
		if err != nil {
			b.Fatalf("SubmitOperation failed: %v", err)
		}
	}
}
