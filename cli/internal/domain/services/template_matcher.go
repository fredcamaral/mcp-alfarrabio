// Package services provides domain services for the lerian-mcp-memory CLI.
package services

import (
	"errors"
	"log/slog"
	"strconv"

	"lerian-mcp-memory-cli/internal/domain/constants"
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
	templates := make([]*TaskTemplate, 0, len(m.templates))
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
	case constants.SeverityLow:
		complexityMultiplier = 1.2
	case constants.SeverityMedium:
		complexityMultiplier = 1.0
	case constants.SeverityHigh:
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
			Pattern:     "cli|command|terminal|console",
			Keywords:    []string{"cli", "command", "terminal", "console", "cobra", "flags"},
			Complexity:  "medium",
			Hours:       6,
			SubTasks:    []string{"design-command", "implement-logic", "add-flags", "write-help"},
			Description: "Create CLI command with proper flag handling and help text",
			Category:    "tools",
		},

		// Modern Development Templates
		{
			ID:          "microservice",
			Name:        "Microservice Implementation",
			Type:        "implementation",
			Pattern:     "microservice|service|grpc|rest",
			Keywords:    []string{"microservice", "service", "grpc", "rest", "api", "distributed"},
			Complexity:  "high",
			Hours:       24,
			SubTasks:    []string{"design-service", "implement-endpoints", "add-middleware", "setup-monitoring", "write-tests", "add-documentation"},
			Description: "Build microservice with proper API design, monitoring, and testing",
			Category:    "architecture",
		},
		{
			ID:          "kubernetes-deployment",
			Name:        "Kubernetes Deployment",
			Type:        "setup",
			Pattern:     "kubernetes|k8s|helm|deployment",
			Keywords:    []string{"kubernetes", "k8s", "helm", "deployment", "pod", "service"},
			Complexity:  "high",
			Hours:       16,
			SubTasks:    []string{"create-manifests", "setup-helm", "configure-ingress", "add-monitoring", "test-deployment"},
			Description: "Deploy application to Kubernetes with proper configuration and monitoring",
			Category:    "devops",
		},
		{
			ID:          "serverless-function",
			Name:        "Serverless Function",
			Type:        "implementation",
			Pattern:     "serverless|lambda|function|faas",
			Keywords:    []string{"serverless", "lambda", "function", "faas", "aws", "azure"},
			Complexity:  "medium",
			Hours:       8,
			SubTasks:    []string{"design-function", "implement-handler", "setup-triggers", "add-monitoring", "deploy"},
			Description: "Create serverless function with proper event handling and monitoring",
			Category:    "cloud",
		},
		{
			ID:          "graphql-api",
			Name:        "GraphQL API",
			Type:        "implementation",
			Pattern:     "graphql|gql|schema|resolver",
			Keywords:    []string{"graphql", "gql", "schema", "resolver", "query", "mutation"},
			Complexity:  "high",
			Hours:       20,
			SubTasks:    []string{"design-schema", "implement-resolvers", "add-mutations", "setup-subscriptions", "add-validation", "write-tests"},
			Description: "Build GraphQL API with schema design, resolvers, and real-time subscriptions",
			Category:    "backend",
		},

		// AI/ML Templates
		{
			ID:          "ml-pipeline",
			Name:        "ML Training Pipeline",
			Type:        "implementation",
			Pattern:     "ml|machine|learning|pipeline|model",
			Keywords:    []string{"ml", "machine", "learning", "pipeline", "model", "training"},
			Complexity:  "high",
			Hours:       32,
			SubTasks:    []string{"prepare-data", "design-model", "implement-training", "add-validation", "setup-monitoring", "deploy-model"},
			Description: "Create end-to-end ML pipeline from data preparation to model deployment",
			Category:    "ai",
		},
		{
			ID:          "data-analysis",
			Name:        "Data Analysis Workflow",
			Type:        "implementation",
			Pattern:     "data|analysis|analytics|visualization",
			Keywords:    []string{"data", "analysis", "analytics", "visualization", "jupyter", "pandas"},
			Complexity:  "medium",
			Hours:       12,
			SubTasks:    []string{"load-data", "clean-data", "perform-analysis", "create-visualizations", "generate-report"},
			Description: "Implement data analysis workflow with cleaning, analysis, and visualization",
			Category:    "data",
		},

		// Infrastructure as Code Templates
		{
			ID:          "terraform-infrastructure",
			Name:        "Terraform Infrastructure",
			Type:        "setup",
			Pattern:     "terraform|infrastructure|iac|cloud",
			Keywords:    []string{"terraform", "infrastructure", "iac", "cloud", "aws", "provisioning"},
			Complexity:  "high",
			Hours:       20,
			SubTasks:    []string{"design-infrastructure", "write-terraform", "setup-state", "add-modules", "validate-deploy"},
			Description: "Create infrastructure as code using Terraform with proper state management",
			Category:    "infrastructure",
		},
		{
			ID:          "monitoring-setup",
			Name:        "Monitoring and Observability",
			Type:        "setup",
			Pattern:     "monitoring|observability|metrics|logs",
			Keywords:    []string{"monitoring", "observability", "metrics", "logs", "prometheus", "grafana"},
			Complexity:  "high",
			Hours:       16,
			SubTasks:    []string{"setup-metrics", "configure-logging", "create-dashboards", "setup-alerts", "test-monitoring"},
			Description: "Implement comprehensive monitoring with metrics, logs, and alerting",
			Category:    "devops",
		},

		// Testing Framework Templates
		{
			ID:          "test-automation",
			Name:        "Test Automation Framework",
			Type:        "setup",
			Pattern:     "automation|testing|framework|qa",
			Keywords:    []string{"automation", "testing", "framework", "qa", "selenium", "playwright"},
			Complexity:  "high",
			Hours:       24,
			SubTasks:    []string{"setup-framework", "create-page-objects", "implement-tests", "add-reporting", "integrate-ci"},
			Description: "Build automated testing framework with proper reporting and CI integration",
			Category:    "testing",
		},
		{
			ID:          "load-testing",
			Name:        "Performance Load Testing",
			Type:        "testing",
			Pattern:     "load|performance|stress|benchmark",
			Keywords:    []string{"load", "performance", "stress", "benchmark", "jmeter", "k6"},
			Complexity:  "medium",
			Hours:       12,
			SubTasks:    []string{"design-scenarios", "setup-tools", "run-tests", "analyze-results", "optimize"},
			Description: "Implement load testing to identify performance bottlenecks and limits",
			Category:    "testing",
		},

		// Security Templates
		{
			ID:          "oauth-integration",
			Name:        "OAuth2/OIDC Integration",
			Type:        "implementation",
			Pattern:     "oauth|oidc|sso|authentication",
			Keywords:    []string{"oauth", "oidc", "sso", "authentication", "openid", "jwt"},
			Complexity:  "high",
			Hours:       16,
			SubTasks:    []string{"setup-provider", "implement-flow", "add-middleware", "handle-tokens", "test-integration"},
			Description: "Integrate OAuth2/OIDC authentication with proper token handling",
			Category:    "security",
		},
		{
			ID:          "security-hardening",
			Name:        "Security Hardening",
			Type:        "enhancement",
			Pattern:     "security|hardening|vulnerability|protection",
			Keywords:    []string{"security", "hardening", "vulnerability", "protection", "encryption", "firewall"},
			Complexity:  "high",
			Hours:       20,
			SubTasks:    []string{"security-scan", "implement-fixes", "add-encryption", "setup-monitoring", "verify-hardening"},
			Description: "Implement security hardening measures and vulnerability fixes",
			Category:    "security",
		},

		// Database Templates
		{
			ID:          "database-migration",
			Name:        "Database Migration",
			Type:        "maintenance",
			Pattern:     "migration|database|schema|upgrade",
			Keywords:    []string{"migration", "database", "schema", "upgrade", "flyway", "liquibase"},
			Complexity:  "medium",
			Hours:       8,
			SubTasks:    []string{"plan-migration", "write-scripts", "test-migration", "backup-data", "execute-migration"},
			Description: "Plan and execute database schema migration with proper backup and rollback",
			Category:    "data",
		},
		{
			ID:          "data-backup",
			Name:        "Data Backup Strategy",
			Type:        "setup",
			Pattern:     "backup|recovery|disaster|restore",
			Keywords:    []string{"backup", "recovery", "disaster", "restore", "snapshot", "archive"},
			Complexity:  "high",
			Hours:       12,
			SubTasks:    []string{"design-strategy", "implement-backup", "test-restore", "automate-process", "monitor-backups"},
			Description: "Implement automated backup and disaster recovery strategy",
			Category:    "data",
		},

		// Mobile Development Templates
		{
			ID:          "mobile-app",
			Name:        "Mobile Application",
			Type:        "implementation",
			Pattern:     "mobile|app|ios|android|react-native",
			Keywords:    []string{"mobile", "app", "ios", "android", "react-native", "flutter"},
			Complexity:  "high",
			Hours:       40,
			SubTasks:    []string{"design-ui", "implement-features", "add-navigation", "integrate-apis", "test-devices", "deploy-stores"},
			Description: "Build cross-platform mobile application with native features",
			Category:    "mobile",
		},

		// Code Quality Templates
		{
			ID:          "code-review",
			Name:        "Code Review Process",
			Type:        "process",
			Pattern:     "review|quality|standards|linting",
			Keywords:    []string{"review", "quality", "standards", "linting", "prettier", "eslint"},
			Complexity:  "medium",
			Hours:       8,
			SubTasks:    []string{"setup-linting", "configure-rules", "add-pre-commit", "document-standards", "train-team"},
			Description: "Establish code review process with automated quality checks",
			Category:    "process",
		},
		{
			ID:          "refactoring",
			Name:        "Code Refactoring",
			Type:        "enhancement",
			Pattern:     "refactor|cleanup|debt|improvement",
			Keywords:    []string{"refactor", "cleanup", "debt", "improvement", "architecture", "design"},
			Complexity:  "high",
			Hours:       20,
			SubTasks:    []string{"analyze-code", "plan-refactoring", "implement-changes", "update-tests", "verify-functionality"},
			Description: "Refactor codebase to improve maintainability and reduce technical debt",
			Category:    "maintenance",
		},

		// AI-Powered Code Analysis Templates (inspired by ai-prompts structure)
		{
			ID:          "codebase-overview-analysis",
			Name:        "Comprehensive Codebase Analysis",
			Type:        "analysis",
			Pattern:     "overview|codebase|analysis|architecture|component",
			Keywords:    []string{"overview", "codebase", "analysis", "architecture", "component", "mapping", "tech-stack"},
			Complexity:  "medium",
			Hours:       4,
			SubTasks:    []string{"scan-tech-stack", "map-components", "identify-patterns", "document-architecture", "create-diagrams"},
			Description: "Create comprehensive codebase overview with component mapping and architectural insights",
			Category:    "analysis",
		},
		{
			ID:          "security-vulnerability-audit",
			Name:        "Security Vulnerability Analysis",
			Type:        "audit",
			Pattern:     "security|vulnerability|audit|scan|penetration",
			Keywords:    []string{"security", "vulnerability", "audit", "scan", "penetration", "owasp", "injection"},
			Complexity:  "high",
			Hours:       12,
			SubTasks:    []string{"scan-vulnerabilities", "analyze-attack-surface", "assess-dependencies", "create-remediation-plan", "implement-fixes"},
			Description: "Comprehensive security audit with vulnerability assessment and remediation roadmap",
			Category:    "security",
		},
		{
			ID:          "performance-optimization-analysis",
			Name:        "Performance Analysis & Optimization",
			Type:        "optimization",
			Pattern:     "performance|optimization|bottleneck|profiling|benchmark",
			Keywords:    []string{"performance", "optimization", "bottleneck", "profiling", "benchmark", "latency", "throughput"},
			Complexity:  "high",
			Hours:       16,
			SubTasks:    []string{"profile-application", "identify-bottlenecks", "benchmark-current-state", "implement-optimizations", "measure-improvements"},
			Description: "Deep performance analysis with optimization recommendations and implementation",
			Category:    "performance",
		},
		{
			ID:          "business-logic-analysis",
			Name:        "Business Logic & Workflow Analysis",
			Type:        "analysis",
			Pattern:     "business|workflow|logic|gap|roi|improvement",
			Keywords:    []string{"business", "workflow", "logic", "gap", "roi", "improvement", "process", "efficiency"},
			Complexity:  "medium",
			Hours:       8,
			SubTasks:    []string{"map-business-flows", "identify-gaps", "analyze-performance", "calculate-roi", "create-improvement-plan"},
			Description: "Analyze business workflows and identify optimization opportunities with ROI assessment",
			Category:    "business",
		},
		{
			ID:          "api-contract-analysis",
			Name:        "API Contract & Documentation Analysis",
			Type:        "analysis",
			Pattern:     "api|contract|documentation|consistency|endpoint",
			Keywords:    []string{"api", "contract", "documentation", "consistency", "endpoint", "swagger", "openapi"},
			Complexity:  "medium",
			Hours:       6,
			SubTasks:    []string{"inventory-endpoints", "check-consistency", "analyze-documentation", "validate-contracts", "generate-improvements"},
			Description: "Comprehensive API analysis with contract validation and documentation assessment",
			Category:    "api",
		},
		{
			ID:          "database-optimization-audit",
			Name:        "Database Performance & Schema Analysis",
			Type:        "optimization",
			Pattern:     "database|schema|query|optimization|index",
			Keywords:    []string{"database", "schema", "query", "optimization", "index", "performance", "slow-query"},
			Complexity:  "high",
			Hours:       10,
			SubTasks:    []string{"analyze-schema", "identify-slow-queries", "review-indexes", "optimize-queries", "implement-improvements"},
			Description: "Database performance analysis with query optimization and schema recommendations",
			Category:    "database",
		},
		{
			ID:          "production-readiness-audit",
			Name:        "Production Readiness Assessment",
			Type:        "audit",
			Pattern:     "production|readiness|deployment|monitoring|reliability",
			Keywords:    []string{"production", "readiness", "deployment", "monitoring", "reliability", "checklist", "scalability"},
			Complexity:  "high",
			Hours:       14,
			SubTasks:    []string{"assess-monitoring", "check-scalability", "verify-security", "validate-deployment", "create-runbooks"},
			Description: "Comprehensive production readiness audit with deployment checklist and operational procedures",
			Category:    "deployment",
		},
		{
			ID:          "code-quality-assessment",
			Name:        "Advanced Code Quality Analysis",
			Type:        "analysis",
			Pattern:     "quality|maintainability|complexity|debt|smell",
			Keywords:    []string{"quality", "maintainability", "complexity", "debt", "smell", "duplication", "coverage"},
			Complexity:  "medium",
			Hours:       6,
			SubTasks:    []string{"analyze-complexity", "detect-code-smells", "measure-duplication", "assess-test-coverage", "create-improvement-plan"},
			Description: "Advanced code quality analysis with maintainability assessment and improvement recommendations",
			Category:    "quality",
		},
		{
			ID:          "dependency-security-analysis",
			Name:        "Dependency Security & License Audit",
			Type:        "audit",
			Pattern:     "dependency|security|license|vulnerability|compliance",
			Keywords:    []string{"dependency", "security", "license", "vulnerability", "compliance", "audit", "third-party"},
			Complexity:  "medium",
			Hours:       4,
			SubTasks:    []string{"scan-dependencies", "check-vulnerabilities", "analyze-licenses", "assess-compliance", "create-update-plan"},
			Description: "Comprehensive dependency audit covering security vulnerabilities and license compliance",
			Category:    "security",
		},
		{
			ID:          "architecture-decision-documentation",
			Name:        "Architecture Decision Records (ADR)",
			Type:        "documentation",
			Pattern:     "adr|decision|architecture|design|rationale",
			Keywords:    []string{"adr", "decision", "architecture", "design", "rationale", "trade-off", "consequence"},
			Complexity:  "low",
			Hours:       3,
			SubTasks:    []string{"identify-decisions", "document-rationale", "analyze-trade-offs", "record-consequences", "create-adr-process"},
			Description: "Create and maintain Architecture Decision Records for key design decisions",
			Category:    "documentation",
		},
	}
}
