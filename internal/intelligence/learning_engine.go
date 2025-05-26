package intelligence

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"mcp-memory/pkg/types"
)

// LearningStrategy represents different learning approaches
type LearningStrategy string

const (
	StrategyReinforcement LearningStrategy = "reinforcement"
	StrategyPatternBased  LearningStrategy = "pattern_based"
	StrategyFeedbackBased LearningStrategy = "feedback_based"
	StrategyUsageBased    LearningStrategy = "usage_based"
	StrategyContextual    LearningStrategy = "contextual"
)

// LearningMetric represents metrics for measuring learning effectiveness
type LearningMetric struct {
	Name           string    `json:"name"`
	Value          float64   `json:"value"`
	Trend          string    `json:"trend"` // "improving", "declining", "stable"
	LastUpdated    time.Time `json:"last_updated"`
	MeasurementCount int     `json:"measurement_count"`
}

// LearningObjective represents what the system should learn
type LearningObjective struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	TargetMetric string                `json:"target_metric"`
	TargetValue float64                `json:"target_value"`
	Priority    int                    `json:"priority"` // 1-10
	Strategy    LearningStrategy       `json:"strategy"`
	Context     map[string]any         `json:"context"`
	CreatedAt   time.Time              `json:"created_at"`
	Progress    float64                `json:"progress"` // 0-1
	IsActive    bool                   `json:"is_active"`
}

// LearningEvent represents an event that can be learned from
type LearningEvent struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Context      map[string]any         `json:"context"`
	Outcome      string                 `json:"outcome"`
	Success      bool                   `json:"success"`
	Feedback     *UserFeedback          `json:"feedback,omitempty"`
	Metrics      map[string]float64     `json:"metrics"`
	Timestamp    time.Time              `json:"timestamp"`
	ChunkIDs     []string               `json:"chunk_ids"`
}

// UserFeedback represents explicit feedback from users
type UserFeedback struct {
	Rating      int                    `json:"rating"`      // 1-5
	Comments    string                 `json:"comments"`
	Helpful     bool                   `json:"helpful"`
	Accurate    bool                   `json:"accurate"`
	Relevant    bool                   `json:"relevant"`
	Suggestions []string               `json:"suggestions"`
	Context     map[string]any         `json:"context"`
	Timestamp   time.Time              `json:"timestamp"`
}

// AdaptationRule represents a rule that modifies behavior based on learning
type AdaptationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Condition   string                 `json:"condition"`
	Action      string                 `json:"action"`
	Parameters  map[string]any         `json:"parameters"`
	Priority    int                    `json:"priority"`
	IsActive    bool                   `json:"is_active"`
	SuccessRate float64                `json:"success_rate"`
	UsageCount  int                    `json:"usage_count"`
	CreatedAt   time.Time              `json:"created_at"`
	LastUsed    time.Time              `json:"last_used"`
}

// LearningEngine coordinates learning and adaptation
type LearningEngine struct {
	patternEngine    *PatternEngine
	knowledgeGraph   *GraphBuilder
	
	// Learning components
	objectives       map[string]*LearningObjective
	adaptationRules  map[string]*AdaptationRule
	metrics          map[string]*LearningMetric
	events           []LearningEvent
	
	// Configuration
	maxEvents        int
	learningRate     float64
	adaptationThreshold float64
	feedbackWeight   float64
	
	// State
	isLearning       bool
	lastUpdate       time.Time
}

// LearningStorage interface for persisting learning data
type LearningStorage interface {
	StoreObjective(ctx context.Context, objective *LearningObjective) error
	GetObjective(ctx context.Context, id string) (*LearningObjective, error)
	ListObjectives(ctx context.Context) ([]*LearningObjective, error)
	
	StoreRule(ctx context.Context, rule *AdaptationRule) error
	GetRule(ctx context.Context, id string) (*AdaptationRule, error)
	ListRules(ctx context.Context) ([]*AdaptationRule, error)
	
	StoreEvent(ctx context.Context, event *LearningEvent) error
	GetRecentEvents(ctx context.Context, limit int) ([]LearningEvent, error)
	
	StoreMetric(ctx context.Context, metric *LearningMetric) error
	GetMetric(ctx context.Context, name string) (*LearningMetric, error)
	ListMetrics(ctx context.Context) ([]*LearningMetric, error)
}

