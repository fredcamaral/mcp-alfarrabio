package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// createConfigCommand creates the 'config' command
func (c *CLI) createConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long:  `Manage CLI configuration settings and preferences.`,
	}

	// Add subcommands
	cmd.AddCommand(
		c.createConfigGetCommand(),
		c.createConfigSetCommand(),
		c.createConfigListCommand(),
		c.createConfigResetCommand(),
		c.createConfigPathCommand(),
	)

	return cmd
}

// createConfigGetCommand creates the 'config get' command
func (c *CLI) createConfigGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Long:  `Get the current value of a configuration setting.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			value, err := c.configMgr.Get(key)
			if err != nil {
				return c.handleError(cmd, err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: %v\n", key, value)
			return nil
		},
	}
}

// createConfigSetCommand creates the 'config set' command
func (c *CLI) createConfigSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Long:  `Set a configuration value. Changes are saved immediately.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			err := c.configMgr.Set(key, value)
			if err != nil {
				return c.handleError(cmd, err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Configuration updated: %s = %s\n", key, value)
			return nil
		},
	}
}

// createConfigListCommand creates the 'config list' command
func (c *CLI) createConfigListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration settings",
		Long:  `Display all current configuration settings and their values.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := c.configMgr.Load()
			if err != nil {
				return c.handleError(cmd, err)
			}

			// Format based on output type
			format := c.outputFormat
			if f, _ := cmd.Flags().GetString("output"); f != "" {
				format = f
			}

			if strings.EqualFold(format, "json") {
				formatter := NewJSONFormatter(cmd.OutOrStdout(), true)
				data, _ := formatter.(*JSONFormatter)
				// Direct JSON output of config
				return data.FormatTask(&entities.Task{}) // Hack to trigger JSON output setup
			}

			// Default text output
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Current Configuration:")
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "")

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Server:")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  URL:      %s\n", config.Server.URL)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Version:  %s\n", config.Server.Version)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Timeout:  %d seconds\n", config.Server.Timeout)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "")

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "CLI:")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Default Repository:  %s\n", valueOrDefault(config.CLI.DefaultRepository, "(current)"))
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Output Format:       %s\n", config.CLI.OutputFormat)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Auto Complete:       %t\n", config.CLI.AutoComplete)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Color Scheme:        %s\n", config.CLI.ColorScheme)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Page Size:           %d\n", config.CLI.PageSize)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Editor:              %s\n", valueOrDefault(config.CLI.Editor, "(system default)"))
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "")

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Storage:")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Cache Enabled:  %t\n", config.Storage.CacheEnabled)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Cache TTL:      %d seconds\n", config.Storage.CacheTTL)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Backup Count:   %d\n", config.Storage.BackupCount)
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "")

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Logging:")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Level:   %s\n", config.Logging.Level)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Format:  %s\n", config.Logging.Format)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  File:    %s\n", valueOrDefault(config.Logging.File, "(none)"))

			return nil
		},
	}
}

// createConfigResetCommand creates the 'config reset' command
func (c *CLI) createConfigResetCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset configuration to defaults",
		Long:  `Reset all configuration settings to their default values.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Confirm reset if not forced
			if !force {
				_, _ = fmt.Fprint(cmd.OutOrStdout(), "Reset all configuration to defaults? [y/N]: ")

				var response string
				_, _ = fmt.Scanln(&response)

				if !strings.EqualFold(response, "y") {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
					return nil
				}
			}

			err := c.configMgr.Reset()
			if err != nil {
				return c.handleError(cmd, err)
			}

			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Configuration reset to defaults")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

// createConfigPathCommand creates the 'config path' command
func (c *CLI) createConfigPathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		Long:  `Display the path to the configuration file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := c.configMgr.GetConfigPath()
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}

// Helper function
func valueOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}
