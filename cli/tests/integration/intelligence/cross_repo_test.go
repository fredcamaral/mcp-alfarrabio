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

type CrossRepoTestSuite struct {
	suite.Suite
	patternStore      *testutils.MockPatternStorage
	insightStore      *testutils.MockInsightStorage
	crossRepoAnalyzer services.CrossRepoAnalyzer
	testData          *testutils.TestDataGenerator
}

func (s *CrossRepoTestSuite) SetupSuite() {
	s.patternStore = testutils.NewMockPatternStorage()
	s.insightStore = testutils.NewMockInsightStorage()
	s.testData = testutils.NewTestDataGenerator()

	s.crossRepoAnalyzer = services.NewCrossRepoAnalyzer(services.CrossRepoAnalyzerDependencies{
		PatternStore: s.patternStore,
		InsightStore: s.insightStore,
		Logger:       slog.Default(),
	})
}

func (s *CrossRepoTestSuite) TearDownTest() {
	s.patternStore.Clear()
	s.insightStore.Clear()
}

func (s *CrossRepoTestSuite) TestPrivacyProtection() {
	ctx := context.Background()

	// Create sensitive pattern with private data
	sensitivePattern := &entities.TaskPattern{
		ID:         uuid.New().String(),
		Type:       entities.PatternTypeSequence,
		Name:       "Customer Data Processing",
		Repository: "private-finance-app",
		Sequence: []entities.PatternStep{
			{Order: 1, TaskType: "collect", Keywords: []string{"customer", "ssn", "account"}},
			{Order: 2, TaskType: "validate", Keywords: []string{"verification", "compliance"}},
			{Order: 3, TaskType: "store", Keywords: []string{"database", "encryption"}},
		},
		CommonKeywords: []string{"customer", "ssn", "credit-card", "personal-data", "finance"},
		Confidence:     0.9,
		SuccessRate:    0.95,
		Frequency:      25,
		ProjectType:    entities.ProjectTypeWebApp,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Metadata: map[string]interface{}{
			"sensitive":    true,
			"data_types":   []string{"pii", "financial"},
			"compliance":   []string{"gdpr", "pci-dss"},
			"customer_ids": []string{"cust-001", "cust-002"},
		},
	}

	err := s.patternStore.Create(ctx, sensitivePattern)
	s.Require().NoError(err)

	// Configure privacy settings
	privacySettings := &services.PrivacySettings{
		SharePatterns:    true,
		ExcludeKeywords:  []string{"ssn", "credit-card", "customer", "personal-data"},
		ExcludeMetadata:  []string{"customer_ids", "sensitive"},
		MinAnonymization: 3,
		HashSensitiveIDs: true,
	}

	// Contribute pattern with privacy protection
	err = s.crossRepoAnalyzer.ContributePattern(ctx, sensitivePattern, privacySettings)
	s.Require().NoError(err)

	// Get shared insights for the same project type
	sharedInsights, err := s.crossRepoAnalyzer.GetSharedInsights(ctx, entities.ProjectTypeWebApp)
	s.Require().NoError(err)
	s.Assert().NotEmpty(sharedInsights, "Should have shared insights")

	// Find the contributed pattern in shared insights
	var contributedInsight *entities.CrossRepoInsight
	for _, insight := range sharedInsights {
		if insight.Pattern != nil && insight.Pattern.Name == "Customer Data Processing" {
			contributedInsight = insight
			break
		}
	}

	s.Require().NotNil(contributedInsight, "Should find contributed pattern in shared insights")

	// Verify privacy protection applied
	sharedPattern := contributedInsight.Pattern

	// Check that sensitive keywords are filtered out
	for _, keyword := range sharedPattern.CommonKeywords {
		s.Assert().NotContains(keyword, "ssn", "SSN keyword should be filtered")
		s.Assert().NotContains(keyword, "credit-card", "Credit card keyword should be filtered")
		s.Assert().NotContains(keyword, "customer", "Customer keyword should be filtered")
		s.Assert().NotContains(keyword, "personal-data", "Personal data keyword should be filtered")
	}

	// Check that sensitive metadata is removed
	s.Assert().NotContains(sharedPattern.Metadata, "customer_ids", "Customer IDs should be removed")
	s.Assert().NotContains(sharedPattern.Metadata, "sensitive", "Sensitive flag should be removed")

	// Check that pattern steps are sanitized
	for _, step := range sharedPattern.Sequence {
		for _, keyword := range step.Keywords {
			s.Assert().NotContains(keyword, "ssn", "Step keywords should be filtered")
			s.Assert().NotContains(keyword, "customer", "Step keywords should be filtered")
		}
	}

	// Verify that repository name is anonymized
	s.Assert().NotEqual("private-finance-app", sharedPattern.Repository, "Repository name should be anonymized")
	s.Assert().True(strings.HasPrefix(sharedPattern.Repository, "repo-"), "Repository should be anonymized with prefix")

	s.T().Logf("Original pattern had %d keywords, shared pattern has %d keywords",
		len(sensitivePattern.CommonKeywords), len(sharedPattern.CommonKeywords))
}

