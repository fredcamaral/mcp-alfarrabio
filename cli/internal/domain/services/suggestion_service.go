package services

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/repositories"
)

// SuggestionService interface defines task suggestion capabilities
type SuggestionService interface {
	// Core suggestion generation
	GenerateSuggestions(ctx context.Context, repository string, maxSuggestions int) ([]*entities.TaskSuggestion, error)
	GenerateSuggestionsForContext(ctx context.Context, workContext *entities.WorkContext, maxSuggestions int) ([]*entities.TaskSuggestion, error)

	// Specific suggestion types
	GenerateNextTaskSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error)
	GeneratePatternBasedSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error)
	GenerateOptimizationSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error)
	GenerateBreakSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error)

	// Suggestion management
	RankSuggestions(suggestions []*entities.TaskSuggestion, workContext *entities.WorkContext) []*entities.TaskSuggestion
	FilterSuggestions(suggestions []*entities.TaskSuggestion, preferences *entities.UserPreferences) []*entities.TaskSuggestion
	ProcessFeedback(ctx context.Context, suggestionID string, feedback *entities.SuggestionFeedback) error

	// Batch operations
	GenerateSuggestionBatch(ctx context.Context, repository string, batchType string) (*entities.SuggestionBatch, error)
	GetPersonalizedSuggestions(ctx context.Context, repository string, preferences *entities.UserPreferences) ([]*entities.TaskSuggestion, error)
}

// SuggestionServiceConfig holds configuration for suggestion service
type SuggestionServiceConfig struct {
	MaxSuggestionsPerType     int     // Maximum suggestions per type
	MinConfidenceThreshold    float64 // Minimum confidence for suggestions
	MinRelevanceThreshold     float64 // Minimum relevance for suggestions
	PatternMatchThreshold     float64 // Threshold for pattern matching
	DefaultExpiryHours        int     // Default expiry time for suggestions
	LearningRate              float64 // Rate for adapting to feedback
	EnableAISuggestions       bool    // Whether to enable AI-powered suggestions
	EnableTemplateSuggestions bool    // Whether to enable template-based suggestions
}

// DefaultSuggestionServiceConfig returns default configuration
func DefaultSuggestionServiceConfig() *SuggestionServiceConfig {
	return &SuggestionServiceConfig{
		MaxSuggestionsPerType:     5,
		MinConfidenceThreshold:    0.6,
		MinRelevanceThreshold:     0.5,
		PatternMatchThreshold:     0.7,
		DefaultExpiryHours:        24,
		LearningRate:              0.1,
		EnableAISuggestions:       true,
		EnableTemplateSuggestions: true,
	}
}

// suggestionServiceImpl implements the SuggestionService interface
type suggestionServiceImpl struct {
	taskRepo        repositories.TaskRepository
	patternRepo     PatternRepository
	sessionRepo     SessionRepository
	contextAnalyzer ContextAnalyzer
	patternDetector PatternDetector
	analytics       AnalyticsEngine
	config          *SuggestionServiceConfig
	logger          *slog.Logger
}

// NewSuggestionService creates a new suggestion service
func NewSuggestionService(
	taskRepo repositories.TaskRepository,
	patternRepo PatternRepository,
	sessionRepo SessionRepository,
	contextAnalyzer ContextAnalyzer,
	patternDetector PatternDetector,
	analytics AnalyticsEngine,
	config *SuggestionServiceConfig,
	logger *slog.Logger,
) SuggestionService {
	if config == nil {
		config = DefaultSuggestionServiceConfig()
	}

	return &suggestionServiceImpl{
		taskRepo:        taskRepo,
		patternRepo:     patternRepo,
		sessionRepo:     sessionRepo,
		contextAnalyzer: contextAnalyzer,
		patternDetector: patternDetector,
		analytics:       analytics,
		config:          config,
		logger:          logger,
	}
}

// GenerateSuggestions generates suggestions for a repository
func (ss *suggestionServiceImpl) GenerateSuggestions(
	ctx context.Context,
	repository string,
	maxSuggestions int,
) ([]*entities.TaskSuggestion, error) {
	ss.logger.Info("generating suggestions",
		slog.String("repository", repository),
		slog.Int("max_suggestions", maxSuggestions))

	// Analyze current context
	workContext, err := ss.contextAnalyzer.AnalyzeCurrentContext(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze context: %w", err)
	}

	return ss.GenerateSuggestionsForContext(ctx, workContext, maxSuggestions)
}

