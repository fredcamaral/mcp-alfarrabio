// Package tasks provides task quality scoring functionality.
package tasks

import (
	"math"
	"strings"

	"lerian-mcp-memory/pkg/types"
)

// Scorer calculates quality scores for tasks
type Scorer struct {
	config ScorerConfig
}

// ScorerConfig represents configuration for task scoring
type ScorerConfig struct {
	ClarityWeight      float64            `json:"clarity_weight"`
	CompletenessWeight float64            `json:"completeness_weight"`
	ActionabilityWeight float64           `json:"actionability_weight"`
	SpecificityWeight  float64            `json:"specificity_weight"`
	FeasibilityWeight  float64            `json:"feasibility_weight"`
	TestabilityWeight  float64            `json:"testability_weight"`
	QualityThresholds  QualityThresholds  `json:"quality_thresholds"`
	ScoringCriteria    ScoringCriteria    `json:"scoring_criteria"`
}

// QualityThresholds represents thresholds for quality scoring
type QualityThresholds struct {
	Excellent float64 `json:"excellent"` // 0.9+
	Good      float64 `json:"good"`      // 0.7+
	Fair      float64 `json:"fair"`      // 0.5+
	Poor      float64 `json:"poor"`      // <0.5
}

// ScoringCriteria represents criteria for different scoring aspects
type ScoringCriteria struct {
	MinTitleLength        int      `json:"min_title_length"`
	MinDescriptionLength  int      `json:"min_description_length"`
	MinAcceptanceCriteria int      `json:"min_acceptance_criteria"`
	ActionVerbs           []string `json:"action_verbs"`
	SpecificityKeywords   []string `json:"specificity_keywords"`
	VagueWords            []string `json:"vague_words"`
	TestableKeywords      []string `json:"testable_keywords"`
}

// DefaultScorerConfig returns default scorer configuration
func DefaultScorerConfig() ScorerConfig {
	return ScorerConfig{
		ClarityWeight:       0.20,
		CompletenessWeight:  0.20,
		ActionabilityWeight: 0.15,
		SpecificityWeight:   0.15,
		FeasibilityWeight:   0.15,
		TestabilityWeight:   0.15,
		QualityThresholds: QualityThresholds{
			Excellent: 0.9,
			Good:      0.7,
			Fair:      0.5,
			Poor:      0.0,
		},
		ScoringCriteria: ScoringCriteria{
			MinTitleLength:        5,
			MinDescriptionLength:  20,
			MinAcceptanceCriteria: 2,
			ActionVerbs: []string{
				"implement", "create", "build", "develop", "design", "fix", "update",
				"add", "remove", "modify", "refactor", "test", "deploy", "setup",
				"configure", "install", "integrate", "optimize", "analyze", "research",
				"document", "review", "audit", "validate", "verify", "enhance",
				"migrate", "upgrade", "improve", "clean", "reorganize", "establish",
			},
			SpecificityKeywords: []string{
				"specific", "exactly", "precisely", "particular", "detailed", "explicit",
				"clear", "defined", "concrete", "measurable", "quantifiable",
			},
			VagueWords: []string{
				"something", "stuff", "things", "various", "misc", "general", "some",
				"maybe", "perhaps", "might", "could", "possibly", "probably",
				"somehow", "somewhat", "kind of", "sort of", "a bit", "a little",
			},
			TestableKeywords: []string{
				"verify", "confirm", "ensure", "check", "validate", "test", "measure",
				"assert", "expect", "should", "must", "will", "shall", "can",
				"returns", "displays", "shows", "contains", "includes", "equals",
			},
		},
	}
}

// NewScorer creates a new task scorer
func NewScorer() *Scorer {
	return &Scorer{
		config: DefaultScorerConfig(),
	}
}

// NewScorerWithConfig creates a new task scorer with custom config
func NewScorerWithConfig(config ScorerConfig) *Scorer {
	return &Scorer{
		config: config,
	}
}

