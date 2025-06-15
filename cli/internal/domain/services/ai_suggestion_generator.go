package services

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
)

// AISuggestionGenerator interface defines AI-powered suggestion capabilities
type AISuggestionGenerator interface {
	// Core AI suggestion generation
	GenerateContextualSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error)
	GenerateCreativeSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error)
	GenerateLearningPathSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error)

	// Specialized AI suggestions
	GenerateProductivityOptimizations(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error)
	GenerateWorkflowImprovements(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error)
	GenerateGoalAlignedSuggestions(ctx context.Context, workContext *entities.WorkContext) ([]*entities.TaskSuggestion, error)

	// Content analysis and generation
	AnalyzeTaskComplexity(task *entities.Task) (*TaskComplexityAnalysis, error)
	GenerateTaskBreakdown(task *entities.Task) ([]*entities.TaskSuggestion, error)
	SuggestTaskOptimizations(task *entities.Task, historical []*entities.Task) ([]*entities.TaskSuggestion, error)
}

// TaskComplexityAnalysis represents AI analysis of task complexity
type TaskComplexityAnalysis struct {
	ComplexityScore    float64       `json:"complexity_score"`    // 0-1 complexity rating
	FactorsIdentified  []string      `json:"factors_identified"`  // Complexity factors found
	EstimatedDuration  time.Duration `json:"estimated_duration"`  // AI-estimated duration
	SkillsRequired     []string      `json:"skills_required"`     // Required skills
	Dependencies       []string      `json:"dependencies"`        // Potential dependencies
	RiskFactors        []string      `json:"risk_factors"`        // Risk factors identified
	Recommendations    []string      `json:"recommendations"`     // AI recommendations
	BreakdownSuggested bool          `json:"breakdown_suggested"` // Whether to break down
	Confidence         float64       `json:"confidence"`          // AI confidence in analysis
}

// AISuggestionConfig holds configuration for AI suggestion generation
type AISuggestionConfig struct {
	MaxSuggestionsPerType     int     // Maximum suggestions per AI type
	MinConfidenceThreshold    float64 // Minimum confidence for AI suggestions
	EnableCreativeSuggestions bool    // Enable creative/innovative suggestions
	EnableLearningPath        bool    // Enable learning path suggestions
	ContextWindowSize         int     // Number of recent tasks to consider
	CreativityWeight          float64 // Weight for creative vs practical suggestions
	LearningWeight            float64 // Weight for learning-oriented suggestions
	PersonalizationWeight     float64 // Weight for personalization factors
}

// DefaultAISuggestionConfig returns default configuration
func DefaultAISuggestionConfig() *AISuggestionConfig {
	return &AISuggestionConfig{
		MaxSuggestionsPerType:     3,
		MinConfidenceThreshold:    0.7,
		EnableCreativeSuggestions: true,
		EnableLearningPath:        true,
		ContextWindowSize:         20,
		CreativityWeight:          0.3,
		LearningWeight:            0.4,
		PersonalizationWeight:     0.7,
	}
}

// aiSuggestionGeneratorImpl implements the AISuggestionGenerator interface
type aiSuggestionGeneratorImpl struct {
	contextAnalyzer ContextAnalyzer
	patternDetector PatternDetector
	analytics       AnalyticsEngine
	config          *AISuggestionConfig
	logger          *slog.Logger
}

// NewAISuggestionGenerator creates a new AI suggestion generator
func NewAISuggestionGenerator(
	contextAnalyzer ContextAnalyzer,
	patternDetector PatternDetector,
	analytics AnalyticsEngine,
	config *AISuggestionConfig,
	logger *slog.Logger,
) AISuggestionGenerator {
	if config == nil {
		config = DefaultAISuggestionConfig()
	}

	return &aiSuggestionGeneratorImpl{
		contextAnalyzer: contextAnalyzer,
		patternDetector: patternDetector,
		analytics:       analytics,
		config:          config,
		logger:          logger,
	}
}

// GenerateContextualSuggestions generates AI suggestions based on current context
func (ai *aiSuggestionGeneratorImpl) GenerateContextualSuggestions(
	ctx context.Context,
	workContext *entities.WorkContext,
) ([]*entities.TaskSuggestion, error) {
	ai.logger.Info("generating contextual AI suggestions",
		slog.String("repository", workContext.Repository),
		slog.Float64("focus_level", workContext.FocusLevel),
		slog.Float64("energy_level", workContext.EnergyLevel))

	var suggestions []*entities.TaskSuggestion

	// Analyze current context deeply
	contextInsights := ai.analyzeContextInsights(workContext)

	// Generate suggestions based on context analysis
	for _, insight := range contextInsights {
		suggestion := ai.createInsightBasedSuggestion(insight, workContext)
		if suggestion != nil {
			suggestions = append(suggestions, suggestion)
		}
	}

	// Generate adaptive suggestions based on patterns
	adaptiveSuggestions := ai.generateAdaptiveSuggestions(workContext)
	suggestions = append(suggestions, adaptiveSuggestions...)

	// Limit suggestions
	if len(suggestions) > ai.config.MaxSuggestionsPerType {
		suggestions = suggestions[:ai.config.MaxSuggestionsPerType]
	}

	ai.logger.Info("contextual AI suggestions generated",
		slog.Int("count", len(suggestions)))

	return suggestions, nil
}

