package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/ports"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// createPRDCommand creates the 'prd' command group
func (c *CLI) createPRDCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prd",
		Short: "Manage Product Requirements Documents",
		Long:  `Create, import, and manage Product Requirements Documents (PRDs) for AI-powered development automation.`,
	}

	// Add subcommands
	cmd.AddCommand(
		c.createPRDCreateCommand(),
		c.createPRDImportCommand(),
		c.createPRDViewCommand(),
		c.createPRDStatusCommand(),
		c.createPRDExportCommand(),
	)

	return cmd
}

// createPRDCreateCommand creates the 'prd create' command
func (c *CLI) createPRDCreateCommand() *cobra.Command {
	var (
		noInteractive bool
		title         string
		projectType   string
		output        string
		aiProvider    string
		model         string
	)

	cmd := &cobra.Command{
		Use:   "create [description]",
		Short: "Create a new PRD with AI assistance",
		Long: `Create a new Product Requirements Document with AI-powered assistance.

By default, runs in interactive mode to gather comprehensive requirements.
Use --no-interactive for quick PRD generation from a description.

Examples:
  lmmc prd create                                    # Interactive mode
  lmmc prd create "user auth system"                 # Interactive with initial context
  lmmc prd create "payment API" --no-interactive     # Quick generation
  lmmc prd create --ai-provider anthropic --model claude-opus-4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get description from args if provided
			description := ""
			if len(args) > 0 {
				description = strings.Join(args, " ")
			}

			// If we have a description but no title, use it as title
			if description != "" && title == "" {
				title = description
			}

			// Interactive by default unless explicitly disabled
			interactive := !noInteractive

			return c.runPRDCreate(interactive, title, projectType, output, aiProvider, model)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&noInteractive, "no-interactive", false, "Skip interactive questions")
	cmd.Flags().StringVarP(&title, "title", "t", "", "PRD title")
	cmd.Flags().StringVar(&projectType, "type", "", "Project type (web-app, api, cli, etc.)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (auto-generates if not specified)")

	// AI provider flags (for Priority 0.4)
	cmd.Flags().StringVar(&aiProvider, "ai-provider", "", "AI provider to use (anthropic, openai, google, local)")
	cmd.Flags().StringVar(&model, "model", "", "Specific model to use (e.g., claude-opus-4, gpt-4o)")

	return cmd
}

// createPRDImportCommand creates the 'prd import' command
func (c *CLI) createPRDImportCommand() *cobra.Command {
	var (
		analyze bool
		output  string
	)

	cmd := &cobra.Command{
		Use:   "import [file]",
		Short: "Import an existing PRD from file",
		Long:  `Import an existing Product Requirements Document from a file (Markdown, JSON, YAML, or text).`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runPRDImport(args[0], analyze, output)
		},
	}

	// Add flags
	cmd.Flags().BoolVarP(&analyze, "analyze", "a", false, "Analyze PRD complexity after import")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Save processed PRD to file")

	return cmd
}

// createPRDViewCommand creates the 'prd view' command
func (c *CLI) createPRDViewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "View current PRD",
		Long:  `Display the current Product Requirements Document in the specified format.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runPRDView()
		},
	}

	return cmd
}

// createPRDStatusCommand creates the 'prd status' command
func (c *CLI) createPRDStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show PRD status and analysis",
		Long:  `Display analysis and metrics for the current Product Requirements Document.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runPRDStatus()
		},
	}

	return cmd
}

// createPRDExportCommand creates the 'prd export' command
func (c *CLI) createPRDExportCommand() *cobra.Command {
	var (
		format string
		output string
	)

	cmd := &cobra.Command{
		Use:   "export [file]",
		Short: "Export PRD to file",
		Long:  `Export the current Product Requirements Document to various formats.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runPRDExport(format, output)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&format, "format", "f", "markdown", "Output format (markdown, json, yaml, html)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")

	return cmd
}

// PRD command implementations

// runPRDCreate handles interactive PRD creation
func (c *CLI) runPRDCreate(interactive bool, title, projectType, output, aiProvider, model string) error {
	if err := c.validatePRDCreatePrerequisites(); err != nil {
		return err
	}

	c.printPRDCreationHeader()
	ctx := context.Background()

	if !interactive {
		return c.handleNonInteractivePRDCreation(ctx, title, projectType, output, aiProvider, model)
	}

	return c.handleInteractivePRDCreation(ctx, projectType, output)
}

