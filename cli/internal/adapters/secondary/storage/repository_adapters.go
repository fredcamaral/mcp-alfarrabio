package storage

import (
	"context"
	"fmt"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
	"lerian-mcp-memory-cli/internal/domain/repositories"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// TaskRepositoryAdapter adapts Storage interface to TaskRepository
type TaskRepositoryAdapter struct {
	storage ports.Storage
}

// NewTaskRepositoryAdapter creates a new task repository adapter
func NewTaskRepositoryAdapter(storage ports.Storage) repositories.TaskRepository {
	return &TaskRepositoryAdapter{storage: storage}
}

// FindByTimeRange finds tasks within a time range
func (t *TaskRepositoryAdapter) FindByTimeRange(ctx context.Context, repository string, startTime, endTime time.Time) ([]*entities.Task, error) {
	// Convert times to ISO string format for filters
	startStr := startTime.Format(time.RFC3339)
	endStr := endTime.Format(time.RFC3339)

	filters := &ports.TaskFilters{
		Repository:    repository,
		CreatedAfter:  &startStr,
		CreatedBefore: &endStr,
	}

	return t.storage.ListTasks(ctx, repository, filters)
}

// FindByRepository finds all tasks for a repository
func (t *TaskRepositoryAdapter) FindByRepository(ctx context.Context, repository string) ([]*entities.Task, error) {
	return t.storage.GetTasksByRepository(ctx, repository)
}

// FindByStatus finds tasks by status
func (t *TaskRepositoryAdapter) FindByStatus(ctx context.Context, repository string, status entities.Status) ([]*entities.Task, error) {
	filters := &ports.TaskFilters{
		Repository: repository,
		Status:     &status,
	}

	return t.storage.ListTasks(ctx, repository, filters)
}

// Create creates a new task
func (t *TaskRepositoryAdapter) Create(ctx context.Context, task *entities.Task) error {
	return t.storage.SaveTask(ctx, task)
}

// Update updates an existing task
func (t *TaskRepositoryAdapter) Update(ctx context.Context, task *entities.Task) error {
	return t.storage.UpdateTask(ctx, task)
}

// Delete deletes a task
func (t *TaskRepositoryAdapter) Delete(ctx context.Context, id string) error {
	return t.storage.DeleteTask(ctx, id)
}

// FindByID finds a task by ID
func (t *TaskRepositoryAdapter) FindByID(ctx context.Context, id string) (*entities.Task, error) {
	return t.storage.GetTask(ctx, id)
}

// Search searches tasks
func (t *TaskRepositoryAdapter) Search(ctx context.Context, query string, repository string) ([]*entities.Task, error) {
	filters := &ports.TaskFilters{
		Repository: repository,
		Search:     query,
	}

	return t.storage.SearchTasks(ctx, query, filters)
}

// TemplateRepositoryAdapter adapts TemplateStorage to TemplateRepository
type TemplateRepositoryAdapter struct {
	storage ports.TemplateStorage
}

// NewTemplateRepositoryAdapter creates a new template repository adapter
func NewTemplateRepositoryAdapter(storage ports.TemplateStorage) repositories.TemplateRepository {
	return &TemplateRepositoryAdapter{storage: storage}
}

// Create creates a new template
func (t *TemplateRepositoryAdapter) Create(ctx context.Context, template *entities.TaskTemplate) error {
	return t.storage.Create(ctx, template)
}

// FindByID finds a template by ID
func (t *TemplateRepositoryAdapter) FindByID(ctx context.Context, id string) (*entities.TaskTemplate, error) {
	return t.storage.GetByID(ctx, id)
}

// FindByProjectType finds templates by project type
func (t *TemplateRepositoryAdapter) FindByProjectType(ctx context.Context, projectType entities.ProjectType) ([]*entities.TaskTemplate, error) {
	return t.storage.GetByProjectType(ctx, projectType)
}

// Update updates an existing template
func (t *TemplateRepositoryAdapter) Update(ctx context.Context, template *entities.TaskTemplate) error {
	return t.storage.Update(ctx, template)
}

// Delete deletes a template
func (t *TemplateRepositoryAdapter) Delete(ctx context.Context, id string) error {
	return t.storage.Delete(ctx, id)
}

// FindByTags finds templates by tags - simplified implementation
func (t *TemplateRepositoryAdapter) FindByTags(ctx context.Context, tags []string) ([]*entities.TaskTemplate, error) {
	// For now, return empty slice as this requires complex filtering
	return []*entities.TaskTemplate{}, nil
}

// FindByAuthor finds templates by author - simplified implementation
func (t *TemplateRepositoryAdapter) FindByAuthor(ctx context.Context, author string) ([]*entities.TaskTemplate, error) {
	// For now, return empty slice as this requires complex filtering
	return []*entities.TaskTemplate{}, nil
}

// FindPublic finds public templates - simplified implementation
func (t *TemplateRepositoryAdapter) FindPublic(ctx context.Context, limit int) ([]*entities.TaskTemplate, error) {
	// For now, return empty slice as this requires complex filtering
	return []*entities.TaskTemplate{}, nil
}

// GetUsageStats gets template usage stats - simplified implementation
func (t *TemplateRepositoryAdapter) GetUsageStats(ctx context.Context, templateID string) (*repositories.TemplateUsageStats, error) {
	return &repositories.TemplateUsageStats{
		TemplateID:     templateID,
		TotalUses:      0,
		SuccessfulUses: 0,
		FailedUses:     0,
		SuccessRate:    0.0,
		AverageRating:  0.0,
		ProjectTypes:   make(map[string]int),
		UserCount:      0,
		RecentActivity: []repositories.TemplateActivity{},
		Metadata:       make(map[string]interface{}),
	}, nil
}

// GetPopularTemplates gets popular templates - simplified implementation
func (t *TemplateRepositoryAdapter) GetPopularTemplates(ctx context.Context, projectType entities.ProjectType, limit int) ([]*entities.TaskTemplate, error) {
	return t.FindByProjectType(ctx, projectType)
}

// UpdateUsageStats updates template usage stats - simplified implementation
func (t *TemplateRepositoryAdapter) UpdateUsageStats(ctx context.Context, templateID string, usage *repositories.TemplateUsageUpdate) error {
	// For now, do nothing as this requires complex tracking
	return nil
}

// Cleanup performs template cleanup - simplified implementation
func (t *TemplateRepositoryAdapter) Cleanup(ctx context.Context) error {
	// For now, do nothing
	return nil
}

// GetTemplateCount gets template count - simplified implementation
func (t *TemplateRepositoryAdapter) GetTemplateCount(ctx context.Context) (int, error) {
	// For now, return 0 as this requires counting
	return 0, nil
}

// SessionRepositoryAdapter adapts SessionStorage to SessionRepository for services
type SessionRepositoryAdapter struct {
	storage ports.SessionStorage
}

// NewSessionRepositoryAdapter creates a new session repository adapter
func NewSessionRepositoryAdapter(storage ports.SessionStorage) services.SessionRepository {
	return &SessionRepositoryAdapter{storage: storage}
}

// GetByRepository gets sessions by repository
func (s *SessionRepositoryAdapter) GetByRepository(ctx context.Context, repository string) ([]*entities.Session, error) {
	return s.storage.GetByRepository(ctx, repository)
}

// GetByTimeRange gets sessions by time range
func (s *SessionRepositoryAdapter) GetByTimeRange(ctx context.Context, repository string, start, end time.Time) ([]*entities.Session, error) {
	return s.storage.GetByTimeRange(ctx, repository, start, end)
}

// FindByRepository finds sessions by repository (alias for GetByRepository)
func (s *SessionRepositoryAdapter) FindByRepository(ctx context.Context, repository string) ([]*entities.Session, error) {
	return s.GetByRepository(ctx, repository)
}

// FindByTimeRange finds sessions by time range (alias for GetByTimeRange)
func (s *SessionRepositoryAdapter) FindByTimeRange(ctx context.Context, repository string, start, end time.Time) ([]*entities.Session, error) {
	return s.GetByTimeRange(ctx, repository, start, end)
}

// PortsTaskRepositoryAdapter adapts Storage interface to ports.TaskRepository
type PortsTaskRepositoryAdapter struct {
	storage ports.Storage
}

// NewPortsTaskRepositoryAdapter creates a new ports task repository adapter
func NewPortsTaskRepositoryAdapter(storage ports.Storage) ports.TaskRepository {
	return &PortsTaskRepositoryAdapter{storage: storage}
}

// Create stores a new task
func (t *PortsTaskRepositoryAdapter) Create(ctx context.Context, task *entities.Task) error {
	return t.storage.SaveTask(ctx, task)
}

// GetByID retrieves a task by ID
func (t *PortsTaskRepositoryAdapter) GetByID(ctx context.Context, id string) (*entities.Task, error) {
	return t.storage.GetTask(ctx, id)
}

// Update updates an existing task
func (t *PortsTaskRepositoryAdapter) Update(ctx context.Context, task *entities.Task) error {
	return t.storage.UpdateTask(ctx, task)
}

// Delete removes a task
func (t *PortsTaskRepositoryAdapter) Delete(ctx context.Context, id string) error {
	return t.storage.DeleteTask(ctx, id)
}

// List retrieves tasks with filtering options
func (t *PortsTaskRepositoryAdapter) List(ctx context.Context, filter ports.TaskFilter) ([]*entities.Task, error) {
	// Convert filter to storage layer filter if needed
	// For now, use GetTasksByRepository with repository filter
	if filter.Repository != "" {
		return t.storage.GetTasksByRepository(ctx, filter.Repository)
	}
	return []*entities.Task{}, nil
}

// GetByRepository gets tasks by repository
func (t *PortsTaskRepositoryAdapter) GetByRepository(ctx context.Context, repository string, period entities.TimePeriod) ([]*entities.Task, error) {
	// For now, ignore period and get all tasks for repository
	return t.storage.GetTasksByRepository(ctx, repository)
}

// PortsSessionRepositoryAdapter adapts SessionStorage to ports.SessionRepository
type PortsSessionRepositoryAdapter struct {
	storage ports.SessionStorage
}

// NewPortsSessionRepositoryAdapter creates a new ports session repository adapter
func NewPortsSessionRepositoryAdapter(storage ports.SessionStorage) ports.SessionRepository {
	return &PortsSessionRepositoryAdapter{storage: storage}
}

// Create stores a new session
func (s *PortsSessionRepositoryAdapter) Create(ctx context.Context, session *entities.Session) error {
	return s.storage.Create(ctx, session)
}

// GetByID retrieves a session by ID
func (s *PortsSessionRepositoryAdapter) GetByID(ctx context.Context, id string) (*entities.Session, error) {
	return s.storage.GetByID(ctx, id)
}

// Update updates an existing session
func (s *PortsSessionRepositoryAdapter) Update(ctx context.Context, session *entities.Session) error {
	return s.storage.Update(ctx, session)
}

// Delete removes a session
func (s *PortsSessionRepositoryAdapter) Delete(ctx context.Context, id string) error {
	return s.storage.Delete(ctx, id)
}

// List retrieves sessions with filtering options
func (s *PortsSessionRepositoryAdapter) List(ctx context.Context, filter ports.SessionFilter) ([]*entities.Session, error) {
	// Convert filter to storage layer filter if needed
	// For now, use GetByRepository with repository filter
	if filter.Repository != "" {
		return s.storage.GetByRepository(ctx, filter.Repository)
	}
	return []*entities.Session{}, nil
}

// GetByRepository retrieves sessions for a specific repository
func (s *PortsSessionRepositoryAdapter) GetByRepository(ctx context.Context, repository string, period entities.TimePeriod) ([]*entities.Session, error) {
	// Use GetByTimeRange with period if available
	return s.storage.GetByTimeRange(ctx, repository, period.Start, period.End)
}

// GetByTimeRange gets sessions by time range
func (s *PortsSessionRepositoryAdapter) GetByTimeRange(ctx context.Context, repository string, start, end time.Time) ([]*entities.Session, error) {
	return s.storage.GetByTimeRange(ctx, repository, start, end)
}

// PatternRepositoryAdapter adapts PatternStorage to PatternRepository for services
type PatternRepositoryAdapter struct {
	storage ports.PatternStorage
}

// NewPatternRepositoryAdapter creates a new pattern repository adapter
func NewPatternRepositoryAdapter(storage ports.PatternStorage) services.PatternRepository {
	return &PatternRepositoryAdapter{storage: storage}
}

// GetByRepository gets patterns by repository
func (p *PatternRepositoryAdapter) GetByRepository(ctx context.Context, repository string) ([]*entities.TaskPattern, error) {
	return p.storage.GetByRepository(ctx, repository)
}

// FindByRepository finds patterns by repository (alias for GetByRepository)
func (p *PatternRepositoryAdapter) FindByRepository(ctx context.Context, repository string) ([]*entities.TaskPattern, error) {
	return p.GetByRepository(ctx, repository)
}

// GetByType gets patterns by type
func (p *PatternRepositoryAdapter) GetByType(ctx context.Context, patternType entities.PatternType) ([]*entities.TaskPattern, error) {
	return p.storage.GetByType(ctx, patternType)
}

// Create creates a new pattern
func (p *PatternRepositoryAdapter) Create(ctx context.Context, pattern *entities.TaskPattern) error {
	return p.storage.Create(ctx, pattern)
}

// Update updates a pattern
func (p *PatternRepositoryAdapter) Update(ctx context.Context, pattern *entities.TaskPattern) error {
	return p.storage.Update(ctx, pattern)
}

// Delete deletes a pattern
func (p *PatternRepositoryAdapter) Delete(ctx context.Context, id string) error {
	return p.storage.Delete(ctx, id)
}

// GetByID gets a pattern by ID
func (p *PatternRepositoryAdapter) GetByID(ctx context.Context, id string) (*entities.TaskPattern, error) {
	return p.storage.GetByID(ctx, id)
}

// Search searches patterns
func (p *PatternRepositoryAdapter) Search(ctx context.Context, query string) ([]*entities.TaskPattern, error) {
	return p.storage.Search(ctx, query)
}

// ServicesTaskStorageAdapter adapts Storage interface to services.TaskStorage
type ServicesTaskStorageAdapter struct {
	storage ports.Storage
}

// NewServicesTaskStorageAdapter creates a new services task storage adapter
func NewServicesTaskStorageAdapter(storage ports.Storage) services.TaskStorage {
	return &ServicesTaskStorageAdapter{storage: storage}
}

// GetByPeriod gets tasks by time period
func (s *ServicesTaskStorageAdapter) GetByPeriod(ctx context.Context, repository string, period entities.TimePeriod) ([]*entities.Task, error) {
	// Convert time period to filters
	startStr := period.Start.Format(time.RFC3339)
	endStr := period.End.Format(time.RFC3339)

	filters := &ports.TaskFilters{
		Repository:    repository,
		CreatedAfter:  &startStr,
		CreatedBefore: &endStr,
	}

	return s.storage.ListTasks(ctx, repository, filters)
}

// ServicesSessionStorageAdapter adapts SessionStorage interface to services.SessionStorage
type ServicesSessionStorageAdapter struct {
	storage ports.SessionStorage
}

// NewServicesSessionStorageAdapter creates a new services session storage adapter
func NewServicesSessionStorageAdapter(storage ports.SessionStorage) services.SessionStorage {
	return &ServicesSessionStorageAdapter{storage: storage}
}

// GetByPeriod gets sessions by time period
func (s *ServicesSessionStorageAdapter) GetByPeriod(ctx context.Context, repository string, period entities.TimePeriod) ([]*entities.Session, error) {
	return s.storage.GetByTimeRange(ctx, repository, period.Start, period.End)
}

// ServicesVisualizerAdapter adapts ports.Visualizer to services.Visualizer
type ServicesVisualizerAdapter struct {
	visualizer ports.Visualizer
}

// NewServicesVisualizerAdapter creates a new services visualizer adapter
func NewServicesVisualizerAdapter(visualizer ports.Visualizer) services.Visualizer {
	return &ServicesVisualizerAdapter{visualizer: visualizer}
}

// GenerateVisualization generates visualizations
func (v *ServicesVisualizerAdapter) GenerateVisualization(metrics *entities.WorkflowMetrics, format entities.VisFormat) ([]byte, error) {
	return v.visualizer.GenerateVisualization(metrics, format)
}

// ServicesAnalyticsExporterAdapter adapts ports.AnalyticsExporter to services.AnalyticsExporter
type ServicesAnalyticsExporterAdapter struct {
	exporter ports.AnalyticsExporter
}

// NewServicesAnalyticsExporterAdapter creates a new services analytics exporter adapter
func NewServicesAnalyticsExporterAdapter(exporter ports.AnalyticsExporter) services.AnalyticsExporter {
	return &ServicesAnalyticsExporterAdapter{exporter: exporter}
}

// Export exports metrics (without outputPath parameter to match services interface)
func (e *ServicesAnalyticsExporterAdapter) Export(metrics *entities.WorkflowMetrics, format entities.ExportFormat) (string, error) {
	// Use a default output path based on format
	defaultPath := fmt.Sprintf("analytics_%s.%s", time.Now().Format("20060102_150405"), string(format))
	return e.exporter.Export(metrics, format, defaultPath)
}
