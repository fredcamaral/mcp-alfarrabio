package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// createAddCommand creates the 'add' command
func (c *CLI) createAddCommand() *cobra.Command {
	var (
		priority  string
		tags      []string
		estimated int
	)

	cmd := &cobra.Command{
		Use:   "add [task description]",
		Short: "Create a new task",
		Long:  `Create a new task with the specified description in the current repository.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content := strings.Join(args, " ")

			// Build task options
			var options []services.TaskOption

			if priority != "" {
				p, err := parsePriority(priority)
				if err != nil {
					return c.handleError(cmd, err)
				}
				options = append(options, services.WithPriority(p))
			}

			if len(tags) > 0 {
				options = append(options, services.WithTags(tags...))
			}

			if estimated > 0 {
				options = append(options, services.WithEstimatedTime(estimated))
			}

			// Create task
			task, err := c.taskService.CreateTask(c.getContext(), content, options...)
			if err != nil {
				return c.handleError(cmd, err)
			}

			// Format output
			formatter := c.getOutputFormatter(cmd)
			return formatter.FormatTask(task)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&priority, "priority", "p", "", "Task priority (low, medium, high)")
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "Task tags")
	cmd.Flags().IntVarP(&estimated, "estimate", "e", 0, "Estimated time in minutes")

	return cmd
}

// createListCommand creates the 'list' command
func (c *CLI) createListCommand() *cobra.Command {
	var (
		status     string
		priority   string
		tags       []string
		repository string
		search     string
		all        bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List tasks with filtering options",
		Long:    `List tasks with optional filtering by status, priority, tags, or repository.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Build filters
			filters := ports.TaskFilters{
				Repository: repository,
				Tags:       tags,
				Search:     search,
			}

			// Parse status filter
			if status != "" {
				s, err := parseStatus(status)
				if err != nil {
					return c.handleError(cmd, err)
				}
				filters.Status = &s
			}

			// Parse priority filter
			if priority != "" {
				p, err := parsePriority(priority)
				if err != nil {
					return c.handleError(cmd, err)
				}
				filters.Priority = &p
			}

			// List tasks
			tasks, err := c.taskService.ListTasks(c.getContext(), &filters)
			if err != nil {
				return c.handleError(cmd, err)
			}

			// Format output
			formatter := c.getOutputFormatter(cmd)
			return formatter.FormatTaskList(tasks)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status (pending, in_progress, completed, cancelled)")
	cmd.Flags().StringVarP(&priority, "priority", "p", "", "Filter by priority (low, medium, high)")
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "Filter by tags")
	cmd.Flags().StringVarP(&repository, "repository", "r", "", "Filter by repository")
	cmd.Flags().StringVar(&search, "search", "", "Search in task content")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Show all tasks including completed")

	return cmd
}

// createStartCommand creates the 'start' command
func (c *CLI) createStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start [task-id]",
		Short: "Mark a task as in progress",
		Long:  `Mark a task as in progress to indicate you're currently working on it.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			err := c.taskService.UpdateTaskStatus(c.getContext(), taskID, entities.StatusInProgress)
			if err != nil {
				return c.handleError(cmd, err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Task %s marked as in progress\n", taskID[:8])
			return nil
		},
	}
}

// createDoneCommand creates the 'done' command
func (c *CLI) createDoneCommand() *cobra.Command {
	var actualTime int

	cmd := &cobra.Command{
		Use:   "done [task-id]",
		Short: "Mark a task as completed",
		Long:  `Mark a task as completed to indicate it's finished.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// Update status
			err := c.taskService.UpdateTaskStatus(c.getContext(), taskID, entities.StatusCompleted)
			if err != nil {
				return c.handleError(cmd, err)
			}

			// Update actual time if provided
			if actualTime > 0 {
				updates := services.TaskUpdates{
					ActualMins: &actualTime,
				}
				if err := c.taskService.UpdateTask(c.getContext(), taskID, &updates); err != nil {
					c.logger.Warn("failed to update actual time",
						"task_id", taskID,
						"error", err)
				}
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Task %s marked as completed\n", taskID[:8])
			return nil
		},
	}

	cmd.Flags().IntVarP(&actualTime, "actual", "a", 0, "Actual time spent in minutes")

	return cmd
}