func (s *CrossRepoTestSuite) TestPatternAggregation() {
	ctx := context.Background()

	// Create multiple similar patterns from different repositories
	webAppPatterns := []*entities.TaskPattern{
		{
			ID:         uuid.New().String(),
			Type:       entities.PatternTypeSequence,
			Name:       "User Registration Flow",
			Repository: "ecommerce-app",
			Sequence: []entities.PatternStep{
				{Order: 1, TaskType: "design", Keywords: []string{"ui", "form", "validation"}},
				{Order: 2, TaskType: "implement", Keywords: []string{"backend", "api", "database"}},
				{Order: 3, TaskType: "test", Keywords: []string{"unit", "integration", "e2e"}},
			},
			CommonKeywords: []string{"user", "registration", "auth", "form"},
			Confidence:     0.85,
			SuccessRate:    0.9,
			Frequency:      10,
			ProjectType:    entities.ProjectTypeWebApp,
			CreatedAt:      time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			ID:         uuid.New().String(),
			Type:       entities.PatternTypeSequence,
			Name:       "User Signup Process",
			Repository: "social-media-app",
			Sequence: []entities.PatternStep{
				{Order: 1, TaskType: "design", Keywords: []string{"wireframe", "form", "ux"}},
				{Order: 2, TaskType: "implement", Keywords: []string{"frontend", "backend", "validation"}},
				{Order: 3, TaskType: "test", Keywords: []string{"automated", "manual", "regression"}},
			},
			CommonKeywords: []string{"user", "signup", "authentication", "onboarding"},
			Confidence:     0.8,
			SuccessRate:    0.88,
			Frequency:      8,
			ProjectType:    entities.ProjectTypeWebApp,
			CreatedAt:      time.Now().Add(-20 * 24 * time.Hour),
		},
		{
			ID:         uuid.New().String(),
			Type:       entities.PatternTypeSequence,
			Name:       "Account Creation Workflow",
			Repository: "banking-app",
			Sequence: []entities.PatternStep{
				{Order: 1, TaskType: "design", Keywords: []string{"security", "compliance", "form"}},
				{Order: 2, TaskType: "implement", Keywords: []string{"secure-backend", "encryption", "storage"}},
				{Order: 3, TaskType: "test", Keywords: []string{"security", "penetration", "compliance"}},
			},
			CommonKeywords: []string{"account", "creation", "security", "verification"},
			Confidence:     0.92,
			SuccessRate:    0.95,
			Frequency:      15,
			ProjectType:    entities.ProjectTypeWebApp,
			CreatedAt:      time.Now().Add(-10 * 24 * time.Hour),
		},
	}

	// Store patterns
	for _, pattern := range webAppPatterns {
		err := s.patternStore.Create(ctx, pattern)
		s.Require().NoError(err)

		// Contribute each pattern
		err = s.crossRepoAnalyzer.ContributePattern(ctx, pattern, &services.PrivacySettings{
			SharePatterns:   true,
			ExcludeKeywords: []string{},
		})
		s.Require().NoError(err)
	}

	// Analyze cross-repository patterns
	aggregatedInsights, err := s.crossRepoAnalyzer.AnalyzeCrossRepoPatterns(ctx, entities.ProjectTypeWebApp)
	s.Require().NoError(err)
	s.Assert().NotEmpty(aggregatedInsights, "Should generate aggregated insights")

	// Look for aggregated user registration pattern
	var userRegInsight *entities.CrossRepoInsight
	for _, insight := range aggregatedInsights {
		if insight.Type == entities.InsightTypeAggregatedPattern {
			name := strings.ToLower(insight.Title)
			if strings.Contains(name, "user") && (strings.Contains(name, "registration") ||
				strings.Contains(name, "signup") || strings.Contains(name, "creation")) {
				userRegInsight = insight
				break
			}
		}
	}

	s.Assert().NotNil(userRegInsight, "Should find aggregated user registration pattern")

	// Verify aggregation quality
	s.Assert().Greater(userRegInsight.Confidence, 0.7, "Aggregated pattern should have good confidence")
	s.Assert().GreaterOrEqual(userRegInsight.Frequency, 3, "Should aggregate from multiple sources")
	s.Assert().Contains(userRegInsight.Description, "common", "Should indicate pattern commonality")

	// Check that evidence includes multiple repositories
	s.Assert().GreaterOrEqual(len(userRegInsight.Evidence), 2, "Should have evidence from multiple repos")

	s.T().Logf("Aggregated insight: %s (confidence: %.2f, frequency: %d)",
		userRegInsight.Title, userRegInsight.Confidence, userRegInsight.Frequency)
}

