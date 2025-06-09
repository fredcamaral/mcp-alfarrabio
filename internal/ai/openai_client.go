// Package ai provides OpenAI GPT-4o client implementation using BaseClient.
package ai

import (
	"fmt"
	"time"

	"lerian-mcp-memory/internal/config"
)

// OpenAIGPTClient implements AI client interface for OpenAI GPT-4o
type OpenAIGPTClient struct {
	*BaseClient
}

// NewOpenAIClient creates a new OpenAI GPT API client
func NewOpenAIClient(cfg *config.OpenAIClientConfig) (*OpenAIGPTClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	// Convert config to BaseConfig
	baseConfig := BaseConfig{
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Timeout:     cfg.Timeout,
		Enabled:     cfg.Enabled,
	}

	// Define OpenAI-specific defaults
	defaults := ProviderDefaults{
		BaseURL:   "https://api.openai.com/v1/chat/completions",
		Model:     "gpt-4o",
		MaxTokens: 4000,
		RateLimits: RateLimits{
			RequestsPerMinute: 500, // OpenAI rate limits for GPT-4o
			TokensPerMinute:   200000,
			ResetTime:         time.Minute,
		},
	}

	// Create converters
	reqConv := &OpenAIRequestConverter{}
	respConv := &OpenAIResponseConverter{}
	auth := &BearerTokenAuth{}

	// Create base client
	baseClient := NewBaseClient(&baseConfig, defaults, auth, reqConv, respConv)

	return &OpenAIGPTClient{
		BaseClient: baseClient,
	}, nil
}

// GetModel returns the model identifier
func (o *OpenAIGPTClient) GetModel() Model {
	return ModelOpenAI
}
