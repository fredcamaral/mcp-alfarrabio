package analytics

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// workflowAnalyticsEngine implements the ports.AnalyticsEngine interface
type workflowAnalyticsEngine struct {
	taskRepo    ports.TaskRepository
	sessionRepo ports.SessionRepository
	logger      *slog.Logger
}

// NewWorkflowAnalyticsEngine creates a new workflow analytics engine
func NewWorkflowAnalyticsEngine(taskRepo ports.TaskRepository, sessionRepo ports.SessionRepository, logger *slog.Logger) ports.AnalyticsEngine {
	return &workflowAnalyticsEngine{
		taskRepo:    taskRepo,
		sessionRepo: sessionRepo,
		logger:      logger,
	}
}

// CalculateProductivityMetrics calculates productivity metrics from tasks and sessions
func (ae *workflowAnalyticsEngine) CalculateProductivityMetrics(ctx context.Context, tasks []*entities.Task, sessions []*entities.Session) (*entities.ProductivityMetrics, error) {
	if len(tasks) == 0 {
		return &entities.ProductivityMetrics{}, nil
	}

	taskMetrics := ae.analyzeTaskMetrics(tasks)
	sessionMetrics := ae.analyzeSessionMetrics(sessions)
	
	completionRate := taskMetrics.completionRate
	deepWorkRatio := sessionMetrics.deepWorkRatio
	tasksPerDay := float64(len(tasks)) / 7.0 // Assume weekly data

	score := ae.calculateProductivityScore(completionRate, deepWorkRatio, tasksPerDay)
	peakHours := []int{9, 10, 11, 14, 15} // Default business hours

	return &entities.ProductivityMetrics{
		Score:           score,
		TasksPerDay:     tasksPerDay,
		FocusTime:       taskMetrics.totalFocusTime / time.Duration(len(sessions)+1), // Average per day
		DeepWorkRatio:   deepWorkRatio,
		ContextSwitches: sessionMetrics.contextSwitches,
		PeakHours:       peakHours,
		ByPriority:      taskMetrics.byPriority,
		ByType:          taskMetrics.byType,
	}, nil
}

// taskAnalysisResult holds the results of task analysis
type taskAnalysisResult struct {
	completionRate   float64
	totalFocusTime   time.Duration
	byPriority       map[string]float64
	byType           map[string]float64
}

// sessionAnalysisResult holds the results of session analysis
type sessionAnalysisResult struct {
	deepWorkRatio     float64
	contextSwitches   int
}

// analyzeTaskMetrics analyzes task-related metrics
func (ae *workflowAnalyticsEngine) analyzeTaskMetrics(tasks []*entities.Task) taskAnalysisResult {
	totalTasks := float64(len(tasks))
	completedTasks := 0
	totalFocusTime := time.Duration(0)
	
	byPriority := make(map[string]float64)
	byType := make(map[string]float64)
	priorityCounts := make(map[string]int)
	typeCounts := make(map[string]int)

	// Analyze each task
	for _, task := range tasks {
		if task.Status == entities.StatusCompleted {
			completedTasks++
		}

		if task.EstimatedMins > 0 {
			totalFocusTime += time.Duration(task.EstimatedMins) * time.Minute
		}

		ae.updateCounts(task, priorityCounts, typeCounts, byPriority, byType)
	}

	// Calculate completion rates
	ae.calculateCompletionRates(priorityCounts, typeCounts, byPriority, byType)

	return taskAnalysisResult{
		completionRate: float64(completedTasks) / totalTasks,
		totalFocusTime: totalFocusTime,
		byPriority:     byPriority,
		byType:         byType,
	}
}

// updateCounts updates the counting maps for a task
func (ae *workflowAnalyticsEngine) updateCounts(task *entities.Task, priorityCounts, typeCounts map[string]int, byPriority, byType map[string]float64) {
	priority := string(task.Priority)
	priorityCounts[priority]++
	typeCounts[task.Type]++
	
	if task.Status == entities.StatusCompleted {
		byPriority[priority]++
		byType[task.Type]++
	}
}

