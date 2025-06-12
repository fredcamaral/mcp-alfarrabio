// Package domains provides clean domain interfaces and separation
// for the refactored MCP Memory Server architecture.
//
// This package establishes clear boundaries between different domains:
// - Memory Domain: Content storage, search, and knowledge management
// - Task Domain: Task management, workflows, and productivity features
// - System Domain: Administrative operations and system management
package domains

import (
	"context"
	"time"

	"lerian-mcp-memory/internal/types"
)

// MemoryDomain handles all memory and knowledge management operations
// This domain is responsible for content storage, search, relationships, and intelligence
type MemoryDomain interface {
	// Content Management
	StoreContent(ctx context.Context, req *StoreContentRequest) (*StoreContentResponse, error)
	UpdateContent(ctx context.Context, req *UpdateContentRequest) (*UpdateContentResponse, error)
	DeleteContent(ctx context.Context, req *DeleteContentRequest) error
	GetContent(ctx context.Context, req *GetContentRequest) (*GetContentResponse, error)
	
	// Search and Discovery
	SearchContent(ctx context.Context, req *SearchContentRequest) (*SearchContentResponse, error)
	FindSimilarContent(ctx context.Context, req *FindSimilarRequest) (*FindSimilarResponse, error)
	FindRelatedContent(ctx context.Context, req *FindRelatedRequest) (*FindRelatedResponse, error)
	
	// Relationships
	CreateRelationship(ctx context.Context, req *CreateRelationshipRequest) (*CreateRelationshipResponse, error)
	GetRelationships(ctx context.Context, req *GetRelationshipsRequest) (*GetRelationshipsResponse, error)
	DeleteRelationship(ctx context.Context, req *DeleteRelationshipRequest) error
	
	// Intelligence and Analysis
	DetectPatterns(ctx context.Context, req *DetectPatternsRequest) (*DetectPatternsResponse, error)
	GenerateInsights(ctx context.Context, req *GenerateInsightsRequest) (*GenerateInsightsResponse, error)
	AnalyzeQuality(ctx context.Context, req *AnalyzeQualityRequest) (*AnalyzeQualityResponse, error)
	DetectConflicts(ctx context.Context, req *DetectConflictsRequest) (*DetectConflictsResponse, error)
}

// TaskDomain handles all task management and productivity operations
// This domain is responsible for task lifecycle, workflows, and project management
type TaskDomain interface {
	// Task Management
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error)
	UpdateTask(ctx context.Context, req *UpdateTaskRequest) (*UpdateTaskResponse, error)
	DeleteTask(ctx context.Context, req *DeleteTaskRequest) error
	GetTask(ctx context.Context, req *GetTaskRequest) (*GetTaskResponse, error)
	ListTasks(ctx context.Context, req *ListTasksRequest) (*ListTasksResponse, error)
	
	// Task Workflows
	TransitionTask(ctx context.Context, req *TransitionTaskRequest) (*TransitionTaskResponse, error)
	AssignTask(ctx context.Context, req *AssignTaskRequest) (*AssignTaskResponse, error)
	CompleteTask(ctx context.Context, req *CompleteTaskRequest) (*CompleteTaskResponse, error)
	
	// Task Analytics
	GetTaskMetrics(ctx context.Context, req *GetTaskMetricsRequest) (*GetTaskMetricsResponse, error)
	AnalyzeTaskPerformance(ctx context.Context, req *AnalyzeTaskPerformanceRequest) (*AnalyzeTaskPerformanceResponse, error)
	
	// Task Dependencies
	CreateDependency(ctx context.Context, req *CreateDependencyRequest) (*CreateDependencyResponse, error)
	GetDependencies(ctx context.Context, req *GetDependenciesRequest) (*GetDependenciesResponse, error)
	
	// Task Templates
	CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*CreateTemplateResponse, error)
	ApplyTemplate(ctx context.Context, req *ApplyTemplateRequest) (*ApplyTemplateResponse, error)
}

