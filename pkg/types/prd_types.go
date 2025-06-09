// Package types provides data structures for PRD (Product Requirements Document) processing.
package types

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// PRDDocument represents a complete PRD document
type PRDDocument struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Version    string        `json:"version"`
	Status     PRDStatus     `json:"status"`
	Content    PRDContent    `json:"content"`
	Metadata   PRDMetadata   `json:"metadata"`
	Analysis   PRDAnalysis   `json:"analysis"`
	Processing PRDProcessing `json:"processing"`
	Timestamps PRDTimestamps `json:"timestamps"`
}

// PRDStatus represents the status of a PRD document
type PRDStatus string

const (
	PRDStatusDraft     PRDStatus = "draft"
	PRDStatusImported  PRDStatus = "imported"
	PRDStatusProcessed PRDStatus = "processed"
	PRDStatusAnalyzed  PRDStatus = "analyzed"
	PRDStatusError     PRDStatus = "error"
)

// PRDContent represents the structured content of a PRD
type PRDContent struct {
	Raw       string       `json:"raw"`
	Sections  []PRDSection `json:"sections"`
	Structure PRDStructure `json:"structure"`
	Format    string       `json:"format"`
	Encoding  string       `json:"encoding"`
	WordCount int          `json:"word_count"`
}

// PRDSection represents a section within a PRD
type PRDSection struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Type     SectionType  `json:"type"`
	Content  string       `json:"content"`
	Level    int          `json:"level"`
	Order    int          `json:"order"`
	Children []PRDSection `json:"children,omitempty"`
}

// SectionType represents the type of a PRD section
type SectionType string

const (
	SectionTypeOverview      SectionType = "overview"
	SectionTypeObjectives    SectionType = "objectives"
	SectionTypeRequirements  SectionType = "requirements"
	SectionTypeFunctional    SectionType = "functional"
	SectionTypeNonFunctional SectionType = "non_functional"
	SectionTypeTechnical     SectionType = "technical"
	SectionTypeDesign        SectionType = "design"
	SectionTypeArchitecture  SectionType = "architecture"
	SectionTypeUserStories   SectionType = "user_stories"
	SectionTypeAcceptance    SectionType = "acceptance_criteria"
	SectionTypeConstraints   SectionType = "constraints"
	SectionTypeAssumptions   SectionType = "assumptions"
	SectionTypeTimeline      SectionType = "timeline"
	SectionTypeResources     SectionType = "resources"
	SectionTypeRisks         SectionType = "risks"
	SectionTypeSuccess       SectionType = "success_metrics"
	SectionTypeOther         SectionType = "other"
)

// PRDStructure represents the overall structure of a PRD
type PRDStructure struct {
	TotalSections  int                 `json:"total_sections"`
	SectionsByType map[SectionType]int `json:"sections_by_type"`
	MaxDepth       int                 `json:"max_depth"`
	HasTOC         bool                `json:"has_toc"`
	HasImages      bool                `json:"has_images"`
	HasTables      bool                `json:"has_tables"`
	HasCode        bool                `json:"has_code"`
	HasDiagrams    bool                `json:"has_diagrams"`
}

// PRDMetadata represents extracted metadata from a PRD
type PRDMetadata struct {
	Title              string               `json:"title"`
	Description        string               `json:"description"`
	Author             string               `json:"author"`
	Owner              string               `json:"owner"`
	Stakeholders       []string             `json:"stakeholders"`
	Priority           PRDPriority          `json:"priority"`
	ProjectType        ProjectType          `json:"project_type"`
	Domain             string               `json:"domain"`
	Technology         []string             `json:"technology"`
	Dependencies       []string             `json:"dependencies"`
	Tags               []string             `json:"tags"`
	EstimatedEffort    EstimatedEffort      `json:"estimated_effort"`
	BusinessValue      BusinessValue        `json:"business_value"`
	RiskLevel          RiskLevelEnum        `json:"risk_level"`
	UserStories        []UserStory          `json:"user_stories"`
	AcceptanceCriteria []AcceptanceCriteria `json:"acceptance_criteria"`
}

// PRDPriority represents the priority level of a PRD
type PRDPriority string

