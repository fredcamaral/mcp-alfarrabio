// Package audit provides comprehensive audit logging capabilities
// for tracking all operations and changes in the MCP Memory Server.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lerian-mcp-memory/internal/logging"
)

// Context key types
type contextKey string

const (
	contextKeySessionID  contextKey = "session_id"
	contextKeyUserID     contextKey = "user_id"
	contextKeyRepository contextKey = "repository"
)

// EventType represents the type of audit event
type EventType string

const (
	// EventTypeMemoryStore represents memory storage operations
	EventTypeMemoryStore EventType = "memory_store"
	// EventTypeMemorySearch represents memory search operations
	EventTypeMemorySearch EventType = "memory_search"
	// EventTypeMemoryUpdate represents memory update operations
	EventTypeMemoryUpdate EventType = "memory_update"
	// EventTypeMemoryDelete represents memory deletion operations
	EventTypeMemoryDelete EventType = "memory_delete"
	// EventTypeDecisionStore represents decision storage operations
	EventTypeDecisionStore EventType = "decision_store"
	// EventTypeRelationshipAdd represents relationship creation operations
	EventTypeRelationshipAdd EventType = "relationship_add"
	// EventTypePatternDetected represents pattern detection events
	EventTypePatternDetected EventType = "pattern_detected"
	// EventTypeContextSwitch represents context switching events
	EventTypeContextSwitch EventType = "context_switch"
	// EventTypeExport represents data export operations
	EventTypeExport EventType = "export"
	// EventTypeImport represents data import operations
	EventTypeImport EventType = "import"
	// EventTypeSystemStart represents system startup events
	EventTypeSystemStart EventType = "system_start"
	// EventTypeSystemShutdown represents system shutdown events
	EventTypeSystemShutdown EventType = "system_shutdown"
	// EventTypeError represents error events
	EventTypeError EventType = "error"
)

// Event represents a single audit log entry
type Event struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	EventType  EventType              `json:"event_type"`
	UserID     string                 `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	Repository string                 `json:"repository,omitempty"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource,omitempty"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Duration   time.Duration          `json:"duration,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
}

// Logger handles persistent audit logging
type Logger struct {
	baseDir     string
	currentFile *os.File
	mu          sync.Mutex
	buffer      []Event
	flushTicker *time.Ticker
	maxFileSize int64
	retention   time.Duration

	// Metrics
	eventCount map[EventType]int64
	errorCount int64
	lastFlush  time.Time
}

// NewLogger creates a new audit logger
func NewLogger(baseDir string) (*Logger, error) {
	// Create audit directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create audit directory: %w", err)
	}

	logger := &Logger{
		baseDir:     baseDir,
		buffer:      make([]Event, 0, 100),
		flushTicker: time.NewTicker(30 * time.Second),
		maxFileSize: 100 * 1024 * 1024,   // 100MB
		retention:   90 * 24 * time.Hour, // 90 days
		eventCount:  make(map[EventType]int64),
		lastFlush:   time.Now(),
	}

	// Open initial log file
	if err := logger.rotateFile(); err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Start background processes
	go logger.flushLoop()
	go logger.cleanupLoop()

	// Log system start
	logger.LogEvent(context.Background(), EventTypeSystemStart, "Audit system started", "", "", nil)

	return logger, nil
}

// LogEvent logs an audit event
func (al *Logger) LogEvent(ctx context.Context, eventType EventType, action, resource, resourceID string, details map[string]interface{}) {
	event := Event{
		ID:         generateEventID(),
		Timestamp:  time.Now().UTC(),
		EventType:  eventType,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		Success:    true,
	}

	// Extract context values if available
	if sessionID, ok := ctx.Value(contextKeySessionID).(string); ok {
		event.SessionID = sessionID
	}
	if userID, ok := ctx.Value(contextKeyUserID).(string); ok {
		event.UserID = userID
	}
	if repo, ok := ctx.Value(contextKeyRepository).(string); ok {
		event.Repository = repo
	}

	al.addEvent(&event)
}

// LogError logs an error event
func (al *Logger) LogError(ctx context.Context, eventType EventType, action, resource string, err error, details map[string]interface{}) {
	event := Event{
		ID:        generateEventID(),
		Timestamp: time.Now().UTC(),
		EventType: eventType,
		Action:    action,
		Resource:  resource,
		Details:   details,
		Success:   false,
		Error:     err.Error(),
	}

	// Extract context values
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		event.SessionID = sessionID
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		event.UserID = userID
	}

	al.addEvent(&event)
	al.errorCount++
}

