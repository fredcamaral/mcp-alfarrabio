package mcp

import (
	"context"
	"errors"
	"fmt"
	"mcp-memory/internal/logging"
	"strings"

	mcp "github.com/fredcamaral/gomcp-sdk"
)

// Constants for repository handling
const (
	GlobalRepository = "global"
)

// registerConsolidatedTools registers the 9 consolidated MCP tools
func (ms *MemoryServer) registerConsolidatedTools() {
	// 1. memory_create - All creation operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_create",
		"Handle all memory creation operations. CRITICAL: 'options' parameter MUST be a JSON object (not a JSON string). REQUIRED fields: repository is mandatory for ALL operations for multi-tenant isolation; store_chunk/store_decision require session_id+repository; create_thread requires name+description+chunk_ids+repository; create_relationship requires source_chunk_id+target_chunk_id+relation_type+repository. Use repository='global' for cross-project architecture decisions.",
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
				"description":          "Operation-specific parameters. REQUIRED fields: repository is mandatory for ALL operations; store_chunk/store_decision require session_id+repository; create_thread requires name+description+chunk_ids+repository; create_relationship requires source_chunk_id+target_chunk_id+relation_type+repository",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (REQUIRED for ALL operations for multi-tenant isolation) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc. Use 'global' for cross-project architecture decisions and knowledge.",
					},
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
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryCreate))

	// 2. memory_read - All read/query operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_read",
		"Handle all memory read operations. CRITICAL: 'options' parameter MUST be a JSON object (not a JSON string). REQUIRED fields: repository parameter is mandatory for ALL operations for multi-tenant isolation; search requires query+repository; get_context requires repository; find_similar requires problem+repository; get_relationships requires chunk_id+repository; search_multi_repo requires query+session_id+repository.",
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
				"description":          "Operation-specific parameters. REQUIRED fields: repository is mandatory for ALL operations for multi-tenant isolation; search requires query+repository; get_context requires repository; find_similar requires problem+repository; get_relationships requires chunk_id+repository; search_multi_repo requires query+session_id+repository",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (required for search, search_multi_repo)",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (REQUIRED for ALL operations for multi-tenant isolation) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc. Use 'global' for cross-project architecture decisions.",
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
		"Handle all memory update operations including thread updates, relationship updates, refreshing memories, and conflict resolution. CRITICAL: 'options' parameter MUST be a JSON object (not a JSON string). REQUIRED fields: repository is mandatory for ALL operations for multi-tenant isolation.",
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
				"description":          "Operation-specific parameters. REQUIRED fields: repository is mandatory for ALL operations; update_thread requires thread_id+repository; update_relationship requires relationship_id+repository; mark_refreshed requires chunk_id+validation_notes+repository; decay_management requires repository+session_id+action",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (REQUIRED for ALL operations for multi-tenant isolation) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc. Use 'global' for cross-project architecture updates.",
					},
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
		"Handle all memory deletion operations including bulk deletions and filtered deletions. CRITICAL: 'options' parameter MUST be a JSON object (not a JSON string). REQUIRED fields: repository parameter is mandatory for ALL operations to prevent cross-tenant data deletion.",
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
				"description":          "Operation-specific parameters. REQUIRED fields: repository is mandatory for ALL operations; bulk_delete requires ids array + repository",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"ids": map[string]interface{}{
						"type":        "array",
						"description": "Array of IDs to delete (required for bulk_delete)",
						"items":       map[string]interface{}{"type": "string"},
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (REQUIRED for ALL delete operations for security and multi-tenant isolation) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc.",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryDelete))

	// 5. memory_analyze - All analysis operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_analyze",
		"Handle memory analysis operations. CRITICAL: 'options' parameter MUST be a JSON object (not a JSON string). REQUIRED fields: repository is mandatory for ALL operations for multi-tenant isolation; health_dashboard requires repository+session_id; cross_repo_patterns requires session_id+repository; find_similar_repositories requires repository+session_id.",
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
				"description":          "Operation-specific parameters. REQUIRED fields: repository is mandatory for ALL operations for multi-tenant isolation; health_dashboard requires repository+session_id; cross_repo_patterns requires session_id+repository; find_similar_repositories requires repository+session_id",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (REQUIRED for ALL operations for multi-tenant isolation) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc. Use 'global' for cross-repository insights and architecture analysis.",
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
		"Handle AI-powered operations. CRITICAL: 'options' parameter MUST be a JSON object (not a JSON string). REQUIRED fields: repository is mandatory for ALL operations for multi-tenant isolation; suggest_related requires current_context+session_id+repository; auto_insights requires repository+session_id; pattern_prediction requires context+repository+session_id.",
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
				"description":          "Operation-specific parameters. REQUIRED fields: repository is mandatory for ALL operations for multi-tenant isolation; suggest_related requires current_context+session_id+repository; auto_insights requires repository+session_id; pattern_prediction requires context+repository+session_id",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (REQUIRED for ALL operations for multi-tenant isolation) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc. Use 'global' for cross-repository AI insights and architecture patterns.",
					},
					"current_context": map[string]interface{}{
						"type":        "string",
						"description": "Current context (required for suggest_related)",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session ID (required for suggest_related, auto_insights, pattern_prediction)",
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
		"Handle data transfer operations with pagination support. CRITICAL: 'options' parameter MUST be a JSON object (not a JSON string). REQUIRED fields: repository is mandatory for ALL operations for multi-tenant isolation; export_project requires repository+session_id (optional: limit, offset, format, include_vectors); import_context requires data+repository+session_id; continuity requires repository.",
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
				"description":          "Operation-specific parameters. REQUIRED fields: repository is mandatory for ALL operations for multi-tenant isolation; export_project requires repository+session_id; import_context requires data+repository+session_id; continuity requires repository",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (REQUIRED for ALL operations for multi-tenant isolation) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc. Use 'global' for cross-repository data transfer and architecture continuity.",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session ID (required for export_project, import_context)",
					},
					"data": map[string]interface{}{
						"type":        "string",
						"description": "Data to import (required for import_context)",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Page size for export_project (default: 100, max: 500) - Controls how many chunks to export per request",
						"minimum":     1,
						"maximum":     500,
						"default":     100,
					},
					"offset": map[string]interface{}{
						"type":        "number",
						"description": "Starting position for export_project pagination (default: 0) - Use with limit for paginated exports",
						"minimum":     0,
						"default":     0,
					},
					"format": map[string]interface{}{
						"type":        "string",
						"description": "Export format for export_project: 'json' (default), 'markdown', or 'archive'",
						"enum":        []string{"json", "markdown", "archive"},
						"default":     "json",
					},
					"include_vectors": map[string]interface{}{
						"type":        "boolean",
						"description": "Include embedding vectors in export_project output (default: false) - Warning: significantly increases response size",
						"default":     false,
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryTransfer))

	// 8. memory_tasks - Task and workflow management operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_tasks",
		"Handle task management and workflow tracking operations. CRITICAL: 'options' parameter MUST be a JSON object (not a JSON string). DECISION GUIDE for session_id: OMIT session_id for cross-session task continuity (RECOMMENDED - allows access to todos from previous conversations). INCLUDE session_id only when you need session-specific task isolation. BEHAVIORAL DIFFERENCE: Without session_id = repository-wide todos visible across all LLM sessions; With session_id = session-isolated todos.",
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
				"description":          "Operation-specific parameters. REQUIRED: repository for all operations. TODO OPERATIONS DECISION: For todo_write/todo_read/todo_update, OMIT session_id for cross-session continuity (recommended), INCLUDE session_id for session isolation. SESSION OPERATIONS: session_create, session_end, workflow_analyze require session_id.",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"todos": map[string]interface{}{
						"type":        "array",
						"description": "Array of todo items (required for todo_write)",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session ID - LLM DECISION GUIDE: OMIT for cross-session task continuity (RECOMMENDED - see todos from previous conversations). INCLUDE only for session-specific task isolation. BEHAVIOR: Without session_id = repository-wide todos across all sessions; With session_id = session-isolated todos. Required for session_create, session_end, workflow_analyze.",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (REQUIRED for ALL operations for multi-tenant isolation). Example: 'github.com/user/repo'",
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
		"Handle system-level memory operations including health checks, status reports, and citation management. CRITICAL: 'options' parameter MUST be a JSON object (not a JSON string). REQUIRED fields: repository parameter for status operations; health checks are global by default.",
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
				"description":          "Operation-specific parameters. REQUIRED fields: status requires repository; generate_citations requires query+chunk_ids+repository; create_inline_citation requires text+response_id; health checks are global by default",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (required for status and citation operations) - must include full URL like 'github.com/user/repo', 'gitlab.com/user/repo', etc. Optional for health checks (defaults to global system health).",
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
		return nil, fmt.Errorf("operation parameter is required. Example: {\"operation\": \"store_chunk\", \"options\": {\"content\": \"Bug fix summary\", \"session_id\": \"session-123\", \"repository\": \"github.com/user/repo\"}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for multi-tenant isolation. Example: {\"content\": \"text\", \"session_id\": \"session-123\", \"repository\": \"github.com/user/repo\"}")
	}

	// SECURITY: Repository parameter is MANDATORY for all create operations
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required for all create operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"content\": \"content-text\", \"session_id\": \"session-123\"} or use \"global\" for cross-project architecture decisions")
	}

	// Allow global storage for architecture decisions but log for security monitoring
	if repository == GlobalRepository {
		logging.Info("GLOBAL STORAGE: Cross-project create operation requested", "operation", operation, "options", options)
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
		validOps := []string{"store_chunk", "store_decision", "create_thread", "create_alias", "create_relationship", "auto_detect_relationships", "import_context", "bulk_import"}
		return nil, fmt.Errorf("unsupported create operation '%s'. Valid operations: %s. Example: {\"operation\": \"store_chunk\", \"options\": {\"repository\": \"github.com/user/repo\", \"content\": \"Fixed authentication bug\", \"session_id\": \"session-123\"}}", operation, strings.Join(validOps, ", "))
	}
}

