// Package mcp provides MCP server implementation
package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"mcp-memory/internal/chunking"
	"mcp-memory/internal/config"
	"mcp-memory/internal/embeddings"
	"mcp-memory/internal/intelligence"
	"mcp-memory/internal/persistence"
	"mcp-memory/internal/storage"
	"mcp-memory/internal/workflow"
	"mcp-memory/pkg/types"
	"strings"
	"time"
	// "github.com/mark3labs/mcp-go/server" // TODO: Re-enable when MCP integration is complete
)

// MemoryServer implements the MCP server for Claude memory
type MemoryServer struct {
	config           *config.Config
	vectorStore      storage.VectorStore
	embeddingService embeddings.EmbeddingService
	chunkingService  *chunking.ChunkingService
	contextSuggester *workflow.ContextSuggester
	backupManager    *persistence.BackupManager
	learningEngine   *intelligence.LearningEngine
	patternAnalyzer  *workflow.PatternAnalyzer
	mcpServer        interface{} // *server.Server - simplified for now
}

// NewMemoryServer creates a new memory MCP server
func NewMemoryServer(cfg *config.Config) (*MemoryServer, error) {
	// Initialize vector store
	vectorStore := storage.NewChromaStore(&cfg.Chroma)

	// Initialize embedding service
	embeddingService := embeddings.NewOpenAIEmbeddingService(&cfg.OpenAI)

	// Initialize chunking service
	chunkingService := chunking.NewChunkingService(&cfg.Chunking, embeddingService)

	// Initialize intelligence layer components
	todoTracker := workflow.NewTodoTracker()
	flowDetector := workflow.NewFlowDetector()
	patternAnalyzer := workflow.NewPatternAnalyzer()
	// Note: Using nil for now due to interface compatibility issues - will be fixed in integration
	contextSuggester := workflow.NewContextSuggester(nil, patternAnalyzer, todoTracker, flowDetector)
	backupManager := persistence.NewBackupManager(nil, "./backups")
	patternEngine := intelligence.NewPatternEngine(nil)
	graphBuilder := intelligence.NewGraphBuilder(patternEngine)
	learningEngine := intelligence.NewLearningEngine(patternEngine, graphBuilder)

	memServer := &MemoryServer{
		config:           cfg,
		vectorStore:      vectorStore,
		embeddingService: embeddingService,
		chunkingService:  chunkingService,
		contextSuggester: contextSuggester,
		backupManager:    backupManager,
		learningEngine:   learningEngine,
		patternAnalyzer:  patternAnalyzer,
	}

	// Create MCP server (simplified for now)
	// mcpServer := server.NewServer(server.Options{
	//     Name:    "claude-memory",
	//     Version: "1.0.0",
	// })

	// memServer.mcpServer = mcpServer
	// memServer.registerTools()
	// memServer.registerResources()

	return memServer, nil
}

// Start initializes and starts the MCP server
func (ms *MemoryServer) Start(ctx context.Context) error {
	// Initialize vector store
	if err := ms.vectorStore.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize vector store: %w", err)
	}

	// Health check services
	if err := ms.embeddingService.HealthCheck(ctx); err != nil {
		log.Printf("Warning: Embedding service health check failed: %v", err)
	}

	if err := ms.vectorStore.HealthCheck(ctx); err != nil {
		log.Printf("Warning: Vector store health check failed: %v", err)
	}

	log.Printf("Claude Memory MCP Server started successfully")
	return nil
}

