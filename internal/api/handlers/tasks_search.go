// Package handlers provides HTTP handlers for task search functionality.
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"lerian-mcp-memory/internal/api/response"
	"lerian-mcp-memory/internal/tasks"
	"lerian-mcp-memory/pkg/types"
)

// TaskSearchHandler handles task search operations
type TaskSearchHandler struct {
	service *tasks.Service
	config  TaskSearchConfig
}

// TaskSearchConfig represents configuration for task search
type TaskSearchConfig struct {
	MaxQueryLength     int           `json:"max_query_length"`
	MinQueryLength     int           `json:"min_query_length"`
	DefaultMaxResults  int           `json:"default_max_results"`
	MaxMaxResults      int           `json:"max_max_results"`
	RequestTimeout     time.Duration `json:"request_timeout"`
	EnableHighlighting bool          `json:"enable_highlighting"`
	EnableFuzzySearch  bool          `json:"enable_fuzzy_search"`
	CacheSearchResults bool          `json:"cache_search_results"`
	CacheTTL           time.Duration `json:"cache_ttl"`
}

// DefaultTaskSearchConfig returns default search configuration
func DefaultTaskSearchConfig() TaskSearchConfig {
	return TaskSearchConfig{
		MaxQueryLength:     1000,
		MinQueryLength:     2,
		DefaultMaxResults:  20,
		MaxMaxResults:      100,
		RequestTimeout:     30 * time.Second,
		EnableHighlighting: true,
		EnableFuzzySearch:  true,
		CacheSearchResults: true,
		CacheTTL:           5 * time.Minute,
	}
}

// NewTaskSearchHandler creates a new task search handler
func NewTaskSearchHandler(service *tasks.Service, config TaskSearchConfig) *TaskSearchHandler {
	return &TaskSearchHandler{
		service: service,
		config:  config,
	}
}

// SearchTasks handles full-text search requests
func (h *TaskSearchHandler) SearchTasks(w http.ResponseWriter, r *http.Request) {
	// Parse search query
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		response.WriteError(w, http.StatusBadRequest, "Search query is required", "Parameter 'q' cannot be empty")
		return
	}

	// Validate query
	if err := h.validateSearchQuery(query); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid search query", err.Error())
		return
	}

	// Parse search options
	options, err := h.parseSearchOptions(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid search options", err.Error())
		return
	}

	// Parse filters
	filters := h.parseFilters(r)

	// Build search query
	searchQuery := tasks.SearchQuery{
		Query:   query,
		Filters: filters,
		Options: options,
	}

	// Get user ID from context
	userID := h.getUserID(r)

	// Perform search
	results, err := h.service.SearchTasks(r.Context(), &searchQuery, userID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Search failed", err.Error())
		return
	}

	// Build response
	searchResponse := TaskSearchResponse{
		Results:     *results,
		SearchQuery: searchQuery,
		UserID:      userID,
		Timestamp:   time.Now(),
		Suggestions: h.generateSearchSuggestions(query, results),
	}

	response.WriteSuccess(w, searchResponse)
}

// AdvancedSearch handles advanced search with complex criteria
func (h *TaskSearchHandler) AdvancedSearch(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req AdvancedSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Validate request
	if err := h.validateAdvancedSearchRequest(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid search request", err.Error())
		return
	}

	// Get user ID from context
	userID := h.getUserID(r)

	// Convert request to search query
	searchQuery := h.buildAdvancedSearchQuery(&req)

	// Perform search
	results, err := h.service.SearchTasks(r.Context(), &searchQuery, userID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Search failed", err.Error())
		return
	}

	// Build response with additional analytics
	searchResponse := AdvancedSearchResponse{
		Results:     *results,
		SearchQuery: searchQuery,
		Analytics:   h.generateSearchAnalytics(results),
		Facets:      h.generateFacets(results),
		UserID:      userID,
		Timestamp:   time.Now(),
	}

	response.WriteSuccess(w, searchResponse)
}

// GetSearchSuggestions handles search suggestion requests
func (h *TaskSearchHandler) GetSearchSuggestions(w http.ResponseWriter, r *http.Request) {
	// Parse partial query
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) < h.config.MinQueryLength {
		response.WriteError(w, http.StatusBadRequest, "Query too short",
			fmt.Sprintf("Minimum query length is %d characters", h.config.MinQueryLength))
		return
	}

	// Get user ID from context
	userID := h.getUserID(r)

	// Generate suggestions (in a real app, this might use a dedicated suggestion engine)
	suggestions := h.generateQuerySuggestions(query, userID)

	response.WriteSuccess(w, SearchSuggestionsResponse{
		Query:       query,
		Suggestions: suggestions,
		GeneratedAt: time.Now(),
	})
}

