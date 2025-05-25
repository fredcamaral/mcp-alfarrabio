package chunking

import (
	"context"
	"mcp-memory/internal/config"
	"mcp-memory/pkg/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test constants
const (
	testSessionID = "session-123"
)

// MockEmbeddingService for testing
type MockEmbeddingService struct {
	mock.Mock
}

func (m *MockEmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	args := m.Called(ctx, text)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockEmbeddingService) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	args := m.Called(ctx, texts)
	return args.Get(0).([][]float64), args.Error(1)
}

func (m *MockEmbeddingService) GetDimension() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockEmbeddingService) GetModel() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEmbeddingService) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewChunkingService(t *testing.T) {
	cfg := &config.ChunkingConfig{
		MinContentLength:     100,
		MaxContentLength:     4000,
		SimilarityThreshold:  0.8,
		TimeThresholdMinutes: 20,
	}

	mockEmbedding := &MockEmbeddingService{}

	service := NewChunkingService(cfg, mockEmbedding)

	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
	assert.Equal(t, mockEmbedding, service.embeddingService)
}

func TestChunkingService_ShouldCreateChunk(t *testing.T) {
	cfg := &config.ChunkingConfig{
		MinContentLength:      100,
		MaxContentLength:      4000,
		SimilarityThreshold:   0.8,
		TimeThresholdMinutes:  20,
		TodoCompletionTrigger: true,
	}

	mockEmbedding := &MockEmbeddingService{}
	service := NewChunkingService(cfg, mockEmbedding)

	tests := []struct {
		name     string
		context  types.ChunkingContext
		expected bool
	}{
		{
			name: "should chunk when todos completed",
			context: types.ChunkingContext{
				ConversationFlow: types.FlowSolution,
				CurrentTodos: []types.TodoItem{
					{Status: "completed", Content: "Task 1"},
					{Status: "completed", Content: "Task 2"},
				},
			},
			expected: true,
		},
		{
			name: "should chunk on significant time elapsed",
			context: types.ChunkingContext{
				ConversationFlow: types.FlowInvestigation,
				TimeElapsed:      25, // Over 20 minutes
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ShouldCreateChunk(tt.context)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChunkingService_CreateChunk(t *testing.T) {
	cfg := &config.ChunkingConfig{
		MinContentLength:     100,
		MaxContentLength:     4000,
		SimilarityThreshold:  0.8,
		TimeThresholdMinutes: 20,
	}

	mockEmbedding := &MockEmbeddingService{}
	mockEmbedding.On("GenerateEmbedding", mock.Anything, mock.AnythingOfType("string")).Return([]float64{0.1, 0.2, 0.3}, nil)

	service := NewChunkingService(cfg, mockEmbedding)
	ctx := context.Background()

	content := "This is a test conversation chunk that we want to create with proper metadata and embeddings."
	sessionID := testSessionID
	metadata := types.ChunkMetadata{
		Outcome:    types.OutcomeSuccess,
		Difficulty: types.DifficultySimple,
	}

	chunk, err := service.CreateChunk(ctx, sessionID, content, metadata)

	assert.NoError(t, err)
	assert.NotNil(t, chunk)
	assert.Equal(t, content, chunk.Content)
	assert.Equal(t, sessionID, chunk.SessionID)
	assert.NotEmpty(t, chunk.ID)
	assert.NotZero(t, chunk.Timestamp)
	assert.Len(t, chunk.Embeddings, 3)

	mockEmbedding.AssertExpectations(t)
}

func TestChunkingService_CreateChunk_EmptyContent(t *testing.T) {
	cfg := &config.ChunkingConfig{
		MinContentLength:     100,
		MaxContentLength:     4000,
		SimilarityThreshold:  0.8,
		TimeThresholdMinutes: 20,
	}

	mockEmbedding := &MockEmbeddingService{}
	service := NewChunkingService(cfg, mockEmbedding)
	ctx := context.Background()

	sessionID := testSessionID
	metadata := types.ChunkMetadata{
		Outcome:    types.OutcomeSuccess,
		Difficulty: types.DifficultySimple,
	}

	chunk, err := service.CreateChunk(ctx, sessionID, "", metadata)

	assert.Error(t, err)
	assert.Nil(t, chunk)
	assert.Contains(t, err.Error(), "content cannot be empty")
}

func TestChunkingService_CreateChunk_EmbeddingError(t *testing.T) {
	cfg := &config.ChunkingConfig{
		MinContentLength:     100,
		MaxContentLength:     4000,
		SimilarityThreshold:  0.8,
		TimeThresholdMinutes: 20,
	}

	mockEmbedding := &MockEmbeddingService{}
	mockEmbedding.On("GenerateEmbedding", mock.Anything, mock.AnythingOfType("string")).Return([]float64{}, assert.AnError)

	service := NewChunkingService(cfg, mockEmbedding)
	ctx := context.Background()

	content := "Test content"
	sessionID := testSessionID
	metadata := types.ChunkMetadata{
		Outcome:    types.OutcomeSuccess,
		Difficulty: types.DifficultySimple,
	}

	chunk, err := service.CreateChunk(ctx, sessionID, content, metadata)

	assert.Error(t, err)
	assert.Nil(t, chunk)
	assert.Contains(t, err.Error(), "failed to generate embeddings")

	mockEmbedding.AssertExpectations(t)
}
