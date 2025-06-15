package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/repositories"

	"github.com/google/uuid"
)

// TemplateService interface defines template management capabilities
type TemplateService interface {
	// Template CRUD operations
	ListTemplates(ctx context.Context, projectType entities.ProjectType) ([]*entities.TaskTemplate, error)
	GetTemplate(ctx context.Context, templateID string) (*entities.TaskTemplate, error)
	CreateTemplate(ctx context.Context, template *entities.TaskTemplate) error
	UpdateTemplate(ctx context.Context, template *entities.TaskTemplate) error
	DeleteTemplate(ctx context.Context, templateID string) error

	// Template matching and instantiation
	InstantiateTemplate(ctx context.Context, templateID string, repository string, vars map[string]interface{}) ([]*entities.Task, error)
	MatchTemplates(ctx context.Context, projectPath string) ([]*entities.TemplateMatch, error)
	SuggestVariables(ctx context.Context, templateID string, projectPath string) (map[string]interface{}, error)

	// Built-in templates
	GetBuiltInTemplates() []*entities.TaskTemplate
	LoadBuiltInTemplates() error

	// Template usage and analytics
	UpdateTemplateUsage(ctx context.Context, templateID string, success bool) error
	GetTemplateAnalytics(ctx context.Context, templateID string) (*TemplateAnalytics, error)
	GetPopularTemplates(ctx context.Context, projectType entities.ProjectType, limit int) ([]*entities.TaskTemplate, error)

	// Template validation
	ValidateTemplate(template *entities.TaskTemplate) []string
	ValidateVariables(template *entities.TaskTemplate, vars map[string]interface{}) error
}

// TemplateAnalytics represents analytics data for templates
type TemplateAnalytics struct {
	TemplateID      string                 `json:"template_id"`
	UsageCount      int                    `json:"usage_count"`
	SuccessRate     float64                `json:"success_rate"`
	AverageTime     time.Duration          `json:"average_time"`
	ProjectTypes    map[string]int         `json:"project_types"`
	CommonVariables map[string]interface{} `json:"common_variables"`
	UserRatings     float64                `json:"user_ratings"`
	LastUsed        time.Time              `json:"last_used"`
	TrendDirection  string                 `json:"trend_direction"` // "up", "down", "stable"
}

// TemplateServiceConfig holds configuration for template service
type TemplateServiceConfig struct {
	BuiltInTemplatesPath  string        `json:"built_in_templates_path"`
	MaxTemplatesPerUser   int           `json:"max_templates_per_user"`
	EnableVersioning      bool          `json:"enable_versioning"`
	EnableSharing         bool          `json:"enable_sharing"`
	ValidationStrict      bool          `json:"validation_strict"`
	CacheTemplates        bool          `json:"cache_templates"`
	CacheTTL              time.Duration `json:"cache_ttl"`
	MinSuccessRateDisplay float64       `json:"min_success_rate_display"`
}

// DefaultTemplateServiceConfig returns default configuration
func DefaultTemplateServiceConfig() *TemplateServiceConfig {
	return &TemplateServiceConfig{
		BuiltInTemplatesPath:  "internal/templates/built-in",
		MaxTemplatesPerUser:   50,
		EnableVersioning:      true,
		EnableSharing:         true,
		ValidationStrict:      true,
		CacheTemplates:        true,
		CacheTTL:              30 * time.Minute,
		MinSuccessRateDisplay: 0.3,
	}
}

// templateServiceImpl implements the TemplateService interface
type templateServiceImpl struct {
	templateRepo      repositories.TemplateRepository
	taskRepo          repositories.TaskRepository
	projectClassifier ProjectClassifier
	builtInTemplates  map[string]*entities.TaskTemplate
	templateCache     map[string]*entities.TaskTemplate
	config            *TemplateServiceConfig
	logger            *slog.Logger
}

