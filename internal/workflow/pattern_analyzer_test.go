package workflow

import (
	"testing"
	"time"

	"mcp-memory/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPatternAnalyzer(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.sequences)
	assert.NotNil(t, analyzer.successPatterns)
	assert.NotNil(t, analyzer.contextSwitches)
	assert.Len(t, analyzer.sequences, 0)
	assert.Len(t, analyzer.successPatterns, 0)
	assert.Len(t, analyzer.contextSwitches, 0)
}

func TestPatternAnalyzer_StartEndSequence(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	t.Run("starts new sequence", func(t *testing.T) {
		analyzer.StartSequence("session1", "test-repo", "bug-fix")
		
		assert.NotNil(t, analyzer.currentSequence)
		assert.Equal(t, "session1", analyzer.currentSequence.SessionID)
		assert.Equal(t, "test-repo", analyzer.currentSequence.Repository)
		assert.Equal(t, "bug-fix", analyzer.currentSequence.ProblemType)
		assert.Len(t, analyzer.currentSequence.Tools, 0)
	})
	
	t.Run("ends sequence with success", func(t *testing.T) {
		analyzer.StartSequence("session2", "test-repo", "feature")
		
		// Add some tool usage
		analyzer.RecordToolUsage("Read", map[string]interface{}{"file": "main.go"}, true)
		analyzer.RecordToolUsage("Edit", map[string]interface{}{"file": "main.go"}, true)
		
		analyzer.EndSequence(types.OutcomeSuccess, "Added new feature")
		
		assert.Nil(t, analyzer.currentSequence)
		assert.Len(t, analyzer.sequences, 1)
		
		sequence := analyzer.sequences[0]
		assert.Equal(t, types.OutcomeSuccess, sequence.Outcome)
		assert.Equal(t, "Added new feature", sequence.Solution)
		assert.Len(t, sequence.Tools, 2)
	})
}

func TestPatternAnalyzer_RecordToolUsage(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	t.Run("records tool usage in existing sequence", func(t *testing.T) {
		analyzer.StartSequence("session1", "test-repo", "debug")
		
		context := map[string]interface{}{
			"file_path": "/path/to/file.go",
			"command":   "grep error",
		}
		
		analyzer.RecordToolUsage("Grep", context, true)
		
		assert.Len(t, analyzer.currentSequence.Tools, 1)
		
		usage := analyzer.currentSequence.Tools[0]
		assert.Equal(t, "Grep", usage.Tool)
		assert.True(t, usage.Success)
		assert.Equal(t, context, usage.Context)
	})
	
	t.Run("auto-starts sequence if none exists", func(t *testing.T) {
		analyzer2 := NewPatternAnalyzer()
		
		analyzer2.RecordToolUsage("Read", map[string]interface{}{}, true)
		
		assert.NotNil(t, analyzer2.currentSequence)
		assert.Len(t, analyzer2.currentSequence.Tools, 1)
	})
}

func TestPatternAnalyzer_DetectPatternType(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	testCases := []struct {
		name        string
		tools       []string
		expectedType PatternType
	}{
		{
			name:         "investigative pattern",
			tools:        []string{"Read", "Grep", "Read", "Edit"},
			expectedType: PatternInvestigative,
		},
		{
			name:         "build-fix pattern",
			tools:        []string{"Bash", "Read", "Edit", "Bash"},
			expectedType: PatternBuildFix,
		},
		{
			name:         "test-driven pattern",
			tools:        []string{"Test", "Edit", "Test"},
			expectedType: PatternTestDriven,
		},
		{
			name:         "exploration pattern",
			tools:        []string{"Glob", "Read", "Grep", "Read"},
			expectedType: PatternExploration,
		},
		{
			name:         "configuration pattern",
			tools:        []string{"Read", "Edit", "Bash"},
			expectedType: PatternConfiguration,
		},
		{
			name:         "debug pattern",
			tools:        []string{"Bash", "Grep", "Read", "Edit"},
			expectedType: PatternDebug,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patternType := analyzer.DetectPatternType(tc.tools)
			assert.Equal(t, tc.expectedType, patternType)
		})
	}
}

