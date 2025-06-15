//go:build ignore

// Package services provides intelligent context analysis with ML-based insights.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// IntelligentContextAnalyzer provides ML-enhanced context analysis
type IntelligentContextAnalyzer struct {
	baseAnalyzer ContextAnalyzer
	mcpClient    ports.MCPClient
	aiService    ports.AIService
	logger       *slog.Logger
	config       *ContextAnalysisConfig
}

// ContextAnalysisConfig contains configuration for intelligent context analysis
type ContextAnalysisConfig struct {
	EnablePatternLearning    bool    `mapstructure:"enable_pattern_learning"`
	EnableProductivityML     bool    `mapstructure:"enable_productivity_ml"`
	EnableBehaviorAnalysis   bool    `mapstructure:"enable_behavior_analysis"`
	EnablePredictiveAnalysis bool    `mapstructure:"enable_predictive_analysis"`
	LearningWindowDays       int     `mapstructure:"learning_window_days"`
	MinPatternConfidence     float64 `mapstructure:"min_pattern_confidence"`
	PersonalizationLevel     string  `mapstructure:"personalization_level"` // basic, moderate, advanced
}

// ProductivityInsight represents an ML-derived insight about productivity
type ProductivityInsight struct {
	Type            string             `json:"type"`
	Description     string             `json:"description"`
	Confidence      float64            `json:"confidence"`
	Impact          float64            `json:"impact"`
	Timeframe       string             `json:"timeframe"`
	Recommendations []string           `json:"recommendations"`
	Metrics         map[string]float64 `json:"metrics"`
	Evidence        []string           `json:"evidence"`
}

// BehaviorPattern represents a learned behavior pattern
type BehaviorPattern struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // productivity, energy, focus, timing
	Description string                 `json:"description"`
	Confidence  float64                `json:"confidence"`
	Frequency   float64                `json:"frequency"`
	Conditions  map[string]interface{} `json:"conditions"`
	Outcomes    map[string]float64     `json:"outcomes"`
	LastSeen    time.Time              `json:"last_seen"`
	Strength    float64                `json:"strength"`
	Predictive  bool                   `json:"predictive"`
	Actionable  bool                   `json:"actionable"`
}

// ContextPrediction represents a prediction about future context
type ContextPrediction struct {
	Timeframe        string             `json:"timeframe"`
	PredictedMetrics map[string]float64 `json:"predicted_metrics"`
	Confidence       float64            `json:"confidence"`
	Reasoning        string             `json:"reasoning"`
	Recommendations  []string           `json:"recommendations"`
	RiskFactors      []string           `json:"risk_factors"`
}

// NewIntelligentContextAnalyzer creates a new intelligent context analyzer
func NewIntelligentContextAnalyzer(
	baseAnalyzer ContextAnalyzer,
	mcpClient ports.MCPClient,
	aiService ports.AIService,
	logger *slog.Logger,
) *IntelligentContextAnalyzer {
	return &IntelligentContextAnalyzer{
		baseAnalyzer: baseAnalyzer,
		mcpClient:    mcpClient,
		aiService:    aiService,
		logger:       logger,
		config:       getDefaultContextAnalysisConfig(),
	}
}

// AnalyzeContext performs enhanced context analysis with ML insights
func (ica *IntelligentContextAnalyzer) AnalyzeContext(ctx context.Context, workContext *entities.WorkContext) (*entities.ContextAnalysis, error) {
	// Get base analysis first
	baseAnalysis, err := ica.baseAnalyzer.AnalyzeContext(ctx, workContext)
	if err != nil {
		return nil, fmt.Errorf("base context analysis failed: %w", err)
	}

	// Enhance with ML-based insights
	if ica.config.EnablePatternLearning {
		patterns, err := ica.learnAndDetectPatterns(ctx, workContext)
		if err != nil {
			ica.logger.Warn("pattern learning failed", slog.String("error", err.Error()))
		} else {
			baseAnalysis.BehaviorPatterns = ica.convertBehaviorPatterns(patterns)
		}
	}

	if ica.config.EnableProductivityML {
		insights, err := ica.generateProductivityInsights(ctx, workContext)
		if err != nil {
			ica.logger.Warn("productivity ML analysis failed", slog.String("error", err.Error()))
		} else {
			baseAnalysis.MLInsights = ica.convertProductivityInsights(insights)
		}
	}

	if ica.config.EnablePredictiveAnalysis {
		predictions, err := ica.generateContextPredictions(ctx, workContext)
		if err != nil {
			ica.logger.Warn("predictive analysis failed", slog.String("error", err.Error()))
		} else {
			baseAnalysis.Predictions = ica.convertPredictions(predictions)
		}
	}

	// Enhance existing metrics with ML adjustments
	ica.enhanceMetricsWithML(baseAnalysis, workContext)

	ica.logger.Info("completed intelligent context analysis",
		slog.String("repository", workContext.Repository),
		slog.Int("patterns", len(baseAnalysis.BehaviorPatterns)),
		slog.Int("insights", len(baseAnalysis.MLInsights)))

	return baseAnalysis, nil
}

