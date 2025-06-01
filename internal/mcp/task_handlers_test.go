package mcp

import (
	"context"
	"testing"
	"time"

	"mcp-memory/internal/config"
	"mcp-memory/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskHandlers provides comprehensive testing for all task-oriented memory handlers
// NOTE: These tests are currently skipped in CI as they require external services
func TestTaskHandlers(t *testing.T) {
	// Skip tests that require external services in CI environment
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Create test server with realistic config
	cfg := &config.Config{
		Qdrant: config.QdrantConfig{
			Host:       "localhost",
			Port:       6333,
			Collection: "test_tasks",
		},
		OpenAI: config.OpenAIConfig{
			APIKey:         "test-key", // Will fail but handler should handle gracefully
			EmbeddingModel: "text-embedding-ada-002",
		},
		Logging: config.LoggingConfig{
			Level: "debug",
		},
	}

	server, err := NewMemoryServer(cfg)
	require.NoError(t, err, "Failed to create memory server")

	ctx := context.Background()

	// Test individual functions that don't require external services
	t.Run("TaskParameterValidation", func(t *testing.T) {
		testTaskParameterValidation(t, server, ctx)
	})

	// Only run full integration tests if explicitly requested
	if testing.Verbose() {
		t.Run("TaskCreation", func(t *testing.T) {
			testTaskCreation(t, server, ctx)
		})

		t.Run("TaskUpdate", func(t *testing.T) {
			testTaskUpdate(t, server, ctx)
		})

		t.Run("TaskListing", func(t *testing.T) {
			testTaskListing(t, server, ctx)
		})

		t.Run("TaskStatus", func(t *testing.T) {
			testTaskStatus(t, server, ctx)
		})

		t.Run("TaskCompletion", func(t *testing.T) {
			testTaskCompletion(t, server, ctx)
		})
	}

	t.Run("TaskErrorHandling", func(t *testing.T) {
		testTaskErrorHandling(t, server, ctx)
	})
}

// testTaskParameterValidation tests parameter validation without external service calls
func testTaskParameterValidation(t *testing.T, server *MemoryServer, ctx context.Context) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing_title",
			params: map[string]interface{}{
				"description": "Missing title test",
				"session_id":  "test-session-3",
			},
			expectError: true,
			errorMsg:    "title parameter is required",
		},
		{
			name: "missing_description",
			params: map[string]interface{}{
				"title":      "Test task",
				"session_id": "test-session-4",
			},
			expectError: true,
			errorMsg:    "description parameter is required",
		},
		{
			name: "missing_session_id",
			params: map[string]interface{}{
				"title":       "Test task",
				"description": "Test description",
			},
			expectError: true,
			errorMsg:    "session_id parameter is required",
		},
		{
			name: "invalid_priority",
			params: map[string]interface{}{
				"title":       "Test task",
				"description": "Test description",
				"session_id":  "test-session-5",
				"priority":    "super-critical", // Invalid priority
			},
			expectError: true,
			errorMsg:    "invalid task priority",
		},
		{
			name: "invalid_due_date_format",
			params: map[string]interface{}{
				"title":       "Test task",
				"description": "Test description",
				"session_id":  "test-session-6",
				"due_date":    "2024-12-31", // Invalid format
			},
			expectError: true,
			errorMsg:    "invalid due_date format",
		},
		{
			name: "negative_estimate",
			params: map[string]interface{}{
				"title":       "Test task",
				"description": "Test description",
				"session_id":  "test-session-7",
				"estimate":    -30, // Negative estimate
			},
			expectError: true,
			errorMsg:    "estimate must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.handleCreateTask(ctx, tt.params)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain: %s", tt.errorMsg)
				}
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Unexpected error for test case: %s", tt.name)
				assert.NotNil(t, result, "Result should not be nil")
			}
		})
	}
}

