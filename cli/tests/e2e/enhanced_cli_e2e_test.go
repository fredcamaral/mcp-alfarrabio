//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"lerian-mcp-memory-cli/internal/adapters/secondary/mcp"
)

// EnhancedCLIE2ESuite tests complete CLI commands end-to-end with AI features
type EnhancedCLIE2ESuite struct {
	suite.Suite
	tempDir       string
	originalHome  string
	cliPath       string
	mockMCPServer *E2EMockMCPServer
	serverURL     string
}

func (s *EnhancedCLIE2ESuite) SetupSuite() {
	// Create temporary directory for test isolation
	tempDir, err := os.MkdirTemp("", "lmmc-e2e-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Setup test home directory
	s.originalHome = os.Getenv("HOME")
	testHome := filepath.Join(tempDir, "home")
	os.MkdirAll(testHome, 0755)
	os.Setenv("HOME", testHome)

	// Build CLI for testing
	s.buildCLI()

	// Start mock MCP server
	s.mockMCPServer = NewE2EMockMCPServer()
	s.Require().NoError(s.mockMCPServer.Start())
	s.serverURL = s.mockMCPServer.URL()

	// Create test configuration
	s.createTestConfig()
}

func (s *EnhancedCLIE2ESuite) TearDownSuite() {
	// Cleanup
	if s.mockMCPServer != nil {
		s.mockMCPServer.Stop()
	}

	os.Setenv("HOME", s.originalHome)
	os.RemoveAll(s.tempDir)

	// Remove CLI binary
	if s.cliPath != "" {
		os.Remove(s.cliPath)
	}
}

func (s *EnhancedCLIE2ESuite) SetupTest() {
	// Reset server state for each test
	s.mockMCPServer.Reset()

	// Clear CLI local storage
	lmmcDir := filepath.Join(s.tempDir, "home", ".lmmc")
	os.RemoveAll(lmmcDir)
}

func (s *EnhancedCLIE2ESuite) buildCLI() {
	// Build the CLI binary
	cliDir := filepath.Join("..", "..", "cmd", "lmmc")
	s.cliPath = filepath.Join(s.tempDir, "lmmc")

	cmd := exec.Command("go", "build", "-o", s.cliPath, ".")
	cmd.Dir = cliDir

	output, err := cmd.CombinedOutput()
	s.Require().NoError(err, "Failed to build CLI: %s", string(output))

	// Verify CLI was built
	s.Require().FileExists(s.cliPath)
}

func (s *EnhancedCLIE2ESuite) createTestConfig() {
	configDir := filepath.Join(s.tempDir, "home", ".lmmc")
	os.MkdirAll(configDir, 0755)

	config := fmt.Sprintf(`
server:
  url: %s
  timeout: 5
cli:
  default_repository: "e2e-test-repo"
  output_format: "json"
  auto_complete: false
  color_scheme: "none"
  page_size: 10
storage:
  cache_enabled: true
  cache_ttl: 300
  backup_count: 3
logging:
  level: "debug"
  format: "text"
`, s.serverURL)

	configPath := filepath.Join(configDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(config), 0644)
	s.Require().NoError(err)
}

func (s *EnhancedCLIE2ESuite) runCLI(args ...string) (string, string, error) {
	cmd := exec.Command(s.cliPath, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", filepath.Join(s.tempDir, "home")))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// Test Core CLI Commands

func (s *EnhancedCLIE2ESuite) TestTaskLifecycleCLI() {
	// Test task creation
	stdout, stderr, err := s.runCLI("add", "Test e2e task creation")
	s.Require().NoError(err, "stderr: %s", stderr)

	// Parse JSON output
	var result map[string]interface{}
	err = json.Unmarshal([]byte(stdout), &result)
	s.Require().NoError(err)

	taskID := result["id"].(string)
	s.Assert().NotEmpty(taskID)
	s.Assert().Equal("Test e2e task creation", result["content"])

	// Test task listing
	stdout, stderr, err = s.runCLI("list")
	s.Require().NoError(err, "stderr: %s", stderr)

	var listResult map[string]interface{}
	err = json.Unmarshal([]byte(stdout), &listResult)
	s.Require().NoError(err)

	tasks := listResult["tasks"].([]interface{})
	s.Assert().Len(tasks, 1)

	// Test task status update
	stdout, stderr, err = s.runCLI("start", taskID)
	s.Require().NoError(err, "stderr: %s", stderr)

	// Verify status change
	stdout, stderr, err = s.runCLI("list", "--status", "in_progress")
	s.Require().NoError(err, "stderr: %s", stderr)

	err = json.Unmarshal([]byte(stdout), &listResult)
	s.Require().NoError(err)

	tasks = listResult["tasks"].([]interface{})
	s.Assert().Len(tasks, 1)

	task := tasks[0].(map[string]interface{})
	s.Assert().Equal("in_progress", task["status"])

	// Test task completion
	stdout, stderr, err = s.runCLI("done", taskID)
	s.Require().NoError(err, "stderr: %s", stderr)

	// Verify completion
	stdout, stderr, err = s.runCLI("list", "--status", "completed")
	s.Require().NoError(err, "stderr: %s", stderr)

	err = json.Unmarshal([]byte(stdout), &listResult)
	s.Require().NoError(err)

	tasks = listResult["tasks"].([]interface{})
	s.Assert().Len(tasks, 1)

	task = tasks[0].(map[string]interface{})
	s.Assert().Equal("completed", task["status"])
}

func (s *EnhancedCLIE2ESuite) TestTaskFilteringCLI() {
	// Create tasks with different properties
	s.runCLI("add", "High priority task", "--priority", "high", "--tags", "urgent,bug")
	s.runCLI("add", "Low priority task", "--priority", "low", "--tags", "feature")
	s.runCLI("add", "Medium priority task", "--priority", "medium")

	// Test priority filtering
	stdout, stderr, err := s.runCLI("list", "--priority", "high")
	s.Require().NoError(err, "stderr: %s", stderr)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(stdout), &result)
	s.Require().NoError(err)

	tasks := result["tasks"].([]interface{})
	s.Assert().Len(tasks, 1)

	task := tasks[0].(map[string]interface{})
	s.Assert().Equal("high", task["priority"])
	s.Assert().Contains(task["content"], "High priority task")

	// Test tag filtering
	stdout, stderr, err = s.runCLI("list", "--tags", "urgent")
	s.Require().NoError(err, "stderr: %s", stderr)

	err = json.Unmarshal([]byte(stdout), &result)
	s.Require().NoError(err)

	tasks = result["tasks"].([]interface{})
	s.Assert().Len(tasks, 1)

	task = tasks[0].(map[string]interface{})
	tags := task["tags"].([]interface{})
	tagStrings := make([]string, len(tags))
	for i, tag := range tags {
		tagStrings[i] = tag.(string)
	}
	s.Assert().Contains(tagStrings, "urgent")
}

// Test New AI Commands

func (s *EnhancedCLIE2ESuite) TestAIProcessCommandCLI() {
	// First create a task
	stdout, stderr, err := s.runCLI("add", "implement user authentication feature")
	s.Require().NoError(err, "stderr: %s", stderr)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(stdout), &result)
	s.Require().NoError(err)

	taskID := result["id"].(string)

	// Test AI process command
	stdout, stderr, err = s.runCLI("ai", "process", taskID)
	s.Require().NoError(err, "stderr: %s", stderr)

	// Verify AI processing output contains expected sections
	s.Assert().Contains(stdout, "AI Task Processing Results")
	s.Assert().Contains(stdout, "Status: Success")

	// Should have generated suggestions or insights
	suggestionsFound := strings.Contains(stdout, "AI Suggestions") ||
		strings.Contains(stdout, "Context Insights") ||
		strings.Contains(stdout, "Enhancements Applied")
	s.Assert().True(suggestionsFound, "AI processing should generate suggestions or insights")
}

func (s *EnhancedCLIE2ESuite) TestAISyncCommandCLI() {
	// Create test files for sync
	testDir := filepath.Join(s.tempDir, "test-sync")
	os.MkdirAll(testDir, 0755)

	testFiles := map[string]string{
		"README.md":   "# Test Project\n\nThis is a test project for AI sync.",
		"main.go":     "package main\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}",
		"config.yaml": "app:\n  name: test-app\n  version: 1.0.0",
	}

	for filename, content := range testFiles {
		err := os.WriteFile(filepath.Join(testDir, filename), []byte(content), 0644)
		s.Require().NoError(err)
	}

	// Test AI sync command
	stdout, stderr, err := s.runCLI("ai", "sync", testDir)
	s.Require().NoError(err, "stderr: %s", stderr)

	// Verify sync output
	s.Assert().Contains(stdout, "AI Memory Sync Results")
	s.Assert().Contains(stdout, "Status: Success")
	s.Assert().Contains(stdout, "Files Processed:")

	// Should show some files were processed
	s.Assert().True(
		strings.Contains(stdout, "Files Processed: 3") ||
			strings.Contains(stdout, "Files Processed: 1") ||
			strings.Contains(stdout, "Files Processed: 2"),
		"Should process some files")
}

func (s *EnhancedCLIE2ESuite) TestAIOptimizeCommandCLI() {
	// Create some tasks first to have data for optimization
	s.runCLI("add", "Task 1 for optimization")
	s.runCLI("add", "Task 2 for optimization")
	s.runCLI("add", "Task 3 for optimization")

	// Test AI optimize command
	stdout, stderr, err := s.runCLI("ai", "optimize")
	s.Require().NoError(err, "stderr: %s", stderr)

	// Verify optimization output
	s.Assert().Contains(stdout, "AI Workflow Optimization Results")
	s.Assert().Contains(stdout, "Status: Success")

	// Should provide optimization recommendations
	recommendationsFound := strings.Contains(stdout, "Optimization Recommendations") ||
		strings.Contains(stdout, "Storage Optimizations") ||
		strings.Contains(stdout, "optimize")
	s.Assert().True(recommendationsFound, "Should provide optimization recommendations")
}

func (s *EnhancedCLIE2ESuite) TestAIAnalyzeCommandCLI() {
	// Create tasks with different patterns
	s.runCLI("add", "Bug fix task", "--tags", "bug,urgent")
	s.runCLI("add", "Feature development", "--tags", "feature")
	s.runCLI("add", "Code review", "--tags", "review")

	// Test AI analyze command
	stdout, stderr, err := s.runCLI("ai", "analyze")
	s.Require().NoError(err, "stderr: %s", stderr)

	// Verify analysis output
	s.Assert().Contains(stdout, "AI Performance Analysis Results")
	s.Assert().Contains(stdout, "Status: Success")

	// Should provide performance metrics or insights
	analysisFound := strings.Contains(stdout, "Performance Metrics") ||
		strings.Contains(stdout, "Analysis Insights") ||
		strings.Contains(stdout, "Analysis Time")
	s.Assert().True(analysisFound, "Should provide performance analysis")
}

func (s *EnhancedCLIE2ESuite) TestAIInsightsCommandCLI() {
	// Create some tasks to generate insights about
	s.runCLI("add", "Data analysis task")
	s.runCLI("add", "Memory optimization task")

	// Test AI insights command
	stdout, stderr, err := s.runCLI("ai", "insights")
	s.Require().NoError(err, "stderr: %s", stderr)

	// Verify insights output
	s.Assert().Contains(stdout, "AI Memory Insights")

	// Should provide insights even if minimal
	insightsFound := strings.Contains(stdout, "No insights available") ||
		strings.Contains(stdout, "Memory Usage Analysis") ||
		strings.Contains(stdout, "Generated:")
	s.Assert().True(insightsFound, "Should provide memory insights or indicate none available")
}

// Test CLI-Server Integration with New Tool Architecture

func (s *EnhancedCLIE2ESuite) TestCLIServerToolMapping() {
	// Create a task which should trigger memory_create
	s.runCLI("add", "Test tool mapping task")

	// Wait for server interaction
	time.Sleep(500 * time.Millisecond)

	// Verify server received new tool calls
	operations := s.mockMCPServer.GetOperations()
	s.Assert().NotEmpty(operations, "Server should have received operations")

	// Should use new tool architecture
	hasNewTools := false
	for _, op := range operations {
		if op.Method == "memory_create" || op.Method == "memory_read" ||
			op.Method == "memory_update" || op.Method == "memory_analyze" {
			hasNewTools = true
			break
		}
	}
	s.Assert().True(hasNewTools, "Should use new 4-tool architecture")
}

func (s *EnhancedCLIE2ESuite) TestOfflineModeCLI() {
	// Stop server to test offline mode
	s.mockMCPServer.Stop()

	// CLI commands should still work locally
	stdout, stderr, err := s.runCLI("add", "Offline task")
	s.Require().NoError(err, "stderr: %s", stderr)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(stdout), &result)
	s.Require().NoError(err)

	s.Assert().Equal("Offline task", result["content"])

	// List should also work
	stdout, stderr, err = s.runCLI("list")
	s.Require().NoError(err, "stderr: %s", stderr)

	err = json.Unmarshal([]byte(stdout), &result)
	s.Require().NoError(err)

	tasks := result["tasks"].([]interface{})
	s.Assert().Len(tasks, 1)

	// Restart server
	s.Require().NoError(s.mockMCPServer.Start())
}

