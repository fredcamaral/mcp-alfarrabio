// Package ai provides AI-powered task generation from PRDs and TRDs.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"lerian-mcp-memory/internal/documents"

	"github.com/google/uuid"
)

// AITaskGenerator handles AI-powered task generation
type AITaskGenerator struct {
	aiService   *Service
	ruleManager *documents.RuleManager
	logger      *slog.Logger
}

// NewAITaskGenerator creates a new AI-powered task generator
func NewAITaskGenerator(aiService *Service, ruleManager *documents.RuleManager, logger *slog.Logger) *AITaskGenerator {
	return &AITaskGenerator{
		aiService:   aiService,
		ruleManager: ruleManager,
		logger:      logger,
	}
}

// TaskGenerationOptions contains options for task generation
type TaskGenerationOptions struct {
	MaxMainTasks       int               `json:"max_main_tasks"`
	MaxSubTasksPerMain int               `json:"max_sub_tasks_per_main"`
	MaxSubTaskHours    int               `json:"max_sub_task_hours"`
	TeamSize           int               `json:"team_size"`
	ExperienceLevel    string            `json:"experience_level"` // junior, mid, senior
	PreferredPhases    []string          `json:"preferred_phases,omitempty"`
	IncludeTestTasks   bool              `json:"include_test_tasks"`
	IncludeDocTasks    bool              `json:"include_doc_tasks"`
	CustomConstraints  map[string]string `json:"custom_constraints,omitempty"`
	ProjectContext     string            `json:"project_context,omitempty"`
}

// GenerateMainTasksFromTRD generates main tasks from a TRD using AI
func (g *AITaskGenerator) GenerateMainTasksFromTRD(ctx context.Context, trd *documents.TRDEntity, prd *documents.PRDEntity, options TaskGenerationOptions) ([]*documents.MainTask, error) {
	if trd == nil {
		return nil, fmt.Errorf("TRD cannot be nil")
	}

	g.logger.Info("generating main tasks from TRD using AI",
		slog.String("trd_id", trd.ID),
		slog.String("trd_title", trd.Title))

	// Get task generation rules
	ruleContent, err := g.getRuleContent(documents.RuleTaskGeneration)
	if err != nil {
		return nil, fmt.Errorf("failed to get task generation rules: %w", err)
	}

	// Build AI prompt for main tasks
	prompt := g.buildMainTaskPrompt(trd, prd, ruleContent, options)

	// Create AI request
	req := &Request{
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert project manager and technical lead breaking down complex projects into manageable development tasks.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Metadata: RequestMetadata{
			Repository: trd.Repository,
			Tags:       []string{"task_generation", "main_tasks"},
		},
	}

	// Process with AI service
	startTime := time.Now()
	resp, err := g.aiService.ProcessRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI main task generation failed: %w", err)
	}
	duration := time.Since(startTime)

	g.logger.Info("AI main task generation completed",
		slog.Duration("duration", duration),
		slog.String("model", string(resp.Model)),
		slog.Int("tokens", resp.TokensUsed.Total))

	// Parse AI response into main tasks
	mainTasks, err := g.parseMainTasksResponse(resp.Content, trd, prd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse main tasks response: %w", err)
	}

	// Validate and enhance main tasks
	for i, task := range mainTasks {
		task.TaskID = fmt.Sprintf("MT-%03d", i+1)
		if err := task.Validate(); err != nil {
			return nil, fmt.Errorf("main task validation failed: %w", err)
		}
	}

	g.logger.Info("generated main tasks from TRD",
		slog.Int("task_count", len(mainTasks)),
		slog.String("trd_id", trd.ID))

	return mainTasks, nil
}

