// Package ai provides AI-powered TRD generation from PRDs.
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

// AITRDGenerator handles AI-powered TRD generation from PRDs
type AITRDGenerator struct {
	aiService   Service
	ruleManager *documents.RuleManager
	processor   *documents.Processor
	logger      *slog.Logger
}

// NewAITRDGenerator creates a new AI-powered TRD generator
func NewAITRDGenerator(aiService Service, ruleManager *documents.RuleManager, processor *documents.Processor, logger *slog.Logger) *AITRDGenerator {
	return &AITRDGenerator{
		aiService:   aiService,
		ruleManager: ruleManager,
		processor:   processor,
		logger:      logger,
	}
}

// GenerateTRDFromPRD generates a TRD from a PRD using AI
func (g *AITRDGenerator) GenerateTRDFromPRD(ctx context.Context, prd *documents.PRDEntity, options TRDGenerationOptions) (*documents.TRDEntity, error) {
	if prd == nil {
		return nil, fmt.Errorf("PRD cannot be nil")
	}

	g.logger.Info("generating TRD from PRD using AI",
		slog.String("prd_id", prd.ID),
		slog.String("prd_title", prd.Title))

	// Get TRD generation rules
	ruleContent, err := g.getRuleContent()
	if err != nil {
		return nil, fmt.Errorf("failed to get TRD generation rules: %w", err)
	}

	// Build AI prompt
	prompt := g.buildTRDPrompt(prd, ruleContent, options)

	// Create AI request
	req := &Request{
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert software architect creating detailed Technical Requirements Documents (TRDs) from Product Requirements Documents (PRDs).",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Metadata: &RequestMetadata{
			Repository: prd.Repository,
			Tags:       []string{"trd_generation", "document_generation"},
		},
	}

	// Process with AI service
	startTime := time.Now()
	resp, err := g.aiService.ProcessRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("AI TRD generation failed: %w", err)
	}
	duration := time.Since(startTime)

	g.logger.Info("AI TRD generation completed",
		slog.Duration("duration", duration),
		slog.String("model", string(resp.Model)),
		slog.Int("tokens", func() int {
			if resp.TokensUsed != nil {
				return resp.TokensUsed.Total
			}
			return 0
		}()))

	// Parse AI response into TRD
	trd, err := g.parseTRDResponse(resp.Content, prd, options)
	if err != nil {
		return nil, fmt.Errorf("failed to parse TRD response: %w", err)
	}

	// Enhance TRD with technical analysis if requested
	if options.IncludeTechnicalAnalysis {
		if err := g.enhanceTechnicalAnalysis(ctx, trd); err != nil {
			g.logger.Warn("failed to enhance technical analysis", slog.String("error", err.Error()))
		}
	}

	// Generate API specifications if requested
	if options.GenerateAPISpecs {
		if err := g.generateAPISpecifications(ctx, trd, prd); err != nil {
			g.logger.Warn("failed to generate API specifications", slog.String("error", err.Error()))
		}
	}

	return trd, nil
}

// TRDGenerationOptions contains options for TRD generation
type TRDGenerationOptions struct {
	TechStack                []string          `json:"tech_stack,omitempty"`
	Architecture             string            `json:"architecture,omitempty"`
	DeploymentTarget         string            `json:"deployment_target,omitempty"`
	SecurityRequirements     []string          `json:"security_requirements,omitempty"`
	PerformanceRequirements  map[string]string `json:"performance_requirements,omitempty"`
	IncludeTechnicalAnalysis bool              `json:"include_technical_analysis"`
	GenerateAPISpecs         bool              `json:"generate_api_specs"`
	GenerateDataModels       bool              `json:"generate_data_models"`
	CustomContext            map[string]string `json:"custom_context,omitempty"`
}

// buildTRDPrompt builds the AI prompt for TRD generation
func (g *AITRDGenerator) buildTRDPrompt(prd *documents.PRDEntity, ruleContent string, options TRDGenerationOptions) string {
	// Base prompt with rules
	prompt := ruleContent + "\n\n"

	// Add PRD content
	prompt += fmt.Sprintf(`Based on the following PRD, create a comprehensive TRD:

PRD Title: %s
PRD Content:
%s

Parsed Requirements:
- Goals: %v
- Key Requirements: %v
- User Stories: %v
- Constraints: %v

`, prd.Title, prd.Content,
		prd.ParsedContent.Goals,
		prd.ParsedContent.Requirements,
		prd.ParsedContent.UserStories,
		prd.ParsedContent.Constraints)

	// Add technical context from options
	if len(options.TechStack) > 0 {
		prompt += fmt.Sprintf("Preferred Technology Stack: %s\n", strings.Join(options.TechStack, ", "))
	}
	if options.Architecture != "" {
		prompt += fmt.Sprintf("Architecture Pattern: %s\n", options.Architecture)
	}
	if options.DeploymentTarget != "" {
		prompt += fmt.Sprintf("Deployment Target: %s\n", options.DeploymentTarget)
	}
	if len(options.SecurityRequirements) > 0 {
		prompt += fmt.Sprintf("Security Requirements: %s\n", strings.Join(options.SecurityRequirements, ", "))
	}

	// Add performance requirements
	if len(options.PerformanceRequirements) > 0 {
		prompt += "Performance Requirements:\n"
		for metric, target := range options.PerformanceRequirements {
			prompt += fmt.Sprintf("- %s: %s\n", metric, target)
		}
	}

	// Add generation instructions
	prompt += `
Generate a complete TRD following the structure and guidelines provided. Include:
1. All sections specified in the rule (0-17)
2. Specific technical decisions and rationale
3. Detailed system architecture
4. Complete API specifications
5. Database schema design
6. Security implementation details
7. Performance optimization strategies
8. Deployment and monitoring plans

Format the response as a valid markdown document with proper section numbering and hierarchy.`

	return prompt
}