const (
	PRDPriorityLow      PRDPriority = "low"
	PRDPriorityMedium   PRDPriority = "medium"
	PRDPriorityHigh     PRDPriority = "high"
	PRDPriorityCritical PRDPriority = "critical"
)

// ProjectType represents the type of project
type ProjectType string

const (
	ProjectTypeFeature     ProjectType = "feature"
	ProjectTypeProduct     ProjectType = "product"
	ProjectTypeIntegration ProjectType = "integration"
	ProjectTypeBugFix      ProjectType = "bugfix"
	ProjectTypeRefactor    ProjectType = "refactor"
	ProjectTypeResearch    ProjectType = "research"
	ProjectTypeOther       ProjectType = "other"
)

// EstimatedEffort represents effort estimation
type EstimatedEffort struct {
	DevelopmentWeeks float64    `json:"development_weeks"`
	TestingWeeks     float64    `json:"testing_weeks"`
	TotalWeeks       float64    `json:"total_weeks"`
	TeamSize         int        `json:"team_size"`
	Complexity       Complexity `json:"complexity"`
	Confidence       float64    `json:"confidence"`
	Methodology      string     `json:"methodology"`
}

// BusinessValue represents business value assessment
type BusinessValue struct {
	Revenue        float64 `json:"revenue"`
	CostSavings    float64 `json:"cost_savings"`
	UserImpact     float64 `json:"user_impact"`
	StrategicValue float64 `json:"strategic_value"`
	OverallScore   float64 `json:"overall_score"`
	ROIEstimate    float64 `json:"roi_estimate"`
}

// Use RiskLevelEnum from enhanced_task_types.go to avoid conflicts

// Complexity represents complexity level
type Complexity string

const (
	ComplexityLow      Complexity = "low"
	ComplexityMedium   Complexity = "medium"
	ComplexityHigh     Complexity = "high"
	ComplexityVeryHigh Complexity = "very_high"
)

// UserStory represents a user story extracted from PRD
type UserStory struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	AsA         string      `json:"as_a"`
	IWant       string      `json:"i_want"`
	SoThat      string      `json:"so_that"`
	Priority    PRDPriority `json:"priority"`
	Effort      float64     `json:"effort"`
	Tags        []string    `json:"tags"`
}

// AcceptanceCriteria represents acceptance criteria
type AcceptanceCriteria struct {
	ID          string      `json:"id"`
	UserStoryID string      `json:"user_story_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Given       string      `json:"given"`
	When        string      `json:"when"`
	Then        string      `json:"then"`
	Priority    PRDPriority `json:"priority"`
	Tags        []string    `json:"tags"`
}

// PRDAnalysis represents analysis results of a PRD
type PRDAnalysis struct {
	ComplexityScore   float64         `json:"complexity_score"`
	CompletenessScore float64         `json:"completeness_score"`
	ClarityScore      float64         `json:"clarity_score"`
	QualityScore      float64         `json:"quality_score"`
	ReadabilityScore  float64         `json:"readability_score"`
	StructureScore    float64         `json:"structure_score"`
	Issues            []AnalysisIssue `json:"issues"`
	Recommendations   []string        `json:"recommendations"`
	MissingElements   []string        `json:"missing_elements"`
	KeyConcepts       []string        `json:"key_concepts"`
	TechnicalTerms    []string        `json:"technical_terms"`
	SimilarPRDs       []string        `json:"similar_prds"`
}

// AnalysisIssue represents an issue found during analysis
type AnalysisIssue struct {
	Type       IssueType `json:"type"`
	Severity   Severity  `json:"severity"`
	Message    string    `json:"message"`
	Location   string    `json:"location"`
	Suggestion string    `json:"suggestion"`
}

// IssueType represents the type of analysis issue
type IssueType string

const (
	IssueTypeStructure    IssueType = "structure"
	IssueTypeContent      IssueType = "content"
	IssueTypeClarity      IssueType = "clarity"
	IssueTypeCompleteness IssueType = "completeness"
	IssueTypeConsistency  IssueType = "consistency"
	IssueTypeFormatting   IssueType = "formatting"
)

// Severity represents the severity of an issue
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// PRDProcessing represents processing information
type PRDProcessing struct {
	ImportMethod     string           `json:"import_method"`
	FileSize         int64            `json:"file_size"`
	ProcessingTime   time.Duration    `json:"processing_time"`
	ParsingErrors    []string         `json:"parsing_errors"`
	ValidationErrors []string         `json:"validation_errors"`
	Warnings         []string         `json:"warnings"`
	ProcessorVersion string           `json:"processor_version"`
	AIModelUsed      string           `json:"ai_model_used"`
	ProcessingSteps  []ProcessingStep `json:"processing_steps"`
}

// ProcessingStep represents a step in the processing pipeline
type ProcessingStep struct {
	Name      string        `json:"name"`
	Status    StepStatus    `json:"status"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Error     string        `json:"error,omitempty"`
	Details   string        `json:"details,omitempty"`
}

