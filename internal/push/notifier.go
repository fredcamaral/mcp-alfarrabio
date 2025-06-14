// Package push provides the main push notification service with multiple notification providers
package push

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"lerian-mcp-memory/internal/websocket"
)

// NotificationService provides the main push notification service interface
type NotificationService struct {
	registry      *Registry
	dispatcher    *Dispatcher
	healthChecker *HealthChecker
	queue         *NotificationQueue
	config        *ServiceConfig
	ctx           context.Context
	cancel        context.CancelFunc
	running       bool
	mu            sync.RWMutex
	metrics       *ServiceMetrics
}

// ServiceConfig configures the notification service
type ServiceConfig struct {
	Registry      *RegistryConfig    `json:"registry"`
	Dispatcher    *DispatcherConfig  `json:"dispatcher"`
	HealthChecker *HealthCheckConfig `json:"health_checker"`
	Queue         *QueueConfig       `json:"queue"`
	AutoStart     bool               `json:"auto_start"`
	MetricsPort   int                `json:"metrics_port"`
}

// RegistryConfig is an alias for convenience
type RegistryConfig struct {
	MaxInactiveAge time.Duration `json:"max_inactive_age"`
}

// ServiceMetrics tracks overall service performance
type ServiceMetrics struct {
	StartTime            time.Time     `json:"start_time"`
	Uptime               time.Duration `json:"uptime"`
	TotalNotifications   int64         `json:"total_notifications"`
	SuccessfulDeliveries int64         `json:"successful_deliveries"`
	FailedDeliveries     int64         `json:"failed_deliveries"`
	ActiveEndpoints      int           `json:"active_endpoints"`
	HealthyEndpoints     int           `json:"healthy_endpoints"`
	mu                   sync.RWMutex
}

// DefaultServiceConfig returns default service configuration
func DefaultServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		Registry: &RegistryConfig{
			MaxInactiveAge: 10 * time.Minute,
		},
		Dispatcher:    DefaultDispatcherConfig(),
		HealthChecker: DefaultHealthCheckConfig(),
		Queue:         DefaultQueueConfig(),
		AutoStart:     true,
		MetricsPort:   0, // Disabled by default
	}
}

// NewNotificationService creates a new push notification service
func NewNotificationService(config *ServiceConfig) *NotificationService {
	if config == nil {
		config = DefaultServiceConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create registry
	registry := NewRegistry()

	// Create dispatcher
	dispatcher := NewDispatcher(registry, config.Dispatcher)

	// Create health checker
	healthChecker := NewHealthChecker(registry, config.HealthChecker)

	// Create queue
	queue := NewNotificationQueue(dispatcher, registry, config.Queue)

	service := &NotificationService{
		registry:      registry,
		dispatcher:    dispatcher,
		healthChecker: healthChecker,
		queue:         queue,
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		running:       false,
		metrics:       &ServiceMetrics{},
	}

	return service
}

// Start starts all components of the notification service
func (ns *NotificationService) Start() error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if ns.running {
		return errors.New("notification service already running")
	}

	log.Println("Starting push notification service...")

	// Start dispatcher
	if err := ns.dispatcher.Start(); err != nil {
		return fmt.Errorf("failed to start dispatcher: %w", err)
	}

	// Start health checker
	if err := ns.healthChecker.Start(); err != nil {
		if err := ns.dispatcher.Stop(); err != nil {
			log.Printf("Failed to stop dispatcher during cleanup: %v", err)
		}
		return fmt.Errorf("failed to start health checker: %w", err)
	}

	// Start queue
	if err := ns.queue.Start(); err != nil {
		if err := ns.dispatcher.Stop(); err != nil {
			log.Printf("Failed to stop dispatcher during cleanup: %v", err)
		}
		if err := ns.healthChecker.Stop(); err != nil {
			log.Printf("Failed to stop health checker during cleanup: %v", err)
		}
		return fmt.Errorf("failed to start queue: %w", err)
	}

	// Initialize metrics
	ns.metrics.StartTime = time.Now()
	ns.running = true

	log.Println("Push notification service started successfully")
	return nil
}

