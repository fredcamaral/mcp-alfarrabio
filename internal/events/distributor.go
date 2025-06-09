// Package events provides event distribution for real-time updates
package events

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"lerian-mcp-memory/internal/push"
)

// EventDistributor distributes events to WebSocket connections and push notifications
type EventDistributor struct {
	eventBus         *EventBus
	eventStore       *EventStore
	metricsCollector *MetricsCollector
	filterEngine     *FilterEngine
	wsManager        WebSocketManager
	pushManager      PushNotificationManager
	config           *DistributorConfig
	subscriptions    map[string]*DistributionSubscription
	ctx              context.Context
	cancel           context.CancelFunc
	mu               sync.RWMutex
	running          bool
	wg               sync.WaitGroup
	eventQueue       chan *DistributionEvent
}

// DistributorConfig configures the event distributor
type DistributorConfig struct {
	QueueSize              int           `json:"queue_size"`
	WorkerCount            int           `json:"worker_count"`
	BatchSize              int           `json:"batch_size"`
	FlushInterval          time.Duration `json:"flush_interval"`
	EnableWebSocket        bool          `json:"enable_websocket"`
	EnablePushNotification bool          `json:"enable_push_notification"`
	EnableFiltering        bool          `json:"enable_filtering"`
	EnablePersistence      bool          `json:"enable_persistence"`
	EnableMetrics          bool          `json:"enable_metrics"`
	RetryAttempts          int           `json:"retry_attempts"`
	RetryDelay             time.Duration `json:"retry_delay"`
	DeduplicationWindow    time.Duration `json:"deduplication_window"`
}

// DistributionSubscription represents a subscription for event distribution
type DistributionSubscription struct {
	ID             string                     `json:"id"`
	SubscriberID   string                     `json:"subscriber_id"`
	SubscriberType SubscriberType             `json:"subscriber_type"`
	Filter         *EventFilter               `json:"filter"`
	DeliveryModes  []DeliveryMode             `json:"delivery_modes"`
	Priority       SubscriptionPriority       `json:"priority"`
	CreatedAt      time.Time                  `json:"created_at"`
	LastActivity   time.Time                  `json:"last_activity"`
	Statistics     *DistributionStats         `json:"statistics"`
	Configuration  *SubscriptionConfiguration `json:"configuration"`
}

// DistributionEvent represents an event ready for distribution
type DistributionEvent struct {
	Event           *Event                          `json:"event"`
	Subscriptions   []*DistributionSubscription     `json:"subscriptions"`
	Priority        EventPriority                   `json:"priority"`
	CreatedAt       time.Time                       `json:"created_at"`
	Attempts        int                             `json:"attempts"`
	LastAttempt     time.Time                       `json:"last_attempt"`
	DeliveryResults map[string]*push.DeliveryResult `json:"delivery_results"`
}

// DistributionStats tracks distribution statistics
type DistributionStats struct {
	EventsReceived      int64         `json:"events_received"`
	EventsDelivered     int64         `json:"events_delivered"`
	EventsFailed        int64         `json:"events_failed"`
	EventsFiltered      int64         `json:"events_filtered"`
	AverageLatency      time.Duration `json:"average_latency"`
	LastDelivery        time.Time     `json:"last_delivery"`
	DeliverySuccessRate float64       `json:"delivery_success_rate"`
	mu                  sync.RWMutex
}

// SubscriberType defines types of subscribers
type SubscriberType string

const (
	SubscriberTypeWebSocket        SubscriberType = "websocket"
	SubscriberTypePushNotification SubscriberType = "push_notification"
	SubscriberTypeCLI              SubscriberType = "cli"
	SubscriberTypeWebhook          SubscriberType = "webhook"
)

// SubscriptionPriority defines subscription priority levels
type SubscriptionPriority int

const (
	SubscriptionPriorityLow      SubscriptionPriority = 1
	SubscriptionPriorityNormal   SubscriptionPriority = 2
	SubscriptionPriorityHigh     SubscriptionPriority = 3
	SubscriptionPriorityCritical SubscriptionPriority = 4
)

