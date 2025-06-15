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
)

// SuggestionRanker interface defines advanced ranking and scoring capabilities
type SuggestionRanker interface {
	// Core ranking functionality
	RankSuggestions(suggestions []*entities.TaskSuggestion, workContext *entities.WorkContext) []*entities.TaskSuggestion
	CalculateRelevanceScore(suggestion *entities.TaskSuggestion, workContext *entities.WorkContext) float64
	CalculateUrgencyScore(suggestion *entities.TaskSuggestion, workContext *entities.WorkContext) float64
	CalculateConfidenceScore(suggestion *entities.TaskSuggestion, workContext *entities.WorkContext) float64

	// Advanced scoring
	CalculatePersonalizationScore(suggestion *entities.TaskSuggestion, workContext *entities.WorkContext) float64
	CalculateContextFitScore(suggestion *entities.TaskSuggestion, workContext *entities.WorkContext) float64
	CalculateTimingScore(suggestion *entities.TaskSuggestion, workContext *entities.WorkContext) float64

	// Multi-criteria ranking
	RankWithCriteria(suggestions []*entities.TaskSuggestion, criteria *RankingCriteria, workContext *entities.WorkContext) []*entities.TaskSuggestion
	CalculateCompositeScore(suggestion *entities.TaskSuggestion, criteria *RankingCriteria, workContext *entities.WorkContext) float64

	// Dynamic adaptation
	AdaptRankingWeights(ctx context.Context, feedback []*entities.SuggestionFeedback) *RankingCriteria
	UpdatePersonalizationModel(ctx context.Context, userFeedback *entities.SuggestionFeedback) error
}

// RankingCriteria defines weights and parameters for ranking
type RankingCriteria struct {
	// Core scoring weights (should sum to 1.0)
	RelevanceWeight       float64 `json:"relevance_weight"`       // Weight for relevance scoring
	UrgencyWeight         float64 `json:"urgency_weight"`         // Weight for urgency scoring
	ConfidenceWeight      float64 `json:"confidence_weight"`      // Weight for confidence scoring
	PersonalizationWeight float64 `json:"personalization_weight"` // Weight for personalization

	// Context-specific weights
	ContextFitWeight float64 `json:"context_fit_weight"` // Weight for context fit
	TimingWeight     float64 `json:"timing_weight"`      // Weight for timing appropriateness
	PatternWeight    float64 `json:"pattern_weight"`     // Weight for pattern matching

	// Penalty factors
	RecencyPenalty    float64 `json:"recency_penalty"`    // Penalty for recently rejected suggestions
	RepetitionPenalty float64 `json:"repetition_penalty"` // Penalty for repetitive suggestions
	ComplexityPenalty float64 `json:"complexity_penalty"` // Penalty for overly complex suggestions

	// Boost factors
	ProductivityBoost  float64 `json:"productivity_boost"`   // Boost for high-productivity contexts
	FocusBoost         float64 `json:"focus_boost"`          // Boost for high-focus contexts
	GoalAlignmentBoost float64 `json:"goal_alignment_boost"` // Boost for goal-aligned suggestions

	// Adaptive parameters
	LearningRate      float64 `json:"learning_rate"`      // Rate of adaptation to feedback
	ExplorationFactor float64 `json:"exploration_factor"` // Factor for exploration vs exploitation
	DiversityTarget   float64 `json:"diversity_target"`   // Target diversity in ranked results
}

// DefaultRankingCriteria returns default ranking criteria
func DefaultRankingCriteria() *RankingCriteria {
	return &RankingCriteria{
		// Core weights (sum to 1.0)
		RelevanceWeight:       0.30,
		UrgencyWeight:         0.25,
		ConfidenceWeight:      0.20,
		PersonalizationWeight: 0.25,

		// Context weights
		ContextFitWeight: 0.15,
		TimingWeight:     0.10,
		PatternWeight:    0.15,

		// Penalties
		RecencyPenalty:    0.2,
		RepetitionPenalty: 0.3,
		ComplexityPenalty: 0.1,

		// Boosts
		ProductivityBoost:  0.15,
		FocusBoost:         0.10,
		GoalAlignmentBoost: 0.20,

		// Adaptive parameters
		LearningRate:      0.1,
		ExplorationFactor: 0.2,
		DiversityTarget:   0.3,
	}
}

// scoredSuggestion represents a suggestion with its calculated score and components
type scoredSuggestion struct {
	suggestion *entities.TaskSuggestion
	score      float64
	components map[string]float64 // Individual scoring components
}

// suggestionRankerImpl implements the SuggestionRanker interface
type suggestionRankerImpl struct {
	criteria             *RankingCriteria
	contextAnalyzer      ContextAnalyzer
	patternDetector      PatternDetector
	feedbackHistory      map[string][]*entities.SuggestionFeedback // Repository -> feedback history
	personalizationModel map[string]*PersonalizationProfile        // Repository -> personalization profile
	logger               *slog.Logger
}

