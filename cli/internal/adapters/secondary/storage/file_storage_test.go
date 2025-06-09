package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

func TestNewFileStorage(t *testing.T) {
	// Test with custom path
	tempDir := t.TempDir()
	storage, err := NewFileStorageWithPath(tempDir)

	require.NoError(t, err)
	require.NotNil(t, storage)
	assert.Equal(t, tempDir, storage.basePath)

	// Verify health check passes
	err = storage.HealthCheck(context.Background())
	assert.NoError(t, err)
}

func TestFileStorage_SaveAndGetTask(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create test task
	task, err := entities.NewTask("Test task content", "test-repo")
	require.NoError(t, err)

	// Save task
	err = storage.SaveTask(ctx, task)
	require.NoError(t, err)

	// Retrieve task
	retrieved, err := storage.GetTask(ctx, task.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Verify task data
	assert.Equal(t, task.ID, retrieved.ID)
	assert.Equal(t, task.Content, retrieved.Content)
	assert.Equal(t, task.Status, retrieved.Status)
	assert.Equal(t, task.Priority, retrieved.Priority)
	assert.Equal(t, task.Repository, retrieved.Repository)
}

func TestFileStorage_UpdateTask(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create and save task
	task, err := entities.NewTask("Original content", "test-repo")
	require.NoError(t, err)
	err = storage.SaveTask(ctx, task)
	require.NoError(t, err)

	// Update task
	_ = task.UpdateContent("Updated content")
	_ = task.SetPriority(entities.PriorityHigh)

	err = storage.UpdateTask(ctx, task)
	require.NoError(t, err)

	// Retrieve and verify
	updated, err := storage.GetTask(ctx, task.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated content", updated.Content)
	assert.Equal(t, entities.PriorityHigh, updated.Priority)
}

func TestFileStorage_DeleteTask(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create and save task
	task, err := entities.NewTask("Task to delete", "test-repo")
	require.NoError(t, err)
	err = storage.SaveTask(ctx, task)
	require.NoError(t, err)

	// Verify task exists
	_, err = storage.GetTask(ctx, task.ID)
	require.NoError(t, err)

	// Delete task
	err = storage.DeleteTask(ctx, task.ID)
	require.NoError(t, err)

	// Verify task is gone
	_, err = storage.GetTask(ctx, task.ID)
	assert.Error(t, err)
}

func TestFileStorage_ListTasks(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create test tasks
	task1, err := entities.NewTask("Task 1", "test-repo")
	require.NoError(t, err)
	_ = task1.SetPriority(entities.PriorityHigh)
	task1.AddTag("urgent")

	task2, err := entities.NewTask("Task 2", "test-repo")
	require.NoError(t, err)
	_ = task2.SetPriority(entities.PriorityLow)
	_ = task2.Start()

	task3, err := entities.NewTask("Task 3", "other-repo")
	require.NoError(t, err)

	// Save tasks
	err = storage.SaveTask(ctx, task1)
	require.NoError(t, err)
	err = storage.SaveTask(ctx, task2)
	require.NoError(t, err)
	err = storage.SaveTask(ctx, task3)
	require.NoError(t, err)

	// Test listing without filters
	tasks, err := storage.ListTasks(ctx, "test-repo", &ports.TaskFilters{})
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Test status filter
	statusFilter := entities.StatusInProgress
	tasks, err = storage.ListTasks(ctx, "test-repo", &ports.TaskFilters{
		Status: &statusFilter,
	})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task2.ID, tasks[0].ID)

	// Test priority filter
	priorityFilter := entities.PriorityHigh
	tasks, err = storage.ListTasks(ctx, "test-repo", &ports.TaskFilters{
		Priority: &priorityFilter,
	})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)

	// Test tag filter
	tasks, err = storage.ListTasks(ctx, "test-repo", &ports.TaskFilters{
		Tags: []string{"urgent"},
	})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)
}

func TestFileStorage_SearchTasks(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create test tasks with different content
	task1, err := entities.NewTask("Fix authentication bug", "test-repo")
	require.NoError(t, err)
	task1.AddTag("bug")

	task2, err := entities.NewTask("Add user authentication", "test-repo")
	require.NoError(t, err)
	task2.AddTag("feature")

	task3, err := entities.NewTask("Update documentation", "test-repo")
	require.NoError(t, err)

	// Save tasks
	err = storage.SaveTask(ctx, task1)
	require.NoError(t, err)
	err = storage.SaveTask(ctx, task2)
	require.NoError(t, err)
	err = storage.SaveTask(ctx, task3)
	require.NoError(t, err)

	// Search for "authentication"
	tasks, err := storage.SearchTasks(ctx, "authentication", &ports.TaskFilters{})
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Search for "bug"
	tasks, err = storage.SearchTasks(ctx, "bug", &ports.TaskFilters{})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task1.ID, tasks[0].ID)

	// Search with filters
	tagFilter := []string{"feature"}
	tasks, err = storage.SearchTasks(ctx, "authentication", &ports.TaskFilters{
		Tags: tagFilter,
	})
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, task2.ID, tasks[0].ID)
}

