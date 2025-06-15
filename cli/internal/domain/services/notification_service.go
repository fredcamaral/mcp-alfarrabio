// Package services provides domain services for the lerian-mcp-memory CLI.
package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"lerian-mcp-memory-cli/internal/adapters/secondary/api"
	"lerian-mcp-memory-cli/internal/domain/constants"
)

// NotificationService handles displaying real-time notifications from the server
type NotificationService struct {
	eventChan   chan *api.TaskEvent
	statusChan  chan bool
	systemChan  chan *api.SystemEvent
	output      io.Writer
	logger      *slog.Logger
	mu          sync.RWMutex
	subscribers map[string]chan NotificationEvent
	isRunning   bool
	settings    NotificationSettings
}

// NotificationSettings configures notification behavior
type NotificationSettings struct {
	ShowTaskEvents       bool `json:"show_task_events"`
	ShowConnectionStatus bool `json:"show_connection_status"`
	ShowSystemMessages   bool `json:"show_system_messages"`
	EnableSounds         bool `json:"enable_sounds"`
	QuietMode            bool `json:"quiet_mode"`
	MaxDisplayLength     int  `json:"max_display_length"`
}

// NotificationEvent represents a processed notification for display
type NotificationEvent struct {
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// DefaultNotificationHub implements the NotificationHub interface
type DefaultNotificationHub struct {
	service *NotificationService
	logger  *slog.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(output io.Writer, logger *slog.Logger) *NotificationService {
	return &NotificationService{
		eventChan:   make(chan *api.TaskEvent, 100),
		statusChan:  make(chan bool, 10),
		systemChan:  make(chan *api.SystemEvent, 50),
		output:      output,
		logger:      logger,
		subscribers: make(map[string]chan NotificationEvent),
		settings: NotificationSettings{
			ShowTaskEvents:       true,
			ShowConnectionStatus: true,
			ShowSystemMessages:   true,
			EnableSounds:         false,
			QuietMode:            false,
			MaxDisplayLength:     80,
		},
	}
}

// NewNotificationHub creates a notification hub that uses the notification service
func NewNotificationHub(service *NotificationService, logger *slog.Logger) *DefaultNotificationHub {
	return &DefaultNotificationHub{
		service: service,
		logger:  logger,
	}
}

// PublishTaskEvent publishes a task-related event
func (h *DefaultNotificationHub) PublishTaskEvent(event *api.TaskEvent) {
	if h.service != nil {
		select {
		case h.service.eventChan <- event:
		default:
			h.logger.Warn("task event channel full, dropping event",
				slog.String("task_id", event.TaskID),
				slog.String("type", event.Type))
		}
	}
}

// PublishConnectionStatus publishes connection status changes
func (h *DefaultNotificationHub) PublishConnectionStatus(connected bool) {
	if h.service != nil {
		select {
		case h.service.statusChan <- connected:
		default:
			h.logger.Warn("status channel full, dropping status update")
		}
	}
}

// PublishSystemEvent publishes system-level events
func (h *DefaultNotificationHub) PublishSystemEvent(event *api.SystemEvent) {
	if h.service != nil {
		select {
		case h.service.systemChan <- event:
		default:
			h.logger.Warn("system event channel full, dropping event",
				slog.String("type", event.Type))
		}
	}
}

// Start begins processing notifications
func (n *NotificationService) Start(ctx context.Context) {
	n.mu.Lock()
	if n.isRunning {
		n.mu.Unlock()
		return
	}
	n.isRunning = true
	n.mu.Unlock()

	n.logger.Info("notification service started")

	for {
		select {
		case <-ctx.Done():
			n.logger.Info("notification service stopped")
			n.mu.Lock()
			n.isRunning = false
			n.mu.Unlock()
			return

		case event := <-n.eventChan:
			n.handleTaskEvent(event)

		case connected := <-n.statusChan:
			n.handleConnectionStatus(connected)

		case systemEvent := <-n.systemChan:
			n.handleSystemEvent(systemEvent)
		}
	}
}

// UpdateSettings updates notification settings
func (n *NotificationService) UpdateSettings(settings NotificationSettings) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.settings = settings
}

// GetSettings returns current notification settings
func (n *NotificationService) GetSettings() NotificationSettings {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.settings
}

// Subscribe allows external components to receive notification events
func (n *NotificationService) Subscribe(id string) <-chan NotificationEvent {
	n.mu.Lock()
	defer n.mu.Unlock()

	ch := make(chan NotificationEvent, 50)
	n.subscribers[id] = ch
	return ch
}

// Unsubscribe removes a subscriber
func (n *NotificationService) Unsubscribe(id string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if ch, exists := n.subscribers[id]; exists {
		close(ch)
		delete(n.subscribers, id)
	}
}

// handleTaskEvent processes task-related notifications
func (n *NotificationService) handleTaskEvent(event *api.TaskEvent) {
	if !n.shouldShowTaskEvents() {
		return
	}

	// Don't show notifications for events triggered by CLI itself
	if event.ChangedBy == "cli" || event.ChangedBy == "lmmc" {
		return
	}

	var message string
	var severity string

	switch event.Type {
	case api.EventTypeTaskCreated:
		if event.Task != nil {
			content := n.truncateContent(event.Task.Content)
			message = "📝 New task created: " + content
		} else {
			message = fmt.Sprintf("📝 New task created (ID: %s)", event.TaskID)
		}
		severity = constants.SeverityInfo

	case api.EventTypeTaskUpdated:
		if event.Task != nil {
			content := n.truncateContent(event.Task.Content)
			message = "✏️  Task updated: " + content
		} else {
			message = fmt.Sprintf("✏️  Task updated (ID: %s)", event.TaskID)
		}
		severity = constants.SeverityInfo

	case api.EventTypeTaskDeleted:
		message = fmt.Sprintf("🗑️  Task deleted (ID: %s)", event.TaskID)
		severity = constants.SeverityWarning

	default:
		message = fmt.Sprintf("📋 Task event: %s (ID: %s)", event.Type, event.TaskID)
		severity = constants.SeverityInfo
	}

	n.displayNotification(message, severity)
	n.publishToSubscribers(NotificationEvent{
		Type:      "task",
		Message:   message,
		Severity:  severity,
		Timestamp: event.Timestamp,
		Data: map[string]interface{}{
			"task_id":    event.TaskID,
			"repository": event.Repository,
			"changed_by": event.ChangedBy,
			"event_type": event.Type,
		},
	})
}

// handleConnectionStatus processes connection status changes
func (n *NotificationService) handleConnectionStatus(connected bool) {
	if !n.shouldShowConnectionStatus() {
		return
	}

	var message string
	var severity string

	if connected {
		message = "✅ Connected to server"
		severity = "success"
	} else {
		message = "⚠️  Disconnected from server (working offline)"
		severity = constants.SeverityWarning
	}

	n.displayNotification(message, severity)
	n.publishToSubscribers(NotificationEvent{
		Type:      "connection",
		Message:   message,
		Severity:  severity,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"connected": connected,
		},
	})
}

