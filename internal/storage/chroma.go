package storage

import (
	"context"
	"fmt"
	"mcp-memory/internal/config"
	"mcp-memory/internal/logging"
	"mcp-memory/pkg/types"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"mcp-memory/internal/chromasimple"
)

// Constants
const (
	connectionStatusError = "error"
)

// ChromaStore implements VectorStore interface for Chroma vector database
type ChromaStore struct {
	client     chromasimple.Client
	collection chromasimple.Collection
	config     *config.ChromaConfig
	metrics    *StorageMetrics
}

// NewChromaStore creates a new Chroma vector store
func NewChromaStore(cfg *config.ChromaConfig) *ChromaStore {
	return &ChromaStore{
		config: cfg,
		metrics: &StorageMetrics{
			OperationCounts:  make(map[string]int64),
			AverageLatency:   make(map[string]float64),
			ErrorCounts:      make(map[string]int64),
			ConnectionStatus: "unknown",
		},
	}
}

// Initialize creates the collection if it doesn't exist
func (cs *ChromaStore) Initialize(ctx context.Context) error {
	start := time.Now()
	defer cs.updateMetrics("initialize", start, nil)

	// Create Chroma client
	client := chromasimple.NewHTTPClient(cs.config.Endpoint)
	cs.client = client

	// Create or get collection
	collectionMetadata := map[string]interface{}{
		"description": "Claude conversation memory storage",
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}

	collection, err := cs.client.GetOrCreateCollection(
		ctx,
		cs.config.Collection,
		collectionMetadata,
	)
	if err != nil {
		cs.metrics.ConnectionStatus = connectionStatusError
		return fmt.Errorf("failed to create/get collection: %w", err)
	}

	cs.collection = collection
	cs.metrics.ConnectionStatus = "connected"
	logging.Info("Chroma collection initialized", "collection", cs.config.Collection)
	return nil
}

// Store saves a conversation chunk to Chroma
func (cs *ChromaStore) Store(ctx context.Context, chunk types.ConversationChunk) error {
	start := time.Now()
	defer cs.updateMetrics("store", start, nil)

	if err := chunk.Validate(); err != nil {
		return fmt.Errorf("invalid chunk: %w", err)
	}

	if len(chunk.Embeddings) == 0 {
		return fmt.Errorf("chunk must have embeddings before storing")
	}

	// Convert chunk to Chroma format
	document := cs.chunkToDocument(chunk)
	metadata := cs.chunkToMetadata(chunk)

	// Add document to collection
	err := cs.collection.Add(
		ctx,
		[]string{chunk.ID},
		[][]float64{chunk.Embeddings},
		[]string{document},
		[]map[string]interface{}{metadata},
	)

	if err != nil {
		cs.updateMetrics("store", start, err)
		return fmt.Errorf("failed to store chunk: %w", err)
	}

	return nil
}

