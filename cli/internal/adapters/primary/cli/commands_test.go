package cli

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// Mock storage for testing
type mockStorage struct{}

func (m *mockStorage) SaveTask(ctx context.Context, task *entities.Task) error {
	return nil
}

func (m *mockStorage) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	return &entities.Task{ID: id}, nil
}

func (m *mockStorage) UpdateTask(ctx context.Context, task *entities.Task) error {
	return nil
}

func (m *mockStorage) DeleteTask(ctx context.Context, id string) error {
	return nil
}

func (m *mockStorage) ListTasks(ctx context.Context, repository string, filters *ports.TaskFilters) ([]*entities.Task, error) {
	return []*entities.Task{}, nil
}

func (m *mockStorage) GetTasksByRepository(ctx context.Context, repository string) ([]*entities.Task, error) {
	return []*entities.Task{}, nil
}

func (m *mockStorage) SearchTasks(ctx context.Context, query string, filters *ports.TaskFilters) ([]*entities.Task, error) {
	return []*entities.Task{}, nil
}

func (m *mockStorage) SaveTasks(ctx context.Context, tasks []*entities.Task) error {
	return nil
}

func (m *mockStorage) DeleteTasks(ctx context.Context, ids []string) error {
	return nil
}

func (m *mockStorage) ListRepositories(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *mockStorage) GetRepositoryStats(ctx context.Context, repository string) (ports.RepositoryStats, error) {
	return ports.RepositoryStats{}, nil
}

func (m *mockStorage) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *mockStorage) Backup(ctx context.Context, backupPath string) error {
	return nil
}

func (m *mockStorage) Restore(ctx context.Context, backupPath string) error {
	return nil
}

// Mock repository detector
type mockRepositoryDetector struct{}

func (m *mockRepositoryDetector) DetectCurrent(ctx context.Context) (*ports.RepositoryInfo, error) {
	return &ports.RepositoryInfo{
		Name:      "test-repo",
		Path:      "/test/path",
		Provider:  "git",
		IsGitRepo: true,
	}, nil
}

func (m *mockRepositoryDetector) DetectFromPath(ctx context.Context, path string) (*ports.RepositoryInfo, error) {
	return &ports.RepositoryInfo{
		Name:      "test-repo",
		Path:      path,
		Provider:  "git",
		IsGitRepo: true,
	}, nil
}

func (m *mockRepositoryDetector) GetRepositoryName(ctx context.Context, path string) (string, error) {
	return "test-repo", nil
}

func (m *mockRepositoryDetector) IsValidRepository(ctx context.Context, path string) bool {
	return true
}

// Helper function to create test CLI instance
func createTestCLI() *CLI {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	storage := &mockStorage{}
	repoDetector := &mockRepositoryDetector{}
	taskService := services.NewTaskService(storage, repoDetector, logger)

	return &CLI{
		RootCmd:      &cobra.Command{Use: "lmmc"},
		taskService:  taskService,
		logger:       logger,
		outputFormat: "table",
	}
}

