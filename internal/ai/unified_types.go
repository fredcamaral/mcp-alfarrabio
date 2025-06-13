// Package ai provides unified types for AI services
// This file consolidates conflicting type definitions to resolve compilation issues
package ai

import (
	"context"
	"time"
)

// UnifiedQualityMetrics represents a consolidated quality assessment structure
type UnifiedQualityMetrics struct {
	// Core quality dimensions
	Confidence   float64 `json:"confidence"`   // Confidence in the assessment (0.0-1.0)
	Relevance    float64 `json:"relevance"`    // How relevant the content is (0.0-1.0)
	Clarity      float64 `json:"clarity"`      // How clear and understandable (0.0-1.0)
	Completeness float64 `json:"completeness"` // How complete the information is (0.0-1.0)

	// Derived metrics
	Score           float64  `json:"score"`            // Overall quality score (0.0-1.0)
	Actionability   float64  `json:"actionability"`    // How actionable the content is (0.0-1.0)
	Uniqueness      float64  `json:"uniqueness"`       // How unique/novel the information is (0.0-1.0)
	OverallScore    float64  `json:"overall_score"`    // Weighted overall quality score (0.0-1.0)
	MissingElements []string `json:"missing_elements"` // What could make this better
}

// AIService defines the unified interface for AI operations
type AIService interface {
	// Core completion operations
	GenerateCompletion(ctx context.Context, prompt string, options *CompletionOptions) (*CompletionResponse, error)

	// Quality assessment
	AssessQuality(ctx context.Context, content string) (*UnifiedQualityMetrics, error)

	// Health and status
	HealthCheck(ctx context.Context) error
}

// CompletionOptions represents options for AI completion requests
type CompletionOptions struct {
	Model       string            `json:"model,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float32           `json:"temperature,omitempty"`
	TopP        float32           `json:"top_p,omitempty"`
	Stop        []string          `json:"stop,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// CompletionResponse represents the response from AI completion
type CompletionResponse struct {
	Content     string                 `json:"content"`
	Model       string                 `json:"model"`
	Usage       *UsageStats            `json:"usage,omitempty"`
	Quality     *UnifiedQualityMetrics `json:"quality,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// UsageStats represents token usage statistics
type UsageStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	Total            int `json:"total"`
}

// AIProvider represents different AI service providers
type AIProvider string

const (
	ProviderOpenAI AIProvider = "openai"
	ProviderClaude AIProvider = "claude"
	ProviderMock   AIProvider = "mock"
)

// AIConfig represents configuration for AI services
type AIConfig struct {
	Provider   AIProvider        `json:"provider"`
	APIKey     string            `json:"api_key,omitempty"`
	BaseURL    string            `json:"base_url,omitempty"`
	Model      string            `json:"model,omitempty"`
	MaxRetries int               `json:"max_retries,omitempty"`
	Timeout    time.Duration     `json:"timeout,omitempty"`
	RateLimit  int               `json:"rate_limit,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Additional types for compatibility

// Service represents the legacy AI service interface
type Service interface {
	GenerateCompletion(ctx context.Context, prompt string, options *CompletionOptions) (*CompletionResponse, error)
	AssessQuality(ctx context.Context, content string) (*UnifiedQualityMetrics, error)
	ProcessRequest(ctx context.Context, req *Request) (*Response, error)
	HealthCheck(ctx context.Context) error
}

// Request represents an AI request structure
type Request struct {
	ID       string                 `json:"id,omitempty"`
	Messages []Message              `json:"messages"`
	Model    string                 `json:"model,omitempty"`
	Options  *CompletionOptions     `json:"options,omitempty"`
	Metadata *RequestMetadata       `json:"metadata,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// Message represents a message in an AI conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents an AI response structure
type Response struct {
	ID           string                 `json:"id"`
	Model        string                 `json:"model"`
	Content      string                 `json:"content"`
	Usage        *UsageStats            `json:"usage,omitempty"`
	TokensUsed   *TokenUsage            `json:"tokens_used,omitempty"` // Legacy compatibility with Total field
	Quality      *UnifiedQualityMetrics `json:"quality,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	FallbackUsed bool                   `json:"fallback_used,omitempty"`
	Latency      int64                  `json:"latency,omitempty"`   // Response latency in milliseconds
	CacheHit     bool                   `json:"cache_hit,omitempty"` // Whether response came from cache
}

// RateLimits represents rate limiting information
type RateLimits struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	TokensPerMinute   int `json:"tokens_per_minute"`
	RequestsPerDay    int `json:"requests_per_day"`
}

// Model represents AI model information
type Model string

const (
	ModelClaude     Model = "claude-3-haiku-20240307"
	ModelOpenAI     Model = "gpt-3.5-turbo"
	ModelPerplexity Model = "llama-3.1-sonar-small-128k-online"
)

// TokenUsage represents token usage statistics (legacy compatibility)
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	Total            int `json:"total"` // Legacy field for compatibility
}

// CompletionRequest represents a request to an AI model (legacy compatibility)
type CompletionRequest struct {
	Prompt        string                 `json:"prompt"`
	Model         string                 `json:"model"`
	SystemMessage string                 `json:"system_message,omitempty"`
	MaxTokens     int                    `json:"max_tokens,omitempty"`
	Temperature   float64                `json:"temperature,omitempty"`
	TopP          float64                `json:"top_p,omitempty"`
	StopSequences []string               `json:"stop_sequences,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timeout       time.Duration          `json:"timeout,omitempty"`
}

// ClientCapabilities represents AI client capabilities
type ClientCapabilities struct {
	MaxTokens             int      `json:"max_tokens"`
	SupportedModels       []string `json:"supported_models"`
	SupportsStreaming     bool     `json:"supports_streaming"`
	SupportsSystemMsg     bool     `json:"supports_system_message"`
	SupportsSystemMessage bool     `json:"supports_system_message_legacy"` // For backward compatibility
	SupportsJSONMode      bool     `json:"supports_json_mode"`
	SupportsToolCalling   bool     `json:"supports_tool_calling"`
	Provider              string   `json:"provider,omitempty"`
}

// Client represents an AI client interface (legacy compatibility)
type Client interface {
	Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error)
	ValidateRequest(request CompletionRequest) error
	GetCapabilities() ClientCapabilities
	ProcessRequest(ctx context.Context, req *Request) (*Response, error)
	IsHealthy() bool
}

// RequestMetadata represents metadata for AI requests
type RequestMetadata struct {
	RequestID  string                 `json:"request_id"`
	SessionID  string                 `json:"session_id"`
	UserID     string                 `json:"user_id"`
	Source     string                 `json:"source"`
	Repository string                 `json:"repository"`
	Priority   string                 `json:"priority"`
	Tags       []string               `json:"tags"`
	Context    map[string]interface{} `json:"context"`
	CreatedAt  time.Time              `json:"created_at"`
}

// ResponseMetadata represents metadata for AI responses
type ResponseMetadata struct {
	ProcessedAt time.Time `json:"processed_at"`
	ServerID    string    `json:"server_id"`
	Version     string    `json:"version"`
}
