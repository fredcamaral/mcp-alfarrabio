package mcp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// mockMCPServer simulates an MCP server for testing
type mockMCPServer struct {
	*httptest.Server
	mu          sync.RWMutex
	tasks       map[string]*entities.Task
	failCount   int
	maxFailures int
}

func newMockMCPServer(_ *testing.T) *mockMCPServer {
	mock := &mockMCPServer{
		tasks:       make(map[string]*entities.Task),
		maxFailures: 0,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", mock.handleMCPRequest)

	mock.Server = httptest.NewServer(mux)
	return mock
}

func (m *mockMCPServer) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var request MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Always allow health checks to succeed
	if request.Method == "memory_system" {
		// Check if this is a health operation
		if paramsMap, ok := request.Params.(map[string]interface{}); ok {
			if operation, ok := paramsMap["operation"].(string); ok && operation == "health" {
				response := MCPResponse{
					JSONRPC: "2.0",
					ID:      request.ID,
					Result:  map[string]string{"status": "healthy"},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response)
				return
			}
		}
	}

	// Simulate failures for retry testing (only for non-health requests)
	if m.failCount < m.maxFailures {
		m.failCount++
		http.Error(w, "Simulated failure", http.StatusInternalServerError)
		return
	}

	response := MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
	}

	switch request.Method {
	case "memory_create":
		m.handleMemoryCreate(request.Params, &response)
	case "memory_read":
		m.handleMemoryRead(request.Params, &response)
	case "memory_update":
		m.handleMemoryUpdate(request.Params, &response)
	case "memory_analyze":
		m.handleMemoryAnalyze(request.Params, &response)
	case "memory_system":
		m.handleMemorySystem(request.Params, &response)
	default:
		response.Error = &MCPError{
			Code:    -32601,
			Message: "Method not found",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (m *mockMCPServer) handleMemoryCreate(params interface{}, response *MCPResponse) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Invalid params"}
		return
	}

	operation, ok := paramsMap["operation"].(string)
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Missing operation"}
		return
	}

	options, ok := paramsMap["options"].(map[string]interface{})
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Missing options"}
		return
	}

	// Handle store_chunk operation (used for storing tasks)
	if operation == "store_chunk" {
		// Extract task information from content
		content, ok := options["content"].(string)
		if !ok {
			response.Error = &MCPError{Code: -32602, Message: "Missing content"}
			return
		}

		repository, ok := options["repository"].(string)
		if !ok {
			response.Error = &MCPError{Code: -32602, Message: "Missing repository"}
			return
		}

		// Create a simple task from the content
		task := &entities.Task{
			ID:         uuid.New().String(),
			Content:    content,
			Status:     entities.StatusPending,
			Priority:   entities.PriorityMedium,
			Repository: repository,
		}

		m.tasks[task.ID] = task
		response.Result = map[string]interface{}{
			"status":   "success",
			"chunk_id": task.ID,
		}
		return
	}

	response.Result = map[string]string{"status": "success"}
}

func (m *mockMCPServer) handleMemoryRead(params interface{}, response *MCPResponse) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Invalid params"}
		return
	}

	operation, ok := paramsMap["operation"].(string)
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Missing operation"}
		return
	}

	options, ok := paramsMap["options"].(map[string]interface{})
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Missing options"}
		return
	}

	repo, ok := options["repository"].(string)
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Invalid repository"}
		return
	}

	// Handle search operation (used for retrieving tasks)
	if operation == "search" {
		chunks := make([]interface{}, 0, 5)
		for _, task := range m.tasks {
			if task.Repository != repo {
				continue
			}
			chunkMap := map[string]interface{}{
				"id":         task.ID,
				"content":    task.Content,
				"type":       "task",
				"repository": task.Repository,
				"created_at": task.CreatedAt.Format(time.RFC3339),
				"updated_at": task.UpdatedAt.Format(time.RFC3339),
				"metadata": map[string]interface{}{
					"status":   string(task.Status),
					"priority": string(task.Priority),
					"tags":     task.Tags,
				},
			}

			// Add optional fields if they exist
			if task.EstimatedMins > 0 {
				chunkMap["estimated_mins"] = task.EstimatedMins
			}
			if task.ActualMins > 0 {
				chunkMap["actual_mins"] = task.ActualMins
			}
			if task.ParentTaskID != "" {
				chunkMap["parent_task_id"] = task.ParentTaskID
			}
			if task.SessionID != "" {
				chunkMap["session_id"] = task.SessionID
			}
			if task.CompletedAt != nil {
				chunkMap["completed_at"] = task.CompletedAt.Format(time.RFC3339)
			}

			chunks = append(chunks, chunkMap)
		}

		response.Result = map[string]interface{}{
			"chunks": chunks,
		}
		return
	}

	response.Result = map[string]interface{}{
		"status": "success",
	}
}

