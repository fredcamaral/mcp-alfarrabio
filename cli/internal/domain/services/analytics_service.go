package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// AnalyticsService interface defines workflow analytics capabilities
type AnalyticsService interface {
	// Core analytics operations
	GetWorkflowMetrics(ctx context.Context, repository string, period entities.TimePeriod) (*entities.WorkflowMetrics, error)
	GetProductivityReport(ctx context.Context, repository string, period entities.TimePeriod) (*entities.ProductivityReport, error)

	// Bottleneck analysis
	DetectBottlenecks(ctx context.Context, repository string, period entities.TimePeriod) ([]*entities.Bottleneck, error)

	// Comparative analysis
	ComparePeriods(ctx context.Context, repository string, period1, period2 entities.TimePeriod) (*entities.PeriodComparison, error)

	// Visualization and export
	GenerateVisualization(ctx context.Context, metrics *entities.WorkflowMetrics, format entities.VisFormat) ([]byte, error)
	ExportAnalytics(ctx context.Context, repository string, period entities.TimePeriod, format entities.ExportFormat) (string, error)

	// Insights and recommendations
	GenerateInsights(ctx context.Context, metrics *entities.WorkflowMetrics) ([]*entities.ProductivityInsight, error)
	GenerateRecommendations(ctx context.Context, metrics *entities.WorkflowMetrics) ([]*entities.Recommendation, error)
}

// AnalyticsServiceConfig holds configuration for analytics service
type AnalyticsServiceConfig struct {
	DefaultPeriod        time.Duration `json:"default_period"`
	MinTasksForAnalysis  int           `json:"min_tasks_for_analysis"`
	BottleneckThreshold  float64       `json:"bottleneck_threshold"`
	TrendConfidenceMin   float64       `json:"trend_confidence_min"`
	VelocityPeriodWeeks  int           `json:"velocity_period_weeks"`
	SeasonalityMinPeriod time.Duration `json:"seasonality_min_period"`
	CacheResults         bool          `json:"cache_results"`
	CacheTTL             time.Duration `json:"cache_ttl"`
}

// DefaultAnalyticsServiceConfig returns default configuration
func DefaultAnalyticsServiceConfig() *AnalyticsServiceConfig {
	return &AnalyticsServiceConfig{
		DefaultPeriod:        30 * 24 * time.Hour,
		MinTasksForAnalysis:  5,
		BottleneckThreshold:  2.0, // 2+ hours lost
		TrendConfidenceMin:   0.6,
		VelocityPeriodWeeks:  12,
		SeasonalityMinPeriod: 90 * 24 * time.Hour,
		CacheResults:         true,
		CacheTTL:             1 * time.Hour,
	}
}

// AnalyticsServiceDependencies holds dependencies for analytics service
type AnalyticsServiceDependencies struct {
	TaskStore    TaskStorage
	PatternStore ports.PatternStorage
	SessionStore SessionStorage
	Visualizer   Visualizer
	Exporter     AnalyticsExporter
	Calculator   *MetricsCalculator
	Config       *AnalyticsServiceConfig
	Logger       *slog.Logger
}

// analyticsServiceImpl implements the AnalyticsService interface
type analyticsServiceImpl struct {
	taskStore    TaskStorage
	patternStore ports.PatternStorage
	sessionStore SessionStorage
	visualizer   Visualizer
	exporter     AnalyticsExporter
	calculator   *MetricsCalculator
	config       *AnalyticsServiceConfig
	logger       *slog.Logger
	cache        map[string]*cacheEntry
}

