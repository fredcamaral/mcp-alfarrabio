// Package ports defines interfaces for external adapters
// for the lerian-mcp-memory CLI application.
package ports

import "lerian-mcp-memory-cli/internal/domain/entities"

// ConfigManager defines the interface for configuration management
type ConfigManager interface {
	// Load reads configuration from all sources (file, env, defaults)
	Load() (*entities.Config, error)

	// Save persists the configuration to file
	Save(config *entities.Config) error

	// Set updates a specific configuration value
	Set(key, value string) error

	// Get retrieves a specific configuration value
	Get(key string) (interface{}, error)

	// GetConfigPath returns the path to the configuration file
	GetConfigPath() string

	// Validate checks if the current configuration is valid
	Validate() error

	// Reset restores configuration to defaults
	Reset() error
}
