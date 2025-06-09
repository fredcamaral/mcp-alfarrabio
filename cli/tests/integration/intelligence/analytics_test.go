//go:build integration
// +build integration

package intelligence

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/services"
	"lerian-mcp-memory-cli/tests/testutils"
)

type AnalyticsTestSuite struct {
	suite.Suite
	taskStore        *testutils.MockTaskStorage
	patternStore     *testutils.MockPatternStorage
	sessionStore     *testutils.MockSessionStorage
	analyticsService services.AnalyticsService
	visualizer       *testutils.MockVisualizer
	exporter         *testutils.MockAnalyticsExporter
	testData         *testutils.TestDataGenerator
}

func (s *AnalyticsTestSuite) SetupSuite() {
	s.taskStore = testutils.NewMockTaskStorage()
	s.patternStore = testutils.NewMockPatternStorage()
	s.sessionStore = testutils.NewMockSessionStorage()
	s.visualizer = testutils.NewMockVisualizer()
	s.exporter = testutils.NewMockAnalyticsExporter()
	s.testData = testutils.NewTestDataGenerator()

	s.analyticsService = services.NewAnalyticsService(services.AnalyticsServiceDependencies{
		TaskStore:    s.taskStore,
		PatternStore: s.patternStore,
		SessionStore: s.sessionStore,
		Visualizer:   s.visualizer,
		Exporter:     s.exporter,
		Calculator:   services.NewMetricsCalculator(),
		Logger:       slog.Default(),
	})
}

func (s *AnalyticsTestSuite) TearDownTest() {
	s.taskStore.Clear()
	s.patternStore.Clear()
	s.sessionStore.Clear()
	s.visualizer.Reset()
	s.exporter.Reset()
}

func (s *AnalyticsTestSuite) TestWorkflowMetricsAccuracy() {
	ctx := context.Background()

	// Generate controlled dataset with known metrics
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()
	period := entities.TimePeriod{Start: startDate, End: endDate}

	// Create tasks with predictable patterns
	tasks := s.generateControlledTasks("analytics-accuracy", startDate, 30)
	sessions := s.generateControlledSessions("analytics-accuracy", startDate, 30)

	// Store data
	for _, task := range tasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	for _, session := range sessions {
		err := s.sessionStore.Create(ctx, session)
		s.Require().NoError(err)
	}

	// Generate metrics
	metrics, err := s.analyticsService.GetWorkflowMetrics(ctx, "analytics-accuracy", period)
	s.Require().NoError(err)

	// Verify productivity metrics accuracy
	expectedCompletedTasks := s.countCompletedTasks(tasks)
	expectedTotalTasks := len(tasks)
	expectedCompletionRate := float64(expectedCompletedTasks) / float64(expectedTotalTasks)

	s.Assert().Equal(expectedTotalTasks, metrics.Completion.TotalTasks, "Total tasks should match")
	s.Assert().Equal(expectedCompletedTasks, metrics.Completion.Completed, "Completed tasks should match")
	s.Assert().InDelta(expectedCompletionRate, metrics.Completion.CompletionRate, 0.01, "Completion rate should be accurate")

	// Verify productivity score is within reasonable range
	s.Assert().Greater(metrics.Productivity.Score, 0.0, "Productivity score should be positive")
	s.Assert().LessOrEqual(metrics.Productivity.Score, 100.0, "Productivity score should not exceed 100")

	// Verify tasks per day calculation
	activeDays := s.countActiveDays(sessions)
	expectedTasksPerDay := float64(expectedCompletedTasks) / float64(activeDays)
	s.Assert().InDelta(expectedTasksPerDay, metrics.Productivity.TasksPerDay, 0.1, "Tasks per day should be accurate")

	s.T().Logf("Metrics accuracy verification:")
	s.T().Logf("  Total tasks: %d (expected: %d)", metrics.Completion.TotalTasks, expectedTotalTasks)
	s.T().Logf("  Completed: %d (expected: %d)", metrics.Completion.Completed, expectedCompletedTasks)
	s.T().Logf("  Completion rate: %.2f (expected: %.2f)", metrics.Completion.CompletionRate, expectedCompletionRate)
	s.T().Logf("  Tasks per day: %.2f (expected: %.2f)", metrics.Productivity.TasksPerDay, expectedTasksPerDay)
}

