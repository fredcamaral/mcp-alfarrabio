package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"lerian-mcp-memory/internal/intelligence"
	"lerian-mcp-memory/internal/logging"
)

func setupTestPatternStorage(t *testing.T) (*PatternSQLStorage, func()) {
	// This would normally connect to a test database
	// For now, we'll skip the actual DB tests
	t.Skip("Skipping database tests - needs test DB setup")

	db, err := sql.Open("postgres", "postgres://test:test@localhost/test?sslmode=disable")
	require.NoError(t, err)

	logger := logging.NewNoOpLogger()
	storage := NewPatternSQLStorage(db, logger).(*PatternSQLStorage)

	cleanup := func() {
		db.Close()
	}

	return storage, cleanup
}

func TestPatternSQLStorage_StoreAndGetPattern(t *testing.T) {
	storage, cleanup := setupTestPatternStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Create test pattern
	pattern := &intelligence.Pattern{
		ID:               uuid.New().String(),
		Type:             intelligence.PatternTypeCode,
		Name:             "Test Pattern",
		Description:      "A test pattern for unit testing",
		Category:         "testing",
		Signature:        map[string]interface{}{"test": true},
		Keywords:         []string{"test", "pattern", "unit"},
		RepositoryURL:    "github.com/test/repo",
		FilePatterns:     []string{"*.go", "*.test"},
		Language:         "go",
		ConfidenceScore:  0.85,
		ConfidenceLevel:  intelligence.ConfidenceHigh,
		ValidationStatus: intelligence.ValidationUnvalidated,
		OccurrenceCount:  1,
		PositiveFeedback: 0,
		NegativeFeedback: 0,
		Version:          1,
		Metadata:         map[string]interface{}{"source": "test"},
		Embeddings:       []float64{0.1, 0.2, 0.3},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Store pattern
	err := storage.StorePattern(ctx, pattern)
	assert.NoError(t, err)

	// Retrieve pattern
	retrieved, err := storage.GetPattern(ctx, pattern.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Compare
	assert.Equal(t, pattern.ID, retrieved.ID)
	assert.Equal(t, pattern.Name, retrieved.Name)
	assert.Equal(t, pattern.Type, retrieved.Type)
	assert.Equal(t, pattern.Description, retrieved.Description)
	assert.Equal(t, pattern.ConfidenceScore, retrieved.ConfidenceScore)
	assert.Equal(t, pattern.Keywords, retrieved.Keywords)
}

func TestPatternSQLStorage_ListPatterns(t *testing.T) {
	storage, cleanup := setupTestPatternStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Create test patterns
	patterns := []*intelligence.Pattern{
		{
			ID:               uuid.New().String(),
			Type:             intelligence.PatternTypeCode,
			Name:             "Code Pattern 1",
			Description:      "First code pattern",
			ConfidenceScore:  0.9,
			ValidationStatus: intelligence.ValidationValidated,
			Keywords:         []string{"code", "pattern1"},
			Metadata:         map[string]interface{}{},
			Signature:        map[string]interface{}{},
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               uuid.New().String(),
			Type:             intelligence.PatternTypeWorkflow,
			Name:             "Workflow Pattern 1",
			Description:      "First workflow pattern",
			ConfidenceScore:  0.7,
			ValidationStatus: intelligence.ValidationUnvalidated,
			Keywords:         []string{"workflow", "pattern1"},
			Metadata:         map[string]interface{}{},
			Signature:        map[string]interface{}{},
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	// Store patterns
	for _, p := range patterns {
		err := storage.StorePattern(ctx, p)
		assert.NoError(t, err)
	}

	// List all patterns
	allPatterns, err := storage.ListPatterns(ctx, nil)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(allPatterns), 2)

	// List by type
	codeType := intelligence.PatternTypeCode
	codePatterns, err := storage.ListPatterns(ctx, &codeType)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(codePatterns), 1)

	// Verify all returned patterns are code type
	for _, p := range codePatterns {
		assert.Equal(t, intelligence.PatternTypeCode, p.Type)
	}
}

func TestPatternSQLStorage_UpdatePattern(t *testing.T) {
	storage, cleanup := setupTestPatternStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Create and store initial pattern
	pattern := &intelligence.Pattern{
		ID:               uuid.New().String(),
		Type:             intelligence.PatternTypeCode,
		Name:             "Original Pattern",
		Description:      "Original description",
		ConfidenceScore:  0.5,
		ValidationStatus: intelligence.ValidationUnvalidated,
		Keywords:         []string{"original"},
		Metadata:         map[string]interface{}{},
		Signature:        map[string]interface{}{},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := storage.StorePattern(ctx, pattern)
	assert.NoError(t, err)

	// Update pattern
	pattern.Name = "Updated Pattern"
	pattern.Description = "Updated description"
	pattern.ConfidenceScore = 0.8
	pattern.ValidationStatus = intelligence.ValidationValidated
	pattern.Keywords = append(pattern.Keywords, "updated")

	err = storage.UpdatePattern(ctx, pattern)
	assert.NoError(t, err)

	// Retrieve and verify
	updated, err := storage.GetPattern(ctx, pattern.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Pattern", updated.Name)
	assert.Equal(t, "Updated description", updated.Description)
	assert.Equal(t, 0.8, updated.ConfidenceScore)
	assert.Equal(t, intelligence.ValidationValidated, updated.ValidationStatus)
	assert.Contains(t, updated.Keywords, "updated")
}

func TestPatternSQLStorage_SearchPatterns(t *testing.T) {
	storage, cleanup := setupTestPatternStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Create test patterns with searchable content
	patterns := []*intelligence.Pattern{
		{
			ID:               uuid.New().String(),
			Type:             intelligence.PatternTypeError,
			Name:             "Error Handling Pattern",
			Description:      "Pattern for handling errors gracefully",
			Category:         "error-handling",
			Keywords:         []string{"error", "handling", "exception"},
			ConfidenceScore:  0.9,
			ValidationStatus: intelligence.ValidationValidated,
			Metadata:         map[string]interface{}{},
			Signature:        map[string]interface{}{},
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               uuid.New().String(),
			Type:             intelligence.PatternTypeOptimization,
			Name:             "Performance Optimization",
			Description:      "Pattern for optimizing performance",
			Category:         "performance",
			Keywords:         []string{"performance", "optimization", "speed"},
			ConfidenceScore:  0.75,
			ValidationStatus: intelligence.ValidationUnvalidated,
			Metadata:         map[string]interface{}{},
			Signature:        map[string]interface{}{},
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	// Store patterns
	for _, p := range patterns {
		err := storage.StorePattern(ctx, p)
		assert.NoError(t, err)
	}

	// Search by name
	results, err := storage.SearchPatterns(ctx, "Error", 10)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
	assert.Contains(t, results[0].Name, "Error")

	// Search by keyword
	results, err = storage.SearchPatterns(ctx, "optimization", 10)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)

	// Search by category
	results, err = storage.SearchPatterns(ctx, "performance", 10)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestPatternSQLStorage_StoreAndGetOccurrence(t *testing.T) {
	storage, cleanup := setupTestPatternStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Create test pattern first
	pattern := &intelligence.Pattern{
		ID:               uuid.New().String(),
		Type:             intelligence.PatternTypeCode,
		Name:             "Test Pattern for Occurrence",
		Description:      "Pattern to test occurrences",
		ConfidenceScore:  0.8,
		ValidationStatus: intelligence.ValidationUnvalidated,
		Keywords:         []string{"test"},
		Metadata:         map[string]interface{}{},
		Signature:        map[string]interface{}{},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := storage.StorePattern(ctx, pattern)
	assert.NoError(t, err)

	// Create occurrence
	occurrence := &intelligence.PatternOccurrence{
		ID:                 uuid.New().String(),
		PatternID:          pattern.ID,
		RepositoryURL:      "github.com/test/repo",
		FilePath:           "main.go",
		LineStart:          10,
		LineEnd:            20,
		CodeSnippet:        "func main() { ... }",
		SurroundingContext: "package main\n\nfunc main() { ... }",
		DetectionScore:     0.85,
		DetectionMethod:    "ai-assisted",
		SessionID:          "test-session",
		ChunkID:            "test-chunk",
		Metadata:           map[string]interface{}{"test": true},
		DetectedAt:         time.Now(),
	}

	// Store occurrence
	err = storage.StoreOccurrence(ctx, occurrence)
	assert.NoError(t, err)

	// Get occurrences
	occurrences, err := storage.GetOccurrences(ctx, pattern.ID, 10)
	assert.NoError(t, err)
	assert.Len(t, occurrences, 1)

	retrieved := occurrences[0]
	assert.Equal(t, occurrence.ID, retrieved.ID)
	assert.Equal(t, occurrence.PatternID, retrieved.PatternID)
	assert.Equal(t, occurrence.FilePath, retrieved.FilePath)
	assert.Equal(t, occurrence.DetectionScore, retrieved.DetectionScore)
}

func TestPatternSQLStorage_UpdateConfidence(t *testing.T) {
	storage, cleanup := setupTestPatternStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Create test pattern
	pattern := &intelligence.Pattern{
		ID:               uuid.New().String(),
		Type:             intelligence.PatternTypeCode,
		Name:             "Confidence Test Pattern",
		Description:      "Testing confidence updates",
		ConfidenceScore:  0.5,
		ConfidenceLevel:  intelligence.ConfidenceMedium,
		ValidationStatus: intelligence.ValidationUnvalidated,
		PositiveFeedback: 1,
		NegativeFeedback: 1,
		Keywords:         []string{"test"},
		Metadata:         map[string]interface{}{},
		Signature:        map[string]interface{}{},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := storage.StorePattern(ctx, pattern)
	assert.NoError(t, err)

	// Update with positive feedback
	err = storage.UpdateConfidence(ctx, pattern.ID, true)
	assert.NoError(t, err)

	// Check updated pattern
	updated, err := storage.GetPattern(ctx, pattern.ID)
	assert.NoError(t, err)
	assert.Equal(t, 2, updated.PositiveFeedback)
	assert.Equal(t, 1, updated.NegativeFeedback)
	assert.Greater(t, updated.ConfidenceScore, 0.5)

	// Update with negative feedback
	err = storage.UpdateConfidence(ctx, pattern.ID, false)
	assert.NoError(t, err)

	// Check updated pattern again
	updated2, err := storage.GetPattern(ctx, pattern.ID)
	assert.NoError(t, err)
	assert.Equal(t, 2, updated2.PositiveFeedback)
	assert.Equal(t, 2, updated2.NegativeFeedback)
	assert.Equal(t, 0.5, updated2.ConfidenceScore) // Should be back to 0.5 (2+1)/(2+2+2)
}

func TestPatternSQLStorage_PatternRelationships(t *testing.T) {
	storage, cleanup := setupTestPatternStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Create two patterns
	pattern1 := &intelligence.Pattern{
		ID:               uuid.New().String(),
		Type:             intelligence.PatternTypeCode,
		Name:             "Pattern 1",
		Description:      "First pattern",
		ConfidenceScore:  0.8,
		ValidationStatus: intelligence.ValidationValidated,
		Keywords:         []string{"pattern1"},
		Metadata:         map[string]interface{}{},
		Signature:        map[string]interface{}{},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	pattern2 := &intelligence.Pattern{
		ID:               uuid.New().String(),
		Type:             intelligence.PatternTypeCode,
		Name:             "Pattern 2",
		Description:      "Second pattern",
		ConfidenceScore:  0.7,
		ValidationStatus: intelligence.ValidationValidated,
		Keywords:         []string{"pattern2"},
		Metadata:         map[string]interface{}{},
		Signature:        map[string]interface{}{},
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := storage.StorePattern(ctx, pattern1)
	assert.NoError(t, err)
	err = storage.StorePattern(ctx, pattern2)
	assert.NoError(t, err)

	// Create relationship
	relationship := &intelligence.PatternRelationship{
		ID:               uuid.New().String(),
		SourcePatternID:  pattern1.ID,
		TargetPatternID:  pattern2.ID,
		RelationshipType: "extends",
		Strength:         0.85,
		Confidence:       0.9,
		Context:          "Pattern 2 extends Pattern 1",
		Examples:         []interface{}{"example1", "example2"},
		Metadata:         map[string]interface{}{"test": true},
		CreatedAt:        time.Now(),
	}

	// Store relationship
	err = storage.StoreRelationship(ctx, relationship)
	assert.NoError(t, err)

	// Get relationships for pattern1
	relationships, err := storage.GetRelationships(ctx, pattern1.ID)
	assert.NoError(t, err)
	assert.Len(t, relationships, 1)

	retrieved := relationships[0]
	assert.Equal(t, relationship.ID, retrieved.ID)
	assert.Equal(t, relationship.SourcePatternID, retrieved.SourcePatternID)
	assert.Equal(t, relationship.TargetPatternID, retrieved.TargetPatternID)
	assert.Equal(t, relationship.RelationshipType, retrieved.RelationshipType)

	// Get relationships for pattern2 (should also return the same relationship)
	relationships2, err := storage.GetRelationships(ctx, pattern2.ID)
	assert.NoError(t, err)
	assert.Len(t, relationships2, 1)
}

func TestPatternSQLStorage_GetPatternStatistics(t *testing.T) {
	storage, cleanup := setupTestPatternStorage(t)
	defer cleanup()

	ctx := context.Background()

	// Create test patterns
	patterns := []*intelligence.Pattern{
		{
			ID:               uuid.New().String(),
			Type:             intelligence.PatternTypeCode,
			Name:             "Code Pattern",
			ConfidenceScore:  0.9,
			ConfidenceLevel:  intelligence.ConfidenceHigh,
			ValidationStatus: intelligence.ValidationValidated,
			OccurrenceCount:  10,
			Keywords:         []string{"code"},
			Metadata:         map[string]interface{}{},
			Signature:        map[string]interface{}{},
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               uuid.New().String(),
			Type:             intelligence.PatternTypeWorkflow,
			Name:             "Workflow Pattern",
			ConfidenceScore:  0.7,
			ConfidenceLevel:  intelligence.ConfidenceMedium,
			ValidationStatus: intelligence.ValidationUnvalidated,
			OccurrenceCount:  5,
			Keywords:         []string{"workflow"},
			Metadata:         map[string]interface{}{},
			Signature:        map[string]interface{}{},
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	// Store patterns
	for _, p := range patterns {
		err := storage.StorePattern(ctx, p)
		assert.NoError(t, err)
	}

	// Get statistics
	stats, err := storage.GetPatternStatistics(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	// Verify statistics structure
	assert.Contains(t, stats, "total_patterns")
	assert.Contains(t, stats, "patterns_by_type")
	assert.Contains(t, stats, "patterns_by_confidence")
	assert.Contains(t, stats, "patterns_by_validation")
	assert.Contains(t, stats, "average_confidence")
	assert.Contains(t, stats, "top_patterns")

	// Verify counts
	totalPatterns, ok := stats["total_patterns"].(int)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, totalPatterns, 2)
}