func TestFileStorage_BulkOperations(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create multiple tasks
	var tasks []*entities.Task
	for i := 0; i < 5; i++ {
		task, err := entities.NewTask(fmt.Sprintf("Task %d", i+1), "bulk-repo")
		require.NoError(t, err)
		tasks = append(tasks, task)
	}

	// Bulk save
	err := storage.SaveTasks(ctx, tasks)
	require.NoError(t, err)

	// Verify all tasks were saved
	allTasks, err := storage.GetTasksByRepository(ctx, "bulk-repo")
	require.NoError(t, err)
	assert.Len(t, allTasks, 5)

	// Bulk delete
	ids := make([]string, 0, 3)
	for _, task := range tasks[:3] {
		ids = append(ids, task.ID)
	}

	err = storage.DeleteTasks(ctx, ids)
	require.NoError(t, err)

	// Verify remaining tasks
	remainingTasks, err := storage.GetTasksByRepository(ctx, "bulk-repo")
	require.NoError(t, err)
	assert.Len(t, remainingTasks, 2)
}

func TestFileStorage_RepositoryOperations(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create tasks in different repositories
	task1, err := entities.NewTask("Task in repo1", "repo1")
	require.NoError(t, err)
	task2, err := entities.NewTask("Task in repo2", "repo2")
	require.NoError(t, err)
	task3, err := entities.NewTask("Another task in repo1", "repo1")
	require.NoError(t, err)
	_ = task3.Complete()

	err = storage.SaveTask(ctx, task1)
	require.NoError(t, err)
	err = storage.SaveTask(ctx, task2)
	require.NoError(t, err)
	err = storage.SaveTask(ctx, task3)
	require.NoError(t, err)

	// List repositories
	repos, err := storage.ListRepositories(ctx)
	require.NoError(t, err)
	assert.Len(t, repos, 2)
	assert.Contains(t, repos, "repo1")
	assert.Contains(t, repos, "repo2")

	// Get repository stats
	stats, err := storage.GetRepositoryStats(ctx, "repo1")
	require.NoError(t, err)
	assert.Equal(t, "repo1", stats.Repository)
	assert.Equal(t, 2, stats.TotalTasks)
	assert.Equal(t, 1, stats.PendingTasks)
	assert.Equal(t, 1, stats.CompletedTasks)
}

func TestFileStorage_BackupAndRestore(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create test data
	task1, err := entities.NewTask("Task 1", "test-repo")
	require.NoError(t, err)
	task2, err := entities.NewTask("Task 2", "other-repo")
	require.NoError(t, err)

	err = storage.SaveTask(ctx, task1)
	require.NoError(t, err)
	err = storage.SaveTask(ctx, task2)
	require.NoError(t, err)

	// Create backup
	backupPath := filepath.Join(t.TempDir(), "backup.json")
	err = storage.Backup(ctx, backupPath)
	require.NoError(t, err)

	// Verify backup file exists
	_, err = os.Stat(backupPath)
	require.NoError(t, err)

	// Create new storage and restore
	newStorage := setupTestStorage(t)
	err = newStorage.Restore(ctx, backupPath)
	require.NoError(t, err)

	// Verify restored data
	restoredTask1, err := newStorage.GetTask(ctx, task1.ID)
	require.NoError(t, err)
	assert.Equal(t, task1.Content, restoredTask1.Content)

	restoredTask2, err := newStorage.GetTask(ctx, task2.ID)
	require.NoError(t, err)
	assert.Equal(t, task2.Content, restoredTask2.Content)
}

