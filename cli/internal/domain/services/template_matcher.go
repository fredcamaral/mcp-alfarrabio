// Package services provides domain services for the lerian-mcp-memory CLI.
package services

import (
	"errors"
	"log/slog"
	"strconv"
)

// TemplateMatcher handles matching tasks to appropriate templates
type TemplateMatcher interface {
	FindBestMatch(feature, projectType string) *TaskTemplate
	LoadTemplates() error
	GetTemplatesByCategory(category string) []*TaskTemplate
	GetTemplateByID(id string) *TaskTemplate
	AddTemplate(template *TaskTemplate) error
	GetAllTemplates() []*TaskTemplate
}

// DefaultTemplateMatcher implements TemplateMatcher
type DefaultTemplateMatcher struct {
	templates map[string]*TaskTemplate
	logger    *slog.Logger
}

// NewTemplateMatcher creates a new template matcher
func NewTemplateMatcher(logger *slog.Logger) *DefaultTemplateMatcher {
	matcher := &DefaultTemplateMatcher{
		templates: make(map[string]*TaskTemplate),
		logger:    logger,
	}

	// Load default templates
	if err := matcher.LoadTemplates(); err != nil {
		logger.Warn("failed to load default templates", slog.Any("error", err))
	}

	return matcher
}

// FindBestMatch finds the best template match for a feature and project type
func (m *DefaultTemplateMatcher) FindBestMatch(feature, projectType string) *TaskTemplate {
	if feature == "" {
		return m.getDefaultTemplate()
	}

	bestMatch := (*TaskTemplate)(nil)
	bestScore := 0.0

	featureLower := toLowerCase(feature)
	projectTypeLower := toLowerCase(projectType)

	for _, template := range m.templates {
		score := m.calculateMatchScore(template, featureLower, projectTypeLower)
		if score > bestScore {
			bestScore = score
			bestMatch = template
		}
	}

	if bestMatch == nil {
		m.logger.Debug("no template match found, using default",
			slog.String("feature", feature),
			slog.String("project_type", projectType))
		return m.getDefaultTemplate()
	}

	m.logger.Debug("found template match",
		slog.String("template_id", bestMatch.ID),
		slog.String("template_name", bestMatch.Name),
		slog.String("score", strconv.FormatFloat(bestScore, 'f', 2, 64)),
		slog.String("feature", feature))

	return bestMatch
}

// LoadTemplates loads default templates into the matcher
func (m *DefaultTemplateMatcher) LoadTemplates() error {
	defaultTemplates := m.getDefaultTemplates()

	for _, template := range defaultTemplates {
		if err := m.AddTemplate(template); err != nil {
			return err
		}
	}

	m.logger.Info("loaded templates", slog.Int("count", len(m.templates)))
	return nil
}

// GetTemplatesByCategory returns templates filtered by category
func (m *DefaultTemplateMatcher) GetTemplatesByCategory(category string) []*TaskTemplate {
	var templates []*TaskTemplate
	categoryLower := toLowerCase(category)

	for _, template := range m.templates {
		if toLowerCase(template.Category) == categoryLower {
			templates = append(templates, template)
		}
	}

	return templates
}

// GetTemplateByID returns a template by its ID
func (m *DefaultTemplateMatcher) GetTemplateByID(id string) *TaskTemplate {
	template, exists := m.templates[id]
	if !exists {
		return nil
	}
	return template
}

// AddTemplate adds a new template to the matcher
func (m *DefaultTemplateMatcher) AddTemplate(template *TaskTemplate) error {
	if template == nil {
		return errors.New("template cannot be nil")
	}
	if template.ID == "" {
		return errors.New("template ID cannot be empty")
	}
	if template.Name == "" {
		return errors.New("template name cannot be empty")
	}

	m.templates[template.ID] = template
	return nil
}

// GetAllTemplates returns all available templates
func (m *DefaultTemplateMatcher) GetAllTemplates() []*TaskTemplate {
	var templates []*TaskTemplate
	for _, template := range m.templates {
		templates = append(templates, template)
	}
	return templates
}

