package cli

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// Mock implementations for testing
type MockConfigManager struct{}

func (m *MockConfigManager) Load() (*entities.Config, error) {
	return &entities.Config{
		CLI: entities.CLIConfig{
			OutputFormat: "table",
			PageSize:     20,
		},
	}, nil
}

func (m *MockConfigManager) Save(config *entities.Config) error {
	return nil
}

func (m *MockConfigManager) GetConfigPath() string {
	return "/tmp/test-config.yaml"
}

func (m *MockConfigManager) Set(key, value string) error {
	return nil
}

func (m *MockConfigManager) Get(key string) (interface{}, error) {
	return "table", nil
}

func (m *MockConfigManager) Validate() error {
	return nil
}

func (m *MockConfigManager) Reset() error {
	return nil
}

// Test NewCLI function structure
func TestNewCLIStructure(t *testing.T) {
	// Create mock dependencies
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	configMgr := &MockConfigManager{}

	// For the actual NewCLI function, we need a real TaskService
	// Let's test the structure without full integration

	// Test CLI struct creation
	cli := &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		taskService:  nil, // Would be a real service in production
		configMgr:    configMgr,
		logger:       logger,
		outputFormat: "table",
		verbose:      false,
	}

	require.NotNil(t, cli)
	assert.Equal(t, configMgr, cli.configMgr)
	assert.Equal(t, logger, cli.logger)
	assert.Equal(t, "table", cli.outputFormat)
	assert.False(t, cli.verbose)
	assert.NotNil(t, cli.RootCmd)
}

// Test Root Command Setup
func TestRootCommandSetup(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cli := &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		configMgr:    &MockConfigManager{},
		logger:       logger,
		outputFormat: "table",
	}

	// Setup root command
	cli.setupRootCommand()

	// Test root command properties
	assert.Equal(t, "lmmc", cli.RootCmd.Use)
	assert.Contains(t, cli.RootCmd.Short, "task management")
	assert.NotEmpty(t, cli.RootCmd.Long)

	// Test that root command has persistent flags
	persistentFlags := cli.RootCmd.PersistentFlags()

	verboseFlag := persistentFlags.Lookup("verbose")
	assert.NotNil(t, verboseFlag)
	assert.Equal(t, "v", verboseFlag.Shorthand)

	// Note: quiet flag is not implemented in current version

	outputFlag := persistentFlags.Lookup("output")
	assert.NotNil(t, outputFlag)
	assert.Equal(t, "o", outputFlag.Shorthand)
}

// Test Commands Setup
func TestCommandsSetup(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cli := &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		configMgr:    &MockConfigManager{},
		logger:       logger,
		outputFormat: "table",
	}

	// Setup commands
	cli.setupCommands()

	// Test that commands are added
	commands := cli.RootCmd.Commands()
	assert.True(t, len(commands) > 0)

	// Get command names
	commandNames := make([]string, len(commands))
	for i, cmd := range commands {
		commandNames[i] = cmd.Name()
	}

	// Test that expected commands exist
	expectedCommands := []string{
		"add", "list", "start", "done", "cancel",
		"edit", "priority", "delete", "stats", "search", "config",
	}

	for _, expected := range expectedCommands {
		assert.Contains(t, commandNames, expected, "Command %s should exist", expected)
	}
}

// Test Execute method
func TestExecute(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cli := &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		configMgr:    &MockConfigManager{},
		logger:       logger,
		outputFormat: "table",
	}

	// Test that Execute method exists and can be called
	// We won't actually execute since we don't have real services
	assert.NotNil(t, cli.Execute)

	// Test that Execute returns error from root command
	cli.RootCmd.SetArgs([]string{"--version"})
	// We can't easily test execution without mocking everything
	// But we can test that the method exists and has the right signature
}

// Test Output Formatter Logic
func TestGetOutputFormatter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cli := &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		configMgr:    &MockConfigManager{},
		logger:       logger,
		outputFormat: "table",
	}

	// Create a test command with flags
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("output", "", "output format")

	// Test default format (table)
	formatter := cli.getOutputFormatter(cmd)
	assert.NotNil(t, formatter)

	// Test with JSON flag
	_ = cmd.Flags().Set("output", "json")
	formatter = cli.getOutputFormatter(cmd)
	assert.NotNil(t, formatter)

	// Test with plain flag
	_ = cmd.Flags().Set("output", "plain")
	formatter = cli.getOutputFormatter(cmd)
	assert.NotNil(t, formatter)

	// Test with invalid flag (should default to table)
	_ = cmd.Flags().Set("output", "invalid")
	formatter = cli.getOutputFormatter(cmd)
	assert.NotNil(t, formatter)
}