// learnAndDetectPatterns uses ML to learn and detect behavior patterns
func (ica *IntelligentContextAnalyzer) learnAndDetectPatterns(ctx context.Context, workContext *entities.WorkContext) ([]*BehaviorPattern, error) {
	// Get historical data from MCP memory
	searchRequest := &ports.MemorySearchRequest{
		Query:      "context analysis behavior patterns productivity",
		Repository: workContext.Repository,
		Options: map[string]interface{}{
			"type":        "context_data",
			"max_results": 100,
			"time_window": fmt.Sprintf("%dd", ica.config.LearningWindowDays),
		},
	}

	searchResponse, err := ica.mcpClient.SearchMemory(ctx, searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve historical context data: %w", err)
	}

	// Use AI to analyze patterns in historical data
	patternPrompt := ica.buildPatternAnalysisPrompt(searchResponse.Results, workContext)

	aiRequest := &ports.AIAnalysisRequest{
		Type:    "pattern_detection",
		Content: patternPrompt,
		Context: map[string]interface{}{
			"repository":      workContext.Repository,
			"analysis_type":   "behavior_patterns",
			"learning_window": ica.config.LearningWindowDays,
		},
		Options: map[string]interface{}{
			"pattern_types":   []string{"productivity", "energy", "focus", "timing", "task_switching"},
			"min_confidence":  ica.config.MinPatternConfidence,
			"personalization": ica.config.PersonalizationLevel,
		},
	}

	aiResponse, err := ica.aiService.AnalyzeWithAI(ctx, aiRequest)
	if err != nil {
		return nil, fmt.Errorf("AI pattern analysis failed: %w", err)
	}

	return ica.parsePatternResponse(aiResponse)
}

// generateProductivityInsights generates ML-based productivity insights
func (ica *IntelligentContextAnalyzer) generateProductivityInsights(ctx context.Context, workContext *entities.WorkContext) ([]*ProductivityInsight, error) {
	// Analyze productivity trends and factors
	insightPrompt := ica.buildProductivityInsightPrompt(workContext)

	aiRequest := &ports.AIAnalysisRequest{
		Type:    "productivity_analysis",
		Content: insightPrompt,
		Context: map[string]interface{}{
			"repository":    workContext.Repository,
			"analysis_type": "productivity_insights",
		},
		Options: map[string]interface{}{
			"insight_types": []string{"performance", "optimization", "blockers", "enhancers"},
			"ml_analysis":   true,
		},
	}

	aiResponse, err := ica.aiService.AnalyzeWithAI(ctx, aiRequest)
	if err != nil {
		return nil, fmt.Errorf("AI productivity analysis failed: %w", err)
	}

	return ica.parseProductivityInsights(aiResponse)
}

// generateContextPredictions generates predictive context analysis
func (ica *IntelligentContextAnalyzer) generateContextPredictions(ctx context.Context, workContext *entities.WorkContext) ([]*ContextPrediction, error) {
	predictionPrompt := ica.buildPredictionPrompt(workContext)

	aiRequest := &ports.AIAnalysisRequest{
		Type:    "context_prediction",
		Content: predictionPrompt,
		Context: map[string]interface{}{
			"repository":       workContext.Repository,
			"analysis_type":    "predictive",
			"prediction_scope": []string{"1hour", "4hours", "1day"},
		},
		Options: map[string]interface{}{
			"prediction_types": []string{"energy", "focus", "productivity", "stress"},
			"confidence_level": "medium",
		},
	}

	aiResponse, err := ica.aiService.AnalyzeWithAI(ctx, aiRequest)
	if err != nil {
		return nil, fmt.Errorf("AI prediction analysis failed: %w", err)
	}

	return ica.parsePredictionResponse(aiResponse)
}

