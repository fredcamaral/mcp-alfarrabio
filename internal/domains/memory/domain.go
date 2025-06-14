// Package memory provides the Memory Domain implementation
// for content storage, search, relationships, and intelligence operations.
package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	domainTypes "lerian-mcp-memory/internal/domains/types"
	"lerian-mcp-memory/internal/storage"
	coreTypes "lerian-mcp-memory/internal/types"
)

// Domain implements the MemoryDomain interface
// This is the pure memory domain without task management mixing
type Domain struct {
	contentStore      storage.ContentStore
	searchStore       storage.SearchStore
	analysisStore     storage.AnalysisStore
	relationshipStore storage.RelationshipStore
	config            *Config
}

// Config represents configuration for the memory domain
type Config struct {
	EmbeddingDimension   int           `json:"embedding_dimension"`
	MaxContentSize       int64         `json:"max_content_size"`
	SearchTimeout        time.Duration `json:"search_timeout"`
	RelationshipTimeout  time.Duration `json:"relationship_timeout"`
	AnalysisTimeout      time.Duration `json:"analysis_timeout"`
	CacheEnabled         bool          `json:"cache_enabled"`
	CacheTTL             time.Duration `json:"cache_ttl"`
	AutoDetectRelations  bool          `json:"auto_detect_relations"`
	AutoGenerateInsights bool          `json:"auto_generate_insights"`
}

// DefaultConfig returns default configuration for memory domain
func DefaultConfig() *Config {
	return &Config{
		EmbeddingDimension:   1536,
		MaxContentSize:       10 * 1024 * 1024, // 10MB
		SearchTimeout:        30 * time.Second,
		RelationshipTimeout:  15 * time.Second,
		AnalysisTimeout:      60 * time.Second,
		CacheEnabled:         true,
		CacheTTL:             15 * time.Minute,
		AutoDetectRelations:  true,
		AutoGenerateInsights: false,
	}
}

// NewDomain creates a new memory domain instance
func NewDomain(
	contentStore storage.ContentStore,
	searchStore storage.SearchStore,
	analysisStore storage.AnalysisStore,
	relationshipStore storage.RelationshipStore,
	config *Config,
) *Domain {
	if config == nil {
		config = DefaultConfig()
	}

	return &Domain{
		contentStore:      contentStore,
		searchStore:       searchStore,
		analysisStore:     analysisStore,
		relationshipStore: relationshipStore,
		config:            config,
	}
}

// Content Management Operations

// StoreContent stores new content in the memory system
func (d *Domain) StoreContent(ctx context.Context, req *domainTypes.StoreContentRequest) (*domainTypes.StoreContentResponse, error) {
	startTime := time.Now()

	// Validate request
	if req.Content == "" {
		return nil, errors.New("content is required")
	}

	// Validate content size
	if len(req.Content) > int(d.config.MaxContentSize) {
		return nil, errors.New("content size exceeds maximum allowed size")
	}

	// Create content struct from request
	now := time.Now()
	content := &coreTypes.Content{
		ProjectID: req.ProjectID,
		SessionID: req.SessionID,
		Type:      req.Type,
		Title:     req.Title,
		Content:   req.Content,
		Summary:   req.Summary,
		Tags:      req.Tags,
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}

	// Generate ID if not provided
	if content.ID == "" {
		content.ID = d.generateContentID()
	}

	// Store content
	if err := d.contentStore.Store(ctx, content); err != nil {
		return nil, fmt.Errorf("failed to store content: %w", err)
	}

	// Auto-detect relationships if enabled
	if d.config.AutoDetectRelations {
		// Use context.WithoutCancel to derive context for background task
		relationCtx := context.WithoutCancel(ctx)
		go d.autoDetectRelationships(relationCtx, content)
	}

	return &domainTypes.StoreContentResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   true,
			Message:   "Content stored successfully",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		ContentID: content.ID,
		Version:   content.Version,
		CreatedAt: content.CreatedAt,
	}, nil
}

// UpdateContent updates existing content
func (d *Domain) UpdateContent(ctx context.Context, req *domainTypes.UpdateContentRequest) (*domainTypes.UpdateContentResponse, error) {
	startTime := time.Now()

	existing, err := d.contentStore.Get(ctx, req.ProjectID, req.ContentID)
	if err != nil {
		return nil, fmt.Errorf("content not found: %w", err)
	}

	d.applyContentUpdates(existing, req)
	d.applyTagOperations(existing, req)
	d.applyQualityUpdates(existing, req)
	d.updateVersionAndTimestamp(existing)

	if err := d.validateContentSize(existing); err != nil {
		return nil, err
	}

	if err := d.contentStore.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update content: %w", err)
	}

	return d.buildUpdateResponse(req.ContentID, existing, startTime), nil
}

