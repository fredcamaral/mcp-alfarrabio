package relationships

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"mcp-memory/pkg/types"
)

// RelationshipType defines the type of relationship between memories
type RelationshipType string

const (
	RelTypeParentChild   RelationshipType = "parent_child"
	RelTypeSupersedes    RelationshipType = "supersedes"
	RelTypeRelated       RelationshipType = "related"
	RelTypeContinuation  RelationshipType = "continuation"
	RelTypeAlternative   RelationshipType = "alternative"
	RelTypeReference     RelationshipType = "reference"
)

// Relationship represents a connection between two memories
type Relationship struct {
	ID           string           `json:"id"`
	FromChunkID  string           `json:"from_chunk_id"`
	ToChunkID    string           `json:"to_chunk_id"`
	Type         RelationshipType `json:"type"`
	Strength     float64          `json:"strength"` // 0.0 to 1.0
	Context      string           `json:"context,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	AutoDetected bool             `json:"auto_detected"`
}

// Manager handles memory relationships
type Manager struct {
	relationships map[string][]Relationship // indexed by chunk ID
	mu            sync.RWMutex
}

// NewManager creates a new relationship manager
func NewManager() *Manager {
	return &Manager{
		relationships: make(map[string][]Relationship),
	}
}

// AddRelationship creates a new relationship between chunks
func (m *Manager) AddRelationship(ctx context.Context, from, to string, relType RelationshipType, strength float64, context string) (*Relationship, error) {
	if from == "" || to == "" {
		return nil, fmt.Errorf("both from and to chunk IDs are required")
	}
	
	if from == to {
		return nil, fmt.Errorf("cannot create relationship to self")
	}
	
	if strength < 0 || strength > 1 {
		return nil, fmt.Errorf("strength must be between 0 and 1")
	}
	
	rel := &Relationship{
		ID:           fmt.Sprintf("%s_%s_%s_%d", from, to, relType, time.Now().UnixNano()),
		FromChunkID:  from,
		ToChunkID:    to,
		Type:         relType,
		Strength:     strength,
		Context:      context,
		CreatedAt:    time.Now().UTC(),
		AutoDetected: false,
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Add to both chunk's relationships for bidirectional lookup
	m.relationships[from] = append(m.relationships[from], *rel)
	
	// For bidirectional relationships and parent-child (for child lookup), also index by the target
	if relType == RelTypeRelated || relType == RelTypeContinuation || relType == RelTypeParentChild || relType == RelTypeSupersedes {
		m.relationships[to] = append(m.relationships[to], *rel)
	}
	
	return rel, nil
}

// GetRelationships returns all relationships for a chunk
func (m *Manager) GetRelationships(chunkID string) []Relationship {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.relationships[chunkID]
}

// GetRelationshipsByType returns relationships of a specific type for a chunk
func (m *Manager) GetRelationshipsByType(chunkID string, relType RelationshipType) []Relationship {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var filtered []Relationship
	for _, rel := range m.relationships[chunkID] {
		if rel.Type == relType {
			filtered = append(filtered, rel)
		}
	}
	return filtered
}

// FindParent returns the parent chunk ID if it exists
func (m *Manager) FindParent(chunkID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, rel := range m.relationships[chunkID] {
		if rel.Type == RelTypeParentChild && rel.ToChunkID == chunkID {
			return rel.FromChunkID, true
		}
	}
	return "", false
}

// FindChildren returns all child chunk IDs
func (m *Manager) FindChildren(chunkID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var children []string
	for _, rel := range m.relationships[chunkID] {
		if rel.Type == RelTypeParentChild && rel.FromChunkID == chunkID {
			children = append(children, rel.ToChunkID)
		}
	}
	return children
}

// FindSupersededBy returns the chunk that supersedes this one
func (m *Manager) FindSupersededBy(chunkID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, rel := range m.relationships[chunkID] {
		if rel.Type == RelTypeSupersedes && rel.ToChunkID == chunkID {
			return rel.FromChunkID, true
		}
	}
	return "", false
}

// DetectRelationships analyzes chunks to automatically detect relationships
func (m *Manager) DetectRelationships(ctx context.Context, newChunk *types.ConversationChunk, existingChunks []types.ConversationChunk) []Relationship {
	var detected []Relationship
	
	// Check for continuation (same session, close in time)
	for _, existing := range existingChunks {
		if existing.SessionID == newChunk.SessionID {
			timeDiff := newChunk.Timestamp.Sub(existing.Timestamp)
			if timeDiff > 0 && timeDiff < 30*time.Minute {
				rel := Relationship{
					ID:           fmt.Sprintf("auto_%s_%s_%d", existing.ID, newChunk.ID, time.Now().UnixNano()),
					FromChunkID:  existing.ID,
					ToChunkID:    newChunk.ID,
					Type:         RelTypeContinuation,
					Strength:     1.0 - (timeDiff.Minutes() / 30.0), // Stronger if closer in time
					Context:      "Same session continuation",
					CreatedAt:    time.Now().UTC(),
					AutoDetected: true,
				}
				detected = append(detected, rel)
			}
		}
		
		// Check for superseding (same problem, newer solution)
		if existing.Type == types.ChunkTypeProblem && newChunk.Type == types.ChunkTypeSolution {
			// Could use similarity scoring here
			if existing.Metadata.Repository == newChunk.Metadata.Repository {
				rel := Relationship{
					ID:           fmt.Sprintf("auto_%s_%s_%d", newChunk.ID, existing.ID, time.Now().UnixNano()),
					FromChunkID:  newChunk.ID,
					ToChunkID:    existing.ID,
					Type:         RelTypeSupersedes,
					Strength:     0.7,
					Context:      "Newer solution for similar problem",
					CreatedAt:    time.Now().UTC(),
					AutoDetected: true,
				}
				detected = append(detected, rel)
			}
		}
		
		// Check for related by tags
		commonTags := findCommonTags(existing.Metadata.Tags, newChunk.Metadata.Tags)
		if len(commonTags) >= 2 {
			strength := float64(len(commonTags)) / float64(len(newChunk.Metadata.Tags))
			rel := Relationship{
				ID:           fmt.Sprintf("auto_%s_%s_%d", existing.ID, newChunk.ID, time.Now().UnixNano()),
				FromChunkID:  existing.ID,
				ToChunkID:    newChunk.ID,
				Type:         RelTypeRelated,
				Strength:     strength,
				Context:      fmt.Sprintf("Common tags: %v", commonTags),
				CreatedAt:    time.Now().UTC(),
				AutoDetected: true,
			}
			detected = append(detected, rel)
		}
	}
	
	// Store detected relationships
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for _, rel := range detected {
		m.relationships[rel.FromChunkID] = append(m.relationships[rel.FromChunkID], rel)
		if rel.Type == RelTypeRelated || rel.Type == RelTypeContinuation {
			m.relationships[rel.ToChunkID] = append(m.relationships[rel.ToChunkID], rel)
		}
	}
	
	return detected
}

// BuildRelationshipGraph creates a graph structure of related memories
func (m *Manager) BuildRelationshipGraph(rootChunkID string, maxDepth int) map[string][]Relationship {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	graph := make(map[string][]Relationship)
	visited := make(map[string]bool)
	
	m.traverseGraph(rootChunkID, graph, visited, 0, maxDepth)
	
	return graph
}

// traverseGraph recursively builds the relationship graph
func (m *Manager) traverseGraph(chunkID string, graph map[string][]Relationship, visited map[string]bool, depth, maxDepth int) {
	if depth > maxDepth || visited[chunkID] {
		return
	}
	
	visited[chunkID] = true
	rels := m.relationships[chunkID]
	graph[chunkID] = rels
	
	for _, rel := range rels {
		nextID := rel.ToChunkID
		if rel.ToChunkID == chunkID {
			nextID = rel.FromChunkID
		}
		m.traverseGraph(nextID, graph, visited, depth+1, maxDepth)
	}
}

// findCommonTags returns tags that appear in both slices
func findCommonTags(tags1, tags2 []string) []string {
	tagMap := make(map[string]bool)
	for _, tag := range tags1 {
		tagMap[tag] = true
	}
	
	var common []string
	for _, tag := range tags2 {
		if tagMap[tag] {
			common = append(common, tag)
		}
	}
	return common
}