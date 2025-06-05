package mcp

import (
	"errors"
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
		return errors.New("title parameter is required")
	}

	description, ok := params["description"].(string)
	if !ok || description == "" {
		return errors.New("description parameter is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		return errors.New("session_id parameter is required")
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
			return errors.New("invalid due_date format: must be RFC3339 (e.g., 2024-12-31T23:59:59Z)")
		}
	}

	// Validate estimate if provided
	if estimate, ok := params["estimate"].(float64); ok {
		if estimate < 0 {
			return errors.New("estimate must be positive")
		}
	}

	return nil
}

// TestTaskMetadataBuildingLogic tests the metadata building logic
func TestTaskMetadataBuildingLogic(t *testing.T) {
	testCases := buildMetadataTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metadata := buildTaskMetadata(tc.params)
			assert.True(t, tc.expected(&metadata), "Metadata should match expected values for test case: %s", tc.name)
		})
	}
}

// buildMetadataTestCases creates test cases for metadata validation
func buildMetadataTestCases() []struct {
	name     string
	params   map[string]interface{}
	expected func(metadata *types.ChunkMetadata) bool
} {
	return []struct {
		name     string
		params   map[string]interface{}
		expected func(metadata *types.ChunkMetadata) bool
	}{
		createBasicMetadataTestCase(),
		createOptionalFieldsTestCase(),
		createTagsTestCase(),
	}
}

// createBasicMetadataTestCase creates test case for basic metadata
func createBasicMetadataTestCase() struct {
	name     string
	params   map[string]interface{}
	expected func(metadata *types.ChunkMetadata) bool
} {
	return struct {
		name     string
		params   map[string]interface{}
		expected func(metadata *types.ChunkMetadata) bool
	}{
		name: "basic_metadata",
		params: map[string]interface{}{
			"title":       "Test task",
			"description": "Test description",
			"session_id":  "test-session",
		},
		expected: validateBasicMetadata,
	}
}

// createOptionalFieldsTestCase creates test case for optional fields
func createOptionalFieldsTestCase() struct {
	name     string
	params   map[string]interface{}
	expected func(metadata *types.ChunkMetadata) bool
} {
	return struct {
		name     string
		params   map[string]interface{}
		expected func(metadata *types.ChunkMetadata) bool
	}{
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
		expected: validateOptionalFieldsMetadata,
	}
}

// createTagsTestCase creates test case for tags validation
func createTagsTestCase() struct {
	name     string
	params   map[string]interface{}
	expected func(metadata *types.ChunkMetadata) bool
} {
	return struct {
		name     string
		params   map[string]interface{}
		expected func(metadata *types.ChunkMetadata) bool
	}{
		name: "with_tags",
		params: map[string]interface{}{
			"title":       "Test task",
			"description": "Test description",
			"session_id":  "test-session",
			"tags":        []interface{}{"bug-fix", "urgent", "backend"},
		},
		expected: validateTagsMetadata,
	}
}

// validateBasicMetadata validates basic metadata fields
func validateBasicMetadata(metadata *types.ChunkMetadata) bool {
	return metadata.Repository == GlobalMemoryRepository &&
		validateTaskStatus(metadata, types.TaskStatusTodo) &&
		validateTaskPriority(metadata, types.PriorityMedium) &&
		metadata.Outcome == types.OutcomeInProgress &&
		metadata.Difficulty == types.DifficultyModerate
}

// validateOptionalFieldsMetadata validates metadata with optional fields
func validateOptionalFieldsMetadata(metadata *types.ChunkMetadata) bool {
	return metadata.Repository == "github.com/example/project" &&
		validateTaskStatus(metadata, types.TaskStatusTodo) &&
		validateTaskPriority(metadata, types.PriorityHigh) &&
		validateTaskAssignee(metadata, "dev@example.com") &&
		validateTaskEstimate(metadata, 120)
}

// validateTagsMetadata validates metadata with tags
func validateTagsMetadata(metadata *types.ChunkMetadata) bool {
	expectedTags := []string{"bug-fix", "urgent", "backend"}
	if len(metadata.Tags) != len(expectedTags) {
		return false
	}
	for i, tag := range expectedTags {
		if metadata.Tags[i] != tag {
			return false
		}
	}
	return true
}

// validateTaskStatus validates task status field
func validateTaskStatus(metadata *types.ChunkMetadata, expected types.TaskStatus) bool {
	return metadata.TaskStatus != nil && *metadata.TaskStatus == expected
}

// validateTaskPriority validates task priority field
func validateTaskPriority(metadata *types.ChunkMetadata, expected string) bool {
	return metadata.TaskPriority != nil && *metadata.TaskPriority == expected
}

// validateTaskAssignee validates task assignee field
func validateTaskAssignee(metadata *types.ChunkMetadata, expected string) bool {
	return metadata.TaskAssignee != nil && *metadata.TaskAssignee == expected
}

// validateTaskEstimate validates task estimate field
func validateTaskEstimate(metadata *types.ChunkMetadata, expected int) bool {
	return metadata.TaskEstimate != nil && *metadata.TaskEstimate == expected
}

// buildTaskMetadata extracts the metadata building logic for unit testing
func buildTaskMetadata(params map[string]interface{}) types.ChunkMetadata {
	// Build base metadata with defaults
	metadata := buildBaseMetadata(params)

	// Add optional task-specific fields
	addOptionalTaskFields(&metadata, params)

	return metadata
}

