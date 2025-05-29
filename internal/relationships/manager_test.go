package relationships

import (
	"context"
	"testing"
	"time"

	"mcp-memory/pkg/types"
)

func TestManager_AddRelationship(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	tests := []struct {
		name     string
		from     string
		to       string
		relType  RelationshipType
		strength float64
		context  string
		wantErr  bool
	}{
		{
			name:     "Valid parent-child relationship",
			from:     "chunk1",
			to:       "chunk2",
			relType:  RelTypeParentChild,
			strength: 0.9,
			context:  "chunk2 is a follow-up to chunk1",
			wantErr:  false,
		},
		{
			name:     "Empty from ID",
			from:     "",
			to:       "chunk2",
			relType:  RelTypeRelated,
			strength: 0.5,
			wantErr:  true,
		},
		{
			name:     "Self relationship",
			from:     "chunk1",
			to:       "chunk1",
			relType:  RelTypeRelated,
			strength: 0.5,
			wantErr:  true,
		},
		{
			name:     "Invalid strength",
			from:     "chunk1",
			to:       "chunk2",
			relType:  RelTypeRelated,
			strength: 1.5,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel, err := manager.AddRelationship(ctx, tt.from, tt.to, tt.relType, tt.strength, tt.context)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddRelationship() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && rel != nil {
				if rel.FromChunkID != tt.from {
					t.Errorf("FromChunkID = %v, want %v", rel.FromChunkID, tt.from)
				}
				if rel.ToChunkID != tt.to {
					t.Errorf("ToChunkID = %v, want %v", rel.ToChunkID, tt.to)
				}
				if rel.Type != tt.relType {
					t.Errorf("Type = %v, want %v", rel.Type, tt.relType)
				}
				if rel.Strength != tt.strength {
					t.Errorf("Strength = %v, want %v", rel.Strength, tt.strength)
				}
			}
		})
	}
}

func TestManager_FindParentAndChildren(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	// Create parent-child relationships
	_, err := manager.AddRelationship(ctx, "parent", "child1", RelTypeParentChild, 1.0, "")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	_, err = manager.AddRelationship(ctx, "parent", "child2", RelTypeParentChild, 1.0, "")
	if err != nil {
		t.Fatalf("Failed to add relationship: %v", err)
	}

	// Test FindParent
	parent, found := manager.FindParent("child1")
	if !found || parent != "parent" {
		t.Errorf("FindParent() = %v, %v; want parent, true", parent, found)
	}

	// Test FindChildren
	children := manager.FindChildren("parent")
	if len(children) != 2 {
		t.Errorf("FindChildren() returned %d children, want 2", len(children))
	}

	// Check both children are found
	foundChild1, foundChild2 := false, false
	for _, child := range children {
		if child == "child1" {
			foundChild1 = true
		}
		if child == "child2" {
			foundChild2 = true
		}
	}

	if !foundChild1 || !foundChild2 {
		t.Errorf("FindChildren() missing expected children")
	}
}

