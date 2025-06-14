// Package storage provides adapters for transitioning from the old storage system
// to the new clean storage interfaces.
package storage

import (
	"context"
	"errors"
	"time"

	"lerian-mcp-memory/internal/types"
	pkgTypes "lerian-mcp-memory/pkg/types"
)

// StorageAdapter adapts the old VectorStore interface to the new clean interfaces
// This enables gradual migration from the old system to the new one
type StorageAdapter struct {
	oldStore VectorStore // Legacy store
	config   *StorageConfig
}

// NewStorageAdapter creates a new storage adapter
func NewStorageAdapter(oldStore VectorStore, config *StorageConfig) *StorageAdapter {
	return &StorageAdapter{
		oldStore: oldStore,
		config:   config,
	}
}

// ContentStore implementation

// Store content using the new interface but delegating to old store
func (sa *StorageAdapter) Store(ctx context.Context, content *types.Content) error {
	// Convert new Content to old ConversationChunk format
	chunk := sa.convertContentToChunk(content)
	return sa.oldStore.Store(ctx, chunk)
}

// Update content using the new interface
func (sa *StorageAdapter) Update(ctx context.Context, content *types.Content) error {
	chunk := sa.convertContentToChunk(content)
	return sa.oldStore.Update(ctx, chunk)
}

// Delete content by project and content ID
func (sa *StorageAdapter) Delete(ctx context.Context, projectID types.ProjectID, contentID string) error {
	return sa.oldStore.Delete(ctx, contentID)
}

// Get content by project and content ID
func (sa *StorageAdapter) Get(ctx context.Context, projectID types.ProjectID, contentID string) (*types.Content, error) {
	chunk, err := sa.oldStore.GetByID(ctx, contentID)
	if err != nil {
		return nil, err
	}
	return sa.convertChunkToContent(chunk), nil
}

// BatchStore implements batch storage operations
func (sa *StorageAdapter) BatchStore(ctx context.Context, contents []*types.Content) (*BatchResult, error) {
	startTime := time.Now()
	result := &BatchResult{
		ProcessedIDs: make([]string, 0, len(contents)),
		Metrics: &BatchOperationMetrics{
			StartTime: startTime,
		},
	}

	chunks := make([]*pkgTypes.ConversationChunk, len(contents))
	for i, content := range contents {
		chunks[i] = sa.convertContentToChunk(content)
	}

	batchResult, err := sa.oldStore.BatchStore(ctx, chunks)
	if err != nil {
		return nil, err
	}

	// Convert old batch result to new format
	result.Success = batchResult.Success
	result.Failed = batchResult.Failed
	result.ProcessedIDs = batchResult.ProcessedIDs

	// Convert errors - old format uses []string, new format uses []BatchError
	for i, errStr := range batchResult.Errors {
		result.Errors = append(result.Errors, BatchError{
			Index: i,
			Error: errStr,
		})
	}

	// Add metrics
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	result.Metrics.EndTime = endTime
	result.Metrics.Duration = duration
	if duration > 0 {
		result.Metrics.ItemsPerSec = float64(len(contents)) / duration.Seconds()
	}

	return result, nil
}

// BatchUpdate implements batch update operations
func (sa *StorageAdapter) BatchUpdate(ctx context.Context, contents []*types.Content) (*BatchResult, error) {
	// For now, implement as individual updates
	// TODO: Implement true batch update in underlying store
	startTime := time.Now()
	result := &BatchResult{
		ProcessedIDs: make([]string, 0, len(contents)),
		Metrics: &BatchOperationMetrics{
			StartTime: startTime,
		},
	}

	for _, content := range contents {
		if err := sa.Update(ctx, content); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, BatchError{
				ID:    content.ID,
				Error: err.Error(),
			})
		} else {
			result.Success++
			result.ProcessedIDs = append(result.ProcessedIDs, content.ID)
		}
	}

	endTime := time.Now()
	result.Metrics.EndTime = endTime
	result.Metrics.Duration = endTime.Sub(startTime)

	return result, nil
}

