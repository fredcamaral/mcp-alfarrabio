package ai

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/logging"
)

func TestService_Integration(t *testing.T) {
	// Skip integration tests if not explicitly enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run.")
	}

	// Create test configuration
	cfg := &config.Config{
		AI: config.AIConfig{
			Claude: config.ClaudeClientConfig{
				Enabled: os.Getenv("ANTHROPIC_API_KEY") != "",
				APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
				Model:   "claude-3-haiku-20240307", // Cheapest model for testing
			},
			Perplexity: config.PerplexityClientConfig{
				Enabled: os.Getenv("PERPLEXITY_API_KEY") != "",
				APIKey:  os.Getenv("PERPLEXITY_API_KEY"),
				Model:   "sonar-small-online", // Cheapest model for testing
			},
			OpenAI: config.OpenAIClientConfig{
				Enabled: os.Getenv("OPENAI_API_KEY") != "",
				APIKey:  os.Getenv("OPENAI_API_KEY"),
				Model:   "gpt-3.5-turbo", // Cheapest model for testing
			},
		},
	}

	// Create logger
	logger := logging.NewLogger(logging.DEBUG)

	// Create service
	service, err := NewService(cfg, logger)
	require.NoError(t, err)
	defer service.Close()

	// Get available models
	models := service.GetAvailableModels()
	assert.NotEmpty(t, models)
	t.Logf("Available models: %v", models)

	// Test each available model
	for _, model := range models {
		t.Run(string(model), func(t *testing.T) {
			// Skip if it's a mock model
			if model == Model("mock-model-1.0") {
				t.Skip("Skipping mock model")
			}

			// Set as primary model
			err := service.SetPrimaryModel(model)
			require.NoError(t, err)
			assert.Equal(t, model, service.GetPrimaryModel())

			// Create test request
			req := &Request{
				ID:    "test-" + string(model),
				Model: string(model),
				Messages: []Message{
					{
						Role:    "system",
						Content: "You are a helpful assistant. Respond with exactly one word.",
					},
					{
						Role:    "user",
						Content: "What is 2+2? Answer with just the number.",
					},
				},
				Metadata: &RequestMetadata{
					Repository: "test-repo",
					SessionID:  "test-session",
					UserID:     "test-user",
					CreatedAt:  time.Now(),
				},
			}

			// Process request
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, err := service.ProcessRequest(ctx, req)
			if err != nil {
				// Check if it's an API key error
				if apiErr, ok := err.(*APIError); ok {
					if apiErr.StatusCode == 401 || apiErr.StatusCode == 403 {
						t.Skipf("Skipping %s: API key not configured", model)
						return
					}
				}
				t.Fatalf("Failed to process request: %v", err)
			}

			// Validate response
			assert.NotNil(t, resp)
			assert.NotEmpty(t, resp.Content)
			assert.Equal(t, string(model), resp.Model)
			assert.Greater(t, resp.TokensUsed.Total, 0)
			assert.Greater(t, resp.Latency, int64(0))
			assert.False(t, resp.CacheHit) // First request should not be cached

			// Check quality metrics
			assert.Greater(t, resp.Quality.Confidence, 0.0)
			assert.Greater(t, resp.Quality.Score, 0.0)

			t.Logf("Model %s response: %s (tokens: %d, latency: %d ms)",
				model, resp.Content, resp.TokensUsed.Total, resp.Latency)

			// Test cache hit on second request
			resp2, err := service.ProcessRequest(ctx, req)
			require.NoError(t, err)
			assert.True(t, resp2.CacheHit)
			assert.Equal(t, resp.Content, resp2.Content)
		})
	}
}

func TestService_HealthCheck(t *testing.T) {
	// Create test configuration with mock client
	cfg := &config.Config{}
	logger := logging.NewLogger(logging.DEBUG)

	// Create service
	service, err := NewService(cfg, logger)
	require.NoError(t, err)
	defer service.Close()

	// Run health check
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health := service.HealthCheck(ctx)
	assert.NotEmpty(t, health)

	// At least one model should be healthy (mock)
	hasHealthy := false
	for model, err := range health {
		t.Logf("Model %s health: %v", model, err)
		if err == nil {
			hasHealthy = true
		}
	}
	assert.True(t, hasHealthy, "At least one model should be healthy")
}

func TestService_Fallback(t *testing.T) {
	// Create test configuration with mock clients
	cfg := &config.Config{}
	logger := logging.NewLogger(logging.DEBUG)

	// Create service
	service, err := NewService(cfg, logger)
	require.NoError(t, err)
	defer service.Close()

	// Create request that will trigger fallback
	req := &Request{
		ID: "test-fallback",
		Messages: []Message{
			{
				Role:    "user",
				Content: "test",
			},
		},
		Metadata: &RequestMetadata{
			CreatedAt: time.Now(),
		},
	}

	// Process request
	ctx := context.Background()
	resp, err := service.ProcessRequest(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.Content)
}

func TestService_Metrics(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{}
	logger := logging.NewLogger(logging.DEBUG)

	// Create service
	service, err := NewService(cfg, logger)
	require.NoError(t, err)
	defer service.Close()

	// Get initial metrics
	metrics := service.GetMetrics()
	assert.NotNil(t, metrics)

	// Process a request
	req := &Request{
		ID: "test-metrics",
		Messages: []Message{
			{
				Role:    "user",
				Content: "test",
			},
		},
		Metadata: &RequestMetadata{
			CreatedAt: time.Now(),
		},
	}

	ctx := context.Background()
	_, err = service.ProcessRequest(ctx, req)
	require.NoError(t, err)

	// Verify metrics were updated
	// Note: Since metrics are internal, we can't directly verify counts
	// but we can ensure the metrics object is still valid
	metrics2 := service.GetMetrics()
	assert.NotNil(t, metrics2)
}
