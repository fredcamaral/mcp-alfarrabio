package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory-cli/internal/domain/ports"
)

func TestNewHTTPAIService(t *testing.T) {
	config := &AIServiceConfig{
		BaseURL: "http://localhost:9080",
		APIKey:  "test-key",
		Timeout: 30 * time.Second,
	}

	service := NewHTTPAIService(config)

	assert.NotNil(t, service)
	assert.Equal(t, config.BaseURL, service.baseURL)
	assert.Equal(t, config.APIKey, service.apiKey)
	assert.Equal(t, config.Timeout, service.timeout)
}

func TestNewHTTPAIService_DefaultTimeout(t *testing.T) {
	config := &AIServiceConfig{
		BaseURL: "http://localhost:9080",
		APIKey:  "test-key",
		// No timeout specified
	}

	service := NewHTTPAIService(config)

	assert.Equal(t, 30*time.Second, service.timeout)
}

func TestHTTPAIService_GeneratePRD_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/ai/generate/prd", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "prd-123",
			"title": "Test Project",
			"description": "A test project description",
			"features": ["feature1", "feature2"],
			"user_stories": ["story1", "story2"],
			"content": "PRD content here",
			"model_used": "claude",
			"generated_at": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	request := &ports.PRDGenerationRequest{
		UserInputs:  []string{"Create a web application"},
		Repository:  "test-repo",
		ProjectType: "web-app",
		Preferences: ports.UserPreferences{
			PreferredTaskSize:   "medium",
			PreferredComplexity: "low",
			IncludeTests:        true,
			IncludeDocs:         true,
		},
	}

	ctx := context.Background()
	response, err := service.GeneratePRD(ctx, request)

	require.NoError(t, err)
	assert.Equal(t, "prd-123", response.ID)
	assert.Equal(t, "Test Project", response.Title)
	assert.Equal(t, "A test project description", response.Description)
	assert.Equal(t, []string{"feature1", "feature2"}, response.Features)
	assert.Equal(t, []string{"story1", "story2"}, response.UserStories)
	assert.Equal(t, "claude", response.ModelUsed)
}

func TestHTTPAIService_GeneratePRD_ServerError(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	request := &ports.PRDGenerationRequest{
		UserInputs:  []string{"Create a web application"},
		Repository:  "test-repo",
		ProjectType: "web-app",
	}

	ctx := context.Background()
	response, err := service.GeneratePRD(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "AI service returned status 500")
}

func TestHTTPAIService_GenerateTRD_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/ai/generate/trd", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "trd-123",
			"prd_id": "prd-123",
			"title": "Technical Requirements for Test Project",
			"architecture": "Microservices architecture",
			"tech_stack": ["Go", "React", "PostgreSQL"],
			"requirements": ["req1", "req2"],
			"implementation": ["step1", "step2"],
			"content": "TRD content here",
			"model_used": "claude",
			"generated_at": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	request := &ports.TRDGenerationRequest{
		PRDID:       "prd-123",
		PRDContent:  "PRD content for analysis",
		Repository:  "test-repo",
		ProjectType: "web-app",
	}

	ctx := context.Background()
	response, err := service.GenerateTRD(ctx, request)

	require.NoError(t, err)
	assert.Equal(t, "trd-123", response.ID)
	assert.Equal(t, "prd-123", response.PRDID)
	assert.Equal(t, "Technical Requirements for Test Project", response.Title)
	assert.Equal(t, "Microservices architecture", response.Architecture)
	assert.Equal(t, []string{"Go", "React", "PostgreSQL"}, response.TechStack)
}

