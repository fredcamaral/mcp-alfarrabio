package entities

import (
	"fmt"
	"time"
)

// MetricType represents different types of metrics
type MetricType string

const (
	MetricTypeProductivity MetricType = "productivity"
	MetricTypeCompletion   MetricType = "completion"
	MetricTypeVelocity     MetricType = "velocity"
	MetricTypeCycleTime    MetricType = "cycle_time"
	MetricTypeEfficiency   MetricType = "efficiency"
	MetricTypeQuality      MetricType = "quality"
)

// TimePeriod represents a time range for analytics
type TimePeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// IsValid checks if the time period is valid
func (tp TimePeriod) IsValid() bool {
	return !tp.Start.IsZero() && !tp.End.IsZero() && tp.Start.Before(tp.End)
}

// Duration returns the duration of the time period
func (tp TimePeriod) Duration() time.Duration {
	return tp.End.Sub(tp.Start)
}

// Contains checks if a time is within the period
func (tp TimePeriod) Contains(t time.Time) bool {
	return t.After(tp.Start) && t.Before(tp.End)
}

// WorkflowMetrics represents comprehensive workflow analytics
type WorkflowMetrics struct {
	Repository   string                 `json:"repository"`
	Period       TimePeriod             `json:"period"`
	Productivity ProductivityMetrics    `json:"productivity"`
	Completion   CompletionMetrics      `json:"completion"`
	Velocity     VelocityMetrics        `json:"velocity"`
	CycleTime    CycleTimeMetrics       `json:"cycle_time"`
	Patterns     PatternMetrics         `json:"patterns"`
	Bottlenecks  []Bottleneck           `json:"bottlenecks"`
	Trends       TrendAnalysis          `json:"trends"`
	Comparisons  *PeriodComparison      `json:"comparisons,omitempty"`
	GeneratedAt  time.Time              `json:"generated_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ProductivityMetrics represents productivity-related metrics
type ProductivityMetrics struct {
	Score           float64            `json:"score"` // 0-100 overall score
	TasksPerDay     float64            `json:"tasks_per_day"`
	FocusTime       time.Duration      `json:"focus_time"`       // Uninterrupted work time
	PeakHours       []int              `json:"peak_hours"`       // Most productive hours
	ByPriority      map[string]float64 `json:"by_priority"`      // Completion by priority
	ByType          map[string]float64 `json:"by_type"`          // Completion by task type
	ContextSwitches int                `json:"context_switches"` // Number of context switches
	DeepWorkRatio   float64            `json:"deep_work_ratio"`  // Ratio of deep work time
}

// CompletionMetrics represents task completion analytics
type CompletionMetrics struct {
	TotalTasks     int            `json:"total_tasks"`
	Completed      int            `json:"completed"`
	InProgress     int            `json:"in_progress"`
	Cancelled      int            `json:"cancelled"`
	CompletionRate float64        `json:"completion_rate"`
	AverageTime    time.Duration  `json:"average_time"`
	ByStatus       map[string]int `json:"by_status"`
	ByPriority     map[string]int `json:"by_priority"`
	OnTimeRate     float64        `json:"on_time_rate"`
	QualityScore   float64        `json:"quality_score"`
}

// VelocityMetrics represents velocity and throughput metrics
type VelocityMetrics struct {
	CurrentVelocity float64          `json:"current_velocity"` // Tasks/week
	TrendDirection  string           `json:"trend_direction"`  // up, down, stable
	TrendPercentage float64          `json:"trend_percentage"`
	ByWeek          []WeeklyVelocity `json:"by_week"`
	Forecast        VelocityForecast `json:"forecast"`
	Consistency     float64          `json:"consistency"` // Velocity consistency score
}

// WeeklyVelocity represents velocity for a specific week
type WeeklyVelocity struct {
	Number    int       `json:"number"` // Week number
	Year      int       `json:"year"`
	Velocity  float64   `json:"velocity"`
	Tasks     int       `json:"tasks"`
	StartDate time.Time `json:"start_date"`
}

// VelocityForecast represents velocity predictions
type VelocityForecast struct {
	PredictedVelocity float64   `json:"predicted_velocity"`
	Confidence        float64   `json:"confidence"`
	Range             []float64 `json:"range"` // [min, max]
	Method            string    `json:"method"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CycleTimeMetrics represents cycle time analytics
type CycleTimeMetrics struct {
	AverageCycleTime time.Duration            `json:"average_cycle_time"`
	MedianCycleTime  time.Duration            `json:"median_cycle_time"`
	P90CycleTime     time.Duration            `json:"p90_cycle_time"`
	ByType           map[string]time.Duration `json:"by_type"`
	ByPriority       map[string]time.Duration `json:"by_priority"`
	Distribution     []CycleTimePoint         `json:"distribution"`
	LeadTime         time.Duration            `json:"lead_time"`
	WaitTime         time.Duration            `json:"wait_time"`
}

// CycleTimePoint represents a point in cycle time distribution
type CycleTimePoint struct {
	Duration   time.Duration `json:"duration"`
	Count      int           `json:"count"`
	Percentile float64       `json:"percentile"`
}

// PatternMetrics represents pattern-related analytics
type PatternMetrics struct {
	TotalPatterns    int                `json:"total_patterns"`
	ActivePatterns   int                `json:"active_patterns"`
	PatternUsage     map[string]int     `json:"pattern_usage"`
	SuccessRates     map[string]float64 `json:"success_rates"`
	TopPatterns      []PatternUsage     `json:"top_patterns"`
	PatternEvolution []PatternEvolution `json:"pattern_evolution"`
	AdherenceRate    float64            `json:"adherence_rate"`
}

// PatternUsage represents usage statistics for a pattern
type PatternUsage struct {
	PatternID   string    `json:"pattern_id"`
	PatternName string    `json:"pattern_name"`
	UsageCount  int       `json:"usage_count"`
	SuccessRate float64   `json:"success_rate"`
	LastUsed    time.Time `json:"last_used"`
}

// PatternEvolution tracks how patterns change over time
type PatternEvolution struct {
	Period      TimePeriod `json:"period"`
	NewPatterns int        `json:"new_patterns"`
	Evolved     int        `json:"evolved"`
	Deprecated  int        `json:"deprecated"`
}

// Bottleneck represents a workflow bottleneck
type Bottleneck struct {
	Type          string                 `json:"type"` // task_type, time_of_day, dependency
	Description   string                 `json:"description"`
	Impact        float64                `json:"impact"`    // Hours lost
	Frequency     int                    `json:"frequency"` // Occurrences
	Severity      BottleneckSeverity     `json:"severity"`
	Suggestions   []string               `json:"suggestions"`
	AffectedTasks []string               `json:"affected_tasks"`
	DetectedAt    time.Time              `json:"detected_at"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// BottleneckSeverity represents bottleneck severity levels
type BottleneckSeverity string

const (
	BottleneckSeverityLow      BottleneckSeverity = "low"
	BottleneckSeverityMedium   BottleneckSeverity = "medium"
	BottleneckSeverityHigh     BottleneckSeverity = "high"
	BottleneckSeverityCritical BottleneckSeverity = "critical"
)

// TrendAnalysis represents trend analysis across metrics
type TrendAnalysis struct {
	ProductivityTrend Trend        `json:"productivity_trend"`
	VelocityTrend     Trend        `json:"velocity_trend"`
	QualityTrend      Trend        `json:"quality_trend"`
	EfficiencyTrend   Trend        `json:"efficiency_trend"`
	Predictions       []Prediction `json:"predictions"`
	Seasonality       Seasonality  `json:"seasonality"`
}

// Trend represents a trend in a specific metric
type Trend struct {
	Direction   TrendDirection `json:"direction"`
	Strength    float64        `json:"strength"`    // 0-1
	Confidence  float64        `json:"confidence"`  // 0-1
	ChangeRate  float64        `json:"change_rate"` // % change per period
	StartValue  float64        `json:"start_value"`
	EndValue    float64        `json:"end_value"`
	TrendLine   []TrendPoint   `json:"trend_line"`
	Description string         `json:"description"`
}

// TrendDirection represents the direction of a trend
type TrendDirection string

const (
	TrendDirectionUp       TrendDirection = "up"
	TrendDirectionDown     TrendDirection = "down"
	TrendDirectionStable   TrendDirection = "stable"
	TrendDirectionVolatile TrendDirection = "volatile"
)

// TrendPoint represents a point on a trend line
type TrendPoint struct {
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

// Prediction represents a future prediction
type Prediction struct {
	Metric      MetricType `json:"metric"`
	Period      TimePeriod `json:"period"`
	Value       float64    `json:"value"`
	Confidence  float64    `json:"confidence"`
	Range       []float64  `json:"range"` // [min, max]
	Method      string     `json:"method"`
	Assumptions []string   `json:"assumptions"`
}

// Seasonality represents seasonal patterns
type Seasonality struct {
	HasSeasonality bool               `json:"has_seasonality"`
	Patterns       []SeasonalPattern  `json:"patterns"`
	WeeklyPattern  map[string]float64 `json:"weekly_pattern"`  // day -> multiplier
	MonthlyPattern map[string]float64 `json:"monthly_pattern"` // month -> multiplier
	HourlyPattern  map[int]float64    `json:"hourly_pattern"`  // hour -> multiplier
}

// SeasonalPattern represents a detected seasonal pattern
type SeasonalPattern struct {
	Type        string  `json:"type"`      // weekly, monthly, quarterly
	Period      int     `json:"period"`    // period length
	Amplitude   float64 `json:"amplitude"` // pattern strength
	Phase       float64 `json:"phase"`     // pattern phase
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description"`
}

// PeriodComparison represents a comparison between two periods
type PeriodComparison struct {
	PeriodA          TimePeriod             `json:"period_a"`
	PeriodB          TimePeriod             `json:"period_b"`
	ProductivityDiff float64                `json:"productivity_diff"`
	VelocityDiff     float64                `json:"velocity_diff"`
	QualityDiff      float64                `json:"quality_diff"`
	CompletionDiff   float64                `json:"completion_diff"`
	Improvements     []string               `json:"improvements"`
	Regressions      []string               `json:"regressions"`
	Summary          string                 `json:"summary"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// ProductivityReport represents a comprehensive productivity report
type ProductivityReport struct {
	Repository      string                `json:"repository"`
	Period          TimePeriod            `json:"period"`
	OverallScore    float64               `json:"overall_score"`
	Metrics         WorkflowMetrics       `json:"metrics"`
	Insights        []ProductivityInsight `json:"insights"`
	Recommendations []Recommendation      `json:"recommendations"`
	Charts          map[string]ChartData  `json:"charts"`
	GeneratedAt     time.Time             `json:"generated_at"`
}

// ProductivityInsight represents an insight about productivity
type ProductivityInsight struct {
	Type        InsightType `json:"type"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Impact      float64     `json:"impact"`
	Confidence  float64     `json:"confidence"`
	Evidence    []string    `json:"evidence"`
	ActionItems []string    `json:"action_items"`
}

// Recommendation represents an actionable recommendation
type Recommendation struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Priority    RecommendationPriority `json:"priority"`
	Impact      float64                `json:"impact"`
	Effort      float64                `json:"effort"`
	Category    string                 `json:"category"`
	Actions     []string               `json:"actions"`
	Evidence    []string               `json:"evidence"`
	CreatedAt   time.Time              `json:"created_at"`
}

