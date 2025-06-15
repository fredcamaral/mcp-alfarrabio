// Package services provides domain services for the lerian-mcp-memory CLI.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// TaskGeneratorService provides AI-powered task generation capabilities
type TaskGeneratorService interface {
	// Core generation methods
	GenerateFromAnalysis(ctx context.Context, analysis *AIAnalysis, context GenerationContext) ([]*entities.Task, error)
	GenerateMainTasks(ctx context.Context, trd *TRDEntity, rule *GenerationRule) ([]*MainTask, error)
	GenerateSubTasks(ctx context.Context, mainTask *MainTask, rule *GenerationRule) ([]*SubTask, error)

	// Analysis and validation
	AnalyzeComplexity(content string) (*ComplexityAnalysis, error)
	ValidateGeneration(tasks interface{}) error
	EstimateTaskEffort(content string) (int, error)

	// Template and pattern matching
	FindTaskTemplates(feature, projectType string) []*TaskTemplate
	ApplyTemplate(template *TaskTemplate, context GenerationContext) (*entities.Task, error)

	// Dependency analysis
	DetectDependencies(tasks []*entities.Task) error
	ValidateTaskHierarchy(mainTasks []*MainTask, subTasks []*SubTask) error
}

// AIAnalysis represents AI analysis results for task generation
type AIAnalysis struct {
	ID             string                 `json:"id"`
	PRDID          string                 `json:"prd_id"`
	KeyFeatures    []string               `json:"key_features"`
	TechnicalReqs  []string               `json:"technical_requirements"`
	BusinessRules  []string               `json:"business_rules"`
	Dependencies   []string               `json:"dependencies"`
	RiskFactors    []string               `json:"risk_factors"`
	Complexity     string                 `json:"complexity"` // low, medium, high
	EstimatedHours int                    `json:"estimated_hours"`
	Confidence     float64                `json:"confidence"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
}

// GenerationRule represents rules for task generation
type GenerationRule struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // main_tasks, sub_tasks
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Template    string                 `json:"template"`
	Constraints map[string]interface{} `json:"constraints"`
	Parameters  map[string]interface{} `json:"parameters"`
	Active      bool                   `json:"active"`
}

// TaskGenerationResult represents the result of task generation
type TaskGenerationResult struct {
	Tasks       []*entities.Task `json:"tasks"`
	MainTasks   []*MainTask      `json:"main_tasks,omitempty"`
	SubTasks    []*SubTask       `json:"sub_tasks,omitempty"`
	GeneratedBy string           `json:"generated_by"`
	GeneratedAt time.Time        `json:"generated_at"`
	TotalHours  int              `json:"total_hours"`
	Complexity  string           `json:"complexity"`
	Confidence  float64          `json:"confidence"`
	Warnings    []string         `json:"warnings,omitempty"`
	Suggestions []string         `json:"suggestions,omitempty"`
}

// DefaultTaskGeneratorService implements TaskGeneratorService
type DefaultTaskGeneratorService struct {
	mcpClient          ports.MCPClient
	complexityAnalyzer ComplexityAnalyzer
	templateMatcher    TemplateMatcher
	logger             *slog.Logger
	config             *TaskGeneratorConfig
}

// TaskGeneratorConfig configures task generation behavior
type TaskGeneratorConfig struct {
	MaxTasksPerPRD            int     `mapstructure:"max_tasks_per_prd"`
	ComplexityThreshold       float64 `mapstructure:"complexity_threshold"`
	MinTaskHours              int     `mapstructure:"min_task_hours"`
	MaxTaskHours              int     `mapstructure:"max_task_hours"`
	AutoSubTasking            bool    `mapstructure:"auto_subtasking"`
	SubTaskMaxHours           int     `mapstructure:"sub_task_max_hours"`
	DefaultPriority           string  `mapstructure:"default_priority"`
	IncludeTestTasks          bool    `mapstructure:"include_test_tasks"`
	IncludeDocTasks           bool    `mapstructure:"include_doc_tasks"`
	EnableDependencyDetection bool    `mapstructure:"enable_dependency_detection"`
}

// NewTaskGeneratorService creates a new enhanced task generator service
func NewTaskGeneratorService(
	mcpClient ports.MCPClient,
	complexityAnalyzer ComplexityAnalyzer,
	templateMatcher TemplateMatcher,
	logger *slog.Logger,
) *DefaultTaskGeneratorService {
	return &DefaultTaskGeneratorService{
		mcpClient:          mcpClient,
		complexityAnalyzer: complexityAnalyzer,
		templateMatcher:    templateMatcher,
		logger:             logger,
		config:             getDefaultTaskGeneratorConfig(),
	}
}

// GenerateFromAnalysis generates tasks from AI analysis results
func (g *DefaultTaskGeneratorService) GenerateFromAnalysis(ctx context.Context, analysis *AIAnalysis, context GenerationContext) ([]*entities.Task, error) {
	if analysis == nil {
		return nil, errors.New("analysis cannot be nil")
	}

	g.logger.Info("generating tasks from AI analysis",
		slog.String("analysis_id", analysis.ID),
		slog.Int("features", len(analysis.KeyFeatures)),
		slog.Int("tech_reqs", len(analysis.TechnicalReqs)))

	var tasks []*entities.Task

	// Generate tasks from key features
	for i, feature := range analysis.KeyFeatures {
		task, err := g.createTaskFromFeature(feature, analysis, context, i)
		if err != nil {
			g.logger.Warn("failed to create task from feature",
				slog.String("feature", feature),
				slog.Any("error", err))
			continue
		}
		tasks = append(tasks, task)

		// Generate sub-tasks if complexity is high and auto-subtasking is enabled
		taskEstimatedHours := task.EstimatedMins / 60
		if g.config.AutoSubTasking && taskEstimatedHours > g.config.MaxTaskHours {
			subTasks, err := g.generateSubTasksFromTask(task, context)
			if err != nil {
				g.logger.Warn("failed to generate sub-tasks",
					slog.String("task_id", task.ID),
					slog.Any("error", err))
			} else {
				tasks = append(tasks, subTasks...)
			}
		}
	}

	// Generate tasks from technical requirements
	for i, techReq := range analysis.TechnicalReqs {
		task, err := g.createTaskFromTechnicalReq(techReq, analysis, context, i)
		if err != nil {
			g.logger.Warn("failed to create task from technical requirement",
				slog.String("requirement", techReq),
				slog.Any("error", err))
			continue
		}
		tasks = append(tasks, task)
	}

	// Add testing tasks if enabled
	if g.config.IncludeTestTasks {
		testTasks := g.generateTestingTasks(analysis, context)
		tasks = append(tasks, testTasks...)
	}

	// Add documentation tasks if enabled
	if g.config.IncludeDocTasks {
		docTasks := g.generateDocumentationTasks(analysis, context)
		tasks = append(tasks, docTasks...)
	}

	// Detect and set dependencies if enabled
	if g.config.EnableDependencyDetection {
		if err := g.DetectDependencies(tasks); err != nil {
			g.logger.Warn("dependency detection failed", slog.Any("error", err))
		}
	}

	// Validate generation results
	if err := g.ValidateGeneration(tasks); err != nil {
		return nil, fmt.Errorf("task generation validation failed: %w", err)
	}

	g.logger.Info("completed task generation from analysis",
		slog.Int("task_count", len(tasks)),
		slog.String("analysis_id", analysis.ID))

	return tasks, nil
}

// GenerateMainTasks generates main tasks from TRD using AI
func (g *DefaultTaskGeneratorService) GenerateMainTasks(ctx context.Context, trd *TRDEntity, rule *GenerationRule) ([]*MainTask, error) {
	if trd == nil {
		return nil, errors.New("TRD cannot be nil")
	}

	g.logger.Info("generating main tasks from TRD",
		slog.String("trd_id", trd.ID),
		slog.Int("requirements", len(trd.Requirements)))

	var mainTasks []*MainTask

	// Use AI to analyze TRD and generate main tasks
	aiPrompt := g.buildMainTaskPrompt(trd, rule)

	// Simulate AI call - in real implementation, this would call MCP AI service
	aiResponse, err := g.callAIForMainTasks(ctx, aiPrompt, trd)
	if err != nil {
		return nil, fmt.Errorf("AI main task generation failed: %w", err)
	}

	// Parse AI response into main tasks
	mainTasks, err = g.parseMainTasksFromAI(aiResponse, trd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Enhance and validate main tasks
	for i, task := range mainTasks {
		task.ID = fmt.Sprintf("MT-%03d", i+1)
		task.AtomicValidation = g.validateAtomicTask(task.Content)
		task.CreatedAt = time.Now()

		// Analyze complexity for duration estimate
		complexity, err := g.complexityAnalyzer.AnalyzeContent(task.Content)
		if err == nil {
			task.Duration = g.estimateMainTaskDuration(complexity)
		}
	}

	// Detect dependencies between main tasks
	g.detectMainTaskDependencies(mainTasks)

	g.logger.Info("generated main tasks from TRD",
		slog.String("trd_id", trd.ID),
		slog.Int("task_count", len(mainTasks)))

	return mainTasks, nil
}

// GenerateSubTasks generates sub-tasks from a main task using AI
func (g *DefaultTaskGeneratorService) GenerateSubTasks(ctx context.Context, mainTask *MainTask, rule *GenerationRule) ([]*SubTask, error) {
	if mainTask == nil {
		return nil, errors.New("main task cannot be nil")
	}

	g.logger.Info("generating sub-tasks from main task",
		slog.String("main_task_id", mainTask.ID),
		slog.String("main_task_name", mainTask.Name))

	// Use AI to break down main task into sub-tasks
	aiPrompt := g.buildSubTaskPrompt(mainTask, rule)

	// Simulate AI call - in real implementation, this would call MCP AI service
	aiResponse, err := g.callAIForSubTasks(ctx, aiPrompt, mainTask)
	if err != nil {
		return nil, fmt.Errorf("AI sub-task generation failed: %w", err)
	}

	// Parse AI response into sub-tasks
	subTasks, err := g.parseSubTasksFromAI(aiResponse, mainTask)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Enhance and validate sub-tasks
	for i, subTask := range subTasks {
		subTask.ID = fmt.Sprintf("ST-%s-%03d", mainTask.ID, i+1)
		subTask.ParentTaskID = mainTask.ID
		subTask.CreatedAt = time.Now()

		// Ensure duration is within constraints
		if subTask.Duration < 2 {
			subTask.Duration = 2
		} else if subTask.Duration > g.config.SubTaskMaxHours {
			subTask.Duration = g.config.SubTaskMaxHours
		}
	}

	// Update main task sub-task count
	mainTask.SubTaskCount = len(subTasks)

	g.logger.Info("generated sub-tasks from main task",
		slog.String("main_task_id", mainTask.ID),
		slog.Int("sub_task_count", len(subTasks)))

	return subTasks, nil
}

// AnalyzeComplexity analyzes the complexity of content
func (g *DefaultTaskGeneratorService) AnalyzeComplexity(content string) (*ComplexityAnalysis, error) {
	return g.complexityAnalyzer.AnalyzeContent(content)
}

// ValidateGeneration validates the generated tasks
func (g *DefaultTaskGeneratorService) ValidateGeneration(tasks interface{}) error {
	switch v := tasks.(type) {
	case []*entities.Task:
		return g.validateEntityTasks(v)
	case []*MainTask:
		return g.validateMainTasks(v)
	case []*SubTask:
		return g.validateSubTasks(v)
	default:
		return fmt.Errorf("unsupported task type: %T", tasks)
	}
}

// EstimateTaskEffort estimates the effort required for a task
func (g *DefaultTaskGeneratorService) EstimateTaskEffort(content string) (int, error) {
	complexity, err := g.complexityAnalyzer.AnalyzeContent(content)
	if err != nil {
		return 0, err
	}
	return g.complexityAnalyzer.EstimateEffort(complexity), nil
}

// FindTaskTemplates finds relevant templates for a feature
func (g *DefaultTaskGeneratorService) FindTaskTemplates(feature, projectType string) []*TaskTemplate {
	bestMatch := g.templateMatcher.FindBestMatch(feature, projectType)
	if bestMatch == nil {
		return []*TaskTemplate{}
	}

	// Return best match and similar templates
	allTemplates := g.templateMatcher.GetAllTemplates()
	var relevantTemplates []*TaskTemplate
	relevantTemplates = append(relevantTemplates, bestMatch)

	// Find similar templates by category
	for _, template := range allTemplates {
		if template.ID != bestMatch.ID && template.Category == bestMatch.Category {
			relevantTemplates = append(relevantTemplates, template)
			if len(relevantTemplates) >= 3 { // Limit to top 3
				break
			}
		}
	}

	return relevantTemplates
}

// ApplyTemplate applies a template to create a task
func (g *DefaultTaskGeneratorService) ApplyTemplate(template *TaskTemplate, context GenerationContext) (*entities.Task, error) {
	if template == nil {
		return nil, errors.New("template cannot be nil")
	}

	// Create task based on template
	task := &entities.Task{
		ID:            uuid.New().String(),
		Content:       template.Description,
		Status:        entities.StatusPending,
		Priority:      g.templateComplexityToPriority(template.Complexity),
		Repository:    context.Repository,
		Tags:          g.generateTagsFromTemplate(template),
		EstimatedMins: template.Hours * 60, // Convert hours to minutes
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AISuggested:   true,
	}

	g.logger.Debug("applied template to create task",
		slog.String("template_id", template.ID),
		slog.String("task_id", task.ID))

	return task, nil
}

// DetectDependencies detects and sets dependencies between tasks
func (g *DefaultTaskGeneratorService) DetectDependencies(tasks []*entities.Task) error {
	if len(tasks) <= 1 {
		return nil // No dependencies to detect
	}

	g.logger.Debug("detecting dependencies between tasks", slog.Int("task_count", len(tasks)))

	// Simple dependency detection based on task content and types
	for i, task := range tasks {
		dependencies := g.findTaskDependencies(task, tasks[:i])
		if len(dependencies) > 0 {
			// Store dependencies indicator in tags (since no metadata field)
			task.Tags = append(task.Tags, "has-dependencies")
		}
	}

	return nil
}

// ValidateTaskHierarchy validates the hierarchy between main tasks and sub-tasks
func (g *DefaultTaskGeneratorService) ValidateTaskHierarchy(mainTasks []*MainTask, subTasks []*SubTask) error {
	// Build map of main task IDs
	mainTaskMap := make(map[string]*MainTask)
	for _, mainTask := range mainTasks {
		mainTaskMap[mainTask.ID] = mainTask
	}

	// Validate each sub-task has a valid parent
	for _, subTask := range subTasks {
		if subTask.ParentTaskID == "" {
			return fmt.Errorf("sub-task %s has no parent task ID", subTask.ID)
		}

		_, exists := mainTaskMap[subTask.ParentTaskID]
		if !exists {
			return fmt.Errorf("sub-task %s references non-existent parent task %s", subTask.ID, subTask.ParentTaskID)
		}
	}

	// Update main task sub-task counts
	subTaskCounts := make(map[string]int)
	for _, subTask := range subTasks {
		subTaskCounts[subTask.ParentTaskID]++
	}

	for _, mainTask := range mainTasks {
		mainTask.SubTaskCount = subTaskCounts[mainTask.ID]
	}

	g.logger.Debug("validated task hierarchy",
		slog.Int("main_tasks", len(mainTasks)),
		slog.Int("sub_tasks", len(subTasks)))

	return nil
}

// Private helper methods

func (g *DefaultTaskGeneratorService) createTaskFromFeature(feature string, _ *AIAnalysis, context GenerationContext, index int) (*entities.Task, error) {
	// Find best matching template
	template := g.templateMatcher.FindBestMatch(feature, context.ProjectType)

	// Analyze complexity
	complexity, err := g.complexityAnalyzer.AnalyzeContent(feature)
	if err != nil {
		g.logger.Warn("complexity analysis failed for feature", slog.String("feature", feature))
		complexity = &ComplexityAnalysis{
			Score:          5.0,
			Level:          "medium",
			EstimatedHours: 4,
			Confidence:     0.5,
		}
	}

	// Generate task content
	content := g.generateTaskContent(feature, template, complexity)

	// Create task entity
	task := &entities.Task{
		ID:            uuid.New().String(),
		Content:       content,
		Status:        entities.StatusPending,
		Priority:      g.determinePriority(complexity, template),
		Repository:    context.Repository,
		Tags:          g.generateTags(feature, template, complexity),
		EstimatedMins: g.complexityAnalyzer.EstimateEffort(complexity) * 60, // Convert hours to minutes
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AISuggested:   true,
	}

	return task, nil
}

func (g *DefaultTaskGeneratorService) createTaskFromTechnicalReq(techReq string, _ *AIAnalysis, context GenerationContext, index int) (*entities.Task, error) {
	// Find template for technical requirement
	template := g.templateMatcher.FindBestMatch(techReq, "backend") // Default to backend for tech reqs

	// Analyze complexity
	complexity, err := g.complexityAnalyzer.AnalyzeContent(techReq)
	if err != nil {
		complexity = &ComplexityAnalysis{
			Score:          6.0,
			Level:          "medium",
			EstimatedHours: 6,
			Confidence:     0.5,
		}
	}

	task := &entities.Task{
		ID:            uuid.New().String(),
		Content:       "Technical Requirement: " + techReq,
		Status:        entities.StatusPending,
		Priority:      entities.PriorityMedium, // Tech reqs typically medium priority
		Repository:    context.Repository,
		Tags:          []string{"technical", "requirement", template.Category},
		EstimatedMins: g.complexityAnalyzer.EstimateEffort(complexity) * 60, // Convert hours to minutes
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AISuggested:   true,
	}

	return task, nil
}

func (g *DefaultTaskGeneratorService) generateSubTasksFromTask(parentTask *entities.Task, _ GenerationContext) ([]*entities.Task, error) {
	// Break down complex task into smaller sub-tasks
	subTaskDescriptions := g.breakDownComplexTask(parentTask.Content)

	var subTasks []*entities.Task
	for _, desc := range subTaskDescriptions {
		subTask := &entities.Task{
			ID:            uuid.New().String(),
			Content:       desc,
			Status:        entities.StatusPending,
			Priority:      parentTask.Priority,
			Repository:    parentTask.Repository,
			Tags:          append(parentTask.Tags, "subtask"),
			EstimatedMins: g.config.SubTaskMaxHours * 60, // Convert max hours to minutes
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			AISuggested:   true,
		}
		subTasks = append(subTasks, subTask)
	}

	return subTasks, nil
}

func (g *DefaultTaskGeneratorService) generateTestingTasks(analysis *AIAnalysis, context GenerationContext) []*entities.Task {
	var testTasks []*entities.Task

	// Generate unit test task
	unitTestTask := &entities.Task{
		ID:            uuid.New().String(),
		Content:       "Implement comprehensive unit tests for core functionality",
		Status:        entities.StatusPending,
		Priority:      entities.PriorityMedium,
		Repository:    context.Repository,
		Tags:          []string{"testing", "unit-tests", "quality"},
		EstimatedMins: 8 * 60, // 8 hours in minutes
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AISuggested:   true,
	}
	testTasks = append(testTasks, unitTestTask)

	// Generate integration test task if complexity is medium or high
	if analysis.Complexity != "low" {
		integrationTestTask := &entities.Task{
			ID:            uuid.New().String(),
			Content:       "Implement integration tests for system components",
			Status:        entities.StatusPending,
			Priority:      entities.PriorityMedium,
			Repository:    context.Repository,
			Tags:          []string{"testing", "integration-tests", "quality"},
			EstimatedMins: 12 * 60, // 12 hours in minutes
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			AISuggested:   true,
		}
		testTasks = append(testTasks, integrationTestTask)
	}

	return testTasks
}

func (g *DefaultTaskGeneratorService) generateDocumentationTasks(_ *AIAnalysis, context GenerationContext) []*entities.Task {
	var docTasks []*entities.Task

	// Generate API documentation task
	apiDocTask := &entities.Task{
		ID:            uuid.New().String(),
		Content:       "Create comprehensive API documentation with examples",
		Status:        entities.StatusPending,
		Priority:      entities.PriorityLow,
		Repository:    context.Repository,
		Tags:          []string{"documentation", "api", "examples"},
		EstimatedMins: 6 * 60, // 6 hours in minutes
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AISuggested:   true,
	}
	docTasks = append(docTasks, apiDocTask)

	// Generate user guide task
	userGuideTask := &entities.Task{
		ID:            uuid.New().String(),
		Content:       "Write user guide and getting started documentation",
		Status:        entities.StatusPending,
		Priority:      entities.PriorityLow,
		Repository:    context.Repository,
		Tags:          []string{"documentation", "user-guide", "onboarding"},
		EstimatedMins: 8 * 60, // 8 hours in minutes
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AISuggested:   true,
	}
	docTasks = append(docTasks, userGuideTask)

	return docTasks
}

// AI service interaction methods (simulated for now)

func (g *DefaultTaskGeneratorService) buildMainTaskPrompt(trd *TRDEntity, _ *GenerationRule) string {
	prompt := fmt.Sprintf(`
Generate main tasks for the following Technical Requirements Document:

Title: %s
Architecture: %s
Tech Stack: %s
Requirements: %s
Implementation: %s

Please generate 5-7 main tasks that are:
1. Atomic and deployable phases
2. Each representing 1-2 weeks of work
3. Logically ordered with clear dependencies
4. Focused on core implementation milestones

Return as JSON array with fields: name, description, phase, duration, dependencies
`, trd.Title, trd.Architecture, joinWords(trd.TechStack), joinWords(trd.Requirements), joinWords(trd.Implementation))

	return prompt
}

func (g *DefaultTaskGeneratorService) buildSubTaskPrompt(mainTask *MainTask, _ *GenerationRule) string {
	prompt := fmt.Sprintf(`
Break down the following main task into 3-5 sub-tasks:

Main Task: %s
Description: %s
Content: %s

Please generate sub-tasks that are:
1. Each 2-4 hours of focused work
2. Implementable by a single developer
3. Have clear deliverables and acceptance criteria
4. Logically ordered within the main task

Return as JSON array with fields: name, duration_hours, type, deliverables, acceptance_criteria, dependencies
`, mainTask.Name, mainTask.Description, mainTask.Content)

	return prompt
}

func (g *DefaultTaskGeneratorService) callAIForMainTasks(ctx context.Context, prompt string, trd *TRDEntity) (string, error) {
	// Simulated AI response - in real implementation, this would call MCP AI service
	response := `[
		{
			"name": "Set up project structure and dependencies",
			"description": "Initialize project with proper structure, dependencies, and development environment",
			"phase": "setup",
			"duration": "3-5 days",
			"dependencies": []
		},
		{
			"name": "Implement core business logic",
			"description": "Develop the main business logic and core functionality",
			"phase": "development",
			"duration": "1-2 weeks",
			"dependencies": ["MT-001"]
		},
		{
			"name": "Develop API endpoints and services",
			"description": "Create REST API endpoints and service layer",
			"phase": "development", 
			"duration": "1 week",
			"dependencies": ["MT-002"]
		},
		{
			"name": "Implement data persistence layer",
			"description": "Set up database, models, and data access layer",
			"phase": "development",
			"duration": "1 week", 
			"dependencies": ["MT-001"]
		},
		{
			"name": "Add security and validation",
			"description": "Implement authentication, authorization, and input validation",
			"phase": "security",
			"duration": "1 week",
			"dependencies": ["MT-003", "MT-004"]
		},
		{
			"name": "Create comprehensive test suite",
			"description": "Implement unit, integration, and end-to-end tests",
			"phase": "testing",
			"duration": "1 week",
			"dependencies": ["MT-002", "MT-003", "MT-004"]
		},
		{
			"name": "Documentation and deployment setup",
			"description": "Create documentation, deployment scripts, and CI/CD pipeline",
			"phase": "deployment",
			"duration": "3-5 days",
			"dependencies": ["MT-006"]
		}
	]`

	return response, nil
}

func (g *DefaultTaskGeneratorService) callAIForSubTasks(ctx context.Context, prompt string, mainTask *MainTask) (string, error) {
	// Simulated AI response - in real implementation, this would call MCP AI service
	response := `[
		{
			"name": "Research and plan implementation approach",
			"duration_hours": 3,
			"type": "research",
			"deliverables": ["research notes", "implementation plan"],
			"acceptance_criteria": ["Approach is documented", "Plan is reviewed and approved"],
			"dependencies": []
		},
		{
			"name": "Implement core functionality",
			"duration_hours": 4,
			"type": "implementation", 
			"deliverables": ["working code", "unit tests"],
			"acceptance_criteria": ["Code is functional", "Tests pass", "Code review completed"],
			"dependencies": ["ST-001"]
		},
		{
			"name": "Add error handling and validation",
			"duration_hours": 2,
			"type": "implementation",
			"deliverables": ["error handling code", "validation logic"],
			"acceptance_criteria": ["Error cases handled", "Validation works correctly"],
			"dependencies": ["ST-002"]
		},
		{
			"name": "Testing and documentation",
			"duration_hours": 3,
			"type": "testing",
			"deliverables": ["test results", "documentation"],
			"acceptance_criteria": ["All tests pass", "Documentation is complete"],
			"dependencies": ["ST-003"]
		}
	]`

	return response, nil
}

func (g *DefaultTaskGeneratorService) parseMainTasksFromAI(aiResponse string, _ *TRDEntity) ([]*MainTask, error) {
	var rawTasks []map[string]interface{}
	if err := json.Unmarshal([]byte(aiResponse), &rawTasks); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	var mainTasks []*MainTask
	for _, rawTask := range rawTasks {
		task := &MainTask{
			Name:        getStringFromMap(rawTask, "name"),
			Description: getStringFromMap(rawTask, "description"),
			Phase:       getStringFromMap(rawTask, "phase"),
			Duration:    getStringFromMap(rawTask, "duration"),
			Content:     getStringFromMap(rawTask, "description"),
		}

		// Parse dependencies
		if deps, ok := rawTask["dependencies"].([]interface{}); ok {
			for _, dep := range deps {
				if depStr, ok := dep.(string); ok {
					task.Dependencies = append(task.Dependencies, depStr)
				}
			}
		}

		mainTasks = append(mainTasks, task)
	}

	return mainTasks, nil
}

func (g *DefaultTaskGeneratorService) parseSubTasksFromAI(aiResponse string, _ *MainTask) ([]*SubTask, error) {
	var rawTasks []map[string]interface{}
	if err := json.Unmarshal([]byte(aiResponse), &rawTasks); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	var subTasks []*SubTask
	for _, rawTask := range rawTasks {
		task := &SubTask{
			Name:     getStringFromMap(rawTask, "name"),
			Duration: getIntFromMap(rawTask, "duration_hours"),
			Type:     getStringFromMap(rawTask, "type"),
			Content:  getStringFromMap(rawTask, "name"),
		}

		// Parse deliverables
		if deliverables, ok := rawTask["deliverables"].([]interface{}); ok {
			for _, deliverable := range deliverables {
				if delStr, ok := deliverable.(string); ok {
					task.Deliverables = append(task.Deliverables, delStr)
				}
			}
		}

		// Parse acceptance criteria
		if criteria, ok := rawTask["acceptance_criteria"].([]interface{}); ok {
			for _, criterion := range criteria {
				if critStr, ok := criterion.(string); ok {
					task.AcceptanceCriteria = append(task.AcceptanceCriteria, critStr)
				}
			}
		}

		// Parse dependencies
		if deps, ok := rawTask["dependencies"].([]interface{}); ok {
			for _, dep := range deps {
				if depStr, ok := dep.(string); ok {
					task.Dependencies = append(task.Dependencies, depStr)
				}
			}
		}

		subTasks = append(subTasks, task)
	}

	return subTasks, nil
}

// Validation methods

func (g *DefaultTaskGeneratorService) validateEntityTasks(tasks []*entities.Task) error {
	if len(tasks) == 0 {
		return errors.New("no tasks generated")
	}

	if len(tasks) > g.config.MaxTasksPerPRD {
		return fmt.Errorf("too many tasks generated: %d (max: %d)", len(tasks), g.config.MaxTasksPerPRD)
	}

	for _, task := range tasks {
		if task.Content == "" {
			return fmt.Errorf("task %s has empty content", task.ID)
		}
		taskEstimatedHours := task.EstimatedMins / 60
		if taskEstimatedHours < g.config.MinTaskHours {
			return fmt.Errorf("task %s estimated hours too low: %d (min: %d)", task.ID, taskEstimatedHours, g.config.MinTaskHours)
		}
		if taskEstimatedHours > g.config.MaxTaskHours {
			return fmt.Errorf("task %s estimated hours too high: %d (max: %d)", task.ID, taskEstimatedHours, g.config.MaxTaskHours)
		}
	}

	return nil
}

func (g *DefaultTaskGeneratorService) validateMainTasks(tasks []*MainTask) error {
	if len(tasks) == 0 {
		return errors.New("no main tasks generated")
	}

	for _, task := range tasks {
		if task.Name == "" {
			return fmt.Errorf("main task %s has empty name", task.ID)
		}
		if task.Content == "" {
			return fmt.Errorf("main task %s has empty content", task.ID)
		}
		if task.Phase == "" {
			return fmt.Errorf("main task %s has empty phase", task.ID)
		}
	}

	return nil
}

func (g *DefaultTaskGeneratorService) validateSubTasks(tasks []*SubTask) error {
	for _, task := range tasks {
		if task.Name == "" {
			return fmt.Errorf("sub-task %s has empty name", task.ID)
		}
		if task.ParentTaskID == "" {
			return fmt.Errorf("sub-task %s has no parent task", task.ID)
		}
		if task.Duration < 2 || task.Duration > g.config.SubTaskMaxHours {
			return fmt.Errorf("sub-task %s duration out of range: %d hours (range: 2-%d)", task.ID, task.Duration, g.config.SubTaskMaxHours)
		}
	}

	return nil
}

// Helper methods

func (g *DefaultTaskGeneratorService) validateAtomicTask(content string) bool {
	// Simple atomic validation - task should be focused and not too complex
	words := splitWords(content)
	return len(words) < 30 && !containsMultipleActions(content)
}

func (g *DefaultTaskGeneratorService) estimateMainTaskDuration(complexity *ComplexityAnalysis) string {
	hours := complexity.EstimatedHours
	switch {
	case hours <= 16:
		return "1-3 days"
	case hours <= 40:
		return "1 week"
	case hours <= 80:
		return "2 weeks"
	default:
		return "3+ weeks"
	}
}

func (g *DefaultTaskGeneratorService) detectMainTaskDependencies(tasks []*MainTask) {
	// Simple sequential dependencies for now
	for i := 1; i < len(tasks); i++ {
		if len(tasks[i].Dependencies) == 0 {
			tasks[i].Dependencies = []string{tasks[i-1].ID}
		}
	}
}

func (g *DefaultTaskGeneratorService) generateTaskContent(feature string, template *TaskTemplate, complexity *ComplexityAnalysis) string {
	baseContent := feature
	if template != nil && template.Description != "" {
		baseContent = template.Description + ": " + feature
	}

	// Add complexity context
	if complexity.Level == "high" {
		baseContent += " (High complexity - consider breaking into smaller tasks)"
	}

	return baseContent
}

func (g *DefaultTaskGeneratorService) determinePriority(complexity *ComplexityAnalysis, _ *TaskTemplate) entities.Priority {
	// Determine priority based on complexity and template
	if complexity.Score >= 8.0 {
		return entities.PriorityHigh
	} else if complexity.Score >= 5.0 {
		return entities.PriorityMedium
	}
	return entities.PriorityLow
}

func (g *DefaultTaskGeneratorService) generateTags(feature string, template *TaskTemplate, complexity *ComplexityAnalysis) []string {
	var tags []string

	// Add complexity tag
	tags = append(tags, complexity.Level+"-complexity")

	// Add template category
	if template != nil {
		tags = append(tags, template.Category)
		tags = append(tags, template.Type)
	}

	// Add feature-based tags
	featureLower := toLowerCase(feature)
	if containsKeyword(featureLower, "api") {
		tags = append(tags, "api")
	}
	if containsKeyword(featureLower, "ui") {
		tags = append(tags, "frontend")
	}
	if containsKeyword(featureLower, "database") {
		tags = append(tags, "backend")
	}

	return tags
}

func (g *DefaultTaskGeneratorService) templateComplexityToPriority(complexity string) entities.Priority {
	switch toLowerCase(complexity) {
	case "high":
		return entities.PriorityHigh
	case "low":
		return entities.PriorityLow
	default:
		return entities.PriorityMedium
	}
}

func (g *DefaultTaskGeneratorService) generateTagsFromTemplate(template *TaskTemplate) []string {
	var tags []string
	tags = append(tags, template.Category)
	tags = append(tags, template.Type)
	tags = append(tags, template.Complexity+"-complexity")

	// Add keyword-based tags
	for _, keyword := range template.Keywords {
		if len(keyword) > 2 { // Skip very short keywords
			tags = append(tags, keyword)
		}
	}

	return tags
}

func (g *DefaultTaskGeneratorService) findTaskDependencies(task *entities.Task, previousTasks []*entities.Task) []string {
	var dependencies []string
	taskContentLower := toLowerCase(task.Content)

	// Simple dependency detection based on content analysis
	for _, prevTask := range previousTasks {
		prevContentLower := toLowerCase(prevTask.Content)

		// Check for common dependency patterns
		if g.hasDependencyRelation(taskContentLower, prevContentLower) {
			dependencies = append(dependencies, prevTask.ID)
		}
	}

	return dependencies
}

func (g *DefaultTaskGeneratorService) hasDependencyRelation(current, previous string) bool {
	// Database tasks typically depend on schema/model tasks
	if containsKeyword(current, "database") && containsKeyword(previous, "model") {
		return true
	}

	// API tasks depend on business logic
	if containsKeyword(current, "api") && containsKeyword(previous, "logic") {
		return true
	}

	// UI tasks depend on API tasks
	if containsKeyword(current, "ui") && containsKeyword(previous, "api") {
		return true
	}

	// Testing tasks depend on implementation tasks
	if containsKeyword(current, "test") && containsKeyword(previous, "implement") {
		return true
	}

	return false
}

func (g *DefaultTaskGeneratorService) breakDownComplexTask(content string) []string {
	// Simple task breakdown strategy
	return []string{
		"Research and design approach for: " + content,
		"Core implementation of: " + content,
		"Testing and validation for: " + content,
		"Documentation and cleanup for: " + content,
	}
}

// Utility functions

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	if val, ok := m[key].(int); ok {
		return val
	}
	return 0
}

func containsMultipleActions(content string) bool {
	actionWords := []string{"implement", "create", "build", "develop", "design", "test", "deploy", "configure"}
	contentLower := toLowerCase(content)
	actionCount := 0

	for _, action := range actionWords {
		if containsKeyword(contentLower, action) {
			actionCount++
			if actionCount > 1 {
				return true
			}
		}
	}

	return false
}

// getDefaultTaskGeneratorConfig returns default configuration
func getDefaultTaskGeneratorConfig() *TaskGeneratorConfig {
	return &TaskGeneratorConfig{
		MaxTasksPerPRD:            20,
		ComplexityThreshold:       6.0,
		MinTaskHours:              1,
		MaxTaskHours:              16,
		AutoSubTasking:            true,
		SubTaskMaxHours:           4,
		DefaultPriority:           "medium",
		IncludeTestTasks:          true,
		IncludeDocTasks:           true,
		EnableDependencyDetection: true,
	}
}
