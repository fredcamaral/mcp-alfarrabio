// Package ai provides AI-powered pattern detection for memory analysis
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/logging"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DetectedPattern represents a pattern detected in memories
type DetectedPattern struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`        // "problem_solving_approach", "decision_making", etc.
	Description string    `json:"description"` // Human-readable description
	Confidence  float64   `json:"confidence"`  // 0.0-1.0 confidence score
	Frequency   int       `json:"frequency"`   // How many times this pattern appeared
	Examples    []string  `json:"examples"`    // Example excerpts that show this pattern
	Tags        []string  `json:"tags"`        // Related keywords and concepts
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PatternAnalysis represents analysis metadata for a memory
type PatternAnalysis struct {
	Patterns   []*DetectedPattern `json:"patterns"`
	AnalyzedAt time.Time          `json:"analyzed_at"`
	Version    string             `json:"version"`
}

// PatternDetector uses AI to identify patterns across memories and conversations
type PatternDetector struct {
	aiService      AIService
	config         *config.Config
	logger         logging.Logger                // Use basic logger interface instead of EnhancedLogger
	patternHistory map[string][]*DetectedPattern // project_id -> patterns
}

// PatternDetectionResult contains comprehensive pattern analysis results
type PatternDetectionResult struct {
	ProjectID          string              `json:"project_id"`
	AnalysisID         string              `json:"analysis_id"`
	Patterns           []*DetectedPattern  `json:"patterns"`
	Trends             []*PatternTrend     `json:"trends"`
	Recommendations    []*Recommendation   `json:"recommendations"`
	QualityInsights    *QualityInsights    `json:"quality_insights"`
	KnowledgeGaps      []*KnowledgeGap     `json:"knowledge_gaps"`
	BehavioralInsights *BehavioralInsights `json:"behavioral_insights"`
	AnalyzedAt         time.Time           `json:"analyzed_at"`
	MemoryCount        int                 `json:"memory_count"`
	TimeSpan           time.Duration       `json:"time_span"`
}

// PatternTrend shows how patterns evolve over time
type PatternTrend struct {
	PatternType  string    `json:"pattern_type"`
	Direction    string    `json:"direction"`    // "increasing", "decreasing", "stable"
	Strength     float64   `json:"strength"`     // 0.0 to 1.0
	TimeFrame    string    `json:"time_frame"`   // "last_week", "last_month", "last_quarter"
	Significance string    `json:"significance"` // "high", "medium", "low"
	Description  string    `json:"description"`
	DataPoints   []float64 `json:"data_points"` // Historical values
}

// Recommendation provides AI-generated suggestions based on pattern analysis
type Recommendation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`     // "process_improvement", "knowledge_capture", "tool_usage"
	Priority    string    `json:"priority"` // "critical", "high", "medium", "low"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ActionItems []string  `json:"action_items"`
	Benefits    []string  `json:"benefits"`
	Effort      string    `json:"effort"`     // "low", "medium", "high"
	Impact      string    `json:"impact"`     // "low", "medium", "high"
	Confidence  float64   `json:"confidence"` // 0.0 to 1.0
	DueDate     time.Time `json:"due_date"`
}

// QualityInsights provides analysis of memory quality patterns
type QualityInsights struct {
	OverallQuality    float64            `json:"overall_quality"`    // 0.0 to 1.0
	QualityTrend      string             `json:"quality_trend"`      // "improving", "declining", "stable"
	HighQualityCount  int                `json:"high_quality_count"` // Memories with quality > 0.8
	LowQualityCount   int                `json:"low_quality_count"`  // Memories with quality < 0.5
	QualityFactors    map[string]float64 `json:"quality_factors"`    // completeness, clarity, etc.
	ImprovementAreas  []string           `json:"improvement_areas"`
	BestPractices     []string           `json:"best_practices"`
	QualityByCategory map[string]float64 `json:"quality_by_category"` // category -> avg quality
}

