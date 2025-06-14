// Package ai provides shared AI types and interfaces
package ai

import (
	"context"
	"time"
)

// AIService defines the interface for AI operations
type AIService interface {
	GeneratePRD(ctx context.Context, request *PRDRequest) (*PRDResponse, error)
	GenerateTRD(ctx context.Context, request *TRDRequest) (*TRDResponse, error)
	GenerateMainTasks(ctx context.Context, request *TaskRequest) (*TaskResponse, error)
	GenerateSubTasks(ctx context.Context, request *TaskRequest) (*TaskResponse, error)
	StartInteractiveSession(ctx context.Context, docType string) (*SessionResponse, error)
	ContinueSession(ctx context.Context, sessionID, userInput string) (*SessionResponse, error)
	EndSession(ctx context.Context, sessionID string) error
	AnalyzeComplexity(ctx context.Context, content string) (int, error)
}

// PRDRequest represents a PRD generation request
type PRDRequest struct {
	UserInputs     []string `json:"user_inputs"`
	ProjectType    string   `json:"project_type"`
	Repository     string   `json:"repository"`
	SessionID      string   `json:"session_id,omitempty"`
	CustomRules    string   `json:"custom_rules,omitempty"` // Custom PRD generation rules (optional)
	UseDefaultRule bool     `json:"use_default_rule"`       // Whether to use default create-prd.mdc rules
}

// PRDResponse represents a PRD generation response
type PRDResponse struct {
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
	SessionID string            `json:"session_id,omitempty"`
}

// TRDRequest represents a TRD generation request
type TRDRequest struct {
	PRDContent     string `json:"prd_content"`
	Repository     string `json:"repository"`
	SessionID      string `json:"session_id,omitempty"`
	CustomRules    string `json:"custom_rules,omitempty"` // Custom TRD generation rules (optional)
	UseDefaultRule bool   `json:"use_default_rule"`       // Whether to use default create-trd.mdc rules
}

// TRDResponse represents a TRD generation response
type TRDResponse struct {
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
	SessionID string            `json:"session_id,omitempty"`
}

// TaskRequest represents a task generation request
type TaskRequest struct {
	Content        string `json:"content"`   // TRD content for main tasks, main task content for sub tasks
	TaskType       string `json:"task_type"` // "main" or "sub"
	Repository     string `json:"repository"`
	SessionID      string `json:"session_id,omitempty"`
	CustomRules    string `json:"custom_rules,omitempty"` // Custom task generation rules (optional)
	UseDefaultRule bool   `json:"use_default_rule"`       // Whether to use default generate-main-tasks.mdc or generate-sub-tasks.mdc rules
}

// TaskResponse represents a task generation response
type TaskResponse struct {
	Tasks     []GeneratedTask   `json:"tasks"`
	Metadata  map[string]string `json:"metadata"`
	SessionID string            `json:"session_id,omitempty"`
}

// GeneratedTask represents a generated task
type GeneratedTask struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Duration     time.Duration `json:"duration"`
	Priority     string        `json:"priority"`
	Dependencies []string      `json:"dependencies,omitempty"`
}

// SessionResponse represents an interactive session response
type SessionResponse struct {
	SessionID   string `json:"session_id"`
	Message     string `json:"message"`
	Question    string `json:"question,omitempty"`
	IsComplete  bool   `json:"is_complete"`
	FinalResult string `json:"final_result,omitempty"`
}

// Config holds AI service configuration
type Config struct {
	Provider   string        `json:"provider"` // "claude", "openai", "perplexity", "mock"
	APIKey     string        `json:"api_key"`
	BaseURL    string        `json:"base_url"`
	Model      string        `json:"model"`
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Provider:   "mock",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// AIClient defines the interface for low-level AI model clients
type AIClient interface {
	Complete(ctx context.Context, request *CompletionRequest) (*CompletionResponse, error)
	Test(ctx context.Context) error
	GetConfig() *BaseConfig
}

// CompletionRequest represents a direct completion request to an AI model
type CompletionRequest struct {
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Message represents a single message in a conversation
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// CompletionResponse represents a completion response from an AI model
type CompletionResponse struct {
	ID             string        `json:"id"`
	Content        string        `json:"content"`
	Model          string        `json:"model"`
	FinishReason   string        `json:"finish_reason"`
	Usage          Usage         `json:"usage"`
	ProcessingTime time.Duration `json:"processing_time"`
	Provider       string        `json:"provider"`
	CreatedAt      time.Time     `json:"created_at"`
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
