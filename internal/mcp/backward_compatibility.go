// Package mcp provides Model Context Protocol server implementation and memory management.
// It includes MCP tool handlers, memory operations, and protocol compatibility layers.
package mcp

import (
	"context"
	"errors"
	"fmt"

	mcp "github.com/fredcamaral/gomcp-sdk"
)

// Constants for bulk operation types
const (
	bulkOperationStore  = "store"
	bulkOperationUpdate = "update"
	bulkOperationDelete = "delete"
)

// registerBackwardCompatibilityLayer registers compatibility wrappers for old tool names
// This allows existing MCP clients to continue using original tool names while internally
// routing to the new consolidated tools
func (ms *MemoryServer) registerBackwardCompatibilityLayer() {
	// Map original tool names to consolidated tool calls
	compatibilityMappings := []struct {
		originalName     string
		description      string
		consolidatedTool string
		operation        string
		scope            string
	}{
		// memory_create mappings
		{"mcp__memory__memory_store_chunk", "Store important conversation moments", "memory_create", "store_chunk", "single"},
		{"mcp__memory__memory_store_decision", "Store architectural/design decisions", "memory_create", "store_decision", "single"},
		{"mcp__memory__memory_create_thread", "Create memory thread from chunks", "memory_create", "create_thread", "single"},
		{"mcp__memory__memory_create_alias", "Create memory aliases", "memory_create", "create_alias", "single"},
		{"mcp__memory__memory_link", "Create relationship between chunks", "memory_create", "create_relationship", "single"},
		{"mcp__memory__memory_auto_detect_relationships", "Auto-detect relationships", "memory_create", "auto_detect_relationships", "single"},
		{"mcp__memory__memory_import_context", "Import conversation context", "memory_create", "import_context", "single"},
		{"mcp__memory__memory_bulk_import", "Import from various formats", "memory_create", "bulk_import", "bulk"},

		// memory_read mappings
		{"mcp__memory__memory_search", "Search past memories", "memory_read", "search", "single"},
		{"mcp__memory__memory_get_context", "Get project overview", "memory_read", "get_context", "single"},
		{"mcp__memory__memory_find_similar", "Find similar problems", "memory_read", "find_similar", "single"},
		{"mcp__memory__memory_get_patterns", "Get recurring patterns", "memory_read", "get_patterns", "single"},
		{"mcp__memory__memory_get_relationships", "Get relationships for chunk", "memory_read", "get_relationships", "single"},
		{"mcp__memory__memory_traverse_graph", "Traverse knowledge graph", "memory_read", "traverse_graph", "single"},
		{"mcp__memory__memory_get_threads", "Retrieve memory threads", "memory_read", "get_threads", "single"},
		{"mcp__memory__memory_search_explained", "Search with explanations", "memory_read", "search_explained", "single"},
		{"mcp__memory__memory_search_multi_repo", "Search across repositories", "memory_read", "search_multi_repo", "cross_repo"},
		{"mcp__memory__memory_resolve_alias", "Resolve alias references", "memory_read", "resolve_alias", "single"},
		{"mcp__memory__memory_list_aliases", "List aliases with filtering", "memory_read", "list_aliases", "single"},
		{"mcp__memory__memory_get_bulk_progress", "Get bulk operation progress", "memory_read", "get_bulk_progress", "bulk"},

		// memory_update mappings
		{"mcp__memory__memory_update_thread", "Update thread properties", "memory_update", "update_thread", "single"},
		{"mcp__memory__memory_update_relationship", "Update relationship metadata", "memory_update", "update_relationship", "single"},
		{"mcp__memory__memory_mark_refreshed", "Mark memory as refreshed", "memory_update", "mark_refreshed", "single"},
		{"mcp__memory__memory_resolve_conflicts", "Resolve memory conflicts", "memory_update", "resolve_conflicts", "single"},
		{"mcp__memory__memory_decay_management", "Manage memory decay", "memory_update", "decay_management", "single"},

		// memory_delete mappings
		{"mcp__memory__memory_bulk_operation_delete", "Bulk delete operations", "memory_delete", "bulk_delete", "bulk"},

		// memory_analyze mappings
		{"mcp__memory__memory_analyze_cross_repo_patterns", "Analyze cross-repo patterns", "memory_analyze", "cross_repo_patterns", "cross_repo"},
		{"mcp__memory__memory_find_similar_repositories", "Find similar repositories", "memory_analyze", "find_similar_repositories", "cross_repo"},
		{"mcp__memory__memory_get_cross_repo_insights", "Get cross-repo insights", "memory_analyze", "cross_repo_insights", "cross_repo"},
		{"mcp__memory__memory_conflicts", "Detect contradictory decisions", "memory_analyze", "detect_conflicts", "single"},
		{"mcp__memory__memory_health_dashboard", "Get health dashboard", "memory_analyze", "health_dashboard", "single"},
		{"mcp__memory__memory_check_freshness", "Check memory staleness", "memory_analyze", "check_freshness", "single"},
		{"mcp__memory__memory_detect_threads", "Auto-detect memory threads", "memory_analyze", "detect_threads", "single"},

		// memory_intelligence mappings
		{"mcp__memory__memory_suggest_related", "Get AI suggestions", "memory_intelligence", "suggest_related", "single"},

		// memory_transfer mappings
		{"mcp__memory__memory_export_project", "Export project memory data", "memory_transfer", "export_project", "project"},
		{"mcp__memory__memory_bulk_export", "Export with filtering", "memory_transfer", "bulk_export", "bulk"},
		{"mcp__memory__memory_continuity", "Get incomplete work", "memory_transfer", "continuity", "single"},

		// memory_system mappings
		{"mcp__memory__memory_health", "Basic health check", "memory_system", "health", "system"},
		{"mcp__memory__memory_status", "Comprehensive status", "memory_system", "status", "repository"},
		{"mcp__memory__memory_generate_citations", "Generate formatted citations", "memory_system", "generate_citations", "single"},
		{"mcp__memory__memory_create_inline_citation", "Create inline citations", "memory_system", "create_inline_citation", "single"},
	}

	// Register each compatibility wrapper
	for _, mapping := range compatibilityMappings {
		ms.registerCompatibilityWrapper(mapping.originalName, mapping.description, mapping.consolidatedTool, mapping.operation, mapping.scope)
	}

	// Special case for memory_bulk_operation which needs operation type routing
	ms.registerBulkOperationCompatibility()
}

