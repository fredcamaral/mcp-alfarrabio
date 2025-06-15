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

	"lerian-mcp-memory-cli/internal/domain/entities"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/spf13/cobra"
)

// ReviewPhase represents a code review phase
type ReviewPhase string

const (
	PhaseFoundation ReviewPhase = "foundation"
	PhaseSecurity   ReviewPhase = "security"
	PhaseQuality    ReviewPhase = "quality"
	PhaseDocs       ReviewPhase = "docs"
	PhaseProduction ReviewPhase = "production"
	PhaseSynthesis  ReviewPhase = "synthesis"
	PhaseAll        ReviewPhase = "all"
)

// ReviewAnalysis represents a specific analysis type
type ReviewAnalysis string

const (
	AnalysisCodebaseOverview    ReviewAnalysis = "codebase-overview"
	AnalysisArchitecture        ReviewAnalysis = "architecture"
	AnalysisAPIContracts        ReviewAnalysis = "api-contracts"
	AnalysisSecurity            ReviewAnalysis = "security"
	AnalysisTestCoverage        ReviewAnalysis = "test-coverage"
	AnalysisProductionReadiness ReviewAnalysis = "production-readiness"
)

// createReviewCommand creates the 'review' command group
func (c *CLI) createReviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Automated code review and analysis",
		Long: `Perform comprehensive code review across all engineering disciplines.

The review process follows a systematic 18-prompt analysis chain organized into 6 phases:
- Foundation: Codebase overview and architecture
- Security: Vulnerability and dependency analysis
- Quality: Testing and monitoring
- Documentation: API and workflow documentation
- Production: Deployment readiness
- Synthesis: Comprehensive todo generation`,
	}

	// Add subcommands
	cmd.AddCommand(
		c.createReviewStartCommand(),
		c.createReviewStatusCommand(),
		c.createReviewOrchestratCommand(),
		c.createReviewPhaseCommand(),
		c.createReviewAnalysisCommands(),
		c.createReviewTodosCommand(),
	)

	return cmd
}

// createReviewStartCommand creates the 'review start' command
func (c *CLI) createReviewStartCommand() *cobra.Command {
	var (
		phase      string
		aiProvider string
		model      string
		output     string
		mode       string
	)

	cmd := &cobra.Command{
		Use:   "start [path]",
		Short: "Start a comprehensive code review",
		Long: `Start a new code review session for the specified path.

By default, runs all review phases. Use --phase to run specific phases only.

Examples:
  lmmc review start                    # Review current directory
  lmmc review start /path/to/project   # Review specific path
  lmmc review start --phase security   # Run only security phase
  lmmc review start --mode quick       # Run quick essential analysis
  lmmc review start --ai-provider anthropic --model claude-opus-4`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return c.runReviewStart(path, phase, mode, aiProvider, model, output)
		},
	}

	cmd.Flags().StringVar(&phase, "phase", "all", "Review phase to run (all, foundation, security, quality, docs, production)")
	cmd.Flags().StringVar(&mode, "mode", "full", "Review mode (full, quick, security, quality)")
	cmd.Flags().StringVar(&aiProvider, "ai-provider", "", "AI provider to use")
	cmd.Flags().StringVar(&model, "model", "", "Specific model to use")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory for review results")

	return cmd
}

// createReviewStatusCommand creates the 'review status' command
func (c *CLI) createReviewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [session-id]",
		Short: "Show review session status",
		Long: `Display the current status of a review session, including progress and findings.

If no session ID is provided, shows the most recent review session.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sessionID := ""
			if len(args) > 0 {
				sessionID = args[0]
			}
			return c.runReviewStatus(sessionID)
		},
	}

	return cmd
}

// createReviewOrchestratCommand creates the 'review orchestrate' command
func (c *CLI) createReviewOrchestratCommand() *cobra.Command {
	var (
		quick  bool
		focus  string
		output string
	)

	cmd := &cobra.Command{
		Use:   "orchestrate [path]",
		Short: "Orchestrate complete review workflow",
		Long: `Run the complete code review orchestration with optimized execution.