// KnowledgeGap represents missing or incomplete knowledge areas
type KnowledgeGap struct {
	ID          string   `json:"id"`
	Area        string   `json:"area"`     // "authentication", "database_design", etc.
	Severity    string   `json:"severity"` // "critical", "important", "minor"
	Description string   `json:"description"`
	Evidence    []string `json:"evidence"`    // What indicates this gap
	Suggestions []string `json:"suggestions"` // How to fill the gap
	Related     []string `json:"related"`     // Related concepts/areas
	Priority    int      `json:"priority"`    // 1 (highest) to 10 (lowest)
}

// BehavioralInsights analyzes patterns in user behavior and work habits
type BehavioralInsights struct {
	WorkPatterns        *WorkPatterns        `json:"work_patterns"`
	DecisionMaking      *DecisionPatterns    `json:"decision_making"`
	ProblemSolving      *ProblemSolvingStyle `json:"problem_solving"`
	LearningStyle       *LearningStyle       `json:"learning_style"`
	CommunicationStyle  *CommunicationStyle  `json:"communication_style"`
	ProductivityMetrics *ProductivityMetrics `json:"productivity_metrics"`
}

// WorkPatterns analyzes how the user works
type WorkPatterns struct {
	PreferredWorkTime    []string `json:"preferred_work_time"`   // ["morning", "afternoon"]
	SessionLength        string   `json:"session_length"`        // "short", "medium", "long"
	TaskSwitching        string   `json:"task_switching"`        // "frequent", "moderate", "rare"
	DeepWorkCapability   float64  `json:"deep_work_capability"`  // 0.0 to 1.0
	MultitaskingTendency float64  `json:"multitasking_tendency"` // 0.0 to 1.0
	PlanningStyle        string   `json:"planning_style"`        // "detailed", "high_level", "minimal"
}

// DecisionPatterns analyzes decision-making behavior
type DecisionPatterns struct {
	DecisionSpeed      string   `json:"decision_speed"`      // "fast", "moderate", "deliberate"
	AnalysisDepth      string   `json:"analysis_depth"`      // "deep", "moderate", "surface"
	RiskTolerance      string   `json:"risk_tolerance"`      // "high", "moderate", "low"
	ConsensusBuilding  string   `json:"consensus_building"`  // "collaborative", "consultative", "independent"
	ChangeReadiness    string   `json:"change_readiness"`    // "adaptable", "cautious", "resistant"
	DocumentationStyle string   `json:"documentation_style"` // "detailed", "summary", "minimal"
	PreferredFactors   []string `json:"preferred_factors"`   // ["cost", "time", "quality", "risk"]
}

// ProblemSolvingStyle analyzes approach to problem solving
type ProblemSolvingStyle struct {
	Approach          string   `json:"approach"`            // "systematic", "intuitive", "experimental"
	ResearchDepth     string   `json:"research_depth"`      // "thorough", "moderate", "minimal"
	ToolPreference    []string `json:"tool_preference"`     // ["documentation", "experimentation", "collaboration"]
	DebugStrategy     string   `json:"debug_strategy"`      // "methodical", "hypothesis_driven", "trial_error"
	SolutionSharing   string   `json:"solution_sharing"`    // "detailed", "summary", "link_only"
	LearningFromError string   `json:"learning_from_error"` // "reflective", "practical", "forward_focused"
}

// LearningStyle analyzes how the user learns and processes information
type LearningStyle struct {
	PreferredFormat   []string `json:"preferred_format"`   // ["visual", "text", "hands_on", "discussion"]
	InformationDepth  string   `json:"information_depth"`  // "comprehensive", "targeted", "minimal"
	RetentionStrategy string   `json:"retention_strategy"` // "notes", "practice", "teaching", "reference"
	FeedbackStyle     string   `json:"feedback_style"`     // "immediate", "periodic", "retrospective"
	KnowledgeSharing  string   `json:"knowledge_sharing"`  // "proactive", "responsive", "minimal"
}

