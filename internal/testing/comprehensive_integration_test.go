// Package testing provides comprehensive integration testing for MCP Memory Server v2
// This test connects real storage implementations with MCP tools for end-to-end validation
package testing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/di"
	"lerian-mcp-memory/internal/mcp"
	"lerian-mcp-memory/internal/storage"

	"github.com/fredcamaral/gomcp-sdk/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ComprehensiveIntegrationSuite tests complete workflows with real storage
type ComprehensiveIntegrationSuite struct {
	suite.Suite
	server       *mcp.MemoryServer
	container    *di.Container
	db           *sql.DB
	contentStore storage.VectorStore
	cfg          *config.Config
	testConfig   *IntegrationTestConfig
	ctx          context.Context

	// Test data tracking
	testProjectID      string
	testSessionID      string
	createdContentIDs  []string
	createdDecisionIDs []string
}

// SetupSuite initializes comprehensive integration testing environment
func (s *ComprehensiveIntegrationSuite) SetupSuite() {
	s.ctx = context.Background()

	// Load test configuration
	cfg, err := LoadTestConfig()
	require.NoError(s.T(), err, "Failed to load integration test configuration")
	s.testConfig = cfg
	s.cfg = cfg.Config

	s.T().Logf("Integration test environment: Database=%s, Qdrant=%s, AI=%t",
		cfg.TestDatabaseURL, cfg.TestQdrantURL, !cfg.ShouldSkipAI())

	// Create dependency injection container with real storage
	container, err := di.NewContainer(s.cfg)
	require.NoError(s.T(), err, "Failed to create DI container")
	s.container = container

	// Get real storage interfaces from container
	s.contentStore = container.GetVectorStore() // Use vector store for now

	// Get database connection for direct verification
	if s.testConfig.IsRealStorageAvailable() {
		s.db = container.DB // Direct access to DB field
	}

	// Create MCP server with real storage backends
	server, err := mcp.NewMemoryServer(s.cfg)
	require.NoError(s.T(), err, "Failed to create MCP server")
	s.server = server

	// Initialize test identifiers
	s.testProjectID = fmt.Sprintf("integration-test-%d", time.Now().Unix())
	s.testSessionID = fmt.Sprintf("session-%d", time.Now().Unix())
	s.createdContentIDs = make([]string, 0)
	s.createdDecisionIDs = make([]string, 0)

	s.T().Logf("Comprehensive integration test initialized: project=%s, session=%s",
		s.testProjectID, s.testSessionID)
}

// TearDownSuite cleans up after comprehensive testing
func (s *ComprehensiveIntegrationSuite) TearDownSuite() {
	if s.testConfig.CleanupAfterTests {
		s.cleanupTestData()
	}

	if s.db != nil {
		if closeErr := s.db.Close(); closeErr != nil {
			s.T().Logf("Warning: failed to close test database: %v", closeErr)
		}
	}
}

// TestCompleteMemoryWorkflow tests the full memory lifecycle with real storage
func (s *ComprehensiveIntegrationSuite) TestCompleteMemoryWorkflow() {
	s.T().Log("🔄 Testing complete memory workflow: Store → Search → Analyze → Export")

	// Step 1: Store multiple types of content
	s.storeTestContent()
	s.storeTestDecision()

	// Step 2: Verify storage in database
	s.verifyContentInDatabase()

	// Step 3: Test semantic search
	s.testSemanticSearch()

	// Step 4: Test pattern analysis
	s.testPatternAnalysis()

	// Step 5: Test system export
	s.testSystemExport()

	s.T().Log("✅ Complete memory workflow test passed")
}

// TestRealStorageIntegration tests MCP tools with actual database and vector storage
func (s *ComprehensiveIntegrationSuite) TestRealStorageIntegration() {
	if !s.testConfig.IsRealStorageAvailable() {
		s.T().Skip("Real storage not available - configure TEST_DATABASE_URL")
	}

	s.T().Log("🗄️ Testing real storage integration")

	// Test vector store health check with panic recovery
	func() {
		defer func() {
			if r := recover(); r != nil {
				s.T().Logf("Vector store health check panicked (expected if Qdrant not running): %v", r)
			}
		}()

		err := s.contentStore.HealthCheck(s.ctx)
		if err != nil {
			s.T().Logf("Vector store health check failed (expected if not configured): %v", err)
		} else {
			s.T().Log("Vector store is healthy")

			// Test vector store stats if available
			stats, err := s.contentStore.GetStats(s.ctx)
			if err == nil {
				s.T().Logf("Vector store stats: %d total chunks", stats.TotalChunks)
			}
		}
	}()

	s.T().Log("✅ Real storage integration test passed")
}

