// Package ai provides OpenAI GPT-4o client implementation.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"lerian-mcp-memory/internal/config"
)

// OpenAIGPTConfig represents OpenAI GPT API configuration
type OpenAIGPTConfig struct {
	APIKey      string        `json:"api_key"`
	BaseURL     string        `json:"base_url"`
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
	Enabled     bool          `json:"enabled"`
}

// OpenAIGPTClient implements AI client interface for OpenAI GPT-4o
type OpenAIGPTClient struct {
	config     OpenAIGPTConfig
	httpClient *http.Client
	rateLimits RateLimits
}

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

// NewOpenAIClient creates a new OpenAI GPT API client
func NewOpenAIClient(cfg config.OpenAIClientConfig) (*OpenAIGPTClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	openaiConfig := OpenAIGPTConfig{
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Timeout:     cfg.Timeout,
		Enabled:     cfg.Enabled,
	}

	// Set defaults
	if openaiConfig.BaseURL == "" {
		openaiConfig.BaseURL = "https://api.openai.com/v1/chat/completions"
	}
	if openaiConfig.Model == "" {
		openaiConfig.Model = "gpt-4o"
	}
	if openaiConfig.MaxTokens == 0 {
		openaiConfig.MaxTokens = 4000
	}
	if openaiConfig.Timeout == 0 {
		openaiConfig.Timeout = 30 * time.Second
	}

	httpClient := &http.Client{
		Timeout: openaiConfig.Timeout,
	}

	rateLimits := RateLimits{
		RequestsPerMinute: 500, // OpenAI rate limits for GPT-4o
		TokensPerMinute:   200000,
		ResetTime:         time.Minute,
	}

	return &OpenAIGPTClient{
		config:     openaiConfig,
		httpClient: httpClient,
		rateLimits: rateLimits,
	}, nil
}

// ProcessRequest processes a request using OpenAI GPT API
func (o *OpenAIGPTClient) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	startTime := time.Now()

	// Convert to OpenAI GPT format
	openaiReq, err := o.convertToOpenAIRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make API call
	openaiResp, err := o.makeAPICall(ctx, openaiReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	// Convert response
	response, err := o.convertFromOpenAIResponse(openaiResp, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return response, nil
}

// convertToOpenAIRequest converts internal request to OpenAI GPT format
func (o *OpenAIGPTClient) convertToOpenAIRequest(req *Request) (*openaiGPTRequest, error) {
	openaiMessages := make([]openaiGPTMessage, len(req.Messages))
	for i, msg := range req.Messages {
		openaiMessages[i] = openaiGPTMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return &openaiGPTRequest{
		Model:       o.config.Model,
		Messages:    openaiMessages,
		MaxTokens:   o.config.MaxTokens,
		Temperature: o.config.Temperature,
		Stream:      false, // We don't support streaming for now
	}, nil
}

// makeAPICall makes the actual HTTP call to OpenAI GPT API
func (o *OpenAIGPTClient) makeAPICall(ctx context.Context, req *openaiGPTRequest) (*openaiGPTResponse, error) {
	// Serialize request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.config.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.config.APIKey)

	// Make request
	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openaiResp openaiGPTResponse
	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API errors
	if openaiResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", openaiResp.Error.Message)
	}

	return &openaiResp, nil
}

// convertFromOpenAIResponse converts OpenAI GPT response to internal format
func (o *OpenAIGPTClient) convertFromOpenAIResponse(openaiResp *openaiGPTResponse, startTime time.Time) (*Response, error) {
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

// GetModel returns the model identifier
func (o *OpenAIGPTClient) GetModel() Model {
	return ModelOpenAI
}

// IsHealthy checks if the OpenAI GPT client is operational
func (o *OpenAIGPTClient) IsHealthy(ctx context.Context) error {
	// Simple health check - try a minimal request
	healthReq := &openaiGPTRequest{
		Model:     o.config.Model,
		Messages:  []openaiGPTMessage{{Role: "user", Content: "Hello"}},
		MaxTokens: 10,
		Stream:    false,
	}

	// Use a shorter timeout for health checks
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := o.makeAPICall(healthCtx, healthReq)
	return err
}

// GetLimits returns rate limiting information
func (o *OpenAIGPTClient) GetLimits() RateLimits {
	return o.rateLimits
}
