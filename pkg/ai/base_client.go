// Package ai provides common base functionality for AI model clients.
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

// BaseConfig represents common configuration for all AI clients
type BaseConfig struct {
	APIKey      string        `json:"api_key"`
	BaseURL     string        `json:"base_url"`
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
	Enabled     bool          `json:"enabled"`
}

// AuthProvider defines how to add authentication to HTTP requests
type AuthProvider interface {
	AddAuth(req *http.Request, apiKey string)
}

// RequestConverter defines how to convert internal requests to provider format
type RequestConverter interface {
	ConvertRequest(req *CompletionRequest, cfg *BaseConfig) (interface{}, error)
}

// ResponseConverter defines how to convert provider responses to internal format
type ResponseConverter interface {
	ConvertResponse(data []byte, startTime time.Time) (*CompletionResponse, error)
}

// BaseClient provides common functionality for all AI clients
type BaseClient struct {
	config            *BaseConfig
	httpClient        *http.Client
	authProvider      AuthProvider
	requestConverter  RequestConverter
	responseConverter ResponseConverter
	maxRetries        int
	retryDelay        time.Duration
}

// NewBaseClient creates a new base client with common functionality
func NewBaseClient(config *BaseConfig, authProvider AuthProvider, requestConverter RequestConverter, responseConverter ResponseConverter) *BaseClient {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	return &BaseClient{
		config:            config,
		httpClient:        &http.Client{Timeout: config.Timeout},
		authProvider:      authProvider,
		requestConverter:  requestConverter,
		responseConverter: responseConverter,
		maxRetries:        3,
		retryDelay:        1 * time.Second,
	}
}

// Complete sends a completion request using the configured provider
func (bc *BaseClient) Complete(ctx context.Context, request *CompletionRequest) (*CompletionResponse, error) {
	if !bc.config.Enabled {
		return nil, errors.New("AI client is disabled")
	}

	// Convert request to provider format
	providerRequest, err := bc.requestConverter.ConvertRequest(request, bc.config)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Serialize request
	requestBody, err := json.Marshal(providerRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request with retries
	startTime := time.Now()
	var lastErr error

	for attempt := 0; attempt < bc.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(bc.retryDelay):
			}
		}

		response, err := bc.makeHTTPRequest(ctx, requestBody)
		if err != nil {
			lastErr = err
			continue
		}

		// Convert response
		return bc.responseConverter.ConvertResponse(response, startTime)
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", bc.maxRetries, lastErr)
}

// makeHTTPRequest makes an HTTP request to the AI provider
func (bc *BaseClient) makeHTTPRequest(ctx context.Context, requestBody []byte) ([]byte, error) {
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", bc.config.BaseURL+"/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "lerian-mcp-memory/1.0")

	// Add authentication
	bc.authProvider.AddAuth(req, bc.config.APIKey)

	// Make request
	resp, err := bc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log error but don't fail the operation since we already have the response
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// Test tests the connection to the AI provider
func (bc *BaseClient) Test(ctx context.Context) error {
	// Simple test request
	testRequest := CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 10,
	}

	_, err := bc.Complete(ctx, &testRequest)
	return err
}

// GetConfig returns the client configuration
func (bc *BaseClient) GetConfig() *BaseConfig {
	return bc.config
}
