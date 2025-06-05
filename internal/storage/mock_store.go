package storage

import (
	"context"
	"errors"
	"mcp-memory/pkg/types"
	"strings"
	"time"
)

// SimpleMockVectorStore is a basic implementation of VectorStore for testing
type SimpleMockVectorStore struct {
	chunks        map[string]types.ConversationChunk
	relationships map[string]*types.MemoryRelationship
	stats         *StoreStats
}

// NewSimpleMockVectorStore creates a new simple mock vector store
func NewSimpleMockVectorStore() VectorStore {
	return &SimpleMockVectorStore{
		chunks:        make(map[string]types.ConversationChunk),
		relationships: make(map[string]*types.MemoryRelationship),
		stats: &StoreStats{
			TotalChunks:  0,
			ChunksByType: make(map[string]int64),
			ChunksByRepo: make(map[string]int64),
		},
	}
}

func (m *SimpleMockVectorStore) Initialize(ctx context.Context) error {
	return nil
}

func (m *SimpleMockVectorStore) Store(ctx context.Context, chunk *types.ConversationChunk) error {
	if chunk.ID == "" {
		return errors.New("chunk ID is required")
	}
	if chunk.Type == "" {
		return errors.New("chunk type is required")
	}
	if len(chunk.Embeddings) == 0 {
		return errors.New("embeddings are required")
	}

	m.chunks[chunk.ID] = *chunk
	m.updateStats(chunk)
	return nil
}

func (m *SimpleMockVectorStore) updateStats(chunk *types.ConversationChunk) {
	m.stats.TotalChunks++
	m.stats.ChunksByType[string(chunk.Type)]++
	if chunk.Metadata.Repository != "" {
		m.stats.ChunksByRepo[chunk.Metadata.Repository]++
	}
}

func (m *SimpleMockVectorStore) Search(ctx context.Context, query *types.MemoryQuery, embeddings []float64) (*types.SearchResults, error) {
	start := time.Now()

	capacity := len(m.chunks)
	if query.Limit > 0 && query.Limit < capacity {
		capacity = query.Limit
	}
	results := make([]types.SearchResult, 0, capacity)
	for _, chunk := range m.chunks {
		// Apply repository filter
		if query.Repository != nil && chunk.Metadata.Repository != *query.Repository {
			continue
		}

		// Apply type filter
		if len(query.Types) > 0 {
			typeMatches := false
			for _, t := range query.Types {
				if chunk.Type == t {
					typeMatches = true
					break
				}
			}
			if !typeMatches {
				continue
			}
		}

		// Simple content matching (basic implementation)
		if query.Query != "" {
			// For testing, just check if query is in content
			// In real implementation, this would use embeddings
			if !strings.Contains(strings.ToLower(chunk.Content), strings.ToLower(query.Query)) {
				continue // Skip chunks that don't match query
			}
		}

		results = append(results, types.SearchResult{
			Chunk: chunk,
			Score: 0.8, // Mock score
		})

		if len(results) >= query.Limit {
			break
		}
	}

	return &types.SearchResults{
		Results:   results,
		Total:     len(results),
		QueryTime: time.Since(start),
	}, nil
}

func (m *SimpleMockVectorStore) GetByID(ctx context.Context, id string) (*types.ConversationChunk, error) {
	chunk, exists := m.chunks[id]
	if !exists {
		return nil, errors.New("chunk not found")
	}
	return &chunk, nil
}

func (m *SimpleMockVectorStore) ListByRepository(ctx context.Context, repository string, limit int, offset int) ([]types.ConversationChunk, error) {
	results := make([]types.ConversationChunk, 0, limit)
	count := 0

	for _, chunk := range m.chunks {
		if chunk.Metadata.Repository == repository {
			if count >= offset {
				results = append(results, chunk)
				if len(results) >= limit {
					break
				}
			}
			count++
		}
	}

	return results, nil
}

func (m *SimpleMockVectorStore) ListBySession(ctx context.Context, sessionID string) ([]types.ConversationChunk, error) {
	results := make([]types.ConversationChunk, 0, len(m.chunks))

	for _, chunk := range m.chunks {
		if chunk.SessionID == sessionID {
			results = append(results, chunk)
		}
	}

	return results, nil
}

func (m *SimpleMockVectorStore) Delete(ctx context.Context, id string) error {
	if _, exists := m.chunks[id]; !exists {
		return errors.New("chunk not found")
	}
	delete(m.chunks, id)
	return nil
}

func (m *SimpleMockVectorStore) Update(ctx context.Context, chunk *types.ConversationChunk) error {
	if chunk.ID == "" {
		return errors.New("chunk ID is required")
	}
	if _, exists := m.chunks[chunk.ID]; !exists {
		return errors.New("chunk not found")
	}

	m.chunks[chunk.ID] = *chunk
	return nil
}

