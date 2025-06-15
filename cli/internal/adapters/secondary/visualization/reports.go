package visualization

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// ReportGeneratorConfig holds configuration for report generation
type ReportGeneratorConfig struct {
	OutputDir      string `json:"output_dir"`
	IncludeCharts  bool   `json:"include_charts"`
	IncludeRaw     bool   `json:"include_raw"`
	Compress       bool   `json:"compress"`
	TimestampFiles bool   `json:"timestamp_files"`
}

// DefaultReportGeneratorConfig returns default configuration
func DefaultReportGeneratorConfig() *ReportGeneratorConfig {
	return &ReportGeneratorConfig{
		OutputDir:      "./analytics_reports",
		IncludeCharts:  true,
		IncludeRaw:     false,
		Compress:       false,
		TimestampFiles: true,
	}
}

// analyticsExporter implements AnalyticsExporter interface
type analyticsExporter struct {
	config     *ReportGeneratorConfig
	visualizer Visualizer
}

// NewReportExporter creates a new report exporter
func NewReportExporter(config *ReportGeneratorConfig, visualizer Visualizer) *analyticsExporter {
	if config == nil {
		config = DefaultReportGeneratorConfig()
	}

	return &analyticsExporter{
		config:     config,
		visualizer: visualizer,
	}
}

// Export exports workflow metrics in the specified format
func (ae *analyticsExporter) Export(
	metrics *entities.WorkflowMetrics,
	format entities.ExportFormat,
) (string, error) {
	// Ensure output directory exists
	if err := ae.ensureOutputDir(); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename
	filename := ae.generateFilename("workflow_metrics", format)
	filePath := filepath.Join(ae.config.OutputDir, filename)

	switch format {
	case entities.ExportFormatJSON:
		return ae.exportJSON(metrics, filePath)
	case entities.ExportFormatCSV:
		return ae.exportCSV(metrics, filePath)
	case entities.ExportFormatHTML:
		return ae.exportHTML(metrics, filePath)
	case entities.ExportFormatPDF:
		return ae.exportPDF(metrics, filePath)
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}
}

// ExportReport exports a complete productivity report
func (ae *analyticsExporter) ExportReport(
	report *entities.ProductivityReport,
	format entities.ExportFormat,
) (string, error) {
	// Ensure output directory exists
	if err := ae.ensureOutputDir(); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename
	filename := ae.generateFilename("productivity_report", format)
	filePath := filepath.Join(ae.config.OutputDir, filename)

	switch format {
	case entities.ExportFormatJSON:
		return ae.exportReportJSON(report, filePath)
	case entities.ExportFormatHTML:
		return ae.exportReportHTML(report, filePath)
	case entities.ExportFormatPDF:
		return ae.exportReportPDF(report, filePath)
	default:
		return "", fmt.Errorf("unsupported export format for reports: %s", format)
	}
}

// GetSupportedFormats returns the list of supported export formats
func (ae *analyticsExporter) GetSupportedFormats() []entities.ExportFormat {
	return []entities.ExportFormat{
		entities.ExportFormatJSON,
		entities.ExportFormatCSV,
		entities.ExportFormatHTML,
		entities.ExportFormatPDF,
	}
}

// JSON export implementations

func (ae *analyticsExporter) exportJSON(metrics *entities.WorkflowMetrics, filePath string) (string, error) {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return "", fmt.Errorf("path traversal detected: %s", filePath)
	}

	file, err := os.Create(filePath) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't return as we're in defer
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(metrics); err != nil {
		return "", fmt.Errorf("failed to encode metrics: %w", err)
	}

	return filePath, nil
}

func (ae *analyticsExporter) exportReportJSON(report *entities.ProductivityReport, filePath string) (string, error) {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return "", fmt.Errorf("path traversal detected: %s", filePath)
	}

	file, err := os.Create(filePath) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't return as we're in defer
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(report); err != nil {
		return "", fmt.Errorf("failed to encode report: %w", err)
	}

	return filePath, nil
}

// CSV export implementation

