// Package tasks provides AI-powered task generation and management capabilities.
package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"lerian-mcp-memory/internal/ai"
	"lerian-mcp-memory/pkg/types"
)

// Generator handles AI-powered task generation from PRDs
type Generator struct {
	aiService    AIService
	complexityAnalyzer *ComplexityAnalyzer
	dependencyDetector *DependencyDetector
	templateMatcher    *TemplateMatcher
	validator         *Validator
	scorer           *Scorer
	config           GeneratorConfig
}

// AIService defines the interface for AI service integration
type AIService interface {
	ProcessRequest(ctx context.Context, req *ai.Request) (*ai.Response, error)
}

// GeneratorConfig represents configuration for the task generator
type GeneratorConfig struct {
	DefaultModel         ai.Model      `json:"default_model"`
	MaxTasksPerRequest   int           `json:"max_tasks_per_request"`
	QualityThreshold     float64       `json:"quality_threshold"`
	GenerationTimeout    time.Duration `json:"generation_timeout"`
	EnableTemplates      bool          `json:"enable_templates"`
	EnableDependencies   bool          `json:"enable_dependencies"`
	EnableComplexityAnalysis bool      `json:"enable_complexity_analysis"`
	RetryAttempts        int           `json:"retry_attempts"`
}

// DefaultGeneratorConfig returns default configuration
func DefaultGeneratorConfig() GeneratorConfig {
	return GeneratorConfig{
		DefaultModel:         ai.ModelClaude,
		MaxTasksPerRequest:   50,
		QualityThreshold:     0.7,
		GenerationTimeout:    60 * time.Second,
		EnableTemplates:      true,
		EnableDependencies:   true,
		EnableComplexityAnalysis: true,
		RetryAttempts:        3,
	}
}

// NewGenerator creates a new task generator
func NewGenerator(aiService AIService, config GeneratorConfig) *Generator {
	return &Generator{
		aiService:         aiService,
		complexityAnalyzer: NewComplexityAnalyzer(),
		dependencyDetector: NewDependencyDetector(),
		templateMatcher:   NewTemplateMatcher(),
		validator:        NewValidator(),
		scorer:          NewScorer(),
		config:          config,
	}
}

// GenerateTasks generates tasks from a PRD document
func (g *Generator) GenerateTasks(ctx context.Context, req *types.TaskSuggestionRequest) (*types.TaskSuggestionResponse, error) {
	startTime := time.Now()
	
	// Validate request
	if err := g.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, g.config.GenerationTimeout)
	defer cancel()

	// Generate initial tasks using AI
	rawTasks, metadata, err := g.generateRawTasks(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate raw tasks: %w", err)
	}

	// Process and enhance tasks
	processedTasks, err := g.processTasks(ctx, rawTasks, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process tasks: %w", err)
	}

	// Filter by quality threshold
	qualityTasks, filteredCount := g.filterByQuality(processedTasks, req.Options.MinQualityScore)

	// Generate dependency graph
	var depGraph types.DependencyGraph
	if g.config.EnableDependencies && req.Options.IncludeDependencies {
		depGraph = g.dependencyDetector.GenerateDependencyGraph(qualityTasks)
	}

	// Generate recommendations
	recommendations := g.generateRecommendations(qualityTasks, req)

	// Generate next steps
	nextSteps := g.generateNextSteps(qualityTasks, &req.ProjectState)

	metadata.GenerationTime = time.Since(startTime)

	return &types.TaskSuggestionResponse{
		Tasks:              qualityTasks,
		TotalGenerated:     len(rawTasks),
		QualityFiltered:    filteredCount,
		GenerationMetadata: metadata,
		DependencyGraph:    depGraph,
		Recommendations:    recommendations,
		NextSteps:          nextSteps,
	}, nil
}

