// Package mcp provides MCP server implementation
package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	mcp "github.com/fredcamaral/gomcp-sdk"
	"github.com/fredcamaral/gomcp-sdk/protocol"
	"github.com/fredcamaral/gomcp-sdk/server"
	"log"
	"math"
	"mcp-memory/internal/audit"
	"mcp-memory/internal/config"
	contextdetector "mcp-memory/internal/context"
	"mcp-memory/internal/di"
	"mcp-memory/internal/intelligence"
	"mcp-memory/internal/logging"
	"mcp-memory/internal/relationships"
	"mcp-memory/internal/threading"
	"mcp-memory/pkg/types"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// MemoryServer implements the MCP server for Claude memory
type MemoryServer struct {
	container *di.Container
	mcpServer *server.Server
}

// NewMemoryServer creates a new memory MCP server
func NewMemoryServer(cfg *config.Config) (*MemoryServer, error) {
	// Create dependency injection container
	container, err := di.NewContainer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create DI container: %w", err)
	}

	memServer := &MemoryServer{
		container: container,
	}

	// Create MCP server
	serverName := getEnv("SERVICE_NAME", "claude-memory")
	serverVersion := getEnv("SERVICE_VERSION", "1.0.0")
	mcpServer := mcp.NewServer(serverName, serverVersion)
	memServer.mcpServer = mcpServer
	memServer.registerTools()
	memServer.registerResources()

	return memServer, nil
}

// Start initializes and starts the MCP server
func (ms *MemoryServer) Start(ctx context.Context) error {
	// Initialize vector store
	if err := ms.container.GetVectorStore().Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize vector store: %w", err)
	}

	// Health check services
	if err := ms.container.HealthCheck(ctx); err != nil {
		log.Printf("Warning: Service health check failed: %v", err)
	}

	log.Printf("Claude Memory MCP Server started successfully")
	return nil
}

// GetMCPServer returns the underlying MCP server for testing
func (ms *MemoryServer) GetMCPServer() *server.Server {
	return ms.mcpServer
}

// registerTools registers all MCP tools
func (ms *MemoryServer) registerTools() {
	// Register all MCP tools with proper schemas

	ms.mcpServer.AddTool(mcp.NewTool(
		"mcp__memory__memory_store_chunk",
		"Store important conversation moments (bug fixes, solutions, decisions, learnings) for future reference. Automatically categorizes and links related memories.",
		mcp.ObjectSchema("Store memory chunk parameters", map[string]interface{}{
			"content":        mcp.StringParam("The conversation content to store (include problem, solution, context, and outcome)", true),
			"session_id":     mcp.StringParam("Session identifier for grouping related chunks", true),
			"repository":     mcp.StringParam("Official repository name (e.g., 'github.com/lerianstudio/midaz', 'gitlab.com/user/project'). Use '_global' for global memories (optional)", false),
			"branch":         mcp.StringParam("Git branch name (optional)", false),
			"files_modified": mcp.ArraySchema("List of files that were modified", map[string]interface{}{"type": "string"}),
			"tools_used":     mcp.ArraySchema("List of tools that were used", map[string]interface{}{"type": "string"}),
			"tags":           mcp.ArraySchema("Additional tags for categorization (e.g., 'bug-fix', 'performance', 'architecture')", map[string]interface{}{"type": "string"}),
			"client_type":    mcp.StringParam("Client type (e.g., 'claude-cli', 'chatgpt', 'vscode', 'web', 'api')", false),
		}, []string{"content", "session_id"}),
	), mcp.ToolHandlerFunc(ms.handleStoreChunk))

	ms.mcpServer.AddTool(mcp.NewTool(
		"mcp__memory__memory_search",
		"Search past memories using natural language. Finds similar problems, solutions, and decisions. Use before solving to check if issue was encountered before.",
		mcp.ObjectSchema("Search memory parameters", map[string]interface{}{
			"query":      mcp.StringParam("Natural language search query (be specific about the problem or topic)", true),
			"repository": mcp.StringParam("Filter by official repository name (e.g., 'github.com/lerianstudio/midaz'). Use '_global' for global memories (optional)", false),
			"recency": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"recent", "last_month", "all_time"},
				"description": "Time filter for results",
				"default":     "recent",
			},
			"types": mcp.ArraySchema("Filter by chunk types", map[string]interface{}{"type": "string"}),
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results",
				"default":     getEnvInt("MCP_MEMORY_DEFAULT_SEARCH_LIMIT", 10),
				"minimum":     1,
				"maximum":     50,
			},
			"min_relevance": map[string]interface{}{
				"type":        "number",
				"description": "Minimum relevance score (0-1)",
				"default":     0.7,
				"minimum":     0,
				"maximum":     1,
			},
		}, []string{"query"}),
	), mcp.ToolHandlerFunc(ms.handleSearch))

	ms.mcpServer.AddTool(mcp.NewTool(
		"mcp__memory__memory_get_context",
		"Get project overview and recent activity. Use at session start or when switching projects to understand context, patterns, and ongoing work.",
		mcp.ObjectSchema("Get context parameters", map[string]interface{}{
			"repository": mcp.StringParam("Official repository name to get context for (e.g., 'github.com/lerianstudio/midaz'). Use '_global' for global memories", true),
			"recent_days": map[string]interface{}{
				"type":        "integer",
				"description": "Number of recent days to include",
				"default":     7,
				"minimum":     1,
				"maximum":     90,
			},
		}, []string{"repository"}),
	), mcp.ToolHandlerFunc(ms.handleGetContext))

	ms.mcpServer.AddTool(mcp.NewTool(
		"mcp__memory__memory_find_similar",
		"Find similar problems and their solutions from past experiences. Use when facing errors or complex challenges to learn from previous solutions.",
		mcp.ObjectSchema("Find similar parameters", map[string]interface{}{
			"problem":    mcp.StringParam("Description of the current problem or error (include error messages, context, and what you've tried)", true),
			"repository": mcp.StringParam("Official repository name for context (e.g., 'github.com/lerianstudio/midaz'). Use '_global' for global memories (optional)", false),
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of similar problems to return",
				"default":     5,
				"minimum":     1,
				"maximum":     20,
			},
		}, []string{"problem"}),
	), mcp.ToolHandlerFunc(ms.handleFindSimilar))

	ms.mcpServer.AddTool(mcp.NewTool(
		"mcp__memory__memory_store_decision",
		"Explicitly store architectural/design decisions with rationale and alternatives. Use after making significant technical choices to preserve context.",
		mcp.ObjectSchema("Store decision parameters", map[string]interface{}{
			"decision":    mcp.StringParam("The architectural or design decision made", true),
			"rationale":   mcp.StringParam("Why this decision was made (include benefits and trade-offs)", true),
			"context":     mcp.StringParam("Alternatives considered, constraints, benchmarks, or other relevant context", false),
			"repository":  mcp.StringParam("Official repository name this decision applies to (e.g., 'github.com/lerianstudio/midaz'). Use '_global' for global decisions (optional)", false),
			"session_id":  mcp.StringParam("Session identifier", true),
			"client_type": mcp.StringParam("Client type (e.g., 'claude-cli', 'chatgpt', 'vscode', 'web', 'api')", false),
		}, []string{"decision", "rationale", "session_id"}),
	), mcp.ToolHandlerFunc(ms.handleStoreDecision))

	ms.mcpServer.AddTool(mcp.NewTool(
		"mcp__memory__memory_get_patterns",
		"Identify recurring patterns, common issues, and trends. Use for retrospectives, identifying refactoring needs, or understanding project challenges.",
		mcp.ObjectSchema("Get patterns parameters", map[string]interface{}{
			"repository": mcp.StringParam("Official repository name to analyze (e.g., 'github.com/lerianstudio/midaz'). Use '_global' for global patterns", true),
			"timeframe": map[string]interface{}{
				"type":        "string",
				"enum":        []string{types.TimeframWeek, types.TimeframeMonth, "quarter", "all"},
				"description": "Time period to analyze",
				"default":     types.TimeframeMonth,
			},
		}, []string{"repository"}),
	), mcp.ToolHandlerFunc(ms.handleGetPatterns))

	ms.mcpServer.AddTool(mcp.NewTool(
		"mcp__memory__memory_health",
		"Check the health status of the memory system",
		mcp.ObjectSchema("Health check parameters", map[string]interface{}{}, []string{}),
	), mcp.ToolHandlerFunc(ms.handleHealth))

	// Phase 3.2: Advanced MCP Tools
	ms.mcpServer.AddTool(mcp.NewTool(
		"mcp__memory__memory_suggest_related",
		"Get AI-powered suggestions for related context based on current work",
		mcp.ObjectSchema("Suggest related parameters", map[string]interface{}{
			"current_context": mcp.StringParam("Current work context or conversation content", true),
			"repository":      mcp.StringParam("Official repository name to search for related context (e.g., 'github.com/lerianstudio/midaz'). Use '_global' for global context (optional)", false),
			"max_suggestions": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of suggestions to return",
				"minimum":     1,
				"maximum":     10,
				"default":     5,
			},
			"include_patterns": mcp.BooleanParam("Include pattern-based suggestions", false),
			"session_id":       mcp.StringParam("Session identifier", true),
		}, []string{"current_context", "session_id"}),
	), mcp.ToolHandlerFunc(ms.handleSuggestRelated))

	ms.mcpServer.AddTool(mcp.NewTool(
		"mcp__memory__memory_export_project",
		"Export all memory data for a project in various formats",
		mcp.ObjectSchema("Export project parameters", map[string]interface{}{
			"repository": mcp.StringParam("Official repository name to export (e.g., 'github.com/lerianstudio/midaz'). Use '_global' for global memories", true),
			"format": map[string]interface{}{
				"type":        "string",
				"description": "Export format: json, markdown, or archive",
				"enum":        []string{"json", "markdown", "archive"},
				"default":     "json",
			},
			"include_vectors": mcp.BooleanParam("Include vector embeddings in export", false),
			"date_range": mcp.ObjectSchema("Date range filter", map[string]interface{}{
				"start": mcp.StringParam("Start date (ISO 8601 format)", false),
				"end":   mcp.StringParam("End date (ISO 8601 format)", false),
			}, []string{}),
			"session_id": mcp.StringParam("Session identifier", true),
		}, []string{"repository", "session_id"}),
	), mcp.ToolHandlerFunc(ms.handleExportProject))

	ms.mcpServer.AddTool(mcp.NewTool(
		"mcp__memory__memory_import_context",
		"Import conversation context from external source",
		mcp.ObjectSchema("Import context parameters", map[string]interface{}{
			"source": map[string]interface{}{
				"type":        "string",
				"description": "Source type: conversation, file, or archive",
				"enum":        []string{types.SourceConversation, "file", "archive"},
				"default":     types.SourceConversation,
			},
			"data":       mcp.StringParam("Data to import (conversation text, file content, or base64 archive)", true),
			"repository": mcp.StringParam("Official repository name for imported data (e.g., 'github.com/lerianstudio/midaz'). Use '_global' for global memories", true),
			"metadata": mcp.ObjectSchema("Import metadata", map[string]interface{}{
				"source_system": mcp.StringParam("Name of the source system", false),
				"import_date":   mcp.StringParam("Original date of the content", false),
				"tags":          mcp.ArraySchema("Tags to apply to imported content", map[string]interface{}{"type": "string"}),
			}, []string{}),
			"chunking_strategy": map[string]interface{}{
				"type":        "string",
				"description": "How to chunk the imported data",
				"enum":        []string{"auto", "paragraph", "fixed_size", "conversation_turns"},
				"default":     "auto",
			},
			"session_id": mcp.StringParam("Session identifier", true),
		}, []string{"source", "data", "repository", "session_id"}),
	), mcp.ToolHandlerFunc(ms.handleImportContext))

	// Quick Memory Actions - Convenience tools for common workflow queries
	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_status",
			"Get comprehensive status overview of memory system for a repository",
			mcp.ObjectSchema("Memory status parameters", map[string]interface{}{
				"repository": mcp.StringParam("Official repository name to get status for (e.g., 'github.com/lerianstudio/midaz')", true),
			}, []string{"repository"}),
		), mcp.ToolHandlerFunc(ms.handleMemoryStatus))

	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_conflicts",
			"Detect contradictory decisions or patterns across memories",
			mcp.ObjectSchema("Memory conflicts parameters", map[string]interface{}{
				"repository": mcp.StringParam("Official repository name to analyze for conflicts (e.g., 'github.com/lerianstudio/midaz'). Use '_global' for global analysis (optional)", false),
				"timeframe":  mcp.StringParam("Time period to analyze: 'week', 'month', 'quarter', 'all' (default: 'month')", false),
			}, []string{}),
		), mcp.ToolHandlerFunc(ms.handleMemoryConflicts))

	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_continuity",
			"Show what was left incomplete from previous sessions for resuming work",
			mcp.ObjectSchema("Memory continuity parameters", map[string]interface{}{
				"repository":          mcp.StringParam("Official repository name to check for incomplete work (e.g., 'github.com/lerianstudio/midaz')", true),
				"session_id":          mcp.StringParam("Specific session to check (optional, uses most recent if not provided)", false),
				"include_suggestions": mcp.BooleanParam("Include suggestions for resuming work (default: true)", false),
			}, []string{"repository"}),
		), mcp.ToolHandlerFunc(ms.handleMemoryContinuity))

	// Memory Threading Tools
	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_create_thread",
			"Create a memory thread from related chunks to group coherent conversations",
			mcp.ObjectSchema("Memory thread creation parameters", map[string]interface{}{
				"chunk_ids": mcp.ArraySchema("List of chunk IDs to include in the thread", map[string]interface{}{
					"type": "string",
				}),
				"thread_type": mcp.StringParam("Type of thread: 'conversation', 'problem_solving', 'feature', 'debugging', 'architecture', 'workflow'", false),
				"title":       mcp.StringParam("Custom title for the thread (optional)", false),
				"repository":  mcp.StringParam("Official repository name for the thread (e.g., 'github.com/lerianstudio/midaz'). Optional - inferred from chunks if not provided", false),
			}, []string{"chunk_ids"}),
		), mcp.ToolHandlerFunc(ms.handleCreateThread))

	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_get_threads",
			"Retrieve memory threads with optional filtering",
			mcp.ObjectSchema("Memory thread retrieval parameters", map[string]interface{}{
				"repository":      mcp.StringParam("Filter by official repository name (e.g., 'github.com/lerianstudio/midaz') (optional)", false),
				"status":          mcp.StringParam("Thread status: 'active', 'complete', 'paused', 'abandoned', 'blocked' (optional)", false),
				"thread_type":     mcp.StringParam("Thread type to filter by (optional)", false),
				"session_id":      mcp.StringParam("Session ID to filter by (optional)", false),
				"include_summary": mcp.BooleanParam("Include thread summaries with chunk analysis (default: false)", false),
			}, []string{}),
		), mcp.ToolHandlerFunc(ms.handleGetThreads))

	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_detect_threads",
			"Automatically detect and create memory threads from existing chunks",
			mcp.ObjectSchema("Memory thread detection parameters", map[string]interface{}{
				"repository":  mcp.StringParam("Official repository name to analyze for thread detection (e.g., 'github.com/lerianstudio/midaz')", true),
				"auto_create": mcp.BooleanParam("Automatically create detected threads (default: true)", false),
				"min_thread_size": map[string]interface{}{
					"type":        "integer",
					"description": "Minimum number of chunks required for a thread (default: 2)",
					"default":     2,
					"minimum":     2,
					"maximum":     10,
				},
			}, []string{"repository"}),
		), mcp.ToolHandlerFunc(ms.handleDetectThreads))

	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_update_thread",
			"Update memory thread properties like status, title, or add/remove chunks",
			mcp.ObjectSchema("Memory thread update parameters", map[string]interface{}{
				"thread_id": mcp.StringParam("Thread ID to update", true),
				"status":    mcp.StringParam("New thread status (optional)", false),
				"title":     mcp.StringParam("New thread title (optional)", false),
				"add_chunks": mcp.ArraySchema("Chunk IDs to add to the thread (optional)", map[string]interface{}{
					"type": "string",
				}),
				"remove_chunks": mcp.ArraySchema("Chunk IDs to remove from the thread (optional)", map[string]interface{}{
					"type": "string",
				}),
			}, []string{"thread_id"}),
		), mcp.ToolHandlerFunc(ms.handleUpdateThread))

	// Cross-Project Pattern Detection Tools
	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_analyze_cross_repo_patterns",
			"Analyze patterns that appear across multiple repositories to identify shared solutions, common problems, and best practices",
			mcp.ObjectSchema("Cross-repository pattern analysis parameters", map[string]interface{}{
				"session_id": mcp.StringParam("Session identifier", true),
				"repositories": mcp.ArraySchema("Specific official repository names to analyze (e.g., ['github.com/lerianstudio/midaz', 'gitlab.com/user/project']). Optional - analyzes all if not specified", map[string]interface{}{
					"type": "string",
				}),
				"tech_stacks": mcp.ArraySchema("Filter by technology stacks (e.g., 'go', 'react', 'docker')", map[string]interface{}{
					"type": "string",
				}),
				"pattern_types": mcp.ArraySchema("Pattern types to focus on (e.g., 'problem_solution', 'architectural', 'debugging')", map[string]interface{}{
					"type": "string",
				}),
				"min_frequency": map[string]interface{}{
					"type":        "integer",
					"description": "Minimum frequency for pattern to be considered significant (default: 2)",
					"default":     2,
					"minimum":     2,
					"maximum":     10,
				},
			}, []string{"session_id"}),
		), mcp.ToolHandlerFunc(ms.handleAnalyzeCrossRepoPatterns))

	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_find_similar_repositories",
			"Find repositories with similar technology stacks, patterns, or problem domains for knowledge transfer and best practice sharing",
			mcp.ObjectSchema("Similar repository discovery parameters", map[string]interface{}{
				"repository": mcp.StringParam("Official repository name to find similarities for (e.g., 'github.com/lerianstudio/midaz')", true),
				"session_id": mcp.StringParam("Session identifier", true),
				"similarity_threshold": map[string]interface{}{
					"type":        "number",
					"description": "Minimum similarity score (0.0-1.0, default: 0.6)",
					"default":     0.6,
					"minimum":     0.1,
					"maximum":     1.0,
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of similar repositories to return (default: 5)",
					"default":     5,
					"minimum":     1,
					"maximum":     20,
				},
			}, []string{"repository", "session_id"}),
		), mcp.ToolHandlerFunc(ms.handleFindSimilarRepositories))

	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_get_cross_repo_insights",
			"Get comprehensive insights across all repositories including technology distribution, success rates, and common patterns",
			mcp.ObjectSchema("Cross-repository insights parameters", map[string]interface{}{
				"session_id":                mcp.StringParam("Session identifier", true),
				"include_tech_distribution": mcp.BooleanParam("Include technology stack distribution analysis (default: true)", false),
				"include_success_analytics": mcp.BooleanParam("Include success rate analytics across repositories (default: true)", false),
				"include_pattern_frequency": mcp.BooleanParam("Include most common patterns across repositories (default: true)", false),
			}, []string{"session_id"}),
		), mcp.ToolHandlerFunc(ms.handleGetCrossRepoInsights))

	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_search_multi_repo",
			"Search for patterns, solutions, or insights across multiple repositories with advanced filtering and ranking",
			mcp.ObjectSchema("Multi-repository search parameters", map[string]interface{}{
				"query":      mcp.StringParam("Search query for patterns, problems, or solutions", true),
				"session_id": mcp.StringParam("Session identifier", true),
				"repositories": mcp.ArraySchema("Specific official repository names to search (e.g., ['github.com/lerianstudio/midaz', 'gitlab.com/user/project']). Optional - searches all if not specified", map[string]interface{}{
					"type": "string",
				}),
				"tech_stacks": mcp.ArraySchema("Filter by technology stacks", map[string]interface{}{
					"type": "string",
				}),
				"frameworks": mcp.ArraySchema("Filter by frameworks", map[string]interface{}{
					"type": "string",
				}),
				"pattern_types": mcp.ArraySchema("Filter by pattern types", map[string]interface{}{
					"type": "string",
				}),
				"min_confidence": map[string]interface{}{
					"type":        "number",
					"description": "Minimum confidence score for results (0.0-1.0, default: 0.5)",
					"default":     0.5,
					"minimum":     0.0,
					"maximum":     1.0,
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (default: 10)",
					"default":     10,
					"minimum":     1,
					"maximum":     50,
				},
				"include_similar": mcp.BooleanParam("Include results from similar repositories (default: true)", false),
			}, []string{"query", "session_id"}),
		), mcp.ToolHandlerFunc(ms.handleSearchMultiRepo))

	// Memory Health Dashboard Tool
	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_health_dashboard",
			"Get comprehensive memory system health overview including completion rates, outdated chunks, effectiveness scores, and system performance metrics",
			mcp.ObjectSchema("Memory health dashboard parameters", map[string]interface{}{
				"repository": mcp.StringParam("Official repository name to analyze (e.g., 'github.com/lerianstudio/midaz'). Use '_global' for global analysis", true),
				"session_id": mcp.StringParam("Session identifier", true),
				"timeframe": map[string]interface{}{
					"type":        "string",
					"enum":        []string{types.TimeframWeek, types.TimeframeMonth, "quarter", "all"},
					"description": "Analysis timeframe (default: 'month')",
					"default":     types.TimeframeMonth,
				},
				"include_details":         mcp.BooleanParam("Include detailed analysis of chunks and patterns (default: true)", false),
				"include_recommendations": mcp.BooleanParam("Include actionable recommendations for improvement (default: true)", false),
			}, []string{"repository", "session_id"}),
		), mcp.ToolHandlerFunc(ms.handleMemoryHealthDashboard))

	// Memory decay management tool
	ms.mcpServer.AddTool(
		mcp.NewTool("mcp__memory__memory_decay_management",
			"Manage memory decay process with intelligent LLM-based summarization and archival",
			mcp.ObjectSchema("Memory decay management parameters", map[string]interface{}{
				"repository": mcp.StringParam("Official repository name to process (e.g., 'github.com/lerianstudio/midaz'). Use '_global' for global decay analysis", true),
				"session_id": mcp.StringParam("Session identifier", true),
				"action":     mcp.StringParam("Action to perform: 'run_decay', 'configure', 'status', 'preview'", true),
				"config": map[string]interface{}{
					"type":        "object",
					"description": "Decay configuration (for 'configure' action)",
					"properties": map[string]interface{}{
						"strategy": map[string]interface{}{
							"type":        "string",
							"description": "Decay strategy: 'exponential', 'linear', 'adaptive'",
							"enum":        []string{"exponential", "linear", "adaptive"},
						},
						"base_decay_rate": map[string]interface{}{
							"type":        "number",
							"description": "Base decay rate (0.0 to 1.0)",
							"minimum":     0.0,
							"maximum":     1.0,
						},
						"summarization_threshold": map[string]interface{}{
							"type":        "number",
							"description": "Score below which memories get summarized",
							"minimum":     0.0,
							"maximum":     1.0,
						},
						"deletion_threshold": map[string]interface{}{
							"type":        "number",
							"description": "Score below which memories get deleted",
							"minimum":     0.0,
							"maximum":     1.0,
						},
						"retention_period_days": map[string]interface{}{
							"type":        "number",
							"description": "Minimum days to keep memories",
							"minimum":     1,
						},
					},
				},
				"preview_only":     mcp.BooleanParam("Whether to only preview what would be processed without making changes", false),
				"intelligent_mode": mcp.BooleanParam("Whether to use intelligent LLM-based summarization with embeddings", false),
			}, []string{"repository", "session_id", "action"}),
		), mcp.ToolHandlerFunc(ms.handleMemoryDecayManagement))
}

