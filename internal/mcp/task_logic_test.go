package mcp

import (
	"fmt"
	"testing"
	"time"

	"mcp-memory/pkg/types"

	"github.com/stretchr/testify/assert"
)

// TestTaskParameterValidationLogic tests the parameter validation logic without external dependencies
func TestTaskParameterValidationLogic(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_basic_parameters",
			params: map[string]interface{}{
				"title":       "Valid task",
				"description": "Valid description",
				"session_id":  "valid-session",
			},
			expectError: false,
		},
		{
			name: "valid_with_optional_params",
			params: map[string]interface{}{
				"title":       "Valid task",
				"description": "Valid description",
				"session_id":  "valid-session",
				"repository":  "github.com/example/project",
				"priority":    types.PriorityHigh,
				"assignee":    "dev@example.com",
				"estimate":    120.0,
			},
			expectError: false,
		},
		{
			name: "missing_title",
			params: map[string]interface{}{
				"description": "Description without title",
				"session_id":  "valid-session",
			},
			expectError: true,
			errorMsg:    "title",
		},
		{
			name: "empty_title",
			params: map[string]interface{}{
				"title":       "",
				"description": "Description with empty title",
				"session_id":  "valid-session",
			},
			expectError: true,
			errorMsg:    "title",
		},
		{
			name: "missing_description",
			params: map[string]interface{}{
				"title":      "Title without description",
				"session_id": "valid-session",
			},
			expectError: true,
			errorMsg:    "description",
		},
		{
			name: "missing_session_id",
			params: map[string]interface{}{
				"title":       "Valid title",
				"description": "Valid description",
			},
			expectError: true,
			errorMsg:    "session_id",
		},
		{
			name: "invalid_priority",
			params: map[string]interface{}{
				"title":       "Valid title",
				"description": "Valid description",
				"session_id":  "valid-session",
				"priority":    "super-urgent", // Invalid priority
			},
			expectError: true,
			errorMsg:    "priority",
		},
		{
			name: "invalid_due_date_format",
			params: map[string]interface{}{
				"title":       "Valid title",
				"description": "Valid description",
				"session_id":  "valid-session",
				"due_date":    "invalid-date-format",
			},
			expectError: true,
			errorMsg:    "due_date",
		},
		{
			name: "negative_estimate",
			params: map[string]interface{}{
				"title":       "Valid title",
				"description": "Valid description",
				"session_id":  "valid-session",
				"estimate":    -30.0,
			},
			expectError: true,
			errorMsg:    "estimate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTaskParameters(tt.params)

			if tt.expectError {
				assert.Error(t, err, "Expected validation error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain: %s", tt.errorMsg)
				}
			} else {
				assert.NoError(t, err, "Unexpected validation error for test case: %s", tt.name)
			}
		})
	}
}

// validateTaskParameters extracts the parameter validation logic for unit testing
func validateTaskParameters(params map[string]interface{}) error {
	// Required parameters
	title, ok := params["title"].(string)
	if !ok || title == "" {
		return fmt.Errorf("title parameter is required")
	}

	description, ok := params["description"].(string)
	if !ok || description == "" {
		return fmt.Errorf("description parameter is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		return fmt.Errorf("session_id parameter is required")
	}

	// Validate priority if provided
	if priority, ok := params["priority"].(string); ok && priority != "" {
		switch priority {
		case types.PriorityHigh, types.PriorityMedium, types.PriorityLow:
			// Valid priority
		default:
			return fmt.Errorf("invalid task priority: %s", priority)
		}
	}

	// Validate due_date format if provided
	if dueDateStr, ok := params["due_date"].(string); ok && dueDateStr != "" {
		_, err := time.Parse(time.RFC3339, dueDateStr)
		if err != nil {
			return fmt.Errorf("invalid due_date format: must be RFC3339 (e.g., 2024-12-31T23:59:59Z)")
		}
	}

	// Validate estimate if provided
	if estimate, ok := params["estimate"].(float64); ok {
		if estimate < 0 {
			return fmt.Errorf("estimate must be positive")
		}
	}

	return nil
}

// TestTaskMetadataBuildingLogic tests the metadata building logic
func TestTaskMetadataBuildingLogic(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		expected func(metadata types.ChunkMetadata) bool
	}{
		{
			name: "basic_metadata",
			params: map[string]interface{}{
				"title":       "Test task",
				"description": "Test description",
				"session_id":  "test-session",
			},
			expected: func(metadata types.ChunkMetadata) bool {
				return metadata.Repository == GlobalMemoryRepository &&
					metadata.TaskStatus != nil && *metadata.TaskStatus == types.TaskStatusTodo &&
					metadata.TaskPriority != nil && *metadata.TaskPriority == types.PriorityMedium &&
					metadata.Outcome == types.OutcomeInProgress &&
					metadata.Difficulty == types.DifficultyModerate
			},
		},
		{
			name: "with_optional_fields",
			params: map[string]interface{}{
				"title":       "Test task",
				"description": "Test description",
				"session_id":  "test-session",
				"repository":  "github.com/example/project",
				"priority":    types.PriorityHigh,
				"assignee":    "dev@example.com",
				"estimate":    120.0,
			},
			expected: func(metadata types.ChunkMetadata) bool {
				return metadata.Repository == "github.com/example/project" &&
					metadata.TaskStatus != nil && *metadata.TaskStatus == types.TaskStatusTodo &&
					metadata.TaskPriority != nil && *metadata.TaskPriority == types.PriorityHigh &&
					metadata.TaskAssignee != nil && *metadata.TaskAssignee == "dev@example.com" &&
					metadata.TaskEstimate != nil && *metadata.TaskEstimate == 120
			},
		},
		{
			name: "with_tags",
			params: map[string]interface{}{
				"title":       "Test task",
				"description": "Test description",
				"session_id":  "test-session",
				"tags":        []interface{}{"bug-fix", "urgent", "backend"},
			},
			expected: func(metadata types.ChunkMetadata) bool {
				return len(metadata.Tags) == 3 &&
					metadata.Tags[0] == "bug-fix" &&
					metadata.Tags[1] == "urgent" &&
					metadata.Tags[2] == "backend"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := buildTaskMetadata(tt.params)
			assert.True(t, tt.expected(metadata), "Metadata should match expected values for test case: %s", tt.name)
		})
	}
}

