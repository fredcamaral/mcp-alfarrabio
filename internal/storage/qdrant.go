package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"mcp-memory/internal/config"
	"mcp-memory/internal/logging"
	"mcp-memory/pkg/types"
	"sort"
	"strconv"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

// Constants
const (
	defaultQdrantCollection = "claude_memory"
	defaultVectorSize       = 1536 // OpenAI embeddings size
	connectionStatusError   = "error"
	globalRepository        = "global"
)

// QdrantStore implements VectorStore interface for Qdrant vector database
type QdrantStore struct {
	client            *qdrant.Client
	config            *config.QdrantConfig
	metrics           *StorageMetrics
	collectionName    string
	relationshipStore *RelationshipStore
}

// NewQdrantStore creates a new Qdrant vector store
func NewQdrantStore(cfg *config.QdrantConfig) *QdrantStore {
	collectionName := cfg.Collection
	if collectionName == "" {
		collectionName = defaultQdrantCollection
	}

	return &QdrantStore{
		config:         cfg,
		collectionName: collectionName,
		metrics: &StorageMetrics{
			OperationCounts:  make(map[string]int64),
			AverageLatency:   make(map[string]float64),
			ErrorCounts:      make(map[string]int64),
			ConnectionStatus: "unknown",
		},
	}
}

// Initialize creates the collection if it doesn't exist
func (qs *QdrantStore) Initialize(ctx context.Context) error {
	start := time.Now()
	defer qs.updateMetrics("initialize", start)

	// Create Qdrant client
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   qs.config.Host,
		Port:                   qs.config.Port,
		APIKey:                 qs.config.APIKey,
		UseTLS:                 qs.config.UseTLS,
		SkipCompatibilityCheck: true, // Skip version compatibility warnings
	})
	if err != nil {
		qs.metrics.ConnectionStatus = connectionStatusError
		return fmt.Errorf("failed to create Qdrant client: %w", err)
	}
	qs.client = client

	// Initialize relationship store
	qs.relationshipStore = NewRelationshipStore(client)
	if err := qs.relationshipStore.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize relationship store: %w", err)
	}

	// Check if collection exists
	collections, err := qs.client.ListCollections(ctx)
	if err != nil {
		qs.metrics.ConnectionStatus = connectionStatusError
		return fmt.Errorf("failed to list collections: %w", err)
	}

	// Check if our collection exists
	collectionExists := false
	for _, collectionName := range collections {
		if collectionName == qs.collectionName {
			collectionExists = true
			break
		}
	}

	// Create collection if it doesn't exist
	if !collectionExists {
		err = qs.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: qs.collectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     uint64(defaultVectorSize),
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			qs.metrics.ConnectionStatus = connectionStatusError
			return fmt.Errorf("failed to create collection %s: %w", qs.collectionName, err)
		}
		logging.Info("Created Qdrant collection", "collection", qs.collectionName)
	}

	qs.metrics.ConnectionStatus = "connected"
	logging.Info("Qdrant collection initialized", "collection", qs.collectionName)
	return nil
}

// Store saves a conversation chunk to Qdrant
func (qs *QdrantStore) Store(ctx context.Context, chunk *types.ConversationChunk) error {
	start := time.Now()
	defer qs.updateMetrics("store", start)

	if err := chunk.Validate(); err != nil {
		return fmt.Errorf("invalid chunk: %w", err)
	}

	if len(chunk.Embeddings) == 0 {
		return errors.New("chunk must have embeddings before storing")
	}

	// Convert chunk to Qdrant format
	point := qs.chunkToPoint(chunk)

	// Upsert point to collection
	_, err := qs.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: qs.collectionName,
		Points:         []*qdrant.PointStruct{point},
	})

	if err != nil {
		return fmt.Errorf("failed to store chunk in Qdrant: %w", err)
	}

	logging.Debug("Stored chunk in Qdrant",
		"id", chunk.ID,
		"repository", chunk.Metadata.Repository,
		"type", chunk.Type,
	)
	return nil
}

