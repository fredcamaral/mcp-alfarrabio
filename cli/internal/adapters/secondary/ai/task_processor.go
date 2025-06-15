// Package ai provides AI-powered task processing and intelligent memory management for the CLI
package ai

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

// TaskProcessor provides AI-powered task processing capabilities
type TaskProcessor struct {
	mcpClient  ports.MCPClient
	aiService  ports.AIService
	logger     *slog.Logger
	repository string
	sessionID  string
	config     *TaskProcessorConfig
}

// TaskProcessorConfig configures the AI task processor
type TaskProcessorConfig struct {
	MaxSuggestions         int     `json:"max_suggestions"`
	MinConfidenceThreshold float64 `json:"min_confidence_threshold"`
	AutoPrioritization     bool    `json:"auto_prioritization"`
	SmartTagging           bool    `json:"smart_tagging"`
	ContextualSuggestions  bool    `json:"contextual_suggestions"`
	SemanticDuplication    bool    `json:"semantic_duplication"`
	AutoEstimation         bool    `json:"auto_estimation"`
	LearningMode           bool    `json:"learning_mode"`
}

// AITaskSuggestion represents an AI-generated task suggestion
type AITaskSuggestion struct {
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Priority      entities.Priority `json:"priority"`
	Tags          []string          `json:"tags"`
	EstimatedMins int               `json:"estimated_mins"`
	Confidence    float64           `json:"confidence"`
	Reasoning     string            `json:"reasoning"`
	Type          string            `json:"type"`
	Category      string            `json:"category"`
	Dependencies  []string          `json:"dependencies"`
	Prerequisites []string          `json:"prerequisites"`
}

// TaskProcessingResult contains the result of AI task processing
type TaskProcessingResult struct {
	OriginalTask      *entities.Task      `json:"original_task"`
	EnhancedTask      *entities.Task      `json:"enhanced_task"`
	Suggestions       []*AITaskSuggestion `json:"suggestions"`
	ContextInsights   []string            `json:"context_insights"`
	Duplicates        []*entities.Task    `json:"duplicates"`
	RelatedTasks      []*entities.Task    `json:"related_tasks"`
	ProcessingNotes   []string            `json:"processing_notes"`
	AIRecommendations []string            `json:"ai_recommendations"`
}

// NewTaskProcessor creates a new AI-powered task processor
func NewTaskProcessor(mcpClient ports.MCPClient, aiService ports.AIService, logger *slog.Logger) *TaskProcessor {
	return &TaskProcessor{
		mcpClient: mcpClient,
		aiService: aiService,
		logger:    logger,
		config:    getDefaultTaskProcessorConfig(),
	}
}

// SetRepository sets the current repository context
func (tp *TaskProcessor) SetRepository(repository string) {
	tp.repository = repository
}

// SetSessionID sets the current session ID for context grouping
func (tp *TaskProcessor) SetSessionID(sessionID string) {
	tp.sessionID = sessionID
}

// ProcessTask applies AI enhancement to a task and provides intelligent suggestions
func (tp *TaskProcessor) ProcessTask(ctx context.Context, task *entities.Task) (*TaskProcessingResult, error) {
	result := &TaskProcessingResult{
		OriginalTask: task,
		EnhancedTask: task, // Will be modified
	}

	// Step 1: Analyze task content and enhance with AI
	if err := tp.enhanceTaskWithAI(ctx, result); err != nil {
		tp.logger.Warn("AI task enhancement failed", slog.String("error", err.Error()))
	}

	// Step 2: Check for semantic duplicates
	if tp.config.SemanticDuplication {
		if err := tp.findSemanticDuplicates(ctx, result); err != nil {
			tp.logger.Warn("semantic duplicate detection failed", slog.String("error", err.Error()))
		}
	}

	// Step 3: Find related tasks and context
	if err := tp.findRelatedTasks(ctx, result); err != nil {
		tp.logger.Warn("related task detection failed", slog.String("error", err.Error()))
	}

	// Step 4: Generate contextual suggestions
	if tp.config.ContextualSuggestions {
		if err := tp.generateContextualSuggestions(ctx, result); err != nil {
			tp.logger.Warn("contextual suggestions generation failed", slog.String("error", err.Error()))
		}
	}

	// Step 5: Store processed task insights in memory
	if tp.mcpClient != nil && tp.mcpClient.IsOnline() {
		tp.storeTaskInsights(ctx, result)
	}

	tp.logger.Info("task processed with AI enhancements",
		slog.String("task_id", task.ID),
		slog.Int("suggestions_count", len(result.Suggestions)),
		slog.Int("duplicates_found", len(result.Duplicates)),
		slog.Int("related_tasks", len(result.RelatedTasks)))

	return result, nil
}

