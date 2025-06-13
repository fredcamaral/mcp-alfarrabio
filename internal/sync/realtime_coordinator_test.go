package sync

import (
	"context"
	"testing"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// MockWebSocketHandler implements a mock WebSocket handler for testing
type MockWebSocketHandler struct {
	lastEvent *BroadcastEvent
}

type BroadcastEvent struct {
	EventType  string
	Action     string
	ChunkID    string
	Repository string
	SessionID  string
	Data       interface{}
}

func (m *MockWebSocketHandler) BroadcastMemoryEvent(eventType, action, chunkID, repository, sessionID string, data interface{}) error {
	m.lastEvent = &BroadcastEvent{
		EventType:  eventType,
		Action:     action,
		ChunkID:    chunkID,
		Repository: repository,
		SessionID:  sessionID,
		Data:       data,
	}
	return nil
}

func TestRealtimeSyncCoordinator_BroadcastChunkEvent(t *testing.T) {
	// Create mock handler
	mockHandler := &MockWebSocketHandler{}

	// Create coordinator
	coordinator := NewRealtimeSyncCoordinator(mockHandler)

	// Create test chunk
	chunk := &types.ConversationChunk{
		ID:        "test-chunk-123",
		Content:   "Test chunk content",
		Summary:   "Test summary",
		SessionID: "test-session",
		Type:      types.ChunkTypeSolution,
		Timestamp: time.Now(),
		Metadata: types.ChunkMetadata{
			Repository:    "github.com/test/repo",
			Tags:          []string{"test", "solution"},
			Outcome:       types.OutcomeSuccess,
			Difficulty:    types.DifficultyModerate,
			FilesModified: []string{"test.go"},
			ToolsUsed:     []string{"go test"},
		},
	}

	// Test broadcasting chunk created event
	ctx := context.Background()
	err := coordinator.BroadcastChunkEvent(ctx, EventTypeChunkCreated, chunk)

	// Verify no error
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify event was broadcast
	if mockHandler.lastEvent == nil {
		t.Fatal("Expected event to be broadcast, but no event was recorded")
	}

	// Verify event details
	event := mockHandler.lastEvent
	if event.EventType != string(EventTypeChunkCreated) {
		t.Errorf("Expected event type %s, got %s", EventTypeChunkCreated, event.EventType)
	}

	if event.Action != "created" {
		t.Errorf("Expected action 'created', got %s", event.Action)
	}

	if event.ChunkID != chunk.ID {
		t.Errorf("Expected chunk ID %s, got %s", chunk.ID, event.ChunkID)
	}

	if event.Repository != chunk.Metadata.Repository {
		t.Errorf("Expected repository %s, got %s", chunk.Metadata.Repository, event.Repository)
	}

	if event.SessionID != chunk.SessionID {
		t.Errorf("Expected session ID %s, got %s", chunk.SessionID, event.SessionID)
	}

	// Verify event data structure
	eventData, ok := event.Data.(MemoryChangeEvent)
	if !ok {
		t.Fatal("Expected event data to be MemoryChangeEvent")
	}

	if eventData.ChunkID != chunk.ID {
		t.Errorf("Expected event data chunk ID %s, got %s", chunk.ID, eventData.ChunkID)
	}

	if eventData.Content != chunk.Content {
		t.Errorf("Expected event data content %s, got %s", chunk.Content, eventData.Content)
	}

	if eventData.Summary != chunk.Summary {
		t.Errorf("Expected event data summary %s, got %s", chunk.Summary, eventData.Summary)
	}

	// Verify metadata
	if eventData.Metadata["type"] != string(chunk.Type) {
		t.Errorf("Expected metadata type %s, got %v", string(chunk.Type), eventData.Metadata["type"])
	}

	if eventData.Metadata["outcome"] != string(chunk.Metadata.Outcome) {
		t.Errorf("Expected metadata outcome %s, got %v", string(chunk.Metadata.Outcome), eventData.Metadata["outcome"])
	}
}

func TestRealtimeSyncCoordinator_ActionFromEventType(t *testing.T) {
	coordinator := &RealtimeSyncCoordinator{}

	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventTypeChunkCreated, "created"},
		{EventTypeChunkUpdated, "updated"},
		{EventTypeChunkDeleted, "deleted"},
		{EventTypeTaskCreated, "created"},
		{EventTypeTaskUpdated, "updated"},
		{EventTypeTaskDeleted, "deleted"},
		{EventType("unknown"), "changed"},
	}

	for _, test := range tests {
		result := coordinator.actionFromEventType(test.eventType)
		if result != test.expected {
			t.Errorf("actionFromEventType(%s) = %s, expected %s",
				test.eventType, result, test.expected)
		}
	}
}

func TestRealtimeSyncCoordinator_Disabled(t *testing.T) {
	// Create coordinator with nil handler (disabled)
	coordinator := NewRealtimeSyncCoordinator(nil)

	// Verify it's disabled
	if coordinator.IsEnabled() {
		t.Error("Expected coordinator to be disabled with nil handler")
	}

	// Create test chunk
	chunk := &types.ConversationChunk{
		ID:        "test-chunk-123",
		Content:   "Test chunk content",
		SessionID: "test-session",
		Type:      types.ChunkTypeSolution,
		Timestamp: time.Now(),
		Metadata: types.ChunkMetadata{
			Repository: "github.com/test/repo",
		},
	}

	// Test that broadcasting when disabled doesn't error
	ctx := context.Background()
	err := coordinator.BroadcastChunkEvent(ctx, EventTypeChunkCreated, chunk)

	// Should not error even when disabled
	if err != nil {
		t.Errorf("Expected no error when disabled, got: %v", err)
	}
}

func TestRealtimeSyncCoordinator_EnableDisable(t *testing.T) {
	mockHandler := &MockWebSocketHandler{}
	coordinator := NewRealtimeSyncCoordinator(mockHandler)

	// Should be enabled by default with valid handler
	if !coordinator.IsEnabled() {
		t.Error("Expected coordinator to be enabled by default with valid handler")
	}

	// Test disable
	coordinator.Disable()
	if coordinator.IsEnabled() {
		t.Error("Expected coordinator to be disabled after calling Disable()")
	}

	// Test enable
	coordinator.Enable()
	if !coordinator.IsEnabled() {
		t.Error("Expected coordinator to be enabled after calling Enable()")
	}
}
