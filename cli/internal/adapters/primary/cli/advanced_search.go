package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// SearchOptions configures advanced search behavior
type SearchOptions struct {
	Fuzzy         bool
	CaseSensitive bool
	ExactMatch    bool
	Fields        []string
	SortBy        string
	Limit         int
	AllRepos      bool
}

// SearchStats provides search execution statistics
type SearchStats struct {
	Query                string
	TotalFound           int
	SearchTime           time.Duration
	RepositoriesSearched int
	FiltersApplied       []string
}

// SavedSearch represents a saved search configuration
type SavedSearch struct {
	Name        string                 `yaml:"name"`
	Query       string                 `yaml:"query"`
	Filters     map[string]interface{} `yaml:"filters"`
	Options     map[string]interface{} `yaml:"options"`
	CreatedAt   time.Time              `yaml:"created_at"`
	Description string                 `yaml:"description,omitempty"`
}

// hasSearchFilters checks if any search filters are provided
func (c *CLI) hasSearchFilters(cmd *cobra.Command) bool {
	filterFlags := []string{
		"status", "priority", "tags", "repository",
		"created-after", "created-before", "updated-after", "updated-before",
		"due-after", "due-before", "completed-after", "completed-before",
		"overdue", "due-soon", "has-due-date",
		"exclude-tags", "session", "parent",
		"estimated-min", "estimated-max",
	}

	for _, flag := range filterFlags {
		if cmd.Flags().Changed(flag) {
			return true
		}
	}
	return false
}

// buildAdvancedFilters constructs TaskFilters from command flags
func (c *CLI) buildAdvancedFilters(repository, status, priority string, tags, excludeTags []string,
	createdAfter, createdBefore, updatedAfter, updatedBefore,
	dueAfter, dueBefore, completedAfter, completedBefore string,
	overdue, dueSoon, hasDueDate bool, sessionID, parentID string,
	estimatedMin, estimatedMax int) (*ports.TaskFilters, error) {
	filters := c.initializeBaseFilters(repository, tags, sessionID, parentID, overdue, dueSoon, hasDueDate)

	if err := c.setStatusAndPriorityFilters(filters, status, priority); err != nil {
		return nil, err
	}

	if err := c.setDateFilters(filters, createdAfter, createdBefore, updatedAfter, updatedBefore,
		dueAfter, dueBefore, completedAfter, completedBefore); err != nil {
		return nil, err
	}

	c.setTimeEstimationFilters(filters, estimatedMin, estimatedMax)
	c.setExcludeTagsFilter(filters, excludeTags)

	return filters, nil
}

// initializeBaseFilters creates a TaskFilters instance with basic fields
func (c *CLI) initializeBaseFilters(repository string, tags []string, sessionID, parentID string,
	overdue, dueSoon, hasDueDate bool) *ports.TaskFilters {
	filters := &ports.TaskFilters{
		Repository:  repository,
		Tags:        tags,
		SessionID:   sessionID,
		ParentID:    parentID,
		OverdueOnly: overdue,
	}

	if dueSoon {
		hours := 72 // 3 days in hours
		filters.DueSoon = &hours
	}
	if hasDueDate {
		filters.HasDueDate = &hasDueDate
	}

	return filters
}

// setStatusAndPriorityFilters sets status and priority filters
func (c *CLI) setStatusAndPriorityFilters(filters *ports.TaskFilters, status, priority string) error {
	if status != "" {
		s, err := parseStatus(status)
		if err != nil {
			return err
		}
		filters.Status = &s
	}

	if priority != "" {
		p, err := parsePriority(priority)
		if err != nil {
			return err
		}
		filters.Priority = &p
	}

	return nil
}

