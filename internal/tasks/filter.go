// Package tasks provides filtering and sorting functionality for task management.
package tasks

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// Field name constants
const (
	FieldTitle       = "title"
	FieldDescription = "description"
)

// FilterManager handles task filtering and sorting operations
type FilterManager struct {
	config FilterConfig
}

// FilterConfig represents configuration for filtering
type FilterConfig struct {
	MaxFilters        int  `json:"max_filters"`
	MaxSortFields     int  `json:"max_sort_fields"`
	CaseSensitive     bool `json:"case_sensitive"`
	EnableFuzzySearch bool `json:"enable_fuzzy_search"`
}

// TaskFilters represents filtering criteria for tasks
type TaskFilters struct {
	Status          []types.TaskStatus      `json:"status,omitempty"`
	Type            []types.TaskType        `json:"type,omitempty"`
	Priority        []types.TaskPriority    `json:"priority,omitempty"`
	Assignee        string                  `json:"assignee,omitempty"`
	Repository      string                  `json:"repository,omitempty"`
	Tags            []string                `json:"tags,omitempty"`
	CreatedAfter    *time.Time              `json:"created_after,omitempty"`
	CreatedBefore   *time.Time              `json:"created_before,omitempty"`
	UpdatedAfter    *time.Time              `json:"updated_after,omitempty"`
	UpdatedBefore   *time.Time              `json:"updated_before,omitempty"`
	DueAfter        *time.Time              `json:"due_after,omitempty"`
	DueBefore       *time.Time              `json:"due_before,omitempty"`
	Complexity      []types.ComplexityLevel `json:"complexity,omitempty"`
	MinQualityScore float64                 `json:"min_quality_score,omitempty"`
	MaxQualityScore float64                 `json:"max_quality_score,omitempty"`
	HasDependencies *bool                   `json:"has_dependencies,omitempty"`
	IsBlocked       *bool                   `json:"is_blocked,omitempty"`
	TextSearch      string                  `json:"text_search,omitempty"`

	// Pagination
	Offset int `json:"offset"`
	Limit  int `json:"limit"`

	// Sorting
	SortBy []SortField `json:"sort_by,omitempty"`
}

// SortField represents a field to sort by
type SortField struct {
	Field string    `json:"field"`
	Order SortOrder `json:"order"`
}

// SortOrder represents sort direction
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// SearchQuery represents a search query with filters
type SearchQuery struct {
	Query   string        `json:"query"`
	Filters TaskFilters   `json:"filters"`
	Options SearchOptions `json:"options"`
}

// SearchOptions represents search configuration options
type SearchOptions struct {
	HighlightMatches bool     `json:"highlight_matches"`
	FuzzySearch      bool     `json:"fuzzy_search"`
	MaxResults       int      `json:"max_results"`
	SearchFields     []string `json:"search_fields,omitempty"`
}

// SearchResults represents search results with metadata
type SearchResults struct {
	Tasks        []types.Task        `json:"tasks"`
	TotalResults int                 `json:"total_results"`
	SearchTime   time.Duration       `json:"search_time"`
	Query        string              `json:"query"`
	Highlights   map[string][]string `json:"highlights,omitempty"`
}

// BatchUpdate represents a batch update operation
type BatchUpdate struct {
	TaskID    string              `json:"task_id"`
	Status    *types.TaskStatus   `json:"status,omitempty"`
	Priority  *types.TaskPriority `json:"priority,omitempty"`
	Assignee  *string             `json:"assignee,omitempty"`
	Tags      []string            `json:"tags,omitempty"`
	DueDate   *time.Time          `json:"due_date,omitempty"`
	UpdatedAt time.Time           `json:"updated_at"`
}

// BatchResult represents the result of a batch operation
type BatchResult struct {
	TotalRequested  int          `json:"total_requested"`
	SuccessfulCount int          `json:"successful_count"`
	FailedCount     int          `json:"failed_count"`
	Successful      []string     `json:"successful"`
	Failed          []BatchError `json:"failed"`
}