// GetSearchHistory handles search history requests
func (h *TaskSearchHandler) GetSearchHistory(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID := h.getUserID(r)

	// Parse limit
	limit := 10 // default
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	// Get search history (in a real app, this would come from a search history store)
	history := h.getSearchHistory(userID, limit)

	response.WriteSuccess(w, SearchHistoryResponse{
		History:     history,
		TotalCount:  len(history),
		UserID:      userID,
		RetrievedAt: time.Now(),
	})
}

// Helper methods

func (h *TaskSearchHandler) validateSearchQuery(query string) error {
	if len(query) < h.config.MinQueryLength {
		return fmt.Errorf("query too short (minimum %d characters)", h.config.MinQueryLength)
	}
	if len(query) > h.config.MaxQueryLength {
		return fmt.Errorf("query too long (maximum %d characters)", h.config.MaxQueryLength)
	}

	// Check for potentially malicious input
	maliciousPatterns := []string{
		"<script", "javascript:", "onload=", "onerror=", "eval(",
		"union select", "drop table", "delete from",
	}

	queryLower := strings.ToLower(query)
	for _, pattern := range maliciousPatterns {
		if strings.Contains(queryLower, pattern) {
			return fmt.Errorf("query contains potentially malicious content")
		}
	}

	return nil
}

func (h *TaskSearchHandler) parseSearchOptions(r *http.Request) (tasks.SearchOptions, error) {
	options := tasks.SearchOptions{
		HighlightMatches: h.config.EnableHighlighting,
		FuzzySearch:      h.config.EnableFuzzySearch,
		MaxResults:       h.config.DefaultMaxResults,
	}

	// Parse max results
	if maxResultsParam := r.URL.Query().Get("max_results"); maxResultsParam != "" {
		if maxResults, err := strconv.Atoi(maxResultsParam); err == nil && maxResults > 0 {
			if maxResults > h.config.MaxMaxResults {
				maxResults = h.config.MaxMaxResults
			}
			options.MaxResults = maxResults
		}
	}

	// Parse highlighting option
	if highlightParam := r.URL.Query().Get("highlight"); highlightParam != "" {
		options.HighlightMatches = (highlightParam == "true" || highlightParam == "1")
	}

	// Parse fuzzy search option
	if fuzzyParam := r.URL.Query().Get("fuzzy"); fuzzyParam != "" {
		options.FuzzySearch = (fuzzyParam == "true" || fuzzyParam == "1")
	}

	// Parse search fields
	if fieldsParam := r.URL.Query().Get("fields"); fieldsParam != "" {
		options.SearchFields = strings.Split(fieldsParam, ",")

		// Validate search fields
		validFields := map[string]bool{
			"title": true, "description": true, "tags": true,
			"acceptance_criteria": true, "type": true,
		}

		for _, field := range options.SearchFields {
			if !validFields[strings.TrimSpace(field)] {
				return options, fmt.Errorf("invalid search field: %s", field)
			}
		}
	}

	return options, nil
}

func (h *TaskSearchHandler) parseFilters(r *http.Request) tasks.TaskFilters {
	filters := tasks.TaskFilters{}
	query := r.URL.Query()

	h.parseStringArrayFilters(&filters, query)
	h.parseSimpleStringFilters(&filters, query)
	h.parseDateFilters(&filters, query)
	h.parseQualityScoreFilters(&filters, query)

	return filters
}

// parseStringArrayFilters handles comma-separated array filters
func (h *TaskSearchHandler) parseStringArrayFilters(filters *tasks.TaskFilters, query url.Values) {
	arrayFilters := map[string]func(string){
		"status": func(value string) {
			for _, status := range h.splitAndTrim(value) {
				filters.Status = append(filters.Status, types.TaskStatus(status))
			}
		},
		"type": func(value string) {
			for _, taskType := range h.splitAndTrim(value) {
				filters.Type = append(filters.Type, types.TaskType(taskType))
			}
		},
		"priority": func(value string) {
			for _, priority := range h.splitAndTrim(value) {
				filters.Priority = append(filters.Priority, types.TaskPriority(priority))
			}
		},
		"tags": func(value string) {
			filters.Tags = h.splitAndTrim(value)
		},
	}

	for param, parser := range arrayFilters {
		if value := query.Get(param); value != "" {
			parser(value)
		}
	}
}