// setDateFilters sets all date-related filters
func (c *CLI) setDateFilters(filters *ports.TaskFilters, createdAfter, createdBefore, updatedAfter, updatedBefore,
	dueAfter, dueBefore, completedAfter, completedBefore string) error {
	dateFields := []struct {
		value  string
		target **string
		field  string
	}{
		{createdAfter, &filters.CreatedAfter, "created-after"},
		{createdBefore, &filters.CreatedBefore, "created-before"},
		{updatedAfter, &filters.UpdatedAfter, "updated-after"},
		{updatedBefore, &filters.UpdatedBefore, "updated-before"},
		{dueAfter, &filters.DueAfter, "due-after"},
		{dueBefore, &filters.DueBefore, "due-before"},
		{completedAfter, &filters.CompletedAfter, "completed-after"},
		{completedBefore, &filters.CompletedBefore, "completed-before"},
	}

	for _, field := range dateFields {
		if field.value != "" {
			if err := c.setDateField(field.target, field.value, field.field); err != nil {
				return err
			}
		}
	}

	return nil
}

// setDateField sets a single date field with error handling
func (c *CLI) setDateField(target **string, value, fieldName string) error {
	date, err := c.parseDate(value)
	if err != nil {
		return fmt.Errorf("invalid %s date: %w", fieldName, err)
	}
	dateStr := date.Format(time.RFC3339)
	*target = &dateStr
	return nil
}

// setTimeEstimationFilters sets estimated time filters
func (c *CLI) setTimeEstimationFilters(filters *ports.TaskFilters, estimatedMin, estimatedMax int) {
	if estimatedMin > 0 {
		minDuration := time.Duration(estimatedMin) * time.Minute
		filters.EstimatedTimeMin = &minDuration
	}
	if estimatedMax > 0 {
		maxDuration := time.Duration(estimatedMax) * time.Minute
		filters.EstimatedTimeMax = &maxDuration
	}
}

// setExcludeTagsFilter sets exclude tags filter
func (c *CLI) setExcludeTagsFilter(filters *ports.TaskFilters, excludeTags []string) {
	if len(excludeTags) > 0 {
		filters.ExcludeTags = excludeTags
	}
}

// parseDate parses various date formats including relative dates
func (c *CLI) parseDate(dateStr string) (*time.Time, error) {
	// Try standard formats first
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t, nil
		}
	}

	// Handle relative dates
	if strings.Contains(dateStr, "ago") {
		return c.parseRelativeDate(dateStr)
	}

	return nil, fmt.Errorf("unsupported date format: %s (use YYYY-MM-DD or relative like '1 week ago')", dateStr)
}

// parseRelativeDate parses relative date expressions like "1 week ago", "3 days ago"
func (c *CLI) parseRelativeDate(dateStr string) (*time.Time, error) {
	dateStr = strings.ToLower(strings.TrimSpace(dateStr))

	// Remove "ago" suffix
	dateStr = strings.TrimSuffix(dateStr, " ago")
	dateStr = strings.TrimSpace(dateStr)

	// Parse number and unit
	parts := strings.Fields(dateStr)
	if len(parts) != 2 {
		return nil, errors.New("invalid relative date format (use like '1 week ago')")
	}

	num, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid number in relative date: %s", parts[0])
	}

	unit := strings.TrimSuffix(parts[1], "s") // Remove plural 's'

	var duration time.Duration
	switch unit {
	case "minute", "min":
		duration = time.Duration(num) * time.Minute
	case "hour", "hr":
		duration = time.Duration(num) * time.Hour
	case "day":
		duration = time.Duration(num) * 24 * time.Hour
	case "week":
		duration = time.Duration(num) * 7 * 24 * time.Hour
	case "month":
		duration = time.Duration(num) * 30 * 24 * time.Hour // Approximate
	case "year":
		duration = time.Duration(num) * 365 * 24 * time.Hour // Approximate
	default:
		return nil, fmt.Errorf("unsupported time unit: %s (use minute, hour, day, week, month, year)", unit)
	}

	result := time.Now().Add(-duration)
	return &result, nil
}

