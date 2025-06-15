// Package mcp provides MCP client implementation
// for the lerian-mcp-memory CLI application.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// MockMCPClient provides a mock implementation of MCPClient for testing
type MockMCPClient struct {
	mu               sync.RWMutex
	tasks            map[string]*entities.Task
	online           bool
	syncTaskFunc     func(ctx context.Context, task *entities.Task) error
	getTasksFunc     func(ctx context.Context, repository string) ([]*entities.Task, error)
	updateStatusFunc func(ctx context.Context, taskID string, status entities.Status) error
}

// NewMockMCPClient creates a new mock MCP client
func NewMockMCPClient() *MockMCPClient {
	return &MockMCPClient{
		tasks:  make(map[string]*entities.Task),
		online: true,
	}
}

// SyncTask mocks syncing a task
func (m *MockMCPClient) SyncTask(ctx context.Context, task *entities.Task) error {
	if m.syncTaskFunc != nil {
		return m.syncTaskFunc(ctx, task)
	}

	if !m.online {
		return ErrMCPOffline
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.tasks[task.ID] = task
	return nil
}

// GetTasks mocks retrieving tasks
func (m *MockMCPClient) GetTasks(ctx context.Context, repository string) ([]*entities.Task, error) {
	if m.getTasksFunc != nil {
		return m.getTasksFunc(ctx, repository)
	}

	if !m.online {
		return nil, ErrMCPOffline
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var tasks []*entities.Task
	for _, task := range m.tasks {
		if task.Repository == repository {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// UpdateTaskStatus mocks updating task status
func (m *MockMCPClient) UpdateTaskStatus(ctx context.Context, taskID string, status entities.Status) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, taskID, status)
	}

	if !m.online {
		return ErrMCPOffline
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return ErrMCPProtocol
	}

	task.Status = status
	return nil
}

// QueryIntelligence mocks intelligence queries (returns empty suggestions when offline)
func (m *MockMCPClient) QueryIntelligence(ctx context.Context, operation string, options map[string]interface{}) (map[string]interface{}, error) {
	if !m.online {
		return nil, ErrMCPOffline
	}

	// Return mock intelligence response based on operation
	switch operation {
	case "suggest_related":
		return map[string]interface{}{
			"suggestions": []map[string]interface{}{
				{
					"name":        "Mock server suggestion",
					"description": "This is a mock suggestion from the pattern engine",
					"type":        "workflow",
					"confidence":  0.75,
				},
			},
		}, nil
	default:
		return map[string]interface{}{
			"result": "mock response",
		}, nil
	}
}

// TestConnection mocks testing the connection
func (m *MockMCPClient) TestConnection(ctx context.Context) error {
	if !m.online {
		return ErrMCPOffline
	}
	return nil
}

// IsOnline returns the online status
func (m *MockMCPClient) IsOnline() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.online
}

// SetOnline sets the online status for testing
func (m *MockMCPClient) SetOnline(online bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.online = online
}

// GetTask gets a task from the mock storage for testing
func (m *MockMCPClient) GetTask(taskID string) *entities.Task {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tasks[taskID]
}

// Reset clears all data
func (m *MockMCPClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = make(map[string]*entities.Task)
	m.online = true
}

// CallMCPTool calls a generic MCP tool with the given parameters
func (m *MockMCPClient) CallMCPTool(ctx context.Context, tool string, params map[string]interface{}) (map[string]interface{}, error) {
	if !m.online {
		return nil, errors.New("mock MCP client is offline")
	}

	// Mock response for memory tools
	switch tool {
	case "memory_create":
		return map[string]interface{}{
			"id":     fmt.Sprintf("chunk-%d", time.Now().Unix()),
			"status": "created",
		}, nil
	case "memory_read":
		return map[string]interface{}{
			"chunks": []interface{}{
				map[string]interface{}{
					"id":      "chunk-123",
					"content": "Mock memory content",
					"score":   0.95,
				},
			},
		}, nil
	case "memory_analyze", "memory_intelligence":
		return map[string]interface{}{
			"patterns": []interface{}{
				map[string]interface{}{
					"pattern":   "Mock pattern",
					"frequency": 5,
				},
			},
		}, nil
	default:
		return map[string]interface{}{"result": "mock result"}, nil
	}
}

// Ensure MockMCPClient implements MCPClient
var _ ports.MCPClient = (*MockMCPClient)(nil)
