// Package types provides data structures for AI-powered task generation and management.
package types

import (
	"time"
)

// Task represents a generated task with all its metadata
type Task struct {
	ID                 string         `json:"id"`
	Title              string         `json:"title"`
	Description        string         `json:"description"`
	Type               TaskType       `json:"type"`
	Priority           TaskPriority   `json:"priority"`
	Status             TaskStatus     `json:"status"`
	Complexity         TaskComplexity `json:"complexity"`
	EstimatedEffort    EffortEstimate `json:"estimated_effort"`
	AcceptanceCriteria []string       `json:"acceptance_criteria"`
	Dependencies       []string       `json:"dependencies"` // Task IDs this task depends on
	Blocks             []string       `json:"blocks"`       // Task IDs this task blocks
	Tags               []string       `json:"tags"`
	SourcePRDID        string         `json:"source_prd_id,omitempty"`
	SourceSection      string         `json:"source_section,omitempty"`
	Assignee           string         `json:"assignee,omitempty"`
	DueDate            *time.Time     `json:"due_date,omitempty"`
	Metadata           TaskMetadata   `json:"metadata"`
	QualityScore       QualityScore   `json:"quality_score"`
	Timestamps         TaskTimestamps `json:"timestamps"`
}

// TaskType represents the type of task (legacy - use TaskTypeEnum for new code)
type TaskType string

const (
	TaskTypeLegacyImplementation TaskType = "implementation"
	TaskTypeLegacyDesign         TaskType = "design"
	TaskTypeLegacyTesting        TaskType = "testing"
	TaskTypeLegacyDocumentation  TaskType = "documentation"
	TaskTypeLegacyResearch       TaskType = "research"
	TaskTypeLegacyReview         TaskType = "review"
	TaskTypeLegacyDeployment     TaskType = "deployment"
	TaskTypeLegacyArchitecture   TaskType = "architecture"
	TaskTypeLegacyBugFix         TaskType = "bugfix"
	TaskTypeLegacyRefactoring    TaskType = "refactoring"
	TaskTypeLegacyIntegration    TaskType = "integration"
	TaskTypeLegacyAnalysis       TaskType = "analysis"
)

// TaskPriority represents task priority levels (legacy - use TaskPriorityEnum for new code)
type TaskPriority string

const (
	TaskPriorityLegacyLow      TaskPriority = "low"
	TaskPriorityLegacyMedium   TaskPriority = "medium"
	TaskPriorityLegacyHigh     TaskPriority = "high"
	TaskPriorityLegacyCritical TaskPriority = "critical"
	TaskPriorityLegacyBlocking TaskPriority = "blocking"
)

// TaskStatus represents task status levels (legacy - use TaskStatusEnum for new code)
type TaskStatus string

const (
	TaskStatusLegacyTodo       TaskStatus = "todo"
	TaskStatusLegacyInProgress TaskStatus = "in_progress"
	TaskStatusLegacyCompleted  TaskStatus = "completed"
	TaskStatusLegacyBlocked    TaskStatus = "blocked"
	TaskStatusLegacyCancelled  TaskStatus = "cancelled"
)

// TaskComplexity represents task complexity analysis
type TaskComplexity struct {
	Level                ComplexityLevel   `json:"level"`
	Score                float64           `json:"score"` // 0.0-1.0
	Factors              ComplexityFactors `json:"factors"`
	TechnicalRisk        RiskLevelEnum     `json:"technical_risk"`
	BusinessImpact       ImpactLevelEnum   `json:"business_impact"`
	RequiredSkills       []string          `json:"required_skills"`
	ExternalDependencies []string          `json:"external_dependencies"`
}

// ComplexityLevel represents complexity levels
type ComplexityLevel string

const (
	ComplexityTrivial     ComplexityLevel = "trivial"
	ComplexitySimple      ComplexityLevel = "simple"
	ComplexityModerate    ComplexityLevel = "moderate"
	ComplexityComplex     ComplexityLevel = "complex"
	ComplexityVeryComplex ComplexityLevel = "very_complex"
)

// ComplexityFactors represents factors contributing to complexity
type ComplexityFactors struct {
	TechnicalComplexity     float64 `json:"technical_complexity"`      // 0.0-1.0
	IntegrationComplexity   float64 `json:"integration_complexity"`    // 0.0-1.0
	BusinessLogicComplexity float64 `json:"business_logic_complexity"` // 0.0-1.0
	DataComplexity          float64 `json:"data_complexity"`           // 0.0-1.0
	UIComplexity            float64 `json:"ui_complexity"`             // 0.0-1.0
	TestingComplexity       float64 `json:"testing_complexity"`        // 0.0-1.0
}

