// Package tasks provides complexity analysis for task generation.
package tasks

import (
	"context"
	"errors"
	"math"
	"strings"

	"lerian-mcp-memory/pkg/types"
)

// ComplexityAnalyzer analyzes task complexity using multiple factors
type ComplexityAnalyzer struct {
	config ComplexityConfig
}

// ComplexityConfig represents configuration for complexity analysis
type ComplexityConfig struct {
	TechnicalWeights   ComplexityWeights     `json:"technical_weights"`
	IntegrationWeights ComplexityWeights     `json:"integration_weights"`
	BusinessWeights    ComplexityWeights     `json:"business_weights"`
	DefaultRiskLevel   types.RiskLevelEnum   `json:"default_risk_level"`
	DefaultImpactLevel types.ImpactLevelEnum `json:"default_impact_level"`
}

// ComplexityWeights represents weights for different complexity factors
type ComplexityWeights struct {
	TechnicalComplexity     float64 `json:"technical_complexity"`
	IntegrationComplexity   float64 `json:"integration_complexity"`
	BusinessLogicComplexity float64 `json:"business_logic_complexity"`
	DataComplexity          float64 `json:"data_complexity"`
	UIComplexity            float64 `json:"ui_complexity"`
	TestingComplexity       float64 `json:"testing_complexity"`
}

// DefaultComplexityWeights returns default complexity weights
func DefaultComplexityWeights() ComplexityWeights {
	return ComplexityWeights{
		TechnicalComplexity:     0.25,
		IntegrationComplexity:   0.20,
		BusinessLogicComplexity: 0.20,
		DataComplexity:          0.15,
		UIComplexity:            0.10,
		TestingComplexity:       0.10,
	}
}

// DefaultComplexityConfig returns default complexity configuration
func DefaultComplexityConfig() ComplexityConfig {
	weights := DefaultComplexityWeights()
	return ComplexityConfig{
		TechnicalWeights:   weights,
		IntegrationWeights: weights,
		BusinessWeights:    weights,
		DefaultRiskLevel:   types.RiskLevelMedium,
		DefaultImpactLevel: types.ImpactLevelMedium,
	}
}

// NewComplexityAnalyzer creates a new complexity analyzer
func NewComplexityAnalyzer() *ComplexityAnalyzer {
	return &ComplexityAnalyzer{
		config: DefaultComplexityConfig(),
	}
}

// NewComplexityAnalyzerWithConfig creates a new complexity analyzer with custom config
func NewComplexityAnalyzerWithConfig(config *ComplexityConfig) *ComplexityAnalyzer {
	return &ComplexityAnalyzer{
		config: *config,
	}
}

// AnalyzeComplexity analyzes the complexity of a task
func (ca *ComplexityAnalyzer) AnalyzeComplexity(ctx context.Context, task *types.Task, projectContext *types.TaskGenerationContext) (types.TaskComplexity, error) {
	if task == nil {
		return types.TaskComplexity{}, errors.New("task cannot be nil")
	}

	// Calculate complexity factors
	factors := ca.calculateComplexityFactors(task, projectContext)

	// Calculate overall complexity score
	score := ca.calculateOverallScore(factors)

	// Determine complexity level
	level := ca.determineComplexityLevel(score)

	// Assess technical risk
	technicalRisk := ca.assessTechnicalRisk(task, factors, projectContext)

	// Assess business impact
	businessImpactLegacy := ca.assessBusinessImpact(task, projectContext)
	businessImpact := types.ImpactLevelEnum(businessImpactLegacy)

	// Extract required skills
	requiredSkills := ca.extractRequiredSkills(task, projectContext)

	// Identify external dependencies
	externalDeps := ca.identifyExternalDependencies(task, projectContext)

	return types.TaskComplexity{
		Level:                level,
		Score:                score,
		Factors:              factors,
		TechnicalRisk:        technicalRisk,
		BusinessImpact:       businessImpact,
		RequiredSkills:       requiredSkills,
		ExternalDependencies: externalDeps,
	}, nil
}

