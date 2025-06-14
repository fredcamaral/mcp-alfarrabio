// Package documents provides data structures and processing for PRD/TRD document management.
package documents

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// Compiled regexes for performance
var (
	bulletPointRegex  = regexp.MustCompile(`^[-*•]\s+`)
	numberedListRegex = regexp.MustCompile(`^\d+\.\s+`)
	headerRegex       = regexp.MustCompile(`^#+\s*`)
)

// Processor handles document parsing and processing
type Processor struct {
	ruleManager *RuleManager
	mdParser    goldmark.Markdown
}

// NewProcessor creates a new document processor
func NewProcessor(ruleManager *RuleManager) *Processor {
	return &Processor{
		ruleManager: ruleManager,
		mdParser:    goldmark.New(),
	}
}

// ProcessPRDFile processes a PRD file and returns a PRDEntity
func (p *Processor) ProcessPRDFile(filePath, repository string) (*PRDEntity, error) {
	// Clean and validate the file path
	cleanPath := filepath.Clean(filePath)

	// Security check: prevent path traversal attacks
	if strings.Contains(cleanPath, "..") {
		return nil, errors.New("invalid file path: path traversal not allowed")
	}

	// Check if path is absolute and ensure it doesn't access system directories
	if filepath.IsAbs(cleanPath) {
		systemDirs := []string{"/etc/", "/usr/", "/bin/", "/sbin/", "/sys/", "/proc/", "/dev/"}
		for _, sysDir := range systemDirs {
			if strings.HasPrefix(cleanPath, sysDir) {
				return nil, errors.New("invalid file path: access to system directory not allowed")
			}
		}
	}

	content, err := os.ReadFile(cleanPath) // #nosec G304 -- Path is cleaned and validated above
	if err != nil {
		return nil, errors.New("failed to read PRD file: " + err.Error())
	}

	// Determine file type
	ext := strings.ToLower(filepath.Ext(filePath))
	var prd *PRDEntity

	switch ext {
	case ".md", ".markdown":
		prd, err = p.ProcessMarkdownPRD(content, repository)
	case ".txt":
		prd, err = p.ProcessTextPRD(content, repository)
	case ".json":
		prd, err = p.ProcessJSONPRD(content, repository)
	case ".yaml", ".yml":
		prd, err = p.ProcessYAMLPRD(content, repository)
	default:
		return nil, errors.New("unsupported file format: " + ext)
	}

	if err != nil {
		return nil, err
	}

	// Add file metadata
	prd.Metadata["source_file"] = filePath
	prd.Metadata["file_format"] = ext

	return prd, nil
}

// ProcessMarkdownPRD processes markdown content into a PRD entity
func (p *Processor) ProcessMarkdownPRD(content []byte, repository string) (*PRDEntity, error) {
	prd := &PRDEntity{
		Content:    string(content),
		Repository: repository,
		Metadata:   make(map[string]string),
		Status:     StatusProcessing,
	}

	// Parse sections
	prd.Sections = ParseSections(string(content))

	// Extract parsed content
	parsedContent := p.extractParsedContent(string(content), prd.Sections)
	prd.ParsedContent = parsedContent

	// Extract title from first heading or content
	if len(prd.Sections) > 0 {
		prd.Title = prd.Sections[0].Title
	} else {
		prd.Title = extractFirstLine(string(content))
	}

	// Estimate complexity
	prd.ComplexityScore = EstimateComplexity(string(content), prd.Sections)
	prd.EstimatedDuration = estimateDuration(prd.ComplexityScore)

	// Validate
	if err := prd.Validate(); err != nil {
		return nil, err
	}

	prd.Status = StatusDraft
	return prd, nil
}

// ProcessTextPRD processes plain text content into a PRD entity
func (p *Processor) ProcessTextPRD(content []byte, repository string) (*PRDEntity, error) {
	prd := &PRDEntity{
		Content:    string(content),
		Repository: repository,
		Metadata:   make(map[string]string),
		Status:     StatusProcessing,
	}

	// For plain text, treat paragraphs as sections
	paragraphs := strings.Split(string(content), "\n\n")
	for i, para := range paragraphs {
		if strings.TrimSpace(para) != "" {
			prd.Sections = append(prd.Sections, Section{
				Title:   "Section " + strconv.Itoa(i+1),
				Content: strings.TrimSpace(para),
				Level:   1,
				Order:   i + 1,
			})
		}
	}

	// Extract title from first line
	prd.Title = extractFirstLine(string(content))

	// Extract parsed content
	parsedContent := p.extractParsedContent(string(content), prd.Sections)
	prd.ParsedContent = parsedContent

	// Estimate complexity
	prd.ComplexityScore = EstimateComplexity(string(content), prd.Sections)
	prd.EstimatedDuration = estimateDuration(prd.ComplexityScore)

	// Validate
	if err := prd.Validate(); err != nil {
		return nil, err
	}

	prd.Status = StatusDraft
	return prd, nil
}

