// Package prd provides PRD document parsing and processing functionality.
package prd

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"lerian-mcp-memory/pkg/types"
)

// Parser handles PRD document parsing
type Parser struct {
	config ParserConfig
}

// ParserConfig represents parser configuration
type ParserConfig struct {
	MaxFileSize        int64 `json:"max_file_size"`
	MaxSections        int   `json:"max_sections"`
	MaxDepth           int   `json:"max_depth"`
	EnableAIProcessing bool  `json:"enable_ai_processing"`
	StrictMode         bool  `json:"strict_mode"`
}

// DefaultParserConfig returns default parser configuration
func DefaultParserConfig() ParserConfig {
	return ParserConfig{
		MaxFileSize:        10 * 1024 * 1024, // 10MB
		MaxSections:        1000,
		MaxDepth:           10,
		EnableAIProcessing: true,
		StrictMode:         false,
	}
}

// NewParser creates a new PRD parser
func NewParser(config ParserConfig) *Parser {
	return &Parser{
		config: config,
	}
}

// ParseDocument parses a PRD document from content
func (p *Parser) ParseDocument(content, format, encoding string) (*types.PRDDocument, error) {
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	// Check file size
	if int64(len(content)) > p.config.MaxFileSize {
		return nil, fmt.Errorf("document size exceeds maximum allowed size")
	}

	startTime := time.Now()

	// Create document structure
	doc := &types.PRDDocument{
		ID:     generateDocumentID(),
		Status: types.PRDStatusImported,
		Content: types.PRDContent{
			Raw:       content,
			Format:    format,
			Encoding:  encoding,
			WordCount: countWords(content),
		},
		Processing: types.PRDProcessing{
			ImportMethod:     "parser",
			FileSize:         int64(len(content)),
			ProcessorVersion: "1.0.0",
			ProcessingSteps:  []types.ProcessingStep{},
		},
		Timestamps: types.PRDTimestamps{
			Created:  time.Now(),
			Imported: time.Now(),
		},
	}

	// Add processing step
	p.addProcessingStep(doc, "document_initialization", types.StepStatusCompleted, time.Now(), time.Now(), "")

	// Parse based on format
	var err error
	switch strings.ToLower(format) {
	case "markdown", "md":
		p.parseMarkdown(doc)
	case "text", "txt", "plain":
		p.parseText(doc)
	default:
		// Try to auto-detect format
		if strings.Contains(content, "#") || strings.Contains(content, "**") {
			p.parseMarkdown(doc)
		} else {
			p.parseText(doc)
		}
	}

	// Parsing completed successfully - no error checking needed as functions don't return errors
	if false { // placeholder to maintain structure
		p.addProcessingStep(doc, "document_parsing", types.StepStatusFailed, startTime, time.Now(), err.Error())
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}

	// Complete processing
	p.addProcessingStep(doc, "document_parsing", types.StepStatusCompleted, startTime, time.Now(), "")
	doc.Processing.ProcessingTime = time.Since(startTime)
	doc.Status = types.PRDStatusProcessed

	return doc, nil
}

// parseMarkdown parses markdown content
func (p *Parser) parseMarkdown(doc *types.PRDDocument) {
	content := doc.Content.Raw
	sections := []types.PRDSection{}

	// Regex patterns for markdown
	headingRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	codeBlockRegex := regexp.MustCompile("```")
	tableRegex := regexp.MustCompile(`\|.*\|`)
	imageRegex := regexp.MustCompile(`!\[.*\]\(.*\)`)

	lines := strings.Split(content, "\n")
	currentSection := types.PRDSection{}
	sectionContent := []string{}
	sectionID := 0
	inCodeBlock := false
	hasCode := false
	hasImages := false
	hasTables := false

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Check for code blocks
		if codeBlockRegex.MatchString(line) {
			inCodeBlock = !inCodeBlock
			hasCode = true
		}

		// Check for images
		if imageRegex.MatchString(line) {
			hasImages = true
		}

		// Check for tables
		if !inCodeBlock && tableRegex.MatchString(line) {
			hasTables = true
		}

		// Check for headings
		if matches := headingRegex.FindStringSubmatch(line); matches != nil {
			// Save previous section if exists
			if currentSection.Title != "" {
				currentSection.Content = strings.Join(sectionContent, "\n")
				sections = append(sections, currentSection)
			}

			// Start new section
			sectionID++
			level := len(matches[1])
			title := matches[2]

			currentSection = types.PRDSection{
				ID:    fmt.Sprintf("section_%d", sectionID),
				Title: title,
				Type:  p.detectSectionType(title),
				Level: level,
				Order: i,
			}
			sectionContent = []string{}
		} else {
			// Add line to current section content
			sectionContent = append(sectionContent, line)
		}
	}

	// Save last section
	if currentSection.Title != "" {
		currentSection.Content = strings.Join(sectionContent, "\n")
		sections = append(sections, currentSection)
	}

	// Update document content
	doc.Content.Sections = sections
	doc.Content.Structure = types.PRDStructure{
		TotalSections:  len(sections),
		SectionsByType: p.countSectionsByType(sections),
		MaxDepth:       p.calculateMaxDepth(sections),
		HasTOC:         p.detectTOC(content),
		HasImages:      hasImages,
		HasTables:      hasTables,
		HasCode:        hasCode,
		HasDiagrams:    p.detectDiagrams(content),
	}
}

