package storage

import (
	"context"
	"fmt"
	"mcp-memory/internal/intelligence"
	"mcp-memory/pkg/types"
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
	for _, chunk := range chunks {
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
