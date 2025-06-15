// Package services provides domain services for the lerian-mcp-memory CLI.
package services

import (
	"errors"
	"log/slog"
	"strconv"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
)

// ComplexityAnalyzer analyzes the complexity of tasks and content
type ComplexityAnalyzer interface {
	AnalyzeContent(content string) (*ComplexityAnalysis, error)
	AnalyzeTask(task *entities.Task) (*ComplexityAnalysis, error)
	EstimateEffort(complexity *ComplexityAnalysis) int
	CalculateRiskScore(content string) float64
	ExtractRequiredSkills(content string) []string
}

// ComplexityAnalysis represents the result of complexity analysis
type ComplexityAnalysis struct {
	Score           float64             `json:"score"` // 0-10
	Level           string              `json:"level"` // low, medium, high
	Factors         []ComplexityFactor  `json:"factors"`
	EstimatedHours  int                 `json:"estimated_hours"`
	Confidence      float64             `json:"confidence"` // 0-1
	Breakdown       ComplexityBreakdown `json:"breakdown"`
	Recommendations []string            `json:"recommendations"`
	RiskScore       float64             `json:"risk_score"` // 0-10
	RequiredSkills  []string            `json:"required_skills"`
}

// ComplexityFactor represents a factor contributing to complexity
type ComplexityFactor struct {
	Name        string  `json:"name"`
	Impact      string  `json:"impact"` // low, medium, high
	Score       float64 `json:"score"`  // 0-10
	Description string  `json:"description"`
	Weight      float64 `json:"weight"` // 0-1
}

// ComplexityBreakdown provides detailed complexity breakdown
type ComplexityBreakdown struct {
	Technical      float64 `json:"technical"`       // 0-10
	Business       float64 `json:"business"`        // 0-10
	Integration    float64 `json:"integration"`     // 0-10
	Testing        float64 `json:"testing"`         // 0-10
	Documentation  float64 `json:"documentation"`   // 0-10
	DataComplexity float64 `json:"data_complexity"` // 0-10
	UIComplexity   float64 `json:"ui_complexity"`   // 0-10
}

// ComplexityWeights defines weights for different complexity factors
type ComplexityWeights struct {
	Technical     float64 `json:"technical"`
	Business      float64 `json:"business"`
	Integration   float64 `json:"integration"`
	Testing       float64 `json:"testing"`
	Documentation float64 `json:"documentation"`
	Data          float64 `json:"data"`
	UI            float64 `json:"ui"`
}

// ComplexityPattern represents patterns that contribute to complexity
type ComplexityPattern struct {
	Keywords    []string `json:"keywords"`
	Impact      float64  `json:"impact"` // multiplier for complexity
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Weight      float64  `json:"weight"`
}

// DefaultComplexityAnalyzer implements ComplexityAnalyzer
type DefaultComplexityAnalyzer struct {
	patterns map[string]ComplexityPattern
	weights  ComplexityWeights
	logger   *slog.Logger
}

// NewComplexityAnalyzer creates a new complexity analyzer
func NewComplexityAnalyzer(logger *slog.Logger) *DefaultComplexityAnalyzer {
	return &DefaultComplexityAnalyzer{
		patterns: loadComplexityPatterns(),
		weights:  getDefaultWeights(),
		logger:   logger,
	}
}