// NewTemplateService creates a new template service
func NewTemplateService(
	templateRepo repositories.TemplateRepository,
	taskRepo repositories.TaskRepository,
	projectClassifier ProjectClassifier,
	config *TemplateServiceConfig,
	logger *slog.Logger,
) TemplateService {
	if config == nil {
		config = DefaultTemplateServiceConfig()
	}

	service := &templateServiceImpl{
		templateRepo:      templateRepo,
		taskRepo:          taskRepo,
		projectClassifier: projectClassifier,
		builtInTemplates:  make(map[string]*entities.TaskTemplate),
		templateCache:     make(map[string]*entities.TaskTemplate),
		config:            config,
		logger:            logger,
	}

	// Load built-in templates
	if err := service.LoadBuiltInTemplates(); err != nil {
		logger.Error("failed to load built-in templates", slog.Any("error", err))
	}

	return service
}

// ListTemplates returns templates filtered by project type
func (ts *templateServiceImpl) ListTemplates(
	ctx context.Context,
	projectType entities.ProjectType,
) ([]*entities.TaskTemplate, error) {
	ts.logger.Debug("listing templates", slog.String("project_type", string(projectType)))

	// Get templates from repository
	templates, err := ts.templateRepo.FindByProjectType(ctx, projectType)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %w", err)
	}

	// Add built-in templates
	for _, builtIn := range ts.builtInTemplates {
		if projectType == "" || builtIn.ProjectType == projectType {
			templates = append(templates, builtIn)
		}
	}

	// Filter by success rate if configured
	if ts.config.MinSuccessRateDisplay > 0 {
		filtered := make([]*entities.TaskTemplate, 0)
		for _, template := range templates {
			if template.SuccessRate >= ts.config.MinSuccessRateDisplay || template.UsageCount < 5 {
				filtered = append(filtered, template)
			}
		}
		templates = filtered
	}

	// Sort by usage and success rate
	sort.Slice(templates, func(i, j int) bool {
		if templates[i].SuccessRate != templates[j].SuccessRate {
			return templates[i].SuccessRate > templates[j].SuccessRate
		}
		return templates[i].UsageCount > templates[j].UsageCount
	})

	return templates, nil
}

// GetTemplate returns a specific template by ID
func (ts *templateServiceImpl) GetTemplate(ctx context.Context, templateID string) (*entities.TaskTemplate, error) {
	// Check cache first
	if ts.config.CacheTemplates {
		if cached, exists := ts.templateCache[templateID]; exists {
			return cached, nil
		}
	}

	// Check built-in templates
	if builtIn, exists := ts.builtInTemplates[templateID]; exists {
		if ts.config.CacheTemplates {
			ts.templateCache[templateID] = builtIn
		}
		return builtIn, nil
	}

	// Get from repository
	template, err := ts.templateRepo.FindByID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// Cache if enabled
	if ts.config.CacheTemplates {
		ts.templateCache[templateID] = template
	}

	return template, nil
}

// CreateTemplate creates a new template
func (ts *templateServiceImpl) CreateTemplate(ctx context.Context, template *entities.TaskTemplate) error {
	ts.logger.Info("creating template", slog.String("name", template.Name))

	// Validate template
	if errors := ts.ValidateTemplate(template); len(errors) > 0 {
		return fmt.Errorf("template validation failed: %v", errors)
	}

	// Set metadata
	template.ID = uuid.New().String()
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	template.UsageCount = 0
	template.SuccessRate = 0.0

	// Save to repository
	if err := ts.templateRepo.Create(ctx, template); err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	// Update cache
	if ts.config.CacheTemplates {
		ts.templateCache[template.ID] = template
	}

	return nil
}

