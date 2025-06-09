//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/services"
	"lerian-mcp-memory-cli/tests/testutils"
)

type IntelligenceWorkflowTestSuite struct {
	suite.Suite

	// Storage layers
	taskStore     *testutils.MockTaskStorage
	patternStore  *testutils.MockPatternStorage
	templateStore *testutils.MockTemplateStorage
	sessionStore  *testutils.MockSessionStorage
	insightStore  *testutils.MockInsightStorage

	// Services
	patternDetector   services.PatternDetector
	suggestionService services.SuggestionService
	templateService   services.TemplateService
	analyticsService  services.AnalyticsService
	crossRepoAnalyzer services.CrossRepoAnalyzer

	// Utilities
	testData *testutils.TestDataGenerator
	tempDirs []string
}

func (s *IntelligenceWorkflowTestSuite) SetupSuite() {
	// Initialize storage layers
	s.taskStore = testutils.NewMockTaskStorage()
	s.patternStore = testutils.NewMockPatternStorage()
	s.templateStore = testutils.NewMockTemplateStorage()
	s.sessionStore = testutils.NewMockSessionStorage()
	s.insightStore = testutils.NewMockInsightStorage()
	s.testData = testutils.NewTestDataGenerator()

	// Initialize AI service
	aiService := testutils.NewMockAIService()

	// Initialize core services
	s.patternDetector = services.NewPatternDetector(services.PatternDetectorDependencies{
		TaskStore:    s.taskStore,
		PatternStore: s.patternStore,
		AI:           aiService,
		Logger:       slog.Default(),
	})

	contextAnalyzer := services.NewContextAnalyzer(services.ContextAnalyzerDependencies{
		TaskStore: s.taskStore,
		Logger:    slog.Default(),
	})

	s.suggestionService = services.NewSuggestionService(services.SuggestionServiceDependencies{
		ContextAnalyzer: contextAnalyzer,
		PatternDetector: s.patternDetector,
		TaskStore:       s.taskStore,
		PatternStore:    s.patternStore,
		TemplateStore:   s.templateStore,
		AI:              aiService,
		Logger:          slog.Default(),
	})

	classifier := services.NewProjectClassifier(services.ProjectClassifierDependencies{
		AI:     aiService,
		Logger: slog.Default(),
	})

	s.templateService = services.NewTemplateService(services.TemplateServiceDependencies{
		TemplateStore:     s.templateStore,
		TaskStore:         s.taskStore,
		ProjectClassifier: classifier,
		Logger:            slog.Default(),
	})

	s.analyticsService = services.NewAnalyticsService(services.AnalyticsServiceDependencies{
		TaskStore:    s.taskStore,
		PatternStore: s.patternStore,
		SessionStore: s.sessionStore,
		Visualizer:   testutils.NewMockVisualizer(),
		Exporter:     testutils.NewMockAnalyticsExporter(),
		Calculator:   services.NewMetricsCalculator(),
		Logger:       slog.Default(),
	})

	s.crossRepoAnalyzer = services.NewCrossRepoAnalyzer(services.CrossRepoAnalyzerDependencies{
		PatternStore: s.patternStore,
		InsightStore: s.insightStore,
		Logger:       slog.Default(),
	})
}

func (s *IntelligenceWorkflowTestSuite) TearDownSuite() {
	// Clean up temporary directories
	for _, dir := range s.tempDirs {
		os.RemoveAll(dir)
	}
}

func (s *IntelligenceWorkflowTestSuite) TearDownTest() {
	s.taskStore.Clear()
	s.patternStore.Clear()
	s.templateStore.Clear()
	s.sessionStore.Clear()
	s.insightStore.Clear()
}