// parseTRDResponse parses the AI response into a TRD entity
func (g *AITRDGenerator) parseTRDResponse(content string, prd *documents.PRDEntity, options TRDGenerationOptions) (*documents.TRDEntity, error) {
	trd := &documents.TRDEntity{
		ID:           uuid.New().String(),
		PRDID:        prd.ID,
		Content:      content,
		Repository:   prd.Repository,
		Status:       documents.StatusGenerated,
		GeneratedAt:  time.Now(),
		LastModified: time.Now(),
		Version:      "1.0",
		Metadata:     make(map[string]string),
	}

	// Parse sections
	trd.Sections = documents.ParseSections(content)

	// Extract title from first section
	if len(trd.Sections) > 0 {
		trd.Title = "TRD: " + trd.Sections[0].Title
	} else {
		trd.Title = "TRD: " + prd.Title
	}

	// Extract technical details
	trd.TechnicalStack = g.extractTechnicalStack(content, options.TechStack)
	trd.Architecture = g.extractArchitecture(content, options.Architecture)
	trd.Dependencies = g.extractDependencies(content)

	// Add metadata
	trd.Metadata["source_prd"] = prd.ID
	trd.Metadata["generation_model"] = "ai"
	trd.Metadata["complexity_score"] = fmt.Sprintf("%d", prd.ComplexityScore)

	// Validate TRD
	if err := trd.Validate(); err != nil {
		return nil, fmt.Errorf("TRD validation failed: %w", err)
	}

	return trd, nil
}

// enhanceTechnicalAnalysis adds deeper technical analysis using AI
func (g *AITRDGenerator) enhanceTechnicalAnalysis(ctx context.Context, trd *documents.TRDEntity) error {
	prompt := fmt.Sprintf(`Analyze the technical aspects of this TRD and provide additional insights:

TRD Content:
%s

Provide technical analysis including:
1. Potential technical risks and mitigation strategies
2. Alternative technical approaches and trade-offs
3. Scalability considerations and growth projections
4. Security vulnerabilities and hardening recommendations
5. Performance bottlenecks and optimization opportunities
6. Integration challenges and solutions

Return as JSON:
{
	"technical_risks": [
		{
			"risk": "description",
			"impact": "high|medium|low",
			"mitigation": "strategy"
		}
	],
	"alternatives": [
		{
			"approach": "description",
			"pros": ["pro1", "pro2"],
			"cons": ["con1", "con2"],
			"recommendation": "when to use"
		}
	],
	"scalability_analysis": {
		"current_capacity": "description",
		"growth_projections": "description",
		"scaling_strategies": ["strategy1", "strategy2"]
	},
	"security_analysis": {
		"vulnerabilities": ["vuln1", "vuln2"],
		"hardening_steps": ["step1", "step2"]
	},
	"performance_analysis": {
		"bottlenecks": ["bottleneck1", "bottleneck2"],
		"optimizations": ["optimization1", "optimization2"]
	}
}`, trd.Content)

	req := &Request{
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a senior technical architect providing deep technical analysis and recommendations.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Metadata: &RequestMetadata{
			Repository: trd.Repository,
			Tags:       []string{"technical_analysis", "trd_enhancement"},
		},
	}

	resp, err := g.aiService.ProcessRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("AI technical analysis failed: %w", err)
	}

	// Parse and integrate the analysis results
	var analysis struct {
		TechnicalRisks []struct {
			Risk       string `json:"risk"`
			Impact     string `json:"impact"`
			Mitigation string `json:"mitigation"`
		} `json:"technical_risks"`
		// ... other fields omitted for brevity
	}

	if err := json.Unmarshal([]byte(resp.Content), &analysis); err != nil {
		return fmt.Errorf("failed to parse technical analysis: %w", err)
	}

	// Add technical risks to TRD metadata
	if len(analysis.TechnicalRisks) > 0 {
		risksJson, _ := json.Marshal(analysis.TechnicalRisks)
		trd.Metadata["technical_risks"] = string(risksJson)
	}

	return nil
}

