// Package events provides event bus for distributing real-time updates
package events

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// EventBus manages event distribution using pub/sub pattern
type EventBus struct {
	subscribers map[string][]*Subscription
	channels    map[string]chan *Event
	metrics     *BusMetrics
	config      *BusConfig
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	running     bool
	wg          sync.WaitGroup
}

// Subscription represents an event subscription
type Subscription struct {
	ID           string
	SubscriberID string
	Filter       *EventFilter
	Channel      chan *Event
	CreatedAt    time.Time
	LastEvent    *time.Time
	DeliveryMode DeliveryMode
	Statistics   *SubscriptionStats
	mu           sync.RWMutex
}

// BusConfig configures the event bus
type BusConfig struct {
	ChannelBufferSize   int           `json:"channel_buffer_size"`
	MaxSubscribers      int           `json:"max_subscribers"`
	EventTTL            time.Duration `json:"event_ttl"`
	CleanupInterval     time.Duration `json:"cleanup_interval"`
	MetricsInterval     time.Duration `json:"metrics_interval"`
	DeduplicationWindow time.Duration `json:"deduplication_window"`
	MaxEventSize        int           `json:"max_event_size"`
	EnablePersistence   bool          `json:"enable_persistence"`
	PersistenceBuffer   int           `json:"persistence_buffer"`
}

// BusMetrics tracks event bus performance
type BusMetrics struct {
	EventsPublished     int64         `json:"events_published"`
	EventsDelivered     int64         `json:"events_delivered"`
	EventsDropped       int64         `json:"events_dropped"`
	EventsDuplicated    int64         `json:"events_duplicated"`
	ActiveSubscriptions int           `json:"active_subscriptions"`
	AverageLatency      time.Duration `json:"average_latency"`
	ThroughputPerSecond float64       `json:"throughput_per_second"`
	LastEventTime       time.Time     `json:"last_event_time"`
	mu                  sync.RWMutex
}

// DefaultBusConfig returns default event bus configuration
func DefaultBusConfig() *BusConfig {
	return &BusConfig{
		ChannelBufferSize:   1000,
		MaxSubscribers:      100,
		EventTTL:            10 * time.Minute,
		CleanupInterval:     time.Minute,
		MetricsInterval:     30 * time.Second,
		DeduplicationWindow: 5 * time.Second,
		MaxEventSize:        1024 * 1024, // 1MB
		EnablePersistence:   true,
		PersistenceBuffer:   10000,
	}
}

// NewEventBus creates a new event bus
func NewEventBus(config *BusConfig) *EventBus {
	if config == nil {
		config = DefaultBusConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &EventBus{
		subscribers: make(map[string][]*Subscription),
		channels:    make(map[string]chan *Event),
		metrics:     &BusMetrics{},
		config:      config,
		ctx:         ctx,
		cancel:      cancel,
		running:     false,
	}
}

// Start starts the event bus
func (eb *EventBus) Start() error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.running {
		return errors.New("event bus already running")
	}

	log.Println("Starting event bus...")

	// Start cleanup routine
	eb.wg.Add(1)
	go eb.cleanupRoutine()

	// Start metrics routine
	eb.wg.Add(1)
	go eb.metricsRoutine()

	eb.running = true
	log.Printf("Event bus started with config: buffer=%d, max_subscribers=%d, ttl=%v",
		eb.config.ChannelBufferSize, eb.config.MaxSubscribers, eb.config.EventTTL)

	return nil
}

// Stop stops the event bus gracefully
func (eb *EventBus) Stop() error {
	eb.mu.Lock()
	if !eb.running {
		eb.mu.Unlock()
		return errors.New("event bus not running")
	}
	eb.running = false
	eb.mu.Unlock()

	log.Println("Stopping event bus...")

	// Cancel context to signal routines to stop
	eb.cancel()

	// Close all subscription channels
	eb.mu.Lock()
	for _, subscriptions := range eb.subscribers {
		for _, sub := range subscriptions {
			close(sub.Channel)
		}
	}
	eb.subscribers = make(map[string][]*Subscription)
	eb.mu.Unlock()

	// Wait for all routines to finish
	eb.wg.Wait()

	log.Println("Event bus stopped")
	return nil
}

// IsRunning returns whether the event bus is running
func (eb *EventBus) IsRunning() bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return eb.running
}

