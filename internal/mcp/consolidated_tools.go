// Package mcp provides Model Context Protocol server implementation and memory management.
// It includes MCP tool handlers, memory operations, and protocol compatibility layers.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"lerian-mcp-memory/internal/logging"
	"lerian-mcp-memory/internal/templates"
	"lerian-mcp-memory/pkg/types"
	"strings"

	mcp "github.com/fredcamaral/gomcp-sdk"
)

// Constants moved to constants.go to avoid duplication

// registerConsolidatedTools registers the 9 consolidated MCP tools
func (ms *MemoryServer) registerConsolidatedTools() {
	// 1. memory_create - All creation operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_create",
		"Store and create new memory content, decisions, threads, and relationships. This tool handles ALL content creation operations in the memory system. USE THIS WHEN: You need to save conversation snippets, store architectural decisions, create knowledge threads, or link related content. PARAMETER GUIDE: Always include 'repository' (your project URL like 'github.com/user/repo' or 'global' for cross-project knowledge). For storing content, include 'session_id' to group related memories. EXAMPLES: Save bug fix → use 'store_chunk' operation; Save design decision → use 'store_decision' operation; Group related memories → use 'create_thread' operation.",
		mcp.ObjectSchema("Memory creation parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type": "string",
				"enum": []string{
					OperationStoreChunk, OperationStoreDecision, "create_thread", "create_alias",
					"create_relationship", "auto_detect_relationships", "import_context", "bulk_import",
				},
				"description": "What you want to create: 'store_chunk' = save conversation/code snippets; 'store_decision' = save architectural choices; 'create_thread' = group related memories; 'create_relationship' = link memories; 'import_context' = import external data; 'bulk_import' = create many items at once",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"single", "bulk"},
				"default":     "single",
				"description": "Single item or bulk operation",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Parameters for the creation operation. ALWAYS REQUIRED: 'repository' field. COMMONLY REQUIRED: 'session_id' field to group related work. See property descriptions for specific requirements per operation type.",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "🔒 REQUIRED: Your project identifier. Use full repository URL like 'github.com/user/repo' or 'global' for cross-project knowledge. This ensures your data is isolated and organized properly.",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "💡 REQUIRED for store_chunk/store_decision: Groups related work together. Use the same session_id for memories that belong to the same conversation or work session. Example: 'auth-feature-2024' or 'bug-fix-session-1'",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "📝 REQUIRED for store_chunk: The actual content to remember. Can be conversation snippets, code examples, problem solutions, or any text you want to save for later retrieval.",
					},
					"decision": map[string]interface{}{
						"type":        "string",
						"description": "🎯 REQUIRED for store_decision: The actual decision made. Example: 'Use PostgreSQL instead of MySQL for user data' or 'Implement JWT tokens for authentication'",
					},
					"rationale": map[string]interface{}{
						"type":        "string",
						"description": "🤔 REQUIRED for store_decision: Why this decision was made. Include factors considered, alternatives rejected, and reasoning. This helps future decisions and prevents repeated discussions.",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "🏷️ REQUIRED for create_thread: A descriptive name for the thread. Example: 'Authentication Implementation' or 'Database Design Decisions'",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "📋 REQUIRED for create_thread: What this thread is about. Explain the common theme that links these memories together.",
					},
					"chunk_ids": map[string]interface{}{
						"type":        "array",
						"description": "🔗 REQUIRED for create_thread: List of memory chunk IDs to group together. Get these IDs from previous store_chunk operations. Example: ['chunk-123', 'chunk-456']",
						"items":       map[string]interface{}{"type": "string"},
					},
					"source_chunk_id": map[string]interface{}{
						"type":        "string",
						"description": "🎯 REQUIRED for create_relationship: The ID of the first memory to connect. This is where the relationship starts from.",
					},
					"target_chunk_id": map[string]interface{}{
						"type":        "string",
						"description": "🎯 REQUIRED for create_relationship: The ID of the second memory to connect. This is where the relationship points to.",
					},
					"relation_type": map[string]interface{}{
						"type":        "string",
						"description": "🔄 REQUIRED for create_relationship: Type of connection. Examples: 'depends_on', 'follows_from', 'contradicts', 'supports', 'implements'",
					},
					"data": map[string]interface{}{
						"type":        "string",
						"description": "📦 REQUIRED for import_context: The external data to import. Can be JSON, text, or structured data from other systems.",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryCreate))

	// 2. memory_read - All read/query operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_read",
		"Search and retrieve stored memories, find similar content, and explore knowledge connections. This tool handles ALL memory retrieval operations. USE THIS WHEN: You want to find past conversations, search for similar problems, get project overview, or explore memory relationships. PARAMETER GUIDE: Always include 'repository' to specify which project to search. Use 'query' for text searches, 'problem' to find similar issues. EXAMPLES: Find past bug fixes → use 'search' operation; Find similar problems → use 'find_similar' operation; Get project overview → use 'get_context' operation; Explore connections → use 'get_relationships' operation.",
		mcp.ObjectSchema("Memory read parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type": "string",
				"enum": []string{
					"search", "get_context", "find_similar", "get_patterns", "get_relationships",
					"traverse_graph", "get_threads", "search_explained", "search_multi_repo",
					"resolve_alias", "list_aliases", "get_bulk_progress",
				},
				"description": "What you want to retrieve: 'search' = find memories by text; 'get_context' = project overview; 'find_similar' = find related problems; 'get_patterns' = discover recurring themes; 'get_relationships' = explore connections; 'get_threads' = grouped memories; 'search_multi_repo' = cross-project search",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"single", "cross_repo", "global"},
				"default":     "single",
				"description": "Where to search: 'single' = current project only; 'cross_repo' = across related projects; 'global' = all accessible knowledge",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Parameters for the search operation. ALWAYS REQUIRED: 'repository' field. OPERATION-SPECIFIC: See property descriptions for what each operation needs.",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "🔍 REQUIRED for search/search_multi_repo: What you're looking for. Use natural language. Examples: 'authentication bugs', 'database connection issues', 'React component patterns'",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "🔒 REQUIRED: Your project identifier. Use full repository URL like 'github.com/user/repo' or 'global' for cross-project search. This determines which memories to search through.",
					},
					"problem": map[string]interface{}{
						"type":        "string",
						"description": "🤔 REQUIRED for find_similar: Describe the problem you're facing. The system will find memories about similar issues you've encountered before. Example: 'Users can't log in after password reset'",
					},
					"chunk_id": map[string]interface{}{
						"type":        "string",
						"description": "🔗 REQUIRED for get_relationships: The ID of a memory chunk to explore connections from. Get this from previous search results or store operations.",
					},
					"start_chunk_id": map[string]interface{}{
						"type":        "string",
						"description": "🚀 REQUIRED for traverse_graph: Starting point for exploring the knowledge graph. The system will show you connected memories from this point.",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "💡 REQUIRED for search_multi_repo: Current session identifier to personalize cross-project search results based on your recent activity.",
					},
					"alias_name": map[string]interface{}{
						"type":        "string",
						"description": "🏷️ REQUIRED for resolve_alias: Name of the alias to look up. Aliases are shortcuts to frequently accessed memories or concepts.",
					},
					"operation_id": map[string]interface{}{
						"type":        "string",
						"description": "⏳ REQUIRED for get_bulk_progress: ID of a bulk operation to check status. Get this from previous bulk import/export operations.",
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
		"Analyze patterns, detect insights, and generate intelligence from stored memories. This tool uses AI to understand your knowledge base. USE THIS WHEN: You want to discover patterns across conversations, detect conflicting decisions, get health insights, or find similar repositories. PARAMETER GUIDE: Always include 'repository' to analyze specific projects. Include 'session_id' for personalized insights. EXAMPLES: Find recurring patterns → use 'cross_repo_patterns'; Check system health → use 'health_dashboard'; Detect conflicts → use 'detect_conflicts'; Get insights → use 'cross_repo_insights'.",
		mcp.ObjectSchema("Memory analysis parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type": "string",
				"enum": []string{
					"cross_repo_patterns", "find_similar_repositories", "cross_repo_insights",
					"detect_conflicts", "health_dashboard", "check_freshness", "detect_threads",
				},
				"description": "What you want to analyze: 'cross_repo_patterns' = find patterns across projects; 'find_similar_repositories' = discover related projects; 'detect_conflicts' = find contradictory decisions; 'health_dashboard' = system insights; 'check_freshness' = identify stale memories; 'detect_threads' = auto-group related memories",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"single", "cross_repo", "global"},
				"default":     "single",
				"description": "Analysis scope: 'single' = current project; 'cross_repo' = across related projects; 'global' = all accessible data",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Parameters for the analysis operation. ALWAYS REQUIRED: 'repository' field. COMMONLY REQUIRED: 'session_id' for personalized insights. See property descriptions for specific requirements.",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "🔒 REQUIRED: Your project identifier. Use full repository URL like 'github.com/user/repo' or 'global' for cross-repository analysis. This determines which data to analyze.",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "💡 REQUIRED for health_dashboard/cross_repo_patterns/find_similar_repositories: Current session to personalize analysis results based on your recent activity and context.",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryAnalyze))

	// 6. memory_intelligence - AI-powered operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_intelligence",
		"Get AI-powered suggestions and intelligent insights from your memory data. This tool uses advanced AI to provide smart recommendations and predictions. USE THIS WHEN: You want AI to suggest related content, generate automatic insights, or predict patterns based on current context. PARAMETER GUIDE: Always include 'repository' and 'session_id' for personalized AI responses. Provide 'current_context' to get relevant suggestions. EXAMPLES: Get related suggestions → use 'suggest_related'; Get AI insights → use 'auto_insights'; Predict patterns → use 'pattern_prediction'.",
		mcp.ObjectSchema("Memory intelligence parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"suggest_related", "auto_insights", "pattern_prediction"},
				"description": "What AI intelligence you want: 'suggest_related' = get AI suggestions for related content; 'auto_insights' = generate automatic insights from your data; 'pattern_prediction' = AI predictions based on patterns",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"single", "cross_repo"},
				"default":     "single",
				"description": "Intelligence scope: 'single' = current project only; 'cross_repo' = insights across related projects",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Parameters for AI intelligence operations. ALWAYS REQUIRED: 'repository' and 'session_id' for personalized AI responses. OPERATION-SPECIFIC: See property descriptions for additional requirements.",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "🔒 REQUIRED: Your project identifier. Use full repository URL like 'github.com/user/repo' or 'global' for cross-repository AI insights. This determines the knowledge base for AI analysis.",
					},
					"current_context": map[string]interface{}{
						"type":        "string",
						"description": "📍 REQUIRED for suggest_related: Describe what you're currently working on. The AI will suggest related memories and insights. Example: 'Working on user authentication feature'",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "💡 REQUIRED for ALL operations: Current session to personalize AI responses based on your recent activity and learning patterns.",
					},
					"context": map[string]interface{}{
						"type":        "string",
						"description": "🔮 REQUIRED for pattern_prediction: Context for AI predictions. Describe the situation and the AI will predict likely patterns or outcomes. Example: 'Planning new microservice architecture'",
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
		"Manage tasks, todos, and workflow tracking across your projects. This tool handles task lifecycle management and workflow analysis. USE THIS WHEN: You need to create, read, or update tasks and todos, manage work sessions, or analyze workflow patterns. PARAMETER GUIDE: Always include 'repository'. For todos: OMIT 'session_id' to see all project tasks (recommended), INCLUDE 'session_id' for session-specific isolation. EXAMPLES: Create todos → use 'todo_write'; Read current tasks → use 'todo_read'; Update task status → use 'todo_update'; Analyze workflow → use 'workflow_analyze'.",
		mcp.ObjectSchema("Memory tasks parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type": "string",
				"enum": []string{
					"todo_write", "todo_read", "todo_update", "session_create", "session_end",
					"session_list", "workflow_analyze", "task_completion_stats",
				},
				"description": "What task operation you need: 'todo_write' = create new tasks; 'todo_read' = get current tasks; 'todo_update' = modify task status; 'session_create' = start work session; 'session_end' = finish session; 'workflow_analyze' = analyze patterns",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"session", "workflow", "global"},
				"default":     "session",
				"description": "Task scope: 'session' = current session tasks; 'workflow' = project workflow; 'global' = cross-project tasks",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Parameters for task operations. ALWAYS REQUIRED: 'repository'. TASK VISIBILITY CHOICE: OMIT 'session_id' to see all project tasks (recommended for continuity), INCLUDE 'session_id' for session isolation.",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"todos": map[string]interface{}{
						"type":        "array",
						"description": "✅ REQUIRED for todo_write: List of task objects to create. Each should have content, status, and priority. Example: [{'content': 'Fix login bug', 'status': 'pending', 'priority': 'high'}]",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "🔄 CHOICE for todos: OMIT to see ALL project tasks across sessions (RECOMMENDED for continuity). INCLUDE for session-specific task isolation. REQUIRED for session operations (create/end/analyze). BEHAVIOR: No session_id = all project tasks; With session_id = session-only tasks.",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "🔒 REQUIRED: Your project identifier. Use full repository URL like 'github.com/user/repo'. This determines which project's tasks to manage.",
					},
					"tool_name": map[string]interface{}{
						"type":        "string",
						"description": "🔧 REQUIRED for todo_update: Name of the tool or feature the task relates to. Used for organizing and filtering task updates.",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemoryTasks))

	// 9. memory_system - System operations
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_system",
		"System administration, health monitoring, and utility operations for the memory server. This tool handles system-level operations and administrative tasks. USE THIS WHEN: You need to check system health, get status reports, export data, generate citations, or manage system operations. PARAMETER GUIDE: 'health' checks work globally, 'status' requires repository, citations need repository and content IDs. EXAMPLES: Check if system is working → use 'health'; Get project status → use 'status'; Export project data → use 'export_project'; Create citations → use 'generate_citations'.",
		mcp.ObjectSchema("Memory system parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{OperationHealth, OperationStatus, "generate_citations", "create_inline_citation", "get_documentation"},
				"description": "What system operation you need: 'health' = check if system is working; 'status' = get project status report; 'generate_citations' = create formatted citations; 'create_inline_citation' = create inline references; 'get_documentation' = access system docs",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"system", "repository"},
				"default":     "system",
				"description": "Operation scope: 'system' = global system operations; 'repository' = project-specific operations",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Parameters for system operations. REQUIREMENTS VARY: 'health' needs no params; 'status' needs repository; citations need repository and content details. See property descriptions for specifics.",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "🔒 REQUIRED for status/citations: Your project identifier. Use full repository URL like 'github.com/user/repo'. Not needed for health checks which are global.",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "🔍 REQUIRED for generate_citations: The original query or search that led to the content you want to cite. This provides context for the citation.",
					},
					"chunk_ids": map[string]interface{}{
						"type":        "array",
						"description": "📚 REQUIRED for generate_citations: List of memory chunk IDs to create citations for. Get these from search results. Example: ['chunk-123', 'chunk-456']",
						"items":       map[string]interface{}{"type": "string"},
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "📝 REQUIRED for create_inline_citation: The text content that you want to create an inline citation for. This will be formatted with proper references.",
					},
					"response_id": map[string]interface{}{
						"type":        "string",
						"description": "🆔 REQUIRED for create_inline_citation: Unique identifier for this response. Used to link citations back to specific conversations or contexts.",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleMemorySystem))

	// 10. document_generation - AI-powered document generation
	ms.mcpServer.AddTool(mcp.NewTool(
		"document_generation",
		"Generate documents using AI including PRDs, TRDs, and tasks. Interactive sessions supported for step-by-step PRD creation. Requires repository for context.",
		mcp.ObjectSchema("Document generation parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"generate_prd", "generate_trd", "generate_main_tasks", "generate_sub_tasks", "start_session", "continue_session", "end_session"},
				"description": "Type of document generation operation to perform",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Operation-specific parameters. REQUIRED: repository for all operations",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository URL (REQUIRED for all operations)",
					},
					"user_inputs": map[string]interface{}{
						"type":        "array",
						"description": "User input strings (required for generate_prd)",
						"items":       map[string]interface{}{"type": "string"},
					},
					"project_type": map[string]interface{}{
						"type":        "string",
						"description": "Type of project (api, web-app, cli, mobile, library, general)",
					},
					"prd_content": map[string]interface{}{
						"type":        "string",
						"description": "PRD content (required for generate_trd)",
					},
					"trd_content": map[string]interface{}{
						"type":        "string",
						"description": "TRD content (required for generate_main_tasks)",
					},
					"main_task_content": map[string]interface{}{
						"type":        "string",
						"description": "Main task content (required for generate_sub_tasks)",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session ID (required for continue_session, end_session)",
					},
					"user_input": map[string]interface{}{
						"type":        "string",
						"description": "User response (for continue_session)",
					},
					"doc_type": map[string]interface{}{
						"type":        "string",
						"description": "Document type (required for start_session): prd, trd, tasks",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleDocumentGeneration))

	// 11. template_management - Template-based task generation
	ms.mcpServer.AddTool(mcp.NewTool(
		"template_management",
		"Manage and instantiate task templates to streamline development workflows. This tool provides access to built-in templates for common development tasks and allows you to generate structured task lists. USE THIS WHEN: You need to start a new feature, fix bugs, set up projects, or follow standard development workflows. PARAMETER GUIDE: Always include 'repository' for context. Use 'list_templates' to explore available templates, 'get_template' for details, 'instantiate_template' to generate tasks. EXAMPLES: See available templates → use 'list_templates'; Get template details → use 'get_template'; Create tasks from template → use 'instantiate_template'.",
		mcp.ObjectSchema("Template management parameters", map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"list_templates", "get_template", "instantiate_template"},
				"description": "What template operation you need: 'list_templates' = browse available templates with filtering; 'get_template' = get detailed template information; 'instantiate_template' = generate tasks from a template",
			},
			"scope": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"builtin", "project", "global"},
				"default":     "builtin",
				"description": "Template scope: 'builtin' = system templates; 'project' = project-specific templates; 'global' = cross-project templates",
			},
			"options": map[string]interface{}{
				"type":                 "object",
				"description":          "Parameters for template operations. ALWAYS REQUIRED: 'repository' for context. OPERATION-SPECIFIC: See property descriptions for additional requirements per operation type.",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "🔒 REQUIRED: Your project identifier. Use full repository URL like 'github.com/user/repo' or 'global' for cross-project templates. This ensures templates are contextually relevant.",
					},
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "📋 REQUIRED for instantiate_template: Project identifier where tasks will be created. Usually same as repository but can be different for sub-projects.",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "🔄 OPTIONAL: Session identifier for grouping generated tasks. Use same session_id for related work. Example: 'auth-feature-impl' or 'bug-fix-session'",
					},
					"template_id": map[string]interface{}{
						"type":        "string",
						"description": "🎯 REQUIRED for get_template/instantiate_template: ID of the template to retrieve or instantiate. Get valid IDs from list_templates operation.",
					},
					"variables": map[string]interface{}{
						"type":                 "object",
						"description":          "🔧 REQUIRED for instantiate_template: Variables to substitute in the template. Each template has specific required variables. Check template details first with get_template.",
						"additionalProperties": true,
					},
					"project_type": map[string]interface{}{
						"type":        "string",
						"description": "🏗️ OPTIONAL for list_templates: Filter by project type. Options: 'web', 'api', 'backend', 'frontend', 'mobile', 'desktop', 'library', 'cli', 'any'",
						"enum":        []string{"web", "api", "backend", "frontend", "mobile", "desktop", "library", "cli", "any"},
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "📂 OPTIONAL for list_templates: Filter by template category. Options: 'feature', 'api', 'maintenance', 'testing', 'documentation', 'deployment', 'security', 'optimization', 'refactoring', 'infrastructure'",
						"enum":        []string{"feature", "api", "maintenance", "testing", "documentation", "deployment", "security", "optimization", "refactoring", "infrastructure"},
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"description": "🏷️ OPTIONAL for list_templates: Filter by tags. Example: ['web', 'api', 'authentication'] to find templates related to web API authentication.",
						"items":       map[string]interface{}{"type": "string"},
					},
					"popular_only": map[string]interface{}{
						"type":        "boolean",
						"description": "⭐ OPTIONAL for list_templates: Show only popular templates based on usage statistics. Default: false",
						"default":     false,
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "📊 OPTIONAL for list_templates: Maximum number of templates to return. Default: 20, Max: 100",
						"default":     20,
						"minimum":     1,
						"maximum":     100,
					},
					"metadata": map[string]interface{}{
						"type":                 "object",
						"description":          "📝 OPTIONAL for instantiate_template: Additional metadata to attach to generated tasks. Useful for tracking or categorization.",
						"additionalProperties": true,
					},
					"prefix": map[string]interface{}{
						"type":        "string",
						"description": "🔖 OPTIONAL for instantiate_template: Prefix to add to task names. Example: 'Sprint 1: ' or 'Auth Feature: '",
					},
				},
			},
		}, []string{"operation", "options"}),
	), mcp.ToolHandlerFunc(ms.handleTemplateManagement))
}

