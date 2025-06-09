package main

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory-cli/internal/adapters/secondary/repository"
	"lerian-mcp-memory-cli/internal/adapters/secondary/storage"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// TestFullTaskWorkflow tests the complete task management workflow
func TestFullTaskWorkflow(t *testing.T) {
	// Setup components
	fileStorage, err := storage.NewFileStorageWithPath(t.TempDir())
	require.NoError(t, err)

	gitDetector := repository.NewGitDetector()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskService := services.NewTaskService(fileStorage, gitDetector, logger)

	ctx := context.Background()

	// Test 1: Create task
	task, err := taskService.CreateTask(ctx, "Implement user authentication",
		services.WithPriority(entities.PriorityHigh),
		services.WithTags("security", "backend"),
		services.WithEstimatedTime(120),
	)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "Implement user authentication", task.Content)
	assert.Equal(t, entities.PriorityHigh, task.Priority)
	assert.True(t, task.HasTag("security"))
	assert.Equal(t, 120, task.EstimatedMins)

	// Test 2: Get task
	retrievedTask, err := taskService.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, task.ID, retrievedTask.ID)
	assert.Equal(t, task.Content, retrievedTask.Content)

	// Test 3: Update task status
	err = taskService.UpdateTaskStatus(ctx, task.ID, entities.StatusInProgress)
	require.NoError(t, err)

	// Verify status change
	updatedTask, err := taskService.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, entities.StatusInProgress, updatedTask.Status)

	// Test 4: Update task content and properties
	newContent := "Implement OAuth authentication"
	newEstimation := 180
	updates := services.TaskUpdates{
		Content:       &newContent,
		EstimatedMins: &newEstimation,
		AddTags:       []string{"oauth"},
		RemoveTags:    []string{"security"},
	}

	err = taskService.UpdateTask(ctx, task.ID, &updates)
	require.NoError(t, err)

	// Verify updates
	updatedTask, err = taskService.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, newContent, updatedTask.Content)
	assert.Equal(t, 180, updatedTask.EstimatedMins)
	assert.True(t, updatedTask.HasTag("oauth"))
	assert.False(t, updatedTask.HasTag("security"))

	// Test 5: Complete task
	err = taskService.UpdateTaskStatus(ctx, task.ID, entities.StatusCompleted)
	require.NoError(t, err)

	// Verify completion
	completedTask, err := taskService.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, entities.StatusCompleted, completedTask.Status)
	assert.NotNil(t, completedTask.CompletedAt)

	// Test 6: List tasks
	filters := ports.TaskFilters{}
	tasks, err := taskService.ListTasks(ctx, &filters)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task.ID, tasks[0].ID)

	// Test 7: Search tasks
	searchResults, err := taskService.SearchTasks(ctx, "OAuth", &ports.TaskFilters{})
	require.NoError(t, err)
	assert.Len(t, searchResults, 1)
	assert.Equal(t, task.ID, searchResults[0].ID)
}

// TestMultipleTasksAndFiltering tests task filtering and sorting
func TestMultipleTasksAndFiltering(t *testing.T) {
	// Setup
	fileStorage, err := storage.NewFileStorageWithPath(t.TempDir())
	require.NoError(t, err)

	gitDetector := repository.NewGitDetector()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskService := services.NewTaskService(fileStorage, gitDetector, logger)

	ctx := context.Background()

	// Create multiple tasks
	task1, err := taskService.CreateTask(ctx, "High priority pending task",
		services.WithPriority(entities.PriorityHigh),
		services.WithTags("urgent"),
	)
	require.NoError(t, err)

	task2, err := taskService.CreateTask(ctx, "Medium priority in progress task",
		services.WithPriority(entities.PriorityMedium),
		services.WithTags("feature"),
	)
	require.NoError(t, err)
	err = taskService.UpdateTaskStatus(ctx, task2.ID, entities.StatusInProgress)
	require.NoError(t, err)

	task3, err := taskService.CreateTask(ctx, "Low priority completed task",
		services.WithPriority(entities.PriorityLow),
		services.WithTags("bug"),
	)
	require.NoError(t, err)
	err = taskService.UpdateTaskStatus(ctx, task3.ID, entities.StatusInProgress)
	require.NoError(t, err)
	err = taskService.UpdateTaskStatus(ctx, task3.ID, entities.StatusCompleted)
	require.NoError(t, err)

	// Test filtering by status
	statusFilter := entities.StatusInProgress
	tasks, err := taskService.ListTasks(ctx, &ports.TaskFilters{
		Status: &statusFilter,
	})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task2.ID, tasks[0].ID)

	// Test filtering by priority
	priorityFilter := entities.PriorityHigh
	tasks, err = taskService.ListTasks(ctx, &ports.TaskFilters{
		Priority: &priorityFilter,
	})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)

	// Test filtering by tags
	tasks, err = taskService.ListTasks(ctx, &ports.TaskFilters{
		Tags: []string{"urgent"},
	})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)

	// Test listing all tasks (should be sorted by priority and status)
	allTasks, err := taskService.ListTasks(ctx, &ports.TaskFilters{})
	require.NoError(t, err)
	assert.Len(t, allTasks, 3)

	// Verify sorting: by priority first (high -> medium -> low), then by status
	assert.Equal(t, task1.ID, allTasks[0].ID) // High priority pending
	assert.Equal(t, task2.ID, allTasks[1].ID) // Medium priority in progress
	assert.Equal(t, task3.ID, allTasks[2].ID) // Low priority completed
}