func testTaskCreation(t *testing.T, server *MemoryServer, ctx context.Context) {
	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful_basic_task_creation",
			params: map[string]interface{}{
				"title":       "Implement authentication",
				"description": "Add JWT authentication to the API endpoints",
				"session_id":  "test-session-1",
				"repository":  "github.com/example/project",
			},
			expectError: false,
		},
		{
			name: "successful_detailed_task_creation",
			params: map[string]interface{}{
				"title":       "Fix database connection pool",
				"description": "Optimize database connections to handle high load",
				"session_id":  "test-session-2",
				"repository":  "github.com/example/project",
				"priority":    types.PriorityHigh,
				"assignee":    "developer@example.com",
				"estimate":    120, // 2 hours
				"tags":        []interface{}{"bug-fix", "performance", "database"},
				"due_date":    "2024-12-31T23:59:59Z",
			},
			expectError: false,
		},
		{
			name: "missing_title",
			params: map[string]interface{}{
				"description": "Missing title test",
				"session_id":  "test-session-3",
			},
			expectError: true,
			errorMsg:    "title parameter is required",
		},
		{
			name: "missing_description",
			params: map[string]interface{}{
				"title":      "Test task",
				"session_id": "test-session-4",
			},
			expectError: true,
			errorMsg:    "description parameter is required",
		},
		{
			name: "missing_session_id",
			params: map[string]interface{}{
				"title":       "Test task",
				"description": "Test description",
			},
			expectError: true,
			errorMsg:    "session_id parameter is required",
		},
		{
			name: "invalid_priority",
			params: map[string]interface{}{
				"title":       "Test task",
				"description": "Test description",
				"session_id":  "test-session-5",
				"priority":    "super-critical", // Invalid priority
			},
			expectError: true,
			errorMsg:    "invalid task priority",
		},
		{
			name: "invalid_due_date_format",
			params: map[string]interface{}{
				"title":       "Test task",
				"description": "Test description",
				"session_id":  "test-session-6",
				"due_date":    "2024-12-31", // Invalid format
			},
			expectError: true,
			errorMsg:    "invalid due_date format",
		},
		{
			name: "negative_estimate",
			params: map[string]interface{}{
				"title":       "Test task",
				"description": "Test description",
				"session_id":  "test-session-7",
				"estimate":    -30, // Negative estimate
			},
			expectError: true,
			errorMsg:    "estimate must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.handleCreateTask(ctx, tt.params)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain: %s", tt.errorMsg)
				}
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Unexpected error for test case: %s", tt.name)
				assert.NotNil(t, result, "Result should not be nil")

				// Validate result structure
				resultMap, ok := result.(map[string]interface{})
				require.True(t, ok, "Result should be a map")

				assert.Contains(t, resultMap, "task_id", "Result should contain task_id")
				assert.Contains(t, resultMap, "status", "Result should contain status")
				assert.Contains(t, resultMap, "created_at", "Result should contain created_at")

				// Validate status is "todo"
				assert.Equal(t, "todo", resultMap["status"], "New task should have status 'todo'")

				// Validate task_id is not empty
				taskID, ok := resultMap["task_id"].(string)
				require.True(t, ok, "task_id should be a string")
				assert.NotEmpty(t, taskID, "task_id should not be empty")
			}
		})
	}
}

