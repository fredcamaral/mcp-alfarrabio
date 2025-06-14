// Package tasks provides task management business logic and orchestration.
package tasks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"lerian-mcp-memory/pkg/types"
)

// User role constants
const (
	UserRoleAdmin = "admin"
)

// Service provides task management business logic
type Service struct {
	repository TaskRepository
	workflow   *WorkflowManager
	auditor    *AuditLogger
	filter     *FilterManager
	config     ServiceConfig
}

// TaskRepository defines the interface for task data access
type TaskRepository interface {
	Create(ctx context.Context, task *types.Task) error
	GetByID(ctx context.Context, id string) (*types.Task, error)
	Update(ctx context.Context, task *types.Task) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filters *TaskFilters) ([]types.Task, error)
	Search(ctx context.Context, query *SearchQuery) (*SearchResults, error)
	BatchUpdate(ctx context.Context, updates []BatchUpdate) error
	GetByIDs(ctx context.Context, ids []string) ([]types.Task, error)
}

// ServiceConfig represents configuration for task service
type ServiceConfig struct {
	MaxTasksPerUser    int           `json:"max_tasks_per_user"`
	DefaultPageSize    int           `json:"default_page_size"`
	MaxPageSize        int           `json:"max_page_size"`
	AuditEnabled       bool          `json:"audit_enabled"`
	WorkflowValidation bool          `json:"workflow_validation"`
	CacheTimeout       time.Duration `json:"cache_timeout"`
}

// DefaultServiceConfig returns default service configuration
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		MaxTasksPerUser:    1000,
		DefaultPageSize:    20,
		MaxPageSize:        100,
		AuditEnabled:       false,
		WorkflowValidation: false,
		CacheTimeout:       5 * time.Minute,
	}
}

// NewService creates a new task service
func NewService(repo TaskRepository, config ServiceConfig) *Service {
	return &Service{
		repository: repo,
		workflow:   NewWorkflowManager(),
		auditor:    NewAuditLogger(),
		filter:     NewFilterManager(),
		config:     config,
	}
}

// CreateTask creates a new task with validation and audit logging
func (s *Service) CreateTask(ctx context.Context, task *types.Task, userID string) error {
	// Validate task
	if err := s.validateTask(task); err != nil {
		return fmt.Errorf("task validation failed: %w", err)
	}

	// Set creation metadata
	now := time.Now()
	if task.ID == "" {
		task.ID = s.generateTaskID()
	}
	task.Timestamps.Created = now
	task.Timestamps.Updated = now
	task.Status = types.TaskStatusLegacyTodo // Default status

	// Set default complexity if not provided
	if task.Complexity.Level == "" {
		task.Complexity.Level = "simple"
	}

	// Validate workflow transition
	if s.config.WorkflowValidation {
		if err := s.workflow.ValidateTransition("", task.Status, userID); err != nil {
			return fmt.Errorf("workflow validation failed: %w", err)
		}
	}

	// Create task
	if err := s.repository.Create(ctx, task); err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	// Audit log
	if s.config.AuditEnabled {
		s.auditor.LogTaskCreated(task.ID, userID, now)
	}

	return nil
}

// GetTask retrieves a task by ID
func (s *Service) GetTask(ctx context.Context, id, userID string) (*types.Task, error) {
	task, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// No access control - all users can access all tasks

	return task, nil
}

// UpdateTask updates an existing task with validation
func (s *Service) UpdateTask(ctx context.Context, task *types.Task, userID string) error {
	// Get existing task
	existing, err := s.repository.GetByID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing task: %w", err)
	}

	// No access control - all users can modify all tasks

	// Validate task
	if err := s.validateTask(task); err != nil {
		return fmt.Errorf("task validation failed: %w", err)
	}

	// Validate workflow transition
	if s.config.WorkflowValidation && existing.Status != task.Status {
		if err := s.workflow.ValidateTransition(existing.Status, task.Status, userID); err != nil {
			return fmt.Errorf("workflow validation failed: %w", err)
		}
	}

	// Update metadata
	task.Timestamps.Updated = time.Now()
	if task.Status != existing.Status {
		s.updateStatusTimestamp(task, existing.Status)
	}

	// Update task
	if err := s.repository.Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Audit log
	if s.config.AuditEnabled {
		s.auditor.LogTaskUpdated(task.ID, userID, s.getChanges(existing, task))
	}

	return nil
}

// DeleteTask deletes a task
func (s *Service) DeleteTask(ctx context.Context, id, userID string) error {
	// Get existing task to verify it exists
	_, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get existing task: %w", err)
	}

	// No access control - all users can delete all tasks

	// Delete task
	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	// Audit log
	if s.config.AuditEnabled {
		s.auditor.LogTaskDeleted(id, userID, time.Now())
	}

	return nil
}

