// Package di provides dependency injection container for the application
package di

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"lerian-mcp-memory/internal/analytics"
	"lerian-mcp-memory/internal/api/handlers"
	"lerian-mcp-memory/internal/audit"
	"lerian-mcp-memory/internal/chains"
	"lerian-mcp-memory/internal/chunking"
	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/embeddings"
	"lerian-mcp-memory/internal/intelligence"
	"lerian-mcp-memory/internal/logging"
	"lerian-mcp-memory/internal/persistence"
	"lerian-mcp-memory/internal/relationships"
	"lerian-mcp-memory/internal/storage"
	"lerian-mcp-memory/internal/sync"
	"lerian-mcp-memory/internal/threading"
	"lerian-mcp-memory/internal/workflow"
	sharedai "lerian-mcp-memory/pkg/ai"
	"log/slog"
	"os"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

const envValueTrue = "true"

// Container holds all application dependencies
type Container struct {
	Config              *config.Config
	DB                  *sql.DB
	VectorStore         storage.VectorStore
	EmbeddingService    embeddings.EmbeddingService
	ChunkingService     *chunking.Service
	ContextSuggester    *workflow.ContextSuggester
	BackupManager       *persistence.BackupManager
	LearningEngine      *intelligence.LearningEngine
	PatternAnalyzer     *workflow.PatternAnalyzer
	TodoTracker         *workflow.TodoTracker
	FlowDetector        *workflow.FlowDetector
	PatternEngine       *intelligence.PatternEngine
	PatternStorage      intelligence.PatternStorage
	GraphBuilder        *intelligence.GraphBuilder
	MultiRepoEngine     *intelligence.MultiRepoEngine
	ChainBuilder        *chains.ChainBuilder
	ChainStore          chains.ChainStore
	RelationshipManager *relationships.Manager
	ThreadManager       *threading.ThreadManager
	ThreadStore         threading.ThreadStore
	MemoryAnalytics     *analytics.MemoryAnalytics
	AuditLogger         *audit.Logger
	AIService           sharedai.AIService
	Logger              logging.Logger
	WebSocketHandler    *handlers.WebSocketHandler
	RealtimeSyncCoord   *sync.RealtimeSyncCoordinator
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *config.Config) (*Container, error) {
	container := &Container{
		Config: cfg,
	}

	// Initialize logger
	logLevel := logging.ParseLogLevel(os.Getenv("MCP_MEMORY_LOG_LEVEL"))
	container.Logger = logging.NewLogger(logLevel)

	// Initialize database if PostgreSQL is configured
	if err := container.initializeDatabase(); err != nil {
		// Log error but continue - database is optional for now
		container.Logger.Warn("Failed to initialize database", "error", err)
	}

	// Initialize AI service
	if err := container.initializeAIService(cfg); err != nil {
		// Log error but continue - AI service may fallback to mock
		container.Logger.Warn("Failed to initialize AI service", "error", err)
	}

	// Initialize in dependency order
	container.initializeStorage()

	container.initializeServices()
	container.initializeIntelligence()
	container.initializeWorkflow()
	container.initializeRealTimeSync()

	return container, nil
}

// initializeDatabase sets up PostgreSQL connection if configured
func (c *Container) initializeDatabase() error {
	// Get database URL from environment or construct from components
	dbURL := buildDatabaseURL()

	// Open database connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		_ = db.Close() // Ignore close error since we're already in error state
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	c.DB = db
	c.Logger.Info("Database connection established")
	return nil
}