// ProcessJSONPRD processes JSON content into a PRD entity
func (p *Processor) ProcessJSONPRD(content []byte, repository string) (*PRDEntity, error) {
	jsonData, err := p.parseJSONContent(content)
	if err != nil {
		return nil, err
	}

	prd := p.createBasePRD(repository)
	p.extractCommonFields(prd, jsonData)
	p.convertJSONToSections(prd, jsonData)
	p.finalizeContent(prd)
	if err := p.setComplexityAndValidate(prd); err != nil {
		return nil, err
	}

	prd.Status = StatusDraft
	return prd, nil
}

// ProcessYAMLPRD processes YAML content into a PRD entity
func (p *Processor) ProcessYAMLPRD(content []byte, repository string) (*PRDEntity, error) {
	var yamlData map[string]interface{}
	if err := yaml.Unmarshal(content, &yamlData); err != nil {
		return nil, errors.New("failed to parse YAML: " + err.Error())
	}

	// Convert YAML to JSON and process
	jsonBytes, err := json.Marshal(yamlData)
	if err != nil {
		return nil, errors.New("failed to convert YAML to JSON: " + err.Error())
	}

	return p.ProcessJSONPRD(jsonBytes, repository)
}

// extractParsedContent extracts structured information from content
func (p *Processor) extractParsedContent(content string, sections []Section) ParsedContent {
	parsed := ParsedContent{
		Goals:          []string{},
		Requirements:   []string{},
		UserStories:    []string{},
		TechnicalNotes: []string{},
		Constraints:    []string{},
		Keywords:       ExtractKeywords(content),
		ExtractedData:  make(map[string]string),
	}

	// Extract project name from title or content
	projectNameRegex := regexp.MustCompile(`(?i)project\s*[:：]\s*([^\n]+)`)
	if matches := projectNameRegex.FindStringSubmatch(content); len(matches) > 1 {
		parsed.ProjectName = strings.TrimSpace(matches[1])
	}

	// Extract summary from sections
	for _, section := range sections {
		lowerTitle := strings.ToLower(section.Title)

		switch {
		case strings.Contains(lowerTitle, "summary") || strings.Contains(lowerTitle, "overview"):
			parsed.Summary = section.Content
		case strings.Contains(lowerTitle, "goal") || strings.Contains(lowerTitle, "objective"):
			parsed.Goals = extractListItems(section.Content)
		case strings.Contains(lowerTitle, "requirement"):
			parsed.Requirements = extractListItems(section.Content)
		case strings.Contains(lowerTitle, "user stor"):
			parsed.UserStories = extractListItems(section.Content)
		case strings.Contains(lowerTitle, "technical"):
			parsed.TechnicalNotes = extractListItems(section.Content)
		case strings.Contains(lowerTitle, "constraint") || strings.Contains(lowerTitle, "limitation"):
			parsed.Constraints = extractListItems(section.Content)
		}
	}

	// Extract user stories from content using patterns
	userStoryRegex := regexp.MustCompile(`(?i)as\s+a\s+[^,]+,\s*I\s+want\s+[^,]+(?:,\s*so\s+that\s+[^.]+)?`)
	if matches := userStoryRegex.FindAllString(content, -1); len(matches) > 0 {
		for _, match := range matches {
			if !contains(parsed.UserStories, match) {
				parsed.UserStories = append(parsed.UserStories, match)
			}
		}
	}

	return parsed
}

// extractListItems extracts list items from content
func extractListItems(content string) []string {
	items := []string{}
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Check for bullet points or numbered lists
		if bulletPointRegex.MatchString(line) {
			item := bulletPointRegex.ReplaceAllString(line, "")
			items = append(items, strings.TrimSpace(item))
		} else if numberedListRegex.MatchString(line) {
			item := numberedListRegex.ReplaceAllString(line, "")
			items = append(items, strings.TrimSpace(item))
		}
	}

	return items
}

// extractFirstLine extracts the first non-empty line from content
func extractFirstLine(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			// Remove markdown headers
			line = headerRegex.ReplaceAllString(line, "")
			return strings.TrimSpace(line)
		}
	}
	return "Untitled PRD"
}

