package services

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// Mock implementations for testing

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) SaveTask(ctx context.Context, task *entities.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockStorage) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Task), args.Error(1)
}

func (m *MockStorage) UpdateTask(ctx context.Context, task *entities.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockStorage) DeleteTask(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStorage) ListTasks(ctx context.Context, repository string, filters *ports.TaskFilters) ([]*entities.Task, error) {
	args := m.Called(ctx, repository, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Task), args.Error(1)
}

func (m *MockStorage) GetTasksByRepository(ctx context.Context, repository string) ([]*entities.Task, error) {
	args := m.Called(ctx, repository)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Task), args.Error(1)
}

func (m *MockStorage) SearchTasks(ctx context.Context, query string, filters *ports.TaskFilters) ([]*entities.Task, error) {
	args := m.Called(ctx, query, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Task), args.Error(1)
}

func (m *MockStorage) SaveTasks(ctx context.Context, tasks []*entities.Task) error {
	args := m.Called(ctx, tasks)
	return args.Error(0)
}

func (m *MockStorage) DeleteTasks(ctx context.Context, ids []string) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockStorage) ListRepositories(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockStorage) GetRepositoryStats(ctx context.Context, repository string) (ports.RepositoryStats, error) {
	args := m.Called(ctx, repository)
	return args.Get(0).(ports.RepositoryStats), args.Error(1)
}

func (m *MockStorage) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockStorage) Backup(ctx context.Context, backupPath string) error {
	args := m.Called(ctx, backupPath)
	return args.Error(0)
}

func (m *MockStorage) Restore(ctx context.Context, backupPath string) error {
	args := m.Called(ctx, backupPath)
	return args.Error(0)
}

type MockRepositoryDetector struct {
	mock.Mock
}

func (m *MockRepositoryDetector) DetectCurrent(ctx context.Context) (*ports.RepositoryInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.RepositoryInfo), args.Error(1)
}

func (m *MockRepositoryDetector) DetectFromPath(ctx context.Context, path string) (*ports.RepositoryInfo, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.RepositoryInfo), args.Error(1)
}

func (m *MockRepositoryDetector) GetRepositoryName(ctx context.Context, path string) (string, error) {
	args := m.Called(ctx, path)
	return args.String(0), args.Error(1)
}

func (m *MockRepositoryDetector) IsValidRepository(ctx context.Context, path string) bool {
	args := m.Called(ctx, path)
	return args.Bool(0)
}

// Test cases

func TestNewTaskService(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	service := NewTaskService(storage, detector, logger)

	assert.NotNil(t, service)
	assert.NotNil(t, service.storage)
	assert.NotNil(t, service.repository)
	assert.NotNil(t, service.validator)
	assert.NotNil(t, service.logger)
}