// calculateMatchScore calculates how well a template matches the given feature and project type
func (m *DefaultTemplateMatcher) calculateMatchScore(template *TaskTemplate, feature, projectType string) float64 {
	score := 0.0

	// Check keyword matches in feature
	for _, keyword := range template.Keywords {
		keywordLower := toLowerCase(keyword)
		if containsKeyword(feature, keywordLower) {
			score += 2.0 // Keyword match gets high score
		}
	}

	// Check pattern match
	if template.Pattern != "" && containsKeyword(feature, toLowerCase(template.Pattern)) {
		score += 3.0 // Pattern match gets highest score
	}

	// Check category match with project type
	if projectType != "" {
		categoryLower := toLowerCase(template.Category)
		if containsKeyword(projectType, categoryLower) {
			score += 1.0 // Category match gets moderate score
		}
	}

	// Check type relevance
	typeLower := toLowerCase(template.Type)
	if containsKeyword(feature, typeLower) {
		score += 1.0
	}

	// Normalize score by template complexity to favor simpler templates when scores are equal
	complexityMultiplier := 1.0
	switch toLowerCase(template.Complexity) {
	case "low":
		complexityMultiplier = 1.2
	case "medium":
		complexityMultiplier = 1.0
	case "high":
		complexityMultiplier = 0.8
	}

	return score * complexityMultiplier
}

// getDefaultTemplate returns a generic default template
func (m *DefaultTemplateMatcher) getDefaultTemplate() *TaskTemplate {
	return &TaskTemplate{
		ID:          "generic-task",
		Name:        "Generic Task Implementation",
		Type:        "implementation",
		Pattern:     "",
		Keywords:    []string{"implementation", "development", "code"},
		Complexity:  "medium",
		Hours:       4,
		SubTasks:    []string{"plan", "implement", "test"},
		Description: "Generic task implementation with standard workflow",
		Category:    "general",
	}
}

