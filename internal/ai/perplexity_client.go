// Package ai provides Perplexity Sonar Pro client implementation using BaseClient.
package ai

import (
	"fmt"
	"time"

	"lerian-mcp-memory/internal/config"
)

// PerplexityClient implements AI client interface for Perplexity Sonar Pro
type PerplexityClient struct {
	*BaseClient
}

// NewPerplexityClient creates a new Perplexity API client
func NewPerplexityClient(cfg *config.PerplexityClientConfig) (*PerplexityClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("perplexity API key is required")
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

	// Define Perplexity-specific defaults
	defaults := ProviderDefaults{
		BaseURL:   "https://api.perplexity.ai/chat/completions",
		Model:     "llama-3.1-sonar-huge-128k-online",
		MaxTokens: 4000,
		RateLimits: RateLimits{
			RequestsPerMinute: 60, // Perplexity rate limits
			TokensPerMinute:   200000,
			ResetTime:         time.Minute,
		},
	}

	// Create converters
	reqConv := &PerplexityRequestConverter{}
	respConv := &PerplexityResponseConverter{}
	auth := &BearerTokenAuth{}

	// Create base client
	baseClient := NewBaseClient(&baseConfig, defaults, auth, reqConv, respConv)

	return &PerplexityClient{
		BaseClient: baseClient,
	}, nil
}

// GetModel returns the model identifier
func (p *PerplexityClient) GetModel() Model {
	return ModelPerplexity
}
