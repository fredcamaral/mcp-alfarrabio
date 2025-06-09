//go:build integration
// +build integration

package intelligence

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/services"
	"lerian-mcp-memory-cli/tests/testutils"
)

type PatternDetectionTestSuite struct {
	suite.Suite
	taskStore    *testutils.MockTaskStorage
	patternStore *testutils.MockPatternStorage
	detector     services.PatternDetector
	testData     *testutils.TestDataGenerator
}

func (s *PatternDetectionTestSuite) SetupSuite() {
	s.taskStore = testutils.NewMockTaskStorage()
	s.patternStore = testutils.NewMockPatternStorage()
	s.testData = testutils.NewTestDataGenerator()

	s.detector = services.NewPatternDetector(services.PatternDetectorDependencies{
		TaskStore:    s.taskStore,
		PatternStore: s.patternStore,
		AI:           testutils.NewMockAIService(),
		Logger:       slog.Default(),
	})
}

func (s *PatternDetectionTestSuite) TearDownTest() {
	s.taskStore.Clear()
	s.patternStore.Clear()
}

func (s *PatternDetectionTestSuite) TestSequencePatternDetection() {
	ctx := context.Background()

	// Generate tasks with known sequence patterns
	tasks := s.testData.GeneratePatternedTasks("test-repo", []testutils.PatternTemplate{
		{
			Name:     "feature-development",
			Sequence: []string{"design", "implement", "test", "document"},
			Count:    8,
		},
		{
			Name:     "bug-fix",
			Sequence: []string{"investigate", "fix", "test", "verify"},
			Count:    12,
		},
	})

	// Store tasks
	for _, task := range tasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Detect sequence patterns
	patterns, err := s.detector.DetectSequencePatterns(ctx, tasks, 0.1)
	s.Require().NoError(err)
	s.Assert().GreaterOrEqual(len(patterns), 2, "Should detect at least 2 patterns")

	// Verify feature development pattern
	featurePattern := s.findSequencePattern(patterns, []string{"design", "implement", "test", "document"})
	s.Assert().NotNil(featurePattern, "Should detect feature development pattern")
	s.Assert().Greater(featurePattern.Confidence, 0.7, "Pattern confidence should be high")
	s.Assert().GreaterOrEqual(featurePattern.Frequency, 7, "Should find most occurrences")

	// Verify bug fix pattern
	bugPattern := s.findSequencePattern(patterns, []string{"investigate", "fix", "test", "verify"})
	s.Assert().NotNil(bugPattern, "Should detect bug fix pattern")
	s.Assert().Greater(bugPattern.Confidence, 0.7, "Pattern confidence should be high")
	s.Assert().GreaterOrEqual(bugPattern.Frequency, 10, "Should find most occurrences")
}

func (s *PatternDetectionTestSuite) TestWorkflowPatternDetection() {
	ctx := context.Background()

	// Generate tasks with workflow patterns (parallel tasks)
	tasks := s.testData.GenerateWorkflowTasks("workflow-repo", []testutils.WorkflowTemplate{
		{
			Name: "parallel-development",
			Phases: []testutils.WorkflowPhase{
				{
					Name:         "setup",
					Tasks:        []string{"init project", "setup ci"},
					Parallelism:  2,
					Dependencies: []string{},
				},
				{
					Name:         "development",
					Tasks:        []string{"implement frontend", "implement backend", "setup database"},
					Parallelism:  3,
					Dependencies: []string{"setup"},
				},
				{
					Name:         "testing",
					Tasks:        []string{"unit tests", "integration tests"},
					Parallelism:  2,
					Dependencies: []string{"development"},
				},
			},
			Count: 5,
		},
	})

	for _, task := range tasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Detect workflow patterns
	patterns, err := s.detector.DetectWorkflowPatterns(ctx, tasks, 0.1)
	s.Require().NoError(err)
	s.Assert().NotEmpty(patterns, "Should detect workflow patterns")

	// Find parallel development pattern
	parallelPattern := s.findWorkflowPattern(patterns, "parallel-development")
	s.Assert().NotNil(parallelPattern, "Should detect parallel development pattern")
	s.Assert().Greater(parallelPattern.Confidence, 0.6, "Pattern confidence should be reasonable")

	// Verify phases are detected
	s.Assert().Len(parallelPattern.Phases, 3, "Should detect 3 phases")
	setupPhase := s.findPhase(parallelPattern.Phases, "setup")
	s.Assert().NotNil(setupPhase, "Should detect setup phase")
	s.Assert().Equal(2, setupPhase.MaxParallelism, "Setup phase should have parallelism of 2")
}