// SubscriptionConfiguration configures subscription behavior
type SubscriptionConfiguration struct {
	MaxRetries          int           `json:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay"`
	EnableDeduplication bool          `json:"enable_deduplication"`
	EnableBatching      bool          `json:"enable_batching"`
	BatchSize           int           `json:"batch_size"`
	BatchTimeout        time.Duration `json:"batch_timeout"`
	EnableCompression   bool          `json:"enable_compression"`
	EnableEncryption    bool          `json:"enable_encryption"`
}

// WebSocketManager interface for WebSocket management
type WebSocketManager interface {
	SendToConnection(connectionID string, data []byte) error
	SendToSession(sessionID string, data []byte) error
	BroadcastToAll(data []byte) error
	GetActiveConnections() []string
	GetConnectionsByFilter(filter func(connectionID string) bool) []string
}

// PushNotificationManager interface for push notification management
type PushNotificationManager interface {
	SendNotification(endpointID string, notification *PushNotification) error
	SendBatch(notifications []*BatchNotification) error
	GetActiveEndpoints() []string
}

// PushNotification represents a push notification
type PushNotification struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Title    string                 `json:"title"`
	Message  string                 `json:"message"`
	Data     map[string]interface{} `json:"data"`
	Priority NotificationPriority   `json:"priority"`
}

// BatchNotification represents a notification in a batch
type BatchNotification struct {
	EndpointID   string            `json:"endpoint_id"`
	Notification *PushNotification `json:"notification"`
}

// NotificationPriority defines notification priority levels
type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "low"
	NotificationPriorityNormal   NotificationPriority = "normal"
	NotificationPriorityHigh     NotificationPriority = "high"
	NotificationPriorityCritical NotificationPriority = "critical"
)

// DefaultDistributorConfig returns default distributor configuration
func DefaultDistributorConfig() *DistributorConfig {
	return &DistributorConfig{
		QueueSize:              10000,
		WorkerCount:            5,
		BatchSize:              50,
		FlushInterval:          5 * time.Second,
		EnableWebSocket:        true,
		EnablePushNotification: true,
		EnableFiltering:        true,
		EnablePersistence:      true,
		EnableMetrics:          true,
		RetryAttempts:          3,
		RetryDelay:             time.Second,
		DeduplicationWindow:    5 * time.Second,
	}
}