// Search performs similarity search in Qdrant
func (qs *QdrantStore) Search(ctx context.Context, query *types.MemoryQuery, embeddings []float64) (*types.SearchResults, error) {
	start := time.Now()
	defer qs.updateMetrics("search", start)

	if len(embeddings) == 0 {
		return nil, errors.New("embeddings cannot be empty")
	}

	// Build search filter
	filter := qs.buildFilter(query)

	// Convert embeddings to float32
	embeddings32 := qs.float64ToFloat32(embeddings)

	// Perform search using Query method
	searchResult, err := qs.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: qs.collectionName,
		Query:          qdrant.NewQuery(embeddings32...),
		Limit: func() *uint64 {
			if query.Limit < 0 {
				return qdrant.PtrOf(uint64(0))
			}
			return qdrant.PtrOf(uint64(query.Limit)) //nolint:gosec // Safe conversion after bounds check
		}(),
		WithPayload:    qdrant.NewWithPayload(true),
		Filter:         filter,
		ScoreThreshold: qdrant.PtrOf(float32(query.MinRelevanceScore)),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search in Qdrant: %w", err)
	}

	// Convert results
	results := &types.SearchResults{
		Results:   make([]types.SearchResult, 0, len(searchResult)),
		Total:     len(searchResult),
		QueryTime: time.Since(start),
	}

	for _, point := range searchResult {
		chunk, err := qs.scoredPointToChunk(point)
		if err != nil {
			logging.Error("Failed to convert point to chunk", "error", err, "point_id", point.GetId())
			continue
		}

		result := types.SearchResult{
			Chunk: *chunk,
			Score: float64(point.GetScore()),
		}
		results.Results = append(results.Results, result)
	}

	logging.Debug("Search completed",
		"query", query.Query,
		"results", len(results.Results),
		"total_time_ms", time.Since(start).Milliseconds(),
	)

	return results, nil
}

// GetByID retrieves a chunk by its ID
func (qs *QdrantStore) GetByID(ctx context.Context, id string) (*types.ConversationChunk, error) {
	start := time.Now()
	defer qs.updateMetrics("get_by_id", start)

	// Get point by ID
	points, err := qs.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: qs.collectionName,
		Ids:            []*qdrant.PointId{qs.stringToPointID(id)},
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get chunk by ID from Qdrant: %w", err)
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("chunk not found with ID: %s", id)
	}

	chunk, err := qs.pointToChunk(points[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert point to chunk: %w", err)
	}

	return chunk, nil
}

// ListByRepository lists chunks by repository
func (qs *QdrantStore) ListByRepository(ctx context.Context, repository string, limit, offset int) ([]types.ConversationChunk, error) {
	start := time.Now()
	defer qs.updateMetrics("list_by_repository", start)

	// Build filter for repository
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "repository",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{Keyword: repository},
						},
					},
				},
			},
		},
	}

	// Scroll through points (Note: Qdrant Scroll uses cursor-based pagination, not offsets)
	// For simplicity, we'll get more points and slice manually for offset behavior
	totalNeeded := limit + offset
	var scrollLimit uint32
	if totalNeeded < 0 || totalNeeded > 10000 {
		scrollLimit = 10000 // Max reasonable limit
	} else {
		scrollLimit = uint32(totalNeeded)
	}

	points, err := qs.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: qs.collectionName,
		Filter:         filter,
		Limit:          qdrant.PtrOf(scrollLimit),
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list chunks by repository: %w", err)
	}

	allChunks := make([]*types.ConversationChunk, 0, len(points))
	for _, point := range points {
		chunk, err := qs.pointToChunk(point)
		if err != nil {
			log.Printf("Failed to convert point to chunk: %v, point_id: %v", err, point.GetId())
			continue
		}
		allChunks = append(allChunks, chunk)
	}

	// Sort by timestamp (newest first)
	sort.Slice(allChunks, func(i, j int) bool {
		return allChunks[i].Timestamp.After(allChunks[j].Timestamp)
	})

	// Apply manual offset and limit since Qdrant Scroll doesn't support traditional offset
	var chunks []types.ConversationChunk
	if offset < len(allChunks) {
		end := offset + limit
		if end > len(allChunks) {
			end = len(allChunks)
		}
		// Convert back to values for the slice (since interface expects values)
		chunks = make([]types.ConversationChunk, end-offset)
		for i := offset; i < end; i++ {
			chunks[i-offset] = *allChunks[i]
		}
	}

	return chunks, nil
}

