// Package services provides hybrid suggestion service that combines local and server-based suggestions
package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// HybridSuggestionService combines local and server-based suggestions
type HybridSuggestionService struct {
	localService SuggestionService
	mcpClient    ports.MCPClient
	logger       *slog.Logger
}

// NewHybridSuggestionService creates a new hybrid suggestion service
func NewHybridSuggestionService(
	localService SuggestionService,
	mcpClient ports.MCPClient,
	logger *slog.Logger,
) SuggestionService {
	return &HybridSuggestionService{
		localService: localService,
		mcpClient:    mcpClient,
		logger:       logger,
	}
}

// GenerateSuggestionsForContext generates suggestions using both local and server intelligence
func (h *HybridSuggestionService) GenerateSuggestionsForContext(
	ctx context.Context,
	workContext *entities.WorkContext,
	maxResults int,
) ([]*entities.TaskSuggestion, error) {
	// Always get local suggestions first (core standalone functionality)
	localSuggestions, err := h.localService.GenerateSuggestionsForContext(ctx, workContext, maxResults)
	if err != nil {
		h.logger.Warn("failed to get local suggestions", slog.Any("error", err))
		localSuggestions = []*entities.TaskSuggestion{} // Continue without local suggestions
	}

	// If server is available, enhance with server-based suggestions
	serverSuggestions := h.getServerSuggestions(ctx, workContext, maxResults)

	// Combine and rank suggestions
	allSuggestions := h.combineAndRankSuggestions(localSuggestions, serverSuggestions, maxResults)

	return allSuggestions, nil
}

// getServerSuggestions queries the server for pattern-based suggestions
func (h *HybridSuggestionService) getServerSuggestions(
	ctx context.Context,
	workContext *entities.WorkContext,
	maxResults int,
) []*entities.TaskSuggestion {
	// Check if MCP client is available and online
	if h.mcpClient == nil || !h.mcpClient.IsOnline() {
		h.logger.Debug("MCP client not available, skipping server suggestions")
		return nil
	}

	// Build context for pattern engine analysis
	contextContent := h.buildContextContent(workContext)

	// Query server for pattern suggestions using memory_intelligence tool
	suggestions, err := h.queryServerPatterns(ctx, contextContent, workContext.Repository, maxResults)
	if err != nil {
		h.logger.Warn("failed to get server suggestions", slog.Any("error", err))
		return nil
	}

	h.logger.Info("retrieved server-based suggestions",
		slog.Int("count", len(suggestions)),
		slog.String("repository", workContext.Repository))

	return suggestions
}

// buildContextContent builds context content from work context for pattern analysis
func (h *HybridSuggestionService) buildContextContent(workContext *entities.WorkContext) string {
	var parts []string

	// Add current goals
	if len(workContext.Goals) > 0 {
		parts = append(parts, "Current goals:")
		for _, goal := range workContext.Goals {
			parts = append(parts, fmt.Sprintf("- %s (priority: %s)", goal.Description, goal.Priority))
		}
	}

	// Add current tasks context
	if len(workContext.CurrentTasks) > 0 {
		parts = append(parts, "Current tasks:")
		for _, task := range workContext.CurrentTasks {
			parts = append(parts, fmt.Sprintf("- %s [%s]", task.Content, task.Status))
		}
	}

	// Add environmental context
	parts = append(parts,
		fmt.Sprintf("Context: %s, %s", workContext.TimeOfDay, workContext.DayOfWeek),
		fmt.Sprintf("Focus: %.1f, Energy: %.1f", workContext.FocusLevel, workContext.EnergyLevel),
	)

	return strings.Join(parts, "\n")
}

// queryServerPatterns queries the server for pattern-based suggestions
func (h *HybridSuggestionService) queryServerPatterns(
	ctx context.Context,
	contextContent string,
	repository string,
	_ int,
) ([]*entities.TaskSuggestion, error) {
	// Use memory_intelligence operation for pattern suggestions
	options := map[string]interface{}{
		"current_context": contextContent,
		"repository":      repository,
		"session_id":      fmt.Sprintf("cli-suggest-%d", ctx.Value("timestamp")),
	}

	// Send request via MCP client
	response, err := h.mcpClient.QueryIntelligence(ctx, "suggest_related", options)
	if err != nil {
		return nil, fmt.Errorf("MCP intelligence query failed: %w", err)
	}

	// Parse pattern suggestions into task suggestions
	return h.parseServerSuggestions(response, repository)
}

