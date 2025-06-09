package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// Test Table Formatter Creation
func TestNewTableFormatter(t *testing.T) {
	output := &bytes.Buffer{}
	formatter := NewTableFormatter(output)

	assert.NotNil(t, formatter)
	assert.Implements(t, (*OutputFormatter)(nil), formatter)
}

func TestNewJSONFormatter(t *testing.T) {
	output := &bytes.Buffer{}
	formatter := NewJSONFormatter(output, true)

	assert.NotNil(t, formatter)
	assert.Implements(t, (*OutputFormatter)(nil), formatter)
}

func TestNewPlainFormatter(t *testing.T) {
	output := &bytes.Buffer{}
	formatter := NewPlainFormatter(output)

	assert.NotNil(t, formatter)
	assert.Implements(t, (*OutputFormatter)(nil), formatter)
}

// Test Single Task Formatting
func TestFormatSingleTaskTable(t *testing.T) {
	now := time.Now()
	task := &entities.Task{
		ID:         "task-123456789", // Long ID to test truncation
		Content:    "Test task content",
		Status:     entities.StatusPending,
		Priority:   entities.PriorityHigh,
		Tags:       []string{"urgent", "bug"},
		Repository: "test-repo",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	output := &bytes.Buffer{}
	formatter := NewTableFormatter(output)

	err := formatter.FormatTask(task)
	require.NoError(t, err)

	result := output.String()

	// Check that table contains key information
	assert.Contains(t, result, "task-123") // Truncated ID
	assert.Contains(t, result, "Test task content")
	assert.Contains(t, result, "pending")
	assert.Contains(t, result, "high")
	assert.Contains(t, result, "urgent, bug")
	assert.Contains(t, result, "test-repo")
}

func TestFormatSingleTaskJSON(t *testing.T) {
	now := time.Now()
	task := &entities.Task{
		ID:            "task-123",
		Content:       "Test task",
		Status:        entities.StatusInProgress,
		Priority:      entities.PriorityMedium,
		Tags:          []string{"test"},
		Repository:    "test-repo",
		EstimatedMins: 60,
		ActualMins:    45,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	output := &bytes.Buffer{}
	formatter := NewJSONFormatter(output, true)

	err := formatter.FormatTask(task)
	require.NoError(t, err)

	result := output.String()

	// Parse JSON to verify structure
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "task-123", parsed["id"])
	assert.Equal(t, "Test task", parsed["content"])
	assert.Equal(t, "in_progress", parsed["status"])
	assert.Equal(t, "medium", parsed["priority"])
	assert.Equal(t, "test-repo", parsed["repository"])
	assert.Equal(t, float64(60), parsed["estimated_mins"])
	assert.Equal(t, float64(45), parsed["actual_mins"])

	// Check tags
	tags, ok := parsed["tags"].([]interface{})
	require.True(t, ok)
	assert.Len(t, tags, 1)
	assert.Equal(t, "test", tags[0])
}

func TestFormatSingleTaskPlain(t *testing.T) {
	now := time.Now()
	task := &entities.Task{
		ID:          "task-123",
		Content:     "Test task",
		Status:      entities.StatusCompleted,
		Priority:    entities.PriorityLow,
		Tags:        []string{"feature"},
		Repository:  "test-repo",
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: &now,
	}

	output := &bytes.Buffer{}
	formatter := NewPlainFormatter(output)

	err := formatter.FormatTask(task)
	require.NoError(t, err)

	result := output.String()

	// Plain format should include key information in readable format
	assert.Contains(t, result, "task-123")
	assert.Contains(t, result, "Test task")
	assert.Contains(t, result, "completed")
	assert.Contains(t, result, "low")
	assert.Contains(t, result, "feature")
	assert.Contains(t, result, "test-repo")
}

// Test Task List Formatting
func TestFormatTaskListTable(t *testing.T) {
	now := time.Now()
	tasks := []*entities.Task{
		{
			ID:         "task-1",
			Content:    "First task",
			Status:     entities.StatusPending,
			Priority:   entities.PriorityHigh,
			Tags:       []string{"urgent"},
			Repository: "repo1",
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			ID:         "task-2",
			Content:    "Second task",
			Status:     entities.StatusInProgress,
			Priority:   entities.PriorityMedium,
			Tags:       []string{"feature"},
			Repository: "repo1",
			CreatedAt:  now.Add(-1 * time.Hour),
			UpdatedAt:  now,
		},
	}

	output := &bytes.Buffer{}
	formatter := NewTableFormatter(output)

	err := formatter.FormatTaskList(tasks)
	require.NoError(t, err)

	result := output.String()

	// Check table structure and content
	assert.Contains(t, result, "ID")
	assert.Contains(t, result, "CONTENT")
	assert.Contains(t, result, "STATUS")
	assert.Contains(t, result, "PRIORITY")
	assert.Contains(t, result, "TAGS")

	// Check task content
	assert.Contains(t, result, "task-1")
	assert.Contains(t, result, "First task")
	assert.Contains(t, result, "pending")
	assert.Contains(t, result, "high")
	assert.Contains(t, result, "urgent")

	assert.Contains(t, result, "task-2")
	assert.Contains(t, result, "Second task")
	assert.Contains(t, result, "in_progress")
	assert.Contains(t, result, "medium")
	assert.Contains(t, result, "feature")
}

func TestFormatTaskListJSON(t *testing.T) {
	now := time.Now()
	tasks := []*entities.Task{
		{
			ID:        "task-1",
			Content:   "First task",
			Status:    entities.StatusPending,
			Priority:  entities.PriorityHigh,
			Tags:      []string{"urgent"},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "task-2",
			Content:     "Second task",
			Status:      entities.StatusCompleted,
			Priority:    entities.PriorityMedium,
			Tags:        []string{"feature"},
			CreatedAt:   now.Add(-1 * time.Hour),
			UpdatedAt:   now,
			CompletedAt: &now,
		},
	}

	output := &bytes.Buffer{}
	formatter := NewJSONFormatter(output, true)

	err := formatter.FormatTaskList(tasks)
	require.NoError(t, err)

	result := output.String()

	// Parse JSON to verify structure
	var parsed struct {
		Tasks []map[string]interface{} `json:"tasks"`
		Count int                      `json:"count"`
	}

	err = json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)

	assert.Equal(t, 2, parsed.Count)
	assert.Len(t, parsed.Tasks, 2)

	// Check first task
	task1 := parsed.Tasks[0]
	assert.Equal(t, "task-1", task1["id"])
	assert.Equal(t, "First task", task1["content"])
	assert.Equal(t, "pending", task1["status"])
	assert.Equal(t, "high", task1["priority"])

	// Check second task
	task2 := parsed.Tasks[1]
	assert.Equal(t, "task-2", task2["id"])
	assert.Equal(t, "completed", task2["status"])
	assert.NotNil(t, task2["completed_at"])
}

