package storage

import (
	"context"
	"errors"
	"mcp-memory/internal/circuitbreaker"
	"mcp-memory/pkg/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockVectorStore for testing circuit breaker wrapper
type MockVectorStore struct {
	mock.Mock
}

func (m *MockVectorStore) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockVectorStore) Store(ctx context.Context, chunk *types.ConversationChunk) error {
	args := m.Called(ctx, chunk)
	return args.Error(0)
}

func (m *MockVectorStore) Search(ctx context.Context, query *types.MemoryQuery, embeddings []float64) (*types.SearchResults, error) {
	args := m.Called(ctx, query, embeddings)
	return args.Get(0).(*types.SearchResults), args.Error(1)
}

func (m *MockVectorStore) GetByID(ctx context.Context, id string) (*types.ConversationChunk, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.ConversationChunk), args.Error(1)
}

func (m *MockVectorStore) ListByRepository(ctx context.Context, repository string, limit, offset int) ([]types.ConversationChunk, error) {
	args := m.Called(ctx, repository, limit, offset)
	return args.Get(0).([]types.ConversationChunk), args.Error(1)
}

func (m *MockVectorStore) ListBySession(ctx context.Context, sessionID string) ([]types.ConversationChunk, error) {
	args := m.Called(ctx, sessionID)
	return args.Get(0).([]types.ConversationChunk), args.Error(1)
}

func (m *MockVectorStore) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockVectorStore) Update(ctx context.Context, chunk *types.ConversationChunk) error {
	args := m.Called(ctx, chunk)
	return args.Error(0)
}

func (m *MockVectorStore) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockVectorStore) GetStats(ctx context.Context) (*StoreStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(*StoreStats), args.Error(1)
}

func (m *MockVectorStore) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	args := m.Called(ctx, retentionDays)
	return args.Int(0), args.Error(1)
}

func (m *MockVectorStore) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockVectorStore) GetAllChunks(ctx context.Context) ([]types.ConversationChunk, error) {
	args := m.Called(ctx)
	return args.Get(0).([]types.ConversationChunk), args.Error(1)
}

func (m *MockVectorStore) DeleteCollection(ctx context.Context, collection string) error {
	args := m.Called(ctx, collection)
	return args.Error(0)
}

func (m *MockVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockVectorStore) FindSimilar(ctx context.Context, content string, chunkType *types.ChunkType, limit int) ([]types.ConversationChunk, error) {
	args := m.Called(ctx, content, chunkType, limit)
	return args.Get(0).([]types.ConversationChunk), args.Error(1)
}

func (m *MockVectorStore) StoreChunk(ctx context.Context, chunk *types.ConversationChunk) error {
	args := m.Called(ctx, chunk)
	return args.Error(0)
}

func (m *MockVectorStore) BatchStore(ctx context.Context, chunks []*types.ConversationChunk) (*BatchResult, error) {
	args := m.Called(ctx, chunks)
	return args.Get(0).(*BatchResult), args.Error(1)
}

func (m *MockVectorStore) BatchDelete(ctx context.Context, ids []string) (*BatchResult, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(*BatchResult), args.Error(1)
}

// Relationship management methods
func (m *MockVectorStore) StoreRelationship(ctx context.Context, sourceID, targetID string, relationType types.RelationType, confidence float64, source types.ConfidenceSource) (*types.MemoryRelationship, error) {
	args := m.Called(ctx, sourceID, targetID, relationType, confidence, source)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.MemoryRelationship), args.Error(1)
}

func (m *MockVectorStore) GetRelationships(ctx context.Context, query *types.RelationshipQuery) ([]types.RelationshipResult, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]types.RelationshipResult), args.Error(1)
}

func (m *MockVectorStore) TraverseGraph(ctx context.Context, startChunkID string, maxDepth int, relationTypes []types.RelationType) (*types.GraphTraversalResult, error) {
	args := m.Called(ctx, startChunkID, maxDepth, relationTypes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.GraphTraversalResult), args.Error(1)
}

func (m *MockVectorStore) UpdateRelationship(ctx context.Context, relationshipID string, confidence float64, factors types.ConfidenceFactors) error {
	args := m.Called(ctx, relationshipID, confidence, factors)
	return args.Error(0)
}

func (m *MockVectorStore) DeleteRelationship(ctx context.Context, relationshipID string) error {
	args := m.Called(ctx, relationshipID)
	return args.Error(0)
}

func (m *MockVectorStore) GetRelationshipByID(ctx context.Context, relationshipID string) (*types.MemoryRelationship, error) {
	args := m.Called(ctx, relationshipID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.MemoryRelationship), args.Error(1)
}

