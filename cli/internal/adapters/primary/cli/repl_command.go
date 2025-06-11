package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/adapters/secondary/tui"
)

// createREPLCommand creates the 'repl' command
func (c *CLI) createREPLCommand() *cobra.Command {
	var (
		httpPort int
		mode     string
	)

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Start interactive TUI dashboard for multi-repository intelligence",
		Long: `Start an interactive Terminal User Interface (TUI) with comprehensive dashboards.
		
The TUI provides:
- 📊 Real-time multi-repository dashboard
- 📈 Advanced analytics with interactive charts  
- 🔄 Pattern detection and workflow analysis
- 💡 Cross-repository insights and recommendations
- 📋 Interactive task management
- 🎯 AI-powered document generation (PRD/TRD)

Navigation:
- F1-F6: Switch between views (Command, Dashboard, Analytics, Tasks, Patterns, Insights)
- Tab/hjkl: Navigate within views
- Type 'help' in command mode for full instructions`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate mode
			replMode := tui.Interactive
			switch mode {
			case "interactive":
				replMode = tui.Interactive
			case "dashboard":
				replMode = tui.Dashboard
			case "analytics":
				replMode = tui.Analytics
			case "workflow":
				replMode = tui.Workflow
			case "debug":
				replMode = tui.Debug
			case "":
				replMode = tui.Interactive
			default:
				return fmt.Errorf("invalid mode: %s (valid: interactive, dashboard, analytics, workflow, debug)", mode)
			}

			// Start the TUI
			return tui.StartREPL(replMode, httpPort)
		},
	}

	// Add flags
	cmd.Flags().IntVarP(&httpPort, "port", "p", 0, "HTTP port for push notifications (0 to disable)")
	cmd.Flags().StringVarP(&mode, "mode", "m", "interactive", "TUI mode (interactive, dashboard, analytics, workflow, debug)")

	return cmd
}
