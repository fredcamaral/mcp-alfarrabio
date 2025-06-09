package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/services"
)

// createWorkflowCommand creates the 'workflow' command group
func (c *CLI) createWorkflowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Run complete development automation workflow",
		Long:  `Execute the complete document generation chain from PRD to sub-tasks.`,
	}

	// Add subcommands
	cmd.AddCommand(
		c.createWorkflowRunCommand(),
		c.createWorkflowStatusCommand(),
	)

	return cmd
}

// createWorkflowRunCommand creates the 'workflow run' command
func (c *CLI) createWorkflowRunCommand() *cobra.Command {
	var (
		input  string
		output string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run complete automation workflow",
		Long:  `Execute the complete document generation chain: PRD → TRD → Main Tasks → Sub-tasks`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runWorkflow(input, output)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&input, "input", "i", "", "Input description or file")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory for generated documents")

	return cmd
}

// createWorkflowStatusCommand creates the 'workflow status' command
func (c *CLI) createWorkflowStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show workflow execution status",
		Long:  `Display the status of running or completed workflow executions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runWorkflowStatus()
		},
	}

	return cmd
}

// runWorkflow executes the complete automation workflow
func (c *CLI) runWorkflow(input, output string) error {
	if c.documentChain == nil {
		return fmt.Errorf("document chain service not available")
	}

	fmt.Printf("🔄 Starting Complete Development Workflow\n")
	fmt.Printf("=====================================\n\n")

	ctx := context.Background()

	// Use default input if not provided
	if input == "" {
		input = "Create a sample application with user management, data persistence, and REST API"
	}

	// Execute full document chain
	result, err := c.documentChain.ExecuteFullChain(ctx, input)
	if err != nil {
		return fmt.Errorf("workflow execution failed: %w", err)
	}

	// Display progress summary
	fmt.Printf("✅ Workflow completed successfully!\n\n")

	fmt.Printf("📊 Results:\n")
	if result.PRD != nil {
		fmt.Printf("   PRD: %s\n", result.PRD.Title)
	}
	if result.TRD != nil {
		fmt.Printf("   TRD: %s\n", result.TRD.Title)
	}
	fmt.Printf("   Main Tasks: %d\n", len(result.MainTasks))
	fmt.Printf("   Sub-tasks: %d\n", len(result.SubTasks))
	fmt.Printf("   Total Duration: %s\n", result.Metadata.Duration)
	fmt.Printf("\n")

	// Display task breakdown
	if len(result.MainTasks) > 0 {
		fmt.Printf("📋 Main Tasks Generated:\n")
		for i, task := range result.MainTasks {
			fmt.Printf("   %d. %s (%s)\n", i+1, task.Name, task.Duration)
		}
		fmt.Printf("\n")
	}

	if len(result.SubTasks) > 0 {
		fmt.Printf("🔧 Sub-tasks Generated:\n")
		totalHours := 0
		for _, task := range result.SubTasks {
			totalHours += task.Duration
		}
		fmt.Printf("   Total: %d sub-tasks (%d hours estimated)\n", len(result.SubTasks), totalHours)
		fmt.Printf("   Average: %.1f hours per sub-task\n", float64(totalHours)/float64(len(result.SubTasks)))
		fmt.Printf("\n")
	}

	// Save documents if output specified
	if output != "" {
		if err := c.saveWorkflowResults(result, output); err != nil {
			fmt.Printf("⚠️  Failed to save results to %s: %v\n", output, err)
		} else {
			fmt.Printf("📁 Documents saved to: %s\n", output)
		}
	}

	fmt.Printf("💡 Next steps:\n")
	fmt.Printf("   - Review generated documents\n")
	fmt.Printf("   - Start implementing tasks using 'lmmc add' and 'lmmc start'\n")
	fmt.Printf("   - Track progress with 'lmmc list' and 'lmmc stats'\n")

	return nil
}

// runWorkflowStatus shows workflow execution status
func (c *CLI) runWorkflowStatus() error {
	if c.documentChain == nil {
		return fmt.Errorf("document chain service not available")
	}

	fmt.Printf("📊 Workflow Status\n")
	fmt.Printf("==================\n\n")

	// Get all chains
	chains, err := c.documentChain.ListChains()
	if err != nil {
		return fmt.Errorf("failed to get workflow status: %w", err)
	}

	if len(chains) == 0 {
		fmt.Printf("No workflow executions found.\n")
		fmt.Printf("Run 'lmmc workflow run' to start a new workflow.\n")
		return nil
	}

	// Display chain status
	for i, chain := range chains {
		fmt.Printf("Workflow %d:\n", i+1)
		fmt.Printf("  ID: %s\n", chain.ChainID)
		fmt.Printf("  Status: %s\n", chain.Status)
		fmt.Printf("  Progress: %.1f%%\n", chain.Progress*100)
		fmt.Printf("  Current Step: %s\n", chain.CurrentStep)
		fmt.Printf("  Started: %s\n", chain.StartTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Last Update: %s\n", chain.LastUpdate.Format("2006-01-02 15:04:05"))

		if chain.ErrorMessage != "" {
			fmt.Printf("  Error: %s\n", chain.ErrorMessage)
		}

		if len(chain.StepsComplete) > 0 {
			fmt.Printf("  Completed Steps: %d\n", len(chain.StepsComplete))
		}

		if len(chain.StepsFailed) > 0 {
			fmt.Printf("  Failed Steps: %d\n", len(chain.StepsFailed))
		}

		fmt.Printf("\n")
	}

	return nil
}

// saveWorkflowResults saves all generated documents to output directory
func (c *CLI) saveWorkflowResults(result *services.ChainResult, outputDir string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save PRD
	if result.PRD != nil {
		prdPath := filepath.Join(outputDir, "prd.md")
		content := c.formatPRDAsMarkdown(result.PRD)
		if err := os.WriteFile(prdPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to save PRD: %w", err)
		}
	}

	// Save TRD
	if result.TRD != nil {
		trdPath := filepath.Join(outputDir, "trd.md")
		content := c.formatTRDAsMarkdown(result.TRD)
		if err := os.WriteFile(trdPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to save TRD: %w", err)
		}
	}

	// Save main tasks
	if len(result.MainTasks) > 0 {
		mainTasksPath := filepath.Join(outputDir, "main_tasks.md")
		content := c.formatMainTasksAsText(result.MainTasks)
		if err := os.WriteFile(mainTasksPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to save main tasks: %w", err)
		}
	}

	// Save sub-tasks
	if len(result.SubTasks) > 0 {
		subTasksPath := filepath.Join(outputDir, "sub_tasks.md")
		content := c.formatSubTasksAsText(result.SubTasks)
		if err := os.WriteFile(subTasksPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to save sub-tasks: %w", err)
		}
	}

	// Save workflow summary
	summaryPath := filepath.Join(outputDir, "workflow_summary.md")
	summaryContent := c.formatWorkflowSummary(result)
	if err := os.WriteFile(summaryPath, []byte(summaryContent), 0600); err != nil {
		return fmt.Errorf("failed to save workflow summary: %w", err)
	}

	return nil
}

// formatWorkflowSummary formats a workflow execution summary
func (c *CLI) formatWorkflowSummary(result *services.ChainResult) string {
	var content strings.Builder

	content.WriteString("# Workflow Execution Summary\n\n")
	content.WriteString(fmt.Sprintf("**Execution ID:** %s\n", result.ID))
	content.WriteString(fmt.Sprintf("**Started:** %s\n", result.Metadata.StartTime.Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("**Completed:** %s\n", result.Metadata.EndTime.Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("**Duration:** %s\n", result.Metadata.Duration))
	content.WriteString(fmt.Sprintf("**Repository:** %s\n", result.Metadata.Repository))
	content.WriteString(fmt.Sprintf("**Project Type:** %s\n\n", result.Metadata.ProjectType))

	content.WriteString("## Generated Documents\n\n")
	if result.PRD != nil {
		content.WriteString(fmt.Sprintf("- **PRD:** %s (prd.md)\n", result.PRD.Title))
	}
	if result.TRD != nil {
		content.WriteString(fmt.Sprintf("- **TRD:** %s (trd.md)\n", result.TRD.Title))
	}
	content.WriteString(fmt.Sprintf("- **Main Tasks:** %d tasks (main_tasks.md)\n", len(result.MainTasks)))
	content.WriteString(fmt.Sprintf("- **Sub-tasks:** %d tasks (sub_tasks.md)\n\n", len(result.SubTasks)))

	content.WriteString("## Execution Statistics\n\n")
	content.WriteString(fmt.Sprintf("- **Total Tasks Generated:** %d\n", result.Metadata.TotalTasks))
	content.WriteString(fmt.Sprintf("- **User Inputs Processed:** %d\n", result.Metadata.UserInputCount))
	content.WriteString(fmt.Sprintf("- **Generated By:** %s\n\n", result.Metadata.GeneratedBy))

	if result.Progress != nil {
		content.WriteString("## Workflow Progress\n\n")
		content.WriteString(fmt.Sprintf("- **Status:** %s\n", result.Progress.Status))
		content.WriteString(fmt.Sprintf("- **Progress:** %.1f%%\n", result.Progress.Progress*100))
		content.WriteString(fmt.Sprintf("- **Steps Completed:** %d\n", len(result.Progress.StepsComplete)))
		content.WriteString(fmt.Sprintf("- **Steps Failed:** %d\n\n", len(result.Progress.StepsFailed)))
	}

	content.WriteString("## Next Steps\n\n")
	content.WriteString("1. Review the generated PRD and TRD documents\n")
	content.WriteString("2. Start implementing tasks using the CLI task management commands\n")
	content.WriteString("3. Track progress and update task status as work progresses\n")
	content.WriteString("4. Use the generated sub-tasks as a detailed implementation guide\n\n")

	content.WriteString("---\n")
	content.WriteString(fmt.Sprintf("*Generated by lmmc workflow on %s*\n", time.Now().Format("2006-01-02 15:04:05")))

	return content.String()
}
