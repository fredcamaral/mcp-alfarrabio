// Package api provides batch operation models for server synchronization
package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// BatchSyncRequest represents a client request for batch synchronization
type BatchSyncRequest struct {
	LastSyncTime *time.Time     `json:"last_sync_time"`
	LocalTasks   []TaskSyncItem `json:"local_tasks"`
	Repository   string         `json:"repository"`
	ClientID     string         `json:"client_id"`
	SyncToken    string         `json:"sync_token,omitempty"`
}

// TaskSyncItem represents a task in sync operations with version tracking
type TaskSyncItem struct {
	ID           string                 `json:"id"`
	Content      string                 `json:"content"`
	Status       entities.Status        `json:"status"`
	Priority     entities.Priority      `json:"priority"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CreatedAt    time.Time              `json:"created_at"`
	LocalVersion int                    `json:"local_version"`
	Checksum     string                 `json:"checksum"`
	Repository   string                 `json:"repository"`
	Tags         []string               `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// BatchSyncResponse contains the server's response to a batch sync request
type BatchSyncResponse struct {
	ServerTasks []TaskSyncItem `json:"server_tasks"`
	Conflicts   []ConflictItem `json:"conflicts"`
	ToCreate    []string       `json:"to_create"`
	ToUpdate    []string       `json:"to_update"`
	ToDelete    []string       `json:"to_delete"`
	ServerTime  time.Time      `json:"server_time"`
	SyncToken   string         `json:"sync_token"`
	SyncStats   SyncStatistics `json:"sync_stats"`
}

// ConflictItem represents a detected conflict between local and server tasks
type ConflictItem struct {
	TaskID       string             `json:"task_id"`
	LocalTask    *TaskSyncItem      `json:"local_task"`
	ServerTask   *TaskSyncItem      `json:"server_task"`
	Resolution   ConflictResolution `json:"resolution"`
	Reason       string             `json:"reason"`
	ConflictType ConflictType       `json:"conflict_type"`
}

// ConflictResolution defines how a conflict should be resolved
type ConflictResolution struct {
	Strategy     ResolutionStrategy `json:"strategy"`
	ResolvedTask *TaskSyncItem      `json:"resolved_task"`
	Confidence   float64            `json:"confidence"`
	AutoApply    bool               `json:"auto_apply"`
}

// ConflictType defines the type of conflict detected
type ConflictType string

const (
	ConflictTypeContent    ConflictType = "content"
	ConflictTypeStatus     ConflictType = "status"
	ConflictTypePriority   ConflictType = "priority"
	ConflictTypeTimestamp  ConflictType = "timestamp"
	ConflictTypeMetadata   ConflictType = "metadata"
	ConflictTypeStructural ConflictType = "structural"
)

// ResolutionStrategy defines how conflicts should be resolved
type ResolutionStrategy string

const (
	StrategyServerWins      ResolutionStrategy = "server_wins"
	StrategyLocalWins       ResolutionStrategy = "local_wins"
	StrategyMerge           ResolutionStrategy = "merge"
	StrategyServerWinsNewer ResolutionStrategy = "server_wins_newer"
	StrategyLocalWinsNewer  ResolutionStrategy = "local_wins_newer"
	StrategyQdrantTruth     ResolutionStrategy = "qdrant_truth"
	StrategyManual          ResolutionStrategy = "manual"
)

// SyncStatistics provides analytics about the sync operation
type SyncStatistics struct {
	TotalTasks        int           `json:"total_tasks"`
	ConflictsDetected int           `json:"conflicts_detected"`
	ConflictsResolved int           `json:"conflicts_resolved"`
	TasksCreated      int           `json:"tasks_created"`
	TasksUpdated      int           `json:"tasks_updated"`
	TasksDeleted      int           `json:"tasks_deleted"`
	SyncDuration      time.Duration `json:"sync_duration"`
	DataTransferred   int64         `json:"data_transferred"`
	CompressionRatio  float64       `json:"compression_ratio"`
}

// SyncState tracks the synchronization state for a repository
type SyncState struct {
	Repository        string    `json:"repository"`
	LastSyncTime      time.Time `json:"last_sync_time"`
	SyncToken         string    `json:"sync_token"`
	PendingChanges    int       `json:"pending_changes"`
	ClientID          string    `json:"client_id"`
	TotalSyncs        int       `json:"total_syncs"`
	LastConflictCount int       `json:"last_conflict_count"`
	SyncVersion       int       `json:"sync_version"`
}

// DeltaSyncRequest represents a request for delta-only synchronization
type DeltaSyncRequest struct {
	Repository   string    `json:"repository"`
	SyncToken    string    `json:"sync_token"`
	LastSyncTime time.Time `json:"last_sync_time"`
	ClientID     string    `json:"client_id"`
}

// DeltaSyncResponse contains only changes since last sync
type DeltaSyncResponse struct {
	ChangedTasks   []TaskSyncItem `json:"changed_tasks"`
	DeletedTaskIDs []string       `json:"deleted_task_ids"`
	NewSyncToken   string         `json:"new_sync_token"`
	ServerTime     time.Time      `json:"server_time"`
	HasMoreChanges bool           `json:"has_more_changes"`
}

// BatchOperation represents a single operation in a batch
type BatchOperation struct {
	Type      OperationType `json:"type"`
	TaskID    string        `json:"task_id"`
	Task      *TaskSyncItem `json:"task,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// OperationType defines the type of batch operation
type OperationType string

const (
	OperationCreate OperationType = "create"
	OperationUpdate OperationType = "update"
	OperationDelete OperationType = "delete"
	OperationMove   OperationType = "move"
)

// Utility functions

// GenerateChecksum creates a checksum for a task to detect changes
func (t *TaskSyncItem) GenerateChecksum() string {
	content := fmt.Sprintf("%s|%s|%s|%s|%v",
		t.Content, t.Status, t.Priority, t.Repository, t.Tags)
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// UpdateChecksum recalculates and updates the task's checksum
func (t *TaskSyncItem) UpdateChecksum() {
	t.Checksum = t.GenerateChecksum()
}

// IsNewer returns true if this task is newer than the other
func (t *TaskSyncItem) IsNewer(other *TaskSyncItem) bool {
	return t.UpdatedAt.After(other.UpdatedAt)
}

// HasConflictWith detects if there's a conflict with another task
func (t *TaskSyncItem) HasConflictWith(other *TaskSyncItem) bool {
	if t.ID != other.ID {
		return false
	}

	// If timestamps are equal but checksums differ, it's a conflict
	if t.UpdatedAt.Equal(other.UpdatedAt) && t.Checksum != other.Checksum {
		return true
	}

	// If one is newer but has different content, potential conflict
	timeDiff := t.UpdatedAt.Sub(other.UpdatedAt)
	if timeDiff.Abs() < time.Minute && t.Checksum != other.Checksum {
		return true
	}

	return false
}

// ToTask converts a TaskSyncItem to a domain Task entity
func (t *TaskSyncItem) ToTask() *entities.Task {
	return &entities.Task{
		ID:         t.ID,
		Content:    t.Content,
		Status:     t.Status,
		Priority:   t.Priority,
		Repository: t.Repository,
		Tags:       t.Tags,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

// FromTask creates a TaskSyncItem from a domain Task entity
func FromTask(task *entities.Task) TaskSyncItem {
	item := TaskSyncItem{
		ID:         task.ID,
		Content:    task.Content,
		Status:     task.Status,
		Priority:   task.Priority,
		Repository: task.Repository,
		Tags:       task.Tags,
		CreatedAt:  task.CreatedAt,
		UpdatedAt:  task.UpdatedAt,
		Metadata:   make(map[string]interface{}), // Initialize empty metadata
	}
	item.UpdateChecksum()
	return item
}

// ValidateSync validates a sync request for basic correctness
func (r *BatchSyncRequest) ValidateSync() error {
	if r.Repository == "" {
		return errors.New("repository is required")
	}

	if r.ClientID == "" {
		return errors.New("client_id is required")
	}

	// Validate task items
	for i, task := range r.LocalTasks {
		if task.ID == "" {
			return fmt.Errorf("task %d: id is required", i)
		}
		if task.Repository != r.Repository {
			return fmt.Errorf("task %d: repository mismatch", i)
		}
		if task.Checksum == "" {
			return fmt.Errorf("task %d: checksum is required", i)
		}
	}

	return nil
}

// GetConflictTypes returns all conflict types detected for this item
func (c *ConflictItem) GetConflictTypes() []ConflictType {
	var types []ConflictType

	if c.LocalTask == nil || c.ServerTask == nil {
		return types
	}

	local := c.LocalTask
	server := c.ServerTask

	if local.Content != server.Content {
		types = append(types, ConflictTypeContent)
	}

	if local.Status != server.Status {
		types = append(types, ConflictTypeStatus)
	}

	if local.Priority != server.Priority {
		types = append(types, ConflictTypePriority)
	}

	if !local.UpdatedAt.Equal(server.UpdatedAt) {
		types = append(types, ConflictTypeTimestamp)
	}

	// Check metadata differences
	if len(local.Metadata) != len(server.Metadata) {
		types = append(types, ConflictTypeMetadata)
	} else {
		for key, localVal := range local.Metadata {
			if serverVal, exists := server.Metadata[key]; !exists || localVal != serverVal {
				types = append(types, ConflictTypeMetadata)
				break
			}
		}
	}

	return types
}

// CalculateConfidence calculates confidence level for a resolution
func (r *ConflictResolution) CalculateConfidence(conflict *ConflictItem) {
	baseConfidence := 0.5

	switch r.Strategy {
	case StrategyQdrantTruth:
		baseConfidence = 0.95 // High confidence when using authoritative source
	case StrategyServerWinsNewer, StrategyLocalWinsNewer:
		baseConfidence = 0.85 // High confidence for timestamp-based resolution
	case StrategyMerge:
		baseConfidence = 0.70 // Medium confidence for merge
	case StrategyServerWins, StrategyLocalWins:
		baseConfidence = 0.60 // Lower confidence for arbitrary choice
	case StrategyManual:
		baseConfidence = 0.0 // No automatic confidence for manual resolution
	}

	// Adjust based on conflict complexity
	conflictTypes := conflict.GetConflictTypes()
	complexityPenalty := float64(len(conflictTypes)) * 0.05

	r.Confidence = baseConfidence - complexityPenalty
	if r.Confidence < 0 {
		r.Confidence = 0
	}
	if r.Confidence > 1 {
		r.Confidence = 1
	}

	// Auto-apply if confidence is high enough
	r.AutoApply = r.Confidence >= 0.8
}