// Consolidated tool handlers

// handleMemoryCreate routes creation operations to appropriate handlers
func (ms *MemoryServer) handleMemoryCreate(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, errors.New("operation parameter is required. Example: {\"operation\": \"store_chunk\", \"options\": {\"content\": \"Bug fix summary\", \"session_id\": \"session-123\", \"repository\": \"github.com/user/repo\"}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, errors.New("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for multi-tenant isolation. Example: {\"content\": \"text\", \"session_id\": \"session-123\", \"repository\": \"github.com/user/repo\"}")
	}

	// SECURITY: Repository parameter is MANDATORY for all create operations
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, errors.New("repository parameter is required for all create operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"content\": \"content-text\", \"session_id\": \"session-123\"} or use \"global\" for cross-project architecture decisions")
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
		return "", nil, errors.New("operation parameter is required. Example: {\"operation\": \"search\", \"options\": {\"query\": \"how to fix build errors\", \"repository\": \"github.com/user/repo\"}}")
	}

	options, ok = args["options"].(map[string]interface{})
	if !ok {
		return "", nil, errors.New("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for multi-tenant isolation. Example: {\"query\": \"search term\", \"repository\": \"github.com/user/repo\"}")
	}

	return operation, options, nil
}

// validateAndLogRepository validates the repository parameter and logs global access
func (ms *MemoryServer) validateAndLogRepository(options map[string]interface{}, operation string) (string, error) {
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return "", errors.New("repository parameter is required for all read operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"query\": \"search terms\"} or use \"global\" for cross-project architecture decisions")
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
		return map[string]interface{}{
			"status":  "not_implemented",
			"message": "Advanced conflict resolution temporarily disabled - use memory_conflicts for basic conflict detection",
		}, nil
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
		return nil, errors.New("operation parameter is required. Example: {\"operation\": \"bulk_delete\", \"options\": {\"repository\": \"github.com/user/repo\", \"ids\": [\"chunk-id-1\"]}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, errors.New("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for security. Example: {\"repository\": \"github.com/user/repo\", \"ids\": [\"chunk-id-1\", \"chunk-id-2\"]}")
	}

	// SECURITY: Repository parameter is MANDATORY for all delete operations
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, errors.New("repository parameter is required for all delete operations for multi-tenant security. Example: {\"repository\": \"github.com/user/repo\", \"ids\": [\"chunk-ids-to-delete\"]}")
	}

	switch operation {
	case "bulk_delete":
		return ms.handleSecureBulkDelete(ctx, options, repository)
	case "delete_expired":
		return ms.handleDeleteExpired(ctx, options, repository)
	case "delete_by_filter":
		// Future implementation with repository scoping
		return nil, errors.New("delete_by_filter operation not yet implemented. Alternative: Use memory_read with repository filter to search for matching chunks, then memory_delete with bulk_delete operation. Example: {\"operation\": \"bulk_delete\", \"options\": {\"repository\": \"github.com/user/repo\", \"ids\": [\"filtered-chunk-ids\"]}}")
	default:
		validOps := []string{"bulk_delete", "delete_expired", "delete_by_filter"}
		return nil, fmt.Errorf("unsupported delete operation '%s'. Valid operations: %s. Example: {\"operation\": \"bulk_delete\", \"options\": {\"repository\": \"github.com/user/repo\", \"ids\": [\"chunk-ids\"]}}", operation, strings.Join(validOps, ", "))
	}
}

