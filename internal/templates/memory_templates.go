// Package templates provides memory templates and content formatting
// for standardizing memory storage and retrieval in the MCP Memory Server.
package templates

import (
	"encoding/json"
	"fmt"
	"lerian-mcp-memory/pkg/types"
	"strings"
	"time"
)

// TemplateField represents a field in a memory template
type TemplateField struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // string, number, boolean, array, object
	Required     bool        `json:"required"`
	Description  string      `json:"description"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Options      []string    `json:"options,omitempty"` // For enum-like fields
	Validation   *Validation `json:"validation,omitempty"`
}

// Validation rules for template fields
type Validation struct {
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Pattern   *string  `json:"pattern,omitempty"` // Regex pattern
	Custom    *string  `json:"custom,omitempty"`  // Custom validation function name
}

// MemoryTemplate defines the structure for creating structured memories
type MemoryTemplate struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Version         string                 `json:"version"`
	ChunkType       types.ChunkType        `json:"chunk_type"`
	RequiredFields  []TemplateField        `json:"required_fields"`
	OptionalFields  []TemplateField        `json:"optional_fields"`
	AutoTags        []string               `json:"auto_tags"` // Tags automatically applied
	DefaultMetadata map[string]interface{} `json:"default_metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	UsageCount      int                    `json:"usage_count"`
}

// TemplateInstance represents a completed template with user data
type TemplateInstance struct {
	TemplateID string                 `json:"template_id"`
	Fields     map[string]interface{} `json:"fields"`
	Metadata   types.ChunkMetadata    `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

// ValidationResult represents the result of template validation
type ValidationResult struct {
	Valid    bool                `json:"valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// TemplateManager manages memory templates
type TemplateManager struct {
	templates map[string]*MemoryTemplate
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	tm := &TemplateManager{
		templates: make(map[string]*MemoryTemplate),
	}

	// Load built-in templates
	tm.loadBuiltinTemplates()

	return tm
}

// loadBuiltinTemplates loads the predefined templates
func (tm *TemplateManager) loadBuiltinTemplates() {
	builtinTemplates := []*MemoryTemplate{
		// Core documentation templates
		tm.createProblemTemplate(),
		tm.createSolutionTemplate(),
		tm.createArchitecturalDecisionTemplate(),
		tm.createBugFixTemplate(),
		tm.createCodeChangeTemplate(),
		tm.createLearningTemplate(),
		tm.createPerformanceTemplate(),
		tm.createSecurityTemplate(),

		// Development workflow templates
		tm.createCodeReviewTemplate(),
		tm.createDeploymentTemplate(),
		tm.createTestingTemplate(),
		tm.createRefactoringTemplate(),
		tm.createAPIDesignTemplate(),
		tm.createDatabaseSchemaTemplate(),

		// DevOps and Infrastructure templates
		tm.createIncidentTemplate(),
		tm.createMonitoringTemplate(),
		tm.createBackupTemplate(),
		tm.createCapacityPlanningTemplate(),

		// Project Management templates
		tm.createMeetingNotesTemplate(),
		tm.createProjectMilestoneTemplate(),
		tm.createPostMortemTemplate(),
		tm.createKnowledgeTransferTemplate(),

		// AI/ML specific templates
		tm.createMLExperimentTemplate(),
		tm.createDataAnalysisTemplate(),
		tm.createModelDeploymentTemplate(),

		// AI Prompt Analysis templates (based on discovered ai-prompts structure)
		tm.createCodebaseOverviewTemplate(),
		tm.createArchitecturalAnalysisTemplate(),
		tm.createBusinessAnalysisTemplate(),
		tm.createSecurityAnalysisTemplate(),
		tm.createPerformanceAnalysisTemplate(),
		tm.createCodeQualityTemplate(),
		tm.createAPIAnalysisTemplate(),
		tm.createDatabaseOptimizationTemplate(),
		tm.createProductionReadinessTemplate(),
	}

	for _, template := range builtinTemplates {
		tm.templates[template.ID] = template
	}
}

// GetTemplate retrieves a template by ID
func (tm *TemplateManager) GetTemplate(id string) (*MemoryTemplate, error) {
	template, exists := tm.templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return template, nil
}

// ListTemplates returns all available templates
func (tm *TemplateManager) ListTemplates() []*MemoryTemplate {
	templates := make([]*MemoryTemplate, 0, len(tm.templates))
	for _, template := range tm.templates {
		templates = append(templates, template)
	}
	return templates
}

// ValidateInstance validates a template instance
func (tm *TemplateManager) ValidateInstance(templateID string, fields map[string]interface{}) *ValidationResult {
	template, err := tm.GetTemplate(templateID)
	if err != nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "template_id",
				Message: err.Error(),
				Code:    "TEMPLATE_NOT_FOUND",
			}},
		}
	}

	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
	}

	// Validate required fields
	for _, field := range template.RequiredFields {
		if err := tm.validateField(&field, fields[field.Name], true, result); err != nil {
			result.Valid = false
		}
	}

	// Validate optional fields (if provided)
	for _, field := range template.OptionalFields {
		if value, exists := fields[field.Name]; exists {
			if err := tm.validateField(&field, value, false, result); err != nil {
				// Error is already added to result.Errors by validateField
				continue
			}
		}
	}

	// Check for unknown fields
	allKnownFields := make(map[string]bool)
	for _, field := range template.RequiredFields {
		allKnownFields[field.Name] = true
	}
	for _, field := range template.OptionalFields {
		allKnownFields[field.Name] = true
	}

	for fieldName := range fields {
		if !allKnownFields[fieldName] {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   fieldName,
				Message: "Unknown field, will be ignored",
				Code:    "UNKNOWN_FIELD",
			})
		}
	}

	return result
}