// AnalyzeContent analyzes the complexity of given content
func (a *DefaultComplexityAnalyzer) AnalyzeContent(content string) (*ComplexityAnalysis, error) {
	if content == "" {
		return nil, errors.New("content cannot be empty")
	}

	contentLower := toLowerCase(content)
	words := splitWords(content)

	var factors []ComplexityFactor
	breakdown := ComplexityBreakdown{}

	// Analyze different complexity dimensions
	breakdown.Technical = a.analyzeTechnicalComplexity(contentLower, words)
	breakdown.Business = a.analyzeBusinessComplexity(contentLower, words)
	breakdown.Integration = a.analyzeIntegrationComplexity(contentLower, words)
	breakdown.Testing = a.analyzeTestingComplexity(contentLower, words)
	breakdown.Documentation = a.analyzeDocumentationComplexity(contentLower, words)
	breakdown.DataComplexity = a.analyzeDataComplexity(contentLower, words)
	breakdown.UIComplexity = a.analyzeUIComplexity(contentLower, words)

	// Add factors for high complexity areas
	factors = a.extractComplexityFactors(breakdown, contentLower)

	// Calculate overall score using weighted average
	overallScore := a.calculateOverallScore(breakdown)

	// Determine complexity level
	level := a.scoreToLevel(overallScore)

	// Estimate hours based on complexity
	estimatedHours := a.scoreToHours(overallScore, len(words))

	// Calculate confidence based on content length and pattern matches
	confidence := a.calculateConfidence(factors, len(words))

	// Calculate risk score
	riskScore := a.CalculateRiskScore(content)

	// Extract required skills
	requiredSkills := a.ExtractRequiredSkills(content)

	// Generate recommendations
	recommendations := a.generateRecommendations(breakdown, factors, level)

	analysis := &ComplexityAnalysis{
		Score:           overallScore,
		Level:           level,
		Factors:         factors,
		EstimatedHours:  estimatedHours,
		Confidence:      confidence,
		Breakdown:       breakdown,
		Recommendations: recommendations,
		RiskScore:       riskScore,
		RequiredSkills:  requiredSkills,
	}

	a.logger.Debug("completed complexity analysis",
		slog.String("level", level),
		slog.String("score", strconv.FormatFloat(overallScore, 'f', 2, 64)),
		slog.Int("hours", estimatedHours),
		slog.Int("factors", len(factors)))

	return analysis, nil
}

// AnalyzeTask analyzes the complexity of a task entity
func (a *DefaultComplexityAnalyzer) AnalyzeTask(task *entities.Task) (*ComplexityAnalysis, error) {
	if task == nil {
		return nil, errors.New("task cannot be nil")
	}

	// Combine task content and tags for analysis
	content := task.Content
	if len(task.Tags) > 0 {
		content += " " + joinWords(task.Tags)
	}

	analysis, err := a.AnalyzeContent(content)
	if err != nil {
		return nil, err
	}

	// Adjust analysis based on task estimates
	taskEstimatedHours := task.EstimatedMins / 60
	if taskEstimatedHours > 0 && analysis.EstimatedHours != taskEstimatedHours {
		// Use existing estimate as a signal for complexity adjustment
		ratio := float64(taskEstimatedHours) / float64(analysis.EstimatedHours)
		if ratio > 1.5 || ratio < 0.5 {
			// Significant difference, adjust score
			analysis.Score *= ratio
			analysis.EstimatedHours = taskEstimatedHours
			analysis.Level = a.scoreToLevel(analysis.Score)
		}
	}

	return analysis, nil
}

// EstimateEffort estimates the effort required based on complexity analysis
func (a *DefaultComplexityAnalyzer) EstimateEffort(complexity *ComplexityAnalysis) int {
	if complexity == nil {
		return 4 // Default to 4 hours
	}

	baseHours := complexity.EstimatedHours

	// Adjust based on risk score
	riskMultiplier := 1.0 + (complexity.RiskScore / 20.0) // 0-50% increase based on risk

	// Adjust based on confidence
	confidenceMultiplier := 1.0
	if complexity.Confidence < 0.7 {
		confidenceMultiplier = 1.3 // 30% buffer for low confidence
	} else if complexity.Confidence < 0.5 {
		confidenceMultiplier = 1.5 // 50% buffer for very low confidence
	}

	adjustedHours := float64(baseHours) * riskMultiplier * confidenceMultiplier

	// Round to reasonable values
	finalHours := int(adjustedHours + 0.5) // Round to nearest hour

	// Ensure reasonable bounds
	if finalHours < 1 {
		finalHours = 1
	} else if finalHours > 80 {
		finalHours = 80
	}

	return finalHours
}