// Stop stops all components of the notification service
func (ns *NotificationService) Stop() error {
	ns.mu.Lock()
	if !ns.running {
		ns.mu.Unlock()
		return errors.New("notification service not running")
	}
	ns.running = false
	ns.mu.Unlock()

	log.Println("Stopping push notification service...")

	// Stop components in reverse order
	var stopErrors []error

	if err := ns.queue.Stop(); err != nil {
		stopErrors = append(stopErrors, fmt.Errorf("queue stop error: %w", err))
	}

	if err := ns.healthChecker.Stop(); err != nil {
		stopErrors = append(stopErrors, fmt.Errorf("health checker stop error: %w", err))
	}

	if err := ns.dispatcher.Stop(); err != nil {
		stopErrors = append(stopErrors, fmt.Errorf("dispatcher stop error: %w", err))
	}

	// Stop registry cleanup
	ns.registry.Stop()

	// Cancel context
	ns.cancel()

	if len(stopErrors) > 0 {
		log.Printf("Errors during shutdown: %v", stopErrors)
		return fmt.Errorf("shutdown errors: %v", stopErrors)
	}

	log.Println("Push notification service stopped successfully")
	return nil
}

// IsRunning returns whether the service is running
func (ns *NotificationService) IsRunning() bool {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.running
}

// RegisterEndpoint registers a new CLI endpoint for push notifications
func (ns *NotificationService) RegisterEndpoint(endpoint *CLIEndpoint) error {
	if !ns.IsRunning() {
		return errors.New("notification service not running")
	}

	return ns.registry.Register(endpoint)
}

// DeregisterEndpoint removes a CLI endpoint from push notifications
func (ns *NotificationService) DeregisterEndpoint(endpointID string) error {
	if !ns.IsRunning() {
		return errors.New("notification service not running")
	}

	return ns.registry.Deregister(endpointID)
}

// SendNotification sends a push notification to all active endpoints
func (ns *NotificationService) SendNotification(notification *Notification) error {
	if !ns.IsRunning() {
		return errors.New("notification service not running")
	}

	// Update metrics
	ns.updateMetrics(func(m *ServiceMetrics) {
		m.TotalNotifications++
	})

	// Queue the notification for processing
	return ns.queue.Enqueue(notification)
}

// SendNotificationToEndpoint sends a push notification to a specific endpoint
func (ns *NotificationService) SendNotificationToEndpoint(notification *Notification, endpointID string) error {
	if !ns.IsRunning() {
		return errors.New("notification service not running")
	}

	// Set target endpoint
	notification.TargetID = endpointID

	// Update metrics
	ns.updateMetrics(func(m *ServiceMetrics) {
		m.TotalNotifications++
	})

	// Send directly through dispatcher
	return ns.dispatcher.DispatchToEndpoint(notification, endpointID)
}

// SendBatchNotifications sends multiple notifications efficiently
func (ns *NotificationService) SendBatchNotifications(notifications []*Notification) []error {
	if !ns.IsRunning() {
		errorList := make([]error, len(notifications))
		serviceErr := errors.New("notification service not running")
		for i := range errorList {
			errorList[i] = serviceErr
		}
		return errorList
	}

	// Update metrics
	ns.updateMetrics(func(m *ServiceMetrics) {
		m.TotalNotifications += int64(len(notifications))
	})

	// Queue all notifications
	return ns.queue.EnqueueBatch(notifications)
}

// GetEndpoints returns all registered endpoints
func (ns *NotificationService) GetEndpoints() []*CLIEndpoint {
	return ns.registry.GetAll()
}

// GetActiveEndpoints returns all active and healthy endpoints
func (ns *NotificationService) GetActiveEndpoints() []*CLIEndpoint {
	return ns.registry.GetActive()
}