// calculateComplexityFactors calculates individual complexity factors
func (ca *ComplexityAnalyzer) calculateComplexityFactors(task *types.Task, projectContext *types.TaskGenerationContext) types.ComplexityFactors {
	return types.ComplexityFactors{
		TechnicalComplexity:     ca.calculateTechnicalComplexity(task, projectContext),
		IntegrationComplexity:   ca.calculateIntegrationComplexity(task, projectContext),
		BusinessLogicComplexity: ca.calculateBusinessLogicComplexity(task, projectContext),
		DataComplexity:          ca.calculateDataComplexity(task, projectContext),
		UIComplexity:            ca.calculateUIComplexity(task, projectContext),
		TestingComplexity:       ca.calculateTestingComplexity(task, projectContext),
	}
}

// calculateTechnicalComplexity calculates technical complexity score
func (ca *ComplexityAnalyzer) calculateTechnicalComplexity(task *types.Task, projectContext *types.TaskGenerationContext) float64 {
	score := 0.0

	// Task type influences technical complexity
	switch task.Type {
	case types.TaskTypeLegacyArchitecture:
		score += 0.8
	case types.TaskTypeLegacyImplementation:
		score += 0.6
	case types.TaskTypeLegacyIntegration:
		score += 0.7
	case types.TaskTypeLegacyRefactoring:
		score += 0.5
	case types.TaskTypeLegacyDesign:
		score += 0.4
	case types.TaskTypeLegacyTesting:
		score += 0.3
	case types.TaskTypeLegacyDocumentation:
		score += 0.2
	default:
		score += 0.4
	}

	// Keywords in title and description that indicate technical complexity
	techKeywords := []string{
		"algorithm", "optimization", "performance", "scalability", "security",
		"encryption", "authentication", "authorization", "caching", "database",
		"distributed", "microservices", "api", "protocol", "framework",
		"infrastructure", "deployment", "ci/cd", "monitoring", "logging",
		"analytics", "machine learning", "ai", "blockchain", "real-time",
	}

	content := strings.ToLower(task.Title + " " + task.Description)
	for _, keyword := range techKeywords {
		if strings.Contains(content, keyword) {
			score += 0.1
		}
	}

	// Technology stack complexity
	complexTech := []string{
		"kubernetes", "docker", "aws", "azure", "gcp", "terraform",
		"react", "angular", "vue", "nodejs", "python", "go", "rust",
		"postgresql", "mongodb", "redis", "elasticsearch", "kafka",
	}

	for _, tech := range projectContext.TechStack {
		for _, complex := range complexTech {
			if strings.Contains(strings.ToLower(tech), complex) {
				score += 0.05
			}
		}
	}

	// Normalize to 0-1 range
	return math.Min(score, 1.0)
}

// calculateIntegrationComplexity calculates integration complexity score
func (ca *ComplexityAnalyzer) calculateIntegrationComplexity(task *types.Task, projectContext *types.TaskGenerationContext) float64 {
	_ = projectContext // unused parameter, kept for potential future context-aware complexity calculation
	score := 0.0

	// Task type influences integration complexity
	if task.Type == types.TaskTypeLegacyIntegration {
		score += 0.8
	}

	// Keywords that indicate integration complexity
	integrationKeywords := []string{
		"api", "integration", "webhook", "sync", "async", "queue",
		"message", "event", "stream", "protocol", "connector",
		"third-party", "external", "service", "endpoint", "gateway",
	}

	content := strings.ToLower(task.Title + " " + task.Description)
	for _, keyword := range integrationKeywords {
		if strings.Contains(content, keyword) {
			score += 0.15
		}
	}

	// Number of dependencies indicates integration complexity
	if len(task.Dependencies) > 3 {
		score += 0.3
	} else if len(task.Dependencies) > 1 {
		score += 0.15
	}

	// External dependencies increase integration complexity
	externalKeywords := []string{"external", "third-party", "vendor", "saas", "cloud"}
	for _, keyword := range externalKeywords {
		if strings.Contains(content, keyword) {
			score += 0.2
		}
	}

	return math.Min(score, 1.0)
}