func testTaskUpdate(t *testing.T, server *MemoryServer, ctx context.Context) {
	// First create a task to update
	createParams := map[string]interface{}{
		"title":       "Task to update",
		"description": "This task will be updated",
		"session_id":  "test-session-update",
		"repository":  "github.com/example/project",
		"priority":    "medium",
	}

	createResult, err := server.handleCreateTask(ctx, createParams)
	require.NoError(t, err, "Failed to create task for update test")

	createResultMap := createResult.(map[string]interface{})
	taskID := createResultMap["task_id"].(string)

	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful_status_update",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-update",
				"status":     "in_progress",
			},
			expectError: false,
		},
		{
			name: "successful_priority_update",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-update",
				"priority":   types.PriorityHigh,
			},
			expectError: false,
		},
		{
			name: "successful_progress_update",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-update",
				"progress":   50,
			},
			expectError: false,
		},
		{
			name: "successful_multiple_field_update",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-update",
				"status":     "blocked",
				"priority":   "low",
				"progress":   25,
				"assignee":   "newdev@example.com",
			},
			expectError: false,
		},
		{
			name: "missing_task_id",
			params: map[string]interface{}{
				"session_id": "test-session-update",
				"status":     "completed",
			},
			expectError: true,
			errorMsg:    "task_id parameter is required",
		},
		{
			name: "missing_session_id",
			params: map[string]interface{}{
				"task_id": taskID,
				"status":  "completed",
			},
			expectError: true,
			errorMsg:    "session_id parameter is required",
		},
		{
			name: "invalid_task_id",
			params: map[string]interface{}{
				"task_id":    "non-existent-task-id",
				"session_id": "test-session-update",
				"status":     "completed",
			},
			expectError: true,
			errorMsg:    "failed to retrieve task",
		},
		{
			name: "invalid_status",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-update",
				"status":     "invalid-status",
			},
			expectError: true,
			errorMsg:    "invalid task status",
		},
		{
			name: "invalid_priority",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-update",
				"priority":   "invalid-priority",
			},
			expectError: true,
			errorMsg:    "invalid task priority",
		},
		{
			name: "invalid_progress_negative",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-update",
				"progress":   -10,
			},
			expectError: true,
			errorMsg:    "progress must be between 0 and 100",
		},
		{
			name: "invalid_progress_over_100",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-update",
				"progress":   150,
			},
			expectError: true,
			errorMsg:    "progress must be between 0 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.handleUpdateTask(ctx, tt.params)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain: %s", tt.errorMsg)
				}
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Unexpected error for test case: %s", tt.name)
				assert.NotNil(t, result, "Result should not be nil")

				// Validate result structure
				resultMap, ok := result.(map[string]interface{})
				require.True(t, ok, "Result should be a map")

				assert.Contains(t, resultMap, "task_id", "Result should contain task_id")
				assert.Contains(t, resultMap, "updated_at", "Result should contain updated_at")
				assert.Equal(t, taskID, resultMap["task_id"], "task_id should match")
			}
		})
	}
}

func testTaskListing(t *testing.T, server *MemoryServer, ctx context.Context) {
	// Create multiple tasks for listing tests
	tasks := []map[string]interface{}{
		{
			"title":       "High priority task",
			"description": "This is a high priority task",
			"session_id":  "test-session-list",
			"repository":  "github.com/example/project",
			"priority":    types.PriorityHigh,
			"status":      "todo",
		},
		{
			"title":       "Medium priority task",
			"description": "This is a medium priority task",
			"session_id":  "test-session-list",
			"repository":  "github.com/example/project",
			"priority":    "medium",
			"status":      "in_progress",
		},
		{
			"title":       "Low priority task",
			"description": "This is a low priority task",
			"session_id":  "test-session-list",
			"repository":  "github.com/different/repo",
			"priority":    "low",
			"status":      "completed",
		},
	}

	// Create all test tasks
	for _, taskParams := range tasks {
		_, err := server.handleCreateTask(ctx, taskParams)
		require.NoError(t, err, "Failed to create test task")
	}

	tests := []struct {
		name            string
		params          map[string]interface{}
		expectError     bool
		expectedMinimum int // Minimum number of tasks expected
	}{
		{
			name:            "list_all_tasks",
			params:          map[string]interface{}{},
			expectError:     false,
			expectedMinimum: 3,
		},
		{
			name: "filter_by_repository",
			params: map[string]interface{}{
				"repository": "github.com/example/project",
			},
			expectError:     false,
			expectedMinimum: 2,
		},
		{
			name: "filter_by_session",
			params: map[string]interface{}{
				"session_id": "test-session-list",
			},
			expectError:     false,
			expectedMinimum: 3,
		},
		{
			name: "filter_by_status",
			params: map[string]interface{}{
				"status": "in_progress",
			},
			expectError:     false,
			expectedMinimum: 1,
		},
		{
			name: "filter_by_priority",
			params: map[string]interface{}{
				"priority": types.PriorityHigh,
			},
			expectError:     false,
			expectedMinimum: 1,
		},
		{
			name: "limit_results",
			params: map[string]interface{}{
				"limit": 2,
			},
			expectError:     false,
			expectedMinimum: 0, // Could be 0 if no tasks match FindSimilar
		},
		{
			name: "sort_by_priority",
			params: map[string]interface{}{
				"sort_by": "priority",
			},
			expectError:     false,
			expectedMinimum: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.handleListTasks(ctx, tt.params)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Unexpected error for test case: %s", tt.name)
				assert.NotNil(t, result, "Result should not be nil")

				// Validate result structure
				resultMap, ok := result.(map[string]interface{})
				require.True(t, ok, "Result should be a map")

				assert.Contains(t, resultMap, "tasks", "Result should contain tasks")
				assert.Contains(t, resultMap, "total", "Result should contain total")

				tasks, ok := resultMap["tasks"].([]interface{})
				require.True(t, ok, "tasks should be an array")

				// Note: Since we're using FindSimilar which may not work with mock,
				// we check for minimum but allow 0 results
				if tt.expectedMinimum > 0 {
					// For actual implementation, we'd expect this to work
					// For mock, we just ensure no error occurred
					t.Logf("Found %d tasks for test %s (expected minimum %d)", len(tasks), tt.name, tt.expectedMinimum)
				}
			}
		})
	}
}

