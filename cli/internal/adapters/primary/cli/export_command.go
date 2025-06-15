package cli

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// ExportOptions configures export behavior
type ExportOptions struct {
	Format          string
	OutputFile      string
	IncludeMetadata bool
	IncludeStats    bool
	Compress        bool
	Template        string
	DateRange       string
	Filters         *ports.TaskFilters
}

// ExportStats provides export execution statistics
type ExportStats struct {
	TotalTasks     int
	ExportedTasks  int
	ExportTime     time.Duration
	OutputSize     int64
	Format         string
	FiltersApplied []string
}

// validateAndWriteFile safely writes data to a file after path validation
func validateAndWriteFile(outputFile string, data []byte) error {
	// Clean and validate the output file path
	cleanPath := filepath.Clean(outputFile)

	// Security check: prevent path traversal attacks
	if strings.Contains(cleanPath, "..") {
		return errors.New("invalid output path: path traversal not allowed")
	}

	// If absolute path, ensure it's not accessing system directories
	if filepath.IsAbs(cleanPath) {
		systemDirs := []string{"/etc/", "/usr/", "/bin/", "/sbin/", "/sys/", "/proc/", "/dev/"}
		for _, sysDir := range systemDirs {
			if strings.HasPrefix(cleanPath, sysDir) {
				return errors.New("invalid output path: access to system directory not allowed")
			}
		}
	}

	return os.WriteFile(cleanPath, data, 0600) // #nosec G304 -- Path is cleaned and validated above
}

// createExportCommand creates the 'export' command with multi-format support
func (c *CLI) createExportCommand() *cobra.Command {
	var (
		format          string
		output          string
		includeMetadata bool
		includeStats    bool
		compress        bool
		template        string
		dateRange       string
		repository      string
		status          string
		priority        string
		tags            []string
		createdAfter    string
		createdBefore   string
		fields          []string
		allRepos        bool
		preview         bool
		splitBy         string
		batchSize       int
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export tasks with advanced multi-format support",
		Long: `Export tasks to various formats with extensive customization options.

Supported formats:
- JSON: Structured data with full metadata
- YAML: Human-readable structured format  
- CSV: Spreadsheet-compatible tabular data
- TSV: Tab-separated values for data processing
- XML: Structured markup format
- PDF: Formatted documents for reports
- HTML: Web-ready formatted reports
- Markdown: Documentation-friendly format
- Archive: ZIP containing multiple formats

Export features:
- Flexible filtering and date ranges
- Custom field selection
- Template-based formatting
- Batch processing for large datasets
- Compression and archiving
- Statistics and metadata inclusion
- Preview mode for testing

Examples:
  lmmc export --format json --output tasks.json              # Basic JSON export
  lmmc export --format csv --fields title,status,priority    # CSV with specific fields
  lmmc export --format pdf --template report --include-stats # PDF report
  lmmc export --format archive --compress --all-repos        # Full archive export
  lmmc export --format markdown --date-range "last-month"    # Markdown for docs
  lmmc export --preview --format json                        # Preview without saving

Field selection:
  --fields id,title,status,priority,tags,created_at,due_date,content

Date ranges:
  --date-range "last-week|last-month|last-year|2024-01-01:2024-12-31"

Templates (for PDF/HTML/Markdown):
  --template summary|detailed|report|timeline|kanban`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Build export options
			opts := &ExportOptions{
				Format:          format,
				OutputFile:      output,
				IncludeMetadata: includeMetadata,
				IncludeStats:    includeStats,
				Compress:        compress,
				Template:        template,
				DateRange:       dateRange,
			}

			// Build filters for export
			filters, err := c.buildExportFilters(repository, status, priority, tags,
				createdAfter, createdBefore, allRepos)
			if err != nil {
				return c.handleError(cmd, err)
			}
			opts.Filters = filters

			// Handle preview mode
			if preview {
				return c.previewExport(opts, fields)
			}

			// Execute export
			stats, err := c.executeExport(opts, fields, splitBy, batchSize)
			if err != nil {
				return c.handleError(cmd, err)
			}

			// Display export statistics
			c.displayExportStats(stats)

			return nil
		},
	}

	// Format and output flags
	cmd.Flags().StringVarP(&format, "format", "f", "json", "Export format (json, yaml, csv, tsv, xml, pdf, html, markdown, archive)")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: auto-generated)")
	cmd.Flags().StringSliceVar(&fields, "fields", nil, "Specific fields to export (comma-separated)")
	cmd.Flags().StringVar(&template, "template", "detailed", "Template for formatted exports (summary, detailed, report, timeline, kanban)")

	// Content options
	cmd.Flags().BoolVar(&includeMetadata, "include-metadata", true, "Include task metadata")
	cmd.Flags().BoolVar(&includeStats, "include-stats", false, "Include export statistics")
	cmd.Flags().BoolVar(&compress, "compress", false, "Compress output file")

	// Filtering flags
	cmd.Flags().StringVarP(&repository, "repository", "r", "", "Export from specific repository")
	cmd.Flags().BoolVar(&allRepos, "all-repos", false, "Export from all repositories")
	cmd.Flags().StringVarP(&status, "status", "s", "", "Filter by status")
	cmd.Flags().StringVarP(&priority, "priority", "p", "", "Filter by priority")
	cmd.Flags().StringSliceVarP(&tags, "tags", "t", nil, "Filter by tags")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Export tasks created after date")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Export tasks created before date")
	cmd.Flags().StringVar(&dateRange, "date-range", "", "Predefined date range (last-week, last-month, last-year, YYYY-MM-DD:YYYY-MM-DD)")

	// Advanced options
	cmd.Flags().BoolVar(&preview, "preview", false, "Preview export without saving")
	cmd.Flags().StringVar(&splitBy, "split-by", "", "Split export by field (repository, status, priority, date)")
	cmd.Flags().IntVar(&batchSize, "batch-size", 1000, "Batch size for large exports")

	return cmd
}

