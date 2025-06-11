// Package di provides dependency injection container
// for the lerian-mcp-memory CLI application.
package di

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"lerian-mcp-memory-cli/internal/adapters/primary/cli"
	"lerian-mcp-memory-cli/internal/adapters/secondary/ai"
	"lerian-mcp-memory-cli/internal/adapters/secondary/analytics"
	"lerian-mcp-memory-cli/internal/adapters/secondary/config"
	"lerian-mcp-memory-cli/internal/adapters/secondary/filesystem"
	"lerian-mcp-memory-cli/internal/adapters/secondary/mcp"
	"lerian-mcp-memory-cli/internal/adapters/secondary/repository"
	"lerian-mcp-memory-cli/internal/adapters/secondary/storage"
	"lerian-mcp-memory-cli/internal/adapters/secondary/visualization"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
	"lerian-mcp-memory-cli/internal/domain/services"
	sharedai "lerian-mcp-memory/pkg/ai"
)

// Container holds all application dependencies
type Container struct {
	Config             *entities.Config
	ConfigManager      ports.ConfigManager
	Logger             *slog.Logger
	Storage            ports.Storage
	MCPClient          ports.MCPClient
	TaskService        *services.TaskService
	RepositoryDetector ports.RepositoryDetector
	AIService          ports.AIService
	DocumentChain      services.DocumentChainService
	BatchSyncService   *services.BatchSyncService
	CLI                *cli.CLI

	// Intelligence Services
	PatternDetector   services.PatternDetector
	SuggestionService services.SuggestionService
	TemplateService   services.TemplateService
	AnalyticsService  services.AnalyticsService
	CrossRepoAnalyzer services.CrossRepoAnalyzer
	ContextAnalyzer   services.ContextAnalyzer
	ProjectClassifier services.ProjectClassifier
	MetricsCalculator *services.MetricsCalculator

	// Intelligence Adapters
	Visualizer        ports.Visualizer
	AnalyticsExporter ports.AnalyticsExporter
	AnalyticsEngine   ports.AnalyticsEngine

	// Storage for intelligence features
	PatternStore  ports.PatternStorage
	TemplateStore ports.TemplateStorage
	SessionStore  ports.SessionStorage
	InsightStore  ports.InsightStorage

	// Internal fields
	logFile *os.File
}

// NewContainer creates a new dependency injection container
func NewContainer() (*Container, error) {
	container := &Container{}

	// Initialize logger first (with default settings)
	container.initLogger()

	// Load configuration
	if err := container.initConfig(); err != nil {
		return nil, fmt.Errorf("failed to initialize config: %w", err)
	}

	// Initialize storage
	if err := container.initStorage(); err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize MCP client
	container.initMCPClient()

	// Initialize repository detector
	container.initRepositoryDetector()

	// Initialize AI service
	container.initAIService()

	// Initialize document chain service
	container.initDocumentChainService()

	// Initialize task service
	container.initTaskService()

	// Initialize batch sync service
	container.initBatchSyncService()

	// Initialize intelligence services
	container.initIntelligenceServices()

	// Initialize CLI
	container.initCLI()

	return container, nil
}

// NewTestContainer creates a container for testing with custom config
func NewTestContainer(cfg *entities.Config) (*Container, error) {
	container := &Container{
		Config: cfg,
	}

	// Initialize logger
	container.initLogger()

	// Reconfigure logger with test config
	if err := container.reconfigureLogger(); err != nil {
		return nil, err
	}

	// Initialize remaining components
	if err := container.initStorage(); err != nil {
		return nil, err
	}

	container.initMCPClient()

	container.initRepositoryDetector()

	container.initAIService()

	container.initDocumentChainService()

	container.initTaskService()

	container.initBatchSyncService()

	container.initIntelligenceServices()

	container.initCLI()

	return container, nil
}

