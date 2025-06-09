// Package handlers provides HTTP request handlers for task generation and suggestions.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"lerian-mcp-memory/internal/ai"
	"lerian-mcp-memory/internal/api/response"
	"lerian-mcp-memory/internal/tasks"
	"lerian-mcp-memory/pkg/types"
)

// TaskHandler handles task generation and suggestion requests
type TaskHandler struct {
	generator  *tasks.Generator
	suggester  *tasks.Suggester
	aiService  tasks.AIService
	prdStorage PRDStorage
	config     TaskHandlerConfig
}

// TaskHandlerConfig represents configuration for task handler
type TaskHandlerConfig struct {
	MaxTasksPerRequest      int           `json:"max_tasks_per_request"`
	DefaultQualityThreshold float64       `json:"default_quality_threshold"`
	RequestTimeout          time.Duration `json:"request_timeout"`
	EnableSuggestions       bool          `json:"enable_suggestions"`
	EnableTemplates         bool          `json:"enable_templates"`
	DefaultAIModel          string        `json:"default_ai_model"`
}

// DefaultTaskHandlerConfig returns default configuration
func DefaultTaskHandlerConfig() TaskHandlerConfig {
	return TaskHandlerConfig{
		MaxTasksPerRequest:      50,
		DefaultQualityThreshold: 0.7,
		RequestTimeout:          120 * time.Second,
		EnableSuggestions:       true,
		EnableTemplates:         true,
		DefaultAIModel:          string(ai.ModelClaude),
	}
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(aiService tasks.AIService, prdStorage PRDStorage, config TaskHandlerConfig) *TaskHandler {
	generatorConfig := tasks.DefaultGeneratorConfig()
	generatorConfig.MaxTasksPerRequest = config.MaxTasksPerRequest
	generatorConfig.QualityThreshold = config.DefaultQualityThreshold
	generatorConfig.GenerationTimeout = config.RequestTimeout
	generatorConfig.EnableTemplates = config.EnableTemplates

	generator := tasks.NewGenerator(aiService, generatorConfig)
	suggester := tasks.NewSuggester()

	return &TaskHandler{
		generator:  generator,
		suggester:  suggester,
		aiService:  aiService,
		prdStorage: prdStorage,
		config:     config,
	}
}

// SuggestTasks handles task suggestion requests
func (h *TaskHandler) SuggestTasks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.config.RequestTimeout)
	defer cancel()

	// Parse request
	var req types.TaskSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Validate request
	if err := h.validateSuggestionRequest(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid request", err.Error())
		return
	}

	// Set defaults
	h.setRequestDefaults(&req)

	// Get PRD content if PRD ID is provided
	if req.PRDID != "" && req.PRDContent == "" {
		prdDoc, err := h.prdStorage.Get(ctx, req.PRDID)
		if err != nil {
			response.WriteError(w, http.StatusNotFound, "PRD not found", err.Error())
			return
		}
		req.PRDContent = prdDoc.Content.Raw
	}

	// Generate tasks using AI
	taskResponse, err := h.generator.GenerateTasks(ctx, &req)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to generate tasks", err.Error())
		return
	}

	// Generate contextual suggestions if enabled
	var suggestions []tasks.TaskSuggestion
	if h.config.EnableSuggestions && req.ProjectState.Phase != "" {
		suggestions, err = h.suggester.SuggestTasks(ctx, &req.ProjectState, req.ExistingTasks, &req.Context)
		if err != nil {
			// Log error but don't fail the request
			log.Printf("Failed to generate task suggestions: %v", err)
			// suggestions will remain empty
		}
	}

	// Create enhanced response
	enhancedResponse := TaskSuggestionEnhancedResponse{
		TaskSuggestionResponse: *taskResponse,
		ContextualSuggestions:  suggestions,
		RequestMetadata: RequestMetadata{
			RequestID:      r.Header.Get("X-Request-ID"),
			ProcessedAt:    time.Now(),
			ProcessingTime: taskResponse.GenerationMetadata.GenerationTime,
			Configuration:  h.getConfigSummary(),
		},
	}

	response.WriteSuccess(w, enhancedResponse)
}

