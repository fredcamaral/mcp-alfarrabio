// Package push provides notification queuing and retry logic
package push

import (
	"context"
	"errors"
	"log"
	"sort"
	"sync"
	"time"
)

// NotificationQueue manages queuing, batching, and retry logic for notifications
type NotificationQueue struct {
	dispatcher   *Dispatcher
	registry     *Registry
	batchSize    int
	batchTimeout time.Duration
	maxQueueSize int
	retryQueue   *RetryQueue
	pendingBatch []*Notification
	queuedNotifs chan *Notification
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	running      bool
	mu           sync.RWMutex
	metrics      *QueueMetrics
}

// RetryQueue manages notifications that need to be retried
type RetryQueue struct {
	items   []*RetryItem
	mu      sync.RWMutex
	maxSize int
}

// RetryItem represents a notification waiting to be retried
type RetryItem struct {
	Notification *Notification `json:"notification"`
	RetryAt      time.Time     `json:"retry_at"`
	Attempts     int           `json:"attempts"`
	LastError    string        `json:"last_error"`
	EndpointID   string        `json:"endpoint_id"`
}

// QueueMetrics tracks notification queue performance
type QueueMetrics struct {
	TotalQueued        int64         `json:"total_queued"`
	TotalProcessed     int64         `json:"total_processed"`
	TotalDropped       int64         `json:"total_dropped"`
	TotalRetried       int64         `json:"total_retried"`
	CurrentQueueSize   int           `json:"current_queue_size"`
	CurrentBatchSize   int           `json:"current_batch_size"`
	RetryQueueSize     int           `json:"retry_queue_size"`
	AverageProcessTime time.Duration `json:"average_process_time"`
	LastProcessed      time.Time     `json:"last_processed"`
	BatchesProcessed   int64         `json:"batches_processed"`
	mu                 sync.RWMutex
}

// QueueConfig configures the notification queue
type QueueConfig struct {
	BatchSize     int           `json:"batch_size"`
	BatchTimeout  time.Duration `json:"batch_timeout"`
	MaxQueueSize  int           `json:"max_queue_size"`
	MaxRetrySize  int           `json:"max_retry_size"`
	RetryInterval time.Duration `json:"retry_interval"`
	MaxRetries    int           `json:"max_retries"`
}

// DefaultQueueConfig returns default queue configuration
func DefaultQueueConfig() *QueueConfig {
	return &QueueConfig{
		BatchSize:     10,
		BatchTimeout:  100 * time.Millisecond,
		MaxQueueSize:  10000,
		MaxRetrySize:  1000,
		RetryInterval: 30 * time.Second,
		MaxRetries:    3,
	}
}