// SystemDomain handles all system administration and configuration operations
// This domain is responsible for health, metrics, exports, and system management
type SystemDomain interface {
	// Health and Monitoring
	GetSystemHealth(ctx context.Context, req *GetSystemHealthRequest) (*GetSystemHealthResponse, error)
	GetSystemMetrics(ctx context.Context, req *GetSystemMetricsRequest) (*GetSystemMetricsResponse, error)
	
	// Data Management
	ExportProject(ctx context.Context, req *ExportProjectRequest) (*ExportProjectResponse, error)
	ImportProject(ctx context.Context, req *ImportProjectRequest) (*ImportProjectResponse, error)
	ValidateIntegrity(ctx context.Context, req *ValidateIntegrityRequest) (*ValidateIntegrityResponse, error)
	
	// Session Management
	CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error)
	GetSession(ctx context.Context, req *GetSessionRequest) (*GetSessionResponse, error)
	UpdateSessionAccess(ctx context.Context, req *UpdateSessionAccessRequest) (*UpdateSessionAccessResponse, error)
	
	// Citations and References
	GenerateCitation(ctx context.Context, req *GenerateCitationRequest) (*GenerateCitationResponse, error)
	FormatCitation(ctx context.Context, req *FormatCitationRequest) (*FormatCitationResponse, error)
}

// DomainCoordinator provides cross-domain operations and orchestration
// This interface manages interactions between domains while maintaining separation
type DomainCoordinator interface {
	// Cross-domain operations
	LinkTaskToContent(ctx context.Context, req *LinkTaskToContentRequest) (*LinkTaskToContentResponse, error)
	GenerateTasksFromContent(ctx context.Context, req *GenerateTasksFromContentRequest) (*GenerateTasksFromContentResponse, error)
	CreateContentFromTask(ctx context.Context, req *CreateContentFromTaskRequest) (*CreateContentFromTaskResponse, error)
	
	// Cross-domain analytics
	AnalyzeCrossDomainPatterns(ctx context.Context, req *AnalyzeCrossDomainPatternsRequest) (*AnalyzeCrossDomainPatternsResponse, error)
	GetUnifiedMetrics(ctx context.Context, req *GetUnifiedMetricsRequest) (*GetUnifiedMetricsResponse, error)
	
	// Cross-domain search
	UnifiedSearch(ctx context.Context, req *UnifiedSearchRequest) (*UnifiedSearchResponse, error)
}

// DomainRegistry provides access to all domains and coordination
// This is the main entry point for domain operations
type DomainRegistry interface {
	Memory() MemoryDomain
	Task() TaskDomain
	System() SystemDomain
	Coordinator() DomainCoordinator
}

// Base request/response types for common fields