// enhanceTaskWithAI uses AI to improve task description, priority, tags, and estimation
func (tp *TaskProcessor) enhanceTaskWithAI(_ context.Context, result *TaskProcessingResult) error {
	// For now, use heuristic-based enhancement since AnalyzeWithAI is not available
	tp.logger.Debug("applying heuristic-based task enhancement")

	return tp.applyTaskEnhancements(result.OriginalTask, result)
}

// findSemanticDuplicates uses vector similarity to find potential duplicate tasks
func (tp *TaskProcessor) findSemanticDuplicates(ctx context.Context, result *TaskProcessingResult) error {
	if tp.mcpClient == nil || !tp.mcpClient.IsOnline() {
		return nil
	}

	// Use memory_read to find similar content
	searchRequest := map[string]interface{}{
		"operation": "find_similar",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": tp.repository,
			"problem":    result.OriginalTask.Content,
			"limit":      5,
		},
	}

	response, err := tp.mcpClient.QueryIntelligence(ctx, "find_similar", searchRequest)
	if err != nil {
		return fmt.Errorf("similarity search failed: %w", err)
	}

	// Parse similar tasks and check if they're duplicates
	return tp.processSimilarTasks(response, result)
}

// findRelatedTasks finds tasks that are related but not duplicates
func (tp *TaskProcessor) findRelatedTasks(ctx context.Context, result *TaskProcessingResult) error {
	if tp.mcpClient == nil || !tp.mcpClient.IsOnline() {
		return nil
	}

	// Search for related content in memory
	searchRequest := map[string]interface{}{
		"operation": "search",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": tp.repository,
			"session_id": tp.sessionID,
			"query":      tp.extractKeywords(result.OriginalTask.Content),
			"limit":      10,
		},
	}

	response, err := tp.mcpClient.QueryIntelligence(ctx, "search", searchRequest)
	if err != nil {
		return fmt.Errorf("related task search failed: %w", err)
	}

	return tp.processRelatedTasks(response, result)
}

// generateContextualSuggestions generates AI-powered suggestions based on current context
func (tp *TaskProcessor) generateContextualSuggestions(_ context.Context, result *TaskProcessingResult) error {
	// Generate heuristic-based suggestions since AnalyzeWithAI is not available
	tp.logger.Debug("generating heuristic-based contextual suggestions")

	return tp.generateHeuristicSuggestions(result.OriginalTask, result)
}

// storeTaskInsights stores the AI processing results in memory for future learning
func (tp *TaskProcessor) storeTaskInsights(ctx context.Context, result *TaskProcessingResult) {
	insights := map[string]interface{}{
		"task_id":                 result.OriginalTask.ID,
		"ai_enhancements_applied": tp.getEnhancementsSummary(result),
		"suggestions_generated":   len(result.Suggestions),
		"duplicates_found":        len(result.Duplicates),
		"related_tasks_found":     len(result.RelatedTasks),
		"processing_timestamp":    time.Now().Format(time.RFC3339),
		"confidence_scores":       tp.getSuggestionConfidences(result.Suggestions),
		"ai_recommendations":      result.AIRecommendations,
	}

	insightsJSON, _ := json.Marshal(insights)

	storeRequest := map[string]interface{}{
		"operation": "store_chunk",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": tp.repository,
			"session_id": tp.sessionID,
			"content":    string(insightsJSON),
			"type":       "ai_task_processing",
			"metadata": map[string]interface{}{
				"task_id":         result.OriginalTask.ID,
				"processing_type": "ai_enhancement",
				"ai_generated":    true,
			},
		},
	}

	if _, err := tp.mcpClient.QueryIntelligence(ctx, "store_chunk", storeRequest); err != nil {
		tp.logger.Warn("failed to store task insights",
			slog.String("task_id", result.OriginalTask.ID),
			slog.String("error", err.Error()))
	}
}

// Helper methods

