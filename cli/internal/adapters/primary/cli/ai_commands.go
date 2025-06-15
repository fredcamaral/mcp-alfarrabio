package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/adapters/secondary/ai"
	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
)

// createAICommand creates the 'ai' command with subcommands for AI-powered operations
func (c *CLI) createAICommand() *cobra.Command {
	var debugAI bool

	cmd := &cobra.Command{
		Use:   "ai",
		Short: "AI-powered task processing and memory management",
		Long: `AI-powered operations that enhance task processing and memory management.
Use AI to automatically enhance tasks, sync files intelligently, and get performance insights.

Examples:
  # Process a task with AI enhancements
  lmmc ai process <task-id>

  # Analyze performance patterns
  lmmc ai analyze

  # Get AI insights about your workflow
  lmmc ai insights

  # Debug AI service connectivity
  lmmc ai --debug-ai process <task-id>

Note: Requires AI provider API key (OPENAI_API_KEY, ANTHROPIC_API_KEY, or PERPLEXITY_API_KEY)`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if debugAI {
				c.printAIServiceDebugInfo(cmd)
			}
			return nil
		},
	}

	// Add persistent flag for AI debugging
	cmd.PersistentFlags().BoolVar(&debugAI, "debug-ai", false, "Show AI service initialization details")

	// Add subcommands
	cmd.AddCommand(c.createAIProcessCommand())
	cmd.AddCommand(c.createAISyncCommand())
	cmd.AddCommand(c.createAIOptimizeCommand())
	cmd.AddCommand(c.createAIAnalyzeCommand())
	cmd.AddCommand(c.createAIInsightsCommand())

	// Add provider management commands
	cmd.AddCommand(c.createAIProviderCommands())
	cmd.AddCommand(c.createAIModelCommands())
	cmd.AddCommand(c.createAICostCommands())
	cmd.AddCommand(c.createAIFallbackCommands())

	return cmd
}

// createAIProcessCommand creates the 'ai process' command for AI task processing
func (c *CLI) createAIProcessCommand() *cobra.Command {
	var taskID string

	cmd := &cobra.Command{
		Use:   "process [task-id]",
		Short: "Process a task with AI enhancements",
		Long: `Apply AI enhancements to a task including:
- Smart description improvement
- Automatic priority adjustment
- Intelligent tagging
- Time estimation
- Contextual suggestions
- Duplicate detection`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get task ID from args or flag
			if len(args) > 0 {
				taskID = args[0]
			}

			if taskID == "" {
				return c.handleError(cmd, errors.New("task ID is required"))
			}

			// Resolve task ID from short form
			fullTaskID, err := c.resolveTaskID(c.getContext(), taskID, "")
			if err != nil {
				return c.handleError(cmd, err)
			}

			// Get the task
			task, err := c.taskService.GetTask(c.getContext(), fullTaskID)
			if err != nil {
				return c.handleError(cmd, fmt.Errorf("failed to get task: %w", err))
			}

			// Get enhanced AI service
			enhancedAI, ok := c.aiService.(*ai.EnhancedAIService)
			if !ok {
				return c.handleError(cmd, errors.New("AI enhancements not available"))
			}

			// Set current context
			repoInfo, _ := c.repositoryDetector.DetectCurrent(c.getContext())
			repository := constants.RepositoryLocal
			if repoInfo != nil {
				repository = repoInfo.Name
			}

			workContext := &entities.WorkContext{
				Repository:        repository,
				EnergyLevel:       0.8, // Default values
				FocusLevel:        0.7,
				ProductivityScore: 0.75,
			}

			sessionID := fmt.Sprintf("cli_session_%d", time.Now().Unix())
			enhancedAI.SetContext(repository, sessionID, workContext)

			// Process task with AI
			result, err := enhancedAI.ProcessTaskWithAI(c.getContext(), task)
			if err != nil {
				return c.handleError(cmd, fmt.Errorf("AI processing failed: %w", err))
			}

			// Update the task if it was enhanced
			if result.TaskResult != nil && result.TaskResult.EnhancedTask != nil {
				enhanced := result.TaskResult.EnhancedTask
				if enhanced.Content != task.Content || enhanced.Priority != task.Priority || len(enhanced.Tags) != len(task.Tags) {
					if err := c.storage.UpdateTask(c.getContext(), enhanced); err != nil {
						c.logger.Warn("failed to save enhanced task", "error", err)
					} else {
						fmt.Printf("✨ Task enhanced with AI improvements\n\n")
					}
				}
			}

			// Display results
			return c.displayAIProcessResult(result)
		},
	}

	cmd.Flags().StringVarP(&taskID, "task", "t", "", "Task ID to process")

	return cmd
}