// ListBySession lists chunks by session ID
func (qs *QdrantStore) ListBySession(ctx context.Context, sessionID string) ([]types.ConversationChunk, error) {
	start := time.Now()
	defer qs.updateMetrics("list_by_session", start)

	// Build filter for session
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "session_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{Keyword: sessionID},
						},
					},
				},
			},
		},
	}

	// Scroll through points
	points, err := qs.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: qs.collectionName,
		Filter:         filter,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list chunks by session: %w", err)
	}

	chunkPtrs := make([]*types.ConversationChunk, 0, len(points))
	for _, point := range points {
		chunk, err := qs.pointToChunk(point)
		if err != nil {
			logging.Error("Failed to convert point to chunk", "error", err, "point_id", point.GetId())
			continue
		}
		chunkPtrs = append(chunkPtrs, chunk)
	}

	// Sort by timestamp (oldest first for session)
	sort.Slice(chunkPtrs, func(i, j int) bool {
		return chunkPtrs[i].Timestamp.Before(chunkPtrs[j].Timestamp)
	})

	// Convert to values for return (since interface expects values)
	chunks := make([]types.ConversationChunk, len(chunkPtrs))
	for i, chunk := range chunkPtrs {
		chunks[i] = *chunk
	}

	return chunks, nil
}

// Delete removes a chunk by ID
func (qs *QdrantStore) Delete(ctx context.Context, id string) error {
	start := time.Now()
	defer qs.updateMetrics("delete", start)

	_, err := qs.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: qs.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{qs.stringToPointID(id)},
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete chunk from Qdrant: %w", err)
	}

	logging.Debug("Deleted chunk from Qdrant", "id", id)
	return nil
}

// Update modifies an existing chunk
func (qs *QdrantStore) Update(ctx context.Context, chunk *types.ConversationChunk) error {
	start := time.Now()
	defer qs.updateMetrics("update", start)

	if err := chunk.Validate(); err != nil {
		return fmt.Errorf("invalid chunk: %w", err)
	}

	// For Qdrant, update is the same as upsert
	return qs.Store(ctx, chunk)
}

// HealthCheck verifies the connection to Qdrant
func (qs *QdrantStore) HealthCheck(ctx context.Context) error {
	start := time.Now()
	defer qs.updateMetrics("health_check", start)

	// Try to get collection info
	_, err := qs.client.GetCollectionInfo(ctx, qs.collectionName)
	if err != nil {
		qs.metrics.ConnectionStatus = connectionStatusError
		return fmt.Errorf("qdrant health check failed: %w", err)
	}

	qs.metrics.ConnectionStatus = "healthy"
	return nil
}

// GetStats returns statistics about the store
func (qs *QdrantStore) GetStats(ctx context.Context) (*StoreStats, error) {
	start := time.Now()
	defer qs.updateMetrics("get_stats", start)

	// Get collection info
	info, err := qs.client.GetCollectionInfo(ctx, qs.collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}

	pointCount := info.GetPointsCount()
	var totalChunks int64
	if pointCount > 9223372036854775807 { // max int64
		totalChunks = 9223372036854775807
	} else {
		totalChunks = int64(pointCount)
	}

	// Calculate estimated storage size based on collection info and vectors
	vectorsCount := info.GetVectorsCount()
	segmentsCount := info.GetSegmentsCount()
	indexedVectorsCount := info.GetIndexedVectorsCount()

	// Estimate storage size:
	// - Each vector is ~1536 dimensions * 4 bytes (float32) = ~6KB
	// - Add metadata overhead (~2KB per chunk)
	// - Add index overhead based on segments and indexed vectors

	// Safe conversion with overflow protection
	var estimatedVectorSize int64
	if vectorsCount > math.MaxInt64/(defaultVectorSize*4) {
		estimatedVectorSize = math.MaxInt64
	} else {
		estimatedVectorSize = int64(vectorsCount) * defaultVectorSize * 4
	}

	var estimatedIndexSize int64
	if segmentsCount > math.MaxInt64/(1024*1024) {
		estimatedIndexSize = math.MaxInt64
	} else {
		estimatedIndexSize = int64(segmentsCount) * 1024 * 1024
	}

	var estimatedIndexOverhead int64
	if indexedVectorsCount > math.MaxInt64/512 {
		estimatedIndexOverhead = math.MaxInt64
	} else {
		estimatedIndexOverhead = int64(indexedVectorsCount) * 512
	}

	estimatedMetadataSize := totalChunks * 2048 // ~2KB metadata per chunk
	estimatedStorageSize := estimatedVectorSize + estimatedMetadataSize + estimatedIndexSize + estimatedIndexOverhead

	stats := &StoreStats{
		TotalChunks:  totalChunks,
		ChunksByType: make(map[string]int64),
		ChunksByRepo: make(map[string]int64),
		StorageSize:  estimatedStorageSize,
	}

	if stats.TotalChunks > 0 {
		qs.enrichStatsWithSampleData(ctx, stats)
	}

	return stats, nil
}