// handleMemoryRead routes read operations to appropriate handlers
func (ms *MemoryServer) handleMemoryRead(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, options, err := ms.validateReadOperationParams(args)
	if err != nil {
		return nil, err
	}

	repository, err := ms.validateAndLogRepository(options, operation)
	if err != nil {
		return nil, err
	}

	return ms.routeReadOperation(ctx, operation, options, repository)
}

// validateReadOperationParams validates the basic parameters for read operations
func (ms *MemoryServer) validateReadOperationParams(args map[string]interface{}) (operation string, options map[string]interface{}, err error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return "", nil, fmt.Errorf("operation parameter is required. Example: {\"operation\": \"search\", \"options\": {\"query\": \"how to fix build errors\", \"repository\": \"github.com/user/repo\"}}")
	}

	options, ok = args["options"].(map[string]interface{})
	if !ok {
		return "", nil, fmt.Errorf("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for multi-tenant isolation. Example: {\"query\": \"search term\", \"repository\": \"github.com/user/repo\"}")
	}

	return operation, options, nil
}

// validateAndLogRepository validates the repository parameter and logs global access
func (ms *MemoryServer) validateAndLogRepository(options map[string]interface{}, operation string) (string, error) {
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return "", fmt.Errorf("repository parameter is required for all read operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"query\": \"search terms\"} or use \"global\" for cross-project architecture decisions")
	}

	if repository == GlobalRepository {
		logging.Info("GLOBAL ACCESS: Cross-project read operation requested", "operation", operation, "options", options)
	}

	return repository, nil
}

