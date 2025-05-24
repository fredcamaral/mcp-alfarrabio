package types

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkType_Valid(t *testing.T) {
	tests := []struct {
		name     string
		ct       ChunkType
		expected bool
	}{
		{"valid problem", ChunkTypeProblem, true},
		{"valid solution", ChunkTypeSolution, true},
		{"valid code change", ChunkTypeCodeChange, true},
		{"valid discussion", ChunkTypeDiscussion, true},
		{"valid architecture decision", ChunkTypeArchitectureDecision, true},
		{"invalid empty", ChunkType(""), false},
		{"invalid random", ChunkType("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.ct.Valid())
		})
	}
}

func TestOutcome_Valid(t *testing.T) {
	tests := []struct {
		name     string
		outcome  Outcome
		expected bool
	}{
		{"valid success", OutcomeSuccess, true},
		{"valid in progress", OutcomeInProgress, true},
		{"valid failed", OutcomeFailed, true},
		{"valid abandoned", OutcomeAbandoned, true},
		{"invalid empty", Outcome(""), false},
		{"invalid random", Outcome("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.outcome.Valid())
		})
	}
}

func TestDifficulty_Valid(t *testing.T) {
	tests := []struct {
		name       string
		difficulty Difficulty
		expected   bool
	}{
		{"valid simple", DifficultySimple, true},
		{"valid moderate", DifficultyModerate, true},
		{"valid complex", DifficultyComplex, true},
		{"invalid empty", Difficulty(""), false},
		{"invalid random", Difficulty("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.difficulty.Valid())
		})
	}
}

func TestConversationFlow_Valid(t *testing.T) {
	tests := []struct {
		name     string
		flow     ConversationFlow
		expected bool
	}{
		{"valid problem", FlowProblem, true},
		{"valid investigation", FlowInvestigation, true},
		{"valid solution", FlowSolution, true},
		{"valid verification", FlowVerification, true},
		{"invalid empty", ConversationFlow(""), false},
		{"invalid random", ConversationFlow("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.flow.Valid())
		})
	}
}

func TestRecency_Valid(t *testing.T) {
	tests := []struct {
		name     string
		recency  Recency
		expected bool
	}{
		{"valid recent", RecencyRecent, true},
		{"valid all time", RecencyAllTime, true},
		{"valid last month", RecencyLastMonth, true},
		{"invalid empty", Recency(""), false},
		{"invalid random", Recency("random"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.recency.Valid())
		})
	}
}

