// Package ports defines interfaces for external adapters
// for the lerian-mcp-memory CLI application.
package ports

import (
	"context"
	"lerian-mcp-memory-cli/internal/domain/entities"
)

// MCPClient defines the interface for MCP server communication
type MCPClient interface {
	// SyncTask syncs a single task with the MCP server
	SyncTask(ctx context.Context, task *entities.Task) error

	// GetTasks retrieves tasks from the MCP server for a repository
	GetTasks(ctx context.Context, repository string) ([]*entities.Task, error)

	// UpdateTaskStatus updates a task's status on the MCP server
	UpdateTaskStatus(ctx context.Context, taskID string, status entities.Status) error

	// TestConnection tests the connection to the MCP server
	TestConnection(ctx context.Context) error

	// IsOnline returns true if the MCP server is reachable
	IsOnline() bool
}