// PersonalizationProfile represents learned user preferences
type PersonalizationProfile struct {
	UserID              string                              `json:"user_id"`
	Repository          string                              `json:"repository"`
	PreferredTypes      map[entities.SuggestionType]float64 `json:"preferred_types"`      // Type -> preference score
	PreferredTiming     map[string]float64                  `json:"preferred_timing"`     // Time of day -> preference
	PreferredComplexity float64                             `json:"preferred_complexity"` // 0-1 complexity preference
	AvoidancePatterns   []string                            `json:"avoidance_patterns"`   // Patterns to avoid
	SuccessfulPatterns  []string                            `json:"successful_patterns"`  // Patterns that work well
	FeedbackCount       int                                 `json:"feedback_count"`       // Number of feedback entries
	LastUpdated         time.Time                           `json:"last_updated"`
	Metadata            map[string]interface{}              `json:"metadata"`
}

// NewSuggestionRanker creates a new suggestion ranker
func NewSuggestionRanker(
	criteria *RankingCriteria,
	contextAnalyzer ContextAnalyzer,
	patternDetector PatternDetector,
	logger *slog.Logger,
) SuggestionRanker {
	if criteria == nil {
		criteria = DefaultRankingCriteria()
	}

	return &suggestionRankerImpl{
		criteria:             criteria,
		contextAnalyzer:      contextAnalyzer,
		patternDetector:      patternDetector,
		feedbackHistory:      make(map[string][]*entities.SuggestionFeedback),
		personalizationModel: make(map[string]*PersonalizationProfile),
		logger:               logger,
	}
}

// RankSuggestions ranks suggestions using default criteria
func (sr *suggestionRankerImpl) RankSuggestions(
	suggestions []*entities.TaskSuggestion,
	workContext *entities.WorkContext,
) []*entities.TaskSuggestion {
	return sr.RankWithCriteria(suggestions, sr.criteria, workContext)
}

// RankWithCriteria ranks suggestions using specified criteria
func (sr *suggestionRankerImpl) RankWithCriteria(
	suggestions []*entities.TaskSuggestion,
	criteria *RankingCriteria,
	workContext *entities.WorkContext,
) []*entities.TaskSuggestion {
	if len(suggestions) == 0 {
		return suggestions
	}

	sr.logger.Info("ranking suggestions with criteria",
		slog.Int("suggestion_count", len(suggestions)),
		slog.String("repository", workContext.Repository))

	// Calculate composite scores for all suggestions
	var scored []scoredSuggestion
	for _, suggestion := range suggestions {
		score := sr.CalculateCompositeScore(suggestion, criteria, workContext)
		components := sr.calculateScoringComponents(suggestion, criteria, workContext)

		scored = append(scored, scoredSuggestion{
			suggestion: suggestion,
			score:      score,
			components: components,
		})

		// Update suggestion scores
		suggestion.Relevance = sr.CalculateRelevanceScore(suggestion, workContext)
		suggestion.Urgency = sr.CalculateUrgencyScore(suggestion, workContext)
		suggestion.Confidence = sr.CalculateConfidenceScore(suggestion, workContext)
	}

	// Sort by composite score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Apply diversity filtering if enabled
	if criteria.DiversityTarget > 0 {
		scored = sr.applyDiversityFiltering(scored, criteria.DiversityTarget)
	}

	// Extract ranked suggestions
	ranked := make([]*entities.TaskSuggestion, len(scored))
	for i, ss := range scored {
		ranked[i] = ss.suggestion
		// Store ranking metadata
		if ranked[i].Metadata == nil {
			ranked[i].Metadata = make(map[string]interface{})
		}
		ranked[i].Metadata["ranking_score"] = ss.score
		ranked[i].Metadata["ranking_components"] = ss.components
		ranked[i].Metadata["ranking_position"] = i + 1
	}

	sr.logger.Info("suggestion ranking completed",
		slog.Int("ranked_count", len(ranked)),
		slog.Float64("top_score", ranked[0].Metadata["ranking_score"].(float64)))

	return ranked
}

