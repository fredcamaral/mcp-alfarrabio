package storage

import (
	"context"
	"fmt"
	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/pkg/types"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockQdrantStore for testing without actual Qdrant connection
type MockQdrantStore struct {
	chunks map[string]types.ConversationChunk
	stats  *StoreStats
}

func NewMockQdrantStore() *MockQdrantStore {
	return &MockQdrantStore{
		chunks: make(map[string]types.ConversationChunk),
		stats: &StoreStats{
			TotalChunks:  0,
			ChunksByType: make(map[string]int64),
			ChunksByRepo: make(map[string]int64),
			StorageSize:  0,
		},
	}
}

// Implement VectorStore interface for MockQdrantStore

func (m *MockQdrantStore) Initialize(ctx context.Context) error {
	return nil
}

//nolint:gocritic // Interface requirement - hugeParam: types.ConversationChunk is required by VectorStore interface
func (m *MockQdrantStore) Store(ctx context.Context, chunk types.ConversationChunk) error {
	if err := chunk.Validate(); err != nil {
		return err
	}
	if len(chunk.Embeddings) == 0 {
		return &ValidationError{
			Component: "chunk",
			Type:      "missing_embeddings",
			Message:   "chunk must have embeddings",
			Severity:  "critical",
			Code:      "VAL001",
		}
	}
	m.chunks[chunk.ID] = chunk
	m.updateStats(chunk)
	return nil
}

//nolint:gocritic // Interface requirement - hugeParam: types.MemoryQuery is required by VectorStore interface
func (m *MockQdrantStore) Search(ctx context.Context, query types.MemoryQuery, embeddings []float64) (*types.SearchResults, error) {
	if len(embeddings) == 0 {
		return nil, &ValidationError{
			Component: "query",
			Type:      "missing_embeddings",
			Message:   "embeddings cannot be empty",
			Severity:  "critical",
			Code:      "VAL002",
		}
	}

	results := &types.SearchResults{
		Results:   []types.SearchResult{},
		Total:     0,
		QueryTime: time.Millisecond * 10,
	}

	// Simple mock search - return matching chunks
	for _, chunk := range m.chunks {
		if query.Repository != nil && chunk.Metadata.Repository != *query.Repository {
			continue
		}
		if len(query.Types) > 0 {
			found := false
			for _, t := range query.Types {
				if chunk.Type == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		results.Results = append(results.Results, types.SearchResult{
			Chunk: chunk,
			Score: 0.85, // Mock score
		})
	}

	results.Total = len(results.Results)
	return results, nil
}

func (m *MockQdrantStore) GetByID(ctx context.Context, id string) (*types.ConversationChunk, error) {
	chunk, exists := m.chunks[id]
	if !exists {
		return nil, &NotFoundError{ID: id}
	}
	return &chunk, nil
}

func (m *MockQdrantStore) ListByRepository(ctx context.Context, repository string, limit, offset int) ([]types.ConversationChunk, error) {
	// First, collect all matching chunks
	var allChunks []types.ConversationChunk
	for chunkID := range m.chunks {
		chunk := m.chunks[chunkID]
		if chunk.Metadata.Repository == repository {
			allChunks = append(allChunks, chunk)
		}
	}

	// Sort by timestamp (newest first) to match Qdrant implementation
	sort.Slice(allChunks, func(i, j int) bool {
		return allChunks[i].Timestamp.After(allChunks[j].Timestamp)
	})

	// Apply offset and limit
	var chunks []types.ConversationChunk
	if offset < len(allChunks) {
		end := offset + limit
		if end > len(allChunks) {
			end = len(allChunks)
		}
		chunks = allChunks[offset:end]
	}

	return chunks, nil
}

func (m *MockQdrantStore) ListBySession(ctx context.Context, sessionID string) ([]types.ConversationChunk, error) {
	chunks := []types.ConversationChunk{}
	for chunkID := range m.chunks {
		chunk := m.chunks[chunkID]
		if chunk.SessionID == sessionID {
			chunks = append(chunks, chunk)
		}
	}
	return chunks, nil
}

func (m *MockQdrantStore) Delete(ctx context.Context, id string) error {
	if _, exists := m.chunks[id]; !exists {
		return &NotFoundError{ID: id}
	}
	delete(m.chunks, id)
	return nil
}

//nolint:gocritic // Interface requirement - hugeParam: types.ConversationChunk is required by VectorStore interface
func (m *MockQdrantStore) Update(ctx context.Context, chunk types.ConversationChunk) error {
	if err := chunk.Validate(); err != nil {
		return err
	}
	if _, exists := m.chunks[chunk.ID]; !exists {
		return &NotFoundError{ID: chunk.ID}
	}
	m.chunks[chunk.ID] = chunk
	return nil
}

func (m *MockQdrantStore) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockQdrantStore) GetStats(ctx context.Context) (*StoreStats, error) {
	stats := &StoreStats{
		TotalChunks:  int64(len(m.chunks)),
		ChunksByType: make(map[string]int64),
		ChunksByRepo: make(map[string]int64),
		StorageSize:  int64(len(m.chunks) * 1000), // Mock size
	}

	for chunkID := range m.chunks {
		chunk := m.chunks[chunkID]
		stats.ChunksByType[string(chunk.Type)]++
		stats.ChunksByRepo[chunk.Metadata.Repository]++
	}

	return stats, nil
}

func (m *MockQdrantStore) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	deleted := 0
	for id := range m.chunks {
		chunk := m.chunks[id]
		if chunk.Timestamp.Before(cutoff) {
			delete(m.chunks, id)
			deleted++
		}
	}
	return deleted, nil
}

func (m *MockQdrantStore) Close() error {
	return nil
}

// New interface methods
func (m *MockQdrantStore) GetAllChunks(ctx context.Context) ([]types.ConversationChunk, error) {
	chunks := make([]types.ConversationChunk, 0, len(m.chunks))
	for chunkID := range m.chunks {
		chunks = append(chunks, m.chunks[chunkID])
	}
	return chunks, nil
}

func (m *MockQdrantStore) DeleteCollection(ctx context.Context, collection string) error {
	m.chunks = make(map[string]types.ConversationChunk)
	return nil
}

func (m *MockQdrantStore) ListCollections(ctx context.Context) ([]string, error) {
	return []string{"claude_memory"}, nil
}

func (m *MockQdrantStore) FindSimilar(ctx context.Context, content string, chunkType *types.ChunkType, limit int) ([]types.ConversationChunk, error) {
	return []types.ConversationChunk{}, nil // Simplified mock
}

//nolint:gocritic // Interface requirement - hugeParam: types.ConversationChunk is required by VectorStore interface
func (m *MockQdrantStore) StoreChunk(ctx context.Context, chunk types.ConversationChunk) error {
	return m.Store(ctx, chunk)
}

func (m *MockQdrantStore) BatchStore(ctx context.Context, chunks []types.ConversationChunk) (*BatchResult, error) {
	result := &BatchResult{
		Success:      0,
		Failed:       0,
		Errors:       []string{},
		ProcessedIDs: []string{},
	}

	for i := range chunks {
		chunk := chunks[i]
		if err := m.Store(ctx, chunk); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Success++
		}
		result.ProcessedIDs = append(result.ProcessedIDs, chunk.ID)
	}

	return result, nil
}

