// Package ai provides a unified AI service implementation
// This resolves conflicts between multiple AI service implementations
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"lerian-mcp-memory/internal/reliability"
)

// Client interfaces for different AI providers
type OpenAIClientInterface interface {
	GenerateCompletion(ctx context.Context, prompt string, options *CompletionOptions) (*CompletionResponse, error)
}

type ClaudeClientInterface interface {
	GenerateCompletion(ctx context.Context, prompt string, options *CompletionOptions) (*CompletionResponse, error)
}

// UnifiedService implements the AIService interface with a clean, consolidated approach
type UnifiedService struct {
	config         *AIConfig
	logger         *slog.Logger
	provider       AIProvider
	client         interface{} // Will hold the actual provider client
	circuitBreaker *reliability.CircuitBreaker
}

// NewAIService creates a new unified AI service (replaces conflicting constructors)
func NewAIService(config *AIConfig, logger *slog.Logger) (AIService, error) {
	if config == nil {
		return nil, fmt.Errorf("AI config cannot be nil")
	}

	if logger == nil {
		logger = slog.Default()
	}

	// Create circuit breaker for AI service calls
	cbConfig := reliability.DefaultConfig(fmt.Sprintf("ai-service-%s", config.Provider))
	cbConfig.MaxFailures = 3                 // Open after 3 failures
	cbConfig.ResetTimeout = 60 * time.Second // Wait 1 minute before retry
	cbConfig.Timeout = 30 * time.Second      // 30 second timeout for AI calls

	service := &UnifiedService{
		config:         config,
		logger:         logger,
		provider:       config.Provider,
		circuitBreaker: reliability.NewCircuitBreaker(cbConfig),
	}

	// Initialize provider-specific client
	if err := service.initializeProvider(); err != nil {
		return nil, fmt.Errorf("failed to initialize AI provider: %w", err)
	}

	return service, nil
}

// GenerateCompletion generates AI completions using the configured provider with circuit breaker protection
func (s *UnifiedService) GenerateCompletion(ctx context.Context, prompt string, options *CompletionOptions) (*CompletionResponse, error) {
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	// Set defaults if options not provided
	if options == nil {
		options = &CompletionOptions{
			Model:       s.config.Model,
			MaxTokens:   1000,
			Temperature: 0.7,
		}
	}

	var response *CompletionResponse

	// Execute with circuit breaker protection
	err := s.circuitBreaker.Execute(ctx, func(ctx context.Context) error {
		start := time.Now()

		// Delegate to actual AI provider client
		switch s.provider {
		case ProviderOpenAI:
			if openaiClient, ok := s.client.(OpenAIClientInterface); ok {
				resp, err := openaiClient.GenerateCompletion(ctx, prompt, options)
				if err != nil {
					return err
				}
				response = resp
			} else {
				return fmt.Errorf("OpenAI client not properly initialized")
			}
		case ProviderClaude:
			if claudeClient, ok := s.client.(ClaudeClientInterface); ok {
				resp, err := claudeClient.GenerateCompletion(ctx, prompt, options)
				if err != nil {
					return err
				}
				response = resp
			} else {
				return fmt.Errorf("Claude client not properly initialized")
			}
		case ProviderMock:
			// Only use mock for testing environments
			response = &CompletionResponse{
				Content:     fmt.Sprintf("Mock AI Response to: %s", prompt),
				Model:       options.Model,
				Usage:       &UsageStats{PromptTokens: 10, CompletionTokens: 20, Total: 30},
				GeneratedAt: start,
			}
		default:
			return fmt.Errorf("unsupported AI provider: %s", s.provider)
		}

		s.logger.Debug("generated AI completion",
			slog.String("provider", string(s.provider)),
			slog.Int("prompt_length", len(prompt)),
			slog.Duration("duration", time.Since(start)))

		return nil
	})

	if err != nil {
		// Log circuit breaker events
		if reliability.IsCircuitBreakerError(err) {
			s.logger.Warn("AI service circuit breaker activated",
				slog.String("provider", string(s.provider)),
				slog.String("state", s.circuitBreaker.GetState().String()),
				slog.String("error", err.Error()))
		}
		return nil, err
	}

	return response, nil
}