// ScoreTask calculates a comprehensive quality score for a task
func (s *Scorer) ScoreTask(task *types.Task, context types.TaskGenerationContext) types.QualityScore {
	if task == nil {
		return types.QualityScore{
			OverallScore: 0.0,
			Issues: []types.QualityIssue{
				{
					Type:        "critical",
					Severity:    "critical",
					Description: "Task is nil",
					Suggestion:  "Provide a valid task object",
				},
			},
		}
	}

	// Calculate individual quality scores
	clarity := s.scoreClarityFactor(task)
	completeness := s.scoreCompletenessFactor(task)
	actionability := s.scoreActionabilityFactor(task)
	specificity := s.scoreSpecificityFactor(task)
	feasibility := s.scoreFeasibilityFactor(task, context)
	testability := s.scoreTestabilityFactor(task)

	// Calculate weighted overall score
	overallScore := (clarity * s.config.ClarityWeight +
		completeness * s.config.CompletenessWeight +
		actionability * s.config.ActionabilityWeight +
		specificity * s.config.SpecificityWeight +
		feasibility * s.config.FeasibilityWeight +
		testability * s.config.TestabilityWeight)

	// Identify issues and generate recommendations
	issues := s.identifyQualityIssues(task, clarity, completeness, actionability, specificity, feasibility, testability)
	recommendations := s.generateRecommendations(task, issues)

	return types.QualityScore{
		OverallScore:    overallScore,
		Clarity:         clarity,
		Completeness:    completeness,
		Actionability:   actionability,
		Specificity:     specificity,
		Feasibility:     feasibility,
		Testability:     testability,
		Issues:          issues,
		Recommendations: recommendations,
	}
}

// scoreClarityFactor scores how clear and understandable the task is
func (s *Scorer) scoreClarityFactor(task *types.Task) float64 {
	score := 1.0
	deductions := 0.0

	content := strings.ToLower(task.Title + " " + task.Description)

	// Check for vague language
	vaguenessCount := 0
	for _, vague := range s.config.ScoringCriteria.VagueWords {
		if strings.Contains(content, vague) {
			vaguenessCount++
		}
	}
	if vaguenessCount > 0 {
		deductions += float64(vaguenessCount) * 0.1
	}

	// Check title clarity
	if len(strings.TrimSpace(task.Title)) < s.config.ScoringCriteria.MinTitleLength {
		deductions += 0.2
	}

	// Check for technical jargon without explanation
	jargonWords := []string{
		"api", "orm", "crud", "oauth", "jwt", "ssl", "cdn", "dns",
		"microservice", "lambda", "kubernetes", "docker", "terraform",
	}
	jargonCount := 0
	for _, jargon := range jargonWords {
		if strings.Contains(content, jargon) {
			jargonCount++
		}
	}
	// Moderate jargon is OK, but too much without context reduces clarity
	if jargonCount > 3 && len(task.Description) < 100 {
		deductions += 0.15
	}

	// Check for ambiguous pronouns
	ambiguousPronouns := []string{"it", "this", "that", "they", "them"}
	pronounCount := 0
	for _, pronoun := range ambiguousPronouns {
		pronounCount += strings.Count(content, " "+pronoun+" ")
	}
	if pronounCount > 3 {
		deductions += 0.1
	}

	return math.Max(score-deductions, 0.0)
}

// scoreCompletenessFactor scores how complete the task definition is
func (s *Scorer) scoreCompletenessFactor(task *types.Task) float64 {
	score := 0.0

	// Required fields (basic completeness)
	if task.Title != "" {
		score += 0.15
	}
	if task.Description != "" && len(task.Description) >= s.config.ScoringCriteria.MinDescriptionLength {
		score += 0.20
	}
	if task.Type != "" {
		score += 0.10
	}
	if task.Priority != "" {
		score += 0.10
	}

	// Acceptance criteria
	if len(task.AcceptanceCriteria) >= s.config.ScoringCriteria.MinAcceptanceCriteria {
		score += 0.15
		// Bonus for detailed criteria
		avgCriteriaLength := 0
		for _, criteria := range task.AcceptanceCriteria {
			avgCriteriaLength += len(criteria)
		}
		if len(task.AcceptanceCriteria) > 0 {
			avgCriteriaLength /= len(task.AcceptanceCriteria)
			if avgCriteriaLength > 30 {
				score += 0.05
			}
		}
	}

	// Effort estimation
	if task.EstimatedEffort.Hours > 0 {
		score += 0.10
		// Bonus for detailed breakdown
		if task.EstimatedEffort.EstimationMethod != "" {
			score += 0.05
		}
	}

	// Dependencies and relationships
	if len(task.Dependencies) > 0 {
		score += 0.05
	}

	// Tags for organization
	if len(task.Tags) > 0 {
		score += 0.05
	}

	return math.Min(score, 1.0)
}