func (s *AnalyticsTestSuite) TestVelocityCalculations() {
	ctx := context.Background()

	// Create tasks with specific weekly patterns
	startDate := time.Now().AddDate(0, 0, -56) // 8 weeks
	period := entities.TimePeriod{Start: startDate, End: time.Now()}

	// Generate tasks with increasing velocity pattern
	weeklyCompletions := []int{3, 4, 5, 6, 7, 8, 9, 10} // Steady increase
	tasks := s.generateWeeklyTasks("velocity-test", startDate, weeklyCompletions)

	for _, task := range tasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Calculate metrics
	metrics, err := s.analyticsService.GetWorkflowMetrics(ctx, "velocity-test", period)
	s.Require().NoError(err)

	// Verify weekly velocity data
	s.Assert().Len(metrics.Velocity.ByWeek, 8, "Should have 8 weeks of data")

	// Check each week's velocity
	for i, week := range metrics.Velocity.ByWeek {
		expectedVelocity := float64(weeklyCompletions[i])
		s.Assert().Equal(expectedVelocity, week.Velocity,
			"Week %d velocity should be %.0f", i+1, expectedVelocity)
	}

	// Verify trend detection (should be upward)
	s.Assert().Equal("up", metrics.Velocity.TrendDirection, "Should detect upward trend")
	s.Assert().Greater(metrics.Velocity.TrendPercentage, 0.0, "Trend percentage should be positive")

	// Verify current velocity (average of last 4 weeks)
	last4Weeks := weeklyCompletions[4:] // [7, 8, 9, 10]
	expectedCurrentVelocity := float64(7+8+9+10) / 4.0
	s.Assert().InDelta(expectedCurrentVelocity, metrics.Velocity.CurrentVelocity, 0.1,
		"Current velocity should be average of last 4 weeks")

	s.T().Logf("Velocity calculations:")
	s.T().Logf("  Current velocity: %.1f (expected: %.1f)", metrics.Velocity.CurrentVelocity, expectedCurrentVelocity)
	s.T().Logf("  Trend: %s (%.1f%%)", metrics.Velocity.TrendDirection, metrics.Velocity.TrendPercentage)
	s.T().Logf("  Consistency: %.2f", metrics.Velocity.Consistency)
}

func (s *AnalyticsTestSuite) TestCycleTimeAnalysis() {
	ctx := context.Background()

	// Create tasks with known cycle times
	startDate := time.Now().AddDate(0, 0, -14)
	period := entities.TimePeriod{Start: startDate, End: time.Now()}

	// Define cycle times by task type
	cycleTimesByType := map[string][]time.Duration{
		"bug-fix":  {2 * time.Hour, 3 * time.Hour, 1 * time.Hour, 4 * time.Hour},
		"feature":  {8 * time.Hour, 12 * time.Hour, 10 * time.Hour, 6 * time.Hour},
		"refactor": {4 * time.Hour, 5 * time.Hour, 3 * time.Hour, 6 * time.Hour},
	}

	tasks := s.generateTasksWithCycleTimes("cycle-time-test", startDate, cycleTimesByType)

	for _, task := range tasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Calculate metrics
	metrics, err := s.analyticsService.GetWorkflowMetrics(ctx, "cycle-time-test", period)
	s.Require().NoError(err)

	// Verify cycle time by type
	for taskType, expectedTimes := range cycleTimesByType {
		if actualTime, exists := metrics.CycleTime.ByType[taskType]; exists {
			expectedAvg := s.averageDuration(expectedTimes)
			s.Assert().InDelta(expectedAvg.Minutes(), actualTime.Minutes(), 30,
				"Average cycle time for %s should be approximately correct", taskType)
		}
	}

	// Verify overall statistics
	allCycleTimes := []time.Duration{}
	for _, times := range cycleTimesByType {
		allCycleTimes = append(allCycleTimes, times...)
	}

	expectedAvg := s.averageDuration(allCycleTimes)
	s.Assert().InDelta(expectedAvg.Minutes(), metrics.CycleTime.AverageCycleTime.Minutes(), 60,
		"Overall average cycle time should be approximately correct")

	// Verify distribution exists
	s.Assert().NotEmpty(metrics.CycleTime.Distribution, "Should have cycle time distribution")

	s.T().Logf("Cycle time analysis:")
	s.T().Logf("  Average: %v (expected: %v)", metrics.CycleTime.AverageCycleTime, expectedAvg)
	s.T().Logf("  Median: %v", metrics.CycleTime.MedianCycleTime)
	s.T().Logf("  P90: %v", metrics.CycleTime.P90CycleTime)
	s.T().Logf("  Distribution points: %d", len(metrics.CycleTime.Distribution))
}

