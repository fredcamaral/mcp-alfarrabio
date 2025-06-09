package services

import (
	"context"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// Repository interfaces for services
type PatternRepository interface {
	Create(ctx context.Context, pattern *entities.TaskPattern) error
	GetByRepository(ctx context.Context, repository string) ([]*entities.TaskPattern, error)
	GetByType(ctx context.Context, patternType entities.PatternType) ([]*entities.TaskPattern, error)
	FindByRepository(ctx context.Context, repository string) ([]*entities.TaskPattern, error)
}

type SessionRepository interface {
	GetByRepository(ctx context.Context, repository string) ([]*entities.Session, error)
	GetByTimeRange(ctx context.Context, repository string, start, end time.Time) ([]*entities.Session, error)
	FindByRepository(ctx context.Context, repository string) ([]*entities.Session, error)
	FindByTimeRange(ctx context.Context, repository string, start, end time.Time) ([]*entities.Session, error)
}

type AnalyticsEngine interface {
	CalculateProductivityMetrics(ctx context.Context, tasks []*entities.Task, sessions []*entities.Session) (*entities.ProductivityMetrics, error)
	CalculateVelocityMetrics(ctx context.Context, tasks []*entities.Task) (*entities.VelocityMetrics, error)
	CalculateCycleTimeMetrics(ctx context.Context, tasks []*entities.Task) (*entities.CycleTimeMetrics, error)
	GenerateWorkflowMetrics(ctx context.Context, repository string, period entities.TimePeriod) (*entities.WorkflowMetrics, error)
	DetectBottlenecks(ctx context.Context, tasks []*entities.Task, sessions []*entities.Session) ([]*entities.Bottleneck, error)
	AnalyzeTrends(ctx context.Context, repository string, period entities.TimePeriod) (*entities.TrendAnalysis, error)
}

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
}

type MCPClient interface {
	Connect() error
	Disconnect() error
	Call(ctx context.Context, method string, params map[string]interface{}) (*MCPResponse, error)
}

type MCPResponse struct {
	SimilarRepositories []SimilarRepository `json:"similar_repositories"`
}

type SimilarRepository struct {
	Repository     string      `json:"repository"`
	Score          float64     `json:"score"`
	Dimensions     interface{} `json:"dimensions"`
	SharedPatterns []string    `json:"shared_patterns"`
}

type ProjectClassifier interface {
	ClassifyProject(ctx context.Context, path string) (entities.ProjectType, float64, error)
	GetProjectCharacteristics(ctx context.Context, path string) (*entities.ProjectCharacteristics, error)
	SuggestProjectType(characteristics *entities.ProjectCharacteristics) (entities.ProjectType, float64)
}