// scoreActionabilityFactor scores how actionable the task is
func (s *Scorer) scoreActionabilityFactor(task *types.Task) float64 {
	score := 0.0

	titleLower := strings.ToLower(task.Title)
	contentLower := strings.ToLower(task.Title + " " + task.Description)

	// Check for action verbs in title
	hasActionVerb := false
	for _, verb := range s.config.ScoringCriteria.ActionVerbs {
		if strings.HasPrefix(titleLower, verb) {
			hasActionVerb = true
			score += 0.3
			break
		}
	}

	if !hasActionVerb {
		// Check for action verbs anywhere in title
		for _, verb := range s.config.ScoringCriteria.ActionVerbs {
			if strings.Contains(titleLower, verb) {
				score += 0.15
				break
			}
		}
	}

	// Check for specific deliverables
	deliverableKeywords := []string{
		"endpoint", "component", "service", "function", "class", "module",
		"interface", "database", "table", "schema", "migration", "test",
		"documentation", "readme", "guide", "tutorial", "api", "ui",
	}
	deliverableCount := 0
	for _, keyword := range deliverableKeywords {
		if strings.Contains(contentLower, keyword) {
			deliverableCount++
		}
	}
	if deliverableCount > 0 {
		score += math.Min(float64(deliverableCount)*0.1, 0.3)
	}

	// Check for clear outcomes
	outcomeKeywords := []string{
		"result", "output", "produce", "generate", "create", "build",
		"deliver", "complete", "finish", "accomplish", "achieve",
	}
	for _, keyword := range outcomeKeywords {
		if strings.Contains(contentLower, keyword) {
			score += 0.2
			break
		}
	}

	// Check for measurable criteria
	measurableKeywords := []string{
		"measure", "count", "number", "percentage", "ratio", "time",
		"performance", "speed", "size", "volume", "quantity",
	}
	for _, keyword := range measurableKeywords {
		if strings.Contains(contentLower, keyword) {
			score += 0.1
			break
		}
	}

	// Deduct for passive language
	passiveLanguage := []string{
		"should be", "needs to be", "ought to", "might be", "could be",
		"would be nice", "it would be good", "someone should",
	}
	for _, passive := range passiveLanguage {
		if strings.Contains(contentLower, passive) {
			score -= 0.1
			break
		}
	}

	return math.Max(math.Min(score, 1.0), 0.0)
}

// scoreSpecificityFactor scores how specific and detailed the task is
func (s *Scorer) scoreSpecificityFactor(task *types.Task) float64 {
	score := 0.5 // Start with baseline

	content := strings.ToLower(task.Title + " " + task.Description)

	// Boost for specific keywords
	for _, keyword := range s.config.ScoringCriteria.SpecificityKeywords {
		if strings.Contains(content, keyword) {
			score += 0.1
		}
	}

	// Deduct for vague language
	vaguenessDeduction := 0.0
	for _, vague := range s.config.ScoringCriteria.VagueWords {
		if strings.Contains(content, vague) {
			vaguenessDeduction += 0.1
		}
	}
	score -= vaguenessDeduction

	// Check for specific technical details
	technicalDetails := []string{
		"endpoint", "url", "port", "protocol", "method", "header",
		"parameter", "response", "status code", "format", "json",
		"xml", "csv", "database", "table", "column", "index",
		"query", "function", "class", "method", "variable",
		"algorithm", "pattern", "framework", "library", "version",
	}
	detailCount := 0
	for _, detail := range technicalDetails {
		if strings.Contains(content, detail) {
			detailCount++
		}
	}
	if detailCount > 0 {
		score += math.Min(float64(detailCount)*0.05, 0.3)
	}

	// Check for numerical specifications
	numericalPatterns := []string{
		"version", "size", "limit", "timeout", "retry", "count",
		"maximum", "minimum", "threshold", "capacity", "scale",
	}
	for _, pattern := range numericalPatterns {
		if strings.Contains(content, pattern) {
			score += 0.05
		}
	}

	// Boost for file names, paths, or specific references
	if strings.Contains(content, ".") || strings.Contains(content, "/") || strings.Contains(content, "\\") {
		score += 0.1
	}

	return math.Max(math.Min(score, 1.0), 0.0)
}

