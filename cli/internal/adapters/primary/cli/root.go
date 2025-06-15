// Package cli provides command-line interface implementation
// for the lerian-mcp-memory CLI application.
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/adapters/primary/commands"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// IntelligenceDependencies holds intelligence service dependencies
type IntelligenceDependencies struct {
	PatternDetector   services.PatternDetector
	SuggestionService services.SuggestionService
	TemplateService   services.TemplateService
	AnalyticsService  services.AnalyticsService
	CrossRepoAnalyzer services.CrossRepoAnalyzer
}

// CLI represents the command-line interface
type CLI struct {
	RootCmd            *cobra.Command // Exported for version setting
	taskService        *services.TaskService
	configMgr          ports.ConfigManager
	logger             *slog.Logger
	storage            ports.Storage
	outputFormat       string
	verbose            bool
	documentChain      services.DocumentChainService
	aiService          ports.AIService
	repositoryDetector ports.RepositoryDetector
	batchSyncService   *services.BatchSyncService
	reviewService      *services.ReviewService

	// Auto sync management
	autoSyncCancel context.CancelFunc
	autoSyncActive bool

	// Intelligence services (optional)
	intelligence *IntelligenceDependencies
}

// NewCLI creates a new CLI instance
func NewCLI(
	taskService *services.TaskService,
	configMgr ports.ConfigManager,
	logger *slog.Logger,
	storage ports.Storage,
	documentChain services.DocumentChainService,
	aiService ports.AIService,
	repositoryDetector ports.RepositoryDetector,
	batchSyncService *services.BatchSyncService,
) *CLI {
	return NewCLIWithIntelligence(
		taskService,
		configMgr,
		logger,
		storage,
		documentChain,
		aiService,
		repositoryDetector,
		batchSyncService,
		IntelligenceDependencies{}, // Empty intelligence dependencies
	)
}

// NewCLIWithIntelligence creates a new CLI instance with intelligence services
func NewCLIWithIntelligence(
	taskService *services.TaskService,
	configMgr ports.ConfigManager,
	logger *slog.Logger,
	storage ports.Storage,
	documentChain services.DocumentChainService,
	aiService ports.AIService,
	repositoryDetector ports.RepositoryDetector,
	batchSyncService *services.BatchSyncService,
	intelligence IntelligenceDependencies,
) *CLI {
	return NewCLIWithIntelligenceAndReview(
		taskService,
		configMgr,
		logger,
		storage,
		documentChain,
		aiService,
		repositoryDetector,
		batchSyncService,
		nil, // No review service
		intelligence,
	)
}

// NewCLIWithIntelligenceAndReview creates a new CLI instance with intelligence and review services
func NewCLIWithIntelligenceAndReview(
	taskService *services.TaskService,
	configMgr ports.ConfigManager,
	logger *slog.Logger,
	storage ports.Storage,
	documentChain services.DocumentChainService,
	aiService ports.AIService,
	repositoryDetector ports.RepositoryDetector,
	batchSyncService *services.BatchSyncService,
	reviewService *services.ReviewService,
	intelligence IntelligenceDependencies,
) *CLI {
	cli := &CLI{
		taskService:        taskService,
		configMgr:          configMgr,
		logger:             logger,
		storage:            storage,
		documentChain:      documentChain,
		aiService:          aiService,
		repositoryDetector: repositoryDetector,
		batchSyncService:   batchSyncService,
		reviewService:      reviewService,
		intelligence:       &intelligence,
	}

	cli.setupRootCommand()
	cli.setupCommands()

	return cli
}

// setupRootCommand configures the root command
func (c *CLI) setupRootCommand() {
	c.RootCmd = &cobra.Command{
		Use:   "lmmc",
		Short: "Lerian MCP Memory CLI - Intelligent task management",
		Long: `Lerian MCP Memory CLI (lmmc) is a powerful task management tool 
that integrates with AI assistants and provides intelligent task suggestions.

It maintains task lists per repository, supports filtering and prioritization,
and can be integrated with the Lerian MCP Memory Server for AI-powered features.`,
		Version: "0.1.0",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			config, err := c.configMgr.Load()
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// Apply CLI configuration
			if c.outputFormat == "" {
				c.outputFormat = config.CLI.OutputFormat
			}

			// Setup logging if verbose
			if c.verbose {
				c.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelDebug,
				}))
			}

			return nil
		},
		SilenceUsage: true,
	}

	// Global flags
	c.RootCmd.PersistentFlags().StringVarP(&c.outputFormat, "output", "o", "",
		"Output format (table, json, plain)")
	c.RootCmd.PersistentFlags().BoolVarP(&c.verbose, "verbose", "v", false,
		"Verbose output for debugging")
}