func (s *AnalyticsTestSuite) TestBottleneckDetection() {
	ctx := context.Background()

	// Create tasks with deliberate bottlenecks
	startDate := time.Now().AddDate(0, 0, -21)
	period := entities.TimePeriod{Start: startDate, End: time.Now()}

	// Create slow tasks (bottleneck pattern)
	slowTasks := s.generateSlowTasks("bottleneck-test", startDate, "database-migration", 6*time.Hour, 5)
	fastTasks := s.generateFastTasks("bottleneck-test", startDate, "bug-fix", 1*time.Hour, 20)

	allTasks := append(slowTasks, fastTasks...)
	for _, task := range allTasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Detect bottlenecks
	bottlenecks, err := s.analyticsService.DetectBottlenecks(ctx, "bottleneck-test", period)
	s.Require().NoError(err)
	s.Assert().NotEmpty(bottlenecks, "Should detect bottlenecks")

	// Find database migration bottleneck
	var dbBottleneck *entities.Bottleneck
	for _, bottleneck := range bottlenecks {
		if bottleneck.Type == "cycle_time" &&
			bottleneck.Description != "" &&
			contains(bottleneck.Description, "database-migration") {
			dbBottleneck = bottleneck
			break
		}
	}

	s.Assert().NotNil(dbBottleneck, "Should detect database migration bottleneck")
	s.Assert().Greater(dbBottleneck.Impact, 10.0, "Database bottleneck should have significant impact")
	s.Assert().Equal(5, dbBottleneck.Frequency, "Should report correct frequency")
	s.Assert().Contains([]entities.BottleneckSeverity{
		entities.BottleneckSeverityHigh,
		entities.BottleneckSeverityCritical,
	}, dbBottleneck.Severity, "Should be high or critical severity")

	s.T().Logf("Bottleneck detection:")
	s.T().Logf("  Total bottlenecks: %d", len(bottlenecks))
	s.T().Logf("  DB migration impact: %.1f hours", dbBottleneck.Impact)
	s.T().Logf("  Severity: %s", dbBottleneck.Severity)
}

func (s *AnalyticsTestSuite) TestTrendAnalysis() {
	ctx := context.Background()

	// Create historical data for trend analysis
	startDate := time.Now().AddDate(0, -3, 0) // 3 months of data
	period := entities.TimePeriod{Start: startDate, End: time.Now()}

	// Generate tasks with changing productivity over time
	tasks := s.generateTrendTasks("trend-test", startDate, 12) // 12 weeks

	for _, task := range tasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Calculate metrics
	metrics, err := s.analyticsService.GetWorkflowMetrics(ctx, "trend-test", period)
	s.Require().NoError(err)

	// Verify trend analysis exists
	s.Assert().NotEmpty(metrics.Trends, "Should have trend analysis")
	s.Assert().NotNil(metrics.Trends.ProductivityTrend, "Should have productivity trend")
	s.Assert().NotNil(metrics.Trends.VelocityTrend, "Should have velocity trend")

	// Verify trend characteristics
	prodTrend := metrics.Trends.ProductivityTrend
	s.Assert().Greater(prodTrend.Confidence, 0.0, "Productivity trend should have confidence")
	s.Assert().Contains([]entities.TrendDirection{
		entities.TrendDirectionUp,
		entities.TrendDirectionDown,
		entities.TrendDirectionStable,
		entities.TrendDirectionVolatile,
	}, prodTrend.Direction, "Should have valid trend direction")

	velTrend := metrics.Trends.VelocityTrend
	s.Assert().Greater(velTrend.Confidence, 0.0, "Velocity trend should have confidence")

	s.T().Logf("Trend analysis:")
	s.T().Logf("  Productivity trend: %s (confidence: %.2f)", prodTrend.Direction, prodTrend.Confidence)
	s.T().Logf("  Velocity trend: %s (confidence: %.2f)", velTrend.Direction, velTrend.Confidence)
	s.T().Logf("  Productivity change rate: %.2f", prodTrend.ChangeRate)
}

