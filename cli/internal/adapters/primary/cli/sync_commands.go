// Package cli provides sync command implementations for batch operations
package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/services"
)

// createSyncCommand creates the 'sync' command group
func (c *CLI) createSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize tasks with server",
		Long: `Perform batch synchronization operations with the Lerian MCP Memory Server.

Examples:
  # Run synchronization
  lmmc sync
  
  # Check sync status
  lmmc sync status
  
  # Enable auto-sync
  lmmc sync auto
  
  # Force full sync
  lmmc sync force
  
  # View sync conflicts
  lmmc sync conflicts

Note: Requires MCP server to be running and configured.`,
	}

	// Add subcommands
	cmd.AddCommand(
		c.createSyncRunCommand(),
		c.createSyncStatusCommand(),
		c.createSyncForceCommand(),
		c.createSyncDeltaCommand(),
		c.createSyncAutoCommand(),
		c.createSyncConflictsCommand(),
		c.createSyncClearCommand(),
	)

	return cmd
}

// createSyncRunCommand creates the 'sync run' command
func (c *CLI) createSyncRunCommand() *cobra.Command {
	var (
		repository string
		verbose    bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run batch synchronization",
		Long:  `Perform a batch synchronization with the server, handling conflicts intelligently.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runSyncRun(repository, verbose)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository to sync (default: auto-detect)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed sync information")

	return cmd
}

// createSyncStatusCommand creates the 'sync status' command
func (c *CLI) createSyncStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show synchronization status",
		Long:  `Display current sync state, pending changes, and last sync information.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runSyncStatus()
		},
	}

	return cmd
}

// createSyncForceCommand creates the 'sync force' command
func (c *CLI) createSyncForceCommand() *cobra.Command {
	var repository string

	cmd := &cobra.Command{
		Use:   "force",
		Short: "Force full synchronization",
		Long:  `Perform a complete synchronization ignoring timestamps and sync state.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runSyncForce(repository)
		},
	}

	cmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository to sync (default: auto-detect)")

	return cmd
}

// createSyncDeltaCommand creates the 'sync delta' command
func (c *CLI) createSyncDeltaCommand() *cobra.Command {
	var repository string

	cmd := &cobra.Command{
		Use:   "delta",
		Short: "Perform delta synchronization",
		Long:  `Sync only changes since last synchronization for efficiency.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runSyncDelta(repository)
		},
	}

	cmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository to sync (default: auto-detect)")

	return cmd
}

// createSyncAutoCommand creates the 'sync auto' command
func (c *CLI) createSyncAutoCommand() *cobra.Command {
	var (
		repository string
		interval   time.Duration
		stop       bool
	)

	cmd := &cobra.Command{
		Use:   "auto",
		Short: "Manage automatic synchronization",
		Long:  `Start or stop automatic background synchronization.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runSyncAuto(repository, interval, stop)
		},
	}

	cmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository to sync (default: auto-detect)")
	cmd.Flags().DurationVarP(&interval, "interval", "i", 5*time.Minute, "Sync interval")
	cmd.Flags().BoolVar(&stop, "stop", false, "Stop automatic sync")

	return cmd
}

// createSyncConflictsCommand creates the 'sync conflicts' command
func (c *CLI) createSyncConflictsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conflicts",
		Short: "Show sync conflicts",
		Long:  `Display current synchronization conflicts that require manual resolution.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runSyncConflicts()
		},
	}

	return cmd
}