// executeAdvancedSearch performs the search with advanced options
func (c *CLI) executeAdvancedSearch(query string, filters *ports.TaskFilters, opts *SearchOptions) ([]*entities.Task, *SearchStats, error) {
	startTime := time.Now()

	// Apply search field filtering
	filters.Search = query
	if len(opts.Fields) > 0 {
		filters.SearchFields = opts.Fields
	}

	// Configure fuzzy search
	if opts.Fuzzy {
		filters.FuzzySearch = true
		filters.FuzzyThreshold = 0.7 // Allow 30% character differences
	}

	// Configure case sensitivity
	filters.CaseSensitive = opts.CaseSensitive

	// Configure exact matching
	filters.ExactMatch = opts.ExactMatch

	// Perform search
	var tasks []*entities.Task
	var err error
	reposSearched := 1

	if opts.AllRepos {
		// Search across all repositories
		tasks, err = c.taskService.SearchAllRepositories(c.getContext(), filters)
		if err != nil {
			return nil, nil, err
		}
		reposSearched = -1 // Indicate all repos
	} else {
		// Search in specific or current repository
		tasks, err = c.taskService.SearchTasks(c.getContext(), query, filters)
		if err != nil {
			return nil, nil, err
		}
	}

	// Apply post-processing
	tasks = c.applySortAndLimit(tasks, opts)

	// Build search statistics
	searchTime := time.Since(startTime)
	stats := &SearchStats{
		Query:                query,
		TotalFound:           len(tasks),
		SearchTime:           searchTime,
		RepositoriesSearched: reposSearched,
		FiltersApplied:       c.getAppliedFilters(filters),
	}

	return tasks, stats, nil
}

// applySortAndLimit applies sorting and result limiting
func (c *CLI) applySortAndLimit(tasks []*entities.Task, opts *SearchOptions) []*entities.Task {
	// Apply sorting
	switch strings.ToLower(opts.SortBy) {
	case "created":
		c.sortTasksByCreated(tasks)
	case "updated":
		c.sortTasksByUpdated(tasks)
	case constants.FieldPriority:
		c.sortTasksByPriority(tasks)
	case "due":
		c.sortTasksByDue(tasks)
	case "relevance":
		fallthrough
	default:
		// Keep default relevance sorting from search
	}

	// Apply limit
	if opts.Limit > 0 && len(tasks) > opts.Limit {
		tasks = tasks[:opts.Limit]
	}

	return tasks
}