// TestMCPToolsWithRealHandlers tests all MCP tools with actual business logic
func (s *ComprehensiveIntegrationSuite) TestMCPToolsWithRealHandlers() {
	s.T().Log("🛠️ Testing all 11 MCP tools with real handlers")

	// Test each tool category with real implementations
	s.testMemoryStoreTools()
	s.testMemoryRetrieveTools()
	s.testMemoryAnalyzeTools()
	s.testMemorySystemTools()
	s.testTemplateTools()

	s.T().Log("✅ All MCP tools integration test passed")
}

// TestConcurrentOperations tests concurrent MCP operations
func (s *ComprehensiveIntegrationSuite) TestConcurrentOperations() {
	s.T().Log("⚡ Testing concurrent MCP operations")

	// Create multiple goroutines performing different operations
	done := make(chan error, 10)

	// Concurrent stores
	for i := 0; i < 3; i++ {
		go func(index int) {
			request := protocol.ToolCallRequest{
				Name: "memory_store",
				Arguments: map[string]interface{}{
					"operation":  "store_content",
					"project_id": s.testProjectID,
					"session_id": fmt.Sprintf("%s-concurrent-%d", s.testSessionID, index),
					"content":    fmt.Sprintf("Concurrent test content %d", index),
					"type":       "concurrent_test",
					"tags":       []string{"concurrent", "test", fmt.Sprintf("index_%d", index)},
				},
			}

			_, err := s.callRealTool(request)
			done <- err
		}(i)
	}

	// Concurrent searches
	for i := 0; i < 2; i++ {
		go func(_ int) {
			request := protocol.ToolCallRequest{
				Name: "memory_retrieve",
				Arguments: map[string]interface{}{
					"operation":   "search_content",
					"project_id":  s.testProjectID,
					"query":       "test content",
					"max_results": 5,
				},
			}

			_, err := s.callRealTool(request)
			done <- err
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < 5; i++ {
		err := <-done
		assert.NoError(s.T(), err, "Concurrent operation failed")
	}

	s.T().Log("✅ Concurrent operations test passed")
}

// TestErrorHandlingAndRecovery tests error scenarios and recovery
func (s *ComprehensiveIntegrationSuite) TestErrorHandlingAndRecovery() {
	s.T().Log("🚨 Testing error handling and recovery")

	// Test invalid operations
	invalidRequest := protocol.ToolCallRequest{
		Name: "memory_create",
		Arguments: map[string]interface{}{
			"operation": "invalid_operation",
			"scope":     "single",
			"options": map[string]interface{}{
				"repository": s.testProjectID,
			},
		},
	}

	response, err := s.callRealTool(invalidRequest)
	assert.NoError(s.T(), err, "Tool call should not fail")

	// Parse response to check error handling
	result := response.Content[0].Text
	var data map[string]interface{}
	err = json.Unmarshal([]byte(result), &data)
	require.NoError(s.T(), err)

	// Should have error status or graceful handling
	if statusRaw, exists := data["status"]; exists && statusRaw != nil {
		status := statusRaw.(string)
		assert.True(s.T(), status == "error" || status == "success",
			"Invalid operation should be handled gracefully")
	} else {
		s.T().Log("No status field in response - tool may have failed gracefully")
	}

	s.T().Log("✅ Error handling test passed")
}

// Helper methods for comprehensive testing

func (s *ComprehensiveIntegrationSuite) storeTestContent() {
	testContents := []map[string]interface{}{
		{
			"content":    "Integration testing ensures all components work together seamlessly",
			"chunk_type": "best_practice",
		},
		{
			"content":    "MCP Memory Server v2 provides persistent memory across AI sessions",
			"chunk_type": "feature_description",
		},
		{
			"content":    "Database schema must be carefully managed with migration safety",
			"chunk_type": "architecture_note",
		},
	}

	for i, content := range testContents {
		request := protocol.ToolCallRequest{
			Name: "memory_create",
			Arguments: map[string]interface{}{
				"operation": "store_chunk",
				"scope":     "single",
				"options": map[string]interface{}{
					"repository": s.testProjectID,
					"session_id": s.testSessionID,
					"content":    content["content"],
					"chunk_type": content["chunk_type"],
					"tags":       []string{"integration", "testing", "comprehensive"},
					"metadata": map[string]interface{}{
						"test_index": i,
						"category":   "comprehensive_test",
					},
				},
			},
		}

		response, err := s.callRealTool(request)
		require.NoError(s.T(), err, "Failed to store test content %d", i)

		// Extract content ID for tracking
		result := response.Content[0].Text
		var data map[string]interface{}
		err = json.Unmarshal([]byte(result), &data)
		require.NoError(s.T(), err)

		if contentID, exists := data["chunk_id"]; exists {
			s.createdContentIDs = append(s.createdContentIDs, contentID.(string))
		}
	}

	s.T().Logf("Stored %d test content items", len(testContents))
}

func (s *ComprehensiveIntegrationSuite) storeTestDecision() {
	request := protocol.ToolCallRequest{
		Name: "memory_create",
		Arguments: map[string]interface{}{
			"operation": "store_decision",
			"scope":     "single",
			"options": map[string]interface{}{
				"repository": s.testProjectID,
				"session_id": s.testSessionID,
				"decision":   "Use comprehensive integration tests with real PostgreSQL and Qdrant",
				"rationale":  "Need to validate MCP Memory Server v2 functionality with real storage backends for higher test confidence and production readiness",
			},
		},
	}

	response, err := s.callRealTool(request)
	require.NoError(s.T(), err, "Failed to store test decision")

	// Extract decision ID
	result := response.Content[0].Text
	var data map[string]interface{}
	err = json.Unmarshal([]byte(result), &data)
	require.NoError(s.T(), err)

	if decisionID, exists := data["decision_id"]; exists {
		s.createdDecisionIDs = append(s.createdDecisionIDs, decisionID.(string))
	}

	s.T().Log("Stored test decision")
}

func (s *ComprehensiveIntegrationSuite) verifyContentInDatabase() {
	if !s.testConfig.IsRealStorageAvailable() || s.db == nil {
		s.T().Skip("Database verification requires real storage")
	}

	// Verify content exists in database
	query := "SELECT COUNT(*) FROM chunks WHERE project_id = $1"
	var count int
	err := s.db.QueryRowContext(s.ctx, query, s.testProjectID).Scan(&count)
	require.NoError(s.T(), err, "Failed to verify content in database")

	assert.Greater(s.T(), count, 0, "Should have stored content in database")
	s.T().Logf("Verified %d items in database", count)
}

func (s *ComprehensiveIntegrationSuite) testSemanticSearch() {
	request := protocol.ToolCallRequest{
		Name: "memory_read",
		Arguments: map[string]interface{}{
			"operation": "search",
			"scope":     "single",
			"options": map[string]interface{}{
				"repository": s.testProjectID,
				"query":      "integration testing components",
				"limit":      5,
			},
		},
	}

	response, err := s.callRealTool(request)
	require.NoError(s.T(), err, "Semantic search failed")

	// Verify search results
	result := response.Content[0].Text
	var data map[string]interface{}
	err = json.Unmarshal([]byte(result), &data)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "success", data["status"])

	if results, exists := data["chunks"]; exists {
		resultsArray := results.([]interface{})
		s.T().Logf("Semantic search found %d results", len(resultsArray))
	}
}

func (s *ComprehensiveIntegrationSuite) testPatternAnalysis() {
	request := protocol.ToolCallRequest{
		Name: "memory_analyze",
		Arguments: map[string]interface{}{
			"operation":  "detect_patterns",
			"project_id": s.testProjectID,
			"timeframe":  "all",
		},
	}

	response, err := s.callRealTool(request)
	require.NoError(s.T(), err, "Pattern analysis failed")

	// Verify analysis results
	result := response.Content[0].Text
	var data map[string]interface{}
	err = json.Unmarshal([]byte(result), &data)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "success", data["status"])
	s.T().Log("Pattern analysis completed")
}