func TestHTTPAIService_GenerateMainTasks_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/ai/generate/main-tasks", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tasks": [
				{
					"id": "MT-001",
					"name": "Setup Project",
					"description": "Initialize project structure",
					"phase": "setup",
					"duration": "2 days",
					"atomic_validation": true,
					"dependencies": [],
					"content": "Setup project structure and dependencies"
				},
				{
					"id": "MT-002",
					"name": "Core Implementation",
					"description": "Implement core functionality",
					"phase": "development",
					"duration": "1 week",
					"atomic_validation": true,
					"dependencies": ["MT-001"],
					"content": "Implement core business logic"
				}
			],
			"model_used": "claude",
			"generated_at": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	request := &ports.MainTaskGenerationRequest{
		TRDID:      "trd-123",
		TRDContent: "TRD content for task generation",
		Repository: "test-repo",
	}

	ctx := context.Background()
	response, err := service.GenerateMainTasks(ctx, request)

	require.NoError(t, err)
	assert.Len(t, response.Tasks, 2)
	assert.Equal(t, "MT-001", response.Tasks[0].ID)
	assert.Equal(t, "Setup Project", response.Tasks[0].Name)
	assert.Equal(t, "claude", response.ModelUsed)
}

func TestHTTPAIService_GenerateSubTasks_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/ai/generate/sub-tasks", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tasks": [
				{
					"id": "ST-MT-001-001",
					"parent_task_id": "MT-001",
					"name": "Create project structure",
					"duration_hours": 2,
					"implementation_type": "setup",
					"deliverables": ["project files", "configuration"],
					"acceptance_criteria": ["structure is created", "builds successfully"],
					"dependencies": [],
					"content": "Create initial project structure"
				}
			],
			"model_used": "claude",
			"generated_at": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	request := &ports.SubTaskGenerationRequest{
		MainTaskID:      "MT-001",
		MainTaskContent: "Setup project structure and dependencies",
		Repository:      "test-repo",
	}

	ctx := context.Background()
	response, err := service.GenerateSubTasks(ctx, request)

	require.NoError(t, err)
	assert.Len(t, response.Tasks, 1)
	assert.Equal(t, "ST-MT-001-001", response.Tasks[0].ID)
	assert.Equal(t, "MT-001", response.Tasks[0].ParentTaskID)
	assert.Equal(t, 2, response.Tasks[0].Duration)
	assert.Equal(t, "claude", response.ModelUsed)
}

func TestHTTPAIService_AnalyzeContent_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/ai/analyze/content", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "analysis-123",
			"summary": "This is a web application project",
			"key_features": ["user auth", "data management"],
			"technical_requirements": ["database", "api"],
			"dependencies": ["React", "Node.js"],
			"complexity": {
				"overall": "medium",
				"score": 6.5,
				"factors": ["authentication", "data complexity"],
				"estimated_hours": 40,
				"confidence": 0.8,
				"categories": {
					"technical": 7.0,
					"business": 5.0
				}
			},
			"sections": [
				{
					"id": "section-1",
					"title": "Overview",
					"content": "Project overview content",
					"type": "overview",
					"order": 1
				}
			],
			"model_used": "claude",
			"processed_at": "2024-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	request := &ports.ContentAnalysisRequest{
		Content: "Create a web application with user authentication",
		Type:    "requirement",
	}

	ctx := context.Background()
	response, err := service.AnalyzeContent(ctx, request)

	require.NoError(t, err)
	assert.Equal(t, "analysis-123", response.ID)
	assert.Equal(t, "This is a web application project", response.Summary)
	assert.Equal(t, []string{"user auth", "data management"}, response.KeyFeatures)
	assert.Equal(t, "medium", response.Complexity.Overall)
	assert.Equal(t, 6.5, response.Complexity.Score)
	assert.Equal(t, 40, response.Complexity.EstimatedHours)
}