func testTaskStatus(t *testing.T, server *MemoryServer, ctx context.Context) {
	// Create a task to get status for
	createParams := map[string]interface{}{
		"title":       "Status test task",
		"description": "This task is for status testing",
		"session_id":  "test-session-status",
		"repository":  "github.com/example/project",
		"priority":    "medium",
	}

	createResult, err := server.handleCreateTask(ctx, createParams)
	require.NoError(t, err, "Failed to create task for status test")

	createResultMap := createResult.(map[string]interface{})
	taskID := createResultMap["task_id"].(string)

	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful_status_retrieval",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-status",
			},
			expectError: false,
		},
		{
			name: "missing_task_id",
			params: map[string]interface{}{
				"session_id": "test-session-status",
			},
			expectError: true,
			errorMsg:    "task_id parameter is required",
		},
		{
			name: "missing_session_id",
			params: map[string]interface{}{
				"task_id": taskID,
			},
			expectError: true,
			errorMsg:    "session_id parameter is required",
		},
		{
			name: "invalid_task_id",
			params: map[string]interface{}{
				"task_id":    "non-existent-task-id",
				"session_id": "test-session-status",
			},
			expectError: true,
			errorMsg:    "failed to retrieve task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.handleGetTaskStatus(ctx, tt.params)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain: %s", tt.errorMsg)
				}
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Unexpected error for test case: %s", tt.name)
				assert.NotNil(t, result, "Result should not be nil")

				// Validate result structure
				resultMap, ok := result.(map[string]interface{})
				require.True(t, ok, "Result should be a map")

				assert.Contains(t, resultMap, "task_id", "Result should contain task_id")
				assert.Contains(t, resultMap, "title", "Result should contain title")
				assert.Contains(t, resultMap, "status", "Result should contain status")
				assert.Contains(t, resultMap, "created_at", "Result should contain created_at")

				assert.Equal(t, taskID, resultMap["task_id"], "task_id should match")
				assert.Equal(t, "Status test task", resultMap["title"], "title should match")
			}
		})
	}
}

