// Package config provides configuration management using Viper
// for the lerian-mcp-memory CLI application.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// ViperConfigManager implements ConfigManager using Viper
type ViperConfigManager struct {
	viper     *viper.Viper
	validator *validator.Validate
	configDir string
	logger    *slog.Logger
}

// NewViperConfigManager creates a new Viper-based configuration manager
func NewViperConfigManager(logger *slog.Logger) (ports.ConfigManager, error) {
	v := viper.New()

	// Setup configuration directory (XDG compliant)
	configDir, err := getConfigDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Setup Viper
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)

	// Environment variable support
	v.SetEnvPrefix("LMMC")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults(v)

	return &ViperConfigManager{
		viper:     v,
		validator: validator.New(),
		configDir: configDir,
		logger:    logger,
	}, nil
}

// Load reads configuration from all sources
func (c *ViperConfigManager) Load() (*entities.Config, error) {
	// Try to read config file
	if err := c.viper.ReadInConfig(); err != nil {
		var configFileNotFoundErr viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundErr) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use defaults
		c.logger.Info("config file not found, using defaults",
			slog.String("expected_path", filepath.Join(c.configDir, "config.yaml")))
	} else {
		c.logger.Info("loaded config file",
			slog.String("path", c.viper.ConfigFileUsed()))
	}

	var config entities.Config
	if err := c.viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := c.validator.Struct(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Save persists the configuration to file
func (c *ViperConfigManager) Save(config *entities.Config) error {
	if err := c.validator.Struct(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Convert struct to map for Viper
	configMap := make(map[string]interface{})
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  &configMap,
		TagName: "mapstructure",
	})
	if err != nil {
		return fmt.Errorf("failed to create decoder: %w", err)
	}

	if err := decoder.Decode(config); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	// Update Viper with new values
	for key, value := range flattenMap("", configMap) {
		c.viper.Set(key, value)
	}

	// Write to file
	configPath := filepath.Join(c.configDir, "config.yaml")
	if err := c.viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	c.logger.Info("config saved",
		slog.String("path", configPath))
	return nil
}

// Set updates a specific configuration value
func (c *ViperConfigManager) Set(key, value string) error {
	// Validate key exists in config structure
	if !c.isValidConfigKey(key) {
		return fmt.Errorf("invalid configuration key: %s", key)
	}

	// Parse value based on expected type
	parsedValue := c.parseValue(key, value)
	c.viper.Set(key, parsedValue)

	// Save updated configuration
	config, err := c.Load()
	if err != nil {
		return err
	}

	return c.Save(config)
}

// Get retrieves a specific configuration value
func (c *ViperConfigManager) Get(key string) (interface{}, error) {
	if !c.viper.IsSet(key) {
		return nil, fmt.Errorf("configuration key not found: %s", key)
	}

	return c.viper.Get(key), nil
}

// GetConfigPath returns the path to the configuration file
func (c *ViperConfigManager) GetConfigPath() string {
	return filepath.Join(c.configDir, "config.yaml")
}

// Validate checks if the current configuration is valid
func (c *ViperConfigManager) Validate() error {
	config, err := c.Load()
	if err != nil {
		return err
	}

	return c.validator.Struct(config)
}

// Reset restores configuration to defaults
func (c *ViperConfigManager) Reset() error {
	// Clear all settings
	for _, key := range c.viper.AllKeys() {
		c.viper.Set(key, nil)
	}

	// Re-apply defaults
	setDefaults(c.viper)

	// Save default configuration
	defaultConfig := entities.DefaultConfig()
	return c.Save(defaultConfig)
}

// Helper functions

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.url", "http://localhost:9080")
	v.SetDefault("server.version", "v1")
	v.SetDefault("server.timeout", 30)

	// CLI defaults
	v.SetDefault("cli.default_repository", "")
	v.SetDefault("cli.output_format", "table")
	v.SetDefault("cli.auto_complete", true)
	v.SetDefault("cli.color_scheme", "auto")
	v.SetDefault("cli.page_size", 20)
	v.SetDefault("cli.editor", "")

	// Storage defaults
	v.SetDefault("storage.cache_enabled", true)
	v.SetDefault("storage.cache_ttl", 300)
	v.SetDefault("storage.backup_count", 3)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("logging.file", "")
}

func getConfigDirectory() (string, error) {
	// XDG Base Directory specification
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "lmmc"), nil
	}

	// Check for LMMC_CONFIG_DIR environment variable
	if lmmcConfigDir := os.Getenv("LMMC_CONFIG_DIR"); lmmcConfigDir != "" {
		return lmmcConfigDir, nil
	}

	// Fallback to ~/.lmmc for compatibility
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".lmmc"), nil
}

func flattenMap(prefix string, m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range m {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// Recursively flatten nested maps
			for k, val := range flattenMap(fullKey, v) {
				result[k] = val
			}
		default:
			result[fullKey] = value
		}
	}

	return result
}

func (c *ViperConfigManager) isValidConfigKey(key string) bool {
	validKeys := []string{
		"server.url",
		"server.version",
		"server.timeout",
		"cli.default_repository",
		"cli.output_format",
		"cli.auto_complete",
		"cli.color_scheme",
		"cli.page_size",
		"cli.editor",
		"storage.cache_enabled",
		"storage.cache_ttl",
		"storage.backup_count",
		"logging.level",
		"logging.format",
		"logging.file",
	}

	for _, validKey := range validKeys {
		if key == validKey {
			return true
		}
	}

	return false
}

func (c *ViperConfigManager) parseValue(key, value string) interface{} {
	// Handle boolean values
	boolKeys := []string{
		"cli.auto_complete",
		"storage.cache_enabled",
	}
	for _, boolKey := range boolKeys {
		if key == boolKey {
			return value == "true" || value == "1" || value == "yes"
		}
	}

	// Handle integer values
	intKeys := []string{
		"server.timeout",
		"cli.page_size",
		"storage.cache_ttl",
		"storage.backup_count",
	}
	for _, intKey := range intKeys {
		if key == intKey {
			var intVal int
			_, _ = fmt.Sscanf(value, "%d", &intVal)
			return intVal
		}
	}

	// Default to string
	return value
}
