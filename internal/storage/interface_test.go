package storage

import (
	"context"
	"testing"
	"time"

	"lerian-mcp-memory/pkg/types"

	"github.com/stretchr/testify/assert"
)

func TestStoreStats(t *testing.T) {
	t.Run("StoreStats initialization", func(t *testing.T) {
		stats := &StoreStats{
			TotalChunks: 100,
			ChunksByType: map[string]int64{
				"problem":  50,
				"solution": 30,
				"decision": 20,
			},
			ChunksByRepo: map[string]int64{
				"repo1": 60,
				"repo2": 40,
			},
			StorageSize: 1024000,
		}

		assert.Equal(t, int64(100), stats.TotalChunks)
		assert.Equal(t, int64(50), stats.ChunksByType["problem"])
		assert.Equal(t, int64(60), stats.ChunksByRepo["repo1"])
		assert.Equal(t, int64(1024000), stats.StorageSize)
	})

	t.Run("Empty stats", func(t *testing.T) {
		stats := &StoreStats{
			ChunksByType: make(map[string]int64),
			ChunksByRepo: make(map[string]int64),
		}

		assert.Equal(t, int64(0), stats.TotalChunks)
		assert.Empty(t, stats.ChunksByType)
		assert.Empty(t, stats.ChunksByRepo)
	})
}

func TestBatchResult(t *testing.T) {
	t.Run("BatchResult success", func(t *testing.T) {
		result := &BatchResult{
			Success:      5,
			Failed:       0,
			Errors:       []string{},
			ProcessedIDs: []string{"id1", "id2", "id3", "id4", "id5"},
		}

		assert.Equal(t, 5, result.Success)
		assert.Equal(t, 0, result.Failed)
		assert.Empty(t, result.Errors)
		assert.Len(t, result.ProcessedIDs, 5)
	})

	t.Run("BatchResult with failures", func(t *testing.T) {
		result := &BatchResult{
			Success:      3,
			Failed:       2,
			Errors:       []string{"error1", "error2"},
			ProcessedIDs: []string{"id1", "id2", "id3", "id4", "id5"},
		}

		assert.Equal(t, 3, result.Success)
		assert.Equal(t, 2, result.Failed)
		assert.Len(t, result.Errors, 2)
		assert.Contains(t, result.Errors, "error1")
		assert.Contains(t, result.Errors, "error2")
	})
}

func TestVectorStoreContract(t *testing.T) {
	// Test that our mock implements the VectorStore interface
	store := NewSimpleMockVectorStore()

	ctx := context.Background()

	// Test interface compliance
	assert.NotNil(t, store)

	// Test all required methods exist and can be called
	err := store.Initialize(ctx)
	assert.NoError(t, err)

	err = store.HealthCheck(ctx)
	assert.NoError(t, err)

	err = store.Close()
	assert.NoError(t, err)
}

func TestVectorStoreSearchFiltering(t *testing.T) {
	store := NewSimpleMockVectorStore()
	ctx := context.Background()

	// Add test chunks with different types and repositories
	chunks := []types.ConversationChunk{
		{
			ID:         "test-1",
			SessionID:  "session-1",
			Type:       types.ChunkTypeProblem,
			Content:    "Test problem 1",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2, 0.3},
			Metadata: types.ChunkMetadata{
				Repository: "repo-a",
				Tags:       []string{"tag1"},
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		},
		{
			ID:         "test-2",
			SessionID:  "session-1",
			Type:       types.ChunkTypeSolution,
			Content:    "Test solution 1",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.4, 0.5, 0.6},
			Metadata: types.ChunkMetadata{
				Repository: "repo-b",
				Tags:       []string{"tag2"},
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultyModerate,
			},
		},
		{
			ID:         "test-3",
			SessionID:  "session-2",
			Type:       types.ChunkTypeArchitectureDecision,
			Content:    "Test decision 1",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.7, 0.8, 0.9},
			Metadata: types.ChunkMetadata{
				Repository: "repo-a",
				Tags:       []string{"tag1", "tag3"},
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultyComplex,
			},
		},
	}

	// Store all chunks
	for i := range chunks {
		err := store.Store(ctx, &chunks[i])
		assert.NoError(t, err)
	}

	t.Run("Filter by repository", func(t *testing.T) {
		repo := "repo-a"
		query := types.MemoryQuery{
			Query:      "test",
			Repository: &repo,
			Limit:      10,
		}
		embeddings := []float64{0.1, 0.2, 0.3}

		results, err := store.Search(ctx, &query, embeddings)
		assert.NoError(t, err)
		assert.Equal(t, 2, results.Total) // Should find test-1 and test-3

		for _, result := range results.Results {
			assert.Equal(t, "repo-a", result.Chunk.Metadata.Repository)
		}
	})

	t.Run("Filter by chunk types", func(t *testing.T) {
		query := types.MemoryQuery{
			Query: "test",
			Types: []types.ChunkType{types.ChunkTypeProblem, types.ChunkTypeSolution},
			Limit: 10,
		}
		embeddings := []float64{0.1, 0.2, 0.3}

		results, err := store.Search(ctx, &query, embeddings)
		assert.NoError(t, err)
		assert.Equal(t, 2, results.Total) // Should find test-1 and test-2

		foundTypes := make(map[types.ChunkType]bool)
		for _, result := range results.Results {
			foundTypes[result.Chunk.Type] = true
		}
		assert.True(t, foundTypes[types.ChunkTypeProblem])
		assert.True(t, foundTypes[types.ChunkTypeSolution])
		assert.False(t, foundTypes[types.ChunkTypeArchitectureDecision])
	})

	t.Run("Filter by repository and types", func(t *testing.T) {
		repo := "repo-a"
		query := types.MemoryQuery{
			Query:      "test",
			Repository: &repo,
			Types:      []types.ChunkType{types.ChunkTypeArchitectureDecision},
			Limit:      10,
		}
		embeddings := []float64{0.1, 0.2, 0.3}

		results, err := store.Search(ctx, &query, embeddings)
		assert.NoError(t, err)
		assert.Equal(t, 1, results.Total) // Should find only test-3

		assert.Equal(t, "test-3", results.Results[0].Chunk.ID)
		assert.Equal(t, types.ChunkTypeArchitectureDecision, results.Results[0].Chunk.Type)
		assert.Equal(t, "repo-a", results.Results[0].Chunk.Metadata.Repository)
	})
}

