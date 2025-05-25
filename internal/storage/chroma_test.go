package storage

import (
	"context"
	"mcp-memory/internal/config"
	"mcp-memory/pkg/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChromaStore(t *testing.T) {
	cfg := &config.ChromaConfig{
		Endpoint:       "http://localhost:8000",
		Collection:     "test_collection",
		HealthCheck:    true,
		RetryAttempts:  3,
		TimeoutSeconds: 30,
		Docker: config.DockerConfig{
			Enabled:       true,
			ContainerName: "test-chroma",
			VolumePath:    "./test-data",
			Image:         "ghcr.io/chroma-core/chroma:latest",
		},
	}

	store := NewChromaStore(cfg)

	assert.NotNil(t, store.config)
	assert.Equal(t, cfg, store.config)
	assert.NotNil(t, store.metrics)
	assert.Equal(t, "unknown", store.metrics.ConnectionStatus)
}

func TestChromaStore_ChunkToDocument(t *testing.T) {
	cfg := &config.ChromaConfig{
		Endpoint:   "http://localhost:8000",
		Collection: "test",
	}
	store := NewChromaStore(cfg)

	timeSpent := 15
	chunk := types.ConversationChunk{
		ID:        "test-id",
		SessionID: "test-session",
		Timestamp: time.Date(2025, 1, 24, 12, 0, 0, 0, time.UTC),
		Type:      types.ChunkTypeProblem,
		Content:   "Test content",
		Summary:   "Test summary",
		Metadata: types.ChunkMetadata{
			Repository:    "test-repo",
			Branch:        "main",
			FilesModified: []string{"file1.go", "file2.go"},
			ToolsUsed:     []string{"edit", "read"},
			Outcome:       types.OutcomeSuccess,
			Tags:          []string{"go", "test"},
			Difficulty:    types.DifficultyModerate,
			TimeSpent:     &timeSpent,
		},
		Embeddings:    []float64{0.1, 0.2, 0.3},
		RelatedChunks: []string{"chunk1", "chunk2"},
	}

	doc := store.chunkToDocument(chunk)
	metadata := store.chunkToMetadata(chunk)

	assert.Contains(t, doc, "Type: problem")
	assert.Contains(t, doc, "Content: Test content")
	assert.Contains(t, doc, "Summary: Test summary")

	// Check metadata
	assert.Equal(t, "test-session", metadata["session_id"])
	assert.Equal(t, "2025-01-24T12:00:00Z", metadata["timestamp"])
	assert.Equal(t, float64(time.Date(2025, 1, 24, 12, 0, 0, 0, time.UTC).Unix()), metadata["timestamp_epoch"])
	assert.Equal(t, "problem", metadata["type"])
	assert.Equal(t, "Test summary", metadata["summary"])
	assert.Equal(t, "test-repo", metadata["repository"])
	assert.Equal(t, "main", metadata["branch"])
	assert.Equal(t, "success", metadata["outcome"])
	assert.Equal(t, "moderate", metadata["difficulty"])
	assert.Equal(t, "go,test", metadata["tags"])
	assert.Equal(t, "edit,read", metadata["tools_used"])
	assert.Equal(t, "file1.go,file2.go", metadata["files_modified"])
	assert.Equal(t, float64(15), metadata["time_spent"])
}