// RecommendationPriority represents recommendation priority levels
type RecommendationPriority string

const (
	RecommendationPriorityLow      RecommendationPriority = "low"
	RecommendationPriorityMedium   RecommendationPriority = "medium"
	RecommendationPriorityHigh     RecommendationPriority = "high"
	RecommendationPriorityCritical RecommendationPriority = "critical"
)

// ChartData represents data for visualization
type ChartData struct {
	Type   ChartType              `json:"type"`
	Title  string                 `json:"title"`
	Data   map[string]interface{} `json:"data"`
	Config map[string]interface{} `json:"config"`
}

// ChartType represents different types of charts
type ChartType string

const (
	ChartTypeBar       ChartType = "bar"
	ChartTypeLine      ChartType = "line"
	ChartTypePie       ChartType = "pie"
	ChartTypeHeatmap   ChartType = "heatmap"
	ChartTypeScatter   ChartType = "scatter"
	ChartTypeGantt     ChartType = "gantt"
	ChartTypeHistogram ChartType = "histogram"
	ChartTypeProgress  ChartType = "progress"
)

// VisFormat represents visualization format types
type VisFormat string

const (
	VisFormatTerminal VisFormat = "terminal"
	VisFormatHTML     VisFormat = "html"
	VisFormatSVG      VisFormat = "svg"
	VisFormatPNG      VisFormat = "png"
)

