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
	"mcp-memory/pkg/mcp"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/server"
	"mcp-memory/pkg/types"
	"strings"
	"time"
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
	mcpServer        *server.Server
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

	// Create MCP server
	mcpServer := mcp.NewServer("claude-memory", "1.0.0")
	memServer.mcpServer = mcpServer
	memServer.registerTools()
	memServer.registerResources()

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

// GetMCPServer returns the underlying MCP server for testing
func (ms *MemoryServer) GetMCPServer() *server.Server {
	return ms.mcpServer
}

// registerTools registers all MCP tools
func (ms *MemoryServer) registerTools() {
	// Register all MCP tools with proper schemas
	
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_store_chunk",
		"Store a conversation chunk in memory with automatic analysis and embedding generation",
		mcp.ObjectSchema("Store memory chunk parameters", map[string]interface{}{
			"content": mcp.StringParam("The conversation content to store", true),
			"session_id": mcp.StringParam("Session identifier for grouping related chunks", true),
			"repository": mcp.StringParam("Repository name (optional)", false),
			"branch": mcp.StringParam("Git branch name (optional)", false),
			"files_modified": mcp.ArraySchema("List of files that were modified", map[string]interface{}{"type": "string"}),
			"tools_used": mcp.ArraySchema("List of tools that were used", map[string]interface{}{"type": "string"}),
			"tags": mcp.ArraySchema("Additional tags for categorization", map[string]interface{}{"type": "string"}),
		}, []string{"content", "session_id"}),
	), mcp.ToolHandlerFunc(ms.handleStoreChunk))

	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_search",
		"Search for similar conversation chunks based on natural language query",
		mcp.ObjectSchema("Search memory parameters", map[string]interface{}{
			"query": mcp.StringParam("Natural language search query", true),
			"repository": mcp.StringParam("Filter by repository name (optional)", false),
			"recency": map[string]interface{}{
				"type": "string",
				"enum": []string{"recent", "last_month", "all_time"},
				"description": "Time filter for results",
				"default": "recent",
			},
			"types": mcp.ArraySchema("Filter by chunk types", map[string]interface{}{"type": "string"}),
			"limit": map[string]interface{}{
				"type": "integer",
				"description": "Maximum number of results",
				"default": 10,
				"minimum": 1,
				"maximum": 50,
			},
			"min_relevance": map[string]interface{}{
				"type": "number",
				"description": "Minimum relevance score (0-1)",
				"default": 0.7,
				"minimum": 0,
				"maximum": 1,
			},
		}, []string{"query"}),
	), mcp.ToolHandlerFunc(ms.handleSearch))

	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_get_context",
		"Get project context and recent activity for session initialization",
		mcp.ObjectSchema("Get context parameters", map[string]interface{}{
			"repository": mcp.StringParam("Repository name to get context for", true),
			"recent_days": map[string]interface{}{
				"type": "integer",
				"description": "Number of recent days to include",
				"default": 7,
				"minimum": 1,
				"maximum": 90,
			},
		}, []string{"repository"}),
	), mcp.ToolHandlerFunc(ms.handleGetContext))

	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_find_similar",
		"Find similar past problems and their solutions",
		mcp.ObjectSchema("Find similar parameters", map[string]interface{}{
			"problem": mcp.StringParam("Description of the current problem or error", true),
			"repository": mcp.StringParam("Repository context (optional)", false),
			"limit": map[string]interface{}{
				"type": "integer",
				"description": "Maximum number of similar problems to return",
				"default": 5,
				"minimum": 1,
				"maximum": 20,
			},
		}, []string{"problem"}),
	), mcp.ToolHandlerFunc(ms.handleFindSimilar))

	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_store_decision",
		"Store an architectural decision with rationale",
		mcp.ObjectSchema("Store decision parameters", map[string]interface{}{
			"decision": mcp.StringParam("The architectural decision made", true),
			"rationale": mcp.StringParam("Reasoning behind the decision", true),
			"context": mcp.StringParam("Additional context and alternatives considered", false),
			"repository": mcp.StringParam("Repository this decision applies to", false),
			"session_id": mcp.StringParam("Session identifier", true),
		}, []string{"decision", "rationale", "session_id"}),
	), mcp.ToolHandlerFunc(ms.handleStoreDecision))

	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_get_patterns",
		"Identify recurring patterns in project history",
		mcp.ObjectSchema("Get patterns parameters", map[string]interface{}{
			"repository": mcp.StringParam("Repository to analyze", true),
			"timeframe": map[string]interface{}{
				"type": "string",
				"enum": []string{"week", "month", "quarter", "all"},
				"description": "Time period to analyze",
				"default": "month",
			},
		}, []string{"repository"}),
	), mcp.ToolHandlerFunc(ms.handleGetPatterns))

	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_health",
		"Check the health status of the memory system",
		mcp.ObjectSchema("Health check parameters", map[string]interface{}{}, []string{}),
	), mcp.ToolHandlerFunc(ms.handleHealth))

	// Phase 3.2: Advanced MCP Tools
	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_suggest_related",
		"Get AI-powered suggestions for related context based on current work",
		mcp.ObjectSchema("Suggest related parameters", map[string]interface{}{
			"current_context": mcp.StringParam("Current work context or conversation content", true),
			"repository": mcp.StringParam("Repository to search for related context", false),
			"max_suggestions": map[string]interface{}{
				"type": "integer",
				"description": "Maximum number of suggestions to return",
				"minimum": 1,
				"maximum": 10,
				"default": 5,
			},
			"include_patterns": mcp.BooleanParam("Include pattern-based suggestions", false),
			"session_id": mcp.StringParam("Session identifier", true),
		}, []string{"current_context", "session_id"}),
	), mcp.ToolHandlerFunc(ms.handleSuggestRelated))

	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_export_project",
		"Export all memory data for a project in various formats",
		mcp.ObjectSchema("Export project parameters", map[string]interface{}{
			"repository": mcp.StringParam("Repository to export", true),
			"format": map[string]interface{}{
				"type": "string",
				"description": "Export format: json, markdown, or archive",
				"enum": []string{"json", "markdown", "archive"},
				"default": "json",
			},
			"include_vectors": mcp.BooleanParam("Include vector embeddings in export", false),
			"date_range": mcp.ObjectSchema("Date range filter", map[string]interface{}{
				"start": mcp.StringParam("Start date (ISO 8601 format)", false),
				"end": mcp.StringParam("End date (ISO 8601 format)", false),
			}, []string{}),
			"session_id": mcp.StringParam("Session identifier", true),
		}, []string{"repository", "session_id"}),
	), mcp.ToolHandlerFunc(ms.handleExportProject))

	ms.mcpServer.AddTool(mcp.NewTool(
		"memory_import_context",
		"Import conversation context from external source",
		mcp.ObjectSchema("Import context parameters", map[string]interface{}{
			"source": map[string]interface{}{
				"type": "string",
				"description": "Source type: conversation, file, or archive",
				"enum": []string{"conversation", "file", "archive"},
				"default": "conversation",
			},
			"data": mcp.StringParam("Data to import (conversation text, file content, or base64 archive)", true),
			"repository": mcp.StringParam("Target repository for imported data", true),
			"metadata": mcp.ObjectSchema("Import metadata", map[string]interface{}{
				"source_system": mcp.StringParam("Name of the source system", false),
				"import_date": mcp.StringParam("Original date of the content", false),
				"tags": mcp.ArraySchema("Tags to apply to imported content", map[string]interface{}{"type": "string"}),
			}, []string{}),
			"chunking_strategy": map[string]interface{}{
				"type": "string",
				"description": "How to chunk the imported data",
				"enum": []string{"auto", "paragraph", "fixed_size", "conversation_turns"},
				"default": "auto",
			},
			"session_id": mcp.StringParam("Session identifier", true),
		}, []string{"source", "data", "repository", "session_id"}),
	), mcp.ToolHandlerFunc(ms.handleImportContext))
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
		chunks, err := ms.vectorStore.ListByRepository(ctx, repository, 20, 0)
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
		chunks, err := ms.vectorStore.ListByRepository(ctx, repository, 100, 0)
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
	currentContext, ok := params["current_context"].(string)
	if !ok {
		return nil, fmt.Errorf("current_context is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("session_id is required")
	}

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
	embedding, err := ms.embeddingService.GenerateEmbedding(ctx, currentContext)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Search for similar content
	query := types.NewMemoryQuery(currentContext)
	query.Repository = &repository
	query.Limit = maxSuggestions * 2 // Get more to filter from
	query.MinRelevanceScore = 0.6

	results, err := ms.vectorStore.Search(ctx, *query, embedding)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	suggestions := []map[string]interface{}{}
	for i, result := range results.Results {
		if i >= maxSuggestions {
			break
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
	// Note: Pattern analysis temporarily disabled due to interface compatibility
	// TODO: Fix interface compatibility and re-enable pattern-based suggestions
	if includePatterns && ms.patternAnalyzer != nil {
		// patterns := ms.patternAnalyzer.AnalyzePatterns(currentContext)
		// Add a simple pattern-based suggestion for now
		if len(suggestions) < maxSuggestions {
			suggestion := map[string]interface{}{
				"content":         "Pattern-based suggestions coming soon",
				"summary":         "AI pattern analysis",
				"relevance_score": 0.5,
				"type":            "pattern_match",
				"pattern_type":    "placeholder",
			}
			suggestions = append(suggestions, suggestion)
		}
	}

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
	repository, ok := params["repository"].(string)
	if !ok {
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
	chunks, err := ms.vectorStore.ListByRepository(ctx, repository, 10000, 0) // Large limit for export
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
			"format":      "json",
			"data":        string(exportJSON),
			"size_bytes":  len(exportJSON),
			"chunks":      len(chunks),
			"repository":  repository,
			"session_id":  sessionID,
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
			"format":      "markdown",
			"data":        markdownData,
			"size_bytes":  len(markdownData),
			"chunks":      len(chunks),
			"repository":  repository,
			"session_id":  sessionID,
		}, nil

	case "archive":
		// Use backup manager to create compressed archive
		if ms.backupManager == nil {
			return nil, fmt.Errorf("backup manager not available")
		}

		// Create a filtered backup for this repository only
		backupData := map[string]interface{}{
			"repository":     repository,
			"export_date":    time.Now().Format(time.RFC3339),
			"chunks":         chunks,
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
			"format":      "archive",
			"data":        archiveB64,
			"size_bytes":  len(archiveJSON),
			"chunks":      len(chunks),
			"repository":  repository,
			"session_id":  sessionID,
			"encoding":    "base64",
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
	case "conversation":
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
		embedding, err := ms.embeddingService.GenerateEmbedding(ctx, chunk.Content)
		if err != nil {
			log.Printf("Failed to generate embedding for chunk %s: %v", chunk.ID, err)
			continue
		}

		chunk.Embeddings = embedding
		
		// Store chunk
		if err := ms.vectorStore.Store(ctx, chunk); err != nil {
			log.Printf("Failed to store chunk %s: %v", chunk.ID, err)
			continue
		}

		storedCount++
	}

	return map[string]interface{}{
		"source":           source,
		"repository":       repository,
		"chunks_processed": len(importedChunks),
		"chunks_stored":    storedCount,
		"chunking_strategy": chunkingStrategy,
		"session_id":       sessionID,
		"import_date":      time.Now().Format(time.RFC3339),
	}, nil
}

// Helper methods for import functionality

func (ms *MemoryServer) importConversationText(ctx context.Context, data, repository, _ string, metadata map[string]interface{}) ([]types.ConversationChunk, error) {
	// Create conversation chunks using the chunking service
	chunkMetadata := types.ChunkMetadata{
		Repository: repository,
		Tags:       []string{"imported", "conversation"},
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

	chunkData, err := ms.chunkingService.CreateChunk(ctx, "import", data, chunkMetadata)
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

	chunkData, err := ms.chunkingService.CreateChunk(ctx, "import", data, chunkMetadata)
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
	return ms.vectorStore.Close()
}