// CalculateCompositeScore calculates the overall composite score
func (sr *suggestionRankerImpl) CalculateCompositeScore(
	suggestion *entities.TaskSuggestion,
	criteria *RankingCriteria,
	workContext *entities.WorkContext,
) float64 {
	// Core scoring components
	relevance := sr.CalculateRelevanceScore(suggestion, workContext)
	urgency := sr.CalculateUrgencyScore(suggestion, workContext)
	confidence := sr.CalculateConfidenceScore(suggestion, workContext)
	personalization := sr.CalculatePersonalizationScore(suggestion, workContext)

	// Additional components
	contextFit := sr.CalculateContextFitScore(suggestion, workContext)
	timing := sr.CalculateTimingScore(suggestion, workContext)
	pattern := sr.calculatePatternScore(suggestion, workContext)

	// Base composite score
	baseScore := (relevance * criteria.RelevanceWeight) +
		(urgency * criteria.UrgencyWeight) +
		(confidence * criteria.ConfidenceWeight) +
		(personalization * criteria.PersonalizationWeight)

	// Add context-specific components
	contextScore := (contextFit * criteria.ContextFitWeight) +
		(timing * criteria.TimingWeight) +
		(pattern * criteria.PatternWeight)

	// Apply boosts
	boosts := sr.calculateBoosts(suggestion, criteria, workContext)

	// Apply penalties
	penalties := sr.calculatePenalties(suggestion, criteria, workContext)

	// Final composite score
	finalScore := baseScore + contextScore + boosts - penalties

	// Ensure score is within bounds
	if finalScore > 1.0 {
		finalScore = 1.0
	} else if finalScore < 0.0 {
		finalScore = 0.0
	}

	return finalScore
}

// CalculateRelevanceScore calculates relevance to current context
func (sr *suggestionRankerImpl) CalculateRelevanceScore(
	suggestion *entities.TaskSuggestion,
	workContext *entities.WorkContext,
) float64 {
	var relevanceFactors []float64

	// Repository relevance
	if suggestion.Repository == workContext.Repository {
		relevanceFactors = append(relevanceFactors, 1.0)
	} else {
		relevanceFactors = append(relevanceFactors, 0.3)
	}

	// Task type relevance
	activeTypes := workContext.GetActiveTaskTypes()
	typeRelevance := 0.5 // Default
	for _, activeType := range activeTypes {
		if suggestion.TaskType == activeType {
			typeRelevance = 0.9
			break
		}
	}
	relevanceFactors = append(relevanceFactors, typeRelevance)

	// Keyword relevance
	keywordRelevance := sr.calculateKeywordRelevance(suggestion, workContext)
	relevanceFactors = append(relevanceFactors, keywordRelevance)

	// Pattern relevance
	patternRelevance := sr.calculatePatternRelevance(suggestion, workContext)
	relevanceFactors = append(relevanceFactors, patternRelevance)

	// Goal relevance
	goalRelevance := sr.calculateGoalRelevance(suggestion, workContext)
	relevanceFactors = append(relevanceFactors, goalRelevance)

	// Calculate weighted average
	return sr.calculateWeightedAverage(relevanceFactors, []float64{0.3, 0.25, 0.2, 0.15, 0.1})
}

// CalculateUrgencyScore calculates urgency based on context and timing
func (sr *suggestionRankerImpl) CalculateUrgencyScore(
	suggestion *entities.TaskSuggestion,
	workContext *entities.WorkContext,
) float64 {
	baseUrgency := suggestion.Urgency

	// Time-based urgency adjustments
	if suggestion.Type == entities.SuggestionTypeBreak {
		if workContext.EnergyLevel < 0.3 || workContext.IsHighStress() {
			baseUrgency += 0.3
		}
	}

	// Priority-based urgency
	switch strings.ToLower(suggestion.Priority) {
	case "critical", "urgent":
		baseUrgency += 0.2
	case "high":
		baseUrgency += 0.1
	case "low":
		baseUrgency -= 0.1
	}

	// Deadline proximity (if related tasks have deadlines)
	deadlineUrgency := sr.calculateDeadlineUrgency(suggestion, workContext)
	baseUrgency += deadlineUrgency * 0.2

	// Productivity context urgency
	if workContext.ProductivityScore < 0.5 && suggestion.Type == entities.SuggestionTypeOptimize {
		baseUrgency += 0.15
	}

	return math.Min(baseUrgency, 1.0)
}

// CalculateConfidenceScore calculates confidence based on multiple factors
func (sr *suggestionRankerImpl) CalculateConfidenceScore(
	suggestion *entities.TaskSuggestion,
	workContext *entities.WorkContext,
) float64 {
	baseConfidence := suggestion.Source.Confidence

	// Source-based confidence adjustments
	switch suggestion.Source.Type {
	case "pattern":
		if suggestion.PatternID != "" {
			// Higher confidence for proven patterns
			for _, pattern := range workContext.ActivePatterns {
				if pattern.ID == suggestion.PatternID {
					baseConfidence += (pattern.SuccessRate - 0.5) * 0.2
					break
				}
			}
		}
	case "ai":
		// AI confidence depends on context richness
		contextRichness := sr.calculateContextRichness(workContext)
		baseConfidence += (contextRichness - 0.5) * 0.1
	case "analytics":
		// Analytics confidence is high for data-rich contexts
		if len(workContext.RecentTasks) > 5 {
			baseConfidence += 0.1
		}
	}

	// Historical performance adjustment
	historicalConfidence := sr.getHistoricalConfidence(suggestion.Type, workContext.Repository)
	baseConfidence += historicalConfidence * 0.15

	// Recency adjustment
	if time.Since(suggestion.GeneratedAt) < time.Hour {
		baseConfidence += 0.05 // Slight boost for fresh suggestions
	}

	return math.Min(baseConfidence, 1.0)
}