func (ae *analyticsExporter) exportCSV(metrics *entities.WorkflowMetrics, filePath string) (string, error) {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return "", fmt.Errorf("path traversal detected: %s", filePath)
	}

	file, err := os.Create(filePath) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't return as we're in defer
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers and data for different metric types

	// Productivity metrics
	if err := ae.writeProductivityCSV(writer, metrics.Productivity); err != nil {
		return "", fmt.Errorf("failed to write productivity metrics: %w", err)
	}

	// Velocity metrics
	if err := ae.writeVelocityCSV(writer, metrics.Velocity); err != nil {
		return "", fmt.Errorf("failed to write velocity metrics: %w", err)
	}

	// Completion metrics
	if err := ae.writeCompletionCSV(writer, metrics.Completion); err != nil {
		return "", fmt.Errorf("failed to write completion metrics: %w", err)
	}

	// Bottlenecks
	if err := ae.writeBottlenecksCSV(writer, metrics.Bottlenecks); err != nil {
		return "", fmt.Errorf("failed to write bottlenecks: %w", err)
	}

	return filePath, nil
}

func (ae *analyticsExporter) writeProductivityCSV(writer *csv.Writer, metrics entities.ProductivityMetrics) error {
	// Productivity summary
	_ = writer.Write([]string{"Metric", "Value"})
	_ = writer.Write([]string{"Productivity Score", fmt.Sprintf("%.1f", metrics.Score)})
	_ = writer.Write([]string{"Tasks Per Day", fmt.Sprintf("%.2f", metrics.TasksPerDay)})
	_ = writer.Write([]string{"Focus Time (minutes)", fmt.Sprintf("%.0f", metrics.FocusTime.Minutes())})
	_ = writer.Write([]string{"Deep Work Ratio", fmt.Sprintf("%.2f", metrics.DeepWorkRatio)})
	_ = writer.Write([]string{"Context Switches", strconv.Itoa(metrics.ContextSwitches)})
	_ = writer.Write([]string{""}) // Empty row

	// Priority completion rates
	_ = writer.Write([]string{"Priority", "Completion Rate"})
	for priority, rate := range metrics.ByPriority {
		_ = writer.Write([]string{priority, fmt.Sprintf("%.2f", rate)})
	}
	_ = writer.Write([]string{""}) // Empty row

	// Type completion rates
	_ = writer.Write([]string{"Task Type", "Completion Rate"})
	for taskType, rate := range metrics.ByType {
		_ = writer.Write([]string{taskType, fmt.Sprintf("%.2f", rate)})
	}
	_ = writer.Write([]string{""}) // Empty row

	return nil
}

func (ae *analyticsExporter) writeVelocityCSV(writer *csv.Writer, metrics entities.VelocityMetrics) error {
	// Velocity summary
	_ = writer.Write([]string{"Velocity Metric", "Value"})
	_ = writer.Write([]string{"Current Velocity", fmt.Sprintf("%.2f", metrics.CurrentVelocity)})
	_ = writer.Write([]string{"Trend Direction", string(metrics.TrendDirection)})
	_ = writer.Write([]string{"Trend Percentage", fmt.Sprintf("%.2f", metrics.TrendPercentage)})
	_ = writer.Write([]string{"Consistency", fmt.Sprintf("%.2f", metrics.Consistency)})
	_ = writer.Write([]string{""}) // Empty row

	// Weekly velocity
	_ = writer.Write([]string{"Week", "Velocity", "Tasks"})
	for _, week := range metrics.ByWeek {
		_ = writer.Write([]string{
			fmt.Sprintf("W%d", week.Number),
			fmt.Sprintf("%.2f", week.Velocity),
			strconv.Itoa(week.Tasks),
		})
	}
	_ = writer.Write([]string{""}) // Empty row

	return nil
}

func (ae *analyticsExporter) writeCompletionCSV(writer *csv.Writer, metrics entities.CompletionMetrics) error {
	// Completion summary
	_ = writer.Write([]string{"Completion Metric", "Value"})
	_ = writer.Write([]string{"Total Tasks", strconv.Itoa(metrics.TotalTasks)})
	_ = writer.Write([]string{"Completed", strconv.Itoa(metrics.Completed)})
	_ = writer.Write([]string{"In Progress", strconv.Itoa(metrics.InProgress)})
	_ = writer.Write([]string{"Cancelled", strconv.Itoa(metrics.Cancelled)})
	_ = writer.Write([]string{"Completion Rate", fmt.Sprintf("%.2f", metrics.CompletionRate)})
	_ = writer.Write([]string{"Average Time (hours)", fmt.Sprintf("%.2f", metrics.AverageTime.Hours())})
	_ = writer.Write([]string{"On Time Rate", fmt.Sprintf("%.2f", metrics.OnTimeRate)})
	_ = writer.Write([]string{"Quality Score", fmt.Sprintf("%.2f", metrics.QualityScore)})
	_ = writer.Write([]string{""}) // Empty row

	return nil
}

