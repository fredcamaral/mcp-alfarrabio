package ai

import (
	"context"
	"testing"
	"time"

	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAPIKey = "test-key"

func TestNewService(t *testing.T) {
	cfg := config.DefaultConfig()

	// Enable at least one client for testing
	cfg.AI.Claude.Enabled = true
	cfg.AI.Claude.APIKey = testAPIKey

	logger := logging.NewLogger(logging.DEBUG)
	service, err := NewService(cfg, logger)

	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.NotNil(t, service.clients)
	assert.NotNil(t, service.fallback)
	assert.NotNil(t, service.cache)
	assert.NotNil(t, service.metrics)
	assert.Equal(t, ModelClaude, service.primaryModel)
}

func TestNewServiceWithNilConfig(t *testing.T) {
	service, err := NewService(nil, nil)

	assert.Error(t, err)
	assert.Nil(t, service)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestNewServiceWithNoEnabledClients(t *testing.T) {
	cfg := config.DefaultConfig()
	// All clients disabled by default and no API keys

	logger := logging.NewLogger(logging.DEBUG)
	service, err := NewService(cfg, logger)

	// Should succeed with mock client when no API keys are provided
	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Contains(t, string(service.primaryModel), "mock-model")
}

func TestGetAvailableModels(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AI.Claude.Enabled = true
	cfg.AI.Claude.APIKey = testAPIKey
	cfg.AI.OpenAI.Enabled = true
	cfg.AI.OpenAI.APIKey = testAPIKey

	logger := logging.NewLogger(logging.DEBUG)
	service, err := NewService(cfg, logger)
	require.NoError(t, err)

	models := service.GetAvailableModels()

	assert.Len(t, models, 2)
	assert.Contains(t, models, ModelClaude)
	assert.Contains(t, models, ModelOpenAI)
}

func TestSetPrimaryModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AI.Claude.Enabled = true
	cfg.AI.Claude.APIKey = testAPIKey
	cfg.AI.OpenAI.Enabled = true
	cfg.AI.OpenAI.APIKey = testAPIKey

	logger := logging.NewLogger(logging.DEBUG)
	service, err := NewService(cfg, logger)
	require.NoError(t, err)

	// Test setting to available model
	err = service.SetPrimaryModel(ModelOpenAI)
	assert.NoError(t, err)
	assert.Equal(t, ModelOpenAI, service.GetPrimaryModel())

	// Test setting to unavailable model
	err = service.SetPrimaryModel(ModelPerplexity)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not available")
}

func TestProcessRequestWithNilRequest(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AI.Claude.Enabled = true
	cfg.AI.Claude.APIKey = testAPIKey

	logger := logging.NewLogger(logging.DEBUG)
	service, err := NewService(cfg, logger)
	require.NoError(t, err)

	ctx := context.Background()
	response, err := service.ProcessRequest(ctx, nil)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "request cannot be nil")
}

func TestProcessRequestWithEmptyModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AI.Claude.Enabled = true
	cfg.AI.Claude.APIKey = testAPIKey

	logger := logging.NewLogger(logging.DEBUG)
	service, err := NewService(cfg, logger)
	require.NoError(t, err)

	req := &Request{
		ID: "test-request",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		Metadata: &RequestMetadata{
			CreatedAt: time.Now(),
		},
	}

	ctx := context.Background()

	// This should fail because we don't have real API keys
	// but it tests that the primary model is set correctly
	_, err = service.ProcessRequest(ctx, req)

	// Should fail at the API call level, not before
	assert.Error(t, err)
	// The request should have the primary model set
	assert.Equal(t, string(ModelClaude), req.Model)
}

func TestServiceClose(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.AI.Claude.Enabled = true
	cfg.AI.Claude.APIKey = testAPIKey

	logger := logging.NewLogger(logging.DEBUG)
	service, err := NewService(cfg, logger)
	require.NoError(t, err)

	err = service.Close()
	assert.NoError(t, err)
}