// Sort helper functions
func (c *CLI) sortTasksByCreated(tasks []*entities.Task) {
	// Sort by creation time, newest first
	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[i].CreatedAt.Before(tasks[j].CreatedAt) {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

func (c *CLI) sortTasksByUpdated(tasks []*entities.Task) {
	// Sort by update time, newest first
	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[i].UpdatedAt.Before(tasks[j].UpdatedAt) {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

func (c *CLI) sortTasksByPriority(tasks []*entities.Task) {
	// Sort by priority: high, medium, low
	priorityOrder := map[entities.Priority]int{
		entities.PriorityHigh:   3,
		entities.PriorityMedium: 2,
		entities.PriorityLow:    1,
	}

	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			if priorityOrder[tasks[i].Priority] < priorityOrder[tasks[j].Priority] {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}
}

func (c *CLI) sortTasksByDue(tasks []*entities.Task) {
	// Sort by due date, earliest first (nil due dates last)
	for i := 0; i < len(tasks)-1; i++ {
		for j := i + 1; j < len(tasks); j++ {
			task1, task2 := tasks[i], tasks[j]

			// Tasks with no due date go to the end
			if task1.DueDate == nil && task2.DueDate != nil {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			} else if task1.DueDate != nil && task2.DueDate != nil {
				if task1.DueDate.After(*task2.DueDate) {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	}
}

// getAppliedFilters returns a list of applied filter descriptions
func (c *CLI) getAppliedFilters(filters *ports.TaskFilters) []string {
	var applied []string

	if filters.Status != nil {
		applied = append(applied, fmt.Sprintf("Status: %s", *filters.Status))
	}
	if filters.Priority != nil {
		applied = append(applied, fmt.Sprintf("Priority: %s", *filters.Priority))
	}
	if len(filters.Tags) > 0 {
		applied = append(applied, "Tags: "+strings.Join(filters.Tags, ","))
	}
	if len(filters.ExcludeTags) > 0 {
		applied = append(applied, "Exclude tags: "+strings.Join(filters.ExcludeTags, ","))
	}
	if filters.Repository != "" {
		applied = append(applied, "Repository: "+filters.Repository)
	}
	if filters.CreatedAfter != nil {
		applied = append(applied, "Created after: "+*filters.CreatedAfter)
	}
	if filters.CreatedBefore != nil {
		applied = append(applied, "Created before: "+*filters.CreatedBefore)
	}
	if filters.OverdueOnly {
		applied = append(applied, "Overdue tasks only")
	}
	if filters.DueSoon != nil && *filters.DueSoon > 0 {
		applied = append(applied, fmt.Sprintf("Due soon (within %d hours)", *filters.DueSoon))
	}
	if filters.FuzzySearch {
		applied = append(applied, "Fuzzy matching enabled")
	}
	if filters.CaseSensitive {
		applied = append(applied, "Case-sensitive search")
	}
	if filters.ExactMatch {
		applied = append(applied, "Exact phrase matching")
	}

	return applied
}

// displaySearchStats shows search execution statistics
func (c *CLI) displaySearchStats(stats *SearchStats) {
	if c.verbose {
		fmt.Printf("🔍 Search Statistics:\n")
		fmt.Printf("   Query: %s\n", stats.Query)
		fmt.Printf("   Results: %d tasks found\n", stats.TotalFound)
		fmt.Printf("   Search time: %v\n", stats.SearchTime)
		if stats.RepositoriesSearched > 0 {
			fmt.Printf("   Repositories: %d searched\n", stats.RepositoriesSearched)
		} else {
			fmt.Printf("   Repositories: all searched\n")
		}
		if len(stats.FiltersApplied) > 0 {
			fmt.Printf("   Filters applied:\n")
			for _, filter := range stats.FiltersApplied {
				fmt.Printf("     - %s\n", filter)
			}
		}
		fmt.Println()
	} else {
		// Compact stats for non-verbose mode
		filterText := ""
		if len(stats.FiltersApplied) > 0 {
			filterText = fmt.Sprintf(" (%d filters)", len(stats.FiltersApplied))
		}
		fmt.Printf("🔍 Found %d tasks in %v%s\n", stats.TotalFound, stats.SearchTime, filterText)
	}
}

// runInteractiveSearch starts an interactive search session
func (c *CLI) runInteractiveSearch() error {
	fmt.Println("🔍 Interactive Search Mode")
	fmt.Println("Type your search query and press Enter. Use 'help' for commands, 'exit' to quit.")
	fmt.Println()

	// Interactive search implementation would go here
	// This would require a more complex input handling system
	fmt.Println("⚠️  Interactive search mode is not yet implemented.")
	fmt.Println("Use regular search with --help to see all available options.")

	return nil
}

// saveSearchConfig saves a search configuration for reuse
func (c *CLI) saveSearchConfig(name, query string, filters *ports.TaskFilters, opts *SearchOptions) error {
	configDir, err := c.getConfigDir()
	if err != nil {
		return err
	}

	searchesDir := filepath.Join(configDir, "searches")
	if err := os.MkdirAll(searchesDir, 0750); err != nil {
		return err
	}

	// Convert filters and options to map for YAML serialization
	filterMap := c.filtersToMap(filters)
	optionsMap := c.optionsToMap(opts)

	savedSearch := SavedSearch{
		Name:      name,
		Query:     query,
		Filters:   filterMap,
		Options:   optionsMap,
		CreatedAt: time.Now(),
	}

	// Save to YAML file
	filename := filepath.Join(searchesDir, name+".yaml")
	data, err := yaml.Marshal(savedSearch)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0600)
}

// runSavedSearch loads and executes a saved search
func (c *CLI) runSavedSearch(name string, cmd *cobra.Command) error {
	configDir, err := c.getConfigDir()
	if err != nil {
		return err
	}

	filename := filepath.Join(configDir, "searches", name+".yaml")

	// Clean and validate the file path
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		return fmt.Errorf("path traversal detected: %s", filename)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("saved search '%s' not found: %w", name, err)
	}

	var savedSearch SavedSearch
	if err := yaml.Unmarshal(data, &savedSearch); err != nil {
		return fmt.Errorf("failed to parse saved search: %w", err)
	}

	fmt.Printf("📁 Loading saved search: %s\n", savedSearch.Name)
	fmt.Printf("   Created: %s\n", savedSearch.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("   Query: %s\n", savedSearch.Query)
	fmt.Println()

	// Convert back to filters and options
	filters := c.mapToFilters(savedSearch.Filters)
	opts := c.mapToOptions(savedSearch.Options)

	// Execute the search
	tasks, searchStats, err := c.executeAdvancedSearch(savedSearch.Query, filters, opts)
	if err != nil {
		return err
	}

	// Display results
	c.displaySearchStats(searchStats)
	formatter := c.getOutputFormatter(cmd)
	return formatter.FormatTaskList(tasks)
}

// Helper functions for serialization

func (c *CLI) getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".lmmc"), nil
}

func (c *CLI) filtersToMap(filters *ports.TaskFilters) map[string]interface{} {
	m := make(map[string]interface{})

	if filters.Status != nil {
		m["status"] = string(*filters.Status)
	}
	if filters.Priority != nil {
		m[constants.FieldPriority] = string(*filters.Priority)
	}
	if filters.Repository != "" {
		m["repository"] = filters.Repository
	}
	if len(filters.Tags) > 0 {
		m["tags"] = filters.Tags
	}
	if len(filters.ExcludeTags) > 0 {
		m["exclude_tags"] = filters.ExcludeTags
	}
	if filters.OverdueOnly {
		m["overdue"] = true
	}
	if filters.DueSoon != nil && *filters.DueSoon > 0 {
		m["due_soon"] = true
	}

	return m
}

func (c *CLI) optionsToMap(opts *SearchOptions) map[string]interface{} {
	m := make(map[string]interface{})

	m["fuzzy"] = opts.Fuzzy
	m["case_sensitive"] = opts.CaseSensitive
	m["exact_match"] = opts.ExactMatch
	m["all_repos"] = opts.AllRepos
	m["sort_by"] = opts.SortBy
	m["limit"] = opts.Limit
	if len(opts.Fields) > 0 {
		m["fields"] = opts.Fields
	}

	return m
}

func (c *CLI) mapToFilters(m map[string]interface{}) *ports.TaskFilters {
	filters := &ports.TaskFilters{}

	if status, ok := m["status"].(string); ok {
		s := entities.Status(status)
		filters.Status = &s
	}
	if priority, ok := m[constants.FieldPriority].(string); ok {
		p := entities.Priority(priority)
		filters.Priority = &p
	}
	if repository, ok := m["repository"].(string); ok {
		filters.Repository = repository
	}
	if tags, ok := m["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				filters.Tags = append(filters.Tags, tagStr)
			}
		}
	}

	return filters
}

func (c *CLI) mapToOptions(m map[string]interface{}) *SearchOptions {
	opts := &SearchOptions{}

	if fuzzy, ok := m["fuzzy"].(bool); ok {
		opts.Fuzzy = fuzzy
	}
	if caseSensitive, ok := m["case_sensitive"].(bool); ok {
		opts.CaseSensitive = caseSensitive
	}
	if exactMatch, ok := m["exact_match"].(bool); ok {
		opts.ExactMatch = exactMatch
	}
	if allRepos, ok := m["all_repos"].(bool); ok {
		opts.AllRepos = allRepos
	}
	if sortBy, ok := m["sort_by"].(string); ok {
		opts.SortBy = sortBy
	}
	if limit, ok := m["limit"].(int); ok {
		opts.Limit = limit
	}

	return opts
}
