// Package services implements business logic and use cases
// for the lerian-mcp-memory CLI application.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

const (
	// LocalRepository represents the default repository name when no repository is detected
	LocalRepository = "local"
)

// TaskService implements core business logic for task management
type TaskService struct {
	storage    ports.Storage
	repository ports.RepositoryDetector
	validator  *validator.Validate
	logger     *slog.Logger
	mcpClient  ports.MCPClient
}

// TaskOption defines functional options for task creation
type TaskOption func(*entities.Task)

// NewTaskService creates a new TaskService with injected dependencies
func NewTaskService(storage ports.Storage, repo ports.RepositoryDetector, logger *slog.Logger) *TaskService {
	return &TaskService{
		storage:    storage,
		repository: repo,
		validator:  validator.New(),
		logger:     logger,
	}
}

// SetMCPClient sets the MCP client for synchronization
func (s *TaskService) SetMCPClient(client ports.MCPClient) {
	s.mcpClient = client
}

// CreateTask creates a new task with validation and repository detection
func (s *TaskService) CreateTask(ctx context.Context, content string, options ...TaskOption) (*entities.Task, error) {
	// Detect repository context
	repoInfo, err := s.repository.DetectCurrent(ctx)
	if err != nil {
		s.logger.Warn("failed to detect repository, using fallback",
			slog.Any("error", err))
		repoInfo = &ports.RepositoryInfo{
			Name: LocalRepository,
			Path: ".",
		}
	}

	// Create task with detected repository
	task, err := entities.NewTask(content, repoInfo.Name)
	if err != nil {
		return nil, fmt.Errorf("invalid task: %w", err)
	}

	// Apply functional options
	for _, option := range options {
		option(task)
	}

	// Validate final task
	if err := task.Validate(); err != nil {
		return nil, fmt.Errorf("task validation failed: %w", err)
	}

	// Save task
	if err := s.storage.SaveTask(ctx, task); err != nil {
		s.logger.Error("failed to save task",
			slog.String("task_id", task.ID),
			slog.String("repository", repoInfo.Name),
			slog.Any("error", err))
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	s.logger.Info("task created",
		slog.String("task_id", task.ID),
		slog.String("repository", repoInfo.Name),
		slog.String("content", content),
		slog.String("priority", string(task.Priority)))

	// Sync with MCP if available and online
	if s.mcpClient != nil && s.mcpClient.IsOnline() {
		go func(parentCtx context.Context) {
			syncCtx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
			defer cancel()

			if err := s.mcpClient.SyncTask(syncCtx, task); err != nil {
				s.logger.Warn("failed to sync task with MCP",
					slog.String("task_id", task.ID),
					slog.Any("error", err))
			} else {
				s.logger.Debug("task synced with MCP",
					slog.String("task_id", task.ID))
			}
		}(ctx)
	}

	return task, nil
}

// ListTasks returns tasks with filtering and sorting
func (s *TaskService) ListTasks(ctx context.Context, filters *ports.TaskFilters) ([]*entities.Task, error) {
	if filters == nil {
		filters = &ports.TaskFilters{}
	}

	// Use current repository if not specified
	if filters.Repository == "" {
		repoInfo, err := s.repository.DetectCurrent(ctx)
		if err == nil {
			filters.Repository = repoInfo.Name
		} else {
			filters.Repository = LocalRepository
		}
	}

	tasks, err := s.storage.ListTasks(ctx, filters.Repository, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Apply additional filtering that might be complex for storage layer
	filtered := s.applyAdvancedFilters(tasks, filters)

	// Sort tasks by priority and creation date
	s.sortTasks(filtered)

	s.logger.Debug("tasks listed",
		slog.String("repository", filters.Repository),
		slog.Int("total_count", len(filtered)))

	return filtered, nil
}

// GetTask retrieves a single task by ID
func (s *TaskService) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	task, err := s.storage.GetTask(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	return task, nil
}

// UpdateTaskStatus updates the status of a task with business rule validation
func (s *TaskService) UpdateTaskStatus(ctx context.Context, taskID string, newStatus entities.Status) error {
	task, err := s.storage.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	oldStatus := task.Status

	// Validate status transition
	if err := s.validateStatusTransition(task.Status, newStatus); err != nil {
		return fmt.Errorf("invalid status transition: %w", err)
	}

	// Apply status change using entity methods
	switch newStatus {
	case entities.StatusInProgress:
		if err := task.Start(); err != nil {
			return fmt.Errorf("failed to start task: %w", err)
		}
	case entities.StatusCompleted:
		if err := task.Complete(); err != nil {
			return fmt.Errorf("failed to complete task: %w", err)
		}
	case entities.StatusCancelled:
		if err := task.Cancel(); err != nil {
			return fmt.Errorf("failed to cancel task: %w", err)
		}
	case entities.StatusPending:
		if err := task.Reset(); err != nil {
			return fmt.Errorf("failed to reset task: %w", err)
		}
	}

	// Save updated task
	if err := s.storage.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Info("task status updated",
		slog.String("task_id", taskID),
		slog.String("old_status", string(oldStatus)),
		slog.String("new_status", string(newStatus)))

	return nil
}

// UpdateTask updates task content and properties
func (s *TaskService) UpdateTask(ctx context.Context, taskID string, updates *TaskUpdates) error {
	if updates == nil {
		return errors.New("updates cannot be nil")
	}

	task, err := s.storage.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if err := s.applyTaskUpdates(task, updates); err != nil {
		return err
	}

	if err := s.storage.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to save updated task: %w", err)
	}

	s.logger.Info("task updated",
		slog.String("task_id", taskID))

	return nil
}

// applyTaskUpdates applies all updates to the task
func (s *TaskService) applyTaskUpdates(task *entities.Task, updates *TaskUpdates) error {
	if err := s.applyBasicUpdates(task, updates); err != nil {
		return err
	}

	s.applyTagUpdates(task, updates)
	s.applyMetadataUpdates(task, updates)

	return nil
}

// applyBasicUpdates applies basic field updates (content, priority, times)
func (s *TaskService) applyBasicUpdates(task *entities.Task, updates *TaskUpdates) error {
	if updates.Content != nil {
		if err := task.UpdateContent(*updates.Content); err != nil {
			return fmt.Errorf("failed to update content: %w", err)
		}
	}

	if updates.Priority != nil {
		if err := task.SetPriority(*updates.Priority); err != nil {
			return fmt.Errorf("failed to update priority: %w", err)
		}
	}

	if updates.DueDate != nil {
		if err := task.SetDueDate(updates.DueDate); err != nil {
			return fmt.Errorf("failed to update due date: %w", err)
		}
	}

	if updates.EstimatedMins != nil {
		if err := task.SetEstimation(*updates.EstimatedMins); err != nil {
			return fmt.Errorf("failed to update estimation: %w", err)
		}
	}

	if updates.ActualMins != nil {
		if err := task.SetActualTime(*updates.ActualMins); err != nil {
			return fmt.Errorf("failed to update actual time: %w", err)
		}
	}

	return nil
}

// applyTagUpdates applies tag additions and removals
func (s *TaskService) applyTagUpdates(task *entities.Task, updates *TaskUpdates) {
	for _, tag := range updates.AddTags {
		task.AddTag(tag)
	}

	for _, tag := range updates.RemoveTags {
		task.RemoveTag(tag)
	}
}

// applyMetadataUpdates applies metadata additions and removals
func (s *TaskService) applyMetadataUpdates(task *entities.Task, updates *TaskUpdates) {
	for key, value := range updates.SetMetadata {
		task.SetMetadata(key, value)
	}

	for _, key := range updates.RemoveMetadata {
		task.RemoveMetadata(key)
	}
}

// DeleteTask removes a task
func (s *TaskService) DeleteTask(ctx context.Context, taskID string) error {
	// Verify task exists
	task, err := s.storage.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if err := s.storage.DeleteTask(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	s.logger.Info("task deleted",
		slog.String("task_id", taskID),
		slog.String("content", task.Content))

	return nil
}

// SearchTasks searches for tasks containing the query
func (s *TaskService) SearchTasks(ctx context.Context, query string, filters *ports.TaskFilters) ([]*entities.Task, error) {
	if filters == nil {
		filters = &ports.TaskFilters{}
	}

	// Use current repository if not specified
	if filters.Repository == "" {
		repoInfo, err := s.repository.DetectCurrent(ctx)
		if err == nil {
			filters.Repository = repoInfo.Name
		} else {
			filters.Repository = LocalRepository
		}
	}

	tasks, err := s.storage.SearchTasks(ctx, query, filters)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	s.logger.Debug("tasks searched",
		slog.String("query", query),
		slog.String("repository", filters.Repository),
		slog.Int("results", len(tasks)))

	return tasks, nil
}

// SearchAllRepositories searches for tasks across all repositories
func (s *TaskService) SearchAllRepositories(ctx context.Context, filters *ports.TaskFilters) ([]*entities.Task, error) {
	if filters == nil {
		filters = &ports.TaskFilters{}
	}

	// Clear repository filter to search all repositories
	originalRepo := filters.Repository
	filters.Repository = ""

	// Get all repositories first
	repositories, err := s.storage.ListRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}

	var allTasks []*entities.Task

	// Search each repository
	for _, repo := range repositories {
		filters.Repository = repo
		tasks, err := s.storage.SearchTasks(ctx, filters.Search, filters)
		if err != nil {
			s.logger.Warn("failed to search repository",
				slog.String("repository", repo),
				slog.String("error", err.Error()))
			continue
		}
		allTasks = append(allTasks, tasks...)
	}

	// Restore original repository filter
	filters.Repository = originalRepo

	// Sort by relevance/creation date
	sort.Slice(allTasks, func(i, j int) bool {
		return allTasks[i].CreatedAt.After(allTasks[j].CreatedAt)
	})

	s.logger.Debug("searched all repositories",
		slog.String("query", filters.Search),
		slog.Int("repositories", len(repositories)),
		slog.Int("total_results", len(allTasks)))

	return allTasks, nil
}

// GetRepositoryStats returns statistics for a repository
func (s *TaskService) GetRepositoryStats(ctx context.Context, repository string) (ports.RepositoryStats, error) {
	if repository == "" {
		repoInfo, err := s.repository.DetectCurrent(ctx)
		if err == nil {
			repository = repoInfo.Name
		} else {
			repository = LocalRepository
		}
	}

	return s.storage.GetRepositoryStats(ctx, repository)
}

// ListRepositories returns all repositories with tasks
func (s *TaskService) ListRepositories(ctx context.Context) ([]string, error) {
	return s.storage.ListRepositories(ctx)
}

// TaskUpdates represents the updates that can be applied to a task
type TaskUpdates struct {
	Content        *string
	Priority       *entities.Priority
	DueDate        *time.Time
	EstimatedMins  *int
	ActualMins     *int
	AddTags        []string
	RemoveTags     []string
	SetMetadata    map[string]interface{}
	RemoveMetadata []string
}

// Functional options for task creation

// WithPriority sets task priority
func WithPriority(priority entities.Priority) TaskOption {
	return func(t *entities.Task) {
		t.Priority = priority
	}
}

// WithTags sets task tags
func WithTags(tags ...string) TaskOption {
	return func(t *entities.Task) {
		for _, tag := range tags {
			t.AddTag(tag)
		}
	}
}

// WithEstimatedTime sets estimated time in minutes
func WithEstimatedTime(minutes int) TaskOption {
	return func(t *entities.Task) {
		t.EstimatedMins = minutes
	}
}

// WithSessionID sets session ID
func WithSessionID(sessionID string) TaskOption {
	return func(t *entities.Task) {
		t.SessionID = sessionID
	}
}

// WithParentTask sets parent task ID
func WithParentTask(parentID string) TaskOption {
	return func(t *entities.Task) {
		t.ParentTaskID = parentID
	}
}

// WithAISuggested marks task as AI-suggested
func WithAISuggested() TaskOption {
	return func(t *entities.Task) {
		t.AISuggested = true
	}
}

// WithDueDate sets task due date
func WithDueDate(dueDate time.Time) TaskOption {
	return func(t *entities.Task) {
		t.DueDate = &dueDate
	}
}

// WithMetadata sets task metadata
func WithMetadata(metadata map[string]interface{}) TaskOption {
	return func(t *entities.Task) {
		if t.Metadata == nil {
			t.Metadata = make(map[string]interface{})
		}
		for k, v := range metadata {
			t.Metadata[k] = v
		}
	}
}

// Helper methods

func (s *TaskService) isEmptyFilter(filters *ports.TaskFilters) bool {
	return filters.Search == "" &&
		filters.CreatedAfter == nil &&
		filters.CreatedBefore == nil &&
		filters.UpdatedAfter == nil &&
		filters.UpdatedBefore == nil &&
		filters.DueAfter == nil &&
		filters.DueBefore == nil &&
		filters.CompletedAfter == nil &&
		filters.CompletedBefore == nil &&
		!filters.OverdueOnly &&
		filters.DueSoon == nil &&
		filters.HasDueDate == nil
}

func (s *TaskService) validateStatusTransition(current, newStatus entities.Status) error {
	// Business rules for status transitions
	switch current {
	case entities.StatusCompleted:
		if newStatus != entities.StatusPending {
			return errors.New("completed tasks can only be reopened to pending")
		}
	case entities.StatusCancelled:
		if newStatus != entities.StatusPending {
			return errors.New("cancelled tasks can only be reopened to pending")
		}
	}

	// Additional business rules
	if current == newStatus {
		return fmt.Errorf("task is already in %s status", string(current))
	}

	return nil
}

func (s *TaskService) applyAdvancedFilters(tasks []*entities.Task, filters *ports.TaskFilters) []*entities.Task {
	if filters == nil || s.isEmptyFilter(filters) {
		return tasks
	}

	filtered := make([]*entities.Task, 0, len(tasks))
	for _, task := range tasks {
		if !s.matchesAdvancedFilters(task, filters) {
			continue
		}
		filtered = append(filtered, task)
	}

	return filtered
}

func (s *TaskService) matchesAdvancedFilters(task *entities.Task, filters *ports.TaskFilters) bool {
	if filters == nil {
		return true
	}

	// Search filter
	if filters.Search != "" {
		if !s.taskMatchesSearch(task, filters.Search) {
			return false
		}
	}

	// Creation date filters
	if filters.CreatedAfter != nil {
		if after, err := time.Parse(time.RFC3339, *filters.CreatedAfter); err == nil {
			if task.CreatedAt.Before(after) {
				return false
			}
		}
	}

	if filters.CreatedBefore != nil {
		if before, err := time.Parse(time.RFC3339, *filters.CreatedBefore); err == nil {
			if task.CreatedAt.After(before) {
				return false
			}
		}
	}

	// Update date filters
	if filters.UpdatedAfter != nil {
		if after, err := time.Parse(time.RFC3339, *filters.UpdatedAfter); err == nil {
			if task.UpdatedAt.Before(after) {
				return false
			}
		}
	}

	if filters.UpdatedBefore != nil {
		if before, err := time.Parse(time.RFC3339, *filters.UpdatedBefore); err == nil {
			if task.UpdatedAt.After(before) {
				return false
			}
		}
	}

	// Due date filters
	if filters.DueAfter != nil {
		if after, err := time.Parse(time.RFC3339, *filters.DueAfter); err == nil {
			if task.DueDate == nil || task.DueDate.Before(after) {
				return false
			}
		}
	}

	if filters.DueBefore != nil {
		if before, err := time.Parse(time.RFC3339, *filters.DueBefore); err == nil {
			if task.DueDate == nil || task.DueDate.After(before) {
				return false
			}
		}
	}

	// Completion date filters
	if filters.CompletedAfter != nil {
		if after, err := time.Parse(time.RFC3339, *filters.CompletedAfter); err == nil {
			if task.CompletedAt == nil || task.CompletedAt.Before(after) {
				return false
			}
		}
	}

	if filters.CompletedBefore != nil {
		if before, err := time.Parse(time.RFC3339, *filters.CompletedBefore); err == nil {
			if task.CompletedAt == nil || task.CompletedAt.After(before) {
				return false
			}
		}
	}

	// Overdue filter
	if filters.OverdueOnly && !task.IsOverdue() {
		return false
	}

	// Due soon filter
	if filters.DueSoon != nil {
		duration := time.Duration(*filters.DueSoon) * time.Hour
		if !task.IsDueSoon(duration) {
			return false
		}
	}

	// Has due date filter
	if filters.HasDueDate != nil {
		hasDueDate := task.DueDate != nil
		if *filters.HasDueDate != hasDueDate {
			return false
		}
	}

	return true
}

// taskMatchesSearch checks if a task matches the search criteria
func (s *TaskService) taskMatchesSearch(task *entities.Task, search string) bool {
	searchLower := strings.ToLower(search)
	contentLower := strings.ToLower(task.Content)

	// Check content first
	if strings.Contains(contentLower, searchLower) {
		return true
	}

	// Also search in tags
	for _, tag := range task.Tags {
		if strings.Contains(strings.ToLower(tag), searchLower) {
			return true
		}
	}

	return false
}

func (s *TaskService) sortTasks(tasks []*entities.Task) {
	sort.Slice(tasks, func(i, j int) bool {
		// Sort by priority first (high -> medium -> low)
		iPriority := s.priorityWeight(tasks[i].Priority)
		jPriority := s.priorityWeight(tasks[j].Priority)

		if iPriority != jPriority {
			return iPriority > jPriority
		}

		// Then by status (in_progress -> pending -> completed -> cancelled)
		iStatus := s.statusWeight(tasks[i].Status)
		jStatus := s.statusWeight(tasks[j].Status)

		if iStatus != jStatus {
			return iStatus > jStatus
		}

		// Finally by creation date (newest first)
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})
}

func (s *TaskService) priorityWeight(priority entities.Priority) int {
	switch priority {
	case entities.PriorityHigh:
		return 3
	case entities.PriorityMedium:
		return 2
	case entities.PriorityLow:
		return 1
	default:
		return 0
	}
}

func (s *TaskService) statusWeight(status entities.Status) int {
	switch status {
	case entities.StatusInProgress:
		return 4
	case entities.StatusPending:
		return 3
	case entities.StatusCompleted:
		return 2
	case entities.StatusCancelled:
		return 1
	default:
		return 0
	}
}
