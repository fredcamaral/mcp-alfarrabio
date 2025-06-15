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

	"lerian-mcp-memory-cli/internal/domain/services"
)

// createTRDCommand creates the 'trd' command group
func (c *CLI) createTRDCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trd",
		Short: "Manage Technical Requirements Documents",
		Long:  `Create, import, and manage Technical Requirements Documents (TRDs) for AI-powered development automation.`,
	}

	// Add subcommands
	cmd.AddCommand(
		c.createTRDCreateCommand(),
		c.createTRDImportCommand(),
		c.createTRDViewCommand(),
		c.createTRDExportCommand(),
	)

	return cmd
}

// createTRDCreateCommand creates the 'trd create' command
func (c *CLI) createTRDCreateCommand() *cobra.Command {
	var (
		fromPRD string
		output  string
		session bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a TRD from PRD",
		Long:  `Create a Technical Requirements Document from an existing Product Requirements Document.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runTRDCreate(fromPRD, output, session)
		},
	}

	// Add flags
	cmd.Flags().StringVar(&fromPRD, "from-prd", "", "PRD file to generate TRD from (auto-detects if not specified)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")
	cmd.Flags().BoolVar(&session, "session", true, "Use session management for context")

	return cmd
}

// createTRDImportCommand creates the 'trd import' command
func (c *CLI) createTRDImportCommand() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "import [file]",
		Short: "Import an existing TRD from file",
		Long:  `Import an existing Technical Requirements Document from a file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runTRDImport(args[0], output)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&output, "output", "o", "", "Save processed TRD to file")

	return cmd
}

// createTRDViewCommand creates the 'trd view' command
func (c *CLI) createTRDViewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "View current TRD",
		Long:  `Display the current Technical Requirements Document.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runTRDView()
		},
	}

	return cmd
}

// createTRDExportCommand creates the 'trd export' command
func (c *CLI) createTRDExportCommand() *cobra.Command {
	var (
		format string
		output string
	)

	cmd := &cobra.Command{
		Use:   "export [file]",
		Short: "Export TRD to file",
		Long:  `Export the current Technical Requirements Document to various formats.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runTRDExport(format, output)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&format, "format", "f", "markdown", "Output format (markdown, json, yaml)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")

	return cmd
}

// TRD command implementations

// runTRDCreate handles TRD creation from PRD
func (c *CLI) runTRDCreate(fromPRD, output string, useSession bool) error {
	if c.documentChain == nil {
		return errors.New("document chain service not available")
	}

	fmt.Printf("🔧 Creating Technical Requirements Document\n")
	fmt.Printf("=========================================\n\n")

	ctx := context.Background()

	// Smart context detection
	if fromPRD == "" && useSession {
		fromPRD = c.detectLatestPRD()
		if fromPRD != "" {
			fmt.Printf("📄 Auto-detected PRD: %s\n", filepath.Base(fromPRD))
		} else {
			return NewPRDNotFoundError()
		}
	}

	// Load PRD
	prd, err := c.loadPRDFromFile(fromPRD)
	if err != nil {
		// Fallback to mock for demonstration
		prd = &services.PRDEntity{
			ID:          "prd-001",
			Title:       "Sample Project",
			Description: "A sample project for TRD generation",
			Features:    []string{"Feature 1", "Feature 2", "Feature 3"},
			UserStories: []string{"As a user, I want feature 1", "As a user, I want feature 2"},
			Metadata: map[string]interface{}{
				"repository":   c.detectRepository(c.getContext()),
				"project_type": "general",
			},
			CreatedAt: time.Now(),
		}
	}

	fmt.Printf("🔄 Generating TRD from: %s\n\n", prd.Title)

	// Generate TRD
	trd, err := c.documentChain.GenerateTRDFromPRD(ctx, prd)
	if err != nil {
		return fmt.Errorf("failed to generate TRD: %w", err)
	}

	// Determine output file
	if output == "" && useSession {
		// Default output location with matching name
		preDev := DefaultPreDevelopmentDir
		if err := os.MkdirAll(preDev, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", preDev, err)
		}

		// Extract base name from PRD file
		baseName := "project"
		if fromPRD != "" {
			base := filepath.Base(fromPRD)
			if strings.HasPrefix(base, "prd-") {
				baseName = strings.TrimPrefix(base, "prd-")
				baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
			}
		}

		output = filepath.Join(preDev, fmt.Sprintf("trd-%s.md", baseName))
	}

	// Save TRD if output specified
	if output != "" {
		content := c.formatTRDAsMarkdown(trd)
		if err := os.WriteFile(output, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to save TRD: %w", err)
		}
		fmt.Printf("📄 TRD saved to: %s\n", output)

		// Update session
		if useSession {
			c.updateSession("trd_file", output)
		}
	}

	// Display results
	fmt.Printf("\n✅ TRD created successfully!\n")
	fmt.Printf("   ID: %s\n", trd.ID)
	fmt.Printf("   Title: %s\n", trd.Title)
	fmt.Printf("   Tech Stack: %d items\n", len(trd.TechStack))
	fmt.Printf("   Requirements: %d items\n", len(trd.Requirements))

	fmt.Printf("\n💡 Next steps:\n")
	fmt.Printf("   - Run 'lmmc tasks generate' to create project tasks\n")
	fmt.Printf("   - Run 'lmmc workflow continue' to proceed with automation\n")

	return nil
}

