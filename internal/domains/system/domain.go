// Package system provides the System Domain implementation
// for administrative operations, health monitoring, and system management.
package system

import (
	"context"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/domains"
	"lerian-mcp-memory/internal/types"
)

// Domain implements the SystemDomain interface
// This is the pure system domain for administrative operations
type Domain struct {
	healthMonitor    HealthMonitor
	dataManager      DataManager
	sessionManager   SessionManager
	citationService  CitationService
	config          *Config
}

// HealthMonitor handles system health and monitoring
type HealthMonitor interface {
	GetSystemHealth(ctx context.Context) (*types.HealthStatus, error)
	GetSystemMetrics(ctx context.Context, metricTypes []string, timeRange *domains.TimeRange) (map[string]interface{}, error)
	CheckComponentHealth(ctx context.Context, components []string) (map[string]interface{}, error)
}

// DataManager handles data import/export and integrity operations
type DataManager interface {
	ExportProject(ctx context.Context, projectID types.ProjectID, format string, options map[string]interface{}) (*types.ExportResult, error)
	ImportProject(ctx context.Context, projectID types.ProjectID, source, format, data string, options map[string]interface{}) (*types.ImportResult, error)
	ValidateIntegrity(ctx context.Context, projectID types.ProjectID, scope string) (*types.IntegrityReport, error)
	BackupProject(ctx context.Context, projectID types.ProjectID) (*types.BackupResult, error)
	RestoreProject(ctx context.Context, projectID types.ProjectID, backupID string) (*types.RestoreResult, error)
}

// SessionManager handles session lifecycle and access control
type SessionManager interface {
	CreateSession(ctx context.Context, projectID types.ProjectID, metadata map[string]interface{}) (*types.Session, error)
	GetSession(ctx context.Context, sessionID types.SessionID) (*types.Session, error)
	UpdateSessionAccess(ctx context.Context, sessionID types.SessionID, accessLevel string, metadata map[string]interface{}) (*types.Session, error)
	DeleteSession(ctx context.Context, sessionID types.SessionID) error
	CleanupExpiredSessions(ctx context.Context) error
}

// CitationService handles citation generation and formatting
type CitationService interface {
	GenerateCitation(ctx context.Context, projectID types.ProjectID, contentID string, style string) (string, string, error)
	FormatCitation(ctx context.Context, citation, style string) (string, error)
	GetSupportedStyles(ctx context.Context) ([]string, error)
	ValidateCitation(ctx context.Context, citation string) error
}

// Config represents configuration for the system domain
type Config struct {
	HealthCheckInterval   time.Duration `json:"health_check_interval"`
	MetricsRetention      time.Duration `json:"metrics_retention"`
	SessionTimeout        time.Duration `json:"session_timeout"`
	ExportTimeout         time.Duration `json:"export_timeout"`
	ImportTimeout         time.Duration `json:"import_timeout"`
	BackupRetention       time.Duration `json:"backup_retention"`
	MaxExportSize         int64         `json:"max_export_size"`
	MaxImportSize         int64         `json:"max_import_size"`
	CitationCacheEnabled  bool          `json:"citation_cache_enabled"`
	CitationCacheTTL      time.Duration `json:"citation_cache_ttl"`
	AutoBackupEnabled     bool          `json:"auto_backup_enabled"`
	AutoBackupInterval    time.Duration `json:"auto_backup_interval"`
}

// DefaultConfig returns default configuration for system domain
func DefaultConfig() *Config {
	return &Config{
		HealthCheckInterval:   30 * time.Second,
		MetricsRetention:      7 * 24 * time.Hour, // 7 days
		SessionTimeout:        24 * time.Hour,
		ExportTimeout:         5 * time.Minute,
		ImportTimeout:         10 * time.Minute,
		BackupRetention:       30 * 24 * time.Hour, // 30 days
		MaxExportSize:         100 * 1024 * 1024,   // 100MB
		MaxImportSize:         100 * 1024 * 1024,   // 100MB
		CitationCacheEnabled:  true,
		CitationCacheTTL:      1 * time.Hour,
		AutoBackupEnabled:     false,
		AutoBackupInterval:    24 * time.Hour,
	}
}

// NewDomain creates a new system domain instance
func NewDomain(
	healthMonitor HealthMonitor,
	dataManager DataManager,
	sessionManager SessionManager,
	citationService CitationService,
	config *Config,
) *Domain {
	if config == nil {
		config = DefaultConfig()
	}
	
	return &Domain{
		healthMonitor:   healthMonitor,
		dataManager:     dataManager,
		sessionManager:  sessionManager,
		citationService: citationService,
		config:          config,
	}
}

// Health and Monitoring Operations

// GetSystemHealth retrieves overall system health status
func (d *Domain) GetSystemHealth(ctx context.Context, req *domains.GetSystemHealthRequest) (*domains.GetSystemHealthResponse, error) {
	startTime := time.Now()
	
	// Set context timeout
	ctx, cancel := context.WithTimeout(ctx, d.config.HealthCheckInterval)
	defer cancel()
	
	// Get system health
	health, err := d.healthMonitor.GetSystemHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system health: %w", err)
	}
	
	// Get detailed component health if requested
	if req.Detailed && len(req.Components) > 0 {
		componentHealth, err := d.healthMonitor.CheckComponentHealth(ctx, req.Components)
		if err == nil {
			// Add component details to health status
			if health.Details == nil {
				health.Details = make(map[string]interface{})
			}
			health.Details["components"] = componentHealth
		}
	}
	
	return &domains.GetSystemHealthResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Health: health,
	}, nil
}