// handleMemoryAnalyze routes analysis operations to appropriate handlers
func (ms *MemoryServer) handleMemoryAnalyze(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, errors.New("operation parameter is required. Example: {\"operation\": \"health_dashboard\", \"options\": {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, errors.New("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}")
	}

	// SECURITY: Repository parameter is MANDATORY for all analyze operations
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, errors.New("repository parameter is required for all analyze operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"} or use \"global\" for cross-repository architecture analysis")
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
		return nil, errors.New("operation parameter is required. Example: {\"operation\": \"auto_insights\", \"options\": {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, errors.New("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}")
	}

	// SECURITY: Repository parameter is MANDATORY for all intelligence operations
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, errors.New("repository parameter is required for all intelligence operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"} or use \"global\" for cross-repository AI insights and architecture patterns")
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
		return nil, errors.New("operation parameter is required. DECISION GUIDE: For todo operations, omit session_id for cross-session continuity (recommended). Example: {\"operation\": \"todo_write\", \"options\": {\"repository\": \"github.com/user/repo\", \"todos\": [...]}} or {\"operation\": \"todo_write\", \"options\": {\"session_id\": \"my-session\", \"repository\": \"github.com/user/repo\", \"todos\": [...]}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, errors.New("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository. DECISION GUIDE: OMIT session_id for cross-session continuity (recommended), INCLUDE for session isolation. Example: {\"repository\": \"github.com/user/repo\"} or {\"session_id\": \"my-session\", \"repository\": \"github.com/user/repo\"}")
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
		return nil, errors.New("operation parameter is required. Example: {\"operation\": \"export_project\", \"options\": {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, errors.New("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"}")
	}

	// SECURITY: Repository parameter is MANDATORY for all transfer operations
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, errors.New("repository parameter is required for all transfer operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"session-123\"} or use \"global\" for cross-repository data transfer and architecture continuity")
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
	operation, options, err := ms.validateSystemOperationParams(args)
	if err != nil {
		return nil, err
	}

	return ms.routeSystemOperation(ctx, operation, options)
}

