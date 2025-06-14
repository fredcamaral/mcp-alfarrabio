// Package intelligence provides an adapter to bridge shared AI service to internal AI interface
package intelligence

import (
	"context"
	"fmt"
	"strings"

	internalAI "lerian-mcp-memory/internal/ai"
	sharedAI "lerian-mcp-memory/pkg/ai"
)

// AIServiceAdapter adapts the shared AI service to the internal AI service interface
type AIServiceAdapter struct {
	sharedService sharedAI.AIService
}

// NewAIServiceAdapter creates a new adapter
func NewAIServiceAdapter(sharedService sharedAI.AIService) *AIServiceAdapter {
	return &AIServiceAdapter{
		sharedService: sharedService,
	}
}

// ProcessRequest adapts the shared AI service to handle generic requests
func (a *AIServiceAdapter) ProcessRequest(ctx context.Context, req *internalAI.Request) (*internalAI.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Extract content from messages
	content := ""
	systemPrompt := ""
	userContent := ""

	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			systemPrompt = msg.Content
		case "user":
			userContent = msg.Content
		}
		content += msg.Content + "\n"
	}

	// Use the shared service's complexity analysis as a base for pattern analysis
	complexity, err := a.sharedService.AnalyzeComplexity(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze content: %w", err)
	}

	// Generate different responses based on the system prompt context
	var responseContent string
	if contains(systemPrompt, "pattern recognition") {
		responseContent = a.generatePatternIdentificationResponse(userContent, complexity)
	} else if contains(systemPrompt, "pattern learning") {
		responseContent = a.generatePatternLearningResponse(userContent, complexity)
	} else if contains(systemPrompt, "pattern suggestion") {
		responseContent = a.generatePatternSuggestionResponse(userContent, complexity)
	} else {
		// Default pattern response
		responseContent = a.generatePatternResponse(content, complexity)
	}

	response := &internalAI.Response{
		ID:      req.ID,
		Content: responseContent,
		TokensUsed: &internalAI.TokenUsage{
			PromptTokens:     len(content) / 4,
			CompletionTokens: len(responseContent) / 4,
			TotalTokens:      (len(content) + len(responseContent)) / 4,
			Total:            (len(content) + len(responseContent)) / 4,
		},
		FallbackUsed: false,
	}

	return response, nil
}

// generatePatternResponse generates a mock pattern analysis response
func (a *AIServiceAdapter) generatePatternResponse(_ string, complexity int) string {
	// This is a simplified pattern response generator
	// In a real implementation, this would use the AI service to generate actual pattern analysis

	patternType := "workflow"
	if complexity > 7 {
		patternType = "architectural"
	} else if complexity > 5 {
		patternType = "code"
	} else if complexity < 3 {
		patternType = "behavioral"
	}

	return fmt.Sprintf(`[{
		"name": "Auto-detected Pattern",
		"type": "%s",
		"description": "Pattern automatically detected from conversation analysis",
		"confidence": %.1f,
		"keywords": ["auto", "detected", "%s"]
	}]`, patternType, float64(complexity)/10.0, patternType)
}

// GetModel returns the model identifier (required by some interfaces)
func (a *AIServiceAdapter) GetModel() string {
	return "shared-ai-adapter"
}

// IsHealthy checks if the adapter is operational
func (a *AIServiceAdapter) IsHealthy(ctx context.Context) error {
	if a.sharedService == nil {
		return fmt.Errorf("shared AI service is nil")
	}
	return nil
}

