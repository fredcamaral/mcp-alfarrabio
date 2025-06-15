package commands

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/services"

	"github.com/spf13/cobra"
)

// AnalyticsCommand creates the analytics command
func NewAnalyticsCommand(deps CommandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analytics [repository]",
		Short: "View productivity analytics and insights",
		Long: `Display comprehensive analytics including productivity metrics, velocity trends, 
workflow bottlenecks, and actionable insights.

Examples:
  # View analytics for current repository
  mcp-memory analytics

  # View analytics for specific repository
  mcp-memory analytics my-project

  # View analytics for last 7 days
  mcp-memory analytics --period 7d

  # Compare with previous period
  mcp-memory analytics --compare

  # Export analytics to file
  mcp-memory analytics --export json

  # Generate detailed report
  mcp-memory analytics --report --export html`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyticsCommand(cmd, args, deps)
		},
	}

	// Add flags
	cmd.Flags().StringP("period", "p", "30d", "Time period (7d, 30d, 90d, or custom)")
	cmd.Flags().StringP("export", "e", "", "Export format (json, csv, html, pdf)")
	cmd.Flags().BoolP("compare", "c", false, "Compare with previous period")
	cmd.Flags().BoolP("report", "r", false, "Generate detailed productivity report")
	cmd.Flags().BoolP("bottlenecks", "b", false, "Focus on bottleneck analysis")
	cmd.Flags().BoolP("trends", "t", false, "Focus on trend analysis")
	cmd.Flags().Bool("no-colors", false, "Disable colored output")
	cmd.Flags().Bool("compact", false, "Use compact display format")

	return cmd
}

// runAnalyticsCommand executes the analytics command
func runAnalyticsCommand(cmd *cobra.Command, args []string, deps CommandDeps) error {
	ctx := cmd.Context()

	// Get repository
	repository := ""
	if deps.Config != nil && deps.Config.CLI.DefaultRepository != "" {
		repository = deps.Config.CLI.DefaultRepository
	}
	if len(args) > 0 {
		repository = args[0]
	}

	if repository == "" {
		return errors.New("no repository specified. Use --repository flag or run from a git repository")
	}

	// Parse period
	periodStr, _ := cmd.Flags().GetString("period")
	period, err := parsePeriod(periodStr)
	if err != nil {
		return fmt.Errorf("invalid period '%s': %w", periodStr, err)
	}

	// Get flags
	exportFormat, _ := cmd.Flags().GetString("export")
	shouldCompare, _ := cmd.Flags().GetBool("compare")
	generateReport, _ := cmd.Flags().GetBool("report")
	focusBottlenecks, _ := cmd.Flags().GetBool("bottlenecks")
	focusTrends, _ := cmd.Flags().GetBool("trends")
	noColors, _ := cmd.Flags().GetBool("no-colors")
	compact, _ := cmd.Flags().GetBool("compact")

	// Configure visualizer
	visualizerConfig := getVisualizerConfig(noColors, compact)

	// Get analytics service
	analyticsService := deps.AnalyticsService
	if analyticsService == nil {
		return errors.New("analytics service not available")
	}

	// Generate analytics
	if generateReport {
		return runReportGeneration(ctx, analyticsService, repository, period, exportFormat)
	}

	if focusBottlenecks {
		return runBottleneckAnalysis(ctx, analyticsService, repository, period, visualizerConfig)
	}

	if focusTrends {
		return runTrendAnalysis(ctx, analyticsService, repository, period, visualizerConfig)
	}

	if shouldCompare {
		return runComparisonAnalysis(ctx, analyticsService, repository, period, visualizerConfig)
	}

	// Default: full analytics dashboard
	return runFullAnalytics(ctx, analyticsService, repository, period, exportFormat, visualizerConfig)
}