// ListTasks lists tasks with filtering and pagination
func (s *Service) ListTasks(ctx context.Context, filters *TaskFilters, userID string) ([]types.Task, error) {
	// Apply user-specific filters
	modifiedFilters := s.applyUserFilters(filters, userID)

	// Validate and sanitize filters
	if err := s.filter.ValidateFilters(&modifiedFilters); err != nil {
		return nil, fmt.Errorf("invalid filters: %w", err)
	}

	// Apply pagination limits
	if modifiedFilters.Limit == 0 {
		modifiedFilters.Limit = s.config.DefaultPageSize
	}
	if modifiedFilters.Limit > s.config.MaxPageSize {
		modifiedFilters.Limit = s.config.MaxPageSize
	}

	// Get tasks
	tasks, err := s.repository.List(ctx, &modifiedFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Filter tasks user can access
	accessibleTasks := make([]types.Task, 0, len(tasks))
	for i := range tasks {
		if s.canAccessTask(&tasks[i], userID) {
			accessibleTasks = append(accessibleTasks, tasks[i])
		}
	}

	return accessibleTasks, nil
}

// SearchTasks performs full-text search on tasks
func (s *Service) SearchTasks(ctx context.Context, query *SearchQuery, userID string) (*SearchResults, error) {
	// Validate search query
	if err := s.validateSearchQuery(query); err != nil {
		return nil, fmt.Errorf("invalid search query: %w", err)
	}

	// Apply user-specific filters
	query.Filters = s.applyUserFilters(&query.Filters, userID)

	// Perform search
	results, err := s.repository.Search(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Filter results by access permissions
	filteredTasks := make([]types.Task, 0, len(results.Tasks))
	for i := range results.Tasks {
		if s.canAccessTask(&results.Tasks[i], userID) {
			filteredTasks = append(filteredTasks, results.Tasks[i])
		}
	}

	results.Tasks = filteredTasks
	results.TotalResults = len(filteredTasks)

	return results, nil
}

// BatchUpdateTasks performs batch operations on multiple tasks
func (s *Service) BatchUpdateTasks(ctx context.Context, updates []BatchUpdate, userID string) (*BatchResult, error) {
	// Validate batch size
	if len(updates) > 100 { // Reasonable limit
		return nil, fmt.Errorf("batch size too large: %d (max 100)", len(updates))
	}

	result := &BatchResult{
		TotalRequested: len(updates),
		Successful:     make([]string, 0),
		Failed:         make([]BatchError, 0),
	}

	// Validate all updates first
	taskIDs := make([]string, len(updates))
	for i, update := range updates {
		taskIDs[i] = update.TaskID
	}

	// Get all tasks to validate permissions
	tasks, err := s.repository.GetByIDs(ctx, taskIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks for batch update: %w", err)
	}

	// Create task map for quick lookup
	taskMap := make(map[string]*types.Task)
	for i := range tasks {
		taskMap[tasks[i].ID] = &tasks[i]
	}

	// Validate permissions and workflow transitions
	validUpdates := make([]BatchUpdate, 0, len(updates))
	for _, update := range updates {
		task, exists := taskMap[update.TaskID]
		if !exists {
			result.Failed = append(result.Failed, BatchError{
				TaskID: update.TaskID,
				Error:  "task not found",
			})
			continue
		}

		// No access control - all users can modify all tasks

		// Validate workflow transition if status is being changed
		if update.Status != nil && s.config.WorkflowValidation {
			if err := s.workflow.ValidateTransition(task.Status, *update.Status, userID); err != nil {
				result.Failed = append(result.Failed, BatchError{
					TaskID: update.TaskID,
					Error:  fmt.Sprintf("workflow validation failed: %v", err),
				})
				continue
			}
		}

		validUpdates = append(validUpdates, update)
	}

	// Perform batch update
	if len(validUpdates) > 0 {
		if err := s.repository.BatchUpdate(ctx, validUpdates); err != nil {
			return nil, fmt.Errorf("batch update failed: %w", err)
		}

		// Add successful updates
		for _, update := range validUpdates {
			result.Successful = append(result.Successful, update.TaskID)
		}

		// Audit log batch operation
		if s.config.AuditEnabled {
			s.auditor.LogBatchUpdate(userID, len(validUpdates), time.Now())
		}
	}

	result.SuccessfulCount = len(result.Successful)
	result.FailedCount = len(result.Failed)

	return result, nil
}

// Helper methods

func (s *Service) validateTask(task *types.Task) error {
	if task.Title == "" {
		return errors.New("task title is required")
	}
	if task.Type == "" {
		return errors.New("task type is required")
	}
	if task.Priority == "" {
		return errors.New("task priority is required")
	}
	return nil
}

func (s *Service) validateSearchQuery(query *SearchQuery) error {
	if query.Query == "" {
		return errors.New("search query cannot be empty")
	}
	if len(query.Query) < 2 {
		return errors.New("search query too short (minimum 2 characters)")
	}
	if len(query.Query) > 1000 {
		return errors.New("search query too long (maximum 1000 characters)")
	}
	return nil
}

func (s *Service) canAccessTask(task *types.Task, userID string) bool {
	// Basic permission check - in a real system this would be more sophisticated
	return task.Assignee == userID || task.Assignee == "" || userID == UserRoleAdmin
}

func (s *Service) applyUserFilters(filters *TaskFilters, userID string) TaskFilters {
	// In a real system, this would apply user-specific access controls
	if userID != UserRoleAdmin {
		// Non-admin users can only see their assigned tasks or unassigned tasks
		if filters.Assignee == "" {
			filters.Assignee = userID
		}
	}
	return *filters
}

func (s *Service) generateTaskID() string {
	return uuid.New().String()
}

func (s *Service) updateStatusTimestamp(task *types.Task, _ types.TaskStatus) {
	now := time.Now()
	switch task.Status {
	case types.TaskStatusLegacyInProgress:
		task.Timestamps.Started = &now
	case types.TaskStatusLegacyCompleted:
		task.Timestamps.Completed = &now
	}
}

func (s *Service) getChanges(old, updated *types.Task) map[string]interface{} {
	changes := make(map[string]interface{})

	if old.Title != updated.Title {
		changes["title"] = map[string]string{"old": old.Title, "new": updated.Title}
	}
	if old.Status != updated.Status {
		changes["status"] = map[string]string{"old": string(old.Status), "new": string(updated.Status)}
	}
	if old.Priority != updated.Priority {
		changes["priority"] = map[string]string{"old": string(old.Priority), "new": string(updated.Priority)}
	}
	if old.Assignee != updated.Assignee {
		changes["assignee"] = map[string]string{"old": old.Assignee, "new": updated.Assignee}
	}

	return changes
}
