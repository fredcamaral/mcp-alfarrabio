package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/repositories"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// AnalyticsEngine interface defines analytics capabilities
type AnalyticsEngine interface {
	// Productivity analytics
	CalculateProductivityScore(session *entities.Session) float64
	GetProductivityTrends(ctx context.Context, repository string, days int) (*ProductivityTrends, error)

	// Task analytics
	GetTaskCompletionTrends(ctx context.Context, repository string, days int) (*CompletionTrends, error)
	AnalyzeTaskDurations(tasks []*entities.Task) *DurationAnalysis
	GetTaskTypeDistribution(ctx context.Context, repository string, days int) (*TypeDistribution, error)

	// Workflow analytics
	DetectBottlenecks(ctx context.Context, workflows []*entities.TaskPattern) ([]*Bottleneck, error)
	AnalyzeWorkflowEfficiency(ctx context.Context, repository string) (*WorkflowEfficiency, error)

	// Performance analytics
	GetVelocityMetrics(ctx context.Context, repository string, days int) (*VelocityMetrics, error)
	AnalyzeFocusPatterns(sessions []*entities.Session) (*FocusAnalysis, error)
	GetBurnoutRiskAssessment(ctx context.Context, repository string) (*BurnoutRisk, error)

	// Predictive analytics
	PredictTaskDuration(task *entities.Task, historicalData []*entities.Task) (time.Duration, float64)
	ForecastProductivity(ctx context.Context, repository string, days int) (*ProductivityForecast, error)
	GetOptimalWorkingHours(sessions []*entities.Session) (*OptimalHours, error)
}

// Analytics data structures

type ProductivityTrends struct {
	Repository   string               `json:"repository"`
	TimeRange    services.TimeRange   `json:"time_range"`
	DailyScores  []DailyProductivity  `json:"daily_scores"`
	WeeklyScores []WeeklyProductivity `json:"weekly_scores"`
	OverallTrend float64              `json:"overall_trend"` // -1 to 1 (declining to improving)
	AverageScore float64              `json:"average_score"`
	BestPeriod   DailyProductivity    `json:"best_period"`
	WorstPeriod  DailyProductivity    `json:"worst_period"`
	Insights     []string             `json:"insights"`
	CalculatedAt time.Time            `json:"calculated_at"`
}

type DailyProductivity struct {
	Date           time.Time     `json:"date"`
	Score          float64       `json:"score"`
	TasksCompleted int           `json:"tasks_completed"`
	TasksStarted   int           `json:"tasks_started"`
	WorkDuration   time.Duration `json:"work_duration"`
	FocusScore     float64       `json:"focus_score"`
	EnergyLevel    float64       `json:"energy_level"`
	Interruptions  int           `json:"interruptions"`
}

type WeeklyProductivity struct {
	WeekStart    time.Time    `json:"week_start"`
	WeekEnd      time.Time    `json:"week_end"`
	AverageScore float64      `json:"average_score"`
	TotalTasks   int          `json:"total_tasks"`
	TotalHours   float64      `json:"total_hours"`
	BestDay      time.Weekday `json:"best_day"`
	WorstDay     time.Weekday `json:"worst_day"`
}

type CompletionTrends struct {
	Repository       string                 `json:"repository"`
	TimeRange        services.TimeRange     `json:"time_range"`
	DailyCompletions []DailyCompletion      `json:"daily_completions"`
	CompletionRate   float64                `json:"completion_rate"` // Overall completion percentage
	Velocity         float64                `json:"velocity"`        // Tasks per day
	TrendDirection   string                 `json:"trend_direction"` // improving, stable, declining
	Predictions      []CompletionPrediction `json:"predictions"`
	CalculatedAt     time.Time              `json:"calculated_at"`
}

type DailyCompletion struct {
	Date       time.Time `json:"date"`
	Completed  int       `json:"completed"`
	Started    int       `json:"started"`
	InProgress int       `json:"in_progress"`
	Rate       float64   `json:"rate"`
}

type CompletionPrediction struct {
	Date           time.Time `json:"date"`
	PredictedTasks int       `json:"predicted_tasks"`
	Confidence     float64   `json:"confidence"`
	BasedOnPattern string    `json:"based_on_pattern"`
}

type DurationAnalysis struct {
	TotalTasks        int                                `json:"total_tasks"`
	AverageDuration   time.Duration                      `json:"average_duration"`
	MedianDuration    time.Duration                      `json:"median_duration"`
	MinDuration       time.Duration                      `json:"min_duration"`
	MaxDuration       time.Duration                      `json:"max_duration"`
	StandardDeviation time.Duration                      `json:"standard_deviation"`
	ByTaskType        map[string]*entities.DurationStats `json:"by_task_type"`
	ByPriority        map[string]*entities.DurationStats `json:"by_priority"`
	Outliers          []*TaskDurationOutlier             `json:"outliers"`
	Insights          []string                           `json:"insights"`
}

type TaskDurationOutlier struct {
	TaskID       string        `json:"task_id"`
	Content      string        `json:"content"`
	Duration     time.Duration `json:"duration"`
	Expected     time.Duration `json:"expected"`
	DeviationPct float64       `json:"deviation_pct"`
	Reason       string        `json:"reason"`
}

type TypeDistribution struct {
	Repository   string                 `json:"repository"`
	TimeRange    services.TimeRange     `json:"time_range"`
	Distribution map[string]TypeMetrics `json:"distribution"`
	TopTypes     []string               `json:"top_types"`
	Trends       map[string]float64     `json:"trends"` // Type -> trend direction
	CalculatedAt time.Time              `json:"calculated_at"`
}

type TypeMetrics struct {
	Count             int           `json:"count"`
	Percentage        float64       `json:"percentage"`
	AvgDuration       time.Duration `json:"avg_duration"`
	CompletionRate    float64       `json:"completion_rate"`
	ProductivityScore float64       `json:"productivity_score"`
}

type Bottleneck struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"` // task_type, workflow_step, time_period
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Severity        float64                `json:"severity"`     // 0-1, how severe the bottleneck is
	ImpactScore     float64                `json:"impact_score"` // 0-1, impact on overall productivity
	Frequency       int                    `json:"frequency"`    // How often this bottleneck occurs
	AverageDuration time.Duration          `json:"average_duration"`
	TotalTimeImpact time.Duration          `json:"total_time_impact"`
	AffectedTasks   []string               `json:"affected_tasks"`
	Suggestions     []string               `json:"suggestions"`
	Metadata        map[string]interface{} `json:"metadata"`
	DetectedAt      time.Time              `json:"detected_at"`
}

type WorkflowEfficiency struct {
	Repository        string                    `json:"repository"`
	OverallEfficiency float64                   `json:"overall_efficiency"` // 0-1
	WorkflowMetrics   map[string]WorkflowMetric `json:"workflow_metrics"`
	BottleneckCount   int                       `json:"bottleneck_count"`
	OptimizationScore float64                   `json:"optimization_score"`
	Recommendations   []string                  `json:"recommendations"`
	CalculatedAt      time.Time                 `json:"calculated_at"`
}