// NewLearningEngine creates a new learning engine
func NewLearningEngine(patternEngine *PatternEngine, knowledgeGraph *GraphBuilder) *LearningEngine {
	engine := &LearningEngine{
		patternEngine:       patternEngine,
		knowledgeGraph:      knowledgeGraph,
		objectives:          make(map[string]*LearningObjective),
		adaptationRules:     make(map[string]*AdaptationRule),
		metrics:             make(map[string]*LearningMetric),
		events:              make([]LearningEvent, 0),
		maxEvents:           1000,
		learningRate:        0.1,
		adaptationThreshold: 0.7,
		feedbackWeight:      0.3,
		isLearning:          true,
		lastUpdate:          time.Now(),
	}
	
	// Initialize default objectives and rules
	engine.initializeDefaults()
	
	return engine
}

// LearnFromConversation learns from a conversation interaction
func (le *LearningEngine) LearnFromConversation(ctx context.Context, chunks []types.ConversationChunk, outcome string, feedback *UserFeedback) error {
	if !le.isLearning || len(chunks) == 0 {
		return nil
	}
	
	// Create learning event
	event := LearningEvent{
		ID:        fmt.Sprintf("conv_%d", time.Now().UnixNano()),
		Type:      "conversation",
		Context:   le.extractConversationContext(chunks),
		Outcome:   outcome,
		Success:   le.isSuccessfulOutcome(outcome),
		Feedback:  feedback,
		Metrics:   le.calculateConversationMetrics(chunks),
		Timestamp: time.Now(),
		ChunkIDs:  extractChunkIDs(chunks),
	}
	
	// Store event
	le.addEvent(event)
	
	// Learn patterns
	err := le.learnPatterns(ctx, chunks, event)
	if err != nil {
		return fmt.Errorf("failed to learn patterns: %w", err)
	}
	
	// Update metrics
	le.updateMetrics(event)
	
	// Adapt behavior based on learning
	err = le.adaptBehavior(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to adapt behavior: %w", err)
	}
	
	// Update objectives progress
	le.updateObjectiveProgress()
	
	le.lastUpdate = time.Now()
	return nil
}

// LearnFromFeedback learns from explicit user feedback
func (le *LearningEngine) LearnFromFeedback(ctx context.Context, chunkID string, feedback *UserFeedback) error {
	if !le.isLearning || feedback == nil {
		return nil
	}
	
	event := LearningEvent{
		ID:        fmt.Sprintf("feedback_%d", time.Now().UnixNano()),
		Type:      "feedback",
		Context:   map[string]any{"chunk_id": chunkID},
		Outcome:   le.feedbackToOutcome(feedback),
		Success:   feedback.Helpful && feedback.Accurate,
		Feedback:  feedback,
		Metrics:   le.calculateFeedbackMetrics(feedback),
		Timestamp: time.Now(),
		ChunkIDs:  []string{chunkID},
	}
	
	le.addEvent(event)
	
	// Adjust confidence and weights based on feedback
	le.adjustFromFeedback(ctx, chunkID, feedback)
	
	return nil
}