// registerTools registers all MCP tools
func (ms *MemoryServer) registerTools() {
	// TODO: Implement MCP tool registration
	// Core tools for Claude would be registered here
	/*
		ms.mcpServer.AddTool(server.Tool{
			Name:        "memory_store_chunk",
			Description: "Store a conversation chunk in memory with automatic analysis and embedding generation",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The conversation content to store",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session identifier for grouping related chunks",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository name (optional)",
					},
					"branch": map[string]interface{}{
						"type":        "string",
						"description": "Git branch name (optional)",
					},
					"files_modified": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of files that were modified",
					},
					"tools_used": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "List of tools that were used",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Additional tags for categorization",
					},
				},
				"required": []string{"content", "session_id"},
			},
		}, ms.handleStoreChunk)

		ms.mcpServer.AddTool(mcp.Tool{
			Name:        "memory_search",
			Description: "Search for similar conversation chunks based on natural language query",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Natural language search query",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Filter by repository name (optional)",
					},
					"recency": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"recent", "last_month", "all_time"},
						"description": "Time filter for results",
						"default":     "recent",
					},
					"types": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Filter by chunk types",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results",
						"default":     10,
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
				},
				"required": []string{"query"},
			},
		}, ms.handleSearch)

		ms.mcpServer.AddTool(mcp.Tool{
			Name:        "memory_get_context",
			Description: "Get project context and recent activity for session initialization",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository name to get context for",
					},
					"recent_days": map[string]interface{}{
						"type":        "integer",
						"description": "Number of recent days to include",
						"default":     7,
						"minimum":     1,
						"maximum":     90,
					},
				},
				"required": []string{"repository"},
			},
		}, ms.handleGetContext)

		ms.mcpServer.AddTool(mcp.Tool{
			Name:        "memory_find_similar",
			Description: "Find similar past problems and their solutions",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"problem": map[string]interface{}{
						"type":        "string",
						"description": "Description of the current problem or error",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository context (optional)",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of similar problems to return",
						"default":     5,
						"minimum":     1,
						"maximum":     20,
					},
				},
				"required": []string{"problem"},
			},
		}, ms.handleFindSimilar)

		// Advanced tools
		ms.mcpServer.AddTool(mcp.Tool{
			Name:        "memory_store_decision",
			Description: "Store an architectural decision with rationale",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"decision": map[string]interface{}{
						"type":        "string",
						"description": "The architectural decision made",
					},
					"rationale": map[string]interface{}{
						"type":        "string",
						"description": "Reasoning behind the decision",
					},
					"context": map[string]interface{}{
						"type":        "string",
						"description": "Additional context and alternatives considered",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository this decision applies to",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session identifier",
					},
				},
				"required": []string{"decision", "rationale", "session_id"},
			},
		}, ms.handleStoreDecision)

		ms.mcpServer.AddTool(mcp.Tool{
			Name:        "memory_get_patterns",
			Description: "Identify recurring patterns in project history",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository to analyze",
					},
					"timeframe": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"week", "month", "quarter", "all"},
						"description": "Time period to analyze",
						"default":     "month",
					},
				},
				"required": []string{"repository"},
			},
		}, ms.handleGetPatterns)

		ms.mcpServer.AddTool(mcp.Tool{
			Name:        "memory_health",
			Description: "Check the health status of the memory system",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		}, ms.handleHealth)

		// Phase 3.2: Advanced MCP Tools
		ms.mcpServer.AddTool(mcp.Tool{
			Name:        "memory_suggest_related",
			Description: "Get AI-powered suggestions for related context based on current work",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"current_context": map[string]interface{}{
						"type":        "string",
						"description": "Current work context or conversation content",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository to search for related context",
					},
					"max_suggestions": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of suggestions to return",
						"minimum":     1,
						"maximum":     10,
						"default":     5,
					},
					"include_patterns": map[string]interface{}{
						"type":        "boolean",
						"description": "Include pattern-based suggestions",
						"default":     true,
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session identifier",
					},
				},
				"required": []string{"current_context", "session_id"},
			},
		}, ms.handleSuggestRelated)

		ms.mcpServer.AddTool(mcp.Tool{
			Name:        "memory_export_project",
			Description: "Export all memory data for a project in various formats",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Repository to export",
					},
					"format": map[string]interface{}{
						"type":        "string",
						"description": "Export format: json, markdown, or archive",
						"enum":        []string{"json", "markdown", "archive"},
						"default":     "json",
					},
					"include_vectors": map[string]interface{}{
						"type":        "boolean",
						"description": "Include vector embeddings in export",
						"default":     false,
					},
					"date_range": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"start": map[string]interface{}{
								"type":        "string",
								"description": "Start date (ISO 8601 format)",
							},
							"end": map[string]interface{}{
								"type":        "string",
								"description": "End date (ISO 8601 format)",
							},
						},
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session identifier",
					},
				},
				"required": []string{"repository", "session_id"},
			},
		}, ms.handleExportProject)

		ms.mcpServer.AddTool(mcp.Tool{
			Name:        "memory_import_context",
			Description: "Import conversation context from external source",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"source": map[string]interface{}{
						"type":        "string",
						"description": "Source type: conversation, file, or archive",
						"enum":        []string{"conversation", "file", "archive"},
						"default":     "conversation",
					},
					"data": map[string]interface{}{
						"type":        "string",
						"description": "Data to import (conversation text, file content, or base64 archive)",
					},
					"repository": map[string]interface{}{
						"type":        "string",
						"description": "Target repository for imported data",
					},
					"metadata": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"source_system": map[string]interface{}{
								"type":        "string",
								"description": "Name of the source system",
							},
							"import_date": map[string]interface{}{
								"type":        "string",
								"description": "Original date of the content",
							},
							"tags": map[string]interface{}{
								"type":        "array",
								"items":       map[string]interface{}{"type": "string"},
								"description": "Tags to apply to imported content",
							},
						},
					},
					"chunking_strategy": map[string]interface{}{
						"type":        "string",
						"description": "How to chunk the imported data",
						"enum":        []string{"auto", "paragraph", "fixed_size", "conversation_turns"},
						"default":     "auto",
					},
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Session identifier",
					},
				},
				"required": []string{"source", "data", "repository", "session_id"},
			},
		}, ms.handleImportContext)
	*/
}