// GetEndpoint returns a specific endpoint by ID
func (ns *NotificationService) GetEndpoint(endpointID string) (*CLIEndpoint, bool) {
	return ns.registry.Get(endpointID)
}

// CheckEndpointHealth performs an immediate health check for an endpoint
func (ns *NotificationService) CheckEndpointHealth(endpointID string) (*HealthCheckResult, error) {
	if !ns.IsRunning() {
		return nil, errors.New("notification service not running")
	}

	return ns.healthChecker.CheckEndpoint(endpointID)
}

// GetServiceStatus returns comprehensive service status
func (ns *NotificationService) GetServiceStatus() map[string]interface{} {
	metrics := ns.GetMetrics()
	registryMetrics := ns.registry.GetMetrics()
	dispatcherAnalytics := ns.dispatcher.GetAnalytics()
	healthMetrics := ns.healthChecker.GetMetrics()
	queueMetrics := ns.queue.GetMetrics()

	return map[string]interface{}{
		"running":    ns.IsRunning(),
		"uptime":     metrics.Uptime.String(),
		"start_time": metrics.StartTime,
		"components": map[string]interface{}{
			"registry": map[string]interface{}{
				"total_endpoints":    registryMetrics.TotalEndpoints,
				"active_endpoints":   registryMetrics.ActiveEndpoints,
				"registration_count": registryMetrics.RegistrationCount,
			},
			"dispatcher": map[string]interface{}{
				"running":               ns.dispatcher.IsRunning(),
				"total_deliveries":      dispatcherAnalytics.TotalDeliveries,
				"successful_deliveries": dispatcherAnalytics.SuccessfulDeliveries,
				"error_rate":            dispatcherAnalytics.ErrorRate,
				"average_latency":       dispatcherAnalytics.AverageLatency.String(),
			},
			"health_checker": map[string]interface{}{
				"running":           ns.healthChecker.IsRunning(),
				"total_checks":      healthMetrics.TotalChecks,
				"healthy_endpoints": healthMetrics.HealthyEndpoints,
				"error_rate":        healthMetrics.ErrorRate,
			},
			"queue": map[string]interface{}{
				"running":            ns.queue.IsRunning(),
				"current_queue_size": queueMetrics.CurrentQueueSize,
				"total_processed":    queueMetrics.TotalProcessed,
				"retry_queue_size":   queueMetrics.RetryQueueSize,
			},
		},
		"metrics": metrics,
	}
}

// GetMetrics returns service metrics
func (ns *NotificationService) GetMetrics() *ServiceMetrics {
	ns.metrics.mu.RLock()
	defer ns.metrics.mu.RUnlock()

	// Calculate current uptime
	uptime := time.Duration(0)
	if !ns.metrics.StartTime.IsZero() {
		uptime = time.Since(ns.metrics.StartTime)
	}

	// Get current endpoint counts
	registryMetrics := ns.registry.GetMetrics()
	healthMetrics := ns.healthChecker.GetMetrics()

	// Return a copy with updated values
	return &ServiceMetrics{
		StartTime:            ns.metrics.StartTime,
		Uptime:               uptime,
		TotalNotifications:   ns.metrics.TotalNotifications,
		SuccessfulDeliveries: ns.metrics.SuccessfulDeliveries,
		FailedDeliveries:     ns.metrics.FailedDeliveries,
		ActiveEndpoints:      registryMetrics.ActiveEndpoints,
		HealthyEndpoints:     healthMetrics.HealthyEndpoints,
	}
}

// updateMetrics safely updates service metrics
func (ns *NotificationService) updateMetrics(updateFunc func(*ServiceMetrics)) {
	ns.metrics.mu.Lock()
	defer ns.metrics.mu.Unlock()
	updateFunc(ns.metrics)
}

// CreateMemoryUpdateNotification creates a notification for memory updates
func (ns *NotificationService) CreateMemoryUpdateNotification(chunkID, repository, sessionID string, content interface{}) *Notification {
	return CreateNotification("memory_update", map[string]interface{}{
		"chunk_id":   chunkID,
		"repository": repository,
		"session_id": sessionID,
		"content":    content,
		"timestamp":  time.Now(),
	})
}

