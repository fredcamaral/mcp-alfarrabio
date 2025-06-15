// Package entities contains domain models for the lerian-mcp-memory CLI application
package entities

import (
	"time"
)

// ReviewSeverity represents the severity level of a finding
type ReviewSeverity string

const (
	SeverityCritical ReviewSeverity = "critical"
	SeverityHigh     ReviewSeverity = "high"
	SeverityMedium   ReviewSeverity = "medium"
	SeverityLow      ReviewSeverity = "low"
)

// ReviewPhase represents a phase in the review process
type ReviewPhase string

const (
	PhaseFoundation    ReviewPhase = "foundation"
	PhaseSecurity      ReviewPhase = "security"
	PhaseQuality       ReviewPhase = "quality"
	PhaseDocumentation ReviewPhase = "documentation"
	PhaseProduction    ReviewPhase = "production"
	PhaseSynthesis     ReviewPhase = "synthesis"
)

// ReviewMode represents the type of review to perform
type ReviewMode string

const (
	ReviewModeFull     ReviewMode = "full"
	ReviewModeQuick    ReviewMode = "quick"
	ReviewModeSecurity ReviewMode = "security"
	ReviewModeQuality  ReviewMode = "quality"
)

// ReviewPrompt represents a single review prompt
type ReviewPrompt struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Phase       ReviewPhase `json:"phase"`
	Order       int         `json:"order"`
	FilePath    string      `json:"file_path"`
	Content     string      `json:"content"`
	DependsOn   []string    `json:"depends_on,omitempty"`
	Tags        []string    `json:"tags,omitempty"`
}

// ReviewFinding represents a single finding from the review
type ReviewFinding struct {
	ID          string                 `json:"id"`
	PromptID    string                 `json:"prompt_id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    ReviewSeverity         `json:"severity"`
	Impact      string                 `json:"impact"`
	Effort      string                 `json:"effort"`
	Files       []string               `json:"files,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ReviewSession represents a complete review session
type ReviewSession struct {
	ID          string                 `json:"id"`
	Mode        ReviewMode             `json:"mode"`
	Repository  string                 `json:"repository"`
	Branch      string                 `json:"branch,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Status      ReviewStatus           `json:"status"`
	Progress    ReviewProgress         `json:"progress"`
	Findings    []ReviewFinding        `json:"findings"`
	Summary     *ReviewSummary         `json:"summary,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ReviewStatus represents the status of a review session
type ReviewStatus string

const (
	ReviewStatusPending    ReviewStatus = "pending"
	ReviewStatusInProgress ReviewStatus = "in_progress"
	ReviewStatusCompleted  ReviewStatus = "completed"
	ReviewStatusFailed     ReviewStatus = "failed"
	ReviewStatusCancelled  ReviewStatus = "cancelled"
)

// ReviewProgress tracks the progress of a review session
type ReviewProgress struct {
	CurrentPhase     ReviewPhase                   `json:"current_phase"`
	CurrentPromptID  string                        `json:"current_prompt_id"`
	TotalPrompts     int                           `json:"total_prompts"`
	CompletedPrompts int                           `json:"completed_prompts"`
	PhaseProgress    map[ReviewPhase]PhaseProgress `json:"phase_progress"`
}

// PhaseProgress tracks progress within a phase
type PhaseProgress struct {
	Status           ReviewStatus `json:"status"`
	TotalPrompts     int          `json:"total_prompts"`
	CompletedPrompts int          `json:"completed_prompts"`
	StartedAt        *time.Time   `json:"started_at,omitempty"`
	CompletedAt      *time.Time   `json:"completed_at,omitempty"`
}

// ReviewSummary contains aggregated results from the review
type ReviewSummary struct {
	TotalFindings       int                    `json:"total_findings"`
	FindingsBySeverity  map[ReviewSeverity]int `json:"findings_by_severity"`
	FindingsByPhase     map[ReviewPhase]int    `json:"findings_by_phase"`
	CriticalIssues      []string               `json:"critical_issues"`
	ImmediateActions    []string               `json:"immediate_actions"`
	EstimatedEffort     string                 `json:"estimated_effort"`
	ProductionReadiness string                 `json:"production_readiness"`
	GeneratedAt         time.Time              `json:"generated_at"`
}

// PromptExecution represents the execution of a single prompt
type PromptExecution struct {
	PromptID    string                 `json:"prompt_id"`
	SessionID   string                 `json:"session_id"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Status      ReviewStatus           `json:"status"`
	Response    string                 `json:"response,omitempty"`
	Findings    []ReviewFinding        `json:"findings,omitempty"`
	Error       *string                `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ReviewConfiguration contains configuration for review execution
type ReviewConfiguration struct {
	Mode             ReviewMode     `json:"mode"`
	IncludePhases    []ReviewPhase  `json:"include_phases,omitempty"`
	ExcludePhases    []ReviewPhase  `json:"exclude_phases,omitempty"`
	SeverityFilter   ReviewSeverity `json:"severity_filter,omitempty"`
	MaxConcurrency   int            `json:"max_concurrency,omitempty"`
	TimeoutPerPrompt time.Duration  `json:"timeout_per_prompt,omitempty"`
	RetryCount       int            `json:"retry_count,omitempty"`
	CustomPrompts    []string       `json:"custom_prompts,omitempty"`
	OutputDirectory  string         `json:"output_directory,omitempty"`
	GenerateTodoList bool           `json:"generate_todo_list"`
	StoreInMemory    bool           `json:"store_in_memory"`
}