type cacheEntry struct {
	data      interface{}
	timestamp time.Time
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(deps AnalyticsServiceDependencies) AnalyticsService {
	if deps.Config == nil {
		deps.Config = DefaultAnalyticsServiceConfig()
	}
	if deps.Calculator == nil {
		deps.Calculator = NewMetricsCalculator()
	}

	return &analyticsServiceImpl{
		taskStore:    deps.TaskStore,
		patternStore: deps.PatternStore,
		sessionStore: deps.SessionStore,
		visualizer:   deps.Visualizer,
		exporter:     deps.Exporter,
		calculator:   deps.Calculator,
		config:       deps.Config,
		logger:       deps.Logger,
		cache:        make(map[string]*cacheEntry),
	}
}

// GetWorkflowMetrics generates comprehensive workflow metrics
func (a *analyticsServiceImpl) GetWorkflowMetrics(
	ctx context.Context,
	repository string,
	period entities.TimePeriod,
) (*entities.WorkflowMetrics, error) {
	a.logger.Info("generating workflow metrics",
		slog.String("repository", repository),
		slog.Time("period_start", period.Start),
		slog.Time("period_end", period.End))

	// Check cache
	cacheKey := fmt.Sprintf("metrics:%s:%d:%d", repository, period.Start.Unix(), period.End.Unix())
	if a.config.CacheResults {
		if entry, exists := a.cache[cacheKey]; exists {
			if time.Since(entry.timestamp) < a.config.CacheTTL {
				if metrics, ok := entry.data.(*entities.WorkflowMetrics); ok {
					return metrics, nil
				}
			}
		}
	}

	// Get data for the period
	tasks, err := a.taskStore.GetByPeriod(ctx, repository, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(tasks) < a.config.MinTasksForAnalysis {
		a.logger.Warn("insufficient tasks for analysis",
			slog.Int("task_count", len(tasks)),
			slog.Int("min_required", a.config.MinTasksForAnalysis))
	}

	sessions, err := a.sessionStore.GetByPeriod(ctx, repository, period)
	if err != nil {
		a.logger.Warn("failed to get sessions", slog.Any("error", err))
		sessions = []*entities.Session{} // Continue without sessions
	}

	patterns, err := a.patternStore.GetByRepository(ctx, repository)
	if err != nil {
		a.logger.Warn("failed to get patterns", slog.Any("error", err))
		patterns = []*entities.TaskPattern{} // Continue without patterns
	}

	// Create metrics structure
	metrics := &entities.WorkflowMetrics{
		Repository:  repository,
		Period:      period,
		GeneratedAt: time.Now(),
		Metadata:    make(map[string]interface{}),
	}

	// Calculate productivity metrics
	metrics.Productivity = a.calculateProductivityMetrics(tasks, sessions)

	// Calculate completion metrics
	metrics.Completion = a.calculateCompletionMetrics(tasks)

	// Calculate velocity metrics
	metrics.Velocity = a.calculateVelocityMetrics(tasks, period)

	// Calculate cycle time metrics
	metrics.CycleTime = a.calculateCycleTimeMetrics(tasks)

	// Calculate pattern metrics
	metrics.Patterns = a.calculatePatternMetrics(patterns, tasks)

	// Detect bottlenecks
	bottleneckPointers := a.detectBottlenecksInternal(tasks, sessions)
	metrics.Bottlenecks = make([]entities.Bottleneck, len(bottleneckPointers))
	for i, b := range bottleneckPointers {
		metrics.Bottlenecks[i] = *b
	}

	// Analyze trends
	metrics.Trends = a.analyzeTrends(ctx, repository, period, metrics)

	// Add metadata
	metrics.Metadata["task_count"] = len(tasks)
	metrics.Metadata["session_count"] = len(sessions)
	metrics.Metadata["pattern_count"] = len(patterns)

	// Cache results
	if a.config.CacheResults {
		a.cache[cacheKey] = &cacheEntry{
			data:      metrics,
			timestamp: time.Now(),
		}
	}

	a.logger.Info("workflow metrics generated",
		slog.Float64("overall_score", metrics.GetOverallScore()),
		slog.Float64("productivity_score", metrics.Productivity.Score))

	return metrics, nil
}

// GetProductivityReport generates a comprehensive productivity report
func (a *analyticsServiceImpl) GetProductivityReport(
	ctx context.Context,
	repository string,
	period entities.TimePeriod,
) (*entities.ProductivityReport, error) {
	// Get workflow metrics
	metrics, err := a.GetWorkflowMetrics(ctx, repository, period)
	if err != nil {
		return nil, err
	}

	// Generate insights
	insightPointers, err := a.GenerateInsights(ctx, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to generate insights: %w", err)
	}

	// Generate recommendations
	recommendationPointers, err := a.GenerateRecommendations(ctx, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recommendations: %w", err)
	}

	// Convert pointer slices to value slices
	insights := make([]entities.ProductivityInsight, len(insightPointers))
	for i, insight := range insightPointers {
		insights[i] = *insight
	}

	recommendations := make([]entities.Recommendation, len(recommendationPointers))
	for i, rec := range recommendationPointers {
		recommendations[i] = *rec
	}

	// Generate charts
	charts := a.generateCharts(metrics)

	report := &entities.ProductivityReport{
		Repository:      repository,
		Period:          period,
		OverallScore:    metrics.GetOverallScore() * 100,
		Metrics:         *metrics,
		Insights:        insights,
		Recommendations: recommendations,
		Charts:          charts,
		GeneratedAt:     time.Now(),
	}

	return report, nil
}

// DetectBottlenecks identifies workflow bottlenecks
func (a *analyticsServiceImpl) DetectBottlenecks(
	ctx context.Context,
	repository string,
	period entities.TimePeriod,
) ([]*entities.Bottleneck, error) {
	tasks, err := a.taskStore.GetByPeriod(ctx, repository, period)
	if err != nil {
		return nil, err
	}

	sessions, err := a.sessionStore.GetByPeriod(ctx, repository, period)
	if err != nil {
		sessions = []*entities.Session{} // Continue without sessions
	}

	return a.detectBottlenecksInternal(tasks, sessions), nil
}

// ComparePeriods compares metrics between two time periods
func (a *analyticsServiceImpl) ComparePeriods(
	ctx context.Context,
	repository string,
	period1, period2 entities.TimePeriod,
) (*entities.PeriodComparison, error) {
	// Get metrics for both periods
	metrics1, err := a.GetWorkflowMetrics(ctx, repository, period1)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics for period 1: %w", err)
	}

	metrics2, err := a.GetWorkflowMetrics(ctx, repository, period2)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics for period 2: %w", err)
	}

	// Calculate differences
	comparison := &entities.PeriodComparison{
		PeriodA:          period1,
		PeriodB:          period2,
		ProductivityDiff: metrics2.Productivity.Score - metrics1.Productivity.Score,
		VelocityDiff:     metrics2.Velocity.CurrentVelocity - metrics1.Velocity.CurrentVelocity,
		QualityDiff:      metrics2.Completion.GetQualityScore() - metrics1.Completion.GetQualityScore(),
		CompletionDiff:   metrics2.Completion.CompletionRate - metrics1.Completion.CompletionRate,
		Improvements:     []string{},
		Regressions:      []string{},
		Metadata:         make(map[string]interface{}),
	}

	// Identify improvements and regressions
	if comparison.ProductivityDiff > 5 {
		comparison.Improvements = append(comparison.Improvements,
			fmt.Sprintf("Productivity increased by %.1f points", comparison.ProductivityDiff))
	} else if comparison.ProductivityDiff < -5 {
		comparison.Regressions = append(comparison.Regressions,
			fmt.Sprintf("Productivity decreased by %.1f points", -comparison.ProductivityDiff))
	}

	if comparison.VelocityDiff > 0.5 {
		comparison.Improvements = append(comparison.Improvements,
			fmt.Sprintf("Velocity increased by %.1f tasks/week", comparison.VelocityDiff))
	} else if comparison.VelocityDiff < -0.5 {
		comparison.Regressions = append(comparison.Regressions,
			fmt.Sprintf("Velocity decreased by %.1f tasks/week", -comparison.VelocityDiff))
	}

	// Generate summary
	if len(comparison.Improvements) > len(comparison.Regressions) {
		comparison.Summary = "Overall improvement in productivity metrics"
	} else if len(comparison.Regressions) > len(comparison.Improvements) {
		comparison.Summary = "Some regression in productivity metrics"
	} else {
		comparison.Summary = "Mixed results with balanced improvements and regressions"
	}

	return comparison, nil
}

