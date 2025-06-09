// Package tasks provides task template matching and application functionality.
package tasks

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// TemplateMatcher matches tasks to templates and applies them
type TemplateMatcher struct {
	templates []types.TaskTemplate
	config    TemplateConfig
}

// TemplateConfig represents configuration for template matching
type TemplateConfig struct {
	MinMatchScore       float64 `json:"min_match_score"`
	EnableFuzzyMatching bool    `json:"enable_fuzzy_matching"`
	MaxTemplates        int     `json:"max_templates"`
	BoostPopularTemplates bool  `json:"boost_popular_templates"`
}

// DefaultTemplateConfig returns default template configuration
func DefaultTemplateConfig() TemplateConfig {
	return TemplateConfig{
		MinMatchScore:         0.6,
		EnableFuzzyMatching:   true,
		MaxTemplates:         100,
		BoostPopularTemplates: true,
	}
}

// NewTemplateMatcher creates a new template matcher
func NewTemplateMatcher() *TemplateMatcher {
	return &TemplateMatcher{
		templates: getBuiltinTemplates(),
		config:    DefaultTemplateConfig(),
	}
}

// NewTemplateMatcherWithConfig creates a new template matcher with custom config
func NewTemplateMatcherWithConfig(templates []types.TaskTemplate, config TemplateConfig) *TemplateMatcher {
	allTemplates := append(getBuiltinTemplates(), templates...)
	return &TemplateMatcher{
		templates: allTemplates,
		config:    config,
	}
}

// FindBestMatch finds the best matching template for a task
func (tm *TemplateMatcher) FindBestMatch(task *types.Task, context types.TaskGenerationContext) *types.TaskTemplate {
	if len(tm.templates) == 0 {
		return nil
	}

	bestScore := 0.0
	var bestTemplate *types.TaskTemplate

	for i := range tm.templates {
		template := &tm.templates[i]
		score := tm.calculateMatchScore(task, template, context)

		if score > bestScore && score >= tm.config.MinMatchScore {
			bestScore = score
			bestTemplate = template
		}
	}

	return bestTemplate
}