// CalculatePersonalizationScore calculates personalization based on user preferences
func (sr *suggestionRankerImpl) CalculatePersonalizationScore(
	suggestion *entities.TaskSuggestion,
	workContext *entities.WorkContext,
) float64 {
	profile := sr.getPersonalizationProfile(workContext.Repository)
	if profile == nil {
		return 0.5 // No personalization data
	}

	var personalizeFactors []float64

	// Type preference
	if typePreference, exists := profile.PreferredTypes[suggestion.Type]; exists {
		personalizeFactors = append(personalizeFactors, typePreference)
	} else {
		personalizeFactors = append(personalizeFactors, 0.5)
	}

	// Timing preference
	timeOfDay := workContext.TimeOfDay
	if timingPreference, exists := profile.PreferredTiming[timeOfDay]; exists {
		personalizeFactors = append(personalizeFactors, timingPreference)
	} else {
		personalizeFactors = append(personalizeFactors, 0.5)
	}

	// Complexity preference
	suggestionComplexity := sr.estimateSuggestionComplexity(suggestion)
	complexityFit := 1.0 - math.Abs(suggestionComplexity-profile.PreferredComplexity)
	personalizeFactors = append(personalizeFactors, complexityFit)

	// Avoidance patterns
	avoidanceScore := 1.0
	for _, pattern := range profile.AvoidancePatterns {
		if strings.Contains(strings.ToLower(suggestion.Content), strings.ToLower(pattern)) {
			avoidanceScore -= 0.3
		}
	}
	personalizeFactors = append(personalizeFactors, math.Max(avoidanceScore, 0.0))

	// Successful patterns
	successScore := 0.5
	for _, pattern := range profile.SuccessfulPatterns {
		if strings.Contains(strings.ToLower(suggestion.Content), strings.ToLower(pattern)) {
			successScore += 0.2
		}
	}
	personalizeFactors = append(personalizeFactors, math.Min(successScore, 1.0))

	return sr.calculateWeightedAverage(personalizeFactors, []float64{0.3, 0.2, 0.2, 0.15, 0.15})
}

// CalculateContextFitScore calculates how well suggestion fits current context
func (sr *suggestionRankerImpl) CalculateContextFitScore(
	suggestion *entities.TaskSuggestion,
	workContext *entities.WorkContext,
) float64 {
	var fitFactors []float64

	// Energy level fit
	energyFit := sr.calculateEnergyFit(suggestion, workContext.EnergyLevel)
	fitFactors = append(fitFactors, energyFit)

	// Focus level fit
	focusFit := sr.calculateFocusFit(suggestion, workContext.FocusLevel)
	fitFactors = append(fitFactors, focusFit)

	// Productivity context fit
	productivityFit := sr.calculateProductivityFit(suggestion, workContext.ProductivityScore)
	fitFactors = append(fitFactors, productivityFit)

	// Stress level fit
	stressFit := sr.calculateStressFit(suggestion, workContext.StressIndicators)
	fitFactors = append(fitFactors, stressFit)

	// Workload fit
	workloadFit := sr.calculateWorkloadFit(suggestion, len(workContext.CurrentTasks))
	fitFactors = append(fitFactors, workloadFit)

	return sr.calculateWeightedAverage(fitFactors, []float64{0.25, 0.25, 0.2, 0.15, 0.15})
}

// CalculateTimingScore calculates timing appropriateness
func (sr *suggestionRankerImpl) CalculateTimingScore(
	suggestion *entities.TaskSuggestion,
	workContext *entities.WorkContext,
) float64 {
	var timingFactors []float64

	// Time of day appropriateness
	timeOfDayScore := sr.getTimeOfDayScore(suggestion.Type, workContext.TimeOfDay)
	timingFactors = append(timingFactors, timeOfDayScore)

	// Day of week appropriateness
	dayOfWeekScore := sr.getDayOfWeekScore(suggestion.Type, workContext.DayOfWeek)
	timingFactors = append(timingFactors, dayOfWeekScore)

	// Working hours appropriateness
	workingHoursScore := 1.0
	if workContext.WorkingHours != nil {
		workingHoursScore = sr.getWorkingHoursScore(suggestion.Type, workContext.WorkingHours)
	}
	timingFactors = append(timingFactors, workingHoursScore)

	// Session duration appropriateness
	sessionScore := sr.getSessionDurationScore(suggestion, workContext)
	timingFactors = append(timingFactors, sessionScore)

	return sr.calculateWeightedAverage(timingFactors, []float64{0.3, 0.2, 0.25, 0.25})
}

