package storage

import (
	"context"
	"fmt"
	chromav2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"mcp-memory/internal/config"
	"mcp-memory/internal/storage/pool"
	"mcp-memory/pkg/types"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

// poolNoOpEmbeddingFunction is a no-op embedding function for pool use
type poolNoOpEmbeddingFunction struct{}

func (n *poolNoOpEmbeddingFunction) EmbedDocuments(ctx context.Context, documents []string) ([]embeddings.Embedding, error) {
	dimension := 1536
	result := make([]embeddings.Embedding, len(documents))
	for i := range result {
		result[i] = embeddings.NewEmbeddingFromFloat64(make([]float64, dimension))
	}
	return result, nil
}

func (n *poolNoOpEmbeddingFunction) EmbedQuery(ctx context.Context, query string) (embeddings.Embedding, error) {
	dimension := 1536
	return embeddings.NewEmbeddingFromFloat64(make([]float64, dimension)), nil
}

func (n *poolNoOpEmbeddingFunction) EmbedRecords(ctx context.Context, records []map[string]interface{}, force bool) error {
	return nil
}

// ChromaPooledConnection implements pool.Connection for Chroma
type ChromaPooledConnection struct {
	client     chromav2.Client
	collection chromav2.Collection
	config     *config.ChromaConfig
	mu         sync.Mutex
	alive      bool
}

// NewChromaPooledConnection creates a new pooled Chroma connection
func NewChromaPooledConnection(ctx context.Context, cfg *config.ChromaConfig) (*ChromaPooledConnection, error) {
	client, err := chromav2.NewHTTPClient(chromav2.WithBaseURL(cfg.Endpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create Chroma client: %w", err)
	}

	// Get or create collection
	embeddingFunc := &poolNoOpEmbeddingFunction{}
	collection, err := client.GetOrCreateCollection(
		ctx,
		cfg.Collection,
		chromav2.WithEmbeddingFunctionCreate(embeddingFunc),
		chromav2.WithHNSWSpaceCreate(embeddings.COSINE),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get/create collection: %w", err)
	}

	return &ChromaPooledConnection{
		client:     client,
		collection: collection,
		config:     cfg,
		alive:      true,
	}, nil
}

// IsAlive checks if the connection is still valid
func (c *ChromaPooledConnection) IsAlive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.alive {
		return false
	}

	// Perform a simple health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.collection.Count(ctx)
	if err != nil {
		c.alive = false
		return false
	}

	return true
}

// Close closes the connection
func (c *ChromaPooledConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.alive = false
	// Chroma client doesn't have explicit close, just mark as not alive
	return nil
}

// Reset resets the connection state
func (c *ChromaPooledConnection) Reset() error {
	// Nothing to reset for Chroma connections
	return nil
}

// PooledChromaStore implements VectorStore using a connection pool
type PooledChromaStore struct {
	pool   *pool.ConnectionPool
	config *config.ChromaConfig
}

// NewPooledChromaStore creates a new pooled Chroma store
func NewPooledChromaStore(cfg *config.ChromaConfig) (*PooledChromaStore, error) {
	poolConfig := &pool.PoolConfig{
		MaxSize:             10,
		MinSize:             2,
		MaxIdleTime:         30 * time.Minute,
		MaxLifetime:         2 * time.Hour,
		HealthCheckInterval: 1 * time.Minute,
	}

	// Override with environment variables if set
	if maxSize := getEnvIntPool("CHROMA_POOL_MAX_SIZE", 0); maxSize > 0 {
		poolConfig.MaxSize = maxSize
	}
	if minSize := getEnvIntPool("CHROMA_POOL_MIN_SIZE", 0); minSize > 0 {
		poolConfig.MinSize = minSize
	}

	factory := func(ctx context.Context) (pool.Connection, error) {
		return NewChromaPooledConnection(ctx, cfg)
	}

	connectionPool, err := pool.NewConnectionPool(poolConfig, factory)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return &PooledChromaStore{
		pool:   connectionPool,
		config: cfg,
	}, nil
}

// getConnection gets a connection from the pool
func (s *PooledChromaStore) getConnection(ctx context.Context) (*ChromaPooledConnection, error) {
	conn, err := s.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection from pool: %w", err)
	}

	// Type assert to ChromaPooledConnection
	// We need to unwrap the wrapped connection
	if wrappedConn, ok := conn.(*pool.WrappedConn); ok {
		if chromaConn, ok := wrappedConn.Unwrap().(*ChromaPooledConnection); ok {
			return chromaConn, nil
		}
	}

	// Try direct cast (shouldn't normally happen)
	if chromaConn, ok := conn.(*ChromaPooledConnection); ok {
		return chromaConn, nil
	}

	_ = conn.Close()
	return nil, fmt.Errorf("invalid connection type")
}

