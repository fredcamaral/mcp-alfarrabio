// Package config provides configuration management for the MCP Memory Server,
// handling environment variables, YAML files, and runtime settings.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config represents the application configuration
type Config struct {
	Server    ServerConfig    `json:"server"`
	Database  DatabaseConfig  `json:"database"`
	Qdrant    QdrantConfig    `json:"qdrant"`
	OpenAI    OpenAIConfig    `json:"openai"`
	AI        AIConfig        `json:"ai"`
	Storage   StorageConfig   `json:"storage"`
	Chunking  ChunkingConfig  `json:"chunking"`
	Search    SearchConfig    `json:"search"`
	Logging   LoggingConfig   `json:"logging"`
	WebSocket WebSocketConfig `json:"websocket"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port         int    `json:"port"`
	Host         string `json:"host"`
	ReadTimeout  int    `json:"read_timeout_seconds"`
	WriteTimeout int    `json:"write_timeout_seconds"`
}

// DatabaseConfig represents PostgreSQL database configuration
type DatabaseConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	Name            string        `json:"name"`
	User            string        `json:"user"`
	Password        string        `json:"-"` // Never serialize password
	SSLMode         string        `json:"ssl_mode"`
	MaxOpenConns    int           `json:"max_open_conns"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`

	// Performance settings
	QueryTimeout       time.Duration `json:"query_timeout"`
	SlowQueryThreshold time.Duration `json:"slow_query_threshold"`
	EnableQueryLogging bool          `json:"enable_query_logging"`
	EnableMetrics      bool          `json:"enable_metrics"`

	// Migration settings
	MigrationTimeout  time.Duration `json:"migration_timeout"`
	EnableAutoMigrate bool          `json:"enable_auto_migrate"`
	MigrationsPath    string        `json:"migrations_path"`
}

// QdrantConfig represents Qdrant vector database configuration
type QdrantConfig struct {
	Host           string       `json:"host"`
	Port           int          `json:"port"`
	APIKey         string       `json:"-"` // Never serialize API key
	UseTLS         bool         `json:"use_tls"`
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

// AIConfig represents multi-model AI service configuration
type AIConfig struct {
	Claude     ClaudeClientConfig     `json:"claude"`
	Perplexity PerplexityClientConfig `json:"perplexity"`
	OpenAI     OpenAIClientConfig     `json:"openai"`
	Cache      CacheClientConfig      `json:"cache"`
}

// ClaudeClientConfig represents Claude API configuration
type ClaudeClientConfig struct {
	APIKey      string        `json:"-"` // Never serialize API key
	BaseURL     string        `json:"base_url"`
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
	Enabled     bool          `json:"enabled"`
}

// PerplexityClientConfig represents Perplexity API configuration
type PerplexityClientConfig struct {
	APIKey      string        `json:"-"` // Never serialize API key
	BaseURL     string        `json:"base_url"`
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
	Enabled     bool          `json:"enabled"`
}

// OpenAIClientConfig represents OpenAI GPT API configuration
type OpenAIClientConfig struct {
	APIKey      string        `json:"-"` // Never serialize API key
	BaseURL     string        `json:"base_url"`
	Model       string        `json:"model"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Timeout     time.Duration `json:"timeout"`
	Enabled     bool          `json:"enabled"`
}