// ExportFormat represents export format types
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatPDF  ExportFormat = "pdf"
	ExportFormatHTML ExportFormat = "html"
)

// GetQualityScore calculates a quality score based on completion metrics
func (cm *CompletionMetrics) GetQualityScore() float64 {
	if cm.TotalTasks == 0 {
		return 0.0
	}

	// Quality = (completion rate * 0.4) + (on-time rate * 0.4) + (quality score * 0.2)
	return (cm.CompletionRate * 0.4) + (cm.OnTimeRate * 0.4) + (cm.QualityScore * 0.2)
}

// GetEfficiencyScore calculates efficiency from cycle time metrics
func (ct *CycleTimeMetrics) GetEfficiencyScore() float64 {
	if ct.AverageCycleTime == 0 {
		return 0.0
	}

	// Efficiency based on cycle time vs lead time ratio
	if ct.LeadTime == 0 {
		return 1.0
	}

	efficiency := float64(ct.AverageCycleTime) / float64(ct.LeadTime)
	if efficiency > 1.0 {
		efficiency = 1.0
	}

	return efficiency
}

// GetOverallScore calculates an overall workflow score
func (wm *WorkflowMetrics) GetOverallScore() float64 {
	productivity := wm.Productivity.Score / 100.0
	completion := wm.Completion.GetQualityScore()
	efficiency := wm.CycleTime.GetEfficiencyScore()
	velocity := wm.Velocity.Consistency

	// Weighted average
	return (productivity * 0.3) + (completion * 0.3) + (efficiency * 0.2) + (velocity * 0.2)
}

// IsHighImpact checks if a bottleneck is high impact
func (b *Bottleneck) IsHighImpact() bool {
	return b.Severity == BottleneckSeverityHigh || b.Severity == BottleneckSeverityCritical
}

// GetImpactDescription returns a human-readable impact description
func (b *Bottleneck) GetImpactDescription() string {
	if b.Impact < 1 {
		return "Low impact"
	} else if b.Impact < 8 {
		return "Medium impact"
	} else if b.Impact < 24 {
		return "High impact"
	}
	return "Critical impact"
}

// IsSignificant checks if a trend is significant
func (t *Trend) IsSignificant() bool {
	return t.Confidence > 0.7 && t.Strength > 0.3
}

// GetTrendDescription returns a human-readable trend description
func (t *Trend) GetTrendDescription() string {
	if !t.IsSignificant() {
		return "No significant trend"
	}

	strength := "slight"
	if t.Strength > 0.7 {
		strength = "strong"
	} else if t.Strength > 0.5 {
		strength = "moderate"
	}

	return fmt.Sprintf("%s %s trend", strength, t.Direction)
}