// CommunicationStyle analyzes communication patterns
type CommunicationStyle struct {
	Verbosity      string   `json:"verbosity"`        // "detailed", "concise", "minimal"
	TechnicalDepth string   `json:"technical_depth"`  // "deep", "moderate", "high_level"
	Audience       []string `json:"audience"`         // ["technical", "business", "mixed"]
	UpdateFreq     string   `json:"update_frequency"` // "frequent", "regular", "milestone_based"
	Clarity        float64  `json:"clarity"`          // 0.0 to 1.0
	Structure      string   `json:"structure"`        // "formal", "informal", "mixed"
}

// ProductivityMetrics measures productivity patterns
type ProductivityMetrics struct {
	TaskCompletionRate  float64            `json:"task_completion_rate"` // 0.0 to 1.0
	AverageTaskDuration time.Duration      `json:"average_task_duration"`
	PeakProductiveTime  []string           `json:"peak_productive_time"` // ["9am-11am", "2pm-4pm"]
	DistractionLevel    string             `json:"distraction_level"`    // "low", "moderate", "high"
	FocusQuality        float64            `json:"focus_quality"`        // 0.0 to 1.0
	WorkflowEfficiency  float64            `json:"workflow_efficiency"`  // 0.0 to 1.0
	BottleneckAreas     []string           `json:"bottleneck_areas"`
	ProductivityTrends  map[string]float64 `json:"productivity_trends"` // week -> productivity score
}

// NewPatternDetector creates a new AI-powered pattern detector
func NewPatternDetector(cfg *config.Config, aiService AIService, logger logging.Logger) *PatternDetector {
	if logger == nil {
		// Use default logger if none provided
		logger = logging.NewLogger(logging.INFO)
	}

	return &PatternDetector{
		aiService:      aiService,
		config:         cfg,
		logger:         logger,
		patternHistory: make(map[string][]*DetectedPattern),
	}
}

// AnalyzeProjectPatterns performs comprehensive pattern analysis for a project
func (pd *PatternDetector) AnalyzeProjectPatterns(ctx context.Context, projectID string, memories []MemoryData, timeSpan time.Duration) (*PatternDetectionResult, error) {
	startTime := time.Now()

	pd.logger.Info("Starting comprehensive pattern analysis",
		"project_id", projectID,
		"memory_count", len(memories),
		"time_span", timeSpan)

	// Detect patterns in the memories
	patterns, err := pd.detectPatterns(ctx, memories, projectID)
	if err != nil {
		return nil, fmt.Errorf("pattern detection failed: %w", err)
	}

	// Analyze quality patterns
	qualityInsights, err := pd.analyzeQualityPatterns(ctx, memories)
	if err != nil {
		pd.logger.Warn("Quality analysis failed", "error", err)
		qualityInsights = &QualityInsights{} // Continue with empty insights
	}

	// Detect knowledge gaps
	knowledgeGaps, err := pd.detectKnowledgeGaps(ctx, memories, patterns)
	if err != nil {
		pd.logger.Warn("Knowledge gap detection failed", "error", err)
		knowledgeGaps = []*KnowledgeGap{} // Continue without gaps
	}

	// Analyze behavioral patterns
	behavioralInsights, err := pd.analyzeBehavioralPatterns(ctx, memories)
	if err != nil {
		pd.logger.Warn("Behavioral analysis failed", "error", err)
		behavioralInsights = &BehavioralInsights{} // Continue without behavioral insights
	}

	// Generate trends
	trends := pd.generateTrends(patterns, projectID)

	// Generate recommendations
	recommendations, err := pd.generateRecommendations(ctx, patterns, qualityInsights, knowledgeGaps)
	if err != nil {
		pd.logger.Warn("Recommendation generation failed", "error", err)
		recommendations = []*Recommendation{} // Continue without recommendations
	}

	result := &PatternDetectionResult{
		ProjectID:          projectID,
		AnalysisID:         uuid.New().String(),
		Patterns:           patterns,
		Trends:             trends,
		Recommendations:    recommendations,
		QualityInsights:    qualityInsights,
		KnowledgeGaps:      knowledgeGaps,
		BehavioralInsights: behavioralInsights,
		AnalyzedAt:         time.Now(),
		MemoryCount:        len(memories),
		TimeSpan:           timeSpan,
	}

	// Store patterns for trend analysis
	pd.patternHistory[projectID] = patterns

	duration := time.Since(startTime)
	pd.logger.Info("Pattern analysis completed",
		"duration", duration,
		"patterns_found", len(patterns),
		"recommendations", len(recommendations),
		"knowledge_gaps", len(knowledgeGaps))

	return result, nil
}