// NewNotificationQueue creates a new notification queue
func NewNotificationQueue(dispatcher *Dispatcher, registry *Registry, config *QueueConfig) *NotificationQueue {
	if config == nil {
		config = DefaultQueueConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &NotificationQueue{
		dispatcher:   dispatcher,
		registry:     registry,
		batchSize:    config.BatchSize,
		batchTimeout: config.BatchTimeout,
		maxQueueSize: config.MaxQueueSize,
		retryQueue:   NewRetryQueue(config.MaxRetrySize),
		pendingBatch: make([]*Notification, 0, config.BatchSize),
		queuedNotifs: make(chan *Notification, config.MaxQueueSize),
		ctx:          ctx,
		cancel:       cancel,
		running:      false,
		metrics:      &QueueMetrics{},
	}
}

// NewRetryQueue creates a new retry queue
func NewRetryQueue(maxSize int) *RetryQueue {
	return &RetryQueue{
		items:   make([]*RetryItem, 0),
		maxSize: maxSize,
	}
}

// Start starts the notification queue processing
func (nq *NotificationQueue) Start() error {
	nq.mu.Lock()
	defer nq.mu.Unlock()

	if nq.running {
		return errors.New("notification queue already running")
	}

	log.Printf("Starting notification queue (batch size: %d, timeout: %v)",
		nq.batchSize, nq.batchTimeout)

	// Start batch processor
	nq.wg.Add(1)
	go nq.batchProcessor()

	// Start retry processor
	nq.wg.Add(1)
	go nq.retryProcessor()

	nq.running = true
	return nil
}

// Stop stops the notification queue gracefully
func (nq *NotificationQueue) Stop() error {
	nq.mu.Lock()
	if !nq.running {
		nq.mu.Unlock()
		return errors.New("notification queue not running")
	}
	nq.running = false
	nq.mu.Unlock()

	log.Println("Stopping notification queue...")

	// Cancel context to signal processors to stop
	nq.cancel()

	// Close notification channel
	close(nq.queuedNotifs)

	// Wait for processors to finish
	nq.wg.Wait()

	// Process any remaining notifications in the batch
	if len(nq.pendingBatch) > 0 {
		nq.processBatch(nq.pendingBatch)
	}

	log.Println("Notification queue stopped")
	return nil
}

// IsRunning returns whether the queue is running
func (nq *NotificationQueue) IsRunning() bool {
	nq.mu.RLock()
	defer nq.mu.RUnlock()
	return nq.running
}

// Enqueue adds a notification to the queue
func (nq *NotificationQueue) Enqueue(notification *Notification) error {
	if !nq.IsRunning() {
		return errors.New("notification queue not running")
	}

	select {
	case nq.queuedNotifs <- notification:
		nq.updateMetrics(func(m *QueueMetrics) {
			m.TotalQueued++
			m.CurrentQueueSize = len(nq.queuedNotifs)
		})
		return nil
	default:
		// Queue is full, drop the notification
		nq.updateMetrics(func(m *QueueMetrics) {
			m.TotalDropped++
		})
		return errors.New("notification queue full, notification dropped")
	}
}

// EnqueueBatch adds multiple notifications to the queue
func (nq *NotificationQueue) EnqueueBatch(notifications []*Notification) []error {
	errorList := make([]error, len(notifications))

	for i, notif := range notifications {
		errorList[i] = nq.Enqueue(notif)
	}

	return errorList
}

// batchProcessor processes notifications in batches
func (nq *NotificationQueue) batchProcessor() {
	defer nq.wg.Done()

	batchTimer := time.NewTimer(nq.batchTimeout)
	defer batchTimer.Stop()

	log.Println("Notification batch processor started")

	for {
		select {
		case notification, ok := <-nq.queuedNotifs:
			if !ok {
				// Channel closed, process final batch and exit
				if len(nq.pendingBatch) > 0 {
					nq.processBatch(nq.pendingBatch)
				}
				log.Println("Notification batch processor stopped")
				return
			}

			// Add to pending batch
			nq.pendingBatch = append(nq.pendingBatch, notification)
			nq.updateMetrics(func(m *QueueMetrics) {
				m.CurrentBatchSize = len(nq.pendingBatch)
				m.CurrentQueueSize = len(nq.queuedNotifs)
			})

			// Process batch if it's full
			if len(nq.pendingBatch) >= nq.batchSize {
				nq.processBatch(nq.pendingBatch)
				nq.pendingBatch = nq.pendingBatch[:0] // Reset slice

				// Reset timer
				if !batchTimer.Stop() {
					<-batchTimer.C
				}
				batchTimer.Reset(nq.batchTimeout)
			}

		case <-batchTimer.C:
			// Timeout reached, process pending batch
			if len(nq.pendingBatch) > 0 {
				nq.processBatch(nq.pendingBatch)
				nq.pendingBatch = nq.pendingBatch[:0] // Reset slice
			}
			batchTimer.Reset(nq.batchTimeout)

		case <-nq.ctx.Done():
			// Context cancelled, process final batch and exit
			if len(nq.pendingBatch) > 0 {
				nq.processBatch(nq.pendingBatch)
			}
			log.Println("Notification batch processor stopped (context cancelled)")
			return
		}
	}
}

// processBatch processes a batch of notifications
func (nq *NotificationQueue) processBatch(batch []*Notification) {
	if len(batch) == 0 {
		return
	}

	startTime := time.Now()

	log.Printf("Processing notification batch of %d notifications", len(batch))

	// Group notifications by priority for processing order
	priorityGroups := nq.groupNotificationsByPriority(batch)

	// Process each priority group in order (critical first)
	priorities := []NotificationPriority{PriorityCritical, PriorityHigh, PriorityNormal, PriorityLow}

	for _, priority := range priorities {
		notifications, exists := priorityGroups[priority]
		if !exists {
			continue
		}

		// Process notifications in this priority group
		for _, notification := range notifications {
			if err := nq.dispatcher.Dispatch(notification); err != nil {
				log.Printf("Failed to dispatch notification %s: %v", notification.ID, err)

				// Add to retry queue if dispatcher is not running or other recoverable error
				nq.addToRetryQueue(notification, err.Error(), "")
			}
		}
	}

	// Update metrics
	processTime := time.Since(startTime)
	nq.updateMetrics(func(m *QueueMetrics) {
		m.TotalProcessed += int64(len(batch))
		m.BatchesProcessed++
		m.LastProcessed = time.Now()
		m.CurrentBatchSize = 0

		// Update average process time
		if m.AverageProcessTime == 0 {
			m.AverageProcessTime = processTime
		} else {
			// Weighted average with 80% weight on previous average
			m.AverageProcessTime = time.Duration(
				int64(m.AverageProcessTime)*8/10 + int64(processTime)*2/10,
			)
		}
	})

	log.Printf("Processed notification batch in %v", processTime)
}

// groupNotificationsByPriority groups notifications by their priority
func (nq *NotificationQueue) groupNotificationsByPriority(notifications []*Notification) map[NotificationPriority][]*Notification {
	groups := make(map[NotificationPriority][]*Notification)

	for _, notif := range notifications {
		priority := notif.Priority
		if priority == "" {
			priority = PriorityNormal
		}

		groups[priority] = append(groups[priority], notif)
	}

	return groups
}

// retryProcessor processes notifications in the retry queue
func (nq *NotificationQueue) retryProcessor() {
	defer nq.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	log.Println("Notification retry processor started")

	for {
		select {
		case <-ticker.C:
			nq.processRetryQueue()

		case <-nq.ctx.Done():
			log.Println("Notification retry processor stopped")
			return
		}
	}
}

// processRetryQueue processes notifications that are ready for retry
func (nq *NotificationQueue) processRetryQueue() {
	now := time.Now()
	readyItems := nq.retryQueue.GetReadyItems(now)

	if len(readyItems) == 0 {
		return
	}

	log.Printf("Processing %d notifications from retry queue", len(readyItems))

	for _, item := range readyItems {
		// Check if endpoint is still registered and healthy
		endpoint, exists := nq.registry.Get(item.EndpointID)
		if !exists {
			log.Printf("Endpoint %s no longer exists, removing notification %s from retry queue",
				item.EndpointID, item.Notification.ID)
			nq.retryQueue.RemoveItem(item)
			continue
		}

		// Skip if endpoint is not healthy and has too many consecutive failures
		if !endpoint.Health.IsHealthy && endpoint.Health.ConsecutiveFailures > 5 {
			log.Printf("Endpoint %s is unhealthy (failures: %d), skipping retry for notification %s",
				item.EndpointID, endpoint.Health.ConsecutiveFailures, item.Notification.ID)
			continue
		}

		// Attempt retry
		if err := nq.dispatcher.DispatchToEndpoint(item.Notification, item.EndpointID); err != nil {
			nq.handleRetryFailure(item, err, now)
			continue
		}

		// Retry successful, remove from retry queue
		log.Printf("Retry successful for notification %s to endpoint %s",
			item.Notification.ID, item.EndpointID)
		nq.retryQueue.RemoveItem(item)
		nq.updateMetrics(func(m *QueueMetrics) {
			m.TotalRetried++
		})
	}
}

// addToRetryQueue adds a notification to the retry queue
func (nq *NotificationQueue) addToRetryQueue(notification *Notification, errorMsg, endpointID string) {
	if endpointID == "" {
		// If no specific endpoint, add retry for all active endpoints
		endpoints := nq.registry.GetActive()
		for _, endpoint := range endpoints {
			nq.addRetryItem(notification, errorMsg, endpoint.ID)
		}
	} else {
		nq.addRetryItem(notification, errorMsg, endpointID)
	}
}

// handleRetryFailure handles the failure of a retry attempt
func (nq *NotificationQueue) handleRetryFailure(item *RetryItem, err error, now time.Time) {
	item.Attempts++
	item.LastError = err.Error()

	if item.Attempts >= item.Notification.MaxAttempts {
		// Max attempts reached, remove from retry queue
		log.Printf("Max retry attempts reached for notification %s to endpoint %s",
			item.Notification.ID, item.EndpointID)
		nq.retryQueue.RemoveItem(item)
		return
	}

	// Schedule next retry with exponential backoff
	backoffDelay := time.Duration(int64(item.Notification.RetryDelay) * (1 << (item.Attempts - 1)))
	if backoffDelay > 5*time.Minute {
		backoffDelay = 5 * time.Minute
	}
	item.RetryAt = now.Add(backoffDelay)

	log.Printf("Rescheduled retry for notification %s to endpoint %s (attempt %d, delay: %v)",
		item.Notification.ID, item.EndpointID, item.Attempts, backoffDelay)
}

// addRetryItem adds a single retry item to the retry queue
func (nq *NotificationQueue) addRetryItem(notification *Notification, errorMsg, endpointID string) {
	item := &RetryItem{
		Notification: notification,
		RetryAt:      time.Now().Add(notification.RetryDelay),
		Attempts:     1,
		LastError:    errorMsg,
		EndpointID:   endpointID,
	}

	if nq.retryQueue.AddItem(item) {
		log.Printf("Added notification %s to retry queue for endpoint %s",
			notification.ID, endpointID)
	} else {
		log.Printf("Retry queue full, dropping notification %s for endpoint %s",
			notification.ID, endpointID)
	}
}

// AddItem adds an item to the retry queue
func (rq *RetryQueue) AddItem(item *RetryItem) bool {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if len(rq.items) >= rq.maxSize {
		return false // Queue full
	}

	rq.items = append(rq.items, item)
	return true
}

// RemoveItem removes an item from the retry queue
func (rq *RetryQueue) RemoveItem(item *RetryItem) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	for i, existing := range rq.items {
		if existing == item {
			// Remove item by swapping with last and truncating
			rq.items[i] = rq.items[len(rq.items)-1]
			rq.items = rq.items[:len(rq.items)-1]
			break
		}
	}
}

