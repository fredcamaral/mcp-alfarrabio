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
	"lerian-mcp-memory-cli/internal/adapters/secondary/config"
	"lerian-mcp-memory-cli/internal/adapters/secondary/mcp"
	"lerian-mcp-memory-cli/internal/adapters/secondary/repository"
	"lerian-mcp-memory-cli/internal/adapters/secondary/storage"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
	"lerian-mcp-memory-cli/internal/domain/services"
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
	CLI                *cli.CLI

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

	// Initialize task service
	container.initTaskService()

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

	container.initTaskService()

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

// initCLI initializes the CLI
func (c *Container) initCLI() {
	c.CLI = cli.NewCLI(c.TaskService, c.ConfigManager, c.Logger)
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