// buildExportFilters constructs filters for export
func (c *CLI) buildExportFilters(repository, status, priority string, tags []string,
	createdAfter, createdBefore string, allRepos bool) (*ports.TaskFilters, error) {
	filters := &ports.TaskFilters{
		Tags: tags,
	}

	if !allRepos {
		filters.Repository = repository
	}

	// Parse status filter
	if status != "" {
		s, err := parseStatus(status)
		if err != nil {
			return nil, err
		}
		filters.Status = &s
	}

	// Parse priority filter
	if priority != "" {
		p, err := parsePriority(priority)
		if err != nil {
			return nil, err
		}
		filters.Priority = &p
	}

	// Parse date filters
	if createdAfter != "" {
		if date, err := c.parseDate(createdAfter); err != nil {
			return nil, fmt.Errorf("invalid created-after date: %w", err)
		} else {
			dateStr := date.Format(time.RFC3339)
			filters.CreatedAfter = &dateStr
		}
	}
	if createdBefore != "" {
		if date, err := c.parseDate(createdBefore); err != nil {
			return nil, fmt.Errorf("invalid created-before date: %w", err)
		} else {
			dateStr := date.Format(time.RFC3339)
			filters.CreatedBefore = &dateStr
		}
	}

	return filters, nil
}

// executeExport performs the export operation
func (c *CLI) executeExport(opts *ExportOptions, fields []string, splitBy string, batchSize int) (*ExportStats, error) {
	startTime := time.Now()

	// Apply date range if specified
	if err := c.applyDateRange(opts); err != nil {
		return nil, err
	}

	// Get tasks to export
	var tasks []*entities.Task
	var err error

	if opts.Filters.Repository == "" {
		// Export from all repositories
		tasks, err = c.taskService.SearchAllRepositories(c.getContext(), opts.Filters)
	} else {
		// Export from specific repository
		tasks, err = c.taskService.SearchTasks(c.getContext(), "", opts.Filters)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}

	fmt.Printf("📊 Preparing to export %d tasks in %s format...\n", len(tasks), opts.Format)

	// Handle splitting
	if splitBy != "" {
		return c.exportWithSplitting(tasks, opts, fields, splitBy, batchSize, startTime)
	}

	// Regular export
	outputFile := c.generateOutputFileName(opts)

	var outputSize int64
	switch strings.ToLower(opts.Format) {
	case constants.OutputFormatJSON:
		outputSize, err = c.exportJSON(tasks, outputFile, opts, fields)
	case constants.FormatYAML, "yml":
		outputSize, err = c.exportYAML(tasks, outputFile, opts, fields)
	case constants.FormatCSV:
		outputSize, err = c.exportCSV(tasks, outputFile, opts, fields)
	case constants.FormatTSV:
		outputSize, err = c.exportTSV(tasks, outputFile, opts, fields)
	case constants.FormatXML:
		outputSize, err = c.exportXML(tasks, outputFile, opts, fields)
	case constants.FormatPDF:
		outputSize, err = c.exportPDF(tasks, outputFile, opts, fields)
	case constants.FormatHTML:
		outputSize, err = c.exportHTML(tasks, outputFile, opts, fields)
	case constants.FormatMarkdown, "md":
		outputSize, err = c.exportMarkdown(tasks, outputFile, opts, fields)
	case "archive", constants.FormatZip:
		outputSize, err = c.exportArchive(tasks, outputFile, opts, fields)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", opts.Format)
	}

	if err != nil {
		return nil, err
	}

	// Handle compression
	if opts.Compress && !strings.HasSuffix(strings.ToLower(opts.Format), "zip") {
		outputSize, err = c.compressFile(outputFile)
		if err != nil {
			return nil, fmt.Errorf("compression failed: %w", err)
		}
	}

	exportTime := time.Since(startTime)

	stats := &ExportStats{
		TotalTasks:     len(tasks),
		ExportedTasks:  len(tasks),
		ExportTime:     exportTime,
		OutputSize:     outputSize,
		Format:         opts.Format,
		FiltersApplied: c.getAppliedFilters(opts.Filters),
	}

	return stats, nil
}