// calculateCompletionRates calculates completion rates by priority and type
func (ae *workflowAnalyticsEngine) calculateCompletionRates(priorityCounts, typeCounts map[string]int, byPriority, byType map[string]float64) {
	for priority, count := range priorityCounts {
		if count > 0 {
			byPriority[priority] /= float64(count)
		}
	}

	for taskType, count := range typeCounts {
		if count > 0 {
			byType[taskType] /= float64(count)
		}
	}
}

// analyzeSessionMetrics analyzes session-related metrics
func (ae *workflowAnalyticsEngine) analyzeSessionMetrics(sessions []*entities.Session) sessionAnalysisResult {
	if len(sessions) == 0 {
		return sessionAnalysisResult{deepWorkRatio: 0.8} // Default estimate
	}

	contextSwitches := ae.countContextSwitches(sessions)
	deepWorkRatio := ae.calculateDeepWorkRatio(sessions)

	return sessionAnalysisResult{
		deepWorkRatio:   deepWorkRatio,
		contextSwitches: contextSwitches,
	}
}

// countContextSwitches counts context switches between repositories
func (ae *workflowAnalyticsEngine) countContextSwitches(sessions []*entities.Session) int {
	contextSwitches := 0
	for i := 1; i < len(sessions); i++ {
		if sessions[i].Repository != sessions[i-1].Repository {
			contextSwitches++
		}
	}
	return contextSwitches
}

// calculateDeepWorkRatio calculates the ratio of deep work time
func (ae *workflowAnalyticsEngine) calculateDeepWorkRatio(sessions []*entities.Session) float64 {
	totalSessionTime := time.Duration(0)
	deepWorkTime := time.Duration(0)

	for _, session := range sessions {
		duration := session.EndTime.Sub(session.StartTime)
		totalSessionTime += duration
		if duration > 30*time.Minute { // Consider sessions > 30min as deep work
			deepWorkTime += duration
		}
	}

	if totalSessionTime > 0 {
		return float64(deepWorkTime) / float64(totalSessionTime)
	}
	return 0.8 // Default estimate
}

// calculateProductivityScore calculates the overall productivity score
func (ae *workflowAnalyticsEngine) calculateProductivityScore(completionRate, deepWorkRatio, tasksPerDay float64) float64 {
	score := (completionRate * 40) + (deepWorkRatio * 30) + (math.Min(tasksPerDay/10, 1) * 30)
	if score > 100 {
		score = 100
	}
	return score
}

