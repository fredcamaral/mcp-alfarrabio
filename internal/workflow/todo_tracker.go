// Package workflow provides intelligent tracking of Claude's workflow patterns
package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// Status constants
const (
	statusCompleted = "completed"
)

// TodoItem represents a todo item from Claude's TodoWrite tool
type TodoItem struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Status   string `json:"status"` // pending, in_progress, completed, cancelled
	Priority string `json:"priority"`
}

// TodoSession tracks the complete lifecycle of todos in a session
type TodoSession struct {
	SessionID    string        `json:"session_id"`
	Repository   string        `json:"repository"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      *time.Time    `json:"end_time,omitempty"`
	Todos        []TodoItem    `json:"todos"`
	WorkContext  string        `json:"work_context"`
	Outcomes     []string      `json:"outcomes"`
	FilesChanged []string      `json:"files_changed"`
	ToolsUsed    []string      `json:"tools_used"`
	Status       types.Outcome `json:"status"`
}

// TodoTracker monitors and analyzes todo-driven workflows
type TodoTracker struct {
	activeSessions map[string]*TodoSession // key format: "repository::sessionID"
	completedWork  []types.ConversationChunk
}

// NewTodoTracker creates a new todo workflow tracker
func NewTodoTracker() *TodoTracker {
	return &TodoTracker{
		activeSessions: make(map[string]*TodoSession),
		completedWork:  make([]types.ConversationChunk, 0),
	}
}

// createSessionKey creates a composite key for multi-tenant session isolation
func (tt *TodoTracker) createSessionKey(repository, sessionID string) string {
	if repository == "" {
		repository = "unknown"
	}
	return repository + "::" + sessionID
}

// ProcessTodoWrite handles a TodoWrite operation from Claude
func (tt *TodoTracker) ProcessTodoWrite(ctx context.Context, sessionID, repository string, todos []TodoItem) error {
	session := tt.getOrCreateSession(sessionID, repository)

	// Track todo changes
	previousTodos := make(map[string]TodoItem)
	for _, todo := range session.Todos {
		previousTodos[todo.ID] = todo
	}

	// Detect completed todos and trigger chunk creation BEFORE updating session
	for _, todo := range todos {
		if previous, exists := previousTodos[todo.ID]; exists {
			if previous.Status != statusCompleted && todo.Status == statusCompleted {
				// Todo was just completed - capture the work
				if err := tt.captureCompletedWork(ctx, session, todo); err != nil {
					return fmt.Errorf("failed to capture completed work: %w", err)
				}
			}
		}
	}

	// Update session with new todos AFTER processing changes
	session.Todos = todos

	return nil
}

// ProcessToolUsage tracks tool usage during todo work
func (tt *TodoTracker) ProcessToolUsage(sessionID, repository, toolName string, toolContext map[string]interface{}) {
	key := tt.createSessionKey(repository, sessionID)
	if session, exists := tt.activeSessions[key]; exists {
		// Add tool to used tools list
		for _, used := range session.ToolsUsed {
			if used == toolName {
				return // Already tracked
			}
		}
		session.ToolsUsed = append(session.ToolsUsed, toolName)

		// Extract file information from tool context
		if files := tt.extractFilesFromContext(toolContext); len(files) > 0 {
			session.FilesChanged = append(session.FilesChanged, files...)
		}

		// Update work context with tool usage
		tt.updateWorkContext(session, toolName, toolContext)
	}
}

// captureCompletedWork creates a conversation chunk from completed todo work
func (tt *TodoTracker) captureCompletedWork(_ context.Context, session *TodoSession, completedTodo TodoItem) error {
	// Build comprehensive content from the todo journey
	content := tt.buildTodoJourneyContent(session, completedTodo)

	// Create metadata for the chunk
	timeSpentMinutes := int(tt.calculateTimeSpent(session, completedTodo).Minutes())
	metadata := types.ChunkMetadata{
		Tags:          tt.extractTags(session, completedTodo),
		Difficulty:    tt.assessDifficulty(session, completedTodo),
		Outcome:       types.OutcomeSuccess, // Assume success since todo completed
		TimeSpent:     &timeSpentMinutes,
		FilesModified: session.FilesChanged,
		ToolsUsed:     session.ToolsUsed,
		Repository:    session.Repository,
	}

	// Create the conversation chunk
	chunk, err := types.NewConversationChunk(
		session.SessionID,
		content,
		types.ChunkTypeSolution,
		&metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create chunk: %w", err)
	}

	// Add to completed work
	tt.completedWork = append(tt.completedWork, *chunk)

	return nil
}

// buildTodoJourneyContent creates rich content from the todo completion journey
func (tt *TodoTracker) buildTodoJourneyContent(session *TodoSession, todo TodoItem) string {
	var content strings.Builder

	// Todo information
	content.WriteString(fmt.Sprintf("# Completed: %s\n\n", todo.Content))
	content.WriteString(fmt.Sprintf("**Priority**: %s\n", todo.Priority))
	content.WriteString(fmt.Sprintf("**Repository**: %s\n\n", session.Repository))

	// Work context
	if session.WorkContext != "" {
		content.WriteString("## Work Context\n")
		content.WriteString(session.WorkContext)
		content.WriteString("\n\n")
	}

	// Tools used
	if len(session.ToolsUsed) > 0 {
		content.WriteString("## Tools Used\n")
		for _, tool := range session.ToolsUsed {
			content.WriteString(fmt.Sprintf("- %s\n", tool))
		}
		content.WriteString("\n")
	}

	// Files changed
	if len(session.FilesChanged) > 0 {
		content.WriteString("## Files Modified\n")
		for _, file := range session.FilesChanged {
			content.WriteString(fmt.Sprintf("- %s\n", file))
		}
		content.WriteString("\n")
	}

	// Related todos (context)
	activeTodos := tt.getActiveTodos(session)
	if len(activeTodos) > 0 {
		content.WriteString("## Related Active Tasks\n")
		for _, relatedTodo := range activeTodos {
			content.WriteString(fmt.Sprintf("- %s (%s)\n", relatedTodo.Content, relatedTodo.Status))
		}
		content.WriteString("\n")
	}

	return content.String()
}

// GetOrCreateSession retrieves or creates a new todo session (exported)
func (tt *TodoTracker) GetOrCreateSession(sessionID, repository string) *TodoSession {
	return tt.getOrCreateSession(sessionID, repository)
}

// getOrCreateSession retrieves or creates a new todo session
func (tt *TodoTracker) getOrCreateSession(sessionID, repository string) *TodoSession {
	key := tt.createSessionKey(repository, sessionID)
	if session, exists := tt.activeSessions[key]; exists {
		return session
	}

	session := &TodoSession{
		SessionID:    sessionID,
		Repository:   repository,
		StartTime:    time.Now(),
		Todos:        make([]TodoItem, 0),
		Outcomes:     make([]string, 0),
		FilesChanged: make([]string, 0),
		ToolsUsed:    make([]string, 0),
		Status:       types.OutcomeInProgress,
	}

	tt.activeSessions[key] = session
	return session
}

// extractFilesFromContext extracts file paths from tool usage context
func (tt *TodoTracker) extractFilesFromContext(toolContext map[string]interface{}) []string {
	files := make([]string, 0)

	// Look for common file path keys
	fileKeys := []string{"file_path", "filepath", "path", "filename"}

	for _, key := range fileKeys {
		if value, exists := toolContext[key]; exists {
			if filePath, ok := value.(string); ok && filePath != "" {
				files = append(files, filePath)
			}
		}
	}

	return files
}

// updateWorkContext updates the session's work context with tool information
func (tt *TodoTracker) updateWorkContext(session *TodoSession, toolName string, toolContext map[string]interface{}) {
	// Add tool usage information to work context
	contextJSON, _ := json.Marshal(toolContext)
	addition := fmt.Sprintf("[%s] %s: %s\n", time.Now().Format("15:04"), toolName, string(contextJSON))
	session.WorkContext += addition
}

// extractTags generates tags from todo and session context
func (tt *TodoTracker) extractTags(session *TodoSession, todo TodoItem) []string {
	tags := make([]string, 0)

	// Add priority as tag
	tags = append(tags, "priority-"+todo.Priority)

	// Add repository
	if session.Repository != "" {
		tags = append(tags, "repo-"+session.Repository)
	}

	// Extract technology tags from tools used
	for _, tool := range session.ToolsUsed {
		switch tool {
		case "Bash":
			tags = append(tags, "bash", "terminal")
		case "Read", "Write", "Edit":
			tags = append(tags, "file-editing")
		case "Grep", "Glob":
			tags = append(tags, "search")
		case "Git":
			tags = append(tags, "git", "version-control")
		}
	}

	// Extract language tags from file extensions
	for _, file := range session.FilesChanged {
		switch {
		case strings.HasSuffix(file, ".go"):
			tags = append(tags, "golang")
		case strings.HasSuffix(file, ".js") || strings.HasSuffix(file, ".ts"):
			tags = append(tags, "javascript", "typescript")
		case strings.HasSuffix(file, ".py"):
			tags = append(tags, "python")
		}
	}

	return tags
}

// assessDifficulty estimates the difficulty based on session data
func (tt *TodoTracker) assessDifficulty(session *TodoSession, todo TodoItem) types.Difficulty {
	score := 0

	// Factor in time spent
	timeSpent := tt.calculateTimeSpent(session, todo)
	if timeSpent > 30*time.Minute {
		score += 2
	} else if timeSpent > 10*time.Minute {
		score++
	}

	// Factor in number of tools used
	if len(session.ToolsUsed) > 5 {
		score += 2
	} else if len(session.ToolsUsed) > 2 {
		score++
	}

	// Factor in number of files changed
	if len(session.FilesChanged) > 3 {
		score++
	}

	// Convert score to difficulty
	switch {
	case score >= 4:
		return types.DifficultyComplex
	case score >= 2:
		return types.DifficultyModerate
	default:
		return types.DifficultySimple
	}
}

// calculateTimeSpent estimates time spent on a todo
func (tt *TodoTracker) calculateTimeSpent(session *TodoSession, _ TodoItem) time.Duration {
	// Simple estimation based on session duration
	// In a real implementation, we'd track individual todo timestamps
	return time.Since(session.StartTime)
}

// getActiveTodos returns todos that are still active (not completed)
func (tt *TodoTracker) getActiveTodos(session *TodoSession) []TodoItem {
	active := make([]TodoItem, 0)
	for _, todo := range session.Todos {
		if todo.Status != statusCompleted && todo.Status != "cancelled" {
			active = append(active, todo)
		}
	}
	return active
}

// GetCompletedWork returns all captured work chunks
func (tt *TodoTracker) GetCompletedWork() []types.ConversationChunk {
	return tt.completedWork
}

// GetActiveSession returns the current session if it exists (requires repository for multi-tenant isolation)
func (tt *TodoTracker) GetActiveSession(sessionID, repository string) (*TodoSession, bool) {
	key := tt.createSessionKey(repository, sessionID)
	session, exists := tt.activeSessions[key]
	return session, exists
}

// EndSession marks a session as complete
func (tt *TodoTracker) EndSession(sessionID, repository string, outcome types.Outcome) {
	key := tt.createSessionKey(repository, sessionID)
	if session, exists := tt.activeSessions[key]; exists {
		now := time.Now()
		session.EndTime = &now
		session.Status = outcome

		// Move to completed work if successful
		if outcome == types.OutcomeSuccess {
			// Create a final summary chunk
			tt.createSessionSummary(session)
		}

		// Remove from active sessions
		delete(tt.activeSessions, key)
	}
}

// createSessionSummary creates a summary chunk for the entire session
func (tt *TodoTracker) createSessionSummary(session *TodoSession) {
	content := tt.buildSessionSummaryContent(session)

	timeSpentMinutes := int(time.Since(session.StartTime).Minutes())
	metadata := types.ChunkMetadata{
		Tags:          []string{"session-summary", "repo-" + session.Repository},
		Difficulty:    tt.assessSessionDifficulty(session),
		Outcome:       session.Status,
		TimeSpent:     &timeSpentMinutes,
		FilesModified: session.FilesChanged,
		ToolsUsed:     session.ToolsUsed,
		Repository:    session.Repository,
	}

	chunk, err := types.NewConversationChunk(
		session.SessionID,
		content,
		types.ChunkTypeSessionSummary,
		&metadata,
	)
	if err == nil {
		tt.completedWork = append(tt.completedWork, *chunk)
	}
}

// buildSessionSummaryContent creates content for session summary
func (tt *TodoTracker) buildSessionSummaryContent(session *TodoSession) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# Session Summary: %s\n\n", session.Repository))
	content.WriteString(fmt.Sprintf("**Duration**: %v\n", time.Since(session.StartTime)))
	content.WriteString(fmt.Sprintf("**Status**: %s\n\n", session.Status))

	// Completed todos
	completedTodos := make([]TodoItem, 0)
	for _, todo := range session.Todos {
		if todo.Status == statusCompleted {
			completedTodos = append(completedTodos, todo)
		}
	}

	if len(completedTodos) > 0 {
		content.WriteString("## Completed Tasks\n")
		for _, todo := range completedTodos {
			content.WriteString(fmt.Sprintf("- %s\n", todo.Content))
		}
		content.WriteString("\n")
	}

	// Tools and files summary
	content.WriteString("## Impact\n")
	content.WriteString(fmt.Sprintf("- **Tools Used**: %d (%s)\n", len(session.ToolsUsed), strings.Join(session.ToolsUsed, ", ")))
	content.WriteString(fmt.Sprintf("- **Files Changed**: %d\n", len(session.FilesChanged)))

	return content.String()
}

// GetActiveSessions returns all active sessions
func (tt *TodoTracker) GetActiveSessions() map[string]*TodoSession {
	return tt.activeSessions
}

// GetActiveSessionsByRepository returns active sessions filtered by repository
func (tt *TodoTracker) GetActiveSessionsByRepository(repository string) map[string]*TodoSession {
	filteredSessions := make(map[string]*TodoSession)
	for key, session := range tt.activeSessions {
		if session.Repository == repository {
			filteredSessions[key] = session
		}
	}
	return filteredSessions
}

// assessSessionDifficulty assesses overall session difficulty
func (tt *TodoTracker) assessSessionDifficulty(session *TodoSession) types.Difficulty {
	totalTodos := len(session.Todos)
	completedTodos := 0

	for _, todo := range session.Todos {
		if todo.Status == statusCompleted {
			completedTodos++
		}
	}

	// Factor in completion rate, duration, and complexity
	duration := time.Since(session.StartTime)
	score := 0

	if duration > time.Hour {
		score += 2
	} else if duration > 30*time.Minute {
		score++
	}

	if totalTodos > 5 {
		score++
	}

	if len(session.ToolsUsed) > 8 {
		score++
	}

	switch {
	case score >= 3:
		return types.DifficultyComplex
	case score >= 1:
		return types.DifficultyModerate
	default:
		return types.DifficultySimple
	}
}