func TestPatternAnalyzer_CalculatePatternMatch(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	t.Run("exact match", func(t *testing.T) {
		tools := []string{"Read", "Edit", "Test"}
		pattern := []string{"Read", "Edit", "Test"}
		
		score := analyzer.calculatePatternMatch(tools, pattern)
		assert.Equal(t, 3, score)
	})
	
	t.Run("partial match", func(t *testing.T) {
		tools := []string{"Read", "Grep", "Edit", "Test"}
		pattern := []string{"Read", "Edit"}
		
		score := analyzer.calculatePatternMatch(tools, pattern)
		assert.Equal(t, 1, score) // Only "Read" matches consecutively at start
	})
	
	t.Run("no match", func(t *testing.T) {
		tools := []string{"Read", "Edit"}
		pattern := []string{"Bash", "Test", "Deploy"}
		
		score := analyzer.calculatePatternMatch(tools, pattern)
		assert.Equal(t, 0, score)
	})
	
	t.Run("subsequence match", func(t *testing.T) {
		tools := []string{"LS", "Read", "Edit", "Test", "Bash"}
		pattern := []string{"Read", "Edit", "Test"}
		
		score := analyzer.calculatePatternMatch(tools, pattern)
		assert.Equal(t, 3, score)
	})
}

func TestPatternAnalyzer_AnalyzeSuccessfulSequence(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	// Create a successful investigative sequence
	analyzer.StartSequence("session1", "test-repo", "bug-fix")
	analyzer.RecordToolUsage("Read", map[string]interface{}{}, true)
	analyzer.RecordToolUsage("Grep", map[string]interface{}{}, true)
	analyzer.RecordToolUsage("Read", map[string]interface{}{}, true)
	analyzer.RecordToolUsage("Edit", map[string]interface{}{}, true)
	analyzer.EndSequence(types.OutcomeSuccess, "Fixed authentication bug")
	
	patterns := analyzer.GetSuccessPatterns()
	require.Len(t, patterns, 1)
	
	pattern := patterns[0]
	assert.Equal(t, PatternInvestigative, pattern.Type)
	assert.Equal(t, 1, pattern.Frequency)
	assert.Equal(t, 1.0, pattern.SuccessRate)
	assert.Contains(t, pattern.Examples, "Fixed authentication bug")
	assert.Contains(t, pattern.Description, "Investigative approach")
}

func TestPatternAnalyzer_UpdateSuccessPattern(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	// First successful sequence
	analyzer.StartSequence("session1", "repo1", "feature")
	analyzer.RecordToolUsage("Test", map[string]interface{}{}, true)
	analyzer.RecordToolUsage("Edit", map[string]interface{}{}, true)
	analyzer.RecordToolUsage("Test", map[string]interface{}{}, true)
	analyzer.EndSequence(types.OutcomeSuccess, "Added tests")
	
	// Second successful sequence of same type
	analyzer.StartSequence("session2", "repo1", "feature")
	analyzer.RecordToolUsage("Test", map[string]interface{}{}, true)
	analyzer.RecordToolUsage("Edit", map[string]interface{}{}, true)
	analyzer.RecordToolUsage("Test", map[string]interface{}{}, true)
	analyzer.EndSequence(types.OutcomeSuccess, "Updated tests")
	
	patterns := analyzer.GetSuccessPatterns()
	require.Len(t, patterns, 1)
	
	pattern := patterns[0]
	assert.Equal(t, 2, pattern.Frequency)
	assert.Equal(t, 1.0, pattern.SuccessRate)
	assert.Len(t, pattern.Examples, 2)
}

func TestPatternAnalyzer_InferSwitchReason(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	testCases := []struct {
		name     string
		usage    ToolUsage
		expected string
	}{
		{
			name: "exploration with LS",
			usage: ToolUsage{
				Tool: "LS",
				Context: map[string]interface{}{},
			},
			expected: "exploring new codebase",
		},
		{
			name: "reading README",
			usage: ToolUsage{
				Tool: "Read",
				Context: map[string]interface{}{
					"file_path": "/project/README.md",
				},
			},
			expected: "reading project documentation",
		},
		{
			name: "changing directories",
			usage: ToolUsage{
				Tool: "Bash",
				Context: map[string]interface{}{
					"command": "cd /new/project",
				},
			},
			expected: "changing directories",
		},
		{
			name: "git operations",
			usage: ToolUsage{
				Tool: "Bash",
				Context: map[string]interface{}{
					"command": "git status",
				},
			},
			expected: "git operations",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reason := analyzer.inferSwitchReason(tc.usage)
			assert.Equal(t, tc.expected, reason)
		})
	}
}

