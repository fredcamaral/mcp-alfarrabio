package ai

import (
	"encoding/json"
	"fmt"
	"time"
)

// OpenAIStyleResponse represents the common structure for OpenAI-style responses
type OpenAIStyleResponse struct {
	ID      string              `json:"id"`
	Model   string              `json:"model"`
	Choices []OpenAIStyleChoice `json:"choices"`
	Usage   OpenAIStyleUsage    `json:"usage"`
	Error   *OpenAIStyleError   `json:"error,omitempty"`
}

// OpenAIStyleChoice represents a choice in an OpenAI-style response
type OpenAIStyleChoice struct {
	Message      OpenAIStyleMessage `json:"message"`
	FinishReason string             `json:"finish_reason"`
}

// OpenAIStyleMessage represents a message in an OpenAI-style response
type OpenAIStyleMessage struct {
	Content string `json:"content"`
}

// OpenAIStyleUsage represents usage statistics in an OpenAI-style response
type OpenAIStyleUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAIStyleError represents an error in an OpenAI-style response
type OpenAIStyleError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// OpenAIStyleConverter provides common conversion logic for OpenAI-style responses
type OpenAIStyleConverter struct {
	ProviderName string
}

// ConvertResponse converts an OpenAI-style response to our internal format
func (c *OpenAIStyleConverter) ConvertResponse(data []byte, startTime time.Time) (*CompletionResponse, error) {
	var resp OpenAIStyleResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", c.ProviderName, err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%s API error: %s", c.ProviderName, resp.Error.Message)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in %s response", c.ProviderName)
	}

	choice := resp.Choices[0]
	return &CompletionResponse{
		ID:           resp.ID,
		Content:      choice.Message.Content,
		Model:        resp.Model,
		FinishReason: choice.FinishReason,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		ProcessingTime: time.Since(startTime),
		Provider:       c.ProviderName,
		CreatedAt:      time.Now(),
	}, nil
}