// routeReadOperation routes the read operation to the appropriate handler
func (ms *MemoryServer) routeReadOperation(ctx context.Context, operation string, options map[string]interface{}, repository string) (interface{}, error) {
	switch operation {
	case "search":
		return ms.handleSecureSearch(ctx, options, repository)
	case "get_context":
		return ms.handleGetContext(ctx, options)
	case "find_similar":
		return ms.handleSecureFindSimilar(ctx, options, repository)
	case "get_patterns":
		return ms.handleSecureGetPatterns(ctx, options, repository)
	case "get_relationships":
		return ms.handleSecureGetRelationships(ctx, options, repository)
	case "traverse_graph":
		return ms.handleSecureTraverseGraph(ctx, options, repository)
	case "get_threads":
		return ms.handleSecureGetThreads(ctx, options, repository)
	case "search_explained":
		return ms.handleSecureSearchExplained(ctx, options, repository)
	case "search_multi_repo":
		return ms.handleSearchMultiRepo(ctx, options)
	case "resolve_alias":
		return ms.handleSecureResolveAlias(ctx, options, repository)
	case "list_aliases":
		return ms.handleSecureListAliases(ctx, options, repository)
	case "get_bulk_progress":
		return ms.handleGetBulkProgress(ctx, options)
	default:
		return ms.buildUnsupportedOperationError(operation)
	}
}

