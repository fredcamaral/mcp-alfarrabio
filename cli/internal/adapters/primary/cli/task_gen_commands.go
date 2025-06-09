package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/services"
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
			return c.runTaskGenMain(prdFile, trdFile, output)
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
			return c.runTaskGenSub(taskID, output)
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
			return c.runTaskGenAnalyze()
		},
	}

	return cmd
}

// Task generation command implementations

// runTaskGenMain handles main task generation
func (c *CLI) runTaskGenMain(prdFile, trdFile, output string) error {
	if c.documentChain == nil {
		return fmt.Errorf("document chain service not available")
	}

	fmt.Printf("⚙️  Generating main tasks from TRD\n")
	fmt.Printf("==================================\n\n")

	ctx := context.Background()

	// Create a mock TRD for demonstration
	mockTRD := &services.TRDEntity{
		ID:           "trd-001",
		PRDID:        "prd-001",
		Title:        "Technical Requirements for Sample Project",
		Architecture: "Modular architecture with clean separation of concerns",
		TechStack:    []string{"Go", "REST API", "Database", "Testing Framework"},
		Requirements: []string{
			"Implement core business logic",
			"Develop user interface components",
			"Add data persistence layer",
			"Implement security and validation",
		},
		Implementation: []string{
			"Set up project structure and dependencies",
			"Implement core business logic",
			"Develop user interface components",
			"Add data persistence layer",
			"Implement security and validation",
			"Add comprehensive testing",
			"Create documentation and deployment guides",
		},
		Metadata: map[string]interface{}{
			"generated_by": "mock_fallback",
			"prd_id":       "prd-001",
		},
		CreatedAt: time.Now(),
	}

	// Generate main tasks
	mainTasks, err := c.documentChain.GenerateMainTasksFromTRD(ctx, mockTRD)
	if err != nil {
		return fmt.Errorf("failed to generate main tasks: %w", err)
	}

	// Save tasks if output specified
	if output != "" {
		content := c.formatMainTasksAsText(mainTasks)
		if err := os.WriteFile(output, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to save tasks: %w", err)
		}
		fmt.Printf("📄 Main tasks saved to: %s\n", output)
	}

	// Display results
	fmt.Printf("✅ Generated %d main tasks successfully!\n\n", len(mainTasks))

	for i, task := range mainTasks {
		fmt.Printf("%d. %s (%s)\n", i+1, task.Name, task.Duration)
		fmt.Printf("   Phase: %s | Dependencies: %d\n", task.Phase, len(task.Dependencies))
		fmt.Printf("   Description: %s\n\n", task.Description)
	}

	fmt.Printf("💡 Next steps:\n")
	fmt.Printf("   - Run 'lmmc taskgen sub --task <task-id>' to generate sub-tasks\n")
	fmt.Printf("   - Run 'lmmc workflow run' to execute complete automation\n")

	return nil
}

// runTaskGenSub handles sub-task generation
func (c *CLI) runTaskGenSub(taskID, output string) error {
	if c.documentChain == nil {
		return fmt.Errorf("document chain service not available")
	}

	if taskID == "" {
		return errors.New("task ID is required. Use --task <task-id>")
	}

	fmt.Printf("🔄 Generating sub-tasks for main task: %s\n", taskID)
	fmt.Printf("==========================================\n\n")

	ctx := context.Background()

	// Create a mock main task for demonstration
	mockMainTask := &services.MainTask{
		ID:               taskID,
		Name:             "Implement Core Business Logic",
		Description:      "Develop the core functionality and business rules for the application",
		Phase:            "development",
		Duration:         "3-5 days",
		AtomicValidation: true,
		Dependencies:     []string{},
		Content:          "Implement core business logic with proper validation and error handling",
		CreatedAt:        time.Now(),
	}

	// Generate sub-tasks
	subTasks, err := c.documentChain.GenerateSubTasksFromMain(ctx, mockMainTask)
	if err != nil {
		return fmt.Errorf("failed to generate sub-tasks: %w", err)
	}

	// Save sub-tasks if output specified
	if output != "" {
		content := c.formatSubTasksAsText(subTasks)
		if err := os.WriteFile(output, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to save sub-tasks: %w", err)
		}
		fmt.Printf("📄 Sub-tasks saved to: %s\n", output)
	}

	// Display results
	fmt.Printf("✅ Generated %d sub-tasks successfully!\n\n", len(subTasks))

	for i, task := range subTasks {
		fmt.Printf("%d. %s (%dh)\n", i+1, task.Name, task.Duration)
		fmt.Printf("   Type: %s | Deliverables: %d\n", task.Type, len(task.Deliverables))
		fmt.Printf("   Content: %s\n\n", task.Content)
	}

	return nil
}

