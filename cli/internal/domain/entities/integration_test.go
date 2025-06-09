package entities

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTaskFullWorkflow demonstrates a complete task lifecycle
func TestTaskFullWorkflow(t *testing.T) {
	// Create a new task
	task, err := NewTask("Implement user authentication", "github.com/user/project")
	require.NoError(t, err)
	require.NotNil(t, task)

	// Verify initial state
	assert.Equal(t, StatusPending, task.Status)
	assert.Equal(t, PriorityMedium, task.Priority)
	assert.False(t, task.AISuggested)
	assert.Empty(t, task.Tags)

	// Add metadata
	task.AddTag("security")
	task.AddTag("backend")
	_ = task.SetPriority(PriorityHigh)
	_ = task.SetEstimation(120) // 2 hours

	// Start working on task
	err = task.Start()
	require.NoError(t, err)
	assert.Equal(t, StatusInProgress, task.Status)

	// Simulate work time
	time.Sleep(10 * time.Millisecond)

	// Complete the task
	err = task.Complete()
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, task.Status)
	assert.NotNil(t, task.CompletedAt)

	// Record actual time spent
	_ = task.SetActualTime(90) // Actually took 1.5 hours

	// Verify final state
	assert.True(t, task.HasTag("security"))
	assert.True(t, task.HasTag("backend"))
	assert.Equal(t, PriorityHigh, task.Priority)
	assert.Equal(t, 120, task.EstimatedMins)
	assert.Equal(t, 90, task.ActualMins)

	// Test serialization
	jsonData, err := json.Marshal(task)
	require.NoError(t, err)

	var deserializedTask Task
	err = json.Unmarshal(jsonData, &deserializedTask)
	require.NoError(t, err)

	// Verify deserialized task maintains all data
	assert.Equal(t, task.ID, deserializedTask.ID)
	assert.Equal(t, task.Status, deserializedTask.Status)
	assert.Equal(t, task.Priority, deserializedTask.Priority)
	assert.Equal(t, task.Tags, deserializedTask.Tags)
	assert.Equal(t, task.EstimatedMins, deserializedTask.EstimatedMins)
	assert.Equal(t, task.ActualMins, deserializedTask.ActualMins)
}

// TestTaskWithAIGeneration demonstrates AI-suggested task creation
func TestTaskWithAIGeneration(t *testing.T) {
	options := TaskOptions{
		Priority:      PriorityLow,
		SessionID:     "ai-session-123",
		EstimatedMins: 30,
		Tags:          []string{"ai-generated", "suggestion"},
		AISuggested:   true,
	}

	task, err := NewTaskWithOptions("Write unit tests for auth module", "github.com/user/project", &options)
	require.NoError(t, err)

	// Verify AI-specific properties
	assert.True(t, task.AISuggested)
	assert.Equal(t, "ai-session-123", task.SessionID)
	assert.Contains(t, task.Tags, "ai-generated")
	assert.Contains(t, task.Tags, "suggestion")

	// Verify it behaves like any other task
	err = task.Start()
	require.NoError(t, err)

	err = task.Complete()
	require.NoError(t, err)

	assert.Equal(t, StatusCompleted, task.Status)
}

// TestSubTaskHierarchy demonstrates parent-child task relationships
func TestSubTaskHierarchy(t *testing.T) {
	// Create parent task
	parentTask, err := NewTask("Implement user management system", "github.com/user/project")
	require.NoError(t, err)

	// Create sub-tasks
	subTask1Options := TaskOptions{
		ParentTaskID: parentTask.ID,
		Priority:     PriorityHigh,
		Tags:         []string{"database"},
	}
	subTask1, err := NewTaskWithOptions("Design user database schema", "github.com/user/project", &subTask1Options)
	require.NoError(t, err)

	subTask2Options := TaskOptions{
		ParentTaskID: parentTask.ID,
		Priority:     PriorityMedium,
		Tags:         []string{"api"},
	}
	subTask2, err := NewTaskWithOptions("Implement user API endpoints", "github.com/user/project", &subTask2Options)
	require.NoError(t, err)

	// Verify relationships
	assert.Equal(t, parentTask.ID, subTask1.ParentTaskID)
	assert.Equal(t, parentTask.ID, subTask2.ParentTaskID)
	assert.Empty(t, parentTask.ParentTaskID) // Parent has no parent

	// Verify all tasks are valid
	assert.NoError(t, parentTask.Validate())
	assert.NoError(t, subTask1.Validate())
	assert.NoError(t, subTask2.Validate())
}

// TestTaskErrorHandling demonstrates robust error handling
func TestTaskErrorHandling(t *testing.T) {
	// Test creation with invalid data
	_, err := NewTask("", "github.com/user/project")
	assert.Error(t, err)

	_, err = NewTask("Valid content", "")
	assert.Error(t, err)

	// Test invalid options
	invalidOptions := TaskOptions{
		ParentTaskID: "invalid-uuid",
	}
	_, err = NewTaskWithOptions("Valid content", "github.com/user/project", &invalidOptions)
	assert.Error(t, err)

	// Test invalid state transitions
	task, err := NewTask("Test task", "github.com/user/project")
	require.NoError(t, err)

	_ = task.Complete() // Skip in_progress
	err = task.Start()  // Cannot start completed task
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidStatusTransition, err)

	// Test validation after invalid updates
	_ = task.Reset()
	originalContent := task.Content

	err = task.UpdateContent("") // Invalid content
	assert.Error(t, err)
	assert.Equal(t, originalContent, task.Content) // Should remain unchanged
}

// TestTaskMCPCompatibility demonstrates MCP tools format compatibility
func TestTaskMCPCompatibility(t *testing.T) {
	task, err := NewTask("Test MCP compatibility", "github.com/user/project")
	require.NoError(t, err)

	task.AddTag("mcp")
	_ = task.SetPriority(PriorityHigh)

	// Serialize to format expected by MCP tools
	jsonData, err := json.Marshal(task)
	require.NoError(t, err)

	// Verify JSON contains expected MCP fields
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)

	// Check required MCP fields are present
	expectedFields := []string{"id", "content", "status", "priority", "repository", "created_at", "updated_at"}
	for _, field := range expectedFields {
		assert.Contains(t, jsonMap, field, "MCP compatibility requires field: %s", field)
	}

	// Verify field types match MCP expectations
	assert.IsType(t, "", jsonMap["id"])
	assert.IsType(t, "", jsonMap["content"])
	assert.IsType(t, "", jsonMap["status"])
	assert.IsType(t, "", jsonMap["priority"])
	assert.IsType(t, "", jsonMap["repository"])
}