// UpdateTemplate updates an existing template
func (ts *templateServiceImpl) UpdateTemplate(ctx context.Context, template *entities.TaskTemplate) error {
	ts.logger.Info("updating template", slog.String("id", template.ID))

	// Validate template
	if errors := ts.ValidateTemplate(template); len(errors) > 0 {
		return fmt.Errorf("template validation failed: %v", errors)
	}

	// Check if template exists
	existing, err := ts.GetTemplate(ctx, template.ID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Don't allow updating built-in templates
	if existing.IsBuiltIn {
		return errors.New("cannot update built-in template")
	}

	// Update metadata
	template.UpdatedAt = time.Now()

	// Save to repository
	if err := ts.templateRepo.Update(ctx, template); err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	// Update cache
	if ts.config.CacheTemplates {
		ts.templateCache[template.ID] = template
	}

	return nil
}

// DeleteTemplate deletes a template
func (ts *templateServiceImpl) DeleteTemplate(ctx context.Context, templateID string) error {
	ts.logger.Info("deleting template", slog.String("id", templateID))

	// Check if template exists
	template, err := ts.GetTemplate(ctx, templateID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Don't allow deleting built-in templates
	if template.IsBuiltIn {
		return errors.New("cannot delete built-in template")
	}

	// Delete from repository
	if err := ts.templateRepo.Delete(ctx, templateID); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	// Remove from cache
	delete(ts.templateCache, templateID)

	return nil
}

// InstantiateTemplate creates tasks from a template
func (ts *templateServiceImpl) InstantiateTemplate(
	ctx context.Context,
	templateID string,
	repository string,
	vars map[string]interface{},
) ([]*entities.Task, error) {
	ts.logger.Info("instantiating template",
		slog.String("template_id", templateID),
		slog.String("repository", repository))

	// Get template
	template, err := ts.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Validate variables
	if err := ts.ValidateVariables(template, vars); err != nil {
		return nil, fmt.Errorf("variable validation failed: %w", err)
	}

	// Apply defaults for missing variables
	vars = ts.applyDefaults(template, vars)

	// Add repository to variables
	vars["repository"] = repository

	// Create tasks from template
	tasks := make([]*entities.Task, 0, len(template.Tasks))
	taskMap := make(map[int]*entities.Task) // For dependency resolution

	for _, tmplTask := range template.Tasks {
		// Substitute variables in content
		content, err := ts.substituteVariables(tmplTask.Content, vars)
		if err != nil {
			return nil, fmt.Errorf("failed to substitute variables: %w", err)
		}

		// Substitute variables in description if present
		description := tmplTask.Description
		if description != "" {
			description, err = ts.substituteVariables(description, vars)
			if err != nil {
				return nil, fmt.Errorf("failed to substitute variables in description: %w", err)
			}
		}

		task := &entities.Task{
			ID:          uuid.New().String(),
			Content:     content,
			Description: description,
			Status:      entities.StatusPending,
			Priority:    entities.Priority(tmplTask.Priority),
			Repository:  repository,
			Tags:        tmplTask.Tags,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		tasks = append(tasks, task)
		taskMap[tmplTask.Order] = task
	}

	// Set dependencies
	for _, tmplTask := range template.Tasks {
		if len(tmplTask.Dependencies) > 0 {
			var dependsOn []string
			for _, depOrder := range tmplTask.Dependencies {
				if depTask, exists := taskMap[depOrder]; exists {
					dependsOn = append(dependsOn, depTask.ID)
				}
			}
			// Store dependencies in current task metadata
			if currentTask, exists := taskMap[tmplTask.Order]; exists {
				if currentTask.Metadata == nil {
					currentTask.Metadata = make(map[string]interface{})
				}
				currentTask.Metadata["depends_on"] = dependsOn
			}
		}
	}

	// Store tasks in repository
	for _, task := range tasks {
		if err := ts.taskRepo.Create(ctx, task); err != nil {
			return nil, fmt.Errorf("failed to create task: %w", err)
		}
	}

	// Update template usage with inherited context
	go func() {
		updateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := ts.UpdateTemplateUsage(updateCtx, templateID, true); err != nil {
			ts.logger.Error("failed to update template usage", slog.Any("error", err))
		}
	}()

	ts.logger.Info("template instantiated successfully",
		slog.Int("tasks_created", len(tasks)))

	return tasks, nil
}

// MatchTemplates finds templates that match a project
func (ts *templateServiceImpl) MatchTemplates(
	ctx context.Context,
	projectPath string,
) ([]*entities.TemplateMatch, error) {
	ts.logger.Debug("matching templates for project", slog.String("path", projectPath))

	// Classify project
	projectType, confidence, err := ts.projectClassifier.ClassifyProject(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to classify project: %w", err)
	}

	// Get project characteristics
	characteristics, err := ts.projectClassifier.GetProjectCharacteristics(ctx, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get project characteristics: %w", err)
	}

	// Get templates for project type
	templates, err := ts.ListTemplates(ctx, projectType)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates: %w", err)
	}

	// Score templates based on project characteristics
	var matches []*entities.TemplateMatch

	for _, template := range templates {
		score := ts.calculateTemplateMatch(template, projectType, confidence, characteristics)
		if score > 0.3 { // Minimum threshold
			reason := ts.generateMatchReason(template, projectType, characteristics)
			variables := ts.suggestVariables(template, characteristics)

			match := &entities.TemplateMatch{
				Template:  template,
				Score:     score,
				Reason:    reason,
				Variables: variables,
			}
			matches = append(matches, match)
		}
	}

	// Sort by score
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Limit to top 10 matches
	if len(matches) > 10 {
		matches = matches[:10]
	}

	return matches, nil
}

// SuggestVariables suggests variable values based on project analysis
func (ts *templateServiceImpl) SuggestVariables(
	ctx context.Context,
	templateID string,
	projectPath string,
) (map[string]interface{}, error) {
	template, err := ts.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}

	characteristics, err := ts.projectClassifier.GetProjectCharacteristics(ctx, projectPath)
	if err != nil {
		return nil, err
	}

	return ts.suggestVariables(template, characteristics), nil
}

// GetBuiltInTemplates returns all built-in templates
func (ts *templateServiceImpl) GetBuiltInTemplates() []*entities.TaskTemplate {
	templates := make([]*entities.TaskTemplate, 0, len(ts.builtInTemplates))
	for _, template := range ts.builtInTemplates {
		templates = append(templates, template)
	}
	return templates
}

// LoadBuiltInTemplates loads built-in templates
func (ts *templateServiceImpl) LoadBuiltInTemplates() error {
	ts.logger.Info("loading built-in templates")

	// Clear existing built-in templates
	ts.builtInTemplates = make(map[string]*entities.TaskTemplate)

	// Load templates (in a real implementation, these would be loaded from files)
	templates := ts.createBuiltInTemplates()

	for _, template := range templates {
		ts.builtInTemplates[template.ID] = template
	}

	ts.logger.Info("built-in templates loaded", slog.Int("count", len(templates)))
	return nil
}

// UpdateTemplateUsage updates template usage statistics
func (ts *templateServiceImpl) UpdateTemplateUsage(ctx context.Context, templateID string, success bool) error {
	template, err := ts.GetTemplate(ctx, templateID)
	if err != nil {
		return err
	}

	// Don't update built-in template stats in repository
	if template.IsBuiltIn {
		template.UpdateUsageStats(success)
		return nil
	}

	// Update template stats
	template.UpdateUsageStats(success)

	// Save to repository
	return ts.templateRepo.Update(ctx, template)
}

// GetTemplateAnalytics returns analytics for a template
func (ts *templateServiceImpl) GetTemplateAnalytics(
	ctx context.Context,
	templateID string,
) (*TemplateAnalytics, error) {
	template, err := ts.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// In a real implementation, this would gather analytics from usage data
	analytics := &TemplateAnalytics{
		TemplateID:     template.ID,
		UsageCount:     template.UsageCount,
		SuccessRate:    template.SuccessRate,
		AverageTime:    time.Duration(template.GetEstimatedTotalHours()) * time.Hour,
		ProjectTypes:   map[string]int{string(template.ProjectType): template.UsageCount},
		UserRatings:    template.SuccessRate * 5, // Convert to 5-star scale
		LastUsed:       time.Now(),
		TrendDirection: "stable",
	}

	if template.LastUsed != nil {
		analytics.LastUsed = *template.LastUsed
	}

	return analytics, nil
}

// GetPopularTemplates returns popular templates for a project type
func (ts *templateServiceImpl) GetPopularTemplates(
	ctx context.Context,
	projectType entities.ProjectType,
	limit int,
) ([]*entities.TaskTemplate, error) {
	templates, err := ts.ListTemplates(ctx, projectType)
	if err != nil {
		return nil, err
	}

	// Sort by popularity (usage count * success rate)
	sort.Slice(templates, func(i, j int) bool {
		scoreI := float64(templates[i].UsageCount) * templates[i].SuccessRate
		scoreJ := float64(templates[j].UsageCount) * templates[j].SuccessRate
		return scoreI > scoreJ
	})

	if len(templates) > limit {
		templates = templates[:limit]
	}

	return templates, nil
}

// ValidateTemplate validates a template structure
func (ts *templateServiceImpl) ValidateTemplate(template *entities.TaskTemplate) []string {
	var errors []string

	// Basic validation
	if template.Name == "" {
		errors = append(errors, "template name is required")
	}

	if len(template.Tasks) == 0 {
		errors = append(errors, "template must have at least one task")
	}

	// Use the template's built-in validation
	templateErrors := template.ValidateTemplate()
	errors = append(errors, templateErrors...)

	// Additional business logic validation
	if ts.config.ValidationStrict {
		if template.GetEstimatedTotalHours() > 100 {
			errors = append(errors, "total estimated hours cannot exceed 100")
		}

		if len(template.Variables) > 20 {
			errors = append(errors, "template cannot have more than 20 variables")
		}
	}

	return errors
}

// ValidateVariables validates provided variables against template requirements
func (ts *templateServiceImpl) ValidateVariables(
	template *entities.TaskTemplate,
	vars map[string]interface{},
) error {
	for _, variable := range template.Variables {
		value, exists := vars[variable.Name]

		// Check required variables
		if variable.Required && (!exists || value == nil) {
			if variable.Default == nil {
				return fmt.Errorf("required variable '%s' is missing", variable.Name)
			}
		}

		// Validate variable type and format
		if exists && value != nil {
			if err := ts.validateVariableValue(variable, value); err != nil {
				return fmt.Errorf("variable '%s': %w", variable.Name, err)
			}
		}
	}

	return nil
}

// Helper methods

func (ts *templateServiceImpl) applyDefaults(
	template *entities.TaskTemplate,
	vars map[string]interface{},
) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy existing variables
	for k, v := range vars {
		result[k] = v
	}

	// Apply defaults for missing variables
	for _, variable := range template.Variables {
		if _, exists := result[variable.Name]; !exists && variable.Default != nil {
			result[variable.Name] = variable.Default
		}
	}

	return result
}