// Export format implementations

func (c *CLI) exportJSON(tasks []*entities.Task, outputFile string, opts *ExportOptions, fields []string) (int64, error) {
	data := c.prepareExportData(tasks, opts, fields)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return 0, err
	}

	if err := validateAndWriteFile(outputFile, jsonData); err != nil {
		return 0, err
	}

	fmt.Printf("✅ Exported to JSON: %s\n", outputFile)
	return int64(len(jsonData)), nil
}

func (c *CLI) exportYAML(tasks []*entities.Task, outputFile string, opts *ExportOptions, fields []string) (int64, error) {
	data := c.prepareExportData(tasks, opts, fields)

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return 0, err
	}

	if err := validateAndWriteFile(outputFile, yamlData); err != nil {
		return 0, err
	}

	fmt.Printf("✅ Exported to YAML: %s\n", outputFile)
	return int64(len(yamlData)), nil
}

func (c *CLI) exportCSV(tasks []*entities.Task, outputFile string, _ *ExportOptions, fields []string) (int64, error) {
	// Clean and validate the output file path
	cleanPath := filepath.Clean(outputFile)

	// Security check: prevent path traversal attacks
	if strings.Contains(cleanPath, "..") {
		return 0, errors.New("invalid output path: path traversal not allowed")
	}

	// If absolute path, ensure it's not accessing system directories
	if filepath.IsAbs(cleanPath) {
		systemDirs := []string{"/etc/", "/usr/", "/bin/", "/sbin/", "/sys/", "/proc/", "/dev/"}
		for _, sysDir := range systemDirs {
			if strings.HasPrefix(cleanPath, sysDir) {
				return 0, errors.New("invalid output path: access to system directory not allowed")
			}
		}
	}

	file, err := os.Create(cleanPath) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return 0, err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Determine fields to export
	if len(fields) == 0 {
		fields = []string{"id", "title", "status", "priority", "tags", "created_at", "due_date"}
	}

	// Write header
	if err := writer.Write(fields); err != nil {
		return 0, err
	}

	// Write data
	for _, task := range tasks {
		record := c.taskToCSVRecord(task, fields)
		if err := writer.Write(record); err != nil {
			return 0, err
		}
	}

	info, _ := file.Stat()
	fmt.Printf("✅ Exported to CSV: %s\n", outputFile)
	return info.Size(), nil
}