// AdaptRankingWeights adapts ranking weights based on feedback
func (sr *suggestionRankerImpl) AdaptRankingWeights(
	ctx context.Context,
	feedback []*entities.SuggestionFeedback,
) *RankingCriteria {
	if len(feedback) == 0 {
		return sr.criteria
	}

	sr.logger.Info("adapting ranking weights based on feedback",
		slog.Int("feedback_count", len(feedback)))

	adaptedCriteria := *sr.criteria // Copy current criteria

	// Analyze feedback patterns
	// Note: In a real implementation, you'd store suggestion type with feedback
	// For now, we'll work with available feedback data
	for _, fb := range feedback {
		if fb.Accepted && fb.Helpful {
			// Increment successful patterns
			sr.logger.Debug("positive feedback received for suggestion")
		} else if !fb.Accepted {
			// Increment rejected patterns
			sr.logger.Debug("negative feedback received for suggestion")
		}
	}

	// Adapt weights based on feedback patterns
	learningRate := adaptedCriteria.LearningRate

	// Example adaptations (simplified)
	acceptanceRate := sr.calculateAcceptanceRate(feedback)
	if acceptanceRate < 0.3 {
		// Low acceptance rate - adjust weights
		adaptedCriteria.PersonalizationWeight += learningRate * 0.1
		adaptedCriteria.RelevanceWeight -= learningRate * 0.05
	} else if acceptanceRate > 0.8 {
		// High acceptance rate - maintain or enhance
		adaptedCriteria.ConfidenceWeight += learningRate * 0.05
	}

	// Normalize weights to ensure they sum to 1.0
	sr.normalizeWeights(&adaptedCriteria)

	return &adaptedCriteria
}

// UpdatePersonalizationModel updates the personalization model with new feedback
func (sr *suggestionRankerImpl) UpdatePersonalizationModel(
	ctx context.Context,
	userFeedback *entities.SuggestionFeedback,
) error {
	sr.logger.Debug("updating personalization model",
		slog.Bool("accepted", userFeedback.Accepted),
		slog.Bool("helpful", userFeedback.Helpful))

	// In a real implementation, you'd:
	// 1. Extract suggestion details from feedback
	// 2. Update user preference model
	// 3. Store updated model in persistent storage

	// For now, just log the update
	sr.logger.Info("personalization model updated")

	return nil
}

// Helper methods

func (sr *suggestionRankerImpl) calculateScoringComponents(
	suggestion *entities.TaskSuggestion,
	_ *RankingCriteria,
	workContext *entities.WorkContext,
) map[string]float64 {
	return map[string]float64{
		"relevance":       sr.CalculateRelevanceScore(suggestion, workContext),
		"urgency":         sr.CalculateUrgencyScore(suggestion, workContext),
		"confidence":      sr.CalculateConfidenceScore(suggestion, workContext),
		"personalization": sr.CalculatePersonalizationScore(suggestion, workContext),
		"context_fit":     sr.CalculateContextFitScore(suggestion, workContext),
		"timing":          sr.CalculateTimingScore(suggestion, workContext),
		"pattern":         sr.calculatePatternScore(suggestion, workContext),
	}
}

func (sr *suggestionRankerImpl) calculatePatternScore(
	suggestion *entities.TaskSuggestion,
	workContext *entities.WorkContext,
) float64 {
	if suggestion.PatternID == "" {
		return 0.5 // No pattern association
	}

	for _, pattern := range workContext.ActivePatterns {
		if pattern.ID == suggestion.PatternID {
			return pattern.Confidence
		}
	}

	return 0.3 // Pattern not found in active patterns
}

func (sr *suggestionRankerImpl) calculateBoosts(
	suggestion *entities.TaskSuggestion,
	criteria *RankingCriteria,
	workContext *entities.WorkContext,
) float64 {
	var boosts float64

	// Productivity boost
	if workContext.ProductivityScore > 0.8 && suggestion.Type == entities.SuggestionTypeNext {
		boosts += criteria.ProductivityBoost
	}

	// Focus boost
	if workContext.FocusLevel > 0.8 &&
		(suggestion.Type == entities.SuggestionTypeNext || suggestion.Type == entities.SuggestionTypePattern) {
		boosts += criteria.FocusBoost
	}

	// Goal alignment boost
	goalAligned := sr.checkGoalAlignment(suggestion, workContext)
	if goalAligned {
		boosts += criteria.GoalAlignmentBoost
	}

	return boosts
}