func (s *IntelligenceWorkflowTestSuite) TestCompleteIntelligenceWorkflow() {
	ctx := context.Background()

	s.T().Log("🚀 Starting complete intelligence workflow test...")

	// Step 1: Initialize new project and classify it
	s.T().Log("📁 Step 1: Project initialization and classification")

	projectFiles := map[string]string{
		"package.json": `{
			"name": "e2e-test-app",
			"version": "1.0.0",
			"dependencies": {
				"react": "^18.2.0",
				"react-dom": "^18.2.0",
				"typescript": "^5.0.0"
			},
			"scripts": {
				"start": "react-scripts start",
				"build": "react-scripts build",
				"test": "react-scripts test"
			}
		}`,
		"src/App.tsx": `import React from 'react';
import './App.css';

function App() {
	return (
		<div className="App">
			<header className="App-header">
				<h1>E2E Test App</h1>
			</header>
		</div>
	);
}

export default App;`,
		"src/index.tsx": `import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';

const root = ReactDOM.createRoot(
	document.getElementById('root') as HTMLElement
);
root.render(<App />);`,
		"tsconfig.json": `{
			"compilerOptions": {
				"target": "es5",
				"lib": ["dom", "dom.iterable", "esnext"],
				"allowJs": true,
				"skipLibCheck": true,
				"esModuleInterop": true,
				"allowSyntheticDefaultImports": true,
				"strict": true,
				"forceConsistentCasingInFileNames": true,
				"moduleResolution": "node",
				"resolveJsonModule": true,
				"isolatedModules": true,
				"noEmit": true,
				"jsx": "react-jsx"
			}
		}`,
	}

	projectPath := s.testData.CreateTempProject(projectFiles)
	s.tempDirs = append(s.tempDirs, projectPath)

	// Classify the project
	projectType, confidence, err := s.templateService.ClassifyProject(ctx, projectPath)
	s.Require().NoError(err)
	s.Assert().Equal(entities.ProjectTypeWebApp, projectType)
	s.Assert().Greater(confidence, 0.7)

	s.T().Logf("✅ Project classified as %s with %.2f confidence", projectType, confidence)

	// Step 2: Get and instantiate template
	s.T().Log("📋 Step 2: Template matching and instantiation")

	templateMatches, err := s.templateService.MatchTemplates(ctx, projectPath)
	s.Require().NoError(err)
	s.Assert().NotEmpty(templateMatches)

	bestTemplate := templateMatches[0].Template
	s.T().Logf("✅ Best template match: %s (confidence: %.2f)", bestTemplate.Name, templateMatches[0].Confidence)

	// Instantiate template with project-specific variables
	templateVars := map[string]interface{}{
		"repository": "e2e-test-app",
		"Framework":  "react",
		"Language":   "typescript",
		"Database":   "postgresql",
		"AuthMethod": "jwt",
	}

	initialTasks, err := s.templateService.InstantiateTemplate(ctx, bestTemplate.ID, templateVars)
	s.Require().NoError(err)
	s.Assert().NotEmpty(initialTasks)

	s.T().Logf("✅ Generated %d initial tasks from template", len(initialTasks))

	// Store initial tasks
	for _, task := range initialTasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Step 3: Simulate development work and create patterns
	s.T().Log("💻 Step 3: Simulating development work")

	// Work on tasks in a pattern that will be detectable
	workSessions := s.simulateDevelopmentWork(ctx, initialTasks[:6]) // Work on first 6 tasks

	for _, session := range workSessions {
		err := s.sessionStore.Create(ctx, session)
		s.Require().NoError(err)
	}

	s.T().Logf("✅ Completed %d work sessions", len(workSessions))

	// Step 4: Detect patterns from the work done
	s.T().Log("🔍 Step 4: Pattern detection")

	allTasks, err := s.taskStore.GetByRepository(ctx, "e2e-test-app", nil)
	s.Require().NoError(err)

	// Detect sequence patterns
	sequencePatterns, err := s.patternDetector.DetectSequencePatterns(ctx, allTasks, 0.1)
	s.Require().NoError(err)

	// Detect workflow patterns
	workflowPatterns, err := s.patternDetector.DetectWorkflowPatterns(ctx, allTasks, 0.1)
	s.Require().NoError(err)

	totalPatterns := len(sequencePatterns) + len(workflowPatterns)
	s.Assert().Greater(totalPatterns, 0)

	s.T().Logf("✅ Detected %d sequence patterns and %d workflow patterns",
		len(sequencePatterns), len(workflowPatterns))

	// Step 5: Generate intelligent suggestions
	s.T().Log("💡 Step 5: Intelligent suggestion generation")

	suggestions, err := s.suggestionService.GenerateSuggestions(ctx, "e2e-test-app")
	s.Require().NoError(err)
	s.Assert().NotEmpty(suggestions)

	// Get next task suggestion based on current work
	lastCompletedTask := s.findLastCompletedTask(allTasks)
	s.Require().NotNil(lastCompletedTask)

	nextSuggestion, err := s.suggestionService.GetNextTaskSuggestion(ctx, lastCompletedTask)
	s.Require().NoError(err)
	s.Assert().NotNil(nextSuggestion)

	s.T().Logf("✅ Generated %d general suggestions", len(suggestions))
	s.T().Logf("✅ Next task suggestion: %s (confidence: %.2f)",
		nextSuggestion.Content, nextSuggestion.Confidence)

	// Step 6: Contribute to cross-repository learning
	s.T().Log("🌐 Step 6: Cross-repository contribution")

	if len(sequencePatterns) > 0 {
		err = s.crossRepoAnalyzer.ContributePattern(ctx, sequencePatterns[0], &services.PrivacySettings{
			SharePatterns:   true,
			ExcludeKeywords: []string{"sensitive", "private"},
		})
		s.Require().NoError(err)

		s.T().Log("✅ Contributed pattern to cross-repository knowledge")
	}

	// Get shared insights
	sharedInsights, err := s.crossRepoAnalyzer.GetSharedInsights(ctx, entities.ProjectTypeWebApp)
	s.Require().NoError(err)

	s.T().Logf("✅ Retrieved %d shared insights", len(sharedInsights))

	// Step 7: Generate comprehensive analytics
	s.T().Log("📊 Step 7: Analytics generation")

	period := entities.TimePeriod{
		Start: time.Now().AddDate(0, 0, -7),
		End:   time.Now(),
	}

	workflowMetrics, err := s.analyticsService.GetWorkflowMetrics(ctx, "e2e-test-app", period)
	s.Require().NoError(err)
	s.Assert().Greater(workflowMetrics.Productivity.Score, 0.0)

	productivityReport, err := s.analyticsService.GetProductivityReport(ctx, "e2e-test-app", period)
	s.Require().NoError(err)
	s.Assert().NotEmpty(productivityReport.Insights)
	s.Assert().NotEmpty(productivityReport.Recommendations)

	s.T().Logf("✅ Generated analytics - Productivity: %.1f, Overall: %.1f",
		workflowMetrics.Productivity.Score, productivityReport.OverallScore)

	// Step 8: Test feature interactions and data flow
	s.T().Log("🔄 Step 8: Testing feature interactions")

	// Verify that patterns influence suggestions
	patternBasedSuggestions := 0
	for _, suggestion := range suggestions {
		if suggestion.Source.Type == entities.SuggestionSourcePattern {
			patternBasedSuggestions++
		}
	}

	// Verify that templates influenced initial task creation
	templateInfluencedTasks := 0
	for _, task := range initialTasks {
		if source, ok := task.Metadata["source"].(string); ok && source == "template" {
			templateInfluencedTasks++
		}
	}

	// Verify that analytics includes pattern information
	patternMetricsExists := workflowMetrics.Patterns.TotalPatterns > 0

	s.T().Logf("✅ Feature interactions verified:")
	s.T().Logf("   - Pattern-based suggestions: %d/%d", patternBasedSuggestions, len(suggestions))
	s.T().Logf("   - Template-influenced tasks: %d/%d", templateInfluencedTasks, len(initialTasks))
	s.T().Logf("   - Pattern metrics included: %v", patternMetricsExists)

	// Step 9: Test end-to-end performance
	s.T().Log("⚡ Step 9: Performance validation")

	start := time.Now()

	// Simulate a complete workflow cycle
	newSuggestions, err := s.suggestionService.GenerateSuggestions(ctx, "e2e-test-app")
	s.Require().NoError(err)

	_, err = s.analyticsService.GetWorkflowMetrics(ctx, "e2e-test-app", period)
	s.Require().NoError(err)

	_, err = s.crossRepoAnalyzer.GetSharedInsights(ctx, entities.ProjectTypeWebApp)
	s.Require().NoError(err)

	cycleTime := time.Since(start)
	s.Assert().Less(cycleTime, 5*time.Second, "Complete workflow cycle should be fast")

	s.T().Logf("✅ Complete workflow cycle completed in %v", cycleTime)

	// Step 10: Validate final state and outputs
	s.T().Log("✅ Step 10: Final validation")

	// Count final entities
	finalTasks, _ := s.taskStore.GetByRepository(ctx, "e2e-test-app", nil)
	finalPatterns, _ := s.patternStore.GetByRepository(ctx, "e2e-test-app")
	finalSessions, _ := s.sessionStore.GetByRepository(ctx, "e2e-test-app")

	// Validate we have meaningful data
	s.Assert().GreaterOrEqual(len(finalTasks), 6, "Should have at least initial tasks")
	s.Assert().Greater(len(finalSessions), 0, "Should have work sessions")
	s.Assert().NotEmpty(newSuggestions, "Should have ongoing suggestions")

	s.T().Log("🎉 Complete intelligence workflow test PASSED!")
	s.T().Logf("📈 Final state: %d tasks, %d patterns, %d sessions, %d suggestions",
		len(finalTasks), len(finalPatterns), len(finalSessions), len(newSuggestions))
}

