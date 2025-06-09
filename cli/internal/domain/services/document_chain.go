// Package services provides domain services for the lerian-mcp-memory CLI.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// DocumentChainService orchestrates the complete document generation chain
type DocumentChainService interface {
	// ExecuteFullChain runs the complete automation chain
	ExecuteFullChain(ctx context.Context, input string) (*ChainResult, error)

	// Individual steps
	GeneratePRDInteractive(ctx context.Context, context *GenerationContext) (*PRDEntity, error)
	GenerateTRDFromPRD(ctx context.Context, prd *PRDEntity) (*TRDEntity, error)
	GenerateMainTasksFromTRD(ctx context.Context, trd *TRDEntity) ([]*MainTask, error)
	GenerateSubTasksFromMain(ctx context.Context, mainTask *MainTask) ([]*SubTask, error)

	// Progress tracking
	GetChainProgress(chainID string) (*ChainProgress, error)
	ListChains() ([]*ChainProgress, error)
}

// ChainResult represents the complete output of a document chain execution
type ChainResult struct {
	ID        string         `json:"id"`
	PRD       *PRDEntity     `json:"prd,omitempty"`
	TRD       *TRDEntity     `json:"trd,omitempty"`
	MainTasks []*MainTask    `json:"main_tasks"`
	SubTasks  []*SubTask     `json:"sub_tasks"`
	Metadata  ChainMetadata  `json:"metadata"`
	Progress  *ChainProgress `json:"progress"`
}

// ChainMetadata contains metadata about the chain execution
type ChainMetadata struct {
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time,omitempty"`
	Duration       string    `json:"duration,omitempty"`
	Repository     string    `json:"repository"`
	TotalTasks     int       `json:"total_tasks"`
	GeneratedBy    string    `json:"generated_by"`
	ProjectType    string    `json:"project_type,omitempty"`
	UserInputCount int       `json:"user_input_count"`
}

// ChainProgress tracks the progress of a document chain execution
type ChainProgress struct {
	ChainID       string            `json:"chain_id"`
	Status        ChainStatus       `json:"status"`
	CurrentStep   ChainStep         `json:"current_step"`
	StepsComplete []ChainStep       `json:"steps_complete"`
	StepsFailed   []ChainStep       `json:"steps_failed"`
	Progress      float64           `json:"progress"` // 0.0 to 1.0
	StartTime     time.Time         `json:"start_time"`
	LastUpdate    time.Time         `json:"last_update"`
	EstimatedEnd  time.Time         `json:"estimated_end,omitempty"`
	ErrorMessage  string            `json:"error_message,omitempty"`
	Context       map[string]string `json:"context"`
}

// ChainStatus represents the current status of a chain execution
type ChainStatus string

const (
	ChainStatusPending   ChainStatus = "pending"
	ChainStatusRunning   ChainStatus = "running"
	ChainStatusCompleted ChainStatus = "completed"
	ChainStatusFailed    ChainStatus = "failed"
	ChainStatusPaused    ChainStatus = "paused"
)

// ChainStep represents individual steps in the document chain
type ChainStep string

const (
	ChainStepPRDGeneration      ChainStep = "prd_generation"
	ChainStepTRDGeneration      ChainStep = "trd_generation"
	ChainStepMainTaskGeneration ChainStep = "main_task_generation"
	ChainStepSubTaskGeneration  ChainStep = "sub_task_generation"
	ChainStepValidation         ChainStep = "validation"
	ChainStepPersistence        ChainStep = "persistence"
)

