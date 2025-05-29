package intelligence

import (
	"context"
	"math"
	"testing"
	"time"

	"mcp-memory/pkg/types"
)

func TestMultiRepoEngineCreation(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	multiRepoEngine := NewMultiRepoEngine(patternEngine, graphBuilder, learningEngine)

	if multiRepoEngine == nil {
		t.Fatal("Expected multi-repo engine to be created")
	}

	if multiRepoEngine.patternEngine != patternEngine {
		t.Error("Expected pattern engine to be set")
	}

	if multiRepoEngine.knowledgeGraph != graphBuilder {
		t.Error("Expected knowledge graph to be set")
	}

	if multiRepoEngine.learningEngine != learningEngine {
		t.Error("Expected learning engine to be set")
	}

	if len(multiRepoEngine.repositories) != 0 {
		t.Error("Expected empty repositories map initially")
	}
}

func TestAddRepository(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	multiRepoEngine := NewMultiRepoEngine(patternEngine, graphBuilder, learningEngine)

	repo := &RepositoryContext{
		ID:           "test-repo",
		Name:         "Test Repository",
		URL:          "https://github.com/test/repo",
		Language:     "Go",
		Framework:    "Gin",
		Architecture: "microservices",
		TeamSize:     5,
		TechStack:    []string{"go", "postgresql", "docker"},
		Configuration: map[string]any{
			"go_version": "1.21",
		},
	}

	err := multiRepoEngine.AddRepository(context.Background(), repo)
	if err != nil {
		t.Fatalf("Expected no error adding repository, got %v", err)
	}

	if len(multiRepoEngine.repositories) != 1 {
		t.Errorf("Expected 1 repository, got %d", len(multiRepoEngine.repositories))
	}

	storedRepo, exists := multiRepoEngine.repositories["test-repo"]
	if !exists {
		t.Error("Expected repository to be stored")
	}

	if storedRepo.Name != "Test Repository" {
		t.Errorf("Expected repository name 'Test Repository', got '%s'", storedRepo.Name)
	}
}

func TestUpdateRepositoryContext(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	multiRepoEngine := NewMultiRepoEngine(patternEngine, graphBuilder, learningEngine)

	// Create test chunks with Go-related content
	chunks := []types.ConversationChunk{
		{
			ID:      "chunk1",
			Content: "I'm working on a Go application using PostgreSQL and Docker",
			Type:    types.ChunkTypeProblem,
		},
		{
			ID:      "chunk2",
			Content: "The debugging process worked well, issue resolved successfully",
			Type:    types.ChunkTypeSolution,
		},
	}

	err := multiRepoEngine.UpdateRepositoryContext(context.Background(), "new-repo", chunks)
	if err != nil {
		t.Fatalf("Expected no error updating repository context, got %v", err)
	}

	// Check that repository was created
	repo, exists := multiRepoEngine.repositories["new-repo"]
	if !exists {
		t.Error("Expected repository to be created")
	}

	if repo.TotalSessions != 1 {
		t.Errorf("Expected 1 session, got %d", repo.TotalSessions)
	}

	// Check that tech stack was extracted
	expectedTech := []string{"go", "postgresql", "docker"}
	foundTech := 0
	for _, expected := range expectedTech {
		for _, actual := range repo.TechStack {
			if expected == actual {
				foundTech++
				break
			}
		}
	}

	if foundTech == 0 {
		t.Error("Expected to extract some tech stack items")
	}
}

func TestAnalyzeCrossRepoPatterns(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	multiRepoEngine := NewMultiRepoEngine(patternEngine, graphBuilder, learningEngine)

	// Add repositories with common patterns
	repo1 := &RepositoryContext{
		ID:             "repo1",
		Name:           "Repository 1",
		Language:       "Go",
		Framework:      "Gin",
		CommonPatterns: []string{"debugging", "testing", "deployment"},
		TechStack:      []string{"go", "postgresql"},
	}

	repo2 := &RepositoryContext{
		ID:             "repo2",
		Name:           "Repository 2",
		Language:       "Go",
		Framework:      "Echo",
		CommonPatterns: []string{"debugging", "performance_optimization", "deployment"},
		TechStack:      []string{"go", "mysql"},
	}

	repo3 := &RepositoryContext{
		ID:             "repo3",
		Name:           "Repository 3",
		Language:       "Python",
		Framework:      "Flask",
		CommonPatterns: []string{"debugging", "api_design"},
		TechStack:      []string{"python", "postgresql"},
	}

	multiRepoEngine.repositories[repo1.ID] = repo1
	multiRepoEngine.repositories[repo2.ID] = repo2
	multiRepoEngine.repositories[repo3.ID] = repo3

	// Force analysis by setting last analysis to past
	multiRepoEngine.lastAnalysis = time.Now().Add(-25 * time.Hour)

	err := multiRepoEngine.AnalyzeCrossRepoPatterns(context.Background())
	if err != nil {
		t.Fatalf("Expected no error analyzing cross-repo patterns, got %v", err)
	}

	// Check that cross-repo patterns were identified
	if len(multiRepoEngine.crossRepoPatterns) == 0 {
		t.Error("Expected to find cross-repo patterns")
	}

	// Look for the debugging pattern that appears in all three repos
	foundDebuggingPattern := false
	for _, pattern := range multiRepoEngine.crossRepoPatterns {
		if pattern.Name == "debugging" {
			foundDebuggingPattern = true
			if len(pattern.Repositories) != 3 {
				t.Errorf("Expected debugging pattern in 3 repositories, got %d", len(pattern.Repositories))
			}
			if pattern.Frequency != 3 {
				t.Errorf("Expected debugging pattern frequency 3, got %d", pattern.Frequency)
			}
		}
	}

	if !foundDebuggingPattern {
		t.Error("Expected to find debugging pattern across repositories")
	}
}