// FindAllMatches finds all matching templates for a task
func (tm *TemplateMatcher) FindAllMatches(task *types.Task, context types.TaskGenerationContext) []TemplateMatch {
	matches := []TemplateMatch{}

	for i := range tm.templates {
		template := &tm.templates[i]
		score := tm.calculateMatchScore(task, template, context)

		if score >= tm.config.MinMatchScore {
			matches = append(matches, TemplateMatch{
				Template: template,
				Score:    score,
				Reasons:  tm.getMatchReasons(task, template, context),
			})
		}
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	return matches
}

// TemplateMatch represents a template match with score and reasons
type TemplateMatch struct {
	Template *types.TaskTemplate `json:"template"`
	Score    float64            `json:"score"`
	Reasons  []string           `json:"reasons"`
}

// ApplyTemplate applies a template to a task
func (tm *TemplateMatcher) ApplyTemplate(task *types.Task, template *types.TaskTemplate) types.Task {
	if template == nil {
		return *task
	}

	// Create a copy of the task
	enhancedTask := *task

	// Apply template metadata
	enhancedTask.Metadata.TemplateID = template.ID

	// Enhance task type if not set or if template is more specific
	if enhancedTask.Type == "" || tm.isMoreSpecificType(template.Type, enhancedTask.Type) {
		enhancedTask.Type = template.Type
	}

	// Enhance priority if template suggests higher priority
	if tm.isHigherPriority(template.DefaultPriority, enhancedTask.Priority) {
		enhancedTask.Priority = template.DefaultPriority
	}

	// Enhance complexity if not analyzed yet
	if enhancedTask.Complexity.Level == "" {
		enhancedTask.Complexity.Level = template.DefaultComplexity
	}

	// Apply effort estimate if not set
	if enhancedTask.EstimatedEffort.Hours == 0 {
		enhancedTask.EstimatedEffort = template.EstimatedEffort
	}

	// Merge acceptance criteria
	enhancedTask.AcceptanceCriteria = tm.mergeAcceptanceCriteria(
		enhancedTask.AcceptanceCriteria,
		template.AcceptanceCriteria,
	)

	// Merge required skills
	enhancedTask.Complexity.RequiredSkills = tm.mergeSkills(
		enhancedTask.Complexity.RequiredSkills,
		template.RequiredSkills,
	)

	// Merge tags
	enhancedTask.Tags = tm.mergeTags(enhancedTask.Tags, template.Tags)

	// Update template usage statistics
	tm.updateTemplateUsage(template.ID)

	return enhancedTask
}

// calculateMatchScore calculates how well a template matches a task
func (tm *TemplateMatcher) calculateMatchScore(task *types.Task, template *types.TaskTemplate, context types.TaskGenerationContext) float64 {
	score := 0.0

	// Task type match (high weight)
	if task.Type == template.Type {
		score += 0.3
	} else if tm.areRelatedTypes(task.Type, template.Type) {
		score += 0.15
	}

	// Keyword matching in title and description
	keywordScore := tm.calculateKeywordMatch(task, template)
	score += keywordScore * 0.25

	// Context applicability
	contextScore := tm.calculateContextMatch(template, context)
	score += contextScore * 0.2

	// Project type match
	if tm.matchesProjectType(template, context.ProjectType) {
		score += 0.1
	}

	// Tech stack match
	techScore := tm.calculateTechStackMatch(template, context.TechStack)
	score += techScore * 0.1

	// Popular template boost
	if tm.config.BoostPopularTemplates && template.SuccessRate > 0.8 {
		score += 0.05
	}

	// Fuzzy matching for similar concepts
	if tm.config.EnableFuzzyMatching {
		fuzzyScore := tm.calculateFuzzyMatch(task, template)
		score += fuzzyScore * 0.1
	}

	return score
}

// calculateKeywordMatch calculates keyword matching score
func (tm *TemplateMatcher) calculateKeywordMatch(task *types.Task, template *types.TaskTemplate) float64 {
	taskContent := strings.ToLower(task.Title + " " + task.Description)
	
	matches := 0
	total := len(template.Applicability.Keywords)
	
	if total == 0 {
		return 0.0
	}

	for _, keyword := range template.Applicability.Keywords {
		if strings.Contains(taskContent, strings.ToLower(keyword)) {
			matches++
		}
	}

	return float64(matches) / float64(total)
}

// calculateContextMatch calculates context matching score
func (tm *TemplateMatcher) calculateContextMatch(template *types.TaskTemplate, context types.TaskGenerationContext) float64 {
	score := 0.0
	factors := 0

	// Project type match
	if tm.matchesProjectType(template, context.ProjectType) {
		score += 1.0
	}
	factors++

	// Tech stack match
	techScore := tm.calculateTechStackMatch(template, context.TechStack)
	score += techScore
	factors++

	// Team size match
	if len(template.Applicability.TeamSizes) > 0 {
		teamMatch := false
		for _, size := range template.Applicability.TeamSizes {
			if size == context.TeamSize {
				teamMatch = true
				break
			}
		}
		if teamMatch {
			score += 1.0
		}
		factors++
	}

	if factors == 0 {
		return 0.0
	}

	return score / float64(factors)
}

// calculateTechStackMatch calculates tech stack matching score
func (tm *TemplateMatcher) calculateTechStackMatch(template *types.TaskTemplate, techStack []string) float64 {
	if len(template.Applicability.TechStacks) == 0 || len(techStack) == 0 {
		return 0.0
	}

	matches := 0
	for _, templateTech := range template.Applicability.TechStacks {
		for _, contextTech := range techStack {
			if strings.Contains(strings.ToLower(contextTech), strings.ToLower(templateTech)) {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(template.Applicability.TechStacks))
}

// calculateFuzzyMatch calculates fuzzy matching score using semantic similarity
func (tm *TemplateMatcher) calculateFuzzyMatch(task *types.Task, template *types.TaskTemplate) float64 {
	// Simple fuzzy matching based on common concepts
	conceptMap := map[string][]string{
		"api":          {"endpoint", "service", "rest", "graphql", "http"},
		"database":     {"db", "sql", "nosql", "storage", "persistence"},
		"frontend":     {"ui", "ux", "client", "web", "interface"},
		"backend":      {"server", "service", "api", "business logic"},
		"testing":      {"qa", "validation", "verification", "quality"},
		"deployment":   {"release", "production", "deploy", "publish"},
		"security":     {"auth", "authentication", "authorization", "encryption"},
		"monitoring":   {"logging", "metrics", "observability", "tracking"},
		"performance":  {"optimization", "speed", "efficiency", "scalability"},
		"integration":  {"connector", "bridge", "sync", "interface"},
	}

	taskContent := strings.ToLower(task.Title + " " + task.Description)
	templateContent := strings.ToLower(template.Name + " " + template.Description)

	commonConcepts := 0
	totalConcepts := 0

	for concept, keywords := range conceptMap {
		taskHasConcept := strings.Contains(taskContent, concept)
		templateHasConcept := strings.Contains(templateContent, concept)

		if !taskHasConcept {
			for _, keyword := range keywords {
				if strings.Contains(taskContent, keyword) {
					taskHasConcept = true
					break
				}
			}
		}

		if !templateHasConcept {
			for _, keyword := range keywords {
				if strings.Contains(templateContent, keyword) {
					templateHasConcept = true
					break
				}
			}
		}

		if taskHasConcept || templateHasConcept {
			totalConcepts++
			if taskHasConcept && templateHasConcept {
				commonConcepts++
			}
		}
	}

	if totalConcepts == 0 {
		return 0.0
	}

	return float64(commonConcepts) / float64(totalConcepts)
}

// getMatchReasons returns reasons why a template matches a task
func (tm *TemplateMatcher) getMatchReasons(task *types.Task, template *types.TaskTemplate, context types.TaskGenerationContext) []string {
	reasons := []string{}

	// Task type match
	if task.Type == template.Type {
		reasons = append(reasons, fmt.Sprintf("Task type matches: %s", template.Type))
	}

	// Keyword matches
	taskContent := strings.ToLower(task.Title + " " + task.Description)
	matchedKeywords := []string{}
	for _, keyword := range template.Applicability.Keywords {
		if strings.Contains(taskContent, strings.ToLower(keyword)) {
			matchedKeywords = append(matchedKeywords, keyword)
		}
	}
	if len(matchedKeywords) > 0 {
		reasons = append(reasons, fmt.Sprintf("Keywords match: %s", strings.Join(matchedKeywords, ", ")))
	}

	// Project type match
	if tm.matchesProjectType(template, context.ProjectType) {
		reasons = append(reasons, fmt.Sprintf("Project type matches: %s", context.ProjectType))
	}

	// Tech stack matches
	matchedTech := []string{}
	for _, templateTech := range template.Applicability.TechStacks {
		for _, contextTech := range context.TechStack {
			if strings.Contains(strings.ToLower(contextTech), strings.ToLower(templateTech)) {
				matchedTech = append(matchedTech, templateTech)
				break
			}
		}
	}
	if len(matchedTech) > 0 {
		reasons = append(reasons, fmt.Sprintf("Tech stack matches: %s", strings.Join(matchedTech, ", ")))
	}

	// High success rate
	if template.SuccessRate > 0.8 {
		reasons = append(reasons, fmt.Sprintf("High success rate: %.1f%%", template.SuccessRate*100))
	}

	return reasons
}

// Helper functions

// areRelatedTypes checks if two task types are related
func (tm *TemplateMatcher) areRelatedTypes(type1, type2 types.TaskType) bool {
	relatedTypes := map[types.TaskType][]types.TaskType{
		types.TaskTypeImplementation: {types.TaskTypeDesign, types.TaskTypeArchitecture},
		types.TaskTypeDesign:         {types.TaskTypeImplementation, types.TaskTypeArchitecture},
		types.TaskTypeTesting:        {types.TaskTypeImplementation, types.TaskTypeReview},
		types.TaskTypeDocumentation:  {types.TaskTypeImplementation, types.TaskTypeReview},
		types.TaskTypeDeployment:     {types.TaskTypeImplementation, types.TaskTypeTesting},
		types.TaskTypeRefactoring:    {types.TaskTypeImplementation, types.TaskTypeArchitecture},
		types.TaskTypeIntegration:    {types.TaskTypeImplementation, types.TaskTypeArchitecture},
	}

	if related, exists := relatedTypes[type1]; exists {
		for _, relatedType := range related {
			if relatedType == type2 {
				return true
			}
		}
	}

	return false
}

// isMoreSpecificType checks if one type is more specific than another
func (tm *TemplateMatcher) isMoreSpecificType(template, current types.TaskType) bool {
	specificityOrder := map[types.TaskType]int{
		types.TaskTypeImplementation: 1,
		types.TaskTypeDesign:         2,
		types.TaskTypeTesting:        3,
		types.TaskTypeDocumentation:  4,
		types.TaskTypeResearch:       5,
		types.TaskTypeReview:         6,
		types.TaskTypeDeployment:     7,
		types.TaskTypeArchitecture:   8,
		types.TaskTypeBugFix:         9,
		types.TaskTypeRefactoring:    10,
		types.TaskTypeIntegration:    11,
		types.TaskTypeAnalysis:       12,
	}

	templateSpec, templateExists := specificityOrder[template]
	currentSpec, currentExists := specificityOrder[current]

	if !templateExists || !currentExists {
		return false
	}

	return templateSpec > currentSpec
}

// isHigherPriority checks if one priority is higher than another
func (tm *TemplateMatcher) isHigherPriority(template, current types.TaskPriority) bool {
	priorityOrder := map[types.TaskPriority]int{
		types.TaskPriorityLow:      1,
		types.TaskPriorityMedium:   2,
		types.TaskPriorityHigh:     3,
		types.TaskPriorityCritical: 4,
		types.TaskPriorityBlocking: 5,
	}

	templatePrio, templateExists := priorityOrder[template]
	currentPrio, currentExists := priorityOrder[current]

	if !templateExists || !currentExists {
		return false
	}

	return templatePrio > currentPrio
}

// matchesProjectType checks if template applies to project type
func (tm *TemplateMatcher) matchesProjectType(template *types.TaskTemplate, projectType types.ProjectType) bool {
	if len(template.Applicability.ProjectTypes) == 0 {
		return true // Template applies to all project types
	}

	for _, templateType := range template.Applicability.ProjectTypes {
		if templateType == projectType {
			return true
		}
	}

	return false
}

// mergeAcceptanceCriteria merges acceptance criteria from task and template
func (tm *TemplateMatcher) mergeAcceptanceCriteria(taskCriteria, templateCriteria []string) []string {
	criteriaMap := make(map[string]bool)
	result := []string{}

	// Add existing task criteria
	for _, criteria := range taskCriteria {
		if !criteriaMap[criteria] {
			result = append(result, criteria)
			criteriaMap[criteria] = true
		}
	}

	// Add template criteria that don't duplicate
	for _, criteria := range templateCriteria {
		if !criteriaMap[criteria] {
			result = append(result, criteria)
			criteriaMap[criteria] = true
		}
	}

	return result
}

// mergeSkills merges required skills from task and template
func (tm *TemplateMatcher) mergeSkills(taskSkills, templateSkills []string) []string {
	skillMap := make(map[string]bool)
	result := []string{}

	// Add existing task skills
	for _, skill := range taskSkills {
		if !skillMap[skill] {
			result = append(result, skill)
			skillMap[skill] = true
		}
	}

	// Add template skills that don't duplicate
	for _, skill := range templateSkills {
		if !skillMap[skill] {
			result = append(result, skill)
			skillMap[skill] = true
		}
	}

	return result
}

// mergeTags merges tags from task and template
func (tm *TemplateMatcher) mergeTags(taskTags, templateTags []string) []string {
	tagMap := make(map[string]bool)
	result := []string{}

	// Add existing task tags
	for _, tag := range taskTags {
		if !tagMap[tag] {
			result = append(result, tag)
			tagMap[tag] = true
		}
	}

	// Add template tags that don't duplicate
	for _, tag := range templateTags {
		if !tagMap[tag] {
			result = append(result, tag)
			tagMap[tag] = true
		}
	}

	return result
}

// updateTemplateUsage updates template usage statistics
func (tm *TemplateMatcher) updateTemplateUsage(templateID string) {
	for i := range tm.templates {
		if tm.templates[i].ID == templateID {
			tm.templates[i].UsageCount++
			tm.templates[i].UpdatedAt = time.Now()
			break
		}
	}
}

// getBuiltinTemplates returns a set of built-in task templates
func getBuiltinTemplates() []types.TaskTemplate {
	now := time.Now()
	
	return []types.TaskTemplate{
		{
			ID:              "api_endpoint_template",
			Name:            "API Endpoint Implementation",
			Description:     "Template for implementing REST API endpoints",
			Category:        "Implementation",
			Type:            types.TaskTypeImplementation,
			DefaultPriority: types.TaskPriorityMedium,
			DefaultComplexity: types.ComplexityModerate,
			EstimatedEffort: types.EffortEstimate{
				Hours:            8.0,
				Days:             1.0,
				Confidence:       0.8,
				EstimationMethod: "template_based",
				Breakdown: types.EffortBreakdown{
					Analysis:       1.0,
					Design:         1.0,
					Implementation: 4.0,
					Testing:        1.5,
					Documentation:  0.5,
				},
			},
			AcceptanceCriteria: []string{
				"API endpoint responds with correct HTTP status codes",
				"Request validation is implemented and tested",
				"Response format matches API specification",
				"Error handling covers all edge cases",
				"API documentation is updated",
			},
			RequiredSkills: []string{"Backend Development", "API Development", "Testing"},
			Tags:           []string{"api", "backend", "endpoint", "rest"},
			Applicability: types.TemplateApplicability{
				ProjectTypes: []types.ProjectType{types.ProjectTypeFeature, types.ProjectTypeProduct},
				Keywords:     []string{"api", "endpoint", "rest", "service", "backend"},
				TechStacks:   []string{"nodejs", "python", "go", "java", "express", "fastapi"},
			},
			CreatedAt:   now,
			UpdatedAt:   now,
			UsageCount:  0,
			SuccessRate: 0.85,
		},
		{
			ID:              "frontend_component_template",
			Name:            "Frontend Component Implementation",
			Description:     "Template for implementing reusable frontend components",
			Category:        "Implementation",
			Type:            types.TaskTypeImplementation,
			DefaultPriority: types.TaskPriorityMedium,
			DefaultComplexity: types.ComplexityModerate,
			EstimatedEffort: types.EffortEstimate{
				Hours:            6.0,
				Days:             0.75,
				Confidence:       0.8,
				EstimationMethod: "template_based",
				Breakdown: types.EffortBreakdown{
					Design:         1.0,
					Implementation: 3.0,
					Testing:        1.5,
					Documentation:  0.5,
				},
			},
			AcceptanceCriteria: []string{
				"Component renders correctly in all target browsers",
				"Component is responsive and accessible",
				"Props and state management is properly implemented",
				"Unit tests cover component functionality",
				"Storybook documentation is created",
			},
			RequiredSkills: []string{"Frontend Development", "UI/UX Design", "Testing"},
			Tags:           []string{"frontend", "component", "ui", "react", "vue"},
			Applicability: types.TemplateApplicability{
				ProjectTypes: []types.ProjectType{types.ProjectTypeFeature, types.ProjectTypeProduct},
				Keywords:     []string{"component", "ui", "frontend", "react", "vue", "angular"},
				TechStacks:   []string{"react", "vue", "angular", "typescript", "javascript"},
			},
			CreatedAt:   now,
			UpdatedAt:   now,
			UsageCount:  0,
			SuccessRate: 0.82,
		},
		{
			ID:              "database_migration_template",
			Name:            "Database Schema Migration",
			Description:     "Template for database schema changes and migrations",
			Category:        "Implementation",
			Type:            types.TaskTypeImplementation,
			DefaultPriority: types.TaskPriorityHigh,
			DefaultComplexity: types.ComplexityComplex,
			EstimatedEffort: types.EffortEstimate{
				Hours:            12.0,
				Days:             1.5,
				Confidence:       0.7,
				EstimationMethod: "template_based",
				Breakdown: types.EffortBreakdown{
					Analysis:       2.0,
					Design:         2.0,
					Implementation: 4.0,
					Testing:        3.0,
					Documentation:  1.0,
				},
			},
			AcceptanceCriteria: []string{
				"Migration script is tested on development environment",
				"Rollback procedure is documented and tested",
				"Data integrity is maintained during migration",
				"Performance impact is analyzed and acceptable",
				"Migration is successfully applied to staging environment",
			},
			RequiredSkills: []string{"Database", "Backend Development", "DevOps"},
			Tags:           []string{"database", "migration", "schema", "sql"},
			Applicability: types.TemplateApplicability{
				ProjectTypes: []types.ProjectType{types.ProjectTypeFeature, types.ProjectTypeRefactor},
				Keywords:     []string{"database", "migration", "schema", "sql", "table"},
				TechStacks:   []string{"postgresql", "mysql", "mongodb", "sql"},
			},
			CreatedAt:   now,
			UpdatedAt:   now,
			UsageCount:  0,
			SuccessRate: 0.78,
		},
		{
			ID:              "testing_suite_template",
			Name:            "Testing Suite Implementation",
			Description:     "Template for comprehensive testing implementation",
			Category:        "Testing",
			Type:            types.TaskTypeTesting,
			DefaultPriority: types.TaskPriorityMedium,
			DefaultComplexity: types.ComplexityModerate,
			EstimatedEffort: types.EffortEstimate{
				Hours:            10.0,
				Days:             1.25,
				Confidence:       0.8,
				EstimationMethod: "template_based",
				Breakdown: types.EffortBreakdown{
					Analysis:       1.0,
					Design:         2.0,
					Implementation: 6.0,
					Testing:        0.5,
					Documentation:  0.5,
				},
			},
			AcceptanceCriteria: []string{
				"Unit tests achieve minimum 80% code coverage",
				"Integration tests cover main user workflows",
				"Test suite runs in CI/CD pipeline",
				"Test results are properly reported",
				"Performance benchmarks are established",
			},
			RequiredSkills: []string{"Testing", "QA", "Test Automation"},
			Tags:           []string{"testing", "qa", "automation", "coverage"},
			Applicability: types.TemplateApplicability{
				ProjectTypes: []types.ProjectType{types.ProjectTypeFeature, types.ProjectTypeProduct},
				Keywords:     []string{"test", "testing", "qa", "automation", "coverage"},
				TechStacks:   []string{"jest", "cypress", "selenium", "pytest", "mocha"},
			},
			CreatedAt:   now,
			UpdatedAt:   now,
			UsageCount:  0,
			SuccessRate: 0.88,
		},
		{
			ID:              "documentation_template",
			Name:            "Technical Documentation",
			Description:     "Template for creating comprehensive technical documentation",
			Category:        "Documentation",
			Type:            types.TaskTypeDocumentation,
			DefaultPriority: types.TaskPriorityLow,
			DefaultComplexity: types.ComplexitySimple,
			EstimatedEffort: types.EffortEstimate{
				Hours:            4.0,
				Days:             0.5,
				Confidence:       0.9,
				EstimationMethod: "template_based",
				Breakdown: types.EffortBreakdown{
					Analysis:      0.5,
					Documentation: 3.0,
					Review:        0.5,
				},
			},
			AcceptanceCriteria: []string{
				"Documentation covers all public APIs",
				"Code examples are provided and tested",
				"Installation and setup instructions are clear",
				"Documentation is published and accessible",
				"Feedback from team members is incorporated",
			},
			RequiredSkills: []string{"Technical Writing", "Documentation"},
			Tags:           []string{"documentation", "readme", "api-docs", "guide"},
			Applicability: types.TemplateApplicability{
				ProjectTypes: []types.ProjectType{types.ProjectTypeFeature, types.ProjectTypeProduct},
				Keywords:     []string{"documentation", "readme", "guide", "manual", "docs"},
			},
			CreatedAt:   now,
			UpdatedAt:   now,
			UsageCount:  0,
			SuccessRate: 0.92,
		},
	}
}