// registerResources registers MCP resources for browsing memory
func (ms *MemoryServer) registerResources() {
	// TODO: Implement MCP resource registration
	/*
		resources := []server.Resource{
			{
				URI:         "memory://recent/{repository}",
				Name:        "Recent Activity",
				Description: "Recent conversation chunks for a repository",
				MimeType:    "application/json",
			},
			{
				URI:         "memory://patterns/{repository}",
				Name:        "Common Patterns",
				Description: "Identified patterns in project history",
				MimeType:    "application/json",
			},
			{
				URI:         "memory://decisions/{repository}",
				Name:        "Architectural Decisions",
				Description: "Key architectural decisions made",
				MimeType:    "application/json",
			},
			{
				URI:         "memory://global/insights",
				Name:        "Global Insights",
				Description: "Cross-project insights and patterns",
				MimeType:    "application/json",
			},
		}

		for _, resource := range resources {
			ms.mcpServer.AddResource(resource, ms.handleResourceRead)
		}
	*/
}

// Tool handlers

func (ms *MemoryServer) handleStoreChunk(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	content, ok := params["content"].(string)
	if !ok || content == "" {
		return nil, fmt.Errorf("content is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}

	// Build metadata from parameters
	metadata := types.ChunkMetadata{
		Outcome:    types.OutcomeInProgress, // Default
		Difficulty: types.DifficultySimple,  // Default
	}

	if repo, ok := params["repository"].(string); ok {
		metadata.Repository = repo
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

	// Create and store chunk
	chunk, err := ms.chunkingService.CreateChunk(ctx, sessionID, content, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}

	if err := ms.vectorStore.Store(ctx, *chunk); err != nil {
		return nil, fmt.Errorf("failed to store chunk: %w", err)
	}

	return map[string]interface{}{
		"chunk_id":  chunk.ID,
		"type":      string(chunk.Type),
		"summary":   chunk.Summary,
		"stored_at": chunk.Timestamp.Format(time.RFC3339),
	}, nil
}

func (ms *MemoryServer) handleSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}

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
	embeddings, err := ms.embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embeddings: %w", err)
	}

	// Search vector store
	results, err := ms.vectorStore.Search(ctx, *memQuery, embeddings)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Format results for response
	response := map[string]interface{}{
		"query":      query,
		"total":      results.Total,
		"query_time": results.QueryTime.String(),
		"results":    []map[string]interface{}{},
	}

	for _, result := range results.Results {
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
		response["results"] = append(response["results"].([]map[string]interface{}), resultMap)
	}

	return response, nil
}

func (ms *MemoryServer) handleGetContext(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository is required")
	}

	recentDays := 7
	if days, ok := params["recent_days"].(float64); ok {
		recentDays = int(days)
	}

	// Get recent chunks for the repository
	chunks, err := ms.vectorStore.ListByRepository(ctx, repository, 20, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository chunks: %w", err)
	}

	// Filter by recent days
	cutoff := time.Now().AddDate(0, 0, -recentDays)
	recentChunks := []types.ConversationChunk{}
	for _, chunk := range chunks {
		if chunk.Timestamp.After(cutoff) {
			recentChunks = append(recentChunks, chunk)
		}
	}

	// Analyze patterns
	patterns := ms.analyzePatterns(recentChunks)
	decisions := ms.extractDecisions(recentChunks)
	techStack := ms.extractTechStack(recentChunks)

	// Build project context
	context := types.NewProjectContext(repository)
	context.TotalSessions = len(recentChunks)
	context.CommonPatterns = patterns
	context.ArchitecturalDecisions = decisions
	context.TechStack = techStack

	return map[string]interface{}{
		"repository":              context.Repository,
		"last_accessed":           context.LastAccessed.Format(time.RFC3339),
		"total_recent_sessions":   len(recentChunks),
		"common_patterns":         context.CommonPatterns,
		"architectural_decisions": context.ArchitecturalDecisions,
		"tech_stack":              context.TechStack,
		"recent_activity": func() []map[string]interface{} {
			activity := []map[string]interface{}{}
			for i, chunk := range recentChunks {
				if i >= 5 { // Limit to 5 most recent
					break
				}
				activity = append(activity, map[string]interface{}{
					"type":      string(chunk.Type),
					"summary":   chunk.Summary,
					"timestamp": chunk.Timestamp.Format(time.RFC3339),
					"outcome":   string(chunk.Metadata.Outcome),
				})
			}
			return activity
		}(),
	}, nil
}