// MemoryData represents the data structure for a memory item
type MemoryData struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Type      string                 `json:"type"` // "chunk", "decision", "thread"
	CreatedAt time.Time              `json:"created_at"`
	Tags      []string               `json:"tags"`
	Quality   *UnifiedQualityMetrics `json:"quality,omitempty"`
	Analysis  *PatternAnalysis       `json:"analysis,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// detectPatterns uses AI to identify patterns across memories
func (pd *PatternDetector) detectPatterns(ctx context.Context, memories []MemoryData, projectID string) ([]*DetectedPattern, error) {
	if pd.aiService == nil {
		return []*DetectedPattern{}, nil
	}

	// Group memories by type and time period for better analysis
	memoryGroups := pd.groupMemoriesForAnalysis(memories)
	allPatterns := make([]*DetectedPattern, 0)

	for groupType, groupMemories := range memoryGroups {
		pd.logger.Info("Analyzing memory group",
			"project_id", projectID,
			"group_type", groupType,
			"memory_count", len(groupMemories))

		patterns, err := pd.analyzeMemoryGroup(ctx, groupMemories, groupType, projectID)
		if err != nil {
			pd.logger.Warn("Failed to analyze memory group",
				"group_type", groupType,
				"error", err)
			continue
		}

		allPatterns = append(allPatterns, patterns...)
	}

	// Deduplicate and rank patterns
	dedupedPatterns := pd.deduplicatePatterns(allPatterns)
	rankedPatterns := pd.rankPatternsByImportance(dedupedPatterns)

	return rankedPatterns, nil
}

// groupMemoriesForAnalysis groups memories for more effective pattern detection
func (pd *PatternDetector) groupMemoriesForAnalysis(memories []MemoryData) map[string][]MemoryData {
	groups := make(map[string][]MemoryData)

	for _, memory := range memories {
		// Group by type
		groupKey := memory.Type
		if groupKey == "" {
			groupKey = "general"
		}

		groups[groupKey] = append(groups[groupKey], memory)
	}

	// Also group by time periods for trend analysis
	groups["recent"] = pd.filterMemoriesByTime(memories, time.Hour*24*7)   // Last week
	groups["monthly"] = pd.filterMemoriesByTime(memories, time.Hour*24*30) // Last month

	return groups
}

// filterMemoriesByTime filters memories within a time period
func (pd *PatternDetector) filterMemoriesByTime(memories []MemoryData, duration time.Duration) []MemoryData {
	cutoff := time.Now().Add(-duration)
	filtered := make([]MemoryData, 0)

	for _, memory := range memories {
		if memory.CreatedAt.After(cutoff) {
			filtered = append(filtered, memory)
		}
	}

	return filtered
}

// analyzeMemoryGroup analyzes a specific group of memories for patterns
func (pd *PatternDetector) analyzeMemoryGroup(ctx context.Context, memories []MemoryData, groupType string, projectID string) ([]*DetectedPattern, error) {
	// Prepare content for analysis
	contents := make([]string, len(memories))
	for i, memory := range memories {
		contents[i] = memory.Content
	}

	// Create analysis prompt
	prompt := pd.buildPatternAnalysisPrompt(contents, groupType, projectID)

	// Get AI analysis
	response, err := pd.aiService.GenerateCompletion(ctx, prompt, &CompletionOptions{
		MaxTokens:   2000,
		Temperature: 0.3,
		TopP:        0.9,
	})
	if err != nil {
		return nil, fmt.Errorf("AI pattern analysis failed: %w", err)
	}

	// Parse response
	patterns, err := pd.parsePatternResponse(response.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pattern response: %w", err)
	}

	return patterns, nil
}

// buildPatternAnalysisPrompt creates a comprehensive prompt for pattern analysis
func (pd *PatternDetector) buildPatternAnalysisPrompt(contents []string, groupType string, projectID string) string {
	contentText := strings.Join(contents, "\n---MEMORY---\n")

	return fmt.Sprintf(`Analyze these %s memories from project %s to identify meaningful patterns:

%s

Identify patterns in:
1. Problem-solving approaches and strategies
2. Decision-making processes and criteria
3. Technical approaches and tool usage
4. Communication and documentation patterns
5. Learning and knowledge acquisition patterns
6. Time management and workflow patterns
7. Quality and attention to detail patterns
8. Collaboration and teamwork patterns

For each pattern, assess:
- Type: What kind of pattern this is
- Description: Clear explanation of the pattern
- Confidence: How confident you are this is a real pattern (0.0-1.0)
- Frequency: How often this pattern appears
- Examples: Specific examples from the content
- Tags: Related concepts and keywords

Return a JSON array of patterns:
[{
  "type": "problem_solving_approach",
  "description": "Prefers systematic debugging with detailed documentation",
  "confidence": 0.85,
  "frequency": 5,
  "examples": ["memory excerpt 1", "memory excerpt 2"],
  "tags": ["debugging", "systematic", "documentation"]
}]

Focus on actionable patterns that provide insights into work habits, preferences, and effectiveness.`, groupType, projectID, contentText)
}

// parsePatternResponse parses the AI response into DetectedPattern objects
func (pd *PatternDetector) parsePatternResponse(response string) ([]*DetectedPattern, error) {
	jsonStr := pd.extractJSONFromResponse(response)

	var patterns []*DetectedPattern
	err := json.Unmarshal([]byte(jsonStr), &patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal patterns JSON: %w", err)
	}

	return patterns, nil
}

// extractJSONFromResponse extracts JSON from AI response
func (pd *PatternDetector) extractJSONFromResponse(response string) string {
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		if strings.HasSuffix(response, "```") {
			response = strings.TrimSuffix(response, "```")
		}
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		if strings.HasSuffix(response, "```") {
			response = strings.TrimSuffix(response, "```")
		}
	}
	return strings.TrimSpace(response)
}

// deduplicatePatterns removes similar patterns
func (pd *PatternDetector) deduplicatePatterns(patterns []*DetectedPattern) []*DetectedPattern {
	seen := make(map[string]*DetectedPattern)

	for _, pattern := range patterns {
		key := pattern.Type + ":" + pattern.Description
		if existing, found := seen[key]; found {
			// Merge with existing pattern
			existing.Frequency += pattern.Frequency
			existing.Confidence = (existing.Confidence + pattern.Confidence) / 2
			existing.Examples = append(existing.Examples, pattern.Examples...)
		} else {
			seen[key] = pattern
		}
	}

	result := make([]*DetectedPattern, 0, len(seen))
	for _, pattern := range seen {
		result = append(result, pattern)
	}

	return result
}

// rankPatternsByImportance sorts patterns by importance score
func (pd *PatternDetector) rankPatternsByImportance(patterns []*DetectedPattern) []*DetectedPattern {
	// Calculate importance score for each pattern
	for _, pattern := range patterns {
		score := pattern.Confidence * float64(pattern.Frequency)
		if pattern.Type == "problem_solving_approach" || pattern.Type == "decision_making" {
			score *= 1.5 // Boost important pattern types
		}
		pattern.Confidence = score // Store score in confidence for sorting
	}

	// Sort by importance score (stored in confidence)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Confidence > patterns[j].Confidence
	})

	// Restore original confidence values (this is a simplification)
	for _, pattern := range patterns {
		pattern.Confidence = pattern.Confidence / float64(pattern.Frequency)
		if pattern.Type == "problem_solving_approach" || pattern.Type == "decision_making" {
			pattern.Confidence /= 1.5
		}
	}

	return patterns
}