// ImpactLevel represents business impact levels
type ImpactLevel string

const (
	ImpactLegacyLow      ImpactLevel = "low"
	ImpactLegacyMedium   ImpactLevel = "medium"
	ImpactLegacyHigh     ImpactLevel = "high"
	ImpactLegacyCritical ImpactLevel = "critical"
)

// EffortEstimate represents effort estimation
type EffortEstimate struct {
	Hours            float64         `json:"hours"`
	Days             float64         `json:"days"`
	StoryPoints      *int            `json:"story_points,omitempty"`
	Confidence       float64         `json:"confidence"` // 0.0-1.0
	EstimationMethod string          `json:"estimation_method"`
	Breakdown        EffortBreakdown `json:"breakdown"`
}

// EffortBreakdown represents detailed effort breakdown
type EffortBreakdown struct {
	Analysis       float64 `json:"analysis"`
	Design         float64 `json:"design"`
	Implementation float64 `json:"implementation"`
	Testing        float64 `json:"testing"`
	Documentation  float64 `json:"documentation"`
	Review         float64 `json:"review"`
	Integration    float64 `json:"integration"`
	Deployment     float64 `json:"deployment"`
}

// TaskMetadata contains additional task metadata
type TaskMetadata struct {
	GenerationSource   string                 `json:"generation_source"` // ai_generated, user_created, template_based
	AIModel            string                 `json:"ai_model,omitempty"`
	TemplateID         string                 `json:"template_id,omitempty"`
	GenerationPrompt   string                 `json:"generation_prompt,omitempty"`
	RelatedUserStories []string               `json:"related_user_stories,omitempty"`
	BusinessValue      BusinessValueScore     `json:"business_value"`
	TechnicalDebt      TechnicalDebtScore     `json:"technical_debt"`
	UserImpact         UserImpactScore        `json:"user_impact"`
	ExtendedData       map[string]interface{} `json:"extended_data,omitempty"`
}

// BusinessValueScore represents business value assessment
type BusinessValueScore struct {
	Score       float64 `json:"score"`        // 0.0-1.0
	Revenue     float64 `json:"revenue"`      // 0.0-1.0
	CostSavings float64 `json:"cost_savings"` // 0.0-1.0
	Strategic   float64 `json:"strategic"`    // 0.0-1.0
	Competitive float64 `json:"competitive"`  // 0.0-1.0
}

// TechnicalDebtScore represents technical debt assessment
type TechnicalDebtScore struct {
	Score           float64 `json:"score"`           // 0.0-1.0
	CreatesDebt     float64 `json:"creates_debt"`    // 0.0-1.0
	ReducesDebt     float64 `json:"reduces_debt"`    // 0.0-1.0
	Maintainability float64 `json:"maintainability"` // 0.0-1.0
	Scalability     float64 `json:"scalability"`     // 0.0-1.0
}

// UserImpactScore represents user impact assessment
type UserImpactScore struct {
	Score         float64 `json:"score"`         // 0.0-1.0
	Usability     float64 `json:"usability"`     // 0.0-1.0
	Performance   float64 `json:"performance"`   // 0.0-1.0
	Accessibility float64 `json:"accessibility"` // 0.0-1.0
	Experience    float64 `json:"experience"`    // 0.0-1.0
}

// QualityScore represents task quality assessment
type QualityScore struct {
	OverallScore    float64        `json:"overall_score"` // 0.0-1.0
	Clarity         float64        `json:"clarity"`       // 0.0-1.0
	Completeness    float64        `json:"completeness"`  // 0.0-1.0
	Actionability   float64        `json:"actionability"` // 0.0-1.0
	Specificity     float64        `json:"specificity"`   // 0.0-1.0
	Feasibility     float64        `json:"feasibility"`   // 0.0-1.0
	Testability     float64        `json:"testability"`   // 0.0-1.0
	Issues          []QualityIssue `json:"issues,omitempty"`
	Recommendations []string       `json:"recommendations,omitempty"`
}

