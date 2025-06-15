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
	StatusPending    Status = "pending" // CLI uses "pending" for backward compatibility
	StatusTodo       Status = "todo"    // Server-compatible status
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
	ID            string                 `json:"id" validate:"required,uuid"`
	Content       string                 `json:"content" validate:"required,min=1,max=1000"` // Deprecated: Use Title and Description
	Title         string                 `json:"title,omitempty" validate:"max=255"`         // Server-compatible field
	Description   string                 `json:"description,omitempty" validate:"max=5000"`  // Server-compatible field
	Type          string                 `json:"type,omitempty"`
	Status        Status                 `json:"status" validate:"required,oneof=pending todo in_progress completed cancelled"`
	Priority      Priority               `json:"priority" validate:"required,oneof=low medium high"`
	Repository    string                 `json:"repository" validate:"required"`
	SessionID     string                 `json:"session_id,omitempty"`
	DueDate       *time.Time             `json:"due_date,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	EstimatedMins int                    `json:"estimated_mins,omitempty" validate:"gte=0"`
	ActualMins    int                    `json:"actual_mins,omitempty" validate:"gte=0"`
	Tags          []string               `json:"tags,omitempty"`
	ParentTaskID  string                 `json:"parent_task_id,omitempty" validate:"omitempty,uuid"`
	AISuggested   bool                   `json:"ai_suggested"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	// Server-compatible fields
	Assignee           string   `json:"assignee,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	Dependencies       []string `json:"dependencies,omitempty"` // Task IDs this depends on
	SourcePRDID        string   `json:"source_prd_id,omitempty"`
	Branch             string   `json:"branch,omitempty"`
	BlockedBy          []string `json:"blocked_by,omitempty"`
	Blocking           []string `json:"blocking,omitempty"`
	TimeTracked        int      `json:"time_tracked,omitempty"`
	Confidence         float64  `json:"confidence,omitempty"`
	Complexity         string   `json:"complexity,omitempty"` // simple, medium, complex
	RiskLevel          string   `json:"risk_level,omitempty"` // low, medium, high
}

// NewTask creates a new task with required fields and default values
func NewTask(content, repository string) (*Task, error) {
	trimmedContent := strings.TrimSpace(content)

	// Truncate title to max 255 chars for server compatibility
	title := trimmedContent
	if len(title) > 255 {
		title = title[:252] + "..."
	}

	task := &Task{
		ID:          uuid.New().String(),
		Content:     trimmedContent,
		Title:       title, // For server compatibility
		Description: "",    // Empty description by default
		Status:      StatusPending,
		Priority:    PriorityMedium,
		Repository:  repository,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AISuggested: false,
		Metadata:    make(map[string]interface{}),
	}

	if err := task.Validate(); err != nil {
		return nil, err
	}

	return task, nil
}

// NewTaskWithOptions creates a new task with optional parameters
func NewTaskWithOptions(content, repository string, options *TaskOptions) (*Task, error) {
	trimmedContent := strings.TrimSpace(content)

	// Truncate title to max 255 chars for server compatibility
	title := trimmedContent
	if len(title) > 255 {
		title = title[:252] + "..."
	}

	task := &Task{
		ID:          uuid.New().String(),
		Content:     trimmedContent,
		Title:       title, // For server compatibility
		Description: "",    // Empty description by default
		Status:      StatusPending,
		Priority:    PriorityMedium,
		Repository:  repository,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AISuggested: false,
		Metadata:    make(map[string]interface{}),
	}

	if options != nil {
		task.SessionID = options.SessionID
		task.DueDate = options.DueDate
		task.EstimatedMins = options.EstimatedMins
		task.Tags = options.Tags
		task.ParentTaskID = options.ParentTaskID

		if options.Priority != "" {
			task.Priority = options.Priority
		}

		if options.AISuggested {
			task.AISuggested = true
		}

		if options.Metadata != nil {
			task.Metadata = options.Metadata
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
	DueDate       *time.Time
	EstimatedMins int
	Tags          []string
	ParentTaskID  string
	AISuggested   bool
	Metadata      map[string]interface{}
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
	originalTitle := t.Title
	originalDescription := t.Description
	originalUpdatedAt := t.UpdatedAt

	trimmedContent := strings.TrimSpace(content)
	t.Content = trimmedContent
	t.Title = trimmedContent // Update title for server compatibility
	// Keep existing description or clear it if content is being updated
	t.UpdatedAt = time.Now()

	if err := t.Validate(); err != nil {
		// Restore original values on validation failure
		t.Content = originalContent
		t.Title = originalTitle
		t.Description = originalDescription
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
	case StatusPending, StatusTodo, StatusInProgress, StatusCompleted, StatusCancelled:
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

// IsOverdue checks if task is overdue based on due date or estimation
func (t *Task) IsOverdue() bool {
	if t.Status == StatusCompleted || t.Status == StatusCancelled {
		return false
	}

	// Check due date first if set
	if t.DueDate != nil {
		return time.Now().After(*t.DueDate)
	}

	// Fall back to estimation-based check
	if t.EstimatedMins == 0 {
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

// SetDueDate sets or updates the task due date
func (t *Task) SetDueDate(dueDate *time.Time) error {
	t.DueDate = dueDate
	t.UpdatedAt = time.Now()
	return t.Validate()
}

// ClearDueDate removes the due date from the task
func (t *Task) ClearDueDate() {
	t.DueDate = nil
	t.UpdatedAt = time.Now()
}

// IsDueSoon checks if task is due within the specified duration
func (t *Task) IsDueSoon(within time.Duration) bool {
	if t.DueDate == nil || t.Status == StatusCompleted || t.Status == StatusCancelled {
		return false
	}

	return time.Until(*t.DueDate) <= within
}

// GetTimeUntilDue returns time remaining until due date
func (t *Task) GetTimeUntilDue() *time.Duration {
	if t.DueDate == nil {
		return nil
	}

	duration := time.Until(*t.DueDate)
	return &duration
}

// SetMetadata sets a metadata key-value pair
func (t *Task) SetMetadata(key string, value interface{}) {
	if t.Metadata == nil {
		t.Metadata = make(map[string]interface{})
	}
	t.Metadata[key] = value
	t.UpdatedAt = time.Now()
}

// GetMetadata retrieves a metadata value by key
func (t *Task) GetMetadata(key string) (interface{}, bool) {
	if t.Metadata == nil {
		return nil, false
	}
	value, exists := t.Metadata[key]
	return value, exists
}

// GetMetadataString retrieves a metadata value as string
func (t *Task) GetMetadataString(key string) (string, bool) {
	if value, exists := t.GetMetadata(key); exists {
		if str, ok := value.(string); ok {
			return str, true
		}
	}
	return "", false
}

// GetMetadataInt retrieves a metadata value as int
func (t *Task) GetMetadataInt(key string) (int, bool) {
	if value, exists := t.GetMetadata(key); exists {
		if i, ok := value.(int); ok {
			return i, true
		}
		if f, ok := value.(float64); ok {
			return int(f), true
		}
	}
	return 0, false
}

// GetMetadataBool retrieves a metadata value as bool
func (t *Task) GetMetadataBool(key string) (bool, bool) {
	if value, exists := t.GetMetadata(key); exists {
		if b, ok := value.(bool); ok {
			return b, true
		}
	}
	return false, false
}

// GetDisplayContent returns the content to display (uses Content if available, otherwise Title)
func (t *Task) GetDisplayContent() string {
	if t.Content != "" {
		return t.Content
	}
	return t.Title
}

// SetTitleAndDescription sets both title and description, updating Content for backward compatibility
func (t *Task) SetTitleAndDescription(title, description string) error {
	originalContent := t.Content
	originalTitle := t.Title
	originalDescription := t.Description
	originalUpdatedAt := t.UpdatedAt

	t.Title = strings.TrimSpace(title)
	t.Description = strings.TrimSpace(description)
	// For backward compatibility, set Content to Title
	t.Content = t.Title
	t.UpdatedAt = time.Now()

	if err := t.Validate(); err != nil {
		// Restore original values on validation failure
		t.Content = originalContent
		t.Title = originalTitle
		t.Description = originalDescription
		t.UpdatedAt = originalUpdatedAt
		return err
	}

	return nil
}

// RemoveMetadata removes a metadata key
func (t *Task) RemoveMetadata(key string) {
	if t.Metadata != nil {
		delete(t.Metadata, key)
		t.UpdatedAt = time.Now()
	}
}

// ClearMetadata removes all metadata
func (t *Task) ClearMetadata() {
	t.Metadata = make(map[string]interface{})
	t.UpdatedAt = time.Now()
}

// HasMetadata checks if a metadata key exists
func (t *Task) HasMetadata(key string) bool {
	if t.Metadata == nil {
		return false
	}
	_, exists := t.Metadata[key]
	return exists
}