func (ts *templateServiceImpl) substituteVariables(
	content string,
	vars map[string]interface{},
) (string, error) {
	tmpl, err := template.New("task").Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, vars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return result.String(), nil
}

func (ts *templateServiceImpl) calculateTemplateMatch(
	template *entities.TaskTemplate,
	projectType entities.ProjectType,
	confidence float64,
	characteristics *entities.ProjectCharacteristics,
) float64 {
	score := 0.0

	// Project type match (40%)
	switch template.ProjectType {
	case projectType:
		score += 0.4 * confidence
	case entities.ProjectTypeUnknown:
		score += 0.2 // Generic templates have lower score
	}

	// Success rate (30%)
	score += 0.3 * template.SuccessRate

	// Usage count (normalized) (20%)
	if template.UsageCount > 0 {
		// Normalize usage count (assuming max 100 uses)
		usageScore := float64(template.UsageCount) / 100.0
		if usageScore > 1.0 {
			usageScore = 1.0
		}
		score += 0.2 * usageScore
	}

	// Complexity match (10%)
	templateComplexity := ts.estimateTemplateComplexity(template)
	projectComplexity := characteristics.GetComplexityScore()
	complexityDiff := 1.0 - (templateComplexity - projectComplexity)
	if complexityDiff < 0 {
		complexityDiff = -complexityDiff
	}
	score += 0.1 * complexityDiff

	return score
}

