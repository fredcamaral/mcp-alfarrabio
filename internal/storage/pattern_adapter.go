// Package storage provides vector database and storage abstractions.
// It includes Qdrant integration, circuit breakers, retry logic, and storage interfaces.
package storage

import (
	"context"
	"fmt"
	"lerian-mcp-memory/internal/intelligence"
	"lerian-mcp-memory/pkg/types"
)

// PatternStorageAdapter adapts VectorStore to PatternStorage interface
type PatternStorageAdapter struct {
	store VectorStore
}

// NewPatternStorageAdapter creates a new pattern storage adapter
func NewPatternStorageAdapter(store VectorStore) intelligence.PatternStorage {
	return &PatternStorageAdapter{
		store: store,
	}
}

// StorePattern stores a pattern (converted to chunk format)
func (p *PatternStorageAdapter) StorePattern(ctx context.Context, pattern *intelligence.Pattern) error {
	// Convert pattern to conversation chunk for storage
	chunk := types.ConversationChunk{
		ID:        pattern.ID,
		SessionID: "pattern-system",
		Type:      types.ChunkTypeAnalysis, // Use analysis type for patterns
		Content:   fmt.Sprintf("Pattern: %s - %s", pattern.Name, pattern.Description),
		Summary:   pattern.Description,
		Metadata: types.ChunkMetadata{
			Repository: "patterns",
			Tags:       []string{"pattern", string(pattern.Type)},
		},
		// Note: Patterns don't have embeddings in the current implementation
		Embeddings: []float64{},
	}

	return p.store.Store(ctx, &chunk)
}

// GetPattern gets a pattern by ID
func (p *PatternStorageAdapter) GetPattern(ctx context.Context, id string) (*intelligence.Pattern, error) {
	chunk, err := p.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert chunk back to pattern (simplified)
	pattern := &intelligence.Pattern{
		ID:          chunk.ID,
		Name:        "Pattern_" + chunk.ID[:8],
		Description: chunk.Summary,
		Type:        intelligence.PatternTypeWorkflow, // Default type
	}

	return pattern, nil
}

// ListPatterns lists patterns by type
func (p *PatternStorageAdapter) ListPatterns(ctx context.Context, patternType *intelligence.PatternType) ([]intelligence.Pattern, error) {
	// Get all chunks from the patterns repository
	chunks, err := p.store.ListByRepository(ctx, "patterns", 1000, 0)
	if err != nil {
		return nil, err
	}

	patterns := make([]intelligence.Pattern, 0, len(chunks))
	for i := range chunks {
		chunk := chunks[i]
		pattern := intelligence.Pattern{
			ID:          chunk.ID,
			Name:        "Pattern_" + chunk.ID[:8],
			Description: chunk.Summary,
			Type:        intelligence.PatternTypeWorkflow, // Default type
		}
		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// UpdatePattern updates an existing pattern
func (p *PatternStorageAdapter) UpdatePattern(ctx context.Context, pattern *intelligence.Pattern) error {
	// Convert pattern to chunk and update
	chunk := types.ConversationChunk{
		ID:        pattern.ID,
		SessionID: "pattern-system",
		Type:      types.ChunkTypeAnalysis, // Use analysis type for patterns
		Content:   fmt.Sprintf("Pattern: %s - %s", pattern.Name, pattern.Description),
		Summary:   pattern.Description,
		Metadata: types.ChunkMetadata{
			Repository: "patterns",
			Tags:       []string{"pattern", string(pattern.Type)},
		},
		Embeddings: []float64{},
	}

	return p.store.Update(ctx, &chunk)
}

// DeletePattern deletes a pattern by ID
func (p *PatternStorageAdapter) DeletePattern(ctx context.Context, id string) error {
	return p.store.Delete(ctx, id)
}

// SearchPatterns searches for patterns
func (p *PatternStorageAdapter) SearchPatterns(ctx context.Context, query string, limit int) ([]intelligence.Pattern, error) {
	// For now, return empty list as pattern search requires more complex implementation
	// This would need embedding generation and proper vector search
	return []intelligence.Pattern{}, nil
}

// StoreOccurrence stores a pattern occurrence (not implemented in adapter)
func (p *PatternStorageAdapter) StoreOccurrence(ctx context.Context, occurrence *intelligence.PatternOccurrence) error {
	// Pattern occurrences are not supported in the vector store adapter
	return nil
}

// GetOccurrences retrieves occurrences for a pattern (not implemented in adapter)
func (p *PatternStorageAdapter) GetOccurrences(ctx context.Context, patternID string, limit int) ([]intelligence.PatternOccurrence, error) {
	// Pattern occurrences are not supported in the vector store adapter
	return []intelligence.PatternOccurrence{}, nil
}

// StoreRelationship stores a pattern relationship (not implemented in adapter)
func (p *PatternStorageAdapter) StoreRelationship(ctx context.Context, relationship *intelligence.PatternRelationship) error {
	// Pattern relationships are not supported in the vector store adapter
	return nil
}

// GetRelationships retrieves relationships for a pattern (not implemented in adapter)
func (p *PatternStorageAdapter) GetRelationships(ctx context.Context, patternID string) ([]intelligence.PatternRelationship, error) {
	// Pattern relationships are not supported in the vector store adapter
	return []intelligence.PatternRelationship{}, nil
}

// UpdateConfidence updates pattern confidence based on feedback (not implemented in adapter)
func (p *PatternStorageAdapter) UpdateConfidence(ctx context.Context, patternID string, isPositive bool) error {
	// Confidence updates are not supported in the vector store adapter
	return nil
}

// GetPatternStatistics retrieves pattern statistics (not implemented in adapter)
func (p *PatternStorageAdapter) GetPatternStatistics(ctx context.Context) (map[string]interface{}, error) {
	// Statistics are not supported in the vector store adapter
	return map[string]interface{}{
		"total_patterns": 0,
		"message":        "Pattern statistics not available in vector store adapter",
	}, nil
}