func TestChunkMetadata_Validate(t *testing.T) {
	tests := []struct {
		name     string
		metadata ChunkMetadata
		wantErr  bool
	}{
		{
			name: "valid metadata",
			metadata: ChunkMetadata{
				Repository:    "test-repo",
				Branch:        "main",
				FilesModified: []string{"file1.go", "file2.go"},
				ToolsUsed:     []string{"edit", "read"},
				Outcome:       OutcomeSuccess,
				Tags:          []string{"go", "test"},
				Difficulty:    DifficultySimple,
				TimeSpent:     nil,
			},
			wantErr: false,
		},
		{
			name: "valid with time spent",
			metadata: ChunkMetadata{
				Outcome:    OutcomeSuccess,
				Difficulty: DifficultyModerate,
				TimeSpent:  func() *int { i := 15; return &i }(),
			},
			wantErr: false,
		},
		{
			name: "invalid outcome",
			metadata: ChunkMetadata{
				Outcome:    Outcome("invalid"),
				Difficulty: DifficultySimple,
			},
			wantErr: true,
		},
		{
			name: "invalid difficulty",
			metadata: ChunkMetadata{
				Outcome:    OutcomeSuccess,
				Difficulty: Difficulty("invalid"),
			},
			wantErr: true,
		},
		{
			name: "negative time spent",
			metadata: ChunkMetadata{
				Outcome:    OutcomeSuccess,
				Difficulty: DifficultySimple,
				TimeSpent:  func() *int { i := -5; return &i }(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewConversationChunk(t *testing.T) {
	sessionID := "test-session"
	content := "test content"
	chunkType := ChunkTypeProblem
	metadata := ChunkMetadata{
		Repository:    "test-repo",
		Outcome:       OutcomeInProgress,
		Difficulty:    DifficultySimple,
		FilesModified: []string{},
		ToolsUsed:     []string{},
		Tags:          []string{},
	}

	t.Run("valid chunk creation", func(t *testing.T) {
		chunk, err := NewConversationChunk(sessionID, content, chunkType, metadata)
		require.NoError(t, err)

		assert.NotEmpty(t, chunk.ID)
		assert.Equal(t, sessionID, chunk.SessionID)
		assert.Equal(t, content, chunk.Content)
		assert.Equal(t, chunkType, chunk.Type)
		assert.Equal(t, metadata, chunk.Metadata)
		assert.Empty(t, chunk.Summary)
		assert.Empty(t, chunk.Embeddings)
		assert.Empty(t, chunk.RelatedChunks)
		assert.False(t, chunk.Timestamp.IsZero())
	})

	t.Run("empty session ID", func(t *testing.T) {
		_, err := NewConversationChunk("", content, chunkType, metadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "session ID cannot be empty")
	})

	t.Run("empty content", func(t *testing.T) {
		_, err := NewConversationChunk(sessionID, "", chunkType, metadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content cannot be empty")
	})

	t.Run("invalid chunk type", func(t *testing.T) {
		_, err := NewConversationChunk(sessionID, content, ChunkType("invalid"), metadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid chunk type")
	})

	t.Run("invalid metadata", func(t *testing.T) {
		invalidMetadata := ChunkMetadata{
			Outcome:    Outcome("invalid"),
			Difficulty: DifficultySimple,
		}
		_, err := NewConversationChunk(sessionID, content, chunkType, invalidMetadata)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid metadata")
	})
}

func TestConversationChunk_Validate(t *testing.T) {
	validChunk := &ConversationChunk{
		ID:        uuid.New().String(),
		SessionID: "test-session",
		Timestamp: time.Now(),
		Type:      ChunkTypeProblem,
		Content:   "test content",
		Summary:   "test summary",
		Metadata: ChunkMetadata{
			Outcome:       OutcomeInProgress,
			Difficulty:    DifficultySimple,
			FilesModified: []string{},
			ToolsUsed:     []string{},
			Tags:          []string{},
		},
		Embeddings:    []float64{0.1, 0.2, 0.3},
		RelatedChunks: []string{},
	}

	t.Run("valid chunk", func(t *testing.T) {
		assert.NoError(t, validChunk.Validate())
	})

	t.Run("empty ID", func(t *testing.T) {
		chunk := *validChunk
		chunk.ID = ""
		assert.Error(t, chunk.Validate())
	})

	t.Run("empty session ID", func(t *testing.T) {
		chunk := *validChunk
		chunk.SessionID = ""
		assert.Error(t, chunk.Validate())
	})

	t.Run("empty content", func(t *testing.T) {
		chunk := *validChunk
		chunk.Content = ""
		assert.Error(t, chunk.Validate())
	})

	t.Run("invalid type", func(t *testing.T) {
		chunk := *validChunk
		chunk.Type = ChunkType("invalid")
		assert.Error(t, chunk.Validate())
	})

	t.Run("zero timestamp", func(t *testing.T) {
		chunk := *validChunk
		chunk.Timestamp = time.Time{}
		assert.Error(t, chunk.Validate())
	})
}

func TestNewProjectContext(t *testing.T) {
	repository := "test-repo"

	context := NewProjectContext(repository)

	assert.Equal(t, repository, context.Repository)
	assert.False(t, context.LastAccessed.IsZero())
	assert.Equal(t, 0, context.TotalSessions)
	assert.Empty(t, context.CommonPatterns)
	assert.Empty(t, context.ArchitecturalDecisions)
	assert.Empty(t, context.TechStack)
	assert.Empty(t, context.TeamPreferences)
}

func TestProjectContext_Validate(t *testing.T) {
	t.Run("valid context", func(t *testing.T) {
		context := &ProjectContext{
			Repository:             "test-repo",
			LastAccessed:           time.Now(),
			TotalSessions:          5,
			CommonPatterns:         []string{"pattern1"},
			ArchitecturalDecisions: []string{"decision1"},
			TechStack:              []string{"go"},
			TeamPreferences:        []string{"preference1"},
		}
		assert.NoError(t, context.Validate())
	})

	t.Run("empty repository", func(t *testing.T) {
		context := &ProjectContext{
			Repository:    "",
			TotalSessions: 5,
		}
		assert.Error(t, context.Validate())
	})

	t.Run("negative sessions", func(t *testing.T) {
		context := &ProjectContext{
			Repository:    "test-repo",
			TotalSessions: -1,
		}
		assert.Error(t, context.Validate())
	})
}

func TestNewMemoryQuery(t *testing.T) {
	query := "test query"

	memQuery := NewMemoryQuery(query)

	assert.Equal(t, query, memQuery.Query)
	assert.Equal(t, RecencyRecent, memQuery.Recency)
	assert.Equal(t, 0.7, memQuery.MinRelevanceScore)
	assert.Equal(t, 10, memQuery.Limit)
	assert.Nil(t, memQuery.Repository)
	assert.Empty(t, memQuery.FileContext)
	assert.Empty(t, memQuery.Types)
}

func TestMemoryQuery_Validate(t *testing.T) {
	t.Run("valid query", func(t *testing.T) {
		query := &MemoryQuery{
			Query:             "test query",
			Recency:           RecencyRecent,
			MinRelevanceScore: 0.7,
			Limit:             10,
			Types:             []ChunkType{ChunkTypeProblem},
		}
		assert.NoError(t, query.Validate())
	})

	t.Run("empty query", func(t *testing.T) {
		query := &MemoryQuery{
			Query:   "",
			Recency: RecencyRecent,
		}
		assert.Error(t, query.Validate())
	})

	t.Run("invalid recency", func(t *testing.T) {
		query := &MemoryQuery{
			Query:   "test",
			Recency: Recency("invalid"),
		}
		assert.Error(t, query.Validate())
	})

	t.Run("invalid relevance score - negative", func(t *testing.T) {
		query := &MemoryQuery{
			Query:             "test",
			Recency:           RecencyRecent,
			MinRelevanceScore: -0.1,
		}
		assert.Error(t, query.Validate())
	})

	t.Run("invalid relevance score - too high", func(t *testing.T) {
		query := &MemoryQuery{
			Query:             "test",
			Recency:           RecencyRecent,
			MinRelevanceScore: 1.1,
		}
		assert.Error(t, query.Validate())
	})

	t.Run("negative limit", func(t *testing.T) {
		query := &MemoryQuery{
			Query:   "test",
			Recency: RecencyRecent,
			Limit:   -1,
		}
		assert.Error(t, query.Validate())
	})

	t.Run("invalid chunk type", func(t *testing.T) {
		query := &MemoryQuery{
			Query:   "test",
			Recency: RecencyRecent,
			Types:   []ChunkType{ChunkType("invalid")},
		}
		assert.Error(t, query.Validate())
	})
}

func TestChunkingContext_Validate(t *testing.T) {
	t.Run("valid context", func(t *testing.T) {
		context := &ChunkingContext{
			CurrentTodos: []TodoItem{
				{ID: "1", Status: "pending", Content: "test todo"},
			},
			FileModifications: []string{"file1.go"},
			ToolsUsed:         []string{"edit"},
			TimeElapsed:       5,
			ConversationFlow:  FlowProblem,
		}
		assert.NoError(t, context.Validate())
	})

	t.Run("negative time elapsed", func(t *testing.T) {
		context := &ChunkingContext{
			TimeElapsed:      -1,
			ConversationFlow: FlowProblem,
		}
		assert.Error(t, context.Validate())
	})

	t.Run("invalid conversation flow", func(t *testing.T) {
		context := &ChunkingContext{
			TimeElapsed:      5,
			ConversationFlow: ConversationFlow("invalid"),
		}
		assert.Error(t, context.Validate())
	})
}

func TestChunkingContext_HasCompletedTodos(t *testing.T) {
	t.Run("has completed todos", func(t *testing.T) {
		context := &ChunkingContext{
			CurrentTodos: []TodoItem{
				{ID: "1", Status: "pending", Content: "todo 1"},
				{ID: "2", Status: "completed", Content: "todo 2"},
				{ID: "3", Status: "in_progress", Content: "todo 3"},
			},
		}
		assert.True(t, context.HasCompletedTodos())
	})

	t.Run("no completed todos", func(t *testing.T) {
		context := &ChunkingContext{
			CurrentTodos: []TodoItem{
				{ID: "1", Status: "pending", Content: "todo 1"},
				{ID: "2", Status: "in_progress", Content: "todo 2"},
			},
		}
		assert.False(t, context.HasCompletedTodos())
	})

	t.Run("empty todos", func(t *testing.T) {
		context := &ChunkingContext{
			CurrentTodos: []TodoItem{},
		}
		assert.False(t, context.HasCompletedTodos())
	})
}

func TestJSONMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{"ChunkType", ChunkTypeProblem},
		{"Outcome", OutcomeSuccess},
		{"Difficulty", DifficultyModerate},
		{"ConversationFlow", FlowSolution},
		{"Recency", RecencyRecent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.data)
			require.NoError(t, err)

			// Unmarshal back
			switch v := tt.data.(type) {
			case ChunkType:
				var result ChunkType
				err = json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, v, result)
			case Outcome:
				var result Outcome
				err = json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, v, result)
			case Difficulty:
				var result Difficulty
				err = json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, v, result)
			case ConversationFlow:
				var result ConversationFlow
				err = json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, v, result)
			case Recency:
				var result Recency
				err = json.Unmarshal(data, &result)
				require.NoError(t, err)
				assert.Equal(t, v, result)
			}
		})
	}
}

