// Package handlers provides HTTP handlers for task batch operations.
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"lerian-mcp-memory/internal/api/response"
	"lerian-mcp-memory/internal/tasks"
	"lerian-mcp-memory/pkg/types"
)

// TaskBatchHandler handles batch task operations
type TaskBatchHandler struct {
	service *tasks.Service
	config  TaskBatchConfig
}

// TaskBatchConfig represents configuration for batch operations
type TaskBatchConfig struct {
	MaxBatchSize        int           `json:"max_batch_size"`
	RequestTimeout      time.Duration `json:"request_timeout"`
	EnableValidation    bool          `json:"enable_validation"`
	EnableTransaction   bool          `json:"enable_transaction"`
	AllowPartialSuccess bool          `json:"allow_partial_success"`
}

// DefaultTaskBatchConfig returns default batch configuration
func DefaultTaskBatchConfig() TaskBatchConfig {
	return TaskBatchConfig{
		MaxBatchSize:        100,
		RequestTimeout:      60 * time.Second,
		EnableValidation:    true,
		EnableTransaction:   true,
		AllowPartialSuccess: true,
	}
}

// NewTaskBatchHandler creates a new batch handler
func NewTaskBatchHandler(service *tasks.Service, config TaskBatchConfig) *TaskBatchHandler {
	return &TaskBatchHandler{
		service: service,
		config:  config,
	}
}

// BatchUpdate handles batch task updates
func (h *TaskBatchHandler) BatchUpdate(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req BatchUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Validate request
	if err := h.validateBatchUpdateRequest(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid batch request", err.Error())
		return
	}

	// Get user ID from context
	userID := h.getUserID(r)

	// Convert request to batch updates
	updates := h.convertToBatchUpdates(&req)

	// Execute batch update
	result, err := h.service.BatchUpdateTasks(r.Context(), updates, userID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Batch update failed", err.Error())
		return
	}

	// Build response
	batchResponse := BatchUpdateResponse{
		Result:      *result,
		RequestID:   req.RequestID,
		ProcessedAt: time.Now(),
		UserID:      userID,
		Summary:     h.generateBatchSummary(result),
	}

	response.WriteSuccess(w, batchResponse)
}

// BatchCreate handles batch task creation
func (h *TaskBatchHandler) BatchCreate(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req BatchCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Validate request
	if err := h.validateBatchCreateRequest(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid batch create request", err.Error())
		return
	}

	// Get user ID from context
	userID := h.getUserID(r)

	// Process batch creation
	result := h.processBatchCreate(&req, userID, r)

	// Build response
	createResponse := BatchCreateResponse{
		Result:      result,
		RequestID:   req.RequestID,
		ProcessedAt: time.Now(),
		UserID:      userID,
		Summary:     h.generateCreateSummary(&result),
	}

	response.WriteSuccess(w, createResponse)
}

// BatchDelete handles batch task deletion
func (h *TaskBatchHandler) BatchDelete(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req BatchDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Validate request
	if err := h.validateBatchDeleteRequest(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid batch delete request", err.Error())
		return
	}

	// Get user ID from context
	userID := h.getUserID(r)

	// Process batch deletion
	result := h.processBatchDelete(&req, userID, r)

	// Build response
	deleteResponse := BatchDeleteResponse{
		Result:      result,
		RequestID:   req.RequestID,
		ProcessedAt: time.Now(),
		UserID:      userID,
		Summary:     h.generateDeleteSummary(&result),
	}

	response.WriteSuccess(w, deleteResponse)
}

// BatchStatusTransition handles batch status transitions
func (h *TaskBatchHandler) BatchStatusTransition(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req BatchStatusTransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Validate request
	if err := h.validateBatchStatusTransitionRequest(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid batch status transition request", err.Error())
		return
	}

	// Get user ID from context
	userID := h.getUserID(r)

	// Process batch status transition
	result := h.processBatchStatusTransition(&req, userID, r)

	// Build response
	transitionResponse := BatchStatusTransitionResponse{
		Result:      result,
		RequestID:   req.RequestID,
		ProcessedAt: time.Now(),
		UserID:      userID,
		Summary:     h.generateTransitionSummary(&result),
	}

	response.WriteSuccess(w, transitionResponse)
}

