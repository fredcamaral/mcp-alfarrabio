// Package task provides the Task Domain implementation
// for task management, workflows, and productivity operations.
package task

import (
	"context"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/domains"
	"lerian-mcp-memory/internal/types"
)

// Domain implements the TaskDomain interface
// This is the pure task domain without memory management mixing
type Domain struct {
	taskStore       TaskStore
	workflowEngine  WorkflowEngine
	templateEngine  TemplateEngine
	metricsCollector MetricsCollector
	config          *Config
}

// TaskStore defines the interface for task persistence
type TaskStore interface {
	Create(ctx context.Context, task *Task) error
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, projectID types.ProjectID, taskID string) error
	Get(ctx context.Context, projectID types.ProjectID, taskID string) (*Task, error)
	List(ctx context.Context, projectID types.ProjectID, filters *TaskFilters) ([]*Task, error)
	
	// Dependency management
	CreateDependency(ctx context.Context, dependency *TaskDependency) error
	GetDependencies(ctx context.Context, projectID types.ProjectID, taskID string) ([]*TaskDependency, error)
	DeleteDependency(ctx context.Context, dependencyID string) error
	
	// Batch operations
	BatchUpdate(ctx context.Context, updates []*TaskUpdate) error
	BatchDelete(ctx context.Context, projectID types.ProjectID, taskIDs []string) error
}

// WorkflowEngine handles task workflow and state transitions
type WorkflowEngine interface {
	ValidateTransition(from, to string) error
	ApplyTransition(ctx context.Context, task *Task, to string, metadata map[string]interface{}) error
	GetValidTransitions(status string) []string
	GetWorkflowDefinition(workflowType string) (*WorkflowDefinition, error)
}

// TemplateEngine handles task templates and generation
type TemplateEngine interface {
	CreateTemplate(ctx context.Context, template *TaskTemplate) error
	GetTemplate(ctx context.Context, templateID string) (*TaskTemplate, error)
	ApplyTemplate(ctx context.Context, templateID string, variables map[string]interface{}) ([]*Task, error)
	ListTemplates(ctx context.Context, projectID types.ProjectID) ([]*TaskTemplate, error)
}

// MetricsCollector handles task analytics and metrics
type MetricsCollector interface {
	RecordTaskEvent(ctx context.Context, event *TaskEvent) error
	GetTaskMetrics(ctx context.Context, projectID types.ProjectID, filters *MetricsFilters) (*TaskMetrics, error)
	AnalyzePerformance(ctx context.Context, projectID types.ProjectID, timeRange *TimeRange) (*PerformanceAnalysis, error)
}

// Config represents configuration for the task domain
type Config struct {
	MaxTasksPerProject   int           `json:"max_tasks_per_project"`
	MaxSubtaskDepth      int           `json:"max_subtask_depth"`
	DefaultWorkflow      string        `json:"default_workflow"`
	AutoAssignEnabled    bool          `json:"auto_assign_enabled"`
	NotificationsEnabled bool          `json:"notifications_enabled"`
	MetricsEnabled       bool          `json:"metrics_enabled"`
	TemplateValidation   bool          `json:"template_validation"`
	CacheTimeout         time.Duration `json:"cache_timeout"`
}

// DefaultConfig returns default configuration for task domain
func DefaultConfig() *Config {
	return &Config{
		MaxTasksPerProject:   10000,
		MaxSubtaskDepth:      5,
		DefaultWorkflow:      "standard",
		AutoAssignEnabled:    true,
		NotificationsEnabled: true,
		MetricsEnabled:       true,
		TemplateValidation:   true,
		CacheTimeout:         15 * time.Minute,
	}
}

// Task represents a task entity in the task domain
type Task struct {
	ID          string                 `json:"id"`
	ProjectID   types.ProjectID        `json:"project_id"`
	SessionID   types.SessionID        `json:"session_id,omitempty"`
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Status      TaskStatus             `json:"status"`
	Priority    TaskPriority           `json:"priority"`
	Type        TaskType               `json:"type,omitempty"`
	
	// Assignment and ownership
	AssigneeID  string                 `json:"assignee_id,omitempty"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	ReviewerID  string                 `json:"reviewer_id,omitempty"`
	
	// Hierarchy
	ParentID    string                 `json:"parent_id,omitempty"`
	SubtaskIDs  []string               `json:"subtask_ids,omitempty"`
	
	// Timing
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DueDate     *time.Time             `json:"due_date,omitempty"`
	StartDate   *time.Time             `json:"start_date,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	
	// Effort tracking
	EstimatedMins int                  `json:"estimated_mins,omitempty"`
	ActualMins    int                  `json:"actual_mins,omitempty"`
	
	// Organization
	Tags        []string               `json:"tags,omitempty"`
	Labels      []string               `json:"labels,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	
	// Content references (but not mixing domains)
	LinkedContentIDs []string           `json:"linked_content_ids,omitempty"` // References only
	
	// Version and audit
	Version     int                    `json:"version"`
	Workflow    string                 `json:"workflow,omitempty"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskStatusBacklog    TaskStatus = "backlog"
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusInReview   TaskStatus = "in_review"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusCancelled  TaskStatus = "cancelled"
	TaskStatusArchived   TaskStatus = "archived"
)