// GetReadyItems returns items that are ready for retry
func (rq *RetryQueue) GetReadyItems(now time.Time) []*RetryItem {
	rq.mu.RLock()
	defer rq.mu.RUnlock()

	var ready []*RetryItem
	for _, item := range rq.items {
		if now.After(item.RetryAt) || now.Equal(item.RetryAt) {
			ready = append(ready, item)
		}
	}

	// Sort by retry time (oldest first)
	sort.Slice(ready, func(i, j int) bool {
		return ready[i].RetryAt.Before(ready[j].RetryAt)
	})

	return ready
}

// GetSize returns the current size of the retry queue
func (rq *RetryQueue) GetSize() int {
	rq.mu.RLock()
	defer rq.mu.RUnlock()
	return len(rq.items)
}

// GetItems returns a copy of all items in the retry queue
func (rq *RetryQueue) GetItems() []*RetryItem {
	rq.mu.RLock()
	defer rq.mu.RUnlock()

	items := make([]*RetryItem, len(rq.items))
	copy(items, rq.items)
	return items
}

// updateMetrics safely updates queue metrics
func (nq *NotificationQueue) updateMetrics(updateFunc func(*QueueMetrics)) {
	nq.metrics.mu.Lock()
	defer nq.metrics.mu.Unlock()
	updateFunc(nq.metrics)
}