// GetBatchOperationStatus handles batch operation status queries
func (h *TaskBatchHandler) GetBatchOperationStatus(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		response.WriteError(w, http.StatusBadRequest, "Request ID is required", "")
		return
	}

	// In a real implementation, this would check operation status from a job queue
	// For now, return a mock status
	status := BatchOperationStatus{
		RequestID:       requestID,
		Status:          "completed",
		Progress:        100,
		StartedAt:       time.Now().Add(-5 * time.Minute),
		CompletedAt:     &[]time.Time{time.Now()}[0],
		TotalItems:      10,
		ProcessedItems:  10,
		SuccessfulItems: 8,
		FailedItems:     2,
	}

	response.WriteSuccess(w, status)
}

// Helper methods

func (h *TaskBatchHandler) validateBatchUpdateRequest(req *BatchUpdateRequest) error {
	if len(req.Updates) == 0 {
		return fmt.Errorf("no updates provided")
	}
	if len(req.Updates) > h.config.MaxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum %d", len(req.Updates), h.config.MaxBatchSize)
	}

	for i, update := range req.Updates {
		if update.TaskID == "" {
			return fmt.Errorf("update %d: task ID is required", i)
		}
		if update.Status == nil && update.Priority == nil && update.Assignee == nil &&
			update.DueDate == nil && len(update.Tags) == 0 {
			return fmt.Errorf("update %d: at least one field must be updated", i)
		}
	}

	return nil
}

func (h *TaskBatchHandler) validateBatchCreateRequest(req *BatchCreateRequest) error {
	if len(req.Tasks) == 0 {
		return fmt.Errorf("no tasks provided")
	}
	if len(req.Tasks) > h.config.MaxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum %d", len(req.Tasks), h.config.MaxBatchSize)
	}

	for i, task := range req.Tasks {
		if task.Title == "" {
			return fmt.Errorf("task %d: title is required", i)
		}
		if task.Type == "" {
			return fmt.Errorf("task %d: type is required", i)
		}
		if task.Priority == "" {
			return fmt.Errorf("task %d: priority is required", i)
		}
	}

	return nil
}

func (h *TaskBatchHandler) validateBatchDeleteRequest(req *BatchDeleteRequest) error {
	if len(req.TaskIDs) == 0 {
		return fmt.Errorf("no task IDs provided")
	}
	if len(req.TaskIDs) > h.config.MaxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum %d", len(req.TaskIDs), h.config.MaxBatchSize)
	}

	for i, taskID := range req.TaskIDs {
		if taskID == "" {
			return fmt.Errorf("task ID %d is empty", i)
		}
	}

	return nil
}

func (h *TaskBatchHandler) validateBatchStatusTransitionRequest(req *BatchStatusTransitionRequest) error {
	if len(req.Transitions) == 0 {
		return fmt.Errorf("no transitions provided")
	}
	if len(req.Transitions) > h.config.MaxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum %d", len(req.Transitions), h.config.MaxBatchSize)
	}

	for i, transition := range req.Transitions {
		if transition.TaskID == "" {
			return fmt.Errorf("transition %d: task ID is required", i)
		}
		if transition.ToStatus == "" {
			return fmt.Errorf("transition %d: target status is required", i)
		}
	}

	return nil
}

func (h *TaskBatchHandler) convertToBatchUpdates(req *BatchUpdateRequest) []tasks.BatchUpdate {
	updates := make([]tasks.BatchUpdate, len(req.Updates))

	for i, update := range req.Updates {
		updates[i] = tasks.BatchUpdate{
			TaskID:    update.TaskID,
			Status:    update.Status,
			Priority:  update.Priority,
			Assignee:  update.Assignee,
			Tags:      update.Tags,
			DueDate:   update.DueDate,
			UpdatedAt: time.Now(),
		}
	}

	return updates
}

