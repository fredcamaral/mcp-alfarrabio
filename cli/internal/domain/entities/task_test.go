package entities

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		repository string
		wantErr    bool
		errType    error
	}{
		{
			name:       "valid task creation",
			content:    "Test task content",
			repository: "github.com/user/repo",
			wantErr:    false,
		},
		{
			name:       "empty content fails",
			content:    "",
			repository: "github.com/user/repo",
			wantErr:    true,
		},
		{
			name:       "whitespace only content fails",
			content:    "   ",
			repository: "github.com/user/repo",
			wantErr:    true,
		},
		{
			name:       "empty repository fails",
			content:    "Valid content",
			repository: "",
			wantErr:    true,
		},
		{
			name:       "content too long fails",
			content:    string(make([]byte, 1001)),
			repository: "github.com/user/repo",
			wantErr:    true,
		},
		{
			name:       "content at limit passes",
			content:    string(make([]byte, 1000)),
			repository: "github.com/user/repo",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := NewTask(tt.content, tt.repository)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, task)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, task)

			// Verify default values
			assert.NotEmpty(t, task.ID)
			assert.True(t, isValidUUID(task.ID))
			assert.Equal(t, tt.content, task.Content)
			assert.Equal(t, StatusPending, task.Status)
			assert.Equal(t, PriorityMedium, task.Priority)
			assert.Equal(t, tt.repository, task.Repository)
			assert.False(t, task.AISuggested)
			assert.WithinDuration(t, time.Now(), task.CreatedAt, time.Second)
			assert.WithinDuration(t, time.Now(), task.UpdatedAt, time.Second)
			assert.Nil(t, task.CompletedAt)
			assert.Empty(t, task.Tags)
		})
	}
}

func TestNewTaskWithOptions(t *testing.T) {
	options := TaskOptions{
		Priority:      PriorityHigh,
		SessionID:     "session123",
		EstimatedMins: 60,
		Tags:          []string{"urgent", "feature"},
		ParentTaskID:  uuid.New().String(),
		AISuggested:   true,
	}

	task, err := NewTaskWithOptions("Test content", "github.com/user/repo", &options)

	require.NoError(t, err)
	require.NotNil(t, task)

	assert.Equal(t, PriorityHigh, task.Priority)
	assert.Equal(t, "session123", task.SessionID)
	assert.Equal(t, 60, task.EstimatedMins)
	assert.Equal(t, []string{"urgent", "feature"}, task.Tags)
	assert.Equal(t, options.ParentTaskID, task.ParentTaskID)
	assert.True(t, task.AISuggested)
}

func TestTaskStatusTransitions(t *testing.T) {
	task, err := NewTask("Test task", "github.com/user/repo")
	require.NoError(t, err)

	// Test start transition
	err = task.Start()
	assert.NoError(t, err)
	assert.Equal(t, StatusInProgress, task.Status)

	// Test complete transition
	err = task.Complete()
	assert.NoError(t, err)
	assert.Equal(t, StatusCompleted, task.Status)
	assert.NotNil(t, task.CompletedAt)
	assert.WithinDuration(t, time.Now(), *task.CompletedAt, time.Second)

	// Test invalid transition from completed to start
	err = task.Start()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidStatusTransition, err)

	// Test reset
	err = task.Reset()
	assert.NoError(t, err)
	assert.Equal(t, StatusPending, task.Status)
	assert.Nil(t, task.CompletedAt)
}

func TestTaskCancel(t *testing.T) {
	task, err := NewTask("Test task", "github.com/user/repo")
	require.NoError(t, err)

	// Can cancel from pending
	err = task.Cancel()
	assert.NoError(t, err)
	assert.Equal(t, StatusCancelled, task.Status)

	// Cannot start after cancel
	err = task.Start()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidStatusTransition, err)

	// Test cancel from in progress
	_ = task.Reset()
	_ = task.Start()
	err = task.Cancel()
	assert.NoError(t, err)
	assert.Equal(t, StatusCancelled, task.Status)

	// Cannot cancel from completed
	_ = task.Reset()
	_ = task.Start()
	_ = task.Complete()
	err = task.Cancel()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidStatusTransition, err)
}

func TestUpdateContent(t *testing.T) {
	task, err := NewTask("Original content", "github.com/user/repo")
	require.NoError(t, err)

	// Valid update
	err = task.UpdateContent("Updated content")
	assert.NoError(t, err)
	assert.Equal(t, "Updated content", task.Content)

	// Empty content fails
	err = task.UpdateContent("")
	assert.Error(t, err)
	assert.Equal(t, "Updated content", task.Content) // Should remain unchanged

	// Too long content fails
	err = task.UpdateContent(string(make([]byte, 1001)))
	assert.Error(t, err)
	assert.Equal(t, "Updated content", task.Content) // Should remain unchanged
}

