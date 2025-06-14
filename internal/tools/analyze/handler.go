// Package analyze provides the memory_analyze tool implementation.
// Handles all analysis and intelligence operations for pattern detection and insights.
package analyze

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/ai"
	"lerian-mcp-memory/internal/session"
	"lerian-mcp-memory/internal/storage"
	"lerian-mcp-memory/internal/tools"
	"lerian-mcp-memory/internal/types"
	"lerian-mcp-memory/internal/validation"
)

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Handler implements the memory_analyze tool
type Handler struct {
	sessionManager  *session.Manager
	validator       *validation.ParameterValidator
	patternDetector *ai.PatternDetector
	aiService       ai.AIService
	searchStore     storage.SearchStore
	analysisStore   storage.AnalysisStore
}

// NewHandler creates a new analyze handler
func NewHandler(
	sessionManager *session.Manager,
	validator *validation.ParameterValidator,
	patternDetector *ai.PatternDetector,
	aiService ai.AIService,
	searchStore storage.SearchStore,
	analysisStore storage.AnalysisStore,
) *Handler {
	return &Handler{
		sessionManager:  sessionManager,
		validator:       validator,
		patternDetector: patternDetector,
		aiService:       aiService,
		searchStore:     searchStore,
		analysisStore:   analysisStore,
	}
}

// DetectPatternsRequest represents a pattern detection request
type DetectPatternsRequest struct {
	types.StandardParams
	Scope         string     `json:"scope"`                    // "project", "session", "timeframe"
	TimeframeType string     `json:"timeframe_type,omitempty"` // "week", "month", "quarter"
	PatternTypes  []string   `json:"pattern_types,omitempty"`  // "code", "decisions", "issues", "solutions"
	DateRange     *DateRange `json:"date_range,omitempty"`
	MinConfidence float64    `json:"min_confidence,omitempty"`
	Limit         int        `json:"limit,omitempty"`
}

// DateRange represents a date filter range
type DateRange struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// SuggestRelatedRequest represents a request for related content suggestions
type SuggestRelatedRequest struct {
	types.StandardParams
	ContentID     string   `json:"content_id,omitempty"`     // Suggest related to specific content
	Context       string   `json:"context,omitempty"`        // Suggest related to context/query
	RelationTypes []string `json:"relation_types,omitempty"` // "similar", "dependencies", "references"
	Limit         int      `json:"limit,omitempty"`
	IncludeScore  bool     `json:"include_score,omitempty"`
}

// AnalyzeQualityRequest represents a content quality analysis request
type AnalyzeQualityRequest struct {
	types.StandardParams
	ContentID string `json:"content_id,omitempty"` // Analyze specific content
	Scope     string `json:"scope,omitempty"`      // "project", "session", "content"
}

// DetectConflictsRequest represents a conflict detection request
type DetectConflictsRequest struct {
	types.StandardParams
	Scope        string   `json:"scope"`                 // "project", "session"
	ConflictType string   `json:"conflict_type"`         // "decisions", "solutions", "requirements"
	ContentIDs   []string `json:"content_ids,omitempty"` // Check specific content for conflicts
}

// Pattern represents a detected pattern
type Pattern struct {
	PatternID   string                 `json:"pattern_id"`
	Type        string                 `json:"type"` // "code", "decision", "issue", "solution"
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"`
	Frequency   int                    `json:"frequency"`
	Examples    []string               `json:"examples"` // Content IDs demonstrating the pattern
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
	Trend       string                 `json:"trend"` // "increasing", "decreasing", "stable"
}