func (h *TaskBatchHandler) processBatchCreate(req *BatchCreateRequest, userID string, r *http.Request) BatchCreateResult {
	result := BatchCreateResult{
		TotalRequested: len(req.Tasks),
		Successful:     make([]string, 0),
		Failed:         make([]BatchCreateError, 0),
	}

	for i, taskReq := range req.Tasks {
		// Convert to task
		task := types.Task{
			Title:              taskReq.Title,
			Description:        taskReq.Description,
			Type:               taskReq.Type,
			Priority:           taskReq.Priority,
			AcceptanceCriteria: taskReq.AcceptanceCriteria,
			Dependencies:       taskReq.Dependencies,
			Tags:               taskReq.Tags,
			Assignee:           taskReq.Assignee,
			DueDate:            taskReq.DueDate,
			SourcePRDID:        taskReq.SourcePRDID,
		}

		// Create task
		if err := h.service.CreateTask(r.Context(), &task, userID); err != nil {
			result.Failed = append(result.Failed, BatchCreateError{
				Index: i,
				Task:  taskReq,
				Error: err.Error(),
			})
		} else {
			result.Successful = append(result.Successful, task.ID)
		}
	}

	result.SuccessfulCount = len(result.Successful)
	result.FailedCount = len(result.Failed)

	return result
}

func (h *TaskBatchHandler) processBatchDelete(req *BatchDeleteRequest, userID string, r *http.Request) BatchDeleteResult {
	result := BatchDeleteResult{
		TotalRequested: len(req.TaskIDs),
		Successful:     make([]string, 0),
		Failed:         make([]BatchDeleteError, 0),
	}

	for _, taskID := range req.TaskIDs {
		if err := h.service.DeleteTask(r.Context(), taskID, userID); err != nil {
			result.Failed = append(result.Failed, BatchDeleteError{
				TaskID: taskID,
				Error:  err.Error(),
			})
		} else {
			result.Successful = append(result.Successful, taskID)
		}
	}

	result.SuccessfulCount = len(result.Successful)
	result.FailedCount = len(result.Failed)

	return result
}

func (h *TaskBatchHandler) processBatchStatusTransition(req *BatchStatusTransitionRequest, userID string, r *http.Request) BatchStatusTransitionResult {
	result := BatchStatusTransitionResult{
		TotalRequested: len(req.Transitions),
		Successful:     make([]StatusTransitionSuccess, 0),
		Failed:         make([]StatusTransitionError, 0),
	}

	for _, transition := range req.Transitions {
		// Get current task
		task, err := h.service.GetTask(r.Context(), transition.TaskID, userID)
		if err != nil {
			result.Failed = append(result.Failed, StatusTransitionError{
				TaskID:   transition.TaskID,
				ToStatus: transition.ToStatus,
				Error:    err.Error(),
			})
			continue
		}

		// Update status
		task.Status = transition.ToStatus
		if err := h.service.UpdateTask(r.Context(), task, userID); err != nil {
			result.Failed = append(result.Failed, StatusTransitionError{
				TaskID:   transition.TaskID,
				ToStatus: transition.ToStatus,
				Error:    err.Error(),
			})
		} else {
			result.Successful = append(result.Successful, StatusTransitionSuccess{
				TaskID:     transition.TaskID,
				FromStatus: task.Status, // This would be the old status in a real implementation
				ToStatus:   transition.ToStatus,
			})
		}
	}

	result.SuccessfulCount = len(result.Successful)
	result.FailedCount = len(result.Failed)

	return result
}

func (h *TaskBatchHandler) generateBatchSummary(result *tasks.BatchResult) BatchSummary {
	successRate := 0.0
	if result.TotalRequested > 0 {
		successRate = float64(result.SuccessfulCount) / float64(result.TotalRequested)
	}

	return BatchSummary{
		TotalRequested:  result.TotalRequested,
		SuccessfulCount: result.SuccessfulCount,
		FailedCount:     result.FailedCount,
		SuccessRate:     successRate,
	}
}

func (h *TaskBatchHandler) generateCreateSummary(result *BatchCreateResult) BatchSummary {
	successRate := 0.0
	if result.TotalRequested > 0 {
		successRate = float64(result.SuccessfulCount) / float64(result.TotalRequested)
	}

	return BatchSummary{
		TotalRequested:  result.TotalRequested,
		SuccessfulCount: result.SuccessfulCount,
		FailedCount:     result.FailedCount,
		SuccessRate:     successRate,
	}
}