func (m *MockQdrantStore) BatchDelete(ctx context.Context, ids []string) (*BatchResult, error) {
	result := &BatchResult{
		Success:      0,
		Failed:       0,
		Errors:       []string{},
		ProcessedIDs: ids,
	}

	for _, id := range ids {
		if err := m.Delete(ctx, id); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Success++
		}
	}

	return result, nil
}

//nolint:gocritic // Interface requirement - hugeParam: types.ConversationChunk is required by VectorStore interface
func (m *MockQdrantStore) updateStats(chunk types.ConversationChunk) {
	// Update mock stats - simplified
}

type NotFoundError struct {
	ID string
}

func (e *NotFoundError) Error() string {
	return "chunk not found: " + e.ID
}

// Test functions

func TestQdrantStoreCreation(t *testing.T) {
	cfg := &config.QdrantConfig{
		Host:       "localhost",
		Port:       6334,
		Collection: "test",
	}

	store := NewQdrantStore(cfg)
	assert.NotNil(t, store)
	assert.Equal(t, "test", store.collectionName)
}

func TestQdrantStoreDefaultCollection(t *testing.T) {
	cfg := &config.QdrantConfig{
		Host: "localhost",
		Port: 6334,
		// Collection not set
	}

	store := NewQdrantStore(cfg)
	assert.Equal(t, "claude_memory", store.collectionName)
}