// initializeAIService sets up the AI service with all configured providers
func (c *Container) initializeAIService(cfg *config.Config) error {
	// Determine primary AI provider based on configuration
	var provider, apiKey, baseURL, model string
	var timeout time.Duration

	// Check which provider is enabled and configured
	if cfg.AI.Claude.Enabled && cfg.AI.Claude.APIKey != "" {
		provider = "claude"
		apiKey = cfg.AI.Claude.APIKey
		baseURL = cfg.AI.Claude.BaseURL
		model = cfg.AI.Claude.Model
		timeout = cfg.AI.Claude.Timeout
	} else if cfg.AI.OpenAI.Enabled && cfg.AI.OpenAI.APIKey != "" {
		provider = "openai"
		apiKey = cfg.AI.OpenAI.APIKey
		baseURL = cfg.AI.OpenAI.BaseURL
		model = cfg.AI.OpenAI.Model
		timeout = cfg.AI.OpenAI.Timeout
	} else if cfg.AI.Perplexity.Enabled && cfg.AI.Perplexity.APIKey != "" {
		provider = "perplexity"
		apiKey = cfg.AI.Perplexity.APIKey
		baseURL = cfg.AI.Perplexity.BaseURL
		model = cfg.AI.Perplexity.Model
		timeout = cfg.AI.Perplexity.Timeout
	} else {
		// No real AI provider is configured - return a configuration error
		return errors.New("no AI provider configured: please configure at least one AI provider (Claude, OpenAI, or Perplexity) in your environment variables or config file. Set API keys via CLAUDE_API_KEY, OPENAI_API_KEY, or PERPLEXITY_API_KEY and enable them via CLAUDE_ENABLED=true, OPENAI_ENABLED=true, or PERPLEXITY_ENABLED=true")
	}

	// Create shared AI service configuration
	aiConfig := &sharedai.Config{
		Provider:   provider,
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Model:      model,
		Timeout:    timeout,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}

	// Create logger adapter for shared AI service (convert server logger to slog)
	aiLogger := c.createSlogAdapter()

	// Create AI service with configuration
	aiService, err := sharedai.NewService(aiConfig, aiLogger)
	if err != nil {
		return fmt.Errorf("failed to create AI service: %w", err)
	}

	c.AIService = aiService
	c.Logger.Info("AI service initialized", "provider", aiConfig.Provider, "model", aiConfig.Model)
	return nil
}

// initializeStorage sets up storage layer
func (c *Container) initializeStorage() {
	var baseStore storage.VectorStore

	// Initialize vector store based on provider
	switch c.Config.Storage.Provider {
	case "qdrant":
		baseStore = storage.NewQdrantStore(&c.Config.Qdrant)
	default:
		// Default to Qdrant for new installations
		baseStore = storage.NewQdrantStore(&c.Config.Qdrant)
	}

	// Wrap with retry logic
	retryStore := storage.NewRetryableVectorStore(baseStore, nil)

	// Wrap with circuit breaker if enabled (check both environment variable formats)
	useCircuitBreaker := os.Getenv("MCP_MEMORY_CIRCUIT_BREAKER_ENABLED") == envValueTrue ||
		os.Getenv("USE_CIRCUIT_BREAKER") == envValueTrue

	if useCircuitBreaker {
		c.VectorStore = storage.NewCircuitBreakerVectorStore(retryStore, nil)
		c.Logger.Info("Vector store circuit breaker enabled")
	} else {
		c.VectorStore = retryStore
		c.Logger.Info("Vector store circuit breaker disabled")
	}
}

