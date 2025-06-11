package cli

import (
	"fmt"
	"strings"
	"time"

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

// createSearchCommand creates the 'search' command with advanced filtering
func (c *CLI) createSearchCommand() *cobra.Command {
	var (
		repository      string
		status          string
		priority        string
		tags            []string
		createdAfter    string
		createdBefore   string
		updatedAfter    string
		updatedBefore   string
		dueAfter        string
		dueBefore       string
		completedAfter  string
		completedBefore string
		overdue         bool
		dueSoon         bool
		hasDueDate      bool
		limit           int
		sortBy          string
		fuzzy           bool
		caseSensitive   bool
		exactMatch      bool
		allRepos        bool
		sessionID       string
		parentID        string
		fields          []string
		excludeTags     []string
		estimatedMin    int
		estimatedMax    int
		interactive     bool
		save            string
		load            string
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Advanced search for tasks with extensive filtering",
		Long: `Advanced search for tasks with comprehensive filtering capabilities.

Search supports multiple query modes:
- Text search in content, tags, and metadata
- Fuzzy matching for typos and variations  
- Date range filtering for creation, updates, completion
- Task relationship filtering (parent/child, sessions)
- Interactive search mode with real-time results
- Saved search configurations

Examples:
  lmmc search "database optimization"                    # Basic text search
  lmmc search bug --priority high --status pending      # Status and priority
  lmmc search --created-after 2024-01-01 --overdue     # Date and overdue tasks
  lmmc search --tags backend,api --exclude-tags deprecated  # Tag inclusion/exclusion
  lmmc search --fuzzy "databse" --all-repos             # Fuzzy search across all repos
  lmmc search --interactive                             # Interactive search mode
  lmmc search --save "high-priority-bugs" --priority high --tags bug  # Save search
  lmmc search --load "high-priority-bugs"               # Load saved search

Field targeting:
  --fields content,tags,metadata                        # Search specific fields only

Date filtering:
  --created-after "2024-01-01"     # Tasks created after date
  --created-before "2024-12-31"    # Tasks created before date  
  --updated-after "1 week ago"     # Relative dates supported
  --due-soon                       # Tasks due within 3 days
  --overdue                        # Past due tasks

Advanced options:
  --sort-by relevance|created|updated|priority|due     # Sort results
  --limit 50                       # Limit number of results
  --exact-match                    # Exact phrase matching only
  --case-sensitive                 # Case-sensitive search`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle interactive mode
			if interactive {
				return c.runInteractiveSearch()
			}

			// Handle load saved search
			if load != "" {
				return c.runSavedSearch(load, cmd)
			}

			// Require query unless using filters only
			var query string
			if len(args) > 0 {
				query = strings.Join(args, " ")
			} else if !c.hasSearchFilters(cmd) {
				return fmt.Errorf("search query required unless using filters (use --help for examples)")
			}

			// Build advanced filters
			filters, err := c.buildAdvancedFilters(cmd, repository, status, priority, tags, excludeTags,
				createdAfter, createdBefore, updatedAfter, updatedBefore,
				dueAfter, dueBefore, completedAfter, completedBefore,
				overdue, dueSoon, hasDueDate, sessionID, parentID, estimatedMin, estimatedMax)
			if err != nil {
				return c.handleError(cmd, err)
			}

			// Configure search options
			searchOpts := &SearchOptions{
				Fuzzy:         fuzzy,
				CaseSensitive: caseSensitive,
				ExactMatch:    exactMatch,
				Fields:        fields,
				SortBy:        sortBy,
				Limit:         limit,
				AllRepos:      allRepos,
			}

			// Execute search
			tasks, searchStats, err := c.executeAdvancedSearch(query, filters, searchOpts)
			if err != nil {
				return c.handleError(cmd, err)
			}

			// Save search if requested
			if save != "" {
				if err := c.saveSearchConfig(save, query, filters, searchOpts); err != nil {
					fmt.Printf("⚠️  Warning: Failed to save search: %v\n", err)
				} else {
					fmt.Printf("💾 Search saved as '%s'\n", save)
				}
			}

			// Display search statistics
			c.displaySearchStats(searchStats)

			// Format and display results
			formatter := c.getOutputFormatter(cmd)
			return formatter.FormatTaskList(tasks)
		},
	}

	// Basic search flags
	cmd.Flags().StringVarP(&repository, "repository", "r", "", "Search in specific repository")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status (pending, in_progress, completed, cancelled)")
	cmd.Flags().StringVarP(&priority, "priority", "p", "", "Filter by priority (low, medium, high)")
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "Filter by tags (comma-separated)")

	// Date filtering flags
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Tasks created after date (YYYY-MM-DD or relative like '1 week ago')")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Tasks created before date")
	cmd.Flags().StringVar(&updatedAfter, "updated-after", "", "Tasks updated after date")
	cmd.Flags().StringVar(&updatedBefore, "updated-before", "", "Tasks updated before date")
	cmd.Flags().StringVar(&dueAfter, "due-after", "", "Tasks due after date")
	cmd.Flags().StringVar(&dueBefore, "due-before", "", "Tasks due before date")
	cmd.Flags().StringVar(&completedAfter, "completed-after", "", "Tasks completed after date")
	cmd.Flags().StringVar(&completedBefore, "completed-before", "", "Tasks completed before date")

	// Special date filters
	cmd.Flags().BoolVar(&overdue, "overdue", false, "Show only overdue tasks")
	cmd.Flags().BoolVar(&dueSoon, "due-soon", false, "Show tasks due within 3 days")
	cmd.Flags().BoolVar(&hasDueDate, "has-due-date", false, "Show only tasks with due dates")

	// Advanced filtering
	cmd.Flags().StringSliceVar(&excludeTags, "exclude-tags", nil, "Exclude tasks with these tags")
	cmd.Flags().StringVar(&sessionID, "session", "", "Filter by session ID")
	cmd.Flags().StringVar(&parentID, "parent", "", "Filter by parent task ID")
	cmd.Flags().IntVar(&estimatedMin, "estimated-min", 0, "Minimum estimated time in minutes")
	cmd.Flags().IntVar(&estimatedMax, "estimated-max", 0, "Maximum estimated time in minutes")

	// Search behavior flags
	cmd.Flags().BoolVar(&fuzzy, "fuzzy", false, "Enable fuzzy matching for typos")
	cmd.Flags().BoolVar(&caseSensitive, "case-sensitive", false, "Case-sensitive search")
	cmd.Flags().BoolVar(&exactMatch, "exact-match", false, "Exact phrase matching only")
	cmd.Flags().BoolVar(&allRepos, "all-repos", false, "Search across all repositories")
	cmd.Flags().StringSliceVar(&fields, "fields", nil, "Search specific fields only (content,tags,metadata)")

	// Output and sorting
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of results")
	cmd.Flags().StringVar(&sortBy, "sort-by", "relevance", "Sort by: relevance, created, updated, priority, due")

	// Interactive and saved searches
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive search mode")
	cmd.Flags().StringVar(&save, "save", "", "Save search configuration with name")
	cmd.Flags().StringVar(&load, "load", "", "Load saved search configuration")

	return cmd
}