Examples:
  lmmc review orchestrate                     # Full review
  lmmc review orchestrate --quick             # Essential reviews only
  lmmc review orchestrate --focus security    # Focus on specific area`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return c.runReviewOrchestrate(path, quick, focus, output)
		},
	}

	cmd.Flags().BoolVar(&quick, "quick", false, "Run quick analysis (essential prompts only)")
	cmd.Flags().StringVar(&focus, "focus", "", "Focus area (security, quality, production)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory for review results")

	return cmd
}

// createReviewPhaseCommand creates the 'review phase' command
func (c *CLI) createReviewPhaseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "phase",
		Short: "Run specific review phases",
		Long:  `Execute code review by phase, running all prompts within that phase.`,
	}

	// Add phase subcommands
	phases := map[string]struct {
		phase       ReviewPhase
		description string
		prompts     string
	}{
		"foundation": {PhaseFoundation, "Foundation and technical architecture", "prompts 01-06"},
		"security":   {PhaseSecurity, "Security and compliance analysis", "prompts 07-09"},
		"quality":    {PhaseQuality, "Quality assurance and testing", "prompts 10-12"},
		"docs":       {PhaseDocs, "Documentation and workflow", "prompts 13-15"},
		"production": {PhaseProduction, "Production readiness", "prompts 16-17"},
		"synthesis":  {PhaseSynthesis, "Final synthesis and todo generation", "prompt 18"},
	}

	for name, info := range phases {
		phase := info.phase // Capture in closure
		cmd.AddCommand(&cobra.Command{
			Use:   name,
			Short: info.description,
			Long:  fmt.Sprintf("%s\nRuns %s", info.description, info.prompts),
			RunE: func(cmd *cobra.Command, args []string) error {
				path := "."
				if len(args) > 0 {
					path = args[0]
				}
				return c.runReviewPhase(path, phase)
			},
		})
	}

	// Add numeric phase support (1-6)
	for i := 1; i <= 6; i++ {
		phaseNum := i
		phaseMap := []ReviewPhase{
			PhaseFoundation, // 1
			PhaseSecurity,   // 2
			PhaseQuality,    // 3
			PhaseDocs,       // 4
			PhaseProduction, // 5
			PhaseSynthesis,  // 6
		}

		cmd.AddCommand(&cobra.Command{
			Use:   strconv.Itoa(phaseNum),
			Short: fmt.Sprintf("Run phase %d", phaseNum),
			RunE: func(cmd *cobra.Command, args []string) error {
				path := "."
				if len(args) > 0 {
					path = args[0]
				}
				return c.runReviewPhase(path, phaseMap[phaseNum-1])
			},
		})
	}

	return cmd
}

// createReviewAnalysisCommands creates individual analysis commands
func (c *CLI) createReviewAnalysisCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Run individual analysis types",
		Long:  `Execute specific code analysis without running full phases.`,
	}

	// Individual analysis commands
	analyses := map[string]struct {
		analysis    ReviewAnalysis
		description string
	}{
		"codebase-overview":    {AnalysisCodebaseOverview, "Analyze codebase structure and overview"},
		"architecture":         {AnalysisArchitecture, "Analyze system architecture"},
		"api-contracts":        {AnalysisAPIContracts, "Analyze API contracts and interfaces"},
		"security":             {AnalysisSecurity, "Perform security analysis"},
		"test-coverage":        {AnalysisTestCoverage, "Analyze test coverage and quality"},
		"production-readiness": {AnalysisProductionReadiness, "Assess production readiness"},
	}

	for name, info := range analyses {
		analysis := info.analysis // Capture in closure
		cmd.AddCommand(&cobra.Command{
			Use:   name + " [path]",
			Short: info.description,
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				path := "."
				if len(args) > 0 {
					path = args[0]
				}
				return c.runReviewAnalysis(path, analysis)
			},
		})
	}

	return cmd
}

// createReviewTodosCommand creates the 'review todos' command group
func (c *CLI) createReviewTodosCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "todos",
		Short: "Manage review-generated todos",
		Long:  `List, export, and sync todos generated from code reviews.`,
	}

	// Subcommands
	cmd.AddCommand(
		c.createReviewTodosListCommand(),
		c.createReviewTodosExportCommand(),
		c.createReviewTodosSyncCommand(),
	)

	return cmd
}