// GenerateVisualization creates visualizations for metrics
func (a *analyticsServiceImpl) GenerateVisualization(
	ctx context.Context,
	metrics *entities.WorkflowMetrics,
	format entities.VisFormat,
) ([]byte, error) {
	if a.visualizer == nil {
		return nil, errors.New("visualizer not configured")
	}

	return a.visualizer.GenerateVisualization(metrics, format)
}

// ExportAnalytics exports analytics data in specified format
func (a *analyticsServiceImpl) ExportAnalytics(
	ctx context.Context,
	repository string,
	period entities.TimePeriod,
	format entities.ExportFormat,
) (string, error) {
	if a.exporter == nil {
		return "", errors.New("exporter not configured")
	}

	metrics, err := a.GetWorkflowMetrics(ctx, repository, period)
	if err != nil {
		return "", err
	}

	return a.exporter.Export(metrics, format)
}

// Helper methods for metric calculations

func (a *analyticsServiceImpl) calculateProductivityMetrics(
	tasks []*entities.Task,
	sessions []*entities.Session,
) entities.ProductivityMetrics {
	metrics := entities.ProductivityMetrics{
		ByPriority: make(map[string]float64),
		ByType:     make(map[string]float64),
	}

	if len(tasks) == 0 {
		return metrics
	}

	// Calculate tasks per day
	if len(sessions) > 0 {
		activeDays := a.calculator.CountActiveDays(sessions)
		completedTasks := a.calculator.CountCompletedTasks(tasks)
		metrics.TasksPerDay = float64(completedTasks) / float64(activeDays)
	}

	// Calculate focus time and peak hours
	if len(sessions) > 0 {
		metrics.FocusTime = a.calculator.CalculateAverageFocusTime(sessions)
		metrics.PeakHours = a.calculator.FindPeakHours(sessions)
		metrics.ContextSwitches = a.calculator.CountContextSwitches(sessions)
		metrics.DeepWorkRatio = a.calculator.CalculateDeepWorkRatio(sessions)
	}

	// Calculate completion rates by priority and type
	metrics.ByPriority = a.calculator.CalculateCompletionByPriority(tasks)
	metrics.ByType = a.calculator.CalculateCompletionByType(tasks)

	// Calculate overall productivity score
	metrics.Score = a.calculator.CalculateProductivityScore(metrics, tasks, sessions)

	return metrics
}