func (ts *templateServiceImpl) generateMatchReason(
	template *entities.TaskTemplate,
	projectType entities.ProjectType,
	characteristics *entities.ProjectCharacteristics,
) string {
	reasons := []string{}

	if template.ProjectType == projectType {
		reasons = append(reasons, fmt.Sprintf("Perfect match for %s projects", projectType))
	}

	if template.SuccessRate > 0.8 {
		reasons = append(reasons, fmt.Sprintf("High success rate (%.0f%%)", template.SuccessRate*100))
	}

	if template.UsageCount > 10 {
		reasons = append(reasons, fmt.Sprintf("Popular template (%d uses)", template.UsageCount))
	}

	// Check for specific technology matches
	primaryLang := characteristics.GetPrimaryLanguage()
	if strings.Contains(strings.ToLower(template.Name), primaryLang) {
		reasons = append(reasons, fmt.Sprintf("Matches your %s project", primaryLang))
	}

	if len(reasons) == 0 {
		return "General template suitable for your project type"
	}

	return strings.Join(reasons, ". ")
}

func (ts *templateServiceImpl) suggestVariables(
	template *entities.TaskTemplate,
	characteristics *entities.ProjectCharacteristics,
) map[string]interface{} {
	suggestions := make(map[string]interface{})

	for _, variable := range template.Variables {
		switch strings.ToLower(variable.Name) {
		case "framework":
			if len(characteristics.Frameworks) > 0 {
				suggestions[variable.Name] = characteristics.Frameworks[0]
			}
		case "language":
			suggestions[variable.Name] = characteristics.GetPrimaryLanguage()
		case "database":
			if characteristics.HasDatabase {
				suggestions[variable.Name] = ts.suggestDatabase(characteristics)
			}
		case "testing_framework":
			suggestions[variable.Name] = ts.suggestTestingFramework(characteristics)
		case "has_docker":
			suggestions[variable.Name] = characteristics.HasDocker
		case "has_ci":
			suggestions[variable.Name] = characteristics.HasCI
		}
	}

	return suggestions
}

