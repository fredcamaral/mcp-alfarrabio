package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/constants"
)

// createWorkflowStatusCommand creates the 'workflow status' command
func (c *CLI) createWorkflowStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show workflow execution status",
		Long:  `Display the current status of your development workflow, including completed steps and next actions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runWorkflowStatus()
		},
	}

	return cmd
}

// runWorkflowStatus displays the current workflow status
func (c *CLI) runWorkflowStatus() error {
	c.displayWorkflowHeader()

	// Load session
	session, err := c.loadSession()
	if err != nil || session.ID == "" {
		c.displayNoActiveSession()
		return nil
	}

	// Display session info
	c.displaySessionInfo(session)

	// Check workflow progress and display status
	status := c.getWorkflowStatus()
	c.displayCompletedSteps(session)
	c.displayNextSteps(status)
	c.displayWorkflowFiles(session)
	c.displayWorkflowCommands()

	return nil
}

// displayWorkflowHeader shows the main header
func (c *CLI) displayWorkflowHeader() {
	fmt.Printf("📊 Workflow Status\n")
	fmt.Printf("=================\n\n")
}

// displayNoActiveSession shows message when no session exists
func (c *CLI) displayNoActiveSession() {
	fmt.Printf("❌ No active workflow session\n\n")
	fmt.Printf("💡 Start a new workflow with:\n")
	fmt.Printf("   - lmmc prd create \"your feature description\"\n")
	fmt.Printf("   - lmmc workflow start \"your feature\"\n")
}

// displaySessionInfo shows current session details
func (c *CLI) displaySessionInfo(session *SessionData) {
	fmt.Printf("📌 Session ID: %s\n", session.ID)
	fmt.Printf("🕐 Started: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("🔄 Last updated: %s ago\n\n", formatDuration(time.Since(session.UpdatedAt)))
}

// displayCompletedSteps shows completed workflow steps
func (c *CLI) displayCompletedSteps(session *SessionData) int {
	fmt.Printf("✅ Completed Steps:\n")

	steps := []struct {
		key   string
		label string
	}{
		{"prd_file", "PRD created"},
		{"trd_file", "TRD generated"},
		{"tasks_file", "Tasks generated"},
		{"subtasks_file", "Sub-tasks generated"},
	}

	completedSteps := 0
	for i, step := range steps {
		if file := session.Values[step.key]; file != "" && fileExists(file) {
			fmt.Printf("   %d. %s: %s\n", i+1, step.label, filepath.Base(file))
			completedSteps++
		}
	}

	if completedSteps == 0 {
		fmt.Printf("   (No steps completed yet)\n")
	}

	fmt.Printf("\n")
	return completedSteps
}

// displayNextSteps shows what to do next based on current status
func (c *CLI) displayNextSteps(status string) {
	fmt.Printf("💡 Next Steps:\n")

	switch status {
	case constants.WorkflowStatusReadyToStart:
		fmt.Printf("   → Run: lmmc prd create \"your feature description\"\n")
		fmt.Printf("   → Or: lmmc workflow start \"your feature\"\n")
	case constants.WorkflowStatusReadyForTRD:
		fmt.Printf("   → Run: lmmc trd create\n")
		fmt.Printf("   → The PRD will be auto-detected from your session\n")
	case constants.WorkflowStatusReadyForTasks:
		fmt.Printf("   → Run: lmmc tasks generate\n")
		fmt.Printf("   → Both PRD and TRD will be auto-detected\n")
	case constants.WorkflowStatusReadyForSubtasks:
		fmt.Printf("   → Run: lmmc subtasks generate MT-001\n")
		fmt.Printf("   → Or: lmmc workflow continue\n")
	case constants.WorkflowStatusReadyForImplementation:
		fmt.Printf("   ✨ All documents generated! Ready to start coding.\n")
		fmt.Printf("   → Run: lmmc add --from-task MT-001\n")
		fmt.Printf("   → Or: lmmc review phase foundation\n")
	}
}

// displayWorkflowFiles shows current workflow files
func (c *CLI) displayWorkflowFiles(session *SessionData) {
	fmt.Printf("\n📁 Workflow Files:\n")

	files := []struct {
		key   string
		label string
	}{
		{"prd_file", "PRD"},
		{"trd_file", "TRD"},
		{"tasks_file", "Tasks"},
		{"subtasks_file", "Sub-tasks"},
	}

	for _, file := range files {
		if value := session.Values[file.key]; value != "" {
			fmt.Printf("   %s: %s\n", file.label, value)
		}
	}
}

// displayWorkflowCommands shows available workflow commands
func (c *CLI) displayWorkflowCommands() {
	fmt.Printf("\n🔧 Workflow Commands:\n")
	fmt.Printf("   - lmmc workflow continue    # Resume from current step\n")
	fmt.Printf("   - lmmc workflow restart     # Start over\n")
	fmt.Printf("   - lmmc workflow clear       # Clear session\n")
}

// Helper functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1f hours", d.Hours())
	}
	return fmt.Sprintf("%.1f days", d.Hours()/24)
}