func (ae *analyticsExporter) writeBottlenecksCSV(writer *csv.Writer, bottlenecks []entities.Bottleneck) error {
	if len(bottlenecks) == 0 {
		return nil
	}

	_ = writer.Write([]string{"Type", "Description", "Impact (hours)", "Frequency", "Severity"})
	for _, bottleneck := range bottlenecks {
		_ = writer.Write([]string{
			bottleneck.Type,
			bottleneck.Description,
			fmt.Sprintf("%.2f", bottleneck.Impact),
			strconv.Itoa(bottleneck.Frequency),
			string(bottleneck.Severity),
		})
	}
	_ = writer.Write([]string{""}) // Empty row

	return nil
}

// HTML export implementations

func (ae *analyticsExporter) exportHTML(metrics *entities.WorkflowMetrics, filePath string) (string, error) {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return "", fmt.Errorf("path traversal detected: %s", filePath)
	}

	file, err := os.Create(filePath) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't return as we're in defer
		}
	}()

	html := ae.generateMetricsHTML(metrics)
	if _, err := file.WriteString(html); err != nil {
		return "", fmt.Errorf("failed to write HTML: %w", err)
	}

	return filePath, nil
}

func (ae *analyticsExporter) exportReportHTML(report *entities.ProductivityReport, filePath string) (string, error) {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return "", fmt.Errorf("path traversal detected: %s", filePath)
	}

	file, err := os.Create(filePath) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't return as we're in defer
		}
	}()

	html := ae.generateReportHTML(report)
	if _, err := file.WriteString(html); err != nil {
		return "", fmt.Errorf("failed to write HTML: %w", err)
	}

	return filePath, nil
}