// GenerateSubTasksFromMainTask generates sub-tasks from a main task using AI
func (g *AITaskGenerator) GenerateSubTasksFromMainTask(ctx context.Context, mainTask *documents.MainTask, trd *documents.TRDEntity, options TaskGenerationOptions) ([]*documents.SubTask, error) {
	if mainTask == nil {
		return nil, fmt.Errorf("main task cannot be nil")
	}

	g.logger.Info("generating sub-tasks from main task using AI",
		slog.String("main_task_id", mainTask.ID),
		slog.String("main_task_name", mainTask.Name))

	// Get sub-task generation rules
	ruleContent, err := g.getRuleContent(documents.RuleSubTaskGeneration)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub-task generation rules: %w", err)
	}

	// Build AI prompt for sub-tasks
	prompt := g.buildSubTaskPrompt(mainTask, trd, ruleContent, options)

	// Create AI request
	req := &Request{
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert technical lead breaking down development tasks into specific, actionable sub-tasks for developers.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Metadata: RequestMetadata{
			Repository: mainTask.Repository,
			Tags:       []string{"task_generation", "sub_tasks"},
		},
	}

	// Process with AI service
	startTime := time.Now()
	resp, err := g.aiService.ProcessRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI sub-task generation failed: %w", err)
	}
	duration := time.Since(startTime)

	g.logger.Info("AI sub-task generation completed",
		slog.Duration("duration", duration),
		slog.String("model", string(resp.Model)),
		slog.Int("tokens", resp.TokensUsed.Total))

	// Parse AI response into sub-tasks
	subTasks, err := g.parseSubTasksResponse(resp.Content, mainTask, options)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sub-tasks response: %w", err)
	}

	// Validate and enhance sub-tasks
	for i, task := range subTasks {
		task.SubTaskID = fmt.Sprintf("ST-%s-%03d", mainTask.TaskID, i+1)
		task.MainTaskID = mainTask.ID
		if err := task.Validate(); err != nil {
			return nil, fmt.Errorf("sub-task validation failed: %w", err)
		}
	}

	g.logger.Info("generated sub-tasks from main task",
		slog.Int("sub_task_count", len(subTasks)),
		slog.String("main_task_id", mainTask.ID))

	return subTasks, nil
}

// buildMainTaskPrompt builds the AI prompt for main task generation
func (g *AITaskGenerator) buildMainTaskPrompt(trd *documents.TRDEntity, prd *documents.PRDEntity, ruleContent string, options TaskGenerationOptions) string {
	prompt := ruleContent + "\n\n"

	// Add TRD context
	prompt += fmt.Sprintf(`Generate main tasks based on this Technical Requirements Document:

TRD Title: %s
TRD Content:
%s

Technical Stack: %v
Architecture: %s
Dependencies: %v

`, trd.Title, trd.Content, trd.TechnicalStack, trd.Architecture, trd.Dependencies)

	// Add PRD context if available
	if prd != nil {
		prompt += fmt.Sprintf(`Related PRD Context:
Goals: %v
Requirements: %v
User Stories: %v
Constraints: %v

`, prd.ParsedContent.Goals,
			prd.ParsedContent.Requirements,
			prd.ParsedContent.UserStories,
			prd.ParsedContent.Constraints)
	}

	// Add generation options
	prompt += fmt.Sprintf(`Generation Parameters:
- Maximum main tasks: %d
- Team size: %d
- Experience level: %s
- Include testing tasks: %t
- Include documentation tasks: %t

`, options.MaxMainTasks, options.TeamSize, options.ExperienceLevel, options.IncludeTestTasks, options.IncludeDocTasks)

	// Add preferred phases if specified
	if len(options.PreferredPhases) > 0 {
		prompt += fmt.Sprintf("Preferred development phases: %s\n", strings.Join(options.PreferredPhases, ", "))
	}

	// Add custom constraints
	if len(options.CustomConstraints) > 0 {
		prompt += "Custom constraints:\n"
		for key, value := range options.CustomConstraints {
			prompt += fmt.Sprintf("- %s: %s\n", key, value)
		}
	}

	// Generation instructions
	prompt += `
Generate 5-7 main tasks that are:
1. Atomic and deployable phases (each task should result in working, deployable code)
2. Properly sequenced with clear dependencies
3. Sized for 1-2 weeks of work (considering team size and experience level)
4. Aligned with the technical architecture specified in the TRD
5. Following development best practices

Return as JSON array with this structure:
{
	"main_tasks": [
		{
			"name": "descriptive task name",
			"description": "detailed description of what needs to be accomplished",
			"phase": "development phase (setup, foundation, core, integration, testing, deployment)",
			"duration_estimate": "1-2 weeks",
			"dependencies": ["list of dependent task names or IDs"],
			"deliverables": ["list of specific deliverables"],
			"acceptance_criteria": ["list of acceptance criteria"],
			"complexity_score": 1-10,
			"technical_requirements": ["list of technical requirements"],
			"skills_required": ["list of required skills"]
		}
	],
	"task_sequence": {
		"parallel_groups": [["tasks that can be done in parallel"]],
		"critical_path": ["sequence of tasks on critical path"]
	},
	"estimates": {
		"total_duration_weeks": number,
		"team_allocation": "recommended team structure"
	}
}`

	return prompt
}