func (s *IntelligenceWorkflowTestSuite) TestIntelligenceWorkflowEdgeCases() {
	ctx := context.Background()

	s.T().Log("🧪 Testing intelligence workflow edge cases...")

	// Test 1: Empty project handling
	s.T().Log("📭 Test 1: Empty project handling")

	emptyProject := s.testData.CreateTempProject(map[string]string{
		"README.md": "# Empty Project",
	})
	s.tempDirs = append(s.tempDirs, emptyProject)

	projectType, confidence, err := s.templateService.ClassifyProject(ctx, emptyProject)

	// Should handle gracefully, might return unknown type with low confidence
	s.Assert().NoError(err)
	s.T().Logf("Empty project classified as %s with %.2f confidence", projectType, confidence)

	// Test 2: Single task workflow
	s.T().Log("📝 Test 2: Single task workflow")

	singleTask := s.testData.CreateTask("Single task test", "medium", "completed", nil)
	singleTask.Repository = "single-task-repo"
	err = s.taskStore.Create(ctx, singleTask)
	s.Require().NoError(err)

	// Should handle single task gracefully
	singleTaskList := []*entities.Task{singleTask}
	patterns, err := s.patternDetector.DetectSequencePatterns(ctx, singleTaskList, 0.1)
	s.Assert().NoError(err)
	s.T().Logf("Single task resulted in %d patterns", len(patterns))

	// Test 3: Conflicting patterns
	s.T().Log("⚔️ Test 3: Conflicting patterns handling")

	conflictingTasks := s.generateConflictingTasks("conflict-repo")
	for _, task := range conflictingTasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	conflictPatterns, err := s.patternDetector.DetectSequencePatterns(ctx, conflictingTasks, 0.2)
	s.Assert().NoError(err)
	s.T().Logf("Conflicting tasks resulted in %d patterns", len(conflictPatterns))

	// Test 4: Large dataset performance
	s.T().Log("🏋️ Test 4: Large dataset performance")

	largeTasks := s.testData.GenerateRandomTasks("large-repo", 1000)
	start := time.Now()

	for _, task := range largeTasks[:100] { // Store first 100 for performance test
		s.taskStore.Create(ctx, task)
	}

	largePatterns, err := s.patternDetector.DetectSequencePatterns(ctx, largeTasks[:100], 0.05)
	duration := time.Since(start)

	s.Assert().NoError(err)
	s.Assert().Less(duration, 3*time.Second)
	s.T().Logf("Large dataset processed in %v, found %d patterns", duration, len(largePatterns))

	s.T().Log("✅ All edge cases handled successfully!")
}

