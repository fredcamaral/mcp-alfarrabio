// Package handlers provides HTTP handlers for task CRUD operations.
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"lerian-mcp-memory/internal/api/response"
	"lerian-mcp-memory/internal/tasks"
	"lerian-mcp-memory/pkg/types"
)

// TaskCRUDHandler handles task CRUD operations
type TaskCRUDHandler struct {
	service *tasks.Service
	config  TaskCRUDConfig
}

// TaskCRUDConfig represents configuration for task CRUD operations
type TaskCRUDConfig struct {
	MaxTitleLength       int           `json:"max_title_length"`
	MaxDescriptionLength int           `json:"max_description_length"`
	DefaultPageSize      int           `json:"default_page_size"`
	MaxPageSize          int           `json:"max_page_size"`
	RequestTimeout       time.Duration `json:"request_timeout"`
	EnableValidation     bool          `json:"enable_validation"`
	EnableAuditLog       bool          `json:"enable_audit_log"`
}

// DefaultTaskCRUDConfig returns default configuration
func DefaultTaskCRUDConfig() TaskCRUDConfig {
	return TaskCRUDConfig{
		MaxTitleLength:       200,
		MaxDescriptionLength: 5000,
		DefaultPageSize:      20,
		MaxPageSize:          100,
		RequestTimeout:       30 * time.Second,
		EnableValidation:     true,
		EnableAuditLog:       true,
	}
}

// NewTaskCRUDHandler creates a new task CRUD handler
func NewTaskCRUDHandler(service *tasks.Service, config TaskCRUDConfig) *TaskCRUDHandler {
	return &TaskCRUDHandler{
		service: service,
		config:  config,
	}
}

// CreateTask handles task creation
func (h *TaskCRUDHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Validate request
	if err := h.validateCreateRequest(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Convert request to task
	task := h.requestToTask(&req)

	// Get user ID from context (in real app this would come from auth middleware)
	userID := h.getUserID(r)

	// Create task
	if err := h.service.CreateTask(r.Context(), &task, userID); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to create task", err.Error())
		return
	}

	// Return created task
	response.WriteSuccess(w, TaskResponse{
		Task:      task,
		Message:   "Task created successfully",
		CreatedAt: time.Now(),
	})
}

// GetTask handles task retrieval by ID
func (h *TaskCRUDHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		response.WriteError(w, http.StatusBadRequest, "Task ID is required", "")
		return
	}

	// Get user ID from context
	userID := h.getUserID(r)

	// Get task
	task, err := h.service.GetTask(r.Context(), taskID, userID)
	if err != nil {
		if isNotFoundError(err) {
			response.WriteError(w, http.StatusNotFound, "Task not found", err.Error())
		} else {
			response.WriteError(w, http.StatusInternalServerError, "Failed to get task", err.Error())
		}
		return
	}

	response.WriteSuccess(w, TaskResponse{
		Task:      *task,
		Message:   "Task retrieved successfully",
		CreatedAt: time.Now(),
	})
}

// UpdateTask handles task updates
func (h *TaskCRUDHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		response.WriteError(w, http.StatusBadRequest, "Task ID is required", "")
		return
	}

	// Parse request body
	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Validate request
	if err := h.validateUpdateRequest(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Validation failed", err.Error())
		return
	}

	// Get user ID from context
	userID := h.getUserID(r)

	// Get existing task first
	existingTask, err := h.service.GetTask(r.Context(), taskID, userID)
	if err != nil {
		if isNotFoundError(err) {
			response.WriteError(w, http.StatusNotFound, "Task not found", err.Error())
		} else {
			response.WriteError(w, http.StatusInternalServerError, "Failed to get task", err.Error())
		}
		return
	}

	// Apply updates to task
	updatedTask := h.applyUpdates(existingTask, &req)
	updatedTask.ID = taskID // Ensure ID matches URL parameter

	// Update task
	if err := h.service.UpdateTask(r.Context(), &updatedTask, userID); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to update task", err.Error())
		return
	}

	response.WriteSuccess(w, TaskResponse{
		Task:      updatedTask,
		Message:   "Task updated successfully",
		CreatedAt: time.Now(),
	})
}

