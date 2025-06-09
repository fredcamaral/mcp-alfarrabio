package entities

import (
	"sort"
	"time"
)

// InsightType represents different types of cross-repository insights
type InsightType string

const (
	InsightTypePattern      InsightType = "pattern"
	InsightTypeWorkflow     InsightType = "workflow"
	InsightTypeBestPractice InsightType = "best_practice"
	InsightTypeAntiPattern  InsightType = "anti_pattern"
	InsightTypeTrend        InsightType = "trend"
	InsightTypeOptimization InsightType = "optimization"
)

// CrossRepoInsight represents an insight derived from cross-repository analysis
type CrossRepoInsight struct {
	ID              string                 `json:"id" validate:"required,uuid"`
	Type            InsightType            `json:"type" validate:"required"`
	Title           string                 `json:"title" validate:"required,min=1,max=200"`
	Description     string                 `json:"description"`
	Pattern         *AnonymizedPattern     `json:"pattern,omitempty"`
	SourceCount     int                    `json:"source_count"` // Number of repos contributing
	Confidence      float64                `json:"confidence"`   // 0-1 confidence score
	Relevance       float64                `json:"relevance"`    // 0-1 relevance to current repo
	Impact          ImpactMetrics          `json:"impact"`
	Applicability   []string               `json:"applicability"` // Project types this applies to
	Prerequisites   []string               `json:"prerequisites"`
	Recommendations []string               `json:"recommendations"`
	Tags            []string               `json:"tags"`
	Metadata        map[string]interface{} `json:"metadata"`
	GeneratedAt     time.Time              `json:"generated_at"`
	ValidUntil      time.Time              `json:"valid_until"`
	IsActionable    bool                   `json:"is_actionable"`
}