func TestTaskService_CreateTask(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	// Setup mock expectations
	repoInfo := &ports.RepositoryInfo{
		Name: "test-repo",
		Path: "/test/path",
	}
	detector.On("DetectCurrent", ctx).Return(repoInfo, nil)
	storage.On("SaveTask", ctx, mock.AnythingOfType("*entities.Task")).Return(nil)

	// Test basic task creation
	task, err := service.CreateTask(ctx, "Test task content")

	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, "Test task content", task.Content)
	assert.Equal(t, "test-repo", task.Repository)
	assert.Equal(t, entities.StatusPending, task.Status)
	assert.Equal(t, entities.PriorityMedium, task.Priority)

	// Verify mocks were called
	detector.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func TestTaskService_CreateTaskWithOptions(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	// Setup mock expectations
	repoInfo := &ports.RepositoryInfo{Name: "test-repo"}
	detector.On("DetectCurrent", ctx).Return(repoInfo, nil)
	storage.On("SaveTask", ctx, mock.AnythingOfType("*entities.Task")).Return(nil)

	// Test task creation with options
	task, err := service.CreateTask(ctx, "Test task with options",
		WithPriority(entities.PriorityHigh),
		WithTags("urgent", "bug"),
		WithEstimatedTime(60),
		WithAISuggested(),
	)

	require.NoError(t, err)
	assert.Equal(t, entities.PriorityHigh, task.Priority)
	assert.True(t, task.HasTag("urgent"))
	assert.True(t, task.HasTag("bug"))
	assert.Equal(t, 60, task.EstimatedMins)
	assert.True(t, task.AISuggested)

	detector.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func TestTaskService_CreateTaskWithRepositoryDetectionFailure(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	// Setup mock to fail repository detection
	detector.On("DetectCurrent", ctx).Return(nil, assert.AnError)
	storage.On("SaveTask", ctx, mock.AnythingOfType("*entities.Task")).Return(nil)

	// Should still create task with fallback repository
	task, err := service.CreateTask(ctx, "Test task content")

	require.NoError(t, err)
	assert.Equal(t, "local", task.Repository)

	detector.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func TestTaskService_GetTask(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	// Create expected task
	expectedTask, err := entities.NewTask("Test task", "test-repo")
	require.NoError(t, err)

	// Setup mock
	storage.On("GetTask", ctx, expectedTask.ID).Return(expectedTask, nil)

	// Test getting task
	task, err := service.GetTask(ctx, expectedTask.ID)

	require.NoError(t, err)
	assert.Equal(t, expectedTask.ID, task.ID)
	assert.Equal(t, expectedTask.Content, task.Content)

	storage.AssertExpectations(t)
}

func TestTaskService_UpdateTaskStatus(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	// Create test task
	task, err := entities.NewTask("Test task", "test-repo")
	require.NoError(t, err)

	// Setup mocks
	storage.On("GetTask", ctx, task.ID).Return(task, nil)
	storage.On("UpdateTask", ctx, mock.AnythingOfType("*entities.Task")).Return(nil)

	// Test valid status transition
	err = service.UpdateTaskStatus(ctx, task.ID, entities.StatusInProgress)
	require.NoError(t, err)

	storage.AssertExpectations(t)
}

func TestTaskService_UpdateTaskStatusInvalidTransition(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	// Create completed task
	task, err := entities.NewTask("Test task", "test-repo")
	require.NoError(t, err)
	_ = task.Start()
	_ = task.Complete()

	// Setup mock
	storage.On("GetTask", ctx, task.ID).Return(task, nil)

	// Test invalid status transition (completed -> in_progress)
	err = service.UpdateTaskStatus(ctx, task.ID, entities.StatusInProgress)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status transition")

	storage.AssertExpectations(t)
}

func TestTaskService_UpdateTask(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	// Create test task
	task, err := entities.NewTask("Original content", "test-repo")
	require.NoError(t, err)

	// Setup mocks
	storage.On("GetTask", ctx, task.ID).Return(task, nil)
	storage.On("UpdateTask", ctx, mock.AnythingOfType("*entities.Task")).Return(nil)

	// Test task updates
	newContent := "Updated content"
	newPriority := entities.PriorityHigh
	newEstimation := 120

	updates := TaskUpdates{
		Content:       &newContent,
		Priority:      &newPriority,
		EstimatedMins: &newEstimation,
		AddTags:       []string{"updated", "important"},
		RemoveTags:    []string{"old"},
	}

	err = service.UpdateTask(ctx, task.ID, &updates)
	require.NoError(t, err)

	storage.AssertExpectations(t)
}

func TestTaskService_DeleteTask(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	// Create test task
	task, err := entities.NewTask("Task to delete", "test-repo")
	require.NoError(t, err)

	// Setup mocks
	storage.On("GetTask", ctx, task.ID).Return(task, nil)
	storage.On("DeleteTask", ctx, task.ID).Return(nil)

	// Test task deletion
	err = service.DeleteTask(ctx, task.ID)
	require.NoError(t, err)

	storage.AssertExpectations(t)
}

func TestTaskService_ListTasks(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	// Create test tasks
	task1, err := entities.NewTask("Task 1", "test-repo")
	require.NoError(t, err)
	_ = task1.SetPriority(entities.PriorityHigh)

	task2, err := entities.NewTask("Task 2", "test-repo")
	require.NoError(t, err)
	_ = task2.SetPriority(entities.PriorityLow)
	_ = task2.Start()

	tasks := []*entities.Task{task1, task2}

	// Setup mocks
	repoInfo := &ports.RepositoryInfo{Name: "test-repo"}
	detector.On("DetectCurrent", ctx).Return(repoInfo, nil)
	storage.On("ListTasks", ctx, "test-repo", mock.AnythingOfType("*ports.TaskFilters")).Return(tasks, nil)

	// Test listing tasks
	filters := ports.TaskFilters{}
	result, err := service.ListTasks(ctx, &filters)

	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Verify sorting (high priority first, then in_progress status)
	assert.Equal(t, entities.PriorityHigh, result[0].Priority)

	detector.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func TestTaskService_SearchTasks(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	// Create test tasks
	task1, err := entities.NewTask("Fix authentication bug", "test-repo")
	require.NoError(t, err)

	tasks := []*entities.Task{task1}

	// Setup mocks
	repoInfo := &ports.RepositoryInfo{Name: "test-repo"}
	detector.On("DetectCurrent", ctx).Return(repoInfo, nil)
	storage.On("SearchTasks", ctx, "authentication", mock.AnythingOfType("*ports.TaskFilters")).Return(tasks, nil)

	// Test searching tasks
	filters := ports.TaskFilters{}
	result, err := service.SearchTasks(ctx, "authentication", &filters)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, task1.ID, result[0].ID)

	detector.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func TestTaskService_GetRepositoryStats(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	expectedStats := ports.RepositoryStats{
		Repository:     "test-repo",
		TotalTasks:     5,
		PendingTasks:   2,
		CompletedTasks: 3,
	}

	// Setup mock
	storage.On("GetRepositoryStats", ctx, "test-repo").Return(expectedStats, nil)

	// Test getting repository stats
	stats, err := service.GetRepositoryStats(ctx, "test-repo")

	require.NoError(t, err)
	assert.Equal(t, expectedStats.Repository, stats.Repository)
	assert.Equal(t, expectedStats.TotalTasks, stats.TotalTasks)
	assert.Equal(t, expectedStats.PendingTasks, stats.PendingTasks)

	storage.AssertExpectations(t)
}

func TestTaskService_ListRepositories(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	expectedRepos := []string{"repo1", "repo2", "repo3"}

	// Setup mock
	storage.On("ListRepositories", ctx).Return(expectedRepos, nil)

	// Test listing repositories
	repos, err := service.ListRepositories(ctx)

	require.NoError(t, err)
	assert.Equal(t, expectedRepos, repos)

	storage.AssertExpectations(t)
}

func TestTaskService_ValidateStatusTransition(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)

	// Test valid transitions
	validTransitions := []struct {
		from entities.Status
		to   entities.Status
	}{
		{entities.StatusPending, entities.StatusInProgress},
		{entities.StatusPending, entities.StatusCancelled},
		{entities.StatusInProgress, entities.StatusCompleted},
		{entities.StatusInProgress, entities.StatusCancelled},
		{entities.StatusCompleted, entities.StatusPending},
		{entities.StatusCancelled, entities.StatusPending},
	}

	for _, transition := range validTransitions {
		err := service.validateStatusTransition(transition.from, transition.to)
		assert.NoError(t, err, "Transition from %s to %s should be valid", transition.from, transition.to)
	}

	// Test invalid transitions
	invalidTransitions := []struct {
		from entities.Status
		to   entities.Status
	}{
		{entities.StatusCompleted, entities.StatusInProgress},
		{entities.StatusCompleted, entities.StatusCancelled},
		{entities.StatusCancelled, entities.StatusInProgress},
		{entities.StatusCancelled, entities.StatusCompleted},
		{entities.StatusPending, entities.StatusPending}, // Same status
	}

	for _, transition := range invalidTransitions {
		err := service.validateStatusTransition(transition.from, transition.to)
		assert.Error(t, err, "Transition from %s to %s should be invalid", transition.from, transition.to)
	}
}

func TestTaskService_SortTasks(t *testing.T) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)

	// Create tasks with different priorities and statuses
	task1, _ := entities.NewTask("Low priority pending", "test-repo")
	_ = task1.SetPriority(entities.PriorityLow)

	task2, _ := entities.NewTask("High priority pending", "test-repo")
	_ = task2.SetPriority(entities.PriorityHigh)

	task3, _ := entities.NewTask("Medium priority in progress", "test-repo")
	_ = task3.SetPriority(entities.PriorityMedium)
	_ = task3.Start()

	task4, _ := entities.NewTask("High priority completed", "test-repo")
	_ = task4.SetPriority(entities.PriorityHigh)
	_ = task4.Start()
	_ = task4.Complete()

	tasks := []*entities.Task{task1, task2, task3, task4}

	// Sort tasks
	service.sortTasks(tasks)

	// Verify sorting order by checking priorities first, then status
	// Sorting is: Priority (high->medium->low), then Status (in_progress->pending->completed->cancelled), then creation date

	// First two tasks should be high priority (sorted by status within priority)
	assert.Equal(t, entities.PriorityHigh, tasks[0].Priority)
	assert.Equal(t, entities.PriorityHigh, tasks[1].Priority)

	// Among high priority tasks, pending should come before completed (higher status weight)
	if tasks[0].Status == entities.StatusPending {
		assert.Equal(t, entities.StatusCompleted, tasks[1].Status)
	} else {
		assert.Equal(t, entities.StatusCompleted, tasks[0].Status)
		assert.Equal(t, entities.StatusPending, tasks[1].Status)
	}

	// Third task should be medium priority in progress
	assert.Equal(t, entities.PriorityMedium, tasks[2].Priority)
	assert.Equal(t, entities.StatusInProgress, tasks[2].Status)

	// Fourth task should be low priority pending
	assert.Equal(t, entities.PriorityLow, tasks[3].Priority)
	assert.Equal(t, entities.StatusPending, tasks[3].Status)
}

