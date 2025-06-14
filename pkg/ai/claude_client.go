// Package ai provides Claude Sonnet 4 client implementation.
package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultClaudeBaseURL = "https://api.anthropic.com/v1"
	defaultClaudeModel   = "claude-3-5-sonnet-20241022"
)

// ClaudeClient implements the AIClient interface for Claude API
type ClaudeClient struct {
	*BaseClient
}

// NewClaudeClient creates a new Claude client
func NewClaudeClient(apiKey, model string) (*ClaudeClient, error) {
	if apiKey == "" {
		return nil, errors.New("claude API key cannot be empty")
	}

	if model == "" {
		model = defaultClaudeModel
	}

	config := &BaseConfig{
		APIKey:      apiKey,
		BaseURL:     defaultClaudeBaseURL,
		Model:       model,
		MaxTokens:   4096,
		Temperature: 0.7,
		Timeout:     60 * time.Second,
		Enabled:     true,
	}

	authProvider := &ClaudeAuthProvider{}
	requestConverter := &ClaudeRequestConverter{}
	responseConverter := &ClaudeResponseConverter{}

	baseClient := NewBaseClient(config, authProvider, requestConverter, responseConverter)

	return &ClaudeClient{
		BaseClient: baseClient,
	}, nil
}

// ClaudeAuthProvider implements authentication for Claude API
type ClaudeAuthProvider struct{}

func (p *ClaudeAuthProvider) AddAuth(req *http.Request, apiKey string) {
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
}

// ClaudeRequestConverter converts internal requests to Claude format
type ClaudeRequestConverter struct{}

func (c *ClaudeRequestConverter) ConvertRequest(req *CompletionRequest, cfg *BaseConfig) (interface{}, error) {
	messages := make([]claudeMessage, 0, len(req.Messages))
	var systemMessage string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemMessage = msg.Content
		} else {
			messages = append(messages, claudeMessage(msg))
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

	claudeReq := claudeRequest{
		Model:       cfg.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	if systemMessage != "" {
		claudeReq.System = systemMessage
	}

	return claudeReq, nil
}

// ClaudeResponseConverter converts Claude responses to internal format
type ClaudeResponseConverter struct{}

func (c *ClaudeResponseConverter) ConvertResponse(data []byte, startTime time.Time) (*CompletionResponse, error) {
	var resp claudeResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Claude response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("claude API error: %s", resp.Error.Message)
	}

	if len(resp.Content) == 0 {
		return nil, errors.New("no content in Claude response")
	}

	// Extract text content
	var content string
	for _, c := range resp.Content {
		if c.Type == "text" {
			content = c.Text
			break
		}
	}

	return &CompletionResponse{
		ID:           resp.ID,
		Content:      content,
		Model:        resp.Model,
		FinishReason: resp.StopReason,
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		ProcessingTime: time.Since(startTime),
		Provider:       "claude",
		CreatedAt:      time.Now(),
	}, nil
}

// Claude API types
type claudeRequest struct {
	Model       string          `json:"model"`
	Messages    []claudeMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature,omitempty"`
	System      string          `json:"system,omitempty"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Role       string          `json:"role"`
	Content    []claudeContent `json:"content"`
	Model      string          `json:"model"`
	StopReason string          `json:"stop_reason"`
	Usage      claudeUsage     `json:"usage"`
	Error      *claudeError    `json:"error,omitempty"`
}

type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type claudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