// CalculateRiskScore calculates the risk score for the given content
func (a *DefaultComplexityAnalyzer) CalculateRiskScore(content string) float64 {
	contentLower := toLowerCase(content)
	riskScore := 0.0

	// High-risk patterns
	highRiskPatterns := []string{
		"security", "authentication", "authorization", "encryption", "payment",
		"migration", "database", "performance", "scalability", "distributed",
		"concurrent", "parallel", "real-time", "sync", "integration",
		"third-party", "external", "legacy", "refactor", "breaking",
	}

	for _, pattern := range highRiskPatterns {
		if containsKeyword(contentLower, pattern) {
			riskScore += 1.0
		}
	}

	// Medium-risk patterns
	mediumRiskPatterns := []string{
		"api", "service", "microservice", "event", "queue", "cache",
		"monitoring", "logging", "testing", "deployment", "docker",
	}

	for _, pattern := range mediumRiskPatterns {
		if containsKeyword(contentLower, pattern) {
			riskScore += 0.5
		}
	}

	// Normalize to 0-10 scale
	if riskScore > 10.0 {
		riskScore = 10.0
	}

	return riskScore
}

// ExtractRequiredSkills extracts the required skills from content
func (a *DefaultComplexityAnalyzer) ExtractRequiredSkills(content string) []string {
	contentLower := toLowerCase(content)
	var skills []string
	skillMap := make(map[string]bool)

	// Technical skills
	technicalSkills := map[string]string{
		"go":         "Go Programming",
		"golang":     "Go Programming",
		"javascript": "JavaScript",
		"typescript": "TypeScript",
		"react":      "React",
		"vue":        "Vue.js",
		"angular":    "Angular",
		"python":     "Python",
		"java":       "Java",
		"sql":        "SQL",
		"nosql":      "NoSQL",
		"docker":     "Docker",
		"kubernetes": "Kubernetes",
		"aws":        "AWS",
		"azure":      "Azure",
		"gcp":        "Google Cloud",
	}

	// Domain skills
	domainSkills := map[string]string{
		"security":     "Security Engineering",
		"performance":  "Performance Optimization",
		"testing":      "Software Testing",
		"devops":       "DevOps",
		"frontend":     "Frontend Development",
		"backend":      "Backend Development",
		"fullstack":    "Full-Stack Development",
		"architecture": "Software Architecture",
		"database":     "Database Design",
		"api":          "API Development",
		"microservice": "Microservices",
		"mobile":       "Mobile Development",
	}

	// Check for technical skills
	for keyword, skill := range technicalSkills {
		if containsKeyword(contentLower, keyword) && !skillMap[skill] {
			skills = append(skills, skill)
			skillMap[skill] = true
		}
	}

	// Check for domain skills
	for keyword, skill := range domainSkills {
		if containsKeyword(contentLower, keyword) && !skillMap[skill] {
			skills = append(skills, skill)
			skillMap[skill] = true
		}
	}

	// Add general programming skill if no specific language detected
	if len(skills) == 0 || (!skillMap["Go Programming"] && !skillMap["JavaScript"] && !skillMap["Python"]) {
		skills = append(skills, "Software Development")
	}

	return skills
}

// Private methods for complexity analysis

func (a *DefaultComplexityAnalyzer) analyzeTechnicalComplexity(content string, words []string) float64 {
	score := 0.0

	// Technical complexity patterns
	technicalPatterns := []string{
		"algorithm", "optimization", "performance", "scalability", "concurrency",
		"parallel", "distributed", "architecture", "design pattern", "framework",
		"library", "sdk", "api", "protocol", "encryption", "security",
	}

	for _, pattern := range technicalPatterns {
		if containsKeyword(content, pattern) {
			score += 1.0
		}
	}

	// Programming language complexity
	complexLanguages := []string{"assembly", "c++", "rust", "haskell", "scala"}
	moderateLanguages := []string{"java", "c#", "go", "typescript"}

	for _, lang := range complexLanguages {
		if containsKeyword(content, lang) {
			score += 2.0
		}
	}

	for _, lang := range moderateLanguages {
		if containsKeyword(content, lang) {
			score += 1.0
		}
	}

	// Content length factor
	if len(words) > 50 {
		score += 1.0
	}
	if len(words) > 100 {
		score += 1.0
	}

	// Cap at 10.0
	if score > 10.0 {
		score = 10.0
	}

	return score
}