// validateSystemOperationParams validates and extracts operation parameters
func (ms *MemoryServer) validateSystemOperationParams(args map[string]interface{}) (operation string, options map[string]interface{}, err error) {
	var ok bool
	operation, ok = args["operation"].(string)
	if !ok {
		return "", nil, errors.New("operation parameter is required. Example: {\"operation\": \"health\"} or {\"operation\": \"status\", \"options\": {\"repository\": \"github.com/user/repo\"}}")
	}

	options, ok = args["options"].(map[string]interface{})
	if !ok {
		// For system operations like health, options might be empty
		options = make(map[string]interface{})
	}

	return operation, options, nil
}

// routeSystemOperation routes the system operation to the appropriate handler
func (ms *MemoryServer) routeSystemOperation(ctx context.Context, operation string, options map[string]interface{}) (interface{}, error) {
	// Repository parameter requirements vary by operation
	repository, hasRepo := options["repository"].(string)

	switch operation {
	case OperationHealth:
		return ms.handleHealthOperation(ctx, options, repository, hasRepo)
	case OperationStatus:
		return ms.handleStatusOperation(ctx, options, repository, hasRepo)
	case "generate_citations":
		return ms.handleCitationOperation(ctx, options, repository, hasRepo)
	case "create_inline_citation":
		return ms.handleInlineCitationOperation(ctx, options, repository, hasRepo)
	case "get_documentation":
		return ms.handleDocumentationOperation(ctx, options)
	default:
		return ms.buildSystemOperationError(operation)
	}
}

