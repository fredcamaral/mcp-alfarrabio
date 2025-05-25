package intelligence

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"mcp-memory/pkg/types"
)

func TestLearningEngineCreation(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	
	if learningEngine == nil {
		t.Fatal("Expected learning engine to be created")
	}
	
	if learningEngine.patternEngine != patternEngine {
		t.Error("Expected pattern engine to be set")
	}
	
	if learningEngine.knowledgeGraph != graphBuilder {
		t.Error("Expected knowledge graph to be set")
	}
	
	if !learningEngine.isLearning {
		t.Error("Expected learning to be enabled by default")
	}
	
	if len(learningEngine.objectives) == 0 {
		t.Error("Expected default objectives to be initialized")
	}
	
	if len(learningEngine.adaptationRules) == 0 {
		t.Error("Expected default adaptation rules to be initialized")
	}
}

func TestLearnFromConversation(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	
	// Create test chunks
	chunks := []types.ConversationChunk{
		{
			ID:        "chunk1",
			SessionID: "session1",
			Content:   "I'm having trouble with my Go code. It won't compile.",
			Type:      types.ChunkTypeProblem,
			Timestamp: time.Now(),
		},
		{
			ID:        "chunk2",
			SessionID: "session1",
			Content:   "Let me help you debug that. Can you share the error message?",
			Type:      types.ChunkTypeAnalysis,
			Timestamp: time.Now().Add(1 * time.Minute),
		},
		{
			ID:        "chunk3",
			SessionID: "session1",
			Content:   "Here's the fix: add 'package main' at the top of your file.",
			Type:      types.ChunkTypeSolution,
			Timestamp: time.Now().Add(2 * time.Minute),
		},
	}
	
	// Create feedback
	feedback := &UserFeedback{
		Rating:    5,
		Comments:  "Very helpful, solved my problem quickly!",
		Helpful:   true,
		Accurate:  true,
		Relevant:  true,
		Timestamp: time.Now(),
	}
	
	// Test learning from conversation
	err := learningEngine.LearnFromConversation(context.Background(), chunks, "success", feedback)
	if err != nil {
		t.Fatalf("Expected no error learning from conversation, got %v", err)
	}
	
	// Check that events were recorded
	if len(learningEngine.events) == 0 {
		t.Error("Expected learning events to be recorded")
	}
	
	// Check that metrics were updated
	if len(learningEngine.metrics) == 0 {
		t.Error("Expected metrics to be updated")
	}
	
	// Verify the event was recorded correctly
	event := learningEngine.events[0]
	if event.Type != "conversation" {
		t.Errorf("Expected event type 'conversation', got '%s'", event.Type)
	}
	
	if event.Outcome != "success" {
		t.Errorf("Expected outcome 'success', got '%s'", event.Outcome)
	}
	
	if !event.Success {
		t.Error("Expected event to be marked as successful")
	}
	
	if event.Feedback != feedback {
		t.Error("Expected feedback to be attached to event")
	}
}

func TestLearnFromFeedback(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	
	// Create feedback
	feedback := &UserFeedback{
		Rating:      4,
		Comments:    "Good answer but could be more detailed",
		Helpful:     true,
		Accurate:    true,
		Relevant:    true,
		Suggestions: []string{"add more examples"},
		Timestamp:   time.Now(),
	}
	
	err := learningEngine.LearnFromFeedback(context.Background(), "chunk123", feedback)
	if err != nil {
		t.Fatalf("Expected no error learning from feedback, got %v", err)
	}
	
	// Check that feedback event was recorded
	if len(learningEngine.events) == 0 {
		t.Error("Expected feedback event to be recorded")
	}
	
	event := learningEngine.events[0]
	if event.Type != "feedback" {
		t.Errorf("Expected event type 'feedback', got '%s'", event.Type)
	}
}

func TestGetAdaptationRecommendations(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	
	// Add some events to simulate poor performance
	for i := 0; i < 10; i++ {
		event := LearningEvent{
			ID:      fmt.Sprintf("event_%d", i),
			Type:    "conversation",
			Outcome: "failure",
			Success: false,
			Metrics: map[string]float64{
				"duration":        6.0, // Slow response
				"pattern_accuracy": 0.4, // Low accuracy
			},
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
		}
		learningEngine.addEvent(event)
	}
	
	recommendations, err := learningEngine.GetAdaptationRecommendations(context.Background())
	if err != nil {
		t.Fatalf("Expected no error getting recommendations, got %v", err)
	}
	
	if len(recommendations) == 0 {
		t.Error("Expected to get adaptation recommendations for poor performance")
	}
	
	// Check for specific recommendations
	foundAccuracyRec := false
	foundSpeedRec := false
	
	for _, rec := range recommendations {
		if strings.Contains(rec.Name, "Accuracy") {
			foundAccuracyRec = true
		}
		if strings.Contains(rec.Name, "Speed") {
			foundSpeedRec = true
		}
	}
	
	if !foundAccuracyRec {
		t.Error("Expected accuracy improvement recommendation")
	}
	if !foundSpeedRec {
		t.Error("Expected speed improvement recommendation")
	}
}