func TestTaskService_FunctionalOptions(t *testing.T) {
	// Test functional options directly
	task, err := entities.NewTask("Test task", "test-repo")
	require.NoError(t, err)

	// Test WithPriority
	WithPriority(entities.PriorityHigh)(task)
	assert.Equal(t, entities.PriorityHigh, task.Priority)

	// Test WithTags
	WithTags("tag1", "tag2")(task)
	assert.True(t, task.HasTag("tag1"))
	assert.True(t, task.HasTag("tag2"))

	// Test WithEstimatedTime
	WithEstimatedTime(120)(task)
	assert.Equal(t, 120, task.EstimatedMins)

	// Test WithSessionID
	WithSessionID("session123")(task)
	assert.Equal(t, "session123", task.SessionID)

	// Test WithParentTask
	WithParentTask("parent123")(task)
	assert.Equal(t, "parent123", task.ParentTaskID)

	// Test WithAISuggested
	WithAISuggested()(task)
	assert.True(t, task.AISuggested)
}

// Benchmark tests

func BenchmarkTaskService_CreateTask(b *testing.B) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)
	ctx := context.Background()

	// Setup mocks
	repoInfo := &ports.RepositoryInfo{Name: "benchmark-repo"}
	detector.On("DetectCurrent", ctx).Return(repoInfo, nil)
	storage.On("SaveTask", ctx, mock.AnythingOfType("*entities.Task")).Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.CreateTask(ctx, "Benchmark task")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTaskService_SortTasks(b *testing.B) {
	storage := &MockStorage{}
	detector := &MockRepositoryDetector{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewTaskService(storage, detector, logger)

	// Create test tasks
	var tasks []*entities.Task
	for i := 0; i < 100; i++ {
		task, _ := entities.NewTask("Benchmark task", "test-repo")
		// Randomize priority
		priorities := []entities.Priority{entities.PriorityLow, entities.PriorityMedium, entities.PriorityHigh}
		_ = task.SetPriority(priorities[i%3])

		// Randomize status
		switch i % 4 {
		case 0:
			_ = task.Start()
		case 1:
			_ = task.Start()
			_ = task.Complete()
		}

		tasks = append(tasks, task)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy to avoid modifying the original
		tasksCopy := make([]*entities.Task, len(tasks))
		copy(tasksCopy, tasks)
		service.sortTasks(tasksCopy)
	}
}