// StepStatus represents the status of a processing step
type StepStatus string

const (
	StepStatusPending    StepStatus = "pending"
	StepStatusProcessing StepStatus = "processing"
	StepStatusCompleted  StepStatus = "completed"
	StepStatusFailed     StepStatus = "failed"
	StepStatusSkipped    StepStatus = "skipped"
)

// PRDTimestamps represents timestamp information
type PRDTimestamps struct {
	Created   time.Time  `json:"created"`
	Updated   time.Time  `json:"updated"`
	Imported  time.Time  `json:"imported"`
	Processed *time.Time `json:"processed,omitempty"`
	Analyzed  *time.Time `json:"analyzed,omitempty"`
}

// PRDImportRequest represents a request to import a PRD
type PRDImportRequest struct {
	Name     string            `json:"name"`
	Content  string            `json:"content"`
	Format   string            `json:"format"`
	Encoding string            `json:"encoding"`
	Metadata map[string]string `json:"metadata"`
	Options  ImportOptions     `json:"options"`
}

// ImportOptions represents options for PRD import
type ImportOptions struct {
	AutoProcess        bool     `json:"auto_process"`
	AutoAnalyze        bool     `json:"auto_analyze"`
	ExtractUserStories bool     `json:"extract_user_stories"`
	GenerateTasks      bool     `json:"generate_tasks"`
	AIProcessing       bool     `json:"ai_processing"`
	ValidationLevel    string   `json:"validation_level"`
	Tags               []string `json:"tags"`
}

// PRDImportResponse represents the response from PRD import
type PRDImportResponse struct {
	DocumentID string        `json:"document_id"`
	Status     PRDStatus     `json:"status"`
	Message    string        `json:"message"`
	Errors     []string      `json:"errors,omitempty"`
	Warnings   []string      `json:"warnings,omitempty"`
	Processing PRDProcessing `json:"processing"`
	NextSteps  []string      `json:"next_steps"`
}

// Enhanced database-compatible PRD types

