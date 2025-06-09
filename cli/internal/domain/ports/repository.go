// Package ports defines interfaces for external adapters
// for the lerian-mcp-memory CLI application.
package ports

import "context"

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