// calculateBusinessLogicComplexity calculates business logic complexity score
func (ca *ComplexityAnalyzer) calculateBusinessLogicComplexity(task *types.Task, projectContext *types.TaskGenerationContext) float64 {
	_ = projectContext // unused parameter, kept for potential future context-aware complexity calculation
	score := 0.0

	// Business logic keywords
	businessKeywords := []string{
		"workflow", "process", "business rule", "validation", "calculation",
		"algorithm", "logic", "condition", "rule", "policy", "compliance",
		"audit", "reporting", "analytics", "dashboard", "notification",
		"approval", "routing", "scheduling", "billing", "payment",
	}

	content := strings.ToLower(task.Title + " " + task.Description)
	for _, keyword := range businessKeywords {
		if strings.Contains(content, keyword) {
			score += 0.1
		}
	}

	// Acceptance criteria complexity
	switch {
	case len(task.AcceptanceCriteria) > 5:
		score += 0.3
	case len(task.AcceptanceCriteria) > 3:
		score += 0.2
	case len(task.AcceptanceCriteria) > 1:
		score += 0.1
	}

	// Complex acceptance criteria patterns
	complexPatterns := []string{
		"if", "when", "unless", "provided", "given", "depending",
		"multiple", "various", "different", "configurable", "customizable",
	}

	for _, criteria := range task.AcceptanceCriteria {
		criteriaLower := strings.ToLower(criteria)
		for _, pattern := range complexPatterns {
			if strings.Contains(criteriaLower, pattern) {
				score += 0.05
			}
		}
	}

	return math.Min(score, 1.0)
}

// calculateDataComplexity calculates data complexity score
func (ca *ComplexityAnalyzer) calculateDataComplexity(task *types.Task, projectContext *types.TaskGenerationContext) float64 {
	_ = projectContext // unused parameter, kept for potential future context-aware complexity calculation
	score := 0.0

	// Data-related keywords
	dataKeywords := []string{
		"database", "data", "model", "schema", "migration", "query",
		"orm", "sql", "nosql", "index", "performance", "optimization",
		"backup", "restore", "sync", "replication", "sharding",
		"transaction", "acid", "consistency", "integrity", "validation",
	}

	content := strings.ToLower(task.Title + " " + task.Description)
	for _, keyword := range dataKeywords {
		if strings.Contains(content, keyword) {
			score += 0.1
		}
	}

	// Complex data operations
	complexDataOps := []string{
		"migration", "transformation", "etl", "aggregation", "analytics",
		"reporting", "big data", "real-time", "streaming", "batch",
	}

	for _, op := range complexDataOps {
		if strings.Contains(content, op) {
			score += 0.15
		}
	}

	return math.Min(score, 1.0)
}

// calculateUIComplexity calculates UI complexity score
func (ca *ComplexityAnalyzer) calculateUIComplexity(task *types.Task, projectContext *types.TaskGenerationContext) float64 {
	_ = projectContext // unused parameter, kept for potential future context-aware complexity calculation
	score := 0.0

	// UI-related task types
	if task.Type == types.TaskTypeLegacyDesign {
		score += 0.5
	}

	// UI keywords
	uiKeywords := []string{
		"ui", "ux", "interface", "frontend", "component", "layout",
		"design", "responsive", "mobile", "accessibility", "animation",
		"interaction", "navigation", "form", "validation", "modal",
		"dropdown", "carousel", "chart", "graph", "visualization",
	}

	content := strings.ToLower(task.Title + " " + task.Description)
	for _, keyword := range uiKeywords {
		if strings.Contains(content, keyword) {
			score += 0.1
		}
	}

	// Complex UI features
	complexUIFeatures := []string{
		"drag and drop", "real-time", "collaborative", "multi-step",
		"wizard", "complex form", "data visualization", "dashboard",
		"rich editor", "interactive", "dynamic", "customizable",
	}

	for _, feature := range complexUIFeatures {
		if strings.Contains(content, feature) {
			score += 0.2
		}
	}

	return math.Min(score, 1.0)
}

