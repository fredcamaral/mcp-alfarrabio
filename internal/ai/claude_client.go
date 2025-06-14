// Package ai provides Claude Sonnet 4 client implementation.
package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/config"
)

// ClaudeClient implements AI client interface for Claude Sonnet 4
type ClaudeClient struct {
	*BaseClient
}

// claudeRequest represents the structure for Claude API requests
type claudeRequest struct {
	Model       string          `json:"model"`
	Messages    []claudeMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature,omitempty"`
	System      string          `json:"system,omitempty"`
}

// claudeMessage represents a message in Claude format
type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse represents the structure for Claude API responses
type claudeResponse struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Role    string          `json:"role"`
	Content []claudeContent `json:"content"`
	Model   string          `json:"model"`
	Usage   claudeUsage     `json:"usage"`
	Error   *claudeError    `json:"error,omitempty"`
}

// claudeContent represents content in Claude response
type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// claudeUsage represents token usage in Claude response
type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// claudeError represents error in Claude response
type claudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ClaudeRequestConverter implements RequestConverter for Claude API
type ClaudeRequestConverter struct{}

// ConvertRequest converts internal request to Claude format
func (c *ClaudeRequestConverter) ConvertRequest(req *Request, cfg *BaseConfig) (interface{}, error) {
	claudeMessages := make([]claudeMessage, len(req.Messages))
	for i, msg := range req.Messages {
		claudeMessages[i] = claudeMessage(msg)
	}

	return &claudeRequest{
		Model:       cfg.Model,
		Messages:    claudeMessages,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	}, nil
}

// ClaudeResponseConverter implements ResponseConverter for Claude API
type ClaudeResponseConverter struct{}

// ConvertResponse converts Claude response to internal format
func (c *ClaudeResponseConverter) ConvertResponse(data []byte, startTime time.Time) (*Response, error) {
	var claudeResp claudeResponse
	if err := json.Unmarshal(data, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API errors
	if claudeResp.Error != nil {
		return nil, fmt.Errorf("claude API error: %s", claudeResp.Error.Message)
	}

	// Extract content text
	var content string
	if len(claudeResp.Content) > 0 {
		content = claudeResp.Content[0].Text
	}

	// Calculate quality metrics (simplified for now)
	quality := &UnifiedQualityMetrics{
		Confidence:   0.9, // Claude typically has high confidence
		Relevance:    0.85,
		Clarity:      0.9,
		Score:        0.88,
		Completeness: 0.85,
		OverallScore: 0.88,
	}

	return &Response{
		ID:      claudeResp.ID,
		Model:   string(ModelClaude),
		Content: content,
		TokensUsed: &TokenUsage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
			Total:            claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
		Usage: &UsageStats{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			Total:            claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
		Quality: quality,
		Metadata: map[string]interface{}{
			"processed_at": time.Now(),
			"server_id":    "claude-client",
			"version":      "1.0.0",
			"latency_ms":   time.Since(startTime).Milliseconds(),
		},
	}, nil
}

// NewClaudeClient creates a new Claude API client
func NewClaudeClient(cfg *config.ClaudeClientConfig) (*ClaudeClient, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("claude API key is required")
	}

	baseConfig := BaseConfig{
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Timeout:     cfg.Timeout,
		Enabled:     cfg.Enabled,
	}

	defaults := ProviderDefaults{
		BaseURL:   "https://api.anthropic.com/v1/messages",
		Model:     "claude-3-5-sonnet-20241022",
		MaxTokens: 4000,
		RateLimits: RateLimits{
			RequestsPerMinute: 50, // Claude rate limits
			TokensPerMinute:   100000,
			RequestsPerDay:    1000, // Daily limit
		},
	}

	baseClient := NewBaseClient(
		&baseConfig,
		defaults,
		&ClaudeAuth{},
		&ClaudeRequestConverter{},
		&ClaudeResponseConverter{},
	)

	return &ClaudeClient{
		BaseClient: baseClient,
	}, nil
}

// GetModel returns the model identifier
func (c *ClaudeClient) GetModel() Model {
	return ModelClaude
}