func (a *analyticsServiceImpl) calculateCompletionMetrics(
	tasks []*entities.Task,
) entities.CompletionMetrics {
	metrics := entities.CompletionMetrics{
		ByStatus:   make(map[string]int),
		ByPriority: make(map[string]int),
	}

	if len(tasks) == 0 {
		return metrics
	}

	// Count tasks by status
	for _, task := range tasks {
		metrics.TotalTasks++
		metrics.ByStatus[string(task.Status)]++

		switch string(task.Status) {
		case string(entities.StatusCompleted):
			metrics.Completed++
		case string(entities.StatusInProgress):
			metrics.InProgress++
		case string(entities.StatusCancelled):
			metrics.Cancelled++
		}

		metrics.ByPriority[string(task.Priority)]++
	}

	// Calculate rates
	if metrics.TotalTasks > 0 {
		metrics.CompletionRate = float64(metrics.Completed) / float64(metrics.TotalTasks)
	}

	// Calculate average completion time
	completedTasks := a.filterTasksByStatus(tasks, string(entities.StatusCompleted))
	if len(completedTasks) > 0 {
		totalTime := time.Duration(0)
		onTimeCount := 0

		for _, task := range completedTasks {
			if duration := a.calculator.GetTaskDuration(task); duration > 0 {
				totalTime += duration
			}

			if a.calculator.IsTaskOnTime(task) {
				onTimeCount++
			}
		}

		if len(completedTasks) > 0 {
			metrics.AverageTime = totalTime / time.Duration(len(completedTasks))
			metrics.OnTimeRate = float64(onTimeCount) / float64(len(completedTasks))
		}
	}

	// Calculate quality score (placeholder - would be based on actual quality metrics)
	metrics.QualityScore = a.calculator.CalculateQualityScore(tasks)

	return metrics
}