// estimateDuration estimates project duration based on complexity
func estimateDuration(complexity int) string {
	switch {
	case complexity < 20:
		return "1-2 weeks"
	case complexity < 40:
		return "2-4 weeks"
	case complexity < 60:
		return "4-8 weeks"
	case complexity < 80:
		return "8-12 weeks"
	default:
		return "12+ weeks"
	}
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ValidatePRD validates a PRD against rules
func (p *Processor) ValidatePRD(prd *PRDEntity) error {
	// Get validation rules
	rules := p.ruleManager.GetRulesByType(RuleValidation)

	for _, rule := range rules {
		if err := p.applyValidationRule(rule, prd); err != nil {
			return err
		}
	}

	return nil
}

// applyValidationRule applies a single validation rule to a PRD
func (p *Processor) applyValidationRule(rule *Rule, prd *PRDEntity) error {
	if rule.Metadata["target"] != "prd" {
		return nil
	}

	requiredSections, ok := rule.Metadata["required_sections"]
	if !ok {
		return nil
	}

	return p.validateRequiredSections(requiredSections, prd.Sections)
}

// validateRequiredSections checks if all required sections are present
func (p *Processor) validateRequiredSections(requiredSections string, sections []Section) error {
	sectionNames := strings.Split(requiredSections, ",")

	for _, required := range sectionNames {
		required = strings.TrimSpace(required)
		if !p.sectionExists(required, sections) {
			return errors.New("required section missing: " + required)
		}
	}
	return nil
}

// sectionExists checks if a section with the given title exists
func (p *Processor) sectionExists(title string, sections []Section) bool {
	for _, section := range sections {
		if strings.EqualFold(section.Title, title) {
			return true
		}
	}
	return false
}

// ProcessMarkdownToSections processes markdown content using goldmark
func (p *Processor) ProcessMarkdownToSections(content []byte) ([]Section, error) {
	reader := text.NewReader(content)
	doc := p.mdParser.Parser().Parse(reader)

	sections := []Section{}
	var currentSection *Section
	var contentBuffer bytes.Buffer
	order := 0

	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Heading:
			// Save previous section
			if currentSection != nil {
				currentSection.Content = strings.TrimSpace(contentBuffer.String())
				sections = append(sections, *currentSection)
				contentBuffer.Reset()
			}

			// Start new section
			order++
			currentSection = &Section{
				Level: node.Level,
				Order: order,
			}

			// Extract heading text
			var headingText bytes.Buffer
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				if textNode, ok := child.(*ast.Text); ok {
					_, _ = headingText.Write(textNode.Segment.Value(reader.Source()))
				}
			}
			currentSection.Title = strings.TrimSpace(headingText.String())

		default:
			if currentSection != nil {
				// Collect content for current section - collect text nodes
				if textNode, ok := n.(*ast.Text); ok {
					_, _ = contentBuffer.Write(textNode.Segment.Value(reader.Source()))
					_, _ = contentBuffer.WriteString("\n")
				}
			}
		}

		return ast.WalkContinue, nil
	})

	// Save last section
	if currentSection != nil {
		currentSection.Content = strings.TrimSpace(contentBuffer.String())
		sections = append(sections, *currentSection)
	}

	return sections, err
}

// ExportPRD exports a PRD entity to the specified format
func (p *Processor) ExportPRD(prd *PRDEntity, format string, writer io.Writer) error {
	switch strings.ToLower(format) {
	case "markdown", "md":
		return p.exportPRDMarkdown(prd, writer)
	case "json":
		return p.exportPRDJSON(prd, writer)
	case "yaml", "yml":
		return p.exportPRDYAML(prd, writer)
	default:
		return errors.New("unsupported export format: " + format)
	}
}

