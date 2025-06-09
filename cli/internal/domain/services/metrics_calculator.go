package services

import (
	"fmt"
	"math"
	"sort"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// MetricsCalculator provides utility methods for calculating various metrics
type MetricsCalculator struct {
	// Configuration for calculations
	FocusTimeThreshold time.Duration // Minimum time for a focus period
	BreakThreshold     time.Duration // Maximum break time within focus period
	DeepWorkThreshold  time.Duration // Minimum time for deep work
}

// NewMetricsCalculator creates a new metrics calculator with default settings
func NewMetricsCalculator() *MetricsCalculator {
	return &MetricsCalculator{
		FocusTimeThreshold: 25 * time.Minute, // Pomodoro-like
		BreakThreshold:     5 * time.Minute,
		DeepWorkThreshold:  90 * time.Minute,
	}
}

// Task-related calculations

// GetTaskDuration calculates the duration of a task from creation to completion
func (mc *MetricsCalculator) GetTaskDuration(task *entities.Task) time.Duration {
	if task.Status != "completed" {
		return 0
	}

	// Use CompletedAt field if available
	if task.CompletedAt != nil {
		return task.CompletedAt.Sub(task.CreatedAt)
	}

	// Fallback to updated time
	return task.UpdatedAt.Sub(task.CreatedAt)
}

// IsTaskOnTime checks if a task was completed within its estimated time
func (mc *MetricsCalculator) IsTaskOnTime(task *entities.Task) bool {
	if task.Status != "completed" {
		return false
	}

	// Use EstimatedMins field from task
	if task.EstimatedMins > 0 {
		estimatedDuration := time.Duration(task.EstimatedMins) * time.Minute
		actualDuration := mc.GetTaskDuration(task)
		return actualDuration <= estimatedDuration*120/100 // 20% buffer
	}

	return true // No estimate = assume on time
}

// CountActiveDays counts the number of days with actual work sessions
func (mc *MetricsCalculator) CountActiveDays(sessions []*entities.Session) int {
	daySet := make(map[string]bool)

	for _, session := range sessions {
		day := session.CreatedAt.Format("2006-01-02")
		daySet[day] = true
	}

	return len(daySet)
}

// CountCompletedTasks counts completed tasks in a slice
func (mc *MetricsCalculator) CountCompletedTasks(tasks []*entities.Task) int {
	count := 0
	for _, task := range tasks {
		if task.Status == "completed" {
			count++
		}
	}
	return count
}

// CalculateCompletionByPriority calculates completion rates grouped by priority
func (mc *MetricsCalculator) CalculateCompletionByPriority(tasks []*entities.Task) map[string]float64 {
	priorityCounts := make(map[string]int)
	priorityCompleted := make(map[string]int)
	rates := make(map[string]float64)

	for _, task := range tasks {
		priorityStr := string(task.Priority)
		priorityCounts[priorityStr]++
		if string(task.Status) == "completed" {
			priorityCompleted[priorityStr]++
		}
	}

	for priority, total := range priorityCounts {
		if total > 0 {
			rates[priority] = float64(priorityCompleted[priority]) / float64(total)
		}
	}

	return rates
}

// CalculateCompletionByType calculates completion rates grouped by task type
func (mc *MetricsCalculator) CalculateCompletionByType(tasks []*entities.Task) map[string]float64 {
	typeCounts := make(map[string]int)
	typeCompleted := make(map[string]int)
	rates := make(map[string]float64)

	for _, task := range tasks {
		typeCounts[task.Type]++
		if task.Status == "completed" {
			typeCompleted[task.Type]++
		}
	}

	for taskType, total := range typeCounts {
		if total > 0 {
			rates[taskType] = float64(typeCompleted[taskType]) / float64(total)
		}
	}

	return rates
}

// Session-related calculations

// CalculateAverageFocusTime calculates the average focus time per session
func (mc *MetricsCalculator) CalculateAverageFocusTime(sessions []*entities.Session) time.Duration {
	if len(sessions) == 0 {
		return 0
	}

	totalFocusTime := time.Duration(0)

	for _, session := range sessions {
		focusPeriods := mc.ExtractFocusPeriods(session)
		for _, period := range focusPeriods {
			totalFocusTime += period.Duration
		}
	}

	return totalFocusTime / time.Duration(len(sessions))
}

// FocusPeriod represents a period of focused work
type FocusPeriod struct {
	StartTime time.Time
	Duration  time.Duration
}

// ExtractFocusPeriods identifies focus periods within a session
func (mc *MetricsCalculator) ExtractFocusPeriods(session *entities.Session) []FocusPeriod {
	// This is a simplified implementation
	// In practice, this would analyze task creation/completion patterns
	// to identify continuous work periods

	var periods []FocusPeriod

	// For now, assume the entire session is a focus period if it's long enough
	sessionDuration := session.Duration
	if sessionDuration >= mc.FocusTimeThreshold {
		periods = append(periods, FocusPeriod{
			StartTime: session.CreatedAt,
			Duration:  sessionDuration,
		})
	}

	return periods
}

// FindPeakHours identifies the most productive hours of the day
func (mc *MetricsCalculator) FindPeakHours(sessions []*entities.Session) []int {
	hourCounts := make(map[int]int)

	for _, session := range sessions {
		hour := session.CreatedAt.Hour()
		hourCounts[hour]++
	}

	// Find hours with above-average activity
	totalSessions := len(sessions)
	averagePerHour := float64(totalSessions) / 24.0
	threshold := averagePerHour * 1.5 // 50% above average

	var peakHours []int
	for hour, count := range hourCounts {
		if float64(count) >= threshold {
			peakHours = append(peakHours, hour)
		}
	}

	sort.Ints(peakHours)
	return peakHours
}

// CountContextSwitches counts the number of context switches in sessions
func (mc *MetricsCalculator) CountContextSwitches(sessions []*entities.Session) int {
	// This would analyze task transitions within sessions
	// Simplified implementation
	if len(sessions) <= 1 {
		return 0
	}

	// Assume each session change is a context switch
	return len(sessions) - 1
}

// CalculateDeepWorkRatio calculates the ratio of deep work time to total time
func (mc *MetricsCalculator) CalculateDeepWorkRatio(sessions []*entities.Session) float64 {
	if len(sessions) == 0 {
		return 0
	}

	totalTime := time.Duration(0)
	deepWorkTime := time.Duration(0)

	for _, session := range sessions {
		totalTime += session.Duration
		if session.Duration >= mc.DeepWorkThreshold {
			deepWorkTime += session.Duration
		}
	}

	if totalTime == 0 {
		return 0
	}

	return float64(deepWorkTime) / float64(totalTime)
}

// Productivity score calculation

// CalculateProductivityScore calculates an overall productivity score
func (mc *MetricsCalculator) CalculateProductivityScore(
	metrics entities.ProductivityMetrics,
	tasks []*entities.Task,
	sessions []*entities.Session,
) float64 {
	score := 0.0
	factors := 0

	// Factor 1: Task completion rate (25%)
	if len(tasks) > 0 {
		completedCount := mc.CountCompletedTasks(tasks)
		completionRate := float64(completedCount) / float64(len(tasks))
		score += completionRate * 25
		factors++
	}

	// Factor 2: Tasks per day (20%)
	if metrics.TasksPerDay > 0 {
		// Normalize to 0-1 scale (assuming 5 tasks/day is excellent)
		normalizedTasksPerDay := math.Min(metrics.TasksPerDay/5.0, 1.0)
		score += normalizedTasksPerDay * 20
		factors++
	}

	// Factor 3: Focus time ratio (20%)
	if len(sessions) > 0 {
		totalSessionTime := time.Duration(0)
		for _, session := range sessions {
			totalSessionTime += session.Duration
		}

		if totalSessionTime > 0 {
			focusRatio := float64(metrics.FocusTime*time.Duration(len(sessions))) / float64(totalSessionTime)
			score += math.Min(focusRatio, 1.0) * 20
			factors++
		}
	}

	// Factor 4: Deep work ratio (15%)
	if metrics.DeepWorkRatio > 0 {
		score += metrics.DeepWorkRatio * 15
		factors++
	}

	// Factor 5: Priority completion balance (10%)
	if len(metrics.ByPriority) > 0 {
		highPriorityRate := metrics.ByPriority["high"]
		mediumPriorityRate := metrics.ByPriority["medium"]

		// Ideal is high completion for high priority, good for medium
		priorityScore := (highPriorityRate * 0.7) + (mediumPriorityRate * 0.3)
		score += priorityScore * 10
		factors++
	}

	// Factor 6: Context switch penalty (10%)
	if len(sessions) > 0 {
		// Fewer context switches = better score
		switchesPerSession := float64(metrics.ContextSwitches) / float64(len(sessions))
		switchPenalty := math.Max(0, 1.0-switchesPerSession/5.0) // Penalize >5 switches/session
		score += switchPenalty * 10
		factors++
	}

	if factors == 0 {
		return 0
	}

	// Normalize to 0-100 scale
	return score * float64(6) / float64(factors) // 6 total factors
}

// CalculateQualityScore calculates a quality score based on task characteristics
func (mc *MetricsCalculator) CalculateQualityScore(tasks []*entities.Task) float64 {
	if len(tasks) == 0 {
		return 0
	}

	qualitySum := 0.0
	qualityCount := 0

	for _, task := range tasks {
		if task.Status == "completed" {
			// Quality indicators:
			// 1. Task completed on time
			// 2. No rework needed (no status changes back to in_progress)
			// 3. Clear description (has description)

			taskQuality := 0.0

			if mc.IsTaskOnTime(task) {
				taskQuality += 0.4
			}

			// Task doesn't have Description field, skip this check
			// if task.Description != "" && len(task.Description) > 10 {
			//     taskQuality += 0.3
			// }

			// Assume no rework if task was completed quickly
			duration := mc.GetTaskDuration(task)
			if duration > 0 && duration < 24*time.Hour {
				taskQuality += 0.3
			}

			qualitySum += taskQuality
			qualityCount++
		}
	}

	if qualityCount == 0 {
		return 0
	}

	return qualitySum / float64(qualityCount)
}

// Time-series calculations

// GroupTasksByWeek groups tasks by week number within a period
func (mc *MetricsCalculator) GroupTasksByWeek(
	tasks []*entities.Task,
	period entities.TimePeriod,
) map[int][]*entities.Task {
	weeklyTasks := make(map[int][]*entities.Task)

	for _, task := range tasks {
		if period.Contains(task.CreatedAt) {
			_, week := task.CreatedAt.ISOWeek()
			weeklyTasks[week] = append(weeklyTasks[week], task)
		}
	}

	return weeklyTasks
}

// Duration calculations

// AverageDuration calculates the average of a slice of durations
func (mc *MetricsCalculator) AverageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}