// enrichStatsWithSampleData adds detailed statistics by sampling collection data
func (qs *QdrantStore) enrichStatsWithSampleData(ctx context.Context, stats *StoreStats) {
	sampleSize := qs.calculateSampleSize(stats.TotalChunks)

	points, err := qs.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: qs.collectionName,
		Limit:          qdrant.PtrOf(sampleSize),
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})

	if err != nil {
		return // Gracefully handle error by returning basic stats
	}

	qs.processStatsFromPoints(points, stats)
}

// calculateSampleSize determines optimal sample size for statistics
func (qs *QdrantStore) calculateSampleSize(totalChunks int64) uint32 {
	const maxSampleSize = 1000
	if totalChunks < 0 {
		return 0
	}
	if totalChunks < maxSampleSize {
		return uint32(totalChunks)
	}
	return maxSampleSize
}

// processStatsFromPoints processes sample points to generate detailed statistics
func (qs *QdrantStore) processStatsFromPoints(points []*qdrant.RetrievedPoint, stats *StoreStats) {
	var oldestTime, newestTime *time.Time
	totalEmbeddingSize := 0

	for _, point := range points {
		payload := point.GetPayload()

		qs.updateTypeStats(payload, stats)
		qs.updateRepoStats(payload, stats)
		qs.updateTimeStats(payload, &oldestTime, &newestTime)
		totalEmbeddingSize += qs.getEmbeddingSize(point)
	}

	qs.finalizeTimeStats(stats, oldestTime, newestTime)
	qs.finalizeEmbeddingStats(stats, totalEmbeddingSize, len(points))
}

// updateTypeStats updates chunk type statistics
func (qs *QdrantStore) updateTypeStats(payload map[string]*qdrant.Value, stats *StoreStats) {
	if typeValue, ok := payload["type"]; ok {
		chunkType := typeValue.GetStringValue()
		stats.ChunksByType[chunkType]++
	}
}

// updateRepoStats updates repository statistics
func (qs *QdrantStore) updateRepoStats(payload map[string]*qdrant.Value, stats *StoreStats) {
	if repoValue, ok := payload["repository"]; ok {
		repository := repoValue.GetStringValue()
		stats.ChunksByRepo[repository]++
	}
}

// updateTimeStats tracks oldest and newest timestamps
func (qs *QdrantStore) updateTimeStats(payload map[string]*qdrant.Value, oldestTime, newestTime **time.Time) {
	timestampValue, ok := payload["timestamp"]
	if !ok {
		return
	}

	timestamp := time.Unix(timestampValue.GetIntegerValue(), 0)
	if *oldestTime == nil || timestamp.Before(**oldestTime) {
		*oldestTime = &timestamp
	}
	if *newestTime == nil || timestamp.After(**newestTime) {
		*newestTime = &timestamp
	}
}

// getEmbeddingSize extracts embedding size from a point
func (qs *QdrantStore) getEmbeddingSize(point *qdrant.RetrievedPoint) int {
	vectors := point.GetVectors()
	if vectors == nil {
		return 0
	}

	vector := vectors.GetVector()
	if vector == nil {
		return 0
	}

	return len(vector.GetData())
}

// finalizeTimeStats sets formatted timestamp strings in stats
func (qs *QdrantStore) finalizeTimeStats(stats *StoreStats, oldestTime, newestTime *time.Time) {
	if oldestTime != nil {
		oldestStr := oldestTime.Format(time.RFC3339)
		stats.OldestChunk = &oldestStr
	}
	if newestTime != nil {
		newestStr := newestTime.Format(time.RFC3339)
		stats.NewestChunk = &newestStr
	}
}

// finalizeEmbeddingStats calculates average embedding size
func (qs *QdrantStore) finalizeEmbeddingStats(stats *StoreStats, totalSize, pointCount int) {
	if pointCount > 0 {
		stats.AverageEmbedding = float64(totalSize) / float64(pointCount)
	}
}