// generateRawTasks generates initial tasks using AI
func (g *Generator) generateRawTasks(ctx context.Context, req *types.TaskSuggestionRequest) ([]types.Task, types.GenerationMetadata, error) {
	// Build AI prompt
	prompt := g.buildGenerationPrompt(req)
	
	// Create AI request
	aiReq := &ai.Request{
		ID:    fmt.Sprintf("task_generation_%d", time.Now().Unix()),
		Model: ai.Model(req.Options.AIModel),
		Messages: []ai.Message{
			{
				Role:    "system",
				Content: g.getSystemPrompt(),
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Context: map[string]string{
			"operation": "task_generation",
			"prd_id":    req.PRDID,
		},
		Metadata: ai.RequestMetadata{
			Repository: req.Context.Repository,
			Tags:       []string{"task_generation", "ai_analysis"},
			CreatedAt:  time.Now(),
		},
	}

	// Set default model if not specified
	if aiReq.Model == "" {
		aiReq.Model = g.config.DefaultModel
	}

	// Call AI service
	resp, err := g.aiService.ProcessRequest(ctx, aiReq)
	if err != nil {
		return nil, types.GenerationMetadata{}, fmt.Errorf("AI request failed: %w", err)
	}

	// Parse AI response
	tasks, err := g.parseAIResponse(resp.Content)
	if err != nil {
		return nil, types.GenerationMetadata{}, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Create generation metadata
	metadata := types.GenerationMetadata{
		AIModel:          string(resp.Model),
		QualityThreshold: req.Options.MinQualityScore,
		ProcessingSteps: []types.ProcessingStep{
			{
				Name:      "ai_generation",
				Status:    types.StepStatusCompleted,
				StartTime: time.Now().Add(-resp.Latency),
				EndTime:   time.Now(),
				Duration:  resp.Latency,
			},
		},
	}

	return tasks, metadata, nil
}

// buildGenerationPrompt builds the AI prompt for task generation
func (g *Generator) buildGenerationPrompt(req *types.TaskSuggestionRequest) string {
	var prompt strings.Builder
	
	prompt.WriteString("Generate actionable development tasks based on the following PRD content.\n\n")
	
	// Add PRD content
	if req.PRDContent != "" {
		prompt.WriteString("PRD CONTENT:\n")
		prompt.WriteString(req.PRDContent)
		prompt.WriteString("\n\n")
	}
	
	// Add context
	prompt.WriteString("PROJECT CONTEXT:\n")
	prompt.WriteString(fmt.Sprintf("- Project: %s\n", req.Context.ProjectName))
	prompt.WriteString(fmt.Sprintf("- Type: %s\n", req.Context.ProjectType))
	if len(req.Context.TechStack) > 0 {
		prompt.WriteString(fmt.Sprintf("- Tech Stack: %s\n", strings.Join(req.Context.TechStack, ", ")))
	}
	if req.Context.TeamSize > 0 {
		prompt.WriteString(fmt.Sprintf("- Team Size: %d\n", req.Context.TeamSize))
	}
	if req.Context.Timeline != "" {
		prompt.WriteString(fmt.Sprintf("- Timeline: %s\n", req.Context.Timeline))
	}
	
	// Add existing tasks for context
	if len(req.ExistingTasks) > 0 {
		prompt.WriteString("\nEXISTING TASKS:\n")
		for _, task := range req.ExistingTasks {
			prompt.WriteString(fmt.Sprintf("- %s (%s, %s)\n", task.Title, task.Type, task.Status))
		}
	}
	
	// Add generation options
	prompt.WriteString("\nGENERATION REQUIREMENTS:\n")
	prompt.WriteString(fmt.Sprintf("- Maximum tasks: %d\n", req.Options.MaxTasks))
	prompt.WriteString(fmt.Sprintf("- Generation style: %s\n", req.Options.GenerationStyle))
	
	if len(req.Options.TaskTypes) > 0 {
		types := make([]string, len(req.Options.TaskTypes))
		for i, t := range req.Options.TaskTypes {
			types[i] = string(t)
		}
		prompt.WriteString(fmt.Sprintf("- Preferred task types: %s\n", strings.Join(types, ", ")))
	}
	
	// Add response format
	prompt.WriteString("\nRESPONSE FORMAT:\n")
	prompt.WriteString("Return tasks as a JSON array with the following structure for each task:\n")
	prompt.WriteString(`{
  "title": "Clear, actionable task title",
  "description": "Detailed task description",
  "type": "implementation|design|testing|documentation|research|review|deployment|architecture|bugfix|refactoring|integration|analysis",
  "priority": "low|medium|high|critical|blocking",
  "estimated_hours": 8.0,
  "acceptance_criteria": ["Specific, testable criteria"],
  "required_skills": ["Required technical skills"],
  "dependencies": ["Dependencies on other tasks"],
  "tags": ["Relevant tags"]
}`)

	return prompt.String()
}

// getSystemPrompt returns the system prompt for task generation
func (g *Generator) getSystemPrompt() string {
	return `You are an expert project manager and software architect specializing in breaking down Product Requirements Documents (PRDs) into actionable development tasks.

Your expertise includes:
- Analyzing complex requirements and user stories
- Creating detailed, actionable tasks with clear acceptance criteria
- Estimating task complexity and effort accurately
- Identifying task dependencies and relationships
- Ensuring tasks are specific, measurable, and achievable
- Optimizing task breakdown for different development methodologies (Agile, Waterfall, Kanban)

Guidelines for task generation:
1. Make tasks specific and actionable - avoid vague descriptions
2. Include clear acceptance criteria that can be tested
3. Provide realistic effort estimates based on complexity
4. Consider technical dependencies and logical order
5. Balance task granularity - not too high-level, not too detailed
6. Include appropriate task types (implementation, testing, documentation, etc.)
7. Assign priorities based on business value and dependencies
8. Consider team skills and project constraints
9. Ensure tasks align with the overall project timeline and goals
10. Include necessary research and design tasks before implementation

Focus on creating high-quality, well-structured tasks that development teams can execute efficiently.`
}

// parseAIResponse parses the AI response and extracts tasks
func (g *Generator) parseAIResponse(content string) ([]types.Task, error) {
	// Find JSON content in the response
	jsonStart := strings.Index(content, "[")
	if jsonStart == -1 {
		return nil, fmt.Errorf("no JSON array found in response")
	}
	
	jsonEnd := strings.LastIndex(content, "]")
	if jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("invalid JSON array in response")
	}
	
	jsonContent := content[jsonStart : jsonEnd+1]
	
	// Parse raw task data
	var rawTasks []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &rawTasks); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	// Convert to task objects
	tasks := make([]types.Task, 0, len(rawTasks))
	for i, raw := range rawTasks {
		task, err := g.convertRawTask(raw, i)
		if err != nil {
			// Log error but continue with other tasks
			continue
		}
		tasks = append(tasks, task)
	}
	
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no valid tasks could be parsed from response")
	}
	
	return tasks, nil
}

