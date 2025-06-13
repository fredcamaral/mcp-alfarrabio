// Package ai provides Perplexity client implementation.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultPerplexityBaseURL = "https://api.perplexity.ai"
	perplexityTimeout        = 60 * time.Second
)

// PerplexitySimpleClient implements the AIClient interface for Perplexity API
type PerplexitySimpleClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
}

// NewPerplexitySimpleClient creates a new Perplexity client with simple API key and base URL
func NewPerplexitySimpleClient(apiKey, baseURL string) (*PerplexitySimpleClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}

	if baseURL == "" {
		baseURL = defaultPerplexityBaseURL
	}

	return &PerplexitySimpleClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: perplexityTimeout,
		},
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}, nil
}

// Complete sends a completion request to Perplexity
func (c *PerplexitySimpleClient) Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error) {
	// Validate request
	if err := c.ValidateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Apply default values
	if request.MaxTokens == 0 {
		request.MaxTokens = 1000
	}
	if request.Temperature == 0 {
		request.Temperature = 0.7
	}
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, request.Timeout)
		defer cancel()
	}

	// Build messages array
	messages := c.buildMessages(request)

	// Build request body - Perplexity uses OpenAI-compatible format
	body := map[string]interface{}{
		"model":       request.Model,
		"messages":    messages,
		"max_tokens":  request.MaxTokens,
		"temperature": request.Temperature,
	}

	if request.TopP > 0 {
		body["top_p"] = request.TopP
	}
	if request.FrequencyPenalty > 0 {
		body["frequency_penalty"] = request.FrequencyPenalty
	}
	if request.PresencePenalty > 0 {
		body["presence_penalty"] = request.PresencePenalty
	}

	// Execute request with retry logic
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
				// Exponential backoff
			}
		}

		resp, err := c.executeRequest(ctx, body)
		if err == nil {
			return c.processResponse(resp, request)
		}

		lastErr = err
		if !c.isRetryableError(err) {
			break
		}
	}

	return nil, lastErr
}

// ValidateRequest validates the completion request
func (c *PerplexitySimpleClient) ValidateRequest(request CompletionRequest) error {
	if request.Prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}
	if request.Model == "" {
		return fmt.Errorf("model cannot be empty")
	}
	if request.MaxTokens < 0 {
		return fmt.Errorf("max tokens must be positive")
	}
	if request.Temperature < 0 || request.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	return nil
}

// GetCapabilities returns the capabilities of the Perplexity client
func (c *PerplexitySimpleClient) GetCapabilities() ClientCapabilities {
	return ClientCapabilities{
		Provider:              "perplexity",
		SupportedModels:       []string{"sonar-small-chat", "sonar-small-online", "sonar-medium-chat", "sonar-medium-online"},
		MaxTokens:             100000, // Perplexity context window
		SupportsStreaming:     true,
		SupportsSystemMessage: true,
		SupportsFunctionCalls: false,
	}
}

// buildMessages constructs the messages array for the API request
func (c *PerplexitySimpleClient) buildMessages(request CompletionRequest) []map[string]string {
	messages := []map[string]string{}

	if request.SystemMessage != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": request.SystemMessage,
		})
	}

	messages = append(messages, map[string]string{
		"role":    "user",
		"content": request.Prompt,
	})

	return messages
}

// executeRequest performs the HTTP request to Perplexity
func (c *PerplexitySimpleClient) executeRequest(ctx context.Context, body map[string]interface{}) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// processResponse processes the Perplexity API response
func (c *PerplexitySimpleClient) processResponse(resp *http.Response, request CompletionRequest) (*CompletionResponse, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp.StatusCode, body)
	}

	// Perplexity uses OpenAI-compatible response format
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &CompletionResponse{
		Content: apiResp.Choices[0].Message.Content,
		Model:   request.Model,
		Usage: TokenUsage{
			Input:  apiResp.Usage.PromptTokens,
			Output: apiResp.Usage.CompletionTokens,
			Total:  apiResp.Usage.TotalTokens,
		},
		Metadata:     request.Metadata,
		FinishReason: apiResp.Choices[0].FinishReason,
		Provider:     "perplexity",
	}, nil
}

// parseErrorResponse parses error responses from Perplexity
func (c *PerplexitySimpleClient) parseErrorResponse(statusCode int, body []byte) error {
	var errorResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
	}

	return &APIError{
		StatusCode: statusCode,
		Message:    errorResp.Error.Message,
		Type:       errorResp.Error.Type,
		Code:       errorResp.Error.Code,
		Provider:   "perplexity",
	}
}

// isRetryableError determines if an error is retryable
func (c *PerplexitySimpleClient) isRetryableError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusTooManyRequests ||
			apiErr.StatusCode >= 500
	}
	return false
}
