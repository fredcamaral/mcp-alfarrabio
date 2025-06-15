// Package api provides synchronization between HTTP and WebSocket clients.
package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// SyncManager coordinates HTTP and WebSocket communications with the server
type SyncManager struct {
	mcpClient        ports.MCPClient
	wsClient         *WebSocketClient
	localStorage     ports.Storage
	notificationSvc  NotificationServiceInterface
	conflictResolver ConflictResolver
	logger           *slog.Logger
	mu               sync.Mutex
	isOnline         bool
	offlineQueue     []*QueuedOperation
	syncInProgress   bool
	lastSyncTime     time.Time
}

// ConflictResolver handles conflicts between local and remote data
type ConflictResolver interface {
	ResolveConflict(local, remote *entities.Task) (*entities.Task, error)
	GetResolutionStrategy() string
}

// NotificationServiceInterface defines the interface for notification services
type NotificationServiceInterface interface {
	IsRunning() bool
	Start(ctx context.Context)
}

// QueuedOperation represents an operation to retry when online
type QueuedOperation struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // create, update, delete
	TaskID    string                 `json:"task_id,omitempty"`
	Task      *entities.Task         `json:"task,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Retries   int                    `json:"retries"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SyncManagerConfig contains configuration for the sync manager
type SyncManagerConfig struct {
	EnableWebSocket      bool          `json:"enable_websocket"`
	SyncInterval         time.Duration `json:"sync_interval"`
	MaxOfflineOperations int           `json:"max_offline_operations"`
	MaxRetries           int           `json:"max_retries"`
	ConflictStrategy     string        `json:"conflict_strategy"` // server_wins, local_wins, merge
}

// ServerTruthResolver implements ConflictResolver with server-wins strategy
type ServerTruthResolver struct {
	logger *slog.Logger
}

// NewSyncManager creates a new sync manager
func NewSyncManager(
	mcpClient ports.MCPClient,
	wsClient *WebSocketClient,
	localStorage ports.Storage,
	notificationSvc NotificationServiceInterface,
	logger *slog.Logger,
) *SyncManager {
	return &SyncManager{
		mcpClient:        mcpClient,
		wsClient:         wsClient,
		localStorage:     localStorage,
		notificationSvc:  notificationSvc,
		conflictResolver: NewServerTruthResolver(logger),
		logger:           logger,
		isOnline:         false,
		offlineQueue:     make([]*QueuedOperation, 0),
	}
}

// NewServerTruthResolver creates a resolver that favors server state
func NewServerTruthResolver(logger *slog.Logger) *ServerTruthResolver {
	return &ServerTruthResolver{logger: logger}
}

// ResolveConflict resolves conflicts by preferring server state
func (r *ServerTruthResolver) ResolveConflict(local, remote *entities.Task) (*entities.Task, error) {
	r.logger.Info("resolving conflict with server truth strategy",
		slog.String("task_id", remote.ID),
		slog.String("local_updated", local.UpdatedAt.Format(time.RFC3339)),
		slog.String("remote_updated", remote.UpdatedAt.Format(time.RFC3339)))

	// Server wins - return remote task but log the conflict
	r.logger.Warn("task conflict resolved in favor of server",
		slog.String("task_id", remote.ID),
		slog.String("local_content", local.Content),
		slog.String("remote_content", remote.Content))

	return remote, nil
}

// GetResolutionStrategy returns the resolution strategy name
func (r *ServerTruthResolver) GetResolutionStrategy() string {
	return "server_wins"
}

// Start initializes the sync manager and starts background processes
func (m *SyncManager) Start(ctx context.Context) error {
	m.logger.Info("starting sync manager")

	// Start notification service if not running
	if m.notificationSvc != nil && !m.notificationSvc.IsRunning() {
		go m.notificationSvc.Start(ctx)
	}

	// Start WebSocket connection if available
	if m.wsClient != nil {
		go m.maintainWebSocketConnection(ctx)
	}

	// Start periodic sync
	go m.periodicSync(ctx)

	// Test initial connection
	m.updateConnectionStatus(ctx)

	return nil
}

// CreateTask creates a task with fallback to offline queue
func (m *SyncManager) CreateTask(ctx context.Context, task *entities.Task) error {
	// Try server first
	if m.isOnline && m.mcpClient != nil {
		err := m.mcpClient.SyncTask(ctx, task)
		if err == nil {
			m.logger.Info("task created on server", slog.String("task_id", task.ID))

			// Store locally
			if localErr := m.localStorage.SaveTask(ctx, task); localErr != nil {
				m.logger.Warn("failed to store task locally after server create",
					slog.String("task_id", task.ID),
					slog.Any("error", localErr))
			}
			return nil
		}

		m.logger.Warn("server create failed, queuing for retry",
			slog.String("task_id", task.ID),
			slog.Any("error", err))
	}

	// Store locally and queue for retry
	if err := m.localStorage.SaveTask(ctx, task); err != nil {
		return fmt.Errorf("failed to store task locally: %w", err)
	}

	// Add to offline queue
	m.queueOperation(&QueuedOperation{
		ID:        fmt.Sprintf("create_%s_%d", task.ID, time.Now().Unix()),
		Type:      "create",
		TaskID:    task.ID,
		Task:      task,
		Timestamp: time.Now(),
		Retries:   0,
	})

	m.logger.Info("task queued for server sync", slog.String("task_id", task.ID))
	return nil
}