func (a *DefaultComplexityAnalyzer) analyzeBusinessComplexity(content string, _ []string) float64 {
	score := 0.0

	// Business complexity patterns
	businessPatterns := []string{
		"requirements", "stakeholder", "compliance", "regulation", "audit",
		"workflow", "process", "business logic", "rules", "policy",
		"integration", "migration", "legacy", "transformation",
	}

	for _, pattern := range businessPatterns {
		if containsKeyword(content, pattern) {
			score += 1.0
		}
	}

	// Domain-specific complexity
	complexDomains := []string{
		"financial", "healthcare", "banking", "insurance", "government",
		"legal", "pharmaceutical", "aviation", "automotive",
	}

	for _, domain := range complexDomains {
		if containsKeyword(content, domain) {
			score += 2.0
		}
	}

	// Cap at 10.0
	if score > 10.0 {
		score = 10.0
	}

	return score
}

func (a *DefaultComplexityAnalyzer) analyzeIntegrationComplexity(content string, _ []string) float64 {
	score := 0.0

	// Integration patterns
	integrationPatterns := []string{
		"integration", "api", "service", "microservice", "webhook",
		"third-party", "external", "sync", "async", "queue",
		"event", "message", "broker", "gateway", "proxy",
	}

	for _, pattern := range integrationPatterns {
		if containsKeyword(content, pattern) {
			score += 1.0
		}
	}

	// Multiple integration complexity
	integrationCount := 0
	countPatterns := []string{"api", "service", "system", "platform"}
	for _, pattern := range countPatterns {
		if containsKeyword(content, pattern) {
			integrationCount++
		}
	}

	if integrationCount > 2 {
		score += 2.0
	} else if integrationCount > 1 {
		score += 1.0
	}

	// Cap at 10.0
	if score > 10.0 {
		score = 10.0
	}

	return score
}

func (a *DefaultComplexityAnalyzer) analyzeTestingComplexity(content string, _ []string) float64 {
	score := 2.0 // Base testing complexity

	// Testing patterns
	testingPatterns := []string{
		"test", "testing", "unit test", "integration test", "e2e",
		"mock", "stub", "fixture", "coverage", "automation",
	}

	for _, pattern := range testingPatterns {
		if containsKeyword(content, pattern) {
			score += 0.5
		}
	}

	// Complex testing scenarios
	complexTestingPatterns := []string{
		"performance test", "load test", "stress test", "security test",
		"penetration test", "chaos test", "property test",
	}

	for _, pattern := range complexTestingPatterns {
		if containsKeyword(content, pattern) {
			score += 1.0
		}
	}

	// Cap at 10.0
	if score > 10.0 {
		score = 10.0
	}

	return score
}

func (a *DefaultComplexityAnalyzer) analyzeDocumentationComplexity(content string, words []string) float64 {
	score := 1.0 // Base documentation complexity

	// Documentation patterns
	docPatterns := []string{
		"documentation", "docs", "manual", "guide", "tutorial",
		"readme", "api docs", "specification", "reference",
	}

	for _, pattern := range docPatterns {
		if containsKeyword(content, pattern) {
			score += 0.5
		}
	}

	// Complex documentation
	complexDocPatterns := []string{
		"architecture", "design", "specification", "protocol",
		"standard", "compliance", "certification",
	}

	for _, pattern := range complexDocPatterns {
		if containsKeyword(content, pattern) {
			score += 1.0
		}
	}

	// Content length affects documentation complexity
	if len(words) > 100 {
		score += 1.0
	}

	// Cap at 10.0
	if score > 10.0 {
		score = 10.0
	}

	return score
}

