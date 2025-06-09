// Package tasks provides task validation functionality.
package tasks

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"lerian-mcp-memory/pkg/types"
)

// Validator validates task quality and completeness
type Validator struct {
	config ValidationConfig
}

// ValidationConfig represents configuration for task validation
type ValidationConfig struct {
	MinTitleLength          int     `json:"min_title_length"`
	MaxTitleLength          int     `json:"max_title_length"`
	MinDescriptionLength    int     `json:"min_description_length"`
	MaxDescriptionLength    int     `json:"max_description_length"`
	MinAcceptanceCriteria   int     `json:"min_acceptance_criteria"`
	MaxAcceptanceCriteria   int     `json:"max_acceptance_criteria"`
	RequireEstimation       bool    `json:"require_estimation"`
	RequireAcceptanceCriteria bool  `json:"require_acceptance_criteria"`
	MinEstimatedHours       float64 `json:"min_estimated_hours"`
	MaxEstimatedHours       float64 `json:"max_estimated_hours"`
	ValidTaskTypes          []types.TaskType     `json:"valid_task_types"`
	ValidPriorities         []types.TaskPriority `json:"valid_priorities"`
	ForbiddenWords          []string             `json:"forbidden_words"`
	RequiredFields          []string             `json:"required_fields"`
}

// DefaultValidationConfig returns default validation configuration
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MinTitleLength:        5,
		MaxTitleLength:        100,
		MinDescriptionLength:  10,
		MaxDescriptionLength:  2000,
		MinAcceptanceCriteria: 1,
		MaxAcceptanceCriteria: 10,
		RequireEstimation:     true,
		RequireAcceptanceCriteria: true,
		MinEstimatedHours:     0.5,
		MaxEstimatedHours:     160.0, // 4 weeks
		ValidTaskTypes: []types.TaskType{
			types.TaskTypeImplementation,
			types.TaskTypeDesign,
			types.TaskTypeTesting,
			types.TaskTypeDocumentation,
			types.TaskTypeResearch,
			types.TaskTypeReview,
			types.TaskTypeDeployment,
			types.TaskTypeArchitecture,
			types.TaskTypeBugFix,
			types.TaskTypeRefactoring,
			types.TaskTypeIntegration,
			types.TaskTypeAnalysis,
		},
		ValidPriorities: []types.TaskPriority{
			types.TaskPriorityLow,
			types.TaskPriorityMedium,
			types.TaskPriorityHigh,
			types.TaskPriorityCritical,
			types.TaskPriorityBlocking,
		},
		ForbiddenWords: []string{
			"todo", "fixme", "hack", "temp", "temporary", "placeholder",
			"test test", "sample", "example", "dummy", "fake",
		},
		RequiredFields: []string{"title", "description", "type", "priority"},
	}
}

// NewValidator creates a new task validator
func NewValidator() *Validator {
	return &Validator{
		config: DefaultValidationConfig(),
	}
}

// NewValidatorWithConfig creates a new task validator with custom config
func NewValidatorWithConfig(config ValidationConfig) *Validator {
	return &Validator{
		config: config,
	}
}

