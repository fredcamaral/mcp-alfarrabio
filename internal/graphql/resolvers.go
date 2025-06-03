package graphql

import (
	"fmt"
	"mcp-memory/internal/di"
	"mcp-memory/internal/workflow"
	"mcp-memory/pkg/types"
	"time"

	"github.com/google/uuid"
	"github.com/graphql-go/graphql"
)

// searchResolver handles memory search queries
func (s *Schema) searchResolver(container *di.Container) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		input := p.Args["input"].(map[string]interface{})

		// Build query
		repo := getStringOrDefault(input, "repository", "")
		var repoPtr *string
		if repo != "" {
			repoPtr = &repo
		}

		// Convert string types to ChunkType
		typeStrings := getStringArrayOrDefault(input, "types")
		chunkTypes := make([]types.ChunkType, len(typeStrings))
		for i, t := range typeStrings {
			chunkTypes[i] = types.ChunkType(t)
		}

		query := types.MemoryQuery{
			Query:             getStringOrDefault(input, "query", ""),
			Repository:        repoPtr,
			Types:             chunkTypes,
			Limit:             getIntOrDefault(input, "limit", 10),
			MinRelevanceScore: getFloatOrDefault(input, "minRelevanceScore", 0.7),
			Recency:           types.RecencyRecent, // Default to recent
		}

		// Handle recency
		if recency := getStringOrDefault(input, "recency", "recent"); recency != "" {
			switch recency {
			case "all_time":
				query.Recency = types.RecencyAllTime
			case "last_month":
				query.Recency = types.RecencyLastMonth
			default:
				query.Recency = types.RecencyRecent
			}
		}

		// Generate embeddings for query
		embeddings, err := container.GetEmbeddingService().GenerateEmbedding(ctx, query.Query)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings: %w", err)
		}

		// Perform search
		results, err := container.GetVectorStore().Search(ctx, query, embeddings)
		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}

		return results, nil
	}
}

// getChunkResolver handles getting a chunk by ID
func (s *Schema) getChunkResolver(container *di.Container) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		id := p.Args["id"].(string)

		chunk, err := container.GetVectorStore().GetByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get chunk: %w", err)
		}

		return chunk, nil
	}
}

// listChunksResolver handles listing chunks by repository
func (s *Schema) listChunksResolver(container *di.Container) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		repository := p.Args["repository"].(string)
		limit := getIntOrDefault(p.Args, "limit", 100)
		offset := getIntOrDefault(p.Args, "offset", 0)

		chunks, err := container.GetVectorStore().ListByRepository(ctx, repository, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to list chunks: %w", err)
		}

		return chunks, nil
	}
}

// getPatternsResolver handles pattern detection queries
func (s *Schema) getPatternsResolver(container *di.Container) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		repository := p.Args["repository"].(string)
		timeframe := getStringOrDefault(p.Args, "timeframe", "month")

		// Pattern functionality is accessed through container

		// Get patterns
		recency := parseTimeframeToRecency(timeframe)
		query := types.MemoryQuery{
			Repository: &repository,
			Recency:    recency,
			Limit:      1000, // Get more data for pattern analysis
		}

		// Search for chunks in timeframe
		ctx := p.Context
		embeddings := make([]float64, 1536) // Empty embeddings for broad search
		results, err := container.GetVectorStore().Search(ctx, query, embeddings)
		if err != nil {
			return nil, fmt.Errorf("failed to search for patterns: %w", err)
		}

		// Extract patterns from results
		// analyzer := container.GetPatternAnalyzer() // Not used for now
		patterns := []map[string]interface{}{}

		// Analyze patterns in chunks
		for _, scored := range results.Results {
			// This is a simplified pattern extraction
			// In a real implementation, we'd use the pattern analyzer more thoroughly
			pattern := map[string]interface{}{
				"name":        scored.Chunk.Type,
				"description": fmt.Sprintf("Pattern found in %s", scored.Chunk.Type),
				"occurrences": 1,
				"confidence":  scored.Score,
				"lastSeen":    scored.Chunk.Timestamp,
				"examples":    []string{scored.Chunk.Summary},
			}
			patterns = append(patterns, pattern)
		}

		return patterns, nil
	}
}