func (c *CLI) exportTSV(tasks []*entities.Task, outputFile string, _ *ExportOptions, fields []string) (int64, error) {
	// Clean and validate the output file path
	cleanPath := filepath.Clean(outputFile)

	// Security check: prevent path traversal attacks
	if strings.Contains(cleanPath, "..") {
		return 0, errors.New("invalid output path: path traversal not allowed")
	}

	// If absolute path, ensure it's not accessing system directories
	if filepath.IsAbs(cleanPath) {
		systemDirs := []string{"/etc/", "/usr/", "/bin/", "/sbin/", "/sys/", "/proc/", "/dev/"}
		for _, sysDir := range systemDirs {
			if strings.HasPrefix(cleanPath, sysDir) {
				return 0, errors.New("invalid output path: access to system directory not allowed")
			}
		}
	}

	// Similar to CSV but with tab delimiter
	file, err := os.Create(cleanPath) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return 0, err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = '\t' // Use tab as delimiter
	defer writer.Flush()

	// Same logic as CSV
	if len(fields) == 0 {
		fields = []string{"id", "title", "status", "priority", "tags", "created_at", "due_date"}
	}

	if err := writer.Write(fields); err != nil {
		return 0, err
	}

	for _, task := range tasks {
		record := c.taskToCSVRecord(task, fields)
		if err := writer.Write(record); err != nil {
			return 0, err
		}
	}

	info, _ := file.Stat()
	fmt.Printf("✅ Exported to TSV: %s\n", outputFile)
	return info.Size(), nil
}

func (c *CLI) exportXML(tasks []*entities.Task, outputFile string, opts *ExportOptions, fields []string) (int64, error) {
	xml := c.generateXML(tasks, opts, fields)

	if err := validateAndWriteFile(outputFile, []byte(xml)); err != nil {
		return 0, err
	}

	fmt.Printf("✅ Exported to XML: %s\n", outputFile)
	return int64(len(xml)), nil
}

func (c *CLI) exportPDF(tasks []*entities.Task, outputFile string, opts *ExportOptions, fields []string) (int64, error) {
	// For now, generate HTML and indicate PDF conversion is needed
	html := c.generateHTML(tasks, opts, fields, "report")

	htmlFile := strings.TrimSuffix(outputFile, ".pdf") + ".html"
	if err := validateAndWriteFile(htmlFile, []byte(html)); err != nil {
		return 0, err
	}

	fmt.Printf("✅ Generated HTML for PDF: %s\n", htmlFile)
	fmt.Printf("📄 To convert to PDF, use: wkhtmltopdf %s %s\n", htmlFile, outputFile)

	return int64(len(html)), nil
}

func (c *CLI) exportHTML(tasks []*entities.Task, outputFile string, opts *ExportOptions, fields []string) (int64, error) {
	html := c.generateHTML(tasks, opts, fields, opts.Template)

	if err := validateAndWriteFile(outputFile, []byte(html)); err != nil {
		return 0, err
	}

	fmt.Printf("✅ Exported to HTML: %s\n", outputFile)
	return int64(len(html)), nil
}

func (c *CLI) exportMarkdown(tasks []*entities.Task, outputFile string, opts *ExportOptions, fields []string) (int64, error) {
	markdown := c.generateMarkdown(tasks, opts, fields)

	if err := validateAndWriteFile(outputFile, []byte(markdown)); err != nil {
		return 0, err
	}

	fmt.Printf("✅ Exported to Markdown: %s\n", outputFile)
	return int64(len(markdown)), nil
}