// LogEventWithDuration logs an event with timing information
func (al *Logger) LogEventWithDuration(ctx context.Context, eventType EventType, action, resource, resourceID string, duration time.Duration, details map[string]interface{}) {
	event := Event{
		ID:         generateEventID(),
		Timestamp:  time.Now().UTC(),
		EventType:  eventType,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
		Success:    true,
		Duration:   duration,
	}

	// Extract context values
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		event.SessionID = sessionID
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		event.UserID = userID
	}
	if repo, ok := ctx.Value("repository").(string); ok {
		event.Repository = repo
	}

	al.addEvent(&event)
}

// addEvent adds an event to the buffer
func (al *Logger) addEvent(event *Event) {
	al.mu.Lock()
	defer al.mu.Unlock()

	al.buffer = append(al.buffer, *event)
	al.eventCount[event.EventType]++

	// Flush if buffer is getting full
	if len(al.buffer) >= 100 {
		al.flush()
	}
}

// flush writes buffered events to disk
func (al *Logger) flush() {
	if len(al.buffer) == 0 {
		return
	}

	// Check if we need to rotate the file
	if al.currentFile != nil {
		if info, err := al.currentFile.Stat(); err == nil {
			if info.Size() > al.maxFileSize {
				_ = al.rotateFile()
			}
		}
	}

	// Write events to file
	encoder := json.NewEncoder(al.currentFile)
	for i := range al.buffer {
		if err := encoder.Encode(al.buffer[i]); err != nil {
			logging.Error("Failed to write audit event", "error", err, "event_id", al.buffer[i].ID)
		}
	}

	// Clear buffer
	al.buffer = al.buffer[:0]
	al.lastFlush = time.Now()
}

// flushLoop periodically flushes the buffer
func (al *Logger) flushLoop() {
	for range al.flushTicker.C {
		al.mu.Lock()
		al.flush()
		al.mu.Unlock()
	}
}

// rotateFile creates a new log file
func (al *Logger) rotateFile() error {
	// Close current file if open
	if al.currentFile != nil {
		_ = al.currentFile.Close()
	}

	// Generate new filename with timestamp
	filename := fmt.Sprintf("audit_%s.jsonl", time.Now().Format("20060102_150405"))
	fullPath := filepath.Join(al.baseDir, filename)

	// Open new file
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) // #nosec G304 -- Path is constructed from sanitized baseDir and timestamp
	if err != nil {
		return fmt.Errorf("failed to open audit file: %w", err)
	}

	al.currentFile = file

	// Create or update symlink to current file
	currentLink := filepath.Join(al.baseDir, "current.jsonl")
	_ = os.Remove(currentLink) // Remove old symlink if exists
	_ = os.Symlink(filename, currentLink)

	return nil
}

// cleanupLoop periodically removes old audit files
func (al *Logger) cleanupLoop() {
	// Run cleanup every hour
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		al.cleanup()
	}
}

// cleanup removes old audit files
func (al *Logger) cleanup() {
	cutoff := time.Now().Add(-al.retention)

	files, err := os.ReadDir(al.baseDir)
	if err != nil {
		logging.Error("Failed to read audit directory", "error", err)
		return
	}

	for _, file := range files {
		if file.IsDir() || !isAuditFile(file.Name()) {
			continue
		}

		fullPath := filepath.Join(al.baseDir, file.Name())
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(fullPath); err != nil {
				logging.Error("Failed to remove old audit file", "file", fullPath, "error", err)
			} else {
				logging.Info("Removed old audit file", "file", file.Name())
			}
		}
	}
}

// GetStatistics returns audit statistics
func (al *Logger) GetStatistics() map[string]interface{} {
	al.mu.Lock()
	defer al.mu.Unlock()

	stats := map[string]interface{}{
		"total_events":   sumEventCounts(al.eventCount),
		"error_count":    al.errorCount,
		"events_by_type": al.eventCount,
		"buffer_size":    len(al.buffer),
		"last_flush":     al.lastFlush,
	}

	return stats
}

// Search searches audit logs
func (al *Logger) Search(_ context.Context, criteria *SearchCriteria) ([]Event, error) {
	events := []Event{}

	// Get list of files to search
	files, err := al.getFilesToSearch(criteria.StartTime, criteria.EndTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit files: %w", err)
	}

	// Search each file
	for _, filename := range files {
		fileEvents, err := al.searchFile(filename, criteria)
		if err != nil {
			logging.Error("Failed to search audit file", "file", filename, "error", err)
			continue
		}
		events = append(events, fileEvents...)
	}

	// Apply limit
	if criteria.Limit > 0 && len(events) > criteria.Limit {
		events = events[:criteria.Limit]
	}

	return events, nil
}

// searchFile searches a single audit file
func (al *Logger) searchFile(filename string, criteria *SearchCriteria) ([]Event, error) {
	return al.searchFileWithCriteria(filename, criteria)
}