// parseSimpleStringFilters handles single-value string filters
func (h *TaskSearchHandler) parseSimpleStringFilters(filters *tasks.TaskFilters, query url.Values) {
	filters.Assignee = query.Get("assignee")
	filters.Repository = query.Get("repository")
}

// parseDateFilters handles date-based filters
func (h *TaskSearchHandler) parseDateFilters(filters *tasks.TaskFilters, query url.Values) {
	dateFilters := map[string]*time.Time{
		"created_after":  nil,
		"created_before": nil,
		"updated_after":  nil,
		"updated_before": nil,
	}

	for param := range dateFilters {
		if dateParam := query.Get(param); dateParam != "" {
			if parsedDate, err := time.Parse(time.RFC3339, dateParam); err == nil {
				switch param {
				case "created_after":
					filters.CreatedAfter = &parsedDate
				case "created_before":
					filters.CreatedBefore = &parsedDate
				case "updated_after":
					filters.UpdatedAfter = &parsedDate
				case "updated_before":
					filters.UpdatedBefore = &parsedDate
				}
			}
		}
	}
}

// parseQualityScoreFilters handles quality score range filters
func (h *TaskSearchHandler) parseQualityScoreFilters(filters *tasks.TaskFilters, query url.Values) {
	scoreFilters := map[string]func(float64){
		"min_quality_score": func(score float64) { filters.MinQualityScore = score },
		"max_quality_score": func(score float64) { filters.MaxQualityScore = score },
	}

	for param, setter := range scoreFilters {
		if scoreParam := query.Get(param); scoreParam != "" {
			if score, err := strconv.ParseFloat(scoreParam, 64); err == nil && score >= 0 && score <= 1 {
				setter(score)
			}
		}
	}
}

// splitAndTrim splits a comma-separated string and trims whitespace
func (h *TaskSearchHandler) splitAndTrim(value string) []string {
	items := strings.Split(value, ",")
	for i, item := range items {
		items[i] = strings.TrimSpace(item)
	}
	return items
}

func (h *TaskSearchHandler) validateAdvancedSearchRequest(req *AdvancedSearchRequest) error {
	if req.Query == "" {
		return fmt.Errorf("search query is required")
	}
	if err := h.validateSearchQuery(req.Query); err != nil {
		return err
	}
	if req.Options.MaxResults > h.config.MaxMaxResults {
		return fmt.Errorf("max_results cannot exceed %d", h.config.MaxMaxResults)
	}
	return nil
}

func (h *TaskSearchHandler) buildAdvancedSearchQuery(req *AdvancedSearchRequest) tasks.SearchQuery {
	return tasks.SearchQuery{
		Query:   req.Query,
		Filters: req.Filters,
		Options: req.Options,
	}
}

func (h *TaskSearchHandler) generateSearchSuggestions(query string, results *tasks.SearchResults) []string {
	suggestions := make([]string, 0)

	// Simple suggestion logic - in production this would be more sophisticated
	if len(results.Tasks) == 0 {
		// No results found, suggest broader terms
		suggestions = append(suggestions, fmt.Sprintf("Try searching for '%s'", strings.ToLower(query)))

		// Suggest removing words
		words := strings.Fields(query)
		if len(words) > 1 {
			suggestions = append(suggestions, fmt.Sprintf("Try '%s'", strings.Join(words[:len(words)-1], " ")))
		}
	} else if len(results.Tasks) < 5 {
		// Few results, suggest related terms
		suggestions = append(suggestions, fmt.Sprintf("Try '%s tasks'", query), fmt.Sprintf("Search for '%s implementation'", query))
	}

	return suggestions
}

func (h *TaskSearchHandler) generateSearchAnalytics(results *tasks.SearchResults) SearchAnalytics {
	analytics := SearchAnalytics{
		TotalResults:      results.TotalResults,
		SearchTime:        results.SearchTime,
		StatusBreakdown:   make(map[string]int),
		TypeBreakdown:     make(map[string]int),
		PriorityBreakdown: make(map[string]int),
	}

	for i := range results.Tasks {
		task := &results.Tasks[i]
		analytics.StatusBreakdown[string(task.Status)]++
		analytics.TypeBreakdown[string(task.Type)]++
		analytics.PriorityBreakdown[string(task.Priority)]++
	}

	return analytics
}