func TestChromaStore_ChromaResultToChunk(t *testing.T) {
	cfg := &config.ChromaConfig{
		Endpoint:   "http://localhost:8000",
		Collection: "test",
	}
	store := NewChromaStore(cfg)

	id := "test-id"
	document := "Type: problem\nContent: Test content\nSummary: Test summary"
	metadata := map[string]interface{}{
		"session_id":      "test-session",
		"timestamp":       "2025-01-24T12:00:00Z",
		"timestamp_epoch": float64(time.Date(2025, 1, 24, 12, 0, 0, 0, time.UTC).Unix()),
		"type":            "problem",
		"summary":         "Test summary",
		"repository":      "test-repo",
		"branch":          "main",
		"outcome":         "success",
		"difficulty":      "moderate",
		"tags":            "go,test",
		"tools_used":      "edit,read",
		"files_modified":  "file1.go,file2.go",
		"time_spent":      15.0, // JSON numbers are floats
	}

	chunk, err := store.chromaResultToChunk(id, document, metadata)
	require.NoError(t, err)

	assert.Equal(t, id, chunk.ID)
	assert.Equal(t, "test-session", chunk.SessionID)
	assert.Equal(t, "Test content", chunk.Content)
	assert.Equal(t, "Test summary", chunk.Summary)
	assert.Equal(t, types.ChunkTypeProblem, chunk.Type)

	expectedTime, _ := time.Parse(time.RFC3339, "2025-01-24T12:00:00Z")
	assert.Equal(t, expectedTime, chunk.Timestamp)

	assert.Equal(t, "test-repo", chunk.Metadata.Repository)
	assert.Equal(t, "main", chunk.Metadata.Branch)
	assert.Equal(t, types.OutcomeSuccess, chunk.Metadata.Outcome)
	assert.Equal(t, types.DifficultyModerate, chunk.Metadata.Difficulty)
	assert.Equal(t, []string{"go", "test"}, chunk.Metadata.Tags)
	assert.Equal(t, []string{"edit", "read"}, chunk.Metadata.ToolsUsed)
	assert.Equal(t, []string{"file1.go", "file2.go"}, chunk.Metadata.FilesModified)
	assert.Equal(t, 15, *chunk.Metadata.TimeSpent)
}

func TestChromaStore_ChromaResultToChunk_EdgeCases(t *testing.T) {
	cfg := &config.ChromaConfig{
		Endpoint:   "http://localhost:8000",
		Collection: "test",
	}
	store := NewChromaStore(cfg)

	t.Run("missing metadata fields", func(t *testing.T) {
		id := "test-id"
		document := "Content: Test content"
		metadata := map[string]interface{}{
			"type": "problem",
		}

		chunk, err := store.chromaResultToChunk(id, document, metadata)
		require.NoError(t, err)

		assert.Equal(t, id, chunk.ID)
		assert.Equal(t, "Test content", chunk.Content)
		assert.Equal(t, types.ChunkTypeProblem, chunk.Type)
		assert.Empty(t, chunk.SessionID)
		assert.Empty(t, chunk.Summary)
		assert.True(t, chunk.Timestamp.IsZero())
	})

	t.Run("time_spent as string", func(t *testing.T) {
		id := "test-id"
		document := "Content: Test content"
		metadata := map[string]interface{}{
			"type":       "problem",
			"time_spent": "20", // String instead of number
		}

		chunk, err := store.chromaResultToChunk(id, document, metadata)
		require.NoError(t, err)

		assert.Equal(t, 20, *chunk.Metadata.TimeSpent)
	})

	t.Run("empty tags and arrays", func(t *testing.T) {
		id := "test-id"
		document := "Content: Test content"
		metadata := map[string]interface{}{
			"type":           "problem",
			"tags":           "", // Empty string
			"tools_used":     "",
			"files_modified": "",
		}

		chunk, err := store.chromaResultToChunk(id, document, metadata)
		require.NoError(t, err)

		assert.Empty(t, chunk.Metadata.Tags)
		assert.Empty(t, chunk.Metadata.ToolsUsed)
		assert.Empty(t, chunk.Metadata.FilesModified)
	})
}

func TestChromaStore_UpdateMetrics(t *testing.T) {
	cfg := &config.ChromaConfig{
		Endpoint:   "http://localhost:8000",
		Collection: "test",
	}
	store := NewChromaStore(cfg)

	operation := "test_operation"
	start := time.Now().Add(-time.Millisecond) // Ensure some elapsed time

	// Test successful operation
	store.updateMetrics(operation, start, nil)

	assert.Equal(t, int64(1), store.metrics.OperationCounts[operation])
	assert.Contains(t, store.metrics.AverageLatency, operation)
	assert.Greater(t, store.metrics.AverageLatency[operation], 0.0)
	assert.NotNil(t, store.metrics.LastOperation)

	// Test operation with error
	testErr := assert.AnError
	store.updateMetrics(operation, start, testErr)

	assert.Equal(t, int64(2), store.metrics.OperationCounts[operation])
	assert.Equal(t, int64(1), store.metrics.ErrorCounts[operation])
}

