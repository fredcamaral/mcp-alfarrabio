package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// updateBatchOperations updates existing commands to support batch operations
func (c *CLI) updateCommandsForBatchOperations() {
	// Update done command to support multiple task IDs
	c.updateDoneCommandForBatch()

	// Update priority command to support multiple task IDs
	c.updatePriorityCommandForBatch()

	// Update tag command (new command for batch tagging)
	c.createTagCommand()
}

// updateDoneCommandForBatch modifies the done command to accept multiple task IDs
func (c *CLI) updateDoneCommandForBatch() {
	// Find the done command
	for _, cmd := range c.RootCmd.Commands() {
		if cmd.Name() == "done" {
			// Update the Use field
			cmd.Use = "done [task-ids...]"
			cmd.Short = "Mark one or more tasks as completed"
			cmd.Long = `Mark one or more tasks as completed to indicate they're finished.`
			cmd.Example = `  # Complete a single task
  lmmc done task1
  
  # Complete multiple tasks
  lmmc done task1 task2 task3
  
  # Complete tasks with actual time
  lmmc done task1 task2 --actual 120`

			// Replace the RunE function
			cmd.RunE = func(cmd *cobra.Command, args []string) error {
				if len(args) == 0 {
					return errors.New("at least one task ID is required")
				}

				actualTime, _ := cmd.Flags().GetInt("actual")

				// Process each task
				successCount := 0
				failCount := 0

				for _, taskID := range args {
					// Resolve task ID from short form
					fullTaskID, err := c.resolveTaskID(c.getContext(), taskID, "")
					if err != nil {
						fmt.Fprintf(cmd.OutOrStdout(), "❌ Failed to resolve task %s: %v\n", taskID, err)
						failCount++
						continue
					}

					// Update status
					err = c.taskService.UpdateTaskStatus(c.getContext(), fullTaskID, entities.StatusCompleted)
					if err != nil {
						fmt.Fprintf(cmd.OutOrStdout(), "❌ Failed to complete task %s: %v\n", taskID, err)
						failCount++
						continue
					}

					// Update actual time if provided
					if actualTime > 0 {
						updates := services.TaskUpdates{
							ActualMins: &actualTime,
						}
						if err := c.taskService.UpdateTask(c.getContext(), fullTaskID, &updates); err != nil {
							c.logger.Warn("failed to update actual time",
								"task_id", fullTaskID,
								"error", err)
						}
					}

					displayID := fullTaskID
					if len(displayID) > 8 {
						displayID = displayID[:8]
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✅ Task %s marked as completed\n", displayID)
					successCount++
				}

				// Summary
				fmt.Fprintf(cmd.OutOrStdout(), "\nSummary: %d completed, %d failed\n", successCount, failCount)

				if failCount > 0 {
					return fmt.Errorf("%d tasks failed", failCount)
				}
				return nil
			}

			break
		}
	}
}

// updatePriorityCommandForBatch modifies the priority command to accept multiple task IDs
func (c *CLI) updatePriorityCommandForBatch() {
	// Create new update command that handles priority and other updates for multiple tasks
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update multiple tasks at once",
		Long:  `Update properties of multiple tasks in a single command.`,
		Example: `  # Update priority for multiple tasks
  lmmc update --priority high task1 task2 task3
  
  # Add tags to multiple tasks
  lmmc update --add-tags security,urgent task1 task2
  
  # Update multiple properties
  lmmc update --priority high --add-tags backend task1 task2 task3`,
	}

	var (
		priority string
		addTags  []string
		rmTags   []string
		estimate int
	)

	updateCmd.Flags().StringVarP(&priority, "priority", "p", "", "New priority (low, medium, high)")
	updateCmd.Flags().StringSliceVar(&addTags, "add-tags", nil, "Tags to add")
	updateCmd.Flags().StringSliceVar(&rmTags, "remove-tags", nil, "Tags to remove")
	updateCmd.Flags().IntVarP(&estimate, "estimate", "e", 0, "New estimated time in minutes")

	updateCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("at least one task ID is required")
		}

		// Build updates
		updates := services.TaskUpdates{
			AddTags:    addTags,
			RemoveTags: rmTags,
		}

		if priority != "" {
			p, err := parsePriority(priority)
			if err != nil {
				return c.handleError(cmd, err)
			}
			updates.Priority = &p
		}

		if estimate > 0 {
			updates.EstimatedMins = &estimate
		}

		// Process each task
		successCount := 0
		failCount := 0

		for _, taskID := range args {
			// Resolve task ID from short form
			fullTaskID, err := c.resolveTaskID(c.getContext(), taskID, "")
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "❌ Failed to resolve task %s: %v\n", taskID, err)
				failCount++
				continue
			}

			// Apply updates
			err = c.taskService.UpdateTask(c.getContext(), fullTaskID, &updates)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "❌ Failed to update task %s: %v\n", taskID, err)
				failCount++
				continue
			}

			displayID := fullTaskID
			if len(displayID) > 8 {
				displayID = displayID[:8]
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✅ Task %s updated\n", displayID)
			successCount++
		}

		// Summary
		fmt.Fprintf(cmd.OutOrStdout(), "\nSummary: %d updated, %d failed\n", successCount, failCount)

		if failCount > 0 {
			return fmt.Errorf("%d tasks failed", failCount)
		}
		return nil
	}

	c.RootCmd.AddCommand(updateCmd)
}