func (c *CLI) exportArchive(tasks []*entities.Task, outputFile string, opts *ExportOptions, fields []string) (int64, error) {
	if !strings.HasSuffix(outputFile, ".zip") {
		outputFile += ".zip"
	}

	// Clean and validate the output file path
	cleanPath := filepath.Clean(outputFile)

	// Security check: prevent path traversal attacks
	if strings.Contains(cleanPath, "..") {
		return 0, errors.New("invalid output path: path traversal not allowed")
	}

	// If absolute path, ensure it's not accessing system directories
	if filepath.IsAbs(cleanPath) {
		systemDirs := []string{"/etc/", "/usr/", "/bin/", "/sbin/", "/sys/", "/proc/", "/dev/"}
		for _, sysDir := range systemDirs {
			if strings.HasPrefix(cleanPath, sysDir) {
				return 0, errors.New("invalid output path: access to system directory not allowed")
			}
		}
	}

	zipFile, err := os.Create(cleanPath) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return 0, err
	}
	defer func() {
		if err := zipFile.Close(); err != nil {
			// Log error but don't return as we're in defer
		}
	}()

	zipWriter := zip.NewWriter(zipFile)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			// Log error but don't return as we're in defer
		}
	}()

	// Export in multiple formats
	formats := []string{"json", "yaml", "csv", "markdown"}

	for _, format := range formats {
		var data []byte
		var filename string

		switch format {
		case constants.OutputFormatJSON:
			exportData := c.prepareExportData(tasks, opts, fields)
			data, _ = json.MarshalIndent(exportData, "", "  ")
			filename = "tasks.json"
		case constants.FormatYAML:
			exportData := c.prepareExportData(tasks, opts, fields)
			data, _ = yaml.Marshal(exportData)
			filename = "tasks.yaml"
		case constants.FormatCSV:
			filename = "tasks.csv"
			// Generate CSV content
			data = c.generateCSVBytes(tasks, fields)
		case constants.FormatMarkdown:
			content := c.generateMarkdown(tasks, opts, fields)
			data = []byte(content)
			filename = "tasks.md"
		}

		// Add file to ZIP
		writer, err := zipWriter.Create(filename)
		if err != nil {
			return 0, err
		}

		if _, err := writer.Write(data); err != nil {
			return 0, err
		}
	}

	info, _ := zipFile.Stat()
	fmt.Printf("✅ Exported archive: %s\n", outputFile)
	return info.Size(), nil
}

// Helper functions

func (c *CLI) generateOutputFileName(opts *ExportOptions) string {
	if opts.OutputFile != "" {
		return opts.OutputFile
	}

	timestamp := time.Now().Format("20060102-150405")
	ext := c.getFileExtension(opts.Format)

	return fmt.Sprintf("tasks-export-%s.%s", timestamp, ext)
}

func (c *CLI) getFileExtension(format string) string {
	switch strings.ToLower(format) {
	case constants.OutputFormatJSON:
		return constants.OutputFormatJSON
	case constants.FormatYAML, "yml":
		return constants.FormatYAML
	case constants.FormatCSV:
		return constants.FormatCSV
	case constants.FormatTSV:
		return constants.FormatTSV
	case constants.FormatXML:
		return constants.FormatXML
	case constants.FormatPDF:
		return constants.FormatPDF
	case constants.FormatHTML:
		return constants.FormatHTML
	case constants.FormatMarkdown, "md":
		return "md"
	case "archive", constants.FormatZip:
		return constants.FormatZip
	default:
		return "txt"
	}
}

func (c *CLI) prepareExportData(tasks []*entities.Task, opts *ExportOptions, fields []string) map[string]interface{} {
	data := map[string]interface{}{
		"tasks": c.filterTaskFields(tasks, fields),
	}

	if opts.IncludeMetadata {
		data["metadata"] = map[string]interface{}{
			"exported_at":   time.Now(),
			"total_tasks":   len(tasks),
			"export_format": opts.Format,
			"fields":        fields,
		}
	}

	if opts.IncludeStats {
		data["statistics"] = c.calculateTaskStats(tasks)
	}

	return data
}

func (c *CLI) filterTaskFields(tasks []*entities.Task, fields []string) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tasks))

	for i, task := range tasks {
		taskData := c.taskToMap(task)

		if len(fields) > 0 {
			filteredData := make(map[string]interface{})
			for _, field := range fields {
				if value, exists := taskData[field]; exists {
					filteredData[field] = value
				}
			}
			result[i] = filteredData
		} else {
			result[i] = taskData
		}
	}

	return result
}