func (s *CrossRepoTestSuite) TestCrossRepoRecommendations() {
	ctx := context.Background()

	// Create patterns from successful projects
	successfulPatterns := []*entities.TaskPattern{
		{
			ID:         uuid.New().String(),
			Type:       entities.PatternTypeWorkflow,
			Name:       "CI/CD Best Practices",
			Repository: "high-performing-team-1",
			Phases: []entities.WorkflowPhase{
				{
					Name:           "testing",
					Tasks:          []string{"unit-tests", "integration-tests", "e2e-tests"},
					MaxParallelism: 3,
				},
				{
					Name:           "deployment",
					Tasks:          []string{"staging-deploy", "smoke-tests", "prod-deploy"},
					MaxParallelism: 1,
				},
			},
			CommonKeywords: []string{"ci", "cd", "testing", "deployment", "automation"},
			Confidence:     0.95,
			SuccessRate:    0.98,
			Frequency:      50,
			ProjectType:    entities.ProjectTypeAPI,
			CreatedAt:      time.Now().Add(-60 * 24 * time.Hour),
			Metadata: map[string]interface{}{
				"performance_score": 9.2,
				"defect_rate":       0.02,
				"deploy_frequency":  "daily",
			},
		},
		{
			ID:         uuid.New().String(),
			Type:       entities.PatternTypeSequence,
			Name:       "Code Review Process",
			Repository: "high-performing-team-2",
			Sequence: []entities.PatternStep{
				{Order: 1, TaskType: "implement", Keywords: []string{"feature", "small-commits"}},
				{Order: 2, TaskType: "review", Keywords: []string{"peer-review", "automated-checks"}},
				{Order: 3, TaskType: "merge", Keywords: []string{"squash", "clean-history"}},
			},
			CommonKeywords: []string{"code-review", "collaboration", "quality", "peer-review"},
			Confidence:     0.9,
			SuccessRate:    0.95,
			Frequency:      40,
			ProjectType:    entities.ProjectTypeAPI,
			CreatedAt:      time.Now().Add(-45 * 24 * time.Hour),
			Metadata: map[string]interface{}{
				"review_time_avg": "2h",
				"approval_rate":   0.85,
				"defect_rate":     0.03,
			},
		},
	}

	for _, pattern := range successfulPatterns {
		err := s.patternStore.Create(ctx, pattern)
		s.Require().NoError(err)

		err = s.crossRepoAnalyzer.ContributePattern(ctx, pattern, &services.PrivacySettings{
			SharePatterns: true,
		})
		s.Require().NoError(err)
	}

	// Generate recommendations for a new API project
	recommendations, err := s.crossRepoAnalyzer.GenerateRecommendations(ctx, entities.ProjectTypeAPI, map[string]interface{}{
		"team_size":         5,
		"project_stage":     "new",
		"quality_focus":     true,
		"deployment_target": "production",
	})
	s.Require().NoError(err)
	s.Assert().NotEmpty(recommendations, "Should generate cross-repo recommendations")

	// Look for CI/CD recommendation
	var cicdRec *entities.CrossRepoRecommendation
	for _, rec := range recommendations {
		if strings.Contains(strings.ToLower(rec.Title), "ci") ||
			strings.Contains(strings.ToLower(rec.Title), "deploy") {
			cicdRec = rec
			break
		}
	}

	s.Assert().NotNil(cicdRec, "Should recommend CI/CD practices")
	s.Assert().Greater(cicdRec.Impact, 0.7, "CI/CD recommendation should have high impact")
	s.Assert().Contains(strings.ToLower(cicdRec.Description), "testing",
		"Should mention testing in CI/CD recommendation")

	// Look for code review recommendation
	var reviewRec *entities.CrossRepoRecommendation
	for _, rec := range recommendations {
		if strings.Contains(strings.ToLower(rec.Title), "review") ||
			strings.Contains(strings.ToLower(rec.Title), "quality") {
			reviewRec = rec
			break
		}
	}

	s.Assert().NotNil(reviewRec, "Should recommend code review practices")
	s.Assert().Greater(reviewRec.Impact, 0.6, "Code review recommendation should have good impact")

	s.T().Logf("Generated %d cross-repo recommendations", len(recommendations))
	for i, rec := range recommendations {
		s.T().Logf("  %d. %s (impact: %.2f, confidence: %.2f)",
			i+1, rec.Title, rec.Impact, rec.Confidence)
	}
}