func (s *AnalyticsTestSuite) TestProductivityReport() {
	ctx := context.Background()

	// Create comprehensive dataset
	startDate := time.Now().AddDate(0, 0, -30)
	period := entities.TimePeriod{Start: startDate, End: time.Now()}

	tasks := s.generateMixedQualityTasks("report-test", startDate, 30)
	sessions := s.generateControlledSessions("report-test", startDate, 30)

	for _, task := range tasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	for _, session := range sessions {
		err := s.sessionStore.Create(ctx, session)
		s.Require().NoError(err)
	}

	// Generate productivity report
	report, err := s.analyticsService.GetProductivityReport(ctx, "report-test", period)
	s.Require().NoError(err)

	// Verify report structure
	s.Assert().NotEmpty(report.Repository, "Report should have repository")
	s.Assert().Equal(period, report.Period, "Report should have correct period")
	s.Assert().Greater(report.OverallScore, 0.0, "Report should have overall score")
	s.Assert().NotEmpty(report.Insights, "Report should have insights")
	s.Assert().NotEmpty(report.Recommendations, "Report should have recommendations")
	s.Assert().NotEmpty(report.Charts, "Report should have charts")

	// Verify insights quality
	for _, insight := range report.Insights {
		s.Assert().NotEmpty(insight.Title, "Insight should have title")
		s.Assert().NotEmpty(insight.Description, "Insight should have description")
		s.Assert().Greater(insight.Confidence, 0.0, "Insight should have confidence")
		s.Assert().Contains([]entities.InsightType{
			entities.InsightTypePattern,
			entities.InsightTypeAntiPattern,
			entities.InsightTypeBestPractice,
		}, insight.Type, "Insight should have valid type")
	}

	// Verify recommendations quality
	for _, rec := range report.Recommendations {
		s.Assert().NotEmpty(rec.Title, "Recommendation should have title")
		s.Assert().NotEmpty(rec.Description, "Recommendation should have description")
		s.Assert().Greater(rec.Impact, 0.0, "Recommendation should have impact")
		s.Assert().Contains([]entities.RecommendationPriority{
			entities.RecommendationPriorityLow,
			entities.RecommendationPriorityMedium,
			entities.RecommendationPriorityHigh,
			entities.RecommendationPriorityCritical,
		}, rec.Priority, "Recommendation should have valid priority")
	}

	s.T().Logf("Productivity report:")
	s.T().Logf("  Overall score: %.1f/100", report.OverallScore)
	s.T().Logf("  Insights: %d", len(report.Insights))
	s.T().Logf("  Recommendations: %d", len(report.Recommendations))
	s.T().Logf("  Charts: %d", len(report.Charts))
}

func (s *AnalyticsTestSuite) TestVisualizationGeneration() {
	ctx := context.Background()

	// Create sample metrics
	metrics := s.createSampleMetrics()

	// Test terminal visualization
	visualization, err := s.analyticsService.GenerateVisualization(ctx, metrics, entities.VisFormatTerminal)
	s.Require().NoError(err)
	s.Assert().NotEmpty(visualization, "Should generate terminal visualization")

	// Verify visualizer was called
	s.Assert().True(s.visualizer.WasCalled("GenerateVisualization"), "Visualizer should be called")

	s.T().Logf("Visualization generated: %d bytes", len(visualization))
}

func (s *AnalyticsTestSuite) TestAnalyticsExport() {
	ctx := context.Background()

	// Create sample data
	startDate := time.Now().AddDate(0, 0, -7)
	period := entities.TimePeriod{Start: startDate, End: time.Now()}

	tasks := s.generateSimpleTasks("export-test", 10)
	for _, task := range tasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Test JSON export
	jsonFile, err := s.analyticsService.ExportAnalytics(ctx, "export-test", period, entities.ExportFormatJSON)
	s.Require().NoError(err)
	s.Assert().NotEmpty(jsonFile, "Should return JSON file path")

	// Test CSV export
	csvFile, err := s.analyticsService.ExportAnalytics(ctx, "export-test", period, entities.ExportFormatCSV)
	s.Require().NoError(err)
	s.Assert().NotEmpty(csvFile, "Should return CSV file path")

	// Verify exporter was called
	s.Assert().True(s.exporter.WasCalled("Export"), "Exporter should be called")

	s.T().Logf("Export files: JSON=%s, CSV=%s", jsonFile, csvFile)
}

