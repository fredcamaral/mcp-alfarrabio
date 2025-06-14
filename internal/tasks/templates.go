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
	MinMatchScore         float64 `json:"min_match_score"`
	EnableFuzzyMatching   bool    `json:"enable_fuzzy_matching"`
	MaxTemplates          int     `json:"max_templates"`
	BoostPopularTemplates bool    `json:"boost_popular_templates"`
}

// DefaultTemplateConfig returns default template configuration
func DefaultTemplateConfig() TemplateConfig {
	return TemplateConfig{
		MinMatchScore:         0.6,
		EnableFuzzyMatching:   true,
		MaxTemplates:          100,
		BoostPopularTemplates: true,
	}
}

// NewTemplateMatcher creates a new template matcher
func NewTemplateMatcher() *TemplateMatcher {
	return &TemplateMatcher{
		templates: []types.TaskTemplate{}, // TODO: implement getBuiltinTemplates with new structure
		config:    DefaultTemplateConfig(),
	}
}

// NewTemplateMatcherWithConfig creates a new template matcher with custom config
func NewTemplateMatcherWithConfig(templates []types.TaskTemplate, config TemplateConfig) *TemplateMatcher {
	allTemplates := templates // TODO: append builtin templates when implemented
	return &TemplateMatcher{
		templates: allTemplates,
		config:    config,
	}
}