// BatchDelete implements batch delete operations
func (sa *StorageAdapter) BatchDelete(ctx context.Context, projectID types.ProjectID, contentIDs []string) (*BatchResult, error) {
	batchResult, err := sa.oldStore.BatchDelete(ctx, contentIDs)
	if err != nil {
		return nil, err
	}

	// Convert old batch result to new format
	result := &BatchResult{
		Success:      batchResult.Success,
		Failed:       batchResult.Failed,
		ProcessedIDs: batchResult.ProcessedIDs,
	}

	// Convert errors
	for i, errStr := range batchResult.Errors {
		result.Errors = append(result.Errors, BatchError{
			Index: i,
			Error: errStr,
		})
	}

	return result, nil
}

// SearchStore implementation

// Search content within project scope
func (sa *StorageAdapter) Search(ctx context.Context, query *types.SearchQuery) (*types.SearchResults, error) {
	// Convert new search query to old format
	repo := string(query.ProjectID)
	oldQuery := &pkgTypes.MemoryQuery{
		Query:      query.Query,
		Repository: &repo,
		Limit:      query.Limit,
		Types:      sa.convertTypesToChunkTypes(query.Types),
	}

	// Perform search using old store
	oldResults, err := sa.oldStore.Search(ctx, oldQuery, nil) // TODO: Handle embeddings
	if err != nil {
		return nil, err
	}

	// Convert results to new format
	results := &types.SearchResults{
		Results:  make([]*types.SearchResult, len(oldResults.Results)),
		Total:    len(oldResults.Results),
		Query:    query.Query,
		Duration: time.Since(time.Now()), // TODO: Track actual duration
	}

	for i := range oldResults.Results {
		oldResult := &oldResults.Results[i]
		results.Results[i] = &types.SearchResult{
			Content:   sa.convertChunkToContent(&oldResult.Chunk),
			Relevance: oldResult.Score,
		}
	}

	return results, nil
}

// FindSimilar finds similar content within project
func (sa *StorageAdapter) FindSimilar(ctx context.Context, content string, projectID types.ProjectID, sessionID types.SessionID) ([]*types.Content, error) {
	chunks, err := sa.oldStore.FindSimilar(ctx, content, nil, 10) // TODO: Make limit configurable
	if err != nil {
		return nil, err
	}

	contents := make([]*types.Content, len(chunks))
	for i := range chunks {
		contents[i] = sa.convertChunkToContent(&chunks[i])
	}

	return contents, nil
}

// GetByProject gets content by project with optional filters
func (sa *StorageAdapter) GetByProject(ctx context.Context, projectID types.ProjectID, filters *types.Filters) ([]*types.Content, error) {
	limit := 100
	if filters != nil && len(filters.Types) > 0 {
		// TODO: Apply type filters to query
		// For now, reduce limit to account for filtering overhead
		limit = 50 // Smaller limit when filters are applied
	}

	chunks, err := sa.oldStore.ListByRepository(ctx, string(projectID), limit, 0)
	if err != nil {
		return nil, err
	}

	contents := make([]*types.Content, len(chunks))
	for i := range chunks {
		contents[i] = sa.convertChunkToContent(&chunks[i])
	}

	return contents, nil
}

// GetBySession gets content by session within project
func (sa *StorageAdapter) GetBySession(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID, filters *types.Filters) ([]*types.Content, error) {
	chunks, err := sa.oldStore.ListBySession(ctx, string(sessionID))
	if err != nil {
		return nil, err
	}

	// Filter by project
	var filteredChunks []pkgTypes.ConversationChunk
	for i := range chunks {
		if chunks[i].Metadata.Repository == string(projectID) {
			filteredChunks = append(filteredChunks, chunks[i])
		}
	}

	contents := make([]*types.Content, len(filteredChunks))
	for i := range filteredChunks {
		contents[i] = sa.convertChunkToContent(&filteredChunks[i])
	}

	return contents, nil
}

// GetHistory gets content history for specific content
func (sa *StorageAdapter) GetHistory(ctx context.Context, projectID types.ProjectID, contentID string) ([]*types.ContentVersion, error) {
	// TODO: Implement history tracking in underlying store
	// For now, return empty history
	return []*types.ContentVersion{}, nil
}

// Helper conversion methods