func TestFileStorage_AtomicOperations(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create task
	task, err := entities.NewTask("Test atomic operations", "atomic-repo")
	require.NoError(t, err)

	// Verify no backup/temp files exist initially
	repoPath := storage.getRepositoryPath("atomic-repo")
	tempFile := filepath.Join(repoPath, TasksFileName+TempSuffix)
	backupFile := filepath.Join(repoPath, TasksFileName+BackupSuffix)

	_, err = os.Stat(tempFile)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(backupFile)
	assert.True(t, os.IsNotExist(err))

	// Save task
	err = storage.SaveTask(ctx, task)
	require.NoError(t, err)

	// Verify no temp/backup files remain after successful operation
	_, err = os.Stat(tempFile)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(backupFile)
	assert.True(t, os.IsNotExist(err))

	// Verify main file exists
	mainFile := filepath.Join(repoPath, TasksFileName)
	_, err = os.Stat(mainFile)
	assert.NoError(t, err)
}

func TestFileStorage_ConcurrentAccess(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Test concurrent writes
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			task, err := entities.NewTask(fmt.Sprintf("Concurrent task %d", index), "concurrent-repo")
			if err != nil {
				t.Errorf("Failed to create task: %v", err)
				done <- false
				return
			}

			err = storage.SaveTask(ctx, task)
			if err != nil {
				t.Errorf("Failed to save task: %v", err)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		success := <-done
		assert.True(t, success)
	}

	// Verify all tasks were saved
	tasks, err := storage.GetTasksByRepository(ctx, "concurrent-repo")
	require.NoError(t, err)
	assert.Len(t, tasks, numGoroutines)
}

func TestFileStorage_ErrorHandling(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Test getting non-existent task
	_, err := storage.GetTask(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test deleting non-existent task
	err = storage.DeleteTask(ctx, "non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test updating non-existent task
	task, err := entities.NewTask("Test task", "test-repo")
	require.NoError(t, err)
	// Use a valid UUID that doesn't exist
	task.ID = "11111111-1111-1111-1111-111111111111"

	err = storage.UpdateTask(ctx, task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test saving invalid task
	invalidTask := &entities.Task{}
	err = storage.SaveTask(ctx, invalidTask)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task")
}

func TestFileStorage_HealthCheck(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Normal health check should pass
	err := storage.HealthCheck(ctx)
	assert.NoError(t, err)

	// Create storage with read-only directory (on Unix systems)
	if os.Getuid() != 0 { // Skip if running as root
		readOnlyDir := filepath.Join(t.TempDir(), "readonly")
		// #nosec G301 -- Intentionally creating read-only directory for testing error conditions
		err := os.MkdirAll(readOnlyDir, 0o555)
		require.NoError(t, err)

		readOnlyStorage, err := NewFileStorageWithPath(readOnlyDir)
		if err == nil {
			// Health check should fail on read-only directory
			err = readOnlyStorage.HealthCheck(ctx)
			assert.Error(t, err)
		}
	}
}

// Helper functions

func setupTestStorage(t *testing.T) *FileStorage {
	t.Helper()
	tempDir := t.TempDir()
	storage, err := NewFileStorageWithPath(tempDir)
	require.NoError(t, err)
	return storage
}

// Benchmark tests

func BenchmarkFileStorage_SaveTask(b *testing.B) {
	storage := setupBenchmarkStorage(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task, err := entities.NewTask(fmt.Sprintf("Benchmark task %d", i), "benchmark-repo")
		if err != nil {
			b.Fatal(err)
		}

		if err := storage.SaveTask(ctx, task); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileStorage_GetTask(b *testing.B) {
	storage := setupBenchmarkStorage(b)
	ctx := context.Background()

	// Create test tasks
	var taskIDs []string
	for i := 0; i < 100; i++ {
		task, err := entities.NewTask(fmt.Sprintf("Benchmark task %d", i), "benchmark-repo")
		if err != nil {
			b.Fatal(err)
		}
		if err := storage.SaveTask(ctx, task); err != nil {
			b.Fatal(err)
		}
		taskIDs = append(taskIDs, task.ID)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.GetTask(ctx, taskIDs[i%len(taskIDs)])
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileStorage_ListTasks(b *testing.B) {
	storage := setupBenchmarkStorage(b)
	ctx := context.Background()

	// Create test tasks
	for i := 0; i < 100; i++ {
		task, err := entities.NewTask(fmt.Sprintf("Benchmark task %d", i), "benchmark-repo")
		if err != nil {
			b.Fatal(err)
		}
		if err := storage.SaveTask(ctx, task); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.ListTasks(ctx, "benchmark-repo", &ports.TaskFilters{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func setupBenchmarkStorage(b *testing.B) *FileStorage {
	b.Helper()
	tempDir := b.TempDir()
	storage, err := NewFileStorageWithPath(tempDir)
	if err != nil {
		b.Fatal(err)
	}
	return storage
}