// suggestRelatedResolver handles context suggestion queries
func (s *Schema) suggestRelatedResolver(container *di.Container) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		currentContext := p.Args["currentContext"].(string)
		sessionID := p.Args["sessionId"].(string)
		repository := getStringOrDefault(p.Args, "repository", "")
		// includePatterns := getBoolOrDefault(p.Args, "includePatterns", true) // Not used yet
		_ = repository
		maxSuggestions := getIntOrDefault(p.Args, "maxSuggestions", 5)

		// Create temporary chunk for context
		// chunk := types.ConversationChunk{
		// 	Content:   currentContext,
		// 	SessionID: sessionID,
		// 	Timestamp: time.Now(),
		// 	Type:      types.ChunkTypeDiscussion,
		// 	Metadata: types.ChunkMetadata{
		// 		Repository: repository,
		// 	},
		// } // Not used yet
		_ = currentContext
		_ = sessionID

		// Get suggestions
		suggester := container.GetContextSuggester()

		// Analyze context to get suggestions
		suggestions, err := suggester.AnalyzeContext(
			ctx,
			sessionID,
			repository,
			currentContext,
			"",                         // toolUsed - not provided in GraphQL
			types.ConversationFlow(""), // currentFlow - not provided
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get suggestions: %w", err)
		}

		// Limit suggestions
		if maxSuggestions < len(suggestions) {
			suggestions = suggestions[:maxSuggestions]
		}

		// Convert to GraphQL format
		relevantChunks := []interface{}{}
		suggestedTasks := []interface{}{}
		relatedConcepts := []interface{}{}
		potentialIssues := []interface{}{}

		for _, suggestion := range suggestions {
			// Convert related chunks
			for _, chunk := range suggestion.RelatedChunks {
				relevantChunks = append(relevantChunks, map[string]interface{}{
					"id":         chunk.ID,
					"content":    chunk.Content,
					"summary":    chunk.Summary,
					"type":       string(chunk.Type),
					"repository": chunk.Metadata.Repository,
					"timestamp":  chunk.Timestamp,
				})
			}

			// Categorize suggestions
			switch suggestion.Type {
			case workflow.SuggestionTypeDuplicateWork, workflow.SuggestionTypeTechnicalDebt:
				potentialIssues = append(potentialIssues, map[string]interface{}{
					"type":        string(suggestion.Type),
					"title":       suggestion.Title,
					"description": suggestion.Description,
					"relevance":   suggestion.Relevance,
				})
			case workflow.SuggestionTypeArchitectural, workflow.SuggestionTypePastDecision,
				workflow.SuggestionTypeSimilarProblem, workflow.SuggestionTypeSuccessfulPattern,
				workflow.SuggestionTypeOptimization, workflow.SuggestionTypeFlowBased,
				workflow.SuggestionTypeDebuggingContext, workflow.SuggestionTypeImplementContext:
				relatedConcepts = append(relatedConcepts, map[string]interface{}{
					"type":        string(suggestion.Type),
					"title":       suggestion.Title,
					"description": suggestion.Description,
					"relevance":   suggestion.Relevance,
				})
			default:
				// Convert action type to task suggestion
				if suggestion.ActionType == workflow.ActionImplement || suggestion.ActionType == workflow.ActionOptimize {
					suggestedTasks = append(suggestedTasks, map[string]interface{}{
						"action":      string(suggestion.ActionType),
						"title":       suggestion.Title,
						"description": suggestion.Description,
						"relevance":   suggestion.Relevance,
					})
				}
			}
		}

		result := map[string]interface{}{
			"relevantChunks":  relevantChunks,
			"suggestedTasks":  suggestedTasks,
			"relatedConcepts": relatedConcepts,
			"potentialIssues": potentialIssues,
		}

		return result, nil
	}
}