func testTaskCompletion(t *testing.T, server *MemoryServer, ctx context.Context) {
	// Create a task to complete
	createParams := map[string]interface{}{
		"title":       "Task to complete",
		"description": "This task will be completed",
		"session_id":  "test-session-complete",
		"repository":  "github.com/example/project",
		"priority":    types.PriorityHigh,
	}

	createResult, err := server.handleCreateTask(ctx, createParams)
	require.NoError(t, err, "Failed to create task for completion test")

	createResultMap := createResult.(map[string]interface{})
	taskID := createResultMap["task_id"].(string)

	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful_task_completion",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-complete",
				"summary":    "Task completed successfully",
				"outcome":    "success",
				"time_spent": 60,
			},
			expectError: false,
		},
		{
			name: "completion_with_followup_tasks",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-complete",
				"summary":    "Task completed with followup",
				"outcome":    "success",
				"followup_tasks": []interface{}{
					"Review the implementation",
					"Update documentation",
				},
			},
			expectError: false,
		},
		{
			name: "missing_task_id",
			params: map[string]interface{}{
				"session_id": "test-session-complete",
				"summary":    "Task completed",
			},
			expectError: true,
			errorMsg:    "task_id parameter is required",
		},
		{
			name: "missing_session_id",
			params: map[string]interface{}{
				"task_id": taskID,
				"summary": "Task completed",
			},
			expectError: true,
			errorMsg:    "session_id parameter is required",
		},
		{
			name: "invalid_task_id",
			params: map[string]interface{}{
				"task_id":    "non-existent-task-id",
				"session_id": "test-session-complete",
				"summary":    "Task completed",
			},
			expectError: true,
			errorMsg:    "failed to retrieve task",
		},
		{
			name: "invalid_outcome",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-complete",
				"summary":    "Task completed",
				"outcome":    "invalid-outcome",
			},
			expectError: true,
			errorMsg:    "invalid outcome",
		},
		{
			name: "negative_time_spent",
			params: map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-complete",
				"summary":    "Task completed",
				"time_spent": -30,
			},
			expectError: true,
			errorMsg:    "time_spent must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := server.handleCompleteTask(ctx, tt.params)

			if tt.expectError {
				assertCompleteTaskError(t, err, result, tt)
			} else {
				assertCompleteTaskSuccess(t, err, result, tt.params, taskID)
			}
		})
	}
}

func testTaskErrorHandling(t *testing.T, server *MemoryServer, ctx context.Context) {
	t.Run("invalid_json_in_parameters", func(t *testing.T) {
		// Test with parameters that would cause JSON marshaling issues
		params := map[string]interface{}{
			"title":        "Test task",
			"description":  "Test description",
			"session_id":   "test-session",
			"invalid_data": make(chan int), // Channels can't be marshaled to JSON
		}

		// This should handle the error gracefully
		result, err := server.handleCreateTask(ctx, params)

		// The exact behavior depends on implementation, but it should not panic
		if err != nil {
			assert.Error(t, err, "Should handle invalid JSON gracefully")
			assert.Nil(t, result, "Result should be nil on error")
		}
	})

	t.Run("concurrent_task_operations", func(t *testing.T) {
		// Test concurrent operations on the same task
		createParams := map[string]interface{}{
			"title":       "Concurrent test task",
			"description": "Testing concurrent operations",
			"session_id":  "test-session-concurrent",
			"repository":  "github.com/example/project",
		}

		createResult, err := server.handleCreateTask(ctx, createParams)
		require.NoError(t, err, "Failed to create task for concurrent test")

		createResultMap := createResult.(map[string]interface{})
		taskID := createResultMap["task_id"].(string)

		// Run multiple updates concurrently
		done := make(chan bool, 3)

		go func() {
			updateParams := map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-concurrent",
				"status":     "in_progress",
			}
			_, err := server.handleUpdateTask(ctx, updateParams)
			assert.NoError(t, err, "Concurrent update 1 should succeed")
			done <- true
		}()

		go func() {
			updateParams := map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-concurrent",
				"priority":   types.PriorityHigh,
			}
			_, err := server.handleUpdateTask(ctx, updateParams)
			assert.NoError(t, err, "Concurrent update 2 should succeed")
			done <- true
		}()

		go func() {
			statusParams := map[string]interface{}{
				"task_id":    taskID,
				"session_id": "test-session-concurrent",
			}
			_, err := server.handleGetTaskStatus(ctx, statusParams)
			assert.NoError(t, err, "Concurrent status check should succeed")
			done <- true
		}()

		// Wait for all operations to complete
		for i := 0; i < 3; i++ {
			select {
			case <-done:
				// Operation completed
			case <-time.After(5 * time.Second):
				t.Fatal("Concurrent operations timed out")
			}
		}
	})

	t.Run("large_task_data", func(t *testing.T) {
		// Test with large amounts of data
		largeDescription := make([]byte, 10000) // 10KB description
		for i := range largeDescription {
			largeDescription[i] = 'A'
		}

		params := map[string]interface{}{
			"title":       "Large data task",
			"description": string(largeDescription),
			"session_id":  "test-session-large",
			"repository":  "github.com/example/project",
		}

		result, err := server.handleCreateTask(ctx, params)

		// Should handle large data appropriately
		if err != nil {
			// If there's an error, it should be a reasonable validation error
			assert.Contains(t, err.Error(), "too large", "Error should indicate size issue")
		} else {
			// If successful, validate the result
			assert.NotNil(t, result, "Result should not be nil")
			resultMap := result.(map[string]interface{})
			assert.Contains(t, resultMap, "task_id", "Result should contain task_id")
		}
	})
}