// exportPRDMarkdown exports PRD as markdown
func (p *Processor) exportPRDMarkdown(prd *PRDEntity, writer io.Writer) error {
	// Write title
	if _, err := fmt.Fprintf(writer, "# %s\n\n", prd.Title); err != nil {
		return errors.New("failed to write title: " + err.Error())
	}

	// Write metadata
	if _, err := fmt.Fprintf(writer, "**Generated:** %s\n", prd.GeneratedAt.Format("2006-01-02")); err != nil {
		return errors.New("failed to write generated date: " + err.Error())
	}
	if _, err := fmt.Fprintf(writer, "**Status:** %s\n", prd.Status); err != nil {
		return errors.New("failed to write status: " + err.Error())
	}
	if _, err := fmt.Fprintf(writer, "**Complexity:** %d/100\n", prd.ComplexityScore); err != nil {
		return errors.New("failed to write complexity: " + err.Error())
	}
	if _, err := fmt.Fprintf(writer, "**Estimated Duration:** %s\n\n", prd.EstimatedDuration); err != nil {
		return errors.New("failed to write duration: " + err.Error())
	}

	// Write sections
	for _, section := range prd.Sections {
		prefix := strings.Repeat("#", section.Level)
		if _, err := fmt.Fprintf(writer, "%s %s\n\n", prefix, section.Title); err != nil {
			return errors.New("failed to write section title: " + err.Error())
		}
		if _, err := fmt.Fprintf(writer, "%s\n\n", section.Content); err != nil {
			return errors.New("failed to write section content: " + err.Error())
		}
	}

	return nil
}

// exportPRDJSON exports PRD as JSON
func (p *Processor) exportPRDJSON(prd *PRDEntity, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(prd)
}

// exportPRDYAML exports PRD as YAML
func (p *Processor) exportPRDYAML(prd *PRDEntity, writer io.Writer) error {
	encoder := yaml.NewEncoder(writer)
	defer func() {
		if err := encoder.Close(); err != nil {
			log.Printf("Failed to close YAML encoder: %v", err)
		}
	}()
	return encoder.Encode(prd)
}

// parseJSONContent parses JSON bytes into a map
func (p *Processor) parseJSONContent(content []byte) (map[string]interface{}, error) {
	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		return nil, errors.New("failed to parse JSON: " + err.Error())
	}
	return jsonData, nil
}

// createBasePRD creates a base PRD entity
func (p *Processor) createBasePRD(repository string) *PRDEntity {
	return &PRDEntity{
		Repository: repository,
		Metadata:   make(map[string]string),
		Status:     StatusProcessing,
	}
}

// extractCommonFields extracts title and content from JSON data
func (p *Processor) extractCommonFields(prd *PRDEntity, jsonData map[string]interface{}) {
	if title, ok := jsonData["title"].(string); ok {
		prd.Title = title
	}
	if desc, ok := jsonData["description"].(string); ok {
		prd.Content = desc
	} else if content, ok := jsonData["content"].(string); ok {
		prd.Content = content
	}
}

// convertJSONToSections converts JSON structure to sections
func (p *Processor) convertJSONToSections(prd *PRDEntity, jsonData map[string]interface{}) {
	sectionOrder := 0
	for key, value := range jsonData {
		if p.isReservedField(key) {
			continue
		}
		sectionOrder++
		section := p.createSectionFromValue(key, value, sectionOrder)
		prd.Sections = append(prd.Sections, section)
	}
}

// isReservedField checks if a field is reserved and shouldn't become a section
func (p *Processor) isReservedField(key string) bool {
	return key == "title" || key == "description" || key == "content"
}

// createSectionFromValue creates a section from a JSON value
func (p *Processor) createSectionFromValue(key string, value interface{}, order int) Section {
	section := Section{
		Title: FormatTitle(key),
		Level: 1,
		Order: order,
	}

	switch v := value.(type) {
	case string:
		section.Content = v
	case []interface{}:
		section.Content = p.convertArrayToString(v)
	default:
		section.Content = fmt.Sprintf("%v", v)
	}

	return section
}

// convertArrayToString converts array interface to string list
func (p *Processor) convertArrayToString(arr []interface{}) string {
	items := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			items = append(items, "- "+s)
		}
	}
	return strings.Join(items, "\n")
}

// finalizeContent generates content from sections if not set
func (p *Processor) finalizeContent(prd *PRDEntity) {
	if prd.Content == "" {
		contentParts := make([]string, 0, len(prd.Sections))
		for _, section := range prd.Sections {
			contentParts = append(contentParts, fmt.Sprintf("# %s\n\n%s", section.Title, section.Content))
		}
		prd.Content = strings.Join(contentParts, "\n\n")
	}

	// Extract parsed content
	parsedContent := p.extractParsedContent(prd.Content, prd.Sections)
	prd.ParsedContent = parsedContent
}

// setComplexityAndValidate estimates complexity and validates the PRD
func (p *Processor) setComplexityAndValidate(prd *PRDEntity) error {
	prd.ComplexityScore = EstimateComplexity(prd.Content, prd.Sections)
	prd.EstimatedDuration = estimateDuration(prd.ComplexityScore)

	return prd.Validate()
}
