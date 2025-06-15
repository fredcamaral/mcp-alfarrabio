// Package mcp provides MCP server implementation
package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	// "lerian-mcp-memory/internal/ai" // Disabled again - interface adapter needed
	"lerian-mcp-memory/internal/audit"
	"lerian-mcp-memory/internal/bulk"
	"lerian-mcp-memory/internal/config"
	contextpkg "lerian-mcp-memory/internal/context"
	"lerian-mcp-memory/internal/di"
	mcperrors "lerian-mcp-memory/internal/errors"
	"lerian-mcp-memory/internal/intelligence"
	"lerian-mcp-memory/internal/logging"
	"lerian-mcp-memory/internal/relationships"
	"lerian-mcp-memory/internal/storage"
	"lerian-mcp-memory/internal/sync"
	"lerian-mcp-memory/internal/templates"
	"lerian-mcp-memory/internal/threading"
	"lerian-mcp-memory/internal/workflow"
	"lerian-mcp-memory/pkg/types"
	"log"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	mcp "github.com/fredcamaral/gomcp-sdk"
	"github.com/fredcamaral/gomcp-sdk/protocol"
	"github.com/fredcamaral/gomcp-sdk/server"
	"github.com/google/uuid"
)

// String constants for repeated values
const (
	StatusSuccess  = "success"
	StatusDegraded = "degraded"
	LevelMedium    = "medium"
	LevelHigh      = "high"
	ValueUnknown   = "unknown"
	ValueDefault   = "default"
)

// MemoryServer implements the MCP server for Claude memory
type MemoryServer struct {
	container *di.Container
	mcpServer *server.Server
	logger    *logging.EnhancedLogger

	// Bulk operations managers
	bulkManager  *bulk.Manager
	bulkImporter *bulk.Importer
	bulkExporter *bulk.Exporter
	aliasManager *bulk.AliasManager

	// Workflow tracking
	todoTracker *workflow.TodoTracker

	// Real-time synchronization
	realtimeSync *sync.RealtimeSyncCoordinator

	// Template management service
	templateService *templates.TemplateService

	// AI-powered enhancement services
	// memoryEnhancer  *ai.MemoryEnhancer - disabled until MemoryEnhancer is available
	// patternDetector *ai.PatternDetector - disabled until interface adapter is created
}

// NewMemoryServer creates a new memory MCP server
func NewMemoryServer(cfg *config.Config) (*MemoryServer, error) {
	// Create enhanced logger for MCP server
	logger := logging.NewEnhancedLogger("mcp")

	// Create dependency injection container
	container, err := di.NewContainer(cfg)
	if err != nil {
		return nil, mcperrors.WrapMCPError(
			fmt.Errorf("failed to create DI container: %w", err),
			"server_initialization",
		)
	}

	memServer := &MemoryServer{
		container: container,
		logger:    logger,
	}

	// Initialize bulk operations managers
	memServer.initializeBulkManagers()

	// Initialize workflow tracking
	memServer.todoTracker = workflow.NewTodoTracker()

	// Initialize real-time synchronization
	memServer.realtimeSync = container.GetRealtimeSyncCoordinator()

	// AI enhancement services temporarily disabled due to interface mismatches
	logger.Info("AI enhancement services temporarily disabled - interface adapter needed")

	// Initialize template service
	if err := memServer.initializeTemplateService(); err != nil {
		logger.Warn("Failed to initialize template service, continuing with limited functionality", "error", err)
		// Continue without template service - server will work with basic functionality
	}

	// Create MCP server with error handling
	if err := memServer.initializeMCPServer(); err != nil {
		return nil, mcperrors.WrapMCPError(err, "mcp_server_initialization")
	}

	logger.Info("MCP Memory Server created successfully")
	return memServer, nil
}

// initializeBulkManagers initializes bulk operation managers
func (ms *MemoryServer) initializeBulkManagers() {
	// Create standard library logger for bulk managers compatibility
	stdLogger := log.New(os.Stdout, "[bulk] ", log.LstdFlags)

	ms.bulkManager = bulk.NewManager(ms.container.GetVectorStore(), stdLogger)
	ms.bulkImporter = bulk.NewImporter(stdLogger)
	ms.bulkExporter = bulk.NewExporter(ms.container.GetVectorStore(), stdLogger)
	ms.aliasManager = bulk.NewAliasManager(ms.container.GetVectorStore(), stdLogger)
}

// initializeTemplateService initializes the template service
func (ms *MemoryServer) initializeTemplateService() error {
	ms.logger.Info("Initializing template service")

	// Create template storage adapter - use vector store for both vector and content storage
	vectorStore := ms.container.GetVectorStore()

	if vectorStore == nil {
		return errors.New("vector store not available for template service")
	}

	// Create real PostgreSQL content store for template persistence
	// Priority 1: Replace mock with real PostgreSQL implementation (CRITICAL)
	contentStore := storage.NewPostgreSQLContentStore(ms.container.DB)
	templateStorage := templates.NewCleanStorageAdapter(vectorStore, contentStore)

	// Create template service with enhanced logger
	// Convert enhanced logger to slog.Logger - template service expects *slog.Logger
	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ms.templateService = templates.NewTemplateService(templateStorage, slogLogger)

	ms.logger.Info("Template service successfully initialized")
	return nil
}

// initializeMCPServer initializes the MCP server with tools and resources
func (ms *MemoryServer) initializeMCPServer() error {
	serverName := getEnv("SERVICE_NAME", "claude-memory")
	serverVersion := getEnv("SERVICE_VERSION", "VERSION_PLACEHOLDER")

	mcpServer := mcp.NewServer(serverName, serverVersion)
	if mcpServer == nil {
		return errors.New("failed to create MCP server instance")
	}

	ms.mcpServer = mcpServer

	ms.registerTools()

	ms.registerResources()

	return nil
}

// Start initializes and starts the MCP server
func (ms *MemoryServer) Start(ctx context.Context) error {
	// Create enhanced context for startup operations
	ctx = contextpkg.NewBuilder(ctx).
		WithComponent("mcp").
		WithOperation("server_start").
		WithCurrentTime().
		Build()

	// Use logger with context-aware trace ID
	//nolint:contextcheck // GetTraceID properly extracts from parent context
	traceID := contextpkg.GetTraceID(ctx)
	logger := ms.logger.WithTraceID(traceID)
	startTime := time.Now()

	logger.Info("Starting MCP Memory Server")

	// Initialize vector store with timeout
	storageCtx, cancel := contextpkg.NewStorageContext(ctx, "initialize")
	defer cancel()

	//nolint:contextcheck // storageCtx properly inherits from parent ctx via NewStorageContext
	if err := ms.container.GetVectorStore().Initialize(storageCtx); err != nil {
		return mcperrors.WrapStorageError(err, "vector_store_initialization")
	}
	logger.Info("Vector store initialized successfully")

	// Health check services with proper error handling
	//nolint:contextcheck // ctx is properly passed from parent
	if err := ms.container.HealthCheck(ctx); err != nil {
		// Log warning but don't fail startup for health check issues
		logger.Warn("Service health check failed, but continuing startup", "error", err.Error())
	} else {
		logger.Info("Service health check passed")
	}

	// Start automatic decay management for old chunks
	//nolint:contextcheck // goroutine gets independent context for background task
	go ms.runPeriodicDecayWithErrorHandling(ctx)

	// Log successful startup with duration
	duration := time.Since(startTime)
	logger.Info("MCP Memory Server started successfully",
		"startup_duration_ms", duration.Milliseconds())
	return nil
}

// runPeriodicDecayWithErrorHandling runs periodic decay with proper error handling
func (ms *MemoryServer) runPeriodicDecayWithErrorHandling(ctx context.Context) {
	logger := ms.logger.WithContext(ctx)

	logger.Info("Starting periodic decay management")

	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in periodic decay management", "panic_value", r)
		}
	}()

	// Run the original periodic decay method but with error handling
	ms.runPeriodicDecay(ctx)
}

// GetMCPServer returns the underlying MCP server for testing
func (ms *MemoryServer) GetMCPServer() *server.Server {
	return ms.mcpServer
}

// GetContainer returns the DI container for accessing services
func (ms *MemoryServer) GetContainer() *di.Container {
	return ms.container
}

// SetWebSocketHub sets the WebSocket hub for broadcasting memory updates
func (ms *MemoryServer) SetWebSocketHub(hub interface{}) {
	// Store hub in a type-safe way - we'll access it when needed
	// For now, just log that it's been set
	if hub != nil {
		log.Printf("WebSocket hub configured for memory updates")
		// In a more complete implementation, we'd store this and use it
		// when memory operations occur to broadcast changes
	}
}

// registerTools registers all MCP tools
func (ms *MemoryServer) registerTools() {
	logger := ms.logger

	// Register the 11 consolidated MCP tools for optimal client compatibility
	logger.Info("Registering 11 consolidated MCP tools")
	ms.registerConsolidatedTools()
	logger.Info("MCP tools registered successfully")
}

// registerResources registers MCP resources for browsing memory
func (ms *MemoryServer) registerResources() {
	logger := ms.logger

	logger.Info("Registering MCP resources")
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

	logger.Info("MCP resources registered successfully")
}

// Tool handlers

// validateStoreChunkParams validates required parameters for storing a chunk
func (ms *MemoryServer) validateStoreChunkParams(params map[string]interface{}) (content, sessionID string, err error) {
	var ok bool
	content, ok = params["content"].(string)
	if !ok || content == "" {
		logging.Error("memory_store_chunk failed: missing content parameter")
		return "", "", errors.New("content parameter is required and must be non-empty string. Example: {\"content\": \"Fixed authentication bug by updating JWT validation\", \"session_id\": \"auth-fix-session\"}")
	}

	sessionID, ok = params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_store_chunk failed: missing session_id parameter")
		return "", "", errors.New("session_id parameter is required and must be non-empty string. Use descriptive session IDs. Example: {\"session_id\": \"bug-fix-2024\", \"content\": \"Solution details\"}")
	}

	return content, sessionID, nil
}

// processParentChildRelationship creates parent-child relationship if specified
func (ms *MemoryServer) processParentChildRelationship(ctx context.Context, chunk *types.ConversationChunk, metadata *types.ChunkMetadata) {
	if metadata.ExtendedMetadata == nil {
		return
	}

	parentID, ok := metadata.ExtendedMetadata[types.EMKeyParentChunk].(string)
	if !ok || parentID == "" {
		return
	}

	relMgr := ms.container.GetRelationshipManager()
	_, err := relMgr.AddRelationship(ctx, parentID, chunk.ID, relationships.RelTypeParentChild, 1.0, "Explicit parent-child relationship")
	if err != nil {
		logging.Warn("Failed to create parent-child relationship", "error", err, "parent", parentID, "child", chunk.ID)
	}
}

// logStoreChunkAudit logs audit events for chunk storage operations
func (ms *MemoryServer) logStoreChunkAudit(ctx context.Context, chunk *types.ConversationChunk, sessionID string, startTime time.Time, storeErr error) {
	auditLogger := ms.container.GetAuditLogger()
	if auditLogger == nil {
		return
	}

	if storeErr != nil {
		auditLogger.LogError(ctx, audit.EventTypeMemoryStore, "Failed to store memory chunk", "memory", storeErr, map[string]interface{}{
			"chunk_id":   chunk.ID,
			"chunk_type": string(chunk.Type),
			"repository": chunk.Metadata.Repository,
			"session_id": sessionID,
		})
		return
	}

	auditLogger.LogEventWithDuration(ctx, audit.EventTypeMemoryStore, "Stored memory chunk", "memory", chunk.ID,
		time.Since(startTime), map[string]interface{}{
			"chunk_type":  string(chunk.Type),
			"repository":  chunk.Metadata.Repository,
			"session_id":  sessionID,
			"tags":        chunk.Metadata.Tags,
			"files_count": len(chunk.Metadata.FilesModified),
			"tools_count": len(chunk.Metadata.ToolsUsed),
			"has_parent":  chunk.Metadata.ExtendedMetadata != nil && chunk.Metadata.ExtendedMetadata[types.EMKeyParentChunk] != nil,
		})
}

// autoDetectRelationships detects and creates relationships with recent chunks
func (ms *MemoryServer) autoDetectRelationships(ctx context.Context, chunk *types.ConversationChunk) {
	go func() {
		repo := chunk.Metadata.Repository
		query := types.MemoryQuery{
			Repository: &repo,
			Limit:      20,
			Recency:    types.RecencyRecent,
		}

		results, err := ms.container.GetVectorStore().Search(ctx, &query, chunk.Embeddings)
		if err != nil || len(results.Results) == 0 {
			return
		}

		existingChunks := make([]types.ConversationChunk, 0, len(results.Results))
		for i := range results.Results {
			result := &results.Results[i]
			if result.Chunk.ID != chunk.ID { // Don't include self
				existingChunks = append(existingChunks, result.Chunk)
			}
		}

		relMgr := ms.container.GetRelationshipManager()
		detected := relMgr.DetectRelationships(ctx, chunk, existingChunks)
		logging.Info("Auto-detected relationships", "chunk_id", chunk.ID, "count", len(detected))
	}()
}

func (ms *MemoryServer) handleStoreChunk(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_store_chunk called", "params", params)

	// Validate required parameters
	content, sessionID, err := ms.validateStoreChunkParams(params)
	if err != nil {
		return nil, err
	}

	// Validate and normalize session ID for proper isolation
	sessionID = ms.validateAndNormalizeSessionID(sessionID)
	logging.Info("Processing chunk storage", "content_length", len(content), "session_id", sessionID)

	// Build metadata from parameters
	metadata := ms.buildMetadataFromParams(params)

	// Create repository-scoped session ID for multi-tenant isolation
	repositoryScopedSessionID := ms.createRepositoryScopedSessionID(metadata.Repository, sessionID)
	logging.Info("Created repository-scoped session", "original_session", sessionID, "scoped_session", repositoryScopedSessionID, "repository", metadata.Repository)

	// Add extended metadata with context detection
	if err := ms.addContextMetadata(&metadata, params); err != nil {
		logging.Warn("Failed to add context metadata", "error", err)
	}

	// Create and store chunk with repository-scoped session ID
	logging.Info("Creating conversation chunk", "session_id", repositoryScopedSessionID)
	chunk, err := ms.container.GetChunkingService().CreateChunk(ctx, repositoryScopedSessionID, content, &metadata)
	if err != nil {
		logging.Error("Failed to create chunk", "error", err, "session_id", repositoryScopedSessionID)
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}
	logging.Info("Chunk created successfully", "chunk_id", chunk.ID, "type", chunk.Type)

	// Process parent-child relationship if specified
	ms.processParentChildRelationship(ctx, chunk, &metadata)

	logging.Info("Storing chunk in vector store", "chunk_id", chunk.ID)
	startTime := time.Now()

	// Store chunk in vector store
	storeErr := ms.container.GetVectorStore().Store(ctx, chunk)
	if storeErr != nil {
		logging.Error("Failed to store chunk", "error", storeErr, "chunk_id", chunk.ID)
		ms.logStoreChunkAudit(ctx, chunk, sessionID, startTime, storeErr)
		return nil, fmt.Errorf("failed to store chunk: %w", storeErr)
	}

	// Log successful audit event
	ms.logStoreChunkAudit(ctx, chunk, sessionID, startTime, nil)

	// Auto-detect relationships with recent chunks
	ms.autoDetectRelationships(ctx, chunk)

	// Broadcast real-time event to connected clients
	if ms.realtimeSync != nil {
		if err := ms.realtimeSync.BroadcastChunkEvent(ctx, sync.EventTypeChunkCreated, chunk); err != nil {
			// Log error but don't fail the operation
			logging.Warn("Failed to broadcast chunk created event", "error", err, "chunk_id", chunk.ID)
		}
	}

	logging.Info("memory_store_chunk completed successfully", "chunk_id", chunk.ID, "session_id", sessionID)
	return map[string]interface{}{
		"chunk_id":  chunk.ID,
		"type":      string(chunk.Type),
		"summary":   chunk.Summary,
		"stored_at": chunk.Timestamp.Format(time.RFC3339),
	}, nil
}

// validateSearchParams validates required parameters for search
func (ms *MemoryServer) validateSearchParams(params map[string]interface{}) (string, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		logging.Error("memory_search failed: missing query parameter")
		return "", errors.New("query parameter is required and must be non-empty string. Use specific search terms. Example: {\"query\": \"authentication bug fix\", \"repository\": \"github.com/user/repo\"}")
	}
	return query, nil
}

// buildMemoryQueryFromParams builds a MemoryQuery from request parameters
func (ms *MemoryServer) buildMemoryQueryFromParams(query string, params map[string]interface{}) *types.MemoryQuery {
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

	return memQuery
}