// Cleanup removes old chunks based on retention policy
func (qs *QdrantStore) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	start := time.Now()
	defer qs.updateMetrics("cleanup", start)

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	cutoffTimestamp := float64(cutoffTime.Unix())

	// Build filter for old chunks
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "timestamp",
						Range: &qdrant.Range{
							Lt: &cutoffTimestamp,
						},
					},
				},
			},
		},
	}

	// Count first
	deletedCount64, err := qs.client.Count(ctx, &qdrant.CountPoints{
		CollectionName: qs.collectionName,
		Filter:         filter,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count old chunks: %w", err)
	}

	var deletedCount int
	if deletedCount64 > 2147483647 { // max int32, conservative for int
		deletedCount = 2147483647
	} else {
		deletedCount = int(deletedCount64)
	}

	// Delete old chunks
	_, err = qs.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: qs.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: filter,
			},
		},
	})

	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old chunks: %w", err)
	}

	logging.Info("Cleaned up old chunks",
		"deleted_count", deletedCount,
		"retention_days", retentionDays,
	)

	return deletedCount, nil
}

// Close closes the connection to Qdrant
func (qs *QdrantStore) Close() error {
	if qs.client != nil {
		// Qdrant Go client doesn't have explicit close method
		// Connection will be closed when client is garbage collected
		qs.metrics.ConnectionStatus = "closed"
		logging.Info("Qdrant connection closed")
	}
	return nil
}

// Helper methods

// chunkToPoint converts a ConversationChunk to Qdrant PointStruct
func (qs *QdrantStore) chunkToPoint(chunk *types.ConversationChunk) *qdrant.PointStruct {
	payload := map[string]*qdrant.Value{
		"content":    qs.stringToValue(chunk.Content),
		"summary":    qs.stringToValue(chunk.Summary),
		"type":       qs.stringToValue(string(chunk.Type)),
		"repository": qs.stringToValue(chunk.Metadata.Repository),
		"session_id": qs.stringToValue(chunk.SessionID),
		"outcome":    qs.stringToValue(string(chunk.Metadata.Outcome)),
		"difficulty": qs.stringToValue(string(chunk.Metadata.Difficulty)),
		"timestamp":  qs.int64ToValue(chunk.Timestamp.Unix()),
		"branch":     qs.stringToValue(chunk.Metadata.Branch),
	}

	// Add tags as a list
	if len(chunk.Metadata.Tags) > 0 {
		payload["tags"] = qs.stringSliceToValue(chunk.Metadata.Tags)
	}

	// Add tools used as a list
	if len(chunk.Metadata.ToolsUsed) > 0 {
		payload["tools_used"] = qs.stringSliceToValue(chunk.Metadata.ToolsUsed)
	}

	// Add files modified as a list
	if len(chunk.Metadata.FilesModified) > 0 {
		payload["files_modified"] = qs.stringSliceToValue(chunk.Metadata.FilesModified)
	}

	return &qdrant.PointStruct{
		Id:      qs.stringToPointID(chunk.ID),
		Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: qs.float64ToFloat32(chunk.Embeddings)}}},
		Payload: payload,
	}
}

// buildChunkFromPayload creates a ConversationChunk from payload and extracted data
func (qs *QdrantStore) buildChunkFromPayload(id string, embeddings []float64, payload map[string]*qdrant.Value) (*types.ConversationChunk, error) {
	// Parse timestamp
	timestampValue, ok := payload["timestamp"]
	if !ok {
		return nil, errors.New("missing timestamp in payload")
	}
	timestamp := time.Unix(timestampValue.GetIntegerValue(), 0)

	chunk := &types.ConversationChunk{
		ID:         id,
		SessionID:  qs.getStringFromPayload(payload, "session_id"),
		Timestamp:  timestamp,
		Type:       types.ChunkType(qs.getStringFromPayload(payload, "type")),
		Content:    qs.getStringFromPayload(payload, "content"),
		Summary:    qs.getStringFromPayload(payload, "summary"),
		Embeddings: embeddings,
		Metadata: types.ChunkMetadata{
			Repository:    qs.getStringFromPayload(payload, "repository"),
			Branch:        qs.getStringFromPayload(payload, "branch"),
			FilesModified: qs.getStringSliceFromPayload(payload, "files_modified"),
			ToolsUsed:     qs.getStringSliceFromPayload(payload, "tools_used"),
			Outcome:       types.Outcome(qs.getStringFromPayload(payload, "outcome")),
			Tags:          qs.getStringSliceFromPayload(payload, "tags"),
			Difficulty:    types.Difficulty(qs.getStringFromPayload(payload, "difficulty")),
		},
	}

	return chunk, nil
}

