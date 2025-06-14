// Package entities defines core data structures and business entities
// for the lerian-mcp-memory CLI application.
package entities

// Config represents the complete CLI configuration
type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	CLI     CLIConfig     `mapstructure:"cli"`
	Storage StorageConfig `mapstructure:"storage"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	URL     string `mapstructure:"url" validate:"omitempty,url"`
	Version string `mapstructure:"version"`
	Timeout int    `mapstructure:"timeout" validate:"min=1,max=300"`
}

// CLIConfig holds CLI behavior configuration
type CLIConfig struct {
	DefaultRepository string `mapstructure:"default_repository"`
	OutputFormat      string `mapstructure:"output_format" validate:"oneof=table json plain"`
	AutoComplete      bool   `mapstructure:"auto_complete"`
	ColorScheme       string `mapstructure:"color_scheme" validate:"oneof=auto always never"`
	PageSize          int    `mapstructure:"page_size" validate:"min=1,max=100"`
	Editor            string `mapstructure:"editor"`
}

// StorageConfig holds storage-related configuration
type StorageConfig struct {
	CacheEnabled bool `mapstructure:"cache_enabled"`
	CacheTTL     int  `mapstructure:"cache_ttl" validate:"min=1"`
	BackupCount  int  `mapstructure:"backup_count" validate:"min=0,max=10"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level" validate:"oneof=debug info warn error"`
	Format string `mapstructure:"format" validate:"oneof=json text"`
	File   string `mapstructure:"file"`
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			URL:     "http://localhost:9080",
			Version: "v1",
			Timeout: 30,
		},
		CLI: CLIConfig{
			DefaultRepository: "",
			OutputFormat:      "table",
			AutoComplete:      true,
			ColorScheme:       "auto",
			PageSize:          20,
			Editor:            "",
		},
		Storage: StorageConfig{
			CacheEnabled: true,
			CacheTTL:     300, // 5 minutes
			BackupCount:  3,
		},
		Logging: LoggingConfig{
			Level:  "warn",
			Format: "text",
			File:   "",
		},
	}
}