// TestRepositoryIsolation tests that tasks are properly isolated by repository
func TestRepositoryIsolation(t *testing.T) {
	// Setup
	fileStorage, err := storage.NewFileStorageWithPath(t.TempDir())
	require.NoError(t, err)

	gitDetector := repository.NewGitDetector()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskService := services.NewTaskService(fileStorage, gitDetector, logger)

	ctx := context.Background()

	// Create tasks in different repositories manually using storage
	task1, err := entities.NewTask("Task in repo1", "repo1")
	require.NoError(t, err)
	err = fileStorage.SaveTask(ctx, task1)
	require.NoError(t, err)

	task2, err := entities.NewTask("Task in repo2", "repo2")
	require.NoError(t, err)
	err = fileStorage.SaveTask(ctx, task2)
	require.NoError(t, err)

	task3, err := entities.NewTask("Another task in repo1", "repo1")
	require.NoError(t, err)
	err = fileStorage.SaveTask(ctx, task3)
	require.NoError(t, err)

	// Test listing repositories
	repos, err := taskService.ListRepositories(ctx)
	require.NoError(t, err)
	assert.Len(t, repos, 2)
	assert.Contains(t, repos, "repo1")
	assert.Contains(t, repos, "repo2")

	// Test filtering by repository
	repo1Tasks, err := fileStorage.ListTasks(ctx, "repo1", &ports.TaskFilters{})
	require.NoError(t, err)
	assert.Len(t, repo1Tasks, 2)

	repo2Tasks, err := fileStorage.ListTasks(ctx, "repo2", &ports.TaskFilters{})
	require.NoError(t, err)
	assert.Len(t, repo2Tasks, 1)

	// Test repository stats
	stats, err := taskService.GetRepositoryStats(ctx, "repo1")
	require.NoError(t, err)
	assert.Equal(t, "repo1", stats.Repository)
	assert.Equal(t, 2, stats.TotalTasks)
	assert.Equal(t, 2, stats.PendingTasks)
}

// TestTaskLifecycleValidation tests business rule validation
func TestTaskLifecycleValidation(t *testing.T) {
	// Setup
	fileStorage, err := storage.NewFileStorageWithPath(t.TempDir())
	require.NoError(t, err)

	gitDetector := repository.NewGitDetector()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskService := services.NewTaskService(fileStorage, gitDetector, logger)

	ctx := context.Background()

	// Create task
	task, err := taskService.CreateTask(ctx, "Test task lifecycle")
	require.NoError(t, err)

	// Test valid status transitions
	err = taskService.UpdateTaskStatus(ctx, task.ID, entities.StatusInProgress)
	assert.NoError(t, err)

	err = taskService.UpdateTaskStatus(ctx, task.ID, entities.StatusCompleted)
	assert.NoError(t, err)

	// Test invalid status transition (completed -> in_progress)
	err = taskService.UpdateTaskStatus(ctx, task.ID, entities.StatusInProgress)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status transition")

	// Test valid reopening (completed -> pending)
	err = taskService.UpdateTaskStatus(ctx, task.ID, entities.StatusPending)
	assert.NoError(t, err)

	// Test cancellation
	err = taskService.UpdateTaskStatus(ctx, task.ID, entities.StatusCancelled)
	assert.NoError(t, err)

	// Test invalid transition from cancelled
	err = taskService.UpdateTaskStatus(ctx, task.ID, entities.StatusCompleted)
	assert.Error(t, err)

	// Test valid reopening from cancelled
	err = taskService.UpdateTaskStatus(ctx, task.ID, entities.StatusPending)
	assert.NoError(t, err)
}

// TestPersistenceAndRecovery tests data persistence across service restarts
func TestPersistenceAndRecovery(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	// First session: create and save tasks
	{
		fileStorage, err := storage.NewFileStorageWithPath(tempDir)
		require.NoError(t, err)

		gitDetector := repository.NewGitDetector()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		taskService := services.NewTaskService(fileStorage, gitDetector, logger)

		task, err := taskService.CreateTask(ctx, "Persistent task",
			services.WithPriority(entities.PriorityHigh),
			services.WithTags("important"),
		)
		require.NoError(t, err)

		err = taskService.UpdateTaskStatus(ctx, task.ID, entities.StatusInProgress)
		require.NoError(t, err)
	}

	// Second session: recreate service and verify data persistence
	{
		fileStorage, err := storage.NewFileStorageWithPath(tempDir)
		require.NoError(t, err)

		gitDetector := repository.NewGitDetector()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		taskService := services.NewTaskService(fileStorage, gitDetector, logger)

		// List tasks
		tasks, err := taskService.ListTasks(ctx, &ports.TaskFilters{})
		require.NoError(t, err)
		assert.Len(t, tasks, 1)

		task := tasks[0]
		assert.Equal(t, "Persistent task", task.Content)
		assert.Equal(t, entities.PriorityHigh, task.Priority)
		assert.Equal(t, entities.StatusInProgress, task.Status)
		assert.True(t, task.HasTag("important"))
	}
}

// TestErrorHandlingAndRecovery tests error scenarios and recovery
func TestErrorHandlingAndRecovery(t *testing.T) {
	// Setup
	fileStorage, err := storage.NewFileStorageWithPath(t.TempDir())
	require.NoError(t, err)

	gitDetector := repository.NewGitDetector()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	taskService := services.NewTaskService(fileStorage, gitDetector, logger)

	ctx := context.Background()

	// Test getting non-existent task
	_, err = taskService.GetTask(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test updating non-existent task
	err = taskService.UpdateTaskStatus(ctx, "non-existent-id", entities.StatusCompleted)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test deleting non-existent task
	err = taskService.DeleteTask(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test creating task with invalid content
	_, err = taskService.CreateTask(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task")

	// Test creating task with very long content
	longContent := string(make([]byte, 1001))
	_, err = taskService.CreateTask(ctx, longContent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task")
}
