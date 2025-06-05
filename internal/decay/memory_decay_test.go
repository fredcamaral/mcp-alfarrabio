package decay

import (
	"context"
	"fmt"
	"lerian-mcp-memory/pkg/types"
	"testing"
	"time"
)

// MockMemoryStore for testing
type MockMemoryStore struct {
	chunks  map[string]types.ConversationChunk
	updates int
	deletes int
}

func NewMockMemoryStore() *MockMemoryStore {
	return &MockMemoryStore{
		chunks: make(map[string]types.ConversationChunk),
	}
}

func (m *MockMemoryStore) GetAllChunks(ctx context.Context, repository string) ([]types.ConversationChunk, error) {
	chunks := make([]types.ConversationChunk, 0)
	for id := range m.chunks {
		if repository == "" || m.chunks[id].Metadata.Repository == repository {
			chunks = append(chunks, m.chunks[id])
		}
	}
	return chunks, nil
}

func (m *MockMemoryStore) UpdateChunk(ctx context.Context, chunk *types.ConversationChunk) error {
	m.chunks[chunk.ID] = *chunk
	m.updates++
	return nil
}

func (m *MockMemoryStore) DeleteChunk(ctx context.Context, chunkID string) error {
	delete(m.chunks, chunkID)
	m.deletes++
	return nil
}

func (m *MockMemoryStore) StoreChunk(ctx context.Context, chunk *types.ConversationChunk) error {
	m.chunks[chunk.ID] = *chunk
	return nil
}

// MockSummarizer for testing
type MockSummarizer struct {
	summarizeCalls int
}

func (m *MockSummarizer) Summarize(ctx context.Context, chunks []types.ConversationChunk) (string, error) {
	m.summarizeCalls++
	return "Test summary", nil
}

func (m *MockSummarizer) SummarizeChain(ctx context.Context, chunks []types.ConversationChunk) (types.ConversationChunk, error) {
	m.summarizeCalls++
	return types.ConversationChunk{
		ID:        "summary-" + chunks[0].ID,
		SessionID: chunks[0].SessionID,
		Timestamp: time.Now(),
		Type:      types.ChunkTypeSessionSummary,
		Content:   "Summary of chunks",
		Summary:   "Test summary",
		Metadata:  types.ChunkMetadata{},
	}, nil
}

func createTestChunk(id string, age time.Duration, chunkType types.ChunkType) types.ConversationChunk {
	return types.ConversationChunk{
		ID:        id,
		SessionID: "test-session",
		Timestamp: time.Now().Add(-age),
		Type:      chunkType,
		Content:   "Test content",
		Summary:   "Test summary",
		Metadata: types.ChunkMetadata{
			Repository: "test-repo",
			Outcome:    types.OutcomeSuccess,
			Difficulty: types.DifficultyModerate,
		},
	}
}

func TestMemoryDecayManager_CalculateScore(t *testing.T) {
	config := DefaultDecayConfig()
	manager := NewMemoryDecayManager(config, nil, nil)

	tests := []struct {
		name     string
		chunk    types.ConversationChunk
		minScore float64
		maxScore float64
	}{
		{
			name:     "Recent chunk",
			chunk:    createTestChunk("1", 1*time.Hour, types.ChunkTypeDiscussion),
			minScore: 0.9,
			maxScore: 1.0,
		},
		{
			name:     "Week old chunk",
			chunk:    createTestChunk("2", 7*24*time.Hour, types.ChunkTypeDiscussion),
			minScore: 0.8,
			maxScore: 0.95,
		},
		{
			name:     "Month old chunk",
			chunk:    createTestChunk("3", 30*24*time.Hour, types.ChunkTypeDiscussion),
			minScore: 0.95, // At exactly 30 days, adaptive decay returns score * 1.0 = 1.0
			maxScore: 1.0,
		},
		{
			name:     "Important decision chunk",
			chunk:    createTestChunk("4", 7*24*time.Hour, types.ChunkTypeArchitectureDecision),
			minScore: 0.85, // Architecture decision type doesn't get boost (only "decision" does)
			maxScore: 0.91,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := manager.calculateChunkScore(&tt.chunk)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("Expected score between %f and %f, got %f", tt.minScore, tt.maxScore, score)
			}
		})
	}
}

func TestMemoryDecayManager_RunDecay(t *testing.T) {
	store := NewMockMemoryStore()
	summarizer := &MockSummarizer{}
	config := &DecayConfig{
		Strategy:               DecayStrategyAdaptive,
		BaseDecayRate:          0.1,
		MinRelevanceScore:      0.7,
		SummarizationThreshold: 0.4,
		DeletionThreshold:      0.1,
		ImportanceBoost:        DefaultDecayConfig().ImportanceBoost,
		RetentionPeriod:        1 * time.Hour, // Short for testing
	}

	manager := NewMemoryDecayManager(config, store, summarizer)
	ctx := context.Background()

	// Add test chunks
	chunks := []types.ConversationChunk{
		createTestChunk("new", 30*time.Minute, types.ChunkTypeDiscussion),                  // Too new, should be kept
		createTestChunk("old", 2*24*time.Hour, types.ChunkTypeDiscussion),                  // Old, might be updated
		createTestChunk("ancient", 180*24*time.Hour, types.ChunkTypeDiscussion),            // Very old, should be deleted
		createTestChunk("important", 30*24*time.Hour, types.ChunkTypeArchitectureDecision), // Old but important
	}

	for i := range chunks {
		_ = store.StoreChunk(ctx, &chunks[i])
	}

	// Run decay
	err := manager.RunDecay(ctx, "")
	if err != nil {
		t.Fatalf("RunDecay failed: %v", err)
	}

	// Check results
	remaining, _ := store.GetAllChunks(ctx, "")

	// New chunk should still exist
	found := false
	for _, chunk := range remaining {
		if chunk.ID == "new" {
			found = true
			break
		}
	}
	if !found {
		t.Error("New chunk should not be decayed")
	}

	// Ancient chunk should be deleted
	found = false
	for _, chunk := range remaining {
		if chunk.ID == "ancient" {
			found = true
			break
		}
	}
	if found {
		t.Error("Ancient chunk should be deleted")
	}

	// Check counters
	if store.deletes == 0 {
		t.Error("Expected at least one deletion")
	}
}