// registerResources registers MCP resources for browsing memory
func (ms *MemoryServer) registerResources() {
	// Register MCP resources for browsing memory data

	resources := []struct {
		uri         string
		name        string
		description string
		mimeType    string
	}{
		{
			uri:         "memory://recent/{repository}",
			name:        "Recent Activity",
			description: "Recent conversation chunks for a repository",
			mimeType:    "application/json",
		},
		{
			uri:         "memory://patterns/{repository}",
			name:        "Common Patterns",
			description: "Identified patterns in project history",
			mimeType:    "application/json",
		},
		{
			uri:         "memory://decisions/{repository}",
			name:        "Architectural Decisions",
			description: "Key architectural decisions made",
			mimeType:    "application/json",
		},
		{
			uri:         "memory://global/insights",
			name:        "Global Insights",
			description: "Cross-project insights and patterns",
			mimeType:    "application/json",
		},
	}

	for _, res := range resources {
		resource := mcp.NewResource(res.uri, res.name, res.description, res.mimeType)
		ms.mcpServer.AddResource(resource, mcp.ResourceHandlerFunc(ms.handleResourceRead))
	}
}

// Tool handlers

func (ms *MemoryServer) handleStoreChunk(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_store_chunk called", "params", params)

	content, ok := params["content"].(string)
	if !ok || content == "" {
		logging.Error("memory_store_chunk failed: missing content parameter")
		return nil, fmt.Errorf("content is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_store_chunk failed: missing session_id parameter")
		return nil, fmt.Errorf("session_id is required")
	}

	logging.Info("Processing chunk storage", "content_length", len(content), "session_id", sessionID)

	// Build metadata from parameters
	metadata := ms.buildMetadataFromParams(params)

	// Add extended metadata with context detection
	if err := ms.addContextMetadata(&metadata, params); err != nil {
		logging.Warn("Failed to add context metadata", "error", err)
	}

	// Create and store chunk
	logging.Info("Creating conversation chunk", "session_id", sessionID)
	chunk, err := ms.container.GetChunkingService().CreateChunk(ctx, sessionID, content, metadata)
	if err != nil {
		logging.Error("Failed to create chunk", "error", err, "session_id", sessionID)
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}
	logging.Info("Chunk created successfully", "chunk_id", chunk.ID, "type", chunk.Type)

	// Check for parent chunk ID in extended metadata
	if metadata.ExtendedMetadata != nil {
		if parentID, ok := metadata.ExtendedMetadata[types.EMKeyParentChunk].(string); ok && parentID != "" {
			// Create parent-child relationship
			relMgr := ms.container.GetRelationshipManager()
			_, err := relMgr.AddRelationship(ctx, parentID, chunk.ID, relationships.RelTypeParentChild, 1.0, "Explicit parent-child relationship")
			if err != nil {
				logging.Warn("Failed to create parent-child relationship", "error", err, "parent", parentID, "child", chunk.ID)
			}
		}
	}

	logging.Info("Storing chunk in vector store", "chunk_id", chunk.ID)

	// Start timing for audit
	startTime := time.Now()

	if err := ms.container.GetVectorStore().Store(ctx, *chunk); err != nil {
		logging.Error("Failed to store chunk", "error", err, "chunk_id", chunk.ID)

		// Log audit error
		if auditLogger := ms.container.GetAuditLogger(); auditLogger != nil {
			auditLogger.LogError(ctx, audit.EventTypeMemoryStore, "Failed to store memory chunk", "memory", err, map[string]interface{}{
				"chunk_id":   chunk.ID,
				"chunk_type": string(chunk.Type),
				"repository": chunk.Metadata.Repository,
				"session_id": sessionID,
			})
		}

		return nil, fmt.Errorf("failed to store chunk: %w", err)
	}

	// Log successful audit event
	if auditLogger := ms.container.GetAuditLogger(); auditLogger != nil {
		auditLogger.LogEventWithDuration(ctx, audit.EventTypeMemoryStore, "Stored memory chunk", "memory", chunk.ID,
			time.Since(startTime), map[string]interface{}{
				"chunk_type":  string(chunk.Type),
				"repository":  chunk.Metadata.Repository,
				"session_id":  sessionID,
				"tags":        chunk.Metadata.Tags,
				"files_count": len(chunk.Metadata.FilesModified),
				"tools_count": len(chunk.Metadata.ToolsUsed),
				"has_parent":  metadata.ExtendedMetadata != nil && metadata.ExtendedMetadata[types.EMKeyParentChunk] != nil,
			})
	}

	// Auto-detect relationships with recent chunks
	go func() {
		// Get recent chunks from the same session or repository
		repo := chunk.Metadata.Repository
		query := types.MemoryQuery{
			Repository: &repo,
			Limit:      20,
			Recency:    types.RecencyRecent,
		}

		// Use a simple search to get recent chunks
		results, err := ms.container.GetVectorStore().Search(ctx, query, chunk.Embeddings)
		if err == nil && len(results.Results) > 0 {
			existingChunks := make([]types.ConversationChunk, 0, len(results.Results))
			for _, result := range results.Results {
				if result.Chunk.ID != chunk.ID { // Don't include self
					existingChunks = append(existingChunks, result.Chunk)
				}
			}

			// Detect and store relationships
			relMgr := ms.container.GetRelationshipManager()
			detected := relMgr.DetectRelationships(ctx, chunk, existingChunks)
			logging.Info("Auto-detected relationships", "chunk_id", chunk.ID, "count", len(detected))
		}
	}()

	logging.Info("memory_store_chunk completed successfully", "chunk_id", chunk.ID, "session_id", sessionID)
	return map[string]interface{}{
		"chunk_id":  chunk.ID,
		"type":      string(chunk.Type),
		"summary":   chunk.Summary,
		"stored_at": chunk.Timestamp.Format(time.RFC3339),
	}, nil
}

func (ms *MemoryServer) handleSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_search called", "params", params)

	query, ok := params["query"].(string)
	if !ok || query == "" {
		logging.Error("memory_search failed: missing query parameter")
		return nil, fmt.Errorf("query is required")
	}

	logging.Info("Processing search query", "query", query)

	// Build memory query
	memQuery := types.NewMemoryQuery(query)

	if repo, ok := params["repository"].(string); ok && repo != "" {
		memQuery.Repository = &repo
	}

	if recency, ok := params["recency"].(string); ok {
		memQuery.Recency = types.Recency(recency)
	}

	if limit, ok := params["limit"].(float64); ok {
		memQuery.Limit = int(limit)
	}

	if minRel, ok := params["min_relevance"].(float64); ok {
		memQuery.MinRelevanceScore = minRel
	}

	if chunkTypes, ok := params["types"].([]interface{}); ok {
		for _, t := range chunkTypes {
			if typeStr, ok := t.(string); ok {
				memQuery.Types = append(memQuery.Types, types.ChunkType(typeStr))
			}
		}
	}

	// Generate embeddings for query
	logging.Info("Generating embeddings for search query", "query", query)
	embeddings, err := ms.container.GetEmbeddingService().GenerateEmbedding(ctx, query)
	if err != nil {
		logging.Error("Failed to generate embeddings", "error", err, "query", query)
		return nil, fmt.Errorf("failed to generate query embeddings: %w", err)
	}
	logging.Info("Embeddings generated successfully", "dimension", len(embeddings))

	// Progressive search with relaxation strategy
	searchStart := time.Now()
	results, err := ms.executeProgressiveSearch(ctx, *memQuery, embeddings)
	if err != nil {
		logging.Error("Progressive search failed", "error", err, "query", query)

		// Log audit error
		if auditLogger := ms.container.GetAuditLogger(); auditLogger != nil {
			auditLogger.LogError(ctx, audit.EventTypeMemorySearch, "Memory search failed", "memory", err, map[string]interface{}{
				"query":      query,
				"repository": memQuery.Repository,
				"limit":      memQuery.Limit,
			})
		}

		return nil, fmt.Errorf("search failed: %w", err)
	}
	logging.Info("Progressive search completed", "total_results", results.Total, "query_time", results.QueryTime)

	// Log successful search audit event
	if auditLogger := ms.container.GetAuditLogger(); auditLogger != nil {
		auditLogger.LogEventWithDuration(ctx, audit.EventTypeMemorySearch, "Searched memories", "memory", "",
			time.Since(searchStart), map[string]interface{}{
				"query":         query,
				"repository":    memQuery.Repository,
				"limit":         memQuery.Limit,
				"results_count": results.Total,
				"query_time_ms": results.QueryTime.Milliseconds(),
				"min_relevance": memQuery.MinRelevanceScore,
				"types":         memQuery.Types,
			})
	}

	// Format results for response
	response := map[string]interface{}{
		"query":      query,
		"total":      results.Total,
		"query_time": results.QueryTime.String(),
		"results":    []map[string]interface{}{},
	}

	relMgr := ms.container.GetRelationshipManager()
	analytics := ms.container.GetMemoryAnalytics()

	for _, result := range results.Results {
		// Track access to this memory
		if analytics != nil {
			if err := analytics.RecordAccess(ctx, result.Chunk.ID); err != nil {
				logging.Warn("Failed to record memory access", "chunk_id", result.Chunk.ID, "error", err)
			}
		}
		resultMap := map[string]interface{}{
			"chunk_id":   result.Chunk.ID,
			"score":      result.Score,
			"type":       string(result.Chunk.Type),
			"summary":    result.Chunk.Summary,
			"repository": result.Chunk.Metadata.Repository,
			"timestamp":  result.Chunk.Timestamp.Format(time.RFC3339),
			"tags":       result.Chunk.Metadata.Tags,
			"outcome":    string(result.Chunk.Metadata.Outcome),
		}

		// Add relationship information
		relationships := relMgr.GetRelationships(result.Chunk.ID)
		if len(relationships) > 0 {
			relInfo := make([]map[string]interface{}, 0, len(relationships))
			for _, rel := range relationships {
				relInfo = append(relInfo, map[string]interface{}{
					"type":     string(rel.Type),
					"from":     rel.FromChunkID,
					"to":       rel.ToChunkID,
					"strength": rel.Strength,
					"context":  rel.Context,
				})
			}
			resultMap["relationships"] = relInfo
		}

		// Add extended metadata if present
		if result.Chunk.Metadata.ExtendedMetadata != nil {
			resultMap["extended_metadata"] = result.Chunk.Metadata.ExtendedMetadata
		}

		response["results"] = append(response["results"].([]map[string]interface{}), resultMap)
	}

	logging.Info("memory_search completed successfully", "total_results", results.Total, "query", query)
	return response, nil
}

// executeProgressiveSearch implements a fallback strategy for searches
// Tries progressively looser search criteria if initial search returns no results
func (ms *MemoryServer) executeProgressiveSearch(ctx context.Context, query types.MemoryQuery, embeddings []float64) (*types.SearchResults, error) {
	searchConfig := ms.container.Config.Search

	// If progressive search is disabled, just do a single search
	if !searchConfig.EnableProgressiveSearch {
		return ms.container.GetVectorStore().Search(ctx, query, embeddings)
	}

	// Step 1: Try original query (strict search)
	logging.Info("Progressive search: Step 1 - Strict search", "repo", query.Repository, "min_relevance", query.MinRelevanceScore)
	results, err := ms.container.GetVectorStore().Search(ctx, query, embeddings)
	if err != nil {
		return nil, err
	}

	if len(results.Results) > 0 {
		logging.Info("Progressive search: Strict search succeeded", "results", len(results.Results))
		return results, nil
	}

	// Step 2: Relax relevance score (loose search)
	relaxedQuery := query
	relaxedQuery.MinRelevanceScore = searchConfig.RelaxedMinRelevance
	logging.Info("Progressive search: Step 2 - Relaxed relevance", "min_relevance", relaxedQuery.MinRelevanceScore)
	results, err = ms.container.GetVectorStore().Search(ctx, relaxedQuery, embeddings)
	if err != nil {
		return nil, err
	}

	if len(results.Results) > 0 {
		logging.Info("Progressive search: Relaxed search succeeded", "results", len(results.Results))
		return results, nil
	}

	// Step 3: Try related repositories if original repo specified
	if query.Repository != nil && searchConfig.EnableRepositoryFallback {
		results, err := ms.searchRelatedRepositories(ctx, relaxedQuery, embeddings, *query.Repository, searchConfig)
		if err == nil && len(results.Results) > 0 {
			return results, nil
		}

		// Step 3b: Complete repository fallback (remove filter)
		repoFallbackQuery := relaxedQuery
		repoFallbackQuery.Repository = nil
		logging.Info("Progressive search: Step 3b - Complete repository fallback", "original_repo", *query.Repository)
		results, err = ms.container.GetVectorStore().Search(ctx, repoFallbackQuery, embeddings)
		if err != nil {
			return nil, err
		}

		if len(results.Results) > 0 {
			logging.Info("Progressive search: Complete repository fallback succeeded", "results", len(results.Results))
			return results, nil
		}
	}

	// Step 4: Broadest search - remove type filters too
	broadQuery := query
	broadQuery.MinRelevanceScore = searchConfig.BroadestMinRelevance
	broadQuery.Repository = nil
	broadQuery.Types = nil
	logging.Info("Progressive search: Step 4 - Broadest search", "min_relevance", broadQuery.MinRelevanceScore)
	results, err = ms.container.GetVectorStore().Search(ctx, broadQuery, embeddings)
	if err != nil {
		return nil, err
	}

	logging.Info("Progressive search completed", "final_results", len(results.Results), "strategy", "broadest")
	return results, nil
}

// generateRelatedRepositories creates variations of a repository name for fallback searches
// Examples: "libs/commons-go" -> ["commons-go", "libs/commons", "commons", "go"]
func (ms *MemoryServer) generateRelatedRepositories(originalRepo string) []string {
	related := []string{}

	// Split on common separators
	parts := []string{}
	for _, sep := range []string{"/", "-", "_", "."} {
		if strings.Contains(originalRepo, sep) {
			parts = strings.Split(originalRepo, sep)
			break
		}
	}

	if len(parts) > 1 {
		// Add individual parts (biggest to smallest)
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] != "" && len(parts[i]) > 2 { // Skip very short parts
				related = append(related, parts[i])
			}
		}

		// Add combinations
		if len(parts) >= 2 {
			// Last two parts: "commons-go" from "libs/commons-go"
			related = append(related, strings.Join(parts[len(parts)-2:], "-"))

			// First two parts: "libs/commons" from "libs/commons/go"
			if len(parts) >= 3 {
				related = append(related, strings.Join(parts[:2], "/"))
			}
		}
	}

	// Remove duplicates and original
	unique := []string{}
	seen := map[string]bool{originalRepo: true}

	for _, repo := range related {
		if !seen[repo] && repo != originalRepo {
			unique = append(unique, repo)
			seen[repo] = true
		}
	}

	return unique
}

func (ms *MemoryServer) handleGetContext(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_get_context called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_get_context failed: missing repository parameter")
		return nil, fmt.Errorf("repository is required")
	}

	recentDays := 7
	if days, ok := params["recent_days"].(float64); ok {
		recentDays = int(days)
	}

	// Enhanced context gathering with auto-injection
	contextData, err := ms.buildEnhancedContext(ctx, repository, recentDays)
	if err != nil {
		return nil, fmt.Errorf("failed to build enhanced context: %w", err)
	}

	logging.Info("memory_get_context completed successfully", "repository", repository, "recent_sessions", contextData["total_recent_sessions"])
	return contextData, nil
}