// CacheClientConfig represents AI response caching configuration
type CacheClientConfig struct {
	Enabled         bool          `json:"enabled"`
	TTL             time.Duration `json:"ttl"`
	MaxSize         int           `json:"max_size"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
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

// SearchConfig represents search behavior configuration
type SearchConfig struct {
	DefaultMinRelevance      float64 `json:"default_min_relevance"`
	RelaxedMinRelevance      float64 `json:"relaxed_min_relevance"`
	BroadestMinRelevance     float64 `json:"broadest_min_relevance"`
	EnableProgressiveSearch  bool    `json:"enable_progressive_search"`
	EnableRepositoryFallback bool    `json:"enable_repository_fallback"`
	MaxRelatedRepos          int     `json:"max_related_repos"`
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

// WebSocketConfig represents WebSocket server configuration
type WebSocketConfig struct {
	MaxConnections    int      `json:"max_connections"`
	ReadBufferSize    int      `json:"read_buffer_size"`
	WriteBufferSize   int      `json:"write_buffer_size"`
	HandshakeTimeout  int      `json:"handshake_timeout"`
	PingInterval      int      `json:"ping_interval"`
	PongTimeout       int      `json:"pong_timeout"`
	WriteTimeout      int      `json:"write_timeout"`
	ReadTimeout       int      `json:"read_timeout"`
	EnableCompression bool     `json:"enable_compression"`
	MaxMessageSize    int      `json:"max_message_size"`
	EnableAuth        bool     `json:"enable_auth"`
	AllowedOrigins    []string `json:"allowed_origins"`
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
		Database: DatabaseConfig{
			Host:               "localhost",
			Port:               5432,
			Name:               "mcp_memory",
			User:               "postgres",
			Password:           "",
			SSLMode:            "disable",
			MaxOpenConns:       25,
			MaxIdleConns:       5,
			ConnMaxLifetime:    time.Hour,
			ConnMaxIdleTime:    time.Minute * 15,
			QueryTimeout:       time.Second * 30,
			SlowQueryThreshold: time.Millisecond * 100,
			EnableQueryLogging: false,
			EnableMetrics:      true,
			MigrationTimeout:   time.Minute * 10,
			EnableAutoMigrate:  false,
			MigrationsPath:     "./migrations",
		},
		Qdrant: QdrantConfig{
			Host:           "localhost",
			Port:           6334,
			UseTLS:         false,
			Collection:     "claude_memory",
			HealthCheck:    true,
			RetryAttempts:  3,
			TimeoutSeconds: 30,
			Docker: DockerConfig{
				Enabled:       true,
				ContainerName: "claude-memory-qdrant",
				VolumePath:    "./data/qdrant",
				Image:         "qdrant/qdrant:latest",
			},
		},
		OpenAI: OpenAIConfig{
			EmbeddingModel: "text-embedding-ada-002",
			MaxTokens:      8191,
			Temperature:    0.0,
			RequestTimeout: 60,
			RateLimitRPM:   60,
		},
		AI: AIConfig{
			Claude: ClaudeClientConfig{
				BaseURL:     "https://api.anthropic.com/v1/messages",
				Model:       "claude-3-5-sonnet-20241022",
				MaxTokens:   4000,
				Temperature: 0.7,
				Timeout:     30 * time.Second,
				Enabled:     false, // Disabled by default
			},
			Perplexity: PerplexityClientConfig{
				BaseURL:     "https://api.perplexity.ai/chat/completions",
				Model:       "llama-3.1-sonar-huge-128k-online",
				MaxTokens:   4000,
				Temperature: 0.7,
				Timeout:     30 * time.Second,
				Enabled:     false, // Disabled by default
			},
			OpenAI: OpenAIClientConfig{
				BaseURL:     "https://api.openai.com/v1/chat/completions",
				Model:       "gpt-4o",
				MaxTokens:   4000,
				Temperature: 0.7,
				Timeout:     30 * time.Second,
				Enabled:     false, // Disabled by default
			},
			Cache: CacheClientConfig{
				Enabled:         true,
				TTL:             30 * time.Minute,
				MaxSize:         1000,
				CleanupInterval: 5 * time.Minute,
			},
		},
		Storage: StorageConfig{
			Provider:       "qdrant",
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
		Search: SearchConfig{
			DefaultMinRelevance:      0.5,
			RelaxedMinRelevance:      0.3,
			BroadestMinRelevance:     0.2,
			EnableProgressiveSearch:  true,
			EnableRepositoryFallback: true,
			MaxRelatedRepos:          3,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			MaxSize:    10,
			MaxBackups: 3,
			MaxAge:     30,
		},
		WebSocket: WebSocketConfig{
			MaxConnections:    100,
			ReadBufferSize:    1024,
			WriteBufferSize:   1024,
			HandshakeTimeout:  10,
			PingInterval:      30,
			PongTimeout:       60,
			WriteTimeout:      10,
			ReadTimeout:       10,
			EnableCompression: true,
			MaxMessageSize:    65536,
			EnableAuth:        false,
			AllowedOrigins:    []string{"*"},
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
	loadDatabaseConfig(config)
	loadQdrantConfig(config)
	loadStorageAndOtherConfig(config)
	loadOpenAIConfig(config)
	loadAIConfig(config)
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

// loadDatabaseConfig loads database configuration from environment
func loadDatabaseConfig(config *Config) {
	// Database connection settings
	config.Database.Host = getStringEnvWithDefault("DB_HOST", config.Database.Host)
	config.Database.Port = getIntEnvWithDefault("DB_PORT", config.Database.Port)
	config.Database.Name = getStringEnvWithDefault("DB_NAME", config.Database.Name)
	config.Database.User = getStringEnvWithDefault("DB_USER", config.Database.User)
	config.Database.Password = getStringEnvWithDefault("DB_PASSWORD", config.Database.Password)
	config.Database.SSLMode = getStringEnvWithDefault("DB_SSLMODE", config.Database.SSLMode)

	// Connection pool settings
	config.Database.MaxOpenConns = getIntEnvWithDefault("DB_MAX_OPEN_CONNS", config.Database.MaxOpenConns)
	config.Database.MaxIdleConns = getIntEnvWithDefault("DB_MAX_IDLE_CONNS", config.Database.MaxIdleConns)

	// Connection timeouts
	if connMaxLifetime := os.Getenv("DB_CONN_MAX_LIFETIME"); connMaxLifetime != "" {
		if duration, err := time.ParseDuration(connMaxLifetime); err == nil {
			config.Database.ConnMaxLifetime = duration
		}
	}
	if connMaxIdleTime := os.Getenv("DB_CONN_MAX_IDLE_TIME"); connMaxIdleTime != "" {
		if duration, err := time.ParseDuration(connMaxIdleTime); err == nil {
			config.Database.ConnMaxIdleTime = duration
		}
	}

	// Performance settings
	if queryTimeout := os.Getenv("DB_QUERY_TIMEOUT"); queryTimeout != "" {
		if duration, err := time.ParseDuration(queryTimeout); err == nil {
			config.Database.QueryTimeout = duration
		}
	}
	if slowQueryThreshold := os.Getenv("DB_SLOW_QUERY_THRESHOLD"); slowQueryThreshold != "" {
		if duration, err := time.ParseDuration(slowQueryThreshold); err == nil {
			config.Database.SlowQueryThreshold = duration
		}
	}

	config.Database.EnableQueryLogging = getBoolEnvWithDefault("DB_ENABLE_QUERY_LOGGING", config.Database.EnableQueryLogging)
	config.Database.EnableMetrics = getBoolEnvWithDefault("DB_ENABLE_METRICS", config.Database.EnableMetrics)

	// Migration settings
	if migrationTimeout := os.Getenv("DB_MIGRATION_TIMEOUT"); migrationTimeout != "" {
		if duration, err := time.ParseDuration(migrationTimeout); err == nil {
			config.Database.MigrationTimeout = duration
		}
	}
	config.Database.EnableAutoMigrate = getBoolEnvWithDefault("DB_ENABLE_AUTO_MIGRATE", config.Database.EnableAutoMigrate)
	config.Database.MigrationsPath = getStringEnvWithDefault("DB_MIGRATIONS_PATH", config.Database.MigrationsPath)
}

// getStringEnvWithDefault gets string environment variable with default value
func getStringEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadQdrantConfig loads Qdrant configuration from environment
func loadQdrantConfig(config *Config) {
	loadQdrantBasicConfig(config)
	loadQdrantDockerConfig(config)
}

// loadQdrantBasicConfig loads basic Qdrant settings
func loadQdrantBasicConfig(config *Config) {
	loadQdrantConnectionSettings(config)
	loadQdrantServiceSettings(config)
}

// loadQdrantConnectionSettings loads host, port, API key, and TLS settings
func loadQdrantConnectionSettings(config *Config) {
	config.Qdrant.Host = getStringEnvWithFallback("MCP_MEMORY_QDRANT_HOST", "QDRANT_HOST", config.Qdrant.Host)
	config.Qdrant.Port = getIntEnvWithFallback("MCP_MEMORY_QDRANT_PORT", "QDRANT_PORT", config.Qdrant.Port)
	config.Qdrant.APIKey = getStringEnvWithFallback("MCP_MEMORY_QDRANT_API_KEY", "QDRANT_API_KEY", config.Qdrant.APIKey)
	config.Qdrant.UseTLS = getBoolEnvWithFallback("MCP_MEMORY_QDRANT_USE_TLS", "QDRANT_USE_TLS", config.Qdrant.UseTLS)
	config.Qdrant.Collection = getStringEnvWithFallback("MCP_MEMORY_QDRANT_COLLECTION", "QDRANT_COLLECTION", config.Qdrant.Collection)
}

// loadQdrantServiceSettings loads service-related settings like health check, retry, and timeout
func loadQdrantServiceSettings(config *Config) {
	config.Qdrant.HealthCheck = getBoolEnvWithDefault("MCP_MEMORY_QDRANT_HEALTH_CHECK", config.Qdrant.HealthCheck)
	config.Qdrant.RetryAttempts = getIntEnvWithDefault("MCP_MEMORY_QDRANT_RETRY_ATTEMPTS", config.Qdrant.RetryAttempts)
	config.Qdrant.TimeoutSeconds = getIntEnvWithDefault("MCP_MEMORY_QDRANT_TIMEOUT_SECONDS", config.Qdrant.TimeoutSeconds)
}

// getStringEnvWithFallback gets string environment variable with fallback to alternate key
func getStringEnvWithFallback(primaryKey, fallbackKey, defaultValue string) string {
	if value := os.Getenv(primaryKey); value != "" {
		return value
	}
	if value := os.Getenv(fallbackKey); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnvWithFallback gets integer environment variable with fallback to alternate key
func getIntEnvWithFallback(primaryKey, fallbackKey string, defaultValue int) int {
	if value := os.Getenv(primaryKey); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	if value := os.Getenv(fallbackKey); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getBoolEnvWithFallback gets boolean environment variable with fallback to alternate key
func getBoolEnvWithFallback(primaryKey, fallbackKey string, defaultValue bool) bool {
	if value := os.Getenv(primaryKey); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	if value := os.Getenv(fallbackKey); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getBoolEnvWithDefault gets boolean environment variable with default value
func getBoolEnvWithDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getIntEnvWithDefault gets integer environment variable with default value
func getIntEnvWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// loadQdrantDockerConfig loads Docker-related Qdrant settings
func loadQdrantDockerConfig(config *Config) {
	if dockerEnabled := os.Getenv("MCP_MEMORY_QDRANT_DOCKER_ENABLED"); dockerEnabled != "" {
		if de, err := strconv.ParseBool(dockerEnabled); err == nil {
			config.Qdrant.Docker.Enabled = de
		}
	}
	if containerName := os.Getenv("QDRANT_CONTAINER_NAME"); containerName != "" {
		config.Qdrant.Docker.ContainerName = containerName
	}
	if volumePath := os.Getenv("QDRANT_VOLUME_PATH"); volumePath != "" {
		config.Qdrant.Docker.VolumePath = volumePath
	}
	if image := os.Getenv("MCP_MEMORY_QDRANT_IMAGE"); image != "" {
		config.Qdrant.Docker.Image = image
	}
}

func loadStorageAndOtherConfig(config *Config) {
	loadStorageConfig(config)
	loadChunkingConfig(config)
	loadLoggingConfig(config)
	loadWebSocketConfig(config)
}

// loadStorageConfig loads storage configuration from environment
func loadStorageConfig(config *Config) {
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
}

// loadChunkingConfig loads chunking configuration from environment
func loadChunkingConfig(config *Config) {
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
}

// loadLoggingConfig loads logging configuration from environment
func loadLoggingConfig(config *Config) {
	if level := os.Getenv("MCP_MEMORY_LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if format := os.Getenv("MCP_MEMORY_LOG_FORMAT"); format != "" {
		config.Logging.Format = format
	}
	if file := os.Getenv("MCP_MEMORY_LOG_FILE"); file != "" {
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
}

// loadAIConfig loads AI provider configuration from environment
func loadAIConfig(config *Config) {
	// Claude configuration
	if claudeAPIKey := os.Getenv("CLAUDE_API_KEY"); claudeAPIKey != "" {
		config.AI.Claude.APIKey = claudeAPIKey
		config.AI.Claude.Enabled = true // Auto-enable if API key is provided
	}
	if claudeEnabled := os.Getenv("CLAUDE_ENABLED"); claudeEnabled != "" {
		if enabled, err := strconv.ParseBool(claudeEnabled); err == nil {
			config.AI.Claude.Enabled = enabled
		}
	}
	if claudeModel := os.Getenv("CLAUDE_MODEL"); claudeModel != "" {
		config.AI.Claude.Model = claudeModel
	}

	// OpenAI AI configuration (separate from embeddings)
	if openaiAPIKey := os.Getenv("OPENAI_API_KEY"); openaiAPIKey != "" {
		config.AI.OpenAI.APIKey = openaiAPIKey
		config.AI.OpenAI.Enabled = true // Auto-enable if API key is provided
	}
	if openaiEnabled := os.Getenv("OPENAI_ENABLED"); openaiEnabled != "" {
		if enabled, err := strconv.ParseBool(openaiEnabled); err == nil {
			config.AI.OpenAI.Enabled = enabled
		}
	}
	if openaiModel := os.Getenv("OPENAI_MODEL"); openaiModel != "" {
		config.AI.OpenAI.Model = openaiModel
	}

	// Perplexity configuration
	if perplexityAPIKey := os.Getenv("PERPLEXITY_API_KEY"); perplexityAPIKey != "" {
		config.AI.Perplexity.APIKey = perplexityAPIKey
		config.AI.Perplexity.Enabled = true // Auto-enable if API key is provided
	}
	if perplexityEnabled := os.Getenv("PERPLEXITY_ENABLED"); perplexityEnabled != "" {
		if enabled, err := strconv.ParseBool(perplexityEnabled); err == nil {
			config.AI.Perplexity.Enabled = enabled
		}
	}
	if perplexityModel := os.Getenv("PERPLEXITY_MODEL"); perplexityModel != "" {
		config.AI.Perplexity.Model = perplexityModel
	}
}

// loadDecayConfig loads decay configuration from environment
func loadDecayConfig(_ *Config) {
	// Add decay config loading if needed
}

// loadIntelligenceConfig loads intelligence configuration from environment
func loadIntelligenceConfig(_ *Config) {
	// Add intelligence config loading if needed
}

// loadPerformanceConfig loads performance configuration from environment
func loadPerformanceConfig(config *Config) {
	// Add performance config loading if needed
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if err := c.validateServerConfig(); err != nil {
		return err
	}

	if err := c.validateDatabaseConfig(); err != nil {
		return err
	}

	if err := c.validateQdrantConfig(); err != nil {
		return err
	}

	if err := c.validateOpenAIConfig(); err != nil {
		return err
	}

	if err := c.validateStorageConfig(); err != nil {
		return err
	}

	if err := c.validateChunkingConfig(); err != nil {
		return err
	}

	return nil
}

// validateServerConfig validates server configuration settings
func (c *Config) validateServerConfig() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Server.Host == "" {
		return errors.New("server host cannot be empty")
	}
	return nil
}

// validateDatabaseConfig validates database configuration settings
func (c *Config) validateDatabaseConfig() error {
	if c.Database.Host == "" {
		return errors.New("database host cannot be empty")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}
	if c.Database.Name == "" {
		return errors.New("database name cannot be empty")
	}
	if c.Database.User == "" {
		return errors.New("database user cannot be empty")
	}
	if c.Database.MaxOpenConns <= 0 {
		return errors.New("max open connections must be positive")
	}
	if c.Database.MaxIdleConns < 0 {
		return errors.New("max idle connections cannot be negative")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return errors.New("max idle connections cannot exceed max open connections")
	}
	return nil
}

// validateQdrantConfig validates Qdrant vector database configuration
func (c *Config) validateQdrantConfig() error {
	if c.Qdrant.Host == "" {
		return errors.New("qdrant host cannot be empty")
	}
	if c.Qdrant.Port <= 0 {
		return errors.New("qdrant port must be greater than 0")
	}
	if c.Qdrant.Collection == "" {
		return errors.New("qdrant collection cannot be empty")
	}
	if c.Qdrant.Docker.Enabled && c.Qdrant.Docker.ContainerName == "" {
		return errors.New("docker container name cannot be empty when docker is enabled")
	}
	return nil
}

// validateOpenAIConfig validates OpenAI API configuration
func (c *Config) validateOpenAIConfig() error {
	if c.OpenAI.APIKey == "" {
		return errors.New("OpenAI API key is required")
	}
	if c.OpenAI.EmbeddingModel == "" {
		return errors.New("OpenAI embedding model cannot be empty")
	}
	return nil
}

// validateStorageConfig validates storage configuration settings
func (c *Config) validateStorageConfig() error {
	if c.Storage.RetentionDays <= 0 {
		return errors.New("retention days must be positive")
	}
	return nil
}

// validateChunkingConfig validates chunking algorithm configuration
func (c *Config) validateChunkingConfig() error {
	if c.Chunking.MinContentLength <= 0 {
		return errors.New("min content length must be positive")
	}
	if c.Chunking.MaxContentLength <= c.Chunking.MinContentLength {
		return errors.New("max content length must be greater than min content length")
	}
	if c.Chunking.SimilarityThreshold < 0 || c.Chunking.SimilarityThreshold > 1 {
		return errors.New("similarity threshold must be between 0 and 1")
	}
	return nil
}

// GetDataDir returns the data directory path, creating it if necessary
func (c *Config) GetDataDir() (string, error) {
	dataDir := c.Qdrant.Docker.VolumePath
	if dataDir == "" {
		dataDir = "./data"
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(dataDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for data directory: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0o750); err != nil {
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

// loadWebSocketConfig loads WebSocket configuration from environment
func loadWebSocketConfig(config *Config) {
	loadWebSocketIntConfig(config)
	loadWebSocketBoolConfig(config)
	loadWebSocketOrigins(config)
}

// loadWebSocketIntConfig loads integer WebSocket configuration values
func loadWebSocketIntConfig(config *Config) {
	setIntFromEnv("WS_MAX_CONNECTIONS", &config.WebSocket.MaxConnections)
	setIntFromEnv("WS_READ_BUFFER_SIZE", &config.WebSocket.ReadBufferSize)
	setIntFromEnv("WS_WRITE_BUFFER_SIZE", &config.WebSocket.WriteBufferSize)
	setIntFromEnv("WS_HANDSHAKE_TIMEOUT", &config.WebSocket.HandshakeTimeout)
	setIntFromEnv("WS_PING_INTERVAL", &config.WebSocket.PingInterval)
	setIntFromEnv("WS_PONG_TIMEOUT", &config.WebSocket.PongTimeout)
	setIntFromEnv("WS_WRITE_TIMEOUT", &config.WebSocket.WriteTimeout)
	setIntFromEnv("WS_READ_TIMEOUT", &config.WebSocket.ReadTimeout)
	setIntFromEnv("WS_MAX_MESSAGE_SIZE", &config.WebSocket.MaxMessageSize)
}

// loadWebSocketBoolConfig loads boolean WebSocket configuration values
func loadWebSocketBoolConfig(config *Config) {
	setBoolFromEnv("WS_ENABLE_COMPRESSION", &config.WebSocket.EnableCompression)
	setBoolFromEnv("WS_ENABLE_AUTH", &config.WebSocket.EnableAuth)
}

// loadWebSocketOrigins loads allowed origins from environment
func loadWebSocketOrigins(config *Config) {
	allowedOrigins := os.Getenv("WS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		return
	}

	origins := strings.Split(allowedOrigins, ",")
	cleaned := make([]string, 0, len(origins))
	for _, origin := range origins {
		if trimmed := strings.TrimSpace(origin); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) > 0 {
		config.WebSocket.AllowedOrigins = cleaned
	}
}

// setIntFromEnv sets an integer config value from environment variable
func setIntFromEnv(envKey string, target *int) {
	if value := os.Getenv(envKey); value != "" {
		if n, err := strconv.Atoi(value); err == nil {
			*target = n
		}
	}
}

// setBoolFromEnv sets a boolean config value from environment variable
func setBoolFromEnv(envKey string, target *bool) {
	if value := os.Getenv(envKey); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			*target = b
		}
	}
}
