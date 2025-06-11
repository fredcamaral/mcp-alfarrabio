// Package ports defines interfaces for external adapters
// for the lerian-mcp-memory CLI application.
package ports

import (
	"context"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// RepositoryInfo contains metadata about a detected repository
type RepositoryInfo struct {
	Path      string
	Name      string
	Provider  string // git, github, gitlab, etc.
	RemoteURL string
	Branch    string
	IsGitRepo bool
	HasRemote bool
}

// RepositoryDetector defines interface for detecting repository context
type RepositoryDetector interface {
	// DetectCurrent detects repository information from current working directory
	DetectCurrent(ctx context.Context) (*RepositoryInfo, error)

	// DetectFromPath detects repository information from specific path
	DetectFromPath(ctx context.Context, path string) (*RepositoryInfo, error)

	// GetRepositoryName extracts a clean repository name for identification
	GetRepositoryName(ctx context.Context, path string) (string, error)

	// IsValidRepository checks if path contains a valid repository
	IsValidRepository(ctx context.Context, path string) bool
}

// TaskRepository defines interface for task data operations
type TaskRepository interface {
	// Create stores a new task
	Create(ctx context.Context, task *entities.Task) error

	// GetByID retrieves a task by ID
	GetByID(ctx context.Context, id string) (*entities.Task, error)

	// Update updates an existing task
	Update(ctx context.Context, task *entities.Task) error

	// Delete removes a task
	Delete(ctx context.Context, id string) error

	// List retrieves tasks with filtering options
	List(ctx context.Context, filter TaskFilter) ([]*entities.Task, error)

	// GetByRepository retrieves tasks for a specific repository
	GetByRepository(ctx context.Context, repository string, period entities.TimePeriod) ([]*entities.Task, error)
}

// TaskFilter defines filtering options for task queries
type TaskFilter struct {
	Repository string
	Status     []string
	Priority   []string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
	Offset     int
}

// SessionRepository defines interface for session data operations
type SessionRepository interface {
	// Create stores a new session
	Create(ctx context.Context, session *entities.Session) error

	// GetByID retrieves a session by ID
	GetByID(ctx context.Context, id string) (*entities.Session, error)

	// Update updates an existing session
	Update(ctx context.Context, session *entities.Session) error

	// Delete removes a session
	Delete(ctx context.Context, id string) error

	// List retrieves sessions with filtering options
	List(ctx context.Context, filter SessionFilter) ([]*entities.Session, error)

	// GetByRepository retrieves sessions for a specific repository
	GetByRepository(ctx context.Context, repository string, period entities.TimePeriod) ([]*entities.Session, error)
}

// SessionFilter defines filtering options for session queries
type SessionFilter struct {
	Repository string
	StartDate  *time.Time
	EndDate    *time.Time
	Limit      int
	Offset     int
}
