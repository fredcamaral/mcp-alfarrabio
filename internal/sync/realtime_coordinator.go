// Package sync provides real-time synchronization capabilities for the MCP Memory Server.
package sync

import (
	"context"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/logging"
	"lerian-mcp-memory/pkg/types"
)

// WebSocketBroadcaster interface for WebSocket broadcasting
type WebSocketBroadcaster interface {
	BroadcastMemoryEvent(eventType, action, chunkID, repository, sessionID string, data interface{}) error
}

// EventType represents the type of real-time sync event
type EventType string

const (
	EventTypeChunkCreated EventType = "chunk_created"
	EventTypeChunkUpdated EventType = "chunk_updated"
	EventTypeChunkDeleted EventType = "chunk_deleted"
	EventTypeTaskCreated  EventType = "task_created"
	EventTypeTaskUpdated  EventType = "task_updated"
	EventTypeTaskDeleted  EventType = "task_deleted"
)

// MemoryChangeEvent represents a memory change event for real-time sync
type MemoryChangeEvent struct {
	Type       EventType              `json:"type"`
	Action     string                 `json:"action"`
	ChunkID    string                 `json:"chunk_id,omitempty"`
	TaskID     string                 `json:"task_id,omitempty"`
	Repository string                 `json:"repository"`
	SessionID  string                 `json:"session_id,omitempty"`
	Content    string                 `json:"content,omitempty"`
	Summary    string                 `json:"summary,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// RealtimeSyncCoordinator coordinates real-time synchronization between server and clients
type RealtimeSyncCoordinator struct {
	wsHandler WebSocketBroadcaster
	logger    *logging.EnhancedLogger
	enabled   bool
}

// NewRealtimeSyncCoordinator creates a new real-time sync coordinator
func NewRealtimeSyncCoordinator(wsHandler WebSocketBroadcaster) *RealtimeSyncCoordinator {
	return &RealtimeSyncCoordinator{
		wsHandler: wsHandler,
		logger:    logging.NewEnhancedLogger("realtime_sync"),
		enabled:   wsHandler != nil,
	}
}

// BroadcastChunkEvent broadcasts a memory chunk change event to all connected clients
func (c *RealtimeSyncCoordinator) BroadcastChunkEvent(ctx context.Context, eventType EventType, chunk *types.ConversationChunk) error {
	if !c.enabled {
		c.logger.Debug("Real-time sync disabled, skipping chunk event broadcast")
		return nil
	}

	// Create event data
	event := MemoryChangeEvent{
		Type:       eventType,
		Action:     c.actionFromEventType(eventType),
		ChunkID:    chunk.ID,
		Repository: chunk.Metadata.Repository,
		SessionID:  chunk.SessionID,
		Content:    chunk.Content,
		Summary:    chunk.Summary,
		Tags:       chunk.Metadata.Tags,
		Timestamp:  time.Now(),
		Metadata: map[string]interface{}{
			"type":           string(chunk.Type),
			"outcome":        string(chunk.Metadata.Outcome),
			"difficulty":     string(chunk.Metadata.Difficulty),
			"files_modified": chunk.Metadata.FilesModified,
			"tools_used":     chunk.Metadata.ToolsUsed,
		},
	}

	// Broadcast via WebSocket
	err := c.wsHandler.BroadcastMemoryEvent(
		string(eventType),
		event.Action,
		chunk.ID,
		chunk.Metadata.Repository,
		chunk.SessionID,
		event,
	)

	if err != nil {
		c.logger.Error("Failed to broadcast chunk event",
			"error", err.Error(),
			"event_type", string(eventType),
			"chunk_id", chunk.ID,
			"repository", chunk.Metadata.Repository,
		)
		return fmt.Errorf("failed to broadcast chunk event: %w", err)
	}

	c.logger.Info("Broadcasted chunk event",
		"event_type", string(eventType),
		"chunk_id", chunk.ID,
		"repository", chunk.Metadata.Repository,
		"session_id", chunk.SessionID,
	)

	return nil
}

// BroadcastTaskEvent broadcasts a task change event to all connected clients
func (c *RealtimeSyncCoordinator) BroadcastTaskEvent(ctx context.Context, eventType EventType, taskID, repository, sessionID string, taskData map[string]interface{}) error {
	if !c.enabled {
		c.logger.Debug("Real-time sync disabled, skipping task event broadcast")
		return nil
	}

	// Create event data
	event := MemoryChangeEvent{
		Type:       eventType,
		Action:     c.actionFromEventType(eventType),
		TaskID:     taskID,
		Repository: repository,
		SessionID:  sessionID,
		Timestamp:  time.Now(),
		Metadata:   taskData,
	}

	// Broadcast via WebSocket
	err := c.wsHandler.BroadcastMemoryEvent(
		string(eventType),
		event.Action,
		taskID,
		repository,
		sessionID,
		event,
	)

	if err != nil {
		c.logger.Error("Failed to broadcast task event",
			"error", err.Error(),
			"event_type", string(eventType),
			"task_id", taskID,
			"repository", repository,
		)
		return fmt.Errorf("failed to broadcast task event: %w", err)
	}

	c.logger.Info("Broadcasted task event",
		"event_type", string(eventType),
		"task_id", taskID,
		"repository", repository,
		"session_id", sessionID,
	)

	return nil
}

// BroadcastCustomEvent broadcasts a custom memory event to all connected clients
func (c *RealtimeSyncCoordinator) BroadcastCustomEvent(ctx context.Context, eventType, action, entityID, repository, sessionID string, data interface{}) error {
	if !c.enabled {
		c.logger.Debug("Real-time sync disabled, skipping custom event broadcast")
		return nil
	}

	err := c.wsHandler.BroadcastMemoryEvent(
		eventType,
		action,
		entityID,
		repository,
		sessionID,
		data,
	)

	if err != nil {
		c.logger.Error("Failed to broadcast custom event",
			"error", err.Error(),
			"event_type", eventType,
			"action", action,
			"entity_id", entityID,
			"repository", repository,
		)
		return fmt.Errorf("failed to broadcast custom event: %w", err)
	}

	c.logger.Info("Broadcasted custom event",
		"event_type", eventType,
		"action", action,
		"entity_id", entityID,
		"repository", repository,
		"session_id", sessionID,
	)

	return nil
}

// actionFromEventType converts event type to action string
func (c *RealtimeSyncCoordinator) actionFromEventType(eventType EventType) string {
	switch eventType {
	case EventTypeChunkCreated, EventTypeTaskCreated:
		return "created"
	case EventTypeChunkUpdated, EventTypeTaskUpdated:
		return "updated"
	case EventTypeChunkDeleted, EventTypeTaskDeleted:
		return "deleted"
	default:
		return "changed"
	}
}

// IsEnabled returns whether real-time sync is enabled
func (c *RealtimeSyncCoordinator) IsEnabled() bool {
	return c.enabled
}

// Enable enables real-time sync
func (c *RealtimeSyncCoordinator) Enable() {
	c.enabled = c.wsHandler != nil
	if c.enabled {
		c.logger.Info("Real-time sync enabled")
	} else {
		c.logger.Warn("Cannot enable real-time sync: WebSocket handler not available")
	}
}

// Disable disables real-time sync
func (c *RealtimeSyncCoordinator) Disable() {
	c.enabled = false
	c.logger.Info("Real-time sync disabled")
}
