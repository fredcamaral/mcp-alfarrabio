package workflow

import (
	"context"
	"testing"
	"time"

	"mcp-memory/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockVectorStorage is a mock implementation of VectorStorage
type MockVectorStorage struct {
	mock.Mock
}

func (m *MockVectorStorage) Search(ctx context.Context, query string, filters map[string]interface{}, limit int) ([]types.ConversationChunk, error) {
	args := m.Called(ctx, query, filters, limit)
	return args.Get(0).([]types.ConversationChunk), args.Error(1)
}

func (m *MockVectorStorage) FindSimilar(ctx context.Context, content string, chunkType *types.ChunkType, limit int) ([]types.ConversationChunk, error) {
	args := m.Called(ctx, content, chunkType, limit)
	return args.Get(0).([]types.ConversationChunk), args.Error(1)
}

func TestNewContextSuggester(t *testing.T) {
	storage := &MockVectorStorage{}
	analyzer := NewPatternAnalyzer()
	tracker := NewTodoTracker()
	detector := NewFlowDetector()
	
	suggester := NewContextSuggester(storage, analyzer, tracker, detector)
	
	assert.NotNil(t, suggester)
	assert.NotNil(t, suggester.triggers)
	assert.NotNil(t, suggester.activeSuggestions)
	assert.Len(t, suggester.triggers, 5) // Should have 5 trigger types
}

func TestContextSuggester_ShouldTrigger(t *testing.T) {
	storage := &MockVectorStorage{}
	suggester := NewContextSuggester(storage, nil, nil, nil)
	
	testCases := []struct {
		name     string
		content  string
		toolUsed string
		flow     types.ConversationFlow
		trigger  SuggestionTrigger
		expected bool
	}{
		{
			name:     "keyword match triggers",
			content:  "I'm getting an error when building",
			toolUsed: "Read",
			flow:     types.FlowProblem,
			trigger: SuggestionTrigger{
				Keywords:     []string{"error", "issue"},
				ToolsUsed:    []string{"Read", "Grep"},
				FlowPatterns: []types.ConversationFlow{types.FlowProblem},
			},
			expected: true,
		},
		{
			name:     "no keyword match",
			content:  "Everything is working fine",
			toolUsed: "Read",
			flow:     types.FlowProblem,
			trigger: SuggestionTrigger{
				Keywords:     []string{"error", "issue"},
				ToolsUsed:    []string{"Read"},
				FlowPatterns: []types.ConversationFlow{types.FlowProblem},
			},
			expected: false,
		},
		{
			name:     "tool mismatch",
			content:  "I'm getting an error",
			toolUsed: "Write",
			flow:     types.FlowProblem,
			trigger: SuggestionTrigger{
				Keywords:     []string{"error"},
				ToolsUsed:    []string{"Read"},
				FlowPatterns: []types.ConversationFlow{types.FlowProblem},
			},
			expected: false,
		},
		{
			name:     "flow mismatch",
			content:  "I'm getting an error",
			toolUsed: "Read",
			flow:     types.FlowSolution,
			trigger: SuggestionTrigger{
				Keywords:     []string{"error"},
				ToolsUsed:    []string{"Read"},
				FlowPatterns: []types.ConversationFlow{types.FlowProblem},
			},
			expected: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := suggester.shouldTrigger(tc.content, tc.toolUsed, tc.flow, tc.trigger)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContextSuggester_CalculateRelevance(t *testing.T) {
	storage := &MockVectorStorage{}
	suggester := NewContextSuggester(storage, nil, nil, nil)
	
	testCases := []struct {
		name       string
		current    string
		historical string
		expected   float64
	}{
		{
			name:       "high relevance - many common words",
			current:    "authentication error in login module",
			historical: "authentication failed in login system",
			expected:   0.5, // Should be > 0.4
		},
		{
			name:       "low relevance - few common words",
			current:    "database connection error",
			historical: "user interface styling issue",
			expected:   0.0, // Should be < 0.2
		},
		{
			name:       "perfect match",
			current:    "build compilation error",
			historical: "build compilation error",
			expected:   1.0,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			relevance := suggester.calculateRelevance(tc.current, tc.historical)
			if tc.expected == 1.0 {
				assert.Equal(t, tc.expected, relevance)
			} else if tc.expected == 0.0 {
				assert.LessOrEqual(t, relevance, 0.2)
			} else {
				assert.GreaterOrEqual(t, relevance, 0.4)
			}
		})
	}
}

func TestContextSuggester_GenerateSimilarProblemSuggestions(t *testing.T) {
	storage := &MockVectorStorage{}
	suggester := NewContextSuggester(storage, nil, nil, nil)
	
	problemChunk := types.ConversationChunk{
		ID:        "problem-1",
		SessionID: "session-1",
		Content:   "Authentication error when logging in",
		Type:      types.ChunkTypeProblem,
		Timestamp: time.Now().Add(-24 * time.Hour),
		Metadata: types.ChunkMetadata{
			Outcome: types.OutcomeSuccess,
		},
	}
	
	solutionChunk := types.ConversationChunk{
		ID:        "solution-1",
		SessionID: "session-1",
		Content:   "Fixed by updating JWT validation logic",
		Type:      types.ChunkTypeSolution,
		Timestamp: time.Now().Add(-24 * time.Hour),
	}
	
	trigger := SuggestionTrigger{
		MinRelevance:   0.5,
		MaxSuggestions: 3,
	}
	
	// Mock the vector storage calls
	problemType := types.ChunkTypeProblem
	storage.On("FindSimilar", mock.Anything, "current authentication issue", &problemType, 6).Return(
		[]types.ConversationChunk{problemChunk}, nil)
	
	storage.On("Search", mock.Anything, problemChunk.Content, mock.MatchedBy(func(filters map[string]interface{}) bool {
		return filters["session_id"] == "session-1" && filters["type"] == types.ChunkTypeSolution
	}), 3).Return([]types.ConversationChunk{solutionChunk}, nil)
	
	suggestions, err := suggester.generateSimilarProblemSuggestions(
		context.Background(), trigger, "test-repo", "current authentication issue")
	
	require.NoError(t, err)
	assert.Len(t, suggestions, 1)
	
	suggestion := suggestions[0]
	assert.Equal(t, SuggestionTypeSimilarProblem, suggestion.Type)
	assert.Equal(t, ActionReview, suggestion.ActionType)
	assert.Equal(t, SourceVectorSearch, suggestion.Source)
	assert.Len(t, suggestion.RelatedChunks, 2) // Problem + Solution
	assert.GreaterOrEqual(t, suggestion.Relevance, trigger.MinRelevance)
	
	storage.AssertExpectations(t)
}

func TestContextSuggester_GenerateArchitecturalSuggestions(t *testing.T) {
	storage := &MockVectorStorage{}
	suggester := NewContextSuggester(storage, nil, nil, nil)
	
	decisionChunk := types.ConversationChunk{
		ID:      "decision-1",
		Content: "Decided to use microservices architecture for scalability",
		Type:    types.ChunkTypeArchitectureDecision,
		Summary: "Microservices architecture decision",
		Timestamp: time.Now().Add(-7 * 24 * time.Hour),
		Metadata: types.ChunkMetadata{
			Repository: "test-repo",
		},
	}
	
	trigger := SuggestionTrigger{
		MinRelevance:   0.6,
		MaxSuggestions: 2,
	}
	
	decisionType := types.ChunkTypeArchitectureDecision
	storage.On("FindSimilar", mock.Anything, "designing system architecture", &decisionType, 2).Return(
		[]types.ConversationChunk{decisionChunk}, nil)
	
	suggestions, err := suggester.generateArchitecturalSuggestions(
		context.Background(), trigger, "test-repo", "designing system architecture")
	
	require.NoError(t, err)
	
	if len(suggestions) > 0 {
		suggestion := suggestions[0]
		assert.Equal(t, SuggestionTypeArchitectural, suggestion.Type)
		assert.Equal(t, ActionConsider, suggestion.ActionType)
		assert.Equal(t, SourceDecisionLog, suggestion.Source)
		assert.Contains(t, suggestion.Description, "Architectural decision")
	}
	
	storage.AssertExpectations(t)
}

func TestContextSuggester_GenerateSuccessfulPatternSuggestions(t *testing.T) {
	storage := &MockVectorStorage{}
	analyzer := NewPatternAnalyzer()
	
	// Add a successful pattern to the analyzer
	analyzer.successPatterns = []SuccessPattern{
		{
			Type:        PatternTestDriven,
			Description: "Test-driven development approach",
			SuccessRate: 0.8,
			Frequency:   5,
		},
		{
			Type:        PatternInvestigative,
			Description: "Investigative debugging approach",
			SuccessRate: 0.6, // Below threshold
			Frequency:   3,
		},
	}
	
	suggester := NewContextSuggester(storage, analyzer, nil, nil)
	
	trigger := SuggestionTrigger{
		MaxSuggestions: 1,
	}
	
	suggestions, err := suggester.generateSuccessfulPatternSuggestions(
		context.Background(), trigger, "test-repo", "writing tests for new feature")
	
	require.NoError(t, err)
	assert.Len(t, suggestions, 1)
	
	suggestion := suggestions[0]
	assert.Equal(t, SuggestionTypeSuccessfulPattern, suggestion.Type)
	assert.Equal(t, ActionConsider, suggestion.ActionType)
	assert.Equal(t, SourcePatternAnalysis, suggestion.Source)
	assert.Contains(t, suggestion.Title, "Test-driven")
	assert.Contains(t, suggestion.Description, "80.0% success rate")
}

func TestContextSuggester_AnalyzeContext(t *testing.T) {
	storage := &MockVectorStorage{}
	analyzer := NewPatternAnalyzer()
	suggester := NewContextSuggester(storage, analyzer, nil, nil)
	
	// Mock a similar problem scenario
	problemChunk := types.ConversationChunk{
		ID:        "problem-1",
		SessionID: "session-1",
		Content:   "Build failure in CI pipeline",
		Type:      types.ChunkTypeProblem,
		Timestamp: time.Now().Add(-24 * time.Hour),
		Metadata: types.ChunkMetadata{
			Outcome: types.OutcomeSuccess,
		},
	}
	
	problemType := types.ChunkTypeProblem
	storage.On("FindSimilar", mock.Anything, "build error in pipeline", &problemType, 6).Return(
		[]types.ConversationChunk{problemChunk}, nil)
	
	storage.On("Search", mock.Anything, problemChunk.Content, mock.Anything, 3).Return(
		[]types.ConversationChunk{}, nil) // No solutions found
	
	suggestions, err := suggester.AnalyzeContext(
		context.Background(), 
		"current-session", 
		"test-repo", 
		"build error in pipeline", 
		"Read", 
		types.FlowProblem,
	)
	
	require.NoError(t, err)
	
	// Should have stored suggestions for the session
	activeSuggestions := suggester.GetActiveSuggestions("current-session")
	assert.Equal(t, suggestions, activeSuggestions)
	
	storage.AssertExpectations(t)
}

func TestContextSuggester_GetAndClearSuggestions(t *testing.T) {
	storage := &MockVectorStorage{}
	suggester := NewContextSuggester(storage, nil, nil, nil)
	
	// Create test suggestions
	testSuggestions := []ContextSuggestion{
		{
			ID:          "sug-1",
			Type:        SuggestionTypeSimilarProblem,
			Title:       "Test suggestion",
			Description: "Test description",
			Relevance:   0.8,
			CreatedAt:   time.Now(),
		},
	}
	
	// Store suggestions
	suggester.activeSuggestions["test-session"] = testSuggestions
	
	// Test GetActiveSuggestions
	retrieved := suggester.GetActiveSuggestions("test-session")
	assert.Equal(t, testSuggestions, retrieved)
	
	// Test non-existent session
	empty := suggester.GetActiveSuggestions("non-existent")
	assert.Len(t, empty, 0)
	
	// Test ClearSuggestions
	suggester.ClearSuggestions("test-session")
	cleared := suggester.GetActiveSuggestions("test-session")
	assert.Len(t, cleared, 0)
}

func TestContextSuggester_BuildDescriptions(t *testing.T) {
	storage := &MockVectorStorage{}
	suggester := NewContextSuggester(storage, nil, nil, nil)
	
	chunk := types.ConversationChunk{
		ID:        "test-1",
		Summary:   "This is a test summary that might be quite long and needs truncation",
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Metadata: types.ChunkMetadata{
			Outcome:    types.OutcomeSuccess,
			Repository: "test-repo",
		},
	}
	
	t.Run("similar problem description", func(t *testing.T) {
		desc := suggester.buildSimilarProblemDescription(chunk, []types.ConversationChunk{chunk})
		assert.Contains(t, desc, "Jan 15")
		assert.Contains(t, desc, "1 solution")
		assert.Contains(t, desc, "success")
	})
	
	t.Run("architectural description", func(t *testing.T) {
		desc := suggester.buildArchitecturalDescription(chunk)
		assert.Contains(t, desc, "Architectural decision")
		assert.Contains(t, desc, "Jan 15")
	})
	
	t.Run("successful pattern description", func(t *testing.T) {
		pattern := SuccessPattern{
			Description: "Test pattern",
			SuccessRate: 0.85,
			Frequency:   10,
		}
		desc := suggester.buildSuccessfulPatternDescription(pattern)
		assert.Contains(t, desc, "Test pattern")
		assert.Contains(t, desc, "85.0%")
		assert.Contains(t, desc, "10 times")
	})
}

func TestContextSuggester_TruncateString(t *testing.T) {
	testCases := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			input:    "this is a very long string that needs truncation",
			maxLen:   20,
			expected: "this is a very lo...",
		},
		{
			input:    "exactly twenty chars",
			maxLen:   20,
			expected: "exactly twenty chars",
		},
	}
	
	for _, tc := range testCases {
		result := truncateString(tc.input, tc.maxLen)
		assert.Equal(t, tc.expected, result)
		assert.LessOrEqual(t, len(result), tc.maxLen)
	}
}

func TestContextSuggester_InitializeTriggers(t *testing.T) {
	storage := &MockVectorStorage{}
	suggester := NewContextSuggester(storage, nil, nil, nil)
	
	// Verify all expected triggers are initialized
	expectedTypes := []SuggestionType{
		SuggestionTypeSimilarProblem,
		SuggestionTypeArchitectural,
		SuggestionTypePastDecision,
		SuggestionTypeDuplicateWork,
		SuggestionTypeSuccessfulPattern,
	}
	
	for _, expectedType := range expectedTypes {
		trigger, exists := suggester.triggers[expectedType]
		assert.True(t, exists, "Trigger type %s should exist", expectedType)
		assert.Greater(t, trigger.MinRelevance, 0.0, "MinRelevance should be positive")
		assert.Greater(t, trigger.MaxSuggestions, 0, "MaxSuggestions should be positive")
	}
	
	// Verify similar problem trigger has expected keywords
	similarProblemTrigger := suggester.triggers[SuggestionTypeSimilarProblem]
	assert.Contains(t, similarProblemTrigger.Keywords, "error")
	assert.Contains(t, similarProblemTrigger.Keywords, "problem")
	assert.Contains(t, similarProblemTrigger.FlowPatterns, types.FlowProblem)
}