// enhanceMetricsWithML applies ML adjustments to base metrics
func (ica *IntelligentContextAnalyzer) enhanceMetricsWithML(analysis *entities.ContextAnalysis, workContext *entities.WorkContext) {
	// Apply learning-based adjustments to productivity score
	if len(analysis.BehaviorPatterns) > 0 {
		productivityAdjustment := ica.calculateMLProductivityAdjustment(analysis.BehaviorPatterns, workContext)
		analysis.ProductivityScore = math.Min(1.0, math.Max(0.0, analysis.ProductivityScore+productivityAdjustment))
	}

	// Apply pattern-based focus adjustments
	focusAdjustment := ica.calculateMLFocusAdjustment(analysis.BehaviorPatterns, workContext)
	analysis.FocusLevel = math.Min(1.0, math.Max(0.0, analysis.FocusLevel+focusAdjustment))

	// Apply energy pattern adjustments
	energyAdjustment := ica.calculateMLEnergyAdjustment(analysis.BehaviorPatterns, workContext)
	analysis.EnergyLevel = math.Min(1.0, math.Max(0.0, analysis.EnergyLevel+energyAdjustment))
}

// Prompt building methods

func (ica *IntelligentContextAnalyzer) buildPatternAnalysisPrompt(historicalData []*ports.MemorySearchResult, workContext *entities.WorkContext) string {
	prompt := fmt.Sprintf(`Analyze behavior patterns from historical context data:

Current Context:
- Energy: %.1f%%
- Focus: %.1f%%
- Productivity: %.1f%%
- Active Tasks: %d
- Time: %s

Historical Data Points: %d records

`, workContext.EnergyLevel*100, workContext.FocusLevel*100,
		workContext.ProductivityScore*100, len(workContext.CurrentTasks),
		time.Now().Format("15:04"), len(historicalData))

	// Add sample of historical data
	for i, data := range historicalData {
		if i < 5 { // Limit to 5 samples for context
			prompt += fmt.Sprintf("- %s (Confidence: %.1f%%)\n", data.Title, data.Confidence*100)
		}
	}

	prompt += `
Identify behavior patterns that:
1. **Productivity Patterns**: Times/conditions when productivity is highest/lowest
2. **Energy Patterns**: Natural energy rhythms and what affects them
3. **Focus Patterns**: When focus is optimal and what disrupts it
4. **Task Switching Patterns**: How task transitions affect performance
5. **Timing Patterns**: Optimal timing for different types of work

For each pattern, provide:
- Pattern name and type
- Confidence level (0.0-1.0)
- Frequency of occurrence
- Conditions that trigger the pattern
- Measurable outcomes/impacts
- Whether it's predictive and actionable

Return as JSON:
{
	"patterns": [
		{
			"id": "unique_id",
			"name": "pattern name",
			"type": "productivity|energy|focus|timing|task_switching",
			"description": "detailed description",
			"confidence": 0.0-1.0,
			"frequency": 0.0-1.0,
			"conditions": {"key": "value"},
			"outcomes": {"metric": value},
			"strength": 0.0-1.0,
			"predictive": true/false,
			"actionable": true/false
		}
	]
}`

	return prompt
}

func (ica *IntelligentContextAnalyzer) buildProductivityInsightPrompt(workContext *entities.WorkContext) string {
	return fmt.Sprintf(`Generate ML-based productivity insights:

Current Metrics:
- Productivity Score: %.1f%%
- Energy Level: %.1f%%
- Focus Level: %.1f%%
- Task Completion Rate: %.1f%%
- Stress Indicators: %d

Context Factors:
- Time of Day: %s
- Active Tasks: %d
- Completed Today: %d
- Session Duration: %s

Analyze and provide insights on:
1. **Performance Drivers**: What's currently helping/hurting productivity
2. **Optimization Opportunities**: Specific areas for improvement
3. **Productivity Blockers**: Factors limiting performance
4. **Performance Enhancers**: What's working well to amplify

Return as JSON:
{
	"insights": [
		{
			"type": "performance|optimization|blocker|enhancer",
			"description": "insight description",
			"confidence": 0.0-1.0,
			"impact": 0.0-1.0,
			"timeframe": "immediate|short|medium|long",
			"recommendations": ["action1", "action2"],
			"metrics": {"metric": value},
			"evidence": ["evidence1", "evidence2"]
		}
	]
}`, workContext.ProductivityScore*100, workContext.EnergyLevel*100,
		workContext.FocusLevel*100, workContext.GetTaskCompletionRate()*100,
		len(workContext.StressIndicators), time.Now().Format("15:04"),
		len(workContext.CurrentTasks), workContext.GetCompletedTasksToday(),
		workContext.GetSessionDuration())
}