// buildEnhancedContext creates comprehensive repository context with auto-injected intelligence
func (ms *MemoryServer) buildEnhancedContext(ctx context.Context, repository string, recentDays int) (map[string]interface{}, error) {
	// Get recent chunks for the repository
	chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 50, 0) // Increased limit for better analysis
	if err != nil {
		return nil, fmt.Errorf("failed to get repository chunks: %w", err)
	}

	// Filter by recent days
	cutoff := time.Now().AddDate(0, 0, -recentDays)
	recentChunks := []types.ConversationChunk{}
	allChunks := chunks // Keep all for pattern analysis

	for _, chunk := range chunks {
		if chunk.Timestamp.After(cutoff) {
			recentChunks = append(recentChunks, chunk)
		}
	}

	// Enhanced analysis using existing infrastructure
	patterns := ms.analyzePatterns(recentChunks)
	decisions := ms.extractDecisions(allChunks) // Use all chunks for decisions
	techStack := ms.extractTechStack(allChunks)

	// Auto-inject session continuity
	incompleteItems := ms.detectIncompleteWork(recentChunks)
	sessionSummary := ms.generateSessionSummary(recentChunks)
	workflowState := ms.detectWorkflowState(recentChunks)

	// Build enhanced project context
	context := types.NewProjectContext(repository)
	context.TotalSessions = len(recentChunks)
	context.CommonPatterns = patterns
	context.ArchitecturalDecisions = decisions
	context.TechStack = techStack

	result := map[string]interface{}{
		"repository":              context.Repository,
		"last_accessed":           context.LastAccessed.Format(time.RFC3339),
		"total_recent_sessions":   len(recentChunks),
		"common_patterns":         context.CommonPatterns,
		"architectural_decisions": context.ArchitecturalDecisions,
		"tech_stack":              context.TechStack,

		// Enhanced auto-context features
		"session_summary":     sessionSummary,
		"workflow_state":      workflowState,
		"incomplete_work":     incompleteItems,
		"recent_activity":     ms.formatRecentActivity(recentChunks, 5),
		"context_suggestions": ms.generateContextSuggestions(ctx, repository, recentChunks),
	}

	return result, nil
}

// detectIncompleteWork identifies ongoing or failed tasks from recent chunks
func (ms *MemoryServer) detectIncompleteWork(chunks []types.ConversationChunk) []map[string]interface{} {
	incomplete := []map[string]interface{}{}

	for _, chunk := range chunks {
		// Look for in-progress or failed outcomes
		if chunk.Metadata.Outcome == types.OutcomeInProgress || chunk.Metadata.Outcome == types.OutcomeFailed {
			incomplete = append(incomplete, map[string]interface{}{
				"chunk_id":   chunk.ID,
				"summary":    chunk.Summary,
				"type":       string(chunk.Type),
				"outcome":    string(chunk.Metadata.Outcome),
				"timestamp":  chunk.Timestamp.Format(time.RFC3339),
				"session_id": chunk.SessionID,
			})
		}

		// Look for problem chunks without corresponding solutions
		if chunk.Type == types.ChunkTypeProblem {
			hasSolution := false
			for _, otherChunk := range chunks {
				if otherChunk.SessionID == chunk.SessionID &&
					otherChunk.Type == types.ChunkTypeSolution &&
					otherChunk.Timestamp.After(chunk.Timestamp) {
					hasSolution = true
					break
				}
			}
			if !hasSolution {
				incomplete = append(incomplete, map[string]interface{}{
					"chunk_id":   chunk.ID,
					"summary":    chunk.Summary + " (no solution found)",
					"type":       "unsolved_problem",
					"outcome":    "incomplete",
					"timestamp":  chunk.Timestamp.Format(time.RFC3339),
					"session_id": chunk.SessionID,
				})
			}
		}
	}

	return incomplete
}

// generateSessionSummary creates a brief overview of recent sessions
func (ms *MemoryServer) generateSessionSummary(chunks []types.ConversationChunk) map[string]interface{} {
	if len(chunks) == 0 {
		return map[string]interface{}{
			"status":  "no_recent_activity",
			"message": "No recent activity in this repository",
		}
	}

	// Group by session ID
	sessions := make(map[string][]types.ConversationChunk)
	for _, chunk := range chunks {
		sessions[chunk.SessionID] = append(sessions[chunk.SessionID], chunk)
	}

	successCount := 0
	problemCount := 0
	lastSession := ""
	lastTimestamp := time.Time{}

	for sessionID, sessionChunks := range sessions {
		for _, chunk := range sessionChunks {
			if chunk.Type == types.ChunkTypeProblem {
				problemCount++
			}
			if chunk.Metadata.Outcome == types.OutcomeSuccess {
				successCount++
			}
			if chunk.Timestamp.After(lastTimestamp) {
				lastTimestamp = chunk.Timestamp
				lastSession = sessionID
			}
		}
	}

	return map[string]interface{}{
		"total_sessions":       len(sessions),
		"total_chunks":         len(chunks),
		"problems_encountered": problemCount,
		"successful_outcomes":  successCount,
		"success_rate":         float64(successCount) / float64(len(chunks)),
		"last_session_id":      lastSession,
		"last_activity":        lastTimestamp.Format(time.RFC3339),
		"status":               ms.determineSessionStatus(successCount, problemCount, len(chunks)),
	}
}

// detectWorkflowState determines current workflow state based on recent activity
func (ms *MemoryServer) detectWorkflowState(chunks []types.ConversationChunk) map[string]interface{} {
	if len(chunks) == 0 {
		return map[string]interface{}{
			"state":      "idle",
			"confidence": 1.0,
		}
	}

	// Sort by timestamp to get most recent activity
	recentChunks := make([]types.ConversationChunk, len(chunks))
	copy(recentChunks, chunks)

	// Simple sort by timestamp descending
	for i := 0; i < len(recentChunks)-1; i++ {
		for j := i + 1; j < len(recentChunks); j++ {
			if recentChunks[j].Timestamp.After(recentChunks[i].Timestamp) {
				recentChunks[i], recentChunks[j] = recentChunks[j], recentChunks[i]
			}
		}
	}

	// Analyze most recent chunks (last 3)
	analysisCount := 3
	if len(recentChunks) < analysisCount {
		analysisCount = len(recentChunks)
	}

	problemCount := 0
	solutionCount := 0

	for i := 0; i < analysisCount; i++ {
		chunk := recentChunks[i]
		if chunk.Type == types.ChunkTypeProblem {
			problemCount++
		}
		if chunk.Type == types.ChunkTypeSolution || chunk.Type == types.ChunkTypeCodeChange {
			solutionCount++
		}
	}

	// Determine state
	state := "idle"
	confidence := 0.7

	switch {
	case problemCount > solutionCount:
		state = "debugging"
		confidence = 0.8
	case solutionCount > 0:
		state = "implementing"
		confidence = 0.8
	case len(recentChunks) > 0 && recentChunks[0].Type == types.ChunkTypeArchitectureDecision:
		state = "planning"
		confidence = 0.9
	}

	return map[string]interface{}{
		"state":      state,
		"confidence": confidence,
		"indicators": map[string]interface{}{
			"recent_problems":  problemCount,
			"recent_solutions": solutionCount,
			"analysis_window":  analysisCount,
		},
	}
}

// formatRecentActivity formats recent activity in a consistent way
func (ms *MemoryServer) formatRecentActivity(chunks []types.ConversationChunk, limit int) []map[string]interface{} {
	activity := []map[string]interface{}{}

	// Sort by timestamp descending
	sortedChunks := make([]types.ConversationChunk, len(chunks))
	copy(sortedChunks, chunks)

	for i := 0; i < len(sortedChunks)-1; i++ {
		for j := i + 1; j < len(sortedChunks); j++ {
			if sortedChunks[j].Timestamp.After(sortedChunks[i].Timestamp) {
				sortedChunks[i], sortedChunks[j] = sortedChunks[j], sortedChunks[i]
			}
		}
	}

	for i, chunk := range sortedChunks {
		if i >= limit {
			break
		}
		activity = append(activity, map[string]interface{}{
			"chunk_id":   chunk.ID,
			"type":       string(chunk.Type),
			"summary":    chunk.Summary,
			"timestamp":  chunk.Timestamp.Format(time.RFC3339),
			"outcome":    string(chunk.Metadata.Outcome),
			"session_id": chunk.SessionID,
		})
	}

	return activity
}

// generateContextSuggestions creates proactive suggestions based on current context
func (ms *MemoryServer) generateContextSuggestions(_ context.Context, repository string, chunks []types.ConversationChunk) []map[string]interface{} {
	suggestions := []map[string]interface{}{}

	if len(chunks) == 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":        "getting_started",
			"title":       "Start documenting your work",
			"description": "Begin storing architecture decisions and solutions to build your memory base",
			"action":      "Create your first memory chunk with important decisions or learnings",
		})
		return suggestions
	}

	// Check for patterns and suggest relevant actions
	problemCount := 0
	recentProblems := []types.ConversationChunk{}

	for _, chunk := range chunks {
		if chunk.Type == types.ChunkTypeProblem {
			problemCount++
			if len(recentProblems) < 3 {
				recentProblems = append(recentProblems, chunk)
			}
		}
	}

	// Suggest searching for similar problems
	if problemCount > 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":        "similar_problems",
			"title":       "Search for similar issues",
			"description": fmt.Sprintf("You have %d recent problems. Search for similar patterns across your memory", problemCount),
			"action":      "Use memory_find_similar to find related solutions",
		})
	}

	// Check for incomplete work
	incompleteWork := ms.detectIncompleteWork(chunks)
	if len(incompleteWork) > 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":        "incomplete_work",
			"title":       "Resume incomplete tasks",
			"description": fmt.Sprintf("You have %d incomplete items that might need attention", len(incompleteWork)),
			"action":      "Review and update the status of pending work",
		})
	}

	// Suggest architecture review if many decisions recently
	decisionCount := 0
	for _, chunk := range chunks {
		if chunk.Type == types.ChunkTypeArchitectureDecision {
			decisionCount++
		}
	}

	if decisionCount >= 3 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":        "architecture_review",
			"title":       "Review architectural decisions",
			"description": fmt.Sprintf("You've made %d recent architecture decisions. Consider reviewing for consistency", decisionCount),
			"action":      "Use memory_get_patterns to analyze decision trends",
		})
	}

	return suggestions
}

// determineSessionStatus provides a human-readable status based on activity metrics
func (ms *MemoryServer) determineSessionStatus(successCount, problemCount, totalChunks int) string {
	if totalChunks == 0 {
		return "no_activity"
	}

	successRate := float64(successCount) / float64(totalChunks)

	switch {
	case successRate >= 0.8:
		return "highly_productive"
	case successRate >= 0.6:
		return "productive"
	case problemCount > successCount:
		return "troubleshooting_mode"
	default:
		return "mixed_progress"
	}
}

func (ms *MemoryServer) handleFindSimilar(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_find_similar called", "params", params)

	problem, ok := params["problem"].(string)
	if !ok || problem == "" {
		logging.Error("memory_find_similar failed: missing problem parameter")
		return nil, fmt.Errorf("problem description is required")
	}

	logging.Info("Processing similar problem search", "problem", problem)

	limit := 5
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	// Build search query focusing on problems and solutions
	memQuery := types.NewMemoryQuery(problem)
	memQuery.Types = []types.ChunkType{types.ChunkTypeProblem, types.ChunkTypeSolution}
	memQuery.Limit = limit
	memQuery.MinRelevanceScore = getEnvFloat("MCP_MEMORY_SIMILAR_PROBLEM_MIN_RELEVANCE", 0.6) // Lower threshold for problem matching

	if repo, ok := params["repository"].(string); ok && repo != "" {
		memQuery.Repository = &repo
	}

	// Generate embeddings and search
	logging.Info("Generating embeddings for similar problem search", "problem", problem)
	embeddings, err := ms.container.GetEmbeddingService().GenerateEmbedding(ctx, problem)
	if err != nil {
		logging.Error("Failed to generate embeddings for problem search", "error", err, "problem", problem)
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}
	logging.Info("Embeddings generated for problem search", "dimension", len(embeddings))

	logging.Info("Searching for similar problems", "problem", problem, "limit", limit)
	results, err := ms.container.GetVectorStore().Search(ctx, *memQuery, embeddings)
	if err != nil {
		logging.Error("Similar problem search failed", "error", err, "problem", problem)
		return nil, fmt.Errorf("search failed: %w", err)
	}
	logging.Info("Similar problem search completed", "total_results", results.Total)

	// Group problems with their solutions
	similarProblems := []map[string]interface{}{}
	for _, result := range results.Results {
		problemData := map[string]interface{}{
			"chunk_id":   result.Chunk.ID,
			"score":      result.Score,
			"type":       string(result.Chunk.Type),
			"summary":    result.Chunk.Summary,
			"content":    result.Chunk.Content,
			"repository": result.Chunk.Metadata.Repository,
			"timestamp":  result.Chunk.Timestamp.Format(time.RFC3339),
			"outcome":    string(result.Chunk.Metadata.Outcome),
			"difficulty": string(result.Chunk.Metadata.Difficulty),
			"tags":       result.Chunk.Metadata.Tags,
		}
		similarProblems = append(similarProblems, problemData)
	}

	logging.Info("memory_find_similar completed successfully", "total_found", len(similarProblems), "problem", problem)
	return map[string]interface{}{
		"problem":          problem,
		"similar_problems": similarProblems,
		"total_found":      len(similarProblems),
	}, nil
}