func TestCircuitBreakerWrapper_SuccessfulOperations(t *testing.T) {
	mockStore := new(MockVectorStore)

	config := &circuitbreaker.Config{
		FailureThreshold: 3,
		Timeout:          time.Second * 5,
		SuccessThreshold: 2,
	}

	wrapper := NewCircuitBreakerVectorStore(mockStore, config)
	ctx := context.Background()

	t.Run("Initialize success", func(t *testing.T) {
		mockStore.On("Initialize", ctx).Return(nil).Once()

		err := wrapper.Initialize(ctx)
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("Store success", func(t *testing.T) {
		chunk := types.ConversationChunk{
			ID:         "test-chunk",
			SessionID:  "session",
			Type:       types.ChunkTypeProblem,
			Content:    "Test content",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2},
			Metadata: types.ChunkMetadata{
				Repository: "test-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		}

		mockStore.On("Store", ctx, chunk).Return(nil).Once()

		err := wrapper.Store(ctx, &chunk)
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("GetByID success", func(t *testing.T) {
		expectedChunk := &types.ConversationChunk{
			ID:      "test-id",
			Content: "Test content",
		}

		mockStore.On("GetByID", ctx, "test-id").Return(expectedChunk, nil).Once()

		chunk, err := wrapper.GetByID(ctx, "test-id")
		assert.NoError(t, err)
		assert.Equal(t, expectedChunk, chunk)
		mockStore.AssertExpectations(t)
	})

	t.Run("Search success", func(t *testing.T) {
		query := types.MemoryQuery{Query: "test"}
		embeddings := []float64{0.1, 0.2}
		expectedResults := &types.SearchResults{
			Results: []types.SearchResult{},
			Total:   0,
		}

		mockStore.On("Search", ctx, query, embeddings).Return(expectedResults, nil).Once()

		results, err := wrapper.Search(ctx, &query, embeddings)
		assert.NoError(t, err)
		assert.Equal(t, expectedResults, results)
		mockStore.AssertExpectations(t)
	})
}

func TestCircuitBreakerWrapper_FailureHandling(t *testing.T) {
	mockStore := new(MockVectorStore)

	config := &circuitbreaker.Config{
		FailureThreshold: 2, // Low threshold for testing
		Timeout:          time.Millisecond * 100,
		SuccessThreshold: 1,
	}

	wrapper := NewCircuitBreakerVectorStore(mockStore, config)
	ctx := context.Background()

	t.Run("Store failures trigger circuit breaker", func(t *testing.T) {
		chunk := types.ConversationChunk{
			ID:         "failing-chunk",
			SessionID:  "session",
			Type:       types.ChunkTypeProblem,
			Content:    "Test content",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2},
			Metadata: types.ChunkMetadata{
				Repository: "test-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		}

		// Mock failures
		testError := errors.New("store operation failed")
		mockStore.On("Store", ctx, chunk).Return(testError).Times(2)

		// First failure
		err := wrapper.Store(ctx, &chunk)
		assert.Error(t, err)
		assert.Equal(t, testError, err)

		// Second failure - should trigger circuit breaker
		err = wrapper.Store(ctx, &chunk)
		assert.Error(t, err)
		assert.Equal(t, testError, err)

		// Third attempt - circuit breaker should be open
		err = wrapper.Store(ctx, &chunk)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker is open")

		mockStore.AssertExpectations(t)
	})
}

func TestCircuitBreakerWrapper_AllMethods(t *testing.T) {
	mockStore := new(MockVectorStore)

	config := &circuitbreaker.Config{
		FailureThreshold: 5,
		Timeout:          time.Second * 5,
		SuccessThreshold: 2,
	}

	wrapper := NewCircuitBreakerVectorStore(mockStore, config)
	ctx := context.Background()

	t.Run("ListByRepository", func(t *testing.T) {
		expectedChunks := []types.ConversationChunk{
			{ID: "chunk1", Content: "Content 1"},
			{ID: "chunk2", Content: "Content 2"},
		}

		mockStore.On("ListByRepository", ctx, "test-repo", 10, 0).Return(expectedChunks, nil).Once()

		chunks, err := wrapper.ListByRepository(ctx, "test-repo", 10, 0)
		assert.NoError(t, err)
		assert.Equal(t, expectedChunks, chunks)
		mockStore.AssertExpectations(t)
	})

	t.Run("ListBySession", func(t *testing.T) {
		expectedChunks := []types.ConversationChunk{
			{ID: "chunk1", SessionID: "session1"},
		}

		mockStore.On("ListBySession", ctx, "session1").Return(expectedChunks, nil).Once()

		chunks, err := wrapper.ListBySession(ctx, "session1")
		assert.NoError(t, err)
		assert.Equal(t, expectedChunks, chunks)
		mockStore.AssertExpectations(t)
	})

	t.Run("Delete", func(t *testing.T) {
		mockStore.On("Delete", ctx, "test-id").Return(nil).Once()

		err := wrapper.Delete(ctx, "test-id")
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("Update", func(t *testing.T) {
		chunk := types.ConversationChunk{ID: "update-test", Content: "Updated content"}
		mockStore.On("Update", ctx, chunk).Return(nil).Once()

		err := wrapper.Update(ctx, &chunk)
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("HealthCheck", func(t *testing.T) {
		mockStore.On("HealthCheck", ctx).Return(nil).Once()

		err := wrapper.HealthCheck(ctx)
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("GetStats", func(t *testing.T) {
		expectedStats := &StoreStats{
			TotalChunks: 100,
			ChunksByType: map[string]int64{
				"problem": 50,
			},
		}

		mockStore.On("GetStats", ctx).Return(expectedStats, nil).Once()

		stats, err := wrapper.GetStats(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedStats, stats)
		mockStore.AssertExpectations(t)
	})

	t.Run("Cleanup", func(t *testing.T) {
		mockStore.On("Cleanup", ctx, 30).Return(5, nil).Once()

		deleted, err := wrapper.Cleanup(ctx, 30)
		assert.NoError(t, err)
		assert.Equal(t, 5, deleted)
		mockStore.AssertExpectations(t)
	})

	t.Run("Close", func(t *testing.T) {
		mockStore.On("Close").Return(nil).Once()

		err := wrapper.Close()
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("GetAllChunks", func(t *testing.T) {
		expectedChunks := []types.ConversationChunk{
			{ID: "all1"}, {ID: "all2"},
		}

		mockStore.On("GetAllChunks", ctx).Return(expectedChunks, nil).Once()

		chunks, err := wrapper.GetAllChunks(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedChunks, chunks)
		mockStore.AssertExpectations(t)
	})

	t.Run("DeleteCollection", func(t *testing.T) {
		mockStore.On("DeleteCollection", ctx, "test-collection").Return(nil).Once()

		err := wrapper.DeleteCollection(ctx, "test-collection")
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("ListCollections", func(t *testing.T) {
		expectedCollections := []string{"collection1", "collection2"}

		mockStore.On("ListCollections", ctx).Return(expectedCollections, nil).Once()

		collections, err := wrapper.ListCollections(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedCollections, collections)
		mockStore.AssertExpectations(t)
	})

	t.Run("FindSimilar", func(t *testing.T) {
		chunkType := types.ChunkTypeProblem
		expectedChunks := []types.ConversationChunk{
			{ID: "similar1", Content: "Similar content"},
		}

		mockStore.On("FindSimilar", ctx, "test content", &chunkType, 5).Return(expectedChunks, nil).Once()

		chunks, err := wrapper.FindSimilar(ctx, "test content", &chunkType, 5)
		assert.NoError(t, err)
		assert.Equal(t, expectedChunks, chunks)
		mockStore.AssertExpectations(t)
	})

	t.Run("StoreChunk", func(t *testing.T) {
		chunk := types.ConversationChunk{ID: "store-chunk-test"}
		mockStore.On("StoreChunk", ctx, chunk).Return(nil).Once()

		err := wrapper.StoreChunk(ctx, &chunk)
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("BatchStore", func(t *testing.T) {
		chunks := []*types.ConversationChunk{
			{ID: "batch1"}, {ID: "batch2"},
		}
		expectedResult := &BatchResult{
			Success:      2,
			Failed:       0,
			ProcessedIDs: []string{"batch1", "batch2"},
		}

		mockStore.On("BatchStore", ctx, chunks).Return(expectedResult, nil).Once()

		result, err := wrapper.BatchStore(ctx, chunks)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("BatchDelete", func(t *testing.T) {
		ids := []string{"delete1", "delete2"}
		expectedResult := &BatchResult{
			Success:      2,
			Failed:       0,
			ProcessedIDs: ids,
		}

		mockStore.On("BatchDelete", ctx, ids).Return(expectedResult, nil).Once()

		result, err := wrapper.BatchDelete(ctx, ids)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		mockStore.AssertExpectations(t)
	})
}

func TestCircuitBreakerWrapper_ErrorPropagation(t *testing.T) {
	mockStore := new(MockVectorStore)

	config := &circuitbreaker.Config{
		FailureThreshold: 10, // High threshold to avoid triggering during test
		Timeout:          time.Second * 5,
		SuccessThreshold: 2,
	}

	wrapper := NewCircuitBreakerVectorStore(mockStore, config)
	ctx := context.Background()

	t.Run("Errors are properly propagated", func(t *testing.T) {
		testError := errors.New("specific test error")
		mockStore.On("HealthCheck", ctx).Return(testError).Once()

		err := wrapper.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Equal(t, testError, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("Nil returns are handled correctly", func(t *testing.T) {
		mockStore.On("GetByID", ctx, "non-existent").Return(nil, errors.New("not found")).Once()

		chunk, err := wrapper.GetByID(ctx, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, chunk)
		mockStore.AssertExpectations(t)
	})
}