// BatchError represents an error in batch processing
type BatchError struct {
	TaskID string `json:"task_id"`
	Error  string `json:"error"`
}

// DefaultFilterConfig returns default filter configuration
func DefaultFilterConfig() FilterConfig {
	return FilterConfig{
		MaxFilters:        20,
		MaxSortFields:     5,
		CaseSensitive:     false,
		EnableFuzzySearch: true,
	}
}

// NewFilterManager creates a new filter manager
func NewFilterManager() *FilterManager {
	return &FilterManager{
		config: DefaultFilterConfig(),
	}
}

// NewFilterManagerWithConfig creates a filter manager with custom config
func NewFilterManagerWithConfig(config FilterConfig) *FilterManager {
	return &FilterManager{
		config: config,
	}
}

// ValidateFilters validates filtering criteria
func (fm *FilterManager) ValidateFilters(filters *TaskFilters) error {
	// Validate filter count
	filterCount := fm.countActiveFilters(filters)
	if filterCount > fm.config.MaxFilters {
		return fmt.Errorf("too many filters: %d (max %d)", filterCount, fm.config.MaxFilters)
	}

	// Validate sort fields
	if len(filters.SortBy) > fm.config.MaxSortFields {
		return fmt.Errorf("too many sort fields: %d (max %d)", len(filters.SortBy), fm.config.MaxSortFields)
	}

	// Validate date ranges
	if err := fm.validateDateRanges(filters); err != nil {
		return fmt.Errorf("invalid date range: %w", err)
	}

	// Validate quality score range
	if filters.MinQualityScore < 0 || filters.MinQualityScore > 1 {
		return errors.New("min_quality_score must be between 0 and 1")
	}
	if filters.MaxQualityScore < 0 || filters.MaxQualityScore > 1 {
		return errors.New("max_quality_score must be between 0 and 1")
	}
	if filters.MinQualityScore > filters.MaxQualityScore {
		return errors.New("min_quality_score cannot be greater than max_quality_score")
	}

	// Validate sort fields
	for _, sortField := range filters.SortBy {
		if err := fm.validateSortField(sortField); err != nil {
			return fmt.Errorf("invalid sort field: %w", err)
		}
	}

	// Validate pagination
	if filters.Offset < 0 {
		return errors.New("offset cannot be negative")
	}
	if filters.Limit < 0 {
		return errors.New("limit cannot be negative")
	}

	return nil
}

// ApplyFilters applies filters to a task list (in-memory filtering)
func (fm *FilterManager) ApplyFilters(tasks []types.Task, filters *TaskFilters) []types.Task {
	filtered := make([]types.Task, 0, len(tasks))

	for i := range tasks {
		if fm.matchesFilters(&tasks[i], filters) {
			filtered = append(filtered, tasks[i])
		}
	}

	// Apply sorting
	if len(filters.SortBy) > 0 {
		filtered = fm.sortTasks(filtered, filters.SortBy)
	}

	// Apply pagination
	if filters.Offset > 0 || filters.Limit > 0 {
		filtered = fm.paginateTasks(filtered, filters.Offset, filters.Limit)
	}

	return filtered
}

// BuildWhereClause builds SQL WHERE clause from filters
func (fm *FilterManager) BuildWhereClause(filters *TaskFilters) (whereClause string, args []interface{}) {
	var conditions []string
	args = make([]interface{}, 0)
	argIndex := 1

	fm.addListFilters(&conditions, &args, &argIndex, filters)
	fm.addStringFilters(&conditions, &args, &argIndex, filters)
	fm.addDateFilters(&conditions, &args, &argIndex, filters)
	fm.addQualityFilters(&conditions, &args, &argIndex, filters)
	fm.addTextSearchFilter(&conditions, &args, &argIndex, filters)

	whereClause = fm.buildFinalClause(conditions)
	return
}