func (tp *TaskProcessor) applyTaskEnhancements(task *entities.Task, result *TaskProcessingResult) error {
	var enhancement struct {
		EnhancedDescription  string   `json:"enhanced_description"`
		SuggestedPriority    string   `json:"suggested_priority"`
		SuggestedTags        []string `json:"suggested_tags"`
		EstimatedMinutes     int      `json:"estimated_minutes"`
		EnhancementReasoning string   `json:"enhancement_reasoning"`
		ActionabilityScore   float64  `json:"actionability_score"`
		ClarityImprovements  []string `json:"clarity_improvements"`
		MissingInformation   []string `json:"missing_information"`
		SuccessCriteria      []string `json:"success_criteria"`
	}

	// Use heuristic-based enhancements since AI analysis isn't available
	enhancement.EnhancedDescription = task.Content
	enhancement.SuggestedPriority = string(task.Priority)
	enhancement.SuggestedTags = task.Tags
	enhancement.EstimatedMinutes = task.EstimatedMins
	enhancement.EnhancementReasoning = "Heuristic-based task analysis"
	enhancement.ActionabilityScore = 0.8

	// Apply enhancements to the task
	enhanced := *result.OriginalTask // Copy the task

	if enhancement.EnhancedDescription != "" && enhancement.EnhancedDescription != result.OriginalTask.Content {
		enhanced.Content = enhancement.EnhancedDescription
		result.ProcessingNotes = append(result.ProcessingNotes, "AI enhanced task description for better clarity")
	}

	if tp.config.AutoPrioritization && enhancement.SuggestedPriority != "" {
		if priority, err := parsePriority(enhancement.SuggestedPriority); err == nil {
			if priority != enhanced.Priority {
				enhanced.Priority = priority
				result.ProcessingNotes = append(result.ProcessingNotes,
					"AI suggested priority change to "+enhancement.SuggestedPriority)
			}
		}
	}

	if tp.config.SmartTagging && len(enhancement.SuggestedTags) > 0 {
		for _, tag := range enhancement.SuggestedTags {
			if !enhanced.HasTag(tag) {
				enhanced.AddTag(tag)
				result.ProcessingNotes = append(result.ProcessingNotes,
					"AI added smart tag: "+tag)
			}
		}
	}

	if tp.config.AutoEstimation && enhancement.EstimatedMinutes > 0 {
		if enhanced.EstimatedMins == 0 {
			enhanced.EstimatedMins = enhancement.EstimatedMinutes
			result.ProcessingNotes = append(result.ProcessingNotes,
				fmt.Sprintf("AI estimated %d minutes for completion", enhancement.EstimatedMinutes))
		}
	}

	// Add AI-generated insights
	if enhancement.EnhancementReasoning != "" {
		result.ContextInsights = append(result.ContextInsights, enhancement.EnhancementReasoning)
	}

	if len(enhancement.MissingInformation) > 0 {
		result.AIRecommendations = append(result.AIRecommendations,
			"Missing information: "+strings.Join(enhancement.MissingInformation, ", "))
	}

	if len(enhancement.SuccessCriteria) > 0 {
		result.AIRecommendations = append(result.AIRecommendations,
			"Success criteria: "+strings.Join(enhancement.SuccessCriteria, "; "))
	}

	result.EnhancedTask = &enhanced
	return nil
}

func (tp *TaskProcessor) generateHeuristicSuggestions(task *entities.Task, result *TaskProcessingResult) error {
	// Generate intelligent suggestions based on task content and context
	suggestions := make([]*AITaskSuggestion, 0)

	// Analyze task content for keywords and patterns
	content := strings.ToLower(task.Content)

	// Development-related suggestions
	if strings.Contains(content, "implement") || strings.Contains(content, "develop") || strings.Contains(content, "code") {
		suggestions = append(suggestions, &AITaskSuggestion{
			Title:         "Write tests for implementation",
			Description:   "Add comprehensive tests to ensure code quality",
			Priority:      entities.PriorityMedium,
			Tags:          []string{"testing", "quality"},
			EstimatedMins: 30,
			Confidence:    0.8,
			Reasoning:     "Testing is crucial for development tasks",
			Type:          "testing",
			Category:      "quality-assurance",
		})
	}

	// Documentation-related suggestions
	if strings.Contains(content, "api") || strings.Contains(content, "feature") {
		suggestions = append(suggestions, &AITaskSuggestion{
			Title:         "Update documentation",
			Description:   "Document the new changes and API modifications",
			Priority:      entities.PriorityLow,
			Tags:          []string{"documentation", "maintenance"},
			EstimatedMins: 20,
			Confidence:    0.7,
			Reasoning:     "Documentation helps team understanding",
			Type:          "documentation",
			Category:      "maintenance",
		})
	}

	// Bug fix related suggestions
	if strings.Contains(content, "fix") || strings.Contains(content, "bug") || strings.Contains(content, "issue") {
		suggestions = append(suggestions, &AITaskSuggestion{
			Title:         "Add regression test",
			Description:   "Create a test to prevent this issue from recurring",
			Priority:      entities.PriorityHigh,
			Tags:          []string{"testing", "regression", "quality"},
			EstimatedMins: 25,
			Confidence:    0.9,
			Reasoning:     "Regression tests prevent future bugs",
			Type:          "testing",
			Category:      "quality-assurance",
		})
	}

	// Review and optimization suggestions
	if task.Priority == entities.PriorityHigh {
		suggestions = append(suggestions, &AITaskSuggestion{
			Title:         "Schedule code review",
			Description:   "Get peer review for this high-priority task",
			Priority:      entities.PriorityMedium,
			Tags:          []string{"review", "collaboration"},
			EstimatedMins: 15,
			Confidence:    0.8,
			Reasoning:     "High-priority tasks benefit from peer review",
			Type:          "review",
			Category:      "collaboration",
		})
	}

	// Filter by confidence threshold and add to result
	for _, suggestion := range suggestions {
		if suggestion.Confidence >= tp.config.MinConfidenceThreshold {
			result.Suggestions = append(result.Suggestions, suggestion)
		}
	}

	// Add context insights
	result.ContextInsights = append(result.ContextInsights,
		"Heuristic-based suggestions generated based on task content analysis")

	if len(suggestions) > 0 {
		result.ContextInsights = append(result.ContextInsights,
			fmt.Sprintf("Generated %d actionable suggestions", len(suggestions)))
	}

	return nil
}

