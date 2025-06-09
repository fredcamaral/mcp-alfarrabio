// Package ai provides multi-model AI service integration with fallback routing and caching.
package ai

import (
	"context"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/config"
)

// Model represents an AI model identifier
type Model string

const (
	ModelClaude     Model = "claude-sonnet-4"
	ModelPerplexity Model = "perplexity-sonar-pro"
	ModelOpenAI     Model = "openai-gpt-4o"
)

// Request represents an AI service request
type Request struct {
	ID       string            `json:"id"`
	Model    Model             `json:"model,omitempty"`
	Messages []Message         `json:"messages"`
	Context  map[string]string `json:"context,omitempty"`
	Metadata RequestMetadata   `json:"metadata"`
}

// Message represents a single message in an AI conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RequestMetadata contains request tracking information
type RequestMetadata struct {
	Repository  string            `json:"repository,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	UserID      string            `json:"user_id,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Preferences map[string]string `json:"preferences,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
}

// Response represents an AI service response
type Response struct {
	ID           string           `json:"id"`
	Model        Model            `json:"model"`
	Content      string           `json:"content"`
	TokensUsed   TokenUsage       `json:"tokens_used"`
	Latency      time.Duration    `json:"latency"`
	CacheHit     bool             `json:"cache_hit"`
	FallbackUsed bool             `json:"fallback_used"`
	Quality      QualityMetrics   `json:"quality"`
	Metadata     ResponseMetadata `json:"metadata"`
	Error        string           `json:"error,omitempty"`
}

// TokenUsage represents token consumption metrics
type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

// QualityMetrics represents response quality assessment
type QualityMetrics struct {
	Confidence float64 `json:"confidence"`
	Relevance  float64 `json:"relevance"`
	Clarity    float64 `json:"clarity"`
	Score      float64 `json:"score"`
}

// ResponseMetadata contains response tracking information
type ResponseMetadata struct {
	ProcessedAt time.Time `json:"processed_at"`
	ServerID    string    `json:"server_id"`
	Version     string    `json:"version"`
}

// Client defines the interface for AI model clients
type Client interface {
	// ProcessRequest sends a request to the AI model
	ProcessRequest(ctx context.Context, req *Request) (*Response, error)
	
	// GetModel returns the model identifier
	GetModel() Model
	
	// IsHealthy checks if the client is operational
	IsHealthy(ctx context.Context) error
	
	// GetLimits returns rate limiting information
	GetLimits() RateLimits
}

// RateLimits represents rate limiting configuration
type RateLimits struct {
	RequestsPerMinute int           `json:"requests_per_minute"`
	TokensPerMinute   int           `json:"tokens_per_minute"`
	ResetTime         time.Duration `json:"reset_time"`
}

// Service provides AI functionality with multi-model support
type Service struct {
	clients    map[Model]Client
	fallback   *FallbackRouter
	cache      *Cache
	metrics    *Metrics
	config     *config.Config
	primaryModel Model
}

// NewService creates a new AI service with configured clients
func NewService(cfg *config.Config) (*Service, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	service := &Service{
		clients:      make(map[Model]Client),
		config:       cfg,
		primaryModel: ModelClaude, // Default primary model
	}

	// Initialize clients
	if err := service.initializeClients(); err != nil {
		return nil, fmt.Errorf("failed to initialize AI clients: %w", err)
	}

	// Initialize fallback router
	service.fallback = NewFallbackRouter(service.clients)

	// Initialize cache
	cache, err := NewCache(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}
	service.cache = cache

	// Initialize metrics
	service.metrics = NewMetrics()

	return service, nil
}

// initializeClients sets up all configured AI model clients
func (s *Service) initializeClients() error {
	// Initialize Claude client
	if s.config.AI.Claude.Enabled {
		claudeClient, err := NewClaudeClient(s.config.AI.Claude)
		if err != nil {
			return fmt.Errorf("failed to create Claude client: %w", err)
		}
		s.clients[ModelClaude] = claudeClient
	}

	// Initialize Perplexity client
	if s.config.AI.Perplexity.Enabled {
		perplexityClient, err := NewPerplexityClient(s.config.AI.Perplexity)
		if err != nil {
			return fmt.Errorf("failed to create Perplexity client: %w", err)
		}
		s.clients[ModelPerplexity] = perplexityClient
	}

	// Initialize OpenAI client
	if s.config.AI.OpenAI.Enabled {
		openaiClient, err := NewOpenAIClient(s.config.AI.OpenAI)
		if err != nil {
			return fmt.Errorf("failed to create OpenAI client: %w", err)
		}
		s.clients[ModelOpenAI] = openaiClient
	}

	if len(s.clients) == 0 {
		return fmt.Errorf("no AI clients configured")
	}

	return nil
}

// ProcessRequest processes an AI request with fallback and caching
func (s *Service) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Set default model if not specified
	if req.Model == "" {
		req.Model = s.primaryModel
	}

	// Set request timestamp
	req.Metadata.CreatedAt = time.Now()

	// Check cache first
	if cachedResponse, found := s.cache.Get(req); found {
		s.metrics.RecordCacheHit(req.Model)
		cachedResponse.CacheHit = true
		return cachedResponse, nil
	}

	// Record cache miss
	s.metrics.RecordCacheMiss(req.Model)

	// Process request with fallback
	response, err := s.fallback.ProcessWithFallback(ctx, req)
	if err != nil {
		s.metrics.RecordError(req.Model, err)
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	// Record successful request
	s.metrics.RecordRequest(response.Model, response.Latency, response.TokensUsed)

	// Cache the response
	s.cache.Set(req, response)

	return response, nil
}

// GetAvailableModels returns list of available AI models
func (s *Service) GetAvailableModels() []Model {
	models := make([]Model, 0, len(s.clients))
	for model := range s.clients {
		models = append(models, model)
	}
	return models
}

// SetPrimaryModel sets the preferred primary model
func (s *Service) SetPrimaryModel(model Model) error {
	if _, exists := s.clients[model]; !exists {
		return fmt.Errorf("model %s is not available", model)
	}
	s.primaryModel = model
	return nil
}

// GetPrimaryModel returns the current primary model
func (s *Service) GetPrimaryModel() Model {
	return s.primaryModel
}

// HealthCheck verifies all clients are operational
func (s *Service) HealthCheck(ctx context.Context) map[Model]error {
	results := make(map[Model]error)
	for model, client := range s.clients {
		results[model] = client.IsHealthy(ctx)
	}
	return results
}

// GetMetrics returns current service metrics
func (s *Service) GetMetrics() *Metrics {
	return s.metrics
}

// Close cleans up resources
func (s *Service) Close() error {
	if s.cache != nil {
		if err := s.cache.Close(); err != nil {
			return fmt.Errorf("failed to close cache: %w", err)
		}
	}
	return nil
}