func TestConversationChunk_JSONRoundtrip(t *testing.T) {
	original := &ConversationChunk{
		ID:        uuid.New().String(),
		SessionID: "test-session",
		Timestamp: time.Now().UTC().Truncate(time.Second), // Truncate to avoid precision issues
		Type:      ChunkTypeProblem,
		Content:   "test content",
		Summary:   "test summary",
		Metadata: ChunkMetadata{
			Repository:    "test-repo",
			Branch:        "main",
			FilesModified: []string{"file1.go", "file2.go"},
			ToolsUsed:     []string{"edit", "read"},
			Outcome:       OutcomeSuccess,
			Tags:          []string{"go", "test"},
			Difficulty:    DifficultyModerate,
			TimeSpent:     func() *int { i := 15; return &i }(),
		},
		Embeddings:    []float64{0.1, 0.2, 0.3},
		RelatedChunks: []string{"chunk1", "chunk2"},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Unmarshal back
	var result ConversationChunk
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.ID, result.ID)
	assert.Equal(t, original.SessionID, result.SessionID)
	assert.Equal(t, original.Timestamp, result.Timestamp)
	assert.Equal(t, original.Type, result.Type)
	assert.Equal(t, original.Content, result.Content)
	assert.Equal(t, original.Summary, result.Summary)
	assert.Equal(t, original.Metadata, result.Metadata)
	assert.Equal(t, original.Embeddings, result.Embeddings)
	assert.Equal(t, original.RelatedChunks, result.RelatedChunks)
}

func TestSearchResults(t *testing.T) {
	chunk := ConversationChunk{
		ID:        "test-id",
		SessionID: "test-session",
		Timestamp: time.Now(),
		Type:      ChunkTypeProblem,
		Content:   "test content",
	}

	results := &SearchResults{
		Results: []SearchResult{
			{Chunk: chunk, Score: 0.95},
			{Chunk: chunk, Score: 0.85},
		},
		Total:     2,
		QueryTime: time.Millisecond * 100,
	}

	assert.Len(t, results.Results, 2)
	assert.Equal(t, 2, results.Total)
	assert.Equal(t, 0.95, results.Results[0].Score)
	assert.Equal(t, 0.85, results.Results[1].Score)
	assert.Equal(t, time.Millisecond*100, results.QueryTime)
}
