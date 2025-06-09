package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"lerian-mcp-memory/internal/documents"
)

// DocumentType represents the type of document to generate
type DocumentType string

const (
	DocumentTypePRD DocumentType = "prd"
	DocumentTypeTRD DocumentType = "trd"
)

// DocumentGenerationRequest represents a request to generate a document
type DocumentGenerationRequest struct {
	Type           DocumentType           `json:"type"`
	Input          string                 `json:"input"`
	Context        map[string]string      `json:"context"`
	Rules          []string               `json:"rules,omitempty"`
	Interactive    bool                   `json:"interactive"`
	SessionID      string                 `json:"session_id,omitempty"`
	Repository     string                 `json:"repository"`
	SourcePRD      *documents.PRDEntity   `json:"source_prd,omitempty"`
	SourceTRD      *documents.TRDEntity   `json:"source_trd,omitempty"`
	SourceMainTask *documents.MainTask    `json:"source_main_task,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// DocumentGenerationResponse represents the response from document generation
type DocumentGenerationResponse struct {
	Document    documents.Document    `json:"document"`
	Type        DocumentType          `json:"type"`
	SessionID   string                `json:"session_id,omitempty"`
	Questions   []InteractiveQuestion `json:"questions,omitempty"`
	Suggestions []string              `json:"suggestions,omitempty"`
	Duration    time.Duration         `json:"duration"`
	ModelUsed   Model                 `json:"model_used"`
	TokensUsed  TokenUsage            `json:"tokens_used"`
}

// InteractiveQuestion represents a question during interactive generation
type InteractiveQuestion struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Type     string   `json:"type"` // text, choice, multiselect
	Options  []string `json:"options,omitempty"`
	Required bool     `json:"required"`
	Default  string   `json:"default,omitempty"`
}

// InteractiveAnswer represents an answer to an interactive question
type InteractiveAnswer struct {
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
}

// DocumentGenerator handles AI-powered document generation
type DocumentGenerator struct {
	service     *Service
	ruleManager *documents.RuleManager
	templates   map[DocumentType]string
}

// NewDocumentGenerator creates a new document generator
func NewDocumentGenerator(service *Service, ruleManager *documents.RuleManager) *DocumentGenerator {
	return &DocumentGenerator{
		service:     service,
		ruleManager: ruleManager,
		templates:   initializeTemplates(),
	}
}

// initializeTemplates sets up document generation templates
func initializeTemplates() map[DocumentType]string {
	return map[DocumentType]string{
		DocumentTypePRD: `You are an expert product manager creating a comprehensive Product Requirements Document (PRD).

{{RULES}}

Based on the following input, create a detailed PRD that follows the structure and guidelines provided:

Input: {{INPUT}}
Context: {{CONTEXT}}

Generate a complete PRD with all required sections. Be thorough, specific, and professional.`,

		DocumentTypeTRD: `You are an expert software architect creating a Technical Requirements Document (TRD) based on a PRD.

{{RULES}}

Based on the following PRD, create a detailed TRD that translates business requirements into technical specifications:

PRD Content:
{{PRD_CONTENT}}

Context: {{CONTEXT}}

Generate a complete TRD that addresses all technical aspects needed to implement the PRD.`,
	}
}

// GenerateDocument generates a document based on the request
func (g *DocumentGenerator) GenerateDocument(ctx context.Context, req *DocumentGenerationRequest) (*DocumentGenerationResponse, error) {
	startTime := time.Now()

	// Get generation rules
	ruleContent, err := g.getRuleContent(req.Type, req.Rules)
	if err != nil {
		return nil, fmt.Errorf("failed to get generation rules: %w", err)
	}

	// Build prompt
	prompt, err := g.buildPrompt(req, ruleContent)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Create AI request
	aiReq := &Request{
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert in software development documentation and project management.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Metadata: RequestMetadata{
			Repository: req.Repository,
			SessionID:  req.SessionID,
			Tags:       []string{string(req.Type), "document_generation"},
		},
	}

	// Process with AI service
	aiResp, err := g.service.ProcessRequest(ctx, aiReq)
	if err != nil {
		return nil, fmt.Errorf("AI processing failed: %w", err)
	}

	// Parse AI response into document
	document, err := g.parseAIResponse(aiResp.Content, req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Create response
	resp := &DocumentGenerationResponse{
		Document:   document,
		Type:       req.Type,
		SessionID:  req.SessionID,
		Duration:   time.Since(startTime),
		ModelUsed:  aiResp.Model,
		TokensUsed: aiResp.TokensUsed,
	}

	// Add suggestions based on document type
	resp.Suggestions = g.generateSuggestions(req.Type, document)

	return resp, nil
}

// GenerateInteractive generates a document interactively
func (g *DocumentGenerator) GenerateInteractive(ctx context.Context, req *DocumentGenerationRequest) (*DocumentGenerationResponse, error) {
	// First, generate initial questions
	questions, err := g.generateQuestions(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate questions: %w", err)
	}

	return &DocumentGenerationResponse{
		Type:      req.Type,
		SessionID: req.SessionID,
		Questions: questions,
	}, nil
}

// ContinueInteractive continues interactive generation with answers
func (g *DocumentGenerator) ContinueInteractive(ctx context.Context, sessionID string, answers []InteractiveAnswer) (*DocumentGenerationResponse, error) {
	// TODO: Implement session management and continuation
	// For now, we'll generate based on the answers

	// Build context from answers
	context := make(map[string]string)
	for _, answer := range answers {
		context[answer.QuestionID] = answer.Answer
	}

	// Create generation request with context
	req := &DocumentGenerationRequest{
		Type:       DocumentTypePRD, // TODO: Get from session
		Context:    context,
		SessionID:  sessionID,
		Repository: context["repository"],
	}

	return g.GenerateDocument(ctx, req)
}

// getRuleContent retrieves and formats rule content
func (g *DocumentGenerator) getRuleContent(docType DocumentType, customRules []string) (string, error) {
	var ruleType documents.RuleType

	switch docType {
	case DocumentTypePRD:
		ruleType = documents.RulePRDGeneration
	case DocumentTypeTRD:
		ruleType = documents.RuleTRDGeneration
	default:
		return "", fmt.Errorf("unknown document type: %s", docType)
	}

	// Get default rule
	defaultRule, err := g.ruleManager.GetRuleContent(ruleType)
	if err != nil {
		return "", err
	}

	// Append custom rules if provided
	if len(customRules) > 0 {
		defaultRule += "\n\nAdditional Rules:\n" + strings.Join(customRules, "\n")
	}

	return defaultRule, nil
}

// buildPrompt builds the AI prompt for document generation
func (g *DocumentGenerator) buildPrompt(req *DocumentGenerationRequest, ruleContent string) (string, error) {
	template, ok := g.templates[req.Type]
	if !ok {
		return "", fmt.Errorf("no template for document type: %s", req.Type)
	}

	// Replace placeholders
	prompt := strings.ReplaceAll(template, "{{RULES}}", ruleContent)
	prompt = strings.ReplaceAll(prompt, "{{INPUT}}", req.Input)

	// Add context
	contextStr := ""
	for k, v := range req.Context {
		contextStr += k + ": " + v + "\n"
	}
	prompt = strings.ReplaceAll(prompt, "{{CONTEXT}}", contextStr)

	// Add source document content for dependent generation
	switch req.Type {
	case DocumentTypeTRD:
		if req.SourcePRD != nil {
			prompt = strings.ReplaceAll(prompt, "{{PRD_CONTENT}}", req.SourcePRD.Content)
		}
	}

	return prompt, nil
}

// parseAIResponse parses AI response into appropriate document type
func (g *DocumentGenerator) parseAIResponse(content string, req *DocumentGenerationRequest) (documents.Document, error) {
	switch req.Type {
	case DocumentTypePRD:
		return g.parsePRDResponse(content, req)
	case DocumentTypeTRD:
		return g.parseTRDResponse(content, req)
	default:
		return nil, fmt.Errorf("unknown document type: %s", req.Type)
	}
}

// parsePRDResponse parses AI response into PRD entity
func (g *DocumentGenerator) parsePRDResponse(content string, req *DocumentGenerationRequest) (*documents.PRDEntity, error) {
	prd := &documents.PRDEntity{
		Content:    content,
		Repository: req.Repository,
		Status:     documents.StatusGenerated,
		Author:     "AI Generated",
		Metadata:   make(map[string]string),
	}

	// Parse sections
	prd.Sections = documents.ParseSections(content)

	// Extract title
	if len(prd.Sections) > 0 {
		prd.Title = prd.Sections[0].Title
	}

	// Estimate complexity
	prd.ComplexityScore = documents.EstimateComplexity(content, prd.Sections)

	// Validate
	if err := prd.Validate(); err != nil {
		return nil, err
	}

	return prd, nil
}

// parseTRDResponse parses AI response into TRD entity
func (g *DocumentGenerator) parseTRDResponse(content string, req *DocumentGenerationRequest) (*documents.TRDEntity, error) {
	trd := &documents.TRDEntity{
		Content:    content,
		Repository: req.Repository,
		Status:     documents.StatusGenerated,
		Metadata:   make(map[string]string),
	}

	// Link to source PRD
	if req.SourcePRD != nil {
		trd.PRDID = req.SourcePRD.ID
	}

	// Parse sections
	trd.Sections = documents.ParseSections(content)

	// Extract title
	if len(trd.Sections) > 0 {
		trd.Title = trd.Sections[0].Title
	}

	// Extract technical details
	trd.TechnicalStack = g.extractTechnicalStack(content)
	trd.Architecture = g.extractArchitecture(content)

	// Validate
	if err := trd.Validate(); err != nil {
		return nil, err
	}

	return trd, nil
}

// parseMainTaskResponse parses AI response into MainTask entities

// generateQuestions generates interactive questions for document creation
func (g *DocumentGenerator) generateQuestions(req *DocumentGenerationRequest) ([]InteractiveQuestion, error) {
	switch req.Type {
	case DocumentTypePRD:
		return g.generatePRDQuestions(), nil
	case DocumentTypeTRD:
		return g.generateTRDQuestions(), nil
	default:
		return []InteractiveQuestion{}, nil
	}
}

// generatePRDQuestions generates questions for PRD creation
func (g *DocumentGenerator) generatePRDQuestions() []InteractiveQuestion {
	return []InteractiveQuestion{
		{
			ID:       "project_name",
			Question: "What is the name of your project?",
			Type:     "text",
			Required: true,
		},
		{
			ID:       "project_type",
			Question: "What type of project is this?",
			Type:     "choice",
			Options:  []string{"Web Application", "Mobile App", "API/Backend", "CLI Tool", "Library/SDK", "Other"},
			Required: true,
		},
		{
			ID:       "target_users",
			Question: "Who are the target users?",
			Type:     "text",
			Required: true,
		},
		{
			ID:       "main_problem",
			Question: "What is the main problem this project solves?",
			Type:     "text",
			Required: true,
		},
		{
			ID:       "key_features",
			Question: "What are the key features? (comma-separated)",
			Type:     "text",
			Required: true,
		},
		{
			ID:       "success_metrics",
			Question: "How will you measure success?",
			Type:     "text",
			Required: false,
		},
		{
			ID:       "constraints",
			Question: "Are there any constraints or limitations?",
			Type:     "text",
			Required: false,
		},
		{
			ID:       "timeline",
			Question: "What is the expected timeline?",
			Type:     "choice",
			Options:  []string{"1-2 weeks", "2-4 weeks", "1-3 months", "3-6 months", "6+ months"},
			Required: false,
		},
	}
}

// generateTRDQuestions generates questions for TRD creation
func (g *DocumentGenerator) generateTRDQuestions() []InteractiveQuestion {
	return []InteractiveQuestion{
		{
			ID:       "tech_stack",
			Question: "What is your preferred technology stack?",
			Type:     "multiselect",
			Options:  []string{"Go", "Python", "JavaScript/TypeScript", "Java", "C++", "Rust", "Other"},
			Required: true,
		},
		{
			ID:       "architecture",
			Question: "What architecture pattern will you use?",
			Type:     "choice",
			Options:  []string{"Monolithic", "Microservices", "Serverless", "Event-Driven", "Hexagonal", "Other"},
			Required: true,
		},
		{
			ID:       "database",
			Question: "What database(s) will you use?",
			Type:     "multiselect",
			Options:  []string{"PostgreSQL", "MySQL", "MongoDB", "Redis", "DynamoDB", "Other", "None"},
			Required: false,
		},
		{
			ID:       "deployment",
			Question: "Where will this be deployed?",
			Type:     "choice",
			Options:  []string{"AWS", "GCP", "Azure", "On-Premise", "Hybrid", "Other"},
			Required: false,
		},
		{
			ID:       "scalability",
			Question: "What are the scalability requirements?",
			Type:     "text",
			Required: false,
		},
		{
			ID:       "security",
			Question: "What are the key security requirements?",
			Type:     "text",
			Required: false,
		},
	}
}

// summarizeDocument creates a summary of a document
func (g *DocumentGenerator) summarizeDocument(doc documents.Document) string {
	content := doc.GetContent()
	// Simple summarization - take first 500 characters
	if len(content) > 500 {
		return content[:500] + "..."
	}
	return content
}

// extractTechnicalStack extracts technology mentions from content
func (g *DocumentGenerator) extractTechnicalStack(content string) []string {
	// Simple extraction based on common technology keywords
	stack := []string{}
	technologies := []string{
		"Go", "Python", "JavaScript", "TypeScript", "Java", "C++", "Rust",
		"React", "Vue", "Angular", "Node.js", "Django", "Flask", "Spring",
		"PostgreSQL", "MySQL", "MongoDB", "Redis", "Elasticsearch",
		"Docker", "Kubernetes", "AWS", "GCP", "Azure",
		"REST", "GraphQL", "gRPC", "WebSocket",
	}

	contentLower := strings.ToLower(content)
	for _, tech := range technologies {
		if strings.Contains(contentLower, strings.ToLower(tech)) {
			stack = append(stack, tech)
		}
	}

	return stack
}

// extractArchitecture extracts architecture pattern from content
func (g *DocumentGenerator) extractArchitecture(content string) string {
	patterns := map[string]string{
		"microservice":  "Microservices",
		"monolith":      "Monolithic",
		"serverless":    "Serverless",
		"event-driven":  "Event-Driven",
		"hexagonal":     "Hexagonal",
		"clean arch":    "Clean Architecture",
		"domain-driven": "Domain-Driven Design",
	}

	contentLower := strings.ToLower(content)
	for key, pattern := range patterns {
		if strings.Contains(contentLower, key) {
			return pattern
		}
	}

	return "Not Specified"
}

// generateSuggestions generates next-step suggestions
func (g *DocumentGenerator) generateSuggestions(docType DocumentType, doc documents.Document) []string {
	switch docType {
	case DocumentTypePRD:
		return []string{
			"Generate TRD from this PRD",
			"Review and refine requirements",
			"Share with stakeholders for feedback",
			"Create user stories from requirements",
		}
	case DocumentTypeTRD:
		return []string{
			"Generate main tasks from PRD and TRD",
			"Review technical decisions",
			"Create architecture diagrams",
			"Define API specifications",
		}
	default:
		return []string{}
	}
}