func (m *mockMCPServer) handleMemoryUpdate(params interface{}, response *MCPResponse) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Invalid params"}
		return
	}

	operation, ok := paramsMap["operation"].(string)
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Missing operation"}
		return
	}

	options, ok := paramsMap["options"].(map[string]interface{})
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Missing options"}
		return
	}

	// Handle update_thread operation (used for updating tasks)
	if operation == "update_thread" {
		taskID, ok := options["thread_id"].(string)
		if !ok {
			response.Error = &MCPError{Code: -32602, Message: "Invalid thread_id"}
			return
		}

		metadata, ok := options["metadata"].(map[string]interface{})
		if !ok {
			response.Error = &MCPError{Code: -32602, Message: "Invalid metadata"}
			return
		}

		task, exists := m.tasks[taskID]
		if !exists {
			response.Error = &MCPError{Code: -32602, Message: "Task not found"}
			return
		}

		// Update task status from metadata
		if status, ok := metadata["status"].(string); ok {
			task.Status = entities.Status(status)
		}

		response.Result = map[string]string{"status": "success"}
		return
	}

	response.Result = map[string]string{"status": "success"}
}

func (m *mockMCPServer) handleMemoryAnalyze(params interface{}, response *MCPResponse) {
	// Basic implementation for analyze operations
	response.Result = map[string]interface{}{
		"status":   "success",
		"insights": []string{"Mock analysis result"},
	}
}

func (m *mockMCPServer) handleMemorySystem(params interface{}, response *MCPResponse) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Invalid params"}
		return
	}

	operation, ok := paramsMap["operation"].(string)
	if !ok {
		response.Error = &MCPError{Code: -32602, Message: "Missing operation"}
		return
	}

	if operation == "health" {
		response.Result = map[string]string{"status": "healthy"}
		return
	}

	response.Result = map[string]string{"status": "success"}
}

func (m *mockMCPServer) hasTask(taskID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.tasks[taskID]
	return exists
}

func (m *mockMCPServer) setMaxFailures(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxFailures = n
	m.failCount = 0
}

func TestHTTPMCPClient_SyncTask(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	config := &entities.Config{
		Server: entities.ServerConfig{
			URL:     server.URL,
			Timeout: 5,
		},
	}

	logger := slog.Default()
	client := NewHTTPMCPClient(config, logger).(*HTTPMCPClient)
	defer func() { _ = client.Close() }()

	// Wait for initial health check
	time.Sleep(100 * time.Millisecond)

	// Create test task
	task, err := entities.NewTask("Test task", "test-repo")
	require.NoError(t, err)

	// Test sync
	ctx := context.Background()
	err = client.SyncTask(ctx, task)
	require.NoError(t, err)

	// Verify task was synced
	assert.True(t, server.hasTask(task.ID))
}

