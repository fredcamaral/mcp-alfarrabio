package intelligence

import (
	"context"
	"testing"
	"time"

	"mcp-memory/pkg/types"
)

func TestConflictResolver_GenerateResolutionStrategies(t *testing.T) {
	resolver := NewConflictResolver()

	// Create a sample architectural conflict
	conflict := Conflict{
		ID:          "test_conflict_1",
		Type:        ConflictTypeArchitectural,
		Severity:    SeverityHigh,
		Title:       "Architecture Decision Conflict",
		Description: "Conflicting decisions about microservices vs monolith",
		Confidence:  0.8,
		PrimaryChunk: types.ConversationChunk{
			ID:      "chunk1",
			Content: "We decided to use microservices",
			Type:    types.ChunkTypeArchitectureDecision,
		},
		ConflictChunk: types.ConversationChunk{
			ID:      "chunk2",
			Content: "We should use monolithic architecture",
			Type:    types.ChunkTypeArchitectureDecision,
		},
		TimeDifference: 2 * time.Hour,
		DetectedAt:     time.Now(),
	}

	strategies := resolver.GenerateResolutionStrategies(&conflict)

	if len(strategies) == 0 {
		t.Error("Expected at least one resolution strategy")
	}

	// Check that strategies are properly ordered by confidence
	for i := 1; i < len(strategies); i++ {
		if strategies[i].Confidence > strategies[i-1].Confidence {
			t.Errorf("Strategies not ordered by confidence: %f > %f at positions %d, %d",
				strategies[i].Confidence, strategies[i-1].Confidence, i, i-1)
		}
	}

	t.Logf("Generated %d resolution strategies for architectural conflict", len(strategies))
	for i, strategy := range strategies {
		t.Logf("Strategy %d: %s (confidence: %.2f)", i+1, strategy.Title, strategy.Confidence)
	}
}

func TestConflictResolver_ResolveConflicts(t *testing.T) {
	resolver := NewConflictResolver()
	ctx := context.Background()

	conflicts := []Conflict{
		{
			ID:          "conflict1",
			Type:        ConflictTypeArchitectural,
			Severity:    SeverityHigh,
			Title:       "Architecture Conflict",
			Description: "Microservices vs Monolith decision conflict",
			Confidence:  0.8,
			PrimaryChunk: types.ConversationChunk{
				ID: "chunk1",
				Metadata: types.ChunkMetadata{
					Repository: "test-repo",
				},
			},
			ConflictChunk: types.ConversationChunk{
				ID: "chunk2",
				Metadata: types.ChunkMetadata{
					Repository: "test-repo",
				},
			},
			TimeDifference: 2 * time.Hour,
			DetectedAt:     time.Now(),
		},
		{
			ID:          "conflict2",
			Type:        ConflictTypeTechnical,
			Severity:    SeverityMedium,
			Title:       "Technical Conflict",
			Description: "Different implementation approaches",
			Confidence:  0.7,
			PrimaryChunk: types.ConversationChunk{
				ID: "chunk3",
				Metadata: types.ChunkMetadata{
					Repository: "test-repo",
				},
			},
			ConflictChunk: types.ConversationChunk{
				ID: "chunk4",
				Metadata: types.ChunkMetadata{
					Repository: "test-repo",
				},
			},
			TimeDifference: 1 * time.Hour,
			DetectedAt:     time.Now(),
		},
	}

	recommendations, err := resolver.ResolveConflicts(ctx, conflicts)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(recommendations) != len(conflicts) {
		t.Errorf("Expected %d recommendations, got %d", len(conflicts), len(recommendations))
	}

	// Check that each recommendation has strategies
	for i, rec := range recommendations {
		if len(rec.Strategies) == 0 {
			t.Errorf("Recommendation %d has no strategies", i)
		}

		if rec.Recommended == nil {
			t.Errorf("Recommendation %d has no recommended strategy", i)
		}

		if rec.ConflictID != conflicts[i].ID {
			t.Errorf("Recommendation %d has wrong conflict ID: expected %s, got %s",
				i, conflicts[i].ID, rec.ConflictID)
		}

		t.Logf("Recommendation %d: %d strategies, recommended: %s",
			i+1, len(rec.Strategies), rec.Recommended.Title)
	}
}

func TestConflictResolver_ArchitecturalStrategies(t *testing.T) {
	resolver := NewConflictResolver()

	conflict := Conflict{
		Type:     ConflictTypeArchitectural,
		Severity: SeverityHigh,
		PrimaryChunk: types.ConversationChunk{
			Content: "Use microservices for scalability",
		},
		ConflictChunk: types.ConversationChunk{
			Content: "Use monolithic architecture for simplicity",
		},
	}

	strategies := resolver.generateArchitecturalStrategies(&conflict)

	if len(strategies) == 0 {
		t.Error("Expected architectural strategies")
	}

	// Check for expected strategy types
	expectedTypes := map[ConflictResolutionType]bool{
		ResolutionAcceptLatest: false,
		ResolutionMerge:        false,
		ResolutionContextual:   false,
	}

	for _, strategy := range strategies {
		if _, exists := expectedTypes[strategy.Type]; exists {
			expectedTypes[strategy.Type] = true
		}
	}

	foundTypes := 0
	for _, found := range expectedTypes {
		if found {
			foundTypes++
		}
	}

	if foundTypes == 0 {
		t.Error("No expected architectural strategy types found")
	}

	t.Logf("Found %d architectural strategies with %d expected types", len(strategies), foundTypes)
}