// generateAPISpecifications generates detailed API specs from TRD
func (g *AITRDGenerator) generateAPISpecifications(ctx context.Context, trd *documents.TRDEntity, prd *documents.PRDEntity) error {
	prompt := fmt.Sprintf(`Based on this TRD and PRD, generate detailed API specifications:

TRD Content:
%s

PRD Requirements:
%v

Generate OpenAPI 3.0 specification including:
1. All endpoints with paths, methods, and descriptions
2. Request/response schemas with validation rules
3. Authentication and authorization details
4. Error responses and status codes
5. Rate limiting and quota information

Return as valid OpenAPI 3.0 YAML specification.`,
		trd.Content,
		prd.ParsedContent.Requirements)

	req := &Request{
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an API architect creating comprehensive OpenAPI specifications.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Metadata: &RequestMetadata{
			Repository: trd.Repository,
			Tags:       []string{"api_generation", "openapi"},
		},
	}

	resp, err := g.aiService.ProcessRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("AI API generation failed: %w", err)
	}

	// Store API spec in metadata
	trd.Metadata["api_specification"] = resp.Content

	// Create a new section for API specification
	apiSection := documents.Section{
		Title:   "API Specification (Generated)",
		Content: "```yaml\n" + resp.Content + "\n```",
		Level:   2,
		Order:   len(trd.Sections) + 1,
	}
	trd.Sections = append(trd.Sections, apiSection)

	return nil
}

// getRuleContent retrieves TRD generation rules
func (g *AITRDGenerator) getRuleContent() (string, error) {
	rule, err := g.ruleManager.GetRuleContent(documents.RuleTRDGeneration)
	if err != nil {
		// Return default rules if custom rules not found
		return g.getDefaultTRDRules(), nil
	}
	return rule, nil
}

// getDefaultTRDRules returns default TRD generation rules
func (g *AITRDGenerator) getDefaultTRDRules() string {
	// This would typically load from the create-trd.mdc file
	return `Generate a comprehensive Technical Requirements Document (TRD) following this structure:

0. **Index**
1. **Executive Summary**
2. **System Architecture**
3. **Technology Stack**
4. **Data Architecture**
5. **API Specifications**
6. **Component Design**
7. **Integration Architecture**
8. **Security Architecture**
9. **Performance Requirements**
10. **Infrastructure Requirements**
11. **Development Standards**
12. **Testing Strategy**
13. **Deployment Architecture**
14. **Monitoring & Observability**
15. **Technical Risks**
16. **Implementation Roadmap**
17. **Technical Decisions**

Each section should be detailed and implementation-ready.`
}

// Helper methods

func (g *AITRDGenerator) extractTechnicalStack(content string, preferred []string) []string {
	// Start with preferred stack
	stack := make([]string, len(preferred))
	copy(stack, preferred)

	// Extract additional technologies mentioned in content
	technologies := []string{
		"Go", "Python", "JavaScript", "TypeScript", "Java", "C++", "Rust",
		"React", "Vue", "Angular", "Node.js", "Django", "Flask", "Spring",
		"PostgreSQL", "MySQL", "MongoDB", "Redis", "Elasticsearch",
		"Docker", "Kubernetes", "AWS", "GCP", "Azure",
		"REST", "GraphQL", "gRPC", "WebSocket",
		"Kafka", "RabbitMQ", "Redis", "NATS",
	}

	contentLower := strings.ToLower(content)
	for _, tech := range technologies {
		if strings.Contains(contentLower, strings.ToLower(tech)) && !trdContains(stack, tech) {
			stack = append(stack, tech)
		}
	}

	return stack
}

func (g *AITRDGenerator) extractArchitecture(content string, preferred string) string {
	if preferred != "" {
		return preferred
	}

	patterns := map[string]string{
		"microservice":     "Microservices",
		"monolith":         "Monolithic",
		"serverless":       "Serverless",
		"event-driven":     "Event-Driven",
		"hexagonal":        "Hexagonal",
		"clean arch":       "Clean Architecture",
		"domain-driven":    "Domain-Driven Design",
		"service-oriented": "Service-Oriented Architecture",
	}

	contentLower := strings.ToLower(content)
	for key, pattern := range patterns {
		if strings.Contains(contentLower, key) {
			return pattern
		}
	}

	return "Not Specified"
}

func (g *AITRDGenerator) extractDependencies(content string) []string {
	var dependencies []string

	// Common dependency patterns
	dependencyKeywords := []string{
		"depends on", "requires", "integrates with", "uses", "connects to",
		"external service", "third-party", "library", "framework", "API",
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		for _, keyword := range dependencyKeywords {
			if strings.Contains(lineLower, keyword) {
				// Extract the dependency (simplified extraction)
				parts := strings.Split(line, keyword)
				if len(parts) > 1 {
					dep := strings.TrimSpace(parts[1])
					dep = strings.TrimSuffix(dep, ".")
					dep = strings.TrimSuffix(dep, ",")
					if dep != "" && !trdContains(dependencies, dep) {
						dependencies = append(dependencies, dep)
					}
				}
			}
		}
	}

	return dependencies
}

func trdContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
