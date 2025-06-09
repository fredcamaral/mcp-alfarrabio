// Package ai provides Perplexity-specific request/response conversion.
package ai

import (
	"encoding/json"
	"fmt"
	"time"
)

// PerplexityRequestConverter implements RequestConverter for Perplexity API
type PerplexityRequestConverter struct{}

// PerplexityResponseConverter implements ResponseConverter for Perplexity API
type PerplexityResponseConverter struct{}

// perplexityRequest represents the structure for Perplexity API requests
type perplexityRequest struct {
	Model       string              `json:"model"`
	Messages    []perplexityMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
	Temperature float64             `json:"temperature,omitempty"`
	Stream      bool                `json:"stream"`
}

// perplexityMessage represents a message in Perplexity format
type perplexityMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// perplexityResponse represents the structure for Perplexity API responses
type perplexityResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []perplexityChoice `json:"choices"`
	Usage   perplexityUsage    `json:"usage"`
	Error   *perplexityError   `json:"error,omitempty"`
}

// perplexityChoice represents a choice in Perplexity response
type perplexityChoice struct {
	Index        int               `json:"index"`
	Message      perplexityMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

// perplexityUsage represents token usage in Perplexity response
type perplexityUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// perplexityError represents error in Perplexity response
type perplexityError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ConvertRequest converts internal request to Perplexity format
func (p *PerplexityRequestConverter) ConvertRequest(req *Request, cfg *BaseConfig) (interface{}, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	perplexityMessages := make([]perplexityMessage, len(req.Messages))
	for i, msg := range req.Messages {
		perplexityMessages[i] = perplexityMessage(msg)
	}

	return &perplexityRequest{
		Model:       cfg.Model,
		Messages:    perplexityMessages,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Stream:      false, // We don't support streaming for now
	}, nil
}

// ConvertResponse converts Perplexity response to internal format
func (p *PerplexityResponseConverter) ConvertResponse(data []byte, startTime time.Time) (*Response, error) {
	var perplexityResp perplexityResponse
	if err := json.Unmarshal(data, &perplexityResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Perplexity response: %w", err)
	}

	// Check for API errors
	if perplexityResp.Error != nil {
		return nil, fmt.Errorf("perplexity API error: %s", perplexityResp.Error.Message)
	}

	// Extract content text from first choice
	var content string
	if len(perplexityResp.Choices) > 0 {
		content = perplexityResp.Choices[0].Message.Content
	}

	// Calculate quality metrics (simplified for now)
	quality := QualityMetrics{
		Confidence: 0.85, // Perplexity typically has good confidence
		Relevance:  0.9,  // Strong at online information
		Clarity:    0.8,
		Score:      0.85,
	}

	return &Response{
		ID:      perplexityResp.ID,
		Model:   ModelPerplexity,
		Content: content,
		TokensUsed: TokenUsage{
			Input:  perplexityResp.Usage.PromptTokens,
			Output: perplexityResp.Usage.CompletionTokens,
			Total:  perplexityResp.Usage.TotalTokens,
		},
		Latency:      time.Since(startTime),
		CacheHit:     false,
		FallbackUsed: false,
		Quality:      quality,
		Metadata: ResponseMetadata{
			ProcessedAt: time.Now(),
			ServerID:    "perplexity-client",
			Version:     "1.0.0",
		},
	}, nil
}
