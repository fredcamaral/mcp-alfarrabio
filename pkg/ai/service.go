// Package ai provides shared AI service implementation
package ai

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Service implements the AIService interface
type Service struct {
	config *Config
	logger *slog.Logger
	client AIClient
}

// NewService creates a new AI service
func NewService(config *Config, logger *slog.Logger) (*Service, error) {
	if config == nil {
		config = DefaultConfig()
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	// Create the appropriate AI client based on provider
	var client AIClient
	var err error

	switch config.Provider {
	case "claude":
		client, err = NewClaudeClient(config.APIKey, config.Model)
	case "openai":
		client, err = NewOpenAIClient(config.APIKey, config.Model)
	case "perplexity":
		client, err = NewPerplexityClient(config.APIKey, config.Model)
	case "mock":
		client = NewMockClient()
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", config.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	service := &Service{
		config: config,
		logger: logger,
		client: client,
	}

	logger.Info("AI service initialized",
		"provider", config.Provider,
		"model", config.Model)

	return service, nil
}

// GeneratePRD generates a Product Requirements Document
func (s *Service) GeneratePRD(ctx context.Context, request PRDRequest) (*PRDResponse, error) {
	s.logger.Debug("Generating PRD",
		"inputs_count", len(request.UserInputs),
		"project_type", request.ProjectType,
		"repository", request.Repository)

	// Build prompt for PRD generation
	prompt := s.buildPRDPrompt(request)

	// Create completion request
	completionRequest := CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "You are an expert product manager and technical writer. Generate comprehensive Product Requirements Documents (PRDs) that are detailed, actionable, and well-structured."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	// Call AI client
	response, err := s.client.Complete(ctx, completionRequest)
	if err != nil {
		s.logger.Error("Failed to generate PRD", "error", err)
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	content := response.Content

	prdResponse := &PRDResponse{
		Content: content,
		Metadata: map[string]string{
			"provider":     response.Provider,
			"model":        response.Model,
			"tokens":       fmt.Sprintf("%d", response.Usage.TotalTokens),
			"generated_at": time.Now().Format(time.RFC3339),
			"project_type": request.ProjectType,
			"repository":   request.Repository,
		},
		SessionID: request.SessionID,
	}

	s.logger.Info("PRD generated",
		"content_length", len(content),
		"session_id", request.SessionID)

	return prdResponse, nil
}

// GenerateTRD generates a Technical Requirements Document
func (s *Service) GenerateTRD(ctx context.Context, request TRDRequest) (*TRDResponse, error) {
	s.logger.Debug("Generating TRD",
		"prd_length", len(request.PRDContent),
		"repository", request.Repository)

	// Build prompt for TRD generation
	prompt := s.buildTRDPrompt(request)

	// Create completion request
	completionRequest := CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "You are an expert technical architect and engineer. Generate comprehensive Technical Requirements Documents (TRDs) that translate business requirements into detailed technical specifications suitable for implementation by developers."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   6144,
		Temperature: 0.7,
	}

	// Call AI client
	response, err := s.client.Complete(ctx, completionRequest)
	if err != nil {
		s.logger.Error("Failed to generate TRD", "error", err)
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	content := response.Content

	trdResponse := &TRDResponse{
		Content: content,
		Metadata: map[string]string{
			"provider":     response.Provider,
			"model":        response.Model,
			"tokens":       fmt.Sprintf("%d", response.Usage.TotalTokens),
			"generated_at": time.Now().Format(time.RFC3339),
			"repository":   request.Repository,
		},
		SessionID: request.SessionID,
	}

	s.logger.Info("TRD generated",
		"content_length", len(content),
		"session_id", request.SessionID)

	return trdResponse, nil
}

// GenerateMainTasks generates main tasks from TRD
func (s *Service) GenerateMainTasks(ctx context.Context, request TaskRequest) (*TaskResponse, error) {
	s.logger.Debug("Generating main tasks",
		"content_length", len(request.Content),
		"repository", request.Repository)

	// Build prompt for main task generation
	prompt := s.buildTaskPrompt(request)

	// Create completion request
	completionRequest := CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "You are an expert project manager and technical lead. Generate atomic, functional main tasks that represent major project phases. Each task must deliver working software that users can interact with."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	// Call AI client
	response, err := s.client.Complete(ctx, completionRequest)
	if err != nil {
		s.logger.Error("Failed to generate main tasks", "error", err)
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// Parse AI response to extract tasks
	tasks := s.parseTasksFromResponse(response.Content, "main")

	taskResponse := &TaskResponse{
		Tasks: tasks,
		Metadata: map[string]string{
			"provider":     response.Provider,
			"model":        response.Model,
			"tokens":       fmt.Sprintf("%d", response.Usage.TotalTokens),
			"generated_at": time.Now().Format(time.RFC3339),
			"task_type":    "main",
			"repository":   request.Repository,
		},
		SessionID: request.SessionID,
	}

	s.logger.Info("Main tasks generated",
		"task_count", len(tasks),
		"session_id", request.SessionID)

	return taskResponse, nil
}

// GenerateSubTasks generates sub-tasks from main task
func (s *Service) GenerateSubTasks(ctx context.Context, request TaskRequest) (*TaskResponse, error) {
	s.logger.Debug("Generating sub tasks",
		"content_length", len(request.Content),
		"repository", request.Repository)

	// Build prompt for sub-task generation
	prompt := s.buildTaskPrompt(request)

	// Create completion request
	completionRequest := CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "You are an expert technical lead and software architect. Generate implementable sub-tasks that break down main tasks into actionable work items. Each sub-task must be completable in a single 2-4 hour development session."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	// Call AI client
	response, err := s.client.Complete(ctx, completionRequest)
	if err != nil {
		s.logger.Error("Failed to generate sub tasks", "error", err)
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// Parse AI response to extract tasks
	tasks := s.parseTasksFromResponse(response.Content, "sub")

	taskResponse := &TaskResponse{
		Tasks: tasks,
		Metadata: map[string]string{
			"provider":     response.Provider,
			"model":        response.Model,
			"tokens":       fmt.Sprintf("%d", response.Usage.TotalTokens),
			"generated_at": time.Now().Format(time.RFC3339),
			"task_type":    "sub",
			"repository":   request.Repository,
		},
		SessionID: request.SessionID,
	}

	s.logger.Info("Sub tasks generated",
		"task_count", len(tasks),
		"session_id", request.SessionID)

	return taskResponse, nil
}

// StartInteractiveSession starts an interactive session
func (s *Service) StartInteractiveSession(ctx context.Context, docType string) (*SessionResponse, error) {
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	s.logger.Debug("Starting interactive session",
		"session_id", sessionID,
		"doc_type", docType)

	response := &SessionResponse{
		SessionID:  sessionID,
		Message:    fmt.Sprintf("Starting interactive %s creation session", docType),
		Question:   s.getFirstQuestion(docType),
		IsComplete: false,
	}

	s.logger.Info("Interactive session started",
		"session_id", sessionID,
		"doc_type", docType)

	return response, nil
}

// ContinueSession continues an interactive session
func (s *Service) ContinueSession(ctx context.Context, sessionID, userInput string) (*SessionResponse, error) {
	s.logger.Debug("Continuing session",
		"session_id", sessionID,
		"input_length", len(userInput))

	// For now, return a mock continuation
	// TODO: Implement actual session state management and AI interaction
	response := &SessionResponse{
		SessionID:   sessionID,
		Message:     fmt.Sprintf("Received input: %s", userInput),
		Question:    "Is there anything else you'd like to add?",
		IsComplete:  false,
		FinalResult: "",
	}

	s.logger.Info("Session continued",
		"session_id", sessionID)

	return response, nil
}

// EndSession ends an interactive session
func (s *Service) EndSession(ctx context.Context, sessionID string) error {
	s.logger.Debug("Ending session", "session_id", sessionID)

	// TODO: Implement session cleanup and final document generation

	s.logger.Info("Session ended", "session_id", sessionID)
	return nil
}

// AnalyzeComplexity analyzes the complexity of content
func (s *Service) AnalyzeComplexity(ctx context.Context, content string) (int, error) {
	s.logger.Debug("Analyzing complexity", "content_length", len(content))

	// Use AI to analyze complexity
	complexity, err := s.analyzeComplexityWithAI(ctx, content)
	if err != nil {
		s.logger.Warn("AI complexity analysis failed, using fallback", "error", err)
		complexity = s.calculateMockComplexity(content)
	}

	s.logger.Info("Complexity analyzed",
		"content_length", len(content),
		"complexity", complexity)

	return complexity, nil
}

// Mock generation methods - TODO: Replace with actual AI implementations

func (s *Service) generateMockPRD(request PRDRequest) string {
	return fmt.Sprintf(`# Product Requirements Document

## 1. Introduction
This PRD was generated based on your inputs for %s project.

## 2. Project Overview
Project Type: %s
Repository: %s

## 3. User Inputs Analysis
%s

## 4. Goals
- Deliver a high-quality solution
- Meet user requirements
- Maintain code quality standards

## 5. Requirements
### Functional Requirements
- Core feature implementation
- User interface design
- Data management

### Non-Functional Requirements
- Performance optimization
- Security measures
- Scalability considerations

## 6. Success Metrics
- User satisfaction > 85%%
- Performance benchmarks met
- Zero critical security issues

Generated by: %s
Generated at: %s`,
		request.ProjectType,
		request.ProjectType,
		request.Repository,
		fmt.Sprintf("Based on %d user inputs", len(request.UserInputs)),
		s.config.Provider,
		time.Now().Format(time.RFC3339))
}

func (s *Service) generateMockTRD(request TRDRequest) string {
	return fmt.Sprintf(`# Technical Requirements Document

## 1. Architecture Overview
This TRD is based on the provided PRD (length: %d characters).

## 2. Technology Stack
- Backend: Go 1.23+
- Database: PostgreSQL/SQLite
- API: REST/GraphQL
- Frontend: React/Next.js (if applicable)

## 3. System Components
### Core Services
- API Gateway
- Authentication Service
- Business Logic Layer
- Data Access Layer

## 4. Infrastructure Requirements
- Container orchestration (Docker/Kubernetes)
- Load balancing
- Monitoring and logging
- CI/CD pipeline

## 5. Security Requirements
- Authentication and authorization
- Data encryption
- Input validation
- Rate limiting

## 6. Performance Requirements
- Response time < 100ms (p95)
- Throughput > 1000 RPS
- Availability > 99.9%%

Generated by: %s
Generated at: %s`,
		len(request.PRDContent),
		s.config.Provider,
		time.Now().Format(time.RFC3339))
}

func (s *Service) generateMockMainTasks(request TaskRequest) []GeneratedTask {
	return []GeneratedTask{
		{
			ID:          "MT-001",
			Name:        "Foundation Setup",
			Description: "Set up project foundation and development environment",
			Duration:    8 * time.Hour,
			Priority:    "high",
		},
		{
			ID:           "MT-002",
			Name:         "Core Implementation",
			Description:  "Implement core business logic and features",
			Duration:     16 * time.Hour,
			Priority:     "high",
			Dependencies: []string{"MT-001"},
		},
		{
			ID:           "MT-003",
			Name:         "Integration Layer",
			Description:  "Develop integration layer and external APIs",
			Duration:     12 * time.Hour,
			Priority:     "medium",
			Dependencies: []string{"MT-002"},
		},
		{
			ID:           "MT-004",
			Name:         "Testing & Quality",
			Description:  "Implement comprehensive testing and quality assurance",
			Duration:     10 * time.Hour,
			Priority:     "medium",
			Dependencies: []string{"MT-003"},
		},
	}
}

func (s *Service) generateMockSubTasks(request TaskRequest) []GeneratedTask {
	return []GeneratedTask{
		{
			ID:          "ST-001",
			Name:        "Environment Setup",
			Description: "Configure development environment and tools",
			Duration:    2 * time.Hour,
			Priority:    "high",
		},
		{
			ID:          "ST-002",
			Name:        "Database Schema",
			Description: "Design and implement database schema",
			Duration:    3 * time.Hour,
			Priority:    "high",
		},
		{
			ID:          "ST-003",
			Name:        "API Endpoints",
			Description: "Create basic API endpoints and routing",
			Duration:    2 * time.Hour,
			Priority:    "medium",
		},
		{
			ID:          "ST-004",
			Name:        "Authentication",
			Description: "Implement user authentication system",
			Duration:    1 * time.Hour,
			Priority:    "medium",
		},
	}
}

func (s *Service) getFirstQuestion(docType string) string {
	switch docType {
	case "prd":
		return "What is the main goal of your project?"
	case "trd":
		return "What technology stack do you prefer?"
	default:
		return "Please describe your requirements."
	}
}

func (s *Service) calculateMockComplexity(content string) int {
	// Simple mock complexity calculation based on content length
	length := len(content)
	switch {
	case length < 500:
		return 3
	case length < 1500:
		return 5
	case length < 3000:
		return 8
	case length < 5000:
		return 13
	default:
		return 21
	}
}

// buildPRDPrompt constructs a prompt for PRD generation using default rules or custom rules
func (s *Service) buildPRDPrompt(request PRDRequest) string {
	userInputsText := ""
	for i, input := range request.UserInputs {
		userInputsText += fmt.Sprintf("%d. %s\n", i+1, input)
	}

	baseInfo := fmt.Sprintf(`Project Information:
- Project Type: %s
- Repository: %s
- User Requirements:
%s`, request.ProjectType, request.Repository, userInputsText)

	// If user provided custom rules, use those
	if request.CustomRules != "" && !request.UseDefaultRule {
		return fmt.Sprintf(`%s

Custom Generation Rules:
%s

Please generate a Product Requirements Document following the custom rules provided above.`, baseInfo, request.CustomRules)
	}

	// Otherwise, use default comprehensive rules from create-prd.mdc
	return fmt.Sprintf(`You are an expert product manager and technical writer. Generate a comprehensive Product Requirements Document (PRD) following the structure defined in create-prd.mdc.

%s

IMPORTANT: Follow the exact PRD structure with numbered sections (0-17):

0. **Index**: ///REVIEW
1. **Introduction/Overview**: Briefly describe the feature and problem it solves
2. **Goals**: List specific, measurable objectives
3. **User Stories**: Detail user narratives with benefits
4. **User Experience**: Describe personas, user flows, UI/UX considerations
5. **Functional Requirements**: Numbered list of specific functionalities
6. **Non-Goals (Out of Scope)**: What this feature will NOT include
7. **Design Considerations**: UI/UX requirements and components
8. **Technical Considerations**: Known constraints, dependencies, suggestions
9. **Success Metrics**: Measurable success criteria
10. **Data Modeling**: Core entities, relationships, key attributes, validation rules
11. **API Modeling**: Key endpoints, HTTP methods, request/response structures
12. **Sequence Diagrams**: User Interaction Flow, System Internal Flow, Full API Workflow
13. **Development Roadmap**: Logical phases with scope definition (max 5 phases)
14. **Logical Dependency Chain**: Foundation-first approach, quick wins, atomic features
15. **Risks and Mitigations**: Technical, product, and resource risks with strategies
16. **Architecture Patterns & Principles**: Hexagonal Architecture, lib-commons integration, SOLID principles
17. **Open Questions**: Remaining questions for clarification

TARGET AUDIENCE: Junior developers - be explicit, unambiguous, avoid jargon.

MANDATORY REQUIREMENTS:
- Use Hexagonal Architecture principles (domain/adapters/infrastructure structure)
- Include lib-commons integration requirements
- Define entity relationships using simple language ("User has many Workflows")
- Provide realistic JSON examples for API endpoints
- Include mermaid sequence diagrams
- Focus on atomic, testable deliverables in each phase
- Include specific validation rules and error handling

Generate a complete, production-ready PRD that a junior developer can use to implement the feature.`, baseInfo)
}

// buildTRDPrompt constructs a prompt for TRD generation using default rules or custom rules
func (s *Service) buildTRDPrompt(request TRDRequest) string {
	baseInfo := fmt.Sprintf(`PRD Content:
%s

Repository: %s`, request.PRDContent, request.Repository)

	// If user provided custom rules, use those
	if request.CustomRules != "" && !request.UseDefaultRule {
		return fmt.Sprintf(`%s

Custom Generation Rules:
%s

Please generate a Technical Requirements Document following the custom rules provided above.`, baseInfo, request.CustomRules)
	}

	// Otherwise, use default comprehensive rules from create-trd.mdc
	return fmt.Sprintf(`You are an expert technical architect and engineer. Generate a comprehensive Technical Requirements Document (TRD) following the structure defined in create-trd.mdc.

%s

IMPORTANT: Follow the exact TRD structure with numbered sections (0-17):

0. **Index**
1. **Executive Summary**: Technical overview linking to PRD objectives
2. **System Architecture**: High-level architecture diagram and component overview
3. **Technology Stack**: Complete list of technologies, languages, frameworks, and tools
4. **Data Architecture**: Detailed database schema, data flow, and storage strategy
5. **API Specifications**: Complete API documentation with OpenAPI/Swagger specs
6. **Component Design**: Detailed design of each system component
7. **Integration Architecture**: How components integrate with each other and external systems
8. **Security Architecture**: Authentication, authorization, encryption, and security measures
9. **Performance Requirements**: Specific metrics, benchmarks, and optimization strategies
10. **Infrastructure Requirements**: Servers, networking, storage, and deployment needs
11. **Development Standards**: Coding standards, patterns, and best practices
12. **Testing Strategy**: Comprehensive testing plan including unit, integration, and e2e tests
13. **Deployment Architecture**: CI/CD pipeline, deployment strategy, and rollback procedures
14. **Monitoring & Observability**: Logging, metrics, tracing, and alerting requirements
15. **Technical Risks**: Technical challenges and mitigation strategies
16. **Implementation Roadmap**: Technical tasks breakdown aligned with PRD phases
17. **Technical Decisions**: Key technical decisions and their rationale

MANDATORY REQUIREMENTS:
- Include mermaid diagrams for system architecture
- Provide complete database schema with DDL
- Include OpenAPI 3.0 specifications
- Define specific performance metrics and targets
- Include Kubernetes deployment configurations
- Specify monitoring and alerting rules
- Map technical tasks to PRD phases
- Include security measures for all identified risks

Generate a complete, production-ready TRD that development teams can use for implementation.`, baseInfo)
}

// buildTaskPrompt constructs a prompt for task generation using default rules or custom rules
func (s *Service) buildTaskPrompt(request TaskRequest) string {
	var taskTypeDesc string
	if request.TaskType == "main" {
		taskTypeDesc = "main tasks that represent major project phases"
	} else {
		taskTypeDesc = "sub-tasks that break down the main task into actionable work items"
	}

	baseInfo := fmt.Sprintf(`Content:
%s

Repository: %s
Task Type: %s (%s)`, request.Content, request.Repository, request.TaskType, taskTypeDesc)

	// If user provided custom rules, use those
	if request.CustomRules != "" && !request.UseDefaultRule {
		return fmt.Sprintf(`%s

Custom Generation Rules:
%s

Please generate %s following the custom rules provided above.`, baseInfo, request.CustomRules, taskTypeDesc)
	}

	// Otherwise, use default comprehensive rules from generate-main-tasks.mdc or generate-sub-tasks.mdc
	if request.TaskType == "main" {
		return fmt.Sprintf(`You are an expert project manager and technical lead. Generate atomic, functional main tasks following the structure defined in generate-main-tasks.mdc.

%s

IMPORTANT: Each main task MUST be atomic and deliver working software that users can interact with.

Main Task Structure (sections 1-10):
1. **Task Overview**: Task ID (MT-###), name, phase type, duration, atomic validation
2. **Deliverable Description**: Primary deliverable, user value, business value, working definition
3. **Functional Scope**: Included features, user capabilities, API endpoints, UI components
4. **Technical Scope**: Architecture components, data model, integration points, infrastructure
5. **Dependencies**: Required previous phases, external/internal/blocking dependencies
6. **Acceptance Criteria**: Functional, technical, quality, and user acceptance criteria
7. **Testing Requirements**: Unit, integration, end-to-end, and performance testing
8. **Deployment Definition**: Target, configuration, data migration, rollback plan
9. **Success Metrics**: Technical, user, business, and quality metrics
10. **Risk Assessment**: Technical, integration, timeline, and quality risks

ATOMIC PRINCIPLES:
- Self-contained: Can be developed, tested, and deployed independently
- Functional: Delivers working software that users can interact with
- Testable: Has clear acceptance criteria and can be validated
- Deployable: Results in a runnable application (even if limited scope)
- Valuable: Provides measurable user or business value

Generate complete main task specifications that development teams can execute.`, baseInfo)
	} else {
		return fmt.Sprintf(`You are an expert technical lead and software architect. Generate implementable sub-tasks following the structure defined in generate-sub-tasks.mdc.

%s

IMPORTANT: Each sub-task MUST be implementable in a single 2-4 hour development session.

Sub-Task Structure (sections 1-8):
1. **Sub-Task Overview**: ID (ST-MT-###-###), name, parent task, duration, implementation type
2. **Deliverable Specification**: Primary output, code location, technical requirements, interface definition
3. **Implementation Details**: Step-by-step approach, code examples, configuration changes, dependencies
4. **Acceptance Criteria**: Functional, technical, integration, and test criteria
5. **Testing Requirements**: Unit tests, integration tests, manual testing, test data
6. **Definition of Done**: Code complete, tests passing, documentation updated, integration verified, review approved
7. **Dependencies and Blockers**: Required sub-tasks, external dependencies, environmental requirements, potential blockers
8. **Integration Notes**: Component interfaces, data flow, error handling, configuration impact

SUB-TASK PRINCIPLES:
- Single Session: Can be completed in 2-4 hours maximum
- Specific: Has clear, unambiguous requirements
- Testable: Can be validated independently
- Incremental: Builds toward main task completion
- Technical: Includes specific implementation details

Generate detailed, implementable sub-task specifications with concrete code examples and testing requirements.`, baseInfo)
	}
}

// parseTasksFromResponse parses AI response to extract structured tasks
func (s *Service) parseTasksFromResponse(content, taskType string) []GeneratedTask {
	// For now, return mock tasks as parsing AI responses requires more sophisticated logic
	// TODO: Implement actual AI response parsing with structured output
	if taskType == "main" {
		return s.generateMockMainTasks(TaskRequest{})
	}
	return s.generateMockSubTasks(TaskRequest{})
}

// analyzeComplexityWithAI uses AI to analyze content complexity
func (s *Service) analyzeComplexityWithAI(ctx context.Context, content string) (int, error) {
	prompt := fmt.Sprintf(`Analyze the complexity of the following content and return a single integer from 1-21 representing the complexity score:

Content to analyze:
%s

Complexity Scale:
- 1-3: Very simple (basic CRUD, single component)
- 5-8: Simple (multiple components, basic logic)
- 13-21: Complex (advanced algorithms, multiple integrations)

Return only the number, no explanation.`, content)

	completionRequest := CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "You are an expert software architect. Analyze content complexity and return only a single integer from 1-21."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   10,
		Temperature: 0.1,
	}

	response, err := s.client.Complete(ctx, completionRequest)
	if err != nil {
		return 0, fmt.Errorf("AI complexity analysis failed: %w", err)
	}

	// Try to parse the response as an integer
	complexity := s.parseComplexityFromResponse(response.Content)
	return complexity, nil
}

// parseComplexityFromResponse extracts complexity score from AI response
func (s *Service) parseComplexityFromResponse(content string) int {
	// Simple parsing - look for first number in response
	for i := 0; i < len(content); i++ {
		if content[i] >= '0' && content[i] <= '9' {
			// Found a digit, try to parse a number
			j := i
			for j < len(content) && content[j] >= '0' && content[j] <= '9' {
				j++
			}
			if num := content[i:j]; len(num) > 0 {
				if complexity := s.parseIntSafe(num); complexity >= 1 && complexity <= 21 {
					return complexity
				}
			}
		}
	}

	// Fallback to mock calculation if parsing fails
	return s.calculateMockComplexity(content)
}

// parseIntSafe safely parses string to int, returns 0 on error
func (s *Service) parseIntSafe(str string) int {
	result := 0
	for _, ch := range str {
		if ch >= '0' && ch <= '9' {
			result = result*10 + int(ch-'0')
		} else {
			return 0
		}
	}
	return result
}
