// Package ports defines interfaces for external adapters
// for the lerian-mcp-memory CLI application.
package ports

import (
	"context"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// TaskFilters defines criteria for filtering tasks (maintains backward compatibility)
type TaskFilters struct {
	Status          *entities.Status
	Priority        *entities.Priority
	Repository      string
	Tags            []string
	ParentID        string
	SessionID       string
	CreatedAfter    *string // ISO date string
	CreatedBefore   *string // ISO date string
	UpdatedAfter    *string // ISO date string
	UpdatedBefore   *string // ISO date string
	DueAfter        *string // ISO date string
	DueBefore       *string // ISO date string
	CompletedAfter  *string // ISO date string
	CompletedBefore *string // ISO date string
	OverdueOnly     bool    // Filter to only overdue tasks
	DueSoon         *int    // Tasks due within X hours
	HasDueDate      *bool   // Filter by presence of due date
	Search          string  // Content search

	// Extended fields for advanced search
	ExcludeTags      []string       // Tags to exclude
	SearchFields     []string       // Specific fields to search (content, tags, metadata)
	FuzzySearch      bool           // Enable fuzzy matching
	FuzzyThreshold   float64        // Fuzzy match threshold (0.0-1.0)
	CaseSensitive    bool           // Case-sensitive search
	ExactMatch       bool           // Exact phrase matching
	EstimatedTimeMin *time.Duration // Minimum estimated time
	EstimatedTimeMax *time.Duration // Maximum estimated time
}

// TaskSortOptions defines sorting criteria for task lists
type TaskSortOptions struct {
	Field     TaskSortField
	Direction SortDirection
}

type TaskSortField string

const (
	SortByCreatedAt TaskSortField = "created_at"
	SortByUpdatedAt TaskSortField = "updated_at"
	SortByPriority  TaskSortField = "priority"
	SortByStatus    TaskSortField = "status"
	SortByContent   TaskSortField = "content"
)

type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// Storage defines the interface for task persistence operations
type Storage interface {
	// Task CRUD operations
	SaveTask(ctx context.Context, task *entities.Task) error
	GetTask(ctx context.Context, id string) (*entities.Task, error)
	UpdateTask(ctx context.Context, task *entities.Task) error
	DeleteTask(ctx context.Context, id string) error

	// Task querying operations
	ListTasks(ctx context.Context, repository string, filters *TaskFilters) ([]*entities.Task, error)
	GetTasksByRepository(ctx context.Context, repository string) ([]*entities.Task, error)
	SearchTasks(ctx context.Context, query string, filters *TaskFilters) ([]*entities.Task, error)

	// Bulk operations
	SaveTasks(ctx context.Context, tasks []*entities.Task) error
	DeleteTasks(ctx context.Context, ids []string) error

	// Repository operations
	ListRepositories(ctx context.Context) ([]string, error)
	GetRepositoryStats(ctx context.Context, repository string) (RepositoryStats, error)

	// Health and maintenance
	HealthCheck(ctx context.Context) error
	Backup(ctx context.Context, backupPath string) error
	Restore(ctx context.Context, backupPath string) error
}

// RepositoryStats provides statistics about a repository's tasks
type RepositoryStats struct {
	Repository      string
	TotalTasks      int
	PendingTasks    int
	InProgressTasks int
	CompletedTasks  int
	CancelledTasks  int
	TotalTags       int
	LastActivity    string // ISO timestamp
}
