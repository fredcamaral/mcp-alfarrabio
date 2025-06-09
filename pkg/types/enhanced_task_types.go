// Package types provides enhanced data structures for the enhanced task table schema.
package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// EnhancedTask represents a task with full database schema compatibility
type EnhancedTask struct {
	// Core identification
	ID          string  `json:"id" db:"id"`
	Title       string  `json:"title" db:"title"`
	Description *string `json:"description,omitempty" db:"description"`
	Content     string  `json:"content" db:"content"`

	// Task classification (ENUMs)
	Type       TaskTypeEnum       `json:"type" db:"type"`
	Status     TaskStatusEnum     `json:"status" db:"status"`
	Priority   TaskPriorityEnum   `json:"priority" db:"priority"`
	Complexity TaskComplexityEnum `json:"complexity,omitempty" db:"complexity"`

	// Ownership and assignment
	Assignee *string `json:"assignee,omitempty" db:"assignee"`

	// Repository and session context
	Repository string  `json:"repository" db:"repository"`
	SessionID  *string `json:"session_id,omitempty" db:"session_id"`

	// Time tracking
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	DueDate     *time.Time `json:"due_date,omitempty" db:"due_date"`

	// Effort estimation
	EstimatedMinutes *int32   `json:"estimated_minutes,omitempty" db:"estimated_minutes"`
	ActualMinutes    *int32   `json:"actual_minutes,omitempty" db:"actual_minutes"`
	EstimatedHours   *float64 `json:"estimated_hours,omitempty" db:"estimated_hours"`
	EstimatedDays    *float64 `json:"estimated_days,omitempty" db:"estimated_days"`
	StoryPoints      *int32   `json:"story_points,omitempty" db:"story_points"`

	// Quality and complexity scoring
	ComplexityScore    *float64 `json:"complexity_score,omitempty" db:"complexity_score"`
	QualityScore       *float64 `json:"quality_score,omitempty" db:"quality_score"`
	ConfidenceScore    *float64 `json:"confidence_score,omitempty" db:"confidence_score"`
	BusinessValueScore *float64 `json:"business_value_score,omitempty" db:"business_value_score"`
	TechnicalDebtScore *float64 `json:"technical_debt_score,omitempty" db:"technical_debt_score"`
	UserImpactScore    *float64 `json:"user_impact_score,omitempty" db:"user_impact_score"`

	// Risk assessment
	TechnicalRisk  *RiskLevelEnum   `json:"technical_risk,omitempty" db:"technical_risk"`
	BusinessImpact *ImpactLevelEnum `json:"business_impact,omitempty" db:"business_impact"`

	// Hierarchical relationships
	ParentTaskID *string `json:"parent_task_id,omitempty" db:"parent_task_id"`

	// JSON fields
	Tags                 JSONArray `json:"tags" db:"tags"`
	Dependencies         JSONArray `json:"dependencies" db:"dependencies"`
	Blocks               JSONArray `json:"blocks" db:"blocks"`
	AcceptanceCriteria   JSONArray `json:"acceptance_criteria" db:"acceptance_criteria"`
	RequiredSkills       JSONArray `json:"required_skills" db:"required_skills"`
	ExternalDependencies JSONArray `json:"external_dependencies" db:"external_dependencies"`

	// AI and generation metadata
	AISuggested      bool    `json:"ai_suggested" db:"ai_suggested"`
	AIModel          *string `json:"ai_model,omitempty" db:"ai_model"`
	GenerationSource *string `json:"generation_source,omitempty" db:"generation_source"`
	GenerationPrompt *string `json:"generation_prompt,omitempty" db:"generation_prompt"`
	TemplateID       *string `json:"template_id,omitempty" db:"template_id"`

	// PRD integration
	PRDID         *string `json:"prd_id,omitempty" db:"prd_id"`
	SourcePRDID   *string `json:"source_prd_id,omitempty" db:"source_prd_id"`
	SourceSection *string `json:"source_section,omitempty" db:"source_section"`
	PatternID     *string `json:"pattern_id,omitempty" db:"pattern_id"`

	// Version control context
	Branch     *string `json:"branch,omitempty" db:"branch"`
	CommitHash *string `json:"commit_hash,omitempty" db:"commit_hash"`

	// Extended metadata
	Metadata     JSONObject `json:"metadata" db:"metadata"`
	ExtendedData JSONObject `json:"extended_data" db:"extended_data"`

	// Audit and soft delete
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	DeletedBy *string    `json:"deleted_by,omitempty" db:"deleted_by"`
	Version   int32      `json:"version" db:"version"`

	// Search (computed field)
	SearchVector *string `json:"-" db:"search_vector"` // Internal use only
}