// initLogger initializes the logger with default settings
func (c *Container) initLogger() {
	// Initially use default logger, will be reconfigured after loading config
	c.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// initConfig initializes the configuration manager and loads config
func (c *Container) initConfig() error {
	configManager, err := config.NewViperConfigManager(c.Logger)
	if err != nil {
		return err
	}
	c.ConfigManager = configManager

	cfg, err := configManager.Load()
	if err != nil {
		return err
	}
	c.Config = cfg

	// Reconfigure logger with loaded settings
	return c.reconfigureLogger()
}

// reconfigureLogger updates logger settings based on configuration
func (c *Container) reconfigureLogger() error {
	var handler slog.Handler

	// Parse log level
	level := slog.LevelInfo
	switch c.Config.Logging.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}

	// Setup log handler
	var err error
	if c.Config.Logging.File != "" {
		handler, err = c.createFileHandler(opts)
		if err != nil {
			return err
		}
	} else {
		handler = c.createConsoleHandler(opts)
	}

	c.Logger = slog.New(handler)
	return nil
}

// createFileHandler creates a file-based log handler
func (c *Container) createFileHandler(opts *slog.HandlerOptions) (slog.Handler, error) {
	// Close existing log file if any
	if c.logFile != nil {
		_ = c.logFile.Close()
	}

	file, err := os.OpenFile(c.Config.Logging.File,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	c.logFile = file

	if c.Config.Logging.Format == "json" {
		return slog.NewJSONHandler(file, opts), nil
	}
	return slog.NewTextHandler(file, opts), nil
}

// createConsoleHandler creates a console-based log handler
func (c *Container) createConsoleHandler(opts *slog.HandlerOptions) slog.Handler {
	if c.Config.Logging.Format == "json" {
		return slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.NewTextHandler(os.Stdout, opts)
}

// initStorage initializes the storage layer
func (c *Container) initStorage() error {
	fileStorage, err := storage.NewFileStorage()
	if err != nil {
		return err
	}
	c.Storage = fileStorage

	c.Logger.Info("storage initialized",
		slog.String("type", "file"),
		slog.String("path", "~/.lmmc"))
	return nil
}

// initMCPClient initializes the MCP client
func (c *Container) initMCPClient() {
	// Check if MCP is enabled
	if c.Config.Server.URL == "" {
		c.Logger.Info("MCP client disabled (no server URL configured)")
		// Use a mock client that always returns offline
		c.MCPClient = &mcp.MockMCPClient{}
		c.MCPClient.(*mcp.MockMCPClient).SetOnline(false)
		return
	}

	client := mcp.NewHTTPMCPClient(c.Config, c.Logger)
	c.MCPClient = client

	// Test connection (non-blocking)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.TestConnection(ctx); err != nil {
			c.Logger.Warn("MCP server not available, running in offline mode",
				slog.Any("error", err))
		} else {
			c.Logger.Info("MCP server connection established")
		}
	}()
}

// initRepositoryDetector initializes the repository detector
func (c *Container) initRepositoryDetector() {
	c.RepositoryDetector = repository.NewGitDetector()
}

// initTaskService initializes the task service
func (c *Container) initTaskService() {
	c.TaskService = services.NewTaskService(c.Storage, c.RepositoryDetector, c.Logger)

	// Set MCP client if available
	if c.MCPClient != nil {
		c.TaskService.SetMCPClient(c.MCPClient)
	}

	c.Logger.Info("task service initialized")
}

// initAIService initializes the AI service using shared AI package
func (c *Container) initAIService() {
	// Try to create AI service from environment variables first (for real providers)
	sharedService, err := sharedai.NewFromEnv(c.Logger)
	if err != nil {
		c.Logger.Warn("failed to create AI service from environment, falling back to mock",
			slog.Any("error", err))

		// Fall back to mock service if real providers aren't configured
		sharedService, err = sharedai.NewMockService(c.Logger)
		if err != nil {
			c.Logger.Error("failed to create mock AI service", slog.Any("error", err))
			c.AIService = nil
			return
		}
		c.Logger.Info("AI service initialized", slog.String("provider", "mock"))
	} else {
		c.Logger.Info("AI service initialized", slog.String("provider", "real-ai-from-env"))
	}

	// Create adapter that bridges shared AI service to CLI's expected interface
	c.AIService = ai.NewSharedAIService(sharedService)
}

// initDocumentChainService initializes the document chain service
func (c *Container) initDocumentChainService() {
	c.DocumentChain = services.NewDocumentChainService(c.MCPClient, c.AIService, c.Storage, c.Logger)
	c.Logger.Info("document chain service initialized")
}