func (s *EnhancedCLIE2ESuite) TestConcurrentCLIOperations() {
	const numOperations = 5
	var wg sync.WaitGroup
	errors := make(chan error, numOperations)

	// Run concurrent task creation
	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, _, err := s.runCLI("add", fmt.Sprintf("Concurrent task %d", index))
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
	stdout, stderr, err := s.runCLI("list")
	s.Require().NoError(err, "stderr: %s", stderr)

	var result map[string]interface{}
	err = json.Unmarshal([]byte(stdout), &result)
	s.Require().NoError(err)

	tasks := result["tasks"].([]interface{})
	s.Assert().Len(tasks, numOperations)
}

// E2E Mock MCP Server

type MCPOperation struct {
	Method    string                 `json:"method"`
	Params    map[string]interface{} `json:"params"`
	Timestamp time.Time              `json:"timestamp"`
}

type E2EMockMCPServer struct {
	server     *httptest.Server
	operations []MCPOperation
	mutex      sync.RWMutex
	url        string
}

func NewE2EMockMCPServer() *E2EMockMCPServer {
	return &E2EMockMCPServer{
		operations: make([]MCPOperation, 0),
	}
}

func (m *E2EMockMCPServer) Start() error {
	m.server = httptest.NewServer(http.HandlerFunc(m.handleRequest))
	m.url = m.server.URL
	return nil
}

