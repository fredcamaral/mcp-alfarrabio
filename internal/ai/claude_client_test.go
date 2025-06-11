package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeClient_NewClaudeClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		baseURL string
		wantErr bool
	}{
		{
			name:    "valid configuration",
			apiKey:  "test-api-key",
			baseURL: "https://api.anthropic.com/v1",
			wantErr: false,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			baseURL: "https://api.anthropic.com/v1",
			wantErr: true,
		},
		{
			name:    "custom base URL",
			apiKey:  "test-api-key",
			baseURL: "https://custom.anthropic.com/v1",
			wantErr: false,
		},
		{
			name:    "empty base URL uses default",
			apiKey:  "test-api-key",
			baseURL: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClaudeSimpleClient(tt.apiKey, tt.baseURL)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.apiKey, client.apiKey)
				if tt.baseURL == "" {
					assert.Equal(t, "https://api.anthropic.com/v1", client.baseURL)
				} else {
					assert.Equal(t, tt.baseURL, client.baseURL)
				}
			}
		})
	}
}

func TestClaudeClient_Complete(t *testing.T) {
	tests := []struct {
		name           string
		request        CompletionRequest
		mockResponse   interface{}
		mockStatusCode int
		wantErr        bool
		validateResp   func(*testing.T, *CompletionResponse)
	}{
		{
			name: "successful completion",
			request: CompletionRequest{
				Prompt:      "Write a test function",
				Model:       "claude-3-sonnet-20240229",
				MaxTokens:   100,
				Temperature: 0.7,
			},
			mockResponse: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "Here's a test function:\n\nfunc TestExample(t *testing.T) {\n    assert.True(t, true)\n}",
					},
				},
				"usage": map[string]interface{}{
					"input_tokens":  10,
					"output_tokens": 20,
				},
				"stop_reason": "end_turn",
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			validateResp: func(t *testing.T, resp *CompletionResponse) {
				assert.Contains(t, resp.Content, "TestExample")
				assert.Equal(t, "claude-3-sonnet-20240229", resp.Model)
				assert.Equal(t, 30, resp.Usage.Total)
			},
		},
		{
			name: "API error response",
			request: CompletionRequest{
				Prompt: "Test prompt",
				Model:  "claude-3-sonnet-20240229",
			},
			mockResponse: map[string]interface{}{
				"error": map[string]interface{}{
					"type":    "invalid_request_error",
					"message": "Invalid API key",
				},
			},
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name: "rate limit exceeded",
			request: CompletionRequest{
				Prompt: "Test prompt",
				Model:  "claude-3-sonnet-20240229",
			},
			mockResponse: map[string]interface{}{
				"error": map[string]interface{}{
					"type":    "rate_limit_error",
					"message": "Rate limit exceeded",
				},
			},
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
		{
			name: "with system message",
			request: CompletionRequest{
				Prompt:        "Write a function",
				SystemMessage: "You are a helpful coding assistant",
				Model:         "claude-3-sonnet-20240229",
			},
			mockResponse: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "func example() {}",
					},
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			validateResp: func(t *testing.T, resp *CompletionResponse) {
				assert.Equal(t, "func example() {}", resp.Content)
			},
		},
		{
			name: "timeout scenario",
			request: CompletionRequest{
				Prompt:  "Test timeout",
				Model:   "claude-3-sonnet-20240229",
				Timeout: 1 * time.Millisecond, // Very short timeout
			},
			mockResponse:   map[string]interface{}{}, // Response doesn't matter
			mockStatusCode: http.StatusOK,
			wantErr:        true,
		},
		{
			name: "with metadata",
			request: CompletionRequest{
				Prompt: "Test",
				Model:  "claude-3-sonnet-20240229",
				Metadata: map[string]interface{}{
					"task_type": "code_generation",
					"language":  "go",
				},
			},
			mockResponse: map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": "response",
					},
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			validateResp: func(t *testing.T, resp *CompletionResponse) {
				assert.Equal(t, "code_generation", resp.Metadata["task_type"])
				assert.Equal(t, "go", resp.Metadata["language"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Validate request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/messages", r.URL.Path)
				assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))

				// Parse request body
				var requestBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				require.NoError(t, err)

				// Validate model
				assert.Equal(t, tt.request.Model, requestBody["model"])

				// For timeout test, delay response
				if tt.name == "timeout scenario" {
					time.Sleep(10 * time.Millisecond)
				}

				// Send response
				w.WriteHeader(tt.mockStatusCode)
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			// Create client with test server URL
			client, err := NewClaudeSimpleClient("test-key", server.URL)
			require.NoError(t, err)

			// Execute request
			ctx := context.Background()
			resp, err := client.Complete(ctx, tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.validateResp != nil {
					tt.validateResp(t, resp)
				}
			}
		})
	}
}

func TestClaudeClient_ValidateRequest(t *testing.T) {
	client, err := NewClaudeSimpleClient("test-key", "")
	require.NoError(t, err)

	tests := []struct {
		name    string
		request CompletionRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: CompletionRequest{
				Prompt: "Test prompt",
				Model:  "claude-3-sonnet-20240229",
			},
			wantErr: false,
		},
		{
			name: "empty prompt",
			request: CompletionRequest{
				Prompt: "",
				Model:  "claude-3-sonnet-20240229",
			},
			wantErr: true,
			errMsg:  "prompt cannot be empty",
		},
		{
			name: "empty model",
			request: CompletionRequest{
				Prompt: "Test prompt",
				Model:  "",
			},
			wantErr: true,
			errMsg:  "model cannot be empty",
		},
		{
			name: "negative max tokens",
			request: CompletionRequest{
				Prompt:    "Test prompt",
				Model:     "claude-3-sonnet-20240229",
				MaxTokens: -1,
			},
			wantErr: true,
			errMsg:  "max tokens must be positive",
		},
		{
			name: "invalid temperature",
			request: CompletionRequest{
				Prompt:      "Test prompt",
				Model:       "claude-3-sonnet-20240229",
				Temperature: 2.5,
			},
			wantErr: true,
			errMsg:  "temperature must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.ValidateRequest(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestClaudeClient_GetCapabilities(t *testing.T) {
	client, err := NewClaudeSimpleClient("test-key", "")
	require.NoError(t, err)

	caps := client.GetCapabilities()

	assert.True(t, caps.SupportsStreaming)
	assert.True(t, caps.SupportsSystemMessage)
	assert.Contains(t, caps.SupportedModels, "claude-3-opus-20240229")
	assert.Contains(t, caps.SupportedModels, "claude-3-sonnet-20240229")
	assert.Contains(t, caps.SupportedModels, "claude-3-haiku-20240307")
	assert.Equal(t, 200000, caps.MaxTokens) // Claude's 200k context window
	assert.Equal(t, "anthropic", caps.Provider)
}
