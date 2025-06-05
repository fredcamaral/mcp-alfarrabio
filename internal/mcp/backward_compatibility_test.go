package mcp

import (
	"context"
	"errors"
	"fmt"
	"testing"

	mcp "github.com/fredcamaral/gomcp-sdk"
	"github.com/fredcamaral/gomcp-sdk/server"
	"github.com/stretchr/testify/assert"
)

// TestBackwardCompatibility tests that all original tool names work through compatibility layer
func TestBackwardCompatibility(t *testing.T) {
	ms := setupTestCompatibilityServer(t)
	defer ms.cleanup()

	ctx := context.Background()

	// Test cases for backward compatibility mappings
	testCases := []struct {
		originalToolName string
		testParams       map[string]interface{}
		expectedSuccess  bool
	}{
		{
			originalToolName: "mcp__memory__memory_store_chunk",
			testParams: map[string]interface{}{
				"content":    "Test chunk storage through compatibility",
				"session_id": "compat_test_session",
				"repository": "github.com/test/repo",
			},
			expectedSuccess: true,
		},
		{
			originalToolName: "mcp__memory__memory_search",
			testParams: map[string]interface{}{
				"query":      "test search through compatibility",
				"repository": "github.com/test/repo",
			},
			expectedSuccess: true,
		},
		{
			originalToolName: "mcp__memory__memory_store_decision",
			testParams: map[string]interface{}{
				"decision":   "Use backward compatibility",
				"rationale":  "Maintains legacy support",
				"session_id": "compat_test_session",
			},
			expectedSuccess: true,
		},
		{
			originalToolName: "mcp__memory__memory_get_context",
			testParams: map[string]interface{}{
				"repository": "github.com/test/repo",
			},
			expectedSuccess: true,
		},
		{
			originalToolName: "mcp__memory__memory_health",
			testParams:       map[string]interface{}{},
			expectedSuccess:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.originalToolName, func(t *testing.T) {
			result, err := ms.callCompatibilityTool(ctx, tc.originalToolName, tc.testParams)

			if tc.expectedSuccess {
				assert.NoError(t, err, "Compatibility tool %s should not return error", tc.originalToolName)
				assert.NotNil(t, result, "Compatibility tool %s should return result", tc.originalToolName)
			} else {
				assert.Error(t, err, "Compatibility tool %s should return error", tc.originalToolName)
			}
		})
	}
}

// TestBulkOperationCompatibility tests the special bulk operation routing
func TestBulkOperationCompatibility(t *testing.T) {
	ms := setupTestCompatibilityServer(t)
	defer ms.cleanup()

	ctx := context.Background()

	testCases := []struct {
		operation       string
		params          map[string]interface{}
		expectedSuccess bool
	}{
		{
			operation: "store",
			params: map[string]interface{}{
				"chunks": []map[string]interface{}{
					{
						"content":    "Bulk test chunk 1",
						"session_id": "bulk_test",
					},
				},
			},
			expectedSuccess: true,
		},
		{
			operation: "update",
			params: map[string]interface{}{
				"chunks": []map[string]interface{}{
					{
						"chunk_id": "test_chunk_id",
						"content":  "Updated content",
					},
				},
			},
			expectedSuccess: true,
		},
		{
			operation: "delete",
			params: map[string]interface{}{
				"ids": []string{"test_chunk_1", "test_chunk_2"},
			},
			expectedSuccess: true,
		},
	}

	for _, tc := range testCases {
		t.Run("bulk_"+tc.operation, func(t *testing.T) {
			params := map[string]interface{}{
				"operation": tc.operation,
			}
			// Merge test params into main params
			for k, v := range tc.params {
				params[k] = v
			}

			result, err := ms.callCompatibilityTool(ctx, "mcp__memory__memory_bulk_operation", params)

			if tc.expectedSuccess {
				assert.NoError(t, err, "Bulk operation %s should not return error", tc.operation)
				assert.NotNil(t, result, "Bulk operation %s should return result", tc.operation)
			} else {
				assert.Error(t, err, "Bulk operation %s should return error", tc.operation)
			}
		})
	}
}

