package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mcp-memory/internal/config"
	"mcp-memory/pkg/types"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// ChromaStore implements VectorStore interface for Chroma vector database
type ChromaStore struct {
	client     *resty.Client
	config     *config.ChromaConfig
	collection string
	metrics    *StorageMetrics
}

// ChromaDocument represents a document in Chroma format
type ChromaDocument struct {
	ID         string                 `json:"id"`
	Embedding  []float64              `json:"embedding"`
	Document   string                 `json:"document"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ChromaResponse represents a response from Chroma API
type ChromaResponse struct {
	IDs       []string                 `json:"ids"`
	Documents []string                 `json:"documents"`
	Metadatas []map[string]interface{} `json:"metadatas"`
	Distances []float64                `json:"distances,omitempty"`
}

// ChromaCollection represents a Chroma collection
type ChromaCollection struct {
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NewChromaStore creates a new Chroma vector store
func NewChromaStore(cfg *config.ChromaConfig) *ChromaStore {
	client := resty.New()
	client.SetBaseURL(cfg.Endpoint)
	client.SetTimeout(time.Duration(cfg.TimeoutSeconds) * time.Second)
	client.SetRetryCount(cfg.RetryAttempts)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(5 * time.Second)

	// Set up error handling
	client.OnError(func(req *resty.Request, err error) {
		log.Printf("Chroma request error: %v", err)
	})

	return &ChromaStore{
		client:     client,
		config:     cfg,
		collection: cfg.Collection,
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

	// Check if collection exists
	resp, err := cs.client.R().
		SetContext(ctx).
		Get("/api/v1/collections")

	if err != nil {
		cs.metrics.ConnectionStatus = "error"
		return fmt.Errorf("failed to list collections: %w", err)
	}

	var collections []ChromaCollection
	if err := json.Unmarshal(resp.Body(), &collections); err != nil {
		return fmt.Errorf("failed to parse collections response: %w", err)
	}

	// Check if our collection exists
	for _, coll := range collections {
		if coll.Name == cs.collection {
			cs.metrics.ConnectionStatus = "connected"
			return nil
		}
	}

	// Create collection if it doesn't exist
	createReq := map[string]interface{}{
		"name": cs.collection,
		"metadata": map[string]interface{}{
			"description": "Claude conversation memory storage",
			"created_at":  time.Now().UTC().Format(time.RFC3339),
		},
	}

	resp, err = cs.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(createReq).
		Post("/api/v1/collections")

	if err != nil {
		cs.metrics.ConnectionStatus = "error"
		return fmt.Errorf("failed to create collection: %w", err)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 201 {
		return fmt.Errorf("failed to create collection, status: %d, body: %s", resp.StatusCode(), string(resp.Body()))
	}

	cs.metrics.ConnectionStatus = "connected"
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

	// Convert chunk to Chroma document
	doc := cs.chunkToDocument(chunk)

	// Prepare request
	addReq := map[string]interface{}{
		"ids":        []string{doc.ID},
		"embeddings": [][]float64{doc.Embedding},
		"documents":  []string{doc.Document},
		"metadatas":  []map[string]interface{}{doc.Metadata},
	}

	resp, err := cs.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(addReq).
		Post(fmt.Sprintf("/api/v1/collections/%s/add", cs.collection))

	if err != nil {
		cs.updateMetrics("store", start, err)
		return fmt.Errorf("failed to store chunk: %w", err)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 201 {
		err := fmt.Errorf("store failed with status %d: %s", resp.StatusCode(), string(resp.Body()))
		cs.updateMetrics("store", start, err)
		return err
	}

	return nil
}

// Search finds similar chunks based on query embeddings
func (cs *ChromaStore) Search(ctx context.Context, query types.MemoryQuery, embeddings []float64) (*types.SearchResults, error) {
	start := time.Now()
	defer cs.updateMetrics("search", start, nil)

	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("embeddings required for search")
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	// Build where clause for metadata filtering
	whereClause := cs.buildWhereClause(query)

	searchReq := map[string]interface{}{
		"query_embeddings": [][]float64{embeddings},
		"n_results":        limit,
	}

	if whereClause != nil {
		searchReq["where"] = whereClause
	}

	resp, err := cs.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(searchReq).
		Post(fmt.Sprintf("/api/v1/collections/%s/query", cs.collection))

	if err != nil {
		cs.updateMetrics("search", start, err)
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	if resp.StatusCode() != 200 {
		err := fmt.Errorf("search failed with status %d: %s", resp.StatusCode(), string(resp.Body()))
		cs.updateMetrics("search", start, err)
		return nil, err
	}

	var chromaResp ChromaResponse
	if err := json.Unmarshal(resp.Body(), &chromaResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	// Convert response to search results
	results := &types.SearchResults{
		Results:   []types.SearchResult{},
		Total:     len(chromaResp.IDs),
		QueryTime: time.Since(start),
	}

	for i, id := range chromaResp.IDs {
		chunk, err := cs.documentToChunk(id, chromaResp.Documents[i], chromaResp.Metadatas[i])
		if err != nil {
			log.Printf("Failed to convert document to chunk: %v", err)
			continue
		}

		score := 1.0 // Default score
		if i < len(chromaResp.Distances) {
			// Convert distance to similarity score (1 - normalized_distance)
			score = 1.0 - chromaResp.Distances[i]
		}

		// Filter by minimum relevance score
		if score >= query.MinRelevanceScore {
			results.Results = append(results.Results, types.SearchResult{
				Chunk: *chunk,
				Score: score,
			})
		}
	}

	return results, nil
}

// GetByID retrieves a chunk by its ID
func (cs *ChromaStore) GetByID(ctx context.Context, id string) (*types.ConversationChunk, error) {
	start := time.Now()
	defer cs.updateMetrics("get_by_id", start, nil)

	getReq := map[string]interface{}{
		"ids": []string{id},
	}

	resp, err := cs.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(getReq).
		Post(fmt.Sprintf("/api/v1/collections/%s/get", cs.collection))

	if err != nil {
		cs.updateMetrics("get_by_id", start, err)
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	if resp.StatusCode() != 200 {
		err := fmt.Errorf("get failed with status %d: %s", resp.StatusCode(), string(resp.Body()))
		cs.updateMetrics("get_by_id", start, err)
		return nil, err
	}

	var chromaResp ChromaResponse
	if err := json.Unmarshal(resp.Body(), &chromaResp); err != nil {
		return nil, fmt.Errorf("failed to parse get response: %w", err)
	}

	if len(chromaResp.IDs) == 0 {
		return nil, fmt.Errorf("chunk not found: %s", id)
	}

	return cs.documentToChunk(chromaResp.IDs[0], chromaResp.Documents[0], chromaResp.Metadatas[0])
}

// ListByRepository lists chunks for a specific repository
func (cs *ChromaStore) ListByRepository(ctx context.Context, repository string, limit int, offset int) ([]types.ConversationChunk, error) {
	start := time.Now()
	defer cs.updateMetrics("list_by_repository", start, nil)

	if limit <= 0 {
		limit = 10
	}

	whereClause := map[string]interface{}{
		"repository": repository,
	}

	getReq := map[string]interface{}{
		"where": whereClause,
		"limit": limit,
		"offset": offset,
	}

	resp, err := cs.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(getReq).
		Post(fmt.Sprintf("/api/v1/collections/%s/get", cs.collection))

	if err != nil {
		cs.updateMetrics("list_by_repository", start, err)
		return nil, fmt.Errorf("failed to list chunks: %w", err)
	}

	if resp.StatusCode() != 200 {
		err := fmt.Errorf("list failed with status %d: %s", resp.StatusCode(), string(resp.Body()))
		cs.updateMetrics("list_by_repository", start, err)
		return nil, err
	}

	var chromaResp ChromaResponse
	if err := json.Unmarshal(resp.Body(), &chromaResp); err != nil {
		return nil, fmt.Errorf("failed to parse list response: %w", err)
	}

	chunks := make([]types.ConversationChunk, 0, len(chromaResp.IDs))
	for i, id := range chromaResp.IDs {
		chunk, err := cs.documentToChunk(id, chromaResp.Documents[i], chromaResp.Metadatas[i])
		if err != nil {
			log.Printf("Failed to convert document to chunk: %v", err)
			continue
		}
		chunks = append(chunks, *chunk)
	}

	return chunks, nil
}

// Delete removes a chunk by ID
func (cs *ChromaStore) Delete(ctx context.Context, id string) error {
	start := time.Now()
	defer cs.updateMetrics("delete", start, nil)

	deleteReq := map[string]interface{}{
		"ids": []string{id},
	}

	resp, err := cs.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(deleteReq).
		Post(fmt.Sprintf("/api/v1/collections/%s/delete", cs.collection))

	if err != nil {
		cs.updateMetrics("delete", start, err)
		return fmt.Errorf("failed to delete chunk: %w", err)
	}

	if resp.StatusCode() != 200 {
		err := fmt.Errorf("delete failed with status %d: %s", resp.StatusCode(), string(resp.Body()))
		cs.updateMetrics("delete", start, err)
		return err
	}

	return nil
}

// Update modifies an existing chunk
func (cs *ChromaStore) Update(ctx context.Context, chunk types.ConversationChunk) error {
	start := time.Now()
	defer cs.updateMetrics("update", start, nil)

	if err := chunk.Validate(); err != nil {
		return fmt.Errorf("invalid chunk: %w", err)
	}

	// Delete existing chunk first
	if err := cs.Delete(ctx, chunk.ID); err != nil {
		return fmt.Errorf("failed to delete existing chunk: %w", err)
	}

	// Store updated chunk
	return cs.Store(ctx, chunk)
}

// HealthCheck verifies the connection to Chroma
func (cs *ChromaStore) HealthCheck(ctx context.Context) error {
	start := time.Now()
	defer cs.updateMetrics("health_check", start, nil)

	resp, err := cs.client.R().
		SetContext(ctx).
		Get("/api/v1/heartbeat")

	if err != nil {
		cs.metrics.ConnectionStatus = "error"
		cs.updateMetrics("health_check", start, err)
		return fmt.Errorf("health check failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		cs.metrics.ConnectionStatus = "error"
		err := fmt.Errorf("health check failed with status %d", resp.StatusCode())
		cs.updateMetrics("health_check", start, err)
		return err
	}

	cs.metrics.ConnectionStatus = "healthy"
	return nil
}

// GetStats returns statistics about the store
func (cs *ChromaStore) GetStats(ctx context.Context) (*StoreStats, error) {
	start := time.Now()
	defer cs.updateMetrics("get_stats", start, nil)

	// Get collection info
	resp, err := cs.client.R().
		SetContext(ctx).
		Get(fmt.Sprintf("/api/v1/collections/%s", cs.collection))

	if err != nil {
		cs.updateMetrics("get_stats", start, err)
		return nil, fmt.Errorf("failed to get collection stats: %w", err)
	}

	// For now, return basic stats - in a real implementation,
	// we'd need to make additional calls to get detailed statistics
	stats := &StoreStats{
		TotalChunks:  0, // Would need to count all documents
		ChunksByType: make(map[string]int64),
		ChunksByRepo: make(map[string]int64),
		StorageSize:  0, // Would need to calculate from collection info
	}

	return stats, nil
}

// Cleanup removes old chunks based on retention policy
func (cs *ChromaStore) Cleanup(ctx context.Context, retentionDays int) (int, error) {
	start := time.Now()
	defer cs.updateMetrics("cleanup", start, nil)

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	cutoffStr := cutoffTime.Format(time.RFC3339)

	whereClause := map[string]interface{}{
		"timestamp": map[string]interface{}{
			"$lt": cutoffStr,
		},
	}

	// First, get the chunks to delete
	getReq := map[string]interface{}{
		"where": whereClause,
	}

	resp, err := cs.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(getReq).
		Post(fmt.Sprintf("/api/v1/collections/%s/get", cs.collection))

	if err != nil {
		cs.updateMetrics("cleanup", start, err)
		return 0, fmt.Errorf("failed to get old chunks: %w", err)
	}

	var chromaResp ChromaResponse
	if err := json.Unmarshal(resp.Body(), &chromaResp); err != nil {
		return 0, fmt.Errorf("failed to parse cleanup response: %w", err)
	}

	if len(chromaResp.IDs) == 0 {
		return 0, nil
	}

	// Delete the old chunks
	deleteReq := map[string]interface{}{
		"ids": chromaResp.IDs,
	}

	resp, err = cs.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(deleteReq).
		Post(fmt.Sprintf("/api/v1/collections/%s/delete", cs.collection))

	if err != nil {
		cs.updateMetrics("cleanup", start, err)
		return 0, fmt.Errorf("failed to delete old chunks: %w", err)
	}

	if resp.StatusCode() != 200 {
		err := fmt.Errorf("cleanup failed with status %d: %s", resp.StatusCode(), string(resp.Body()))
		cs.updateMetrics("cleanup", start, err)
		return 0, err
	}

	return len(chromaResp.IDs), nil
}

// Close closes the client connection
func (cs *ChromaStore) Close() error {
	// resty.Client doesn't need explicit closing
	cs.metrics.ConnectionStatus = "closed"
	return nil
}

// Helper methods

func (cs *ChromaStore) chunkToDocument(chunk types.ConversationChunk) ChromaDocument {
	// Serialize the chunk content and metadata
	content := fmt.Sprintf("Type: %s\nContent: %s\nSummary: %s", chunk.Type, chunk.Content, chunk.Summary)

	metadata := map[string]interface{}{
		"session_id":   chunk.SessionID,
		"timestamp":    chunk.Timestamp.Format(time.RFC3339),
		"type":         string(chunk.Type),
		"summary":      chunk.Summary,
		"repository":   chunk.Metadata.Repository,
		"branch":       chunk.Metadata.Branch,
		"outcome":      string(chunk.Metadata.Outcome),
		"difficulty":   string(chunk.Metadata.Difficulty),
		"tags":         strings.Join(chunk.Metadata.Tags, ","),
		"tools_used":   strings.Join(chunk.Metadata.ToolsUsed, ","),
		"files_modified": strings.Join(chunk.Metadata.FilesModified, ","),
	}

	if chunk.Metadata.TimeSpent != nil {
		metadata["time_spent"] = *chunk.Metadata.TimeSpent
	}

	return ChromaDocument{
		ID:        chunk.ID,
		Embedding: chunk.Embeddings,
		Document:  content,
		Metadata:  metadata,
	}
}

func (cs *ChromaStore) documentToChunk(id, document string, metadata map[string]interface{}) (*types.ConversationChunk, error) {
	// Parse metadata back to chunk
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

	// Extract content from document (this is a simplified extraction)
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
		if timeSpentFloat, ok := timeSpent.(float64); ok {
			timeSpentInt := int(timeSpentFloat)
			chunkMetadata.TimeSpent = &timeSpentInt
		} else if timeSpentStr, ok := timeSpent.(string); ok {
			if timeSpentInt, err := strconv.Atoi(timeSpentStr); err == nil {
				chunkMetadata.TimeSpent = &timeSpentInt
			}
		}
	}

	chunk.Metadata = chunkMetadata

	return chunk, nil
}

func (cs *ChromaStore) buildWhereClause(query types.MemoryQuery) map[string]interface{} {
	where := make(map[string]interface{})

	if query.Repository != nil {
		where["repository"] = *query.Repository
	}

	if len(query.Types) > 0 {
		typeStrings := make([]string, len(query.Types))
		for i, t := range query.Types {
			typeStrings[i] = string(t)
		}
		where["type"] = map[string]interface{}{
			"$in": typeStrings,
		}
	}

	// Add time-based filtering based on recency
	switch query.Recency {
	case types.RecencyRecent:
		recentTime := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
		where["timestamp"] = map[string]interface{}{
			"$gt": recentTime,
		}
	case types.RecencyLastMonth:
		monthTime := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
		where["timestamp"] = map[string]interface{}{
			"$gt": monthTime,
		}
	}

	if len(where) == 0 {
		return nil
	}

	return where
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