func TestSetPriority(t *testing.T) {
	task, err := NewTask("Test task", "github.com/user/repo")
	require.NoError(t, err)

	err = task.SetPriority(PriorityHigh)
	assert.NoError(t, err)
	assert.Equal(t, PriorityHigh, task.Priority)

	err = task.SetPriority(PriorityLow)
	assert.NoError(t, err)
	assert.Equal(t, PriorityLow, task.Priority)
}

func TestTaskTags(t *testing.T) {
	task, err := NewTask("Test task", "github.com/user/repo")
	require.NoError(t, err)

	// Add tags
	task.AddTag("feature")
	task.AddTag("urgent")
	assert.Equal(t, []string{"feature", "urgent"}, task.Tags)

	// Duplicate tag not added
	task.AddTag("feature")
	assert.Equal(t, []string{"feature", "urgent"}, task.Tags)

	// Empty tag not added
	task.AddTag("")
	task.AddTag("   ")
	assert.Equal(t, []string{"feature", "urgent"}, task.Tags)

	// Has tag check
	assert.True(t, task.HasTag("feature"))
	assert.True(t, task.HasTag("urgent"))
	assert.False(t, task.HasTag("missing"))

	// Remove tag
	task.RemoveTag("feature")
	assert.Equal(t, []string{"urgent"}, task.Tags)
	assert.False(t, task.HasTag("feature"))

	// Remove non-existent tag
	task.RemoveTag("missing")
	assert.Equal(t, []string{"urgent"}, task.Tags)
}

func TestSetEstimation(t *testing.T) {
	task, err := NewTask("Test task", "github.com/user/repo")
	require.NoError(t, err)

	// Valid estimation
	err = task.SetEstimation(60)
	assert.NoError(t, err)
	assert.Equal(t, 60, task.EstimatedMins)

	// Zero estimation valid
	err = task.SetEstimation(0)
	assert.NoError(t, err)
	assert.Equal(t, 0, task.EstimatedMins)

	// Negative estimation invalid
	err = task.SetEstimation(-1)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidEstimation, err)
	assert.Equal(t, 0, task.EstimatedMins) // Should remain unchanged
}

func TestSetActualTime(t *testing.T) {
	task, err := NewTask("Test task", "github.com/user/repo")
	require.NoError(t, err)

	// Valid actual time
	err = task.SetActualTime(45)
	assert.NoError(t, err)
	assert.Equal(t, 45, task.ActualMins)

	// Zero actual time valid
	err = task.SetActualTime(0)
	assert.NoError(t, err)
	assert.Equal(t, 0, task.ActualMins)

	// Negative actual time invalid
	err = task.SetActualTime(-1)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidActualTime, err)
	assert.Equal(t, 0, task.ActualMins) // Should remain unchanged
}

func TestIsValidStatus(t *testing.T) {
	validStatuses := []string{"pending", "in_progress", "completed", "cancelled"}
	for _, status := range validStatuses {
		assert.True(t, IsValidStatus(status), "Status %s should be valid", status)
	}

	invalidStatuses := []string{"", "invalid", "PENDING", "done", "active"}
	for _, status := range invalidStatuses {
		assert.False(t, IsValidStatus(status), "Status %s should be invalid", status)
	}
}

func TestIsValidPriority(t *testing.T) {
	validPriorities := []string{"low", "medium", "high"}
	for _, priority := range validPriorities {
		assert.True(t, IsValidPriority(priority), "Priority %s should be valid", priority)
	}

	invalidPriorities := []string{"", "invalid", "LOW", "critical", "normal"}
	for _, priority := range invalidPriorities {
		assert.False(t, IsValidPriority(priority), "Priority %s should be invalid", priority)
	}
}

func TestGetDuration(t *testing.T) {
	task, err := NewTask("Test task", "github.com/user/repo")
	require.NoError(t, err)

	// Before completion
	duration := task.GetDuration()
	assert.True(t, duration >= 0)
	assert.True(t, duration < time.Second) // Should be very short for new task

	// After completion
	time.Sleep(10 * time.Millisecond)
	_ = task.Start()
	_ = task.Complete()

	duration = task.GetDuration()
	assert.True(t, duration >= 10*time.Millisecond)
	assert.True(t, duration < time.Second)
}

