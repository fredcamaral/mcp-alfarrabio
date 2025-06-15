// Package repositories defines the repository interfaces for domain entities.
package repositories

import (
	"context"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// TaskRepository interface defines task storage operations for services
type TaskRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, task *entities.Task) error
	FindByID(ctx context.Context, id string) (*entities.Task, error)
	Update(ctx context.Context, task *entities.Task) error
	Delete(ctx context.Context, id string) error

	// Query operations
	FindByRepository(ctx context.Context, repository string) ([]*entities.Task, error)
	FindByTimeRange(ctx context.Context, repository string, startTime, endTime time.Time) ([]*entities.Task, error)
	FindByStatus(ctx context.Context, repository string, status entities.Status) ([]*entities.Task, error)
	Search(ctx context.Context, query string, repository string) ([]*entities.Task, error)
}