// Helper function to convert float64 slice to float32
func float64ToFloat32(input []float64) []float32 {
	output := make([]float32, len(input))
	for i, v := range input {
		output[i] = float32(v)
	}
	return output
}

func TestFloat64ToFloat32(t *testing.T) {
	input := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	output := float64ToFloat32(input)

	assert.Len(t, output, len(input))
	for i, v := range input {
		assert.Equal(t, float32(v), output[i])
	}
}

// Helper function to safely get document from nested slice
func getDocument(documents [][]string, batchIdx, docIdx int) string {
	if documents == nil || batchIdx >= len(documents) || batchIdx < 0 {
		return ""
	}
	if docIdx >= len(documents[batchIdx]) || docIdx < 0 {
		return ""
	}
	return documents[batchIdx][docIdx]
}

func TestGetDocument(t *testing.T) {
	documents := [][]string{
		{"doc1", "doc2", "doc3"},
		{"doc4", "doc5"},
	}

	// Valid access
	assert.Equal(t, "doc1", getDocument(documents, 0, 0))
	assert.Equal(t, "doc3", getDocument(documents, 0, 2))
	assert.Equal(t, "doc5", getDocument(documents, 1, 1))

	// Out of bounds
	assert.Equal(t, "", getDocument(documents, 2, 0))  // Invalid batch
	assert.Equal(t, "", getDocument(documents, 0, 10)) // Invalid index
	assert.Equal(t, "", getDocument(nil, 0, 0))        // Nil documents
}

// Helper function to safely get metadata from nested slice
func getMetadata(metadatas [][]map[string]interface{}, batchIdx, metaIdx int) map[string]interface{} {
	if metadatas == nil || batchIdx >= len(metadatas) || batchIdx < 0 {
		return nil
	}
	if metaIdx >= len(metadatas[batchIdx]) || metaIdx < 0 {
		return nil
	}
	return metadatas[batchIdx][metaIdx]
}

func TestGetMetadata(t *testing.T) {
	metadatas := [][]map[string]interface{}{
		{
			{"key": "value1"},
			{"key": "value2"},
		},
		{
			{"key": "value3"},
		},
	}

	// Valid access
	assert.Equal(t, map[string]interface{}{"key": "value1"}, getMetadata(metadatas, 0, 0))
	assert.Equal(t, map[string]interface{}{"key": "value3"}, getMetadata(metadatas, 1, 0))

	// Out of bounds
	assert.Nil(t, getMetadata(metadatas, 2, 0))  // Invalid batch
	assert.Nil(t, getMetadata(metadatas, 0, 10)) // Invalid index
	assert.Nil(t, getMetadata(nil, 0, 0))        // Nil metadatas
}

func TestNoOpEmbeddingFunction(t *testing.T) {
	fn := &noOpEmbeddingFunction{}
	ctx := context.Background()

	// Test EmbedDocuments
	docs := []string{"doc1", "doc2", "doc3"}
	embeddings, err := fn.EmbedDocuments(ctx, docs)
	assert.NoError(t, err)
	assert.Len(t, embeddings, 3)
	for _, emb := range embeddings {
		assert.Equal(t, 1536, emb.Len())
	}

	// Test EmbedQuery
	queryEmb, err := fn.EmbedQuery(ctx, "test query")
	assert.NoError(t, err)
	assert.Equal(t, 1536, queryEmb.Len())

	// Test EmbedRecords
	records := []map[string]interface{}{
		{"id": "1", "text": "test"},
	}
	err = fn.EmbedRecords(ctx, records, false)
	assert.NoError(t, err)
}