// Search finds similar chunks based on query embeddings
func (cs *ChromaStore) Search(ctx context.Context, query types.MemoryQuery, queryEmbeddings []float64) (*types.SearchResults, error) {
	start := time.Now()
	defer cs.updateMetrics("search", start, nil)

	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	if len(queryEmbeddings) == 0 {
		return nil, fmt.Errorf("embeddings required for search")
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	// Build where filters
	whereFilter := cs.buildWhereFilter(query)

	// Query the collection
	qr, err := cs.collection.Query(
		ctx,
		[][]float64{queryEmbeddings},
		limit,
		whereFilter,
		[]string{"documents", "metadatas", "distances"},
	)
	if err != nil {
		cs.updateMetrics("search", start, err)
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Convert response to search results
	results := &types.SearchResults{
		Results:   []types.SearchResult{},
		Total:     0,
		QueryTime: time.Since(start),
	}

	// Process results
	if qr == nil {
		return results, nil
	}

	return cs.processQueryResults(qr, &query, results), nil
}

func (cs *ChromaStore) processQueryResults(qr *chromasimple.QueryResult, query *types.MemoryQuery, results *types.SearchResults) *types.SearchResults {
	if qr == nil || len(qr.IDs) == 0 || len(qr.IDs[0]) == 0 {
		return results
	}

	results.Total = len(qr.IDs[0])

	// Ensure all result groups have data
	if len(qr.Documents) == 0 || len(qr.Metadatas) == 0 || len(qr.Distances) == 0 {
		return results
	}

	for i := 0; i < len(qr.IDs[0]); i++ {
		result := cs.processSearchResult(i, qr.IDs[0], qr.Documents[0], qr.Metadatas[0], qr.Distances[0], query.MinRelevanceScore)
		if result != nil {
			results.Results = append(results.Results, *result)
		}
	}

	return results
}

func (cs *ChromaStore) processSearchResult(index int, ids []string, docs []string, metadatas []map[string]interface{}, distances []float32, minScore float64) *types.SearchResult {
	chunkID := cs.extractChunkID(index, ids)
	metadata := cs.extractMetadata(index, metadatas)
	content := cs.extractContent(index, docs)

	chunk := cs.chromaResultToChunk(chunkID, content, metadata)

	score := cs.calculateScore(index, distances)
	if score < minScore {
		return nil
	}

	return &types.SearchResult{
		Chunk: *chunk,
		Score: score,
	}
}

func (cs *ChromaStore) extractChunkID(index int, ids []string) string {
	if len(ids) > index {
		return ids[index]
	}
	return ""
}

func (cs *ChromaStore) extractMetadata(index int, metadatas []map[string]interface{}) map[string]interface{} {
	if len(metadatas) > index {
		return metadatas[index]
	}
	return nil
}

func (cs *ChromaStore) extractContent(index int, docs []string) string {
	if len(docs) > index {
		return docs[index]
	}
	return ""
}

func (cs *ChromaStore) calculateScore(index int, distances []float32) float64 {
	if len(distances) > index {
		return 1.0 - float64(distances[index])
	}
	return 1.0
}

// These methods are no longer needed as we work with native Go types

// GetByID retrieves a chunk by its ID
func (cs *ChromaStore) GetByID(ctx context.Context, id string) (*types.ConversationChunk, error) {
	start := time.Now()
	defer cs.updateMetrics("get_by_id", start, nil)

	result, err := cs.collection.Get(
		ctx,
		[]string{id},
		nil,
		[]string{"documents", "metadatas"},
	)

	if err != nil {
		cs.updateMetrics("get_by_id", start, err)
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	if result == nil || len(result.Documents) == 0 {
		return nil, fmt.Errorf("chunk not found: %s", id)
	}

	if len(result.IDs) > 0 && len(result.Documents) > 0 {
		var metadata map[string]interface{}
		if len(result.Metadatas) > 0 {
			metadata = result.Metadatas[0]
		}
		return cs.chromaResultToChunk(result.IDs[0], result.Documents[0], metadata), nil
	}

	return nil, fmt.Errorf("chunk not found: %s", id)
}

// ListByRepository lists chunks for a specific repository
func (cs *ChromaStore) ListByRepository(ctx context.Context, repository string, limit int, offset int) ([]types.ConversationChunk, error) {
	start := time.Now()
	defer cs.updateMetrics("list_by_repository", start, nil)

	if limit <= 0 {
		limit = 10
	}

	// Build where filter for repository
	whereFilter := map[string]interface{}{
		"repository": repository,
	}

	// Note: our simple client doesn't support limit/offset, would need to add this
	result, err := cs.collection.Get(
		ctx,
		nil, // ids - nil means get all matching where clause
		whereFilter,
		[]string{"documents", "metadatas"},
	)
	if err != nil {
		cs.updateMetrics("list_by_repository", start, err)
		return nil, fmt.Errorf("failed to list chunks: %w", err)
	}

	chunks := make([]types.ConversationChunk, 0)
	if result != nil {
		for i := 0; i < len(result.IDs); i++ {
			var chunkID string
			if i < len(result.IDs) {
				chunkID = result.IDs[i]
			}

			var metadata map[string]interface{}
			if i < len(result.Metadatas) {
				metadata = result.Metadatas[i]
			}

			var content string
			if i < len(result.Documents) {
				content = result.Documents[i]
			}

			chunk := cs.chromaResultToChunk(chunkID, content, metadata)
			chunks = append(chunks, *chunk)
		}
		
		// Apply limit and offset manually since our simple client doesn't support it
		if offset > 0 && offset < len(chunks) {
			chunks = chunks[offset:]
		} else if offset >= len(chunks) {
			chunks = []types.ConversationChunk{}
		}
		
		if limit > 0 && len(chunks) > limit {
			chunks = chunks[:limit]
		}
	}

	return chunks, nil
}

// ListBySession lists chunks for a specific session ID
func (cs *ChromaStore) ListBySession(ctx context.Context, sessionID string) ([]types.ConversationChunk, error) {
	start := time.Now()
	defer cs.updateMetrics("list_by_session", start, nil)

	// Build where filter for session
	whereFilter := map[string]interface{}{
		"session_id": sessionID,
	}

	result, err := cs.collection.Get(
		ctx,
		nil, // ids - nil means get all matching where clause
		whereFilter,
		[]string{"documents", "metadatas"},
	)
	if err != nil {
		cs.updateMetrics("list_by_session", start, err)
		return nil, fmt.Errorf("failed to list chunks by session: %w", err)
	}

	chunks := make([]types.ConversationChunk, 0)
	if result != nil {
		for i := 0; i < len(result.IDs); i++ {
			var chunkID string
			if i < len(result.IDs) {
				chunkID = result.IDs[i]
			}

			var content string
			if i < len(result.Documents) {
				content = result.Documents[i]
			}

			var metadata map[string]interface{}
			if i < len(result.Metadatas) {
				metadata = result.Metadatas[i]
			}

			chunk := cs.chromaResultToChunk(chunkID, content, metadata)
			chunks = append(chunks, *chunk)
		}
	}

	// Sort by timestamp
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Timestamp.Before(chunks[j].Timestamp)
	})

	return chunks, nil
}

// Delete removes a chunk by ID
func (cs *ChromaStore) Delete(ctx context.Context, id string) error {
	start := time.Now()
	defer cs.updateMetrics("delete", start, nil)

	err := cs.collection.Delete(ctx, []string{id})
	if err != nil {
		cs.updateMetrics("delete", start, err)
		return fmt.Errorf("failed to delete chunk: %w", err)
	}

	return nil
}

// Update modifies an existing chunk
func (cs *ChromaStore) Update(ctx context.Context, chunk types.ConversationChunk) error {
	// Chroma doesn't have a direct update, so we delete and re-add
	if err := cs.Delete(ctx, chunk.ID); err != nil {
		return err
	}
	return cs.Store(ctx, chunk)
}

// HealthCheck verifies the connection to Chroma
func (cs *ChromaStore) HealthCheck(ctx context.Context) error {
	start := time.Now()
	defer cs.updateMetrics("health_check", start, nil)

	// Check if collection is initialized
	if cs.collection == nil {
		err := fmt.Errorf("collection not initialized - call Initialize() first")
		cs.updateMetrics("health_check", start, err)
		return err
	}

	// Try to count documents as a health check
	count, err := cs.collection.Count(ctx)
	if err != nil {
		cs.metrics.ConnectionStatus = connectionStatusError
		cs.updateMetrics("health_check", start, err)
		return fmt.Errorf("health check failed: %w", err)
	}

	logging.Info("Chroma health check passed", "document_count", count)
	cs.metrics.ConnectionStatus = "healthy"
	return nil
}

// GetStats returns statistics about the store
func (cs *ChromaStore) GetStats(ctx context.Context) (*StoreStats, error) {
	start := time.Now()
	defer cs.updateMetrics("get_stats", start, nil)

	count, err := cs.collection.Count(ctx)
	if err != nil {
		cs.updateMetrics("get_stats", start, err)
		return nil, fmt.Errorf("failed to get collection stats: %w", err)
	}

	// For detailed stats, we'd need to query and aggregate
	stats := &StoreStats{
		TotalChunks:  int64(count),
		ChunksByType: make(map[string]int64),
		ChunksByRepo: make(map[string]int64),
		StorageSize:  0, // Not available via API
	}

	return stats, nil
}

// Cleanup removes old chunks based on retention policy
func (cs *ChromaStore) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	start := time.Now()
	defer cs.updateMetrics("cleanup", start, nil)

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	cutoffEpoch := cutoffTime.Unix()

	// Get old chunks using timestamp filter
	whereFilter := map[string]interface{}{
		"timestamp_epoch": map[string]interface{}{
			"$lt": float64(cutoffEpoch),
		},
	}

	result, err := cs.collection.Get(
		ctx,
		nil, // ids - nil means get all matching where clause
		whereFilter,
		[]string{"metadatas"},
	)

	if err != nil {
		cs.updateMetrics("cleanup", start, err)
		return 0, fmt.Errorf("failed to get old chunks: %w", err)
	}

	if result == nil || len(result.IDs) == 0 {
		return 0, nil
	}

	// Delete the old chunks
	err = cs.collection.Delete(ctx, result.IDs)
	if err != nil {
		cs.updateMetrics("cleanup", start, err)
		return 0, fmt.Errorf("failed to delete old chunks: %w", err)
	}

	return len(result.IDs), nil
}