// runFullAnalytics runs the complete analytics dashboard
func runFullAnalytics(
	ctx context.Context,
	analyticsService services.AnalyticsService,
	repository string,
	period entities.TimePeriod,
	exportFormat string,
	visualizerConfig *VisualizerConfig,
) error {
	// Show loading message
	fmt.Printf("🔍 Analyzing workflow data for %s...\n\n", repository)

	// Get workflow metrics
	metrics, err := analyticsService.GetWorkflowMetrics(ctx, repository, period)
	if err != nil {
		return fmt.Errorf("failed to get workflow metrics: %w", err)
	}

	// Create visualizer
	visualizer := createVisualizer(visualizerConfig)

	// Generate and display visualization
	if sv, ok := visualizer.(*simpleVisualizer); ok {
		viz, err := sv.GenerateVisualization(metrics, entities.VisFormatTerminal)
		if err != nil {
			return fmt.Errorf("failed to generate visualization: %w", err)
		}
		fmt.Print(string(viz))
	}

	// Export if requested
	if exportFormat != "" {
		format := entities.ExportFormat(exportFormat)
		filename, err := analyticsService.ExportAnalytics(ctx, repository, period, format)
		if err != nil {
			fmt.Printf("\n⚠️  Export failed: %v\n", err)
		} else {
			fmt.Printf("\n📄 Analytics exported to: %s\n", filename)
		}
	}

	// Show summary and suggestions
	showAnalyticsSummary(metrics)

	return nil
}

// runReportGeneration generates a detailed productivity report
func runReportGeneration(
	ctx context.Context,
	analyticsService services.AnalyticsService,
	repository string,
	period entities.TimePeriod,
	exportFormat string,
) error {
	fmt.Printf("📊 Generating productivity report for %s...\n\n", repository)

	// Get productivity report
	report, err := analyticsService.GetProductivityReport(ctx, repository, period)
	if err != nil {
		return fmt.Errorf("failed to generate productivity report: %w", err)
	}

	// Display report summary
	fmt.Printf("📈 PRODUCTIVITY REPORT\n")
	fmt.Printf("═══════════════════════\n\n")
	fmt.Printf("Repository: %s\n", report.Repository)
	fmt.Printf("Period: %s to %s\n",
		report.Period.Start.Format("Jan 2"),
		report.Period.End.Format("Jan 2, 2006"))
	fmt.Printf("Overall Score: %.0f/100\n\n", report.OverallScore)

	// Show insights
	if len(report.Insights) > 0 {
		fmt.Printf("💡 KEY INSIGHTS:\n")
		for i, insight := range report.Insights {
			if i >= 5 { // Limit to top 5
				break
			}
			impactIcon := getImpactIcon(insight.Impact)
			fmt.Printf("  %s %s\n", impactIcon, insight.Title)
			fmt.Printf("     %s\n", insight.Description)
		}
		fmt.Printf("\n")
	}

	// Show recommendations
	if len(report.Recommendations) > 0 {
		fmt.Printf("🎯 RECOMMENDATIONS:\n")
		for i, rec := range report.Recommendations {
			if i >= 3 { // Limit to top 3
				break
			}
			priorityIcon := getPriorityIcon(rec.Priority)
			fmt.Printf("  %s %s\n", priorityIcon, rec.Title)
			fmt.Printf("     %s\n", rec.Description)
			if len(rec.Actions) > 0 {
				fmt.Printf("     Actions: %s\n", strings.Join(rec.Actions, ", "))
			}
		}
		fmt.Printf("\n")
	}

	// Export report if requested
	if exportFormat != "" {
		filename, err := exportProductivityReport(ctx, analyticsService, report, exportFormat)
		if err != nil {
			fmt.Printf("\n⚠️  Export failed: %v\n", err)
		} else {
			fmt.Printf("\n📄 Report exported to: %s\n", filename)
		}
	}

	return nil
}

