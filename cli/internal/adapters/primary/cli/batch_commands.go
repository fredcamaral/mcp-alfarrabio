package cli

import (
	"errors"
	"fmt"
	"strings"

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
				return c.processDoneTasks(cmd, args, actualTime)
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

		return c.processTagTasks(cmd, taskIDs, tags, removeTags)
	}

	return cmd
}

// processDoneTasks handles the batch completion of tasks with reduced nesting
func (c *CLI) processDoneTasks(cmd *cobra.Command, taskIDs []string, actualTime int) error {
	successCount := 0
	failCount := 0

	for _, taskID := range taskIDs {
		success := c.processSingleDoneTask(cmd, taskID, actualTime)
		if success {
			successCount++
		} else {
			failCount++
		}
	}

	// Summary
	fmt.Fprintf(cmd.OutOrStdout(), "\nSummary: %d completed, %d failed\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("%d tasks failed", failCount)
	}
	return nil
}

// processSingleDoneTask handles completion of a single task, returns true if successful
func (c *CLI) processSingleDoneTask(cmd *cobra.Command, taskID string, actualTime int) bool {
	// Resolve task ID from short form
	fullTaskID, err := c.resolveTaskID(c.getContext(), taskID, "")
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "❌ Failed to resolve task %s: %v\n", taskID, err)
		return false
	}

	// Update status
	err = c.taskService.UpdateTaskStatus(c.getContext(), fullTaskID, entities.StatusCompleted)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "❌ Failed to complete task %s: %v\n", taskID, err)
		return false
	}

	// Update actual time if provided
	c.updateActualTimeIfProvided(fullTaskID, actualTime)

	// Display success message
	displayID := c.formatDisplayID(fullTaskID)
	fmt.Fprintf(cmd.OutOrStdout(), "✅ Task %s marked as completed\n", displayID)
	return true
}

// updateActualTimeIfProvided updates the actual time for a task if actualTime > 0
func (c *CLI) updateActualTimeIfProvided(fullTaskID string, actualTime int) {
	if actualTime <= 0 {
		return
	}

	updates := services.TaskUpdates{
		ActualMins: &actualTime,
	}
	if err := c.taskService.UpdateTask(c.getContext(), fullTaskID, &updates); err != nil {
		c.logger.Warn("failed to update actual time",
			"task_id", fullTaskID,
			"error", err)
	}
}

// formatDisplayID formats a task ID for display (truncates if too long)
func (c *CLI) formatDisplayID(fullTaskID string) string {
	if len(fullTaskID) > 8 {
		return fullTaskID[:8]
	}
	return fullTaskID
}

// processTagTasks handles batch tagging/untagging of tasks with reduced nesting
func (c *CLI) processTagTasks(cmd *cobra.Command, taskIDs []string, tags []string, removeTags bool) error {
	successCount := 0
	failCount := 0

	for _, taskID := range taskIDs {
		success := c.processSingleTagTask(cmd, taskID, tags, removeTags)
		if success {
			successCount++
		} else {
			failCount++
		}
	}

	// Summary
	action := c.getTagAction(removeTags)
	fmt.Fprintf(cmd.OutOrStdout(), "\nSummary: %d tasks %s, %d failed\n", successCount, action, failCount)

	if failCount > 0 {
		return fmt.Errorf("%d tasks failed", failCount)
	}
	return nil
}

// processSingleTagTask handles tagging/untagging of a single task, returns true if successful
func (c *CLI) processSingleTagTask(cmd *cobra.Command, taskID string, tags []string, removeTags bool) bool {
	// Resolve task ID from short form
	fullTaskID, err := c.resolveTaskID(c.getContext(), taskID, "")
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "❌ Failed to resolve task %s: %v\n", taskID, err)
		return false
	}

	// Build and apply updates
	updates := c.buildTagUpdates(tags, removeTags)
	err = c.taskService.UpdateTask(c.getContext(), fullTaskID, &updates)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "❌ Failed to update task %s: %v\n", taskID, err)
		return false
	}

	// Display success message
	displayID := c.formatDisplayID(fullTaskID)
	c.displayTagSuccessMessage(cmd, displayID, removeTags)
	return true
}

// buildTagUpdates creates TaskUpdates for tag operations
func (c *CLI) buildTagUpdates(tags []string, removeTags bool) services.TaskUpdates {
	updates := services.TaskUpdates{}
	if removeTags {
		updates.RemoveTags = tags
	} else {
		updates.AddTags = tags
	}
	return updates
}

// getTagAction returns the appropriate action string for tag operations
func (c *CLI) getTagAction(removeTags bool) string {
	if removeTags {
		return "untagged"
	}
	return "tagged"
}

// displayTagSuccessMessage displays the appropriate success message for tag operations
func (c *CLI) displayTagSuccessMessage(cmd *cobra.Command, displayID string, removeTags bool) {
	if removeTags {
		fmt.Fprintf(cmd.OutOrStdout(), "✅ Removed tags from task %s\n", displayID)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "✅ Added tags to task %s\n", displayID)
	}
}