// NewEventDistributor creates a new event distributor
func NewEventDistributor(config *DistributorConfig) *EventDistributor {
	if config == nil {
		config = DefaultDistributorConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &EventDistributor{
		config:        config,
		subscriptions: make(map[string]*DistributionSubscription),
		ctx:           ctx,
		cancel:        cancel,
		running:       false,
		eventQueue:    make(chan *DistributionEvent, config.QueueSize),
	}
}

// SetEventBus sets the event bus
func (ed *EventDistributor) SetEventBus(eventBus *EventBus) {
	ed.eventBus = eventBus
}

// SetEventStore sets the event store
func (ed *EventDistributor) SetEventStore(eventStore *EventStore) {
	ed.eventStore = eventStore
}

// SetMetricsCollector sets the metrics collector
func (ed *EventDistributor) SetMetricsCollector(metricsCollector *MetricsCollector) {
	ed.metricsCollector = metricsCollector
}

// SetFilterEngine sets the filter engine
func (ed *EventDistributor) SetFilterEngine(filterEngine *FilterEngine) {
	ed.filterEngine = filterEngine
}

// SetWebSocketManager sets the WebSocket manager
func (ed *EventDistributor) SetWebSocketManager(wsManager WebSocketManager) {
	ed.wsManager = wsManager
}

// SetPushNotificationManager sets the push notification manager
func (ed *EventDistributor) SetPushNotificationManager(pushManager PushNotificationManager) {
	ed.pushManager = pushManager
}

// Start starts the event distributor
func (ed *EventDistributor) Start() error {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	if ed.running {
		return errors.New("event distributor already running")
	}

	log.Println("Starting event distributor...")

	// Start worker goroutines
	for i := 0; i < ed.config.WorkerCount; i++ {
		ed.wg.Add(1)
		go ed.distributionWorker(i)
	}

	// Start metrics collection if enabled
	if ed.config.EnableMetrics && ed.metricsCollector != nil {
		ed.wg.Add(1)
		go ed.metricsWorker()
	}

	ed.running = true
	log.Printf("Event distributor started with %d workers", ed.config.WorkerCount)

	return nil
}

// Stop stops the event distributor
func (ed *EventDistributor) Stop() error {
	ed.mu.Lock()
	if !ed.running {
		ed.mu.Unlock()
		return errors.New("event distributor not running")
	}
	ed.running = false
	ed.mu.Unlock()

	log.Println("Stopping event distributor...")

	// Cancel context to signal workers to stop
	ed.cancel()

	// Close event queue
	close(ed.eventQueue)

	// Wait for all workers to finish
	ed.wg.Wait()

	log.Println("Event distributor stopped")
	return nil
}

// IsRunning returns whether the distributor is running
func (ed *EventDistributor) IsRunning() bool {
	ed.mu.RLock()
	defer ed.mu.RUnlock()
	return ed.running
}

// Subscribe creates a new distribution subscription
func (ed *EventDistributor) Subscribe(subscriberID string, subscriberType SubscriberType, filter *EventFilter, deliveryModes []DeliveryMode, priority SubscriptionPriority) (*DistributionSubscription, error) {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	subscription := &DistributionSubscription{
		ID:             generateDistributionSubscriptionID(),
		SubscriberID:   subscriberID,
		SubscriberType: subscriberType,
		Filter:         filter,
		DeliveryModes:  deliveryModes,
		Priority:       priority,
		CreatedAt:      time.Now(),
		LastActivity:   time.Now(),
		Statistics:     &DistributionStats{},
		Configuration: &SubscriptionConfiguration{
			MaxRetries:          ed.config.RetryAttempts,
			RetryDelay:          ed.config.RetryDelay,
			EnableDeduplication: true,
			EnableBatching:      false,
			BatchSize:           1,
			BatchTimeout:        time.Second,
			EnableCompression:   false,
			EnableEncryption:    false,
		},
	}

	ed.subscriptions[subscription.ID] = subscription

	log.Printf("Created distribution subscription %s for subscriber %s (type: %s)",
		subscription.ID, subscriberID, subscriberType)

	return subscription, nil
}

// Unsubscribe removes a distribution subscription
func (ed *EventDistributor) Unsubscribe(subscriptionID string) error {
	ed.mu.Lock()
	defer ed.mu.Unlock()

	subscription, exists := ed.subscriptions[subscriptionID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	delete(ed.subscriptions, subscriptionID)

	log.Printf("Removed distribution subscription %s for subscriber %s",
		subscriptionID, subscription.SubscriberID)

	return nil
}

// DistributeEvent distributes an event to all matching subscribers
func (ed *EventDistributor) DistributeEvent(event *Event) error {
	if !ed.IsRunning() {
		return errors.New("event distributor not running")
	}

	startTime := time.Now()

	// Apply filters if enabled
	if ed.config.EnableFiltering && ed.filterEngine != nil {
		filterResult := ed.filterEngine.ApplyFilters(event)
		if !filterResult.Allowed {
			log.Printf("Event %s filtered out by filter engine", event.ID)
			return nil
		}

		// Use transformed event if transformation was applied
		if filterResult.Transformed != nil {
			event = filterResult.Transformed
		}
	}

	// Find matching subscriptions
	matchingSubscriptions := ed.findMatchingSubscriptions(event)
	if len(matchingSubscriptions) == 0 {
		log.Printf("No matching subscriptions for event %s", event.ID)
		return nil
	}

	// Create distribution event
	distributionEvent := &DistributionEvent{
		Event:           event,
		Subscriptions:   matchingSubscriptions,
		Priority:        event.GetPriority(),
		CreatedAt:       time.Now(),
		Attempts:        0,
		DeliveryResults: make(map[string]*push.DeliveryResult),
	}

	// Queue for distribution
	select {
	case ed.eventQueue <- distributionEvent:
		log.Printf("Queued event %s for distribution to %d subscribers",
			event.ID, len(matchingSubscriptions))
	default:
		// Queue is full, drop event
		log.Printf("Distribution queue full, dropping event %s", event.ID)
		return errors.New("distribution queue full")
	}

	// Record metrics if enabled
	if ed.config.EnableMetrics && ed.metricsCollector != nil {
		ed.metricsCollector.RecordEvent(event, time.Since(startTime), true)
	}

	return nil
}

// GetSubscriptions returns all distribution subscriptions
func (ed *EventDistributor) GetSubscriptions() map[string]*DistributionSubscription {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	// Return a copy
	result := make(map[string]*DistributionSubscription)
	for id, subscription := range ed.subscriptions {
		result[id] = subscription
	}

	return result
}

// GetSubscriptionsBySubscriber returns subscriptions for a specific subscriber
func (ed *EventDistributor) GetSubscriptionsBySubscriber(subscriberID string) []*DistributionSubscription {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	var result []*DistributionSubscription
	for _, subscription := range ed.subscriptions {
		if subscription.SubscriberID == subscriberID {
			result = append(result, subscription)
		}
	}

	return result
}

// GetStatistics returns distribution statistics
func (ed *EventDistributor) GetStatistics() map[string]*DistributionStats {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	result := make(map[string]*DistributionStats)
	for id, subscription := range ed.subscriptions {
		subscription.Statistics.mu.RLock()
		result[id] = &DistributionStats{
			EventsReceived:      subscription.Statistics.EventsReceived,
			EventsDelivered:     subscription.Statistics.EventsDelivered,
			EventsFailed:        subscription.Statistics.EventsFailed,
			EventsFiltered:      subscription.Statistics.EventsFiltered,
			AverageLatency:      subscription.Statistics.AverageLatency,
			LastDelivery:        subscription.Statistics.LastDelivery,
			DeliverySuccessRate: subscription.Statistics.DeliverySuccessRate,
		}
		subscription.Statistics.mu.RUnlock()
	}

	return result
}

// findMatchingSubscriptions finds subscriptions that match an event
func (ed *EventDistributor) findMatchingSubscriptions(event *Event) []*DistributionSubscription {
	ed.mu.RLock()
	defer ed.mu.RUnlock()

	var matching []*DistributionSubscription

	for _, subscription := range ed.subscriptions {
		if subscription.Filter == nil || event.Matches(subscription.Filter) {
			matching = append(matching, subscription)
		}
	}

	// Sort by priority (higher priority first)
	for i := 0; i < len(matching)-1; i++ {
		for j := i + 1; j < len(matching); j++ {
			if matching[i].Priority < matching[j].Priority {
				matching[i], matching[j] = matching[j], matching[i]
			}
		}
	}

	return matching
}

// distributionWorker processes distribution events
func (ed *EventDistributor) distributionWorker(workerID int) {
	defer ed.wg.Done()

	log.Printf("Distribution worker %d started", workerID)

	for {
		select {
		case distributionEvent, ok := <-ed.eventQueue:
			if !ok {
				log.Printf("Distribution worker %d stopped (queue closed)", workerID)
				return
			}

			ed.processDistributionEvent(distributionEvent)

		case <-ed.ctx.Done():
			log.Printf("Distribution worker %d stopped (context cancelled)", workerID)
			return
		}
	}
}

// processDistributionEvent processes a single distribution event
func (ed *EventDistributor) processDistributionEvent(distributionEvent *DistributionEvent) {
	startTime := time.Now()
	distributionEvent.Attempts++
	distributionEvent.LastAttempt = startTime

	for _, subscription := range distributionEvent.Subscriptions {
		ed.deliverToSubscription(distributionEvent.Event, subscription, distributionEvent)
	}

	// Persist event if enabled
	if ed.config.EnablePersistence && ed.eventStore != nil {
		if err := ed.eventStore.Store(distributionEvent.Event); err != nil {
			log.Printf("Failed to persist event %s: %v", distributionEvent.Event.ID, err)
		}
	}

	log.Printf("Processed distribution event %s (took %v, attempt %d)",
		distributionEvent.Event.ID, time.Since(startTime), distributionEvent.Attempts)
}

// deliverToSubscription delivers an event to a specific subscription
func (ed *EventDistributor) deliverToSubscription(event *Event, subscription *DistributionSubscription, distributionEvent *DistributionEvent) {
	startTime := time.Now()

	// Update subscription activity
	subscription.LastActivity = time.Now()
	subscription.Statistics.mu.Lock()
	subscription.Statistics.EventsReceived++
	subscription.Statistics.mu.Unlock()

	deliveryResults := make([]*push.DeliveryResult, 0, len(subscription.DeliveryModes))

	// Deliver via each configured delivery mode
	for _, deliveryMode := range subscription.DeliveryModes {
		result := ed.deliverViaMode(event, subscription, deliveryMode)
		deliveryResults = append(deliveryResults, result)

		// Store result in distribution event
		resultKey := fmt.Sprintf("%s_%s", subscription.ID, deliveryMode)
		distributionEvent.DeliveryResults[resultKey] = result
	}

	// Update subscription statistics
	latency := time.Since(startTime)
	ed.updateSubscriptionStatistics(subscription, deliveryResults, latency)

	// Record metrics for subscriber activity
	if ed.config.EnableMetrics && ed.metricsCollector != nil {
		successful := int64(0)
		failed := int64(0)
		for _, result := range deliveryResults {
			if result.Success {
				successful++
			} else {
				failed++
			}
		}

		ed.metricsCollector.RecordSubscriberActivity(
			subscription.SubscriberID, 1, successful, failed, latency)
	}
}

// deliverViaMode delivers an event via a specific delivery mode
func (ed *EventDistributor) deliverViaMode(event *Event, subscription *DistributionSubscription, mode DeliveryMode) *push.DeliveryResult {
	result := &push.DeliveryResult{
		NotificationID: event.ID,
		EndpointID:     subscription.SubscriberID,
		Timestamp:      time.Now(),
		Attempt:        1,
	}

	switch mode {
	case DeliveryModeWebSocket:
		if ed.config.EnableWebSocket && ed.wsManager != nil {
			result = ed.deliverViaWebSocket(event, subscription)
		} else {
			result.Success = false
			result.Error = "WebSocket delivery not available"
		}

	case DeliveryModePush:
		if ed.config.EnablePushNotification && ed.pushManager != nil {
			result = ed.deliverViaPushNotification(event, subscription)
		} else {
			result.Success = false
			result.Error = "Push notification delivery not available"
		}

	default:
		result.Success = false
		result.Error = fmt.Sprintf("Unsupported delivery mode: %s", mode)
	}

	result.Duration = time.Since(result.Timestamp)
	return result
}

// deliverViaWebSocket delivers an event via WebSocket
func (ed *EventDistributor) deliverViaWebSocket(event *Event, subscription *DistributionSubscription) *push.DeliveryResult {
	result := &push.DeliveryResult{
		NotificationID: event.ID,
		EndpointID:     subscription.SubscriberID,
		Timestamp:      time.Now(),
		Attempt:        1,
	}

	// Convert event to JSON
	eventJSON, err := event.ToJSON()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to marshal event: %v", err)
		return result
	}

	// Send via WebSocket based on subscriber type
	switch subscription.SubscriberType {
	case SubscriberTypeWebSocket:
		err = ed.wsManager.SendToConnection(subscription.SubscriberID, eventJSON)
	default:
		err = ed.wsManager.SendToSession(subscription.SubscriberID, eventJSON)
	}

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("WebSocket delivery failed: %v", err)
	} else {
		result.Success = true
		result.StatusCode = 200
	}

	return result
}

