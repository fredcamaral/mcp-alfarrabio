package visualization

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// Visualizer interface for generating different types of visualizations
type Visualizer interface {
	GenerateVisualization(metrics *entities.WorkflowMetrics, format entities.VisFormat) ([]byte, error)
	RenderProductivityChart(metrics entities.ProductivityMetrics) string
	RenderVelocityChart(metrics entities.VelocityMetrics) string
	RenderCompletionChart(metrics entities.CompletionMetrics) string
	RenderBottlenecks(bottlenecks []*entities.Bottleneck) string
	RenderTrends(trends entities.TrendAnalysis) string
	RenderCycleTimeChart(metrics entities.CycleTimeMetrics) string
}

// TerminalVisualizerConfig holds configuration for terminal visualization
type TerminalVisualizerConfig struct {
	Width       int  `json:"width"`
	Height      int  `json:"height"`
	Colors      bool `json:"colors"`
	Unicode     bool `json:"unicode"`
	Compact     bool `json:"compact"`
	MaxBarWidth int  `json:"max_bar_width"`
}

// DefaultTerminalVisualizerConfig returns default configuration
func DefaultTerminalVisualizerConfig() *TerminalVisualizerConfig {
	return &TerminalVisualizerConfig{
		Width:       80,
		Height:      24,
		Colors:      true,
		Unicode:     true,
		Compact:     false,
		MaxBarWidth: 40,
	}
}

// terminalVisualizer implements terminal-based visualization
type terminalVisualizer struct {
	config *TerminalVisualizerConfig
}

// NewTerminalVisualizer creates a new terminal visualizer
func NewTerminalVisualizer(config *TerminalVisualizerConfig) Visualizer {
	if config == nil {
		config = DefaultTerminalVisualizerConfig()
	}

	return &terminalVisualizer{
		config: config,
	}
}