type WorkflowMetric struct {
	Name               string        `json:"name"`
	AverageDuration    time.Duration `json:"average_duration"`
	CompletionRate     float64       `json:"completion_rate"`
	SuccessRate        float64       `json:"success_rate"`
	EfficiencyScore    float64       `json:"efficiency_score"`
	ParallelizationPct float64       `json:"parallelization_pct"`
}

type VelocityMetrics struct {
	Repository         string               `json:"repository"`
	TimeRange          services.TimeRange   `json:"time_range"`
	CurrentVelocity    float64              `json:"current_velocity"` // Tasks per day
	AverageVelocity    float64              `json:"average_velocity"`
	VelocityTrend      float64              `json:"velocity_trend"` // -1 to 1
	DailyVelocities    []DailyVelocity      `json:"daily_velocities"`
	PeakVelocity       DailyVelocity        `json:"peak_velocity"`
	LowestVelocity     DailyVelocity        `json:"lowest_velocity"`
	VelocityByType     map[string]float64   `json:"velocity_by_type"`
	VelocityByPriority map[string]float64   `json:"velocity_by_priority"`
	PredictedVelocity  []VelocityPrediction `json:"predicted_velocity"`
	CalculatedAt       time.Time            `json:"calculated_at"`
}

type DailyVelocity struct {
	Date     time.Time `json:"date"`
	Velocity float64   `json:"velocity"`
	Tasks    int       `json:"tasks"`
	Hours    float64   `json:"hours"`
}

type VelocityPrediction struct {
	Date       time.Time     `json:"date"`
	Velocity   float64       `json:"velocity"`
	Confidence float64       `json:"confidence"`
	Range      VelocityRange `json:"range"`
}

type VelocityRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type FocusAnalysis struct {
	OverallFocusScore   float64            `json:"overall_focus_score"`
	AverageFocusTime    time.Duration      `json:"average_focus_time"`
	MaxFocusTime        time.Duration      `json:"max_focus_time"`
	InterruptionRate    float64            `json:"interruption_rate"`
	FocusByTimeOfDay    map[string]float64 `json:"focus_by_time_of_day"`
	FocusByDayOfWeek    map[string]float64 `json:"focus_by_day_of_week"`
	OptimalFocusPeriods []FocusPeriod      `json:"optimal_focus_periods"`
	DistractionSources  map[string]int     `json:"distraction_sources"`
	FocusImprovement    float64            `json:"focus_improvement"` // Trend over time
	Recommendations     []string           `json:"recommendations"`
}

type FocusPeriod struct {
	StartTime   string        `json:"start_time"`
	EndTime     string        `json:"end_time"`
	FocusScore  float64       `json:"focus_score"`
	Duration    time.Duration `json:"duration"`
	Consistency float64       `json:"consistency"`
}

type BurnoutRisk struct {
	Repository        string             `json:"repository"`
	RiskLevel         string             `json:"risk_level"` // low, medium, high, critical
	RiskScore         float64            `json:"risk_score"` // 0-1
	RiskFactors       []RiskFactor       `json:"risk_factors"`
	ProtectiveFactors []ProtectiveFactor `json:"protective_factors"`
	TrendDirection    string             `json:"trend_direction"` // improving, stable, worsening
	Recommendations   []string           `json:"recommendations"`
	NextAssessment    time.Time          `json:"next_assessment"`
	CalculatedAt      time.Time          `json:"calculated_at"`
}

type RiskFactor struct {
	Name        string  `json:"name"`
	Severity    float64 `json:"severity"` // 0-1
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
}

type ProtectiveFactor struct {
	Name        string  `json:"name"`
	Strength    float64 `json:"strength"` // 0-1
	Description string  `json:"description"`
}

type ProductivityForecast struct {
	Repository      string                      `json:"repository"`
	ForecastPeriod  services.TimeRange          `json:"forecast_period"`
	DailyForecasts  []DailyProductivityForecast `json:"daily_forecasts"`
	OverallTrend    float64                     `json:"overall_trend"`
	ConfidenceLevel float64                     `json:"confidence_level"`
	BasedOnPatterns []string                    `json:"based_on_patterns"`
	Assumptions     []string                    `json:"assumptions"`
	CalculatedAt    time.Time                   `json:"calculated_at"`
}

type DailyProductivityForecast struct {
	Date            time.Time         `json:"date"`
	ForecastScore   float64           `json:"forecast_score"`
	ConfidenceRange ProductivityRange `json:"confidence_range"`
	PredictedTasks  int               `json:"predicted_tasks"`
	Factors         []ForecastFactor  `json:"factors"`
}

type ProductivityRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type ForecastFactor struct {
	Name   string  `json:"name"`
	Impact float64 `json:"impact"` // -1 to 1
}

type OptimalHours struct {
	BestHours           []int           `json:"best_hours"` // Hours of day (0-23)
	WorstHours          []int           `json:"worst_hours"`
	BestDays            []time.Weekday  `json:"best_days"`
	EnergyPattern       map[int]float64 `json:"energy_pattern"`       // Hour -> energy level
	ProductivityPattern map[int]float64 `json:"productivity_pattern"` // Hour -> productivity
	RecommendedSchedule *Schedule       `json:"recommended_schedule"`
	Confidence          float64         `json:"confidence"`
}

type Schedule struct {
	DeepWorkHours []TimeSlot `json:"deep_work_hours"`
	MeetingHours  []TimeSlot `json:"meeting_hours"`
	BreakTimes    []TimeSlot `json:"break_times"`
	LearningHours []TimeSlot `json:"learning_hours"`
}

type TimeSlot struct {
	Start string `json:"start"` // HH:MM format
	End   string `json:"end"`
}

// analyticsEngineImpl implements the AnalyticsEngine interface
type analyticsEngineImpl struct {
	taskRepo    repositories.TaskRepository
	sessionRepo services.SessionRepository
	patternRepo services.PatternRepository
	calculator  *ProductivityCalculator
	logger      *slog.Logger
}

// NewAnalyticsEngine creates a new analytics engine
func NewAnalyticsEngine(
	taskRepo repositories.TaskRepository,
	sessionRepo services.SessionRepository,
	patternRepo services.PatternRepository,
	calculator *ProductivityCalculator,
	logger *slog.Logger,
) services.AnalyticsEngine {
	return &analyticsEngineImpl{
		taskRepo:    taskRepo,
		sessionRepo: sessionRepo,
		patternRepo: patternRepo,
		calculator:  calculator,
		logger:      logger,
	}
}

// CalculateProductivityScore calculates productivity score for a session
func (ae *analyticsEngineImpl) CalculateProductivityScore(session *entities.Session) float64 {
	if session == nil {
		return 0.0
	}

	// Use the session's built-in calculation
	return session.CalculateProductivityScore()
}