func (s *CrossRepoTestSuite) TestInsightEvolution() {
	ctx := context.Background()

	// Create initial pattern
	initialPattern := &entities.TaskPattern{
		ID:         uuid.New().String(),
		Type:       entities.PatternTypeSequence,
		Name:       "Basic API Development",
		Repository: "learning-project",
		Sequence: []entities.PatternStep{
			{Order: 1, TaskType: "implement", Keywords: []string{"endpoint", "basic"}},
			{Order: 2, TaskType: "test", Keywords: []string{"manual", "postman"}},
		},
		CommonKeywords: []string{"api", "basic", "manual"},
		Confidence:     0.6,
		SuccessRate:    0.7,
		Frequency:      5,
		ProjectType:    entities.ProjectTypeAPI,
		CreatedAt:      time.Now().Add(-90 * 24 * time.Hour),
	}

	err := s.patternStore.Create(ctx, initialPattern)
	s.Require().NoError(err)
	err = s.crossRepoAnalyzer.ContributePattern(ctx, initialPattern, &services.PrivacySettings{SharePatterns: true})
	s.Require().NoError(err)

	// Evolve to intermediate pattern
	evolvedPattern := &entities.TaskPattern{
		ID:         uuid.New().String(),
		Type:       entities.PatternTypeSequence,
		Name:       "Improved API Development",
		Repository: "maturing-project",
		Sequence: []entities.PatternStep{
			{Order: 1, TaskType: "design", Keywords: []string{"openapi", "spec"}},
			{Order: 2, TaskType: "implement", Keywords: []string{"endpoint", "validation"}},
			{Order: 3, TaskType: "test", Keywords: []string{"automated", "integration"}},
		},
		CommonKeywords: []string{"api", "design", "automated", "validation"},
		Confidence:     0.8,
		SuccessRate:    0.85,
		Frequency:      12,
		ProjectType:    entities.ProjectTypeAPI,
		CreatedAt:      time.Now().Add(-30 * 24 * time.Hour),
	}

	err = s.patternStore.Create(ctx, evolvedPattern)
	s.Require().NoError(err)
	err = s.crossRepoAnalyzer.ContributePattern(ctx, evolvedPattern, &services.PrivacySettings{SharePatterns: true})
	s.Require().NoError(err)

	// Add mature pattern
	maturePattern := &entities.TaskPattern{
		ID:         uuid.New().String(),
		Type:       entities.PatternTypeWorkflow,
		Name:       "Enterprise API Development",
		Repository: "enterprise-project",
		Phases: []entities.WorkflowPhase{
			{
				Name:           "design",
				Tasks:          []string{"architecture", "openapi", "review"},
				MaxParallelism: 2,
			},
			{
				Name:           "development",
				Tasks:          []string{"implement", "security", "performance"},
				MaxParallelism: 3,
			},
			{
				Name:           "quality",
				Tasks:          []string{"unit-test", "integration-test", "security-scan"},
				MaxParallelism: 3,
			},
		},
		CommonKeywords: []string{"api", "enterprise", "security", "performance", "architecture"},
		Confidence:     0.95,
		SuccessRate:    0.95,
		Frequency:      25,
		ProjectType:    entities.ProjectTypeAPI,
		CreatedAt:      time.Now().Add(-5 * 24 * time.Hour),
	}

	err = s.patternStore.Create(ctx, maturePattern)
	s.Require().NoError(err)
	err = s.crossRepoAnalyzer.ContributePattern(ctx, maturePattern, &services.PrivacySettings{SharePatterns: true})
	s.Require().NoError(err)

	// Analyze evolution
	evolution, err := s.crossRepoAnalyzer.AnalyzePatternEvolution(ctx, entities.ProjectTypeAPI,
		map[string]interface{}{
			"keyword_focus": "api",
			"time_window":   "90d",
		})
	s.Require().NoError(err)
	s.Assert().NotEmpty(evolution, "Should detect pattern evolution")

	// Find API development evolution
	var apiEvolution *entities.PatternEvolution
	for _, evo := range evolution {
		if strings.Contains(strings.ToLower(evo.PatternName), "api") {
			apiEvolution = evo
			break
		}
	}

	s.Assert().NotNil(apiEvolution, "Should find API development evolution")
	s.Assert().Len(apiEvolution.Stages, 3, "Should detect 3 evolution stages")

	// Verify evolution stages are ordered chronologically
	for i := 0; i < len(apiEvolution.Stages)-1; i++ {
		current := apiEvolution.Stages[i]
		next := apiEvolution.Stages[i+1]
		s.Assert().True(current.Timestamp.Before(next.Timestamp),
			"Evolution stages should be chronologically ordered")
		s.Assert().LessOrEqual(current.Complexity, next.Complexity,
			"Complexity should generally increase over time")
	}

	// Verify evolution insights
	s.Assert().NotEmpty(apiEvolution.KeyChanges, "Should identify key changes")
	s.Assert().Greater(apiEvolution.MaturityScore, 0.5, "Should calculate maturity score")

	s.T().Logf("API evolution: %s (maturity: %.2f)", apiEvolution.PatternName, apiEvolution.MaturityScore)
	for i, stage := range apiEvolution.Stages {
		s.T().Logf("  Stage %d: complexity %.1f, success %.2f (%s)",
			i+1, stage.Complexity, stage.SuccessRate, stage.Timestamp.Format("2006-01-02"))
	}
}