// handleHealthOperation handles health check operations
func (ms *MemoryServer) handleHealthOperation(ctx context.Context, options map[string]interface{}, repository string, hasRepo bool) (interface{}, error) {
	// Health checks are global by default but can be repository-specific
	if hasRepo && repository != "" {
		logging.Info("Repository-specific health check requested", "repository", repository)
	} else {
		logging.Info("Global system health check requested")
	}
	return ms.handleHealth(ctx, options)
}

// handleStatusOperation handles status operations with repository validation
func (ms *MemoryServer) handleStatusOperation(ctx context.Context, options map[string]interface{}, repository string, hasRepo bool) (interface{}, error) {
	// Status operations require repository for multi-tenant isolation
	if !hasRepo || repository == "" {
		return nil, errors.New("repository parameter is required for status operations for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\"}")
	}
	return ms.handleMemoryStatus(ctx, options)
}

// handleCitationOperation handles citation generation with repository validation
func (ms *MemoryServer) handleCitationOperation(ctx context.Context, options map[string]interface{}, repository string, hasRepo bool) (interface{}, error) {
	// Citations require repository for proper scoping
	if !hasRepo || repository == "" {
		return nil, errors.New("repository parameter is required for citation generation for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"query\": \"search terms\", \"chunk_ids\": [\"id1\", \"id2\"]}")
	}
	return ms.handleGenerateCitations(ctx, options)
}

