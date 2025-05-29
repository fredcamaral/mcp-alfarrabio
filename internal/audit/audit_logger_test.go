package audit

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestAuditLogger_LogEvent(t *testing.T) {
	// Create temporary directory for audit logs
	tempDir, err := os.MkdirTemp("", "audit_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Create audit logger
	logger, err := NewAuditLogger(tempDir)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Stop()

	// Create context with values
	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKeySessionID, "test-session")
	ctx = context.WithValue(ctx, contextKeyUserID, "test-user")
	ctx = context.WithValue(ctx, contextKeyRepository, "test-repo")

	// Log some events
	logger.LogEvent(ctx, EventTypeMemoryStore, "Store memory chunk", "memory", "chunk-123", map[string]interface{}{
		"chunk_type": "solution",
		"tags":       []string{"bug-fix", "performance"},
	})

	logger.LogEvent(ctx, EventTypeMemorySearch, "Search memories", "memory", "", map[string]interface{}{
		"query": "fix database connection",
		"limit": 10,
	})

	// Log an error
	logger.LogError(ctx, EventTypeError, "Failed to store chunk", "memory",
		fmt.Errorf("database connection failed"), nil)

	// Force flush
	logger.mu.Lock()
	logger.flush()
	logger.mu.Unlock()

	// Verify statistics
	stats := logger.GetStatistics()
	totalEvents, ok := stats["total_events"].(int64)
	if !ok || totalEvents < 3 { // Including system start event
		t.Errorf("Expected at least 3 events, got %v", totalEvents)
	}

	errorCount, ok := stats["error_count"].(int64)
	if !ok || errorCount != 1 {
		t.Errorf("Expected 1 error, got %v", errorCount)
	}

	// Verify file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	foundAuditFile := false
	for _, file := range files {
		if isAuditFile(file.Name()) {
			foundAuditFile = true
			break
		}
	}

	if !foundAuditFile {
		t.Error("No audit file was created")
	}
}

func TestAuditLogger_Search(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "audit_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	logger, err := NewAuditLogger(tempDir)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Stop()

	ctx := context.Background()
	ctx = context.WithValue(ctx, contextKeySessionID, "search-test")

	// Log various events
	logger.LogEvent(ctx, EventTypeMemoryStore, "Store chunk 1", "memory", "chunk-1", nil)
	logger.LogEvent(ctx, EventTypeMemoryStore, "Store chunk 2", "memory", "chunk-2", nil)
	logger.LogEvent(ctx, EventTypeMemorySearch, "Search memories", "memory", "", nil)
	logger.LogError(ctx, EventTypeError, "Test error", "system", fmt.Errorf("test error"), nil)

	// Force flush
	logger.mu.Lock()
	logger.flush()
	logger.mu.Unlock()

	// Search for memory store events
	criteria := SearchCriteria{
		EventTypes: []EventType{EventTypeMemoryStore},
		SessionID:  "search-test",
	}

	events, err := logger.Search(ctx, criteria)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 memory store events, got %d", len(events))
	}

	// Search for errors
	successFalse := false
	criteria = SearchCriteria{
		Success: &successFalse,
	}

	events, err = logger.Search(ctx, criteria)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 error event, got %d", len(events))
	}
}

func TestAuditLogger_FileRotation(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "audit_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	logger, err := NewAuditLogger(tempDir)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Stop()

	// Set small max file size and manually rotate
	logger.maxFileSize = 1024 // 1KB

	ctx := context.Background()

	// Log some events
	for i := 0; i < 10; i++ {
		logger.LogEvent(ctx, EventTypeMemoryStore,
			fmt.Sprintf("Store chunk %d", i),
			"memory",
			fmt.Sprintf("chunk-%d", i),
			map[string]interface{}{
				"index": i,
				"data":  "Test data",
			})
	}

	// Force flush and rotate
	logger.mu.Lock()
	logger.flush()
	logger.mu.Unlock()

	// Wait a bit to ensure different timestamp
	time.Sleep(1100 * time.Millisecond) // More than 1 second to ensure different timestamp

	// Manually trigger rotation for test
	logger.mu.Lock()
	_ = logger.rotateFile()
	logger.mu.Unlock()

	// Log more events to the new file
	for i := 10; i < 20; i++ {
		logger.LogEvent(ctx, EventTypeMemoryStore,
			fmt.Sprintf("Store chunk %d", i),
			"memory",
			fmt.Sprintf("chunk-%d", i),
			map[string]interface{}{
				"index": i,
				"data":  "Test data",
			})
	}

	// Final flush
	logger.mu.Lock()
	logger.flush()
	logger.mu.Unlock()

	// Check for multiple audit files
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	auditFileCount := 0
	for _, file := range files {
		if isAuditFile(file.Name()) {
			auditFileCount++
			t.Logf("Found audit file: %s", file.Name())
		}
	}

	if auditFileCount < 2 {
		t.Errorf("Expected at least 2 audit files due to rotation, got %d", auditFileCount)
	}
}

func TestSearchCriteria_Matches(t *testing.T) {
	event := AuditEvent{
		ID:         "test-1",
		Timestamp:  time.Now(),
		EventType:  EventTypeMemoryStore,
		SessionID:  "session-1",
		UserID:     "user-1",
		Repository: "repo-1",
		Resource:   "memory",
		Success:    true,
	}

	tests := []struct {
		name     string
		criteria SearchCriteria
		want     bool
	}{
		{
			name:     "Match all",
			criteria: SearchCriteria{},
			want:     true,
		},
		{
			name: "Match event type",
			criteria: SearchCriteria{
				EventTypes: []EventType{EventTypeMemoryStore},
			},
			want: true,
		},
		{
			name: "No match event type",
			criteria: SearchCriteria{
				EventTypes: []EventType{EventTypeMemorySearch},
			},
			want: false,
		},
		{
			name: "Match session ID",
			criteria: SearchCriteria{
				SessionID: "session-1",
			},
			want: true,
		},
		{
			name: "No match session ID",
			criteria: SearchCriteria{
				SessionID: "session-2",
			},
			want: false,
		},
		{
			name: "Match success",
			criteria: SearchCriteria{
				Success: &[]bool{true}[0],
			},
			want: true,
		},
		{
			name: "No match success",
			criteria: SearchCriteria{
				Success: &[]bool{false}[0],
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.criteria.Matches(event); got != tt.want {
				t.Errorf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