func (ica *IntelligentContextAnalyzer) buildPredictionPrompt(workContext *entities.WorkContext) string {
	return fmt.Sprintf(`Generate context predictions based on current state:

Current State:
- Energy: %.1f%% (trend: %s)
- Focus: %.1f%% (trend: %s) 
- Productivity: %.1f%% (trend: %s)
- Time: %s
- Session Duration: %s

Predict context for next:
1. **1 Hour**: How will metrics change in the next hour
2. **4 Hours**: Medium-term predictions
3. **1 Day**: End-of-day predictions

Consider factors:
- Natural circadian rhythms
- Work session fatigue
- Task completion momentum
- Stress accumulation
- Energy depletion patterns

Return as JSON:
{
	"predictions": [
		{
			"timeframe": "1hour|4hours|1day",
			"predicted_metrics": {
				"energy": 0.0-1.0,
				"focus": 0.0-1.0,
				"productivity": 0.0-1.0
			},
			"confidence": 0.0-1.0,
			"reasoning": "why these predictions",
			"recommendations": ["action1", "action2"],
			"risk_factors": ["risk1", "risk2"]
		}
	]
}`, workContext.EnergyLevel*100, ica.getTrend(workContext, "energy"),
		workContext.FocusLevel*100, ica.getTrend(workContext, "focus"),
		workContext.ProductivityScore*100, ica.getTrend(workContext, "productivity"),
		time.Now().Format("15:04"), workContext.GetSessionDuration())
}

// Parsing methods

func (ica *IntelligentContextAnalyzer) parsePatternResponse(response *ports.AIAnalysisResponse) ([]*BehaviorPattern, error) {
	var result struct {
		Patterns []*BehaviorPattern `json:"patterns"`
	}

	if err := json.Unmarshal([]byte(response.Analysis), &result); err != nil {
		return nil, fmt.Errorf("failed to parse pattern response: %w", err)
	}

	// Filter by confidence threshold
	var validPatterns []*BehaviorPattern
	for _, pattern := range result.Patterns {
		if pattern.Confidence >= ica.config.MinPatternConfidence {
			pattern.LastSeen = time.Now()
			validPatterns = append(validPatterns, pattern)
		}
	}

	// Sort by confidence
	sort.Slice(validPatterns, func(i, j int) bool {
		return validPatterns[i].Confidence > validPatterns[j].Confidence
	})

	return validPatterns, nil
}

func (ica *IntelligentContextAnalyzer) parseProductivityInsights(response *ports.AIAnalysisResponse) ([]*ProductivityInsight, error) {
	var result struct {
		Insights []*ProductivityInsight `json:"insights"`
	}

	if err := json.Unmarshal([]byte(response.Analysis), &result); err != nil {
		return nil, fmt.Errorf("failed to parse productivity insights: %w", err)
	}

	return result.Insights, nil
}

func (ica *IntelligentContextAnalyzer) parsePredictionResponse(response *ports.AIAnalysisResponse) ([]*ContextPrediction, error) {
	var result struct {
		Predictions []*ContextPrediction `json:"predictions"`
	}

	if err := json.Unmarshal([]byte(response.Analysis), &result); err != nil {
		return nil, fmt.Errorf("failed to parse prediction response: %w", err)
	}

	return result.Predictions, nil
}

// Conversion methods

func (ica *IntelligentContextAnalyzer) convertBehaviorPatterns(patterns []*BehaviorPattern) []*entities.Pattern {
	entityPatterns := make([]*entities.Pattern, 0, len(patterns))

	for _, pattern := range patterns {
		entityPattern := &entities.Pattern{
			ID:          pattern.ID,
			Name:        pattern.Name,
			Type:        pattern.Type,
			Description: pattern.Description,
			Confidence:  pattern.Confidence,
			Frequency:   pattern.Frequency,
			LastSeen:    pattern.LastSeen,
			Metadata: map[string]interface{}{
				"strength":   pattern.Strength,
				"predictive": pattern.Predictive,
				"actionable": pattern.Actionable,
				"conditions": pattern.Conditions,
				"outcomes":   pattern.Outcomes,
			},
		}
		entityPatterns = append(entityPatterns, entityPattern)
	}

	return entityPatterns
}