// applyContentUpdates applies basic content field updates
func (d *Domain) applyContentUpdates(existing *coreTypes.Content, req *domainTypes.UpdateContentRequest) {
	if req.Content != "" {
		existing.Content = req.Content
	}
	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Summary != "" {
		existing.Summary = req.Summary
	}
	if req.Tags != nil {
		existing.Tags = req.Tags
	}

	d.applyMetadataUpdates(existing, req)
}

// applyMetadataUpdates applies metadata updates
func (d *Domain) applyMetadataUpdates(existing *coreTypes.Content, req *domainTypes.UpdateContentRequest) {
	if req.Metadata == nil {
		return
	}

	if existing.Metadata == nil {
		existing.Metadata = make(map[string]interface{})
	}

	for k, v := range req.Metadata {
		existing.Metadata[k] = v
	}
}

// applyTagOperations handles tag addition and removal
func (d *Domain) applyTagOperations(existing *coreTypes.Content, req *domainTypes.UpdateContentRequest) {
	if req.AddTags != nil {
		existing.Tags = append(existing.Tags, req.AddTags...)
	}

	if req.RemoveTags != nil {
		d.removeTags(existing, req.RemoveTags)
	}
}

// removeTags removes specified tags from the content
func (d *Domain) removeTags(existing *coreTypes.Content, tagsToRemove []string) {
	for _, removeTag := range tagsToRemove {
		for i, tag := range existing.Tags {
			if tag == removeTag {
				existing.Tags = append(existing.Tags[:i], existing.Tags[i+1:]...)
				break
			}
		}
	}
}

// applyQualityUpdates applies quality and confidence updates
func (d *Domain) applyQualityUpdates(existing *coreTypes.Content, req *domainTypes.UpdateContentRequest) {
	if req.Quality != nil {
		existing.Quality = *req.Quality
	}
	if req.Confidence != nil {
		existing.Confidence = *req.Confidence
	}
}

// updateVersionAndTimestamp updates version and timestamp
func (d *Domain) updateVersionAndTimestamp(existing *coreTypes.Content) {
	existing.Version++
	existing.UpdatedAt = time.Now()
}

// validateContentSize validates the updated content size
func (d *Domain) validateContentSize(content *coreTypes.Content) error {
	if len(content.Content) > int(d.config.MaxContentSize) {
		return errors.New("updated content size exceeds maximum allowed size")
	}
	return nil
}

// buildUpdateResponse builds the update response
func (d *Domain) buildUpdateResponse(contentID string, existing *coreTypes.Content, startTime time.Time) *domainTypes.UpdateContentResponse {
	return &domainTypes.UpdateContentResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   true,
			Message:   "Content updated successfully",
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		ContentID: contentID,
		Version:   existing.Version,
		UpdatedAt: existing.UpdatedAt,
	}
}

// DeleteContent removes content from the system
func (d *Domain) DeleteContent(ctx context.Context, req *domainTypes.DeleteContentRequest) error {
	// Delete relationships if Force is enabled
	if req.Force {
		// Get relationships first
		relationships, err := d.relationshipStore.GetRelationships(ctx, req.ProjectID, req.ContentID, nil)
		if err == nil {
			// Delete each relationship
			for _, rel := range relationships {
				_ = d.relationshipStore.DeleteRelationship(ctx, rel.ID)
			}
		}
	}

	// Delete the content
	return d.contentStore.Delete(ctx, req.ProjectID, req.ContentID)
}

// GetContent retrieves content by ID
func (d *Domain) GetContent(ctx context.Context, req *domainTypes.GetContentRequest) (*domainTypes.GetContentResponse, error) {
	startTime := time.Now()

	// Get content
	content, err := d.contentStore.Get(ctx, req.ProjectID, req.ContentID)
	if err != nil {
		return nil, fmt.Errorf("content not found: %w", err)
	}

	response := &domainTypes.GetContentResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Content: content,
	}

	// Include relationships if requested
	if req.IncludeRelated {
		d.includeRelatedContent(ctx, req, response)
	}

	// Include history if requested
	if req.IncludeHistory {
		history, err := d.searchStore.GetHistory(ctx, req.ProjectID, req.ContentID)
		if err == nil {
			response.History = history
		}
	}

	return response, nil
}