// Enum types for database compatibility

type TaskTypeEnum string

const (
	TaskTypeImplementation TaskTypeEnum = "implementation"
	TaskTypeDesign         TaskTypeEnum = "design"
	TaskTypeTesting        TaskTypeEnum = "testing"
	TaskTypeDocumentation  TaskTypeEnum = "documentation"
	TaskTypeResearch       TaskTypeEnum = "research"
	TaskTypeReview         TaskTypeEnum = "review"
	TaskTypeDeployment     TaskTypeEnum = "deployment"
	TaskTypeArchitecture   TaskTypeEnum = "architecture"
	TaskTypeBugfix         TaskTypeEnum = "bugfix"
	TaskTypeRefactoring    TaskTypeEnum = "refactoring"
	TaskTypeIntegration    TaskTypeEnum = "integration"
	TaskTypeAnalysis       TaskTypeEnum = "analysis"
)

type TaskStatusEnum string

const (
	TaskStatusPending    TaskStatusEnum = "pending"
	TaskStatusInProgress TaskStatusEnum = "in_progress"
	TaskStatusCompleted  TaskStatusEnum = "completed"
	TaskStatusCancelled  TaskStatusEnum = "cancelled"
	TaskStatusBlocked    TaskStatusEnum = "blocked"
	TaskStatusTodo       TaskStatusEnum = "todo"
)

type TaskPriorityEnum string

const (
	TaskPriorityLow      TaskPriorityEnum = "low"
	TaskPriorityMedium   TaskPriorityEnum = "medium"
	TaskPriorityHigh     TaskPriorityEnum = "high"
	TaskPriorityCritical TaskPriorityEnum = "critical"
	TaskPriorityBlocking TaskPriorityEnum = "blocking"
)

type TaskComplexityEnum string

const (
	TaskComplexityTrivial     TaskComplexityEnum = "trivial"
	TaskComplexitySimple      TaskComplexityEnum = "simple"
	TaskComplexityModerate    TaskComplexityEnum = "moderate"
	TaskComplexityComplex     TaskComplexityEnum = "complex"
	TaskComplexityVeryComplex TaskComplexityEnum = "very_complex"
)

type RiskLevelEnum string

const (
	RiskLevelLow      RiskLevelEnum = "low"
	RiskLevelMedium   RiskLevelEnum = "medium"
	RiskLevelHigh     RiskLevelEnum = "high"
	RiskLevelCritical RiskLevelEnum = "critical"
)

type ImpactLevelEnum string

const (
	ImpactLevelLow      ImpactLevelEnum = "low"
	ImpactLevelMedium   ImpactLevelEnum = "medium"
	ImpactLevelHigh     ImpactLevelEnum = "high"
	ImpactLevelCritical ImpactLevelEnum = "critical"
)

// Custom types for JSON handling

// JSONArray represents a JSON array stored in the database
type JSONArray []interface{}

// Scan implements the sql.Scanner interface
func (ja *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*ja = JSONArray{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, ja)
	case string:
		return json.Unmarshal([]byte(v), ja)
	default:
		return fmt.Errorf("cannot scan %T into JSONArray", value)
	}
}

