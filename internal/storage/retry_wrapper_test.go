package storage

import (
	"context"
	"errors"
	"lerian-mcp-memory/internal/retry"
	"lerian-mcp-memory/pkg/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRetryWrapper_SuccessfulOperations(t *testing.T) {
	mockStore := new(MockVectorStore)

	retryConfig := &retry.Config{
		MaxAttempts:  3,
		InitialDelay: time.Millisecond * 10,
		MaxDelay:     time.Millisecond * 100,
		Multiplier:   2.0,
	}

	wrapper := NewRetryableVectorStore(mockStore, retryConfig)
	ctx := context.Background()

	t.Run("Initialize success on first try", func(t *testing.T) {
		mockStore.On("Initialize", ctx).Return(nil).Once()

		err := wrapper.Initialize(ctx)
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("Store success on first try", func(t *testing.T) {
		chunk := types.ConversationChunk{
			ID:         "retry-test",
			SessionID:  "session",
			Type:       types.ChunkTypeProblem,
			Content:    "Retry test content",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2},
			Metadata: types.ChunkMetadata{
				Repository: "retry-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		}

		mockStore.On("Store", ctx, &chunk).Return(nil).Once()

		err := wrapper.Store(ctx, &chunk)
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})
}

func TestRetryWrapper_RetryLogic(t *testing.T) {
	mockStore := new(MockVectorStore)

	retryConfig := &retry.Config{
		MaxAttempts:  2,
		InitialDelay: time.Millisecond * 1, // Very short for testing
		MaxDelay:     time.Millisecond * 10,
		Multiplier:   2.0,
	}

	wrapper := NewRetryableVectorStore(mockStore, retryConfig)
	ctx := context.Background()

	t.Run("Store succeeds after retry", func(t *testing.T) {
		chunk := types.ConversationChunk{
			ID:         "retry-success",
			SessionID:  "session",
			Type:       types.ChunkTypeProblem,
			Content:    "Retry success test",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2},
			Metadata: types.ChunkMetadata{
				Repository: "retry-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		}

		// First call fails, second succeeds
		mockStore.On("Store", ctx, &chunk).Return(errors.New("temporary failure")).Once()
		mockStore.On("Store", ctx, &chunk).Return(nil).Once()

		err := wrapper.Store(ctx, &chunk)
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("Store fails after max retries", func(t *testing.T) {
		chunk := types.ConversationChunk{
			ID:         "retry-fail",
			SessionID:  "session",
			Type:       types.ChunkTypeProblem,
			Content:    "Retry fail test",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2},
			Metadata: types.ChunkMetadata{
				Repository: "retry-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		}

		persistentError := errors.New("timeout")
		// Fail on initial call + retry attempts (MaxAttempts=2) = 2 total calls
		mockStore.On("Store", ctx, &chunk).Return(persistentError).Times(2)

		err := wrapper.Store(ctx, &chunk)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
		mockStore.AssertExpectations(t)
	})
}

func TestRetryWrapper_Search(t *testing.T) {
	mockStore := new(MockVectorStore)

	retryConfig := &retry.Config{
		MaxAttempts:  2,
		InitialDelay: time.Millisecond * 1,
		MaxDelay:     time.Millisecond * 10,
		Multiplier:   2.0,
	}

	wrapper := NewRetryableVectorStore(mockStore, retryConfig)
	ctx := context.Background()

	t.Run("Search success with retries", func(t *testing.T) {
		query := types.MemoryQuery{Query: "retry search test"}
		embeddings := []float64{0.1, 0.2, 0.3}
		expectedResults := &types.SearchResults{
			Results: []types.SearchResult{
				{
					Chunk: types.ConversationChunk{ID: "result1"},
					Score: 0.9,
				},
			},
			Total: 1,
		}

		// Fail first, succeed second
		mockStore.On("Search", ctx, mock.AnythingOfType("*types.MemoryQuery"), embeddings).Return((*types.SearchResults)(nil), errors.New("search failed")).Once()
		mockStore.On("Search", ctx, mock.AnythingOfType("*types.MemoryQuery"), embeddings).Return(expectedResults, nil).Once()

		results, err := wrapper.Search(ctx, &query, embeddings)
		assert.NoError(t, err)
		assert.Equal(t, expectedResults, results)
		mockStore.AssertExpectations(t)
	})
}

func TestRetryWrapper_AllMethods(t *testing.T) {
	retryConfig := &retry.Config{
		MaxAttempts:  2,
		InitialDelay: time.Millisecond * 1,
		MaxDelay:     time.Millisecond * 5,
		Multiplier:   2.0,
	}

	ctx := context.Background()

	t.Run("GetByID with retry", func(t *testing.T) {
		mockStore := new(MockVectorStore)
		wrapper := NewRetryableVectorStore(mockStore, retryConfig)
		expectedChunk := &types.ConversationChunk{
			ID:      "retry-get-test",
			Content: "Retrieved content",
		}

		// Fail once, then succeed
		mockStore.On("GetByID", ctx, "retry-get-test").Return((*types.ConversationChunk)(nil), errors.New("get failed")).Once()
		mockStore.On("GetByID", ctx, "retry-get-test").Return(expectedChunk, nil).Once()

		chunk, err := wrapper.GetByID(ctx, "retry-get-test")
		assert.NoError(t, err)
		assert.Equal(t, expectedChunk, chunk)
		mockStore.AssertExpectations(t)
	})

	t.Run("ListByRepository with retry", func(t *testing.T) {
		mockStore := new(MockVectorStore)
		wrapper := NewRetryableVectorStore(mockStore, retryConfig)
		expectedChunks := []types.ConversationChunk{
			{ID: "repo-chunk1"}, {ID: "repo-chunk2"},
		}

		mockStore.On("ListByRepository", ctx, "retry-repo", 10, 0).Return([]types.ConversationChunk(nil), errors.New("list failed")).Once()
		mockStore.On("ListByRepository", ctx, "retry-repo", 10, 0).Return(expectedChunks, nil).Once()

		chunks, err := wrapper.ListByRepository(ctx, "retry-repo", 10, 0)
		assert.NoError(t, err)
		assert.Equal(t, expectedChunks, chunks)
		mockStore.AssertExpectations(t)
	})

	t.Run("ListBySession with retry", func(t *testing.T) {
		mockStore := new(MockVectorStore)
		wrapper := NewRetryableVectorStore(mockStore, retryConfig)
		expectedChunks := []types.ConversationChunk{
			{ID: "session-chunk1", SessionID: "retry-session"},
		}

		mockStore.On("ListBySession", ctx, "retry-session").Return([]types.ConversationChunk(nil), errors.New("session list failed")).Once()
		mockStore.On("ListBySession", ctx, "retry-session").Return(expectedChunks, nil).Once()

		chunks, err := wrapper.ListBySession(ctx, "retry-session")
		assert.NoError(t, err)
		assert.Equal(t, expectedChunks, chunks)
		mockStore.AssertExpectations(t)
	})

	t.Run("Delete with retry", func(t *testing.T) {
		mockStore := new(MockVectorStore)
		wrapper := NewRetryableVectorStore(mockStore, retryConfig)
		mockStore.On("Delete", ctx, "retry-delete").Return(errors.New("delete failed")).Once()
		mockStore.On("Delete", ctx, "retry-delete").Return(nil).Once()

		err := wrapper.Delete(ctx, "retry-delete")
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("Update with retry", func(t *testing.T) {
		mockStore := new(MockVectorStore)
		wrapper := NewRetryableVectorStore(mockStore, retryConfig)
		chunk := types.ConversationChunk{
			ID:      "retry-update",
			Content: "Updated content",
		}

		mockStore.On("Update", ctx, &chunk).Return(errors.New("update failed")).Once()
		mockStore.On("Update", ctx, &chunk).Return(nil).Once()

		err := wrapper.Update(ctx, &chunk)
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("HealthCheck with retry", func(t *testing.T) {
		mockStore := new(MockVectorStore)
		wrapper := NewRetryableVectorStore(mockStore, retryConfig)
		mockStore.On("HealthCheck", ctx).Return(errors.New("timeout")).Once()
		mockStore.On("HealthCheck", ctx).Return(nil).Once()

		err := wrapper.HealthCheck(ctx)
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("GetStats with retry", func(t *testing.T) {
		mockStore := new(MockVectorStore)
		wrapper := NewRetryableVectorStore(mockStore, retryConfig)
		expectedStats := &StoreStats{
			TotalChunks: 50,
			ChunksByType: map[string]int64{
				"problem":  30,
				"solution": 20,
			},
		}

		mockStore.On("GetStats", ctx).Return((*StoreStats)(nil), errors.New("stats failed")).Once()
		mockStore.On("GetStats", ctx).Return(expectedStats, nil).Once()

		stats, err := wrapper.GetStats(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedStats, stats)
		mockStore.AssertExpectations(t)
	})

	t.Run("Cleanup with retry", func(t *testing.T) {
		mockStore := new(MockVectorStore)
		wrapper := NewRetryableVectorStore(mockStore, retryConfig)
		mockStore.On("Cleanup", ctx, 30).Return(0, errors.New("timeout")).Once()
		mockStore.On("Cleanup", ctx, 30).Return(10, nil).Once()

		deleted, err := wrapper.Cleanup(ctx, 30)
		assert.NoError(t, err)
		assert.Equal(t, 10, deleted)
		mockStore.AssertExpectations(t)
	})

	t.Run("Close without retry", func(t *testing.T) {
		mockStore := new(MockVectorStore)
		wrapper := NewRetryableVectorStore(mockStore, retryConfig)
		mockStore.On("Close").Return(nil).Once()

		err := wrapper.Close()
		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("BatchStore with retry", func(t *testing.T) {
		mockStore := new(MockVectorStore)
		wrapper := NewRetryableVectorStore(mockStore, retryConfig)
		chunks := []*types.ConversationChunk{
			{ID: "batch-retry1"}, {ID: "batch-retry2"},
		}
		expectedResult := &BatchResult{
			Success:      2,
			Failed:       0,
			ProcessedIDs: []string{"batch-retry1", "batch-retry2"},
		}

		mockStore.On("BatchStore", ctx, chunks).Return((*BatchResult)(nil), errors.New("batch store failed")).Once()
		mockStore.On("BatchStore", ctx, chunks).Return(expectedResult, nil).Once()

		result, err := wrapper.BatchStore(ctx, chunks)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		mockStore.AssertExpectations(t)
	})

	t.Run("BatchDelete with retry", func(t *testing.T) {
		mockStore := new(MockVectorStore)
		wrapper := NewRetryableVectorStore(mockStore, retryConfig)
		ids := []string{"retry-delete1", "retry-delete2"}
		expectedResult := &BatchResult{
			Success:      2,
			Failed:       0,
			ProcessedIDs: ids,
		}

		mockStore.On("BatchDelete", ctx, ids).Return((*BatchResult)(nil), errors.New("batch delete failed")).Once()
		mockStore.On("BatchDelete", ctx, ids).Return(expectedResult, nil).Once()

		result, err := wrapper.BatchDelete(ctx, ids)
		assert.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		mockStore.AssertExpectations(t)
	})
}

func TestRetryWrapper_NonRetriableErrors(t *testing.T) {
	mockStore := new(MockVectorStore)

	retryConfig := &retry.Config{
		MaxAttempts:  2,
		InitialDelay: time.Millisecond * 1,
		MaxDelay:     time.Millisecond * 5,
		Multiplier:   2.0,
	}

	wrapper := NewRetryableVectorStore(mockStore, retryConfig)
	ctx := context.Background()

	t.Run("Context cancellation not retried", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		chunk := types.ConversationChunk{
			ID:         "cancel-test",
			SessionID:  "session",
			Type:       types.ChunkTypeProblem,
			Content:    "Cancel test",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2},
			Metadata: types.ChunkMetadata{
				Repository: "cancel-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		}

		// Context is cancelled before Store is called, so no expectations on mockStore

		err := wrapper.Store(cancelCtx, &chunk)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context cancel")
		// Don't assert expectations as Store should not be called
	})
}

func TestRetryWrapper_TimeoutBehavior(t *testing.T) {
	mockStore := new(MockVectorStore)

	retryConfig := &retry.Config{
		MaxAttempts:  2,
		InitialDelay: time.Millisecond * 5,
		MaxDelay:     time.Millisecond * 10,
		Multiplier:   2.0,
	}

	wrapper := NewRetryableVectorStore(mockStore, retryConfig)
	ctx := context.Background()

	t.Run("Retry with exponential backoff", func(t *testing.T) {
		chunk := types.ConversationChunk{
			ID:         "timeout-test",
			SessionID:  "session",
			Type:       types.ChunkTypeProblem,
			Content:    "Timeout test",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2},
			Metadata: types.ChunkMetadata{
				Repository: "timeout-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		}

		start := time.Now()

		// Fail once, then succeed
		mockStore.On("Store", ctx, &chunk).Return(errors.New("timeout failure")).Once()
		mockStore.On("Store", ctx, &chunk).Return(nil).Once()

		err := wrapper.Store(ctx, &chunk)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		// Should have waited at least the initial wait time
		assert.True(t, elapsed >= retryConfig.InitialDelay, "Should have waited for retry backoff")
		mockStore.AssertExpectations(t)
	})
}