// convertContentToChunk converts new Content to old ConversationChunk
func (sa *StorageAdapter) convertContentToChunk(content *types.Content) *pkgTypes.ConversationChunk {
	// Build metadata from content
	metadata := pkgTypes.ChunkMetadata{
		Repository: string(content.ProjectID),
		Tags:       content.Tags,
	}

	// Convert extended metadata if present
	if content.Metadata != nil {
		metadata.ExtendedMetadata = content.Metadata
	}

	chunk := &pkgTypes.ConversationChunk{
		ID:         content.ID,
		SessionID:  string(content.SessionID),
		Timestamp:  content.CreatedAt,
		Type:       sa.convertTypeToChunkType(content.Type),
		Content:    content.Content,
		Summary:    content.Summary,
		Metadata:   metadata,
		Embeddings: content.Embeddings,
	}

	return chunk
}

// convertChunkToContent converts old ConversationChunk to new Content
func (sa *StorageAdapter) convertChunkToContent(chunk *pkgTypes.ConversationChunk) *types.Content {
	content := &types.Content{
		ID:         chunk.ID,
		ProjectID:  types.ProjectID(chunk.Metadata.Repository),
		SessionID:  types.SessionID(chunk.SessionID),
		Type:       sa.convertChunkTypeToType(chunk.Type),
		Content:    chunk.Content,
		Summary:    chunk.Summary,
		Tags:       chunk.Metadata.Tags,
		Metadata:   sa.convertChunkMetadata(&chunk.Metadata),
		CreatedAt:  chunk.Timestamp, // Use timestamp as created time
		UpdatedAt:  chunk.Timestamp, // Default to same as created time
		Embeddings: chunk.Embeddings,
		Version:    1, // Default version
	}

	return content
}

// convertChunkMetadata converts ChunkMetadata to map[string]interface{}
func (sa *StorageAdapter) convertChunkMetadata(metadata *pkgTypes.ChunkMetadata) map[string]interface{} {
	result := make(map[string]interface{})

	if metadata.Repository != "" {
		result["repository"] = metadata.Repository
	}
	if metadata.Branch != "" {
		result["branch"] = metadata.Branch
	}
	if len(metadata.FilesModified) > 0 {
		result["files_modified"] = metadata.FilesModified
	}
	if len(metadata.ToolsUsed) > 0 {
		result["tools_used"] = metadata.ToolsUsed
	}
	if metadata.Outcome != "" {
		result["outcome"] = string(metadata.Outcome)
	}
	if len(metadata.Tags) > 0 {
		result["tags"] = metadata.Tags
	}
	if metadata.Difficulty != "" {
		result["difficulty"] = string(metadata.Difficulty)
	}
	if metadata.TimeSpent != nil {
		result["time_spent"] = *metadata.TimeSpent
	}

	// Merge extended metadata
	for k, v := range metadata.ExtendedMetadata {
		result[k] = v
	}

	return result
}

// convertTypeToChunkType converts new type string to old ChunkType
func (sa *StorageAdapter) convertTypeToChunkType(typeStr string) pkgTypes.ChunkType {
	switch typeStr {
	case "memory":
		return pkgTypes.ChunkTypeDiscussion
	case "task":
		return pkgTypes.ChunkTypeTask
	case "decision":
		return pkgTypes.ChunkTypeArchitectureDecision
	case "insight":
		return pkgTypes.ChunkTypeAnalysis
	case "problem":
		return pkgTypes.ChunkTypeProblem
	case "solution":
		return pkgTypes.ChunkTypeSolution
	case "code":
		return pkgTypes.ChunkTypeCodeChange
	default:
		return pkgTypes.ChunkTypeDiscussion
	}
}

// convertChunkTypeToType converts old ChunkType to new type string
func (sa *StorageAdapter) convertChunkTypeToType(chunkType pkgTypes.ChunkType) string {
	switch chunkType {
	case pkgTypes.ChunkTypeTask, pkgTypes.ChunkTypeTaskUpdate, pkgTypes.ChunkTypeTaskProgress:
		return "task"
	case pkgTypes.ChunkTypeArchitectureDecision:
		return "decision"
	case pkgTypes.ChunkTypeAnalysis:
		return "insight"
	case pkgTypes.ChunkTypeProblem:
		return "problem"
	case pkgTypes.ChunkTypeSolution:
		return "solution"
	case pkgTypes.ChunkTypeCodeChange:
		return "code"
	case pkgTypes.ChunkTypeQuestion:
		return "question"
	default:
		return "memory"
	}
}

