//go:build ignore

// Package services provides enhanced AI-powered task suggestion capabilities.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// EnhancedSuggestionService provides AI-powered task suggestions with context analysis
type EnhancedSuggestionService struct {
	baseSuggestionService SuggestionService
	aiService             ports.AIService
	mcpClient             ports.MCPClient
	contextAnalyzer       ContextAnalyzer
	logger                *slog.Logger
	config                *SuggestionConfig
}

// NewEnhancedSuggestionService creates a new enhanced suggestion service
func NewEnhancedSuggestionService(
	baseSuggestionService SuggestionService,
	aiService ports.AIService,
	mcpClient ports.MCPClient,
	contextAnalyzer ContextAnalyzer,
	logger *slog.Logger,
) *EnhancedSuggestionService {
	return &EnhancedSuggestionService{
		baseSuggestionService: baseSuggestionService,
		aiService:             aiService,
		mcpClient:             mcpClient,
		contextAnalyzer:       contextAnalyzer,
		logger:                logger,
		config:                getDefaultSuggestionConfig(),
	}
}

// AI-powered suggestion types
type AISuggestionRequest struct {
	WorkContext    *entities.WorkContext  `json:"work_context"`
	RequestType    string                 `json:"request_type"`
	ContextualInfo map[string]interface{} `json:"contextual_info"`
	PreferredStyle string                 `json:"preferred_style"`
	MaxSuggestions int                    `json:"max_suggestions"`
	Repository     string                 `json:"repository"`
}

type AISuggestionResponse struct {
	Suggestions     []*AISuggestionResult `json:"suggestions"`
	ContextInsights []string              `json:"context_insights"`
	Reasoning       string                `json:"reasoning"`
	ModelUsed       string                `json:"model_used"`
	TokensUsed      int                   `json:"tokens_used"`
}

type AISuggestionResult struct {
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Type          string            `json:"type"`
	Confidence    float64           `json:"confidence"`
	Relevance     float64           `json:"relevance"`
	Urgency       float64           `json:"urgency"`
	EstimatedTime int               `json:"estimated_time_minutes"`
	Category      string            `json:"category"`
	Tags          []string          `json:"tags"`
	Prerequisites []string          `json:"prerequisites"`
	Benefits      []string          `json:"benefits"`
	Reasoning     string            `json:"reasoning"`
	Metadata      map[string]string `json:"metadata"`
}

// GenerateTaskSuggestions generates AI-enhanced task suggestions
func (s *EnhancedSuggestionService) GenerateTaskSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	// Get base suggestions first
	baseSuggestions, err := s.baseSuggestionService.GenerateTaskSuggestions(ctx, workContext)
	if err != nil {
		return nil, fmt.Errorf("failed to generate base suggestions: %w", err)
	}

	// Generate AI-enhanced suggestions
	aiSuggestions, err := s.generateAISuggestions(ctx, workContext)
	if err != nil {
		s.logger.Warn("AI suggestion generation failed, using base suggestions only",
			slog.String("error", err.Error()))
		return baseSuggestions, nil
	}

	// Combine and rank all suggestions
	allSuggestions := append(baseSuggestions, aiSuggestions...)

	// Apply AI-powered ranking and filtering
	rankedSuggestions := s.rankSuggestionsWithAI(ctx, allSuggestions, workContext)

	s.logger.Info("generated enhanced task suggestions",
		slog.Int("base_suggestions", len(baseSuggestions)),
		slog.Int("ai_suggestions", len(aiSuggestions)),
		slog.Int("final_suggestions", len(rankedSuggestions)))

	return rankedSuggestions, nil
}

// generateAISuggestions generates creative AI-powered suggestions
func (s *EnhancedSuggestionService) generateAISuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	// Create comprehensive context for AI
	contextPrompt := s.buildContextPrompt(workContext)

	// Make request to MCP AI service
	request := &ports.AIAnalysisRequest{
		Type:    "task_suggestions",
		Content: contextPrompt,
		Context: map[string]interface{}{
			"repository":         workContext.Repository,
			"energy_level":       workContext.EnergyLevel,
			"focus_level":        workContext.FocusLevel,
			"productivity_score": workContext.ProductivityScore,
			"stress_indicators":  workContext.StressIndicators,
			"current_time":       time.Now().Format(time.RFC3339),
			"active_patterns":    workContext.ActivePatterns,
		},
		Options: map[string]interface{}{
			"max_suggestions":  s.config.MaxAISuggestions,
			"creativity_level": "high",
			"focus_areas":      []string{"productivity", "learning", "optimization", "creative"},
		},
	}

	response, err := s.aiService.AnalyzeWithAI(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("AI analysis request failed: %w", err)
	}

	// Parse AI response into suggestions
	return s.parseAISuggestionResponse(response, workContext)
}

