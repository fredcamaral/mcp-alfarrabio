package cli

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/adapters/secondary/ai"
)

// createStatusCommand creates the 'status' command to show service status
func (c *CLI) createStatusCommand() *cobra.Command {
	var detailed bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show status of all services and features",
		Long: `Display comprehensive status information about all configured services,
including AI providers, MCP server connectivity, storage, and feature availability.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c.printSystemStatus(cmd, detailed)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "Show detailed status information")

	return cmd
}

// printSystemStatus displays comprehensive system status
func (c *CLI) printSystemStatus(cmd *cobra.Command, detailed bool) {
	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "🔍 LMMC System Status\n")
	fmt.Fprintf(out, "=====================\n\n")

	// System Information
	fmt.Fprintf(out, "System Information:\n")
	fmt.Fprintf(out, "  Version: %s\n", c.RootCmd.Version)
	fmt.Fprintf(out, "  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(out, "  Go Version: %s\n", runtime.Version())

	// Repository Detection
	fmt.Fprintf(out, "\nRepository:\n")
	repo := c.detectRepository()
	if repo != "" {
		fmt.Fprintf(out, "  Current: %s ✅\n", repo)
	} else {
		fmt.Fprintf(out, "  Current: (none detected) ❌\n")
		fmt.Fprintf(out, "  💡 Tip: Run from a git repository for full functionality\n")
	}

	// Configuration
	fmt.Fprintf(out, "\nConfiguration:\n")
	if config, err := c.configMgr.Load(); err == nil {
		fmt.Fprintf(out, "  Config file: %s ✅\n", c.configMgr.GetConfigPath())
		fmt.Fprintf(out, "  Log level: %s\n", config.Logging.Level)
		fmt.Fprintf(out, "  Output format: %s\n", config.CLI.OutputFormat)
	} else {
		fmt.Fprintf(out, "  Config file: Error loading ❌\n")
	}

	// Storage Status
	fmt.Fprintf(out, "\nStorage:\n")
	if c.storage != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := c.storage.HealthCheck(ctx); err == nil {
			fmt.Fprintf(out, "  Local storage: ✅ Healthy\n")

			// Get task count
			if c.taskService != nil {
				if tasks, err := c.taskService.ListTasks(ctx, nil); err == nil {
					fmt.Fprintf(out, "  Total tasks: %d\n", len(tasks))
				}
			}
		} else {
			fmt.Fprintf(out, "  Local storage: ❌ Error (%v)\n", err)
		}
	} else {
		fmt.Fprintf(out, "  Local storage: ❌ Not initialized\n")
	}

	// AI Service Status
	fmt.Fprintf(out, "\nAI Service:\n")
	if c.aiService == nil {
		fmt.Fprintf(out, "  Status: ❌ Not initialized\n")
	} else {
		// Check if it's enhanced service
		if _, ok := c.aiService.(*ai.EnhancedAIService); ok {
			fmt.Fprintf(out, "  Type: Enhanced (with task processing) ✅\n")
		} else {
			fmt.Fprintf(out, "  Type: Basic\n")
		}

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := c.aiService.TestConnection(ctx); err != nil {
			fmt.Fprintf(out, "  Status: ❌ Error (%v)\n", err)
		} else if c.aiService.IsOnline() {
			fmt.Fprintf(out, "  Status: ✅ Online\n")

			// Detect provider
			if key := os.Getenv("OPENAI_API_KEY"); key != "" {
				fmt.Fprintf(out, "  Provider: OpenAI\n")
			} else if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
				fmt.Fprintf(out, "  Provider: Anthropic Claude\n")
			} else if key := os.Getenv("PERPLEXITY_API_KEY"); key != "" {
				fmt.Fprintf(out, "  Provider: Perplexity\n")
			}
		} else {
			fmt.Fprintf(out, "  Status: ⚠️  Mock/Offline mode\n")
		}
	}

	// MCP Server Status
	fmt.Fprintf(out, "\nMCP Server:\n")
	if c.taskService != nil && c.taskService.GetMCPClient() != nil {
		client := c.taskService.GetMCPClient()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		config, _ := c.configMgr.Load()
		fmt.Fprintf(out, "  URL: %s\n", config.Server.URL)

		if err := client.TestConnection(ctx); err != nil {
			fmt.Fprintf(out, "  Status: ❌ Offline\n")
			if detailed {
				fmt.Fprintf(out, "  Error: %v\n", err)
			}
		} else {
			fmt.Fprintf(out, "  Status: ✅ Online\n")
		}
	} else {
		fmt.Fprintf(out, "  Status: ❌ Not configured\n")
	}

	// Intelligence Features
	fmt.Fprintf(out, "\nIntelligence Features:\n")
	if c.intelligence != nil {
		if c.intelligence.AnalyticsService != nil {
			fmt.Fprintf(out, "  Analytics: ✅ Available\n")
		} else {
			fmt.Fprintf(out, "  Analytics: ❌ Not available\n")
		}

		if c.intelligence.SuggestionService != nil {
			fmt.Fprintf(out, "  Suggestions: ✅ Available\n")
		} else {
			fmt.Fprintf(out, "  Suggestions: ❌ Not available\n")
		}

		if c.intelligence.PatternDetector != nil {
			fmt.Fprintf(out, "  Pattern Detection: ✅ Available\n")
		} else {
			fmt.Fprintf(out, "  Pattern Detection: ❌ Not available\n")
		}

		if c.intelligence.CrossRepoAnalyzer != nil {
			fmt.Fprintf(out, "  Cross-Repo Analysis: ✅ Available\n")
		} else {
			fmt.Fprintf(out, "  Cross-Repo Analysis: ❌ Not available\n")
		}
	} else {
		fmt.Fprintf(out, "  Intelligence: ❌ Not initialized\n")
	}

	// Feature Availability Summary
	fmt.Fprintf(out, "\nFeature Availability:\n")

	// Basic features
	fmt.Fprintf(out, "  ✅ Task Management (add, list, edit, delete)\n")
	fmt.Fprintf(out, "  ✅ Repository Detection\n")
	fmt.Fprintf(out, "  ✅ Configuration Management\n")

	// Advanced features
	if c.aiService != nil && c.aiService.IsOnline() {
		fmt.Fprintf(out, "  ✅ AI-Powered Task Processing\n")
		fmt.Fprintf(out, "  ✅ Document Generation (PRD/TRD)\n")
	} else {
		fmt.Fprintf(out, "  ❌ AI-Powered Task Processing (no AI provider)\n")
		fmt.Fprintf(out, "  ❌ Document Generation (no AI provider)\n")
	}

	if c.taskService != nil && c.taskService.GetMCPClient() != nil {
		fmt.Fprintf(out, "  ✅ Server Synchronization\n")
	} else {
		fmt.Fprintf(out, "  ⚠️  Server Synchronization (offline mode)\n")
	}

	if c.intelligence != nil && c.intelligence.AnalyticsService != nil {
		fmt.Fprintf(out, "  ✅ Analytics & Insights\n")
	} else {
		fmt.Fprintf(out, "  ⚠️  Analytics & Insights (limited data)\n")
	}

	// Troubleshooting Tips
	fmt.Fprintf(out, "\n💡 Troubleshooting Tips:\n")

	hasIssues := false

	if c.aiService == nil || !c.aiService.IsOnline() {
		fmt.Fprintf(out, "  • Set OPENAI_API_KEY for AI features\n")
		fmt.Fprintf(out, "  • Run 'lmmc ai --debug-ai process' to diagnose AI issues\n")
		hasIssues = true
	}

	if c.taskService == nil || c.taskService.GetMCPClient() == nil {
		fmt.Fprintf(out, "  • Configure server URL: 'lmmc config set server.url <url>'\n")
		fmt.Fprintf(out, "  • Start the MCP server for synchronization\n")
		hasIssues = true
	}

	if repo == "" {
		fmt.Fprintf(out, "  • Run from a git repository for full functionality\n")
		hasIssues = true
	}

	if !hasIssues {
		fmt.Fprintf(out, "  All systems operational! 🚀\n")
	}

	fmt.Fprintf(out, "\n")
}