func (ts *templateServiceImpl) suggestDatabase(characteristics *entities.ProjectCharacteristics) string {
	// Simple heuristic for database suggestion
	primaryLang := characteristics.GetPrimaryLanguage()
	switch primaryLang {
	case constants.LanguageJavaScript, "typescript":
		return "mongodb"
	case constants.LanguagePython:
		return constants.LanguagePostgreSQL
	case "java", constants.LanguageCSharp:
		return constants.LanguagePostgreSQL
	case "go":
		return constants.LanguagePostgreSQL
	default:
		return "sqlite"
	}
}

func (ts *templateServiceImpl) suggestTestingFramework(characteristics *entities.ProjectCharacteristics) string {
	primaryLang := characteristics.GetPrimaryLanguage()
	switch primaryLang {
	case "javascript", "typescript":
		return "jest"
	case "python":
		return "pytest"
	case "java":
		return "junit"
	case "go":
		return constants.TaskTypeTesting
	case "csharp":
		return "xunit"
	default:
		return "standard"
	}
}

func (ts *templateServiceImpl) estimateTemplateComplexity(template *entities.TaskTemplate) float64 {
	complexity := 0.0

	// Number of tasks
	complexity += float64(len(template.Tasks)) * 0.1

	// Number of variables
	complexity += float64(len(template.Variables)) * 0.05

	// Total estimated hours
	complexity += template.GetEstimatedTotalHours() * 0.01

	// Dependencies complexity
	for _, task := range template.Tasks {
		complexity += float64(len(task.Dependencies)) * 0.02
	}

	// Cap at 1.0
	if complexity > 1.0 {
		complexity = 1.0
	}

	return complexity
}

func (ts *templateServiceImpl) validateVariableValue(variable entities.TemplateVariable, value interface{}) error {
	switch variable.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return errors.New("expected string value")
		}
		if variable.ValidationRegex != "" {
			strValue := value.(string)
			matched, err := regexp.MatchString(variable.ValidationRegex, strValue)
			if err != nil {
				return fmt.Errorf("regex validation error: %w", err)
			}
			if !matched {
				return errors.New("value does not match required pattern")
			}
		}

	case "number":
		switch value.(type) {
		case int, int64, float64:
			// Valid number types
		default:
			return errors.New("expected number value")
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return errors.New("expected boolean value")
		}

	case "choice":
		strValue, ok := value.(string)
		if !ok {
			return errors.New("expected string value for choice")
		}

		validChoice := false
		for _, option := range variable.Options {
			if strValue == option {
				validChoice = true
				break
			}
		}
		if !validChoice {
			return fmt.Errorf("invalid choice, must be one of: %v", variable.Options)
		}
	}

	return nil
}