func TestIsOverdue(t *testing.T) {
	task, err := NewTask("Test task", "github.com/user/repo")
	require.NoError(t, err)

	// No estimation means not overdue
	assert.False(t, task.IsOverdue())

	// Set very short estimation and wait
	_ = task.SetEstimation(0) // 0 minutes
	time.Sleep(10 * time.Millisecond)
	assert.False(t, task.IsOverdue()) // 0 estimation means not overdue

	// Set estimation in the past
	task.CreatedAt = time.Now().Add(-2 * time.Hour)
	_ = task.SetEstimation(60) // 1 hour
	assert.True(t, task.IsOverdue())

	// Completed tasks are not overdue
	_ = task.Complete()
	assert.False(t, task.IsOverdue())

	// Cancelled tasks are not overdue
	_ = task.Reset()
	_ = task.Cancel()
	assert.False(t, task.IsOverdue())
}

func TestTaskValidation(t *testing.T) {
	// Test invalid UUID for ParentTaskID
	task := &Task{
		ID:           uuid.New().String(),
		Content:      "Test content",
		Status:       StatusPending,
		Priority:     PriorityMedium,
		Repository:   "github.com/user/repo",
		ParentTaskID: "invalid-uuid",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := task.Validate()
	assert.Error(t, err)

	// Test invalid status
	task.ParentTaskID = ""
	task.Status = Status("invalid")
	err = task.Validate()
	assert.Error(t, err)

	// Test invalid priority
	task.Status = StatusPending
	task.Priority = Priority("invalid")
	err = task.Validate()
	assert.Error(t, err)
}

func TestTaskJSONSerialization(t *testing.T) {
	originalTask, err := NewTask("Test task", "github.com/user/repo")
	require.NoError(t, err)

	originalTask.AddTag("test")
	_ = originalTask.SetEstimation(60)
	_ = originalTask.Start()

	// Marshal to JSON
	jsonData, err := json.Marshal(originalTask)
	require.NoError(t, err)

	// Unmarshal from JSON
	var deserializedTask Task
	err = json.Unmarshal(jsonData, &deserializedTask)
	require.NoError(t, err)

	// Verify all fields are preserved
	assert.Equal(t, originalTask.ID, deserializedTask.ID)
	assert.Equal(t, originalTask.Content, deserializedTask.Content)
	assert.Equal(t, originalTask.Status, deserializedTask.Status)
	assert.Equal(t, originalTask.Priority, deserializedTask.Priority)
	assert.Equal(t, originalTask.Repository, deserializedTask.Repository)
	assert.Equal(t, originalTask.EstimatedMins, deserializedTask.EstimatedMins)
	assert.Equal(t, originalTask.Tags, deserializedTask.Tags)
	assert.Equal(t, originalTask.AISuggested, deserializedTask.AISuggested)

	// Verify timestamps are preserved (within 1ms precision)
	assert.WithinDuration(t, originalTask.CreatedAt, deserializedTask.CreatedAt, time.Millisecond)
	assert.WithinDuration(t, originalTask.UpdatedAt, deserializedTask.UpdatedAt, time.Millisecond)

	// Verify validation still works after deserialization
	err = deserializedTask.Validate()
	assert.NoError(t, err)
}

func TestTaskWithCompletedAtSerialization(t *testing.T) {
	task, err := NewTask("Test task", "github.com/user/repo")
	require.NoError(t, err)

	_ = task.Start()
	_ = task.Complete()

	// Marshal to JSON
	jsonData, err := json.Marshal(task)
	require.NoError(t, err)

	// Verify completed_at is included
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonMap)
	require.NoError(t, err)
	assert.Contains(t, jsonMap, "completed_at")
	assert.NotNil(t, jsonMap["completed_at"])

	// Unmarshal and verify
	var deserializedTask Task
	err = json.Unmarshal(jsonData, &deserializedTask)
	require.NoError(t, err)

	require.NotNil(t, deserializedTask.CompletedAt)
	assert.WithinDuration(t, *task.CompletedAt, *deserializedTask.CompletedAt, time.Millisecond)
}

// Helper function to validate UUID format
func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

// Benchmark tests for performance validation
func BenchmarkNewTask(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := NewTask("Benchmark task", "github.com/user/repo")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTaskValidation(b *testing.B) {
	task, err := NewTask("Benchmark task", "github.com/user/repo")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := task.Validate()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTaskJSONMarshal(b *testing.B) {
	task, err := NewTask("Benchmark task", "github.com/user/repo")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(task)
		if err != nil {
			b.Fatal(err)
		}
	}
}
