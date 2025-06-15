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

	c.printSystemInfo(out)
	repo := c.printRepositoryStatus(out)
	c.printConfigurationStatus(out)
	c.printStorageStatus(out)
	c.printAIServiceStatus(out)
	c.printMCPServerStatus(out, detailed)
	c.printIntelligenceFeatures(out)
	c.printFeatureAvailability(out)
	c.printTroubleshootingTips(out, repo)

	fmt.Fprintf(out, "\n")
}

// printSystemInfo displays basic system information
func (c *CLI) printSystemInfo(out interface{ Write([]byte) (int, error) }) {
	fmt.Fprintf(out, "System Information:\n")
	fmt.Fprintf(out, "  Version: %s\n", c.RootCmd.Version)
	fmt.Fprintf(out, "  Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(out, "  Go Version: %s\n", runtime.Version())
}

// printRepositoryStatus displays repository detection information
func (c *CLI) printRepositoryStatus(out interface{ Write([]byte) (int, error) }) string {
	fmt.Fprintf(out, "\nRepository:\n")
	repo := c.detectRepository(c.getContext())
	if repo != "" {
		fmt.Fprintf(out, "  Current: %s ✅\n", repo)
	} else {
		fmt.Fprintf(out, "  Current: (none detected) ❌\n")
		fmt.Fprintf(out, "  💡 Tip: Run from a git repository for full functionality\n")
	}
	return repo
}

// printConfigurationStatus displays configuration status
func (c *CLI) printConfigurationStatus(out interface{ Write([]byte) (int, error) }) {
	fmt.Fprintf(out, "\nConfiguration:\n")
	if config, err := c.configMgr.Load(); err == nil {
		fmt.Fprintf(out, "  Config file: %s ✅\n", c.configMgr.GetConfigPath())
		fmt.Fprintf(out, "  Log level: %s\n", config.Logging.Level)
		fmt.Fprintf(out, "  Output format: %s\n", config.CLI.OutputFormat)
	} else {
		fmt.Fprintf(out, "  Config file: Error loading ❌\n")
	}
}

// printStorageStatus displays storage and task information
func (c *CLI) printStorageStatus(out interface{ Write([]byte) (int, error) }) {
	fmt.Fprintf(out, "\nStorage:\n")
	if c.storage == nil {
		fmt.Fprintf(out, "  Local storage: ❌ Not initialized\n")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.storage.HealthCheck(ctx); err == nil {
		fmt.Fprintf(out, "  Local storage: ✅ Healthy\n")
		c.printTaskCount(out, ctx)
	} else {
		fmt.Fprintf(out, "  Local storage: ❌ Error (%v)\n", err)
	}
}

// printTaskCount displays the current task count
func (c *CLI) printTaskCount(out interface{ Write([]byte) (int, error) }, ctx context.Context) {
	if c.taskService != nil {
		if tasks, err := c.taskService.ListTasks(ctx, nil); err == nil {
			fmt.Fprintf(out, "  Total tasks: %d\n", len(tasks))
		}
	}
}

// printAIServiceStatus displays AI service status and provider information
func (c *CLI) printAIServiceStatus(out interface{ Write([]byte) (int, error) }) {
	fmt.Fprintf(out, "\nAI Service:\n")
	if c.aiService == nil {
		fmt.Fprintf(out, "  Status: ❌ Not initialized\n")
		return
	}

	c.printAIServiceType(out)
	c.printAIConnectionStatus(out)
}

// printAIServiceType displays the type of AI service
func (c *CLI) printAIServiceType(out interface{ Write([]byte) (int, error) }) {
	if _, ok := c.aiService.(*ai.EnhancedAIService); ok {
		fmt.Fprintf(out, "  Type: Enhanced (with task processing) ✅\n")
	} else {
		fmt.Fprintf(out, "  Type: Basic\n")
	}
}

// printAIConnectionStatus tests and displays AI connection status
func (c *CLI) printAIConnectionStatus(out interface{ Write([]byte) (int, error) }) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.aiService.TestConnection(ctx); err != nil {
		fmt.Fprintf(out, "  Status: ❌ Error (%v)\n", err)
		return
	}

	if c.aiService.IsOnline() {
		fmt.Fprintf(out, "  Status: ✅ Online\n")
		c.detectAndPrintAIProvider(out)
	} else {
		fmt.Fprintf(out, "  Status: ⚠️  Mock/Offline mode\n")
	}
}

// detectAndPrintAIProvider detects and displays the current AI provider
func (c *CLI) detectAndPrintAIProvider(out interface{ Write([]byte) (int, error) }) {
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		fmt.Fprintf(out, "  Provider: OpenAI\n")
	} else if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		fmt.Fprintf(out, "  Provider: Anthropic Claude\n")
	} else if key := os.Getenv("PERPLEXITY_API_KEY"); key != "" {
		fmt.Fprintf(out, "  Provider: Perplexity\n")
	}
}