// Initialize initializes the store
func (s *PooledChromaStore) Initialize(ctx context.Context) error {
	// Pool is already initialized, just test a connection
	conn, err := s.getConnection(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	return nil
}

// Store stores a chunk
func (s *PooledChromaStore) Store(ctx context.Context, chunk types.ConversationChunk) error {
	conn, err := s.getConnection(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	// Convert to Chroma format
	metadata := chunkToMetadata(chunk)
	attrs := metadataToAttributes(metadata)

	err = conn.collection.Add(
		ctx,
		chromav2.WithIDs(chromav2.DocumentID(chunk.ID)),
		chromav2.WithEmbeddings(embeddings.NewEmbeddingFromFloat64(chunk.Embeddings)),
		chromav2.WithMetadatas(chromav2.NewDocumentMetadata(attrs...)),
		chromav2.WithTexts(chunk.Content),
	)
	if err != nil {
		return fmt.Errorf("failed to add to collection: %w", err)
	}

	return nil
}

// Search performs semantic search
func (s *PooledChromaStore) Search(ctx context.Context, query types.MemoryQuery, queryEmbeddings []float64) (*types.SearchResults, error) {
	conn, err := s.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	// Build query options
	queryOptions := []chromav2.CollectionQueryOption{
		chromav2.WithQueryEmbeddings(embeddings.NewEmbeddingFromFloat64(queryEmbeddings)),
		chromav2.WithNResults(query.Limit),
		chromav2.WithIncludeQuery(chromav2.IncludeDocuments, chromav2.IncludeMetadatas),
	}

	// Add where filter if needed
	whereFilter := buildWhereFilter(query)
	if whereFilter != nil {
		queryOptions = append(queryOptions, chromav2.WithWhereQuery(whereFilter))
	}

	// Perform query
	results, err := conn.collection.Query(ctx, queryOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection: %w", err)
	}

	return processQueryResults(results, query.MinRelevanceScore), nil
}

// GetByID gets a chunk by ID
func (s *PooledChromaStore) GetByID(ctx context.Context, id string) (*types.ConversationChunk, error) {
	conn, err := s.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	getOptions := []chromav2.CollectionGetOption{
		chromav2.WithIDsGet(chromav2.DocumentID(id)),
		chromav2.WithIncludeGet(chromav2.IncludeDocuments, chromav2.IncludeMetadatas),
	}

	results, err := conn.collection.Get(ctx, getOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to get by ID: %w", err)
	}

	if results == nil || len(results.GetIDs()) == 0 {
		return nil, fmt.Errorf("chunk not found")
	}

	return getResultToChunk(results, 0)
}

// ListByRepository lists chunks by repository
func (s *PooledChromaStore) ListByRepository(ctx context.Context, repository string, limit int, offset int) ([]types.ConversationChunk, error) {
	conn, err := s.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	getOptions := []chromav2.CollectionGetOption{
		chromav2.WithWhereGet(chromav2.EqString("repository", repository)),
		chromav2.WithIncludeGet(chromav2.IncludeDocuments, chromav2.IncludeMetadatas),
	}

	// Chroma doesn't support offset directly, so we need to get all and slice
	results, err := conn.collection.Get(ctx, getOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to list by repository: %w", err)
	}

	if results == nil {
		return []types.ConversationChunk{}, nil
	}

	ids := results.GetIDs()
	chunks := make([]types.ConversationChunk, 0)
	for i := offset; i < len(ids) && i < offset+limit; i++ {
		chunk, err := getResultToChunk(results, i)
		if err != nil {
			continue
		}
		chunks = append(chunks, *chunk)
	}

	return chunks, nil
}

// ListBySession lists chunks by session ID
func (s *PooledChromaStore) ListBySession(ctx context.Context, sessionID string) ([]types.ConversationChunk, error) {
	conn, err := s.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	getOptions := []chromav2.CollectionGetOption{
		chromav2.WithWhereGet(chromav2.EqString("session_id", sessionID)),
		chromav2.WithIncludeGet(chromav2.IncludeDocuments, chromav2.IncludeMetadatas),
	}

	results, err := conn.collection.Get(ctx, getOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to list by session: %w", err)
	}

	if results == nil {
		return []types.ConversationChunk{}, nil
	}

	ids := results.GetIDs()
	chunks := make([]types.ConversationChunk, 0, len(ids))
	for i := 0; i < len(ids); i++ {
		chunk, err := getResultToChunk(results, i)
		if err != nil {
			continue
		}
		chunks = append(chunks, *chunk)
	}

	// Sort by timestamp
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Timestamp.Before(chunks[j].Timestamp)
	})

	return chunks, nil
}

