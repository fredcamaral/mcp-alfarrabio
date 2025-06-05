// Package chains provides memory chain functionality for linking related memories
package chains

import (
	"context"
	"fmt"
	"lerian-mcp-memory/pkg/types"
	"sort"
	"sync"
	"time"
)

// ChainType represents the type of relationship between memories
type ChainType string

const (
	// ChainTypeContinuation represents a chain where memories continue from each other
	ChainTypeContinuation ChainType = "continuation"
	// ChainTypeSolution represents a chain from problem to solution
	ChainTypeSolution ChainType = "solution"
	// ChainTypeReference represents a chain of references between memories
	ChainTypeReference ChainType = "reference"
	// ChainTypeEvolution represents how memories evolve over time
	ChainTypeEvolution ChainType = "evolution"
	// ChainTypeConflict represents conflicting memories
	ChainTypeConflict ChainType = "conflict"
	// ChainTypeSupport represents supporting evidence or context
	ChainTypeSupport ChainType = "support"
	// ChainTypeThread represents memory thread grouping
	ChainTypeThread ChainType = "thread" // Memory thread grouping
	// ChainTypeWorkflow represents workflow-based chains
	ChainTypeWorkflow ChainType = "workflow" // Workflow-based chains
)

// ChainLink represents a link between two memories
type ChainLink struct {
	FromChunkID string                 `json:"from_chunk_id"`
	ToChunkID   string                 `json:"to_chunk_id"`
	Type        ChainType              `json:"type"`
	Strength    float64                `json:"strength"` // 0.0 to 1.0
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MemoryChain represents a chain of related memories
type MemoryChain struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	ChunkIDs    []string               `json:"chunk_ids"`
	Links       []ChainLink            `json:"links"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ChainStore interface for persisting chains
type ChainStore interface {
	StoreChain(ctx context.Context, chain *MemoryChain) error
	GetChain(ctx context.Context, chainID string) (*MemoryChain, error)
	GetChainsByChunkID(ctx context.Context, chunkID string) ([]*MemoryChain, error)
	UpdateChain(ctx context.Context, chain *MemoryChain) error
	DeleteChain(ctx context.Context, chainID string) error
	ListChains(ctx context.Context, limit, offset int) ([]*MemoryChain, error)
}

// ChainBuilder builds memory chains
type ChainBuilder struct {
	store    ChainStore
	analyzer ChainAnalyzer
	mu       sync.RWMutex
	chains   map[string]*MemoryChain
}

// ChainAnalyzer analyzes relationships between memories
type ChainAnalyzer interface {
	AnalyzeRelationship(ctx context.Context, chunk1, chunk2 *types.ConversationChunk) (ChainType, float64, error)
	SuggestChainName(ctx context.Context, chunks []*types.ConversationChunk) (string, string, error)
}

// NewChainBuilder creates a new chain builder
func NewChainBuilder(store ChainStore, analyzer ChainAnalyzer) *ChainBuilder {
	return &ChainBuilder{
		store:    store,
		analyzer: analyzer,
		chains:   make(map[string]*MemoryChain),
	}
}

// CreateChain creates a new memory chain
func (cb *ChainBuilder) CreateChain(ctx context.Context, name, description string, initialChunks []*types.ConversationChunk) (*MemoryChain, error) {
	if len(initialChunks) < 2 {
		return nil, fmt.Errorf("chain must have at least 2 chunks")
	}

	// Create chain
	chain := &MemoryChain{
		ID:          generateChainID(),
		Name:        name,
		Description: description,
		ChunkIDs:    extractChunkIDs(initialChunks),
		Links:       []ChainLink{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Analyze relationships between chunks
	for i := 0; i < len(initialChunks)-1; i++ {
		for j := i + 1; j < len(initialChunks); j++ {
			linkType, strength, err := cb.analyzer.AnalyzeRelationship(ctx, initialChunks[i], initialChunks[j])
			if err != nil {
				continue
			}

			if strength > 0.5 { // Only create strong links
				link := ChainLink{
					FromChunkID: initialChunks[i].ID,
					ToChunkID:   initialChunks[j].ID,
					Type:        linkType,
					Strength:    strength,
					CreatedAt:   time.Now(),
				}
				chain.Links = append(chain.Links, link)
			}
		}
	}

	// Store chain
	if err := cb.store.StoreChain(ctx, chain); err != nil {
		return nil, fmt.Errorf("failed to store chain: %w", err)
	}

	// Cache it
	cb.mu.Lock()
	cb.chains[chain.ID] = chain
	cb.mu.Unlock()

	return chain, nil
}

// AddToChain adds a chunk to an existing chain
func (cb *ChainBuilder) AddToChain(ctx context.Context, chainID string, chunk *types.ConversationChunk, relatedChunks []*types.ConversationChunk) error {
	// Get chain
	chain, err := cb.getOrLoadChain(ctx, chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain: %w", err)
	}

	// Check if chunk already in chain
	for _, id := range chain.ChunkIDs {
		if id == chunk.ID {
			return fmt.Errorf("chunk already in chain")
		}
	}

	// Add chunk ID
	chain.ChunkIDs = append(chain.ChunkIDs, chunk.ID)

	// Analyze relationships with specified related chunks
	for i := range relatedChunks {
		linkType, strength, err := cb.analyzer.AnalyzeRelationship(ctx, chunk, relatedChunks[i])
		if err != nil {
			continue
		}

		if strength > 0.5 {
			link := ChainLink{
				FromChunkID: relatedChunks[i].ID,
				ToChunkID:   chunk.ID,
				Type:        linkType,
				Strength:    strength,
				CreatedAt:   time.Now(),
			}
			chain.Links = append(chain.Links, link)
		}
	}

	// Update chain
	chain.UpdatedAt = time.Now()
	if err := cb.store.UpdateChain(ctx, chain); err != nil {
		return fmt.Errorf("failed to update chain: %w", err)
	}

	return nil
}

// AutoCreateChain automatically creates chains based on chunk relationships
func (cb *ChainBuilder) AutoCreateChain(ctx context.Context, chunks []*types.ConversationChunk) (*MemoryChain, error) {
	if len(chunks) < 2 {
		return nil, fmt.Errorf("need at least 2 chunks for auto-chain creation")
	}

	// Get suggested name and description
	name, description, err := cb.analyzer.SuggestChainName(ctx, chunks)
	if err != nil {
		// Fallback name
		name = fmt.Sprintf("Chain-%s", time.Now().Format("2006-01-02-15:04"))
		description = "Automatically created memory chain"
	}

	return cb.CreateChain(ctx, name, description, chunks)
}

// GetRelatedChains gets all chains containing a specific chunk
func (cb *ChainBuilder) GetRelatedChains(ctx context.Context, chunkID string) ([]*MemoryChain, error) {
	return cb.store.GetChainsByChunkID(ctx, chunkID)
}

// GetChainPath finds the path between two chunks in a chain
func (cb *ChainBuilder) GetChainPath(ctx context.Context, chainID, fromChunkID, toChunkID string) ([]string, error) {
	chain, err := cb.getOrLoadChain(ctx, chainID)
	if err != nil {
		return nil, err
	}

	// Build adjacency list
	graph := make(map[string][]string)
	for _, link := range chain.Links {
		graph[link.FromChunkID] = append(graph[link.FromChunkID], link.ToChunkID)
		// For undirected relationships
		if link.Type == ChainTypeReference || link.Type == ChainTypeSupport {
			graph[link.ToChunkID] = append(graph[link.ToChunkID], link.FromChunkID)
		}
	}

	// BFS to find path
	path := cb.bfsPath(graph, fromChunkID, toChunkID)
	return path, nil
}

// GetStrongestLinks gets the strongest links in a chain
func (cb *ChainBuilder) GetStrongestLinks(ctx context.Context, chainID string, limit int) ([]ChainLink, error) {
	chain, err := cb.getOrLoadChain(ctx, chainID)
	if err != nil {
		return nil, err
	}

	// Sort by strength
	links := make([]ChainLink, len(chain.Links))
	copy(links, chain.Links)
	sort.Slice(links, func(i, j int) bool {
		return links[i].Strength > links[j].Strength
	})

	if limit > 0 && limit < len(links) {
		links = links[:limit]
	}

	return links, nil
}

// MergeChains merges two chains into one
func (cb *ChainBuilder) MergeChains(ctx context.Context, chainID1, chainID2, newName, newDescription string) (*MemoryChain, error) {
	// Get both chains
	chain1, err := cb.getOrLoadChain(ctx, chainID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain1: %w", err)
	}

	chain2, err := cb.getOrLoadChain(ctx, chainID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain2: %w", err)
	}

	// Merge chunk IDs (unique)
	chunkIDSet := make(map[string]bool)
	for _, id := range chain1.ChunkIDs {
		chunkIDSet[id] = true
	}
	for _, id := range chain2.ChunkIDs {
		chunkIDSet[id] = true
	}

	mergedChunkIDs := make([]string, 0, len(chunkIDSet))
	for id := range chunkIDSet {
		mergedChunkIDs = append(mergedChunkIDs, id)
	}

	// Merge links
	linkMap := make(map[string]ChainLink)
	for _, link := range chain1.Links {
		key := fmt.Sprintf("%s-%s", link.FromChunkID, link.ToChunkID)
		linkMap[key] = link
	}
	for _, link := range chain2.Links {
		key := fmt.Sprintf("%s-%s", link.FromChunkID, link.ToChunkID)
		if existing, exists := linkMap[key]; exists {
			// Keep the stronger link
			if link.Strength > existing.Strength {
				linkMap[key] = link
			}
		} else {
			linkMap[key] = link
		}
	}

	mergedLinks := make([]ChainLink, 0, len(linkMap))
	for _, link := range linkMap {
		mergedLinks = append(mergedLinks, link)
	}

	// Create new chain
	mergedChain := &MemoryChain{
		ID:          generateChainID(),
		Name:        newName,
		Description: newDescription,
		ChunkIDs:    mergedChunkIDs,
		Links:       mergedLinks,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata: map[string]interface{}{
			"merged_from": []string{chainID1, chainID2},
		},
	}

	// Store merged chain
	if err := cb.store.StoreChain(ctx, mergedChain); err != nil {
		return nil, fmt.Errorf("failed to store merged chain: %w", err)
	}

	// Optionally delete old chains
	// cb.store.DeleteChain(ctx, chainID1)
	// cb.store.DeleteChain(ctx, chainID2)

	return mergedChain, nil
}

// Helper functions

func (cb *ChainBuilder) getOrLoadChain(ctx context.Context, chainID string) (*MemoryChain, error) {
	cb.mu.RLock()
	chain, exists := cb.chains[chainID]
	cb.mu.RUnlock()

	if exists {
		return chain, nil
	}

	// Load from store
	chain, err := cb.store.GetChain(ctx, chainID)
	if err != nil {
		return nil, err
	}

	// Cache it
	cb.mu.Lock()
	cb.chains[chainID] = chain
	cb.mu.Unlock()

	return chain, nil
}

func (cb *ChainBuilder) bfsPath(graph map[string][]string, start, end string) []string {
	if start == end {
		return []string{start}
	}

	visited := make(map[string]bool)
	queue := [][]string{{start}}
	visited[start] = true

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]

		node := path[len(path)-1]

		for _, neighbor := range graph[node] {
			if neighbor == end {
				return append(path, neighbor)
			}

			if !visited[neighbor] {
				visited[neighbor] = true
				newPath := make([]string, len(path)+1)
				copy(newPath, path)
				newPath[len(path)] = neighbor
				queue = append(queue, newPath)
			}
		}
	}

	return nil // No path found
}

func extractChunkIDs(chunks []*types.ConversationChunk) []string {
	ids := make([]string, len(chunks))
	for i := range chunks {
		ids[i] = chunks[i].ID
	}
	return ids
}

func generateChainID() string {
	return fmt.Sprintf("chain_%d_%s", time.Now().Unix(), generateRandomString(8))
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