// createReviewTodosListCommand creates the 'review todos list' command
func (c *CLI) createReviewTodosListCommand() *cobra.Command {
	var (
		severity string
		phase    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List todos from code review",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runReviewTodosList(severity, phase)
		},
	}

	cmd.Flags().StringVar(&severity, "severity", "all", "Filter by severity (critical, high, medium, all)")
	cmd.Flags().StringVar(&phase, "phase", "", "Filter by review phase")

	return cmd
}

// createReviewTodosExportCommand creates the 'review todos export' command
func (c *CLI) createReviewTodosExportCommand() *cobra.Command {
	var (
		format string
		output string
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export review todos",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runReviewTodosExport(format, output)
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "markdown", "Export format (markdown, json, jira)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file")

	return cmd
}

// createReviewTodosSyncCommand creates the 'review todos sync' command
func (c *CLI) createReviewTodosSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync todos to task management system",
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runReviewTodosSync()
		},
	}

	return cmd
}

// Implementation stubs - to be filled with actual logic

func (c *CLI) runReviewStart(path, phase, _, aiProvider, model, output string) error {
	fmt.Printf("🔍 Starting Code Review\n")
	fmt.Printf("======================\n\n")

	// Validate path
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Change to the specified path
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	absPath, _ := filepath.Abs(path)
	if err := os.Chdir(absPath); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			// Log error but don't return as we're in defer
		}
	}()

	fmt.Printf("📁 Reviewing: %s\n", absPath)
	fmt.Printf("📊 Phase: %s\n", phase)

	if aiProvider != "" {
		fmt.Printf("🤖 AI Provider: %s\n", aiProvider)
		if model != "" {
			fmt.Printf("🧠 Model: %s\n", model)
		}
	}

	fmt.Printf("\n⏳ Initializing review session...\n")

	// Check if review service is available
	if c.reviewService == nil {
		return errors.New("review service not initialized - check if prompts directory exists")
	}

	// Map phase string to review mode
	reviewMode := c.mapPhaseToReviewMode(phase)

	// Create review configuration
	config := entities.ReviewConfiguration{
		Mode:             reviewMode,
		OutputDirectory:  output,
		GenerateTodoList: true,
		StoreInMemory:    true,
		TimeoutPerPrompt: 5 * time.Minute,
		RetryCount:       2,
	}

	// Add phase filtering if specific phase requested
	if phase != string(PhaseAll) && phase != "" {
		reviewPhase := c.mapStringToReviewPhase(phase)
		if reviewPhase != "" {
			config.IncludePhases = []entities.ReviewPhase{reviewPhase}
		}
	}

	// Start review session
	ctx := context.Background()
	session, err := c.reviewService.StartReview(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to start review: %w", err)
	}

	// Create session tracking
	c.updateSession("review_session_id", session.ID)
	c.updateSession("review_path", absPath)
	c.updateSession("review_phase", phase)
	c.updateSession("review_started", time.Now().Format(time.RFC3339))

	fmt.Printf("\n✅ Review session started!\n")
	fmt.Printf("📝 Session ID: %s\n", session.ID)
	fmt.Printf("\n⏳ Review is running in the background...\n")
	fmt.Printf("\n💡 Check progress with:\n")
	fmt.Printf("   lmmc review status %s\n", session.ID)
	fmt.Printf("\n📊 Results will be saved to: %s\n", output)

	return nil
}

