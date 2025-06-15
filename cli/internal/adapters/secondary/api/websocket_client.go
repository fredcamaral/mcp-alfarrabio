// Package api provides HTTP and WebSocket clients for server communication.
package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"log/slog"

	"github.com/gorilla/websocket"
)

// WebSocketClient handles real-time WebSocket connections to the MCP server
type WebSocketClient struct {
	serverURL       string
	version         string
	conn            *websocket.Conn
	hub             NotificationHub
	reconnectDelay  time.Duration
	maxReconnect    time.Duration
	done            chan struct{}
	logger          *slog.Logger
	subscribedRepos []string
	mu              sync.RWMutex
	isConnected     bool
	reconnectCount  int
}

// NotificationHub handles publishing events to subscribers
type NotificationHub interface {
	PublishTaskEvent(event *TaskEvent)
	PublishConnectionStatus(connected bool)
	PublishSystemEvent(event *SystemEvent)
}

// WebSocket event types
const (
	EventTypeTaskCreated    = "task.created"
	EventTypeTaskUpdated    = "task.updated"
	EventTypeTaskDeleted    = "task.deleted"
	EventTypeSubscribe      = "subscribe"
	EventTypePing           = "ping"
	EventTypePong           = "pong"
	EventTypeError          = "error"
	EventTypeSystemMessage  = "system.message"
	EventTypeRepositorySync = "repository.sync"
)

// WebSocket configuration constants
const (
	maxMessageSize = 512 * 1024 // 512KB
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	writeWait      = 10 * time.Second
)

// Event represents a WebSocket event
type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// TaskEvent represents a task-related event
type TaskEvent struct {
	Type       string    `json:"type"`
	TaskID     string    `json:"task_id"`
	Repository string    `json:"repository"`
	Task       *TaskData `json:"task,omitempty"`
	ChangedBy  string    `json:"changed_by"`
	Timestamp  time.Time `json:"timestamp"`
}

// TaskData represents task information in events
type TaskData struct {
	ID            string    `json:"id"`
	Content       string    `json:"content"`
	Status        string    `json:"status"`
	Priority      string    `json:"priority"`
	EstimatedMins int       `json:"estimated_mins"`
	Repository    string    `json:"repository"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SubscribeEvent represents a subscription request
type SubscribeEvent struct {
	Repositories []string `json:"repositories"`
	EventTypes   []string `json:"event_types,omitempty"`
}

// SystemEvent represents system-level notifications
type SystemEvent struct {
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"` // info, warning, error
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewWebSocketClient creates a new WebSocket client
func NewWebSocketClient(serverURL, version string, hub NotificationHub, logger *slog.Logger) *WebSocketClient {
	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)

	return &WebSocketClient{
		serverURL:      wsURL,
		version:        version,
		hub:            hub,
		reconnectDelay: 1 * time.Second,
		maxReconnect:   60 * time.Second,
		done:           make(chan struct{}),
		logger:         logger,
		isConnected:    false,
	}
}

// Connect establishes a WebSocket connection to the server
func (c *WebSocketClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected && c.conn != nil {
		return nil // Already connected
	}

	// Parse WebSocket URL
	u, err := url.Parse(c.serverURL + "/ws")
	if err != nil {
		return fmt.Errorf("invalid WebSocket URL: %w", err)
	}

	headers := http.Header{}
	headers.Set("X-Version", c.version)
	headers.Set("User-Agent", "lmmc-cli/"+c.version)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	c.logger.Info("connecting to WebSocket server", slog.String("url", u.String()))

	conn, resp, err := dialer.DialContext(ctx, u.String(), headers)
	if err != nil {
		if resp != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				c.logger.Debug("failed to close response body", slog.Any("error", closeErr))
			}
			return fmt.Errorf("websocket dial failed (status %d): %w", resp.StatusCode, err)
		}
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	if resp != nil {
		if closeErr := resp.Body.Close(); closeErr != nil {
			c.logger.Debug("failed to close response body", slog.Any("error", closeErr))
		}
	}

	c.conn = conn
	c.isConnected = true
	c.reconnectCount = 0

	// Notify connection status
	if c.hub != nil {
		c.hub.PublishConnectionStatus(true)
	}

	// Start message handling goroutines
	go c.readPump(ctx)
	go c.pingPump()

	// Subscribe to repositories if any
	if len(c.subscribedRepos) > 0 {
		if err := c.sendSubscribe(); err != nil {
			c.logger.Warn("failed to send initial subscription", slog.Any("error", err))
		}
	}

	c.logger.Info("websocket connected successfully",
		slog.String("url", c.serverURL),
		slog.Int("subscribed_repos", len(c.subscribedRepos)))

	return nil
}

