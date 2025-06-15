package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// createSubtasksCommand creates the 'subtasks' command group
func (c *CLI) createSubtasksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subtasks",
		Short: "Manage sub-tasks for main project tasks",
		Long:  `Generate and manage detailed sub-tasks for main project tasks.`,
	}

	// Add subcommands
	cmd.AddCommand(
		c.createSubtasksGenerateCommand(),
	)

	return cmd
}

// createSubtasksGenerateCommand creates the 'subtasks generate' command
func (c *CLI) createSubtasksGenerateCommand() *cobra.Command {
	var (
		fromTask string
		output   string
		session  bool
	)

	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate sub-tasks for a main task",
		Long:    `Generate detailed, implementable sub-tasks for a specific main task.`,
		Aliases: []string{"sub"}, // Backward compatibility with 'taskgen sub'
		RunE: func(cmd *cobra.Command, args []string) error {
			// Support both flag and positional argument
			if fromTask == "" && len(args) > 0 {
				fromTask = args[0]
			}
			return c.runSubtasksGenerate(fromTask, output, session)
		},
	}

	// Add flags with consistent naming
	cmd.Flags().StringVar(&fromTask, "from-task", "", "Main task ID to generate sub-tasks for")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file for generated sub-tasks")
	cmd.Flags().BoolVar(&session, "session", true, "Use session management for context")

	// Legacy flag for backward compatibility
	cmd.Flags().StringVar(&fromTask, "task", "", "Main task ID (deprecated, use --from-task)")
	markFlagHidden(cmd, "task")

	return cmd
}

// runSubtasksGenerate handles sub-task generation
func (c *CLI) runSubtasksGenerate(fromTask, output string, useSession bool) error {
	if c.documentChain == nil {
		return errors.New("document chain service not available")
	}

	if fromTask == "" {
		// Try to get from session or suggest
		if useSession {
			fromTask = c.getSessionValue("current_task")
		}
		if fromTask == "" {
			return errors.New("task ID is required. Use --from-task <task-id> or provide as argument")
		}
	}

	fmt.Printf("🔄 Generating Sub-tasks for: %s\n", fromTask)
	fmt.Printf("====================================\n\n")

	ctx := context.Background()

	// Load the main task
	mainTask := c.loadMainTask(fromTask)

	fmt.Printf("Main Task: %s\n", mainTask.Name)
	fmt.Printf("Phase: %s | Duration: %s\n\n", mainTask.Phase, mainTask.Duration)

	// Generate sub-tasks
	subTasks, err := c.documentChain.GenerateSubTasksFromMain(ctx, mainTask)
	if err != nil {
		return fmt.Errorf("failed to generate sub-tasks: %w", err)
	}

	// Determine output file
	if output == "" && useSession {
		// Default output location
		output = c.getDefaultSubtasksOutputPath(fromTask)
	}

	// Save sub-tasks if output specified
	if output != "" {
		content := c.formatSubTasksAsMarkdown(mainTask, subTasks)
		if err := c.saveToFile(output, content); err != nil {
			return fmt.Errorf("failed to save sub-tasks: %w", err)
		}
		fmt.Printf("📄 Sub-tasks saved to: %s\n\n", output)

		// Update session
		if useSession {
			c.updateSession("subtasks_file", output)
		}
	}

	// Display results
	fmt.Printf("✅ Generated %d sub-tasks successfully!\n\n", len(subTasks))

	totalHours := 0
	for i, task := range subTasks {
		fmt.Printf("%d. %s (%dh)\n", i+1, task.Name, task.Duration)
		fmt.Printf("   Type: %s | Deliverables: %d\n", task.Type, len(task.Deliverables))
		if task.Content != "" {
			fmt.Printf("   Content: %s\n", task.Content)
		}
		fmt.Println()
		totalHours += task.Duration
	}

	fmt.Printf("📊 Summary:\n")
	fmt.Printf("   - Total sub-tasks: %d\n", len(subTasks))
	fmt.Printf("   - Total effort: %d hours\n", totalHours)
	fmt.Printf("   - Average per sub-task: %.1f hours\n\n", float64(totalHours)/float64(len(subTasks)))

	fmt.Printf("💡 Next steps:\n")
	fmt.Printf("   - Review generated sub-tasks for completeness\n")
	fmt.Printf("   - Create task items with 'lmmc add --from-subtask ST-001'\n")
	fmt.Printf("   - Run 'lmmc workflow continue' to proceed\n")

	return nil
}

// Helper methods