func TestFormatTaskListEmpty(t *testing.T) {
	tests := []struct {
		name      string
		formatter OutputFormatter
	}{
		{"table", NewTableFormatter(&bytes.Buffer{})},
		{"json", NewJSONFormatter(&bytes.Buffer{}, true)},
		{"plain", NewPlainFormatter(&bytes.Buffer{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.formatter.FormatTaskList([]*entities.Task{})
			assert.NoError(t, err)
		})
	}
}

// Test Stats Formatting
func TestFormatStatsTable(t *testing.T) {
	now := time.Now()
	stats := ports.RepositoryStats{
		Repository:      "test-repo",
		TotalTasks:      25,
		PendingTasks:    10,
		InProgressTasks: 5,
		CompletedTasks:  8,
		CancelledTasks:  2,
		LastActivity:    now.Format(time.RFC3339),
	}

	output := &bytes.Buffer{}
	formatter := NewTableFormatter(output)

	err := formatter.FormatStats(&stats)
	require.NoError(t, err)

	result := output.String()

	// Check stats content
	assert.Contains(t, result, "test-repo")
	assert.Contains(t, result, "25")
	assert.Contains(t, result, "10")
	assert.Contains(t, result, "5")
	assert.Contains(t, result, "8")
	assert.Contains(t, result, "2")
}

func TestFormatStatsJSON(t *testing.T) {
	now := time.Now()
	stats := ports.RepositoryStats{
		Repository:      "test-repo",
		TotalTasks:      15,
		PendingTasks:    5,
		InProgressTasks: 3,
		CompletedTasks:  6,
		CancelledTasks:  1,
		LastActivity:    now.Format(time.RFC3339),
	}

	output := &bytes.Buffer{}
	formatter := NewJSONFormatter(output, true)

	err := formatter.FormatStats(&stats)
	require.NoError(t, err)

	result := output.String()

	// Parse JSON to verify structure
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "test-repo", parsed["Repository"])
	assert.Equal(t, float64(15), parsed["TotalTasks"])
	assert.Equal(t, float64(5), parsed["PendingTasks"])
	assert.Equal(t, float64(3), parsed["InProgressTasks"])
	assert.Equal(t, float64(6), parsed["CompletedTasks"])
	assert.Equal(t, float64(1), parsed["CancelledTasks"])
	assert.NotNil(t, parsed["LastActivity"])
}