// parseText parses plain text content
func (p *Parser) parseText(doc *types.PRDDocument) {
	content := doc.Content.Raw
	lines := strings.Split(content, "\n")

	// Simple text parsing - look for numbered sections, blank lines, etc.
	sections := []types.PRDSection{}
	currentSection := types.PRDSection{}
	sectionContent := []string{}
	sectionID := 0

	// Regex for numbered sections
	numberedRegex := regexp.MustCompile(`^(\d+\.?\d*\.?)\s+(.+)$`)

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines at the beginning
		if line == "" && currentSection.Title == "" {
			continue
		}

		// Check for numbered sections
		if matches := numberedRegex.FindStringSubmatch(line); matches != nil {
			// Save previous section if exists
			if currentSection.Title != "" {
				currentSection.Content = strings.Join(sectionContent, "\n")
				sections = append(sections, currentSection)
			}

			// Start new section
			sectionID++
			title := matches[2]

			currentSection = types.PRDSection{
				ID:    fmt.Sprintf("section_%d", sectionID),
				Title: title,
				Type:  p.detectSectionType(title),
				Level: 1,
				Order: i,
			}
			sectionContent = []string{}
		} else {
			// Add line to current section content
			sectionContent = append(sectionContent, line)
		}
	}

	// Save last section
	if currentSection.Title != "" {
		currentSection.Content = strings.Join(sectionContent, "\n")
		sections = append(sections, currentSection)
	}

	// If no sections found, create a single section with all content
	if len(sections) == 0 {
		sections = []types.PRDSection{
			{
				ID:      "section_1",
				Title:   "Document Content",
				Type:    types.SectionTypeOther,
				Content: content,
				Level:   1,
				Order:   0,
			},
		}
	}

	// Update document content
	doc.Content.Sections = sections
	doc.Content.Structure = types.PRDStructure{
		TotalSections:  len(sections),
		SectionsByType: p.countSectionsByType(sections),
		MaxDepth:       p.calculateMaxDepth(sections),
		HasTOC:         false,
		HasImages:      false,
		HasTables:      false,
		HasCode:        false,
		HasDiagrams:    false,
	}
}

// detectSectionType detects the type of a section based on its title
func (p *Parser) detectSectionType(title string) types.SectionType {
	titleLower := strings.ToLower(title)

	// Define keywords for each section type
	keywords := map[types.SectionType][]string{
		types.SectionTypeOverview:      {"overview", "introduction", "summary", "about"},
		types.SectionTypeObjectives:    {"objectives", "goals", "purpose", "mission"},
		types.SectionTypeRequirements:  {"requirements", "req", "specification", "specs"},
		types.SectionTypeFunctional:    {"functional", "features", "functionality"},
		types.SectionTypeNonFunctional: {"non-functional", "performance", "scalability", "security"},
		types.SectionTypeTechnical:     {"technical", "technology", "tech", "implementation"},
		types.SectionTypeDesign:        {"design", "ui", "ux", "interface", "wireframe"},
		types.SectionTypeArchitecture:  {"architecture", "system", "infrastructure", "deployment"},
		types.SectionTypeUserStories:   {"user stories", "stories", "scenarios", "use cases"},
		types.SectionTypeAcceptance:    {"acceptance", "criteria", "testing", "validation"},
		types.SectionTypeConstraints:   {"constraints", "limitations", "restrictions"},
		types.SectionTypeAssumptions:   {"assumptions", "dependencies", "prerequisites"},
		types.SectionTypeTimeline:      {"timeline", "schedule", "milestones", "roadmap"},
		types.SectionTypeResources:     {"resources", "team", "budget", "cost"},
		types.SectionTypeRisks:         {"risks", "challenges", "issues", "concerns"},
		types.SectionTypeSuccess:       {"success", "metrics", "kpi", "measurement"},
	}

	// Check for specific compound matches first
	if strings.Contains(titleLower, "functional") && strings.Contains(titleLower, "requirement") {
		return types.SectionTypeFunctional
	}
	if strings.Contains(titleLower, "non-functional") && strings.Contains(titleLower, "requirement") {
		return types.SectionTypeNonFunctional
	}
	if strings.Contains(titleLower, "technical") && strings.Contains(titleLower, "architecture") {
		return types.SectionTypeTechnical
	}
	if strings.Contains(titleLower, "user") && strings.Contains(titleLower, "stories") {
		return types.SectionTypeUserStories
	}
	if strings.Contains(titleLower, "acceptance") && strings.Contains(titleLower, "criteria") {
		return types.SectionTypeAcceptance
	}

	// Check for single keyword matches
	for sectionType, keywordList := range keywords {
		for _, keyword := range keywordList {
			if strings.Contains(titleLower, keyword) {
				return sectionType
			}
		}
	}

	return types.SectionTypeOther
}