func (sr *suggestionRankerImpl) calculatePenalties(
	suggestion *entities.TaskSuggestion,
	criteria *RankingCriteria,
	workContext *entities.WorkContext,
) float64 {
	var penalties float64

	// Recency penalty for recently rejected similar suggestions
	if sr.wasRecentlyRejected(suggestion, workContext.Repository) {
		penalties += criteria.RecencyPenalty
	}

	// Repetition penalty for similar suggestions
	if sr.isRepetitive(suggestion, workContext.Repository) {
		penalties += criteria.RepetitionPenalty
	}

	// Complexity penalty for overly complex suggestions
	complexity := sr.estimateSuggestionComplexity(suggestion)
	if complexity > 0.8 && workContext.EnergyLevel < 0.4 {
		penalties += criteria.ComplexityPenalty
	}

	return penalties
}

func (sr *suggestionRankerImpl) applyDiversityFiltering(
	scored []scoredSuggestion,
	diversityTarget float64,
) []scoredSuggestion {
	if diversityTarget <= 0 || len(scored) <= 1 {
		return scored
	}

	var diversified []scoredSuggestion
	typesSeen := make(map[entities.SuggestionType]int)

	// First, add top suggestions ensuring diversity
	for _, ss := range scored {
		typeCount := typesSeen[ss.suggestion.Type]
		maxPerType := int(float64(len(scored)) * diversityTarget)

		if typeCount < maxPerType || len(diversified) < 3 { // Always include top 3
			diversified = append(diversified, ss)
			typesSeen[ss.suggestion.Type]++
		}
	}

	return diversified
}

func (sr *suggestionRankerImpl) calculateKeywordRelevance(
	suggestion *entities.TaskSuggestion,
	workContext *entities.WorkContext,
) float64 {
	if len(suggestion.Keywords) == 0 {
		return 0.5
	}

	// Extract keywords from current context
	contextKeywords := sr.contextAnalyzer.ExtractKeywords(workContext.CurrentTasks)
	contextKeywords = append(contextKeywords, sr.contextAnalyzer.ExtractKeywords(workContext.RecentTasks)...)

	if len(contextKeywords) == 0 {
		return 0.5
	}

	// Calculate Jaccard similarity
	suggestionSet := make(map[string]bool)
	for _, keyword := range suggestion.Keywords {
		suggestionSet[strings.ToLower(keyword)] = true
	}

	contextSet := make(map[string]bool)
	for _, keyword := range contextKeywords {
		contextSet[strings.ToLower(keyword)] = true
	}

	intersection := 0
	for keyword := range suggestionSet {
		if contextSet[keyword] {
			intersection++
		}
	}

	union := len(suggestionSet) + len(contextSet) - intersection
	if union == 0 {
		return 0.5
	}

	return float64(intersection) / float64(union)
}

func (sr *suggestionRankerImpl) calculatePatternRelevance(
	suggestion *entities.TaskSuggestion,
	workContext *entities.WorkContext,
) float64 {
	if suggestion.PatternID == "" {
		return 0.5
	}

	// Check if pattern is in primary patterns
	primaryPatterns := workContext.GetPrimaryPatterns()
	for _, pattern := range primaryPatterns {
		if pattern.ID == suggestion.PatternID {
			return pattern.Confidence
		}
	}

	// Check if pattern is in active patterns
	for _, pattern := range workContext.ActivePatterns {
		if pattern.ID == suggestion.PatternID {
			return pattern.Confidence * 0.8 // Slightly lower score
		}
	}

	return 0.3 // Pattern not found
}

func (sr *suggestionRankerImpl) calculateGoalRelevance(
	suggestion *entities.TaskSuggestion,
	workContext *entities.WorkContext,
) float64 {
	if len(workContext.Goals) == 0 {
		return 0.5
	}

	// Check if suggestion aligns with any current goals
	relevance := 0.0
	for _, goal := range workContext.Goals {
		alignment := sr.calculateGoalAlignment(suggestion, goal)
		if alignment > relevance {
			relevance = alignment
		}
	}

	return relevance
}

func (sr *suggestionRankerImpl) calculateGoalAlignment(
	suggestion *entities.TaskSuggestion,
	goal entities.SessionGoal,
) float64 {
	// Simple keyword-based alignment
	suggestionText := strings.ToLower(suggestion.Content + " " + suggestion.Description)
	goalText := strings.ToLower(goal.Description)

	// Check for common words
	suggestionWords := strings.Fields(suggestionText)
	goalWords := strings.Fields(goalText)

	commonWords := 0
	for _, sWord := range suggestionWords {
		for _, gWord := range goalWords {
			if sWord == gWord && len(sWord) > 3 {
				commonWords++
			}
		}
	}

	if len(suggestionWords)+len(goalWords) == 0 {
		return 0.5
	}

	return float64(commonWords*2) / float64(len(suggestionWords)+len(goalWords))
}

func (sr *suggestionRankerImpl) calculateWeightedAverage(values, weights []float64) float64 {
	if len(values) != len(weights) || len(values) == 0 {
		return 0.5
	}

	var weightedSum, totalWeight float64
	for i, value := range values {
		weightedSum += value * weights[i]
		totalWeight += weights[i]
	}

	if totalWeight == 0 {
		return 0.5
	}

	return weightedSum / totalWeight
}