// addListFilters handles status, type, and priority filters
func (fm *FilterManager) addListFilters(conditions *[]string, args *[]interface{}, argIndex *int, filters *TaskFilters) {
	// Status filter
	if len(filters.Status) > 0 {
		placeholders := fm.buildPlaceholders(len(filters.Status), argIndex)
		*conditions = append(*conditions, fmt.Sprintf("status IN (%s)", placeholders))
		for _, status := range filters.Status {
			*args = append(*args, string(status))
		}
	}

	// Type filter
	if len(filters.Type) > 0 {
		placeholders := fm.buildPlaceholders(len(filters.Type), argIndex)
		*conditions = append(*conditions, fmt.Sprintf("type IN (%s)", placeholders))
		for _, taskType := range filters.Type {
			*args = append(*args, string(taskType))
		}
	}

	// Priority filter
	if len(filters.Priority) > 0 {
		placeholders := fm.buildPlaceholders(len(filters.Priority), argIndex)
		*conditions = append(*conditions, fmt.Sprintf("priority IN (%s)", placeholders))
		for _, priority := range filters.Priority {
			*args = append(*args, string(priority))
		}
	}
}

// addStringFilters handles assignee and repository filters
func (fm *FilterManager) addStringFilters(conditions *[]string, args *[]interface{}, argIndex *int, filters *TaskFilters) {
	if filters.Assignee != "" {
		*conditions = append(*conditions, fmt.Sprintf("assignee = $%d", *argIndex))
		*args = append(*args, filters.Assignee)
		(*argIndex)++
	}

	if filters.Repository != "" {
		*conditions = append(*conditions, fmt.Sprintf("repository = $%d", *argIndex))
		*args = append(*args, filters.Repository)
		(*argIndex)++
	}
}

// addDateFilters handles date range filters
func (fm *FilterManager) addDateFilters(conditions *[]string, args *[]interface{}, argIndex *int, filters *TaskFilters) {
	if filters.CreatedAfter != nil {
		*conditions = append(*conditions, fmt.Sprintf("created_at >= $%d", *argIndex))
		*args = append(*args, *filters.CreatedAfter)
		(*argIndex)++
	}
	if filters.CreatedBefore != nil {
		*conditions = append(*conditions, fmt.Sprintf("created_at <= $%d", *argIndex))
		*args = append(*args, *filters.CreatedBefore)
		(*argIndex)++
	}
	if filters.UpdatedAfter != nil {
		*conditions = append(*conditions, fmt.Sprintf("updated_at >= $%d", *argIndex))
		*args = append(*args, *filters.UpdatedAfter)
		(*argIndex)++
	}
	if filters.UpdatedBefore != nil {
		*conditions = append(*conditions, fmt.Sprintf("updated_at <= $%d", *argIndex))
		*args = append(*args, *filters.UpdatedBefore)
		(*argIndex)++
	}
}

// addQualityFilters handles quality score filters
func (fm *FilterManager) addQualityFilters(conditions *[]string, args *[]interface{}, argIndex *int, filters *TaskFilters) {
	if filters.MinQualityScore > 0 {
		*conditions = append(*conditions, fmt.Sprintf("quality_score >= $%d", *argIndex))
		*args = append(*args, filters.MinQualityScore)
		(*argIndex)++
	}
	if filters.MaxQualityScore > 0 {
		*conditions = append(*conditions, fmt.Sprintf("quality_score <= $%d", *argIndex))
		*args = append(*args, filters.MaxQualityScore)
		(*argIndex)++
	}
}