// GetAdaptationRecommendations returns recommendations for behavior adaptation
func (le *LearningEngine) GetAdaptationRecommendations(ctx context.Context) ([]AdaptationRule, error) {
	var recommendations []AdaptationRule
	
	// Analyze recent performance
	recentEvents := le.getRecentEvents(50)
	performance := le.analyzePerformance(recentEvents)
	
	// Generate recommendations based on performance
	if performance.successRate < 0.7 {
		recommendations = append(recommendations, AdaptationRule{
			ID:          "improve_accuracy",
			Name:        "Improve Answer Accuracy",
			Condition:   "success_rate < 0.7",
			Action:      "increase_confidence_threshold",
			Parameters:  map[string]any{"threshold": 0.8},
			Priority:    8,
			IsActive:    true,
			CreatedAt:   time.Now(),
		})
	}
	
	if performance.avgResponseTime > 5.0 {
		recommendations = append(recommendations, AdaptationRule{
			ID:          "improve_speed",
			Name:        "Improve Response Speed",
			Condition:   "avg_response_time > 5.0",
			Action:      "optimize_search_params",
			Parameters:  map[string]any{"max_results": 5},
			Priority:    6,
			IsActive:    true,
			CreatedAt:   time.Now(),
		})
	}
	
	// Check if patterns are being learned effectively
	if performance.patternUtilization < 0.5 {
		recommendations = append(recommendations, AdaptationRule{
			ID:          "enhance_pattern_learning",
			Name:        "Enhance Pattern Learning",
			Condition:   "pattern_utilization < 0.5",
			Action:      "increase_pattern_sensitivity",
			Parameters:  map[string]any{"min_confidence": 0.5},
			Priority:    7,
			IsActive:    true,
			CreatedAt:   time.Now(),
		})
	}
	
	return recommendations, nil
}

// GetLearningStats returns current learning statistics
func (le *LearningEngine) GetLearningStats() map[string]any {
	stats := make(map[string]any)
	
	stats["is_learning"] = le.isLearning
	stats["last_update"] = le.lastUpdate
	stats["total_events"] = len(le.events)
	stats["total_objectives"] = len(le.objectives)
	stats["total_rules"] = len(le.adaptationRules)
	stats["total_metrics"] = len(le.metrics)
	
	// Calculate success rate
	recentEvents := le.getRecentEvents(100)
	performance := le.analyzePerformance(recentEvents)
	stats["success_rate"] = performance.successRate
	stats["avg_response_time"] = performance.avgResponseTime
	stats["pattern_utilization"] = performance.patternUtilization
	
	// Objective progress
	var totalProgress float64
	activeObjectives := 0
	for _, obj := range le.objectives {
		if obj.IsActive {
			totalProgress += obj.Progress
			activeObjectives++
		}
	}
	if activeObjectives > 0 {
		stats["avg_objective_progress"] = totalProgress / float64(activeObjectives)
	}
	
	return stats
}

// Helper methods

func (le *LearningEngine) initializeDefaults() {
	// Default learning objectives
	objectives := []*LearningObjective{
		{
			ID:           "improve_relevance",
			Type:         "suggestion_quality",
			Description:  "Improve relevance of context suggestions",
			TargetMetric: "suggestion_relevance",
			TargetValue:  0.85,
			Priority:     9,
			Strategy:     StrategyFeedbackBased,
			Context:      map[string]any{"component": "context_suggester"},
			CreatedAt:    time.Now(),
			Progress:     0.0,
			IsActive:     true,
		},
		{
			ID:           "pattern_recognition",
			Type:         "pattern_learning",
			Description:  "Improve pattern recognition accuracy",
			TargetMetric: "pattern_accuracy",
			TargetValue:  0.80,
			Priority:     8,
			Strategy:     StrategyPatternBased,
			Context:      map[string]any{"component": "pattern_engine"},
			CreatedAt:    time.Now(),
			Progress:     0.0,
			IsActive:     true,
		},
		{
			ID:           "response_speed",
			Type:         "performance",
			Description:  "Optimize response time",
			TargetMetric: "avg_response_time",
			TargetValue:  2.0,
			Priority:     6,
			Strategy:     StrategyUsageBased,
			Context:      map[string]any{"component": "system"},
			CreatedAt:    time.Now(),
			Progress:     0.0,
			IsActive:     true,
		},
	}
	
	for _, obj := range objectives {
		le.objectives[obj.ID] = obj
	}
	
	// Default adaptation rules
	rules := []*AdaptationRule{
		{
			ID:          "confidence_boost",
			Name:        "Boost Confidence for Successful Patterns",
			Condition:   "pattern_success_rate > 0.8",
			Action:      "increase_pattern_confidence",
			Parameters:  map[string]any{"boost": 0.1},
			Priority:    7,
			IsActive:    true,
			SuccessRate: 0.0,
			UsageCount:  0,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "reduce_noise",
			Name:        "Reduce Low-Quality Suggestions",
			Condition:   "suggestion_relevance < 0.3",
			Action:      "filter_suggestions",
			Parameters:  map[string]any{"min_relevance": 0.5},
			Priority:    8,
			IsActive:    true,
			SuccessRate: 0.0,
			UsageCount:  0,
			CreatedAt:   time.Now(),
		},
	}
	
	for _, rule := range rules {
		le.adaptationRules[rule.ID] = rule
	}
}