// CreateChunkFromTemplate creates a conversation chunk from a template instance
func (tm *TemplateManager) CreateChunkFromTemplate(templateID, sessionID string, fields map[string]interface{}, metadata *types.ChunkMetadata) (*types.ConversationChunk, error) {
	template, err := tm.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	// Validate the instance
	validation := tm.ValidateInstance(templateID, fields)
	if !validation.Valid {
		return nil, fmt.Errorf("template validation failed: %v", validation.Errors)
	}

	// Build content from template fields
	content := tm.buildContentFromTemplate(template, fields)

	// Apply auto tags
	metadata.Tags = append(metadata.Tags, template.AutoTags...)

	// Apply default metadata
	if template.DefaultMetadata != nil {
		if metadata.ExtendedMetadata == nil {
			metadata.ExtendedMetadata = make(map[string]interface{})
		}
		for key, value := range template.DefaultMetadata {
			if _, exists := metadata.ExtendedMetadata[key]; !exists {
				metadata.ExtendedMetadata[key] = value
			}
		}
	}

	// Store template information in metadata
	if metadata.ExtendedMetadata == nil {
		metadata.ExtendedMetadata = make(map[string]interface{})
	}
	metadata.ExtendedMetadata["template_id"] = templateID
	metadata.ExtendedMetadata["template_version"] = template.Version
	metadata.ExtendedMetadata["template_fields"] = fields

	// Create the chunk
	chunk, err := types.NewConversationChunk(sessionID, content, template.ChunkType, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}

	// Update template usage count
	template.UsageCount++
	template.UpdatedAt = time.Now().UTC()

	return chunk, nil
}

// validateField validates a single field
func (tm *TemplateManager) validateField(field *TemplateField, value interface{}, required bool, result *ValidationResult) error {
	// Check if required field is missing
	if required && value == nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   field.Name,
			Message: "Required field is missing",
			Code:    "REQUIRED_FIELD_MISSING",
		})
		return fmt.Errorf("required field missing: %s", field.Name)
	}

	// Skip validation if field is not provided and not required
	if value == nil {
		return nil
	}

	// Type validation
	if err := tm.validateFieldType(field, value, result); err != nil {
		return err
	}

	// Custom validation rules
	if field.Validation != nil {
		tm.validateFieldRules(field, value, result)
	}

	return nil
}

// validateFieldType validates the field type
func (tm *TemplateManager) validateFieldType(field *TemplateField, value interface{}, result *ValidationResult) error {
	switch field.Type {
	case "string":
		if _, ok := value.(string); !ok {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field.Name,
				Message: "Field must be a string",
				Code:    "INVALID_TYPE",
			})
			return fmt.Errorf("invalid type for field %s", field.Name)
		}
	case "number":
		switch value.(type) {
		case float64, int, int64, float32:
			// Valid number types
		default:
			result.Errors = append(result.Errors, ValidationError{
				Field:   field.Name,
				Message: "Field must be a number",
				Code:    "INVALID_TYPE",
			})
			return fmt.Errorf("invalid type for field %s", field.Name)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field.Name,
				Message: "Field must be a boolean",
				Code:    "INVALID_TYPE",
			})
			return fmt.Errorf("invalid type for field %s", field.Name)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field.Name,
				Message: "Field must be an array",
				Code:    "INVALID_TYPE",
			})
			return fmt.Errorf("invalid type for field %s", field.Name)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field.Name,
				Message: "Field must be an object",
				Code:    "INVALID_TYPE",
			})
			return fmt.Errorf("invalid type for field %s", field.Name)
		}
	}

	return nil
}

// validateFieldRules validates custom field rules
func (tm *TemplateManager) validateFieldRules(field *TemplateField, value interface{}, result *ValidationResult) {
	validation := field.Validation

	// String length validation
	if strValue, ok := value.(string); ok {
		if validation.MinLength != nil && len(strValue) < *validation.MinLength {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field.Name,
				Message: fmt.Sprintf("Field must be at least %d characters", *validation.MinLength),
				Code:    "MIN_LENGTH_VIOLATION",
			})
		}
		if validation.MaxLength != nil && len(strValue) > *validation.MaxLength {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field.Name,
				Message: fmt.Sprintf("Field must be at most %d characters", *validation.MaxLength),
				Code:    "MAX_LENGTH_VIOLATION",
			})
		}
	}

	// Number range validation
	if numValue, ok := tm.getNumericValue(value); ok {
		if validation.Min != nil && numValue < *validation.Min {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field.Name,
				Message: fmt.Sprintf("Field must be at least %f", *validation.Min),
				Code:    "MIN_VALUE_VIOLATION",
			})
		}
		if validation.Max != nil && numValue > *validation.Max {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field.Name,
				Message: fmt.Sprintf("Field must be at most %f", *validation.Max),
				Code:    "MAX_VALUE_VIOLATION",
			})
		}
	}
}

// getNumericValue converts various number types to float64
func (tm *TemplateManager) getNumericValue(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	default:
		return 0, false
	}
}

// buildContentFromTemplate builds content string from template fields
func (tm *TemplateManager) buildContentFromTemplate(template *MemoryTemplate, fields map[string]interface{}) string {
	var content strings.Builder

	// Add template header
	content.WriteString(fmt.Sprintf("# %s\n\n", template.Name))

	// Add required fields
	if len(template.RequiredFields) > 0 {
		for _, field := range template.RequiredFields {
			if value, exists := fields[field.Name]; exists && value != nil {
				content.WriteString(fmt.Sprintf("**%s**: %s\n\n", field.Description, tm.formatFieldValue(value)))
			}
		}
	}

	// Add optional fields
	if len(template.OptionalFields) > 0 {
		hasOptionalData := false
		var optionalContent strings.Builder

		for _, field := range template.OptionalFields {
			if value, exists := fields[field.Name]; exists && value != nil {
				if !hasOptionalData {
					optionalContent.WriteString("## Additional Information\n\n")
					hasOptionalData = true
				}
				optionalContent.WriteString(fmt.Sprintf("**%s**: %s\n\n", field.Description, tm.formatFieldValue(value)))
			}
		}

		if hasOptionalData {
			content.WriteString(optionalContent.String())
		}
	}

	// Add timestamp
	content.WriteString(fmt.Sprintf("---\n*Generated from template %s at %s*", template.Name, time.Now().Format(time.RFC3339)))

	return content.String()
}

// formatFieldValue formats a field value for display
func (tm *TemplateManager) formatFieldValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []interface{}:
		items := make([]string, len(v))
		for i, item := range v {
			items[i] = fmt.Sprintf("%v", item)
		}
		return strings.Join(items, ", ")
	case map[string]interface{}:
		jsonBytes, _ := json.MarshalIndent(v, "", "  ")
		return fmt.Sprintf("```json\n%s\n```", string(jsonBytes))
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Built-in template definitions