// convertRawTask converts raw task data to Task struct
func (g *Generator) convertRawTask(raw map[string]interface{}, index int) (types.Task, error) {
	task := types.Task{
		ID:        fmt.Sprintf("task_%d_%d", time.Now().Unix(), index),
		Timestamps: types.TaskTimestamps{
			Created: time.Now(),
			Updated: time.Now(),
		},
		Metadata: types.TaskMetadata{
			GenerationSource: "ai_generated",
		},
	}
	
	// Extract required fields
	if title, ok := raw["title"].(string); ok {
		task.Title = title
	} else {
		return task, fmt.Errorf("missing or invalid title")
	}
	
	if desc, ok := raw["description"].(string); ok {
		task.Description = desc
	} else {
		return task, fmt.Errorf("missing or invalid description")
	}
	
	// Extract task type
	if taskType, ok := raw["type"].(string); ok {
		task.Type = types.TaskType(taskType)
	} else {
		task.Type = types.TaskTypeImplementation // default
	}
	
	// Extract priority
	if priority, ok := raw["priority"].(string); ok {
		task.Priority = types.TaskPriority(priority)
	} else {
		task.Priority = types.TaskPriorityMedium // default
	}
	
	// Extract effort estimate
	if hours, ok := raw["estimated_hours"].(float64); ok {
		task.EstimatedEffort = types.EffortEstimate{
			Hours:            hours,
			Days:             hours / 8.0,
			Confidence:       0.7, // Default confidence
			EstimationMethod: "ai_generated",
		}
	}
	
	// Extract acceptance criteria
	if criteria, ok := raw["acceptance_criteria"].([]interface{}); ok {
		task.AcceptanceCriteria = make([]string, 0, len(criteria))
		for _, c := range criteria {
			if cStr, ok := c.(string); ok {
				task.AcceptanceCriteria = append(task.AcceptanceCriteria, cStr)
			}
		}
	}
	
	// Extract required skills
	if skills, ok := raw["required_skills"].([]interface{}); ok {
		skillList := make([]string, 0, len(skills))
		for _, s := range skills {
			if sStr, ok := s.(string); ok {
				skillList = append(skillList, sStr)
			}
		}
		task.Complexity.RequiredSkills = skillList
	}
	
	// Extract dependencies
	if deps, ok := raw["dependencies"].([]interface{}); ok {
		task.Dependencies = make([]string, 0, len(deps))
		for _, d := range deps {
			if dStr, ok := d.(string); ok {
				task.Dependencies = append(task.Dependencies, dStr)
			}
		}
	}
	
	// Extract tags
	if tags, ok := raw["tags"].([]interface{}); ok {
		task.Tags = make([]string, 0, len(tags))
		for _, t := range tags {
			if tStr, ok := t.(string); ok {
				task.Tags = append(task.Tags, tStr)
			}
		}
	}
	
	// Set default status
	task.Status = types.TaskStatusTodo
	
	return task, nil
}