// ValidateTask validates a task and returns validation results
func (v *Validator) ValidateTask(task *types.Task) types.TaskValidationResult {
	if task == nil {
		return types.TaskValidationResult{
			IsValid: false,
			Errors: []types.ValidationError{
				{
					Field:    "task",
					Type:     "null_task",
					Message:  "Task cannot be nil",
					Severity: "critical",
					Code:     "TASK_NULL",
				},
			},
			Score: 0.0,
		}
	}

	result := types.TaskValidationResult{
		IsValid:     true,
		Errors:      []types.ValidationError{},
		Warnings:    []types.ValidationWarning{},
		Suggestions: []string{},
		Score:       1.0,
	}

	// Validate required fields
	v.validateRequiredFields(task, &result)

	// Validate title
	v.validateTitle(task, &result)

	// Validate description
	v.validateDescription(task, &result)

	// Validate task type
	v.validateTaskType(task, &result)

	// Validate priority
	v.validatePriority(task, &result)

	// Validate acceptance criteria
	v.validateAcceptanceCriteria(task, &result)

	// Validate effort estimation
	v.validateEffortEstimation(task, &result)

	// Validate dependencies
	v.validateDependencies(task, &result)

	// Validate tags
	v.validateTags(task, &result)

	// Check for forbidden words
	v.checkForbiddenWords(task, &result)

	// Validate task consistency
	v.validateTaskConsistency(task, &result)

	// Calculate overall validation score
	result.Score = v.calculateValidationScore(&result)
	result.IsValid = len(result.Errors) == 0 && result.Score >= 0.6

	// Generate suggestions for improvement
	v.generateSuggestions(task, &result)

	return result
}

// validateRequiredFields validates that all required fields are present
func (v *Validator) validateRequiredFields(task *types.Task, result *types.TaskValidationResult) {
	for _, field := range v.config.RequiredFields {
		switch field {
		case "title":
			if task.Title == "" {
				v.addError(result, "title", "required_field", "Title is required", "critical", "TITLE_REQUIRED")
			}
		case "description":
			if task.Description == "" {
				v.addError(result, "description", "required_field", "Description is required", "critical", "DESCRIPTION_REQUIRED")
			}
		case "type":
			if task.Type == "" {
				v.addError(result, "type", "required_field", "Task type is required", "critical", "TYPE_REQUIRED")
			}
		case "priority":
			if task.Priority == "" {
				v.addError(result, "priority", "required_field", "Priority is required", "critical", "PRIORITY_REQUIRED")
			}
		}
	}
}

// validateTitle validates the task title
func (v *Validator) validateTitle(task *types.Task, result *types.TaskValidationResult) {
	title := strings.TrimSpace(task.Title)
	titleLength := utf8.RuneCountInString(title)

	// Check length
	if titleLength < v.config.MinTitleLength {
		v.addError(result, "title", "length", 
			fmt.Sprintf("Title too short (minimum %d characters)", v.config.MinTitleLength),
			"high", "TITLE_TOO_SHORT")
	}

	if titleLength > v.config.MaxTitleLength {
		v.addError(result, "title", "length",
			fmt.Sprintf("Title too long (maximum %d characters)", v.config.MaxTitleLength),
			"medium", "TITLE_TOO_LONG")
	}

	// Check for actionable language
	if !v.hasActionableLanguage(title) {
		v.addWarning(result, "title", "clarity",
			"Title should start with an action verb (e.g., 'Implement', 'Design', 'Fix')",
			"Start title with an action verb to make it more actionable",
			"TITLE_NOT_ACTIONABLE")
	}

	// Check for specificity
	vaguePhrases := []string{"something", "stuff", "things", "various", "misc", "general"}
	titleLower := strings.ToLower(title)
	for _, phrase := range vaguePhrases {
		if strings.Contains(titleLower, phrase) {
			v.addWarning(result, "title", "specificity",
				"Title contains vague language that should be made more specific",
				"Replace vague terms with specific details",
				"TITLE_VAGUE")
			break
		}
	}
}

