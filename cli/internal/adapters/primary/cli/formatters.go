package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// OutputFormatter defines the interface for formatting task output
type OutputFormatter interface {
	FormatTask(task *entities.Task) error
	FormatTaskList(tasks []*entities.Task) error
	FormatStats(stats *ports.RepositoryStats) error
	FormatError(err error) error
	FormatDocument(doc interface{}) error
}

// TableFormatter formats output as ASCII tables
type TableFormatter struct {
	writer io.Writer
}

// NewTableFormatter creates a new table formatter
func NewTableFormatter(w io.Writer) OutputFormatter {
	return &TableFormatter{writer: w}
}

// FormatTask formats a single task as a table
func (f *TableFormatter) FormatTask(task *entities.Task) error {
	table := tablewriter.NewWriter(f.writer)
	table.Header("Field", "Value")

	_ = table.Append([]string{"ID", truncateID(task.ID)})
	_ = table.Append([]string{"Content", task.Content})
	_ = table.Append([]string{"Status", string(task.Status)})
	_ = table.Append([]string{"Priority", string(task.Priority)})
	_ = table.Append([]string{"Repository", task.Repository})
	_ = table.Append([]string{"Created", task.CreatedAt.Format("2006-01-02 15:04")})
	_ = table.Append([]string{"Updated", task.UpdatedAt.Format("2006-01-02 15:04")})

	if len(task.Tags) > 0 {
		_ = table.Append([]string{"Tags", strings.Join(task.Tags, ", ")})
	}

	if task.EstimatedMins > 0 {
		_ = table.Append([]string{"Estimated", fmt.Sprintf("%d mins", task.EstimatedMins)})
	}

	if task.ActualMins > 0 {
		_ = table.Append([]string{"Actual", fmt.Sprintf("%d mins", task.ActualMins)})
	}

	return table.Render()
}

// FormatTaskList formats multiple tasks as a table
func (f *TableFormatter) FormatTaskList(tasks []*entities.Task) error {
	if len(tasks) == 0 {
		_, _ = fmt.Fprintln(f.writer, "No tasks found.")
		return nil
	}

	table := tablewriter.NewWriter(f.writer)
	table.Header("#", "ID", "Content", "Status", "Priority", "Tags", "Created")

	for i, task := range tasks {
		_ = table.Append([]string{
			strconv.Itoa(i + 1),
			truncateID(task.ID),
			truncateContent(task.Content, 50),
			string(task.Status),
			string(task.Priority),
			strings.Join(task.Tags, ","),
			task.CreatedAt.Format("01/02"),
		})
	}

	if err := table.Render(); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(f.writer, "\nTotal: %d tasks (use the # number to delete)\n", len(tasks))
	return nil
}

// FormatStats formats repository statistics as a table
func (f *TableFormatter) FormatStats(stats *ports.RepositoryStats) error {
	table := tablewriter.NewWriter(f.writer)
	table.Header("Metric", "Value")

	_ = table.Append([]string{"Repository", stats.Repository})
	_ = table.Append([]string{"Total Tasks", strconv.Itoa(stats.TotalTasks)})
	_ = table.Append([]string{"Pending", strconv.Itoa(stats.PendingTasks)})
	_ = table.Append([]string{"In Progress", strconv.Itoa(stats.InProgressTasks)})
	_ = table.Append([]string{"Completed", strconv.Itoa(stats.CompletedTasks)})
	_ = table.Append([]string{"Cancelled", strconv.Itoa(stats.CancelledTasks)})
	_ = table.Append([]string{"Total Tags", strconv.Itoa(stats.TotalTags)})

	if stats.LastActivity != "" {
		_ = table.Append([]string{"Last Activity", stats.LastActivity})
	}

	return table.Render()
}

// FormatError formats an error message
func (f *TableFormatter) FormatError(err error) error {
	_, _ = fmt.Fprintf(f.writer, "Error: %s\n", err.Error())
	return nil
}

// FormatDocument formats a document as a table
func (f *TableFormatter) FormatDocument(doc interface{}) error {
	// For table format, just output basic info
	_, _ = fmt.Fprintf(f.writer, "Document: %+v\n", doc)
	return nil
}