func (le *LearningEngine) extractConversationContext(chunks []types.ConversationChunk) map[string]any {
	context := make(map[string]any)
	
	context["chunk_count"] = len(chunks)
	context["has_code"] = le.containsCode(chunks)
	context["has_errors"] = le.containsErrors(chunks)
	context["conversation_type"] = le.inferConversationType(chunks)
	
	if len(chunks) > 0 {
		context["start_time"] = chunks[0].Timestamp
		context["end_time"] = chunks[len(chunks)-1].Timestamp
		context["duration"] = chunks[len(chunks)-1].Timestamp.Sub(chunks[0].Timestamp).Seconds()
	}
	
	return context
}

func (le *LearningEngine) isSuccessfulOutcome(outcome string) bool {
	successOutcomes := map[string]bool{
		"success":    true,
		"completed":  true,
		"resolved":   true,
		"fixed":      true,
		"helpful":    true,
		"accurate":   true,
	}
	
	return successOutcomes[outcome]
}

func (le *LearningEngine) calculateConversationMetrics(chunks []types.ConversationChunk) map[string]float64 {
	metrics := make(map[string]float64)
	
	metrics["chunk_count"] = float64(len(chunks))
	metrics["avg_chunk_length"] = le.calculateAvgChunkLength(chunks)
	metrics["code_density"] = le.calculateCodeDensity(chunks)
	metrics["error_density"] = le.calculateErrorDensity(chunks)
	
	if len(chunks) > 1 {
		duration := chunks[len(chunks)-1].Timestamp.Sub(chunks[0].Timestamp).Seconds()
		metrics["duration"] = duration
		metrics["chunks_per_minute"] = float64(len(chunks)) / (duration / 60.0)
	}
	
	return metrics
}

func (le *LearningEngine) addEvent(event LearningEvent) {
	le.events = append(le.events, event)
	
	// Limit events to maxEvents
	if len(le.events) > le.maxEvents {
		le.events = le.events[len(le.events)-le.maxEvents:]
	}
}

func (le *LearningEngine) learnPatterns(ctx context.Context, chunks []types.ConversationChunk, event LearningEvent) error {
	if le.patternEngine == nil {
		return nil
	}
	
	// Determine outcome for pattern learning
	var outcome PatternOutcome
	if event.Success {
		outcome = OutcomeSuccess
	} else {
		outcome = OutcomeFailure
	}
	
	return le.patternEngine.LearnPattern(ctx, chunks, outcome)
}

func (le *LearningEngine) updateMetrics(event LearningEvent) {
	// Update or create metrics based on the event
	for metricName, value := range event.Metrics {
		if existing, exists := le.metrics[metricName]; exists {
			// Update existing metric
			oldValue := existing.Value
			existing.Value = (existing.Value*float64(existing.MeasurementCount) + value) / float64(existing.MeasurementCount+1)
			existing.MeasurementCount++
			existing.LastUpdated = time.Now()
			
			// Determine trend
			switch {
			case existing.Value > oldValue*1.05:
				existing.Trend = "improving"
			case existing.Value < oldValue*0.95:
				existing.Trend = "declining"
			default:
				existing.Trend = "stable"
			}
		} else {
			// Create new metric
			le.metrics[metricName] = &LearningMetric{
				Name:             metricName,
				Value:            value,
				Trend:            "stable",
				LastUpdated:      time.Now(),
				MeasurementCount: 1,
			}
		}
	}
}