// GenerateFromPRD handles task generation specifically from PRD documents
func (h *TaskHandler) GenerateFromPRD(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.config.RequestTimeout)
	defer cancel()

	// Parse request
	var req PRDTaskGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Validate request
	if req.PRDID == "" {
		response.WriteError(w, http.StatusBadRequest, "Missing PRD ID", "PRD ID is required")
		return
	}

	// Get PRD document
	prdDoc, err := h.prdStorage.Get(ctx, req.PRDID)
	if err != nil {
		response.WriteError(w, http.StatusNotFound, "PRD not found", err.Error())
		return
	}

	// Create task suggestion request from PRD
	taskReq := h.createTaskRequestFromPRD(prdDoc, &req)

	// Set project state if provided
	if req.ProjectState != nil {
		taskReq.ProjectState = *req.ProjectState
	}

	// Generate tasks
	taskResponse, err := h.generator.GenerateTasks(ctx, taskReq)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to generate tasks", err.Error())
		return
	}

	// Create PRD-specific response
	prdResponse := PRDTaskGenerationResponse{
		TaskSuggestionResponse: *taskResponse,
		PRDID:                  req.PRDID,
		PRDMetadata: PRDMetadata{
			Title:        prdDoc.Name,
			Status:       string(prdDoc.Status),
			WordCount:    prdDoc.Content.WordCount,
			Sections:     len(prdDoc.Content.Sections),
			QualityScore: prdDoc.Analysis.QualityScore,
		},
		TaskBreakdown: h.analyzeTaskBreakdown(taskResponse.Tasks),
		Insights:      h.generatePRDInsights(prdDoc, taskResponse.Tasks),
	}

	response.WriteSuccess(w, prdResponse)
}

// GetTaskSuggestions handles contextual task suggestions based on project state
func (h *TaskHandler) GetTaskSuggestions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.config.RequestTimeout)
	defer cancel()

	// Parse request
	var req ContextualSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Validate request
	if req.ProjectState.Phase == "" {
		response.WriteError(w, http.StatusBadRequest, "Missing project phase", "Project phase is required")
		return
	}

	// Generate suggestions
	suggestions, err := h.suggester.SuggestTasks(ctx, &req.ProjectState, req.ExistingTasks, &req.Context)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "Failed to generate suggestions", err.Error())
		return
	}

	// Create response
	suggestionResponse := ContextualSuggestionResponse{
		Suggestions:              suggestions,
		TotalSuggestions:         len(suggestions),
		ProjectPhase:             req.ProjectState.Phase,
		SuggestionCategories:     h.categorizeSuggestions(suggestions),
		NextPhaseRecommendations: h.getNextPhaseRecommendations(&req.ProjectState),
		RequestMetadata: RequestMetadata{
			RequestID:      r.Header.Get("X-Request-ID"),
			ProcessedAt:    time.Now(),
			ProcessingTime: time.Since(time.Now()), // Will be very small for suggestions
			Configuration:  h.getConfigSummary(),
		},
	}

	response.WriteSuccess(w, suggestionResponse)
}

// ValidateTask handles task validation requests
func (h *TaskHandler) ValidateTask(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var task types.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Create validator and validate
	validator := tasks.NewValidator()
	validationResult := validator.ValidateTask(&task)

	// Create response
	validationResponse := TaskValidationResponse{
		TaskID:           task.ID,
		ValidationResult: validationResult,
		ValidatedAt:      time.Now(),
		Recommendations:  h.generateValidationRecommendations(&validationResult),
	}

	response.WriteSuccess(w, validationResponse)
}

// ScoreTask handles task quality scoring requests
func (h *TaskHandler) ScoreTask(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req TaskScoringRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "Invalid JSON request", err.Error())
		return
	}

	// Create scorer and calculate score
	scorer := tasks.NewScorer()
	qualityScore := scorer.ScoreTask(&req.Task, &req.Context)

	// Create response
	scoreResponse := TaskScoringResponse{
		TaskID:       req.Task.ID,
		QualityScore: qualityScore,
		QualityLevel: scorer.GetQualityLevel(qualityScore.OverallScore),
		ScoredAt:     time.Now(),
		Insights:     h.generateScoringInsights(&qualityScore),
	}

	response.WriteSuccess(w, scoreResponse)
}

// Helper functions

