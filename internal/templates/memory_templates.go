package templates

import (
	"encoding/json"
	"fmt"
	"mcp-memory/pkg/types"
	"strings"
	"time"
)

// TemplateField represents a field in a memory template
type TemplateField struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`          // string, number, boolean, array, object
	Required     bool        `json:"required"`
	Description  string      `json:"description"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Options      []string    `json:"options,omitempty"`     // For enum-like fields
	Validation   *Validation `json:"validation,omitempty"`
}

// Validation rules for template fields
type Validation struct {
	MinLength *int     `json:"min_length,omitempty"`
	MaxLength *int     `json:"max_length,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Pattern   *string  `json:"pattern,omitempty"`   // Regex pattern
	Custom    *string  `json:"custom,omitempty"`    // Custom validation function name
}

// MemoryTemplate defines the structure for creating structured memories
type MemoryTemplate struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	Version          string          `json:"version"`
	ChunkType        types.ChunkType `json:"chunk_type"`
	RequiredFields   []TemplateField `json:"required_fields"`
	OptionalFields   []TemplateField `json:"optional_fields"`
	AutoTags         []string        `json:"auto_tags"`         // Tags automatically applied
	DefaultMetadata  map[string]interface{} `json:"default_metadata,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	UsageCount       int             `json:"usage_count"`
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
	Valid   bool                    `json:"valid"`
	Errors  []ValidationError       `json:"errors,omitempty"`
	Warnings []ValidationWarning    `json:"warnings,omitempty"`
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
		tm.createProblemTemplate(),
		tm.createSolutionTemplate(),
		tm.createArchitecturalDecisionTemplate(),
		tm.createBugFixTemplate(),
		tm.createCodeChangeTemplate(),
		tm.createLearningTemplate(),
		tm.createPerformanceTemplate(),
		tm.createSecurityTemplate(),
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
		if err := tm.validateField(field, fields[field.Name], true, result); err != nil {
			result.Valid = false
		}
	}
	
	// Validate optional fields (if provided)
	for _, field := range template.OptionalFields {
		if value, exists := fields[field.Name]; exists {
			if err := tm.validateField(field, value, false, result); err != nil {
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
func (tm *TemplateManager) CreateChunkFromTemplate(templateID, sessionID string, fields map[string]interface{}, metadata types.ChunkMetadata) (*types.ConversationChunk, error) {
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
	allTags := append(metadata.Tags, template.AutoTags...)
	metadata.Tags = allTags
	
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
func (tm *TemplateManager) validateField(field TemplateField, value interface{}, required bool, result *ValidationResult) error {
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
func (tm *TemplateManager) validateFieldType(field TemplateField, value interface{}, result *ValidationResult) error {
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
func (tm *TemplateManager) validateFieldRules(field TemplateField, value interface{}, result *ValidationResult) {
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

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}