// MedianDuration calculates the median of a slice of durations
func (mc *MetricsCalculator) MedianDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	if len(sorted)%2 == 0 {
		// Even number of elements
		mid1 := sorted[len(sorted)/2-1]
		mid2 := sorted[len(sorted)/2]
		return (mid1 + mid2) / 2
	}

	// Odd number of elements
	return sorted[len(sorted)/2]
}

// PercentileDuration calculates the nth percentile of durations
func (mc *MetricsCalculator) PercentileDuration(durations []time.Duration, percentile int) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := int(float64(len(sorted)) * float64(percentile) / 100.0)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// CreateCycleTimeDistribution creates a distribution of cycle times
func (mc *MetricsCalculator) CreateCycleTimeDistribution(durations []time.Duration) []entities.CycleTimePoint {
	if len(durations) == 0 {
		return []entities.CycleTimePoint{}
	}

	// Create histogram buckets
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Create 10 buckets
	numBuckets := 10
	maxDuration := sorted[len(sorted)-1]
	bucketSize := maxDuration / time.Duration(numBuckets)

	buckets := make([]int, numBuckets)

	for _, duration := range sorted {
		bucketIndex := int(duration / bucketSize)
		if bucketIndex >= numBuckets {
			bucketIndex = numBuckets - 1
		}
		buckets[bucketIndex]++
	}

	var distribution []entities.CycleTimePoint
	for i, count := range buckets {
		if count > 0 {
			duration := time.Duration(i+1) * bucketSize
			percentile := float64(i+1) * 100.0 / float64(numBuckets)

			distribution = append(distribution, entities.CycleTimePoint{
				Duration:   duration,
				Count:      count,
				Percentile: percentile,
			})
		}
	}

	return distribution
}