// includeRelatedContent adds related content to the response
func (d *Domain) includeRelatedContent(ctx context.Context, req *domainTypes.GetContentRequest, response *domainTypes.GetContentResponse) {
	relationships, err := d.relationshipStore.GetRelationships(ctx, req.ProjectID, req.ContentID, nil)
	if err != nil {
		return // Silently ignore relationship errors to not break the main response
	}

	related := d.convertRelationshipsToRelatedContent(ctx, req, relationships)
	response.Related = related
}

// convertRelationshipsToRelatedContent converts relationships to related content
func (d *Domain) convertRelationshipsToRelatedContent(ctx context.Context, req *domainTypes.GetContentRequest, relationships []*coreTypes.Relationship) []*coreTypes.RelatedContent {
	var related []*coreTypes.RelatedContent

	for _, rel := range relationships {
		relatedContentID := d.getRelatedContentID(rel, req.ContentID)
		relatedContent := d.getRelatedContent(ctx, req.ProjectID, relatedContentID)

		if relatedContent != nil {
			related = append(related, &coreTypes.RelatedContent{
				Content:      relatedContent,
				Relationship: rel,
				Distance:     1,
				Relevance:    rel.Confidence,
			})
		}
	}

	return related
}

// getRelatedContentID gets the related content ID from a relationship
func (d *Domain) getRelatedContentID(rel *coreTypes.Relationship, contentID string) string {
	if rel.SourceID == contentID {
		return rel.TargetID
	}
	return rel.SourceID
}

// getRelatedContent gets related content by ID, returning nil on error
func (d *Domain) getRelatedContent(ctx context.Context, projectID coreTypes.ProjectID, contentID string) *coreTypes.Content {
	content, err := d.contentStore.Get(ctx, projectID, contentID)
	if err != nil {
		return nil
	}
	return content
}

// Search and Discovery Operations

// SearchContent performs semantic search across content
func (d *Domain) SearchContent(ctx context.Context, req *domainTypes.SearchContentRequest) (*domainTypes.SearchContentResponse, error) {
	startTime := time.Now()

	// Create search query
	searchQuery := &coreTypes.SearchQuery{
		ProjectID: req.ProjectID,
		SessionID: req.SessionID,
		Query:     req.Query,
		Filters:   req.Filters,
	}

	// Apply options from request
	searchQuery.Limit = req.Limit
	searchQuery.Offset = req.Offset
	searchQuery.MinRelevance = req.MinRelevance
	searchQuery.SortBy = req.SortBy
	searchQuery.SortOrder = req.SortOrder

	// Set context timeout
	ctx, cancel := context.WithTimeout(ctx, d.config.SearchTimeout)
	defer cancel()

	// Execute search
	results, err := d.searchStore.Search(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return &domainTypes.SearchContentResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Results:      results.Results,
		Total:        results.Total,
		Query:        req.Query,
		Duration:     results.Duration,
		MaxRelevance: results.MaxRelevance,
	}, nil
}

// FindSimilarContent finds semantically similar content
func (d *Domain) FindSimilarContent(ctx context.Context, req *domainTypes.FindSimilarRequest) (*domainTypes.FindSimilarResponse, error) {
	startTime := time.Now()

	var content string

	// Get content string
	if req.Content != "" {
		content = req.Content
	} else if req.ContentID != "" {
		existingContent, err := d.contentStore.Get(ctx, req.ProjectID, req.ContentID)
		if err != nil {
			return nil, fmt.Errorf("content not found: %w", err)
		}
		content = existingContent.Content
	} else {
		return nil, errors.New("either content or content_id must be provided")
	}

	// Find similar content
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	similar, err := d.searchStore.FindSimilar(ctx, content, req.ProjectID, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar content: %w", err)
	}

	// Convert to response format
	response := &domainTypes.FindSimilarResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Similar: make([]*domainTypes.SimilarContent, 0, len(similar)),
	}

	for _, s := range similar {
		if len(response.Similar) >= limit {
			break
		}

		// Apply threshold filter
		// TODO: Calculate actual similarity score from vector database
		similarityScore := 0.85 // Placeholder score
		if req.MinSimilarity > 0 && similarityScore < req.MinSimilarity {
			continue // Skip items below threshold
		}

		response.Similar = append(response.Similar, &domainTypes.SimilarContent{
			Content:    s,
			Similarity: similarityScore,
			Context:    "Semantic similarity based on content embeddings",
		})
	}

	return response, nil
}