func TestQueryMultiRepo(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	multiRepoEngine := NewMultiRepoEngine(patternEngine, graphBuilder, learningEngine)

	// Add test repositories
	repo1 := &RepositoryContext{
		ID:           "go-repo",
		Name:         "Go Repository",
		Language:     "Go",
		Framework:    "Gin",
		TechStack:    []string{"go", "postgresql", "docker"},
		SuccessRate:  0.9,
		LastActivity: time.Now().Add(-1 * time.Hour),
	}

	repo2 := &RepositoryContext{
		ID:           "python-repo",
		Name:         "Python Repository",
		Language:     "Python",
		Framework:    "Flask",
		TechStack:    []string{"python", "postgresql", "redis"},
		SuccessRate:  0.8,
		LastActivity: time.Now().Add(-2 * time.Hour),
	}

	multiRepoEngine.repositories[repo1.ID] = repo1
	multiRepoEngine.repositories[repo2.ID] = repo2

	// Query for Go repositories
	query := MultiRepoQuery{
		Query:         "debugging",
		TechStacks:    []string{"go"},
		MinConfidence: 0.5,
		MaxResults:    10,
	}

	results, err := multiRepoEngine.QueryMultiRepo(context.Background(), query)
	if err != nil {
		t.Fatalf("Expected no error querying multi-repo, got %v", err)
	}

	// Should only return the Go repository
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if len(results) > 0 && results[0].Repository != "go-repo" {
		t.Errorf("Expected go-repo in results, got %s", results[0].Repository)
	}

	// Query for PostgreSQL repositories
	postgresQuery := MultiRepoQuery{
		Query:         "database",
		TechStacks:    []string{"postgresql"},
		MinConfidence: 0.3,
		MaxResults:    10,
	}

	postgresResults, err := multiRepoEngine.QueryMultiRepo(context.Background(), postgresQuery)
	if err != nil {
		t.Fatalf("Expected no error querying PostgreSQL repos, got %v", err)
	}

	// Should return both repositories since both use PostgreSQL
	if len(postgresResults) != 2 {
		t.Errorf("Expected 2 results for PostgreSQL query, got %d", len(postgresResults))
	}
}

func TestGetSimilarRepositories(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	multiRepoEngine := NewMultiRepoEngine(patternEngine, graphBuilder, learningEngine)

	// Add similar repositories
	repo1 := &RepositoryContext{
		ID:             "go-api-1",
		Language:       "Go",
		Framework:      "Gin",
		TechStack:      []string{"go", "postgresql", "docker", "redis"},
		CommonPatterns: []string{"api_design", "testing"},
	}

	repo2 := &RepositoryContext{
		ID:             "go-api-2",
		Language:       "Go",
		Framework:      "Echo",                                     // Different framework but same language
		TechStack:      []string{"go", "postgresql", "kubernetes"}, // Some overlap
		CommonPatterns: []string{"api_design", "deployment"},
	}

	repo3 := &RepositoryContext{
		ID:             "python-api",
		Language:       "Python", // Different language
		Framework:      "Flask",
		TechStack:      []string{"python", "postgresql"},
		CommonPatterns: []string{"api_design"},
	}

	multiRepoEngine.repositories[repo1.ID] = repo1
	multiRepoEngine.repositories[repo2.ID] = repo2
	multiRepoEngine.repositories[repo3.ID] = repo3

	// Set a lower similarity threshold for testing
	multiRepoEngine.similarityThreshold = 0.3

	similar, err := multiRepoEngine.GetSimilarRepositories(context.Background(), "go-api-1", 5)
	if err != nil {
		t.Fatalf("Expected no error getting similar repositories, got %v", err)
	}

	// Should find at least the other Go repository
	if len(similar) == 0 {
		t.Error("Expected to find similar repositories")
	}

	// Check that go-api-2 is in the results (more similar due to same language and some tech overlap)
	foundGoApi2 := false
	for _, repo := range similar {
		if repo.ID == "go-api-2" {
			foundGoApi2 = true
			break
		}
	}

	if !foundGoApi2 {
		t.Error("Expected to find go-api-2 as similar repository")
	}
}