// Close closes the client connection
func (cs *ChromaStore) Close() error {
	cs.metrics.ConnectionStatus = "closed"
	if cs.client != nil {
		return cs.client.Close()
	}
	return nil
}

// Helper methods

func (cs *ChromaStore) chunkToDocument(chunk types.ConversationChunk) string {
	return fmt.Sprintf("Type: %s\nContent: %s\nSummary: %s", chunk.Type, chunk.Content, chunk.Summary)
}

// documentMetadataToMap is no longer needed as we work with native Go maps

func (cs *ChromaStore) chunkToMetadata(chunk types.ConversationChunk) map[string]interface{} {
	metadata := map[string]interface{}{
		"session_id":      chunk.SessionID,
		"timestamp":       chunk.Timestamp.Format(time.RFC3339),
		"timestamp_epoch": float64(chunk.Timestamp.Unix()), // Use float64 for numeric comparisons
		"type":            string(chunk.Type),
		"summary":         chunk.Summary,
		"repository":      chunk.Metadata.Repository,
		"branch":          chunk.Metadata.Branch,
		"outcome":         string(chunk.Metadata.Outcome),
		"difficulty":      string(chunk.Metadata.Difficulty),
		"tags":            strings.Join(chunk.Metadata.Tags, ","),
		"tools_used":      strings.Join(chunk.Metadata.ToolsUsed, ","),
		"files_modified":  strings.Join(chunk.Metadata.FilesModified, ","),
	}

	if chunk.Metadata.TimeSpent != nil {
		metadata["time_spent"] = float64(*chunk.Metadata.TimeSpent)
	}

	return metadata
}