func (s *PatternDetectionTestSuite) TestTemporalPatternDetection() {
	ctx := context.Background()

	// Generate tasks with temporal patterns
	now := time.Now()
	tasks := s.testData.GenerateTemporalTasks("temporal-repo", []testutils.TemporalTemplate{
		{
			Name:      "daily-standup",
			TaskName:  "daily standup meeting",
			Frequency: 24 * time.Hour,
			Count:     20,
			StartTime: now.Add(-20 * 24 * time.Hour),
		},
		{
			Name:      "weekly-review",
			TaskName:  "weekly sprint review",
			Frequency: 7 * 24 * time.Hour,
			Count:     8,
			StartTime: now.Add(-8 * 7 * 24 * time.Hour),
		},
	})

	for _, task := range tasks {
		err := s.taskStore.Create(ctx, task)
		s.Require().NoError(err)
	}

	// Detect temporal patterns
	patterns, err := s.detector.DetectTemporalPatterns(ctx, tasks, 0.2)
	s.Require().NoError(err)
	s.Assert().NotEmpty(patterns, "Should detect temporal patterns")

	// Verify daily pattern
	dailyPattern := s.findTemporalPattern(patterns, "daily standup meeting")
	s.Assert().NotNil(dailyPattern, "Should detect daily standup pattern")
	s.Assert().InDelta(24*time.Hour, dailyPattern.Frequency, float64(4*time.Hour), "Should detect ~24h frequency")

	// Verify weekly pattern
	weeklyPattern := s.findTemporalPattern(patterns, "weekly sprint review")
	s.Assert().NotNil(weeklyPattern, "Should detect weekly review pattern")
	s.Assert().InDelta(7*24*time.Hour, weeklyPattern.Frequency, float64(24*time.Hour), "Should detect ~weekly frequency")
}

func (s *PatternDetectionTestSuite) TestPatternDetectionPerformance() {
	ctx := context.Background()

	// Generate large dataset
	largeTasks := s.testData.GenerateRandomTasks("perf-repo", 5000)

	for _, task := range largeTasks {
		s.taskStore.Create(ctx, task)
	}

	// Measure sequence pattern detection performance
	start := time.Now()
	patterns, err := s.detector.DetectSequencePatterns(ctx, largeTasks, 0.05)
	sequenceDuration := time.Since(start)

	s.Require().NoError(err)
	s.Assert().Less(sequenceDuration, 10*time.Second, "Sequence pattern detection should complete within 10 seconds")
	s.T().Logf("Detected %d sequence patterns from %d tasks in %v", len(patterns), len(largeTasks), sequenceDuration)

	// Measure workflow pattern detection performance
	start = time.Now()
	workflowPatterns, err := s.detector.DetectWorkflowPatterns(ctx, largeTasks, 0.05)
	workflowDuration := time.Since(start)

	s.Require().NoError(err)
	s.Assert().Less(workflowDuration, 15*time.Second, "Workflow pattern detection should complete within 15 seconds")
	s.T().Logf("Detected %d workflow patterns from %d tasks in %v", len(workflowPatterns), len(largeTasks), workflowDuration)
}

func (s *PatternDetectionTestSuite) TestPatternPersistence() {
	ctx := context.Background()

	// Create a pattern
	pattern := &entities.TaskPattern{
		ID:   uuid.New().String(),
		Type: entities.PatternTypeSequence,
		Name: "test-pattern",
		Sequence: []entities.PatternStep{
			{Order: 1, TaskType: "design", Keywords: []string{"ui", "mockup"}},
			{Order: 2, TaskType: "implement", Keywords: []string{"component", "react"}},
		},
		Confidence:  0.85,
		Frequency:   10,
		SuccessRate: 0.9,
		Repository:  "test-repo",
		ProjectType: entities.ProjectTypeWebApp,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    map[string]interface{}{"test": true},
	}

	// Store pattern
	err := s.patternStore.Create(ctx, pattern)
	s.Require().NoError(err)

	// Retrieve and verify
	retrieved, err := s.patternStore.GetByID(ctx, pattern.ID)
	s.Require().NoError(err)
	s.Assert().Equal(pattern.Name, retrieved.Name)
	s.Assert().Equal(pattern.Type, retrieved.Type)
	s.Assert().Equal(pattern.Confidence, retrieved.Confidence)
	s.Assert().Len(retrieved.Sequence, 2)

	// Update pattern
	pattern.Confidence = 0.95
	pattern.Frequency = 15
	err = s.patternStore.Update(ctx, pattern)
	s.Require().NoError(err)

	// Verify update
	updated, err := s.patternStore.GetByID(ctx, pattern.ID)
	s.Require().NoError(err)
	s.Assert().Equal(0.95, updated.Confidence)
	s.Assert().Equal(15, updated.Frequency)
}