// Statistical calculations

// CoefficientOfVariation calculates the coefficient of variation for a set of values
func (mc *MetricsCalculator) CoefficientOfVariation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	if mean == 0 {
		return 0
	}

	// Calculate standard deviation
	sumSquaredDiffs := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiffs += diff * diff
	}
	variance := sumSquaredDiffs / float64(len(values))
	stdDev := math.Sqrt(variance)

	// Coefficient of variation = stddev / mean
	return stdDev / mean
}

// ForecastVelocity creates a velocity forecast based on historical data
func (mc *MetricsCalculator) ForecastVelocity(weeklyVelocities []entities.WeeklyVelocity) entities.VelocityForecast {
	forecast := entities.VelocityForecast{
		Method:    "simple_moving_average",
		UpdatedAt: time.Now(),
	}

	if len(weeklyVelocities) < 2 {
		forecast.Confidence = 0.1
		return forecast
	}

	// Use simple moving average of last 4 weeks (or all if fewer)
	recentWeeks := weeklyVelocities
	if len(recentWeeks) > 4 {
		recentWeeks = recentWeeks[len(recentWeeks)-4:]
	}

	sum := 0.0
	velocities := make([]float64, len(recentWeeks))

	for i, week := range recentWeeks {
		sum += week.Velocity
		velocities[i] = week.Velocity
	}

	forecast.PredictedVelocity = sum / float64(len(recentWeeks))

	// Calculate confidence based on consistency
	cv := mc.CoefficientOfVariation(velocities)
	forecast.Confidence = math.Max(0.1, 1.0-cv) // Lower CV = higher confidence

	// Calculate range (±20%)
	variance := forecast.PredictedVelocity * 0.2
	forecast.Range = []float64{
		math.Max(0, forecast.PredictedVelocity-variance),
		forecast.PredictedVelocity + variance,
	}

	return forecast
}

