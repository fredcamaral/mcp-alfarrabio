// Package ports defines interfaces for external adapters
// for the lerian-mcp-memory CLI application.
package ports

import (
	"context"
	"time"
)

// AIService defines the interface for AI-powered document generation and analysis
type AIService interface {
	// Document generation
	GeneratePRD(ctx context.Context, request *PRDGenerationRequest) (*PRDGenerationResponse, error)
	GenerateTRD(ctx context.Context, request *TRDGenerationRequest) (*TRDGenerationResponse, error)
	GenerateMainTasks(ctx context.Context, request *MainTaskGenerationRequest) (*MainTaskGenerationResponse, error)
	GenerateSubTasks(ctx context.Context, request *SubTaskGenerationRequest) (*SubTaskGenerationResponse, error)

	// Document analysis
	AnalyzeContent(ctx context.Context, request *ContentAnalysisRequest) (*ContentAnalysisResponse, error)
	EstimateComplexity(ctx context.Context, content string) (*ComplexityEstimate, error)

	// Interactive sessions
	StartInteractiveSession(ctx context.Context, docType string) (*InteractiveSession, error)
	ContinueSession(ctx context.Context, sessionID string, userInput string) (*SessionResponse, error)
	EndSession(ctx context.Context, sessionID string) error

	// Health and status
	TestConnection(ctx context.Context) error
	IsOnline() bool
	GetAvailableModels() []string
}

// Generation request/response structures

// PRDGenerationRequest contains the context for PRD generation
type PRDGenerationRequest struct {
	UserInputs  []string               `json:"user_inputs"`
	Repository  string                 `json:"repository"`
	ProjectType string                 `json:"project_type"`
	Preferences UserPreferences        `json:"preferences"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Model       string                 `json:"model,omitempty"`
}

// PRDGenerationResponse contains the generated PRD
type PRDGenerationResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Features    []string  `json:"features"`
	UserStories []string  `json:"user_stories"`
	Content     string    `json:"content"`
	ModelUsed   string    `json:"model_used"`
	GeneratedAt time.Time `json:"generated_at"`
}

// TRDGenerationRequest contains the context for TRD generation
type TRDGenerationRequest struct {
	PRDID       string                 `json:"prd_id"`
	PRDContent  string                 `json:"prd_content"`
	Repository  string                 `json:"repository"`
	ProjectType string                 `json:"project_type"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Model       string                 `json:"model,omitempty"`
}

// TRDGenerationResponse contains the generated TRD
type TRDGenerationResponse struct {
	ID             string    `json:"id"`
	PRDID          string    `json:"prd_id"`
	Title          string    `json:"title"`
	Architecture   string    `json:"architecture"`
	TechStack      []string  `json:"tech_stack"`
	Requirements   []string  `json:"requirements"`
	Implementation []string  `json:"implementation"`
	Content        string    `json:"content"`
	ModelUsed      string    `json:"model_used"`
	GeneratedAt    time.Time `json:"generated_at"`
}

// MainTaskGenerationRequest contains the context for main task generation
type MainTaskGenerationRequest struct {
	TRDID      string                 `json:"trd_id"`
	TRDContent string                 `json:"trd_content"`
	Repository string                 `json:"repository"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Model      string                 `json:"model,omitempty"`
}

// MainTaskGenerationResponse contains the generated main tasks
type MainTaskGenerationResponse struct {
	Tasks       []*GeneratedMainTask `json:"tasks"`
	ModelUsed   string               `json:"model_used"`
	GeneratedAt time.Time            `json:"generated_at"`
}

// GeneratedMainTask represents a generated main task
type GeneratedMainTask struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Phase            string   `json:"phase"`
	Duration         string   `json:"duration"`
	AtomicValidation bool     `json:"atomic_validation"`
	Dependencies     []string `json:"dependencies"`
	Content          string   `json:"content"`
}

// SubTaskGenerationRequest contains the context for sub-task generation
type SubTaskGenerationRequest struct {
	MainTaskID      string                 `json:"main_task_id"`
	MainTaskContent string                 `json:"main_task_content"`
	Repository      string                 `json:"repository"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Model           string                 `json:"model,omitempty"`
}

