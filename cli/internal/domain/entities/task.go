// Package entities defines core data structures and business entities
// for the lerian-mcp-memory CLI application.
package entities

import (
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Status represents the current state of a task
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
)

// Priority indicates task importance level
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// Task represents a single task with all metadata and validation
type Task struct {
	ID            string     `json:"id" validate:"required,uuid"`
	Content       string     `json:"content" validate:"required,min=1,max=1000"`
	Status        Status     `json:"status" validate:"required,oneof=pending in_progress completed cancelled"`
	Priority      Priority   `json:"priority" validate:"required,oneof=low medium high"`
	Repository    string     `json:"repository" validate:"required"`
	SessionID     string     `json:"session_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	EstimatedMins int        `json:"estimated_mins,omitempty" validate:"gte=0"`
	ActualMins    int        `json:"actual_mins,omitempty" validate:"gte=0"`
	Tags          []string   `json:"tags,omitempty"`
	ParentTaskID  string     `json:"parent_task_id,omitempty" validate:"omitempty,uuid"`
	AISuggested   bool       `json:"ai_suggested"`
}

// NewTask creates a new task with required fields and default values
func NewTask(content, repository string) (*Task, error) {
	task := &Task{
		ID:          uuid.New().String(),
		Content:     strings.TrimSpace(content),
		Status:      StatusPending,
		Priority:    PriorityMedium,
		Repository:  repository,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AISuggested: false,
	}

	if err := task.Validate(); err != nil {
		return nil, err
	}

	return task, nil
}

// NewTaskWithOptions creates a new task with optional parameters
func NewTaskWithOptions(content, repository string, options *TaskOptions) (*Task, error) {
	task := &Task{
		ID:          uuid.New().String(),
		Content:     strings.TrimSpace(content),
		Status:      StatusPending,
		Priority:    PriorityMedium,
		Repository:  repository,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AISuggested: false,
	}

	if options != nil {
		task.SessionID = options.SessionID
		task.EstimatedMins = options.EstimatedMins
		task.Tags = options.Tags
		task.ParentTaskID = options.ParentTaskID

		if options.Priority != "" {
			task.Priority = options.Priority
		}

		if options.AISuggested {
			task.AISuggested = true
		}
	}

	if err := task.Validate(); err != nil {
		return nil, err
	}

	return task, nil
}

// TaskOptions holds optional parameters for task creation
type TaskOptions struct {
	Priority      Priority
	SessionID     string
	EstimatedMins int
	Tags          []string
	ParentTaskID  string
	AISuggested   bool
}

// Validate checks if task fields meet business rules
func (t *Task) Validate() error {
	validate := validator.New()
	return validate.Struct(t)
}

// Start marks task as in progress and updates timestamp
func (t *Task) Start() error {
	if t.Status == StatusCompleted || t.Status == StatusCancelled {
		return ErrInvalidStatusTransition
	}

	t.Status = StatusInProgress
	t.UpdatedAt = time.Now()
	return t.Validate()
}

// Complete marks task as finished and sets completion timestamp
func (t *Task) Complete() error {
	if t.Status == StatusCancelled {
		return ErrInvalidStatusTransition
	}

	now := time.Now()
	t.Status = StatusCompleted
	t.UpdatedAt = now
	t.CompletedAt = &now

	return t.Validate()
}

// Cancel marks task as cancelled and updates timestamp
func (t *Task) Cancel() error {
	if t.Status == StatusCompleted {
		return ErrInvalidStatusTransition
	}

	t.Status = StatusCancelled
	t.UpdatedAt = time.Now()
	return t.Validate()
}

// Reset moves task back to pending state
func (t *Task) Reset() error {
	t.Status = StatusPending
	t.UpdatedAt = time.Now()
	t.CompletedAt = nil
	return t.Validate()
}

// UpdateContent modifies task description with validation
func (t *Task) UpdateContent(content string) error {
	originalContent := t.Content
	originalUpdatedAt := t.UpdatedAt

	t.Content = strings.TrimSpace(content)
	t.UpdatedAt = time.Now()

	if err := t.Validate(); err != nil {
		// Restore original values on validation failure
		t.Content = originalContent
		t.UpdatedAt = originalUpdatedAt
		return err
	}

	return nil
}

// SetPriority changes task priority
func (t *Task) SetPriority(priority Priority) error {
	originalPriority := t.Priority
	originalUpdatedAt := t.UpdatedAt

	t.Priority = priority
	t.UpdatedAt = time.Now()

	if err := t.Validate(); err != nil {
		// Restore original values on validation failure
		t.Priority = originalPriority
		t.UpdatedAt = originalUpdatedAt
		return err
	}

	return nil
}

// AddTag appends a new tag to the task
func (t *Task) AddTag(tag string) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return
	}

	for _, existingTag := range t.Tags {
		if existingTag == tag {
			return
		}
	}

	t.Tags = append(t.Tags, tag)
	t.UpdatedAt = time.Now()
}

// RemoveTag removes a tag from the task
func (t *Task) RemoveTag(tag string) {
	for i, existingTag := range t.Tags {
		if existingTag == tag {
			t.Tags = append(t.Tags[:i], t.Tags[i+1:]...)
			t.UpdatedAt = time.Now()
			break
		}
	}
}

// SetEstimation updates estimated time in minutes
func (t *Task) SetEstimation(minutes int) error {
	if minutes < 0 {
		return ErrInvalidEstimation
	}

	originalEstimatedMins := t.EstimatedMins
	originalUpdatedAt := t.UpdatedAt

	t.EstimatedMins = minutes
	t.UpdatedAt = time.Now()

	if err := t.Validate(); err != nil {
		// Restore original values on validation failure
		t.EstimatedMins = originalEstimatedMins
		t.UpdatedAt = originalUpdatedAt
		return err
	}

	return nil
}

// SetActualTime records actual time spent on task
func (t *Task) SetActualTime(minutes int) error {
	if minutes < 0 {
		return ErrInvalidActualTime
	}

	originalActualMins := t.ActualMins
	originalUpdatedAt := t.UpdatedAt

	t.ActualMins = minutes
	t.UpdatedAt = time.Now()

	if err := t.Validate(); err != nil {
		// Restore original values on validation failure
		t.ActualMins = originalActualMins
		t.UpdatedAt = originalUpdatedAt
		return err
	}

	return nil
}

// IsValidStatus checks if a status value is valid
func IsValidStatus(status string) bool {
	switch Status(status) {
	case StatusPending, StatusInProgress, StatusCompleted, StatusCancelled:
		return true
	default:
		return false
	}
}

// IsValidPriority checks if a priority value is valid
func IsValidPriority(priority string) bool {
	switch Priority(priority) {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	default:
		return false
	}
}

// GetDuration calculates time elapsed since creation
func (t *Task) GetDuration() time.Duration {
	if t.CompletedAt != nil {
		return t.CompletedAt.Sub(t.CreatedAt)
	}
	return time.Since(t.CreatedAt)
}

// IsOverdue checks if task is overdue based on estimation
func (t *Task) IsOverdue() bool {
	if t.EstimatedMins == 0 || t.Status == StatusCompleted || t.Status == StatusCancelled {
		return false
	}

	estimatedDuration := time.Duration(t.EstimatedMins) * time.Minute
	return time.Since(t.CreatedAt) > estimatedDuration
}

// HasTag checks if task contains a specific tag
func (t *Task) HasTag(tag string) bool {
	for _, existingTag := range t.Tags {
		if existingTag == tag {
			return true
		}
	}
	return false
}