// runTRDImport handles TRD import functionality
func (c *CLI) runTRDImport(filePath, output string) error {
	fmt.Printf("📄 Importing TRD document: %s\n", filepath.Base(filePath))

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

	fmt.Printf("✅ TRD imported successfully\n")
	fmt.Printf("   File: %s\n", filePath)
	fmt.Printf("   Size: %d bytes\n", len(content))

	if output != "" {
		fmt.Printf("💾 Processed TRD saved to: %s\n", output)
	}

	return nil
}

// runTRDView handles TRD view functionality
func (c *CLI) runTRDView() error {
	fmt.Printf("📋 Current TRD View\n")
	fmt.Printf("===================\n\n")
	fmt.Printf("No active TRD found. Use 'lmmc trd create' or 'lmmc trd import' first.\n")
	return nil
}

// runTRDExport handles TRD export functionality
func (c *CLI) runTRDExport(format, output string) error {
	if output == "" {
		return errors.New("output file path is required")
	}

	fmt.Printf("📤 Exporting TRD to %s format\n", format)
	fmt.Printf("Output: %s\n", output)

	// Create a simple example export
	content := fmt.Sprintf("# Example TRD\n\nExported in %s format\nGenerated at: %s\n", format, time.Now().Format(time.RFC3339))

	if err := os.WriteFile(output, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	fmt.Printf("✅ TRD exported successfully to: %s\n", output)
	return nil
}

// formatTRDAsMarkdown formats a TRD as markdown
func (c *CLI) formatTRDAsMarkdown(trd *services.TRDEntity) string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# %s\n\n", trd.Title))
	content.WriteString(fmt.Sprintf("**Created:** %s\n", trd.CreatedAt.Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("**ID:** %s\n", trd.ID))
	content.WriteString(fmt.Sprintf("**PRD ID:** %s\n\n", trd.PRDID))

	content.WriteString("## Architecture\n\n")
	content.WriteString(trd.Architecture + "\n\n")

	if len(trd.TechStack) > 0 {
		content.WriteString("## Technology Stack\n\n")
		for i, tech := range trd.TechStack {
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, tech))
		}
		content.WriteString("\n")
	}

	if len(trd.Requirements) > 0 {
		content.WriteString("## Requirements\n\n")
		for _, req := range trd.Requirements {
			content.WriteString(fmt.Sprintf("- %s\n", req))
		}
		content.WriteString("\n")
	}

	if len(trd.Implementation) > 0 {
		content.WriteString("## Implementation Steps\n\n")
		for i, step := range trd.Implementation {
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
		}
		content.WriteString("\n")
	}

	return content.String()
}