// initializeServices sets up core services
func (c *Container) initializeServices() {
	// Initialize embedding service
	// Convert global OpenAI config to embeddings config
	embeddingConfig := &embeddings.OpenAIConfig{
		APIKey:         c.Config.OpenAI.APIKey,
		BaseURL:        "https://api.openai.com/v1", // Default base URL
		Model:          c.Config.OpenAI.EmbeddingModel,
		Timeout:        time.Duration(c.Config.OpenAI.RequestTimeout) * time.Second,
		MaxRetries:     3,           // Default max retries
		RetryDelay:     time.Second, // Default retry delay
		CacheSize:      1000,        // Default cache size
		CacheTTL:       time.Hour,   // Default TTL
		RequestsPerMin: c.Config.OpenAI.RateLimitRPM,
	}

	// Convert logger interface to *slog.Logger
	// For now, always use default logger regardless of c.Logger state
	// TODO: Implement proper logger conversion when needed
	slogLogger := slog.Default()

	baseEmbedding, embeddingErr := embeddings.NewOpenAIService(embeddingConfig, slogLogger)
	if embeddingErr != nil {
		// Log error but continue with mock embeddings service
		c.Logger.Warn("Failed to initialize embedding service", "error", embeddingErr)
		c.EmbeddingService = &mockEmbeddingService{}
	} else {
		// For now, use base service without retries and circuit breakers
		// TODO: Implement wrapper services for retry and circuit breaking
		c.EmbeddingService = baseEmbedding
	}

	// Initialize chunking service
	c.ChunkingService = chunking.NewService(&c.Config.Chunking, c.EmbeddingService)

	// Initialize backup manager
	backupDir := os.Getenv("MCP_MEMORY_BACKUP_DIRECTORY")
	if backupDir == "" {
		backupDir = "./backups"
	}
	c.BackupManager = persistence.NewBackupManager(c.VectorStore, backupDir)

	// Initialize relationship manager
	c.RelationshipManager = relationships.NewManager()

	// Initialize chain components
	c.ChainStore = chains.NewInMemoryChainStore()
	chainAnalyzer := chains.NewDefaultChainAnalyzer(c.EmbeddingService)
	c.ChainBuilder = chains.NewChainBuilder(c.ChainStore, chainAnalyzer)

	// Initialize threading components
	c.ThreadStore = threading.NewInMemoryThreadStore()
	c.ThreadManager = threading.NewThreadManager(c.ChainBuilder, c.RelationshipManager, c.ThreadStore)

	// Initialize memory analytics
	c.MemoryAnalytics = analytics.NewMemoryAnalytics(c.VectorStore)

	// Initialize audit logger
	auditDir := os.Getenv("MCP_MEMORY_AUDIT_DIRECTORY")
	if auditDir == "" {
		auditDir = "./audit_logs"
	}
	var err error
	c.AuditLogger, err = audit.NewLogger(auditDir)
	if err != nil {
		// Log error but don't fail initialization
		fmt.Printf("Warning: Failed to initialize audit logger: %v\n", err)
	}
}

// initializeIntelligence sets up intelligence layer
func (c *Container) initializeIntelligence() {
	// Initialize pattern storage - use SQL if database is available, otherwise use adapter
	if c.DB != nil {
		c.PatternStorage = storage.NewPatternSQLStorage(c.DB, c.Logger)
		c.Logger.Info("Using SQL-based pattern storage")
	} else {
		c.PatternStorage = storage.NewPatternStorageAdapter(c.VectorStore)
		c.Logger.Info("Using vector store adapter for pattern storage")
	}

	// Initialize pattern engine with full dependencies including AI
	if c.DB != nil && c.AIService != nil && c.EmbeddingService != nil {
		// Create pattern engine with full AI integration using adapter
		engineConfig := intelligence.DefaultPatternEngineConfig()
		aiAdapter := intelligence.NewAIServiceAdapter(c.AIService)
		c.PatternEngine = intelligence.NewPatternEngineWithDependencies(
			c.DB,
			c.PatternStorage,
			aiAdapter,
			c.EmbeddingService,
			c.Logger,
			engineConfig,
		)
		c.Logger.Info("Pattern engine initialized with full AI integration")
	} else {
		// Fall back to basic pattern engine without AI
		c.PatternEngine = intelligence.NewPatternEngine(c.PatternStorage)
		c.Logger.Info("Pattern engine initialized in basic mode (missing dependencies)")
	}

	// Initialize graph builder
	c.GraphBuilder = intelligence.NewGraphBuilder(c.PatternEngine)

	// Initialize learning engine
	c.LearningEngine = intelligence.NewLearningEngine(c.PatternEngine, c.GraphBuilder)

	// Initialize multi-repository engine
	c.MultiRepoEngine = intelligence.NewMultiRepoEngine(c.PatternEngine, c.GraphBuilder, c.LearningEngine)
}

// initializeWorkflow sets up workflow components
func (c *Container) initializeWorkflow() {
	// Initialize workflow components
	c.TodoTracker = workflow.NewTodoTracker()
	c.FlowDetector = workflow.NewFlowDetector()
	c.PatternAnalyzer = workflow.NewPatternAnalyzer()

	// Initialize context suggester with dependencies and adapter
	vectorStorage := storage.NewVectorStorageAdapter(c.VectorStore)
	c.ContextSuggester = workflow.NewContextSuggester(
		vectorStorage,
		c.PatternAnalyzer,
		c.TodoTracker,
		c.FlowDetector,
	)
}