func (h *TaskSearchHandler) generateFacets(results *tasks.SearchResults) map[string]interface{} {
	facets := make(map[string]interface{})

	// Status facet
	statusCounts := make(map[string]int)
	typeCounts := make(map[string]int)
	assigneeCounts := make(map[string]int)

	for i := range results.Tasks {
		task := &results.Tasks[i]
		statusCounts[string(task.Status)]++
		typeCounts[string(task.Type)]++
		if task.Assignee != "" {
			assigneeCounts[task.Assignee]++
		}
	}

	facets["status"] = statusCounts
	facets["type"] = typeCounts
	facets["assignee"] = assigneeCounts

	return facets
}

func (h *TaskSearchHandler) generateQuerySuggestions(query, userID string) []string {
	_ = userID // unused parameter, kept for future user-specific suggestions
	suggestions := make([]string, 0)

	// Simple suggestion logic
	queryLower := strings.ToLower(query)

	// Common task-related suggestions
	commonQueries := []string{
		"bug fix", "feature implementation", "documentation", "testing",
		"review", "deployment", "architecture", "refactoring",
	}

	for _, common := range commonQueries {
		if strings.HasPrefix(common, queryLower) {
			suggestions = append(suggestions, common)
		}
	}

	// Limit suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

func (h *TaskSearchHandler) getSearchHistory(userID string, limit int) []SearchHistoryEntry {
	// In a real application, this would fetch from a search history store
	// For now, return empty history
	return []SearchHistoryEntry{}
}

func (h *TaskSearchHandler) getUserID(r *http.Request) string {
	// In a real application, this would extract user ID from JWT token or session
	if userID := r.Header.Get("X-User-ID"); userID != "" {
		return userID
	}
	return DefaultUserID
}

// Request/Response types

// AdvancedSearchRequest represents an advanced search request
type AdvancedSearchRequest struct {
	Query   string              `json:"query"`
	Filters tasks.TaskFilters   `json:"filters"`
	Options tasks.SearchOptions `json:"options"`
}

// TaskSearchResponse represents a search response
type TaskSearchResponse struct {
	Results     tasks.SearchResults `json:"results"`
	SearchQuery tasks.SearchQuery   `json:"search_query"`
	UserID      string              `json:"user_id"`
	Timestamp   time.Time           `json:"timestamp"`
	Suggestions []string            `json:"suggestions,omitempty"`
}

// AdvancedSearchResponse represents an advanced search response
type AdvancedSearchResponse struct {
	Results     tasks.SearchResults    `json:"results"`
	SearchQuery tasks.SearchQuery      `json:"search_query"`
	Analytics   SearchAnalytics        `json:"analytics"`
	Facets      map[string]interface{} `json:"facets"`
	UserID      string                 `json:"user_id"`
	Timestamp   time.Time              `json:"timestamp"`
}

// SearchAnalytics represents search result analytics
type SearchAnalytics struct {
	TotalResults      int            `json:"total_results"`
	SearchTime        time.Duration  `json:"search_time"`
	StatusBreakdown   map[string]int `json:"status_breakdown"`
	TypeBreakdown     map[string]int `json:"type_breakdown"`
	PriorityBreakdown map[string]int `json:"priority_breakdown"`
}

// SearchSuggestionsResponse represents search suggestions
type SearchSuggestionsResponse struct {
	Query       string    `json:"query"`
	Suggestions []string  `json:"suggestions"`
	GeneratedAt time.Time `json:"generated_at"`
}

// SearchHistoryResponse represents search history
type SearchHistoryResponse struct {
	History     []SearchHistoryEntry `json:"history"`
	TotalCount  int                  `json:"total_count"`
	UserID      string               `json:"user_id"`
	RetrievedAt time.Time            `json:"retrieved_at"`
}

// SearchHistoryEntry represents a search history entry
type SearchHistoryEntry struct {
	Query       string    `json:"query"`
	ResultCount int       `json:"result_count"`
	SearchedAt  time.Time `json:"searched_at"`
}