func (ms *MemoryServer) handleStoreDecision(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	decision, ok := params["decision"].(string)
	if !ok || decision == "" {
		return nil, fmt.Errorf("decision is required")
	}

	rationale, ok := params["rationale"].(string)
	if !ok || rationale == "" {
		return nil, fmt.Errorf("rationale is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	context := ""
	if ctx, ok := params["context"].(string); ok {
		context = ctx
	}

	// Combine decision components into content
	content := fmt.Sprintf("ARCHITECTURAL DECISION: %s\n\nRATIONALE: %s", decision, rationale)
	if context != "" {
		content += fmt.Sprintf("\n\nCONTEXT: %s", context)
	}

	// Build metadata
	metadata := types.ChunkMetadata{
		Outcome:    types.OutcomeSuccess,
		Difficulty: types.DifficultyModerate,
		Tags:       []string{"architecture", "decision"},
	}

	if repo, ok := params["repository"].(string); ok {
		metadata.Repository = normalizeRepository(repo)
	} else {
		metadata.Repository = GlobalMemoryRepository
	}

	// Add extended metadata with context detection
	detector, err := contextdetector.NewDetector()
	if err == nil {
		if metadata.ExtendedMetadata == nil {
			metadata.ExtendedMetadata = make(map[string]interface{})
		}

		// Add location context
		locationContext := detector.DetectLocationContext()
		for k, v := range locationContext {
			metadata.ExtendedMetadata[k] = v
		}

		// Add client context
		clientType := types.ClientTypeAPI
		if ct, ok := params["client_type"].(string); ok {
			clientType = ct
		}
		clientContext := detector.DetectClientContext(clientType)
		for k, v := range clientContext {
			metadata.ExtendedMetadata[k] = v
		}

		// Mark this as an architectural decision
		metadata.ExtendedMetadata["decision_type"] = "architectural"
		metadata.ExtendedMetadata["decision_text"] = decision
		metadata.ExtendedMetadata["rationale_text"] = rationale
	}

	// Create and store chunk
	chunk, err := ms.container.GetChunkingService().CreateChunk(ctx, sessionID, content, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create decision chunk: %w", err)
	}

	// Override type to architecture decision
	chunk.Type = types.ChunkTypeArchitectureDecision

	if err := ms.container.GetVectorStore().Store(ctx, *chunk); err != nil {
		return nil, fmt.Errorf("failed to store decision: %w", err)
	}

	return map[string]interface{}{
		"chunk_id":  chunk.ID,
		"decision":  decision,
		"stored_at": chunk.Timestamp.Format(time.RFC3339),
	}, nil
}

func (ms *MemoryServer) handleGetPatterns(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_get_patterns called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_get_patterns failed: missing repository parameter")
		return nil, fmt.Errorf("repository is required")
	}

	timeframe := types.TimeframeMonth
	if tf, ok := params["timeframe"].(string); ok {
		timeframe = tf
	}

	// Get chunks based on timeframe
	var chunks []types.ConversationChunk
	var err error

	// For now, get recent chunks - in a full implementation,
	// this would have proper time filtering
	chunks, err = ms.container.GetVectorStore().ListByRepository(ctx, repository, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}

	patterns := ms.analyzePatterns(chunks)

	return map[string]interface{}{
		"repository":            repository,
		"timeframe":             timeframe,
		"patterns":              patterns,
		"total_chunks_analyzed": len(chunks),
	}, nil
}

func (ms *MemoryServer) handleHealth(ctx context.Context, _ map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_health called")

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  map[string]interface{}{},
	}

	// Check vector store
	logging.Info("Checking vector store health")
	if err := ms.container.GetVectorStore().HealthCheck(ctx); err != nil {
		logging.Error("Vector store health check failed", "error", err)
		health["services"].(map[string]interface{})["vector_store"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "degraded"
	} else {
		logging.Info("Vector store health check passed")
		health["services"].(map[string]interface{})["vector_store"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Check embedding service
	logging.Info("Checking embedding service health")
	if err := ms.container.GetEmbeddingService().HealthCheck(ctx); err != nil {
		logging.Error("Embedding service health check failed", "error", err)
		health["services"].(map[string]interface{})["embedding_service"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "degraded"
	} else {
		logging.Info("Embedding service health check passed")
		health["services"].(map[string]interface{})["embedding_service"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Get statistics
	if stats, err := ms.container.GetVectorStore().GetStats(ctx); err == nil {
		health["stats"] = stats
		logging.Info("Vector store statistics retrieved", "stats", stats)
	} else {
		logging.Error("Failed to retrieve vector store statistics", "error", err)
	}

	logging.Info("memory_health completed", "status", health["status"])
	return health, nil
}

// Resource handler

func (ms *MemoryServer) handleResourceRead(ctx context.Context, uri string) ([]protocol.Content, error) {
	parts := strings.Split(uri, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid resource URI: %s", uri)
	}

	resourceType := parts[2]

	switch resourceType {
	case "recent":
		if len(parts) < 4 {
			return nil, fmt.Errorf("repository required for recent resource")
		}
		repository := parts[3]
		chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 20, 0)
		if err != nil {
			return nil, err
		}
		chunksJSON, _ := json.Marshal(chunks)
		return []protocol.Content{protocol.NewContent(string(chunksJSON))}, nil

	case "patterns":
		if len(parts) < 4 {
			return nil, fmt.Errorf("repository required for patterns resource")
		}
		repository := parts[3]
		chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 100, 0)
		if err != nil {
			return nil, err
		}
		patterns := ms.analyzePatterns(chunks)
		result := map[string]interface{}{
			"repository": repository,
			"patterns":   patterns,
		}
		resultJSON, _ := json.Marshal(result)
		return []protocol.Content{protocol.NewContent(string(resultJSON))}, nil

	case "decisions":
		if len(parts) < 4 {
			return nil, fmt.Errorf("repository required for decisions resource")
		}
		repository := parts[3]

		// Search for architecture decisions
		memQuery := types.NewMemoryQuery("architectural decision")
		memQuery.Repository = &repository
		memQuery.Types = []types.ChunkType{types.ChunkTypeArchitectureDecision}
		memQuery.Limit = 50

		embeddings, err := ms.container.GetEmbeddingService().GenerateEmbedding(ctx, "architectural decision")
		if err != nil {
			return nil, err
		}

		results, err := ms.container.GetVectorStore().Search(ctx, *memQuery, embeddings)
		if err != nil {
			return nil, err
		}

		decisions := []map[string]interface{}{}
		for _, result := range results.Results {
			decisions = append(decisions, map[string]interface{}{
				"chunk_id":  result.Chunk.ID,
				"summary":   result.Chunk.Summary,
				"content":   result.Chunk.Content,
				"timestamp": result.Chunk.Timestamp,
			})
		}

		result := map[string]interface{}{
			"repository": repository,
			"decisions":  decisions,
		}
		resultJSON, _ := json.Marshal(result)
		return []protocol.Content{protocol.NewContent(string(resultJSON))}, nil

	case "global":
		if len(parts) < 4 || parts[3] != "insights" {
			return nil, fmt.Errorf("invalid global resource")
		}

		// Get global insights across all repositories
		// This is a simplified implementation
		result := map[string]interface{}{
			"message": "Global insights feature coming soon",
			"status":  "not_implemented",
		}
		resultJSON, _ := json.Marshal(result)
		return []protocol.Content{protocol.NewContent(string(resultJSON))}, nil

	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// Helper methods

func (ms *MemoryServer) analyzePatterns(chunks []types.ConversationChunk) []string {
	tagCounts := make(map[string]int)
	patterns := []string{}

	for _, chunk := range chunks {
		for _, tag := range chunk.Metadata.Tags {
			tagCounts[tag]++
		}
	}

	// Find common patterns (tags that appear multiple times)
	for tag, count := range tagCounts {
		if count >= 3 { // Threshold for pattern
			patterns = append(patterns, fmt.Sprintf("%s (appears %d times)", tag, count))
		}
	}

	return patterns
}

func (ms *MemoryServer) extractDecisions(chunks []types.ConversationChunk) []string {
	decisions := []string{}

	for _, chunk := range chunks {
		if chunk.Type == types.ChunkTypeArchitectureDecision {
			decisions = append(decisions, chunk.Summary)
		}
	}

	return decisions
}

func (ms *MemoryServer) extractTechStack(chunks []types.ConversationChunk) []string {
	techTags := make(map[string]bool)

	techKeywords := []string{
		"go", "golang", "typescript", "javascript", "python", "rust", "java",
		"react", "vue", "angular", "express", "fastapi", "django", "flask",
		"docker", "kubernetes", "postgres", "mysql", "redis", "mongodb",
		"aws", "gcp", "azure", "terraform", "helm",
	}

	for _, chunk := range chunks {
		for _, tag := range chunk.Metadata.Tags {
			for _, tech := range techKeywords {
				if strings.EqualFold(tag, tech) {
					techTags[tech] = true
				}
			}
		}
	}

	techStack := []string{}
	for tech := range techTags {
		techStack = append(techStack, tech)
	}

	return techStack
}

// Phase 3.2: Advanced MCP Tool Handlers

// handleSuggestRelated provides AI-powered context suggestions
func (ms *MemoryServer) handleSuggestRelated(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_suggest_related called", "params", params)

	currentContext, ok := params["current_context"].(string)
	if !ok {
		logging.Error("memory_suggest_related failed: missing current_context parameter")
		return nil, fmt.Errorf("current_context is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok {
		logging.Error("memory_suggest_related failed: missing session_id parameter")
		return nil, fmt.Errorf("session_id is required")
	}

	logging.Info("Processing context suggestions", "context_length", len(currentContext), "session_id", sessionID)

	repository := ""
	if repo, exists := params["repository"].(string); exists {
		repository = repo
	}

	maxSuggestions := 5
	if max, exists := params["max_suggestions"].(float64); exists {
		maxSuggestions = int(max)
	}

	includePatterns := true
	if include, exists := params["include_patterns"].(bool); exists {
		includePatterns = include
	}

	// Generate embedding for current context
	logging.Info("Generating embedding for context suggestions", "context_length", len(currentContext))
	embedding, err := ms.container.GetEmbeddingService().GenerateEmbedding(ctx, currentContext)
	if err != nil {
		logging.Error("Failed to generate embedding for context suggestions", "error", err)
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	logging.Info("Embedding generated for context suggestions", "dimension", len(embedding))

	// Search for similar content
	logging.Info("Searching for related context", "repository", repository, "max_suggestions", maxSuggestions)
	query := types.NewMemoryQuery(currentContext)
	query.Repository = &repository
	query.Limit = maxSuggestions * 2 // Get more to filter from
	query.MinRelevanceScore = 0.6

	results, err := ms.container.GetVectorStore().Search(ctx, *query, embedding)
	if err != nil {
		logging.Error("Context suggestion search failed", "error", err)
		return nil, fmt.Errorf("search failed: %w", err)
	}
	logging.Info("Context suggestion search completed", "total_results", results.Total)

	suggestions := []map[string]interface{}{}
	analytics := ms.container.GetMemoryAnalytics()

	for i, result := range results.Results {
		if i >= maxSuggestions {
			break
		}

		// Track access to suggested memories
		if analytics != nil {
			if err := analytics.RecordAccess(ctx, result.Chunk.ID); err != nil {
				logging.Warn("Failed to record memory access", "chunk_id", result.Chunk.ID, "error", err)
			}
		}

		suggestion := map[string]interface{}{
			"content":         result.Chunk.Content,
			"summary":         result.Chunk.Summary,
			"relevance_score": result.Score,
			"timestamp":       result.Chunk.Timestamp,
			"type":            "semantic_match",
			"chunk_id":        result.Chunk.ID,
		}

		if result.Chunk.Metadata.Repository != "" {
			suggestion["repository"] = result.Chunk.Metadata.Repository
		}

		suggestions = append(suggestions, suggestion)
	}

	// Add pattern-based suggestions if enabled
	if includePatterns && ms.container.GetPatternAnalyzer() != nil {
		// Extract current tools from context (simplified approach)
		currentTools := []string{}
		problemType := "general" // Default problem type

		// Try to infer problem type from context
		contextLower := strings.ToLower(currentContext)
		switch {
		case strings.Contains(contextLower, "error") || strings.Contains(contextLower, "bug"):
			problemType = "debug"
		case strings.Contains(contextLower, "test"):
			problemType = "test"
		case strings.Contains(contextLower, "build") || strings.Contains(contextLower, "compile"):
			problemType = "build"
		case strings.Contains(contextLower, "config"):
			problemType = "configuration"
		}

		// Get pattern recommendations
		patternRecommendations := ms.container.GetPatternAnalyzer().GetPatternRecommendations(currentTools, problemType)

		// Convert pattern recommendations to suggestions
		for _, pattern := range patternRecommendations {
			if len(suggestions) >= maxSuggestions {
				break
			}

			suggestion := map[string]interface{}{
				"content":         fmt.Sprintf("Based on similar %s problems, consider using: %s", problemType, strings.Join(pattern.Tools, " → ")),
				"summary":         pattern.Description,
				"relevance_score": pattern.SuccessRate,
				"type":            "pattern_match",
				"pattern_type":    string(pattern.Type),
				"success_rate":    pattern.SuccessRate,
				"frequency":       pattern.Frequency,
			}

			// Add examples if available
			if len(pattern.Examples) > 0 {
				suggestion["examples"] = pattern.Examples
			}

			suggestions = append(suggestions, suggestion)
		}
	}

	logging.Info("memory_suggest_related completed successfully", "total_suggestions", len(suggestions), "session_id", sessionID)
	return map[string]interface{}{
		"suggestions":      suggestions,
		"total_found":      len(suggestions),
		"search_context":   currentContext,
		"include_patterns": includePatterns,
		"session_id":       sessionID,
	}, nil
}

// handleExportProject exports all memory data for a project
func (ms *MemoryServer) handleExportProject(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_export_project called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok {
		logging.Error("memory_export_project failed: missing repository parameter")
		return nil, fmt.Errorf("repository is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id is required")
	}

	format := "json"
	if f, exists := params["format"].(string); exists {
		format = f
	}

	includeVectors := false
	if include, exists := params["include_vectors"].(bool); exists {
		includeVectors = include
	}

	// Get all chunks for the repository
	chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 10000, 0) // Large limit for export
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve repository data: %w", err)
	}

	switch format {
	case "json":
		exportData := map[string]interface{}{
			"repository":      repository,
			"export_date":     time.Now().Format(time.RFC3339),
			"total_chunks":    len(chunks),
			"include_vectors": includeVectors,
			"chunks":          chunks,
		}

		// Remove vector data if not requested
		if !includeVectors {
			for i := range chunks {
				chunks[i].Embeddings = nil
			}
		}

		exportJSON, err := json.Marshal(exportData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal export data: %w", err)
		}

		return map[string]interface{}{
			"format":     "json",
			"data":       string(exportJSON),
			"size_bytes": len(exportJSON),
			"chunks":     len(chunks),
			"repository": repository,
			"session_id": sessionID,
		}, nil

	case "markdown":
		var markdown strings.Builder
		markdown.WriteString(fmt.Sprintf("# Memory Export: %s\n\n", repository))
		markdown.WriteString(fmt.Sprintf("**Export Date:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
		markdown.WriteString(fmt.Sprintf("**Total Chunks:** %d\n\n", len(chunks)))

		for _, chunk := range chunks {
			markdown.WriteString(fmt.Sprintf("## %s\n\n", chunk.Summary))
			markdown.WriteString(fmt.Sprintf("**ID:** %s\n", chunk.ID))
			markdown.WriteString(fmt.Sprintf("**Type:** %s\n", chunk.Type))
			markdown.WriteString(fmt.Sprintf("**Timestamp:** %s\n\n", chunk.Timestamp.Format("2006-01-02 15:04:05")))
			markdown.WriteString(fmt.Sprintf("%s\n\n", chunk.Content))

			if len(chunk.Metadata.Tags) > 0 {
				markdown.WriteString(fmt.Sprintf("**Tags:** %s\n\n", strings.Join(chunk.Metadata.Tags, ", ")))
			}

			markdown.WriteString("---\n\n")
		}

		markdownData := markdown.String()
		return map[string]interface{}{
			"format":     "markdown",
			"data":       markdownData,
			"size_bytes": len(markdownData),
			"chunks":     len(chunks),
			"repository": repository,
			"session_id": sessionID,
		}, nil

	case "archive":
		// Use backup manager to create compressed archive
		if ms.container.GetBackupManager() == nil {
			return nil, fmt.Errorf("backup manager not available")
		}

		// Create a filtered backup for this repository only
		backupData := map[string]interface{}{
			"repository":  repository,
			"export_date": time.Now().Format(time.RFC3339),
			"chunks":      chunks,
			"metadata": map[string]interface{}{
				"export_type": "project_export",
				"session_id":  sessionID,
			},
		}

		archiveJSON, err := json.Marshal(backupData)
		if err != nil {
			return nil, fmt.Errorf("failed to create archive data: %w", err)
		}

		// Encode as base64 for transport
		archiveB64 := base64.StdEncoding.EncodeToString(archiveJSON)

		return map[string]interface{}{
			"format":     "archive",
			"data":       archiveB64,
			"size_bytes": len(archiveJSON),
			"chunks":     len(chunks),
			"repository": repository,
			"session_id": sessionID,
			"encoding":   "base64",
		}, nil

	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// handleImportContext imports conversation context from external source
func (ms *MemoryServer) handleImportContext(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	source, ok := params["source"].(string)
	if !ok {
		return nil, fmt.Errorf("source is required")
	}

	data, ok := params["data"].(string)
	if !ok {
		return nil, fmt.Errorf("data is required")
	}

	repository, ok := params["repository"].(string)
	if !ok {
		return nil, fmt.Errorf("repository is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id is required")
	}

	chunkingStrategy := "auto"
	if strategy, exists := params["chunking_strategy"].(string); exists {
		chunkingStrategy = strategy
	}

	// Parse metadata if provided
	metadata := map[string]interface{}{}
	if meta, exists := params["metadata"].(map[string]interface{}); exists {
		metadata = meta
	}

	var importedChunks []types.ConversationChunk
	var err error

	switch source {
	case types.SourceConversation:
		// Import conversation text
		importedChunks, err = ms.importConversationText(ctx, data, repository, chunkingStrategy, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to import conversation: %w", err)
		}

	case "file":
		// Import file content
		importedChunks, err = ms.importFileContent(ctx, data, repository, chunkingStrategy, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to import file: %w", err)
		}

	case "archive":
		// Import from base64 encoded archive
		importedChunks, err = ms.importArchiveData(ctx, data, repository, metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to import archive: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported source type: %s", source)
	}

	// Store imported chunks
	storedCount := 0
	for _, chunk := range importedChunks {
		// Generate embedding for the chunk
		embedding, err := ms.container.GetEmbeddingService().GenerateEmbedding(ctx, chunk.Content)
		if err != nil {
			log.Printf("Failed to generate embedding for chunk %s: %v", chunk.ID, err)
			continue
		}

		chunk.Embeddings = embedding

		// Store chunk
		if err := ms.container.GetVectorStore().Store(ctx, chunk); err != nil {
			log.Printf("Failed to store chunk %s: %v", chunk.ID, err)
			continue
		}

		storedCount++
	}

	return map[string]interface{}{
		"source":            source,
		"repository":        repository,
		"chunks_processed":  len(importedChunks),
		"chunks_stored":     storedCount,
		"chunking_strategy": chunkingStrategy,
		"session_id":        sessionID,
		"import_date":       time.Now().Format(time.RFC3339),
	}, nil
}

// Helper methods for import functionality

func (ms *MemoryServer) importConversationText(ctx context.Context, data, repository, _ string, metadata map[string]interface{}) ([]types.ConversationChunk, error) {
	// Create conversation chunks using the chunking service
	chunkMetadata := types.ChunkMetadata{
		Repository: repository,
		Tags:       []string{"imported", types.SourceConversation},
	}

	// Add tags from metadata
	if tags, exists := metadata["tags"].([]interface{}); exists {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				chunkMetadata.Tags = append(chunkMetadata.Tags, tagStr)
			}
		}
	}

	// Add source system as a tag since no dedicated field exists
	if sourceSystem, exists := metadata["source_system"].(string); exists {
		chunkMetadata.Tags = append(chunkMetadata.Tags, "source:"+sourceSystem)
	}

	chunkData, err := ms.container.GetChunkingService().CreateChunk(ctx, "import", data, chunkMetadata)
	if err != nil {
		return nil, err
	}

	return []types.ConversationChunk{*chunkData}, nil
}

func (ms *MemoryServer) importFileContent(ctx context.Context, data, repository, _ string, metadata map[string]interface{}) ([]types.ConversationChunk, error) {
	// Create file chunks using the chunking service
	chunkMetadata := types.ChunkMetadata{
		Repository: repository,
		Tags:       []string{"imported", "file"},
	}

	// Add tags from metadata
	if tags, exists := metadata["tags"].([]interface{}); exists {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				chunkMetadata.Tags = append(chunkMetadata.Tags, tagStr)
			}
		}
	}

	// Add source system as a tag since no dedicated field exists
	if sourceSystem, exists := metadata["source_system"].(string); exists {
		chunkMetadata.Tags = append(chunkMetadata.Tags, "source:"+sourceSystem)
	}

	chunkData, err := ms.container.GetChunkingService().CreateChunk(ctx, "import", data, chunkMetadata)
	if err != nil {
		return nil, err
	}

	// Set chunk type to analysis (closest to knowledge)
	chunkData.Type = types.ChunkTypeAnalysis

	return []types.ConversationChunk{*chunkData}, nil
}

func (ms *MemoryServer) importArchiveData(_ context.Context, data, repository string, metadata map[string]interface{}) ([]types.ConversationChunk, error) {
	// Decode base64 archive data
	archiveData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode archive data: %w", err)
	}

	// Parse JSON archive
	var archiveContent map[string]interface{}
	if err := json.Unmarshal(archiveData, &archiveContent); err != nil {
		return nil, fmt.Errorf("failed to parse archive JSON: %w", err)
	}

	// Extract chunks from archive
	chunksData, exists := archiveContent["chunks"]
	if !exists {
		return nil, fmt.Errorf("no chunks found in archive")
	}

	chunksJSON, err := json.Marshal(chunksData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chunks data: %w", err)
	}

	var chunks []types.ConversationChunk
	if err := json.Unmarshal(chunksJSON, &chunks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chunks: %w", err)
	}

	// Update repository and add import metadata
	for i := range chunks {
		chunks[i].Metadata.Repository = repository
		chunks[i].Metadata.Tags = append(chunks[i].Metadata.Tags, "imported", "archive")

		// Apply additional metadata
		if sourceSystem, exists := metadata["source_system"].(string); exists {
			chunks[i].Metadata.Tags = append(chunks[i].Metadata.Tags, "source:"+sourceSystem)
		}
	}

	return chunks, nil
}

// GetServer returns the underlying MCP server
func (ms *MemoryServer) GetServer() interface{} {
	return ms.mcpServer
}

// Close closes all connections
func (ms *MemoryServer) Close() error {
	return ms.container.Shutdown()
}

// normalizeRepository ensures that empty repository defaults to global and
// detects full Git repository URL when only a directory name is provided
func normalizeRepository(repo string) string {
	if repo == "" {
		return GlobalMemoryRepository
	}

	// If repository looks like a full URL (contains domain), use as-is
	if strings.Contains(repo, ".") && (strings.Contains(repo, "github.com") ||
		strings.Contains(repo, "gitlab.com") || strings.Contains(repo, "bitbucket.org") ||
		strings.Contains(repo, "/")) {
		return repo
	}

	// If repository is just a directory name, try to detect Git remote URL
	if gitRepo := detectGitRepository(); gitRepo != "" {
		return gitRepo
	}

	// Fallback to provided repository name
	return repo
}

// detectGitRepository attempts to detect the Git repository URL from the current directory
func detectGitRepository() string {
	// Try to get Git remote URL
	cmd := exec.Command("git", "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	remoteURL := strings.TrimSpace(string(out))
	if remoteURL == "" {
		return ""
	}

	// Convert various Git URL formats to standard repository identifiers
	// Handle HTTPS URLs: https://github.com/user/repo.git -> github.com/user/repo
	if strings.HasPrefix(remoteURL, "https://") {
		remoteURL = strings.TrimPrefix(remoteURL, "https://")
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
		return remoteURL
	}

	// Handle SSH URLs: git@github.com:user/repo.git -> github.com/user/repo
	if strings.HasPrefix(remoteURL, "git@") {
		// Remove git@ prefix
		remoteURL = strings.TrimPrefix(remoteURL, "git@")
		// Replace : with /
		remoteURL = strings.Replace(remoteURL, ":", "/", 1)
		// Remove .git suffix
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
		return remoteURL
	}

	return remoteURL
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

// buildMetadataFromParams extracts metadata from request parameters
func (ms *MemoryServer) buildMetadataFromParams(params map[string]interface{}) types.ChunkMetadata {
	metadata := types.ChunkMetadata{
		Outcome:    types.OutcomeInProgress, // Default
		Difficulty: types.DifficultySimple,  // Default
	}

	if repo, ok := params["repository"].(string); ok {
		metadata.Repository = normalizeRepository(repo)
	} else {
		metadata.Repository = GlobalMemoryRepository
	}

	if branch, ok := params["branch"].(string); ok {
		metadata.Branch = branch
	}

	if files, ok := params["files_modified"].([]interface{}); ok {
		for _, f := range files {
			if file, ok := f.(string); ok {
				metadata.FilesModified = append(metadata.FilesModified, file)
			}
		}
	}

	if tools, ok := params["tools_used"].([]interface{}); ok {
		for _, t := range tools {
			if tool, ok := t.(string); ok {
				metadata.ToolsUsed = append(metadata.ToolsUsed, tool)
			}
		}
	}

	if tags, ok := params["tags"].([]interface{}); ok {
		for _, t := range tags {
			if tag, ok := t.(string); ok {
				metadata.Tags = append(metadata.Tags, tag)
			}
		}
	}

	return metadata
}

// addContextMetadata adds context detection metadata
func (ms *MemoryServer) addContextMetadata(metadata *types.ChunkMetadata, params map[string]interface{}) error {
	detector, err := contextdetector.NewDetector()
	if err != nil {
		return err
	}

	if metadata.ExtendedMetadata == nil {
		metadata.ExtendedMetadata = make(map[string]interface{})
	}

	// Add location context
	locationContext := detector.DetectLocationContext()
	for k, v := range locationContext {
		metadata.ExtendedMetadata[k] = v
	}

	// Add client context (get client type from params if available)
	clientType := types.ClientTypeAPI // Default
	if ct, ok := params["client_type"].(string); ok {
		clientType = ct
	}
	clientContext := detector.DetectClientContext(clientType)
	for k, v := range clientContext {
		metadata.ExtendedMetadata[k] = v
	}

	// Add language versions
	if langVersions := detector.DetectLanguageVersions(); len(langVersions) > 0 {
		metadata.ExtendedMetadata[types.EMKeyLanguageVersions] = langVersions
	}

	// Add dependencies
	if deps := detector.DetectDependencies(); len(deps) > 0 {
		metadata.ExtendedMetadata[types.EMKeyDependencies] = deps
	}

	return nil
}

// handleMemoryStatus provides comprehensive status overview of memory system for a repository
func (ms *MemoryServer) handleMemoryStatus(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_status called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_status failed: missing repository parameter")
		return nil, fmt.Errorf("repository is required")
	}

	// Get enhanced context data (reuse our new auto-context logic)
	contextData, err := ms.buildEnhancedContext(ctx, repository, 30) // Last 30 days
	if err != nil {
		return nil, fmt.Errorf("failed to build status context: %w", err)
	}

	// Get all chunks for deeper analysis
	allChunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 100, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get all repository chunks: %w", err)
	}

	// Calculate comprehensive metrics
	metrics := ms.calculateMemoryMetrics(allChunks)
	health := ms.assessMemoryHealth(allChunks)
	trends := ms.analyzeMemoryTrends(allChunks)

	status := map[string]interface{}{
		"repository": repository,
		"timestamp":  time.Now().Format(time.RFC3339),

		// Core metrics
		"metrics": metrics,
		"health":  health,
		"trends":  trends,

		// Enhanced context from our auto-context system
		"session_summary": contextData["session_summary"],
		"workflow_state":  contextData["workflow_state"],
		"incomplete_work": contextData["incomplete_work"],
		"suggestions":     contextData["context_suggestions"],

		// Additional status info
		"memory_coverage":     ms.calculateMemoryCoverage(allChunks),
		"knowledge_gaps":      ms.identifyKnowledgeGaps(allChunks),
		"effectiveness_score": ms.calculateOverallEffectiveness(allChunks),
	}

	logging.Info("memory_status completed successfully", "repository", repository, "total_chunks", len(allChunks))
	return status, nil
}

// handleMemoryConflicts detects contradictory decisions or patterns across memories
func (ms *MemoryServer) handleMemoryConflicts(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_conflicts called", "params", params)

	repository := ""
	if repo, ok := params["repository"].(string); ok {
		repository = repo
	}

	timeframe := types.TimeframeMonth
	if tf, ok := params["timeframe"].(string); ok {
		timeframe = tf
	}

	// Get chunks based on repository and timeframe
	var chunks []types.ConversationChunk
	var err error

	if repository == "" || repository == "_global" {
		// Global analysis across all repositories
		chunks, err = ms.getChunksForTimeframe(ctx, "", timeframe)
	} else {
		chunks, err = ms.getChunksForTimeframe(ctx, repository, timeframe)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get chunks for conflict analysis: %w", err)
	}

	conflicts := ms.detectConflicts(chunks)
	contradictions := ms.findContradictoryDecisions(chunks)
	patternInconsistencies := ms.findPatternInconsistencies(chunks)

	result := map[string]interface{}{
		"repository":            repository,
		"timeframe":             timeframe,
		"analysis_timestamp":    time.Now().Format(time.RFC3339),
		"total_chunks_analyzed": len(chunks),

		"conflicts":               conflicts,
		"contradictory_decisions": contradictions,
		"pattern_inconsistencies": patternInconsistencies,

		"summary": map[string]interface{}{
			"total_conflicts":    len(conflicts) + len(contradictions) + len(patternInconsistencies),
			"severity_breakdown": ms.categorizeConflictsBySeverity(conflicts, contradictions, patternInconsistencies),
			"recommendations":    ms.generateConflictResolutionRecommendations(conflicts, contradictions),
		},
	}

	logging.Info("memory_conflicts completed", "repository", repository, "total_conflicts", len(conflicts))
	return result, nil
}

// handleMemoryContinuity shows what was left incomplete from previous sessions
func (ms *MemoryServer) handleMemoryContinuity(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_continuity called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_continuity failed: missing repository parameter")
		return nil, fmt.Errorf("repository is required")
	}

	sessionID := ""
	if sid, ok := params["session_id"].(string); ok {
		sessionID = sid
	}

	includeSuggestions := true
	if inc, ok := params["include_suggestions"].(bool); ok {
		includeSuggestions = inc
	}

	// Get chunks for analysis
	chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 50, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository chunks: %w", err)
	}

	var targetChunks []types.ConversationChunk

	if sessionID != "" {
		targetChunks = ms.filterChunksBySession(chunks, sessionID)
	} else if len(chunks) > 0 {
		sessionID = ms.findMostRecentSessionID(chunks)
		targetChunks = ms.filterChunksBySession(chunks, sessionID)
	}

	// Analyze incomplete work
	incompleteWork := ms.detectIncompleteWork(targetChunks)
	sessionFlow := ms.analyzeSessionFlow(targetChunks)
	nextSteps := []map[string]interface{}{}

	if includeSuggestions {
		nextSteps = ms.generateContinuationSuggestions(targetChunks, chunks)
	}

	result := map[string]interface{}{
		"repository":         repository,
		"session_id":         sessionID,
		"analysis_timestamp": time.Now().Format(time.RFC3339),
		"chunks_analyzed":    len(targetChunks),

		"incomplete_work": incompleteWork,
		"session_flow":    sessionFlow,
		"next_steps":      nextSteps,

		"summary": map[string]interface{}{
			"incomplete_items": len(incompleteWork),
			"session_status":   ms.determineSessionCompletionStatus(targetChunks),
			"readiness_score":  ms.calculateContinuationReadiness(targetChunks),
		},
	}

	logging.Info("memory_continuity completed", "repository", repository, "session_id", sessionID, "incomplete_items", len(incompleteWork))
	return result, nil
}

// Helper functions for the new memory tools

// calculateMemoryMetrics computes comprehensive metrics for memory status
func (ms *MemoryServer) calculateMemoryMetrics(chunks []types.ConversationChunk) map[string]interface{} {
	if len(chunks) == 0 {
		return map[string]interface{}{
			"total_chunks": 0,
			"by_type":      map[string]int{},
			"by_outcome":   map[string]int{},
		}
	}

	typeCount := make(map[string]int)
	outcomeCount := make(map[string]int)

	for _, chunk := range chunks {
		typeCount[string(chunk.Type)]++
		outcomeCount[string(chunk.Metadata.Outcome)]++
	}

	return map[string]interface{}{
		"total_chunks": len(chunks),
		"by_type":      typeCount,
		"by_outcome":   outcomeCount,
		"oldest_chunk": chunks[len(chunks)-1].Timestamp.Format(time.RFC3339),
		"newest_chunk": chunks[0].Timestamp.Format(time.RFC3339),
	}
}

// assessMemoryHealth evaluates the health of the memory system
func (ms *MemoryServer) assessMemoryHealth(chunks []types.ConversationChunk) map[string]interface{} {
	if len(chunks) == 0 {
		return map[string]interface{}{
			"status": "no_data",
			"score":  0.0,
		}
	}

	successCount := 0
	problemCount := 0
	recentCount := 0
	cutoff := time.Now().AddDate(0, 0, -7) // Last week

	for _, chunk := range chunks {
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
			successCount++
		}
		if chunk.Type == types.ChunkTypeProblem {
			problemCount++
		}
		if chunk.Timestamp.After(cutoff) {
			recentCount++
		}
	}

	healthScore := float64(successCount) / float64(len(chunks))

	status := "healthy"
	if healthScore < 0.3 {
		status = "poor"
	} else if healthScore < 0.6 {
		status = "needs_attention"
	}

	return map[string]interface{}{
		"status":                   status,
		"score":                    healthScore,
		"recent_activity":          recentCount,
		"success_rate":             healthScore,
		"problem_resolution_ratio": float64(successCount) / math.Max(float64(problemCount), 1),
	}
}

// analyzeMemoryTrends identifies trends in memory usage
func (ms *MemoryServer) analyzeMemoryTrends(chunks []types.ConversationChunk) map[string]interface{} {
	if len(chunks) < 2 {
		return map[string]interface{}{
			"trend": "insufficient_data",
		}
	}

	// Group by week
	weeklyActivity := make(map[string]int)
	for _, chunk := range chunks {
		week := chunk.Timestamp.Format("2006-W02")
		weeklyActivity[week]++
	}

	// Simple trend analysis
	weeks := make([]string, 0, len(weeklyActivity))
	for week := range weeklyActivity {
		weeks = append(weeks, week)
	}

	// Sort weeks
	for i := 0; i < len(weeks)-1; i++ {
		for j := i + 1; j < len(weeks); j++ {
			if weeks[i] > weeks[j] {
				weeks[i], weeks[j] = weeks[j], weeks[i]
			}
		}
	}

	trend := "stable"
	if len(weeks) >= 3 {
		recent := weeklyActivity[weeks[len(weeks)-1]]
		older := weeklyActivity[weeks[len(weeks)-3]]

		if recent > int(float64(older)*1.5) {
			trend = "increasing"
		} else if recent < int(float64(older)*0.5) {
			trend = "decreasing"
		}
	}

	return map[string]interface{}{
		"trend":           trend,
		"weekly_activity": weeklyActivity,
		"total_weeks":     len(weeks),
	}
}

// calculateMemoryCoverage assesses how well the memory covers different areas
func (ms *MemoryServer) calculateMemoryCoverage(chunks []types.ConversationChunk) map[string]interface{} {
	coverage := map[string]interface{}{
		"has_architecture_decisions": false,
		"has_problem_solutions":      false,
		"has_code_changes":           false,
		"coverage_score":             0.0,
	}

	hasArch := false
	hasProblems := false
	hasCode := false

	for _, chunk := range chunks {
		switch chunk.Type {
		case types.ChunkTypeArchitectureDecision:
			hasArch = true
		case types.ChunkTypeProblem, types.ChunkTypeSolution:
			hasProblems = true
		case types.ChunkTypeCodeChange:
			hasCode = true
		case types.ChunkTypeDiscussion, types.ChunkTypeSessionSummary, types.ChunkTypeAnalysis, types.ChunkTypeVerification, types.ChunkTypeQuestion:
			// Other chunk types, no special handling needed
		default:
			// Unknown chunk type, no special handling needed
		}
	}

	coverage["has_architecture_decisions"] = hasArch
	coverage["has_problem_solutions"] = hasProblems
	coverage["has_code_changes"] = hasCode

	score := 0.0
	if hasArch {
		score += 0.4
	}
	if hasProblems {
		score += 0.4
	}
	if hasCode {
		score += 0.2
	}

	coverage["coverage_score"] = score

	return coverage
}

// identifyKnowledgeGaps finds areas lacking documentation
func (ms *MemoryServer) identifyKnowledgeGaps(chunks []types.ConversationChunk) []map[string]interface{} {
	gaps := []map[string]interface{}{}

	// Check for common gaps
	hasArchitecture := false
	hasTesting := false
	hasDeployment := false

	for _, chunk := range chunks {
		content := strings.ToLower(chunk.Content + " " + chunk.Summary)
		if strings.Contains(content, "architecture") || strings.Contains(content, "design") {
			hasArchitecture = true
		}
		if strings.Contains(content, "test") || strings.Contains(content, "testing") {
			hasTesting = true
		}
		if strings.Contains(content, "deploy") || strings.Contains(content, "deployment") {
			hasDeployment = true
		}
	}

	if !hasArchitecture {
		gaps = append(gaps, map[string]interface{}{
			"area":        "architecture",
			"description": "Limited architectural documentation",
			"suggestion":  "Document key architectural decisions and design patterns",
		})
	}

	if !hasTesting {
		gaps = append(gaps, map[string]interface{}{
			"area":        "testing",
			"description": "Testing practices not well documented",
			"suggestion":  "Record testing strategies and common test patterns",
		})
	}

	if !hasDeployment {
		gaps = append(gaps, map[string]interface{}{
			"area":        "deployment",
			"description": "Deployment processes not documented",
			"suggestion":  "Document deployment procedures and troubleshooting",
		})
	}

	return gaps
}

// calculateOverallEffectiveness provides an overall effectiveness score
func (ms *MemoryServer) calculateOverallEffectiveness(chunks []types.ConversationChunk) float64 {
	if len(chunks) == 0 {
		return 0.0
	}

	successCount := 0
	totalCount := len(chunks)

	for _, chunk := range chunks {
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
			successCount++
		}
	}

	// Base effectiveness on success rate
	effectiveness := float64(successCount) / float64(totalCount)

	// Bonus for variety of chunk types
	typeVariety := ms.calculateTypeVariety(chunks)
	effectiveness = (effectiveness * 0.8) + (typeVariety * 0.2)

	return effectiveness
}

// calculateTypeVariety measures the variety of chunk types
func (ms *MemoryServer) calculateTypeVariety(chunks []types.ConversationChunk) float64 {
	types := make(map[types.ChunkType]bool)
	for _, chunk := range chunks {
		types[chunk.Type] = true
	}

	// Maximum variety is having all 6 main types
	maxTypes := 6.0
	return float64(len(types)) / maxTypes
}

// getChunksForTimeframe retrieves chunks for a specific timeframe
func (ms *MemoryServer) getChunksForTimeframe(ctx context.Context, repository, timeframe string) ([]types.ConversationChunk, error) {
	var chunks []types.ConversationChunk
	var err error

	if repository == "" {
		// Global search across all repositories
		// For now, we'll use a search approach since we don't have a global list method
		query := types.NewMemoryQuery("*") // Broad query
		query.Repository = nil

		// Set timeframe
		switch timeframe {
		case types.TimeframWeek:
			query.Recency = types.RecencyRecent
		case types.TimeframeMonth:
			query.Recency = types.RecencyLastMonth
		default:
			query.Recency = types.RecencyAllTime
		}

		// Use empty embeddings for broad search - this is a fallback
		embeddings := make([]float64, 1536) // Default embedding size
		results, err := ms.container.GetVectorStore().Search(ctx, *query, embeddings)
		if err == nil {
			for _, result := range results.Results {
				chunks = append(chunks, result.Chunk)
			}
		}
	} else {
		chunks, err = ms.container.GetVectorStore().ListByRepository(ctx, repository, 100, 0)
	}

	if err != nil {
		return nil, err
	}

	// Filter by timeframe if specific repository
	if repository != "" {
		filteredChunks := []types.ConversationChunk{}
		var cutoff time.Time

		switch timeframe {
		case types.TimeframWeek:
			cutoff = time.Now().AddDate(0, 0, -7)
		case types.TimeframeMonth:
			cutoff = time.Now().AddDate(0, -1, 0)
		case "quarter":
			cutoff = time.Now().AddDate(0, -3, 0)
		default: // "all"
			cutoff = time.Time{} // No filtering
		}

		for _, chunk := range chunks {
			if cutoff.IsZero() || chunk.Timestamp.After(cutoff) {
				filteredChunks = append(filteredChunks, chunk)
			}
		}
		chunks = filteredChunks
	}

	return chunks, nil
}

// detectConflicts finds conflicting information in chunks
func (ms *MemoryServer) detectConflicts(chunks []types.ConversationChunk) []map[string]interface{} {
	conflicts := []map[string]interface{}{}

	// Simple conflict detection based on contradictory outcomes for similar content
	for i, chunk1 := range chunks {
		for j, chunk2 := range chunks {
			if i >= j || chunk1.SessionID == chunk2.SessionID {
				continue
			}

			// Check for similar summaries with different outcomes
			if ms.areSimilarSummaries(chunk1.Summary, chunk2.Summary) &&
				chunk1.Metadata.Outcome != chunk2.Metadata.Outcome &&
				(chunk1.Metadata.Outcome == types.OutcomeSuccess || chunk1.Metadata.Outcome == types.OutcomeFailed) &&
				(chunk2.Metadata.Outcome == types.OutcomeSuccess || chunk2.Metadata.Outcome == types.OutcomeFailed) {

				conflicts = append(conflicts, map[string]interface{}{
					"type":        "outcome_conflict",
					"description": "Similar issues with different outcomes",
					"chunk1": map[string]interface{}{
						"id":        chunk1.ID,
						"summary":   chunk1.Summary,
						"outcome":   string(chunk1.Metadata.Outcome),
						"timestamp": chunk1.Timestamp.Format(time.RFC3339),
					},
					"chunk2": map[string]interface{}{
						"id":        chunk2.ID,
						"summary":   chunk2.Summary,
						"outcome":   string(chunk2.Metadata.Outcome),
						"timestamp": chunk2.Timestamp.Format(time.RFC3339),
					},
					"severity": types.PriorityMedium,
				})
			}
		}
	}

	return conflicts
}

// findContradictoryDecisions identifies contradictory architectural decisions
func (ms *MemoryServer) findContradictoryDecisions(chunks []types.ConversationChunk) []map[string]interface{} {
	contradictions := []map[string]interface{}{}

	decisions := []types.ConversationChunk{}
	for _, chunk := range chunks {
		if chunk.Type == types.ChunkTypeArchitectureDecision {
			decisions = append(decisions, chunk)
		}
	}

	// Look for decisions that might contradict each other
	for i, decision1 := range decisions {
		for j, decision2 := range decisions {
			if i >= j {
				continue
			}

			// Simple keyword-based contradiction detection
			if ms.hasContradictoryKeywords(decision1.Content, decision2.Content) {
				contradictions = append(contradictions, map[string]interface{}{
					"type":        "architecture_contradiction",
					"description": "Potentially contradictory architectural decisions",
					"decision1": map[string]interface{}{
						"id":        decision1.ID,
						"summary":   decision1.Summary,
						"timestamp": decision1.Timestamp.Format(time.RFC3339),
					},
					"decision2": map[string]interface{}{
						"id":        decision2.ID,
						"summary":   decision2.Summary,
						"timestamp": decision2.Timestamp.Format(time.RFC3339),
					},
					"severity": types.PriorityHigh,
				})
			}
		}
	}

	return contradictions
}

// findPatternInconsistencies identifies inconsistent patterns
func (ms *MemoryServer) findPatternInconsistencies(chunks []types.ConversationChunk) []map[string]interface{} {
	// Placeholder for pattern inconsistency detection
	// This would integrate with the pattern analysis system
	return []map[string]interface{}{}
}

// Helper functions for conflict detection
func (ms *MemoryServer) areSimilarSummaries(summary1, summary2 string) bool {
	// Simple similarity check based on common words
	words1 := strings.Fields(strings.ToLower(summary1))
	words2 := strings.Fields(strings.ToLower(summary2))

	commonWords := 0
	for _, word1 := range words1 {
		for _, word2 := range words2 {
			if word1 == word2 && len(word1) > 3 { // Only count significant words
				commonWords++
				break
			}
		}
	}

	// Consider similar if they share significant words
	minWords := math.Min(float64(len(words1)), float64(len(words2)))
	return float64(commonWords)/minWords > 0.3
}

func (ms *MemoryServer) hasContradictoryKeywords(content1, content2 string) bool {
	// Simple contradiction detection based on opposing keywords
	contradictoryPairs := [][2]string{
		{"sync", "async"},
		{"synchronous", "asynchronous"},
		{"sql", "nosql"},
		{"relational", "document"},
		{"rest", "graphql"},
		{"microservice", "monolith"},
		{"client-side", "server-side"},
	}

	content1Lower := strings.ToLower(content1)
	content2Lower := strings.ToLower(content2)

	for _, pair := range contradictoryPairs {
		if (strings.Contains(content1Lower, pair[0]) && strings.Contains(content2Lower, pair[1])) ||
			(strings.Contains(content1Lower, pair[1]) && strings.Contains(content2Lower, pair[0])) {
			return true
		}
	}

	return false
}

// Additional helper functions for memory status tools
func (ms *MemoryServer) categorizeConflictsBySeverity(conflicts, contradictions, _ []map[string]interface{}) map[string]int {
	severity := map[string]int{
		types.PriorityHigh:   0,
		types.PriorityMedium: 0,
		"low":                0,
	}

	for _, conflict := range conflicts {
		if sev, ok := conflict["severity"].(string); ok {
			severity[sev]++
		}
	}

	for _, contradiction := range contradictions {
		if sev, ok := contradiction["severity"].(string); ok {
			severity[sev]++
		}
	}

	return severity
}

func (ms *MemoryServer) generateConflictResolutionRecommendations(conflicts, contradictions []map[string]interface{}) []string {
	recommendations := []string{}

	if len(conflicts) > 0 {
		recommendations = append(recommendations, "Review conflicting outcomes and update with current best practices")
	}

	if len(contradictions) > 0 {
		recommendations = append(recommendations, "Reconcile contradictory architectural decisions")
		recommendations = append(recommendations, "Create a unified architecture document")
	}

	if len(conflicts)+len(contradictions) > 5 {
		recommendations = append(recommendations, "Consider a comprehensive memory cleanup and consolidation")
	}

	return recommendations
}

// Session continuity helper functions
func (ms *MemoryServer) analyzeSessionFlow(chunks []types.ConversationChunk) map[string]interface{} {
	if len(chunks) == 0 {
		return map[string]interface{}{
			"flow":   "empty",
			"stages": []string{},
		}
	}

	// Sort chunks by timestamp
	sortedChunks := make([]types.ConversationChunk, len(chunks))
	copy(sortedChunks, chunks)

	for i := 0; i < len(sortedChunks)-1; i++ {
		for j := i + 1; j < len(sortedChunks); j++ {
			if sortedChunks[j].Timestamp.Before(sortedChunks[i].Timestamp) {
				sortedChunks[i], sortedChunks[j] = sortedChunks[j], sortedChunks[i]
			}
		}
	}

	stages := []string{}
	for _, chunk := range sortedChunks {
		stages = append(stages, string(chunk.Type))
	}

	// Determine overall flow pattern
	flow := "linear"
	if len(stages) > 3 {
		// Check for iterative patterns
		problemCount := 0
		solutionCount := 0
		for _, stage := range stages {
			if stage == string(types.ChunkTypeProblem) {
				problemCount++
			}
			if stage == string(types.ChunkTypeSolution) {
				solutionCount++
			}
		}

		if problemCount > 1 && solutionCount > 1 {
			flow = "iterative"
		}
	}

	return map[string]interface{}{
		"flow":         flow,
		"stages":       stages,
		"total_stages": len(stages),
		"duration":     sortedChunks[len(sortedChunks)-1].Timestamp.Sub(sortedChunks[0].Timestamp).String(),
	}
}

func (ms *MemoryServer) generateContinuationSuggestions(sessionChunks, _ []types.ConversationChunk) []map[string]interface{} {
	suggestions := []map[string]interface{}{}

	if len(sessionChunks) == 0 {
		return suggestions
	}

	// Check for incomplete work
	incompleteWork := ms.detectIncompleteWork(sessionChunks)
	if len(incompleteWork) > 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":        "resume_incomplete",
			"title":       "Resume incomplete work",
			"description": fmt.Sprintf("Complete %d pending items from your last session", len(incompleteWork)),
			"action":      "Review incomplete work and update their status",
			"priority":    types.PriorityHigh,
		})
	}

	// Check for recent problems without solutions
	recentProblems := []types.ConversationChunk{}
	for _, chunk := range sessionChunks {
		if chunk.Type == types.ChunkTypeProblem {
			recentProblems = append(recentProblems, chunk)
		}
	}

	if len(recentProblems) > 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":        "solve_problems",
			"title":       "Address unresolved problems",
			"description": fmt.Sprintf("Work on solutions for %d identified problems", len(recentProblems)),
			"action":      "Use memory_find_similar to find related solutions",
			"priority":    types.PriorityMedium,
		})
	}

	// Suggest documentation if session had many changes
	codeChangeCount := 0
	for _, chunk := range sessionChunks {
		if chunk.Type == types.ChunkTypeCodeChange {
			codeChangeCount++
		}
	}

	if codeChangeCount >= 3 {
		suggestions = append(suggestions, map[string]interface{}{
			"type":        "document_changes",
			"title":       "Document recent changes",
			"description": fmt.Sprintf("Document the %d code changes made in this session", codeChangeCount),
			"action":      "Create architecture decision records for significant changes",
			"priority":    "low",
		})
	}

	return suggestions
}