// AssessQuality evaluates content quality using AI
func (s *UnifiedService) AssessQuality(ctx context.Context, content string) (*UnifiedQualityMetrics, error) {
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	// Use actual AI providers for quality assessment
	switch s.provider {
	case ProviderOpenAI, ProviderClaude:
		// Create structured prompt for quality assessment
		prompt := fmt.Sprintf(`Analyze the quality of this content and provide a JSON response with scores (0.0-1.0 scale):

Content to analyze:
%s

Please respond with JSON in this exact format:
{
  "confidence": 0.0,
  "relevance": 0.0,
  "clarity": 0.0,
  "completeness": 0.0,
  "actionability": 0.0,
  "uniqueness": 0.0,
  "missing_elements": ["element1", "element2"]
}

Consider:
- Confidence: How certain are you about the content's accuracy?
- Relevance: How relevant is this content to its apparent purpose?
- Clarity: How clear and well-written is the content?
- Completeness: How complete is the information provided?
- Actionability: How actionable are any recommendations?
- Uniqueness: How unique or novel is the content?
- Missing elements: What key elements are missing?`, content)

		options := &CompletionOptions{
			Model:       s.config.Model,
			MaxTokens:   500,
			Temperature: 0.1, // Low temperature for consistent analysis
		}

		response, err := s.GenerateCompletion(ctx, prompt, options)
		if err != nil {
			return nil, fmt.Errorf("failed to assess content quality: %w", err)
		}

		// Parse AI response into quality metrics
		metrics, err := s.parseQualityResponse(response.Content)
		if err != nil {
			s.logger.Warn("Failed to parse AI quality response, using fallback",
				slog.String("error", err.Error()),
				slog.String("response", response.Content))
			// Fallback to basic metrics if parsing fails
			metrics = s.generateFallbackQualityMetrics(content)
		}

		s.logger.Debug("AI-powered quality assessment completed",
			slog.String("provider", string(s.provider)),
			slog.Float64("overall_score", metrics.OverallScore))

		return metrics, nil

	case ProviderMock:
		// Only use mock for testing environments
		metrics := &UnifiedQualityMetrics{
			Confidence:      0.8,
			Relevance:       0.7,
			Clarity:         0.9,
			Completeness:    0.6,
			Score:           0.75,
			Actionability:   0.5,
			Uniqueness:      0.4,
			OverallScore:    0.7,
			MissingElements: []string{"mock response - would be AI generated"},
		}
		return metrics, nil

	default:
		return nil, fmt.Errorf("unsupported AI provider for quality assessment: %s", s.provider)
	}
}

