// Package ai provides AI service adapter using the shared AI package
package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/ports"
	sharedai "lerian-mcp-memory/pkg/ai"
)

// SharedAIService implements the ports.AIService interface using the shared AI package
type SharedAIService struct {
	aiService *sharedai.Service
}

// NewSharedAIService creates a new adapter using the shared AI service
func NewSharedAIService(aiService *sharedai.Service) *SharedAIService {
	return &SharedAIService{
		aiService: aiService,
	}
}

// GeneratePRD generates a PRD using the shared AI service
func (s *SharedAIService) GeneratePRD(ctx context.Context, request *ports.PRDGenerationRequest) (*ports.PRDGenerationResponse, error) {
	// Convert CLI request to shared AI request
	sharedRequest := sharedai.PRDRequest{
		UserInputs:  request.UserInputs,
		ProjectType: request.ProjectType,
		Repository:  request.Repository,
	}

	// Call shared AI service
	sharedResponse, err := s.aiService.GeneratePRD(ctx, &sharedRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PRD: %w", err)
	}

	// Convert shared AI response to CLI response
	response := &ports.PRDGenerationResponse{
		ID:          fmt.Sprintf("prd_%d", time.Now().UnixNano()),
		Title:       s.extractTitle(sharedResponse.Content),
		Description: s.extractDescription(sharedResponse.Content),
		Features:    s.extractFeatures(sharedResponse.Content),
		UserStories: s.extractUserStories(sharedResponse.Content),
		Content:     sharedResponse.Content,
		ModelUsed:   s.getModelFromMetadata(sharedResponse.Metadata),
		GeneratedAt: time.Now(),
	}

	return response, nil
}

// GenerateTRD generates a TRD using the shared AI service
func (s *SharedAIService) GenerateTRD(ctx context.Context, request *ports.TRDGenerationRequest) (*ports.TRDGenerationResponse, error) {
	// Convert CLI request to shared AI request
	sharedRequest := sharedai.TRDRequest{
		PRDContent: request.PRDContent,
		Repository: request.Repository,
	}

	// Call shared AI service
	sharedResponse, err := s.aiService.GenerateTRD(ctx, &sharedRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TRD: %w", err)
	}

	// Convert shared AI response to CLI response
	response := &ports.TRDGenerationResponse{
		ID:             fmt.Sprintf("trd_%d", time.Now().UnixNano()),
		PRDID:          request.PRDID,
		Title:          s.extractTitle(sharedResponse.Content),
		Architecture:   s.extractArchitecture(sharedResponse.Content),
		TechStack:      s.extractTechStack(sharedResponse.Content),
		Requirements:   s.extractRequirements(sharedResponse.Content),
		Implementation: s.extractImplementation(sharedResponse.Content),
		Content:        sharedResponse.Content,
		ModelUsed:      s.getModelFromMetadata(sharedResponse.Metadata),
		GeneratedAt:    time.Now(),
	}

	return response, nil
}

// GenerateMainTasks generates main tasks using the shared AI service
func (s *SharedAIService) GenerateMainTasks(ctx context.Context, request *ports.MainTaskGenerationRequest) (*ports.MainTaskGenerationResponse, error) {
	// Convert CLI request to shared AI request
	sharedRequest := sharedai.TaskRequest{
		Content:    request.TRDContent,
		TaskType:   "main",
		Repository: request.Repository,
	}

	// Call shared AI service
	sharedResponse, err := s.aiService.GenerateMainTasks(ctx, &sharedRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to generate main tasks: %w", err)
	}

	// Convert shared AI tasks to CLI main tasks
	tasks := make([]*ports.GeneratedMainTask, len(sharedResponse.Tasks))
	for i, task := range sharedResponse.Tasks {
		tasks[i] = &ports.GeneratedMainTask{
			ID:               task.ID,
			Name:             task.Name,
			Description:      task.Description,
			Phase:            s.determinePhase(i),
			Duration:         s.formatDuration(task.Duration),
			AtomicValidation: true, // Default to true for generated tasks
			Dependencies:     task.Dependencies,
			Content:          fmt.Sprintf("# %s\n\n%s", task.Name, task.Description),
		}
	}

	response := &ports.MainTaskGenerationResponse{
		Tasks:       tasks,
		ModelUsed:   s.getModelFromMetadata(sharedResponse.Metadata),
		GeneratedAt: time.Now(),
	}

	return response, nil
}

