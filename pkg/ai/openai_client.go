// Package ai provides OpenAI GPT-4o client implementation.
package ai

import (
	"errors"
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
func NewOpenAIClient(apiKey, model string) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, errors.New("OpenAI API key cannot be empty")
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
		messages[i] = openAIMessage(msg)
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
type OpenAIResponseConverter struct {
	*OpenAIStyleConverter
}

// NewOpenAIResponseConverter creates a new OpenAI response converter
func NewOpenAIResponseConverter() *OpenAIResponseConverter {
	return &OpenAIResponseConverter{
		OpenAIStyleConverter: &OpenAIStyleConverter{
			ProviderName: "openai",
		},
	}
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