// validateSuggestionRequest validates the task suggestion request
func (h *TaskHandler) validateSuggestionRequest(req *types.TaskSuggestionRequest) error {
	if req.PRDID == "" && req.PRDContent == "" {
		return fmt.Errorf("either PRD ID or PRD content must be provided")
	}

	if req.Options.MaxTasks <= 0 {
		return fmt.Errorf("max tasks must be positive")
	}

	if req.Options.MaxTasks > h.config.MaxTasksPerRequest {
		return fmt.Errorf("max tasks exceeds limit of %d", h.config.MaxTasksPerRequest)
	}

	if req.Options.MinQualityScore < 0 || req.Options.MinQualityScore > 1 {
		return fmt.Errorf("min quality score must be between 0 and 1")
	}

	return nil
}

// setRequestDefaults sets default values for the request
func (h *TaskHandler) setRequestDefaults(req *types.TaskSuggestionRequest) {
	if req.Options.MaxTasks == 0 {
		req.Options.MaxTasks = 20
	}

	if req.Options.MinQualityScore == 0 {
		req.Options.MinQualityScore = h.config.DefaultQualityThreshold
	}

	if req.Options.AIModel == "" {
		req.Options.AIModel = h.config.DefaultAIModel
	}

	if req.Options.GenerationStyle == "" {
		req.Options.GenerationStyle = types.GenerationStyleAgile
	}

	// Set default options
	req.Options.IncludeEstimation = true
	req.Options.IncludeDependencies = true
	req.Options.IncludeAcceptanceCriteria = true
	req.Options.UseTemplates = h.config.EnableTemplates
}

// createTaskRequestFromPRD creates a task request from PRD document
func (h *TaskHandler) createTaskRequestFromPRD(prdDoc *types.PRDDocument, req *PRDTaskGenerationRequest) *types.TaskSuggestionRequest {
	return &types.TaskSuggestionRequest{
		PRDID:      req.PRDID,
		PRDContent: prdDoc.Content.Raw,
		Context: types.TaskGenerationContext{
			ProjectName: prdDoc.Name,
			ProjectType: prdDoc.Metadata.ProjectType,
			TechStack:   prdDoc.Metadata.Technology,
			Timeline:    req.Timeline,
			Budget:      req.Budget,
			Constraints: req.Constraints,
			Repository:  req.Repository,
		},
		Options: types.TaskGenerationOptions{
			MaxTasks:                  req.MaxTasks,
			MinQualityScore:           req.MinQualityScore,
			IncludeEstimation:         true,
			IncludeDependencies:       true,
			IncludeAcceptanceCriteria: true,
			TaskTypes:                 req.TaskTypes,
			AIModel:                   req.AIModel,
			UseTemplates:              h.config.EnableTemplates,
			GenerationStyle:           req.GenerationStyle,
		},
		ExistingTasks: req.ExistingTasks,
	}
}

// analyzeTaskBreakdown analyzes the breakdown of generated tasks
func (h *TaskHandler) analyzeTaskBreakdown(taskList []types.Task) TaskBreakdown {
	breakdown := TaskBreakdown{
		TotalTasks:   len(taskList),
		ByType:       make(map[string]int),
		ByPriority:   make(map[string]int),
		ByComplexity: make(map[string]int),
	}

	totalEffort := 0.0
	for i := range taskList {
		task := &taskList[i]
		// Count by type
		breakdown.ByType[string(task.Type)]++

		// Count by priority
		breakdown.ByPriority[string(task.Priority)]++

		// Count by complexity
		breakdown.ByComplexity[string(task.Complexity.Level)]++

		// Sum effort
		totalEffort += task.EstimatedEffort.Hours
	}

	breakdown.TotalEstimatedHours = totalEffort
	breakdown.AverageTaskSize = totalEffort / float64(len(taskList))

	return breakdown
}

