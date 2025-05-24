package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Chroma   ChromaConfig   `json:"chroma"`
	OpenAI   OpenAIConfig   `json:"openai"`
	Storage  StorageConfig  `json:"storage"`
	Chunking ChunkingConfig `json:"chunking"`
	Logging  LoggingConfig  `json:"logging"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port         int    `json:"port"`
	Host         string `json:"host"`
	ReadTimeout  int    `json:"read_timeout_seconds"`
	WriteTimeout int    `json:"write_timeout_seconds"`
}

// ChromaConfig represents Chroma vector database configuration
type ChromaConfig struct {
	Endpoint       string       `json:"endpoint"`
	Collection     string       `json:"collection"`
	Docker         DockerConfig `json:"docker"`
	HealthCheck    bool         `json:"health_check"`
	RetryAttempts  int          `json:"retry_attempts"`
	TimeoutSeconds int          `json:"timeout_seconds"`
}

// DockerConfig represents Docker-specific configuration
type DockerConfig struct {
	Enabled       bool   `json:"enabled"`
	ContainerName string `json:"container_name"`
	VolumePath    string `json:"volume_path"`
	Image         string `json:"image"`
}

// OpenAIConfig represents OpenAI API configuration
type OpenAIConfig struct {
	APIKey         string  `json:"-"` // Never serialize API key
	EmbeddingModel string  `json:"embedding_model"`
	MaxTokens      int     `json:"max_tokens"`
	Temperature    float64 `json:"temperature"`
	RequestTimeout int     `json:"request_timeout_seconds"`
	RateLimitRPM   int     `json:"rate_limit_rpm"`
}

// StorageConfig represents storage configuration
type StorageConfig struct {
	Provider       string                `json:"provider"`
	RetentionDays  int                   `json:"retention_days"`
	BackupEnabled  bool                  `json:"backup_enabled"`
	BackupInterval int                   `json:"backup_interval_hours"`
	Repositories   map[string]RepoConfig `json:"repositories"`
}

// RepoConfig represents repository-specific configuration
type RepoConfig struct {
	Enabled         bool     `json:"enabled"`
	Sensitivity     string   `json:"sensitivity"`
	ExcludePatterns []string `json:"exclude_patterns"`
	Tags            []string `json:"tags"`
}

// ChunkingConfig represents chunking algorithm configuration
type ChunkingConfig struct {
	Strategy              string  `json:"strategy"`
	MinContentLength      int     `json:"min_content_length"`
	MaxContentLength      int     `json:"max_content_length"`
	TodoCompletionTrigger bool    `json:"todo_completion_trigger"`
	FileChangeThreshold   int     `json:"file_change_threshold"`
	TimeThresholdMinutes  int     `json:"time_threshold_minutes"`
	SimilarityThreshold   float64 `json:"similarity_threshold"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	File       string `json:"file,omitempty"`
	MaxSize    int    `json:"max_size_mb"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age_days"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8080,
			Host:         "localhost",
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Chroma: ChromaConfig{
			Endpoint:       "http://localhost:8000",
			Collection:     "claude_memory",
			HealthCheck:    true,
			RetryAttempts:  3,
			TimeoutSeconds: 30,
			Docker: DockerConfig{
				Enabled:       true,
				ContainerName: "claude-memory-chroma",
				VolumePath:    "./data/chroma",
				Image:         "ghcr.io/chroma-core/chroma:latest",
			},
		},
		OpenAI: OpenAIConfig{
			EmbeddingModel: "text-embedding-ada-002",
			MaxTokens:      8191,
			Temperature:    0.0,
			RequestTimeout: 60,
			RateLimitRPM:   60,
		},
		Storage: StorageConfig{
			Provider:       "chroma",
			RetentionDays:  90,
			BackupEnabled:  false,
			BackupInterval: 24,
			Repositories:   make(map[string]RepoConfig),
		},
		Chunking: ChunkingConfig{
			Strategy:              "smart",
			MinContentLength:      50,
			MaxContentLength:      10000,
			TodoCompletionTrigger: true,
			FileChangeThreshold:   3,
			TimeThresholdMinutes:  10,
			SimilarityThreshold:   0.8,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     30,
		},
	}
}

// LoadConfig loads configuration from environment variables and defaults
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Don't fail if .env doesn't exist
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	config := DefaultConfig()

	// Override with environment variables
	if err := loadFromEnv(config); err != nil {
		return nil, fmt.Errorf("error loading config from environment: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(config *Config) error {
	// Server configuration
	if port := os.Getenv("MCP_MEMORY_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	if host := os.Getenv("MCP_MEMORY_HOST"); host != "" {
		config.Server.Host = host
	}

	// Chroma configuration
	if endpoint := os.Getenv("CHROMA_ENDPOINT"); endpoint != "" {
		config.Chroma.Endpoint = endpoint
	}
	if collection := os.Getenv("CHROMA_COLLECTION"); collection != "" {
		config.Chroma.Collection = collection
	}
	if containerName := os.Getenv("CHROMA_CONTAINER_NAME"); containerName != "" {
		config.Chroma.Docker.ContainerName = containerName
	}
	if volumePath := os.Getenv("CHROMA_VOLUME_PATH"); volumePath != "" {
		config.Chroma.Docker.VolumePath = volumePath
	}

	// OpenAI configuration
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.OpenAI.APIKey = apiKey
	}
	if model := os.Getenv("OPENAI_EMBEDDING_MODEL"); model != "" {
		config.OpenAI.EmbeddingModel = model
	}

	// Storage configuration
	if retention := os.Getenv("RETENTION_DAYS"); retention != "" {
		if r, err := strconv.Atoi(retention); err == nil {
			config.Storage.RetentionDays = r
		}
	}

	// Logging configuration
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Logging.Format = format
	}
	if file := os.Getenv("LOG_FILE"); file != "" {
		config.Logging.File = file
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	// Validate Chroma config
	if c.Chroma.Endpoint == "" {
		return fmt.Errorf("chroma endpoint cannot be empty")
	}
	if c.Chroma.Collection == "" {
		return fmt.Errorf("chroma collection cannot be empty")
	}
	if c.Chroma.Docker.Enabled && c.Chroma.Docker.ContainerName == "" {
		return fmt.Errorf("docker container name cannot be empty when docker is enabled")
	}

	// Validate OpenAI config
	if c.OpenAI.APIKey == "" {
		return fmt.Errorf("OpenAI API key is required")
	}
	if c.OpenAI.EmbeddingModel == "" {
		return fmt.Errorf("OpenAI embedding model cannot be empty")
	}

	// Validate storage config
	if c.Storage.RetentionDays <= 0 {
		return fmt.Errorf("retention days must be positive")
	}

	// Validate chunking config
	if c.Chunking.MinContentLength <= 0 {
		return fmt.Errorf("min content length must be positive")
	}
	if c.Chunking.MaxContentLength <= c.Chunking.MinContentLength {
		return fmt.Errorf("max content length must be greater than min content length")
	}
	if c.Chunking.SimilarityThreshold < 0 || c.Chunking.SimilarityThreshold > 1 {
		return fmt.Errorf("similarity threshold must be between 0 and 1")
	}

	return nil
}

// GetDataDir returns the data directory path, creating it if necessary
func (c *Config) GetDataDir() (string, error) {
	dataDir := c.Chroma.Docker.VolumePath
	if dataDir == "" {
		dataDir = "./data"
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(dataDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for data directory: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return absPath, nil
}

// GetRepoConfig returns configuration for a specific repository
func (c *Config) GetRepoConfig(repository string) RepoConfig {
	if repoConfig, exists := c.Storage.Repositories[repository]; exists {
		return repoConfig
	}

	// Return default repo config
	return RepoConfig{
		Enabled:         true,
		Sensitivity:     "normal",
		ExcludePatterns: []string{"*.env", "*.key", "*.pem", "*.p12"},
		Tags:            []string{},
	}
}

// SetRepoConfig sets configuration for a specific repository
func (c *Config) SetRepoConfig(repository string, config RepoConfig) {
	if c.Storage.Repositories == nil {
		c.Storage.Repositories = make(map[string]RepoConfig)
	}
	c.Storage.Repositories[repository] = config
}

// IsRepositoryEnabled checks if a repository is enabled for memory storage
func (c *Config) IsRepositoryEnabled(repository string) bool {
	repoConfig := c.GetRepoConfig(repository)
	return repoConfig.Enabled
}
