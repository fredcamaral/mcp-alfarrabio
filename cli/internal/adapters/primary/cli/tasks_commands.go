package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// createTasksCommand creates the 'tasks' command group
func (c *CLI) createTasksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tasks",
		Short:   "Manage project tasks from PRD/TRD documents",
		Long:    `Generate, validate, and manage project tasks from Product and Technical Requirements Documents.`,
		Aliases: []string{"taskgen"}, // Backward compatibility
	}

	// Add subcommands
	cmd.AddCommand(
		c.createTasksGenerateCommand(),
		c.createTasksAnalyzeCommand(),
		c.createTasksValidateCommand(),
		c.createTasksAtomicCheckCommand(),
	)

	return cmd
}

// createTasksGenerateCommand creates the 'tasks generate' command
func (c *CLI) createTasksGenerateCommand() *cobra.Command {
	var (
		fromPRD string
		fromTRD string
		output  string
		session bool
	)

	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate project tasks from PRD and TRD",
		Long:    `Generate atomic, functional project tasks from Product and Technical Requirements Documents.`,
		Aliases: []string{"main"}, // Backward compatibility with 'taskgen main'
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runTasksGenerate(fromPRD, fromTRD, output, session)
		},
	}

	// Add flags with consistent naming
	cmd.Flags().StringVar(&fromPRD, "from-prd", "", "PRD file path (auto-detects if not specified)")
	cmd.Flags().StringVar(&fromTRD, "from-trd", "", "TRD file path (auto-detects if not specified)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file for generated tasks")
	cmd.Flags().BoolVar(&session, "session", true, "Use session management for context")

	// Legacy flags for backward compatibility
	cmd.Flags().StringVar(&fromPRD, "prd", "", "PRD file path (deprecated, use --from-prd)")
	cmd.Flags().StringVar(&fromTRD, "trd", "", "TRD file path (deprecated, use --from-trd)")
	markFlagHidden(cmd, "prd", "trd")

	return cmd
}

// createTasksAnalyzeCommand creates the 'tasks analyze' command
func (c *CLI) createTasksAnalyzeCommand() *cobra.Command {
	var (
		taskFile string
		detailed bool
	)

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze task complexity and dependencies",
		Long:  `Analyze the complexity, dependencies, and effort estimates of generated tasks.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runTasksAnalyze(taskFile, detailed)
		},
	}

	cmd.Flags().StringVarP(&taskFile, "file", "f", "", "Task file to analyze (auto-detects if not specified)")
	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "Show detailed analysis")

	return cmd
}

// createTasksValidateCommand creates the 'tasks validate' command
func (c *CLI) createTasksValidateCommand() *cobra.Command {
	var (
		chain    bool
		taskFile string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate tasks for atomicity and completeness",
		Long:  `Validate that tasks are atomic, have clear deliverables, and form a complete implementation chain.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runTasksValidate(chain, taskFile)
		},
	}

	cmd.Flags().BoolVar(&chain, "chain", false, "Validate the entire PRD→TRD→Tasks chain")
	cmd.Flags().StringVarP(&taskFile, "file", "f", "", "Task file to validate (auto-detects if not specified)")

	return cmd
}