// DeleteTask handles task deletion
func (h *TaskCRUDHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		response.WriteError(w, http.StatusBadRequest, "Task ID is required", "")
		return
	}

	// Get user ID from context
	userID := h.getUserID(r)

	// Delete task
	if err := h.service.DeleteTask(r.Context(), taskID, userID); err != nil {
		if isNotFoundError(err) {
			response.WriteError(w, http.StatusNotFound, "Task not found", err.Error())
		} else {
			response.WriteError(w, http.StatusInternalServerError, "Failed to delete task", err.Error())
		}
		return
	}

	response.WriteSuccess(w, map[string]interface{}{
		"message":    "Task deleted successfully",
		"task_id":    taskID,
		"deleted_at": time.Now(),
	})
}

// ListTasks handles task listing with filtering and pagination
func (h *TaskCRUDHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filters := h.parseFilters(r)

	// Get user ID from context
	userID := h.getUserID(r)

	// List tasks
	taskList, err := h.service.ListTasks(r.Context(), &filters, userID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to list tasks", err.Error())
		return
	}

	// Build response
	response.WriteSuccess(w, TaskListResponse{
		Tasks:       taskList,
		TotalCount:  len(taskList),
		Filters:     filters,
		Message:     "Tasks retrieved successfully",
		RetrievedAt: time.Now(),
	})
}

// GetTaskMetrics handles task metrics and statistics
func (h *TaskCRUDHandler) GetTaskMetrics(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID := h.getUserID(r)

	// Get all tasks for the user (could be optimized with dedicated metrics query)
	filters := tasks.TaskFilters{Limit: 1000} // Large limit for metrics
	taskList, err := h.service.ListTasks(r.Context(), &filters, userID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to get tasks for metrics", err.Error())
		return
	}

	// Calculate metrics
	metrics := h.calculateMetrics(taskList)

	response.WriteSuccess(w, TaskMetricsResponse{
		Metrics:     metrics,
		TotalTasks:  len(taskList),
		GeneratedAt: time.Now(),
	})
}

// Helper methods

func (h *TaskCRUDHandler) validateCreateRequest(req *CreateTaskRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(req.Title) > h.config.MaxTitleLength {
		return fmt.Errorf("title too long (max %d characters)", h.config.MaxTitleLength)
	}
	if len(req.Description) > h.config.MaxDescriptionLength {
		return fmt.Errorf("description too long (max %d characters)", h.config.MaxDescriptionLength)
	}
	if req.Type == "" {
		return fmt.Errorf("type is required")
	}
	if req.Priority == "" {
		return fmt.Errorf("priority is required")
	}
	return nil
}

func (h *TaskCRUDHandler) validateUpdateRequest(req *UpdateTaskRequest) error {
	if req.Title != nil && *req.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if req.Title != nil && len(*req.Title) > h.config.MaxTitleLength {
		return fmt.Errorf("title too long (max %d characters)", h.config.MaxTitleLength)
	}
	if req.Description != nil && len(*req.Description) > h.config.MaxDescriptionLength {
		return fmt.Errorf("description too long (max %d characters)", h.config.MaxDescriptionLength)
	}
	return nil
}