// runBottleneckAnalysis focuses on bottleneck detection and analysis
func runBottleneckAnalysis(
	ctx context.Context,
	analyticsService services.AnalyticsService,
	repository string,
	period entities.TimePeriod,
	visualizerConfig *VisualizerConfig,
) error {
	fmt.Printf("🚨 Analyzing workflow bottlenecks for %s...\n\n", repository)

	// Get bottlenecks
	bottlenecks, err := analyticsService.DetectBottlenecks(ctx, repository, period)
	if err != nil {
		return fmt.Errorf("failed to detect bottlenecks: %w", err)
	}

	// Create visualizer and render bottlenecks
	visualizer := createVisualizer(visualizerConfig)
	if sv, ok := visualizer.(*simpleVisualizer); ok {
		output := sv.RenderBottlenecks(bottlenecks)
		fmt.Print(output)
	}

	// Show bottleneck summary
	if len(bottlenecks) > 0 {
		fmt.Printf("\n📊 BOTTLENECK SUMMARY:\n")
		totalImpact := 0.0
		criticalCount := 0

		for _, bottleneck := range bottlenecks {
			totalImpact += bottleneck.Impact
			if bottleneck.Severity == entities.BottleneckSeverityCritical ||
				bottleneck.Severity == entities.BottleneckSeverityHigh {
				criticalCount++
			}
		}

		fmt.Printf("• Total bottlenecks found: %d\n", len(bottlenecks))
		fmt.Printf("• High/Critical severity: %d\n", criticalCount)
		fmt.Printf("• Total impact: %.1f hours lost\n", totalImpact)
		fmt.Printf("• Average impact: %.1f hours per bottleneck\n", totalImpact/float64(len(bottlenecks)))
	} else {
		fmt.Printf("✨ Great news! No significant bottlenecks detected.\n")
		fmt.Printf("Your workflow appears to be running smoothly.\n")
	}

	return nil
}

// runTrendAnalysis focuses on trend analysis
func runTrendAnalysis(
	ctx context.Context,
	analyticsService services.AnalyticsService,
	repository string,
	period entities.TimePeriod,
	visualizerConfig *VisualizerConfig,
) error {
	fmt.Printf("📈 Analyzing trends for %s...\n\n", repository)

	// Get workflow metrics for trends
	metrics, err := analyticsService.GetWorkflowMetrics(ctx, repository, period)
	if err != nil {
		return fmt.Errorf("failed to get workflow metrics: %w", err)
	}

	// Create visualizer and render trends
	visualizer := createVisualizer(visualizerConfig)

	// Show velocity trends
	if sv, ok := visualizer.(*simpleVisualizer); ok {
		fmt.Print(sv.RenderVelocityChart(metrics.Velocity))
		fmt.Print("\n")
	}

	// Show overall trends
	fmt.Printf("📈 TREND ANALYSIS\n")
	fmt.Printf("═════════════════\n\n")

	// Render detailed trend visualization
	if sv, ok := visualizer.(*simpleVisualizer); ok {
		trendViz := sv.RenderTrendAnalysis(metrics.Trends)
		fmt.Print(trendViz)
	}

	// Trend analysis summary
	fmt.Printf("\n📊 TREND SUMMARY:\n")
	trends := []struct {
		name  string
		trend entities.Trend
	}{
		{"Productivity", metrics.Trends.ProductivityTrend},
		{"Velocity", metrics.Trends.VelocityTrend},
		{"Quality", metrics.Trends.QualityTrend},
		{"Efficiency", metrics.Trends.EfficiencyTrend},
	}

	improvingCount := 0
	decliningCount := 0

	for _, t := range trends {
		// Check if trend is significant (placeholder implementation)
		if t.trend.Confidence > 0.7 {
			if t.trend.Direction == entities.TrendDirectionUp {
				improvingCount++
			} else if t.trend.Direction == entities.TrendDirectionDown {
				decliningCount++
			}
		}
	}

	fmt.Printf("• Metrics improving: %d\n", improvingCount)
	fmt.Printf("• Metrics declining: %d\n", decliningCount)
	fmt.Printf("• Stable metrics: %d\n", len(trends)-improvingCount-decliningCount)

	// Show predictions if available
	if len(metrics.Trends.Predictions) > 0 {
		fmt.Printf("\n🔮 PREDICTIONS:\n")
		for _, prediction := range metrics.Trends.Predictions {
			if prediction.Confidence > 0.6 {
				fmt.Printf("• %s: %.1f (%.0f%% confidence)\n",
					strings.Title(string(prediction.Metric)),
					prediction.Value,
					prediction.Confidence*100)
			}
		}
	}

	return nil
}