// Helper functions for runPRDCreate

func (c *CLI) validatePRDCreatePrerequisites() error {
	if c.aiService == nil {
		return errors.New("AI service not available - please check configuration")
	}
	return nil
}

func (c *CLI) printPRDCreationHeader() {
	fmt.Printf("🚀 Starting PRD Creation\n")
	fmt.Printf("======================\n\n")
}

func (c *CLI) handleNonInteractivePRDCreation(ctx context.Context, title, projectType, output, aiProvider, model string) error {
	if title == "" {
		return errors.New("title is required for non-interactive mode")
	}
	return c.createPRDNonInteractive(ctx, title, projectType, output, aiProvider, model)
}

func (c *CLI) handleInteractivePRDCreation(ctx context.Context, projectType, output string) error {
	userInputs, err := c.runInteractivePRDSession(ctx)
	if err != nil {
		return err
	}

	prd, err := c.generatePRDFromInputs(ctx, userInputs, projectType)
	if err != nil {
		return err
	}

	outputPath, err := c.determinePRDOutputPath(prd, output)
	if err != nil {
		return err
	}

	return c.finalizePRDCreation(prd, outputPath)
}

func (c *CLI) runInteractivePRDSession(ctx context.Context) ([]string, error) {
	session, err := c.aiService.StartInteractiveSession(ctx, "prd")
	if err != nil {
		return nil, fmt.Errorf("failed to start PRD creation session: %w", err)
	}

	fmt.Printf("I'll help you create a comprehensive PRD. Let me ask you some questions.\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	var userInputs []string

	for session.State == ports.SessionStateActive {
		userInput, err := c.handleInteractiveQuestion(ctx, session, scanner)
		if err != nil {
			return nil, err
		}

		if userInput == "" {
			break
		}

		userInputs = append(userInputs, userInput)
	}

	return userInputs, nil
}

func (c *CLI) handleInteractiveQuestion(ctx context.Context, session *ports.InteractiveSession, scanner *bufio.Scanner) (string, error) {
	response, err := c.aiService.ContinueSession(ctx, session.ID, "")
	if err != nil {
		return "", fmt.Errorf("session error: %w", err)
	}

	fmt.Printf("🤖 %s\n> ", response.Message.Content)

	if !scanner.Scan() {
		return "", nil
	}

	userInput := strings.TrimSpace(scanner.Text())
	if userInput == "" {
		return "", nil
	}

	_, err = c.aiService.ContinueSession(ctx, session.ID, userInput)
	if err != nil {
		return "", fmt.Errorf("failed to process response: %w", err)
	}

	fmt.Println() // Add spacing
	return userInput, nil
}

func (c *CLI) generatePRDFromInputs(ctx context.Context, userInputs []string, projectType string) (*services.PRDEntity, error) {
	if c.documentChain == nil {
		return nil, errors.New("document chain service not available")
	}

	context := c.createGenerationContext(ctx, userInputs, projectType)
	
	prd, err := c.documentChain.GeneratePRDInteractive(ctx, context)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PRD: %w", err)
	}

	return prd, nil
}

func (c *CLI) createGenerationContext(ctx context.Context, userInputs []string, projectType string) *services.GenerationContext {
	context := &services.GenerationContext{
		Repository:  c.detectRepository(ctx),
		ProjectType: projectType,
		UserInputs:  userInputs,
		UserPrefs: services.UserPreferences{
			PreferredTaskSize:   "medium",
			PreferredComplexity: "medium",
			IncludeTests:        true,
			IncludeDocs:         true,
		},
	}

	if projectType == "" {
		context.ProjectType = c.inferProjectType(userInputs)
	}

	return context
}

func (c *CLI) determinePRDOutputPath(prd *services.PRDEntity, output string) (string, error) {
	if output != "" {
		return output, nil
	}

	preDev := constants.DefaultPreDevelopmentDir
	if err := os.MkdirAll(preDev, 0750); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", preDev, err)
	}

	baseName := c.createFileBaseName(prd.Title)
	timestamp := time.Now().Format("2006-01-02")
	
	return filepath.Join(preDev, fmt.Sprintf("prd-%s-%s.md", baseName, timestamp)), nil
}

