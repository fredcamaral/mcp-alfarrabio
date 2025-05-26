package chains

import (
	"context"
	"mcp-memory/pkg/types"
	"testing"
	"time"
)

// MockChainAnalyzer for testing
type MockChainAnalyzer struct {
	analyzeFunc     func(ctx context.Context, chunk1, chunk2 types.ConversationChunk) (ChainType, float64, error)
	suggestNameFunc func(ctx context.Context, chunks []types.ConversationChunk) (string, string, error)
}

func (m *MockChainAnalyzer) AnalyzeRelationship(ctx context.Context, chunk1, chunk2 types.ConversationChunk) (ChainType, float64, error) {
	if m.analyzeFunc != nil {
		return m.analyzeFunc(ctx, chunk1, chunk2)
	}
	return ChainTypeContinuation, 0.8, nil
}

func (m *MockChainAnalyzer) SuggestChainName(ctx context.Context, chunks []types.ConversationChunk) (string, string, error) {
	if m.suggestNameFunc != nil {
		return m.suggestNameFunc(ctx, chunks)
	}
	return "Test Chain", "Test Description", nil
}

func createTestChunks(n int) []types.ConversationChunk {
	chunks := make([]types.ConversationChunk, n)
	baseTime := time.Now()
	
	for i := 0; i < n; i++ {
		chunks[i] = types.ConversationChunk{
			ID:         generateChainID(),
			Content:    "Test content " + string(rune(i)),
			SessionID:  "test-session",
			Timestamp:  baseTime.Add(time.Duration(i) * time.Hour),
			Summary:    "Test summary",
			Type:       types.ChunkTypeDiscussion,
			Embeddings: make([]float64, 1536), // Mock embeddings
			Metadata: types.ChunkMetadata{
				Repository: "test-repo",
				Tags:       []string{"concept1", "concept2"},
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultyModerate,
			},
		}
	}
	
	return chunks
}

func TestChainBuilder_CreateChain(t *testing.T) {
	store := NewInMemoryChainStore()
	analyzer := &MockChainAnalyzer{}
	builder := NewChainBuilder(store, analyzer)
	
	ctx := context.Background()
	chunks := createTestChunks(3)
	
	chain, err := builder.CreateChain(ctx, "Test Chain", "Test Description", chunks)
	if err != nil {
		t.Fatalf("Failed to create chain: %v", err)
	}
	
	// Verify chain properties
	if chain.Name != "Test Chain" {
		t.Errorf("Expected name 'Test Chain', got %s", chain.Name)
	}
	
	if len(chain.ChunkIDs) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chain.ChunkIDs))
	}
	
	// Should have links between chunks (3 chunks = 3 possible links with strength > 0.5)
	if len(chain.Links) == 0 {
		t.Error("Expected at least one link between chunks")
	}
	
	// Verify chain was stored
	storedChain, err := store.GetChain(ctx, chain.ID)
	if err != nil {
		t.Fatalf("Failed to get stored chain: %v", err)
	}
	
	if storedChain.ID != chain.ID {
		t.Error("Stored chain ID doesn't match")
	}
}

func TestChainBuilder_AddToChain(t *testing.T) {
	store := NewInMemoryChainStore()
	analyzer := &MockChainAnalyzer{}
	builder := NewChainBuilder(store, analyzer)
	
	ctx := context.Background()
	initialChunks := createTestChunks(2)
	
	// Create initial chain
	chain, err := builder.CreateChain(ctx, "Test Chain", "Test Description", initialChunks)
	if err != nil {
		t.Fatalf("Failed to create chain: %v", err)
	}
	
	// Store original link count
	originalLinkCount := len(chain.Links)
	t.Logf("Initial chain has %d links", originalLinkCount)
	
	// Add new chunk
	newChunk := createTestChunks(1)[0]
	err = builder.AddToChain(ctx, chain.ID, newChunk, initialChunks)
	if err != nil {
		t.Fatalf("Failed to add to chain: %v", err)
	}
	
	// Verify chunk was added
	updatedChain, err := store.GetChain(ctx, chain.ID)
	if err != nil {
		t.Fatalf("Failed to get updated chain: %v", err)
	}
	
	if len(updatedChain.ChunkIDs) != 3 {
		t.Errorf("Expected 3 chunks after addition, got %d", len(updatedChain.ChunkIDs))
	}
	
	// Should have more links now (at least 2 new links from the new chunk to the initial chunks)
	t.Logf("Original links: %d, Updated links: %d", originalLinkCount, len(updatedChain.Links))
	expectedMinLinks := originalLinkCount + len(initialChunks)
	if len(updatedChain.Links) < expectedMinLinks {
		t.Errorf("Expected at least %d links after adding chunk (original %d + %d new). Got: %d", 
			expectedMinLinks, originalLinkCount, len(initialChunks), len(updatedChain.Links))
	}
}

