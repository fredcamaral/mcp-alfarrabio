package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MockClient implements Client interface for testing
type MockClient struct {
	Name          string
	Model         Model
	ResponseDelay time.Duration
	ErrorRate     float32
	responseCount int
}

// NewMockClient creates a new mock AI client
func NewMockClient(name string) *MockClient {
	return &MockClient{
		Name:  name,
		Model: "mock-model-1.0",
	}
}

// ProcessRequest implements the Client interface
func (m *MockClient) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	// Simulate processing delay
	if m.ResponseDelay > 0 {
		select {
		case <-time.After(m.ResponseDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Simulate errors based on error rate
	m.responseCount++
	if m.ErrorRate > 0 && float32(m.responseCount%100) < m.ErrorRate*100 {
		return nil, fmt.Errorf("mock error: simulated failure")
	}

	// Generate mock response based on last message
	var lastMessage string
	if len(req.Messages) > 0 {
		lastMessage = req.Messages[len(req.Messages)-1].Content
	}

	response := generateMockResponse(lastMessage)

	return &Response{
		ID:      req.ID,
		Model:   m.Model,
		Content: response,
		TokensUsed: TokenUsage{
			Input:  len(lastMessage) / 4, // Rough token estimate
			Output: len(response) / 4,
			Total:  (len(lastMessage) + len(response)) / 4,
		},
		Latency: 100 * time.Millisecond,
		Quality: QualityMetrics{
			Confidence: 0.95,
			Relevance:  0.90,
			Clarity:    0.92,
			Score:      0.92,
		},
		Metadata: ResponseMetadata{
			ProcessedAt: time.Now(),
			ServerID:    "mock-server-1",
			Version:     "1.0.0",
		},
	}, nil
}

// GetModel implements the Client interface
func (m *MockClient) GetModel() Model {
	return m.Model
}

// IsHealthy implements the Client interface
func (m *MockClient) IsHealthy(ctx context.Context) error {
	return nil
}

// GetLimits implements the Client interface
func (m *MockClient) GetLimits() RateLimits {
	return RateLimits{
		RequestsPerMinute:  1000,
		TokensPerMinute:    100000,
		ConcurrentRequests: 100,
	}
}

// generateMockResponse generates a mock response based on the input
func generateMockResponse(input string) string {
	inputLower := strings.ToLower(input)

	// PRD generation
	if strings.Contains(inputLower, "prd") || strings.Contains(inputLower, "product requirements") {
		return generateMockPRD(input)
	}

	// TRD generation
	if strings.Contains(inputLower, "trd") || strings.Contains(inputLower, "technical requirements") {
		return generateMockTRD(input)
	}

	// Task generation
	if strings.Contains(inputLower, "task") || strings.Contains(inputLower, "generate tasks") {
		return generateMockTasks(input)
	}

	// Default response
	return fmt.Sprintf("Mock response for: %s", input)
}

func generateMockPRD(input string) string {
	return `# Product Requirements Document

## Project Overview
This is a mock PRD generated for testing purposes based on: ` + input + `

## Goals
1. Primary Goal: Deliver a functional system that meets user needs
2. Secondary Goal: Ensure scalability and maintainability
3. Tertiary Goal: Provide excellent user experience

## Functional Requirements
### User Management
- FR1: Users should be able to register and login
- FR2: Users should be able to manage their profiles
- FR3: Role-based access control should be implemented

### Core Features
- FR4: Main functionality as described in the input
- FR5: Data persistence and retrieval
- FR6: Real-time updates where applicable

## Non-Functional Requirements
### Performance
- NFR1: Response time under 200ms for 95% of requests
- NFR2: Support for 10,000 concurrent users
- NFR3: 99.9% uptime

### Security
- NFR4: Data encryption at rest and in transit
- NFR5: Regular security audits
- NFR6: OWASP compliance

## User Stories
1. As a user, I want to access the system easily
2. As an admin, I want to manage users and permissions
3. As a developer, I want clear API documentation

## Constraints
- Budget: $100,000
- Timeline: 6 months
- Technology: Modern web stack

## Success Criteria
- All functional requirements implemented
- Performance targets met
- User satisfaction score > 4.5/5

---
Generated: ` + time.Now().Format("2006-01-02 15:04:05")
}

func generateMockTRD(input string) string {
	return `# Technical Requirements Document

## System Architecture
Based on the requirements, we propose a microservices architecture.

## Technology Stack
### Backend
- Language: Go 1.21+
- Framework: Gin/Echo
- Database: PostgreSQL 15+
- Cache: Redis 7+

### Frontend
- Framework: React 18+
- State Management: Redux Toolkit
- UI Library: Material-UI

### Infrastructure
- Container: Docker
- Orchestration: Kubernetes
- Cloud: AWS/GCP
- CI/CD: GitHub Actions

## System Design
### High-Level Architecture

[Frontend] -----> [API Gateway] -----> [Services]
                        |                     |
                        v                     v
                    [Cache]             [Database]

### API Design
- RESTful API following OpenAPI 3.0
- GraphQL for complex queries
- WebSocket for real-time features

## Implementation Details
### Authentication
- JWT-based authentication
- OAuth2 integration
- Session management with Redis

### Database Schema
- Normalized relational design
- Indexing strategy for performance
- Migration tooling with golang-migrate

### Security Measures
- TLS 1.3 for all communications
- API rate limiting
- Input validation and sanitization
- Regular dependency updates

## Performance Considerations
- Horizontal scaling capability
- Database connection pooling
- Caching strategy (Redis)
- CDN for static assets

## Deployment Strategy
- Blue-green deployment
- Rolling updates
- Health checks and monitoring
- Automated rollback capability

---
Generated: ` + time.Now().Format("2006-01-02 15:04:05")
}

func generateMockTasks(input string) string {
	return `{
  "main_tasks": [
    {
      "task_id": "MT-001",
      "name": "Project Setup and Infrastructure",
      "phase": "setup",
      "description": "Initialize project structure, setup development environment, and configure CI/CD",
      "duration_estimate": "1 week",
      "complexity_score": 3,
      "dependencies": [],
      "deliverables": ["Project repository", "Development environment", "CI/CD pipeline"]
    },
    {
      "task_id": "MT-002",
      "name": "Backend Development",
      "phase": "development",
      "description": "Implement core backend services, APIs, and database schema",
      "duration_estimate": "4 weeks",
      "complexity_score": 8,
      "dependencies": ["MT-001"],
      "deliverables": ["API services", "Database schema", "Authentication system"]
    },
    {
      "task_id": "MT-003",
      "name": "Frontend Development",
      "phase": "development",
      "description": "Build user interface components and integrate with backend APIs",
      "duration_estimate": "3 weeks",
      "complexity_score": 6,
      "dependencies": ["MT-001"],
      "deliverables": ["UI components", "State management", "API integration"]
    },
    {
      "task_id": "MT-004",
      "name": "Testing and Quality Assurance",
      "phase": "testing",
      "description": "Comprehensive testing including unit, integration, and E2E tests",
      "duration_estimate": "2 weeks",
      "complexity_score": 5,
      "dependencies": ["MT-002", "MT-003"],
      "deliverables": ["Test suite", "Bug fixes", "Performance optimization"]
    },
    {
      "task_id": "MT-005",
      "name": "Deployment and Launch",
      "phase": "deployment",
      "description": "Deploy to production environment and perform launch activities",
      "duration_estimate": "1 week",
      "complexity_score": 4,
      "dependencies": ["MT-004"],
      "deliverables": ["Production deployment", "Monitoring setup", "Documentation"]
    }
  ],
  "sub_tasks": {
    "MT-001": [
      {
        "sub_task_id": "ST-001-001",
        "name": "Initialize Git repository",
        "estimated_hours": 2,
        "priority": "high"
      },
      {
        "sub_task_id": "ST-001-002",
        "name": "Setup Docker environment",
        "estimated_hours": 4,
        "priority": "high"
      },
      {
        "sub_task_id": "ST-001-003",
        "name": "Configure CI/CD pipeline",
        "estimated_hours": 8,
        "priority": "medium"
      }
    ]
  }
}`
}