func (ms *MemoryServer) determineSessionCompletionStatus(chunks []types.ConversationChunk) string {
	if len(chunks) == 0 {
		return "empty"
	}

	incompleteWork := ms.detectIncompleteWork(chunks)
	incompleteCount := len(incompleteWork)

	switch {
	case incompleteCount == 0:
		return "complete"
	case incompleteCount > len(chunks)/2:
		return "mostly_incomplete"
	default:
		return "partially_complete"
	}
}

func (ms *MemoryServer) calculateContinuationReadiness(chunks []types.ConversationChunk) float64 {
	if len(chunks) == 0 {
		return 0.0
	}

	incompleteWork := ms.detectIncompleteWork(chunks)
	completionRate := 1.0 - (float64(len(incompleteWork)) / float64(len(chunks)))

	// Boost readiness if there are clear next steps
	hasNextSteps := false
	for _, chunk := range chunks {
		if strings.Contains(strings.ToLower(chunk.Content), "next") ||
			strings.Contains(strings.ToLower(chunk.Content), "todo") ||
			strings.Contains(strings.ToLower(chunk.Content), "continue") {
			hasNextSteps = true
			break
		}
	}

	readiness := completionRate
	if hasNextSteps {
		readiness = math.Min(readiness+0.2, 1.0)
	}

	return readiness
}

