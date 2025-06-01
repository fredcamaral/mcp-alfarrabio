package mcp

import (
	"context"
	"fmt"

	mcp "github.com/fredcamaral/gomcp-sdk"
)

// registerConsolidatedTools registers the 9 consolidated MCP tools
func (ms *MemoryServer) registerConsolidatedTools() {
	// 1. memory_create - All creation operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_create",
		"Handle all memory creation operations. REQUIRED fields vary by operation: store_chunk/store_decision require session_id; create_thread requires name+description+chunk_ids; create_relationship requires source_chunk_id+target_chunk_id+relation_type.",
		mcp.ObjectSchema("Memory creation parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type": "string",
				"enum": []string{
					OperationStoreChunk, OperationStoreDecision, "create_thread", "create_alias",
					"create_relationship", "auto_detect_relationships", "import_context", "bulk_import",
				},
				"description": "Type of creation operation to perform",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"single", "bulk"},
				"default":     "single",
				"description": "Operation scope",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Operation-specific parameters. REQUIRED fields vary: store_chunk/store_decision require session_id; create_thread requires name+description+chunk_ids; create_relationship requires source_chunk_id+target_chunk_id+relation_type",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session ID (required for store_chunk, store_decision, import_context)",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Content to store (required for store_chunk)",
					},
					"decision": map[string]interface{}{
						"type":        "string",
						"description": "Decision text (required for store_decision)",
					},
					"rationale": map[string]interface{}{
						"type":        "string",
						"description": "Decision rationale (required for store_decision)",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Thread name (required for create_thread)",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Thread description (required for create_thread)",
					},
					"chunk_ids": map[string]interface{}{
						"type":        "array",
						"description": "Array of chunk IDs (required for create_thread)",
						"items":       map[string]interface{}{"type": "string"},
					},
					"source_chunk_id": map[string]interface{}{
						"type":        "string",
						"description": "Source chunk ID (required for create_relationship)",
					},
					"target_chunk_id": map[string]interface{}{
						"type":        "string",
						"description": "Target chunk ID (required for create_relationship)",
					},
					"relation_type": map[string]interface{}{
						"type":        "string",
						"description": "Relationship type (required for create_relationship)",
					},
					"data": map[string]interface{}{
						"type":        "string",
						"description": "Data to import (required for import_context)",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (required for import_context) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc.",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryCreate))

	// 2. memory_read - All read/query operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_read",
		"Handle all memory read operations. REQUIRED fields: search requires query; get_context requires repository; find_similar requires problem; get_relationships requires chunk_id; search_multi_repo requires query+session_id.",
		mcp.ObjectSchema("Memory read parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type": "string",
				"enum": []string{
					"search", "get_context", "find_similar", "get_patterns", "get_relationships",
					"traverse_graph", "get_threads", "search_explained", "search_multi_repo",
					"resolve_alias", "list_aliases", "get_bulk_progress",
				},
				"description": "Type of read operation to perform",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"single", "cross_repo", "global"},
				"default":     "single",
				"description": "Search scope",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Operation-specific parameters. REQUIRED fields: search requires query; get_context requires repository; find_similar requires problem; get_relationships requires chunk_id; search_multi_repo requires query+session_id",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (required for search, search_multi_repo)",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (required for get_context) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc.",
					},
					"problem": map[string]interface{}{
						"type":        "string",
						"description": "Problem description (required for find_similar)",
					},
					"chunk_id": map[string]interface{}{
						"type":        "string",
						"description": "Chunk ID (required for get_relationships)",
					},
					"start_chunk_id": map[string]interface{}{
						"type":        "string",
						"description": "Starting chunk ID (required for traverse_graph)",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session ID (required for search_multi_repo)",
					},
					"alias_name": map[string]interface{}{
						"type":        "string",
						"description": "Alias name (required for resolve_alias)",
					},
					"operation_id": map[string]interface{}{
						"type":        "string",
						"description": "Operation ID (required for get_bulk_progress)",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryRead))

	// 3. memory_update - All update operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_update",
		"Handle all memory update operations including thread updates, relationship updates, refreshing memories, and conflict resolution.",
		mcp.ObjectSchema("Memory update parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type": "string",
				"enum": []string{
					"update_thread", "update_relationship", "mark_refreshed",
					"resolve_conflicts", "bulk_update", "decay_management",
				},
				"description": "Type of update operation to perform",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"single", "bulk"},
				"default":     "single",
				"description": "Update scope",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Operation-specific parameters. REQUIRED fields: update_thread requires thread_id; update_relationship requires relationship_id; mark_refreshed requires chunk_id+validation_notes; decay_management requires repository+session_id+action",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"thread_id": map[string]interface{}{
						"type":        "string",
						"description": "Thread ID (required for update_thread)",
					},
					"relationship_id": map[string]interface{}{
						"type":        "string",
						"description": "Relationship ID (required for update_relationship)",
					},
					"chunk_id": map[string]interface{}{
						"type":        "string",
						"description": "Chunk ID (required for mark_refreshed)",
					},
					"validation_notes": map[string]interface{}{
						"type":        "string",
						"description": "Validation notes (required for mark_refreshed)",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (required for decay_management) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc.",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session ID (required for decay_management)",
					},
					"action": map[string]interface{}{
						"type":        "string",
						"description": "Decay action (required for decay_management)",
					},
					"conflict_ids": map[string]interface{}{
						"type":        "array",
						"description": "Array of conflict IDs (required for resolve_conflicts)",
						"items":       map[string]interface{}{"type": "string"},
					},
					"chunks": map[string]interface{}{
						"type":        "array",
						"description": "Array of chunks to update (required for bulk_update)",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryUpdate))

	// 4. memory_delete - All deletion operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_delete",
		"Handle all memory deletion operations including bulk deletions and filtered deletions.",
		mcp.ObjectSchema("Memory delete parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"bulk_delete", "delete_expired", "delete_by_filter"},
				"description": "Type of deletion operation to perform",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"bulk", "filtered"},
				"default":     "bulk",
				"description": "Deletion scope",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Operation-specific parameters. REQUIRED fields: bulk_delete requires ids array",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"ids": map[string]interface{}{
						"type":        "array",
						"description": "Array of IDs to delete (required for bulk_delete)",
						"items":       map[string]interface{}{"type": "string"},
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryDelete))

	// 5. memory_analyze - All analysis operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_analyze",
		"Handle memory analysis operations. REQUIRED fields: health_dashboard requires repository+session_id; cross_repo_patterns requires session_id; find_similar_repositories requires repository+session_id.",
		mcp.ObjectSchema("Memory analysis parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type": "string",
				"enum": []string{
					"cross_repo_patterns", "find_similar_repositories", "cross_repo_insights",
					"detect_conflicts", "health_dashboard", "check_freshness", "detect_threads",
				},
				"description": "Type of analysis operation to perform",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"single", "cross_repo", "global"},
				"default":     "single",
				"description": "Analysis scope",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Operation-specific parameters. REQUIRED fields: health_dashboard requires repository+session_id; cross_repo_patterns requires session_id; find_similar_repositories requires repository+session_id",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (required for health_dashboard, find_similar_repositories, check_freshness, detect_threads) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc.",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session ID (required for health_dashboard, cross_repo_patterns, find_similar_repositories)",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryAnalyze))

	// 6. memory_intelligence - AI-powered operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_intelligence",
		"Handle AI-powered operations. REQUIRED fields: suggest_related requires current_context+session_id; auto_insights requires repository+session_id; pattern_prediction requires context+repository+session_id.",
		mcp.ObjectSchema("Memory intelligence parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"suggest_related", "auto_insights", "pattern_prediction"},
				"description": "Type of intelligence operation to perform",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"single", "cross_repo"},
				"default":     "single",
				"description": "Intelligence scope",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Operation-specific parameters. REQUIRED fields: suggest_related requires current_context+session_id; auto_insights requires repository+session_id; pattern_prediction requires context+repository+session_id",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"current_context": map[string]interface{}{
						"type":        "string",
						"description": "Current context (required for suggest_related)",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session ID (required for suggest_related, auto_insights, pattern_prediction)",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (required for auto_insights, pattern_prediction) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc.",
					},
					"context": map[string]interface{}{
						"type":        "string",
						"description": "Context for prediction (required for pattern_prediction)",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryIntelligence))

	// 7. memory_transfer - Data transfer operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_transfer",
		"Handle data transfer operations. REQUIRED fields: export_project requires repository+session_id; import_context requires data+repository+session_id; continuity requires repository.",
		mcp.ObjectSchema("Memory transfer parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"export_project", "bulk_export", "continuity", "import_context"},
				"description": "Type of transfer operation to perform",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"single", "bulk", "project"},
				"default":     "single",
				"description": "Transfer scope",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Operation-specific parameters. REQUIRED fields: export_project requires repository+session_id; import_context requires data+repository+session_id; continuity requires repository",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (required for export_project, import_context, continuity) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc.",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session ID (required for export_project, import_context)",
					},
					"data": map[string]interface{}{
						"type":        "string",
						"description": "Data to import (required for import_context)",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryTransfer))

	// 8. memory_tasks - Task and workflow management operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_tasks",
		"Handle task management and workflow tracking operations including todo management, session tracking, and workflow analysis.",
		mcp.ObjectSchema("Memory tasks parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type": "string",
				"enum": []string{
					"todo_write", "todo_read", "todo_update", "session_create", "session_end",
					"session_list", "workflow_analyze", "task_completion_stats",
				},
				"description": "Type of task operation to perform",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"session", "workflow", "global"},
				"default":     "session",
				"description": "Task operation scope",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Operation-specific parameters. REQUIRED fields: todo_write requires todos array; session_create requires session_id; session_end requires session_id; workflow_analyze requires session_id; todo_update requires tool_name",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"todos": map[string]interface{}{
						"type":        "array",
						"description": "Array of todo items (required for todo_write)",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session ID (required for session_create, session_end, workflow_analyze)",
					},
					"tool_name": map[string]interface{}{
						"type":        "string",
						"description": "Tool name (required for todo_update)",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryTasks))

	// 9. memory_system - System operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_system",
		"Handle system-level memory operations including health checks, status reports, and citation management.",
		mcp.ObjectSchema("Memory system parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{OperationHealth, OperationStatus, "generate_citations", "create_inline_citation", "get_documentation"},
				"description": "Type of system operation to perform",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"system", "repository"},
				"default":     "system",
				"description": "System operation scope",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Operation-specific parameters. REQUIRED fields: status requires repository; generate_citations requires query+chunk_ids; create_inline_citation requires text+response_id",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (required for status) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc.",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Query text (required for generate_citations)",
					},
					"chunk_ids": map[string]interface{}{
						"type":        "array",
						"description": "Array of chunk IDs (required for generate_citations)",
						"items":       map[string]interface{}{"type": "string"},
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Text content (required for create_inline_citation)",
					},
					"response_id": map[string]interface{}{
						"type":        "string",
						"description": "Response ID (required for create_inline_citation)",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemorySystem))
}