func (a *DefaultComplexityAnalyzer) analyzeDataComplexity(content string, _ []string) float64 {
	score := 0.0

	// Data complexity patterns
	dataPatterns := []string{
		"database", "data", "model", "schema", "migration",
		"sql", "nosql", "query", "index", "relation",
		"etl", "pipeline", "warehouse", "analytics",
	}

	for _, pattern := range dataPatterns {
		if containsKeyword(content, pattern) {
			score += 1.0
		}
	}

	// Complex data operations
	complexDataPatterns := []string{
		"big data", "real-time", "streaming", "batch", "distributed",
		"sharding", "replication", "consistency", "transaction",
	}

	for _, pattern := range complexDataPatterns {
		if containsKeyword(content, pattern) {
			score += 2.0
		}
	}

	// Cap at 10.0
	if score > 10.0 {
		score = 10.0
	}

	return score
}

func (a *DefaultComplexityAnalyzer) analyzeUIComplexity(content string, _ []string) float64 {
	score := 0.0

	// UI complexity patterns
	uiPatterns := []string{
		"ui", "interface", "frontend", "component", "react",
		"vue", "angular", "html", "css", "responsive",
		"mobile", "desktop", "web", "app",
	}

	for _, pattern := range uiPatterns {
		if containsKeyword(content, pattern) {
			score += 1.0
		}
	}

	// Complex UI features
	complexUIPatterns := []string{
		"animation", "transition", "visualization", "chart",
		"graph", "3d", "canvas", "webgl", "real-time",
		"interactive", "drag", "drop", "gesture",
	}

	for _, pattern := range complexUIPatterns {
		if containsKeyword(content, pattern) {
			score += 2.0
		}
	}

	// Cap at 10.0
	if score > 10.0 {
		score = 10.0
	}

	return score
}

func (a *DefaultComplexityAnalyzer) calculateOverallScore(breakdown ComplexityBreakdown) float64 {
	return (breakdown.Technical*a.weights.Technical +
		breakdown.Business*a.weights.Business +
		breakdown.Integration*a.weights.Integration +
		breakdown.Testing*a.weights.Testing +
		breakdown.Documentation*a.weights.Documentation +
		breakdown.DataComplexity*a.weights.Data +
		breakdown.UIComplexity*a.weights.UI) /
		(a.weights.Technical + a.weights.Business + a.weights.Integration +
			a.weights.Testing + a.weights.Documentation + a.weights.Data + a.weights.UI)
}

func (a *DefaultComplexityAnalyzer) extractComplexityFactors(breakdown ComplexityBreakdown, _ string) []ComplexityFactor {
	var factors []ComplexityFactor

	if breakdown.Technical >= 7.0 {
		factors = append(factors, ComplexityFactor{
			Name:        "High Technical Complexity",
			Impact:      "high",
			Score:       breakdown.Technical,
			Description: "Complex technical implementation required",
			Weight:      a.weights.Technical,
		})
	}

	if breakdown.Integration >= 6.0 {
		factors = append(factors, ComplexityFactor{
			Name:        "Integration Complexity",
			Impact:      "medium",
			Score:       breakdown.Integration,
			Description: "Multiple system integrations required",
			Weight:      a.weights.Integration,
		})
	}

	if breakdown.Business >= 7.0 {
		factors = append(factors, ComplexityFactor{
			Name:        "Business Logic Complexity",
			Impact:      "high",
			Score:       breakdown.Business,
			Description: "Complex business rules and workflows",
			Weight:      a.weights.Business,
		})
	}

	if breakdown.DataComplexity >= 6.0 {
		factors = append(factors, ComplexityFactor{
			Name:        "Data Complexity",
			Impact:      "medium",
			Score:       breakdown.DataComplexity,
			Description: "Complex data operations and modeling",
			Weight:      a.weights.Data,
		})
	}

	return factors
}

func (a *DefaultComplexityAnalyzer) scoreToLevel(score float64) string {
	switch {
	case score >= 7.0:
		return constants.SeverityHigh
	case score >= 4.0:
		return constants.SeverityMedium
	default:
		return constants.SeverityLow
	}
}

