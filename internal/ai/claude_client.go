// Package ai provides Claude Sonnet 4 client implementation.
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

// ClaudeConfig represents Claude API configuration
type ClaudeConfig struct {
	APIKey      string        `json:"api_key"`
	BaseURL     string        `json:"base_url"`
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
	Enabled     bool          `json:"enabled"`
}

// ClaudeClient implements AI client interface for Claude Sonnet 4
type ClaudeClient struct {
	config     ClaudeConfig
	httpClient *http.Client
	rateLimits RateLimits
}

// claudeRequest represents the structure for Claude API requests
type claudeRequest struct {
	Model       string          `json:"model"`
	Messages    []claudeMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature,omitempty"`
	System      string          `json:"system,omitempty"`
}

// claudeMessage represents a message in Claude format
type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse represents the structure for Claude API responses
type claudeResponse struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Role    string         `json:"role"`
	Content []claudeContent `json:"content"`
	Model   string         `json:"model"`
	Usage   claudeUsage    `json:"usage"`
	Error   *claudeError   `json:"error,omitempty"`
}

// claudeContent represents content in Claude response
type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// claudeUsage represents token usage in Claude response
type claudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// claudeError represents error in Claude response
type claudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewClaudeClient creates a new Claude API client
func NewClaudeClient(cfg config.ClaudeClientConfig) (*ClaudeClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Claude API key is required")
	}

	claudeConfig := ClaudeConfig{
		APIKey:      cfg.APIKey,
		BaseURL:     cfg.BaseURL,
		Model:       cfg.Model,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
		Timeout:     cfg.Timeout,
		Enabled:     cfg.Enabled,
	}

	// Set defaults
	if claudeConfig.BaseURL == "" {
		claudeConfig.BaseURL = "https://api.anthropic.com/v1/messages"
	}
	if claudeConfig.Model == "" {
		claudeConfig.Model = "claude-3-5-sonnet-20241022"
	}
	if claudeConfig.MaxTokens == 0 {
		claudeConfig.MaxTokens = 4000
	}
	if claudeConfig.Timeout == 0 {
		claudeConfig.Timeout = 30 * time.Second
	}

	httpClient := &http.Client{
		Timeout: claudeConfig.Timeout,
	}

	rateLimits := RateLimits{
		RequestsPerMinute: 50,  // Claude rate limits
		TokensPerMinute:   100000,
		ResetTime:         time.Minute,
	}

	return &ClaudeClient{
		config:     claudeConfig,
		httpClient: httpClient,
		rateLimits: rateLimits,
	}, nil
}

// ProcessRequest processes a request using Claude API
func (c *ClaudeClient) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	startTime := time.Now()

	// Convert to Claude format
	claudeReq, err := c.convertToClaudeRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make API call
	claudeResp, err := c.makeAPICall(ctx, claudeReq)
	if err != nil {
		return nil, fmt.Errorf("Claude API call failed: %w", err)
	}

	// Convert response
	response, err := c.convertFromClaudeResponse(claudeResp, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return response, nil
}

// convertToClaudeRequest converts internal request to Claude format
func (c *ClaudeClient) convertToClaudeRequest(req *Request) (*claudeRequest, error) {
	claudeMessages := make([]claudeMessage, len(req.Messages))
	for i, msg := range req.Messages {
		claudeMessages[i] = claudeMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	return &claudeRequest{
		Model:       c.config.Model,
		Messages:    claudeMessages,
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
	}, nil
}

// makeAPICall makes the actual HTTP call to Claude API
func (c *ClaudeClient) makeAPICall(ctx context.Context, req *claudeRequest) (*claudeResponse, error) {
	// Serialize request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Make request
	resp, err := c.httpClient.Do(httpReq)
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
		return nil, fmt.Errorf("Claude API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var claudeResp claudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API errors
	if claudeResp.Error != nil {
		return nil, fmt.Errorf("Claude API error: %s", claudeResp.Error.Message)
	}

	return &claudeResp, nil
}

// convertFromClaudeResponse converts Claude response to internal format
func (c *ClaudeClient) convertFromClaudeResponse(claudeResp *claudeResponse, startTime time.Time) (*Response, error) {
	// Extract content text
	var content string
	if len(claudeResp.Content) > 0 {
		content = claudeResp.Content[0].Text
	}

	// Calculate quality metrics (simplified for now)
	quality := QualityMetrics{
		Confidence: 0.9, // Claude typically has high confidence
		Relevance:  0.85,
		Clarity:    0.9,
		Score:      0.88,
	}

	return &Response{
		ID:       claudeResp.ID,
		Model:    ModelClaude,
		Content:  content,
		TokensUsed: TokenUsage{
			Input:  claudeResp.Usage.InputTokens,
			Output: claudeResp.Usage.OutputTokens,
			Total:  claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
		Latency:      time.Since(startTime),
		CacheHit:     false,
		FallbackUsed: false,
		Quality:      quality,
		Metadata: ResponseMetadata{
			ProcessedAt: time.Now(),
			ServerID:    "claude-client",
			Version:     "1.0.0",
		},
	}, nil
}

// GetModel returns the model identifier
func (c *ClaudeClient) GetModel() Model {
	return ModelClaude
}

// IsHealthy checks if the Claude client is operational
func (c *ClaudeClient) IsHealthy(ctx context.Context) error {
	// Simple health check - try a minimal request
	healthReq := &claudeRequest{
		Model:     c.config.Model,
		Messages:  []claudeMessage{{Role: "user", Content: "Hello"}},
		MaxTokens: 10,
	}

	// Use a shorter timeout for health checks
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.makeAPICall(healthCtx, healthReq)
	return err
}

// GetLimits returns rate limiting information
func (c *ClaudeClient) GetLimits() RateLimits {
	return c.rateLimits
}