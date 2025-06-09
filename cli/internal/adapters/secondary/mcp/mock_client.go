// Package mcp provides MCP client implementation
// for the lerian-mcp-memory CLI application.
package mcp

import (
	"context"
	"sync"

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

// Ensure MockMCPClient implements MCPClient
var _ ports.MCPClient = (*MockMCPClient)(nil)
