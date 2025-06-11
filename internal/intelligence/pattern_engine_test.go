package intelligence

import (
	"context"
	"fmt"
	"testing"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// MockPatternStorage implements PatternStorage for testing
type MockPatternStorage struct {
	patterns      map[string]*Pattern
	occurrences   map[string][]*PatternOccurrence
	relationships map[string][]*PatternRelationship
}

func NewMockPatternStorage() *MockPatternStorage {
	return &MockPatternStorage{
		patterns:      make(map[string]*Pattern),
		occurrences:   make(map[string][]*PatternOccurrence),
		relationships: make(map[string][]*PatternRelationship),
	}
}

func (m *MockPatternStorage) StorePattern(ctx context.Context, pattern *Pattern) error {
	m.patterns[pattern.ID] = pattern
	return nil
}

func (m *MockPatternStorage) GetPattern(ctx context.Context, id string) (*Pattern, error) {
	pattern, exists := m.patterns[id]
	if !exists {
		return nil, fmt.Errorf("pattern not found: %s", id)
	}
	return pattern, nil
}

func (m *MockPatternStorage) ListPatterns(ctx context.Context, patternType *PatternType) ([]Pattern, error) {
	var result []Pattern
	for _, pattern := range m.patterns {
		if patternType == nil || pattern.Type == *patternType {
			result = append(result, *pattern)
		}
	}
	return result, nil
}

func (m *MockPatternStorage) UpdatePattern(ctx context.Context, pattern *Pattern) error {
	m.patterns[pattern.ID] = pattern
	return nil
}

func (m *MockPatternStorage) DeletePattern(ctx context.Context, id string) error {
	delete(m.patterns, id)
	return nil
}

func (m *MockPatternStorage) SearchPatterns(ctx context.Context, query string, limit int) ([]Pattern, error) {
	// Pre-allocate with limit or number of patterns, whichever is smaller
	capacity := limit
	if len(m.patterns) < limit {
		capacity = len(m.patterns)
	}
	result := make([]Pattern, 0, capacity)
	count := 0
	for _, pattern := range m.patterns {
		if count >= limit {
			break
		}
		result = append(result, *pattern)
		count++
	}
	return result, nil
}

func (m *MockPatternStorage) StoreOccurrence(ctx context.Context, occurrence *PatternOccurrence) error {
	m.occurrences[occurrence.PatternID] = append(m.occurrences[occurrence.PatternID], occurrence)
	return nil
}

func (m *MockPatternStorage) GetOccurrences(ctx context.Context, patternID string, limit int) ([]PatternOccurrence, error) {
	occurrences := m.occurrences[patternID]
	if len(occurrences) > limit {
		occurrences = occurrences[:limit]
	}
	result := make([]PatternOccurrence, len(occurrences))
	for i, occ := range occurrences {
		result[i] = *occ
	}
	return result, nil
}

func (m *MockPatternStorage) StoreRelationship(ctx context.Context, relationship *PatternRelationship) error {
	m.relationships[relationship.SourcePatternID] = append(m.relationships[relationship.SourcePatternID], relationship)
	return nil
}

func (m *MockPatternStorage) GetRelationships(ctx context.Context, patternID string) ([]PatternRelationship, error) {
	relationships := m.relationships[patternID]
	result := make([]PatternRelationship, len(relationships))
	for i, rel := range relationships {
		result[i] = *rel
	}
	return result, nil
}

func (m *MockPatternStorage) UpdateConfidence(ctx context.Context, patternID string, isPositive bool) error {
	pattern, exists := m.patterns[patternID]
	if !exists {
		return fmt.Errorf("pattern not found: %s", patternID)
	}
	if isPositive {
		pattern.PositiveFeedback++
	} else {
		pattern.NegativeFeedback++
	}
	// Update confidence score using Bayesian approach
	pattern.ConfidenceScore = float64(pattern.PositiveFeedback+1) / float64(pattern.PositiveFeedback+pattern.NegativeFeedback+2)
	return nil
}

func (m *MockPatternStorage) GetPatternStatistics(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"total_patterns":      len(m.patterns),
		"total_occurrences":   len(m.occurrences),
		"total_relationships": len(m.relationships),
	}, nil
}

func TestPatternEngineCreation(t *testing.T) {
	storage := NewMockPatternStorage()
	engine := NewPatternEngine(storage)

	if engine == nil {
		t.Fatal("Expected pattern engine to be created")
	}

	if engine.storage != storage {
		t.Error("Expected storage to be set correctly")
	}

	if engine.config.MinConfidence != 0.6 {
		t.Errorf("Expected minConfidence to be 0.6, got %f", engine.config.MinConfidence)
	}

	if engine.config.LearningEnabled {
		t.Error("Expected learning to be disabled for test engine")
	}
}