// HealthCheck performs health checks on all services
func (c *Container) HealthCheck(ctx context.Context) error {
	// Check vector store
	if err := c.VectorStore.HealthCheck(ctx); err != nil {
		return fmt.Errorf("vector store health check failed: %w", err)
	}

	// Check embedding service
	if err := c.EmbeddingService.HealthCheck(ctx); err != nil {
		return fmt.Errorf("embedding service health check failed: %w", err)
	}

	return nil
}

// createSlogAdapter creates an slog.Logger that forwards to the server's logger
func (c *Container) createSlogAdapter() *slog.Logger {
	return slog.New(&loggerAdapter{logger: c.Logger})
}

// loggerAdapter adapts the server's Logger interface to slog.Handler
type loggerAdapter struct {
	logger logging.Logger
}

func (a *loggerAdapter) Enabled(ctx context.Context, level slog.Level) bool {
	return true // Enable all levels, let the underlying logger filter
}

//nolint:gocritic // hugeParam: slog.Handler interface requires Record by value
func (a *loggerAdapter) Handle(ctx context.Context, record slog.Record) error {
	// Convert slog attributes to key-value pairs
	fields := make([]interface{}, 0, record.NumAttrs()*2)
	record.Attrs(func(attr slog.Attr) bool {
		fields = append(fields, attr.Key, attr.Value.Any())
		return true
	})

	// Forward to the server's logger based on level
	switch record.Level {
	case slog.LevelDebug:
		a.logger.Debug(record.Message, fields...)
	case slog.LevelInfo:
		a.logger.Info(record.Message, fields...)
	case slog.LevelWarn:
		a.logger.Warn(record.Message, fields...)
	case slog.LevelError:
		a.logger.Error(record.Message, fields...)
	default:
		a.logger.Info(record.Message, fields...)
	}
	return nil
}

func (a *loggerAdapter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return a // For simplicity, we don't track persistent attributes
}

func (a *loggerAdapter) WithGroup(name string) slog.Handler {
	return a // For simplicity, we don't support groups
}

// Shutdown gracefully shuts down all services
func (c *Container) Shutdown() error {
	// Stop analytics first to flush any pending data
	if c.MemoryAnalytics != nil {
		c.MemoryAnalytics.Stop()
	}

	// Stop audit logger to flush pending logs
	if c.AuditLogger != nil {
		c.AuditLogger.Stop()
	}

	// Close pattern engine if it has batch processing
	if c.PatternEngine != nil {
		if err := c.PatternEngine.Close(); err != nil {
			c.Logger.Error("Failed to close pattern engine", "error", err)
		}
	}

	// Close database connection
	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			c.Logger.Error("Failed to close database", "error", err)
		}
	}

	if c.VectorStore != nil {
		if err := c.VectorStore.Close(); err != nil {
			return fmt.Errorf("failed to close vector store: %w", err)
		}
	}
	return nil
}

// Provider functions for individual services

// GetVectorStore returns the vector store instance
func (c *Container) GetVectorStore() storage.VectorStore {
	return c.VectorStore
}

// GetEmbeddingService returns the embedding service instance
func (c *Container) GetEmbeddingService() embeddings.EmbeddingService {
	return c.EmbeddingService
}

// GetChunkingService returns the chunking service instance
func (c *Container) GetChunkingService() *chunking.Service {
	return c.ChunkingService
}

// GetContextSuggester returns the context suggester instance
func (c *Container) GetContextSuggester() *workflow.ContextSuggester {
	return c.ContextSuggester
}

// GetBackupManager returns the backup manager instance
func (c *Container) GetBackupManager() *persistence.BackupManager {
	return c.BackupManager
}

// GetRelationshipManager returns the relationship manager instance
func (c *Container) GetRelationshipManager() *relationships.Manager {
	return c.RelationshipManager
}

// GetLearningEngine returns the learning engine instance
func (c *Container) GetLearningEngine() *intelligence.LearningEngine {
	return c.LearningEngine
}

// GetPatternAnalyzer returns the pattern analyzer instance
func (c *Container) GetPatternAnalyzer() *workflow.PatternAnalyzer {
	return c.PatternAnalyzer
}