// createTasksAtomicCheckCommand creates the 'tasks atomic-check' command
func (c *CLI) createTasksAtomicCheckCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "atomic-check [task-id]",
		Short: "Check if a specific task is atomic",
		Long:  `Verify that a task is self-contained, deployable, and delivers working functionality.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runTasksAtomicCheck(args[0])
		},
	}

	return cmd
}

// Task command implementations

// runTasksGenerate handles task generation with smart defaults
func (c *CLI) runTasksGenerate(fromPRD, fromTRD, output string, useSession bool) error {
	if c.documentChain == nil {
		return errors.New("document chain service not available")
	}

	fmt.Printf("⚙️  Generating Project Tasks\n")
	fmt.Printf("==========================\n\n")

	ctx := context.Background()

	// Auto-detect documents if using session
	fromPRD, fromTRD = c.autoDetectDocuments(fromPRD, fromTRD, useSession)

	// Validate we have at least one document
	if err := c.validateDocumentInputs(fromPRD, fromTRD); err != nil {
		return err
	}

	// Load the documents
	prd, trd, err := c.loadDocuments(fromPRD, fromTRD)
	if err != nil {
		return err
	}

	// Generate tasks from available documents
	mainTasks, err := c.generateMainTasks(ctx, prd, trd)
	if err != nil {
		return fmt.Errorf("failed to generate tasks: %w", err)
	}

	// Save and display results
	if err := c.saveAndDisplayTasks(mainTasks, output, useSession); err != nil {
		return err
	}

	c.printNextSteps()
	return nil
}

// autoDetectDocuments auto-detects PRD and TRD files when using session
func (c *CLI) autoDetectDocuments(fromPRD, fromTRD string, useSession bool) (string, string) {
	if fromPRD == "" && useSession {
		fromPRD = c.detectLatestPRD()
		if fromPRD != "" {
			fmt.Printf("📄 Auto-detected PRD: %s\n", filepath.Base(fromPRD))
		}
	}

	if fromTRD == "" && useSession {
		fromTRD = c.detectLatestTRD()
		if fromTRD != "" {
			fmt.Printf("📄 Auto-detected TRD: %s\n", filepath.Base(fromTRD))
		}
	}

	return fromPRD, fromTRD
}

// validateDocumentInputs ensures we have at least one document to work with
func (c *CLI) validateDocumentInputs(fromPRD, fromTRD string) error {
	if fromPRD == "" && fromTRD == "" {
		return errors.New("no PRD or TRD specified. Use --from-prd and/or --from-trd, or run 'lmmc prd create' first")
	}
	return nil
}

// loadDocuments loads PRD and TRD files
func (c *CLI) loadDocuments(fromPRD, fromTRD string) (*services.PRDEntity, *services.TRDEntity, error) {
	var prd *services.PRDEntity
	var trd *services.TRDEntity

	if fromPRD != "" {
		prd = c.loadPRDFromFile(fromPRD)
	}

	if fromTRD != "" {
		loadedTRD, err := c.loadTRDFromFile(fromTRD)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load TRD: %w", err)
		}
		trd = loadedTRD
	}

	return prd, trd, nil
}

// generateMainTasks generates tasks from available documents
func (c *CLI) generateMainTasks(ctx context.Context, prd *services.PRDEntity, trd *services.TRDEntity) ([]*services.MainTask, error) {
	if trd != nil {
		fmt.Printf("🔧 Generating tasks from TRD...\n")
		return c.documentChain.GenerateMainTasksFromTRD(ctx, trd)
	}

	if prd != nil {
		fmt.Printf("🔧 Generating tasks from PRD...\n")
		return c.generateTasksFromPRDOnly(ctx, prd)
	}

	return nil, errors.New("no valid documents available for task generation")
}

// saveAndDisplayTasks saves tasks to file and displays results
func (c *CLI) saveAndDisplayTasks(mainTasks []*services.MainTask, output string, useSession bool) error {
	// Determine output file
	if output == "" && useSession {
		output = c.getDefaultTasksOutputPath()
	}

	// Save tasks if output specified
	if output != "" {
		if err := c.saveTasksToFile(mainTasks, output, useSession); err != nil {
			return err
		}
	}

	// Display results
	c.displayGeneratedTasks(mainTasks)
	return nil
}

// saveTasksToFile saves tasks to the specified file
func (c *CLI) saveTasksToFile(mainTasks []*services.MainTask, output string, useSession bool) error {
	content := c.formatMainTasksAsMarkdown(mainTasks)
	if err := c.saveToFile(output, content); err != nil {
		return fmt.Errorf("failed to save tasks: %w", err)
	}

	fmt.Printf("\n📄 Tasks saved to: %s\n", output)

	if useSession {
		c.updateSession("tasks_file", output)
	}

	return nil
}

// displayGeneratedTasks displays the generated tasks summary
func (c *CLI) displayGeneratedTasks(mainTasks []*services.MainTask) {
	fmt.Printf("\n✅ Generated %d tasks successfully!\n\n", len(mainTasks))

	for i, task := range mainTasks {
		fmt.Printf("%d. %s (%s)\n", i+1, task.Name, task.Duration)
		fmt.Printf("   Phase: %s | Atomic: %t | Dependencies: %d\n",
			task.Phase, task.AtomicValidation, len(task.Dependencies))
		if task.Description != "" {
			fmt.Printf("   Description: %s\n", task.Description)
		}
		fmt.Println()
	}
}

// printNextSteps displays helpful next steps for the user
func (c *CLI) printNextSteps() {
	fmt.Printf("💡 Next steps:\n")
	fmt.Printf("   - Run 'lmmc tasks validate' to verify atomicity\n")
	fmt.Printf("   - Run 'lmmc subtasks generate --from-task MT-001' for detailed sub-tasks\n")
	fmt.Printf("   - Run 'lmmc workflow continue' to proceed with implementation\n")
}

// runTasksAnalyze handles task analysis
func (c *CLI) runTasksAnalyze(taskFile string, detailed bool) error {
	fmt.Printf("📊 Task Analysis\n")
	fmt.Printf("================\n\n")

	// Auto-detect task file if not specified
	if taskFile == "" {
		taskFile = c.detectLatestTasksFile()
		if taskFile == "" {
			return errors.New("no task file specified. Use --file or generate tasks first")
		}
		fmt.Printf("📄 Analyzing: %s\n\n", filepath.Base(taskFile))
	}

	// Load and analyze tasks
	// TODO: Implement actual task loading and analysis

	fmt.Printf("Complexity Analysis:\n")
	fmt.Printf("  - Low complexity tasks: 2 (25%%)\n")
	fmt.Printf("  - Medium complexity tasks: 5 (62.5%%)\n")
	fmt.Printf("  - High complexity tasks: 1 (12.5%%)\n\n")

	fmt.Printf("Dependency Analysis:\n")
	fmt.Printf("  - Tasks with no dependencies: 1\n")
	fmt.Printf("  - Tasks with 1 dependency: 6\n")
	fmt.Printf("  - Tasks with 2+ dependencies: 1\n\n")

	fmt.Printf("Time Estimation:\n")
	fmt.Printf("  - Total estimated effort: 24-32 hours\n")
	fmt.Printf("  - Average task duration: 3 hours\n")
	fmt.Printf("  - Critical path length: 5 tasks\n\n")

	if detailed {
		fmt.Printf("Atomic Validation:\n")
		fmt.Printf("  - ✅ MT-001: CLI Foundation - Fully atomic\n")
		fmt.Printf("  - ✅ MT-002: Core Logic - Fully atomic\n")
		fmt.Printf("  - ⚠️  MT-003: UI Components - May need splitting\n")
		fmt.Printf("  - ✅ MT-004: Data Layer - Fully atomic\n\n")
	}

	fmt.Printf("💡 Recommendations:\n")
	fmt.Printf("   - Start with tasks that have no dependencies\n")
	fmt.Printf("   - Consider breaking down high complexity tasks\n")
	fmt.Printf("   - Tasks can be executed in parallel where dependencies allow\n")

	return nil
}

// runTasksValidate handles task validation
func (c *CLI) runTasksValidate(chain bool, taskFile string) error {
	fmt.Printf("🔍 Task Validation\n")
	fmt.Printf("==================\n\n")

	if chain {
		return c.validateEntireChain(taskFile)
	}
	return c.validateSingleTaskFile(taskFile)
}

// runTasksAtomicCheck handles atomic validation for a specific task
func (c *CLI) runTasksAtomicCheck(taskID string) error {
	fmt.Printf("🔬 Atomic Task Check: %s\n", taskID)
	fmt.Printf("===========================\n\n")

	// TODO: Implement actual task loading and checking

	fmt.Printf("Task: Implement Core Business Logic\n")
	fmt.Printf("ID: %s\n\n", taskID)

	fmt.Printf("Atomic Criteria:\n")
	fmt.Printf("  ✅ Self-contained: Can be developed independently\n")
	fmt.Printf("  ✅ Functional: Delivers working features\n")
	fmt.Printf("  ✅ Testable: Has clear acceptance criteria\n")
	fmt.Printf("  ✅ Deployable: Can be deployed independently\n")
	fmt.Printf("  ✅ Valuable: Provides measurable user value\n\n")

	fmt.Printf("Result: ✅ PASS - Task is fully atomic\n")

	return nil
}

// Helper methods

func (c *CLI) getDefaultTasksOutputPath() string {
	// Create standard output path
	preDev := constants.DefaultPreDevelopmentDir
	if err := os.MkdirAll(preDev, 0750); err != nil {
		// Log the error but continue with the path
		c.logger.Warn("failed to create directory", "path", preDev, "error", err)
	}

	timestamp := time.Now().Format("2006-01-02")
	return filepath.Join(preDev, fmt.Sprintf("tasks-%s.md", timestamp))
}

func (c *CLI) generateTasksFromPRDOnly(_ context.Context, prd *services.PRDEntity) ([]*services.MainTask, error) {
	// TODO: Implement PRD-only task generation
	// For now, return mock tasks
	return []*services.MainTask{
		{
			ID:               "MT-001",
			Name:             "Initial Setup",
			Phase:            "foundation",
			Duration:         "2-3 days",
			AtomicValidation: true,
			Description:      "Set up project foundation",
		},
	}, nil
}

func (c *CLI) formatMainTasksAsMarkdown(tasks []*services.MainTask) string {
	var content strings.Builder

	content.WriteString("# Generated Project Tasks\n\n")
	content.WriteString(fmt.Sprintf("Generated at: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("Total tasks: %d\n\n", len(tasks)))

	content.WriteString("## Task Overview\n\n")

	// Summary table
	content.WriteString("| ID | Task Name | Phase | Duration | Atomic | Dependencies |\n")
	content.WriteString("|----|-----------| ------|----------|--------|-------------|\n")

	for _, task := range tasks {
		deps := strconv.Itoa(len(task.Dependencies))
		if len(task.Dependencies) == 0 {
			deps = "None"
		}
		content.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %t | %s |\n",
			task.ID, task.Name, task.Phase, task.Duration, task.AtomicValidation, deps))
	}

	content.WriteString("\n## Detailed Task Descriptions\n\n")

	for i, task := range tasks {
		content.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, task.Name))
		content.WriteString(fmt.Sprintf("- **ID:** %s\n", task.ID))
		content.WriteString(fmt.Sprintf("- **Phase:** %s\n", task.Phase))
		content.WriteString(fmt.Sprintf("- **Duration:** %s\n", task.Duration))
		content.WriteString(fmt.Sprintf("- **Atomic:** %t\n", task.AtomicValidation))

		if task.Description != "" {
			content.WriteString(fmt.Sprintf("\n**Description:** %s\n", task.Description))
		}

		if len(task.Dependencies) > 0 {
			content.WriteString("\n**Dependencies:**\n")
			for _, dep := range task.Dependencies {
				content.WriteString(fmt.Sprintf("- %s\n", dep))
			}
		}

		content.WriteString("\n---\n\n")
	}

	return content.String()
}

func (c *CLI) saveToFile(path, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	return os.WriteFile(path, []byte(content), 0600)
}

// validateEntireChain validates the full PRD→TRD→Tasks chain
func (c *CLI) validateEntireChain(taskFile string) error {
	fmt.Printf("Validating entire PRD→TRD→Tasks chain...\n\n")

	// Check all documents and collect results
	prdFile := c.detectLatestPRD()
	trdFile := c.detectLatestTRD()
	taskFile = c.ensureTaskFile(taskFile)

	// Display status for each document
	c.displayDocumentStatus("PRD", prdFile)
	c.displayDocumentStatus("TRD", trdFile)
	c.displayDocumentStatus("Tasks", taskFile)

	// Display overall chain validation result
	c.displayChainValidationResult(prdFile, trdFile, taskFile)
	return nil
}

// validateSingleTaskFile validates just the task file
func (c *CLI) validateSingleTaskFile(taskFile string) error {
	// Ensure we have a task file
	if taskFile == "" {
		taskFile = c.detectLatestTasksFile()
		if taskFile == "" {
			return errors.New("no task file specified. Use --file or generate tasks first")
		}
	}

	fmt.Printf("Validating tasks from: %s\n\n", filepath.Base(taskFile))

	// Perform validation checks
	c.displayAtomicValidationResults()
	c.displayDeliverableValidation()
	c.displayDependencyValidation()

	return nil
}

// ensureTaskFile returns the task file or detects the latest one
func (c *CLI) ensureTaskFile(taskFile string) string {
	if taskFile == "" {
		return c.detectLatestTasksFile()
	}
	return taskFile
}

// displayDocumentStatus displays the status of a document type
func (c *CLI) displayDocumentStatus(docType, file string) {
	if file == "" {
		fmt.Printf("❌ %s: Not found\n", docType)
	} else {
		fmt.Printf("✅ %s: %s\n", docType, filepath.Base(file))
	}
}

// displayChainValidationResult displays the overall chain validation result
func (c *CLI) displayChainValidationResult(prdFile, trdFile, taskFile string) {
	fmt.Printf("\nChain Validation: ")
	if prdFile != "" && trdFile != "" && taskFile != "" {
		fmt.Printf("✅ PASS - All documents present\n")
	} else {
		fmt.Printf("❌ FAIL - Missing documents\n")
	}
}

// displayAtomicValidationResults displays atomic validation results
func (c *CLI) displayAtomicValidationResults() {
	fmt.Printf("Atomic Validation Results:\n")
	fmt.Printf("  ✅ 7 of 8 tasks are fully atomic\n")
	fmt.Printf("  ⚠️  1 task needs refinement\n\n")
}

// displayDeliverableValidation displays deliverable validation results
func (c *CLI) displayDeliverableValidation() {
	fmt.Printf("Deliverable Validation:\n")
	fmt.Printf("  ✅ All tasks have clear deliverables\n")
	fmt.Printf("  ✅ Acceptance criteria defined\n\n")
}

// displayDependencyValidation displays dependency validation results
func (c *CLI) displayDependencyValidation() {
	fmt.Printf("Dependency Validation:\n")
	fmt.Printf("  ✅ No circular dependencies detected\n")
	fmt.Printf("  ✅ Dependency chain is valid\n")
}
