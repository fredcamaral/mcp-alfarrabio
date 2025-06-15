// Package ai provides AI service adapter for HTTP communication with the MCP AI service.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"lerian-mcp-memory-cli/internal/domain/ports"
)

// HTTPAIService implements the AIService interface by communicating with the MCP AI service via HTTP
type HTTPAIService struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	timeout    time.Duration
}

// AIServiceConfig contains configuration for the AI service adapter
type AIServiceConfig struct {
	BaseURL string        `mapstructure:"base_url"`
	APIKey  string        `mapstructure:"api_key"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// NewHTTPAIService creates a new HTTP-based AI service adapter
func NewHTTPAIService(config *AIServiceConfig) *HTTPAIService {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &HTTPAIService{
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
		timeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GeneratePRD generates a PRD using the AI service
func (s *HTTPAIService) GeneratePRD(ctx context.Context, request *ports.PRDGenerationRequest) (*ports.PRDGenerationResponse, error) {
	endpoint := "/ai/generate/prd"

	var response ports.PRDGenerationResponse
	err := s.makeRequest(ctx, "POST", endpoint, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PRD: %w", err)
	}

	return &response, nil
}

// GenerateTRD generates a TRD using the AI service
func (s *HTTPAIService) GenerateTRD(ctx context.Context, request *ports.TRDGenerationRequest) (*ports.TRDGenerationResponse, error) {
	endpoint := "/ai/generate/trd"

	var response ports.TRDGenerationResponse
	err := s.makeRequest(ctx, "POST", endpoint, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TRD: %w", err)
	}

	return &response, nil
}

// GenerateMainTasks generates main tasks using the AI service
func (s *HTTPAIService) GenerateMainTasks(ctx context.Context, request *ports.MainTaskGenerationRequest) (*ports.MainTaskGenerationResponse, error) {
	endpoint := "/ai/generate/main-tasks"

	var response ports.MainTaskGenerationResponse
	err := s.makeRequest(ctx, "POST", endpoint, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to generate main tasks: %w", err)
	}

	return &response, nil
}

// GenerateSubTasks generates sub-tasks using the AI service
func (s *HTTPAIService) GenerateSubTasks(ctx context.Context, request *ports.SubTaskGenerationRequest) (*ports.SubTaskGenerationResponse, error) {
	endpoint := "/ai/generate/sub-tasks"

	var response ports.SubTaskGenerationResponse
	err := s.makeRequest(ctx, "POST", endpoint, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sub-tasks: %w", err)
	}

	return &response, nil
}

// AnalyzeContent analyzes content using the AI service
func (s *HTTPAIService) AnalyzeContent(ctx context.Context, request *ports.ContentAnalysisRequest) (*ports.ContentAnalysisResponse, error) {
	endpoint := "/ai/analyze/content"

	var response ports.ContentAnalysisResponse
	err := s.makeRequest(ctx, "POST", endpoint, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze content: %w", err)
	}

	return &response, nil
}

// EstimateComplexity estimates complexity using the AI service
func (s *HTTPAIService) EstimateComplexity(ctx context.Context, content string) (*ports.ComplexityEstimate, error) {
	endpoint := "/ai/analyze/complexity"

	request := map[string]string{
		"content": content,
	}

	var response ports.ComplexityEstimate
	err := s.makeRequest(ctx, "POST", endpoint, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate complexity: %w", err)
	}

	return &response, nil
}

// StartInteractiveSession starts an interactive AI session
func (s *HTTPAIService) StartInteractiveSession(ctx context.Context, docType string) (*ports.InteractiveSession, error) {
	endpoint := "/ai/session/start"

	request := map[string]string{
		"type": docType,
	}

	var response ports.InteractiveSession
	err := s.makeRequest(ctx, "POST", endpoint, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to start interactive session: %w", err)
	}

	return &response, nil
}

// ContinueSession continues an interactive AI session
func (s *HTTPAIService) ContinueSession(ctx context.Context, sessionID, userInput string) (*ports.SessionResponse, error) {
	endpoint := fmt.Sprintf("/ai/session/%s/continue", sessionID)

	request := map[string]string{
		"input": userInput,
	}

	var response ports.SessionResponse
	err := s.makeRequest(ctx, "POST", endpoint, request, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to continue session: %w", err)
	}

	return &response, nil
}

// EndSession ends an interactive AI session
func (s *HTTPAIService) EndSession(ctx context.Context, sessionID string) error {
	endpoint := fmt.Sprintf("/ai/session/%s/end", sessionID)

	err := s.makeRequest(ctx, "POST", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	return nil
}

// TestConnection tests the connection to the AI service
func (s *HTTPAIService) TestConnection(ctx context.Context) error {
	endpoint := "/health"

	err := s.makeRequest(ctx, "GET", endpoint, nil, nil)
	if err != nil {
		return fmt.Errorf("AI service connection test failed: %w", err)
	}

	return nil
}

// IsOnline checks if the AI service is online
func (s *HTTPAIService) IsOnline() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.TestConnection(ctx) == nil
}

// GetAvailableModels returns the available AI models
func (s *HTTPAIService) GetAvailableModels() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	endpoint := "/ai/models"

	var response struct {
		Models []string `json:"models"`
	}

	err := s.makeRequest(ctx, "GET", endpoint, nil, &response)
	if err != nil {
		// Return default models if request fails
		return []string{"claude", "openai", "perplexity"}
	}

	return response.Models
}

// makeRequest makes an HTTP request to the AI service
func (s *HTTPAIService) makeRequest(ctx context.Context, method, endpoint string, body, result interface{}) error {
	// Build URL
	reqURL, err := url.JoinPath(s.baseURL, endpoint)
	if err != nil {
		return fmt.Errorf("failed to build request URL: %w", err)
	}

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	// Make request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log error but don't return it as the main response is more important
			_ = closeErr // Explicitly acknowledge we're discarding the error
		}
	}()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("AI service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response if result is provided
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// RequestOptions contains options for AI requests
type RequestOptions struct {
	Model     string            `json:"model,omitempty"`
	MaxTokens int               `json:"max_tokens,omitempty"`
	Timeout   time.Duration     `json:"timeout,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// WithOptions adds options to a request
func WithOptions(req interface{}, opts *RequestOptions) interface{} {
	if opts == nil {
		return req
	}

	// Use type assertion to add options to specific request types
	switch r := req.(type) {
	case *ports.PRDGenerationRequest:
		applyModelOption(&r.Model, opts.Model)
		applyMetadataToRequest(&r.Metadata, opts.Metadata)
	case *ports.TRDGenerationRequest:
		applyModelOption(&r.Model, opts.Model)
		applyMetadataToRequest(&r.Metadata, opts.Metadata)
	case *ports.MainTaskGenerationRequest:
		applyModelOption(&r.Model, opts.Model)
		applyMetadataToRequest(&r.Metadata, opts.Metadata)
	case *ports.SubTaskGenerationRequest:
		applyModelOption(&r.Model, opts.Model)
		applyMetadataToRequest(&r.Metadata, opts.Metadata)
	case *ports.ContentAnalysisRequest:
		applyModelOption(&r.Model, opts.Model)
		applyMetadataToContext(&r.Context, opts.Metadata)
	}

	return req
}

// applyModelOption sets the model if provided
func applyModelOption(targetModel *string, optionModel string) {
	if optionModel != "" {
		*targetModel = optionModel
	}
}

// applyMetadataToRequest applies metadata to a request's metadata field
func applyMetadataToRequest(targetMetadata *map[string]interface{}, optionMetadata map[string]string) {
	if optionMetadata == nil {
		return
	}

	if *targetMetadata == nil {
		*targetMetadata = make(map[string]interface{})
	}

	for k, v := range optionMetadata {
		(*targetMetadata)[k] = v
	}
}

// applyMetadataToContext applies metadata to a context field
func applyMetadataToContext(targetContext *map[string]interface{}, optionMetadata map[string]string) {
	if optionMetadata == nil {
		return
	}

	if *targetContext == nil {
		*targetContext = make(map[string]interface{})
	}

	for k, v := range optionMetadata {
		(*targetContext)[k] = v
	}
}

// RetryConfig contains configuration for request retries
type RetryConfig struct {
	MaxRetries int           `json:"max_retries"`
	Backoff    time.Duration `json:"backoff"`
}

// WithRetry wraps a request with retry logic
func (s *HTTPAIService) WithRetry(config *RetryConfig, fn func() error) error {
	if config == nil {
		return fn()
	}

	var lastErr error
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on the last attempt
		if attempt == config.MaxRetries {
			break
		}

		// Exponential backoff
		delay := config.Backoff * time.Duration(1<<attempt)
		time.Sleep(delay)
	}

	return fmt.Errorf("request failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}