func (h *TaskBatchHandler) generateDeleteSummary(result *BatchDeleteResult) BatchSummary {
	successRate := 0.0
	if result.TotalRequested > 0 {
		successRate = float64(result.SuccessfulCount) / float64(result.TotalRequested)
	}

	return BatchSummary{
		TotalRequested:  result.TotalRequested,
		SuccessfulCount: result.SuccessfulCount,
		FailedCount:     result.FailedCount,
		SuccessRate:     successRate,
	}
}

func (h *TaskBatchHandler) generateTransitionSummary(result *BatchStatusTransitionResult) BatchSummary {
	successRate := 0.0
	if result.TotalRequested > 0 {
		successRate = float64(result.SuccessfulCount) / float64(result.TotalRequested)
	}

	return BatchSummary{
		TotalRequested:  result.TotalRequested,
		SuccessfulCount: result.SuccessfulCount,
		FailedCount:     result.FailedCount,
		SuccessRate:     successRate,
	}
}

func (h *TaskBatchHandler) getUserID(r *http.Request) string {
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID
	}
	return "default_user"
}

// Request/Response types

// BatchUpdateRequest represents a batch update request
type BatchUpdateRequest struct {
	RequestID string             `json:"request_id"`
	Updates   []BatchUpdateItem  `json:"updates"`
	Options   BatchUpdateOptions `json:"options,omitempty"`
}

// BatchUpdateItem represents a single update in a batch
type BatchUpdateItem struct {
	TaskID   string              `json:"task_id"`
	Status   *types.TaskStatus   `json:"status,omitempty"`
	Priority *types.TaskPriority `json:"priority,omitempty"`
	Assignee *string             `json:"assignee,omitempty"`
	Tags     []string            `json:"tags,omitempty"`
	DueDate  *time.Time          `json:"due_date,omitempty"`
}

// BatchUpdateOptions represents options for batch updates
type BatchUpdateOptions struct {
	ContinueOnError bool `json:"continue_on_error"`
	ValidateOnly    bool `json:"validate_only"`
}

// BatchCreateRequest represents a batch create request
type BatchCreateRequest struct {
	RequestID string                `json:"request_id"`
	Tasks     []BatchCreateTaskItem `json:"tasks"`
	Options   BatchCreateOptions    `json:"options,omitempty"`
}

// BatchCreateTaskItem represents a task creation item
type BatchCreateTaskItem struct {
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
}

// BatchCreateOptions represents options for batch creation
type BatchCreateOptions struct {
	ContinueOnError bool `json:"continue_on_error"`
	ValidateOnly    bool `json:"validate_only"`
}

// BatchDeleteRequest represents a batch delete request
type BatchDeleteRequest struct {
	RequestID string             `json:"request_id"`
	TaskIDs   []string           `json:"task_ids"`
	Options   BatchDeleteOptions `json:"options,omitempty"`
}

// BatchDeleteOptions represents options for batch deletion
type BatchDeleteOptions struct {
	ContinueOnError bool `json:"continue_on_error"`
	Force           bool `json:"force"`
}

// BatchStatusTransitionRequest represents a batch status transition request
type BatchStatusTransitionRequest struct {
	RequestID   string                       `json:"request_id"`
	Transitions []StatusTransitionItem       `json:"transitions"`
	Options     BatchStatusTransitionOptions `json:"options,omitempty"`
}

// StatusTransitionItem represents a single status transition
type StatusTransitionItem struct {
	TaskID   string           `json:"task_id"`
	ToStatus types.TaskStatus `json:"to_status"`
	Comment  string           `json:"comment,omitempty"`
}

// BatchStatusTransitionOptions represents options for batch status transitions
type BatchStatusTransitionOptions struct {
	ContinueOnError bool `json:"continue_on_error"`
	ValidateOnly    bool `json:"validate_only"`
}

// Response types