// buildSubTaskPrompt builds the AI prompt for sub-task generation
func (g *AITaskGenerator) buildSubTaskPrompt(mainTask *documents.MainTask, trd *documents.TRDEntity, ruleContent string, options TaskGenerationOptions) string {
	prompt := ruleContent + "\n\n"

	// Add main task context
	prompt += fmt.Sprintf(`Break down this main task into specific sub-tasks:

Main Task: %s
Description: %s
Phase: %s
Duration Estimate: %s
Deliverables: %v
Acceptance Criteria: %v

`, mainTask.Name, mainTask.Description, mainTask.Phase, mainTask.DurationEstimate, mainTask.Deliverables, mainTask.AcceptanceCriteria)

	// Add TRD context for technical details
	prompt += fmt.Sprintf(`Technical Context (from TRD):
Technical Stack: %v
Architecture: %s

`, trd.TechnicalStack, trd.Architecture)

	// Add generation options
	prompt += fmt.Sprintf(`Generation Parameters:
- Maximum sub-tasks: %d
- Maximum hours per sub-task: %d
- Team experience level: %s

`, options.MaxSubTasksPerMain, options.MaxSubTaskHours, options.ExperienceLevel)

	// Generation instructions
	prompt += `
Generate 3-6 sub-tasks that are:
1. Specific and actionable (a developer knows exactly what to do)
2. Sized for 2-8 hours of work (considering experience level)
3. Properly sequenced within the main task
4. Include clear acceptance criteria
5. Specify implementation type (code, configuration, documentation, testing)

Return as JSON array with this structure:
{
	"sub_tasks": [
		{
			"name": "specific sub-task name",
			"description": "detailed description of the work to be done",
			"estimated_hours": 2-8,
			"implementation_type": "code|config|docs|testing|research",
			"dependencies": ["list of dependent sub-task names"],
			"acceptance_criteria": ["specific criteria for completion"],
			"technical_details": {
				"files_to_modify": ["list of files"],
				"new_files_to_create": ["list of new files"],
				"apis_to_implement": ["list of APIs"],
				"tests_to_write": ["list of tests"]
			},
			"skills_required": ["specific skills needed"],
			"difficulty": "junior|mid|senior"
		}
	],
	"implementation_order": ["recommended order of sub-tasks"],
	"testing_strategy": "how to test the main task completion"
}`

	return prompt
}

// parseMainTasksResponse parses the AI response into MainTask entities
func (g *AITaskGenerator) parseMainTasksResponse(content string, trd *documents.TRDEntity, prd *documents.PRDEntity) ([]*documents.MainTask, error) {
	var response struct {
		MainTasks []struct {
			Name                  string   `json:"name"`
			Description           string   `json:"description"`
			Phase                 string   `json:"phase"`
			DurationEstimate      string   `json:"duration_estimate"`
			Dependencies          []string `json:"dependencies"`
			Deliverables          []string `json:"deliverables"`
			AcceptanceCriteria    []string `json:"acceptance_criteria"`
			ComplexityScore       int      `json:"complexity_score"`
			TechnicalRequirements []string `json:"technical_requirements"`
			SkillsRequired        []string `json:"skills_required"`
		} `json:"main_tasks"`
		TaskSequence struct {
			ParallelGroups [][]string `json:"parallel_groups"`
			CriticalPath   []string   `json:"critical_path"`
		} `json:"task_sequence"`
		Estimates struct {
			TotalDurationWeeks int    `json:"total_duration_weeks"`
			TeamAllocation     string `json:"team_allocation"`
		} `json:"estimates"`
	}

	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	var mainTasks []*documents.MainTask
	for _, taskData := range response.MainTasks {
		task := &documents.MainTask{
			ID:                 uuid.New().String(),
			TRDID:              trd.ID,
			Name:               taskData.Name,
			Description:        taskData.Description,
			Phase:              taskData.Phase,
			DurationEstimate:   taskData.DurationEstimate,
			Dependencies:       taskData.Dependencies,
			Deliverables:       taskData.Deliverables,
			AcceptanceCriteria: taskData.AcceptanceCriteria,
			ComplexityScore:    taskData.ComplexityScore,
			Status:             documents.StatusDraft,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
			Repository:         trd.Repository,
		}

		if prd != nil {
			task.PRDID = prd.ID
		}

		mainTasks = append(mainTasks, task)
	}

	return mainTasks, nil
}