// CreateTaskUpdateNotification creates a notification for task updates
func (ns *NotificationService) CreateTaskUpdateNotification(taskID, status string, metadata map[string]interface{}) *Notification {
	payload := map[string]interface{}{
		"task_id":   taskID,
		"status":    status,
		"timestamp": time.Now(),
	}

	// Add metadata if provided
	for key, value := range metadata {
		payload[key] = value
	}

	return CreateNotification("task_update", payload)
}

// CreateSystemAlertNotification creates a notification for system alerts
func (ns *NotificationService) CreateSystemAlertNotification(alertType, message, severity string) *Notification {
	notification := CreateNotification("system_alert", map[string]interface{}{
		"alert_type": alertType,
		"message":    message,
		"severity":   severity,
		"timestamp":  time.Now(),
	})

	// Set priority based on severity
	switch severity {
	case "critical":
		notification.Priority = PriorityCritical
	case "high":
		notification.Priority = PriorityHigh
	case "low":
		notification.Priority = PriorityLow
	default:
		notification.Priority = PriorityNormal
	}

	return notification
}

// BroadcastMemoryUpdate broadcasts a memory update to all active endpoints
func (ns *NotificationService) BroadcastMemoryUpdate(chunkID, repository, sessionID string, content interface{}) error {
	notification := ns.CreateMemoryUpdateNotification(chunkID, repository, sessionID, content)
	return ns.SendNotification(notification)
}

// BroadcastTaskUpdate broadcasts a task update to all active endpoints
func (ns *NotificationService) BroadcastTaskUpdate(taskID, status string, metadata map[string]interface{}) error {
	notification := ns.CreateTaskUpdateNotification(taskID, status, metadata)
	return ns.SendNotification(notification)
}

// BroadcastSystemAlert broadcasts a system alert to all active endpoints
func (ns *NotificationService) BroadcastSystemAlert(alertType, message, severity string) error {
	notification := ns.CreateSystemAlertNotification(alertType, message, severity)
	return ns.SendNotification(notification)
}

// FlushQueue processes all pending notifications immediately
func (ns *NotificationService) FlushQueue() {
	if ns.IsRunning() {
		ns.queue.Flush()
	}
}

// ForceHealthCheck triggers an immediate health check cycle
func (ns *NotificationService) ForceHealthCheck() {
	if ns.IsRunning() {
		ns.healthChecker.ForceHealthCheck()
	}
}

// GetDeliveryResults returns delivery results for a specific notification
func (ns *NotificationService) GetDeliveryResults(notificationID string) ([]*DeliveryResult, bool) {
	return ns.dispatcher.GetDeliveryResults(notificationID)
}

// GetComponentHealth returns health status of all service components
func (ns *NotificationService) GetComponentHealth() map[string]interface{} {
	return map[string]interface{}{
		"service":        ns.IsRunning(),
		"dispatcher":     ns.dispatcher.IsRunning(),
		"health_checker": ns.healthChecker.IsRunning(),
		"queue":          ns.queue.IsRunning(),
		"registry":       true, // Registry doesn't have a running state
	}
}

// Notifier interface defines the contract for notification providers
type Notifier interface {
	// Send sends a notification and returns the result
	Send(ctx context.Context, notification *Notification, endpoint *CLIEndpoint) (*DeliveryResult, error)

	// GetType returns the notifier type
	GetType() string

	// IsHealthy checks if the notifier is healthy
	IsHealthy() bool

	// GetMetrics returns notifier-specific metrics
	GetMetrics() map[string]interface{}
}

// InMemoryNotifier provides in-memory notification for development/testing
type InMemoryNotifier struct {
	mu            sync.RWMutex
	notifications []NotificationRecord
	maxRecords    int
	metrics       NotifierMetrics
}

