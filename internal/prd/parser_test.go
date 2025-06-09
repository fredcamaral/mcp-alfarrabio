package prd

import (
	"testing"

	"lerian-mcp-memory/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParser(t *testing.T) {
	config := DefaultParserConfig()
	parser := NewParser(config)

	assert.NotNil(t, parser)
	assert.Equal(t, config, parser.config)
}

func TestParseMarkdownDocument(t *testing.T) {
	parser := NewParser(DefaultParserConfig())

	markdownContent := `# Product Requirements Document

## Overview
This is a test PRD for a new feature.

## Objectives
- Goal 1: Improve user experience
- Goal 2: Increase performance

## Requirements
### Functional Requirements
The system must support user authentication.

### Non-Functional Requirements
The system must handle 1000 concurrent users.

## User Stories
As a user, I want to login quickly.
`

	doc, err := parser.ParseDocument(markdownContent, "markdown", "utf-8")

	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, types.PRDStatusProcessed, doc.Status)
	assert.Equal(t, "markdown", doc.Content.Format)
	assert.Greater(t, len(doc.Content.Sections), 0)
	assert.Greater(t, doc.Content.WordCount, 0)

	// Check that sections were parsed correctly
	sectionTitles := make([]string, len(doc.Content.Sections))
	for i, section := range doc.Content.Sections {
		sectionTitles[i] = section.Title
	}

	assert.Contains(t, sectionTitles, "Product Requirements Document")
	assert.Contains(t, sectionTitles, "Overview")
	assert.Contains(t, sectionTitles, "Objectives")
	assert.Contains(t, sectionTitles, "Requirements")
}

func TestParseTextDocument(t *testing.T) {
	parser := NewParser(DefaultParserConfig())

	textContent := `1. Overview
This is a test PRD in plain text format.

2. Objectives
Goal 1: Improve user experience
Goal 2: Increase performance

3. Requirements
The system must support user authentication.
`

	doc, err := parser.ParseDocument(textContent, "text", "utf-8")

	require.NoError(t, err)
	assert.NotNil(t, doc)
	assert.Equal(t, types.PRDStatusProcessed, doc.Status)
	assert.Equal(t, "text", doc.Content.Format)
	assert.Greater(t, len(doc.Content.Sections), 0)
}

func TestValidateContent(t *testing.T) {
	parser := NewParser(DefaultParserConfig())

	// Test valid content
	err := parser.ValidateContent("# Valid markdown content", "markdown", "utf-8")
	assert.NoError(t, err)

	// Test empty content
	err = parser.ValidateContent("", "markdown", "utf-8")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content is empty")

	// Test oversized content
	config := DefaultParserConfig()
	config.MaxFileSize = 10 // Very small limit
	parser = NewParser(config)

	err = parser.ValidateContent("This content is too long for the limit", "markdown", "utf-8")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed size")
}

func TestDetectSectionType(t *testing.T) {
	parser := NewParser(DefaultParserConfig())

	tests := []struct {
		title    string
		expected types.SectionType
	}{
		{"Overview", types.SectionTypeOverview},
		{"Project Overview", types.SectionTypeOverview},
		{"Goals and Objectives", types.SectionTypeObjectives},
		{"Functional Requirements", types.SectionTypeFunctional},
		{"User Stories", types.SectionTypeUserStories},
		{"Acceptance Criteria", types.SectionTypeAcceptance},
		{"Technical Architecture", types.SectionTypeTechnical},
		{"Random Title", types.SectionTypeOther},
	}

	for _, test := range tests {
		result := parser.detectSectionType(test.title)
		assert.Equal(t, test.expected, result, "Failed for title: %s", test.title)
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		content  string
		expected int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"This is a test document.", 5},
		{"Multiple\nlines\nwith\nwords", 4},
	}

	for _, test := range tests {
		result := countWords(test.content)
		assert.Equal(t, test.expected, result, "Failed for content: %s", test.content)
	}
}

func TestParseDocumentWithEmptyContent(t *testing.T) {
	parser := NewParser(DefaultParserConfig())

	doc, err := parser.ParseDocument("", "markdown", "utf-8")

	assert.Error(t, err)
	assert.Nil(t, doc)
	assert.Contains(t, err.Error(), "content cannot be empty")
}

func TestParseDocumentWithInvalidFormat(t *testing.T) {
	parser := NewParser(DefaultParserConfig())

	// Should still work as it falls back to text parsing
	doc, err := parser.ParseDocument("Some content", "unknown", "utf-8")

	require.NoError(t, err)
	assert.NotNil(t, doc)
}