// validateDescription validates the task description
func (v *Validator) validateDescription(task *types.Task, result *types.TaskValidationResult) {
	description := strings.TrimSpace(task.Description)
	descLength := utf8.RuneCountInString(description)

	// Check length
	if descLength < v.config.MinDescriptionLength {
		v.addError(result, "description", "length",
			fmt.Sprintf("Description too short (minimum %d characters)", v.config.MinDescriptionLength),
			"high", "DESCRIPTION_TOO_SHORT")
	}

	if descLength > v.config.MaxDescriptionLength {
		v.addWarning(result, "description", "length",
			fmt.Sprintf("Description very long (maximum %d characters recommended)", v.config.MaxDescriptionLength),
			"Consider breaking down into smaller, more focused tasks",
			"DESCRIPTION_TOO_LONG")
	}

	// Check for completeness indicators
	completenessKeywords := []string{"what", "why", "how", "when", "where", "who"}
	hasCompleteness := 0
	descLower := strings.ToLower(description)
	
	for _, keyword := range completenessKeywords {
		if strings.Contains(descLower, keyword) {
			hasCompleteness++
		}
	}

	if hasCompleteness < 2 {
		v.addWarning(result, "description", "completeness",
			"Description should address what, why, and how the task should be completed",
			"Add more details about the task's purpose and implementation approach",
			"DESCRIPTION_INCOMPLETE")
	}

	// Check for technical details
	technicalKeywords := []string{
		"api", "database", "interface", "component", "service", "endpoint",
		"method", "function", "class", "module", "library", "framework",
	}
	
	hasTechnicalDetails := false
	for _, keyword := range technicalKeywords {
		if strings.Contains(descLower, keyword) {
			hasTechnicalDetails = true
			break
		}
	}

	if task.Type == types.TaskTypeImplementation && !hasTechnicalDetails {
		v.addWarning(result, "description", "technical_detail",
			"Implementation tasks should include technical details",
			"Add specific technical requirements or implementation notes",
			"DESCRIPTION_LACKS_TECHNICAL_DETAIL")
	}
}

// validateTaskType validates the task type
func (v *Validator) validateTaskType(task *types.Task, result *types.TaskValidationResult) {
	if task.Type == "" {
		return // Already handled in required fields
	}

	// Check if type is valid
	validType := false
	for _, validTaskType := range v.config.ValidTaskTypes {
		if task.Type == validTaskType {
			validType = true
			break
		}
	}

	if !validType {
		v.addError(result, "type", "invalid_value",
			fmt.Sprintf("Invalid task type: %s", task.Type),
			"high", "INVALID_TASK_TYPE")
	}

	// Check type consistency with content
	v.validateTypeConsistency(task, result)
}

// validatePriority validates the task priority
func (v *Validator) validatePriority(task *types.Task, result *types.TaskValidationResult) {
	if task.Priority == "" {
		return // Already handled in required fields
	}

	// Check if priority is valid
	validPriority := false
	for _, validPrio := range v.config.ValidPriorities {
		if task.Priority == validPrio {
			validPriority = true
			break
		}
	}

	if !validPriority {
		v.addError(result, "priority", "invalid_value",
			fmt.Sprintf("Invalid priority: %s", task.Priority),
			"high", "INVALID_PRIORITY")
	}

	// Check priority consistency
	v.validatePriorityConsistency(task, result)
}

// validateAcceptanceCriteria validates acceptance criteria
func (v *Validator) validateAcceptanceCriteria(task *types.Task, result *types.TaskValidationResult) {
	criteriaCount := len(task.AcceptanceCriteria)

	if v.config.RequireAcceptanceCriteria && criteriaCount == 0 {
		v.addError(result, "acceptance_criteria", "required_field",
			"Acceptance criteria are required",
			"high", "ACCEPTANCE_CRITERIA_REQUIRED")
		return
	}

	if criteriaCount < v.config.MinAcceptanceCriteria {
		v.addWarning(result, "acceptance_criteria", "count",
			fmt.Sprintf("Consider adding more acceptance criteria (recommended minimum: %d)", v.config.MinAcceptanceCriteria),
			"Add specific, testable criteria for task completion",
			"INSUFFICIENT_ACCEPTANCE_CRITERIA")
	}

	if criteriaCount > v.config.MaxAcceptanceCriteria {
		v.addWarning(result, "acceptance_criteria", "count",
			fmt.Sprintf("Too many acceptance criteria (recommended maximum: %d)", v.config.MaxAcceptanceCriteria),
			"Consider consolidating or breaking down the task",
			"TOO_MANY_ACCEPTANCE_CRITERIA")
	}

	// Validate individual criteria
	for i, criteria := range task.AcceptanceCriteria {
		v.validateSingleCriteria(criteria, i, result)
	}
}