// getDefaultTemplates returns the default set of templates
func (m *DefaultTemplateMatcher) getDefaultTemplates() []*TaskTemplate {
	return []*TaskTemplate{
		// Backend Templates
		{
			ID:          "api-endpoint",
			Name:        "REST API Endpoint",
			Type:        "implementation",
			Pattern:     "api|endpoint|rest|http",
			Keywords:    []string{"api", "endpoint", "rest", "http", "service", "handler"},
			Complexity:  "medium",
			Hours:       8,
			SubTasks:    []string{"design-endpoint", "implement-handler", "add-validation", "write-tests"},
			Description: "Implement REST API endpoint with proper validation and error handling",
			Category:    "backend",
		},
		{
			ID:          "database-model",
			Name:        "Database Model",
			Type:        "implementation",
			Pattern:     "database|model|schema|table",
			Keywords:    []string{"database", "model", "schema", "table", "entity", "migration"},
			Complexity:  "medium",
			Hours:       6,
			SubTasks:    []string{"design-schema", "create-migration", "implement-model", "add-relations"},
			Description: "Create database model with proper schema and relationships",
			Category:    "data",
		},
		{
			ID:          "authentication",
			Name:        "Authentication System",
			Type:        "implementation",
			Pattern:     "auth|authentication|login|security",
			Keywords:    []string{"auth", "authentication", "login", "security", "jwt", "oauth"},
			Complexity:  "high",
			Hours:       16,
			SubTasks:    []string{"setup-auth", "implement-login", "add-middleware", "test-security"},
			Description: "Implement secure authentication system with proper session management",
			Category:    "security",
		},
		{
			ID:          "middleware",
			Name:        "Middleware Component",
			Type:        "implementation",
			Pattern:     "middleware|interceptor|filter",
			Keywords:    []string{"middleware", "interceptor", "filter", "cors", "logging"},
			Complexity:  "low",
			Hours:       4,
			SubTasks:    []string{"design-middleware", "implement-logic", "add-tests", "integrate"},
			Description: "Create reusable middleware component for request processing",
			Category:    "backend",
		},

		// Frontend Templates
		{
			ID:          "ui-component",
			Name:        "UI Component",
			Type:        "implementation",
			Pattern:     "ui|component|interface|widget",
			Keywords:    []string{"ui", "component", "interface", "react", "vue", "widget"},
			Complexity:  "medium",
			Hours:       6,
			SubTasks:    []string{"design-component", "implement-ui", "add-styles", "write-tests"},
			Description: "Develop reusable UI component with proper styling and interactions",
			Category:    "frontend",
		},
		{
			ID:          "form-handling",
			Name:        "Form Implementation",
			Type:        "implementation",
			Pattern:     "form|input|validation",
			Keywords:    []string{"form", "input", "validation", "submit", "field"},
			Complexity:  "medium",
			Hours:       8,
			SubTasks:    []string{"design-form", "add-validation", "handle-submit", "add-feedback"},
			Description: "Create form with proper validation and user feedback",
			Category:    "frontend",
		},
		{
			ID:          "state-management",
			Name:        "State Management",
			Type:        "implementation",
			Pattern:     "state|store|redux|context",
			Keywords:    []string{"state", "store", "redux", "context", "management"},
			Complexity:  "high",
			Hours:       12,
			SubTasks:    []string{"design-store", "implement-actions", "connect-components", "add-persistence"},
			Description: "Implement application state management with proper data flow",
			Category:    "frontend",
		},

		// Testing Templates
		{
			ID:          "unit-tests",
			Name:        "Unit Test Suite",
			Type:        "testing",
			Pattern:     "test|testing|unit|spec",
			Keywords:    []string{"test", "testing", "unit", "spec", "jest", "mocha"},
			Complexity:  "low",
			Hours:       4,
			SubTasks:    []string{"setup-tests", "write-test-cases", "add-mocks", "run-coverage"},
			Description: "Create comprehensive unit test suite with good coverage",
			Category:    "testing",
		},
		{
			ID:          "integration-tests",
			Name:        "Integration Tests",
			Type:        "testing",
			Pattern:     "integration|e2e|end-to-end",
			Keywords:    []string{"integration", "e2e", "end-to-end", "cypress", "selenium"},
			Complexity:  "high",
			Hours:       12,
			SubTasks:    []string{"setup-framework", "write-scenarios", "add-fixtures", "run-pipeline"},
			Description: "Implement integration tests for complete user workflows",
			Category:    "testing",
		},

		// DevOps Templates
		{
			ID:          "ci-cd-pipeline",
			Name:        "CI/CD Pipeline",
			Type:        "setup",
			Pattern:     "ci|cd|pipeline|deployment",
			Keywords:    []string{"ci", "cd", "pipeline", "deployment", "github", "actions"},
			Complexity:  "high",
			Hours:       16,
			SubTasks:    []string{"setup-pipeline", "add-tests", "configure-deployment", "monitor"},
			Description: "Set up automated CI/CD pipeline with testing and deployment",
			Category:    "devops",
		},
		{
			ID:          "docker-setup",
			Name:        "Docker Configuration",
			Type:        "setup",
			Pattern:     "docker|container|dockerfile",
			Keywords:    []string{"docker", "container", "dockerfile", "compose", "image"},
			Complexity:  "medium",
			Hours:       6,
			SubTasks:    []string{"create-dockerfile", "setup-compose", "optimize-image", "test-container"},
			Description: "Create Docker configuration for application containerization",
			Category:    "devops",
		},

		// Documentation Templates
		{
			ID:          "api-documentation",
			Name:        "API Documentation",
			Type:        "documentation",
			Pattern:     "documentation|docs|api|swagger",
			Keywords:    []string{"documentation", "docs", "api", "swagger", "openapi"},
			Complexity:  "low",
			Hours:       4,
			SubTasks:    []string{"document-endpoints", "add-examples", "generate-schema", "review"},
			Description: "Create comprehensive API documentation with examples",
			Category:    "documentation",
		},
		{
			ID:          "user-guide",
			Name:        "User Guide",
			Type:        "documentation",
			Pattern:     "guide|manual|tutorial|help",
			Keywords:    []string{"guide", "manual", "tutorial", "help", "user", "instructions"},
			Complexity:  "medium",
			Hours:       8,
			SubTasks:    []string{"outline-guide", "write-sections", "add-screenshots", "review"},
			Description: "Create user guide with step-by-step instructions",
			Category:    "documentation",
		},

		// Security Templates
		{
			ID:          "security-audit",
			Name:        "Security Audit",
			Type:        "audit",
			Pattern:     "security|audit|vulnerability|scan",
			Keywords:    []string{"security", "audit", "vulnerability", "scan", "penetration"},
			Complexity:  "high",
			Hours:       12,
			SubTasks:    []string{"run-scans", "analyze-results", "fix-issues", "document-findings"},
			Description: "Perform comprehensive security audit and fix vulnerabilities",
			Category:    "security",
		},

		// Performance Templates
		{
			ID:          "performance-optimization",
			Name:        "Performance Optimization",
			Type:        "optimization",
			Pattern:     "performance|optimization|speed|cache",
			Keywords:    []string{"performance", "optimization", "speed", "cache", "benchmark"},
			Complexity:  "high",
			Hours:       16,
			SubTasks:    []string{"profile-application", "identify-bottlenecks", "implement-fixes", "measure-improvements"},
			Description: "Optimize application performance through profiling and improvements",
			Category:    "performance",
		},

		// CLI Templates
		{
			ID:          "cli-command",
			Name:        "CLI Command",
			Type:        "implementation",
			Pattern:     "cli|command|tool|script",
			Keywords:    []string{"cli", "command", "tool", "script", "cobra", "flag"},
			Complexity:  "low",
			Hours:       4,
			SubTasks:    []string{"design-command", "implement-logic", "add-flags", "write-help"},
			Description: "Implement CLI command with proper flag handling and help text",
			Category:    "cli",
		},
	}
}
