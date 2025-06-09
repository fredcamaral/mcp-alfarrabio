// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"lerian-mcp-memory-cli/internal/adapters/secondary/mcp"
	"lerian-mcp-memory-cli/internal/di"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
	"lerian-mcp-memory-cli/internal/domain/services"
)

type CLIIntegrationSuite struct {
	suite.Suite
	container     *di.Container
	tempDir       string
	originalHome  string
	mockMCPServer *MockMCPServer
}

func (s *CLIIntegrationSuite) SetupSuite() {
	// Create temporary directory for test isolation
	tempDir, err := os.MkdirTemp("", "lmmc-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Setup test home directory
	s.originalHome = os.Getenv("HOME")
	testHome := filepath.Join(tempDir, "home")
	os.MkdirAll(testHome, 0755)
	os.Setenv("HOME", testHome)

	// Start mock MCP server
	s.mockMCPServer = NewMockMCPServer()
	s.Require().NoError(s.mockMCPServer.Start())

	// Initialize container with test configuration
	container, err := s.setupTestContainer()
	s.Require().NoError(err)
	s.container = container
}

func (s *CLIIntegrationSuite) TearDownSuite() {
	// Cleanup
	if s.mockMCPServer != nil {
		s.mockMCPServer.Stop()
	}

	if s.container != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		s.container.Shutdown(ctx)
		cancel()
	}

	os.Setenv("HOME", s.originalHome)
	os.RemoveAll(s.tempDir)
}

func (s *CLIIntegrationSuite) SetupTest() {
	// Clean state for each test
	s.mockMCPServer.Reset()

	// Clear local storage
	lmmcDir := filepath.Join(s.tempDir, "home", ".lmmc")
	os.RemoveAll(lmmcDir)
}

func (s *CLIIntegrationSuite) setupTestContainer() (*di.Container, error) {
	// Create test configuration
	testConfig := &entities.Config{
		Server: entities.ServerConfig{
			URL:     s.mockMCPServer.URL(),
			Timeout: 5,
		},
		CLI: entities.CLIConfig{
			DefaultRepository: "test-repo",
			OutputFormat:      "json",
			AutoComplete:      false,
			ColorScheme:       "none",
			PageSize:          10,
		},
		Storage: entities.StorageConfig{
			CacheEnabled: true,
			CacheTTL:     300,
			BackupCount:  3,
		},
		Logging: entities.LoggingConfig{
			Level:  "debug",
			Format: "text",
		},
	}

	// Initialize container with test config
	return di.NewTestContainer(testConfig)
}

// Core CLI operation tests
func (s *CLIIntegrationSuite) TestTaskCreationWorkflow() {
	taskService := s.container.TaskService

	// Test task creation
	task, err := taskService.CreateTask(context.Background(),
		"Test task for integration",
		services.WithPriority(entities.PriorityHigh))
	s.Require().NoError(err)
	s.Assert().Equal("Test task for integration", task.Content)
	s.Assert().Equal(entities.PriorityHigh, task.Priority)
	s.Assert().Equal(entities.StatusPending, task.Status)

	// Verify task is stored locally
	ctx := context.Background()
	storedTask, err := s.container.Storage.GetTask(ctx, task.ID)
	s.Require().NoError(err)
	s.Assert().Equal(task.ID, storedTask.ID)

	// Verify MCP sync occurred
	s.Eventually(func() bool {
		return s.mockMCPServer.HasTask(task.ID)
	}, 5*time.Second, 100*time.Millisecond)
}