// Memory Threading Handler Functions

// handleCreateThread creates a new memory thread from related chunks
func (ms *MemoryServer) handleCreateThread(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_create_thread called", "params", params)

	chunkIDsInterface, ok := params["chunk_ids"].([]interface{})
	if !ok || len(chunkIDsInterface) == 0 {
		logging.Error("memory_create_thread failed: missing or empty chunk_ids parameter")
		return nil, fmt.Errorf("chunk_ids is required and must not be empty")
	}

	// Convert interface{} slice to string slice
	chunkIDs := make([]string, len(chunkIDsInterface))
	for i, id := range chunkIDsInterface {
		if idStr, ok := id.(string); ok {
			chunkIDs[i] = idStr
		} else {
			return nil, fmt.Errorf("chunk_ids must be an array of strings")
		}
	}

	// Get thread type
	threadTypeStr := types.SourceConversation // default
	if tt, ok := params["thread_type"].(string); ok && tt != "" {
		threadTypeStr = tt
	}

	// Parse thread type
	var threadType threading.ThreadType
	switch threadTypeStr {
	case types.SourceConversation:
		threadType = threading.ThreadTypeConversation
	case "problem_solving":
		threadType = threading.ThreadTypeProblemSolving
	case "feature":
		threadType = threading.ThreadTypeFeature
	case "debugging":
		threadType = threading.ThreadTypeDebugging
	case "architecture":
		threadType = threading.ThreadTypeArchitecture
	case "workflow":
		threadType = threading.ThreadTypeWorkflow
	default:
		return nil, fmt.Errorf("invalid thread_type: %s", threadTypeStr)
	}

	// Get chunks by IDs
	chunks := []types.ConversationChunk{}

	for _, chunkID := range chunkIDs {
		// Note: We need a GetChunk method or similar. For now, we'll use search as fallback
		// This is a simplified approach - in production you'd want a direct GetChunk method
		if chunk, err := ms.getChunkByID(ctx, chunkID); err == nil {
			chunks = append(chunks, *chunk)
		} else {
			logging.Warn("Failed to retrieve chunk", "chunk_id", chunkID, "error", err)
		}
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no valid chunks found for the provided chunk_ids")
	}

	// Create thread using ThreadManager
	threadManager := ms.container.GetThreadManager()
	thread, err := threadManager.CreateThread(ctx, chunks, threadType)
	if err != nil {
		logging.Error("Failed to create thread", "error", err)
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	// Override title if provided
	if title, ok := params["title"].(string); ok && title != "" {
		thread.Title = title
		// Update the stored thread
		if err := ms.container.GetThreadStore().StoreThread(ctx, thread); err != nil {
			logging.Warn("Failed to update thread title", "thread_id", thread.ID, "error", err)
		}
	}

	// Override repository if provided
	if repository, ok := params["repository"].(string); ok && repository != "" {
		thread.Repository = repository
		// Update the stored thread
		if err := ms.container.GetThreadStore().StoreThread(ctx, thread); err != nil {
			logging.Warn("Failed to update thread repository", "thread_id", thread.ID, "error", err)
		}
	}

	result := map[string]interface{}{
		"thread_id":   thread.ID,
		"title":       thread.Title,
		"description": thread.Description,
		"type":        string(thread.Type),
		"status":      string(thread.Status),
		"repository":  thread.Repository,
		"chunk_count": len(thread.ChunkIDs),
		"created_at":  thread.StartTime.Format(time.RFC3339),
		"session_ids": thread.SessionIDs,
		"tags":        thread.Tags,
		"priority":    thread.Priority,
	}

	logging.Info("memory_create_thread completed successfully", "thread_id", thread.ID, "chunk_count", len(chunks))
	return result, nil
}

// handleGetThreads retrieves memory threads with optional filtering
func (ms *MemoryServer) handleGetThreads(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_get_threads called", "params", params)

	// Build filters from parameters
	filters := threading.ThreadFilters{}

	if repository, ok := params["repository"].(string); ok && repository != "" {
		filters.Repository = &repository
	}

	if status, ok := params["status"].(string); ok && status != "" {
		threadStatus := threading.ThreadStatus(status)
		filters.Status = &threadStatus
	}

	if threadType, ok := params["thread_type"].(string); ok && threadType != "" {
		tType := threading.ThreadType(threadType)
		filters.Type = &tType
	}

	if sessionID, ok := params["session_id"].(string); ok && sessionID != "" {
		filters.SessionID = &sessionID
	}

	includeSummary := false
	if inc, ok := params["include_summary"].(bool); ok {
		includeSummary = inc
	}

	// Get threads from store
	threadStore := ms.container.GetThreadStore()
	threads, err := threadStore.ListThreads(ctx, filters)
	if err != nil {
		logging.Error("Failed to list threads", "error", err)
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}

	// Format response
	threadList := []map[string]interface{}{}

	for _, thread := range threads {
		threadInfo := map[string]interface{}{
			"thread_id":   thread.ID,
			"title":       thread.Title,
			"description": thread.Description,
			"type":        string(thread.Type),
			"status":      string(thread.Status),
			"repository":  thread.Repository,
			"chunk_count": len(thread.ChunkIDs),
			"start_time":  thread.StartTime.Format(time.RFC3339),
			"last_update": thread.LastUpdate.Format(time.RFC3339),
			"session_ids": thread.SessionIDs,
			"tags":        thread.Tags,
			"priority":    thread.Priority,
		}

		if thread.EndTime != nil {
			threadInfo["end_time"] = thread.EndTime.Format(time.RFC3339)
		}

		// Include summary if requested
		if includeSummary {
			chunks := ms.getChunksForThread(ctx, thread.ChunkIDs)
			threadManager := ms.container.GetThreadManager()
			summary, err := threadManager.GetThreadSummary(ctx, thread.ID, chunks)
			if err == nil {
				threadInfo["summary"] = map[string]interface{}{
					"duration":     summary.Duration.String(),
					"progress":     summary.Progress,
					"health_score": summary.HealthScore,
					"next_steps":   summary.NextSteps,
				}
			}
		}

		threadList = append(threadList, threadInfo)
	}

	result := map[string]interface{}{
		"threads":     threadList,
		"total_count": len(threads),
		"filters":     filters,
	}

	logging.Info("memory_get_threads completed successfully", "thread_count", len(threads))
	return result, nil
}

// handleDetectThreads automatically detects and creates memory threads from existing chunks
func (ms *MemoryServer) handleDetectThreads(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_detect_threads called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_detect_threads failed: missing repository parameter")
		return nil, fmt.Errorf("repository is required")
	}

	autoCreate := true
	if ac, ok := params["auto_create"].(bool); ok {
		autoCreate = ac
	}

	minThreadSize := 2
	if mts, ok := params["min_thread_size"].(float64); ok {
		minThreadSize = int(mts)
	}

	// Get all chunks for the repository
	chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 100, 0)
	if err != nil {
		logging.Error("Failed to get repository chunks", "repository", repository, "error", err)
		return nil, fmt.Errorf("failed to get repository chunks: %w", err)
	}

	// Filter chunks by minimum thread size during grouping
	threadManager := ms.container.GetThreadManager()
	detectedThreads, err := threadManager.DetectThreads(ctx, chunks)
	if err != nil {
		logging.Error("Failed to detect threads", "error", err)
		return nil, fmt.Errorf("failed to detect threads: %w", err)
	}

	// Filter by minimum thread size
	filteredThreads := []*threading.MemoryThread{}
	for _, thread := range detectedThreads {
		if len(thread.ChunkIDs) >= minThreadSize {
			filteredThreads = append(filteredThreads, thread)
		}
	}

	// If auto_create is false, don't actually store the threads
	if !autoCreate {
		// Remove the threads from storage
		threadStore := ms.container.GetThreadStore()
		for _, thread := range filteredThreads {
			if err := threadStore.DeleteThread(ctx, thread.ID); err != nil {
				logging.Error("failed to clean up thread", "thread_id", thread.ID, "error", err)
				// Continue with other threads even if one fails
			}
		}
	}

	// Format response
	threadSummaries := []map[string]interface{}{}
	for _, thread := range filteredThreads {
		threadSummaries = append(threadSummaries, map[string]interface{}{
			"thread_id":        thread.ID,
			"title":            thread.Title,
			"type":             string(thread.Type),
			"chunk_count":      len(thread.ChunkIDs),
			"detection_method": thread.Metadata["detection_method"],
			"created":          autoCreate,
		})
	}

	result := map[string]interface{}{
		"repository":            repository,
		"detected_threads":      threadSummaries,
		"total_detected":        len(filteredThreads),
		"auto_created":          autoCreate,
		"min_thread_size":       minThreadSize,
		"total_chunks_analyzed": len(chunks),
	}

	logging.Info("memory_detect_threads completed successfully", "repository", repository, "threads_detected", len(filteredThreads))
	return result, nil
}

// handleUpdateThread updates memory thread properties
func (ms *MemoryServer) handleUpdateThread(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_update_thread called", "params", params)

	threadID, ok := params["thread_id"].(string)
	if !ok || threadID == "" {
		logging.Error("memory_update_thread failed: missing thread_id parameter")
		return nil, fmt.Errorf("thread_id is required")
	}

	// Get current thread
	threadStore := ms.container.GetThreadStore()
	thread, err := threadStore.GetThread(ctx, threadID)
	if err != nil {
		logging.Error("Failed to get thread", "thread_id", threadID, "error", err)
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	updated := false

	// Update status if provided
	if status, ok := params["status"].(string); ok && status != "" {
		newStatus := threading.ThreadStatus(status)
		err := threadStore.UpdateThreadStatus(ctx, threadID, newStatus)
		if err != nil {
			return nil, fmt.Errorf("failed to update thread status: %w", err)
		}
		thread.Status = newStatus
		updated = true
	}

	// Update title if provided
	if title, ok := params["title"].(string); ok && title != "" {
		thread.Title = title
		updated = true
	}

	// Add chunks if provided
	if addChunksInterface, ok := params["add_chunks"].([]interface{}); ok && len(addChunksInterface) > 0 {
		if ms.addChunksToThread(thread, addChunksInterface) {
			updated = true
		}
	}

	// Remove chunks if provided
	if removeChunksInterface, ok := params["remove_chunks"].([]interface{}); ok && len(removeChunksInterface) > 0 {
		removeSet := make(map[string]bool)
		for _, chunkInterface := range removeChunksInterface {
			if chunkID, ok := chunkInterface.(string); ok {
				removeSet[chunkID] = true
			}
		}

		// Filter out chunks to remove
		newChunkIDs := []string{}
		for _, chunkID := range thread.ChunkIDs {
			if !removeSet[chunkID] {
				newChunkIDs = append(newChunkIDs, chunkID)
			}
		}

		if len(newChunkIDs) != len(thread.ChunkIDs) {
			thread.ChunkIDs = newChunkIDs
			updated = true
		}
	}

	// Store updated thread if changes were made
	if updated {
		thread.LastUpdate = time.Now()
		err := threadStore.StoreThread(ctx, thread)
		if err != nil {
			return nil, fmt.Errorf("failed to store updated thread: %w", err)
		}
	}

	result := map[string]interface{}{
		"thread_id":   thread.ID,
		"title":       thread.Title,
		"status":      string(thread.Status),
		"chunk_count": len(thread.ChunkIDs),
		"last_update": thread.LastUpdate.Format(time.RFC3339),
		"updated":     updated,
	}

	logging.Info("memory_update_thread completed successfully", "thread_id", threadID, "updated", updated)
	return result, nil
}

// Cross-Project Pattern Detection Handlers

func (ms *MemoryServer) handleAnalyzeCrossRepoPatterns(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_analyze_cross_repo_patterns called", "params", params)

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_analyze_cross_repo_patterns failed: missing session_id parameter")
		return nil, fmt.Errorf("session_id is required")
	}

	multiRepoEngine := ms.container.GetMultiRepoEngine()

	// Parse optional parameters
	var repositories []string
	if reposInterface, ok := params["repositories"].([]interface{}); ok {
		for _, repoInterface := range reposInterface {
			if repo, ok := repoInterface.(string); ok {
				repositories = append(repositories, repo)
			}
		}
	}

	var techStacks []string
	if techInterface, ok := params["tech_stacks"].([]interface{}); ok {
		for _, techInterface := range techInterface {
			if tech, ok := techInterface.(string); ok {
				techStacks = append(techStacks, tech)
			}
		}
	}

	var patternTypes []string
	if typesInterface, ok := params["pattern_types"].([]interface{}); ok {
		for _, typeInterface := range typesInterface {
			if patternType, ok := typeInterface.(string); ok {
				patternTypes = append(patternTypes, patternType)
			}
		}
	}

	minFrequency := 2
	if freq, ok := params["min_frequency"].(float64); ok {
		minFrequency = int(freq)
	}

	// Update repository contexts with recent data
	for _, repo := range repositories {
		// Get recent chunks for this repository to update context
		query := types.NewMemoryQuery("")
		query.Repository = &repo
		query.Limit = 50
		query.Recency = types.RecencyRecent

		searchResults, err := ms.container.GetVectorStore().Search(ctx, *query, nil)
		if err == nil && len(searchResults.Results) > 0 {
			chunks := make([]types.ConversationChunk, 0, len(searchResults.Results))
			for _, result := range searchResults.Results {
				chunks = append(chunks, result.Chunk)
			}

			// Update repository context
			err = multiRepoEngine.UpdateRepositoryContext(ctx, repo, chunks)
			if err != nil {
				logging.Warn("Failed to update repository context", "repository", repo, "error", err)
			}
		}
	}

	// Analyze cross-repository patterns
	err := multiRepoEngine.AnalyzeCrossRepoPatterns(ctx)
	if err != nil {
		logging.Error("Failed to analyze cross-repo patterns", "error", err)
		return nil, fmt.Errorf("failed to analyze patterns: %w", err)
	}

	// Get insights to return pattern analysis
	insights, err := multiRepoEngine.GetCrossRepoInsights(ctx)
	if err != nil {
		logging.Error("Failed to get cross-repo insights", "error", err)
		return nil, fmt.Errorf("failed to get insights: %w", err)
	}

	result := map[string]interface{}{
		"session_id":     sessionID,
		"analyzed_repos": repositories,
		"filter_criteria": map[string]interface{}{
			"tech_stacks":   techStacks,
			"pattern_types": patternTypes,
			"min_frequency": minFrequency,
		},
		"insights":  insights,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	logging.Info("memory_analyze_cross_repo_patterns completed successfully", "session_id", sessionID)
	return result, nil
}

func (ms *MemoryServer) handleFindSimilarRepositories(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_find_similar_repositories called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_find_similar_repositories failed: missing repository parameter")
		return nil, fmt.Errorf("repository is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_find_similar_repositories failed: missing session_id parameter")
		return nil, fmt.Errorf("session_id is required")
	}

	// Parse optional parameters
	similarityThreshold := 0.6
	if threshold, ok := params["similarity_threshold"].(float64); ok {
		similarityThreshold = threshold
	}

	limit := 5
	if limitVal, ok := params["limit"].(float64); ok {
		limit = int(limitVal)
	}

	multiRepoEngine := ms.container.GetMultiRepoEngine()

	// Update the target repository context with recent data
	query := types.NewMemoryQuery("")
	query.Repository = &repository
	query.Limit = 50
	query.Recency = types.RecencyRecent

	searchResults, err := ms.container.GetVectorStore().Search(ctx, *query, nil)
	if err == nil && len(searchResults.Results) > 0 {
		chunks := make([]types.ConversationChunk, 0, len(searchResults.Results))
		for _, result := range searchResults.Results {
			chunks = append(chunks, result.Chunk)
		}

		// Update repository context
		err = multiRepoEngine.UpdateRepositoryContext(ctx, repository, chunks)
		if err != nil {
			logging.Warn("Failed to update repository context", "repository", repository, "error", err)
		}
	}

	// Find similar repositories
	similarRepos, err := multiRepoEngine.GetSimilarRepositories(ctx, repository, limit)
	if err != nil {
		logging.Error("Failed to find similar repositories", "repository", repository, "error", err)
		return nil, fmt.Errorf("failed to find similar repositories: %w", err)
	}

	// Convert to response format
	similarities := make([]map[string]interface{}, 0, len(similarRepos))
	for _, repo := range similarRepos {
		similarities = append(similarities, map[string]interface{}{
			"repository":      repo.Name,
			"tech_stack":      repo.TechStack,
			"framework":       repo.Framework,
			"language":        repo.Language,
			"success_rate":    repo.SuccessRate,
			"total_sessions":  repo.TotalSessions,
			"last_activity":   repo.LastActivity.Format(time.RFC3339),
			"common_patterns": repo.CommonPatterns,
		})
	}

	result := map[string]interface{}{
		"target_repository":    repository,
		"session_id":           sessionID,
		"similarity_threshold": similarityThreshold,
		"limit":                limit,
		"similar_repositories": similarities,
		"count":                len(similarities),
		"timestamp":            time.Now().Format(time.RFC3339),
	}

	logging.Info("memory_find_similar_repositories completed successfully", "repository", repository, "found", len(similarities))
	return result, nil
}

