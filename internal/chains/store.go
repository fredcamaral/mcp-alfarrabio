package chains

import (
	"context"
	"fmt"
	"sync"
)

// InMemoryChainStore implements ChainStore using in-memory storage
type InMemoryChainStore struct {
	mu     sync.RWMutex
	chains map[string]*MemoryChain
	// Index for chunk ID to chain IDs mapping
	chunkIndex map[string][]string
}

// NewInMemoryChainStore creates a new in-memory chain store
func NewInMemoryChainStore() *InMemoryChainStore {
	return &InMemoryChainStore{
		chains:     make(map[string]*MemoryChain),
		chunkIndex: make(map[string][]string),
	}
}

// StoreChain stores a memory chain
func (s *InMemoryChainStore) StoreChain(ctx context.Context, chain *MemoryChain) error {
	if chain == nil || chain.ID == "" {
		return fmt.Errorf("invalid chain")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Store chain
	s.chains[chain.ID] = chain

	// Update chunk index
	for _, chunkID := range chain.ChunkIDs {
		if _, exists := s.chunkIndex[chunkID]; !exists {
			s.chunkIndex[chunkID] = []string{}
		}
		// Add chain ID if not already present
		found := false
		for _, existingChainID := range s.chunkIndex[chunkID] {
			if existingChainID == chain.ID {
				found = true
				break
			}
		}
		if !found {
			s.chunkIndex[chunkID] = append(s.chunkIndex[chunkID], chain.ID)
		}
	}

	return nil
}

// GetChain retrieves a chain by ID
func (s *InMemoryChainStore) GetChain(ctx context.Context, chainID string) (*MemoryChain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chain, exists := s.chains[chainID]
	if !exists {
		return nil, fmt.Errorf("chain not found: %s", chainID)
	}

	// Return a copy to prevent external modifications
	return s.copyChain(chain), nil
}

// GetChainsByChunkID retrieves all chains containing a specific chunk
func (s *InMemoryChainStore) GetChainsByChunkID(ctx context.Context, chunkID string) ([]*MemoryChain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chainIDs, exists := s.chunkIndex[chunkID]
	if !exists {
		return []*MemoryChain{}, nil
	}

	chains := make([]*MemoryChain, 0, len(chainIDs))
	for _, chainID := range chainIDs {
		if chain, exists := s.chains[chainID]; exists {
			chains = append(chains, s.copyChain(chain))
		}
	}

	return chains, nil
}

// UpdateChain updates an existing chain
func (s *InMemoryChainStore) UpdateChain(ctx context.Context, chain *MemoryChain) error {
	if chain == nil || chain.ID == "" {
		return fmt.Errorf("invalid chain")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldChain, exists := s.chains[chain.ID]
	if !exists {
		return fmt.Errorf("chain not found: %s", chain.ID)
	}

	// Update chunk index - remove old entries
	for _, chunkID := range oldChain.ChunkIDs {
		if chainIDs, exists := s.chunkIndex[chunkID]; exists {
			newChainIDs := []string{}
			for _, id := range chainIDs {
				if id != chain.ID {
					newChainIDs = append(newChainIDs, id)
				}
			}
			if len(newChainIDs) == 0 {
				delete(s.chunkIndex, chunkID)
			} else {
				s.chunkIndex[chunkID] = newChainIDs
			}
		}
	}

	// Update chunk index - add new entries
	for _, chunkID := range chain.ChunkIDs {
		if _, exists := s.chunkIndex[chunkID]; !exists {
			s.chunkIndex[chunkID] = []string{}
		}
		found := false
		for _, existingChainID := range s.chunkIndex[chunkID] {
			if existingChainID == chain.ID {
				found = true
				break
			}
		}
		if !found {
			s.chunkIndex[chunkID] = append(s.chunkIndex[chunkID], chain.ID)
		}
	}

	// Store updated chain
	s.chains[chain.ID] = chain

	return nil
}

// DeleteChain deletes a chain
func (s *InMemoryChainStore) DeleteChain(ctx context.Context, chainID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	chain, exists := s.chains[chainID]
	if !exists {
		return fmt.Errorf("chain not found: %s", chainID)
	}

	// Remove from chunk index
	for _, chunkID := range chain.ChunkIDs {
		if chainIDs, exists := s.chunkIndex[chunkID]; exists {
			newChainIDs := []string{}
			for _, id := range chainIDs {
				if id != chainID {
					newChainIDs = append(newChainIDs, id)
				}
			}
			if len(newChainIDs) == 0 {
				delete(s.chunkIndex, chunkID)
			} else {
				s.chunkIndex[chunkID] = newChainIDs
			}
		}
	}

	// Delete chain
	delete(s.chains, chainID)

	return nil
}

// ListChains lists all chains with pagination
func (s *InMemoryChainStore) ListChains(ctx context.Context, limit, offset int) ([]*MemoryChain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get all chain IDs and sort by creation time
	chains := make([]*MemoryChain, 0, len(s.chains))
	for _, chain := range s.chains {
		chains = append(chains, s.copyChain(chain))
	}

	// Sort by creation time (newest first)
	for i := 0; i < len(chains)-1; i++ {
		for j := i + 1; j < len(chains); j++ {
			if chains[j].CreatedAt.After(chains[i].CreatedAt) {
				chains[i], chains[j] = chains[j], chains[i]
			}
		}
	}

	// Apply pagination
	start := offset
	end := offset + limit
	if start > len(chains) {
		return []*MemoryChain{}, nil
	}
	if end > len(chains) {
		end = len(chains)
	}

	return chains[start:end], nil
}

// copyChain creates a deep copy of a chain
func (s *InMemoryChainStore) copyChain(chain *MemoryChain) *MemoryChain {
	if chain == nil {
		return nil
	}

	// Copy basic fields
	copy := &MemoryChain{
		ID:          chain.ID,
		Name:        chain.Name,
		Description: chain.Description,
		CreatedAt:   chain.CreatedAt,
		UpdatedAt:   chain.UpdatedAt,
	}

	// Copy chunk IDs
	copy.ChunkIDs = make([]string, len(chain.ChunkIDs))
	for i, id := range chain.ChunkIDs {
		copy.ChunkIDs[i] = id
	}

	// Copy links
	copy.Links = make([]ChainLink, len(chain.Links))
	for i, link := range chain.Links {
		copy.Links[i] = ChainLink{
			FromChunkID: link.FromChunkID,
			ToChunkID:   link.ToChunkID,
			Type:        link.Type,
			Strength:    link.Strength,
			CreatedAt:   link.CreatedAt,
		}
		// Copy metadata if present
		if link.Metadata != nil {
			copy.Links[i].Metadata = make(map[string]interface{})
			for k, v := range link.Metadata {
				copy.Links[i].Metadata[k] = v
			}
		}
	}

	// Copy metadata
	if chain.Metadata != nil {
		copy.Metadata = make(map[string]interface{})
		for k, v := range chain.Metadata {
			copy.Metadata[k] = v
		}
	}

	return copy
}

// GetStats returns statistics about the chain store
func (s *InMemoryChainStore) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalLinks := 0
	for _, chain := range s.chains {
		totalLinks += len(chain.Links)
	}

	avgChunksPerChain := 0.0
	if len(s.chains) > 0 {
		totalChunks := 0
		for _, chain := range s.chains {
			totalChunks += len(chain.ChunkIDs)
		}
		avgChunksPerChain = float64(totalChunks) / float64(len(s.chains))
	}

	return map[string]interface{}{
		"total_chains":        len(s.chains),
		"total_indexed_chunks": len(s.chunkIndex),
		"total_links":         totalLinks,
		"avg_chunks_per_chain": avgChunksPerChain,
	}
}