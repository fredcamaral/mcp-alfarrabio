// Package ai provides adapter to connect OpenAIClient to the Service interface
package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/config"
)

// OpenAIAdapter adapts the simple OpenAIClient to the Service Client interface
type OpenAIAdapter struct {
	client *OpenAIClient
	model  string
}

// NewOpenAIAdapter creates a new adapter for OpenAI
func NewOpenAIAdapter(cfg *config.OpenAIClientConfig) (*OpenAIAdapter, error) {
	// Use API key from config or environment
	apiKey := cfg.APIKey
	if apiKey == "" {
		return nil, errors.New("OpenAI API key is required")
	}

	// Use base URL from config or default
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	// Create the underlying client
	client, err := NewOpenAIClient(apiKey, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	// Use model from config or default
	model := cfg.Model
	if model == "" {
		model = "gpt-4o"
	}

	return &OpenAIAdapter{
		client: client,
		model:  model,
	}, nil
}

// ProcessRequest implements the Client interface
func (a *OpenAIAdapter) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
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
	resp, err := a.client.Complete(ctx, &completionReq)
	if err != nil {
		return nil, err
	}
	latency := time.Since(startTime)

	// Convert CompletionResponse to Response
	return &Response{
		ID:      req.ID,
		Model:   string(ModelOpenAI),
		Content: resp.Content,
		TokensUsed: &TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.Total,
			Total:            resp.Usage.Total,
		},
		Usage: resp.Usage,
		Quality: &UnifiedQualityMetrics{
			Confidence:   0.9, // Default high confidence for OpenAI
			Relevance:    0.9,
			Clarity:      0.9,
			Completeness: 0.85,
			Score:        0.9,
			OverallScore: 0.89,
		},
		Metadata: map[string]interface{}{
			"processed_at": time.Now(),
			"server_id":    "openai-adapter",
			"version":      "1.0.0",
			"latency_ms":   latency.Milliseconds(),
		},
	}, nil
}

// GetModel implements the Client interface
func (a *OpenAIAdapter) GetModel() Model {
	return ModelOpenAI
}

// IsHealthy implements the Client interface
func (a *OpenAIAdapter) IsHealthy(ctx context.Context) error {
	// Simple health check - try a minimal completion
	req := CompletionRequest{
		Prompt:    "test",
		Model:     a.model,
		MaxTokens: 1,
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := a.client.Complete(ctx, &req)
	return err
}

// GetLimits implements the Client interface
func (a *OpenAIAdapter) GetLimits() RateLimits {
	return RateLimits{
		RequestsPerMinute: 500,    // OpenAI GPT-4 tier limits
		TokensPerMinute:   200000, // Approximate for GPT-4
		RequestsPerDay:    10000,  // Daily limit estimate
	}
}