// GenerateVisualization generates a complete visualization based on format
func (tv *terminalVisualizer) GenerateVisualization(
	metrics *entities.WorkflowMetrics,
	format entities.VisFormat,
) ([]byte, error) {
	if format != entities.VisFormatTerminal {
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	var output strings.Builder

	// Header
	output.WriteString(tv.renderHeader(metrics))
	output.WriteString("\n")

	// Productivity section
	output.WriteString(tv.RenderProductivityChart(metrics.Productivity))
	output.WriteString("\n")

	// Velocity section
	output.WriteString(tv.RenderVelocityChart(metrics.Velocity))
	output.WriteString("\n")

	// Completion section
	output.WriteString(tv.RenderCompletionChart(metrics.Completion))
	output.WriteString("\n")

	// Cycle time section
	output.WriteString(tv.RenderCycleTimeChart(metrics.CycleTime))
	output.WriteString("\n")

	// Bottlenecks section
	if len(metrics.Bottlenecks) > 0 {
		// Convert []Bottleneck to []*Bottleneck
		bottleneckPtrs := make([]*entities.Bottleneck, len(metrics.Bottlenecks))
		for i := range metrics.Bottlenecks {
			bottleneckPtrs[i] = &metrics.Bottlenecks[i]
		}
		output.WriteString(tv.RenderBottlenecks(bottleneckPtrs))
		output.WriteString("\n")
	}

	// Trends section
	output.WriteString(tv.RenderTrends(metrics.Trends))

	return []byte(output.String()), nil
}

// RenderProductivityChart renders productivity metrics as ASCII chart
func (tv *terminalVisualizer) RenderProductivityChart(metrics entities.ProductivityMetrics) string {
	var builder strings.Builder

	// Section header
	builder.WriteString(tv.colorize("📊 PRODUCTIVITY METRICS", "bold"))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", tv.config.Width))
	builder.WriteString("\n\n")

	// Overall score
	builder.WriteString("Overall Score: ")
	builder.WriteString(tv.renderProgressBar(metrics.Score, 100, 30))
	builder.WriteString(fmt.Sprintf(" %.1f/100", metrics.Score))
	if metrics.Score >= 80 {
		builder.WriteString(tv.colorize(" 🔥 Excellent!", "green"))
	} else if metrics.Score >= 60 {
		builder.WriteString(tv.colorize(" ✨ Good", "yellow"))
	} else {
		builder.WriteString(tv.colorize(" 📈 Needs improvement", "red"))
	}
	builder.WriteString("\n\n")

	// Key metrics
	builder.WriteString(fmt.Sprintf("📈 Tasks per Day:     %.1f\n", metrics.TasksPerDay))
	builder.WriteString(fmt.Sprintf("⏱️  Average Focus:     %s\n", tv.formatDuration(metrics.FocusTime)))
	builder.WriteString(fmt.Sprintf("🧠 Deep Work Ratio:   %.0f%%\n", metrics.DeepWorkRatio*100))
	builder.WriteString(fmt.Sprintf("🔄 Context Switches:  %d\n", metrics.ContextSwitches))
	builder.WriteString("\n")

	// Peak hours
	if len(metrics.PeakHours) > 0 {
		builder.WriteString("🕐 Peak Hours: ")
		for i, hour := range metrics.PeakHours {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(fmt.Sprintf("%02d:00", hour))
		}
		builder.WriteString("\n\n")
	}

	// Priority completion rates
	if len(metrics.ByPriority) > 0 {
		builder.WriteString("Priority Completion Rates:\n")
		priorities := []string{"high", "medium", "low"}

		for _, priority := range priorities {
			if rate, exists := metrics.ByPriority[priority]; exists {
				icon := tv.getPriorityIcon(priority)
				builder.WriteString(fmt.Sprintf("  %s %-6s ", icon, strings.Title(priority)))
				builder.WriteString(tv.renderMiniBar(rate, 20))
				builder.WriteString(fmt.Sprintf(" %.0f%%\n", rate*100))
			}
		}
		builder.WriteString("\n")
	}

	// Type completion rates (top 5)
	if len(metrics.ByType) > 0 {
		builder.WriteString("Task Type Performance (Top 5):\n")

		// Sort by completion rate
		type typeRate struct {
			name string
			rate float64
		}

		var typeRates []typeRate
		for taskType, rate := range metrics.ByType {
			typeRates = append(typeRates, typeRate{name: taskType, rate: rate})
		}

		sort.Slice(typeRates, func(i, j int) bool {
			return typeRates[i].rate > typeRates[j].rate
		})

		// Show top 5
		maxShow := 5
		if len(typeRates) < maxShow {
			maxShow = len(typeRates)
		}

		for i := 0; i < maxShow; i++ {
			tr := typeRates[i]
			builder.WriteString(fmt.Sprintf("  %-12s ", tv.truncateString(tr.name, 12)))
			builder.WriteString(tv.renderMiniBar(tr.rate, 15))
			builder.WriteString(fmt.Sprintf(" %.0f%%\n", tr.rate*100))
		}
	}

	return builder.String()
}

// RenderVelocityChart renders velocity metrics and trends
func (tv *terminalVisualizer) RenderVelocityChart(metrics entities.VelocityMetrics) string {
	var builder strings.Builder

	// Section header
	builder.WriteString(tv.colorize("📏 VELOCITY METRICS", "bold"))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", tv.config.Width))
	builder.WriteString("\n\n")

	// Current velocity with trend
	trendIcon := tv.getTrendIcon(entities.TrendDirection(metrics.TrendDirection))
	trendColor := tv.getTrendColor(entities.TrendDirection(metrics.TrendDirection))

	builder.WriteString(fmt.Sprintf("Current Velocity: %.1f tasks/week ", metrics.CurrentVelocity))
	builder.WriteString(tv.colorize(fmt.Sprintf("%s %.1f%%", trendIcon, metrics.TrendPercentage), trendColor))
	builder.WriteString("\n")

	builder.WriteString(fmt.Sprintf("Consistency:      %.0f%%", metrics.Consistency*100))
	if metrics.Consistency > 0.8 {
		builder.WriteString(tv.colorize(" 🎯 Very consistent", "green"))
	} else if metrics.Consistency > 0.6 {
		builder.WriteString(tv.colorize(" ⚡ Somewhat variable", "yellow"))
	} else {
		builder.WriteString(tv.colorize(" 🌊 Highly variable", "red"))
	}
	builder.WriteString("\n\n")

	// Weekly velocity chart
	if len(metrics.ByWeek) > 0 {
		builder.WriteString("Weekly Velocity Trend:\n")

		// Find max velocity for scaling
		maxVelocity := 0.0
		for _, week := range metrics.ByWeek {
			if week.Velocity > maxVelocity {
				maxVelocity = week.Velocity
			}
		}

		// Show last 8 weeks if more available
		weekData := metrics.ByWeek
		if len(weekData) > 8 {
			weekData = weekData[len(weekData)-8:]
		}

		for _, week := range weekData {
			barLength := tv.calculateBarLength(week.Velocity, maxVelocity, 25)
			builder.WriteString(fmt.Sprintf("W%02d │", week.Number))

			// Render bar
			if week.Velocity > 0 {
				bar := strings.Repeat("█", barLength)
				if barLength < 25 {
					bar += strings.Repeat("░", 25-barLength)
				}
				builder.WriteString(bar)
			} else {
				builder.WriteString(strings.Repeat("░", 25))
			}

			builder.WriteString(fmt.Sprintf("│ %.1f\n", week.Velocity))
		}
		builder.WriteString("\n")
	}

	// Forecast
	if metrics.Forecast.Confidence > 0.5 {
		builder.WriteString("📮 Forecast:\n")
		builder.WriteString(fmt.Sprintf("  Next Week: %.1f tasks ", metrics.Forecast.PredictedVelocity))
		builder.WriteString(fmt.Sprintf("(%.0f%% confidence)\n", metrics.Forecast.Confidence*100))

		if len(metrics.Forecast.Range) == 2 {
			builder.WriteString(fmt.Sprintf("  Range: %.1f - %.1f tasks\n",
				metrics.Forecast.Range[0], metrics.Forecast.Range[1]))
		}
	}

	return builder.String()
}

// RenderCompletionChart renders completion metrics
func (tv *terminalVisualizer) RenderCompletionChart(metrics entities.CompletionMetrics) string {
	var builder strings.Builder

	// Section header
	builder.WriteString(tv.colorize("✅ COMPLETION METRICS", "bold"))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", tv.config.Width))
	builder.WriteString("\n\n")

	// Overall completion stats
	builder.WriteString(fmt.Sprintf("Total Tasks:      %d\n", metrics.TotalTasks))
	builder.WriteString(fmt.Sprintf("Completed:        %d (%.0f%%)\n",
		metrics.Completed, metrics.CompletionRate*100))
	builder.WriteString(fmt.Sprintf("In Progress:      %d\n", metrics.InProgress))
	builder.WriteString(fmt.Sprintf("Cancelled:        %d\n", metrics.Cancelled))
	builder.WriteString("\n")

	// Completion rate visualization
	builder.WriteString("Completion Rate: ")
	builder.WriteString(tv.renderProgressBar(metrics.CompletionRate*100, 100, 25))
	builder.WriteString(fmt.Sprintf(" %.0f%%", metrics.CompletionRate*100))
	if metrics.CompletionRate >= 0.8 {
		builder.WriteString(tv.colorize(" 🎉 Excellent!", "green"))
	} else if metrics.CompletionRate >= 0.6 {
		builder.WriteString(tv.colorize(" 👍 Good", "yellow"))
	} else {
		builder.WriteString(tv.colorize(" 📋 Room for improvement", "red"))
	}
	builder.WriteString("\n\n")

	// Quality metrics
	builder.WriteString(fmt.Sprintf("Average Cycle Time: %s\n", tv.formatDuration(metrics.AverageTime)))
	builder.WriteString(fmt.Sprintf("On-Time Rate:       %.0f%%\n", metrics.OnTimeRate*100))
	builder.WriteString(fmt.Sprintf("Quality Score:      %.0f%%\n", metrics.QualityScore*100))
	builder.WriteString("\n")

	// Status distribution
	if len(metrics.ByStatus) > 0 {
		builder.WriteString("Task Status Distribution:\n")
		statusOrder := []string{"completed", "in_progress", "pending", "cancelled"}

		for _, status := range statusOrder {
			if count, exists := metrics.ByStatus[status]; exists && count > 0 {
				percentage := float64(count) / float64(metrics.TotalTasks) * 100
				icon := tv.getStatusIcon(status)
				builder.WriteString(fmt.Sprintf("  %s %-12s ", icon, strings.Title(status)))
				builder.WriteString(tv.renderMiniBar(percentage/100, 15))
				builder.WriteString(fmt.Sprintf(" %d (%.0f%%)\n", count, percentage))
			}
		}
		builder.WriteString("\n")
	}

	// Priority distribution
	if len(metrics.ByPriority) > 0 {
		builder.WriteString("Priority Distribution:\n")
		priorities := []string{"high", "medium", "low"}

		for _, priority := range priorities {
			if count, exists := metrics.ByPriority[priority]; exists && count > 0 {
				percentage := float64(count) / float64(metrics.TotalTasks) * 100
				icon := tv.getPriorityIcon(priority)
				builder.WriteString(fmt.Sprintf("  %s %-6s ", icon, strings.Title(priority)))
				builder.WriteString(tv.renderMiniBar(percentage/100, 15))
				builder.WriteString(fmt.Sprintf(" %d (%.0f%%)\n", count, percentage))
			}
		}
	}

	return builder.String()
}

// RenderCycleTimeChart renders cycle time metrics
func (tv *terminalVisualizer) RenderCycleTimeChart(metrics entities.CycleTimeMetrics) string {
	var builder strings.Builder

	// Section header
	builder.WriteString(tv.colorize("⏰ CYCLE TIME ANALYSIS", "bold"))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", tv.config.Width))
	builder.WriteString("\n\n")

	// Basic cycle time metrics
	builder.WriteString(fmt.Sprintf("Average Cycle Time: %s\n", tv.formatDuration(metrics.AverageCycleTime)))
	builder.WriteString(fmt.Sprintf("Median Cycle Time:  %s\n", tv.formatDuration(metrics.MedianCycleTime)))
	builder.WriteString(fmt.Sprintf("90th Percentile:    %s\n", tv.formatDuration(metrics.P90CycleTime)))
	builder.WriteString("\n")

	// Efficiency metrics
	if metrics.LeadTime > 0 {
		efficiency := metrics.GetEfficiencyScore()
		builder.WriteString(fmt.Sprintf("Lead Time:          %s\n", tv.formatDuration(metrics.LeadTime)))
		builder.WriteString(fmt.Sprintf("Wait Time:          %s\n", tv.formatDuration(metrics.WaitTime)))
		builder.WriteString("Efficiency:         ")
		builder.WriteString(tv.renderProgressBar(efficiency*100, 100, 20))
		builder.WriteString(fmt.Sprintf(" %.0f%%\n", efficiency*100))
		builder.WriteString("\n")
	}

	// Cycle time by type
	if len(metrics.ByType) > 0 {
		builder.WriteString("Cycle Time by Task Type:\n")

		// Sort by cycle time
		type typeCycle struct {
			name string
			time time.Duration
		}

		var typeCycles []typeCycle
		for taskType, cycleTime := range metrics.ByType {
			typeCycles = append(typeCycles, typeCycle{name: taskType, time: cycleTime})
		}

		sort.Slice(typeCycles, func(i, j int) bool {
			return typeCycles[i].time > typeCycles[j].time
		})

		// Find max for scaling
		maxTime := time.Duration(0)
		for _, tc := range typeCycles {
			if tc.time > maxTime {
				maxTime = tc.time
			}
		}

		// Show top 5
		maxShow := 5
		if len(typeCycles) < maxShow {
			maxShow = len(typeCycles)
		}

		for i := 0; i < maxShow; i++ {
			tc := typeCycles[i]
			builder.WriteString(fmt.Sprintf("  %-12s ", tv.truncateString(tc.name, 12)))

			barLength := tv.calculateBarLength(float64(tc.time), float64(maxTime), 20)
			builder.WriteString(strings.Repeat("█", barLength))
			builder.WriteString(strings.Repeat("░", 20-barLength))
			builder.WriteString(fmt.Sprintf(" %s\n", tv.formatDuration(tc.time)))
		}
		builder.WriteString("\n")
	}

	// Cycle time distribution
	if len(metrics.Distribution) > 0 {
		builder.WriteString("Cycle Time Distribution:\n")

		for _, point := range metrics.Distribution {
			if point.Count > 0 {
				builder.WriteString(fmt.Sprintf("  ≤%-8s ", tv.formatDuration(point.Duration)))

				// Create mini histogram
				maxCount := 0
				for _, p := range metrics.Distribution {
					if p.Count > maxCount {
						maxCount = p.Count
					}
				}

				barLength := tv.calculateBarLength(float64(point.Count), float64(maxCount), 15)
				builder.WriteString(strings.Repeat("▇", barLength))
				builder.WriteString(fmt.Sprintf(" %d tasks\n", point.Count))
			}
		}
	}

	return builder.String()
}

// RenderBottlenecks renders detected bottlenecks
func (tv *terminalVisualizer) RenderBottlenecks(bottlenecks []*entities.Bottleneck) string {
	var builder strings.Builder

	// Section header
	builder.WriteString(tv.colorize("🚨 WORKFLOW BOTTLENECKS", "bold"))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", tv.config.Width))
	builder.WriteString("\n\n")

	if len(bottlenecks) == 0 {
		builder.WriteString(tv.colorize("✨ No significant bottlenecks detected!", "green"))
		builder.WriteString("\n")
		return builder.String()
	}

	// Show top bottlenecks
	maxShow := 5
	if len(bottlenecks) < maxShow {
		maxShow = len(bottlenecks)
	}

	for i := 0; i < maxShow; i++ {
		bottleneck := bottlenecks[i]

		// Severity icon and color
		severityIcon := tv.getSeverityIcon(bottleneck.Severity)
		severityColor := tv.getSeverityColor(bottleneck.Severity)

		builder.WriteString(tv.colorize(fmt.Sprintf("%s %s", severityIcon, strings.ToUpper(string(bottleneck.Severity))), severityColor))
		builder.WriteString(fmt.Sprintf(" - %s\n", bottleneck.Description))

		// Impact and frequency
		builder.WriteString(fmt.Sprintf("   Impact: %.1f hours lost", bottleneck.Impact))
		if bottleneck.Frequency > 1 {
			builder.WriteString(fmt.Sprintf(" (%d occurrences)", bottleneck.Frequency))
		}
		builder.WriteString("\n")

		// Suggestions
		if len(bottleneck.Suggestions) > 0 {
			builder.WriteString("   Suggestions:\n")
			for _, suggestion := range bottleneck.Suggestions {
				builder.WriteString(fmt.Sprintf("   • %s\n", suggestion))
			}
		}

		if i < maxShow-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// RenderTrends renders trend analysis
func (tv *terminalVisualizer) RenderTrends(trends entities.TrendAnalysis) string {
	var builder strings.Builder

	// Section header
	builder.WriteString(tv.colorize("📈 TREND ANALYSIS", "bold"))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("─", tv.config.Width))
	builder.WriteString("\n\n")

	// Productivity trend
	builder.WriteString("Productivity: ")
	builder.WriteString(tv.renderTrend(trends.ProductivityTrend))
	builder.WriteString("\n")

	// Velocity trend
	builder.WriteString("Velocity:     ")
	builder.WriteString(tv.renderTrend(trends.VelocityTrend))
	builder.WriteString("\n")

	// Quality trend
	builder.WriteString("Quality:      ")
	builder.WriteString(tv.renderTrend(trends.QualityTrend))
	builder.WriteString("\n")

	// Efficiency trend
	builder.WriteString("Efficiency:   ")
	builder.WriteString(tv.renderTrend(trends.EfficiencyTrend))
	builder.WriteString("\n\n")

	// Predictions
	if len(trends.Predictions) > 0 {
		builder.WriteString("🔮 Predictions:\n")
		for _, prediction := range trends.Predictions {
			if prediction.Confidence > 0.6 {
				builder.WriteString(fmt.Sprintf("  %s: %.1f (%.0f%% confidence)\n",
					strings.Title(string(prediction.Metric)),
					prediction.Value,
					prediction.Confidence*100))
			}
		}
		builder.WriteString("\n")
	}

	// Seasonality
	if trends.Seasonality.HasSeasonality {
		builder.WriteString("📅 Seasonal Patterns Detected:\n")
		for _, pattern := range trends.Seasonality.Patterns {
			if pattern.Confidence > 0.6 {
				builder.WriteString(fmt.Sprintf("  %s: %s (%.0f%% confidence)\n",
					strings.Title(pattern.Type),
					pattern.Description,
					pattern.Confidence*100))
			}
		}
	}

	return builder.String()
}

// Helper methods for rendering components

func (tv *terminalVisualizer) renderHeader(metrics *entities.WorkflowMetrics) string {
	var builder strings.Builder

	builder.WriteString(tv.colorize("📊 WORKFLOW ANALYTICS DASHBOARD", "bold"))
	builder.WriteString("\n")
	builder.WriteString(tv.colorize("Repository: "+metrics.Repository, "cyan"))
	builder.WriteString("\n")
	builder.WriteString(tv.colorize(fmt.Sprintf("Period: %s to %s",
		metrics.Period.Start.Format("Jan 2"),
		metrics.Period.End.Format("Jan 2, 2006")), "cyan"))
	builder.WriteString("\n")
	builder.WriteString(tv.colorize(fmt.Sprintf("Overall Score: %.0f/100", metrics.GetOverallScore()*100), "cyan"))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("═", tv.config.Width))

	return builder.String()
}

func (tv *terminalVisualizer) renderProgressBar(value, max float64, width int) string {
	if max == 0 {
		return strings.Repeat("░", width)
	}

	percentage := value / max
	if percentage > 1.0 {
		percentage = 1.0
	}
	if percentage < 0.0 {
		percentage = 0.0
	}

	filled := int(percentage * float64(width))
	empty := width - filled

	var bar string
	if tv.config.Unicode {
		bar = strings.Repeat("█", filled) + strings.Repeat("░", empty)
	} else {
		bar = strings.Repeat("=", filled) + strings.Repeat("-", empty)
	}

	// Add color based on value
	if tv.config.Colors {
		if percentage >= 0.8 {
			return tv.colorize(bar, "green")
		} else if percentage >= 0.6 {
			return tv.colorize(bar, "yellow")
		} else if percentage >= 0.3 {
			return tv.colorize(bar, "orange")
		} else {
			return tv.colorize(bar, "red")
		}
	}

	return bar
}

func (tv *terminalVisualizer) renderMiniBar(value float64, width int) string {
	percentage := value
	if percentage > 1.0 {
		percentage = 1.0
	}
	if percentage < 0.0 {
		percentage = 0.0
	}

	filled := int(percentage * float64(width))
	empty := width - filled

	var bar string
	if tv.config.Unicode {
		bar = strings.Repeat("▇", filled) + strings.Repeat("░", empty)
	} else {
		bar = strings.Repeat("=", filled) + strings.Repeat(".", empty)
	}

	return bar
}

func (tv *terminalVisualizer) renderTrend(trend entities.Trend) string {
	icon := tv.getTrendIcon(trend.Direction)
	color := tv.getTrendColor(trend.Direction)

	var strength string
	if trend.Strength > 0.7 {
		strength = "Strong"
	} else if trend.Strength > 0.4 {
		strength = "Moderate"
	} else {
		strength = "Weak"
	}

	trendText := fmt.Sprintf("%s %s %s", icon, strength, trend.Direction)
	if trend.IsSignificant() {
		trendText += fmt.Sprintf(" (%.0f%% confidence)", trend.Confidence*100)
	}

	return tv.colorize(trendText, color)
}

func (tv *terminalVisualizer) calculateBarLength(value, max float64, width int) int {
	if max == 0 {
		return 0
	}

	percentage := value / max
	if percentage > 1.0 {
		percentage = 1.0
	}
	if percentage < 0.0 {
		percentage = 0.0
	}

	return int(percentage * float64(width))
}

// Icon and color helpers

func (tv *terminalVisualizer) getPriorityIcon(priority string) string {
	if !tv.config.Unicode {
		return ">"
	}

	switch priority {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

func (tv *terminalVisualizer) getStatusIcon(status string) string {
	if !tv.config.Unicode {
		return ">"
	}

	switch status {
	case "completed":
		return "✅"
	case "in_progress":
		return "🔄"
	case "pending":
		return "⏳"
	case "cancelled":
		return "❌"
	default:
		return "❓"
	}
}

func (tv *terminalVisualizer) getTrendIcon(direction entities.TrendDirection) string {
	if !tv.config.Unicode {
		switch direction {
		case entities.TrendDirectionUp:
			return "^"
		case entities.TrendDirectionDown:
			return "v"
		default:
			return "-"
		}
	}

	switch direction {
	case entities.TrendDirectionUp:
		return "📈"
	case entities.TrendDirectionDown:
		return "📉"
	case entities.TrendDirectionVolatile:
		return "📊"
	default:
		return "➡️"
	}
}

func (tv *terminalVisualizer) getTrendColor(direction entities.TrendDirection) string {
	switch direction {
	case entities.TrendDirectionUp:
		return "green"
	case entities.TrendDirectionDown:
		return "red"
	case entities.TrendDirectionVolatile:
		return "orange"
	default:
		return "white"
	}
}

func (tv *terminalVisualizer) getSeverityIcon(severity entities.BottleneckSeverity) string {
	if !tv.config.Unicode {
		return "!"
	}

	switch severity {
	case entities.BottleneckSeverityCritical:
		return "🚨"
	case entities.BottleneckSeverityHigh:
		return "⚠️"
	case entities.BottleneckSeverityMedium:
		return "🔶"
	default:
		return "🔵"
	}
}

func (tv *terminalVisualizer) getSeverityColor(severity entities.BottleneckSeverity) string {
	switch severity {
	case entities.BottleneckSeverityCritical:
		return "red"
	case entities.BottleneckSeverityHigh:
		return "orange"
	case entities.BottleneckSeverityMedium:
		return "yellow"
	default:
		return "white"
	}
}

// Utility methods

func (tv *terminalVisualizer) colorize(text, color string) string {
	if !tv.config.Colors {
		return text
	}

	colors := map[string]string{
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
		"orange":  "\033[33m", // fallback to yellow
		"bold":    "\033[1m",
		"reset":   "\033[0m",
	}

	if colorCode, exists := colors[color]; exists {
		return colorCode + text + colors["reset"]
	}

	return text
}

func (tv *terminalVisualizer) formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh%dm", hours, minutes)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		if hours == 0 {
			return fmt.Sprintf("%dd", days)
		}
		return fmt.Sprintf("%dd%dh", days, hours)
	}
}

func (tv *terminalVisualizer) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}