// Delete deletes a chunk
func (s *PooledChromaStore) Delete(ctx context.Context, id string) error {
	conn, err := s.getConnection(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	err = conn.collection.Delete(ctx, chromav2.WithIDsDelete(chromav2.DocumentID(id)))
	if err != nil {
		return fmt.Errorf("failed to delete chunk: %w", err)
	}

	return nil
}

// Update updates a chunk
func (s *PooledChromaStore) Update(ctx context.Context, chunk types.ConversationChunk) error {
	conn, err := s.getConnection(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	// Chroma doesn't have direct update, so delete and re-add
	if err := s.Delete(ctx, chunk.ID); err != nil {
		return err
	}

	return s.Store(ctx, chunk)
}

// HealthCheck performs health check
func (s *PooledChromaStore) HealthCheck(ctx context.Context) error {
	conn, err := s.getConnection(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	_, err = conn.collection.Count(ctx)
	return err
}

// GetStats returns store statistics
func (s *PooledChromaStore) GetStats(ctx context.Context) (*StoreStats, error) {
	conn, err := s.getConnection(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()

	count, err := conn.collection.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get count: %w", err)
	}

	// Get pool stats
	poolStats := s.pool.Stats()

	stats := &StoreStats{
		TotalChunks:  int64(count),
		ChunksByType: make(map[string]int64),
		ChunksByRepo: make(map[string]int64),
		// Add pool stats as metadata
		StorageSize: int64(poolStats.CurrentSize),
	}

	return stats, nil
}

// Cleanup removes old chunks
func (s *PooledChromaStore) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	conn, err := s.getConnection(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = conn.Close() }()

	// Calculate cutoff time
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	// Get all chunks to find old ones
	getOptions := []chromav2.CollectionGetOption{
		chromav2.WithIncludeGet(chromav2.IncludeMetadatas),
		// Note: WithLimit might not be available in Get, only in Query
	}

	results, err := conn.collection.Get(ctx, getOptions...)
	if err != nil {
		return 0, fmt.Errorf("failed to get chunks for cleanup: %w", err)
	}

	if results == nil {
		return 0, nil
	}

	// Find chunks to delete
	var toDelete []chromav2.DocumentID
	ids := results.GetIDs()
	metas := results.GetMetadatas()

	for i, meta := range metas {
		if meta == nil {
			continue
		}

		// Check timestamp
		if timestamp, ok := meta.GetFloat("timestamp"); ok {
			chunkTime := time.Unix(int64(timestamp), 0)
			if chunkTime.Before(cutoff) && i < len(ids) {
				toDelete = append(toDelete, ids[i])
			}
		}
	}

	if len(toDelete) == 0 {
		return 0, nil
	}

	// Delete old chunks
	err = conn.collection.Delete(ctx, chromav2.WithIDsDelete(toDelete...))
	if err != nil {
		return 0, fmt.Errorf("failed to delete old chunks: %w", err)
	}

	return len(toDelete), nil
}

// Close closes the store
func (s *PooledChromaStore) Close() error {
	return s.pool.Close()
}

// GetPoolStats returns connection pool statistics
func (s *PooledChromaStore) GetPoolStats() pool.PoolStats {
	return s.pool.Stats()
}

// Helper functions

// getEnvIntPool gets an integer from environment variable for pool config
func getEnvIntPool(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// chunkToMetadata converts a chunk to Chroma metadata
func chunkToMetadata(chunk types.ConversationChunk) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Required fields
	metadata["session_id"] = chunk.SessionID
	metadata["timestamp"] = chunk.Timestamp.Unix()
	metadata["type"] = string(chunk.Type)
	metadata["summary"] = chunk.Summary

	// Metadata fields
	if chunk.Metadata.Repository != "" {
		metadata["repository"] = chunk.Metadata.Repository
	}

	if chunk.Metadata.Branch != "" {
		metadata["branch"] = chunk.Metadata.Branch
	}

	// Arrays
	if len(chunk.Metadata.Tags) > 0 {
		metadata["tags"] = chunk.Metadata.Tags
	}

	if len(chunk.Metadata.FilesModified) > 0 {
		metadata["files_modified"] = chunk.Metadata.FilesModified
	}

	if len(chunk.Metadata.ToolsUsed) > 0 {
		metadata["tools_used"] = chunk.Metadata.ToolsUsed
	}

	// Optional fields
	if chunk.Metadata.TimeSpent != nil {
		metadata["time_spent"] = *chunk.Metadata.TimeSpent
	}

	metadata["outcome"] = string(chunk.Metadata.Outcome)
	metadata["difficulty"] = string(chunk.Metadata.Difficulty)

	return metadata
}

// These functions are now at the end of the file after the new helper functions

// metadataToAttributes converts metadata to Chroma attributes
func metadataToAttributes(metadata map[string]interface{}) []*chromav2.MetaAttribute {
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
			attrs = append(attrs, chromav2.NewStringAttribute(key, strconv.FormatBool(v)))
		default:
			attrs = append(attrs, chromav2.NewStringAttribute(key, fmt.Sprintf("%v", v)))
		}
	}

	return attrs
}