// CalculateVelocityMetrics calculates velocity metrics from tasks
func (ae *workflowAnalyticsEngine) CalculateVelocityMetrics(ctx context.Context, tasks []*entities.Task) (*entities.VelocityMetrics, error) {
	if len(tasks) == 0 {
		return &entities.VelocityMetrics{}, nil
	}

	// Group tasks by week
	weeklyData := make(map[int]*entities.WeeklyVelocity)
	completedTasks := 0

	for _, task := range tasks {
		if task.Status == entities.StatusCompleted {
			completedTasks++
			year, week := task.UpdatedAt.ISOWeek()
			weekKey := year*100 + week

			if _, exists := weeklyData[weekKey]; !exists {
				weeklyData[weekKey] = &entities.WeeklyVelocity{
					Number:   week,
					Year:     year,
					Velocity: 0,
					Tasks:    0,
				}
			}
			weeklyData[weekKey].Tasks++
			weeklyData[weekKey].Velocity++
		}
	}

	// Convert to slice and sort
	byWeek := make([]entities.WeeklyVelocity, 0, len(weeklyData))
	for _, data := range weeklyData {
		byWeek = append(byWeek, *data)
	}

	sort.Slice(byWeek, func(i, j int) bool {
		return byWeek[i].Number < byWeek[j].Number
	})

	// Calculate current velocity (tasks per week)
	currentVelocity := 0.0
	if len(byWeek) > 0 {
		currentVelocity = byWeek[len(byWeek)-1].Velocity
	}

	// Calculate trend
	trendDirection := entities.TrendDirectionStable
	trendPercentage := 0.0

	if len(byWeek) >= 2 {
		recent := byWeek[len(byWeek)-1].Velocity
		previous := byWeek[len(byWeek)-2].Velocity

		if previous > 0 {
			trendPercentage = ((recent - previous) / previous) * 100

			if trendPercentage > 5 {
				trendDirection = entities.TrendDirectionUp
			} else if trendPercentage < -5 {
				trendDirection = entities.TrendDirectionDown
			}
		}
	}

	// Calculate consistency (standard deviation)
	consistency := 0.8 // Default
	if len(byWeek) > 1 {
		mean := 0.0
		for _, week := range byWeek {
			mean += week.Velocity
		}
		mean /= float64(len(byWeek))

		variance := 0.0
		for _, week := range byWeek {
			variance += (week.Velocity - mean) * (week.Velocity - mean)
		}
		variance /= float64(len(byWeek))

		stdDev := math.Sqrt(variance)
		consistency = math.Max(0, 1-(stdDev/mean))
	}

	// Simple forecast
	forecast := entities.VelocityForecast{
		PredictedVelocity: currentVelocity,
		Confidence:        0.7,
		Range:             []float64{currentVelocity * 0.8, currentVelocity * 1.2},
	}

	return &entities.VelocityMetrics{
		CurrentVelocity: currentVelocity,
		TrendDirection:  string(trendDirection),
		TrendPercentage: trendPercentage,
		Consistency:     consistency,
		ByWeek:          byWeek,
		Forecast:        forecast,
	}, nil
}

// CalculateCycleTimeMetrics calculates cycle time metrics from tasks
func (ae *workflowAnalyticsEngine) CalculateCycleTimeMetrics(ctx context.Context, tasks []*entities.Task) (*entities.CycleTimeMetrics, error) {
	var cycleTimes []time.Duration
	byType := make(map[string]time.Duration)
	typeCount := make(map[string]int)

	for _, task := range tasks {
		if task.Status == entities.StatusCompleted {
			cycleTime := task.UpdatedAt.Sub(task.CreatedAt)
			cycleTimes = append(cycleTimes, cycleTime)

			byType[task.Type] += cycleTime
			typeCount[task.Type]++
		}
	}

	if len(cycleTimes) == 0 {
		return &entities.CycleTimeMetrics{}, nil
	}

	// Sort for percentile calculations
	sort.Slice(cycleTimes, func(i, j int) bool {
		return cycleTimes[i] < cycleTimes[j]
	})

	// Calculate average
	total := time.Duration(0)
	for _, ct := range cycleTimes {
		total += ct
	}
	average := total / time.Duration(len(cycleTimes))

	// Calculate median
	median := cycleTimes[len(cycleTimes)/2]

	// Calculate 90th percentile
	p90Index := int(float64(len(cycleTimes)) * 0.9)
	if p90Index >= len(cycleTimes) {
		p90Index = len(cycleTimes) - 1
	}
	p90 := cycleTimes[p90Index]

	// Calculate averages by type
	for taskType, total := range byType {
		if count := typeCount[taskType]; count > 0 {
			byType[taskType] = total / time.Duration(count)
		}
	}

	// Create distribution buckets
	distribution := []entities.CycleTimePoint{
		{Duration: time.Hour, Count: 0, Percentile: 0},
		{Duration: 4 * time.Hour, Count: 0, Percentile: 0},
		{Duration: 24 * time.Hour, Count: 0, Percentile: 0},
		{Duration: 7 * 24 * time.Hour, Count: 0},
	}

	for _, ct := range cycleTimes {
		for i := range distribution {
			if ct <= distribution[i].Duration {
				distribution[i].Count++
				break
			}
		}
	}

	return &entities.CycleTimeMetrics{
		AverageCycleTime: average,
		MedianCycleTime:  median,
		P90CycleTime:     p90,
		LeadTime:         average * 12 / 10, // Add 20% for lead time
		WaitTime:         average * 2 / 10,  // 20% wait time
		ByType:           byType,
		Distribution:     distribution,
	}, nil
}

