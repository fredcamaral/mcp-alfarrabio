package visualization

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// AnalyticsExporter implements analytics export functionality
type AnalyticsExporter struct {
	logger *slog.Logger
}

// NewAnalyticsExporter creates a new analytics exporter
func NewAnalyticsExporter(logger *slog.Logger) ports.AnalyticsExporter {
	return &AnalyticsExporter{
		logger: logger,
	}
}

// Export exports analytics data to specified format
func (e *AnalyticsExporter) Export(metrics *entities.WorkflowMetrics, format entities.ExportFormat, outputPath string) (string, error) {
	filename := filepath.Join(outputPath, fmt.Sprintf("analytics_%s.%s", metrics.Repository, e.getFileExtension(format)))

	e.logger.Debug("exporting analytics",
		slog.String("repository", metrics.Repository),
		slog.String("format", string(format)),
		slog.String("filename", filename))

	switch format {
	case entities.ExportFormatJSON:
		return e.exportJSON(metrics, filename)
	case entities.ExportFormatCSV:
		return e.exportCSV(metrics, filename)
	case entities.ExportFormatHTML:
		return e.exportHTML(metrics, filename)
	case entities.ExportFormatPDF:
		return e.exportPDF(metrics, filename)
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}
}

func (e *AnalyticsExporter) getFileExtension(format entities.ExportFormat) string {
	switch format {
	case entities.ExportFormatJSON:
		return "json"
	case entities.ExportFormatCSV:
		return "csv"
	case entities.ExportFormatHTML:
		return "html"
	case entities.ExportFormatPDF:
		return "pdf"
	default:
		return "txt"
	}
}

func (e *AnalyticsExporter) exportJSON(metrics *entities.WorkflowMetrics, filename string) (string, error) {
	e.logger.Debug("exporting to JSON", slog.String("filename", filename))

	// Create export data structure with organized metrics
	exportData := map[string]interface{}{
		"export_info": map[string]interface{}{
			"exported_at": time.Now().Format(time.RFC3339),
			"format":      "json",
			"version":     "1.0.0",
		},
		"repository": metrics.Repository,
		"period": map[string]interface{}{
			"start": metrics.Period.Start.Format(time.RFC3339),
			"end":   metrics.Period.End.Format(time.RFC3339),
		},
		"overall_score": metrics.GetOverallScore(),
		"productivity": map[string]interface{}{
			"score":            metrics.Productivity.Score,
			"tasks_per_day":    metrics.Productivity.TasksPerDay,
			"focus_time_hours": metrics.Productivity.FocusTime.Hours(),
			"deep_work_ratio":  metrics.Productivity.DeepWorkRatio,
			"context_switches": metrics.Productivity.ContextSwitches,
			"peak_hours":       metrics.Productivity.PeakHours,
			"by_priority":      metrics.Productivity.ByPriority,
			"by_type":          metrics.Productivity.ByType,
		},
		"velocity": map[string]interface{}{
			"current_velocity": metrics.Velocity.CurrentVelocity,
			"trend_direction":  metrics.Velocity.TrendDirection,
			"trend_percentage": metrics.Velocity.TrendPercentage,
			"consistency":      metrics.Velocity.Consistency,
			"weekly_data":      metrics.Velocity.ByWeek,
			"forecast": map[string]interface{}{
				"predicted_velocity": metrics.Velocity.Forecast.PredictedVelocity,
				"confidence":         metrics.Velocity.Forecast.Confidence,
				"range":              metrics.Velocity.Forecast.Range,
			},
		},
		"completion": map[string]interface{}{
			"total_tasks":        metrics.Completion.TotalTasks,
			"completed":          metrics.Completion.Completed,
			"in_progress":        metrics.Completion.InProgress,
			"cancelled":          metrics.Completion.Cancelled,
			"completion_rate":    metrics.Completion.CompletionRate,
			"average_time_hours": metrics.Completion.AverageTime.Hours(),
			"on_time_rate":       metrics.Completion.OnTimeRate,
			"quality_score":      metrics.Completion.QualityScore,
			"by_status":          metrics.Completion.ByStatus,
			"by_priority":        metrics.Completion.ByPriority,
		},
		"cycle_time": map[string]interface{}{
			"average_hours":    metrics.CycleTime.AverageCycleTime.Hours(),
			"median_hours":     metrics.CycleTime.MedianCycleTime.Hours(),
			"p90_hours":        metrics.CycleTime.P90CycleTime.Hours(),
			"lead_time_hours":  metrics.CycleTime.LeadTime.Hours(),
			"wait_time_hours":  metrics.CycleTime.WaitTime.Hours(),
			"efficiency_score": metrics.CycleTime.GetEfficiencyScore(),
			"by_type":          e.convertDurationMapToHours(metrics.CycleTime.ByType),
			"distribution":     e.convertDistribution(metrics.CycleTime.Distribution),
		},
		"bottlenecks": e.convertBottlenecks(metrics.Bottlenecks),
		"trends": map[string]interface{}{
			"productivity_trend": e.convertTrend(metrics.Trends.ProductivityTrend),
			"velocity_trend":     e.convertTrend(metrics.Trends.VelocityTrend),
			"quality_trend":      e.convertTrend(metrics.Trends.QualityTrend),
			"efficiency_trend":   e.convertTrend(metrics.Trends.EfficiencyTrend),
			"predictions":        metrics.Trends.Predictions,
			"seasonality":        metrics.Trends.Seasonality,
		},
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Marshal to JSON with pretty printing
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write JSON file: %w", err)
	}

	e.logger.Info("analytics exported to JSON", slog.String("filename", filename))
	return filename, nil
}