// buildContextPrompt creates a comprehensive prompt for AI analysis
func (s *EnhancedSuggestionService) buildContextPrompt(workContext *entities.WorkContext) string {
	prompt := fmt.Sprintf(`Analyze the following work context and generate intelligent task suggestions:

Current State:
- Energy Level: %.1f%% 
- Focus Level: %.1f%%
- Productivity Score: %.1f%%
- Stress Level: %s

Current Tasks:
`, workContext.EnergyLevel*100, workContext.FocusLevel*100,
		workContext.ProductivityScore*100, s.getStressLevel(workContext))

	for i, task := range workContext.CurrentTasks {
		if i < 5 { // Limit to top 5 tasks
			prompt += fmt.Sprintf("- %s (Priority: %s, Status: %s)\n",
				task.Content, task.Priority, task.Status)
		}
	}

	prompt += fmt.Sprintf(`
Recent Patterns:
`)

	for i, pattern := range workContext.ActivePatterns {
		if i < 3 { // Limit to top 3 patterns
			prompt += fmt.Sprintf("- %s (Confidence: %.1f%%)\n",
				pattern.Name, pattern.Confidence*100)
		}
	}

	prompt += `
Generate 5-7 intelligent, actionable task suggestions that:

1. **Optimize current productivity** - Based on energy/focus levels
2. **Address identified patterns** - Leverage or break patterns as needed  
3. **Provide learning opportunities** - Suggest skill development or knowledge gaps
4. **Offer creative solutions** - Think outside the box for process improvements
5. **Consider time context** - Account for current time of day and work rhythms
6. **Address stress factors** - Help manage or reduce stress indicators
7. **Build momentum** - Create positive feedback loops

For each suggestion, provide:
- Clear, actionable title (max 60 characters)
- Detailed description with specific next steps
- Confidence score (0.0-1.0) based on context fit
- Relevance score (0.0-1.0) for current situation  
- Urgency score (0.0-1.0) for timing importance
- Estimated time required (in minutes)
- Category (productivity, learning, creative, optimization, health, planning)
- Benefits and reasoning for the suggestion

Return as JSON array with structure:
{
	"suggestions": [
		{
			"title": "specific actionable title",
			"description": "detailed description with next steps",
			"type": "next|optimize|learn|creative|break|pattern",
			"confidence": 0.0-1.0,
			"relevance": 0.0-1.0, 
			"urgency": 0.0-1.0,
			"estimated_time_minutes": number,
			"category": "productivity|learning|creative|optimization|health|planning",
			"tags": ["relevant", "tags"],
			"prerequisites": ["if any"],
			"benefits": ["specific benefits"],
			"reasoning": "why this suggestion makes sense now"
		}
	],
	"context_insights": ["key insights about current context"],
	"reasoning": "overall analysis of the work context"
}`

	return prompt
}