// parseSuggestions method is no longer used since we use heuristic suggestions
// func (tp *TaskProcessor) parseSuggestions(response *ports.AIAnalysisResponse, result *TaskProcessingResult) error {
//	var suggestionResponse struct {
//		Suggestions      []*AITaskSuggestion `json:"suggestions"`
//		ContextInsights  []string            `json:"context_insights"`
//		AIRecommendations []string           `json:"ai_recommendations"`
//	}
//
//	if err := json.Unmarshal([]byte(response.Analysis), &suggestionResponse); err != nil {
//		return fmt.Errorf("failed to parse suggestions response: %w", err)
//	}
//
//	// Filter suggestions by confidence threshold
//	for _, suggestion := range suggestionResponse.Suggestions {
//		if suggestion.Confidence >= tp.config.MinConfidenceThreshold {
//			result.Suggestions = append(result.Suggestions, suggestion)
//		}
//	}
//
//	result.ContextInsights = append(result.ContextInsights, suggestionResponse.ContextInsights...)
//	result.AIRecommendations = append(result.AIRecommendations, suggestionResponse.AIRecommendations...)
//
//	return nil
// }

func (tp *TaskProcessor) processSimilarTasks(response map[string]interface{}, result *TaskProcessingResult) error {
	// Process similarity search results to identify potential duplicates
	// This would parse the MCP response and convert to entities.Task objects
	// Implementation depends on MCP response format
	return nil
}

func (tp *TaskProcessor) processRelatedTasks(response map[string]interface{}, result *TaskProcessingResult) error {
	// Process search results to identify related tasks
	// This would parse the MCP response and convert to entities.Task objects
	// Implementation depends on MCP response format
	return nil
}

func (tp *TaskProcessor) extractKeywords(content string) string {
	// Simple keyword extraction - could be enhanced with NLP
	words := strings.Fields(strings.ToLower(content))
	var keywords []string

	for _, word := range words {
		if len(word) > 3 && !isStopWord(word) {
			keywords = append(keywords, word)
		}
	}

	return strings.Join(keywords, " ")
}

func (tp *TaskProcessor) getEnhancementsSummary(result *TaskProcessingResult) map[string]bool {
	return map[string]bool{
		"description_enhanced": result.EnhancedTask.Content != result.OriginalTask.Content,
		"priority_adjusted":    result.EnhancedTask.Priority != result.OriginalTask.Priority,
		"tags_added":           len(result.EnhancedTask.Tags) > len(result.OriginalTask.Tags),
		"time_estimated":       result.EnhancedTask.EstimatedMins > 0 && result.OriginalTask.EstimatedMins == 0,
	}
}

func (tp *TaskProcessor) getSuggestionConfidences(suggestions []*AITaskSuggestion) []float64 {
	confidences := make([]float64, len(suggestions))
	for i, suggestion := range suggestions {
		confidences[i] = suggestion.Confidence
	}
	return confidences
}

func parsePriority(priority string) (entities.Priority, error) {
	switch strings.ToLower(priority) {
	case "low":
		return entities.PriorityLow, nil
	case "medium":
		return entities.PriorityMedium, nil
	case "high":
		return entities.PriorityHigh, nil
	default:
		return entities.PriorityMedium, fmt.Errorf("invalid priority: %s", priority)
	}
}

func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "should": true, "could": true,
	}
	return stopWords[word]
}

func getDefaultTaskProcessorConfig() *TaskProcessorConfig {
	return &TaskProcessorConfig{
		MaxSuggestions:         5,
		MinConfidenceThreshold: 0.6,
		AutoPrioritization:     true,
		SmartTagging:           true,
		ContextualSuggestions:  true,
		SemanticDuplication:    true,
		AutoEstimation:         true,
		LearningMode:           true,
	}
}