// NotificationRecord stores a notification with delivery status
type NotificationRecord struct {
	Notification *Notification   `json:"notification"`
	Endpoint     *CLIEndpoint    `json:"endpoint"`
	DeliveryTime time.Time       `json:"delivery_time"`
	Success      bool            `json:"success"`
	Error        string          `json:"error,omitempty"`
	Result       *DeliveryResult `json:"result"`
}

// NotifierMetrics tracks notifier performance
type NotifierMetrics struct {
	TotalSent    int64         `json:"total_sent"`
	SuccessCount int64         `json:"success_count"`
	FailureCount int64         `json:"failure_count"`
	AverageTime  time.Duration `json:"average_time"`
	LastSent     time.Time     `json:"last_sent"`
	Healthy      bool          `json:"healthy"`
}

// NewInMemoryNotifier creates a new in-memory notifier
func NewInMemoryNotifier(maxRecords int) *InMemoryNotifier {
	if maxRecords <= 0 {
		maxRecords = 1000
	}
	return &InMemoryNotifier{
		notifications: make([]NotificationRecord, 0, maxRecords),
		maxRecords:    maxRecords,
		metrics: NotifierMetrics{
			Healthy: true,
		},
	}
}

// Send sends a notification to the in-memory store
func (n *InMemoryNotifier) Send(ctx context.Context, notification *Notification, endpoint *CLIEndpoint) (*DeliveryResult, error) {
	startTime := time.Now()

	// Simulate processing time
	select {
	case <-time.After(10 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	// Create delivery result
	result := &DeliveryResult{
		NotificationID: notification.ID,
		EndpointID:     endpoint.ID,
		Success:        true,
		StatusCode:     200,
		Response:       "Notification stored in memory",
		Duration:       time.Since(startTime),
		Timestamp:      time.Now(),
		Attempt:        notification.Attempts,
	}

	// Store notification record
	record := NotificationRecord{
		Notification: notification,
		Endpoint:     endpoint,
		DeliveryTime: time.Now(),
		Success:      true,
		Result:       result,
	}

	// Maintain max records limit
	if len(n.notifications) >= n.maxRecords {
		n.notifications = n.notifications[1:]
	}
	n.notifications = append(n.notifications, record)

	// Update metrics
	n.metrics.TotalSent++
	n.metrics.SuccessCount++
	n.metrics.LastSent = time.Now()
	n.updateAverageTime(result.Duration)

	log.Printf("InMemoryNotifier: Stored notification %s for endpoint %s", notification.ID, endpoint.ID)

	return result, nil
}

// GetType returns the notifier type
func (n *InMemoryNotifier) GetType() string {
	return "in-memory"
}

// IsHealthy checks if the notifier is healthy
func (n *InMemoryNotifier) IsHealthy() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.metrics.Healthy
}

// GetMetrics returns notifier metrics
func (n *InMemoryNotifier) GetMetrics() map[string]interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return map[string]interface{}{
		"type":          n.GetType(),
		"total_sent":    n.metrics.TotalSent,
		"success_count": n.metrics.SuccessCount,
		"failure_count": n.metrics.FailureCount,
		"average_time":  n.metrics.AverageTime.String(),
		"last_sent":     n.metrics.LastSent,
		"healthy":       n.metrics.Healthy,
		"stored_count":  len(n.notifications),
		"max_records":   n.maxRecords,
	}
}

// GetNotifications returns stored notifications (for testing/debugging)
func (n *InMemoryNotifier) GetNotifications() []NotificationRecord {
	n.mu.RLock()
	defer n.mu.RUnlock()

	records := make([]NotificationRecord, len(n.notifications))
	copy(records, n.notifications)
	return records
}

// updateAverageTime updates the average processing time
func (n *InMemoryNotifier) updateAverageTime(duration time.Duration) {
	if n.metrics.AverageTime == 0 {
		n.metrics.AverageTime = duration
	} else {
		// Weighted average with 90% weight on previous average
		n.metrics.AverageTime = time.Duration(
			int64(n.metrics.AverageTime)*9/10 + int64(duration)/10,
		)
	}
}

