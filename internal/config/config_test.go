package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testAPIKey = "test-key"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Server defaults
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Server.Host)
	assert.Equal(t, 30, cfg.Server.ReadTimeout)
	assert.Equal(t, 30, cfg.Server.WriteTimeout)

	// Chroma defaults
	assert.Equal(t, "http://localhost:9000", cfg.Chroma.Endpoint)
	assert.Equal(t, "claude_memory", cfg.Chroma.Collection)
	assert.True(t, cfg.Chroma.HealthCheck)
	assert.Equal(t, 3, cfg.Chroma.RetryAttempts)
	assert.Equal(t, 30, cfg.Chroma.TimeoutSeconds)

	// Docker defaults
	assert.True(t, cfg.Chroma.Docker.Enabled)
	assert.Equal(t, "claude-memory-chroma", cfg.Chroma.Docker.ContainerName)
	assert.Equal(t, "./data/chroma", cfg.Chroma.Docker.VolumePath)
	assert.Equal(t, "ghcr.io/chroma-core/chroma:latest", cfg.Chroma.Docker.Image)

	// OpenAI defaults
	assert.Equal(t, "text-embedding-ada-002", cfg.OpenAI.EmbeddingModel)
	assert.Equal(t, 8191, cfg.OpenAI.MaxTokens)
	assert.Equal(t, 0.0, cfg.OpenAI.Temperature)
	assert.Equal(t, 60, cfg.OpenAI.RequestTimeout)
	assert.Equal(t, 60, cfg.OpenAI.RateLimitRPM)

	// Storage defaults
	assert.Equal(t, "chroma", cfg.Storage.Provider)
	assert.Equal(t, 90, cfg.Storage.RetentionDays)
	assert.False(t, cfg.Storage.BackupEnabled)
	assert.Equal(t, 24, cfg.Storage.BackupInterval)
	assert.NotNil(t, cfg.Storage.Repositories)

	// Chunking defaults
	assert.Equal(t, "smart", cfg.Chunking.Strategy)
	assert.Equal(t, 50, cfg.Chunking.MinContentLength)
	assert.Equal(t, 10000, cfg.Chunking.MaxContentLength)
	assert.True(t, cfg.Chunking.TodoCompletionTrigger)
	assert.Equal(t, 3, cfg.Chunking.FileChangeThreshold)
	assert.Equal(t, 10, cfg.Chunking.TimeThresholdMinutes)
	assert.Equal(t, 0.8, cfg.Chunking.SimilarityThreshold)

	// Logging defaults
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, 10, cfg.Logging.MaxSize)
	assert.Equal(t, 3, cfg.Logging.MaxBackups)
	assert.Equal(t, 30, cfg.Logging.MaxAge)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  func() *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				return cfg
			},
			wantErr: false,
		},
		{
			name: "invalid server port - too low",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.Server.Port = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "invalid server port - too high",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.Server.Port = 70000
				return cfg
			},
			wantErr: true,
			errMsg:  "invalid server port",
		},
		{
			name: "empty server host",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.Server.Host = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "server host cannot be empty",
		},
		{
			name: "empty chroma endpoint",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.Chroma.Endpoint = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "chroma endpoint cannot be empty",
		},
		{
			name: "empty chroma collection",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.Chroma.Collection = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "chroma collection cannot be empty",
		},
		{
			name: "empty docker container name with docker enabled",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.Chroma.Docker.Enabled = true
				cfg.Chroma.Docker.ContainerName = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "docker container name cannot be empty when docker is enabled",
		},
		{
			name: "missing OpenAI API key",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "OpenAI API key is required",
		},
		{
			name: "empty embedding model",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.OpenAI.EmbeddingModel = ""
				return cfg
			},
			wantErr: true,
			errMsg:  "OpenAI embedding model cannot be empty",
		},
		{
			name: "invalid retention days",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.Storage.RetentionDays = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "retention days must be positive",
		},
		{
			name: "invalid min content length",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.Chunking.MinContentLength = 0
				return cfg
			},
			wantErr: true,
			errMsg:  "min content length must be positive",
		},
		{
			name: "invalid max content length",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.Chunking.MaxContentLength = 40 // Less than min (50)
				return cfg
			},
			wantErr: true,
			errMsg:  "max content length must be greater than min content length",
		},
		{
			name: "invalid similarity threshold - too low",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.Chunking.SimilarityThreshold = -0.1
				return cfg
			},
			wantErr: true,
			errMsg:  "similarity threshold must be between 0 and 1",
		},
		{
			name: "invalid similarity threshold - too high",
			config: func() *Config {
				cfg := DefaultConfig()
				cfg.OpenAI.APIKey = testAPIKey
				cfg.Chunking.SimilarityThreshold = 1.1
				return cfg
			},
			wantErr: true,
			errMsg:  "similarity threshold must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config()
			err := cfg.Validate()

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

func TestLoadConfig_WithEnvVars(t *testing.T) {
	// Set up environment variables
	envVars := map[string]string{
		"MCP_MEMORY_PORT":        "9090",
		"MCP_MEMORY_HOST":        "0.0.0.0",
		"CHROMA_ENDPOINT":        "http://custom:8001",
		"CHROMA_COLLECTION":      "custom_memory",
		"CHROMA_CONTAINER_NAME":  "custom-chroma",
		"CHROMA_VOLUME_PATH":     "/custom/data",
		"OPENAI_API_KEY":         "test-api-key",
		"OPENAI_EMBEDDING_MODEL": "text-embedding-3-small",
		"RETENTION_DAYS":         "30",
		"LOG_LEVEL":              "debug",
		"LOG_FORMAT":             "text",
		"LOG_FILE":               "/var/log/memory.log",
	}

	// Set environment variables
	for key, value := range envVars {
		_ = os.Setenv(key, value)
	}

	// Clean up after test
	defer func() {
		for key := range envVars {
			_ = os.Unsetenv(key)
		}
	}()

	cfg, err := LoadConfig()
	require.NoError(t, err)

	// Verify overrides
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, "http://custom:8001", cfg.Chroma.Endpoint)
	assert.Equal(t, "custom_memory", cfg.Chroma.Collection)
	assert.Equal(t, "custom-chroma", cfg.Chroma.Docker.ContainerName)
	assert.Equal(t, "/custom/data", cfg.Chroma.Docker.VolumePath)
	assert.Equal(t, "test-api-key", cfg.OpenAI.APIKey)
	assert.Equal(t, "text-embedding-3-small", cfg.OpenAI.EmbeddingModel)
	assert.Equal(t, 30, cfg.Storage.RetentionDays)
	assert.Equal(t, "debug", cfg.Logging.Level)
	assert.Equal(t, "text", cfg.Logging.Format)
	assert.Equal(t, "/var/log/memory.log", cfg.Logging.File)
}