func (a *analyticsServiceImpl) calculateVelocityMetrics(
	tasks []*entities.Task,
	period entities.TimePeriod,
) entities.VelocityMetrics {
	metrics := entities.VelocityMetrics{
		ByWeek: []entities.WeeklyVelocity{},
	}

	if len(tasks) == 0 {
		return metrics
	}

	// Group tasks by week
	weeklyTasks := a.calculator.GroupTasksByWeek(tasks, period)

	for weekNum, weekTasks := range weeklyTasks {
		completed := a.calculator.CountCompletedTasks(weekTasks)
		velocity := entities.WeeklyVelocity{
			Number:   weekNum,
			Velocity: float64(completed),
			Tasks:    len(weekTasks),
		}
		metrics.ByWeek = append(metrics.ByWeek, velocity)
	}

	// Sort by week number
	sort.Slice(metrics.ByWeek, func(i, j int) bool {
		return metrics.ByWeek[i].Number < metrics.ByWeek[j].Number
	})

	// Calculate current velocity (last 4 weeks average)
	if len(metrics.ByWeek) > 0 {
		recentWeeks := metrics.ByWeek
		if len(recentWeeks) > 4 {
			recentWeeks = recentWeeks[len(recentWeeks)-4:]
		}

		totalVelocity := 0.0
		for _, week := range recentWeeks {
			totalVelocity += week.Velocity
		}
		metrics.CurrentVelocity = totalVelocity / float64(len(recentWeeks))
	}

	// Calculate trend
	if len(metrics.ByWeek) >= 2 {
		recent := metrics.ByWeek[len(metrics.ByWeek)-1].Velocity
		previous := metrics.ByWeek[len(metrics.ByWeek)-2].Velocity

		if recent > previous*1.1 {
			metrics.TrendDirection = "up"
			metrics.TrendPercentage = ((recent - previous) / previous) * 100
		} else if recent < previous*0.9 {
			metrics.TrendDirection = "down"
			metrics.TrendPercentage = ((previous - recent) / previous) * 100
		} else {
			metrics.TrendDirection = "stable"
			metrics.TrendPercentage = 0
		}
	}

	// Calculate consistency (coefficient of variation)
	if len(metrics.ByWeek) > 1 {
		velocities := make([]float64, len(metrics.ByWeek))
		for i, week := range metrics.ByWeek {
			velocities[i] = week.Velocity
		}
		metrics.Consistency = 1.0 - a.calculator.CoefficientOfVariation(velocities)
	}

	// Generate forecast
	metrics.Forecast = a.calculator.ForecastVelocity(metrics.ByWeek)

	return metrics
}

func (a *analyticsServiceImpl) calculateCycleTimeMetrics(
	tasks []*entities.Task,
) entities.CycleTimeMetrics {
	metrics := entities.CycleTimeMetrics{
		ByType:       make(map[string]time.Duration),
		ByPriority:   make(map[string]time.Duration),
		Distribution: []entities.CycleTimePoint{},
	}

	completedTasks := a.filterTasksByStatus(tasks, string(entities.StatusCompleted))
	if len(completedTasks) == 0 {
		return metrics
	}

	// Calculate cycle times
	cycleTimes := make([]time.Duration, 0, len(completedTasks))
	typeGroups := make(map[string][]time.Duration)
	priorityGroups := make(map[string][]time.Duration)

	for _, task := range completedTasks {
		cycleTime := a.calculator.GetTaskDuration(task)
		if cycleTime > 0 {
			cycleTimes = append(cycleTimes, cycleTime)

			// Group by type (using first tag as type)
			taskType := "default"
			if len(task.Tags) > 0 {
				taskType = task.Tags[0]
			}
			if _, exists := typeGroups[taskType]; !exists {
				typeGroups[taskType] = []time.Duration{}
			}
			typeGroups[taskType] = append(typeGroups[taskType], cycleTime)

			// Group by priority
			priorityStr := string(task.Priority)
			if _, exists := priorityGroups[priorityStr]; !exists {
				priorityGroups[priorityStr] = []time.Duration{}
			}
			priorityGroups[priorityStr] = append(priorityGroups[priorityStr], cycleTime)
		}
	}

	if len(cycleTimes) > 0 {
		// Calculate basic metrics
		metrics.AverageCycleTime = a.calculator.AverageDuration(cycleTimes)
		metrics.MedianCycleTime = a.calculator.MedianDuration(cycleTimes)
		metrics.P90CycleTime = a.calculator.PercentileDuration(cycleTimes, 90)

		// Calculate by type
		for taskType, times := range typeGroups {
			metrics.ByType[taskType] = a.calculator.AverageDuration(times)
		}

		// Calculate by priority
		for priority, times := range priorityGroups {
			metrics.ByPriority[priority] = a.calculator.AverageDuration(times)
		}

		// Create distribution
		metrics.Distribution = a.calculator.CreateCycleTimeDistribution(cycleTimes)
	}

	return metrics
}