// processTasks processes and enhances generated tasks
func (g *Generator) processTasks(ctx context.Context, tasks []types.Task, req *types.TaskSuggestionRequest) ([]types.Task, error) {
	processedTasks := make([]types.Task, 0, len(tasks))
	
	for _, task := range tasks {
		// Enhance with complexity analysis
		if g.config.EnableComplexityAnalysis {
			complexity, err := g.complexityAnalyzer.AnalyzeComplexity(ctx, &task, req.Context)
			if err != nil {
				// Log error but continue
				task.Complexity = types.TaskComplexity{
					Level: types.ComplexityModerate,
					Score: 0.5,
				}
			} else {
				task.Complexity = complexity
			}
		}
		
		// Apply template matching if enabled
		if g.config.EnableTemplates && req.Options.UseTemplates {
			if template := g.templateMatcher.FindBestMatch(&task, req.Context); template != nil {
				task = g.templateMatcher.ApplyTemplate(&task, template)
			}
		}
		
		// Validate task
		validation := g.validator.ValidateTask(&task)
		if !validation.IsValid {
			// Skip invalid tasks or try to fix them
			continue
		}
		
		// Calculate quality score
		qualityScore := g.scorer.ScoreTask(&task, req.Context)
		task.QualityScore = qualityScore
		
		processedTasks = append(processedTasks, task)
	}
	
	return processedTasks, nil
}

// filterByQuality filters tasks by quality threshold
func (g *Generator) filterByQuality(tasks []types.Task, threshold float64) ([]types.Task, int) {
	if threshold <= 0 {
		return tasks, 0
	}
	
	filtered := make([]types.Task, 0, len(tasks))
	filteredCount := 0
	
	for _, task := range tasks {
		if task.QualityScore.OverallScore >= threshold {
			filtered = append(filtered, task)
		} else {
			filteredCount++
		}
	}
	
	return filtered, filteredCount
}