func (h *TaskCRUDHandler) requestToTask(req *CreateTaskRequest) types.Task {
	now := time.Now()

	task := types.Task{
		Title:              req.Title,
		Description:        req.Description,
		Type:               req.Type,
		Priority:           req.Priority,
		Status:             types.TaskStatusLegacyTodo, // Default status
		AcceptanceCriteria: req.AcceptanceCriteria,
		Dependencies:       req.Dependencies,
		Tags:               req.Tags,
		Assignee:           req.Assignee,
		DueDate:            req.DueDate,
		SourcePRDID:        req.SourcePRDID,
		Timestamps: types.TaskTimestamps{
			Created: now,
			Updated: now,
		},
		Metadata: types.TaskMetadata{
			GenerationSource: "user_created",
			ExtendedData:     make(map[string]interface{}),
		},
	}

	// Set repository and branch if provided
	if req.Repository != "" {
		task.Metadata.ExtendedData["repository"] = req.Repository
	}
	if req.Branch != "" {
		task.Metadata.ExtendedData["branch"] = req.Branch
	}

	return task
}

func (h *TaskCRUDHandler) applyUpdates(existing *types.Task, req *UpdateTaskRequest) types.Task {
	updated := *existing // Copy existing task

	// Apply updates only for non-nil fields
	if req.Title != nil {
		updated.Title = *req.Title
	}
	if req.Description != nil {
		updated.Description = *req.Description
	}
	if req.Type != nil {
		updated.Type = *req.Type
	}
	if req.Priority != nil {
		updated.Priority = *req.Priority
	}
	if req.Status != nil {
		updated.Status = *req.Status
	}
	if req.Assignee != nil {
		updated.Assignee = *req.Assignee
	}
	if req.DueDate != nil {
		updated.DueDate = req.DueDate
	}
	if req.AcceptanceCriteria != nil {
		updated.AcceptanceCriteria = req.AcceptanceCriteria
	}
	if req.Dependencies != nil {
		updated.Dependencies = req.Dependencies
	}
	if req.Tags != nil {
		updated.Tags = req.Tags
	}

	// Always update the timestamp
	updated.Timestamps.Updated = time.Now()

	return updated
}

func (h *TaskCRUDHandler) parseFilters(r *http.Request) tasks.TaskFilters {
	filters := tasks.TaskFilters{}

	// Parse status filter
	if statusParam := r.URL.Query().Get("status"); statusParam != "" {
		filters.Status = []types.TaskStatus{types.TaskStatus(statusParam)}
	}

	// Parse type filter
	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		filters.Type = []types.TaskType{types.TaskType(typeParam)}
	}

	// Parse priority filter
	if priorityParam := r.URL.Query().Get("priority"); priorityParam != "" {
		filters.Priority = []types.TaskPriority{types.TaskPriority(priorityParam)}
	}

	// Parse assignee filter
	filters.Assignee = r.URL.Query().Get("assignee")

	// Parse repository filter
	filters.Repository = r.URL.Query().Get("repository")

	// Parse text search
	filters.TextSearch = r.URL.Query().Get("search")

	// Parse pagination
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if limit, err := strconv.Atoi(limitParam); err == nil && limit > 0 {
			if limit > h.config.MaxPageSize {
				limit = h.config.MaxPageSize
			}
			filters.Limit = limit
		}
	}
	if filters.Limit == 0 {
		filters.Limit = h.config.DefaultPageSize
	}

	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if offset, err := strconv.Atoi(offsetParam); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Parse sort
	if sortParam := r.URL.Query().Get("sort"); sortParam != "" {
		order := tasks.SortOrderAsc
		if orderParam := r.URL.Query().Get("order"); orderParam == "desc" {
			order = tasks.SortOrderDesc
		}
		filters.SortBy = []tasks.SortField{
			{Field: sortParam, Order: order},
		}
	}

	return filters
}

