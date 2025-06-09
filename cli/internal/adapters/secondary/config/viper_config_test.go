package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

func TestNewViperConfigManager(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Set temporary config directory
	tempDir := t.TempDir()
	_ = os.Setenv("LMMC_CONFIG_DIR", tempDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	manager, err := NewViperConfigManager(logger)
	require.NoError(t, err)
	require.NotNil(t, manager)

	vcm := manager.(*ViperConfigManager)
	assert.Equal(t, tempDir, vcm.configDir)
}

func TestViperConfigManager_Load(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir := t.TempDir()
	_ = os.Setenv("LMMC_CONFIG_DIR", tempDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	manager, err := NewViperConfigManager(logger)
	require.NoError(t, err)

	// Test loading with no config file (should use defaults)
	config, err := manager.Load()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Verify defaults
	assert.Equal(t, "http://localhost:9080", config.Server.URL)
	assert.Equal(t, "table", config.CLI.OutputFormat)
	assert.True(t, config.CLI.AutoComplete)
	assert.Equal(t, 20, config.CLI.PageSize)
	assert.True(t, config.Storage.CacheEnabled)
	assert.Equal(t, "info", config.Logging.Level)
}

func TestViperConfigManager_SaveAndLoad(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir := t.TempDir()
	_ = os.Setenv("LMMC_CONFIG_DIR", tempDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	manager, err := NewViperConfigManager(logger)
	require.NoError(t, err)

	// Create custom config
	config := &entities.Config{
		Server: entities.ServerConfig{
			URL:     "https://custom.server.com",
			Version: "v2",
			Timeout: 60,
		},
		CLI: entities.CLIConfig{
			DefaultRepository: "my-repo",
			OutputFormat:      "json",
			AutoComplete:      false,
			ColorScheme:       "never",
			PageSize:          50,
			Editor:            "vim",
		},
		Storage: entities.StorageConfig{
			CacheEnabled: false,
			CacheTTL:     600,
			BackupCount:  5,
		},
		Logging: entities.LoggingConfig{
			Level:  "debug",
			Format: "json",
			File:   "/tmp/lmmc.log",
		},
	}

	// Save config
	err = manager.Save(config)
	require.NoError(t, err)

	// Verify config file was created
	configPath := filepath.Join(tempDir, "config.yaml")
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Create new manager and load config
	manager2, err := NewViperConfigManager(logger)
	require.NoError(t, err)

	loadedConfig, err := manager2.Load()
	require.NoError(t, err)

	// Verify loaded config matches saved config
	assert.Equal(t, config.Server.URL, loadedConfig.Server.URL)
	assert.Equal(t, config.Server.Version, loadedConfig.Server.Version)
	assert.Equal(t, config.Server.Timeout, loadedConfig.Server.Timeout)
	assert.Equal(t, config.CLI.DefaultRepository, loadedConfig.CLI.DefaultRepository)
	assert.Equal(t, config.CLI.OutputFormat, loadedConfig.CLI.OutputFormat)
	assert.Equal(t, config.CLI.AutoComplete, loadedConfig.CLI.AutoComplete)
	assert.Equal(t, config.CLI.Editor, loadedConfig.CLI.Editor)
	assert.Equal(t, config.Storage.CacheEnabled, loadedConfig.Storage.CacheEnabled)
	assert.Equal(t, config.Logging.Level, loadedConfig.Logging.Level)
	assert.Equal(t, config.Logging.File, loadedConfig.Logging.File)
}

func TestViperConfigManager_EnvironmentVariables(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir := t.TempDir()
	_ = os.Setenv("LMMC_CONFIG_DIR", tempDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	// Set environment variables
	_ = os.Setenv("LMMC_SERVER_URL", "https://env.server.com")
	_ = os.Setenv("LMMC_CLI_OUTPUT_FORMAT", "plain")
	_ = os.Setenv("LMMC_CLI_PAGE_SIZE", "100")
	_ = os.Setenv("LMMC_STORAGE_CACHE_ENABLED", "false")
	_ = os.Setenv("LMMC_LOGGING_LEVEL", "error")

	defer func() {
		_ = os.Unsetenv("LMMC_SERVER_URL")
		_ = os.Unsetenv("LMMC_CLI_OUTPUT_FORMAT")
		_ = os.Unsetenv("LMMC_CLI_PAGE_SIZE")
		_ = os.Unsetenv("LMMC_STORAGE_CACHE_ENABLED")
		_ = os.Unsetenv("LMMC_LOGGING_LEVEL")
	}()

	manager, err := NewViperConfigManager(logger)
	require.NoError(t, err)

	config, err := manager.Load()
	require.NoError(t, err)

	// Verify environment variables override defaults
	assert.Equal(t, "https://env.server.com", config.Server.URL)
	assert.Equal(t, "plain", config.CLI.OutputFormat)
	assert.Equal(t, 100, config.CLI.PageSize)
	assert.False(t, config.Storage.CacheEnabled)
	assert.Equal(t, "error", config.Logging.Level)
}

func TestViperConfigManager_Set(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir := t.TempDir()
	_ = os.Setenv("LMMC_CONFIG_DIR", tempDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	manager, err := NewViperConfigManager(logger)
	require.NoError(t, err)

	// Test setting various types of values
	tests := []struct {
		key   string
		value string
		check func(t *testing.T, config *entities.Config)
	}{
		{
			key:   "server.url",
			value: "https://new.server.com",
			check: func(t *testing.T, config *entities.Config) {
				assert.Equal(t, "https://new.server.com", config.Server.URL)
			},
		},
		{
			key:   "cli.output_format",
			value: "json",
			check: func(t *testing.T, config *entities.Config) {
				assert.Equal(t, "json", config.CLI.OutputFormat)
			},
		},
		{
			key:   "cli.page_size",
			value: "75",
			check: func(t *testing.T, config *entities.Config) {
				assert.Equal(t, 75, config.CLI.PageSize)
			},
		},
		{
			key:   "storage.cache_enabled",
			value: "false",
			check: func(t *testing.T, config *entities.Config) {
				assert.False(t, config.Storage.CacheEnabled)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			err := manager.Set(tt.key, tt.value)
			require.NoError(t, err)

			config, err := manager.Load()
			require.NoError(t, err)

			tt.check(t, config)
		})
	}

	// Test setting invalid key
	err = manager.Set("invalid.key", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid configuration key")
}

func TestViperConfigManager_Get(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir := t.TempDir()
	_ = os.Setenv("LMMC_CONFIG_DIR", tempDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	manager, err := NewViperConfigManager(logger)
	require.NoError(t, err)

	// Get default values
	value, err := manager.Get("server.url")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:9080", value)

	value, err = manager.Get("cli.page_size")
	require.NoError(t, err)
	assert.Equal(t, 20, value)

	// Get non-existent key
	_, err = manager.Get("non.existent.key")
	assert.Error(t, err)
}

func TestViperConfigManager_Validate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir := t.TempDir()
	_ = os.Setenv("LMMC_CONFIG_DIR", tempDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	manager, err := NewViperConfigManager(logger)
	require.NoError(t, err)

	// Default config should be valid
	err = manager.Validate()
	assert.NoError(t, err)

	// Test invalid configuration by directly modifying viper
	vcm := manager.(*ViperConfigManager)
	vcm.viper.Set("server.url", "not-a-url")
	vcm.viper.Set("cli.output_format", "invalid")
	vcm.viper.Set("logging.level", "invalid")

	err = manager.Validate()
	assert.Error(t, err)
}

func TestViperConfigManager_Reset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir := t.TempDir()
	_ = os.Setenv("LMMC_CONFIG_DIR", tempDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	manager, err := NewViperConfigManager(logger)
	require.NoError(t, err)

	// Modify some settings
	err = manager.Set("server.url", "https://custom.server.com")
	require.NoError(t, err)
	err = manager.Set("cli.page_size", "50")
	require.NoError(t, err)

	// Reset to defaults
	err = manager.Reset()
	require.NoError(t, err)

	// Verify defaults are restored
	config, err := manager.Load()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:9080", config.Server.URL)
	assert.Equal(t, 20, config.CLI.PageSize)
}

func TestViperConfigManager_GetConfigPath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir := t.TempDir()
	_ = os.Setenv("LMMC_CONFIG_DIR", tempDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	manager, err := NewViperConfigManager(logger)
	require.NoError(t, err)

	expectedPath := filepath.Join(tempDir, "config.yaml")
	assert.Equal(t, expectedPath, manager.GetConfigPath())
}

func TestGetConfigDirectory(t *testing.T) {
	// Test XDG_CONFIG_HOME
	xdgHome := "/custom/xdg/config"
	_ = os.Setenv("XDG_CONFIG_HOME", xdgHome)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	dir, err := getConfigDirectory()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(xdgHome, "lmmc"), dir)

	_ = os.Unsetenv("XDG_CONFIG_HOME")

	// Test LMMC_CONFIG_DIR
	customDir := "/custom/lmmc/config"
	_ = os.Setenv("LMMC_CONFIG_DIR", customDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	dir, err = getConfigDirectory()
	require.NoError(t, err)
	assert.Equal(t, customDir, dir)

	_ = os.Unsetenv("LMMC_CONFIG_DIR")

	// Test fallback to home directory
	homeDir, _ := os.UserHomeDir()
	dir, err = getConfigDirectory()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(homeDir, ".lmmc"), dir)
}

func TestFlattenMap(t *testing.T) {
	input := map[string]interface{}{
		"server": map[string]interface{}{
			"url":     "http://localhost",
			"timeout": 30,
		},
		"cli": map[string]interface{}{
			"output_format": "table",
			"page_size":     20,
		},
		"simple": "value",
	}

	expected := map[string]interface{}{
		"server.url":        "http://localhost",
		"server.timeout":    30,
		"cli.output_format": "table",
		"cli.page_size":     20,
		"simple":            "value",
	}

	result := flattenMap("", input)
	assert.Equal(t, expected, result)
}

// Integration test
func TestViperConfigManager_Integration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir := t.TempDir()

	// Create a config file manually
	configContent := `server:
  url: https://test.server.com
  timeout: 45
cli:
  output_format: json
  page_size: 30
  auto_complete: false
logging:
  level: debug
`
	configPath := filepath.Join(tempDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0o600)
	require.NoError(t, err)

	// Set environment to use our temp directory
	_ = os.Setenv("LMMC_CONFIG_DIR", tempDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	// Also set an environment variable to test override
	_ = os.Setenv("LMMC_CLI_PAGE_SIZE", "100")
	defer func() { _ = os.Unsetenv("LMMC_CLI_PAGE_SIZE") }()

	// Create manager and load config
	manager, err := NewViperConfigManager(logger)
	require.NoError(t, err)

	config, err := manager.Load()
	require.NoError(t, err)

	// Verify file values are loaded
	assert.Equal(t, "https://test.server.com", config.Server.URL)
	assert.Equal(t, 45, config.Server.Timeout)
	assert.Equal(t, "json", config.CLI.OutputFormat)
	assert.False(t, config.CLI.AutoComplete)
	assert.Equal(t, "debug", config.Logging.Level)

	// Verify environment variable overrides file
	assert.Equal(t, 100, config.CLI.PageSize)
}

// Benchmark test
func BenchmarkViperConfigManager_Load(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir := b.TempDir()
	_ = os.Setenv("LMMC_CONFIG_DIR", tempDir)
	defer func() { _ = os.Unsetenv("LMMC_CONFIG_DIR") }()

	manager, err := NewViperConfigManager(logger)
	if err != nil {
		b.Fatal(err)
	}

	// Create a config file
	config := entities.DefaultConfig()
	if err := manager.Save(config); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.Load()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Mock ConfigManager for testing
type MockConfigManager struct {
	config     *entities.Config
	configPath string
	data       map[string]interface{}
}

func NewMockConfigManager() ports.ConfigManager {
	return &MockConfigManager{
		config:     entities.DefaultConfig(),
		configPath: "/tmp/mock-config.yaml",
		data:       make(map[string]interface{}),
	}
}

func (m *MockConfigManager) Load() (*entities.Config, error) {
	return m.config, nil
}

func (m *MockConfigManager) Save(config *entities.Config) error {
	m.config = config
	return nil
}

func (m *MockConfigManager) Set(key, value string) error {
	m.data[key] = value
	return nil
}

func (m *MockConfigManager) Get(key string) (interface{}, error) {
	if value, ok := m.data[key]; ok {
		return value, nil
	}
	return nil, os.ErrNotExist
}

func (m *MockConfigManager) GetConfigPath() string {
	return m.configPath
}

func (m *MockConfigManager) Validate() error {
	return nil
}

func (m *MockConfigManager) Reset() error {
	m.config = entities.DefaultConfig()
	m.data = make(map[string]interface{})
	return nil
}