func (m *E2EMockMCPServer) Stop() {
	if m.server != nil {
		m.server.Close()
		m.server = nil
	}
}

func (m *E2EMockMCPServer) URL() string {
	return m.url
}

func (m *E2EMockMCPServer) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.operations = make([]MCPOperation, 0)
}

func (m *E2EMockMCPServer) GetOperations() []MCPOperation {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make([]MCPOperation, len(m.operations))
	copy(result, m.operations)
	return result
}

func (m *E2EMockMCPServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	var request mcp.MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Record operation
	m.recordOperation(request.Method, request.Params)

	response := mcp.MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
	}

	// Handle different methods
	switch request.Method {
	case "memory_create":
		response.Result = map[string]interface{}{
			"chunk_id": fmt.Sprintf("chunk_%d", time.Now().UnixNano()),
			"status":   "created",
		}
	case "memory_read":
		response.Result = map[string]interface{}{
			"chunks": []map[string]interface{}{
				{"id": "chunk_1", "content": "Sample content", "type": "task"},
			},
			"total": 1,
		}
	case "memory_update":
		response.Result = map[string]interface{}{
			"updated": true,
			"status":  "success",
		}
	case "memory_analyze":
		response.Result = map[string]interface{}{
			"analysis": map[string]interface{}{
				"patterns":    []string{"productivity_pattern"},
				"insights":    []string{"High productivity detected"},
				"suggestions": []string{"Continue current approach"},
			},
			"confidence": 0.85,
		}
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
		// Handle legacy methods or return success for any method
		response.Result = map[string]interface{}{"status": "success"}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (m *E2EMockMCPServer) recordOperation(method string, params interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	paramsMap := make(map[string]interface{})
	if params != nil {
		if pMap, ok := params.(map[string]interface{}); ok {
			paramsMap = pMap
		}
	}

	operation := MCPOperation{
		Method:    method,
		Params:    paramsMap,
		Timestamp: time.Now(),
	}

	m.operations = append(m.operations, operation)
}

// Test suite runner
func TestEnhancedCLIE2E(t *testing.T) {
	suite.Run(t, new(EnhancedCLIE2ESuite))
}
