// Package di provides dependency injection container for the application
package di

import (
	"context"
	"fmt"
	"os"
	"mcp-memory/internal/chunking"
	"mcp-memory/internal/config"
	"mcp-memory/internal/embeddings"
	"mcp-memory/internal/intelligence"
	"mcp-memory/internal/persistence"
	"mcp-memory/internal/storage"
	"mcp-memory/internal/workflow"
	"mcp-memory/internal/chains"
)

// Container holds all application dependencies
type Container struct {
	Config           *config.Config
	VectorStore      storage.VectorStore
	EmbeddingService embeddings.EmbeddingService
	ChunkingService  *chunking.ChunkingService
	ContextSuggester *workflow.ContextSuggester
	BackupManager    *persistence.BackupManager
	LearningEngine   *intelligence.LearningEngine
	PatternAnalyzer  *workflow.PatternAnalyzer
	TodoTracker      *workflow.TodoTracker
	FlowDetector     *workflow.FlowDetector
	PatternEngine    *intelligence.PatternEngine
	GraphBuilder     *intelligence.GraphBuilder
	ChainBuilder     *chains.ChainBuilder
	ChainStore       chains.ChainStore
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *config.Config) (*Container, error) {
	container := &Container{
		Config: cfg,
	}

	// Initialize in dependency order
	if err := container.initializeStorage(); err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	if err := container.initializeServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	if err := container.initializeIntelligence(); err != nil {
		return nil, fmt.Errorf("failed to initialize intelligence: %w", err)
	}

	if err := container.initializeWorkflow(); err != nil {
		return nil, fmt.Errorf("failed to initialize workflow: %w", err)
	}

	return container, nil
}

// initializeStorage sets up storage layer
func (c *Container) initializeStorage() error {
	// Initialize vector store
	var baseStore storage.VectorStore
	var err error
	
	// Use pooled store if connection pooling is enabled
	if usePooling := os.Getenv("CHROMA_USE_POOLING"); usePooling == "true" {
		baseStore, err = storage.NewPooledChromaStore(&c.Config.Chroma)
		if err != nil {
			return fmt.Errorf("failed to create pooled Chroma store: %w", err)
		}
	} else {
		baseStore = storage.NewChromaStore(&c.Config.Chroma)
	}
	
	// Wrap with retry logic
	retryStore := storage.NewRetryableVectorStore(baseStore, nil)
	
	// Wrap with circuit breaker if enabled
	if useCircuitBreaker := os.Getenv("USE_CIRCUIT_BREAKER"); useCircuitBreaker == "true" {
		c.VectorStore = storage.NewCircuitBreakerVectorStore(retryStore, nil)
	} else {
		c.VectorStore = retryStore
	}
	
	return nil
}

// initializeServices sets up core services
func (c *Container) initializeServices() error {
	// Initialize embedding service
	baseEmbedding := embeddings.NewOpenAIEmbeddingService(&c.Config.OpenAI)
	
	// Wrap with retry logic
	retryEmbedding := embeddings.NewRetryableEmbeddingService(baseEmbedding, nil)
	
	// Wrap with circuit breaker if enabled
	if useCircuitBreaker := os.Getenv("USE_CIRCUIT_BREAKER"); useCircuitBreaker == "true" {
		c.EmbeddingService = embeddings.NewCircuitBreakerEmbeddingService(retryEmbedding, nil)
	} else {
		c.EmbeddingService = retryEmbedding
	}

	// Initialize chunking service
	c.ChunkingService = chunking.NewChunkingService(&c.Config.Chunking, c.EmbeddingService)

	// Initialize backup manager
	backupDir := os.Getenv("MCP_MEMORY_BACKUP_DIRECTORY")
	if backupDir == "" {
		backupDir = "./backups"
	}
	c.BackupManager = persistence.NewBackupManager(nil, backupDir) // Note: VectorStore interface compatibility issue

	return nil
}

// initializeIntelligence sets up intelligence layer
func (c *Container) initializeIntelligence() error {
	// Initialize pattern engine
	// Note: VectorStore interface compatibility issue - using nil for now
	c.PatternEngine = intelligence.NewPatternEngine(nil)

	// Initialize graph builder
	c.GraphBuilder = intelligence.NewGraphBuilder(c.PatternEngine)

	// Initialize learning engine
	c.LearningEngine = intelligence.NewLearningEngine(c.PatternEngine, c.GraphBuilder)

	return nil
}

// initializeWorkflow sets up workflow components
func (c *Container) initializeWorkflow() error {
	// Initialize workflow components
	c.TodoTracker = workflow.NewTodoTracker()
	c.FlowDetector = workflow.NewFlowDetector()
	c.PatternAnalyzer = workflow.NewPatternAnalyzer()

	// Initialize context suggester with dependencies
	// Note: VectorStore interface compatibility issue - using nil for now
	c.ContextSuggester = workflow.NewContextSuggester(
		nil,
		c.PatternAnalyzer,
		c.TodoTracker,
		c.FlowDetector,
	)

	return nil
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
func (c *Container) GetChunkingService() *chunking.ChunkingService {
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