// generatePRDInsights generates insights from PRD analysis and task generation
func (h *TaskHandler) generatePRDInsights(prdDoc *types.PRDDocument, taskList []types.Task) []string {
	insights := []string{}

	// PRD quality insights
	if prdDoc.Analysis.QualityScore < 0.7 {
		insights = append(insights, "PRD quality score is below recommended threshold - consider improving documentation")
	}

	// Task complexity insights
	complexTasks := 0
	for i := range taskList {
		task := &taskList[i]
		if task.Complexity.Level == types.ComplexityComplex || task.Complexity.Level == types.ComplexityVeryComplex {
			complexTasks++
		}
	}

	if float64(complexTasks)/float64(len(taskList)) > 0.3 {
		insights = append(insights, "High proportion of complex tasks - consider breaking down into smaller components")
	}

	// Missing elements insights
	if len(prdDoc.Analysis.MissingElements) > 0 {
		insights = append(insights, fmt.Sprintf("PRD missing key elements: %v", prdDoc.Analysis.MissingElements))
	}

	// Task type distribution insights
	typeCount := make(map[types.TaskType]int)
	for i := range taskList {
		typeCount[taskList[i].Type]++
	}

	if typeCount[types.TaskTypeLegacyTesting] == 0 {
		insights = append(insights, "No testing tasks generated - consider adding quality assurance tasks")
	}

	if typeCount[types.TaskTypeLegacyDocumentation] == 0 {
		insights = append(insights, "No documentation tasks generated - consider adding technical documentation")
	}

	return insights
}

// categorizeSuggestions categorizes suggestions by type
func (h *TaskHandler) categorizeSuggestions(suggestions []tasks.TaskSuggestion) map[string]int {
	categories := make(map[string]int)
	for i := range suggestions {
		categories[string(suggestions[i].Category)]++
	}
	return categories
}

// getNextPhaseRecommendations provides recommendations for moving to next phase
func (h *TaskHandler) getNextPhaseRecommendations(projectState *types.ProjectState) []string {
	recommendations := []string{}

	switch projectState.Phase {
	case types.PhaseDiscovery:
		recommendations = append(recommendations, "Complete user research and market analysis", "Define clear success metrics and KPIs")
	case types.PhaseRequirements:
		recommendations = append(recommendations, "Create comprehensive PRD with user stories", "Validate requirements with stakeholders")
	case types.PhaseDesign:
		recommendations = append(recommendations, "Finalize system architecture and technical decisions", "Create detailed UI/UX designs and prototypes")
	case types.PhaseDevelopment:
		recommendations = append(recommendations, "Ensure adequate test coverage and quality gates", "Prepare deployment and monitoring infrastructure")
	case types.PhaseTesting:
		recommendations = append(recommendations, "Complete comprehensive testing across all scenarios", "Prepare rollback and incident response procedures")
	case types.PhaseDeployment:
		recommendations = append(recommendations, "Monitor system performance and user feedback", "Plan for ongoing maintenance and improvements")
	}

	return recommendations
}

// generateValidationRecommendations generates recommendations from validation results
func (h *TaskHandler) generateValidationRecommendations(result *types.TaskValidationResult) []string {
	recommendations := []string{}

	if len(result.Errors) > 0 {
		recommendations = append(recommendations, "Fix all validation errors before proceeding")
	}

	if len(result.Warnings) > 0 {
		recommendations = append(recommendations, "Address validation warnings to improve task quality")
	}

	if result.Score < 0.7 {
		recommendations = append(recommendations, "Task quality is below recommended threshold", "Add more specific acceptance criteria and technical details")
	}

	return recommendations
}

// generateScoringInsights generates insights from quality scoring
func (h *TaskHandler) generateScoringInsights(score *types.QualityScore) []string {
	insights := []string{}

	if score.Clarity < 0.7 {
		insights = append(insights, "Task description could be clearer and more understandable")
	}

	if score.Actionability < 0.7 {
		insights = append(insights, "Task should be more actionable with specific deliverables")
	}

	if score.Testability < 0.7 {
		insights = append(insights, "Task outcomes should be more measurable and testable")
	}

	if score.Specificity < 0.7 {
		insights = append(insights, "Task needs more specific technical details and requirements")
	}

	return insights
}

// getConfigSummary returns a summary of handler configuration
func (h *TaskHandler) getConfigSummary() map[string]interface{} {
	return map[string]interface{}{
		"max_tasks_per_request":     h.config.MaxTasksPerRequest,
		"default_quality_threshold": h.config.DefaultQualityThreshold,
		"suggestions_enabled":       h.config.EnableSuggestions,
		"templates_enabled":         h.config.EnableTemplates,
		"default_ai_model":          h.config.DefaultAIModel,
	}
}

// Response types