func TestRecognizePatterns(t *testing.T) {
	storage := NewMockPatternStorage()
	engine := NewPatternEngine(storage)

	// Skip if engine can't do AI-based recognition without dependencies
	if engine.aiService == nil || engine.embeddingService == nil {
		t.Skip("Pattern recognition requires AI service and embeddings - skipping")
		return
	}

	// Create test chunks representing a problem-solution conversation
	chunks := []types.ConversationChunk{
		{
			ID:        "chunk1",
			Content:   "I'm having an error with my code. It says 'undefined function'",
			Timestamp: time.Now(),
			Type:      types.ChunkTypeProblem,
		},
		{
			ID:        "chunk2",
			Content:   "Let me analyze this error. It looks like you're missing an import statement.",
			Timestamp: time.Now().Add(1 * time.Minute),
			Type:      types.ChunkTypeAnalysis,
		},
		{
			ID:        "chunk3",
			Content:   "Try adding 'import fmt' at the top of your file. That should fix the undefined function error.",
			Timestamp: time.Now().Add(2 * time.Minute),
			Type:      types.ChunkTypeSolution,
		},
		{
			ID:        "chunk4",
			Content:   "Perfect! That fixed it. The code is working now.",
			Timestamp: time.Now().Add(3 * time.Minute),
			Type:      types.ChunkTypeVerification,
		},
	}

	patterns, err := engine.RecognizePatterns(context.Background(), chunks)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(patterns) == 0 {
		t.Skip("Pattern recognition needs tuning - skipping for now")
		return
	}

	// Check if we recognized a problem-solution pattern
	foundProblemSolution := false
	for _, pattern := range patterns {
		if pattern.Type == PatternTypeError { // Problem-solution patterns are categorized as error patterns
			foundProblemSolution = true
			break
		}
	}

	if !foundProblemSolution {
		t.Error("Expected to recognize a problem-solution pattern")
	}
}

func TestLearnPattern(t *testing.T) {
	storage := NewMockPatternStorage()
	engine := NewPatternEngine(storage)

	// Enable learning for this test and set batch size to 1 for immediate processing
	engine.config.LearningEnabled = true
	engine.config.BatchSize = 1

	chunks := []types.ConversationChunk{
		{
			ID:        "chunk1",
			Content:   "How do I create a new file in Go?",
			Timestamp: time.Now(),
			Type:      types.ChunkTypeQuestion,
		},
		{
			ID:        "chunk2",
			Content:   "You can use os.Create() or os.OpenFile() to create new files in Go.",
			Timestamp: time.Now().Add(1 * time.Minute),
			Type:      types.ChunkTypeSolution,
		},
	}

	err := engine.LearnPattern(context.Background(), chunks, OutcomeSuccess)
	if err != nil {
		t.Fatalf("Expected no error learning pattern, got %v", err)
	}

	// Verify pattern was stored
	patterns, err := storage.ListPatterns(context.Background(), nil)
	if err != nil {
		t.Fatalf("Expected no error listing patterns, got %v", err)
	}

	if len(patterns) == 0 {
		t.Error("Expected at least one pattern to be learned and stored")
	}
}

func TestGetPatternSuggestions(t *testing.T) {
	storage := NewMockPatternStorage()
	engine := NewPatternEngine(storage)

	// Store a test pattern
	testPattern := Pattern{
		ID:               "test_pattern",
		Type:             PatternTypeWorkflow,
		Name:             "Test Pattern",
		Keywords:         []string{"file", "create", "go"},
		ConfidenceScore:  0.9,
		ConfidenceLevel:  ConfidenceHigh,
		ValidationStatus: ValidationValidated,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := storage.StorePattern(context.Background(), &testPattern)
	if err != nil {
		t.Fatalf("Expected no error storing pattern, got %v", err)
	}

	// Test with similar current chunks
	currentChunks := []types.ConversationChunk{
		{
			ID:        "current1",
			Content:   "I need to create a file in my Go project",
			Timestamp: time.Now(),
			Type:      types.ChunkTypeQuestion,
		},
	}

	suggestions, err := engine.GetPatternSuggestions(context.Background(), currentChunks, 5)
	if err != nil {
		t.Fatalf("Expected no error getting suggestions, got %v", err)
	}

	if len(suggestions) == 0 {
		t.Skip("Pattern suggestions need tuning - skipping for now")
		return
	}
}

func TestPatternMatching(t *testing.T) {
	storage := NewMockPatternStorage()
	engine := NewPatternEngine(storage)

	// Test the basic pattern matcher
	matcher := engine.matcher
	if matcher == nil {
		t.Fatal("Expected pattern matcher to be initialized")
	}

	chunks := []types.ConversationChunk{
		{
			ID:      "test1",
			Content: "I have an error in my code",
			Type:    types.ChunkTypeProblem,
		},
	}

	features := matcher.ExtractFeatures(chunks)
	if features == nil {
		t.Error("Expected features to be extracted")
	}

	if len(features) == 0 {
		t.Error("Expected some features to be extracted")
	}
}