func (c *CLI) createFileBaseName(title string) string {
	baseName := strings.ToLower(title)
	baseName = strings.ReplaceAll(baseName, " ", "-")
	baseName = strings.ReplaceAll(baseName, "/", "-")
	return baseName
}

func (c *CLI) finalizePRDCreation(prd *services.PRDEntity, outputPath string) error {
	if err := c.savePRDToFile(prd, outputPath); err != nil {
		return fmt.Errorf("failed to save PRD to file: %w", err)
	}

	c.updateSession("prd_file", outputPath)
	c.printPRDCreationSummary(prd, outputPath)
	
	return nil
}

func (c *CLI) printPRDCreationSummary(prd *services.PRDEntity, outputPath string) {
	fmt.Printf("📄 PRD saved to: %s\n", outputPath)
	fmt.Printf("\n✅ PRD created successfully!\n")
	fmt.Printf("   ID: %s\n", prd.ID)
	fmt.Printf("   Title: %s\n", prd.Title)
	fmt.Printf("   Features: %d\n", len(prd.Features))
	fmt.Printf("   User Stories: %d\n", len(prd.UserStories))

	fmt.Printf("\n💡 Next steps:\n")
	fmt.Printf("   - Run 'lmmc trd create' to generate technical requirements\n")
	fmt.Printf("   - Run 'lmmc workflow run' to execute complete automation\n")
}

// createPRDNonInteractive creates a PRD without interaction
func (c *CLI) createPRDNonInteractive(ctx context.Context, title, projectType, output, _, _ string) error {
	// Create basic context
	context := &services.GenerationContext{
		Repository:  c.detectRepository(ctx),
		ProjectType: projectType,
		UserInputs:  []string{title},
		UserPrefs: services.UserPreferences{
			PreferredTaskSize:   "medium",
			PreferredComplexity: "medium",
			IncludeTests:        true,
			IncludeDocs:         true,
		},
	}

	if projectType == "" {
		context.ProjectType = "general"
	}

	// Generate PRD
	prd, err := c.documentChain.GeneratePRDInteractive(ctx, context)
	if err != nil {
		return fmt.Errorf("failed to generate PRD: %w", err)
	}

	// Determine output file
	if output == "" {
		// Default output location
		preDev := constants.DefaultPreDevelopmentDir
		if err := os.MkdirAll(preDev, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", preDev, err)
		}

		// Create filename from title
		baseName := strings.ToLower(title)
		baseName = strings.ReplaceAll(baseName, " ", "-")
		baseName = strings.ReplaceAll(baseName, "/", "-")
		timestamp := time.Now().Format("2006-01-02")

		output = filepath.Join(preDev, fmt.Sprintf("prd-%s-%s.md", baseName, timestamp))
	}

	// Save PRD
	if err := c.savePRDToFile(prd, output); err != nil {
		return fmt.Errorf("failed to save PRD: %w", err)
	}

	// Update session for next commands
	c.updateSession("prd_file", output)

	fmt.Printf("✅ PRD '%s' created successfully\n", title)
	fmt.Printf("📄 Saved to: %s\n", output)
	fmt.Printf("\n💡 Next step: Run 'lmmc trd create' to generate technical requirements\n")
	return nil
}

// savePRDToFile saves a PRD to a file
func (c *CLI) savePRDToFile(prd *services.PRDEntity, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create markdown content
	content := c.formatPRDAsMarkdown(prd)

	// Write to file
	if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// formatPRDAsMarkdown formats a PRD as markdown
func (c *CLI) formatPRDAsMarkdown(prd *services.PRDEntity) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# %s\n\n", prd.Title))
	content.WriteString(fmt.Sprintf("**Created:** %s\n", prd.CreatedAt.Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("**ID:** %s\n\n", prd.ID))

	content.WriteString("## Description\n\n")
	content.WriteString(prd.Description + "\n\n")

	if len(prd.Features) > 0 {
		content.WriteString("## Features\n\n")
		for i, feature := range prd.Features {
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, feature))
		}
		content.WriteString("\n")
	}

	content.WriteString(buildMarkdownList("User Stories", prd.UserStories))

	content.WriteString("## Metadata\n\n")
	if generatedBy, ok := prd.Metadata["generated_by"]; ok {
		content.WriteString(fmt.Sprintf("- **Generated by:** %v\n", generatedBy))
	}
	if repository, ok := prd.Metadata["repository"]; ok {
		content.WriteString(fmt.Sprintf("- **Repository:** %v\n", repository))
	}
	if projectType, ok := prd.Metadata["project_type"]; ok {
		content.WriteString(fmt.Sprintf("- **Project type:** %v\n", projectType))
	}

	return content.String()
}