func (a *analyticsServiceImpl) calculatePatternMetrics(
	patterns []*entities.TaskPattern,
	tasks []*entities.Task,
) entities.PatternMetrics {
	metrics := entities.PatternMetrics{
		PatternUsage:     make(map[string]int),
		SuccessRates:     make(map[string]float64),
		TopPatterns:      []entities.PatternUsage{},
		PatternEvolution: []entities.PatternEvolution{},
	}

	metrics.TotalPatterns = len(patterns)

	// Count active patterns (used recently)
	recentThreshold := time.Now().AddDate(0, 0, -30)
	for _, pattern := range patterns {
		if pattern.UpdatedAt.After(recentThreshold) {
			metrics.ActivePatterns++
		}

		// Track usage
		metrics.PatternUsage[pattern.Name] = int(pattern.Frequency * 100) // Convert to percentage
		metrics.SuccessRates[pattern.Name] = pattern.SuccessRate
	}

	// Calculate pattern adherence rate
	if len(tasks) > 0 && len(patterns) > 0 {
		// This would require more sophisticated pattern matching
		// For now, use a simplified calculation
		metrics.AdherenceRate = 0.7 // Placeholder
	}

	return metrics
}

func (a *analyticsServiceImpl) detectBottlenecksInternal(
	tasks []*entities.Task,
	sessions []*entities.Session,
) []*entities.Bottleneck {
	var bottlenecks []*entities.Bottleneck

	// Detect cycle time bottlenecks
	bottlenecks = append(bottlenecks, a.detectCycleTimeBottlenecks(tasks)...)

	// Detect dependency bottlenecks
	bottlenecks = append(bottlenecks, a.detectDependencyBottlenecks(tasks)...)

	// Detect time-of-day bottlenecks
	if len(sessions) > 0 {
		bottlenecks = append(bottlenecks, a.detectTimeBottlenecks(sessions)...)
	}

	// Sort by impact
	sort.Slice(bottlenecks, func(i, j int) bool {
		return bottlenecks[i].Impact > bottlenecks[j].Impact
	})

	return bottlenecks
}

