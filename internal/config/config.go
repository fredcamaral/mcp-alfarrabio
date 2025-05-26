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
			Endpoint:       "http://localhost:9000",
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
	loadFromEnv(config)

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(config *Config) {
	loadServerConfig(config)
	loadChromaConfig(config)
	loadOpenAIConfig(config)
	loadDecayConfig(config)
	loadIntelligenceConfig(config)
	loadPerformanceConfig(config)
}

// loadServerConfig loads server configuration from environment
func loadServerConfig(config *Config) {
	// Server configuration
	if port := os.Getenv("MCP_MEMORY_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.Server.Port = p
		}
	}
	if host := os.Getenv("MCP_MEMORY_HOST"); host != "" {
		config.Server.Host = host
	}

	// Server timeouts
	if readTimeout := os.Getenv("MCP_MEMORY_READ_TIMEOUT_SECONDS"); readTimeout != "" {
		if rt, err := strconv.Atoi(readTimeout); err == nil {
			config.Server.ReadTimeout = rt
		}
	}
	if writeTimeout := os.Getenv("MCP_MEMORY_WRITE_TIMEOUT_SECONDS"); writeTimeout != "" {
		if wt, err := strconv.Atoi(writeTimeout); err == nil {
			config.Server.WriteTimeout = wt
		}
	}
}