func TestConflictResolver_TechnicalStrategies(t *testing.T) {
	resolver := NewConflictResolver()

	conflict := Conflict{
		Type:     ConflictTypeTechnical,
		Severity: SeverityMedium,
		PrimaryChunk: types.ConversationChunk{
			Content: "Use Redis for caching",
		},
		ConflictChunk: types.ConversationChunk{
			Content: "Use Memcached for better performance",
		},
	}

	strategies := resolver.generateTechnicalStrategies(&conflict)

	if len(strategies) == 0 {
		t.Error("Expected technical strategies")
	}

	// Check for expected strategy types
	expectedTypes := map[ConflictResolutionType]bool{
		ResolutionAcceptHighest: false,
		ResolutionEvolutionary:  false,
	}

	for _, strategy := range strategies {
		if _, exists := expectedTypes[strategy.Type]; exists {
			expectedTypes[strategy.Type] = true
		}
	}

	foundTypes := 0
	for _, found := range expectedTypes {
		if found {
			foundTypes++
		}
	}

	t.Logf("Found %d technical strategies with %d expected types", len(strategies), foundTypes)
}

func TestConflictResolver_CalculateStrategyConfidence(t *testing.T) {
	resolver := NewConflictResolver()

	tests := []struct {
		conflict        Conflict
		strategy        ResolutionStrategy
		expectedMinimum float64
		name            string
	}{
		{
			conflict: Conflict{
				Type:       ConflictTypeArchitectural,
				Severity:   SeverityCritical,
				Confidence: 0.9,
			},
			strategy: ResolutionStrategy{
				Type: ResolutionManualReview,
			},
			expectedMinimum: 0.5,
			name:            "critical_conflict_manual_review",
		},
		{
			conflict: Conflict{
				Type:       ConflictTypeTechnical,
				Severity:   SeverityHigh,
				Confidence: 0.8,
			},
			strategy: ResolutionStrategy{
				Type: ResolutionAcceptHighest,
			},
			expectedMinimum: 0.5,
			name:            "technical_conflict_benchmarking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := resolver.calculateStrategyConfidence(&tt.conflict, &tt.strategy)

			if confidence < tt.expectedMinimum {
				t.Errorf("Expected confidence >= %.2f, got %.2f", tt.expectedMinimum, confidence)
			}

			if confidence > 1.0 {
				t.Errorf("Confidence should not exceed 1.0, got %.2f", confidence)
			}

			if confidence < 0.0 {
				t.Errorf("Confidence should not be negative, got %.2f", confidence)
			}
		})
	}
}

func TestConflictResolver_BuildConflictContext(t *testing.T) {
	resolver := NewConflictResolver()

	conflict := Conflict{
		PrimaryChunk: types.ConversationChunk{
			Metadata: types.ChunkMetadata{
				Repository:    "test-repo",
				FilesModified: []string{"file1.go", "file2.go"},
			},
		},
		ConflictChunk: types.ConversationChunk{
			Metadata: types.ChunkMetadata{
				Repository:    "test-repo",
				FilesModified: []string{"file2.go", "file3.go"},
			},
		},
		Type: ConflictTypeArchitectural,
	}

	conflictCtx := resolver.buildConflictContext(&conflict)

	if conflictCtx.Repository != "test-repo" {
		t.Errorf("Expected repository 'test-repo', got '%s'", conflictCtx.Repository)
	}

	expectedFiles := 3 // file1.go, file2.go, file3.go (file2.go should not be duplicated)
	if len(conflictCtx.AffectedFiles) != expectedFiles {
		t.Errorf("Expected %d affected files, got %d", expectedFiles, len(conflictCtx.AffectedFiles))
	}

	if len(conflictCtx.StakeholderImpact) == 0 {
		t.Error("Expected stakeholder impact to be populated")
	}

	// Check stakeholder impact for architectural conflicts
	if impact, exists := conflictCtx.StakeholderImpact["architects"]; !exists || impact != string(SeverityHigh) {
		t.Error("Expected high impact for architects on architectural conflicts")
	}
}

func TestConflictResolver_ExtractStakeholderImpact(t *testing.T) {
	tests := []struct {
		conflictType    ConflictType
		expectedImpacts map[string]string
		name            string
	}{
		{
			conflictType: ConflictTypeArchitectural,
			expectedImpacts: map[string]string{
				"architects": "high",
				"developers": "medium",
			},
			name: "architectural_conflict",
		},
		{
			conflictType: ConflictTypeTechnical,
			expectedImpacts: map[string]string{
				"developers": "high",
				"qa":         "medium",
			},
			name: "technical_conflict",
		},
		{
			conflictType: ConflictTypeDecision,
			expectedImpacts: map[string]string{
				"leadership": "high",
				"product":    "medium",
			},
			name: "decision_conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflict := Conflict{Type: tt.conflictType}
			impact := extractStakeholderImpact(&conflict)

			for role, expectedLevel := range tt.expectedImpacts {
				if actualLevel, exists := impact[role]; !exists {
					t.Errorf("Expected impact for %s role", role)
				} else if actualLevel != expectedLevel {
					t.Errorf("Expected %s impact for %s, got %s", expectedLevel, role, actualLevel)
				}
			}
		})
	}
}