// analyzeQualityPatterns analyzes quality trends across memories
func (pd *PatternDetector) analyzeQualityPatterns(ctx context.Context, memories []MemoryData) (*QualityInsights, error) {
	if len(memories) == 0 {
		return &QualityInsights{}, nil
	}

	qualityScores := make([]float64, 0, len(memories))
	qualityByCategory := make(map[string][]float64)
	highQualityCount := 0
	lowQualityCount := 0

	for _, memory := range memories {
		if memory.Quality != nil {
			score := memory.Quality.OverallScore
			qualityScores = append(qualityScores, score)

			if score > 0.8 {
				highQualityCount++
			} else if score < 0.5 {
				lowQualityCount++
			}

			// Group by memory type
			memoryType := memory.Type
			if memoryType == "" {
				memoryType = "general"
			}
			qualityByCategory[memoryType] = append(qualityByCategory[memoryType], score)
		}
	}

	// Calculate overall quality
	overallQuality := pd.calculateAverage(qualityScores)

	// Calculate quality by category averages
	categoryAverages := make(map[string]float64)
	for category, scores := range qualityByCategory {
		categoryAverages[category] = pd.calculateAverage(scores)
	}

	// Determine quality trend (simplified)
	qualityTrend := "stable"
	if len(qualityScores) > 5 {
		recent := pd.calculateAverage(qualityScores[len(qualityScores)-5:])
		older := pd.calculateAverage(qualityScores[:5])
		if recent > older+0.1 {
			qualityTrend = "improving"
		} else if recent < older-0.1 {
			qualityTrend = "declining"
		}
	}

	return &QualityInsights{
		OverallQuality:    overallQuality,
		QualityTrend:      qualityTrend,
		HighQualityCount:  highQualityCount,
		LowQualityCount:   lowQualityCount,
		QualityByCategory: categoryAverages,
		ImprovementAreas:  []string{"Add more context", "Include decision rationale"},
		BestPractices:     []string{"Document assumptions", "Include examples"},
	}, nil
}

// calculateAverage calculates the average of a slice of float64
func (pd *PatternDetector) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

// detectKnowledgeGaps identifies areas where knowledge is missing or incomplete
func (pd *PatternDetector) detectKnowledgeGaps(ctx context.Context, memories []MemoryData, patterns []*DetectedPattern) ([]*KnowledgeGap, error) {
	if pd.aiService == nil {
		return []*KnowledgeGap{}, nil
	}

	// Analyze memories and patterns to identify gaps
	content := pd.prepareContentForGapAnalysis(memories, patterns)

	prompt := fmt.Sprintf(`Analyze this project's memories and patterns to identify knowledge gaps:

%s

Identify areas where:
1. Problems are mentioned but solutions are incomplete
2. Decisions are made without clear rationale
3. Technologies are used but not well understood
4. Repeated issues suggest missing foundational knowledge
5. Questions are asked but not answered
6. Implementation details are missing

Return a JSON array of knowledge gaps:
[{
  "area": "authentication_best_practices",
  "severity": "important",
  "description": "Limited understanding of secure authentication patterns",
  "evidence": ["Multiple login issues", "Security concerns raised"],
  "suggestions": ["Study OAuth 2.0", "Review security guidelines"],
  "related": ["security", "user_management"],
  "priority": 3
}]`, content)

	response, err := pd.aiService.GenerateCompletion(ctx, prompt, &CompletionOptions{
		MaxTokens:   1000,
		Temperature: 0.3,
		TopP:        0.9,
	})
	if err != nil {
		return nil, err
	}

	var gaps []*KnowledgeGap
	jsonStr := pd.extractJSONFromResponse(response.Content)
	err = json.Unmarshal([]byte(jsonStr), &gaps)
	if err != nil {
		return nil, fmt.Errorf("failed to parse knowledge gaps: %w", err)
	}

	// Add IDs to gaps
	for _, gap := range gaps {
		gap.ID = uuid.New().String()
	}

	return gaps, nil
}