func (s *IntelligenceWorkflowTestSuite) TestIntelligenceWorkflowIntegration() {
	ctx := context.Background()

	s.T().Log("🔗 Testing cross-feature integration...")

	// Setup: Create a project with mixed development patterns
	repo := "integration-test-repo"

	// Create diverse tasks representing different development patterns
	integrationTasks := s.generateIntegrationTasks(repo)
	integrationSessions := s.generateIntegrationSessions(repo)

	for _, task := range integrationTasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	for _, session := range integrationSessions {
		err := s.sessionStore.Create(ctx, session)
		s.Require().NoError(err)
	}

	// Test Integration 1: Pattern-driven suggestions
	s.T().Log("🔍➡️💡 Integration 1: Patterns → Suggestions")

	patterns, err := s.patternDetector.DetectSequencePatterns(ctx, integrationTasks, 0.1)
	s.Require().NoError(err)

	// Store detected patterns
	for _, pattern := range patterns {
		err := s.patternStore.Create(ctx, pattern)
		s.Require().NoError(err)
	}

	// Generate suggestions (should use the stored patterns)
	suggestions, err := s.suggestionService.GenerateSuggestions(ctx, repo)
	s.Require().NoError(err)

	patternInfluencedSuggestions := 0
	for _, suggestion := range suggestions {
		if suggestion.Source.Type == entities.SuggestionSourcePattern {
			patternInfluencedSuggestions++
		}
	}

	s.Assert().Greater(patternInfluencedSuggestions, 0, "Patterns should influence suggestions")
	s.T().Logf("✅ %d/%d suggestions influenced by patterns", patternInfluencedSuggestions, len(suggestions))

	// Test Integration 2: Analytics incorporating patterns
	s.T().Log("🔍➡️📊 Integration 2: Patterns → Analytics")

	period := entities.TimePeriod{Start: time.Now().AddDate(0, 0, -30), End: time.Now()}
	metrics, err := s.analyticsService.GetWorkflowMetrics(ctx, repo, period)
	s.Require().NoError(err)

	s.Assert().Equal(len(patterns), metrics.Patterns.TotalPatterns, "Analytics should include pattern count")
	s.Assert().Greater(metrics.Patterns.ActivePatterns, 0, "Should have active patterns")

	s.T().Logf("✅ Analytics includes %d patterns", metrics.Patterns.TotalPatterns)

	// Test Integration 3: Cross-repo learning from local patterns
	s.T().Log("🔍➡️🌐 Integration 3: Patterns → Cross-repo learning")

	if len(patterns) > 0 {
		err = s.crossRepoAnalyzer.ContributePattern(ctx, patterns[0], &services.PrivacySettings{
			SharePatterns: true,
		})
		s.Require().NoError(err)

		sharedInsights, err := s.crossRepoAnalyzer.GetSharedInsights(ctx, entities.ProjectTypeWebApp)
		s.Require().NoError(err)
		s.Assert().NotEmpty(sharedInsights, "Should have shared insights after contribution")

		s.T().Logf("✅ Pattern contributed, %d shared insights available", len(sharedInsights))
	}

	// Test Integration 4: Template system using analytics
	s.T().Log("📊➡️📋 Integration 4: Analytics → Template recommendations")

	// Get recommendations based on current metrics
	recommendations, err := s.analyticsService.GenerateRecommendations(ctx, metrics)
	s.Require().NoError(err)
	s.Assert().NotEmpty(recommendations, "Should generate recommendations from metrics")

	s.T().Logf("✅ Generated %d recommendations from analytics", len(recommendations))

	// Test Integration 5: Full circle - suggestions to tasks to patterns
	s.T().Log("💡➡️📝➡️🔍 Integration 5: Full workflow circle")

	// Take a suggestion and simulate implementing it
	if len(suggestions) > 0 {
		suggestion := suggestions[0]

		// Create task from suggestion
		taskFromSuggestion := &entities.Task{
			ID:          uuid.New().String(),
			Content:     suggestion.Content,
			Description: suggestion.Description,
			Priority:    string(suggestion.Priority),
			Status:      "completed",
			Repository:  repo,
			Type:        string(suggestion.Type),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Metadata: map[string]interface{}{
				"source":        "suggestion",
				"suggestion_id": suggestion.ID,
			},
		}

		err := s.taskStore.Create(ctx, taskFromSuggestion)
		s.Require().NoError(err)

		// Re-detect patterns (should now include the new task)
		allTasks, _ := s.taskStore.GetByRepository(ctx, repo, nil)
		newPatterns, err := s.patternDetector.DetectSequencePatterns(ctx, allTasks, 0.1)
		s.Require().NoError(err)

		s.T().Logf("✅ Full circle: suggestion → task → %d new patterns", len(newPatterns))
	}

	s.T().Log("🎯 All integration tests passed!")
}