// metadataToAttributes is no longer needed as we work with native Go maps

func (cs *ChromaStore) chromaResultToChunk(id string, document string, metadata map[string]interface{}) *types.ConversationChunk {
	chunk := &types.ConversationChunk{
		ID: id,
	}

	if sessionID, ok := metadata["session_id"].(string); ok {
		chunk.SessionID = sessionID
	}

	if timestampStr, ok := metadata["timestamp"].(string); ok {
		if timestamp, err := time.Parse(time.RFC3339, timestampStr); err == nil {
			chunk.Timestamp = timestamp
		}
	}

	if typeStr, ok := metadata["type"].(string); ok {
		chunk.Type = types.ChunkType(typeStr)
	}

	if summary, ok := metadata["summary"].(string); ok {
		chunk.Summary = summary
	}

	// Extract content from document
	lines := strings.Split(document, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Content: ") {
			chunk.Content = strings.TrimPrefix(line, "Content: ")
			break
		}
	}

	// Build metadata
	chunkMetadata := types.ChunkMetadata{}

	if repo, ok := metadata["repository"].(string); ok {
		chunkMetadata.Repository = repo
	}

	if branch, ok := metadata["branch"].(string); ok {
		chunkMetadata.Branch = branch
	}

	if outcome, ok := metadata["outcome"].(string); ok {
		chunkMetadata.Outcome = types.Outcome(outcome)
	}

	if difficulty, ok := metadata["difficulty"].(string); ok {
		chunkMetadata.Difficulty = types.Difficulty(difficulty)
	}

	if tags, ok := metadata["tags"].(string); ok && tags != "" {
		chunkMetadata.Tags = strings.Split(tags, ",")
	}

	if toolsUsed, ok := metadata["tools_used"].(string); ok && toolsUsed != "" {
		chunkMetadata.ToolsUsed = strings.Split(toolsUsed, ",")
	}

	if filesModified, ok := metadata["files_modified"].(string); ok && filesModified != "" {
		chunkMetadata.FilesModified = strings.Split(filesModified, ",")
	}

	if timeSpent, ok := metadata["time_spent"]; ok {
		var timeSpentInt int
		switch v := timeSpent.(type) {
		case float64:
			timeSpentInt = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				timeSpentInt = parsed
			}
		}
		if timeSpentInt > 0 {
			chunkMetadata.TimeSpent = &timeSpentInt
		}
	}

	chunk.Metadata = chunkMetadata
	return chunk
}