// Subscribe creates a new event subscription
func (eb *EventBus) Subscribe(subscriberID string, filter *EventFilter, deliveryMode DeliveryMode) (*Subscription, error) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if !eb.running {
		return nil, errors.New("event bus not running")
	}

	// Check subscriber limits
	totalSubscriptions := 0
	for _, subscriptions := range eb.subscribers {
		totalSubscriptions += len(subscriptions)
	}

	if totalSubscriptions >= eb.config.MaxSubscribers {
		return nil, fmt.Errorf("maximum subscribers reached: %d", eb.config.MaxSubscribers)
	}

	// Create subscription
	subscription := &Subscription{
		ID:           generateSubscriptionID(),
		SubscriberID: subscriberID,
		Filter:       filter,
		Channel:      make(chan *Event, eb.config.ChannelBufferSize),
		CreatedAt:    time.Now(),
		DeliveryMode: deliveryMode,
		Statistics:   &SubscriptionStats{},
	}

	// Add to subscribers map
	if eb.subscribers[subscriberID] == nil {
		eb.subscribers[subscriberID] = make([]*Subscription, 0)
	}
	eb.subscribers[subscriberID] = append(eb.subscribers[subscriberID], subscription)

	log.Printf("Created subscription %s for subscriber %s (total subscriptions: %d)",
		subscription.ID, subscriberID, totalSubscriptions+1)

	return subscription, nil
}

// Unsubscribe removes a subscription
func (eb *EventBus) Unsubscribe(subscriberID, subscriptionID string) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscriptions, exists := eb.subscribers[subscriberID]
	if !exists {
		return fmt.Errorf("subscriber not found: %s", subscriberID)
	}

	// Find and remove subscription
	for i, sub := range subscriptions {
		if sub.ID != subscriptionID {
			continue
		}

		// Close channel
		close(sub.Channel)

		// Remove from slice
		eb.subscribers[subscriberID] = append(subscriptions[:i], subscriptions[i+1:]...)

		// Clean up empty subscriber
		if len(eb.subscribers[subscriberID]) == 0 {
			delete(eb.subscribers, subscriberID)
		}

		log.Printf("Removed subscription %s for subscriber %s", subscriptionID, subscriberID)
		return nil
	}

	return fmt.Errorf("subscription not found: %s", subscriptionID)
}

// UnsubscribeAll removes all subscriptions for a subscriber
func (eb *EventBus) UnsubscribeAll(subscriberID string) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscriptions, exists := eb.subscribers[subscriberID]
	if !exists {
		return nil // Already unsubscribed
	}

	// Close all channels
	for _, sub := range subscriptions {
		close(sub.Channel)
	}

	// Remove subscriber
	delete(eb.subscribers, subscriberID)

	log.Printf("Removed all subscriptions for subscriber %s (%d subscriptions)",
		subscriberID, len(subscriptions))

	return nil
}

// Publish publishes an event to all matching subscribers
func (eb *EventBus) Publish(event *Event) error {
	if !eb.IsRunning() {
		return errors.New("event bus not running")
	}

	if event == nil {
		return errors.New("event cannot be nil")
	}

	// Check event size
	if eventSize := eb.estimateEventSize(event); eventSize > eb.config.MaxEventSize {
		return fmt.Errorf("event too large: %d bytes (max: %d)", eventSize, eb.config.MaxEventSize)
	}

	// Check for expired events
	if event.IsExpired() {
		eb.updateMetrics(func(m *BusMetrics) {
			m.EventsDropped++
		})
		return errors.New("event expired")
	}

	startTime := time.Now()

	eb.mu.RLock()
	defer eb.mu.RUnlock()

	delivered := 0
	dropped := 0

	// Distribute to all matching subscribers
	for subscriberID, subscriptions := range eb.subscribers {
		for _, subscription := range subscriptions {
			if eb.eventMatches(event, subscription.Filter) {
				select {
				case subscription.Channel <- event:
					delivered++
					eb.updateSubscriptionStats(subscription, event, time.Since(startTime))
				default:
					// Channel is full, drop event
					dropped++
					log.Printf("Dropped event %s for subscriber %s (channel full)", event.ID, subscriberID)
				}
			}
		}
	}

	// Update metrics
	eb.updateMetrics(func(m *BusMetrics) {
		m.EventsPublished++
		m.EventsDelivered += int64(delivered)
		m.EventsDropped += int64(dropped)
		m.LastEventTime = time.Now()

		// Update average latency
		latency := time.Since(startTime)
		if m.AverageLatency == 0 {
			m.AverageLatency = latency
		} else {
			// Weighted average with 90% weight on previous average
			m.AverageLatency = time.Duration(
				int64(m.AverageLatency)*9/10 + int64(latency)/10,
			)
		}
	})

	log.Printf("Published event %s (type: %s, delivered: %d, dropped: %d, latency: %v)",
		event.ID, event.Type, delivered, dropped, time.Since(startTime))

	return nil
}

