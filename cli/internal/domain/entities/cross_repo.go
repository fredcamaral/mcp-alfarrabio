package entities

import (
	"time"
)

// CrossRepoAnalysis represents analysis across multiple repositories
type CrossRepoAnalysis struct {
	Repositories     []string                  `json:"repositories"`
	Period           TimePeriod                `json:"period"`
	Comparisons      []RepositoryComparison    `json:"comparisons"`
	CommonPatterns   []CommonPattern           `json:"common_patterns"`
	BestPractices    []BestPractice            `json:"best_practices"`
	Outliers         []Outlier                 `json:"outliers"`
	Recommendations  []CrossRepoRecommendation `json:"recommendations"`
	Insights         []CrossRepoInsight        `json:"insights"`
	AggregateMetrics AggregateMetrics          `json:"aggregate_metrics"`
	GeneratedAt      time.Time                 `json:"generated_at"`
	Metadata         map[string]interface{}    `json:"metadata"`
}

// RepositoryComparison represents comparison data between repositories
type RepositoryComparison struct {
	RepositoryA      string                 `json:"repository_a"`
	RepositoryB      string                 `json:"repository_b"`
	ProductivityDiff float64                `json:"productivity_diff"`
	VelocityDiff     float64                `json:"velocity_diff"`
	QualityDiff      float64                `json:"quality_diff"`
	EfficiencyDiff   float64                `json:"efficiency_diff"`
	SimilarityScore  float64                `json:"similarity_score"`
	KeyDifferences   []string               `json:"key_differences"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// CommonPattern represents patterns found across multiple repositories
type CommonPattern struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Type              PatternType            `json:"type"`
	Frequency         float64                `json:"frequency"`
	SuccessRate       float64                `json:"success_rate"`
	RepositoryCount   int                    `json:"repository_count"`
	Repositories      []string               `json:"repositories"`
	Sequence          []PatternStep          `json:"sequence"`
	EstimatedDuration time.Duration          `json:"estimated_duration"`
	Confidence        float64                `json:"confidence"`
	Benefits          []string               `json:"benefits"`
	Prerequisites     []string               `json:"prerequisites"`
	FirstDetected     time.Time              `json:"first_detected"`
	LastSeen          time.Time              `json:"last_seen"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// BestPractice represents a recommended practice found across repositories
type BestPractice struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	Impact          float64                `json:"impact"`
	AdoptionRate    float64                `json:"adoption_rate"`
	SuccessRate     float64                `json:"success_rate"`
	Repositories    []string               `json:"repositories"`
	Evidence        []string               `json:"evidence"`
	Implementation  []string               `json:"implementation"`
	Benefits        []string               `json:"benefits"`
	Challenges      []string               `json:"challenges"`
	RelatedPatterns []string               `json:"related_patterns"`
	Confidence      float64                `json:"confidence"`
	Priority        RecommendationPriority `json:"priority"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// Outlier represents unusual behavior or performance in repositories
type Outlier struct {
	ID              string                 `json:"id"`
	Repository      string                 `json:"repository"`
	Type            string                 `json:"type"`
	Description     string                 `json:"description"`
	Value           float64                `json:"value"`
	ExpectedValue   float64                `json:"expected_value"`
	Deviation       float64                `json:"deviation"`
	Severity        OutlierSeverity        `json:"severity"`
	Metric          MetricType             `json:"metric"`
	PossibleCauses  []string               `json:"possible_causes"`
	Recommendations []string               `json:"recommendations"`
	DetectedAt      time.Time              `json:"detected_at"`
	LastOccurrence  time.Time              `json:"last_occurrence"`
	Frequency       int                    `json:"frequency"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// OutlierSeverity represents the severity level of an outlier
type OutlierSeverity string

const (
	OutlierSeverityLow      OutlierSeverity = "low"
	OutlierSeverityMedium   OutlierSeverity = "medium"
	OutlierSeverityHigh     OutlierSeverity = "high"
	OutlierSeverityCritical OutlierSeverity = "critical"
)

// CrossRepoRecommendation represents recommendations based on cross-repository analysis
type CrossRepoRecommendation struct {
	ID                 string                 `json:"id"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	Category           string                 `json:"category"`
	Priority           RecommendationPriority `json:"priority"`
	Impact             float64                `json:"impact"`
	Effort             float64                `json:"effort"`
	TargetRepositories []string               `json:"target_repositories"`
	SourceRepositories []string               `json:"source_repositories"`
	Actions            []string               `json:"actions"`
	Evidence           []string               `json:"evidence"`
	Benefits           []string               `json:"benefits"`
	Risks              []string               `json:"risks"`
	Timeline           string                 `json:"timeline"`
	Dependencies       []string               `json:"dependencies"`
	SuccessMetrics     []string               `json:"success_metrics"`
	Confidence         float64                `json:"confidence"`
	CreatedAt          time.Time              `json:"created_at"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// Note: CrossRepoInsight is defined in insights.go

// AggregateMetrics represents aggregated metrics across repositories
type AggregateMetrics struct {
	TotalRepositories int                       `json:"total_repositories"`
	AvgProductivity   float64                   `json:"avg_productivity"`
	AvgVelocity       float64                   `json:"avg_velocity"`
	AvgQuality        float64                   `json:"avg_quality"`
	AvgEfficiency     float64                   `json:"avg_efficiency"`
	TotalTasks        int                       `json:"total_tasks"`
	TotalPRs          int                       `json:"total_prs"`
	AvgCycleTime      time.Duration             `json:"avg_cycle_time"`
	AvgLeadTime       time.Duration             `json:"avg_lead_time"`
	TopPerformers     []string                  `json:"top_performers"`
	UnderPerformers   []string                  `json:"under_performers"`
	ConsistencyScore  float64                   `json:"consistency_score"`
	Variance          float64                   `json:"variance"`
	Trends            map[string]TrendDirection `json:"trends"`
	Metadata          map[string]interface{}    `json:"metadata"`
}

// Note: InsightType and related constants are defined in insights.go

// Additional insight types for cross-repository analysis
const (
	InsightTypePerformance     InsightType = "performance"
	InsightTypeAnomaly         InsightType = "anomaly"
	InsightTypeOpportunity     InsightType = "opportunity"
	InsightTypeBottleneck      InsightType = "bottleneck"
	InsightTypeComparison      InsightType = "comparison"
	InsightCategoryPerformance InsightType = "performance" // Alias for backward compatibility
)