// findSimilarResolver handles finding similar problems
func (s *Schema) findSimilarResolver(container *di.Container) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		problem := p.Args["problem"].(string)
		repository := getStringOrDefault(p.Args, "repository", "")
		limit := getIntOrDefault(p.Args, "limit", 5)

		// Generate embeddings for problem
		ctx := p.Context
		embeddings, err := container.GetEmbeddingService().GenerateEmbedding(ctx, problem)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings: %w", err)
		}

		// Search for similar problems
		var repoPtr *string
		if repository != "" {
			repoPtr = &repository
		}

		query := types.MemoryQuery{
			Query:             problem,
			Repository:        repoPtr,
			Types:             []types.ChunkType{types.ChunkTypeProblem},
			Limit:             limit,
			MinRelevanceScore: 0.7,
			Recency:           types.RecencyAllTime,
		}

		results, err := container.GetVectorStore().Search(ctx, query, embeddings)
		if err != nil {
			return nil, fmt.Errorf("search failed: %w", err)
		}

		// Extract chunks
		chunks := make([]types.ConversationChunk, 0, len(results.Results))
		for _, scored := range results.Results {
			chunks = append(chunks, scored.Chunk)
		}

		return chunks, nil
	}
}

// storeChunkResolver handles storing conversation chunks
func (s *Schema) storeChunkResolver(container *di.Container) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		input := p.Args["input"].(map[string]interface{})

		// Create chunk
		chunk := types.ConversationChunk{
			ID:        uuid.New().String(),
			Content:   input["content"].(string),
			SessionID: input["sessionId"].(string),
			Timestamp: time.Now(),
			Type:      types.ChunkTypeArchitectureDecision,
			Metadata: types.ChunkMetadata{
				Repository:    getStringOrDefault(input, "repository", "_global"),
				Branch:        getStringOrDefault(input, "branch", ""),
				Tags:          getStringArrayOrDefault(input, "tags"),
				ToolsUsed:     getStringArrayOrDefault(input, "toolsUsed"),
				FilesModified: getStringArrayOrDefault(input, "filesModified"),
				Outcome:       types.OutcomeSuccess,   // Default to success
				Difficulty:    types.DifficultySimple, // Default to simple
			},
		}

		// Generate embeddings
		embeddings, err := container.GetEmbeddingService().GenerateEmbedding(ctx, chunk.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings: %w", err)
		}
		chunk.Embeddings = embeddings

		// Process with chunking service
		chunkingService := container.GetChunkingService()
		processedChunks, err := chunkingService.ProcessConversation(ctx, chunk.SessionID, chunk.Content, &chunk.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to process conversation: %w", err)
		}

		// Store all chunks
		for _, processedChunk := range processedChunks {
			if err := container.GetVectorStore().Store(ctx, processedChunk); err != nil {
				return nil, fmt.Errorf("failed to store chunk: %w", err)
			}
		}

		// Return the first chunk (primary chunk)
		if len(processedChunks) > 0 {
			return &processedChunks[0], nil
		}

		return nil, fmt.Errorf("no chunks processed")
	}
}

// storeDecisionResolver handles storing architectural decisions
func (s *Schema) storeDecisionResolver(container *di.Container) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		input := p.Args["input"].(map[string]interface{})

		// Create decision chunk
		chunk := types.ConversationChunk{
			ID:        uuid.New().String(),
			Content:   input["decision"].(string),
			SessionID: input["sessionId"].(string),
			Timestamp: time.Now(),
			Type:      types.ChunkTypeArchitectureDecision,
			Summary:   fmt.Sprintf("Decision: %s", input["decision"].(string)),
			Metadata: types.ChunkMetadata{
				Repository: getStringOrDefault(input, "repository", "_global"),
			},
		}

		// Add context if provided
		if context, ok := input["context"].(string); ok && context != "" {
			chunk.Content = fmt.Sprintf("%s\n\nContext: %s", chunk.Content, context)
		}

		// Generate embeddings
		embeddings, err := container.GetEmbeddingService().GenerateEmbedding(ctx, chunk.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embeddings: %w", err)
		}
		chunk.Embeddings = embeddings

		// Store the decision
		if err := container.GetVectorStore().Store(ctx, chunk); err != nil {
			return nil, fmt.Errorf("failed to store decision: %w", err)
		}

		return &chunk, nil
	}
}

// deleteChunkResolver handles deleting chunks
func (s *Schema) deleteChunkResolver(container *di.Container) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		id := p.Args["id"].(string)

		if err := container.GetVectorStore().Delete(ctx, id); err != nil {
			return false, fmt.Errorf("failed to delete chunk: %w", err)
		}

		return true, nil
	}
}

