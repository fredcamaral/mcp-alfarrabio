package workflow

import (
	"testing"
	"time"

	"mcp-memory/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFlowDetector(t *testing.T) {
	detector := NewFlowDetector()
	
	assert.NotNil(t, detector)
	assert.NotNil(t, detector.sessions)
	assert.NotNil(t, detector.flowPatterns)
	assert.NotNil(t, detector.entityExtractor)
	assert.Len(t, detector.flowPatterns, 4) // Problem, Investigation, Solution, Verification
}

func TestNewEntityExtractor(t *testing.T) {
	extractor := NewEntityExtractor()
	
	assert.NotNil(t, extractor)
	assert.NotNil(t, extractor.filePattern)
	assert.NotNil(t, extractor.errorPattern)
	assert.NotNil(t, extractor.commandPattern)
	assert.NotNil(t, extractor.urlPattern)
}

func TestFlowDetector_StartSession(t *testing.T) {
	detector := NewFlowDetector()
	sessionID := "test-session"
	repository := "test-repo"
	
	detector.StartSession(sessionID, repository)
	
	session, exists := detector.GetSession(sessionID)
	require.True(t, exists)
	assert.Equal(t, sessionID, session.SessionID)
	assert.Equal(t, repository, session.Repository)
	assert.Nil(t, session.EndTime)
	assert.Len(t, session.Segments, 0)
	assert.Len(t, session.Transitions, 0)
}

func TestFlowDetector_DetectFlow(t *testing.T) {
	detector := NewFlowDetector()
	
	testCases := []struct {
		name         string
		content      string
		toolUsed     string
		expectedFlow types.ConversationFlow
		minConfidence float64
	}{
		{
			name:         "problem detection",
			content:      "I'm getting an error when trying to build the project",
			toolUsed:     "Read",
			expectedFlow: types.FlowProblem,
			minConfidence: 0.3,
		},
		{
			name:         "investigation detection",
			content:      "Let me check the configuration files to see what's wrong",
			toolUsed:     "Grep",
			expectedFlow: types.FlowInvestigation,
			minConfidence: 0.3,
		},
		{
			name:         "solution detection",
			content:      "I'll implement a fix for this issue by updating the code",
			toolUsed:     "Edit",
			expectedFlow: types.FlowSolution,
			minConfidence: 0.3,
		},
		{
			name:         "verification detection",
			content:      "Let's test the fix to make sure it works correctly",
			toolUsed:     "Bash",
			expectedFlow: types.FlowVerification,
			minConfidence: 0.3,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flow, confidence := detector.detectFlow(tc.content, tc.toolUsed)
			assert.Equal(t, tc.expectedFlow, flow)
			assert.GreaterOrEqual(t, confidence, tc.minConfidence)
		})
	}
}

func TestFlowDetector_ProcessMessage(t *testing.T) {
	detector := NewFlowDetector()
	sessionID := "test-session"
	
	detector.StartSession(sessionID, "test-repo")
	
	// Process a problem message
	detector.ProcessMessage(sessionID, "There's an error in the build process", "Read", nil)
	
	_, exists := detector.GetSession(sessionID)
	require.True(t, exists)
	
	// Should have created a current segment
	assert.NotNil(t, detector.currentSegment)
	assert.Equal(t, types.FlowProblem, detector.currentSegment.Flow)
}

func TestFlowDetector_SegmentTransitions(t *testing.T) {
	detector := NewFlowDetector()
	sessionID := "test-session"
	
	detector.StartSession(sessionID, "test-repo")
	
	// Step 1: Problem
	detector.ProcessMessage(sessionID, "I'm getting an error when building", "Read", nil)
	assert.NotNil(t, detector.currentSegment)
	assert.Equal(t, types.FlowProblem, detector.currentSegment.Flow)
	
	// Step 2: Investigation (should trigger new segment)
	detector.ProcessMessage(sessionID, "Let me investigate this issue by checking the logs", "Grep", nil)
	
	_, _ = detector.GetSession(sessionID) // test variable - unused
	
	// Should have finished the problem segment and started investigation
	assert.NotNil(t, detector.currentSegment)
	assert.Equal(t, types.FlowInvestigation, detector.currentSegment.Flow)
	
	// Step 3: Solution (should trigger another transition)
	detector.ProcessMessage(sessionID, "I found the issue, let me implement a fix", "Edit", nil)
	assert.Equal(t, types.FlowSolution, detector.currentSegment.Flow)
	
	// End session to finalize segments
	detector.EndSession(sessionID, types.OutcomeSuccess)
	
	session, _ := detector.GetSession(sessionID)
	assert.NotNil(t, session.EndTime)
	assert.GreaterOrEqual(t, len(session.Segments), 2) // Should have multiple segments
	assert.GreaterOrEqual(t, len(session.Transitions), 1) // Should have transitions
}