// prepareContentForGapAnalysis prepares content for knowledge gap analysis
func (pd *PatternDetector) prepareContentForGapAnalysis(memories []MemoryData, patterns []*DetectedPattern) string {
	var builder strings.Builder

	// Add memory contents
	builder.WriteString("=== MEMORIES ===\n")
	for i, memory := range memories {
		if i >= 20 { // Limit to avoid token limits
			break
		}
		builder.WriteString(fmt.Sprintf("Memory %d (%s): %s\n", i+1, memory.Type, memory.Content))
	}

	// Add detected patterns
	builder.WriteString("\n=== DETECTED PATTERNS ===\n")
	for _, pattern := range patterns {
		builder.WriteString(fmt.Sprintf("Pattern: %s - %s\n", pattern.Type, pattern.Description))
	}

	return builder.String()
}

// analyzeBehavioralPatterns analyzes user behavior patterns
func (pd *PatternDetector) analyzeBehavioralPatterns(ctx context.Context, memories []MemoryData) (*BehavioralInsights, error) {
	// This is a simplified implementation
	// In a full implementation, this would analyze timestamps, session patterns, etc.

	return &BehavioralInsights{
		WorkPatterns: &WorkPatterns{
			SessionLength:        "medium",
			TaskSwitching:        "moderate",
			DeepWorkCapability:   0.7,
			MultitaskingTendency: 0.5,
			PlanningStyle:        "moderate",
		},
		DecisionMaking: &DecisionPatterns{
			DecisionSpeed:      "moderate",
			AnalysisDepth:      "moderate",
			RiskTolerance:      "moderate",
			ConsensusBuilding:  "consultative",
			DocumentationStyle: "summary",
		},
		ProblemSolving: &ProblemSolvingStyle{
			Approach:        "systematic",
			ResearchDepth:   "moderate",
			DebugStrategy:   "methodical",
			SolutionSharing: "detailed",
		},
		ProductivityMetrics: &ProductivityMetrics{
			TaskCompletionRate: 0.8,
			DistractionLevel:   "moderate",
			FocusQuality:       0.7,
			WorkflowEfficiency: 0.75,
		},
	}, nil
}

// generateTrends creates trend analysis based on pattern history
func (pd *PatternDetector) generateTrends(patterns []*DetectedPattern, projectID string) []*PatternTrend {
	trends := make([]*PatternTrend, 0)

	// This is a simplified implementation
	// In a full implementation, this would compare with historical data

	for _, pattern := range patterns {
		trend := &PatternTrend{
			PatternType:  pattern.Type,
			Direction:    "stable",
			Strength:     pattern.Confidence,
			TimeFrame:    "last_month",
			Significance: "medium",
			Description:  fmt.Sprintf("Pattern %s appears to be stable", pattern.Type),
			DataPoints:   []float64{pattern.Confidence},
		}
		trends = append(trends, trend)
	}

	return trends
}

