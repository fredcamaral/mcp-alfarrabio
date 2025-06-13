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

// min returns the minimum of two integers
func min(a, b int) int {
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
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req DetectPatternsRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse detect patterns request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpDetectPatterns), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.Scope == "" {
		req.Scope = "project"
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.MinConfidence <= 0 {
		req.MinConfidence = 0.6
	}

	// Update session access if session provided
	if !req.SessionID.IsEmpty() {
		if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
			return nil, fmt.Errorf("failed to update session access: %w", err)
		}
	}

	// Get memories from storage for pattern analysis
	filters := &types.Filters{}
	if len(req.PatternTypes) > 0 {
		filters.Types = req.PatternTypes
	}

	// Apply date range filtering if provided
	if req.DateRange != nil {
		filters.CreatedAfter = req.DateRange.Start
		filters.CreatedBefore = req.DateRange.End
	}

	var memories []*types.Content
	var retrieveErr error

	if !req.SessionID.IsEmpty() {
		// Get session-specific memories for expanded access
		memories, retrieveErr = h.searchStore.GetBySession(ctx, req.ProjectID, req.SessionID, filters)
	} else {
		// Get project-wide memories for read-only access
		memories, retrieveErr = h.searchStore.GetByProject(ctx, req.ProjectID, filters)
	}

	if retrieveErr != nil {
		return nil, fmt.Errorf("failed to retrieve memories for pattern analysis: %w", retrieveErr)
	}

	// Convert to MemoryData format for pattern detector
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

	// Perform AI-powered pattern detection
	timeSpan := time.Hour * 24 * 30 // Default to last 30 days
	if req.TimeframeType == "week" {
		timeSpan = time.Hour * 24 * 7
	} else if req.TimeframeType == "quarter" {
		timeSpan = time.Hour * 24 * 90
	}

	result, err := h.patternDetector.AnalyzeProjectPatterns(ctx, string(req.ProjectID), memoryData, timeSpan)
	if err != nil {
		return nil, fmt.Errorf("pattern detection failed: %w", err)
	}

	// Convert detected patterns to response format
	patterns := make([]Pattern, len(result.Patterns))
	for i, detectedPattern := range result.Patterns {
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

	// Apply confidence filtering
	if req.MinConfidence > 0 {
		filteredPatterns := make([]Pattern, 0)
		for _, pattern := range patterns {
			if pattern.Confidence >= req.MinConfidence {
				filteredPatterns = append(filteredPatterns, pattern)
			}
		}
		patterns = filteredPatterns
	}

	// Apply limit
	if req.Limit > 0 && len(patterns) > req.Limit {
		patterns = patterns[:req.Limit]
	}

	return map[string]interface{}{
		"patterns":     patterns,
		"total":        len(patterns),
		"scope":        req.Scope,
		"analyzed_at":  time.Now(),
		"analysis_id":  result.AnalysisID,
		"memory_count": result.MemoryCount,
		"time_span":    timeSpan.String(),
	}, nil
}

