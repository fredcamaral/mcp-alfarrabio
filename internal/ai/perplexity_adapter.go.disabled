// Package ai provides adapter to connect PerplexitySimpleClient to the Service interface
package ai

import (
	"context"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/config"
)

// PerplexityAdapter adapts the simple PerplexitySimpleClient to the Service Client interface
type PerplexityAdapter struct {
	client *PerplexitySimpleClient
	model  string
}

// NewPerplexityAdapter creates a new adapter for Perplexity
func NewPerplexityAdapter(cfg *config.PerplexityClientConfig) (*PerplexityAdapter, error) {
	// Use API key from config or environment
	apiKey := cfg.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("Perplexity API key is required")
	}

	// Use base URL from config or default
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultPerplexityBaseURL
	}

	// Create the underlying client
	client, err := NewPerplexitySimpleClient(apiKey, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Perplexity client: %w", err)
	}

	// Use model from config or default
	model := cfg.Model
	if model == "" {
		model = "sonar-medium-online"
	}

	return &PerplexityAdapter{
		client: client,
		model:  model,
	}, nil
}

// ProcessRequest implements the Client interface
func (a *PerplexityAdapter) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Convert Request to CompletionRequest
	completionReq := CompletionRequest{
		Model: a.model,
	}

	// Build prompt from messages
	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			completionReq.SystemMessage = msg.Content
		case "user", "assistant":
			if completionReq.Prompt != "" {
				completionReq.Prompt += "\n"
			}
			completionReq.Prompt += msg.Content
		}
	}

	// Add metadata
	completionReq.Metadata = map[string]interface{}{
		"repository": req.Metadata.Repository,
		"session_id": req.Metadata.SessionID,
		"user_id":    req.Metadata.UserID,
	}

	// Execute the request
	startTime := time.Now()
	resp, err := a.client.Complete(ctx, completionReq)
	if err != nil {
		return nil, err
	}
	latency := time.Since(startTime)

	// Convert CompletionResponse to Response
	return &Response{
		ID:      req.ID,
		Model:   ModelPerplexity,
		Content: resp.Content,
		TokensUsed: TokenUsage{
			Input:  resp.Usage.Input,
			Output: resp.Usage.Output,
			Total:  resp.Usage.Total,
		},
		Latency:      latency,
		CacheHit:     false,
		FallbackUsed: false,
		Quality: QualityMetrics{
			Confidence: 0.85, // Perplexity with real-time search
			Relevance:  0.95, // High relevance due to search
			Clarity:    0.85,
			Score:      0.88,
		},
		Metadata: ResponseMetadata{
			ProcessedAt: time.Now(),
			ServerID:    "perplexity-adapter",
			Version:     "1.0.0",
		},
	}, nil
}

// GetModel implements the Client interface
func (a *PerplexityAdapter) GetModel() Model {
	return ModelPerplexity
}

// IsHealthy implements the Client interface
func (a *PerplexityAdapter) IsHealthy(ctx context.Context) error {
	// Simple health check - try a minimal completion
	req := CompletionRequest{
		Prompt:    "test",
		Model:     a.model,
		MaxTokens: 1,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := a.client.Complete(ctx, req)
	return err
}

// GetLimits implements the Client interface
func (a *PerplexityAdapter) GetLimits() RateLimits {
	return RateLimits{
		RequestsPerMinute:  100,    // Perplexity rate limits
		TokensPerMinute:    150000, // Approximate for Perplexity
		ConcurrentRequests: 20,
		ResetTime:          time.Minute,
	}
}