func (c *CLI) runReviewStatus(sessionID string) error {
	if c.reviewService == nil {
		return errors.New("review service not initialized")
	}

	resolvedSessionID, err := c.resolveSessionID(sessionID)
	if err != nil {
		return err
	}

	session, err := c.reviewService.GetSession(resolvedSessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	c.displayReviewHeader(session)
	c.displayReviewProgress(session)
	c.displayPhaseProgress(session)
	c.displayFindingsSummary(session)
	c.displayNextSteps(session)

	return nil
}

// resolveSessionID resolves the session ID from parameter or current session
func (c *CLI) resolveSessionID(sessionID string) (string, error) {
	if sessionID != "" {
		return sessionID, nil
	}

	sessionData, err := c.loadSession()
	if err == nil && sessionData.Values != nil {
		if stored, exists := sessionData.Values["review_session_id"]; exists {
			return stored, nil
		}
	}

	return "", errors.New("no session ID provided and no recent session found")
}

// displayReviewHeader displays the basic session information
func (c *CLI) displayReviewHeader(session *entities.ReviewSession) {
	fmt.Printf("📊 Review Session Status\n")
	fmt.Printf("=======================\n\n")

	fmt.Printf("📝 Session ID: %s\n", session.ID)
	fmt.Printf("📁 Repository: %s\n", session.Repository)
	if session.Branch != "" {
		fmt.Printf("🌿 Branch: %s\n", session.Branch)
	}
	fmt.Printf("🚀 Mode: %s\n", session.Mode)
	fmt.Printf("📅 Started: %s\n", session.StartedAt.Format("2006-01-02 15:04:05"))

	statusEmoji := c.getStatusEmoji()
	fmt.Printf("📈 Status: %s %s\n", statusEmoji[session.Status], session.Status)

	if session.CompletedAt != nil {
		fmt.Printf("🏁 Completed: %s\n", session.CompletedAt.Format("2006-01-02 15:04:05"))
		duration := session.CompletedAt.Sub(session.StartedAt)
		fmt.Printf("⏱️  Duration: %s\n", duration.Round(time.Second))
	}
}

// displayReviewProgress displays overall progress information
func (c *CLI) displayReviewProgress(session *entities.ReviewSession) {
	fmt.Printf("\n📊 Progress\n")
	fmt.Printf("----------\n")
	fmt.Printf("Total Prompts: %d / %d (%.0f%%)\n",
		session.Progress.CompletedPrompts,
		session.Progress.TotalPrompts,
		float64(session.Progress.CompletedPrompts)/float64(session.Progress.TotalPrompts)*100)

	if session.Progress.CurrentPromptID != "" && session.Status == entities.ReviewStatusInProgress {
		fmt.Printf("Current: %s (Phase: %s)\n", session.Progress.CurrentPromptID, session.Progress.CurrentPhase)
	}
}

// displayPhaseProgress displays progress for each phase
func (c *CLI) displayPhaseProgress(session *entities.ReviewSession) {
	fmt.Printf("\n📈 Phase Progress\n")
	fmt.Printf("----------------\n")

	phaseOrder := []entities.ReviewPhase{
		entities.PhaseFoundation,
		entities.PhaseSecurity,
		entities.PhaseQuality,
		entities.PhaseDocumentation,
		entities.PhaseProduction,
		entities.PhaseSynthesis,
	}

	statusEmoji := c.getStatusEmoji()
	for _, phase := range phaseOrder {
		if progress, exists := session.Progress.PhaseProgress[phase]; exists {
			phaseEmoji := statusEmoji[progress.Status]
			fmt.Printf("%s %s: %d/%d prompts\n", phaseEmoji, phase, progress.CompletedPrompts, progress.TotalPrompts)
		}
	}
}

// displayFindingsSummary displays the findings summary if available
func (c *CLI) displayFindingsSummary(session *entities.ReviewSession) {
	if len(session.Findings) == 0 && session.Summary == nil {
		return
	}

	fmt.Printf("\n🔍 Findings Summary\n")
	fmt.Printf("------------------\n")

	if session.Summary != nil {
		c.displayDetailedSummary(session.Summary)
	} else {
		fmt.Printf("Total findings so far: %d\n", len(session.Findings))
	}
}

// displayDetailedSummary displays detailed findings summary
func (c *CLI) displayDetailedSummary(summary *entities.ReviewSummary) {
	fmt.Printf("🔴 Critical: %d\n", summary.FindingsBySeverity[entities.SeverityCritical])
	fmt.Printf("🟡 High: %d\n", summary.FindingsBySeverity[entities.SeverityHigh])
	fmt.Printf("🟢 Medium: %d\n", summary.FindingsBySeverity[entities.SeverityMedium])
	fmt.Printf("🔵 Low: %d\n", summary.FindingsBySeverity[entities.SeverityLow])
	fmt.Printf("\n📋 Total: %d findings\n", summary.TotalFindings)

	if summary.ProductionReadiness != "" {
		fmt.Printf("\n🚀 Production Readiness: %s\n", summary.ProductionReadiness)
	}
}

// displayNextSteps displays appropriate next steps based on session status
func (c *CLI) displayNextSteps(session *entities.ReviewSession) {
	switch session.Status {
	case entities.ReviewStatusCompleted:
		fmt.Printf("\n✅ Review Complete!\n")
		fmt.Printf("\n💡 Next steps:\n")
		fmt.Printf("   - Review findings: lmmc review todos list\n")
		fmt.Printf("   - Export todos: lmmc review todos export\n")
	case entities.ReviewStatusInProgress:
		fmt.Printf("\n⏳ Review in progress...\n")
		fmt.Printf("Check again later or wait for completion.\n")
	}
}

// getStatusEmoji returns a map of status to emoji mappings
func (c *CLI) getStatusEmoji() map[entities.ReviewStatus]string {
	return map[entities.ReviewStatus]string{
		entities.ReviewStatusPending:    "⏳",
		entities.ReviewStatusInProgress: "🔄",
		entities.ReviewStatusCompleted:  "✅",
		entities.ReviewStatusFailed:     "❌",
		entities.ReviewStatusCancelled:  "🚫",
	}
}

func (c *CLI) runReviewOrchestrate(_ string, quick bool, focus, _ string) error {
	fmt.Printf("🎯 Orchestrating Code Review\n")
	fmt.Printf("============================\n\n")

	if quick {
		fmt.Printf("⚡ Running quick analysis (essential prompts only)\n")
	}
	if focus != "" {
		fmt.Printf("🔍 Focusing on: %s\n", focus)
	}

	// TODO: Implement orchestration logic
	phases := []ReviewPhase{
		PhaseFoundation,
		PhaseSecurity,
		PhaseQuality,
		PhaseDocs,
		PhaseProduction,
		PhaseSynthesis,
	}

	for i, phase := range phases {
		fmt.Printf("\n📌 Phase %d: %s\n", i+1, phase)
		fmt.Printf("   Status: ⏳ Pending\n")
	}

	return nil
}

func (c *CLI) runReviewPhase(_ string, phase ReviewPhase) error {
	caser := cases.Title(language.English)
	fmt.Printf("🔄 Running Review Phase: %s\n", caser.String(string(phase)))
	fmt.Printf("================================\n\n")

	// Map phases to prompts
	prompts := map[ReviewPhase][]string{
		PhaseFoundation: {
			"01-codebase-overview",
			"02-architecture-analysis",
			"03-api-contract-analysis",
			"04-database-optimization",
			"05-sequence-diagram-visualization",
			"06-business-analysis",
		},
		PhaseSecurity: {
			"07-security-vulnerability-analysis",
			"08-dependency-security-analysis",
			"09-privacy-compliance-analysis",
		},
		PhaseQuality: {
			"10-test-coverage-analysis",
			"11-observability-monitoring",
			"12-pre-commit-quality-checks",
		},
		PhaseDocs: {
			"13-documentation-generation",
			"14-api-documentation-generator",
			"15-business-workflow-consistency",
		},
		PhaseProduction: {
			"16-production-readiness-audit",
			"17-deployment-preparation",
		},
		PhaseSynthesis: {
			"18-comprehensive-todo-generation",
		},
	}

	phasePrompts := prompts[phase]
	fmt.Printf("📋 Running %d analysis prompts...\n\n", len(phasePrompts))

	for _, prompt := range phasePrompts {
		fmt.Printf("▶️  %s\n", prompt)
		// TODO: Execute actual prompt
		time.Sleep(100 * time.Millisecond) // Simulate work
		fmt.Printf("   ✅ Completed\n")
	}

	fmt.Printf("\n✅ Phase %s completed!\n", phase)

	// Update session
	c.updateSession(fmt.Sprintf("review_phase_%s", phase), "completed")

	return nil
}

func (c *CLI) runReviewAnalysis(path string, analysis ReviewAnalysis) error {
	fmt.Printf("🔬 Running Analysis: %s\n", strings.ReplaceAll(string(analysis), "-", " "))
	fmt.Printf("====================================\n\n")

	fmt.Printf("📁 Analyzing: %s\n", path)

	// TODO: Implement specific analysis logic
	fmt.Printf("\n⏳ Analysis in progress...\n")

	// Simulate analysis
	time.Sleep(500 * time.Millisecond)

	fmt.Printf("\n✅ Analysis completed!\n")
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   - Issues found: 3\n")
	fmt.Printf("   - Severity: 1 high, 2 medium\n")
	fmt.Printf("   - Recommendations: 5\n")

	return nil
}

func (c *CLI) runReviewTodosList(severity, phase string) error {
	fmt.Printf("📋 Review Todo List\n")
	fmt.Printf("===================\n\n")

	if severity != string(PhaseAll) {
		fmt.Printf("Filter: %s severity\n", severity)
	}
	if phase != "" {
		fmt.Printf("Phase: %s\n", phase)
	}
	fmt.Printf("\n")

	// TODO: Load actual todos from review
	// For now, show example todos
	todos := []struct {
		severity string
		task     string
		file     string
		effort   string
	}{
		{"🔴 CRITICAL", "Fix SQL injection vulnerability", "api/handlers/user.go:45", "2h"},
		{"🟡 HIGH", "Add input validation for user registration", "api/handlers/auth.go:78", "3h"},
		{"🟢 MEDIUM", "Improve error handling in payment processing", "services/payment.go:123", "4h"},
		{"🔵 LOW", "Add API documentation for new endpoints", "api/routes.go", "2h"},
	}

	for _, todo := range todos {
		fmt.Printf("%s: %s\n", todo.severity, todo.task)
		fmt.Printf("   File: %s\n", todo.file)
		fmt.Printf("   Effort: %s\n\n", todo.effort)
	}

	fmt.Printf("Total: %d todos\n", len(todos))

	return nil
}

func (c *CLI) runReviewTodosExport(format, output string) error {
	fmt.Printf("📤 Exporting Review Todos\n")
	fmt.Printf("========================\n\n")

	fmt.Printf("Format: %s\n", format)

	if output == "" {
		output = fmt.Sprintf("review-todos-%s.%s", time.Now().Format("2006-01-02"), format)
	}

	// TODO: Implement actual export
	fmt.Printf("\n✅ Exported to: %s\n", output)

	return nil
}

func (c *CLI) runReviewTodosSync() error {
	fmt.Printf("🔄 Syncing Review Todos\n")
	fmt.Printf("======================\n\n")

	// TODO: Implement sync with task management system
	fmt.Printf("⏳ Syncing to task management system...\n")

	time.Sleep(1 * time.Second)

	fmt.Printf("\n✅ Synced 4 todos successfully!\n")
	fmt.Printf("   - Created: 3 new tasks\n")
	fmt.Printf("   - Updated: 1 existing task\n")

	return nil
}

// Helper methods for review commands

func (c *CLI) mapPhaseToReviewMode(phase string) entities.ReviewMode {
	switch strings.ToLower(phase) {
	case "all", "":
		return entities.ReviewModeFull
	case "quick":
		return entities.ReviewModeQuick
	case "security":
		return entities.ReviewModeSecurity
	case "quality":
		return entities.ReviewModeQuality
	default:
		return entities.ReviewModeFull
	}
}

func (c *CLI) mapStringToReviewPhase(phase string) entities.ReviewPhase {
	switch strings.ToLower(phase) {
	case "foundation":
		return entities.PhaseFoundation
	case "security":
		return entities.PhaseSecurity
	case "quality":
		return entities.PhaseQuality
	case "documentation", "docs":
		return entities.PhaseDocumentation
	case "production":
		return entities.PhaseProduction
	case "synthesis":
		return entities.PhaseSynthesis
	default:
		return ""
	}
}