// AnonymizedPattern represents a pattern with sensitive information removed
type AnonymizedPattern struct {
	Type           string                 `json:"type"`
	Sequence       []string               `json:"sequence"` // Generalized task types
	Frequency      float64                `json:"frequency"`
	SuccessRate    float64                `json:"success_rate"`
	TimeMetrics    CycleTimeMetrics       `json:"time_metrics"`
	CommonKeywords []string               `json:"common_keywords"`
	ProjectTypes   []string               `json:"project_types"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// ImpactMetrics represents the potential impact of applying an insight
type ImpactMetrics struct {
	ProductivityGain   float64 `json:"productivity_gain"`   // Percentage improvement
	TimeReduction      float64 `json:"time_reduction"`      // Hours saved
	QualityImprovement float64 `json:"quality_improvement"` // Bug reduction %
	AdoptionRate       float64 `json:"adoption_rate"`       // % of repos using
}

// RepositorySimilarity represents similarity between two repositories
type RepositorySimilarity struct {
	RepositoryA    string                 `json:"repository_a"`
	RepositoryB    string                 `json:"repository_b"`
	Score          float64                `json:"score"` // 0-1 similarity
	Dimensions     SimilarityDimensions   `json:"dimensions"`
	SharedPatterns []string               `json:"shared_patterns"` // Pattern IDs
	LastCalculated time.Time              `json:"last_calculated"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// SimilarityDimensions represents different aspects of repository similarity
type SimilarityDimensions struct {
	ProjectType  float64 `json:"project_type"`
	TechStack    float64 `json:"tech_stack"`
	TeamSize     float64 `json:"team_size"`
	Complexity   float64 `json:"complexity"`
	Domain       float64 `json:"domain"`
	WorkPatterns float64 `json:"work_patterns"`
}

// AggregatedPattern represents a pattern aggregated from multiple repositories
type AggregatedPattern struct {
	Type         string                 `json:"type"`
	Sequence     []PatternStep          `json:"sequence"`
	Frequency    float64                `json:"frequency"`
	SuccessRate  float64                `json:"success_rate"`
	TimeMetrics  CycleTimeMetrics       `json:"time_metrics"`
	Keywords     []string               `json:"keywords"`
	ProjectTypes []string               `json:"project_types"`
	SourceCount  int                    `json:"source_count"`
	Confidence   float64                `json:"confidence"`
	Sources      []PatternSource        `json:"sources"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// PatternSource represents a source repository for a pattern
type PatternSource struct {
	Repository      string                 `json:"repository"`
	Weight          float64                `json:"weight"`
	SuccessRate     float64                `json:"success_rate"`
	Contribution    float64                `json:"contribution"`
	LastContributed time.Time              `json:"last_contributed"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// RepositoryCharacteristics represents analyzed characteristics of a repository
type RepositoryCharacteristics struct {
	ProjectType    ProjectType            `json:"project_type"`
	Languages      map[string]int         `json:"languages"`
	Frameworks     []string               `json:"frameworks"`
	Dependencies   []string               `json:"dependencies"`
	TeamSize       int                    `json:"team_size"`
	Complexity     float64                `json:"complexity"`
	Domain         string                 `json:"domain"`
	ActivityLevel  float64                `json:"activity_level"`
	QualityMetrics QualityMetrics         `json:"quality_metrics"`
	WorkPatterns   []WorkPattern          `json:"work_patterns"`
	Technologies   []Technology           `json:"technologies"`
	Metadata       map[string]interface{} `json:"metadata"`
	AnalyzedAt     time.Time              `json:"analyzed_at"`
}

// QualityMetrics represents quality-related metrics
type QualityMetrics struct {
	TestCoverage    float64 `json:"test_coverage"`
	BugRate         float64 `json:"bug_rate"`
	CodeQuality     float64 `json:"code_quality"`
	Documentation   float64 `json:"documentation"`
	Maintainability float64 `json:"maintainability"`
}

// WorkPattern represents a typical work pattern in a repository
type WorkPattern struct {
	Type        string    `json:"type"`
	Frequency   float64   `json:"frequency"`
	Duration    float64   `json:"duration"`
	SuccessRate float64   `json:"success_rate"`
	Context     []string  `json:"context"`
	LastSeen    time.Time `json:"last_seen"`
}

// Technology represents a technology used in a repository
type Technology struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"` // "language", "framework", "tool", "service"
	Version     string  `json:"version"`
	Usage       float64 `json:"usage"`       // Usage intensity 0-1
	Maturity    float64 `json:"maturity"`    // How mature the usage is 0-1
	Criticality float64 `json:"criticality"` // How critical to the project 0-1
}

// PrivacySettings represents privacy configuration for cross-repo learning
type PrivacySettings struct {
	SharePatterns      bool     `json:"share_patterns"`
	ShareMetrics       bool     `json:"share_metrics"`
	ShareTechnologies  bool     `json:"share_technologies"`
	ExcludeKeywords    []string `json:"exclude_keywords"`
	ExcludePatterns    []string `json:"exclude_patterns"`
	MinAnonymization   int      `json:"min_anonymization"`   // Min repos before sharing
	MaxDataAge         int      `json:"max_data_age"`        // Days
	AnonymizationLevel string   `json:"anonymization_level"` // "basic", "medium", "high"
}

// InsightRecommendation represents a specific recommendation from an insight
type InsightRecommendation struct {
	ID              string                 `json:"id"`
	InsightID       string                 `json:"insight_id"`
	Type            string                 `json:"type"`     // "action", "warning", "suggestion", "best_practice"
	Priority        string                 `json:"priority"` // "low", "medium", "high", "critical"
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Actions         []RecommendedAction    `json:"actions"`
	Prerequisites   []string               `json:"prerequisites"`
	EstimatedEffort string                 `json:"estimated_effort"`
	ExpectedImpact  string                 `json:"expected_impact"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// RecommendedAction represents a specific action to take
type RecommendedAction struct {
	Type        string                 `json:"type"` // "create_task", "modify_workflow", "adopt_tool"
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Order       int                    `json:"order"`
	Optional    bool                   `json:"optional"`
}

// Helper methods for CrossRepoInsight

// IsExpired checks if the insight is past its valid period
func (i *CrossRepoInsight) IsExpired() bool {
	return time.Now().After(i.ValidUntil)
}

// GetQualityScore calculates a quality score for the insight
func (i *CrossRepoInsight) GetQualityScore() float64 {
	score := 0.0

	// Source count factor (more sources = higher quality)
	sourceScore := float64(i.SourceCount) / 10.0 // Normalize to 10 sources
	if sourceScore > 1.0 {
		sourceScore = 1.0
	}
	score += sourceScore * 0.3

	// Confidence factor
	score += i.Confidence * 0.4

	// Relevance factor
	score += i.Relevance * 0.2

	// Actionability factor
	if i.IsActionable {
		score += 0.1
	}

	return score
}

// GetPriorityLevel returns a priority level based on quality and impact
func (i *CrossRepoInsight) GetPriorityLevel() string {
	qualityScore := i.GetQualityScore()
	impactScore := (i.Impact.ProductivityGain + i.Impact.QualityImprovement) / 2.0

	combinedScore := (qualityScore + impactScore) / 2.0

	switch {
	case combinedScore >= 0.8:
		return "critical"
	case combinedScore >= 0.6:
		return "high"
	case combinedScore >= 0.4:
		return "medium"
	default:
		return "low"
	}
}

// Helper methods for AggregatedPattern

// Merge combines this pattern with another pattern
func (ap *AggregatedPattern) Merge(other *TaskPattern, weight float64) {
	// Update source count
	ap.SourceCount++

	// Update success rate (weighted average)
	totalWeight := float64(ap.SourceCount)
	ap.SuccessRate = ((ap.SuccessRate * (totalWeight - 1)) + (other.SuccessRate * weight)) / totalWeight

	// Update frequency (weighted average)
	ap.Frequency = ((ap.Frequency * (totalWeight - 1)) + (other.Frequency * weight)) / totalWeight

	// Merge keywords
	keywordSet := make(map[string]bool)
	for _, keyword := range ap.Keywords {
		keywordSet[keyword] = true
	}
	if otherKeywords, ok := other.Metadata["keywords"].([]string); ok {
		for _, keyword := range otherKeywords {
			if !keywordSet[keyword] {
				ap.Keywords = append(ap.Keywords, keyword)
				keywordSet[keyword] = true
			}
		}
	}

	// Update project types
	if otherProjectType, ok := other.Metadata["project_type"].(string); ok {
		projectTypeSet := make(map[string]bool)
		for _, pt := range ap.ProjectTypes {
			projectTypeSet[pt] = true
		}
		if !projectTypeSet[otherProjectType] {
			ap.ProjectTypes = append(ap.ProjectTypes, otherProjectType)
		}
	}

	// Update confidence based on source count and consistency
	ap.Confidence = ap.calculateConfidence()

	// Update timestamp
	ap.UpdatedAt = time.Now()
}

// calculateConfidence calculates confidence based on source count and consistency
func (ap *AggregatedPattern) calculateConfidence() float64 {
	// Base confidence from source count
	sourceConfidence := float64(ap.SourceCount) / 20.0 // Normalize to 20 sources
	if sourceConfidence > 1.0 {
		sourceConfidence = 1.0
	}

	// Success rate factor
	successConfidence := ap.SuccessRate

	// Frequency factor (more frequent patterns are more confident)
	frequencyConfidence := ap.Frequency

	// Weighted combination
	confidence := (sourceConfidence * 0.4) + (successConfidence * 0.3) + (frequencyConfidence * 0.3)

	return confidence
}

// Helper methods for RepositorySimilarity

// GetOverallSimilarity calculates weighted overall similarity score
func (rs *RepositorySimilarity) GetOverallSimilarity() float64 {
	weights := map[string]float64{
		"project_type":  0.25,
		"tech_stack":    0.20,
		"team_size":     0.10,
		"complexity":    0.15,
		"domain":        0.15,
		"work_patterns": 0.15,
	}

	score := 0.0
	score += rs.Dimensions.ProjectType * weights["project_type"]
	score += rs.Dimensions.TechStack * weights["tech_stack"]
	score += rs.Dimensions.TeamSize * weights["team_size"]
	score += rs.Dimensions.Complexity * weights["complexity"]
	score += rs.Dimensions.Domain * weights["domain"]
	score += rs.Dimensions.WorkPatterns * weights["work_patterns"]

	return score
}

// IsRecentlyCalculated checks if the similarity was calculated recently
func (rs *RepositorySimilarity) IsRecentlyCalculated(threshold time.Duration) bool {
	return time.Since(rs.LastCalculated) < threshold
}

// Helper methods for PrivacySettings

// DefaultPrivacySettings returns default privacy settings
func DefaultPrivacySettings() *PrivacySettings {
	return &PrivacySettings{
		SharePatterns:      true,
		ShareMetrics:       true,
		ShareTechnologies:  true,
		ExcludeKeywords:    []string{"password", "secret", "key", "token", "private"},
		ExcludePatterns:    []string{"*_secret_*", "*_private_*", "*_internal_*"},
		MinAnonymization:   3,
		MaxDataAge:         30,
		AnonymizationLevel: "medium",
	}
}

// ShouldShare checks if a pattern should be shared based on settings
func (ps *PrivacySettings) ShouldShare(pattern *TaskPattern) bool {
	if !ps.SharePatterns {
		return false
	}

	// Check keywords
	if keywords, ok := pattern.Metadata["keywords"].([]string); ok {
		for _, keyword := range keywords {
			for _, excluded := range ps.ExcludeKeywords {
				if keyword == excluded {
					return false
				}
			}
		}
	}

	// Check pattern type against excluded patterns
	for _, excludePattern := range ps.ExcludePatterns {
		if string(pattern.Type) == excludePattern {
			return false
		}
	}

	// Check data age
	if ps.MaxDataAge > 0 {
		maxAge := time.Duration(ps.MaxDataAge) * 24 * time.Hour
		if time.Since(pattern.CreatedAt) > maxAge {
			return false
		}
	}

	return true
}

// Helper methods for sorting and filtering

// SortInsightsByQuality sorts insights by quality score (descending)
func SortInsightsByQuality(insights []*CrossRepoInsight) {
	sort.Slice(insights, func(i, j int) bool {
		return insights[i].GetQualityScore() > insights[j].GetQualityScore()
	})
}

// FilterInsightsByRelevance filters insights by minimum relevance threshold
func FilterInsightsByRelevance(insights []*CrossRepoInsight, minRelevance float64) []*CrossRepoInsight {
	var filtered []*CrossRepoInsight
	for _, insight := range insights {
		if insight.Relevance >= minRelevance {
			filtered = append(filtered, insight)
		}
	}
	return filtered
}

// FilterInsightsByType filters insights by type
func FilterInsightsByType(insights []*CrossRepoInsight, insightType InsightType) []*CrossRepoInsight {
	var filtered []*CrossRepoInsight
	for _, insight := range insights {
		if insight.Type == insightType {
			filtered = append(filtered, insight)
		}
	}
	return filtered
}