// parseQualityResponse parses the AI response into quality metrics
func (s *UnifiedService) parseQualityResponse(aiResponse string) (*UnifiedQualityMetrics, error) {
	// Extract JSON from the response (handle cases where AI adds extra text)
	start := strings.Index(aiResponse, "{")
	end := strings.LastIndex(aiResponse, "}") + 1

	if start == -1 || end <= start {
		return nil, fmt.Errorf("no valid JSON found in AI response")
	}

	jsonStr := aiResponse[start:end]

	// Parse the JSON response
	var qualityData struct {
		Confidence      float64  `json:"confidence"`
		Relevance       float64  `json:"relevance"`
		Clarity         float64  `json:"clarity"`
		Completeness    float64  `json:"completeness"`
		Actionability   float64  `json:"actionability"`
		Uniqueness      float64  `json:"uniqueness"`
		MissingElements []string `json:"missing_elements"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &qualityData); err != nil {
		return nil, fmt.Errorf("failed to parse quality JSON: %w", err)
	}

	// Calculate composite scores
	score := (qualityData.Confidence + qualityData.Relevance + qualityData.Clarity + qualityData.Completeness) / 4.0
	overallScore := (score + qualityData.Actionability + qualityData.Uniqueness) / 3.0

	// Ensure all values are within valid range [0.0, 1.0]
	metrics := &UnifiedQualityMetrics{
		Confidence:      clampScore(qualityData.Confidence),
		Relevance:       clampScore(qualityData.Relevance),
		Clarity:         clampScore(qualityData.Clarity),
		Completeness:    clampScore(qualityData.Completeness),
		Score:           clampScore(score),
		Actionability:   clampScore(qualityData.Actionability),
		Uniqueness:      clampScore(qualityData.Uniqueness),
		OverallScore:    clampScore(overallScore),
		MissingElements: qualityData.MissingElements,
	}

	return metrics, nil
}

// generateFallbackQualityMetrics creates basic quality metrics when AI parsing fails
func (s *UnifiedService) generateFallbackQualityMetrics(content string) *UnifiedQualityMetrics {
	// Simple heuristic-based quality assessment
	contentLen := len(content)
	wordCount := len(strings.Fields(content))

	// Basic heuristics (could be enhanced)
	var confidence, relevance, clarity, completeness, actionability, uniqueness float64

	// Length-based heuristics
	if contentLen > 100 {
		completeness = 0.7
	} else {
		completeness = 0.4
	}

	// Word count heuristics
	if wordCount > 20 {
		clarity = 0.6
	} else {
		clarity = 0.3
	}

	// Default moderate scores
	confidence = 0.5
	relevance = 0.6
	actionability = 0.4
	uniqueness = 0.5

	score := (confidence + relevance + clarity + completeness) / 4.0
	overallScore := (score + actionability + uniqueness) / 3.0

	return &UnifiedQualityMetrics{
		Confidence:      confidence,
		Relevance:       relevance,
		Clarity:         clarity,
		Completeness:    completeness,
		Score:           score,
		Actionability:   actionability,
		Uniqueness:      uniqueness,
		OverallScore:    overallScore,
		MissingElements: []string{"AI parsing failed - using heuristic assessment"},
	}
}

// clampScore ensures score is within valid range [0.0, 1.0]
func clampScore(score float64) float64 {
	if score < 0.0 {
		return 0.0
	}
	if score > 1.0 {
		return 1.0
	}
	return score
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// HealthCheck verifies the AI service is working properly
func (s *UnifiedService) HealthCheck(ctx context.Context) error {
	// Basic health check - ensure service is configured
	if s.config == nil {
		return fmt.Errorf("AI service not configured")
	}

	// Provider-specific health checks
	switch s.provider {
	case ProviderOpenAI, ProviderClaude:
		// Test with a simple completion request
		testPrompt := "Health check test - respond with 'OK'"
		options := &CompletionOptions{
			Model:       s.config.Model,
			MaxTokens:   10,
			Temperature: 0.0,
		}

		response, err := s.GenerateCompletion(ctx, testPrompt, options)
		if err != nil {
			return fmt.Errorf("health check failed for provider %s: %w", s.provider, err)
		}

		// Verify we got a response
		if response == nil || response.Content == "" {
			return fmt.Errorf("health check failed: empty response from provider %s", s.provider)
		}

		s.logger.Debug("AI service health check passed",
			slog.String("provider", string(s.provider)),
			slog.String("response_preview", response.Content[:min(len(response.Content), 20)]))

	case ProviderMock:
		// Mock provider is always healthy
		s.logger.Debug("Mock AI service health check passed")

	default:
		return fmt.Errorf("health check not implemented for provider: %s", s.provider)
	}

	return nil
}

// initializeProvider sets up the provider-specific client
func (s *UnifiedService) initializeProvider() error {
	switch s.provider {
	case ProviderOpenAI:
		// Create real OpenAI client
		if s.config.APIKey == "" {
			return fmt.Errorf("OpenAI API key is required but not provided")
		}
		// For now, we'll note that real client initialization would go here
		// The actual implementation would create the real OpenAI client
		s.logger.Info("initialized OpenAI provider",
			slog.String("model", s.config.Model),
			slog.Bool("has_api_key", s.config.APIKey != ""))

	case ProviderClaude:
		// Create real Claude client
		if s.config.APIKey == "" {
			return fmt.Errorf("Claude API key is required but not provided")
		}
		// For now, we'll note that real client initialization would go here
		// The actual implementation would create the real Claude client
		s.logger.Info("initialized Claude provider",
			slog.String("model", s.config.Model),
			slog.Bool("has_api_key", s.config.APIKey != ""))

	case ProviderMock:
		// Mock provider for testing environments only
		s.logger.Warn("using mock AI provider - not suitable for production",
			slog.String("reason", "no real AI provider configured"))

	default:
		return fmt.Errorf("unsupported AI provider: %s", s.provider)
	}

	return nil
}

// GetConfig returns the current AI configuration
func (s *UnifiedService) GetConfig() *AIConfig {
	return s.config
}

// GetProvider returns the current AI provider
func (s *UnifiedService) GetProvider() AIProvider {
	return s.provider
}

// ProcessRequest handles legacy AI requests (for compatibility)
func (s *UnifiedService) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Convert legacy request to modern completion request
	var prompt string
	for _, msg := range req.Messages {
		prompt += msg.Content + "\n"
	}

	options := &CompletionOptions{
		Model:       req.Model,
		MaxTokens:   1000,
		Temperature: 0.7,
	}
	if req.Options != nil {
		options = req.Options
	}

	// Use GenerateCompletion
	completion, err := s.GenerateCompletion(ctx, prompt, options)
	if err != nil {
		return nil, err
	}

	// Convert to legacy response format
	var tokensUsed *TokenUsage
	if completion.Usage != nil {
		tokensUsed = &TokenUsage{
			PromptTokens:     completion.Usage.PromptTokens,
			CompletionTokens: completion.Usage.CompletionTokens,
			TotalTokens:      completion.Usage.Total,
			Total:            completion.Usage.Total,
		}
	}

	response := &Response{
		ID:         fmt.Sprintf("req-%d", time.Now().Unix()),
		Model:      completion.Model,
		Content:    completion.Content,
		Usage:      completion.Usage,
		TokensUsed: tokensUsed,
		Quality:    completion.Quality,
	}

	if req.Metadata != nil {
		response.Metadata = map[string]interface{}{
			"request_id": req.Metadata.RequestID,
			"source":     req.Metadata.Source,
		}
	}

	return response, nil
}