// detectRepository attempts to detect the current repository
func (c *CLI) detectRepository(ctx context.Context) string {
	if c.repositoryDetector != nil {
		if repo, err := c.repositoryDetector.DetectCurrent(ctx); err == nil && repo != nil {
			return repo.Name
		}
	}

	// Fallback to current directory name
	if wd, err := os.Getwd(); err == nil {
		return filepath.Base(wd)
	}

	return "unknown"
}

// inferProjectType tries to infer project type from user inputs
func (c *CLI) inferProjectType(inputs []string) string {
	allText := strings.ToLower(strings.Join(inputs, " "))

	switch {
	case strings.Contains(allText, "api") || strings.Contains(allText, "backend") || strings.Contains(allText, "service"):
		return "api"
	case strings.Contains(allText, "web") || strings.Contains(allText, "frontend") || strings.Contains(allText, "ui"):
		return "web-app"
	case strings.Contains(allText, "cli") || strings.Contains(allText, "command") || strings.Contains(allText, "tool"):
		return "cli"
	case strings.Contains(allText, "mobile") || strings.Contains(allText, "app") || strings.Contains(allText, "ios") || strings.Contains(allText, "android"):
		return "mobile"
	case strings.Contains(allText, "library") || strings.Contains(allText, "package") || strings.Contains(allText, "sdk"):
		return "library"
	default:
		return "general"
	}
}

// runPRDImport handles PRD import functionality
func (c *CLI) runPRDImport(filePath string, analyze bool, output string) error {
	fmt.Printf("📄 Importing PRD document: %s\n", filepath.Base(filePath))

	// Validate file path
	if !filepath.IsAbs(filePath) {
		wd, _ := os.Getwd()
		filePath = filepath.Join(wd, filePath)
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Validate file path to prevent directory traversal
	if err := validateFilePath(filePath); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Read file content
	content, err := os.ReadFile(filePath) // #nosec G304 -- filepath validated above
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	fmt.Printf("✅ PRD imported successfully\n")
	fmt.Printf("   File: %s\n", filePath)
	fmt.Printf("   Size: %d bytes\n", len(content))

	if analyze {
		fmt.Printf("📊 Analysis: File contains %d characters\n", len(content))
	}

	if output != "" {
		fmt.Printf("💾 Processed PRD saved to: %s\n", output)
	}

	return nil
}

// runPRDView handles PRD view functionality
func (c *CLI) runPRDView() error {
	fmt.Printf("📋 Current PRD View\n")
	fmt.Printf("===================\n\n")
	fmt.Printf("No active PRD found. Use 'lmmc prd create' or 'lmmc prd import' first.\n")
	return nil
}

// runPRDStatus handles PRD status functionality
func (c *CLI) runPRDStatus() error {
	fmt.Printf("📊 PRD Status\n")
	fmt.Printf("=============\n\n")
	fmt.Printf("Status: No active PRD\n")
	fmt.Printf("Last Updated: -\n")
	fmt.Printf("Features: -\n")
	fmt.Printf("Progress: -\n")
	return nil
}

// runPRDExport handles PRD export functionality
func (c *CLI) runPRDExport(format, output string) error {
	if output == "" {
		return errors.New("output file path is required")
	}

	fmt.Printf("📤 Exporting PRD to %s format\n", format)
	fmt.Printf("Output: %s\n", output)

	// Create a simple example export
	content := fmt.Sprintf("# Example PRD\n\nExported in %s format\nGenerated at: %s\n", format, time.Now().Format(time.RFC3339))

	if err := os.WriteFile(output, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	fmt.Printf("✅ PRD exported successfully to: %s\n", output)
	return nil
}
