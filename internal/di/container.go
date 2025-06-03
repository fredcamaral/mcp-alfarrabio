// Package di provides dependency injection container for the application
package di

import (
	"context"
	"fmt"
	"mcp-memory/internal/analytics"
	"mcp-memory/internal/audit"
	"mcp-memory/internal/chains"
	"mcp-memory/internal/chunking"
	"mcp-memory/internal/config"
	"mcp-memory/internal/embeddings"
	"mcp-memory/internal/intelligence"
	"mcp-memory/internal/persistence"
	"mcp-memory/internal/relationships"
	"mcp-memory/internal/storage"
	"mcp-memory/internal/threading"
	"mcp-memory/internal/workflow"
	"os"
)

const envValueTrue = "true"

// Container holds all application dependencies
type Container struct {
	Config              *config.Config
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
	GraphBuilder        *intelligence.GraphBuilder
	MultiRepoEngine     *intelligence.MultiRepoEngine
	ChainBuilder        *chains.ChainBuilder
	ChainStore          chains.ChainStore
	RelationshipManager *relationships.Manager
	ThreadManager       *threading.ThreadManager
	ThreadStore         threading.ThreadStore
	MemoryAnalytics     *analytics.MemoryAnalytics
	AuditLogger         *audit.Logger
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *config.Config) (*Container, error) {
	container := &Container{
		Config: cfg,
	}

	// Initialize in dependency order
	container.initializeStorage()

	container.initializeServices()
	container.initializeIntelligence()
	container.initializeWorkflow()

	return container, nil
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

	// Wrap with circuit breaker if enabled
	if useCircuitBreaker := os.Getenv("USE_CIRCUIT_BREAKER"); useCircuitBreaker == envValueTrue {
		c.VectorStore = storage.NewCircuitBreakerVectorStore(retryStore, nil)
	} else {
		c.VectorStore = retryStore
	}
}

// initializeServices sets up core services
func (c *Container) initializeServices() {
	// Initialize embedding service
	baseEmbedding := embeddings.NewOpenAIEmbeddingService(&c.Config.OpenAI)

	// Wrap with retry logic
	retryEmbedding := embeddings.NewRetryableEmbeddingService(baseEmbedding, nil)

	// Wrap with circuit breaker if enabled
	if useCircuitBreaker := os.Getenv("USE_CIRCUIT_BREAKER"); useCircuitBreaker == envValueTrue {
		c.EmbeddingService = embeddings.NewCircuitBreakerEmbeddingService(retryEmbedding, nil)
	} else {
		c.EmbeddingService = retryEmbedding
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
	// Initialize pattern engine with adapter
	patternStorage := storage.NewPatternStorageAdapter(c.VectorStore)
	c.PatternEngine = intelligence.NewPatternEngine(patternStorage)

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