// FindRelatedContent finds content connected through relationships
func (d *Domain) FindRelatedContent(ctx context.Context, req *domainTypes.FindRelatedRequest) (*domainTypes.FindRelatedResponse, error) {
	startTime := time.Now()

	maxDepth := req.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 3
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}

	// Find related content through relationships
	related, err := d.relationshipStore.FindRelated(ctx, req.ProjectID, req.ContentID, maxDepth)
	if err != nil {
		return nil, fmt.Errorf("failed to find related content: %w", err)
	}

	// Apply limit
	if len(related) > limit {
		related = related[:limit]
	}

	return &domainTypes.FindRelatedResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   true,
			Timestamp: time.Now(),
			Duration:  time.Since(startTime),
		},
		Related: related,
	}, nil
}

// Relationship Operations

// CreateRelationship creates a relationship between content items
func (d *Domain) CreateRelationship(ctx context.Context, req *domainTypes.CreateRelationshipRequest) (*domainTypes.CreateRelationshipResponse, error) {
	// TODO: Implement relationship creation
	return &domainTypes.CreateRelationshipResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   false,
			Message:   "Relationship creation not yet implemented",
			Timestamp: time.Now(),
		},
	}, errors.New("relationship creation not yet implemented")
}

// GetRelationships retrieves relationships for content
func (d *Domain) GetRelationships(ctx context.Context, req *domainTypes.GetRelationshipsRequest) (*domainTypes.GetRelationshipsResponse, error) {
	// TODO: Implement relationship retrieval
	return &domainTypes.GetRelationshipsResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   false,
			Message:   "Relationship retrieval not yet implemented",
			Timestamp: time.Now(),
		},
	}, errors.New("relationship retrieval not yet implemented")
}

// DeleteRelationship removes a relationship
func (d *Domain) DeleteRelationship(ctx context.Context, req *domainTypes.DeleteRelationshipRequest) error {
	// TODO: Implement relationship deletion
	return errors.New("relationship deletion not yet implemented")
}

// Intelligence and Analysis Operations

// DetectPatterns identifies patterns in content and behavior
func (d *Domain) DetectPatterns(ctx context.Context, req *domainTypes.DetectPatternsRequest) (*domainTypes.DetectPatternsResponse, error) {
	// TODO: Implement pattern detection
	return &domainTypes.DetectPatternsResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   false,
			Message:   "Pattern detection not yet implemented",
			Timestamp: time.Now(),
		},
	}, errors.New("pattern detection not yet implemented")
}

// GenerateInsights generates insights from content analysis
func (d *Domain) GenerateInsights(ctx context.Context, req *domainTypes.GenerateInsightsRequest) (*domainTypes.GenerateInsightsResponse, error) {
	// TODO: Implement insight generation
	return &domainTypes.GenerateInsightsResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   false,
			Message:   "Insight generation not yet implemented",
			Timestamp: time.Now(),
		},
	}, errors.New("insight generation not yet implemented")
}

// AnalyzeQuality analyzes content quality
func (d *Domain) AnalyzeQuality(ctx context.Context, req *domainTypes.AnalyzeQualityRequest) (*domainTypes.AnalyzeQualityResponse, error) {
	// TODO: Implement quality analysis
	return &domainTypes.AnalyzeQualityResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   false,
			Message:   "Quality analysis not yet implemented",
			Timestamp: time.Now(),
		},
	}, errors.New("quality analysis not yet implemented")
}

// DetectConflicts identifies conflicting information
func (d *Domain) DetectConflicts(ctx context.Context, req *domainTypes.DetectConflictsRequest) (*domainTypes.DetectConflictsResponse, error) {
	// TODO: Implement conflict detection
	return &domainTypes.DetectConflictsResponse{
		BaseResponse: domainTypes.BaseResponse{
			Success:   false,
			Message:   "Conflict detection not yet implemented",
			Timestamp: time.Now(),
		},
	}, errors.New("conflict detection not yet implemented")
}

// Helper methods

// autoDetectRelationships automatically detects and creates relationships for new content
func (d *Domain) autoDetectRelationships(ctx context.Context, content *coreTypes.Content) {
	// TODO: Implement automatic relationship detection
	// This would use AI/ML to find semantic relationships between content
}

// generateContentID generates a unique content ID
func (d *Domain) generateContentID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes) // crypto/rand.Read never returns an error
	return hex.EncodeToString(bytes)
}