// scoreFeasibilityFactor scores how feasible the task is given context
func (s *Scorer) scoreFeasibilityFactor(task *types.Task, context types.TaskGenerationContext) float64 {
	score := 0.8 // Start with good baseline

	// Check effort vs complexity alignment
	if task.EstimatedEffort.Hours > 0 && task.Complexity.Level != "" {
		complexityHours := map[types.ComplexityLevel]float64{
			types.ComplexityTrivial:     2.0,
			types.ComplexitySimple:      8.0,
			types.ComplexityModerate:    24.0,
			types.ComplexityComplex:     80.0,
			types.ComplexityVeryComplex: 200.0,
		}

		if expectedHours, exists := complexityHours[task.Complexity.Level]; exists {
			ratio := task.EstimatedEffort.Hours / expectedHours
			if ratio < 0.3 || ratio > 3.0 {
				score -= 0.2 // Significantly misaligned effort vs complexity
			} else if ratio < 0.5 || ratio > 2.0 {
				score -= 0.1 // Somewhat misaligned
			}
		}
	}

	// Check for unrealistic timelines
	if task.EstimatedEffort.Hours > 160 { // More than 4 weeks
		score -= 0.3
	} else if task.EstimatedEffort.Hours > 80 { // More than 2 weeks
		score -= 0.1
	}

	// Check for required skills vs team size
	if len(task.Complexity.RequiredSkills) > context.TeamSize*2 {
		score -= 0.2 // Too many diverse skills required
	}

	// Check for technology stack alignment
	content := strings.ToLower(task.Title + " " + task.Description)
	mentionedTech := 0
	for _, tech := range context.TechStack {
		if strings.Contains(content, strings.ToLower(tech)) {
			mentionedTech++
		}
	}
	if len(context.TechStack) > 0 && mentionedTech == 0 {
		// Task doesn't mention any known technologies
		score -= 0.1
	}

	// Check for external dependencies
	if len(task.Complexity.ExternalDependencies) > 3 {
		score -= 0.1 // Many external dependencies increase risk
	}

	return math.Max(score, 0.0)
}

// scoreTestabilityFactor scores how testable the task and its outcomes are
func (s *Scorer) scoreTestabilityFactor(task *types.Task) float64 {
	score := 0.0

	// Check acceptance criteria for testability
	testableCriteria := 0
	for _, criteria := range task.AcceptanceCriteria {
		criteriaLower := strings.ToLower(criteria)
		for _, keyword := range s.config.ScoringCriteria.TestableKeywords {
			if strings.Contains(criteriaLower, keyword) {
				testableCriteria++
				break
			}
		}
	}
	if len(task.AcceptanceCriteria) > 0 {
		score += (float64(testableCriteria) / float64(len(task.AcceptanceCriteria))) * 0.4
	}

	// Check for specific test types mentioned
	testTypes := []string{
		"unit test", "integration test", "e2e test", "performance test",
		"security test", "load test", "stress test", "regression test",
		"acceptance test", "smoke test", "api test", "ui test",
	}
	content := strings.ToLower(task.Title + " " + task.Description)
	mentionedTests := 0
	for _, testType := range testTypes {
		if strings.Contains(content, testType) {
			mentionedTests++
		}
	}
	if mentionedTests > 0 {
		score += math.Min(float64(mentionedTests)*0.1, 0.3)
	}

	// Check for measurable outcomes
	measurableOutcomes := []string{
		"response time", "error rate", "success rate", "throughput",
		"availability", "performance", "accuracy", "coverage",
		"count", "percentage", "ratio", "metric", "kpi",
	}
	measurableCount := 0
	for _, outcome := range measurableOutcomes {
		if strings.Contains(content, outcome) {
			measurableCount++
		}
	}
	if measurableCount > 0 {
		score += math.Min(float64(measurableCount)*0.05, 0.2)
	}

	// Deduct for untestable language
	untestableWords := []string{
		"improved", "better", "enhanced", "optimized", "cleaner",
		"nicer", "user-friendly", "intuitive", "smooth", "elegant",
	}
	for _, word := range untestableWords {
		if strings.Contains(content, word) {
			score -= 0.05
		}
	}

	return math.Max(math.Min(score, 1.0), 0.0)
}

