package repositories

import (
	"context"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"time"
)

// TemplateRepository interface defines template storage operations
type TemplateRepository interface {
	// CRUD operations
	Create(ctx context.Context, template *entities.TaskTemplate) error
	FindByID(ctx context.Context, id string) (*entities.TaskTemplate, error)
	FindByProjectType(ctx context.Context, projectType entities.ProjectType) ([]*entities.TaskTemplate, error)
	Update(ctx context.Context, template *entities.TaskTemplate) error
	Delete(ctx context.Context, id string) error

	// Search operations
	FindByTags(ctx context.Context, tags []string) ([]*entities.TaskTemplate, error)
	FindByAuthor(ctx context.Context, author string) ([]*entities.TaskTemplate, error)
	FindPublic(ctx context.Context, limit int) ([]*entities.TaskTemplate, error)

	// Analytics operations
	GetUsageStats(ctx context.Context, templateID string) (*TemplateUsageStats, error)
	GetPopularTemplates(ctx context.Context, projectType entities.ProjectType, limit int) ([]*entities.TaskTemplate, error)
	UpdateUsageStats(ctx context.Context, templateID string, usage *TemplateUsageUpdate) error

	// Maintenance operations
	Cleanup(ctx context.Context) error
	GetTemplateCount(ctx context.Context) (int, error)
}

// TemplateUsageStats represents template usage analytics
type TemplateUsageStats struct {
	TemplateID     string                 `json:"template_id"`
	TotalUses      int                    `json:"total_uses"`
	SuccessfulUses int                    `json:"successful_uses"`
	FailedUses     int                    `json:"failed_uses"`
	SuccessRate    float64                `json:"success_rate"`
	AverageRating  float64                `json:"average_rating"`
	ProjectTypes   map[string]int         `json:"project_types"`
	UserCount      int                    `json:"user_count"`
	RecentActivity []TemplateActivity     `json:"recent_activity"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// TemplateActivity represents a single template usage event
type TemplateActivity struct {
	UserID      string    `json:"user_id,omitempty"`
	Repository  string    `json:"repository"`
	ProjectType string    `json:"project_type"`
	Success     bool      `json:"success"`
	UsedAt      time.Time `json:"used_at"`
	Rating      *int      `json:"rating,omitempty"` // 1-5 stars
}

// TemplateUsageUpdate represents an update to template usage
type TemplateUsageUpdate struct {
	Success     bool                   `json:"success"`
	Repository  string                 `json:"repository"`
	ProjectType entities.ProjectType   `json:"project_type"`
	UserID      string                 `json:"user_id,omitempty"`
	Rating      *int                   `json:"rating,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// TaskStats represents task analytics
type TaskStats struct {
	TotalTasks      int                    `json:"total_tasks"`
	CompletedTasks  int                    `json:"completed_tasks"`
	PendingTasks    int                    `json:"pending_tasks"`
	InProgressTasks int                    `json:"in_progress_tasks"`
	CompletionRate  float64                `json:"completion_rate"`
	ByPriority      map[string]int         `json:"by_priority"`
	ByStatus        map[string]int         `json:"by_status"`
	ByType          map[string]int         `json:"by_type"`
	Metadata        map[string]interface{} `json:"metadata"`
}