// parseSubTasksResponse parses the AI response into SubTask entities
func (g *AITaskGenerator) parseSubTasksResponse(content string, mainTask *documents.MainTask, options TaskGenerationOptions) ([]*documents.SubTask, error) {
	var response struct {
		SubTasks []struct {
			Name               string                 `json:"name"`
			Description        string                 `json:"description"`
			EstimatedHours     int                    `json:"estimated_hours"`
			ImplementationType string                 `json:"implementation_type"`
			Dependencies       []string               `json:"dependencies"`
			AcceptanceCriteria []string               `json:"acceptance_criteria"`
			TechnicalDetails   map[string]interface{} `json:"technical_details"`
			SkillsRequired     []string               `json:"skills_required"`
			Difficulty         string                 `json:"difficulty"`
		} `json:"sub_tasks"`
		ImplementationOrder []string `json:"implementation_order"`
		TestingStrategy     string   `json:"testing_strategy"`
	}

	if err := json.Unmarshal([]byte(content), &response); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	var subTasks []*documents.SubTask
	for _, taskData := range response.SubTasks {
		// Convert technical details to string map
		techDetailsMap := make(map[string]string)
		for key, value := range taskData.TechnicalDetails {
			switch v := value.(type) {
			case string:
				techDetailsMap[key] = v
			case []interface{}:
				// Convert array to comma-separated string
				var items []string
				for _, item := range v {
					if str, ok := item.(string); ok {
						items = append(items, str)
					}
				}
				techDetailsMap[key] = strings.Join(items, ", ")
			default:
				techDetailsMap[key] = fmt.Sprintf("%v", v)
			}
		}

		// Ensure estimated hours is within constraints
		estimatedHours := taskData.EstimatedHours
		if estimatedHours < 1 {
			estimatedHours = 1
		}
		if estimatedHours > options.MaxSubTaskHours {
			estimatedHours = options.MaxSubTaskHours
		}

		task := &documents.SubTask{
			ID:                 uuid.New().String(),
			MainTaskID:         mainTask.ID,
			Name:               taskData.Name,
			Description:        taskData.Description,
			EstimatedHours:     estimatedHours,
			ImplementationType: taskData.ImplementationType,
			Dependencies:       taskData.Dependencies,
			AcceptanceCriteria: taskData.AcceptanceCriteria,
			TechnicalDetails:   techDetailsMap,
			Status:             documents.StatusDraft,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
			Repository:         mainTask.Repository,
		}

		subTasks = append(subTasks, task)
	}

	return subTasks, nil
}

// getRuleContent retrieves generation rules
func (g *AITaskGenerator) getRuleContent(ruleType documents.RuleType) (string, error) {
	rule, err := g.ruleManager.GetRuleContent(ruleType)
	if err != nil {
		// Return default rules if custom rules not found
		return g.getDefaultRules(ruleType), nil
	}
	return rule, nil
}

// getDefaultRules returns default generation rules
func (g *AITaskGenerator) getDefaultRules(ruleType documents.RuleType) string {
	switch ruleType {
	case documents.RuleTaskGeneration:
		return `Generate development tasks following these principles:

1. **Atomic Tasks**: Each main task should be deployable and demonstrate working functionality
2. **Proper Sizing**: Main tasks 1-2 weeks, sub-tasks 2-8 hours
3. **Clear Dependencies**: Explicit dependencies between tasks
4. **Testable Deliverables**: Each task produces testable, measurable output
5. **Technical Alignment**: Tasks align with specified architecture and tech stack

Focus on creating a logical development progression that builds toward the final product.`

	case documents.RuleSubTaskGeneration:
		return `Generate sub-tasks following these principles:

1. **Actionable**: Developer knows exactly what to implement
2. **Specific**: Clear scope and boundaries
3. **Testable**: Each sub-task has verifiable completion criteria
4. **Appropriately Sized**: 2-8 hours of work
5. **Implementation-Ready**: Includes technical details and file specifications

Break down implementation work into logical, sequential steps.`

	default:
		return "Generate tasks with clear objectives and acceptance criteria."
	}
}

// DefaultTaskGenerationOptions returns sensible defaults
func DefaultTaskGenerationOptions() TaskGenerationOptions {
	return TaskGenerationOptions{
		MaxMainTasks:       7,
		MaxSubTasksPerMain: 6,
		MaxSubTaskHours:    8,
		TeamSize:           3,
		ExperienceLevel:    "mid",
		IncludeTestTasks:   true,
		IncludeDocTasks:    true,
	}
}