// SubscribeToRepositories sets up repository subscriptions
func (c *WebSocketClient) SubscribeToRepositories(repos []string) error {
	c.mu.Lock()
	c.subscribedRepos = repos
	isConnected := c.isConnected && c.conn != nil
	c.mu.Unlock()

	if isConnected {
		return c.sendSubscribe()
	}

	c.logger.Info("subscription queued for connection",
		slog.String("repos", strings.Join(repos, ", ")))
	return nil
}

// sendSubscribe sends subscription request to server
func (c *WebSocketClient) sendSubscribe() error {
	event := Event{
		Type:      EventTypeSubscribe,
		Timestamp: time.Now(),
		Data: SubscribeEvent{
			Repositories: c.subscribedRepos,
			EventTypes: []string{
				EventTypeTaskCreated,
				EventTypeTaskUpdated,
				EventTypeTaskDeleted,
				EventTypeRepositorySync,
			},
		},
	}

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return errors.New("not connected")
	}

	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		c.logger.Debug("failed to set write deadline", slog.Any("error", err))
	}
	if err := conn.WriteJSON(event); err != nil {
		return fmt.Errorf("failed to send subscription: %w", err)
	}

	c.logger.Info("sent subscription request",
		slog.String("repos", strings.Join(c.subscribedRepos, ", ")))

	return nil
}

// readPump handles incoming WebSocket messages
func (c *WebSocketClient) readPump(ctx context.Context) {
	defer func() {
		c.mu.Lock()
		if c.conn != nil {
			if err := c.conn.Close(); err != nil {
				c.logger.Debug("error closing connection in readPump", slog.Any("error", err))
			}
			c.conn = nil
		}
		c.isConnected = false
		c.mu.Unlock()

		// Notify disconnection
		if c.hub != nil {
			c.hub.PublishConnectionStatus(false)
		}

		// Attempt reconnection unless explicitly closed
		select {
		case <-c.done:
			c.logger.Info("websocket client shut down")
		default:
			c.logger.Info("websocket disconnected, attempting reconnection")
			go c.reconnectLoop(ctx)
		}
	}()

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return
	}

	conn.SetReadLimit(maxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.logger.Debug("failed to set initial read deadline", slog.Any("error", err))
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			c.logger.Debug("failed to set pong read deadline", slog.Any("error", err))
		}
		return nil
	})

	for {
		var event Event
		err := conn.ReadJSON(&event)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("websocket read error", slog.Any("error", err))
			} else {
				c.logger.Debug("websocket connection closed", slog.Any("error", err))
			}
			break
		}

		c.handleEvent(event)
	}
}

// pingPump sends periodic ping messages to keep connection alive
func (c *WebSocketClient) pingPump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			isConnected := c.isConnected
			c.mu.RUnlock()

			if !isConnected || conn == nil {
				return
			}

			if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.logger.Debug("failed to set write deadline for ping", slog.Any("error", err))
				return
			}
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Debug("ping failed", slog.Any("error", err))
				return
			}

		case <-c.done:
			return
		}
	}
}

// handleEvent processes incoming WebSocket events
func (c *WebSocketClient) handleEvent(event Event) {
	c.logger.Debug("received websocket event",
		slog.String("type", event.Type),
		slog.String("timestamp", event.Timestamp.Format(time.RFC3339)))

	switch event.Type {
	case EventTypeTaskCreated, EventTypeTaskUpdated, EventTypeTaskDeleted:
		c.handleTaskEvent(event)

	case EventTypePing:
		c.handlePing()

	case EventTypeError:
		c.logger.Error("server error event",
			slog.String("error", event.Error),
			slog.String("request_id", event.RequestID))

	case EventTypeSystemMessage:
		c.handleSystemEvent(event)

	case EventTypeRepositorySync:
		c.handleRepositorySync(event)

	default:
		c.logger.Debug("unknown event type", slog.String("type", event.Type))
	}
}

// handleTaskEvent processes task-related events
func (c *WebSocketClient) handleTaskEvent(event Event) {
	if c.hub == nil {
		return
	}

	// Parse task event data
	eventData, ok := event.Data.(map[string]interface{})
	if !ok {
		c.logger.Warn("invalid task event data format")
		return
	}

	taskEvent := &TaskEvent{
		Type:      event.Type,
		Timestamp: event.Timestamp,
	}

	// Extract task event fields
	if taskID, ok := eventData["task_id"].(string); ok {
		taskEvent.TaskID = taskID
	}
	if repository, ok := eventData["repository"].(string); ok {
		taskEvent.Repository = repository
	}
	if changedBy, ok := eventData["changed_by"].(string); ok {
		taskEvent.ChangedBy = changedBy
	}

	// Extract task data if present
	if taskData, ok := eventData["task"].(map[string]interface{}); ok {
		taskEvent.Task = c.parseTaskData(taskData)
	}

	c.hub.PublishTaskEvent(taskEvent)
}