// buildUnsupportedOperationError builds error message for unsupported operations
func (ms *MemoryServer) buildUnsupportedOperationError(operation string) (interface{}, error) {
	validOps := []string{"search", "get_context", "find_similar", "get_patterns", "get_relationships", "traverse_graph", "get_threads", "search_explained", "search_multi_repo", "resolve_alias", "list_aliases", "get_bulk_progress"}
	return nil, fmt.Errorf("unsupported read operation '%s'. Valid operations: %s. Example: {\"operation\": \"search\", \"options\": {\"repository\": \"github.com/user/repo\", \"query\": \"authentication issues\"}}", operation, strings.Join(validOps, ", "))
}

// handleMemoryUpdate routes update operations to appropriate handlers
func (ms *MemoryServer) handleMemoryUpdate(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, errors.New("operation parameter is required")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, errors.New("options parameter is required and MUST be a JSON object (not a JSON string)")
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
		return nil, fmt.Errorf("operation parameter is required. Example: {\"operation\": \"bulk_delete\", \"options\": {\"repository\": \"github.com/user/repo\", \"ids\": [\"chunk-id-1\"]}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for security. Example: {\"repository\": \"github.com/user/repo\", \"ids\": [\"chunk-id-1\", \"chunk-id-2\"]}")
	}

	// SECURITY: Repository parameter is MANDATORY for all delete operations
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required for all delete operations for multi-tenant security. Example: {\"repository\": \"github.com/user/repo\", \"ids\": [\"chunk-ids-to-delete\"]}")
	}

	switch operation {
	case "bulk_delete":
		return ms.handleSecureBulkDelete(ctx, options, repository)
	case "delete_expired":
		return ms.handleDeleteExpired(ctx, options, repository)
	case "delete_by_filter":
		// Future implementation with repository scoping
		return nil, fmt.Errorf("delete_by_filter operation not yet implemented. Alternative: Use memory_read with repository filter to search for matching chunks, then memory_delete with bulk_delete operation. Example: {\"operation\": \"bulk_delete\", \"options\": {\"repository\": \"github.com/user/repo\", \"ids\": [\"filtered-chunk-ids\"]}}")
	default:
		validOps := []string{"bulk_delete", "delete_expired", "delete_by_filter"}
		return nil, fmt.Errorf("unsupported delete operation '%s'. Valid operations: %s. Example: {\"operation\": \"bulk_delete\", \"options\": {\"repository\": \"github.com/user/repo\", \"ids\": [\"chunk-ids\"]}}", operation, strings.Join(validOps, ", "))
	}
}

// handleMemoryAnalyze routes analysis operations to appropriate handlers
func (ms *MemoryServer) handleMemoryAnalyze(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required. Example: {\"operation\": \"health_dashboard\", \"options\": {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}")
	}

	// SECURITY: Repository parameter is MANDATORY for all analyze operations
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required for all analyze operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"} or use \"global\" for cross-repository architecture analysis")
	}

	// Allow global analysis for cross-repository insights but log for security monitoring
	if repository == GlobalRepository {
		logging.Info("GLOBAL ANALYSIS: Cross-repository analyze operation requested", "operation", operation, "options", options)
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
		validOps := []string{"cross_repo_patterns", "find_similar_repositories", "cross_repo_insights", "detect_conflicts", "health_dashboard", "check_freshness", "detect_threads"}
		return nil, fmt.Errorf("unsupported analyze operation '%s'. Valid operations: %s. Example: {\"operation\": \"health_dashboard\", \"options\": {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}}", operation, strings.Join(validOps, ", "))
	}
}

// handleMemoryIntelligence routes intelligence operations to appropriate handlers
func (ms *MemoryServer) handleMemoryIntelligence(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required. Example: {\"operation\": \"auto_insights\", \"options\": {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}")
	}

	// SECURITY: Repository parameter is MANDATORY for all intelligence operations
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required for all intelligence operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"} or use \"global\" for cross-repository AI insights and architecture patterns")
	}

	// Allow global intelligence for cross-repository insights but log for security monitoring
	if repository == GlobalRepository {
		logging.Info("GLOBAL INTELLIGENCE: Cross-repository AI operation requested", "operation", operation, "options", options)
	}

	switch operation {
	case "suggest_related":
		return ms.handleSuggestRelated(ctx, options)
	case "auto_insights":
		return ms.handleAutoInsights(ctx, options)
	case "pattern_prediction":
		return ms.handlePatternPrediction(ctx, options)
	default:
		validOps := []string{"suggest_related", "auto_insights", "pattern_prediction"}
		return nil, fmt.Errorf("unsupported intelligence operation '%s'. Valid operations: %s. Example: {\"operation\": \"auto_insights\", \"options\": {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}}", operation, strings.Join(validOps, ", "))
	}
}