// generateRecommendations generates recommendations based on generated tasks
func (g *Generator) generateRecommendations(tasks []types.Task, req *types.TaskSuggestionRequest) []string {
	recommendations := []string{}
	
	// Analyze task distribution
	typeCount := make(map[types.TaskType]int)
	priorityCount := make(map[types.TaskPriority]int)
	complexityCount := make(map[types.ComplexityLevel]int)
	
	for _, task := range tasks {
		typeCount[task.Type]++
		priorityCount[task.Priority]++
		complexityCount[task.Complexity.Level]++
	}
	
	// Generate recommendations based on analysis
	if typeCount[types.TaskTypeTesting] < len(tasks)/4 {
		recommendations = append(recommendations, "Consider adding more testing tasks to ensure quality")
	}
	
	if typeCount[types.TaskTypeDocumentation] < len(tasks)/10 {
		recommendations = append(recommendations, "Documentation tasks may be needed for maintainability")
	}
	
	if priorityCount[types.TaskPriorityHigh] > len(tasks)/2 {
		recommendations = append(recommendations, "High number of high-priority tasks - consider prioritization review")
	}
	
	if complexityCount[types.ComplexityVeryComplex] > len(tasks)/3 {
		recommendations = append(recommendations, "Many complex tasks detected - consider breaking down further")
	}
	
	// Context-based recommendations
	if req.Context.TeamSize > 0 && len(tasks) < req.Context.TeamSize*2 {
		recommendations = append(recommendations, "Task count may be low for team size - consider more granular breakdown")
	}
	
	return recommendations
}

// generateNextSteps generates next steps based on tasks and project state
func (g *Generator) generateNextSteps(tasks []types.Task, projectState *types.ProjectState) []string {
	nextSteps := []string{}
	
	if len(tasks) > 0 {
		nextSteps = append(nextSteps, "Review and prioritize generated tasks")
		nextSteps = append(nextSteps, "Assign tasks to team members based on skills and availability")
		nextSteps = append(nextSteps, "Create project timeline and sprint planning")
	}
	
	// Find tasks with no dependencies (can start immediately)
	readyTasks := 0
	for _, task := range tasks {
		if len(task.Dependencies) == 0 {
			readyTasks++
		}
	}
	
	if readyTasks > 0 {
		nextSteps = append(nextSteps, fmt.Sprintf("Start with %d tasks that have no dependencies", readyTasks))
	}
	
	// Project state specific recommendations
	if projectState != nil {
		switch projectState.Phase {
		case types.PhaseRequirements:
			nextSteps = append(nextSteps, "Complete requirements analysis before implementation tasks")
		case types.PhaseDesign:
			nextSteps = append(nextSteps, "Finalize design decisions before development")
		case types.PhaseDevelopment:
			nextSteps = append(nextSteps, "Focus on implementation and testing tasks")
		}
		
		if len(projectState.CurrentBottlenecks) > 0 {
			nextSteps = append(nextSteps, "Address current bottlenecks before starting new tasks")
		}
	}
	
	return nextSteps
}

// validateRequest validates the task suggestion request
func (g *Generator) validateRequest(req *types.TaskSuggestionRequest) error {
	if req.PRDID == "" && req.PRDContent == "" {
		return fmt.Errorf("either PRD ID or PRD content must be provided")
	}
	
	if req.Options.MaxTasks <= 0 {
		return fmt.Errorf("max tasks must be positive")
	}
	
	if req.Options.MaxTasks > g.config.MaxTasksPerRequest {
		return fmt.Errorf("max tasks exceeds limit of %d", g.config.MaxTasksPerRequest)
	}
	
	if req.Options.MinQualityScore < 0 || req.Options.MinQualityScore > 1 {
		return fmt.Errorf("min quality score must be between 0 and 1")
	}
	
	return nil
}