func (m *SimpleMockVectorStore) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *SimpleMockVectorStore) GetStats(ctx context.Context) (*StoreStats, error) {
	return m.stats, nil
}

func (m *SimpleMockVectorStore) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	return 0, nil
}

func (m *SimpleMockVectorStore) Close() error {
	return nil
}

func (m *SimpleMockVectorStore) GetAllChunks(ctx context.Context) ([]types.ConversationChunk, error) {
	results := make([]types.ConversationChunk, 0, len(m.chunks))
	for _, chunk := range m.chunks {
		results = append(results, chunk)
	}
	return results, nil
}

func (m *SimpleMockVectorStore) DeleteCollection(ctx context.Context, collection string) error {
	return nil
}

func (m *SimpleMockVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	return []string{"default"}, nil
}

func (m *SimpleMockVectorStore) FindSimilar(ctx context.Context, content string, chunkType *types.ChunkType, limit int) ([]types.ConversationChunk, error) {
	results := make([]types.ConversationChunk, 0, limit)
	count := 0

	for _, chunk := range m.chunks {
		if chunkType != nil && chunk.Type != *chunkType {
			continue
		}

		results = append(results, chunk)
		count++
		if count >= limit {
			break
		}
	}

	return results, nil
}

func (m *SimpleMockVectorStore) StoreChunk(ctx context.Context, chunk *types.ConversationChunk) error {
	return m.Store(ctx, chunk)
}

func (m *SimpleMockVectorStore) BatchStore(ctx context.Context, chunks []*types.ConversationChunk) (*BatchResult, error) {
	result := &BatchResult{
		Success:      0,
		Failed:       0,
		Errors:       []string{},
		ProcessedIDs: []string{},
	}

	for _, chunk := range chunks {
		err := m.Store(ctx, chunk)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Success++
		}
		result.ProcessedIDs = append(result.ProcessedIDs, chunk.ID)
	}

	return result, nil
}

func (m *SimpleMockVectorStore) BatchDelete(ctx context.Context, ids []string) (*BatchResult, error) {
	result := &BatchResult{
		Success:      0,
		Failed:       0,
		Errors:       []string{},
		ProcessedIDs: ids,
	}

	for _, id := range ids {
		err := m.Delete(ctx, id)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Success++
		}
	}

	return result, nil
}

// StoreRelationship stores a relationship between two memory chunks
func (m *SimpleMockVectorStore) StoreRelationship(ctx context.Context, sourceID, targetID string, relationType types.RelationType, confidence float64, source types.ConfidenceSource) (*types.MemoryRelationship, error) {
	rel := &types.MemoryRelationship{
		ID:               "rel-" + sourceID + "-" + targetID,
		SourceChunkID:    sourceID,
		TargetChunkID:    targetID,
		RelationType:     relationType,
		Confidence:       confidence,
		ConfidenceSource: source,
		CreatedAt:        time.Now(),
	}

	m.relationships[rel.ID] = rel
	return rel, nil
}

func (m *SimpleMockVectorStore) GetRelationships(ctx context.Context, query *types.RelationshipQuery) ([]types.RelationshipResult, error) {
	results := make([]types.RelationshipResult, 0, len(m.relationships))

	for _, rel := range m.relationships {
		// Apply filters based on query
		results = append(results, types.RelationshipResult{
			Relationship: *rel,
		})
	}

	return results, nil
}

func (m *SimpleMockVectorStore) TraverseGraph(ctx context.Context, startChunkID string, maxDepth int, relationTypes []types.RelationType) (*types.GraphTraversalResult, error) {
	return &types.GraphTraversalResult{
		Paths: []types.GraphPath{},
		Nodes: []types.GraphNode{},
		Edges: []types.GraphEdge{},
	}, nil
}

func (m *SimpleMockVectorStore) UpdateRelationship(ctx context.Context, relationshipID string, confidence float64, factors types.ConfidenceFactors) error {
	rel, exists := m.relationships[relationshipID]
	if !exists {
		return errors.New("relationship not found")
	}

	rel.Confidence = confidence
	return nil
}

func (m *SimpleMockVectorStore) DeleteRelationship(ctx context.Context, relationshipID string) error {
	if _, exists := m.relationships[relationshipID]; !exists {
		return errors.New("relationship not found")
	}
	delete(m.relationships, relationshipID)
	return nil
}

func (m *SimpleMockVectorStore) GetRelationshipByID(ctx context.Context, relationshipID string) (*types.MemoryRelationship, error) {
	rel, exists := m.relationships[relationshipID]
	if !exists {
		return nil, errors.New("relationship not found")
	}
	return rel, nil
}