// handleInlineCitationOperation handles inline citation operations
func (ms *MemoryServer) handleInlineCitationOperation(ctx context.Context, options map[string]interface{}, repository string, hasRepo bool) (interface{}, error) {
	// Inline citations are repository-agnostic but logged for monitoring
	if hasRepo && repository != "" {
		logging.Info("Repository-specific inline citation requested", "repository", repository)
	}
	return ms.handleCreateInlineCitation(ctx, options)
}

// handleDocumentationOperation handles documentation requests
func (ms *MemoryServer) handleDocumentationOperation(ctx context.Context, options map[string]interface{}) (interface{}, error) {
	// Documentation is global by nature
	logging.Info("Global documentation request")
	return ms.handleGetDocumentation(ctx, options)
}

// buildSystemOperationError builds error message for unsupported system operations
func (ms *MemoryServer) buildSystemOperationError(operation string) (interface{}, error) {
	validOps := []string{"health", "status", "generate_citations", "create_inline_citation", "get_documentation"}
	return nil, fmt.Errorf("unsupported system operation '%s'. Valid operations: %s. Example: {\"operation\": \"health\"} or {\"operation\": \"status\", \"options\": {\"repository\": \"github.com/user/repo\"}}", operation, strings.Join(validOps, ", "))
}

// handleDocumentGeneration routes document generation operations to appropriate handlers
func (ms *MemoryServer) handleDocumentGeneration(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, errors.New("operation parameter is required. Example: {\"operation\": \"generate_prd\", \"options\": {\"repository\": \"github.com/user/repo\", \"user_inputs\": [\"Create a task management app\"]}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, errors.New("options parameter is required and MUST be a JSON object. Must contain repository. Example: {\"repository\": \"github.com/user/repo\", \"user_inputs\": [\"Create a task management app\"]}")
	}

	// Repository parameter is required for all document generation operations
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, errors.New("repository parameter is required for all document generation operations. Example: {\"repository\": \"github.com/user/repo\"}")
	}

	switch operation {
	case "generate_prd":
		return map[string]interface{}{
			"status":  "not_implemented",
			"message": "AI document generation moved to shared AI package - use CLI for document generation",
		}, nil
	case "generate_trd":
		return map[string]interface{}{
			"status":  "not_implemented",
			"message": "AI document generation moved to shared AI package - use CLI for document generation",
		}, nil
	case "generate_main_tasks":
		return map[string]interface{}{
			"status":  "not_implemented",
			"message": "AI task generation moved to shared AI package - use CLI for task generation",
		}, nil
	case "generate_sub_tasks":
		return map[string]interface{}{
			"status":  "not_implemented",
			"message": "AI task generation moved to shared AI package - use CLI for task generation",
		}, nil
	case "start_session":
		return map[string]interface{}{
			"status":  "not_implemented",
			"message": "Interactive sessions moved to shared AI package - use CLI for interactive sessions",
		}, nil
	case "continue_session":
		return map[string]interface{}{
			"status":  "not_implemented",
			"message": "Interactive sessions moved to shared AI package - use CLI for interactive sessions",
		}, nil
	case "end_session":
		return map[string]interface{}{
			"status":  "not_implemented",
			"message": "Interactive sessions moved to shared AI package - use CLI for interactive sessions",
		}, nil
	default:
		validOps := []string{"generate_prd", "generate_trd", "generate_main_tasks", "generate_sub_tasks", "start_session", "continue_session", "end_session"}
		return nil, fmt.Errorf("unsupported document generation operation '%s'. Valid operations: %s", operation, strings.Join(validOps, ", "))
	}
}