func (al *Logger) searchFileWithCriteria(filename string, criteria *SearchCriteria) ([]Event, error) {
	// Clean and validate the filename
	cleanPath := filepath.Clean(filepath.Join(al.baseDir, filename))
	if !strings.HasPrefix(cleanPath, filepath.Clean(al.baseDir)) {
		return nil, errors.New("invalid filename")
	}

	file, err := os.Open(cleanPath) // #nosec G304 -- Path is cleaned and validated
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	events := []Event{}
	decoder := json.NewDecoder(file)

	for decoder.More() {
		var event Event
		if err := decoder.Decode(&event); err != nil {
			continue
		}

		if criteria.Matches(&event) {
			events = append(events, event)
		}
	}

	return events, nil
}

// getFilesToSearch returns audit files within the time range
func (al *Logger) getFilesToSearch(_ /* start */, _ /* end */ time.Time) ([]string, error) {
	files, err := os.ReadDir(al.baseDir)
	if err != nil {
		return nil, err
	}

	filenames := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !isAuditFile(file.Name()) {
			continue
		}

		// TODO: Parse timestamp from filename to filter by date range
		filenames = append(filenames, file.Name())
	}

	return filenames, nil
}

// Stop gracefully stops the audit logger
func (al *Logger) Stop() {
	// Stop tickers
	al.flushTicker.Stop()

	// Final flush
	al.mu.Lock()
	defer al.mu.Unlock()

	// Log shutdown
	al.buffer = append(al.buffer, Event{
		ID:        generateEventID(),
		Timestamp: time.Now().UTC(),
		EventType: EventTypeSystemShutdown,
		Action:    "Audit system shutdown",
		Success:   true,
	})

	al.flush()

	// Close file
	if al.currentFile != nil {
		_ = al.currentFile.Close()
	}
}

// SearchCriteria defines search parameters for audit logs
type SearchCriteria struct {
	StartTime  time.Time
	EndTime    time.Time
	EventTypes []EventType
	SessionID  string
	UserID     string
	Repository string
	Resource   string
	Success    *bool
	Limit      int
}

// Matches checks if an event matches the criteria
func (sc *SearchCriteria) Matches(event *Event) bool {
	return sc.matchesTimeRange(event) &&
		sc.matchesEventTypes(event) &&
		sc.matchesStringFields(event) &&
		sc.matchesSuccessStatus(event)
}

// matchesTimeRange checks if the event falls within the specified time range
func (sc *SearchCriteria) matchesTimeRange(event *Event) bool {
	if !sc.StartTime.IsZero() && event.Timestamp.Before(sc.StartTime) {
		return false
	}
	if !sc.EndTime.IsZero() && event.Timestamp.After(sc.EndTime) {
		return false
	}
	return true
}

// matchesEventTypes checks if the event type is in the allowed list
func (sc *SearchCriteria) matchesEventTypes(event *Event) bool {
	if len(sc.EventTypes) == 0 {
		return true
	}

	for _, et := range sc.EventTypes {
		if event.EventType == et {
			return true
		}
	}
	return false
}

// matchesStringFields checks if all string-based criteria match
func (sc *SearchCriteria) matchesStringFields(event *Event) bool {
	return sc.matchesSessionID(event) &&
		sc.matchesUserID(event) &&
		sc.matchesRepository(event) &&
		sc.matchesResource(event)
}

// matchesSessionID checks if the session ID matches the criteria
func (sc *SearchCriteria) matchesSessionID(event *Event) bool {
	return sc.SessionID == "" || event.SessionID == sc.SessionID
}

// matchesUserID checks if the user ID matches the criteria
func (sc *SearchCriteria) matchesUserID(event *Event) bool {
	return sc.UserID == "" || event.UserID == sc.UserID
}

// matchesRepository checks if the repository matches the criteria
func (sc *SearchCriteria) matchesRepository(event *Event) bool {
	return sc.Repository == "" || event.Repository == sc.Repository
}

// matchesResource checks if the resource matches the criteria
func (sc *SearchCriteria) matchesResource(event *Event) bool {
	return sc.Resource == "" || event.Resource == sc.Resource
}

// matchesSuccessStatus checks if the success status matches the criteria
func (sc *SearchCriteria) matchesSuccessStatus(event *Event) bool {
	return sc.Success == nil || event.Success == *sc.Success
}

// Helper functions

func generateEventID() string {
	return fmt.Sprintf("evt_%d_%d", time.Now().UnixNano(), os.Getpid())
}

func isAuditFile(filename string) bool {
	return len(filename) > 6 && filename[:6] == "audit_" && filepath.Ext(filename) == ".jsonl"
}

func sumEventCounts(counts map[EventType]int64) int64 {
	var total int64
	for _, count := range counts {
		total += count
	}
	return total
}