// registerCompatibilityWrapper creates a wrapper tool that routes to consolidated tools
func (ms *MemoryServer) registerCompatibilityWrapper(originalName, description, consolidatedTool, operation, scope string) {
	ms.mcpServer.AddTool(mcp.NewTool(
		originalName,
		fmt.Sprintf("[LEGACY] %s - Use %s with operation='%s' instead", description, consolidatedTool, operation),
		mcp.ObjectSchema("Legacy tool parameters (will be passed as options)", map[string]interface{}{
			"additionalProperties": true,
		}, []string{}),
	), mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		// Route to appropriate consolidated tool
		consolidatedArgs := map[string]interface{}{
			"operation": operation,
			"scope":     scope,
			"options":   params,
		}

		switch consolidatedTool {
		case "memory_create":
			return ms.handleMemoryCreate(ctx, consolidatedArgs)
		case "memory_read":
			return ms.handleMemoryRead(ctx, consolidatedArgs)
		case "memory_update":
			return ms.handleMemoryUpdate(ctx, consolidatedArgs)
		case "memory_delete":
			return ms.handleMemoryDelete(ctx, consolidatedArgs)
		case "memory_analyze":
			return ms.handleMemoryAnalyze(ctx, consolidatedArgs)
		case "memory_intelligence":
			return ms.handleMemoryIntelligence(ctx, consolidatedArgs)
		case "memory_transfer":
			return ms.handleMemoryTransfer(ctx, consolidatedArgs)
		case "memory_system":
			return ms.handleMemorySystem(ctx, consolidatedArgs)
		default:
			return nil, fmt.Errorf("unknown consolidated tool: %s", consolidatedTool)
		}
	}))
}

// registerBulkOperationCompatibility handles the special case of memory_bulk_operation
// which routes to different consolidated tools based on the operation parameter
func (ms *MemoryServer) registerBulkOperationCompatibility() {
	ms.mcpServer.AddTool(mcp.NewTool(
		"mcp__memory__memory_bulk_operation",
		"[LEGACY] Execute bulk operations - Use memory_create, memory_update, or memory_delete instead",
		mcp.ObjectSchema("Bulk operation parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{bulkOperationStore, bulkOperationUpdate, bulkOperationDelete},
				"description": "Type of bulk operation to perform",
			},
			"additionalProperties": true,
		}, []string{"operation"}),
	), mcp.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
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
		case bulkOperationStore:
			consolidatedArgs := map[string]interface{}{
				"operation": "bulk_import",
				"scope":     "bulk",
				"options":   options,
			}
			return ms.handleMemoryCreate(ctx, consolidatedArgs)

		case bulkOperationUpdate:
			consolidatedArgs := map[string]interface{}{
				"operation": "bulk_update",
				"scope":     "bulk",
				"options":   options,
			}
			return ms.handleMemoryUpdate(ctx, consolidatedArgs)

		case bulkOperationDelete:
			consolidatedArgs := map[string]interface{}{
				"operation": "bulk_delete",
				"scope":     "bulk",
				"options":   options,
			}
			return ms.handleMemoryDelete(ctx, consolidatedArgs)

		default:
			return nil, fmt.Errorf("unsupported bulk operation type: %s", operationType)
		}
	}))
}