func TestVectorStoreInterface(t *testing.T) {
	store := NewMockQdrantStore()
	ctx := context.Background()

	// Test initialization
	err := store.Initialize(ctx)
	assert.NoError(t, err)

	// Create test chunk
	chunk := types.ConversationChunk{
		ID:         "test-chunk-1",
		SessionID:  "test-session",
		Type:       types.ChunkTypeProblem,
		Content:    "Test problem content",
		Summary:    "Test problem",
		Timestamp:  time.Now(),
		Embeddings: []float64{0.1, 0.2, 0.3, 0.4},
		Metadata: types.ChunkMetadata{
			Repository: "test-repo",
			Tags:       []string{"test"},
			Outcome:    types.OutcomeSuccess,
			Difficulty: types.DifficultySimple,
		},
	}

	// Test store
	err = store.Store(ctx, chunk)
	assert.NoError(t, err)

	// Test get by ID
	retrieved, err := store.GetByID(ctx, chunk.ID)
	require.NoError(t, err)
	assert.Equal(t, chunk.ID, retrieved.ID)
	assert.Equal(t, chunk.Content, retrieved.Content)

	// Test search
	query := types.MemoryQuery{
		Query:      "problem",
		Repository: &chunk.Metadata.Repository,
		Limit:      10,
	}
	embeddings := []float64{0.1, 0.2, 0.3, 0.4}

	results, err := store.Search(ctx, query, embeddings)
	require.NoError(t, err)
	assert.Equal(t, 1, results.Total)
	assert.Equal(t, chunk.ID, results.Results[0].Chunk.ID)

	// Test list by repository
	repoChunks, err := store.ListByRepository(ctx, "test-repo", 10, 0)
	require.NoError(t, err)
	assert.Len(t, repoChunks, 1)

	// Test list by session
	sessionChunks, err := store.ListBySession(ctx, "test-session")
	require.NoError(t, err)
	assert.Len(t, sessionChunks, 1)

	// Test update
	chunk.Content = "Updated content"
	err = store.Update(ctx, chunk)
	assert.NoError(t, err)

	// Test delete
	err = store.Delete(ctx, chunk.ID)
	assert.NoError(t, err)

	// Verify deletion
	_, err = store.GetByID(ctx, chunk.ID)
	assert.Error(t, err)
}

func TestBatchOperations(t *testing.T) {
	store := NewMockQdrantStore()
	ctx := context.Background()

	// Create test chunks
	chunks := []types.ConversationChunk{
		{
			ID:         "batch-1",
			SessionID:  "batch-session",
			Type:       types.ChunkTypeProblem,
			Content:    "Batch problem 1",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2},
			Metadata: types.ChunkMetadata{
				Repository: "batch-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		},
		{
			ID:         "batch-2",
			SessionID:  "batch-session",
			Type:       types.ChunkTypeSolution,
			Content:    "Batch solution 1",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.3, 0.4},
			Metadata: types.ChunkMetadata{
				Repository: "batch-repo",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		},
	}

	// Test batch store
	result, err := store.BatchStore(ctx, chunks)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Success)
	assert.Equal(t, 0, result.Failed)
	assert.Len(t, result.ProcessedIDs, 2)

	// Verify chunks were stored
	for _, chunk := range chunks {
		retrieved, err := store.GetByID(ctx, chunk.ID)
		require.NoError(t, err)
		assert.Equal(t, chunk.ID, retrieved.ID)
	}

	// Test batch delete
	ids := []string{"batch-1", "batch-2"}
	deleteResult, err := store.BatchDelete(ctx, ids)
	require.NoError(t, err)
	assert.Equal(t, 2, deleteResult.Success)
	assert.Equal(t, 0, deleteResult.Failed)

	// Verify chunks were deleted
	for _, id := range ids {
		_, err := store.GetByID(ctx, id)
		assert.Error(t, err)
	}
}