// generatePatternIdentificationResponse generates a response for pattern identification
func (a *AIServiceAdapter) generatePatternIdentificationResponse(content string, complexity int) string {
	// Extract different pattern types based on content analysis
	patterns := []string{}

	contentLower := strings.ToLower(content)

	// Detect different pattern types
	if strings.Contains(contentLower, "error") || strings.Contains(contentLower, "bug") {
		patterns = append(patterns, `{
			"name": "Error Handling Pattern",
			"type": "error",
			"description": "Pattern for handling and resolving errors in the system",
			"confidence": 0.8,
			"keywords": ["error", "exception", "debugging", "fix"]
		}`)
	}

	if strings.Contains(contentLower, "function") || strings.Contains(contentLower, "method") {
		patterns = append(patterns, `{
			"name": "Function Definition Pattern",
			"type": "code",
			"description": "Pattern for defining and implementing functions",
			"confidence": 0.7,
			"keywords": ["function", "method", "implementation", "code"]
		}`)
	}

	if strings.Contains(contentLower, "workflow") || strings.Contains(contentLower, "process") {
		patterns = append(patterns, `{
			"name": "Workflow Process Pattern",
			"type": "workflow",
			"description": "Pattern for organizing and executing sequential processes",
			"confidence": 0.6,
			"keywords": ["workflow", "process", "sequence", "steps"]
		}`)
	}

	// Add a complexity-based pattern
	if complexity > 7 {
		patterns = append(patterns, `{
			"name": "Complex System Pattern",
			"type": "architectural",
			"description": "Pattern for managing complex system interactions",
			"confidence": 0.9,
			"keywords": ["complex", "system", "architecture", "design"]
		}`)
	}

	// If no specific patterns detected, add a general one
	if len(patterns) == 0 {
		patterns = append(patterns, fmt.Sprintf(`{
			"name": "General Pattern",
			"type": "behavioral",
			"description": "General behavioral pattern detected from content",
			"confidence": %.1f,
			"keywords": ["general", "behavior", "pattern"]
		}`, float64(complexity)/10.0))
	}

	return "[" + strings.Join(patterns, ",") + "]"
}

// generatePatternLearningResponse generates a response for pattern learning
func (a *AIServiceAdapter) generatePatternLearningResponse(content string, complexity int) string {
	contentLower := strings.ToLower(content)

	// Determine pattern characteristics based on content
	patternType := "behavioral"
	category := "general"

	if strings.Contains(contentLower, "code") || strings.Contains(contentLower, "function") {
		patternType = "code"
		category = "programming"
	} else if strings.Contains(contentLower, "error") || strings.Contains(contentLower, "bug") {
		patternType = "error"
		category = "troubleshooting"
	} else if strings.Contains(contentLower, "workflow") || strings.Contains(contentLower, "process") {
		patternType = "workflow"
		category = "process"
	} else if complexity > 7 {
		patternType = "architectural"
		category = "design"
	}

	return fmt.Sprintf(`{
		"name": "Learned Pattern from Conversation",
		"type": "%s",
		"description": "Pattern learned from user conversation and interactions",
		"category": "%s",
		"keywords": ["learned", "conversation", "%s", "adaptive"],
		"signature": {
			"complexity": %d,
			"content_length": %d,
			"detected_type": "%s"
		},
		"file_patterns": ["*.go", "*.md", "*.txt"],
		"language": "general"
	}`, patternType, category, patternType, complexity, len(content), patternType)
}

// generatePatternSuggestionResponse generates a response for pattern suggestions
func (a *AIServiceAdapter) generatePatternSuggestionResponse(content string, complexity int) string {
	contentLower := strings.ToLower(content)
	suggestions := []string{}

	// Suggest patterns based on content context
	if strings.Contains(contentLower, "implement") || strings.Contains(contentLower, "create") {
		suggestions = append(suggestions, `{
			"name": "Implementation Pattern",
			"type": "code",
			"description": "Suggested pattern for implementing new functionality",
			"confidence": 0.8,
			"keywords": ["implement", "create", "build", "develop"]
		}`)
	}

	if strings.Contains(contentLower, "test") || strings.Contains(contentLower, "verify") {
		suggestions = append(suggestions, `{
			"name": "Testing Pattern",
			"type": "workflow",
			"description": "Suggested pattern for testing and verification",
			"confidence": 0.7,
			"keywords": ["test", "verify", "validate", "check"]
		}`)
	}

	if strings.Contains(contentLower, "optimize") || strings.Contains(contentLower, "improve") {
		suggestions = append(suggestions, `{
			"name": "Optimization Pattern",
			"type": "optimization",
			"description": "Suggested pattern for performance optimization",
			"confidence": 0.6,
			"keywords": ["optimize", "improve", "performance", "efficiency"]
		}`)
	}

	// Always suggest a context-appropriate pattern
	suggestions = append(suggestions, fmt.Sprintf(`{
		"name": "Context-Aware Pattern",
		"type": "behavioral",
		"description": "Pattern suggestion based on current conversation context",
		"confidence": %.1f,
		"keywords": ["context", "adaptive", "responsive"]
	}`, float64(complexity)/10.0))

	return "[" + strings.Join(suggestions, ",") + "]"
}

// contains checks if a string contains a substring (case-insensitive)
func contains(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}
