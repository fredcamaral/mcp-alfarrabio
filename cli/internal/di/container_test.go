package di

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

func TestNewContainer(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Create minimal config file
	configDir := filepath.Join(tempDir, ".lmmc")
	err := os.MkdirAll(configDir, 0o750)
	require.NoError(t, err)

	configFile := filepath.Join(configDir, "config.yaml")
	configContent := `
server:
  url: "http://localhost:9080"
  timeout: 5
cli:
  output_format: "table"
logging:
  level: "info"
  format: "text"
`
	err = os.WriteFile(configFile, []byte(configContent), 0o600)
	require.NoError(t, err)

	// Create container
	container, err := NewContainer()
	require.NoError(t, err)
	require.NotNil(t, container)

	// Verify all components are initialized
	assert.NotNil(t, container.Config)
	assert.NotNil(t, container.ConfigManager)
	assert.NotNil(t, container.Logger)
	assert.NotNil(t, container.Storage)
	assert.NotNil(t, container.MCPClient)
	assert.NotNil(t, container.TaskService)
	assert.NotNil(t, container.RepositoryDetector)
	assert.NotNil(t, container.CLI)

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = container.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestNewTestContainer(t *testing.T) {
	// Create test config
	testConfig := &entities.Config{
		Server: entities.ServerConfig{
			URL:     "http://localhost:9999",
			Timeout: 5,
		},
		CLI: entities.CLIConfig{
			OutputFormat: "json",
			PageSize:     20,
		},
		Storage: entities.StorageConfig{
			CacheEnabled: true,
			CacheTTL:     300,
			BackupCount:  3,
		},
		Logging: entities.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}

	// Create test container
	container, err := NewTestContainer(testConfig)
	require.NoError(t, err)
	require.NotNil(t, container)

	// Verify config was applied
	assert.Equal(t, testConfig, container.Config)
	assert.NotNil(t, container.Logger)
	assert.NotNil(t, container.Storage)
	assert.NotNil(t, container.MCPClient)
	assert.NotNil(t, container.TaskService)
	assert.NotNil(t, container.RepositoryDetector)
	assert.NotNil(t, container.CLI)

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = container.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestContainer_HealthCheck(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Create test config
	testConfig := &entities.Config{
		Server: entities.ServerConfig{
			URL:     "", // No MCP server
			Timeout: 5,
		},
		CLI: entities.CLIConfig{
			OutputFormat: "table",
		},
		Storage: entities.StorageConfig{
			CacheEnabled: true,
			CacheTTL:     300,
			BackupCount:  3,
		},
		Logging: entities.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	// Create container
	container, err := NewTestContainer(testConfig)
	require.NoError(t, err)

	// Run health check
	ctx := context.Background()
	err = container.HealthCheck(ctx)
	assert.NoError(t, err)

	// Cleanup
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = container.Shutdown(shutdownCtx)
	assert.NoError(t, err)
}

func TestContainer_LoggerConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		config       entities.LoggingConfig
		expectFile   bool
		expectFormat string
	}{
		{
			name: "console text logger",
			config: entities.LoggingConfig{
				Level:  "info",
				Format: "text",
				File:   "",
			},
			expectFile:   false,
			expectFormat: "text",
		},
		{
			name: "console json logger",
			config: entities.LoggingConfig{
				Level:  "debug",
				Format: "json",
				File:   "",
			},
			expectFile:   false,
			expectFormat: "json",
		},
		{
			name: "file text logger",
			config: entities.LoggingConfig{
				Level:  "warn",
				Format: "text",
				File:   "test.log",
			},
			expectFile:   true,
			expectFormat: "text",
		},
		{
			name: "file json logger",
			config: entities.LoggingConfig{
				Level:  "error",
				Format: "json",
				File:   "test.log",
			},
			expectFile:   true,
			expectFormat: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temp directory for log files
			tempDir := t.TempDir()
			if tt.config.File != "" {
				tt.config.File = filepath.Join(tempDir, tt.config.File)
			}

			// Create test config
			testConfig := &entities.Config{
				Server: entities.ServerConfig{
					URL:     "",
					Timeout: 5,
				},
				CLI: entities.CLIConfig{
					OutputFormat: "table",
				},
				Storage: entities.StorageConfig{
					CacheEnabled: true,
					CacheTTL:     300,
					BackupCount:  3,
				},
				Logging: tt.config,
			}

			// Create container
			container, err := NewTestContainer(testConfig)
			require.NoError(t, err)

			// Verify logger is configured
			assert.NotNil(t, container.Logger)

			// If file logging, verify file exists
			if tt.expectFile {
				assert.FileExists(t, tt.config.File)
			}

			// Cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = container.Shutdown(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestContainer_MCPClientConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		serverURL      string
		expectOnline   bool
		expectDisabled bool
	}{
		{
			name:           "MCP enabled with valid URL",
			serverURL:      "http://localhost:9080",
			expectOnline:   false, // Will be offline since server doesn't exist
			expectDisabled: false,
		},
		{
			name:           "MCP disabled with empty URL",
			serverURL:      "",
			expectOnline:   false,
			expectDisabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test config
			testConfig := &entities.Config{
				Server: entities.ServerConfig{
					URL:     tt.serverURL,
					Timeout: 1, // Short timeout for tests
				},
				CLI: entities.CLIConfig{
					OutputFormat: "table",
				},
				Storage: entities.StorageConfig{
					CacheEnabled: true,
					CacheTTL:     300,
					BackupCount:  3,
				},
				Logging: entities.LoggingConfig{
					Level:  "info",
					Format: "text",
				},
			}

			// Create container
			container, err := NewTestContainer(testConfig)
			require.NoError(t, err)

			// Verify MCP client
			assert.NotNil(t, container.MCPClient)

			// Check online status
			assert.Equal(t, tt.expectOnline, container.MCPClient.IsOnline())

			// Cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = container.Shutdown(ctx)
			assert.NoError(t, err)
		})
	}
}

func TestContainer_Shutdown(t *testing.T) {
	// Setup test environment
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// Create test config with file logging
	testConfig := &entities.Config{
		Server: entities.ServerConfig{
			URL:     "",
			Timeout: 5,
		},
		CLI: entities.CLIConfig{
			OutputFormat: "table",
		},
		Storage: entities.StorageConfig{
			CacheEnabled: true,
			CacheTTL:     300,
			BackupCount:  3,
		},
		Logging: entities.LoggingConfig{
			Level:  "info",
			Format: "text",
			File:   logFile,
		},
	}

	// Create container
	container, err := NewTestContainer(testConfig)
	require.NoError(t, err)

	// Verify log file is created
	assert.FileExists(t, logFile)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = container.Shutdown(ctx)
	assert.NoError(t, err)

	// Verify clean shutdown
	assert.NotNil(t, container.logFile) // File should be closed but reference remains
}

func TestContainer_InitializationError(t *testing.T) {
	// Setup invalid environment (no HOME directory)
	originalHome := os.Getenv("HOME")
	_ = os.Unsetenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Try to create container - should fail
	container, err := NewContainer()
	assert.Error(t, err)
	assert.Nil(t, container)
	assert.Contains(t, err.Error(), "failed to initialize config")
}
