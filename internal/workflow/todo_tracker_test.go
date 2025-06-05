package workflow

import (
	"context"
	"testing"
	"time"

	"lerian-mcp-memory/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testRepo        = "test-repo"
	testTodoSession = "test-session"
)

func TestNewTodoTracker(t *testing.T) {
	tracker := NewTodoTracker()

	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.activeSessions)
	assert.NotNil(t, tracker.completedWork)
	assert.Len(t, tracker.activeSessions, 0)
	assert.Len(t, tracker.completedWork, 0)
}

func TestTodoTracker_ProcessTodoWrite(t *testing.T) {
	t.Run("creates new session", func(t *testing.T) {
		tracker := NewTodoTracker()
		ctx := context.Background()
		sessionID := "test-session-1"
		repository := testRepo
		todos := []TodoItem{
			{ID: "1", Content: "Fix bug", Status: "pending", Priority: "high"},
		}

		err := tracker.ProcessTodoWrite(ctx, sessionID, repository, todos)
		require.NoError(t, err)

		session, exists := tracker.GetActiveSession(sessionID, repository)
		assert.True(t, exists)
		assert.Equal(t, sessionID, session.SessionID)
		assert.Equal(t, repository, session.Repository)
		assert.Len(t, session.Todos, 1)
	})

	t.Run("detects completed todos", func(t *testing.T) {
		tracker := NewTodoTracker()
		ctx := context.Background()
		sessionID := "test-session-2"
		repository := testRepo

		// First, add a todo in pending state
		todos := []TodoItem{
			{ID: "1", Content: "Fix bug", Status: "pending", Priority: "high"},
		}
		err := tracker.ProcessTodoWrite(ctx, sessionID, repository, todos)
		require.NoError(t, err)

		// Then mark it as completed (create new slice to avoid modifying original)
		completedTodos := []TodoItem{
			{ID: "1", Content: "Fix bug", Status: "completed", Priority: "high"},
		}
		err = tracker.ProcessTodoWrite(ctx, sessionID, repository, completedTodos)
		require.NoError(t, err)

		// Should have captured work
		completedWork := tracker.GetCompletedWork()
		assert.Len(t, completedWork, 1)
		assert.Contains(t, completedWork[0].Content, "Fix bug")
		assert.Equal(t, types.ChunkTypeSolution, completedWork[0].Type)
	})
}

func TestTodoTracker_ProcessToolUsage(t *testing.T) {
	tracker := NewTodoTracker()
	sessionID := testTodoSession
	repository := testRepo

	// Create a session first
	todos := []TodoItem{
		{ID: "1", Content: "Test task", Status: "in_progress", Priority: "medium"},
	}
	err := tracker.ProcessTodoWrite(context.Background(), sessionID, repository, todos)
	require.NoError(t, err)

	t.Run("tracks tool usage", func(t *testing.T) {
		tracker.ProcessToolUsage(sessionID, repository, "Bash", map[string]interface{}{
			"command": "ls -la",
		})

		session, exists := tracker.GetActiveSession(sessionID, repository)
		require.True(t, exists)
		assert.Contains(t, session.ToolsUsed, "Bash")
	})

	t.Run("extracts file information", func(t *testing.T) {
		tracker.ProcessToolUsage(sessionID, repository, "Edit", map[string]interface{}{
			"file_path": "/path/to/file.go",
		})

		session, exists := tracker.GetActiveSession(sessionID, repository)
		require.True(t, exists)
		assert.Contains(t, session.FilesChanged, "/path/to/file.go")
	})

	t.Run("ignores non-existent session", func(t *testing.T) {
		// Should not panic
		tracker.ProcessToolUsage("non-existent", testRepo, "Read", map[string]interface{}{})

		// Session should not be created
		_, exists := tracker.GetActiveSession("non-existent", testRepo)
		assert.False(t, exists)
	})
}

func TestTodoTracker_ExtractTags(t *testing.T) {
	tracker := NewTodoTracker()

	session := &TodoSession{
		Repository:   testRepo,
		ToolsUsed:    []string{"Bash", "Edit", "Grep"},
		FilesChanged: []string{"/path/file.go", "/path/file.js"},
	}

	todo := TodoItem{
		Priority: "high",
	}

	tags := tracker.extractTags(session, todo)

	assert.Contains(t, tags, "priority-high")
	assert.Contains(t, tags, "repo-test-repo")
	assert.Contains(t, tags, "bash")
	assert.Contains(t, tags, "terminal")
	assert.Contains(t, tags, "file-editing")
	assert.Contains(t, tags, "search")
	assert.Contains(t, tags, "golang")
	assert.Contains(t, tags, "javascript")
}