func (s *CLIIntegrationSuite) TestTaskStatusUpdates() {
	// Create initial task
	task, err := s.container.TaskService.CreateTask(context.Background(),
		"Task for status testing")
	s.Require().NoError(err)

	// Test status transition: pending -> in_progress
	err = s.container.TaskService.UpdateTaskStatus(context.Background(),
		task.ID, entities.StatusInProgress)
	s.Require().NoError(err)

	// Verify status update
	ctx := context.Background()
	updatedTask, err := s.container.Storage.GetTask(ctx, task.ID)
	s.Require().NoError(err)
	s.Assert().Equal(entities.StatusInProgress, updatedTask.Status)

	// Test status transition: in_progress -> completed
	err = s.container.TaskService.UpdateTaskStatus(context.Background(),
		task.ID, entities.StatusCompleted)
	s.Require().NoError(err)

	// Verify completion
	completedTask, err := s.container.Storage.GetTask(ctx, task.ID)
	s.Require().NoError(err)
	s.Assert().Equal(entities.StatusCompleted, completedTask.Status)
	s.Assert().NotNil(completedTask.CompletedAt)
}

func (s *CLIIntegrationSuite) TestOfflineMode() {
	// Stop MCP server to simulate offline mode
	s.mockMCPServer.Stop()

	// Wait for client to detect offline status
	time.Sleep(100 * time.Millisecond)

	// Tasks should still work locally
	task, err := s.container.TaskService.CreateTask(context.Background(),
		"Offline task")
	s.Require().NoError(err)

	// Verify local storage works
	ctx := context.Background()
	storedTask, err := s.container.Storage.GetTask(ctx, task.ID)
	s.Require().NoError(err)
	s.Assert().Equal(task.ID, storedTask.ID)

	// Restart server
	s.Require().NoError(s.mockMCPServer.Start())

	// Wait for reconnection
	time.Sleep(100 * time.Millisecond)
}

func (s *CLIIntegrationSuite) TestTaskFiltering() {
	ctx := context.Background()

	// Create tasks with different attributes
	task1, err := s.container.TaskService.CreateTask(ctx, "High priority task",
		services.WithPriority(entities.PriorityHigh),
		services.WithTags("urgent", "bug"))
	s.Require().NoError(err)

	task2, err := s.container.TaskService.CreateTask(ctx, "Low priority task",
		services.WithPriority(entities.PriorityLow),
		services.WithTags("feature"))
	s.Require().NoError(err)

	task3, err := s.container.TaskService.CreateTask(ctx, "Medium priority task",
		services.WithPriority(entities.PriorityMedium))
	s.Require().NoError(err)

	// Start one task
	err = s.container.TaskService.UpdateTaskStatus(ctx, task2.ID, entities.StatusInProgress)
	s.Require().NoError(err)

	// Test filtering by priority
	highPriority := entities.PriorityHigh
	tasks, err := s.container.TaskService.ListTasks(ctx, ports.TaskFilters{
		Priority: &highPriority,
	})
	s.Require().NoError(err)
	s.Assert().Len(tasks, 1)
	s.Assert().Equal(task1.ID, tasks[0].ID)

	// Test filtering by status
	inProgressStatus := entities.StatusInProgress
	tasks, err = s.container.TaskService.ListTasks(ctx, ports.TaskFilters{
		Status: &inProgressStatus,
	})
	s.Require().NoError(err)
	s.Assert().Len(tasks, 1)
	s.Assert().Equal(task2.ID, tasks[0].ID)

	// Test filtering by tags
	tasks, err = s.container.TaskService.ListTasks(ctx, ports.TaskFilters{
		Tags: []string{"urgent"},
	})
	s.Require().NoError(err)
	s.Assert().Len(tasks, 1)
	s.Assert().Equal(task1.ID, tasks[0].ID)
}

func (s *CLIIntegrationSuite) TestConcurrentTaskOperations() {
	ctx := context.Background()
	const numGoroutines = 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Create tasks concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			content := fmt.Sprintf("Concurrent task %d", index)
			_, err := s.container.TaskService.CreateTask(ctx, content)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		s.Require().NoError(err)
	}

	// Verify all tasks were created
	tasks, err := s.container.TaskService.ListTasks(ctx, ports.TaskFilters{})
	s.Require().NoError(err)
	s.Assert().Len(tasks, numGoroutines)
}

// Mock MCP server for testing
type MockMCPServer struct {
	server *httptest.Server
	tasks  map[string]*entities.Task
	mutex  sync.RWMutex
	url    string
}