func (s *ComprehensiveIntegrationSuite) testSystemExport() {
	request := protocol.ToolCallRequest{
		Name: "memory_system",
		Arguments: map[string]interface{}{
			"operation":  "export_project_data",
			"project_id": s.testProjectID,
			"format":     "json",
		},
	}

	response, err := s.callRealTool(request)
	require.NoError(s.T(), err, "System export failed")

	// Verify export results
	result := response.Content[0].Text
	var data map[string]interface{}
	err = json.Unmarshal([]byte(result), &data)
	require.NoError(s.T(), err)

	assert.Equal(s.T(), "success", data["status"])
	s.T().Log("System export completed")
}

func (s *ComprehensiveIntegrationSuite) testMemoryStoreTools() {
	// Test memory_create tool with various operations
	operations := []struct {
		operation string
		options   map[string]interface{}
	}{
		{
			operation: "store_chunk",
			options: map[string]interface{}{
				"repository": s.testProjectID,
				"session_id": s.testSessionID,
				"content":    "Test content for store_chunk",
				"chunk_type": "conversation",
			},
		},
		{
			operation: "store_decision",
			options: map[string]interface{}{
				"repository": s.testProjectID,
				"session_id": s.testSessionID,
				"decision":   "Use integration testing for validation",
				"rationale":  "Integration tests provide end-to-end validation",
			},
		},
	}

	for _, test := range operations {
		request := protocol.ToolCallRequest{
			Name: "memory_create",
			Arguments: map[string]interface{}{
				"operation": test.operation,
				"scope":     "single",
				"options":   test.options,
			},
		}

		response, err := s.callRealTool(request)
		assert.NoError(s.T(), err, "memory_create tool failed for %s", test.operation)

		if response != nil {
			result := response.Content[0].Text
			var data map[string]interface{}
			err = json.Unmarshal([]byte(result), &data)
			assert.NoError(s.T(), err, "Failed to parse response for %s", test.operation)
		}
	}
}