func (a *DefaultComplexityAnalyzer) scoreToHours(score float64, wordCount int) int {
	// Base hours calculation
	baseHours := 2

	// Complexity multiplier (exponential growth for high complexity)
	complexityMultiplier := 1.0
	if score >= 8.0 {
		complexityMultiplier = 4.0
	} else if score >= 6.0 {
		complexityMultiplier = 2.5
	} else if score >= 4.0 {
		complexityMultiplier = 1.5
	}

	// Content length factor
	lengthMultiplier := 1.0
	if wordCount > 100 {
		lengthMultiplier = 1.5
	} else if wordCount > 50 {
		lengthMultiplier = 1.2
	}

	hours := float64(baseHours) * complexityMultiplier * lengthMultiplier

	// Round to reasonable values
	finalHours := int(hours + 0.5)

	// Ensure bounds
	if finalHours < 1 {
		finalHours = 1
	} else if finalHours > 40 {
		finalHours = 40
	}

	return finalHours
}

func (a *DefaultComplexityAnalyzer) calculateConfidence(factors []ComplexityFactor, wordCount int) float64 {
	confidence := 0.7 // Base confidence

	// More factors = higher confidence in analysis
	if len(factors) >= 3 {
		confidence += 0.2
	} else if len(factors) >= 1 {
		confidence += 0.1
	}

	// Longer content = higher confidence
	if wordCount > 50 {
		confidence += 0.1
	}
	if wordCount > 100 {
		confidence += 0.1
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

func (a *DefaultComplexityAnalyzer) generateRecommendations(breakdown ComplexityBreakdown, _ []ComplexityFactor, level string) []string {
	var recommendations []string

	switch level {
	case constants.SeverityHigh:
		recommendations = append(recommendations, "Consider breaking this into smaller, more manageable tasks")
		recommendations = append(recommendations, "Plan for additional testing and validation time")
		recommendations = append(recommendations, "Consider involving senior developers or architects")

	case constants.SeverityMedium:
		recommendations = append(recommendations, "Ensure proper planning and design before implementation")
		recommendations = append(recommendations, "Include time for code review and testing")

	case constants.SeverityLow:
		recommendations = append(recommendations, "Good candidate for junior developers or quick implementation")
		recommendations = append(recommendations, "Consider using existing templates or patterns")
	}

	// Specific recommendations based on complexity areas
	if breakdown.Technical >= 7.0 {
		recommendations = append(recommendations, "Research technical approaches and create proof of concept")
	}

	if breakdown.Integration >= 6.0 {
		recommendations = append(recommendations, "Plan integration testing and error handling carefully")
	}

	if breakdown.Testing >= 7.0 {
		recommendations = append(recommendations, "Allocate significant time for comprehensive testing")
	}

	return recommendations
}

// loadComplexityPatterns loads default complexity patterns
func loadComplexityPatterns() map[string]ComplexityPattern {
	patterns := make(map[string]ComplexityPattern)

	// Technical patterns
	patterns["algorithm"] = ComplexityPattern{
		Keywords:    []string{"algorithm", "sorting", "searching", "graph", "tree"},
		Impact:      2.0,
		Category:    "technical",
		Description: "Algorithmic complexity",
		Weight:      0.3,
	}

	patterns["security"] = ComplexityPattern{
		Keywords:    []string{"security", "authentication", "authorization", "encryption"},
		Impact:      2.5,
		Category:    "security",
		Description: "Security implementation complexity",
		Weight:      0.4,
	}

	patterns["performance"] = ComplexityPattern{
		Keywords:    []string{"performance", "optimization", "scalability", "cache"},
		Impact:      2.0,
		Category:    "performance",
		Description: "Performance optimization complexity",
		Weight:      0.3,
	}

	return patterns
}

// getDefaultWeights returns default complexity weights
func getDefaultWeights() ComplexityWeights {
	return ComplexityWeights{
		Technical:     0.25,
		Business:      0.20,
		Integration:   0.20,
		Testing:       0.15,
		Documentation: 0.05,
		Data:          0.10,
		UI:            0.05,
	}
}
