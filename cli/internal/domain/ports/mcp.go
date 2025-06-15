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

	// QueryIntelligence queries the server's intelligence capabilities (patterns, suggestions, etc.)
	QueryIntelligence(ctx context.Context, operation string, options map[string]interface{}) (map[string]interface{}, error)

	// CallMCPTool calls a generic MCP tool with the given parameters
	CallMCPTool(ctx context.Context, tool string, params map[string]interface{}) (map[string]interface{}, error)

	// TestConnection tests the connection to the MCP server
	TestConnection(ctx context.Context) error

	// IsOnline returns true if the MCP server is reachable
	IsOnline() bool
}