// GetChainBuilder returns the chain builder instance
func (c *Container) GetChainBuilder() *chains.ChainBuilder {
	return c.ChainBuilder
}

// GetChainStore returns the chain store instance
func (c *Container) GetChainStore() chains.ChainStore {
	return c.ChainStore
}

// GetMemoryAnalytics returns the memory analytics instance
func (c *Container) GetMemoryAnalytics() *analytics.MemoryAnalytics {
	return c.MemoryAnalytics
}

// GetAuditLogger returns the audit logger instance
func (c *Container) GetAuditLogger() *audit.Logger {
	return c.AuditLogger
}

// GetThreadManager returns the thread manager instance
func (c *Container) GetThreadManager() *threading.ThreadManager {
	return c.ThreadManager
}

// GetThreadStore returns the thread store instance
func (c *Container) GetThreadStore() threading.ThreadStore {
	return c.ThreadStore
}

// GetMultiRepoEngine returns the multi-repository engine instance
func (c *Container) GetMultiRepoEngine() *intelligence.MultiRepoEngine {
	return c.MultiRepoEngine
}

// GetPatternStorage returns the pattern storage instance
func (c *Container) GetPatternStorage() intelligence.PatternStorage {
	return c.PatternStorage
}

// GetPatternEngine returns the pattern engine instance
func (c *Container) GetPatternEngine() *intelligence.PatternEngine {
	return c.PatternEngine
}

// GetAIService returns the AI service instance
func (c *Container) GetAIService() sharedai.AIService {
	return c.AIService
}

// GetDB returns the database connection
func (c *Container) GetDB() *sql.DB {
	return c.DB
}

// GetLogger returns the logger instance
func (c *Container) GetLogger() logging.Logger {
	return c.Logger
}

// initializeRealTimeSync sets up WebSocket handler and real-time sync coordinator
func (c *Container) initializeRealTimeSync() {
	// Initialize WebSocket handler
	c.WebSocketHandler = handlers.NewWebSocketHandler(c.Config, nil)

	// Initialize real-time sync coordinator
	c.RealtimeSyncCoord = sync.NewRealtimeSyncCoordinator(c.WebSocketHandler)

	c.Logger.Info("Real-time sync components initialized")
}

// GetWebSocketHandler returns the WebSocket handler instance
func (c *Container) GetWebSocketHandler() *handlers.WebSocketHandler {
	return c.WebSocketHandler
}

// GetRealtimeSyncCoordinator returns the real-time sync coordinator instance
func (c *Container) GetRealtimeSyncCoordinator() *sync.RealtimeSyncCoordinator {
	return c.RealtimeSyncCoord
}

// mockEmbeddingService provides a mock implementation for testing
type mockEmbeddingService struct{}

func (m *mockEmbeddingService) Generate(ctx context.Context, text string) ([]float64, error) {
	// Return a mock embedding vector of 1536 dimensions (OpenAI ada-002 size)
	embedding := make([]float64, 1536)
	for i := range embedding {
		embedding[i] = 0.1 // Simple mock value
	}
	return embedding, nil
}

func (m *mockEmbeddingService) GenerateBatch(ctx context.Context, texts []string) ([][]float64, error) {
	result := make([][]float64, len(texts))
	for i := range texts {
		result[i] = make([]float64, 1536)
		for j := range result[i] {
			result[i][j] = 0.1 // Simple mock value
		}
	}
	return result, nil
}

func (m *mockEmbeddingService) GetDimensions() int {
	return 1536
}

func (m *mockEmbeddingService) HealthCheck(ctx context.Context) error {
	return nil
}

// buildDatabaseURL constructs database URL from environment variables
func buildDatabaseURL() string {
	// Try to get full URL first
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return dbURL
	}

	// Construct from individual components with defaults
	host := getEnvWithDefault("POSTGRES_HOST", "localhost")
	port := getEnvWithDefault("POSTGRES_PORT", "5432")
	user := getEnvWithDefault("POSTGRES_USER", "postgres")
	password := getEnvWithDefault("POSTGRES_PASSWORD", "postgres")
	dbname := getEnvWithDefault("POSTGRES_DB", "mcp_memory")
	sslmode := getEnvWithDefault("POSTGRES_SSLMODE", "disable")

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