func TestChainBuilder_GetRelatedChains(t *testing.T) {
	store := NewInMemoryChainStore()
	analyzer := &MockChainAnalyzer{}
	builder := NewChainBuilder(store, analyzer)
	
	ctx := context.Background()
	chunks := createTestChunks(3)
	
	// Create two chains sharing a chunk
	chain1, _ := builder.CreateChain(ctx, "Chain 1", "Description 1", chunks[:2])
	chain2, _ := builder.CreateChain(ctx, "Chain 2", "Description 2", chunks[1:])
	
	// Get chains related to the shared chunk
	relatedChains, err := builder.GetRelatedChains(ctx, chunks[1].ID)
	if err != nil {
		t.Fatalf("Failed to get related chains: %v", err)
	}
	
	if len(relatedChains) != 2 {
		t.Errorf("Expected 2 related chains, got %d", len(relatedChains))
	}
	
	// Verify both chains are returned
	foundChain1, foundChain2 := false, false
	for _, chain := range relatedChains {
		if chain.ID == chain1.ID {
			foundChain1 = true
		}
		if chain.ID == chain2.ID {
			foundChain2 = true
		}
	}
	
	if !foundChain1 || !foundChain2 {
		t.Error("Not all related chains were returned")
	}
}

func TestChainBuilder_MergeChains(t *testing.T) {
	store := NewInMemoryChainStore()
	analyzer := &MockChainAnalyzer{}
	builder := NewChainBuilder(store, analyzer)
	
	ctx := context.Background()
	chunks := createTestChunks(4)
	
	// Create two separate chains
	chain1, _ := builder.CreateChain(ctx, "Chain 1", "Description 1", chunks[:2])
	chain2, _ := builder.CreateChain(ctx, "Chain 2", "Description 2", chunks[2:])
	
	// Merge chains
	mergedChain, err := builder.MergeChains(ctx, chain1.ID, chain2.ID, "Merged Chain", "Merged Description")
	if err != nil {
		t.Fatalf("Failed to merge chains: %v", err)
	}
	
	// Verify merged chain properties
	if mergedChain.Name != "Merged Chain" {
		t.Errorf("Expected name 'Merged Chain', got %s", mergedChain.Name)
	}
	
	if len(mergedChain.ChunkIDs) != 4 {
		t.Errorf("Expected 4 chunks in merged chain, got %d", len(mergedChain.ChunkIDs))
	}
	
	// Check metadata
	mergedFrom, exists := mergedChain.Metadata["merged_from"].([]string)
	if !exists || len(mergedFrom) != 2 {
		t.Error("Merged chain should have metadata about source chains")
	}
}