func TestHTTPMCPClient_GetTasks(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	// Pre-populate server with tasks
	task1ID := uuid.New().String()
	task2ID := uuid.New().String()

	task1 := &entities.Task{
		ID:         task1ID,
		Content:    "Task 1",
		Status:     entities.StatusPending,
		Priority:   entities.PriorityMedium,
		Repository: "test-repo",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	task2 := &entities.Task{
		ID:         task2ID,
		Content:    "Task 2",
		Status:     entities.StatusInProgress,
		Priority:   entities.PriorityHigh,
		Repository: "test-repo",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	server.mu.Lock()
	server.tasks[task1.ID] = task1
	server.tasks[task2.ID] = task2
	server.mu.Unlock()

	config := &entities.Config{
		Server: entities.ServerConfig{
			URL:     server.URL,
			Timeout: 5,
		},
	}

	logger := slog.Default()
	client := NewHTTPMCPClient(config, logger).(*HTTPMCPClient)
	defer func() { _ = client.Close() }()

	// Wait for initial health check
	time.Sleep(100 * time.Millisecond)

	// Get tasks
	ctx := context.Background()
	tasks, err := client.GetTasks(ctx, "test-repo")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Verify task data
	taskIDs := []string{task1ID, task2ID}
	for _, task := range tasks {
		assert.Contains(t, taskIDs, task.ID)
		assert.Equal(t, "test-repo", task.Repository)
	}
}

func TestHTTPMCPClient_UpdateTaskStatus(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	// Pre-populate server with task
	taskID := uuid.New().String()
	task := &entities.Task{
		ID:         taskID,
		Content:    "Test task",
		Status:     entities.StatusPending,
		Priority:   entities.PriorityMedium,
		Repository: "test-repo",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	server.mu.Lock()
	server.tasks[task.ID] = task
	server.mu.Unlock()

	config := &entities.Config{
		Server: entities.ServerConfig{
			URL:     server.URL,
			Timeout: 5,
		},
	}

	logger := slog.Default()
	client := NewHTTPMCPClient(config, logger).(*HTTPMCPClient)
	defer func() { _ = client.Close() }()

	// Wait for initial health check
	time.Sleep(100 * time.Millisecond)

	// Update status
	ctx := context.Background()
	err := client.UpdateTaskStatus(ctx, task.ID, entities.StatusCompleted)
	require.NoError(t, err)

	// Verify status was updated
	server.mu.RLock()
	updatedTask := server.tasks[task.ID]
	server.mu.RUnlock()
	assert.Equal(t, entities.StatusCompleted, updatedTask.Status)
}

func TestHTTPMCPClient_RetryLogic(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	// Configure server to fail twice before succeeding
	server.setMaxFailures(2)

	config := &entities.Config{
		Server: entities.ServerConfig{
			URL:     server.URL,
			Timeout: 5,
		},
	}

	logger := slog.Default()
	client := NewHTTPMCPClient(config, logger).(*HTTPMCPClient)
	defer func() { _ = client.Close() }()

	// Reduce retry delays for faster testing
	client.retryConfig.BaseDelay = 10 * time.Millisecond
	client.retryConfig.MaxDelay = 50 * time.Millisecond

	// Wait for initial health check to complete and client to be online
	for i := 0; i < 100; i++ {
		if client.IsOnline() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.True(t, client.IsOnline(), "Client should be online before running retry test")

	// Create test task
	task, err := entities.NewTask("Test retry task", "test-repo")
	require.NoError(t, err)

	// Test sync with retries
	ctx := context.Background()
	err = client.SyncTask(ctx, task)
	require.NoError(t, err)

	// Verify task was eventually synced
	assert.True(t, server.hasTask(task.ID))
}

func TestHTTPMCPClient_OfflineMode(t *testing.T) {
	// Use invalid URL to simulate offline mode
	config := &entities.Config{
		Server: entities.ServerConfig{
			URL:     "http://localhost:1", // Invalid port
			Timeout: 1,
		},
	}

	logger := slog.Default()
	client := NewHTTPMCPClient(config, logger).(*HTTPMCPClient)
	defer func() { _ = client.Close() }()

	// Wait for health check to fail
	time.Sleep(100 * time.Millisecond)

	// Verify client is offline
	assert.False(t, client.IsOnline())

	// Test operations fail gracefully
	task, err := entities.NewTask("Offline task", "test-repo")
	require.NoError(t, err)

	ctx := context.Background()
	err = client.SyncTask(ctx, task)
	assert.ErrorIs(t, err, ErrMCPOffline)

	_, err = client.GetTasks(ctx, "test-repo")
	assert.ErrorIs(t, err, ErrMCPOffline)

	err = client.UpdateTaskStatus(ctx, task.ID, entities.StatusCompleted)
	assert.ErrorIs(t, err, ErrMCPOffline)
}

func TestHTTPMCPClient_HealthCheck(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	config := &entities.Config{
		Server: entities.ServerConfig{
			URL:     server.URL,
			Timeout: 5,
		},
	}

	logger := slog.Default()
	client := NewHTTPMCPClient(config, logger).(*HTTPMCPClient)
	defer func() { _ = client.Close() }()

	// Test connection
	ctx := context.Background()
	err := client.TestConnection(ctx)
	require.NoError(t, err)

	// Client should be online
	assert.True(t, client.IsOnline())
}

func TestHTTPMCPClient_ContextCancellation(t *testing.T) {
	server := newMockMCPServer(t)
	defer server.Close()

	// Configure server to delay response
	server.setMaxFailures(5) // Will cause delays due to retries

	config := &entities.Config{
		Server: entities.ServerConfig{
			URL:     server.URL,
			Timeout: 5,
		},
	}

	logger := slog.Default()
	client := NewHTTPMCPClient(config, logger).(*HTTPMCPClient)
	defer func() { _ = client.Close() }()

	// Wait for initial health check to complete
	time.Sleep(100 * time.Millisecond)

	// Create test task
	task, err := entities.NewTask("Test cancel task", "test-repo")
	require.NoError(t, err)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	// Operation should fail with context error
	err = client.SyncTask(ctx, task)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestHTTPMCPClient_TaskConversion(t *testing.T) {
	config := &entities.Config{
		Server: entities.ServerConfig{
			URL:     "http://localhost:1",
			Timeout: 1,
		},
	}

	logger := slog.Default()
	client := NewHTTPMCPClient(config, logger).(*HTTPMCPClient)
	defer func() { _ = client.Close() }()

	// Create task with all fields
	task, err := entities.NewTask("Test conversion", "test-repo")
	require.NoError(t, err)

	_ = task.SetPriority(entities.PriorityHigh)
	task.AddTag("test")
	task.AddTag("conversion")
	task.EstimatedMins = 30
	task.ActualMins = 25
	task.ParentTaskID = uuid.New().String()
	task.SessionID = "session-456"

	// Convert to MCP format
	mcpTask := client.convertToMCPFormat(task)

	// Verify all fields
	assert.Equal(t, task.ID, mcpTask["id"])
	assert.Equal(t, task.Content, mcpTask["content"])
	assert.Equal(t, string(task.Status), mcpTask["status"])
	assert.Equal(t, string(task.Priority), mcpTask["priority"])
	assert.Equal(t, task.Repository, mcpTask["repository"])
	assert.Equal(t, task.Tags, mcpTask["tags"])
	assert.Equal(t, task.EstimatedMins, mcpTask["estimated_mins"])
	assert.Equal(t, task.ActualMins, mcpTask["actual_mins"])
	assert.Equal(t, task.ParentTaskID, mcpTask["parent_task_id"])
	assert.Equal(t, task.SessionID, mcpTask["session_id"])

	// Test conversion back
	data := map[string]interface{}{
		"todos": []interface{}{mcpTask},
	}

	tasks, err := client.convertFromMCPFormat(data)
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	convertedTask := tasks[0]
	assert.Equal(t, task.ID, convertedTask.ID)
	assert.Equal(t, task.Content, convertedTask.Content)
	assert.Equal(t, task.Status, convertedTask.Status)
	assert.Equal(t, task.Priority, convertedTask.Priority)
	assert.Equal(t, task.Repository, convertedTask.Repository)
	assert.Equal(t, task.Tags, convertedTask.Tags)
	assert.Equal(t, task.EstimatedMins, convertedTask.EstimatedMins)
	assert.Equal(t, task.ActualMins, convertedTask.ActualMins)
	assert.Equal(t, task.ParentTaskID, convertedTask.ParentTaskID)
	assert.Equal(t, task.SessionID, convertedTask.SessionID)
}

func TestMockMCPClient(t *testing.T) {
	client := NewMockMCPClient()
	ctx := context.Background()

	// Test sync task
	task, err := entities.NewTask("Mock task", "test-repo")
	require.NoError(t, err)

	err = client.SyncTask(ctx, task)
	require.NoError(t, err)

	// Verify task was stored
	storedTask := client.GetTask(task.ID)
	assert.NotNil(t, storedTask)
	assert.Equal(t, task.ID, storedTask.ID)

	// Test get tasks
	tasks, err := client.GetTasks(ctx, "test-repo")
	require.NoError(t, err)
	assert.Len(t, tasks, 1)

	// Test update status
	err = client.UpdateTaskStatus(ctx, task.ID, entities.StatusCompleted)
	require.NoError(t, err)

	storedTask = client.GetTask(task.ID)
	assert.Equal(t, entities.StatusCompleted, storedTask.Status)

	// Test offline mode
	client.SetOnline(false)
	assert.False(t, client.IsOnline())

	err = client.SyncTask(ctx, task)
	assert.ErrorIs(t, err, ErrMCPOffline)

	// Test reset
	client.Reset()
	assert.True(t, client.IsOnline())
	assert.Nil(t, client.GetTask(task.ID))
}