// runComparisonAnalysis compares current period with previous period
func runComparisonAnalysis(
	ctx context.Context,
	analyticsService services.AnalyticsService,
	repository string,
	period entities.TimePeriod,
	visualizerConfig *VisualizerConfig,
) error {
	fmt.Printf("📊 Comparing periods for %s...\n\n", repository)

	// Calculate previous period
	duration := period.End.Sub(period.Start)
	previousPeriod := entities.TimePeriod{
		Start: period.Start.Add(-duration),
		End:   period.Start,
	}

	// Get comparison
	comparison, err := analyticsService.ComparePeriods(ctx, repository, previousPeriod, period)
	if err != nil {
		return fmt.Errorf("failed to compare periods: %w", err)
	}

	// Display comparison
	fmt.Printf("📈 PERIOD COMPARISON\n")
	fmt.Printf("═══════════════════\n\n")
	fmt.Printf("Previous: %s to %s\n",
		comparison.PeriodA.Start.Format("Jan 2"),
		comparison.PeriodA.End.Format("Jan 2"))
	fmt.Printf("Current:  %s to %s\n\n",
		comparison.PeriodB.Start.Format("Jan 2"),
		comparison.PeriodB.End.Format("Jan 2"))

	// Show key differences
	fmt.Printf("📊 KEY CHANGES:\n")
	fmt.Printf("• Productivity: %+.1f points\n", comparison.ProductivityDiff)
	fmt.Printf("• Velocity: %+.1f tasks/week\n", comparison.VelocityDiff)
	fmt.Printf("• Quality: %+.1f%%\n", comparison.QualityDiff*100)
	fmt.Printf("• Completion: %+.1f%%\n", comparison.CompletionDiff*100)
	fmt.Printf("\n")

	// Show improvements
	if len(comparison.Improvements) > 0 {
		fmt.Printf("✅ IMPROVEMENTS:\n")
		for _, improvement := range comparison.Improvements {
			fmt.Printf("• %s\n", improvement)
		}
		fmt.Printf("\n")
	}

	// Show regressions
	if len(comparison.Regressions) > 0 {
		fmt.Printf("⚠️  REGRESSIONS:\n")
		for _, regression := range comparison.Regressions {
			fmt.Printf("• %s\n", regression)
		}
		fmt.Printf("\n")
	}

	// Overall summary
	fmt.Printf("📝 SUMMARY: %s\n", comparison.Summary)

	return nil
}

// Helper functions