// Trend calculations

// CalculateProductivityTrend calculates productivity trend over time
func (mc *MetricsCalculator) CalculateProductivityTrend(
	tasks []*entities.Task,
	period entities.TimePeriod,
) entities.Trend {
	// Group tasks by week and calculate weekly productivity
	weeklyTasks := mc.GroupTasksByWeek(tasks, period)

	var trendPoints []entities.TrendPoint
	weekNumbers := make([]int, 0, len(weeklyTasks))

	for week := range weeklyTasks {
		weekNumbers = append(weekNumbers, week)
	}
	sort.Ints(weekNumbers)

	for _, week := range weekNumbers {
		weekTasks := weeklyTasks[week]
		completedCount := mc.CountCompletedTasks(weekTasks)

		// Simple productivity score: completed tasks / total tasks * 100
		productivity := 0.0
		if len(weekTasks) > 0 {
			productivity = float64(completedCount) / float64(len(weekTasks)) * 100
		}

		// Approximate week start time
		weekStart := period.Start.AddDate(0, 0, (week-1)*7)

		trendPoints = append(trendPoints, entities.TrendPoint{
			Time:  weekStart,
			Value: productivity,
		})
	}

	return mc.calculateTrendFromPoints(trendPoints)
}

// CalculateVelocityTrend calculates velocity trend over time
func (mc *MetricsCalculator) CalculateVelocityTrend(
	tasks []*entities.Task,
	period entities.TimePeriod,
) entities.Trend {
	weeklyTasks := mc.GroupTasksByWeek(tasks, period)

	var trendPoints []entities.TrendPoint
	weekNumbers := make([]int, 0, len(weeklyTasks))

	for week := range weeklyTasks {
		weekNumbers = append(weekNumbers, week)
	}
	sort.Ints(weekNumbers)

	for _, week := range weekNumbers {
		weekTasks := weeklyTasks[week]
		completedCount := mc.CountCompletedTasks(weekTasks)

		weekStart := period.Start.AddDate(0, 0, (week-1)*7)

		trendPoints = append(trendPoints, entities.TrendPoint{
			Time:  weekStart,
			Value: float64(completedCount),
		})
	}

	return mc.calculateTrendFromPoints(trendPoints)
}