// WebSocketNotifier sends notifications via WebSocket
type WebSocketNotifier struct {
	server  *websocket.Server
	metrics NotifierMetrics
	mu      sync.RWMutex
}

// NewWebSocketNotifier creates a new WebSocket notifier
func NewWebSocketNotifier(server *websocket.Server) *WebSocketNotifier {
	return &WebSocketNotifier{
		server: server,
		metrics: NotifierMetrics{
			Healthy: true,
		},
	}
}

// Send sends a notification via WebSocket
func (n *WebSocketNotifier) Send(ctx context.Context, notification *Notification, endpoint *CLIEndpoint) (*DeliveryResult, error) {
	startTime := time.Now()

	// Check if server is running
	if !n.server.IsRunning() {
		return nil, errors.New("WebSocket server not running")
	}

	// Convert notification to memory event
	event := &websocket.MemoryEvent{
		Type:      "push_notification",
		Action:    notification.Type,
		Timestamp: notification.CreatedAt,
		Data: map[string]interface{}{
			"notification_id": notification.ID,
			"priority":        notification.Priority,
			"payload":         notification.Payload,
			"metadata":        notification.Metadata,
			"target_id":       notification.TargetID,
		},
	}

	// Extract repository and session from notification metadata
	if repo, ok := notification.Metadata["repository"]; ok {
		event.Repository = repo
	}
	if session, ok := notification.Metadata["session_id"]; ok {
		event.SessionID = session
	}

	// Broadcast event
	n.server.BroadcastEvent(event)

	duration := time.Since(startTime)

	// Update metrics
	n.mu.Lock()
	n.metrics.TotalSent++
	n.metrics.SuccessCount++
	n.metrics.LastSent = time.Now()
	n.updateAverageTime(duration)
	n.mu.Unlock()

	// Create delivery result
	result := &DeliveryResult{
		NotificationID: notification.ID,
		EndpointID:     endpoint.ID,
		Success:        true,
		StatusCode:     200,
		Response:       "Notification sent via WebSocket",
		Duration:       duration,
		Timestamp:      time.Now(),
		Attempt:        notification.Attempts,
	}

	log.Printf("WebSocketNotifier: Sent notification %s via WebSocket", notification.ID)

	return result, nil
}

// GetType returns the notifier type
func (n *WebSocketNotifier) GetType() string {
	return "websocket"
}

// IsHealthy checks if the notifier is healthy
func (n *WebSocketNotifier) IsHealthy() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.metrics.Healthy && n.server.IsRunning()
}

// GetMetrics returns notifier metrics
func (n *WebSocketNotifier) GetMetrics() map[string]interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return map[string]interface{}{
		"type":              n.GetType(),
		"total_sent":        n.metrics.TotalSent,
		"success_count":     n.metrics.SuccessCount,
		"failure_count":     n.metrics.FailureCount,
		"average_time":      n.metrics.AverageTime.String(),
		"last_sent":         n.metrics.LastSent,
		"healthy":           n.metrics.Healthy,
		"server_running":    n.server.IsRunning(),
		"connected_clients": n.server.GetConnectionCount(),
	}
}

// updateAverageTime updates the average processing time
func (n *WebSocketNotifier) updateAverageTime(duration time.Duration) {
	if n.metrics.AverageTime == 0 {
		n.metrics.AverageTime = duration
	} else {
		n.metrics.AverageTime = time.Duration(
			int64(n.metrics.AverageTime)*9/10 + int64(duration)/10,
		)
	}
}

// MultiNotifier sends notifications to multiple providers
type MultiNotifier struct {
	notifiers []Notifier
	mu        sync.RWMutex
	metrics   MultiNotifierMetrics
}