// parsePeriod parses a period string into a TimePeriod
func parsePeriod(periodStr string) (entities.TimePeriod, error) {
	now := time.Now()

	switch periodStr {
	case "7d":
		return entities.TimePeriod{
			Start: now.AddDate(0, 0, -7),
			End:   now,
		}, nil
	case "30d":
		return entities.TimePeriod{
			Start: now.AddDate(0, 0, -30),
			End:   now,
		}, nil
	case "90d":
		return entities.TimePeriod{
			Start: now.AddDate(0, 0, -90),
			End:   now,
		}, nil
	case "1y":
		return entities.TimePeriod{
			Start: now.AddDate(-1, 0, 0),
			End:   now,
		}, nil
	default:
		// Try to parse custom format: "YYYY-MM-DD:YYYY-MM-DD"
		if strings.Contains(periodStr, ":") {
			parts := strings.Split(periodStr, ":")
			if len(parts) != 2 {
				return entities.TimePeriod{}, errors.New("invalid period format, use YYYY-MM-DD:YYYY-MM-DD")
			}

			start, err := time.Parse("2006-01-02", parts[0])
			if err != nil {
				return entities.TimePeriod{}, fmt.Errorf("invalid start date: %w", err)
			}

			end, err := time.Parse("2006-01-02", parts[1])
			if err != nil {
				return entities.TimePeriod{}, fmt.Errorf("invalid end date: %w", err)
			}

			return entities.TimePeriod{Start: start, End: end}, nil
		}

		// Try to parse as number of days
		if strings.HasSuffix(periodStr, "d") {
			daysStr := strings.TrimSuffix(periodStr, "d")
			days, err := strconv.Atoi(daysStr)
			if err != nil {
				return entities.TimePeriod{}, fmt.Errorf("invalid number of days: %w", err)
			}

			return entities.TimePeriod{
				Start: now.AddDate(0, 0, -days),
				End:   now,
			}, nil
		}

		return entities.TimePeriod{}, fmt.Errorf("unsupported period format: %s", periodStr)
	}
}

// VisualizerConfig holds configuration for the visualizer
type VisualizerConfig struct {
	NoColors bool
	Compact  bool
	Width    int
	Height   int
}

// getVisualizerConfig creates visualizer configuration from flags
func getVisualizerConfig(noColors, compact bool) *VisualizerConfig {
	return &VisualizerConfig{
		NoColors: noColors,
		Compact:  compact,
		Width:    80, // Default terminal width
		Height:   24, // Default terminal height
	}
}

// createVisualizer creates a visualizer based on configuration
func createVisualizer(config *VisualizerConfig) interface{} {
	// For now, return a simple interface that has the methods we need
	// TODO: This should create the actual visualizer when the ports.Visualizer interface is finalized
	return &simpleVisualizer{config: config}
}

// simpleVisualizer is a placeholder visualizer implementation
type simpleVisualizer struct {
	config *VisualizerConfig
}

func (v *simpleVisualizer) RenderVelocityChart(metrics entities.VelocityMetrics) string {
	return fmt.Sprintf("📊 Velocity Chart: Current=%.1f, Trend=%s\n",
		metrics.CurrentVelocity, metrics.TrendDirection)
}

func (v *simpleVisualizer) RenderBottlenecks(bottlenecks []*entities.Bottleneck) string {
	if len(bottlenecks) == 0 {
		return "✅ No bottlenecks detected\n"
	}

	result := "🚨 BOTTLENECKS DETECTED:\n"
	for i, b := range bottlenecks {
		if i >= 5 { // Limit to top 5
			break
		}
		result += fmt.Sprintf("  • %s (Impact: %.1fh, Severity: %s)\n",
			b.Description, b.Impact, b.Severity)
	}
	return result
}

func (v *simpleVisualizer) RenderTrendAnalysis(trends entities.TrendAnalysis) string {
	result := ""

	// Render individual trend charts
	trendItems := []struct {
		name  string
		trend entities.Trend
		icon  string
	}{
		{"Productivity", trends.ProductivityTrend, "📊"},
		{"Velocity", trends.VelocityTrend, "🚀"},
		{"Quality", trends.QualityTrend, "✨"},
		{"Efficiency", trends.EfficiencyTrend, "⚡"},
	}

	for _, item := range trendItems {
		result += v.renderSingleTrend(item.name, item.trend, item.icon)
		result += "\n"
	}

	// Show predictions if available
	if len(trends.Predictions) > 0 {
		result += v.renderPredictions(trends.Predictions)
		result += "\n"
	}

	// Show seasonality analysis
	if trends.Seasonality.HasSeasonality {
		result += v.renderSeasonality(trends.Seasonality)
		result += "\n"
	}

	return result
}

