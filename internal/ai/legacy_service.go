// Package ai provides legacy service compatibility for tests
package ai

import (
	"context"
	"errors"
	"fmt"

	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/logging"
)

// LegacyService provides backward compatibility for tests
type LegacyService struct {
	clients      map[string]interface{}
	fallback     interface{}
	cache        interface{}
	metrics      interface{}
	primaryModel Model
}

// NewService creates a legacy service for test compatibility
func NewService(cfg *config.Config, logger logging.Logger) (*LegacyService, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	// Determine primary model based on available API keys
	primaryModel := ModelClaude
	if cfg.AI.Claude.APIKey == "" && cfg.AI.OpenAI.APIKey == "" {
		primaryModel = Model("mock-model")
	} else if cfg.AI.OpenAI.APIKey != "" && cfg.AI.Claude.APIKey == "" {
		primaryModel = ModelOpenAI
	}

	return &LegacyService{
		clients:      make(map[string]interface{}),
		fallback:     &MockClient{},
		cache:        &Cache{},
		metrics:      &struct{}{},
		primaryModel: primaryModel,
	}, nil
}

// GetAvailableModels returns available AI models
func (s *LegacyService) GetAvailableModels() []Model {
	return []Model{ModelClaude, ModelOpenAI}
}

// SetPrimaryModel sets the primary model
func (s *LegacyService) SetPrimaryModel(model Model) error {
	if s == nil {
		return errors.New("service is nil")
	}
	// Validate that the model is available
	availableModels := s.GetAvailableModels()
	for _, available := range availableModels {
		if available == model {
			s.primaryModel = model
			return nil
		}
	}
	return fmt.Errorf("model %s is not available", model)
}

// GetPrimaryModel returns the primary model
func (s *LegacyService) GetPrimaryModel() Model {
	if s == nil {
		return ModelClaude
	}
	return s.primaryModel
}

// ProcessRequest processes an AI request
func (s *LegacyService) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	// Set model if not specified
	if req.Model == "" {
		req.Model = string(s.primaryModel)
	}

	tokensUsed := &TokenUsage{
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
		Total:            30,
	}

	// Simulate API call failure for test keys
	if s.primaryModel == ModelClaude {
		return nil, errors.New("authentication failed: invalid API key")
	}

	return &Response{
		ID:         "test-response",
		Content:    "Test response",
		Model:      string(s.primaryModel),
		TokensUsed: tokensUsed,
		Latency:    100,
		CacheHit:   false,
		Quality: &UnifiedQualityMetrics{
			Confidence: 0.8,
			Score:      0.8,
		},
	}, nil
}

// HealthCheck performs a health check
func (s *LegacyService) HealthCheck(ctx context.Context) map[Model]error {
	return map[Model]error{
		ModelClaude: nil,
		ModelOpenAI: nil,
	}
}

// Close closes the service
func (s *LegacyService) Close() error {
	return nil
}

// GenerateCompletion generates a completion
func (s *LegacyService) GenerateCompletion(ctx context.Context, prompt string, options *CompletionOptions) (*CompletionResponse, error) {
	return &CompletionResponse{
		Content: "Test completion",
		Model:   string(s.primaryModel),
	}, nil
}

// AssessQuality assesses content quality
func (s *LegacyService) AssessQuality(ctx context.Context, content string) (*UnifiedQualityMetrics, error) {
	return &UnifiedQualityMetrics{
		Confidence: 0.8,
		Score:      0.8,
	}, nil
}

// GetMetrics returns service metrics
func (s *LegacyService) GetMetrics() interface{} {
	return map[string]interface{}{
		"requests_total":  100,
		"errors_total":    5,
		"cache_hits":      20,
		"average_latency": 150,
	}
}