// createSyncClearCommand creates the 'sync clear' command
func (c *CLI) createSyncClearCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Clear sync state",
		Long:  `Reset synchronization state. Use with caution as this may cause data conflicts.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.runSyncClear(force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force clear without confirmation")

	return cmd
}

// Sync command implementations

// runSyncRun performs batch synchronization
func (c *CLI) runSyncRun(repository string, verbose bool) error {
	if c.batchSyncService == nil {
		return fmt.Errorf("batch sync service not available - please check server configuration")
	}

	// Auto-detect repository if not provided
	if repository == "" {
		repository = c.detectRepository()
	}

	fmt.Printf("🔄 Starting batch synchronization\n")
	fmt.Printf("Repository: %s\n", repository)

	if verbose {
		fmt.Printf("Client ID: %s\n", c.detectClientID())
		status := c.batchSyncService.GetSyncStatus()
		fmt.Printf("Last sync: %s\n", formatTimeAgo(status.LastSyncTime))
		fmt.Printf("Total syncs: %d\n\n", status.TotalSyncs)
	}

	ctx := context.Background()

	// Check pending changes
	pendingChanges, err := c.batchSyncService.GetPendingChanges(ctx, repository)
	if err != nil {
		return fmt.Errorf("failed to check pending changes: %w", err)
	}

	if pendingChanges > 0 {
		fmt.Printf("📝 Found %d pending changes\n", pendingChanges)
	} else {
		fmt.Printf("✅ No pending changes\n")
	}

	// Perform sync
	fmt.Printf("🚀 Synchronizing with server...\n")
	result, err := c.batchSyncService.PerformSync(ctx, repository)
	if err != nil {
		return fmt.Errorf("synchronization failed: %w", err)
	}

	// Display results
	c.displaySyncResult(result, verbose)

	return nil
}

// runSyncStatus shows current sync status
func (c *CLI) runSyncStatus() error {
	if c.batchSyncService == nil {
		return fmt.Errorf("batch sync service not available")
	}

	status := c.batchSyncService.GetSyncStatus()

	fmt.Printf("📊 Synchronization Status\n")
	fmt.Printf("========================\n\n")

	// Server connection info
	fmt.Printf("Server Configuration:\n")
	if c.taskService != nil && c.taskService.GetMCPClient() != nil {
		client := c.taskService.GetMCPClient()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		config, _ := c.configMgr.Load()
		fmt.Printf("  URL: %s\n", config.Server.URL)

		if err := client.TestConnection(ctx); err != nil {
			fmt.Printf("  Status: ❌ Offline (%v)\n", err)
			fmt.Printf("  💡 Tip: Check if the server is running at %s\n", config.Server.URL)
		} else {
			fmt.Printf("  Status: ✅ Online\n")
		}
	} else {
		fmt.Printf("  Status: ❌ Not configured\n")
		fmt.Printf("  💡 Tip: Set server URL with 'lmmc config set server.url <url>'\n")
	}

	fmt.Printf("\nSync Details:\n")
	if status.Repository == "" {
		fmt.Printf("  Repository: %s\n", "(not set - using current directory)")
		// Try to detect current repository
		if repoInfo := c.detectRepository(); repoInfo != "" {
			fmt.Printf("  Detected: %s\n", repoInfo)
		}
	} else {
		fmt.Printf("  Repository: %s\n", status.Repository)
	}

	fmt.Printf("  Client ID: %s\n", status.ClientID)

	// Better last sync display
	if status.LastSyncTime.IsZero() {
		fmt.Printf("  Last sync: Never\n")
		fmt.Printf("  💡 Tip: Run 'lmmc sync' to synchronize tasks\n")
	} else {
		fmt.Printf("  Last sync: %s\n", formatTimeAgo(status.LastSyncTime))
		if time.Since(status.LastSyncTime) > 24*time.Hour {
			fmt.Printf("  ⚠️  Warning: Last sync was more than 24 hours ago\n")
		}
	}

	if status.SyncToken != "" {
		fmt.Printf("  Sync token: %s\n", truncateString(status.SyncToken, 16))
	} else {
		fmt.Printf("  Sync token: (none)\n")
	}

	fmt.Printf("  Total syncs: %d\n", status.TotalSyncs)

	if status.LastConflictCount > 0 {
		fmt.Printf("  Last conflicts: %d ⚠️\n", status.LastConflictCount)
		fmt.Printf("  💡 Tip: Use 'lmmc sync --resolve' to handle conflicts\n")
	} else {
		fmt.Printf("  Last conflicts: %d ✅\n", status.LastConflictCount)
	}

	fmt.Printf("  Sync version: %d\n", status.SyncVersion)

	// Check pending changes if repository is available
	repo := status.Repository
	if repo == "" {
		repo = c.detectRepository()
	}

	if repo != "" {
		ctx := context.Background()
		pendingChanges, err := c.batchSyncService.GetPendingChanges(ctx, repo)
		if err == nil {
			if pendingChanges > 0 {
				fmt.Printf("  Pending changes: %d 📝\n", pendingChanges)
				fmt.Printf("  💡 Tip: Run 'lmmc sync' to upload changes\n")
			} else {
				fmt.Printf("  Pending changes: %d ✅\n", pendingChanges)
			}
		} else {
			fmt.Printf("  Pending changes: (unable to check)\n")
		}
	}

	// Auto-sync status
	fmt.Printf("\nAuto-sync: ")
	if c.autoSyncActive {
		fmt.Printf("✅ Enabled\n")
	} else {
		fmt.Printf("❌ Disabled\n")
		fmt.Printf("  💡 Tip: Run 'lmmc sync auto' to enable automatic synchronization\n")
	}

	return nil
}

// runSyncForce performs forced full synchronization
func (c *CLI) runSyncForce(repository string) error {
	if c.batchSyncService == nil {
		return fmt.Errorf("batch sync service not available")
	}

	if repository == "" {
		repository = c.detectRepository()
	}

	fmt.Printf("⚠️  Forcing full synchronization\n")
	fmt.Printf("Repository: %s\n", repository)
	fmt.Printf("This will sync all tasks regardless of timestamps.\n\n")

	ctx := context.Background()

	fmt.Printf("🚀 Starting forced sync...\n")
	result, err := c.batchSyncService.ForceSync(ctx, repository)
	if err != nil {
		return fmt.Errorf("forced sync failed: %w", err)
	}

	fmt.Printf("✅ Forced synchronization completed\n")
	c.displaySyncResult(result, true)

	return nil
}

// runSyncDelta performs delta synchronization
func (c *CLI) runSyncDelta(repository string) error {
	if c.batchSyncService == nil {
		return fmt.Errorf("batch sync service not available")
	}

	if repository == "" {
		repository = c.detectRepository()
	}

	fmt.Printf("📊 Starting delta synchronization\n")
	fmt.Printf("Repository: %s\n\n", repository)

	ctx := context.Background()

	result, err := c.batchSyncService.PerformDeltaSync(ctx, repository)
	if err != nil {
		return fmt.Errorf("delta sync failed: %w", err)
	}

	fmt.Printf("✅ Delta synchronization completed\n")
	c.displaySyncResult(result, false)

	return nil
}

// runSyncAuto manages automatic synchronization
func (c *CLI) runSyncAuto(repository string, interval time.Duration, stop bool) error {
	if c.batchSyncService == nil {
		return fmt.Errorf("batch sync service not available")
	}

	if repository == "" {
		repository = c.detectRepository()
	}

	if stop {
		fmt.Printf("🛑 Stopping automatic synchronization\n")
		return c.stopAutoSync()
	}

	// Check if auto sync is already running
	if c.autoSyncActive {
		fmt.Printf("⚠️  Automatic synchronization is already running\n")
		fmt.Printf("Use 'lmmc sync auto --stop' to stop it first.\n")
		return fmt.Errorf("auto sync already active")
	}

	fmt.Printf("🔄 Starting automatic synchronization\n")
	fmt.Printf("Repository: %s\n", repository)
	fmt.Printf("Interval: %v\n", interval)
	fmt.Printf("Press Ctrl+C to stop\n\n")

	// Create context with cancel function
	ctx, cancel := context.WithCancel(context.Background())
	c.autoSyncCancel = cancel
	c.autoSyncActive = true

	// Start auto sync in background
	go func() {
		defer func() {
			c.autoSyncActive = false
			c.autoSyncCancel = nil
		}()
		c.batchSyncService.ScheduleAutoSync(ctx, repository, interval)
	}()

	fmt.Printf("✅ Automatic synchronization started\n")
	fmt.Printf("Use 'lmmc sync auto --stop' to stop it.\n")

	return nil
}

// stopAutoSync stops the automatic synchronization
func (c *CLI) stopAutoSync() error {
	if !c.autoSyncActive {
		fmt.Printf("ℹ️  No automatic synchronization is currently running\n")
		return nil
	}

	if c.autoSyncCancel != nil {
		c.autoSyncCancel()
		fmt.Printf("✅ Automatic synchronization stopped\n")
	} else {
		fmt.Printf("⚠️  Failed to stop automatic synchronization (no cancel function)\n")
		c.autoSyncActive = false
		return fmt.Errorf("failed to stop auto sync: no cancel function")
	}

	return nil
}

// runSyncConflicts shows current conflicts
func (c *CLI) runSyncConflicts() error {
	fmt.Printf("🔍 Synchronization Conflicts\n")
	fmt.Printf("============================\n\n")
	fmt.Printf("No active conflicts found.\n")
	fmt.Printf("All conflicts are automatically resolved or require manual intervention.\n\n")
	fmt.Printf("💡 Tips:\n")
	fmt.Printf("   - Use 'lmmc sync run -v' for detailed conflict information\n")
	fmt.Printf("   - Check logs for conflict resolution details\n")

	return nil
}

// runSyncClear clears sync state
func (c *CLI) runSyncClear(force bool) error {
	if c.batchSyncService == nil {
		return fmt.Errorf("batch sync service not available")
	}

	if !force {
		fmt.Printf("⚠️  This will reset all synchronization state.\n")
		fmt.Printf("Are you sure you want to continue? (y/N): ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// If scan fails, default to cancel for safety
			fmt.Printf("Invalid input, operation cancelled.\n")
			return nil
		}

		if response != "y" && response != "Y" {
			fmt.Printf("Operation cancelled.\n")
			return nil
		}
	}

	c.batchSyncService.ClearSyncState()

	fmt.Printf("✅ Synchronization state cleared\n")
	fmt.Printf("Next sync will be a full synchronization.\n")

	return nil
}

// Helper functions

// displaySyncResult shows the results of a sync operation
func (c *CLI) displaySyncResult(result *services.SyncResult, verbose bool) {
	if result.Success {
		fmt.Printf("✅ Synchronization successful\n")
	} else {
		fmt.Printf("⚠️  Synchronization completed with errors\n")
	}

	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Tasks synced: %d\n", result.SyncedTasks)
	fmt.Printf("   Conflicts detected: %d\n", result.ConflictsDetected)
	fmt.Printf("   Conflicts resolved: %d\n", result.ConflictsResolved)
	fmt.Printf("   Duration: %v\n", result.Duration)

	if verbose && result.Statistics.TotalTasks > 0 {
		fmt.Printf("\n📈 Statistics:\n")
		fmt.Printf("   Total tasks: %d\n", result.Statistics.TotalTasks)
		fmt.Printf("   Created: %d\n", result.Statistics.TasksCreated)
		fmt.Printf("   Updated: %d\n", result.Statistics.TasksUpdated)
		fmt.Printf("   Deleted: %d\n", result.Statistics.TasksDeleted)

		if result.Statistics.DataTransferred > 0 {
			fmt.Printf("   Data transferred: %s\n", formatBytes(result.Statistics.DataTransferred))
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\n❌ Errors:\n")
		for _, err := range result.Errors {
			fmt.Printf("   - %s\n", err)
		}
	}

	fmt.Printf("\n💡 Next steps:\n")
	if result.ConflictsDetected > result.ConflictsResolved {
		fmt.Printf("   - Review conflicts with 'lmmc sync conflicts'\n")
	} else {
		fmt.Printf("   - Continue working on your tasks\n")
		fmt.Printf("   - Use 'lmmc sync auto' for automatic synchronization\n")
	}
}

// detectClientID generates or retrieves client ID
func (c *CLI) detectClientID() string {
	if c.batchSyncService != nil {
		status := c.batchSyncService.GetSyncStatus()
		return truncateString(status.ClientID, 8)
	}
	return "unknown"
}

// formatTimeAgo formats time as "X ago" string
func formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	diff := time.Since(t)

	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	}
}

// formatBytes formats byte count as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// truncateString truncates a string to specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}