// QualityIssue represents a quality issue with a task
type QualityIssue struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// TaskTimestamps represents task timestamp information
type TaskTimestamps struct {
	Created   time.Time  `json:"created"`
	Updated   time.Time  `json:"updated"`
	Started   *time.Time `json:"started,omitempty"`
	Completed *time.Time `json:"completed,omitempty"`
	DueDate   *time.Time `json:"due_date,omitempty"`
}

// TaskSuggestionRequest represents a request for task suggestions
type TaskSuggestionRequest struct {
	PRDID         string                `json:"prd_id,omitempty"`
	PRDContent    string                `json:"prd_content,omitempty"`
	Context       TaskGenerationContext `json:"context"`
	Options       TaskGenerationOptions `json:"options"`
	ExistingTasks []Task                `json:"existing_tasks,omitempty"`
	ProjectState  ProjectState          `json:"project_state,omitempty"`
}

// TaskGenerationContext represents context for task generation
type TaskGenerationContext struct {
	ProjectName string            `json:"project_name"`
	ProjectType ProjectType       `json:"project_type"`
	TechStack   []string          `json:"tech_stack,omitempty"`
	TeamSize    int               `json:"team_size,omitempty"`
	Timeline    string            `json:"timeline,omitempty"`
	Budget      string            `json:"budget,omitempty"`
	Constraints []string          `json:"constraints,omitempty"`
	Preferences map[string]string `json:"preferences,omitempty"`
	Repository  string            `json:"repository,omitempty"`
	Branch      string            `json:"branch,omitempty"`
}

// TaskGenerationOptions represents options for task generation
type TaskGenerationOptions struct {
	MaxTasks                  int                    `json:"max_tasks"`
	MinQualityScore           float64                `json:"min_quality_score"`
	IncludeEstimation         bool                   `json:"include_estimation"`
	IncludeDependencies       bool                   `json:"include_dependencies"`
	IncludeAcceptanceCriteria bool                   `json:"include_acceptance_criteria"`
	TaskTypes                 []TaskType             `json:"task_types,omitempty"`
	PriorityDistribution      PriorityDistribution   `json:"priority_distribution"`
	ComplexityDistribution    ComplexityDistribution `json:"complexity_distribution"`
	AIModel                   string                 `json:"ai_model,omitempty"`
	UseTemplates              bool                   `json:"use_templates"`
	GenerationStyle           GenerationStyle        `json:"generation_style"`
}

// PriorityDistribution represents desired priority distribution
type PriorityDistribution struct {
	Critical float64 `json:"critical"` // 0.0-1.0
	High     float64 `json:"high"`     // 0.0-1.0
	Medium   float64 `json:"medium"`   // 0.0-1.0
	Low      float64 `json:"low"`      // 0.0-1.0
}

// ComplexityDistribution represents desired complexity distribution
type ComplexityDistribution struct {
	VeryComplex float64 `json:"very_complex"` // 0.0-1.0
	Complex     float64 `json:"complex"`      // 0.0-1.0
	Moderate    float64 `json:"moderate"`     // 0.0-1.0
	Simple      float64 `json:"simple"`       // 0.0-1.0
	Trivial     float64 `json:"trivial"`      // 0.0-1.0
}

// GenerationStyle represents different task generation styles
type GenerationStyle string

const (
	GenerationStyleAgile     GenerationStyle = "agile"     // User stories and sprints
	GenerationStyleWaterfall GenerationStyle = "waterfall" // Sequential phases
	GenerationStyleKanban    GenerationStyle = "kanban"    // Flow-based tasks
	GenerationStyleHybrid    GenerationStyle = "hybrid"    // Mixed approach
	GenerationStyleCustom    GenerationStyle = "custom"    // Custom methodology
)

// ProjectState represents current project state for contextual suggestions
type ProjectState struct {
	Phase               ProjectPhase `json:"phase"`
	CompletedTasks      []string     `json:"completed_tasks"`   // Task IDs
	InProgressTasks     []string     `json:"in_progress_tasks"` // Task IDs
	BlockedTasks        []string     `json:"blocked_tasks"`     // Task IDs
	RecentlyCompleted   []Task       `json:"recently_completed"`
	CurrentBottlenecks  []string     `json:"current_bottlenecks"`
	UpcomingMilestones  []Milestone  `json:"upcoming_milestones"`
	AvailableResources  ResourceInfo `json:"available_resources"`
	TechnicalChallenges []string     `json:"technical_challenges,omitempty"`
}

// ProjectPhase represents current project phase
type ProjectPhase string