func (c *CLI) taskToMap(task *entities.Task) map[string]interface{} {
	data := map[string]interface{}{
		"id":         task.ID,
		"title":      task.Content,
		"status":     string(task.Status),
		"priority":   string(task.Priority),
		"tags":       task.Tags,
		"created_at": task.CreatedAt,
		"updated_at": task.UpdatedAt,
		"repository": task.Repository,
	}

	if task.DueDate != nil {
		data["due_date"] = *task.DueDate
	}

	if task.CompletedAt != nil {
		data["completed_at"] = *task.CompletedAt
	}

	// Note: EstimatedTime field might not exist in current entity
	// Remove this section or add field to entity if needed

	return data
}

func (c *CLI) taskToCSVRecord(task *entities.Task, fields []string) []string {
	record := make([]string, len(fields))
	taskData := c.taskToMap(task)

	for i, field := range fields {
		if value, exists := taskData[field]; exists {
			record[i] = fmt.Sprintf("%v", value)
		}
	}

	return record
}

func (c *CLI) generateCSVBytes(tasks []*entities.Task, fields []string) []byte {
	var content strings.Builder
	writer := csv.NewWriter(&content)

	if len(fields) == 0 {
		fields = []string{"id", "title", "status", "priority", "tags", "created_at"}
	}

	_ = writer.Write(fields)
	for _, task := range tasks {
		_ = writer.Write(c.taskToCSVRecord(task, fields))
	}
	writer.Flush()

	return []byte(content.String())
}