func (s *CrossRepoTestSuite) TestKnowledgeGraph() {
	ctx := context.Background()

	// Create interconnected patterns
	patterns := []*entities.TaskPattern{
		{
			ID:             uuid.New().String(),
			Name:           "Frontend Component Development",
			ProjectType:    entities.ProjectTypeWebApp,
			CommonKeywords: []string{"react", "component", "testing", "storybook"},
			Confidence:     0.85,
			Repository:     "ui-library",
		},
		{
			ID:             uuid.New().String(),
			Name:           "API Integration",
			ProjectType:    entities.ProjectTypeWebApp,
			CommonKeywords: []string{"api", "rest", "axios", "testing"},
			Confidence:     0.8,
			Repository:     "web-app",
		},
		{
			ID:             uuid.New().String(),
			Name:           "Backend Testing Strategy",
			ProjectType:    entities.ProjectTypeAPI,
			CommonKeywords: []string{"testing", "unit", "integration", "mocking"},
			Confidence:     0.9,
			Repository:     "api-service",
		},
		{
			ID:             uuid.New().String(),
			Name:           "Database Migration Process",
			ProjectType:    entities.ProjectTypeAPI,
			CommonKeywords: []string{"database", "migration", "testing", "rollback"},
			Confidence:     0.88,
			Repository:     "data-service",
		},
	}

	for _, pattern := range patterns {
		pattern.CreatedAt = time.Now()
		pattern.UpdatedAt = time.Now()
		err := s.patternStore.Create(ctx, pattern)
		s.Require().NoError(err)
		err = s.crossRepoAnalyzer.ContributePattern(ctx, pattern, &services.PrivacySettings{SharePatterns: true})
		s.Require().NoError(err)
	}

	// Build knowledge graph
	knowledgeGraph, err := s.crossRepoAnalyzer.BuildKnowledgeGraph(ctx)
	s.Require().NoError(err)
	s.Assert().NotEmpty(knowledgeGraph.Nodes, "Knowledge graph should have nodes")
	s.Assert().NotEmpty(knowledgeGraph.Edges, "Knowledge graph should have edges")

	// Verify testing connections (common keyword)
	testingConnections := 0
	for _, edge := range knowledgeGraph.Edges {
		if edge.Relationship == "shares_keyword" && edge.Metadata["keyword"] == "testing" {
			testingConnections++
		}
	}
	s.Assert().Greater(testingConnections, 0, "Should find testing keyword connections")

	// Query related patterns
	relatedPatterns, err := s.crossRepoAnalyzer.FindRelatedPatterns(ctx, "testing", 2)
	s.Require().NoError(err)
	s.Assert().GreaterOrEqual(len(relatedPatterns), 3, "Should find multiple testing-related patterns")

	// Verify relevance scores
	for _, related := range relatedPatterns {
		s.Assert().Greater(related.RelevanceScore, 0.0, "Related patterns should have relevance scores")
		s.Assert().Contains(related.CommonKeywords, "testing", "Related patterns should contain testing keyword")
	}

	s.T().Logf("Knowledge graph: %d nodes, %d edges", len(knowledgeGraph.Nodes), len(knowledgeGraph.Edges))
	s.T().Logf("Found %d testing connections", testingConnections)
	s.T().Logf("Found %d related patterns for 'testing'", len(relatedPatterns))
}