// createCancelCommand creates the 'cancel' command
func (c *CLI) createCancelCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel [task-id]",
		Short: "Cancel a task",
		Long:  `Cancel a task to indicate it won't be completed.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			err := c.taskService.UpdateTaskStatus(c.getContext(), taskID, entities.StatusCancelled)
			if err != nil {
				return c.handleError(cmd, err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Task %s cancelled\n", taskID[:8])
			return nil
		},
	}
}

// createEditCommand creates the 'edit' command
func (c *CLI) createEditCommand() *cobra.Command {
	var (
		content  string
		addTags  []string
		rmTags   []string
		estimate int
	)

	cmd := &cobra.Command{
		Use:   "edit [task-id]",
		Short: "Edit task details",
		Long:  `Edit task content, tags, or time estimates.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// Build updates
			updates := services.TaskUpdates{
				AddTags:    addTags,
				RemoveTags: rmTags,
			}

			if content != "" {
				updates.Content = &content
			}

			if estimate > 0 {
				updates.EstimatedMins = &estimate
			}

			// Apply updates
			err := c.taskService.UpdateTask(c.getContext(), taskID, &updates)
			if err != nil {
				return c.handleError(cmd, err)
			}

			// Get and display updated task
			task, err := c.taskService.GetTask(c.getContext(), taskID)
			if err != nil {
				return c.handleError(cmd, err)
			}

			formatter := c.getOutputFormatter(cmd)
			return formatter.FormatTask(task)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&content, "content", "c", "", "New task content")
	cmd.Flags().StringSliceVar(&addTags, "add-tags", nil, "Tags to add")
	cmd.Flags().StringSliceVar(&rmTags, "remove-tags", nil, "Tags to remove")
	cmd.Flags().IntVarP(&estimate, "estimate", "e", 0, "New estimated time in minutes")

	return cmd
}

// createPriorityCommand creates the 'priority' command
func (c *CLI) createPriorityCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "priority [task-id] [priority]",
		Short: "Update task priority",
		Long:  `Update the priority of a task (low, medium, high).`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			priority, err := parsePriority(args[1])
			if err != nil {
				return c.handleError(cmd, err)
			}

			updates := services.TaskUpdates{
				Priority: &priority,
			}

			err = c.taskService.UpdateTask(c.getContext(), taskID, &updates)
			if err != nil {
				return c.handleError(cmd, err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Task %s priority updated to %s\n", taskID[:8], priority)
			return nil
		},
	}
}

// createDeleteCommand creates the 'delete' command
func (c *CLI) createDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete [task-id]",
		Aliases: []string{"rm"},
		Short:   "Delete a task",
		Long:    `Delete a task permanently. This action cannot be undone.`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			// Confirm deletion if not forced
			if !force {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Delete task %s? [y/N]: ", taskID[:8])

				var response string
				if _, err := fmt.Scanln(&response); err != nil {
					// If we can't read input, treat as cancelled
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
					return nil
				}

				if !strings.EqualFold(response, "y") {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled")
					return nil
				}
			}

			err := c.taskService.DeleteTask(c.getContext(), taskID)
			if err != nil {
				return c.handleError(cmd, err)
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Task %s deleted\n", taskID[:8])
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

// createStatsCommand creates the 'stats' command
func (c *CLI) createStatsCommand() *cobra.Command {
	var repository string

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show task statistics",
		Long:  `Display statistics about tasks in the current or specified repository.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			stats, err := c.taskService.GetRepositoryStats(c.getContext(), repository)
			if err != nil {
				return c.handleError(cmd, err)
			}

			formatter := c.getOutputFormatter(cmd)
			return formatter.FormatStats(&stats)
		},
	}

	cmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository to show stats for")

	return cmd
}

// createSearchCommand creates the 'search' command
func (c *CLI) createSearchCommand() *cobra.Command {
	var (
		repository string
		status     string
		priority   string
		tags       []string
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search tasks",
		Long:  `Search for tasks containing the specified query in their content or tags.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			// Build filters
			filters := ports.TaskFilters{
				Repository: repository,
				Tags:       tags,
			}

			// Parse status filter
			if status != "" {
				s, err := parseStatus(status)
				if err != nil {
					return c.handleError(cmd, err)
				}
				filters.Status = &s
			}

			// Parse priority filter
			if priority != "" {
				p, err := parsePriority(priority)
				if err != nil {
					return c.handleError(cmd, err)
				}
				filters.Priority = &p
			}

			// Search tasks
			tasks, err := c.taskService.SearchTasks(c.getContext(), query, &filters)
			if err != nil {
				return c.handleError(cmd, err)
			}

			// Format output
			formatter := c.getOutputFormatter(cmd)
			return formatter.FormatTaskList(tasks)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&repository, "repository", "r", "", "Search in specific repository")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status")
	cmd.Flags().StringVarP(&priority, "priority", "p", "", "Filter by priority")
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "Filter by tags")

	return cmd
}