// Test helper functions for parsing
func TestParsePriority(t *testing.T) {
	tests := []struct {
		input    string
		expected entities.Priority
		hasError bool
	}{
		{"high", entities.PriorityHigh, false},
		{"medium", entities.PriorityMedium, false},
		{"low", entities.PriorityLow, false},
		{"med", entities.PriorityMedium, false},
		{"HIGH", entities.PriorityHigh, false},
		{"Medium", entities.PriorityMedium, false},
		{"h", entities.Priority(""), true},
		{"m", entities.Priority(""), true},
		{"l", entities.Priority(""), true},
		{"1", entities.Priority(""), true},
		{"2", entities.Priority(""), true},
		{"3", entities.Priority(""), true},
		{"invalid", entities.Priority(""), true},
		{"", entities.Priority(""), true},
		{"0", entities.Priority(""), true},
		{"4", entities.Priority(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parsePriority(tt.input)
			if tt.hasError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid priority")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected entities.Status
		hasError bool
	}{
		{"pending", entities.StatusPending, false},
		{"in_progress", entities.StatusInProgress, false},
		{"in-progress", entities.StatusInProgress, false},
		{"inprogress", entities.StatusInProgress, false},
		{"completed", entities.StatusCompleted, false},
		{"done", entities.StatusCompleted, false},
		{"cancelled", entities.StatusCancelled, false},
		{"canceled", entities.StatusCancelled, false},
		{"PENDING", entities.StatusPending, false},
		{"In_Progress", entities.StatusInProgress, false},
		{"COMPLETED", entities.StatusCompleted, false},
		{"invalid", entities.StatusPending, true},
		{"", entities.StatusPending, true},
		{"active", entities.StatusPending, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseStatus(tt.input)
			if tt.hasError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid status")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Test command structure and flags
func TestAddCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createAddCommand()

	// Test command properties
	assert.Equal(t, "add", cmd.Name())
	assert.Contains(t, cmd.Short, "Create a new task")
	assert.NotEmpty(t, cmd.Long)

	// Test flags
	priorityFlag := cmd.Flags().Lookup("priority")
	assert.NotNil(t, priorityFlag)
	assert.Equal(t, "p", priorityFlag.Shorthand)

	tagsFlag := cmd.Flags().Lookup("tags")
	assert.NotNil(t, tagsFlag)
	assert.Equal(t, "t", tagsFlag.Shorthand)

	estimateFlag := cmd.Flags().Lookup("estimate")
	assert.NotNil(t, estimateFlag)
	assert.Equal(t, "e", estimateFlag.Shorthand)

	// Test args validation
	assert.True(t, cmd.Args != nil)
}

func TestListCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createListCommand()

	// Test command properties
	assert.Equal(t, "list", cmd.Name())
	assert.Contains(t, cmd.Short, "List tasks")

	// Test flags
	statusFlag := cmd.Flags().Lookup("status")
	assert.NotNil(t, statusFlag)
	assert.Equal(t, "s", statusFlag.Shorthand)

	priorityFlag := cmd.Flags().Lookup("priority")
	assert.NotNil(t, priorityFlag)
	assert.Equal(t, "p", priorityFlag.Shorthand)

	tagsFlag := cmd.Flags().Lookup("tags")
	assert.NotNil(t, tagsFlag)
	assert.Equal(t, "t", tagsFlag.Shorthand)

	repositoryFlag := cmd.Flags().Lookup("repository")
	assert.NotNil(t, repositoryFlag)
	assert.Equal(t, "r", repositoryFlag.Shorthand)
}

func TestStartCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createStartCommand()

	// Test command properties
	assert.Equal(t, "start", cmd.Name())
	assert.Contains(t, cmd.Short, "Mark a task as in progress")

	// Test args validation - should require exactly 1 arg
	assert.True(t, cmd.Args != nil)
}

func TestDoneCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createDoneCommand()

	// Test command properties
	assert.Equal(t, "done", cmd.Name())
	assert.Contains(t, cmd.Short, "Mark a task as completed")

	// Test flags
	actualFlag := cmd.Flags().Lookup("actual")
	assert.NotNil(t, actualFlag)
	assert.Equal(t, "a", actualFlag.Shorthand)

	// Test args validation
	assert.True(t, cmd.Args != nil)
}

func TestCancelCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createCancelCommand()

	// Test command properties
	assert.Equal(t, "cancel", cmd.Name())
	assert.Contains(t, cmd.Short, "Cancel")

	// Test args validation
	assert.True(t, cmd.Args != nil)
}

func TestEditCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createEditCommand()

	// Test command properties
	assert.Equal(t, "edit", cmd.Name())
	assert.Contains(t, cmd.Short, "Edit")

	// Test flags
	contentFlag := cmd.Flags().Lookup("content")
	assert.NotNil(t, contentFlag)
	assert.Equal(t, "c", contentFlag.Shorthand)

	addTagFlag := cmd.Flags().Lookup("add-tags")
	assert.NotNil(t, addTagFlag)

	removeTagFlag := cmd.Flags().Lookup("remove-tags")
	assert.NotNil(t, removeTagFlag)

	estimateFlag := cmd.Flags().Lookup("estimate")
	assert.NotNil(t, estimateFlag)
	assert.Equal(t, "e", estimateFlag.Shorthand)

	// Test args validation
	assert.True(t, cmd.Args != nil)
}

func TestPriorityCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createPriorityCommand()

	// Test command properties
	assert.Equal(t, "priority", cmd.Name())
	assert.Contains(t, cmd.Short, "priority")

	// Test args validation - should require exactly 2 args
	assert.True(t, cmd.Args != nil)
}

func TestDeleteCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createDeleteCommand()

	// Test command properties
	assert.Equal(t, "delete", cmd.Name())
	assert.Contains(t, cmd.Short, "Delete")

	// Test flags
	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)

	// Test args validation
	assert.True(t, cmd.Args != nil)
}

func TestStatsCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createStatsCommand()

	// Test command properties
	assert.Equal(t, "stats", cmd.Name())
	assert.Contains(t, cmd.Short, "statistics")

	// Test flags
	repositoryFlag := cmd.Flags().Lookup("repository")
	assert.NotNil(t, repositoryFlag)
	assert.Equal(t, "r", repositoryFlag.Shorthand)
}

func TestSearchCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createSearchCommand()

	// Test command properties
	assert.Equal(t, "search", cmd.Name())
	assert.Contains(t, cmd.Short, "Search")

	// Test flags
	statusFlag := cmd.Flags().Lookup("status")
	assert.NotNil(t, statusFlag)
	assert.Equal(t, "s", statusFlag.Shorthand)

	priorityFlag := cmd.Flags().Lookup("priority")
	assert.NotNil(t, priorityFlag)
	assert.Equal(t, "p", priorityFlag.Shorthand)

	tagsFlag := cmd.Flags().Lookup("tags")
	assert.NotNil(t, tagsFlag)
	assert.Equal(t, "t", tagsFlag.Shorthand)

	// Test args validation
	assert.True(t, cmd.Args != nil)
}

// Test command validation logic
func TestCommandArgsValidation(t *testing.T) {
	cli := createTestCLI()

	tests := []struct {
		name     string
		cmd      *cobra.Command
		args     []string
		hasError bool
	}{
		{"add with content", cli.createAddCommand(), []string{"Test task"}, false},
		{"add without content", cli.createAddCommand(), []string{}, true},
		{"add with multiple words", cli.createAddCommand(), []string{"Task", "with", "multiple", "words"}, false},

		{"start with ID", cli.createStartCommand(), []string{"task-123"}, false},
		{"start without ID", cli.createStartCommand(), []string{}, true},
		{"start with multiple args", cli.createStartCommand(), []string{"task-123", "extra"}, true},

		{"done with ID", cli.createDoneCommand(), []string{"task-123"}, false},
		{"done without ID", cli.createDoneCommand(), []string{}, true},
		{"done with multiple args", cli.createDoneCommand(), []string{"task-123", "extra"}, true},

		{"cancel with ID", cli.createCancelCommand(), []string{"task-123"}, false},
		{"cancel without ID", cli.createCancelCommand(), []string{}, true},

		{"edit with ID", cli.createEditCommand(), []string{"task-123"}, false},
		{"edit without ID", cli.createEditCommand(), []string{}, true},

		{"priority with ID and value", cli.createPriorityCommand(), []string{"task-123", "high"}, false},
		{"priority with only ID", cli.createPriorityCommand(), []string{"task-123"}, true},
		{"priority without args", cli.createPriorityCommand(), []string{}, true},

		{"delete with ID", cli.createDeleteCommand(), []string{"task-123"}, false},
		{"delete without ID", cli.createDeleteCommand(), []string{}, true},

		{"search with query", cli.createSearchCommand(), []string{"test"}, false},
		{"search without query", cli.createSearchCommand(), []string{}, true},
		{"search with multiple words", cli.createSearchCommand(), []string{"test", "query"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cmd.Args(tt.cmd, tt.args)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test flag validation
func TestFlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		flagValue   string
		parseFunc   func(string) (interface{}, error)
		expectError bool
	}{
		// Priority validation
		{"valid priority high", "high", func(s string) (interface{}, error) { return parsePriority(s) }, false},
		{"valid priority medium", "medium", func(s string) (interface{}, error) { return parsePriority(s) }, false},
		{"valid priority low", "low", func(s string) (interface{}, error) { return parsePriority(s) }, false},
		{"invalid priority", "invalid", func(s string) (interface{}, error) { return parsePriority(s) }, true},

		// Status validation
		{"valid status pending", "pending", func(s string) (interface{}, error) { return parseStatus(s) }, false},
		{"valid status in_progress", "in_progress", func(s string) (interface{}, error) { return parseStatus(s) }, false},
		{"valid status completed", "completed", func(s string) (interface{}, error) { return parseStatus(s) }, false},
		{"invalid status", "invalid", func(s string) (interface{}, error) { return parseStatus(s) }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.parseFunc(tt.flagValue)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test output format handling
func TestOutputFormatDetection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name          string
		defaultFormat string
		flagFormat    string
		expectedType  string
	}{
		{"default table", "table", "", "TableFormatter"},
		{"default json", "json", "", "JSONFormatter"},
		{"default plain", "plain", "", "PlainFormatter"},
		{"flag override table", "json", "table", "TableFormatter"},
		{"flag override json", "table", "json", "JSONFormatter"},
		{"flag override plain", "table", "plain", "PlainFormatter"},
		{"invalid default", "invalid", "", "TableFormatter"},
		{"invalid flag", "table", "invalid", "TableFormatter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := &CLI{
				RootCmd:      &cobra.Command{Use: "lmmc"},
				logger:       logger,
				outputFormat: tt.defaultFormat,
			}

			cmd := &cobra.Command{}
			cmd.Flags().String("output", "", "output format")
			if tt.flagFormat != "" {
				_ = cmd.Flags().Set("output", tt.flagFormat)
			}

			formatter := cli.getOutputFormatter(cmd)
			formatterType := strings.Contains(fmt.Sprintf("%T", formatter), tt.expectedType)
			assert.True(t, formatterType, "Expected %s in %T", tt.expectedType, formatter)
		})
	}
}

// Test context handling
func TestContextHandling(t *testing.T) {
	cli := createTestCLI()

	ctx := cli.getContext()
	require.NotNil(t, ctx)

	// Test that it's a proper context
	select {
	case <-ctx.Done():
		t.Error("Context should not be done immediately")
	default:
		// Expected behavior
	}

	// Test context has deadline (or not)
	_, hasDeadline := ctx.Deadline()
	assert.False(t, hasDeadline, "Background context should not have deadline")
}

// Test error handling
func TestErrorHandling(t *testing.T) {
	cli := createTestCLI()
	output := &bytes.Buffer{}

	cmd := &cobra.Command{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.Flags().String("output", "", "output format")

	testErr := entities.ErrInvalidStatusTransition
	handledErr := cli.handleError(cmd, testErr)

	// Error should be returned as-is
	assert.Equal(t, testErr, handledErr)

	// Output should contain error information
	outputStr := output.String()
	assert.Contains(t, outputStr, "invalid")
}

// Test command aliases and shortcuts
func TestCommandAliases(t *testing.T) {
	cli := createTestCLI()

	// Test that commands have expected aliases
	listCmd := cli.createListCommand()
	assert.Contains(t, listCmd.Aliases, "ls")

	// Note: Other commands might have aliases too, check the implementation
}

// Test command help text
func TestCommandHelpText(t *testing.T) {
	cli := createTestCLI()

	commands := []*cobra.Command{
		cli.createAddCommand(),
		cli.createListCommand(),
		cli.createStartCommand(),
		cli.createDoneCommand(),
		cli.createCancelCommand(),
		cli.createEditCommand(),
		cli.createPriorityCommand(),
		cli.createDeleteCommand(),
		cli.createStatsCommand(),
		cli.createSearchCommand(),
	}

	for _, cmd := range commands {
		t.Run(cmd.Name(), func(t *testing.T) {
			// All commands should have short help
			assert.NotEmpty(t, cmd.Short)

			// All commands should have usage
			assert.NotEmpty(t, cmd.Use)

			// Commands should have reasonable help text
			assert.True(t, len(cmd.Short) > 10, "Short help should be descriptive")
			assert.True(t, len(cmd.Short) < 100, "Short help should be concise")
		})
	}
}

// Test flag consistency across commands
func TestFlagConsistency(t *testing.T) {
	cli := createTestCLI()

	// Note: Output flag is a persistent flag at root level, not per-command

	// Commands that should have status flag
	commandsWithStatus := []*cobra.Command{
		cli.createListCommand(),
		cli.createSearchCommand(),
	}

	for _, cmd := range commandsWithStatus {
		t.Run(cmd.Name()+"_status_flag", func(t *testing.T) {
			statusFlag := cmd.Flags().Lookup("status")
			assert.NotNil(t, statusFlag)
			assert.Equal(t, "s", statusFlag.Shorthand)
		})
	}

	// Commands that should have priority flag
	commandsWithPriority := []*cobra.Command{
		cli.createAddCommand(),
		cli.createListCommand(),
		cli.createSearchCommand(),
	}

	for _, cmd := range commandsWithPriority {
		t.Run(cmd.Name()+"_priority_flag", func(t *testing.T) {
			priorityFlag := cmd.Flags().Lookup("priority")
			assert.NotNil(t, priorityFlag)
			assert.Equal(t, "p", priorityFlag.Shorthand)
		})
	}
}