func TestVectorStoreEdgeCases(t *testing.T) {
	store := NewSimpleMockVectorStore()
	ctx := context.Background()

	t.Run("Search with no matching criteria", func(t *testing.T) {
		// Add a chunk first
		chunk := types.ConversationChunk{
			ID:         "edge-case-1",
			SessionID:  "session-edge",
			Type:       types.ChunkTypeProblem,
			Content:    "Edge case content",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2},
			Metadata: types.ChunkMetadata{
				Repository: "edge-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		}
		err := store.Store(ctx, &chunk)
		assert.NoError(t, err)

		// Search for non-matching repository
		nonMatchingRepo := "non-existent-repo"
		query := types.MemoryQuery{
			Query:      "test",
			Repository: &nonMatchingRepo,
			Limit:      10,
		}
		embeddings := []float64{0.1, 0.2}

		results, err := store.Search(ctx, &query, embeddings)
		assert.NoError(t, err)
		assert.Equal(t, 0, results.Total)
		assert.Empty(t, results.Results)
	})

	t.Run("Search with non-matching types", func(t *testing.T) {
		query := types.MemoryQuery{
			Query: "test",
			Types: []types.ChunkType{types.ChunkTypeSolution}, // Chunk is problem type
			Limit: 10,
		}
		embeddings := []float64{0.1, 0.2}

		results, err := store.Search(ctx, &query, embeddings)
		assert.NoError(t, err)
		assert.Equal(t, 0, results.Total)
		assert.Empty(t, results.Results)
	})

	t.Run("List by non-existent repository", func(t *testing.T) {
		chunks, err := store.ListByRepository(ctx, "non-existent-repo", 10, 0)
		assert.NoError(t, err)
		assert.Empty(t, chunks)
	})

	t.Run("List by non-existent session", func(t *testing.T) {
		chunks, err := store.ListBySession(ctx, "non-existent-session")
		assert.NoError(t, err)
		assert.Empty(t, chunks)
	})
}

func TestVectorStoreValidation(t *testing.T) {
	store := NewSimpleMockVectorStore()
	ctx := context.Background()

	t.Run("Store chunk without required fields", func(t *testing.T) {
		invalidChunk := types.ConversationChunk{
			// Missing required fields like ID, Type, etc.
			Content: "Some content",
		}

		err := store.Store(ctx, &invalidChunk)
		assert.Error(t, err)
	})

	t.Run("Store chunk without embeddings", func(t *testing.T) {
		invalidChunk := types.ConversationChunk{
			ID:        "no-embeddings",
			SessionID: "session",
			Type:      types.ChunkTypeProblem,
			Content:   "Content without embeddings",
			Timestamp: time.Now(),
			// Embeddings is empty/nil
			Metadata: types.ChunkMetadata{
				Repository: "test-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		}

		err := store.Store(ctx, &invalidChunk)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "embeddings")
	})

	t.Run("Update chunk without required fields", func(t *testing.T) {
		invalidChunk := types.ConversationChunk{
			ID: "update-invalid",
			// Missing other required fields
		}

		err := store.Update(ctx, &invalidChunk)
		assert.Error(t, err)
	})
}

func TestVectorStoreQueryTime(t *testing.T) {
	store := NewSimpleMockVectorStore()
	ctx := context.Background()

	// Add a test chunk
	chunk := types.ConversationChunk{
		ID:         "query-time-test",
		SessionID:  "session",
		Type:       types.ChunkTypeProblem,
		Content:    "Query time test",
		Timestamp:  time.Now(),
		Embeddings: []float64{0.1, 0.2},
		Metadata: types.ChunkMetadata{
			Repository: "test-repo",
			Outcome:    types.OutcomeSuccess,
			Difficulty: types.DifficultySimple,
		},
	}
	err := store.Store(ctx, &chunk)
	assert.NoError(t, err)

	query := types.MemoryQuery{
		Query: "test",
		Limit: 10,
	}
	embeddings := []float64{0.1, 0.2}

	start := time.Now()
	results, err := store.Search(ctx, &query, embeddings)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.True(t, results.QueryTime > 0, "QueryTime should be greater than 0")
	assert.True(t, elapsed >= results.QueryTime, "Actual elapsed time should be >= reported query time")
}
