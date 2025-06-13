// Package storage provides clean storage interfaces and implementations
// for the refactored MCP Memory Server architecture.
package storage

import (
	"context"
	"time"

	"lerian-mcp-memory/internal/types"
)

// ContentStore handles all content persistence operations
// Replaces fragmented storage interfaces with clean contracts
type ContentStore interface {
	// Store content with proper project isolation
	Store(ctx context.Context, content *types.Content) error

	// Update existing content
	Update(ctx context.Context, content *types.Content) error

	// Delete content by project and content ID
	Delete(ctx context.Context, projectID types.ProjectID, contentID string) error

	// Get content by project and content ID
	Get(ctx context.Context, projectID types.ProjectID, contentID string) (*types.Content, error)

	// Batch operations for efficiency
	BatchStore(ctx context.Context, contents []*types.Content) (*BatchResult, error)
	BatchUpdate(ctx context.Context, contents []*types.Content) (*BatchResult, error)
	BatchDelete(ctx context.Context, projectID types.ProjectID, contentIDs []string) (*BatchResult, error)
}

// SearchStore handles all search and retrieval operations
// Provides clean search interface with proper scoping
type SearchStore interface {
	// Search content within project scope
	Search(ctx context.Context, query *types.SearchQuery) (*types.SearchResults, error)

	// Find similar content within project
	FindSimilar(ctx context.Context, content string, projectID types.ProjectID, sessionID types.SessionID) ([]*types.Content, error)

	// Get content by project with optional filters
	GetByProject(ctx context.Context, projectID types.ProjectID, filters *types.Filters) ([]*types.Content, error)

	// Get content by session within project
	GetBySession(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID, filters *types.Filters) ([]*types.Content, error)

	// Get content history for specific content
	GetHistory(ctx context.Context, projectID types.ProjectID, contentID string) ([]*types.ContentVersion, error)
}

// AnalysisStore handles all analysis and intelligence data
// Stores patterns, insights, and analytical results
type AnalysisStore interface {
	// Store detected patterns
	StorePattern(ctx context.Context, pattern *types.Pattern) error

	// Get patterns for project
	GetPatterns(ctx context.Context, projectID types.ProjectID, filters *types.PatternFilters) ([]*types.Pattern, error)

	// Store generated insights
	StoreInsight(ctx context.Context, insight *types.Insight) error

	// Get insights for project
	GetInsights(ctx context.Context, projectID types.ProjectID, filters *types.InsightFilters) ([]*types.Insight, error)

	// Store conflict detection results
	StoreConflict(ctx context.Context, conflict *types.Conflict) error

	// Get conflicts for project
	GetConflicts(ctx context.Context, projectID types.ProjectID, filters *types.ConflictFilters) ([]*types.Conflict, error)

	// Store quality analysis results
	StoreQualityAnalysis(ctx context.Context, analysis *types.QualityAnalysis) error

	// Get quality analysis for project
	GetQualityAnalysis(ctx context.Context, projectID types.ProjectID, contentID string) (*types.QualityAnalysis, error)
}

// RelationshipStore handles content relationships and graph operations
// Manages connections between content items
type RelationshipStore interface {
	// Store relationship between content items
	StoreRelationship(ctx context.Context, relationship *types.Relationship) error

	// Get relationships for content
	GetRelationships(ctx context.Context, projectID types.ProjectID, contentID string, relationTypes []string) ([]*types.Relationship, error)

	// Find related content through relationships
	FindRelated(ctx context.Context, projectID types.ProjectID, contentID string, maxDepth int) ([]*types.RelatedContent, error)

	// Delete relationship
	DeleteRelationship(ctx context.Context, relationshipID string) error

	// Update relationship confidence
	UpdateRelationshipConfidence(ctx context.Context, relationshipID string, confidence float64) error
}

// SessionStore handles session management and access control
// Manages session data and access levels
type SessionStore interface {
	// Create new session
	CreateSession(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID, metadata map[string]interface{}) error

	// Get session information
	GetSession(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID) (*types.Session, error)

	// Update session access time
	UpdateSessionAccess(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID) error

	// List sessions for project
	ListSessions(ctx context.Context, projectID types.ProjectID, filters *types.SessionFilters) ([]*types.Session, error)

	// Delete session
	DeleteSession(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID) error

	// Get session statistics
	GetSessionStats(ctx context.Context, projectID types.ProjectID) (*types.SessionStats, error)
}