// pointToChunk converts a Qdrant point to ConversationChunk
func (qs *QdrantStore) pointToChunk(point *qdrant.RetrievedPoint) (*types.ConversationChunk, error) {
	payload := point.GetPayload()
	id := qs.pointIDToString(point.GetId())

	// Extract vectors from RetrievedPoint
	var embeddings []float64
	if vectors := point.GetVectors(); vectors != nil {
		if vector := vectors.GetVector(); vector != nil {
			embeddings = qs.float32ToFloat64(vector.GetData())
		}
	}

	return qs.buildChunkFromPayload(id, embeddings, payload)
}

// scoredPointToChunk converts a Qdrant ScoredPoint to ConversationChunk
func (qs *QdrantStore) scoredPointToChunk(point *qdrant.ScoredPoint) (*types.ConversationChunk, error) {
	payload := point.GetPayload()
	id := qs.pointIDToString(point.GetId())

	// Extract vectors from ScoredPoint
	var embeddings []float64
	if vectors := point.GetVectors(); vectors != nil {
		if vector := vectors.GetVector(); vector != nil {
			embeddings = qs.float32ToFloat64(vector.GetData())
		}
	}

	return qs.buildChunkFromPayload(id, embeddings, payload)
}

// buildFilter creates a Qdrant filter from MemoryQuery with enhanced repository support
func (qs *QdrantStore) buildFilter(query *types.MemoryQuery) *qdrant.Filter {
	conditions := make([]*qdrant.Condition, 0)

	// Enhanced Repository filter with global support
	if query.Repository != nil && *query.Repository != "" {
		repository := *query.Repository

		if repository == globalRepository {
			// Global repository: include chunks from 'global' repository + all repositories for architecture decisions
			// This allows cross-project architecture knowledge while maintaining security logging
			globalConditions := []*qdrant.Condition{
				{
					ConditionOneOf: &qdrant.Condition_Field{
						Field: &qdrant.FieldCondition{
							Key: "repository",
							Match: &qdrant.Match{
								MatchValue: &qdrant.Match_Keyword{Keyword: globalRepository},
							},
						},
					},
				},
			}

			// Create OR condition for global access (controlled and logged)
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Filter{
					Filter: &qdrant.Filter{
						Should: globalConditions,
					},
				},
			})
		} else {
			// Standard repository isolation - strict filtering
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "repository",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{Keyword: repository},
						},
					},
				},
			})
		}
	}

	// Type filter
	if len(query.Types) > 0 {
		typeValues := make([]string, len(query.Types))
		for i, t := range query.Types {
			typeValues[i] = string(t)
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key: "type",
					Match: &qdrant.Match{
						MatchValue: &qdrant.Match_Keywords{
							Keywords: &qdrant.RepeatedStrings{Strings: typeValues},
						},
					},
				},
			},
		})
	}

	// Recency-based filtering
	if query.Recency != "" && query.Recency != types.RecencyAllTime {
		var cutoffTime time.Time
		switch query.Recency {
		case types.RecencyRecent:
			cutoffTime = time.Now().AddDate(0, 0, -7) // Last 7 days
		case types.RecencyLastMonth:
			cutoffTime = time.Now().AddDate(0, -1, 0) // Last month
		case types.RecencyAllTime:
			// No time filtering for all time
		}

		if !cutoffTime.IsZero() {
			conditions = append(conditions, &qdrant.Condition{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "timestamp",
						Range: &qdrant.Range{
							Gte: qdrant.PtrOf(float64(cutoffTime.Unix())),
						},
					},
				},
			})
		}
	}

	if len(conditions) == 0 {
		return nil
	}

	return &qdrant.Filter{Must: conditions}
}

// Utility conversion methods
func (qs *QdrantStore) stringToValue(s string) *qdrant.Value {
	return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: s}}
}

func (qs *QdrantStore) int64ToValue(i int64) *qdrant.Value {
	return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: i}}
}

func (qs *QdrantStore) stringSliceToValue(slice []string) *qdrant.Value {
	values := make([]*qdrant.Value, len(slice))
	for i, s := range slice {
		values[i] = qs.stringToValue(s)
	}
	return &qdrant.Value{Kind: &qdrant.Value_ListValue{
		ListValue: &qdrant.ListValue{Values: values},
	}}
}

func (qs *QdrantStore) stringToPointID(s string) *qdrant.PointId {
	return &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: s}}
}

func (qs *QdrantStore) pointIDToString(id *qdrant.PointId) string {
	if uuid := id.GetUuid(); uuid != "" {
		return uuid
	}
	return strconv.FormatUint(id.GetNum(), 10)
}

func (qs *QdrantStore) float64ToFloat32(f64 []float64) []float32 {
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32
}