func TestGetLearningStats(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	
	// Add some test events
	successEvent := LearningEvent{
		ID:        "success_event",
		Type:      "conversation",
		Outcome:   "success",
		Success:   true,
		Metrics:   map[string]float64{"duration": 2.0},
		Timestamp: time.Now(),
	}
	
	failureEvent := LearningEvent{
		ID:        "failure_event",
		Type:      "conversation",
		Outcome:   "failure",
		Success:   false,
		Metrics:   map[string]float64{"duration": 5.0},
		Timestamp: time.Now(),
	}
	
	learningEngine.addEvent(successEvent)
	learningEngine.addEvent(failureEvent)
	
	stats := learningEngine.GetLearningStats()
	
	// Check basic stats
	if stats["is_learning"] != true {
		t.Error("Expected is_learning to be true")
	}
	
	if stats["total_events"] != 2 {
		t.Errorf("Expected 2 total events, got %v", stats["total_events"])
	}
	
	if stats["total_objectives"] == 0 {
		t.Error("Expected some objectives")
	}
	
	if stats["total_rules"] == 0 {
		t.Error("Expected some adaptation rules")
	}
	
	// Check performance stats
	successRate, ok := stats["success_rate"].(float64)
	if !ok {
		t.Error("Expected success_rate to be float64")
	} else if successRate != 0.5 {
		t.Errorf("Expected success rate 0.5, got %f", successRate)
	}
}

func TestLearningMetricsUpdate(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	
	// Create event with metrics
	event := LearningEvent{
		ID:   "test_event",
		Type: "test",
		Metrics: map[string]float64{
			"accuracy":     0.8,
			"relevance":    0.9,
			"response_time": 2.5,
		},
		Timestamp: time.Now(),
	}
	
	// Update metrics
	learningEngine.updateMetrics(event)
	
	// Check that metrics were created
	if len(learningEngine.metrics) != 3 {
		t.Errorf("Expected 3 metrics, got %d", len(learningEngine.metrics))
	}
	
	// Check specific metric
	if metric, exists := learningEngine.metrics["accuracy"]; exists {
		if metric.Value != 0.8 {
			t.Errorf("Expected accuracy metric value 0.8, got %f", metric.Value)
		}
		if metric.MeasurementCount != 1 {
			t.Errorf("Expected measurement count 1, got %d", metric.MeasurementCount)
		}
	} else {
		t.Error("Expected accuracy metric to exist")
	}
	
	// Add another event to test averaging
	event2 := LearningEvent{
		ID:   "test_event_2",
		Type: "test",
		Metrics: map[string]float64{
			"accuracy": 0.6,
		},
		Timestamp: time.Now(),
	}
	
	learningEngine.updateMetrics(event2)
	
	// Check that accuracy was averaged
	if metric, exists := learningEngine.metrics["accuracy"]; exists {
		expected := (0.8 + 0.6) / 2.0
		if metric.Value != expected {
			t.Errorf("Expected averaged accuracy %f, got %f", expected, metric.Value)
		}
		if metric.MeasurementCount != 2 {
			t.Errorf("Expected measurement count 2, got %d", metric.MeasurementCount)
		}
	}
}

func TestObjectiveProgressUpdate(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	
	// Find an objective and its target metric
	var testObjective *LearningObjective
	for _, obj := range learningEngine.objectives {
		if obj.IsActive {
			testObjective = obj
			break
		}
	}
	
	if testObjective == nil {
		t.Fatal("Expected at least one active objective")
	}
	
	// Create a metric that matches the objective
	learningEngine.metrics[testObjective.TargetMetric] = &LearningMetric{
		Name:             testObjective.TargetMetric,
		Value:            testObjective.TargetValue * 0.7, // 70% of target
		Trend:            "improving",
		LastUpdated:      time.Now(),
		MeasurementCount: 1,
	}
	
	// Update objective progress
	learningEngine.updateObjectiveProgress()
	
	// Check that progress was calculated
	expectedProgress := 0.7 // 70% of target
	if math.Abs(testObjective.Progress - expectedProgress) > 0.001 {
		t.Errorf("Expected progress %f, got %f", expectedProgress, testObjective.Progress)
	}
}

func TestLearningEngineEnableDisable(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	graphBuilder := NewGraphBuilder(patternEngine)
	learningEngine := NewLearningEngine(patternEngine, graphBuilder)
	
	// Test initial state
	if !learningEngine.IsLearning() {
		t.Error("Expected learning to be enabled initially")
	}
	
	// Test disable
	learningEngine.DisableLearning()
	if learningEngine.IsLearning() {
		t.Error("Expected learning to be disabled")
	}
	
	// Test enable
	learningEngine.EnableLearning()
	if !learningEngine.IsLearning() {
		t.Error("Expected learning to be enabled")
	}
	
	// Test that learning is skipped when disabled
	learningEngine.DisableLearning()
	
	chunks := []types.ConversationChunk{
		{
			ID:      "test_chunk",
			Content: "test content",
			Type:    types.ChunkTypeProblem,
		},
	}
	
	initialEventCount := len(learningEngine.events)
	
	err := learningEngine.LearnFromConversation(context.Background(), chunks, "success", nil)
	if err != nil {
		t.Fatalf("Expected no error even when learning disabled, got %v", err)
	}
	
	// Should not have added any events
	if len(learningEngine.events) != initialEventCount {
		t.Error("Expected no events to be added when learning is disabled")
	}
}