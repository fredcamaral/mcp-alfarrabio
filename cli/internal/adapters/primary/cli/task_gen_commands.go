package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// createTaskGenCommand creates the 'taskgen' command group
func (c *CLI) createTaskGenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "taskgen",
		Short: "Generate tasks from PRD/TRD documents",
		Long:  `Generate main tasks and sub-tasks from Product and Technical Requirements Documents.`,
	}

	// Add subcommands
	cmd.AddCommand(
		c.createTaskGenMainCommand(),
		c.createTaskGenSubCommand(),
		c.createTaskGenAnalyzeCommand(),
	)

	return cmd
}

// createTaskGenMainCommand creates the 'taskgen main' command
func (c *CLI) createTaskGenMainCommand() *cobra.Command {
	var (
		prdFile string
		trdFile string
		output  string
	)

	cmd := &cobra.Command{
		Use:   "main",
		Short: "Generate main tasks from PRD and TRD",
		Long:  `Generate main project tasks from Product and Technical Requirements Documents.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("task generation functionality not yet implemented in standalone CLI - coming soon")
		},
	}

	// Add flags
	cmd.Flags().StringVar(&prdFile, "prd", "", "PRD file path")
	cmd.Flags().StringVar(&trdFile, "trd", "", "TRD file path")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file for generated tasks")

	return cmd
}

// createTaskGenSubCommand creates the 'taskgen sub' command
func (c *CLI) createTaskGenSubCommand() *cobra.Command {
	var (
		taskID string
		output string
	)

	cmd := &cobra.Command{
		Use:   "sub",
		Short: "Generate sub-tasks for a main task",
		Long:  `Generate detailed sub-tasks for a specific main task.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("sub-task generation functionality not yet implemented in standalone CLI - coming soon")
		},
	}

	// Add flags
	cmd.Flags().StringVar(&taskID, "task", "", "Main task ID to generate sub-tasks for")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file for generated sub-tasks")

	return cmd
}

// createTaskGenAnalyzeCommand creates the 'taskgen analyze' command
func (c *CLI) createTaskGenAnalyzeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze task complexity and dependencies",
		Long:  `Analyze the complexity and dependencies of generated tasks.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("task analysis functionality not yet implemented in standalone CLI - coming soon")
		},
	}

	return cmd
}