// GenerationContext provides context for document generation
type GenerationContext struct {
	Repository    string                 `json:"repository"`
	ProjectType   string                 `json:"project_type"`
	UserInputs    []string               `json:"user_inputs"`
	ExistingTasks []*entities.Task       `json:"existing_tasks,omitempty"`
	Templates     []*TaskTemplate        `json:"templates,omitempty"`
	UserPrefs     UserPreferences        `json:"user_preferences"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// UserPreferences contains user preferences for generation
type UserPreferences struct {
	PreferredTaskSize   string   `json:"preferred_task_size"`  // small, medium, large
	PreferredComplexity string   `json:"preferred_complexity"` // low, medium, high
	IncludeTests        bool     `json:"include_tests"`
	IncludeDocs         bool     `json:"include_docs"`
	FavoriteTemplates   []string `json:"favorite_templates,omitempty"`
	AvoidPatterns       []string `json:"avoid_patterns,omitempty"`
}

// Document entities for generation chain
type PRDEntity struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Features    []string               `json:"features"`
	UserStories []string               `json:"user_stories"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

type TRDEntity struct {
	ID             string                 `json:"id"`
	PRDID          string                 `json:"prd_id"`
	Title          string                 `json:"title"`
	Architecture   string                 `json:"architecture"`
	TechStack      []string               `json:"tech_stack"`
	Requirements   []string               `json:"requirements"`
	Implementation []string               `json:"implementation"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
}

type MainTask struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Phase            string    `json:"phase"`
	Duration         string    `json:"duration"`
	AtomicValidation bool      `json:"atomic_validation"`
	Dependencies     []string  `json:"dependencies"`
	SubTaskCount     int       `json:"sub_task_count"`
	Content          string    `json:"content"`
	CreatedAt        time.Time `json:"created_at"`
}

type SubTask struct {
	ID                 string    `json:"id"`
	ParentTaskID       string    `json:"parent_task_id"`
	Name               string    `json:"name"`
	Duration           int       `json:"duration_hours"`
	Type               string    `json:"implementation_type"`
	Deliverables       []string  `json:"deliverables"`
	AcceptanceCriteria []string  `json:"acceptance_criteria"`
	Dependencies       []string  `json:"dependencies"`
	Content            string    `json:"content"`
	CreatedAt          time.Time `json:"created_at"`
}

// TaskTemplate represents a template for common task patterns
type TaskTemplate struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Pattern     string   `json:"pattern"`
	Keywords    []string `json:"keywords"`
	Complexity  string   `json:"complexity"`
	Hours       int      `json:"estimated_hours"`
	SubTasks    []string `json:"subtask_templates"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
}

// DefaultDocumentChainService implements DocumentChainService
type DefaultDocumentChainService struct {
	mcpClient ports.MCPClient
	storage   ports.Storage
	logger    *slog.Logger
	chains    map[string]*ChainProgress
	templates []*TaskTemplate
}

// NewDocumentChainService creates a new document chain service
func NewDocumentChainService(
	mcpClient ports.MCPClient,
	storage ports.Storage,
	logger *slog.Logger,
) *DefaultDocumentChainService {
	return &DefaultDocumentChainService{
		mcpClient: mcpClient,
		storage:   storage,
		logger:    logger,
		chains:    make(map[string]*ChainProgress),
		templates: loadDefaultTemplates(),
	}
}

// ExecuteFullChain runs the complete document generation chain
func (s *DefaultDocumentChainService) ExecuteFullChain(ctx context.Context, input string) (*ChainResult, error) {
	chainID := uuid.New().String()
	s.logger.Info("starting document chain execution",
		slog.String("chain_id", chainID),
		slog.String("input_length", strconv.Itoa(len(input))))

	// Initialize chain progress
	progress := &ChainProgress{
		ChainID:     chainID,
		Status:      ChainStatusRunning,
		CurrentStep: ChainStepPRDGeneration,
		Progress:    0.0,
		StartTime:   time.Now(),
		LastUpdate:  time.Now(),
		Context:     make(map[string]string),
	}
	s.chains[chainID] = progress

	// Create generation context
	context := &GenerationContext{
		Repository:  s.detectRepository(),
		ProjectType: s.detectProjectType(input),
		UserInputs:  []string{input},
		Templates:   s.templates,
		UserPrefs:   s.getDefaultUserPreferences(),
	}

	result := &ChainResult{
		ID:       chainID,
		Progress: progress,
		Metadata: ChainMetadata{
			StartTime:      time.Now(),
			Repository:     context.Repository,
			GeneratedBy:    "ai_chain",
			UserInputCount: len(context.UserInputs),
		},
	}

	// Step 1: Generate PRD interactively
	s.updateProgress(progress, ChainStepPRDGeneration, 0.2)
	prd, err := s.GeneratePRDInteractive(ctx, context)
	if err != nil {
		return s.handleChainError(progress, result, "PRD generation failed", err)
	}
	result.PRD = prd
	progress.StepsComplete = append(progress.StepsComplete, ChainStepPRDGeneration)

	// Step 2: Generate TRD from PRD
	s.updateProgress(progress, ChainStepTRDGeneration, 0.4)
	trd, err := s.GenerateTRDFromPRD(ctx, prd)
	if err != nil {
		return s.handleChainError(progress, result, "TRD generation failed", err)
	}
	result.TRD = trd
	progress.StepsComplete = append(progress.StepsComplete, ChainStepTRDGeneration)

	// Step 3: Generate main tasks from TRD
	s.updateProgress(progress, ChainStepMainTaskGeneration, 0.6)
	mainTasks, err := s.GenerateMainTasksFromTRD(ctx, trd)
	if err != nil {
		return s.handleChainError(progress, result, "main task generation failed", err)
	}
	result.MainTasks = mainTasks
	progress.StepsComplete = append(progress.StepsComplete, ChainStepMainTaskGeneration)

	// Step 4: Generate sub-tasks for each main task
	s.updateProgress(progress, ChainStepSubTaskGeneration, 0.8)
	var allSubTasks []*SubTask
	for _, mainTask := range mainTasks {
		subTasks, err := s.GenerateSubTasksFromMain(ctx, mainTask)
		if err != nil {
			s.logger.Warn("sub-task generation failed for main task",
				slog.String("main_task", mainTask.ID),
				slog.Any("error", err))
			progress.StepsFailed = append(progress.StepsFailed, ChainStepSubTaskGeneration)
			continue
		}
		allSubTasks = append(allSubTasks, subTasks...)
	}
	result.SubTasks = allSubTasks
	progress.StepsComplete = append(progress.StepsComplete, ChainStepSubTaskGeneration)

	// Step 5: Finalize and complete
	s.updateProgress(progress, ChainStepValidation, 1.0)
	progress.Status = ChainStatusCompleted
	progress.CurrentStep = ""

	// Update metadata
	result.Metadata.EndTime = time.Now()
	result.Metadata.Duration = result.Metadata.EndTime.Sub(result.Metadata.StartTime).String()
	result.Metadata.TotalTasks = len(mainTasks) + len(allSubTasks)
	result.Metadata.ProjectType = context.ProjectType

	s.logger.Info("document chain execution completed",
		slog.String("chain_id", chainID),
		slog.Int("main_tasks", len(mainTasks)),
		slog.Int("sub_tasks", len(allSubTasks)),
		slog.String("duration", result.Metadata.Duration))

	return result, nil
}

// GeneratePRDInteractive generates a PRD using AI with interactive prompts
func (s *DefaultDocumentChainService) GeneratePRDInteractive(ctx context.Context, context *GenerationContext) (*PRDEntity, error) {
	if len(context.UserInputs) == 0 {
		return nil, errors.New("no user input provided for PRD generation")
	}

	// Simulate AI generation for now - in full implementation this would call the MCP AI service
	prd := &PRDEntity{
		ID:          uuid.New().String(),
		Title:       s.extractProjectTitle(context.UserInputs[0]),
		Description: context.UserInputs[0],
		Features:    s.extractFeatures(context.UserInputs[0]),
		UserStories: s.generateUserStories(context.UserInputs[0]),
		Metadata: map[string]interface{}{
			"generated_by": "ai_chain",
			"repository":   context.Repository,
			"project_type": context.ProjectType,
		},
		CreatedAt: time.Now(),
	}

	s.logger.Info("generated PRD",
		slog.String("prd_id", prd.ID),
		slog.String("title", prd.Title),
		slog.Int("features", len(prd.Features)))

	return prd, nil
}

// GenerateTRDFromPRD generates a TRD from a PRD using AI
func (s *DefaultDocumentChainService) GenerateTRDFromPRD(ctx context.Context, prd *PRDEntity) (*TRDEntity, error) {
	if prd == nil {
		return nil, errors.New("PRD cannot be nil")
	}

	// Simulate AI generation for now
	trd := &TRDEntity{
		ID:             uuid.New().String(),
		PRDID:          prd.ID,
		Title:          "Technical Requirements for " + prd.Title,
		Architecture:   s.generateArchitecture(prd),
		TechStack:      s.generateTechStack(prd),
		Requirements:   s.generateTechnicalRequirements(prd),
		Implementation: s.generateImplementationSteps(prd),
		Metadata: map[string]interface{}{
			"generated_by": "ai_chain",
			"prd_id":       prd.ID,
			"features":     len(prd.Features),
		},
		CreatedAt: time.Now(),
	}

	s.logger.Info("generated TRD",
		slog.String("trd_id", trd.ID),
		slog.String("prd_id", prd.ID),
		slog.Int("requirements", len(trd.Requirements)))

	return trd, nil
}

// GenerateMainTasksFromTRD generates main tasks from a TRD
func (s *DefaultDocumentChainService) GenerateMainTasksFromTRD(ctx context.Context, trd *TRDEntity) ([]*MainTask, error) {
	if trd == nil {
		return nil, errors.New("TRD cannot be nil")
	}

	var mainTasks []*MainTask

	// Generate tasks based on implementation steps
	for i, step := range trd.Implementation {
		task := &MainTask{
			ID:               fmt.Sprintf("MT-%03d", i+1),
			Name:             s.extractTaskName(step),
			Description:      step,
			Phase:            s.determinePhase(step, i, len(trd.Implementation)),
			Duration:         s.estimateMainTaskDuration(step),
			AtomicValidation: s.validateAtomicTask(step),
			Dependencies:     s.extractDependencies(step, i, mainTasks),
			Content:          step,
			CreatedAt:        time.Now(),
		}

		mainTasks = append(mainTasks, task)
	}

	// Detect dependencies between main tasks
	s.detectMainTaskDependencies(mainTasks)

	s.logger.Info("generated main tasks from TRD",
		slog.String("trd_id", trd.ID),
		slog.Int("task_count", len(mainTasks)))

	return mainTasks, nil
}

// GenerateSubTasksFromMain generates sub-tasks from a main task
func (s *DefaultDocumentChainService) GenerateSubTasksFromMain(ctx context.Context, mainTask *MainTask) ([]*SubTask, error) {
	if mainTask == nil {
		return nil, errors.New("main task cannot be nil")
	}

	var subTasks []*SubTask

	// Break down main task into 2-4 hour sub-tasks
	steps := s.breakDownTask(mainTask.Content)
	for i, step := range steps {
		subTask := &SubTask{
			ID:                 fmt.Sprintf("ST-%s-%03d", mainTask.ID, i+1),
			ParentTaskID:       mainTask.ID,
			Name:               s.extractSubTaskName(step),
			Duration:           s.estimateSubTaskDuration(step),
			Type:               s.determineImplementationType(step),
			Deliverables:       s.extractDeliverables(step),
			AcceptanceCriteria: s.generateAcceptanceCriteria(step),
			Dependencies:       s.extractSubTaskDependencies(step, i, subTasks),
			Content:            step,
			CreatedAt:          time.Now(),
		}

		// Ensure duration is within 2-4 hour range
		if subTask.Duration < 2 {
			subTask.Duration = 2
		} else if subTask.Duration > 4 {
			subTask.Duration = 4
		}

		subTasks = append(subTasks, subTask)
	}

	// Update main task sub-task count
	mainTask.SubTaskCount = len(subTasks)

	s.logger.Info("generated sub-tasks from main task",
		slog.String("main_task_id", mainTask.ID),
		slog.Int("sub_task_count", len(subTasks)))

	return subTasks, nil
}

// GetChainProgress returns the progress of a specific chain
func (s *DefaultDocumentChainService) GetChainProgress(chainID string) (*ChainProgress, error) {
	progress, exists := s.chains[chainID]
	if !exists {
		return nil, fmt.Errorf("chain not found: %s", chainID)
	}
	return progress, nil
}

// ListChains returns all chain progress records
func (s *DefaultDocumentChainService) ListChains() ([]*ChainProgress, error) {
	var chains []*ChainProgress
	for _, progress := range s.chains {
		chains = append(chains, progress)
	}
	return chains, nil
}

// Helper methods

func (s *DefaultDocumentChainService) updateProgress(progress *ChainProgress, step ChainStep, completionRatio float64) {
	progress.CurrentStep = step
	progress.Progress = completionRatio
	progress.LastUpdate = time.Now()
}

func (s *DefaultDocumentChainService) handleChainError(progress *ChainProgress, result *ChainResult, message string, err error) (*ChainResult, error) {
	progress.Status = ChainStatusFailed
	progress.ErrorMessage = fmt.Sprintf("%s: %v", message, err)
	progress.LastUpdate = time.Now()
	progress.StepsFailed = append(progress.StepsFailed, progress.CurrentStep)

	result.Metadata.EndTime = time.Now()
	result.Metadata.Duration = result.Metadata.EndTime.Sub(result.Metadata.StartTime).String()

	s.logger.Error("document chain execution failed",
		slog.String("chain_id", result.ID),
		slog.String("step", string(progress.CurrentStep)),
		slog.Any("error", err))

	return result, fmt.Errorf("%s: %w", message, err)
}

func (s *DefaultDocumentChainService) detectRepository() string {
	// In a real implementation, this would detect the git repository
	return "current-project"
}

func (s *DefaultDocumentChainService) detectProjectType(input string) string {
	// Simple heuristics to detect project type
	input = input + " " // Ensure we can check for patterns
	switch {
	case contains(input, "api", "backend", "service"):
		return "api"
	case contains(input, "web", "frontend", "ui"):
		return "web-app"
	case contains(input, "cli", "command", "tool"):
		return "cli"
	case contains(input, "mobile", "app", "ios", "android"):
		return "mobile"
	default:
		return "general"
	}
}

func (s *DefaultDocumentChainService) getDefaultUserPreferences() UserPreferences {
	return UserPreferences{
		PreferredTaskSize:   "medium",
		PreferredComplexity: "medium",
		IncludeTests:        true,
		IncludeDocs:         true,
	}
}

func (s *DefaultDocumentChainService) extractProjectTitle(input string) string {
	// Simple title extraction
	words := splitWords(input)
	if len(words) > 0 {
		return words[0] + " Project"
	}
	return "New Project"
}

func (s *DefaultDocumentChainService) extractFeatures(input string) []string {
	// Simple feature extraction
	features := []string{
		"Core functionality implementation",
		"User interface development",
		"Data management",
		"Testing and validation",
	}
	return features
}

func (s *DefaultDocumentChainService) generateUserStories(input string) []string {
	return []string{
		"As a user, I want to use the main functionality",
		"As a user, I want reliable performance",
		"As a user, I want clear documentation",
	}
}

func (s *DefaultDocumentChainService) generateArchitecture(prd *PRDEntity) string {
	return "Modular architecture with clean separation of concerns"
}

func (s *DefaultDocumentChainService) generateTechStack(prd *PRDEntity) []string {
	return []string{"Go", "REST API", "Database", "Testing Framework"}
}

func (s *DefaultDocumentChainService) generateTechnicalRequirements(prd *PRDEntity) []string {
	var reqs []string
	for _, feature := range prd.Features {
		reqs = append(reqs, "Implement "+feature)
	}
	return reqs
}

func (s *DefaultDocumentChainService) generateImplementationSteps(prd *PRDEntity) []string {
	return []string{
		"Set up project structure and dependencies",
		"Implement core business logic",
		"Develop user interface components",
		"Add data persistence layer",
		"Implement security and validation",
		"Add comprehensive testing",
		"Create documentation and deployment guides",
	}
}

func (s *DefaultDocumentChainService) extractTaskName(step string) string {
	words := splitWords(step)
	if len(words) > 3 {
		return joinWords(words[:3]) + "..."
	}
	return step
}

func (s *DefaultDocumentChainService) determinePhase(step string, index, total int) string {
	ratio := float64(index) / float64(total)
	switch {
	case ratio < 0.3:
		return "setup"
	case ratio < 0.7:
		return "development"
	default:
		return "testing"
	}
}

func (s *DefaultDocumentChainService) estimateMainTaskDuration(step string) string {
	wordCount := len(splitWords(step))
	switch {
	case wordCount < 10:
		return "1-2 days"
	case wordCount < 20:
		return "3-5 days"
	default:
		return "1-2 weeks"
	}
}

func (s *DefaultDocumentChainService) validateAtomicTask(step string) bool {
	// Simple atomic validation - check if task is focused
	return len(splitWords(step)) < 20
}

func (s *DefaultDocumentChainService) extractDependencies(step string, index int, existingTasks []*MainTask) []string {
	var deps []string
	if index > 0 && len(existingTasks) > 0 {
		// Simple dependency: each task depends on the previous one
		deps = append(deps, existingTasks[index-1].ID)
	}
	return deps
}

func (s *DefaultDocumentChainService) detectMainTaskDependencies(tasks []*MainTask) {
	// Simple sequential dependencies
	for i := 1; i < len(tasks); i++ {
		if len(tasks[i].Dependencies) == 0 {
			tasks[i].Dependencies = []string{tasks[i-1].ID}
		}
	}
}

func (s *DefaultDocumentChainService) breakDownTask(content string) []string {
	// Simple task breakdown
	return []string{
		"Research and planning for: " + content,
		"Core implementation of: " + content,
		"Testing and validation for: " + content,
		"Documentation and cleanup for: " + content,
	}
}

func (s *DefaultDocumentChainService) extractSubTaskName(step string) string {
	words := splitWords(step)
	if len(words) > 4 {
		return joinWords(words[:4]) + "..."
	}
	return step
}

func (s *DefaultDocumentChainService) estimateSubTaskDuration(step string) int {
	wordCount := len(splitWords(step))
	switch {
	case wordCount < 10:
		return 2
	case wordCount < 15:
		return 3
	default:
		return 4
	}
}

func (s *DefaultDocumentChainService) determineImplementationType(step string) string {
	switch {
	case contains(step, "test", "testing"):
		return "testing"
	case contains(step, "doc", "documentation"):
		return "documentation"
	case contains(step, "research", "planning"):
		return "research"
	default:
		return "implementation"
	}
}

func (s *DefaultDocumentChainService) extractDeliverables(step string) []string {
	stepType := s.determineImplementationType(step)
	switch stepType {
	case "testing":
		return []string{"test cases", "test results"}
	case "documentation":
		return []string{"documentation files", "code comments"}
	case "research":
		return []string{"research notes", "design decisions"}
	default:
		return []string{"working code", "unit tests"}
	}
}

func (s *DefaultDocumentChainService) generateAcceptanceCriteria(step string) []string {
	return []string{
		"Implementation is complete and functional",
		"Code follows project standards",
		"Tests pass successfully",
		"Documentation is updated",
	}
}

func (s *DefaultDocumentChainService) extractSubTaskDependencies(step string, index int, existingSubTasks []*SubTask) []string {
	var deps []string
	if index > 0 && len(existingSubTasks) > 0 {
		deps = append(deps, existingSubTasks[index-1].ID)
	}
	return deps
}

// loadDefaultTemplates loads default task templates
func loadDefaultTemplates() []*TaskTemplate {
	return []*TaskTemplate{
		{
			ID:          "api-endpoint",
			Name:        "API Endpoint Implementation",
			Type:        "implementation",
			Pattern:     "api|endpoint|rest|http",
			Keywords:    []string{"api", "endpoint", "rest", "http", "service"},
			Complexity:  "medium",
			Hours:       8,
			SubTasks:    []string{"design", "implement", "test", "document"},
			Description: "Implement REST API endpoint with validation and error handling",
			Category:    "backend",
		},
		{
			ID:          "ui-component",
			Name:        "UI Component Development",
			Type:        "implementation",
			Pattern:     "ui|component|interface|frontend",
			Keywords:    []string{"ui", "component", "interface", "frontend", "react"},
			Complexity:  "medium",
			Hours:       6,
			SubTasks:    []string{"design", "implement", "style", "test"},
			Description: "Develop reusable UI component with proper styling",
			Category:    "frontend",
		},
		{
			ID:          "database-setup",
			Name:        "Database Schema Setup",
			Type:        "setup",
			Pattern:     "database|schema|migration|table",
			Keywords:    []string{"database", "schema", "migration", "table", "sql"},
			Complexity:  "high",
			Hours:       12,
			SubTasks:    []string{"design", "create", "migrate", "test"},
			Description: "Set up database schema with proper indexing and relationships",
			Category:    "data",
		},
	}
}

// Utility functions

func contains(text string, keywords ...string) bool {
	for _, keyword := range keywords {
		if containsKeyword(text, keyword) {
			return true
		}
	}
	return false
}

func containsKeyword(text, keyword string) bool {
	// Simple case-insensitive substring check
	textLower := toLowerCase(text)
	keywordLower := toLowerCase(keyword)
	return findSubstring(textLower, keywordLower) >= 0
}

func toLowerCase(s string) string {
	var result []rune
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			result = append(result, r+32)
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func findSubstring(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func splitWords(text string) []string {
	var words []string
	var current []rune

	for _, r := range text {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		} else {
			current = append(current, r)
		}
	}

	if len(current) > 0 {
		words = append(words, string(current))
	}

	return words
}

func joinWords(words []string) string {
	if len(words) == 0 {
		return ""
	}

	result := words[0]
	for i := 1; i < len(words); i++ {
		result += " " + words[i]
	}
	return result
}