// handleMemoryTasks routes task management operations to appropriate handlers
func (ms *MemoryServer) handleMemoryTasks(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required. DECISION GUIDE: For todo operations, omit session_id for cross-session continuity (recommended). Example: {\"operation\": \"todo_write\", \"options\": {\"repository\": \"github.com/user/repo\", \"todos\": [...]}} or {\"operation\": \"todo_write\", \"options\": {\"session_id\": \"my-session\", \"repository\": \"github.com/user/repo\", \"todos\": [...]}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository. DECISION GUIDE: OMIT session_id for cross-session continuity (recommended), INCLUDE for session isolation. Example: {\"repository\": \"github.com/user/repo\"} or {\"session_id\": \"my-session\", \"repository\": \"github.com/user/repo\"}")
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
		return nil, fmt.Errorf("operation parameter is required. Example: {\"operation\": \"export_project\", \"options\": {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}")
	}

	// SECURITY: Repository parameter is MANDATORY for all transfer operations
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository parameter is required for all transfer operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"} or use \"global\" for cross-repository data transfer and architecture continuity")
	}

	// Allow global transfer for cross-repository continuity but log for security monitoring
	if repository == GlobalRepository {
		logging.Info("GLOBAL TRANSFER: Cross-repository transfer operation requested", "operation", operation, "options", options)
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
		validOps := []string{"export_project", "bulk_export", "continuity", "import_context"}
		return nil, fmt.Errorf("unsupported transfer operation '%s'. Valid operations: %s. Example: {\"operation\": \"export_project\", \"options\": {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}}", operation, strings.Join(validOps, ", "))
	}
}

// handleMemorySystem routes system operations to appropriate handlers
func (ms *MemoryServer) handleMemorySystem(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation parameter is required. Example: {\"operation\": \"health\"} or {\"operation\": \"status\", \"options\": {\"repository\": \"github.com/user/repo\"}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		// For system operations like health, options might be empty
		options = make(map[string]interface{})
	}

	// Repository parameter requirements vary by operation
	repository, hasRepo := options["repository"].(string)

	switch operation {
	case OperationHealth:
		// Health checks are global by default but can be repository-specific
		if hasRepo && repository != "" {
			logging.Info("Repository-specific health check requested", "repository", repository)
		} else {
			logging.Info("Global system health check requested")
		}
		return ms.handleHealth(ctx, options)
	case OperationStatus:
		// Status operations require repository for multi-tenant isolation
		if !hasRepo || repository == "" {
			return nil, fmt.Errorf("repository parameter is required for status operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\"}")
		}
		return ms.handleMemoryStatus(ctx, options)
	case "generate_citations":
		// Citations require repository for proper scoping
		if !hasRepo || repository == "" {
			return nil, fmt.Errorf("repository parameter is required for citation generation for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"query\": \"search terms\", \"chunk_ids\": [\"id1\", \"id2\"]}")
		}
		return ms.handleGenerateCitations(ctx, options)
	case "create_inline_citation":
		// Inline citations are repository-agnostic but logged for monitoring
		if hasRepo && repository != "" {
			logging.Info("Repository-specific inline citation requested", "repository", repository)
		}
		return ms.handleCreateInlineCitation(ctx, options)
	case "get_documentation":
		// Documentation is global by nature
		logging.Info("Global documentation request")
		return ms.handleGetDocumentation(ctx, options)
	default:
		validOps := []string{"health", "status", "generate_citations", "create_inline_citation", "get_documentation"}
		return nil, fmt.Errorf("unsupported system operation '%s'. Valid operations: %s. Example: {\"operation\": \"health\"} or {\"operation\": \"status\", \"options\": {\"repository\": \"github.com/user/repo\"}}", operation, strings.Join(validOps, ", "))
	}
}