func (le *LearningEngine) adaptBehavior(ctx context.Context, event LearningEvent) error {
	// Apply adaptation rules based on the event
	for _, rule := range le.adaptationRules {
		if !rule.IsActive {
			continue
		}
		
		if le.ruleMatches(rule, event) {
			err := le.applyRule(ctx, rule, event)
			if err != nil {
				return err
			}
			
			rule.UsageCount++
			rule.LastUsed = time.Now()
		}
	}
	
	return nil
}

func (le *LearningEngine) updateObjectiveProgress() {
	for _, objective := range le.objectives {
		if !objective.IsActive {
			continue
		}
		
		if metric, exists := le.metrics[objective.TargetMetric]; exists {
			// Calculate progress toward target
			currentValue := metric.Value
			targetValue := objective.TargetValue
			
			// Assuming higher values are better (adjust as needed)
			progress := math.Min(currentValue/targetValue, 1.0)
			objective.Progress = progress
		}
	}
}

func (le *LearningEngine) feedbackToOutcome(feedback *UserFeedback) string {
	if feedback.Helpful && feedback.Accurate {
		return "success"
	}
	if feedback.Rating >= 4 {
		return "positive"
	}
	if feedback.Rating <= 2 {
		return "negative"
	}
	return "neutral"
}

func (le *LearningEngine) calculateFeedbackMetrics(feedback *UserFeedback) map[string]float64 {
	metrics := make(map[string]float64)
	
	metrics["rating"] = float64(feedback.Rating)
	if feedback.Helpful {
		metrics["helpfulness"] = 1.0
	} else {
		metrics["helpfulness"] = 0.0
	}
	if feedback.Accurate {
		metrics["accuracy"] = 1.0
	} else {
		metrics["accuracy"] = 0.0
	}
	if feedback.Relevant {
		metrics["relevance"] = 1.0
	} else {
		metrics["relevance"] = 0.0
	}
	
	return metrics
}

func (le *LearningEngine) adjustFromFeedback(_ context.Context, _ string, feedback *UserFeedback) {
	// Adjust pattern confidence based on feedback
	if le.patternEngine != nil && le.knowledgeGraph != nil {
		// Find patterns associated with this chunk
		// This would require integration with the graph to find related patterns
		// For now, we'll adjust general learning parameters
		
		if feedback.Helpful && feedback.Accurate {
			// Positive feedback - boost learning rate temporarily
			le.learningRate = math.Min(le.learningRate*1.1, 0.5)
		} else {
			// Negative feedback - be more conservative
			le.learningRate = math.Max(le.learningRate*0.9, 0.01)
		}
	}
}

// Performance analysis types and methods

type PerformanceAnalysis struct {
	successRate        float64
	avgResponseTime    float64
	patternUtilization float64
	eventCount         int
}

func (le *LearningEngine) getRecentEvents(limit int) []LearningEvent {
	if len(le.events) <= limit {
		return le.events
	}
	return le.events[len(le.events)-limit:]
}

func (le *LearningEngine) analyzePerformance(events []LearningEvent) PerformanceAnalysis {
	if len(events) == 0 {
		return PerformanceAnalysis{}
	}
	
	successCount := 0
	totalResponseTime := 0.0
	patternEvents := 0
	
	for _, event := range events {
		if event.Success {
			successCount++
		}
		
		if responseTime, exists := event.Metrics["duration"]; exists {
			totalResponseTime += responseTime
		}
		
		if event.Type == "pattern" {
			patternEvents++
		}
	}
	
	return PerformanceAnalysis{
		successRate:        float64(successCount) / float64(len(events)),
		avgResponseTime:    totalResponseTime / float64(len(events)),
		patternUtilization: float64(patternEvents) / float64(len(events)),
		eventCount:         len(events),
	}
}