// deliverViaPushNotification delivers an event via push notification
func (ed *EventDistributor) deliverViaPushNotification(event *Event, subscription *DistributionSubscription) *push.DeliveryResult {
	result := &push.DeliveryResult{
		NotificationID: event.ID,
		EndpointID:     subscription.SubscriberID,
		Timestamp:      time.Now(),
		Attempt:        1,
	}

	// Create push notification from event
	notification := ed.createPushNotificationFromEvent(event)

	err := ed.pushManager.SendNotification(subscription.SubscriberID, notification)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Push notification delivery failed: %v", err)
	} else {
		result.Success = true
		result.StatusCode = 200
	}

	return result
}

// createPushNotificationFromEvent creates a push notification from an event
func (ed *EventDistributor) createPushNotificationFromEvent(event *Event) *PushNotification {
	// Map event priority to notification priority
	var notificationPriority NotificationPriority
	switch event.GetPriority() {
	case PriorityCritical:
		notificationPriority = NotificationPriorityCritical
	case PriorityHigh:
		notificationPriority = NotificationPriorityHigh
	case PriorityNormal:
		notificationPriority = NotificationPriorityNormal
	default:
		notificationPriority = NotificationPriorityLow
	}

	// Create notification title and message based on event type
	title := fmt.Sprintf("Event: %s", event.Type)
	message := fmt.Sprintf("Action: %s from %s", event.Action, event.Source)

	return &PushNotification{
		ID:       event.ID,
		Type:     string(event.Type),
		Title:    title,
		Message:  message,
		Data:     event.Metadata,
		Priority: notificationPriority,
	}
}