// GetMetrics returns queue metrics
func (nq *NotificationQueue) GetMetrics() *QueueMetrics {
	nq.metrics.mu.RLock()
	defer nq.metrics.mu.RUnlock()

	// Return a copy
	return &QueueMetrics{
		TotalQueued:        nq.metrics.TotalQueued,
		TotalProcessed:     nq.metrics.TotalProcessed,
		TotalDropped:       nq.metrics.TotalDropped,
		TotalRetried:       nq.metrics.TotalRetried,
		CurrentQueueSize:   nq.metrics.CurrentQueueSize,
		CurrentBatchSize:   nq.metrics.CurrentBatchSize,
		RetryQueueSize:     nq.retryQueue.GetSize(),
		AverageProcessTime: nq.metrics.AverageProcessTime,
		LastProcessed:      nq.metrics.LastProcessed,
		BatchesProcessed:   nq.metrics.BatchesProcessed,
	}
}

// GetQueueStatus returns the current status of the queue
func (nq *NotificationQueue) GetQueueStatus() map[string]interface{} {
	metrics := nq.GetMetrics()

	return map[string]interface{}{
		"running":              nq.IsRunning(),
		"batch_size":           nq.batchSize,
		"batch_timeout":        nq.batchTimeout.String(),
		"max_queue_size":       nq.maxQueueSize,
		"current_queue_size":   metrics.CurrentQueueSize,
		"current_batch_size":   metrics.CurrentBatchSize,
		"retry_queue_size":     metrics.RetryQueueSize,
		"total_queued":         metrics.TotalQueued,
		"total_processed":      metrics.TotalProcessed,
		"total_dropped":        metrics.TotalDropped,
		"total_retried":        metrics.TotalRetried,
		"batches_processed":    metrics.BatchesProcessed,
		"average_process_time": metrics.AverageProcessTime.String(),
		"last_processed":       metrics.LastProcessed,
	}
}

// Flush processes all pending notifications immediately
func (nq *NotificationQueue) Flush() {
	if !nq.IsRunning() {
		return
	}

	// Process current batch
	if len(nq.pendingBatch) > 0 {
		nq.processBatch(nq.pendingBatch)
		nq.pendingBatch = nq.pendingBatch[:0]
	}

	// Process any remaining queued notifications
	for {
		select {
		case notification := <-nq.queuedNotifs:
			batch := []*Notification{notification}
			nq.processBatch(batch)
		default:
			return // No more notifications
		}
	}
}