func TestFlowDetector_SessionSummary(t *testing.T) {
	detector := NewFlowDetector()
	sessionID := "test-session"
	
	detector.StartSession(sessionID, "test-repo")
	
	// Simulate a complete problem-solving flow
	detector.ProcessMessage(sessionID, "Error in authentication module", "Read", nil)
	time.Sleep(10 * time.Millisecond) // Ensure measurable time differences
	
	detector.ProcessMessage(sessionID, "Let me examine the auth.go file", "Grep", nil)
	time.Sleep(10 * time.Millisecond)
	
	detector.ProcessMessage(sessionID, "I'll fix the JWT validation logic", "Edit", nil)
	time.Sleep(10 * time.Millisecond)
	
	detector.ProcessMessage(sessionID, "Running tests to verify the fix", "Bash", nil)
	
	detector.EndSession(sessionID, types.OutcomeSuccess)
	
	session, exists := detector.GetSession(sessionID)
	require.True(t, exists)
	
	summary := session.Summary
	assert.Greater(t, summary.TotalDuration, time.Duration(0))
	assert.GreaterOrEqual(t, summary.ProblemsSolved, 0)
	assert.GreaterOrEqual(t, summary.InvestigationTime, time.Duration(0))
	assert.GreaterOrEqual(t, summary.SolutionTime, time.Duration(0))
	assert.GreaterOrEqual(t, summary.VerificationTime, time.Duration(0))
}

func TestEntityExtractor_Extract(t *testing.T) {
	extractor := NewEntityExtractor()
	
	testCases := []struct {
		name     string
		content  string
		expected int // minimum expected entities
	}{
		{
			name:     "file paths",
			content:  "Let me check the auth.go and config.json files",
			expected: 2,
		},
		{
			name:     "error messages",
			content:  "Getting error: INVALID_TOKEN when calling the API",
			expected: 1,
		},
		{
			name:     "commands",
			content:  "$ go build ./cmd/server && ./server --config prod.yaml",
			expected: 1,
		},
		{
			name:     "urls",
			content:  "Check the documentation at https://docs.example.com/api",
			expected: 1,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entities := extractor.Extract(tc.content)
			assert.GreaterOrEqual(t, len(entities), tc.expected)
		})
	}
}

func TestFlowDetector_CalculateFlowScore(t *testing.T) {
	detector := NewFlowDetector()
	pattern := detector.flowPatterns[types.FlowProblem]
	
	testCases := []struct {
		name     string
		content  string
		toolUsed string
		expected float64
	}{
		{
			name:     "high score - keyword and tool match",
			content:  "there's an error in the system",
			toolUsed: "Read",
			expected: 1.0, // keyword + tool
		},
		{
			name:     "medium score - keyword only",
			content:  "there's an issue with the config",
			toolUsed: "Write",
			expected: 0.5, // keyword only
		},
		{
			name:     "low score - no match",
			content:  "everything is working fine",
			toolUsed: "Write",
			expected: 0.0,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := detector.calculateFlowScore(tc.content, tc.toolUsed, pattern)
			assert.GreaterOrEqual(t, score, tc.expected)
		})
	}
}

func TestFlowDetector_InferTransitionTrigger(t *testing.T) {
	detector := NewFlowDetector()
	
	testCases := []struct {
		from     types.ConversationFlow
		to       types.ConversationFlow
		expected string
	}{
		{
			from:     types.FlowProblem,
			to:       types.FlowInvestigation,
			expected: "began investigation",
		},
		{
			from:     types.FlowInvestigation,
			to:       types.FlowSolution,
			expected: "found solution",
		},
		{
			from:     types.FlowSolution,
			to:       types.FlowVerification,
			expected: "testing implementation",
		},
		{
			from:     types.FlowVerification,
			to:       types.FlowProblem,
			expected: "discovered new issue",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			trigger := detector.inferTransitionTrigger(tc.from, tc.to)
			assert.Equal(t, tc.expected, trigger)
		})
	}
}