// Helper functions for extracting values with defaults
func getStringOrDefault(m interface{}, key string, defaultValue string) string {
	if mapValue, ok := m.(map[string]interface{}); ok {
		if value, exists := mapValue[key]; exists && value != nil {
			if strValue, ok := value.(string); ok {
				return strValue
			}
		}
	}
	return defaultValue
}

func getIntOrDefault(m interface{}, key string, defaultValue int) int {
	if mapValue, ok := m.(map[string]interface{}); ok {
		if value, exists := mapValue[key]; exists && value != nil {
			if intValue, ok := value.(int); ok {
				return intValue
			}
		}
	}
	return defaultValue
}

func getFloatOrDefault(m interface{}, key string, defaultValue float64) float64 {
	if mapValue, ok := m.(map[string]interface{}); ok {
		if value, exists := mapValue[key]; exists && value != nil {
			if floatValue, ok := value.(float64); ok {
				return floatValue
			}
		}
	}
	return defaultValue
}

func getStringArrayOrDefault(m interface{}, key string) []string {
	mapValue, ok := m.(map[string]interface{})
	if !ok {
		return nil
	}

	value, exists := mapValue[key]
	if !exists || value == nil {
		return nil
	}

	arr, ok := value.([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if str, ok := v.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// parseTimeframeToRecency converts a timeframe string to Recency
func parseTimeframeToRecency(timeframe string) types.Recency {
	switch timeframe {
	case "week", "recent":
		return types.RecencyRecent
	case "month":
		return types.RecencyLastMonth
	case "quarter", "all":
		return types.RecencyAllTime
	default:
		return types.RecencyRecent
	}
}

// traceSessionResolver traces all memories in a session
func (s *Schema) traceSessionResolver(container *di.Container) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		sessionID := p.Args["sessionId"].(string)

		// Get all chunks for this session
		chunks, err := container.GetVectorStore().ListBySession(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to trace session: %w", err)
		}

		// Sort by timestamp
		// Chunks should already be sorted by the store, but let's ensure it
		return chunks, nil
	}
}

// traceRelatedResolver traces related memories using similarity search
func (s *Schema) traceRelatedResolver(container *di.Container) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		ctx := p.Context
		chunkID := p.Args["chunkId"].(string)
		depth := getIntOrDefault(p.Args, "depth", 2)

		// Get the starting chunk
		startChunk, err := container.GetVectorStore().GetByID(ctx, chunkID)
		if err != nil {
			return nil, fmt.Errorf("failed to get chunk: %w", err)
		}

		// Build a map to track visited chunks
		visited := make(map[string]bool)
		visited[chunkID] = true

		// Results to return
		relatedChunks := []types.ConversationChunk{*startChunk}

		// BFS to find related chunks up to depth
		currentLevel := []types.ConversationChunk{*startChunk}

		for level := 1; level <= depth && len(currentLevel) > 0; level++ {
			nextLevel := []types.ConversationChunk{}

			for _, chunk := range currentLevel {
				// Search for similar chunks using content
				query := types.MemoryQuery{
					Query:             chunk.Content,
					Limit:             5,
					MinRelevanceScore: 0.7,
					Recency:           types.RecencyAllTime,
				}

				// Use existing embeddings if available, otherwise generate new ones
				var embeddings []float64
				if len(chunk.Embeddings) > 0 {
					embeddings = chunk.Embeddings
				} else {
					embeddings, err = container.GetEmbeddingService().GenerateEmbedding(ctx, chunk.Content)
					if err != nil {
						continue // Skip this chunk if we can't generate embeddings
					}
				}

				results, err := container.GetVectorStore().Search(ctx, query, embeddings)
				if err != nil {
					continue // Skip this chunk if search fails
				}

				// Add unvisited related chunks
				for _, scored := range results.Results {
					if !visited[scored.Chunk.ID] && scored.Score > 0.7 {
						visited[scored.Chunk.ID] = true
						relatedChunks = append(relatedChunks, scored.Chunk)
						nextLevel = append(nextLevel, scored.Chunk)
					}
				}
			}

			currentLevel = nextLevel
		}

		return relatedChunks, nil
	}
}