// updateSubscriptionStatistics updates statistics for a subscription
func (ed *EventDistributor) updateSubscriptionStatistics(subscription *DistributionSubscription, results []*push.DeliveryResult, latency time.Duration) {
	subscription.Statistics.mu.Lock()
	defer subscription.Statistics.mu.Unlock()

	successCount := int64(0)
	failureCount := int64(0)

	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	subscription.Statistics.EventsDelivered += successCount
	subscription.Statistics.EventsFailed += failureCount
	subscription.Statistics.LastDelivery = time.Now()

	// Update average latency
	if subscription.Statistics.AverageLatency == 0 {
		subscription.Statistics.AverageLatency = latency
	} else {
		subscription.Statistics.AverageLatency = time.Duration(
			int64(subscription.Statistics.AverageLatency)*9/10 + int64(latency)/10,
		)
	}

	// Calculate delivery success rate
	totalDeliveries := subscription.Statistics.EventsDelivered + subscription.Statistics.EventsFailed
	if totalDeliveries > 0 {
		subscription.Statistics.DeliverySuccessRate = float64(subscription.Statistics.EventsDelivered) / float64(totalDeliveries) * 100
	}
}

// metricsWorker collects and reports metrics
func (ed *EventDistributor) metricsWorker() {
	defer ed.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ed.collectMetrics()
		case <-ed.ctx.Done():
			return
		}
	}
}

