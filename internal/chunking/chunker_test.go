package chunking

import (
	"context"
	"testing"

	"mcp-memory/internal/config"
	"mcp-memory/pkg/types"
)

// MockEmbeddingService implements a mock embedding service for testing
type MockEmbeddingService struct{}

func (m *MockEmbeddingService) GenerateEmbedding(_ context.Context, content string) ([]float64, error) {
	return []float64{0.1, 0.2, 0.3, 0.4, 0.5}, nil
}

func (m *MockEmbeddingService) GenerateBatchEmbeddings(_ context.Context, contents []string) ([][]float64, error) {
	embeddings := make([][]float64, len(contents))
	for i := range contents {
		embeddings[i] = []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	}
	return embeddings, nil
}

func (m *MockEmbeddingService) HealthCheck(_ context.Context) error {
	return nil
}

func (m *MockEmbeddingService) GetDimension() int {
	return 5
}

func (m *MockEmbeddingService) GetModel() string {
	return "mock-model"
}

func TestProcessConversation(t *testing.T) {
	cfg := &config.ChunkingConfig{
		MaxContentLength:      1000,
		TimeThresholdMinutes:  30,
		FileChangeThreshold:   5,
		TodoCompletionTrigger: true,
	}

	embeddingService := &MockEmbeddingService{}
	cs := NewService(cfg, embeddingService)

	ctx := context.Background()
	sessionID := "test-session"

	// Test simple conversation
	conversation := "Human: How do I fix this error?\n\nAssistant: You need to check the logs."
	metadata := types.ChunkMetadata{Repository: "test-repo"}

	chunks, err := cs.ProcessConversation(ctx, sessionID, conversation, metadata)
	if err != nil {
		t.Fatalf("ProcessConversation failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Verify chunk properties
	for _, chunk := range chunks {
		if chunk.SessionID != sessionID {
			t.Errorf("Chunk has wrong session ID: got %s, want %s", chunk.SessionID, sessionID)
		}

		if len(chunk.Embeddings) == 0 {
			t.Error("Chunk missing embeddings")
		}

		if chunk.Summary == "" {
			t.Error("Chunk missing summary")
		}
	}

	// Test empty conversation
	_, err = cs.ProcessConversation(ctx, sessionID, "", metadata)
	if err == nil {
		t.Error("Expected error for empty conversation")
	}
}

func TestCreateChunk(t *testing.T) {
	cfg := &config.ChunkingConfig{
		MaxContentLength:      1000,
		TimeThresholdMinutes:  30,
		FileChangeThreshold:   5,
		TodoCompletionTrigger: true,
	}

	embeddingService := &MockEmbeddingService{}
	cs := NewService(cfg, embeddingService)

	ctx := context.Background()
	sessionID := "test-session-2"

	// Test problem detection
	content := "I'm getting an error when running the tests"
	metadata := types.ChunkMetadata{Repository: "test-repo"}

	chunk, err := cs.CreateChunk(ctx, sessionID, content, metadata)
	if err != nil {
		t.Fatalf("CreateChunk failed: %v", err)
	}

	if chunk.Type != types.ChunkTypeProblem {
		t.Errorf("Expected chunk type %v, got %v", types.ChunkTypeProblem, chunk.Type)
	}

	// Test solution detection
	content = "I fixed the issue by updating the dependencies"
	chunk, err = cs.CreateChunk(ctx, sessionID, content, metadata)
	if err != nil {
		t.Fatalf("CreateChunk failed: %v", err)
	}

	if chunk.Type != types.ChunkTypeSolution {
		t.Errorf("Expected chunk type %v, got %v", types.ChunkTypeSolution, chunk.Type)
	}
}