// UpdateTask updates a task with conflict resolution
func (m *SyncManager) UpdateTask(ctx context.Context, task *entities.Task) error {
	// Try server first
	if m.isOnline && m.mcpClient != nil {
		err := m.mcpClient.UpdateTaskStatus(ctx, task.ID, task.Status)
		if err == nil {
			m.logger.Info("task updated on server", slog.String("task_id", task.ID))

			// Update locally
			if localErr := m.localStorage.UpdateTask(ctx, task); localErr != nil {
				m.logger.Warn("failed to update task locally after server update",
					slog.String("task_id", task.ID),
					slog.Any("error", localErr))
			}
			return nil
		}

		m.logger.Warn("server update failed, queuing for retry",
			slog.String("task_id", task.ID),
			slog.Any("error", err))
	}

	// Update locally and queue for retry
	if err := m.localStorage.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("failed to update task locally: %w", err)
	}

	// Add to offline queue
	m.queueOperation(&QueuedOperation{
		ID:        fmt.Sprintf("update_%s_%d", task.ID, time.Now().Unix()),
		Type:      "update",
		TaskID:    task.ID,
		Task:      task,
		Timestamp: time.Now(),
		Retries:   0,
	})

	m.logger.Info("task update queued for server sync", slog.String("task_id", task.ID))
	return nil
}

// SyncWithServer performs full synchronization with the server
func (m *SyncManager) SyncWithServer(ctx context.Context) error {
	m.mu.Lock()
	if m.syncInProgress {
		m.mu.Unlock()
		return errors.New("sync already in progress")
	}
	m.syncInProgress = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.syncInProgress = false
		m.lastSyncTime = time.Now()
		m.mu.Unlock()
	}()

	m.logger.Info("starting full server synchronization")

	// Update connection status
	m.updateConnectionStatus(ctx)

	if !m.isOnline {
		return errors.New("server not available for sync")
	}

	// Process offline queue
	m.processOfflineQueue(ctx)

	// Sync tasks from server
	if err := m.syncTasksFromServer(ctx); err != nil {
		m.logger.Error("failed to sync tasks from server", slog.Any("error", err))
		return err
	}

	m.logger.Info("server synchronization completed")
	return nil
}

// processOfflineQueue attempts to sync queued operations
func (m *SyncManager) processOfflineQueue(ctx context.Context) {
	m.mu.Lock()
	queue := make([]*QueuedOperation, len(m.offlineQueue))
	copy(queue, m.offlineQueue)
	m.offlineQueue = m.offlineQueue[:0] // Clear queue
	m.mu.Unlock()

	if len(queue) == 0 {
		return
	}

	m.logger.Info("processing offline queue", slog.Int("operations", len(queue)))

	var failedOps []*QueuedOperation

	for _, op := range queue {
		if err := m.processQueuedOperation(ctx, op); err != nil {
			op.Retries++
			if op.Retries < 3 { // Max retries
				failedOps = append(failedOps, op)
			}
			m.logger.Warn("queued operation failed",
				slog.String("operation_id", op.ID),
				slog.String("type", op.Type),
				slog.Int("retries", op.Retries),
				slog.Any("error", err))
		} else {
			m.logger.Info("queued operation succeeded",
				slog.String("operation_id", op.ID),
				slog.String("type", op.Type))
		}
	}

	// Re-queue failed operations
	m.mu.Lock()
	m.offlineQueue = append(m.offlineQueue, failedOps...)
	m.mu.Unlock()
}

// processQueuedOperation executes a single queued operation
func (m *SyncManager) processQueuedOperation(ctx context.Context, op *QueuedOperation) error {
	if m.mcpClient == nil {
		return errors.New("MCP client not available")
	}

	switch op.Type {
	case "create":
		if op.Task != nil {
			return m.mcpClient.SyncTask(ctx, op.Task)
		}

	case "update":
		if op.Task != nil {
			return m.mcpClient.UpdateTaskStatus(ctx, op.TaskID, op.Task.Status)
		}

	case "delete":
		// Note: Delete operation would need to be implemented in MCP client
		m.logger.Warn("delete operation not yet implemented", slog.String("task_id", op.TaskID))
		return nil

	default:
		return fmt.Errorf("unknown operation type: %s", op.Type)
	}

	return fmt.Errorf("insufficient data for operation: %s", op.Type)
}