// parseServerSuggestions converts server pattern responses to task suggestions
func (h *HybridSuggestionService) parseServerSuggestions(
	response map[string]interface{},
	repository string,
) ([]*entities.TaskSuggestion, error) {
	suggestions := []*entities.TaskSuggestion{}

	// Extract suggestions from response
	if result, ok := response["result"].(map[string]interface{}); ok {
		if suggestionsData, ok := result["suggestions"].([]interface{}); ok {
			for _, item := range suggestionsData {
				if suggestionData, ok := item.(map[string]interface{}); ok {
					suggestion := h.convertServerSuggestion(suggestionData, repository)
					if suggestion != nil {
						suggestions = append(suggestions, suggestion)
					}
				}
			}
		}
	}

	return suggestions, nil
}

// convertServerSuggestion converts a server pattern to a task suggestion
func (h *HybridSuggestionService) convertServerSuggestion(
	data map[string]interface{},
	repository string,
) *entities.TaskSuggestion {
	// Extract pattern data
	name, _ := data["name"].(string)
	description, _ := data["description"].(string)
	patternType, _ := data["type"].(string)
	confidence, _ := data["confidence"].(float64)

	if name == "" {
		return nil
	}

	// Convert pattern to task suggestion
	suggestion := &entities.TaskSuggestion{
		Content:     name,
		Description: description,
		Priority:    h.mapPatternTypeToPriority(patternType),
		Tags:        []string{"ai-suggested", "pattern-based", patternType},
		Confidence:  confidence,
		Reasoning:   fmt.Sprintf("Based on %s pattern analysis from memory server", patternType),
		Source: entities.SuggestionSource{
			Type:       "server-pattern",
			Name:       "mcp-pattern-engine",
			Confidence: confidence,
			Algorithm:  "pattern-matching",
			Metadata:   map[string]interface{}{"pattern_type": patternType},
		},
		Repository: repository,
	}

	return suggestion
}

// mapPatternTypeToPriority maps pattern types to task priorities
func (h *HybridSuggestionService) mapPatternTypeToPriority(patternType string) string {
	switch strings.ToLower(patternType) {
	case "error", "bug":
		return constants.SeverityHigh
	case "optimization", "refactoring":
		return constants.SeverityMedium
	case "architectural", "workflow":
		return constants.SeverityMedium
	default:
		return constants.SeverityLow
	}
}

// combineAndRankSuggestions merges local and server suggestions, deduplicates, and ranks them
func (h *HybridSuggestionService) combineAndRankSuggestions(
	local []*entities.TaskSuggestion,
	server []*entities.TaskSuggestion,
	maxResults int,
) []*entities.TaskSuggestion {
	// Combine all suggestions
	allSuggestions := make([]*entities.TaskSuggestion, 0, len(local)+len(server))
	allSuggestions = append(allSuggestions, local...)
	allSuggestions = append(allSuggestions, server...)

	// Deduplicate based on content similarity
	deduped := h.deduplicateSuggestions(allSuggestions)

	// Rank by confidence and priority
	ranked := h.rankSuggestions(deduped)

	// Limit results
	if len(ranked) > maxResults {
		ranked = ranked[:maxResults]
	}

	return ranked
}

// deduplicateSuggestions removes duplicate suggestions based on content similarity
func (h *HybridSuggestionService) deduplicateSuggestions(suggestions []*entities.TaskSuggestion) []*entities.TaskSuggestion {
	seen := make(map[string]*entities.TaskSuggestion)
	unique := []*entities.TaskSuggestion{}

	for _, suggestion := range suggestions {
		// Create a key based on normalized content
		key := strings.ToLower(strings.TrimSpace(suggestion.Content))

		// If we've seen similar content, keep the one with higher confidence
		if existing, exists := seen[key]; exists {
			if suggestion.Confidence > existing.Confidence {
				// Replace with higher confidence suggestion
				for i, s := range unique {
					if s == existing {
						unique[i] = suggestion
						break
					}
				}
				seen[key] = suggestion
			}
		} else {
			// New suggestion
			seen[key] = suggestion
			unique = append(unique, suggestion)
		}
	}

	return unique
}

