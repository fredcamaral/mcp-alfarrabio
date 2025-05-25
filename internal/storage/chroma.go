package storage

import (
	"context"
	"fmt"
	"mcp-memory/internal/config"
	"mcp-memory/internal/logging"
	"mcp-memory/pkg/types"
	"os"
	"strconv"
	"strings"
	"time"

	chromav2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// Constants
const (
	connectionStatusError = "error"
)

// ChromaStore implements VectorStore interface for Chroma vector database
type ChromaStore struct {
	client     chromav2.Client
	collection chromav2.Collection
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
	client, err := chromav2.NewHTTPClient(chromav2.WithBaseURL(cs.config.Endpoint))
	if err != nil {
		cs.metrics.ConnectionStatus = connectionStatusError
		return fmt.Errorf("failed to create Chroma client: %w", err)
	}
	cs.client = client

	// Create or get collection with custom embedding function wrapper
	embeddingFunc := &noOpEmbeddingFunction{}
	
	collection, err := cs.client.GetOrCreateCollection(
		ctx,
		cs.config.Collection,
		chromav2.WithEmbeddingFunctionCreate(embeddingFunc),
		chromav2.WithCollectionMetadataCreate(
			chromav2.NewMetadata(
				chromav2.NewStringAttribute("description", "Claude conversation memory storage"),
				chromav2.NewStringAttribute("created_at", time.Now().UTC().Format(time.RFC3339)),
			),
		),
		chromav2.WithHNSWSpaceCreate(embeddings.COSINE),
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

	// Convert metadata to Chroma attributes
	attrs := cs.metadataToAttributes(metadata)

	// Add document to collection
	err := cs.collection.Add(
		ctx,
		chromav2.WithIDs(chromav2.DocumentID(chunk.ID)),
		chromav2.WithEmbeddings(embeddings.NewEmbeddingFromFloat64(chunk.Embeddings)),
		chromav2.WithMetadatas(chromav2.NewDocumentMetadata(attrs...)),
		chromav2.WithTexts(document),
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

	// Build query options
	queryOptions := []chromav2.CollectionQueryOption{
		chromav2.WithQueryEmbeddings(embeddings.NewEmbeddingFromFloat64(queryEmbeddings)),
		chromav2.WithNResults(limit),
		chromav2.WithIncludeQuery(chromav2.IncludeDocuments, chromav2.IncludeMetadatas),
	}

	// Add where filters if needed
	whereFilter := cs.buildWhereFilter(query)
	if whereFilter != nil {
		queryOptions = append(queryOptions, chromav2.WithWhereQuery(whereFilter))
	}

	// Query the collection
	qr, err := cs.collection.Query(ctx, queryOptions...)
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

func (cs *ChromaStore) processQueryResults(qr chromav2.QueryResult, query *types.MemoryQuery, results *types.SearchResults) *types.SearchResults {
	docs := qr.GetDocumentsGroups()
	if len(docs) == 0 || len(docs[0]) == 0 {
		return results
	}
	
	results.Total = len(docs[0])
	metadatas := qr.GetMetadatasGroups()
	distances := qr.GetDistancesGroups()
	ids := qr.GetIDGroups()
	
	// Ensure all result groups have data
	if len(metadatas) == 0 || len(distances) == 0 || len(ids) == 0 {
		return results
	}
	
	for i := 0; i < len(docs[0]); i++ {
		result := cs.processSearchResult(i, ids[0], docs[0], cs.convertMetadatas(metadatas[0]), cs.convertDistances(distances[0]), query.MinRelevanceScore)
		if result != nil {
			results.Results = append(results.Results, *result)
		}
	}
	
	return results
}

func (cs *ChromaStore) processSearchResult(index int, ids []chromav2.DocumentID, docs []chromav2.Document, metadatas []map[string]interface{}, distances []float32, minScore float64) *types.SearchResult {
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

func (cs *ChromaStore) extractChunkID(index int, ids []chromav2.DocumentID) string {
	if len(ids) > index {
		return string(ids[index])
	}
	return ""
}

func (cs *ChromaStore) extractMetadata(index int, metadatas []map[string]interface{}) map[string]interface{} {
	if len(metadatas) > index {
		return metadatas[index]
	}
	return nil
}

func (cs *ChromaStore) extractContent(index int, docs []chromav2.Document) string {
	if len(docs) > index && docs[index] != nil {
		return docs[index].ContentString()
	}
	return ""
}

func (cs *ChromaStore) calculateScore(index int, distances []float32) float64 {
	if len(distances) > index {
		return 1.0 - float64(distances[index])
	}
	return 1.0
}

func (cs *ChromaStore) convertDistances(distances embeddings.Distances) []float32 {
	result := make([]float32, len(distances))
	for i, d := range distances {
		result[i] = float32(d)
	}
	return result
}

func (cs *ChromaStore) convertMetadatas(metadatas chromav2.DocumentMetadatas) []map[string]interface{} {
	result := make([]map[string]interface{}, len(metadatas))
	for i, meta := range metadatas {
		result[i] = cs.documentMetadataToMap(meta)
	}
	return result
}

// GetByID retrieves a chunk by its ID
func (cs *ChromaStore) GetByID(ctx context.Context, id string) (*types.ConversationChunk, error) {
	start := time.Now()
	defer cs.updateMetrics("get_by_id", start, nil)

	result, err := cs.collection.Get(
		ctx,
		chromav2.WithIDsGet(chromav2.DocumentID(id)),
		chromav2.WithIncludeGet(chromav2.IncludeDocuments, chromav2.IncludeMetadatas),
	)

	if err != nil {
		cs.updateMetrics("get_by_id", start, err)
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	if result == nil || len(result.GetDocuments()) == 0 {
		return nil, fmt.Errorf("chunk not found: %s", id)
	}

	docs := result.GetDocuments()
	metadatas := result.GetMetadatas()
	ids := result.GetIDs()

	if len(ids) > 0 && len(docs) > 0 {
		var metadata map[string]interface{}
		if len(metadatas) > 0 {
			metadata = cs.documentMetadataToMap(metadatas[0])
		}
		// Extract content from Document interface
		var content string
		if doc := docs[0]; doc != nil {
			content = doc.ContentString()
		}
		return cs.chromaResultToChunk(string(ids[0]), content, metadata), nil
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

	// Build options for get
	getOptions := []chromav2.CollectionGetOption{
		chromav2.WithWhereGet(chromav2.EqString("repository", repository)),
		chromav2.WithIncludeGet(chromav2.IncludeDocuments, chromav2.IncludeMetadatas),
	}

	if limit > 0 {
		getOptions = append(getOptions, chromav2.WithLimitGet(limit))
	}
	if offset > 0 {
		getOptions = append(getOptions, chromav2.WithOffsetGet(offset))
	}

	result, err := cs.collection.Get(ctx, getOptions...)
	if err != nil {
		cs.updateMetrics("list_by_repository", start, err)
		return nil, fmt.Errorf("failed to list chunks: %w", err)
	}

	chunks := make([]types.ConversationChunk, 0)
	if result != nil {
		docs := result.GetDocuments()
		metadatas := result.GetMetadatas()
		ids := result.GetIDs()

		for i := 0; i < len(docs); i++ {
			var chunkID string
			if i < len(ids) {
				chunkID = string(ids[i])
			}

			var metadata map[string]interface{}
			if i < len(metadatas) {
				metadata = cs.documentMetadataToMap(metadatas[i])
			}

			// Extract content from Document interface
			var content string
			if i < len(docs) && docs[i] != nil {
				content = docs[i].ContentString()
			}

			chunk := cs.chromaResultToChunk(chunkID, content, metadata)
			chunks = append(chunks, *chunk)
		}
	}

	return chunks, nil
}

// Delete removes a chunk by ID
func (cs *ChromaStore) Delete(ctx context.Context, id string) error {
	start := time.Now()
	defer cs.updateMetrics("delete", start, nil)

	err := cs.collection.Delete(ctx, chromav2.WithIDsDelete(chromav2.DocumentID(id)))
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

	// Get old chunks
	result, err := cs.collection.Get(
		ctx,
		chromav2.WithWhereGet(chromav2.LtFloat("timestamp_epoch", float32(cutoffEpoch))),
		chromav2.WithIncludeGet(chromav2.IncludeMetadatas),
	)

	if err != nil {
		cs.updateMetrics("cleanup", start, err)
		return 0, fmt.Errorf("failed to get old chunks: %w", err)
	}

	if result == nil || len(result.GetIDs()) == 0 {
		return 0, nil
	}

	// Delete the old chunks
	ids := result.GetIDs()
	err = cs.collection.Delete(ctx, chromav2.WithIDsDelete(ids...))
	if err != nil {
		cs.updateMetrics("cleanup", start, err)
		return 0, fmt.Errorf("failed to delete old chunks: %w", err)
	}

	return len(ids), nil
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

// documentMetadataToMap converts a DocumentMetadata interface to map[string]interface{}
func (cs *ChromaStore) documentMetadataToMap(dm chromav2.DocumentMetadata) map[string]interface{} {
	if dm == nil {
		return make(map[string]interface{})
	}
	
	// Get the underlying implementation to access Keys() method
	impl, ok := dm.(*chromav2.DocumentMetadataImpl)
	if !ok {
		return make(map[string]interface{})
	}
	
	result := make(map[string]interface{})
	
	// Iterate through all keys and extract values
	for _, key := range impl.Keys() {
		if val, ok := dm.GetString(key); ok {
			result[key] = val
		} else if val, ok := dm.GetInt(key); ok {
			result[key] = val
		} else if val, ok := dm.GetFloat(key); ok {
			result[key] = val
		} else if val, ok := dm.GetBool(key); ok {
			result[key] = val
		}
	}
	
	return result
}

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

func (cs *ChromaStore) metadataToAttributes(metadata map[string]interface{}) []*chromav2.MetaAttribute {
	attrs := make([]*chromav2.MetaAttribute, 0, len(metadata))
	
	for key, value := range metadata {
		switch v := value.(type) {
		case string:
			attrs = append(attrs, chromav2.NewStringAttribute(key, v))
		case float64:
			attrs = append(attrs, chromav2.NewFloatAttribute(key, v))
		case int:
			attrs = append(attrs, chromav2.NewIntAttribute(key, int64(v)))
		case int64:
			attrs = append(attrs, chromav2.NewIntAttribute(key, v))
		case bool:
			// Convert bool to string for Chroma
			attrs = append(attrs, chromav2.NewStringAttribute(key, strconv.FormatBool(v)))
		default:
			// Convert other types to string
			attrs = append(attrs, chromav2.NewStringAttribute(key, fmt.Sprintf("%v", v)))
		}
	}
	
	return attrs
}

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

func (cs *ChromaStore) buildWhereFilter(query types.MemoryQuery) chromav2.WhereClause {
	filters := []chromav2.WhereClause{}

	logging.Info("ChromaStore: Building where filters", "repository", query.Repository, "types", query.Types, "recency", query.Recency)

	if query.Repository != nil && *query.Repository != "" {
		filters = append(filters, chromav2.EqString("repository", *query.Repository))
		logging.Info("ChromaStore: Added repository filter", "repository", *query.Repository)
	}

	if len(query.Types) > 0 {
		typeStrings := make([]string, len(query.Types))
		for i, t := range query.Types {
			typeStrings[i] = string(t)
		}
		filters = append(filters, chromav2.InString("type", typeStrings...))
		logging.Info("ChromaStore: Added type filter", "types", typeStrings)
	}

	// Add time-based filtering using epoch timestamps
	switch query.Recency {
	case types.RecencyRecent:
		recentEpoch := float32(time.Now().AddDate(0, 0, -7).Unix())
		filters = append(filters, chromav2.GtFloat("timestamp_epoch", recentEpoch))
		logging.Info("ChromaStore: Added recent time filter", "since_epoch", recentEpoch)
	case types.RecencyLastMonth:
		monthEpoch := float32(time.Now().AddDate(0, -1, 0).Unix())
		filters = append(filters, chromav2.GtFloat("timestamp_epoch", monthEpoch))
		logging.Info("ChromaStore: Added month time filter", "since_epoch", monthEpoch)
	case types.RecencyAllTime:
		logging.Info("ChromaStore: No time filter (all time)")
	}

	if len(filters) == 0 {
		return nil
	}

	if len(filters) == 1 {
		return filters[0]
	}

	// Combine multiple filters with AND
	logging.Info("ChromaStore: Combining filters with AND", "filter_count", len(filters))
	return chromav2.And(filters...)
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

// noOpEmbeddingFunction is a no-op embedding function since we handle embeddings externally
type noOpEmbeddingFunction struct{}

func (n *noOpEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	// Return empty embeddings - we handle embeddings externally
	dimension := getEnvInt("MCP_MEMORY_EMBEDDING_DIMENSION", 1536)
	result := make([]embeddings.Embedding, len(documents))
	for i := range result {
		result[i] = embeddings.NewEmbeddingFromFloat64(make([]float64, dimension))
	}
	return result, nil
}

func (n *noOpEmbeddingFunction) EmbedQuery(ctx context.Context, query string) (embeddings.Embedding, error) {
	// Return empty embedding - we handle embeddings externally
	dimension := getEnvInt("MCP_MEMORY_EMBEDDING_DIMENSION", 1536)
	return embeddings.NewEmbeddingFromFloat64(make([]float64, dimension)), nil
}

func (n *noOpEmbeddingFunction) EmbedRecords(ctx context.Context, records []map[string]interface{}, force bool) error {
	// No-op - we handle embeddings externally
	return nil
}

// getEnvInt gets an integer from environment variable with a default
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}