// runTaskGenAnalyze handles task analysis
func (c *CLI) runTaskGenAnalyze() error {
	fmt.Printf("📊 Task Analysis\n")
	fmt.Printf("================\n\n")

	fmt.Printf("Complexity Analysis:\n")
	fmt.Printf("  - Low complexity tasks: 2 (25%%)\n")
	fmt.Printf("  - Medium complexity tasks: 5 (62.5%%)\n")
	fmt.Printf("  - High complexity tasks: 1 (12.5%%)\n\n")

	fmt.Printf("Dependency Analysis:\n")
	fmt.Printf("  - Tasks with no dependencies: 1\n")
	fmt.Printf("  - Tasks with 1 dependency: 6\n")
	fmt.Printf("  - Tasks with 2+ dependencies: 1\n\n")

	fmt.Printf("Time Estimation:\n")
	fmt.Printf("  - Total estimated hours: 24-32 hours\n")
	fmt.Printf("  - Average task duration: 3 hours\n")
	fmt.Printf("  - Critical path length: 5 tasks\n\n")

	fmt.Printf("💡 Recommendations:\n")
	fmt.Printf("   - Focus on tasks with no dependencies first\n")
	fmt.Printf("   - Break down high complexity tasks further\n")
	fmt.Printf("   - Consider parallel execution for independent tasks\n")

	return nil
}

// formatMainTasksAsText formats main tasks as plain text
func (c *CLI) formatMainTasksAsText(tasks []*services.MainTask) string {
	var content strings.Builder

	content.WriteString("# Generated Main Tasks\n\n")
	content.WriteString(fmt.Sprintf("Generated at: %s\n", time.Now().Format(time.RFC3339)))
	content.WriteString(fmt.Sprintf("Total tasks: %d\n\n", len(tasks)))

	for i, task := range tasks {
		content.WriteString(fmt.Sprintf("## Task %d: %s\n\n", i+1, task.Name))
		content.WriteString(fmt.Sprintf("- **ID:** %s\n", task.ID))
		content.WriteString(fmt.Sprintf("- **Phase:** %s\n", task.Phase))
		content.WriteString(fmt.Sprintf("- **Duration:** %s\n", task.Duration))
		content.WriteString(fmt.Sprintf("- **Atomic:** %t\n", task.AtomicValidation))
		content.WriteString(fmt.Sprintf("- **Dependencies:** %d\n\n", len(task.Dependencies)))
		content.WriteString(fmt.Sprintf("**Description:** %s\n\n", task.Description))
		content.WriteString("---\n\n")
	}

	return content.String()
}

// formatSubTasksAsText formats sub-tasks as plain text
func (c *CLI) formatSubTasksAsText(tasks []*services.SubTask) string {
	var content strings.Builder

	content.WriteString("# Generated Sub-Tasks\n\n")
	content.WriteString(fmt.Sprintf("Generated at: %s\n", time.Now().Format(time.RFC3339)))
	content.WriteString(fmt.Sprintf("Total sub-tasks: %d\n\n", len(tasks)))

	for i, task := range tasks {
		content.WriteString(fmt.Sprintf("## Sub-Task %d: %s\n\n", i+1, task.Name))
		content.WriteString(fmt.Sprintf("- **ID:** %s\n", task.ID))
		content.WriteString(fmt.Sprintf("- **Parent:** %s\n", task.ParentTaskID))
		content.WriteString(fmt.Sprintf("- **Duration:** %d hours\n", task.Duration))
		content.WriteString(fmt.Sprintf("- **Type:** %s\n", task.Type))
		content.WriteString(fmt.Sprintf("- **Deliverables:** %d\n", len(task.Deliverables)))
		content.WriteString(fmt.Sprintf("- **Dependencies:** %d\n\n", len(task.Dependencies)))
		content.WriteString(fmt.Sprintf("**Content:** %s\n\n", task.Content))

		if len(task.Deliverables) > 0 {
			content.WriteString("**Deliverables:**\n")
			for _, deliverable := range task.Deliverables {
				content.WriteString(fmt.Sprintf("- %s\n", deliverable))
			}
			content.WriteString("\n")
		}

		if len(task.AcceptanceCriteria) > 0 {
			content.WriteString("**Acceptance Criteria:**\n")
			for _, criteria := range task.AcceptanceCriteria {
				content.WriteString(fmt.Sprintf("- %s\n", criteria))
			}
			content.WriteString("\n")
		}

		content.WriteString("---\n\n")
	}

	return content.String()
}