// handleSystemEvent processes system-level notifications
func (n *NotificationService) handleSystemEvent(event *api.SystemEvent) {
	if !n.shouldShowSystemMessages() {
		return
	}

	var icon string
	switch event.Severity {
	case constants.SeverityError:
		icon = "❌"
	case constants.SeverityWarning:
		icon = "⚠️ "
	case "success":
		icon = "✅"
	default:
		icon = "ℹ️ "
	}

	message := fmt.Sprintf("%s %s", icon, event.Message)
	n.displayNotification(message, event.Severity)

	n.publishToSubscribers(NotificationEvent{
		Type:      "system",
		Message:   message,
		Severity:  event.Severity,
		Timestamp: event.Timestamp,
		Data:      event.Data,
	})
}

// displayNotification shows a notification to the user
func (n *NotificationService) displayNotification(message, severity string) {
	if n.isQuietMode() {
		return
	}

	timestamp := time.Now().Format("15:04:05")

	// Format message with timestamp and prompt continuation
	formattedMessage := fmt.Sprintf("\n[%s] %s\n> ", timestamp, message)

	// Write to output
	if n.output != nil {
		if _, err := fmt.Fprint(n.output, formattedMessage); err != nil {
			// Log error but don't return as this is a notification
			n.logger.Error("failed to write notification", slog.String("error", err.Error()))
		}
	}

	// Log the notification
	switch severity {
	case constants.SeverityError:
		n.logger.Error("notification displayed", slog.String("message", message))
	case constants.SeverityWarning:
		n.logger.Warn("notification displayed", slog.String("message", message))
	default:
		n.logger.Info("notification displayed", slog.String("message", message))
	}
}

// publishToSubscribers sends events to all subscribers
func (n *NotificationService) publishToSubscribers(event NotificationEvent) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	for id, ch := range n.subscribers {
		select {
		case ch <- event:
		default:
			n.logger.Warn("subscriber channel full, dropping event",
				slog.String("subscriber_id", id))
		}
	}
}

// truncateContent truncates content to maximum display length
func (n *NotificationService) truncateContent(content string) string {
	maxLen := n.getMaxDisplayLength()
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen-3] + "..."
}

// Settings helper methods
func (n *NotificationService) shouldShowTaskEvents() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.settings.ShowTaskEvents
}

func (n *NotificationService) shouldShowConnectionStatus() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.settings.ShowConnectionStatus
}

func (n *NotificationService) shouldShowSystemMessages() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.settings.ShowSystemMessages
}

func (n *NotificationService) isQuietMode() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.settings.QuietMode
}

func (n *NotificationService) getMaxDisplayLength() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if n.settings.MaxDisplayLength > 0 {
		return n.settings.MaxDisplayLength
	}
	return 80
}

// IsRunning returns whether the notification service is currently running
func (n *NotificationService) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.isRunning
}

// GetStats returns notification statistics
func (n *NotificationService) GetStats() NotificationStats {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return NotificationStats{
		EventChannelLength:  len(n.eventChan),
		StatusChannelLength: len(n.statusChan),
		SystemChannelLength: len(n.systemChan),
		SubscriberCount:     len(n.subscribers),
		IsRunning:           n.isRunning,
	}
}

// NotificationStats provides statistics about the notification service
type NotificationStats struct {
	EventChannelLength  int  `json:"event_channel_length"`
	StatusChannelLength int  `json:"status_channel_length"`
	SystemChannelLength int  `json:"system_channel_length"`
	SubscriberCount     int  `json:"subscriber_count"`
	IsRunning           bool `json:"is_running"`
}

// EnableQuietMode temporarily disables notifications
func (n *NotificationService) EnableQuietMode() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.settings.QuietMode = true
}

// DisableQuietMode re-enables notifications
func (n *NotificationService) DisableQuietMode() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.settings.QuietMode = false
}

// ClearChannels drains all notification channels (useful for testing)
func (n *NotificationService) ClearChannels() {
	// Drain event channel
	for {
		select {
		case <-n.eventChan:
		default:
			goto drainStatus
		}
	}

drainStatus:
	// Drain status channel
	for {
		select {
		case <-n.statusChan:
		default:
			goto drainSystem
		}
	}

drainSystem:
	// Drain system channel
	for {
		select {
		case <-n.systemChan:
		default:
			return
		}
	}
}