// validateEffortEstimation validates effort estimation
func (v *Validator) validateEffortEstimation(task *types.Task, result *types.TaskValidationResult) {
	if v.config.RequireEstimation && task.EstimatedEffort.Hours == 0 {
		v.addError(result, "estimated_effort", "required_field",
			"Effort estimation is required",
			"medium", "EFFORT_ESTIMATION_REQUIRED")
		return
	}

	if task.EstimatedEffort.Hours != 0 {
		// Check effort bounds
		if task.EstimatedEffort.Hours < v.config.MinEstimatedHours {
			v.addWarning(result, "estimated_effort", "value",
				fmt.Sprintf("Estimated effort seems too low (minimum %g hours)", v.config.MinEstimatedHours),
				"Consider if the task can really be completed in less time",
				"EFFORT_TOO_LOW")
		}

		if task.EstimatedEffort.Hours > v.config.MaxEstimatedHours {
			v.addWarning(result, "estimated_effort", "value",
				fmt.Sprintf("Estimated effort seems too high (maximum %g hours)", v.config.MaxEstimatedHours),
				"Consider breaking down into smaller tasks",
				"EFFORT_TOO_HIGH")
		}

		// Check confidence level
		if task.EstimatedEffort.Confidence < 0.3 {
			v.addWarning(result, "estimated_effort", "confidence",
				"Low confidence in effort estimation",
				"Consider researching or prototyping to improve estimation confidence",
				"LOW_ESTIMATION_CONFIDENCE")
		}
	}
}

// validateDependencies validates task dependencies
func (v *Validator) validateDependencies(task *types.Task, result *types.TaskValidationResult) {
	// Check for self-dependencies
	for _, depID := range task.Dependencies {
		if depID == task.ID {
			v.addError(result, "dependencies", "circular_dependency",
				"Task cannot depend on itself",
				"critical", "SELF_DEPENDENCY")
		}
	}

	// Check for duplicate dependencies
	depMap := make(map[string]bool)
	for _, depID := range task.Dependencies {
		if depMap[depID] {
			v.addWarning(result, "dependencies", "duplicate",
				"Duplicate dependency found",
				"Remove duplicate dependencies",
				"DUPLICATE_DEPENDENCY")
		}
		depMap[depID] = true
	}

	// Warn about too many dependencies
	if len(task.Dependencies) > 5 {
		v.addWarning(result, "dependencies", "complexity",
			"Task has many dependencies which may indicate complexity",
			"Consider simplifying or breaking down the task",
			"TOO_MANY_DEPENDENCIES")
	}
}

// validateTags validates task tags
func (v *Validator) validateTags(task *types.Task, result *types.TaskValidationResult) {
	// Check for reasonable number of tags
	if len(task.Tags) > 10 {
		v.addWarning(result, "tags", "count",
			"Too many tags may reduce their effectiveness",
			"Use fewer, more meaningful tags",
			"TOO_MANY_TAGS")
	}

	// Check for empty or very short tags
	for _, tag := range task.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			v.addWarning(result, "tags", "empty",
				"Empty tag found",
				"Remove empty tags",
				"EMPTY_TAG")
		} else if len(tag) < 2 {
			v.addWarning(result, "tags", "length",
				"Very short tag found",
				"Use more descriptive tags",
				"SHORT_TAG")
		}
	}
}

// checkForbiddenWords checks for forbidden words in task content
func (v *Validator) checkForbiddenWords(task *types.Task, result *types.TaskValidationResult) {
	content := strings.ToLower(task.Title + " " + task.Description)
	
	for _, word := range v.config.ForbiddenWords {
		if strings.Contains(content, word) {
			v.addWarning(result, "content", "forbidden_word",
				fmt.Sprintf("Contains potentially problematic word: '%s'", word),
				"Replace with more professional language",
				"FORBIDDEN_WORD")
		}
	}
}

