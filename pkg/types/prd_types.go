// Package types provides data structures for PRD (Product Requirements Document) processing.
package types

import (
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
	RiskLevel          RiskLevel            `json:"risk_level"`
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

// RiskLevel represents risk assessment
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

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