// createSuggestCommand creates the 'suggest' command for AI-powered task suggestions
func (c *CLI) createSuggestCommand() *cobra.Command {
	var (
		context     string
		repository  string
		maxResults  int
		includeDesc bool
	)

	cmd := &cobra.Command{
		Use:   "suggest [context...]",
		Short: "Get AI-powered task suggestions",
		Long: `Get intelligent task suggestions based on current work context, repository patterns,
and project history. The AI analyzes your current tasks, repository structure, and
workflow patterns to provide relevant task suggestions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if intelligence services are available
			if c.intelligence == nil || c.intelligence.SuggestionService == nil {
				return c.handleError(cmd, fmt.Errorf("suggestion service not available - intelligence features not configured"))
			}

			// Build context from args if provided
			if len(args) > 0 {
				context = strings.Join(args, " ")
			}

			// Auto-detect repository if not specified
			if repository == "" {
				if repoInfo, err := c.repositoryDetector.DetectCurrent(c.getContext()); err == nil && repoInfo != nil {
					repository = repoInfo.Name
				}
			}

			// Get current tasks for context
			filters := ports.TaskFilters{
				Repository: repository,
			}
			activeTasks, err := c.taskService.ListTasks(c.getContext(), &filters)
			if err != nil {
				c.logger.Warn("Failed to load current tasks for context", "error", err)
			}

			// Build work context for suggestion generation
			workContext := &entities.WorkContext{
				Repository:   repository,
				CurrentTasks: activeTasks,
				// Set reasonable defaults for other fields
				TimeOfDay:         getTimeOfDay(),
				DayOfWeek:         time.Now().Weekday().String(),
				FocusLevel:        0.8,  // Assume good focus
				EnergyLevel:       0.7,  // Assume moderate energy
				ProductivityScore: 0.75, // Assume decent productivity
				Velocity:          2.0,  // 2 tasks per hour
			}

			// Set user context if provided
			if context != "" {
				// Create a simple goal from the context
				workContext.Goals = []entities.SessionGoal{
					{
						Description: context,
						Priority:    "medium",
						Type:        "task",
						Target:      maxResults,
					},
				}
			}

			// Get suggestions
			suggestions, err := c.intelligence.SuggestionService.GenerateSuggestionsForContext(c.getContext(), workContext, maxResults)
			if err != nil {
				return c.handleError(cmd, fmt.Errorf("failed to get suggestions: %w", err))
			}

			if len(suggestions) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No suggestions available for the current context.\n")
				return nil
			}

			// Format and display suggestions
			formatter := c.getOutputFormatter(cmd)
			// Convert to slice of values instead of pointers
			valueSlice := make([]entities.TaskSuggestion, len(suggestions))
			for i, s := range suggestions {
				valueSlice[i] = *s
			}
			return c.formatSuggestions(cmd, formatter, valueSlice)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&context, "context", "c", "", "Additional context for suggestions")
	cmd.Flags().StringVarP(&repository, "repository", "r", "", "Target repository (auto-detected if not specified)")
	cmd.Flags().IntVarP(&maxResults, "max", "m", 5, "Maximum number of suggestions to return")
	cmd.Flags().BoolVarP(&includeDesc, "describe", "d", false, "Include detailed descriptions in suggestions")

	return cmd
}

// formatSuggestions formats and displays task suggestions
func (c *CLI) formatSuggestions(cmd *cobra.Command, formatter OutputFormatter, suggestions []entities.TaskSuggestion) error {
	switch formatter.(type) {
	case *JSONFormatter:
		return formatter.FormatDocument(suggestions)
	default:
		// Table/plain format - custom formatting for suggestions
		fmt.Fprintf(cmd.OutOrStdout(), "\n🤖 AI Task Suggestions:\n\n")

		for i, suggestion := range suggestions {
			// Header with confidence score
			fmt.Fprintf(cmd.OutOrStdout(), "%d. %s ", i+1, suggestion.Content)

			// Show confidence with colored indicators
			confidence := suggestion.Confidence
			var indicator string
			switch {
			case confidence >= 0.8:
				indicator = "🟢 High"
			case confidence >= 0.6:
				indicator = "🟡 Medium"
			case confidence >= 0.4:
				indicator = "🟠 Low"
			default:
				indicator = "🔴 Very Low"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "(%s confidence: %.0f%%)\n", indicator, confidence*100)

			// Description if available
			if suggestion.Description != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", suggestion.Description)
			}

			// Show reasoning if available
			if suggestion.Reasoning != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "   💡 Why: %s\n", suggestion.Reasoning)
			}

			// Show suggested priority/tags if available
			if suggestion.Priority != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "   📌 Priority: %s", suggestion.Priority)
				if len(suggestion.Tags) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), " | Tags: %s", strings.Join(suggestion.Tags, ", "))
				}
				fmt.Fprintf(cmd.OutOrStdout(), "\n")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\n")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "💡 Use 'lmmc add \"<suggestion>\"' to create a task from these suggestions.\n\n")
		return nil
	}
}

// getTimeOfDay returns the current time of day as a string
func getTimeOfDay() string {
	hour := time.Now().Hour()
	switch {
	case hour >= 5 && hour < 12:
		return "morning"
	case hour >= 12 && hour < 17:
		return "afternoon"
	case hour >= 17 && hour < 21:
		return "evening"
	default:
		return "night"
	}
}