func (tm *TemplateManager) createProblemTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "problem",
		Name:        "Problem Report",
		Description: "Template for documenting problems and issues",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeProblem,
		RequiredFields: []TemplateField{
			{
				Name:        "description",
				Type:        "string",
				Required:    true,
				Description: "What is the problem?",
				Validation: &Validation{
					MinLength: intPtr(10),
					MaxLength: intPtr(2000),
				},
			},
			{
				Name:        "impact",
				Type:        "string",
				Required:    true,
				Description: "How does this affect the system/users?",
				Options:     []string{"low", "medium", "high", "critical"},
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "error_messages",
				Type:        "array",
				Description: "Error messages encountered",
			},
			{
				Name:        "steps_to_reproduce",
				Type:        "array",
				Description: "Steps to reproduce the problem",
			},
			{
				Name:        "environment",
				Type:        "object",
				Description: "Environment details where problem occurred",
			},
			{
				Name:        "workaround",
				Type:        "string",
				Description: "Temporary workaround if available",
			},
		},
		AutoTags:  []string{"problem", "issue"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createSolutionTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "solution",
		Name:        "Solution Documentation",
		Description: "Template for documenting solutions and fixes",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeSolution,
		RequiredFields: []TemplateField{
			{
				Name:        "solution",
				Type:        "string",
				Required:    true,
				Description: "How was the problem solved?",
				Validation: &Validation{
					MinLength: intPtr(20),
				},
			},
			{
				Name:        "verification_steps",
				Type:        "array",
				Required:    true,
				Description: "How to verify the solution works",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "related_problem_id",
				Type:        "string",
				Description: "ID of the related problem chunk",
			},
			{
				Name:        "code_changes",
				Type:        "array",
				Description: "Code changes made",
			},
			{
				Name:        "configuration_changes",
				Type:        "object",
				Description: "Configuration changes made",
			},
			{
				Name:        "prevention_measures",
				Type:        "string",
				Description: "How to prevent this problem in the future",
			},
			{
				Name:        "side_effects",
				Type:        "array",
				Description: "Any side effects or considerations",
			},
		},
		AutoTags:  []string{"solution", "fix"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createArchitecturalDecisionTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "architectural_decision",
		Name:        "Architectural Decision Record",
		Description: "Template for documenting architectural decisions",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeArchitectureDecision,
		RequiredFields: []TemplateField{
			{
				Name:        "decision",
				Type:        "string",
				Required:    true,
				Description: "What was decided?",
			},
			{
				Name:        "rationale",
				Type:        "string",
				Required:    true,
				Description: "Why was this decision made?",
			},
			{
				Name:        "status",
				Type:        "string",
				Required:    true,
				Description: "Status of the decision",
				Options:     []string{"proposed", "accepted", "deprecated", "superseded"},
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "alternatives_considered",
				Type:        "array",
				Description: "What other options were evaluated?",
			},
			{
				Name:        "trade_offs",
				Type:        "object",
				Description: "What are the trade-offs?",
			},
			{
				Name:        "consequences",
				Type:        "array",
				Description: "Expected consequences of this decision",
			},
			{
				Name:        "review_date",
				Type:        "string",
				Description: "When should this decision be reviewed?",
			},
			{
				Name:        "stakeholders",
				Type:        "array",
				Description: "Who was involved in this decision?",
			},
		},
		AutoTags:  []string{"architecture", "decision", "adr"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createBugFixTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "bug_fix",
		Name:        "Bug Fix Documentation",
		Description: "Template for documenting bug fixes",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeCodeChange,
		RequiredFields: []TemplateField{
			{
				Name:        "bug_description",
				Type:        "string",
				Required:    true,
				Description: "What was the bug?",
			},
			{
				Name:        "fix_description",
				Type:        "string",
				Required:    true,
				Description: "How was it fixed?",
			},
			{
				Name:        "root_cause",
				Type:        "string",
				Required:    true,
				Description: "What was the root cause?",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "reproduction_steps",
				Type:        "array",
				Description: "Steps to reproduce the bug",
			},
			{
				Name:        "fix_complexity",
				Type:        "string",
				Description: "Complexity of the fix",
				Options:     []string{"simple", "moderate", "complex"},
			},
			{
				Name:        "testing_performed",
				Type:        "array",
				Description: "Testing performed to verify the fix",
			},
			{
				Name:        "regression_risk",
				Type:        "string",
				Description: "Risk of introducing regressions",
				Options:     []string{"low", "medium", "high"},
			},
		},
		AutoTags:  []string{"bug-fix", "debugging"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createCodeChangeTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "code_change",
		Name:        "Code Change Documentation",
		Description: "Template for documenting significant code changes",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeCodeChange,
		RequiredFields: []TemplateField{
			{
				Name:        "change_description",
				Type:        "string",
				Required:    true,
				Description: "What code changes were made?",
			},
			{
				Name:        "motivation",
				Type:        "string",
				Required:    true,
				Description: "Why were these changes made?",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "files_changed",
				Type:        "array",
				Description: "List of files that were modified",
			},
			{
				Name:        "breaking_changes",
				Type:        "boolean",
				Description: "Does this introduce breaking changes?",
			},
			{
				Name:        "performance_impact",
				Type:        "string",
				Description: "Expected performance impact",
				Options:     []string{"positive", "neutral", "negative", "unknown"},
			},
			{
				Name:        "migration_notes",
				Type:        "string",
				Description: "Notes for migrating existing code",
			},
		},
		AutoTags:  []string{"code-change", "refactor"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createLearningTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "learning",
		Name:        "Learning Documentation",
		Description: "Template for documenting lessons learned",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "lesson",
				Type:        "string",
				Required:    true,
				Description: "What was learned?",
			},
			{
				Name:        "context",
				Type:        "string",
				Required:    true,
				Description: "In what context was this learned?",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "previous_understanding",
				Type:        "string",
				Description: "What was the previous understanding?",
			},
			{
				Name:        "new_insights",
				Type:        "array",
				Description: "Key insights gained",
			},
			{
				Name:        "applicable_scenarios",
				Type:        "array",
				Description: "Where else might this apply?",
			},
			{
				Name:        "confidence_level",
				Type:        "string",
				Description: "How confident are you in this learning?",
				Options:     []string{"low", "medium", "high"},
			},
		},
		AutoTags:  []string{"learning", "insight"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createPerformanceTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "performance",
		Name:        "Performance Analysis",
		Description: "Template for documenting performance analysis and optimizations",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "performance_issue",
				Type:        "string",
				Required:    true,
				Description: "What performance issue was identified?",
			},
			{
				Name:        "measurement_method",
				Type:        "string",
				Required:    true,
				Description: "How was performance measured?",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "before_metrics",
				Type:        "object",
				Description: "Performance metrics before optimization",
			},
			{
				Name:        "after_metrics",
				Type:        "object",
				Description: "Performance metrics after optimization",
			},
			{
				Name:        "optimization_techniques",
				Type:        "array",
				Description: "Optimization techniques applied",
			},
			{
				Name:        "bottlenecks_identified",
				Type:        "array",
				Description: "Performance bottlenecks identified",
			},
			{
				Name:        "tools_used",
				Type:        "array",
				Description: "Profiling or measurement tools used",
			},
		},
		AutoTags:  []string{"performance", "optimization"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createSecurityTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "security",
		Name:        "Security Analysis",
		Description: "Template for documenting security findings and fixes",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "security_issue",
				Type:        "string",
				Required:    true,
				Description: "What security issue was found?",
			},
			{
				Name:        "severity",
				Type:        "string",
				Required:    true,
				Description: "Severity level of the security issue",
				Options:     []string{"low", "medium", "high", "critical"},
			},
			{
				Name:        "remediation",
				Type:        "string",
				Required:    true,
				Description: "How was the issue remediated?",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "attack_vector",
				Type:        "string",
				Description: "How could this vulnerability be exploited?",
			},
			{
				Name:        "affected_components",
				Type:        "array",
				Description: "Which components are affected?",
			},
			{
				Name:        "detection_method",
				Type:        "string",
				Description: "How was this vulnerability detected?",
			},
			{
				Name:        "prevention_measures",
				Type:        "array",
				Description: "Measures to prevent similar issues",
			},
			{
				Name:        "compliance_impact",
				Type:        "string",
				Description: "Impact on compliance requirements",
			},
		},
		AutoTags:  []string{"security", "vulnerability"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// Development workflow templates

func (tm *TemplateManager) createCodeReviewTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "code_review",
		Name:        "Code Review Session",
		Description: "Template for documenting code review findings and feedback",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "reviewer",
				Type:        "string",
				Required:    true,
				Description: "Name of the code reviewer",
			},
			{
				Name:        "files_reviewed",
				Type:        "array",
				Required:    true,
				Description: "List of files reviewed",
			},
			{
				Name:        "findings",
				Type:        "array",
				Required:    true,
				Description: "Code review findings and recommendations",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "approval_status",
				Type:        "string",
				Description: "Review approval status",
				Options:     []string{"approved", "needs_changes", "rejected"},
			},
			{
				Name:        "security_concerns",
				Type:        "array",
				Description: "Security-related concerns identified",
			},
			{
				Name:        "performance_notes",
				Type:        "array",
				Description: "Performance optimization suggestions",
			},
		},
		AutoTags:  []string{"code-review", "quality"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createDeploymentTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "deployment",
		Name:        "Deployment Documentation",
		Description: "Template for documenting deployment processes and outcomes",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeCodeChange,
		RequiredFields: []TemplateField{
			{
				Name:        "environment",
				Type:        "string",
				Required:    true,
				Description: "Target deployment environment",
				Options:     []string{"development", "staging", "production"},
			},
			{
				Name:        "version",
				Type:        "string",
				Required:    true,
				Description: "Application version being deployed",
			},
			{
				Name:        "deployment_steps",
				Type:        "array",
				Required:    true,
				Description: "Steps taken during deployment",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "rollback_plan",
				Type:        "string",
				Description: "Rollback procedure if needed",
			},
			{
				Name:        "downtime_duration",
				Type:        "string",
				Description: "Duration of any service downtime",
			},
			{
				Name:        "post_deployment_checks",
				Type:        "array",
				Description: "Verification steps performed after deployment",
			},
		},
		AutoTags:  []string{"deployment", "infrastructure"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createTestingTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "testing",
		Name:        "Testing Documentation",
		Description: "Template for documenting testing activities and results",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "test_type",
				Type:        "string",
				Required:    true,
				Description: "Type of testing performed",
				Options:     []string{"unit", "integration", "e2e", "performance", "security"},
			},
			{
				Name:        "test_results",
				Type:        "object",
				Required:    true,
				Description: "Summary of test results and metrics",
			},
			{
				Name:        "coverage_report",
				Type:        "object",
				Required:    true,
				Description: "Test coverage analysis",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "failed_tests",
				Type:        "array",
				Description: "List of failed tests with reasons",
			},
			{
				Name:        "test_improvements",
				Type:        "array",
				Description: "Suggestions for improving test suite",
			},
			{
				Name:        "flaky_tests",
				Type:        "array",
				Description: "Tests that show inconsistent behavior",
			},
		},
		AutoTags:  []string{"testing", "quality"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createRefactoringTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "refactoring",
		Name:        "Code Refactoring",
		Description: "Template for documenting refactoring activities",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeCodeChange,
		RequiredFields: []TemplateField{
			{
				Name:        "refactoring_goal",
				Type:        "string",
				Required:    true,
				Description: "Primary goal of the refactoring effort",
			},
			{
				Name:        "files_modified",
				Type:        "array",
				Required:    true,
				Description: "Files that were refactored",
			},
			{
				Name:        "improvements_made",
				Type:        "array",
				Required:    true,
				Description: "Specific improvements implemented",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "performance_impact",
				Type:        "object",
				Description: "Measured performance improvements",
			},
			{
				Name:        "breaking_changes",
				Type:        "array",
				Description: "Any breaking changes introduced",
			},
			{
				Name:        "future_refactoring",
				Type:        "array",
				Description: "Additional refactoring opportunities identified",
			},
		},
		AutoTags:  []string{"refactoring", "improvement"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createAPIDesignTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "api_design",
		Name:        "API Design Documentation",
		Description: "Template for documenting API design decisions",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeArchitectureDecision,
		RequiredFields: []TemplateField{
			{
				Name:        "endpoint_specification",
				Type:        "object",
				Required:    true,
				Description: "Detailed API endpoint specification",
			},
			{
				Name:        "design_rationale",
				Type:        "string",
				Required:    true,
				Description: "Reasoning behind API design choices",
			},
			{
				Name:        "authentication_method",
				Type:        "string",
				Required:    true,
				Description: "Authentication approach for the API",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "versioning_strategy",
				Type:        "string",
				Description: "API versioning approach",
			},
			{
				Name:        "rate_limiting",
				Type:        "object",
				Description: "Rate limiting configuration",
			},
			{
				Name:        "error_handling",
				Type:        "object",
				Description: "Error response format and codes",
			},
		},
		AutoTags:  []string{"api", "design"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createDatabaseSchemaTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "database_schema",
		Name:        "Database Schema Design",
		Description: "Template for documenting database schema changes",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeArchitectureDecision,
		RequiredFields: []TemplateField{
			{
				Name:        "schema_changes",
				Type:        "array",
				Required:    true,
				Description: "List of schema modifications made",
			},
			{
				Name:        "migration_script",
				Type:        "string",
				Required:    true,
				Description: "SQL migration script",
			},
			{
				Name:        "data_impact",
				Type:        "string",
				Required:    true,
				Description: "Impact on existing data",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "rollback_script",
				Type:        "string",
				Description: "SQL script to rollback changes",
			},
			{
				Name:        "performance_impact",
				Type:        "object",
				Description: "Expected performance impact",
			},
			{
				Name:        "indexes_added",
				Type:        "array",
				Description: "New indexes created",
			},
		},
		AutoTags:  []string{"database", "schema"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// DevOps and Infrastructure templates

func (tm *TemplateManager) createIncidentTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "incident",
		Name:        "Incident Report",
		Description: "Template for documenting production incidents",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeProblem,
		RequiredFields: []TemplateField{
			{
				Name:        "incident_summary",
				Type:        "string",
				Required:    true,
				Description: "Brief summary of the incident",
			},
			{
				Name:        "severity",
				Type:        "string",
				Required:    true,
				Description: "Incident severity level",
				Options:     []string{"critical", "high", "medium", "low"},
			},
			{
				Name:        "timeline",
				Type:        "array",
				Required:    true,
				Description: "Chronological timeline of events",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "root_cause",
				Type:        "string",
				Description: "Identified root cause of the incident",
			},
			{
				Name:        "resolution_steps",
				Type:        "array",
				Description: "Steps taken to resolve the incident",
			},
			{
				Name:        "lessons_learned",
				Type:        "array",
				Description: "Key learnings from the incident",
			},
			{
				Name:        "prevention_measures",
				Type:        "array",
				Description: "Measures to prevent similar incidents",
			},
		},
		AutoTags:  []string{"incident", "production"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createMonitoringTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "monitoring",
		Name:        "Monitoring Setup",
		Description: "Template for documenting monitoring configurations",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeArchitectureDecision,
		RequiredFields: []TemplateField{
			{
				Name:        "metrics_configured",
				Type:        "array",
				Required:    true,
				Description: "List of metrics being monitored",
			},
			{
				Name:        "alerting_rules",
				Type:        "array",
				Required:    true,
				Description: "Alert configurations and thresholds",
			},
			{
				Name:        "dashboard_setup",
				Type:        "object",
				Required:    true,
				Description: "Monitoring dashboard configuration",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "retention_policy",
				Type:        "string",
				Description: "Data retention policy for metrics",
			},
			{
				Name:        "notification_channels",
				Type:        "array",
				Description: "Alert notification channels configured",
			},
			{
				Name:        "baseline_metrics",
				Type:        "object",
				Description: "Baseline performance metrics",
			},
		},
		AutoTags:  []string{"monitoring", "observability"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createBackupTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "backup",
		Name:        "Backup Configuration",
		Description: "Template for documenting backup and recovery procedures",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeArchitectureDecision,
		RequiredFields: []TemplateField{
			{
				Name:        "backup_strategy",
				Type:        "string",
				Required:    true,
				Description: "Overall backup strategy and approach",
			},
			{
				Name:        "backup_schedule",
				Type:        "object",
				Required:    true,
				Description: "Backup frequency and timing",
			},
			{
				Name:        "recovery_procedures",
				Type:        "array",
				Required:    true,
				Description: "Step-by-step recovery procedures",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "retention_policy",
				Type:        "object",
				Description: "Backup retention and cleanup policy",
			},
			{
				Name:        "recovery_testing",
				Type:        "object",
				Description: "Recovery testing schedule and results",
			},
			{
				Name:        "backup_verification",
				Type:        "array",
				Description: "Backup integrity verification steps",
			},
		},
		AutoTags:  []string{"backup", "disaster-recovery"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createCapacityPlanningTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "capacity_planning",
		Name:        "Capacity Planning Analysis",
		Description: "Template for documenting capacity planning decisions",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "current_utilization",
				Type:        "object",
				Required:    true,
				Description: "Current resource utilization metrics",
			},
			{
				Name:        "growth_projections",
				Type:        "object",
				Required:    true,
				Description: "Projected growth and resource needs",
			},
			{
				Name:        "scaling_recommendations",
				Type:        "array",
				Required:    true,
				Description: "Recommended scaling actions",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "cost_analysis",
				Type:        "object",
				Description: "Cost implications of scaling decisions",
			},
			{
				Name:        "bottleneck_analysis",
				Type:        "array",
				Description: "Identified system bottlenecks",
			},
			{
				Name:        "timeline",
				Type:        "object",
				Description: "Timeline for implementing capacity changes",
			},
		},
		AutoTags:  []string{"capacity", "planning", "scalability"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// Project Management templates

func (tm *TemplateManager) createMeetingNotesTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "meeting_notes",
		Name:        "Meeting Notes",
		Description: "Template for documenting meeting discussions and decisions",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "meeting_purpose",
				Type:        "string",
				Required:    true,
				Description: "Purpose and agenda of the meeting",
			},
			{
				Name:        "attendees",
				Type:        "array",
				Required:    true,
				Description: "List of meeting attendees",
			},
			{
				Name:        "key_decisions",
				Type:        "array",
				Required:    true,
				Description: "Important decisions made during the meeting",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "action_items",
				Type:        "array",
				Description: "Action items with owners and deadlines",
			},
			{
				Name:        "open_questions",
				Type:        "array",
				Description: "Unresolved questions requiring follow-up",
			},
			{
				Name:        "next_steps",
				Type:        "array",
				Description: "Planned next steps and follow-up meetings",
			},
		},
		AutoTags:  []string{"meeting", "collaboration"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createProjectMilestoneTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "project_milestone",
		Name:        "Project Milestone",
		Description: "Template for documenting project milestone achievements",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "milestone_name",
				Type:        "string",
				Required:    true,
				Description: "Name and description of the milestone",
			},
			{
				Name:        "completion_status",
				Type:        "string",
				Required:    true,
				Description: "Current completion status",
				Options:     []string{"completed", "in_progress", "delayed", "blocked"},
			},
			{
				Name:        "deliverables",
				Type:        "array",
				Required:    true,
				Description: "List of deliverables for this milestone",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "challenges_faced",
				Type:        "array",
				Description: "Challenges encountered during milestone execution",
			},
			{
				Name:        "lessons_learned",
				Type:        "array",
				Description: "Key insights gained",
			},
			{
				Name:        "impact_assessment",
				Type:        "object",
				Description: "Impact on project timeline and resources",
			},
		},
		AutoTags:  []string{"milestone", "project-management"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createPostMortemTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "post_mortem",
		Name:        "Post-Mortem Analysis",
		Description: "Template for post-mortem analysis of incidents or projects",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "event_summary",
				Type:        "string",
				Required:    true,
				Description: "Summary of the event being analyzed",
			},
			{
				Name:        "what_went_well",
				Type:        "array",
				Required:    true,
				Description: "Things that worked well",
			},
			{
				Name:        "what_went_wrong",
				Type:        "array",
				Required:    true,
				Description: "Issues and problems encountered",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "root_causes",
				Type:        "array",
				Description: "Identified root causes of problems",
			},
			{
				Name:        "action_items",
				Type:        "array",
				Description: "Specific actions to prevent future issues",
			},
			{
				Name:        "process_improvements",
				Type:        "array",
				Description: "Suggested process improvements",
			},
		},
		AutoTags:  []string{"post-mortem", "retrospective"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createKnowledgeTransferTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "knowledge_transfer",
		Name:        "Knowledge Transfer Session",
		Description: "Template for documenting knowledge transfer activities",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "knowledge_area",
				Type:        "string",
				Required:    true,
				Description: "Area of knowledge being transferred",
			},
			{
				Name:        "from_person",
				Type:        "string",
				Required:    true,
				Description: "Person transferring knowledge",
			},
			{
				Name:        "to_person",
				Type:        "string",
				Required:    true,
				Description: "Person receiving knowledge",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "key_concepts",
				Type:        "array",
				Description: "Important concepts covered",
			},
			{
				Name:        "documentation_references",
				Type:        "array",
				Description: "Relevant documentation and resources",
			},
			{
				Name:        "follow_up_sessions",
				Type:        "array",
				Description: "Planned follow-up sessions",
			},
		},
		AutoTags:  []string{"knowledge-transfer", "documentation"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// AI/ML specific templates

func (tm *TemplateManager) createMLExperimentTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "ml_experiment",
		Name:        "ML Experiment Documentation",
		Description: "Template for documenting machine learning experiments",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "experiment_objective",
				Type:        "string",
				Required:    true,
				Description: "Goal and hypothesis of the experiment",
			},
			{
				Name:        "dataset_info",
				Type:        "object",
				Required:    true,
				Description: "Information about the dataset used",
			},
			{
				Name:        "model_configuration",
				Type:        "object",
				Required:    true,
				Description: "Model architecture and hyperparameters",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "performance_metrics",
				Type:        "object",
				Description: "Model performance evaluation metrics",
			},
			{
				Name:        "feature_importance",
				Type:        "array",
				Description: "Analysis of feature importance",
			},
			{
				Name:        "next_experiments",
				Type:        "array",
				Description: "Ideas for follow-up experiments",
			},
		},
		AutoTags:  []string{"ml", "experiment", "data-science"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createDataAnalysisTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "data_analysis",
		Name:        "Data Analysis Report",
		Description: "Template for documenting data analysis findings",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "analysis_question",
				Type:        "string",
				Required:    true,
				Description: "Research question or hypothesis being investigated",
			},
			{
				Name:        "data_sources",
				Type:        "array",
				Required:    true,
				Description: "Data sources and collection methods",
			},
			{
				Name:        "key_findings",
				Type:        "array",
				Required:    true,
				Description: "Main insights and discoveries",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "methodology",
				Type:        "string",
				Description: "Analysis methodology and approach",
			},
			{
				Name:        "visualizations",
				Type:        "array",
				Description: "Charts and graphs created",
			},
			{
				Name:        "recommendations",
				Type:        "array",
				Description: "Actionable recommendations based on findings",
			},
		},
		AutoTags:  []string{"data-analysis", "insights"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createModelDeploymentTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "model_deployment",
		Name:        "ML Model Deployment",
		Description: "Template for documenting model deployment processes",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeCodeChange,
		RequiredFields: []TemplateField{
			{
				Name:        "model_version",
				Type:        "string",
				Required:    true,
				Description: "Version of the model being deployed",
			},
			{
				Name:        "deployment_environment",
				Type:        "string",
				Required:    true,
				Description: "Target deployment environment",
			},
			{
				Name:        "performance_baseline",
				Type:        "object",
				Required:    true,
				Description: "Baseline performance metrics for monitoring",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "rollback_procedure",
				Type:        "string",
				Description: "Steps to rollback model deployment",
			},
			{
				Name:        "monitoring_setup",
				Type:        "object",
				Description: "Model performance monitoring configuration",
			},
			{
				Name:        "a_b_testing",
				Type:        "object",
				Description: "A/B testing configuration if applicable",
			},
		},
		AutoTags:  []string{"ml", "deployment", "production"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// AI Prompt Analysis Templates (based on discovered ai-prompts structure)

func (tm *TemplateManager) createCodebaseOverviewTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "codebase_overview",
		Name:        "Codebase Overview Analysis",
		Description: "Comprehensive codebase analysis with component mapping and architectural insights",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "tech_stack",
				Type:        "object",
				Required:    true,
				Description: "Technology stack details including languages, frameworks, and tools",
			},
			{
				Name:        "entry_points",
				Type:        "array",
				Required:    true,
				Description: "Main entry points and initialization files",
			},
			{
				Name:        "component_analysis",
				Type:        "object",
				Required:    true,
				Description: "Major components and their boundaries",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "architecture_style",
				Type:        "string",
				Description: "Overall architecture pattern (monolith, microservices, etc.)",
				Options:     []string{"monolith", "microservices", "modular-monolith"},
			},
			{
				Name:        "design_patterns",
				Type:        "array",
				Description: "Design patterns identified in the codebase",
			},
			{
				Name:        "improvement_opportunities",
				Type:        "array",
				Description: "High-level improvement suggestions",
			},
			{
				Name:        "documentation_coverage",
				Type:        "string",
				Description: "Assessment of documentation quality",
				Options:     []string{"excellent", "good", "fair", "poor"},
			},
		},
		AutoTags: []string{"codebase-overview", "analysis", "architecture"},
		DefaultMetadata: map[string]interface{}{
			"analysis_type": "codebase_overview",
			"chain_prompt":  0,
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createArchitecturalAnalysisTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "architectural_analysis",
		Name:        "Architecture Analysis",
		Description: "Deep architectural analysis with component diagrams and design decisions",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeArchitectureDecision,
		RequiredFields: []TemplateField{
			{
				Name:        "component_diagram",
				Type:        "string",
				Required:    true,
				Description: "Mermaid diagram showing system components and relationships",
				Validation: &Validation{
					MinLength: intPtr(100),
				},
			},
			{
				Name:        "data_flow",
				Type:        "string",
				Required:    true,
				Description: "Sequence diagram showing data flow through the system",
			},
			{
				Name:        "design_decisions",
				Type:        "array",
				Required:    true,
				Description: "Key architectural decisions with rationale",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "deployment_architecture",
				Type:        "string",
				Description: "Production deployment architecture diagram",
			},
			{
				Name:        "scaling_strategy",
				Type:        "object",
				Description: "Horizontal and vertical scaling approaches",
			},
			{
				Name:        "performance_characteristics",
				Type:        "object",
				Description: "Current performance metrics and bottlenecks",
			},
			{
				Name:        "security_architecture",
				Type:        "object",
				Description: "Security measures and authentication flow",
			},
		},
		AutoTags: []string{"architecture", "design-decisions", "analysis"},
		DefaultMetadata: map[string]interface{}{
			"analysis_type": "architectural_analysis",
			"chain_prompt":  1,
			"requires_deps": []string{"codebase_overview"},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createBusinessAnalysisTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "business_analysis",
		Name:        "Business Workflow Analysis",
		Description: "Business logic analysis with performance optimization and ROI assessment",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "performance_bottlenecks",
				Type:        "array",
				Required:    true,
				Description: "Identified performance issues with impact assessment",
			},
			{
				Name:        "quick_wins",
				Type:        "array",
				Required:    true,
				Description: "High-impact, low-effort improvements",
			},
			{
				Name:        "roi_analysis",
				Type:        "object",
				Required:    true,
				Description: "Return on investment analysis for improvements",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "technical_debt_score",
				Type:        "number",
				Description: "Technical debt assessment score (0-100)",
				Validation: &Validation{
					Min: floatPtr(0),
					Max: floatPtr(100),
				},
			},
			{
				Name:        "missing_features",
				Type:        "array",
				Description: "Incomplete or missing business features",
			},
			{
				Name:        "workflow_gaps",
				Type:        "array",
				Description: "Gaps in business workflow implementation",
			},
			{
				Name:        "implementation_roadmap",
				Type:        "object",
				Description: "Phased improvement implementation plan",
			},
		},
		AutoTags: []string{"business-analysis", "performance", "roi"},
		DefaultMetadata: map[string]interface{}{
			"analysis_type": "business_analysis",
			"chain_prompt":  2,
			"requires_deps": []string{"codebase_overview", "architectural_analysis"},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createSecurityAnalysisTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "security_analysis",
		Name:        "Security Vulnerability Analysis",
		Description: "Comprehensive security audit with vulnerability assessment and remediation plan",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "risk_level",
				Type:        "string",
				Required:    true,
				Description: "Overall security risk assessment",
				Options:     []string{"critical", "high", "medium", "low"},
			},
			{
				Name:        "critical_vulnerabilities",
				Type:        "array",
				Required:    true,
				Description: "Critical security vulnerabilities requiring immediate action",
			},
			{
				Name:        "remediation_plan",
				Type:        "object",
				Required:    true,
				Description: "Phased remediation plan with priorities and timelines",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "attack_surface",
				Type:        "object",
				Description: "Attack surface mapping with entry points",
			},
			{
				Name:        "compliance_status",
				Type:        "object",
				Description: "Compliance assessment (OWASP, SOC2, etc.)",
			},
			{
				Name:        "dependency_vulnerabilities",
				Type:        "array",
				Description: "Third-party dependency security issues",
			},
			{
				Name:        "security_monitoring",
				Type:        "object",
				Description: "Security monitoring and alerting recommendations",
			},
		},
		AutoTags: []string{"security", "vulnerability", "audit"},
		DefaultMetadata: map[string]interface{}{
			"analysis_type": "security_analysis",
			"chain_prompt":  3,
			"requires_deps": []string{"codebase_overview", "architectural_analysis"},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// analysisTemplateConfig provides a configuration structure for analysis templates
type analysisTemplateConfig struct {
	ID             string
	Name           string
	Description    string
	RequiredFields []TemplateField
	OptionalFields []TemplateField
	AutoTags       []string
	Category       string
}

// buildAnalysisTemplate builds an analysis template from configuration
func (tm *TemplateManager) buildAnalysisTemplate(config *analysisTemplateConfig) *MemoryTemplate {
	return &MemoryTemplate{
		ID:             config.ID,
		Name:           config.Name,
		Description:    config.Description,
		Version:        "1.0",
		ChunkType:      types.ChunkTypeAnalysis,
		RequiredFields: config.RequiredFields,
		OptionalFields: config.OptionalFields,
		AutoTags:       config.AutoTags,
		DefaultMetadata: map[string]interface{}{
			"analysis_type": config.Category,
			"category":      "infrastructure",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

func (tm *TemplateManager) createPerformanceAnalysisTemplate() *MemoryTemplate {
	config := analysisTemplateConfig{
		ID:          "performance_analysis",
		Name:        "Performance Optimization Analysis",
		Description: "Performance profiling and optimization recommendations",
		RequiredFields: []TemplateField{
			{
				Name:        "performance_metrics",
				Type:        "object",
				Required:    true,
				Description: "Current performance metrics (response time, throughput, etc.)",
			},
			{
				Name:        "bottleneck_analysis",
				Type:        "array",
				Required:    true,
				Description: "Identified performance bottlenecks with impact",
			},
			{
				Name:        "optimization_recommendations",
				Type:        "array",
				Required:    true,
				Description: "Specific optimization recommendations with expected impact",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "benchmark_results",
				Type:        "object",
				Description: "Performance benchmark test results",
			},
			{
				Name:        "scalability_assessment",
				Type:        "object",
				Description: "Current and projected scalability characteristics",
			},
			{
				Name:        "resource_utilization",
				Type:        "object",
				Description: "CPU, memory, and I/O utilization analysis",
			},
			{
				Name:        "caching_strategy",
				Type:        "object",
				Description: "Caching implementation and optimization opportunities",
			},
		},
		AutoTags: []string{"performance", "optimization", "analysis"},
		Category: "infrastructure",
	}

	return tm.buildAnalysisTemplate(&config)
}

func (tm *TemplateManager) createCodeQualityTemplate() *MemoryTemplate {
	config := analysisTemplateConfig{
		ID:          "code_quality",
		Name:        "Code Quality Assessment",
		Description: "Code quality analysis with maintainability and technical debt assessment",
		RequiredFields: []TemplateField{
			{
				Name:        "quality_score",
				Type:        "number",
				Required:    true,
				Description: "Overall code quality score (0-100)",
				Validation: &Validation{
					Min: floatPtr(0),
					Max: floatPtr(100),
				},
			},
			{
				Name:        "code_smells",
				Type:        "array",
				Required:    true,
				Description: "Identified code smells and anti-patterns",
			},
			{
				Name:        "refactoring_opportunities",
				Type:        "array",
				Required:    true,
				Description: "Code refactoring opportunities with priority",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "test_coverage",
				Type:        "object",
				Description: "Test coverage analysis and gaps",
			},
			{
				Name:        "duplication_analysis",
				Type:        "object",
				Description: "Code duplication detection and consolidation opportunities",
			},
			{
				Name:        "complexity_metrics",
				Type:        "object",
				Description: "Cyclomatic complexity and maintainability metrics",
			},
			{
				Name:        "documentation_quality",
				Type:        "object",
				Description: "Code documentation coverage and quality assessment",
			},
		},
		AutoTags: []string{"code-quality", "refactoring", "technical-debt"},
		Category: "maintainability",
	}

	return tm.buildAnalysisTemplate(&config)
}

func (tm *TemplateManager) createAPIAnalysisTemplate() *MemoryTemplate {
	config := analysisTemplateConfig{
		ID:          "api_analysis",
		Name:        "API Contract Analysis",
		Description: "API design analysis with contract consistency and documentation assessment",
		RequiredFields: []TemplateField{
			{
				Name:        "endpoint_inventory",
				Type:        "array",
				Required:    true,
				Description: "Complete inventory of API endpoints with methods and parameters",
			},
			{
				Name:        "consistency_analysis",
				Type:        "object",
				Required:    true,
				Description: "API consistency assessment across endpoints",
			},
			{
				Name:        "documentation_status",
				Type:        "object",
				Required:    true,
				Description: "API documentation coverage and quality",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "versioning_strategy",
				Type:        "object",
				Description: "API versioning approach and recommendations",
			},
			{
				Name:        "error_handling",
				Type:        "object",
				Description: "Error response consistency and best practices",
			},
			{
				Name:        "authentication_flow",
				Type:        "object",
				Description: "Authentication and authorization implementation",
			},
			{
				Name:        "rate_limiting",
				Type:        "object",
				Description: "Rate limiting and throttling implementation",
			},
		},
		AutoTags: []string{"api", "documentation", "consistency"},
		Category: "interface",
	}

	return tm.buildAnalysisTemplate(&config)
}

func (tm *TemplateManager) createDatabaseOptimizationTemplate() *MemoryTemplate {
	config := analysisTemplateConfig{
		ID:          "database_optimization",
		Name:        "Database Optimization Analysis",
		Description: "Database performance analysis with query optimization and schema recommendations",
		RequiredFields: []TemplateField{
			{
				Name:        "schema_analysis",
				Type:        "object",
				Required:    true,
				Description: "Database schema structure and relationship analysis",
			},
			{
				Name:        "query_performance",
				Type:        "array",
				Required:    true,
				Description: "Slow query analysis with optimization recommendations",
			},
			{
				Name:        "indexing_strategy",
				Type:        "object",
				Required:    true,
				Description: "Current and recommended indexing strategy",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "normalization_assessment",
				Type:        "object",
				Description: "Database normalization and denormalization opportunities",
			},
			{
				Name:        "connection_pooling",
				Type:        "object",
				Description: "Connection pooling configuration and optimization",
			},
			{
				Name:        "backup_strategy",
				Type:        "object",
				Description: "Database backup and recovery strategy assessment",
			},
			{
				Name:        "scaling_recommendations",
				Type:        "object",
				Description: "Database scaling approach (vertical, horizontal, sharding)",
			},
		},
		AutoTags: []string{"database", "optimization", "performance"},
		Category: "data",
	}

	return tm.buildAnalysisTemplate(&config)
}

func (tm *TemplateManager) createProductionReadinessTemplate() *MemoryTemplate {
	return &MemoryTemplate{
		ID:          "production_readiness",
		Name:        "Production Readiness Audit",
		Description: "Comprehensive production readiness assessment with deployment checklist",
		Version:     "1.0",
		ChunkType:   types.ChunkTypeAnalysis,
		RequiredFields: []TemplateField{
			{
				Name:        "readiness_score",
				Type:        "number",
				Required:    true,
				Description: "Overall production readiness score (0-100)",
				Validation: &Validation{
					Min: floatPtr(0),
					Max: floatPtr(100),
				},
			},
			{
				Name:        "critical_blockers",
				Type:        "array",
				Required:    true,
				Description: "Critical issues preventing production deployment",
			},
			{
				Name:        "deployment_checklist",
				Type:        "array",
				Required:    true,
				Description: "Complete deployment readiness checklist",
			},
		},
		OptionalFields: []TemplateField{
			{
				Name:        "monitoring_setup",
				Type:        "object",
				Description: "Production monitoring and alerting configuration",
			},
			{
				Name:        "disaster_recovery",
				Type:        "object",
				Description: "Disaster recovery and business continuity plan",
			},
			{
				Name:        "compliance_checklist",
				Type:        "array",
				Description: "Regulatory and compliance requirements assessment",
			},
			{
				Name:        "rollback_strategy",
				Type:        "object",
				Description: "Deployment rollback and recovery procedures",
			},
		},
		AutoTags: []string{"production", "deployment", "readiness"},
		DefaultMetadata: map[string]interface{}{
			"analysis_type": "production_readiness",
			"category":      "deployment",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// Helper function to create float64 pointer
func floatPtr(f float64) *float64 {
	return &f
}