// parseAISuggestionResponse parses AI response into TaskSuggestion entities
func (s *EnhancedSuggestionService) parseAISuggestionResponse(response *ports.AIAnalysisResponse, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	var aiResponse AISuggestionResponse
	if err := json.Unmarshal([]byte(response.Analysis), &aiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	var suggestions []*entities.TaskSuggestion

	for _, aiSuggestion := range aiResponse.Suggestions {
		// Convert AI suggestion type to entity type
		suggestionType := s.convertAITypeToEntity(aiSuggestion.Type)

		suggestion := entities.NewTaskSuggestion(
			suggestionType,
			aiSuggestion.Title,
			entities.SuggestionSource{
				Type:       "ai",
				Name:       "enhanced_ai_service",
				Algorithm:  "llm_context_analysis",
				Confidence: aiSuggestion.Confidence,
			},
			workContext.Repository,
		)

		// Set AI-generated properties
		suggestion.Description = aiSuggestion.Description
		suggestion.Confidence = aiSuggestion.Confidence
		suggestion.Relevance = aiSuggestion.Relevance
		suggestion.Urgency = aiSuggestion.Urgency
		suggestion.Reasoning = aiSuggestion.Reasoning
		suggestion.EstimatedDuration = time.Duration(aiSuggestion.EstimatedTime) * time.Minute
		suggestion.Category = aiSuggestion.Category
		suggestion.Tags = aiSuggestion.Tags

		// Add AI-specific metadata
		suggestion.Metadata["ai_generated"] = "true"
		suggestion.Metadata["ai_model"] = response.ModelUsed
		suggestion.Metadata["ai_category"] = aiSuggestion.Category
		suggestion.Metadata["prerequisites"] = strings.Join(aiSuggestion.Prerequisites, ", ")
		suggestion.Metadata["benefits"] = strings.Join(aiSuggestion.Benefits, ", ")

		suggestions = append(suggestions, suggestion)
	}

	s.logger.Debug("parsed AI suggestions",
		slog.Int("suggestions_count", len(suggestions)),
		slog.String("model_used", response.ModelUsed),
		slog.Int("tokens_used", response.TokensUsed))

	return suggestions, nil
}

// generateTemplateSuggestions generates template-based suggestions using AI
func (s *EnhancedSuggestionService) generateTemplateSuggestions(workContext *entities.WorkContext) []*entities.TaskSuggestion {
	// Get best matching templates from MCP memory
	templateRequest := &ports.MemorySearchRequest{
		Query:      s.buildTemplateQuery(workContext),
		Repository: workContext.Repository,
		Options: map[string]interface{}{
			"type":        "template",
			"max_results": 5,
		},
	}

	// Search for relevant templates
	searchResponse, err := s.mcpClient.SearchMemory(context.Background(), templateRequest)
	if err != nil {
		s.logger.Warn("template search failed", slog.String("error", err.Error()))
		return []*entities.TaskSuggestion{}
	}

	var suggestions []*entities.TaskSuggestion

	// Convert templates to suggestions
	for _, result := range searchResponse.Results {
		suggestion := entities.NewTaskSuggestion(
			entities.SuggestionTypeTemplate,
			fmt.Sprintf("Apply template: %s", result.Title),
			entities.SuggestionSource{
				Type:       "template",
				Name:       "memory_template_matcher",
				Algorithm:  "similarity_search",
				Confidence: result.Confidence,
			},
			workContext.Repository,
		)

		suggestion.Description = fmt.Sprintf("Apply the '%s' template based on similar past work", result.Title)
		suggestion.Confidence = result.Confidence
		suggestion.Relevance = s.calculateTemplateRelevance(result, workContext)
		suggestion.Urgency = 0.5 // Templates are generally medium urgency
		suggestion.TemplateID = result.ID
		suggestion.Reasoning = fmt.Sprintf("Template matches current context with %.1f%% confidence", result.Confidence*100)

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

// generateMorningStartupBatch generates AI-powered morning startup suggestions
func (s *EnhancedSuggestionService) generateMorningStartupBatch(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	prompt := s.buildMorningStartupPrompt(workContext)

	request := &ports.AIAnalysisRequest{
		Type:    "morning_startup",
		Content: prompt,
		Context: map[string]interface{}{
			"time_of_day": "morning",
			"batch_type":  "startup",
			"repository":  workContext.Repository,
		},
		Options: map[string]interface{}{
			"max_suggestions": 5,
			"focus_areas":     []string{"planning", "energy", "prioritization", "momentum"},
		},
	}

	response, err := s.aiService.AnalyzeWithAI(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("morning startup AI analysis failed: %w", err)
	}

	return s.parseAISuggestionResponse(response, workContext)
}

// generateFocusedWorkBatch generates AI-powered deep work suggestions
func (s *EnhancedSuggestionService) generateFocusedWorkBatch(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	prompt := s.buildFocusedWorkPrompt(workContext)

	request := &ports.AIAnalysisRequest{
		Type:    "focused_work",
		Content: prompt,
		Context: map[string]interface{}{
			"focus_session": true,
			"batch_type":    "focused_work",
			"repository":    workContext.Repository,
		},
		Options: map[string]interface{}{
			"max_suggestions": 4,
			"focus_areas":     []string{"deep_work", "concentration", "flow_state", "complexity"},
		},
	}

	response, err := s.aiService.AnalyzeWithAI(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("focused work AI analysis failed: %w", err)
	}

	return s.parseAISuggestionResponse(response, workContext)
}

// generateQuickWinsBatch generates AI-powered quick wins suggestions
func (s *EnhancedSuggestionService) generateQuickWinsBatch(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	prompt := s.buildQuickWinsPrompt(workContext)

	request := &ports.AIAnalysisRequest{
		Type:    "quick_wins",
		Content: prompt,
		Context: map[string]interface{}{
			"quick_wins": true,
			"batch_type": "quick_wins",
			"repository": workContext.Repository,
		},
		Options: map[string]interface{}{
			"max_suggestions": 6,
			"focus_areas":     []string{"momentum", "completion", "energy", "confidence"},
		},
	}

	response, err := s.aiService.AnalyzeWithAI(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("quick wins AI analysis failed: %w", err)
	}

	return s.parseAISuggestionResponse(response, workContext)
}

// rankSuggestionsWithAI applies AI-powered ranking to suggestions
func (s *EnhancedSuggestionService) rankSuggestionsWithAI(ctx context.Context, suggestions []*entities.TaskSuggestion, workContext *entities.WorkContext) []*entities.TaskSuggestion {
	if len(suggestions) <= 1 {
		return suggestions
	}

	// Use AI to re-rank suggestions based on current context
	rankingPrompt := s.buildRankingPrompt(suggestions, workContext)

	request := &ports.AIAnalysisRequest{
		Type:    "suggestion_ranking",
		Content: rankingPrompt,
		Context: map[string]interface{}{
			"ranking_task": true,
			"repository":   workContext.Repository,
		},
		Options: map[string]interface{}{
			"return_ranked_list": true,
		},
	}

	response, err := s.aiService.AnalyzeWithAI(ctx, request)
	if err != nil {
		s.logger.Warn("AI ranking failed, using default ranking", slog.String("error", err.Error()))
		return s.applyDefaultRanking(suggestions)
	}

	rankedSuggestions := s.parseRankingResponse(response, suggestions)
	if len(rankedSuggestions) == 0 {
		return s.applyDefaultRanking(suggestions)
	}

	return rankedSuggestions
}

// Helper methods

func (s *EnhancedSuggestionService) buildTemplateQuery(workContext *entities.WorkContext) string {
	var queryParts []string

	// Add current task types
	taskTypes := workContext.GetActiveTaskTypes()
	if len(taskTypes) > 0 {
		queryParts = append(queryParts, "task types: "+strings.Join(taskTypes, " "))
	}

	// Add primary patterns
	patterns := workContext.GetPrimaryPatterns()
	if len(patterns) > 0 {
		queryParts = append(queryParts, "patterns: "+patterns[0].Name)
	}

	// Add context indicators
	if workContext.EnergyLevel < 0.5 {
		queryParts = append(queryParts, "low energy")
	}
	if workContext.FocusLevel > 0.8 {
		queryParts = append(queryParts, "high focus")
	}

	return strings.Join(queryParts, " ")
}

func (s *EnhancedSuggestionService) buildMorningStartupPrompt(workContext *entities.WorkContext) string {
	return fmt.Sprintf(`Generate morning startup suggestions for optimal day planning:

Current Context:
- Energy: %.1f%% (morning energy levels)
- Current Goals: %d active tasks
- Repository: %s

Generate 5 morning startup suggestions focusing on:
1. Day planning and prioritization
2. Energy optimization 
3. Goal setting and clarity
4. Momentum building activities
5. Environment preparation

Each suggestion should be actionable and help establish a productive foundation for the day.`,
		workContext.EnergyLevel*100, len(workContext.CurrentTasks), workContext.Repository)
}

func (s *EnhancedSuggestionService) buildFocusedWorkPrompt(workContext *entities.WorkContext) string {
	return fmt.Sprintf(`Generate deep work suggestions for focused productivity:

Current Context:
- Focus Level: %.1f%%
- Energy Level: %.1f%%
- Active Tasks: %d
- Stress Level: %s

Generate 4 focused work suggestions emphasizing:
1. Deep work and concentration
2. Complex task tackling
3. Flow state optimization  
4. Distraction elimination

Prioritize suggestions that leverage high focus periods for maximum productivity.`,
		workContext.FocusLevel*100, workContext.EnergyLevel*100,
		len(workContext.CurrentTasks), s.getStressLevel(workContext))
}

func (s *EnhancedSuggestionService) buildQuickWinsPrompt(workContext *entities.WorkContext) string {
	return fmt.Sprintf(`Generate quick wins suggestions for momentum and confidence:

Current Context:
- Productivity Score: %.1f%%
- Energy Level: %.1f%%
- Completed Tasks Today: %d

Generate 6 quick win suggestions focusing on:
1. Easy completion opportunities
2. Momentum building activities  
3. Confidence boosting tasks
4. Energy restoration
5. Progress visibility
6. Motivation enhancement

Emphasize tasks that can be completed quickly (5-30 minutes) with high impact on motivation.`,
		workContext.ProductivityScore*100, workContext.EnergyLevel*100,
		workContext.GetCompletedTasksToday())
}

func (s *EnhancedSuggestionService) buildRankingPrompt(suggestions []*entities.TaskSuggestion, workContext *entities.WorkContext) string {
	prompt := fmt.Sprintf(`Rank these %d task suggestions by relevance to current context:

Current Context:
- Energy: %.1f%%
- Focus: %.1f%% 
- Productivity: %.1f%%
- Time: %s

Suggestions to rank:
`, len(suggestions), workContext.EnergyLevel*100, workContext.FocusLevel*100,
		workContext.ProductivityScore*100, time.Now().Format("15:04"))

	for i, suggestion := range suggestions {
		prompt += fmt.Sprintf("%d. %s (Type: %s, Confidence: %.1f%%)\n",
			i+1, suggestion.Title, suggestion.Type, suggestion.Confidence*100)
	}

	prompt += `
Return ranked order as JSON array of indices (1-based):
{"ranked_order": [1, 3, 2, 5, 4, ...]}`

	return prompt
}

func (s *EnhancedSuggestionService) parseRankingResponse(response *ports.AIAnalysisResponse, originalSuggestions []*entities.TaskSuggestion) []*entities.TaskSuggestion {
	var ranking struct {
		RankedOrder []int `json:"ranked_order"`
	}

	if err := json.Unmarshal([]byte(response.Analysis), &ranking); err != nil {
		return originalSuggestions
	}

	var rankedSuggestions []*entities.TaskSuggestion
	for _, index := range ranking.RankedOrder {
		if index >= 1 && index <= len(originalSuggestions) {
			rankedSuggestions = append(rankedSuggestions, originalSuggestions[index-1])
		}
	}

	return rankedSuggestions
}

func (s *EnhancedSuggestionService) applyDefaultRanking(suggestions []*entities.TaskSuggestion) []*entities.TaskSuggestion {
	// Simple scoring-based ranking
	type scoredSuggestion struct {
		suggestion *entities.TaskSuggestion
		score      float64
	}

	var scored []scoredSuggestion
	for _, suggestion := range suggestions {
		score := (suggestion.Confidence + suggestion.Relevance + suggestion.Urgency) / 3.0
		scored = append(scored, scoredSuggestion{suggestion, score})
	}

	// Sort by score descending
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[i].score < scored[j].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	var ranked []*entities.TaskSuggestion
	for _, s := range scored {
		ranked = append(ranked, s.suggestion)
	}

	return ranked
}

func (s *EnhancedSuggestionService) convertAITypeToEntity(aiType string) entities.SuggestionType {
	switch strings.ToLower(aiType) {
	case "next":
		return entities.SuggestionTypeNext
	case "optimize":
		return entities.SuggestionTypeOptimize
	case "learn":
		return entities.SuggestionTypeLearn
	case "creative":
		return entities.SuggestionTypeCreative
	case "break":
		return entities.SuggestionTypeBreak
	case "pattern":
		return entities.SuggestionTypePattern
	default:
		return entities.SuggestionTypeNext
	}
}

func (s *EnhancedSuggestionService) calculateTemplateRelevance(result *ports.MemorySearchResult, workContext *entities.WorkContext) float64 {
	baseRelevance := result.Confidence

	// Boost relevance if template matches current work patterns
	if len(workContext.ActivePatterns) > 0 {
		for _, pattern := range workContext.ActivePatterns {
			if strings.Contains(strings.ToLower(result.Content), strings.ToLower(pattern.Name)) {
				baseRelevance += 0.1
			}
		}
	}

	// Limit to maximum relevance
	if baseRelevance > 1.0 {
		baseRelevance = 1.0
	}

	return baseRelevance
}

func (s *EnhancedSuggestionService) getStressLevel(workContext *entities.WorkContext) string {
	if workContext.IsHighStress() {
		return "high"
	} else if len(workContext.StressIndicators) > 0 {
		return "medium"
	}
	return "low"
}

func getDefaultSuggestionConfig() *SuggestionConfig {
	return &SuggestionConfig{
		MaxSuggestions:         10,
		MaxAISuggestions:       5,
		MinConfidenceThreshold: 0.3,
		MinRelevanceThreshold:  0.4,
		EnableTemplates:        true,
		EnableAISuggestions:    true,
		RankingAlgorithm:       "ai_enhanced",
	}
}