// TaskPriority represents the priority of a task
type TaskPriority string

const (
	TaskPriorityLow      TaskPriority = "low"
	TaskPriorityMedium   TaskPriority = "medium"
	TaskPriorityHigh     TaskPriority = "high"
	TaskPriorityCritical TaskPriority = "critical"
)

// TaskType represents the type/category of a task
type TaskType string

const (
	TaskTypeBug         TaskType = "bug"
	TaskTypeFeature     TaskType = "feature"
	TaskTypeImprovement TaskType = "improvement"
	TaskTypeResearch    TaskType = "research"
	TaskTypeDocumentation TaskType = "documentation"
	TaskTypeMaintenance TaskType = "maintenance"
)

// Supporting types

type TaskFilters struct {
	Status       []TaskStatus     `json:"status,omitempty"`
	Priority     []TaskPriority   `json:"priority,omitempty"`
	Type         []TaskType       `json:"type,omitempty"`
	AssigneeIDs  []string         `json:"assignee_ids,omitempty"`
	Tags         []string         `json:"tags,omitempty"`
	DueBefore    *time.Time       `json:"due_before,omitempty"`
	DueAfter     *time.Time       `json:"due_after,omitempty"`
	CreatedAfter *time.Time       `json:"created_after,omitempty"`
	CreatedBefore *time.Time      `json:"created_before,omitempty"`
	HasSubtasks  *bool            `json:"has_subtasks,omitempty"`
	ParentID     string           `json:"parent_id,omitempty"`
}