// identifyQualityIssues identifies specific quality issues with the task
func (s *Scorer) identifyQualityIssues(task *types.Task, clarity, completeness, actionability, specificity, feasibility, testability float64) []types.QualityIssue {
	issues := []types.QualityIssue{}

	// Check for critical issues
	if clarity < 0.3 {
		issues = append(issues, types.QualityIssue{
			Type:        "clarity",
			Severity:    "high",
			Description: "Task description is unclear or confusing",
			Suggestion:  "Rewrite using simpler, more direct language",
		})
	}

	if completeness < 0.4 {
		issues = append(issues, types.QualityIssue{
			Type:        "completeness",
			Severity:    "high",
			Description: "Task is missing essential information",
			Suggestion:  "Add acceptance criteria, effort estimation, and detailed description",
		})
	}

	if actionability < 0.4 {
		issues = append(issues, types.QualityIssue{
			Type:        "actionability",
			Severity:    "medium",
			Description: "Task is not clearly actionable",
			Suggestion:  "Start with an action verb and specify clear deliverables",
		})
	}

	if specificity < 0.4 {
		issues = append(issues, types.QualityIssue{
			Type:        "specificity",
			Severity:    "medium",
			Description: "Task lacks specific technical details",
			Suggestion:  "Add specific requirements, technologies, and implementation details",
		})
	}

	if feasibility < 0.5 {
		issues = append(issues, types.QualityIssue{
			Type:        "feasibility",
			Severity:    "high",
			Description: "Task may not be feasible given current constraints",
			Suggestion:  "Review effort estimation, complexity, and resource requirements",
		})
	}

	if testability < 0.4 {
		issues = append(issues, types.QualityIssue{
			Type:        "testability",
			Severity:    "medium",
			Description: "Task outcomes are difficult to test or verify",
			Suggestion:  "Add measurable acceptance criteria and specific test requirements",
		})
	}

	// Check for specific content issues
	content := strings.ToLower(task.Title + " " + task.Description)
	
	// Check for vague language
	vaguenessCount := 0
	for _, vague := range s.config.ScoringCriteria.VagueWords {
		if strings.Contains(content, vague) {
			vaguenessCount++
		}
	}
	if vaguenessCount > 2 {
		issues = append(issues, types.QualityIssue{
			Type:        "clarity",
			Severity:    "medium",
			Description: "Task contains vague language that reduces clarity",
			Suggestion:  "Replace vague terms with specific, concrete language",
		})
	}

	return issues
}

// generateRecommendations generates improvement recommendations
func (s *Scorer) generateRecommendations(task *types.Task, issues []types.QualityIssue) []string {
	recommendations := []string{}

	// Add issue-specific recommendations
	for _, issue := range issues {
		recommendations = append(recommendations, issue.Suggestion)
	}

	// Add general recommendations based on missing elements
	if len(task.AcceptanceCriteria) < 2 {
		recommendations = append(recommendations, "Add at least 2-3 specific acceptance criteria")
	}

	if task.EstimatedEffort.Hours == 0 {
		recommendations = append(recommendations, "Provide effort estimation to help with planning")
	}

	if len(task.Tags) == 0 {
		recommendations = append(recommendations, "Add relevant tags for better organization")
	}

	if task.Complexity.Level == "" {
		recommendations = append(recommendations, "Assess and document task complexity")
	}

	if len(task.Complexity.RequiredSkills) == 0 {
		recommendations = append(recommendations, "Identify required skills for resource planning")
	}

	// Remove duplicates
	seen := make(map[string]bool)
	uniqueRecommendations := []string{}
	for _, rec := range recommendations {
		if !seen[rec] {
			uniqueRecommendations = append(uniqueRecommendations, rec)
			seen[rec] = true
		}
	}

	return uniqueRecommendations
}

// GetQualityLevel returns a human-readable quality level based on score
func (s *Scorer) GetQualityLevel(score float64) string {
	if score >= s.config.QualityThresholds.Excellent {
		return "Excellent"
	} else if score >= s.config.QualityThresholds.Good {
		return "Good"
	} else if score >= s.config.QualityThresholds.Fair {
		return "Fair"
	} else {
		return "Poor"
	}
}

// ScoreMultipleTasks scores multiple tasks and returns aggregate statistics
func (s *Scorer) ScoreMultipleTasks(tasks []types.Task, context types.TaskGenerationContext) TaskSetScore {
	if len(tasks) == 0 {
		return TaskSetScore{
			TaskCount:    0,
			AverageScore: 0.0,
			ScoreDistribution: map[string]int{
				"Excellent": 0,
				"Good":      0,
				"Fair":      0,
				"Poor":      0,
			},
		}
	}

	totalScore := 0.0
	distribution := map[string]int{
		"Excellent": 0,
		"Good":      0,
		"Fair":      0,
		"Poor":      0,
	}

	for _, task := range tasks {
		quality := s.ScoreTask(&task, context)
		totalScore += quality.OverallScore
		level := s.GetQualityLevel(quality.OverallScore)
		distribution[level]++
	}

	return TaskSetScore{
		TaskCount:         len(tasks),
		AverageScore:      totalScore / float64(len(tasks)),
		ScoreDistribution: distribution,
	}
}

// TaskSetScore represents aggregate scoring for a set of tasks
type TaskSetScore struct {
	TaskCount         int            `json:"task_count"`
	AverageScore      float64        `json:"average_score"`
	ScoreDistribution map[string]int `json:"score_distribution"`
}