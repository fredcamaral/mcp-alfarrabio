package ports

import (
	"context"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// PatternStorage defines the interface for pattern persistence operations
type PatternStorage interface {
	Create(ctx context.Context, pattern *entities.TaskPattern) error
	GetByID(ctx context.Context, id string) (*entities.TaskPattern, error)
	GetByRepository(ctx context.Context, repository string) ([]*entities.TaskPattern, error)
	Update(ctx context.Context, pattern *entities.TaskPattern) error
	Delete(ctx context.Context, id string) error
	GetByType(ctx context.Context, patternType entities.PatternType) ([]*entities.TaskPattern, error)
	GetByProjectType(ctx context.Context, projectType entities.ProjectType) ([]*entities.TaskPattern, error)
	Search(ctx context.Context, query string) ([]*entities.TaskPattern, error)
}

// SessionStorage defines the interface for session persistence operations
type SessionStorage interface {
	Create(ctx context.Context, session *entities.Session) error
	GetByID(ctx context.Context, id string) (*entities.Session, error)
	GetByRepository(ctx context.Context, repository string) ([]*entities.Session, error)
	Update(ctx context.Context, session *entities.Session) error
	Delete(ctx context.Context, id string) error
	GetActiveSessions(ctx context.Context, repository string) ([]*entities.Session, error)
	GetByTimeRange(ctx context.Context, repository string, start, end time.Time) ([]*entities.Session, error)
}

// TemplateStorage defines the interface for template persistence operations
type TemplateStorage interface {
	Create(ctx context.Context, template *entities.TaskTemplate) error
	GetByID(ctx context.Context, id string) (*entities.TaskTemplate, error)
	GetByProjectType(ctx context.Context, projectType entities.ProjectType) ([]*entities.TaskTemplate, error)
	Update(ctx context.Context, template *entities.TaskTemplate) error
	Delete(ctx context.Context, id string) error
	GetBuiltInTemplates(ctx context.Context) ([]*entities.TaskTemplate, error)
	Search(ctx context.Context, query string) ([]*entities.TaskTemplate, error)
}

// InsightStorage defines the interface for insight persistence operations
type InsightStorage interface {
	Create(ctx context.Context, insight *entities.CrossRepoInsight) error
	GetByID(ctx context.Context, id string) (*entities.CrossRepoInsight, error)
	GetByProjectType(ctx context.Context, projectType entities.ProjectType) ([]*entities.CrossRepoInsight, error)
	Update(ctx context.Context, insight *entities.CrossRepoInsight) error
	Delete(ctx context.Context, id string) error
	GetShared(ctx context.Context) ([]*entities.CrossRepoInsight, error)
	Search(ctx context.Context, query string) ([]*entities.CrossRepoInsight, error)
}

// Visualizer defines the interface for visualization operations
type Visualizer interface {
	RenderProductivityChart(metrics *entities.WorkflowMetrics) ([]byte, error)
	RenderVelocityChart(metrics *entities.VelocityMetrics) ([]byte, error)
	RenderCycleTimeChart(metrics *entities.CycleTimeMetrics) ([]byte, error)
	RenderBottlenecks(bottlenecks []*entities.Bottleneck) ([]byte, error)
	GenerateVisualization(metrics *entities.WorkflowMetrics, format entities.VisFormat) ([]byte, error)
}

// AnalyticsExporter defines the interface for exporting analytics
type AnalyticsExporter interface {
	Export(metrics *entities.WorkflowMetrics, format entities.ExportFormat, outputPath string) (string, error)
	ExportReport(report *entities.ProductivityReport, format entities.ExportFormat, outputPath string) (string, error)
}

// AnalyticsEngine defines the interface for analytics computation
type AnalyticsEngine interface {
	CalculateProductivityMetrics(ctx context.Context, tasks []*entities.Task, sessions []*entities.Session) (*entities.ProductivityMetrics, error)
	CalculateVelocityMetrics(ctx context.Context, tasks []*entities.Task) (*entities.VelocityMetrics, error)
	CalculateCycleTimeMetrics(ctx context.Context, tasks []*entities.Task) (*entities.CycleTimeMetrics, error)
	GenerateWorkflowMetrics(ctx context.Context, repository string, period entities.TimePeriod) (*entities.WorkflowMetrics, error)
	DetectBottlenecks(ctx context.Context, tasks []*entities.Task, sessions []*entities.Session) ([]*entities.Bottleneck, error)
	AnalyzeTrends(ctx context.Context, repository string, period entities.TimePeriod) (*entities.TrendAnalysis, error)
}