func TestNewInterfaceMethods(t *testing.T) {
	store := NewMockQdrantStore()
	ctx := context.Background()

	// Add some test data
	chunk := types.ConversationChunk{
		ID:         "new-method-test",
		SessionID:  "test-session",
		Type:       types.ChunkTypeProblem,
		Content:    "New method test",
		Timestamp:  time.Now(),
		Embeddings: []float64{0.1, 0.2},
		Metadata: types.ChunkMetadata{
			Repository: "new-repo",
			Outcome:    types.OutcomeSuccess,
			Difficulty: types.DifficultySimple,
		},
	}

	err := store.Store(ctx, chunk)
	require.NoError(t, err)

	// Test GetAllChunks
	allChunks, err := store.GetAllChunks(ctx)
	require.NoError(t, err)
	assert.Len(t, allChunks, 1)

	// Test ListCollections
	collections, err := store.ListCollections(ctx)
	require.NoError(t, err)
	assert.Contains(t, collections, "claude_memory")

	// Test StoreChunk (alias method)
	chunk2 := chunk
	chunk2.ID = "alias-test"
	err = store.StoreChunk(ctx, chunk2)
	assert.NoError(t, err)

	// Test DeleteCollection
	err = store.DeleteCollection(ctx, "test-collection")
	assert.NoError(t, err)
}

func TestStatsGeneration(t *testing.T) {
	store := NewMockQdrantStore()
	ctx := context.Background()

	// Add test chunks of different types
	chunks := []types.ConversationChunk{
		{
			ID:         "stats-1",
			Type:       types.ChunkTypeProblem,
			SessionID:  "stats-session",
			Content:    "Problem 1",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.1, 0.2},
			Metadata: types.ChunkMetadata{
				Repository: "repo1",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		},
		{
			ID:         "stats-2",
			Type:       types.ChunkTypeSolution,
			SessionID:  "stats-session",
			Content:    "Solution 1",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.3, 0.4},
			Metadata: types.ChunkMetadata{
				Repository: "repo1",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		},
		{
			ID:         "stats-3",
			Type:       types.ChunkTypeProblem,
			SessionID:  "stats-session",
			Content:    "Problem 2",
			Timestamp:  time.Now(),
			Embeddings: []float64{0.5, 0.6},
			Metadata: types.ChunkMetadata{
				Repository: "repo2",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		},
	}

	// Store chunks
	for _, chunk := range chunks {
		err := store.Store(ctx, chunk)
		require.NoError(t, err)
	}

	// Get stats
	stats, err := store.GetStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, int64(3), stats.TotalChunks)
	assert.Equal(t, int64(2), stats.ChunksByType[string(types.ChunkTypeProblem)])
	assert.Equal(t, int64(1), stats.ChunksByType[string(types.ChunkTypeSolution)])
	assert.Equal(t, int64(2), stats.ChunksByRepo["repo1"])
	assert.Equal(t, int64(1), stats.ChunksByRepo["repo2"])
}

func TestErrorHandling(t *testing.T) {
	store := NewMockQdrantStore()
	ctx := context.Background()

	// Test store with invalid chunk (no embeddings)
	invalidChunk := types.ConversationChunk{
		ID:        "invalid",
		Type:      types.ChunkTypeProblem,
		Content:   "Invalid chunk",
		Timestamp: time.Now(),
		// No embeddings
	}

	err := store.Store(ctx, invalidChunk)
	assert.Error(t, err)

	// Test get non-existent chunk
	_, err = store.GetByID(ctx, "non-existent")
	assert.Error(t, err)

	// Test search with no embeddings
	query := types.MemoryQuery{Query: "test"}
	_, err = store.Search(ctx, query, []float64{})
	assert.Error(t, err)

	// Test update non-existent chunk
	err = store.Update(ctx, types.ConversationChunk{
		ID:         "non-existent",
		SessionID:  "test-session",
		Type:       types.ChunkTypeProblem,
		Content:    "Update test",
		Timestamp:  time.Now(),
		Embeddings: []float64{0.1, 0.2},
		Metadata: types.ChunkMetadata{
			Repository: "test-repo",
			Outcome:    types.OutcomeSuccess,
			Difficulty: types.DifficultySimple,
		},
	})
	assert.Error(t, err)

	// Test delete non-existent chunk
	err = store.Delete(ctx, "non-existent")
	assert.Error(t, err)
}