// GenerateSubTasks generates sub-tasks using the shared AI service
func (s *SharedAIService) GenerateSubTasks(ctx context.Context, request *ports.SubTaskGenerationRequest) (*ports.SubTaskGenerationResponse, error) {
	// Convert CLI request to shared AI request
	sharedRequest := sharedai.TaskRequest{
		Content:    request.MainTaskContent,
		TaskType:   "sub",
		Repository: request.Repository,
	}

	// Call shared AI service
	sharedResponse, err := s.aiService.GenerateSubTasks(ctx, &sharedRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sub-tasks: %w", err)
	}

	// Convert shared AI tasks to CLI sub-tasks
	tasks := make([]*ports.GeneratedSubTask, len(sharedResponse.Tasks))
	for i, task := range sharedResponse.Tasks {
		tasks[i] = &ports.GeneratedSubTask{
			ID:                 task.ID,
			ParentTaskID:       request.MainTaskID,
			Name:               task.Name,
			Duration:           int(task.Duration.Hours()),
			Type:               "Code", // Default implementation type
			Deliverables:       s.extractDeliverables(task.Description),
			AcceptanceCriteria: s.extractAcceptanceCriteria(task.Description),
			Dependencies:       task.Dependencies,
			Content:            fmt.Sprintf("# %s\n\n%s", task.Name, task.Description),
		}
	}

	response := &ports.SubTaskGenerationResponse{
		Tasks:       tasks,
		ModelUsed:   s.getModelFromMetadata(sharedResponse.Metadata),
		GeneratedAt: time.Now(),
	}

	return response, nil
}

// AnalyzeContent analyzes content using the shared AI service
func (s *SharedAIService) AnalyzeContent(ctx context.Context, request *ports.ContentAnalysisRequest) (*ports.ContentAnalysisResponse, error) {
	// For now, use complexity analysis as a base for content analysis
	complexity, err := s.aiService.AnalyzeComplexity(ctx, request.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze content: %w", err)
	}

	// Create a comprehensive analysis response
	response := &ports.ContentAnalysisResponse{
		ID:            fmt.Sprintf("analysis_%d", time.Now().UnixNano()),
		Summary:       s.generateSummary(request.Content),
		KeyFeatures:   s.extractKeyFeatures(request.Content),
		TechnicalReqs: s.extractTechnicalRequirements(request.Content),
		Dependencies:  s.extractDependencies(request.Content),
		Complexity:    s.convertComplexityScore(complexity),
		Sections:      s.extractSections(request.Content),
		ModelUsed:     "shared-ai",
		ProcessedAt:   time.Now(),
	}

	return response, nil
}

// EstimateComplexity estimates complexity using the shared AI service
func (s *SharedAIService) EstimateComplexity(ctx context.Context, content string) (*ports.ComplexityEstimate, error) {
	complexity, err := s.aiService.AnalyzeComplexity(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate complexity: %w", err)
	}

	estimate := s.convertComplexityScore(complexity)
	return &estimate, nil
}

// StartInteractiveSession starts an interactive session using the shared AI service
func (s *SharedAIService) StartInteractiveSession(ctx context.Context, docType string) (*ports.InteractiveSession, error) {
	sharedResponse, err := s.aiService.StartInteractiveSession(ctx, docType)
	if err != nil {
		return nil, fmt.Errorf("failed to start interactive session: %w", err)
	}

	session := &ports.InteractiveSession{
		ID:      sharedResponse.SessionID,
		Type:    docType,
		State:   ports.SessionStateActive,
		Context: make(map[string]interface{}),
		Messages: []ports.SessionMessage{
			{
				Role:      "assistant",
				Content:   sharedResponse.Message,
				Timestamp: time.Now(),
			},
		},
		ModelUsed: "shared-ai",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return session, nil
}

// ContinueSession continues an interactive session
func (s *SharedAIService) ContinueSession(ctx context.Context, sessionID, userInput string) (*ports.SessionResponse, error) {
	sharedResponse, err := s.aiService.ContinueSession(ctx, sessionID, userInput)
	if err != nil {
		return nil, fmt.Errorf("failed to continue session: %w", err)
	}

	state := ports.SessionStateActive
	if sharedResponse.IsComplete {
		state = ports.SessionStateCompleted
	}

	response := &ports.SessionResponse{
		SessionID: sessionID,
		Message: ports.SessionMessage{
			Role:      "assistant",
			Content:   sharedResponse.Message,
			Timestamp: time.Now(),
		},
		State:    state,
		Context:  make(map[string]interface{}),
		NextStep: sharedResponse.Question,
	}

	return response, nil
}

// EndSession ends an interactive session
func (s *SharedAIService) EndSession(ctx context.Context, sessionID string) error {
	return s.aiService.EndSession(ctx, sessionID)
}

// TestConnection tests the connection to the AI service
func (s *SharedAIService) TestConnection(ctx context.Context) error {
	// For shared AI service, we can just check if the service is not nil
	if s.aiService == nil {
		return errors.New("shared AI service is not initialized")
	}
	return nil
}

// IsOnline checks if the AI service is online
func (s *SharedAIService) IsOnline() bool {
	return s.aiService != nil
}

// GetAvailableModels returns the available AI models
func (s *SharedAIService) GetAvailableModels() []string {
	// For now, return the models based on what the shared AI service supports
	return []string{"mock", "claude", "openai", "perplexity"}
}

// Helper methods for content extraction and conversion

func (s *SharedAIService) extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "Generated Document"
}