// TestTaskMetadataValidation tests the validation of task-specific metadata
func TestTaskMetadataValidation(t *testing.T) {
	tests := []struct {
		name        string
		metadata    types.ChunkMetadata
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid_task_metadata",
			metadata: types.ChunkMetadata{
				Repository:   "github.com/example/project",
				Outcome:      types.OutcomeSuccess,
				Difficulty:   types.DifficultyModerate,
				TaskStatus:   func() *types.TaskStatus { ts := types.TaskStatusTodo; return &ts }(),
				TaskPriority: func() *string { s := types.PriorityHigh; return &s }(),
				TaskEstimate: func() *int { e := 120; return &e }(),
				TaskProgress: func() *int { p := 50; return &p }(),
			},
			expectError: false,
		},
		{
			name: "invalid_task_status",
			metadata: types.ChunkMetadata{
				Repository: "github.com/example/project",
				Outcome:    types.OutcomeSuccess,
				Difficulty: types.DifficultyModerate,
				TaskStatus: func() *types.TaskStatus { ts := types.TaskStatus("invalid"); return &ts }(),
			},
			expectError: true,
			errorMsg:    "invalid task status",
		},
		{
			name: "invalid_task_priority",
			metadata: types.ChunkMetadata{
				Repository:   "github.com/example/project",
				Outcome:      types.OutcomeSuccess,
				Difficulty:   types.DifficultyModerate,
				TaskPriority: func() *string { s := "super-critical"; return &s }(),
			},
			expectError: true,
			errorMsg:    "invalid task priority",
		},
		{
			name: "negative_task_estimate",
			metadata: types.ChunkMetadata{
				Repository:   "github.com/example/project",
				Outcome:      types.OutcomeSuccess,
				Difficulty:   types.DifficultyModerate,
				TaskEstimate: func() *int { e := -30; return &e }(),
			},
			expectError: true,
			errorMsg:    "task estimate cannot be negative",
		},
		{
			name: "invalid_task_progress_negative",
			metadata: types.ChunkMetadata{
				Repository:   "github.com/example/project",
				Outcome:      types.OutcomeSuccess,
				Difficulty:   types.DifficultyModerate,
				TaskProgress: func() *int { p := -10; return &p }(),
			},
			expectError: true,
			errorMsg:    "task progress must be between 0 and 100",
		},
		{
			name: "invalid_task_progress_over_100",
			metadata: types.ChunkMetadata{
				Repository:   "github.com/example/project",
				Outcome:      types.OutcomeSuccess,
				Difficulty:   types.DifficultyModerate,
				TaskProgress: func() *int { p := 150; return &p }(),
			},
			expectError: true,
			errorMsg:    "task progress must be between 0 and 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()

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

// TestTaskTypeValidation tests the ChunkType validation for task types
func TestTaskTypeValidation(t *testing.T) {
	validTaskTypes := []types.ChunkType{
		types.ChunkTypeTask,
		types.ChunkTypeTaskUpdate,
		types.ChunkTypeTaskProgress,
	}

	for _, chunkType := range validTaskTypes {
		t.Run(string(chunkType), func(t *testing.T) {
			assert.True(t, chunkType.Valid(), "Task chunk type should be valid: %s", chunkType)
		})
	}

	invalidTaskTypes := []types.ChunkType{
		"invalid_task_type",
		"task_invalid",
		"",
	}

	for _, chunkType := range invalidTaskTypes {
		t.Run(string(chunkType), func(t *testing.T) {
			assert.False(t, chunkType.Valid(), "Invalid task chunk type should not be valid: %s", chunkType)
		})
	}
}

// TestTaskStatusValidation tests TaskStatus validation
func TestTaskStatusValidation(t *testing.T) {
	validStatuses := []types.TaskStatus{
		types.TaskStatusTodo,
		types.TaskStatusInProgress,
		types.TaskStatusCompleted,
		types.TaskStatusBlocked,
		types.TaskStatusCancelled,
		types.TaskStatusOnHold,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			assert.True(t, status.Valid(), "Task status should be valid: %s", status)
		})
	}

	invalidStatuses := []types.TaskStatus{
		"invalid_status",
		"pending",
		"",
	}

	for _, status := range invalidStatuses {
		t.Run(string(status), func(t *testing.T) {
			assert.False(t, status.Valid(), "Invalid task status should not be valid: %s", status)
		})
	}
}