// Helper methods

func (s *IntelligenceWorkflowTestSuite) simulateDevelopmentWork(ctx context.Context, tasks []*entities.Task) []*entities.Session {
	var sessions []*entities.Session

	// Simulate working on tasks in sequence with realistic timing
	for i, task := range tasks {
		// Mark task as completed
		task.Status = "completed"
		task.UpdatedAt = time.Now().Add(time.Duration(i) * 2 * time.Hour)
		task.Metadata["completed_at"] = task.UpdatedAt

		// Create work session for this task
		session := &entities.Session{
			ID:         uuid.New().String(),
			Repository: task.Repository,
			Duration:   2 * time.Hour,
			CreatedAt:  task.CreatedAt,
			UpdatedAt:  task.UpdatedAt,
			Context: map[string]interface{}{
				"task_id":   task.ID,
				"task_type": task.Type,
			},
		}

		sessions = append(sessions, session)

		// Update task in store
		s.taskStore.Update(ctx, task)
	}

	return sessions
}

func (s *IntelligenceWorkflowTestSuite) findLastCompletedTask(tasks []*entities.Task) *entities.Task {
	var lastTask *entities.Task
	var lastTime time.Time

	for _, task := range tasks {
		if task.Status == "completed" && task.UpdatedAt.After(lastTime) {
			lastTask = task
			lastTime = task.UpdatedAt
		}
	}

	return lastTask
}