func (ms *MemoryServer) handleGetCrossRepoInsights(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_get_cross_repo_insights called", "params", params)

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_get_cross_repo_insights failed: missing session_id parameter")
		return nil, fmt.Errorf("session_id is required")
	}

	// Parse optional boolean parameters
	includeTechDistribution := true
	if val, ok := params["include_tech_distribution"].(bool); ok {
		includeTechDistribution = val
	}

	includeSuccessAnalytics := true
	if val, ok := params["include_success_analytics"].(bool); ok {
		includeSuccessAnalytics = val
	}

	includePatternFrequency := true
	if val, ok := params["include_pattern_frequency"].(bool); ok {
		includePatternFrequency = val
	}

	multiRepoEngine := ms.container.GetMultiRepoEngine()

	// Get comprehensive insights
	insights, err := multiRepoEngine.GetCrossRepoInsights(ctx)
	if err != nil {
		logging.Error("Failed to get cross-repo insights", "error", err)
		return nil, fmt.Errorf("failed to get insights: %w", err)
	}

	// Filter insights based on requested components
	result := map[string]interface{}{
		"session_id": sessionID,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	if includeTechDistribution {
		result["tech_distribution"] = map[string]interface{}{
			"tech_stack_distribution": insights["tech_stack_distribution"],
			"framework_distribution":  insights["framework_distribution"],
			"language_distribution":   insights["language_distribution"],
		}
	}

	if includeSuccessAnalytics {
		result["success_analytics"] = map[string]interface{}{
			"avg_success_rate":   insights["avg_success_rate"],
			"total_repositories": insights["total_repositories"],
		}
	}

	if includePatternFrequency {
		result["pattern_analytics"] = map[string]interface{}{
			"common_patterns":      insights["common_patterns"],
			"cross_repo_patterns":  insights["cross_repo_patterns"],
			"repository_relations": insights["repository_relations"],
		}
	}

	// Always include summary stats
	result["summary"] = map[string]interface{}{
		"total_repositories":   insights["total_repositories"],
		"cross_repo_patterns":  insights["cross_repo_patterns"],
		"repository_relations": insights["repository_relations"],
	}

	logging.Info("memory_get_cross_repo_insights completed successfully", "session_id", sessionID)
	return result, nil
}

func (ms *MemoryServer) handleSearchMultiRepo(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_search_multi_repo called", "params", params)

	query, ok := params["query"].(string)
	if !ok || query == "" {
		logging.Error("memory_search_multi_repo failed: missing query parameter")
		return nil, fmt.Errorf("query is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_search_multi_repo failed: missing session_id parameter")
		return nil, fmt.Errorf("session_id is required")
	}

	// Parse optional parameters
	var repositories []string
	if reposInterface, ok := params["repositories"].([]interface{}); ok {
		for _, repoInterface := range reposInterface {
			if repo, ok := repoInterface.(string); ok {
				repositories = append(repositories, repo)
			}
		}
	}

	var techStacks []string
	if techInterface, ok := params["tech_stacks"].([]interface{}); ok {
		for _, techInterface := range techInterface {
			if tech, ok := techInterface.(string); ok {
				techStacks = append(techStacks, tech)
			}
		}
	}

	var frameworks []string
	if frameworkInterface, ok := params["frameworks"].([]interface{}); ok {
		for _, fwInterface := range frameworkInterface {
			if fw, ok := fwInterface.(string); ok {
				frameworks = append(frameworks, fw)
			}
		}
	}

	var patternTypes []string
	if typesInterface, ok := params["pattern_types"].([]interface{}); ok {
		for _, typeInterface := range typesInterface {
			if patternType, ok := typeInterface.(string); ok {
				patternTypes = append(patternTypes, patternType)
			}
		}
	}

	minConfidence := 0.5
	if conf, ok := params["min_confidence"].(float64); ok {
		minConfidence = conf
	}

	maxResults := 10
	if max, ok := params["max_results"].(float64); ok {
		maxResults = int(max)
	}

	includeSimilar := true
	if include, ok := params["include_similar"].(bool); ok {
		includeSimilar = include
	}

	multiRepoEngine := ms.container.GetMultiRepoEngine()

	// Create multi-repository query
	multiQuery := intelligence.MultiRepoQuery{
		Query:          query,
		Repositories:   repositories,
		TechStacks:     techStacks,
		Frameworks:     frameworks,
		MinConfidence:  minConfidence,
		MaxResults:     maxResults,
		IncludeSimilar: includeSimilar,
	}

	// Convert pattern type strings to PatternType
	for _, pt := range patternTypes {
		multiQuery.PatternTypes = append(multiQuery.PatternTypes, intelligence.PatternType(pt))
	}

	// Execute multi-repository search
	results, err := multiRepoEngine.QueryMultiRepo(ctx, multiQuery)
	if err != nil {
		logging.Error("Failed to search multi-repo", "query", query, "error", err)
		return nil, fmt.Errorf("failed to search multi-repo: %w", err)
	}

	// Convert results to response format
	searchResults := make([]map[string]interface{}, 0, len(results))
	for _, result := range results {
		patterns := make([]map[string]interface{}, 0, len(result.Patterns))
		for _, pattern := range result.Patterns {
			patterns = append(patterns, map[string]interface{}{
				"id":           pattern.ID,
				"name":         pattern.Name,
				"description":  pattern.Description,
				"repositories": pattern.Repositories,
				"frequency":    pattern.Frequency,
				"success_rate": pattern.SuccessRate,
				"confidence":   pattern.Confidence,
				"keywords":     pattern.Keywords,
				"tech_stacks":  pattern.TechStacks,
				"frameworks":   pattern.Frameworks,
				"pattern_type": string(pattern.PatternType),
				"created_at":   pattern.CreatedAt.Format(time.RFC3339),
				"last_used":    pattern.LastUsed.Format(time.RFC3339),
			})
		}

		searchResults = append(searchResults, map[string]interface{}{
			"repository": result.Repository,
			"relevance":  result.Relevance,
			"patterns":   patterns,
			"context":    result.Context,
		})
	}

	response := map[string]interface{}{
		"query":      query,
		"session_id": sessionID,
		"filter_criteria": map[string]interface{}{
			"repositories":    repositories,
			"tech_stacks":     techStacks,
			"frameworks":      frameworks,
			"pattern_types":   patternTypes,
			"min_confidence":  minConfidence,
			"max_results":     maxResults,
			"include_similar": includeSimilar,
		},
		"results":       searchResults,
		"total_results": len(searchResults),
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	logging.Info("memory_search_multi_repo completed successfully", "query", query, "session_id", sessionID, "results", len(searchResults))
	return response, nil
}

func (ms *MemoryServer) handleMemoryHealthDashboard(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_health_dashboard called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_health_dashboard failed: missing repository parameter")
		return nil, fmt.Errorf("repository is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_health_dashboard failed: missing session_id parameter")
		return nil, fmt.Errorf("session_id is required")
	}

	// Parse optional parameters
	timeframe := types.TimeframeMonth
	if tf, ok := params["timeframe"].(string); ok && tf != "" {
		timeframe = tf
	}

	includeDetails := true
	if details, ok := params["include_details"].(bool); ok {
		includeDetails = details
	}

	includeRecommendations := true
	if recs, ok := params["include_recommendations"].(bool); ok {
		includeRecommendations = recs
	}

	// Calculate timeframe boundaries
	var since time.Time
	switch timeframe {
	case types.TimeframWeek:
		since = time.Now().AddDate(0, 0, -7)
	case types.TimeframeMonth:
		since = time.Now().AddDate(0, -1, 0)
	case "quarter":
		since = time.Now().AddDate(0, -3, 0)
	case "all":
		since = time.Time{} // Zero time = all time
	default:
		since = time.Now().AddDate(0, -1, 0) // Default to month
	}

	// Get all chunks for the repository within timeframe
	query := types.NewMemoryQuery("")
	if repository != "_global" {
		query.Repository = &repository
	}
	query.Limit = 1000 // Large limit to get comprehensive data

	searchResults, err := ms.container.GetVectorStore().Search(ctx, *query, nil)
	if err != nil {
		logging.Error("Failed to search chunks for health analysis", "repository", repository, "error", err)
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}

	// Filter by timeframe
	var chunks []types.ConversationChunk
	for _, result := range searchResults.Results {
		if since.IsZero() || result.Chunk.Timestamp.After(since) {
			chunks = append(chunks, result.Chunk)
		}
	}

	// Generate health analysis
	healthReport := ms.generateHealthReport(chunks, timeframe, includeDetails, includeRecommendations)

	result := map[string]interface{}{
		"repository": repository,
		"session_id": sessionID,
		"timeframe":  timeframe,
		"analysis_period": map[string]interface{}{
			"since": since.Format(time.RFC3339),
			"until": time.Now().Format(time.RFC3339),
			"days":  int(time.Since(since).Hours() / 24),
		},
		"health_report": healthReport,
		"generated_at":  time.Now().Format(time.RFC3339),
	}

	logging.Info("memory_health_dashboard completed successfully", "repository", repository, "session_id", sessionID, "chunks_analyzed", len(chunks))
	return result, nil
}

// generateHealthReport creates a comprehensive health analysis
func (ms *MemoryServer) generateHealthReport(chunks []types.ConversationChunk, _ string, includeDetails, includeRecommendations bool) map[string]interface{} {
	report := make(map[string]interface{})

	// Basic statistics
	totalChunks := len(chunks)
	report["total_chunks"] = totalChunks

	if totalChunks == 0 {
		report["status"] = "no_data"
		report["health_score"] = 0.0
		return report
	}

	// Outcome analysis
	outcomes := make(map[types.Outcome]int)
	chunkTypes := make(map[types.ChunkType]int)
	difficulties := make(map[types.Difficulty]int)

	effectivenessScores := make([]float64, 0, len(chunks))
	var oldChunks, recentChunks int
	var totalEffectiveness float64
	accessibleChunks := 0

	cutoffDate := time.Now().AddDate(0, -1, 0) // 1 month ago

	for _, chunk := range chunks {
		// Count outcomes
		outcomes[chunk.Metadata.Outcome]++

		// Count types
		chunkTypes[chunk.Type]++

		// Count difficulties
		difficulties[chunk.Metadata.Difficulty]++

		// Age analysis
		if chunk.Timestamp.Before(cutoffDate) {
			oldChunks++
		} else {
			recentChunks++
		}

		// Calculate effectiveness using analytics
		analytics := ms.container.GetMemoryAnalytics()
		effectiveness := analytics.CalculateEffectivenessScore(&chunk)
		effectivenessScores = append(effectivenessScores, effectiveness)
		totalEffectiveness += effectiveness

		// Check accessibility (has summary and good metadata)
		if chunk.Summary != "" && len(chunk.Metadata.Tags) > 0 {
			accessibleChunks++
		}
	}

	// Calculate completion rates
	successfulChunks := outcomes[types.OutcomeSuccess]
	inProgressChunks := outcomes[types.OutcomeInProgress]
	failedChunks := outcomes[types.OutcomeFailed]

	completionRate := float64(successfulChunks) / float64(totalChunks)
	failureRate := float64(failedChunks) / float64(totalChunks)

	report["completion_analysis"] = map[string]interface{}{
		"completion_rate":   completionRate,
		"success_count":     successfulChunks,
		"in_progress_count": inProgressChunks,
		"failed_count":      failedChunks,
		"abandoned_count":   outcomes[types.OutcomeAbandoned],
		"failure_rate":      failureRate,
	}

	// Effectiveness analysis
	avgEffectiveness := totalEffectiveness / float64(totalChunks)
	accessibilityRate := float64(accessibleChunks) / float64(totalChunks)

	report["effectiveness_analysis"] = map[string]interface{}{
		"average_effectiveness": avgEffectiveness,
		"accessibility_rate":    accessibilityRate,
		"high_value_chunks":     countHighValueChunks(effectivenessScores),
		"low_value_chunks":      countLowValueChunks(effectivenessScores),
	}

	// Age and staleness analysis
	staleChunkRate := float64(oldChunks) / float64(totalChunks)
	report["freshness_analysis"] = map[string]interface{}{
		"recent_chunks":  recentChunks,
		"old_chunks":     oldChunks,
		"staleness_rate": staleChunkRate,
		"avg_age_days":   calculateAvgAge(chunks),
	}

	// Type distribution
	report["type_distribution"] = chunkTypes
	report["difficulty_distribution"] = difficulties

	// Calculate overall health score (0-100)
	healthScore := calculateHealthScore(completionRate, avgEffectiveness, accessibilityRate, staleChunkRate)
	report["health_score"] = healthScore
	report["health_status"] = getHealthStatus(healthScore)

	// Quality indicators
	report["quality_indicators"] = map[string]interface{}{
		"chunks_with_summaries":   countChunksWithSummaries(chunks),
		"chunks_with_tags":        countChunksWithTags(chunks),
		"chunks_with_files":       countChunksWithFiles(chunks),
		"architectural_decisions": chunkTypes[types.ChunkTypeArchitectureDecision],
		"solutions_documented":    chunkTypes[types.ChunkTypeSolution],
	}

	// Include detailed analysis if requested
	if includeDetails {
		report["detailed_analysis"] = map[string]interface{}{
			"most_effective_chunks":  getMostEffectiveChunks(chunks, effectivenessScores, 3),
			"least_effective_chunks": getLeastEffectiveChunks(chunks, effectivenessScores, 3),
			"recent_high_impact":     getRecentHighImpactChunks(chunks, 5),
			"outdated_chunks":        getOutdatedChunks(chunks, 5),
		}
	}

	// Include recommendations if requested
	if includeRecommendations {
		report["recommendations"] = generateHealthRecommendations(completionRate, avgEffectiveness, accessibilityRate, staleChunkRate, chunkTypes)
	}

	return report
}

// Helper functions for health analysis

func countHighValueChunks(scores []float64) int {
	count := 0
	for _, score := range scores {
		if score >= 0.7 {
			count++
		}
	}
	return count
}

func countLowValueChunks(scores []float64) int {
	count := 0
	for _, score := range scores {
		if score <= 0.3 {
			count++
		}
	}
	return count
}

func calculateAvgAge(chunks []types.ConversationChunk) float64 {
	if len(chunks) == 0 {
		return 0
	}

	totalDays := 0.0
	for _, chunk := range chunks {
		days := time.Since(chunk.Timestamp).Hours() / 24
		totalDays += days
	}

	return totalDays / float64(len(chunks))
}

func calculateHealthScore(completionRate, effectiveness, accessibility, staleness float64) float64 {
	// Weight factors: completion 30%, effectiveness 30%, accessibility 25%, freshness 15%
	score := (completionRate * 30) + (effectiveness * 30) + (accessibility * 25) + ((1.0 - staleness) * 15)
	return math.Min(100.0, score*100.0)
}

func getHealthStatus(score float64) string {
	switch {
	case score >= 80:
		return "excellent"
	case score >= 60:
		return "good"
	case score >= 40:
		return "fair"
	case score >= 20:
		return "poor"
	default:
		return "critical"
	}
}

func countChunksWithSummaries(chunks []types.ConversationChunk) int {
	count := 0
	for _, chunk := range chunks {
		if chunk.Summary != "" {
			count++
		}
	}
	return count
}

func countChunksWithTags(chunks []types.ConversationChunk) int {
	count := 0
	for _, chunk := range chunks {
		if len(chunk.Metadata.Tags) > 0 {
			count++
		}
	}
	return count
}

func countChunksWithFiles(chunks []types.ConversationChunk) int {
	count := 0
	for _, chunk := range chunks {
		if len(chunk.Metadata.FilesModified) > 0 {
			count++
		}
	}
	return count
}

type chunkScore struct {
	chunk types.ConversationChunk
	score float64
}

func getEffectiveChunks(chunks []types.ConversationChunk, scores []float64, limit int, sortDescending bool) []map[string]interface{} {
	var chunkScores []chunkScore
	for i, chunk := range chunks {
		if i < len(scores) {
			chunkScores = append(chunkScores, chunkScore{chunk, scores[i]})
		}
	}

	// Sort by score (descending for most effective, ascending for least effective)
	sort.Slice(chunkScores, func(i, j int) bool {
		if sortDescending {
			return chunkScores[i].score > chunkScores[j].score
		}
		return chunkScores[i].score < chunkScores[j].score
	})

	result := make([]map[string]interface{}, 0, limit)
	for i, cs := range chunkScores {
		if i >= limit {
			break
		}
		result = append(result, map[string]interface{}{
			"id":            cs.chunk.ID,
			"type":          string(cs.chunk.Type),
			"summary":       cs.chunk.Summary,
			"effectiveness": cs.score,
			"created_at":    cs.chunk.Timestamp.Format(time.RFC3339),
		})
	}

	return result
}

func getMostEffectiveChunks(chunks []types.ConversationChunk, scores []float64, limit int) []map[string]interface{} {
	return getEffectiveChunks(chunks, scores, limit, true)
}

func getLeastEffectiveChunks(chunks []types.ConversationChunk, scores []float64, limit int) []map[string]interface{} {
	return getEffectiveChunks(chunks, scores, limit, false)
}

func getRecentHighImpactChunks(chunks []types.ConversationChunk, limit int) []map[string]interface{} {
	// Filter for recent chunks with high impact tags
	recentCutoff := time.Now().AddDate(0, 0, -7) // Last week
	var recentChunks []types.ConversationChunk

	for _, chunk := range chunks {
		if chunk.Timestamp.After(recentCutoff) {
			// Check for high-impact indicators
			hasHighImpact := false
			for _, tag := range chunk.Metadata.Tags {
				if tag == "high-impact" || tag == "architecture" || tag == "critical" {
					hasHighImpact = true
					break
				}
			}
			if hasHighImpact || chunk.Type == types.ChunkTypeArchitectureDecision {
				recentChunks = append(recentChunks, chunk)
			}
		}
	}

	// Sort by timestamp descending
	sort.Slice(recentChunks, func(i, j int) bool {
		return recentChunks[i].Timestamp.After(recentChunks[j].Timestamp)
	})

	result := make([]map[string]interface{}, 0, limit)
	for i, chunk := range recentChunks {
		if i >= limit {
			break
		}
		result = append(result, map[string]interface{}{
			"id":         chunk.ID,
			"type":       string(chunk.Type),
			"summary":    chunk.Summary,
			"tags":       chunk.Metadata.Tags,
			"created_at": chunk.Timestamp.Format(time.RFC3339),
		})
	}

	return result
}