// EnhancedPRD represents a PRD with full database schema compatibility
type EnhancedPRD struct {
	// Core identification
	ID         string `json:"id" db:"id"`
	Repository string `json:"repository" db:"repository"`
	Filename   string `json:"filename" db:"filename"`
	Content    string `json:"content" db:"content"`

	// Parse tracking
	ParsedAt        *time.Time `json:"parsed_at,omitempty" db:"parsed_at"`
	TaskCount       int32      `json:"task_count" db:"task_count"`
	ComplexityScore *float64   `json:"complexity_score,omitempty" db:"complexity_score"`
	ValidationScore *float64   `json:"validation_score,omitempty" db:"validation_score"`

	// Version and integrity
	Version           int32   `json:"version" db:"version"`
	FileSizeBytes     *int64  `json:"file_size_bytes,omitempty" db:"file_size_bytes"`
	FileHash          *string `json:"file_hash,omitempty" db:"file_hash"`
	ContentType       string  `json:"content_type" db:"content_type"`
	Author            *string `json:"author,omitempty" db:"author"`
	LastParsedVersion int32   `json:"last_parsed_version" db:"last_parsed_version"`

	// Parse status and quality
	ParseStatus string    `json:"parse_status" db:"parse_status"`
	ParseErrors JSONArray `json:"parse_errors" db:"parse_errors"`

	// Document classification
	DocumentType  string `json:"document_type" db:"document_type"`
	PriorityLevel string `json:"priority_level" db:"priority_level"`
	Status        string `json:"status" db:"status"`

	// Metadata
	Metadata JSONObject `json:"metadata" db:"metadata"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Search (computed field)
	SearchVector *string `json:"-" db:"search_vector"` // Internal use only
}

// TaskTemplate represents a reusable task template with database compatibility
type TaskTemplate struct {
	// Core identification
	ID          string  `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	Description *string `json:"description,omitempty" db:"description"`
	Category    *string `json:"category,omitempty" db:"category"`

	// Template structure
	TemplateData  JSONObject `json:"template_data" db:"template_data"`
	Applicability JSONObject `json:"applicability" db:"applicability"`
	Variables     JSONArray  `json:"variables" db:"variables"`

	// Template metadata
	ProjectType          *string   `json:"project_type,omitempty" db:"project_type"`
	ComplexityLevel      string    `json:"complexity_level" db:"complexity_level"`
	EstimatedEffortHours *float64  `json:"estimated_effort_hours,omitempty" db:"estimated_effort_hours"`
	RequiredSkills       JSONArray `json:"required_skills" db:"required_skills"`

	// Metrics and ratings
	UsageCount             int32    `json:"usage_count" db:"usage_count"`
	SuccessRate            *float64 `json:"success_rate,omitempty" db:"success_rate"`
	AvgCompletionTimeHours *float64 `json:"avg_completion_time_hours,omitempty" db:"avg_completion_time_hours"`
	UserRating             *float64 `json:"user_rating,omitempty" db:"user_rating"`
	FeedbackCount          int32    `json:"feedback_count" db:"feedback_count"`

	// Template versioning
	Version          int32   `json:"version" db:"version"`
	ParentTemplateID *string `json:"parent_template_id,omitempty" db:"parent_template_id"`
	IsActive         bool    `json:"is_active" db:"is_active"`

	// User and metadata
	CreatedBy *string    `json:"created_by,omitempty" db:"created_by"`
	Tags      JSONArray  `json:"tags" db:"tags"`
	Metadata  JSONObject `json:"metadata" db:"metadata"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Search (computed field)
	SearchVector *string `json:"-" db:"search_vector"` // Internal use only
}

// TaskPattern represents a machine learning pattern for task sequences
type TaskPattern struct {
	// Core identification
	ID          string  `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	Description *string `json:"description,omitempty" db:"description"`
	PatternType string  `json:"pattern_type" db:"pattern_type"`

	// Pattern definition
	Template     JSONObject `json:"template" db:"template"`
	Conditions   JSONObject `json:"conditions" db:"conditions"`
	TaskSequence JSONArray  `json:"task_sequence" db:"task_sequence"`

	// Pattern metrics
	OccurrenceCount      int32    `json:"occurrence_count" db:"occurrence_count"`
	AvgCompletionMinutes *int32   `json:"avg_completion_time_minutes,omitempty" db:"avg_completion_time_minutes"`
	SuccessRate          *float64 `json:"success_rate,omitempty" db:"success_rate"`
	EfficiencyScore      *float64 `json:"efficiency_score,omitempty" db:"efficiency_score"`

	// Context and applicability
	Repositories     JSONArray `json:"repositories" db:"repositories"`
	ProjectTypes     JSONArray `json:"project_types" db:"project_types"`
	TeamSizes        JSONArray `json:"team_sizes" db:"team_sizes"`
	ComplexityLevels JSONArray `json:"complexity_levels" db:"complexity_levels"`

	// Pattern relationships
	ParentPatternID *string   `json:"parent_pattern_id,omitempty" db:"parent_pattern_id"`
	RelatedPatterns JSONArray `json:"related_patterns" db:"related_patterns"`

	// Machine learning features
	ConfidenceScore *float64   `json:"confidence_score,omitempty" db:"confidence_score"`
	FeatureVector   JSONObject `json:"feature_vector" db:"feature_vector"`
	LastTrainedAt   *time.Time `json:"last_trained_at,omitempty" db:"last_trained_at"`

	// Usage tracking
	LastUsedAt         *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	AutoSuggestedCount int32      `json:"auto_suggested_count" db:"auto_suggested_count"`
	UserAcceptedCount  int32      `json:"user_accepted_count" db:"user_accepted_count"`
	UserRejectedCount  int32      `json:"user_rejected_count" db:"user_rejected_count"`

	// Status and lifecycle
	Status           string `json:"status" db:"status"`
	ValidationStatus string `json:"validation_status" db:"validation_status"`

	// Discovery and metadata
	DiscoveredBy string     `json:"discovered_by" db:"discovered_by"`
	CreatedBy    *string    `json:"created_by,omitempty" db:"created_by"`
	Tags         JSONArray  `json:"tags" db:"tags"`
	Metadata     JSONObject `json:"metadata" db:"metadata"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Search (computed field)
	SearchVector *string `json:"-" db:"search_vector"` // Internal use only
}

// WorkSession represents a productivity tracking session
type WorkSession struct {
	// Session identification
	ID         string  `json:"id" db:"id"`
	SessionID  string  `json:"session_id" db:"session_id"`
	Repository string  `json:"repository" db:"repository"`
	Branch     *string `json:"branch,omitempty" db:"branch"`

	// Time tracking
	StartTime       time.Time  `json:"start_time" db:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty" db:"end_time"`
	DurationMinutes *int32     `json:"duration_minutes,omitempty" db:"duration_minutes"`

	// Task activity
	TasksCompleted  JSONArray `json:"tasks_completed" db:"tasks_completed"`
	TasksStarted    JSONArray `json:"tasks_started" db:"tasks_started"`
	TasksInProgress JSONArray `json:"tasks_in_progress" db:"tasks_in_progress"`
	TasksBlocked    JSONArray `json:"tasks_blocked" db:"tasks_blocked"`

	// Development activity
	ToolsUsed        JSONArray `json:"tools_used" db:"tools_used"`
	FilesChanged     JSONArray `json:"files_changed" db:"files_changed"`
	CommitsMade      JSONArray `json:"commits_made" db:"commits_made"`
	CommandsExecuted JSONArray `json:"commands_executed" db:"commands_executed"`

	// Session context
	SessionType    string  `json:"session_type" db:"session_type"`
	WorkMode       string  `json:"work_mode" db:"work_mode"`
	SessionSummary *string `json:"session_summary,omitempty" db:"session_summary"`
	SessionNotes   *string `json:"session_notes,omitempty" db:"session_notes"`

	// Productivity metrics
	ProductivityScore *float64 `json:"productivity_score,omitempty" db:"productivity_score"`
	FocusScore        *float64 `json:"focus_score,omitempty" db:"focus_score"`
	EfficiencyScore   *float64 `json:"efficiency_score,omitempty" db:"efficiency_score"`
	QualityScore      *float64 `json:"quality_score,omitempty" db:"quality_score"`

	// Activity counts
	TotalTasksTouched   int32 `json:"total_tasks_touched" db:"total_tasks_touched"`
	TasksCompletedCount int32 `json:"tasks_completed_count" db:"tasks_completed_count"`
	FilesModifiedCount  int32 `json:"files_modified_count" db:"files_modified_count"`
	LinesAdded          int32 `json:"lines_added" db:"lines_added"`
	LinesDeleted        int32 `json:"lines_deleted" db:"lines_deleted"`

	// Tool and environment
	CLIMode         bool    `json:"cli_mode" db:"cli_mode"`
	IDEUsed         *string `json:"ide_used,omitempty" db:"ide_used"`
	OperatingSystem *string `json:"operating_system,omitempty" db:"operating_system"`
	Environment     *string `json:"environment,omitempty" db:"environment"`

	// Interruptions and breaks
	InterruptionCount    int32 `json:"interruption_count" db:"interruption_count"`
	BreakDurationMinutes int32 `json:"break_duration_minutes" db:"break_duration_minutes"`
	ContextSwitches      int32 `json:"context_switches" db:"context_switches"`

	// AI assistance
	AIInteractionsCount   int32 `json:"ai_interactions_count" db:"ai_interactions_count"`
	AISuggestionsAccepted int32 `json:"ai_suggestions_accepted" db:"ai_suggestions_accepted"`
	AISuggestionsRejected int32 `json:"ai_suggestions_rejected" db:"ai_suggestions_rejected"`
	AIGeneratedTasksCount int32 `json:"ai_generated_tasks_count" db:"ai_generated_tasks_count"`

	// Session status
	Status           string  `json:"status" db:"status"`
	CompletionReason *string `json:"completion_reason,omitempty" db:"completion_reason"`

	// Goals and outcomes
	SessionGoals        JSONArray `json:"session_goals" db:"session_goals"`
	GoalsAchieved       JSONArray `json:"goals_achieved" db:"goals_achieved"`
	BlockersEncountered JSONArray `json:"blockers_encountered" db:"blockers_encountered"`
	Learnings           JSONArray `json:"learnings" db:"learnings"`

	// Team collaboration
	Collaborators       JSONArray `json:"collaborators" db:"collaborators"`
	PairProgramming     bool      `json:"pair_programming" db:"pair_programming"`
	CodeReviewsGiven    int32     `json:"code_reviews_given" db:"code_reviews_given"`
	CodeReviewsReceived int32     `json:"code_reviews_received" db:"code_reviews_received"`

	// Metadata and tags
	Tags     JSONArray  `json:"tags" db:"tags"`
	Metadata JSONObject `json:"metadata" db:"metadata"`

	// User identification
	UserID    *string `json:"user_id,omitempty" db:"user_id"`
	UserEmail *string `json:"user_email,omitempty" db:"user_email"`

	// External integrations
	GithubSessionID  *string    `json:"github_session_id,omitempty" db:"github_session_id"`
	JiraSessionID    *string    `json:"jira_session_id,omitempty" db:"jira_session_id"`
	ExternalToolData JSONObject `json:"external_tool_data" db:"external_tool_data"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Search (computed field)
	SearchVector *string `json:"-" db:"search_vector"` // Internal use only
}

// Enhanced validation types

// ValidationError represents a validation error with enhanced details
type ValidationError struct {
	Field      string `json:"field"`
	Type       string `json:"type"`
	Message    string `json:"message"`
	Severity   string `json:"severity"`
	Code       string `json:"code"`
	Suggestion string `json:"suggestion,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field      string `json:"field"`
	Type       string `json:"type"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	Code       string `json:"code"`
}

// TaskValidationResult represents the result of task validation
type TaskValidationResult struct {
	IsValid     bool                `json:"is_valid"`
	Errors      []ValidationError   `json:"errors,omitempty"`
	Warnings    []ValidationWarning `json:"warnings,omitempty"`
	Suggestions []string            `json:"suggestions,omitempty"`
	Score       float64             `json:"score"` // Overall validation score 0.0-1.0
}

// Enum types for database compatibility

type DocumentTypeEnum string

const (
	DocumentTypePRD          DocumentTypeEnum = "prd"
	DocumentTypeSpec         DocumentTypeEnum = "spec"
	DocumentTypeRequirements DocumentTypeEnum = "requirements"
	DocumentTypeDesign       DocumentTypeEnum = "design"
	DocumentTypeOther        DocumentTypeEnum = "other"
)

type ParseStatusEnum string

const (
	ParseStatusPending ParseStatusEnum = "pending"
	ParseStatusSuccess ParseStatusEnum = "success"
	ParseStatusPartial ParseStatusEnum = "partial"
	ParseStatusFailed  ParseStatusEnum = "failed"
)

type PriorityLevelEnum string

const (
	PriorityLevelLow      PriorityLevelEnum = "low"
	PriorityLevelMedium   PriorityLevelEnum = "medium"
	PriorityLevelHigh     PriorityLevelEnum = "high"
	PriorityLevelCritical PriorityLevelEnum = "critical"
)

type StatusEnum string

const (
	StatusActive     StatusEnum = "active"
	StatusArchived   StatusEnum = "archived"
	StatusDeprecated StatusEnum = "deprecated"
	StatusDraft      StatusEnum = "draft"
)

// Value method for database driver compatibility
func (dt DocumentTypeEnum) Value() (driver.Value, error) {
	return string(dt), nil
}

func (ps ParseStatusEnum) Value() (driver.Value, error) {
	return string(ps), nil
}

func (pl PriorityLevelEnum) Value() (driver.Value, error) {
	return string(pl), nil
}

func (s StatusEnum) Value() (driver.Value, error) {
	return string(s), nil
}

// Conversion methods

// ToLegacyPRD converts EnhancedPRD to legacy PRDDocument structure
func (ep *EnhancedPRD) ToLegacyPRD() *PRDDocument {
	prd := &PRDDocument{
		ID:      ep.ID,
		Name:    ep.Filename,
		Version: fmt.Sprintf("v%d", ep.Version),
		Status:  PRDStatusProcessed,
		Content: PRDContent{
			Raw:       ep.Content,
			Format:    ep.ContentType,
			WordCount: len(ep.Content),
		},
		Metadata: PRDMetadata{
			Title:    ep.Filename,
			Priority: PRDPriorityMedium,
		},
		Analysis: PRDAnalysis{
			ComplexityScore: 0.0,
			QualityScore:    0.0,
		},
		Processing: PRDProcessing{
			FileSize: *ep.FileSizeBytes,
		},
		Timestamps: PRDTimestamps{
			Created: ep.CreatedAt,
			Updated: ep.UpdatedAt,
		},
	}

	if ep.ComplexityScore != nil {
		prd.Analysis.ComplexityScore = *ep.ComplexityScore
	}

	if ep.ValidationScore != nil {
		prd.Analysis.QualityScore = *ep.ValidationScore
	}

	if ep.ParsedAt != nil {
		prd.Timestamps.Processed = ep.ParsedAt
	}

	// Map status
	switch ep.Status {
	case "active":
		prd.Status = PRDStatusProcessed
	case "draft":
		prd.Status = PRDStatusDraft
	default:
		prd.Status = PRDStatusImported
	}

	// Map priority
	switch ep.PriorityLevel {
	case "low":
		prd.Metadata.Priority = PRDPriorityLow
	case "medium":
		prd.Metadata.Priority = PRDPriorityMedium
	case "high":
		prd.Metadata.Priority = PRDPriorityHigh
	case "critical":
		prd.Metadata.Priority = PRDPriorityCritical
	}

	return prd
}

// FromLegacyPRD converts legacy PRDDocument to EnhancedPRD structure
func (ep *EnhancedPRD) FromLegacyPRD(prd *PRDDocument) {
	ep.ID = prd.ID
	ep.Filename = prd.Name
	ep.Content = prd.Content.Raw
	ep.ContentType = prd.Content.Format
	ep.CreatedAt = prd.Timestamps.Created
	ep.UpdatedAt = prd.Timestamps.Updated

	// Set parsed timestamp
	if prd.Timestamps.Processed != nil {
		ep.ParsedAt = prd.Timestamps.Processed
	}

	// Map complexity and quality scores
	if prd.Analysis.ComplexityScore > 0 {
		ep.ComplexityScore = &prd.Analysis.ComplexityScore
	}

	if prd.Analysis.QualityScore > 0 {
		ep.ValidationScore = &prd.Analysis.QualityScore
	}

	// Set file size
	if prd.Processing.FileSize > 0 {
		ep.FileSizeBytes = &prd.Processing.FileSize
	}

	// Map status
	switch prd.Status {
	case PRDStatusProcessed:
		ep.Status = "active"
	case PRDStatusDraft:
		ep.Status = "draft"
	default:
		ep.Status = "active"
	}

	// Map priority
	switch prd.Metadata.Priority {
	case PRDPriorityLow:
		ep.PriorityLevel = "low"
	case PRDPriorityMedium:
		ep.PriorityLevel = "medium"
	case PRDPriorityHigh:
		ep.PriorityLevel = "high"
	case PRDPriorityCritical:
		ep.PriorityLevel = "critical"
	default:
		ep.PriorityLevel = "medium"
	}

	// Set defaults
	ep.Version = 1
	ep.TaskCount = 0
	ep.DocumentType = "prd"
	ep.ParseStatus = "pending"
	ep.LastParsedVersion = 1
}