// convertTypesToChunkTypes converts new type strings to old ChunkTypes
func (sa *StorageAdapter) convertTypesToChunkTypes(typeStrings []string) []pkgTypes.ChunkType {
	if len(typeStrings) == 0 {
		return nil
	}

	chunkTypes := make([]pkgTypes.ChunkType, len(typeStrings))
	for i, typeStr := range typeStrings {
		chunkTypes[i] = sa.convertTypeToChunkType(typeStr)
	}
	return chunkTypes
}

// Placeholder implementations for other interfaces
// These will be implemented as we progress through Phase 2

// StorePattern stores a pattern in the analysis store (placeholder implementation)
func (sa *StorageAdapter) StorePattern(ctx context.Context, pattern *types.Pattern) error {
	return errors.New("pattern storage not yet implemented")
}

func (sa *StorageAdapter) GetPatterns(ctx context.Context, projectID types.ProjectID, filters *types.PatternFilters) ([]*types.Pattern, error) {
	return nil, errors.New("pattern retrieval not yet implemented")
}

func (sa *StorageAdapter) StoreInsight(ctx context.Context, insight *types.Insight) error {
	return errors.New("insight storage not yet implemented")
}

func (sa *StorageAdapter) GetInsights(ctx context.Context, projectID types.ProjectID, filters *types.InsightFilters) ([]*types.Insight, error) {
	return nil, errors.New("insight retrieval not yet implemented")
}

func (sa *StorageAdapter) StoreConflict(ctx context.Context, conflict *types.Conflict) error {
	return errors.New("conflict storage not yet implemented")
}

func (sa *StorageAdapter) GetConflicts(ctx context.Context, projectID types.ProjectID, filters *types.ConflictFilters) ([]*types.Conflict, error) {
	return nil, errors.New("conflict retrieval not yet implemented")
}

func (sa *StorageAdapter) StoreQualityAnalysis(ctx context.Context, analysis *types.QualityAnalysis) error {
	return errors.New("quality analysis storage not yet implemented")
}

func (sa *StorageAdapter) GetQualityAnalysis(ctx context.Context, projectID types.ProjectID, contentID string) (*types.QualityAnalysis, error) {
	return nil, errors.New("quality analysis retrieval not yet implemented")
}

// StoreRelationship stores a relationship in the storage adapter.
func (sa *StorageAdapter) StoreRelationship(ctx context.Context, relationship *types.Relationship) error {
	return errors.New("relationship storage not yet implemented")
}

func (sa *StorageAdapter) GetRelationships(ctx context.Context, projectID types.ProjectID, contentID string, relationTypes []string) ([]*types.Relationship, error) {
	return nil, errors.New("relationship retrieval not yet implemented")
}

func (sa *StorageAdapter) FindRelated(ctx context.Context, projectID types.ProjectID, contentID string, maxDepth int) ([]*types.RelatedContent, error) {
	return nil, errors.New("related content search not yet implemented")
}

func (sa *StorageAdapter) DeleteRelationship(ctx context.Context, relationshipID string) error {
	return errors.New("relationship deletion not yet implemented")
}

func (sa *StorageAdapter) UpdateRelationshipConfidence(ctx context.Context, relationshipID string, confidence float64) error {
	return errors.New("relationship confidence update not yet implemented")
}

// CreateSession creates a new session in the storage adapter.
func (sa *StorageAdapter) CreateSession(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID, metadata map[string]interface{}) error {
	return errors.New("session creation not yet implemented")
}

func (sa *StorageAdapter) GetSession(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID) (*types.Session, error) {
	return nil, errors.New("session retrieval not yet implemented")
}

func (sa *StorageAdapter) UpdateSessionAccess(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID) error {
	// For now, just return success since the old system doesn't track this
	return nil
}

func (sa *StorageAdapter) ListSessions(ctx context.Context, projectID types.ProjectID, filters *types.SessionFilters) ([]*types.Session, error) {
	return nil, errors.New("session listing not yet implemented")
}