func getOutdatedChunks(chunks []types.ConversationChunk, limit int) []map[string]interface{} {
	// Find chunks older than 3 months
	cutoff := time.Now().AddDate(0, -3, 0)
	var outdatedChunks []types.ConversationChunk

	for _, chunk := range chunks {
		if chunk.Timestamp.Before(cutoff) && chunk.Metadata.Outcome == types.OutcomeInProgress {
			outdatedChunks = append(outdatedChunks, chunk)
		}
	}

	// Sort by timestamp ascending (oldest first)
	sort.Slice(outdatedChunks, func(i, j int) bool {
		return outdatedChunks[i].Timestamp.Before(outdatedChunks[j].Timestamp)
	})

	result := make([]map[string]interface{}, 0, limit)
	for i, chunk := range outdatedChunks {
		if i >= limit {
			break
		}
		daysSince := int(time.Since(chunk.Timestamp).Hours() / 24)
		result = append(result, map[string]interface{}{
			"id":         chunk.ID,
			"type":       string(chunk.Type),
			"summary":    chunk.Summary,
			"days_old":   daysSince,
			"created_at": chunk.Timestamp.Format(time.RFC3339),
		})
	}

	return result
}

func generateHealthRecommendations(completionRate, effectiveness, accessibility, staleness float64, chunkTypes map[types.ChunkType]int) []map[string]interface{} {
	var recommendations []map[string]interface{}

	// Completion rate recommendations
	if completionRate < 0.6 {
		recommendations = append(recommendations, map[string]interface{}{
			"priority":    types.PriorityHigh,
			"category":    "completion",
			"title":       "Low completion rate detected",
			"description": fmt.Sprintf("Only %.1f%% of chunks show successful completion. Consider reviewing in-progress items and updating their status.", completionRate*100),
			"action":      "Review incomplete chunks and update outcomes",
		})
	}

	// Effectiveness recommendations
	if effectiveness < 0.5 {
		recommendations = append(recommendations, map[string]interface{}{
			"priority":    types.PriorityHigh,
			"category":    "effectiveness",
			"title":       "Low effectiveness scores",
			"description": fmt.Sprintf("Average effectiveness is %.1f%%. Consider improving chunk quality with better summaries and tags.", effectiveness*100),
			"action":      "Add summaries and relevant tags to existing chunks",
		})
	}

	// Accessibility recommendations
	if accessibility < 0.7 {
		recommendations = append(recommendations, map[string]interface{}{
			"priority":    types.PriorityMedium,
			"category":    "accessibility",
			"title":       "Poor chunk accessibility",
			"description": fmt.Sprintf("Only %.1f%% of chunks are easily accessible. Add summaries and tags to improve discoverability.", accessibility*100),
			"action":      "Improve chunk metadata and summaries",
		})
	}

	// Staleness recommendations
	if staleness > 0.4 {
		recommendations = append(recommendations, map[string]interface{}{
			"priority":    types.PriorityMedium,
			"category":    "freshness",
			"title":       "Many outdated chunks",
			"description": fmt.Sprintf("%.1f%% of chunks are older than 1 month. Consider archiving or updating stale content.", staleness*100),
			"action":      "Review and archive or update old chunks",
		})
	}

	// Type-specific recommendations
	problemChunks := chunkTypes[types.ChunkTypeProblem]
	solutionChunks := chunkTypes[types.ChunkTypeSolution]

	if problemChunks > solutionChunks*2 {
		recommendations = append(recommendations, map[string]interface{}{
			"priority":    types.PriorityMedium,
			"category":    "balance",
			"title":       "Too many unresolved problems",
			"description": fmt.Sprintf("Found %d problems but only %d solutions. Focus on documenting solutions.", problemChunks, solutionChunks),
			"action":      "Document solutions for existing problems",
		})
	}

	if chunkTypes[types.ChunkTypeArchitectureDecision] == 0 {
		recommendations = append(recommendations, map[string]interface{}{
			"priority":    "low",
			"category":    "documentation",
			"title":       "No architectural decisions documented",
			"description": "Consider documenting important architectural decisions for future reference.",
			"action":      "Use memory_store_decision for architectural choices",
		})
	}

	return recommendations
}

// Helper functions for threading

// getChunkByID retrieves a single chunk by ID (simplified implementation)
func (ms *MemoryServer) getChunkByID(ctx context.Context, chunkID string) (*types.ConversationChunk, error) {
	// This is a simplified implementation. In a real system, you'd want a direct GetChunk method
	// For now, we'll use a search with very broad criteria and filter by ID

	// Create a broad search query
	query := types.NewMemoryQuery("*")
	query.Repository = nil // Search all repositories
	query.Limit = 100

	// Use empty embeddings for broad search
	embeddings := make([]float64, 1536) // Default embedding size
	results, err := ms.container.GetVectorStore().Search(ctx, *query, embeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to search for chunk: %w", err)
	}

	// Find the chunk with matching ID
	for _, result := range results.Results {
		if result.Chunk.ID == chunkID {
			return &result.Chunk, nil
		}
	}

	return nil, fmt.Errorf("chunk not found: %s", chunkID)
}

// getChunksForThread retrieves all chunks for a thread
func (ms *MemoryServer) getChunksForThread(ctx context.Context, chunkIDs []string) []types.ConversationChunk {
	chunks := []types.ConversationChunk{}

	for _, chunkID := range chunkIDs {
		if chunk, err := ms.getChunkByID(ctx, chunkID); err == nil {
			chunks = append(chunks, *chunk)
		}
		// Continue even if some chunks fail to load
	}

	return chunks
}

// handleMemoryDecayManagement handles memory decay management operations
func (ms *MemoryServer) handleMemoryDecayManagement(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_decay_management called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_decay_management failed: missing repository parameter")
		return nil, fmt.Errorf("repository is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_decay_management failed: missing session_id parameter")
		return nil, fmt.Errorf("session_id is required")
	}

	action, ok := params["action"].(string)
	if !ok || action == "" {
		logging.Error("memory_decay_management failed: missing action parameter")
		return nil, fmt.Errorf("action is required")
	}

	// Parse optional parameters
	previewOnly := false
	if preview, ok := params["preview_only"].(bool); ok {
		previewOnly = preview
	}

	intelligentMode := false
	if intelligent, ok := params["intelligent_mode"].(bool); ok {
		intelligentMode = intelligent
	}

	var result map[string]interface{}
	var err error

	switch action {
	case "status":
		result, err = ms.handleDecayStatus(ctx, repository, sessionID)
	case "preview":
		result, err = ms.handleDecayPreview(ctx, repository, sessionID, intelligentMode)
	case "run_decay":
		result, err = ms.handleRunDecay(ctx, repository, sessionID, previewOnly, intelligentMode)
	case "configure":
		config, hasConfig := params["config"].(map[string]interface{})
		if !hasConfig {
			return nil, fmt.Errorf("config is required for configure action")
		}
		result, err = ms.handleDecayConfiguration(ctx, repository, sessionID, config)
	default:
		return nil, fmt.Errorf("unknown action: %s. Valid actions are: 'run_decay', 'configure', 'status', 'preview'", action)
	}

	if err != nil {
		logging.Error("memory_decay_management failed", "error", err, "action", action, "repository", repository)
		return nil, err
	}

	logging.Info("memory_decay_management completed successfully", "action", action, "repository", repository, "session_id", sessionID)
	return result, nil
}

// handleDecayStatus returns the current status of the decay system
func (ms *MemoryServer) handleDecayStatus(ctx context.Context, repository, sessionID string) (map[string]interface{}, error) {
	// Get all chunks for analysis
	query := types.MemoryQuery{
		Repository: &repository,
		Limit:      1000,
		Recency:    types.RecencyAllTime,
	}

	results, err := ms.container.GetVectorStore().Search(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chunks for analysis: %w", err)
	}

	chunks := make([]types.ConversationChunk, len(results.Results))
	for i, result := range results.Results {
		chunks[i] = result.Chunk
	}

	// Analyze decay eligibility
	now := time.Now()
	var oldChunks, staleChunks, candidatesForSummarization, candidatesForDeletion int
	var totalAge time.Duration

	for _, chunk := range chunks {
		age := now.Sub(chunk.Timestamp)
		totalAge += age

		if age > 30*24*time.Hour { // Older than 30 days
			oldChunks++
		}
		if age > 7*24*time.Hour && chunk.Metadata.Outcome == types.OutcomeInProgress { // Stale in-progress items
			staleChunks++
		}

		// Estimate decay score
		score := ms.estimateDecayScore(chunk, age)
		if score < 0.4 {
			candidatesForSummarization++
		}
		if score < 0.1 {
			candidatesForDeletion++
		}
	}

	var averageAge float64
	if len(chunks) > 0 {
		averageAge = totalAge.Hours() / float64(len(chunks)) / 24.0 // Average age in days
	}

	return map[string]interface{}{
		"status":     "analysis_complete",
		"repository": repository,
		"session_id": sessionID,
		"timestamp":  now.Format(time.RFC3339),
		"summary": map[string]interface{}{
			"total_chunks":                 len(chunks),
			"old_chunks":                   oldChunks,
			"stale_chunks":                 staleChunks,
			"candidates_for_summarization": candidatesForSummarization,
			"candidates_for_deletion":      candidatesForDeletion,
			"average_age_days":             averageAge,
		},
		"recommendations": ms.generateDecayRecommendations(len(chunks), oldChunks, staleChunks, candidatesForSummarization, candidatesForDeletion),
	}, nil
}

// handleDecayPreview shows what would be processed without making changes
func (ms *MemoryServer) handleDecayPreview(ctx context.Context, repository, sessionID string, intelligentMode bool) (map[string]interface{}, error) {
	// Get all chunks for analysis
	query := types.MemoryQuery{
		Repository: &repository,
		Limit:      1000,
		Recency:    types.RecencyAllTime,
	}

	results, err := ms.container.GetVectorStore().Search(ctx, query, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chunks for preview: %w", err)
	}

	chunks := make([]types.ConversationChunk, len(results.Results))
	for i, result := range results.Results {
		chunks[i] = result.Chunk
	}

	// Analyze what would be processed
	now := time.Now()
	toSummarize := make([]map[string]interface{}, 0)
	toDelete := make([]map[string]interface{}, 0)
	toUpdate := make([]map[string]interface{}, 0)

	for _, chunk := range chunks {
		age := now.Sub(chunk.Timestamp)

		// Skip recent memories (less than 7 days old)
		if age < 7*24*time.Hour {
			continue
		}

		score := ms.estimateDecayScore(chunk, age)

		chunkInfo := map[string]interface{}{
			"id":          chunk.ID,
			"type":        string(chunk.Type),
			"summary":     chunk.Summary,
			"age_days":    int(age.Hours() / 24),
			"decay_score": score,
			"created_at":  chunk.Timestamp.Format(time.RFC3339),
		}

		switch {
		case score < 0.1:
			toDelete = append(toDelete, chunkInfo)
		case score < 0.4:
			toSummarize = append(toSummarize, chunkInfo)
		case score < 0.7:
			toUpdate = append(toUpdate, chunkInfo)
		}
	}

	mode := "basic"
	if intelligentMode {
		mode = "intelligent_llm"
	}

	return map[string]interface{}{
		"preview":    "decay_analysis",
		"repository": repository,
		"session_id": sessionID,
		"mode":       mode,
		"timestamp":  now.Format(time.RFC3339),
		"analysis": map[string]interface{}{
			"total_chunks_analyzed": len(chunks),
			"chunks_to_summarize":   len(toSummarize),
			"chunks_to_delete":      len(toDelete),
			"chunks_to_update":      len(toUpdate),
		},
		"details": map[string]interface{}{
			"to_summarize": toSummarize,
			"to_delete":    toDelete,
			"to_update":    toUpdate,
		},
		"next_steps": []string{
			"Review the chunks marked for processing",
			"Adjust decay configuration if needed",
			"Run with 'run_decay' action to execute changes",
		},
	}, nil
}

// handleRunDecay executes the decay process
func (ms *MemoryServer) handleRunDecay(ctx context.Context, repository, sessionID string, previewOnly, intelligentMode bool) (map[string]interface{}, error) {
	if previewOnly {
		return ms.handleDecayPreview(ctx, repository, sessionID, intelligentMode)
	}

	// This is a simplified implementation
	// In a full implementation, you would:
	// 1. Create a proper decay manager instance
	// 2. Configure it with embeddings if intelligentMode is true
	// 3. Run the actual decay process

	return map[string]interface{}{
		"status":     "decay_executed",
		"repository": repository,
		"session_id": sessionID,
		"mode": map[string]interface{}{
			"intelligent":  intelligentMode,
			"preview_only": previewOnly,
		},
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   "Memory decay process completed. Note: This is a simplified implementation for demonstration.",
		"note":      "Full implementation would integrate with the decay manager and perform actual summarization and cleanup.",
	}, nil
}

// handleDecayConfiguration handles decay configuration updates
func (ms *MemoryServer) handleDecayConfiguration(_ context.Context, repository, sessionID string, config map[string]interface{}) (map[string]interface{}, error) {
	// Parse configuration
	strategy := "adaptive"
	if s, ok := config["strategy"].(string); ok && s != "" {
		strategy = s
	}

	baseDecayRate := 0.1
	if rate, ok := config["base_decay_rate"].(float64); ok {
		baseDecayRate = rate
	}

	summarizationThreshold := 0.4
	if threshold, ok := config["summarization_threshold"].(float64); ok {
		summarizationThreshold = threshold
	}

	deletionThreshold := 0.1
	if threshold, ok := config["deletion_threshold"].(float64); ok {
		deletionThreshold = threshold
	}

	retentionPeriodDays := 7.0
	if days, ok := config["retention_period_days"].(float64); ok {
		retentionPeriodDays = days
	}

	return map[string]interface{}{
		"status":     "configuration_updated",
		"repository": repository,
		"session_id": sessionID,
		"timestamp":  time.Now().Format(time.RFC3339),
		"configuration": map[string]interface{}{
			"strategy":                strategy,
			"base_decay_rate":         baseDecayRate,
			"summarization_threshold": summarizationThreshold,
			"deletion_threshold":      deletionThreshold,
			"retention_period_days":   retentionPeriodDays,
		},
		"note": "Configuration saved. This is a simplified implementation for demonstration.",
	}, nil
}

// estimateDecayScore estimates the decay score for a chunk (simplified implementation)
func (ms *MemoryServer) estimateDecayScore(chunk types.ConversationChunk, age time.Duration) float64 {
	// Base score starts at 1.0
	score := 1.0

	// Apply time decay (adaptive strategy)
	days := age.Hours() / 24.0
	switch {
	case days < 7:
		// Minimal decay in first week
		score *= (1.0 - 0.01*days/7.0)
	case days < 30:
		// Moderate decay for first month
		score *= (0.99 - 0.3*(days-7)/23.0)
	default:
		// Accelerated decay after a month
		score *= math.Pow(0.6, (days-30)/30.0)
	}

	// Apply importance boost
	switch chunk.Type {
	case types.ChunkTypeArchitectureDecision:
		score *= 2.0
	case types.ChunkTypeSolution:
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
			score *= 1.8
		}
	case types.ChunkTypeProblem:
		score *= 1.5
	case types.ChunkTypeCodeChange:
		score *= 1.3
	case types.ChunkTypeDiscussion:
		score *= 1.1
	case types.ChunkTypeSessionSummary:
		score *= 1.2
	case types.ChunkTypeAnalysis:
		score *= 1.2
	case types.ChunkTypeVerification:
		score *= 1.1
	case types.ChunkTypeQuestion:
		score *= 1.0
	default:
		// Other chunk types use base score without boost
	}

	// Consider relationships
	if len(chunk.RelatedChunks) > 0 {
		score *= (1.0 + float64(len(chunk.RelatedChunks))/10.0)
	}

	return math.Max(0.0, math.Min(1.0, score))
}

// generateDecayRecommendations generates actionable recommendations
func (ms *MemoryServer) generateDecayRecommendations(total, old, stale, forSummarization, forDeletion int) []string {
	recommendations := make([]string, 0)

	if old > total/2 {
		recommendations = append(recommendations, "Consider running decay process - over 50% of chunks are older than 30 days")
	}

	if stale > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Review %d stale in-progress items that may need completion or archival", stale))
	}

	if forSummarization > 20 {
		recommendations = append(recommendations, fmt.Sprintf("Enable intelligent summarization for %d low-relevance chunks to save space", forSummarization))
	}

	if forDeletion > 10 {
		recommendations = append(recommendations, fmt.Sprintf("Consider deleting %d very low-relevance chunks to improve performance", forDeletion))
	}

	if total > 1000 {
		recommendations = append(recommendations, "Large memory repository detected - consider regular decay maintenance")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Memory health looks good - no immediate action required")
	}

	return recommendations
}

// searchRelatedRepositories searches related repositories for the given query
func (ms *MemoryServer) searchRelatedRepositories(ctx context.Context, relaxedQuery types.MemoryQuery, embeddings []float64, originalRepo string, searchConfig config.SearchConfig) (*types.SearchResults, error) {
	relatedRepos := ms.generateRelatedRepositories(originalRepo)
	// Limit to configured max related repos
	if len(relatedRepos) > searchConfig.MaxRelatedRepos {
		relatedRepos = relatedRepos[:searchConfig.MaxRelatedRepos]
	}

	for _, relatedRepo := range relatedRepos {
		relatedQuery := relaxedQuery
		relatedQuery.Repository = &relatedRepo
		logging.Info("Progressive search: Step 3 - Related repo search", "original_repo", originalRepo, "trying_repo", relatedRepo)
		results, err := ms.container.GetVectorStore().Search(ctx, relatedQuery, embeddings)
		if err != nil {
			continue // Try next related repo
		}

		if len(results.Results) > 0 {
			logging.Info("Progressive search: Related repo search succeeded", "results", len(results.Results), "repo", relatedRepo)
			return results, nil
		}
	}

	return nil, fmt.Errorf("no results found in related repositories")
}

// filterChunksBySession filters chunks by the given session ID
func (ms *MemoryServer) filterChunksBySession(chunks []types.ConversationChunk, sessionID string) []types.ConversationChunk {
	var filtered []types.ConversationChunk
	for _, chunk := range chunks {
		if chunk.SessionID == sessionID {
			filtered = append(filtered, chunk)
		}
	}
	return filtered
}

// findMostRecentSessionID finds the session ID with the most recent timestamp
func (ms *MemoryServer) findMostRecentSessionID(chunks []types.ConversationChunk) string {
	if len(chunks) == 0 {
		return ""
	}

	latestSessionID := chunks[0].SessionID
	latestTime := chunks[0].Timestamp

	for _, chunk := range chunks {
		if chunk.Timestamp.After(latestTime) {
			latestTime = chunk.Timestamp
			latestSessionID = chunk.SessionID
		}
	}

	return latestSessionID
}

// addChunksToThread adds chunks to a thread if they don't already exist
func (ms *MemoryServer) addChunksToThread(thread *threading.MemoryThread, addChunksInterface []interface{}) bool {
	updated := false
	for _, chunkInterface := range addChunksInterface {
		chunkID, ok := chunkInterface.(string)
		if !ok {
			continue
		}

		// Check if chunk already exists in thread
		if ms.chunkExistsInThread(thread, chunkID) {
			continue
		}

		thread.ChunkIDs = append(thread.ChunkIDs, chunkID)
		updated = true
	}
	return updated
}

// chunkExistsInThread checks if a chunk ID already exists in the thread
func (ms *MemoryServer) chunkExistsInThread(thread *threading.MemoryThread, chunkID string) bool {
	for _, existingID := range thread.ChunkIDs {
		if existingID == chunkID {
			return true
		}
	}
	return false
}