func (v *simpleVisualizer) renderSingleTrend(name string, trend entities.Trend, icon string) string {
	var result strings.Builder

	// Header with trend direction
	directionIcon := v.getTrendDirectionIcon(trend.Direction)
	confidenceIcon := v.getConfidenceIcon(trend.Confidence)

	result.WriteString(fmt.Sprintf("%s %s TREND %s %s\n", icon, strings.ToUpper(name), directionIcon, confidenceIcon))

	// Trend details
	result.WriteString(fmt.Sprintf("  Direction: %s (%.1f%% confidence)\n",
		v.getTrendDirectionText(trend.Direction), trend.Confidence*100))
	result.WriteString(fmt.Sprintf("  Strength: %.3f\n", trend.Strength))
	result.WriteString(fmt.Sprintf("  Change Rate: %+.1f%%\n", trend.ChangeRate))
	result.WriteString(fmt.Sprintf("  Start Value: %.1f\n", trend.StartValue))
	result.WriteString(fmt.Sprintf("  End Value: %.1f\n", trend.EndValue))

	// Render mini chart using ASCII if trend line data available
	if len(trend.TrendLine) > 1 {
		result.WriteString("  Chart: ")
		result.WriteString(v.renderTrendLineChart(trend.TrendLine))
		result.WriteString("\n")
	}

	// Description if available
	if trend.Description != "" {
		result.WriteString(fmt.Sprintf("  Summary: %s\n", trend.Description))
	}

	return result.String()
}

func (v *simpleVisualizer) renderTrendLineChart(trendLine []entities.TrendPoint) string {
	if len(trendLine) == 0 {
		return "No data"
	}

	// Find min/max for normalization
	minVal, maxVal := trendLine[0].Value, trendLine[0].Value
	for _, point := range trendLine {
		if point.Value < minVal {
			minVal = point.Value
		}
		if point.Value > maxVal {
			maxVal = point.Value
		}
	}

	// Avoid division by zero
	if maxVal == minVal {
		return strings.Repeat("─", len(trendLine))
	}

	// Create ASCII chart
	var chart strings.Builder
	chars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

	for _, point := range trendLine {
		// Normalize to 0-7 range for character selection
		normalized := (point.Value - minVal) / (maxVal - minVal)
		charIndex := int(normalized * 7)
		if charIndex > 7 {
			charIndex = 7
		}
		chart.WriteString(chars[charIndex])
	}

	return chart.String()
}

func (v *simpleVisualizer) renderPredictions(predictions []entities.Prediction) string {
	result := "🔮 PREDICTIONS:\n"

	if len(predictions) == 0 {
		return result + "  No predictions available\n"
	}

	for i, prediction := range predictions {
		if i >= 5 { // Limit to top 5 predictions
			break
		}

		confidenceIcon := v.getConfidenceIcon(prediction.Confidence)
		result += fmt.Sprintf("  %s %s: %.1f (%.0f%% confidence)\n",
			confidenceIcon,
			strings.Title(string(prediction.Metric)),
			prediction.Value,
			prediction.Confidence*100)

		if len(prediction.Range) == 2 {
			result += fmt.Sprintf("    Range: %.1f - %.1f\n", prediction.Range[0], prediction.Range[1])
		}

		if prediction.Method != "" {
			result += fmt.Sprintf("    Method: %s\n", prediction.Method)
		}
	}

	return result
}