// rankSuggestions sorts suggestions by confidence and priority
func (h *HybridSuggestionService) rankSuggestions(suggestions []*entities.TaskSuggestion) []*entities.TaskSuggestion {
	// Simple ranking: sort by confidence descending
	// In production, this could be more sophisticated
	ranked := make([]*entities.TaskSuggestion, len(suggestions))
	copy(ranked, suggestions)

	// Sort by confidence (descending) with priority as tiebreaker
	for i := 0; i < len(ranked)-1; i++ {
		for j := i + 1; j < len(ranked); j++ {
			// Higher confidence wins
			if ranked[j].Confidence > ranked[i].Confidence {
				ranked[i], ranked[j] = ranked[j], ranked[i]
			} else if ranked[j].Confidence == ranked[i].Confidence {
				// If confidence is equal, prioritize by priority
				if h.priorityScore(ranked[j].Priority) > h.priorityScore(ranked[i].Priority) {
					ranked[i], ranked[j] = ranked[j], ranked[i]
				}
			}
		}
	}

	return ranked
}

// priorityScore assigns numeric scores to priorities for ranking
func (h *HybridSuggestionService) priorityScore(priority string) int {
	switch strings.ToLower(priority) {
	case constants.SeverityHigh:
		return 3
	case constants.SeverityMedium:
		return 2
	case constants.SeverityLow:
		return 1
	default:
		return 0
	}
}

// Implement remaining SuggestionService interface methods by delegating to local service

func (h *HybridSuggestionService) GenerateSuggestions(ctx context.Context, repository string, maxSuggestions int) ([]*entities.TaskSuggestion, error) {
	return h.localService.GenerateSuggestions(ctx, repository, maxSuggestions)
}

func (h *HybridSuggestionService) GenerateNextTaskSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	return h.localService.GenerateNextTaskSuggestions(ctx, workContext)
}

func (h *HybridSuggestionService) GeneratePatternBasedSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	// For pattern-based suggestions, we can enhance with server patterns
	localSuggestions, err := h.localService.GeneratePatternBasedSuggestions(ctx, workContext)
	if err != nil {
		h.logger.Warn("failed to get local pattern suggestions", slog.Any("error", err))
		localSuggestions = []*entities.TaskSuggestion{}
	}

	// Get server pattern suggestions if available
	serverSuggestions := h.getServerSuggestions(ctx, workContext, 5)

	// Combine pattern-based suggestions
	return h.combineAndRankSuggestions(localSuggestions, serverSuggestions, 10), nil
}

func (h *HybridSuggestionService) GenerateOptimizationSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	return h.localService.GenerateOptimizationSuggestions(ctx, workContext)
}

func (h *HybridSuggestionService) GenerateBreakSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	return h.localService.GenerateBreakSuggestions(ctx, workContext)
}

func (h *HybridSuggestionService) RankSuggestions(suggestions []*entities.TaskSuggestion, workContext *entities.WorkContext) []*entities.TaskSuggestion {
	return h.localService.RankSuggestions(suggestions, workContext)
}

func (h *HybridSuggestionService) FilterSuggestions(suggestions []*entities.TaskSuggestion, preferences *entities.UserPreferences) []*entities.TaskSuggestion {
	return h.localService.FilterSuggestions(suggestions, preferences)
}

func (h *HybridSuggestionService) ProcessFeedback(ctx context.Context, suggestionID string, feedback *entities.SuggestionFeedback) error {
	return h.localService.ProcessFeedback(ctx, suggestionID, feedback)
}

func (h *HybridSuggestionService) GenerateSuggestionBatch(ctx context.Context, repository string, batchType string) (*entities.SuggestionBatch, error) {
	return h.localService.GenerateSuggestionBatch(ctx, repository, batchType)
}

func (h *HybridSuggestionService) GetPersonalizedSuggestions(ctx context.Context, repository string, preferences *entities.UserPreferences) ([]*entities.TaskSuggestion, error) {
	return h.localService.GetPersonalizedSuggestions(ctx, repository, preferences)
}