func (s *AnalyticsTestSuite) TestAnalyticsPerformance() {
	ctx := context.Background()

	// Generate large dataset
	startDate := time.Now().AddDate(0, 0, -90)
	period := entities.TimePeriod{Start: startDate, End: time.Now()}

	largeTasks := s.testData.GenerateRandomTasks("perf-analytics", 5000)
	largeSessions := s.testData.GenerateRandomSessions("perf-analytics", 1000)

	for _, task := range largeTasks {
		s.taskStore.Create(ctx, task)
	}

	for _, session := range largeSessions {
		s.sessionStore.Create(ctx, session)
	}

	// Measure analytics performance
	start := time.Now()
	metrics, err := s.analyticsService.GetWorkflowMetrics(ctx, "perf-analytics", period)
	analyticsTime := time.Since(start)

	s.Require().NoError(err)
	s.Assert().Less(analyticsTime, 10*time.Second, "Analytics should complete within 10 seconds")

	// Measure report generation performance
	start = time.Now()
	report, err := s.analyticsService.GetProductivityReport(ctx, "perf-analytics", period)
	reportTime := time.Since(start)

	s.Require().NoError(err)
	s.Assert().Less(reportTime, 15*time.Second, "Report generation should complete within 15 seconds")

	s.T().Logf("Performance metrics:")
	s.T().Logf("  Analytics generation: %v for %d tasks", analyticsTime, len(largeTasks))
	s.T().Logf("  Report generation: %v", reportTime)
	s.T().Logf("  Productivity score: %.1f", metrics.Productivity.Score)
	s.T().Logf("  Overall score: %.1f", report.OverallScore)
}

// Helper methods for test data generation

func (s *AnalyticsTestSuite) generateControlledTasks(repository string, startDate time.Time, days int) []*entities.Task {
	var tasks []*entities.Task

	// Generate 5 tasks per day, 80% completion rate
	for day := 0; day < days; day++ {
		dayStart := startDate.Add(time.Duration(day) * 24 * time.Hour)

		for i := 0; i < 5; i++ {
			task := s.testData.CreateTask(
				fmt.Sprintf("Task %d-%d", day+1, i+1),
				"medium",
				"completed", // 80% will be completed, 20% pending
				map[string]interface{}{
					"day": day + 1,
				},
			)

			if i == 4 { // Make every 5th task pending (20%)
				task.Status = "pending"
			}

			task.Repository = repository
			task.CreatedAt = dayStart.Add(time.Duration(i) * time.Hour)
			task.UpdatedAt = task.CreatedAt.Add(2 * time.Hour)

			if task.Status == "completed" {
				task.Metadata["completed_at"] = task.UpdatedAt
				task.Metadata["duration"] = 2 * time.Hour
			}

			tasks = append(tasks, task)
		}
	}

	return tasks
}

func (s *AnalyticsTestSuite) generateControlledSessions(repository string, startDate time.Time, days int) []*entities.Session {
	var sessions []*entities.Session

	// Generate 2 sessions per day
	for day := 0; day < days; day++ {
		dayStart := startDate.Add(time.Duration(day) * 24 * time.Hour)

		// Morning session
		morningSession := &entities.Session{
			ID:         uuid.New().String(),
			Repository: repository,
			Duration:   4 * time.Hour,
			CreatedAt:  dayStart.Add(9 * time.Hour),  // 9 AM
			UpdatedAt:  dayStart.Add(13 * time.Hour), // 1 PM
		}

		// Afternoon session
		afternoonSession := &entities.Session{
			ID:         uuid.New().String(),
			Repository: repository,
			Duration:   3 * time.Hour,
			CreatedAt:  dayStart.Add(14 * time.Hour), // 2 PM
			UpdatedAt:  dayStart.Add(17 * time.Hour), // 5 PM
		}

		sessions = append(sessions, morningSession, afternoonSession)
	}

	return sessions
}