func (v *simpleVisualizer) renderSeasonality(seasonality entities.Seasonality) string {
	result := "📅 SEASONALITY ANALYSIS:\n"

	if !seasonality.HasSeasonality {
		return result + "  No significant seasonal patterns detected\n"
	}

	// Show seasonal patterns
	if len(seasonality.Patterns) > 0 {
		result += "  Detected Patterns:\n"
		for _, pattern := range seasonality.Patterns {
			result += fmt.Sprintf("    • %s: %s (%.1f%% confidence)\n",
				strings.Title(pattern.Type),
				pattern.Description,
				pattern.Confidence*100)
		}
	}

	// Show weekly pattern if available
	if len(seasonality.WeeklyPattern) > 0 {
		result += "\n  Weekly Pattern:\n"
		days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
		for _, day := range days {
			if multiplier, exists := seasonality.WeeklyPattern[strings.ToLower(day)]; exists {
				intensity := v.getSeasonalIntensity(multiplier)
				result += fmt.Sprintf("    %s: %.2fx %s\n", day, multiplier, intensity)
			}
		}
	}

	// Show hourly pattern if available
	if len(seasonality.HourlyPattern) > 0 {
		result += "\n  Peak Hours: "
		var peakHours []string
		for hour, multiplier := range seasonality.HourlyPattern {
			if multiplier > 1.2 { // 20% above average
				peakHours = append(peakHours, fmt.Sprintf("%02d:00", hour))
			}
		}
		if len(peakHours) > 0 {
			result += strings.Join(peakHours, ", ") + "\n"
		} else {
			result += "No significant peak hours detected\n"
		}
	}

	return result
}

func (v *simpleVisualizer) getTrendDirectionIcon(direction entities.TrendDirection) string {
	switch direction {
	case entities.TrendDirectionUp:
		return "📈"
	case entities.TrendDirectionDown:
		return "📉"
	case entities.TrendDirectionStable:
		return "➡️"
	default:
		return "❓"
	}
}

func (v *simpleVisualizer) getTrendDirectionText(direction entities.TrendDirection) string {
	switch direction {
	case entities.TrendDirectionUp:
		return "Improving"
	case entities.TrendDirectionDown:
		return "Declining"
	case entities.TrendDirectionStable:
		return "Stable"
	default:
		return "Unknown"
	}
}

func (v *simpleVisualizer) getConfidenceIcon(confidence float64) string {
	if confidence >= 0.9 {
		return "🟢" // High confidence
	} else if confidence >= 0.7 {
		return "🟡" // Medium confidence
	} else if confidence >= 0.5 {
		return "🟠" // Low confidence
	} else {
		return "🔴" // Very low confidence
	}
}

func (v *simpleVisualizer) getSeasonalIntensity(multiplier float64) string {
	if multiplier >= 1.5 {
		return "🔥 Very High"
	} else if multiplier >= 1.2 {
		return "⬆️ High"
	} else if multiplier >= 0.8 {
		return "➡️ Normal"
	} else if multiplier >= 0.5 {
		return "⬇️ Low"
	} else {
		return "❄️ Very Low"
	}
}

func (v *simpleVisualizer) getCorrelationStrength(correlation float64) string {
	abs := correlation
	if abs < 0 {
		abs = -abs
	}

	if abs >= 0.8 {
		return "strong"
	} else if abs >= 0.6 {
		return "moderate"
	} else if abs >= 0.3 {
		return "weak"
	} else {
		return "very weak"
	}
}

func (v *simpleVisualizer) GenerateVisualization(metrics *entities.WorkflowMetrics, format entities.VisFormat) ([]byte, error) {
	output := fmt.Sprintf(`📊 WORKFLOW ANALYTICS DASHBOARD
════════════════════════════════

📈 PRODUCTIVITY METRICS:
• Overall Score: %.1f/100
• Tasks per Day: %.1f
• Focus Time: %v

🚀 VELOCITY METRICS:
• Current Velocity: %.1f tasks/week
• Trend: %s

✅ COMPLETION METRICS:
• Total Tasks: %d
• Completed: %d
• Completion Rate: %.1f%%

⏱️  CYCLE TIME:
• Average: %v
• Median: %v

`,
		metrics.Productivity.Score,
		metrics.Productivity.TasksPerDay,
		metrics.Productivity.FocusTime,
		metrics.Velocity.CurrentVelocity,
		metrics.Velocity.TrendDirection,
		metrics.Completion.TotalTasks,
		metrics.Completion.Completed,
		metrics.Completion.CompletionRate*100,
		metrics.CycleTime.AverageCycleTime,
		metrics.CycleTime.MedianCycleTime,
	)

	if len(metrics.Bottlenecks) > 0 {
		output += "\n🚨 BOTTLENECKS:\n"
		for i, bottleneck := range metrics.Bottlenecks {
			if i >= 3 { // Limit to top 3
				break
			}
			output += fmt.Sprintf("• %s (%.1fh impact)\n", bottleneck.Description, bottleneck.Impact)
		}
	}

	return []byte(output), nil
}