// Test Error Formatting
func TestFormatError(t *testing.T) {
	testErr := entities.ErrInvalidStatusTransition

	tests := []struct {
		name      string
		formatter OutputFormatter
	}{
		{"table", NewTableFormatter(&bytes.Buffer{})},
		{"json", NewJSONFormatter(&bytes.Buffer{}, true)},
		{"plain", NewPlainFormatter(&bytes.Buffer{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.formatter.FormatError(testErr)
			assert.NoError(t, err)
		})
	}
}

// Test Long Content Handling
func TestFormatLongContent(t *testing.T) {
	longContent := strings.Repeat("Very long task description ", 20)
	task := &entities.Task{
		ID:        "task-long",
		Content:   longContent,
		Status:    entities.StatusPending,
		Priority:  entities.PriorityMedium,
		Tags:      []string{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tests := []struct {
		name      string
		formatter OutputFormatter
	}{
		{"table", NewTableFormatter(&bytes.Buffer{})},
		{"json", NewJSONFormatter(&bytes.Buffer{}, true)},
		{"plain", NewPlainFormatter(&bytes.Buffer{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.formatter.FormatTask(task)
			assert.NoError(t, err)
		})
	}
}

// Test Task with All Fields
func TestFormatTaskWithAllFields(t *testing.T) {
	now := time.Now()
	task := &entities.Task{
		ID:            "task-complete",
		Content:       "Complete task example",
		Status:        entities.StatusCompleted,
		Priority:      entities.PriorityHigh,
		Tags:          []string{"feature", "urgent", "backend"},
		Repository:    "main-repo",
		EstimatedMins: 120,
		ActualMins:    90,
		SessionID:     "session-456",
		ParentTaskID:  "parent-789",
		AISuggested:   true,
		CreatedAt:     now.Add(-3 * time.Hour),
		UpdatedAt:     now.Add(-1 * time.Hour),
		// StartedAt field not available in current Task entity
		CompletedAt: &now,
	}

	tests := []struct {
		name      string
		formatter OutputFormatter
	}{
		{"table", NewTableFormatter(&bytes.Buffer{})},
		{"json", NewJSONFormatter(&bytes.Buffer{}, true)},
		{"plain", NewPlainFormatter(&bytes.Buffer{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.formatter.FormatTask(task)
			assert.NoError(t, err)
		})
	}
}

// Test Interface Compliance
func TestFormatterInterfaceCompliance(t *testing.T) {
	var _ = NewTableFormatter(&bytes.Buffer{})
	var _ = NewJSONFormatter(&bytes.Buffer{}, true)
	var _ = NewPlainFormatter(&bytes.Buffer{})
}

// Test Formatter with Different Writers
func TestFormatterWithDifferentWriters(t *testing.T) {
	task := &entities.Task{
		ID:        "test-task",
		Content:   "Test task",
		Status:    entities.StatusPending,
		Priority:  entities.PriorityMedium,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test with different io.Writer implementations
	writers := []io.Writer{
		&bytes.Buffer{},
		&strings.Builder{},
	}

	for i, writer := range writers {
		t.Run(fmt.Sprintf("writer_%d", i), func(t *testing.T) {
			formatter := NewTableFormatter(writer)
			err := formatter.FormatTask(task)
			assert.NoError(t, err)
		})
	}
}

// Benchmark formatters
func BenchmarkTableFormatter(b *testing.B) {
	task := &entities.Task{
		ID:        "benchmark-task",
		Content:   "Benchmark task content",
		Status:    entities.StatusPending,
		Priority:  entities.PriorityMedium,
		Tags:      []string{"benchmark", "test"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	output := &bytes.Buffer{}
	formatter := NewTableFormatter(output)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		output.Reset()
		_ = formatter.FormatTask(task)
	}
}

func BenchmarkJSONFormatter(b *testing.B) {
	task := &entities.Task{
		ID:        "benchmark-task",
		Content:   "Benchmark task content",
		Status:    entities.StatusPending,
		Priority:  entities.PriorityMedium,
		Tags:      []string{"benchmark", "test"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	output := &bytes.Buffer{}
	formatter := NewJSONFormatter(output, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		output.Reset()
		_ = formatter.FormatTask(task)
	}
}
