// Package sync provides real-time synchronization handlers for the CLI.
package sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"lerian-mcp-memory-cli/internal/adapters/secondary/api"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// RealtimeHandler handles incoming real-time sync events from the server
type RealtimeHandler struct {
	taskRepo         ports.TaskRepository
	wsClient         *api.WebSocketClient
	notificationHub  api.NotificationHub
	logger           *slog.Logger
	conflictResolver ConflictResolver
	enabled          bool
}

// ConflictResolver handles conflicts between local and remote changes
type ConflictResolver interface {
	ResolveConflict(ctx context.Context, localTask, remoteTask *entities.Task) (*entities.Task, error)
	DetectConflict(localTask, remoteTask *entities.Task) bool
}

// MemoryChangeEvent represents an incoming memory change event
type MemoryChangeEvent struct {
	Type       string                 `json:"type"`
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

// NewRealtimeHandler creates a new real-time sync handler
func NewRealtimeHandler(
	taskRepo ports.TaskRepository,
	wsClient *api.WebSocketClient,
	notificationHub api.NotificationHub,
	logger *slog.Logger,
) *RealtimeHandler {
	return &RealtimeHandler{
		taskRepo:        taskRepo,
		wsClient:        wsClient,
		notificationHub: notificationHub,
		logger:          logger,
		enabled:         true,
	}
}

// HandleIncomingEvent processes incoming real-time events from the server
func (h *RealtimeHandler) HandleIncomingEvent(ctx context.Context, eventData []byte) error {
	if !h.enabled {
		h.logger.Debug("Real-time sync disabled, ignoring incoming event")
		return nil
	}

	var event MemoryChangeEvent
	if err := json.Unmarshal(eventData, &event); err != nil {
		return fmt.Errorf("failed to unmarshal memory event: %w", err)
	}

	h.logger.Info("Received real-time event",
		"type", event.Type,
		"action", event.Action,
		"chunk_id", event.ChunkID,
		"repository", event.Repository,
	)

	switch event.Type {
	case "chunk_created":
		return h.handleChunkCreated(ctx, &event)
	case "chunk_updated":
		return h.handleChunkUpdated(ctx, &event)
	case "chunk_deleted":
		return h.handleChunkDeleted(ctx, &event)
	case "task_created":
		return h.handleTaskCreated(ctx, &event)
	case "task_updated":
		return h.handleTaskUpdated(ctx, &event)
	case "task_deleted":
		return h.handleTaskDeleted(ctx, &event)
	default:
		h.logger.Debug("Unknown event type", "type", event.Type)
		return nil
	}
}

// handleChunkCreated processes chunk creation events
func (h *RealtimeHandler) handleChunkCreated(ctx context.Context, event *MemoryChangeEvent) error {
	h.logger.Info("Processing chunk created event",
		"chunk_id", event.ChunkID,
		"repository", event.Repository,
		"summary", event.Summary,
	)

	// For chunk events, we might want to trigger local task suggestions or updates
	// This could involve:
	// 1. Checking if the chunk is relevant to current local tasks
	// 2. Updating local context or suggestions
	// 3. Notifying the user about new relevant information

	// Publish system event to notify UI about new memory chunk
	if h.notificationHub != nil {
		systemEvent := &api.SystemEvent{
			Type:      "memory.chunk_created",
			Message:   "New memory chunk: " + event.Summary,
			Timestamp: event.Timestamp,
			Data: map[string]interface{}{
				"chunk_id":   event.ChunkID,
				"repository": event.Repository,
				"summary":    event.Summary,
				"tags":       event.Tags,
			},
		}
		h.notificationHub.PublishSystemEvent(systemEvent)
	}

	return nil
}

// handleChunkUpdated processes chunk update events
func (h *RealtimeHandler) handleChunkUpdated(ctx context.Context, event *MemoryChangeEvent) error {
	h.logger.Info("Processing chunk updated event",
		"chunk_id", event.ChunkID,
		"repository", event.Repository,
	)

	// Similar to creation, but for updates
	if h.notificationHub != nil {
		systemEvent := &api.SystemEvent{
			Type:      "memory.chunk_updated",
			Message:   "Memory chunk updated: " + event.Summary,
			Timestamp: event.Timestamp,
			Data: map[string]interface{}{
				"chunk_id":   event.ChunkID,
				"repository": event.Repository,
				"summary":    event.Summary,
			},
		}
		h.notificationHub.PublishSystemEvent(systemEvent)
	}

	return nil
}

// handleChunkDeleted processes chunk deletion events
func (h *RealtimeHandler) handleChunkDeleted(ctx context.Context, event *MemoryChangeEvent) error {
	h.logger.Info("Processing chunk deleted event",
		"chunk_id", event.ChunkID,
		"repository", event.Repository,
	)

	if h.notificationHub != nil {
		systemEvent := &api.SystemEvent{
			Type:      "memory.chunk_deleted",
			Message:   "Memory chunk deleted: " + event.ChunkID,
			Timestamp: event.Timestamp,
			Data: map[string]interface{}{
				"chunk_id":   event.ChunkID,
				"repository": event.Repository,
			},
		}
		h.notificationHub.PublishSystemEvent(systemEvent)
	}

	return nil
}

// handleTaskCreated processes task creation events from other clients
func (h *RealtimeHandler) handleTaskCreated(ctx context.Context, event *MemoryChangeEvent) error {
	if h.taskRepo == nil {
		return nil // No task repository available
	}

	h.logger.Info("Processing remote task created event",
		"task_id", event.TaskID,
		"repository", event.Repository,
	)

	// Convert remote task data to local task entity
	remoteTask, err := h.convertEventToTask(event)
	if err != nil {
		return fmt.Errorf("failed to convert event to task: %w", err)
	}

	// Check if task already exists locally (could be a late delivery)
	existingTask, err := h.taskRepo.GetByID(ctx, remoteTask.ID)
	if err == nil && existingTask != nil {
		h.logger.Debug("Task already exists locally, skipping creation",
			"task_id", remoteTask.ID)
		return nil
	}

	// Create the task in local repository
	if err := h.taskRepo.Create(ctx, remoteTask); err != nil {
		return fmt.Errorf("failed to create remote task locally: %w", err)
	}

	// Notify about the new task
	if h.notificationHub != nil {
		taskEvent := &api.TaskEvent{
			Type:       api.EventTypeTaskCreated,
			TaskID:     remoteTask.ID,
			Repository: remoteTask.Repository,
			ChangedBy:  "remote_sync",
			Timestamp:  event.Timestamp,
			Task: &api.TaskData{
				ID:            remoteTask.ID,
				Content:       remoteTask.Content,
				Status:        string(remoteTask.Status),
				Priority:      string(remoteTask.Priority),
				EstimatedMins: remoteTask.EstimatedMins,
				Repository:    remoteTask.Repository,
				CreatedAt:     remoteTask.CreatedAt,
				UpdatedAt:     remoteTask.UpdatedAt,
			},
		}
		h.notificationHub.PublishTaskEvent(taskEvent)
	}

	h.logger.Info("Successfully created remote task locally",
		"task_id", remoteTask.ID,
		"content", remoteTask.Content,
	)

	return nil
}

// handleTaskUpdated processes task update events from other clients
func (h *RealtimeHandler) handleTaskUpdated(ctx context.Context, event *MemoryChangeEvent) error {
	if h.taskRepo == nil {
		return nil
	}

	h.logger.Info("Processing remote task updated event",
		"task_id", event.TaskID,
		"repository", event.Repository,
	)

	// Convert remote task data to local task entity
	remoteTask, err := h.convertEventToTask(event)
	if err != nil {
		return fmt.Errorf("failed to convert event to task: %w", err)
	}

	// Get the current local version
	localTask, err := h.taskRepo.GetByID(ctx, remoteTask.ID)
	if err != nil {
		// Task doesn't exist locally, treat as creation
		return h.handleTaskCreated(ctx, event)
	}

	// Check for conflicts
	if h.conflictResolver != nil && h.conflictResolver.DetectConflict(localTask, remoteTask) {
		h.logger.Warn("Conflict detected between local and remote task",
			"task_id", remoteTask.ID,
			"local_updated", localTask.UpdatedAt,
			"remote_updated", remoteTask.UpdatedAt,
		)

		// Resolve the conflict
		resolvedTask, err := h.conflictResolver.ResolveConflict(ctx, localTask, remoteTask)
		if err != nil {
			return fmt.Errorf("failed to resolve task conflict: %w", err)
		}
		remoteTask = resolvedTask
	}

	// Update the local task
	if err := h.taskRepo.Update(ctx, remoteTask); err != nil {
		return fmt.Errorf("failed to update remote task locally: %w", err)
	}

	// Notify about the update
	if h.notificationHub != nil {
		taskEvent := &api.TaskEvent{
			Type:       api.EventTypeTaskUpdated,
			TaskID:     remoteTask.ID,
			Repository: remoteTask.Repository,
			ChangedBy:  "remote_sync",
			Timestamp:  event.Timestamp,
			Task: &api.TaskData{
				ID:            remoteTask.ID,
				Content:       remoteTask.Content,
				Status:        string(remoteTask.Status),
				Priority:      string(remoteTask.Priority),
				EstimatedMins: remoteTask.EstimatedMins,
				Repository:    remoteTask.Repository,
				CreatedAt:     remoteTask.CreatedAt,
				UpdatedAt:     remoteTask.UpdatedAt,
			},
		}
		h.notificationHub.PublishTaskEvent(taskEvent)
	}

	h.logger.Info("Successfully updated remote task locally",
		"task_id", remoteTask.ID,
		"content", remoteTask.Content,
	)

	return nil
}

// handleTaskDeleted processes task deletion events from other clients
func (h *RealtimeHandler) handleTaskDeleted(ctx context.Context, event *MemoryChangeEvent) error {
	if h.taskRepo == nil {
		return nil
	}

	h.logger.Info("Processing remote task deleted event",
		"task_id", event.TaskID,
		"repository", event.Repository,
	)

	// Check if task exists locally
	localTask, err := h.taskRepo.GetByID(ctx, event.TaskID)
	if err != nil {
		h.logger.Debug("Task not found locally, skipping deletion",
			"task_id", event.TaskID)
		return nil
	}

	// Delete the local task
	if err := h.taskRepo.Delete(ctx, event.TaskID); err != nil {
		return fmt.Errorf("failed to delete remote task locally: %w", err)
	}

	// Notify about the deletion
	if h.notificationHub != nil {
		taskEvent := &api.TaskEvent{
			Type:       api.EventTypeTaskDeleted,
			TaskID:     event.TaskID,
			Repository: event.Repository,
			ChangedBy:  "remote_sync",
			Timestamp:  event.Timestamp,
			Task: &api.TaskData{
				ID:            localTask.ID,
				Content:       localTask.Content,
				Status:        string(localTask.Status),
				Priority:      string(localTask.Priority),
				EstimatedMins: localTask.EstimatedMins,
				Repository:    localTask.Repository,
				CreatedAt:     localTask.CreatedAt,
				UpdatedAt:     localTask.UpdatedAt,
			},
		}
		h.notificationHub.PublishTaskEvent(taskEvent)
	}

	h.logger.Info("Successfully deleted remote task locally",
		"task_id", event.TaskID,
	)

	return nil
}

// convertEventToTask converts a memory change event to a task entity
func (h *RealtimeHandler) convertEventToTask(event *MemoryChangeEvent) (*entities.Task, error) {
	// Extract task data from the event metadata
	metadata := event.Metadata
	if metadata == nil {
		return nil, errors.New("no metadata in task event")
	}

	task := &entities.Task{
		ID:         event.TaskID,
		Repository: event.Repository,
		UpdatedAt:  event.Timestamp,
		CreatedAt:  event.Timestamp,
	}

	// Extract fields from metadata
	if content, ok := metadata["content"].(string); ok {
		task.Content = content
	}

	if taskType, ok := metadata["type"].(string); ok {
		task.Type = taskType
	}

	if status, ok := metadata["status"].(string); ok {
		task.Status = entities.Status(status)
	}

	if priority, ok := metadata["priority"].(string); ok {
		task.Priority = entities.Priority(priority)
	}

	if estimatedMins, ok := metadata["estimated_mins"].(float64); ok {
		task.EstimatedMins = int(estimatedMins)
	}

	if tags, ok := metadata["tags"].([]interface{}); ok {
		task.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				task.Tags[i] = tagStr
			}
		}
	}

	// Store additional metadata if available
	task.Metadata = make(map[string]interface{})
	for k, v := range metadata {
		if k != "content" && k != "type" && k != "status" && k != "priority" && k != "estimated_mins" && k != "tags" {
			task.Metadata[k] = v
		}
	}

	return task, nil
}

// Enable enables real-time sync handling
func (h *RealtimeHandler) Enable() {
	h.enabled = true
	h.logger.Info("Real-time sync handler enabled")
}

// Disable disables real-time sync handling
func (h *RealtimeHandler) Disable() {
	h.enabled = false
	h.logger.Info("Real-time sync handler disabled")
}

// IsEnabled returns whether real-time sync is enabled
func (h *RealtimeHandler) IsEnabled() bool {
	return h.enabled
}

// SetConflictResolver sets the conflict resolver
func (h *RealtimeHandler) SetConflictResolver(resolver ConflictResolver) {
	h.conflictResolver = resolver
}