// createAISyncCommand creates the 'ai sync' command for intelligent file synchronization
func (c *CLI) createAISyncCommand() *cobra.Command {
	var localPath string
	var autoApply bool

	cmd := &cobra.Command{
		Use:   "sync [path]",
		Short: "Intelligently sync local files with memory server",
		Long: `Sync local files with the MCP memory server using AI to:
- Determine optimal sync strategy
- Detect and resolve conflicts
- Organize files intelligently
- Provide sync insights`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get local path from args or flag
			if len(args) > 0 {
				localPath = args[0]
			}
			if localPath == "" {
				localPath = "." // Default to current directory
			}

			// Get enhanced AI service
			enhancedAI, ok := c.aiService.(*ai.EnhancedAIService)
			if !ok {
				return c.handleError(cmd, errors.New("AI enhancements not available"))
			}

			// Set current context
			repoInfo, _ := c.repositoryDetector.DetectCurrent(c.getContext())
			repository := constants.RepositoryLocal
			if repoInfo != nil {
				repository = repoInfo.Name
			}

			sessionID := fmt.Sprintf("cli_session_%d", time.Now().Unix())
			enhancedAI.SetContext(repository, sessionID, nil)

			fmt.Printf("🤖 Starting AI-powered file sync for: %s\n", localPath)

			// Perform AI-enhanced sync
			result, err := enhancedAI.SyncMemoryWithAI(c.getContext(), localPath)
			if err != nil {
				return c.handleError(cmd, fmt.Errorf("AI sync failed: %w", err))
			}

			// Display results
			return c.displayAISyncResult(result)
		},
	}

	cmd.Flags().StringVarP(&localPath, "path", "p", "", "Local path to sync")
	cmd.Flags().BoolVarP(&autoApply, "auto-apply", "a", false, "Automatically apply AI recommendations")

	return cmd
}

// executeAIOperation is a helper function to execute AI operations with common setup
func (c *CLI) executeAIOperation(
	cmd *cobra.Command,
	startMessage string,
	operation func(context.Context, *ai.EnhancedAIService) (interface{}, error),
	displayResult func(interface{}) error,
) error {
	// Get enhanced AI service
	enhancedAI, ok := c.aiService.(*ai.EnhancedAIService)
	if !ok {
		return c.handleError(cmd, errors.New("AI enhancements not available"))
	}

	// Set current context
	repoInfo, _ := c.repositoryDetector.DetectCurrent(c.getContext())
	repository := "local"
	if repoInfo != nil {
		repository = repoInfo.Name
	}

	sessionID := fmt.Sprintf("cli_session_%d", time.Now().Unix())
	enhancedAI.SetContext(repository, sessionID, nil)

	fmt.Printf("%s\n", startMessage)

	// Perform the operation
	result, err := operation(c.getContext(), enhancedAI)
	if err != nil {
		return err
	}

	// Display results
	return displayResult(result)
}

// createAIOptimizeCommand creates the 'ai optimize' command for workflow optimization
func (c *CLI) createAIOptimizeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Optimize workflow and performance with AI",
		Long: `Use AI to analyze and optimize your workflow:
- Storage optimization
- Performance analysis
- Workflow improvements
- Automatic optimizations`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.executeAIOperation(
				cmd,
				"🚀 Starting AI workflow optimization...",
				func(ctx context.Context, ai *ai.EnhancedAIService) (interface{}, error) {
					result, err := ai.OptimizeWorkflow(ctx)
					if err != nil {
						return nil, fmt.Errorf("workflow optimization failed: %w", err)
					}
					return result, nil
				},
				func(result interface{}) error {
					return c.displayAIOptimizationResult(result.(*ai.AICommandResult))
				},
			)
		},
	}

	return cmd
}

// createAIAnalyzeCommand creates the 'ai analyze' command for performance analysis
func (c *CLI) createAIAnalyzeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze performance and patterns with AI",
		Long: `Use AI to analyze your performance patterns:
- Task completion patterns
- Productivity insights
- Memory usage analysis
- Performance recommendations`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.executeAIOperation(
				cmd,
				"📊 Starting AI performance analysis...",
				func(ctx context.Context, ai *ai.EnhancedAIService) (interface{}, error) {
					result, err := ai.AnalyzePerformance(ctx)
					if err != nil {
						return nil, fmt.Errorf("performance analysis failed: %w", err)
					}
					return result, nil
				},
				func(result interface{}) error {
					return c.displayAIAnalysisResult(result.(*ai.AICommandResult))
				},
			)
		},
	}

	return cmd
}