// validateTaskConsistency validates consistency between different task fields
func (v *Validator) validateTaskConsistency(task *types.Task, result *types.TaskValidationResult) {
	// Check priority vs complexity consistency
	if task.Priority == types.TaskPriorityCritical && task.Complexity.Level == types.ComplexityTrivial {
		v.addWarning(result, "priority", "consistency",
			"Critical priority with trivial complexity seems inconsistent",
			"Review priority or complexity assessment",
			"PRIORITY_COMPLEXITY_MISMATCH")
	}

	// Check estimated effort vs complexity consistency
	if task.Complexity.Level == types.ComplexityVeryComplex && task.EstimatedEffort.Hours < 8 {
		v.addWarning(result, "estimated_effort", "consistency",
			"Very complex task with low effort estimate seems inconsistent",
			"Review complexity assessment or effort estimation",
			"EFFORT_COMPLEXITY_MISMATCH")
	}
}

// Helper validation functions

// validateTypeConsistency validates that task type matches content
func (v *Validator) validateTypeConsistency(task *types.Task, result *types.TaskValidationResult) {
	content := strings.ToLower(task.Title + " " + task.Description)
	
	typeKeywords := map[types.TaskType][]string{
		types.TaskTypeImplementation: {"implement", "build", "create", "develop", "code"},
		types.TaskTypeDesign:         {"design", "mockup", "wireframe", "prototype", "ui", "ux"},
		types.TaskTypeTesting:        {"test", "qa", "verify", "validate", "check"},
		types.TaskTypeDocumentation:  {"document", "readme", "guide", "manual", "docs"},
		types.TaskTypeResearch:       {"research", "investigate", "analyze", "study", "explore"},
		types.TaskTypeReview:         {"review", "audit", "inspect", "evaluate", "assess"},
		types.TaskTypeDeployment:     {"deploy", "release", "publish", "launch", "ship"},
		types.TaskTypeArchitecture:   {"architecture", "design", "structure", "system", "framework"},
		types.TaskTypeBugFix:         {"fix", "bug", "issue", "error", "defect"},
		types.TaskTypeRefactoring:    {"refactor", "cleanup", "improve", "optimize", "restructure"},
	}

	if keywords, exists := typeKeywords[task.Type]; exists {
		hasMatchingKeyword := false
		for _, keyword := range keywords {
			if strings.Contains(content, keyword) {
				hasMatchingKeyword = true
				break
			}
		}
		
		if !hasMatchingKeyword {
			v.addWarning(result, "type", "consistency",
				fmt.Sprintf("Task type '%s' doesn't match content keywords", task.Type),
				"Ensure task type accurately reflects the work to be done",
				"TYPE_CONTENT_MISMATCH")
		}
	}
}

// validatePriorityConsistency validates priority consistency
func (v *Validator) validatePriorityConsistency(task *types.Task, result *types.TaskValidationResult) {
	content := strings.ToLower(task.Title + " " + task.Description)
	
	urgentKeywords := []string{"urgent", "critical", "asap", "immediately", "emergency", "blocker"}
	hasUrgentKeywords := false
	for _, keyword := range urgentKeywords {
		if strings.Contains(content, keyword) {
			hasUrgentKeywords = true
			break
		}
	}

	if hasUrgentKeywords && (task.Priority == types.TaskPriorityLow || task.Priority == types.TaskPriorityMedium) {
		v.addWarning(result, "priority", "consistency",
			"Content suggests urgency but priority is not high",
			"Consider increasing priority or removing urgent language",
			"PRIORITY_URGENCY_MISMATCH")
	}
}