func (s *PatternDetectionTestSuite) TestPatternQuality() {
	ctx := context.Background()

	// Generate high-quality patterns (consistent sequences)
	highQualityTasks := s.testData.GeneratePatternedTasks("high-quality", []testutils.PatternTemplate{
		{
			Name:     "consistent-pattern",
			Sequence: []string{"analyze", "design", "implement", "test", "deploy"},
			Count:    20,
			Variance: 0.1, // Low variance = high consistency
		},
	})

	// Generate low-quality patterns (inconsistent sequences)
	lowQualityTasks := s.testData.GeneratePatternedTasks("low-quality", []testutils.PatternTemplate{
		{
			Name:     "inconsistent-pattern",
			Sequence: []string{"start", "work", "maybe-test", "sometimes-deploy"},
			Count:    10,
			Variance: 0.8, // High variance = low consistency
		},
	})

	allTasks := append(highQualityTasks, lowQualityTasks...)
	for _, task := range allTasks {
		s.taskStore.Create(ctx, task)
	}

	// Detect patterns
	patterns, err := s.detector.DetectSequencePatterns(ctx, allTasks, 0.1)
	s.Require().NoError(err)

	// High-quality pattern should have higher confidence
	highQualityPattern := s.findSequencePattern(patterns, []string{"analyze", "design", "implement", "test", "deploy"})
	if s.Assert().NotNil(highQualityPattern) {
		s.Assert().Greater(highQualityPattern.Confidence, 0.8, "High-quality pattern should have high confidence")
	}

	// Low-quality pattern should have lower confidence (if detected at all)
	lowQualityPattern := s.findSequencePattern(patterns, []string{"start", "work", "maybe-test", "sometimes-deploy"})
	if lowQualityPattern != nil {
		s.Assert().Less(lowQualityPattern.Confidence, 0.6, "Low-quality pattern should have lower confidence")
	}
}

// Helper methods

func (s *PatternDetectionTestSuite) findSequencePattern(patterns []*entities.TaskPattern, sequence []string) *entities.TaskPattern {
	for _, pattern := range patterns {
		if pattern.Type == entities.PatternTypeSequence && len(pattern.Sequence) == len(sequence) {
			match := true
			for i, step := range pattern.Sequence {
				if step.TaskType != sequence[i] {
					match = false
					break
				}
			}
			if match {
				return pattern
			}
		}
	}
	return nil
}

func (s *PatternDetectionTestSuite) findWorkflowPattern(patterns []*entities.TaskPattern, name string) *entities.TaskPattern {
	for _, pattern := range patterns {
		if pattern.Type == entities.PatternTypeWorkflow && pattern.Name == name {
			return pattern
		}
	}
	return nil
}

func (s *PatternDetectionTestSuite) findTemporalPattern(patterns []*entities.TaskPattern, taskName string) *entities.TaskPattern {
	for _, pattern := range patterns {
		if pattern.Type == entities.PatternTypeTemporal {
			for _, keyword := range pattern.CommonKeywords {
				if keyword == taskName {
					return pattern
				}
			}
		}
	}
	return nil
}

func (s *PatternDetectionTestSuite) findPhase(phases []entities.WorkflowPhase, name string) *entities.WorkflowPhase {
	for _, phase := range phases {
		if phase.Name == name {
			return &phase
		}
	}
	return nil
}

func TestPatternDetection(t *testing.T) {
	suite.Run(t, new(PatternDetectionTestSuite))
}