// MultiNotifierMetrics tracks multi-notifier performance
type MultiNotifierMetrics struct {
	TotalNotifiers int                               `json:"total_notifiers"`
	HealthyCount   int                               `json:"healthy_count"`
	MetricsByType  map[string]map[string]interface{} `json:"metrics_by_type"`
}

// NewMultiNotifier creates a new multi-notifier
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{
		notifiers: notifiers,
		metrics: MultiNotifierMetrics{
			MetricsByType: make(map[string]map[string]interface{}),
		},
	}
}

// Send sends a notification to all configured notifiers
func (m *MultiNotifier) Send(ctx context.Context, notification *Notification, endpoint *CLIEndpoint) (*DeliveryResult, error) {
	startTime := time.Now()

	m.mu.RLock()
	notifiers := make([]Notifier, len(m.notifiers))
	copy(notifiers, m.notifiers)
	m.mu.RUnlock()

	if len(notifiers) == 0 {
		return nil, errors.New("no notifiers configured")
	}

	var successCount int
	var lastError error

	// Send to all notifiers concurrently
	resultChan := make(chan *DeliveryResult, len(notifiers))
	errorChan := make(chan error, len(notifiers))

	for _, notifier := range notifiers {
		go func(n Notifier) {
			result, err := n.Send(ctx, notification, endpoint)
			if err != nil {
				errorChan <- err
			} else {
				resultChan <- result
			}
		}(notifier)
	}

	// Wait for all results
	for i := 0; i < len(notifiers); i++ {
		select {
		case result := <-resultChan:
			if result.Success {
				successCount++
			}
		case err := <-errorChan:
			lastError = err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Create aggregated result
	successful := successCount > 0
	response := fmt.Sprintf("Sent to %d/%d notifiers", successCount, len(notifiers))
	if lastError != nil {
		response += fmt.Sprintf(" (last error: %v)", lastError)
	}

	result := &DeliveryResult{
		NotificationID: notification.ID,
		EndpointID:     endpoint.ID,
		Success:        successful,
		StatusCode:     200,
		Response:       response,
		Duration:       time.Since(startTime),
		Timestamp:      time.Now(),
		Attempt:        notification.Attempts,
	}

	return result, nil
}

// GetType returns the notifier type
func (m *MultiNotifier) GetType() string {
	return "multi"
}

// IsHealthy checks if any notifier is healthy
func (m *MultiNotifier) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, notifier := range m.notifiers {
		if notifier.IsHealthy() {
			return true
		}
	}
	return false
}

// GetMetrics returns aggregated metrics from all notifiers
func (m *MultiNotifier) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	healthyCount := 0
	metricsByType := make(map[string]map[string]interface{})

	for _, notifier := range m.notifiers {
		if notifier.IsHealthy() {
			healthyCount++
		}
		metricsByType[notifier.GetType()] = notifier.GetMetrics()
	}

	return map[string]interface{}{
		"type":            m.GetType(),
		"total_notifiers": len(m.notifiers),
		"healthy_count":   healthyCount,
		"metrics_by_type": metricsByType,
	}
}

// AddNotifier adds a notifier to the multi-notifier
func (m *MultiNotifier) AddNotifier(notifier Notifier) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifiers = append(m.notifiers, notifier)
}

// RemoveNotifier removes a notifier by type
func (m *MultiNotifier) RemoveNotifier(notifierType string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, notifier := range m.notifiers {
		if notifier.GetType() == notifierType {
			m.notifiers = append(m.notifiers[:i], m.notifiers[i+1:]...)
			return true
		}
	}
	return false
}

// HTTPNotifier sends notifications via HTTP POST (placeholder for future implementation)
type HTTPNotifier struct {
	// TODO: Implement HTTP-based notification delivery
}

// EmailNotifier sends notifications via email (placeholder for future implementation)
type EmailNotifier struct {
	// TODO: Implement email-based notification delivery
}

// WebhookNotifier sends notifications via webhooks (placeholder for future implementation)
type WebhookNotifier struct {
	// TODO: Implement webhook-based notification delivery
}