// FindBestMatch finds the best matching template for a task
func (tm *TemplateMatcher) FindBestMatch(task *types.Task, context *types.TaskGenerationContext) *types.TaskTemplate {
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
func (tm *TemplateMatcher) FindAllMatches(task *types.Task, context *types.TaskGenerationContext) []TemplateMatch {
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
	Score    float64             `json:"score"`
	Reasons  []string            `json:"reasons"`
}

// ApplyTemplate applies a template to a task
func (tm *TemplateMatcher) ApplyTemplate(task *types.Task, template *types.TaskTemplate) types.Task {
	if template == nil {
		return *task
	}

	enhancedTask := *task
	tm.applyTemplateMetadata(&enhancedTask, template)
	tm.applyTemplateFields(&enhancedTask, template)
	tm.applyTemplateComplexity(&enhancedTask, template)
	tm.applyTemplateEffort(&enhancedTask, template)
	tm.applyTemplateCollections(&enhancedTask, template)
	tm.updateTemplateUsage(template.ID)

	return enhancedTask
}

// applyTemplateMetadata sets template metadata
func (tm *TemplateMatcher) applyTemplateMetadata(task *types.Task, template *types.TaskTemplate) {
	task.Metadata.TemplateID = template.ID
}

// applyTemplateFields applies type and priority from template
func (tm *TemplateMatcher) applyTemplateFields(task *types.Task, template *types.TaskTemplate) {
	// Extract type from template data if available
	if templateData, ok := template.TemplateData["type"].(string); ok && task.Type == "" {
		task.Type = types.TaskType(templateData)
	}

	// Extract priority from template data if available
	if templatePriority, ok := template.TemplateData["priority"].(string); ok {
		priority := types.TaskPriority(templatePriority)
		if tm.isHigherPriorityLegacy(priority, task.Priority) {
			task.Priority = priority
		}
	}
}

// applyTemplateComplexity applies complexity settings from template
func (tm *TemplateMatcher) applyTemplateComplexity(task *types.Task, template *types.TaskTemplate) {
	if task.Complexity.Level == "" && template.ComplexityLevel != "" {
		task.Complexity.Level = types.ComplexityLevel(template.ComplexityLevel)
	}

	// Merge required skills
	templateSkills := tm.jsonArrayToStringSlice(template.RequiredSkills)
	task.Complexity.RequiredSkills = tm.mergeSkills(
		task.Complexity.RequiredSkills,
		templateSkills,
	)
}

// applyTemplateEffort applies effort estimation from template
func (tm *TemplateMatcher) applyTemplateEffort(task *types.Task, template *types.TaskTemplate) {
	if task.EstimatedEffort.Hours == 0 && template.EstimatedEffortHours != nil {
		task.EstimatedEffort.Hours = *template.EstimatedEffortHours
		task.EstimatedEffort.EstimationMethod = "template_default"
	}
}

// applyTemplateCollections applies acceptance criteria and tags from template
func (tm *TemplateMatcher) applyTemplateCollections(task *types.Task, template *types.TaskTemplate) {
	// Extract and merge acceptance criteria
	templateAC := tm.extractStringArrayFromTemplate(template.TemplateData, "acceptance_criteria")
	task.AcceptanceCriteria = tm.mergeAcceptanceCriteria(task.AcceptanceCriteria, templateAC)

	// Extract and merge tags
	templateTags := tm.extractStringArrayFromTemplate(template.TemplateData, "tags")
	task.Tags = tm.mergeTags(task.Tags, templateTags)
}

// extractStringArrayFromTemplate extracts string array from template data
func (tm *TemplateMatcher) extractStringArrayFromTemplate(templateData map[string]interface{}, key string) []string {
	var result []string
	if items, ok := templateData[key].([]interface{}); ok {
		for _, item := range items {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
	}
	return result
}

// calculateMatchScore calculates how well a template matches a task
func (tm *TemplateMatcher) calculateMatchScore(task *types.Task, template *types.TaskTemplate, context *types.TaskGenerationContext) float64 {
	score := 0.0

	// Task type match (high weight)
	if templateType, ok := template.TemplateData["type"].(string); ok {
		taskType := types.TaskType(templateType)
		if task.Type == taskType {
			score += 0.3
		} else if tm.areRelatedTypes(task.Type, taskType) {
			score += 0.15
		}
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
	if tm.config.BoostPopularTemplates && template.SuccessRate != nil && *template.SuccessRate > 0.8 {
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
	// Extract keywords from applicability JSON
	keywords := []string{}
	if keywordData, ok := template.Applicability["keywords"].([]interface{}); ok {
		for _, kw := range keywordData {
			if kwStr, ok := kw.(string); ok {
				keywords = append(keywords, kwStr)
			}
		}
	}

	total := len(keywords)
	if total == 0 {
		return 0.0
	}

	for _, keyword := range keywords {
		if strings.Contains(taskContent, strings.ToLower(keyword)) {
			matches++
		}
	}

	return float64(matches) / float64(total)
}

// calculateContextMatch calculates context matching score
func (tm *TemplateMatcher) calculateContextMatch(template *types.TaskTemplate, context *types.TaskGenerationContext) float64 {
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
	// Extract team sizes from applicability JSON
	teamSizes := []int{}
	if teamData, ok := template.Applicability["team_sizes"].([]interface{}); ok {
		for _, ts := range teamData {
			if tsFloat, ok := ts.(float64); ok {
				teamSizes = append(teamSizes, int(tsFloat))
			}
		}
	}

	if len(teamSizes) > 0 {
		teamMatch := false
		for _, size := range teamSizes {
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
	// Extract tech stacks from applicability JSON
	templateTechStacks := []string{}
	if techData, ok := template.Applicability["tech_stacks"].([]interface{}); ok {
		for _, tech := range techData {
			if techStr, ok := tech.(string); ok {
				templateTechStacks = append(templateTechStacks, techStr)
			}
		}
	}

	if len(templateTechStacks) == 0 || len(techStack) == 0 {
		return 0.0
	}

	matches := 0
	for _, templateTech := range templateTechStacks {
		for _, contextTech := range techStack {
			if strings.Contains(strings.ToLower(contextTech), strings.ToLower(templateTech)) {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(templateTechStacks))
}

// calculateFuzzyMatch calculates fuzzy matching score using semantic similarity
func (tm *TemplateMatcher) calculateFuzzyMatch(task *types.Task, template *types.TaskTemplate) float64 {
	// Simple fuzzy matching based on common concepts
	conceptMap := map[string][]string{
		"api":         {"endpoint", "service", "rest", "graphql", "http"},
		"database":    {"db", "sql", "nosql", "storage", "persistence"},
		"frontend":    {"ui", "ux", "client", "web", "interface"},
		"backend":     {"server", "service", "api", "business logic"},
		"testing":     {"qa", "validation", "verification", "quality"},
		"deployment":  {"release", "production", "deploy", "publish"},
		"security":    {"auth", "authentication", "authorization", "encryption"},
		"monitoring":  {"logging", "metrics", "observability", "tracking"},
		"performance": {"optimization", "speed", "efficiency", "scalability"},
		"integration": {"connector", "bridge", "sync", "interface"},
	}

	taskContent := strings.ToLower(task.Title + " " + task.Description)
	templateDesc := ""
	if template.Description != nil {
		templateDesc = *template.Description
	}
	templateContent := strings.ToLower(template.Name + " " + templateDesc)

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
func (tm *TemplateMatcher) getMatchReasons(task *types.Task, template *types.TaskTemplate, context *types.TaskGenerationContext) []string {
	var reasons []string

	// Task type match
	if typeReason := tm.getTypeMatchReason(task, template); typeReason != "" {
		reasons = append(reasons, typeReason)
	}

	// Keyword matches
	if keywordReason := tm.getKeywordMatchReason(task, template); keywordReason != "" {
		reasons = append(reasons, keywordReason)
	}

	// Project type match
	if projectReason := tm.getProjectTypeMatchReason(template, context); projectReason != "" {
		reasons = append(reasons, projectReason)
	}

	// Tech stack matches
	if techReason := tm.getTechStackMatchReason(template, context); techReason != "" {
		reasons = append(reasons, techReason)
	}

	// High success rate
	if successReason := tm.getSuccessRateReason(template); successReason != "" {
		reasons = append(reasons, successReason)
	}

	return reasons
}

// getTypeMatchReason checks for task type matches
func (tm *TemplateMatcher) getTypeMatchReason(task *types.Task, template *types.TaskTemplate) string {
	if templateType, ok := template.TemplateData["type"].(string); ok {
		if string(task.Type) == templateType {
			return "Task type matches: " + templateType
		}
	}
	return ""
}

// getKeywordMatchReason checks for keyword matches
func (tm *TemplateMatcher) getKeywordMatchReason(task *types.Task, template *types.TaskTemplate) string {
	taskContent := strings.ToLower(task.Title + " " + task.Description)
	var matchedKeywords []string

	// Extract keywords from applicability JSON
	if keywordData, ok := template.Applicability["keywords"].([]interface{}); ok {
		for _, kw := range keywordData {
			if kwStr, ok := kw.(string); ok {
				if strings.Contains(taskContent, strings.ToLower(kwStr)) {
					matchedKeywords = append(matchedKeywords, kwStr)
				}
			}
		}
	}

	if len(matchedKeywords) > 0 {
		return "Keywords match: " + strings.Join(matchedKeywords, ", ")
	}
	return ""
}

// getProjectTypeMatchReason checks for project type matches
func (tm *TemplateMatcher) getProjectTypeMatchReason(template *types.TaskTemplate, context *types.TaskGenerationContext) string {
	if tm.matchesProjectType(template, context.ProjectType) {
		return fmt.Sprintf("Project type matches: %s", context.ProjectType)
	}
	return ""
}

// getTechStackMatchReason checks for tech stack matches
func (tm *TemplateMatcher) getTechStackMatchReason(template *types.TaskTemplate, context *types.TaskGenerationContext) string {
	var matchedTech []string

	// Extract tech stacks from applicability JSON
	if techData, ok := template.Applicability["tech_stacks"].([]interface{}); ok {
		for _, tech := range techData {
			if techStr, ok := tech.(string); ok {
				for _, contextTech := range context.TechStack {
					if strings.Contains(strings.ToLower(contextTech), strings.ToLower(techStr)) {
						matchedTech = append(matchedTech, techStr)
						break
					}
				}
			}
		}
	}

	if len(matchedTech) > 0 {
		return "Tech stack matches: " + strings.Join(matchedTech, ", ")
	}
	return ""
}

// getSuccessRateReason checks for high success rate
func (tm *TemplateMatcher) getSuccessRateReason(template *types.TaskTemplate) string {
	if template.SuccessRate != nil && *template.SuccessRate > 0.8 {
		return fmt.Sprintf("High success rate: %.1f%%", *template.SuccessRate*100)
	}
	return ""
}

// Helper functions

// areRelatedTypes checks if two task types are related
func (tm *TemplateMatcher) areRelatedTypes(type1, type2 types.TaskType) bool {
	relatedTypes := map[types.TaskType][]types.TaskType{
		types.TaskTypeLegacyImplementation: {types.TaskTypeLegacyDesign, types.TaskTypeLegacyArchitecture},
		types.TaskTypeLegacyDesign:         {types.TaskTypeLegacyImplementation, types.TaskTypeLegacyArchitecture},
		types.TaskTypeLegacyTesting:        {types.TaskTypeLegacyImplementation, types.TaskTypeLegacyReview},
		types.TaskTypeLegacyDocumentation:  {types.TaskTypeLegacyImplementation, types.TaskTypeLegacyReview},
		types.TaskTypeLegacyDeployment:     {types.TaskTypeLegacyImplementation, types.TaskTypeLegacyTesting},
		types.TaskTypeLegacyRefactoring:    {types.TaskTypeLegacyImplementation, types.TaskTypeLegacyArchitecture},
		types.TaskTypeLegacyIntegration:    {types.TaskTypeLegacyImplementation, types.TaskTypeLegacyArchitecture},
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

// matchesProjectType checks if template applies to project type
func (tm *TemplateMatcher) matchesProjectType(template *types.TaskTemplate, projectType types.ProjectType) bool {
	// Extract project types from applicability JSON
	projectTypes := []string{}
	if projectTypeData, ok := template.Applicability["project_types"].([]interface{}); ok {
		for _, pt := range projectTypeData {
			if ptStr, ok := pt.(string); ok {
				projectTypes = append(projectTypes, ptStr)
			}
		}
	}

	if len(projectTypes) == 0 {
		return true // Template applies to all project types
	}

	for _, templateType := range projectTypes {
		if templateType == string(projectType) {
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
// TODO: Update to use new TaskTemplate structure with JSONObject fields
/*
func getBuiltinTemplates() []types.TaskTemplate {
	now := time.Now()

	return []types.TaskTemplate{
		{
			ID:                "api_endpoint_template",
			Name:              "API Endpoint Implementation",
			Description:       "Template for implementing REST API endpoints",
			Category:          "Implementation",
			Type:              types.TaskTypeImplementation,
			DefaultPriority:   types.TaskPriorityMedium,
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
			ID:                "frontend_component_template",
			Name:              "Frontend Component Implementation",
			Description:       "Template for implementing reusable frontend components",
			Category:          "Implementation",
			Type:              types.TaskTypeImplementation,
			DefaultPriority:   types.TaskPriorityMedium,
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
			ID:                "database_migration_template",
			Name:              "Database Schema Migration",
			Description:       "Template for database schema changes and migrations",
			Category:          "Implementation",
			Type:              types.TaskTypeImplementation,
			DefaultPriority:   types.TaskPriorityHigh,
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
			ID:                "testing_suite_template",
			Name:              "Testing Suite Implementation",
			Description:       "Template for comprehensive testing implementation",
			Category:          "Testing",
			Type:              types.TaskTypeTesting,
			DefaultPriority:   types.TaskPriorityMedium,
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
			ID:                "documentation_template",
			Name:              "Technical Documentation",
			Description:       "Template for creating comprehensive technical documentation",
			Category:          "Documentation",
			Type:              types.TaskTypeDocumentation,
			DefaultPriority:   types.TaskPriorityLow,
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
*/

// Helper methods for template enhancement

// jsonArrayToStringSlice converts JSONArray to []string
func (tm *TemplateMatcher) jsonArrayToStringSlice(jsonArr types.JSONArray) []string {
	result := make([]string, 0, len(jsonArr))
	for _, item := range jsonArr {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// isHigherPriorityLegacy checks if the first priority is higher than the second
func (tm *TemplateMatcher) isHigherPriorityLegacy(p1, p2 types.TaskPriority) bool {
	priorities := map[types.TaskPriority]int{
		types.TaskPriorityLegacyLow:      1,
		types.TaskPriorityLegacyMedium:   2,
		types.TaskPriorityLegacyHigh:     3,
		types.TaskPriorityLegacyCritical: 4,
		types.TaskPriorityLegacyBlocking: 5,
	}
	return priorities[p1] > priorities[p2]
}