// JSONFormatter formats output as JSON
type JSONFormatter struct {
	writer io.Writer
	pretty bool
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(w io.Writer, pretty bool) OutputFormatter {
	return &JSONFormatter{writer: w, pretty: pretty}
}

// FormatTask formats a single task as JSON
func (f *JSONFormatter) FormatTask(task *entities.Task) error {
	var data []byte
	var err error

	if f.pretty {
		data, err = json.MarshalIndent(task, "", "  ")
	} else {
		data, err = json.Marshal(task)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	_, _ = fmt.Fprintln(f.writer, string(data))
	return nil
}

// FormatTaskList formats multiple tasks as JSON
func (f *JSONFormatter) FormatTaskList(tasks []*entities.Task) error {
	var data []byte
	var err error

	result := map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	}

	if f.pretty {
		data, err = json.MarshalIndent(result, "", "  ")
	} else {
		data, err = json.Marshal(result)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	_, _ = fmt.Fprintln(f.writer, string(data))
	return nil
}

// FormatStats formats repository statistics as JSON
func (f *JSONFormatter) FormatStats(stats *ports.RepositoryStats) error {
	var data []byte
	var err error

	if f.pretty {
		data, err = json.MarshalIndent(stats, "", "  ")
	} else {
		data, err = json.Marshal(stats)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	_, _ = fmt.Fprintln(f.writer, string(data))
	return nil
}

// FormatError formats an error as JSON
func (f *JSONFormatter) FormatError(err error) error {
	result := map[string]string{
		"error": err.Error(),
	}

	var data []byte
	var marshalErr error

	if f.pretty {
		data, marshalErr = json.MarshalIndent(result, "", "  ")
	} else {
		data, marshalErr = json.Marshal(result)
	}

	if marshalErr != nil {
		return fmt.Errorf("failed to marshal error: %w", marshalErr)
	}

	_, _ = fmt.Fprintln(f.writer, string(data))
	return nil
}

// FormatDocument formats a document as JSON
func (f *JSONFormatter) FormatDocument(doc interface{}) error {
	var data []byte
	var err error

	if f.pretty {
		data, err = json.MarshalIndent(doc, "", "  ")
	} else {
		data, err = json.Marshal(doc)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	_, _ = fmt.Fprintln(f.writer, string(data))
	return nil
}

// PlainFormatter formats output as plain text
type PlainFormatter struct {
	writer io.Writer
}

// NewPlainFormatter creates a new plain text formatter
func NewPlainFormatter(w io.Writer) OutputFormatter {
	return &PlainFormatter{writer: w}
}

// FormatTask formats a single task as plain text
func (f *PlainFormatter) FormatTask(task *entities.Task) error {
	_, _ = fmt.Fprintf(f.writer, "[%s] %s\n", truncateID(task.ID), task.Content)
	_, _ = fmt.Fprintf(f.writer, "Status: %s | Priority: %s | Repository: %s\n",
		task.Status, task.Priority, task.Repository)

	if len(task.Tags) > 0 {
		_, _ = fmt.Fprintf(f.writer, "Tags: %s\n", strings.Join(task.Tags, ", "))
	}

	_, _ = fmt.Fprintf(f.writer, "Created: %s | Updated: %s\n",
		task.CreatedAt.Format(time.RFC3339),
		task.UpdatedAt.Format(time.RFC3339))

	return nil
}

// FormatTaskList formats multiple tasks as plain text
func (f *PlainFormatter) FormatTaskList(tasks []*entities.Task) error {
	if len(tasks) == 0 {
		_, _ = fmt.Fprintln(f.writer, "No tasks found.")
		return nil
	}

	for i, task := range tasks {
		_, _ = fmt.Fprintf(f.writer, "%d. [%s] %s (%s/%s)\n",
			i+1,
			truncateID(task.ID),
			truncateContent(task.Content, 60),
			task.Status,
			task.Priority)

		if len(task.Tags) > 0 {
			_, _ = fmt.Fprintf(f.writer, "   Tags: %s\n", strings.Join(task.Tags, ", "))
		}
	}

	_, _ = fmt.Fprintf(f.writer, "\nTotal: %d tasks\n", len(tasks))
	return nil
}

// FormatStats formats repository statistics as plain text
func (f *PlainFormatter) FormatStats(stats *ports.RepositoryStats) error {
	_, _ = fmt.Fprintf(f.writer, "Repository: %s\n", stats.Repository)
	_, _ = fmt.Fprintf(f.writer, "Total Tasks: %d\n", stats.TotalTasks)
	_, _ = fmt.Fprintf(f.writer, "  Pending: %d\n", stats.PendingTasks)
	_, _ = fmt.Fprintf(f.writer, "  In Progress: %d\n", stats.InProgressTasks)
	_, _ = fmt.Fprintf(f.writer, "  Completed: %d\n", stats.CompletedTasks)
	_, _ = fmt.Fprintf(f.writer, "  Cancelled: %d\n", stats.CancelledTasks)
	_, _ = fmt.Fprintf(f.writer, "Total Tags: %d\n", stats.TotalTags)

	if stats.LastActivity != "" {
		_, _ = fmt.Fprintf(f.writer, "Last Activity: %s\n", stats.LastActivity)
	}

	return nil
}

// FormatError formats an error as plain text
func (f *PlainFormatter) FormatError(err error) error {
	_, _ = fmt.Fprintf(f.writer, "Error: %s\n", err.Error())
	return nil
}

// FormatDocument formats a document as plain text
func (f *PlainFormatter) FormatDocument(doc interface{}) error {
	// For plain format, output readable representation
	_, _ = fmt.Fprintf(f.writer, "%+v\n", doc)
	return nil
}

// Helper function to truncate content
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}

// Helper function to safely truncate ID to 8 characters
func truncateID(id string) string {
	const maxLen = 8
	if len(id) <= maxLen {
		return id
	}
	return id[:maxLen]
}