// BatchUpdateResponse represents a batch update response
type BatchUpdateResponse struct {
	Result      tasks.BatchResult `json:"result"`
	RequestID   string            `json:"request_id"`
	ProcessedAt time.Time         `json:"processed_at"`
	UserID      string            `json:"user_id"`
	Summary     BatchSummary      `json:"summary"`
}

// BatchCreateResponse represents a batch create response
type BatchCreateResponse struct {
	Result      BatchCreateResult `json:"result"`
	RequestID   string            `json:"request_id"`
	ProcessedAt time.Time         `json:"processed_at"`
	UserID      string            `json:"user_id"`
	Summary     BatchSummary      `json:"summary"`
}

// BatchDeleteResponse represents a batch delete response
type BatchDeleteResponse struct {
	Result      BatchDeleteResult `json:"result"`
	RequestID   string            `json:"request_id"`
	ProcessedAt time.Time         `json:"processed_at"`
	UserID      string            `json:"user_id"`
	Summary     BatchSummary      `json:"summary"`
}

// BatchStatusTransitionResponse represents a batch status transition response
type BatchStatusTransitionResponse struct {
	Result      BatchStatusTransitionResult `json:"result"`
	RequestID   string                      `json:"request_id"`
	ProcessedAt time.Time                   `json:"processed_at"`
	UserID      string                      `json:"user_id"`
	Summary     BatchSummary                `json:"summary"`
}

// Result types

// BatchCreateResult represents batch creation results
type BatchCreateResult struct {
	TotalRequested  int                `json:"total_requested"`
	SuccessfulCount int                `json:"successful_count"`
	FailedCount     int                `json:"failed_count"`
	Successful      []string           `json:"successful"`
	Failed          []BatchCreateError `json:"failed"`
}

// BatchDeleteResult represents batch deletion results
type BatchDeleteResult struct {
	TotalRequested  int                `json:"total_requested"`
	SuccessfulCount int                `json:"successful_count"`
	FailedCount     int                `json:"failed_count"`
	Successful      []string           `json:"successful"`
	Failed          []BatchDeleteError `json:"failed"`
}

// BatchStatusTransitionResult represents batch status transition results
type BatchStatusTransitionResult struct {
	TotalRequested  int                       `json:"total_requested"`
	SuccessfulCount int                       `json:"successful_count"`
	FailedCount     int                       `json:"failed_count"`
	Successful      []StatusTransitionSuccess `json:"successful"`
	Failed          []StatusTransitionError   `json:"failed"`
}

// Error types

// BatchCreateError represents a batch creation error
type BatchCreateError struct {
	Index int                 `json:"index"`
	Task  BatchCreateTaskItem `json:"task"`
	Error string              `json:"error"`
}

// BatchDeleteError represents a batch deletion error
type BatchDeleteError struct {
	TaskID string `json:"task_id"`
	Error  string `json:"error"`
}

// StatusTransitionError represents a status transition error
type StatusTransitionError struct {
	TaskID   string           `json:"task_id"`
	ToStatus types.TaskStatus `json:"to_status"`
	Error    string           `json:"error"`
}

// Success types

// StatusTransitionSuccess represents a successful status transition
type StatusTransitionSuccess struct {
	TaskID     string           `json:"task_id"`
	FromStatus types.TaskStatus `json:"from_status"`
	ToStatus   types.TaskStatus `json:"to_status"`
}

// BatchSummary represents a summary of batch operation results
type BatchSummary struct {
	TotalRequested  int     `json:"total_requested"`
	SuccessfulCount int     `json:"successful_count"`
	FailedCount     int     `json:"failed_count"`
	SuccessRate     float64 `json:"success_rate"`
}

// BatchOperationStatus represents the status of a batch operation
type BatchOperationStatus struct {
	RequestID       string     `json:"request_id"`
	Status          string     `json:"status"`
	Progress        int        `json:"progress"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	TotalItems      int        `json:"total_items"`
	ProcessedItems  int        `json:"processed_items"`
	SuccessfulItems int        `json:"successful_items"`
	FailedItems     int        `json:"failed_items"`
	Errors          []string   `json:"errors,omitempty"`
}