// handleTemplateManagement routes template management operations to appropriate handlers
func (ms *MemoryServer) handleTemplateManagement(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, errors.New("operation parameter is required. Example: {\"operation\": \"list_templates\", \"options\": {\"repository\": \"github.com/user/repo\"}}")
	}

	options, ok := args["options"].(map[string]interface{})
	if !ok {
		return nil, errors.New("options parameter is required and MUST be a JSON object (not a JSON string). Must contain repository for context. Example: {\"repository\": \"github.com/user/repo\"}")
	}

	// Repository parameter is required for context
	repository, ok := options["repository"].(string)
	if !ok || repository == "" {
		return nil, errors.New("repository parameter is required for all template operations for context. Example: {\"repository\": \"github.com/user/repo\"}")
	}

	// Get or create template service if not available
	if ms.templateService == nil {
		return nil, errors.New("template service not available - check server configuration")
	}

	switch operation {
	case "list_templates":
		return ms.handleListTemplates(ctx, options)
	case "get_template":
		return ms.handleGetTemplate(ctx, options)
	case "instantiate_template":
		return ms.handleInstantiateTemplate(ctx, options)
	default:
		validOps := []string{"list_templates", "get_template", "instantiate_template"}
		return nil, fmt.Errorf("unsupported template operation '%s'. Valid operations: %s", operation, strings.Join(validOps, ", "))
	}
}