func (c *CLI) loadMainTask(taskID string) *services.MainTask {
	// First try to load from tasks file
	tasksFile := c.detectLatestTasksFile()
	if tasksFile != "" {
		// TODO: Implement actual task loading from file
		// For now, check if it's a known format
		_ = tasksFile // Acknowledge we're not using this yet
	}

	// For now, return a mock task
	// TODO: Implement actual task loading
	return &services.MainTask{
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
}

func (c *CLI) getDefaultSubtasksOutputPath(taskID string) string {
	// Create standard output path
	preDev := constants.DefaultPreDevelopmentDir + "/tasks"
	if err := os.MkdirAll(preDev, 0750); err != nil {
		// Log the error but continue with the path
		c.logger.Warn("failed to create directory", "path", preDev, "error", err)
	}

	timestamp := time.Now().Format("2006-01-02")
	return filepath.Join(preDev, fmt.Sprintf("subtasks-%s-%s.md", taskID, timestamp))
}

func (c *CLI) formatSubTasksAsMarkdown(mainTask *services.MainTask, subTasks []*services.SubTask) string {
	var content strings.Builder

	c.writeMarkdownHeader(&content, mainTask, subTasks)
	c.writeMarkdownSummary(&content, mainTask, subTasks)
	c.writeMarkdownOverviewTable(&content, subTasks)
	c.writeMarkdownDetailedDescriptions(&content, subTasks)
	c.writeMarkdownImplementationOrder(&content, subTasks)

	return content.String()
}

// Helper functions for formatSubTasksAsMarkdown

func (c *CLI) writeMarkdownHeader(content *strings.Builder, mainTask *services.MainTask, subTasks []*services.SubTask) {
	content.WriteString(fmt.Sprintf("# Sub-tasks for: %s\n\n", mainTask.Name))
	content.WriteString(fmt.Sprintf("**Main Task ID:** %s\n", mainTask.ID))
	content.WriteString(fmt.Sprintf("**Generated at:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("**Total sub-tasks:** %d\n\n", len(subTasks)))
}

func (c *CLI) writeMarkdownSummary(content *strings.Builder, mainTask *services.MainTask, subTasks []*services.SubTask) {
	totalHours := c.calculateTotalHours(subTasks)

	content.WriteString("## Summary\n\n")
	content.WriteString(fmt.Sprintf("- **Main Task Phase:** %s\n", mainTask.Phase))
	content.WriteString(fmt.Sprintf("- **Main Task Duration:** %s\n", mainTask.Duration))
	content.WriteString(fmt.Sprintf("- **Total Sub-task Hours:** %d hours\n", totalHours))
	content.WriteString(fmt.Sprintf("- **Number of Sub-tasks:** %d\n\n", len(subTasks)))
}

func (c *CLI) calculateTotalHours(subTasks []*services.SubTask) int {
	totalHours := 0
	for _, task := range subTasks {
		totalHours += task.Duration
	}
	return totalHours
}

func (c *CLI) writeMarkdownOverviewTable(content *strings.Builder, subTasks []*services.SubTask) {
	content.WriteString("## Sub-tasks Overview\n\n")
	content.WriteString("| ID | Sub-task Name | Type | Duration | Deliverables |\n")
	content.WriteString("|----|---------------|------|----------|-------------|\n")

	for _, task := range subTasks {
		content.WriteString(fmt.Sprintf("| %s | %s | %s | %dh | %d |\n",
			task.ID, task.Name, task.Type, task.Duration, len(task.Deliverables)))
	}
	content.WriteString("\n")
}

func (c *CLI) writeMarkdownDetailedDescriptions(content *strings.Builder, subTasks []*services.SubTask) {
	content.WriteString("## Detailed Sub-task Descriptions\n\n")

	for i, task := range subTasks {
		c.writeSubTaskDetails(content, task, i+1)
	}
}

func (c *CLI) writeSubTaskDetails(content *strings.Builder, task *services.SubTask, index int) {
	content.WriteString(fmt.Sprintf("### %d. %s\n\n", index, task.Name))
	content.WriteString(fmt.Sprintf("- **ID:** %s\n", task.ID))
	content.WriteString(fmt.Sprintf("- **Parent Task:** %s\n", task.ParentTaskID))
	content.WriteString(fmt.Sprintf("- **Type:** %s\n", task.Type))
	content.WriteString(fmt.Sprintf("- **Duration:** %d hours\n", task.Duration))

	c.writeSubTaskOptionalFields(content, task)
	content.WriteString("\n---\n\n")
}

func (c *CLI) writeSubTaskOptionalFields(content *strings.Builder, task *services.SubTask) {
	if task.Content != "" {
		content.WriteString(fmt.Sprintf("\n**Description:**\n%s\n", task.Content))
	}

	c.writeStringSliceSection(content, "Deliverables", task.Deliverables)
	c.writeStringSliceSection(content, "Acceptance Criteria", task.AcceptanceCriteria)
	c.writeStringSliceSection(content, "Dependencies", task.Dependencies)
}

func (c *CLI) writeStringSliceSection(content *strings.Builder, title string, items []string) {
	if len(items) > 0 {
		content.WriteString(fmt.Sprintf("\n**%s:**\n", title))
		for _, item := range items {
			content.WriteString(fmt.Sprintf("- %s\n", item))
		}
	}
}

func (c *CLI) writeMarkdownImplementationOrder(content *strings.Builder, subTasks []*services.SubTask) {
	content.WriteString("## Suggested Implementation Order\n\n")
	content.WriteString("Based on dependencies and complexity:\n\n")

	noDeps, withDeps := c.groupTasksByDependencies(subTasks)
	c.writeImplementationPhases(content, noDeps, withDeps)
}

func (c *CLI) groupTasksByDependencies(subTasks []*services.SubTask) ([]*services.SubTask, []*services.SubTask) {
	noDeps := make([]*services.SubTask, 0)
	withDeps := make([]*services.SubTask, 0)

	for _, task := range subTasks {
		if len(task.Dependencies) == 0 {
			noDeps = append(noDeps, task)
		} else {
			withDeps = append(withDeps, task)
		}
	}

	return noDeps, withDeps
}

func (c *CLI) writeImplementationPhases(content *strings.Builder, noDeps, withDeps []*services.SubTask) {
	phase := 1

	if len(noDeps) > 0 {
		content.WriteString(fmt.Sprintf("**Phase %d (No Dependencies):**\n", phase))
		for _, task := range noDeps {
			content.WriteString(fmt.Sprintf("- %s: %s\n", task.ID, task.Name))
		}
		content.WriteString("\n")
		phase++
	}

	if len(withDeps) > 0 {
		content.WriteString(fmt.Sprintf("**Phase %d (With Dependencies):**\n", phase))
		for _, task := range withDeps {
			content.WriteString(fmt.Sprintf("- %s: %s (depends on: %s)\n",
				task.ID, task.Name, strings.Join(task.Dependencies, ", ")))
		}
	}
}