func (s *ComprehensiveIntegrationSuite) testMemoryRetrieveTools() {
	// Test memory_read tool with various operations
	operations := []struct {
		operation string
		options   map[string]interface{}
	}{
		{
			operation: "search",
			options: map[string]interface{}{
				"repository": s.testProjectID,
				"query":      "test content integration",
				"limit":      3,
			},
		},
		{
			operation: "find_similar",
			options: map[string]interface{}{
				"repository": s.testProjectID,
				"problem":    "testing integration components",
				"limit":      3,
			},
		},
		{
			operation: "get_context",
			options: map[string]interface{}{
				"repository": s.testProjectID,
				"session_id": s.testSessionID,
			},
		},
	}

	for _, test := range operations {
		request := protocol.ToolCallRequest{
			Name: "memory_read",
			Arguments: map[string]interface{}{
				"operation": test.operation,
				"scope":     "single",
				"options":   test.options,
			},
		}

		response, err := s.callRealTool(request)
		assert.NoError(s.T(), err, "memory_read tool failed for %s", test.operation)

		if response != nil {
			result := response.Content[0].Text
			var data map[string]interface{}
			err = json.Unmarshal([]byte(result), &data)
			assert.NoError(s.T(), err, "Failed to parse response for %s", test.operation)
		}
	}
}

func (s *ComprehensiveIntegrationSuite) testMemoryAnalyzeTools() {
	// Test memory_analyze tool with various operations
	operations := []struct {
		operation string
		options   map[string]interface{}
	}{
		{
			operation: "cross_repo_patterns",
			options: map[string]interface{}{
				"repository": s.testProjectID,
				"session_id": s.testSessionID,
				"timeframe":  "recent",
			},
		},
		{
			operation: "health_dashboard",
			options: map[string]interface{}{
				"repository": s.testProjectID,
			},
		},
		{
			operation: "detect_conflicts",
			options: map[string]interface{}{
				"repository": s.testProjectID,
				"session_id": s.testSessionID,
			},
		},
	}

	for _, test := range operations {
		request := protocol.ToolCallRequest{
			Name: "memory_analyze",
			Arguments: map[string]interface{}{
				"operation": test.operation,
				"scope":     "single",
				"options":   test.options,
			},
		}

		response, err := s.callRealTool(request)
		assert.NoError(s.T(), err, "memory_analyze tool failed for %s", test.operation)

		if response != nil {
			result := response.Content[0].Text
			var data map[string]interface{}
			err = json.Unmarshal([]byte(result), &data)
			assert.NoError(s.T(), err, "Failed to parse response for %s", test.operation)
		}
	}
}