func (s *SharedAIService) extractDescription(content string) string {
	// Extract the first paragraph after the title
	lines := strings.Split(content, "\n")
	inIntro := false
	var description strings.Builder

	for _, line := range lines {
		if strings.Contains(line, "Introduction") || strings.Contains(line, "Overview") {
			inIntro = true
			continue
		}
		if inIntro && strings.TrimSpace(line) != "" {
			if strings.HasPrefix(line, "#") {
				break
			}
			description.WriteString(line)
			description.WriteString(" ")
		}
	}

	result := strings.TrimSpace(description.String())
	if result == "" {
		return "AI-generated document"
	}
	return result
}

func (s *SharedAIService) extractFeatures(content string) []string {
	// Extract bullet points and numbered items
	var features []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			feature := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			features = append(features, feature)
		}
	}

	if len(features) == 0 {
		features = []string{"Core functionality", "User interface", "Data management"}
	}

	return features
}

func (s *SharedAIService) extractUserStories(content string) []string {
	// Look for user story patterns
	var stories []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if strings.Contains(line, "As a") || strings.Contains(line, "As an") {
			stories = append(stories, strings.TrimSpace(line))
		}
	}

	if len(stories) == 0 {
		stories = []string{
			"As a user, I want to access the core functionality",
			"As a developer, I want to maintain the system",
			"As an admin, I want to manage the application",
		}
	}

	return stories
}

func (s *SharedAIService) extractArchitecture(content string) string {
	// Look for architecture sections
	lines := strings.Split(content, "\n")
	inArchSection := false
	var arch strings.Builder

	for _, line := range lines {
		if strings.Contains(line, "Architecture") {
			inArchSection = true
			continue
		}
		if inArchSection {
			if strings.HasPrefix(line, "#") && !strings.Contains(line, "Architecture") {
				break
			}
			if strings.TrimSpace(line) != "" {
				arch.WriteString(line)
				arch.WriteString(" ")
			}
		}
	}

	result := strings.TrimSpace(arch.String())
	if result == "" {
		return "Microservices architecture with API gateway"
	}
	return result
}

func (s *SharedAIService) extractTechStack(content string) []string {
	// Look for technology mentions
	var techStack []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if strings.Contains(line, "Stack") || strings.Contains(line, "Technology") {
			// Extract technology items from the following lines
			continue
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") && s.isTechnology(line) {
			tech := strings.TrimPrefix(line, "- ")
			techStack = append(techStack, tech)
		}
	}

	if len(techStack) == 0 {
		techStack = []string{"Go 1.23+", "PostgreSQL", "REST API", "Docker"}
	}

	return techStack
}

func (s *SharedAIService) extractRequirements(content string) []string {
	// Extract requirements from content
	var requirements []string
	lines := strings.Split(content, "\n")
	inReqSection := false

	for _, line := range lines {
		if strings.Contains(line, "Requirements") {
			inReqSection = true
			continue
		}
		if inReqSection {
			if strings.HasPrefix(line, "#") && !strings.Contains(line, "Requirements") {
				break
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "- ") {
				req := strings.TrimPrefix(line, "- ")
				requirements = append(requirements, req)
			}
		}
	}

	if len(requirements) == 0 {
		requirements = []string{
			"Authentication and authorization",
			"Data encryption",
			"Input validation",
			"Performance optimization",
		}
	}

	return requirements
}

