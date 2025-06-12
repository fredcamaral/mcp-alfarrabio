// Package retrieve provides the memory_retrieve tool implementation.
// Handles all data retrieval operations with flexible access levels.
package retrieve

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/session"
	"lerian-mcp-memory/internal/tools"
	"lerian-mcp-memory/internal/types"
	"lerian-mcp-memory/internal/validation"
	pkgTypes "lerian-mcp-memory/pkg/types"
)

// Handler implements the memory_retrieve tool
type Handler struct {
	sessionManager *session.Manager
	validator      *validation.ParameterValidator
	// TODO: Add storage interfaces when ready
	// searchStore   storage.SearchStore
	// contentStore  storage.ContentStore
}

// NewHandler creates a new retrieve handler
func NewHandler(sessionManager *session.Manager, validator *validation.ParameterValidator) *Handler {
	return &Handler{
		sessionManager: sessionManager,
		validator:      validator,
	}
}

// SearchRequest represents a search request
type SearchRequest struct {
	types.StandardParams
	Query             string              `json:"query"`
	Types             []pkgTypes.ChunkType `json:"types,omitempty"`
	Tags              []string            `json:"tags,omitempty"`
	DateRange         *DateRange          `json:"date_range,omitempty"`
	MinRelevanceScore float64             `json:"min_relevance_score,omitempty"`
	Limit             int                 `json:"limit,omitempty"`
	IncludeMetadata   bool                `json:"include_metadata,omitempty"`
	IncludeEmbeddings bool                `json:"include_embeddings,omitempty"`
	SessionScope      bool                `json:"session_scope,omitempty"` // If true, search only session data
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
	Content           string              `json:"content"`
	Types             []pkgTypes.ChunkType `json:"types,omitempty"`
	MinSimilarity     float64             `json:"min_similarity,omitempty"`
	Limit             int                 `json:"limit,omitempty"`
	ExcludeContentIDs []string            `json:"exclude_content_ids,omitempty"`
}

// GetThreadsRequest represents a request to get conversation threads
type GetThreadsRequest struct {
	types.StandardParams
	ThreadID     string    `json:"thread_id,omitempty"`    // Get specific thread
	Tags         []string  `json:"tags,omitempty"`         // Filter by tags
	DateRange    *DateRange `json:"date_range,omitempty"`  // Filter by date
	Limit        int       `json:"limit,omitempty"`
	IncludeEmpty bool      `json:"include_empty,omitempty"` // Include threads with no content
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
	ContentID   string                 `json:"content_id"`
	Type        pkgTypes.ChunkType      `json:"type"`
	Content     string                 `json:"content"`
	Summary     string                 `json:"summary,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	ThreadID    string                 `json:"thread_id,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   *time.Time             `json:"updated_at,omitempty"`
	Score       float64                `json:"score,omitempty"`        // Relevance/similarity score
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Embeddings  []float64              `json:"embeddings,omitempty"`
	Relations   []Relation             `json:"relations,omitempty"`
}