func TestTodoTracker_AssessDifficulty(t *testing.T) {
	tracker := NewTodoTracker()

	t.Run("easy task", func(t *testing.T) {
		session := &TodoSession{
			StartTime:    time.Now().Add(-5 * time.Minute),
			ToolsUsed:    []string{"Read"},
			FilesChanged: []string{},
		}

		todo := TodoItem{}

		difficulty := tracker.assessDifficulty(session, todo)
		assert.Equal(t, types.DifficultySimple, difficulty)
	})

	t.Run("medium task", func(t *testing.T) {
		session := &TodoSession{
			StartTime:    time.Now().Add(-15 * time.Minute),
			ToolsUsed:    []string{"Read", "Edit", "Bash"},
			FilesChanged: []string{"file1.go", "file2.go"},
		}

		todo := TodoItem{}

		difficulty := tracker.assessDifficulty(session, todo)
		assert.Equal(t, types.DifficultyModerate, difficulty)
	})

	t.Run("hard task", func(t *testing.T) {
		session := &TodoSession{
			StartTime:    time.Now().Add(-45 * time.Minute),
			ToolsUsed:    []string{"Read", "Edit", "Bash", "Grep", "Glob", "Write"},
			FilesChanged: []string{"file1.go", "file2.go", "file3.go", "file4.go"},
		}

		todo := TodoItem{}

		difficulty := tracker.assessDifficulty(session, todo)
		assert.Equal(t, types.DifficultyComplex, difficulty)
	})
}

func TestTodoTracker_BuildTodoJourneyContent(t *testing.T) {
	tracker := NewTodoTracker()

	session := &TodoSession{
		SessionID:    testTodoSession,
		Repository:   testRepo,
		WorkContext:  "Working on fixing authentication bug",
		ToolsUsed:    []string{"Read", "Edit", "Bash"},
		FilesChanged: []string{"auth.go", "auth_test.go"},
		Todos: []TodoItem{
			{ID: "1", Content: "Fix auth bug", Status: "completed", Priority: "high"},
			{ID: "2", Content: "Add tests", Status: "in_progress", Priority: "medium"},
		},
	}

	todo := TodoItem{
		ID:       "1",
		Content:  "Fix auth bug",
		Status:   "completed",
		Priority: "high",
	}

	content := tracker.buildTodoJourneyContent(session, todo)

	assert.Contains(t, content, "Fix auth bug")
	assert.Contains(t, content, "Priority**: high")
	assert.Contains(t, content, testRepo)
	assert.Contains(t, content, "Working on fixing authentication bug")
	assert.Contains(t, content, "Read")
	assert.Contains(t, content, "Edit")
	assert.Contains(t, content, "Bash")
	assert.Contains(t, content, "auth.go")
	assert.Contains(t, content, "auth_test.go")
	assert.Contains(t, content, "Add tests (in_progress)")
}

func TestTodoTracker_EndSession(t *testing.T) {
	tracker := NewTodoTracker()
	sessionID := testTodoSession
	repository := testRepo

	// Create session
	todos := []TodoItem{
		{ID: "1", Content: "Complete task", Status: "completed", Priority: "medium"},
	}
	err := tracker.ProcessTodoWrite(context.Background(), sessionID, repository, todos)
	require.NoError(t, err)

	// Add some tool usage
	tracker.ProcessToolUsage(sessionID, repository, "Edit", map[string]interface{}{
		"file_path": "test.go",
	})

	// End session successfully
	tracker.EndSession(sessionID, repository, types.OutcomeSuccess)

	// Session should be removed from active
	_, exists := tracker.GetActiveSession(sessionID, repository)
	assert.False(t, exists)

	// Should have created session summary
	completedWork := tracker.GetCompletedWork()
	// Should have at least the todo completion chunk and session summary
	assert.GreaterOrEqual(t, len(completedWork), 1)

	// Find session summary chunk
	hasSummary := false
	for _, chunk := range completedWork {
		if chunk.Type == types.ChunkTypeSessionSummary {
			hasSummary = true
			assert.Contains(t, chunk.Content, "Session Summary")
			assert.Contains(t, chunk.Content, repository)
			break
		}
	}
	assert.True(t, hasSummary, "Should have created session summary chunk")
}

func TestTodoTracker_ExtractFilesFromContext(t *testing.T) {
	tracker := NewTodoTracker()

	t.Run("extracts file_path", func(t *testing.T) {
		contextData := map[string]interface{}{
			"file_path": "/path/to/file.go",
		}

		files := tracker.extractFilesFromContext(contextData)
		assert.Contains(t, files, "/path/to/file.go")
	})

	t.Run("extracts multiple file keys", func(t *testing.T) {
		contextData := map[string]interface{}{
			"file_path": "/path/to/file1.go",
			"filepath":  "/path/to/file2.go",
		}

		files := tracker.extractFilesFromContext(contextData)
		assert.Contains(t, files, "/path/to/file1.go")
		assert.Contains(t, files, "/path/to/file2.go")
	})

	t.Run("ignores non-string values", func(t *testing.T) {
		contextData := map[string]interface{}{
			"file_path": 123,
			"path":      "",
		}

		files := tracker.extractFilesFromContext(contextData)
		assert.Len(t, files, 0)
	})
}

func TestTodoTracker_GetActiveTodos(t *testing.T) {
	tracker := NewTodoTracker()

	session := &TodoSession{
		Todos: []TodoItem{
			{ID: "1", Status: "pending"},
			{ID: "2", Status: "in_progress"},
			{ID: "3", Status: "completed"},
			{ID: "4", Status: "cancelled"},
		},
	}

	active := tracker.getActiveTodos(session)

	assert.Len(t, active, 2)
	assert.Equal(t, "1", active[0].ID)
	assert.Equal(t, "2", active[1].ID)
}