func (cs *ChromaStore) buildWhereFilter(query types.MemoryQuery) map[string]interface{} {
	filters := make(map[string]interface{})

	logging.Info("ChromaStore: Building where filters", "repository", query.Repository, "types", query.Types, "recency", query.Recency)

	if query.Repository != nil && *query.Repository != "" {
		filters["repository"] = *query.Repository
		logging.Info("ChromaStore: Added repository filter", "repository", *query.Repository)
	}

	if len(query.Types) > 0 {
		typeStrings := make([]string, len(query.Types))
		for i, t := range query.Types {
			typeStrings[i] = string(t)
		}
		filters["type"] = map[string]interface{}{
			"$in": typeStrings,
		}
		logging.Info("ChromaStore: Added type filter", "types", typeStrings)
	}

	// Add time-based filtering using epoch timestamps
	switch query.Recency {
	case types.RecencyRecent:
		recentEpoch := float64(time.Now().AddDate(0, 0, -7).Unix())
		filters["timestamp_epoch"] = map[string]interface{}{
			"$gt": recentEpoch,
		}
		logging.Info("ChromaStore: Added recent time filter", "since_epoch", recentEpoch)
	case types.RecencyLastMonth:
		monthEpoch := float64(time.Now().AddDate(0, -1, 0).Unix())
		filters["timestamp_epoch"] = map[string]interface{}{
			"$gt": monthEpoch,
		}
		logging.Info("ChromaStore: Added month time filter", "since_epoch", monthEpoch)
	case types.RecencyAllTime:
		logging.Info("ChromaStore: No time filter (all time)")
	}

	if len(filters) == 0 {
		return nil
	}

	logging.Info("ChromaStore: Built where filter", "filter_count", len(filters))
	return filters
}

func (cs *ChromaStore) updateMetrics(operation string, start time.Time, err error) {
	duration := time.Since(start)

	cs.metrics.OperationCounts[operation]++

	// Update average latency
	if currentLatency, exists := cs.metrics.AverageLatency[operation]; exists {
		count := cs.metrics.OperationCounts[operation]
		cs.metrics.AverageLatency[operation] = (currentLatency*float64(count-1) + duration.Seconds()*1000) / float64(count)
	} else {
		cs.metrics.AverageLatency[operation] = duration.Seconds() * 1000
	}

	if err != nil {
		cs.metrics.ErrorCounts[operation]++
	}

	now := time.Now().Format(time.RFC3339)
	cs.metrics.LastOperation = &now
}

// Embedding functions are no longer needed with simplified client

// getEnvInt gets an integer from environment variable with a default
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}
