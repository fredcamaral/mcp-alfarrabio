// Package ai provides Perplexity Sonar Pro client implementation.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"lerian-mcp-memory/internal/config"
)

// PerplexityConfig represents Perplexity API configuration
type PerplexityConfig struct {
	APIKey      string        `json:"api_key"`
	BaseURL     string        `json:"base_url"`
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
	Enabled     bool          `json:"enabled"`
}

// PerplexityClient implements AI client interface for Perplexity Sonar Pro
type PerplexityClient struct {
	config     PerplexityConfig
	httpClient *http.Client
	rateLimits RateLimits
}

// perplexityRequest represents the structure for Perplexity API requests
type perplexityRequest struct {
	Model       string                `json:"model"`
	Messages    []perplexityMessage   `json:"messages"`
	MaxTokens   int                   `json:"max_tokens,omitempty"`
	Temperature float64               `json:"temperature,omitempty"`
	Stream      bool                  `json:"stream"`
}

// perplexityMessage represents a message in Perplexity format
type perplexityMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// perplexityResponse represents the structure for Perplexity API responses
type perplexityResponse struct {
	ID      string              `json:"id"`
	Object  string              `json:"object"`
	Created int64               `json:"created"`
	Model   string              `json:"model"`
	Choices []perplexityChoice  `json:"choices"`
	Usage   perplexityUsage     `json:"usage"`
	Error   *perplexityError    `json:"error,omitempty"`
}

// perplexityChoice represents a choice in Perplexity response
type perplexityChoice struct {
	Index        int                   `json:"index"`
	Message      perplexityMessage     `json:"message"`
	FinishReason string                `json:"finish_reason"`
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

// NewPerplexityClient creates a new Perplexity API client
func NewPerplexityClient(cfg config.PerplexityClientConfig) (*PerplexityClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Perplexity API key is required")
	}

	perplexityConfig := PerplexityConfig{
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Timeout:     cfg.Timeout,
		Enabled:     cfg.Enabled,
	}

	// Set defaults
	if perplexityConfig.BaseURL == "" {
		perplexityConfig.BaseURL = "https://api.perplexity.ai/chat/completions"
	}
	if perplexityConfig.Model == "" {
		perplexityConfig.Model = "llama-3.1-sonar-huge-128k-online"
	}
	if perplexityConfig.MaxTokens == 0 {
		perplexityConfig.MaxTokens = 4000
	}
	if perplexityConfig.Timeout == 0 {
		perplexityConfig.Timeout = 30 * time.Second
	}

	httpClient := &http.Client{
		Timeout: perplexityConfig.Timeout,
	}

	rateLimits := RateLimits{
		RequestsPerMinute: 60,  // Perplexity rate limits
		TokensPerMinute:   200000,
		ResetTime:         time.Minute,
	}

	return &PerplexityClient{
		config:     perplexityConfig,
		httpClient: httpClient,
		rateLimits: rateLimits,
	}, nil
}

// ProcessRequest processes a request using Perplexity API
func (p *PerplexityClient) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	startTime := time.Now()

	// Convert to Perplexity format
	perplexityReq, err := p.convertToPerplexityRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make API call
	perplexityResp, err := p.makeAPICall(ctx, perplexityReq)
	if err != nil {
		return nil, fmt.Errorf("Perplexity API call failed: %w", err)
	}

	// Convert response
	response, err := p.convertFromPerplexityResponse(perplexityResp, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return response, nil
}

// convertToPerplexityRequest converts internal request to Perplexity format
func (p *PerplexityClient) convertToPerplexityRequest(req *Request) (*perplexityRequest, error) {
	perplexityMessages := make([]perplexityMessage, len(req.Messages))
	for i, msg := range req.Messages {
		perplexityMessages[i] = perplexityMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return &perplexityRequest{
		Model:       p.config.Model,
		Messages:    perplexityMessages,
		MaxTokens:   p.config.MaxTokens,
		Temperature: p.config.Temperature,
		Stream:      false, // We don't support streaming for now
	}, nil
}

// makeAPICall makes the actual HTTP call to Perplexity API
func (p *PerplexityClient) makeAPICall(ctx context.Context, req *perplexityRequest) (*perplexityResponse, error) {
	// Serialize request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.config.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	// Make request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Perplexity API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var perplexityResp perplexityResponse
	if err := json.Unmarshal(body, &perplexityResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API errors
	if perplexityResp.Error != nil {
		return nil, fmt.Errorf("Perplexity API error: %s", perplexityResp.Error.Message)
	}

	return &perplexityResp, nil
}

// convertFromPerplexityResponse converts Perplexity response to internal format
func (p *PerplexityClient) convertFromPerplexityResponse(perplexityResp *perplexityResponse, startTime time.Time) (*Response, error) {
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
		ID:       perplexityResp.ID,
		Model:    ModelPerplexity,
		Content:  content,
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

// GetModel returns the model identifier
func (p *PerplexityClient) GetModel() Model {
	return ModelPerplexity
}

// IsHealthy checks if the Perplexity client is operational
func (p *PerplexityClient) IsHealthy(ctx context.Context) error {
	// Simple health check - try a minimal request
	healthReq := &perplexityRequest{
		Model:    p.config.Model,
		Messages: []perplexityMessage{{Role: "user", Content: "Hello"}},
		MaxTokens: 10,
		Stream:   false,
	}

	// Use a shorter timeout for health checks
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := p.makeAPICall(healthCtx, healthReq)
	return err
}

// GetLimits returns rate limiting information
func (p *PerplexityClient) GetLimits() RateLimits {
	return p.rateLimits
}