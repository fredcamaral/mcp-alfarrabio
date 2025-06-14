// Package ai provides common base functionality for AI model clients.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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
	ConvertRequest(req *Request, cfg *BaseConfig) (interface{}, error)
}

// ResponseConverter defines how to convert provider responses to internal format
type ResponseConverter interface {
	ConvertResponse(data []byte, startTime time.Time) (*Response, error)
}

// ProviderDefaults defines provider-specific default values
type ProviderDefaults struct {
	BaseURL    string
	Model      string
	MaxTokens  int
	RateLimits RateLimits
}

// BaseClient provides common HTTP client functionality for AI providers
type BaseClient struct {
	config     BaseConfig
	httpClient *http.Client
	rateLimits RateLimits
	auth       AuthProvider
	reqConv    RequestConverter
	respConv   ResponseConverter
}

// NewBaseClient creates a new base client with common functionality
func NewBaseClient(
	config *BaseConfig,
	defaults ProviderDefaults,
	auth AuthProvider,
	reqConv RequestConverter,
	respConv ResponseConverter,
) *BaseClient {
	// Apply defaults if not set
	if config.BaseURL == "" {
		config.BaseURL = defaults.BaseURL
	}
	if config.Model == "" {
		config.Model = defaults.Model
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = defaults.MaxTokens
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return &BaseClient{
		config:     *config,
		httpClient: httpClient,
		rateLimits: defaults.RateLimits,
		auth:       auth,
		reqConv:    reqConv,
		respConv:   respConv,
	}
}

// ProcessRequest processes a request using the configured provider
func (b *BaseClient) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	startTime := time.Now()

	// Convert to provider format
	providerReq, err := b.reqConv.ConvertRequest(req, &b.config)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Make API call
	responseData, err := b.makeAPICall(ctx, providerReq)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	// Convert response
	response, err := b.respConv.ConvertResponse(responseData, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return response, nil
}

// makeAPICall makes the actual HTTP call to the provider API
func (b *BaseClient) makeAPICall(ctx context.Context, req interface{}) ([]byte, error) {
	// Serialize request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", b.config.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set common headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Add provider-specific authentication
	b.auth.AddAuth(httpReq, b.config.APIKey)

	// Make request
	resp, err := b.httpClient.Do(httpReq)
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
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// IsHealthy checks if the client is operational with a minimal request
func (b *BaseClient) IsHealthy(ctx context.Context) error {
	// Create a minimal health check request
	healthReq := &Request{
		Messages: []Message{{Role: "user", Content: "Hello"}},
	}

	// Use a shorter timeout for health checks
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Convert to provider format
	providerReq, err := b.reqConv.ConvertRequest(healthReq, &b.config)
	if err != nil {
		return fmt.Errorf("failed to convert health check request: %w", err)
	}

	// Make API call (ignore response content for health check)
	_, err = b.makeAPICall(healthCtx, providerReq)
	return err
}

// GetLimits returns rate limiting information
func (b *BaseClient) GetLimits() RateLimits {
	return b.rateLimits
}

// GetConfig returns the current configuration
func (b *BaseClient) GetConfig() BaseConfig {
	return b.config
}
