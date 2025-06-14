// Package ai provides Perplexity Sonar Pro client implementation.
package ai

import (
	"errors"
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
func NewPerplexityClient(apiKey, model string) (*PerplexityClient, error) {
	if apiKey == "" {
		return nil, errors.New("perplexity API key cannot be empty")
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
		messages[i] = perplexityMessage(msg)
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
type PerplexityResponseConverter struct {
	*OpenAIStyleConverter
}

// NewPerplexityResponseConverter creates a new Perplexity response converter
func NewPerplexityResponseConverter() *PerplexityResponseConverter {
	return &PerplexityResponseConverter{
		OpenAIStyleConverter: &OpenAIStyleConverter{
			ProviderName: "perplexity",
		},
	}
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
