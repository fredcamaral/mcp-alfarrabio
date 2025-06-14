package ai

import (
	"os"
	"testing"
)

func TestAutoDetectProvider(t *testing.T) {
	// Save original environment
	originalClaude := os.Getenv("CLAUDE_API_KEY")
	originalOpenAI := os.Getenv("OPENAI_API_KEY")
	originalPerplexity := os.Getenv("PERPLEXITY_API_KEY")

	// Clean up after test
	defer func() {
		if originalClaude != "" {
			_ = os.Setenv("CLAUDE_API_KEY", originalClaude)
		} else {
			_ = os.Unsetenv("CLAUDE_API_KEY")
		}
		if originalOpenAI != "" {
			_ = os.Setenv("OPENAI_API_KEY", originalOpenAI)
		} else {
			_ = os.Unsetenv("OPENAI_API_KEY")
		}
		if originalPerplexity != "" {
			_ = os.Setenv("PERPLEXITY_API_KEY", originalPerplexity)
		} else {
			_ = os.Unsetenv("PERPLEXITY_API_KEY")
		}
	}()

	tests := []struct {
		name             string
		claudeKey        string
		openaiKey        string
		perplexityKey    string
		expectedProvider string
	}{
		{
			name:             "No API keys - should default to mock",
			claudeKey:        "",
			openaiKey:        "",
			perplexityKey:    "",
			expectedProvider: "mock",
		},
		{
			name:             "Only Claude key - should choose Claude",
			claudeKey:        "sk-ant-test",
			openaiKey:        "",
			perplexityKey:    "",
			expectedProvider: "claude",
		},
		{
			name:             "Only OpenAI key - should choose OpenAI",
			claudeKey:        "",
			openaiKey:        "sk-test",
			perplexityKey:    "",
			expectedProvider: "openai",
		},
		{
			name:             "Only Perplexity key - should choose Perplexity",
			claudeKey:        "",
			openaiKey:        "",
			perplexityKey:    "pplx-test",
			expectedProvider: "perplexity",
		},
		{
			name:             "Claude and OpenAI keys - should choose Claude (higher priority)",
			claudeKey:        "sk-ant-test",
			openaiKey:        "sk-test",
			perplexityKey:    "",
			expectedProvider: "claude",
		},
		{
			name:             "OpenAI and Perplexity keys - should choose OpenAI (higher priority)",
			claudeKey:        "",
			openaiKey:        "sk-test",
			perplexityKey:    "pplx-test",
			expectedProvider: "openai",
		},
		{
			name:             "All three keys - should choose Claude (highest priority)",
			claudeKey:        "sk-ant-test",
			openaiKey:        "sk-test",
			perplexityKey:    "pplx-test",
			expectedProvider: "claude",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all environment variables first
			_ = os.Unsetenv("CLAUDE_API_KEY")
			_ = os.Unsetenv("OPENAI_API_KEY")
			_ = os.Unsetenv("PERPLEXITY_API_KEY")

			// Set test environment variables
			if tt.claudeKey != "" {
				_ = os.Setenv("CLAUDE_API_KEY", tt.claudeKey)
			}
			if tt.openaiKey != "" {
				_ = os.Setenv("OPENAI_API_KEY", tt.openaiKey)
			}
			if tt.perplexityKey != "" {
				_ = os.Setenv("PERPLEXITY_API_KEY", tt.perplexityKey)
			}

			// Test auto-detection
			result := autoDetectProvider()
			if result != tt.expectedProvider {
				t.Errorf("autoDetectProvider() = %v, want %v", result, tt.expectedProvider)
			}
		})
	}
}

func TestNewFromEnvWithAutoDetection(t *testing.T) {
	// Save original environment
	originalProvider := os.Getenv("AI_PROVIDER")
	originalClaude := os.Getenv("CLAUDE_API_KEY")
	originalOpenAI := os.Getenv("OPENAI_API_KEY")

	// Clean up after test
	defer func() {
		if originalProvider != "" {
			_ = os.Setenv("AI_PROVIDER", originalProvider)
		} else {
			_ = os.Unsetenv("AI_PROVIDER")
		}
		if originalClaude != "" {
			_ = os.Setenv("CLAUDE_API_KEY", originalClaude)
		} else {
			_ = os.Unsetenv("CLAUDE_API_KEY")
		}
		if originalOpenAI != "" {
			_ = os.Setenv("OPENAI_API_KEY", originalOpenAI)
		} else {
			_ = os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	t.Run("Explicit AI_PROVIDER overrides auto-detection", func(t *testing.T) {
		// Set up environment
		_ = os.Setenv("AI_PROVIDER", "openai")
		_ = os.Setenv("CLAUDE_API_KEY", "sk-ant-test")
		_ = os.Setenv("OPENAI_API_KEY", "sk-test")

		// Create service
		service, err := NewFromEnv(nil)
		if err != nil {
			t.Fatalf("NewFromEnv() error = %v", err)
		}

		// Should use explicitly set provider (openai) despite having Claude key
		if service.config.Provider != "openai" {
			t.Errorf("Expected provider 'openai', got '%s'", service.config.Provider)
		}
	})

	t.Run("Auto-detection when AI_PROVIDER not set", func(t *testing.T) {
		// Clear AI_PROVIDER
		_ = os.Unsetenv("AI_PROVIDER")
		_ = os.Setenv("CLAUDE_API_KEY", "sk-ant-test")
		_ = os.Unsetenv("OPENAI_API_KEY")

		// Create service
		service, err := NewFromEnv(nil)
		if err != nil {
			t.Fatalf("NewFromEnv() error = %v", err)
		}

		// Should auto-detect Claude
		if service.config.Provider != "claude" {
			t.Errorf("Expected provider 'claude', got '%s'", service.config.Provider)
		}
	})
}