// Consolidated tool handlers

// handleMemoryCreate routes creation operations to appropriate handlers
func (ms *MemoryServer) handleMemoryCreate(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required")
	}

	switch operation {
	case OperationStoreChunk:
		return ms.handleStoreChunk(ctx, options)
	case OperationStoreDecision:
		return ms.handleStoreDecision(ctx, options)
	case "create_thread":
		return ms.handleCreateThread(ctx, options)
	case "create_alias":
		return ms.handleCreateAlias(ctx, options)
	case "create_relationship":
		return ms.handleMemoryLink(ctx, options)
	case "auto_detect_relationships":
		return ms.handleAutoDetectRelationships(ctx, options)
	case "import_context":
		return ms.handleImportContext(ctx, options)
	case "bulk_import":
		return ms.handleBulkImport(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported create operation: %s", operation)
	}
}

// handleMemoryRead routes read operations to appropriate handlers
func (ms *MemoryServer) handleMemoryRead(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required")
	}

	switch operation {
	case "search":
		return ms.handleSearch(ctx, options)
	case "get_context":
		return ms.handleGetContext(ctx, options)
	case "find_similar":
		return ms.handleFindSimilar(ctx, options)
	case "get_patterns":
		return ms.handleGetPatterns(ctx, options)
	case "get_relationships":
		return ms.handleGetRelationships(ctx, options)
	case "traverse_graph":
		return ms.handleTraverseGraph(ctx, options)
	case "get_threads":
		return ms.handleGetThreads(ctx, options)
	case "search_explained":
		return ms.handleSearchExplained(ctx, options)
	case "search_multi_repo":
		return ms.handleSearchMultiRepo(ctx, options)
	case "resolve_alias":
		return ms.handleResolveAlias(ctx, options)
	case "list_aliases":
		return ms.handleListAliases(ctx, options)
	case "get_bulk_progress":
		return ms.handleGetBulkProgress(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported read operation: %s", operation)
	}
}