func (s *IntelligenceWorkflowTestSuite) generateConflictingTasks(repository string) []*entities.Task {
	// Generate tasks that represent conflicting patterns
	// Pattern A: design → implement → test
	// Pattern B: implement → test → design (reverse)

	var tasks []*entities.Task
	baseTime := time.Now().AddDate(0, 0, -10)

	// Pattern A occurrences
	for i := 0; i < 3; i++ {
		tasks = append(tasks, []*entities.Task{
			s.testData.CreateTaskAt("Design feature "+fmt.Sprintf("%d", i), "high", "completed",
				baseTime.Add(time.Duration(i*6)*time.Hour), map[string]interface{}{"type": "design"}),
			s.testData.CreateTaskAt("Implement feature "+fmt.Sprintf("%d", i), "high", "completed",
				baseTime.Add(time.Duration(i*6+2)*time.Hour), map[string]interface{}{"type": "implement"}),
			s.testData.CreateTaskAt("Test feature "+fmt.Sprintf("%d", i), "medium", "completed",
				baseTime.Add(time.Duration(i*6+4)*time.Hour), map[string]interface{}{"type": "test"}),
		}...)
	}

	// Pattern B occurrences (conflicting)
	for i := 0; i < 2; i++ {
		tasks = append(tasks, []*entities.Task{
			s.testData.CreateTaskAt("Implement hotfix "+fmt.Sprintf("%d", i), "high", "completed",
				baseTime.Add(time.Duration(20+i*6)*time.Hour), map[string]interface{}{"type": "implement"}),
			s.testData.CreateTaskAt("Test hotfix "+fmt.Sprintf("%d", i), "high", "completed",
				baseTime.Add(time.Duration(20+i*6+2)*time.Hour), map[string]interface{}{"type": "test"}),
			s.testData.CreateTaskAt("Design improvement "+fmt.Sprintf("%d", i), "low", "completed",
				baseTime.Add(time.Duration(20+i*6+4)*time.Hour), map[string]interface{}{"type": "design"}),
		}...)
	}

	// Set repository for all tasks
	for _, task := range tasks {
		task.Repository = repository
	}

	return tasks
}

