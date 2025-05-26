package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Notifier manages sending notifications to MCP clients
type Notifier struct {
	queue      chan NotificationMessage
	handlers   map[string][]DeliveryHandler
	clientInfo map[string]*ClientInfo
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// DeliveryHandler sends notifications to clients
type DeliveryHandler func(clientID string, notification *Notification) error

// ClientInfo stores information about a connected client
type ClientInfo struct {
	ID               string
	SupportsProgress bool
	SupportsLogging  bool
	Connected        time.Time
	LastSeen         time.Time
}

// NewNotifier creates a new notification manager
func NewNotifier(queueSize int) *Notifier {
	return &Notifier{
		queue:      make(chan NotificationMessage, queueSize),
		handlers:   make(map[string][]DeliveryHandler),
		clientInfo: make(map[string]*ClientInfo),
	}
}

// Start starts the notifier
func (n *Notifier) Start(ctx context.Context) {
	n.ctx, n.cancel = context.WithCancel(ctx)
	n.wg.Add(1)
	go n.processQueue()
}

// Stop stops the notifier
func (n *Notifier) Stop() {
	if n.cancel != nil {
		n.cancel()
	}
	n.wg.Wait()
	close(n.queue)
}

// RegisterClient registers a new client
func (n *Notifier) RegisterClient(clientID string, supportsProgress, supportsLogging bool) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.clientInfo[clientID] = &ClientInfo{
		ID:               clientID,
		SupportsProgress: supportsProgress,
		SupportsLogging:  supportsLogging,
		Connected:        time.Now(),
		LastSeen:         time.Now(),
	}
}

// UnregisterClient removes a client
func (n *Notifier) UnregisterClient(clientID string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	delete(n.clientInfo, clientID)
}

// RegisterHandler registers a delivery handler for a transport type
func (n *Notifier) RegisterHandler(transportType string, handler DeliveryHandler) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.handlers[transportType] = append(n.handlers[transportType], handler)
}

// SendToClient sends a notification to a specific client
func (n *Notifier) SendToClient(clientID string, method string, params interface{}) error {
	// Marshal params
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	notification := &Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}

	msg := NotificationMessage{
		ClientID:     clientID,
		Notification: notification,
		Timestamp:    time.Now(),
	}

	select {
	case n.queue <- msg:
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("notification queue full")
	}
}

// Broadcast sends a notification to all clients
func (n *Notifier) Broadcast(method string, params interface{}) error {
	n.mutex.RLock()
	clientIDs := make([]string, 0, len(n.clientInfo))
	for clientID := range n.clientInfo {
		clientIDs = append(clientIDs, clientID)
	}
	n.mutex.RUnlock()

	// Send to each client
	var lastErr error
	for _, clientID := range clientIDs {
		if err := n.SendToClient(clientID, method, params); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// NotifyToolsListChanged notifies all clients that the tools list has changed
func (n *Notifier) NotifyToolsListChanged() error {
	return n.Broadcast(NotificationToolsListChanged, &ListChangedParams{})
}

// NotifyResourcesListChanged notifies all clients that the resources list has changed
func (n *Notifier) NotifyResourcesListChanged() error {
	return n.Broadcast(NotificationResourcesListChanged, &ListChangedParams{})
}

// NotifyPromptsListChanged notifies all clients that the prompts list has changed
func (n *Notifier) NotifyPromptsListChanged() error {
	return n.Broadcast(NotificationPromptsListChanged, &ListChangedParams{})
}

// NotifyRootsListChanged notifies all clients that the roots list has changed
func (n *Notifier) NotifyRootsListChanged() error {
	return n.Broadcast(NotificationRootsListChanged, &ListChangedParams{})
}

// NotifyResourceChanged notifies subscribers that a resource has changed
func (n *Notifier) NotifyResourceChanged(uri string) error {
	params := &ResourceChangedParams{
		URI: uri,
	}
	return n.Broadcast(NotificationResourceChanged, params)
}

// SendProgress sends a progress notification to a client
func (n *Notifier) SendProgress(clientID, progressToken string, progress float64, message string) error {
	n.mutex.RLock()
	client, exists := n.clientInfo[clientID]
	n.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("client %s not found", clientID)
	}

	if !client.SupportsProgress {
		return nil // Silent skip if client doesn't support progress
	}

	params := &ProgressParams{
		ProgressToken: progressToken,
		Progress:      progress,
		Message:       message,
	}

	return n.SendToClient(clientID, NotificationProgress, params)
}

// SendLogMessage sends a log message to clients that support logging
func (n *Notifier) SendLogMessage(level, logger, message string, data interface{}) error {
	params := &LogMessageParams{
		Level:   level,
		Logger:  logger,
		Message: message,
		Data:    data,
	}

	n.mutex.RLock()
	clientIDs := make([]string, 0)
	for clientID, client := range n.clientInfo {
		if client.SupportsLogging {
			clientIDs = append(clientIDs, clientID)
		}
	}
	n.mutex.RUnlock()

	// Send to clients that support logging
	var lastErr error
	for _, clientID := range clientIDs {
		if err := n.SendToClient(clientID, NotificationLogMessage, params); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// processQueue processes the notification queue
func (n *Notifier) processQueue() {
	defer n.wg.Done()

	for {
		select {
		case <-n.ctx.Done():
			return
		case msg := <-n.queue:
			n.deliverNotification(msg)
		}
	}
}

// deliverNotification delivers a notification to a client
func (n *Notifier) deliverNotification(msg NotificationMessage) {
	n.mutex.RLock()
	client, exists := n.clientInfo[msg.ClientID]
	handlers := n.handlers // Copy handlers map reference
	n.mutex.RUnlock()

	if !exists {
		return // Client no longer connected
	}

	// Update last seen
	n.mutex.Lock()
	client.LastSeen = time.Now()
	n.mutex.Unlock()

	// Try each handler type
	delivered := false
	for _, handlerList := range handlers {
		for _, handler := range handlerList {
			if err := handler(msg.ClientID, msg.Notification); err == nil {
				delivered = true
				break
			}
		}
		if delivered {
			break
		}
	}

	if !delivered {
		// Log or handle undelivered notification
		fmt.Printf("Failed to deliver notification to client %s: %s\n", msg.ClientID, msg.Notification.Method)
	}
}

// GetClientInfo returns information about a client
func (n *Notifier) GetClientInfo(clientID string) (*ClientInfo, bool) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	info, exists := n.clientInfo[clientID]
	return info, exists
}

// GetConnectedClients returns a list of connected client IDs
func (n *Notifier) GetConnectedClients() []string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	clients := make([]string, 0, len(n.clientInfo))
	for clientID := range n.clientInfo {
		clients = append(clients, clientID)
	}

	return clients
}

// CleanupInactiveClients removes clients that haven't been seen recently
func (n *Notifier) CleanupInactiveClients(maxInactive time.Duration) int {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	cutoff := time.Now().Add(-maxInactive)
	removed := 0

	for clientID, client := range n.clientInfo {
		if client.LastSeen.Before(cutoff) {
			delete(n.clientInfo, clientID)
			removed++
		}
	}

	return removed
}