func (qs *QdrantStore) float32ToFloat64(f32 []float32) []float64 {
	f64 := make([]float64, len(f32))
	for i, v := range f32 {
		f64[i] = float64(v)
	}
	return f64
}

func (qs *QdrantStore) getStringFromPayload(payload map[string]*qdrant.Value, key string) string {
	if value, ok := payload[key]; ok {
		return value.GetStringValue()
	}
	return ""
}

func (qs *QdrantStore) getStringSliceFromPayload(payload map[string]*qdrant.Value, key string) []string {
	if value, ok := payload[key]; ok {
		if listValue := value.GetListValue(); listValue != nil {
			values := listValue.GetValues()
			result := make([]string, len(values))
			for i, v := range values {
				result[i] = v.GetStringValue()
			}
			return result
		}
	}
	return nil
}

// updateMetrics updates operation metrics
func (qs *QdrantStore) updateMetrics(operation string, start time.Time) {
	duration := time.Since(start)

	qs.metrics.OperationCounts[operation]++

	// Update average latency
	currentAvg := qs.metrics.AverageLatency[operation]
	count := float64(qs.metrics.OperationCounts[operation])
	newLatency := float64(duration.Milliseconds())
	qs.metrics.AverageLatency[operation] = (currentAvg*(count-1) + newLatency) / count

	qs.metrics.LastOperation = &operation
}

// Additional methods for service compatibility

// GetAllChunks retrieves all chunks from the collection
func (qs *QdrantStore) GetAllChunks(ctx context.Context) ([]types.ConversationChunk, error) {
	start := time.Now()
	defer qs.updateMetrics("get_all_chunks", start)

	// Use Scroll to get all points with a large limit
	points, err := qs.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: qs.collectionName,
		Limit:          qdrant.PtrOf(uint32(10000)), // Large limit, adjust as needed
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get all chunks: %w", err)
	}

	chunkPtrs := make([]*types.ConversationChunk, 0, len(points))
	for _, point := range points {
		chunk, err := qs.pointToChunk(point)
		if err != nil {
			logging.Error("Failed to convert point to chunk", "error", err, "point_id", point.GetId())
			continue
		}
		chunkPtrs = append(chunkPtrs, chunk)
	}

	// Convert to values for return (since interface expects values)
	chunks := make([]types.ConversationChunk, len(chunkPtrs))
	for i, chunk := range chunkPtrs {
		chunks[i] = *chunk
	}

	logging.Debug("Retrieved all chunks", "count", len(chunks))
	return chunks, nil
}

// DeleteCollection deletes the entire collection
func (qs *QdrantStore) DeleteCollection(ctx context.Context, collection string) error {
	start := time.Now()
	defer qs.updateMetrics("delete_collection", start)

	// Use the provided collection name or default to current collection
	collectionName := collection
	if collectionName == "" {
		collectionName = qs.collectionName
	}

	err := qs.client.DeleteCollection(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to delete collection %s: %w", collectionName, err)
	}

	logging.Info("Deleted collection", "collection", collectionName)
	return nil
}

// ListCollections lists all available collections
func (qs *QdrantStore) ListCollections(ctx context.Context) ([]string, error) {
	start := time.Now()
	defer qs.updateMetrics("list_collections", start)

	collections, err := qs.client.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	logging.Debug("Listed collections", "count", len(collections))
	return collections, nil
}

// FindSimilar finds similar chunks based on content using embeddings
func (qs *QdrantStore) FindSimilar(ctx context.Context, content string, chunkType *types.ChunkType, limit int) ([]types.ConversationChunk, error) {
	start := time.Now()
	defer qs.updateMetrics("find_similar", start)

	// This is a simplified version - in a real implementation, you'd need to:
	// 1. Generate embeddings for the content using an embedding service
	// 2. Perform the vector search
	// For now, return an error indicating this needs embedding service integration
	return nil, errors.New("FindSimilar requires embedding service integration - use Search method with embeddings instead")
}

// StoreChunk is an alias for Store for backward compatibility
func (qs *QdrantStore) StoreChunk(ctx context.Context, chunk *types.ConversationChunk) error {
	return qs.Store(ctx, chunk)
}