// createBuiltInTemplates creates the built-in templates
func (ts *templateServiceImpl) createBuiltInTemplates() []*entities.TaskTemplate {
	var templates []*entities.TaskTemplate

	// Web App Template
	webAppTemplate := &entities.TaskTemplate{
		ID:          "builtin-web-app-init",
		Name:        "Web Application Setup",
		Description: "Complete setup for a modern web application with frontend and backend",
		ProjectType: entities.ProjectTypeWebApp,
		Category:    "initialization",
		Version:     "1.0.0",
		Author:      "system",
		IsBuiltIn:   true,
		IsPublic:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Tasks: []entities.TemplateTask{
			{
				Order:          1,
				Content:        "Setup project structure and initial configuration",
				Description:    "Create project directories, initialize version control, and setup basic configuration files",
				Priority:       "high",
				Type:           "setup",
				EstimatedHours: 2,
				Tags:           []string{"setup", "config", "initialization"},
			},
			{
				Order:          2,
				Content:        "Configure {{.Framework}} framework with TypeScript",
				Description:    "Install and configure the chosen frontend framework with TypeScript support",
				Priority:       "high",
				Type:           "implementation",
				EstimatedHours: 3,
				Dependencies:   []int{1},
				Tags:           []string{"framework", "typescript", "frontend"},
			},
			{
				Order:          3,
				Content:        "Setup {{.Database}} database and ORM",
				Description:    "Configure database connection and setup ORM/ODM for data persistence",
				Priority:       "high",
				Type:           "implementation",
				EstimatedHours: 2,
				Dependencies:   []int{1},
				Tags:           []string{"database", "backend", "persistence"},
			},
			{
				Order:          4,
				Content:        "Implement authentication system",
				Description:    "Setup user authentication and authorization with secure session management",
				Priority:       "high",
				Type:           "feature",
				EstimatedHours: 4,
				Dependencies:   []int{2, 3},
				Tags:           []string{"auth", "security", "users"},
			},
			{
				Order:          5,
				Content:        "Create initial UI components and layouts",
				Description:    "Build reusable UI components and main application layouts",
				Priority:       "medium",
				Type:           "implementation",
				EstimatedHours: 3,
				Dependencies:   []int{2},
				Tags:           []string{"ui", "frontend", "components"},
			},
			{
				Order:          6,
				Content:        "Setup testing framework and write initial tests",
				Description:    "Configure testing environment and write unit/integration tests",
				Priority:       "medium",
				Type:           "testing",
				EstimatedHours: 2,
				Dependencies:   []int{2, 3},
				Tags:           []string{"testing", "quality", "automation"},
			},
		},
		Variables: []entities.TemplateVariable{
			{
				Name:        "Framework",
				Description: "Frontend framework to use",
				Type:        "choice",
				Default:     "react",
				Required:    true,
				Options:     []string{"react", "vue", "angular", "svelte"},
			},
			{
				Name:        "Database",
				Description: "Database system to use",
				Type:        "choice",
				Default:     "postgresql",
				Required:    true,
				Options:     []string{"postgresql", "mysql", "mongodb", "sqlite"},
			},
		},
		Tags: []string{"web", "fullstack", "modern"},
	}

	// CLI Tool Template
	cliTemplate := &entities.TaskTemplate{
		ID:          "builtin-cli-tool",
		Name:        "CLI Tool Development",
		Description: "Setup for a command-line interface tool with proper argument parsing and help system",
		ProjectType: entities.ProjectTypeCLI,
		Category:    "initialization",
		Version:     "1.0.0",
		Author:      "system",
		IsBuiltIn:   true,
		IsPublic:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Tasks: []entities.TemplateTask{
			{
				Order:          1,
				Content:        "Initialize CLI project structure",
				Description:    "Create project directories and initialize with CLI framework",
				Priority:       "high",
				Type:           "setup",
				EstimatedHours: 1,
				Tags:           []string{"setup", "cli", "initialization"},
			},
			{
				Order:          2,
				Content:        "Setup command parsing with {{.CLIFramework}}",
				Description:    "Implement command structure and argument parsing",
				Priority:       "high",
				Type:           "implementation",
				EstimatedHours: 2,
				Dependencies:   []int{1},
				Tags:           []string{"commands", "parsing", "interface"},
			},
			{
				Order:          3,
				Content:        "Implement core functionality",
				Description:    "Build the main business logic of the CLI tool",
				Priority:       "high",
				Type:           "implementation",
				EstimatedHours: 4,
				Dependencies:   []int{2},
				Tags:           []string{"core", "logic", "features"},
			},
			{
				Order:          4,
				Content:        "Add configuration file support",
				Description:    "Implement configuration file parsing and management",
				Priority:       "medium",
				Type:           "feature",
				EstimatedHours: 1.5,
				Dependencies:   []int{2},
				Tags:           []string{"config", "files", "settings"},
			},
			{
				Order:          5,
				Content:        "Implement comprehensive help system",
				Description:    "Create detailed help text and usage examples",
				Priority:       "medium",
				Type:           "documentation",
				EstimatedHours: 1,
				Dependencies:   []int{2},
				Tags:           []string{"help", "documentation", "usability"},
			},
		},
		Variables: []entities.TemplateVariable{
			{
				Name:        "CLIFramework",
				Description: "CLI framework to use",
				Type:        "choice",
				Default:     "cobra",
				Required:    true,
				Options:     []string{"cobra", "cli", "flag", "kingpin"},
			},
		},
		Tags: []string{"cli", "tool", "command-line"},
	}

	// API Service Template
	apiTemplate := &entities.TaskTemplate{
		ID:          "builtin-api-service",
		Name:        "REST API Service",
		Description: "RESTful API service with proper routing, middleware, and database integration",
		ProjectType: entities.ProjectTypeAPI,
		Category:    "initialization",
		Version:     "1.0.0",
		Author:      "system",
		IsBuiltIn:   true,
		IsPublic:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Tasks: []entities.TemplateTask{
			{
				Order:          1,
				Content:        "Setup API project structure",
				Description:    "Initialize project with proper directory structure for API development",
				Priority:       "high",
				Type:           "setup",
				EstimatedHours: 1,
				Tags:           []string{"setup", "api", "structure"},
			},
			{
				Order:          2,
				Content:        "Configure {{.APIFramework}} router and middleware",
				Description:    "Setup HTTP router, logging, CORS, and other essential middleware",
				Priority:       "high",
				Type:           "implementation",
				EstimatedHours: 2,
				Dependencies:   []int{1},
				Tags:           []string{"router", "middleware", "http"},
			},
			{
				Order:          3,
				Content:        "Setup database models and migrations",
				Description:    "Create database schema, models, and migration system",
				Priority:       "high",
				Type:           "implementation",
				EstimatedHours: 3,
				Dependencies:   []int{1},
				Tags:           []string{"database", "models", "schema"},
			},
			{
				Order:          4,
				Content:        "Implement CRUD endpoints",
				Description:    "Create REST endpoints for Create, Read, Update, Delete operations",
				Priority:       "high",
				Type:           "implementation",
				EstimatedHours: 4,
				Dependencies:   []int{2, 3},
				Tags:           []string{"crud", "endpoints", "rest"},
			},
			{
				Order:          5,
				Content:        "Add authentication middleware",
				Description:    "Implement JWT or session-based authentication",
				Priority:       "high",
				Type:           "security",
				EstimatedHours: 2,
				Dependencies:   []int{2},
				Tags:           []string{"auth", "security", "jwt"},
			},
			{
				Order:          6,
				Content:        "Setup API documentation",
				Description:    "Generate OpenAPI/Swagger documentation for the API",
				Priority:       "medium",
				Type:           "documentation",
				EstimatedHours: 1.5,
				Dependencies:   []int{4},
				Tags:           []string{"docs", "swagger", "openapi"},
			},
		},
		Variables: []entities.TemplateVariable{
			{
				Name:        "APIFramework",
				Description: "API framework to use",
				Type:        "choice",
				Default:     "gin",
				Required:    true,
				Options:     []string{"gin", "echo", "chi", "gorilla/mux"},
			},
		},
		Tags: []string{"api", "rest", "service", "backend"},
	}

	templates = append(templates, webAppTemplate, cliTemplate, apiTemplate)
	return templates
}