// BaseRequest contains common fields for all domain requests
type BaseRequest struct {
	ProjectID types.ProjectID `json:"project_id"`
	SessionID types.SessionID `json:"session_id,omitempty"`
	UserID    string          `json:"user_id,omitempty"`
	RequestID string          `json:"request_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// BaseResponse contains common fields for all domain responses
type BaseResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
}

// Memory Domain Request/Response Types

type StoreContentRequest struct {
	BaseRequest
	Content *types.Content `json:"content"`
	Options *StoreOptions  `json:"options,omitempty"`
}

type StoreOptions struct {
	GenerateEmbeddings bool                   `json:"generate_embeddings,omitempty"`
	DetectRelationships bool                   `json:"detect_relationships,omitempty"`
	ExtractMetadata    bool                   `json:"extract_metadata,omitempty"`
	CustomMetadata     map[string]interface{} `json:"custom_metadata,omitempty"`
}

type StoreContentResponse struct {
	BaseResponse
	ContentID string            `json:"content_id"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateContentRequest struct {
	BaseRequest
	ContentID string         `json:"content_id"`
	Updates   *ContentUpdates `json:"updates"`
	Options   *UpdateOptions `json:"options,omitempty"`
}

type ContentUpdates struct {
	Content  *string                `json:"content,omitempty"`
	Summary  *string                `json:"summary,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateOptions struct {
	RegenerateEmbeddings bool `json:"regenerate_embeddings,omitempty"`
	UpdateRelationships  bool `json:"update_relationships,omitempty"`
	PreservePrevious     bool `json:"preserve_previous,omitempty"`
}

type UpdateContentResponse struct {
	BaseResponse
	ContentID string    `json:"content_id"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DeleteContentRequest struct {
	BaseRequest
	ContentID string         `json:"content_id"`
	Options   *DeleteOptions `json:"options,omitempty"`
}

type DeleteOptions struct {
	Hard                bool `json:"hard,omitempty"`                // Permanent deletion
	DeleteRelationships bool `json:"delete_relationships,omitempty"` // Also delete relationships
	PreserveReferences  bool `json:"preserve_references,omitempty"`  // Keep references but mark as deleted
}

type GetContentRequest struct {
	BaseRequest
	ContentID    string       `json:"content_id"`
	IncludeRefs  bool         `json:"include_refs,omitempty"`
	IncludeHist  bool         `json:"include_history,omitempty"`
	Options      *GetOptions  `json:"options,omitempty"`
}

type GetOptions struct {
	IncludeEmbeddings   bool `json:"include_embeddings,omitempty"`
	IncludeRelationships bool `json:"include_relationships,omitempty"`
	IncludeMetadata     bool `json:"include_metadata,omitempty"`
	Format              string `json:"format,omitempty"` // "full", "summary", "minimal"
}

type GetContentResponse struct {
	BaseResponse
	Content       *types.Content          `json:"content"`
	Relationships []*types.Relationship   `json:"relationships,omitempty"`
	History       []*types.ContentVersion `json:"history,omitempty"`
	References    []string               `json:"references,omitempty"`
}

type SearchContentRequest struct {
	BaseRequest
	Query   string         `json:"query"`
	Filters *types.Filters `json:"filters,omitempty"`
	Options *SearchOptions `json:"options,omitempty"`
}

type SearchOptions struct {
	Limit           int     `json:"limit,omitempty"`
	Offset          int     `json:"offset,omitempty"`
	MinRelevance    float64 `json:"min_relevance,omitempty"`
	IncludeContext  bool    `json:"include_context,omitempty"`
	IncludeHighlights bool   `json:"include_highlights,omitempty"`
	SortBy          string  `json:"sort_by,omitempty"`
	SortOrder       string  `json:"sort_order,omitempty"`
}

type SearchContentResponse struct {
	BaseResponse
	Results    []*types.SearchResult `json:"results"`
	Total      int                   `json:"total"`
	Page       int                   `json:"page"`
	PerPage    int                   `json:"per_page"`
	Facets     map[string]interface{} `json:"facets,omitempty"`
	Duration   time.Duration          `json:"duration"`
}

type FindSimilarRequest struct {
	BaseRequest
	Content   string         `json:"content,omitempty"`
	ContentID string         `json:"content_id,omitempty"`
	Limit     int            `json:"limit,omitempty"`
	Threshold float64        `json:"threshold,omitempty"`
	Options   *SimilarOptions `json:"options,omitempty"`
}

type SimilarOptions struct {
	IncludeSelf     bool     `json:"include_self,omitempty"`
	ContentTypes    []string `json:"content_types,omitempty"`
	ExcludeIDs      []string `json:"exclude_ids,omitempty"`
	SimilarityMethod string   `json:"similarity_method,omitempty"` // "cosine", "euclidean", "manhattan"
}

type FindSimilarResponse struct {
	BaseResponse
	Similar []*SimilarContent `json:"similar"`
}

type SimilarContent struct {
	Content    *types.Content `json:"content"`
	Similarity float64        `json:"similarity"`
	Explanation string        `json:"explanation,omitempty"`
}

type FindRelatedRequest struct {
	BaseRequest
	ContentID     string   `json:"content_id"`
	RelationTypes []string `json:"relation_types,omitempty"`
	MaxDepth      int      `json:"max_depth,omitempty"`
	Limit         int      `json:"limit,omitempty"`
}

type FindRelatedResponse struct {
	BaseResponse
	Related []*types.RelatedContent `json:"related"`
}

// Task Domain Request/Response Types

type CreateTaskRequest struct {
	BaseRequest
	Task    *TaskData     `json:"task"`
	Options *TaskOptions  `json:"options,omitempty"`
}

type TaskData struct {
	Title         string                 `json:"title"`
	Description   string                 `json:"description,omitempty"`
	Priority      string                 `json:"priority,omitempty"`
	Status        string                 `json:"status,omitempty"`
	AssigneeID    string                 `json:"assignee_id,omitempty"`
	DueDate       *time.Time             `json:"due_date,omitempty"`
	EstimatedMins int                    `json:"estimated_mins,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Dependencies  []string               `json:"dependencies,omitempty"`
	LinkedContent []string               `json:"linked_content,omitempty"`
}

type TaskOptions struct {
	AutoAssign          bool `json:"auto_assign,omitempty"`
	DetectDependencies  bool `json:"detect_dependencies,omitempty"`
	CreateSubtasks      bool `json:"create_subtasks,omitempty"`
	NotifyAssignee      bool `json:"notify_assignee,omitempty"`
}

type CreateTaskResponse struct {
	BaseResponse
	TaskID      string   `json:"task_id"`
	SubtaskIDs  []string `json:"subtask_ids,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type UpdateTaskRequest struct {
	BaseRequest
	TaskID  string      `json:"task_id"`
	Updates *TaskUpdates `json:"updates"`
	Options *TaskOptions `json:"options,omitempty"`
}

type TaskUpdates struct {
	Title         *string                `json:"title,omitempty"`
	Description   *string                `json:"description,omitempty"`
	Status        *string                `json:"status,omitempty"`
	Priority      *string                `json:"priority,omitempty"`
	AssigneeID    *string                `json:"assignee_id,omitempty"`
	DueDate       *time.Time             `json:"due_date,omitempty"`
	EstimatedMins *int                   `json:"estimated_mins,omitempty"`
	ActualMins    *int                   `json:"actual_mins,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateTaskResponse struct {
	BaseResponse
	TaskID    string    `json:"task_id"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DeleteTaskRequest struct {
	BaseRequest
	TaskID  string            `json:"task_id"`
	Options *TaskDeleteOptions `json:"options,omitempty"`
}

type TaskDeleteOptions struct {
	DeleteSubtasks    bool `json:"delete_subtasks,omitempty"`
	PreserveHistory   bool `json:"preserve_history,omitempty"`
	NotifyAssignee    bool `json:"notify_assignee,omitempty"`
}

type GetTaskRequest struct {
	BaseRequest
	TaskID  string          `json:"task_id"`
	Options *TaskGetOptions `json:"options,omitempty"`
}

type TaskGetOptions struct {
	IncludeDependencies bool `json:"include_dependencies,omitempty"`
	IncludeSubtasks     bool `json:"include_subtasks,omitempty"`
	IncludeHistory      bool `json:"include_history,omitempty"`
	IncludeLinkedContent bool `json:"include_linked_content,omitempty"`
}

type GetTaskResponse struct {
	BaseResponse
	Task         interface{}     `json:"task"`
	Dependencies []interface{}   `json:"dependencies,omitempty"`
	Subtasks     []interface{}   `json:"subtasks,omitempty"`
	History      []interface{}   `json:"history,omitempty"`
	LinkedContent []*types.Content `json:"linked_content,omitempty"`
}

type ListTasksRequest struct {
	BaseRequest
	Filters *TaskFilters    `json:"filters,omitempty"`
	Options *TaskListOptions `json:"options,omitempty"`
}

type TaskFilters struct {
	Status       []string   `json:"status,omitempty"`
	Priority     []string   `json:"priority,omitempty"`
	AssigneeIDs  []string   `json:"assignee_ids,omitempty"`
	Tags         []string   `json:"tags,omitempty"`
	DueBefore    *time.Time `json:"due_before,omitempty"`
	DueAfter     *time.Time `json:"due_after,omitempty"`
	CreatedAfter *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	HasSubtasks  *bool      `json:"has_subtasks,omitempty"`
}

type TaskListOptions struct {
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
	SortBy      string `json:"sort_by,omitempty"`
	SortOrder   string `json:"sort_order,omitempty"`
	IncludeMeta bool   `json:"include_meta,omitempty"`
}

type ListTasksResponse struct {
	BaseResponse
	Tasks   []interface{} `json:"tasks"`
	Total   int           `json:"total"`
	Page    int           `json:"page"`
	PerPage int           `json:"per_page"`
}

// System Domain Request/Response Types

type GetSystemHealthRequest struct {
	BaseRequest
	Detailed    bool `json:"detailed,omitempty"`
	Components  []string `json:"components,omitempty"`
}

type GetSystemHealthResponse struct {
	BaseResponse
	Health *types.HealthStatus `json:"health"`
}

type GetSystemMetricsRequest struct {
	BaseRequest
	MetricTypes []string   `json:"metric_types,omitempty"`
	TimeRange   *TimeRange `json:"time_range,omitempty"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type GetSystemMetricsResponse struct {
	BaseResponse
	Metrics map[string]interface{} `json:"metrics"`
}

type ExportProjectRequest struct {
	BaseRequest
	Format  string                 `json:"format"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type ExportProjectResponse struct {
	BaseResponse
	Export *types.ExportResult `json:"export"`
}

type ImportProjectRequest struct {
	BaseRequest
	Source  string                 `json:"source"`
	Format  string                 `json:"format"`
	Data    string                 `json:"data,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type ImportProjectResponse struct {
	BaseResponse
	Import *types.ImportResult `json:"import"`
}

type ValidateIntegrityRequest struct {
	BaseRequest
	Scope string `json:"scope,omitempty"` // "project", "session", "system"
}

type ValidateIntegrityResponse struct {
	BaseResponse
	Report *types.IntegrityReport `json:"report"`
}

type CreateSessionRequest struct {
	BaseRequest
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type CreateSessionResponse struct {
	BaseResponse
	Session *types.Session `json:"session"`
}

type GetSessionRequest struct {
	BaseRequest
}

type GetSessionResponse struct {
	BaseResponse
	Session *types.Session `json:"session"`
}

type UpdateSessionAccessRequest struct {
	BaseRequest
	AccessLevel string                 `json:"access_level,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type UpdateSessionAccessResponse struct {
	BaseResponse
	Session *types.Session `json:"session"`
}

type GenerateCitationRequest struct {
	BaseRequest
	ContentID string `json:"content_id"`
	Style     string `json:"style,omitempty"`
}

type GenerateCitationResponse struct {
	BaseResponse
	Citation string `json:"citation"`
	BibEntry string `json:"bib_entry,omitempty"`
}

type FormatCitationRequest struct {
	BaseRequest
	Citation string `json:"citation"`
	Style    string `json:"style"`
}

type FormatCitationResponse struct {
	BaseResponse
	FormattedCitation string `json:"formatted_citation"`
}

// Cross-domain Request/Response Types

type LinkTaskToContentRequest struct {
	BaseRequest
	TaskID    string `json:"task_id"`
	ContentID string `json:"content_id"`
	LinkType  string `json:"link_type,omitempty"` // "references", "created_from", "depends_on"
}

type LinkTaskToContentResponse struct {
	BaseResponse
	LinkID string `json:"link_id"`
}

type GenerateTasksFromContentRequest struct {
	BaseRequest
	ContentID string                 `json:"content_id"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

type GenerateTasksFromContentResponse struct {
	BaseResponse
	TaskIDs []string `json:"task_ids"`
}

type CreateContentFromTaskRequest struct {
	BaseRequest
	TaskID  string                 `json:"task_id"`
	Type    string                 `json:"type"`    // "solution", "documentation", "analysis"
	Options map[string]interface{} `json:"options,omitempty"`
}

type CreateContentFromTaskResponse struct {
	BaseResponse
	ContentID string `json:"content_id"`
}

// Placeholder types for remaining requests/responses
type AnalyzeCrossDomainPatternsRequest struct{ BaseRequest }
type AnalyzeCrossDomainPatternsResponse struct{ BaseResponse }
type GetUnifiedMetricsRequest struct{ BaseRequest }
type GetUnifiedMetricsResponse struct{ BaseResponse }
type UnifiedSearchRequest struct{ BaseRequest }
type UnifiedSearchResponse struct{ BaseResponse }

// And all remaining method request/response types that were referenced but not defined
type CreateRelationshipRequest struct{ BaseRequest }
type CreateRelationshipResponse struct{ BaseResponse }
type GetRelationshipsRequest struct{ BaseRequest }
type GetRelationshipsResponse struct{ BaseResponse }
type DeleteRelationshipRequest struct{ BaseRequest }
type DetectPatternsRequest struct{ BaseRequest }
type DetectPatternsResponse struct{ BaseResponse }
type GenerateInsightsRequest struct{ BaseRequest }
type GenerateInsightsResponse struct{ BaseResponse }
type AnalyzeQualityRequest struct{ BaseRequest }
type AnalyzeQualityResponse struct{ BaseResponse }
type DetectConflictsRequest struct{ BaseRequest }
type DetectConflictsResponse struct{ BaseResponse }
type TransitionTaskRequest struct{ BaseRequest }
type TransitionTaskResponse struct{ BaseResponse }
type AssignTaskRequest struct{ BaseRequest }
type AssignTaskResponse struct{ BaseResponse }
type CompleteTaskRequest struct{ BaseRequest }
type CompleteTaskResponse struct{ BaseResponse }
type GetTaskMetricsRequest struct{ BaseRequest }
type GetTaskMetricsResponse struct{ BaseResponse }
type AnalyzeTaskPerformanceRequest struct{ BaseRequest }
type AnalyzeTaskPerformanceResponse struct{ BaseResponse }
type CreateDependencyRequest struct{ BaseRequest }
type CreateDependencyResponse struct{ BaseResponse }
type GetDependenciesRequest struct{ BaseRequest }
type GetDependenciesResponse struct{ BaseResponse }
type CreateTemplateRequest struct{ BaseRequest }
type CreateTemplateResponse struct{ BaseResponse }
type ApplyTemplateRequest struct{ BaseRequest }
type ApplyTemplateResponse struct{ BaseResponse }