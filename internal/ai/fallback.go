// Package ai provides fallback routing logic for AI model clients.
package ai

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// FallbackStrategy defines the fallback chain order
type FallbackStrategy int

const (
	// StrategyClaudeFirst uses Claude → Perplexity → OpenAI
	StrategyClaudeFirst FallbackStrategy = iota
	// StrategyOpenAIFirst uses OpenAI → Claude → Perplexity
	StrategyOpenAIFirst
	// StrategyFastest uses the historically fastest model first
	StrategyFastest
)

// FallbackRouter manages AI model fallback logic
type FallbackRouter struct {
	clients  map[Model]Client
	strategy FallbackStrategy
	timeout  time.Duration
}

// NewFallbackRouter creates a new fallback router
func NewFallbackRouter(clients map[Model]Client) *FallbackRouter {
	return &FallbackRouter{
		clients:  clients,
		strategy: StrategyClaudeFirst, // Default strategy
		timeout:  30 * time.Second,    // Default timeout per model
	}
}

// SetStrategy updates the fallback strategy
func (fr *FallbackRouter) SetStrategy(strategy FallbackStrategy) {
	fr.strategy = strategy
}

// SetTimeout sets the timeout for individual model requests
func (fr *FallbackRouter) SetTimeout(timeout time.Duration) {
	fr.timeout = timeout
}

// ProcessWithFallback attempts to process request with fallback chain
func (fr *FallbackRouter) ProcessWithFallback(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Get fallback chain based on strategy and requested model
	chain := fr.getFallbackChain(req.Model)
	if len(chain) == 0 {
		return nil, fmt.Errorf("no available models for fallback")
	}

	var lastErr error
	fallbackUsed := false

	// Try each model in the fallback chain
	for i, model := range chain {
		client, exists := fr.clients[model]
		if !exists {
			lastErr = fmt.Errorf("client for model %s not available", model)
			continue
		}

		// Set fallback flag if we're not using the primary model
		if i > 0 {
			fallbackUsed = true
		}

		// Create context with timeout for this attempt
		modelCtx, cancel := context.WithTimeout(ctx, fr.timeout)
		
		// Create request copy with current model
		modelReq := *req
		modelReq.Model = model

		// Attempt to process with current model
		response, err := client.ProcessRequest(modelCtx, &modelReq)
		cancel()

		if err != nil {
			lastErr = err
			// Log the failure and try next model
			continue
		}

		// Success - mark if fallback was used
		if response != nil {
			response.FallbackUsed = fallbackUsed
		}

		return response, nil
	}

	// All models failed
	if lastErr == nil {
		lastErr = errors.New("all AI models failed without specific error")
	}

	return nil, fmt.Errorf("all fallback attempts failed, last error: %w", lastErr)
}

// getFallbackChain returns the model chain based on strategy
func (fr *FallbackRouter) getFallbackChain(preferredModel Model) []Model {
	var chain []Model

	switch fr.strategy {
	case StrategyClaudeFirst:
		chain = fr.getClaudeFirstChain(preferredModel)
	case StrategyOpenAIFirst:
		chain = fr.getOpenAIFirstChain(preferredModel)
	case StrategyFastest:
		chain = fr.getFastestFirstChain(preferredModel)
	default:
		chain = fr.getClaudeFirstChain(preferredModel)
	}

	// Filter out unavailable models
	return fr.filterAvailableModels(chain)
}

// getClaudeFirstChain returns Claude → Perplexity → OpenAI chain
func (fr *FallbackRouter) getClaudeFirstChain(preferredModel Model) []Model {
	switch preferredModel {
	case ModelClaude:
		return []Model{ModelClaude, ModelPerplexity, ModelOpenAI}
	case ModelPerplexity:
		return []Model{ModelPerplexity, ModelClaude, ModelOpenAI}
	case ModelOpenAI:
		return []Model{ModelOpenAI, ModelClaude, ModelPerplexity}
	default:
		return []Model{ModelClaude, ModelPerplexity, ModelOpenAI}
	}
}

// getOpenAIFirstChain returns OpenAI → Claude → Perplexity chain
func (fr *FallbackRouter) getOpenAIFirstChain(preferredModel Model) []Model {
	switch preferredModel {
	case ModelOpenAI:
		return []Model{ModelOpenAI, ModelClaude, ModelPerplexity}
	case ModelClaude:
		return []Model{ModelClaude, ModelOpenAI, ModelPerplexity}
	case ModelPerplexity:
		return []Model{ModelPerplexity, ModelOpenAI, ModelClaude}
	default:
		return []Model{ModelOpenAI, ModelClaude, ModelPerplexity}
	}
}

// getFastestFirstChain returns chain ordered by historical performance
func (fr *FallbackRouter) getFastestFirstChain(preferredModel Model) []Model {
	// TODO: Implement based on historical metrics
	// For now, use Claude first as it's typically fastest for most tasks
	return fr.getClaudeFirstChain(preferredModel)
}

// filterAvailableModels removes unavailable models from chain
func (fr *FallbackRouter) filterAvailableModels(chain []Model) []Model {
	var available []Model
	for _, model := range chain {
		if _, exists := fr.clients[model]; exists {
			available = append(available, model)
		}
	}
	return available
}

// GetAvailableModels returns all available models
func (fr *FallbackRouter) GetAvailableModels() []Model {
	models := make([]Model, 0, len(fr.clients))
	for model := range fr.clients {
		models = append(models, model)
	}
	return models
}

// IsModelAvailable checks if a specific model is available
func (fr *FallbackRouter) IsModelAvailable(model Model) bool {
	_, exists := fr.clients[model]
	return exists
}

// HealthCheck verifies all clients in fallback chain
func (fr *FallbackRouter) HealthCheck(ctx context.Context) map[Model]error {
	results := make(map[Model]error)
	for model, client := range fr.clients {
		results[model] = client.IsHealthy(ctx)
	}
	return results
}