// Package types provides shared type definitions for domain interfaces
// to avoid circular dependency issues between domain packages.
package types

import (
	"time"

	"lerian-mcp-memory/internal/types"
)

// BaseRequest represents the common request structure
type BaseRequest struct {
	ProjectID types.ProjectID        `json:"project_id"`
	SessionID types.SessionID        `json:"session_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Timestamp time.Time              `json:"timestamp,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// BaseResponse represents the common response structure
type BaseResponse struct {
	Success   bool          `json:"success"`
	Message   string        `json:"message,omitempty"`
	RequestID string        `json:"request_id,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
}

// StoreContentRequest represents a request to store content
type StoreContentRequest struct {
	BaseRequest
	Content    string                 `json:"content"`
	Type       string                 `json:"type"`
	Title      string                 `json:"title,omitempty"`
	Summary    string                 `json:"summary,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	ThreadID   string                 `json:"thread_id,omitempty"`
	ParentID   string                 `json:"parent_id,omitempty"`
	Source     string                 `json:"source,omitempty"`
	SourcePath string                 `json:"source_path,omitempty"`
}

// StoreContentResponse represents the response from storing content
type StoreContentResponse struct {
	BaseResponse
	ContentID string    `json:"content_id"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateContentRequest represents a request to update content
type UpdateContentRequest struct {
	BaseRequest
	ContentID  string                 `json:"content_id"`
	Content    string                 `json:"content,omitempty"`
	Title      string                 `json:"title,omitempty"`
	Summary    string                 `json:"summary,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	AddTags    []string               `json:"add_tags,omitempty"`
	RemoveTags []string               `json:"remove_tags,omitempty"`
	Quality    *float64               `json:"quality,omitempty"`
	Confidence *float64               `json:"confidence,omitempty"`
}

// UpdateContentResponse represents the response from updating content
type UpdateContentResponse struct {
	BaseResponse
	ContentID string    `json:"content_id"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeleteContentRequest represents a request to delete content
type DeleteContentRequest struct {
	BaseRequest
	ContentID string `json:"content_id"`
	Force     bool   `json:"force,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// GetContentRequest represents a request to get content
type GetContentRequest struct {
	BaseRequest
	ContentID      string   `json:"content_id"`
	IncludeHistory bool     `json:"include_history,omitempty"`
	IncludeRelated bool     `json:"include_related,omitempty"`
	Fields         []string `json:"fields,omitempty"`
}

// GetContentResponse represents the response from getting content
type GetContentResponse struct {
	BaseResponse
	Content *types.Content          `json:"content,omitempty"`
	History []*types.ContentVersion `json:"history,omitempty"`
	Related []*types.RelatedContent `json:"related,omitempty"`
}

// SearchContentRequest represents a search request
type SearchContentRequest struct {
	BaseRequest
	Query          string         `json:"query"`
	Types          []string       `json:"types,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	Filters        *types.Filters `json:"filters,omitempty"`
	Limit          int            `json:"limit,omitempty"`
	Offset         int            `json:"offset,omitempty"`
	MinRelevance   float64        `json:"min_relevance,omitempty"`
	SortBy         string         `json:"sort_by,omitempty"`
	SortOrder      string         `json:"sort_order,omitempty"`
	IncludeContext bool           `json:"include_context,omitempty"`
}

// SearchContentResponse represents the response from searching content
type SearchContentResponse struct {
	BaseResponse
	Results      []*types.SearchResult `json:"results"`
	Total        int                   `json:"total"`
	Query        string                `json:"query"`
	Duration     time.Duration         `json:"duration"`
	MaxRelevance float64               `json:"max_relevance"`
}

// FindSimilarRequest represents a request to find similar content
type FindSimilarRequest struct {
	BaseRequest
	Content        string         `json:"content,omitempty"`
	ContentID      string         `json:"content_id,omitempty"`
	Limit          int            `json:"limit,omitempty"`
	MinSimilarity  float64        `json:"min_similarity,omitempty"`
	Types          []string       `json:"types,omitempty"`
	ExcludeTypes   []string       `json:"exclude_types,omitempty"`
	IncludeContext bool           `json:"include_context,omitempty"`
	Filters        *types.Filters `json:"filters,omitempty"`
}

// SimilarContent represents a similar content item
type SimilarContent struct {
	Content    *types.Content `json:"content"`
	Similarity float64        `json:"similarity"`
	Context    string         `json:"context,omitempty"`
}

// FindSimilarResponse represents the response from finding similar content
type FindSimilarResponse struct {
	BaseResponse
	Similar  []*SimilarContent `json:"similar"`
	Query    string            `json:"query,omitempty"`
	Total    int               `json:"total"`
	Duration time.Duration     `json:"duration"`
}

// FindRelatedRequest represents a request to find related content
type FindRelatedRequest struct {
	BaseRequest
	ContentID     string   `json:"content_id"`
	MaxDepth      int      `json:"max_depth,omitempty"`
	RelationTypes []string `json:"relation_types,omitempty"`
	Limit         int      `json:"limit,omitempty"`
}

// FindRelatedResponse represents the response from finding related content
type FindRelatedResponse struct {
	BaseResponse
	Related  []*types.RelatedContent `json:"related"`
	Total    int                     `json:"total"`
	Duration time.Duration           `json:"duration"`
}

// CreateRelationshipRequest represents a request to create a relationship
type CreateRelationshipRequest struct{ BaseRequest }

// CreateRelationshipResponse represents the response from creating a relationship
type CreateRelationshipResponse struct{ BaseResponse }

// GetRelationshipsRequest represents a request to get relationships
type GetRelationshipsRequest struct{ BaseRequest }

// GetRelationshipsResponse represents the response from getting relationships
type GetRelationshipsResponse struct{ BaseResponse }

// DeleteRelationshipRequest represents a request to delete a relationship
type DeleteRelationshipRequest struct{ BaseRequest }

// DetectPatternsRequest represents a request to detect patterns
type DetectPatternsRequest struct{ BaseRequest }

// DetectPatternsResponse represents the response from detecting patterns
type DetectPatternsResponse struct{ BaseResponse }

// GenerateInsightsRequest represents a request to generate insights
type GenerateInsightsRequest struct{ BaseRequest }

// GenerateInsightsResponse represents the response from generating insights
type GenerateInsightsResponse struct{ BaseResponse }

// AnalyzeQualityRequest represents a request to analyze quality
type AnalyzeQualityRequest struct{ BaseRequest }

// AnalyzeQualityResponse represents the response from analyzing quality
type AnalyzeQualityResponse struct{ BaseResponse }

// DetectConflictsRequest represents a request to detect conflicts
type DetectConflictsRequest struct{ BaseRequest }

// DetectConflictsResponse represents the response from detecting conflicts
type DetectConflictsResponse struct{ BaseResponse }

// Cross-domain request/response types

// LinkTaskToContentRequest represents a request to link a task to content
type LinkTaskToContentRequest struct {
	BaseRequest
	TaskID    string `json:"task_id"`
	ContentID string `json:"content_id"`
	LinkType  string `json:"link_type,omitempty"`
}

// LinkTaskToContentResponse represents the response from linking task to content
type LinkTaskToContentResponse struct {
	BaseResponse
	LinkID string `json:"link_id"`
}

// GenerateTasksFromContentRequest represents a request to generate tasks from content
type GenerateTasksFromContentRequest struct {
	BaseRequest
	ContentID  string   `json:"content_id"`
	TaskTypes  []string `json:"task_types,omitempty"`
	Priority   string   `json:"priority,omitempty"`
	AutoAssign bool     `json:"auto_assign,omitempty"`
	DueDate    string   `json:"due_date,omitempty"`
	MaxTasks   int      `json:"max_tasks,omitempty"`
}

// GenerateTasksFromContentResponse represents the response from generating tasks
type GenerateTasksFromContentResponse struct {
	BaseResponse
	TaskIDs []string `json:"task_ids"`
	Count   int      `json:"count"`
}

// CreateContentFromTaskRequest represents a request to create content from a task
type CreateContentFromTaskRequest struct {
	BaseRequest
	TaskID      string `json:"task_id"`
	ContentType string `json:"content_type,omitempty"`
	Template    string `json:"template,omitempty"`
	IncludeMeta bool   `json:"include_meta,omitempty"`
}

// CreateContentFromTaskResponse represents the response from creating content from task
type CreateContentFromTaskResponse struct {
	BaseResponse
	ContentID string `json:"content_id"`
}

// AnalyzeCrossDomainPatternsRequest represents a request to analyze patterns across domains
type AnalyzeCrossDomainPatternsRequest struct {
	BaseRequest
	Domains     []string `json:"domains,omitempty"`
	TimeRange   string   `json:"time_range,omitempty"`
	PatternType string   `json:"pattern_type,omitempty"`
	MinSupport  float64  `json:"min_support,omitempty"`
}

// AnalyzeCrossDomainPatternsResponse represents the response from analyzing cross-domain patterns
type AnalyzeCrossDomainPatternsResponse struct {
	BaseResponse
	Patterns []CrossDomainPattern `json:"patterns"`
	Total    int                  `json:"total"`
}

// CrossDomainPattern represents a pattern found across multiple domains
type CrossDomainPattern struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Domains     []string               `json:"domains"`
	Support     float64                `json:"support"`
	Confidence  float64                `json:"confidence"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// GetTaskRequest represents a request to get task information
type GetTaskRequest struct {
	BaseRequest
	TaskID string `json:"task_id"`
}

// GetTaskResponse represents the response from getting task information
type GetTaskResponse struct {
	BaseResponse
	Task interface{} `json:"task,omitempty"`
}

// GetUnifiedMetricsRequest represents a request to get unified metrics across domains
type GetUnifiedMetricsRequest struct {
	BaseRequest
	Domains    []string `json:"domains,omitempty"`
	MetricType string   `json:"metric_type,omitempty"`
	TimeRange  string   `json:"time_range,omitempty"`
}

// GetUnifiedMetricsResponse represents the response from getting unified metrics
type GetUnifiedMetricsResponse struct {
	BaseResponse
	Metrics map[string]interface{} `json:"metrics"`
}

// UnifiedSearchRequest represents a unified search request across domains
type UnifiedSearchRequest struct {
	BaseRequest
	Query     string   `json:"query"`
	Domains   []string `json:"domains,omitempty"`
	Limit     int      `json:"limit,omitempty"`
	Offset    int      `json:"offset,omitempty"`
	SortBy    string   `json:"sort_by,omitempty"`
	SortOrder string   `json:"sort_order,omitempty"`
}

// UnifiedSearchResponse represents the response from unified search
type UnifiedSearchResponse struct {
	BaseResponse
	Results []UnifiedSearchResult `json:"results"`
	Total   int                   `json:"total"`
}

// UnifiedSearchResult represents a search result from unified search
type UnifiedSearchResult struct {
	Domain    string                 `json:"domain"`
	Type      string                 `json:"type"`
	ID        string                 `json:"id"`
	Title     string                 `json:"title,omitempty"`
	Content   string                 `json:"content,omitempty"`
	Relevance float64                `json:"relevance"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