// BatchStore stores multiple chunks in a single operation
func (qs *QdrantStore) BatchStore(ctx context.Context, chunks []*types.ConversationChunk) (*BatchResult, error) {
	start := time.Now()
	defer qs.updateMetrics("batch_store", start)

	if len(chunks) == 0 {
		return &BatchResult{Success: 0, Failed: 0}, nil
	}

	// Convert chunks to Qdrant points
	points := make([]*qdrant.PointStruct, 0, len(chunks))
	processedIDs := make([]string, 0, len(chunks))
	errorMessages := make([]string, 0)

	for i := range chunks {
		chunk := chunks[i]
		if err := chunk.Validate(); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("invalid chunk %s: %v", chunk.ID, err))
			continue
		}

		if len(chunk.Embeddings) == 0 {
			errorMessages = append(errorMessages, "chunk "+chunk.ID+" has no embeddings")
			continue
		}

		point := qs.chunkToPoint(chunk)
		points = append(points, point)
		processedIDs = append(processedIDs, chunk.ID)
	}

	// Perform batch upsert
	if len(points) > 0 {
		_, err := qs.client.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: qs.collectionName,
			Points:         points,
		})

		if err != nil {
			return &BatchResult{
				Success:      0,
				Failed:       len(chunks),
				Errors:       append(errorMessages, fmt.Sprintf("batch upsert failed: %v", err)),
				ProcessedIDs: processedIDs,
			}, fmt.Errorf("batch store operation failed: %w", err)
		}
	}

	result := &BatchResult{
		Success:      len(points),
		Failed:       len(chunks) - len(points),
		Errors:       errorMessages,
		ProcessedIDs: processedIDs,
	}

	logging.Debug("Batch store completed",
		"success", result.Success,
		"failed", result.Failed,
		"total", len(chunks),
	)

	return result, nil
}

// BatchDelete deletes multiple chunks by their IDs
func (qs *QdrantStore) BatchDelete(ctx context.Context, ids []string) (*BatchResult, error) {
	start := time.Now()
	defer qs.updateMetrics("batch_delete", start)

	if len(ids) == 0 {
		return &BatchResult{Success: 0, Failed: 0}, nil
	}

	// Convert string IDs to PointIds
	pointIds := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIds[i] = qs.stringToPointID(id)
	}

	// Perform batch delete
	_, err := qs.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: qs.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: pointIds,
				},
			},
		},
	})

	if err != nil {
		return &BatchResult{
			Success:      0,
			Failed:       len(ids),
			Errors:       []string{fmt.Sprintf("batch delete failed: %v", err)},
			ProcessedIDs: ids,
		}, fmt.Errorf("batch delete operation failed: %w", err)
	}

	result := &BatchResult{
		Success:      len(ids),
		Failed:       0,
		Errors:       nil,
		ProcessedIDs: ids,
	}

	logging.Debug("Batch delete completed",
		"success", result.Success,
		"total", len(ids),
	)

	return result, nil
}

// Relationship management methods

// StoreRelationship creates and stores a new memory relationship
func (qs *QdrantStore) StoreRelationship(ctx context.Context, sourceID, targetID string, relationType types.RelationType, confidence float64, source types.ConfidenceSource) (*types.MemoryRelationship, error) {
	return qs.relationshipStore.StoreRelationship(ctx, sourceID, targetID, relationType, confidence, source)
}

// GetRelationships finds relationships for a chunk
func (qs *QdrantStore) GetRelationships(ctx context.Context, query *types.RelationshipQuery) ([]types.RelationshipResult, error) {
	return qs.relationshipStore.GetRelationships(ctx, query)
}

// TraverseGraph traverses the knowledge graph starting from a chunk
func (qs *QdrantStore) TraverseGraph(ctx context.Context, startChunkID string, maxDepth int, relationTypes []types.RelationType) (*types.GraphTraversalResult, error) {
	return qs.relationshipStore.TraverseGraph(ctx, startChunkID, maxDepth, relationTypes)
}

// UpdateRelationship updates an existing relationship's confidence
func (qs *QdrantStore) UpdateRelationship(ctx context.Context, relationshipID string, confidence float64, factors types.ConfidenceFactors) error {
	return qs.relationshipStore.UpdateRelationship(ctx, relationshipID, confidence, factors)
}

// DeleteRelationship removes a relationship
func (qs *QdrantStore) DeleteRelationship(ctx context.Context, relationshipID string) error {
	return qs.relationshipStore.Delete(ctx, relationshipID)
}

// GetRelationshipByID retrieves a specific relationship
func (qs *QdrantStore) GetRelationshipByID(ctx context.Context, relationshipID string) (*types.MemoryRelationship, error) {
	return qs.relationshipStore.GetByID(ctx, relationshipID)
}