// calculateTestingComplexity calculates testing complexity score
func (ca *ComplexityAnalyzer) calculateTestingComplexity(task *types.Task, projectContext *types.TaskGenerationContext) float64 {
	_ = projectContext // unused parameter, kept for potential future context-aware complexity calculation
	score := 0.0

	// Testing task type
	if task.Type == types.TaskTypeLegacyTesting {
		score += 0.6
	}

	// Testing keywords
	testingKeywords := []string{
		"test", "testing", "qa", "quality", "unit test", "integration test",
		"e2e", "automation", "performance test", "load test", "security test",
		"accessibility test", "regression", "smoke test", "acceptance test",
	}

	content := strings.ToLower(task.Title + " " + task.Description)
	for _, keyword := range testingKeywords {
		if strings.Contains(content, keyword) {
			score += 0.1
		}
	}

	// Complex testing scenarios
	complexTestScenarios := []string{
		"end-to-end", "performance", "load", "stress", "security",
		"penetration", "accessibility", "cross-browser", "multi-device",
		"automation", "continuous", "regression", "compatibility",
	}

	for _, scenario := range complexTestScenarios {
		if strings.Contains(content, scenario) {
			score += 0.15
		}
	}

	return math.Min(score, 1.0)
}

// calculateOverallScore calculates the overall complexity score
func (ca *ComplexityAnalyzer) calculateOverallScore(factors types.ComplexityFactors) float64 {
	weights := ca.config.TechnicalWeights

	score := (factors.TechnicalComplexity*weights.TechnicalComplexity +
		factors.IntegrationComplexity*weights.IntegrationComplexity +
		factors.BusinessLogicComplexity*weights.BusinessLogicComplexity +
		factors.DataComplexity*weights.DataComplexity +
		factors.UIComplexity*weights.UIComplexity +
		factors.TestingComplexity*weights.TestingComplexity)

	return math.Min(score, 1.0)
}

// determineComplexityLevel determines complexity level from score
func (ca *ComplexityAnalyzer) determineComplexityLevel(score float64) types.ComplexityLevel {
	switch {
	case score >= 0.8:
		return types.ComplexityVeryComplex
	case score >= 0.6:
		return types.ComplexityComplex
	case score >= 0.4:
		return types.ComplexityModerate
	case score >= 0.2:
		return types.ComplexitySimple
	default:
		return types.ComplexityTrivial
	}
}

// assessTechnicalRisk assesses technical risk level
func (ca *ComplexityAnalyzer) assessTechnicalRisk(task *types.Task, factors types.ComplexityFactors, projectContext *types.TaskGenerationContext) types.RiskLevelEnum {
	_ = projectContext // unused parameter, kept for potential future context-aware risk assessment
	riskScore := 0.0

	// High technical complexity increases risk
	riskScore += factors.TechnicalComplexity * 0.4

	// Integration complexity adds risk
	riskScore += factors.IntegrationComplexity * 0.3

	// Data complexity adds risk
	riskScore += factors.DataComplexity * 0.2

	// Unknown or cutting-edge technologies increase risk
	riskTech := []string{"new", "experimental", "beta", "alpha", "cutting-edge", "prototype"}
	content := strings.ToLower(task.Title + " " + task.Description)
	for _, tech := range riskTech {
		if strings.Contains(content, tech) {
			riskScore += 0.2
		}
	}

	// Determine risk level
	switch {
	case riskScore >= 0.7:
		return types.RiskLevelCritical
	case riskScore >= 0.5:
		return types.RiskLevelHigh
	case riskScore >= 0.3:
		return types.RiskLevelMedium
	default:
		return types.RiskLevelLow
	}
}

// assessBusinessImpact assesses business impact level
func (ca *ComplexityAnalyzer) assessBusinessImpact(task *types.Task, _ *types.TaskGenerationContext) types.ImpactLevel {
	impactScore := 0.0

	// Priority influences business impact
	switch task.Priority {
	case types.TaskPriorityLegacyCritical, types.TaskPriorityLegacyBlocking:
		impactScore += 0.8
	case types.TaskPriorityLegacyHigh:
		impactScore += 0.6
	case types.TaskPriorityLegacyMedium:
		impactScore += 0.4
	case types.TaskPriorityLegacyLow:
		impactScore += 0.2
	}

	// Business impact keywords
	impactKeywords := []string{
		"revenue", "customer", "user", "business", "critical", "urgent",
		"blocking", "compliance", "security", "performance", "scalability",
		"launch", "release", "milestone", "deadline", "strategic",
	}

	content := strings.ToLower(task.Title + " " + task.Description)
	for _, keyword := range impactKeywords {
		if strings.Contains(content, keyword) {
			impactScore += 0.1
		}
	}

	// Determine impact level
	switch {
	case impactScore >= 0.7:
		return types.ImpactLegacyCritical
	case impactScore >= 0.5:
		return types.ImpactLegacyHigh
	case impactScore >= 0.3:
		return types.ImpactLegacyMedium
	default:
		return types.ImpactLegacyLow
	}
}