// handleListTemplates handles template listing with filtering
func (ms *MemoryServer) handleListTemplates(ctx context.Context, options map[string]interface{}) (interface{}, error) {
	// Create request from options
	req := &templates.ListTemplatesRequest{}

	// Parse project type
	if projectType, ok := options["project_type"].(string); ok && projectType != "" {
		req.ProjectType = types.ProjectType(projectType)
	}

	// Parse category
	if category, ok := options["category"].(string); ok {
		req.Category = category
	}

	// Parse tags
	if tagsRaw, ok := options["tags"]; ok {
		if tagsSlice, ok := tagsRaw.([]interface{}); ok {
			for _, tagRaw := range tagsSlice {
				if tag, ok := tagRaw.(string); ok {
					req.Tags = append(req.Tags, tag)
				}
			}
		}
	}

	// Parse popular only
	if popularOnly, ok := options["popular_only"].(bool); ok {
		req.PopularOnly = popularOnly
	}

	// Parse limit
	switch limit := options["limit"].(type) {
	case float64:
		req.Limit = int(limit)
	case int:
		req.Limit = limit
	}

	// Call template service
	result, err := ms.templateService.ListTemplates(ctx, req)
	if err != nil {
		return map[string]interface{}{
			"status":    "error",
			"message":   "Failed to list templates: " + err.Error(),
			"templates": []interface{}{},
			"total":     0,
			"filtered":  0,
		}, nil
	}

	return map[string]interface{}{
		"status":    "success",
		"message":   fmt.Sprintf("Found %d templates (showing %d)", result.Total, result.Filtered),
		"templates": result.Templates,
		"total":     result.Total,
		"filtered":  result.Filtered,
	}, nil
}

// handleGetTemplate handles getting specific template details
func (ms *MemoryServer) handleGetTemplate(ctx context.Context, options map[string]interface{}) (interface{}, error) {
	// Get template ID
	templateID, ok := options["template_id"].(string)
	if !ok || templateID == "" {
		return map[string]interface{}{
			"status":   "error",
			"message":  "template_id is required",
			"template": nil,
		}, nil
	}

	// Call template service
	template, err := ms.templateService.GetTemplate(ctx, templateID)
	if err != nil {
		return map[string]interface{}{
			"status":   "error",
			"message":  "Failed to get template: " + err.Error(),
			"template": nil,
		}, nil
	}

	return map[string]interface{}{
		"status":   "success",
		"message":  "Template retrieved successfully",
		"template": template,
	}, nil
}

// handleInstantiateTemplate handles creating tasks from a template
func (ms *MemoryServer) handleInstantiateTemplate(ctx context.Context, options map[string]interface{}) (interface{}, error) {
	// Get required parameters
	templateID, ok := options["template_id"].(string)
	if !ok || templateID == "" {
		return map[string]interface{}{
			"status":  "error",
			"message": "template_id is required",
			"result":  nil,
		}, nil
	}

	projectID, ok := options["project_id"].(string)
	if !ok || projectID == "" {
		// Use repository as project_id if not provided
		if repo, repoOk := options["repository"].(string); repoOk {
			projectID = repo
		} else {
			return map[string]interface{}{
				"status":  "error",
				"message": "project_id is required",
				"result":  nil,
			}, nil
		}
	}

	// Get optional parameters
	sessionID, _ := options["session_id"].(string)
	prefix, _ := options["prefix"].(string)

	// Parse variables
	variables := make(map[string]interface{})
	if varsRaw, ok := options["variables"]; ok {
		if varsMap, ok := varsRaw.(map[string]interface{}); ok {
			variables = varsMap
		}
	}

	// Parse metadata
	metadata := make(map[string]interface{})
	if metaRaw, ok := options["metadata"]; ok {
		if metaMap, ok := metaRaw.(map[string]interface{}); ok {
			metadata = metaMap
		}
	}

	// Create instantiation request
	req := &templates.TemplateInstantiationRequest{
		TemplateID: templateID,
		ProjectID:  projectID,
		SessionID:  sessionID,
		Variables:  variables,
		Metadata:   metadata,
		Prefix:     prefix,
	}

	// Call template service
	result, err := ms.templateService.InstantiateTemplate(ctx, req)
	if err != nil {
		return map[string]interface{}{
			"status":  "error",
			"message": "Failed to instantiate template: " + err.Error(),
			"result":  nil,
		}, nil
	}

	// Build success message
	message := "Template instantiated successfully"
	if result.TaskCount > 0 {
		message += fmt.Sprintf(" with %d tasks", result.TaskCount)
	}
	if result.EstimatedTime != "" {
		message += fmt.Sprintf(" (estimated time: %s)", result.EstimatedTime)
	}
	if len(result.Warnings) > 0 {
		message += fmt.Sprintf(" with %d warnings", len(result.Warnings))
	}

	return map[string]interface{}{
		"status":  "success",
		"message": message,
		"result":  result,
	}, nil
}
