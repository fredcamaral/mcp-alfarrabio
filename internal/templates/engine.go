// Package templates provides template engine for task generation
package templates

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"

	"lerian-mcp-memory/pkg/types"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TemplateEngine handles template instantiation and task generation
type TemplateEngine struct {
	templates map[string]BuiltinTemplate
	funcMap   template.FuncMap
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine() *TemplateEngine {
	engine := &TemplateEngine{
		templates: make(map[string]BuiltinTemplate),
		funcMap:   createTemplateFuncMap(),
	}

	// Load builtin templates
	builtinTemplates := GetBuiltinTemplates()
	for i := range builtinTemplates {
		engine.templates[builtinTemplates[i].ID] = builtinTemplates[i]
	}

	return engine
}

// createTemplateFuncMap creates custom functions for template processing
func createTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"title":     cases.Title(language.English, cases.NoLower).String,
		"upper":     strings.ToUpper,
		"lower":     strings.ToLower,
		"replace":   strings.ReplaceAll,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"trim":      strings.TrimSpace,
		"split":     strings.Split,
		"join":      strings.Join,
		"now":       time.Now,
		"uuid":      func() string { return uuid.New().String() },
		"formatTime": func(format string, t time.Time) string {
			return t.Format(format)
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
	}
}

// TemplateInstantiationRequest represents a request to instantiate a template
type TemplateInstantiationRequest struct {
	TemplateID string                 `json:"template_id"`
	ProjectID  string                 `json:"project_id"`
	SessionID  string                 `json:"session_id,omitempty"`
	Variables  map[string]interface{} `json:"variables"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Prefix     string                 `json:"prefix,omitempty"` // Optional prefix for task names
}

// TemplateInstantiationResult represents the result of template instantiation
type TemplateInstantiationResult struct {
	TemplateID    string                 `json:"template_id"`
	TemplateName  string                 `json:"template_name"`
	ProjectID     string                 `json:"project_id"`
	SessionID     string                 `json:"session_id,omitempty"`
	Tasks         []GeneratedTask        `json:"tasks"`
	Variables     map[string]interface{} `json:"variables"`
	GeneratedAt   time.Time              `json:"generated_at"`
	EstimatedTime string                 `json:"estimated_time"`
	TaskCount     int                    `json:"task_count"`
	Warnings      []string               `json:"warnings,omitempty"`
	Errors        []string               `json:"errors,omitempty"`
}

// GeneratedTask represents a task generated from a template
type GeneratedTask struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Type          string                 `json:"type"`
	Priority      string                 `json:"priority"`
	EstimatedTime string                 `json:"estimated_time"`
	Dependencies  []string               `json:"dependencies"` // IDs of other generated tasks
	Tags          []string               `json:"tags"`
	Metadata      map[string]interface{} `json:"metadata"`
	ProjectID     string                 `json:"project_id"`
	SessionID     string                 `json:"session_id,omitempty"`
	TemplateID    string                 `json:"template_id"`
	CreatedAt     time.Time              `json:"created_at"`
	Status        string                 `json:"status"` // "pending", "in_progress", "completed"
}

// InstantiateTemplate creates tasks from a template with provided variables
func (te *TemplateEngine) InstantiateTemplate(req *TemplateInstantiationRequest) (*TemplateInstantiationResult, error) {
	// Validate request
	if err := te.validateInstantiationRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Get template
	tmpl, exists := te.templates[req.TemplateID]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", req.TemplateID)
	}

	// Validate variables
	if err := te.validateVariables(&tmpl, req.Variables); err != nil {
		return nil, fmt.Errorf("invalid variables: %w", err)
	}

	// Generate tasks
	tasks, warnings, generateErrors := te.generateTasks(&tmpl, req)

	// Calculate total estimated time
	estimatedTime := te.calculateTotalEstimatedTime(tasks)

	result := &TemplateInstantiationResult{
		TemplateID:    req.TemplateID,
		TemplateName:  tmpl.Name,
		ProjectID:     req.ProjectID,
		SessionID:     req.SessionID,
		Tasks:         tasks,
		Variables:     req.Variables,
		GeneratedAt:   time.Now(),
		EstimatedTime: estimatedTime,
		TaskCount:     len(tasks),
		Warnings:      warnings,
		Errors:        generateErrors,
	}

	return result, nil
}

// validateInstantiationRequest validates the instantiation request
func (te *TemplateEngine) validateInstantiationRequest(req *TemplateInstantiationRequest) error {
	if req == nil {
		return errors.New("request cannot be nil")
	}

	if req.TemplateID == "" {
		return errors.New("template_id is required")
	}

	if req.ProjectID == "" {
		return errors.New("project_id is required")
	}

	if req.Variables == nil {
		req.Variables = make(map[string]interface{})
	}

	return nil
}

// validateVariables validates that required variables are provided and have correct types
func (te *TemplateEngine) validateVariables(tmpl *BuiltinTemplate, variables map[string]interface{}) error {
	var validationErrors []string

	// Check required variables
	for _, variable := range tmpl.Variables {
		if variable.Required {
			value, exists := variables[variable.Name]
			if !exists {
				validationErrors = append(validationErrors, fmt.Sprintf("required variable '%s' is missing", variable.Name))
				continue
			}

			// Type validation
			if err := te.validateVariableType(&variable, value); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("variable '%s': %v", variable.Name, err))
			}
		}
	}

	// Set default values for missing optional variables
	for _, variable := range tmpl.Variables {
		if !variable.Required {
			if _, exists := variables[variable.Name]; !exists && variable.DefaultValue != nil {
				variables[variable.Name] = variable.DefaultValue
			}
		}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(validationErrors, "; "))
	}

	return nil
}

// validateVariableType validates the type of a variable value
func (te *TemplateEngine) validateVariableType(variable *TemplateVariable, value interface{}) error {
	switch variable.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string, got %T", value)
		}
		if variable.Validation != "" {
			if matched, _ := regexp.MatchString(variable.Validation, str); !matched {
				return fmt.Errorf("does not match pattern %s", variable.Validation)
			}
		}
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// Valid number types
		default:
			return fmt.Errorf("expected number, got %T", value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}
	case "choice":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("expected string for choice, got %T", value)
		}
		if len(variable.Options) > 0 {
			found := false
			for _, option := range variable.Options {
				if option == str {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("value '%s' not in allowed options: %v", str, variable.Options)
			}
		}
	}

	return nil
}

// generateTasks creates tasks from template with variable substitution
func (te *TemplateEngine) generateTasks(tmpl *BuiltinTemplate, req *TemplateInstantiationRequest) (tasks []GeneratedTask, warnings, taskErrors []string) {
	tasks = make([]GeneratedTask, 0, len(tmpl.Tasks))

	// Create task ID mapping for dependency resolution
	taskNameToID := make(map[string]string)

	// First pass: create tasks with basic info
	for i := range tmpl.Tasks {
		taskID := uuid.New().String()
		taskNameToID[tmpl.Tasks[i].Name] = taskID

		// Process template strings
		name, err := te.processTemplateString(tmpl.Tasks[i].Name, req.Variables)
		if err != nil {
			taskErrors = append(taskErrors, fmt.Sprintf("error processing task name '%s': %v", tmpl.Tasks[i].Name, err))
			continue
		}

		description, err := te.processTemplateString(tmpl.Tasks[i].Description, req.Variables)
		if err != nil {
			taskErrors = append(taskErrors, fmt.Sprintf("error processing task description '%s': %v", tmpl.Tasks[i].Description, err))
			continue
		}

		templateStr, err := te.processTemplateString(tmpl.Tasks[i].Template, req.Variables)
		if err != nil {
			taskErrors = append(taskErrors, fmt.Sprintf("error processing task template '%s': %v", tmpl.Tasks[i].Template, err))
			continue
		}

		priority, _ := te.processTemplateString(tmpl.Tasks[i].Priority, req.Variables)

		// Apply prefix if provided
		if req.Prefix != "" {
			name = req.Prefix + " " + name
		}

		// Process tags
		var processedTags []string
		for _, tag := range tmpl.Tasks[i].Tags {
			processedTag, err := te.processTemplateString(tag, req.Variables)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("error processing tag '%s': %v", tag, err))
				processedTags = append(processedTags, tag) // Use original tag
			} else {
				processedTags = append(processedTags, processedTag)
			}
		}

		// Create metadata
		metadata := make(map[string]interface{})
		for k, v := range tmpl.Tasks[i].Metadata {
			metadata[k] = v
		}
		metadata["template_id"] = req.TemplateID
		metadata["template_task_name"] = tmpl.Tasks[i].Name
		metadata["generated_from_template"] = true

		// Add request metadata
		for k, v := range req.Metadata {
			metadata[k] = v
		}

		task := GeneratedTask{
			ID:            taskID,
			Name:          name,
			Description:   description + "\n\n" + templateStr,
			Type:          tmpl.Tasks[i].Type,
			Priority:      priority,
			EstimatedTime: tmpl.Tasks[i].EstimatedTime,
			Dependencies:  []string{}, // Will be filled in second pass
			Tags:          processedTags,
			Metadata:      metadata,
			ProjectID:     req.ProjectID,
			SessionID:     req.SessionID,
			TemplateID:    req.TemplateID,
			CreatedAt:     time.Now(),
			Status:        "pending",
		}

		tasks = append(tasks, task)
	}

	// Second pass: resolve dependencies
	for i := range tmpl.Tasks {
		if i >= len(tasks) {
			break // Skip if task creation failed
		}

		var dependencyIDs []string
		for _, depName := range tmpl.Tasks[i].Dependencies {
			if depID, exists := taskNameToID[depName]; exists {
				dependencyIDs = append(dependencyIDs, depID)
			} else {
				warnings = append(warnings, fmt.Sprintf("dependency '%s' not found for task '%s'", depName, tmpl.Tasks[i].Name))
			}
		}
		tasks[i].Dependencies = dependencyIDs
	}

	return tasks, warnings, taskErrors
}

// processTemplateString processes a template string with variables
func (te *TemplateEngine) processTemplateString(templateStr string, variables map[string]interface{}) (string, error) {
	if templateStr == "" {
		return "", nil
	}

	tmpl, err := template.New("task_template").Funcs(te.funcMap).Parse(templateStr)
	if err != nil {
		return templateStr, fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return templateStr, fmt.Errorf("template execution error: %w", err)
	}

	return buf.String(), nil
}

// calculateTotalEstimatedTime calculates the total estimated time for all tasks
func (te *TemplateEngine) calculateTotalEstimatedTime(tasks []GeneratedTask) string {
	totalMinutes := 0

	for i := range tasks {
		minutes := te.parseEstimatedTime(tasks[i].EstimatedTime)
		totalMinutes += minutes
	}

	if totalMinutes == 0 {
		return "unknown"
	}

	// Convert to human readable format
	switch {
	case totalMinutes < 60:
		return fmt.Sprintf("%dm", totalMinutes)
	case totalMinutes < 60*24:
		hours := totalMinutes / 60
		remainingMinutes := totalMinutes % 60
		if remainingMinutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh %dm", hours, remainingMinutes)
	default:
		days := totalMinutes / (60 * 24)
		remainingHours := (totalMinutes % (60 * 24)) / 60
		if remainingHours == 0 {
			return fmt.Sprintf("%dd", days)
		}
		return fmt.Sprintf("%dd %dh", days, remainingHours)
	}
}

// parseEstimatedTime parses estimated time string to minutes
func (te *TemplateEngine) parseEstimatedTime(timeStr string) int {
	if timeStr == "" {
		return 0
	}

	timeStr = strings.ToLower(strings.TrimSpace(timeStr))

	// Parse patterns like "30m", "2h", "1d", "1h 30m"
	var totalMinutes int

	// Simple regex patterns
	patterns := map[string]int{
		`(\d+)d`: 60 * 24, // days
		`(\d+)h`: 60,      // hours
		`(\d+)m`: 1,       // minutes
	}

	for pattern, multiplier := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(timeStr, -1)
		for _, match := range matches {
			if len(match) > 1 {
				var value int
				_, _ = fmt.Sscanf(match[1], "%d", &value)
				totalMinutes += value * multiplier
			}
		}
	}

	return totalMinutes
}

// GetTemplate returns a template by ID
func (te *TemplateEngine) GetTemplate(templateID string) (*BuiltinTemplate, error) {
	tmpl, exists := te.templates[templateID]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}
	return &tmpl, nil
}

// ListTemplates returns all available templates
func (te *TemplateEngine) ListTemplates() []BuiltinTemplate {
	templates := make([]BuiltinTemplate, 0, len(te.templates))
	for id := range te.templates {
		templates = append(templates, te.templates[id])
	}
	return templates
}

// ListTemplatesByCategory returns templates filtered by category
func (te *TemplateEngine) ListTemplatesByCategory(category string) []BuiltinTemplate {
	var templates []BuiltinTemplate
	for id := range te.templates {
		tmpl := te.templates[id]
		if tmpl.Category == category {
			templates = append(templates, tmpl)
		}
	}
	return templates
}

// ListTemplatesByProjectType returns templates filtered by project type
func (te *TemplateEngine) ListTemplatesByProjectType(projectType types.ProjectType) []BuiltinTemplate {
	var templates []BuiltinTemplate
	for id := range te.templates {
		tmpl := te.templates[id]
		if tmpl.ProjectType == projectType || tmpl.ProjectType == types.ProjectTypeAny {
			templates = append(templates, tmpl)
		}
	}
	return templates
}

// ValidateTemplate validates a template structure
func (te *TemplateEngine) ValidateTemplate(tmpl *BuiltinTemplate) []string {
	var templateErrors []string

	templateErrors = append(templateErrors, te.validateBasicTemplateFields(tmpl)...)
	templateErrors = append(templateErrors, te.validateTasks(tmpl)...)
	templateErrors = append(templateErrors, te.validateTaskDependencies(tmpl)...)
	templateErrors = append(templateErrors, te.validateVariableDefinitions(tmpl)...)

	return templateErrors
}

// validateBasicTemplateFields validates basic template fields
func (te *TemplateEngine) validateBasicTemplateFields(tmpl *BuiltinTemplate) []string {
	var fieldErrors []string

	if tmpl.ID == "" {
		fieldErrors = append(fieldErrors, "template ID is required")
	}

	if tmpl.Name == "" {
		fieldErrors = append(fieldErrors, "template name is required")
	}

	if len(tmpl.Tasks) == 0 {
		fieldErrors = append(fieldErrors, "template must have at least one task")
	}

	return fieldErrors
}

// validateTasks validates task structure and uniqueness
func (te *TemplateEngine) validateTasks(tmpl *BuiltinTemplate) []string {
	var taskValidationErrors []string
	taskNames := make(map[string]bool)

	for i := range tmpl.Tasks {
		task := &tmpl.Tasks[i]
		if task.Name == "" {
			taskValidationErrors = append(taskValidationErrors, "task name is required")
			continue
		}

		if taskNames[task.Name] {
			taskValidationErrors = append(taskValidationErrors, "duplicate task name: "+task.Name)
		}

		taskNames[task.Name] = true
	}

	return taskValidationErrors
}

// validateTaskDependencies validates that task dependencies exist
func (te *TemplateEngine) validateTaskDependencies(tmpl *BuiltinTemplate) []string {
	var depErrors []string
	taskNames := te.getTaskNamesMap(tmpl)

	for i := range tmpl.Tasks {
		task := &tmpl.Tasks[i]
		for _, dep := range task.Dependencies {
			if !taskNames[dep] {
				depErrors = append(depErrors, fmt.Sprintf("task '%s' has invalid dependency: %s", task.Name, dep))
			}
		}
	}

	return depErrors
}

// getTaskNamesMap creates a map of task names for dependency validation
func (te *TemplateEngine) getTaskNamesMap(tmpl *BuiltinTemplate) map[string]bool {
	taskNames := make(map[string]bool)
	for i := range tmpl.Tasks {
		task := &tmpl.Tasks[i]
		if task.Name != "" {
			taskNames[task.Name] = true
		}
	}
	return taskNames
}

// validateVariables validates template variables
func (te *TemplateEngine) validateVariableDefinitions(tmpl *BuiltinTemplate) []string {
	var varDefErrors []string
	variableNames := make(map[string]bool)

	for _, variable := range tmpl.Variables {
		if variable.Name == "" {
			varDefErrors = append(varDefErrors, "variable name is required")
			continue
		}

		if variableNames[variable.Name] {
			varDefErrors = append(varDefErrors, "duplicate variable name: "+variable.Name)
		}

		variableNames[variable.Name] = true
		varDefErrors = append(varDefErrors, te.validateVariableTypeDefinition(&variable)...)
	}

	return varDefErrors
}

// validateVariableTypeDefinition validates a single variable's type definition
func (te *TemplateEngine) validateVariableTypeDefinition(variable *TemplateVariable) []string {
	var typeErrors []string
	validTypes := []string{"string", "number", "boolean", "choice"}

	validType := false
	for _, vt := range validTypes {
		if variable.Type == vt {
			validType = true
			break
		}
	}

	if !validType {
		typeErrors = append(typeErrors, fmt.Sprintf("variable '%s' has invalid type: %s", variable.Name, variable.Type))
	}

	return typeErrors
}

// AddCustomTemplate adds a custom template to the engine
func (te *TemplateEngine) AddCustomTemplate(tmpl *BuiltinTemplate) error {
	// Validate template
	if validationErrors := te.ValidateTemplate(tmpl); len(validationErrors) > 0 {
		return fmt.Errorf("template validation failed: %s", strings.Join(validationErrors, "; "))
	}

	te.templates[tmpl.ID] = *tmpl
	return nil
}

// RemoveTemplate removes a template from the engine
func (te *TemplateEngine) RemoveTemplate(templateID string) error {
	if _, exists := te.templates[templateID]; !exists {
		return fmt.Errorf("template not found: %s", templateID)
	}

	delete(te.templates, templateID)
	return nil
}