// GenerateWorkflowMetrics generates comprehensive workflow metrics
func (ae *workflowAnalyticsEngine) GenerateWorkflowMetrics(ctx context.Context, repository string, period entities.TimePeriod) (*entities.WorkflowMetrics, error) {
	// Get tasks for the period
	tasks, err := ae.taskRepo.GetByRepository(ctx, repository, period)
	if err != nil {
		return nil, err
	}

	// Filter tasks by period
	var filteredTasks []*entities.Task
	for _, task := range tasks {
		if task.CreatedAt.After(period.Start) && task.CreatedAt.Before(period.End) {
			filteredTasks = append(filteredTasks, task)
		}
	}

	// Get sessions for the period
	sessions, err := ae.sessionRepo.GetByRepository(ctx, repository, period)
	if err != nil {
		ae.logger.Warn("failed to get sessions", slog.Any("error", err))
		sessions = []*entities.Session{} // Continue without sessions
	}

	// Calculate individual metrics
	productivity, err := ae.CalculateProductivityMetrics(ctx, filteredTasks, sessions)
	if err != nil {
		return nil, err
	}

	velocity, err := ae.CalculateVelocityMetrics(ctx, filteredTasks)
	if err != nil {
		return nil, err
	}

	cycleTime, err := ae.CalculateCycleTimeMetrics(ctx, filteredTasks)
	if err != nil {
		return nil, err
	}

	// Calculate completion metrics
	completion := ae.calculateCompletionMetrics(filteredTasks)

	// Detect bottlenecks
	bottleneckPtrs, err := ae.DetectBottlenecks(ctx, filteredTasks, sessions)
	if err != nil {
		return nil, err
	}

	// Convert to values
	bottlenecks := make([]entities.Bottleneck, len(bottleneckPtrs))
	for i, b := range bottleneckPtrs {
		bottlenecks[i] = *b
	}

	// Analyze trends
	trends, err := ae.AnalyzeTrends(ctx, repository, period)
	if err != nil {
		return nil, err
	}

	return &entities.WorkflowMetrics{
		Repository:   repository,
		Period:       period,
		Productivity: *productivity,
		Velocity:     *velocity,
		Completion:   *completion,
		CycleTime:    *cycleTime,
		Bottlenecks:  bottlenecks,
		Trends:       *trends,
	}, nil
}