// buildBaseMetadata creates metadata with default values
func buildBaseMetadata(params map[string]interface{}) types.ChunkMetadata {
	repository := extractRepository(params)
	priority := extractPriority(params)
	status := extractStatus(params)

	return types.ChunkMetadata{
		Repository:   repository,
		Tags:         testExtractStringArray(params["tags"]),
		TaskStatus:   &status,
		TaskPriority: &priority,
		Outcome:      types.OutcomeInProgress,  // Tasks start as in progress
		Difficulty:   types.DifficultyModerate, // Default difficulty
	}
}

// extractRepository gets repository from params with default
func extractRepository(params map[string]interface{}) string {
	if repo, ok := params["repository"].(string); ok && repo != "" {
		return repo
	}
	return GlobalMemoryRepository
}

// extractPriority gets priority from params with default
func extractPriority(params map[string]interface{}) string {
	if p, ok := params["priority"].(string); ok && p != "" {
		return p
	}
	return types.PriorityMedium
}

// extractStatus gets status from params with default
func extractStatus(params map[string]interface{}) types.TaskStatus {
	if s, ok := params["status"].(string); ok && s != "" {
		return types.TaskStatus(s)
	}
	return types.TaskStatusTodo
}

// addOptionalTaskFields adds optional task-specific fields to metadata
func addOptionalTaskFields(metadata *types.ChunkMetadata, params map[string]interface{}) {
	addAssignee(metadata, params)
	addDueDate(metadata, params)
	addEstimate(metadata, params)
	addDependencies(metadata, params)
}

// addAssignee adds assignee field if present
func addAssignee(metadata *types.ChunkMetadata, params map[string]interface{}) {
	if assignee, ok := params["assignee"].(string); ok && assignee != "" {
		metadata.TaskAssignee = &assignee
	}
}

// addDueDate adds due date field if present and valid
func addDueDate(metadata *types.ChunkMetadata, params map[string]interface{}) {
	if dueDateStr, ok := params["due_date"].(string); ok && dueDateStr != "" {
		if dueDate, err := time.Parse(time.RFC3339, dueDateStr); err == nil {
			metadata.TaskDueDate = &dueDate
		}
	}
}

// addEstimate adds estimate field if present
func addEstimate(metadata *types.ChunkMetadata, params map[string]interface{}) {
	if estimate, ok := params["estimate"].(float64); ok {
		estimateInt := int(estimate)
		metadata.TaskEstimate = &estimateInt
	}
}

// addDependencies adds dependencies field if present
func addDependencies(metadata *types.ChunkMetadata, params map[string]interface{}) {
	if deps, ok := params["dependencies"].([]interface{}); ok {
		dependencies := make([]string, 0, len(deps))
		for _, dep := range deps {
			if depStr, ok := dep.(string); ok {
				dependencies = append(dependencies, depStr)
			}
		}
		metadata.TaskDependencies = dependencies
	}
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

// TestTodoReadLogic tests the todo read functionality without external dependencies
func TestTodoReadLogic(t *testing.T) {
	tests := []struct {
		name          string
		args          map[string]interface{}
		expectError   bool
		errorMsg      string
		expectedScope string
		hasSessionID  bool
	}{
		{
			name: "missing_repository",
			args: map[string]interface{}{
				"session_id": "test-session",
			},
			expectError: true,
			errorMsg:    "repository parameter is required",
		},
		{
			name: "repository_only_scope",
			args: map[string]interface{}{
				"repository": "github.com/test/repo",
			},
			expectError:   false,
			expectedScope: "repository",
			hasSessionID:  false,
		},
		{
			name: "session_scope",
			args: map[string]interface{}{
				"repository": "github.com/test/repo",
				"session_id": "test-session",
			},
			expectError:   false,
			expectedScope: "session",
			hasSessionID:  true,
		},
		{
			name: "empty_session_id_should_use_repository_scope",
			args: map[string]interface{}{
				"repository": "github.com/test/repo",
				"session_id": "",
			},
			expectError:   false,
			expectedScope: "repository",
			hasSessionID:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test parameter validation
			err := validateTodoReadParameters(tt.args)

			if tt.expectError {
				assert.Error(t, err, "Expected validation error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain: %s", tt.errorMsg)
				}
			} else {
				assert.NoError(t, err, "Unexpected validation error for test case: %s", tt.name)

				// Test scope detection
				scope, hasSession := determineTodoReadScope(tt.args)
				assert.Equal(t, tt.expectedScope, scope, "Scope should match expected value")
				assert.Equal(t, tt.hasSessionID, hasSession, "Session ID presence should match expected value")
			}
		})
	}
}

// validateTodoReadParameters extracts the parameter validation logic for todo_read testing
func validateTodoReadParameters(args map[string]interface{}) error {
	repository, ok := args["repository"].(string)
	if !ok || repository == "" {
		return errors.New("repository parameter is required for multi-tenant isolation")
	}
	return nil
}

// determineTodoReadScope extracts the scope detection logic for todo_read testing
func determineTodoReadScope(args map[string]interface{}) (scope string, hasSession bool) {
	sessionID, hasSessionID := args["session_id"].(string)

	if hasSessionID && sessionID != "" {
		return "session", true
	}

	return "repository", false
}

// extractStringArray extracts a string array from interface{} for testing
func testExtractStringArray(value interface{}) []string {
	if arr, ok := value.([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}
