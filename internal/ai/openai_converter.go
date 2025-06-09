// Package ai provides OpenAI-specific request/response conversion.
package ai

import (
	"encoding/json"
	"fmt"
	"time"
)

// OpenAIRequestConverter implements RequestConverter for OpenAI GPT API
type OpenAIRequestConverter struct{}

// OpenAIResponseConverter implements ResponseConverter for OpenAI GPT API
type OpenAIResponseConverter struct{}

// openaiGPTRequest represents the structure for OpenAI GPT API requests
type openaiGPTRequest struct {
	Model       string             `json:"model"`
	Messages    []openaiGPTMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	Temperature float64            `json:"temperature,omitempty"`
	Stream      bool               `json:"stream"`
}

// openaiGPTMessage represents a message in OpenAI GPT format
type openaiGPTMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiGPTResponse represents the structure for OpenAI GPT API responses
type openaiGPTResponse struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []openaiGPTChoice `json:"choices"`
	Usage   openaiGPTUsage    `json:"usage"`
	Error   *openaiGPTError   `json:"error,omitempty"`
}

// openaiGPTChoice represents a choice in OpenAI GPT response
type openaiGPTChoice struct {
	Index        int              `json:"index"`
	Message      openaiGPTMessage `json:"message"`
	FinishReason string           `json:"finish_reason"`
}

// openaiGPTUsage represents token usage in OpenAI GPT response
type openaiGPTUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// openaiGPTError represents error in OpenAI GPT response
type openaiGPTError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ConvertRequest converts internal request to OpenAI GPT format
func (o *OpenAIRequestConverter) ConvertRequest(req *Request, cfg *BaseConfig) (interface{}, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	openaiMessages := make([]openaiGPTMessage, len(req.Messages))
	for i, msg := range req.Messages {
		openaiMessages[i] = openaiGPTMessage(msg)
	}

	return &openaiGPTRequest{
		Model:       cfg.Model,
		Messages:    openaiMessages,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Stream:      false, // We don't support streaming for now
	}, nil
}

// ConvertResponse converts OpenAI GPT response to internal format
func (o *OpenAIResponseConverter) ConvertResponse(data []byte, startTime time.Time) (*Response, error) {
	var openaiResp openaiGPTResponse
	if err := json.Unmarshal(data, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAI response: %w", err)
	}

	// Check for API errors
	if openaiResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", openaiResp.Error.Message)
	}

	// Extract content text from first choice
	var content string
	if len(openaiResp.Choices) > 0 {
		content = openaiResp.Choices[0].Message.Content
	}

	// Calculate quality metrics (simplified for now)
	quality := QualityMetrics{
		Confidence: 0.9,  // GPT-4o typically has high confidence
		Relevance:  0.85, // Good relevance
		Clarity:    0.9,  // Very clear responses
		Score:      0.88,
	}

	return &Response{
		ID:      openaiResp.ID,
		Model:   ModelOpenAI,
		Content: content,
		TokensUsed: TokenUsage{
			Input:  openaiResp.Usage.PromptTokens,
			Output: openaiResp.Usage.CompletionTokens,
			Total:  openaiResp.Usage.TotalTokens,
		},
		Latency:      time.Since(startTime),
		CacheHit:     false,
		FallbackUsed: false,
		Quality:      quality,
		Metadata: ResponseMetadata{
			ProcessedAt: time.Now(),
			ServerID:    "openai-client",
			Version:     "1.0.0",
		},
	}, nil
}