// showAnalyticsSummary shows a summary of the analytics results
func showAnalyticsSummary(metrics *entities.WorkflowMetrics) {
	// Calculate overall score (placeholder implementation)
	overallScore := metrics.Productivity.Score

	fmt.Printf("\n📊 ANALYTICS SUMMARY\n")
	fmt.Printf("═══════════════════\n")
	fmt.Printf("Overall Score: %.0f/100", overallScore)

	if overallScore >= 80 {
		fmt.Printf(" 🎉 Excellent!\n")
		fmt.Printf("Your workflow is performing exceptionally well.\n")
	} else if overallScore >= 60 {
		fmt.Printf(" 👍 Good\n")
		fmt.Printf("Your workflow is performing well with room for improvement.\n")
	} else if overallScore >= 40 {
		fmt.Printf(" 📈 Needs Improvement\n")
		fmt.Printf("There are several areas where your workflow can be optimized.\n")
	} else {
		fmt.Printf(" 🚨 Needs Attention\n")
		fmt.Printf("Your workflow has significant bottlenecks that need addressing.\n")
	}

	// Quick tips based on metrics
	fmt.Printf("\n💡 QUICK TIPS:\n")

	if metrics.Productivity.Score < 60 {
		fmt.Printf("• Focus on improving task completion rate\n")
	}

	if entities.TrendDirection(metrics.Velocity.TrendDirection) == entities.TrendDirectionDown {
		fmt.Printf("• Consider addressing velocity decline\n")
	}

	if len(metrics.Bottlenecks) > 0 {
		fmt.Printf("• Address workflow bottlenecks for immediate impact\n")
	}

	if metrics.Productivity.ContextSwitches > 10 {
		fmt.Printf("• Reduce context switching for better focus\n")
	}

	if metrics.Completion.OnTimeRate < 0.7 {
		fmt.Printf("• Improve task estimation and time management\n")
	}
}

// getImpactIcon returns an icon based on impact level
func getImpactIcon(impact float64) string {
	if impact >= 0.8 {
		return "🔥"
	} else if impact >= 0.6 {
		return "⚡"
	} else if impact >= 0.4 {
		return "💡"
	} else {
		return "ℹ️"
	}
}

// exportProductivityReport exports a productivity report to the specified format
func exportProductivityReport(
	ctx context.Context,
	analyticsService services.AnalyticsService,
	report *entities.ProductivityReport,
	exportFormat string,
) (string, error) {
	// Convert export format string to entities.ExportFormat
	format := entities.ExportFormat(exportFormat)

	// Use the existing ExportAnalytics method with the report's metrics
	filename, err := analyticsService.ExportAnalytics(
		ctx,
		report.Repository,
		report.Period,
		format,
	)

	if err != nil {
		return "", fmt.Errorf("failed to export productivity report: %w", err)
	}

	return filename, nil
}

// getPriorityIcon returns an icon based on recommendation priority
func getPriorityIcon(priority entities.RecommendationPriority) string {
	switch priority {
	case entities.RecommendationPriorityCritical:
		return "🚨"
	case entities.RecommendationPriorityHigh:
		return "⚠️"
	case entities.RecommendationPriorityMedium:
		return "🔶"
	default:
		return "🔵"
	}
}