// validateSingleCriteria validates a single acceptance criteria
func (v *Validator) validateSingleCriteria(criteria string, index int, result *types.TaskValidationResult) {
	criteria = strings.TrimSpace(criteria)
	
	if criteria == "" {
		v.addWarning(result, "acceptance_criteria", "empty",
			fmt.Sprintf("Empty acceptance criteria at index %d", index),
			"Remove empty criteria or add meaningful content",
			"EMPTY_CRITERIA")
		return
	}

	// Check for testable language
	testableKeywords := []string{
		"should", "must", "can", "will", "is", "are", "has", "have",
		"verify", "confirm", "ensure", "check", "validate",
	}
	
	criteriaLower := strings.ToLower(criteria)
	hasTestableLanguage := false
	for _, keyword := range testableKeywords {
		if strings.Contains(criteriaLower, keyword) {
			hasTestableLanguage = true
			break
		}
	}

	if !hasTestableLanguage {
		v.addWarning(result, "acceptance_criteria", "testability",
			fmt.Sprintf("Criteria '%s' may not be testable", criteria),
			"Use clear, testable language for acceptance criteria",
			"CRITERIA_NOT_TESTABLE")
	}

	// Check length
	if len(criteria) < 10 {
		v.addWarning(result, "acceptance_criteria", "length",
			"Acceptance criteria seems too short to be specific",
			"Add more specific details to the criteria",
			"CRITERIA_TOO_SHORT")
	}
}

// hasActionableLanguage checks if text contains actionable language
func (v *Validator) hasActionableLanguage(text string) bool {
	actionVerbs := []string{
		"implement", "create", "build", "develop", "design", "fix", "update",
		"add", "remove", "modify", "refactor", "test", "deploy", "setup",
		"configure", "install", "integrate", "optimize", "analyze", "research",
		"document", "review", "audit", "validate", "verify", "enhance",
	}

	textLower := strings.ToLower(text)
	for _, verb := range actionVerbs {
		if strings.HasPrefix(textLower, verb) || strings.Contains(textLower, " "+verb) {
			return true
		}
	}

	return false
}

// calculateValidationScore calculates overall validation score
func (v *Validator) calculateValidationScore(result *types.TaskValidationResult) float64 {
	if len(result.Errors) > 0 {
		// Deduct points for errors
		errorDeduction := float64(len(result.Errors)) * 0.2
		return 1.0 - errorDeduction
	}

	// Deduct smaller amounts for warnings
	warningDeduction := float64(len(result.Warnings)) * 0.1
	score := 1.0 - warningDeduction

	if score < 0 {
		score = 0
	}

	return score
}

// generateSuggestions generates improvement suggestions
func (v *Validator) generateSuggestions(task *types.Task, result *types.TaskValidationResult) {
	if len(result.Errors) > 0 {
		result.Suggestions = append(result.Suggestions, "Fix all validation errors before proceeding")
	}

	if len(task.AcceptanceCriteria) == 0 {
		result.Suggestions = append(result.Suggestions, "Add specific, testable acceptance criteria")
	}

	if task.EstimatedEffort.Hours == 0 {
		result.Suggestions = append(result.Suggestions, "Provide effort estimation to help with planning")
	}

	if len(task.Tags) == 0 {
		result.Suggestions = append(result.Suggestions, "Add relevant tags to improve task organization")
	}

	if task.Complexity.Level == "" {
		result.Suggestions = append(result.Suggestions, "Assess task complexity to help with resource allocation")
	}
}

// Helper functions for adding errors and warnings

func (v *Validator) addError(result *types.TaskValidationResult, field, errorType, message, severity, code string) {
	result.Errors = append(result.Errors, types.ValidationError{
		Field:    field,
		Type:     errorType,
		Message:  message,
		Severity: severity,
		Code:     code,
	})
}

func (v *Validator) addWarning(result *types.TaskValidationResult, field, warningType, message, suggestion, code string) {
	result.Warnings = append(result.Warnings, types.ValidationWarning{
		Field:      field,
		Type:       warningType,
		Message:    message,
		Suggestion: suggestion,
		Code:       code,
	})
}