func TestChainBuilder_GetChainPath(t *testing.T) {
	store := NewInMemoryChainStore()
	
	
	// Custom analyzer that creates a linear chain
	analyzer := &MockChainAnalyzer{
		analyzeFunc: func(ctx context.Context, chunk1, chunk2 types.ConversationChunk) (ChainType, float64, error) {
			// Map chunks to their indices based on content
			idx1 := -1
			idx2 := -1
			for i := 0; i < 4; i++ {
				if chunk1.Content == "Test content " + string(rune(i)) {
					idx1 = i
				}
				if chunk2.Content == "Test content " + string(rune(i)) {
					idx2 = i
				}
			}
			
			// Create strong links between consecutive chunks (0->1, 1->2, 2->3)
			if (idx1 >= 0 && idx2 >= 0) && (idx1 == idx2-1 || idx2 == idx1-1) {
				return ChainTypeContinuation, 0.9, nil
			}
			return ChainTypeReference, 0.3, nil // Weak link
		},
	}
	
	builder := NewChainBuilder(store, analyzer)
	
	ctx := context.Background()
	chunks := createTestChunks(4)
	
	// Create chain
	chain, _ := builder.CreateChain(ctx, "Path Test Chain", "Description", chunks)
	
	// Find path between first and last chunk
	path, err := builder.GetChainPath(ctx, chain.ID, chunks[0].ID, chunks[3].ID)
	if err != nil {
		t.Fatalf("Failed to get chain path: %v", err)
	}
	
	if path == nil || len(path) == 0 {
		t.Fatal("Expected to find a path, but got nil or empty path")
	}
	
	// Path should include at least start and end
	if len(path) < 2 {
		t.Errorf("Path should have at least 2 nodes, got %d", len(path))
	}
	
	if path[0] != chunks[0].ID {
		t.Errorf("Path should start with chunk[0].ID (%s), but starts with %s", chunks[0].ID, path[0])
	}
	
	if path[len(path)-1] != chunks[3].ID {
		t.Errorf("Path should end with chunk[3].ID (%s), but ends with %s", chunks[3].ID, path[len(path)-1])
	}
}

func TestInMemoryChainStore_Operations(t *testing.T) {
	store := NewInMemoryChainStore()
	ctx := context.Background()
	
	// Create test chain
	chain := &MemoryChain{
		ID:          "test-chain-1",
		Name:        "Test Chain",
		Description: "Test Description",
		ChunkIDs:    []string{"chunk1", "chunk2", "chunk3"},
		Links:       []ChainLink{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    map[string]interface{}{"test": true},
	}
	
	// Test Store
	err := store.StoreChain(ctx, chain)
	if err != nil {
		t.Fatalf("Failed to store chain: %v", err)
	}
	
	// Test Get
	retrieved, err := store.GetChain(ctx, chain.ID)
	if err != nil {
		t.Fatalf("Failed to get chain: %v", err)
	}
	
	if retrieved.ID != chain.ID {
		t.Error("Retrieved chain ID doesn't match")
	}
	
	// Test GetChainsByChunkID
	chains, err := store.GetChainsByChunkID(ctx, "chunk2")
	if err != nil {
		t.Fatalf("Failed to get chains by chunk ID: %v", err)
	}
	
	if len(chains) != 1 {
		t.Errorf("Expected 1 chain, got %d", len(chains))
	}
	
	// Test Update
	chain.Name = "Updated Chain"
	chain.ChunkIDs = append(chain.ChunkIDs, "chunk4")
	err = store.UpdateChain(ctx, chain)
	if err != nil {
		t.Fatalf("Failed to update chain: %v", err)
	}
	
	// Verify update
	updated, _ := store.GetChain(ctx, chain.ID)
	if updated.Name != "Updated Chain" {
		t.Error("Chain name not updated")
	}
	
	if len(updated.ChunkIDs) != 4 {
		t.Error("Chunk IDs not updated")
	}
	
	// Test List
	chains, err = store.ListChains(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list chains: %v", err)
	}
	
	if len(chains) != 1 {
		t.Errorf("Expected 1 chain in list, got %d", len(chains))
	}
	
	// Test Delete
	err = store.DeleteChain(ctx, chain.ID)
	if err != nil {
		t.Fatalf("Failed to delete chain: %v", err)
	}
	
	// Verify deletion
	_, err = store.GetChain(ctx, chain.ID)
	if err == nil {
		t.Error("Expected error when getting deleted chain")
	}
	
	// Test stats
	stats := store.GetStats()
	if stats["total_chains"].(int) != 0 {
		t.Error("Expected 0 chains after deletion")
	}
}