// createAIInsightsCommand creates the 'ai insights' command for memory insights
func (c *CLI) createAIInsightsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "insights",
		Short: "Get AI-powered insights about memory usage",
		Long: `Get intelligent insights about your memory usage patterns:
- Memory efficiency analysis
- Usage pattern detection
- Optimization recommendations
- Health dashboard`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.executeAIOperation(
				cmd,
				"💡 Generating AI-powered memory insights...",
				func(ctx context.Context, ai *ai.EnhancedAIService) (interface{}, error) {
					insights, err := ai.GetMemoryInsights(ctx)
					if err != nil {
						return nil, fmt.Errorf("failed to get memory insights: %w", err)
					}
					return insights, nil
				},
				func(result interface{}) error {
					return c.displayMemoryInsights(result.([]*ai.MemoryInsight))
				},
			)
		},
	}

	return cmd
}

// Display methods for AI command results

func (c *CLI) displayAIProcessResult(result *ai.AICommandResult) error {
	fmt.Printf("🎯 AI Task Processing Results\n")
	fmt.Printf("═══════════════════════════════\n\n")

	if result.TaskResult != nil {
		tr := result.TaskResult

		// Show enhancements
		if len(tr.ProcessingNotes) > 0 {
			fmt.Printf("✨ Enhancements Applied:\n")
			for _, note := range tr.ProcessingNotes {
				fmt.Printf("  • %s\n", note)
			}
			fmt.Printf("\n")
		}

		// Show suggestions
		if len(tr.Suggestions) > 0 {
			fmt.Printf("💡 AI Suggestions (%d):\n", len(tr.Suggestions))
			for i, suggestion := range tr.Suggestions {
				fmt.Printf("  %d. %s\n", i+1, suggestion.Title)
				fmt.Printf("     %s\n", suggestion.Description)
				fmt.Printf("     Priority: %s | Confidence: %.1f%% | Est: %dm\n",
					suggestion.Priority, suggestion.Confidence*100, suggestion.EstimatedMins)
				if suggestion.Reasoning != "" {
					fmt.Printf("     💭 %s\n", suggestion.Reasoning)
				}
				fmt.Printf("\n")
			}
		}

		// Show duplicates if found
		if len(tr.Duplicates) > 0 {
			fmt.Printf("⚠️  Potential Duplicates Found (%d):\n", len(tr.Duplicates))
			for _, dup := range tr.Duplicates {
				fmt.Printf("  • %s (ID: %s)\n", dup.Content, dup.ID)
			}
			fmt.Printf("\n")
		}

		// Show related tasks
		if len(tr.RelatedTasks) > 0 {
			fmt.Printf("🔗 Related Tasks (%d):\n", len(tr.RelatedTasks))
			for _, related := range tr.RelatedTasks {
				fmt.Printf("  • %s (%s)\n", related.Content, related.Status)
			}
			fmt.Printf("\n")
		}
	}

	// Show context insights
	if len(result.ContextInsights) > 0 {
		fmt.Printf("🧠 Context Insights:\n")
		for _, insight := range result.ContextInsights {
			fmt.Printf("  • %s\n", insight)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("⏱️  Processing Time: %v\n", result.ProcessingTime)
	fmt.Printf("✅ Status: %s\n", map[bool]string{true: "Success", false: "Failed"}[result.Success])

	return nil
}

func (c *CLI) displayAISyncResult(result *ai.AICommandResult) error {
	fmt.Printf("🔄 AI Memory Sync Results\n")
	fmt.Printf("═══════════════════════════\n\n")

	if result.MemoryResult != nil {
		mr := result.MemoryResult

		fmt.Printf("📊 Sync Statistics:\n")
		fmt.Printf("  • Files Processed: %d\n", mr.FilesProcessed)
		fmt.Printf("  • Memories Created: %d\n", mr.MemoriesCreated)
		fmt.Printf("  • Memories Updated: %d\n", mr.MemoriesUpdated)

		if len(mr.Conflicts) > 0 {
			fmt.Printf("  • Conflicts Resolved: %d\n", len(mr.Conflicts))
		}

		fmt.Printf("  • Processing Time: %v\n", mr.ProcessingTime)
		fmt.Printf("  • Storage Used: %s\n", formatBytes(mr.StorageUsed))
		fmt.Printf("\n")

		// Show insights
		if len(mr.Insights) > 0 {
			fmt.Printf("💡 Sync Insights:\n")
			for _, insight := range mr.Insights {
				fmt.Printf("  • %s: %s\n", insight.Title, insight.Description)
			}
			fmt.Printf("\n")
		}

		// Show recommendations
		if len(mr.Recommendations) > 0 {
			fmt.Printf("📋 Recommendations:\n")
			for _, rec := range mr.Recommendations {
				fmt.Printf("  • %s\n", rec)
			}
			fmt.Printf("\n")
		}
	}

	// Show context insights
	if len(result.ContextInsights) > 0 {
		fmt.Printf("🧠 AI Insights:\n")
		for _, insight := range result.ContextInsights {
			fmt.Printf("  • %s\n", insight)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("✅ Status: %s\n", map[bool]string{true: "Success", false: "Failed"}[result.Success])

	return nil
}

func (c *CLI) displayAIOptimizationResult(result *ai.AICommandResult) error {
	fmt.Printf("🚀 AI Workflow Optimization Results\n")
	fmt.Printf("══════════════════════════════════\n\n")

	if len(result.ContextInsights) > 0 {
		fmt.Printf("💡 Optimization Recommendations:\n")
		for i, insight := range result.ContextInsights {
			fmt.Printf("  %d. %s\n", i+1, insight)
		}
		fmt.Printf("\n")
	}

	if result.MemoryResult != nil && len(result.MemoryResult.Insights) > 0 {
		fmt.Printf("🔧 Storage Optimizations:\n")
		for _, insight := range result.MemoryResult.Insights {
			fmt.Printf("  • %s: %s\n", insight.Title, insight.Description)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("⏱️  Processing Time: %v\n", result.ProcessingTime)
	fmt.Printf("✅ Status: %s\n", map[bool]string{true: "Success", false: "Failed"}[result.Success])

	return nil
}

func (c *CLI) displayAIAnalysisResult(result *ai.AICommandResult) error {
	fmt.Printf("📊 AI Performance Analysis Results\n")
	fmt.Printf("═══════════════════════════════════\n\n")

	// Show performance metrics
	if len(result.PerformanceMetrics) > 0 {
		fmt.Printf("📈 Performance Metrics:\n")
		for metric, value := range result.PerformanceMetrics {
			fmt.Printf("  • %s: %v\n", strings.Title(strings.ReplaceAll(metric, "_", " ")), value)
		}
		fmt.Printf("\n")
	}

	// Show insights
	if len(result.ContextInsights) > 0 {
		fmt.Printf("🧠 Analysis Insights:\n")
		for _, insight := range result.ContextInsights {
			fmt.Printf("  • %s\n", insight)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("⏱️  Analysis Time: %v\n", result.ProcessingTime)
	fmt.Printf("✅ Status: %s\n", map[bool]string{true: "Success", false: "Failed"}[result.Success])

	return nil
}

func (c *CLI) displayMemoryInsights(insights []*ai.MemoryInsight) error {
	fmt.Printf("💡 AI Memory Insights\n")
	fmt.Printf("═══════════════════════\n\n")

	if len(insights) == 0 {
		fmt.Printf("No insights available at this time.\n")
		return nil
	}

	for _, insight := range insights {
		priority := "📌"
		switch insight.Priority {
		case "high":
			priority = "🔴"
		case "medium":
			priority = "🟡"
		case "low":
			priority = "🟢"
		}

		fmt.Printf("%s %s\n", priority, insight.Title)
		fmt.Printf("   %s\n", insight.Description)

		if len(insight.ActionItems) > 0 {
			fmt.Printf("   Actions:\n")
			for _, action := range insight.ActionItems {
				fmt.Printf("   • %s\n", action)
			}
		}

		if insight.Confidence > 0 {
			fmt.Printf("   Confidence: %.1f%%\n", insight.Confidence*100)
		}

		fmt.Printf("   Generated: %s\n\n", insight.GeneratedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// Helper functions
// Note: formatBytes function is defined in sync_commands.go

// printAIServiceDebugInfo prints detailed information about AI service initialization
func (c *CLI) printAIServiceDebugInfo(cmd *cobra.Command) {
	out := cmd.OutOrStdout()

	c.printDebugHeader(out)
	aiProvider := c.printEnvironmentVariables(out)
	foundKey := c.printAIProviderKeys(out)
	c.printAIServiceStatus(out)
	c.printMCPServerStatus(out)
	c.printTroubleshootingTips(out, foundKey, aiProvider)
}

// printDebugHeader prints the debug information header
func (c *CLI) printDebugHeader(out io.Writer) {
	_, _ = fmt.Fprintf(out, "🔍 AI Service Debug Information\n")
	_, _ = fmt.Fprintf(out, "================================\n\n")
}

// printEnvironmentVariables prints AI_PROVIDER info and returns its value
func (c *CLI) printEnvironmentVariables(out io.Writer) string {
	_, _ = fmt.Fprintf(out, "Environment Variables:\n")
	aiProvider := os.Getenv("AI_PROVIDER")
	if aiProvider != "" {
		_, _ = fmt.Fprintf(out, "  ✓ AI_PROVIDER: %s\n", aiProvider)
	} else {
		_, _ = fmt.Fprintf(out, "  ✗ AI_PROVIDER: not set (auto-detection enabled)\n")
	}
	return aiProvider
}

// printAIProviderKeys prints API key information and returns whether any key was found
func (c *CLI) printAIProviderKeys(out io.Writer) bool {
	providers := map[string]string{
		"OPENAI_API_KEY":     "OpenAI",
		"ANTHROPIC_API_KEY":  "Anthropic Claude",
		"PERPLEXITY_API_KEY": "Perplexity",
	}

	foundKey := false
	for envVar, provider := range providers {
		if key := os.Getenv(envVar); key != "" {
			masked := c.maskAPIKey(key)
			_, _ = fmt.Fprintf(out, "  ✓ %s: %s (key: %s)\n", envVar, provider, masked)
			foundKey = true
		} else {
			fmt.Fprintf(out, "  ✗ %s: not set\n", envVar)
		}
	}

	if !foundKey {
		fmt.Fprintf(out, "\n⚠️  No AI provider API keys found!\n")
	}
	return foundKey
}

// maskAPIKey masks an API key for display
func (c *CLI) maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:8] + "..."
}

// printAIServiceStatus prints AI service status information
func (c *CLI) printAIServiceStatus(out io.Writer) {
	fmt.Fprintf(out, "\nAI Service Status:\n")

	if c.aiService == nil {
		fmt.Fprintf(out, "  ✗ AI Service: not initialized\n")
		return
	}

	c.printEnhancedServiceInfo(out)
}

// printEnhancedServiceInfo prints enhanced service specific information
func (c *CLI) printEnhancedServiceInfo(out io.Writer) {
	if _, ok := c.aiService.(*ai.EnhancedAIService); ok {
		fmt.Fprintf(out, "  ✓ AI Service Type: Enhanced (with task processing)\n")
		c.testAIServiceConnection(out)
	} else {
		fmt.Fprintf(out, "  ✓ AI Service Type: Basic\n")
	}
}

// testAIServiceConnection tests the AI service connection
func (c *CLI) testAIServiceConnection(out io.Writer) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := c.aiService.TestConnection(ctx); err != nil {
		fmt.Fprintf(out, "  ✗ AI Service Test: failed (%v)\n", err)
		return
	}

	if c.aiService.IsOnline() {
		fmt.Fprintf(out, "  ✓ AI Service Mode: REAL (connected to AI provider)\n")
	} else {
		fmt.Fprintf(out, "  ⚠️  AI Service Mode: MOCK or OFFLINE\n")
	}
}

// printMCPServerStatus prints MCP server connectivity information
func (c *CLI) printMCPServerStatus(out io.Writer) {
	fmt.Fprintf(out, "\nMCP Server Status:\n")

	if c.taskService == nil || c.taskService.GetMCPClient() == nil {
		fmt.Fprintf(out, "  ✗ MCP Server: not configured\n")
		return
	}

	client := c.taskService.GetMCPClient()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.TestConnection(ctx); err != nil {
		fmt.Fprintf(out, "  ✗ MCP Server: offline (%v)\n", err)
	} else {
		fmt.Fprintf(out, "  ✓ MCP Server: online\n")
	}
}

// printTroubleshootingTips prints troubleshooting tips
func (c *CLI) printTroubleshootingTips(out io.Writer, foundKey bool, aiProvider string) {
	fmt.Fprintf(out, "\n💡 Troubleshooting Tips:\n")

	if !foundKey {
		fmt.Fprintf(out, "  • Set OPENAI_API_KEY environment variable for OpenAI\n")
		fmt.Fprintf(out, "  • Set ANTHROPIC_API_KEY for Claude\n")
		fmt.Fprintf(out, "  • Set PERPLEXITY_API_KEY for Perplexity\n")
	}

	if aiProvider == "" {
		fmt.Fprintf(out, "  • Set AI_PROVIDER to force a specific provider\n")
	}

	fmt.Fprintf(out, "  • Run 'lmmc config list' to check server configuration\n")
	fmt.Fprintf(out, "  • Use --verbose flag for more detailed logs\n")
	fmt.Fprintf(out, "\n")
}