// handleMemoryUpdate routes update operations to appropriate handlers
func (ms *MemoryServer) handleMemoryUpdate(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required")
	}

	switch operation {
	case "update_thread":
		return ms.handleUpdateThread(ctx, options)
	case "update_relationship":
		return ms.handleUpdateRelationship(ctx, options)
	case "mark_refreshed":
		return ms.handleMarkRefreshed(ctx, options)
	case "resolve_conflicts":
		return ms.handleMemoryResolveConflicts(ctx, options)
	case "bulk_update":
		// Create a modified options map for bulk update
		bulkOptions := make(map[string]interface{})
		for k, v := range options {
			bulkOptions[k] = v
		}
		bulkOptions["operation"] = "update"
		return ms.handleBulkOperation(ctx, bulkOptions)
	case "decay_management":
		return ms.handleMemoryDecayManagement(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported update operation: %s", operation)
	}
}

// handleMemoryDelete routes deletion operations to appropriate handlers
func (ms *MemoryServer) handleMemoryDelete(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required")
	}

	switch operation {
	case "bulk_delete":
		// Create a modified options map for bulk delete
		bulkOptions := make(map[string]interface{})
		for k, v := range options {
			bulkOptions[k] = v
		}
		bulkOptions["operation"] = "delete"
		return ms.handleBulkOperation(ctx, bulkOptions)
	case "delete_expired":
		// Future implementation
		return nil, fmt.Errorf("delete_expired not yet implemented")
	case "delete_by_filter":
		// Future implementation
		return nil, fmt.Errorf("delete_by_filter not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported delete operation: %s", operation)
	}
}