func (ms *MemoryServer) handleFindSimilar(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	problem, ok := params["problem"].(string)
	if !ok || problem == "" {
		return nil, fmt.Errorf("problem description is required")
	}

	limit := 5
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	// Build search query focusing on problems and solutions
	memQuery := types.NewMemoryQuery(problem)
	memQuery.Types = []types.ChunkType{types.ChunkTypeProblem, types.ChunkTypeSolution}
	memQuery.Limit = limit
	memQuery.MinRelevanceScore = 0.6 // Lower threshold for problem matching

	if repo, ok := params["repository"].(string); ok && repo != "" {
		memQuery.Repository = &repo
	}

	// Generate embeddings and search
	embeddings, err := ms.embeddingService.GenerateEmbedding(ctx, problem)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	results, err := ms.vectorStore.Search(ctx, *memQuery, embeddings)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

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
		metadata.Repository = repo
	}

	// Create and store chunk
	chunk, err := ms.chunkingService.CreateChunk(ctx, sessionID, content, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create decision chunk: %w", err)
	}

	// Override type to architecture decision
	chunk.Type = types.ChunkTypeArchitectureDecision

	if err := ms.vectorStore.Store(ctx, *chunk); err != nil {
		return nil, fmt.Errorf("failed to store decision: %w", err)
	}

	return map[string]interface{}{
		"chunk_id":  chunk.ID,
		"decision":  decision,
		"stored_at": chunk.Timestamp.Format(time.RFC3339),
	}, nil
}

func (ms *MemoryServer) handleGetPatterns(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		return nil, fmt.Errorf("repository is required")
	}

	timeframe := "month"
	if tf, ok := params["timeframe"].(string); ok {
		timeframe = tf
	}

	// Get chunks based on timeframe
	var chunks []types.ConversationChunk
	var err error

	// For now, get recent chunks - in a full implementation,
	// this would have proper time filtering
	chunks, err = ms.vectorStore.ListByRepository(ctx, repository, 100, 0)
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
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  map[string]interface{}{},
	}

	// Check vector store
	if err := ms.vectorStore.HealthCheck(ctx); err != nil {
		health["services"].(map[string]interface{})["vector_store"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "degraded"
	} else {
		health["services"].(map[string]interface{})["vector_store"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Check embedding service
	if err := ms.embeddingService.HealthCheck(ctx); err != nil {
		health["services"].(map[string]interface{})["embedding_service"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "degraded"
	} else {
		health["services"].(map[string]interface{})["embedding_service"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Get statistics
	if stats, err := ms.vectorStore.GetStats(ctx); err == nil {
		health["stats"] = stats
	}

	return health, nil
}

// Resource handler

func (ms *MemoryServer) handleResourceRead(ctx context.Context, uri string) (interface{}, error) {
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
		chunks, err := ms.vectorStore.ListByRepository(ctx, repository, 20, 0)
		if err != nil {
			return nil, err
		}
		return chunks, nil

	case "patterns":
		if len(parts) < 4 {
			return nil, fmt.Errorf("repository required for patterns resource")
		}
		repository := parts[3]
		chunks, err := ms.vectorStore.ListByRepository(ctx, repository, 100, 0)
		if err != nil {
			return nil, err
		}
		patterns := ms.analyzePatterns(chunks)
		return map[string]interface{}{
			"repository": repository,
			"patterns":   patterns,
		}, nil

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

		embeddings, err := ms.embeddingService.GenerateEmbedding(ctx, "architectural decision")
		if err != nil {
			return nil, err
		}

		results, err := ms.vectorStore.Search(ctx, *memQuery, embeddings)
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

		return map[string]interface{}{
			"repository": repository,
			"decisions":  decisions,
		}, nil

	case "global":
		if len(parts) < 4 || parts[3] != "insights" {
			return nil, fmt.Errorf("invalid global resource")
		}

		// Get global insights across all repositories
		// This is a simplified implementation
		return map[string]interface{}{
			"message": "Global insights feature coming soon",
			"status":  "not_implemented",
		}, nil

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

// GetServer returns the underlying MCP server
func (ms *MemoryServer) GetServer() interface{} {
	return ms.mcpServer
}

// Close closes all connections
func (ms *MemoryServer) Close() error {
	return ms.vectorStore.Close()
}