func TestHealthCheck(t *testing.T) {
	store := NewMockQdrantStore()
	ctx := context.Background()

	err := store.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestCleanup(t *testing.T) {
	store := NewMockQdrantStore()
	ctx := context.Background()

	// Add old and new chunks
	oldChunk := types.ConversationChunk{
		ID:         "old-chunk",
		SessionID:  "cleanup-session",
		Type:       types.ChunkTypeProblem,
		Content:    "Old chunk",
		Timestamp:  time.Now().AddDate(0, 0, -10), // 10 days old
		Embeddings: []float64{0.1, 0.2},
		Metadata: types.ChunkMetadata{
			Repository: "old-repo",
			Outcome:    types.OutcomeSuccess,
			Difficulty: types.DifficultySimple,
		},
	}

	newChunk := types.ConversationChunk{
		ID:         "new-chunk",
		SessionID:  "cleanup-session",
		Type:       types.ChunkTypeProblem,
		Content:    "New chunk",
		Timestamp:  time.Now(),
		Embeddings: []float64{0.3, 0.4},
		Metadata: types.ChunkMetadata{
			Repository: "new-repo",
			Outcome:    types.OutcomeSuccess,
			Difficulty: types.DifficultySimple,
		},
	}

	err := store.Store(ctx, oldChunk)
	require.NoError(t, err)
	err = store.Store(ctx, newChunk)
	require.NoError(t, err)

	// Cleanup chunks older than 7 days
	deleted, err := store.Cleanup(ctx, 7)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)

	// Verify old chunk is gone, new chunk remains
	_, err = store.GetByID(ctx, "old-chunk")
	assert.Error(t, err)

	_, err = store.GetByID(ctx, "new-chunk")
	assert.NoError(t, err)
}

func TestListByRepositoryPagination(t *testing.T) {
	store := NewMockQdrantStore()
	ctx := context.Background()

	// Add multiple chunks to test pagination
	repository := "pagination-test-repo"
	for i := 0; i < 5; i++ {
		chunk := types.ConversationChunk{
			ID:         fmt.Sprintf("chunk-%d", i),
			SessionID:  "pagination-session",
			Type:       types.ChunkTypeProblem,
			Content:    fmt.Sprintf("Chunk content %d", i),
			Timestamp:  time.Now().Add(time.Duration(i) * time.Minute), // Different timestamps
			Embeddings: []float64{0.1, 0.2},
			Metadata: types.ChunkMetadata{
				Repository: repository,
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultySimple,
			},
		}
		err := store.Store(ctx, chunk)
		require.NoError(t, err)
	}

	// Test first page (limit 2, offset 0)
	firstPage, err := store.ListByRepository(ctx, repository, 2, 0)
	require.NoError(t, err)
	assert.Len(t, firstPage, 2)

	// Test second page (limit 2, offset 2)
	secondPage, err := store.ListByRepository(ctx, repository, 2, 2)
	require.NoError(t, err)
	assert.Len(t, secondPage, 2)

	// Test third page (limit 2, offset 4) - should have 1 item
	thirdPage, err := store.ListByRepository(ctx, repository, 2, 4)
	require.NoError(t, err)
	assert.Len(t, thirdPage, 1)

	// Test beyond available items (limit 2, offset 10)
	beyondPage, err := store.ListByRepository(ctx, repository, 2, 10)
	require.NoError(t, err)
	assert.Len(t, beyondPage, 0)

	// Verify no duplicate chunks between pages
	firstPageIDs := make(map[string]bool)
	for _, chunk := range firstPage {
		firstPageIDs[chunk.ID] = true
	}

	for _, chunk := range secondPage {
		assert.False(t, firstPageIDs[chunk.ID], "Found duplicate chunk ID between pages: %s", chunk.ID)
	}
}