// countSectionsByType counts sections by their type
func (p *Parser) countSectionsByType(sections []types.PRDSection) map[types.SectionType]int {
	counts := make(map[types.SectionType]int)
	for _, section := range sections {
		counts[section.Type]++
	}
	return counts
}

// calculateMaxDepth calculates the maximum nesting depth
func (p *Parser) calculateMaxDepth(sections []types.PRDSection) int {
	maxDepth := 0
	for _, section := range sections {
		if section.Level > maxDepth {
			maxDepth = section.Level
		}
	}
	return maxDepth
}

// detectTOC detects if the document has a table of contents
func (p *Parser) detectTOC(content string) bool {
	tocPatterns := []string{
		"table of contents",
		"contents",
		"toc",
		"index",
	}

	contentLower := strings.ToLower(content)
	for _, pattern := range tocPatterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}

	return false
}

// detectDiagrams detects if the document contains diagrams
func (p *Parser) detectDiagrams(content string) bool {
	diagramPatterns := []string{
		"diagram",
		"flowchart",
		"mermaid",
		"plantuml",
		"sequence",
		"architecture",
	}

	contentLower := strings.ToLower(content)
	for _, pattern := range diagramPatterns {
		if strings.Contains(contentLower, pattern) {
			return true
		}
	}

	return false
}

// countWords counts the number of words in content
func countWords(content string) int {
	if content == "" {
		return 0
	}

	scanner := bufio.NewScanner(strings.NewReader(content))
	scanner.Split(bufio.ScanWords)

	count := 0
	for scanner.Scan() {
		count++
	}

	return count
}

// generateDocumentID generates a unique document ID
func generateDocumentID() string {
	return fmt.Sprintf("prd_%d", time.Now().UnixNano())
}

// addProcessingStep adds a processing step to the document
func (p *Parser) addProcessingStep(doc *types.PRDDocument, name string, status types.StepStatus, startTime, endTime time.Time, errorMsg string) {
	step := types.ProcessingStep{
		Name:      name,
		Status:    status,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
	}

	if errorMsg != "" {
		step.Error = errorMsg
	}

	doc.Processing.ProcessingSteps = append(doc.Processing.ProcessingSteps, step)
}

// ValidateContent validates the content before parsing
func (p *Parser) ValidateContent(content, format, encoding string) error {
	// Check if content is empty
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content is empty")
	}

	// Check file size
	if int64(len(content)) > p.config.MaxFileSize {
		return fmt.Errorf("content size %d exceeds maximum allowed size %d", len(content), p.config.MaxFileSize)
	}

	// Check encoding
	if encoding == "utf-8" || encoding == "" {
		if !utf8.ValidString(content) {
			return fmt.Errorf("content is not valid UTF-8")
		}
	}

	// Format-specific validation
	switch strings.ToLower(format) {
	case "markdown", "md":
		return p.validateMarkdown(content)
	case "text", "txt", "plain":
		return p.validateText(content)
	}

	return nil
}

// validateMarkdown validates markdown content
func (p *Parser) validateMarkdown(content string) error {
	// Check for basic markdown structure
	if !strings.Contains(content, "#") && !strings.Contains(content, "**") && !strings.Contains(content, "*") {
		if p.config.StrictMode {
			return fmt.Errorf("content does not appear to be valid markdown")
		}
	}

	return nil
}

// validateText validates plain text content
func (p *Parser) validateText(content string) error {
	// Basic text validation - check for reasonable line lengths
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if len(line) > 1000 {
			if p.config.StrictMode {
				return fmt.Errorf("line length exceeds reasonable limit")
			}
		}
	}

	return nil
}