func TestGetCrossRepoInsights(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	multiRepoEngine := NewMultiRepoEngine(patternEngine, graphBuilder, learningEngine)

	// Add test data
	repo1 := &RepositoryContext{
		ID:            "repo1",
		Language:      "Go",
		Framework:     "Gin",
		TechStack:     []string{"go", "postgresql"},
		SuccessRate:   0.9,
		TotalSessions: 10,
	}

	repo2 := &RepositoryContext{
		ID:            "repo2",
		Language:      "Go",
		Framework:     "Echo",
		TechStack:     []string{"go", "mysql"},
		SuccessRate:   0.8,
		TotalSessions: 5,
	}

	multiRepoEngine.repositories[repo1.ID] = repo1
	multiRepoEngine.repositories[repo2.ID] = repo2

	// Add a cross-repo pattern
	pattern := &CrossRepoPattern{
		ID:           "pattern1",
		Name:         "debugging",
		Repositories: []string{"repo1", "repo2"},
		Frequency:    2,
	}
	multiRepoEngine.crossRepoPatterns[pattern.ID] = pattern

	insights, err := multiRepoEngine.GetCrossRepoInsights(context.Background())
	if err != nil {
		t.Fatalf("Expected no error getting insights, got %v", err)
	}

	// Check basic statistics
	if insights["total_repositories"] != 2 {
		t.Errorf("Expected 2 repositories, got %v", insights["total_repositories"])
	}

	if insights["cross_repo_patterns"] != 1 {
		t.Errorf("Expected 1 cross-repo pattern, got %v", insights["cross_repo_patterns"])
	}

	// Check tech stack distribution
	techDist, ok := insights["tech_stack_distribution"].(map[string]int)
	if !ok {
		t.Error("Expected tech_stack_distribution to be map[string]int")
	} else if techDist["go"] != 2 {
		t.Errorf("Expected Go to appear 2 times, got %d", techDist["go"])
	}

	// Check language distribution
	langDist, ok := insights["language_distribution"].(map[string]int)
	if !ok {
		t.Error("Expected language_distribution to be map[string]int")
	} else if langDist["Go"] != 2 {
		t.Errorf("Expected Go language to appear 2 times, got %d", langDist["Go"])
	}

	// Check average success rate
	avgSuccessRate, ok := insights["avg_success_rate"].(float64)
	if !ok {
		t.Error("Expected avg_success_rate to be float64")
	} else {
		expected := (0.9 + 0.8) / 2.0
		if math.Abs(avgSuccessRate-expected) > 0.001 {
			t.Errorf("Expected average success rate %f, got %f", expected, avgSuccessRate)
		}
	}
}

func TestRepositorySimilarityCalculation(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	multiRepoEngine := NewMultiRepoEngine(patternEngine, graphBuilder, learningEngine)

	repo1 := &RepositoryContext{
		Language:       "Go",
		Framework:      "Gin",
		TechStack:      []string{"go", "postgresql", "docker"},
		CommonPatterns: []string{"api_design", "testing"},
	}

	repo2 := &RepositoryContext{
		Language:       "Go",
		Framework:      "Gin",
		TechStack:      []string{"go", "postgresql", "kubernetes"},
		CommonPatterns: []string{"api_design", "deployment"},
	}

	repo3 := &RepositoryContext{
		Language:       "Python",
		Framework:      "Flask",
		TechStack:      []string{"python", "mysql"},
		CommonPatterns: []string{"debugging"},
	}

	// Test high similarity (same language, framework, some common tech stack)
	similarity12 := multiRepoEngine.calculateRepositorySimilarity(repo1, repo2)
	if similarity12 < 0.5 {
		t.Errorf("Expected high similarity between repo1 and repo2, got %f", similarity12)
	}

	// Test low similarity (different language, framework, tech stack)
	similarity13 := multiRepoEngine.calculateRepositorySimilarity(repo1, repo3)
	if similarity13 > 0.3 {
		t.Errorf("Expected low similarity between repo1 and repo3, got %f", similarity13)
	}

	// Test that similarity is symmetric
	similarity21 := multiRepoEngine.calculateRepositorySimilarity(repo2, repo1)
	if similarity12 != similarity21 {
		t.Errorf("Expected symmetric similarity, got %f and %f", similarity12, similarity21)
	}
}