const (
	PhaseDiscovery    ProjectPhase = "discovery"
	PhaseRequirements ProjectPhase = "requirements"
	PhaseDesign       ProjectPhase = "design"
	PhaseDevelopment  ProjectPhase = "development"
	PhaseTesting      ProjectPhase = "testing"
	PhaseDeployment   ProjectPhase = "deployment"
	PhaseMaintenance  ProjectPhase = "maintenance"
)

// Milestone represents a project milestone
type Milestone struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	DueDate      time.Time `json:"due_date"`
	Dependencies []string  `json:"dependencies"` // Task IDs
	Completed    bool      `json:"completed"`
}

// ResourceInfo represents available project resources
type ResourceInfo struct {
	Developers      int      `json:"developers"`
	Designers       int      `json:"designers"`
	QAEngineers     int      `json:"qa_engineers"`
	DevOpsEngineers int      `json:"devops_engineers"`
	ProductManagers int      `json:"product_managers"`
	AvailableHours  float64  `json:"available_hours"`
	Skills          []string `json:"skills"`
	Tools           []string `json:"tools"`
}

// TaskSuggestionResponse represents the response from task suggestion
type TaskSuggestionResponse struct {
	Tasks              []Task             `json:"tasks"`
	TotalGenerated     int                `json:"total_generated"`
	QualityFiltered    int                `json:"quality_filtered"`
	GenerationMetadata GenerationMetadata `json:"generation_metadata"`
	DependencyGraph    DependencyGraph    `json:"dependency_graph"`
	Recommendations    []string           `json:"recommendations"`
	Warnings           []string           `json:"warnings,omitempty"`
	NextSteps          []string           `json:"next_steps"`
}

// GenerationMetadata contains metadata about the generation process
type GenerationMetadata struct {
	AIModel          string           `json:"ai_model"`
	GenerationTime   time.Duration    `json:"generation_time"`
	PromptTokens     int              `json:"prompt_tokens,omitempty"`
	CompletionTokens int              `json:"completion_tokens,omitempty"`
	QualityThreshold float64          `json:"quality_threshold"`
	TemplatesUsed    []string         `json:"templates_used,omitempty"`
	ProcessingSteps  []ProcessingStep `json:"processing_steps"`
}

// DependencyGraph represents task dependencies
type DependencyGraph struct {
	Nodes []DependencyNode `json:"nodes"`
	Edges []DependencyEdge `json:"edges"`
}

// DependencyNode represents a task node in the dependency graph
type DependencyNode struct {
	TaskID     string          `json:"task_id"`
	Title      string          `json:"title"`
	Type       TaskType        `json:"type"`
	Priority   TaskPriority    `json:"priority"`
	Complexity ComplexityLevel `json:"complexity"`
}

// DependencyEdge represents a dependency relationship
type DependencyEdge struct {
	FromTaskID  string         `json:"from_task_id"`
	ToTaskID    string         `json:"to_task_id"`
	Type        DependencyType `json:"type"`
	Strength    float64        `json:"strength"` // 0.0-1.0
	Description string         `json:"description"`
}

// DependencyType represents types of dependencies
type DependencyType string

const (
	DependencyTypeBlocking    DependencyType = "blocking"    // Hard dependency
	DependencyTypePreferred   DependencyType = "preferred"   // Soft dependency
	DependencyTypeConflicting DependencyType = "conflicting" // Conflicting tasks
	DependencyTypeRelated     DependencyType = "related"     // Related but not dependent
)

// Note: TaskTemplate is defined in prd_types.go to avoid conflicts

// TemplateApplicability defines when a template is applicable
type TemplateApplicability struct {
	ProjectTypes  []ProjectType  `json:"project_types"`
	TechStacks    []string       `json:"tech_stacks"`
	TeamSizes     []int          `json:"team_sizes"` // Applicable team sizes
	ProjectPhases []ProjectPhase `json:"project_phases"`
	Keywords      []string       `json:"keywords"`   // Keywords for matching
	Conditions    []string       `json:"conditions"` // Conditional logic
}

// TemplateVariable represents a variable in a template
type TemplateVariable struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"` // string, number, boolean, list
	Description  string   `json:"description"`
	Required     bool     `json:"required"`
	DefaultValue string   `json:"default_value,omitempty"`
	Options      []string `json:"options,omitempty"` // For enum-type variables
}

// Note: TaskValidationResult, ValidationError, and ValidationWarning are defined in prd_types.go to avoid conflicts