// SystemStore handles system-level operations and metadata
// Manages system health, metrics, and administrative operations
type SystemStore interface {
	// Health check for storage systems
	HealthCheck(ctx context.Context) (*types.HealthStatus, error)

	// Get storage statistics
	GetStats(ctx context.Context) (*types.StorageStats, error)

	// Get project statistics
	GetProjectStats(ctx context.Context, projectID types.ProjectID) (*types.ProjectStats, error)

	// Export project data
	ExportProject(ctx context.Context, projectID types.ProjectID, format string, options *types.ExportOptions) (*types.ExportResult, error)

	// Import project data
	ImportProject(ctx context.Context, projectID types.ProjectID, data string, format string, options *types.ImportOptions) (*types.ImportResult, error)

	// Validate data integrity
	ValidateIntegrity(ctx context.Context, projectID types.ProjectID) (*types.IntegrityReport, error)

	// Cleanup old data based on retention policies
	Cleanup(ctx context.Context, projectID types.ProjectID, retentionDays int) (*types.CleanupResult, error)
}

// UnifiedStore combines all storage interfaces for convenience
// Provides a single interface for all storage operations
type UnifiedStore interface {
	ContentStore
	SearchStore
	AnalysisStore
	RelationshipStore
	SessionStore
	SystemStore

	// Transaction support
	WithTransaction(ctx context.Context, fn func(tx UnifiedStore) error) error

	// Connection management
	Close() error

	// Migration support
	Migrate(ctx context.Context) error
}

// BatchResult represents the result of batch operations
type BatchResult struct {
	Success      int                    `json:"success"`
	Failed       int                    `json:"failed"`
	Errors       []BatchError           `json:"errors,omitempty"`
	ProcessedIDs []string               `json:"processed_ids,omitempty"`
	Metrics      *BatchOperationMetrics `json:"metrics,omitempty"`
}

// BatchError represents an error in batch operations
type BatchError struct {
	Index int    `json:"index"`
	ID    string `json:"id,omitempty"`
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// BatchOperationMetrics represents metrics for batch operations
type BatchOperationMetrics struct {
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
	ItemsPerSec float64       `json:"items_per_sec"`
	TotalBytes  int64         `json:"total_bytes"`
	BytesPerSec float64       `json:"bytes_per_sec"`
}

// Filter types for different operations
type ProjectFilter struct {
	ProjectIDs    []types.ProjectID      `json:"project_ids,omitempty"`
	CreatedAfter  *time.Time             `json:"created_after,omitempty"`
	CreatedBefore *time.Time             `json:"created_before,omitempty"`
	UpdatedAfter  *time.Time             `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time             `json:"updated_before,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AccessLevel represents different levels of data access
type AccessLevel string

const (
	AccessLevelReadOnly AccessLevel = "read_only" // Limited project data
	AccessLevelSession  AccessLevel = "session"   // Session + project data
	AccessLevelProject  AccessLevel = "project"   // All project data
	AccessLevelGlobal   AccessLevel = "global"    // Global system access
)

// StorageConfig represents configuration for storage implementations
type StorageConfig struct {
	// Database connection settings
	DatabaseURL     string `json:"database_url"`
	MaxConnections  int    `json:"max_connections"`
	ConnMaxLifetime string `json:"conn_max_lifetime"`

	// Vector database settings
	VectorURL       string `json:"vector_url"`
	VectorDimension int    `json:"vector_dimension"`

	// Performance settings
	BatchSize     int           `json:"batch_size"`
	QueryTimeout  time.Duration `json:"query_timeout"`
	RetryAttempts int           `json:"retry_attempts"`
	RetryDelay    time.Duration `json:"retry_delay"`

	// Security settings
	EncryptionKey  string `json:"encryption_key,omitempty"`
	EnableAuditLog bool   `json:"enable_audit_log"`

	// Feature flags
	EnableCaching bool `json:"enable_caching"`
	EnableMetrics bool `json:"enable_metrics"`
	EnableTracing bool `json:"enable_tracing"`
}
