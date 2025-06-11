// Package ai provides factory functions for creating AI services
package ai

import (
	"log/slog"
	"os"
	"time"
)

// NewFromConfig creates an AI service from configuration
func NewFromConfig(config *Config, logger *slog.Logger) (*Service, error) {
	return NewService(config, logger)
}

// NewFromEnv creates an AI service configured from environment variables
func NewFromEnv(logger *slog.Logger) (*Service, error) {
	config := &Config{
		Provider:   getEnvOrDefault("AI_PROVIDER", "mock"),
		APIKey:     os.Getenv("AI_API_KEY"),
		BaseURL:    getEnvOrDefault("AI_BASE_URL", ""),
		Model:      getEnvOrDefault("AI_MODEL", ""),
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}

	// Override with provider-specific environment variables
	switch config.Provider {
	case "claude":
		if apiKey := os.Getenv("CLAUDE_API_KEY"); apiKey != "" {
			config.APIKey = apiKey
		}
		if baseURL := os.Getenv("CLAUDE_BASE_URL"); baseURL != "" {
			config.BaseURL = baseURL
		}
		if model := os.Getenv("CLAUDE_MODEL"); model != "" {
			config.Model = model
		}
		if config.BaseURL == "" {
			config.BaseURL = "https://api.anthropic.com"
		}
		if config.Model == "" {
			config.Model = "claude-sonnet-4"
		}

	case "openai":
		if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
			config.APIKey = apiKey
		}
		if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
			config.BaseURL = baseURL
		}
		if model := os.Getenv("OPENAI_MODEL"); model != "" {
			config.Model = model
		}
		if config.BaseURL == "" {
			config.BaseURL = "https://api.openai.com/v1"
		}
		if config.Model == "" {
			config.Model = "gpt-4o"
		}

	case "perplexity":
		if apiKey := os.Getenv("PERPLEXITY_API_KEY"); apiKey != "" {
			config.APIKey = apiKey
		}
		if baseURL := os.Getenv("PERPLEXITY_BASE_URL"); baseURL != "" {
			config.BaseURL = baseURL
		}
		if model := os.Getenv("PERPLEXITY_MODEL"); model != "" {
			config.Model = model
		}
		if config.BaseURL == "" {
			config.BaseURL = "https://api.perplexity.ai"
		}
		if config.Model == "" {
			config.Model = "sonar-pro"
		}
	}

	return NewService(config, logger)
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

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
