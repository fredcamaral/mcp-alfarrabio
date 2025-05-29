package storage

import (
	"testing"

	chromav2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"mcp-memory/pkg/types"
)

func TestBuildWhereFilter(t *testing.T) {
	tests := []struct {
		name     string
		query    types.MemoryQuery
		wantNil  bool
		validate func(t *testing.T, filter chromav2.WhereClause)
	}{
		{
			name: "empty query returns nil",
			query: types.MemoryQuery{
				Recency: types.RecencyAllTime,
			},
			wantNil: true,
		},
		{
			name: "repository only",
			query: types.MemoryQuery{
				Repository: stringPtr("test-repo"),
				Recency:    types.RecencyAllTime,
			},
			wantNil: false,
			validate: func(t *testing.T, filter chromav2.WhereClause) {
				if filter == nil {
					t.Error("expected non-nil filter")
				}
				// The filter should be an EqString for repository
				if filter.Key() != "repository" {
					t.Errorf("expected key 'repository', got %s", filter.Key())
				}
			},
		},
		{
			name: "types only",
			query: types.MemoryQuery{
				Types:   []types.ChunkType{types.ChunkTypeProblem, types.ChunkTypeSolution},
				Recency: types.RecencyAllTime,
			},
			wantNil: false,
			validate: func(t *testing.T, filter chromav2.WhereClause) {
				if filter == nil {
					t.Error("expected non-nil filter")
				}
				// The filter should be an InString for types
				if filter.Key() != "type" {
					t.Errorf("expected key 'type', got %s", filter.Key())
				}
			},
		},
		{
			name: "recency recent",
			query: types.MemoryQuery{
				Recency: types.RecencyRecent,
			},
			wantNil: false,
			validate: func(t *testing.T, filter chromav2.WhereClause) {
				if filter == nil {
					t.Error("expected non-nil filter")
				}
				// The filter should be a GteFloat for timestamp
				if filter.Key() != "timestamp" {
					t.Errorf("expected key 'timestamp', got %s", filter.Key())
				}
			},
		},
		{
			name: "multiple filters combined",
			query: types.MemoryQuery{
				Repository: stringPtr("test-repo"),
				Types:      []types.ChunkType{types.ChunkTypeProblem},
				Recency:    types.RecencyRecent,
			},
			wantNil: false,
			validate: func(t *testing.T, filter chromav2.WhereClause) {
				if filter == nil {
					t.Error("expected non-nil filter")
				}
				// Should be an AND clause combining multiple filters
				// The And clause doesn't expose its clauses directly,
				// but we can validate it exists and is valid
				if err := filter.Validate(); err != nil {
					t.Errorf("filter validation failed: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWhereFilter(tt.query)

			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil filter, got %v", got)
				}
				return
			}

			if got == nil {
				t.Error("expected non-nil filter, got nil")
				return
			}

			// Validate the filter
			if err := got.Validate(); err != nil {
				t.Errorf("filter validation failed: %v", err)
			}

			// Run custom validation if provided
			if tt.validate != nil {
				tt.validate(t, got)
			}

			// Log the filter string for debugging
			t.Logf("Filter string: %s", got.String())
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestWhereFilterJSON(t *testing.T) {
	// Test that our filters can be properly marshaled to JSON
	repo := "test-repo"
	query := types.MemoryQuery{
		Repository: &repo,
		Types:      []types.ChunkType{types.ChunkTypeProblem},
		Recency:    types.RecencyRecent,
	}

	filter := buildWhereFilter(query)
	if filter == nil {
		t.Fatal("expected non-nil filter")
	}

	// Test JSON marshaling
	jsonData, err := filter.MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal filter to JSON: %v", err)
	}

	t.Logf("Filter JSON: %s", string(jsonData))

	// Ensure JSON is valid
	if len(jsonData) == 0 {
		t.Error("expected non-empty JSON")
	}
}
