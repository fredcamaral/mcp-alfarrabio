// Package ai provides Perplexity Sonar Pro client implementation.
package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultPerplexityBaseURL = "https://api.perplexity.ai"
	defaultPerplexityModel   = "llama-3.1-sonar-large-128k-online"
)

// PerplexityClient implements the AIClient interface for Perplexity API
type PerplexityClient struct {
	*BaseClient
}

// NewPerplexityClient creates a new Perplexity client
func NewPerplexityClient(apiKey string, model string) (*PerplexityClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Perplexity API key cannot be empty")
	}

	if model == "" {
		model = defaultPerplexityModel
	}

	config := &BaseConfig{
		APIKey:      apiKey,
		BaseURL:     defaultPerplexityBaseURL,
		Model:       model,
		MaxTokens:   4096,
		Temperature: 0.7,
		Timeout:     60 * time.Second,
		Enabled:     true,
	}

	authProvider := &PerplexityAuthProvider{}
	requestConverter := &PerplexityRequestConverter{}
	responseConverter := &PerplexityResponseConverter{}

	baseClient := NewBaseClient(config, authProvider, requestConverter, responseConverter)

	return &PerplexityClient{
		BaseClient: baseClient,
	}, nil
}

// PerplexityAuthProvider implements authentication for Perplexity API
type PerplexityAuthProvider struct{}

func (p *PerplexityAuthProvider) AddAuth(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
}

// PerplexityRequestConverter converts internal requests to Perplexity format
type PerplexityRequestConverter struct{}

func (c *PerplexityRequestConverter) ConvertRequest(req *CompletionRequest, cfg *BaseConfig) (interface{}, error) {
	messages := make([]perplexityMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = perplexityMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = cfg.MaxTokens
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = cfg.Temperature
	}

	return perplexityRequest{
		Model:       cfg.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Stream:      false,
	}, nil
}

// PerplexityResponseConverter converts Perplexity responses to internal format
type PerplexityResponseConverter struct{}

func (c *PerplexityResponseConverter) ConvertResponse(data []byte, startTime time.Time) (*CompletionResponse, error) {
	var resp perplexityResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Perplexity response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("Perplexity API error: %s", resp.Error.Message)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in Perplexity response")
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
		Provider:       "perplexity",
		CreatedAt:      time.Now(),
	}, nil
}

// Perplexity API types
type perplexityRequest struct {
	Model       string              `json:"model"`
	Messages    []perplexityMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature"`
	Stream      bool                `json:"stream"`
}

type perplexityMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type perplexityResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []perplexityChoice `json:"choices"`
	Usage   perplexityUsage    `json:"usage"`
	Error   *perplexityError   `json:"error,omitempty"`
}

type perplexityChoice struct {
	Index        int               `json:"index"`
	Message      perplexityMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
}

type perplexityUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type perplexityError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}