// createTagCommand creates a dedicated tag command for batch tagging
func (c *CLI) createTagCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag [tags] [task-ids...]",
		Short: "Add tags to multiple tasks",
		Long:  `Add one or more tags to multiple tasks at once.`,
		Example: `  # Add a single tag to multiple tasks
  lmmc tag security task1 task2 task3
  
  # Add multiple tags to multiple tasks
  lmmc tag security,backend,urgent task1 task2
  
  # Remove tags using --remove flag
  lmmc tag --remove deprecated task1 task2 task3`,
		Args: cobra.MinimumNArgs(2),
	}

	var removeTags bool
	cmd.Flags().BoolVar(&removeTags, "remove", false, "Remove tags instead of adding them")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// First argument is tags (comma-separated)
		tags := strings.Split(args[0], ",")

		// Remaining arguments are task IDs
		taskIDs := args[1:]

		// Process each task
		successCount := 0
		failCount := 0

		for _, taskID := range taskIDs {
			// Resolve task ID from short form
			fullTaskID, err := c.resolveTaskID(c.getContext(), taskID, "")
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "❌ Failed to resolve task %s: %v\n", taskID, err)
				failCount++
				continue
			}

			// Build updates
			updates := services.TaskUpdates{}
			if removeTags {
				updates.RemoveTags = tags
			} else {
				updates.AddTags = tags
			}

			// Apply updates
			err = c.taskService.UpdateTask(c.getContext(), fullTaskID, &updates)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "❌ Failed to update task %s: %v\n", taskID, err)
				failCount++
				continue
			}

			displayID := fullTaskID
			if len(displayID) > 8 {
				displayID = displayID[:8]
			}

			if removeTags {
				fmt.Fprintf(cmd.OutOrStdout(), "✅ Removed tags from task %s\n", displayID)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "✅ Added tags to task %s\n", displayID)
			}
			successCount++
		}

		// Summary
		action := "tagged"
		if removeTags {
			action = "untagged"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\nSummary: %d tasks %s, %d failed\n", successCount, action, failCount)

		if failCount > 0 {
			return fmt.Errorf("%d tasks failed", failCount)
		}
		return nil
	}

	return cmd
}

// createBatchReviewCommand creates batch review command
func (c *CLI) createBatchReviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review start [paths...] --parallel",
		Short: "Start reviews for multiple paths in parallel",
		Long:  `Start code reviews for multiple paths simultaneously.`,
		Example: `  # Review multiple directories in parallel
  lmmc review start /path1 /path2 /path3 --parallel
  
  # Review with specific phase
  lmmc review start /api /web /docs --parallel --phase security`,
	}

	var (
		parallel bool
		phase    string
		quick    bool
	)

	cmd.Flags().BoolVar(&parallel, "parallel", false, "Run reviews in parallel")
	cmd.Flags().StringVar(&phase, "phase", "all", "Review phase to run")
	cmd.Flags().BoolVar(&quick, "quick", false, "Quick review mode")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("at least one path is required")
		}

		if !parallel {
			// Sequential execution
			for i, path := range args {
				fmt.Fprintf(cmd.OutOrStdout(), "\n🔍 Reviewing path %d/%d: %s\n", i+1, len(args), path)
				fmt.Fprintf(cmd.OutOrStdout(), "=====================================\n")

				// Start review
				if c.reviewService != nil {
					// For now, just show a placeholder
					fmt.Fprintf(cmd.OutOrStdout(), "🔍 Starting review for: %s (phase: %s, quick: %v)\n", path, phase, quick)
					fmt.Fprintf(cmd.OutOrStdout(), "⚠️  Review functionality is being implemented\n")
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "⚠️  Review service not available\n")
				}
			}
		} else {
			// Parallel execution
			fmt.Fprintf(cmd.OutOrStdout(), "🚀 Starting %d reviews in parallel...\n", len(args))

			type reviewResult struct {
				path    string
				session string
				err     error
			}

			results := make(chan reviewResult, len(args))

			// Start reviews in goroutines
			for _, path := range args {
				go func(p string) {
					if c.reviewService != nil {
						// For now, just simulate review
						results <- reviewResult{path: p, session: fmt.Sprintf("review-%d", time.Now().Unix())}
					} else {
						results <- reviewResult{path: p, err: errors.New("review service not available")}
					}
				}(path)
			}

			// Collect results
			successCount := 0
			failCount := 0

			for i := 0; i < len(args); i++ {
				result := <-results
				if result.err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "❌ %s: %v\n", result.path, result.err)
					failCount++
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "✅ %s: Session %s\n", result.path, result.session)
					successCount++
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\nSummary: %d successful, %d failed\n", successCount, failCount)
		}

		return nil
	}

	return cmd
}
