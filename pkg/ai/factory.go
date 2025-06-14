// Package ai provides factory functions for creating AI services
package ai

import (
	"log/slog"
	"os"
	"time"
)

// Provider constants
const (
	ProviderClaude     = "claude"
	ProviderOpenAI     = "openai"
	ProviderPerplexity = "perplexity"
)

// NewFromConfig creates an AI service from configuration
func NewFromConfig(config *Config, logger *slog.Logger) (*Service, error) {
	return NewService(config, logger)
}

// NewFromEnv creates an AI service configured from environment variables
func NewFromEnv(logger *slog.Logger) (*Service, error) {
	provider := determineProvider()
	config := createBaseConfig(provider)
	applyProviderSpecificConfig(config)

	return NewService(config, logger)
}

// determineProvider determines the AI provider to use
func determineProvider() string {
	provider := os.Getenv("AI_PROVIDER")
	if provider == "" {
		provider = autoDetectProvider()
	}
	return provider
}

// createBaseConfig creates the base configuration with defaults
func createBaseConfig(provider string) *Config {
	return &Config{
		Provider:   provider,
		APIKey:     os.Getenv("AI_API_KEY"),
		BaseURL:    getEnvOrDefault("AI_BASE_URL", ""),
		Model:      getEnvOrDefault("AI_MODEL", ""),
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// applyProviderSpecificConfig applies provider-specific configuration overrides
func applyProviderSpecificConfig(config *Config) {
	switch config.Provider {
	case ProviderClaude:
		applyClaudeConfig(config)
	case ProviderOpenAI:
		applyOpenAIConfig(config)
	case ProviderPerplexity:
		applyPerplexityConfig(config)
	}
}

// applyClaudeConfig applies Claude-specific configuration
func applyClaudeConfig(config *Config) {
	overrideFromEnv(&config.APIKey, "CLAUDE_API_KEY")
	overrideFromEnv(&config.BaseURL, "CLAUDE_BASE_URL")
	overrideFromEnv(&config.Model, "CLAUDE_MODEL")

	setDefaultIfEmpty(&config.BaseURL, "https://api.anthropic.com")
	setDefaultIfEmpty(&config.Model, "claude-sonnet-4")
}

// applyOpenAIConfig applies OpenAI-specific configuration
func applyOpenAIConfig(config *Config) {
	overrideFromEnv(&config.APIKey, "OPENAI_API_KEY")
	overrideFromEnv(&config.BaseURL, "OPENAI_BASE_URL")
	overrideFromEnv(&config.Model, "OPENAI_MODEL")

	setDefaultIfEmpty(&config.BaseURL, "https://api.openai.com/v1")
	setDefaultIfEmpty(&config.Model, "gpt-4o")
}

// applyPerplexityConfig applies Perplexity-specific configuration
func applyPerplexityConfig(config *Config) {
	overrideFromEnv(&config.APIKey, "PERPLEXITY_API_KEY")
	overrideFromEnv(&config.BaseURL, "PERPLEXITY_BASE_URL")
	overrideFromEnv(&config.Model, "PERPLEXITY_MODEL")

	setDefaultIfEmpty(&config.BaseURL, "https://api.perplexity.ai")
	setDefaultIfEmpty(&config.Model, "sonar-pro")
}

// overrideFromEnv overrides a config field with environment variable if it exists
func overrideFromEnv(field *string, envKey string) {
	if value := os.Getenv(envKey); value != "" {
		*field = value
	}
}

// setDefaultIfEmpty sets a default value if the field is empty
func setDefaultIfEmpty(field *string, defaultValue string) {
	if *field == "" {
		*field = defaultValue
	}
}

// NewMockService creates a service with mock AI for testing
func NewMockService(logger *slog.Logger) (*Service, error) {
	config := &Config{
		Provider:   "mock",
		Model:      "mock-model-1.0",
		Timeout:    time.Second,
		MaxRetries: 1,
		RetryDelay: 100 * time.Millisecond,
	}

	return NewService(config, logger)
}

// autoDetectProvider automatically detects the AI provider based on available API keys
// Priority: Claude > OpenAI > Perplexity > Mock
func autoDetectProvider() string {
	// Check for Claude API key first (often most capable for complex tasks)
	if os.Getenv("CLAUDE_API_KEY") != "" {
		return ProviderClaude
	}

	// Check for OpenAI API key second (widely used, good compatibility)
	if os.Getenv("OPENAI_API_KEY") != "" {
		return ProviderOpenAI
	}

	// Check for Perplexity API key third (good for research/search tasks)
	if os.Getenv("PERPLEXITY_API_KEY") != "" {
		return ProviderPerplexity
	}

	// Fall back to mock provider if no real API keys found
	return "mock"
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