// TaskSuggestionEnhancedResponse represents enhanced task suggestion response
type TaskSuggestionEnhancedResponse struct {
	types.TaskSuggestionResponse
	ContextualSuggestions []tasks.TaskSuggestion `json:"contextual_suggestions"`
	RequestMetadata       RequestMetadata        `json:"request_metadata"`
}

// PRDTaskGenerationRequest represents request for PRD-specific task generation
type PRDTaskGenerationRequest struct {
	PRDID           string                `json:"prd_id"`
	MaxTasks        int                   `json:"max_tasks"`
	MinQualityScore float64               `json:"min_quality_score"`
	TaskTypes       []types.TaskType      `json:"task_types,omitempty"`
	GenerationStyle types.GenerationStyle `json:"generation_style"`
	AIModel         string                `json:"ai_model,omitempty"`
	Timeline        string                `json:"timeline,omitempty"`
	Budget          string                `json:"budget,omitempty"`
	Constraints     []string              `json:"constraints,omitempty"`
	Repository      string                `json:"repository,omitempty"`
	ExistingTasks   []types.Task          `json:"existing_tasks,omitempty"`
	ProjectState    *types.ProjectState   `json:"project_state,omitempty"`
}

// PRDTaskGenerationResponse represents PRD-specific task generation response
type PRDTaskGenerationResponse struct {
	types.TaskSuggestionResponse
	PRDID         string        `json:"prd_id"`
	PRDMetadata   PRDMetadata   `json:"prd_metadata"`
	TaskBreakdown TaskBreakdown `json:"task_breakdown"`
	Insights      []string      `json:"insights"`
}

// PRDMetadata represents metadata about the PRD
type PRDMetadata struct {
	Title        string  `json:"title"`
	Status       string  `json:"status"`
	WordCount    int     `json:"word_count"`
	Sections     int     `json:"sections"`
	QualityScore float64 `json:"quality_score"`
}

// TaskBreakdown represents analysis of task breakdown
type TaskBreakdown struct {
	TotalTasks          int            `json:"total_tasks"`
	TotalEstimatedHours float64        `json:"total_estimated_hours"`
	AverageTaskSize     float64        `json:"average_task_size"`
	ByType              map[string]int `json:"by_type"`
	ByPriority          map[string]int `json:"by_priority"`
	ByComplexity        map[string]int `json:"by_complexity"`
}

// ContextualSuggestionRequest represents request for contextual suggestions
type ContextualSuggestionRequest struct {
	ProjectState  types.ProjectState          `json:"project_state"`
	ExistingTasks []types.Task                `json:"existing_tasks"`
	Context       types.TaskGenerationContext `json:"context"`
}

// ContextualSuggestionResponse represents contextual suggestion response
type ContextualSuggestionResponse struct {
	Suggestions              []tasks.TaskSuggestion `json:"suggestions"`
	TotalSuggestions         int                    `json:"total_suggestions"`
	ProjectPhase             types.ProjectPhase     `json:"project_phase"`
	SuggestionCategories     map[string]int         `json:"suggestion_categories"`
	NextPhaseRecommendations []string               `json:"next_phase_recommendations"`
	RequestMetadata          RequestMetadata        `json:"request_metadata"`
}

// TaskValidationResponse represents task validation response
type TaskValidationResponse struct {
	TaskID           string                     `json:"task_id"`
	ValidationResult types.TaskValidationResult `json:"validation_result"`
	ValidatedAt      time.Time                  `json:"validated_at"`
	Recommendations  []string                   `json:"recommendations"`
}

// TaskScoringRequest represents task scoring request
type TaskScoringRequest struct {
	Task    types.Task                  `json:"task"`
	Context types.TaskGenerationContext `json:"context"`
}

// TaskScoringResponse represents task scoring response
type TaskScoringResponse struct {
	TaskID       string             `json:"task_id"`
	QualityScore types.QualityScore `json:"quality_score"`
	QualityLevel string             `json:"quality_level"`
	ScoredAt     time.Time          `json:"scored_at"`
	Insights     []string           `json:"insights"`
}

// RequestMetadata represents metadata about the request processing
type RequestMetadata struct {
	RequestID      string                 `json:"request_id"`
	ProcessedAt    time.Time              `json:"processed_at"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Configuration  map[string]interface{} `json:"configuration"`
}