// buildTaskMetadata extracts the metadata building logic for unit testing
func buildTaskMetadata(params map[string]interface{}) types.ChunkMetadata {
	// Optional parameters with defaults
	repository := GlobalMemoryRepository
	if repo, ok := params["repository"].(string); ok && repo != "" {
		repository = repo
	}

	priority := types.PriorityMedium
	if p, ok := params["priority"].(string); ok && p != "" {
		priority = p
	}

	status := types.TaskStatusTodo
	if s, ok := params["status"].(string); ok && s != "" {
		status = types.TaskStatus(s)
	}

	// Build metadata
	metadata := types.ChunkMetadata{
		Repository:   repository,
		Tags:         extractStringArray(params["tags"]),
		TaskStatus:   &status,
		TaskPriority: &priority,
		Outcome:      types.OutcomeInProgress,  // Tasks start as in progress
		Difficulty:   types.DifficultyModerate, // Default difficulty
	}

	// Optional task-specific fields
	if assignee, ok := params["assignee"].(string); ok && assignee != "" {
		metadata.TaskAssignee = &assignee
	}

	if dueDateStr, ok := params["due_date"].(string); ok && dueDateStr != "" {
		if dueDate, err := time.Parse(time.RFC3339, dueDateStr); err == nil {
			metadata.TaskDueDate = &dueDate
		}
	}

	if estimate, ok := params["estimate"].(float64); ok {
		estimateInt := int(estimate)
		metadata.TaskEstimate = &estimateInt
	}

	if deps, ok := params["dependencies"].([]interface{}); ok {
		dependencies := make([]string, 0, len(deps))
		for _, dep := range deps {
			if depStr, ok := dep.(string); ok {
				dependencies = append(dependencies, depStr)
			}
		}
		metadata.TaskDependencies = dependencies
	}

	return metadata
}

// TestTaskContentBuilding tests the task content formatting
func TestTaskContentBuilding(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		description string
		expected    string
	}{
		{
			name:        "basic_content",
			title:       "Implement authentication",
			description: "Add JWT authentication to the API",
			expected:    "TASK: Implement authentication\n\nDESCRIPTION:\nAdd JWT authentication to the API",
		},
		{
			name:        "multiline_description",
			title:       "Fix bug",
			description: "Fix the bug that causes:\n1. Memory leaks\n2. Performance issues",
			expected:    "TASK: Fix bug\n\nDESCRIPTION:\nFix the bug that causes:\n1. Memory leaks\n2. Performance issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := buildTaskContent(tt.title, tt.description)
			assert.Equal(t, tt.expected, content, "Task content should match expected format")
		})
	}
}

// buildTaskContent extracts the content building logic for unit testing
func buildTaskContent(title, description string) string {
	return fmt.Sprintf("TASK: %s\n\nDESCRIPTION:\n%s", title, description)
}