// RelatedSuggestion represents a suggestion for related content
type RelatedSuggestion struct {
	ContentID    string                 `json:"content_id"`
	Type         string                 `json:"type"`
	Title        string                 `json:"title,omitempty"`
	Summary      string                 `json:"summary"`
	Relevance    float64                `json:"relevance"`
	Relationship string                 `json:"relationship"` // Why it's related
	Tags         []string               `json:"tags,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// QualityAnalysis represents content quality analysis results
type QualityAnalysis struct {
	OverallScore    float64            `json:"overall_score"` // 0.0-1.0
	Completeness    float64            `json:"completeness"`  // 0.0-1.0
	Clarity         float64            `json:"clarity"`       // 0.0-1.0
	Relevance       float64            `json:"relevance"`     // 0.0-1.0
	Recency         float64            `json:"recency"`       // 0.0-1.0
	Issues          []QualityIssue     `json:"issues,omitempty"`
	Recommendations []string           `json:"recommendations,omitempty"`
	Metrics         map[string]float64 `json:"metrics,omitempty"`
	AnalyzedAt      time.Time          `json:"analyzed_at"`
}

// QualityIssue represents a quality issue
type QualityIssue struct {
	Type        string `json:"type"`     // "incomplete", "unclear", "outdated", "inconsistent"
	Severity    string `json:"severity"` // "low", "medium", "high", "critical"
	Description string `json:"description"`
	Suggestion  string `json:"suggestion,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

// Conflict represents a detected conflict
type Conflict struct {
	ConflictID  string    `json:"conflict_id"`
	Type        string    `json:"type"`     // "decision", "solution", "requirement"
	Severity    string    `json:"severity"` // "low", "medium", "high", "critical"
	Description string    `json:"description"`
	ContentIDs  []string  `json:"content_ids"`           // Conflicting content
	Suggestions []string  `json:"suggestions,omitempty"` // How to resolve
	DetectedAt  time.Time `json:"detected_at"`
	Confidence  float64   `json:"confidence"`
}

// Insight represents a generated insight
type Insight struct {
	InsightID   string                 `json:"insight_id"`
	Type        string                 `json:"type"` // "trend", "opportunity", "risk", "recommendation"
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"`
	Impact      string                 `json:"impact"`            // "low", "medium", "high"
	Evidence    []string               `json:"evidence"`          // Supporting content IDs
	Actions     []string               `json:"actions,omitempty"` // Recommended actions
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// HandleOperation handles all analyze operations
func (h *Handler) HandleOperation(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error) {
	switch operation {
	case string(tools.OpDetectPatterns):
		return h.handleDetectPatterns(ctx, params)
	case string(tools.OpSuggestRelated):
		return h.handleSuggestRelated(ctx, params)
	case string(tools.OpAnalyzeQuality):
		return h.handleAnalyzeQuality(ctx, params)
	case string(tools.OpDetectConflicts):
		return h.handleDetectConflicts(ctx, params)
	case string(tools.OpGenerateInsights):
		return h.handleGenerateInsights(ctx, params)
	case string(tools.OpPredictTrends):
		return h.handlePredictTrends(ctx, params)
	default:
		return nil, fmt.Errorf("unknown analyze operation: %s", operation)
	}
}

// handleDetectPatterns identifies patterns in content and behavior
func (h *Handler) handleDetectPatterns(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	req, err := h.parseDetectPatternsRequest(params)
	if err != nil {
		return nil, err
	}

	if err := h.validateDetectPatternsRequest(req); err != nil {
		return nil, err
	}

	if err := h.updateSessionIfNeeded(req.ProjectID, req.SessionID); err != nil {
		return nil, err
	}

	memories, err := h.retrieveMemoriesForPatternAnalysis(ctx, req)
	if err != nil {
		return nil, err
	}

	memoryData := h.convertToMemoryData(memories)
	timeSpan := h.calculateTimeSpan(req.TimeframeType)

	result, err := h.performPatternDetection(ctx, req.ProjectID, memoryData, timeSpan)
	if err != nil {
		return nil, err
	}

	patterns := h.convertDetectedPatterns(result.Patterns)
	patterns = h.applyPatternsFiltering(patterns, req)

	return h.buildPatternsResponse(patterns, req, result, timeSpan), nil
}

// parseDetectPatternsRequest parses the detect patterns request parameters
func (h *Handler) parseDetectPatternsRequest(params map[string]interface{}) (*DetectPatternsRequest, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req DetectPatternsRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse detect patterns request: %w", err)
	}

	return &req, nil
}

// validateDetectPatternsRequest validates the detect patterns request parameters
func (h *Handler) validateDetectPatternsRequest(req *DetectPatternsRequest) error {
	if err := h.validator.ValidateOperation(string(tools.OpDetectPatterns), &req.StandardParams); err != nil {
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	h.setDetectPatternsDefaults(req)
	return nil
}

// setDetectPatternsDefaults sets default values for detect patterns request
func (h *Handler) setDetectPatternsDefaults(req *DetectPatternsRequest) {
	if req.Scope == "" {
		req.Scope = "project"
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.MinConfidence <= 0 {
		req.MinConfidence = 0.6
	}
}

// retrieveMemoriesForPatternAnalysis retrieves memories based on the request
func (h *Handler) retrieveMemoriesForPatternAnalysis(ctx context.Context, req *DetectPatternsRequest) ([]*types.Content, error) {
	filters := h.buildPatternAnalysisFilters(req)

	if !req.SessionID.IsEmpty() {
		return h.searchStore.GetBySession(ctx, req.ProjectID, req.SessionID, filters)
	}
	return h.searchStore.GetByProject(ctx, req.ProjectID, filters)
}

// buildPatternAnalysisFilters builds filters for pattern analysis
func (h *Handler) buildPatternAnalysisFilters(req *DetectPatternsRequest) *types.Filters {
	filters := &types.Filters{}

	if len(req.PatternTypes) > 0 {
		filters.Types = req.PatternTypes
	}

	if req.DateRange != nil {
		filters.CreatedAfter = req.DateRange.Start
		filters.CreatedBefore = req.DateRange.End
	}

	return filters
}

// convertToMemoryData converts Content to MemoryData format
func (h *Handler) convertToMemoryData(memories []*types.Content) []ai.MemoryData {
	memoryData := make([]ai.MemoryData, len(memories))
	for i, memory := range memories {
		memoryData[i] = ai.MemoryData{
			ID:        memory.ID,
			Content:   memory.Content,
			Type:      memory.Type,
			CreatedAt: memory.CreatedAt,
			Tags:      memory.Tags,
			Metadata:  memory.Metadata,
		}
	}
	return memoryData
}

// calculateTimeSpan calculates the time span based on timeframe type
func (h *Handler) calculateTimeSpan(timeframeType string) time.Duration {
	switch timeframeType {
	case "week":
		return time.Hour * 24 * 7
	case "quarter":
		return time.Hour * 24 * 90
	default:
		return time.Hour * 24 * 30 // Default to last 30 days
	}
}

// performPatternDetection performs AI-powered pattern detection
func (h *Handler) performPatternDetection(ctx context.Context, projectID types.ProjectID, memoryData []ai.MemoryData, timeSpan time.Duration) (*ai.PatternDetectionResult, error) {
	result, err := h.patternDetector.AnalyzeProjectPatterns(ctx, string(projectID), memoryData, timeSpan)
	if err != nil {
		return nil, fmt.Errorf("pattern detection failed: %w", err)
	}
	return result, nil
}

// convertDetectedPatterns converts detected patterns to response format
func (h *Handler) convertDetectedPatterns(detectedPatterns []*ai.DetectedPattern) []Pattern {
	patterns := make([]Pattern, len(detectedPatterns))
	for i, detectedPattern := range detectedPatterns {
		patterns[i] = Pattern{
			PatternID:   detectedPattern.ID,
			Type:        detectedPattern.Type,
			Description: detectedPattern.Description,
			Confidence:  detectedPattern.Confidence,
			Frequency:   detectedPattern.Frequency,
			Examples:    detectedPattern.Examples,
			FirstSeen:   detectedPattern.CreatedAt,
			LastSeen:    detectedPattern.UpdatedAt,
			Trend:       "stable", // Default, could be enhanced
			Metadata: map[string]interface{}{
				"tags": detectedPattern.Tags,
			},
		}
	}
	return patterns
}

// applyPatternsFiltering applies confidence and limit filtering to patterns
func (h *Handler) applyPatternsFiltering(patterns []Pattern, req *DetectPatternsRequest) []Pattern {
	patterns = h.applyConfidenceFiltering(patterns, req.MinConfidence)
	patterns = h.applyPatternsLimit(patterns, req.Limit)
	return patterns
}

// applyConfidenceFiltering filters patterns by minimum confidence
func (h *Handler) applyConfidenceFiltering(patterns []Pattern, minConfidence float64) []Pattern {
	if minConfidence <= 0 {
		return patterns
	}

	filteredPatterns := make([]Pattern, 0)
	for i := range patterns {
		pattern := &patterns[i]
		if pattern.Confidence >= minConfidence {
			filteredPatterns = append(filteredPatterns, *pattern)
		}
	}
	return filteredPatterns
}

// applyPatternsLimit applies limit to patterns
func (h *Handler) applyPatternsLimit(patterns []Pattern, limit int) []Pattern {
	if limit > 0 && len(patterns) > limit {
		return patterns[:limit]
	}
	return patterns
}

// buildPatternsResponse builds the patterns response
func (h *Handler) buildPatternsResponse(patterns []Pattern, req *DetectPatternsRequest, result *ai.PatternDetectionResult, timeSpan time.Duration) map[string]interface{} {
	return map[string]interface{}{
		"patterns":     patterns,
		"total":        len(patterns),
		"scope":        req.Scope,
		"analyzed_at":  time.Now(),
		"analysis_id":  result.AnalysisID,
		"memory_count": result.MemoryCount,
		"time_span":    timeSpan.String(),
	}
}

// handleSuggestRelated suggests related content
func (h *Handler) handleSuggestRelated(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	req, err := h.parseSuggestRelatedRequest(params)
	if err != nil {
		return nil, err
	}

	if err := h.validateSuggestRelatedRequest(req); err != nil {
		return nil, err
	}

	if err := h.updateSessionIfNeeded(req.ProjectID, req.SessionID); err != nil {
		return nil, err
	}

	suggestions, err := h.getSuggestions(ctx, req)
	if err != nil {
		return nil, err
	}

	suggestions = h.applyRelationTypeFiltering(suggestions, req.RelationTypes)
	suggestions = h.applyLimit(suggestions, req.Limit)

	return map[string]interface{}{
		"suggestions": suggestions,
		"total":       len(suggestions),
		"context":     req.Context,
		"content_id":  req.ContentID,
	}, nil
}

// parseSuggestRelatedRequest parses the request parameters
func (h *Handler) parseSuggestRelatedRequest(params map[string]interface{}) (*SuggestRelatedRequest, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req SuggestRelatedRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse suggest related request: %w", err)
	}

	return &req, nil
}

// validateSuggestRelatedRequest validates the request parameters
func (h *Handler) validateSuggestRelatedRequest(req *SuggestRelatedRequest) error {
	if err := h.validator.ValidateOperation(string(tools.OpSuggestRelated), &req.StandardParams); err != nil {
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.ContentID == "" && req.Context == "" {
		return fmt.Errorf("either content_id or context is required")
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}

	return nil
}

// updateSessionIfNeeded updates session access if session ID is provided
func (h *Handler) updateSessionIfNeeded(projectID types.ProjectID, sessionID types.SessionID) error {
	if sessionID.IsEmpty() {
		return nil
	}

	return h.sessionManager.UpdateSessionAccess(projectID, sessionID)
}

// getSuggestions retrieves suggestions based on content ID or context
func (h *Handler) getSuggestions(ctx context.Context, req *SuggestRelatedRequest) ([]RelatedSuggestion, error) {
	if req.ContentID != "" {
		return h.getSuggestionsByContentID(ctx, req)
	}
	if req.Context != "" {
		return h.getSuggestionsByContext(ctx, req)
	}
	return nil, fmt.Errorf("no valid suggestion source provided")
}

// getSuggestionsByContentID gets suggestions based on content ID
func (h *Handler) getSuggestionsByContentID(ctx context.Context, req *SuggestRelatedRequest) ([]RelatedSuggestion, error) {
	targetContent, err := h.findTargetContent(ctx, req.ProjectID, req.ContentID)
	if err != nil {
		return nil, err
	}

	similarContent, err := h.searchStore.FindSimilar(ctx, targetContent.Content, req.ProjectID, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar content: %w", err)
	}

	return h.convertToSimilarSuggestions(similarContent, req.ContentID), nil
}

// getSuggestionsByContext gets suggestions based on context
func (h *Handler) getSuggestionsByContext(ctx context.Context, req *SuggestRelatedRequest) ([]RelatedSuggestion, error) {
	searchQuery := &types.SearchQuery{
		Query:     req.Context,
		ProjectID: req.ProjectID,
		SessionID: req.SessionID,
		Limit:     req.Limit * 2, // Get more to filter down
	}

	searchResults, err := h.searchStore.Search(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search for related content: %w", err)
	}

	return h.convertToContextSuggestions(searchResults.Results), nil
}

// findTargetContent finds the target content by ID
func (h *Handler) findTargetContent(ctx context.Context, projectID types.ProjectID, contentID string) (*types.Content, error) {
	allContent, err := h.searchStore.GetByProject(ctx, projectID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	for _, content := range allContent {
		if content.ID == contentID {
			return content, nil
		}
	}

	return nil, fmt.Errorf("target content not found: %s", contentID)
}

// convertToSimilarSuggestions converts content to similar suggestions
func (h *Handler) convertToSimilarSuggestions(content []*types.Content, excludeID string) []RelatedSuggestion {
	suggestions := make([]RelatedSuggestion, 0, len(content))
	for _, c := range content {
		if c.ID != excludeID {
			suggestion := RelatedSuggestion{
				ContentID:    c.ID,
				Type:         c.Type,
				Title:        c.Summary,
				Summary:      c.Content[:minInt(200, len(c.Content))],
				Relevance:    0.8, // TODO: Calculate actual relevance score
				Relationship: "Semantically similar content",
				Tags:         c.Tags,
				CreatedAt:    c.CreatedAt,
				Metadata:     c.Metadata,
			}
			suggestions = append(suggestions, suggestion)
		}
	}
	return suggestions
}

// convertToContextSuggestions converts search results to context suggestions
func (h *Handler) convertToContextSuggestions(results []*types.SearchResult) []RelatedSuggestion {
	suggestions := make([]RelatedSuggestion, 0, len(results))
	for _, result := range results {
		suggestion := RelatedSuggestion{
			ContentID:    result.Content.ID,
			Type:         result.Content.Type,
			Title:        result.Content.Summary,
			Summary:      result.Content.Content[:minInt(200, len(result.Content.Content))],
			Relevance:    result.Relevance,
			Relationship: "Matches search context",
			Tags:         result.Content.Tags,
			CreatedAt:    result.Content.CreatedAt,
			Metadata:     result.Content.Metadata,
		}
		suggestions = append(suggestions, suggestion)
	}
	return suggestions
}

// applyRelationTypeFiltering filters suggestions by relation types
func (h *Handler) applyRelationTypeFiltering(suggestions []RelatedSuggestion, relationTypes []string) []RelatedSuggestion {
	if len(relationTypes) == 0 {
		return suggestions
	}

	var filtered []RelatedSuggestion
	for i := range suggestions {
		if h.matchesRelationType(&suggestions[i], relationTypes) {
			filtered = append(filtered, suggestions[i])
		}
	}
	return filtered
}

// matchesRelationType checks if suggestion matches any of the relation types
func (h *Handler) matchesRelationType(suggestion *RelatedSuggestion, relationTypes []string) bool {
	for _, relType := range relationTypes {
		if h.isRelationTypeMatch(suggestion, relType) {
			return true
		}
	}
	return false
}

// isRelationTypeMatch checks if suggestion matches a specific relation type
func (h *Handler) isRelationTypeMatch(sug *RelatedSuggestion, relType string) bool {
	switch relType {
	case "similar":
		return sug.Relationship == "Semantically similar content"
	case "references":
		return sug.Relationship == "Matches search context"
	default:
		return false
	}
}

// applyLimit applies the limit to suggestions
func (h *Handler) applyLimit(suggestions []RelatedSuggestion, limit int) []RelatedSuggestion {
	if limit > 0 && len(suggestions) > limit {
		return suggestions[:limit]
	}
	return suggestions
}

// handleAnalyzeQuality analyzes content quality
func (h *Handler) handleAnalyzeQuality(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	req, err := h.parseAnalyzeQualityRequest(params)
	if err != nil {
		return nil, err
	}

	if err := h.validateAnalyzeQualityRequest(req); err != nil {
		return nil, err
	}

	if err := h.updateSessionIfNeeded(req.ProjectID, req.SessionID); err != nil {
		return nil, err
	}

	return h.performQualityAnalysis(ctx, req)
}

// parseAnalyzeQualityRequest parses the request parameters
func (h *Handler) parseAnalyzeQualityRequest(params map[string]interface{}) (*AnalyzeQualityRequest, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req AnalyzeQualityRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse analyze quality request: %w", err)
	}

	return &req, nil
}

// validateAnalyzeQualityRequest validates the request parameters
func (h *Handler) validateAnalyzeQualityRequest(req *AnalyzeQualityRequest) error {
	if err := h.validator.ValidateOperation(string(tools.OpAnalyzeQuality), &req.StandardParams); err != nil {
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.Scope == "" {
		req.Scope = "project"
	}

	return nil
}

// performQualityAnalysis performs the actual quality analysis
func (h *Handler) performQualityAnalysis(ctx context.Context, req *AnalyzeQualityRequest) (*QualityAnalysis, error) {
	if req.ContentID != "" {
		return h.analyzeSingleContent(ctx, req)
	}
	return h.analyzeMultipleContent(ctx, req)
}

// analyzeSingleContent analyzes a single content item
func (h *Handler) analyzeSingleContent(ctx context.Context, req *AnalyzeQualityRequest) (*QualityAnalysis, error) {
	content, err := h.findTargetContent(ctx, req.ProjectID, req.ContentID)
	if err != nil {
		return nil, err
	}

	qualityMetrics, err := h.aiService.AssessQuality(ctx, content.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to assess content quality: %w", err)
	}

	return &QualityAnalysis{
		OverallScore:    qualityMetrics.OverallScore,
		Completeness:    qualityMetrics.Completeness,
		Clarity:         qualityMetrics.Clarity,
		Relevance:       qualityMetrics.Relevance,
		Recency:         h.calculateRecencyScore(content.CreatedAt),
		Issues:          h.extractQualityIssues(qualityMetrics, content.ID),
		Recommendations: h.extractRecommendations(qualityMetrics),
		AnalyzedAt:      time.Now(),
	}, nil
}

// analyzeMultipleContent analyzes multiple content items
func (h *Handler) analyzeMultipleContent(ctx context.Context, req *AnalyzeQualityRequest) (*QualityAnalysis, error) {
	contents, err := h.getContentsForAnalysis(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(contents) == 0 {
		return h.createEmptyAnalysis(), nil
	}

	return h.aggregateQualityMetrics(ctx, contents)
}

// getContentsForAnalysis retrieves contents based on the request scope
func (h *Handler) getContentsForAnalysis(ctx context.Context, req *AnalyzeQualityRequest) ([]*types.Content, error) {
	if !req.SessionID.IsEmpty() {
		return h.searchStore.GetBySession(ctx, req.ProjectID, req.SessionID, nil)
	}
	return h.searchStore.GetByProject(ctx, req.ProjectID, nil)
}

// aggregateQualityMetrics aggregates quality metrics from multiple content items
func (h *Handler) aggregateQualityMetrics(ctx context.Context, contents []*types.Content) (*QualityAnalysis, error) {
	aggregator := newQualityAggregator()

	for _, item := range contents {
		if err := h.processContentForAggregation(ctx, item, aggregator); err != nil {
			continue // Skip failed assessments but continue with others
		}
	}

	return aggregator.buildAnalysis(len(contents)), nil
}

// processContentForAggregation processes a single content item for aggregation
func (h *Handler) processContentForAggregation(ctx context.Context, content *types.Content, aggregator *qualityAggregator) error {
	qualityMetrics, err := h.aiService.AssessQuality(ctx, content.Content)
	if err != nil {
		return err
	}

	aggregator.addMetrics(qualityMetrics, content, h)
	return nil
}

// createEmptyAnalysis creates an empty quality analysis
func (h *Handler) createEmptyAnalysis() *QualityAnalysis {
	return &QualityAnalysis{
		OverallScore: 0.0,
		AnalyzedAt:   time.Now(),
	}
}

// qualityAggregator helps aggregate quality metrics
type qualityAggregator struct {
	totalScore        float64
	totalCompleteness float64
	totalClarity      float64
	totalRelevance    float64
	totalRecency      float64
	issues            []QualityIssue
	recommendations   map[string]bool
}

// newQualityAggregator creates a new quality aggregator
func newQualityAggregator() *qualityAggregator {
	return &qualityAggregator{
		issues:          make([]QualityIssue, 0),
		recommendations: make(map[string]bool),
	}
}

// addMetrics adds metrics from a content item to the aggregator
func (qa *qualityAggregator) addMetrics(metrics interface{}, content *types.Content, h *Handler) {
	// Type assertion to extract metrics (assuming metrics has these fields)
	type QualityMetrics struct {
		OverallScore float64
		Completeness float64
		Clarity      float64
		Relevance    float64
	}

	qm, ok := metrics.(QualityMetrics)
	if !ok {
		return
	}

	qa.totalScore += qm.OverallScore
	qa.totalCompleteness += qm.Completeness
	qa.totalClarity += qm.Clarity
	qa.totalRelevance += qm.Relevance
	qa.totalRecency += h.calculateRecencyScore(content.CreatedAt)

	// Extract issues from this content
	if qualityMetrics, ok := metrics.(*ai.UnifiedQualityMetrics); ok {
		contentIssues := h.extractQualityIssues(qualityMetrics, content.ID)
		qa.issues = append(qa.issues, contentIssues...)

		// Collect recommendations
		for _, rec := range h.extractRecommendations(qualityMetrics) {
			qa.recommendations[rec] = true
		}
	}
}

// buildAnalysis builds the final quality analysis
func (qa *qualityAggregator) buildAnalysis(count int) *QualityAnalysis {
	fCount := float64(count)
	recommendations := make([]string, 0, len(qa.recommendations))
	for rec := range qa.recommendations {
		recommendations = append(recommendations, rec)
	}

	return &QualityAnalysis{
		OverallScore:    qa.totalScore / fCount,
		Completeness:    qa.totalCompleteness / fCount,
		Clarity:         qa.totalClarity / fCount,
		Relevance:       qa.totalRelevance / fCount,
		Recency:         qa.totalRecency / fCount,
		Issues:          qa.issues,
		Recommendations: recommendations,
		AnalyzedAt:      time.Now(),
	}
}

// handleDetectConflicts identifies conflicting information
func (h *Handler) handleDetectConflicts(ctx context.Context, params map[string]interface{}) (interface{}, error) { //nolint:unparam // context part of MCP handler interface
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req DetectConflictsRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse detect conflicts request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpDetectConflicts), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.ConflictType == "" {
		req.ConflictType = "decisions"
	}

	// Update session access if session provided
	if !req.SessionID.IsEmpty() {
		if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
			return nil, fmt.Errorf("failed to update session access: %w", err)
		}
	}

	// TODO: Implement actual conflict detection
	mockConflicts := []Conflict{
		{
			ConflictID:  "conflict_1",
			Type:        "decision",
			Severity:    "medium",
			Description: "Conflicting architectural decisions about database choice",
			ContentIDs:  []string{"decision_1", "decision_4"},
			Suggestions: []string{"Review both decisions and establish clear criteria", "Document final decision with rationale"},
			DetectedAt:  time.Now(),
			Confidence:  0.82,
		},
	}

	return map[string]interface{}{
		"conflicts":     mockConflicts,
		"total":         len(mockConflicts),
		"conflict_type": req.ConflictType,
		"scope":         req.Scope,
	}, nil
}

// handleGenerateInsights generates insights from data patterns
func (h *Handler) handleGenerateInsights(ctx context.Context, params map[string]interface{}) (interface{}, error) { //nolint:unparam // context part of MCP handler interface
	// TODO: Implement insight generation
	mockInsights := []Insight{
		{
			InsightID:   "insight_1",
			Type:        "trend",
			Title:       "Increasing Focus on Error Handling",
			Description: "There's been a 40% increase in discussions about error handling patterns over the past month",
			Confidence:  0.85,
			Impact:      "medium",
			Evidence:    []string{"content_1", "content_5", "pattern_1"},
			Actions:     []string{"Consider creating error handling guidelines", "Document common patterns"},
			GeneratedAt: time.Now(),
		},
	}

	return map[string]interface{}{
		"insights": mockInsights,
		"total":    len(mockInsights),
	}, nil
}

// handlePredictTrends predicts future trends
func (h *Handler) handlePredictTrends(ctx context.Context, params map[string]interface{}) (interface{}, error) { //nolint:unparam // context part of MCP handler interface
	// TODO: Implement trend prediction
	return map[string]interface{}{
		"trends": []map[string]interface{}{
			{
				"trend_id":    "trend_1",
				"type":        "technology",
				"description": "Predicted increase in microservices adoption",
				"confidence":  0.75,
				"timeframe":   "3 months",
				"probability": 0.82,
			},
		},
		"total": 1,
	}, nil
}

// calculateRecencyScore calculates a recency score based on content age
func (h *Handler) calculateRecencyScore(createdAt time.Time) float64 {
	age := time.Since(createdAt)
	days := age.Hours() / 24

	// Newer content gets higher scores
	switch {
	case days <= 7:
		return 1.0 // Very recent
	case days <= 30:
		return 0.8 // Recent
	case days <= 90:
		return 0.6 // Moderately recent
	case days <= 365:
		return 0.4 // Older
	default:
		return 0.2 // Very old
	}
}

// extractQualityIssues extracts quality issues from AI metrics
func (h *Handler) extractQualityIssues(metrics *ai.UnifiedQualityMetrics, contentID string) []QualityIssue {
	issues := make([]QualityIssue, 0)

	// Check for common quality issues based on metrics
	if metrics.Completeness < 0.5 {
		issues = append(issues, QualityIssue{
			Type:        "incomplete",
			Severity:    "medium",
			Description: "Content appears to be incomplete or missing key information",
			Suggestion:  "Add more details and complete missing sections",
			ContentID:   contentID,
		})
	}

	if metrics.Clarity < 0.5 {
		issues = append(issues, QualityIssue{
			Type:        "unclear",
			Severity:    "medium",
			Description: "Content could be clearer and easier to understand",
			Suggestion:  "Simplify language and improve structure",
			ContentID:   contentID,
		})
	}

	if metrics.Relevance < 0.5 {
		issues = append(issues, QualityIssue{
			Type:        "irrelevant",
			Severity:    "low",
			Description: "Content may not be highly relevant to the current context",
			Suggestion:  "Consider updating or removing if no longer relevant",
			ContentID:   contentID,
		})
	}

	// Add issues from missing elements
	for _, missing := range metrics.MissingElements {
		issues = append(issues, QualityIssue{
			Type:        "incomplete",
			Severity:    "low",
			Description: fmt.Sprintf("Missing element: %s", missing),
			Suggestion:  fmt.Sprintf("Add %s to improve content quality", missing),
			ContentID:   contentID,
		})
	}

	return issues
}

// extractRecommendations extracts recommendations from AI metrics
func (h *Handler) extractRecommendations(metrics *ai.UnifiedQualityMetrics) []string {
	recommendations := make([]string, 0)

	if metrics.Completeness < 0.7 {
		recommendations = append(recommendations, "Add more comprehensive details and context")
	}

	if metrics.Clarity < 0.7 {
		recommendations = append(recommendations, "Improve clarity with better structure and simpler language")
	}

	if metrics.Actionability < 0.7 {
		recommendations = append(recommendations, "Include more actionable steps and concrete examples")
	}

	if metrics.Uniqueness < 0.5 {
		recommendations = append(recommendations, "Add unique insights or perspectives to differentiate content")
	}

	// Generic quality improvement suggestions
	if metrics.OverallScore < 0.6 {
		recommendations = append(recommendations,
			"Consider restructuring content for better readability",
			"Add relevant examples and use cases",
			"Include links to related resources")
	}

	return recommendations
}

// GetToolDefinition returns the MCP tool definition for memory_analyze
func (h *Handler) GetToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        string(tools.ToolMemoryAnalyze),
		"description": "Analyze memory content for patterns, relationships, quality, conflicts, and insights. Provides intelligence and analytical capabilities for stored content.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"enum":        tools.GetOperationsForTool(tools.ToolMemoryAnalyze),
					"description": "The analysis operation to perform",
				},
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "Project identifier for data isolation (required)",
				},
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session identifier for expanded access (optional)",
				},
				"scope": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"project", "session", "timeframe"},
					"description": "Analysis scope",
					"default":     "project",
				},
				"content_id": map[string]interface{}{
					"type":        "string",
					"description": "Content ID for specific content analysis",
				},
				"context": map[string]interface{}{
					"type":        "string",
					"description": "Context for analysis or suggestions",
				},
				"pattern_types": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Types of patterns to detect",
				},
				"conflict_type": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"decisions", "solutions", "requirements"},
					"description": "Type of conflicts to detect",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return",
					"default":     10,
				},
			},
			"required": []string{"operation", "project_id"},
		},
	}
}