// initBatchSyncService initializes the batch sync service
func (c *Container) initBatchSyncService() {
	c.BatchSyncService = services.NewBatchSyncService(c.MCPClient, c.Storage, c.Logger)
	c.Logger.Info("batch sync service initialized")
}

// initIntelligenceServices initializes all intelligence-related services
func (c *Container) initIntelligenceServices() {
	// Initialize storage layers for intelligence features
	c.initIntelligenceStorage()

	// Initialize adapters
	c.initIntelligenceAdapters()

	// Initialize core intelligence services
	c.initCoreIntelligenceServices()

	c.Logger.Info("intelligence services initialized")
}

// initIntelligenceStorage initializes storage layers for intelligence features
func (c *Container) initIntelligenceStorage() {
	// For now, use file-based storage adapters
	// In production, these could be replaced with database implementations
	var err error

	c.PatternStore, err = storage.NewFilePatternStorage(c.Logger)
	if err != nil {
		c.Logger.Error("failed to initialize pattern storage", slog.Any("error", err))
		// Use a noop implementation or panic - for now we'll panic in development
		panic(fmt.Sprintf("failed to initialize pattern storage: %v", err))
	}

	c.TemplateStore, err = storage.NewFileTemplateStorage(c.Logger)
	if err != nil {
		c.Logger.Error("failed to initialize template storage", slog.Any("error", err))
		panic(fmt.Sprintf("failed to initialize template storage: %v", err))
	}

	c.SessionStore, err = storage.NewFileSessionStorage(c.Logger)
	if err != nil {
		c.Logger.Error("failed to initialize session storage", slog.Any("error", err))
		panic(fmt.Sprintf("failed to initialize session storage: %v", err))
	}

	c.InsightStore, err = storage.NewFileInsightStorage(c.Logger)
	if err != nil {
		c.Logger.Error("failed to initialize insight storage", slog.Any("error", err))
		panic(fmt.Sprintf("failed to initialize insight storage: %v", err))
	}
}

// initIntelligenceAdapters initializes adapters for intelligence features
func (c *Container) initIntelligenceAdapters() {
	// Initialize visualizer adapter to bridge visualization.Visualizer to ports.Visualizer
	terminalViz := visualization.NewSimpleTerminalVisualizer()
	c.Visualizer = visualization.NewVisualizerAdapter(terminalViz)

	// Initialize analytics exporter
	c.AnalyticsExporter = visualization.NewAnalyticsExporter(c.Logger)

	// Initialize analytics engine
	portsTaskRepo := storage.NewPortsTaskRepositoryAdapter(c.Storage)
	portsSessionRepo := storage.NewPortsSessionRepositoryAdapter(c.SessionStore)
	c.AnalyticsEngine = analytics.NewWorkflowAnalyticsEngine(portsTaskRepo, portsSessionRepo, c.Logger)
}