// SubTaskGenerationResponse contains the generated sub-tasks
type SubTaskGenerationResponse struct {
	Tasks       []*GeneratedSubTask `json:"tasks"`
	ModelUsed   string              `json:"model_used"`
	GeneratedAt time.Time           `json:"generated_at"`
}

// GeneratedSubTask represents a generated sub-task
type GeneratedSubTask struct {
	ID                 string   `json:"id"`
	ParentTaskID       string   `json:"parent_task_id"`
	Name               string   `json:"name"`
	Duration           int      `json:"duration_hours"`
	Type               string   `json:"implementation_type"`
	Deliverables       []string `json:"deliverables"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	Dependencies       []string `json:"dependencies"`
	Content            string   `json:"content"`
}

// Analysis request/response structures

// ContentAnalysisRequest contains content to analyze
type ContentAnalysisRequest struct {
	Content string                 `json:"content"`
	Type    string                 `json:"type"` // prd, trd, requirement, etc.
	Context map[string]interface{} `json:"context,omitempty"`
	Model   string                 `json:"model,omitempty"`
}

// ContentAnalysisResponse contains the analysis result
type ContentAnalysisResponse struct {
	ID            string             `json:"id"`
	Summary       string             `json:"summary"`
	KeyFeatures   []string           `json:"key_features"`
	TechnicalReqs []string           `json:"technical_requirements"`
	Dependencies  []string           `json:"dependencies"`
	Complexity    ComplexityEstimate `json:"complexity"`
	Sections      []ContentSection   `json:"sections"`
	ModelUsed     string             `json:"model_used"`
	ProcessedAt   time.Time          `json:"processed_at"`
}

// ComplexityEstimate represents complexity analysis
type ComplexityEstimate struct {
	Overall        string             `json:"overall"` // low, medium, high
	Score          float64            `json:"score"`   // 0-10
	Factors        []string           `json:"factors"`
	EstimatedHours int                `json:"estimated_hours"`
	Confidence     float64            `json:"confidence"` // 0-1
	Categories     map[string]float64 `json:"categories"` // technical, business, integration, etc.
}

// ContentSection represents a section of analyzed content
type ContentSection struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Type    string `json:"type"`
	Order   int    `json:"order"`
}

// Interactive session structures

// InteractiveSession represents an active AI session
type InteractiveSession struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // prd, trd, task
	State     SessionState           `json:"state"`
	Context   map[string]interface{} `json:"context"`
	Messages  []SessionMessage       `json:"messages"`
	ModelUsed string                 `json:"model_used"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// SessionState represents the state of an interactive session
type SessionState string

const (
	SessionStateActive    SessionState = "active"
	SessionStateCompleted SessionState = "completed"
	SessionStateCancelled SessionState = "cancelled"
)

// SessionMessage represents a message in an interactive session
type SessionMessage struct {
	Role      string    `json:"role"` // user, assistant
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionResponse contains the AI response in an interactive session
type SessionResponse struct {
	SessionID string                 `json:"session_id"`
	Message   SessionMessage         `json:"message"`
	State     SessionState           `json:"state"`
	Context   map[string]interface{} `json:"context"`
	NextStep  string                 `json:"next_step,omitempty"`
}

// UserPreferences contains user preferences for generation
type UserPreferences struct {
	PreferredTaskSize   string   `json:"preferred_task_size"`  // small, medium, large
	PreferredComplexity string   `json:"preferred_complexity"` // low, medium, high
	IncludeTests        bool     `json:"include_tests"`
	IncludeDocs         bool     `json:"include_docs"`
	FavoriteTemplates   []string `json:"favorite_templates,omitempty"`
	AvoidPatterns       []string `json:"avoid_patterns,omitempty"`
	PreferredModel      string   `json:"preferred_model,omitempty"`
}