func (e *AnalyticsExporter) exportCSV(metrics *entities.WorkflowMetrics, filename string) (string, error) {
	e.logger.Debug("exporting to CSV", slog.String("filename", filename))

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create CSV file
	file, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			e.logger.Warn("failed to close CSV file", slog.Any("error", err))
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Helper function to write CSV records with error handling
	writeRecord := func(record []string) error {
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
		return nil
	}

	// Write metadata section
	records := [][]string{
		{"# Analytics Export Metadata"},
		{"Repository", metrics.Repository},
		{"Period Start", metrics.Period.Start.Format("2006-01-02")},
		{"Period End", metrics.Period.End.Format("2006-01-02")},
		{"Overall Score", fmt.Sprintf("%.2f", metrics.GetOverallScore())},
		{"Exported At", time.Now().Format(time.RFC3339)},
		{}, // Empty row
		{"# Productivity Metrics"},
		{"Metric", "Value", "Unit"},
		{"Score", fmt.Sprintf("%.1f", metrics.Productivity.Score), "points"},
		{"Tasks per Day", fmt.Sprintf("%.1f", metrics.Productivity.TasksPerDay), "tasks"},
		{"Focus Time", fmt.Sprintf("%.1f", metrics.Productivity.FocusTime.Hours()), "hours"},
		{"Deep Work Ratio", fmt.Sprintf("%.0f", metrics.Productivity.DeepWorkRatio*100), "percent"},
		{"Context Switches", strconv.Itoa(metrics.Productivity.ContextSwitches), "count"},
		{}, // Empty row
		{"# Velocity Metrics"},
	}

	for _, record := range records {
		if err := writeRecord(record); err != nil {
			return "", err
		}
	}
	writer.Write([]string{"Metric", "Value", "Unit"})
	writer.Write([]string{"Current Velocity", fmt.Sprintf("%.1f", metrics.Velocity.CurrentVelocity), "tasks/week"})
	writer.Write([]string{"Trend Direction", string(metrics.Velocity.TrendDirection), ""})
	writer.Write([]string{"Trend Percentage", fmt.Sprintf("%.1f", metrics.Velocity.TrendPercentage), "percent"})
	writer.Write([]string{"Consistency", fmt.Sprintf("%.0f", metrics.Velocity.Consistency*100), "percent"})
	writer.Write([]string{}) // Empty row

	// Write weekly velocity data
	if len(metrics.Velocity.ByWeek) > 0 {
		writer.Write([]string{"# Weekly Velocity Data"})
		writer.Write([]string{"Week Number", "Velocity", "Tasks Completed"})
		for _, week := range metrics.Velocity.ByWeek {
			writer.Write([]string{
				fmt.Sprintf("W%02d", week.Number),
				fmt.Sprintf("%.1f", week.Velocity),
				strconv.Itoa(week.Tasks),
			})
		}
		writer.Write([]string{}) // Empty row
	}

	// Write completion metrics
	writer.Write([]string{"# Completion Metrics"})
	writer.Write([]string{"Metric", "Value", "Unit"})
	writer.Write([]string{"Total Tasks", strconv.Itoa(metrics.Completion.TotalTasks), "count"})
	writer.Write([]string{"Completed", strconv.Itoa(metrics.Completion.Completed), "count"})
	writer.Write([]string{"In Progress", strconv.Itoa(metrics.Completion.InProgress), "count"})
	writer.Write([]string{"Cancelled", strconv.Itoa(metrics.Completion.Cancelled), "count"})
	writer.Write([]string{"Completion Rate", fmt.Sprintf("%.0f", metrics.Completion.CompletionRate*100), "percent"})
	writer.Write([]string{"Average Time", fmt.Sprintf("%.1f", metrics.Completion.AverageTime.Hours()), "hours"})
	writer.Write([]string{"On Time Rate", fmt.Sprintf("%.0f", metrics.Completion.OnTimeRate*100), "percent"})
	writer.Write([]string{"Quality Score", fmt.Sprintf("%.0f", metrics.Completion.QualityScore*100), "percent"})
	writer.Write([]string{}) // Empty row

	// Write cycle time metrics
	writer.Write([]string{"# Cycle Time Metrics"})
	writer.Write([]string{"Metric", "Value", "Unit"})
	writer.Write([]string{"Average Cycle Time", fmt.Sprintf("%.1f", metrics.CycleTime.AverageCycleTime.Hours()), "hours"})
	writer.Write([]string{"Median Cycle Time", fmt.Sprintf("%.1f", metrics.CycleTime.MedianCycleTime.Hours()), "hours"})
	writer.Write([]string{"90th Percentile", fmt.Sprintf("%.1f", metrics.CycleTime.P90CycleTime.Hours()), "hours"})
	writer.Write([]string{"Lead Time", fmt.Sprintf("%.1f", metrics.CycleTime.LeadTime.Hours()), "hours"})
	writer.Write([]string{"Wait Time", fmt.Sprintf("%.1f", metrics.CycleTime.WaitTime.Hours()), "hours"})
	writer.Write([]string{"Efficiency Score", fmt.Sprintf("%.0f", metrics.CycleTime.GetEfficiencyScore()*100), "percent"})
	writer.Write([]string{}) // Empty row

	// Write bottlenecks if any
	if len(metrics.Bottlenecks) > 0 {
		writer.Write([]string{"# Bottlenecks"})
		writer.Write([]string{"Severity", "Description", "Impact (hours)", "Frequency"})
		for _, bottleneck := range metrics.Bottlenecks {
			writer.Write([]string{
				string(bottleneck.Severity),
				bottleneck.Description,
				fmt.Sprintf("%.1f", bottleneck.Impact),
				strconv.Itoa(bottleneck.Frequency),
			})
		}
		writer.Write([]string{}) // Empty row
	}

	// Write priority breakdown
	if len(metrics.Productivity.ByPriority) > 0 {
		writer.Write([]string{"# Priority Performance"})
		writer.Write([]string{"Priority", "Completion Rate", "Percentage"})
		for priority, rate := range metrics.Productivity.ByPriority {
			writer.Write([]string{
				strings.Title(priority),
				fmt.Sprintf("%.1f", rate),
				fmt.Sprintf("%.0f", rate*100),
			})
		}
		writer.Write([]string{}) // Empty row
	}

	// Write task type breakdown
	if len(metrics.Productivity.ByType) > 0 {
		writer.Write([]string{"# Task Type Performance"})
		writer.Write([]string{"Task Type", "Completion Rate", "Percentage"})
		for taskType, rate := range metrics.Productivity.ByType {
			writer.Write([]string{
				taskType,
				fmt.Sprintf("%.1f", rate),
				fmt.Sprintf("%.0f", rate*100),
			})
		}
	}

	e.logger.Info("analytics exported to CSV", slog.String("filename", filename))
	return filename, nil
}