func (ica *IntelligentContextAnalyzer) convertProductivityInsights(insights []*ProductivityInsight) []map[string]interface{} {
	converted := make([]map[string]interface{}, 0, len(insights))

	for _, insight := range insights {
		converted = append(converted, map[string]interface{}{
			"type":            insight.Type,
			"description":     insight.Description,
			"confidence":      insight.Confidence,
			"impact":          insight.Impact,
			"timeframe":       insight.Timeframe,
			"recommendations": insight.Recommendations,
			"metrics":         insight.Metrics,
			"evidence":        insight.Evidence,
		})
	}

	return converted
}

func (ica *IntelligentContextAnalyzer) convertPredictions(predictions []*ContextPrediction) []map[string]interface{} {
	converted := make([]map[string]interface{}, 0, len(predictions))

	for _, prediction := range predictions {
		converted = append(converted, map[string]interface{}{
			"timeframe":         prediction.Timeframe,
			"predicted_metrics": prediction.PredictedMetrics,
			"confidence":        prediction.Confidence,
			"reasoning":         prediction.Reasoning,
			"recommendations":   prediction.Recommendations,
			"risk_factors":      prediction.RiskFactors,
		})
	}

	return converted
}

// ML adjustment calculations

func (ica *IntelligentContextAnalyzer) calculateMLProductivityAdjustment(patterns []*entities.Pattern, workContext *entities.WorkContext) float64 {
	adjustment := 0.0

	for _, pattern := range patterns {
		if pattern.Type == "productivity" && pattern.Confidence > 0.7 {
			// Apply pattern-based adjustments
			if outcomes, ok := pattern.Metadata["outcomes"].(map[string]float64); ok {
				if productivityImpact, exists := outcomes["productivity"]; exists {
					adjustment += productivityImpact * pattern.Confidence * 0.1 // Max 10% adjustment
				}
			}
		}
	}

	return math.Max(-0.2, math.Min(0.2, adjustment)) // Limit to ±20%
}

func (ica *IntelligentContextAnalyzer) calculateMLFocusAdjustment(patterns []*entities.Pattern, workContext *entities.WorkContext) float64 {
	adjustment := 0.0

	for _, pattern := range patterns {
		if pattern.Type == "focus" && pattern.Confidence > 0.6 {
			if outcomes, ok := pattern.Metadata["outcomes"].(map[string]float64); ok {
				if focusImpact, exists := outcomes["focus"]; exists {
					adjustment += focusImpact * pattern.Confidence * 0.05 // Max 5% adjustment
				}
			}
		}
	}

	return math.Max(-0.15, math.Min(0.15, adjustment)) // Limit to ±15%
}

func (ica *IntelligentContextAnalyzer) calculateMLEnergyAdjustment(patterns []*entities.Pattern, workContext *entities.WorkContext) float64 {
	adjustment := 0.0

	for _, pattern := range patterns {
		if pattern.Type == "energy" && pattern.Confidence > 0.6 {
			if outcomes, ok := pattern.Metadata["outcomes"].(map[string]float64); ok {
				if energyImpact, exists := outcomes["energy"]; exists {
					adjustment += energyImpact * pattern.Confidence * 0.05 // Max 5% adjustment
				}
			}
		}
	}

	return math.Max(-0.15, math.Min(0.15, adjustment)) // Limit to ±15%
}

// Helper methods

func (ica *IntelligentContextAnalyzer) getTrend(workContext *entities.WorkContext, metric string) string {
	// Simplified trend calculation - in real implementation would analyze historical data
	switch metric {
	case "energy":
		if workContext.EnergyLevel > 0.7 {
			return "stable"
		} else if workContext.EnergyLevel < 0.4 {
			return "declining"
		}
		return "moderate"
	case "focus":
		if workContext.FocusLevel > 0.8 {
			return "high"
		} else if workContext.FocusLevel < 0.5 {
			return "declining"
		}
		return "stable"
	case "productivity":
		if workContext.ProductivityScore > 0.8 {
			return "increasing"
		} else if workContext.ProductivityScore < 0.5 {
			return "declining"
		}
		return "stable"
	default:
		return "unknown"
	}
}

func getDefaultContextAnalysisConfig() *ContextAnalysisConfig {
	return &ContextAnalysisConfig{
		EnablePatternLearning:    true,
		EnableProductivityML:     true,
		EnableBehaviorAnalysis:   true,
		EnablePredictiveAnalysis: true,
		LearningWindowDays:       30,
		MinPatternConfidence:     0.6,
		PersonalizationLevel:     "moderate",
	}
}