// GetProductivityTrends gets productivity trends over time
func (ae *analyticsEngineImpl) GetProductivityTrends(
	ctx context.Context,
	repository string,
	days int,
) (*ProductivityTrends, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	sessions, err := ae.sessionRepo.FindByTimeRange(ctx, repository, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	// Group sessions by day
	dailyGroups := ae.groupSessionsByDay(sessions)

	dailyScores := make([]DailyProductivity, 0, len(dailyGroups))
	var totalScore float64
	var bestPeriod, worstPeriod DailyProductivity
	bestScore, worstScore := -1.0, 2.0

	for date, daySessions := range dailyGroups {
		daily := ae.calculateDailyProductivity(date, daySessions)
		dailyScores = append(dailyScores, daily)
		totalScore += daily.Score

		if daily.Score > bestScore {
			bestScore = daily.Score
			bestPeriod = daily
		}
		if daily.Score < worstScore {
			worstScore = daily.Score
			worstPeriod = daily
		}
	}

	// Sort by date
	sort.Slice(dailyScores, func(i, j int) bool {
		return dailyScores[i].Date.Before(dailyScores[j].Date)
	})

	// Calculate trend
	trend := ae.calculateTrend(dailyScores)

	// Calculate weekly scores
	weeklyScores := ae.calculateWeeklyScores(dailyScores)

	// Generate insights
	insights := ae.generateProductivityInsights(dailyScores, trend)

	return &ProductivityTrends{
		Repository:   repository,
		TimeRange:    services.TimeRange{Start: startTime, End: endTime},
		DailyScores:  dailyScores,
		WeeklyScores: weeklyScores,
		OverallTrend: trend,
		AverageScore: totalScore / float64(len(dailyScores)),
		BestPeriod:   bestPeriod,
		WorstPeriod:  worstPeriod,
		Insights:     insights,
		CalculatedAt: time.Now(),
	}, nil
}

// GetTaskCompletionTrends gets task completion trends
func (ae *analyticsEngineImpl) GetTaskCompletionTrends(
	ctx context.Context,
	repository string,
	days int,
) (*CompletionTrends, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	tasks, err := ae.taskRepo.FindByTimeRange(ctx, repository, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Group tasks by day
	dailyGroups := ae.groupTasksByDay(tasks)

	dailyCompletions := make([]DailyCompletion, 0, len(dailyGroups))
	totalCompleted, totalStarted := 0, 0

	for date, dayTasks := range dailyGroups {
		completed := ae.countTasksByStatus(dayTasks, "completed")
		started := len(dayTasks)
		inProgress := ae.countTasksByStatus(dayTasks, "in_progress")

		rate := 0.0
		if started > 0 {
			rate = float64(completed) / float64(started)
		}

		dailyCompletions = append(dailyCompletions, DailyCompletion{
			Date:       date,
			Completed:  completed,
			Started:    started,
			InProgress: inProgress,
			Rate:       rate,
		})

		totalCompleted += completed
		totalStarted += started
	}

	// Sort by date
	sort.Slice(dailyCompletions, func(i, j int) bool {
		return dailyCompletions[i].Date.Before(dailyCompletions[j].Date)
	})

	// Calculate overall metrics
	completionRate := 0.0
	if totalStarted > 0 {
		completionRate = float64(totalCompleted) / float64(totalStarted)
	}

	velocity := float64(totalCompleted) / float64(days)
	trendDirection := ae.calculateCompletionTrend(dailyCompletions)

	// Generate predictions
	predictions := ae.generateCompletionPredictions(dailyCompletions, 7) // 7-day forecast

	return &CompletionTrends{
		Repository:       repository,
		TimeRange:        services.TimeRange{Start: startTime, End: endTime},
		DailyCompletions: dailyCompletions,
		CompletionRate:   completionRate,
		Velocity:         velocity,
		TrendDirection:   trendDirection,
		Predictions:      predictions,
		CalculatedAt:     time.Now(),
	}, nil
}

// AnalyzeTaskDurations analyzes task duration patterns
func (ae *analyticsEngineImpl) AnalyzeTaskDurations(tasks []*entities.Task) *DurationAnalysis {
	if len(tasks) == 0 {
		return &DurationAnalysis{
			TotalTasks: 0,
			ByTaskType: make(map[string]*entities.DurationStats),
			ByPriority: make(map[string]*entities.DurationStats),
			Outliers:   make([]*TaskDurationOutlier, 0),
			Insights:   []string{"No tasks available for duration analysis"},
		}
	}

	// Extract durations from completed tasks
	var durations []time.Duration
	byType := make(map[string][]time.Duration)
	byPriority := make(map[string][]time.Duration)

	for _, task := range tasks {
		if task.Status == "completed" && !task.CompletedAt.IsZero() && !task.CreatedAt.IsZero() {
			duration := task.CompletedAt.Sub(task.CreatedAt)
			if duration > 0 {
				durations = append(durations, duration)

				if task.Type != "" {
					byType[task.Type] = append(byType[task.Type], duration)
				}
				if task.Priority != "" {
					byPriority[string(task.Priority)] = append(byPriority[string(task.Priority)], duration)
				}
			}
		}
	}

	if len(durations) == 0 {
		return &DurationAnalysis{
			TotalTasks: len(tasks),
			ByTaskType: make(map[string]*entities.DurationStats),
			ByPriority: make(map[string]*entities.DurationStats),
			Outliers:   make([]*TaskDurationOutlier, 0),
			Insights:   []string{"No completed tasks with valid durations found"},
		}
	}

	// Calculate basic statistics
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	analysis := &DurationAnalysis{
		TotalTasks:     len(tasks),
		MinDuration:    durations[0],
		MaxDuration:    durations[len(durations)-1],
		MedianDuration: durations[len(durations)/2],
		ByTaskType:     make(map[string]*entities.DurationStats),
		ByPriority:     make(map[string]*entities.DurationStats),
		Outliers:       make([]*TaskDurationOutlier, 0),
	}

	// Calculate average
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	analysis.AverageDuration = total / time.Duration(len(durations))

	// Calculate standard deviation
	variance := time.Duration(0)
	for _, d := range durations {
		diff := d - analysis.AverageDuration
		variance += time.Duration(int64(diff) * int64(diff) / int64(len(durations)))
	}
	analysis.StandardDeviation = time.Duration(math.Sqrt(float64(variance)))

	// Calculate statistics by type and priority
	for taskType, typeDurations := range byType {
		stats := &entities.DurationStats{}
		for _, d := range typeDurations {
			stats.UpdateDurationStats(d)
		}
		analysis.ByTaskType[taskType] = stats
	}

	for priority, priorityDurations := range byPriority {
		stats := &entities.DurationStats{}
		for _, d := range priorityDurations {
			stats.UpdateDurationStats(d)
		}
		analysis.ByPriority[priority] = stats
	}

	// Detect outliers (tasks taking significantly longer than average)
	analysis.Outliers = ae.detectDurationOutliers(tasks, analysis.AverageDuration, analysis.StandardDeviation)

	// Generate insights
	analysis.Insights = ae.generateDurationInsights(analysis)

	return analysis
}

// Helper methods

func (ae *analyticsEngineImpl) groupSessionsByDay(sessions []*entities.Session) map[time.Time][]*entities.Session {
	groups := make(map[time.Time][]*entities.Session)

	for _, session := range sessions {
		date := time.Date(
			session.StartTime.Year(),
			session.StartTime.Month(),
			session.StartTime.Day(),
			0, 0, 0, 0,
			session.StartTime.Location(),
		)
		groups[date] = append(groups[date], session)
	}

	return groups
}

func (ae *analyticsEngineImpl) groupTasksByDay(tasks []*entities.Task) map[time.Time][]*entities.Task {
	groups := make(map[time.Time][]*entities.Task)

	for _, task := range tasks {
		date := time.Date(
			task.CreatedAt.Year(),
			task.CreatedAt.Month(),
			task.CreatedAt.Day(),
			0, 0, 0, 0,
			task.CreatedAt.Location(),
		)
		groups[date] = append(groups[date], task)
	}

	return groups
}

func (ae *analyticsEngineImpl) calculateDailyProductivity(
	date time.Time,
	sessions []*entities.Session,
) DailyProductivity {
	if len(sessions) == 0 {
		return DailyProductivity{
			Date:  date,
			Score: 0.0,
		}
	}

	var totalScore, totalFocus, totalEnergy float64
	var totalTasks, totalInterruptions int
	var totalDuration time.Duration

	for _, session := range sessions {
		totalScore += session.ProductivityScore
		totalFocus += session.FocusScore
		totalTasks += session.TasksCompleted
		totalInterruptions += len(session.Interruptions)
		totalDuration += session.Duration

		if session.Environment != nil {
			totalEnergy += session.EnergyLevel.Overall
		}
	}

	count := float64(len(sessions))
	return DailyProductivity{
		Date:           date,
		Score:          totalScore / count,
		TasksCompleted: totalTasks,
		WorkDuration:   totalDuration,
		FocusScore:     totalFocus / count,
		EnergyLevel:    totalEnergy / count,
		Interruptions:  totalInterruptions,
	}
}

func (ae *analyticsEngineImpl) calculateTrend(dailyScores []DailyProductivity) float64 {
	if len(dailyScores) < 2 {
		return 0.0
	}

	// Simple linear regression to calculate trend
	n := float64(len(dailyScores))
	var sumX, sumY, sumXY, sumX2 float64

	for i, score := range dailyScores {
		x := float64(i)
		y := score.Score
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Normalize slope to -1 to 1 range
	if slope > 0.1 {
		return 1.0
	} else if slope < -0.1 {
		return -1.0
	}
	return slope * 10 // Scale to make small changes more visible
}

func (ae *analyticsEngineImpl) calculateWeeklyScores(dailyScores []DailyProductivity) []WeeklyProductivity {
	if len(dailyScores) == 0 {
		return nil
	}

	var weeklyScores []WeeklyProductivity
	currentWeek := &WeeklyProductivity{}
	weekScores := make([]float64, 0, 7) // Pre-allocate for up to 7 days per week
	var weekTasks int
	var weekHours float64

	for _, daily := range dailyScores {
		// Start of week (Monday)
		weekStart := getWeekStart(daily.Date)

		if currentWeek.WeekStart.IsZero() {
			currentWeek.WeekStart = weekStart
			currentWeek.WeekEnd = weekStart.AddDate(0, 0, 6)
		}

		if daily.Date.After(currentWeek.WeekEnd) {
			// Finalize current week
			if len(weekScores) > 0 {
				currentWeek.AverageScore = average(weekScores)
				currentWeek.TotalTasks = weekTasks
				currentWeek.TotalHours = weekHours
				weeklyScores = append(weeklyScores, *currentWeek)
			}

			// Start new week
			currentWeek = &WeeklyProductivity{
				WeekStart: weekStart,
				WeekEnd:   weekStart.AddDate(0, 0, 6),
			}
			weekScores = nil
			weekTasks = 0
			weekHours = 0
		}

		weekScores = append(weekScores, daily.Score)
		weekTasks += daily.TasksCompleted
		weekHours += daily.WorkDuration.Hours()
	}

	// Add final week
	if len(weekScores) > 0 {
		currentWeek.AverageScore = average(weekScores)
		currentWeek.TotalTasks = weekTasks
		currentWeek.TotalHours = weekHours
		weeklyScores = append(weeklyScores, *currentWeek)
	}

	return weeklyScores
}

func (ae *analyticsEngineImpl) generateProductivityInsights(
	dailyScores []DailyProductivity,
	trend float64,
) []string {
	var insights []string

	if trend > 0.1 {
		insights = append(insights, "Productivity trend is improving over time")
	} else if trend < -0.1 {
		insights = append(insights, "Productivity trend is declining - consider reviewing work patterns")
	} else {
		insights = append(insights, "Productivity trend is stable")
	}

	// Find patterns in the data
	if len(dailyScores) >= 7 {
		bestDay := findBestDayOfWeek(dailyScores)
		insights = append(insights, "Best productivity typically on "+bestDay)
	}

	return insights
}

func (ae *analyticsEngineImpl) countTasksByStatus(tasks []*entities.Task, status string) int {
	count := 0
	for _, task := range tasks {
		if string(task.Status) == status {
			count++
		}
	}
	return count
}

func (ae *analyticsEngineImpl) calculateCompletionTrend(completions []DailyCompletion) string {
	if len(completions) < 3 {
		return constants.StatusStable
	}

	// Look at last 7 days vs previous 7 days
	recentLen := minInt(7, len(completions))
	recent := completions[len(completions)-recentLen:]

	recentAvg := 0.0
	for _, completion := range recent {
		recentAvg += completion.Rate
	}
	recentAvg /= float64(len(recent))

	if len(completions) < recentLen*2 {
		return constants.StatusStable
	}

	previousLen := minInt(recentLen, len(completions)-recentLen)
	previous := completions[len(completions)-recentLen-previousLen : len(completions)-recentLen]

	previousAvg := 0.0
	for _, completion := range previous {
		previousAvg += completion.Rate
	}
	previousAvg /= float64(len(previous))

	diff := recentAvg - previousAvg
	if diff > 0.1 {
		return "improving"
	} else if diff < -0.1 {
		return "declining"
	}
	return "stable"
}

func (ae *analyticsEngineImpl) generateCompletionPredictions(
	completions []DailyCompletion,
	days int,
) []CompletionPrediction {
	if len(completions) < 3 {
		return nil
	}

	// Simple moving average prediction
	windowSize := minInt(7, len(completions))
	recent := completions[len(completions)-windowSize:]

	avgTasks := 0.0
	for _, completion := range recent {
		avgTasks += float64(completion.Completed)
	}
	avgTasks /= float64(len(recent))

	var predictions []CompletionPrediction
	lastDate := completions[len(completions)-1].Date

	for i := 1; i <= days; i++ {
		futureDate := lastDate.AddDate(0, 0, i)
		confidence := math.Max(0.1, 0.9-float64(i)*0.1) // Decreasing confidence

		predictions = append(predictions, CompletionPrediction{
			Date:           futureDate,
			PredictedTasks: int(avgTasks),
			Confidence:     confidence,
			BasedOnPattern: "moving_average",
		})
	}

	return predictions
}

func (ae *analyticsEngineImpl) detectDurationOutliers(
	tasks []*entities.Task,
	avgDuration time.Duration,
	stdDev time.Duration,
) []*TaskDurationOutlier {
	var outliers []*TaskDurationOutlier
	threshold := avgDuration + 2*stdDev // Tasks taking more than 2 standard deviations

	for _, task := range tasks {
		if task.Status == "completed" && !task.CompletedAt.IsZero() && !task.CreatedAt.IsZero() {
			duration := task.CompletedAt.Sub(task.CreatedAt)
			if duration > threshold {
				deviationPct := float64(duration-avgDuration) / float64(avgDuration) * 100

				outlier := &TaskDurationOutlier{
					TaskID:       task.ID,
					Content:      task.Content,
					Duration:     duration,
					Expected:     avgDuration,
					DeviationPct: deviationPct,
					Reason:       ae.inferOutlierReason(task, duration, avgDuration),
				}
				outliers = append(outliers, outlier)
			}
		}
	}

	return outliers
}

func (ae *analyticsEngineImpl) inferOutlierReason(_ *entities.Task, duration, avgDuration time.Duration) string {
	ratio := float64(duration) / float64(avgDuration)

	if ratio > 5 {
		return "Significantly more complex than typical tasks"
	} else if ratio > 3 {
		return "Higher complexity or unexpected complications"
	} else if ratio > 2 {
		return "Above average complexity"
	}

	return "Longer than expected duration"
}

func (ae *analyticsEngineImpl) generateDurationInsights(analysis *DurationAnalysis) []string {
	var insights []string

	// Compare median vs average
	if analysis.MedianDuration < analysis.AverageDuration {
		insights = append(insights, "Most tasks are shorter than average - a few long tasks are skewing the average")
	}

	// Check standard deviation
	if analysis.StandardDeviation > analysis.AverageDuration {
		insights = append(insights, "High variability in task durations - consider better estimation practices")
	}

	// Outlier insights
	if len(analysis.Outliers) > 0 {
		insights = append(insights, fmt.Sprintf("%d tasks took significantly longer than expected", len(analysis.Outliers)))
	}

	return insights
}

// Utility functions

func getWeekStart(date time.Time) time.Time {
	weekday := date.Weekday()
	daysFromMonday := (int(weekday) - int(time.Monday) + 7) % 7
	monday := date.AddDate(0, 0, -daysFromMonday)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func findBestDayOfWeek(dailyScores []DailyProductivity) string {
	dayScores := make(map[time.Weekday][]float64)

	for _, score := range dailyScores {
		day := score.Date.Weekday()
		dayScores[day] = append(dayScores[day], score.Score)
	}

	bestDay := time.Sunday
	bestAvg := -1.0

	for day, scores := range dayScores {
		avg := average(scores)
		if avg > bestAvg {
			bestAvg = avg
			bestDay = day
		}
	}

	return bestDay.String()
}

// ProductivityCalculator handles complex productivity calculations
type ProductivityCalculator struct {
	logger *slog.Logger
}

// NewProductivityCalculator creates a new productivity calculator
func NewProductivityCalculator(logger *slog.Logger) *ProductivityCalculator {
	return &ProductivityCalculator{
		logger: logger,
	}
}

// CalculateWeightedScore calculates a weighted productivity score
func (pc *ProductivityCalculator) CalculateWeightedScore(
	completionRate float64,
	efficiencyScore float64,
	focusScore float64,
	qualityScore float64,
) float64 {
	// Default weights
	weights := map[string]float64{
		"completion": 0.30,
		"efficiency": 0.25,
		"focus":      0.25,
		"quality":    0.20,
	}

	score := (completionRate * weights["completion"]) +
		(efficiencyScore * weights["efficiency"]) +
		(focusScore * weights["focus"]) +
		(qualityScore * weights["quality"])

	// Ensure score is between 0 and 1
	if score > 1.0 {
		score = 1.0
	} else if score < 0.0 {
		score = 0.0
	}

	return score
}

// CalculateProductivityMetrics calculates productivity metrics for tasks and sessions
func (ae *analyticsEngineImpl) CalculateProductivityMetrics(
	ctx context.Context,
	tasks []*entities.Task,
	sessions []*entities.Session,
) (*entities.ProductivityMetrics, error) {
	if len(tasks) == 0 && len(sessions) == 0 {
		return &entities.ProductivityMetrics{}, nil
	}

	tasksPerDay := ae.calculateTasksPerDay(tasks)
	byPriority, byType := ae.calculateCompletionRates(tasks)
	focusTime, contextSwitches, deepWorkRatio := ae.calculateSessionMetrics(sessions)
	peakHours := ae.calculatePeakHours()
	completionRate := ae.calculateOverallCompletionRate(tasks)
	score := ae.calculateProductivityScore(completionRate, deepWorkRatio, tasksPerDay)

	return &entities.ProductivityMetrics{
		Score:           score,
		TasksPerDay:     tasksPerDay,
		FocusTime:       focusTime,
		PeakHours:       peakHours,
		ByPriority:      byPriority,
		ByType:          byType,
		ContextSwitches: contextSwitches,
		DeepWorkRatio:   deepWorkRatio,
	}, nil
}

// calculateTasksPerDay calculates the average number of tasks per day
func (ae *analyticsEngineImpl) calculateTasksPerDay(tasks []*entities.Task) float64 {
	if len(tasks) == 0 {
		return 0.0
	}

	days := 1.0 // Default to 1 day if all tasks are on the same day
	if len(tasks) > 1 {
		oldest, newest := ae.findTaskDateRange(tasks)
		if daysDiff := newest.Sub(oldest).Hours() / 24; daysDiff > 0 {
			days = daysDiff
		}
	}

	return float64(len(tasks)) / days
}

// findTaskDateRange finds the oldest and newest task creation times
func (ae *analyticsEngineImpl) findTaskDateRange(tasks []*entities.Task) (time.Time, time.Time) {
	oldest := tasks[0].CreatedAt
	newest := tasks[0].CreatedAt

	for _, task := range tasks {
		if task.CreatedAt.Before(oldest) {
			oldest = task.CreatedAt
		}
		if task.CreatedAt.After(newest) {
			newest = task.CreatedAt
		}
	}

	return oldest, newest
}

// calculateCompletionRates calculates completion rates by priority and type
func (ae *analyticsEngineImpl) calculateCompletionRates(tasks []*entities.Task) (map[string]float64, map[string]float64) {
	byPriority := make(map[string]float64)
	byType := make(map[string]float64)

	priorityCounts, priorityCompleted := ae.countTasksByPriority(tasks)
	typeCounts, typeCompleted := ae.countTasksByType(tasks)

	for priority, total := range priorityCounts {
		completed := priorityCompleted[priority]
		byPriority[priority] = float64(completed) / float64(total)
	}

	for taskType, total := range typeCounts {
		completed := typeCompleted[taskType]
		byType[taskType] = float64(completed) / float64(total)
	}

	return byPriority, byType
}

// countTasksByPriority counts tasks and completed tasks by priority
func (ae *analyticsEngineImpl) countTasksByPriority(tasks []*entities.Task) (map[string]int, map[string]int) {
	priorityCounts := make(map[string]int)
	priorityCompleted := make(map[string]int)

	for _, task := range tasks {
		priority := string(task.Priority)
		priorityCounts[priority]++

		if task.Status == entities.StatusCompleted {
			priorityCompleted[priority]++
		}
	}

	return priorityCounts, priorityCompleted
}

// countTasksByType counts tasks and completed tasks by type
func (ae *analyticsEngineImpl) countTasksByType(tasks []*entities.Task) (map[string]int, map[string]int) {
	typeCounts := make(map[string]int)
	typeCompleted := make(map[string]int)

	for _, task := range tasks {
		taskType := task.Type
		typeCounts[taskType]++

		if task.Status == entities.StatusCompleted {
			typeCompleted[taskType]++
		}
	}

	return typeCounts, typeCompleted
}

// calculateSessionMetrics calculates focus time and session-based metrics
func (ae *analyticsEngineImpl) calculateSessionMetrics(sessions []*entities.Session) (time.Duration, int, float64) {
	if len(sessions) == 0 {
		return 0, 0, 0.0
	}

	totalSessionTime := time.Duration(0)
	totalFocusTime := time.Duration(0)
	totalContextSwitches := 0

	for _, session := range sessions {
		sessionTime, focusTime, switches := ae.calculateSingleSessionMetrics(session)
		totalSessionTime += sessionTime
		totalFocusTime += focusTime
		totalContextSwitches += switches
	}

	deepWorkRatio := 0.0
	if totalSessionTime > 0 {
		deepWorkRatio = float64(totalFocusTime) / float64(totalSessionTime)
	}

	return totalFocusTime, totalContextSwitches, deepWorkRatio
}

// calculateSingleSessionMetrics calculates metrics for a single session
func (ae *analyticsEngineImpl) calculateSingleSessionMetrics(session *entities.Session) (time.Duration, time.Duration, int) {
	sessionTime := session.Duration
	contextSwitches := len(session.Interruptions)

	// Estimate focus time as 70% of session time minus interruptions
	focusTime := time.Duration(float64(sessionTime) * 0.7)
	if contextSwitches > 0 {
		// Reduce focus time by 5 minutes per interruption
		interruptionPenalty := time.Duration(contextSwitches) * 5 * time.Minute
		if focusTime > interruptionPenalty {
			focusTime -= interruptionPenalty
		} else {
			focusTime = 0
		}
	}

	return sessionTime, focusTime, contextSwitches
}

// calculatePeakHours returns the default peak productive hours
func (ae *analyticsEngineImpl) calculatePeakHours() []int {
	// Simplified - would need more data for accurate calculation
	return []int{9, 10, 11, 14, 15, 16}
}

// calculateOverallCompletionRate calculates the overall task completion rate
func (ae *analyticsEngineImpl) calculateOverallCompletionRate(tasks []*entities.Task) float64 {
	if len(tasks) == 0 {
		return 0.0
	}

	completed := 0
	for _, task := range tasks {
		if task.Status == entities.StatusCompleted {
			completed++
		}
	}

	return float64(completed) / float64(len(tasks))
}

// calculateProductivityScore calculates the overall productivity score
func (ae *analyticsEngineImpl) calculateProductivityScore(completionRate, deepWorkRatio, tasksPerDay float64) float64 {
	score := (completionRate * 40) + (deepWorkRatio * 30) + (tasksPerDay * 20) + 10 // Base 10 points
	if score > 100 {
		score = 100
	}
	return score
}

// CalculateVelocityMetrics calculates velocity metrics for tasks
func (ae *analyticsEngineImpl) CalculateVelocityMetrics(
	ctx context.Context,
	tasks []*entities.Task,
) (*entities.VelocityMetrics, error) {
	if len(tasks) == 0 {
		return &entities.VelocityMetrics{
			CurrentVelocity: 0,
			TrendDirection:  "stable",
			TrendPercentage: 0,
			ByWeek:          []entities.WeeklyVelocity{},
			Consistency:     0,
		}, nil
	}

	// Group tasks by week
	weeklyTasks := make(map[string][]*entities.Task)
	for _, task := range tasks {
		if task.Status == entities.StatusCompleted && !task.CompletedAt.IsZero() {
			year, week := task.CompletedAt.ISOWeek()
			weekKey := fmt.Sprintf("%d-W%02d", year, week)
			weeklyTasks[weekKey] = append(weeklyTasks[weekKey], task)
		}
	}

	// Calculate weekly velocities
	weeklyVelocities := make([]entities.WeeklyVelocity, 0, len(weeklyTasks))
	for weekKey, weekTasks := range weeklyTasks {
		// Parse week key to get year and week number
		var year, week int
		if _, err := fmt.Sscanf(weekKey, "%d-W%d", &year, &week); err != nil {
			ae.logger.Warn("failed to parse week key", slog.String("key", weekKey), slog.Any("error", err))
			continue
		}

		// Calculate start of week
		startOfWeek := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		for startOfWeek.Weekday() != time.Monday {
			startOfWeek = startOfWeek.AddDate(0, 0, 1)
		}
		startOfWeek = startOfWeek.AddDate(0, 0, (week-1)*7)

		weeklyVelocities = append(weeklyVelocities, entities.WeeklyVelocity{
			Number:    week,
			Year:      year,
			Velocity:  float64(len(weekTasks)),
			Tasks:     len(weekTasks),
			StartDate: startOfWeek,
		})
	}

	// Sort by date
	sort.Slice(weeklyVelocities, func(i, j int) bool {
		return weeklyVelocities[i].StartDate.Before(weeklyVelocities[j].StartDate)
	})

	// Calculate current velocity (last week)
	currentVelocity := 0.0
	if len(weeklyVelocities) > 0 {
		currentVelocity = weeklyVelocities[len(weeklyVelocities)-1].Velocity
	}

	// Calculate trend
	trendDirection := "stable"
	trendPercentage := 0.0
	if len(weeklyVelocities) >= 2 {
		recent := weeklyVelocities[len(weeklyVelocities)-1].Velocity
		previous := weeklyVelocities[len(weeklyVelocities)-2].Velocity

		if previous > 0 {
			trendPercentage = ((recent - previous) / previous) * 100
			if trendPercentage > 5 {
				trendDirection = "up"
			} else if trendPercentage < -5 {
				trendDirection = "down"
			}
		}
	}

	// Calculate consistency (coefficient of variation)
	consistency := 0.0
	if len(weeklyVelocities) > 1 {
		velocities := make([]float64, len(weeklyVelocities))
		sum := 0.0
		for i, wv := range weeklyVelocities {
			velocities[i] = wv.Velocity
			sum += wv.Velocity
		}

		mean := sum / float64(len(velocities))
		variance := 0.0
		for _, v := range velocities {
			variance += (v - mean) * (v - mean)
		}
		variance /= float64(len(velocities))
		stdDev := math.Sqrt(variance)

		if mean > 0 {
			cv := stdDev / mean
			consistency = math.Max(0, 1.0-cv) // Higher consistency = lower coefficient of variation
		}
	}

	// Generate forecast
	forecast := entities.VelocityForecast{
		PredictedVelocity: currentVelocity,
		Confidence:        0.7,
		Range:             []float64{currentVelocity * 0.8, currentVelocity * 1.2},
		Method:            "simple_average",
		UpdatedAt:         time.Now(),
	}

	return &entities.VelocityMetrics{
		CurrentVelocity: currentVelocity,
		TrendDirection:  trendDirection,
		TrendPercentage: trendPercentage,
		ByWeek:          weeklyVelocities,
		Forecast:        forecast,
		Consistency:     consistency,
	}, nil
}

// CalculateCycleTimeMetrics calculates cycle time metrics for tasks
func (ae *analyticsEngineImpl) CalculateCycleTimeMetrics(
	ctx context.Context,
	tasks []*entities.Task,
) (*entities.CycleTimeMetrics, error) {
	completedTasks := make([]*entities.Task, 0)
	for _, task := range tasks {
		if task.Status == entities.StatusCompleted && !task.CompletedAt.IsZero() {
			completedTasks = append(completedTasks, task)
		}
	}

	if len(completedTasks) == 0 {
		return &entities.CycleTimeMetrics{
			AverageCycleTime: 0,
			MedianCycleTime:  0,
			P90CycleTime:     0,
			ByType:           make(map[string]time.Duration),
			ByPriority:       make(map[string]time.Duration),
			Distribution:     []entities.CycleTimePoint{},
			LeadTime:         0,
			WaitTime:         0,
		}, nil
	}

	// Calculate cycle times
	cycleTimes := make([]time.Duration, len(completedTasks))
	byType := make(map[string][]time.Duration)
	byPriority := make(map[string][]time.Duration)

	for i, task := range completedTasks {
		cycleTime := task.CompletedAt.Sub(task.CreatedAt)
		cycleTimes[i] = cycleTime

		// Group by type
		if task.Type != "" {
			byType[task.Type] = append(byType[task.Type], cycleTime)
		}

		// Group by priority
		priority := string(task.Priority)
		byPriority[priority] = append(byPriority[priority], cycleTime)
	}

	// Sort cycle times for percentile calculations
	sort.Slice(cycleTimes, func(i, j int) bool {
		return cycleTimes[i] < cycleTimes[j]
	})

	// Calculate average
	var totalTime time.Duration
	for _, ct := range cycleTimes {
		totalTime += ct
	}
	averageCycleTime := totalTime / time.Duration(len(cycleTimes))

	// Calculate median
	medianCycleTime := cycleTimes[len(cycleTimes)/2]

	// Calculate P90
	p90Index := int(float64(len(cycleTimes)) * 0.9)
	if p90Index >= len(cycleTimes) {
		p90Index = len(cycleTimes) - 1
	}
	p90CycleTime := cycleTimes[p90Index]

	// Calculate averages by type and priority
	typeAverages := make(map[string]time.Duration)
	for taskType, times := range byType {
		var total time.Duration
		for _, t := range times {
			total += t
		}
		typeAverages[taskType] = total / time.Duration(len(times))
	}

	priorityAverages := make(map[string]time.Duration)
	for priority, times := range byPriority {
		var total time.Duration
		for _, t := range times {
			total += t
		}
		priorityAverages[priority] = total / time.Duration(len(times))
	}

	// Create distribution (simplified)
	distribution := make([]entities.CycleTimePoint, 0)
	buckets := 10
	maxTime := cycleTimes[len(cycleTimes)-1]
	bucketSize := maxTime / time.Duration(buckets)

	for i := 0; i < buckets; i++ {
		bucketStart := time.Duration(i) * bucketSize
		bucketEnd := time.Duration(i+1) * bucketSize
		count := 0

		for _, ct := range cycleTimes {
			if ct >= bucketStart && ct < bucketEnd {
				count++
			}
		}

		if count > 0 {
			distribution = append(distribution, entities.CycleTimePoint{
				Duration:   bucketStart + bucketSize/2, // Mid-point of bucket
				Count:      count,
				Percentile: float64(i+1) / float64(buckets),
			})
		}
	}

	return &entities.CycleTimeMetrics{
		AverageCycleTime: averageCycleTime,
		MedianCycleTime:  medianCycleTime,
		P90CycleTime:     p90CycleTime,
		ByType:           typeAverages,
		ByPriority:       priorityAverages,
		Distribution:     distribution,
		LeadTime:         averageCycleTime, // Simplified: lead time = cycle time
		WaitTime:         0,                // Would need more data to calculate actual wait time
	}, nil
}

// GenerateWorkflowMetrics generates comprehensive workflow metrics
func (ae *analyticsEngineImpl) GenerateWorkflowMetrics(
	ctx context.Context,
	repository string,
	period entities.TimePeriod,
) (*entities.WorkflowMetrics, error) {
	// Get tasks and sessions for the period
	tasks, err := ae.taskRepo.FindByTimeRange(ctx, repository, period.Start, period.End)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	sessions, err := ae.sessionRepo.FindByTimeRange(ctx, repository, period.Start, period.End)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	// Calculate individual metrics
	productivity, err := ae.CalculateProductivityMetrics(ctx, tasks, sessions)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate productivity metrics: %w", err)
	}

	velocity, err := ae.CalculateVelocityMetrics(ctx, tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate velocity metrics: %w", err)
	}

	cycleTime, err := ae.CalculateCycleTimeMetrics(ctx, tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate cycle time metrics: %w", err)
	}

	bottleneckPtrs, err := ae.DetectBottlenecks(ctx, tasks, sessions)
	if err != nil {
		return nil, fmt.Errorf("failed to detect bottlenecks: %w", err)
	}

	// Convert []*entities.Bottleneck to []entities.Bottleneck
	bottlenecks := make([]entities.Bottleneck, len(bottleneckPtrs))
	for i, bp := range bottleneckPtrs {
		bottlenecks[i] = *bp
	}

	trends, err := ae.AnalyzeTrends(ctx, repository, period)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze trends: %w", err)
	}

	// Calculate completion metrics
	totalTasks := len(tasks)
	completed := 0
	inProgress := 0
	cancelled := 0

	for _, task := range tasks {
		switch task.Status {
		case entities.StatusCompleted:
			completed++
		case entities.StatusInProgress:
			inProgress++
		case entities.StatusCancelled:
			cancelled++
		}
	}

	completionRate := 0.0
	if totalTasks > 0 {
		completionRate = float64(completed) / float64(totalTasks)
	}

	completion := entities.CompletionMetrics{
		TotalTasks:     totalTasks,
		Completed:      completed,
		InProgress:     inProgress,
		Cancelled:      cancelled,
		CompletionRate: completionRate,
		AverageTime:    cycleTime.AverageCycleTime,
		ByStatus:       map[string]int{"completed": completed, "in_progress": inProgress, "cancelled": cancelled},
		ByPriority:     make(map[string]int),
		OnTimeRate:     0.8, // Simplified
		QualityScore:   0.9, // Simplified
	}

	// Pattern metrics (simplified)
	patterns := entities.PatternMetrics{
		TotalPatterns:    0,
		ActivePatterns:   0,
		PatternUsage:     make(map[string]int),
		SuccessRates:     make(map[string]float64),
		TopPatterns:      []entities.PatternUsage{},
		PatternEvolution: []entities.PatternEvolution{},
		AdherenceRate:    0.0,
	}

	return &entities.WorkflowMetrics{
		Repository:   repository,
		Period:       period,
		Productivity: *productivity,
		Completion:   completion,
		Velocity:     *velocity,
		CycleTime:    *cycleTime,
		Patterns:     patterns,
		Bottlenecks:  bottlenecks,
		Trends:       *trends,
		Comparisons:  nil, // Would need historical data
		GeneratedAt:  time.Now(),
		Metadata:     make(map[string]interface{}),
	}, nil
}

// DetectBottlenecks detects workflow bottlenecks
func (ae *analyticsEngineImpl) DetectBottlenecks(
	ctx context.Context,
	tasks []*entities.Task,
	sessions []*entities.Session,
) ([]*entities.Bottleneck, error) {
	var bottlenecks []*entities.Bottleneck

	// Analyze task duration bottlenecks
	if len(tasks) > 0 {
		durationAnalysis := ae.AnalyzeTaskDurations(tasks)

		// Tasks taking too long are bottlenecks
		for _, outlier := range durationAnalysis.Outliers {
			severity := entities.BottleneckSeverityMedium
			if outlier.DeviationPct > 200 {
				severity = entities.BottleneckSeverityHigh
			}
			if outlier.DeviationPct > 500 {
				severity = entities.BottleneckSeverityCritical
			}

			bottleneck := &entities.Bottleneck{
				Type:          "task_duration",
				Description:   fmt.Sprintf("Task '%s' took %.1f%% longer than expected", outlier.Content[:minInt(50, len(outlier.Content))], outlier.DeviationPct),
				Impact:        outlier.Duration.Hours(),
				Frequency:     1,
				Severity:      severity,
				Suggestions:   []string{"Break down into smaller tasks", "Review task complexity", "Allocate more resources"},
				AffectedTasks: []string{outlier.TaskID},
				DetectedAt:    time.Now(),
				Metadata:      map[string]interface{}{"deviation_pct": outlier.DeviationPct},
			}
			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	// Analyze session interruption bottlenecks
	if len(sessions) > 0 {
		totalInterruptions := 0
		highInterruptionSessions := 0

		for _, session := range sessions {
			interruptions := len(session.Interruptions)
			totalInterruptions += interruptions

			if interruptions > 5 { // Threshold for high interruptions
				highInterruptionSessions++
			}
		}

		if highInterruptionSessions > 0 {
			avgInterruptions := float64(totalInterruptions) / float64(len(sessions))
			severity := entities.BottleneckSeverityMedium
			if avgInterruptions > 10 {
				severity = entities.BottleneckSeverityHigh
			}

			bottleneck := &entities.Bottleneck{
				Type:        "interruptions",
				Description: fmt.Sprintf("High interruption rate: %.1f per session", avgInterruptions),
				Impact:      avgInterruptions * 0.25, // 15 minutes per interruption
				Frequency:   highInterruptionSessions,
				Severity:    severity,
				Suggestions: []string{"Schedule focused work blocks", "Turn off notifications", "Communicate availability to team"},
				DetectedAt:  time.Now(),
				Metadata:    map[string]interface{}{"avg_interruptions": avgInterruptions},
			}
			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	return bottlenecks, nil
}

// AnalyzeTrends analyzes trends across metrics
func (ae *analyticsEngineImpl) AnalyzeTrends(
	ctx context.Context,
	repository string,
	period entities.TimePeriod,
) (*entities.TrendAnalysis, error) {
	// For now, return a simplified trend analysis
	// In a real implementation, this would analyze historical data

	return &entities.TrendAnalysis{
		ProductivityTrend: entities.Trend{
			Direction:   entities.TrendDirectionStable,
			Strength:    0.5,
			Confidence:  0.6,
			ChangeRate:  0.0,
			StartValue:  0.0,
			EndValue:    0.0,
			TrendLine:   []entities.TrendPoint{},
			Description: "No significant productivity trend detected",
		},
		VelocityTrend: entities.Trend{
			Direction:   entities.TrendDirectionStable,
			Strength:    0.5,
			Confidence:  0.6,
			ChangeRate:  0.0,
			StartValue:  0.0,
			EndValue:    0.0,
			TrendLine:   []entities.TrendPoint{},
			Description: "No significant velocity trend detected",
		},
		QualityTrend: entities.Trend{
			Direction:   entities.TrendDirectionStable,
			Strength:    0.5,
			Confidence:  0.6,
			ChangeRate:  0.0,
			StartValue:  0.0,
			EndValue:    0.0,
			TrendLine:   []entities.TrendPoint{},
			Description: "No significant quality trend detected",
		},
		EfficiencyTrend: entities.Trend{
			Direction:   entities.TrendDirectionStable,
			Strength:    0.5,
			Confidence:  0.6,
			ChangeRate:  0.0,
			StartValue:  0.0,
			EndValue:    0.0,
			TrendLine:   []entities.TrendPoint{},
			Description: "No significant efficiency trend detected",
		},
		Predictions: []entities.Prediction{},
		Seasonality: entities.Seasonality{
			HasSeasonality: false,
			Patterns:       []entities.SeasonalPattern{},
			WeeklyPattern:  make(map[string]float64),
			MonthlyPattern: make(map[string]float64),
			HourlyPattern:  make(map[int]float64),
		},
	}, nil
}
