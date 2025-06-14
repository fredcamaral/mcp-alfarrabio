// Package retrieve provides the memory_retrieve tool implementation.
// Handles all data retrieval operations with flexible access levels.
package retrieve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"lerian-mcp-memory/internal/session"
	"lerian-mcp-memory/internal/tools"
	"lerian-mcp-memory/internal/types"
	"lerian-mcp-memory/internal/validation"
	pkgTypes "lerian-mcp-memory/pkg/types"
)

// SearchStore defines the interface for search operations.
type SearchStore interface {
	Search(ctx context.Context, query *types.SearchQuery) (*types.SearchResults, error)
	FindSimilar(ctx context.Context, content string, projectID types.ProjectID, sessionID types.SessionID) ([]*types.Content, error)
	GetByProject(ctx context.Context, projectID types.ProjectID, filters *types.Filters) ([]*types.Content, error)
	GetBySession(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID, filters *types.Filters) ([]*types.Content, error)
	GetHistory(ctx context.Context, projectID types.ProjectID, contentID string) ([]*types.ContentVersion, error)
}

type ContentStore interface {
	Get(ctx context.Context, projectID types.ProjectID, contentID string) (*types.Content, error)
}

// Handler implements the memory_retrieve tool
type Handler struct {
	sessionManager *session.Manager
	validator      *validation.ParameterValidator
	searchStore    SearchStore
	contentStore   ContentStore
}

// NewHandler creates a new retrieve handler
func NewHandler(sessionManager *session.Manager, validator *validation.ParameterValidator, searchStore SearchStore, contentStore ContentStore) *Handler {
	return &Handler{
		sessionManager: sessionManager,
		validator:      validator,
		searchStore:    searchStore,
		contentStore:   contentStore,
	}
}

// SearchRequest represents a search request
type SearchRequest struct {
	types.StandardParams
	Query             string               `json:"query"`
	Types             []pkgTypes.ChunkType `json:"types,omitempty"`
	Tags              []string             `json:"tags,omitempty"`
	DateRange         *DateRange           `json:"date_range,omitempty"`
	MinRelevanceScore float64              `json:"min_relevance_score,omitempty"`
	Limit             int                  `json:"limit,omitempty"`
	IncludeMetadata   bool                 `json:"include_metadata,omitempty"`
	IncludeEmbeddings bool                 `json:"include_embeddings,omitempty"`
	SessionScope      bool                 `json:"session_scope,omitempty"` // If true, search only session data
}

// DateRange represents a date filter range
type DateRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// GetContentRequest represents a request to get specific content
type GetContentRequest struct {
	types.StandardParams
	ContentID         string `json:"content_id"`
	IncludeMetadata   bool   `json:"include_metadata,omitempty"`
	IncludeEmbeddings bool   `json:"include_embeddings,omitempty"`
	IncludeRelations  bool   `json:"include_relations,omitempty"`
}

// FindSimilarRequest represents a request to find similar content
type FindSimilarRequest struct {
	types.StandardParams
	Content           string               `json:"content"`
	Types             []pkgTypes.ChunkType `json:"types,omitempty"`
	MinSimilarity     float64              `json:"min_similarity,omitempty"`
	Limit             int                  `json:"limit,omitempty"`
	ExcludeContentIDs []string             `json:"exclude_content_ids,omitempty"`
}

// GetThreadsRequest represents a request to get conversation threads
type GetThreadsRequest struct {
	types.StandardParams
	ThreadID     string     `json:"thread_id,omitempty"`  // Get specific thread
	Tags         []string   `json:"tags,omitempty"`       // Filter by tags
	DateRange    *DateRange `json:"date_range,omitempty"` // Filter by date
	Limit        int        `json:"limit,omitempty"`
	IncludeEmpty bool       `json:"include_empty,omitempty"` // Include threads with no content
}

// SearchResponse represents search results
type SearchResponse struct {
	Results   []ContentResult `json:"results"`
	Total     int             `json:"total"`
	QueryTime time.Duration   `json:"query_time"`
	HasMore   bool            `json:"has_more"`
}

// ContentResult represents a single content result
type ContentResult struct {
	ContentID  string                 `json:"content_id"`
	Type       pkgTypes.ChunkType     `json:"type"`
	Content    string                 `json:"content"`
	Summary    string                 `json:"summary,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	ThreadID   string                 `json:"thread_id,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  *time.Time             `json:"updated_at,omitempty"`
	Score      float64                `json:"score,omitempty"` // Relevance/similarity score
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Embeddings []float64              `json:"embeddings,omitempty"`
	Relations  []Relation             `json:"relations,omitempty"`
}

