// Package ai provides types and interfaces for AI client implementations
package ai

import (
	"context"
	"time"
)

// AIClient defines the simplified interface for AI providers
type AIClient interface {
	// Complete sends a completion request to the AI provider
	Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error)

	// ValidateRequest validates a completion request
	ValidateRequest(request CompletionRequest) error

	// GetCapabilities returns the capabilities of the AI provider
	GetCapabilities() ClientCapabilities
}

// CompletionRequest represents a request to an AI model
type CompletionRequest struct {
	// Core fields
	Prompt        string `json:"prompt"`
	Model         string `json:"model"`
	SystemMessage string `json:"system_message,omitempty"`

	// Parameters
	MaxTokens        int     `json:"max_tokens,omitempty"`
	Temperature      float64 `json:"temperature,omitempty"`
	TopP             float64 `json:"top_p,omitempty"`
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`
	PresencePenalty  float64 `json:"presence_penalty,omitempty"`

	// Advanced options
	StopSequences []string               `json:"stop_sequences,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timeout       time.Duration          `json:"timeout,omitempty"`
}

// CompletionResponse represents a response from an AI model
type CompletionResponse struct {
	Content      string                 `json:"content"`
	Model        string                 `json:"model"`
	Usage        TokenUsage             `json:"usage"`
	FinishReason string                 `json:"finish_reason,omitempty"`
	Provider     string                 `json:"provider"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Note: TokenUsage is already defined in service.go

// ClientCapabilities describes what an AI client supports
type ClientCapabilities struct {
	Provider              string   `json:"provider"`
	SupportedModels       []string `json:"supported_models"`
	MaxTokens             int      `json:"max_tokens"`
	SupportsStreaming     bool     `json:"supports_streaming"`
	SupportsSystemMessage bool     `json:"supports_system_message"`
	SupportsFunctionCalls bool     `json:"supports_function_calls"`
}