func TestStoreStats(t *testing.T) {
	stats := &StoreStats{
		TotalChunks: 100,
		ChunksByType: map[string]int64{
			"problem":    50,
			"solution":   30,
			"discussion": 20,
		},
		ChunksByRepo: map[string]int64{
			"repo1": 60,
			"repo2": 40,
		},
		StorageSize:      1024000,
		AverageEmbedding: 1536.0,
	}

	assert.Equal(t, int64(100), stats.TotalChunks)
	assert.Equal(t, int64(50), stats.ChunksByType["problem"])
	assert.Equal(t, int64(60), stats.ChunksByRepo["repo1"])
	assert.Equal(t, int64(1024000), stats.StorageSize)
	assert.Equal(t, 1536.0, stats.AverageEmbedding)
}

func TestSearchFilter(t *testing.T) {
	filter := &SearchFilter{
		Repository: func() *string { s := "test-repo"; return &s }(),
		ChunkTypes: []types.ChunkType{types.ChunkTypeProblem},
		TimeRange: &TimeRange{
			Start: func() *string { s := "2025-01-01T00:00:00Z"; return &s }(),
			End:   func() *string { s := "2025-01-31T23:59:59Z"; return &s }(),
		},
		Tags:         []string{"go", "test"},
		Outcomes:     []types.Outcome{types.OutcomeSuccess},
		Difficulties: []types.Difficulty{types.DifficultyModerate},
		FilePatterns: []string{"*.go", "*.test"},
	}

	assert.Equal(t, "test-repo", *filter.Repository)
	assert.Contains(t, filter.ChunkTypes, types.ChunkTypeProblem)
	assert.Equal(t, "2025-01-01T00:00:00Z", *filter.TimeRange.Start)
	assert.Contains(t, filter.Tags, "go")
	assert.Contains(t, filter.Outcomes, types.OutcomeSuccess)
	assert.Contains(t, filter.Difficulties, types.DifficultyModerate)
	assert.Contains(t, filter.FilePatterns, "*.go")
}

func TestBatchOperation(t *testing.T) {
	chunk := types.ConversationChunk{
		ID:        "test-id",
		SessionID: "test-session",
		Type:      types.ChunkTypeProblem,
		Content:   "test content",
	}

	batch := &BatchOperation{
		Operation: "store",
		Chunks:    []types.ConversationChunk{chunk},
	}

	assert.Equal(t, "store", batch.Operation)
	assert.Len(t, batch.Chunks, 1)
	assert.Equal(t, "test-id", batch.Chunks[0].ID)

	// Test delete operation
	deleteBatch := &BatchOperation{
		Operation: "delete",
		IDs:       []string{"id1", "id2", "id3"},
	}

	assert.Equal(t, "delete", deleteBatch.Operation)
	assert.Len(t, deleteBatch.IDs, 3)
}

func TestBatchResult(t *testing.T) {
	result := &BatchResult{
		Success:      5,
		Failed:       2,
		Errors:       []string{"error1", "error2"},
		ProcessedIDs: []string{"id1", "id2", "id3", "id4", "id5"},
	}

	assert.Equal(t, 5, result.Success)
	assert.Equal(t, 2, result.Failed)
	assert.Len(t, result.Errors, 2)
	assert.Len(t, result.ProcessedIDs, 5)
}

func TestStorageMetrics(t *testing.T) {
	metrics := &StorageMetrics{
		OperationCounts: map[string]int64{
			"store":  100,
			"search": 50,
			"get":    25,
		},
		AverageLatency: map[string]float64{
			"store":  150.5,
			"search": 75.2,
			"get":    25.1,
		},
		ErrorCounts: map[string]int64{
			"store":  2,
			"search": 1,
		},
		ConnectionStatus: "healthy",
	}

	assert.Equal(t, int64(100), metrics.OperationCounts["store"])
	assert.Equal(t, 150.5, metrics.AverageLatency["store"])
	assert.Equal(t, int64(2), metrics.ErrorCounts["store"])
	assert.Equal(t, "healthy", metrics.ConnectionStatus)
}