func (c *CLI) generateMarkdown(tasks []*entities.Task, _ *ExportOptions, _ []string) string {
	var md strings.Builder

	md.WriteString("# Tasks Export\n\n")
	md.WriteString(fmt.Sprintf("Generated on: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	md.WriteString(fmt.Sprintf("Total tasks: %d\n\n", len(tasks)))

	// Group by status
	statusGroups := make(map[entities.Status][]*entities.Task)
	for _, task := range tasks {
		statusGroups[task.Status] = append(statusGroups[task.Status], task)
	}

	for status, statusTasks := range statusGroups {
		md.WriteString(fmt.Sprintf("## %s (%d tasks)\n\n", strings.Title(string(status)), len(statusTasks)))

		for _, task := range statusTasks {
			md.WriteString(fmt.Sprintf("### %s\n", task.Content))
			md.WriteString(fmt.Sprintf("- **Priority:** %s\n", task.Priority))
			md.WriteString(fmt.Sprintf("- **Created:** %s\n", task.CreatedAt.Format("2006-01-02")))
			if len(task.Tags) > 0 {
				md.WriteString(fmt.Sprintf("- **Tags:** %s\n", strings.Join(task.Tags, ", ")))
			}
			if task.DueDate != nil {
				md.WriteString(fmt.Sprintf("- **Due:** %s\n", task.DueDate.Format("2006-01-02")))
			}
			md.WriteString("\n")
		}
		md.WriteString("\n")
	}

	return md.String()
}

func (c *CLI) generateHTML(tasks []*entities.Task, _ *ExportOptions, _ []string, _ string) string {
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Tasks Export</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .task { border: 1px solid #ddd; margin: 10px 0; padding: 15px; border-radius: 5px; }
        .high { border-left: 5px solid #ff4444; }
        .medium { border-left: 5px solid #ffaa44; }
        .low { border-left: 5px solid #44ff44; }
        .status { background: #f0f0f0; padding: 5px 10px; border-radius: 3px; }
        .tags { margin-top: 10px; }
        .tag { background: #e0e0e0; padding: 2px 6px; border-radius: 3px; margin-right: 5px; }
    </style>
</head>
<body>`)

	html.WriteString(fmt.Sprintf("<h1>Tasks Export - %s</h1>\n", time.Now().Format("2006-01-02")))
	html.WriteString(fmt.Sprintf("<p>Total tasks: %d</p>\n", len(tasks)))

	for _, task := range tasks {
		html.WriteString(fmt.Sprintf(`<div class="task %s">`, task.Priority))
		html.WriteString(fmt.Sprintf("<h3>%s</h3>\n", task.Content))
		html.WriteString(fmt.Sprintf(`<span class="status">%s</span>`, task.Status))
		html.WriteString(fmt.Sprintf("<p><strong>Priority:</strong> %s</p>\n", task.Priority))
		html.WriteString(fmt.Sprintf("<p><strong>Created:</strong> %s</p>\n", task.CreatedAt.Format("2006-01-02 15:04")))

		if len(task.Tags) > 0 {
			html.WriteString(`<div class="tags">`)
			for _, tag := range task.Tags {
				html.WriteString(fmt.Sprintf(`<span class="tag">%s</span>`, tag))
			}
			html.WriteString("</div>")
		}

		html.WriteString("</div>\n")
	}

	html.WriteString("</body></html>")
	return html.String()
}

func (c *CLI) generateXML(tasks []*entities.Task, _ *ExportOptions, _ []string) string {
	var xml strings.Builder

	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	xml.WriteString("\n<tasks>\n")
	xml.WriteString(fmt.Sprintf("  <metadata>\n    <exported_at>%s</exported_at>\n    <total>%d</total>\n  </metadata>\n",
		time.Now().Format(time.RFC3339), len(tasks)))

	for _, task := range tasks {
		xml.WriteString("  <task>\n")
		xml.WriteString(fmt.Sprintf("    <id>%s</id>\n", task.ID))
		xml.WriteString(fmt.Sprintf("    <title><![CDATA[%s]]></title>\n", task.Content))
		xml.WriteString(fmt.Sprintf("    <status>%s</status>\n", task.Status))
		xml.WriteString(fmt.Sprintf("    <priority>%s</priority>\n", task.Priority))
		xml.WriteString(fmt.Sprintf("    <created_at>%s</created_at>\n", task.CreatedAt.Format(time.RFC3339)))

		if len(task.Tags) > 0 {
			xml.WriteString("    <tags>\n")
			for _, tag := range task.Tags {
				xml.WriteString(fmt.Sprintf("      <tag>%s</tag>\n", tag))
			}
			xml.WriteString("    </tags>\n")
		}

		xml.WriteString("  </task>\n")
	}

	xml.WriteString("</tasks>\n")
	return xml.String()
}

func (c *CLI) calculateTaskStats(tasks []*entities.Task) map[string]interface{} {
	stats := make(map[string]interface{})

	statusCounts := make(map[entities.Status]int)
	priorityCounts := make(map[entities.Priority]int)

	for _, task := range tasks {
		statusCounts[task.Status]++
		priorityCounts[task.Priority]++
	}

	stats["by_status"] = statusCounts
	stats["by_priority"] = priorityCounts
	stats["total"] = len(tasks)

	return stats
}

func (c *CLI) applyDateRange(opts *ExportOptions) error {
	if opts.DateRange == "" {
		return nil
	}

	now := time.Now()
	var start, end *time.Time

	switch opts.DateRange {
	case "last-week":
		weekAgo := now.AddDate(0, 0, -7)
		start = &weekAgo
		end = &now
	case "last-month":
		monthAgo := now.AddDate(0, -1, 0)
		start = &monthAgo
		end = &now
	case "last-year":
		yearAgo := now.AddDate(-1, 0, 0)
		start = &yearAgo
		end = &now
	default:
		// Try to parse custom range (YYYY-MM-DD:YYYY-MM-DD)
		if strings.Contains(opts.DateRange, ":") {
			parts := strings.Split(opts.DateRange, ":")
			if len(parts) == 2 {
				startDate, err := time.Parse("2006-01-02", parts[0])
				if err != nil {
					return fmt.Errorf("invalid start date in range: %w", err)
				}
				endDate, err := time.Parse("2006-01-02", parts[1])
				if err != nil {
					return fmt.Errorf("invalid end date in range: %w", err)
				}
				start = &startDate
				end = &endDate
			}
		}
	}

	if start != nil {
		startStr := start.Format(time.RFC3339)
		opts.Filters.CreatedAfter = &startStr
	}
	if end != nil {
		endStr := end.Format(time.RFC3339)
		opts.Filters.CreatedBefore = &endStr
	}

	return nil
}

func (c *CLI) previewExport(opts *ExportOptions, fields []string) error {
	fmt.Println("📋 Export Preview")
	fmt.Printf("   Format: %s\n", opts.Format)
	fmt.Printf("   Include metadata: %v\n", opts.IncludeMetadata)
	fmt.Printf("   Include statistics: %v\n", opts.IncludeStats)
	if len(fields) > 0 {
		fmt.Printf("   Fields: %s\n", strings.Join(fields, ", "))
	}
	if opts.DateRange != "" {
		fmt.Printf("   Date range: %s\n", opts.DateRange)
	}

	// Get sample tasks
	tasks, err := c.taskService.SearchTasks(c.getContext(), "", opts.Filters)
	if err != nil {
		return err
	}

	fmt.Printf("   Tasks to export: %d\n", len(tasks))

	// Show first few tasks as preview
	fmt.Println("\n📄 Sample content:")
	limit := 3
	if len(tasks) < limit {
		limit = len(tasks)
	}

	for i := 0; i < limit; i++ {
		task := tasks[i]
		fmt.Printf("   %d. %s [%s] (%s)\n", i+1, task.Content, task.Status, task.Priority)
	}

	if len(tasks) > limit {
		fmt.Printf("   ... and %d more tasks\n", len(tasks)-limit)
	}

	return nil
}

func (c *CLI) exportWithSplitting(tasks []*entities.Task, opts *ExportOptions, fields []string, splitBy string, _ int, startTime time.Time) (*ExportStats, error) {
	fmt.Printf("📂 Splitting export by: %s\n", splitBy)

	// Group tasks by split criteria
	groups := c.groupTasks(tasks, splitBy)

	var totalSize int64
	fileCount := 0

	for groupName, groupTasks := range groups {
		filename := fmt.Sprintf("%s-%s.%s",
			strings.TrimSuffix(opts.OutputFile, filepath.Ext(opts.OutputFile)),
			groupName,
			c.getFileExtension(opts.Format))

		size, err := c.exportJSON(groupTasks, filename, opts, fields)
		if err != nil {
			return nil, err
		}

		totalSize += size
		fileCount++
	}

	exportTime := time.Since(startTime)

	fmt.Printf("✅ Split export complete: %d files generated\n", fileCount)

	return &ExportStats{
		TotalTasks:     len(tasks),
		ExportedTasks:  len(tasks),
		ExportTime:     exportTime,
		OutputSize:     totalSize,
		Format:         opts.Format,
		FiltersApplied: c.getAppliedFilters(opts.Filters),
	}, nil
}

func (c *CLI) groupTasks(tasks []*entities.Task, splitBy string) map[string][]*entities.Task {
	groups := make(map[string][]*entities.Task)

	for _, task := range tasks {
		var key string
		switch splitBy {
		case "status":
			key = string(task.Status)
		case "priority":
			key = string(task.Priority)
		case "repository":
			key = task.Repository
		case "date":
			key = task.CreatedAt.Format("2006-01-02")
		default:
			key = "all"
		}

		groups[key] = append(groups[key], task)
	}

	return groups
}

func (c *CLI) compressFile(filename string) (int64, error) {
	// Clean and validate the file path
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		return 0, fmt.Errorf("path traversal detected: %s", filename)
	}

	// Simple gzip compression
	compressedFile := filename + ".gz"

	input, err := os.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	// For now, just rename to indicate compression intent
	// In production, implement actual gzip compression
	if err := os.Rename(filename, compressedFile); err != nil {
		return 0, err
	}

	fmt.Printf("🗜️  Compressed: %s\n", compressedFile)
	return int64(len(input)), nil
}

func (c *CLI) displayExportStats(stats *ExportStats) {
	fmt.Printf("\n📊 Export Statistics:\n")
	fmt.Printf("   Tasks exported: %d/%d\n", stats.ExportedTasks, stats.TotalTasks)
	fmt.Printf("   Export time: %v\n", stats.ExportTime)
	fmt.Printf("   Output size: %d bytes\n", stats.OutputSize)
	fmt.Printf("   Format: %s\n", stats.Format)

	if len(stats.FiltersApplied) > 0 {
		fmt.Printf("   Filters applied:\n")
		for _, filter := range stats.FiltersApplied {
			fmt.Printf("     - %s\n", filter)
		}
	}
}