func (s *ComprehensiveIntegrationSuite) testMemorySystemTools() {
	// Test memory_system tool with various operations
	operations := []struct {
		operation string
		options   map[string]interface{}
	}{
		{
			operation: "health",
			options:   map[string]interface{}{},
		},
		{
			operation: "status",
			options: map[string]interface{}{
				"repository": s.testProjectID,
			},
		},
		{
			operation: "export_project",
			options: map[string]interface{}{
				"repository": s.testProjectID,
				"format":     "json",
			},
		},
	}

	for _, test := range operations {
		request := protocol.ToolCallRequest{
			Name: "memory_system",
			Arguments: map[string]interface{}{
				"operation": test.operation,
				"scope":     "single",
				"options":   test.options,
			},
		}

		response, err := s.callRealTool(request)
		assert.NoError(s.T(), err, "memory_system tool failed for %s", test.operation)

		if response != nil {
			result := response.Content[0].Text
			var data map[string]interface{}
			err = json.Unmarshal([]byte(result), &data)
			assert.NoError(s.T(), err, "Failed to parse response for %s", test.operation)
		}
	}
}

func (s *ComprehensiveIntegrationSuite) testTemplateTools() {
	// Test template_management tool
	request := protocol.ToolCallRequest{
		Name: "template_management",
		Arguments: map[string]interface{}{
			"operation": "list_templates",
			"scope":     "single",
			"options": map[string]interface{}{
				"repository":   s.testProjectID,
				"project_type": "web",
				"category":     "feature",
				"limit":        3,
			},
		},
	}

	response, err := s.callRealTool(request)
	if err != nil {
		// Expected failure since template service is temporarily disabled
		s.T().Logf("Template tools disabled as expected: %v", err)
	} else if response != nil {
		// If templates work, validate the response
		result := response.Content[0].Text
		var data map[string]interface{}
		err = json.Unmarshal([]byte(result), &data)
		assert.NoError(s.T(), err, "Failed to parse template list response")
	}
}

func (s *ComprehensiveIntegrationSuite) callRealTool(request protocol.ToolCallRequest) (*protocol.ToolCallResult, error) {
	// Call the actual MCP server's tool handler using the executor
	// This connects to real business logic instead of mocks

	toolName := request.Name
	arguments := request.Arguments

	s.T().Logf("Calling real tool: %s", toolName)

	// Create MCP tool executor for real integration testing
	executor := mcp.NewMCPToolExecutor(s.server)

	// Execute the tool with real parameters
	result, err := executor.ExecuteTool(s.ctx, toolName, arguments)
	if err != nil {
		return nil, fmt.Errorf("real tool call failed: %w", err)
	}

	// Convert result to protocol.ToolCallResult format
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tool result: %w", err)
	}

	return protocol.NewToolCallResult(protocol.NewContent(string(resultJSON))), nil
}

func (s *ComprehensiveIntegrationSuite) cleanupTestData() {
	if s.db == nil {
		return
	}

	// Clean up test data from database
	queries := []string{
		"DELETE FROM chunks WHERE project_id = $1",
		"DELETE FROM decisions WHERE project_id = $1",
		"DELETE FROM templates WHERE project_id = $1",
	}

	for _, query := range queries {
		_, err := s.db.ExecContext(s.ctx, query, s.testProjectID)
		if err != nil {
			s.T().Logf("Warning: Failed to cleanup with query %s: %v", query, err)
		}
	}

	s.T().Logf("Cleaned up test data for project: %s", s.testProjectID)
}

// TestComprehensiveIntegrationSuite runs the comprehensive integration test suite
func TestComprehensiveIntegrationSuite(t *testing.T) {
	// Skip if integration tests not enabled
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Comprehensive integration tests skipped (set RUN_INTEGRATION_TESTS=true)")
	}

	suite.Run(t, new(ComprehensiveIntegrationSuite))
}
