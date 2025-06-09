package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Test Config Command Structure
func TestConfigCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createConfigCommand()

	// Test command properties
	assert.Equal(t, "config", cmd.Name())
	assert.Contains(t, cmd.Short, "configuration")
	assert.NotEmpty(t, cmd.Long)
}

func TestConfigGetCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createConfigGetCommand()

	// Test command properties
	assert.Equal(t, "get", cmd.Name())
	assert.Contains(t, cmd.Short, "Get")

	// Test args validation - should require exactly 1 arg
	assert.True(t, cmd.Args != nil)
}

func TestConfigSetCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createConfigSetCommand()

	// Test command properties
	assert.Equal(t, "set", cmd.Name())
	assert.Contains(t, cmd.Short, "Set")

	// Test args validation - should require exactly 2 args
	assert.True(t, cmd.Args != nil)
}

func TestConfigListCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createConfigListCommand()

	// Test command properties
	assert.Equal(t, "list", cmd.Name())
	assert.Contains(t, cmd.Short, "List")

	// Config list command doesn't have specific flags
}

func TestConfigResetCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createConfigResetCommand()

	// Test command properties
	assert.Equal(t, "reset", cmd.Name())
	assert.Contains(t, cmd.Short, "Reset")

	// Test flags
	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)
}

func TestConfigPathCommandStructure(t *testing.T) {
	cli := createTestCLI()

	cmd := cli.createConfigPathCommand()

	// Test command properties
	assert.Equal(t, "path", cmd.Name())
	assert.Contains(t, cmd.Short, "path")
}

// Test Config Subcommand Arguments
func TestConfigSubcommandArgs(t *testing.T) {
	cli := createTestCLI()

	tests := []struct {
		name     string
		cmd      *cobra.Command
		args     []string
		hasError bool
	}{
		// Config get tests
		{"get with key", cli.createConfigGetCommand(), []string{"cli.output_format"}, false},
		{"get without key", cli.createConfigGetCommand(), []string{}, true},
		{"get with multiple keys", cli.createConfigGetCommand(), []string{"key1", "key2"}, true},

		// Config set tests
		{"set with key and value", cli.createConfigSetCommand(), []string{"cli.output_format", "json"}, false},
		{"set with only key", cli.createConfigSetCommand(), []string{"cli.output_format"}, true},
		{"set without args", cli.createConfigSetCommand(), []string{}, true},
		{"set with extra args", cli.createConfigSetCommand(), []string{"key", "value", "extra"}, true},

		// Config list tests (no args required)
		{"list without args", cli.createConfigListCommand(), []string{}, false},
		{"list with args", cli.createConfigListCommand(), []string{"extra"}, false}, // Usually allowed

		// Config reset tests (no args required)
		{"reset without args", cli.createConfigResetCommand(), []string{}, false},
		{"reset with args", cli.createConfigResetCommand(), []string{"extra"}, false}, // Usually allowed

		// Config path tests (no args required)
		{"path without args", cli.createConfigPathCommand(), []string{}, false},
		{"path with args", cli.createConfigPathCommand(), []string{"extra"}, false}, // Usually allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cmd.Args != nil {
				err := tt.cmd.Args(tt.cmd, tt.args)
				if tt.hasError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

// Test Config Command Tree Structure
func TestConfigCommandTree(t *testing.T) {
	cli := createTestCLI()

	// Get the config command with its subcommands already added
	configCmd := cli.createConfigCommand()

	// Test that all subcommands are present
	subcommands := configCmd.Commands()
	assert.Len(t, subcommands, 5)

	subcommandNames := make([]string, len(subcommands))
	for i, cmd := range subcommands {
		subcommandNames[i] = cmd.Name()
	}

	assert.Contains(t, subcommandNames, "get")
	assert.Contains(t, subcommandNames, "set")
	assert.Contains(t, subcommandNames, "list")
	assert.Contains(t, subcommandNames, "reset")
	assert.Contains(t, subcommandNames, "path")
}

// Test Config Help Text
func TestConfigCommandHelp(t *testing.T) {
	cli := createTestCLI()

	commands := []*cobra.Command{
		cli.createConfigCommand(),
		cli.createConfigGetCommand(),
		cli.createConfigSetCommand(),
		cli.createConfigListCommand(),
		cli.createConfigResetCommand(),
		cli.createConfigPathCommand(),
	}

	for _, cmd := range commands {
		t.Run(cmd.Name(), func(t *testing.T) {
			// All commands should have short help
			assert.NotEmpty(t, cmd.Short)

			// All commands should have usage
			assert.NotEmpty(t, cmd.Use)

			// Commands should have reasonable help text
			assert.True(t, len(cmd.Short) > 5, "Short help should be descriptive")
			assert.True(t, len(cmd.Short) < 150, "Short help should be concise")
		})
	}
}

// Test Config Flag Consistency
func TestConfigFlagConsistency(t *testing.T) {
	cli := createTestCLI()

	// Note: Config commands use root persistent output flag, not their own

	// Commands that should have force flag
	commandsWithForce := []*cobra.Command{
		cli.createConfigResetCommand(),
	}

	for _, cmd := range commandsWithForce {
		t.Run(cmd.Name()+"_force_flag", func(t *testing.T) {
			forceFlag := cmd.Flags().Lookup("force")
			assert.NotNil(t, forceFlag)
			assert.Equal(t, "f", forceFlag.Shorthand)
		})
	}
}