func (a *analyticsServiceImpl) detectCycleTimeBottlenecks(tasks []*entities.Task) []*entities.Bottleneck {
	// Group tasks by type and calculate average cycle times
	typeGroups := make(map[string][]time.Duration)

	for _, task := range tasks {
		if task.Status == entities.StatusCompleted {
			duration := a.calculator.GetTaskDuration(task)
			if duration > 0 {
				taskType := constants.TaskTypeDefault
				if len(task.Tags) > 0 {
					taskType = task.Tags[0]
				}
				typeGroups[taskType] = append(typeGroups[taskType], duration)
			}
		}
	}

	var bottlenecks []*entities.Bottleneck

	for taskType, durations := range typeGroups {
		if len(durations) < 3 { // Need minimum samples
			continue
		}

		avgDuration := a.calculator.AverageDuration(durations)
		if avgDuration > time.Duration(a.config.BottleneckThreshold)*time.Hour {
			impact := float64(avgDuration) / float64(time.Hour) * float64(len(durations))

			severity := entities.BottleneckSeverityLow
			if impact > 24 {
				severity = entities.BottleneckSeverityCritical
			} else if impact > 8 {
				severity = entities.BottleneckSeverityHigh
			} else if impact > 2 {
				severity = entities.BottleneckSeverityMedium
			}

			bottleneck := &entities.Bottleneck{
				Type:        "cycle_time",
				Description: fmt.Sprintf("Tasks of type '%s' take %.1f hours on average", taskType, avgDuration.Hours()),
				Impact:      impact,
				Frequency:   len(durations),
				Severity:    severity,
				Suggestions: []string{
					"Break down complex tasks into smaller subtasks",
					"Identify and eliminate blockers",
					"Consider automation opportunities",
				},
				DetectedAt: time.Now(),
				Metadata:   map[string]interface{}{"task_type": taskType, "avg_duration": avgDuration},
			}

			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	return bottlenecks
}

func (a *analyticsServiceImpl) detectDependencyBottlenecks(tasks []*entities.Task) []*entities.Bottleneck {
	// This would analyze task dependencies to find blocking patterns
	// Simplified implementation for now
	return []*entities.Bottleneck{}
}

func (a *analyticsServiceImpl) detectTimeBottlenecks(sessions []*entities.Session) []*entities.Bottleneck {
	// Analyze productivity by time of day to identify low-productivity periods
	// Simplified implementation for now
	return []*entities.Bottleneck{}
}

func (a *analyticsServiceImpl) analyzeTrends(
	ctx context.Context,
	repository string,
	period entities.TimePeriod,
	metrics *entities.WorkflowMetrics,
) entities.TrendAnalysis {
	trends := entities.TrendAnalysis{
		Predictions: []entities.Prediction{},
	}

	// Get historical data for trend analysis
	historicalPeriod := entities.TimePeriod{
		Start: period.Start.AddDate(0, -3, 0), // 3 months back
		End:   period.End,
	}

	historicalTasks, err := a.taskStore.GetByPeriod(ctx, repository, historicalPeriod)
	if err != nil {
		a.logger.Warn("failed to get historical data for trends", slog.Any("error", err))
		return trends
	}

	// Calculate trends (simplified implementation)
	trends.ProductivityTrend = a.calculator.CalculateProductivityTrend(historicalTasks, period)
	trends.VelocityTrend = a.calculator.CalculateVelocityTrend(historicalTasks, period)
	trends.QualityTrend = a.calculator.CalculateQualityTrend(historicalTasks, period)

	return trends
}

// GenerateInsights creates productivity insights from metrics
func (a *analyticsServiceImpl) GenerateInsights(
	ctx context.Context,
	metrics *entities.WorkflowMetrics,
) ([]*entities.ProductivityInsight, error) {
	var insights []*entities.ProductivityInsight

	// Productivity insights
	if metrics.Productivity.Score > 80 {
		insights = append(insights, &entities.ProductivityInsight{
			Type:        entities.InsightTypeBestPractice,
			Title:       "Excellent Productivity",
			Description: fmt.Sprintf("Your productivity score of %.1f is excellent. Keep up the great work!", metrics.Productivity.Score),
			Impact:      0.9,
			Confidence:  0.95,
			Evidence:    []string{fmt.Sprintf("High productivity score: %.1f", metrics.Productivity.Score)},
			ActionItems: []string{"Maintain current working patterns", "Share best practices with team"},
		})
	} else if metrics.Productivity.Score < 60 {
		insights = append(insights, &entities.ProductivityInsight{
			Type:        entities.InsightTypeAntiPattern,
			Title:       "Low Productivity Detected",
			Description: fmt.Sprintf("Your productivity score of %.1f indicates room for improvement.", metrics.Productivity.Score),
			Impact:      0.8,
			Confidence:  0.85,
			Evidence:    []string{fmt.Sprintf("Low productivity score: %.1f", metrics.Productivity.Score)},
			ActionItems: []string{"Review workflow bottlenecks", "Consider time management techniques"},
		})
	}

	// Velocity insights
	if metrics.Velocity.TrendDirection == "up" {
		insights = append(insights, &entities.ProductivityInsight{
			Type:        entities.InsightTypePattern,
			Title:       "Increasing Velocity",
			Description: fmt.Sprintf("Your velocity is trending upward by %.1f%%", metrics.Velocity.TrendPercentage),
			Impact:      0.7,
			Confidence:  0.8,
			Evidence:    []string{fmt.Sprintf("Velocity trend: %s (%.1f%%)", metrics.Velocity.TrendDirection, metrics.Velocity.TrendPercentage)},
			ActionItems: []string{"Identify what's working well", "Scale successful practices"},
		})
	}

	return insights, nil
}

// GenerateRecommendations creates actionable recommendations
func (a *analyticsServiceImpl) GenerateRecommendations(
	ctx context.Context,
	metrics *entities.WorkflowMetrics,
) ([]*entities.Recommendation, error) {
	var recommendations []*entities.Recommendation

	// High-impact bottleneck recommendations
	for _, bottleneck := range metrics.Bottlenecks {
		if bottleneck.IsHighImpact() {
			recommendation := &entities.Recommendation{
				ID:          "bottleneck_" + bottleneck.Type,
				Title:       fmt.Sprintf("Address %s Bottleneck", bottleneck.Type),
				Description: bottleneck.Description,
				Priority:    entities.RecommendationPriorityHigh,
				Impact:      bottleneck.Impact / 10, // Normalize to 0-1
				Effort:      0.5,                    // Medium effort
				Category:    "workflow",
				Actions:     bottleneck.Suggestions,
				Evidence:    []string{bottleneck.GetImpactDescription()},
				CreatedAt:   time.Now(),
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	// Low completion rate recommendations
	if metrics.Completion.CompletionRate < 0.7 {
		recommendation := &entities.Recommendation{
			ID:          "low_completion_rate",
			Title:       "Improve Task Completion Rate",
			Description: fmt.Sprintf("Current completion rate is %.1f%%, below the recommended 80%%", metrics.Completion.CompletionRate*100),
			Priority:    entities.RecommendationPriorityMedium,
			Impact:      0.8,
			Effort:      0.6,
			Category:    "completion",
			Actions: []string{
				"Review and prioritize task backlog",
				"Break down large tasks into smaller ones",
				"Set clear deadlines and milestones",
			},
			Evidence:  []string{fmt.Sprintf("Completion rate: %.1f%%", metrics.Completion.CompletionRate*100)},
			CreatedAt: time.Now(),
		}
		recommendations = append(recommendations, recommendation)
	}

	return recommendations, nil
}

func (a *analyticsServiceImpl) generateCharts(metrics *entities.WorkflowMetrics) map[string]entities.ChartData {
	charts := make(map[string]entities.ChartData)

	// Productivity chart
	charts["productivity"] = entities.ChartData{
		Type:  entities.ChartTypeProgress,
		Title: "Productivity Score",
		Data: map[string]interface{}{
			"value": metrics.Productivity.Score,
			"max":   100,
		},
	}

	// Velocity trend chart
	if len(metrics.Velocity.ByWeek) > 0 {
		weeks := make([]string, len(metrics.Velocity.ByWeek))
		velocities := make([]float64, len(metrics.Velocity.ByWeek))

		for i, week := range metrics.Velocity.ByWeek {
			weeks[i] = fmt.Sprintf("W%d", week.Number)
			velocities[i] = week.Velocity
		}

		charts["velocity"] = entities.ChartData{
			Type:  entities.ChartTypeLine,
			Title: "Velocity Trend",
			Data: map[string]interface{}{
				"labels": weeks,
				"values": velocities,
			},
		}
	}

	// Completion by priority chart
	if len(metrics.Productivity.ByPriority) > 0 {
		labels := make([]string, 0, len(metrics.Productivity.ByPriority))
		values := make([]float64, 0, len(metrics.Productivity.ByPriority))

		for priority, rate := range metrics.Productivity.ByPriority {
			labels = append(labels, priority)
			values = append(values, rate*100)
		}

		charts["priority_completion"] = entities.ChartData{
			Type:  entities.ChartTypeBar,
			Title: "Completion Rate by Priority",
			Data: map[string]interface{}{
				"labels": labels,
				"values": values,
			},
		}
	}

	return charts
}

// Helper methods

func (a *analyticsServiceImpl) filterTasksByStatus(tasks []*entities.Task, status string) []*entities.Task {
	var filtered []*entities.Task
	for _, task := range tasks {
		if string(task.Status) == status {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// Storage interfaces (to be implemented by storage layer)
type TaskStorage interface {
	GetByPeriod(ctx context.Context, repository string, period entities.TimePeriod) ([]*entities.Task, error)
}

type SessionStorage interface {
	GetByPeriod(ctx context.Context, repository string, period entities.TimePeriod) ([]*entities.Session, error)
}

// Visualizer interface for generating visualizations
type Visualizer interface {
	GenerateVisualization(metrics *entities.WorkflowMetrics, format entities.VisFormat) ([]byte, error)
}

// AnalyticsExporter interface for exporting analytics
type AnalyticsExporter interface {
	Export(metrics *entities.WorkflowMetrics, format entities.ExportFormat) (string, error)
}