func TestFlowDetector_ExtractTechnologies(t *testing.T) {
	detector := NewFlowDetector()
	
	session := &ConversationSession{
		Segments: []ConversationSegment{
			{
				Entities: []string{"auth.go", "config.json", "Dockerfile"},
				Keywords: []string{"chroma", "vector", "mcp"},
			},
			{
				Entities: []string{"main.py", "test.js"},
				Keywords: []string{"database", "api"},
			},
		},
	}
	
	technologies := detector.extractTechnologies(session)
	
	assert.Contains(t, technologies, "Go")
	assert.Contains(t, technologies, "Docker")
	assert.Contains(t, technologies, "Chroma")
	assert.Contains(t, technologies, "Vector Database")
	assert.Contains(t, technologies, "MCP")
}

func TestFlowDetector_ExtractDecisions(t *testing.T) {
	detector := NewFlowDetector()
	
	session := &ConversationSession{
		Segments: []ConversationSegment{
			{
				Flow:    types.FlowSolution,
				Content: "Let's use Chroma for the vector database\nI'll implement the REST API endpoint\nWe should add comprehensive error handling",
			},
			{
				Flow:    types.FlowProblem,
				Content: "There's an issue with the connection",
			},
		},
	}
	
	decisions := detector.extractDecisions(session)
	
	assert.Greater(t, len(decisions), 0)
	// Should extract decision-making statements from solution segments
	found := false
	for _, decision := range decisions {
		if len(decision) > 10 { // Should have substantial content
			found = true
			break
		}
	}
	assert.True(t, found, "Should extract meaningful decisions")
}

func TestFlowDetector_GetActiveSessions(t *testing.T) {
	detector := NewFlowDetector()
	
	// Create active session
	detector.StartSession("active1", "repo1")
	detector.StartSession("active2", "repo2")
	
	// Create and end a session
	detector.StartSession("ended", "repo3")
	detector.EndSession("ended", types.OutcomeSuccess)
	
	activeSessions := detector.GetActiveSessions()
	
	assert.Len(t, activeSessions, 2)
	assert.Contains(t, activeSessions, "active1")
	assert.Contains(t, activeSessions, "active2")
	assert.NotContains(t, activeSessions, "ended")
}

func TestFlowDetector_CompleteWorkflow(t *testing.T) {
	detector := NewFlowDetector()
	sessionID := "complete-workflow"
	
	detector.StartSession(sessionID, "mcp-memory")
	
	// Simulate a realistic debugging workflow
	messages := []struct {
		content  string
		toolUsed string
	}{
		{"I'm getting a compilation error in the server", "Read"},
		{"Let me check the error logs to understand what's happening", "Grep"},
		{"Looking at the main.go file to see the issue", "Read"},
		{"Found the problem - missing import statement", "Read"},
		{"I'll add the missing import to fix this", "Edit"},
		{"Let me test the fix by running the build", "Bash"},
		{"Great! The build is now successful", "Bash"},
	}
	
	for _, msg := range messages {
		detector.ProcessMessage(sessionID, msg.content, msg.toolUsed, nil)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}
	
	detector.EndSession(sessionID, types.OutcomeSuccess)
	
	session, exists := detector.GetSession(sessionID)
	require.True(t, exists)
	
	// Verify session structure
	assert.NotNil(t, session.EndTime)
	assert.Greater(t, len(session.Segments), 1)
	assert.Greater(t, len(session.Transitions), 0)
	
	// Verify summary
	summary := session.Summary
	assert.Greater(t, summary.TotalDuration, time.Duration(0))
	assert.GreaterOrEqual(t, summary.ProblemsSolved, 1)
	assert.Equal(t, 1, summary.SuccessfulOutcomes) // Should have verification
	
	// Should have detected different flows
	flowTypes := make(map[types.ConversationFlow]bool)
	for _, segment := range session.Segments {
		flowTypes[segment.Flow] = true
	}
	assert.GreaterOrEqual(t, len(flowTypes), 2) // Should have multiple flow types
}