func NewMockMCPServer() *MockMCPServer {
	mock := &MockMCPServer{
		tasks: make(map[string]*entities.Task),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", mock.handleMCPRequest)

	return mock
}

func (m *MockMCPServer) Start() error {
	m.server = httptest.NewServer(http.HandlerFunc(m.handleMCPRequest))
	m.url = m.server.URL
	return nil
}

func (m *MockMCPServer) Stop() {
	if m.server != nil {
		m.server.Close()
		m.server = nil
	}
}

func (m *MockMCPServer) URL() string {
	return m.url
}

func (m *MockMCPServer) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	var request mcp.MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := mcp.MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
	}

	switch request.Method {
	case "memory_tasks/todo_write":
		m.handleTaskWrite(request.Params, &response)
	case "memory_tasks/todo_read":
		m.handleTaskRead(request.Params, &response)
	case "memory_system/health":
		response.Result = map[string]string{"status": "healthy"}
	default:
		response.Error = &mcp.MCPError{
			Code:    -32601,
			Message: "Method not found",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *MockMCPServer) handleTaskWrite(params interface{}, response *mcp.MCPResponse) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		response.Error = &mcp.MCPError{Code: -32602, Message: "Invalid params"}
		return
	}

	todos, ok := paramsMap["todos"].([]interface{})
	if !ok {
		response.Error = &mcp.MCPError{Code: -32602, Message: "Invalid todos"}
		return
	}

	for _, todo := range todos {
		taskMap, ok := todo.(map[string]interface{})
		if !ok {
			continue
		}

		task := m.convertMapToTask(taskMap)
		if task != nil {
			m.tasks[task.ID] = task
		}
	}

	response.Result = map[string]string{"status": "success"}
}

func (m *MockMCPServer) handleTaskRead(params interface{}, response *mcp.MCPResponse) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		response.Error = &mcp.MCPError{Code: -32602, Message: "Invalid params"}
		return
	}

	repo, ok := paramsMap["repository"].(string)
	if !ok {
		response.Error = &mcp.MCPError{Code: -32602, Message: "Invalid repository"}
		return
	}

	var todos []interface{}
	for _, task := range m.tasks {
		if task.Repository == repo {
			todos = append(todos, m.convertTaskToMap(task))
		}
	}

	response.Result = map[string]interface{}{
		"todos": todos,
	}
}

func (m *MockMCPServer) HasTask(taskID string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	_, exists := m.tasks[taskID]
	return exists
}

func (m *MockMCPServer) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.tasks = make(map[string]*entities.Task)
}

func (m *MockMCPServer) convertMapToTask(taskMap map[string]interface{}) *entities.Task {
	task := &entities.Task{}

	if id, ok := taskMap["id"].(string); ok {
		task.ID = id
	}
	if content, ok := taskMap["content"].(string); ok {
		task.Content = content
	}
	if status, ok := taskMap["status"].(string); ok {
		task.Status = entities.Status(status)
	}
	if priority, ok := taskMap["priority"].(string); ok {
		task.Priority = entities.Priority(priority)
	}
	if repo, ok := taskMap["repository"].(string); ok {
		task.Repository = repo
	}

	// Parse times
	if createdAt, ok := taskMap["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			task.CreatedAt = t
		}
	}
	if updatedAt, ok := taskMap["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			task.UpdatedAt = t
		}
	}

	return task
}

func (m *MockMCPServer) convertTaskToMap(task *entities.Task) map[string]interface{} {
	return map[string]interface{}{
		"id":         task.ID,
		"content":    task.Content,
		"status":     string(task.Status),
		"priority":   string(task.Priority),
		"repository": task.Repository,
		"created_at": task.CreatedAt.Format(time.RFC3339),
		"updated_at": task.UpdatedAt.Format(time.RFC3339),
	}
}

// Test suite runner
func TestCLIIntegration(t *testing.T) {
	suite.Run(t, new(CLIIntegrationSuite))
}