func (ae *analyticsExporter) generateMetricsHTML(metrics *entities.WorkflowMetrics) string {
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Workflow Analytics Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 40px; border-bottom: 2px solid #eee; padding-bottom: 20px; }
        .metric-section { margin-bottom: 30px; }
        .metric-title { color: #333; border-left: 4px solid #007acc; padding-left: 15px; margin-bottom: 15px; }
        .metric-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; }
        .metric-card { background: #f9f9f9; padding: 20px; border-radius: 6px; border: 1px solid #ddd; }
        .metric-value { font-size: 24px; font-weight: bold; color: #007acc; }
        .metric-label { color: #666; margin-bottom: 5px; }
        .progress-bar { width: 100%; height: 20px; background: #eee; border-radius: 10px; overflow: hidden; margin: 10px 0; }
        .progress-fill { height: 100%; background: linear-gradient(90deg, #ff6b6b, #feca57, #48dbfb, #ff9ff3); transition: width 0.3s; }
        .bottleneck { background: #fff5f5; border-left: 4px solid #e53e3e; padding: 15px; margin: 10px 0; border-radius: 4px; }
        .bottleneck.high { border-left-color: #e53e3e; }
        .bottleneck.medium { border-left-color: #dd6b20; }
        .bottleneck.low { border-left-color: #38a169; }
        table { width: 100%; border-collapse: collapse; margin: 15px 0; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f8f9fa; font-weight: 600; }
        .trend-up { color: #38a169; }
        .trend-down { color: #e53e3e; }
        .trend-stable { color: #666; }
    </style>
</head>
<body>
    <div class="container">`)

	// Header
	html.WriteString(fmt.Sprintf(`
        <div class="header">
            <h1>📊 Workflow Analytics Report</h1>
            <p><strong>Repository:</strong> %s</p>
            <p><strong>Period:</strong> %s to %s</p>
            <p><strong>Generated:</strong> %s</p>
            <p><strong>Overall Score:</strong> <span class="metric-value">%.0f/100</span></p>
        </div>`,
		metrics.Repository,
		metrics.Period.Start.Format("January 2, 2006"),
		metrics.Period.End.Format("January 2, 2006"),
		metrics.GeneratedAt.Format("January 2, 2006 at 3:04 PM"),
		metrics.GetOverallScore()*100))

	// Productivity metrics
	html.WriteString(ae.generateProductivityHTML(metrics.Productivity))

	// Velocity metrics
	html.WriteString(ae.generateVelocityHTML(metrics.Velocity))

	// Completion metrics
	html.WriteString(ae.generateCompletionHTML(metrics.Completion))

	// Bottlenecks
	if len(metrics.Bottlenecks) > 0 {
		html.WriteString(ae.generateBottlenecksHTML(metrics.Bottlenecks))
	}

	// Trends
	html.WriteString(ae.generateTrendsHTML(metrics.Trends))

	html.WriteString(`
    </div>
</body>
</html>`)

	return html.String()
}

func (ae *analyticsExporter) generateProductivityHTML(metrics entities.ProductivityMetrics) string {
	var html strings.Builder

	html.WriteString(`
        <div class="metric-section">
            <h2 class="metric-title">📊 Productivity Metrics</h2>
            <div class="metric-grid">`)

	// Overall score
	html.WriteString(fmt.Sprintf(`
                <div class="metric-card">
                    <div class="metric-label">Overall Score</div>
                    <div class="metric-value">%.1f/100</div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: %.1f%%"></div>
                    </div>
                </div>`, metrics.Score, metrics.Score))

	// Tasks per day
	html.WriteString(fmt.Sprintf(`
                <div class="metric-card">
                    <div class="metric-label">Tasks per Day</div>
                    <div class="metric-value">%.1f</div>
                </div>`, metrics.TasksPerDay))

	// Focus time
	html.WriteString(fmt.Sprintf(`
                <div class="metric-card">
                    <div class="metric-label">Average Focus Time</div>
                    <div class="metric-value">%s</div>
                </div>`, ae.formatDuration(metrics.FocusTime)))

	// Deep work ratio
	html.WriteString(fmt.Sprintf(`
                <div class="metric-card">
                    <div class="metric-label">Deep Work Ratio</div>
                    <div class="metric-value">%.0f%%</div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: %.0f%%"></div>
                    </div>
                </div>`, metrics.DeepWorkRatio*100, metrics.DeepWorkRatio*100))

	html.WriteString(`
            </div>`)

	// Priority completion table
	if len(metrics.ByPriority) > 0 {
		html.WriteString(`
            <h3>Completion by Priority</h3>
            <table>
                <thead>
                    <tr><th>Priority</th><th>Completion Rate</th></tr>
                </thead>
                <tbody>`)

		priorities := []string{"high", "medium", "low"}
		for _, priority := range priorities {
			if rate, exists := metrics.ByPriority[priority]; exists {
				html.WriteString(fmt.Sprintf(`
                    <tr>
                        <td>%s</td>
                        <td>%.0f%%</td>
                    </tr>`, strings.Title(priority), rate*100))
			}
		}

		html.WriteString(`
                </tbody>
            </table>`)
	}

	html.WriteString(`
        </div>`)

	return html.String()
}

func (ae *analyticsExporter) generateVelocityHTML(metrics entities.VelocityMetrics) string {
	var html strings.Builder

	html.WriteString(`
        <div class="metric-section">
            <h2 class="metric-title">📏 Velocity Metrics</h2>
            <div class="metric-grid">`)

	// Current velocity
	trendClass := ae.getTrendClass(entities.TrendDirection(metrics.TrendDirection))
	html.WriteString(fmt.Sprintf(`
                <div class="metric-card">
                    <div class="metric-label">Current Velocity</div>
                    <div class="metric-value">%.1f tasks/week</div>
                    <div class="%s">%s %.1f%%</div>
                </div>`, metrics.CurrentVelocity, trendClass,
		ae.getTrendSymbol(entities.TrendDirection(metrics.TrendDirection)), metrics.TrendPercentage))

	// Consistency
	html.WriteString(fmt.Sprintf(`
                <div class="metric-card">
                    <div class="metric-label">Consistency</div>
                    <div class="metric-value">%.0f%%</div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: %.0f%%"></div>
                    </div>
                </div>`, metrics.Consistency*100, metrics.Consistency*100))

	html.WriteString(`
            </div>`)

	// Weekly velocity chart (simplified)
	if len(metrics.ByWeek) > 0 {
		html.WriteString(`
            <h3>Weekly Velocity</h3>
            <table>
                <thead>
                    <tr><th>Week</th><th>Velocity</th><th>Tasks</th></tr>
                </thead>
                <tbody>`)

		for _, week := range metrics.ByWeek {
			html.WriteString(fmt.Sprintf(`
                <tr>
                    <td>W%d</td>
                    <td>%.1f</td>
                    <td>%d</td>
                </tr>`, week.Number, week.Velocity, week.Tasks))
		}

		html.WriteString(`
                </tbody>
            </table>`)
	}

	html.WriteString(`
        </div>`)

	return html.String()
}

func (ae *analyticsExporter) generateCompletionHTML(metrics entities.CompletionMetrics) string {
	var html strings.Builder

	html.WriteString(`
        <div class="metric-section">
            <h2 class="metric-title">✅ Completion Metrics</h2>
            <div class="metric-grid">`)

	// Total tasks
	html.WriteString(fmt.Sprintf(`
                <div class="metric-card">
                    <div class="metric-label">Total Tasks</div>
                    <div class="metric-value">%d</div>
                </div>`, metrics.TotalTasks))

	// Completion rate
	html.WriteString(fmt.Sprintf(`
                <div class="metric-card">
                    <div class="metric-label">Completion Rate</div>
                    <div class="metric-value">%.0f%%</div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: %.0f%%"></div>
                    </div>
                </div>`, metrics.CompletionRate*100, metrics.CompletionRate*100))

	// Average time
	html.WriteString(fmt.Sprintf(`
                <div class="metric-card">
                    <div class="metric-label">Average Cycle Time</div>
                    <div class="metric-value">%s</div>
                </div>`, ae.formatDuration(metrics.AverageTime)))

	// Quality score
	html.WriteString(fmt.Sprintf(`
                <div class="metric-card">
                    <div class="metric-label">Quality Score</div>
                    <div class="metric-value">%.0f%%</div>
                    <div class="progress-bar">
                        <div class="progress-fill" style="width: %.0f%%"></div>
                    </div>
                </div>`, metrics.QualityScore*100, metrics.QualityScore*100))

	html.WriteString(`
            </div>
        </div>`)

	return html.String()
}

func (ae *analyticsExporter) generateBottlenecksHTML(bottlenecks []entities.Bottleneck) string {
	var html strings.Builder

	html.WriteString(`
        <div class="metric-section">
            <h2 class="metric-title">🚨 Workflow Bottlenecks</h2>`)

	for _, bottleneck := range bottlenecks {
		severityClass := strings.ToLower(string(bottleneck.Severity))
		html.WriteString(fmt.Sprintf(`
            <div class="bottleneck %s">
                <h4>%s - %s</h4>
                <p>%s</p>
                <p><strong>Impact:</strong> %.1f hours lost</p>
                <p><strong>Frequency:</strong> %d occurrences</p>`,
			severityClass,
			strings.ToUpper(string(bottleneck.Severity)),
			bottleneck.Type,
			bottleneck.Description,
			bottleneck.Impact,
			bottleneck.Frequency))

		if len(bottleneck.Suggestions) > 0 {
			html.WriteString(`<p><strong>Suggestions:</strong></p><ul>`)
			for _, suggestion := range bottleneck.Suggestions {
				html.WriteString(fmt.Sprintf(`<li>%s</li>`, suggestion))
			}
			html.WriteString(`</ul>`)
		}

		html.WriteString(`</div>`)
	}

	html.WriteString(`
        </div>`)

	return html.String()
}

func (ae *analyticsExporter) generateTrendsHTML(trends entities.TrendAnalysis) string {
	var html strings.Builder

	html.WriteString(`
        <div class="metric-section">
            <h2 class="metric-title">📈 Trend Analysis</h2>
            <table>
                <thead>
                    <tr><th>Metric</th><th>Trend</th><th>Confidence</th></tr>
                </thead>
                <tbody>`)

	// Add trend rows
	trendData := []struct {
		name  string
		trend entities.Trend
	}{
		{"Productivity", trends.ProductivityTrend},
		{"Velocity", trends.VelocityTrend},
		{"Quality", trends.QualityTrend},
		{"Efficiency", trends.EfficiencyTrend},
	}

	for _, td := range trendData {
		trendClass := ae.getTrendClass(td.trend.Direction)
		trendSymbol := ae.getTrendSymbol(td.trend.Direction)

		html.WriteString(fmt.Sprintf(`
                <tr>
                    <td>%s</td>
                    <td class="%s">%s %s</td>
                    <td>%.0f%%</td>
                </tr>`, td.name, trendClass, trendSymbol, td.trend.Direction, td.trend.Confidence*100))
	}

	html.WriteString(`
                </tbody>
            </table>
        </div>`)

	return html.String()
}

func (ae *analyticsExporter) generateReportHTML(report *entities.ProductivityReport) string {
	// Generate enhanced HTML with insights and recommendations
	html := ae.generateMetricsHTML(&report.Metrics)

	// Add insights section
	if len(report.Insights) > 0 {
		insightsHTML := `
        <div class="metric-section">
            <h2 class="metric-title">💡 Insights</h2>`

		for _, insight := range report.Insights {
			insightsHTML += fmt.Sprintf(`
            <div class="metric-card">
                <h4>%s</h4>
                <p>%s</p>
                <p><strong>Impact:</strong> %.0f%% | <strong>Confidence:</strong> %.0f%%</p>
            </div>`, insight.Title, insight.Description, insight.Impact*100, insight.Confidence*100)
		}

		insightsHTML += `
        </div>`

		// Insert before closing body tag
		html = strings.Replace(html, "</div>\n</body>", insightsHTML+"</div>\n</body>", 1)
	}

	return html
}

// PDF export implementations (simplified - would need PDF library)

func (ae *analyticsExporter) exportPDF(metrics *entities.WorkflowMetrics, filePath string) (string, error) {
	// For now, export as HTML and note that PDF conversion is needed
	htmlPath := strings.Replace(filePath, ".pdf", ".html", 1)
	_, err := ae.exportHTML(metrics, htmlPath)
	if err != nil {
		return "", err
	}

	// Note: In a real implementation, you would use a PDF library like:
	// - github.com/jung-kurt/gofpdf
	// - github.com/go-pdf/fpdf
	// - wkhtmltopdf wrapper

	return htmlPath, fmt.Errorf("PDF export not yet implemented - HTML file created at %s", htmlPath)
}

func (ae *analyticsExporter) exportReportPDF(report *entities.ProductivityReport, filePath string) (string, error) {
	// For now, export as HTML and note that PDF conversion is needed
	htmlPath := strings.Replace(filePath, ".pdf", ".html", 1)
	_, err := ae.exportReportHTML(report, htmlPath)
	if err != nil {
		return "", err
	}

	return htmlPath, fmt.Errorf("PDF export not yet implemented - HTML file created at %s", htmlPath)
}

// Utility methods

func (ae *analyticsExporter) ensureOutputDir() error {
	return os.MkdirAll(ae.config.OutputDir, 0750)
}

func (ae *analyticsExporter) generateFilename(base string, format entities.ExportFormat) string {
	timestamp := ""
	if ae.config.TimestampFiles {
		timestamp = "_" + time.Now().Format("20060102_150405")
	}

	extension := string(format)
	if format == entities.ExportFormatHTML {
		extension = "html"
	}

	return fmt.Sprintf("%s%s.%s", base, timestamp, extension)
}

func (ae *analyticsExporter) formatDuration(d time.Duration) string {
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

func (ae *analyticsExporter) getTrendClass(direction entities.TrendDirection) string {
	switch direction {
	case entities.TrendDirectionUp:
		return "trend-up"
	case entities.TrendDirectionDown:
		return "trend-down"
	default:
		return "trend-stable"
	}
}

func (ae *analyticsExporter) getTrendSymbol(direction entities.TrendDirection) string {
	switch direction {
	case entities.TrendDirectionUp:
		return "↗"
	case entities.TrendDirectionDown:
		return "↘"
	case entities.TrendDirectionVolatile:
		return "↕"
	default:
		return "→"
	}
}