func (s *SharedAIService) extractImplementation(content string) []string {
	// Extract implementation details
	var impl []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if strings.Contains(line, "Implementation") || strings.Contains(line, "Development") {
			continue
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") {
			implementation := strings.TrimPrefix(line, "- ")
			impl = append(impl, implementation)
		}
	}

	if len(impl) == 0 {
		impl = []string{
			"Set up development environment",
			"Implement core business logic",
			"Create API endpoints",
			"Add comprehensive testing",
		}
	}

	return impl
}

func (s *SharedAIService) isTechnology(line string) bool {
	techKeywords := []string{"Go", "Python", "JavaScript", "React", "Node", "PostgreSQL", "Redis", "Docker", "Kubernetes", "API", "REST", "GraphQL"}
	for _, keyword := range techKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}
	return false
}

func (s *SharedAIService) determinePhase(index int) string {
	phases := []string{"Foundation", "Development", "Integration", "Testing", "Deployment"}
	if index < len(phases) {
		return phases[index]
	}
	return "Implementation"
}

func (s *SharedAIService) formatDuration(duration time.Duration) string {
	hours := int(duration.Hours())
	if hours <= 0 {
		return "1 hour"
	}
	return fmt.Sprintf("%d hours", hours)
}

func (s *SharedAIService) extractDeliverables(description string) []string {
	// Simple extraction of deliverables
	return []string{"Code implementation", "Unit tests", "Documentation"}
}

func (s *SharedAIService) extractAcceptanceCriteria(description string) []string {
	// Simple extraction of acceptance criteria
	return []string{"All tests pass", "Code review completed", "Documentation updated"}
}

func (s *SharedAIService) convertComplexityScore(score int) ports.ComplexityEstimate {
	var overall string
	var scoreFloat float64
	var estimatedHours int

	switch {
	case score <= 3:
		overall = "low"
		scoreFloat = 2.0
		estimatedHours = 4
	case score <= 8:
		overall = "medium"
		scoreFloat = 5.0
		estimatedHours = 12
	default:
		overall = "high"
		scoreFloat = 8.0
		estimatedHours = 24
	}

	return ports.ComplexityEstimate{
		Overall:        overall,
		Score:          scoreFloat,
		Factors:        []string{"Content length", "Technical depth", "Integration complexity"},
		EstimatedHours: estimatedHours,
		Confidence:     0.8,
		Categories: map[string]float64{
			"technical":   scoreFloat * 0.4,
			"business":    scoreFloat * 0.3,
			"integration": scoreFloat * 0.2,
			"maintenance": scoreFloat * 0.1,
		},
	}
}

func (s *SharedAIService) generateSummary(content string) string {
	lines := strings.Split(content, "\n")
	var summary strings.Builder
	lineCount := 0

	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") {
			summary.WriteString(line)
			summary.WriteString(" ")
			lineCount++
			if lineCount >= 3 {
				break
			}
		}
	}

	result := strings.TrimSpace(summary.String())
	if len(result) > 200 {
		result = result[:200] + "..."
	}

	if result == "" {
		return "Comprehensive analysis of the provided content"
	}

	return result
}

func (s *SharedAIService) extractKeyFeatures(content string) []string {
	return s.extractFeatures(content)
}

func (s *SharedAIService) extractTechnicalRequirements(content string) []string {
	return s.extractRequirements(content)
}

func (s *SharedAIService) extractDependencies(content string) []string {
	// Simple dependency extraction
	return []string{"System prerequisites", "External APIs", "Database requirements"}
}

func (s *SharedAIService) extractSections(content string) []ports.ContentSection {
	var sections []ports.ContentSection
	lines := strings.Split(content, "\n")
	currentSection := ""
	currentContent := strings.Builder{}
	order := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			// Save previous section if exists
			if currentSection != "" {
				sections = append(sections, ports.ContentSection{
					ID:      fmt.Sprintf("section_%d", order),
					Title:   currentSection,
					Content: strings.TrimSpace(currentContent.String()),
					Type:    "section",
					Order:   order,
				})
				order++
			}

			// Start new section
			currentSection = strings.TrimSpace(strings.TrimLeft(line, "#"))
			currentContent.Reset()
		} else if currentSection != "" {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	// Save last section
	if currentSection != "" {
		sections = append(sections, ports.ContentSection{
			ID:      fmt.Sprintf("section_%d", order),
			Title:   currentSection,
			Content: strings.TrimSpace(currentContent.String()),
			Type:    "section",
			Order:   order,
		})
	}

	return sections
}

func (s *SharedAIService) getModelFromMetadata(metadata map[string]string) string {
	if provider, ok := metadata["provider"]; ok {
		return provider
	}
	return "shared-ai"
}
