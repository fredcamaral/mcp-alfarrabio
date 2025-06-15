package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"

	"github.com/spf13/cobra"
)

// addFormatFlag adds the common format flag to a command
func addFormatFlag(cmd *cobra.Command, format *string, _ string) {
	validFormats := fmt.Sprintf("%s, %s, %s",
		constants.OutputFormatTable,
		constants.OutputFormatJSON,
		constants.OutputFormatPlain)
	cmd.Flags().StringVarP(format, "format", "f", constants.OutputFormatTable,
		fmt.Sprintf("Output format (%s)", validFormats))
}

// addCommonFlags adds common flags used across multiple commands
func addCommonFlags(cmd *cobra.Command, format *string, limit *int) {
	addFormatFlag(cmd, format, constants.OutputFormatTable)
	if limit != nil {
		cmd.Flags().IntVarP(limit, "limit", "l", 10, "Maximum number of results")
	}
}

// hasEmptySlice checks if a slice is empty and returns early if so
func hasEmptySlice(slice []string) bool {
	return len(slice) == 0
}

// buildTagsHTML builds HTML representation of tags
func buildTagsHTML(tags []string) string {
	if hasEmptySlice(tags) {
		return ""
	}

	var html strings.Builder
	html.WriteString(`<div class="tags">`)
	for _, tag := range tags {
		html.WriteString(fmt.Sprintf(`<span class="tag">%s</span>`, tag))
	}
	html.WriteString("</div>")
	return html.String()
}

// buildTechListMarkdown builds markdown list of technologies
func buildTechListMarkdown(technologies []string) string {
	var content strings.Builder
	content.WriteString("### Technology Stack\n")
	for _, tech := range technologies {
		content.WriteString(fmt.Sprintf("- %s\n", tech))
	}
	content.WriteString("\n")
	return content.String()
}

// validateAndReadFile validates a file path and reads its content
func validateAndReadFile(file string) ([]byte, error) {
	cleanPath := filepath.Clean(file)

	// Validate file access
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", cleanPath)
	}

	// Read file content
	content, err := os.ReadFile(cleanPath) // #nosec G304 - Path validated above
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, nil
}

// buildTagsXML builds XML representation of tags
func buildTagsXML(tags []string) string {
	if hasEmptySlice(tags) {
		return ""
	}

	var xml strings.Builder
	xml.WriteString("    <tags>\n")
	for _, tag := range tags {
		xml.WriteString(fmt.Sprintf("      <tag>%s</tag>\n", tag))
	}
	xml.WriteString("    </tags>\n")
	return xml.String()
}

// buildMarkdownList builds a markdown list with a title
func buildMarkdownList(title string, items []string) string {
	if hasEmptySlice(items) {
		return ""
	}

	var content strings.Builder
	content.WriteString(fmt.Sprintf("## %s\n\n", title))
	for _, item := range items {
		content.WriteString(fmt.Sprintf("- %s\n", item))
	}
	content.WriteString("\n")
	return content.String()
}

// groupTasksByStatus groups tasks by their status
func groupTasksByStatus(tasks []*entities.Task) map[entities.Status][]*entities.Task {
	statusGroups := make(map[entities.Status][]*entities.Task)
	for _, task := range tasks {
		statusGroups[task.Status] = append(statusGroups[task.Status], task)
	}
	return statusGroups
}