// Value implements the driver.Valuer interface
func (ja JSONArray) Value() (driver.Value, error) {
	if ja == nil {
		return "[]", nil
	}
	return json.Marshal(ja)
}

// JSONObject represents a JSON object stored in the database
type JSONObject map[string]interface{}

// Scan implements the sql.Scanner interface
func (jo *JSONObject) Scan(value interface{}) error {
	if value == nil {
		*jo = JSONObject{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, jo)
	case string:
		return json.Unmarshal([]byte(v), jo)
	default:
		return fmt.Errorf("cannot scan %T into JSONObject", value)
	}
}

// Value implements the driver.Valuer interface
func (jo JSONObject) Value() (driver.Value, error) {
	if jo == nil {
		return "{}", nil
	}
	return json.Marshal(jo)
}

// Supporting database entities

// Note: EnhancedPRD, TaskPattern, and TaskTemplate are defined in prd_types.go to avoid conflicts

// TaskEffortBreakdown represents detailed effort estimation
type TaskEffortBreakdown struct {
	ID                  string    `json:"id" db:"id"`
	TaskID              string    `json:"task_id" db:"task_id"`
	AnalysisHours       *float64  `json:"analysis_hours,omitempty" db:"analysis_hours"`
	DesignHours         *float64  `json:"design_hours,omitempty" db:"design_hours"`
	ImplementationHours *float64  `json:"implementation_hours,omitempty" db:"implementation_hours"`
	TestingHours        *float64  `json:"testing_hours,omitempty" db:"testing_hours"`
	DocumentationHours  *float64  `json:"documentation_hours,omitempty" db:"documentation_hours"`
	ReviewHours         *float64  `json:"review_hours,omitempty" db:"review_hours"`
	IntegrationHours    *float64  `json:"integration_hours,omitempty" db:"integration_hours"`
	DeploymentHours     *float64  `json:"deployment_hours,omitempty" db:"deployment_hours"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// TaskQualityIssue represents a quality issue with a task
type TaskQualityIssue struct {
	ID          string     `json:"id" db:"id"`
	TaskID      string     `json:"task_id" db:"task_id"`
	IssueType   string     `json:"issue_type" db:"issue_type"`
	Severity    string     `json:"severity" db:"severity"`
	Description string     `json:"description" db:"description"`
	Suggestion  *string    `json:"suggestion,omitempty" db:"suggestion"`
	Resolved    bool       `json:"resolved" db:"resolved"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// TaskAuditLog represents an audit trail entry
type TaskAuditLog struct {
	ID               string     `json:"id" db:"id"`
	TaskID           string     `json:"task_id" db:"task_id"`
	Operation        string     `json:"operation" db:"operation"`
	OldValues        JSONObject `json:"old_values,omitempty" db:"old_values"`
	NewValues        JSONObject `json:"new_values,omitempty" db:"new_values"`
	ChangedFields    JSONArray  `json:"changed_fields" db:"changed_fields"`
	UserID           *string    `json:"user_id,omitempty" db:"user_id"`
	SessionID        *string    `json:"session_id,omitempty" db:"session_id"`
	IPAddress        *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent        *string    `json:"user_agent,omitempty" db:"user_agent"`
	Timestamp        time.Time  `json:"timestamp" db:"timestamp"`
	TransactionID    *int64     `json:"transaction_id,omitempty" db:"transaction_id"`
	Repository       *string    `json:"repository,omitempty" db:"repository"`
	Branch           *string    `json:"branch,omitempty" db:"branch"`
	OperationContext *string    `json:"operation_context,omitempty" db:"operation_context"`
	SourceSystem     *string    `json:"source_system,omitempty" db:"source_system"`
	CreatedDate      time.Time  `json:"created_date" db:"created_date"`
}

// Conversion methods

// ToLegacyTask converts EnhancedTask to legacy Task structure
func (et *EnhancedTask) ToLegacyTask() *Task {
	task := et.createBaseTask()
	et.setOptionalFields(task)
	et.setArrayFields(task)
	et.setComplexityFields(task)
	et.setEffortFields(task)
	et.setQualityFields(task)
	et.setMetadataFields(task)
	return task
}

// createBaseTask creates the base task structure
func (et *EnhancedTask) createBaseTask() *Task {
	return &Task{
		ID:          et.ID,
		Title:       et.Title,
		Description: "",
		Type:        TaskType(et.Type),
		Priority:    TaskPriority(et.Priority),
		Status:      TaskStatus(et.Status),
		Timestamps: TaskTimestamps{
			Created:   et.CreatedAt,
			Updated:   et.UpdatedAt,
			Started:   et.StartedAt,
			Completed: et.CompletedAt,
			DueDate:   et.DueDate,
		},
	}
}

// setOptionalFields sets optional string fields
func (et *EnhancedTask) setOptionalFields(task *Task) {
	if et.Description != nil {
		task.Description = *et.Description
	}
	if et.Assignee != nil {
		task.Assignee = *et.Assignee
	}
	if et.SourcePRDID != nil {
		task.SourcePRDID = *et.SourcePRDID
	}
	if et.SourceSection != nil {
		task.SourceSection = *et.SourceSection
	}
}

// setArrayFields sets array fields from JSON
func (et *EnhancedTask) setArrayFields(task *Task) {
	task.Tags = jsonArrayToStringSlice(et.Tags)
	task.Dependencies = jsonArrayToStringSlice(et.Dependencies)
	task.Blocks = jsonArrayToStringSlice(et.Blocks)
	task.AcceptanceCriteria = jsonArrayToStringSlice(et.AcceptanceCriteria)
}

// setComplexityFields sets complexity-related fields
func (et *EnhancedTask) setComplexityFields(task *Task) {
	if et.Complexity != "" {
		task.Complexity = TaskComplexity{
			Level: ComplexityLevel(et.Complexity),
		}
		if et.ComplexityScore != nil {
			task.Complexity.Score = *et.ComplexityScore
		}
	}
}

// setEffortFields sets effort estimation fields
func (et *EnhancedTask) setEffortFields(task *Task) {
	if et.EstimatedHours != nil {
		task.EstimatedEffort = EffortEstimate{
			Hours: *et.EstimatedHours,
		}
		if et.EstimatedDays != nil {
			task.EstimatedEffort.Days = *et.EstimatedDays
		}
		if et.StoryPoints != nil {
			storyPoints := int(*et.StoryPoints)
			task.EstimatedEffort.StoryPoints = &storyPoints
		}
	}
}

// setQualityFields sets quality score fields
func (et *EnhancedTask) setQualityFields(task *Task) {
	if et.QualityScore != nil {
		task.QualityScore = QualityScore{
			OverallScore: *et.QualityScore,
		}
	}
}

// setMetadataFields sets metadata fields
func (et *EnhancedTask) setMetadataFields(task *Task) {
	if et.Metadata != nil {
		metadata := TaskMetadata{
			ExtendedData: map[string]interface{}(et.Metadata),
		}
		if et.GenerationSource != nil {
			metadata.GenerationSource = *et.GenerationSource
		}
		if et.AIModel != nil {
			metadata.AIModel = *et.AIModel
		}
		if et.GenerationPrompt != nil {
			metadata.GenerationPrompt = *et.GenerationPrompt
		}
		if et.TemplateID != nil {
			metadata.TemplateID = *et.TemplateID
		}
		task.Metadata = metadata
	}
}

// FromLegacyTask converts legacy Task to EnhancedTask structure
func (et *EnhancedTask) FromLegacyTask(task *Task) {
	et.setBasicFields(task)
	et.setOptionalStringFields(task)
	et.setArrayFieldsFromLegacy(task)
	et.setComplexityFromLegacy(task)
	et.setEffortFromLegacy(task)
	et.setQualityFromLegacy(task)
	et.setMetadataFromLegacy(task)
	et.Version = 1
}

// Helper functions

func jsonArrayToStringSlice(ja JSONArray) []string {
	var result []string
	for _, item := range ja {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

func stringSliceToJSONArray(slice []string) JSONArray {
	result := make(JSONArray, 0, len(slice))
	for _, str := range slice {
		result = append(result, str)
	}
	return result
}

// setBasicFields sets the basic required fields
func (et *EnhancedTask) setBasicFields(task *Task) {
	et.ID = task.ID
	et.Title = task.Title
	et.Description = &task.Description
	et.Content = task.Description // Use description as content
	et.Type = TaskTypeEnum(task.Type)
	et.Priority = TaskPriorityEnum(task.Priority)
	et.Status = TaskStatusEnum(task.Status)
	et.CreatedAt = task.Timestamps.Created
	et.UpdatedAt = task.Timestamps.Updated
	et.StartedAt = task.Timestamps.Started
	et.CompletedAt = task.Timestamps.Completed
	et.DueDate = task.Timestamps.DueDate
}

// setOptionalStringFields sets optional string fields from legacy task
func (et *EnhancedTask) setOptionalStringFields(task *Task) {
	if task.Assignee != "" {
		et.Assignee = &task.Assignee
	}
	if task.SourcePRDID != "" {
		et.SourcePRDID = &task.SourcePRDID
	}
	if task.SourceSection != "" {
		et.SourceSection = &task.SourceSection
	}
}

// setArrayFieldsFromLegacy converts string slices to JSON arrays
func (et *EnhancedTask) setArrayFieldsFromLegacy(task *Task) {
	et.Tags = stringSliceToJSONArray(task.Tags)
	et.Dependencies = stringSliceToJSONArray(task.Dependencies)
	et.Blocks = stringSliceToJSONArray(task.Blocks)
	et.AcceptanceCriteria = stringSliceToJSONArray(task.AcceptanceCriteria)
}

// setComplexityFromLegacy sets complexity fields from legacy task
func (et *EnhancedTask) setComplexityFromLegacy(task *Task) {
	if task.Complexity.Level != "" {
		et.Complexity = TaskComplexityEnum(task.Complexity.Level)
		if task.Complexity.Score > 0 {
			et.ComplexityScore = &task.Complexity.Score
		}
	}
}

// setEffortFromLegacy sets effort estimation fields from legacy task
func (et *EnhancedTask) setEffortFromLegacy(task *Task) {
	if task.EstimatedEffort.Hours > 0 {
		et.EstimatedHours = &task.EstimatedEffort.Hours
		if task.EstimatedEffort.Days > 0 {
			et.EstimatedDays = &task.EstimatedEffort.Days
		}
		if task.EstimatedEffort.StoryPoints != nil {
			sp := *task.EstimatedEffort.StoryPoints
			if sp > math.MaxInt32 {
				sp = math.MaxInt32
			}
			// #nosec G115 - Integer overflow is handled by the check above
			storyPoints := int32(sp)
			et.StoryPoints = &storyPoints
		}
	}
}

// setQualityFromLegacy sets quality score from legacy task
func (et *EnhancedTask) setQualityFromLegacy(task *Task) {
	if task.QualityScore.OverallScore > 0 {
		et.QualityScore = &task.QualityScore.OverallScore
	}
}

// setMetadataFromLegacy sets metadata fields from legacy task
func (et *EnhancedTask) setMetadataFromLegacy(task *Task) {
	if task.Metadata.GenerationSource != "" {
		et.GenerationSource = &task.Metadata.GenerationSource
	}
	if task.Metadata.AIModel != "" {
		et.AIModel = &task.Metadata.AIModel
	}
	if task.Metadata.GenerationPrompt != "" {
		et.GenerationPrompt = &task.Metadata.GenerationPrompt
	}
	if task.Metadata.TemplateID != "" {
		et.TemplateID = &task.Metadata.TemplateID
	}
	if task.Metadata.ExtendedData != nil {
		et.Metadata = JSONObject(task.Metadata.ExtendedData)
	}
}

// Validation methods

// ValidateEnhancedTask validates an enhanced task
func ValidateEnhancedTask(task *EnhancedTask) []ValidationError {
	var errors []ValidationError

	errors = append(errors, validateRequiredFields(task)...)
	errors = append(errors, validateScoreFields(task)...)
	errors = append(errors, validateTemporalFields(task)...)
	errors = append(errors, validateStatusConsistency(task)...)

	return errors
}

// validateRequiredFields validates required fields of a task
func validateRequiredFields(task *EnhancedTask) []ValidationError {
	var errors []ValidationError

	if task.Title == "" {
		errors = append(errors, ValidationError{
			Field:    "title",
			Type:     "required",
			Message:  "Title is required",
			Severity: "critical",
			Code:     "REQUIRED_FIELD",
		})
	}

	if task.Content == "" {
		errors = append(errors, ValidationError{
			Field:    "content",
			Type:     "required",
			Message:  "Content is required",
			Severity: "critical",
			Code:     "REQUIRED_FIELD",
		})
	}

	if task.Repository == "" {
		errors = append(errors, ValidationError{
			Field:    "repository",
			Type:     "required",
			Message:  "Repository is required",
			Severity: "critical",
			Code:     "REQUIRED_FIELD",
		})
	}

	return errors
}

// validateScoreFields validates score-related fields
func validateScoreFields(task *EnhancedTask) []ValidationError {
	var errors []ValidationError

	if task.ComplexityScore != nil && (*task.ComplexityScore < 0 || *task.ComplexityScore > 1) {
		errors = append(errors, ValidationError{
			Field:    "complexity_score",
			Type:     "range",
			Message:  "Complexity score must be between 0 and 1",
			Severity: "major",
			Code:     "INVALID_RANGE",
		})
	}

	if task.QualityScore != nil && (*task.QualityScore < 0 || *task.QualityScore > 1) {
		errors = append(errors, ValidationError{
			Field:    "quality_score",
			Type:     "range",
			Message:  "Quality score must be between 0 and 1",
			Severity: "major",
			Code:     "INVALID_RANGE",
		})
	}

	return errors
}

// validateTemporalFields validates time-related fields
func validateTemporalFields(task *EnhancedTask) []ValidationError {
	var errors []ValidationError

	if task.StartedAt != nil && task.StartedAt.Before(task.CreatedAt) {
		errors = append(errors, ValidationError{
			Field:    "started_at",
			Type:     "temporal",
			Message:  "Start time cannot be before creation time",
			Severity: "major",
			Code:     "INVALID_TEMPORAL_ORDER",
		})
	}

	if task.CompletedAt != nil && task.StartedAt != nil && task.CompletedAt.Before(*task.StartedAt) {
		errors = append(errors, ValidationError{
			Field:    "completed_at",
			Type:     "temporal",
			Message:  "Completion time cannot be before start time",
			Severity: "major",
			Code:     "INVALID_TEMPORAL_ORDER",
		})
	}

	return errors
}

// validateStatusConsistency validates status consistency
func validateStatusConsistency(task *EnhancedTask) []ValidationError {
	var errors []ValidationError

	if task.Status == TaskStatusCompleted && task.CompletedAt == nil {
		errors = append(errors, ValidationError{
			Field:    "completed_at",
			Type:     "consistency",
			Message:  "Completed tasks must have completion timestamp",
			Severity: "major",
			Code:     "INCONSISTENT_STATUS",
		})
	}

	if task.Status == TaskStatusInProgress && task.StartedAt == nil {
		errors = append(errors, ValidationError{
			Field:    "started_at",
			Type:     "consistency",
			Message:  "In-progress tasks should have start timestamp",
			Severity: "minor",
			Code:     "MISSING_START_TIME",
		})
	}

	return errors
}