// Relation represents a relationship between content items
type Relation struct {
	RelationID   string    `json:"relation_id"`
	ToContentID  string    `json:"to_content_id"`
	RelationType string    `json:"relation_type"`
	Description  string    `json:"description,omitempty"`
	Strength     float64   `json:"strength,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// HandleOperation handles all retrieve operations
func (h *Handler) HandleOperation(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error) {
	switch operation {
	case string(tools.OpSearchContent):
		return h.handleSearch(ctx, params)
	case string(tools.OpGetContentByID):
		return h.handleGetContent(ctx, params)
	case string(tools.OpFindSimilarContent):
		return h.handleFindSimilar(ctx, params)
	case string(tools.OpGetContentByProject):
		return h.handleGetThreads(ctx, params)
	case string(tools.OpGetContentRelationships):
		return h.handleGetRelationships(ctx, params)
	case string(tools.OpGetContentHistory):
		return h.handleGetHistory(ctx, params)
	default:
		return nil, fmt.Errorf("unknown retrieve operation: %s", operation)
	}
}

// handleSearch performs semantic search across content
func (h *Handler) handleSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	req, err := h.parseSearchRequest(params)
	if err != nil {
		return nil, err
	}

	if err := h.validateSearchRequest(req); err != nil {
		return nil, err
	}

	// Update session access if session provided
	if !req.SessionID.IsEmpty() {
		if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
			return nil, fmt.Errorf("failed to update session access: %w", err)
		}
	}

	searchQuery := h.buildSearchQuery(req)
	searchResults, err := h.performSearch(ctx, searchQuery)
	if err != nil {
		return nil, err
	}

	return h.buildSearchResponse(req, searchResults), nil
}

// parseSearchRequest parses the search request parameters
func (h *Handler) parseSearchRequest(params map[string]interface{}) (*SearchRequest, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req SearchRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse search request: %w", err)
	}

	return &req, nil
}

// validateSearchRequest validates the search request parameters
func (h *Handler) validateSearchRequest(req *SearchRequest) error {
	if err := h.validator.ValidateOperation(string(tools.OpSearchContent), &req.StandardParams); err != nil {
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.Query == "" {
		return errors.New("query cannot be empty")
	}

	h.setSearchDefaults(req)
	return nil
}

// setSearchDefaults sets default values for search request
func (h *Handler) setSearchDefaults(req *SearchRequest) {
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.MinRelevanceScore <= 0 {
		req.MinRelevanceScore = 0.3
	}
}

// buildSearchQuery builds the search query from the request
func (h *Handler) buildSearchQuery(req *SearchRequest) *types.SearchQuery {
	searchQuery := &types.SearchQuery{
		ProjectID:    req.ProjectID,
		SessionID:    req.SessionID,
		Query:        req.Query,
		Limit:        req.Limit,
		MinRelevance: req.MinRelevanceScore,
		SortBy:       "relevance",
		SortOrder:    "desc",
	}

	h.addTypesFilter(searchQuery, req.Types)
	h.addFilters(searchQuery, req)
	h.setSessionScope(searchQuery, req)

	return searchQuery
}

// addTypesFilter adds type filter to search query
func (h *Handler) addTypesFilter(searchQuery *types.SearchQuery, reqTypes []pkgTypes.ChunkType) {
	if len(reqTypes) == 0 {
		return
	}

	searchQuery.Types = make([]string, len(reqTypes))
	for i, t := range reqTypes {
		searchQuery.Types[i] = string(t)
	}
}

// addFilters adds date range and tag filters to search query
func (h *Handler) addFilters(searchQuery *types.SearchQuery, req *SearchRequest) {
	if req.DateRange == nil && len(req.Tags) == 0 {
		return
	}

	filters := &types.Filters{}

	if len(req.Tags) > 0 {
		filters.Tags = req.Tags
	}

	h.addDateRangeFilter(filters, req.DateRange)
	searchQuery.Filters = filters
}

// addDateRangeFilter adds date range filter to filters
func (h *Handler) addDateRangeFilter(filters *types.Filters, dateRange *DateRange) {
	if dateRange == nil {
		return
	}

	if dateRange.Start != nil {
		filters.CreatedAfter = dateRange.Start
	}
	if dateRange.End != nil {
		filters.CreatedBefore = dateRange.End
	}
}

// setSessionScope sets session scope if requested
func (h *Handler) setSessionScope(searchQuery *types.SearchQuery, req *SearchRequest) {
	if req.SessionScope && !req.SessionID.IsEmpty() {
		searchQuery.SessionID = req.SessionID
	}
}

// performSearch executes the search query
func (h *Handler) performSearch(ctx context.Context, searchQuery *types.SearchQuery) (*types.SearchResults, error) {
	searchResults, err := h.searchStore.Search(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	return searchResults, nil
}

// buildSearchResponse builds the search response from results
func (h *Handler) buildSearchResponse(req *SearchRequest, searchResults *types.SearchResults) *SearchResponse {
	results := h.convertSearchResults(req, searchResults.Results)

	return &SearchResponse{
		Results:   results,
		Total:     searchResults.Total,
		QueryTime: searchResults.Duration,
		HasMore:   len(results) == req.Limit, // Simple heuristic
	}
}

// convertSearchResults converts search results to response format
func (h *Handler) convertSearchResults(req *SearchRequest, results []*types.SearchResult) []ContentResult {
	converted := make([]ContentResult, 0, len(results))
	for _, result := range results {
		contentResult := h.convertSingleResult(req, result)
		converted = append(converted, contentResult)
	}
	return converted
}

// convertSingleResult converts a single search result to response format
func (h *Handler) convertSingleResult(req *SearchRequest, result *types.SearchResult) ContentResult {
	contentResult := ContentResult{
		ContentID: result.Content.ID,
		Type:      pkgTypes.ChunkType(result.Content.Type),
		Content:   result.Content.Content,
		Summary:   result.Content.Summary,
		Tags:      result.Content.Tags,
		ThreadID:  result.Content.ThreadID,
		CreatedAt: result.Content.CreatedAt,
		Score:     result.Relevance,
	}

	h.setUpdatedAt(&contentResult, result)
	h.setOptionalFields(&contentResult, req, result)

	return contentResult
}

// setUpdatedAt sets the updated at field if content was updated
func (h *Handler) setUpdatedAt(contentResult *ContentResult, result *types.SearchResult) {
	if result.Content.UpdatedAt.After(result.Content.CreatedAt) {
		contentResult.UpdatedAt = &result.Content.UpdatedAt
	}
}

// setOptionalFields sets optional fields based on request parameters
func (h *Handler) setOptionalFields(contentResult *ContentResult, req *SearchRequest, result *types.SearchResult) {
	if req.IncludeMetadata && result.Content.Metadata != nil {
		contentResult.Metadata = result.Content.Metadata
	}

	if req.IncludeEmbeddings && len(result.Content.Embeddings) > 0 {
		contentResult.Embeddings = result.Content.Embeddings
	}
}

// handleGetContent retrieves specific content by ID
func (h *Handler) handleGetContent(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req GetContentRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse get content request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpGetContentByID), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.ContentID == "" {
		return nil, errors.New("content_id is required")
	}

	// Update session access if session provided
	if !req.SessionID.IsEmpty() {
		if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
			return nil, fmt.Errorf("failed to update session access: %w", err)
		}
	}

	// Get content using real storage implementation
	content, err := h.contentStore.Get(ctx, req.ProjectID, req.ContentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get content: %w", err)
	}
	if content == nil {
		return nil, fmt.Errorf("content with ID %s not found", req.ContentID)
	}

	// Convert to response format
	result := &ContentResult{
		ContentID: content.ID,
		Type:      pkgTypes.ChunkType(content.Type),
		Content:   content.Content,
		Summary:   content.Summary,
		Tags:      content.Tags,
		ThreadID:  content.ThreadID,
		CreatedAt: content.CreatedAt,
		Score:     1.0, // Direct retrieval has perfect score
	}

	if content.UpdatedAt.After(content.CreatedAt) {
		result.UpdatedAt = &content.UpdatedAt
	}

	// Include metadata if requested
	if req.IncludeMetadata && content.Metadata != nil {
		result.Metadata = content.Metadata
	}

	// Include embeddings if requested
	if req.IncludeEmbeddings && len(content.Embeddings) > 0 {
		result.Embeddings = content.Embeddings
	}

	// TODO: Include relations if requested - requires relationship store implementation
	if req.IncludeRelations {
		// For now, leave relations empty until we implement relationship store
		result.Relations = []Relation{}
	}

	return result, nil
}

// handleFindSimilar finds content similar to provided text
func (h *Handler) handleFindSimilar(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	req, err := h.parseFindSimilarRequest(params)
	if err != nil {
		return nil, err
	}

	if err := h.validateFindSimilarRequest(req); err != nil {
		return nil, err
	}

	if err := h.updateSessionAccessIfNeeded(req.ProjectID, req.SessionID); err != nil {
		return nil, err
	}

	start := time.Now()
	similarContent, err := h.findSimilarContent(ctx, req)
	if err != nil {
		return nil, err
	}

	results := h.filterAndConvertSimilarResults(req, similarContent)

	return h.buildSimilarSearchResponse(results, similarContent, start), nil
}

// parseFindSimilarRequest parses the find similar request parameters
func (h *Handler) parseFindSimilarRequest(params map[string]interface{}) (*FindSimilarRequest, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req FindSimilarRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse find similar request: %w", err)
	}

	return &req, nil
}

// validateFindSimilarRequest validates the find similar request parameters
func (h *Handler) validateFindSimilarRequest(req *FindSimilarRequest) error {
	if err := h.validator.ValidateOperation(string(tools.OpFindSimilarContent), &req.StandardParams); err != nil {
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.Content == "" {
		return errors.New("content is required for similarity search")
	}

	h.setFindSimilarDefaults(req)
	return nil
}

// setFindSimilarDefaults sets default values for find similar request
func (h *Handler) setFindSimilarDefaults(req *FindSimilarRequest) {
	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.MinSimilarity <= 0 {
		req.MinSimilarity = 0.5
	}
}

// updateSessionAccessIfNeeded updates session access if session is provided
func (h *Handler) updateSessionAccessIfNeeded(projectID types.ProjectID, sessionID types.SessionID) error {
	if sessionID.IsEmpty() {
		return nil
	}

	return h.sessionManager.UpdateSessionAccess(projectID, sessionID)
}

// findSimilarContent performs the similarity search
func (h *Handler) findSimilarContent(ctx context.Context, req *FindSimilarRequest) ([]*types.Content, error) {
	similarContent, err := h.searchStore.FindSimilar(ctx, req.Content, req.ProjectID, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("similarity search failed: %w", err)
	}
	return similarContent, nil
}

// filterAndConvertSimilarResults filters and converts similar content to results
func (h *Handler) filterAndConvertSimilarResults(req *FindSimilarRequest, similarContent []*types.Content) []ContentResult {
	excludeSet := h.buildExcludeSet(req.ExcludeContentIDs)
	results := make([]ContentResult, 0)

	for _, content := range similarContent {
		if h.shouldExcludeContent(content, excludeSet, req) {
			continue
		}

		score := calculateTextSimilarity(req.Content, content.Content)
		if score < req.MinSimilarity {
			continue
		}

		contentResult := h.createContentResult(content, score)
		results = append(results, contentResult)

		if len(results) >= req.Limit {
			break
		}
	}

	return results
}

// buildExcludeSet creates a set for excluded content IDs
func (h *Handler) buildExcludeSet(excludeIDs []string) map[string]bool {
	excludeSet := make(map[string]bool)
	for _, id := range excludeIDs {
		excludeSet[id] = true
	}
	return excludeSet
}

// shouldExcludeContent checks if content should be excluded
func (h *Handler) shouldExcludeContent(content *types.Content, excludeSet map[string]bool, req *FindSimilarRequest) bool {
	// Check exclude list
	if excludeSet[content.ID] {
		return true
	}

	// Check type filter
	return h.shouldExcludeByType(content, req.Types)
}

// shouldExcludeByType checks if content should be excluded by type
func (h *Handler) shouldExcludeByType(content *types.Content, chunkTypes []pkgTypes.ChunkType) bool {
	if len(chunkTypes) == 0 {
		return false
	}

	for _, filterType := range chunkTypes {
		if content.Type == string(filterType) {
			return false
		}
	}
	return true
}

// createContentResult creates a content result from content and score
func (h *Handler) createContentResult(content *types.Content, score float64) ContentResult {
	contentResult := ContentResult{
		ContentID: content.ID,
		Type:      pkgTypes.ChunkType(content.Type),
		Content:   content.Content,
		Summary:   content.Summary,
		Tags:      content.Tags,
		ThreadID:  content.ThreadID,
		CreatedAt: content.CreatedAt,
		Score:     score,
	}

	if content.UpdatedAt.After(content.CreatedAt) {
		contentResult.UpdatedAt = &content.UpdatedAt
	}

	return contentResult
}

// buildSimilarSearchResponse builds the search response for similar content
func (h *Handler) buildSimilarSearchResponse(results []ContentResult, similarContent []*types.Content, start time.Time) *SearchResponse {
	return &SearchResponse{
		Results:   results,
		Total:     len(results),
		QueryTime: time.Since(start),
		HasMore:   len(similarContent) > len(results),
	}
}

// handleGetThreads retrieves conversation threads
func (h *Handler) handleGetThreads(ctx context.Context, params map[string]interface{}) (interface{}, error) { //nolint:unparam // context part of MCP handler interface
	// TODO: Implement thread retrieval
	return map[string]interface{}{
		"threads": []map[string]interface{}{
			{
				"thread_id":     "thread_1",
				"title":         "Mock Thread",
				"description":   "A mock conversation thread",
				"content_count": 5,
				"created_at":    time.Now().Add(-24 * time.Hour),
				"last_updated":  time.Now().Add(-1 * time.Hour),
			},
		},
		"total": 1,
	}, nil
}

// handleGetRelationships retrieves relationships between content
func (h *Handler) handleGetRelationships(ctx context.Context, params map[string]interface{}) (interface{}, error) { //nolint:unparam // context part of MCP handler interface
	// TODO: Implement relationship retrieval
	return map[string]interface{}{
		"relationships": []Relation{
			{
				RelationID:   "rel_1",
				ToContentID:  "content_2",
				RelationType: "references",
				Description:  "Mock relationship",
				Strength:     0.9,
				CreatedAt:    time.Now().Add(-2 * time.Hour),
			},
		},
		"total": 1,
	}, nil
}

// handleGetHistory retrieves content change history
func (h *Handler) handleGetHistory(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse request
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req GetContentRequest // Reuse this struct since it has the needed fields
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse get history request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpGetContentHistory), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.ContentID == "" {
		return nil, errors.New("content_id is required for history retrieval")
	}

	// Update session access if session provided
	if !req.SessionID.IsEmpty() {
		if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
			return nil, fmt.Errorf("failed to update session access: %w", err)
		}
	}

	// Get history using real storage implementation
	history, err := h.searchStore.GetHistory(ctx, req.ProjectID, req.ContentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get content history: %w", err)
	}

	// Convert to response format
	historyResults := make([]map[string]interface{}, len(history))
	for i, version := range history {
		historyResults[i] = map[string]interface{}{
			"content_id": version.ContentID,
			"version":    version.Version,
			"content":    version.Content,
			"summary":    version.Summary,
			"changes":    version.Changes,
			"changed_by": version.ChangedBy,
			"changed_at": version.ChangedAt,
			"metadata":   version.Metadata,
		}
	}

	return map[string]interface{}{
		"history": historyResults,
		"total":   len(historyResults),
	}, nil
}

// GetToolDefinition returns the MCP tool definition for memory_retrieve
func (h *Handler) GetToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        string(tools.ToolMemoryRetrieve),
		"description": "Retrieve and search memory content with flexible access levels. Supports semantic search, content retrieval, similarity matching, thread exploration, and relationship discovery.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"enum":        tools.GetOperationsForTool(tools.ToolMemoryRetrieve),
					"description": "The retrieve operation to perform",
				},
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "Project identifier for data isolation (required)",
				},
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session identifier for expanded access (optional - without it, you get read-only project access)",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query for semantic search operations",
				},
				"content_id": map[string]interface{}{
					"type":        "string",
					"description": "Content ID for specific content retrieval",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content text for similarity search",
				},
				"types": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Filter by content types",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return",
					"default":     10,
				},
				"min_relevance_score": map[string]interface{}{
					"type":        "number",
					"description": "Minimum relevance score for search results",
					"default":     0.3,
				},
			},
			"required": []string{"operation", "project_id"},
		},
	}
}

// calculateTextSimilarity calculates simple text similarity between two strings
// This is a placeholder for proper vector similarity scoring
func calculateTextSimilarity(text1, text2 string) float64 {
	if text1 == "" || text2 == "" {
		return 0.0
	}

	// Simple word overlap calculation
	words1 := make(map[string]bool)
	words2 := make(map[string]bool)

	// Tokenize and normalize
	for _, word := range strings.Fields(strings.ToLower(text1)) {
		words1[word] = true
	}
	for _, word := range strings.Fields(strings.ToLower(text2)) {
		words2[word] = true
	}

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Calculate intersection
	intersection := 0
	for word := range words1 {
		if words2[word] {
			intersection++
		}
	}

	// Jaccard similarity
	union := len(words1) + len(words2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}