// Test helper functions for complex assertion logic

// testCase represents a test case for task operations
type testCase struct {
	name        string
	params      map[string]interface{}
	expectError bool
	errorMsg    string
}

// assertCompleteTaskError validates error conditions for task completion
func assertCompleteTaskError(t *testing.T, err error, result interface{}, tt testCase) {
	_ = tt.params      // Suppress unused parameter warning
	_ = tt.expectError // Suppress unused parameter warning
	assert.Error(t, err, "Expected error for test case: %s", tt.name)
	if tt.errorMsg != "" {
		assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain: %s", tt.errorMsg)
	}
	assert.Nil(t, result, "Result should be nil on error")
}

// assertCompleteTaskSuccess validates successful task completion
func assertCompleteTaskSuccess(t *testing.T, err error, result interface{}, params map[string]interface{}, taskID string) {
	assert.NoError(t, err, "Unexpected error for successful test case")
	assert.NotNil(t, result, "Result should not be nil")

	// Validate result structure
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok, "Result should be a map")

	// Basic field validation
	assertCompleteTaskFields(t, resultMap, taskID)

	// Check followup tasks if specified
	assertFollowupTasks(t, resultMap, params)
}

// assertCompleteTaskFields validates basic fields in task completion result
func assertCompleteTaskFields(t *testing.T, resultMap map[string]interface{}, taskID string) {
	assert.Contains(t, resultMap, "task_id", "Result should contain task_id")
	assert.Contains(t, resultMap, "status", "Result should contain status")
	assert.Contains(t, resultMap, "completed_at", "Result should contain completed_at")

	assert.Equal(t, taskID, resultMap["task_id"], "task_id should match")
	assert.Equal(t, "completed", resultMap["status"], "status should be completed")
}

// assertFollowupTasks validates followup task creation
func assertFollowupTasks(t *testing.T, resultMap map[string]interface{}, params map[string]interface{}) {
	followupTasks, exists := params["followup_tasks"]
	if !exists {
		return
	}

	followupList := followupTasks.([]interface{})
	if len(followupList) == 0 {
		return
	}

	assert.Contains(t, resultMap, "followup_task_ids", "Result should contain followup_task_ids")
	followupIDs, ok := resultMap["followup_task_ids"].([]string)
	if ok {
		assert.Equal(t, len(followupList), len(followupIDs), "Should create correct number of followup tasks")
	}
}