// collectMetrics collects distribution metrics
func (ed *EventDistributor) collectMetrics() {
	ed.mu.RLock()
	subscriptionCount := len(ed.subscriptions)
	queueLength := len(ed.eventQueue)
	ed.mu.RUnlock()

	log.Printf("Distribution metrics: subscriptions=%d, queue_length=%d",
		subscriptionCount, queueLength)
}

// GetStatus returns the current status of the distributor
func (ed *EventDistributor) GetStatus() map[string]interface{} {
	ed.mu.RLock()
	subscriptionCount := len(ed.subscriptions)
	queueLength := len(ed.eventQueue)
	queueCapacity := cap(ed.eventQueue)
	ed.mu.RUnlock()

	return map[string]interface{}{
		"running":             ed.IsRunning(),
		"worker_count":        ed.config.WorkerCount,
		"subscription_count":  subscriptionCount,
		"queue_length":        queueLength,
		"queue_capacity":      queueCapacity,
		"queue_utilization":   float64(queueLength) / float64(queueCapacity) * 100,
		"websocket_enabled":   ed.config.EnableWebSocket,
		"push_enabled":        ed.config.EnablePushNotification,
		"filtering_enabled":   ed.config.EnableFiltering,
		"persistence_enabled": ed.config.EnablePersistence,
		"metrics_enabled":     ed.config.EnableMetrics,
	}
}

// generateDistributionSubscriptionID generates a unique subscription ID
func generateDistributionSubscriptionID() string {
	return fmt.Sprintf("dist_sub_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
}