func (s *AnalyticsTestSuite) generateWeeklyTasks(repository string, startDate time.Time, weeklyCompletions []int) []*entities.Task {
	var tasks []*entities.Task

	for week, completions := range weeklyCompletions {
		weekStart := startDate.Add(time.Duration(week) * 7 * 24 * time.Hour)

		for i := 0; i < completions; i++ {
			task := s.testData.CreateTask(
				fmt.Sprintf("Week %d Task %d", week+1, i+1),
				"medium",
				"completed",
				nil,
			)

			task.Repository = repository
			task.CreatedAt = weekStart.Add(time.Duration(i) * 24 * time.Hour)
			task.UpdatedAt = task.CreatedAt.Add(4 * time.Hour)
			task.Metadata["completed_at"] = task.UpdatedAt

			tasks = append(tasks, task)
		}
	}

	return tasks
}

func (s *AnalyticsTestSuite) generateTasksWithCycleTimes(repository string, startDate time.Time, cycleTimesByType map[string][]time.Duration) []*entities.Task {
	var tasks []*entities.Task
	taskIndex := 0

	for taskType, cycleTimes := range cycleTimesByType {
		for i, cycleTime := range cycleTimes {
			task := s.testData.CreateTask(
				fmt.Sprintf("%s task %d", taskType, i+1),
				"medium",
				"completed",
				map[string]interface{}{
					"type":     taskType,
					"duration": cycleTime,
				},
			)

			task.Repository = repository
			task.CreatedAt = startDate.Add(time.Duration(taskIndex) * 6 * time.Hour)
			task.UpdatedAt = task.CreatedAt.Add(cycleTime)
			task.Metadata["completed_at"] = task.UpdatedAt

			tasks = append(tasks, task)
			taskIndex++
		}
	}

	return tasks
}

func (s *AnalyticsTestSuite) generateSlowTasks(repository string, startDate time.Time, taskType string, avgDuration time.Duration, count int) []*entities.Task {
	var tasks []*entities.Task

	for i := 0; i < count; i++ {
		// Add some variance (±20%)
		variance := time.Duration(float64(avgDuration) * 0.2 * (rand.Float64() - 0.5))
		duration := avgDuration + variance

		task := s.testData.CreateTask(
			fmt.Sprintf("%s task %d", taskType, i+1),
			"high",
			"completed",
			map[string]interface{}{
				"type":     taskType,
				"duration": duration,
			},
		)

		task.Repository = repository
		task.CreatedAt = startDate.Add(time.Duration(i) * 12 * time.Hour)
		task.UpdatedAt = task.CreatedAt.Add(duration)
		task.Metadata["completed_at"] = task.UpdatedAt

		tasks = append(tasks, task)
	}

	return tasks
}

func (s *AnalyticsTestSuite) generateFastTasks(repository string, startDate time.Time, taskType string, avgDuration time.Duration, count int) []*entities.Task {
	var tasks []*entities.Task

	for i := 0; i < count; i++ {
		// Add some variance (±30%)
		variance := time.Duration(float64(avgDuration) * 0.3 * (rand.Float64() - 0.5))
		duration := avgDuration + variance
		if duration < 0 {
			duration = 30 * time.Minute
		}

		task := s.testData.CreateTask(
			fmt.Sprintf("%s task %d", taskType, i+1),
			"low",
			"completed",
			map[string]interface{}{
				"type":     taskType,
				"duration": duration,
			},
		)

		task.Repository = repository
		task.CreatedAt = startDate.Add(time.Duration(i) * 2 * time.Hour)
		task.UpdatedAt = task.CreatedAt.Add(duration)
		task.Metadata["completed_at"] = task.UpdatedAt

		tasks = append(tasks, task)
	}

	return tasks
}

