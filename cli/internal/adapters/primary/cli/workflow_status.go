package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
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
	fmt.Printf("📊 Workflow Status\n")
	fmt.Printf("=================\n\n")

	// Load session
	session, err := c.loadSession()
	if err != nil || session.ID == "" {
		fmt.Printf("❌ No active workflow session\n\n")
		fmt.Printf("💡 Start a new workflow with:\n")
		fmt.Printf("   - lmmc prd create \"your feature description\"\n")
		fmt.Printf("   - lmmc workflow start \"your feature\"\n")
		return nil
	}

	// Display session info
	fmt.Printf("📌 Session ID: %s\n", session.ID)
	fmt.Printf("🕐 Started: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("🔄 Last updated: %s ago\n\n", formatDuration(time.Since(session.UpdatedAt)))

	// Check workflow progress
	status := c.getWorkflowStatus()

	// Display completed steps
	fmt.Printf("✅ Completed Steps:\n")
	completedSteps := 0

	if prdFile := session.Values["prd_file"]; prdFile != "" {
		if fileExists(prdFile) {
			fmt.Printf("   1. PRD created: %s\n", filepath.Base(prdFile))
			completedSteps++
		}
	}

	if trdFile := session.Values["trd_file"]; trdFile != "" {
		if fileExists(trdFile) {
			fmt.Printf("   2. TRD generated: %s\n", filepath.Base(trdFile))
			completedSteps++
		}
	}

	if tasksFile := session.Values["tasks_file"]; tasksFile != "" {
		if fileExists(tasksFile) {
			fmt.Printf("   3. Tasks generated: %s\n", filepath.Base(tasksFile))
			completedSteps++
		}
	}

	if subtasksFile := session.Values["subtasks_file"]; subtasksFile != "" {
		if fileExists(subtasksFile) {
			fmt.Printf("   4. Sub-tasks generated: %s\n", filepath.Base(subtasksFile))
			completedSteps++
		}
	}

	if completedSteps == 0 {
		fmt.Printf("   (No steps completed yet)\n")
	}

	fmt.Printf("\n")

	// Display next steps based on status
	fmt.Printf("💡 Next Steps:\n")

	switch status {
	case "ready_to_start":
		fmt.Printf("   → Run: lmmc prd create \"your feature description\"\n")
		fmt.Printf("   → Or: lmmc workflow start \"your feature\"\n")

	case "ready_for_trd":
		fmt.Printf("   → Run: lmmc trd create\n")
		fmt.Printf("   → The PRD will be auto-detected from your session\n")

	case "ready_for_tasks":
		fmt.Printf("   → Run: lmmc tasks generate\n")
		fmt.Printf("   → Both PRD and TRD will be auto-detected\n")

	case "ready_for_subtasks":
		fmt.Printf("   → Run: lmmc subtasks generate MT-001\n")
		fmt.Printf("   → Or: lmmc workflow continue\n")

	case "ready_for_implementation":
		fmt.Printf("   ✨ All documents generated! Ready to start coding.\n")
		fmt.Printf("   → Run: lmmc add --from-task MT-001\n")
		fmt.Printf("   → Or: lmmc review phase foundation\n")
	}

	fmt.Printf("\n📁 Workflow Files:\n")
	if session.Values["prd_file"] != "" {
		fmt.Printf("   PRD: %s\n", session.Values["prd_file"])
	}
	if session.Values["trd_file"] != "" {
		fmt.Printf("   TRD: %s\n", session.Values["trd_file"])
	}
	if session.Values["tasks_file"] != "" {
		fmt.Printf("   Tasks: %s\n", session.Values["tasks_file"])
	}
	if session.Values["subtasks_file"] != "" {
		fmt.Printf("   Sub-tasks: %s\n", session.Values["subtasks_file"])
	}

	fmt.Printf("\n🔧 Workflow Commands:\n")
	fmt.Printf("   - lmmc workflow continue    # Resume from current step\n")
	fmt.Printf("   - lmmc workflow restart     # Start over\n")
	fmt.Printf("   - lmmc workflow clear       # Clear session\n")

	return nil
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
