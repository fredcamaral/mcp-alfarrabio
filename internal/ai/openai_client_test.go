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

func TestOpenAIClient_NewOpenAIClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		baseURL string
		wantErr bool
	}{
		{
			name:    "valid configuration",
			apiKey:  "test-api-key",
			baseURL: "https://api.openai.com/v1",
			wantErr: false,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			baseURL: "https://api.openai.com/v1",
			wantErr: true,
		},
		{
			name:    "custom base URL",
			apiKey:  "test-api-key",
			baseURL: "https://openrouter.ai/api/v1",
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
			client, err := NewOpenAIClient(tt.apiKey, tt.baseURL)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.apiKey, client.apiKey)
				if tt.baseURL == "" {
					assert.Equal(t, "https://api.openai.com/v1", client.baseURL)
				} else {
					assert.Equal(t, tt.baseURL, client.baseURL)
				}
			}
		})
	}
}

func TestOpenAIClient_Complete(t *testing.T) {
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
				Model:       "gpt-4",
				MaxTokens:   100,
				Temperature: 0.7,
			},
			mockResponse: map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"content": "Here's a test function:\n\nfunc TestExample(t *testing.T) {\n    assert.True(t, true)\n}",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 20,
					"total_tokens":      30,
				},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
			validateResp: func(t *testing.T, resp *CompletionResponse) {
				assert.Contains(t, resp.Content, "TestExample")
				assert.Equal(t, "gpt-4", resp.Model)
				assert.Equal(t, 30, resp.Usage.Total)
			},
		},
		{
			name: "API error response",
			request: CompletionRequest{
				Prompt: "Test prompt",
				Model:  "gpt-4",
			},
			mockResponse: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Invalid API key",
					"type":    "invalid_request_error",
					"code":    "invalid_api_key",
				},
			},
			mockStatusCode: http.StatusUnauthorized,
			wantErr:        true,
		},
		{
			name: "rate limit exceeded",
			request: CompletionRequest{
				Prompt: "Test prompt",
				Model:  "gpt-4",
			},
			mockResponse: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Rate limit exceeded",
					"type":    "rate_limit_error",
				},
			},
			mockStatusCode: http.StatusTooManyRequests,
			wantErr:        true,
		},
		{
			name: "empty response",
			request: CompletionRequest{
				Prompt: "Test prompt",
				Model:  "gpt-4",
			},
			mockResponse: map[string]interface{}{
				"choices": []map[string]interface{}{},
			},
			mockStatusCode: http.StatusOK,
			wantErr:        true,
		},
		{
			name: "with system message",
			request: CompletionRequest{
				Prompt:        "Write a function",
				SystemMessage: "You are a helpful coding assistant",
				Model:         "gpt-4",
			},
			mockResponse: map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"content": "func example() {}",
						},
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
			name: "with metadata",
			request: CompletionRequest{
				Prompt: "Test",
				Model:  "gpt-4",
				Metadata: map[string]interface{}{
					"task_type": "code_generation",
					"language":  "go",
				},
			},
			mockResponse: map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"content": "response",
						},
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
		{
			name: "timeout scenario",
			request: CompletionRequest{
				Prompt:  "Test timeout",
				Model:   "gpt-4",
				Timeout: 1 * time.Millisecond, // Very short timeout
			},
			mockResponse:   map[string]interface{}{}, // Response doesn't matter
			mockStatusCode: http.StatusOK,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Validate request
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/chat/completions", r.URL.Path)
				assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

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
			client, err := NewOpenAIClient("test-key", server.URL)
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

func TestOpenAIClient_ValidateRequest(t *testing.T) {
	client, err := NewOpenAIClient("test-key", "")
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
				Model:  "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "empty prompt",
			request: CompletionRequest{
				Prompt: "",
				Model:  "gpt-4",
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
				Model:     "gpt-4",
				MaxTokens: -1,
			},
			wantErr: true,
			errMsg:  "max tokens must be positive",
		},
		{
			name: "invalid temperature",
			request: CompletionRequest{
				Prompt:      "Test prompt",
				Model:       "gpt-4",
				Temperature: 2.5,
			},
			wantErr: true,
			errMsg:  "temperature must be between 0 and 2",
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

func TestOpenAIClient_GetCapabilities(t *testing.T) {
	client, err := NewOpenAIClient("test-key", "")
	require.NoError(t, err)

	caps := client.GetCapabilities()

	assert.True(t, caps.SupportsStreaming)
	assert.True(t, caps.SupportsSystemMessage)
	assert.Contains(t, caps.SupportedModels, "gpt-4")
	assert.Contains(t, caps.SupportedModels, "gpt-4-turbo")
	assert.Contains(t, caps.SupportedModels, "gpt-3.5-turbo")
	assert.Equal(t, 128000, caps.MaxTokens)
	assert.Equal(t, "openai", caps.Provider)
}

func TestOpenAIClient_RetryLogic(t *testing.T) {
	retryCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		retryCount++
		if retryCount < 3 {
			// Return rate limit error for first 2 attempts
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Rate limit exceeded",
					"type":    "rate_limit_error",
				},
			})
		} else {
			// Success on third attempt
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"content": "Success after retries",
						},
					},
				},
			})
		}
	}))
	defer server.Close()

	client, err := NewOpenAIClient("test-key", server.URL)
	require.NoError(t, err)

	// Enable retry for rate limits
	client.maxRetries = 3
	client.retryDelay = 10 * time.Millisecond

	ctx := context.Background()
	resp, err := client.Complete(ctx, CompletionRequest{
		Prompt: "Test retry",
		Model:  "gpt-4",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Success after retries", resp.Content)
	assert.Equal(t, 3, retryCount)
}

func TestOpenAIClient_ConcurrentRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "Concurrent response",
					},
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewOpenAIClient("test-key", server.URL)
	require.NoError(t, err)

	// Run multiple concurrent requests
	concurrency := 10
	errChan := make(chan error, concurrency)
	respChan := make(chan *CompletionResponse, concurrency)

	ctx := context.Background()
	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			resp, err := client.Complete(ctx, CompletionRequest{
				Prompt: "Concurrent test",
				Model:  "gpt-4",
			})
			if err != nil {
				errChan <- err
			} else {
				respChan <- resp
			}
		}(i)
	}

	// Collect results
	responses := 0
	errors := 0
	for i := 0; i < concurrency; i++ {
		select {
		case err := <-errChan:
			errors++
			t.Errorf("Concurrent request failed: %v", err)
		case resp := <-respChan:
			responses++
			assert.Equal(t, "Concurrent response", resp.Content)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent responses")
		}
	}

	assert.Equal(t, concurrency, responses)
	assert.Equal(t, 0, errors)
}

func TestOpenAIClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer server.Close()

	client, err := NewOpenAIClient("test-key", server.URL)
	require.NoError(t, err)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	resp, err := client.Complete(ctx, CompletionRequest{
		Prompt: "Test cancellation",
		Model:  "gpt-4",
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "context canceled")
}