func TestHTTPAIService_EstimateComplexity_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/ai/analyze/complexity", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"overall": "high",
			"score": 8.5,
			"factors": ["complex algorithms", "multiple integrations"],
			"estimated_hours": 80,
			"confidence": 0.9,
			"categories": {
				"technical": 9.0,
				"business": 7.0,
				"integration": 8.0
			}
		}`))
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	ctx := context.Background()
	response, err := service.EstimateComplexity(ctx, "Complex ML algorithm implementation")

	require.NoError(t, err)
	assert.Equal(t, "high", response.Overall)
	assert.Equal(t, 8.5, response.Score)
	assert.Equal(t, 80, response.EstimatedHours)
	assert.Equal(t, 0.9, response.Confidence)
	assert.Equal(t, 9.0, response.Categories["technical"])
}

func TestHTTPAIService_TestConnection_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	ctx := context.Background()
	err := service.TestConnection(ctx)

	assert.NoError(t, err)
}

func TestHTTPAIService_TestConnection_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	ctx := context.Background()
	err := service.TestConnection(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AI service connection test failed")
}

func TestHTTPAIService_IsOnline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	online := service.IsOnline()
	assert.True(t, online)

	// Close server and test offline
	server.Close()
	online = service.IsOnline()
	assert.False(t, online)
}

func TestHTTPAIService_GetAvailableModels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/ai/models", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"models": ["claude", "openai", "perplexity"]
		}`))
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	models := service.GetAvailableModels()

	assert.Equal(t, []string{"claude", "openai", "perplexity"}, models)
}

func TestHTTPAIService_GetAvailableModels_Fallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &AIServiceConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Timeout: 10 * time.Second,
	}
	service := NewHTTPAIService(config)

	models := service.GetAvailableModels()

	// Should return default models when request fails
	assert.Equal(t, []string{"claude", "openai", "perplexity"}, models)
}

func TestWithOptions(t *testing.T) {
	tests := []struct {
		name     string
		request  interface{}
		options  *RequestOptions
		expected func(t *testing.T, result interface{})
	}{
		{
			name: "PRD request with options",
			request: &ports.PRDGenerationRequest{
				UserInputs: []string{"test"},
			},
			options: &RequestOptions{
				Model: "claude",
				Metadata: map[string]string{
					"test_key": "test_value",
				},
			},
			expected: func(t *testing.T, result interface{}) {
				req := result.(*ports.PRDGenerationRequest)
				assert.Equal(t, "claude", req.Model)
				assert.Equal(t, "test_value", req.Metadata["test_key"])
			},
		},
		{
			name: "TRD request with options",
			request: &ports.TRDGenerationRequest{
				PRDID: "test",
			},
			options: &RequestOptions{
				Model: "openai",
			},
			expected: func(t *testing.T, result interface{}) {
				req := result.(*ports.TRDGenerationRequest)
				assert.Equal(t, "openai", req.Model)
			},
		},
		{
			name:    "nil options",
			request: &ports.PRDGenerationRequest{},
			options: nil,
			expected: func(t *testing.T, result interface{}) {
				// Should return unchanged request
				assert.IsType(t, &ports.PRDGenerationRequest{}, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithOptions(tt.request, tt.options)
			tt.expected(t, result)
		})
	}
}

func TestHTTPAIService_WithRetry(t *testing.T) {
	service := &HTTPAIService{}

	t.Run("success on first try", func(t *testing.T) {
		callCount := 0
		config := &RetryConfig{
			MaxRetries: 3,
			Backoff:    1 * time.Millisecond,
		}

		err := service.WithRetry(config, func() error {
			callCount++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("success after retries", func(t *testing.T) {
		callCount := 0
		config := &RetryConfig{
			MaxRetries: 3,
			Backoff:    1 * time.Millisecond,
		}

		err := service.WithRetry(config, func() error {
			callCount++
			if callCount < 3 {
				return assert.AnError
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, callCount)
	})

	t.Run("failure after max retries", func(t *testing.T) {
		callCount := 0
		config := &RetryConfig{
			MaxRetries: 2,
			Backoff:    1 * time.Millisecond,
		}

		err := service.WithRetry(config, func() error {
			callCount++
			return assert.AnError
		})

		assert.Error(t, err)
		assert.Equal(t, 3, callCount) // Initial + 2 retries
		assert.Contains(t, err.Error(), "request failed after 3 attempts")
	})

	t.Run("nil config", func(t *testing.T) {
		callCount := 0

		err := service.WithRetry(nil, func() error {
			callCount++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, callCount)
	})
}