func (le *LearningEngine) ruleMatches(rule *AdaptationRule, event LearningEvent) bool {
	// Simple rule matching - in practice, this would be more sophisticated
	switch rule.Condition {
	case "pattern_success_rate > 0.8":
		if rate, exists := event.Metrics["pattern_success_rate"]; exists {
			return rate > 0.8
		}
	case "suggestion_relevance < 0.3":
		if relevance, exists := event.Metrics["suggestion_relevance"]; exists {
			return relevance < 0.3
		}
	}
	
	return false
}

func (le *LearningEngine) applyRule(_ context.Context, rule *AdaptationRule, _ LearningEvent) error {
	// Apply the rule's action
	switch rule.Action {
	case "increase_pattern_confidence":
		if boost, exists := rule.Parameters["boost"].(float64); exists {
			// This would integrate with pattern engine to boost confidence
			_ = boost
		}
	case "filter_suggestions":
		if minRelevance, exists := rule.Parameters["min_relevance"].(float64); exists {
			// This would update suggestion filtering parameters
			_ = minRelevance
		}
	}
	
	return nil
}

// Utility methods

func (le *LearningEngine) containsCode(chunks []types.ConversationChunk) bool {
	for _, chunk := range chunks {
		if strings.Contains(chunk.Content, "```") {
			return true
		}
	}
	return false
}

func (le *LearningEngine) containsErrors(chunks []types.ConversationChunk) bool {
	for _, chunk := range chunks {
		content := strings.ToLower(chunk.Content)
		if strings.Contains(content, "error") || strings.Contains(content, "exception") {
			return true
		}
	}
	return false
}

func (le *LearningEngine) inferConversationType(chunks []types.ConversationChunk) string {
	if len(chunks) == 0 {
		return "unknown"
	}
	
	if le.containsErrors(chunks) {
		return "debugging"
	}
	if le.containsCode(chunks) {
		return "development"
	}
	
	// Look at chunk types
	problemCount := 0
	solutionCount := 0
	for _, chunk := range chunks {
		switch chunk.Type {
		case types.ChunkTypeProblem:
			problemCount++
		case types.ChunkTypeSolution:
			solutionCount++
		case types.ChunkTypeDiscussion, types.ChunkTypeArchitectureDecision, types.ChunkTypeQuestion:
			// These chunk types don't affect the categorization
		case types.ChunkTypeCodeChange, types.ChunkTypeSessionSummary, types.ChunkTypeAnalysis, types.ChunkTypeVerification:
			// These chunk types also don't affect the categorization
		default:
			// Unknown chunk types are ignored for categorization
		}
	}
	
	if problemCount > 0 && solutionCount > 0 {
		return "problem_solving"
	}
	
	return "general"
}

func (le *LearningEngine) calculateAvgChunkLength(chunks []types.ConversationChunk) float64 {
	if len(chunks) == 0 {
		return 0.0
	}
	
	totalLength := 0
	for _, chunk := range chunks {
		totalLength += len(chunk.Content)
	}
	
	return float64(totalLength) / float64(len(chunks))
}

func (le *LearningEngine) calculateCodeDensity(chunks []types.ConversationChunk) float64 {
	if len(chunks) == 0 {
		return 0.0
	}
	
	codeChunks := 0
	for _, chunk := range chunks {
		if strings.Contains(chunk.Content, "```") {
			codeChunks++
		}
	}
	
	return float64(codeChunks) / float64(len(chunks))
}

func (le *LearningEngine) calculateErrorDensity(chunks []types.ConversationChunk) float64 {
	if len(chunks) == 0 {
		return 0.0
	}
	
	errorChunks := 0
	for _, chunk := range chunks {
		if le.containsErrors([]types.ConversationChunk{chunk}) {
			errorChunks++
		}
	}
	
	return float64(errorChunks) / float64(len(chunks))
}

// Enable/Disable learning
func (le *LearningEngine) EnableLearning() {
	le.isLearning = true
}

func (le *LearningEngine) DisableLearning() {
	le.isLearning = false
}

func (le *LearningEngine) IsLearning() bool {
	return le.isLearning
}