// parseTaskData converts map data to TaskData struct
func (c *WebSocketClient) parseTaskData(data map[string]interface{}) *TaskData {
	task := &TaskData{}

	if id, ok := data["id"].(string); ok {
		task.ID = id
	}
	if content, ok := data["content"].(string); ok {
		task.Content = content
	}
	if status, ok := data["status"].(string); ok {
		task.Status = status
	}
	if priority, ok := data["priority"].(string); ok {
		task.Priority = priority
	}
	if estimatedMins, ok := data["estimated_mins"].(float64); ok {
		task.EstimatedMins = int(estimatedMins)
	}
	if repository, ok := data["repository"].(string); ok {
		task.Repository = repository
	}

	// Parse timestamps
	if createdAt, ok := data["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			task.CreatedAt = t
		}
	}
	if updatedAt, ok := data["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			task.UpdatedAt = t
		}
	}

	return task
}

// handlePing responds to server ping with pong
func (c *WebSocketClient) handlePing() {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn != nil {
		pongEvent := Event{
			Type:      EventTypePong,
			Timestamp: time.Now(),
		}

		if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
			c.logger.Debug("failed to set write deadline for pong", slog.Any("error", err))
			return
		}
		if err := conn.WriteJSON(pongEvent); err != nil {
			c.logger.Debug("failed to send pong", slog.Any("error", err))
		}
	}
}

// handleSystemEvent processes system-level events
func (c *WebSocketClient) handleSystemEvent(event Event) {
	if c.hub == nil {
		return
	}

	eventData, ok := event.Data.(map[string]interface{})
	if !ok {
		return
	}

	systemEvent := &SystemEvent{
		Type:      event.Type,
		Timestamp: event.Timestamp,
	}

	if message, ok := eventData["message"].(string); ok {
		systemEvent.Message = message
	}
	if severity, ok := eventData["severity"].(string); ok {
		systemEvent.Severity = severity
	}
	if data, ok := eventData["data"].(map[string]interface{}); ok {
		systemEvent.Data = data
	}

	c.hub.PublishSystemEvent(systemEvent)
}

// handleRepositorySync processes repository synchronization events
func (c *WebSocketClient) handleRepositorySync(event Event) {
	c.logger.Info("repository sync event received",
		slog.String("timestamp", event.Timestamp.Format(time.RFC3339)))

	// Can be extended to handle specific sync operations
	if c.hub != nil {
		systemEvent := &SystemEvent{
			Type:      "repository.sync",
			Message:   "Repository synchronization completed",
			Severity:  "info",
			Timestamp: event.Timestamp,
		}
		c.hub.PublishSystemEvent(systemEvent)
	}
}

// reconnectLoop handles automatic reconnection with exponential backoff
func (c *WebSocketClient) reconnectLoop(ctx context.Context) {
	delay := c.reconnectDelay

	for {
		select {
		case <-c.done:
			return
		case <-ctx.Done():
			return
		default:
		}

		c.reconnectCount++
		c.logger.Info("attempting to reconnect",
			slog.Int("attempt", c.reconnectCount),
			slog.Duration("delay", delay))

		time.Sleep(delay)

		connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := c.Connect(connectCtx)
		cancel()

		if err == nil {
			c.logger.Info("reconnection successful",
				slog.Int("attempts", c.reconnectCount))
			return
		}

		c.logger.Warn("reconnection failed",
			slog.Any("error", err),
			slog.Int("attempt", c.reconnectCount))

		// Exponential backoff with maximum limit
		delay *= 2
		if delay > c.maxReconnect {
			delay = c.maxReconnect
		}
	}
}

// IsConnected returns the current connection status
func (c *WebSocketClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected && c.conn != nil
}

// GetSubscribedRepositories returns the list of subscribed repositories
func (c *WebSocketClient) GetSubscribedRepositories() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]string, len(c.subscribedRepos))
	copy(result, c.subscribedRepos)
	return result
}

// Close gracefully closes the WebSocket connection
func (c *WebSocketClient) Close() error {
	close(c.done)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		// Send close message
		if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
			c.logger.Debug("failed to set write deadline for close", slog.Any("error", err))
		}
		err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			c.logger.Debug("failed to send close message", slog.Any("error", err))
		}

		// Close connection
		closeErr := c.conn.Close()
		c.conn = nil
		c.isConnected = false

		c.logger.Info("websocket connection closed")
		return closeErr
	}

	return nil
}

// SendMessage sends a custom message to the server (for testing or special cases)
func (c *WebSocketClient) SendMessage(eventType string, data interface{}) error {
	c.mu.RLock()
	conn := c.conn
	isConnected := c.isConnected
	c.mu.RUnlock()

	if !isConnected || conn == nil {
		return errors.New("not connected to WebSocket server")
	}

	event := Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return fmt.Errorf("failed to set write deadline: %w", err)
	}
	if err := conn.WriteJSON(event); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
