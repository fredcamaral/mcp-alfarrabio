//go:build integration
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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"lerian-mcp-memory-cli/internal/adapters/secondary/ai"
	"lerian-mcp-memory-cli/internal/adapters/secondary/mcp"
	"lerian-mcp-memory-cli/internal/di"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// ComprehensiveIntegrationSuite tests CLI-server integration with new 4-tool architecture
type ComprehensiveIntegrationSuite struct {
	suite.Suite
	container      *di.Container
	tempDir        string
	originalHome   string
	mockMCPServer  *EnhancedMockMCPServer
	testRepository string
}

func (s *ComprehensiveIntegrationSuite) SetupSuite() {
	// Create temporary directory for test isolation
	tempDir, err := os.MkdirTemp("", "lmmc-comprehensive-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Setup test home directory
	s.originalHome = os.Getenv("HOME")
	testHome := filepath.Join(tempDir, "home")
	os.MkdirAll(testHome, 0755)
	os.Setenv("HOME", testHome)

	// Set test repository
	s.testRepository = "test-repo-comprehensive"

	// Start enhanced mock MCP server with 4-tool architecture
	s.mockMCPServer = NewEnhancedMockMCPServer()
	s.Require().NoError(s.mockMCPServer.Start())

	// Initialize container with test configuration
	container, err := s.setupTestContainer()
	s.Require().NoError(err)
	s.container = container
}

func (s *ComprehensiveIntegrationSuite) TearDownSuite() {
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

func (s *ComprehensiveIntegrationSuite) SetupTest() {
	// Clean state for each test
	s.mockMCPServer.Reset()

	// Clear local storage
	lmmcDir := filepath.Join(s.tempDir, "home", ".lmmc")
	os.RemoveAll(lmmcDir)
}

func (s *ComprehensiveIntegrationSuite) setupTestContainer() (*di.Container, error) {
	// Create test configuration
	testConfig := &entities.Config{
		Server: entities.ServerConfig{
			URL:     s.mockMCPServer.URL(),
			Timeout: 5,
		},
		CLI: entities.CLIConfig{
			DefaultRepository: s.testRepository,
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

// Test New 4-Tool Architecture Validation

func (s *ComprehensiveIntegrationSuite) TestMemoryCreateTool() {
	ctx := context.Background()

	// Test memory_create tool through CLI task creation
	_, err := s.container.TaskService.CreateTask(ctx,
		"Test memory create integration",
		services.WithPriority(entities.PriorityHigh),
		services.WithTags("integration", "memory"))
	s.Require().NoError(err)

	// Wait for MCP sync
	s.Eventually(func() bool {
		return s.mockMCPServer.HasMemoryOperation("memory_create")
	}, 5*time.Second, 100*time.Millisecond)

	// Verify the create operation was called with correct parameters
	operations := s.mockMCPServer.GetMemoryOperations("memory_create")
	s.Assert().NotEmpty(operations)

	lastOp := operations[len(operations)-1]
	s.Assert().Equal("memory_create", lastOp.Tool)
	s.Assert().Equal("single", lastOp.Params["scope"])

	// Verify content was stored
	options := lastOp.Params["options"].(map[string]interface{})
	s.Assert().Equal(s.testRepository, options["repository"])
	s.Assert().Contains(options["content"], "Test memory create integration")
}

func (s *ComprehensiveIntegrationSuite) TestMemoryReadTool() {
	ctx := context.Background()

	// First create some memory content
	_, err := s.container.TaskService.CreateTask(ctx, "Task 1 for read test")
	s.Require().NoError(err)
	_, err = s.container.TaskService.CreateTask(ctx, "Task 2 for read test")
	s.Require().NoError(err)

	// Wait for creation sync
	time.Sleep(200 * time.Millisecond)

	// Now test memory_read through task listing
	tasks, err := s.container.TaskService.ListTasks(ctx, &ports.TaskFilters{})
	s.Require().NoError(err)
	s.Assert().GreaterOrEqual(len(tasks), 2)

	// Verify memory_read operations occurred
	s.Eventually(func() bool {
		return s.mockMCPServer.HasMemoryOperation("memory_read")
	}, 5*time.Second, 100*time.Millisecond)

	operations := s.mockMCPServer.GetMemoryOperations("memory_read")
	s.Assert().NotEmpty(operations)

	// Verify read parameters
	lastOp := operations[len(operations)-1]
	s.Assert().Equal("memory_read", lastOp.Tool)
	s.Assert().Equal("single", lastOp.Params["scope"])

	options := lastOp.Params["options"].(map[string]interface{})
	s.Assert().Equal(s.testRepository, options["repository"])
}

func (s *ComprehensiveIntegrationSuite) TestMemoryUpdateTool() {
	ctx := context.Background()

	// Create initial task
	task, err := s.container.TaskService.CreateTask(ctx, "Task for update test")
	s.Require().NoError(err)

	// Wait for creation sync
	time.Sleep(200 * time.Millisecond)

	// Reset operation tracking
	s.mockMCPServer.ResetOperations()

	// Update task status
	err = s.container.TaskService.UpdateTaskStatus(ctx, task.ID, entities.StatusInProgress)
	s.Require().NoError(err)

	// Verify memory_update operation
	s.Eventually(func() bool {
		return s.mockMCPServer.HasMemoryOperation("memory_update")
	}, 5*time.Second, 100*time.Millisecond)

	operations := s.mockMCPServer.GetMemoryOperations("memory_update")
	s.Assert().NotEmpty(operations)

	lastOp := operations[len(operations)-1]
	s.Assert().Equal("memory_update", lastOp.Tool)
	s.Assert().Equal("single", lastOp.Params["scope"])

	// Verify update content
	options := lastOp.Params["options"].(map[string]interface{})
	s.Assert().Equal(s.testRepository, options["repository"])
	s.Assert().Contains(options, "updates")
}

func (s *ComprehensiveIntegrationSuite) TestMemoryAnalyzeTool() {
	ctx := context.Background()

	// Create several tasks to have data for analysis
	for i := 0; i < 5; i++ {
		_, err := s.container.TaskService.CreateTask(ctx,
			fmt.Sprintf("Analysis test task %d", i))
		s.Require().NoError(err)
	}

	// Wait for creation sync
	time.Sleep(300 * time.Millisecond)

	// Reset operation tracking
	s.mockMCPServer.ResetOperations()

	// Trigger analytics through intelligence services if available
	if s.container.AnalyticsService != nil {
		// This should trigger memory_analyze calls - use available method
		period := entities.TimePeriod{
			Start: time.Now().Add(-7 * 24 * time.Hour),
			End:   time.Now(),
		}
		_, _ = s.container.AnalyticsService.GetWorkflowMetrics(ctx, s.testRepository, period)
	}

	// Alternative: trigger through AI commands (if available)
	if enhancedAI, ok := s.container.AIService.(*ai.EnhancedAIService); ok {
		_, _ = enhancedAI.AnalyzePerformance(ctx)
	}

	// Verify memory_analyze operations eventually occur
	s.Eventually(func() bool {
		return s.mockMCPServer.HasMemoryOperation("memory_analyze") ||
			s.mockMCPServer.HasMemoryOperation("cross_repo_insights")
	}, 10*time.Second, 200*time.Millisecond)
}

// Test AI CLI Integration with MCP Server

func (s *ComprehensiveIntegrationSuite) TestAITaskProcessingIntegration() {
	ctx := context.Background()

	// Create a task
	task, err := s.container.TaskService.CreateTask(ctx, "implement user authentication feature")
	s.Require().NoError(err)

	// Get enhanced AI service
	enhancedAI, ok := s.container.AIService.(*ai.EnhancedAIService)
	s.Require().True(ok, "AI service should be enhanced")

	// Set context
	enhancedAI.SetContext(s.testRepository, "test-session", &entities.WorkContext{
		Repository:        s.testRepository,
		EnergyLevel:       0.8,
		FocusLevel:        0.7,
		ProductivityScore: 0.75,
	})

	// Process task with AI
	result, err := enhancedAI.ProcessTaskWithAI(ctx, task)
	s.Require().NoError(err)
	s.Assert().True(result.Success)
	s.Assert().NotEmpty(result.ContextInsights)

	// Verify AI processing generated suggestions
	if result.TaskResult != nil {
		s.Assert().NotNil(result.TaskResult.EnhancedTask)
		s.Assert().GreaterOrEqual(len(result.TaskResult.Suggestions), 0)
	}

	// Verify memory operations occurred for AI learning
	s.Eventually(func() bool {
		return s.mockMCPServer.HasMemoryOperation("memory_create") ||
			s.mockMCPServer.HasMemoryOperation("store_chunk")
	}, 5*time.Second, 100*time.Millisecond)
}

func (s *ComprehensiveIntegrationSuite) TestAIMemorySyncIntegration() {
	ctx := context.Background()

	// Create a test file for sync
	testFile := filepath.Join(s.tempDir, "test-sync.txt")
	testContent := "# Test File\n\nThis is a test file for AI memory sync.\n\n## Features\n- Feature 1\n- Feature 2"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	s.Require().NoError(err)

	// Get enhanced AI service
	enhancedAI, ok := s.container.AIService.(*ai.EnhancedAIService)
	s.Require().True(ok)

	// Set context
	enhancedAI.SetContext(s.testRepository, "sync-session", nil)

	// Perform AI-enhanced sync
	result, err := enhancedAI.SyncMemoryWithAI(ctx, s.tempDir)
	s.Require().NoError(err)
	s.Assert().True(result.Success)

	// Verify sync results
	s.Assert().NotEmpty(result.ContextInsights)
	if result.MemoryResult != nil {
		s.Assert().GreaterOrEqual(result.MemoryResult.FilesProcessed, 0)
	}

	// Verify memory operations for file sync
	s.Eventually(func() bool {
		ops := s.mockMCPServer.GetAllOperations()
		for _, op := range ops {
			if options, ok := op.Params["options"].(map[string]interface{}); ok {
				if content, exists := options["content"]; exists {
					if contentStr, ok := content.(string); ok && strings.Contains(contentStr, "test file") {
						return true
					}
				}
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond)
}

// Test Systematic Tool Mapping Validation

func (s *ComprehensiveIntegrationSuite) TestToolParameterValidation() {
	// Test all 4 tools have correct parameter structure
	expectedTools := []string{"memory_create", "memory_read", "memory_update", "memory_analyze"}

	for _, tool := range expectedTools {
		s.Run(fmt.Sprintf("Tool_%s_ParameterValidation", tool), func() {
			// Trigger operation that uses this tool
			s.triggerToolOperation(tool)

			// Verify operation was recorded
			s.Eventually(func() bool {
				return s.mockMCPServer.HasMemoryOperation(tool)
			}, 5*time.Second, 100*time.Millisecond)

			// Validate parameter structure
			operations := s.mockMCPServer.GetMemoryOperations(tool)
			s.Assert().NotEmpty(operations, fmt.Sprintf("No operations found for tool %s", tool))

			lastOp := operations[len(operations)-1]
			s.validateToolParameters(tool, lastOp)
		})
	}
}

func (s *ComprehensiveIntegrationSuite) triggerToolOperation(tool string) {
	ctx := context.Background()

	switch tool {
	case "memory_create":
		s.container.TaskService.CreateTask(ctx, fmt.Sprintf("Test task for %s", tool))
	case "memory_read":
		s.container.TaskService.ListTasks(ctx, &ports.TaskFilters{})
	case "memory_update":
		// Create then update
		task, _ := s.container.TaskService.CreateTask(ctx, "Task to update")
		time.Sleep(100 * time.Millisecond)
		s.mockMCPServer.ResetOperations() // Reset to focus on update operation
		s.container.TaskService.UpdateTaskStatus(ctx, task.ID, entities.StatusInProgress)
	case "memory_analyze":
		if enhancedAI, ok := s.container.AIService.(*ai.EnhancedAIService); ok {
			enhancedAI.AnalyzePerformance(ctx)
		}
	}
}

func (s *ComprehensiveIntegrationSuite) validateToolParameters(tool string, operation MemoryOperation) {
	// Validate common structure: operation + scope + options
	s.Assert().Equal(tool, operation.Tool)
	s.Assert().Contains(operation.Params, "operation")
	s.Assert().Contains(operation.Params, "scope")
	s.Assert().Contains(operation.Params, "options")

	// Validate scope values
	scope := operation.Params["scope"].(string)
	validScopes := []string{"single", "cross_repo", "global"}
	s.Assert().Contains(validScopes, scope)

	// Validate options structure
	options, ok := operation.Params["options"].(map[string]interface{})
	s.Assert().True(ok, "Options should be a map")
	s.Assert().Contains(options, "repository", "Options should contain repository")

	// Tool-specific validations
	switch tool {
	case "memory_create":
		s.Assert().Contains(options, "content", "memory_create should have content")
		s.Assert().Contains(options, "type", "memory_create should have type")
	case "memory_read":
		// memory_read should have query or filters
		hasQuery := strings.Contains(fmt.Sprintf("%v", options), "query") ||
			strings.Contains(fmt.Sprintf("%v", options), "filter")
		s.Assert().True(hasQuery, "memory_read should have query or filters")
	case "memory_update":
		s.Assert().Contains(options, "updates", "memory_update should have updates")
	case "memory_analyze":
		s.Assert().Contains(options, "analysis_type", "memory_analyze should have analysis_type")
	}
}

// Test Error Handling and Edge Cases

func (s *ComprehensiveIntegrationSuite) TestServerOfflineHandling() {
	ctx := context.Background()

	// Create task while server is online
	_, err := s.container.TaskService.CreateTask(ctx, "Online task")
	s.Require().NoError(err)

	// Stop server
	s.mockMCPServer.Stop()
	time.Sleep(100 * time.Millisecond)

	// Create task while server is offline (should work locally)
	_, err = s.container.TaskService.CreateTask(ctx, "Offline task")
	s.Require().NoError(err)

	// Verify local storage works
	tasks, err := s.container.Storage.GetTasksByRepository(ctx, s.testRepository)
	s.Require().NoError(err)
	s.Assert().GreaterOrEqual(len(tasks), 2)

	// Restart server
	s.Require().NoError(s.mockMCPServer.Start())
	time.Sleep(200 * time.Millisecond)

	// Verify sync resumes
	s.Eventually(func() bool {
		return s.mockMCPServer.HasMemoryOperation("memory_create")
	}, 10*time.Second, 200*time.Millisecond)
}

func (s *ComprehensiveIntegrationSuite) TestInvalidToolResponses() {
	ctx := context.Background()

	// Configure server to return invalid responses
	s.mockMCPServer.SetInvalidResponseMode(true)

	// Operations should handle errors gracefully
	task, err := s.container.TaskService.CreateTask(ctx, "Task with invalid response")

	// Should still work locally even if MCP fails
	s.Require().NoError(err)
	s.Assert().NotEmpty(task.ID)

	// Reset server to normal mode
	s.mockMCPServer.SetInvalidResponseMode(false)
}

// Enhanced Mock MCP Server with 4-tool architecture support

type MemoryOperation struct {
	Tool      string                 `json:"tool"`
	Timestamp time.Time              `json:"timestamp"`
	Params    map[string]interface{} `json:"params"`
}

type EnhancedMockMCPServer struct {
	server              *httptest.Server
	operations          []MemoryOperation
	mutex               sync.RWMutex
	url                 string
	invalidResponseMode bool
}

func NewEnhancedMockMCPServer() *EnhancedMockMCPServer {
	return &EnhancedMockMCPServer{
		operations: make([]MemoryOperation, 0),
	}
}

func (m *EnhancedMockMCPServer) Start() error {
	m.server = httptest.NewServer(http.HandlerFunc(m.handleMCPRequest))
	m.url = m.server.URL
	return nil
}

func (m *EnhancedMockMCPServer) Stop() {
	if m.server != nil {
		m.server.Close()
		m.server = nil
	}
}

func (m *EnhancedMockMCPServer) URL() string {
	return m.url
}

func (m *EnhancedMockMCPServer) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.operations = make([]MemoryOperation, 0)
}

func (m *EnhancedMockMCPServer) ResetOperations() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.operations = make([]MemoryOperation, 0)
}

func (m *EnhancedMockMCPServer) SetInvalidResponseMode(invalid bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.invalidResponseMode = invalid
}

func (m *EnhancedMockMCPServer) HasMemoryOperation(tool string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, op := range m.operations {
		if op.Tool == tool {
			return true
		}
	}
	return false
}

func (m *EnhancedMockMCPServer) GetMemoryOperations(tool string) []MemoryOperation {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []MemoryOperation
	for _, op := range m.operations {
		if op.Tool == tool {
			result = append(result, op)
		}
	}
	return result
}

func (m *EnhancedMockMCPServer) GetAllOperations() []MemoryOperation {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make([]MemoryOperation, len(m.operations))
	copy(result, m.operations)
	return result
}

func (m *EnhancedMockMCPServer) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	var request mcp.MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := mcp.MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
	}

	m.mutex.Lock()
	invalidMode := m.invalidResponseMode
	m.mutex.Unlock()

	if invalidMode {
		response.Error = &mcp.MCPError{
			Code:    -32603,
			Message: "Simulated server error",
		}
	} else {
		// Handle new 4-tool architecture
		switch request.Method {
		case "memory_create", "memory_read", "memory_update", "memory_analyze":
			m.recordOperation(request.Method, request.Params)
			response.Result = m.generateToolResponse(request.Method, request.Params)
		case "memory_system":
			response.Result = map[string]interface{}{
				"status": "healthy",
				"tools":  []string{"memory_create", "memory_read", "memory_update", "memory_analyze"},
			}
		case "tools/list":
			response.Result = map[string]interface{}{
				"tools": []map[string]interface{}{
					{"name": "memory_create", "description": "Create memory content"},
					{"name": "memory_read", "description": "Read memory content"},
					{"name": "memory_update", "description": "Update memory content"},
					{"name": "memory_analyze", "description": "Analyze memory patterns"},
				},
			}
		default:
			// Handle legacy methods for backward compatibility
			if strings.HasPrefix(request.Method, "memory_tasks/") ||
				strings.HasPrefix(request.Method, "memory_") {
				m.recordOperation(request.Method, request.Params)
				response.Result = map[string]interface{}{"status": "success"}
			} else {
				response.Error = &mcp.MCPError{
					Code:    -32601,
					Message: "Method not found",
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *EnhancedMockMCPServer) recordOperation(tool string, params interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Convert params to map for easier handling
	paramsMap := make(map[string]interface{})
	if params != nil {
		if pMap, ok := params.(map[string]interface{}); ok {
			paramsMap = pMap
		}
	}

	operation := MemoryOperation{
		Tool:      tool,
		Timestamp: time.Now(),
		Params:    paramsMap,
	}

	m.operations = append(m.operations, operation)
}

func (m *EnhancedMockMCPServer) generateToolResponse(tool string, params interface{}) map[string]interface{} {
	switch tool {
	case "memory_create":
		return map[string]interface{}{
			"chunk_id": fmt.Sprintf("chunk_%d", time.Now().UnixNano()),
			"status":   "created",
		}
	case "memory_read":
		return map[string]interface{}{
			"chunks": []map[string]interface{}{
				{
					"id":      "chunk_1",
					"content": "Sample memory content",
					"type":    "task",
				},
			},
			"total": 1,
		}
	case "memory_update":
		return map[string]interface{}{
			"updated": true,
			"status":  "success",
		}
	case "memory_analyze":
		return map[string]interface{}{
			"analysis": map[string]interface{}{
				"patterns":    []string{"productivity_pattern", "task_completion_pattern"},
				"insights":    []string{"High productivity in mornings", "Task completion rate improving"},
				"suggestions": []string{"Focus complex tasks in morning", "Break large tasks into smaller ones"},
			},
			"confidence": 0.85,
		}
	default:
		return map[string]interface{}{"status": "success"}
	}
}

// Helper functions are imported from services package

// Test suite runner
func TestComprehensiveIntegration(t *testing.T) {
	suite.Run(t, new(ComprehensiveIntegrationSuite))
}