// initCoreIntelligenceServices initializes the core intelligence services
func (c *Container) initCoreIntelligenceServices() {
	// Initialize metrics calculator
	c.MetricsCalculator = services.NewMetricsCalculator()

	// Create repository adapters for services that need them
	taskRepo := storage.NewTaskRepositoryAdapter(c.Storage)
	sessionRepo := storage.NewSessionRepositoryAdapter(c.SessionStore)
	patternRepo := storage.NewPatternRepositoryAdapter(c.PatternStore)

	// Initialize context analyzer
	c.ContextAnalyzer = services.NewContextAnalyzer(
		taskRepo,
		sessionRepo,
		patternRepo,
		nil, // analytics engine - will be initialized later
		nil, // config - will use defaults
		c.Logger,
	)

	// Initialize project classifier (needs file analyzer)
	fileAnalyzer := filesystem.NewFileAnalyzer(nil, c.Logger)
	c.ProjectClassifier = services.NewProjectClassifier(
		fileAnalyzer,
		nil, // config - will use defaults
		c.Logger,
	)

	// Initialize pattern detector
	c.PatternDetector = services.NewPatternDetector(
		taskRepo,
		patternRepo,
		sessionRepo,
		nil, // analytics engine - will be initialized later
		nil, // config - will use defaults
		c.Logger,
	)

	// Initialize local suggestion service
	localSuggestionService := services.NewSuggestionService(
		taskRepo,
		patternRepo,
		sessionRepo,
		c.ContextAnalyzer,
		c.PatternDetector,
		nil, // analytics engine - will be initialized later
		nil, // config - will use defaults
		c.Logger,
	)

	// Wrap with hybrid service that can use server intelligence when available
	c.SuggestionService = services.NewHybridSuggestionService(
		localSuggestionService,
		c.MCPClient,
		c.Logger,
	)

	// Initialize template service
	templateRepo := storage.NewTemplateRepositoryAdapter(c.TemplateStore)
	c.TemplateService = services.NewTemplateService(
		templateRepo,
		taskRepo,
		c.ProjectClassifier,
		nil, // config - will use defaults
		c.Logger,
	)

	// Create adapters for analytics service
	taskStore := storage.NewServicesTaskStorageAdapter(c.Storage)
	sessionStore := storage.NewServicesSessionStorageAdapter(c.SessionStore)
	visualizer := storage.NewServicesVisualizerAdapter(c.Visualizer)
	exporter := storage.NewServicesAnalyticsExporterAdapter(c.AnalyticsExporter)

	// Initialize analytics service using the analytics engine
	c.AnalyticsService = services.NewAnalyticsService(
		services.AnalyticsServiceDependencies{
			TaskStore:    taskStore,
			PatternStore: c.PatternStore,
			SessionStore: sessionStore,
			Visualizer:   visualizer,
			Exporter:     exporter,
			Calculator:   c.MetricsCalculator,
			Config:       nil, // will use defaults
			Logger:       c.Logger,
		},
	)

	// Create ports adapters for cross-repo analyzer
	portsTaskRepo := storage.NewPortsTaskRepositoryAdapter(c.Storage)
	portsSessionRepo := storage.NewPortsSessionRepositoryAdapter(c.SessionStore)

	// Initialize cross-repository analyzer
	c.CrossRepoAnalyzer = analytics.NewCrossRepoAnalyzer(
		c.AnalyticsEngine,
		portsTaskRepo,
		portsSessionRepo,
		c.PatternStore,
		c.InsightStore,
		c.Logger,
	)
}

// initCLI initializes the CLI
func (c *Container) initCLI() {
	// Create CLI dependencies struct for intelligence services
	intelligenceDeps := cli.IntelligenceDependencies{
		PatternDetector:   c.PatternDetector,
		SuggestionService: c.SuggestionService,
		TemplateService:   c.TemplateService,
		AnalyticsService:  c.AnalyticsService,
		CrossRepoAnalyzer: c.CrossRepoAnalyzer,
	}

	c.CLI = cli.NewCLIWithIntelligence(
		c.TaskService,
		c.ConfigManager,
		c.Logger,
		c.DocumentChain,
		c.AIService,
		c.RepositoryDetector,
		c.BatchSyncService,
		intelligenceDeps,
	)
}

// HealthCheck validates critical dependencies
func (c *Container) HealthCheck(ctx context.Context) error {
	// Check storage accessibility
	if err := c.Storage.HealthCheck(ctx); err != nil {
		return fmt.Errorf("storage health check failed: %w", err)
	}

	// Check repository detection (not critical)
	if _, err := c.RepositoryDetector.DetectCurrent(ctx); err != nil {
		c.Logger.Warn("repository detection failed", slog.Any("error", err))
		// Not critical, continue
	}

	c.Logger.Debug("health check passed")
	return nil
}

// Shutdown gracefully shuts down all components
func (c *Container) Shutdown(ctx context.Context) error {
	c.Logger.Info("shutting down application")

	var errs []error

	// Close MCP client if it supports closing
	if closer, ok := c.MCPClient.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close MCP client: %w", err))
		}
	}

	// Close storage if it supports closing
	if closer, ok := c.Storage.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close storage: %w", err))
		}
	}

	// Close log file if open
	if c.logFile != nil {
		if err := c.logFile.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close log file: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	c.Logger.Info("application shutdown complete")
	return nil
}

// Close gracefully closes the container and all its resources
func (c *Container) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.Shutdown(ctx)
}
