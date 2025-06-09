// Package tasks provides audit logging functionality for task operations.
package tasks

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// AuditLogger handles audit logging for task operations
type AuditLogger struct {
	config AuditConfig
	buffer []AuditEntry
	mutex  sync.RWMutex
}

// AuditConfig represents audit logging configuration
type AuditConfig struct {
	Enabled         bool          `json:"enabled"`
	BufferSize      int           `json:"buffer_size"`
	FlushInterval   time.Duration `json:"flush_interval"`
	LogLevel        AuditLevel    `json:"log_level"`
	IncludeMetadata bool          `json:"include_metadata"`
	RetentionDays   int           `json:"retention_days"`
}

// AuditLevel represents the level of audit logging
type AuditLevel string

const (
	AuditLevelAll      AuditLevel = "all"
	AuditLevelChanges  AuditLevel = "changes"
	AuditLevelCritical AuditLevel = "critical"
)

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    string                 `json:"user_id"`
	Action    AuditAction            `json:"action"`
	Resource  AuditResource          `json:"resource"`
	Details   map[string]interface{} `json:"details,omitempty"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
}

// AuditAction represents the type of action performed
type AuditAction string

const (
	AuditActionCreate     AuditAction = "create"
	AuditActionRead       AuditAction = "read"
	AuditActionUpdate     AuditAction = "update"
	AuditActionDelete     AuditAction = "delete"
	AuditActionBatch      AuditAction = "batch"
	AuditActionSearch     AuditAction = "search"
	AuditActionTransition AuditAction = "transition"
	AuditActionAssign     AuditAction = "assign"
	AuditActionComment    AuditAction = "comment"
)

// AuditResource represents the type of resource being audited
type AuditResource struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// AuditSummary represents audit statistics
type AuditSummary struct {
	TotalEntries    int                 `json:"total_entries"`
	EntriesByAction map[AuditAction]int `json:"entries_by_action"`
	EntriesByUser   map[string]int      `json:"entries_by_user"`
	SuccessRate     float64             `json:"success_rate"`
	TimeRange       AuditTimeRange      `json:"time_range"`
	TopUsers        []UserAuditStats    `json:"top_users"`
	RecentErrors    []AuditEntry        `json:"recent_errors"`
}

// AuditTimeRange represents a time range for audit queries
type AuditTimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// UserAuditStats represents audit statistics for a user
type UserAuditStats struct {
	UserID      string    `json:"user_id"`
	ActionCount int       `json:"action_count"`
	SuccessRate float64   `json:"success_rate"`
	LastAction  time.Time `json:"last_action"`
}

// DefaultAuditConfig returns default audit configuration
func DefaultAuditConfig() AuditConfig {
	return AuditConfig{
		Enabled:         true,
		BufferSize:      1000,
		FlushInterval:   5 * time.Minute,
		LogLevel:        AuditLevelChanges,
		IncludeMetadata: true,
		RetentionDays:   90,
	}
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{
		config: DefaultAuditConfig(),
		buffer: make([]AuditEntry, 0),
	}
}

// NewAuditLoggerWithConfig creates an audit logger with custom config
func NewAuditLoggerWithConfig(config AuditConfig) *AuditLogger {
	return &AuditLogger{
		config: config,
		buffer: make([]AuditEntry, 0),
	}
}

// LogTaskCreated logs task creation
func (al *AuditLogger) LogTaskCreated(taskID, userID string, timestamp time.Time) {
	if !al.config.Enabled {
		return
	}

	entry := AuditEntry{
		ID:        al.generateEntryID(),
		Timestamp: timestamp,
		UserID:    userID,
		Action:    AuditActionCreate,
		Resource: AuditResource{
			Type: "task",
			ID:   taskID,
		},
		Success: true,
		Details: map[string]interface{}{
			"operation": "task_creation",
		},
	}

	al.addEntry(&entry)
}

// LogTaskUpdated logs task updates
func (al *AuditLogger) LogTaskUpdated(taskID, userID string, changes map[string]interface{}) {
	if !al.config.Enabled {
		return
	}

	entry := AuditEntry{
		ID:        al.generateEntryID(),
		Timestamp: time.Now(),
		UserID:    userID,
		Action:    AuditActionUpdate,
		Resource: AuditResource{
			Type: "task",
			ID:   taskID,
		},
		Success: true,
		Details: map[string]interface{}{
			"operation": "task_update",
			"changes":   changes,
		},
	}

	al.addEntry(&entry)
}

// LogTaskDeleted logs task deletion
func (al *AuditLogger) LogTaskDeleted(taskID, userID string, timestamp time.Time) {
	if !al.config.Enabled {
		return
	}

	entry := AuditEntry{
		ID:        al.generateEntryID(),
		Timestamp: timestamp,
		UserID:    userID,
		Action:    AuditActionDelete,
		Resource: AuditResource{
			Type: "task",
			ID:   taskID,
		},
		Success: true,
		Details: map[string]interface{}{
			"operation": "task_deletion",
		},
	}

	al.addEntry(&entry)
}

// LogTaskSearch logs search operations
func (al *AuditLogger) LogTaskSearch(userID, query string, resultCount int) {
	if !al.config.Enabled || al.config.LogLevel == AuditLevelCritical {
		return
	}

	entry := AuditEntry{
		ID:        al.generateEntryID(),
		Timestamp: time.Now(),
		UserID:    userID,
		Action:    AuditActionSearch,
		Resource: AuditResource{
			Type: "task",
		},
		Success: true,
		Details: map[string]interface{}{
			"operation":    "task_search",
			"query":        query,
			"result_count": resultCount,
		},
	}

	al.addEntry(&entry)
}

// LogBatchUpdate logs batch operations
func (al *AuditLogger) LogBatchUpdate(userID string, updateCount int, timestamp time.Time) {
	if !al.config.Enabled {
		return
	}

	entry := AuditEntry{
		ID:        al.generateEntryID(),
		Timestamp: timestamp,
		UserID:    userID,
		Action:    AuditActionBatch,
		Resource: AuditResource{
			Type: "task",
		},
		Success: true,
		Details: map[string]interface{}{
			"operation":    "batch_update",
			"update_count": updateCount,
		},
	}

	al.addEntry(&entry)
}

// LogStatusTransition logs task status transitions
func (al *AuditLogger) LogStatusTransition(taskID, userID, fromStatus, toStatus string, success bool, reason string) {
	if !al.config.Enabled {
		return
	}

	entry := AuditEntry{
		ID:        al.generateEntryID(),
		Timestamp: time.Now(),
		UserID:    userID,
		Action:    AuditActionTransition,
		Resource: AuditResource{
			Type: "task",
			ID:   taskID,
		},
		Success: success,
		Details: map[string]interface{}{
			"operation":   "status_transition",
			"from_status": fromStatus,
			"to_status":   toStatus,
		},
	}

	if !success {
		entry.Error = reason
	}

	al.addEntry(&entry)
}

// LogTaskAssignment logs task assignment changes
func (al *AuditLogger) LogTaskAssignment(taskID, userID, fromAssignee, toAssignee string) {
	if !al.config.Enabled {
		return
	}

	entry := AuditEntry{
		ID:        al.generateEntryID(),
		Timestamp: time.Now(),
		UserID:    userID,
		Action:    AuditActionAssign,
		Resource: AuditResource{
			Type: "task",
			ID:   taskID,
		},
		Success: true,
		Details: map[string]interface{}{
			"operation":     "task_assignment",
			"from_assignee": fromAssignee,
			"to_assignee":   toAssignee,
		},
	}

	al.addEntry(&entry)
}

// LogError logs error events
func (al *AuditLogger) LogError(userID, operation, errorMsg string, resource AuditResource) {
	if !al.config.Enabled {
		return
	}

	entry := AuditEntry{
		ID:        al.generateEntryID(),
		Timestamp: time.Now(),
		UserID:    userID,
		Action:    AuditAction(operation),
		Resource:  resource,
		Success:   false,
		Error:     errorMsg,
		Details: map[string]interface{}{
			"operation": operation,
		},
	}

	al.addEntry(&entry)
}

// GetAuditHistory returns audit history with filtering
func (al *AuditLogger) GetAuditHistory(userID string, timeRange AuditTimeRange, actions []AuditAction, limit int) []AuditEntry {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	results := make([]AuditEntry, 0, limit)

	for i := range al.buffer {
		entry := &al.buffer[i]
		// Filter by user (if specified)
		if userID != "" && entry.UserID != userID {
			continue
		}

		// Filter by time range
		if !timeRange.Start.IsZero() && entry.Timestamp.Before(timeRange.Start) {
			continue
		}
		if !timeRange.End.IsZero() && entry.Timestamp.After(timeRange.End) {
			continue
		}

		// Filter by actions
		if len(actions) > 0 {
			found := false
			for _, action := range actions {
				if entry.Action == action {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		results = append(results, *entry)

		// Apply limit
		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results
}

// GetAuditSummary returns audit statistics
func (al *AuditLogger) GetAuditSummary(timeRange AuditTimeRange) AuditSummary {
	al.mutex.RLock()
	defer al.mutex.RUnlock()

	summary := AuditSummary{
		EntriesByAction: make(map[AuditAction]int),
		EntriesByUser:   make(map[string]int),
		TimeRange:       timeRange,
		RecentErrors:    make([]AuditEntry, 0),
	}

	successCount := 0
	userStats := make(map[string]*UserAuditStats)

	for i := range al.buffer {
		entry := &al.buffer[i]
		// Filter by time range
		if !timeRange.Start.IsZero() && entry.Timestamp.Before(timeRange.Start) {
			continue
		}
		if !timeRange.End.IsZero() && entry.Timestamp.After(timeRange.End) {
			continue
		}

		summary.TotalEntries++

		// Count by action
		summary.EntriesByAction[entry.Action]++

		// Count by user
		summary.EntriesByUser[entry.UserID]++

		// Track success rate
		if entry.Success {
			successCount++
		} else if len(summary.RecentErrors) < 10 {
			// Add to recent errors (limit to 10)
			summary.RecentErrors = append(summary.RecentErrors, *entry)
		}

		// Track user stats
		if _, exists := userStats[entry.UserID]; !exists {
			userStats[entry.UserID] = &UserAuditStats{
				UserID:     entry.UserID,
				LastAction: entry.Timestamp,
			}
		}
		userStats[entry.UserID].ActionCount++
		if entry.Success {
			userStats[entry.UserID].SuccessRate++
		}
		if entry.Timestamp.After(userStats[entry.UserID].LastAction) {
			userStats[entry.UserID].LastAction = entry.Timestamp
		}
	}

	// Calculate success rate
	if summary.TotalEntries > 0 {
		summary.SuccessRate = float64(successCount) / float64(summary.TotalEntries)
	}

	// Convert user stats and calculate success rates
	for _, stats := range userStats {
		if stats.ActionCount > 0 {
			stats.SuccessRate /= float64(stats.ActionCount)
		}
		summary.TopUsers = append(summary.TopUsers, *stats)
	}

	return summary
}

// FlushBuffer flushes the audit buffer to persistent storage
func (al *AuditLogger) FlushBuffer() error {
	al.mutex.Lock()
	defer al.mutex.Unlock()

	if len(al.buffer) == 0 {
		return nil
	}

	// In a real implementation, this would write to persistent storage
	// For now, we'll just log to console
	for i := range al.buffer {
		entry := &al.buffer[i]
		if jsonBytes, err := json.Marshal(*entry); err == nil {
			log.Printf("[AUDIT] %s", string(jsonBytes))
		}
	}

	// Clear buffer
	al.buffer = al.buffer[:0]

	return nil
}

// StartPeriodicFlush starts periodic buffer flushing
func (al *AuditLogger) StartPeriodicFlush() {
	if !al.config.Enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(al.config.FlushInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := al.FlushBuffer(); err != nil {
				log.Printf("Error flushing audit buffer: %v", err)
			}
		}
	}()
}

// GetConfig returns the current audit configuration
func (al *AuditLogger) GetConfig() AuditConfig {
	return al.config
}

// UpdateConfig updates audit configuration
func (al *AuditLogger) UpdateConfig(config AuditConfig) {
	al.mutex.Lock()
	defer al.mutex.Unlock()
	al.config = config
}

// Private methods

func (al *AuditLogger) addEntry(entry *AuditEntry) {
	al.mutex.Lock()
	defer al.mutex.Unlock()

	// Check if we should log based on level
	switch al.config.LogLevel {
	case AuditLevelCritical:
		if entry.Action != AuditActionCreate && entry.Action != AuditActionDelete && !entry.Success {
			return
		}
	case AuditLevelChanges:
		if entry.Action == AuditActionRead || entry.Action == AuditActionSearch {
			return
		}
	case AuditLevelAll:
		// Log everything
	}

	al.buffer = append(al.buffer, *entry)

	// Check buffer size and flush if necessary
	if len(al.buffer) >= al.config.BufferSize {
		go func() {
			if err := al.FlushBuffer(); err != nil {
				log.Printf("Error flushing audit buffer: %v", err)
			}
		}()
	}
}

func (al *AuditLogger) generateEntryID() string {
	return fmt.Sprintf("audit_%d", time.Now().UnixNano())
}