// PublishBatch publishes multiple events as a batch
func (eb *EventBus) PublishBatch(events []*Event) error {
	if !eb.IsRunning() {
		return errors.New("event bus not running")
	}

	if len(events) == 0 {
		return nil
	}

	startTime := time.Now()
	published := 0
	failed := 0

	for _, event := range events {
		if err := eb.Publish(event); err != nil {
			failed++
			log.Printf("Failed to publish event %s in batch: %v", event.ID, err)
		} else {
			published++
		}
	}

	log.Printf("Published batch of %d events (%d successful, %d failed, took %v)",
		len(events), published, failed, time.Since(startTime))

	return nil
}

// GetSubscription returns a subscription by ID
func (eb *EventBus) GetSubscription(subscriberID, subscriptionID string) (*Subscription, error) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	subscriptions, exists := eb.subscribers[subscriberID]
	if !exists {
		return nil, fmt.Errorf("subscriber not found: %s", subscriberID)
	}

	for _, sub := range subscriptions {
		if sub.ID == subscriptionID {
			return sub, nil
		}
	}

	return nil, fmt.Errorf("subscription not found: %s", subscriptionID)
}

// GetSubscriptions returns all subscriptions for a subscriber
func (eb *EventBus) GetSubscriptions(subscriberID string) ([]*Subscription, error) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	subscriptions, exists := eb.subscribers[subscriberID]
	if !exists {
		return []*Subscription{}, nil
	}

	// Return a copy
	result := make([]*Subscription, len(subscriptions))
	copy(result, subscriptions)
	return result, nil
}

// GetAllSubscriptions returns all active subscriptions
func (eb *EventBus) GetAllSubscriptions() map[string][]*Subscription {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Return a deep copy
	result := make(map[string][]*Subscription)
	for subscriberID, subscriptions := range eb.subscribers {
		result[subscriberID] = make([]*Subscription, len(subscriptions))
		copy(result[subscriberID], subscriptions)
	}

	return result
}

// GetMetrics returns current bus metrics
func (eb *EventBus) GetMetrics() *BusMetrics {
	eb.metrics.mu.RLock()
	defer eb.metrics.mu.RUnlock()

	// Update active subscriptions count
	eb.mu.RLock()
	activeSubscriptions := 0
	for _, subscriptions := range eb.subscribers {
		activeSubscriptions += len(subscriptions)
	}
	eb.mu.RUnlock()

	return &BusMetrics{
		EventsPublished:     eb.metrics.EventsPublished,
		EventsDelivered:     eb.metrics.EventsDelivered,
		EventsDropped:       eb.metrics.EventsDropped,
		EventsDuplicated:    eb.metrics.EventsDuplicated,
		ActiveSubscriptions: activeSubscriptions,
		AverageLatency:      eb.metrics.AverageLatency,
		ThroughputPerSecond: eb.metrics.ThroughputPerSecond,
		LastEventTime:       eb.metrics.LastEventTime,
	}
}

// eventMatches checks if an event matches a subscription filter
func (eb *EventBus) eventMatches(event *Event, filter *EventFilter) bool {
	if filter == nil {
		return true // No filter means match all
	}

	return event.Matches(filter)
}

// updateMetrics updates bus metrics safely
func (eb *EventBus) updateMetrics(updateFunc func(*BusMetrics)) {
	eb.metrics.mu.Lock()
	defer eb.metrics.mu.Unlock()
	updateFunc(eb.metrics)
}

