package integration

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Test HTTP client types for integration testing

// HTTPClient represents a simple HTTP client for API testing
type HTTPClient struct {
	baseURL string
	version string
	logger  *slog.Logger
}

// NewHTTPClient creates a new HTTP client for testing
func NewHTTPClient(baseURL, version string, logger *slog.Logger) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		version: version,
		logger:  logger,
	}
}

// CreateTaskRequest represents a task creation request
type CreateTaskRequest struct {
	Content    string   `json:"content"`
	Priority   string   `json:"priority"`
	Repository string   `json:"repository"`
	Tags       []string `json:"tags,omitempty"`
}

// UpdateTaskRequest represents a task update request
type UpdateTaskRequest struct {
	Content  *string `json:"content,omitempty"`
	Priority *string `json:"priority,omitempty"`
	Status   *string `json:"status,omitempty"`
}

// TaskResponse represents a task response
type TaskResponse struct {
	ID         string    `json:"id"`
	Content    string    `json:"content"`
	Priority   string    `json:"priority"`
	Status     string    `json:"status"`
	Repository string    `json:"repository"`
	Tags       []string  `json:"tags"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// BatchSyncRequest represents a batch sync request
type BatchSyncRequest struct {
	LastSyncTime *time.Time     `json:"last_sync_time,omitempty"`
	LocalTasks   []TaskSyncItem `json:"local_tasks"`
	Repository   string         `json:"repository"`
	ClientID     string         `json:"client_id,omitempty"`
}

// TaskSyncItem represents a task for synchronization
type TaskSyncItem struct {
	ID           string    `json:"id"`
	Content      string    `json:"content"`
	Status       string    `json:"status"`
	Priority     string    `json:"priority"`
	UpdatedAt    time.Time `json:"updated_at"`
	LocalVersion int       `json:"local_version"`
}

// BatchSyncResponse represents a batch sync response
type BatchSyncResponse struct {
	ServerTasks []TaskSyncItem `json:"server_tasks"`
	Conflicts   []ConflictItem `json:"conflicts"`
	ToCreate    []string       `json:"to_create"`
	ToUpdate    []string       `json:"to_update"`
	ToDelete    []string       `json:"to_delete"`
	ServerTime  time.Time      `json:"server_time"`
	SyncToken   string         `json:"sync_token"`
}

// ConflictItem represents a sync conflict
type ConflictItem struct {
	TaskID     string             `json:"task_id"`
	LocalTask  *TaskSyncItem      `json:"local_task"`
	ServerTask *TaskSyncItem      `json:"server_task"`
	Resolution ConflictResolution `json:"resolution"`
	Reason     string             `json:"reason"`
}

// ConflictResolution represents how a conflict was resolved
type ConflictResolution struct {
	Strategy     string        `json:"strategy"`
	ResolvedTask *TaskSyncItem `json:"resolved_task"`
}

// CreateTask creates a new task via HTTP API
func (c *HTTPClient) CreateTask(ctx context.Context, req *CreateTaskRequest) (*TaskResponse, error) {
	// This is a mock implementation for testing
	// In real integration tests, this would make actual HTTP requests

	task := &TaskResponse{
		ID:         generateID(),
		Content:    req.Content,
		Priority:   req.Priority,
		Status:     "pending",
		Repository: req.Repository,
		Tags:       req.Tags,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	c.logger.Info("created task via HTTP",
		slog.String("task_id", task.ID),
		slog.String("content", task.Content))

	return task, nil
}

// UpdateTask updates an existing task via HTTP API
func (c *HTTPClient) UpdateTask(ctx context.Context, taskID string, req *UpdateTaskRequest) (*TaskResponse, error) {
	// Mock implementation for testing
	task := &TaskResponse{
		ID:        taskID,
		Content:   "Updated content",
		Priority:  "medium",
		Status:    "pending",
		UpdatedAt: time.Now(),
	}

	if req.Content != nil {
		task.Content = *req.Content
	}
	if req.Priority != nil {
		task.Priority = *req.Priority
	}
	if req.Status != nil {
		task.Status = *req.Status
	}

	c.logger.Info("updated task via HTTP",
		slog.String("task_id", taskID))

	return task, nil
}

// BatchClient represents a client for batch operations
type BatchClient struct {
	baseURL string
	version string
	logger  *slog.Logger
}

// NewBatchClient creates a new batch client for testing
func NewBatchClient(baseURL, version string, logger *slog.Logger) *BatchClient {
	return &BatchClient{
		baseURL: baseURL,
		version: version,
		logger:  logger,
	}
}

// BatchSync performs a batch synchronization
func (c *BatchClient) BatchSync(ctx context.Context, req *BatchSyncRequest) (*BatchSyncResponse, error) {
	// Mock implementation that simulates conflicts
	resp := &BatchSyncResponse{
		ServerTasks: []TaskSyncItem{},
		Conflicts:   []ConflictItem{},
		ToCreate:    []string{},
		ToUpdate:    []string{},
		ToDelete:    []string{},
		ServerTime:  time.Now(),
		SyncToken:   generateID(),
	}

	// Simulate some conflicts based on the request
	for _, localTask := range req.LocalTasks {
		// Simulate that tasks updated more than 1 minute ago create conflicts
		if time.Since(localTask.UpdatedAt) > time.Minute {
			conflict := ConflictItem{
				TaskID:    localTask.ID,
				LocalTask: &localTask,
				ServerTask: &TaskSyncItem{
					ID:        localTask.ID,
					Content:   "Server version of " + localTask.Content,
					Status:    localTask.Status,
					Priority:  localTask.Priority,
					UpdatedAt: time.Now(),
				},
				Resolution: ConflictResolution{
					Strategy: "server_wins",
					ResolvedTask: &TaskSyncItem{
						ID:        localTask.ID,
						Content:   "Server version of " + localTask.Content,
						Status:    localTask.Status,
						Priority:  localTask.Priority,
						UpdatedAt: time.Now(),
					},
				},
				Reason: "server has newer timestamp",
			}
			resp.Conflicts = append(resp.Conflicts, conflict)
		} else {
			// Add as server task (no conflict)
			resp.ServerTasks = append(resp.ServerTasks, localTask)
		}
	}

	c.logger.Info("batch sync completed",
		slog.Int("conflicts", len(resp.Conflicts)),
		slog.Int("server_tasks", len(resp.ServerTasks)))

	return resp, nil
}

// WebSocket client types for testing

// NotificationHub represents a hub for managing notifications
type NotificationHub struct {
	subscribers map[string]func(*TaskEvent)
	logger      *slog.Logger
}

// NewNotificationHub creates a new notification hub
func NewNotificationHub(logger *slog.Logger) *NotificationHub {
	return &NotificationHub{
		subscribers: make(map[string]func(*TaskEvent)),
		logger:      logger,
	}
}

// Subscribe adds a subscriber to the hub
func (h *NotificationHub) Subscribe(name string, handler func(*TaskEvent)) {
	h.subscribers[name] = handler
}

// SubscribeToConnectionStatus subscribes to connection status changes
func (h *NotificationHub) SubscribeToConnectionStatus(handler func(bool)) {
	// Mock implementation
}

// PublishTaskEvent publishes a task event to all subscribers
func (h *NotificationHub) PublishTaskEvent(event *TaskEvent) {
	for _, handler := range h.subscribers {
		handler(event)
	}
}

// TaskEvent represents a task-related event
type TaskEvent struct {
	Type       EventType     `json:"type"`
	TaskID     string        `json:"task_id"`
	Repository string        `json:"repository"`
	ChangedBy  string        `json:"changed_by,omitempty"`
	Task       *TaskResponse `json:"task,omitempty"`
}

// EventType represents the type of event
type EventType string

const (
	EventTypeTaskCreated = EventType("task_created")
	EventTypeTaskUpdated = EventType("task_updated")
	EventTypeTaskDeleted = EventType("task_deleted")
)

// WebSocketClient represents a WebSocket client for testing
type WebSocketClient struct {
	url       string
	version   string
	hub       *NotificationHub
	logger    *slog.Logger
	connected bool
}

// NewWebSocketClient creates a new WebSocket client for testing
func NewWebSocketClient(url, version string, hub *NotificationHub, logger *slog.Logger) *WebSocketClient {
	return &WebSocketClient{
		url:     url,
		version: version,
		hub:     hub,
		logger:  logger,
	}
}

// Connect establishes a WebSocket connection
func (c *WebSocketClient) Connect(ctx context.Context) error {
	c.connected = true
	c.logger.Info("WebSocket connected", slog.String("url", c.url))
	return nil
}

// Close closes the WebSocket connection
func (c *WebSocketClient) Close() {
	c.connected = false
	c.logger.Info("WebSocket closed")
}

// IsConnected returns whether the client is connected
func (c *WebSocketClient) IsConnected() bool {
	return c.connected
}

// SubscribeToRepositories subscribes to events from specific repositories
func (c *WebSocketClient) SubscribeToRepositories(repos []string) error {
	c.logger.Info("subscribed to repositories", slog.Any("repos", repos))
	return nil
}

// ForceDisconnect simulates a forced disconnection
func (c *WebSocketClient) ForceDisconnect() {
	c.connected = false
	c.logger.Info("WebSocket force disconnected")
}

// SyncManager represents a sync manager for testing
type SyncManager struct {
	httpClient *HTTPClient
	wsClient   *WebSocketClient
	logger     *slog.Logger
}

// NewSyncManager creates a new sync manager for testing
func NewSyncManager(httpClient *HTTPClient, wsClient *WebSocketClient, logger *slog.Logger) *SyncManager {
	return &SyncManager{
		httpClient: httpClient,
		wsClient:   wsClient,
		logger:     logger,
	}
}

// Helper functions

func generateID() string {
	return fmt.Sprintf("test-%d", time.Now().UnixNano())
}