// handleMemoryAnalyze routes analysis operations to appropriate handlers
func (ms *MemoryServer) handleMemoryAnalyze(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required")
	}

	switch operation {
	case "cross_repo_patterns":
		return ms.handleAnalyzeCrossRepoPatterns(ctx, options)
	case "find_similar_repositories":
		return ms.handleFindSimilarRepositories(ctx, options)
	case "cross_repo_insights":
		return ms.handleGetCrossRepoInsights(ctx, options)
	case "detect_conflicts":
		return ms.handleMemoryConflicts(ctx, options)
	case "health_dashboard":
		return ms.handleMemoryHealthDashboard(ctx, options)
	case "check_freshness":
		return ms.handleCheckFreshness(ctx, options)
	case "detect_threads":
		return ms.handleDetectThreads(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported analyze operation: %s", operation)
	}
}

// handleMemoryIntelligence routes intelligence operations to appropriate handlers
func (ms *MemoryServer) handleMemoryIntelligence(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required")
	}

	switch operation {
	case "suggest_related":
		return ms.handleSuggestRelated(ctx, options)
	case "auto_insights":
		return ms.handleAutoInsights(ctx, options)
	case "pattern_prediction":
		return ms.handlePatternPrediction(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported intelligence operation: %s", operation)
	}
}

// handleMemoryTasks routes task management operations to appropriate handlers
func (ms *MemoryServer) handleMemoryTasks(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required")
	}

	switch operation {
	case "todo_write":
		return ms.handleTodoWrite(ctx, options)
	case "todo_read":
		return ms.handleTodoRead(ctx, options)
	case "todo_update":
		return ms.handleTodoUpdate(ctx, options)
	case "session_create":
		return ms.handleSessionCreate(ctx, options)
	case "session_end":
		return ms.handleSessionEnd(ctx, options)
	case "session_list":
		return ms.handleSessionList(ctx, options)
	case "workflow_analyze":
		return ms.handleWorkflowAnalyze(ctx, options)
	case "task_completion_stats":
		return ms.handleTaskCompletionStats(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported tasks operation: %s", operation)
	}
}

// handleMemoryTransfer routes transfer operations to appropriate handlers
func (ms *MemoryServer) handleMemoryTransfer(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required")
	}

	switch operation {
	case "export_project":
		return ms.handleExportProject(ctx, options)
	case "bulk_export":
		return ms.handleBulkExport(ctx, options)
	case "continuity":
		return ms.handleMemoryContinuity(ctx, options)
	case "import_context":
		return ms.handleImportContext(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported transfer operation: %s", operation)
	}
}

// handleMemorySystem routes system operations to appropriate handlers
func (ms *MemoryServer) handleMemorySystem(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		// For system operations like health, options might be empty
		options = make(map[string]interface{})
	}

	switch operation {
	case OperationHealth:
		return ms.handleHealth(ctx, options)
	case OperationStatus:
		return ms.handleMemoryStatus(ctx, options)
	case "generate_citations":
		return ms.handleGenerateCitations(ctx, options)
	case "create_inline_citation":
		return ms.handleCreateInlineCitation(ctx, options)
	case "get_documentation":
		return ms.handleGetDocumentation(ctx, options)
	default:
		return nil, fmt.Errorf("unsupported system operation: %s", operation)
	}
}