func TestMemoryDecayManager_Summarization(t *testing.T) {
	store := NewMockMemoryStore()
	summarizer := &MockSummarizer{}
	config := &DecayConfig{
		Strategy:               DecayStrategyAdaptive,
		BaseDecayRate:          0.1,
		MinRelevanceScore:      0.7,
		SummarizationThreshold: 0.8, // Higher threshold for testing
		DeletionThreshold:      0.1,
		ImportanceBoost:        DefaultDecayConfig().ImportanceBoost,
		RetentionPeriod:        1 * time.Hour,
	}

	manager := NewMemoryDecayManager(config, store, summarizer)
	ctx := context.Background()

	// Add related chunks that should be summarized
	sessionID := "summarize-session"
	baseTime := time.Now().Add(-50 * 24 * time.Hour) // 50 days ago

	for i := 0; i < 5; i++ {
		chunk := types.ConversationChunk{
			ID:        fmt.Sprintf("chunk-%d", i),
			SessionID: sessionID,
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
			Type:      types.ChunkTypeDiscussion,
			Content:   fmt.Sprintf("Discussion part %d", i),
			Summary:   fmt.Sprintf("Summary %d", i),
			Metadata:  types.ChunkMetadata{},
		}
		_ = store.StoreChunk(ctx, &chunk)
	}

	// Run decay
	err := manager.RunDecay(ctx, "")
	if err != nil {
		t.Fatalf("RunDecay failed: %v", err)
	}

	// Check that summarization was called
	if summarizer.summarizeCalls == 0 {
		t.Error("Expected summarization to be called")
	}

	// Check that we have a summary chunk
	remaining, _ := store.GetAllChunks(ctx, "")
	foundSummary := false
	for _, chunk := range remaining {
		if chunk.Type == types.ChunkTypeSessionSummary {
			foundSummary = true
			break
		}
	}
	if !foundSummary {
		t.Error("Expected to find a summary chunk")
	}
}

func TestDecayStrategies(t *testing.T) {
	tests := []struct {
		name     string
		strategy DecayStrategy
		age      time.Duration
		expected float64
		delta    float64
	}{
		{
			name:     "Linear decay - 1 day",
			strategy: DecayStrategyLinear,
			age:      24 * time.Hour,
			expected: 0.9967, // Formula: 1.0 * (1.0 - 0.1*1/30) = 0.9967
			delta:    0.001,
		},
		{
			name:     "Exponential decay - 30 days",
			strategy: DecayStrategyExponential,
			age:      30 * 24 * time.Hour,
			expected: 0.5,
			delta:    0.01,
		},
		{
			name:     "Adaptive decay - 5 days",
			strategy: DecayStrategyAdaptive,
			age:      5 * 24 * time.Hour,
			expected: 0.993, // Formula: 1.0 * (1.0 - 0.1*0.1*5/7) = 0.993
			delta:    0.01,
		},
		{
			name:     "Adaptive decay - 35 days",
			strategy: DecayStrategyAdaptive,
			age:      35 * 24 * time.Hour,
			expected: 0.912, // Formula: 1.0 * 0.6^(5/30) = 0.912
			delta:    0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultDecayConfig()
			config.Strategy = tt.strategy
			manager := NewMemoryDecayManager(config, nil, nil)

			score := manager.applyTimeDecay(1.0, tt.age)
			if score < tt.expected-tt.delta || score > tt.expected+tt.delta {
				t.Errorf("Expected score around %f (±%f), got %f", tt.expected, tt.delta, score)
			}
		})
	}
}

func TestDefaultSummarizer(t *testing.T) {
	summarizer := NewDefaultSummarizer()
	ctx := context.Background()

	chunks := []types.ConversationChunk{
		createTestChunk("1", 24*time.Hour, types.ChunkTypeProblem),
		createTestChunk("2", 23*time.Hour, types.ChunkTypeSolution),
		createTestChunk("3", 22*time.Hour, types.ChunkTypeVerification),
	}

	// Test Summarize
	summary, err := summarizer.Summarize(ctx, chunks)
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}

	if summary == "" {
		t.Error("Expected non-empty summary")
	}

	// Test SummarizeChain
	summaryChunk, err := summarizer.SummarizeChain(ctx, chunks)
	if err != nil {
		t.Fatalf("SummarizeChain failed: %v", err)
	}

	if summaryChunk.Type != types.ChunkTypeSessionSummary {
		t.Errorf("Expected summary chunk type, got %v", summaryChunk.Type)
	}

	if len(summaryChunk.RelatedChunks) != len(chunks) {
		t.Errorf("Expected %d related chunks, got %d", len(chunks), len(summaryChunk.RelatedChunks))
	}
}