// GenerateCreativeSuggestions generates innovative and creative task suggestions
func (ai *aiSuggestionGeneratorImpl) GenerateCreativeSuggestions(
	ctx context.Context,
	workContext *entities.WorkContext,
) ([]*entities.TaskSuggestion, error) {
	if !ai.config.EnableCreativeSuggestions {
		return nil, nil
	}

	ai.logger.Info("generating creative AI suggestions")

	// Analyze potential for creative work
	creativeOpportunities := ai.identifyCreativeOpportunities(workContext)
	suggestions := make([]*entities.TaskSuggestion, 0, len(creativeOpportunities))

	for _, opportunity := range creativeOpportunities {
		suggestion := ai.createBasicTaskSuggestion(
			entities.SuggestionTypeLearning,
			opportunity.suggestion,
			"creative_ai",
			"creative_opportunity_analysis",
			opportunity.confidence,
			workContext.Repository,
		)
		suggestion.Relevance = opportunity.relevance
		suggestion.Urgency = 0.4 // Creative tasks are generally lower urgency
		suggestion.Reasoning = opportunity.reasoning

		suggestion.AddAction("explore", "Explore this creative opportunity", 1)
		suggestion.AddAction("prototype", "Create a quick prototype or proof of concept", 2)

		suggestion.SetEstimatedTime(opportunity.estimatedTime, "AI analysis of creative tasks")
		suggestion.AddKeywords(opportunity.keywords...)

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// GenerateLearningPathSuggestions generates learning-oriented suggestions
func (ai *aiSuggestionGeneratorImpl) GenerateLearningPathSuggestions(
	ctx context.Context,
	workContext *entities.WorkContext,
) ([]*entities.TaskSuggestion, error) {
	if !ai.config.EnableLearningPath {
		return nil, nil
	}

	ai.logger.Info("generating learning path AI suggestions")

	// Identify knowledge gaps and learning opportunities
	learningGaps := ai.identifyLearningGaps(workContext)
	suggestions := make([]*entities.TaskSuggestion, 0, len(learningGaps))

	for _, gap := range learningGaps {
		suggestion := ai.createBasicTaskSuggestion(
			entities.SuggestionTypeLearning,
			"Learn: "+gap.topic,
			"learning_path_ai",
			"knowledge_gap_analysis",
			gap.confidence,
			workContext.Repository,
		)
		suggestion.Relevance = gap.relevance
		suggestion.Urgency = gap.urgency
		suggestion.Reasoning = gap.reasoning

		// Add learning-specific actions
		suggestion.AddAction("research", "Research the topic online", 1)
		suggestion.AddAction("practice", "Find hands-on practice opportunities", 2)
		suggestion.AddAction("apply", "Apply learning to current projects", 3)

		suggestion.SetEstimatedTime(gap.estimatedTime, "AI analysis of learning requirements")
		suggestion.AddKeywords(gap.keywords...)

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// GenerateProductivityOptimizations generates AI-driven productivity suggestions
func (ai *aiSuggestionGeneratorImpl) GenerateProductivityOptimizations(
	ctx context.Context,
	workContext *entities.WorkContext,
) ([]*entities.TaskSuggestion, error) {
	bottlenecks := ai.analyzeProductivityBottlenecks(workContext)
	return generateSuggestionsFromAnalysis(
		ai,
		"generating productivity optimization suggestions",
		bottlenecks,
		workContext,
		func(b productivityBottleneck) suggestionConfig {
			return suggestionConfig{
				SuggestionType: entities.SuggestionTypeOptimize,
				Title:          "Optimize: " + b.area,
				SourceName:     "productivity_optimizer_ai",
				Algorithm:      "bottleneck_analysis",
				Confidence:     b.confidence,
				Relevance:      b.impact,
				Urgency:        b.severity,
				Reasoning:      b.explanation,
				Actions:        b.actions,
				ActionType:     "optimize",
				EstimatedTime:  b.estimatedTime,
				TimeReason:     "AI analysis of optimization time",
			}
		},
	)
}

// GenerateWorkflowImprovements generates workflow improvement suggestions
func (ai *aiSuggestionGeneratorImpl) GenerateWorkflowImprovements(
	ctx context.Context,
	workContext *entities.WorkContext,
) ([]*entities.TaskSuggestion, error) {
	improvements := ai.analyzeWorkflowImprovements(workContext)
	return generateSuggestionsFromAnalysis(
		ai,
		"generating workflow improvement suggestions",
		improvements,
		workContext,
		func(i workflowImprovement) suggestionConfig {
			return suggestionConfig{
				SuggestionType: entities.SuggestionTypeWorkflow,
				Title:          "Improve workflow: " + i.title,
				SourceName:     "workflow_analyzer_ai",
				Algorithm:      "workflow_efficiency_analysis",
				Confidence:     i.confidence,
				Relevance:      i.relevance,
				Urgency:        i.urgency,
				Reasoning:      i.reasoning,
				Actions:        i.steps,
				ActionType:     "improve",
				EstimatedTime:  i.estimatedTime,
				TimeReason:     "AI workflow analysis",
			}
		},
	)
}

// GenerateGoalAlignedSuggestions generates suggestions aligned with user goals
func (ai *aiSuggestionGeneratorImpl) GenerateGoalAlignedSuggestions(
	ctx context.Context,
	workContext *entities.WorkContext,
) ([]*entities.TaskSuggestion, error) {
	alignments := ai.analyzeGoalAlignment(workContext)
	return generateSuggestionsFromAnalysis(
		ai,
		"generating goal-aligned suggestions",
		alignments,
		workContext,
		func(a goalAlignment) suggestionConfig {
			return suggestionConfig{
				SuggestionType: entities.SuggestionTypePriority,
				Title:          "Align with goal: " + a.goal,
				SourceName:     "goal_alignment_ai",
				Algorithm:      "goal_alignment_analysis",
				Confidence:     a.confidence,
				Relevance:      a.relevance,
				Urgency:        a.urgency,
				Reasoning:      a.reasoning,
				Actions:        a.actions,
				ActionType:     "align",
				EstimatedTime:  a.estimatedTime,
				TimeReason:     "AI goal alignment analysis",
			}
		},
	)
}

// AnalyzeTaskComplexity performs AI analysis of task complexity
func (ai *aiSuggestionGeneratorImpl) AnalyzeTaskComplexity(task *entities.Task) (*TaskComplexityAnalysis, error) {
	ai.logger.Debug("analyzing task complexity", slog.String("task_id", task.ID))

	analysis := &TaskComplexityAnalysis{
		FactorsIdentified: make([]string, 0),
		SkillsRequired:    make([]string, 0),
		Dependencies:      make([]string, 0),
		RiskFactors:       make([]string, 0),
		Recommendations:   make([]string, 0),
	}

	content := strings.ToLower(task.Content)
	contentLength := len(task.Content)

	// Calculate base complexity score
	complexityScore := ai.calculateBaseComplexity(content, contentLength, analysis)

	// Analyze different aspects
	complexityScore += ai.analyzeComplexityKeywords(content, analysis)
	ai.analyzeSkillRequirements(content, analysis)
	complexityScore += ai.analyzeDependencies(content, analysis)
	complexityScore += ai.analyzeRiskFactors(content, analysis)

	// Finalize analysis
	analysis.ComplexityScore = ai.normalizeScore(complexityScore)
	analysis.EstimatedDuration = ai.calculateDuration(analysis.ComplexityScore)
	ai.generateRecommendations(analysis)
	analysis.Confidence = ai.calculateConfidence(analysis, contentLength)

	return analysis, nil
}

// calculateBaseComplexity determines initial complexity based on content length
func (ai *aiSuggestionGeneratorImpl) calculateBaseComplexity(_ string, contentLength int, analysis *TaskComplexityAnalysis) float64 {
	complexityScore := 0.5 // Start with neutral

	if contentLength > 200 {
		complexityScore += 0.2
		analysis.FactorsIdentified = append(analysis.FactorsIdentified, "long_description")
	} else if contentLength < 50 {
		complexityScore -= 0.1
		analysis.FactorsIdentified = append(analysis.FactorsIdentified, "short_description")
	}

	return complexityScore
}

// analyzeComplexityKeywords analyzes content for complexity-indicating keywords
func (ai *aiSuggestionGeneratorImpl) analyzeComplexityKeywords(content string, analysis *TaskComplexityAnalysis) float64 {
	complexityKeywords := map[string]float64{
		"complex":     0.3,
		"difficult":   0.2,
		"challenging": 0.2,
		"research":    0.15,
		"analyze":     0.1,
		"design":      0.15,
		"implement":   0.1,
		"integrate":   0.2,
		"optimize":    0.15,
		"refactor":    0.1,
		"test":        0.05,
		"debug":       0.1,
		"simple":      -0.1,
		"quick":       -0.15,
		"easy":        -0.1,
	}

	scoreAdjustment := 0.0
	for keyword, weight := range complexityKeywords {
		if strings.Contains(content, keyword) {
			scoreAdjustment += weight
			analysis.FactorsIdentified = append(analysis.FactorsIdentified, keyword)
		}
	}

	return scoreAdjustment
}

// analyzeSkillRequirements identifies required skills from content
func (ai *aiSuggestionGeneratorImpl) analyzeSkillRequirements(content string, analysis *TaskComplexityAnalysis) {
	skillKeywords := map[string][]string{
		"programming":   {"code", "implement", "debug", "algorithm"},
		"design":        {"design", "ui", "ux", "interface", "prototype"},
		"analysis":      {"analyze", "research", "investigate", "study"},
		"communication": {"meeting", "present", "write", "document"},
		"planning":      {"plan", "strategy", "roadmap", "timeline"},
	}

	for skill, keywords := range skillKeywords {
		for _, keyword := range keywords {
			if strings.Contains(content, keyword) {
				analysis.SkillsRequired = append(analysis.SkillsRequired, skill)
				break
			}
		}
	}
}

// analyzeDependencies checks for external dependencies
func (ai *aiSuggestionGeneratorImpl) analyzeDependencies(content string, analysis *TaskComplexityAnalysis) float64 {
	dependencyKeywords := []string{"depends", "requires", "needs", "after", "before", "prerequisite"}

	for _, keyword := range dependencyKeywords {
		if strings.Contains(content, keyword) {
			analysis.Dependencies = append(analysis.Dependencies, "external_dependency")
			return 0.1
		}
	}

	return 0.0
}

// analyzeRiskFactors identifies potential risk factors
func (ai *aiSuggestionGeneratorImpl) analyzeRiskFactors(content string, analysis *TaskComplexityAnalysis) float64 {
	riskKeywords := map[string]string{
		"deadline":     "time_pressure",
		"urgent":       "urgency_risk",
		"critical":     "criticality_risk",
		"new":          "unknown_territory",
		"first_time":   "inexperience_risk",
		"experimental": "uncertainty_risk",
	}

	scoreAdjustment := 0.0
	for keyword, risk := range riskKeywords {
		if strings.Contains(content, keyword) {
			analysis.RiskFactors = append(analysis.RiskFactors, risk)
			scoreAdjustment += 0.05
		}
	}

	return scoreAdjustment
}

// normalizeScore ensures score is within valid range
func (ai *aiSuggestionGeneratorImpl) normalizeScore(score float64) float64 {
	if score > 1.0 {
		return 1.0
	}
	if score < 0.0 {
		return 0.0
	}
	return score
}

// calculateDuration estimates task duration based on complexity
func (ai *aiSuggestionGeneratorImpl) calculateDuration(complexityScore float64) time.Duration {
	switch {
	case complexityScore > 0.8:
		return 4 * time.Hour
	case complexityScore > 0.6:
		return 2 * time.Hour
	case complexityScore < 0.3:
		return 30 * time.Minute
	default:
		return time.Hour
	}
}

// generateRecommendations creates actionable recommendations
func (ai *aiSuggestionGeneratorImpl) generateRecommendations(analysis *TaskComplexityAnalysis) {
	if analysis.ComplexityScore > 0.7 {
		analysis.BreakdownSuggested = true
		analysis.Recommendations = append(analysis.Recommendations, "Consider breaking this task into smaller subtasks")
	}

	if len(analysis.SkillsRequired) > 2 {
		analysis.Recommendations = append(analysis.Recommendations, "Multiple skills required - consider collaboration or learning time")
	}

	if len(analysis.RiskFactors) > 0 {
		analysis.Recommendations = append(analysis.Recommendations, "Risk factors identified - plan mitigation strategies")
	}
}

// calculateConfidence determines analysis confidence level
func (ai *aiSuggestionGeneratorImpl) calculateConfidence(analysis *TaskComplexityAnalysis, contentLength int) float64 {
	confidence := 0.6 // Base confidence

	if len(analysis.FactorsIdentified) > 2 {
		confidence += 0.2
	}
	if len(analysis.SkillsRequired) > 0 {
		confidence += 0.1
	}
	if contentLength > 100 {
		confidence += 0.1 // More content = better analysis
	}

	return math.Min(confidence, 1.0)
}

// GenerateTaskBreakdown generates subtask suggestions for complex tasks
func (ai *aiSuggestionGeneratorImpl) GenerateTaskBreakdown(task *entities.Task) ([]*entities.TaskSuggestion, error) {
	ai.logger.Debug("generating task breakdown", slog.String("task_id", task.ID))

	// Analyze task complexity first
	complexity, err := ai.AnalyzeTaskComplexity(task)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze complexity: %w", err)
	}

	if !complexity.BreakdownSuggested {
		return nil, nil // No breakdown needed
	}

	// Generate breakdown based on task type and content
	breakdownSteps := ai.generateBreakdownSteps(task, complexity)
	suggestions := make([]*entities.TaskSuggestion, 0, len(breakdownSteps))

	for i, step := range breakdownSteps {
		suggestion := ai.createBasicTaskSuggestion(
			entities.SuggestionTypeTemplate,
			fmt.Sprintf("Subtask %d: %s", i+1, step.title),
			"task_breakdown_ai",
			"complexity_based_breakdown",
			step.confidence,
			task.Repository,
		)
		suggestion.Relevance = 0.9 // Subtasks are highly relevant
		// Convert priority to urgency score
		if string(task.Priority) == constants.SeverityHigh {
			suggestion.Urgency = 0.8
		} else if string(task.Priority) == constants.SeverityMedium {
			suggestion.Urgency = 0.5
		} else {
			suggestion.Urgency = 0.3
		}
		suggestion.Priority = string(task.Priority)
		suggestion.TaskType = step.taskType
		suggestion.RelatedTaskIDs = []string{task.ID}

		suggestion.Reasoning = step.reasoning
		suggestion.SetEstimatedTime(step.estimatedTime, "AI breakdown analysis")

		// Add breakdown actions
		suggestion.AddAction("create_subtask", "Create this as a separate subtask", 1)
		suggestion.AddAction("track_progress", "Track progress on this component", 2)

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// SuggestTaskOptimizations suggests optimizations for similar tasks
func (ai *aiSuggestionGeneratorImpl) SuggestTaskOptimizations(
	task *entities.Task,
	historical []*entities.Task,
) ([]*entities.TaskSuggestion, error) {
	ai.logger.Debug("generating task optimizations", slog.String("task_id", task.ID))

	// Analyze similar historical tasks
	similarTasks := ai.findSimilarTasks(task, historical)
	if len(similarTasks) == 0 {
		return nil, nil
	}

	// Generate optimization insights
	optimizations := ai.analyzeTaskOptimizations(task, similarTasks)
	suggestions := make([]*entities.TaskSuggestion, 0, len(optimizations))

	for _, opt := range optimizations {
		suggestion := ai.createBasicTaskSuggestion(
			entities.SuggestionTypeOptimize,
			"Optimize: "+opt.title,
			"task_optimizer_ai",
			"historical_analysis",
			opt.confidence,
			task.Repository,
		)
		suggestion.Relevance = opt.relevance
		suggestion.Urgency = 0.5
		suggestion.Reasoning = opt.reasoning
		suggestion.RelatedTaskIDs = []string{task.ID}

		// Add optimization actions
		for i, action := range opt.actions {
			suggestion.AddAction("optimize", action, i+1)
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// suggestionConfig holds configuration for creating a task suggestion
type suggestionConfig struct {
	SuggestionType entities.SuggestionType
	Title          string
	SourceName     string
	Algorithm      string
	Confidence     float64
	Relevance      float64
	Urgency        float64
	Reasoning      string
	Actions        []string
	ActionType     string
	EstimatedTime  time.Duration
	TimeReason     string
}

// generateSuggestionsFromAnalysis is a generic helper that eliminates code duplication
// for AI suggestion generation methods that follow the same pattern
func generateSuggestionsFromAnalysis[T any](
	ai *aiSuggestionGeneratorImpl,
	logMessage string,
	analysisResults []T,
	workContext *entities.WorkContext,
	configFunc func(T) suggestionConfig,
) ([]*entities.TaskSuggestion, error) {
	ai.logger.Info(logMessage)

	suggestions := make([]*entities.TaskSuggestion, 0, len(analysisResults))

	for _, result := range analysisResults {
		config := configFunc(result)

		suggestion := entities.NewTaskSuggestion(
			config.SuggestionType,
			config.Title,
			entities.SuggestionSource{
				Type:       "ai",
				Name:       config.SourceName,
				Algorithm:  config.Algorithm,
				Confidence: config.Confidence,
			},
			workContext.Repository,
		)

		suggestion.Confidence = config.Confidence
		suggestion.Relevance = config.Relevance
		suggestion.Urgency = config.Urgency
		suggestion.Reasoning = config.Reasoning

		// Add actions
		for i, action := range config.Actions {
			suggestion.AddAction(config.ActionType, action, i+1)
		}

		suggestion.SetEstimatedTime(config.EstimatedTime, config.TimeReason)

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// createBasicTaskSuggestion creates a basic task suggestion with common fields set
// This eliminates code duplication across various suggestion creation methods
func (ai *aiSuggestionGeneratorImpl) createBasicTaskSuggestion(
	suggestionType entities.SuggestionType,
	title string,
	sourceName string,
	algorithm string,
	confidence float64,
	repository string,
) *entities.TaskSuggestion {
	suggestion := entities.NewTaskSuggestion(
		suggestionType,
		title,
		entities.SuggestionSource{
			Type:       "ai",
			Name:       sourceName,
			Algorithm:  algorithm,
			Confidence: confidence,
		},
		repository,
	)

	suggestion.Confidence = confidence
	return suggestion
}

// Helper types for AI analysis

type contextInsight struct {
	type_       string
	description string
	confidence  float64
	relevance   float64
	reasoning   string
}

type creativeOpportunity struct {
	suggestion    string
	confidence    float64
	relevance     float64
	reasoning     string
	estimatedTime time.Duration
	keywords      []string
}

type learningGap struct {
	topic         string
	confidence    float64
	relevance     float64
	urgency       float64
	reasoning     string
	estimatedTime time.Duration
	keywords      []string
}

type productivityBottleneck struct {
	area          string
	confidence    float64
	impact        float64
	severity      float64
	explanation   string
	actions       []string
	estimatedTime time.Duration
}

type workflowImprovement struct {
	title         string
	confidence    float64
	relevance     float64
	urgency       float64
	reasoning     string
	steps         []string
	estimatedTime time.Duration
}

type goalAlignment struct {
	goal          string
	confidence    float64
	relevance     float64
	urgency       float64
	reasoning     string
	actions       []string
	estimatedTime time.Duration
}

type breakdownStep struct {
	title         string
	taskType      string
	confidence    float64
	reasoning     string
	estimatedTime time.Duration
}

type taskOptimization struct {
	title      string
	confidence float64
	relevance  float64
	reasoning  string
	actions    []string
}

// Helper method implementations

func (ai *aiSuggestionGeneratorImpl) analyzeContextInsights(workContext *entities.WorkContext) []contextInsight {
	var insights []contextInsight

	// Energy-based insights
	if workContext.EnergyLevel < 0.4 {
		insights = append(insights, contextInsight{
			type_:       "energy_management",
			description: "Focus on energy restoration",
			confidence:  0.8,
			relevance:   1.0 - workContext.EnergyLevel,
			reasoning:   "Low energy levels detected, prioritizing restoration activities",
		})
	}

	// Focus-based insights
	if workContext.FocusLevel < 0.5 {
		insights = append(insights, contextInsight{
			type_:       "focus_enhancement",
			description: "Implement focus improvement strategies",
			confidence:  0.75,
			relevance:   1.0 - workContext.FocusLevel,
			reasoning:   "Low focus levels suggest need for concentration techniques",
		})
	}

	// Productivity insights
	if workContext.ProductivityScore > 0.8 {
		insights = append(insights, contextInsight{
			type_:       "momentum_maintenance",
			description: "Maintain high productivity momentum",
			confidence:  0.9,
			relevance:   workContext.ProductivityScore,
			reasoning:   "High productivity detected, focus on maintaining momentum",
		})
	}

	return insights
}

func (ai *aiSuggestionGeneratorImpl) createInsightBasedSuggestion(insight contextInsight, workContext *entities.WorkContext) *entities.TaskSuggestion {
	suggestion := ai.createBasicTaskSuggestion(
		entities.SuggestionTypeOptimize,
		insight.description,
		"context_insight_ai",
		"context_analysis",
		insight.confidence,
		workContext.Repository,
	)
	suggestion.Relevance = insight.relevance
	suggestion.Urgency = 0.6
	suggestion.Reasoning = insight.reasoning

	return suggestion
}

func (ai *aiSuggestionGeneratorImpl) generateAdaptiveSuggestions(workContext *entities.WorkContext) []*entities.TaskSuggestion {
	var suggestions []*entities.TaskSuggestion

	// Adaptive suggestions based on patterns and context
	if len(workContext.ActivePatterns) > 0 {
		primaryPattern := workContext.GetPrimaryPatterns()[0]

		suggestion := ai.createBasicTaskSuggestion(
			entities.SuggestionTypePattern,
			"Continue successful pattern: "+primaryPattern.Name,
			"adaptive_pattern_ai",
			"pattern_adaptation",
			primaryPattern.Confidence,
			workContext.Repository,
		)
		suggestion.Relevance = 0.8
		suggestion.Urgency = 0.5
		suggestion.PatternID = primaryPattern.ID
		suggestion.Reasoning = fmt.Sprintf("Pattern '%s' has been successful with %.1f%% confidence",
			primaryPattern.Name, primaryPattern.Confidence*100)

		suggestions = append(suggestions, suggestion)
	}

	return suggestions
}

func (ai *aiSuggestionGeneratorImpl) identifyCreativeOpportunities(workContext *entities.WorkContext) []creativeOpportunity {
	var opportunities []creativeOpportunity

	// Analyze current tasks for creative potential
	for _, task := range workContext.CurrentTasks {
		content := strings.ToLower(task.Content)

		if strings.Contains(content, "design") || strings.Contains(content, "creative") {
			opportunities = append(opportunities, creativeOpportunity{
				suggestion:    "Explore alternative design approaches",
				confidence:    0.7,
				relevance:     0.8,
				reasoning:     "Design-related task detected, suggesting creative exploration",
				estimatedTime: 45 * time.Minute,
				keywords:      []string{"design", "creative", "alternative"},
			})
		}

		if strings.Contains(content, "problem") || strings.Contains(content, "solve") {
			opportunities = append(opportunities, creativeOpportunity{
				suggestion:    "Apply creative problem-solving techniques",
				confidence:    0.65,
				relevance:     0.7,
				reasoning:     "Problem-solving task identified, suggesting creative approaches",
				estimatedTime: 30 * time.Minute,
				keywords:      []string{"problem-solving", "creative", "innovation"},
			})
		}
	}

	return opportunities
}

func (ai *aiSuggestionGeneratorImpl) identifyLearningGaps(workContext *entities.WorkContext) []learningGap {
	var gaps []learningGap

	// Analyze task types to identify knowledge gaps
	taskTypes := workContext.GetActiveTaskTypes()

	for _, taskType := range taskTypes {
		switch taskType {
		case constants.TaskTypeResearch:
			gaps = append(gaps, learningGap{
				topic:         "Advanced research methodologies",
				confidence:    0.6,
				relevance:     0.7,
				urgency:       0.5,
				reasoning:     "Research tasks detected, suggesting methodology improvement",
				estimatedTime: time.Hour,
				keywords:      []string{"research", "methodology", "analysis"},
			})
		case "implementation":
			gaps = append(gaps, learningGap{
				topic:         "Implementation best practices",
				confidence:    0.65,
				relevance:     0.8,
				urgency:       0.6,
				reasoning:     "Implementation tasks suggest need for best practice knowledge",
				estimatedTime: 90 * time.Minute,
				keywords:      []string{"implementation", "best-practices", "efficiency"},
			})
		}
	}

	return gaps
}

func (ai *aiSuggestionGeneratorImpl) analyzeProductivityBottlenecks(workContext *entities.WorkContext) []productivityBottleneck {
	var bottlenecks []productivityBottleneck

	// Analyze productivity metrics for bottlenecks
	if workContext.ProductivityScore < 0.6 {
		bottlenecks = append(bottlenecks, productivityBottleneck{
			area:          "Task prioritization",
			confidence:    0.7,
			impact:        1.0 - workContext.ProductivityScore,
			severity:      0.8,
			explanation:   "Low productivity suggests issues with task prioritization",
			actions:       []string{"Review task priorities", "Apply prioritization framework", "Eliminate low-value tasks"},
			estimatedTime: 30 * time.Minute,
		})
	}

	if workContext.FocusLevel < 0.5 {
		bottlenecks = append(bottlenecks, productivityBottleneck{
			area:          "Focus and concentration",
			confidence:    0.75,
			impact:        1.0 - workContext.FocusLevel,
			severity:      0.7,
			explanation:   "Low focus levels are limiting productivity",
			actions:       []string{"Eliminate distractions", "Use focus techniques", "Optimize environment"},
			estimatedTime: 20 * time.Minute,
		})
	}

	return bottlenecks
}

func (ai *aiSuggestionGeneratorImpl) analyzeWorkflowImprovements(workContext *entities.WorkContext) []workflowImprovement {
	var improvements []workflowImprovement

	// Analyze workflow patterns for improvements
	if len(workContext.ActivePatterns) > 0 {
		improvements = append(improvements, workflowImprovement{
			title:         "Optimize task sequencing",
			confidence:    0.7,
			relevance:     0.8,
			urgency:       0.5,
			reasoning:     "Active patterns suggest opportunities for sequence optimization",
			steps:         []string{"Analyze current sequence", "Identify dependencies", "Optimize order"},
			estimatedTime: 45 * time.Minute,
		})
	}

	return improvements
}

func (ai *aiSuggestionGeneratorImpl) analyzeGoalAlignment(workContext *entities.WorkContext) []goalAlignment {
	var alignments []goalAlignment

	// Analyze goals for alignment opportunities
	for _, goal := range workContext.Goals {
		if !goal.Completed && goal.Achieved < goal.Target {
			alignments = append(alignments, goalAlignment{
				goal:          goal.Description,
				confidence:    0.8,
				relevance:     0.9,
				urgency:       0.7,
				reasoning:     fmt.Sprintf("Goal '%s' is %d%% complete, needs attention", goal.Description, (goal.Achieved*100)/goal.Target),
				actions:       []string{"Focus on goal-aligned tasks", "Break down goal into steps", "Track progress"},
				estimatedTime: time.Hour,
			})
		}
	}

	return alignments
}

func (ai *aiSuggestionGeneratorImpl) generateBreakdownSteps(_ *entities.Task, complexity *TaskComplexityAnalysis) []breakdownStep {
	var steps []breakdownStep

	// Generate breakdown based on complexity factors
	if complexity.ComplexityScore > 0.7 {
		steps = append(steps, breakdownStep{
			title:         "Research and planning phase",
			taskType:      "planning",
			confidence:    0.8,
			reasoning:     "Complex tasks benefit from thorough planning",
			estimatedTime: 30 * time.Minute,
		})

		steps = append(steps, breakdownStep{
			title:         "Core implementation",
			taskType:      "implementation",
			confidence:    0.9,
			reasoning:     "Main implementation work",
			estimatedTime: 2 * time.Hour,
		})

		steps = append(steps, breakdownStep{
			title:         "Testing and validation",
			taskType:      "testing",
			confidence:    0.7,
			reasoning:     "Complex tasks require thorough testing",
			estimatedTime: 45 * time.Minute,
		})
	}

	return steps
}

func (ai *aiSuggestionGeneratorImpl) findSimilarTasks(task *entities.Task, historical []*entities.Task) []*entities.Task {
	var similar []*entities.Task

	for _, hist := range historical {
		similarity := ai.calculateTaskSimilarity(task, hist)
		if similarity > 0.6 {
			similar = append(similar, hist)
		}
	}

	return similar
}

func (ai *aiSuggestionGeneratorImpl) calculateTaskSimilarity(task1, task2 *entities.Task) float64 {
	// Simple similarity based on content and type
	similarity := 0.0

	// Compare task types using tags
	if len(task1.Tags) > 0 && len(task2.Tags) > 0 {
		commonTags := ai.countCommonTags(task1.Tags, task2.Tags)
		if commonTags > 0 {
			similarity += 0.4
		}
	}

	if task1.Priority == task2.Priority {
		similarity += 0.2
	}

	// Content similarity (simplified)
	content1 := strings.ToLower(task1.Content)
	content2 := strings.ToLower(task2.Content)

	words1 := strings.Fields(content1)
	words2 := strings.Fields(content2)

	commonWords := 0
	totalWords := len(words1) + len(words2)

	for _, word1 := range words1 {
		for _, word2 := range words2 {
			if word1 == word2 && len(word1) > 3 { // Only count meaningful words
				commonWords++
			}
		}
	}

	if totalWords > 0 {
		similarity += float64(commonWords*2) / float64(totalWords) * 0.4
	}

	return similarity
}

// countCommonTags counts common tags between two tag slices
func (ai *aiSuggestionGeneratorImpl) countCommonTags(tags1, tags2 []string) int {
	tagSet := make(map[string]bool)
	for _, tag := range tags2 {
		tagSet[tag] = true
	}

	commonCount := 0
	for _, tag := range tags1 {
		if tagSet[tag] {
			commonCount++
		}
	}

	return commonCount
}

func (ai *aiSuggestionGeneratorImpl) analyzeTaskOptimizations(_ *entities.Task, similar []*entities.Task) []taskOptimization {
	var optimizations []taskOptimization

	// Analyze common patterns in similar tasks
	if len(similar) > 2 {
		optimizations = append(optimizations, taskOptimization{
			title:      "Apply proven approaches from similar tasks",
			confidence: 0.75,
			relevance:  0.8,
			reasoning:  fmt.Sprintf("Found %d similar tasks with successful patterns", len(similar)),
			actions:    []string{"Review similar task approaches", "Adopt proven strategies", "Avoid common pitfalls"},
		})
	}

	return optimizations
}
