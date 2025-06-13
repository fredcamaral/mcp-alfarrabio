// Package ai provides mock AI client implementation for testing.
package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MockClient implements the AIClient interface for testing
type MockClient struct {
	config *BaseConfig
}

// NewMockClient creates a new mock AI client
func NewMockClient() *MockClient {
	return &MockClient{
		config: &BaseConfig{
			Model:       "mock-model",
			MaxTokens:   4096,
			Temperature: 0.7,
			Enabled:     true,
		},
	}
}

// Complete returns a mock completion response
func (c *MockClient) Complete(ctx context.Context, request CompletionRequest) (*CompletionResponse, error) {
	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Generate mock response based on the request
	content := c.generateMockContent(request)

	return &CompletionResponse{
		ID:           fmt.Sprintf("mock_%d", time.Now().UnixNano()),
		Content:      content,
		Model:        c.config.Model,
		FinishReason: "stop",
		Usage: Usage{
			PromptTokens:     len(request.Messages) * 10,
			CompletionTokens: len(content) / 4, // Rough token estimation
			TotalTokens:      len(request.Messages)*10 + len(content)/4,
		},
		ProcessingTime: 100 * time.Millisecond,
		Provider:       "mock",
		CreatedAt:      time.Now(),
	}, nil
}

// Test always returns nil for mock client
func (c *MockClient) Test(ctx context.Context) error {
	return nil
}

// GetConfig returns the mock client configuration
func (c *MockClient) GetConfig() *BaseConfig {
	return c.config
}

// generateMockContent creates mock content based on the request
func (c *MockClient) generateMockContent(request CompletionRequest) string {
	// Analyze the request to generate appropriate mock content
	if len(request.Messages) == 0 {
		return "Mock AI response"
	}

	lastMessage := request.Messages[len(request.Messages)-1].Content

	// Generate different types of mock content based on keywords
	switch {
	case contains(lastMessage, "PRD", "product", "requirements"):
		return c.generateMockPRD()
	case contains(lastMessage, "TRD", "technical", "architecture"):
		return c.generateMockTRD()
	case contains(lastMessage, "task", "main task", "sub task"):
		return c.generateMockTasks()
	case contains(lastMessage, "complex", "estimate", "difficulty"):
		return "3"
	default:
		return fmt.Sprintf("Mock AI response to: %s", lastMessage)
	}
}

// generateMockPRD creates a mock PRD
func (c *MockClient) generateMockPRD() string {
	return `# Product Requirements Document

## Introduction
This is a mock PRD generated for testing purposes.

## Features
- Core functionality implementation
- User authentication and authorization
- Data management and storage
- RESTful API endpoints

## User Stories
- As a user, I want to access the core functionality
- As a developer, I want to maintain the system
- As an admin, I want to manage the application

## Technical Requirements
- Go 1.23+ runtime environment
- PostgreSQL database
- Redis for caching
- Docker containerization`
}

// generateMockTRD creates a mock TRD
func (c *MockClient) generateMockTRD() string {
	return `# Technical Requirements Document

## Architecture
Microservices architecture with API gateway and service mesh.

## Technology Stack
- Go 1.23+ for backend services
- PostgreSQL for data persistence
- Redis for caching and session management
- Docker and Kubernetes for containerization

## Implementation
- Set up development environment
- Implement core business logic
- Create API endpoints
- Add comprehensive testing
- Deploy to production environment`
}

// generateMockTasks creates mock tasks
func (c *MockClient) generateMockTasks() string {
	return `# Generated Tasks

## MT-001: Foundation Setup
Duration: 8 hours
Dependencies: []

Set up the foundational infrastructure and development environment.

## MT-002: Core Implementation
Duration: 16 hours  
Dependencies: [MT-001]

Implement the core business logic and data models.

## MT-003: API Development
Duration: 12 hours
Dependencies: [MT-002]

Create RESTful API endpoints and integration layer.`
}

// contains checks if a string contains any of the given keywords (case insensitive)
func contains(text string, keywords ...string) bool {
	textLower := strings.ToLower(fmt.Sprintf(" %s ", text)) // Add spaces for word boundary matching
	for _, keyword := range keywords {
		keywordPattern := strings.ToLower(fmt.Sprintf(" %s ", keyword))
		if strings.Contains(textLower, keywordPattern) {
			return true
		}
	}
	return false
}