// setupCommands adds all subcommands to root
func (c *CLI) setupCommands() {
	commands := []*cobra.Command{
		c.createAddCommand(),
		c.createListCommand(),
		c.createStartCommand(),
		c.createDoneCommand(),
		c.createCancelCommand(),
		c.createEditCommand(),
		c.createPriorityCommand(),
		c.createDeleteCommand(),
		c.createConfigCommand(),
		c.createStatsCommand(),
		c.createSearchCommand(),
		c.createPRDCommand(),
		c.createTRDCommand(),
		c.createTasksCommand(),    // Replaces taskgen
		c.createSubtasksCommand(), // New subtasks command
		c.createWorkflowCommand(),
		c.createREPLCommand(), // TUI with comprehensive dashboards
		c.createSyncCommand(),
		c.createAICommand(),         // AI-powered task processing and memory management
		c.createCompletionCommand(), // Shell completion support
		c.createVersionCommand(),    // Version information
		c.createUpdateCommand(),     // Self-update mechanism
		c.createDocsCommand(),       // OpenAPI documentation generation
		c.createExportCommand(),     // Multi-format export capabilities
		c.createStatusCommand(),     // Service status overview
		c.createReviewCommand(),     // Code review functionality
		c.createMemoryCommand(),     // Memory MCP integration
	}

	// Add intelligence-based commands if available
	if c.intelligence != nil && c.intelligence.AnalyticsService != nil {
		commands = append(commands, c.createAnalyticsCommand())
	}
	if c.intelligence != nil && c.intelligence.SuggestionService != nil {
		commands = append(commands, c.createSuggestCommand())
	}

	c.RootCmd.AddCommand(commands...)
}

// Execute runs the CLI
func (c *CLI) Execute() error {
	return c.RootCmd.Execute()
}

// getOutputFormatter returns the appropriate formatter based on output format
func (c *CLI) getOutputFormatter(cmd *cobra.Command) OutputFormatter {
	format := c.outputFormat

	// Check if format was overridden in command
	if f, _ := cmd.Flags().GetString("output"); f != "" {
		format = f
	}

	writer := cmd.OutOrStdout()

	switch strings.ToLower(format) {
	case "json":
		return NewJSONFormatter(writer, true)
	case "plain":
		return NewPlainFormatter(writer)
	default:
		return NewTableFormatter(writer)
	}
}

// handleError formats and displays an error
func (c *CLI) handleError(cmd *cobra.Command, err error) error {
	formatter := c.getOutputFormatter(cmd)
	_ = formatter.FormatError(err)
	return err
}

// parseStatus converts string to Status enum
func parseStatus(s string) (entities.Status, error) {
	switch strings.ToLower(s) {
	case "pending":
		return entities.StatusPending, nil
	case "in_progress", "in-progress", "inprogress":
		return entities.StatusInProgress, nil
	case "completed", "done":
		return entities.StatusCompleted, nil
	case "cancelled", "canceled":
		return entities.StatusCancelled, nil
	default:
		return "", fmt.Errorf("invalid status: %s (valid: pending, in_progress, completed, cancelled)", s)
	}
}

// parsePriority converts string to Priority enum
func parsePriority(p string) (entities.Priority, error) {
	switch strings.ToLower(p) {
	case "low":
		return entities.PriorityLow, nil
	case "medium", "med":
		return entities.PriorityMedium, nil
	case "high":
		return entities.PriorityHigh, nil
	default:
		return "", fmt.Errorf("invalid priority: %s (valid: low, medium, high)", p)
	}
}

// getContext returns a context for command execution
func (c *CLI) getContext() context.Context {
	return context.Background()
}

// getMCPClient returns the MCP client if available
func (c *CLI) getMCPClient() ports.MCPClient {
	if c.batchSyncService != nil {
		return c.batchSyncService.GetMCPClient()
	}
	return nil
}

// createAnalyticsCommand creates the analytics command when intelligence services are available
func (c *CLI) createAnalyticsCommand() *cobra.Command {
	// Create dependencies for analytics command
	var config *entities.Config
	if c.configMgr != nil {
		config, _ = c.configMgr.Load() // Use loaded config or create minimal config
	}
	if config == nil {
		// Create minimal config if none available
		config = &entities.Config{
			CLI: entities.CLIConfig{
				DefaultRepository: "default",
				OutputFormat:      "table",
			},
		}
	}

	deps := commands.CommandDeps{
		TaskService:      c.taskService,
		Logger:           c.logger,
		Config:           config,
		AnalyticsService: c.intelligence.AnalyticsService,
		// Add other dependencies as needed
	}

	return commands.NewAnalyticsCommand(deps)
}
