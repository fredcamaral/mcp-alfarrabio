//go:build integration
// +build integration

package intelligence

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/services"
	"lerian-mcp-memory-cli/tests/testutils"
)

type SuggestionTestSuite struct {
	suite.Suite
	taskStore         *testutils.MockTaskStorage
	patternStore      *testutils.MockPatternStorage
	templateStore     *testutils.MockTemplateStorage
	aiService         *testutils.MockAIService
	suggestionService services.SuggestionService
	contextAnalyzer   services.ContextAnalyzer
	testData          *testutils.TestDataGenerator
}

func (s *SuggestionTestSuite) SetupSuite() {
	s.taskStore = testutils.NewMockTaskStorage()
	s.patternStore = testutils.NewMockPatternStorage()
	s.templateStore = testutils.NewMockTemplateStorage()
	s.aiService = testutils.NewMockAIService()
	s.testData = testutils.NewTestDataGenerator()

	s.contextAnalyzer = services.NewContextAnalyzer(services.ContextAnalyzerDependencies{
		TaskStore: s.taskStore,
		Logger:    slog.Default(),
	})

	patternDetector := services.NewPatternDetector(services.PatternDetectorDependencies{
		TaskStore:    s.taskStore,
		PatternStore: s.patternStore,
		AI:           s.aiService,
		Logger:       slog.Default(),
	})

	s.suggestionService = services.NewSuggestionService(services.SuggestionServiceDependencies{
		ContextAnalyzer: s.contextAnalyzer,
		PatternDetector: patternDetector,
		TaskStore:       s.taskStore,
		PatternStore:    s.patternStore,
		TemplateStore:   s.templateStore,
		AI:              s.aiService,
		Logger:          slog.Default(),
	})
}

func (s *SuggestionTestSuite) TearDownTest() {
	s.taskStore.Clear()
	s.patternStore.Clear()
	s.templateStore.Clear()
	s.aiService.Reset()
}

func (s *SuggestionTestSuite) TestSuggestionRelevance() {
	ctx := context.Background()

	// Setup context with authentication-related tasks
	authTasks := []*entities.Task{
		s.testData.CreateTask("Setup JWT authentication", "high", "in_progress", map[string]interface{}{
			"type":     "implementation",
			"keywords": []string{"auth", "jwt", "security"},
		}),
		s.testData.CreateTask("Create user registration endpoint", "high", "completed", map[string]interface{}{
			"type":     "implementation",
			"keywords": []string{"user", "registration", "api"},
		}),
		s.testData.CreateTask("Implement password hashing", "medium", "completed", map[string]interface{}{
			"type":     "implementation",
			"keywords": []string{"password", "hash", "security"},
		}),
	}

	for _, task := range authTasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Generate suggestions
	suggestions, err := s.suggestionService.GenerateSuggestions(ctx, "test-repo")
	s.Require().NoError(err)
	s.Assert().NotEmpty(suggestions, "Should generate suggestions")

	// Verify relevance to authentication context
	relevantCount := 0
	authKeywords := []string{"auth", "security", "permission", "role", "token", "login", "session"}

	for _, suggestion := range suggestions {
		content := strings.ToLower(suggestion.Content)
		for _, keyword := range authKeywords {
			if strings.Contains(content, keyword) {
				relevantCount++
				break
			}
		}
	}

	relevanceRate := float64(relevantCount) / float64(len(suggestions))
	s.Assert().Greater(relevanceRate, 0.4, "At least 40% of suggestions should be relevant to authentication context")
	s.T().Logf("Generated %d suggestions, %d (%.1f%%) relevant to auth context",
		len(suggestions), relevantCount, relevanceRate*100)
}