// GenerateSuggestionsForContext generates suggestions based on work context
func (ss *suggestionServiceImpl) GenerateSuggestionsForContext(
	ctx context.Context,
	workContext *entities.WorkContext,
	maxSuggestions int,
) ([]*entities.TaskSuggestion, error) {
	ss.logger.Info("generating suggestions for context",
		slog.String("repository", workContext.Repository),
		slog.Int("current_tasks", len(workContext.CurrentTasks)),
		slog.Float64("focus_level", workContext.FocusLevel),
		slog.Float64("energy_level", workContext.EnergyLevel))

	var allSuggestions []*entities.TaskSuggestion

	// Generate suggestions from different sources
	sources := []struct {
		name string
		fn   func(context.Context, *entities.WorkContext) ([]*entities.TaskSuggestion, error)
	}{
		{"next_task", ss.GenerateNextTaskSuggestions},
		{"pattern_based", ss.GeneratePatternBasedSuggestions},
		{"optimization", ss.GenerateOptimizationSuggestions},
		{"break", ss.GenerateBreakSuggestions},
	}

	for _, source := range sources {
		suggestions, err := source.fn(ctx, workContext)
		if err != nil {
			ss.logger.Warn("failed to generate suggestions from source",
				slog.String("source", source.name),
				slog.Any("error", err))
			continue
		}

		// Limit suggestions per source
		maxPerSource := ss.config.MaxSuggestionsPerType
		if len(suggestions) > maxPerSource {
			suggestions = suggestions[:maxPerSource]
		}

		allSuggestions = append(allSuggestions, suggestions...)
	}

	// Additional sources if enabled
	if ss.config.EnableTemplateSuggestions {
		templateSuggestions := ss.generateTemplateSuggestions(workContext)
		allSuggestions = append(allSuggestions, templateSuggestions...)
	}

	if ss.config.EnableAISuggestions {
		aiSuggestions := ss.generateAISuggestions(ctx, workContext)
		allSuggestions = append(allSuggestions, aiSuggestions...)
	}

	// Rank and filter suggestions
	rankedSuggestions := ss.RankSuggestions(allSuggestions, workContext)

	// Apply filters based on confidence and relevance
	filteredSuggestions := ss.filterByThresholds(rankedSuggestions)

	// Limit to max suggestions
	if len(filteredSuggestions) > maxSuggestions {
		filteredSuggestions = filteredSuggestions[:maxSuggestions]
	}

	ss.logger.Info("suggestion generation completed",
		slog.Int("total_generated", len(allSuggestions)),
		slog.Int("after_ranking", len(rankedSuggestions)),
		slog.Int("final_count", len(filteredSuggestions)))

	return filteredSuggestions, nil
}

