// Package ai provides OpenAI GPT-4o client implementation.
package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultOpenAIModel   = "gpt-4o"
)

// OpenAIClient implements the AIClient interface for OpenAI API
type OpenAIClient struct {
	*BaseClient
}

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient(apiKey string, model string) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key cannot be empty")
	}

	if model == "" {
		model = defaultOpenAIModel
	}

	config := &BaseConfig{
		APIKey:      apiKey,
		BaseURL:     defaultOpenAIBaseURL,
		Model:       model,
		MaxTokens:   4096,
		Temperature: 0.7,
		Timeout:     60 * time.Second,
		Enabled:     true,
	}

	authProvider := &OpenAIAuthProvider{}
	requestConverter := &OpenAIRequestConverter{}
	responseConverter := &OpenAIResponseConverter{}

	baseClient := NewBaseClient(config, authProvider, requestConverter, responseConverter)

	return &OpenAIClient{
		BaseClient: baseClient,
	}, nil
}

// OpenAIAuthProvider implements authentication for OpenAI API
type OpenAIAuthProvider struct{}

func (p *OpenAIAuthProvider) AddAuth(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
}

// OpenAIRequestConverter converts internal requests to OpenAI format
type OpenAIRequestConverter struct{}

func (c *OpenAIRequestConverter) ConvertRequest(req *CompletionRequest, cfg *BaseConfig) (interface{}, error) {
	messages := make([]openAIMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openAIMessage{
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

	return openAIRequest{
		Model:       cfg.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}, nil
}

// OpenAIResponseConverter converts OpenAI responses to internal format
type OpenAIResponseConverter struct{}

func (c *OpenAIResponseConverter) ConvertResponse(data []byte, startTime time.Time) (*CompletionResponse, error) {
	var resp openAIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAI response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", resp.Error.Message)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
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
		Provider:       "openai",
		CreatedAt:      time.Now(),
	}, nil
}

// OpenAI API types
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
	Error   *openAIError   `json:"error,omitempty"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}