func (s *AnalyticsTestSuite) generateTrendTasks(repository string, startDate time.Time, weeks int) []*entities.Task {
	var tasks []*entities.Task

	// Create improving trend: start with 3 tasks/week, end with 8 tasks/week
	for week := 0; week < weeks; week++ {
		weekStart := startDate.Add(time.Duration(week) * 7 * 24 * time.Hour)
		tasksThisWeek := 3 + (5 * week / weeks) // Linear increase from 3 to 8

		for i := 0; i < tasksThisWeek; i++ {
			task := s.testData.CreateTask(
				fmt.Sprintf("Trend week %d task %d", week+1, i+1),
				"medium",
				"completed",
				map[string]interface{}{
					"week": week + 1,
				},
			)

			task.Repository = repository
			task.CreatedAt = weekStart.Add(time.Duration(i) * 24 * time.Hour)
			task.UpdatedAt = task.CreatedAt.Add(3 * time.Hour)
			task.Metadata["completed_at"] = task.UpdatedAt

			tasks = append(tasks, task)
		}
	}

	return tasks
}

func (s *AnalyticsTestSuite) generateMixedQualityTasks(repository string, startDate time.Time, days int) []*entities.Task {
	var tasks []*entities.Task

	statuses := []string{"completed", "completed", "completed", "in_progress", "cancelled"} // 60% completed
	priorities := []string{"high", "medium", "medium", "low"}                               // Mixed priorities

	for day := 0; day < days; day++ {
		dayStart := startDate.Add(time.Duration(day) * 24 * time.Hour)

		for i := 0; i < 3; i++ { // 3 tasks per day
			status := statuses[rand.Intn(len(statuses))]
			priority := priorities[rand.Intn(len(priorities))]

			task := s.testData.CreateTask(
				fmt.Sprintf("Mixed day %d task %d", day+1, i+1),
				priority,
				status,
				map[string]interface{}{
					"day":     day + 1,
					"quality": rand.Float64(), // Random quality score
				},
			)

			task.Repository = repository
			task.CreatedAt = dayStart.Add(time.Duration(i) * 4 * time.Hour)

			if status == "completed" {
				task.UpdatedAt = task.CreatedAt.Add(2 * time.Hour)
				task.Metadata["completed_at"] = task.UpdatedAt
				task.Metadata["duration"] = 2 * time.Hour
			} else {
				task.UpdatedAt = task.CreatedAt.Add(30 * time.Minute)
			}

			tasks = append(tasks, task)
		}
	}

	return tasks
}

func (s *AnalyticsTestSuite) generateSimpleTasks(repository string, count int) []*entities.Task {
	var tasks []*entities.Task

	for i := 0; i < count; i++ {
		task := s.testData.CreateTask(
			fmt.Sprintf("Simple task %d", i+1),
			"medium",
			"completed",
			nil,
		)

		task.Repository = repository
		task.CreatedAt = time.Now().Add(time.Duration(-i) * time.Hour)
		task.UpdatedAt = task.CreatedAt.Add(2 * time.Hour)
		task.Metadata["completed_at"] = task.UpdatedAt

		tasks = append(tasks, task)
	}

	return tasks
}

func (s *AnalyticsTestSuite) createSampleMetrics() *entities.WorkflowMetrics {
	return &entities.WorkflowMetrics{
		Repository: "sample-repo",
		Period: entities.TimePeriod{
			Start: time.Now().AddDate(0, 0, -7),
			End:   time.Now(),
		},
		Productivity: entities.ProductivityMetrics{
			Score:       75.5,
			TasksPerDay: 4.2,
			FocusTime:   3 * time.Hour,
		},
		Completion: entities.CompletionMetrics{
			TotalTasks:     50,
			Completed:      40,
			CompletionRate: 0.8,
		},
		Velocity: entities.VelocityMetrics{
			CurrentVelocity: 28.0,
			TrendDirection:  "up",
		},
		GeneratedAt: time.Now(),
	}
}

// Utility functions

func (s *AnalyticsTestSuite) countCompletedTasks(tasks []*entities.Task) int {
	count := 0
	for _, task := range tasks {
		if task.Status == "completed" {
			count++
		}
	}
	return count
}

func (s *AnalyticsTestSuite) countActiveDays(sessions []*entities.Session) int {
	daySet := make(map[string]bool)
	for _, session := range sessions {
		day := session.CreatedAt.Format("2006-01-02")
		daySet[day] = true
	}
	return len(daySet)
}

func (s *AnalyticsTestSuite) averageDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	total := time.Duration(0)
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func TestAnalytics(t *testing.T) {
	suite.Run(t, new(AnalyticsTestSuite))
}