func (e *AnalyticsExporter) exportHTML(metrics *entities.WorkflowMetrics, filename string) (string, error) {
	e.logger.Debug("exporting to HTML", slog.String("filename", filename))

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create HTML template
	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Analytics Report - {{.Repository}}</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 40px; border-bottom: 2px solid #e1e5e9; padding-bottom: 20px; }
        .header h1 { color: #2c3e50; margin: 0; font-size: 2.5em; }
        .header .subtitle { color: #7f8c8d; margin-top: 10px; font-size: 1.1em; }
        .score-card { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; border-radius: 8px; text-align: center; margin-bottom: 30px; }
        .score-card .score { font-size: 3em; font-weight: bold; margin: 0; }
        .score-card .label { font-size: 1.1em; opacity: 0.9; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 30px; margin-bottom: 30px; }
        .metric-card { background: white; border: 1px solid #e1e5e9; border-radius: 8px; padding: 20px; }
        .metric-card h3 { margin-top: 0; color: #2c3e50; border-bottom: 2px solid #3498db; padding-bottom: 10px; }
        .metric-row { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; border-bottom: 1px solid #ecf0f1; }
        .metric-row:last-child { border-bottom: none; }
        .metric-label { font-weight: 500; color: #34495e; }
        .metric-value { font-weight: bold; color: #2c3e50; }
        .progress-bar { background: #ecf0f1; height: 8px; border-radius: 4px; overflow: hidden; margin: 5px 0; }
        .progress-fill { height: 100%; background: linear-gradient(90deg, #3498db, #2ecc71); transition: width 0.3s ease; }
        .bottlenecks { margin-top: 30px; }
        .bottleneck { background: #fff5f5; border-left: 4px solid #e74c3c; padding: 15px; margin: 10px 0; border-radius: 0 4px 4px 0; }
        .bottleneck.high { border-left-color: #e74c3c; background: #fff5f5; }
        .bottleneck.medium { border-left-color: #f39c12; background: #fffaf0; }
        .bottleneck.low { border-left-color: #3498db; background: #f0f8ff; }
        .bottleneck-title { font-weight: bold; margin-bottom: 5px; }
        .bottleneck-impact { font-size: 0.9em; color: #7f8c8d; }
        .trend { display: inline-flex; align-items: center; gap: 5px; }
        .trend.up { color: #27ae60; }
        .trend.down { color: #e74c3c; }
        .trend.stable { color: #7f8c8d; }
        .export-info { margin-top: 40px; padding-top: 20px; border-top: 1px solid #e1e5e9; font-size: 0.9em; color: #7f8c8d; text-align: center; }
        .chart-placeholder { background: #f8f9fa; border: 2px dashed #dee2e6; padding: 40px; text-align: center; color: #6c757d; border-radius: 4px; margin: 15px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>📊 Analytics Report</h1>
            <div class="subtitle">{{.Repository}} • {{.Period.Start.Format "Jan 2"}} - {{.Period.End.Format "Jan 2, 2006"}}</div>
        </div>

        <div class="score-card">
            <div class="score">{{printf "%.0f" (.GetOverallScore | mul 100)}}</div>
            <div class="label">Overall Performance Score</div>
        </div>

        <div class="grid">
            <div class="metric-card">
                <h3>📈 Productivity</h3>
                <div class="metric-row">
                    <span class="metric-label">Score</span>
                    <span class="metric-value">{{printf "%.1f" .Productivity.Score}}/100</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill" style="width: {{.Productivity.Score}}%"></div>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Tasks per Day</span>
                    <span class="metric-value">{{printf "%.1f" .Productivity.TasksPerDay}}</span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Focus Time</span>
                    <span class="metric-value">{{printf "%.1f" .Productivity.FocusTime.Hours}}h</span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Deep Work</span>
                    <span class="metric-value">{{printf "%.0f" (mul .Productivity.DeepWorkRatio 100)}}%</span>
                </div>
            </div>

            <div class="metric-card">
                <h3>🚀 Velocity</h3>
                <div class="metric-row">
                    <span class="metric-label">Current</span>
                    <span class="metric-value">{{printf "%.1f" .Velocity.CurrentVelocity}} tasks/week</span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Trend</span>
                    <span class="metric-value trend {{if eq .Velocity.TrendDirection "up"}}up{{else if eq .Velocity.TrendDirection "down"}}down{{else}}stable{{end}}">
                        {{if eq .Velocity.TrendDirection "up"}}📈{{else if eq .Velocity.TrendDirection "down"}}📉{{else}}➡️{{end}}
                        {{printf "%.1f" .Velocity.TrendPercentage}}%
                    </span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Consistency</span>
                    <span class="metric-value">{{printf "%.0f" (mul .Velocity.Consistency 100)}}%</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill" style="width: {{mul .Velocity.Consistency 100}}%"></div>
                </div>
            </div>

            <div class="metric-card">
                <h3>✅ Completion</h3>
                <div class="metric-row">
                    <span class="metric-label">Total Tasks</span>
                    <span class="metric-value">{{.Completion.TotalTasks}}</span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Completed</span>
                    <span class="metric-value">{{.Completion.Completed}} ({{printf "%.0f" (mul .Completion.CompletionRate 100)}}%)</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill" style="width: {{mul .Completion.CompletionRate 100}}%"></div>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Average Time</span>
                    <span class="metric-value">{{printf "%.1f" .Completion.AverageTime.Hours}}h</span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Quality Score</span>
                    <span class="metric-value">{{printf "%.0f" (mul .Completion.QualityScore 100)}}%</span>
                </div>
            </div>

            <div class="metric-card">
                <h3>⏰ Cycle Time</h3>
                <div class="metric-row">
                    <span class="metric-label">Average</span>
                    <span class="metric-value">{{printf "%.1f" .CycleTime.AverageCycleTime.Hours}}h</span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Median</span>
                    <span class="metric-value">{{printf "%.1f" .CycleTime.MedianCycleTime.Hours}}h</span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">90th Percentile</span>
                    <span class="metric-value">{{printf "%.1f" .CycleTime.P90CycleTime.Hours}}h</span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Efficiency</span>
                    <span class="metric-value">{{printf "%.0f" (mul (.CycleTime.GetEfficiencyScore) 100)}}%</span>
                </div>
                <div class="progress-bar">
                    <div class="progress-fill" style="width: {{mul (.CycleTime.GetEfficiencyScore) 100}}%"></div>
                </div>
            </div>
        </div>

        {{if .Bottlenecks}}
        <div class="bottlenecks">
            <h3>🚨 Detected Bottlenecks</h3>
            {{range .Bottlenecks}}
            <div class="bottleneck {{.Severity}}">
                <div class="bottleneck-title">{{.Description}}</div>
                <div class="bottleneck-impact">Impact: {{printf "%.1f" .Impact}} hours lost • Frequency: {{.Frequency}} occurrences</div>
            </div>
            {{end}}
        </div>
        {{end}}

        <div class="export-info">
            Report generated on {{.Now.Format "January 2, 2006 at 3:04 PM"}} • 
            Export format: HTML • 
            Data period: {{.Period.Start.Format "Jan 2"}} - {{.Period.End.Format "Jan 2, 2006"}}
        </div>
    </div>
</body>
</html>`

	// Helper functions for template
	funcMap := template.FuncMap{
		"mul": func(a, b float64) float64 { return a * b },
		"add": func(a, b int) int { return a + b },
	}

	// Parse template
	tmpl, err := template.New("analytics").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	// Prepare template data
	templateData := struct {
		*entities.WorkflowMetrics
		Now time.Time
	}{
		WorkflowMetrics: metrics,
		Now:             time.Now(),
	}

	// Execute template
	var htmlBuffer bytes.Buffer
	if err := tmpl.Execute(&htmlBuffer, templateData); err != nil {
		return "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, htmlBuffer.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write HTML file: %w", err)
	}

	e.logger.Info("analytics exported to HTML", slog.String("filename", filename))
	return filename, nil
}

func (e *AnalyticsExporter) exportPDF(metrics *entities.WorkflowMetrics, filename string) (string, error) {
	e.logger.Debug("exporting to PDF", slog.String("filename", filename))

	// For PDF export, we'll create a structured text report that can be easily converted
	// In a production environment, you would use a PDF library like gofpdf
	// For now, we'll create a well-formatted text report with PDF extension

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	var report strings.Builder

	// Header
	report.WriteString("====================================================================\n")
	report.WriteString("                    ANALYTICS REPORT (PDF)\n")
	report.WriteString("====================================================================\n\n")

	report.WriteString(fmt.Sprintf("Repository: %s\n", metrics.Repository))
	report.WriteString(fmt.Sprintf("Period: %s to %s\n",
		metrics.Period.Start.Format("Jan 2, 2006"),
		metrics.Period.End.Format("Jan 2, 2006")))
	report.WriteString(fmt.Sprintf("Overall Score: %.0f/100\n", metrics.GetOverallScore()*100))
	report.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("January 2, 2006 at 3:04 PM")))

	// Productivity Section
	report.WriteString("--------------------------------------------------------------------\n")
	report.WriteString("📈 PRODUCTIVITY METRICS\n")
	report.WriteString("--------------------------------------------------------------------\n")
	report.WriteString(fmt.Sprintf("Score:              %.1f/100\n", metrics.Productivity.Score))
	report.WriteString(fmt.Sprintf("Tasks per Day:      %.1f\n", metrics.Productivity.TasksPerDay))
	report.WriteString(fmt.Sprintf("Focus Time:         %.1f hours\n", metrics.Productivity.FocusTime.Hours()))
	report.WriteString(fmt.Sprintf("Deep Work Ratio:    %.0f%%\n", metrics.Productivity.DeepWorkRatio*100))
	report.WriteString(fmt.Sprintf("Context Switches:   %d\n", metrics.Productivity.ContextSwitches))

	if len(metrics.Productivity.PeakHours) > 0 {
		report.WriteString("Peak Hours:         ")
		for i, hour := range metrics.Productivity.PeakHours {
			if i > 0 {
				report.WriteString(", ")
			}
			report.WriteString(fmt.Sprintf("%02d:00", hour))
		}
		report.WriteString("\n")
	}
	report.WriteString("\n")

	// Velocity Section
	report.WriteString("--------------------------------------------------------------------\n")
	report.WriteString("🚀 VELOCITY METRICS\n")
	report.WriteString("--------------------------------------------------------------------\n")
	report.WriteString(fmt.Sprintf("Current Velocity:   %.1f tasks/week\n", metrics.Velocity.CurrentVelocity))
	report.WriteString(fmt.Sprintf("Trend Direction:    %s\n", metrics.Velocity.TrendDirection))
	report.WriteString(fmt.Sprintf("Trend Percentage:   %.1f%%\n", metrics.Velocity.TrendPercentage))
	report.WriteString(fmt.Sprintf("Consistency:        %.0f%%\n", metrics.Velocity.Consistency*100))

	if metrics.Velocity.Forecast.Confidence > 0.5 {
		report.WriteString(fmt.Sprintf("Forecast:           %.1f tasks (%.0f%% confidence)\n",
			metrics.Velocity.Forecast.PredictedVelocity,
			metrics.Velocity.Forecast.Confidence*100))
	}
	report.WriteString("\n")

	// Completion Section
	report.WriteString("--------------------------------------------------------------------\n")
	report.WriteString("✅ COMPLETION METRICS\n")
	report.WriteString("--------------------------------------------------------------------\n")
	report.WriteString(fmt.Sprintf("Total Tasks:        %d\n", metrics.Completion.TotalTasks))
	report.WriteString(fmt.Sprintf("Completed:          %d (%.0f%%)\n",
		metrics.Completion.Completed,
		metrics.Completion.CompletionRate*100))
	report.WriteString(fmt.Sprintf("In Progress:        %d\n", metrics.Completion.InProgress))
	report.WriteString(fmt.Sprintf("Cancelled:          %d\n", metrics.Completion.Cancelled))
	report.WriteString(fmt.Sprintf("Average Time:       %.1f hours\n", metrics.Completion.AverageTime.Hours()))
	report.WriteString(fmt.Sprintf("On Time Rate:       %.0f%%\n", metrics.Completion.OnTimeRate*100))
	report.WriteString(fmt.Sprintf("Quality Score:      %.0f%%\n", metrics.Completion.QualityScore*100))
	report.WriteString("\n")

	// Cycle Time Section
	report.WriteString("--------------------------------------------------------------------\n")
	report.WriteString("⏰ CYCLE TIME ANALYSIS\n")
	report.WriteString("--------------------------------------------------------------------\n")
	report.WriteString(fmt.Sprintf("Average Cycle Time: %.1f hours\n", metrics.CycleTime.AverageCycleTime.Hours()))
	report.WriteString(fmt.Sprintf("Median Cycle Time:  %.1f hours\n", metrics.CycleTime.MedianCycleTime.Hours()))
	report.WriteString(fmt.Sprintf("90th Percentile:    %.1f hours\n", metrics.CycleTime.P90CycleTime.Hours()))
	report.WriteString(fmt.Sprintf("Lead Time:          %.1f hours\n", metrics.CycleTime.LeadTime.Hours()))
	report.WriteString(fmt.Sprintf("Wait Time:          %.1f hours\n", metrics.CycleTime.WaitTime.Hours()))
	report.WriteString(fmt.Sprintf("Efficiency Score:   %.0f%%\n", metrics.CycleTime.GetEfficiencyScore()*100))
	report.WriteString("\n")

	// Bottlenecks Section
	if len(metrics.Bottlenecks) > 0 {
		report.WriteString("--------------------------------------------------------------------\n")
		report.WriteString("🚨 DETECTED BOTTLENECKS\n")
		report.WriteString("--------------------------------------------------------------------\n")
		for i, bottleneck := range metrics.Bottlenecks {
			if i >= 5 { // Limit to top 5
				break
			}
			report.WriteString(fmt.Sprintf("%d. [%s] %s\n",
				i+1,
				strings.ToUpper(string(bottleneck.Severity)),
				bottleneck.Description))
			report.WriteString(fmt.Sprintf("   Impact: %.1f hours lost • Frequency: %d occurrences\n",
				bottleneck.Impact,
				bottleneck.Frequency))
			if len(bottleneck.Suggestions) > 0 {
				report.WriteString("   Suggestions:\n")
				for _, suggestion := range bottleneck.Suggestions {
					report.WriteString(fmt.Sprintf("   • %s\n", suggestion))
				}
			}
			report.WriteString("\n")
		}
	}

	// Priority Breakdown
	if len(metrics.Productivity.ByPriority) > 0 {
		report.WriteString("--------------------------------------------------------------------\n")
		report.WriteString("📊 PRIORITY PERFORMANCE\n")
		report.WriteString("--------------------------------------------------------------------\n")
		priorities := []string{"high", "medium", "low"}
		for _, priority := range priorities {
			if rate, exists := metrics.Productivity.ByPriority[priority]; exists {
				report.WriteString(fmt.Sprintf("%-8s: %.0f%% completion rate\n",
					strings.Title(priority),
					rate*100))
			}
		}
		report.WriteString("\n")
	}

	// Task Type Breakdown
	if len(metrics.Productivity.ByType) > 0 {
		report.WriteString("--------------------------------------------------------------------\n")
		report.WriteString("📋 TASK TYPE PERFORMANCE\n")
		report.WriteString("--------------------------------------------------------------------\n")
		count := 0
		for taskType, rate := range metrics.Productivity.ByType {
			if count >= 10 { // Limit to top 10
				break
			}
			report.WriteString(fmt.Sprintf("%-20s: %.0f%%\n", taskType, rate*100))
			count++
		}
		report.WriteString("\n")
	}

	// Footer
	report.WriteString("====================================================================\n")
	report.WriteString("End of Report\n")
	report.WriteString("====================================================================\n")

	// Note about PDF generation
	report.WriteString("\nNOTE: This is a text-based PDF export. For full PDF functionality\n")
	report.WriteString("with graphics and charts, consider using a dedicated PDF library.\n")

	// Write to file
	if err := os.WriteFile(filename, []byte(report.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write PDF file: %w", err)
	}

	e.logger.Info("analytics exported to PDF (text format)", slog.String("filename", filename))
	return filename, nil
}

// convertReportToMetrics converts a ProductivityReport to WorkflowMetrics for export
func (e *AnalyticsExporter) convertReportToMetrics(report *entities.ProductivityReport) *entities.WorkflowMetrics {
	// Use the metrics from the report directly, as ProductivityReport contains WorkflowMetrics
	return &report.Metrics
}

// Helper methods for JSON export

// convertDurationMapToHours converts a map of durations to hours for JSON export
func (e *AnalyticsExporter) convertDurationMapToHours(durationMap map[string]time.Duration) map[string]float64 {
	result := make(map[string]float64)
	for key, duration := range durationMap {
		result[key] = duration.Hours()
	}
	return result
}

// convertDistribution converts cycle time distribution to JSON-friendly format
func (e *AnalyticsExporter) convertDistribution(distribution []entities.CycleTimePoint) []map[string]interface{} {
	result := make([]map[string]interface{}, len(distribution))
	for i, point := range distribution {
		result[i] = map[string]interface{}{
			"duration_hours": point.Duration.Hours(),
			"count":          point.Count,
			"percentile":     point.Percentile,
		}
	}
	return result
}

// convertBottlenecks converts bottlenecks to JSON-friendly format
func (e *AnalyticsExporter) convertBottlenecks(bottlenecks []entities.Bottleneck) []map[string]interface{} {
	result := make([]map[string]interface{}, len(bottlenecks))
	for i, bottleneck := range bottlenecks {
		result[i] = map[string]interface{}{
			"type":           bottleneck.Type,
			"description":    bottleneck.Description,
			"impact_hours":   bottleneck.Impact,
			"frequency":      bottleneck.Frequency,
			"severity":       string(bottleneck.Severity),
			"suggestions":    bottleneck.Suggestions,
			"affected_tasks": bottleneck.AffectedTasks,
			"detected_at":    bottleneck.DetectedAt.Format(time.RFC3339),
		}
	}
	return result
}

// convertTrend converts trend data to JSON-friendly format
func (e *AnalyticsExporter) convertTrend(trend entities.Trend) map[string]interface{} {
	trendLine := make([]map[string]interface{}, len(trend.TrendLine))
	for i, point := range trend.TrendLine {
		trendLine[i] = map[string]interface{}{
			"time":  point.Time.Format(time.RFC3339),
			"value": point.Value,
		}
	}

	return map[string]interface{}{
		"direction":   string(trend.Direction),
		"strength":    trend.Strength,
		"confidence":  trend.Confidence,
		"change_rate": trend.ChangeRate,
		"start_value": trend.StartValue,
		"end_value":   trend.EndValue,
		"trend_line":  trendLine,
		"description": trend.Description,
	}
}

// ExportReport exports productivity report to specified format
func (e *AnalyticsExporter) ExportReport(report *entities.ProductivityReport, format entities.ExportFormat, outputPath string) (string, error) {
	filename := filepath.Join(outputPath, fmt.Sprintf("report_%s.%s", report.Repository, e.getFileExtension(format)))

	e.logger.Debug("exporting report",
		slog.String("repository", report.Repository),
		slog.String("format", string(format)),
		slog.String("filename", filename))

	// Convert ProductivityReport to WorkflowMetrics for export
	metrics := e.convertReportToMetrics(report)

	switch format {
	case entities.ExportFormatJSON:
		return e.exportJSON(metrics, filename)
	case entities.ExportFormatCSV:
		return e.exportCSV(metrics, filename)
	case entities.ExportFormatHTML:
		return e.exportHTML(metrics, filename)
	case entities.ExportFormatPDF:
		return e.exportPDF(metrics, filename)
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}
}