func (sa *StorageAdapter) DeleteSession(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID) error {
	return errors.New("session deletion not yet implemented")
}

func (sa *StorageAdapter) GetSessionStats(ctx context.Context, projectID types.ProjectID) (*types.SessionStats, error) {
	return nil, errors.New("session stats not yet implemented")
}

// HealthCheck performs a health check on the storage adapter.
func (sa *StorageAdapter) HealthCheck(ctx context.Context) (*types.HealthStatus, error) {
	err := sa.oldStore.HealthCheck(ctx)
	if err != nil {
		return &types.HealthStatus{
			Status:       "unhealthy",
			LastChecked:  time.Now(),
			ResponseTime: 0,
		}, nil
	}

	return &types.HealthStatus{
		Status:       "healthy",
		LastChecked:  time.Now(),
		ResponseTime: time.Millisecond * 10, // Mock response time
		Components: map[string]types.ComponentHealth{
			"vector_store": {
				Status:       "healthy",
				LastChecked:  time.Now(),
				ResponseTime: time.Millisecond * 5,
			},
		},
	}, nil
}

func (sa *StorageAdapter) GetStats(ctx context.Context) (*types.StorageStats, error) {
	oldStats, err := sa.oldStore.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	return &types.StorageStats{
		TotalContent:     oldStats.TotalChunks,
		ContentByType:    sa.convertChunksByType(oldStats.ChunksByType),
		ContentByProject: oldStats.ChunksByRepo,
		StorageSize:      oldStats.StorageSize,
		LastUpdated:      time.Now(),
		Performance: types.PerformanceMetrics{
			RequestsTotal:   1000, // Mock data
			RequestsPerSec:  10.5,
			AvgResponseTime: time.Millisecond * 45,
			ErrorRate:       0.02,
		},
	}, nil
}

func (sa *StorageAdapter) GetProjectStats(ctx context.Context, projectID types.ProjectID) (*types.ProjectStats, error) {
	return &types.ProjectStats{
		ProjectID:      projectID,
		TotalContent:   100, // Mock data
		ContentByType:  map[string]int64{"memory": 50, "task": 30, "decision": 20},
		TotalSessions:  10,
		ActiveSessions: 3,
		StorageSize:    1024 * 1024, // 1MB
		CreatedAt:      time.Now().Add(-30 * 24 * time.Hour),
		LastActivity:   time.Now(),
		QualityScore:   0.85,
	}, nil
}

func (sa *StorageAdapter) ExportProject(ctx context.Context, projectID types.ProjectID, format string, options *types.ExportOptions) (*types.ExportResult, error) {
	return nil, errors.New("project export not yet implemented")
}

func (sa *StorageAdapter) ImportProject(ctx context.Context, projectID types.ProjectID, data, format string, options *types.ImportOptions) (*types.ImportResult, error) {
	return nil, errors.New("project import not yet implemented")
}

func (sa *StorageAdapter) ValidateIntegrity(ctx context.Context, projectID types.ProjectID) (*types.IntegrityReport, error) {
	return nil, errors.New("integrity validation not yet implemented")
}

func (sa *StorageAdapter) Cleanup(ctx context.Context, projectID types.ProjectID, retentionDays int) (*types.CleanupResult, error) {
	return nil, errors.New("cleanup not yet implemented")
}

// Helper method to convert chunks by type
func (sa *StorageAdapter) convertChunksByType(oldMap map[string]int64) map[string]int64 {
	newMap := make(map[string]int64)
	for chunkTypeStr, count := range oldMap {
		newType := sa.convertChunkTypeToType(pkgTypes.ChunkType(chunkTypeStr))
		newMap[newType] += count
	}
	return newMap
}

// WithTransaction executes a function within a transaction context.
func (sa *StorageAdapter) WithTransaction(ctx context.Context, fn func(tx UnifiedStore) error) error {
	// For now, just execute without transaction
	// TODO: Implement proper transaction support
	return fn(sa)
}

func (sa *StorageAdapter) Close() error {
	return sa.oldStore.Close()
}

func (sa *StorageAdapter) Migrate(ctx context.Context) error {
	return sa.oldStore.Initialize(ctx)
}