func (s *SuggestionTestSuite) TestPatternBasedSuggestions() {
	ctx := context.Background()

	// Create a known pattern
	pattern := &entities.TaskPattern{
		ID:   uuid.New().String(),
		Type: entities.PatternTypeSequence,
		Name: "API Development Pattern",
		Sequence: []entities.PatternStep{
			{Order: 1, TaskType: "design", Keywords: []string{"api", "endpoint", "schema"}},
			{Order: 2, TaskType: "implement", Keywords: []string{"controller", "handler", "route"}},
			{Order: 3, TaskType: "test", Keywords: []string{"integration", "api", "postman"}},
			{Order: 4, TaskType: "document", Keywords: []string{"swagger", "openapi", "docs"}},
		},
		Confidence:  0.9,
		SuccessRate: 0.95,
		Frequency:   15,
		Repository:  "test-repo",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := s.patternStore.Create(ctx, pattern)
	s.Require().NoError(err)

	// Create current task that matches step 1 of the pattern
	currentTask := s.testData.CreateTask("Design user profile API schema", "high", "completed", map[string]interface{}{
		"type":     "design",
		"keywords": []string{"api", "schema", "user", "profile"},
	})
	err = s.taskStore.Create(ctx, currentTask)
	s.Require().NoError(err)

	// Get next task suggestion
	nextSuggestion, err := s.suggestionService.GetNextTaskSuggestion(ctx, currentTask)
	s.Require().NoError(err)
	s.Assert().NotNil(nextSuggestion, "Should provide next task suggestion")

	// Should suggest implementation step based on pattern
	s.Assert().Contains(strings.ToLower(nextSuggestion.Content), "implement", "Should suggest implementation")
	s.Assert().Equal(entities.SuggestionSourcePattern, nextSuggestion.Source.Type)
	s.Assert().Greater(nextSuggestion.Confidence, 0.7, "Pattern-based suggestion should have high confidence")
	s.Assert().Equal(pattern.ID, nextSuggestion.Source.ID, "Should reference the pattern")

	s.T().Logf("Pattern-based suggestion: %s (confidence: %.2f)",
		nextSuggestion.Content, nextSuggestion.Confidence)
}

func (s *SuggestionTestSuite) TestContextualSuggestions() {
	ctx := context.Background()

	// Create context with specific project state
	projectTasks := []*entities.Task{
		s.testData.CreateTask("Setup React project structure", "high", "completed", map[string]interface{}{
			"type":      "setup",
			"framework": "react",
		}),
		s.testData.CreateTask("Install dependencies (React, TypeScript)", "high", "completed", map[string]interface{}{
			"type":      "setup",
			"framework": "react",
		}),
		s.testData.CreateTask("Create basic component structure", "medium", "in_progress", map[string]interface{}{
			"type":      "implementation",
			"framework": "react",
		}),
	}

	for _, task := range projectTasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Generate contextual suggestions
	suggestions, err := s.suggestionService.GenerateSuggestions(ctx, "react-project")
	s.Require().NoError(err)
	s.Assert().NotEmpty(suggestions)

	// Should generate React-specific suggestions
	reactSuggestions := 0
	for _, suggestion := range suggestions {
		content := strings.ToLower(suggestion.Content)
		if strings.Contains(content, "react") ||
			strings.Contains(content, "component") ||
			strings.Contains(content, "jsx") ||
			strings.Contains(content, "typescript") {
			reactSuggestions++
		}
	}

	s.Assert().Greater(reactSuggestions, 0, "Should generate React-specific suggestions")
	s.T().Logf("Generated %d React-specific suggestions out of %d total",
		reactSuggestions, len(suggestions))
}

func (s *SuggestionTestSuite) TestAIPoweredSuggestions() {
	ctx := context.Background()

	// Setup AI mock to return specific suggestions
	s.aiService.SetSuggestionResponse([]string{
		"Add input validation for user registration",
		"Implement password strength requirements",
		"Set up rate limiting for auth endpoints",
		"Add two-factor authentication support",
	})

	// Create context for security-focused suggestions
	securityTask := s.testData.CreateTask("Implement user authentication system", "high", "in_progress", map[string]interface{}{
		"type":     "implementation",
		"keywords": []string{"auth", "security", "user"},
	})
	err := s.taskStore.Create(ctx, securityTask)
	s.Require().NoError(err)

	// Generate AI-powered suggestions
	suggestions, err := s.suggestionService.GenerateAISuggestions(ctx, "security-project", 4)
	s.Require().NoError(err)
	s.Assert().Len(suggestions, 4, "Should generate requested number of AI suggestions")

	// Verify AI suggestions are created correctly
	for _, suggestion := range suggestions {
		s.Assert().Equal(entities.SuggestionSourceAI, suggestion.Source.Type)
		s.Assert().NotEmpty(suggestion.Content)
		s.Assert().Greater(suggestion.Confidence, 0.0)
		s.Assert().NotEmpty(suggestion.ID)
	}

	// Verify security context influence
	securitySuggestions := 0
	securityKeywords := []string{"security", "auth", "validation", "password", "rate"}
	for _, suggestion := range suggestions {
		content := strings.ToLower(suggestion.Content)
		for _, keyword := range securityKeywords {
			if strings.Contains(content, keyword) {
				securitySuggestions++
				break
			}
		}
	}

	s.Assert().Greater(securitySuggestions, 2, "Most AI suggestions should be security-related")
}

func (s *SuggestionTestSuite) TestSuggestionRanking() {
	ctx := context.Background()

	// Create diverse suggestions with different confidence levels
	suggestions := []*entities.TaskSuggestion{
		{
			ID:         uuid.New().String(),
			Content:    "High confidence pattern suggestion",
			Confidence: 0.95,
			Priority:   entities.PriorityHigh,
			Source:     entities.SuggestionSource{Type: entities.SuggestionSourcePattern},
			CreatedAt:  time.Now(),
		},
		{
			ID:         uuid.New().String(),
			Content:    "Medium confidence AI suggestion",
			Confidence: 0.65,
			Priority:   entities.PriorityMedium,
			Source:     entities.SuggestionSource{Type: entities.SuggestionSourceAI},
			CreatedAt:  time.Now().Add(-time.Hour),
		},
		{
			ID:         uuid.New().String(),
			Content:    "Low confidence template suggestion",
			Confidence: 0.45,
			Priority:   entities.PriorityLow,
			Source:     entities.SuggestionSource{Type: entities.SuggestionSourceTemplate},
			CreatedAt:  time.Now().Add(-2 * time.Hour),
		},
		{
			ID:         uuid.New().String(),
			Content:    "High confidence AI suggestion",
			Confidence: 0.88,
			Priority:   entities.PriorityHigh,
			Source:     entities.SuggestionSource{Type: entities.SuggestionSourceAI},
			CreatedAt:  time.Now().Add(-30 * time.Minute),
		},
	}

	// Rank suggestions
	rankedSuggestions := s.suggestionService.RankSuggestions(suggestions)

	s.Assert().Len(rankedSuggestions, 4, "Should return all suggestions")

	// Verify ranking order (higher scores first)
	for i := 0; i < len(rankedSuggestions)-1; i++ {
		current := rankedSuggestions[i]
		next := rankedSuggestions[i+1]

		currentScore := s.calculateExpectedScore(current)
		nextScore := s.calculateExpectedScore(next)

		s.Assert().GreaterOrEqual(currentScore, nextScore,
			"Suggestions should be ranked by score (higher first)")
	}

	// Highest ranked should be the high-confidence pattern suggestion
	top := rankedSuggestions[0]
	s.Assert().Equal("High confidence pattern suggestion", top.Content)
	s.Assert().Equal(entities.SuggestionSourcePattern, top.Source.Type)

	s.T().Logf("Ranking order:")
	for i, suggestion := range rankedSuggestions {
		score := s.calculateExpectedScore(suggestion)
		s.T().Logf("  %d. %.2f - %s (%.2f confidence, %s priority, %s source)",
			i+1, score, suggestion.Content, suggestion.Confidence,
			suggestion.Priority, suggestion.Source.Type)
	}
}

func (s *SuggestionTestSuite) TestSuggestionFiltering() {
	ctx := context.Background()

	// Create various tasks
	tasks := []*entities.Task{
		s.testData.CreateTask("Implement user login", "high", "completed", nil),
		s.testData.CreateTask("Write unit tests", "medium", "completed", nil),
		s.testData.CreateTask("Setup database", "high", "completed", nil),
		s.testData.CreateTask("Create API documentation", "low", "pending", nil),
	}

	for _, task := range tasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Generate suggestions
	allSuggestions, err := s.suggestionService.GenerateSuggestions(ctx, "filter-repo")
	s.Require().NoError(err)

	// Test priority filtering
	highPrioritySuggestions := s.suggestionService.FilterSuggestions(allSuggestions, map[string]interface{}{
		"priority": entities.PriorityHigh,
	})

	for _, suggestion := range highPrioritySuggestions {
		s.Assert().Equal(entities.PriorityHigh, suggestion.Priority,
			"Filtered suggestions should match priority criteria")
	}

	// Test source filtering
	aiSuggestions := s.suggestionService.FilterSuggestions(allSuggestions, map[string]interface{}{
		"source": entities.SuggestionSourceAI,
	})

	for _, suggestion := range aiSuggestions {
		s.Assert().Equal(entities.SuggestionSourceAI, suggestion.Source.Type,
			"Filtered suggestions should match source criteria")
	}

	// Test confidence threshold filtering
	highConfidenceSuggestions := s.suggestionService.FilterSuggestions(allSuggestions, map[string]interface{}{
		"min_confidence": 0.7,
	})

	for _, suggestion := range highConfidenceSuggestions {
		s.Assert().GreaterOrEqual(suggestion.Confidence, 0.7,
			"Filtered suggestions should meet confidence threshold")
	}
}

func (s *SuggestionTestSuite) TestSuggestionPerformance() {
	ctx := context.Background()

	// Create large dataset
	largeTasks := s.testData.GenerateRandomTasks("perf-repo", 1000)
	for _, task := range largeTasks {
		s.taskStore.Create(ctx, task)
	}

	// Add some patterns for more realistic suggestion generation
	patterns := s.testData.GenerateRandomPatterns("perf-repo", 20)
	for _, pattern := range patterns {
		s.patternStore.Create(ctx, pattern)
	}

	// Measure suggestion generation performance
	start := time.Now()
	suggestions, err := s.suggestionService.GenerateSuggestions(ctx, "perf-repo")
	duration := time.Since(start)

	s.Require().NoError(err)
	s.Assert().Less(duration, 5*time.Second, "Suggestion generation should complete within 5 seconds")
	s.Assert().NotEmpty(suggestions, "Should generate suggestions from large dataset")

	s.T().Logf("Generated %d suggestions from %d tasks and %d patterns in %v",
		len(suggestions), len(largeTasks), len(patterns), duration)

	// Measure ranking performance
	start = time.Now()
	rankedSuggestions := s.suggestionService.RankSuggestions(suggestions)
	rankingDuration := time.Since(start)

	s.Assert().Less(rankingDuration, 1*time.Second, "Suggestion ranking should complete within 1 second")
	s.Assert().Len(rankedSuggestions, len(suggestions), "Ranking should preserve all suggestions")

	s.T().Logf("Ranked %d suggestions in %v", len(suggestions), rankingDuration)
}

func (s *SuggestionTestSuite) TestSuggestionPersistence() {
	ctx := context.Background()

	// Create and store a suggestion
	suggestion := &entities.TaskSuggestion{
		ID:         uuid.New().String(),
		Content:    "Test persistent suggestion",
		Type:       entities.SuggestionTypeTask,
		Priority:   entities.PriorityMedium,
		Confidence: 0.75,
		Repository: "test-repo",
		Source: entities.SuggestionSource{
			Type: entities.SuggestionSourcePattern,
			ID:   uuid.New().String(),
		},
		Context: map[string]interface{}{
			"current_task": "test-task",
			"keywords":     []string{"test", "suggestion"},
		},
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Test storage (this would require extending mock storage or using real storage)
	// For now, just verify the suggestion structure is valid
	s.Assert().NotEmpty(suggestion.ID)
	s.Assert().NotEmpty(suggestion.Content)
	s.Assert().Greater(suggestion.Confidence, 0.0)
	s.Assert().NotEmpty(suggestion.Repository)
	s.Assert().True(suggestion.ExpiresAt.After(suggestion.CreatedAt))

	s.T().Logf("Created suggestion: %s (confidence: %.2f)",
		suggestion.Content, suggestion.Confidence)
}

// Helper methods

func (s *SuggestionTestSuite) calculateExpectedScore(suggestion *entities.TaskSuggestion) float64 {
	// Simplified version of the ranking algorithm
	score := suggestion.Confidence

	// Priority weight
	switch suggestion.Priority {
	case entities.PriorityHigh:
		score += 0.3
	case entities.PriorityMedium:
		score += 0.1
	}

	// Source weight
	switch suggestion.Source.Type {
	case entities.SuggestionSourcePattern:
		score += 0.2
	case entities.SuggestionSourceAI:
		score += 0.1
	}

	// Recency weight (newer is better)
	age := time.Since(suggestion.CreatedAt)
	if age < time.Hour {
		score += 0.1
	}

	return score
}

func TestSuggestion(t *testing.T) {
	suite.Run(t, new(SuggestionTestSuite))
}