type TaskDependency struct {
	ID           string          `json:"id"`
	ProjectID    types.ProjectID `json:"project_id"`
	TaskID       string          `json:"task_id"`
	DependsOnID  string          `json:"depends_on_id"`
	Type         DependencyType  `json:"type"`
	CreatedAt    time.Time       `json:"created_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type DependencyType string

const (
	DependencyTypeBlockedBy    DependencyType = "blocked_by"
	DependencyTypeSubtaskOf    DependencyType = "subtask_of"
	DependencyTypeRelatedTo    DependencyType = "related_to"
	DependencyTypeDuplicateOf  DependencyType = "duplicate_of"
)

type TaskUpdate struct {
	TaskID    string                 `json:"task_id"`
	Fields    map[string]interface{} `json:"fields"`
	UpdatedBy string                 `json:"updated_by"`
	Reason    string                 `json:"reason,omitempty"`
}

type WorkflowDefinition struct {
	Name        string                  `json:"name"`
	States      []string                `json:"states"`
	Transitions map[string][]string     `json:"transitions"`
	Rules       map[string]WorkflowRule `json:"rules"`
}

type WorkflowRule struct {
	RequiredFields []string               `json:"required_fields,omitempty"`
	Conditions     map[string]interface{} `json:"conditions,omitempty"`
	Actions        []string               `json:"actions,omitempty"`
}

type TaskTemplate struct {
	ID          string                 `json:"id"`
	ProjectID   types.ProjectID        `json:"project_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Category    string                 `json:"category,omitempty"`
	Template    map[string]interface{} `json:"template"`
	Variables   []TemplateVariable     `json:"variables,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	CreatedBy   string                 `json:"created_by"`
}

type TemplateVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Description  string      `json:"description,omitempty"`
}

type TaskEvent struct {
	ID        string                 `json:"id"`
	ProjectID types.ProjectID        `json:"project_id"`
	TaskID    string                 `json:"task_id"`
	EventType string                 `json:"event_type"`
	UserID    string                 `json:"user_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type MetricsFilters struct {
	TimeRange   *TimeRange     `json:"time_range,omitempty"`
	UserIDs     []string       `json:"user_ids,omitempty"`
	TaskTypes   []TaskType     `json:"task_types,omitempty"`
	Priorities  []TaskPriority `json:"priorities,omitempty"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type TaskMetrics struct {
	TotalTasks        int                        `json:"total_tasks"`
	TasksByStatus     map[TaskStatus]int         `json:"tasks_by_status"`
	TasksByPriority   map[TaskPriority]int       `json:"tasks_by_priority"`
	TasksByType       map[TaskType]int           `json:"tasks_by_type"`
	TasksByAssignee   map[string]int             `json:"tasks_by_assignee"`
	CompletionRate    float64                    `json:"completion_rate"`
	AverageLeadTime   time.Duration              `json:"average_lead_time"`
	AverageCycleTime  time.Duration              `json:"average_cycle_time"`
	Throughput        float64                    `json:"throughput"`
	OverdueTasks      int                        `json:"overdue_tasks"`
}

type PerformanceAnalysis struct {
	Period            TimeRange                  `json:"period"`
	Velocity          float64                    `json:"velocity"`
	Burndown          []BurndownPoint            `json:"burndown"`
	CycleTimeAnalysis CycleTimeAnalysis          `json:"cycle_time_analysis"`
	BottleneckAnalysis BottleneckAnalysis        `json:"bottleneck_analysis"`
	Trends            map[string]TrendAnalysis   `json:"trends"`
}

type BurndownPoint struct {
	Date      time.Time `json:"date"`
	Planned   int       `json:"planned"`
	Actual    int       `json:"actual"`
	Remaining int       `json:"remaining"`
}

type CycleTimeAnalysis struct {
	Average    time.Duration              `json:"average"`
	Median     time.Duration              `json:"median"`
	P95        time.Duration              `json:"p95"`
	ByStatus   map[TaskStatus]time.Duration `json:"by_status"`
	ByPriority map[TaskPriority]time.Duration `json:"by_priority"`
}

type BottleneckAnalysis struct {
	StatusBottlenecks    map[TaskStatus]int    `json:"status_bottlenecks"`
	AssigneeBottlenecks  map[string]int        `json:"assignee_bottlenecks"`
	PriorityBottlenecks  map[TaskPriority]int  `json:"priority_bottlenecks"`
	Recommendations      []string              `json:"recommendations"`
}

type TrendAnalysis struct {
	Direction  string  `json:"direction"` // "increasing", "decreasing", "stable"
	Magnitude  float64 `json:"magnitude"`
	Confidence float64 `json:"confidence"`
}

// NewDomain creates a new task domain instance
func NewDomain(
	taskStore TaskStore,
	workflowEngine WorkflowEngine,
	templateEngine TemplateEngine,
	metricsCollector MetricsCollector,
	config *Config,
) *Domain {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &Domain{
		taskStore:        taskStore,
		workflowEngine:   workflowEngine,
		templateEngine:   templateEngine,
		metricsCollector: metricsCollector,
		config:           config,
	}
}

// Task Management Operations

// CreateTask creates a new task
func (d *Domain) CreateTask(ctx context.Context, req *domains.CreateTaskRequest) (*domains.CreateTaskResponse, error) {
	startTime := time.Now()
	
	// Create task entity
	task := &Task{
		ID:          generateTaskID(),
		ProjectID:   req.ProjectID,
		SessionID:   req.SessionID,
		Title:       req.Task.Title,
		Description: req.Task.Description,
		Status:      TaskStatus(req.Task.Status),
		Priority:    TaskPriority(req.Task.Priority),
		AssigneeID:  req.Task.AssigneeID,
		DueDate:     req.Task.DueDate,
		EstimatedMins: req.Task.EstimatedMins,
		Tags:        req.Task.Tags,
		Metadata:    req.Task.Metadata,
		LinkedContentIDs: req.Task.LinkedContent,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     1,
		Workflow:    d.config.DefaultWorkflow,
	}
	
	// Set defaults
	if task.Status == "" {
		task.Status = TaskStatusTodo
	}
	if task.Priority == "" {
		task.Priority = TaskPriorityMedium
	}
	
	// Validate task
	if err := d.validateTask(task); err != nil {
		return nil, fmt.Errorf("task validation failed: %w", err)
	}
	
	// Store task
	if err := d.taskStore.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	
	// Record metrics
	if d.config.MetricsEnabled {
		event := &TaskEvent{
			ID:        generateEventID(),
			ProjectID: task.ProjectID,
			TaskID:    task.ID,
			EventType: "task_created",
			UserID:    req.UserID,
			Timestamp: time.Now(),
		}
		_ = d.metricsCollector.RecordTaskEvent(ctx, event)
	}
	
	return &domains.CreateTaskResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Message:   "Task created successfully",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		TaskID:    task.ID,
		CreatedAt: task.CreatedAt,
	}, nil
}

// UpdateTask updates an existing task
func (d *Domain) UpdateTask(ctx context.Context, req *domains.UpdateTaskRequest) (*domains.UpdateTaskResponse, error) {
	startTime := time.Now()
	
	// Get existing task
	task, err := d.taskStore.Get(ctx, req.ProjectID, req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}
	
	// Apply updates
	if req.Updates.Title != nil {
		task.Title = *req.Updates.Title
	}
	if req.Updates.Description != nil {
		task.Description = *req.Updates.Description
	}
	if req.Updates.Status != nil {
		newStatus := TaskStatus(*req.Updates.Status)
		
		// Validate transition
		if err := d.workflowEngine.ValidateTransition(string(task.Status), string(newStatus)); err != nil {
			return nil, fmt.Errorf("invalid status transition: %w", err)
		}
		
		task.Status = newStatus
		
		// Set completion time if completed
		if newStatus == TaskStatusCompleted && task.CompletedAt == nil {
			now := time.Now()
			task.CompletedAt = &now
		}
	}
	if req.Updates.Priority != nil {
		task.Priority = TaskPriority(*req.Updates.Priority)
	}
	if req.Updates.AssigneeID != nil {
		task.AssigneeID = *req.Updates.AssigneeID
	}
	if req.Updates.DueDate != nil {
		task.DueDate = req.Updates.DueDate
	}
	if req.Updates.EstimatedMins != nil {
		task.EstimatedMins = *req.Updates.EstimatedMins
	}
	if req.Updates.ActualMins != nil {
		task.ActualMins = *req.Updates.ActualMins
	}
	if req.Updates.Tags != nil {
		task.Tags = req.Updates.Tags
	}
	if req.Updates.Metadata != nil {
		if task.Metadata == nil {
			task.Metadata = make(map[string]interface{})
		}
		for k, v := range req.Updates.Metadata {
			task.Metadata[k] = v
		}
	}
	
	// Update version and timestamp
	task.Version++
	task.UpdatedAt = time.Now()
	
	// Validate updated task
	if err := d.validateTask(task); err != nil {
		return nil, fmt.Errorf("task validation failed: %w", err)
	}
	
	// Update task
	if err := d.taskStore.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}
	
	// Record metrics
	if d.config.MetricsEnabled {
		event := &TaskEvent{
			ID:        generateEventID(),
			ProjectID: task.ProjectID,
			TaskID:    task.ID,
			EventType: "task_updated",
			UserID:    req.UserID,
			Timestamp: time.Now(),
		}
		_ = d.metricsCollector.RecordTaskEvent(ctx, event)
	}
	
	return &domains.UpdateTaskResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Message:   "Task updated successfully",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		TaskID:    task.ID,
		Version:   task.Version,
		UpdatedAt: task.UpdatedAt,
	}, nil
}

// DeleteTask removes a task
func (d *Domain) DeleteTask(ctx context.Context, req *domains.DeleteTaskRequest) error {
	// Record metrics before deletion
	if d.config.MetricsEnabled {
		event := &TaskEvent{
			ID:        generateEventID(),
			ProjectID: req.ProjectID,
			TaskID:    req.TaskID,
			EventType: "task_deleted",
			UserID:    req.UserID,
			Timestamp: time.Now(),
		}
		_ = d.metricsCollector.RecordTaskEvent(ctx, event)
	}
	
	return d.taskStore.Delete(ctx, req.ProjectID, req.TaskID)
}

// GetTask retrieves a task by ID
func (d *Domain) GetTask(ctx context.Context, req *domains.GetTaskRequest) (*domains.GetTaskResponse, error) {
	startTime := time.Now()
	
	// Get task
	task, err := d.taskStore.Get(ctx, req.ProjectID, req.TaskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}
	
	response := &domains.GetTaskResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Task: task,
	}
	
	// Include dependencies if requested
	if req.Options != nil && req.Options.IncludeDependencies {
		dependencies, err := d.taskStore.GetDependencies(ctx, req.ProjectID, req.TaskID)
		if err == nil {
			response.Dependencies = make([]interface{}, len(dependencies))
			for i, dep := range dependencies {
				response.Dependencies[i] = dep
			}
		}
	}
	
	return response, nil
}

// ListTasks retrieves tasks with filters
func (d *Domain) ListTasks(ctx context.Context, req *domains.ListTasksRequest) (*domains.ListTasksResponse, error) {
	startTime := time.Now()
	
	// Convert filters
	filters := &TaskFilters{}
	if req.Filters != nil {
		// Convert domain filters to task filters
		// This keeps the domains separate while allowing filter translation
		if len(req.Filters.Status) > 0 {
			filters.Status = make([]TaskStatus, len(req.Filters.Status))
			for i, s := range req.Filters.Status {
				filters.Status[i] = TaskStatus(s)
			}
		}
		
		if len(req.Filters.Priority) > 0 {
			filters.Priority = make([]TaskPriority, len(req.Filters.Priority))
			for i, p := range req.Filters.Priority {
				filters.Priority[i] = TaskPriority(p)
			}
		}
		
		filters.AssigneeIDs = req.Filters.AssigneeIDs
		filters.Tags = req.Filters.Tags
		filters.DueBefore = req.Filters.DueBefore
		filters.DueAfter = req.Filters.DueAfter
		filters.CreatedAfter = req.Filters.CreatedAfter
		filters.CreatedBefore = req.Filters.CreatedBefore
		filters.HasSubtasks = req.Filters.HasSubtasks
	}
	
	// Get tasks
	tasks, err := d.taskStore.List(ctx, req.ProjectID, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	
	// Convert to interface{} for response
	taskInterfaces := make([]interface{}, len(tasks))
	for i, task := range tasks {
		taskInterfaces[i] = task
	}
	
	return &domains.ListTasksResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Tasks: taskInterfaces,
		Total: len(tasks),
	}, nil
}

// Placeholder implementations for remaining methods
func (d *Domain) TransitionTask(ctx context.Context, req *domains.TransitionTaskRequest) (*domains.TransitionTaskResponse, error) {
	return &domains.TransitionTaskResponse{}, fmt.Errorf("not yet implemented")
}

func (d *Domain) AssignTask(ctx context.Context, req *domains.AssignTaskRequest) (*domains.AssignTaskResponse, error) {
	return &domains.AssignTaskResponse{}, fmt.Errorf("not yet implemented")
}

func (d *Domain) CompleteTask(ctx context.Context, req *domains.CompleteTaskRequest) (*domains.CompleteTaskResponse, error) {
	return &domains.CompleteTaskResponse{}, fmt.Errorf("not yet implemented")
}

func (d *Domain) GetTaskMetrics(ctx context.Context, req *domains.GetTaskMetricsRequest) (*domains.GetTaskMetricsResponse, error) {
	return &domains.GetTaskMetricsResponse{}, fmt.Errorf("not yet implemented")
}

func (d *Domain) AnalyzeTaskPerformance(ctx context.Context, req *domains.AnalyzeTaskPerformanceRequest) (*domains.AnalyzeTaskPerformanceResponse, error) {
	return &domains.AnalyzeTaskPerformanceResponse{}, fmt.Errorf("not yet implemented")
}

func (d *Domain) CreateDependency(ctx context.Context, req *domains.CreateDependencyRequest) (*domains.CreateDependencyResponse, error) {
	return &domains.CreateDependencyResponse{}, fmt.Errorf("not yet implemented")
}

func (d *Domain) GetDependencies(ctx context.Context, req *domains.GetDependenciesRequest) (*domains.GetDependenciesResponse, error) {
	return &domains.GetDependenciesResponse{}, fmt.Errorf("not yet implemented")
}

func (d *Domain) CreateTemplate(ctx context.Context, req *domains.CreateTemplateRequest) (*domains.CreateTemplateResponse, error) {
	return &domains.CreateTemplateResponse{}, fmt.Errorf("not yet implemented")
}

func (d *Domain) ApplyTemplate(ctx context.Context, req *domains.ApplyTemplateRequest) (*domains.ApplyTemplateResponse, error) {
	return &domains.ApplyTemplateResponse{}, fmt.Errorf("not yet implemented")
}

// Helper functions

func (d *Domain) validateTask(task *Task) error {
	if task.Title == "" {
		return fmt.Errorf("task title is required")
	}
	
	if task.ProjectID == "" {
		return fmt.Errorf("project ID is required")
	}
	
	// Validate status
	validStatuses := []TaskStatus{
		TaskStatusBacklog, TaskStatusTodo, TaskStatusInProgress,
		TaskStatusInReview, TaskStatusBlocked, TaskStatusCompleted,
		TaskStatusCancelled, TaskStatusArchived,
	}
	
	valid := false
	for _, status := range validStatuses {
		if task.Status == status {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid task status: %s", task.Status)
	}
	
	return nil
}

func generateTaskID() string {
	// TODO: Implement proper ID generation
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}

func generateEventID() string {
	// TODO: Implement proper ID generation
	return fmt.Sprintf("event_%d", time.Now().UnixNano())
}