// handleSuggestRelated suggests related content
func (h *Handler) handleSuggestRelated(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req SuggestRelatedRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse suggest related request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpSuggestRelated), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.ContentID == "" && req.Context == "" {
		return nil, fmt.Errorf("either content_id or context is required")
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}

	// Update session access if session provided
	if !req.SessionID.IsEmpty() {
		if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
			return nil, fmt.Errorf("failed to update session access: %w", err)
		}
	}

	var suggestions []RelatedSuggestion

	if req.ContentID != "" {
		// Find similar content based on content ID
		// First, get all content and then filter for the specific ID
		allContent, err := h.searchStore.GetByProject(ctx, req.ProjectID, nil)
		var targetContent []*types.Content
		if err == nil {
			for _, content := range allContent {
				if content.ID == req.ContentID {
					targetContent = append(targetContent, content)
					break
				}
			}
		}
		if err != nil || len(targetContent) == 0 {
			return nil, fmt.Errorf("target content not found: %s", req.ContentID)
		}

		// Find similar content using semantic search
		similarContent, err := h.searchStore.FindSimilar(ctx, targetContent[0].Content, req.ProjectID, req.SessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to find similar content: %w", err)
		}

		// Convert to suggestions
		for _, content := range similarContent {
			if content.ID != req.ContentID { // Exclude the original content
				suggestion := RelatedSuggestion{
					ContentID:    content.ID,
					Type:         content.Type,
					Title:        content.Summary,
					Summary:      content.Content[:min(200, len(content.Content))], // First 200 chars
					Relevance:    0.8,                                              // TODO: Calculate actual relevance score
					Relationship: "Semantically similar content",
					Tags:         content.Tags,
					CreatedAt:    content.CreatedAt,
					Metadata:     content.Metadata,
				}
				suggestions = append(suggestions, suggestion)
			}
		}
	} else if req.Context != "" {
		// Find content related to the context/query
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

		// Convert search results to suggestions
		for _, result := range searchResults.Results {
			suggestion := RelatedSuggestion{
				ContentID:    result.Content.ID,
				Type:         result.Content.Type,
				Title:        result.Content.Summary,
				Summary:      result.Content.Content[:min(200, len(result.Content.Content))],
				Relevance:    result.Relevance,
				Relationship: "Matches search context",
				Tags:         result.Content.Tags,
				CreatedAt:    result.Content.CreatedAt,
				Metadata:     result.Content.Metadata,
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	// Apply relation type filtering if specified
	if len(req.RelationTypes) > 0 {
		filteredSuggestions := make([]RelatedSuggestion, 0)
		for _, suggestion := range suggestions {
			for _, relType := range req.RelationTypes {
				if relType == "similar" && suggestion.Relationship == "Semantically similar content" {
					filteredSuggestions = append(filteredSuggestions, suggestion)
					break
				} else if relType == "references" && suggestion.Relationship == "Matches search context" {
					filteredSuggestions = append(filteredSuggestions, suggestion)
					break
				}
			}
		}
		suggestions = filteredSuggestions
	}

	// Apply limit
	if req.Limit > 0 && len(suggestions) > req.Limit {
		suggestions = suggestions[:req.Limit]
	}

	return map[string]interface{}{
		"suggestions": suggestions,
		"total":       len(suggestions),
		"context":     req.Context,
		"content_id":  req.ContentID,
	}, nil
}

// handleAnalyzeQuality analyzes content quality
func (h *Handler) handleAnalyzeQuality(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req AnalyzeQualityRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse analyze quality request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpAnalyzeQuality), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.Scope == "" {
		req.Scope = "project"
	}

	// Update session access if session provided
	if !req.SessionID.IsEmpty() {
		if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
			return nil, fmt.Errorf("failed to update session access: %w", err)
		}
	}

	var content *types.Content
	var contents []*types.Content

	if req.ContentID != "" {
		// Analyze specific content
		// First, get all content and then filter for the specific ID
		allContent, err := h.searchStore.GetByProject(ctx, req.ProjectID, nil)
		var contentItems []*types.Content
		if err == nil {
			for _, content := range allContent {
				if content.ID == req.ContentID {
					contentItems = append(contentItems, content)
					break
				}
			}
		}
		if err != nil || len(contentItems) == 0 {
			return nil, fmt.Errorf("content not found: %s", req.ContentID)
		}
		content = contentItems[0]
	} else {
		// Analyze project or session content
		var err error
		if !req.SessionID.IsEmpty() {
			contents, err = h.searchStore.GetBySession(ctx, req.ProjectID, req.SessionID, nil)
		} else {
			contents, err = h.searchStore.GetByProject(ctx, req.ProjectID, nil)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve content for analysis: %w", err)
		}
	}

	var analysis *QualityAnalysis

	if content != nil {
		// Analyze single content item
		qualityMetrics, err := h.aiService.AssessQuality(ctx, content.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to assess content quality: %w", err)
		}

		analysis = &QualityAnalysis{
			OverallScore:    qualityMetrics.OverallScore,
			Completeness:    qualityMetrics.Completeness,
			Clarity:         qualityMetrics.Clarity,
			Relevance:       qualityMetrics.Relevance,
			Recency:         h.calculateRecencyScore(content.CreatedAt),
			Issues:          h.extractQualityIssues(qualityMetrics, content.ID),
			Recommendations: h.extractRecommendations(qualityMetrics),
			AnalyzedAt:      time.Now(),
		}
	} else {
		// Analyze multiple content items (project/session analysis)
		totalScore := 0.0
		totalCompleteness := 0.0
		totalClarity := 0.0
		totalRelevance := 0.0
		totalRecency := 0.0
		issues := make([]QualityIssue, 0)
		allRecommendations := make(map[string]bool) // Use map to deduplicate

		for _, item := range contents {
			qualityMetrics, err := h.aiService.AssessQuality(ctx, item.Content)
			if err != nil {
				continue // Skip failed assessments but continue with others
			}

			totalScore += qualityMetrics.OverallScore
			totalCompleteness += qualityMetrics.Completeness
			totalClarity += qualityMetrics.Clarity
			totalRelevance += qualityMetrics.Relevance
			totalRecency += h.calculateRecencyScore(item.CreatedAt)

			// Extract issues from this content
			contentIssues := h.extractQualityIssues(qualityMetrics, item.ID)
			issues = append(issues, contentIssues...)

			// Collect recommendations
			for _, rec := range h.extractRecommendations(qualityMetrics) {
				allRecommendations[rec] = true
			}
		}

		count := float64(len(contents))
		if count > 0 {
			recommendations := make([]string, 0, len(allRecommendations))
			for rec := range allRecommendations {
				recommendations = append(recommendations, rec)
			}

			analysis = &QualityAnalysis{
				OverallScore:    totalScore / count,
				Completeness:    totalCompleteness / count,
				Clarity:         totalClarity / count,
				Relevance:       totalRelevance / count,
				Recency:         totalRecency / count,
				Issues:          issues,
				Recommendations: recommendations,
				AnalyzedAt:      time.Now(),
			}
		} else {
			// No content to analyze
			analysis = &QualityAnalysis{
				OverallScore: 0.0,
				AnalyzedAt:   time.Now(),
			}
		}
	}

	return analysis, nil
}

// handleDetectConflicts identifies conflicting information
func (h *Handler) handleDetectConflicts(ctx context.Context, params map[string]interface{}) (interface{}, error) {
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
func (h *Handler) handleGenerateInsights(ctx context.Context, params map[string]interface{}) (interface{}, error) {
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
func (h *Handler) handlePredictTrends(ctx context.Context, params map[string]interface{}) (interface{}, error) {
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
	if days <= 7 {
		return 1.0 // Very recent
	} else if days <= 30 {
		return 0.8 // Recent
	} else if days <= 90 {
		return 0.6 // Moderately recent
	} else if days <= 365 {
		return 0.4 // Older
	} else {
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
		recommendations = append(recommendations, "Consider restructuring content for better readability")
		recommendations = append(recommendations, "Add relevant examples and use cases")
		recommendations = append(recommendations, "Include links to related resources")
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