// syncTasksFromServer pulls tasks from server and resolves conflicts
func (m *SyncManager) syncTasksFromServer(ctx context.Context) error {
	if m.mcpClient == nil {
		return errors.New("MCP client not available")
	}

	// Get repository from current directory or config
	repository := "current" // Simplified for now

	serverTasks, err := m.mcpClient.GetTasks(ctx, repository)
	if err != nil {
		return fmt.Errorf("failed to get tasks from server: %w", err)
	}

	localTasks, err := m.localStorage.GetTasksByRepository(ctx, repository)
	if err != nil {
		return fmt.Errorf("failed to get local tasks: %w", err)
	}

	// Create maps for easier lookup
	serverTaskMap := make(map[string]*entities.Task)
	for _, task := range serverTasks {
		serverTaskMap[task.ID] = task
	}

	localTaskMap := make(map[string]*entities.Task)
	for _, task := range localTasks {
		localTaskMap[task.ID] = task
	}

	// Resolve conflicts and update local storage
	for _, serverTask := range serverTasks {
		localTask, exists := localTaskMap[serverTask.ID]

		if !exists {
			// New task from server
			if err := m.localStorage.SaveTask(ctx, serverTask); err != nil {
				m.logger.Error("failed to create local task from server",
					slog.String("task_id", serverTask.ID),
					slog.Any("error", err))
			}
		} else if !localTask.UpdatedAt.Equal(serverTask.UpdatedAt) {
			// Conflict - resolve using conflict resolver
			resolved, err := m.conflictResolver.ResolveConflict(localTask, serverTask)
			if err != nil {
				m.logger.Error("failed to resolve conflict",
					slog.String("task_id", serverTask.ID),
					slog.Any("error", err))
				continue
			}

			if err := m.localStorage.UpdateTask(ctx, resolved); err != nil {
				m.logger.Error("failed to update local task after conflict resolution",
					slog.String("task_id", resolved.ID),
					slog.Any("error", err))
			}
		}
	}

	m.logger.Info("synced tasks from server",
		slog.Int("server_tasks", len(serverTasks)),
		slog.Int("local_tasks", len(localTasks)))

	return nil
}

// queueOperation adds an operation to the offline queue
func (m *SyncManager) queueOperation(op *QueuedOperation) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Limit queue size to prevent memory issues
	maxOps := 1000
	if len(m.offlineQueue) >= maxOps {
		// Remove oldest operation
		m.offlineQueue = m.offlineQueue[1:]
	}

	m.offlineQueue = append(m.offlineQueue, op)
}

// maintainWebSocketConnection manages WebSocket connection lifecycle
func (m *SyncManager) maintainWebSocketConnection(ctx context.Context) {
	if m.wsClient == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if !m.wsClient.IsConnected() {
			m.logger.Info("attempting WebSocket connection")
			if err := m.wsClient.Connect(ctx); err != nil {
				m.logger.Warn("WebSocket connection failed", slog.Any("error", err))
				time.Sleep(30 * time.Second) // Wait before retry
				continue
			}
		}

		time.Sleep(10 * time.Second) // Check connection every 10 seconds
	}
}

// periodicSync performs periodic synchronization with the server
func (m *SyncManager) periodicSync(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Sync every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.SyncWithServer(ctx); err != nil {
				m.logger.Debug("periodic sync failed", slog.Any("error", err))
			}
		}
	}
}

// updateConnectionStatus checks and updates connection status
func (m *SyncManager) updateConnectionStatus(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var online bool
	if m.mcpClient != nil {
		online = m.mcpClient.IsOnline()
		if !online {
			// Double-check with a test connection
			if err := m.mcpClient.TestConnection(ctx); err == nil {
				online = true
			}
		}
	}

	m.mu.Lock()
	wasOnline := m.isOnline
	m.isOnline = online
	m.mu.Unlock()

	// Notify if status changed
	if wasOnline != online {
		m.logger.Info("connection status changed", slog.Bool("online", online))
	}
}

// GetStatus returns current sync manager status
func (m *SyncManager) GetStatus() SyncManagerStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	return SyncManagerStatus{
		IsOnline:           m.isOnline,
		LastSyncTime:       m.lastSyncTime,
		QueuedOperations:   len(m.offlineQueue),
		SyncInProgress:     m.syncInProgress,
		WebSocketConnected: m.wsClient != nil && m.wsClient.IsConnected(),
	}
}

// SyncManagerStatus represents the current status of the sync manager
type SyncManagerStatus struct {
	IsOnline           bool      `json:"is_online"`
	LastSyncTime       time.Time `json:"last_sync_time"`
	QueuedOperations   int       `json:"queued_operations"`
	SyncInProgress     bool      `json:"sync_in_progress"`
	WebSocketConnected bool      `json:"websocket_connected"`
}

// GetOfflineQueueSize returns the number of queued operations
func (m *SyncManager) GetOfflineQueueSize() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.offlineQueue)
}

// ClearOfflineQueue clears all queued operations (use with caution)
func (m *SyncManager) ClearOfflineQueue() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.offlineQueue = m.offlineQueue[:0]
	m.logger.Info("offline queue cleared")
}

// Close gracefully shuts down the sync manager
func (m *SyncManager) Close() error {
	m.logger.Info("shutting down sync manager")

	// Close WebSocket connection
	if m.wsClient != nil {
		if err := m.wsClient.Close(); err != nil {
			m.logger.Warn("failed to close WebSocket client", slog.Any("error", err))
		}
	}

	return nil
}