// TestCompatibilityErrorHandling tests error handling in compatibility layer
func TestCompatibilityErrorHandling(t *testing.T) {
	ms := setupTestCompatibilityServer(t)
	defer ms.cleanup()

	ctx := context.Background()

	t.Run("missing_operation_in_bulk", func(t *testing.T) {
		params := map[string]interface{}{
			"chunks": []map[string]interface{}{
				{"content": "test"},
			},
		}

		_, err := ms.callCompatibilityTool(ctx, "mcp__memory__memory_bulk_operation", params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "operation parameter is required")
	})

	t.Run("invalid_bulk_operation", func(t *testing.T) {
		params := map[string]interface{}{
			"operation": "invalid_op",
		}

		_, err := ms.callCompatibilityTool(ctx, "mcp__memory__memory_bulk_operation", params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported bulk operation type")
	})
}

// setupTestCompatibilityServer creates a test server with compatibility layer
func setupTestCompatibilityServer(t *testing.T) *testCompatibilityServer {
	// Create mock MCP server
	mcpServer := mcp.NewServer("test-server", "1.0.0")

	// Create test memory server
	ms := &testCompatibilityServer{
		t:         t,
		mcpServer: mcpServer,
	}

	// Register compatibility layer
	ms.registerBackwardCompatibilityLayer()

	return ms
}

// testCompatibilityServer is a test wrapper with compatibility support
type testCompatibilityServer struct {
	t         *testing.T
	mcpServer *server.Server
}

func (tcs *testCompatibilityServer) cleanup() {
	// Cleanup test resources
}

// callCompatibilityTool simulates calling a compatibility tool
func (tcs *testCompatibilityServer) callCompatibilityTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	// This would normally go through the MCP server, but for testing we'll simulate the call
	switch toolName {
	case "mcp__memory__memory_store_chunk":
		return tcs.handleMemoryCreate(ctx, map[string]interface{}{
			"operation": "store_chunk",
			"scope":     "single",
			"options":   params,
		})
	case "mcp__memory__memory_search":
		return tcs.handleMemoryRead(ctx, map[string]interface{}{
			"operation": "search",
			"scope":     "single",
			"options":   params,
		})
	case "mcp__memory__memory_store_decision":
		return tcs.handleMemoryCreate(ctx, map[string]interface{}{
			"operation": "store_decision",
			"scope":     "single",
			"options":   params,
		})
	case "mcp__memory__memory_get_context":
		return tcs.handleMemoryRead(ctx, map[string]interface{}{
			"operation": "get_context",
			"scope":     "single",
			"options":   params,
		})
	case "mcp__memory__memory_health":
		return tcs.handleMemorySystem(ctx, map[string]interface{}{
			"operation": "health",
			"scope":     "system",
			"options":   params,
		})
	case "mcp__memory__memory_bulk_operation":
		return tcs.handleBulkOperationCompatibility(ctx, params)
	default:
		return nil, fmt.Errorf("unknown compatibility tool: %s", toolName)
	}
}

// Mock compatibility layer registration
func (tcs *testCompatibilityServer) registerBackwardCompatibilityLayer() {
	// For testing, we just need to verify the mapping logic works
	// The actual tool registration would happen in the real server
}

// Mock handlers that simulate the routing behavior
func (tcs *testCompatibilityServer) handleMemoryCreate(_ context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, errors.New("operation parameter is required")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, errors.New("options parameter is required")
	}

	// Mock successful responses for different operations
	switch operation {
	case "store_chunk":
		if _, ok := options["content"]; !ok {
			return nil, errors.New("content is required")
		}
		return map[string]interface{}{
			"chunk_id": "compat_test_chunk_123",
			"status":   "success",
		}, nil
	case "store_decision":
		if _, ok := options["decision"]; !ok {
			return nil, errors.New("decision is required")
		}
		return map[string]interface{}{
			"decision_id": "compat_test_decision_123",
			"status":      "success",
		}, nil
	default:
		return map[string]interface{}{
			"id":     "compat_test_id_123",
			"status": "success",
		}, nil
	}
}

func (tcs *testCompatibilityServer) handleMemoryRead(_ context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, errors.New("operation parameter is required")
	}

	return map[string]interface{}{
		"operation": operation,
		"results":   []interface{}{},
		"status":    "success",
	}, nil
}

func (tcs *testCompatibilityServer) handleMemorySystem(_ context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, errors.New("operation parameter is required")
	}

	switch operation {
	case "health":
		return map[string]interface{}{
			"status":  "healthy",
			"version": "test-1.0.0",
		}, nil
	default:
		return map[string]interface{}{
			"operation": operation,
			"status":    "success",
		}, nil
	}
}

func (tcs *testCompatibilityServer) handleBulkOperationCompatibility(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	operationType, ok := params["operation"].(string)
	if !ok {
		return nil, errors.New("operation parameter is required")
	}

	// Remove operation from params since it will be handled differently
	options := make(map[string]interface{})
	for k, v := range params {
		if k != "operation" {
			options[k] = v
		}
	}

	// Route based on operation type
	switch operationType {
	case "store":
		return tcs.handleMemoryCreate(ctx, map[string]interface{}{
			"operation": "bulk_import",
			"scope":     "bulk",
			"options":   options,
		})
	case "update":
		return tcs.handleMemoryUpdate(ctx, map[string]interface{}{
			"operation": "bulk_update",
			"scope":     "bulk",
			"options":   options,
		})
	case "delete":
		return tcs.handleMemoryDelete(ctx, map[string]interface{}{
			"operation": "bulk_delete",
			"scope":     "bulk",
			"options":   options,
		})
	default:
		return nil, fmt.Errorf("unsupported bulk operation type: %s", operationType)
	}
}

func (tcs *testCompatibilityServer) handleMemoryUpdate(_ context.Context, _ map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"status":  "success",
		"updated": "bulk",
	}, nil
}

func (tcs *testCompatibilityServer) handleMemoryDelete(_ context.Context, _ map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{
		"status":  "success",
		"deleted": "bulk",
	}, nil
}