// logSearchAudit logs audit events for search operations
func (ms *MemoryServer) logSearchAudit(ctx context.Context, query string, memQuery *types.MemoryQuery, results *types.SearchResults, searchStart time.Time, searchErr error) {
	auditLogger := ms.container.GetAuditLogger()
	if auditLogger == nil {
		return
	}

	if searchErr != nil {
		auditLogger.LogError(ctx, audit.EventTypeMemorySearch, "Memory search failed", "memory", searchErr, map[string]interface{}{
			"query":      query,
			"repository": memQuery.Repository,
			"limit":      memQuery.Limit,
		})
		return
	}

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

// formatSearchResults formats search results for response
func (ms *MemoryServer) formatSearchResults(ctx context.Context, query string, results *types.SearchResults) map[string]interface{} {
	response := map[string]interface{}{
		"query":      query,
		"total":      results.Total,
		"query_time": results.QueryTime.String(),
		"results":    []map[string]interface{}{},
	}

	relMgr := ms.container.GetRelationshipManager()
	analytics := ms.container.GetMemoryAnalytics()

	for i := range results.Results {
		result := &results.Results[i]

		// Track access to this memory
		if analytics != nil {
			if err := analytics.RecordAccess(ctx, result.Chunk.ID); err != nil {
				logging.Warn("Failed to record memory access", "chunk_id", result.Chunk.ID, "error", err)
			}
		}

		resultMap := ms.buildResultMap(result, relMgr)
		response["results"] = append(response["results"].([]map[string]interface{}), resultMap)
	}

	return response
}

// buildResultMap builds a result map for a single search result
func (ms *MemoryServer) buildResultMap(result *types.SearchResult, relMgr *relationships.Manager) map[string]interface{} {
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
	ms.addRelationshipInfo(resultMap, result.Chunk.ID, relMgr)

	// Add extended metadata if present
	if result.Chunk.Metadata.ExtendedMetadata != nil {
		resultMap["extended_metadata"] = result.Chunk.Metadata.ExtendedMetadata
	}

	return resultMap
}

// addRelationshipInfo adds relationship information to the result map
func (ms *MemoryServer) addRelationshipInfo(resultMap map[string]interface{}, chunkID string, relMgr *relationships.Manager) {
	chunkRelationships := relMgr.GetRelationships(chunkID)
	if len(chunkRelationships) == 0 {
		return
	}

	relInfo := make([]map[string]interface{}, 0, len(chunkRelationships))
	for _, rel := range chunkRelationships {
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

func (ms *MemoryServer) handleSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_search called", "params", params)

	// Validate required parameters
	query, err := ms.validateSearchParams(params)
	if err != nil {
		return nil, err
	}

	logging.Info("Processing search query", "query", query)

	// Build memory query from parameters
	memQuery := ms.buildMemoryQueryFromParams(query, params)

	// Generate embeddings for query
	logging.Info("Generating embeddings for search query", "query", query)
	embeddings, err := ms.container.GetEmbeddingService().Generate(ctx, query)
	if err != nil {
		logging.Error("Failed to generate embeddings", "error", err, "query", query)
		return nil, fmt.Errorf("failed to generate query embeddings: %w", err)
	}
	logging.Info("Embeddings generated successfully", "dimension", len(embeddings))

	// Execute progressive search with relaxation strategy
	searchStart := time.Now()
	results, err := ms.executeProgressiveSearch(ctx, memQuery, embeddings)
	if err != nil {
		logging.Error("Progressive search failed", "error", err, "query", query)
		ms.logSearchAudit(ctx, query, memQuery, nil, searchStart, err)
		return nil, fmt.Errorf("search failed: %w", err)
	}
	logging.Info("Progressive search completed", "total_results", results.Total, "query_time", results.QueryTime)

	// Log successful search audit event
	ms.logSearchAudit(ctx, query, memQuery, results, searchStart, nil)

	// Format results for response
	response := ms.formatSearchResults(ctx, query, results)

	logging.Info("memory_search completed successfully", "total_results", results.Total, "query", query)
	return response, nil
}

// executeProgressiveSearch implements a fallback strategy for searches
// Tries progressively looser search criteria if initial search returns no results
func (ms *MemoryServer) executeProgressiveSearch(ctx context.Context, query *types.MemoryQuery, embeddings []float64) (*types.SearchResults, error) {
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
	relaxedQuery := *query // Copy the query
	relaxedQuery.MinRelevanceScore = searchConfig.RelaxedMinRelevance
	logging.Info("Progressive search: Step 2 - Relaxed relevance", "min_relevance", relaxedQuery.MinRelevanceScore)
	results, err = ms.container.GetVectorStore().Search(ctx, &relaxedQuery, embeddings)
	if err != nil {
		return nil, err
	}

	if len(results.Results) > 0 {
		logging.Info("Progressive search: Relaxed search succeeded", "results", len(results.Results))
		return results, nil
	}

	// Step 3: Try related repositories if original repo specified
	if query.Repository != nil && searchConfig.EnableRepositoryFallback {
		results, err := ms.searchRelatedRepositories(ctx, &relaxedQuery, embeddings, *query.Repository, searchConfig)
		if err == nil && len(results.Results) > 0 {
			return results, nil
		}

		// Step 3b: Complete repository fallback (remove filter)
		repoFallbackQuery := relaxedQuery
		repoFallbackQuery.Repository = nil
		logging.Info("Progressive search: Step 3b - Complete repository fallback", "original_repo", *query.Repository)
		results, err = ms.container.GetVectorStore().Search(ctx, &repoFallbackQuery, embeddings)
		if err != nil {
			return nil, err
		}

		if len(results.Results) > 0 {
			logging.Info("Progressive search: Complete repository fallback succeeded", "results", len(results.Results))
			return results, nil
		}
	}

	// Step 4: Broadest search - remove type filters too
	broadQuery := *query // Copy the query
	broadQuery.MinRelevanceScore = searchConfig.BroadestMinRelevance
	broadQuery.Repository = nil
	broadQuery.Types = nil
	logging.Info("Progressive search: Step 4 - Broadest search", "min_relevance", broadQuery.MinRelevanceScore)
	results, err = ms.container.GetVectorStore().Search(ctx, &broadQuery, embeddings)
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
		return nil, errors.New("repository parameter is required and must be non-empty string. Use full repository URLs. Example: {\"repository\": \"github.com/user/project\", \"recent_days\": 7}")
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

	for i := range chunks {
		chunk := &chunks[i]
		if chunk.Timestamp.After(cutoff) {
			recentChunks = append(recentChunks, *chunk)
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
	projectContext := types.NewProjectContext(repository)
	projectContext.TotalSessions = len(recentChunks)
	projectContext.CommonPatterns = patterns
	projectContext.ArchitecturalDecisions = decisions
	projectContext.TechStack = techStack

	result := map[string]interface{}{
		"repository":              projectContext.Repository,
		"last_accessed":           projectContext.LastAccessed.Format(time.RFC3339),
		"total_recent_sessions":   len(recentChunks),
		"common_patterns":         projectContext.CommonPatterns,
		"architectural_decisions": projectContext.ArchitecturalDecisions,
		"tech_stack":              projectContext.TechStack,

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

	for i := range chunks {
		chunk := &chunks[i]
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
			for j := range chunks {
				otherChunk := &chunks[j]
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
	for i := range chunks {
		chunk := &chunks[i]
		sessions[chunk.SessionID] = append(sessions[chunk.SessionID], *chunk)
	}

	successCount := 0
	problemCount := 0
	lastSession := ""
	lastTimestamp := time.Time{}

	for sessionID, sessionChunks := range sessions {
		for i := range sessionChunks {
			chunk := &sessionChunks[i]
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

	for i := range sortedChunks {
		if i >= limit {
			break
		}
		chunk := &sortedChunks[i]
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
func (ms *MemoryServer) generateContextSuggestions(_ context.Context, _ string, chunks []types.ConversationChunk) []map[string]interface{} {
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

	for i := range chunks {
		chunk := &chunks[i]
		if chunk.Type == types.ChunkTypeProblem {
			problemCount++
			if len(recentProblems) < 3 {
				recentProblems = append(recentProblems, *chunk)
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
	for i := range chunks {
		chunk := &chunks[i]
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

func (ms *MemoryServer) handleStoreDecision(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Validate required parameters
	decision, rationale, sessionID, err := ms.validateStoreDecisionParams(params)
	if err != nil {
		return nil, err
	}

	// Build decision content and metadata
	content := ms.buildDecisionContent(decision, rationale, params)
	metadata := ms.buildDecisionMetadata(params, decision, rationale)

	// Create and store decision chunk
	chunk, err := ms.createDecisionChunk(ctx, sessionID, content, &metadata)
	if err != nil {
		return nil, err
	}

	// Broadcast real-time event to connected clients
	if ms.realtimeSync != nil {
		if err := ms.realtimeSync.BroadcastChunkEvent(ctx, sync.EventTypeChunkCreated, chunk); err != nil {
			// Log error but don't fail the operation
			logging.Warn("Failed to broadcast decision created event", "error", err, "chunk_id", chunk.ID)
		}
	}

	return map[string]interface{}{
		"chunk_id":  chunk.ID,
		"decision":  decision,
		"stored_at": chunk.Timestamp.Format(time.RFC3339),
	}, nil
}

// validateStoreDecisionParams validates required parameters for storing decisions
func (ms *MemoryServer) validateStoreDecisionParams(params map[string]interface{}) (decision, rationale, sessionID string, err error) {
	decision, ok := params["decision"].(string)
	if !ok || decision == "" {
		return "", "", "", errors.New("decision is required")
	}

	rationale, ok = params["rationale"].(string)
	if !ok || rationale == "" {
		return "", "", "", errors.New("rationale is required")
	}

	sessionID, ok = params["session_id"].(string)
	if !ok || sessionID == "" {
		return "", "", "", errors.New("session_id is required")
	}

	return decision, rationale, sessionID, nil
}

// buildDecisionContent creates formatted decision content
func (ms *MemoryServer) buildDecisionContent(decision, rationale string, params map[string]interface{}) string {
	content := "ARCHITECTURAL DECISION: " + decision + "\n\nRATIONALE: " + rationale

	if decisionContext, ok := params["context"].(string); ok && decisionContext != "" {
		content += "\n\nCONTEXT: " + decisionContext
	}

	return content
}

// buildDecisionMetadata creates metadata for decision chunks
func (ms *MemoryServer) buildDecisionMetadata(params map[string]interface{}, decision, rationale string) types.ChunkMetadata {
	metadata := types.ChunkMetadata{
		Outcome:    types.OutcomeSuccess,
		Difficulty: types.DifficultyModerate,
		Tags:       []string{"architecture", "decision"},
	}

	// Set repository
	if repo, ok := params["repository"].(string); ok {
		metadata.Repository = normalizeRepository(repo)
	} else {
		metadata.Repository = GlobalMemoryRepository
	}

	// Add extended metadata
	ms.addDecisionExtendedMetadata(&metadata, params, decision, rationale)

	return metadata
}

// addDecisionExtendedMetadata adds extended metadata with context detection
func (ms *MemoryServer) addDecisionExtendedMetadata(metadata *types.ChunkMetadata, params map[string]interface{}, decision, rationale string) {
	detector, err := contextpkg.NewDetector()
	if err != nil {
		return // Skip extended metadata if detector fails
	}

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

	// Mark as architectural decision
	metadata.ExtendedMetadata["decision_type"] = "architectural"
	metadata.ExtendedMetadata["decision_text"] = decision
	metadata.ExtendedMetadata["rationale_text"] = rationale
}

// createDecisionChunk creates and stores the decision chunk
func (ms *MemoryServer) createDecisionChunk(ctx context.Context, sessionID, content string, metadata *types.ChunkMetadata) (*types.ConversationChunk, error) {
	// Create repository-scoped session ID
	repositoryScopedSessionID := ms.createRepositoryScopedSessionID(metadata.Repository, sessionID)
	logging.Info("Created repository-scoped session for decision", "original_session", sessionID, "scoped_session", repositoryScopedSessionID, "repository", metadata.Repository)

	// Create chunk
	chunk, err := ms.container.GetChunkingService().CreateChunk(ctx, repositoryScopedSessionID, content, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create decision chunk: %w", err)
	}

	// Override type to architecture decision
	chunk.Type = types.ChunkTypeArchitectureDecision

	// Store chunk
	if err := ms.container.GetVectorStore().Store(ctx, chunk); err != nil {
		return nil, fmt.Errorf("failed to store decision: %w", err)
	}

	return chunk, nil
}

func (ms *MemoryServer) handleGetPatterns(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_get_patterns called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_get_patterns failed: missing repository parameter")
		return nil, errors.New("repository is required")
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
	// Add recovery to prevent crashes during health checks
	defer func() {
		if r := recover(); r != nil {
			logging.Error("Panic in handleHealth", "error", r)
		}
	}()

	logging.Info("MCP TOOL: memory_health called")

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  map[string]interface{}{},
	}

	// Check vector store with nil check
	logging.Info("Checking vector store health")
	if ms.container == nil || ms.container.GetVectorStore() == nil {
		logging.Error("Vector store is nil")
		health["services"].(map[string]interface{})["vector_store"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  "vector store not initialized",
		}
		health["status"] = StatusDegraded
	} else if err := ms.container.GetVectorStore().HealthCheck(ctx); err != nil {
		logging.Error("Vector store health check failed", "error", err)
		health["services"].(map[string]interface{})["vector_store"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = StatusDegraded
	} else {
		logging.Info("Vector store health check passed")
		health["services"].(map[string]interface{})["vector_store"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Check embedding service with nil check
	logging.Info("Checking embedding service health")
	if ms.container == nil || ms.container.GetEmbeddingService() == nil {
		logging.Error("Embedding service is nil")
		health["services"].(map[string]interface{})["embedding_service"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  "embedding service not initialized",
		}
		health["status"] = StatusDegraded
	} else if err := ms.container.GetEmbeddingService().HealthCheck(ctx); err != nil {
		logging.Error("Embedding service health check failed", "error", err)
		health["services"].(map[string]interface{})["embedding_service"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = StatusDegraded
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
		return ms.handleRecentResource(ctx, parts)
	case "patterns":
		return ms.handlePatternsResource(ctx, parts)
	case "decisions":
		return ms.handleDecisionsResource(ctx, parts)
	case GlobalRepository:
		return ms.handleGlobalResource(parts)
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// handleRecentResource handles recent chunks resource requests
func (ms *MemoryServer) handleRecentResource(ctx context.Context, parts []string) ([]protocol.Content, error) {
	if len(parts) < 4 {
		return nil, errors.New("repository required for recent resource")
	}
	repository := parts[3]
	chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 20, 0)
	if err != nil {
		return nil, err
	}
	chunksJSON, err := json.Marshal(chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chunks: %w", err)
	}
	return []protocol.Content{protocol.NewContent(string(chunksJSON))}, nil
}

// handlePatternsResource handles patterns resource requests
func (ms *MemoryServer) handlePatternsResource(ctx context.Context, parts []string) ([]protocol.Content, error) {
	if len(parts) < 4 {
		return nil, errors.New("repository required for patterns resource")
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
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patterns result: %w", err)
	}
	return []protocol.Content{protocol.NewContent(string(resultJSON))}, nil
}

// handleDecisionsResource handles architecture decisions resource requests
func (ms *MemoryServer) handleDecisionsResource(ctx context.Context, parts []string) ([]protocol.Content, error) {
	if len(parts) < 4 {
		return nil, errors.New("repository required for decisions resource")
	}
	repository := parts[3]

	// Search for architecture decisions
	memQuery := types.NewMemoryQuery("architectural decision")
	memQuery.Repository = &repository
	memQuery.Types = []types.ChunkType{types.ChunkTypeArchitectureDecision}
	memQuery.Limit = 50

	embeddings, err := ms.container.GetEmbeddingService().Generate(ctx, "architectural decision")
	if err != nil {
		return nil, err
	}

	results, err := ms.container.GetVectorStore().Search(ctx, memQuery, embeddings)
	if err != nil {
		return nil, err
	}

	decisions := ms.formatDecisionResults(results.Results)
	result := map[string]interface{}{
		"repository": repository,
		"decisions":  decisions,
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal decisions result: %w", err)
	}
	return []protocol.Content{protocol.NewContent(string(resultJSON))}, nil
}

// handleGlobalResource handles global insights resource requests
func (ms *MemoryServer) handleGlobalResource(parts []string) ([]protocol.Content, error) {
	if len(parts) < 4 || parts[3] != "insights" {
		return nil, errors.New("invalid global resource")
	}

	// Get global insights across all repositories
	// This is a simplified implementation
	result := map[string]interface{}{
		"message": "Global insights feature coming soon",
		"status":  "not_implemented",
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal global result: %w", err)
	}
	return []protocol.Content{protocol.NewContent(string(resultJSON))}, nil
}

// formatDecisionResults formats decision search results
func (ms *MemoryServer) formatDecisionResults(results []types.SearchResult) []map[string]interface{} {
	decisions := make([]map[string]interface{}, 0, len(results))
	for i := range results {
		result := &results[i]
		decisions = append(decisions, map[string]interface{}{
			"chunk_id":  result.Chunk.ID,
			"summary":   result.Chunk.Summary,
			"content":   result.Chunk.Content,
			"timestamp": result.Chunk.Timestamp,
		})
	}
	return decisions
}

// Helper methods

func (ms *MemoryServer) analyzePatterns(chunks []types.ConversationChunk) []string {
	tagCounts := make(map[string]int)
	patterns := []string{}

	for i := range chunks {
		chunk := &chunks[i]
		for _, tag := range chunk.Metadata.Tags {
			tagCounts[tag]++
		}
	}

	// Find common patterns (tags that appear multiple times)
	for tag, count := range tagCounts {
		if count >= 3 { // Threshold for pattern
			patterns = append(patterns, tag+" (appears "+strconv.Itoa(count)+" times)")
		}
	}

	return patterns
}

func (ms *MemoryServer) extractDecisions(chunks []types.ConversationChunk) []string {
	decisions := []string{}

	for i := range chunks {
		chunk := &chunks[i]
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

	for i := range chunks {
		chunk := &chunks[i]
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

	// Validate and extract parameters
	suggestConfig, err := ms.validateSuggestRelatedParams(params)
	if err != nil {
		return nil, err
	}

	logging.Info("Processing context suggestions", "context_length", len(suggestConfig.CurrentContext), "session_id", suggestConfig.SessionID)

	// Generate embedding and search for similar content
	results, err := ms.searchForRelatedContent(ctx, suggestConfig)
	if err != nil {
		return nil, err
	}

	// Process search results into suggestions
	suggestions := ms.processSearchResults(ctx, results, suggestConfig.MaxSuggestions)

	// Add pattern-based suggestions if enabled
	ms.addPatternSuggestions(suggestConfig, &suggestions)

	logging.Info("memory_suggest_related completed successfully", "total_suggestions", len(suggestions), "session_id", suggestConfig.SessionID)
	return map[string]interface{}{
		"suggestions":      suggestions,
		"total_found":      len(suggestions),
		"search_context":   suggestConfig.CurrentContext,
		"include_patterns": suggestConfig.IncludePatterns,
		"session_id":       suggestConfig.SessionID,
	}, nil
}

// suggestRelatedConfig holds configuration for related suggestions
type suggestRelatedConfig struct {
	CurrentContext  string
	SessionID       string
	Repository      string
	MaxSuggestions  int
	IncludePatterns bool
}

// validateSuggestRelatedParams validates and extracts parameters for suggest related
func (ms *MemoryServer) validateSuggestRelatedParams(params map[string]interface{}) (*suggestRelatedConfig, error) {
	currentContext, ok := params["current_context"].(string)
	if !ok {
		logging.Error("memory_suggest_related failed: missing current_context parameter")
		return nil, errors.New("current_context is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok {
		logging.Error("memory_suggest_related failed: missing session_id parameter")
		return nil, errors.New("session_id is required")
	}

	repository := ""
	if repo, exists := params["repository"].(string); exists {
		repository = repo
	}

	maxSuggestions := 5
	if maxVal, exists := params["max_suggestions"].(float64); exists {
		maxSuggestions = int(maxVal)
	}

	includePatterns := true
	if include, exists := params["include_patterns"].(bool); exists {
		includePatterns = include
	}

	return &suggestRelatedConfig{
		CurrentContext:  currentContext,
		SessionID:       sessionID,
		Repository:      repository,
		MaxSuggestions:  maxSuggestions,
		IncludePatterns: includePatterns,
	}, nil
}

// searchForRelatedContent generates embedding and searches for similar content
func (ms *MemoryServer) searchForRelatedContent(ctx context.Context, suggestConfig *suggestRelatedConfig) (*types.SearchResults, error) {
	// Generate embedding for current context
	logging.Info("Generating embedding for context suggestions", "context_length", len(suggestConfig.CurrentContext))
	embedding, err := ms.container.GetEmbeddingService().Generate(ctx, suggestConfig.CurrentContext)
	if err != nil {
		logging.Error("Failed to generate embedding for context suggestions", "error", err)
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	logging.Info("Embedding generated for context suggestions", "dimension", len(embedding))

	// Search for similar content
	logging.Info("Searching for related context", "repository", suggestConfig.Repository, "max_suggestions", suggestConfig.MaxSuggestions)
	query := types.NewMemoryQuery(suggestConfig.CurrentContext)
	query.Repository = &suggestConfig.Repository
	query.Limit = suggestConfig.MaxSuggestions * 2 // Get more to filter from
	query.MinRelevanceScore = 0.6

	results, err := ms.container.GetVectorStore().Search(ctx, query, embedding)
	if err != nil {
		logging.Error("Context suggestion search failed", "error", err)
		return nil, fmt.Errorf("search failed: %w", err)
	}
	logging.Info("Context suggestion search completed", "total_results", results.Total)

	return results, nil
}

// processSearchResults converts search results into suggestion format
func (ms *MemoryServer) processSearchResults(ctx context.Context, results *types.SearchResults, maxSuggestions int) []map[string]interface{} {
	suggestions := []map[string]interface{}{}
	analytics := ms.container.GetMemoryAnalytics()

	for i := range results.Results {
		if i >= maxSuggestions {
			break
		}
		result := &results.Results[i]

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

	return suggestions
}

// addPatternSuggestions adds pattern-based suggestions if enabled
func (ms *MemoryServer) addPatternSuggestions(suggestConfig *suggestRelatedConfig, suggestions *[]map[string]interface{}) {
	if !suggestConfig.IncludePatterns || ms.container.GetPatternAnalyzer() == nil {
		return
	}

	// Infer problem type from context
	problemType := ms.inferProblemType(suggestConfig.CurrentContext)

	// Get pattern recommendations
	currentTools := []string{} // Extract current tools from context (simplified approach)
	patternRecommendations := ms.container.GetPatternAnalyzer().GetPatternRecommendations(currentTools, problemType)

	// Convert pattern recommendations to suggestions
	for _, pattern := range patternRecommendations {
		if len(*suggestions) >= suggestConfig.MaxSuggestions {
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

		*suggestions = append(*suggestions, suggestion)
	}
}

// inferProblemType infers problem type from context content
func (ms *MemoryServer) inferProblemType(problemContext string) string {
	contextLower := strings.ToLower(problemContext)
	switch {
	case strings.Contains(contextLower, "error") || strings.Contains(contextLower, "bug"):
		return "debug"
	case strings.Contains(contextLower, "test"):
		return "test"
	case strings.Contains(contextLower, "build") || strings.Contains(contextLower, "compile"):
		return "build"
	case strings.Contains(contextLower, "config"):
		return "configuration"
	default:
		return "general"
	}
}

// handleAutoInsights generates automated insights about memory patterns and trends
func (ms *MemoryServer) handleAutoInsights(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_auto_insights called", "params", params)

	// Validate and extract parameters
	insightsConfig, err := ms.validateAutoInsightsParams(params)
	if err != nil {
		return nil, err
	}

	logging.Info("Generating auto insights", "repository", insightsConfig.Repository, "timeframe", insightsConfig.Timeframe, "session_id", insightsConfig.SessionID)

	// Get chunks for analysis
	chunks, err := ms.getChunksForInsights(ctx, insightsConfig.Repository, insightsConfig.Timeframe)
	if err != nil {
		return nil, err
	}

	// Build base insights
	insights := map[string]interface{}{
		"repository":     insightsConfig.Repository,
		"session_id":     insightsConfig.SessionID,
		"timeframe":      insightsConfig.Timeframe,
		"generated_at":   time.Now().Format(time.RFC3339),
		"total_memories": len(chunks),
	}

	// Analyze chunks and add distribution data
	ms.addDistributionInsights(chunks, insights)

	// Generate pattern insights if enabled
	if insightsConfig.IncludePatterns && ms.container.GetPatternAnalyzer() != nil {
		ms.addPatternInsights(chunks, insights)
	}

	// Generate trend insights if enabled
	if insightsConfig.IncludeTrends {
		ms.addTrendInsights(chunks, insights)
	}

	// Generate recommendations
	recommendations := ms.generateInsightRecommendations(chunks)
	insights["recommendations"] = recommendations

	logging.Info("Auto insights generated successfully", "insights_count", len(insights), "recommendations", len(recommendations))

	return insights, nil
}

// autoInsightsConfig holds configuration for auto insights generation
type autoInsightsConfig struct {
	Repository      string
	SessionID       string
	Timeframe       string
	IncludePatterns bool
	IncludeTrends   bool
}

// validateAutoInsightsParams validates and extracts parameters for auto insights
func (ms *MemoryServer) validateAutoInsightsParams(params map[string]interface{}) (*autoInsightsConfig, error) {
	repository, ok := params["repository"].(string)
	if !ok {
		logging.Error("memory_auto_insights failed: missing repository parameter")
		return nil, errors.New("repository is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok {
		logging.Error("memory_auto_insights failed: missing session_id parameter")
		return nil, errors.New("session_id is required")
	}

	timeframe := "week"
	if tf, exists := params["timeframe"].(string); exists {
		timeframe = tf
	}

	includePatterns := true
	if include, exists := params["include_patterns"].(bool); exists {
		includePatterns = include
	}

	includeTrends := true
	if include, exists := params["include_trends"].(bool); exists {
		includeTrends = include
	}

	return &autoInsightsConfig{
		Repository:      repository,
		SessionID:       sessionID,
		Timeframe:       timeframe,
		IncludePatterns: includePatterns,
		IncludeTrends:   includeTrends,
	}, nil
}

// getChunksForInsights retrieves chunks for insights analysis
func (ms *MemoryServer) getChunksForInsights(ctx context.Context, repository, timeframe string) ([]types.ConversationChunk, error) {
	limit := 50
	if timeframe == "month" {
		limit = 200
	}

	chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, limit, 0)
	if err != nil {
		logging.Error("Failed to retrieve chunks for insights", "error", err)
		return nil, fmt.Errorf("failed to retrieve memory data: %w", err)
	}

	return chunks, nil
}

// addDistributionInsights analyzes chunk distributions and adds to insights
func (ms *MemoryServer) addDistributionInsights(chunks []types.ConversationChunk, insights map[string]interface{}) {
	typeDistribution := make(map[string]int)
	outcomeDistribution := make(map[string]int)
	var recentActivity []map[string]interface{}
	effectiveness := make([]float64, 0, len(chunks))

	for i := range chunks {
		chunk := &chunks[i]
		typeDistribution[string(chunk.Type)]++
		outcomeDistribution[string(chunk.Metadata.Outcome)]++

		// Calculate effectiveness score
		score := ms.calculateEffectivenessScore(chunk)
		effectiveness = append(effectiveness, score)

		// Add to recent activity (last 10)
		if len(recentActivity) < 10 {
			recentActivity = append(recentActivity, map[string]interface{}{
				"summary":   chunk.Summary,
				"type":      chunk.Type,
				"outcome":   chunk.Metadata.Outcome,
				"timestamp": chunk.Timestamp.Format(time.RFC3339),
				"tags":      chunk.Metadata.Tags,
			})
		}
	}

	insights["type_distribution"] = typeDistribution
	insights["outcome_distribution"] = outcomeDistribution
	insights["recent_activity"] = recentActivity

	// Calculate average effectiveness
	if len(effectiveness) > 0 {
		sum := 0.0
		for _, e := range effectiveness {
			sum += e
		}
		insights["average_effectiveness"] = sum / float64(len(effectiveness))
	}
}

// calculateEffectivenessScore calculates effectiveness score for a chunk
func (ms *MemoryServer) calculateEffectivenessScore(chunk *types.ConversationChunk) float64 {
	score := 0.7 // Base score
	if chunk.Metadata.Outcome == StatusSuccess {
		score += 0.2
	}
	if len(chunk.Metadata.Tags) > 2 {
		score += 0.1
	}
	return score
}

// addPatternInsights generates pattern insights from chunks
func (ms *MemoryServer) addPatternInsights(chunks []types.ConversationChunk, insights map[string]interface{}) {
	patterns := []map[string]interface{}{}

	// Simple keyword frequency analysis
	keywordFreq := make(map[string]int)
	for i := range chunks {
		chunk := &chunks[i]
		words := strings.Fields(strings.ToLower(chunk.Content))
		for _, word := range words {
			if len(word) > 3 && !isStopWord(word) {
				keywordFreq[word]++
			}
		}
	}

	// Get top keywords
	topKeywords := ms.getTopKeywords(keywordFreq, 10)
	patterns = append(patterns, map[string]interface{}{
		"type":        "keyword_frequency",
		"description": "Most frequently mentioned terms",
		"data":        topKeywords,
	})

	insights["patterns"] = patterns
}

// getTopKeywords extracts top N keywords from frequency map
func (ms *MemoryServer) getTopKeywords(keywordFreq map[string]int, limit int) []map[string]interface{} {
	type keywordPair struct {
		word  string
		count int
	}

	var keywords []keywordPair
	for word, count := range keywordFreq {
		if count > 1 {
			keywords = append(keywords, keywordPair{word, count})
		}
	}

	// Sort by frequency
	sort.Slice(keywords, func(i, j int) bool {
		return keywords[i].count > keywords[j].count
	})

	// Take top N
	topKeywords := []map[string]interface{}{}
	for i, kw := range keywords {
		if i >= limit {
			break
		}
		topKeywords = append(topKeywords, map[string]interface{}{
			"keyword":   kw.word,
			"frequency": kw.count,
		})
	}

	return topKeywords
}

// addTrendInsights generates trend insights from chunks
func (ms *MemoryServer) addTrendInsights(chunks []types.ConversationChunk, insights map[string]interface{}) {
	trends := map[string]interface{}{
		"memory_creation_trend": "stable",
		"success_rate_trend":    "improving",
		"complexity_trend":      "increasing",
	}

	// Calculate outcome distribution for trend analysis
	outcomeDistribution := make(map[string]int)
	for i := range chunks {
		chunk := &chunks[i]
		outcomeDistribution[string(chunk.Metadata.Outcome)]++
	}

	if outcomeDistribution[StatusSuccess] > outcomeDistribution["failure"] {
		trends["overall_trend"] = "positive"
	} else {
		trends["overall_trend"] = "needs_attention"
	}

	insights["trends"] = trends
}

// generateInsightRecommendations generates recommendations based on chunk analysis
func (ms *MemoryServer) generateInsightRecommendations(chunks []types.ConversationChunk) []string {
	recommendations := []string{}

	// Calculate distributions for recommendations
	outcomeDistribution := make(map[string]int)
	typeDistribution := make(map[string]int)
	for i := range chunks {
		chunk := &chunks[i]
		outcomeDistribution[string(chunk.Metadata.Outcome)]++
		typeDistribution[string(chunk.Type)]++
	}

	if outcomeDistribution["failure"] > len(chunks)/3 {
		recommendations = append(recommendations, "Consider reviewing failed memories to identify improvement opportunities")
	}

	if typeDistribution["architecture_decision"] == 0 {
		recommendations = append(recommendations, "Consider documenting more architectural decisions for future reference")
	}

	if len(chunks) < 5 {
		recommendations = append(recommendations, "Increase memory storage to build more comprehensive project knowledge")
	}

	return recommendations
}

// handlePatternPrediction predicts future patterns based on historical memory data
func (ms *MemoryServer) handlePatternPrediction(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_pattern_prediction called", "params", params)

	// Validate required parameters
	patternContext, ok := params["context"].(string)
	if !ok {
		logging.Error("memory_pattern_prediction failed: missing context parameter")
		return nil, errors.New("context is required")
	}

	repository, ok := params["repository"].(string)
	if !ok {
		logging.Error("memory_pattern_prediction failed: missing repository parameter")
		return nil, errors.New("repository is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok {
		logging.Error("memory_pattern_prediction failed: missing session_id parameter")
		return nil, errors.New("session_id is required")
	}

	// Set optional parameters with defaults
	predictionType := "general"
	if pt, exists := params["prediction_type"].(string); exists {
		predictionType = pt
	}

	confidenceThreshold := 0.7
	if ct, exists := params["confidence_threshold"].(float64); exists {
		confidenceThreshold = ct
	}

	logging.Info("Generating pattern predictions", "context", patternContext, "repository", repository, "type", predictionType)

	// Get historical data for pattern analysis
	chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 100, 0)
	if err != nil {
		logging.Error("Failed to retrieve chunks for pattern prediction", "error", err)
		return nil, fmt.Errorf("failed to retrieve memory data: %w", err)
	}

	// Generate context embedding (optional)
	_, err = ms.container.GetEmbeddingService().Generate(ctx, patternContext)
	if err != nil {
		logging.Warn("Failed to generate context embedding, using text analysis", "error", err)
	}

	// Analyze historical patterns
	analysis := analyzeHistoricalPatterns(chunks, patternContext)

	// Generate predictions
	predictions := []map[string]interface{}{}

	// Add outcome prediction
	if outcomePred := predictOutcome(analysis, len(chunks), confidenceThreshold); outcomePred != nil {
		predictions = append(predictions, outcomePred)
	}

	// Add technology prediction
	if techPred := predictTechnology(analysis, len(chunks), confidenceThreshold); techPred != nil {
		predictions = append(predictions, techPred)
	}

	// Add complexity prediction (always included)
	predictions = append(predictions, predictComplexity(chunks))

	// Add timing prediction
	if timingPred := predictOptimalTiming(analysis, len(chunks), confidenceThreshold); timingPred != nil {
		predictions = append(predictions, timingPred)
	}

	// Build response
	prediction := map[string]interface{}{
		"context":                     patternContext,
		"repository":                  repository,
		"session_id":                  sessionID,
		"prediction_type":             predictionType,
		"confidence_threshold":        confidenceThreshold,
		"generated_at":                time.Now().Format(time.RFC3339),
		"total_data_points":           len(chunks),
		"predictions":                 predictions,
		"high_confidence_predictions": len(predictions),
		"recommendations":             generatePredictionRecommendations(predictions),
	}

	logging.Info("Pattern predictions generated", "predictions_count", len(predictions), "avg_confidence", confidenceThreshold)

	return prediction, nil
}

// patternAnalysisResult holds the results of analyzing historical patterns
type patternAnalysisResult struct {
	patterns     map[string]int
	outcomes     map[string]int
	technologies map[string]int
	timePatterns map[string]int
}

// findMostFrequent returns the key with the highest count in a map
func findMostFrequent(data map[string]int) (mostFrequent string, maxCount int) {
	for key, count := range data {
		if count > maxCount {
			maxCount = count
			mostFrequent = key
		}
	}
	return mostFrequent, maxCount
}

// analyzeHistoricalPatterns extracts patterns from historical memory chunks
func analyzeHistoricalPatterns(chunks []types.ConversationChunk, patternContext string) *patternAnalysisResult {
	result := &patternAnalysisResult{
		patterns:     make(map[string]int),
		outcomes:     make(map[string]int),
		technologies: make(map[string]int),
		timePatterns: make(map[string]int),
	}

	for i := range chunks {
		chunk := &chunks[i]
		// Extract patterns from content
		words := strings.Fields(strings.ToLower(chunk.Content))
		for _, word := range words {
			if len(word) > 3 && !isStopWord(word) && strings.Contains(strings.ToLower(patternContext), word) {
				result.patterns[word]++
			}
		}

		// Analyze outcomes
		result.outcomes[string(chunk.Metadata.Outcome)]++

		// Extract technologies from tags
		for _, tag := range chunk.Metadata.Tags {
			if strings.Contains(tag, "tech") || strings.Contains(tag, "lang") {
				result.technologies[tag]++
			}
		}

		// Time-based patterns (day of week)
		dayOfWeek := chunk.Timestamp.Weekday().String()
		result.timePatterns[dayOfWeek]++
	}

	return result
}

// predictOutcome generates outcome prediction based on historical outcomes
func predictOutcome(analysis *patternAnalysisResult, totalChunks int, confidenceThreshold float64) map[string]interface{} {
	mostLikelyOutcome, maxOutcomeCount := findMostFrequent(analysis.outcomes)
	if mostLikelyOutcome == "" {
		mostLikelyOutcome = StatusSuccess
	}

	outcomeConfidence := float64(maxOutcomeCount) / float64(totalChunks)
	if outcomeConfidence >= confidenceThreshold {
		return map[string]interface{}{
			"type":       "outcome",
			"prediction": mostLikelyOutcome,
			"confidence": outcomeConfidence,
			"basis":      fmt.Sprintf("Based on %d historical data points", maxOutcomeCount),
		}
	}
	return nil
}

// predictTechnology generates technology prediction based on historical technology usage
func predictTechnology(analysis *patternAnalysisResult, totalChunks int, confidenceThreshold float64) map[string]interface{} {
	if len(analysis.technologies) == 0 {
		return nil
	}

	mostLikelyTech, maxTechCount := findMostFrequent(analysis.technologies)
	techConfidence := float64(maxTechCount) / float64(totalChunks)

	if techConfidence >= confidenceThreshold {
		return map[string]interface{}{
			"type":       "technology",
			"prediction": mostLikelyTech,
			"confidence": techConfidence,
			"basis":      fmt.Sprintf("Technology appears in %d similar contexts", maxTechCount),
		}
	}
	return nil
}

// predictComplexity generates complexity prediction based on content metrics
func predictComplexity(chunks []types.ConversationChunk) map[string]interface{} {
	avgContentLength := 0
	avgTagCount := 0

	for i := range chunks {
		chunk := &chunks[i]
		avgContentLength += len(chunk.Content)
		avgTagCount += len(chunk.Metadata.Tags)
	}

	if len(chunks) > 0 {
		avgContentLength /= len(chunks)
		avgTagCount /= len(chunks)
	}

	complexityLevel := LevelMedium
	complexityConfidence := 0.6

	if avgContentLength > 500 || avgTagCount > 5 {
		complexityLevel = LevelHigh
		complexityConfidence = 0.8
	} else if avgContentLength < 200 && avgTagCount < 3 {
		complexityLevel = "low"
		complexityConfidence = 0.7
	}

	return map[string]interface{}{
		"type":       "complexity",
		"prediction": complexityLevel,
		"confidence": complexityConfidence,
		"basis":      fmt.Sprintf("Average content length: %d chars, tags: %d", avgContentLength, avgTagCount),
	}
}

// predictOptimalTiming generates timing prediction based on historical patterns
func predictOptimalTiming(analysis *patternAnalysisResult, totalChunks int, confidenceThreshold float64) map[string]interface{} {
	bestDay, maxDayCount := findMostFrequent(analysis.timePatterns)

	if maxDayCount <= 1 {
		return nil
	}

	dayConfidence := float64(maxDayCount) / float64(totalChunks)
	if dayConfidence >= confidenceThreshold {
		return map[string]interface{}{
			"type":       "timing",
			"prediction": "Best day for this type of work: " + bestDay,
			"confidence": dayConfidence,
			"basis":      fmt.Sprintf("%d similar activities occurred on %s", maxDayCount, bestDay),
		}
	}
	return nil
}

// generatePredictionRecommendations creates recommendations based on predictions
func generatePredictionRecommendations(predictions []map[string]interface{}) []string {
	recommendations := []string{}

	for _, pred := range predictions {
		if pred["confidence"].(float64) > 0.8 {
			recommendations = append(recommendations, fmt.Sprintf("High confidence: %s likely to be %s", pred["type"], pred["prediction"]))
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Need more historical data for reliable predictions")
	}

	return recommendations
}

// isStopWord checks if a word is a common stop word
func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true, "not": true,
		"you": true, "all": true, "can": true, "had": true, "her": true, "was": true,
		"one": true, "our": true, "out": true, "day": true, "get": true, "has": true,
		"him": true, "his": true, "how": true, "man": true, "new": true, "now": true,
		"old": true, "see": true, "two": true, "way": true, "who": true, "boy": true,
		"did": true, "its": true, "let": true, "put": true, "say": true, "she": true,
		"too": true, "use": true, "will": true, "with": true, "from": true, "this": true,
		"that": true, "have": true, "they": true, "would": true, "there": true, "been": true,
		"were": true, "said": true, "each": true, "which": true, "their": true, "time": true,
		"more": true, "very": true, "what": true, "know": true, "just": true, "first": true,
		"into": true, "over": true, "think": true, "also": true, "your": true, "work": true,
		"life": true, "only": true, "than": true, "other": true, "after": true, "back": true,
		"little": true, "where": true, "much": true, "before": true, "right": true, "through": true,
	}
	return stopWords[word]
}

// exportParams holds export parameters with validation
type exportParams struct {
	repository     string
	sessionID      string
	format         string
	includeVectors bool
	limit          int
	offset         int
}

// validateExportProjectParams validates and parses export parameters
func (ms *MemoryServer) validateExportProjectParams(params map[string]interface{}) (*exportParams, error) {
	repository, ok := params["repository"].(string)
	if !ok {
		logging.Error("memory_export_project failed: missing repository parameter")
		return nil, errors.New("repository is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok {
		return nil, errors.New("session_id is required")
	}

	format := "json"
	if f, exists := params["format"].(string); exists {
		format = f
	}

	includeVectors := false
	if include, exists := params["include_vectors"].(bool); exists {
		includeVectors = include
	}

	// Pagination support to handle large exports
	limit := 100 // Default page size to stay under token limits
	if l, exists := params["limit"].(float64); exists {
		limit = int(l)
		if limit > 500 { // Max limit to prevent token overflow
			limit = 500
		}
	}

	offset := 0
	if o, exists := params["offset"].(float64); exists {
		offset = int(o)
	}

	return &exportParams{
		repository:     repository,
		sessionID:      sessionID,
		format:         format,
		includeVectors: includeVectors,
		limit:          limit,
		offset:         offset,
	}, nil
}

// getRepositoryDataForExport retrieves repository chunks with pagination info
func (ms *MemoryServer) getRepositoryDataForExport(ctx context.Context, repository string, limit, offset int) ([]types.ConversationChunk, int, error) {
	chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve repository data: %w", err)
	}

	// Get total count for pagination info
	totalChunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 10000, 0)
	totalCount := len(totalChunks)
	if err != nil {
		totalCount = -1 // Unknown if we can't fetch
	}

	return chunks, totalCount, nil
}

// createPaginationInfo creates pagination metadata
func (ms *MemoryServer) createPaginationInfo(limit, offset, chunksInPage, totalCount int) map[string]interface{} {
	return map[string]interface{}{
		"limit":       limit,
		"offset":      offset,
		"has_more":    offset+chunksInPage < totalCount,
		"next_offset": offset + chunksInPage,
	}
}

// exportToJSON exports chunks as JSON format
func (ms *MemoryServer) exportToJSON(chunks []types.ConversationChunk, exportParams *exportParams, totalCount int) (interface{}, error) {
	exportData := map[string]interface{}{
		"repository":      exportParams.repository,
		"export_date":     time.Now().Format(time.RFC3339),
		"total_chunks":    totalCount,
		"chunks_in_page":  len(chunks),
		"include_vectors": exportParams.includeVectors,
		"pagination":      ms.createPaginationInfo(exportParams.limit, exportParams.offset, len(chunks), totalCount),
		"chunks":          chunks,
	}

	// Remove vector data if not requested
	if !exportParams.includeVectors {
		for i := range chunks {
			chunks[i].Embeddings = nil
		}
	}

	exportJSON, err := json.Marshal(exportData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal export data: %w", err)
	}

	return map[string]interface{}{
		"format":       "json",
		"data":         string(exportJSON),
		"size_bytes":   len(exportJSON),
		"chunks":       len(chunks),
		"total_chunks": totalCount,
		"repository":   exportParams.repository,
		"session_id":   exportParams.sessionID,
		"pagination":   ms.createPaginationInfo(exportParams.limit, exportParams.offset, len(chunks), totalCount),
	}, nil
}

// exportToMarkdown exports chunks as Markdown format
func (ms *MemoryServer) exportToMarkdown(chunks []types.ConversationChunk, exportParams *exportParams, totalCount int) (interface{}, error) {
	var markdown strings.Builder
	markdown.WriteString(fmt.Sprintf("# Memory Export: %s\n\n", exportParams.repository))
	markdown.WriteString(fmt.Sprintf("**Export Date:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
	markdown.WriteString(fmt.Sprintf("**Total Chunks:** %d\n", totalCount))
	markdown.WriteString(fmt.Sprintf("**Chunks in Page:** %d (offset: %d, limit: %d)\n\n", len(chunks), exportParams.offset, exportParams.limit))

	for i := range chunks {
		chunk := &chunks[i]
		markdown.WriteString("## " + chunk.Summary + "\n\n")
		markdown.WriteString(fmt.Sprintf("**ID:** %s\n", chunk.ID))
		markdown.WriteString(fmt.Sprintf("**Type:** %s\n", chunk.Type))
		markdown.WriteString(fmt.Sprintf("**Timestamp:** %s\n\n", chunk.Timestamp.Format("2006-01-02 15:04:05")))
		markdown.WriteString(chunk.Content + "\n\n")

		if len(chunk.Metadata.Tags) > 0 {
			markdown.WriteString(fmt.Sprintf("**Tags:** %s\n\n", strings.Join(chunk.Metadata.Tags, ", ")))
		}

		markdown.WriteString("---\n\n")
	}

	markdownData := markdown.String()
	return map[string]interface{}{
		"format":       "markdown",
		"data":         markdownData,
		"size_bytes":   len(markdownData),
		"chunks":       len(chunks),
		"total_chunks": totalCount,
		"repository":   exportParams.repository,
		"session_id":   exportParams.sessionID,
		"pagination":   ms.createPaginationInfo(exportParams.limit, exportParams.offset, len(chunks), totalCount),
	}, nil
}

// exportToArchive exports chunks as compressed archive format
func (ms *MemoryServer) exportToArchive(chunks []types.ConversationChunk, exportParams *exportParams) (interface{}, error) {
	// Use backup manager to create compressed archive
	if ms.container.GetBackupManager() == nil {
		return nil, errors.New("backup manager not available")
	}

	// Create a filtered backup for this repository only
	backupData := map[string]interface{}{
		"repository":  exportParams.repository,
		"export_date": time.Now().Format(time.RFC3339),
		"chunks":      chunks,
		"metadata": map[string]interface{}{
			"export_type": "project_export",
			"session_id":  exportParams.sessionID,
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
		"repository": exportParams.repository,
		"session_id": exportParams.sessionID,
		"encoding":   "base64",
	}, nil
}

// handleExportProject exports all memory data for a project
func (ms *MemoryServer) handleExportProject(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_export_project called", "params", params)

	// Validate and parse parameters
	exportParams, err := ms.validateExportProjectParams(params)
	if err != nil {
		return nil, err
	}

	// Get repository data with pagination
	chunks, totalCount, err := ms.getRepositoryDataForExport(ctx, exportParams.repository, exportParams.limit, exportParams.offset)
	if err != nil {
		return nil, err
	}

	// Export in the requested format
	switch exportParams.format {
	case "json":
		return ms.exportToJSON(chunks, exportParams, totalCount)
	case "markdown":
		return ms.exportToMarkdown(chunks, exportParams, totalCount)
	case "archive":
		return ms.exportToArchive(chunks, exportParams)
	default:
		return nil, fmt.Errorf("unsupported format: %s", exportParams.format)
	}
}

// handleImportContext imports conversation context from external source
func (ms *MemoryServer) handleImportContext(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	importParams, err := ms.parseImportParams(params)
	if err != nil {
		return nil, err
	}

	importedChunks, err := ms.importChunksBySource(ctx, importParams)
	if err != nil {
		return nil, err
	}

	storedCount := ms.storeImportedChunks(ctx, importedChunks, importParams)

	return ms.buildImportResponse(importParams, importedChunks, storedCount), nil
}

// importParams holds parsed import parameters
type importParams struct {
	source           string
	data             string
	repository       string
	sessionID        string
	chunkingStrategy string
	metadata         map[string]interface{}
}

// parseImportParams extracts and validates import parameters
func (ms *MemoryServer) parseImportParams(params map[string]interface{}) (*importParams, error) {
	source, ok := params["source"].(string)
	if !ok {
		return nil, errors.New("source is required")
	}

	data, ok := params["data"].(string)
	if !ok {
		return nil, errors.New("data is required")
	}

	repository, ok := params["repository"].(string)
	if !ok {
		return nil, errors.New("repository is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok {
		return nil, errors.New("session_id is required")
	}

	chunkingStrategy := "auto"
	if strategy, exists := params["chunking_strategy"].(string); exists {
		chunkingStrategy = strategy
	}

	metadata := map[string]interface{}{}
	if meta, exists := params["metadata"].(map[string]interface{}); exists {
		metadata = meta
	}

	return &importParams{
		source:           source,
		data:             data,
		repository:       repository,
		sessionID:        sessionID,
		chunkingStrategy: chunkingStrategy,
		metadata:         metadata,
	}, nil
}

// importChunksBySource imports chunks based on source type
func (ms *MemoryServer) importChunksBySource(ctx context.Context, params *importParams) ([]types.ConversationChunk, error) {
	switch params.source {
	case types.SourceConversation:
		return ms.importConversationText(ctx, params.data, params.repository, params.chunkingStrategy, params.metadata)
	case "file":
		return ms.importFileContent(ctx, params.data, params.repository, params.chunkingStrategy, params.metadata)
	case "archive":
		return ms.importArchiveData(ctx, params.data, params.repository, params.metadata)
	default:
		return nil, fmt.Errorf("unsupported source type: %s", params.source)
	}
}

// storeImportedChunks stores imported chunks with embeddings
func (ms *MemoryServer) storeImportedChunks(ctx context.Context, chunks []types.ConversationChunk, params *importParams) int {
	repositoryScopedSessionID := ms.createRepositoryScopedSessionID(params.repository, params.sessionID)
	logging.Info("Created repository-scoped session for import", "original_session", params.sessionID, "scoped_session", repositoryScopedSessionID, "repository", params.repository)

	storedCount := 0
	for i := range chunks {
		chunk := &chunks[i]
		chunk.SessionID = repositoryScopedSessionID

		if err := ms.processAndStoreChunk(ctx, chunk); err != nil {
			log.Printf("Failed to process chunk %s: %v", chunk.ID, err)
			continue
		}

		storedCount++
	}

	return storedCount
}

// processAndStoreChunk generates embedding and stores a single chunk
func (ms *MemoryServer) processAndStoreChunk(ctx context.Context, chunk *types.ConversationChunk) error {
	embedding, err := ms.container.GetEmbeddingService().Generate(ctx, chunk.Content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	chunk.Embeddings = embedding

	if err := ms.container.GetVectorStore().Store(ctx, chunk); err != nil {
		return fmt.Errorf("failed to store chunk: %w", err)
	}

	return nil
}

// buildImportResponse creates the import response structure
func (ms *MemoryServer) buildImportResponse(params *importParams, chunks []types.ConversationChunk, storedCount int) map[string]interface{} {
	return map[string]interface{}{
		"source":            params.source,
		"repository":        params.repository,
		"chunks_processed":  len(chunks),
		"chunks_stored":     storedCount,
		"chunking_strategy": params.chunkingStrategy,
		"session_id":        params.sessionID,
		"import_date":       time.Now().Format(time.RFC3339),
	}
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

	chunkData, err := ms.container.GetChunkingService().CreateChunk(ctx, "import", data, &chunkMetadata)
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

	chunkData, err := ms.container.GetChunkingService().CreateChunk(ctx, "import", data, &chunkMetadata)
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
		return nil, errors.New("no chunks found in archive")
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

// Relationship management handlers

// handleMemoryLink creates a relationship between two memory chunks
func (ms *MemoryServer) handleMemoryLink(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	sourceChunkID, ok := params["source_chunk_id"].(string)
	if !ok || sourceChunkID == "" {
		return nil, errors.New("source_chunk_id is required")
	}

	targetChunkID, ok := params["target_chunk_id"].(string)
	if !ok || targetChunkID == "" {
		return nil, errors.New("target_chunk_id is required")
	}

	relationTypeStr, ok := params["relation_type"].(string)
	if !ok || relationTypeStr == "" {
		return nil, errors.New("relation_type is required")
	}

	relationType := types.RelationType(relationTypeStr)
	if !relationType.Valid() {
		validTypes := make([]string, 0, len(types.AllValidRelationTypes()))
		for _, vt := range types.AllValidRelationTypes() {
			validTypes = append(validTypes, string(vt))
		}
		return nil, fmt.Errorf("invalid relation type: %s. Valid types are: %v", relationTypeStr, validTypes)
	}

	confidence := 0.8 // default
	if c, ok := params["confidence"].(float64); ok {
		confidence = c
	}

	// Get storage from container
	vectorStore := ms.container.VectorStore

	// Create the relationship
	relationship, err := vectorStore.StoreRelationship(ctx, sourceChunkID, targetChunkID, relationType, confidence, types.ConfidenceExplicit)
	if err != nil {
		return nil, fmt.Errorf("failed to create relationship: %w", err)
	}

	// Log the action
	ms.container.AuditLogger.LogEvent(ctx, audit.EventTypeRelationshipAdd, "memory_link", "relationship", relationship.ID, map[string]interface{}{
		"source_chunk_id": sourceChunkID,
		"target_chunk_id": targetChunkID,
		"relation_type":   relationTypeStr,
		"confidence":      confidence,
		"relationship_id": relationship.ID,
	})

	return map[string]interface{}{
		"relationship_id":   relationship.ID,
		"source_chunk_id":   relationship.SourceChunkID,
		"target_chunk_id":   relationship.TargetChunkID,
		"relation_type":     string(relationship.RelationType),
		"confidence":        relationship.Confidence,
		"confidence_source": string(relationship.ConfidenceSource),
		"created_at":        relationship.CreatedAt.Format(time.RFC3339),
		"message":           "Relationship created successfully",
	}, nil
}

// handleGetRelationships retrieves relationships for a memory chunk
func (ms *MemoryServer) handleGetRelationships(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	chunkID, err := ms.validateChunkID(params)
	if err != nil {
		return nil, err
	}

	query := ms.buildRelationshipQuery(params, chunkID)
	rels, err := ms.container.VectorStore.GetRelationships(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships: %w", err)
	}

	return ms.formatRelationshipResponse(rels, chunkID, query), nil
}

// validateChunkID extracts and validates chunk ID from parameters
func (ms *MemoryServer) validateChunkID(params map[string]interface{}) (string, error) {
	chunkID, ok := params["chunk_id"].(string)
	if !ok || chunkID == "" {
		return "", errors.New("chunk_id is required")
	}
	return chunkID, nil
}

// buildRelationshipQuery constructs relationship query from parameters
func (ms *MemoryServer) buildRelationshipQuery(params map[string]interface{}, chunkID string) *types.RelationshipQuery {
	query := types.NewRelationshipQuery(chunkID)

	ms.setQueryDirection(params, query)
	ms.setQueryLimits(params, query)
	ms.setQueryOptions(params, query)
	ms.setQueryRelationTypes(params, query)

	return query
}

// setQueryDirection sets the query direction if provided
func (ms *MemoryServer) setQueryDirection(params map[string]interface{}, query *types.RelationshipQuery) {
	if direction, ok := params["direction"].(string); ok {
		query.Direction = direction
	}
}

// setQueryLimits sets numeric limits for the query
func (ms *MemoryServer) setQueryLimits(params map[string]interface{}, query *types.RelationshipQuery) {
	if minConfidence, ok := params["min_confidence"].(float64); ok {
		query.MinConfidence = minConfidence
	}
	if maxDepth, ok := params["max_depth"].(float64); ok {
		query.MaxDepth = int(maxDepth)
	}
	if limit, ok := params["limit"].(float64); ok {
		query.Limit = int(limit)
	}
}

// setQueryOptions sets boolean options for the query
func (ms *MemoryServer) setQueryOptions(params map[string]interface{}, query *types.RelationshipQuery) {
	if includeChunks, ok := params["include_chunks"].(bool); ok {
		query.IncludeChunks = includeChunks
	}
}

// setQueryRelationTypes sets relation types filter if provided
func (ms *MemoryServer) setQueryRelationTypes(params map[string]interface{}, query *types.RelationshipQuery) {
	relationTypesRaw, ok := params["relation_types"].([]interface{})
	if !ok {
		return
	}

	relationTypes := make([]types.RelationType, 0, len(relationTypesRaw))
	for _, rt := range relationTypesRaw {
		if rtStr, ok := rt.(string); ok {
			relationTypes = append(relationTypes, types.RelationType(rtStr))
		}
	}
	query.RelationTypes = relationTypes
}

// formatRelationshipResponse formats the relationship query results
func (ms *MemoryServer) formatRelationshipResponse(rels []types.RelationshipResult, chunkID string, query *types.RelationshipQuery) map[string]interface{} {
	result := map[string]interface{}{
		"chunk_id":      chunkID,
		"relationships": make([]map[string]interface{}, len(rels)),
		"total_count":   len(rels),
		"query":         query,
	}

	for i := range rels {
		result["relationships"].([]map[string]interface{})[i] = ms.formatSingleRelationship(&rels[i], query.IncludeChunks)
	}

	return result
}

// formatSingleRelationship formats a single relationship with optional chunk info
func (ms *MemoryServer) formatSingleRelationship(rel *types.RelationshipResult, includeChunks bool) map[string]interface{} {
	relData := map[string]interface{}{
		"relationship_id":   rel.Relationship.ID,
		"source_chunk_id":   rel.Relationship.SourceChunkID,
		"target_chunk_id":   rel.Relationship.TargetChunkID,
		"relation_type":     string(rel.Relationship.RelationType),
		"confidence":        rel.Relationship.Confidence,
		"confidence_source": string(rel.Relationship.ConfidenceSource),
		"created_at":        rel.Relationship.CreatedAt.Format(time.RFC3339),
		"validation_count":  rel.Relationship.ValidationCount,
	}

	if rel.Relationship.LastValidated != nil {
		relData["last_validated"] = rel.Relationship.LastValidated.Format(time.RFC3339)
	}

	if includeChunks {
		ms.addChunkDetails(relData, rel)
	}

	return relData
}

// addChunkDetails adds source and target chunk details to relationship data
func (ms *MemoryServer) addChunkDetails(relData map[string]interface{}, rel *types.RelationshipResult) {
	if rel.SourceChunk != nil {
		relData["source_chunk"] = map[string]interface{}{
			"id":      rel.SourceChunk.ID,
			"type":    string(rel.SourceChunk.Type),
			"summary": rel.SourceChunk.Summary,
		}
	}
	if rel.TargetChunk != nil {
		relData["target_chunk"] = map[string]interface{}{
			"id":      rel.TargetChunk.ID,
			"type":    string(rel.TargetChunk.Type),
			"summary": rel.TargetChunk.Summary,
		}
	}
}

// handleTraverseGraph traverses the knowledge graph from a starting chunk
func (ms *MemoryServer) handleTraverseGraph(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	startChunkID, ok := params["start_chunk_id"].(string)
	if !ok || startChunkID == "" {
		return nil, errors.New("start_chunk_id is required")
	}

	maxDepth := 3 // default
	if md, ok := params["max_depth"].(float64); ok {
		maxDepth = int(md)
	}

	minConfidence := 0.5 // default
	if mc, ok := params["min_confidence"].(float64); ok {
		minConfidence = mc
	}

	var relationTypes []types.RelationType
	if relationTypesRaw, ok := params["relation_types"].([]interface{}); ok {
		relationTypes = make([]types.RelationType, 0, len(relationTypesRaw))
		for _, rt := range relationTypesRaw {
			if rtStr, ok := rt.(string); ok {
				relationTypes = append(relationTypes, types.RelationType(rtStr))
			}
		}
	}

	// Get storage from container
	vectorStore := ms.container.VectorStore

	// Perform graph traversal
	traversalResult, err := vectorStore.TraverseGraph(ctx, startChunkID, maxDepth, relationTypes)
	if err != nil {
		return nil, fmt.Errorf("failed to traverse graph: %w", err)
	}

	// Format response
	result := map[string]interface{}{
		"start_chunk_id": startChunkID,
		"max_depth":      maxDepth,
		"min_confidence": minConfidence,
		"paths":          make([]map[string]interface{}, len(traversalResult.Paths)),
		"nodes":          make([]map[string]interface{}, len(traversalResult.Nodes)),
		"edges":          make([]map[string]interface{}, len(traversalResult.Edges)),
		"summary": map[string]interface{}{
			"total_paths": len(traversalResult.Paths),
			"total_nodes": len(traversalResult.Nodes),
			"total_edges": len(traversalResult.Edges),
		},
	}

	// Format paths
	for i, path := range traversalResult.Paths {
		result["paths"].([]map[string]interface{})[i] = map[string]interface{}{
			"chunk_ids": path.ChunkIDs,
			"score":     path.Score,
			"depth":     path.Depth,
			"path_type": path.PathType,
		}
	}

	// Format nodes
	for i, node := range traversalResult.Nodes {
		nodeData := map[string]interface{}{
			"chunk_id":   node.ChunkID,
			"degree":     node.Degree,
			"centrality": node.Centrality,
		}
		if node.Chunk != nil {
			nodeData["chunk"] = map[string]interface{}{
				"id":      node.Chunk.ID,
				"type":    string(node.Chunk.Type),
				"summary": node.Chunk.Summary,
			}
		}
		result["nodes"].([]map[string]interface{})[i] = nodeData
	}

	// Format edges
	for i := range traversalResult.Edges {
		edge := &traversalResult.Edges[i]
		result["edges"].([]map[string]interface{})[i] = map[string]interface{}{
			"relationship_id": edge.Relationship.ID,
			"source_chunk_id": edge.Relationship.SourceChunkID,
			"target_chunk_id": edge.Relationship.TargetChunkID,
			"relation_type":   string(edge.Relationship.RelationType),
			"confidence":      edge.Relationship.Confidence,
			"weight":          edge.Weight,
		}
	}

	return result, nil
}

// handleAutoDetectRelationships automatically detects relationships for a chunk
func (ms *MemoryServer) handleAutoDetectRelationships(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract and validate parameters
	chunkID, sessionID, err := ms.validateRelationshipDetectionParams(params)
	if err != nil {
		return nil, err
	}

	// Build detection configuration
	detectionConfig := ms.buildRelationshipDetectionConfig(params)

	// Perform relationship detection
	detectionResult, err := ms.performRelationshipDetection(ctx, chunkID, detectionConfig, params)
	if err != nil {
		return nil, err
	}

	// Format and return response
	result := ms.formatRelationshipDetectionResponse(chunkID, sessionID, detectionResult, params)
	ms.logRelationshipDetection(ctx, chunkID, sessionID, detectionResult, params)

	return result, nil
}

// validateRelationshipDetectionParams validates parameters for relationship detection
func (ms *MemoryServer) validateRelationshipDetectionParams(params map[string]interface{}) (chunkID, sessionID string, err error) {
	chunkID, ok := params["chunk_id"].(string)
	if !ok || chunkID == "" {
		return "", "", errors.New("chunk_id is required")
	}

	sessionID, ok = params["session_id"].(string)
	if !ok || sessionID == "" {
		return "", "", errors.New("session_id is required")
	}

	return chunkID, sessionID, nil
}

// buildRelationshipDetectionConfig creates detection configuration from parameters
func (ms *MemoryServer) buildRelationshipDetectionConfig(params map[string]interface{}) *relationships.DetectionConfig {
	minConfidence := 0.6 // default
	if mc, ok := params["min_confidence"].(float64); ok {
		minConfidence = mc
	}

	var enabledDetectors []string
	if detectorsRaw, ok := params["enabled_detectors"].([]interface{}); ok {
		enabledDetectors = make([]string, 0, len(detectorsRaw))
		for _, d := range detectorsRaw {
			if dStr, ok := d.(string); ok {
				enabledDetectors = append(enabledDetectors, dStr)
			}
		}
	} else {
		enabledDetectors = []string{"temporal", "causal", "reference", "problem_solution"}
	}

	return &relationships.DetectionConfig{
		MinConfidence:               minConfidence,
		MaxTimeDistance:             24 * time.Hour,
		SemanticSimilarityThreshold: 0.7,
		EnabledDetectors:            enabledDetectors,
		RelationshipConfidence: map[types.RelationType]float64{
			types.RelationLedTo:      0.7,
			types.RelationSolvedBy:   0.8,
			types.RelationDependsOn:  0.6,
			types.RelationFollowsUp:  0.7,
			types.RelationRelatedTo:  0.5,
			types.RelationReferences: 0.8,
		},
	}
}

// performRelationshipDetection executes the relationship detection process
func (ms *MemoryServer) performRelationshipDetection(ctx context.Context, chunkID string, detectionConfig *relationships.DetectionConfig, params map[string]interface{}) (*relationships.DetectionResult, error) {
	vectorStore := ms.container.VectorStore

	// Get the chunk to analyze
	chunk, err := vectorStore.GetByID(ctx, chunkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	// Create relationship detector
	detector := relationships.NewRelationshipDetector(vectorStore)

	// Determine if we should auto-store
	autoStore := true // default
	if as, ok := params["auto_store"].(bool); ok {
		autoStore = as
	}

	// Detect relationships
	if autoStore {
		err = detector.AutoDetectAndStore(ctx, chunk, detectionConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-detect and store relationships: %w", err)
		}
		// For stored results, we need to get the detection result separately
		return detector.DetectRelationships(ctx, chunk, detectionConfig)
	}

	return detector.DetectRelationships(ctx, chunk, detectionConfig)
}

// formatRelationshipDetectionResponse formats the detection result
func (ms *MemoryServer) formatRelationshipDetectionResponse(chunkID, sessionID string, detectionResult *relationships.DetectionResult, params map[string]interface{}) map[string]interface{} {
	autoStore := true
	if as, ok := params["auto_store"].(bool); ok {
		autoStore = as
	}

	result := map[string]interface{}{
		"chunk_id":               chunkID,
		"session_id":             sessionID,
		"auto_stored":            autoStore,
		"processing_time_ms":     detectionResult.ProcessingTime.Milliseconds(),
		"relationships_detected": len(detectionResult.RelationshipsDetected),
		"detection_methods":      detectionResult.DetectionMethods,
		"relationships":          make([]map[string]interface{}, len(detectionResult.RelationshipsDetected)),
	}

	for i := range detectionResult.RelationshipsDetected {
		rel := &detectionResult.RelationshipsDetected[i]
		result["relationships"].([]map[string]interface{})[i] = map[string]interface{}{
			"source_chunk_id":   rel.SourceChunkID,
			"target_chunk_id":   rel.TargetChunkID,
			"relation_type":     string(rel.RelationType),
			"confidence":        rel.Confidence,
			"confidence_source": string(rel.ConfidenceSource),
		}
	}

	return result
}

// logRelationshipDetection logs the relationship detection operation
func (ms *MemoryServer) logRelationshipDetection(ctx context.Context, chunkID, sessionID string, detectionResult *relationships.DetectionResult, params map[string]interface{}) {
	autoStore := true
	if as, ok := params["auto_store"].(bool); ok {
		autoStore = as
	}

	ms.container.AuditLogger.LogEvent(ctx, audit.EventTypeRelationshipAdd, "auto_detect_relationships", "chunk", chunkID, map[string]interface{}{
		"session_id":             sessionID,
		"relationships_detected": len(detectionResult.RelationshipsDetected),
		"auto_stored":            autoStore,
	})
}

// handleUpdateRelationship updates an existing relationship
func (ms *MemoryServer) handleUpdateRelationship(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Extract parameters
	relationshipID, ok := params["relationship_id"].(string)
	if !ok || relationshipID == "" {
		return nil, errors.New("relationship_id is required")
	}

	// Get storage from container
	vectorStore := ms.container.VectorStore

	// Build confidence factors
	factors := types.ConfidenceFactors{}

	if userCertainty, ok := params["user_certainty"].(float64); ok {
		factors.UserCertainty = &userCertainty
	}

	var newConfidence float64
	var hasConfidence bool
	if confidence, ok := params["confidence"].(float64); ok {
		newConfidence = confidence
		hasConfidence = true
	}

	// If no new confidence provided, we need to get the current relationship
	if !hasConfidence {
		relationship, err := vectorStore.GetRelationshipByID(ctx, relationshipID)
		if err != nil {
			return nil, fmt.Errorf("failed to get relationship: %w", err)
		}
		newConfidence = relationship.Confidence
	}

	// Update the relationship
	err := vectorStore.UpdateRelationship(ctx, relationshipID, newConfidence, factors)
	if err != nil {
		return nil, fmt.Errorf("failed to update relationship: %w", err)
	}

	// Get the updated relationship
	relationship, err := vectorStore.GetRelationshipByID(ctx, relationshipID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated relationship: %w", err)
	}

	// Log the action
	logData := map[string]interface{}{
		"new_confidence": newConfidence,
	}
	if validationNote, ok := params["validation_note"].(string); ok {
		logData["validation_note"] = validationNote
	}
	ms.container.AuditLogger.LogEvent(ctx, audit.EventTypeMemoryUpdate, "update_relationship", "relationship", relationshipID, logData)

	// Format response
	result := map[string]interface{}{
		"relationship_id":   relationship.ID,
		"source_chunk_id":   relationship.SourceChunkID,
		"target_chunk_id":   relationship.TargetChunkID,
		"relation_type":     string(relationship.RelationType),
		"confidence":        relationship.Confidence,
		"confidence_source": string(relationship.ConfidenceSource),
		"validation_count":  relationship.ValidationCount,
		"created_at":        relationship.CreatedAt.Format(time.RFC3339),
		"message":           "Relationship updated successfully",
	}

	if relationship.LastValidated != nil {
		result["last_validated"] = relationship.LastValidated.Format(time.RFC3339)
	}

	return result, nil
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// buildMetadataFromParams extracts metadata from request parameters
func (ms *MemoryServer) buildMetadataFromParams(params map[string]interface{}) types.ChunkMetadata {
	metadata := types.ChunkMetadata{
		Outcome:    types.OutcomeInProgress, // Default, will be auto-updated based on content
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
	detector, err := contextpkg.NewDetector()
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
		return nil, errors.New("repository is required")
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

	// Get chunks based on repository and timeframe - ENABLED
	var chunks []types.ConversationChunk
	var err error

	if repository == "" || repository == GlobalMemoryRepository {
		// Global analysis across all repositories
		chunks, err = ms.getChunksForTimeframe(ctx, "", timeframe)
	} else {
		chunks, err = ms.getChunksForTimeframe(ctx, repository, timeframe)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get chunks for conflict analysis: %w", err)
	}

	// Use the enhanced conflict detection system - ENABLED (basic intelligence module only)
	baseDetector := intelligence.NewConflictDetector()
	baseResolver := intelligence.NewConflictResolver()

	// Initialize Qdrant-enhanced components if available - DISABLED
	// These components are disabled due to incomplete implementation
	var qdrantDetector interface{} = nil // placeholder for future Qdrant enhancement
	var qdrantResolver interface{} = nil // placeholder for future Qdrant enhancement
	_ = qdrantDetector                   // suppress unused variable warning
	_ = qdrantResolver                   // suppress unused variable warning

	// For now, use basic conflict detection only
	// TODO: Enable Qdrant enhancement when the API compatibility issues are resolved
	logging.Info("Using basic conflict detection and resolution", "repository", repository)

	// Detect conflicts using the basic system
	var conflictResult *intelligence.ConflictDetectionResult

	if qdrantDetector != nil {
		// Qdrant enhancement not available yet - fall back to base detection
		logging.Info("Qdrant enhancement not available, using base conflict detection")
		conflictResult, err = baseDetector.DetectConflicts(ctx, chunks)
	} else {
		// Use base conflict detection
		conflictResult, err = baseDetector.DetectConflicts(ctx, chunks)
	}

	if err != nil {
		logging.Error("Advanced conflict detection failed, falling back to simple detection", "error", err)
		// Fallback to simple detection
		conflicts := ms.detectConflictsSimple(chunks)
		contradictions := ms.findContradictoryDecisions(chunks)
		patternInconsistencies := ms.findPatternInconsistencies(chunks)

		result := map[string]interface{}{
			"repository":              repository,
			"timeframe":               timeframe,
			"analysis_timestamp":      time.Now().Format(time.RFC3339),
			"total_chunks_analyzed":   len(chunks),
			"conflicts":               conflicts,
			"contradictory_decisions": contradictions,
			"pattern_inconsistencies": patternInconsistencies,
			"summary": map[string]interface{}{
				"total_conflicts":    len(conflicts) + len(contradictions) + len(patternInconsistencies),
				"severity_breakdown": ms.categorizeConflictsBySeverity(conflicts, contradictions, patternInconsistencies),
				"recommendations":    ms.generateConflictResolutionRecommendations(conflicts, contradictions),
			},
		}
		return result, nil
	}

	// Generate resolution recommendations using basic resolver
	var recommendations []intelligence.ResolutionRecommendation
	if qdrantResolver != nil && len(conflictResult.Conflicts) > 0 {
		// Qdrant enhancement not available yet - use base resolver
		logging.Info("Qdrant enhancement not available, using base conflict resolver")
		recommendations, err = baseResolver.ResolveConflicts(ctx, conflictResult.Conflicts)
	} else {
		recommendations, err = baseResolver.ResolveConflicts(ctx, conflictResult.Conflicts)
	}

	if err != nil {
		logging.Warn("Failed to generate conflict resolution recommendations", "error", err)
		recommendations = []intelligence.ResolutionRecommendation{}
	}

	// Convert conflicts to legacy format for compatibility
	legacyConflicts := ms.convertConflictsToLegacyFormat(conflictResult.Conflicts)

	// Build enhanced result with basic conflict analysis
	result := map[string]interface{}{
		"repository":            repository,
		"timeframe":             timeframe,
		"analysis_timestamp":    time.Now().Format(time.RFC3339),
		"total_chunks_analyzed": len(chunks),
		"processing_time":       conflictResult.ProcessingTime,

		// Enhanced conflict data
		"conflicts_v2": map[string]interface{}{
			"total_found":     conflictResult.ConflictsFound,
			"conflicts":       conflictResult.Conflicts,
			"recommendations": recommendations,
			"analysis_time":   conflictResult.AnalysisTime,
			"vector_enhanced": false, // Basic implementation only
		},

		// Legacy format for backward compatibility
		"conflicts":               legacyConflicts["outcome_conflicts"],
		"contradictory_decisions": legacyConflicts["architectural_conflicts"],
		"pattern_inconsistencies": legacyConflicts["pattern_conflicts"],

		"summary": map[string]interface{}{
			"total_conflicts":         conflictResult.ConflictsFound,
			"conflicts_by_type":       ms.categorizeConflictsByType(conflictResult.Conflicts),
			"conflicts_by_severity":   ms.categorizeConflictsBySeverity2(conflictResult.Conflicts),
			"resolution_strategies":   len(recommendations),
			"high_priority_conflicts": ms.countHighPriorityConflicts(conflictResult.Conflicts),
			"recommendations_summary": ms.summarizeRecommendations(recommendations),
			"analysis_method":         ms.getAnalysisMethod(false), // Basic implementation
		},
	}

	// Vector-specific analysis not available in basic implementation
	// TODO: Add vector analysis when Qdrant enhancement is enabled

	logging.Info("memory_conflicts completed", "repository", repository, "total_conflicts", conflictResult.ConflictsFound)
	return result, nil
}

// getAnalysisMethod returns a description of the analysis method used
func (ms *MemoryServer) getAnalysisMethod(vectorEnhanced bool) string {
	if vectorEnhanced {
		return "Qdrant-enhanced vector semantic analysis with conflict detection"
	}
	return "Standard pattern-based conflict detection"
}

// handleMemoryContinuity shows what was left incomplete from previous sessions
func (ms *MemoryServer) handleMemoryContinuity(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_continuity called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_continuity failed: missing repository parameter")
		return nil, errors.New("repository is required")
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
		// Create repository-scoped session ID for filtering
		repositoryScopedSessionID := ms.createRepositoryScopedSessionID(repository, sessionID)
		logging.Info("Filtering by repository-scoped session", "original_session", sessionID, "scoped_session", repositoryScopedSessionID, "repository", repository)
		targetChunks = ms.filterChunksBySession(chunks, repositoryScopedSessionID)
	} else if len(chunks) > 0 {
		// findMostRecentSessionID now returns repository-scoped session ID
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

	for i := range chunks {
		chunk := &chunks[i]
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

	for i := range chunks {
		chunk := &chunks[i]
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
	for i := range chunks {
		chunk := &chunks[i]
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

	for i := range chunks {
		chunk := &chunks[i]
		switch chunk.Type {
		case types.ChunkTypeArchitectureDecision:
			hasArch = true
		case types.ChunkTypeProblem, types.ChunkTypeSolution:
			hasProblems = true
		case types.ChunkTypeCodeChange:
			hasCode = true
		case types.ChunkTypeDiscussion, types.ChunkTypeSessionSummary, types.ChunkTypeAnalysis, types.ChunkTypeVerification, types.ChunkTypeQuestion:
			// Other chunk types, no special handling needed
		// Task-oriented chunk types
		case types.ChunkTypeTask:
			// Tasks represent both problems and potentially solutions
			hasProblems = true
		case types.ChunkTypeTaskUpdate, types.ChunkTypeTaskProgress:
			// Task updates contribute to problem-solving coverage
			hasProblems = true
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

	for i := range chunks {
		chunk := &chunks[i]
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

	for i := range chunks {
		chunk := &chunks[i]
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
	chunkTypes := make(map[types.ChunkType]bool)
	for i := range chunks {
		chunk := &chunks[i]
		chunkTypes[chunk.Type] = true
	}

	// Maximum variety is having all 6 main types
	maxTypes := 6.0
	return float64(len(chunkTypes)) / maxTypes
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
		results, err := ms.container.GetVectorStore().Search(ctx, query, embeddings)
		if err == nil {
			for i := range results.Results {
				chunks = append(chunks, results.Results[i].Chunk)
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

		for i := range chunks {
			if cutoff.IsZero() || chunks[i].Timestamp.After(cutoff) {
				filteredChunks = append(filteredChunks, chunks[i])
			}
		}
		chunks = filteredChunks
	}

	return chunks, nil
}

// detectConflictsSimple finds conflicting information in chunks (legacy method)
func (ms *MemoryServer) detectConflictsSimple(chunks []types.ConversationChunk) []map[string]interface{} {
	conflicts := []map[string]interface{}{}

	// Simple conflict detection based on contradictory outcomes for similar content
	for i := range chunks {
		for j := range chunks {
			if i >= j || chunks[i].SessionID == chunks[j].SessionID {
				continue
			}

			// Check for similar summaries with different outcomes
			if ms.areSimilarSummaries(chunks[i].Summary, chunks[j].Summary) &&
				chunks[i].Metadata.Outcome != chunks[j].Metadata.Outcome &&
				(chunks[i].Metadata.Outcome == types.OutcomeSuccess || chunks[i].Metadata.Outcome == types.OutcomeFailed) &&
				(chunks[j].Metadata.Outcome == types.OutcomeSuccess || chunks[j].Metadata.Outcome == types.OutcomeFailed) {
				conflicts = append(conflicts, map[string]interface{}{
					"type":        "outcome_conflict",
					"description": "Similar issues with different outcomes",
					"chunk1": map[string]interface{}{
						"id":        chunks[i].ID,
						"summary":   chunks[i].Summary,
						"outcome":   string(chunks[i].Metadata.Outcome),
						"timestamp": chunks[i].Timestamp.Format(time.RFC3339),
					},
					"chunk2": map[string]interface{}{
						"id":        chunks[j].ID,
						"summary":   chunks[j].Summary,
						"outcome":   string(chunks[j].Metadata.Outcome),
						"timestamp": chunks[j].Timestamp.Format(time.RFC3339),
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
	for i := range chunks {
		if chunks[i].Type == types.ChunkTypeArchitectureDecision {
			decisions = append(decisions, chunks[i])
		}
	}

	// Look for decisions that might contradict each other
	for i := range decisions {
		for j := range decisions {
			if i >= j {
				continue
			}

			// Simple keyword-based contradiction detection
			if ms.hasContradictoryKeywords(decisions[i].Content, decisions[j].Content) {
				contradictions = append(contradictions, map[string]interface{}{
					"type":        "architecture_contradiction",
					"description": "Potentially contradictory architectural decisions",
					"decision1": map[string]interface{}{
						"id":        decisions[i].ID,
						"summary":   decisions[i].Summary,
						"timestamp": decisions[i].Timestamp.Format(time.RFC3339),
					},
					"decision2": map[string]interface{}{
						"id":        decisions[j].ID,
						"summary":   decisions[j].Summary,
						"timestamp": decisions[j].Timestamp.Format(time.RFC3339),
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
		recommendations = append(recommendations, "Reconcile contradictory architectural decisions", "Create a unified architecture document")
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
	for i := range sortedChunks {
		stages = append(stages, string(sortedChunks[i].Type))
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
	for i := range sessionChunks {
		if sessionChunks[i].Type == types.ChunkTypeProblem {
			recentProblems = append(recentProblems, sessionChunks[i])
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
	for i := range sessionChunks {
		if sessionChunks[i].Type == types.ChunkTypeCodeChange {
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
	for i := range chunks {
		if strings.Contains(strings.ToLower(chunks[i].Content), "next") ||
			strings.Contains(strings.ToLower(chunks[i].Content), "todo") ||
			strings.Contains(strings.ToLower(chunks[i].Content), "continue") {
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

	// Validate and extract chunk IDs
	chunkIDs, err := ms.validateAndExtractChunkIDs(params)
	if err != nil {
		return nil, err
	}

	// Parse and validate thread type
	threadType, err := ms.parseThreadType(params)
	if err != nil {
		return nil, err
	}

	// Retrieve and validate chunks
	chunks, err := ms.retrieveChunksByIDs(ctx, chunkIDs)
	if err != nil {
		return nil, err
	}

	// Create thread and apply metadata overrides
	thread, err := ms.createThreadWithMetadata(ctx, chunks, threadType, params)
	if err != nil {
		return nil, err
	}

	// Build and return response
	result := ms.buildCreateThreadResponse(thread)
	logging.Info("memory_create_thread completed successfully", "thread_id", thread.ID, "chunk_count", len(chunks))
	return result, nil
}

// validateAndExtractChunkIDs validates and extracts chunk IDs from parameters
func (ms *MemoryServer) validateAndExtractChunkIDs(params map[string]interface{}) ([]string, error) {
	chunkIDsInterface, ok := params["chunk_ids"].([]interface{})
	if !ok || len(chunkIDsInterface) == 0 {
		logging.Error("memory_create_thread failed: missing or empty chunk_ids parameter")
		return nil, errors.New("chunk_ids is required and must not be empty")
	}

	// Convert interface{} slice to string slice
	chunkIDs := make([]string, len(chunkIDsInterface))
	for i, id := range chunkIDsInterface {
		if idStr, ok := id.(string); ok {
			chunkIDs[i] = idStr
		} else {
			return nil, errors.New("chunk_ids must be an array of strings")
		}
	}

	return chunkIDs, nil
}

// parseThreadType parses and validates thread type from parameters
func (ms *MemoryServer) parseThreadType(params map[string]interface{}) (threading.ThreadType, error) {
	// Get thread type
	threadTypeStr := types.SourceConversation // default
	if tt, ok := params["thread_type"].(string); ok && tt != "" {
		threadTypeStr = tt
	}

	// Parse thread type
	switch threadTypeStr {
	case types.SourceConversation:
		return threading.ThreadTypeConversation, nil
	case "problem_solving":
		return threading.ThreadTypeProblemSolving, nil
	case "feature":
		return threading.ThreadTypeFeature, nil
	case "debugging":
		return threading.ThreadTypeDebugging, nil
	case "architecture":
		return threading.ThreadTypeArchitecture, nil
	case "workflow":
		return threading.ThreadTypeWorkflow, nil
	default:
		return "", fmt.Errorf("invalid thread_type: %s", threadTypeStr)
	}
}

// retrieveChunksByIDs retrieves and validates chunks by their IDs
func (ms *MemoryServer) retrieveChunksByIDs(ctx context.Context, chunkIDs []string) ([]types.ConversationChunk, error) {
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
		return nil, errors.New("no valid chunks found for the provided chunk_ids")
	}

	return chunks, nil
}

// createThreadWithMetadata creates thread and applies metadata overrides
func (ms *MemoryServer) createThreadWithMetadata(ctx context.Context, chunks []types.ConversationChunk, threadType threading.ThreadType, params map[string]interface{}) (*threading.MemoryThread, error) {
	// Create thread using ThreadManager
	threadManager := ms.container.GetThreadManager()
	thread, err := threadManager.CreateThread(ctx, chunks, threadType)
	if err != nil {
		logging.Error("Failed to create thread", "error", err)
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	// Apply metadata overrides
	ms.applyThreadMetadataOverrides(ctx, thread, params)

	return thread, nil
}

// applyThreadMetadataOverrides applies title and repository overrides to thread
func (ms *MemoryServer) applyThreadMetadataOverrides(ctx context.Context, thread *threading.MemoryThread, params map[string]interface{}) {
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
}

// buildCreateThreadResponse builds the response for thread creation
func (ms *MemoryServer) buildCreateThreadResponse(thread *threading.MemoryThread) map[string]interface{} {
	return map[string]interface{}{
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
}

// handleGetThreads retrieves memory threads with optional filtering
func (ms *MemoryServer) handleGetThreads(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_get_threads called", "params", params)

	// Build filters and get configuration
	filters := ms.buildThreadFilters(params)
	includeSummary := ms.shouldIncludeSummary(params)

	// Get and format threads
	threads, err := ms.getThreadsFromStore(ctx, filters)
	if err != nil {
		return nil, err
	}

	threadList := ms.formatThreadList(ctx, threads, includeSummary)

	result := map[string]interface{}{
		"threads":     threadList,
		"total_count": len(threads),
		"filters":     filters,
	}

	logging.Info("memory_get_threads completed successfully", "thread_count", len(threads))
	return result, nil
}

// buildThreadFilters creates thread filters from request parameters
func (ms *MemoryServer) buildThreadFilters(params map[string]interface{}) threading.ThreadFilters {
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

	ms.addSessionIDFilter(&filters, params)

	return filters
}

// addSessionIDFilter adds session ID filter with repository scoping
func (ms *MemoryServer) addSessionIDFilter(filters *threading.ThreadFilters, params map[string]interface{}) {
	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		return
	}

	// Create repository-scoped session ID if repository is specified
	if filters.Repository != nil {
		repositoryScopedSessionID := ms.createRepositoryScopedSessionID(*filters.Repository, sessionID)
		logging.Info("Using repository-scoped session for thread filtering", "original_session", sessionID, "scoped_session", repositoryScopedSessionID, "repository", *filters.Repository)
		filters.SessionID = &repositoryScopedSessionID
	} else {
		// If no repository filter, use the sessionID as-is (less secure but backwards compatible)
		logging.Warn("Thread filtering by sessionID without repository filter - potential cross-tenant access", "session_id", sessionID)
		filters.SessionID = &sessionID
	}
}

// shouldIncludeSummary determines if thread summaries should be included
func (ms *MemoryServer) shouldIncludeSummary(params map[string]interface{}) bool {
	includeSummary := false
	if inc, ok := params["include_summary"].(bool); ok {
		includeSummary = inc
	}
	return includeSummary
}

// getThreadsFromStore retrieves threads from the store
func (ms *MemoryServer) getThreadsFromStore(ctx context.Context, filters threading.ThreadFilters) ([]*threading.MemoryThread, error) {
	threadStore := ms.container.GetThreadStore()
	threads, err := threadStore.ListThreads(ctx, filters)
	if err != nil {
		logging.Error("Failed to list threads", "error", err)
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}
	return threads, nil
}

// formatThreadList formats threads for the response
func (ms *MemoryServer) formatThreadList(ctx context.Context, threads []*threading.MemoryThread, includeSummary bool) []map[string]interface{} {
	threadList := make([]map[string]interface{}, 0, len(threads))

	for _, thread := range threads {
		threadInfo := ms.formatSingleThread(thread)

		if includeSummary {
			ms.addThreadSummary(ctx, threadInfo, thread)
		}

		threadList = append(threadList, threadInfo)
	}

	return threadList
}

// formatSingleThread formats basic thread information
func (ms *MemoryServer) formatSingleThread(thread *threading.MemoryThread) map[string]interface{} {
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

	return threadInfo
}

// addThreadSummary adds summary information to thread data
func (ms *MemoryServer) addThreadSummary(ctx context.Context, threadInfo map[string]interface{}, thread *threading.MemoryThread) {
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

// handleDetectThreads automatically detects and creates memory threads from existing chunks
func (ms *MemoryServer) handleDetectThreads(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_detect_threads called", "params", params)

	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_detect_threads failed: missing repository parameter")
		return nil, errors.New("repository is required")
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
// validateUpdateThreadParams validates thread update parameters
func (ms *MemoryServer) validateUpdateThreadParams(params map[string]interface{}) (string, error) {
	threadID, ok := params["thread_id"].(string)
	if !ok || threadID == "" {
		logging.Error("memory_update_thread failed: missing thread_id parameter")
		return "", errors.New("thread_id is required")
	}
	return threadID, nil
}

// updateThreadStatus updates thread status if provided
func (ms *MemoryServer) updateThreadStatus(ctx context.Context, params map[string]interface{}, thread *threading.MemoryThread, threadStore threading.ThreadStore) (bool, error) {
	status, ok := params["status"].(string)
	if !ok || status == "" {
		return false, nil
	}

	newStatus := threading.ThreadStatus(status)
	err := threadStore.UpdateThreadStatus(ctx, thread.ID, newStatus)
	if err != nil {
		return false, fmt.Errorf("failed to update thread status: %w", err)
	}

	thread.Status = newStatus
	return true, nil
}

// updateThreadTitle updates thread title if provided
func (ms *MemoryServer) updateThreadTitle(params map[string]interface{}, thread *threading.MemoryThread) bool {
	title, ok := params["title"].(string)
	if !ok || title == "" {
		return false
	}

	thread.Title = title
	return true
}

// addChunksToThreadFromParams adds chunks to thread if provided
func (ms *MemoryServer) addChunksToThreadFromParams(params map[string]interface{}, thread *threading.MemoryThread) bool {
	addChunksInterface, ok := params["add_chunks"].([]interface{})
	if !ok || len(addChunksInterface) == 0 {
		return false
	}

	return ms.addChunksToThread(thread, addChunksInterface)
}

// removeChunksFromThread removes chunks from thread if provided
func (ms *MemoryServer) removeChunksFromThread(params map[string]interface{}, thread *threading.MemoryThread) bool {
	removeChunksInterface, ok := params["remove_chunks"].([]interface{})
	if !ok || len(removeChunksInterface) == 0 {
		return false
	}

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
		return true
	}

	return false
}

// storeUpdatedThread stores the updated thread if changes were made
func (ms *MemoryServer) storeUpdatedThread(ctx context.Context, thread *threading.MemoryThread, threadStore threading.ThreadStore, updated bool) error {
	if !updated {
		return nil
	}

	thread.LastUpdate = time.Now()
	err := threadStore.StoreThread(ctx, thread)
	if err != nil {
		return fmt.Errorf("failed to store updated thread: %w", err)
	}

	return nil
}

// buildThreadUpdateResult builds the response for thread update
func (ms *MemoryServer) buildThreadUpdateResult(thread *threading.MemoryThread, updated bool) map[string]interface{} {
	return map[string]interface{}{
		"thread_id":   thread.ID,
		"title":       thread.Title,
		"status":      string(thread.Status),
		"chunk_count": len(thread.ChunkIDs),
		"last_update": thread.LastUpdate.Format(time.RFC3339),
		"updated":     updated,
	}
}

func (ms *MemoryServer) handleUpdateThread(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_update_thread called", "params", params)

	// Validate parameters
	threadID, err := ms.validateUpdateThreadParams(params)
	if err != nil {
		return nil, err
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
	if statusUpdated, err := ms.updateThreadStatus(ctx, params, thread, threadStore); err != nil {
		return nil, err
	} else if statusUpdated {
		updated = true
	}

	// Update title if provided
	if ms.updateThreadTitle(params, thread) {
		updated = true
	}

	// Add chunks if provided
	if ms.addChunksToThreadFromParams(params, thread) {
		updated = true
	}

	// Remove chunks if provided
	if ms.removeChunksFromThread(params, thread) {
		updated = true
	}

	// Store updated thread if changes were made
	err = ms.storeUpdatedThread(ctx, thread, threadStore, updated)
	if err != nil {
		return nil, err
	}

	// Build and return result
	result := ms.buildThreadUpdateResult(thread, updated)

	logging.Info("memory_update_thread completed successfully", "thread_id", threadID, "updated", updated)
	return result, nil
}

// Cross-Project Pattern Detection Handlers

// validateCrossRepoAnalysisParams validates required parameters for cross-repo analysis
func (ms *MemoryServer) validateCrossRepoAnalysisParams(params map[string]interface{}) (string, error) {
	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_analyze_cross_repo_patterns failed: missing session_id parameter")
		return "", errors.New("session_id is required")
	}
	return sessionID, nil
}

// parseCrossRepoAnalysisParams parses optional parameters for cross-repo analysis
func (ms *MemoryServer) parseCrossRepoAnalysisParams(params map[string]interface{}) (repositories, includePatterns, excludePatterns []string, maxResults int) {
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

	return repositories, techStacks, patternTypes, minFrequency
}

// getRepositoriesForAnalysis gets repositories to analyze (discovered if not specified)
func (ms *MemoryServer) getRepositoriesForAnalysis(ctx context.Context, repositories []string) []string {
	if len(repositories) == 0 {
		return ms.discoverRepositories(ctx)
	}
	return repositories
}

// updateRepositoryContexts updates contexts for all repositories with recent data
func (ms *MemoryServer) updateRepositoryContexts(ctx context.Context, repositories []string, multiRepoEngine *intelligence.MultiRepoEngine) {
	for _, repo := range repositories {
		ms.updateSingleRepositoryContext(ctx, repo, multiRepoEngine)
	}
}

// updateSingleRepositoryContext updates context for a single repository
func (ms *MemoryServer) updateSingleRepositoryContext(ctx context.Context, repo string, multiRepoEngine *intelligence.MultiRepoEngine) {
	chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repo, 50, 0)
	if err != nil || len(chunks) == 0 {
		return
	}

	// Filter for recent chunks (last 7 days for RecencyRecent)
	cutoff := time.Now().AddDate(0, 0, -7)
	var recentChunks []types.ConversationChunk
	for i := range chunks {
		if chunks[i].Timestamp.After(cutoff) {
			recentChunks = append(recentChunks, chunks[i])
		}
	}

	if len(recentChunks) == 0 {
		return
	}

	// Update repository context
	err = multiRepoEngine.UpdateRepositoryContext(ctx, repo, recentChunks)
	if err != nil {
		logging.Warn("Failed to update repository context", "repository", repo, "error", err)
	}
}

// executeCrossRepoAnalysis executes the analysis and returns insights
func (ms *MemoryServer) executeCrossRepoAnalysis(ctx context.Context, multiRepoEngine *intelligence.MultiRepoEngine) (interface{}, error) {
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

	return insights, nil
}

func (ms *MemoryServer) handleAnalyzeCrossRepoPatterns(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_analyze_cross_repo_patterns called", "params", params)

	// Validate required parameters
	sessionID, err := ms.validateCrossRepoAnalysisParams(params)
	if err != nil {
		return nil, err
	}

	multiRepoEngine := ms.container.GetMultiRepoEngine()

	// Parse optional parameters
	repositories, techStacks, patternTypes, minFrequency := ms.parseCrossRepoAnalysisParams(params)

	// Get repositories to analyze (discover if not specified)
	repositories = ms.getRepositoriesForAnalysis(ctx, repositories)

	// Update repository contexts with recent data
	ms.updateRepositoryContexts(ctx, repositories, multiRepoEngine)

	// Execute cross-repository analysis
	insights, err := ms.executeCrossRepoAnalysis(ctx, multiRepoEngine)
	if err != nil {
		return nil, err
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
		return nil, errors.New("repository is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_find_similar_repositories failed: missing session_id parameter")
		return nil, errors.New("session_id is required")
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
	chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 50, 0)
	if err == nil && len(chunks) > 0 {
		// Filter for recent chunks (last 7 days for RecencyRecent)
		cutoff := time.Now().AddDate(0, 0, -7)
		var recentChunks []types.ConversationChunk
		for i := range chunks {
			if chunks[i].Timestamp.After(cutoff) {
				recentChunks = append(recentChunks, chunks[i])
			}
		}
		chunks = recentChunks

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
		return nil, errors.New("session_id is required")
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

	// Validate required parameters
	multiSearchConfig, err := ms.validateMultiRepoSearchParams(params)
	if err != nil {
		return nil, err
	}

	// Extract optional parameters
	ms.extractMultiRepoSearchParams(multiSearchConfig, params)

	// Execute multi-repository search
	results, err := ms.executeMultiRepoSearch(ctx, multiSearchConfig)
	if err != nil {
		return nil, err
	}

	// Convert results to response format
	searchResults := ms.formatMultiRepoResults(results)

	// Build and return response
	return ms.buildMultiRepoSearchResponse(multiSearchConfig, searchResults), nil
}

// multiRepoSearchConfig holds configuration for multi-repository search
type multiRepoSearchConfig struct {
	Query          string
	SessionID      string
	Repositories   []string
	TechStacks     []string
	Frameworks     []string
	PatternTypes   []string
	MinConfidence  float64
	MaxResults     int
	IncludeSimilar bool
}

// validateMultiRepoSearchParams validates required parameters for multi-repo search
func (ms *MemoryServer) validateMultiRepoSearchParams(params map[string]interface{}) (*multiRepoSearchConfig, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		logging.Error("memory_search_multi_repo failed: missing query parameter")
		return nil, errors.New("query is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_search_multi_repo failed: missing session_id parameter")
		return nil, errors.New("session_id is required")
	}

	return &multiRepoSearchConfig{
		Query:          query,
		SessionID:      sessionID,
		MinConfidence:  0.5,
		MaxResults:     10,
		IncludeSimilar: true,
	}, nil
}

// extractMultiRepoSearchParams extracts optional parameters for multi-repo search
func (ms *MemoryServer) extractMultiRepoSearchParams(searchConfig *multiRepoSearchConfig, params map[string]interface{}) {
	ms.extractRepositoriesParam(searchConfig, params)
	ms.extractTechStacksParam(searchConfig, params)
	ms.extractFrameworksParam(searchConfig, params)
	ms.extractPatternTypesParam(searchConfig, params)
	ms.extractNumericParams(searchConfig, params)
}

// extractRepositoriesParam extracts repositories from parameters
func (ms *MemoryServer) extractRepositoriesParam(searchConfig *multiRepoSearchConfig, params map[string]interface{}) {
	if reposInterface, ok := params["repositories"].([]interface{}); ok {
		for _, repoInterface := range reposInterface {
			if repo, ok := repoInterface.(string); ok {
				searchConfig.Repositories = append(searchConfig.Repositories, repo)
			}
		}
	}
}

// extractTechStacksParam extracts tech stacks from parameters
func (ms *MemoryServer) extractTechStacksParam(searchConfig *multiRepoSearchConfig, params map[string]interface{}) {
	if techInterface, ok := params["tech_stacks"].([]interface{}); ok {
		for _, techInterface := range techInterface {
			if tech, ok := techInterface.(string); ok {
				searchConfig.TechStacks = append(searchConfig.TechStacks, tech)
			}
		}
	}
}

// extractFrameworksParam extracts frameworks from parameters
func (ms *MemoryServer) extractFrameworksParam(searchConfig *multiRepoSearchConfig, params map[string]interface{}) {
	if frameworkInterface, ok := params["frameworks"].([]interface{}); ok {
		for _, fwInterface := range frameworkInterface {
			if fw, ok := fwInterface.(string); ok {
				searchConfig.Frameworks = append(searchConfig.Frameworks, fw)
			}
		}
	}
}

// extractPatternTypesParam extracts pattern types from parameters
func (ms *MemoryServer) extractPatternTypesParam(searchConfig *multiRepoSearchConfig, params map[string]interface{}) {
	if typesInterface, ok := params["pattern_types"].([]interface{}); ok {
		for _, typeInterface := range typesInterface {
			if patternType, ok := typeInterface.(string); ok {
				searchConfig.PatternTypes = append(searchConfig.PatternTypes, patternType)
			}
		}
	}
}

// extractNumericParams extracts numeric configuration parameters
func (ms *MemoryServer) extractNumericParams(searchConfig *multiRepoSearchConfig, params map[string]interface{}) {
	if conf, ok := params["min_confidence"].(float64); ok {
		searchConfig.MinConfidence = conf
	}

	if maxVal, ok := params["max_results"].(float64); ok {
		searchConfig.MaxResults = int(maxVal)
	}

	if include, ok := params["include_similar"].(bool); ok {
		searchConfig.IncludeSimilar = include
	}
}

// executeMultiRepoSearch creates and executes multi-repository search
func (ms *MemoryServer) executeMultiRepoSearch(ctx context.Context, searchConfig *multiRepoSearchConfig) ([]intelligence.MultiRepoResult, error) {
	multiRepoEngine := ms.container.GetMultiRepoEngine()

	// Create multi-repository query
	multiQuery := intelligence.MultiRepoQuery{
		Query:          searchConfig.Query,
		Repositories:   searchConfig.Repositories,
		TechStacks:     searchConfig.TechStacks,
		Frameworks:     searchConfig.Frameworks,
		MinConfidence:  searchConfig.MinConfidence,
		MaxResults:     searchConfig.MaxResults,
		IncludeSimilar: searchConfig.IncludeSimilar,
	}

	// Convert pattern type strings to PatternType
	for _, pt := range searchConfig.PatternTypes {
		multiQuery.PatternTypes = append(multiQuery.PatternTypes, intelligence.PatternType(pt))
	}

	// Execute multi-repository search
	results, err := multiRepoEngine.QueryMultiRepo(ctx, &multiQuery)
	if err != nil {
		logging.Error("Failed to search multi-repo", "query", searchConfig.Query, "error", err)
		return nil, fmt.Errorf("failed to search multi-repo: %w", err)
	}

	return results, nil
}

// formatMultiRepoResults converts search results to response format
func (ms *MemoryServer) formatMultiRepoResults(results []intelligence.MultiRepoResult) []map[string]interface{} {
	searchResults := make([]map[string]interface{}, 0, len(results))
	for _, result := range results {
		patterns := ms.formatPatternResults(result.Patterns)
		searchResults = append(searchResults, map[string]interface{}{
			"repository": result.Repository,
			"relevance":  result.Relevance,
			"patterns":   patterns,
			"context":    result.Context,
		})
	}
	return searchResults
}

// formatPatternResults formats pattern results for response
func (ms *MemoryServer) formatPatternResults(patterns []intelligence.CrossRepoPattern) []map[string]interface{} {
	formattedPatterns := make([]map[string]interface{}, 0, len(patterns))
	for i := range patterns {
		formattedPatterns = append(formattedPatterns, map[string]interface{}{
			"id":           patterns[i].ID,
			"name":         patterns[i].Name,
			"description":  patterns[i].Description,
			"repositories": patterns[i].Repositories,
			"frequency":    patterns[i].Frequency,
			"success_rate": patterns[i].SuccessRate,
			"confidence":   patterns[i].Confidence,
			"keywords":     patterns[i].Keywords,
			"tech_stacks":  patterns[i].TechStacks,
			"frameworks":   patterns[i].Frameworks,
			"pattern_type": string(patterns[i].PatternType),
			"created_at":   patterns[i].CreatedAt.Format(time.RFC3339),
			"last_used":    patterns[i].LastUsed.Format(time.RFC3339),
		})
	}
	return formattedPatterns
}

// buildMultiRepoSearchResponse builds the final response for multi-repo search
func (ms *MemoryServer) buildMultiRepoSearchResponse(searchConfig *multiRepoSearchConfig, searchResults []map[string]interface{}) map[string]interface{} {
	response := map[string]interface{}{
		"query":      searchConfig.Query,
		"session_id": searchConfig.SessionID,
		"filter_criteria": map[string]interface{}{
			"repositories":    searchConfig.Repositories,
			"tech_stacks":     searchConfig.TechStacks,
			"frameworks":      searchConfig.Frameworks,
			"pattern_types":   searchConfig.PatternTypes,
			"min_confidence":  searchConfig.MinConfidence,
			"max_results":     searchConfig.MaxResults,
			"include_similar": searchConfig.IncludeSimilar,
		},
		"results":       searchResults,
		"total_results": len(searchResults),
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	logging.Info("memory_search_multi_repo completed successfully", "query", searchConfig.Query, "session_id", searchConfig.SessionID, "results", len(searchResults))
	return response
}

func (ms *MemoryServer) handleMemoryHealthDashboard(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_health_dashboard called", "params", params)

	repository, sessionID, err := ms.validateHealthDashboardParams(params)
	if err != nil {
		return nil, err
	}

	timeframe, includeDetails, includeRecommendations := ms.parseHealthDashboardOptions(params)

	since := ms.calculateTimeframeBoundaries(timeframe)

	chunks, err := ms.getChunksForHealthAnalysis(ctx, repository, since)
	if err != nil {
		return nil, err
	}

	healthReport := ms.generateHealthReport(chunks, timeframe, includeDetails, includeRecommendations)

	result := ms.buildHealthDashboardResponse(repository, sessionID, timeframe, since, healthReport)

	logging.Info("memory_health_dashboard completed successfully", "repository", repository, "session_id", sessionID, "chunks_analyzed", len(chunks))
	return result, nil
}

// validateHealthDashboardParams validates required parameters for health dashboard
func (ms *MemoryServer) validateHealthDashboardParams(params map[string]interface{}) (repository, sessionID string, err error) {
	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_health_dashboard failed: missing repository parameter")
		return "", "", errors.New("repository is required")
	}

	sessionID, ok = params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_health_dashboard failed: missing session_id parameter")
		return "", "", errors.New("session_id is required")
	}

	return repository, sessionID, nil
}

// parseHealthDashboardOptions parses optional parameters for health dashboard
func (ms *MemoryServer) parseHealthDashboardOptions(params map[string]interface{}) (timeframe string, includeDetails, includeRecommendations bool) {
	timeframe = types.TimeframeMonth
	if tf, ok := params["timeframe"].(string); ok && tf != "" {
		timeframe = tf
	}

	includeDetails = true
	if details, ok := params["include_details"].(bool); ok {
		includeDetails = details
	}

	includeRecommendations = true
	if recs, ok := params["include_recommendations"].(bool); ok {
		includeRecommendations = recs
	}

	return timeframe, includeDetails, includeRecommendations
}

// calculateTimeframeBoundaries calculates the time boundary for analysis
func (ms *MemoryServer) calculateTimeframeBoundaries(timeframe string) time.Time {
	switch timeframe {
	case types.TimeframWeek:
		return time.Now().AddDate(0, 0, -7)
	case types.TimeframeMonth:
		return time.Now().AddDate(0, -1, 0)
	case "quarter":
		return time.Now().AddDate(0, -3, 0)
	case types.TimeframeAll:
		return time.Time{} // Zero time = all time
	default:
		return time.Now().AddDate(0, -1, 0) // Default to month
	}
}

// getChunksForHealthAnalysis retrieves and filters chunks for health analysis
func (ms *MemoryServer) getChunksForHealthAnalysis(ctx context.Context, repository string, since time.Time) ([]types.ConversationChunk, error) {
	var chunks []types.ConversationChunk
	var err error

	if repository == GlobalMemoryRepository {
		chunks, err = ms.container.GetVectorStore().GetAllChunks(ctx)
		if err != nil {
			logging.Error("Failed to get all chunks for global health analysis", "error", err)
			return nil, fmt.Errorf("failed to get chunks: %w", err)
		}
	} else {
		chunks, err = ms.container.GetVectorStore().ListByRepository(ctx, repository, 1000, 0)
		if err != nil {
			logging.Error("Failed to list chunks for repository health analysis", "repository", repository, "error", err)
			return nil, fmt.Errorf("failed to get chunks: %w", err)
		}
	}

	// Filter by timeframe
	var filteredChunks []types.ConversationChunk
	for i := range chunks {
		if since.IsZero() || chunks[i].Timestamp.After(since) {
			filteredChunks = append(filteredChunks, chunks[i])
		}
	}

	return filteredChunks, nil
}

// buildHealthDashboardResponse builds the final response for health dashboard
func (ms *MemoryServer) buildHealthDashboardResponse(repository, sessionID, timeframe string, since time.Time, healthReport map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
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

	for i := range chunks {
		// Count outcomes
		outcomes[chunks[i].Metadata.Outcome]++

		// Count types
		chunkTypes[chunks[i].Type]++

		// Count difficulties
		difficulties[chunks[i].Metadata.Difficulty]++

		// Age analysis
		if chunks[i].Timestamp.Before(cutoffDate) {
			oldChunks++
		} else {
			recentChunks++
		}

		// Calculate effectiveness using analytics
		analytics := ms.container.GetMemoryAnalytics()
		effectiveness := analytics.CalculateEffectivenessScore(&chunks[i])
		effectivenessScores = append(effectivenessScores, effectiveness)
		totalEffectiveness += effectiveness

		// Check accessibility (has summary and good metadata)
		if chunks[i].Summary != "" && len(chunks[i].Metadata.Tags) > 0 {
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
	for i := range chunks {
		days := time.Since(chunks[i].Timestamp).Hours() / 24
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
	for i := range chunks {
		if chunks[i].Summary != "" {
			count++
		}
	}
	return count
}

func countChunksWithTags(chunks []types.ConversationChunk) int {
	count := 0
	for i := range chunks {
		if len(chunks[i].Metadata.Tags) > 0 {
			count++
		}
	}
	return count
}

func countChunksWithFiles(chunks []types.ConversationChunk) int {
	count := 0
	for i := range chunks {
		if len(chunks[i].Metadata.FilesModified) > 0 {
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
	for i := range chunks {
		if i < len(scores) {
			chunkScores = append(chunkScores, chunkScore{chunks[i], scores[i]})
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
	for i := range chunkScores {
		if i >= limit {
			break
		}
		cs := &chunkScores[i]
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

	for i := range chunks {
		if chunks[i].Timestamp.After(recentCutoff) {
			// Check for high-impact indicators
			hasHighImpact := false
			for _, tag := range chunks[i].Metadata.Tags {
				if tag == "high-impact" || tag == "architecture" || tag == "critical" {
					hasHighImpact = true
					break
				}
			}
			if hasHighImpact || chunks[i].Type == types.ChunkTypeArchitectureDecision {
				recentChunks = append(recentChunks, chunks[i])
			}
		}
	}

	// Sort by timestamp descending
	sort.Slice(recentChunks, func(i, j int) bool {
		return recentChunks[i].Timestamp.After(recentChunks[j].Timestamp)
	})

	result := make([]map[string]interface{}, 0, limit)
	for i := range recentChunks {
		if i >= limit {
			break
		}
		chunk := &recentChunks[i]
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

	for i := range chunks {
		if chunks[i].Timestamp.Before(cutoff) && chunks[i].Metadata.Outcome == types.OutcomeInProgress {
			outdatedChunks = append(outdatedChunks, chunks[i])
		}
	}

	// Sort by timestamp ascending (oldest first)
	sort.Slice(outdatedChunks, func(i, j int) bool {
		return outdatedChunks[i].Timestamp.Before(outdatedChunks[j].Timestamp)
	})

	result := make([]map[string]interface{}, 0, limit)
	for i := range outdatedChunks {
		if i >= limit {
			break
		}
		chunk := &outdatedChunks[i]
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

// getChunkByID retrieves a single chunk by ID using the direct GetByID method
func (ms *MemoryServer) getChunkByID(ctx context.Context, chunkID string) (*types.ConversationChunk, error) {
	// Use the direct GetByID method from the vector store interface
	chunk, err := ms.container.GetVectorStore().GetByID(ctx, chunkID)
	if err != nil {
		return nil, fmt.Errorf("chunk not found: %s - %w", chunkID, err)
	}

	return chunk, nil
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
		return nil, errors.New("repository is required")
	}

	sessionID, ok := params["session_id"].(string)
	if !ok || sessionID == "" {
		logging.Error("memory_decay_management failed: missing session_id parameter")
		return nil, errors.New("session_id is required")
	}

	action, ok := params["action"].(string)
	if !ok || action == "" {
		logging.Error("memory_decay_management failed: missing action parameter")
		return nil, errors.New("action is required")
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
		decayConfig, hasConfig := params["config"].(map[string]interface{})
		if !hasConfig {
			return nil, errors.New("config is required for configure action")
		}
		result = ms.handleDecayConfiguration(ctx, repository, sessionID, decayConfig)
		err = nil
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
	// Get all chunks for repository using ListByRepository instead of Search with nil embeddings
	allChunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chunks for analysis: %w", err)
	}

	// Convert to search results format for compatibility
	results := &types.SearchResults{
		Results: make([]types.SearchResult, len(allChunks)),
		Total:   len(allChunks),
	}
	for i := range allChunks {
		results.Results[i] = types.SearchResult{
			Chunk: allChunks[i],
			Score: 1.0, // Default score since we're not doing vector search
		}
	}

	chunks := make([]types.ConversationChunk, len(results.Results))
	for i := range results.Results {
		chunks[i] = results.Results[i].Chunk
	}

	// Analyze decay eligibility
	now := time.Now()
	var oldChunks, staleChunks, candidatesForSummarization, candidatesForDeletion int
	var totalAge time.Duration

	for i := range chunks {
		age := now.Sub(chunks[i].Timestamp)
		totalAge += age

		if age > 30*24*time.Hour { // Older than 30 days
			oldChunks++
		}
		if age > 7*24*time.Hour && chunks[i].Metadata.Outcome == types.OutcomeInProgress { // Stale in-progress items
			staleChunks++
		}

		// Estimate decay score
		score := ms.estimateDecayScore(&chunks[i], age)
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
	// Get all chunks for repository using ListByRepository instead of Search with nil embeddings
	allChunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chunks for preview: %w", err)
	}

	// Convert to search results format for compatibility
	results := &types.SearchResults{
		Results: make([]types.SearchResult, len(allChunks)),
		Total:   len(allChunks),
	}
	for i := range allChunks {
		results.Results[i] = types.SearchResult{
			Chunk: allChunks[i],
			Score: 1.0, // Default score since we're not doing vector search
		}
	}

	chunks := make([]types.ConversationChunk, len(results.Results))
	for i := range results.Results {
		chunks[i] = results.Results[i].Chunk
	}

	// Analyze what would be processed
	now := time.Now()
	toSummarize := make([]map[string]interface{}, 0)
	toDelete := make([]map[string]interface{}, 0)
	toUpdate := make([]map[string]interface{}, 0)

	for i := range chunks {
		age := now.Sub(chunks[i].Timestamp)

		// Skip recent memories (less than 7 days old)
		if age < 7*24*time.Hour {
			continue
		}

		score := ms.estimateDecayScore(&chunks[i], age)

		chunkInfo := map[string]interface{}{
			"id":          chunks[i].ID,
			"type":        string(chunks[i].Type),
			"summary":     chunks[i].Summary,
			"age_days":    int(age.Hours() / 24),
			"decay_score": score,
			"created_at":  chunks[i].Timestamp.Format(time.RFC3339),
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
func (ms *MemoryServer) handleDecayConfiguration(_ context.Context, repository, sessionID string, decayConfig map[string]interface{}) map[string]interface{} {
	// Parse configuration
	strategy := "adaptive"
	if s, ok := decayConfig["strategy"].(string); ok && s != "" {
		strategy = s
	}

	baseDecayRate := 0.1
	if rate, ok := decayConfig["base_decay_rate"].(float64); ok {
		baseDecayRate = rate
	}

	summarizationThreshold := 0.4
	if threshold, ok := decayConfig["summarization_threshold"].(float64); ok {
		summarizationThreshold = threshold
	}

	deletionThreshold := 0.1
	if threshold, ok := decayConfig["deletion_threshold"].(float64); ok {
		deletionThreshold = threshold
	}

	retentionPeriodDays := 7.0
	if days, ok := decayConfig["retention_period_days"].(float64); ok {
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
	}
}

// estimateDecayScore estimates the decay score for a chunk (simplified implementation)
// calculateTimeDecay applies time-based decay to the score
func (ms *MemoryServer) calculateTimeDecay(age time.Duration) float64 {
	days := age.Hours() / 24.0
	switch {
	case days < 7:
		// Minimal decay in first week
		return 1.0 - 0.01*days/7.0
	case days < 30:
		// Moderate decay for first month
		return 0.99 - 0.3*(days-7)/23.0
	default:
		// Accelerated decay after a month
		return math.Pow(0.6, (days-30)/30.0)
	}
}

// calculateImportanceBoost calculates importance boost based on chunk type
func (ms *MemoryServer) calculateImportanceBoost(chunk *types.ConversationChunk) float64 {
	switch chunk.Type {
	case types.ChunkTypeArchitectureDecision:
		return 2.0
	case types.ChunkTypeSolution:
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
			return 1.8
		}
		return 1.0
	case types.ChunkTypeProblem:
		return 1.5
	case types.ChunkTypeCodeChange:
		return 1.3
	case types.ChunkTypeDiscussion:
		return 1.1
	case types.ChunkTypeSessionSummary:
		return 1.2
	case types.ChunkTypeAnalysis:
		return 1.2
	case types.ChunkTypeVerification:
		return 1.1
	case types.ChunkTypeQuestion:
		return 1.0
	case types.ChunkTypeTask:
		return ms.calculateTaskBoost(chunk)
	case types.ChunkTypeTaskUpdate:
		return 1.2 // Updates are moderately important
	case types.ChunkTypeTaskProgress:
		return 1.1 // Progress tracking is somewhat important
	default:
		return 1.0 // Other chunk types use base score without boost
	}
}

// calculateTaskBoost calculates boost for task-oriented chunks
func (ms *MemoryServer) calculateTaskBoost(chunk *types.ConversationChunk) float64 {
	boost := 1.4 // Base boost for tasks

	if chunk.Metadata.TaskPriority != nil && *chunk.Metadata.TaskPriority == LevelHigh {
		boost *= 1.3 // High priority tasks get extra boost
	}

	if chunk.Metadata.TaskStatus != nil && *chunk.Metadata.TaskStatus == "completed" {
		boost *= 1.5 // Completed tasks are very valuable
	}

	return boost
}

// calculateRelationshipBoost calculates boost based on relationships
func (ms *MemoryServer) calculateRelationshipBoost(chunk *types.ConversationChunk) float64 {
	if len(chunk.RelatedChunks) > 0 {
		return 1.0 + float64(len(chunk.RelatedChunks))/10.0
	}
	return 1.0
}

func (ms *MemoryServer) estimateDecayScore(chunk *types.ConversationChunk, age time.Duration) float64 {
	// Base score starts at 1.0
	score := 1.0

	// Apply time decay (adaptive strategy)
	score *= ms.calculateTimeDecay(age)

	// Apply importance boost
	score *= ms.calculateImportanceBoost(chunk)

	// Consider relationships
	score *= ms.calculateRelationshipBoost(chunk)

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
func (ms *MemoryServer) searchRelatedRepositories(ctx context.Context, relaxedQuery *types.MemoryQuery, embeddings []float64, originalRepo string, searchConfig config.SearchConfig) (*types.SearchResults, error) {
	relatedRepos := ms.generateRelatedRepositories(originalRepo)
	// Limit to configured max related repos
	if len(relatedRepos) > searchConfig.MaxRelatedRepos {
		relatedRepos = relatedRepos[:searchConfig.MaxRelatedRepos]
	}

	for _, relatedRepo := range relatedRepos {
		relatedQuery := *relaxedQuery
		relatedQuery.Repository = &relatedRepo
		logging.Info("Progressive search: Step 3 - Related repo search", "original_repo", originalRepo, "trying_repo", relatedRepo)
		results, err := ms.container.GetVectorStore().Search(ctx, &relatedQuery, embeddings)
		if err != nil {
			continue // Try next related repo
		}

		if len(results.Results) > 0 {
			logging.Info("Progressive search: Related repo search succeeded", "results", len(results.Results), "repo", relatedRepo)
			return results, nil
		}
	}

	return nil, errors.New("no results found in related repositories")
}

// filterChunksBySession filters chunks by the given session ID (supports both regular and composite session IDs)
func (ms *MemoryServer) filterChunksBySession(chunks []types.ConversationChunk, sessionID string) []types.ConversationChunk {
	var filtered []types.ConversationChunk
	for i := range chunks {
		// Support both direct match and repository-scoped matching
		if chunks[i].SessionID == sessionID {
			filtered = append(filtered, chunks[i])
		} else if strings.Contains(sessionID, "::") {
			// For composite sessionID, also check if chunk's sessionID matches the session part
			expectedSession := ms.extractSessionFromComposite(sessionID)
			expectedRepo := ms.extractRepositoryFromComposite(sessionID)
			if chunks[i].SessionID == expectedSession && chunks[i].Metadata.Repository == expectedRepo {
				filtered = append(filtered, chunks[i])
			}
		}
	}
	return filtered
}

// findMostRecentSessionID finds the session ID with the most recent timestamp
// Returns repository-scoped session ID when chunks have repository information
func (ms *MemoryServer) findMostRecentSessionID(chunks []types.ConversationChunk) string {
	if len(chunks) == 0 {
		return ""
	}

	latestSessionID := chunks[0].SessionID
	latestRepository := chunks[0].Metadata.Repository
	latestTime := chunks[0].Timestamp

	for i := range chunks {
		if chunks[i].Timestamp.After(latestTime) {
			latestTime = chunks[i].Timestamp
			latestSessionID = chunks[i].SessionID
			latestRepository = chunks[i].Metadata.Repository
		}
	}

	// Return repository-scoped session ID for better isolation
	if latestRepository != "" {
		return ms.createRepositoryScopedSessionID(latestRepository, latestSessionID)
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

// runPeriodicDecay runs automatic cleanup of old chunks periodically
func (ms *MemoryServer) runPeriodicDecay(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour) // Run daily
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logging.Info("Stopping periodic decay due to context cancellation")
			return
		case <-ticker.C:
			logging.Info("Running automatic memory decay cleanup")

			// Clean up chunks older than retention period (90 days by default)
			retentionDays := 90
			deletedCount, err := ms.container.GetVectorStore().Cleanup(ctx, retentionDays)
			if err != nil {
				logging.Error("Failed to run automatic cleanup", "error", err)
				continue
			}

			if deletedCount > 0 {
				logging.Info("Automatic cleanup completed", "deleted_chunks", deletedCount, "retention_days", retentionDays)

				// Store cleanup result as memory chunk for tracking
				content := fmt.Sprintf("Automatic memory cleanup completed. Deleted %d old chunks (retention: %d days)", deletedCount, retentionDays)
				ms.storeCleanupResult(ctx, content)
			}
		}
	}
}

// storeCleanupResult stores the cleanup operation result as a memory chunk
func (ms *MemoryServer) storeCleanupResult(ctx context.Context, content string) {
	// Create metadata for cleanup result
	metadata := types.ChunkMetadata{
		Repository: GlobalMemoryRepository,
		Outcome:    types.OutcomeSuccess,
		Difficulty: types.DifficultySimple,
		Tags:       []string{"cleanup", "maintenance", "automatic"},
	}

	// Create and store cleanup chunk
	chunk, err := ms.container.GetChunkingService().CreateChunk(ctx, "system-cleanup", content, &metadata)
	if err != nil {
		logging.Error("Failed to create cleanup result chunk", "error", err)
		return
	}

	// Store the chunk
	err = ms.container.GetVectorStore().Store(ctx, chunk)
	if err != nil {
		logging.Error("Failed to store cleanup result chunk", "error", err)
	} else {
		logging.Debug("Stored cleanup result chunk", "chunk_id", chunk.ID)
	}
}

// createRepositoryScopedSessionID creates a composite session key for multi-tenant isolation
func (ms *MemoryServer) createRepositoryScopedSessionID(repository, sessionID string) string {
	if repository == "" {
		repository = "unknown"
	}
	// Use the same pattern as TodoTracker for consistency
	return repository + "::" + sessionID
}

// extractSessionFromComposite extracts the original sessionID from a composite key
func (ms *MemoryServer) extractSessionFromComposite(compositeSessionID string) string {
	parts := strings.Split(compositeSessionID, "::")
	if len(parts) >= 2 {
		return parts[1]
	}
	return compositeSessionID // Return as-is if not composite
}

// extractRepositoryFromComposite extracts the repository from a composite key
func (ms *MemoryServer) extractRepositoryFromComposite(compositeSessionID string) string {
	parts := strings.Split(compositeSessionID, "::")
	if len(parts) >= 2 {
		return parts[0]
	}
	return "unknown" // Default for non-composite keys
}

// validateAndNormalizeSessionID ensures proper session isolation and prevents cross-contamination
func (ms *MemoryServer) validateAndNormalizeSessionID(sessionID string) string {
	// Remove any whitespace and normalize
	sessionID = strings.TrimSpace(sessionID)

	// Validate session ID format (alphanumeric, hyphens, underscores, colons for composite keys)
	validSessionPattern := regexp.MustCompile(`^[a-zA-Z0-9_:-]+$`)
	if !validSessionPattern.MatchString(sessionID) {
		// Generate a safe session ID if invalid
		safeID := regexp.MustCompile(`[^a-zA-Z0-9_:-]`).ReplaceAllString(sessionID, "_")
		if safeID == "" {
			safeID = fmt.Sprintf("session_%d", time.Now().Unix())
		}
		logging.Warn("Invalid session ID format, normalized", "original", sessionID, "normalized", safeID)
		return safeID
	}

	// Ensure session ID is not too long (max 200 chars for composite keys)
	if len(sessionID) > 200 {
		sessionID = sessionID[:200]
		logging.Warn("Session ID truncated to 200 characters", "session_id", sessionID)
	}

	// Add timestamp suffix if session seems too generic (helps with isolation)
	originalSessionID := ms.extractSessionFromComposite(sessionID)
	genericSessions := []string{"session", "test", "demo", "example", ValueDefault}
	for _, generic := range genericSessions {
		if strings.EqualFold(originalSessionID, generic) {
			// For composite keys, rebuild with timestamped session part
			if strings.Contains(sessionID, "::") {
				repository := ms.extractRepositoryFromComposite(sessionID)
				timestampedSession := fmt.Sprintf("%s_%d", originalSessionID, time.Now().Unix())
				sessionID = ms.createRepositoryScopedSessionID(repository, timestampedSession)
			} else {
				sessionID = fmt.Sprintf("%s_%d", sessionID, time.Now().Unix())
			}
			logging.Info("Added timestamp to generic session ID", "session_id", sessionID)
			break
		}
	}

	return sessionID
}

// handleCheckFreshness checks the freshness of memory chunks
func (ms *MemoryServer) handleCheckFreshness(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_check_freshness called", "params", params)

	// Get required repository parameter
	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		logging.Error("memory_check_freshness failed: missing repository parameter")
		return nil, errors.New("repository parameter is required")
	}

	// Check if we're checking a single chunk or repository
	var response map[string]interface{}
	var err error

	if chunkID, ok := params["chunk_id"].(string); ok && chunkID != "" {
		// Check single chunk freshness
		response, err = ms.checkSingleChunkFreshness(ctx, chunkID)
	} else {
		// Check repository freshness
		response, err = ms.checkRepositoryFreshness(ctx, repository)
	}

	if err != nil {
		logging.Error("memory_check_freshness failed", "error", err)
		return nil, fmt.Errorf("failed to check freshness: %w", err)
	}

	logging.Info("memory_check_freshness completed successfully")
	return response, nil
}

// handleMarkRefreshed marks a memory chunk as refreshed
func (ms *MemoryServer) handleMarkRefreshed(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_mark_refreshed called", "params", params)

	// Get required parameters
	chunkID, ok := params["chunk_id"].(string)
	if !ok || chunkID == "" {
		logging.Error("memory_mark_refreshed failed: missing chunk_id parameter")
		return nil, errors.New("chunk_id parameter is required")
	}

	// Validate UUID format
	if _, err := uuid.Parse(chunkID); err != nil {
		logging.Error("memory_mark_refreshed failed: invalid chunk_id format", "chunk_id", chunkID, "error", err)
		return nil, fmt.Errorf("invalid chunk_id format: expected UUID, got '%s'. Note: chunk IDs are UUIDs, not todo IDs", chunkID)
	}

	validationNotes := ""
	if notes, ok := params["validation_notes"].(string); ok {
		validationNotes = notes
	}

	// Create freshness manager
	freshnessManager := intelligence.NewFreshnessManager(ms.container.VectorStore)

	// Mark the chunk as refreshed
	err := freshnessManager.MarkRefreshed(ctx, chunkID, validationNotes)
	if err != nil {
		logging.Error("memory_mark_refreshed failed", "error", err)
		return nil, fmt.Errorf("failed to mark chunk as refreshed: %w", err)
	}

	// Get updated chunk for response
	chunk, err := ms.container.VectorStore.GetByID(ctx, chunkID)
	if err != nil {
		logging.Error("memory_mark_refreshed failed to retrieve updated chunk", "error", err)
		return nil, fmt.Errorf("failed to retrieve updated chunk: %w", err)
	}

	response := map[string]interface{}{
		"chunk_id":         chunkID,
		"marked_refreshed": true,
		"timestamp":        time.Now().Format(time.RFC3339),
		"validation_notes": validationNotes,
		"updated_chunk": map[string]interface{}{
			"id":             chunk.ID,
			"type":           string(chunk.Type),
			"summary":        chunk.Summary,
			"repository":     chunk.Metadata.Repository,
			"last_refreshed": chunk.Metadata.ExtendedMetadata["last_refreshed"],
		},
	}

	logging.Info("memory_mark_refreshed completed successfully", "chunk_id", chunkID)
	return response, nil
}

// checkSingleChunkFreshness checks freshness for a single chunk
func (ms *MemoryServer) checkSingleChunkFreshness(ctx context.Context, chunkID string) (map[string]interface{}, error) {
	// Get the chunk
	chunk, err := ms.container.VectorStore.GetByID(ctx, chunkID)
	if err != nil {
		return nil, fmt.Errorf("chunk not found: %w", err)
	}

	// Create freshness manager
	freshnessManager := intelligence.NewFreshnessManager(ms.container.VectorStore)

	// Check freshness
	status, err := freshnessManager.CheckFreshness(ctx, chunk)
	if err != nil {
		return nil, err
	}

	response := map[string]interface{}{
		"chunk_id": chunkID,
		"freshness_status": map[string]interface{}{
			"is_fresh":        status.IsFresh,
			"is_stale":        status.IsStale,
			"freshness_score": status.FreshnessScore,
			"days_old":        status.DaysOld,
			"decay_rate":      status.DecayRate,
			"last_checked":    status.LastChecked.Format(time.RFC3339),
		},
		"alerts":            status.Alerts,
		"suggested_actions": status.SuggestedActions,
		"chunk_info": map[string]interface{}{
			"type":       string(chunk.Type),
			"repository": chunk.Metadata.Repository,
			"timestamp":  chunk.Timestamp.Format(time.RFC3339),
			"summary":    chunk.Summary,
		},
	}

	return response, nil
}

// checkRepositoryFreshness checks freshness for an entire repository
func (ms *MemoryServer) checkRepositoryFreshness(ctx context.Context, repository string) (map[string]interface{}, error) {
	// Create freshness manager
	freshnessManager := intelligence.NewFreshnessManager(ms.container.VectorStore)

	// Check repository freshness
	batch, err := freshnessManager.CheckRepositoryFreshness(ctx, repository)
	if err != nil {
		return nil, err
	}

	// Format batch results
	results := make([]map[string]interface{}, len(batch.Results))
	for i := range batch.Results {
		result := &batch.Results[i]
		results[i] = map[string]interface{}{
			"chunk_id":   result.ChunkID,
			"type":       string(result.Type),
			"repository": result.Repository,
			"freshness_status": map[string]interface{}{
				"is_fresh":                result.FreshnessStatus.IsFresh,
				"is_stale":                result.FreshnessStatus.IsStale,
				"freshness_score":         result.FreshnessStatus.FreshnessScore,
				"days_old":                result.FreshnessStatus.DaysOld,
				"decay_rate":              result.FreshnessStatus.DecayRate,
				"alerts_count":            len(result.FreshnessStatus.Alerts),
				"suggested_actions_count": len(result.FreshnessStatus.SuggestedActions),
			},
		}
	}

	response := map[string]interface{}{
		"repository": repository,
		"batch_summary": map[string]interface{}{
			"total_checked":      batch.TotalChecked,
			"fresh_count":        batch.FreshCount,
			"stale_count":        batch.StaleCount,
			"alerts_generated":   batch.AlertsGenerated,
			"processing_time_ms": batch.ProcessingTime.Milliseconds(),
		},
		"results": results,
		"summary": map[string]interface{}{
			"by_repository":  batch.Summary.ByRepository,
			"by_type":        batch.Summary.ByType,
			"by_technology":  batch.Summary.ByTechnology,
			"overall_health": batch.Summary.OverallHealth,
		},
	}

	return response, nil
}

// handleGenerateCitations generates formatted citations for memory chunks
func (ms *MemoryServer) handleGenerateCitations(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_generate_citations called", "params", params)

	query, chunkIDs, err := ms.validateCitationParams(params)
	if err != nil {
		return nil, err
	}

	citationStyle, groupSources, includeContext := ms.parseCitationOptions(params)
	_ = includeContext // TODO: Use in citation configuration

	results, err := ms.buildSearchResultsForCitation(ctx, chunkIDs)
	if err != nil {
		return nil, err
	}

	citations, err := ms.generateCitationsFromResults(ctx, results, query, citationStyle, groupSources)
	if err != nil {
		return nil, err
	}

	response := ms.buildCitationResponse(citations)

	logging.Info("memory_generate_citations completed successfully", "response_id", citations.ResponseID)
	return response, nil
}

// validateCitationParams validates required parameters for citation generation
func (ms *MemoryServer) validateCitationParams(params map[string]interface{}) (query string, chunkIDs []string, err error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		logging.Error("memory_generate_citations failed: missing query parameter")
		return "", nil, errors.New("query parameter is required")
	}

	chunkIDsInterface, ok := params["chunk_ids"].([]interface{})
	if !ok {
		logging.Error("memory_generate_citations failed: missing or invalid chunk_ids parameter")
		return "", nil, errors.New("chunk_ids parameter is required and must be an array")
	}

	chunkIDs = make([]string, len(chunkIDsInterface))
	for i, id := range chunkIDsInterface {
		chunkID, ok := id.(string)
		if !ok {
			return "", nil, errors.New("all chunk IDs must be strings")
		}
		chunkIDs[i] = chunkID
	}

	return query, chunkIDs, nil
}

// parseCitationOptions parses optional parameters for citation generation
func (ms *MemoryServer) parseCitationOptions(params map[string]interface{}) (citationStyle string, groupSources, includeContext bool) {
	citationStyle = "simple"
	if style, ok := params["citation_style"].(string); ok {
		citationStyle = style
	}

	groupSources = true
	if group, ok := params["group_sources"].(bool); ok {
		groupSources = group
	}

	includeContext = true
	if include, ok := params["include_context"].(bool); ok {
		includeContext = include
	}

	return citationStyle, groupSources, includeContext
}

// buildSearchResultsForCitation creates search results from chunk IDs for citation
func (ms *MemoryServer) buildSearchResultsForCitation(ctx context.Context, chunkIDs []string) ([]types.SearchResult, error) {
	results := make([]types.SearchResult, 0, len(chunkIDs))

	for _, chunkID := range chunkIDs {
		chunk, err := ms.container.VectorStore.GetByID(ctx, chunkID)
		if err != nil {
			logging.Warn("Failed to get chunk for citation", "chunk_id", chunkID, "error", err)
			continue
		}

		result := types.SearchResult{
			Chunk: *chunk,
			Score: 1.0, // Default relevance for manual citations
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		return nil, errors.New("no valid chunks found for citation generation")
	}

	return results, nil
}

// generateCitationsFromResults generates citations using the citation manager
func (ms *MemoryServer) generateCitationsFromResults(ctx context.Context, results []types.SearchResult, query, citationStyle string, groupSources bool) (*intelligence.ResponseCitations, error) {
	citationManager := intelligence.NewCitationManager(ms.container.VectorStore)

	citationConfig := intelligence.DefaultCitationConfig()
	citationConfig.CitationStyle = citationStyle
	citationConfig.GroupSimilarSources = groupSources
	if citationStyle == "apa" {
		citationConfig.IncludeTimestamps = true
		citationConfig.IncludeRepository = true
	}
	citationManager.SetConfig(citationConfig)

	citations, err := citationManager.GenerateCitations(ctx, results, query)
	if err != nil {
		logging.Error("memory_generate_citations failed", "error", err)
		return nil, fmt.Errorf("failed to generate citations: %w", err)
	}

	return citations, nil
}

// buildCitationResponse builds the final citation response
func (ms *MemoryServer) buildCitationResponse(citations *intelligence.ResponseCitations) map[string]interface{} {
	response := map[string]interface{}{
		"response_id":            citations.ResponseID,
		"query":                  citations.Query,
		"total_citations":        citations.TotalCitations,
		"style":                  citations.Style,
		"formatted_bibliography": citations.FormattedBibliography,
		"generated_at":           citations.GeneratedAt.Format(time.RFC3339),
	}

	if len(citations.Groups) > 0 {
		response["groups"] = citations.Groups
	}

	if len(citations.IndividualCitations) > 0 {
		response["individual_citations"] = citations.IndividualCitations
	}

	if len(citations.InlineCitations) > 0 {
		response["inline_citations"] = citations.InlineCitations
	}

	return response
}

// handleCreateInlineCitation creates inline citation for specific text
func (ms *MemoryServer) handleCreateInlineCitation(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context parameter required for MCP handler interface consistency
	logging.Info("MCP TOOL: memory_create_inline_citation called", "params", params)

	// Get required parameters
	text, ok := params["text"].(string)
	if !ok || text == "" {
		logging.Error("memory_create_inline_citation failed: missing text parameter")
		return nil, errors.New("text parameter is required")
	}

	responseID, ok := params["response_id"].(string)
	if !ok || responseID == "" {
		logging.Error("memory_create_inline_citation failed: missing response_id parameter")
		return nil, errors.New("response_id parameter is required")
	}

	// Get optional format parameter
	format := "bracket"
	if f, ok := params["format"].(string); ok {
		format = f
	}

	// For this implementation, we'll create a simplified inline citation
	// In a production system, we would look up the stored citations by response_id

	// Create citation manager with inline format
	citationManager := intelligence.NewCitationManager(ms.container.VectorStore)
	citationConfig := intelligence.DefaultCitationConfig()

	// Configure inline format based on requested format
	switch format {
	case "superscript":
		citationConfig.CustomFormats["inline"] = intelligence.CitationFormat{
			Template:  "^{id}",
			Fields:    []string{"id"},
			Separator: ",",
		}
	case "parenthetical":
		citationConfig.CustomFormats["inline"] = intelligence.CitationFormat{
			Template:  "({id})",
			Fields:    []string{"id"},
			Separator: ", ",
		}
	default: // bracket
		citationConfig.CustomFormats["inline"] = intelligence.CitationFormat{
			Template:  "[{id}]",
			Fields:    []string{"id"},
			Separator: ", ",
		}
	}

	citationManager.SetConfig(citationConfig)

	// For demonstration, create a simple inline reference
	// In production, this would reference the actual stored citations
	inlineRef := ""
	switch format {
	case "superscript":
		inlineRef = "^1"
	case "parenthetical":
		inlineRef = "(1)"
	default:
		inlineRef = "[1]"
	}

	response := map[string]interface{}{
		"original_text":    text,
		"cited_text":       text + " " + inlineRef,
		"inline_reference": inlineRef,
		"format":           format,
		"response_id":      responseID,
		"created_at":       time.Now().Format(time.RFC3339),
	}

	logging.Info("memory_create_inline_citation completed successfully", "response_id", responseID)
	return response, nil
}

// Helper methods for enhanced conflict detection

// convertConflictsToLegacyFormat converts new conflict format to legacy format for backward compatibility
func (ms *MemoryServer) convertConflictsToLegacyFormat(conflicts []intelligence.Conflict) map[string][]map[string]interface{} {
	result := map[string][]map[string]interface{}{
		"outcome_conflicts":       []map[string]interface{}{},
		"architectural_conflicts": []map[string]interface{}{},
		"pattern_conflicts":       []map[string]interface{}{},
		"technical_conflicts":     []map[string]interface{}{},
		"temporal_conflicts":      []map[string]interface{}{},
	}

	for i := range conflicts {
		conflict := &conflicts[i]
		legacyConflict := map[string]interface{}{
			"type":        string(conflict.Type),
			"description": conflict.Description,
			"severity":    string(conflict.Severity),
			"confidence":  conflict.Confidence,
			"chunk1": map[string]interface{}{
				"id":        conflict.PrimaryChunk.ID,
				"summary":   conflict.PrimaryChunk.Summary,
				"timestamp": conflict.PrimaryChunk.Timestamp.Format(time.RFC3339),
			},
			"chunk2": map[string]interface{}{
				"id":        conflict.ConflictChunk.ID,
				"summary":   conflict.ConflictChunk.Summary,
				"timestamp": conflict.ConflictChunk.Timestamp.Format(time.RFC3339),
			},
		}

		switch conflict.Type {
		case intelligence.ConflictTypeOutcome:
			result["outcome_conflicts"] = append(result["outcome_conflicts"], legacyConflict)
		case intelligence.ConflictTypeArchitectural, intelligence.ConflictTypeDecision:
			result["architectural_conflicts"] = append(result["architectural_conflicts"], legacyConflict)
		case intelligence.ConflictTypePattern, intelligence.ConflictTypeMethodology:
			result["pattern_conflicts"] = append(result["pattern_conflicts"], legacyConflict)
		case intelligence.ConflictTypeTechnical:
			result["technical_conflicts"] = append(result["technical_conflicts"], legacyConflict)
		case intelligence.ConflictTypeTemporal:
			result["temporal_conflicts"] = append(result["temporal_conflicts"], legacyConflict)
		default:
			result["outcome_conflicts"] = append(result["outcome_conflicts"], legacyConflict)
		}
	}

	return result
}

// categorizeConflictsByType categorizes conflicts by their type
func (ms *MemoryServer) categorizeConflictsByType(conflicts []intelligence.Conflict) map[string]int {
	categories := map[string]int{
		"architectural": 0,
		"technical":     0,
		"temporal":      0,
		"outcome":       0,
		"decision":      0,
		"methodology":   0,
		"pattern":       0,
	}

	for i := range conflicts {
		conflict := &conflicts[i]
		switch conflict.Type {
		case intelligence.ConflictTypeArchitectural:
			categories["architectural"]++
		case intelligence.ConflictTypeTechnical:
			categories["technical"]++
		case intelligence.ConflictTypeTemporal:
			categories["temporal"]++
		case intelligence.ConflictTypeOutcome:
			categories["outcome"]++
		case intelligence.ConflictTypeDecision:
			categories["decision"]++
		case intelligence.ConflictTypeMethodology:
			categories["methodology"]++
		case intelligence.ConflictTypePattern:
			categories["pattern"]++
		}
	}

	return categories
}

// categorizeConflictsBySeverity2 categorizes conflicts by their severity (new version)
func (ms *MemoryServer) categorizeConflictsBySeverity2(conflicts []intelligence.Conflict) map[string]int {
	severities := map[string]int{
		"critical":  0,
		LevelHigh:   0,
		LevelMedium: 0,
		"low":       0,
		"info":      0,
	}

	for i := range conflicts {
		conflict := &conflicts[i]
		switch conflict.Severity {
		case intelligence.SeverityCritical:
			severities["critical"]++
		case intelligence.SeverityHigh:
			severities[LevelHigh]++
		case intelligence.SeverityMedium:
			severities[LevelMedium]++
		case intelligence.SeverityLow:
			severities["low"]++
		case intelligence.SeverityInfo:
			severities["info"]++
		}
	}

	return severities
}

// countHighPriorityConflicts counts conflicts with high or critical severity
func (ms *MemoryServer) countHighPriorityConflicts(conflicts []intelligence.Conflict) int {
	count := 0
	for i := range conflicts {
		conflict := &conflicts[i]
		if conflict.Severity == intelligence.SeverityCritical || conflict.Severity == intelligence.SeverityHigh {
			count++
		}
	}
	return count
}

// summarizeRecommendations provides a summary of resolution recommendations
func (ms *MemoryServer) summarizeRecommendations(recommendations []intelligence.ResolutionRecommendation) map[string]interface{} {
	if len(recommendations) == 0 {
		return map[string]interface{}{
			"total_recommendations":       0,
			"strategy_distribution":       map[string]int{},
			"avg_strategies_per_conflict": 0.0,
		}
	}

	strategyDistribution := map[string]int{
		"accept_latest":   0,
		"accept_highest":  0,
		"merge":           0,
		"manual_review":   0,
		"contextual":      0,
		"evolutionary":    0,
		"domain_specific": 0,
	}

	totalStrategies := 0
	for i := range recommendations {
		totalStrategies += len(recommendations[i].Strategies)
		for j := range recommendations[i].Strategies {
			strategy := &recommendations[i].Strategies[j]
			switch strategy.Type {
			case intelligence.ResolutionAcceptLatest:
				strategyDistribution["accept_latest"]++
			case intelligence.ResolutionAcceptHighest:
				strategyDistribution["accept_highest"]++
			case intelligence.ResolutionMerge:
				strategyDistribution["merge"]++
			case intelligence.ResolutionManualReview:
				strategyDistribution["manual_review"]++
			case intelligence.ResolutionContextual:
				strategyDistribution["contextual"]++
			case intelligence.ResolutionEvolutionary:
				strategyDistribution["evolutionary"]++
			case intelligence.ResolutionDomain:
				strategyDistribution["domain_specific"]++
			}
		}
	}

	avgStrategiesPerConflict := float64(totalStrategies) / float64(len(recommendations))

	return map[string]interface{}{
		"total_recommendations":       len(recommendations),
		"strategy_distribution":       strategyDistribution,
		"avg_strategies_per_conflict": avgStrategiesPerConflict,
		"total_strategies":            totalStrategies,
	}
}

// Bulk operations handlers

// handleBulkOperation executes bulk operations on multiple memories
func (ms *MemoryServer) handleBulkOperation(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_bulk_operation called", "params", params)

	operation, err := ms.validateBulkOperationParams(params)
	if err != nil {
		return nil, err
	}

	req, err := ms.buildBulkOperationRequest(params, operation)
	if err != nil {
		return nil, err
	}

	bulkReq, err := req.ToRequest()
	if err != nil {
		return nil, fmt.Errorf("invalid bulk request: %w", err)
	}

	progress, err := ms.bulkManager.SubmitOperation(ctx, &bulkReq)
	if err != nil {
		return nil, fmt.Errorf("failed to submit bulk operation: %w", err)
	}

	response := ms.buildBulkOperationResponse(progress, string(bulkReq.Operation))

	logging.Info("memory_bulk_operation completed successfully", "operation_id", progress.OperationID)
	return response, nil
}

// validateBulkOperationParams validates required parameters for bulk operation
func (ms *MemoryServer) validateBulkOperationParams(params map[string]interface{}) (string, error) {
	operation, ok := params["operation"].(string)
	if !ok {
		return "", errors.New("operation parameter is required")
	}
	return operation, nil
}

// buildBulkOperationRequest builds the bulk operation request from parameters
func (ms *MemoryServer) buildBulkOperationRequest(params map[string]interface{}, operation string) (bulk.OperationRequest, error) {
	req := bulk.OperationRequest{
		Operation: operation,
	}

	if err := ms.parseBulkChunks(params, &req); err != nil {
		return req, err
	}

	ms.parseBulkIDs(params, &req)
	ms.parseBulkConfiguration(params, &req)

	return req, nil
}

// parseBulkChunks parses chunk data for bulk operations
func (ms *MemoryServer) parseBulkChunks(params map[string]interface{}, req *bulk.OperationRequest) error {
	chunks, ok := params["chunks"].([]interface{})
	if !ok {
		return nil
	}

	req.Chunks = make([]types.ConversationChunk, len(chunks))
	for i := range chunks {
		chunkData, err := json.Marshal(chunks[i])
		if err != nil {
			return fmt.Errorf("failed to marshal chunk at index %d: %w", i, err)
		}

		if err := json.Unmarshal(chunkData, &req.Chunks[i]); err != nil {
			return fmt.Errorf("invalid chunk at index %d: %w", i, err)
		}
	}

	return nil
}

// parseBulkIDs parses ID list for bulk operations
func (ms *MemoryServer) parseBulkIDs(params map[string]interface{}, req *bulk.OperationRequest) {
	ids, ok := params["ids"].([]interface{})
	if !ok {
		return
	}

	req.IDs = make([]string, len(ids))
	for i, id := range ids {
		if idStr, ok := id.(string); ok {
			req.IDs[i] = idStr
		}
	}
}

// parseBulkConfiguration parses configuration options for bulk operations
func (ms *MemoryServer) parseBulkConfiguration(params map[string]interface{}, req *bulk.OperationRequest) {
	if batchSize, ok := params["batch_size"].(float64); ok {
		req.BatchSize = int(batchSize)
	}
	if maxConcurrency, ok := params["max_concurrency"].(float64); ok {
		req.MaxConcurrency = int(maxConcurrency)
	}
	if validateFirst, ok := params["validate_first"].(bool); ok {
		req.ValidateFirst = validateFirst
	}
	if continueOnError, ok := params["continue_on_error"].(bool); ok {
		req.ContinueOnError = continueOnError
	}
	if dryRun, ok := params["dry_run"].(bool); ok {
		req.DryRun = dryRun
	}
	if conflictPolicy, ok := params["conflict_policy"].(string); ok {
		req.ConflictPolicy = conflictPolicy
	}
}

// buildBulkOperationResponse builds the response for bulk operation
func (ms *MemoryServer) buildBulkOperationResponse(progress *bulk.Progress, operation string) map[string]interface{} {
	return map[string]interface{}{
		"operation_id": progress.OperationID,
		"status":       string(progress.Status),
		"total_items":  progress.TotalItems,
		"message":      fmt.Sprintf("Bulk %s operation started with %d items", operation, progress.TotalItems),
	}
}

// handleBulkImport imports memories from various formats
func (ms *MemoryServer) handleBulkImport(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_bulk_import called", "params", params)

	// Required data parameter
	data, ok := params["data"].(string)
	if !ok || data == "" {
		return nil, errors.New("data parameter is required")
	}

	// Build import options
	options := bulk.ImportOptions{
		Format:           bulk.FormatAuto,
		ChunkingStrategy: bulk.ChunkingAuto,
		ConflictPolicy:   bulk.ConflictPolicySkip,
		ValidateChunks:   true,
	}

	// Optional parameters
	if format, ok := params["format"].(string); ok {
		options.Format = bulk.ImportFormat(format)
	}
	if repo, ok := params["repository"].(string); ok {
		options.Repository = repo
	}
	if sessionID, ok := params["default_session_id"].(string); ok {
		options.DefaultSessionID = sessionID
	}
	if tags, ok := params["default_tags"].([]interface{}); ok {
		options.DefaultTags = make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				options.DefaultTags[i] = tagStr
			}
		}
	}
	if strategy, ok := params["chunking_strategy"].(string); ok {
		options.ChunkingStrategy = bulk.ChunkingStrategy(strategy)
	}
	if policy, ok := params["conflict_policy"].(string); ok {
		options.ConflictPolicy = bulk.ConflictPolicy(policy)
	}
	if validate, ok := params["validate_chunks"].(bool); ok {
		options.ValidateChunks = validate
	}

	// Parse metadata if provided
	ms.parseImportMetadata(params, &options)

	// Import data
	result, err := ms.bulkImporter.Import(ctx, data, &options)
	if err != nil {
		return nil, fmt.Errorf("import failed: %w", err)
	}

	// Store imported chunks if successful
	if len(result.Chunks) > 0 {
		bulkReq := bulk.Request{
			Operation: bulk.OperationStore,
			Chunks:    result.Chunks,
			Options: bulk.Options{
				BatchSize:       50,
				MaxConcurrency:  3,
				ValidateFirst:   false, // Already validated during import
				ContinueOnError: true,
			},
		}

		progress, err := ms.bulkManager.SubmitOperation(ctx, &bulkReq)
		if err != nil {
			logging.Warn("Failed to store imported chunks", "error", err)
		} else {
			result.Summary += fmt.Sprintf(". Storage operation %s started.", progress.OperationID)
		}
	}

	logging.Info("memory_bulk_import completed successfully", "imported_items", result.SuccessfulItems)
	return map[string]interface{}{
		"total_items":      result.TotalItems,
		"processed_items":  result.ProcessedItems,
		"successful_items": result.SuccessfulItems,
		"failed_items":     result.FailedItems,
		"skipped_items":    result.SkippedItems,
		"errors":           result.Errors,
		"warnings":         result.Warnings,
		"summary":          result.Summary,
	}, nil
}

// parseImportMetadata extracts metadata from parameters and updates import options
func (ms *MemoryServer) parseImportMetadata(params map[string]interface{}, options *bulk.ImportOptions) {
	metadataInterface, ok := params["metadata"].(map[string]interface{})
	if !ok {
		return
	}

	if sourceSystem, ok := metadataInterface["source_system"].(string); ok {
		options.Metadata.SourceSystem = sourceSystem
	}

	if importDate, ok := metadataInterface["import_date"].(string); ok {
		options.Metadata.ImportDate = importDate
	}

	if tags, ok := metadataInterface["tags"].([]interface{}); ok {
		options.Metadata.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				options.Metadata.Tags[i] = tagStr
			}
		}
	}
}

// parseBulkExportOptions extracts and validates export options from parameters
func (ms *MemoryServer) parseBulkExportOptions(params map[string]interface{}) (bulk.ExportOptions, error) {
	options := bulk.ExportOptions{
		Format:           bulk.ExportFormatJSON,
		Compression:      bulk.CompressionNone,
		IncludeVectors:   false,
		IncludeMetadata:  true,
		IncludeRelations: false,
		PrettyPrint:      true,
	}

	// Parse basic options
	if format, ok := params["format"].(string); ok {
		options.Format = bulk.ExportFormat(format)
	}
	if compression, ok := params["compression"].(string); ok {
		options.Compression = bulk.CompressionType(compression)
	}
	if includeVectors, ok := params["include_vectors"].(bool); ok {
		options.IncludeVectors = includeVectors
	}
	if includeMetadata, ok := params["include_metadata"].(bool); ok {
		options.IncludeMetadata = includeMetadata
	}
	if includeRelations, ok := params["include_relations"].(bool); ok {
		options.IncludeRelations = includeRelations
	}
	if prettyPrint, ok := params["pretty_print"].(bool); ok {
		options.PrettyPrint = prettyPrint
	}

	// Parse complex options
	if err := ms.parseExportFilter(params, &options); err != nil {
		return options, err
	}
	ms.parseExportSorting(params, &options)
	ms.parseExportPagination(params, &options)

	return options, nil
}

// parseExportFilter parses filter options for export
func (ms *MemoryServer) parseExportFilter(params map[string]interface{}, options *bulk.ExportOptions) error {
	filterInterface, ok := params["filter"].(map[string]interface{})
	if !ok {
		return nil
	}

	filter := bulk.ExportFilter{}

	if repo, ok := filterInterface["repository"].(string); ok {
		filter.Repository = &repo
	}

	if err := ms.parseExportFilterArrays(filterInterface, &filter); err != nil {
		return err
	}

	if contentFilter, ok := filterInterface["content_filter"].(string); ok {
		filter.ContentFilter = &contentFilter
	}

	if err := ms.parseExportDateRange(filterInterface, &filter); err != nil {
		return err
	}

	options.Filter = filter
	return nil
}

// parseExportFilterArrays parses array fields in export filter
func (ms *MemoryServer) parseExportFilterArrays(filterInterface map[string]interface{}, filter *bulk.ExportFilter) error {
	if sessionIDs, ok := filterInterface["session_ids"].([]interface{}); ok {
		filter.SessionIDs = make([]string, len(sessionIDs))
		for i, id := range sessionIDs {
			if idStr, ok := id.(string); ok {
				filter.SessionIDs[i] = idStr
			}
		}
	}

	if chunkTypes, ok := filterInterface["chunk_types"].([]interface{}); ok {
		filter.ChunkTypes = make([]types.ChunkType, len(chunkTypes))
		for i, ct := range chunkTypes {
			if ctStr, ok := ct.(string); ok {
				filter.ChunkTypes[i] = types.ChunkType(ctStr)
			}
		}
	}

	if tags, ok := filterInterface["tags"].([]interface{}); ok {
		filter.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				filter.Tags[i] = tagStr
			}
		}
	}

	return nil
}

// parseExportDateRange parses date range for export filter
func (ms *MemoryServer) parseExportDateRange(filterInterface map[string]interface{}, filter *bulk.ExportFilter) error {
	dateRange, ok := filterInterface["date_range"].(map[string]interface{})
	if !ok {
		return nil
	}

	dr := &bulk.ExportDateRange{}
	if start, ok := dateRange["start"].(string); ok {
		if startTime, err := time.Parse(time.RFC3339, start); err == nil {
			dr.Start = &startTime
		}
	}
	if end, ok := dateRange["end"].(string); ok {
		if endTime, err := time.Parse(time.RFC3339, end); err == nil {
			dr.End = &endTime
		}
	}
	filter.DateRange = dr
	return nil
}

// parseExportSorting parses sorting options for export
func (ms *MemoryServer) parseExportSorting(params map[string]interface{}, options *bulk.ExportOptions) {
	sortInterface, ok := params["sorting"].(map[string]interface{})
	if !ok {
		return
	}

	sorting := bulk.ExportSorting{}
	if field, ok := sortInterface["field"].(string); ok {
		sorting.Field = field
	}
	if order, ok := sortInterface["order"].(string); ok {
		sorting.Order = order
	}
	options.Sorting = sorting
}

// parseExportPagination parses pagination options for export
func (ms *MemoryServer) parseExportPagination(params map[string]interface{}, options *bulk.ExportOptions) {
	paginationInterface, ok := params["pagination"].(map[string]interface{})
	if !ok {
		return
	}

	pagination := bulk.ExportPagination{}
	if limit, ok := paginationInterface["limit"].(float64); ok {
		pagination.Limit = int(limit)
	}
	if offset, ok := paginationInterface["offset"].(float64); ok {
		pagination.Offset = int(offset)
	}
	options.Pagination = pagination
}

// buildExportResponse constructs the response for bulk export
func (ms *MemoryServer) buildExportResponse(result *bulk.ExportResult) map[string]interface{} {
	return map[string]interface{}{
		"format":         string(result.Format),
		"total_items":    result.TotalItems,
		"exported_items": result.ExportedItems,
		"data_size":      result.DataSize,
		"data":           result.Data,
		"metadata":       result.Metadata,
		"warnings":       result.Warnings,
		"generated_at":   result.GeneratedAt.Format(time.RFC3339),
	}
}

// handleBulkExport exports memories to various formats
func (ms *MemoryServer) handleBulkExport(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_bulk_export called", "params", params)

	// Parse export options
	options, err := ms.parseBulkExportOptions(params)
	if err != nil {
		return nil, err
	}

	// Export data
	result, err := ms.bulkExporter.Export(ctx, &options)
	if err != nil {
		return nil, fmt.Errorf("export failed: %w", err)
	}

	logging.Info("memory_bulk_export completed successfully", "exported_items", result.ExportedItems)
	return ms.buildExportResponse(result), nil
}

// handleSecureBulkDelete securely deletes multiple memories with repository validation
func (ms *MemoryServer) handleSecureBulkDelete(ctx context.Context, params map[string]interface{}, repository string) (interface{}, error) {
	logging.Info("MCP TOOL: memory_secure_bulk_delete called", "params", params, "repository", repository)

	// Parse IDs to delete
	idsInterface, ok := params["ids"]
	if !ok {
		return nil, errors.New("ids parameter is required for bulk delete. Example: {\"ids\": [\"chunk-id-1\", \"chunk-id-2\"], \"repository\": \"github.com/user/repo\"}")
	}

	idsSlice, ok := idsInterface.([]interface{})
	if !ok {
		return nil, errors.New("ids parameter must be an array of strings. Example: {\"ids\": [\"chunk-id-1\", \"chunk-id-2\"], \"repository\": \"github.com/user/repo\"}")
	}

	if len(idsSlice) == 0 {
		return map[string]interface{}{
			"status":         "success",
			"deleted_count":  0,
			"verified_count": 0,
			"rejected_count": 0,
			"message":        "No IDs provided for deletion",
		}, nil
	}

	// Convert to string slice and validate ownership
	validIds := []string{}
	rejectedIds := []string{}

	vectorStore := ms.container.GetVectorStore()

	for _, idInterface := range idsSlice {
		id, ok := idInterface.(string)
		if !ok {
			rejectedIds = append(rejectedIds, fmt.Sprintf("invalid_id_%v", idInterface))
			continue
		}

		// SECURITY: Verify chunk belongs to the specified repository before deletion
		chunk, err := vectorStore.GetByID(ctx, id)
		if err != nil {
			logging.Warn("Chunk not found during secure delete validation", "id", id, "error", err)
			rejectedIds = append(rejectedIds, id)
			continue
		}

		// Verify repository ownership - CRITICAL SECURITY CHECK
		if chunk.Metadata.Repository != repository {
			logging.Warn("SECURITY: Attempted cross-repository deletion blocked",
				"chunk_id", id,
				"chunk_repository", chunk.Metadata.Repository,
				"requested_repository", repository)
			rejectedIds = append(rejectedIds, id)
			continue
		}

		validIds = append(validIds, id)
	}

	// Only delete chunks that belong to the specified repository
	deletedCount := 0
	for _, id := range validIds {
		err := vectorStore.Delete(ctx, id)
		if err != nil {
			logging.Error("Failed to delete chunk", "id", id, "error", err)
			rejectedIds = append(rejectedIds, id)
		} else {
			deletedCount++
		}
	}

	result := map[string]interface{}{
		"status":          "success",
		"repository":      repository,
		"total_requested": len(idsSlice),
		"verified_count":  len(validIds),
		"deleted_count":   deletedCount,
		"rejected_count":  len(rejectedIds),
	}

	if len(rejectedIds) > 0 {
		result["rejected_ids"] = rejectedIds
		result["security_note"] = "Some deletions were rejected due to repository ownership validation"
	}

	logging.Info("Secure bulk delete completed",
		"repository", repository,
		"deleted", deletedCount,
		"rejected", len(rejectedIds),
		"total", len(idsSlice))

	return result, nil
}

// handleSecureSearch performs repository-scoped search without progressive fallback
func (ms *MemoryServer) handleSecureSearch(ctx context.Context, params map[string]interface{}, repository string) (interface{}, error) {
	logging.Info("MCP TOOL: memory_secure_search called", "params", params, "repository", repository)

	// Parse search query
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, errors.New("query parameter is required for search. Example: {\"query\": \"authentication bug fix\", \"repository\": \"github.com/user/repo\"}")
	}

	// Create memory query with strict repository isolation
	memQuery := types.MemoryQuery{
		Query: query,
		Limit: 10, // Default limit
	}

	// SECURITY: Always enforce repository isolation (no fallback)
	if repository != GlobalRepository {
		memQuery.Repository = &repository
	}

	// Parse optional parameters
	if limit, ok := params["limit"].(float64); ok && limit > 0 {
		memQuery.Limit = int(limit)
	}

	if minRelevance, ok := params["min_relevance"].(float64); ok && minRelevance > 0 {
		memQuery.MinRelevanceScore = minRelevance
	} else {
		memQuery.MinRelevanceScore = 0.5 // Default relevance threshold
	}

	// Parse type filters if provided
	if typesInterface, ok := params["types"].([]interface{}); ok {
		chunkTypes := make([]types.ChunkType, 0, len(typesInterface))
		for _, typeInterface := range typesInterface {
			if typeStr, ok := typeInterface.(string); ok {
				chunkTypes = append(chunkTypes, types.ChunkType(typeStr))
			}
		}
		memQuery.Types = chunkTypes
	}

	// Parse recency filter
	if recency, ok := params["recency"].(string); ok {
		memQuery.Recency = types.Recency(recency)
	}

	// Generate embeddings for the query
	embeddingService := ms.container.GetEmbeddingService()
	embeddings, err := embeddingService.Generate(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Perform SECURE search (no progressive fallback that breaks repository isolation)
	vectorStore := ms.container.GetVectorStore()
	results, err := vectorStore.Search(ctx, &memQuery, embeddings)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Build response
	response := map[string]interface{}{
		"status":        "success",
		"repository":    repository,
		"query":         query,
		"total":         results.Total,
		"results":       results.Results,
		"query_time":    results.QueryTime.Milliseconds(),
		"security_note": "Repository-scoped search with no cross-tenant fallback",
	}

	if repository == GlobalRepository {
		response["scope"] = GlobalRepository
		response["security_note"] = "Global search across all repositories for architecture decisions"
	}

	logging.Info("Secure search completed",
		"repository", repository,
		"query", query,
		"results_count", results.Total,
		"query_time_ms", results.QueryTime.Milliseconds())

	return response, nil
}

// handleSecureFindSimilar performs repository-scoped similarity search
func (ms *MemoryServer) handleSecureFindSimilar(ctx context.Context, params map[string]interface{}, repository string) (interface{}, error) {
	logging.Info("MCP TOOL: memory_secure_find_similar called", "params", params, "repository", repository)

	// Parse problem description
	problem, ok := params["problem"].(string)
	if !ok || problem == "" {
		return nil, errors.New("problem parameter is required for find_similar. Example: {\"problem\": \"authentication timeout error\", \"repository\": \"github.com/user/repo\"}")
	}

	// Use the secure search with problem as query
	searchParams := map[string]interface{}{
		"query":         problem,
		"limit":         5,   // Default for similarity search
		"min_relevance": 0.7, // Higher threshold for similarity
	}

	// Copy additional parameters
	if limit, ok := params["limit"]; ok {
		searchParams["limit"] = limit
	}
	if chunkType, ok := params["chunk_type"].(string); ok {
		searchParams["types"] = []string{chunkType}
	}

	return ms.handleSecureSearch(ctx, searchParams, repository)
}

// handleSecureSearchExplained performs repository-scoped search with explanations
func (ms *MemoryServer) handleSecureSearchExplained(ctx context.Context, params map[string]interface{}, repository string) (interface{}, error) {
	logging.Info("MCP TOOL: memory_secure_search_explained called", "params", params, "repository", repository)

	// Perform secure search first
	result, err := ms.handleSecureSearch(ctx, params, repository)
	if err != nil {
		return nil, err
	}

	// Add explanation to the result
	if resultMap, ok := result.(map[string]interface{}); ok {
		resultMap["explanation"] = map[string]interface{}{
			"search_strategy":   "Repository-scoped search with strict isolation",
			"fallback_disabled": "Progressive search fallback disabled for security",
			"repository_scope":  repository,
		}
		if repository == GlobalRepository {
			resultMap["explanation"].(map[string]interface{})["scope_note"] = "Global search enabled for cross-project architecture decisions"
		}
	}

	return result, nil
}

// Placeholder secure handlers for other operations
func (ms *MemoryServer) handleSecureGetPatterns(ctx context.Context, params map[string]interface{}, repository string) (interface{}, error) {
	// For now, delegate to existing handler but add repository logging
	logging.Info("Secure get patterns", "repository", repository)
	return ms.handleGetPatterns(ctx, params)
}

func (ms *MemoryServer) handleSecureGetRelationships(ctx context.Context, params map[string]interface{}, repository string) (interface{}, error) {
	logging.Info("Secure get relationships", "repository", repository)
	return ms.handleGetRelationships(ctx, params)
}

func (ms *MemoryServer) handleSecureTraverseGraph(ctx context.Context, params map[string]interface{}, repository string) (interface{}, error) {
	logging.Info("Secure traverse graph", "repository", repository)
	return ms.handleTraverseGraph(ctx, params)
}

func (ms *MemoryServer) handleSecureGetThreads(ctx context.Context, params map[string]interface{}, repository string) (interface{}, error) {
	logging.Info("Secure get threads", "repository", repository)
	return ms.handleGetThreads(ctx, params)
}

func (ms *MemoryServer) handleSecureResolveAlias(ctx context.Context, params map[string]interface{}, repository string) (interface{}, error) {
	logging.Info("Secure resolve alias", "repository", repository)
	return ms.handleResolveAlias(ctx, params)
}

func (ms *MemoryServer) handleSecureListAliases(ctx context.Context, params map[string]interface{}, repository string) (interface{}, error) {
	logging.Info("Secure list aliases", "repository", repository)
	return ms.handleListAliases(ctx, params)
}

// aliasParams holds parsed parameters for alias creation
type aliasParams struct {
	Name        string
	Type        string
	Description string
	Repository  string
	Target      interface{} // Can be string or complex object
	Metadata    map[string]interface{}
}

// parseAliasParams extracts and validates alias creation parameters
func (ms *MemoryServer) parseAliasParams(params map[string]interface{}) (*aliasParams, error) {
	// Required parameters
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return nil, errors.New("name parameter is required")
	}

	aliasType, ok := params["type"].(string)
	if !ok || aliasType == "" {
		return nil, errors.New("type parameter is required")
	}

	// Try string target first (simplified), then fall back to complex object
	var targetInterface interface{}
	switch target := params["target"].(type) {
	case string:
		targetInterface = target
	case map[string]interface{}:
		targetInterface = target
	default:
		return nil, errors.New("target parameter is required")
	}

	// Repository is required
	repository, ok := params["repository"].(string)
	if !ok {
		repository = ValueUnknown // Default value
	}

	result := &aliasParams{
		Name:       name,
		Type:       aliasType,
		Repository: repository,
		Target:     targetInterface,
	}

	// Optional parameters
	if desc, ok := params["description"].(string); ok {
		result.Description = desc
	}

	if metadataInterface, ok := params["metadata"].(map[string]interface{}); ok {
		result.Metadata = metadataInterface
	}

	return result, nil
}

// buildAliasFromParams constructs an Alias from parsed parameters
func (ms *MemoryServer) buildAliasFromParams(params *aliasParams) (bulk.Alias, error) {
	alias := bulk.Alias{
		Name:        params.Name,
		Type:        bulk.AliasType(params.Type),
		Description: params.Description,
	}

	// Parse target
	target, err := ms.parseAliasTarget(params.Target)
	if err != nil {
		return alias, err
	}
	alias.Target = target

	// Parse metadata if provided
	if params.Metadata != nil {
		alias.Metadata = ms.parseAliasMetadata(params.Metadata)
	}

	return alias, nil
}

// parseAliasTarget parses the target configuration for an alias
func (ms *MemoryServer) parseAliasTarget(targetInterface interface{}) (bulk.AliasTarget, error) {
	target := bulk.AliasTarget{}

	// Handle simple string target (simplified API)
	if targetStr, ok := targetInterface.(string); ok {
		target.Type = bulk.TargetTypeQuery
		target.Query = &bulk.QueryTarget{
			Query: targetStr,
		}
		return target, nil
	}

	// Handle complex object target (original API)
	targetMap, ok := targetInterface.(map[string]interface{})
	if !ok {
		return target, errors.New("target must be a string or object")
	}

	targetType, ok := targetMap["type"].(string)
	if !ok {
		return target, errors.New("missing or invalid target type")
	}

	target.Type = bulk.TargetType(targetType)

	switch target.Type {
	case bulk.TargetTypeChunks:
		target.ChunkIDs = ms.parseChunkIDs(targetMap)
		if len(target.ChunkIDs) == 0 {
			return target, errors.New("chunks target must specify at least one chunk_id")
		}
	case bulk.TargetTypeQuery:
		target.Query = ms.parseQueryTarget(targetMap)
		if target.Query == nil || target.Query.Query == "" {
			return target, errors.New("query target must specify a valid query")
		}
	case bulk.TargetTypeFilter:
		target.Filter = ms.parseFilterTarget(targetMap)
		if target.Filter == nil {
			return target, errors.New("filter target must specify valid filter criteria")
		}
	case bulk.TargetTypeCollection:
		target.Collection = ms.parseCollectionTarget(targetMap)
		if target.Collection == nil || target.Collection.Name == "" {
			return target, errors.New("collection target must specify a valid collection name")
		}
	default:
		return target, fmt.Errorf("unsupported target type: %s", targetType)
	}

	return target, nil
}

// parseChunkIDs extracts chunk IDs from target configuration
func (ms *MemoryServer) parseChunkIDs(targetInterface map[string]interface{}) []string {
	chunkIDs, ok := targetInterface["chunk_ids"].([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(chunkIDs))
	for _, id := range chunkIDs {
		if idStr, ok := id.(string); ok {
			result = append(result, idStr)
		}
	}
	return result
}

// parseQueryTarget parses query target configuration
func (ms *MemoryServer) parseQueryTarget(targetInterface map[string]interface{}) *bulk.QueryTarget {
	queryInterface, ok := targetInterface["query"].(map[string]interface{})
	if !ok {
		return nil
	}

	queryTarget := &bulk.QueryTarget{}
	if query, ok := queryInterface["query"].(string); ok {
		queryTarget.Query = query
	}
	if repo, ok := queryInterface["repository"].(string); ok {
		queryTarget.Repository = &repo
	}
	if maxResults, ok := queryInterface["max_results"].(float64); ok {
		queryTarget.MaxResults = int(maxResults)
	}
	return queryTarget
}

// parseFilterTarget parses filter target configuration
func (ms *MemoryServer) parseFilterTarget(targetInterface map[string]interface{}) *bulk.FilterTarget {
	filterInterface, ok := targetInterface["filter"].(map[string]interface{})
	if !ok {
		return nil
	}

	filterTarget := &bulk.FilterTarget{}
	if repo, ok := filterInterface["repository"].(string); ok {
		filterTarget.Repository = &repo
	}
	return filterTarget
}

// parseCollectionTarget parses collection target configuration
func (ms *MemoryServer) parseCollectionTarget(targetInterface map[string]interface{}) *bulk.CollectionTarget {
	collectionInterface, ok := targetInterface["collection"].(map[string]interface{})
	if !ok {
		return nil
	}

	collectionTarget := &bulk.CollectionTarget{}
	if collectionName, ok := collectionInterface["name"].(string); ok {
		collectionTarget.Name = collectionName
	}
	if desc, ok := collectionInterface["description"].(string); ok {
		collectionTarget.Description = desc
	}
	if chunkIDs, ok := collectionInterface["chunk_ids"].([]interface{}); ok {
		collectionTarget.ChunkIDs = make([]string, len(chunkIDs))
		for i, id := range chunkIDs {
			if idStr, ok := id.(string); ok {
				collectionTarget.ChunkIDs[i] = idStr
			}
		}
	}
	return collectionTarget
}

// parseAliasMetadata parses metadata configuration for an alias
func (ms *MemoryServer) parseAliasMetadata(metadataInterface map[string]interface{}) bulk.AliasMetadata {
	metadata := bulk.AliasMetadata{}
	if repo, ok := metadataInterface["repository"].(string); ok {
		metadata.Repository = repo
	}
	if tags, ok := metadataInterface["tags"].([]interface{}); ok {
		metadata.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				metadata.Tags[i] = tagStr
			}
		}
	}
	if isPublic, ok := metadataInterface["is_public"].(bool); ok {
		metadata.IsPublic = isPublic
	}
	if isFavorite, ok := metadataInterface["is_favorite"].(bool); ok {
		metadata.IsFavorite = isFavorite
	}
	return metadata
}

// buildAliasResponse constructs the response for alias creation
func (ms *MemoryServer) buildAliasResponse(createdAlias *bulk.Alias) map[string]interface{} {
	return map[string]interface{}{
		"id":          createdAlias.ID,
		"name":        createdAlias.Name,
		"type":        string(createdAlias.Type),
		"description": createdAlias.Description,
		"created_at":  createdAlias.CreatedAt.Format(time.RFC3339),
		"message":     fmt.Sprintf("Alias '%s' created successfully", createdAlias.Name),
	}
}

// handleCreateAlias creates a memory alias
func (ms *MemoryServer) handleCreateAlias(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_create_alias called", "params", params)

	// Parse and validate parameters
	aliasParams, err := ms.parseAliasParams(params)
	if err != nil {
		return nil, err
	}

	// Build alias from parameters
	alias, err := ms.buildAliasFromParams(aliasParams)
	if err != nil {
		return nil, err
	}

	// Create alias
	createdAlias, err := ms.aliasManager.CreateAlias(ctx, &alias)
	if err != nil {
		return nil, fmt.Errorf("failed to create alias: %w", err)
	}

	logging.Info("memory_create_alias completed successfully", "alias_id", createdAlias.ID, "name", createdAlias.Name)
	return ms.buildAliasResponse(createdAlias), nil
}

// handleResolveAlias resolves an alias reference
func (ms *MemoryServer) handleResolveAlias(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logging.Info("MCP TOOL: memory_resolve_alias called", "params", params)

	// Required parameter
	aliasName, ok := params["alias_name"].(string)
	if !ok || aliasName == "" {
		return nil, errors.New("alias_name parameter is required")
	}

	// Resolve alias
	result, err := ms.aliasManager.ResolveAlias(ctx, aliasName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve alias: %w", err)
	}

	// Format chunks for response
	chunks := make([]map[string]interface{}, len(result.Chunks))
	for i := range result.Chunks {
		chunks[i] = map[string]interface{}{
			"id":         result.Chunks[i].ID,
			"session_id": result.Chunks[i].SessionID,
			"timestamp":  result.Chunks[i].Timestamp.Format(time.RFC3339),
			"type":       string(result.Chunks[i].Type),
			"summary":    result.Chunks[i].Summary,
			"repository": result.Chunks[i].Metadata.Repository,
			"tags":       result.Chunks[i].Metadata.Tags,
			"outcome":    string(result.Chunks[i].Metadata.Outcome),
		}
	}

	logging.Info("memory_resolve_alias completed successfully", "alias_name", aliasName, "chunks_found", result.TotalFound)
	return map[string]interface{}{
		"alias": map[string]interface{}{
			"id":           result.Alias.ID,
			"name":         result.Alias.Name,
			"type":         string(result.Alias.Type),
			"description":  result.Alias.Description,
			"access_count": result.Alias.AccessCount,
		},
		"chunks":      chunks,
		"total_found": result.TotalFound,
		"message":     result.Message,
		"warnings":    result.Warnings,
	}, nil
}

// handleListAliases lists memory aliases
func (ms *MemoryServer) handleListAliases(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context parameter required for MCP handler interface consistency
	logging.Info("MCP TOOL: memory_list_aliases called", "params", params)

	// Build filter
	filter := bulk.AliasListFilter{
		SortBy: "usage",
		Limit:  20,
	}

	// Optional parameters
	if aliasType, ok := params["type"].(string); ok {
		at := bulk.AliasType(aliasType)
		filter.Type = &at
	}
	if repo, ok := params["repository"].(string); ok {
		filter.Repository = &repo
	}
	if tags, ok := params["tags"].([]interface{}); ok {
		filter.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				filter.Tags[i] = tagStr
			}
		}
	}
	if query, ok := params["query"].(string); ok {
		filter.Query = &query
	}
	if sortBy, ok := params["sort_by"].(string); ok {
		filter.SortBy = sortBy
	}
	if limit, ok := params["limit"].(float64); ok {
		filter.Limit = int(limit)
	}

	// List aliases
	aliases, err := ms.aliasManager.ListAliases(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list aliases: %w", err)
	}

	// Format response
	aliasesData := make([]map[string]interface{}, len(aliases))
	for i, alias := range aliases {
		aliasesData[i] = map[string]interface{}{
			"id":           alias.ID,
			"name":         alias.Name,
			"type":         string(alias.Type),
			"description":  alias.Description,
			"repository":   alias.Metadata.Repository,
			"tags":         alias.Metadata.Tags,
			"access_count": alias.AccessCount,
			"created_at":   alias.CreatedAt.Format(time.RFC3339),
			"updated_at":   alias.UpdatedAt.Format(time.RFC3339),
		}
		if alias.LastAccessed != nil {
			aliasesData[i]["last_accessed"] = alias.LastAccessed.Format(time.RFC3339)
		}
	}

	logging.Info("memory_list_aliases completed successfully", "aliases_found", len(aliases))
	return map[string]interface{}{
		"aliases":     aliasesData,
		"total_found": len(aliases),
		"filter_used": filter,
	}, nil
}

// handleGetBulkProgress gets bulk operation progress
func (ms *MemoryServer) handleGetBulkProgress(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	_ = ctx // Context parameter required for MCP handler interface consistency
	logging.Info("MCP TOOL: memory_get_bulk_progress called", "params", params)

	// Required parameter
	operationID, ok := params["operation_id"].(string)
	if !ok || operationID == "" {
		return nil, errors.New("operation_id parameter is required")
	}

	// Get progress
	progress, err := ms.bulkManager.GetProgress(operationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bulk progress: %w", err)
	}

	// Format errors for response
	bulkErrors := make([]map[string]interface{}, len(progress.Errors))
	for i, err := range progress.Errors {
		bulkErrors[i] = map[string]interface{}{
			"item_index": err.ItemIndex,
			"item_id":    err.ItemID,
			"error":      err.Error,
			"timestamp":  err.Timestamp.Format(time.RFC3339),
		}
	}

	logging.Info("memory_get_bulk_progress completed successfully", "operation_id", operationID, "status", progress.Status)
	return map[string]interface{}{
		"operation_id":      progress.OperationID,
		"status":            string(progress.Status),
		"total_items":       progress.TotalItems,
		"processed_items":   progress.ProcessedItems,
		"successful_items":  progress.SuccessfulItems,
		"failed_items":      progress.FailedItems,
		"skipped_items":     progress.SkippedItems,
		"current_batch":     progress.CurrentBatch,
		"total_batches":     progress.TotalBatches,
		"start_time":        progress.StartTime.Format(time.RFC3339),
		"elapsed_time":      progress.ElapsedTime.String(),
		"estimated_time":    progress.EstimatedTime.String(),
		"errors":            bulkErrors,
		"validation_errors": progress.ValidationErrors,
	}, nil
}

// Task Handler Implementations

// handleTodoWrite processes TodoWrite operations from Claude
func (ms *MemoryServer) handleTodoWrite(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// session_id is now OPTIONAL for repository-wide task persistence
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		sessionID = "repo-wide" // Default for repository-scoped tasks
	}

	repository, ok := args["repository"].(string)
	if !ok {
		return nil, errors.New("repository parameter is required for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"todos\": [...]} or {\"repository\": \"github.com/user/repo\", \"session_id\": \"my-session\", \"todos\": [...]}")
	}

	todosRaw, ok := args["todos"]
	if !ok {
		return nil, errors.New("todos parameter is required")
	}

	todosJSON, err := json.Marshal(todosRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal todos: %w", err)
	}

	var todos []workflow.TodoItem
	if err := json.Unmarshal(todosJSON, &todos); err != nil {
		return nil, fmt.Errorf("failed to unmarshal todos: %w", err)
	}

	if err := ms.todoTracker.ProcessTodoWrite(ctx, sessionID, repository, todos); err != nil {
		return nil, fmt.Errorf("failed to process todo write: %w", err)
	}

	return map[string]interface{}{
		StatusSuccess: true,
		"message":     fmt.Sprintf("Processed %d todos for repository %s (session: %s)", len(todos), repository, sessionID),
	}, nil
}

// handleTodoRead retrieves current todo state
// Supports two modes:
// 1. Session-specific: When session_id is provided, returns todos for that specific session
// 2. Repository-wide: When session_id is omitted or empty, returns all todos across all active sessions for the repository
func (ms *MemoryServer) handleTodoRead(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_ = ctx // Context unused but required by handler interface

	repository, ok := args["repository"].(string)
	if !ok {
		return nil, errors.New("repository parameter is required for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\"} or {\"repository\": \"github.com/user/repo\", \"session_id\": \"my-session\"}")
	}

	// session_id is now OPTIONAL - when not provided, return ALL todos for repository
	sessionID, hasSessionID := args["session_id"].(string)

	if hasSessionID && sessionID != "" {
		// Get todos for specific session
		session, exists := ms.todoTracker.GetActiveSession(sessionID, repository)
		if !exists {
			return map[string]interface{}{
				"todos":          []workflow.TodoItem{},
				"session_exists": false,
				"repository":     repository,
				"session_id":     sessionID,
				"scope":          "session",
			}, nil
		}

		return map[string]interface{}{
			"todos":          session.Todos,
			"session_id":     session.SessionID,
			"repository":     session.Repository,
			"start_time":     session.StartTime,
			"tools_used":     session.ToolsUsed,
			"files_changed":  session.FilesChanged,
			"session_exists": true,
			"scope":          "session",
		}, nil
	}

	// No session_id provided - return ALL todos for repository across all sessions
	repositorySessions := ms.todoTracker.GetActiveSessionsByRepository(repository)

	if len(repositorySessions) == 0 {
		return map[string]interface{}{
			"todos":                 []workflow.TodoItem{},
			"repository":            repository,
			"scope":                 "repository",
			"active_sessions_count": 0,
		}, nil
	}

	// Aggregate all todos from all sessions
	allTodos := make([]workflow.TodoItem, 0)
	sessionDetails := make([]map[string]interface{}, 0)

	for _, session := range repositorySessions {
		// Add session metadata to todos for context
		allTodos = append(allTodos, session.Todos...)

		sessionDetails = append(sessionDetails, map[string]interface{}{
			"session_id":    session.SessionID,
			"start_time":    session.StartTime,
			"todo_count":    len(session.Todos),
			"tools_used":    session.ToolsUsed,
			"files_changed": session.FilesChanged,
		})
	}

	return map[string]interface{}{
		"todos":                 allTodos,
		"repository":            repository,
		"scope":                 "repository",
		"active_sessions_count": len(repositorySessions),
		"session_details":       sessionDetails,
	}, nil
}

// handleTodoUpdate updates todo status and processes completion
func (ms *MemoryServer) handleTodoUpdate(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_ = ctx // Context unused but required by handler interface

	repository, ok := args["repository"].(string)
	if !ok {
		return nil, errors.New("repository parameter is required for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"tool_name\": \"Edit\"} or {\"repository\": \"github.com/user/repo\", \"session_id\": \"my-session\", \"tool_name\": \"Edit\"}")
	}

	// session_id is now OPTIONAL for repository-wide task updates
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		sessionID = "repo-wide" // Default for repository-scoped updates
	}

	toolName, ok := args["tool_name"].(string)
	if !ok {
		return nil, errors.New("tool_name parameter is required. Example: {\"tool_name\": \"Edit\", \"repository\": \"github.com/user/repo\"} or {\"tool_name\": \"Edit\", \"session_id\": \"my-session\", \"repository\": \"github.com/user/repo\"}")
	}

	toolContext, ok := args["tool_context"].(map[string]interface{})
	if !ok {
		toolContext = make(map[string]interface{})
	}

	ms.todoTracker.ProcessToolUsage(sessionID, repository, toolName, toolContext)

	return map[string]interface{}{
		StatusSuccess: true,
		"message":     fmt.Sprintf("Updated repository %s (session: %s) with tool usage: %s", repository, sessionID, toolName),
	}, nil
}

// handleSessionCreate creates a new workflow session
func (ms *MemoryServer) handleSessionCreate(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_ = ctx // Context unused but required by handler interface
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return nil, errors.New("session_id parameter is required for multi-tenant isolation. Example: {\"session_id\": \"my-session\", \"repository\": \"github.com/user/repo\"}")
	}

	repository, ok := args["repository"].(string)
	if !ok {
		return nil, errors.New("repository parameter is required for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"my-session\"}")
	}

	// Create session by accessing it (auto-created in TodoTracker)
	session := ms.todoTracker.GetOrCreateSession(sessionID, repository)

	return map[string]interface{}{
		StatusSuccess: true,
		"session_id":  session.SessionID,
		"repository":  session.Repository,
		"start_time":  session.StartTime,
		"message":     fmt.Sprintf("Session %s created for repository %s", sessionID, repository),
	}, nil
}

// handleSessionEnd ends a workflow session
func (ms *MemoryServer) handleSessionEnd(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_ = ctx // Context unused but required by handler interface
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return nil, errors.New("session_id parameter is required for multi-tenant isolation. Example: {\"session_id\": \"my-session\", \"repository\": \"github.com/user/repo\"}")
	}

	repository, ok := args["repository"].(string)
	if !ok {
		return nil, errors.New("repository parameter is required for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"my-session\"}")
	}

	outcomeStr, ok := args["outcome"].(string)
	if !ok {
		outcomeStr = StatusSuccess
	}

	var outcome types.Outcome
	switch outcomeStr {
	case StatusSuccess:
		outcome = types.OutcomeSuccess
	case "failed":
		outcome = types.OutcomeFailed
	case "in_progress":
		outcome = types.OutcomeInProgress
	default:
		outcome = types.OutcomeSuccess
	}

	ms.todoTracker.EndSession(sessionID, repository, outcome)

	return map[string]interface{}{
		StatusSuccess: true,
		"message":     fmt.Sprintf("Session %s in repository %s ended with outcome: %s", sessionID, repository, outcomeStr),
	}, nil
}

// handleSessionList lists active sessions filtered by repository
func (ms *MemoryServer) handleSessionList(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_ = ctx // Context unused but required by handler interface

	repository, ok := args["repository"].(string)
	if !ok {
		return nil, errors.New("repository parameter is required for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\"}")
	}

	// Get active sessions filtered by repository
	sessions := ms.todoTracker.GetActiveSessionsByRepository(repository)

	return map[string]interface{}{
		"sessions":   sessions,
		"count":      len(sessions),
		"repository": repository,
	}, nil
}

// handleWorkflowAnalyze analyzes workflow patterns
func (ms *MemoryServer) handleWorkflowAnalyze(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_ = ctx // Context unused but required by handler interface
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return nil, errors.New("session_id parameter is required for multi-tenant isolation. Example: {\"session_id\": \"my-session\", \"repository\": \"github.com/user/repo\"}")
	}

	repository, ok := args["repository"].(string)
	if !ok {
		return nil, errors.New("repository parameter is required for multi-tenant isolation. Example: {\"repository\": \"github.com/user/repo\", \"session_id\": \"my-session\"}")
	}

	session, exists := ms.todoTracker.GetActiveSession(sessionID, repository)
	if !exists {
		return nil, fmt.Errorf("session %s in repository %s not found", sessionID, repository)
	}

	// Basic workflow analysis
	completedTodos := 0
	totalTodos := len(session.Todos)
	for _, todo := range session.Todos {
		if todo.Status == "completed" {
			completedTodos++
		}
	}

	analysis := map[string]interface{}{
		"session_id":       sessionID,
		"repository":       session.Repository,
		"total_todos":      totalTodos,
		"completed_todos":  completedTodos,
		"completion_rate":  float64(completedTodos) / float64(totalTodos) * 100,
		"tools_used":       len(session.ToolsUsed),
		"files_changed":    len(session.FilesChanged),
		"session_duration": time.Since(session.StartTime).String(),
	}

	return analysis, nil
}

// handleTaskCompletionStats provides task completion statistics
func (ms *MemoryServer) handleTaskCompletionStats(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_ = ctx  // Context unused but required by handler interface
	_ = args // Args unused but required by handler interface
	// Get completed work from todoTracker
	completedWork := ms.todoTracker.GetCompletedWork()

	stats := map[string]interface{}{
		"total_completed_tasks": len(completedWork),
		"recent_completions":    len(completedWork), // Simplified for now
	}

	// Add repository breakdown if we have completed work
	if len(completedWork) > 0 {
		repoStats := make(map[string]int)
		for i := range completedWork {
			if completedWork[i].Metadata.Repository != "" {
				repoStats[completedWork[i].Metadata.Repository]++
			}
		}
		stats["by_repository"] = repoStats
	}

	return stats, nil
}

// handleGetDocumentation streams documentation files to clients
func (ms *MemoryServer) handleGetDocumentation(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_ = ctx // Context unused but required by handler interface
	docType, ok := args["doc_type"].(string)
	if !ok {
		docType = "both" // Default to both documents
	}

	var content strings.Builder

	switch docType {
	case "mappings":
		// Stream CONSOLIDATED_TOOL_MAPPINGS.md
		if data, err := os.ReadFile("docs/CONSOLIDATED_TOOL_MAPPINGS.md"); err == nil { // #nosec G304 -- Hardcoded safe path
			content.WriteString("# MCP Tool Parameter Mappings\n\n")
			content.WriteString(string(data))
		} else {
			return nil, fmt.Errorf("failed to read tool mappings documentation: %w", err)
		}
	case "examples":
		// Stream CONSOLIDATION_EXAMPLES.md
		if data, err := os.ReadFile("docs/CONSOLIDATION_EXAMPLES.md"); err == nil { // #nosec G304 -- Hardcoded safe path
			content.WriteString("# MCP Tool Usage Examples\n\n")
			content.WriteString(string(data))
		} else {
			return nil, fmt.Errorf("failed to read examples documentation: %w", err)
		}
	case "both", "":
		// Stream both files with clear separation
		content.WriteString("# MCP Memory Server Documentation\n\n")

		content.WriteString("## Tool Parameter Mappings\n\n")
		if data, err := os.ReadFile("docs/CONSOLIDATED_TOOL_MAPPINGS.md"); err == nil { // #nosec G304 -- Hardcoded safe path
			content.WriteString(string(data))
		} else {
			content.WriteString("*Tool mappings documentation not available*\n")
		}

		content.WriteString("\n\n---\n\n")

		content.WriteString("## Usage Examples\n\n")
		if data, err := os.ReadFile("docs/CONSOLIDATION_EXAMPLES.md"); err == nil { // #nosec G304 -- Hardcoded safe path
			content.WriteString(string(data))
		} else {
			content.WriteString("*Usage examples documentation not available*\n")
		}
	default:
		return nil, fmt.Errorf("unsupported doc_type: %s (use 'mappings', 'examples', or 'both')", docType)
	}

	return map[string]interface{}{
		"doc_type":  docType,
		"content":   content.String(),
		"length":    content.Len(),
		"timestamp": time.Now().Format(time.RFC3339),
	}, nil
}

// handleDeleteExpired deletes expired chunks from a repository
func (ms *MemoryServer) handleDeleteExpired(ctx context.Context, options map[string]interface{}, repository string) (interface{}, error) {
	logging.Info("MCP TOOL: handleDeleteExpired called", "repository", repository, "options", options)

	// Parse expiration criteria
	maxAge := "30d" // Default: delete chunks older than 30 days
	if age, exists := options["max_age"].(string); exists {
		maxAge = age
	}

	cutoffTime, err := parseMaxAge(maxAge)
	if err != nil {
		return nil, fmt.Errorf("invalid max_age format: %s. Use formats like '30d', '7d', '24h', '1h'", maxAge)
	}

	// Get all chunks for the repository
	chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repository, 10000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve repository chunks: %w", err)
	}

	// Find expired chunks
	var expiredIDs []string
	for i := range chunks {
		if chunks[i].Timestamp.Before(cutoffTime) {
			expiredIDs = append(expiredIDs, chunks[i].ID)
		}
	}

	if len(expiredIDs) == 0 {
		return map[string]interface{}{
			"repository":    repository,
			"cutoff_time":   cutoffTime.Format(time.RFC3339),
			"max_age":       maxAge,
			"deleted_count": 0,
			"total_chunks":  len(chunks),
			"message":       "No expired chunks found",
		}, nil
	}

	// Delete expired chunks using existing secure bulk delete
	deleteResult, err := ms.handleSecureBulkDelete(ctx, map[string]interface{}{
		"ids": expiredIDs,
	}, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to delete expired chunks: %w", err)
	}

	// Extract delete count from result
	deleteCount := 0
	if result, ok := deleteResult.(map[string]interface{}); ok {
		if count, ok := result["deleted_count"].(int); ok {
			deleteCount = count
		}
	}

	return map[string]interface{}{
		"repository":    repository,
		"cutoff_time":   cutoffTime.Format(time.RFC3339),
		"max_age":       maxAge,
		"deleted_count": deleteCount,
		"total_chunks":  len(chunks),
		"expired_found": len(expiredIDs),
		"message":       fmt.Sprintf("Successfully deleted %d expired chunks (older than %s)", deleteCount, maxAge),
	}, nil
}

// parseMaxAge parses duration strings like "30d", "7d", "24h", "1h"
func parseMaxAge(maxAge string) (time.Time, error) {
	now := time.Now()

	if strings.HasSuffix(maxAge, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(maxAge, "d"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid days format: %s", maxAge)
		}
		return now.AddDate(0, 0, -days), nil
	}

	if strings.HasSuffix(maxAge, "h") {
		hours, err := strconv.Atoi(strings.TrimSuffix(maxAge, "h"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid hours format: %s", maxAge)
		}
		return now.Add(-time.Duration(hours) * time.Hour), nil
	}

	if strings.HasSuffix(maxAge, "m") {
		minutes, err := strconv.Atoi(strings.TrimSuffix(maxAge, "m"))
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid minutes format: %s", maxAge)
		}
		return now.Add(-time.Duration(minutes) * time.Minute), nil
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s. Use formats like '30d', '7d', '24h', '1h'", maxAge)
}

// discoverRepositories discovers available repositories from the vector store
func (ms *MemoryServer) discoverRepositories(ctx context.Context) []string {
	repoSet := make(map[string]bool)

	// Get chunks from global and try to discover repositories
	globalChunks, err := ms.container.GetVectorStore().ListByRepository(ctx, "_global", 1000, 0)
	if err == nil {
		for i := range globalChunks {
			if globalChunks[i].Metadata.Repository != "" && globalChunks[i].Metadata.Repository != "_global" {
				repoSet[globalChunks[i].Metadata.Repository] = true
			}
		}
	}

	// Also try common repository patterns to discover more repos
	commonRepos := []string{
		"github.com/LerianStudio/lerian-mcp-memory",
		"github.com/lerianstudio/midaz",
		"github.com/LerianStudio/lib-commons",
	}
	for _, repo := range commonRepos {
		chunks, err := ms.container.GetVectorStore().ListByRepository(ctx, repo, 10, 0)
		if err == nil && len(chunks) > 0 {
			repoSet[repo] = true
		}
	}

	repositories := make([]string, 0, len(repoSet))
	for repo := range repoSet {
		repositories = append(repositories, repo)
	}
	return repositories
}

// AI-friendly error helpers are implemented inline for better MCP client guidance