// Test Context Handling
func TestGetContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cli := &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		configMgr:    &MockConfigManager{},
		logger:       logger,
		outputFormat: "table",
	}

	ctx := cli.getContext()
	require.NotNil(t, ctx)

	// Test that it's a proper context
	select {
	case <-ctx.Done():
		t.Error("Context should not be done immediately")
	default:
		// Expected behavior
	}

	// Test context type
	assert.IsType(t, context.Background(), ctx)
}

// Test Error Handling
func TestHandleError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cli := &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		configMgr:    &MockConfigManager{},
		logger:       logger,
		outputFormat: "table",
	}

	// Create a test command
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("output", "", "output format")

	// Test error handling
	testErr := entities.ErrInvalidStatusTransition
	handledErr := cli.handleError(cmd, testErr)

	// Error should be returned as-is
	assert.Equal(t, testErr, handledErr)
}

// Test CLI Field Access
func TestCLIFields(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	configMgr := &MockConfigManager{}

	cli := &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		taskService:  nil,
		configMgr:    configMgr,
		logger:       logger,
		outputFormat: "json",
		verbose:      true,
	}

	// Test field access
	assert.Equal(t, "lmmc", cli.RootCmd.Use)
	assert.Nil(t, cli.taskService)
	assert.Equal(t, configMgr, cli.configMgr)
	assert.Equal(t, logger, cli.logger)
	assert.Equal(t, "json", cli.outputFormat)
	assert.True(t, cli.verbose)
}

// Test CLI Method Signatures
func TestCLIMethodSignatures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cli := &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		configMgr:    &MockConfigManager{},
		logger:       logger,
		outputFormat: "table",
	}

	// Test that required methods exist with correct signatures

	// Execute method
	assert.NotNil(t, cli.Execute)

	// getOutputFormatter method
	cmd := &cobra.Command{}
	cmd.Flags().String("output", "", "")
	formatter := cli.getOutputFormatter(cmd)
	assert.NotNil(t, formatter)

	// getContext method
	ctx := cli.getContext()
	assert.NotNil(t, ctx)

	// handleError method
	err := cli.handleError(cmd, entities.ErrInvalidStatusTransition)
	assert.Error(t, err)

	// setupRootCommand method
	assert.NotPanics(t, func() {
		cli.setupRootCommand()
	})

	// setupCommands method
	assert.NotPanics(t, func() {
		cli.setupCommands()
	})
}

// Test Command Factory Methods
func TestCommandFactoryMethods(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cli := &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		configMgr:    &MockConfigManager{},
		logger:       logger,
		outputFormat: "table",
	}

	// Test that all command factory methods exist and return commands
	commands := map[string]func() *cobra.Command{
		"add":      cli.createAddCommand,
		"list":     cli.createListCommand,
		"start":    cli.createStartCommand,
		"done":     cli.createDoneCommand,
		"cancel":   cli.createCancelCommand,
		"edit":     cli.createEditCommand,
		"priority": cli.createPriorityCommand,
		"delete":   cli.createDeleteCommand,
		"stats":    cli.createStatsCommand,
		"search":   cli.createSearchCommand,
		"config":   cli.createConfigCommand,
	}

	for name, factory := range commands {
		t.Run(name, func(t *testing.T) {
			cmd := factory()
			assert.NotNil(t, cmd)
			assert.Equal(t, name, cmd.Name())
			assert.NotEmpty(t, cmd.Short)
		})
	}
}

// Test Config Command Factory Methods
func TestConfigCommandFactoryMethods(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cli := &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		configMgr:    &MockConfigManager{},
		logger:       logger,
		outputFormat: "table",
	}

	// Test config subcommand factory methods
	configCommands := map[string]func() *cobra.Command{
		"get":   cli.createConfigGetCommand,
		"set":   cli.createConfigSetCommand,
		"list":  cli.createConfigListCommand,
		"reset": cli.createConfigResetCommand,
		"path":  cli.createConfigPathCommand,
	}

	for name, factory := range configCommands {
		t.Run(name, func(t *testing.T) {
			cmd := factory()
			assert.NotNil(t, cmd)
			assert.Equal(t, name, cmd.Name())
			assert.NotEmpty(t, cmd.Short)
		})
	}
}