func (h *TaskCRUDHandler) calculateMetrics(taskList []types.Task) TaskMetrics {
	metrics := TaskMetrics{
		StatusCounts:     make(map[string]int),
		TypeCounts:       make(map[string]int),
		PriorityCounts:   make(map[string]int),
		ComplexityCounts: make(map[string]int),
	}

	totalQualityScore := 0.0
	totalEstimatedHours := 0.0

	for i := range taskList {
		task := &taskList[i]
		// Count by status
		metrics.StatusCounts[string(task.Status)]++

		// Count by type
		metrics.TypeCounts[string(task.Type)]++

		// Count by priority
		metrics.PriorityCounts[string(task.Priority)]++

		// Count by complexity
		if task.Complexity.Level != "" {
			metrics.ComplexityCounts[string(task.Complexity.Level)]++
		}

		// Accumulate scores
		totalQualityScore += task.QualityScore.OverallScore
		totalEstimatedHours += task.EstimatedEffort.Hours
	}

	// Calculate averages
	if len(taskList) > 0 {
		metrics.AverageQualityScore = totalQualityScore / float64(len(taskList))
		metrics.AverageEstimatedHours = totalEstimatedHours / float64(len(taskList))
	}

	return metrics
}

func (h *TaskCRUDHandler) getUserID(r *http.Request) string {
	// In a real application, this would extract user ID from JWT token or session
	// For now, return a default user ID
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID
	}
	return DefaultUserID
}

func isNotFoundError(err error) bool {
	return err != nil && (err.Error() == "task not found" ||
		err.Error() == "access denied" ||
		strings.Contains(err.Error(), "not found"))
}

// Request/Response types

// CreateTaskRequest represents a task creation request
type CreateTaskRequest struct {
	Title              string             `json:"title"`
	Description        string             `json:"description"`
	Type               types.TaskType     `json:"type"`
	Priority           types.TaskPriority `json:"priority"`
	AcceptanceCriteria []string           `json:"acceptance_criteria,omitempty"`
	Dependencies       []string           `json:"dependencies,omitempty"`
	Tags               []string           `json:"tags,omitempty"`
	Assignee           string             `json:"assignee,omitempty"`
	DueDate            *time.Time         `json:"due_date,omitempty"`
	SourcePRDID        string             `json:"source_prd_id,omitempty"`
	Repository         string             `json:"repository,omitempty"`
	Branch             string             `json:"branch,omitempty"`
}

// UpdateTaskRequest represents a task update request
type UpdateTaskRequest struct {
	Title              *string             `json:"title,omitempty"`
	Description        *string             `json:"description,omitempty"`
	Type               *types.TaskType     `json:"type,omitempty"`
	Priority           *types.TaskPriority `json:"priority,omitempty"`
	Status             *types.TaskStatus   `json:"status,omitempty"`
	AcceptanceCriteria []string            `json:"acceptance_criteria,omitempty"`
	Dependencies       []string            `json:"dependencies,omitempty"`
	Tags               []string            `json:"tags,omitempty"`
	Assignee           *string             `json:"assignee,omitempty"`
	DueDate            *time.Time          `json:"due_date,omitempty"`
}

// TaskResponse represents a single task response
type TaskResponse struct {
	Task      types.Task `json:"task"`
	Message   string     `json:"message"`
	CreatedAt time.Time  `json:"created_at"`
}

// TaskListResponse represents a task list response
type TaskListResponse struct {
	Tasks       []types.Task      `json:"tasks"`
	TotalCount  int               `json:"total_count"`
	Filters     tasks.TaskFilters `json:"filters"`
	Message     string            `json:"message"`
	RetrievedAt time.Time         `json:"retrieved_at"`
}

// TaskMetricsResponse represents task metrics response
type TaskMetricsResponse struct {
	Metrics     TaskMetrics `json:"metrics"`
	TotalTasks  int         `json:"total_tasks"`
	GeneratedAt time.Time   `json:"generated_at"`
}

// TaskMetrics represents task statistics
type TaskMetrics struct {
	StatusCounts          map[string]int `json:"status_counts"`
	TypeCounts            map[string]int `json:"type_counts"`
	PriorityCounts        map[string]int `json:"priority_counts"`
	ComplexityCounts      map[string]int `json:"complexity_counts"`
	AverageQualityScore   float64        `json:"average_quality_score"`
	AverageEstimatedHours float64        `json:"average_estimated_hours"`
}
