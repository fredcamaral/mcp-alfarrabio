// Package ai provides adapter to connect ClaudeSimpleClient to the Service interface
package ai

import (
	"context"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/config"
)

// ClaudeAdapter adapts the simple ClaudeSimpleClient to the Service Client interface
type ClaudeAdapter struct {
	client *ClaudeSimpleClient
	model  string
}

// NewClaudeAdapter creates a new adapter for Claude
func NewClaudeAdapter(cfg *config.ClaudeClientConfig) (*ClaudeAdapter, error) {
	// Use API key from config or environment
	apiKey := cfg.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("claude API key is required")
	}

	// Use base URL from config or default
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultClaudeBaseURL
	}

	// Create the underlying client
	client, err := NewClaudeSimpleClient(apiKey, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Claude client: %w", err)
	}

	// Use model from config or default
	model := cfg.Model
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}

	return &ClaudeAdapter{
		client: client,
		model:  model,
	}, nil
}

// ProcessRequest implements the Client interface
func (a *ClaudeAdapter) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
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
	resp, err := a.client.Complete(ctx, &completionReq)
	if err != nil {
		return nil, err
	}
	latency := time.Since(startTime)

	// Convert CompletionResponse to Response
	return &Response{
		ID:      req.ID,
		Model:   string(ModelClaude),
		Content: resp.Content,
		TokensUsed: &TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.Total,
			Total:            resp.Usage.Total,
		},
		Usage: &UsageStats{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			Total:            resp.Usage.Total,
		},
		Quality: &UnifiedQualityMetrics{
			Confidence:   0.95, // Claude typically has high confidence
			Relevance:    0.9,
			Clarity:      0.95,
			Score:        0.93,
			Completeness: 0.9,
			OverallScore: 0.93,
		},
		Metadata: map[string]interface{}{
			"processed_at": time.Now(),
			"server_id":    "claude-adapter",
			"version":      "1.0.0",
			"latency_ms":   latency.Milliseconds(),
		},
	}, nil
}

// GetModel implements the Client interface
func (a *ClaudeAdapter) GetModel() Model {
	return ModelClaude
}

// IsHealthy implements the Client interface
func (a *ClaudeAdapter) IsHealthy(ctx context.Context) error {
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
func (a *ClaudeAdapter) GetLimits() RateLimits {
	return RateLimits{
		RequestsPerMinute: 50,     // Claude rate limits
		TokensPerMinute:   100000, // Approximate for Claude
		RequestsPerDay:    1000,   // Daily limit
	}
}