// loadChromaConfig loads Chroma configuration from environment
func loadChromaConfig(config *Config) {
	// Chroma configuration - check both prefixed and non-prefixed env vars
	if endpoint := os.Getenv("MCP_MEMORY_CHROMA_ENDPOINT"); endpoint != "" {
		config.Chroma.Endpoint = endpoint
	} else if endpoint := os.Getenv("CHROMA_ENDPOINT"); endpoint != "" {
		config.Chroma.Endpoint = endpoint
	}
	if collection := os.Getenv("MCP_MEMORY_CHROMA_COLLECTION"); collection != "" {
		config.Chroma.Collection = collection
	} else if collection := os.Getenv("CHROMA_COLLECTION"); collection != "" {
		config.Chroma.Collection = collection
	}
	if healthCheck := os.Getenv("MCP_MEMORY_CHROMA_HEALTH_CHECK"); healthCheck != "" {
		if hc, err := strconv.ParseBool(healthCheck); err == nil {
			config.Chroma.HealthCheck = hc
		}
	}
	if retryAttempts := os.Getenv("MCP_MEMORY_CHROMA_RETRY_ATTEMPTS"); retryAttempts != "" {
		if ra, err := strconv.Atoi(retryAttempts); err == nil {
			config.Chroma.RetryAttempts = ra
		}
	}
	if timeoutSeconds := os.Getenv("MCP_MEMORY_CHROMA_TIMEOUT_SECONDS"); timeoutSeconds != "" {
		if ts, err := strconv.Atoi(timeoutSeconds); err == nil {
			config.Chroma.TimeoutSeconds = ts
		}
	}
	// Docker configuration
	if dockerEnabled := os.Getenv("MCP_MEMORY_CHROMA_DOCKER_ENABLED"); dockerEnabled != "" {
		if de, err := strconv.ParseBool(dockerEnabled); err == nil {
			config.Chroma.Docker.Enabled = de
		}
	}
	if containerName := os.Getenv("CHROMA_CONTAINER_NAME"); containerName != "" {
		config.Chroma.Docker.ContainerName = containerName
	}
	if volumePath := os.Getenv("CHROMA_VOLUME_PATH"); volumePath != "" {
		config.Chroma.Docker.VolumePath = volumePath
	}
	if image := os.Getenv("MCP_MEMORY_CHROMA_IMAGE"); image != "" {
		config.Chroma.Docker.Image = image
	}

	// OpenAI configuration
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.OpenAI.APIKey = apiKey
	}
	if model := os.Getenv("OPENAI_EMBEDDING_MODEL"); model != "" {
		config.OpenAI.EmbeddingModel = model
	}
	if maxTokens := os.Getenv("MCP_MEMORY_OPENAI_MAX_TOKENS"); maxTokens != "" {
		if mt, err := strconv.Atoi(maxTokens); err == nil {
			config.OpenAI.MaxTokens = mt
		}
	}
	if temperature := os.Getenv("MCP_MEMORY_OPENAI_TEMPERATURE"); temperature != "" {
		if temp, err := strconv.ParseFloat(temperature, 64); err == nil {
			config.OpenAI.Temperature = temp
		}
	}
	if requestTimeout := os.Getenv("MCP_MEMORY_OPENAI_REQUEST_TIMEOUT_SECONDS"); requestTimeout != "" {
		if rt, err := strconv.Atoi(requestTimeout); err == nil {
			config.OpenAI.RequestTimeout = rt
		}
	}
	if rateLimitRPM := os.Getenv("MCP_MEMORY_OPENAI_RATE_LIMIT_RPM"); rateLimitRPM != "" {
		if rl, err := strconv.Atoi(rateLimitRPM); err == nil {
			config.OpenAI.RateLimitRPM = rl
		}
	}

	// Storage configuration
	if provider := os.Getenv("MCP_MEMORY_STORAGE_PROVIDER"); provider != "" {
		config.Storage.Provider = provider
	}
	if retention := os.Getenv("RETENTION_DAYS"); retention != "" {
		if r, err := strconv.Atoi(retention); err == nil {
			config.Storage.RetentionDays = r
		}
	}
	if backupEnabled := os.Getenv("MCP_MEMORY_BACKUP_ENABLED"); backupEnabled != "" {
		if be, err := strconv.ParseBool(backupEnabled); err == nil {
			config.Storage.BackupEnabled = be
		}
	}
	if backupInterval := os.Getenv("MCP_MEMORY_BACKUP_INTERVAL_HOURS"); backupInterval != "" {
		if bi, err := strconv.Atoi(backupInterval); err == nil {
			config.Storage.BackupInterval = bi
		}
	}

	// Chunking configuration
	if strategy := os.Getenv("MCP_MEMORY_CHUNKING_STRATEGY"); strategy != "" {
		config.Chunking.Strategy = strategy
	}
	if minLength := os.Getenv("MCP_MEMORY_CHUNKING_MIN_LENGTH"); minLength != "" {
		if ml, err := strconv.Atoi(minLength); err == nil {
			config.Chunking.MinContentLength = ml
		}
	}
	if maxLength := os.Getenv("MCP_MEMORY_CHUNKING_MAX_LENGTH"); maxLength != "" {
		if ml, err := strconv.Atoi(maxLength); err == nil {
			config.Chunking.MaxContentLength = ml
		}
	}
	if todoTrigger := os.Getenv("MCP_MEMORY_CHUNKING_TODO_TRIGGER"); todoTrigger != "" {
		if tt, err := strconv.ParseBool(todoTrigger); err == nil {
			config.Chunking.TodoCompletionTrigger = tt
		}
	}
	if fileThreshold := os.Getenv("MCP_MEMORY_CHUNKING_FILE_THRESHOLD"); fileThreshold != "" {
		if ft, err := strconv.Atoi(fileThreshold); err == nil {
			config.Chunking.FileChangeThreshold = ft
		}
	}
	if timeThreshold := os.Getenv("MCP_MEMORY_CHUNKING_TIME_THRESHOLD_MINUTES"); timeThreshold != "" {
		if tt, err := strconv.Atoi(timeThreshold); err == nil {
			config.Chunking.TimeThresholdMinutes = tt
		}
	}
	if similarity := os.Getenv("MCP_MEMORY_CHUNKING_SIMILARITY_THRESHOLD"); similarity != "" {
		if st, err := strconv.ParseFloat(similarity, 64); err == nil {
			config.Chunking.SimilarityThreshold = st
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
	if maxSize := os.Getenv("MCP_MEMORY_LOG_MAX_SIZE_MB"); maxSize != "" {
		if ms, err := strconv.Atoi(maxSize); err == nil {
			config.Logging.MaxSize = ms
		}
	}
	if maxBackups := os.Getenv("MCP_MEMORY_LOG_MAX_BACKUPS"); maxBackups != "" {
		if mb, err := strconv.Atoi(maxBackups); err == nil {
			config.Logging.MaxBackups = mb
		}
	}
	if maxAge := os.Getenv("MCP_MEMORY_LOG_MAX_AGE_DAYS"); maxAge != "" {
		if ma, err := strconv.Atoi(maxAge); err == nil {
			config.Logging.MaxAge = ma
		}
	}
}

// loadOpenAIConfig loads OpenAI configuration from environment
func loadOpenAIConfig(config *Config) {
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.OpenAI.APIKey = apiKey
	}
	if model := os.Getenv("OPENAI_EMBEDDING_MODEL"); model != "" {
		config.OpenAI.EmbeddingModel = model
	}
}

// loadDecayConfig loads decay configuration from environment
func loadDecayConfig(config *Config) {
	// Add decay config loading if needed
}

// loadIntelligenceConfig loads intelligence configuration from environment
func loadIntelligenceConfig(config *Config) {
	// Add intelligence config loading if needed
}

// loadPerformanceConfig loads performance configuration from environment
func loadPerformanceConfig(config *Config) {
	// Add performance config loading if needed  
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
	if err := os.MkdirAll(absPath, 0750); err != nil {
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