func (sr *suggestionRankerImpl) calculateEnergyFit(suggestion *entities.TaskSuggestion, energyLevel float64) float64 {
	switch suggestion.Type {
	case entities.SuggestionTypeBreak:
		return 1.0 - energyLevel // Higher fit when energy is low
	case entities.SuggestionTypeNext:
		return energyLevel // Higher fit when energy is high
	case entities.SuggestionTypeOptimize:
		return 0.5 + (energyLevel-0.5)*0.5 // Moderate energy preferred
	default:
		return 0.7 // Default moderate fit
	}
}

func (sr *suggestionRankerImpl) calculateFocusFit(suggestion *entities.TaskSuggestion, focusLevel float64) float64 {
	switch suggestion.Type {
	case entities.SuggestionTypeBreak:
		return 1.0 - focusLevel // Higher fit when focus is low
	case entities.SuggestionTypeNext, entities.SuggestionTypePattern:
		return focusLevel // Higher fit when focus is high
	default:
		return 0.6
	}
}

func (sr *suggestionRankerImpl) calculateProductivityFit(suggestion *entities.TaskSuggestion, productivityScore float64) float64 {
	switch suggestion.Type {
	case entities.SuggestionTypeOptimize:
		return 1.0 - productivityScore // Higher fit when productivity is low
	case entities.SuggestionTypeNext:
		return productivityScore // Higher fit when productivity is high
	default:
		return 0.6
	}
}

func (sr *suggestionRankerImpl) calculateStressFit(suggestion *entities.TaskSuggestion, stressIndicators []entities.StressIndicator) float64 {
	stressLevel := float64(len(stressIndicators)) / 5.0 // Normalize to 0-1
	if stressLevel > 1.0 {
		stressLevel = 1.0
	}

	switch suggestion.Type {
	case entities.SuggestionTypeBreak:
		return stressLevel // Higher fit when stress is high
	case entities.SuggestionTypeOptimize:
		return stressLevel * 0.8 // Good fit for moderate stress
	default:
		return 1.0 - stressLevel*0.5 // Lower fit when stress is high
	}
}

func (sr *suggestionRankerImpl) calculateWorkloadFit(suggestion *entities.TaskSuggestion, currentTaskCount int) float64 {
	workloadFactor := math.Min(float64(currentTaskCount)/10.0, 1.0) // Normalize to 0-1

	switch suggestion.Type {
	case entities.SuggestionTypeBreak:
		return workloadFactor // Higher fit when workload is high
	case entities.SuggestionTypeNext:
		return 1.0 - workloadFactor*0.5 // Lower fit when heavily loaded
	default:
		return 0.7 - workloadFactor*0.2
	}
}

func (sr *suggestionRankerImpl) getTimeOfDayScore(suggestionType entities.SuggestionType, timeOfDay string) float64 {
	scores := map[string]map[entities.SuggestionType]float64{
		"morning": {
			entities.SuggestionTypeNext:     0.9,
			entities.SuggestionTypeLearning: 0.8,
			entities.SuggestionTypePattern:  0.8,
			entities.SuggestionTypeBreak:    0.3,
		},
		"afternoon": {
			entities.SuggestionTypeNext:     0.8,
			entities.SuggestionTypeOptimize: 0.9,
			entities.SuggestionTypeWorkflow: 0.8,
			entities.SuggestionTypeBreak:    0.7,
		},
		"evening": {
			entities.SuggestionTypeOptimize: 0.7,
			entities.SuggestionTypeBreak:    0.8,
			entities.SuggestionTypeNext:     0.6,
		},
	}

	if timeScores, exists := scores[timeOfDay]; exists {
		if score, exists := timeScores[suggestionType]; exists {
			return score
		}
	}

	return 0.6 // Default score
}

func (sr *suggestionRankerImpl) getDayOfWeekScore(suggestionType entities.SuggestionType, dayOfWeek string) float64 {
	// Simple heuristic - weekdays vs weekends
	if dayOfWeek == "Saturday" || dayOfWeek == "Sunday" {
		switch suggestionType {
		case entities.SuggestionTypeBreak, entities.SuggestionTypeLearning:
			return 0.8
		default:
			return 0.5
		}
	}

	return 0.7 // Weekday default
}

func (sr *suggestionRankerImpl) getWorkingHoursScore(suggestionType entities.SuggestionType, workingHours *entities.WorkingHours) float64 {
	// Check if current time is within working hours
	now := time.Now()
	currentTime := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())

	if currentTime >= workingHours.StartTime && currentTime <= workingHours.EndTime {
		// Within working hours
		switch suggestionType {
		case entities.SuggestionTypeNext, entities.SuggestionTypePattern, entities.SuggestionTypeWorkflow:
			return 0.9
		case entities.SuggestionTypeBreak:
			return 0.6
		default:
			return 0.7
		}
	} else {
		// Outside working hours
		switch suggestionType {
		case entities.SuggestionTypeBreak, entities.SuggestionTypeLearning:
			return 0.8
		default:
			return 0.4
		}
	}
}

