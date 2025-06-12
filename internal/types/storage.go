// Package types provides extended type definitions for the refactored storage layer
package types

import (
	"time"
)

// Content represents a piece of stored content with proper project isolation
type Content struct {
	ID          string                 `json:"id"`
	ProjectID   ProjectID              `json:"project_id"`
	SessionID   SessionID              `json:"session_id,omitempty"`
	Type        string                 `json:"type"`        // "memory", "task", "decision", "insight"
	Title       string                 `json:"title,omitempty"`
	Content     string                 `json:"content"`
	Summary     string                 `json:"summary,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	
	// Vector embeddings
	Embeddings  []float64              `json:"embeddings,omitempty"`
	
	// Timestamps
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	AccessedAt  *time.Time             `json:"accessed_at,omitempty"`
	
	// Quality and confidence
	Quality     float64                `json:"quality,omitempty"`      // 0.0-1.0
	Confidence  float64                `json:"confidence,omitempty"`   // 0.0-1.0
	
	// Relationships
	ParentID    string                 `json:"parent_id,omitempty"`
	ThreadID    string                 `json:"thread_id,omitempty"`
	
	// Source information
	Source      string                 `json:"source,omitempty"`       // "conversation", "file", "api"
	SourcePath  string                 `json:"source_path,omitempty"`
	Version     int                    `json:"version"`
}

// ContentVersion represents a version of content for history tracking
type ContentVersion struct {
	ID          string                 `json:"id"`
	ContentID   string                 `json:"content_id"`
	Version     int                    `json:"version"`
	Content     string                 `json:"content"`
	Summary     string                 `json:"summary,omitempty"`
	Changes     string                 `json:"changes,omitempty"`
	ChangedBy   string                 `json:"changed_by,omitempty"`
	ChangedAt   time.Time              `json:"changed_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SearchQuery represents a search request with proper scoping
type SearchQuery struct {
	ProjectID    ProjectID   `json:"project_id"`
	SessionID    SessionID   `json:"session_id,omitempty"`
	Query        string      `json:"query"`
	Types        []string    `json:"types,omitempty"`
	Tags         []string    `json:"tags,omitempty"`
	Filters      *Filters    `json:"filters,omitempty"`
	Limit        int         `json:"limit,omitempty"`
	Offset       int         `json:"offset,omitempty"`
	MinRelevance float64     `json:"min_relevance,omitempty"`
	SortBy       string      `json:"sort_by,omitempty"`     // "relevance", "created_at", "updated_at"
	SortOrder    string      `json:"sort_order,omitempty"`  // "asc", "desc"
}

// SearchResults represents search results with relevance scoring
type SearchResults struct {
	Results     []*SearchResult `json:"results"`
	Total       int             `json:"total"`
	Page        int             `json:"page,omitempty"`
	PerPage     int             `json:"per_page,omitempty"`
	Query       string          `json:"query"`
	Duration    time.Duration   `json:"duration"`
	MaxRelevance float64        `json:"max_relevance,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Content     *Content  `json:"content"`
	Relevance   float64   `json:"relevance"`
	Highlights  []string  `json:"highlights,omitempty"`
	Context     string    `json:"context,omitempty"`
	Explanation string    `json:"explanation,omitempty"`
}

// Filters represents content filtering options
type Filters struct {
	Types         []string               `json:"types,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	CreatedAfter  *time.Time             `json:"created_after,omitempty"`
	CreatedBefore *time.Time             `json:"created_before,omitempty"`
	UpdatedAfter  *time.Time             `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time             `json:"updated_before,omitempty"`
	MinQuality    *float64               `json:"min_quality,omitempty"`
	MinConfidence *float64               `json:"min_confidence,omitempty"`
	HasParent     *bool                  `json:"has_parent,omitempty"`
	ThreadID      string                 `json:"thread_id,omitempty"`
	Source        string                 `json:"source,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Pattern represents a detected pattern in content
type Pattern struct {
	ID          string                 `json:"id"`
	ProjectID   ProjectID              `json:"project_id"`
	Type        string                 `json:"type"`        // "code", "decision", "issue", "solution"
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"`
	Frequency   int                    `json:"frequency"`
	Examples    []string               `json:"examples"`     // Content IDs
	Keywords    []string               `json:"keywords,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
	Trend       string                 `json:"trend"`       // "increasing", "decreasing", "stable"
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PatternFilters represents filters for pattern queries
type PatternFilters struct {
	Types         []string   `json:"types,omitempty"`
	MinConfidence *float64   `json:"min_confidence,omitempty"`
	MinFrequency  *int       `json:"min_frequency,omitempty"`
	Trends        []string   `json:"trends,omitempty"`
	DateRange     *DateRange `json:"date_range,omitempty"`
	Keywords      []string   `json:"keywords,omitempty"`
	Limit         int        `json:"limit,omitempty"`
}

// DateRange represents a date range filter
type DateRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// Insight represents a generated insight from analysis
type Insight struct {
	ID          string                 `json:"id"`
	ProjectID   ProjectID              `json:"project_id"`
	Type        string                 `json:"type"`        // "trend", "opportunity", "risk", "recommendation"
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"`
	Impact      string                 `json:"impact"`      // "low", "medium", "high"
	Evidence    []string               `json:"evidence"`    // Supporting content IDs
	Actions     []string               `json:"actions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	GeneratedAt time.Time              `json:"generated_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	AppliedAt   *time.Time             `json:"applied_at,omitempty"`
	Status      string                 `json:"status"`      // "active", "applied", "dismissed", "expired"
}

// InsightFilters represents filters for insight queries
type InsightFilters struct {
	Types         []string   `json:"types,omitempty"`
	Impact        []string   `json:"impact,omitempty"`
	Status        []string   `json:"status,omitempty"`
	MinConfidence *float64   `json:"min_confidence,omitempty"`
	DateRange     *DateRange `json:"date_range,omitempty"`
	Limit         int        `json:"limit,omitempty"`
}

// Conflict represents a detected conflict between content
type Conflict struct {
	ID          string    `json:"id"`
	ProjectID   ProjectID `json:"project_id"`
	Type        string    `json:"type"`         // "decision", "solution", "requirement"
	Severity    string    `json:"severity"`     // "low", "medium", "high", "critical"
	Description string    `json:"description"`
	ContentIDs  []string  `json:"content_ids"`  // Conflicting content
	Suggestions []string  `json:"suggestions,omitempty"`
	Confidence  float64   `json:"confidence"`
	DetectedAt  time.Time `json:"detected_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	Resolution  string    `json:"resolution,omitempty"`
	Status      string    `json:"status"`       // "active", "resolved", "ignored"
}

// ConflictFilters represents filters for conflict queries
type ConflictFilters struct {
	Types         []string   `json:"types,omitempty"`
	Severity      []string   `json:"severity,omitempty"`
	Status        []string   `json:"status,omitempty"`
	MinConfidence *float64   `json:"min_confidence,omitempty"`
	DateRange     *DateRange `json:"date_range,omitempty"`
	Limit         int        `json:"limit,omitempty"`
}

// QualityAnalysis represents content quality analysis results
type QualityAnalysis struct {
	ID              string          `json:"id"`
	ProjectID       ProjectID       `json:"project_id"`
	ContentID       string          `json:"content_id"`
	OverallScore    float64         `json:"overall_score"`    // 0.0-1.0
	Completeness    float64         `json:"completeness"`     // 0.0-1.0
	Clarity         float64         `json:"clarity"`          // 0.0-1.0
	Relevance       float64         `json:"relevance"`        // 0.0-1.0
	Recency         float64         `json:"recency"`          // 0.0-1.0
	Issues          []QualityIssue  `json:"issues,omitempty"`
	Recommendations []string        `json:"recommendations,omitempty"`
	Metrics         map[string]float64 `json:"metrics,omitempty"`
	AnalyzedAt      time.Time       `json:"analyzed_at"`
	ValidUntil      *time.Time      `json:"valid_until,omitempty"`
}

// QualityIssue represents a specific quality issue
type QualityIssue struct {
	Type        string  `json:"type"`        // "incomplete", "unclear", "outdated", "inconsistent"
	Severity    string  `json:"severity"`    // "low", "medium", "high", "critical"
	Description string  `json:"description"`
	Suggestion  string  `json:"suggestion,omitempty"`
	Line        *int    `json:"line,omitempty"`
	Column      *int    `json:"column,omitempty"`
	Context     string  `json:"context,omitempty"`
}

// Relationship represents a relationship between content items
type Relationship struct {
	ID           string                 `json:"id"`
	ProjectID    ProjectID              `json:"project_id"`
	SourceID     string                 `json:"source_id"`
	TargetID     string                 `json:"target_id"`
	Type         string                 `json:"type"`         // "references", "depends_on", "similar_to", "contradicts"
	Confidence   float64                `json:"confidence"`
	Weight       float64                `json:"weight,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CreatedBy    string                 `json:"created_by,omitempty"`  // "system", "user", "ai"
	ValidatedBy  string                 `json:"validated_by,omitempty"`
	ValidatedAt  *time.Time             `json:"validated_at,omitempty"`
}

// RelatedContent represents content found through relationship traversal
type RelatedContent struct {
	Content      *Content     `json:"content"`
	Relationship *Relationship `json:"relationship"`
	Path         []string     `json:"path"`        // IDs of content in the path
	Distance     int          `json:"distance"`    // Degrees of separation
	Relevance    float64      `json:"relevance"`
}

// Session represents a user session with project access
type Session struct {
	ID           SessionID              `json:"id"`
	ProjectID    ProjectID              `json:"project_id"`
	UserID       string                 `json:"user_id,omitempty"`
	AccessLevel  string                 `json:"access_level"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	LastAccessed time.Time              `json:"last_accessed"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
	IsActive     bool                   `json:"is_active"`
}

// SessionFilters represents filters for session queries
type SessionFilters struct {
	UserIDs      []string   `json:"user_ids,omitempty"`
	AccessLevels []string   `json:"access_levels,omitempty"`
	IsActive     *bool      `json:"is_active,omitempty"`
	CreatedAfter *time.Time `json:"created_after,omitempty"`
	AccessedAfter *time.Time `json:"accessed_after,omitempty"`
	Limit        int        `json:"limit,omitempty"`
}

// SessionStats represents session statistics for a project
type SessionStats struct {
	TotalSessions   int                    `json:"total_sessions"`
	ActiveSessions  int                    `json:"active_sessions"`
	UserCounts      map[string]int         `json:"user_counts"`
	AccessLevels    map[string]int         `json:"access_levels"`
	AverageLifetime time.Duration          `json:"average_lifetime"`
	LastActivity    *time.Time             `json:"last_activity,omitempty"`
}

// HealthStatus represents the health status of storage systems
type HealthStatus struct {
	Status       string                 `json:"status"`       // "healthy", "degraded", "unhealthy"
	Components   map[string]ComponentHealth `json:"components"`
	LastChecked  time.Time              `json:"last_checked"`
	ResponseTime time.Duration          `json:"response_time"`
	Version      string                 `json:"version,omitempty"`
	Uptime       time.Duration          `json:"uptime,omitempty"`
}

// ComponentHealth represents the health of a storage component
type ComponentHealth struct {
	Status       string                 `json:"status"`
	Message      string                 `json:"message,omitempty"`
	LastChecked  time.Time              `json:"last_checked"`
	ResponseTime time.Duration          `json:"response_time,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
}

// StorageStats represents overall storage statistics
type StorageStats struct {
	TotalContent    int64              `json:"total_content"`
	ContentByType   map[string]int64   `json:"content_by_type"`
	ContentByProject map[string]int64   `json:"content_by_project"`
	TotalSessions   int64              `json:"total_sessions"`
	ActiveSessions  int64              `json:"active_sessions"`
	StorageSize     int64              `json:"storage_size_bytes"`
	IndexSize       int64              `json:"index_size_bytes"`
	LastUpdated     time.Time          `json:"last_updated"`
	Performance     PerformanceMetrics `json:"performance"`
}

// ProjectStats represents statistics for a specific project
type ProjectStats struct {
	ProjectID       ProjectID          `json:"project_id"`
	TotalContent    int64              `json:"total_content"`
	ContentByType   map[string]int64   `json:"content_by_type"`
	TotalSessions   int64              `json:"total_sessions"`
	ActiveSessions  int64              `json:"active_sessions"`
	TotalPatterns   int64              `json:"total_patterns"`
	TotalInsights   int64              `json:"total_insights"`
	TotalConflicts  int64              `json:"total_conflicts"`
	StorageSize     int64              `json:"storage_size_bytes"`
	CreatedAt       time.Time          `json:"created_at"`
	LastActivity    time.Time          `json:"last_activity"`
	QualityScore    float64            `json:"quality_score,omitempty"`
}

// PerformanceMetrics represents performance statistics
type PerformanceMetrics struct {
	RequestsTotal    int64         `json:"requests_total"`
	RequestsPerSec   float64       `json:"requests_per_sec"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	ErrorRate        float64       `json:"error_rate"`
	CacheHitRate     float64       `json:"cache_hit_rate,omitempty"`
	QueueDepth       int           `json:"queue_depth,omitempty"`
}

// Export/Import types
type ExportOptions struct {
	Format      string    `json:"format"`      // "json", "yaml", "archive"
	IncludeData bool      `json:"include_data"`
	DateRange   *DateRange `json:"date_range,omitempty"`
	Types       []string  `json:"types,omitempty"`
	Compress    bool      `json:"compress,omitempty"`
}

type ExportResult struct {
	Success     bool      `json:"success"`
	Message     string    `json:"message"`
	Format      string    `json:"format"`
	Size        int64     `json:"size,omitempty"`
	ItemCount   int       `json:"item_count,omitempty"`
	ExportedAt  time.Time `json:"exported_at"`
	DownloadURL string    `json:"download_url,omitempty"`
	Data        string    `json:"data,omitempty"`
}

type ImportOptions struct {
	Validate      bool                   `json:"validate,omitempty"`
	Overwrite     bool                   `json:"overwrite,omitempty"`
	DryRun        bool                   `json:"dry_run,omitempty"`
	SkipDuplicates bool                   `json:"skip_duplicates,omitempty"`
	Mapping       map[string]interface{} `json:"mapping,omitempty"`
}

type ImportResult struct {
	Success       bool      `json:"success"`
	Message       string    `json:"message"`
	ItemsImported int       `json:"items_imported"`
	ItemsSkipped  int       `json:"items_skipped"`
	ItemsError    int       `json:"items_error"`
	Errors        []string  `json:"errors,omitempty"`
	ImportedAt    time.Time `json:"imported_at"`
	DryRun        bool      `json:"dry_run,omitempty"`
}

// IntegrityReport represents data integrity validation results
type IntegrityReport struct {
	ProjectID        ProjectID `json:"project_id"`
	Status           string    `json:"status"`        // "healthy", "issues", "corrupted"
	CheckedAt        time.Time `json:"checked_at"`
	TotalItems       int64     `json:"total_items"`
	ValidItems       int64     `json:"valid_items"`
	CorruptedItems   int64     `json:"corrupted_items"`
	OrphanedItems    int64     `json:"orphaned_items"`
	MissingEmbeddings int64     `json:"missing_embeddings"`
	Issues           []IntegrityIssue `json:"issues,omitempty"`
	Recommendations  []string  `json:"recommendations,omitempty"`
}

// IntegrityIssue represents a specific integrity issue
type IntegrityIssue struct {
	Type        string `json:"type"`        // "corruption", "orphan", "missing_ref", "invalid_data"
	Severity    string `json:"severity"`    // "low", "medium", "high", "critical"
	Description string `json:"description"`
	ContentID   string `json:"content_id,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// CleanupResult represents the result of cleanup operations
type CleanupResult struct {
	Success        bool      `json:"success"`
	Message        string    `json:"message"`
	ItemsDeleted   int       `json:"items_deleted"`
	BytesFreed     int64     `json:"bytes_freed"`
	RetentionDays  int       `json:"retention_days"`
	CleanedAt      time.Time `json:"cleaned_at"`
	NextCleanup    *time.Time `json:"next_cleanup,omitempty"`
}