// CalculateQualityTrend calculates quality trend over time
func (mc *MetricsCalculator) CalculateQualityTrend(
	tasks []*entities.Task,
	period entities.TimePeriod,
) entities.Trend {
	weeklyTasks := mc.GroupTasksByWeek(tasks, period)

	var trendPoints []entities.TrendPoint
	weekNumbers := make([]int, 0, len(weeklyTasks))

	for week := range weeklyTasks {
		weekNumbers = append(weekNumbers, week)
	}
	sort.Ints(weekNumbers)

	for _, week := range weekNumbers {
		weekTasks := weeklyTasks[week]
		qualityScore := mc.CalculateQualityScore(weekTasks) * 100

		weekStart := period.Start.AddDate(0, 0, (week-1)*7)

		trendPoints = append(trendPoints, entities.TrendPoint{
			Time:  weekStart,
			Value: qualityScore,
		})
	}

	return mc.calculateTrendFromPoints(trendPoints)
}

// calculateTrendFromPoints calculates trend characteristics from data points
func (mc *MetricsCalculator) calculateTrendFromPoints(points []entities.TrendPoint) entities.Trend {
	trend := entities.Trend{
		TrendLine: points,
		Direction: entities.TrendDirectionStable,
	}

	if len(points) < 2 {
		trend.Confidence = 0.1
		return trend
	}

	// Simple linear regression to determine trend
	n := float64(len(points))
	sumX, sumY, sumXY, sumXX := 0.0, 0.0, 0.0, 0.0

	for i, point := range points {
		x := float64(i)
		y := point.Value

		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// Calculate slope (change rate)
	denominator := n*sumXX - sumX*sumX
	if denominator == 0 {
		trend.Confidence = 0.1
		return trend
	}

	slope := (n*sumXY - sumX*sumY) / denominator

	// Set trend characteristics
	trend.StartValue = points[0].Value
	trend.EndValue = points[len(points)-1].Value
	trend.ChangeRate = slope

	// Determine direction
	if slope > 2 {
		trend.Direction = entities.TrendDirectionUp
	} else if slope < -2 {
		trend.Direction = entities.TrendDirectionDown
	} else {
		trend.Direction = entities.TrendDirectionStable
	}

	// Calculate strength based on how consistent the trend is
	// Use correlation coefficient as a proxy for strength
	meanX := sumX / n
	meanY := sumY / n

	ssXX := sumXX - n*meanX*meanX
	ssYY := 0.0
	ssXY := sumXY - n*meanX*meanY

	for _, point := range points {
		ssYY += (point.Value - meanY) * (point.Value - meanY)
	}

	if ssXX > 0 && ssYY > 0 {
		correlation := ssXY / math.Sqrt(ssXX*ssYY)
		trend.Strength = math.Abs(correlation)
		trend.Confidence = trend.Strength
	} else {
		trend.Strength = 0
		trend.Confidence = 0.1
	}

	// Generate description
	if trend.Strength > 0.7 {
		trend.Description = fmt.Sprintf("Strong %s trend", trend.Direction)
	} else if trend.Strength > 0.4 {
		trend.Description = fmt.Sprintf("Moderate %s trend", trend.Direction)
	} else {
		trend.Description = "No clear trend"
	}

	return trend
}