// generateRecommendations creates AI-powered recommendations based on analysis
func (pd *PatternDetector) generateRecommendations(ctx context.Context, patterns []*DetectedPattern, quality *QualityInsights, gaps []*KnowledgeGap) ([]*Recommendation, error) {
	if pd.aiService == nil {
		return pd.generateFallbackRecommendations(patterns, quality, gaps), nil
	}

	// Prepare analysis summary for recommendation generation
	summary := pd.prepareAnalysisSummary(patterns, quality, gaps)

	prompt := fmt.Sprintf(`Based on this analysis of work patterns and quality, generate actionable recommendations:

%s

Generate 3-5 high-impact recommendations that would:
1. Address identified knowledge gaps
2. Improve work quality and efficiency
3. Leverage existing strengths
4. Prevent recurring issues
5. Enhance learning and growth

Return a JSON array of recommendations:
[{
  "type": "process_improvement",
  "priority": "high",
  "title": "Implement systematic documentation review",
  "description": "Regular review process to ensure completeness",
  "action_items": ["Schedule weekly reviews", "Create checklist"],
  "benefits": ["Higher quality", "Better knowledge retention"],
  "effort": "medium",
  "impact": "high",
  "confidence": 0.8
}]`, summary)

	response, err := pd.aiService.GenerateCompletion(ctx, prompt, &CompletionOptions{
		MaxTokens:   1200,
		Temperature: 0.3,
		TopP:        0.9,
	})
	if err != nil {
		return pd.generateFallbackRecommendations(patterns, quality, gaps), nil
	}

	var recommendations []*Recommendation
	jsonStr := pd.extractJSONFromResponse(response.Content)
	err = json.Unmarshal([]byte(jsonStr), &recommendations)
	if err != nil {
		pd.logger.Warn("Failed to parse recommendations, using fallback", "error", err)
		return pd.generateFallbackRecommendations(patterns, quality, gaps), nil
	}

	// Add IDs and due dates
	for _, rec := range recommendations {
		rec.ID = uuid.New().String()
		rec.DueDate = time.Now().Add(time.Hour * 24 * 30) // 30 days from now
	}

	return recommendations, nil
}

// prepareAnalysisSummary creates a summary for recommendation generation
func (pd *PatternDetector) prepareAnalysisSummary(patterns []*DetectedPattern, quality *QualityInsights, gaps []*KnowledgeGap) string {
	var builder strings.Builder

	builder.WriteString("=== PATTERN SUMMARY ===\n")
	for _, pattern := range patterns {
		builder.WriteString(fmt.Sprintf("- %s: %s (confidence: %.2f)\n", pattern.Type, pattern.Description, pattern.Confidence))
	}

	builder.WriteString("\n=== QUALITY INSIGHTS ===\n")
	builder.WriteString(fmt.Sprintf("Overall Quality: %.2f, Trend: %s\n", quality.OverallQuality, quality.QualityTrend))
	builder.WriteString(fmt.Sprintf("High Quality: %d, Low Quality: %d\n", quality.HighQualityCount, quality.LowQualityCount))

	builder.WriteString("\n=== KNOWLEDGE GAPS ===\n")
	for _, gap := range gaps {
		builder.WriteString(fmt.Sprintf("- %s: %s (severity: %s)\n", gap.Area, gap.Description, gap.Severity))
	}

	return builder.String()
}

// generateFallbackRecommendations creates basic recommendations when AI is unavailable
func (pd *PatternDetector) generateFallbackRecommendations(patterns []*DetectedPattern, quality *QualityInsights, gaps []*KnowledgeGap) []*Recommendation {
	recommendations := make([]*Recommendation, 0)

	// Quality-based recommendations
	if quality.OverallQuality < 0.7 {
		rec := &Recommendation{
			ID:          uuid.New().String(),
			Type:        "quality_improvement",
			Priority:    "high",
			Title:       "Improve Memory Quality",
			Description: "Focus on creating more complete and clear memories",
			ActionItems: []string{"Add more context", "Include examples", "Document decisions"},
			Benefits:    []string{"Better knowledge retention", "Easier future reference"},
			Effort:      "medium",
			Impact:      "high",
			Confidence:  0.8,
			DueDate:     time.Now().Add(time.Hour * 24 * 14),
		}
		recommendations = append(recommendations, rec)
	}

	// Knowledge gap recommendations
	if len(gaps) > 0 {
		rec := &Recommendation{
			ID:          uuid.New().String(),
			Type:        "knowledge_capture",
			Priority:    "medium",
			Title:       "Address Knowledge Gaps",
			Description: "Focus on filling identified knowledge gaps",
			ActionItems: []string{"Research missing areas", "Document learnings", "Create reference materials"},
			Benefits:    []string{"More complete knowledge base", "Reduced repeated problems"},
			Effort:      "high",
			Impact:      "high",
			Confidence:  0.7,
			DueDate:     time.Now().Add(time.Hour * 24 * 30),
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations
}
