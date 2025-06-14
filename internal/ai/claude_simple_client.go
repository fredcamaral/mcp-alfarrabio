// Package ai provides Claude client implementation.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultClaudeBaseURL = "https://api.anthropic.com/v1"
	claudeTimeout        = 60 * time.Second
	anthropicVersion     = "2023-06-01"
)

// ClaudeSimpleClient implements the AIClient interface for Anthropic Claude API
type ClaudeSimpleClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
}

// NewClaudeSimpleClient creates a new Claude client with simple API key and base URL
func NewClaudeSimpleClient(apiKey, baseURL string) (*ClaudeSimpleClient, error) {
	if apiKey == "" {
		return nil, errors.New("API key cannot be empty")
	}

	if baseURL == "" {
		baseURL = defaultClaudeBaseURL
	}

	return &ClaudeSimpleClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: claudeTimeout,
		},
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}, nil
}

// Complete sends a completion request to Claude
func (c *ClaudeSimpleClient) Complete(ctx context.Context, request *CompletionRequest) (*CompletionResponse, error) {
	if err := c.ValidateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	processedRequest := c.applyDefaults(request)
	ctx = c.applyTimeout(ctx, processedRequest)

	body := c.buildRequestBody(processedRequest)

	return c.executeWithRetry(ctx, body, processedRequest)
}

// applyDefaults applies default values to the request
func (c *ClaudeSimpleClient) applyDefaults(request *CompletionRequest) *CompletionRequest {
	if request.MaxTokens == 0 {
		request.MaxTokens = 1000
	}
	if request.Temperature == 0 {
		request.Temperature = 0.7
	}
	return request
}

// applyTimeout applies timeout to context if specified
func (c *ClaudeSimpleClient) applyTimeout(ctx context.Context, request *CompletionRequest) context.Context {
	if request.Timeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, request.Timeout)
		// Note: In production, you'd want to handle this cancel func properly
		_ = cancel
		return timeoutCtx
	}
	return ctx
}

// buildRequestBody builds the request body for Claude API
func (c *ClaudeSimpleClient) buildRequestBody(request *CompletionRequest) map[string]interface{} {
	messages := c.buildMessages(request)

	body := map[string]interface{}{
		"model":       request.Model,
		"messages":    messages,
		"max_tokens":  request.MaxTokens,
		"temperature": request.Temperature,
	}

	c.addOptionalParameters(body, request)
	return body
}

// addOptionalParameters adds optional parameters to the request body
func (c *ClaudeSimpleClient) addOptionalParameters(body map[string]interface{}, request *CompletionRequest) {
	if request.TopP > 0 {
		body["top_p"] = request.TopP
	}
	if len(request.StopSequences) > 0 {
		body["stop_sequences"] = request.StopSequences
	}
	if request.SystemMessage != "" {
		body["system"] = request.SystemMessage
	}
}

// executeWithRetry executes the request with retry logic
func (c *ClaudeSimpleClient) executeWithRetry(ctx context.Context, body map[string]interface{}, request *CompletionRequest) (*CompletionResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := c.waitForRetry(ctx, attempt); err != nil {
				return nil, err
			}
		}

		resp, err := c.executeRequest(ctx, body)
		if err == nil {
			return c.processResponse(resp, request)
		}

		c.closeResponseBody(resp)
		lastErr = err

		if !c.isRetryableError(err) {
			break
		}
	}

	return nil, lastErr
}

// waitForRetry waits for the appropriate retry delay
func (c *ClaudeSimpleClient) waitForRetry(ctx context.Context, attempt int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(c.retryDelay * time.Duration(attempt)):
		return nil
	}
}

// closeResponseBody safely closes the response body
func (c *ClaudeSimpleClient) closeResponseBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

// ValidateRequest validates the completion request
func (c *ClaudeSimpleClient) ValidateRequest(request *CompletionRequest) error {
	if request.Prompt == "" {
		return errors.New("prompt cannot be empty")
	}
	if request.Model == "" {
		return errors.New("model cannot be empty")
	}
	if request.MaxTokens < 0 {
		return errors.New("max tokens must be positive")
	}
	if request.Temperature < 0 || request.Temperature > 1 {
		return errors.New("temperature must be between 0 and 1")
	}
	return nil
}

// GetCapabilities returns the capabilities of the Claude client
func (c *ClaudeSimpleClient) GetCapabilities() ClientCapabilities {
	return ClientCapabilities{
		SupportedModels:       []string{"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
		MaxTokens:             200000, // Claude's 200k context window
		SupportsStreaming:     true,
		SupportsSystemMsg:     true,
		SupportsSystemMessage: true, // For backward compatibility
		SupportsJSONMode:      false,
		SupportsToolCalling:   false, // Claude doesn't support function calling yet
		Provider:              "anthropic",
	}
}

// buildMessages constructs the messages array for the Claude API request
func (c *ClaudeSimpleClient) buildMessages(request *CompletionRequest) []map[string]interface{} {
	// Claude uses a different format for messages
	messages := []map[string]interface{}{
		{
			"role":    "user",
			"content": request.Prompt,
		},
	}

	return messages
}

// executeRequest performs the HTTP request to Claude
func (c *ClaudeSimpleClient) executeRequest(ctx context.Context, body map[string]interface{}) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Claude uses different headers
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// processResponse processes the Claude API response
func (c *ClaudeSimpleClient) processResponse(resp *http.Response, request *CompletionRequest) (*CompletionResponse, error) {
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log error but don't fail the operation since we already have the response
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp.StatusCode, body)
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		StopReason string `json:"stop_reason"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, errors.New("no content in response")
	}

	// Extract text content
	var content string
	for _, c := range apiResp.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &CompletionResponse{
		Content: content,
		Model:   request.Model,
		Usage: &UsageStats{
			PromptTokens:     apiResp.Usage.InputTokens,
			CompletionTokens: apiResp.Usage.OutputTokens,
			Total:            apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
		Metadata:    request.Metadata, // Propagate metadata from request
		GeneratedAt: time.Now(),
	}, nil
}

// parseErrorResponse parses error responses from Claude
func (c *ClaudeSimpleClient) parseErrorResponse(statusCode int, body []byte) error {
	var errorResp struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
	}

	return &APIError{
		StatusCode: statusCode,
		Message:    errorResp.Error.Message,
		Type:       errorResp.Error.Type,
		Provider:   "anthropic",
	}
}

// isRetryableError determines if an error is retryable
func (c *ClaudeSimpleClient) isRetryableError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests ||
			apiErr.StatusCode >= 500
	}
	return false
}