func TestManager_DetectRelationships(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	// Create test chunks
	baseTime := time.Now()

	existingChunks := []types.ConversationChunk{
		{
			ID:        "chunk1",
			SessionID: "session1",
			Timestamp: baseTime.Add(-10 * time.Minute),
			Type:      types.ChunkTypeProblem,
			Metadata: types.ChunkMetadata{
				Repository: "repo1",
				Tags:       []string{"bug", "performance", "database"},
			},
		},
		{
			ID:        "chunk2",
			SessionID: "session1",
			Timestamp: baseTime.Add(-5 * time.Minute),
			Type:      types.ChunkTypeDiscussion,
			Metadata: types.ChunkMetadata{
				Repository: "repo1",
				Tags:       []string{"architecture", "design"},
			},
		},
		{
			ID:        "chunk3",
			SessionID: "session2",
			Timestamp: baseTime.Add(-1 * time.Hour),
			Type:      types.ChunkTypeProblem,
			Metadata: types.ChunkMetadata{
				Repository: "repo1",
				Tags:       []string{"bug", "security"},
			},
		},
	}

	newChunk := &types.ConversationChunk{
		ID:        "chunk4",
		SessionID: "session1",
		Timestamp: baseTime,
		Type:      types.ChunkTypeSolution,
		Metadata: types.ChunkMetadata{
			Repository: "repo1",
			Tags:       []string{"bug", "performance", "fix"},
		},
	}

	// Detect relationships
	detected := manager.DetectRelationships(ctx, newChunk, existingChunks)

	// Should detect at least:
	// 1. Continuation from chunk2 (same session, recent)
	// 2. Supersedes chunk1 (problem -> solution)
	// 3. Related to chunk1 (common tags: bug, performance)

	if len(detected) < 2 {
		t.Errorf("DetectRelationships() detected %d relationships, want at least 2", len(detected))
	}

	// Check for continuation relationship
	foundContinuation := false
	foundSupersedes := false

	for _, rel := range detected {
		if rel.Type == RelTypeContinuation && rel.FromChunkID == "chunk2" {
			foundContinuation = true
		}
		if rel.Type == RelTypeSupersedes && rel.ToChunkID == "chunk1" {
			foundSupersedes = true
		}

		// All detected relationships should be marked as auto-detected
		if !rel.AutoDetected {
			t.Error("Detected relationship not marked as auto-detected")
		}
	}

	if !foundContinuation {
		t.Error("Failed to detect continuation relationship")
	}
	if !foundSupersedes {
		t.Error("Failed to detect supersedes relationship")
	}
}

func TestManager_BuildRelationshipGraph(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	// Create a graph: A -> B -> C
	//                 \-> D
	_, _ = manager.AddRelationship(ctx, "A", "B", RelTypeParentChild, 1.0, "")
	_, _ = manager.AddRelationship(ctx, "A", "D", RelTypeParentChild, 1.0, "")
	_, _ = manager.AddRelationship(ctx, "B", "C", RelTypeParentChild, 1.0, "")
	_, _ = manager.AddRelationship(ctx, "C", "D", RelTypeRelated, 0.5, "")

	// Build graph from A with max depth 2
	graph := manager.BuildRelationshipGraph("A", 2)

	// Should have entries for A, B, and D (depth 1) and C (depth 2)
	if len(graph) < 3 {
		t.Errorf("BuildRelationshipGraph() returned %d nodes, want at least 3", len(graph))
	}

	// Check A has 2 relationships
	if len(graph["A"]) != 2 {
		t.Errorf("Node A has %d relationships, want 2", len(graph["A"]))
	}
}

func TestFindCommonTags(t *testing.T) {
	tests := []struct {
		name  string
		tags1 []string
		tags2 []string
		want  []string
	}{
		{
			name:  "Some common tags",
			tags1: []string{"bug", "performance", "database"},
			tags2: []string{"bug", "fix", "performance"},
			want:  []string{"bug", "performance"},
		},
		{
			name:  "No common tags",
			tags1: []string{"frontend", "ui"},
			tags2: []string{"backend", "api"},
			want:  []string{},
		},
		{
			name:  "All tags common",
			tags1: []string{"bug", "critical"},
			tags2: []string{"bug", "critical"},
			want:  []string{"bug", "critical"},
		},
		{
			name:  "Empty slices",
			tags1: []string{},
			tags2: []string{"bug"},
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findCommonTags(tt.tags1, tt.tags2)

			if len(got) != len(tt.want) {
				t.Errorf("findCommonTags() returned %v, want %v", got, tt.want)
				return
			}

			// Check all expected tags are present
			for _, wantTag := range tt.want {
				found := false
				for _, gotTag := range got {
					if gotTag == wantTag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected tag %s not found in result", wantTag)
				}
			}
		})
	}
}