// printMCPServerStatus displays MCP server connection status
func (c *CLI) printMCPServerStatus(out interface{ Write([]byte) (int, error) }, detailed bool) {
	fmt.Fprintf(out, "\nMCP Server:\n")
	if c.taskService == nil || c.taskService.GetMCPClient() == nil {
		fmt.Fprintf(out, "  Status: ❌ Not configured\n")
		return
	}

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
}

// printIntelligenceFeatures displays intelligence feature availability
func (c *CLI) printIntelligenceFeatures(out interface{ Write([]byte) (int, error) }) {
	fmt.Fprintf(out, "\nIntelligence Features:\n")
	if c.intelligence == nil {
		fmt.Fprintf(out, "  Intelligence: ❌ Not initialized\n")
		return
	}

	c.printFeatureStatus(out, "Analytics", c.intelligence.AnalyticsService != nil)
	c.printFeatureStatus(out, "Suggestions", c.intelligence.SuggestionService != nil)
	c.printFeatureStatus(out, "Pattern Detection", c.intelligence.PatternDetector != nil)
	c.printFeatureStatus(out, "Cross-Repo Analysis", c.intelligence.CrossRepoAnalyzer != nil)
}

// printFeatureStatus displays status for a single feature
func (c *CLI) printFeatureStatus(out interface{ Write([]byte) (int, error) }, name string, available bool) {
	if available {
		fmt.Fprintf(out, "  %s: ✅ Available\n", name)
	} else {
		fmt.Fprintf(out, "  %s: ❌ Not available\n", name)
	}
}

// printFeatureAvailability displays overall feature availability summary
func (c *CLI) printFeatureAvailability(out interface{ Write([]byte) (int, error) }) {
	fmt.Fprintf(out, "\nFeature Availability:\n")

	// Basic features are always available
	fmt.Fprintf(out, "  ✅ Task Management (add, list, edit, delete)\n")
	fmt.Fprintf(out, "  ✅ Repository Detection\n")
	fmt.Fprintf(out, "  ✅ Configuration Management\n")

	c.printAIFeatureAvailability(out)
	c.printSyncFeatureAvailability(out)
	c.printAnalyticsFeatureAvailability(out)
}

// printAIFeatureAvailability displays AI-dependent feature status
func (c *CLI) printAIFeatureAvailability(out interface{ Write([]byte) (int, error) }) {
	if c.aiService != nil && c.aiService.IsOnline() {
		fmt.Fprintf(out, "  ✅ AI-Powered Task Processing\n")
		fmt.Fprintf(out, "  ✅ Document Generation (PRD/TRD)\n")
	} else {
		fmt.Fprintf(out, "  ❌ AI-Powered Task Processing (no AI provider)\n")
		fmt.Fprintf(out, "  ❌ Document Generation (no AI provider)\n")
	}
}

// printSyncFeatureAvailability displays synchronization feature status
func (c *CLI) printSyncFeatureAvailability(out interface{ Write([]byte) (int, error) }) {
	if c.taskService != nil && c.taskService.GetMCPClient() != nil {
		fmt.Fprintf(out, "  ✅ Server Synchronization\n")
	} else {
		fmt.Fprintf(out, "  ⚠️  Server Synchronization (offline mode)\n")
	}
}

// printAnalyticsFeatureAvailability displays analytics feature status
func (c *CLI) printAnalyticsFeatureAvailability(out interface{ Write([]byte) (int, error) }) {
	if c.intelligence != nil && c.intelligence.AnalyticsService != nil {
		fmt.Fprintf(out, "  ✅ Analytics & Insights\n")
	} else {
		fmt.Fprintf(out, "  ⚠️  Analytics & Insights (limited data)\n")
	}
}

// printTroubleshootingTips displays relevant troubleshooting suggestions
func (c *CLI) printTroubleshootingTips(out interface{ Write([]byte) (int, error) }, repo string) {
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
}