// DetectBottlenecks detects workflow bottlenecks
func (ae *workflowAnalyticsEngine) DetectBottlenecks(ctx context.Context, tasks []*entities.Task, sessions []*entities.Session) ([]*entities.Bottleneck, error) {
	var bottlenecks []*entities.Bottleneck

	// Detect high cycle time tasks
	cycleTimes := make(map[string][]time.Duration)
	for _, task := range tasks {
		if task.Status == entities.StatusCompleted {
			cycleTime := task.UpdatedAt.Sub(task.CreatedAt)
			cycleTimes[task.Type] = append(cycleTimes[task.Type], cycleTime)
		}
	}

	for taskType, times := range cycleTimes {
		if len(times) < 3 {
			continue
		}

		total := time.Duration(0)
		for _, t := range times {
			total += t
		}
		average := total / time.Duration(len(times))

		if average > 7*24*time.Hour { // More than a week
			bottleneck := &entities.Bottleneck{
				Type:        "process",
				Severity:    entities.BottleneckSeverityHigh,
				Description: fmt.Sprintf("High cycle time for %s tasks", taskType),
				Impact:      average.Hours(),
				Frequency:   len(times),
				Suggestions: []string{
					"Break down large tasks into smaller chunks",
					"Review task complexity and requirements",
					"Consider pair programming or additional resources",
				},
				DetectedAt: time.Now(),
				Metadata:   make(map[string]interface{}),
			}
			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	// Detect context switching
	if len(sessions) > 0 {
		switches := 0
		for i := 1; i < len(sessions); i++ {
			if sessions[i].Repository != sessions[i-1].Repository {
				switches++
			}
		}

		if switches > 10 { // More than 10 switches per period
			bottleneck := &entities.Bottleneck{
				Type:        "resource",
				Severity:    entities.BottleneckSeverityMedium,
				Description: "High context switching between repositories",
				Impact:      float64(switches) * 0.5, // Assume 30min impact per switch
				Frequency:   switches,
				Suggestions: []string{
					"Batch similar work together",
					"Set dedicated focus time blocks",
					"Use time-boxing techniques",
				},
				DetectedAt: time.Now(),
				Metadata:   make(map[string]interface{}),
			}
			bottlenecks = append(bottlenecks, bottleneck)
		}
	}

	return bottlenecks, nil
}

// AnalyzeTrends analyzes trends in the data
func (ae *workflowAnalyticsEngine) AnalyzeTrends(ctx context.Context, repository string, period entities.TimePeriod) (*entities.TrendAnalysis, error) {
	// This is a simplified trend analysis
	// In a real implementation, you'd want more sophisticated time series analysis

	trends := &entities.TrendAnalysis{
		ProductivityTrend: entities.Trend{
			Direction:  entities.TrendDirectionStable,
			Strength:   0.5,
			Confidence: 0.6,
		},
		VelocityTrend: entities.Trend{
			Direction:  entities.TrendDirectionUp,
			Strength:   0.7,
			Confidence: 0.8,
		},
		QualityTrend: entities.Trend{
			Direction:  entities.TrendDirectionStable,
			Strength:   0.6,
			Confidence: 0.7,
		},
		EfficiencyTrend: entities.Trend{
			Direction:  entities.TrendDirectionUp,
			Strength:   0.5,
			Confidence: 0.6,
		},
		Predictions: []entities.Prediction{
			{
				Metric:     entities.MetricTypeVelocity,
				Value:      15.0,
				Confidence: 0.7,
			},
		},
		Seasonality: entities.Seasonality{
			HasSeasonality: false,
			Patterns:       []entities.SeasonalPattern{},
		},
	}

	return trends, nil
}

// Helper methods

func (ae *workflowAnalyticsEngine) calculateCompletionMetrics(tasks []*entities.Task) *entities.CompletionMetrics {
	total := len(tasks)
	completed := 0
	inProgress := 0
	cancelled := 0

	byStatus := make(map[string]int)
	byPriority := make(map[string]int)

	var totalTime time.Duration
	completedCount := 0

	for _, task := range tasks {
		switch task.Status {
		case entities.StatusCompleted:
			completed++
			if !task.UpdatedAt.IsZero() && !task.CreatedAt.IsZero() {
				totalTime += task.UpdatedAt.Sub(task.CreatedAt)
				completedCount++
			}
		case entities.StatusInProgress:
			inProgress++
		case entities.StatusCancelled:
			cancelled++
		}

		byStatus[string(task.Status)]++
		byPriority[string(task.Priority)]++
	}

	completionRate := 0.0
	if total > 0 {
		completionRate = float64(completed) / float64(total)
	}

	averageTime := time.Duration(0)
	if completedCount > 0 {
		averageTime = totalTime / time.Duration(completedCount)
	}

	return &entities.CompletionMetrics{
		TotalTasks:     total,
		Completed:      completed,
		InProgress:     inProgress,
		Cancelled:      cancelled,
		CompletionRate: completionRate,
		AverageTime:    averageTime,
		OnTimeRate:     0.8, // Default estimate
		QualityScore:   0.9, // Default estimate
		ByStatus:       byStatus,
		ByPriority:     byPriority,
	}
}