// GenerateNextTaskSuggestions generates suggestions for next tasks to work on
func (ss *suggestionServiceImpl) GenerateNextTaskSuggestions(
	ctx context.Context,
	workContext *entities.WorkContext,
) ([]*entities.TaskSuggestion, error) {
	var suggestions []*entities.TaskSuggestion

	// Get pending tasks
	endTime := time.Now().AddDate(0, 0, 7)    // Next week
	startTime := time.Now().AddDate(0, 0, -1) // Yesterday

	allTasks, err := ss.taskRepo.FindByTimeRange(ctx, workContext.Repository, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Filter to pending tasks
	var pendingTasks []*entities.Task
	for _, task := range allTasks {
		if task.Status == "pending" || task.Status == "in_progress" {
			pendingTasks = append(pendingTasks, task)
		}
	}

	// Score tasks based on context
	type taskScore struct {
		task  *entities.Task
		score float64
	}

	var scoredTasks []taskScore
	for _, task := range pendingTasks {
		score := ss.calculateTaskScore(task, workContext)
		if score > 0.3 { // Minimum threshold
			scoredTasks = append(scoredTasks, taskScore{task: task, score: score})
		}
	}

	// Sort by score
	sort.Slice(scoredTasks, func(i, j int) bool {
		return scoredTasks[i].score > scoredTasks[j].score
	})

	// Create suggestions for top tasks
	for i, ts := range scoredTasks {
		if i >= ss.config.MaxSuggestionsPerType {
			break
		}

		suggestion := entities.NewTaskSuggestion(
			entities.SuggestionTypeNext,
			fmt.Sprintf("Work on: %s", ts.task.Content),
			entities.SuggestionSource{
				Type:       "analytics",
				Name:       "next_task_scorer",
				Algorithm:  "context_based_scoring",
				Confidence: ts.score,
			},
			workContext.Repository,
		)

		suggestion.Confidence = ts.score
		suggestion.Relevance = ss.calculateRelevance(ts.task, workContext)
		suggestion.Urgency = ss.calculateUrgency(ts.task)
		suggestion.Priority = string(ts.task.Priority)
		suggestion.TaskType = ts.task.Type
		suggestion.RelatedTaskIDs = []string{ts.task.ID}

		// Add reasoning
		suggestion.Reasoning = ss.generateTaskReasoning(ts.task, workContext, ts.score)

		// Add actions
		suggestion.AddAction("open_task", "Open and start working on this task", 1)

		// Set estimated time
		estimatedTime := ss.estimateTaskDuration(ts.task, workContext)
		suggestion.SetEstimatedTime(estimatedTime, "historical data and context analysis")

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// GeneratePatternBasedSuggestions generates suggestions based on detected patterns
func (ss *suggestionServiceImpl) GeneratePatternBasedSuggestions(
	ctx context.Context,
	workContext *entities.WorkContext,
) ([]*entities.TaskSuggestion, error) {
	var suggestions []*entities.TaskSuggestion

	// Get pattern suggestions from pattern detector
	patternSuggestions, err := ss.patternDetector.GetPatternSuggestions(ctx, workContext.CurrentTasks)
	if err != nil {
		return nil, fmt.Errorf("failed to get pattern suggestions: %w", err)
	}

	for _, pattern := range patternSuggestions {
		// Generate suggestion based on pattern
		suggestion := ss.createPatternSuggestion(pattern, workContext)
		if suggestion != nil {
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions, nil
}

// GenerateOptimizationSuggestions generates suggestions for workflow optimization
func (ss *suggestionServiceImpl) GenerateOptimizationSuggestions(
	ctx context.Context,
	workContext *entities.WorkContext,
) ([]*entities.TaskSuggestion, error) {
	var suggestions []*entities.TaskSuggestion

	// Analyze current productivity and suggest optimizations
	productivityScore := workContext.ProductivityScore
	focusLevel := workContext.FocusLevel
	energyLevel := workContext.EnergyLevel

	// Low productivity suggestions
	if productivityScore < 0.6 {
		suggestion := entities.NewTaskSuggestion(
			entities.SuggestionTypeOptimize,
			"Review and optimize your current workflow",
			entities.SuggestionSource{
				Type:       "analytics",
				Name:       "productivity_optimizer",
				Algorithm:  "threshold_analysis",
				Confidence: 0.8,
			},
			workContext.Repository,
		)

		suggestion.Confidence = 0.8
		suggestion.Relevance = 1.0 - productivityScore // Higher relevance for lower productivity
		suggestion.Urgency = 0.7
		suggestion.Reasoning = fmt.Sprintf("Your current productivity score is %.1f%%. Consider breaking down large tasks or eliminating distractions.", productivityScore*100)

		suggestion.AddAction("review_tasks", "Review current task list and priorities", 1)
		suggestion.AddAction("eliminate_distractions", "Identify and eliminate distractions", 2)

		suggestions = append(suggestions, suggestion)
	}

	// Low focus suggestions
	if focusLevel < 0.5 {
		suggestion := entities.NewTaskSuggestion(
			entities.SuggestionTypeOptimize,
			"Take steps to improve focus and concentration",
			entities.SuggestionSource{
				Type:       "analytics",
				Name:       "focus_optimizer",
				Algorithm:  "focus_analysis",
				Confidence: 0.7,
			},
			workContext.Repository,
		)

		suggestion.Confidence = 0.7
		suggestion.Relevance = 1.0 - focusLevel
		suggestion.Urgency = 0.6
		suggestion.Reasoning = fmt.Sprintf("Your focus level is %.1f%%. Consider using focus techniques or changing your environment.", focusLevel*100)

		suggestion.AddAction("pomodoro", "Try the Pomodoro Technique for focused work", 1)
		suggestion.AddAction("change_environment", "Change your work environment", 2)

		suggestions = append(suggestions, suggestion)
	}

	// Low energy suggestions
	if energyLevel < 0.4 {
		suggestion := entities.NewTaskSuggestion(
			entities.SuggestionTypeOptimize,
			"Boost your energy levels for better performance",
			entities.SuggestionSource{
				Type:       "analytics",
				Name:       "energy_optimizer",
				Algorithm:  "energy_analysis",
				Confidence: 0.75,
			},
			workContext.Repository,
		)

		suggestion.Confidence = 0.75
		suggestion.Relevance = 1.0 - energyLevel
		suggestion.Urgency = 0.8
		suggestion.Reasoning = fmt.Sprintf("Your energy level is %.1f%%. Consider taking a break, having a healthy snack, or doing light exercise.", energyLevel*100)

		suggestion.AddAction("take_break", "Take a 10-15 minute break", 1)
		suggestion.AddAction("hydrate", "Drink water and have a healthy snack", 2)

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// GenerateBreakSuggestions generates suggestions for breaks based on context
func (ss *suggestionServiceImpl) GenerateBreakSuggestions(
	ctx context.Context,
	workContext *entities.WorkContext,
) ([]*entities.TaskSuggestion, error) {
	var suggestions []*entities.TaskSuggestion

	// Check if a break is needed based on various factors
	needsBreak := ss.assessBreakNeed(workContext)
	if !needsBreak {
		return suggestions, nil
	}

	breakType := ss.determineBreakType(workContext)

	suggestion := entities.NewTaskSuggestion(
		entities.SuggestionTypeBreak,
		fmt.Sprintf("Take a %s break to recharge", breakType),
		entities.SuggestionSource{
			Type:       "analytics",
			Name:       "break_advisor",
			Algorithm:  "break_need_analysis",
			Confidence: 0.85,
		},
		workContext.Repository,
	)

	suggestion.Confidence = 0.85
	suggestion.Relevance = ss.calculateBreakRelevance(workContext)
	suggestion.Urgency = 0.9
	suggestion.Reasoning = ss.generateBreakReasoning(workContext, breakType)

	// Add break-specific actions
	switch breakType {
	case "short":
		suggestion.AddAction("stretch", "Do some light stretching", 1)
		suggestion.AddAction("walk", "Take a short walk", 2)
		suggestion.SetEstimatedTime(10*time.Minute, "standard short break duration")
	case "medium":
		suggestion.AddAction("walk_outside", "Take a walk outside", 1)
		suggestion.AddAction("mindfulness", "Practice mindfulness or meditation", 2)
		suggestion.SetEstimatedTime(20*time.Minute, "standard medium break duration")
	case "long":
		suggestion.AddAction("lunch_break", "Have a proper lunch break", 1)
		suggestion.AddAction("exercise", "Do some exercise", 2)
		suggestion.SetEstimatedTime(45*time.Minute, "standard long break duration")
	}

	suggestions = append(suggestions, suggestion)
	return suggestions, nil
}

// RankSuggestions ranks suggestions based on context and scoring
func (ss *suggestionServiceImpl) RankSuggestions(
	suggestions []*entities.TaskSuggestion,
	workContext *entities.WorkContext,
) []*entities.TaskSuggestion {
	if len(suggestions) == 0 {
		return suggestions
	}

	// Calculate composite scores for ranking
	for _, suggestion := range suggestions {
		suggestion.Relevance = ss.calculateContextualRelevance(suggestion, workContext)
		suggestion.Urgency = ss.calculateContextualUrgency(suggestion, workContext)
	}

	// Sort by composite score (calculated in TaskSuggestion.CalculateScore())
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].CalculateScore() > suggestions[j].CalculateScore()
	})

	return suggestions
}

// FilterSuggestions filters suggestions based on user preferences
func (ss *suggestionServiceImpl) FilterSuggestions(
	suggestions []*entities.TaskSuggestion,
	preferences *entities.UserPreferences,
) []*entities.TaskSuggestion {
	if preferences == nil {
		return suggestions
	}

	var filtered []*entities.TaskSuggestion

	for _, suggestion := range suggestions {
		// Check if suggestion type is preferred
		if len(preferences.PreferredSuggestionTypes) > 0 {
			typePreferred := false
			for _, prefType := range preferences.PreferredSuggestionTypes {
				if suggestion.Type == prefType {
					typePreferred = true
					break
				}
			}
			if !typePreferred {
				continue
			}
		}

		// Check confidence threshold
		if suggestion.Confidence < preferences.MinConfidence {
			continue
		}

		// Check avoidance patterns
		shouldAvoid := false
		for _, pattern := range preferences.AvoidancePatterns {
			if strings.Contains(strings.ToLower(suggestion.Content), strings.ToLower(pattern)) {
				shouldAvoid = true
				break
			}
		}
		if shouldAvoid {
			continue
		}

		filtered = append(filtered, suggestion)

		// Limit to max suggestions
		if len(filtered) >= preferences.MaxSuggestions {
			break
		}
	}

	return filtered
}

// ProcessFeedback processes user feedback on suggestions
func (ss *suggestionServiceImpl) ProcessFeedback(
	ctx context.Context,
	suggestionID string,
	feedback *entities.SuggestionFeedback,
) error {
	ss.logger.Info("processing suggestion feedback",
		slog.String("suggestion_id", suggestionID),
		slog.Bool("accepted", feedback.Accepted),
		slog.Bool("helpful", feedback.Helpful))

	// In a full implementation, you would:
	// 1. Store the feedback in a repository
	// 2. Update suggestion algorithms based on feedback
	// 3. Adjust user preferences and learning parameters
	// 4. Update pattern weights and confidence scores

	// For now, just log the feedback
	ss.logger.Debug("feedback details",
		slog.Int("rating", feedback.Rating),
		slog.String("reason", feedback.Reason),
		slog.String("comment", feedback.Comment))

	return nil
}

// GenerateSuggestionBatch generates a batch of related suggestions
func (ss *suggestionServiceImpl) GenerateSuggestionBatch(
	ctx context.Context,
	repository string,
	batchType string,
) (*entities.SuggestionBatch, error) {
	// Analyze context
	workContext, err := ss.contextAnalyzer.AnalyzeCurrentContext(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze context: %w", err)
	}

	var suggestions []*entities.TaskSuggestion
	var title, description string

	switch batchType {
	case "morning_startup":
		title = "Morning Startup Batch"
		description = "Suggested tasks to start your productive morning"
		suggestions, err = ss.generateMorningStartupBatch(ctx, workContext)
	case "focused_work":
		title = "Focused Work Session"
		description = "Deep work tasks suited for your current focus level"
		suggestions, err = ss.generateFocusedWorkBatch(ctx, workContext)
	case "quick_wins":
		title = "Quick Wins"
		description = "Small tasks you can complete quickly for momentum"
		suggestions, err = ss.generateQuickWinsBatch(ctx, workContext)
	default:
		return nil, fmt.Errorf("unknown batch type: %s", batchType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate batch: %w", err)
	}

	batch := entities.NewSuggestionBatch(batchType, title, suggestions)
	batch.Description = description
	batch.Context = workContext

	return batch, nil
}

// GetPersonalizedSuggestions gets suggestions personalized for user preferences
func (ss *suggestionServiceImpl) GetPersonalizedSuggestions(
	ctx context.Context,
	repository string,
	preferences *entities.UserPreferences,
) ([]*entities.TaskSuggestion, error) {
	// Generate base suggestions
	suggestions, err := ss.GenerateSuggestions(ctx, repository, preferences.MaxSuggestions*2) // Generate more to filter
	if err != nil {
		return nil, err
	}

	// Apply personalization filters
	personalized := ss.FilterSuggestions(suggestions, preferences)

	return personalized, nil
}

// Helper methods

func (ss *suggestionServiceImpl) calculateTaskScore(task *entities.Task, workContext *entities.WorkContext) float64 {
	var score float64

	// Priority scoring (30%)
	priorityScore := ss.getPriorityScore(string(task.Priority))
	score += priorityScore * 0.3

	// Due date urgency (25%)
	urgencyScore := ss.getUrgencyScore(task)
	score += urgencyScore * 0.25

	// Context fit (25%)
	contextScore := ss.getContextFitScore(task, workContext)
	score += contextScore * 0.25

	// Pattern match (20%)
	patternScore := ss.getPatternMatchScore(task, workContext)
	score += patternScore * 0.2

	return score
}

func (ss *suggestionServiceImpl) getPriorityScore(priority string) float64 {
	switch strings.ToLower(priority) {
	case "critical", "urgent":
		return 1.0
	case "high":
		return 0.8
	case "medium", "normal":
		return 0.6
	case "low":
		return 0.4
	default:
		return 0.5
	}
}

func (ss *suggestionServiceImpl) getUrgencyScore(task *entities.Task) float64 {
	if task.DueDate == nil {
		return 0.5 // No due date
	}

	timeUntilDue := time.Until(*task.DueDate)
	if timeUntilDue < 0 {
		return 1.0 // Overdue
	}
	if timeUntilDue < 24*time.Hour {
		return 0.9 // Due soon
	}
	if timeUntilDue < 3*24*time.Hour {
		return 0.7 // Due within 3 days
	}

	return 0.5 // Default urgency score
}

func (ss *suggestionServiceImpl) getContextFitScore(task *entities.Task, workContext *entities.WorkContext) float64 {
	// Predict optimal task type for current context
	optimalType := ss.contextAnalyzer.PredictOptimalTaskType(workContext)

	if task.Type == optimalType {
		return 1.0
	}

	// Partial scoring for related types
	relatedScore := ss.getTypeRelationScore(task.Type, optimalType)
	return relatedScore
}

func (ss *suggestionServiceImpl) getTypeRelationScore(taskType, optimalType string) float64 {
	// Simple heuristic for type relationships
	relations := map[string]map[string]float64{
		"creative":       {"planning": 0.7, "research": 0.6, "learning": 0.8},
		"analytical":     {"planning": 0.8, "review": 0.7, "testing": 0.9},
		"routine":        {"organizing": 0.9, "documentation": 0.7, "communication": 0.6},
		"implementation": {"coding": 1.0, "building": 0.8, "testing": 0.7},
	}

	if related, exists := relations[optimalType]; exists {
		if score, exists := related[taskType]; exists {
			return score
		}
	}

	return 0.5 // Default neutral score
}

func (ss *suggestionServiceImpl) getPatternMatchScore(task *entities.Task, workContext *entities.WorkContext) float64 {
	// Check if task matches any active patterns
	for _, pattern := range workContext.ActivePatterns {
		score := ss.patternDetector.CalculatePatternScore(pattern, []*entities.Task{task})
		if score > ss.config.PatternMatchThreshold {
			return score
		}
	}
	return 0.3 // No strong pattern match
}

func (ss *suggestionServiceImpl) calculateRelevance(task *entities.Task, workContext *entities.WorkContext) float64 {
	// Calculate how relevant this task is to current context
	var relevanceFactors []float64

	// Project relevance
	if task.Repository == workContext.Repository {
		relevanceFactors = append(relevanceFactors, 1.0)
	}

	// Type relevance
	activeTypes := workContext.GetActiveTaskTypes()
	for _, activeType := range activeTypes {
		if task.Type == activeType {
			relevanceFactors = append(relevanceFactors, 0.8)
			break
		}
	}

	// Pattern relevance
	patternScore := ss.getPatternMatchScore(task, workContext)
	relevanceFactors = append(relevanceFactors, patternScore)

	// Calculate average relevance
	if len(relevanceFactors) == 0 {
		return 0.5
	}

	total := 0.0
	for _, factor := range relevanceFactors {
		total += factor
	}
	return total / float64(len(relevanceFactors))
}

func (ss *suggestionServiceImpl) calculateUrgency(task *entities.Task) float64 {
	return ss.getUrgencyScore(task)
}

func (ss *suggestionServiceImpl) generateTaskReasoning(task *entities.Task, workContext *entities.WorkContext, score float64) string {
	reasons := []string{}

	if score > 0.8 {
		reasons = append(reasons, "High priority and excellent fit for current context")
	} else if score > 0.6 {
		reasons = append(reasons, "Good fit for current context and priorities")
	}

	if task.DueDate != nil {
		timeUntilDue := time.Until(*task.DueDate)
		if timeUntilDue < 24*time.Hour {
			reasons = append(reasons, "Due soon - high urgency")
		}
	}

	optimalType := ss.contextAnalyzer.PredictOptimalTaskType(workContext)
	if task.Type == optimalType {
		reasons = append(reasons, fmt.Sprintf("Matches optimal task type for current context (%s)", optimalType))
	}

	if len(reasons) == 0 {
		return "Suggested based on current priorities and context"
	}

	return strings.Join(reasons, ". ")
}

func (ss *suggestionServiceImpl) estimateTaskDuration(task *entities.Task, workContext *entities.WorkContext) time.Duration {
	// Simple estimation based on task type and content length
	baseTime := time.Hour

	// Adjust based on task type
	switch strings.ToLower(task.Type) {
	case "quick", "simple", "routine":
		baseTime = 30 * time.Minute
	case "complex", "research", "learning":
		baseTime = 2 * time.Hour
	case "creative", "planning":
		baseTime = 90 * time.Minute
	}

	// Adjust based on content length
	contentLength := len(task.Content)
	if contentLength > 200 {
		baseTime = time.Duration(float64(baseTime) * 1.5)
	} else if contentLength < 50 {
		baseTime = time.Duration(float64(baseTime) * 0.7)
	}

	// Adjust based on current productivity
	productivityMultiplier := 1.0
	if workContext.ProductivityScore > 0 {
		productivityMultiplier = 1.0 / workContext.ProductivityScore
	}

	estimatedTime := time.Duration(float64(baseTime) * productivityMultiplier)

	// Cap estimates
	if estimatedTime < 15*time.Minute {
		estimatedTime = 15 * time.Minute
	} else if estimatedTime > 4*time.Hour {
		estimatedTime = 4 * time.Hour
	}

	return estimatedTime
}

func (ss *suggestionServiceImpl) createPatternSuggestion(pattern *entities.TaskPattern, workContext *entities.WorkContext) *entities.TaskSuggestion {
	if len(pattern.Sequence) == 0 {
		return nil
	}

	// Create suggestion based on next step in pattern
	// TODO: Implement GetNextStep logic or use pattern.Sequence
	if len(pattern.Sequence) == 0 {
		return nil
	}
	nextStep := &pattern.Sequence[0] // Use first step for now

	suggestion := entities.NewTaskSuggestion(
		entities.SuggestionTypePattern,
		fmt.Sprintf("Continue pattern: %s", nextStep.TaskType),
		entities.SuggestionSource{
			Type:       "pattern",
			Name:       pattern.Name,
			Algorithm:  "pattern_matching",
			Confidence: pattern.Confidence,
		},
		workContext.Repository,
	)

	suggestion.Confidence = pattern.Confidence
	suggestion.Relevance = 0.8 // Patterns are generally relevant
	suggestion.Urgency = 0.6
	suggestion.PatternID = pattern.ID
	suggestion.TaskType = nextStep.TaskType
	suggestion.Priority = nextStep.Priority

	suggestion.Reasoning = fmt.Sprintf("Based on pattern '%s' with %.1f%% confidence. This pattern has been successful %d times.",
		pattern.Name, pattern.Confidence*100, pattern.Occurrences)

	suggestion.AddAction("follow_pattern", "Follow the established pattern", 1)

	return suggestion
}

func (ss *suggestionServiceImpl) assessBreakNeed(workContext *entities.WorkContext) bool {
	// Multiple factors for break assessment
	factors := []bool{
		workContext.EnergyLevel < 0.4,         // Low energy
		workContext.FocusLevel < 0.3,          // Very low focus
		workContext.IsHighStress(),            // High stress
		len(workContext.StressIndicators) > 2, // Multiple stress indicators
	}

	// Need break if any factor is true
	for _, factor := range factors {
		if factor {
			return true
		}
	}

	// Check session duration if available
	if workContext.CurrentSession != nil {
		sessionHours := workContext.CurrentSession.Duration.Hours()
		if sessionHours > 2 { // Working for more than 2 hours
			return true
		}
	}

	return false
}

func (ss *suggestionServiceImpl) determineBreakType(workContext *entities.WorkContext) string {
	if workContext.EnergyLevel < 0.3 || workContext.IsHighStress() {
		return "long"
	} else if workContext.FocusLevel < 0.4 {
		return "medium"
	} else {
		return "short"
	}
}

func (ss *suggestionServiceImpl) calculateBreakRelevance(workContext *entities.WorkContext) float64 {
	relevance := 0.5 // Base relevance

	if workContext.EnergyLevel < 0.4 {
		relevance += 0.3
	}
	if workContext.FocusLevel < 0.4 {
		relevance += 0.2
	}
	if workContext.IsHighStress() {
		relevance += 0.3
	}

	return math.Min(relevance, 1.0)
}

func (ss *suggestionServiceImpl) generateBreakReasoning(workContext *entities.WorkContext, breakType string) string {
	reasons := []string{}

	if workContext.EnergyLevel < 0.4 {
		reasons = append(reasons, fmt.Sprintf("Energy level is low (%.1f%%)", workContext.EnergyLevel*100))
	}

	if workContext.FocusLevel < 0.4 {
		reasons = append(reasons, fmt.Sprintf("Focus level is low (%.1f%%)", workContext.FocusLevel*100))
	}

	if workContext.IsHighStress() {
		reasons = append(reasons, "High stress indicators detected")
	}

	if workContext.CurrentSession != nil && workContext.CurrentSession.Duration.Hours() > 2 {
		reasons = append(reasons, fmt.Sprintf("Working for %.1f hours without a break", workContext.CurrentSession.Duration.Hours()))
	}

	reasoning := strings.Join(reasons, ". ")
	if reasoning != "" {
		reasoning += fmt.Sprintf(". A %s break will help restore your energy and focus.", breakType)
	} else {
		reasoning = fmt.Sprintf("A %s break will help maintain productivity and well-being.", breakType)
	}

	return reasoning
}

func (ss *suggestionServiceImpl) filterByThresholds(suggestions []*entities.TaskSuggestion) []*entities.TaskSuggestion {
	var filtered []*entities.TaskSuggestion

	for _, suggestion := range suggestions {
		if suggestion.Confidence >= ss.config.MinConfidenceThreshold &&
			suggestion.Relevance >= ss.config.MinRelevanceThreshold {
			filtered = append(filtered, suggestion)
		}
	}

	return filtered
}

func (ss *suggestionServiceImpl) calculateContextualRelevance(suggestion *entities.TaskSuggestion, workContext *entities.WorkContext) float64 {
	// Adjust relevance based on current context
	baseRelevance := suggestion.Relevance

	// Time-based adjustments
	timeOptimal := ss.contextAnalyzer.PredictOptimalTaskType(workContext)
	if suggestion.TaskType == timeOptimal {
		baseRelevance += 0.1
	}

	// Energy-based adjustments
	if workContext.EnergyLevel < 0.4 && suggestion.Type == entities.SuggestionTypeBreak {
		baseRelevance += 0.2
	}

	// Focus-based adjustments
	if workContext.FocusLevel > 0.8 && suggestion.Type == entities.SuggestionTypeNext {
		baseRelevance += 0.1
	}

	return math.Min(baseRelevance, 1.0)
}

func (ss *suggestionServiceImpl) calculateContextualUrgency(suggestion *entities.TaskSuggestion, workContext *entities.WorkContext) float64 {
	baseUrgency := suggestion.Urgency

	// Stress-based urgency adjustments
	if workContext.IsHighStress() && suggestion.Type == entities.SuggestionTypeBreak {
		baseUrgency += 0.2
	}

	// Productivity-based adjustments
	if workContext.ProductivityScore < 0.5 && suggestion.Type == entities.SuggestionTypeOptimize {
		baseUrgency += 0.15
	}

	return math.Min(baseUrgency, 1.0)
}

// Placeholder implementations for additional suggestion types

func (ss *suggestionServiceImpl) generateTemplateSuggestions(workContext *entities.WorkContext) []*entities.TaskSuggestion {
	// Placeholder for template-based suggestions
	return []*entities.TaskSuggestion{}
}

func (ss *suggestionServiceImpl) generateAISuggestions(ctx context.Context, workContext *entities.WorkContext) []*entities.TaskSuggestion {
	// Placeholder for AI-powered suggestions
	return []*entities.TaskSuggestion{}
}

func (ss *suggestionServiceImpl) generateMorningStartupBatch(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	// Placeholder for morning startup batch
	return []*entities.TaskSuggestion{}, nil
}

func (ss *suggestionServiceImpl) generateFocusedWorkBatch(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	// Placeholder for focused work batch
	return []*entities.TaskSuggestion{}, nil
}

func (ss *suggestionServiceImpl) generateQuickWinsBatch(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error) {
	// Placeholder for quick wins batch
	return []*entities.TaskSuggestion{}, nil
}