// updateSubscriptionStats updates subscription statistics
func (eb *EventBus) updateSubscriptionStats(subscription *Subscription, event *Event, latency time.Duration) {
	_ = event // unused parameter, kept for potential future event-specific stats
	subscription.mu.Lock()
	defer subscription.mu.Unlock()

	now := time.Now()
	subscription.LastEvent = &now
	subscription.Statistics.EventsReceived++
	subscription.Statistics.LastEventTime = &now

	// Update average latency
	if subscription.Statistics.AverageLatency == 0 {
		subscription.Statistics.AverageLatency = latency
	} else {
		// Weighted average with 80% weight on previous average
		subscription.Statistics.AverageLatency = time.Duration(
			int64(subscription.Statistics.AverageLatency)*8/10 + int64(latency)*2/10,
		)
	}
}

// estimateEventSize estimates the size of an event in bytes
func (eb *EventBus) estimateEventSize(event *Event) int {
	// This is a rough estimation
	size := len(event.ID) + len(event.Action) + len(event.Source) + len(event.Repository) +
		len(event.SessionID) + len(event.UserID) + len(event.ClientID) +
		len(event.CorrelationID) + len(event.CausationID) + len(event.ParentID)

	// Add tags size
	for _, tag := range event.Tags {
		size += len(tag)
	}

	// Add metadata size (rough estimation)
	for key, value := range event.Metadata {
		size += len(key) + len(fmt.Sprintf("%v", value))
	}

	// Add payload size (rough estimation)
	size += len(fmt.Sprintf("%v", event.Payload))

	return size
}

// cleanupRoutine performs periodic cleanup of expired subscriptions and events
func (eb *EventBus) cleanupRoutine() {
	defer eb.wg.Done()

	ticker := time.NewTicker(eb.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			eb.performCleanup()
		case <-eb.ctx.Done():
			return
		}
	}
}

// metricsRoutine calculates and updates metrics periodically
func (eb *EventBus) metricsRoutine() {
	defer eb.wg.Done()

	ticker := time.NewTicker(eb.config.MetricsInterval)
	defer ticker.Stop()

	lastEventCount := int64(0)
	lastTime := time.Now()

	for {
		select {
		case <-ticker.C:
			eb.updateMetrics(func(m *BusMetrics) {
				// Calculate throughput
				now := time.Now()
				timeDiff := now.Sub(lastTime).Seconds()
				eventDiff := m.EventsPublished - lastEventCount

				if timeDiff > 0 {
					m.ThroughputPerSecond = float64(eventDiff) / timeDiff
				}

				lastEventCount = m.EventsPublished
				lastTime = now
			})
		case <-eb.ctx.Done():
			return
		}
	}
}

// performCleanup removes inactive subscriptions and expired events
func (eb *EventBus) performCleanup() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	cleaned := 0
	cutoff := time.Now().Add(-eb.config.EventTTL)

	for subscriberID, subscriptions := range eb.subscribers {
		activeSubscriptions := make([]*Subscription, 0, len(subscriptions))

		for _, sub := range subscriptions {
			// Check if subscription is still active (received events recently)
			sub.mu.RLock()
			isActive := sub.LastEvent == nil || sub.LastEvent.After(cutoff)
			sub.mu.RUnlock()

			if isActive {
				activeSubscriptions = append(activeSubscriptions, sub)
			} else {
				// Close inactive subscription
				close(sub.Channel)
				cleaned++
				log.Printf("Cleaned up inactive subscription %s for subscriber %s", sub.ID, subscriberID)
			}
		}

		if len(activeSubscriptions) == 0 {
			delete(eb.subscribers, subscriberID)
		} else {
			eb.subscribers[subscriberID] = activeSubscriptions
		}
	}

	if cleaned > 0 {
		log.Printf("Cleanup completed: removed %d inactive subscriptions", cleaned)
	}
}

// generateSubscriptionID generates a unique subscription ID
func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond())
}

// GetStatus returns the current status of the event bus
func (eb *EventBus) GetStatus() map[string]interface{} {
	metrics := eb.GetMetrics()

	eb.mu.RLock()
	subscriberCount := len(eb.subscribers)
	eb.mu.RUnlock()

	return map[string]interface{}{
		"running":              eb.IsRunning(),
		"subscriber_count":     subscriberCount,
		"active_subscriptions": metrics.ActiveSubscriptions,
		"events_published":     metrics.EventsPublished,
		"events_delivered":     metrics.EventsDelivered,
		"events_dropped":       metrics.EventsDropped,
		"average_latency":      metrics.AverageLatency.String(),
		"throughput_per_sec":   metrics.ThroughputPerSecond,
		"last_event_time":      metrics.LastEventTime,
	}
}