// Relation represents a relationship between content items
type Relation struct {
	RelationID   string  `json:"relation_id"`
	ToContentID  string  `json:"to_content_id"`
	RelationType string  `json:"relation_type"`
	Description  string  `json:"description,omitempty"`
	Strength     float64 `json:"strength,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// HandleOperation handles all retrieve operations
func (h *Handler) HandleOperation(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error) {
	switch operation {
	case string(tools.OpSearch):
		return h.handleSearch(ctx, params)
	case string(tools.OpGetContent):
		return h.handleGetContent(ctx, params)
	case string(tools.OpFindSimilar):
		return h.handleFindSimilar(ctx, params)
	case string(tools.OpGetThreads):
		return h.handleGetThreads(ctx, params)
	case string(tools.OpGetRelationships):
		return h.handleGetRelationships(ctx, params)
	case string(tools.OpGetHistory):
		return h.handleGetHistory(ctx, params)
	default:
		return nil, fmt.Errorf("unknown retrieve operation: %s", operation)
	}
}

// handleSearch performs semantic search across content
func (h *Handler) handleSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	start := time.Now()
	
	// Parse request
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}
	
	var req SearchRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse search request: %w", err)
	}
	
	// Validate parameters - search allows project scope (read-only)
	if err := h.validator.ValidateOperation(string(tools.OpSearch), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}
	
	if req.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	
	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.MinRelevanceScore <= 0 {
		req.MinRelevanceScore = 0.3
	}
	
	// Determine access level
	accessLevel := h.sessionManager.GetAccessLevel(req.ProjectID, req.SessionID)
	
	// Update session access if session provided
	if !req.SessionID.IsEmpty() {
		if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
			return nil, fmt.Errorf("failed to update session access: %w", err)
		}
	}
	
	// TODO: Implement actual search logic based on access level
	// For now, return mock results
	mockResults := []ContentResult{
		{
			ContentID: "content_1",
			Type:      pkgTypes.ChunkTypeSolution,
			Content:   "Mock search result for: " + req.Query,
			Summary:   "This is a mock search result",
			Tags:      []string{"mock", "search"},
			CreatedAt: time.Now().Add(-1 * time.Hour),
			Score:     0.95,
		},
	}
	
	response := &SearchResponse{
		Results:   mockResults,
		Total:     len(mockResults),
		QueryTime: time.Since(start),
		HasMore:   false,
	}
	
	return response, nil
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
	if err := h.validator.ValidateOperation(string(tools.OpGetContent), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}
	
	if req.ContentID == "" {
		return nil, fmt.Errorf("content_id is required")
	}
	
	// Update session access if session provided
	if !req.SessionID.IsEmpty() {
		if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
			return nil, fmt.Errorf("failed to update session access: %w", err)
		}
	}
	
	// TODO: Implement actual content retrieval
	mockResult := &ContentResult{
		ContentID: req.ContentID,
		Type:      pkgTypes.ChunkTypeCodeChange,
		Content:   "Mock content for ID: " + req.ContentID,
		Summary:   "This is mock content",
		Tags:      []string{"mock", "content"},
		CreatedAt: time.Now().Add(-2 * time.Hour),
		Score:     1.0,
	}
	
	if req.IncludeRelations {
		mockResult.Relations = []Relation{
			{
				RelationID:   "rel_1",
				ToContentID:  "content_2",
				RelationType: "references",
				Description:  "Related to implementation",
				Strength:     0.8,
				CreatedAt:    time.Now().Add(-1 * time.Hour),
			},
		}
	}
	
	return mockResult, nil
}

// handleFindSimilar finds content similar to provided text
func (h *Handler) handleFindSimilar(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}
	
	var req FindSimilarRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse find similar request: %w", err)
	}
	
	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpFindSimilar), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}
	
	if req.Content == "" {
		return nil, fmt.Errorf("content is required for similarity search")
	}
	
	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.MinSimilarity <= 0 {
		req.MinSimilarity = 0.5
	}
	
	// Update session access if session provided
	if !req.SessionID.IsEmpty() {
		if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
			return nil, fmt.Errorf("failed to update session access: %w", err)
		}
	}
	
	// TODO: Implement actual similarity search
	mockResults := []ContentResult{
		{
			ContentID: "similar_1",
			Type:      pkgTypes.ChunkTypeProblem,
			Content:   "Similar content to: " + req.Content[:min(50, len(req.Content))] + "...",
			Summary:   "Similar content found",
			Tags:      []string{"similar", "mock"},
			CreatedAt: time.Now().Add(-3 * time.Hour),
			Score:     0.87,
		},
	}
	
	response := &SearchResponse{
		Results:   mockResults,
		Total:     len(mockResults),
		QueryTime: 50 * time.Millisecond,
		HasMore:   false,
	}
	
	return response, nil
}

// handleGetThreads retrieves conversation threads
func (h *Handler) handleGetThreads(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// TODO: Implement thread retrieval
	return map[string]interface{}{
		"threads": []map[string]interface{}{
			{
				"thread_id":    "thread_1",
				"title":        "Mock Thread",
				"description":  "A mock conversation thread",
				"content_count": 5,
				"created_at":   time.Now().Add(-24 * time.Hour),
				"last_updated": time.Now().Add(-1 * time.Hour),
			},
		},
		"total": 1,
	}, nil
}

// handleGetRelationships retrieves relationships between content
func (h *Handler) handleGetRelationships(ctx context.Context, params map[string]interface{}) (interface{}, error) {
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
	// TODO: Implement history retrieval
	return map[string]interface{}{
		"history": []map[string]interface{}{
			{
				"content_id": "content_1",
				"version":    2,
				"changed_at": time.Now().Add(-1 * time.Hour),
				"changes":    "Updated content and tags",
			},
		},
		"total": 1,
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
					"type": "string",
					"enum": tools.GetOperationsForTool(tools.ToolMemoryRetrieve),
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
			"required": ["operation", "project_id"],
		},
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}