// GetSystemMetrics retrieves system performance metrics
func (d *Domain) GetSystemMetrics(ctx context.Context, req *domains.GetSystemMetricsRequest) (*domains.GetSystemMetricsResponse, error) {
	startTime := time.Now()
	
	// Get metrics
	metrics, err := d.healthMonitor.GetSystemMetrics(ctx, req.MetricTypes, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get system metrics: %w", err)
	}
	
	return &domains.GetSystemMetricsResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Metrics: metrics,
	}, nil
}

// Data Management Operations

// ExportProject exports project data in specified format
func (d *Domain) ExportProject(ctx context.Context, req *domains.ExportProjectRequest) (*domains.ExportProjectResponse, error) {
	startTime := time.Now()
	
	// Set context timeout
	ctx, cancel := context.WithTimeout(ctx, d.config.ExportTimeout)
	defer cancel()
	
	// Validate format
	if req.Format == "" {
		req.Format = "json" // Default format
	}
	
	// Export project data
	exportResult, err := d.dataManager.ExportProject(ctx, req.ProjectID, req.Format, req.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to export project: %w", err)
	}
	
	return &domains.ExportProjectResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Message:   "Project exported successfully",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Export: exportResult,
	}, nil
}

// ImportProject imports project data from specified source
func (d *Domain) ImportProject(ctx context.Context, req *domains.ImportProjectRequest) (*domains.ImportProjectResponse, error) {
	startTime := time.Now()
	
	// Set context timeout
	ctx, cancel := context.WithTimeout(ctx, d.config.ImportTimeout)
	defer cancel()
	
	// Validate import size
	if len(req.Data) > int(d.config.MaxImportSize) {
		return nil, fmt.Errorf("import data size exceeds maximum allowed size")
	}
	
	// Import project data
	importResult, err := d.dataManager.ImportProject(ctx, req.ProjectID, req.Source, req.Format, req.Data, req.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to import project: %w", err)
	}
	
	return &domains.ImportProjectResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Message:   "Project imported successfully",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Import: importResult,
	}, nil
}

// ValidateIntegrity validates data integrity for the specified scope
func (d *Domain) ValidateIntegrity(ctx context.Context, req *domains.ValidateIntegrityRequest) (*domains.ValidateIntegrityResponse, error) {
	startTime := time.Now()
	
	// Default scope to project if not specified
	scope := req.Scope
	if scope == "" {
		scope = "project"
	}
	
	// Validate integrity
	report, err := d.dataManager.ValidateIntegrity(ctx, req.ProjectID, scope)
	if err != nil {
		return nil, fmt.Errorf("failed to validate integrity: %w", err)
	}
	
	return &domains.ValidateIntegrityResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Message:   "Integrity validation completed",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Report: report,
	}, nil
}

// Session Management Operations

// CreateSession creates a new user session
func (d *Domain) CreateSession(ctx context.Context, req *domains.CreateSessionRequest) (*domains.CreateSessionResponse, error) {
	startTime := time.Now()
	
	// Create session
	session, err := d.sessionManager.CreateSession(ctx, req.ProjectID, req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	
	return &domains.CreateSessionResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Message:   "Session created successfully",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Session: session,
	}, nil
}

// GetSession retrieves session information
func (d *Domain) GetSession(ctx context.Context, req *domains.GetSessionRequest) (*domains.GetSessionResponse, error) {
	startTime := time.Now()
	
	// Get session
	session, err := d.sessionManager.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	
	return &domains.GetSessionResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Session: session,
	}, nil
}

// UpdateSessionAccess updates session access level and metadata
func (d *Domain) UpdateSessionAccess(ctx context.Context, req *domains.UpdateSessionAccessRequest) (*domains.UpdateSessionAccessResponse, error) {
	startTime := time.Now()
	
	// Update session access
	session, err := d.sessionManager.UpdateSessionAccess(ctx, req.SessionID, req.AccessLevel, req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to update session access: %w", err)
	}
	
	return &domains.UpdateSessionAccessResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Message:   "Session access updated successfully",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Session: session,
	}, nil
}

// Citation Operations

// GenerateCitation generates a citation for specified content
func (d *Domain) GenerateCitation(ctx context.Context, req *domains.GenerateCitationRequest) (*domains.GenerateCitationResponse, error) {
	startTime := time.Now()
	
	// Default style
	style := req.Style
	if style == "" {
		style = "apa" // Default to APA style
	}
	
	// Generate citation
	citation, bibEntry, err := d.citationService.GenerateCitation(ctx, req.ProjectID, req.ContentID, style)
	if err != nil {
		return nil, fmt.Errorf("failed to generate citation: %w", err)
	}
	
	return &domains.GenerateCitationResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Message:   "Citation generated successfully",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Citation: citation,
		BibEntry: bibEntry,
	}, nil
}

// FormatCitation formats a citation in the specified style
func (d *Domain) FormatCitation(ctx context.Context, req *domains.FormatCitationRequest) (*domains.FormatCitationResponse, error) {
	startTime := time.Now()
	
	// Format citation
	formattedCitation, err := d.citationService.FormatCitation(ctx, req.Citation, req.Style)
	if err != nil {
		return nil, fmt.Errorf("failed to format citation: %w", err)
	}
	
	return &domains.FormatCitationResponse{
		BaseResponse: domains.BaseResponse{
			Success:   true,
			Message:   "Citation formatted successfully",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		FormattedCitation: formattedCitation,
	}, nil
}

// Helper methods

// validateSystemRequest validates common system domain request fields
func (d *Domain) validateSystemRequest(req domains.BaseRequest) error {
	if req.ProjectID == "" {
		return fmt.Errorf("project_id is required for system operations")
	}
	
	return nil
}

// generateSystemID generates a unique ID for system operations
func generateSystemID() string {
	return fmt.Sprintf("sys_%d", time.Now().UnixNano())
}