// interfaceSliceToStringSlice converts []interface{} to []string
func interfaceSliceToStringSlice(slice []interface{}) []string {
	result := make([]string, 0, len(slice))
	for _, v := range slice {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// buildWhereFilter builds a where filter for ChromaDB queries
func buildWhereFilter(query types.MemoryQuery) chromav2.WhereClause {
	var clauses []chromav2.WhereClause

	// Add repository filter
	if query.Repository != nil && *query.Repository != "" {
		clauses = append(clauses, chromav2.EqString("repository", *query.Repository))
	}

	// Add types filter
	if len(query.Types) > 0 {
		// Convert ChunkType to string slice
		typeStrings := make([]string, len(query.Types))
		for i, t := range query.Types {
			typeStrings[i] = string(t)
		}
		clauses = append(clauses, chromav2.InString("type", typeStrings...))
	}

	// Add recency filter based on the Recency enum
	var startTime time.Time
	now := time.Now()
	switch query.Recency {
	case types.RecencyRecent:
		startTime = now.AddDate(0, 0, -7) // Last 7 days
	case types.RecencyLastMonth:
		startTime = now.AddDate(0, -1, 0) // Last month
	case types.RecencyAllTime:
		// No time filter for all time
		startTime = time.Time{}
	}

	if !startTime.IsZero() {
		// Add timestamp filter - timestamps are stored as Unix timestamps (float64)
		clauses = append(clauses, chromav2.GteFloat("timestamp", float32(startTime.Unix())))
	}

	// Combine all clauses with AND
	switch len(clauses) {
	case 0:
		return nil
	case 1:
		return clauses[0]
	default:
		return chromav2.And(clauses...)
	}
}

// processQueryResults processes ChromaDB query results
func processQueryResults(result chromav2.QueryResult, minScore float64) *types.SearchResults {
	searchResults := &types.SearchResults{
		Results: make([]types.SearchResult, 0),
		Total:   0,
	}

	if result == nil {
		return searchResults
	}

	// ChromaDB v2 returns results in groups (for batch queries)
	// We only query one at a time, so we get the first group
	idGroups := result.GetIDGroups()
	docGroups := result.GetDocumentsGroups()
	metaGroups := result.GetMetadatasGroups()
	distGroups := result.GetDistancesGroups()

	if len(idGroups) == 0 {
		return searchResults
	}

	// Get first group (we only have one query)
	ids := idGroups[0]
	docs := docGroups[0]
	metas := metaGroups[0]
	dists := distGroups[0]

	// Process each result
	for i := 0; i < len(ids); i++ {
		// Convert distance to similarity score
		score := 1.0
		if i < len(dists) {
			score = 1.0 - float64(dists[i])
		}

		if score < minScore {
			continue
		}

		chunk := &types.ConversationChunk{
			ID:       string(ids[i]),
			Metadata: types.ChunkMetadata{},
		}

		// Set content
		if i < len(docs) {
			chunk.Content = docs[i].ContentString()
		}

		// Parse metadata
		if i < len(metas) && metas[i] != nil {
			parseMetadataV2(metas[i], chunk)
		}

		searchResults.Results = append(searchResults.Results, types.SearchResult{
			Chunk: *chunk,
			Score: score,
		})
	}

	searchResults.Total = len(searchResults.Results)
	return searchResults
}

// getResultToChunk converts a get result to a ConversationChunk
func getResultToChunk(results chromav2.GetResult, index int) (*types.ConversationChunk, error) {
	if results == nil {
		return nil, fmt.Errorf("nil results")
	}

	ids := results.GetIDs()
	docs := results.GetDocuments()
	metas := results.GetMetadatas()

	if index >= len(ids) {
		return nil, fmt.Errorf("index out of bounds")
	}

	chunk := &types.ConversationChunk{
		ID:       string(ids[index]),
		Metadata: types.ChunkMetadata{},
	}

	if index < len(docs) {
		chunk.Content = docs[index].ContentString()
	}

	if index < len(metas) && metas[index] != nil {
		parseMetadataV2(metas[index], chunk)
	}

	return chunk, nil
}

// parseMetadataV2 parses ChromaDB v2 metadata into chunk metadata
func parseMetadataV2(metadata chromav2.DocumentMetadata, chunk *types.ConversationChunk) {
	// Required fields
	if v, ok := metadata.GetString("session_id"); ok {
		chunk.SessionID = v
	}
	if v, ok := metadata.GetString("repository"); ok {
		chunk.Metadata.Repository = v
	}
	if v, ok := metadata.GetString("branch"); ok {
		chunk.Metadata.Branch = v
	}

	// Timestamp
	if v, ok := metadata.GetFloat("timestamp"); ok {
		chunk.Timestamp = time.Unix(int64(v), 0)
	}

	// Type
	if v, ok := metadata.GetString("type"); ok {
		chunk.Type = types.ChunkType(v)
	}

	// Summary
	if v, ok := metadata.GetString("summary"); ok {
		chunk.Summary = v
	}

	// Arrays - these need special handling as they're stored as raw interface{}
	if v, ok := metadata.GetRaw("tags"); ok {
		if tags, ok := v.([]interface{}); ok {
			chunk.Metadata.Tags = interfaceSliceToStringSlice(tags)
		} else if tags, ok := v.([]string); ok {
			chunk.Metadata.Tags = tags
		}
	}

	if v, ok := metadata.GetRaw("files_modified"); ok {
		if files, ok := v.([]interface{}); ok {
			chunk.Metadata.FilesModified = interfaceSliceToStringSlice(files)
		} else if files, ok := v.([]string); ok {
			chunk.Metadata.FilesModified = files
		}
	}

	if v, ok := metadata.GetRaw("tools_used"); ok {
		if tools, ok := v.([]interface{}); ok {
			chunk.Metadata.ToolsUsed = interfaceSliceToStringSlice(tools)
		} else if tools, ok := v.([]string); ok {
			chunk.Metadata.ToolsUsed = tools
		}
	}

	// Optional fields
	if v, ok := metadata.GetInt("time_spent"); ok {
		i := int(v)
		chunk.Metadata.TimeSpent = &i
	}

	// Outcome
	if v, ok := metadata.GetString("outcome"); ok {
		chunk.Metadata.Outcome = types.Outcome(v)
	}

	// Difficulty
	if v, ok := metadata.GetString("difficulty"); ok {
		chunk.Metadata.Difficulty = types.Difficulty(v)
	}
}