func (s *CrossRepoTestSuite) TestCrossRepoPerformance() {
	ctx := context.Background()

	// Generate large dataset of patterns
	largePatternSet := s.testData.GenerateRandomPatterns("cross-repo-perf", 500)

	for _, pattern := range largePatternSet {
		err := s.patternStore.Create(ctx, pattern)
		s.Require().NoError(err)
	}

	// Measure contribution performance
	start := time.Now()
	for _, pattern := range largePatternSet[:100] { // Contribute first 100
		err := s.crossRepoAnalyzer.ContributePattern(ctx, pattern, &services.PrivacySettings{
			SharePatterns:   true,
			ExcludeKeywords: []string{"sensitive", "private"},
		})
		s.Require().NoError(err)
	}
	contributionDuration := time.Since(start)

	s.Assert().Less(contributionDuration, 30*time.Second,
		"Contributing 100 patterns should complete within 30 seconds")

	// Measure insight generation performance
	start = time.Now()
	insights, err := s.crossRepoAnalyzer.GetSharedInsights(ctx, entities.ProjectTypeWebApp)
	insightDuration := time.Since(start)

	s.Require().NoError(err)
	s.Assert().Less(insightDuration, 5*time.Second,
		"Generating insights should complete within 5 seconds")

	// Measure knowledge graph building performance
	start = time.Now()
	knowledgeGraph, err := s.crossRepoAnalyzer.BuildKnowledgeGraph(ctx)
	graphDuration := time.Since(start)

	s.Require().NoError(err)
	s.Assert().Less(graphDuration, 10*time.Second,
		"Building knowledge graph should complete within 10 seconds")

	s.T().Logf("Performance metrics:")
	s.T().Logf("  Pattern contribution: %v for 100 patterns", contributionDuration)
	s.T().Logf("  Insight generation: %v for %d insights", insightDuration, len(insights))
	s.T().Logf("  Knowledge graph: %v for %d nodes", graphDuration, len(knowledgeGraph.Nodes))
}

func TestCrossRepo(t *testing.T) {
	suite.Run(t, new(CrossRepoTestSuite))
}