// addTextSearchFilter handles text search filters
func (fm *FilterManager) addTextSearchFilter(conditions *[]string, args *[]interface{}, argIndex *int, filters *TaskFilters) {
	if filters.TextSearch != "" {
		searchPattern := fmt.Sprintf("%%%s%%", filters.TextSearch)
		if fm.config.CaseSensitive {
			*conditions = append(*conditions, fmt.Sprintf("(title LIKE $%d OR description LIKE $%d)", *argIndex, *argIndex))
		} else {
			*conditions = append(*conditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", *argIndex, *argIndex))
		}
		*args = append(*args, searchPattern)
		(*argIndex)++
	}
}

// buildFinalClause combines conditions into final WHERE clause
func (fm *FilterManager) buildFinalClause(conditions []string) string {
	if len(conditions) > 0 {
		return "WHERE " + strings.Join(conditions, " AND ")
	}
	return ""
}

// BuildOrderClause builds SQL ORDER BY clause from sort fields
func (fm *FilterManager) BuildOrderClause(sortFields []SortField) string {
	if len(sortFields) == 0 {
		return "ORDER BY created_at DESC" // Default sort
	}

	var orderClauses []string
	for _, field := range sortFields {
		validField := fm.sanitizeSortField(field.Field)
		if validField != "" {
			direction := "ASC"
			if field.Order == SortOrderDesc {
				direction = "DESC"
			}
			orderClauses = append(orderClauses, fmt.Sprintf("%s %s", validField, direction))
		}
	}

	if len(orderClauses) == 0 {
		return "ORDER BY created_at DESC"
	}

	return "ORDER BY " + strings.Join(orderClauses, ", ")
}

// Helper methods

func (fm *FilterManager) countActiveFilters(filters *TaskFilters) int {
	return fm.countArrayFilters(filters) +
		fm.countStringFilters(filters) +
		fm.countDateFilters(filters) +
		fm.countNumericFilters(filters) +
		fm.countBooleanFilters(filters)
}

// countArrayFilters counts active array-based filters
func (fm *FilterManager) countArrayFilters(filters *TaskFilters) int {
	count := 0
	arrayFilters := []interface{}{
		filters.Status,
		filters.Type,
		filters.Priority,
		filters.Tags,
		filters.Complexity,
	}

	for _, filter := range arrayFilters {
		switch f := filter.(type) {
		case []types.TaskStatus:
			if len(f) > 0 {
				count++
			}
		case []types.TaskType:
			if len(f) > 0 {
				count++
			}
		case []types.TaskPriority:
			if len(f) > 0 {
				count++
			}
		case []string:
			if len(f) > 0 {
				count++
			}
		case []types.ComplexityLevel:
			if len(f) > 0 {
				count++
			}
		}
	}
	return count
}

// countStringFilters counts active string-based filters
func (fm *FilterManager) countStringFilters(filters *TaskFilters) int {
	count := 0
	stringFilters := []string{
		filters.Assignee,
		filters.Repository,
		filters.TextSearch,
	}

	for _, filter := range stringFilters {
		if filter != "" {
			count++
		}
	}
	return count
}

// countDateFilters counts active date-based filters
func (fm *FilterManager) countDateFilters(filters *TaskFilters) int {
	count := 0
	dateFilters := []*time.Time{
		filters.CreatedAfter,
		filters.CreatedBefore,
		filters.UpdatedAfter,
		filters.UpdatedBefore,
		filters.DueAfter,
		filters.DueBefore,
	}

	for _, filter := range dateFilters {
		if filter != nil {
			count++
		}
	}
	return count
}

// countNumericFilters counts active numeric-based filters
func (fm *FilterManager) countNumericFilters(filters *TaskFilters) int {
	count := 0
	if filters.MinQualityScore > 0 {
		count++
	}
	if filters.MaxQualityScore > 0 {
		count++
	}
	return count
}

// countBooleanFilters counts active boolean-based filters
func (fm *FilterManager) countBooleanFilters(filters *TaskFilters) int {
	count := 0
	booleanFilters := []*bool{
		filters.HasDependencies,
		filters.IsBlocked,
	}

	for _, filter := range booleanFilters {
		if filter != nil {
			count++
		}
	}
	return count
}

func (fm *FilterManager) validateDateRanges(filters *TaskFilters) error {
	if filters.CreatedAfter != nil && filters.CreatedBefore != nil {
		if filters.CreatedAfter.After(*filters.CreatedBefore) {
			return errors.New("created_after cannot be after created_before")
		}
	}
	if filters.UpdatedAfter != nil && filters.UpdatedBefore != nil {
		if filters.UpdatedAfter.After(*filters.UpdatedBefore) {
			return errors.New("updated_after cannot be after updated_before")
		}
	}
	if filters.DueAfter != nil && filters.DueBefore != nil {
		if filters.DueAfter.After(*filters.DueBefore) {
			return errors.New("due_after cannot be after due_before")
		}
	}
	return nil
}

func (fm *FilterManager) validateSortField(field SortField) error {
	validFields := map[string]bool{
		"id": true, "title": true, "status": true, "priority": true, "type": true,
		"created_at": true, "updated_at": true, "due_date": true, "assignee": true,
		"quality_score": true, "complexity": true,
	}

	if !validFields[field.Field] {
		return fmt.Errorf("invalid sort field: %s", field.Field)
	}

	if field.Order != SortOrderAsc && field.Order != SortOrderDesc {
		return fmt.Errorf("invalid sort order: %s", field.Order)
	}

	return nil
}

func (fm *FilterManager) sanitizeSortField(field string) string {
	// Whitelist approach for security
	validFields := map[string]string{
		"id": "id", "title": "title", "status": "status", "priority": "priority",
		"type": "type", "created_at": "created_at", "updated_at": "updated_at",
		"due_date": "due_date", "assignee": "assignee", "quality_score": "quality_score",
		"complexity": "complexity",
	}
	return validFields[field]
}

func (fm *FilterManager) buildPlaceholders(count int, argIndex *int) string {
	placeholders := make([]string, count)
	for i := 0; i < count; i++ {
		placeholders[i] = fmt.Sprintf("$%d", *argIndex)
		*argIndex++
	}
	return strings.Join(placeholders, ", ")
}

func (fm *FilterManager) matchesFilters(task *types.Task, filters *TaskFilters) bool {
	return fm.matchesStringFilters(task, filters) &&
		fm.matchesTextSearch(task, filters) &&
		fm.matchesDateFilters(task, filters) &&
		fm.matchesQualityFilters(task, filters)
}

// matchesStringFilters checks status, type, priority, and assignee filters
func (fm *FilterManager) matchesStringFilters(task *types.Task, filters *TaskFilters) bool {
	stringChecks := []struct {
		filterSlice []string
		taskValue   string
	}{
		{fm.statusToStrings(filters.Status), string(task.Status)},
		{fm.typeToStrings(filters.Type), string(task.Type)},
		{fm.priorityToStrings(filters.Priority), string(task.Priority)},
	}

	for _, check := range stringChecks {
		if len(check.filterSlice) > 0 {
			if !fm.containsString(check.filterSlice, check.taskValue) {
				return false
			}
		}
	}

	// Assignee filter (single value, not slice)
	if filters.Assignee != "" && task.Assignee != filters.Assignee {
		return false
	}

	return true
}

// matchesTextSearch performs text search in task content
func (fm *FilterManager) matchesTextSearch(task *types.Task, filters *TaskFilters) bool {
	if filters.TextSearch == "" {
		return true
	}

	searchText := filters.TextSearch
	content := task.Title + " " + task.Description

	if !fm.config.CaseSensitive {
		searchText = strings.ToLower(searchText)
		content = strings.ToLower(content)
	}

	return strings.Contains(content, searchText)
}

// matchesDateFilters checks created and updated date ranges
func (fm *FilterManager) matchesDateFilters(task *types.Task, filters *TaskFilters) bool {
	dateChecks := []struct {
		filterDate *time.Time
		taskDate   time.Time
		isAfter    bool
	}{
		{filters.CreatedAfter, task.Timestamps.Created, true},
		{filters.CreatedBefore, task.Timestamps.Created, false},
		{filters.UpdatedAfter, task.Timestamps.Updated, true},
		{filters.UpdatedBefore, task.Timestamps.Updated, false},
	}

	for _, check := range dateChecks {
		if check.filterDate != nil {
			if check.isAfter && check.taskDate.Before(*check.filterDate) {
				return false
			}
			if !check.isAfter && check.taskDate.After(*check.filterDate) {
				return false
			}
		}
	}

	return true
}

// matchesQualityFilters checks quality score ranges
func (fm *FilterManager) matchesQualityFilters(task *types.Task, filters *TaskFilters) bool {
	if filters.MinQualityScore > 0 && task.QualityScore.OverallScore < filters.MinQualityScore {
		return false
	}
	if filters.MaxQualityScore > 0 && task.QualityScore.OverallScore > filters.MaxQualityScore {
		return false
	}
	return true
}

// containsString checks if a slice contains a specific string value
func (fm *FilterManager) containsString(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// statusToStrings converts TaskStatus slice to string slice
func (fm *FilterManager) statusToStrings(statuses []types.TaskStatus) []string {
	result := make([]string, len(statuses))
	for i, status := range statuses {
		result[i] = string(status)
	}
	return result
}

// typeToStrings converts TaskType slice to string slice
func (fm *FilterManager) typeToStrings(taskTypes []types.TaskType) []string {
	result := make([]string, len(taskTypes))
	for i, taskType := range taskTypes {
		result[i] = string(taskType)
	}
	return result
}

// priorityToStrings converts TaskPriority slice to string slice
func (fm *FilterManager) priorityToStrings(priorities []types.TaskPriority) []string {
	result := make([]string, len(priorities))
	for i, priority := range priorities {
		result[i] = string(priority)
	}
	return result
}

func (fm *FilterManager) sortTasks(tasks []types.Task, sortFields []SortField) []types.Task {
	// Simple implementation - in production would use more sophisticated sorting
	// For now, just sort by the first field
	if len(sortFields) == 0 {
		return tasks
	}

	// This is a simplified implementation
	// In production, you'd implement proper multi-field sorting
	return tasks
}

func (fm *FilterManager) paginateTasks(tasks []types.Task, offset, limit int) []types.Task {
	if offset >= len(tasks) {
		return []types.Task{}
	}

	end := offset + limit
	if limit == 0 || end > len(tasks) {
		end = len(tasks)
	}

	return tasks[offset:end]
}

// PerformTextSearch performs full-text search on tasks
func (fm *FilterManager) PerformTextSearch(tasks []types.Task, query string, options SearchOptions) *SearchResults {
	startTime := time.Now()

	var matchedTasks []types.Task
	highlights := make(map[string][]string)

	// Clean and prepare query
	cleanQuery := strings.TrimSpace(query)
	if !fm.config.CaseSensitive {
		cleanQuery = strings.ToLower(cleanQuery)
	}

	// Search fields to check
	searchFields := options.SearchFields
	if len(searchFields) == 0 {
		searchFields = []string{FieldTitle, FieldDescription, "tags"}
	}

	for i := range tasks {
		task := &tasks[i]
		matched := false
		taskHighlights := make([]string, 0)

		// Check each search field
		for _, field := range searchFields {
			var content string
			switch field {
			case FieldTitle:
				content = task.Title
			case FieldDescription:
				content = task.Description
			case "tags":
				content = strings.Join(task.Tags, " ")
			default:
				continue
			}

			if !fm.config.CaseSensitive {
				content = strings.ToLower(content)
			}

			// Simple contains search (could be enhanced with fuzzy matching)
			if strings.Contains(content, cleanQuery) {
				matched = true
				if options.HighlightMatches {
					taskHighlights = append(taskHighlights, fm.highlightText(content, cleanQuery))
				}
			}
		}

		if matched {
			matchedTasks = append(matchedTasks, *task)
			if len(taskHighlights) > 0 {
				highlights[task.ID] = taskHighlights
			}
		}

		// Apply max results limit
		if options.MaxResults > 0 && len(matchedTasks) >= options.MaxResults {
			break
		}
	}

	return &SearchResults{
		Tasks:        matchedTasks,
		TotalResults: len(matchedTasks),
		SearchTime:   time.Since(startTime),
		Query:        query,
		Highlights:   highlights,
	}
}

func (fm *FilterManager) highlightText(content, query string) string {
	// Simple highlighting - wrap matches in <mark> tags
	re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(query))
	return re.ReplaceAllStringFunc(content, func(match string) string {
		return "<mark>" + match + "</mark>"
	})
}