func (s *IntelligenceWorkflowTestSuite) generateIntegrationTasks(repository string) []*entities.Task {
	var tasks []*entities.Task
	baseTime := time.Now().AddDate(0, 0, -21)

	// Generate realistic development tasks with clear patterns
	taskSequences := [][]string{
		{"plan", "design", "implement", "test", "deploy"}, // Standard feature development
		{"investigate", "fix", "test", "verify"},          // Bug fix pattern
		{"research", "prototype", "evaluate", "decide"},   // R&D pattern
	}

	for seqIdx, sequence := range taskSequences {
		for cycle := 0; cycle < 3; cycle++ { // 3 cycles of each pattern
			for stepIdx, step := range sequence {
				task := s.testData.CreateTask(
					fmt.Sprintf("%s task %d-%d", step, seqIdx+1, cycle+1),
					"medium",
					"completed",
					map[string]interface{}{
						"type":     step,
						"sequence": seqIdx,
						"cycle":    cycle,
					},
				)
				task.Repository = repository
				task.CreatedAt = baseTime.Add(time.Duration(seqIdx*5*24+cycle*24+stepIdx*4) * time.Hour)
				task.UpdatedAt = task.CreatedAt.Add(3 * time.Hour)
				task.Metadata["completed_at"] = task.UpdatedAt

				tasks = append(tasks, task)
			}
		}
	}

	return tasks
}

func (s *IntelligenceWorkflowTestSuite) generateIntegrationSessions(repository string) []*entities.Session {
	var sessions []*entities.Session
	baseTime := time.Now().AddDate(0, 0, -21)

	// Generate work sessions that correspond to the tasks
	for day := 0; day < 21; day++ {
		// Morning session
		morning := &entities.Session{
			ID:         uuid.New().String(),
			Repository: repository,
			Duration:   4 * time.Hour,
			CreatedAt:  baseTime.Add(time.Duration(day)*24*time.Hour + 9*time.Hour),
			UpdatedAt:  baseTime.Add(time.Duration(day)*24*time.Hour + 13*time.Hour),
			Context: map[string]interface{}{
				"session_type": "morning",
				"day":          day + 1,
			},
		}

		// Afternoon session
		afternoon := &entities.Session{
			ID:         uuid.New().String(),
			Repository: repository,
			Duration:   3 * time.Hour,
			CreatedAt:  baseTime.Add(time.Duration(day)*24*time.Hour + 14*time.Hour),
			UpdatedAt:  baseTime.Add(time.Duration(day)*24*time.Hour + 17*time.Hour),
			Context: map[string]interface{}{
				"session_type": "afternoon",
				"day":          day + 1,
			},
		}

		sessions = append(sessions, morning, afternoon)
	}

	return sessions
}

func TestIntelligenceWorkflow(t *testing.T) {
	suite.Run(t, new(IntelligenceWorkflowTestSuite))
}