func TestLoadConfig_WithInvalidEnvVars(t *testing.T) {
	// Set invalid port
	_ = os.Setenv("MCP_MEMORY_PORT", "invalid")
	_ = os.Setenv("OPENAI_API_KEY", testAPIKey)

	defer func() {
		_ = os.Unsetenv("MCP_MEMORY_PORT")
		_ = os.Unsetenv("OPENAI_API_KEY")
	}()

	cfg, err := LoadConfig()
	require.NoError(t, err)

	// Should use default port when invalid port is provided
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestConfig_GetDataDir(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("default data directory", func(t *testing.T) {
		dataDir, err := cfg.GetDataDir()
		require.NoError(t, err)

		// Should be absolute path
		assert.True(t, filepath.IsAbs(dataDir))

		// Directory should exist after call
		_, err = os.Stat(dataDir)
		assert.NoError(t, err)
	})

	t.Run("custom data directory", func(t *testing.T) {
		cfg.Chroma.Docker.VolumePath = "./test-data"

		dataDir, err := cfg.GetDataDir()
		require.NoError(t, err)

		assert.True(t, filepath.IsAbs(dataDir))

		// Clean up
		_ = os.RemoveAll(dataDir)
	})
}

func TestConfig_GetRepoConfig(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("existing repository config", func(t *testing.T) {
		repoConfig := RepoConfig{
			Enabled:         false,
			Sensitivity:     "high",
			ExcludePatterns: []string{"*.secret"},
			Tags:            []string{"sensitive"},
		}
		cfg.SetRepoConfig("test-repo", repoConfig)

		result := cfg.GetRepoConfig("test-repo")
		assert.Equal(t, repoConfig, result)
	})

	t.Run("non-existing repository config", func(t *testing.T) {
		result := cfg.GetRepoConfig("non-existing-repo")

		// Should return default config
		assert.True(t, result.Enabled)
		assert.Equal(t, "normal", result.Sensitivity)
		assert.Contains(t, result.ExcludePatterns, "*.env")
		assert.Contains(t, result.ExcludePatterns, "*.key")
		assert.Empty(t, result.Tags)
	})
}

func TestConfig_SetRepoConfig(t *testing.T) {
	cfg := DefaultConfig()

	repoConfig := RepoConfig{
		Enabled:         false,
		Sensitivity:     "high",
		ExcludePatterns: []string{"*.secret"},
		Tags:            []string{"sensitive"},
	}

	cfg.SetRepoConfig("test-repo", repoConfig)

	// Verify it was set
	result := cfg.GetRepoConfig("test-repo")
	assert.Equal(t, repoConfig, result)

	// Verify it's in the repositories map
	assert.Contains(t, cfg.Storage.Repositories, "test-repo")
	assert.Equal(t, repoConfig, cfg.Storage.Repositories["test-repo"])
}

func TestConfig_IsRepositoryEnabled(t *testing.T) {
	cfg := DefaultConfig()

	t.Run("default repository - enabled", func(t *testing.T) {
		enabled := cfg.IsRepositoryEnabled("any-repo")
		assert.True(t, enabled)
	})

	t.Run("explicitly disabled repository", func(t *testing.T) {
		cfg.SetRepoConfig("disabled-repo", RepoConfig{
			Enabled:     false,
			Sensitivity: "normal",
		})

		enabled := cfg.IsRepositoryEnabled("disabled-repo")
		assert.False(t, enabled)
	})

	t.Run("explicitly enabled repository", func(t *testing.T) {
		cfg.SetRepoConfig("enabled-repo", RepoConfig{
			Enabled:     true,
			Sensitivity: "normal",
		})

		enabled := cfg.IsRepositoryEnabled("enabled-repo")
		assert.True(t, enabled)
	})
}

func TestLoadConfig_MissingEnvFile(t *testing.T) {
	// Ensure no .env file exists by using a temp directory
	originalWd, _ := os.Getwd()
	tempDir := t.TempDir()
	_ = os.Chdir(tempDir)
	defer func() { _ = os.Chdir(originalWd) }()

	// Set required env var
	_ = os.Setenv("OPENAI_API_KEY", testAPIKey)
	defer func() { _ = os.Unsetenv("OPENAI_API_KEY") }()

	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestLoadConfig_InvalidConfig(t *testing.T) {
	// Set environment that will result in invalid config
	_ = os.Setenv("OPENAI_API_KEY", "") // Empty API key should cause validation error
	defer func() { _ = os.Unsetenv("OPENAI_API_KEY") }()

	_, err := LoadConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid configuration")
}