// Unit tests for methods that test actual network errors
func TestChromaStore_NetworkErrors(t *testing.T) {
	cfg := &config.ChromaConfig{
		Endpoint:   "http://invalid-endpoint:9999",
		Collection: "test",
	}
	store := NewChromaStore(cfg)
	ctx := context.Background()

	// Initialize will fail when trying to create client
	err := store.Initialize(ctx)
	assert.Error(t, err)
	assert.Equal(t, "error", store.metrics.ConnectionStatus)
}

func TestChromaStore_ValidationErrors(t *testing.T) {
	cfg := &config.ChromaConfig{
		Endpoint:   "http://localhost:8000",
		Collection: "test",
	}
	store := NewChromaStore(cfg)
	ctx := context.Background()
	mockEmbeddings := []float64{0.1, 0.2, 0.3}

	// Test Store with invalid chunk (missing required fields)
	invalidChunk := types.ConversationChunk{}
	err := store.Store(ctx, invalidChunk)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid chunk")

	// Test Store with chunk without embeddings but valid otherwise
	chunkNoEmbeddings := types.ConversationChunk{
		ID:        "test-id",
		SessionID: "session-1",
		Content:   "test content",
		Type:      types.ChunkTypeProblem,
		Timestamp: time.Now(),
		Metadata: types.ChunkMetadata{
			Outcome:    types.OutcomeSuccess,
			Difficulty: types.DifficultySimple,
		},
	}
	err = store.Store(ctx, chunkNoEmbeddings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chunk must have embeddings")

	// Test Search with invalid query (missing required query field)
	invalidQuery := types.MemoryQuery{
		Limit: 10, // Limit without Query will fail validation
	}
	results, err := store.Search(ctx, invalidQuery, mockEmbeddings)
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "invalid query")
}

// Additional unit tests for uncovered methods
func TestChromaStore_AdditionalMethods(t *testing.T) {
	cfg := &config.ChromaConfig{
		Endpoint:   "http://localhost:8000",
		Collection: "test",
	}
	store := NewChromaStore(cfg)

	// Test Close
	err := store.Close()
	assert.NoError(t, err) // Close should not error
	assert.Equal(t, "closed", store.metrics.ConnectionStatus)
}

// Integration test that would work with a real Chroma instance
func TestChromaStore_Integration(t *testing.T) {
	t.Skip("Integration test - requires running Chroma instance")

	cfg := &config.ChromaConfig{
		Endpoint:       "http://localhost:8000",
		Collection:     "test_integration",
		HealthCheck:    true,
		RetryAttempts:  3,
		TimeoutSeconds: 30,
	}

	store := NewChromaStore(cfg)
	ctx := context.Background()

	t.Run("initialize", func(t *testing.T) {
		err := store.Initialize(ctx)
		assert.NoError(t, err)
	})

	t.Run("health check", func(t *testing.T) {
		err := store.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("store and retrieve chunk", func(t *testing.T) {
		chunk := types.ConversationChunk{
			ID:        "integration-test-id",
			SessionID: "integration-session",
			Timestamp: time.Now(),
			Type:      types.ChunkTypeProblem,
			Content:   "Integration test content",
			Summary:   "Integration test summary",
			Metadata: types.ChunkMetadata{
				Repository: "integration-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
			Embeddings: []float64{0.1, 0.2, 0.3, 0.4, 0.5}, // Simple test embedding
		}

		// Store
		err := store.Store(ctx, chunk)
		require.NoError(t, err)

		// Retrieve
		retrieved, err := store.GetByID(ctx, chunk.ID)
		require.NoError(t, err)
		assert.Equal(t, chunk.ID, retrieved.ID)
		assert.Equal(t, chunk.Content, retrieved.Content)

		// Clean up
		err = store.Delete(ctx, chunk.ID)
		assert.NoError(t, err)
	})
}