func TestPatternAnalyzer_ExtractSequenceTags(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	sequence := ToolSequence{
		SessionID:   "test",
		Repository:  "test-repo",
		StartTime:   time.Now().Add(-45 * time.Minute),
		EndTime:     time.Now(),
		Outcome:     types.OutcomeSuccess,
		ProblemType: "bug-fix",
		Tools: []ToolUsage{
			{Tool: "Read"}, {Tool: "Read"}, {Tool: "Read"}, {Tool: "Read"}, // 4x Read
			{Tool: "Edit"}, {Tool: "Bash"},
		},
	}
	
	tags := analyzer.extractSequenceTags(sequence)
	
	assert.Contains(t, tags, "success")
	assert.Contains(t, tags, "repo-test-repo")
	assert.Contains(t, tags, "long-session")  // 45 minutes
	assert.Contains(t, tags, "heavy-read")    // 4+ reads
}

func TestPatternAnalyzer_GetPatternRecommendations(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	// Create a high-success pattern
	analyzer.successPatterns = []SuccessPattern{
		{
			Type:        PatternTestDriven,
			Tools:       []string{"Test", "Edit", "Test"},
			SuccessRate: 0.8,
			Frequency:   5,
		},
		{
			Type:        PatternInvestigative,
			Tools:       []string{"Read", "Grep", "Edit"},
			SuccessRate: 0.6, // Below threshold
			Frequency:   3,
		},
		{
			Type:        PatternBuildFix,
			Tools:       []string{"Bash", "Edit", "Bash"},
			SuccessRate: 0.9,
			Frequency:   1, // Below frequency threshold
		},
	}
	
	currentTools := []string{"Test", "Edit"} // Partially matches test-driven
	
	recommendations := analyzer.GetPatternRecommendations(currentTools, "feature")
	
	assert.Len(t, recommendations, 1)
	assert.Equal(t, PatternTestDriven, recommendations[0].Type)
}

func TestPatternAnalyzer_GeneratePatternDescription(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	testCases := []struct {
		patternType PatternType
		tools       []string
		expected    string
	}{
		{
			patternType: PatternInvestigative,
			tools:       []string{"Read", "Grep"},
			expected:    "Investigative approach: Read → Grep",
		},
		{
			patternType: PatternBuildFix,
			tools:       []string{"Build", "Fix"},
			expected:    "Build-fix cycle: Build → Fix",
		},
		{
			patternType: PatternTestDriven,
			tools:       []string{"Test", "Code"},
			expected:    "Test-driven development: Test → Code",
		},
	}
	
	for _, tc := range testCases {
		t.Run(string(tc.patternType), func(t *testing.T) {
			description := analyzer.generatePatternDescription(tc.patternType, tc.tools)
			assert.Equal(t, tc.expected, description)
		})
	}
}

func TestPatternAnalyzer_ContextSwitchDetection(t *testing.T) {
	analyzer := NewPatternAnalyzer()
	
	// Create first sequence in repo1
	analyzer.StartSequence("session1", "repo1", "feature")
	analyzer.RecordToolUsage("Read", map[string]interface{}{}, true)
	analyzer.EndSequence(types.OutcomeSuccess, "Done")
	
	// Start new sequence in repo2 (should trigger context switch)
	analyzer.StartSequence("session1", "repo2", "bugfix")
	analyzer.RecordToolUsage("LS", map[string]interface{}{}, true)
	
	switches := analyzer.GetContextSwitches()
	require.Len(t, switches, 1)
	
	contextSwitch := switches[0]
	assert.Equal(t, "repo1", contextSwitch.FromRepo)
	assert.Equal(t, "repo2", contextSwitch.ToRepo)
	assert.Equal(t, "LS", contextSwitch.TriggerTool)
	assert.Equal(t, "exploring new codebase", contextSwitch.Reason)
}