func (sr *suggestionRankerImpl) getSessionDurationScore(suggestion *entities.TaskSuggestion, workContext *entities.WorkContext) float64 {
	if workContext.CurrentSession == nil {
		return 0.6
	}

	sessionHours := workContext.CurrentSession.Duration.Hours()

	switch suggestion.Type {
	case entities.SuggestionTypeBreak:
		if sessionHours > 2 {
			return 0.9
		} else if sessionHours > 1 {
			return 0.6
		} else {
			return 0.3
		}
	case entities.SuggestionTypeNext:
		if sessionHours < 0.5 {
			return 0.9 // Good to start with next tasks
		} else if sessionHours > 4 {
			return 0.4 // Been working too long
		} else {
			return 0.7
		}
	default:
		return 0.6
	}
}

// Additional helper methods (simplified implementations)

func (sr *suggestionRankerImpl) calculateContextRichness(workContext *entities.WorkContext) float64 {
	richness := 0.0
	richness += float64(len(workContext.CurrentTasks)) / 10.0
	richness += float64(len(workContext.RecentTasks)) / 20.0
	richness += float64(len(workContext.ActivePatterns)) / 5.0
	return math.Min(richness, 1.0)
}

func (sr *suggestionRankerImpl) getHistoricalConfidence(suggestionType entities.SuggestionType, repository string) float64 {
	// Placeholder - would query historical success rates
	return 0.7
}

func (sr *suggestionRankerImpl) getPersonalizationProfile(repository string) *PersonalizationProfile {
	if profile, exists := sr.personalizationModel[repository]; exists {
		return profile
	}
	return nil
}

func (sr *suggestionRankerImpl) estimateSuggestionComplexity(suggestion *entities.TaskSuggestion) float64 {
	// Simple heuristic based on content length and type
	complexity := 0.5

	if len(suggestion.Content) > 100 {
		complexity += 0.2
	}

	switch suggestion.Type {
	case entities.SuggestionTypeLearning, entities.SuggestionTypeWorkflow:
		complexity += 0.2
	case entities.SuggestionTypeBreak:
		complexity -= 0.3
	}

	return math.Max(0.0, math.Min(complexity, 1.0))
}

func (sr *suggestionRankerImpl) calculateDeadlineUrgency(suggestion *entities.TaskSuggestion, workContext *entities.WorkContext) float64 {
	// Check related tasks for deadlines
	// Note: Task entity doesn't have DueDate field, so we'll use a simplified urgency calculation
	for _, taskID := range suggestion.RelatedTaskIDs {
		for _, task := range workContext.CurrentTasks {
			if task.ID == taskID {
				// Use task creation time and priority as urgency indicators
				age := time.Since(task.CreatedAt)
				if string(task.Priority) == "high" && age > 24*time.Hour {
					return 0.9
				} else if string(task.Priority) == "medium" && age > 3*24*time.Hour {
					return 0.6
				} else if age > 7*24*time.Hour {
					return 0.3
				}
			}
		}
	}
	return 0.0
}

func (sr *suggestionRankerImpl) checkGoalAlignment(suggestion *entities.TaskSuggestion, workContext *entities.WorkContext) bool {
	for _, goal := range workContext.Goals {
		alignment := sr.calculateGoalAlignment(suggestion, goal)
		if alignment > 0.6 {
			return true
		}
	}
	return false
}

func (sr *suggestionRankerImpl) wasRecentlyRejected(suggestion *entities.TaskSuggestion, repository string) bool {
	// Placeholder - would check recent feedback history
	return false
}

func (sr *suggestionRankerImpl) isRepetitive(suggestion *entities.TaskSuggestion, repository string) bool {
	// Placeholder - would check for similar recent suggestions
	return false
}

func (sr *suggestionRankerImpl) calculateAcceptanceRate(feedback []*entities.SuggestionFeedback) float64 {
	if len(feedback) == 0 {
		return 0.5
	}

	accepted := 0
	for _, fb := range feedback {
		if fb.Accepted {
			accepted++
		}
	}

	return float64(accepted) / float64(len(feedback))
}

func (sr *suggestionRankerImpl) normalizeWeights(criteria *RankingCriteria) {
	total := criteria.RelevanceWeight + criteria.UrgencyWeight +
		criteria.ConfidenceWeight + criteria.PersonalizationWeight

	if total > 0 {
		criteria.RelevanceWeight /= total
		criteria.UrgencyWeight /= total
		criteria.ConfidenceWeight /= total
		criteria.PersonalizationWeight /= total
	}
}