// extractRequiredSkills extracts required skills from task content
func (ca *ComplexityAnalyzer) extractRequiredSkills(task *types.Task, _ *types.TaskGenerationContext) []string {
	skills := make(map[string]bool)

	// Add existing required skills
	for _, skill := range task.Complexity.RequiredSkills {
		skills[skill] = true
	}

	// Skill keywords mapping
	skillKeywords := map[string][]string{
		"Frontend Development": {"react", "angular", "vue", "javascript", "typescript", "html", "css", "frontend"},
		"Backend Development":  {"nodejs", "python", "java", "go", "rust", "php", "backend", "server"},
		"Database":             {"sql", "postgresql", "mysql", "mongodb", "redis", "database", "orm"},
		"DevOps":               {"docker", "kubernetes", "aws", "azure", "gcp", "terraform", "ansible", "ci/cd"},
		"Mobile Development":   {"react native", "flutter", "ios", "android", "mobile", "app"},
		"Testing":              {"testing", "qa", "automation", "selenium", "cypress", "jest", "pytest"},
		"Security":             {"security", "authentication", "authorization", "encryption", "oauth", "jwt"},
		"UI/UX Design":         {"design", "ui", "ux", "figma", "sketch", "adobe", "prototype"},
		"Data Science":         {"machine learning", "ai", "analytics", "data science", "python", "r"},
		"API Development":      {"api", "rest", "graphql", "microservices", "grpc", "websocket"},
	}

	content := strings.ToLower(task.Title + " " + task.Description)

	for skill, keywords := range skillKeywords {
		for _, keyword := range keywords {
			if strings.Contains(content, keyword) {
				skills[skill] = true
				break
			}
		}
	}

	// Convert to slice
	result := make([]string, 0, len(skills))
	for skill := range skills {
		result = append(result, skill)
	}

	return result
}

// identifyExternalDependencies identifies external dependencies
func (ca *ComplexityAnalyzer) identifyExternalDependencies(task *types.Task, _ *types.TaskGenerationContext) []string {
	deps := make(map[string]bool)

	// External dependency keywords
	externalKeywords := map[string][]string{
		"Third-party APIs":  {"api", "third-party", "external api", "webhook", "integration"},
		"Cloud Services":    {"aws", "azure", "gcp", "cloud", "saas", "paas"},
		"Payment Systems":   {"payment", "stripe", "paypal", "billing", "checkout"},
		"Authentication":    {"oauth", "auth0", "okta", "saml", "sso", "ldap"},
		"Analytics":         {"analytics", "google analytics", "mixpanel", "amplitude"},
		"Monitoring":        {"monitoring", "logging", "sentry", "datadog", "newrelic"},
		"Communication":     {"email", "sms", "notifications", "sendgrid", "twilio"},
		"Storage":           {"s3", "storage", "cdn", "cloudfront", "cloudflare"},
		"Database Services": {"rds", "dynamodb", "cosmos", "atlas", "database service"},
		"Development Tools": {"github", "gitlab", "jenkins", "docker hub", "npm"},
	}

	content := strings.ToLower(task.Title + " " + task.Description)

	for dep, keywords := range externalKeywords {
		for _, keyword := range keywords {
			if strings.Contains(content, keyword) {
				deps[dep] = true
				break
			}
		}
	}

	// Convert to slice
	result := make([]string, 0, len(deps))
	for dep := range deps {
		result = append(result, dep)
	}

	return result
}
