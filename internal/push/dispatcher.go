// Package push provides push notification dispatcher for CLI endpoints
package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Notification represents a push notification to be delivered
type Notification struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	TargetID    string                 `json:"target_id"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   time.Time              `json:"expires_at"`
	Priority    NotificationPriority   `json:"priority"`
	Metadata    map[string]string      `json:"metadata"`
	Attempts    int                    `json:"attempts"`
	MaxAttempts int                    `json:"max_attempts"`
	RetryDelay  time.Duration          `json:"retry_delay"`
}

// NotificationPriority defines notification priority levels
type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityNormal   NotificationPriority = "normal"
	PriorityHigh     NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
)

// DeliveryResult represents the result of a notification delivery attempt
type DeliveryResult struct {
	NotificationID string        `json:"notification_id"`
	EndpointID     string        `json:"endpoint_id"`
	Success        bool          `json:"success"`
	StatusCode     int           `json:"status_code"`
	Response       string        `json:"response"`
	Error          string        `json:"error"`
	Duration       time.Duration `json:"duration"`
	Timestamp      time.Time     `json:"timestamp"`
	Attempt        int           `json:"attempt"`
}

// DeliveryTracker tracks notification delivery status
type DeliveryTracker struct {
	mu        sync.RWMutex
	results   map[string][]*DeliveryResult
	analytics *DeliveryAnalytics
}

// DeliveryAnalytics tracks delivery performance metrics
type DeliveryAnalytics struct {
	TotalDeliveries      int64         `json:"total_deliveries"`
	SuccessfulDeliveries int64         `json:"successful_deliveries"`
	FailedDeliveries     int64         `json:"failed_deliveries"`
	AverageLatency       time.Duration `json:"average_latency"`
	RetryRate            float64       `json:"retry_rate"`
	ErrorRate            float64       `json:"error_rate"`
	LastDelivery         time.Time     `json:"last_delivery"`
	mu                   sync.RWMutex
}

// Dispatcher manages notification delivery to CLI endpoints
type Dispatcher struct {
	registry    *Registry
	tracker     *DeliveryTracker
	httpClient  *http.Client
	workerCount int
	jobQueue    chan *DeliveryJob
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	running     bool
	mu          sync.RWMutex
}

// DeliveryJob represents a notification delivery job
type DeliveryJob struct {
	Notification *Notification
	Endpoint     *CLIEndpoint
	Attempt      int
	ScheduledAt  time.Time
}

// DispatcherConfig configures the notification dispatcher
type DispatcherConfig struct {
	WorkerCount    int           `json:"worker_count"`
	QueueSize      int           `json:"queue_size"`
	Timeout        time.Duration `json:"timeout"`
	RetryBackoff   time.Duration `json:"retry_backoff"`
	MaxConcurrency int           `json:"max_concurrency"`
	CircuitBreaker bool          `json:"circuit_breaker"`
}

// DefaultDispatcherConfig returns default dispatcher configuration
func DefaultDispatcherConfig() *DispatcherConfig {
	return &DispatcherConfig{
		WorkerCount:    5,
		QueueSize:      1000,
		Timeout:        5 * time.Second,
		RetryBackoff:   time.Second,
		MaxConcurrency: 10,
		CircuitBreaker: true,
	}
}

// NewDispatcher creates a new push notification dispatcher
func NewDispatcher(registry *Registry, config *DispatcherConfig) *Dispatcher {
	if config == nil {
		config = DefaultDispatcherConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	dispatcher := &Dispatcher{
		registry:    registry,
		tracker:     NewDeliveryTracker(),
		httpClient:  &http.Client{Timeout: config.Timeout},
		workerCount: config.WorkerCount,
		jobQueue:    make(chan *DeliveryJob, config.QueueSize),
		ctx:         ctx,
		cancel:      cancel,
		running:     false,
	}

	return dispatcher
}

// NewDeliveryTracker creates a new delivery tracker
func NewDeliveryTracker() *DeliveryTracker {
	return &DeliveryTracker{
		results:   make(map[string][]*DeliveryResult),
		analytics: &DeliveryAnalytics{},
	}
}

// Start starts the dispatcher workers
func (d *Dispatcher) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("dispatcher already running")
	}

	log.Printf("Starting notification dispatcher with %d workers", d.workerCount)

	// Start worker goroutines
	for i := 0; i < d.workerCount; i++ {
		d.wg.Add(1)
		go d.worker(i)
	}

	d.running = true
	return nil
}

// Stop stops the dispatcher gracefully
func (d *Dispatcher) Stop() error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return fmt.Errorf("dispatcher not running")
	}
	d.running = false
	d.mu.Unlock()

	log.Println("Stopping notification dispatcher...")

	// Cancel context to signal workers to stop
	d.cancel()

	// Close job queue
	close(d.jobQueue)

	// Wait for all workers to finish
	d.wg.Wait()

	log.Println("Notification dispatcher stopped")
	return nil
}

// IsRunning returns whether the dispatcher is running
func (d *Dispatcher) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// Dispatch sends a notification to all active CLI endpoints
func (d *Dispatcher) Dispatch(notification *Notification) error {
	if !d.IsRunning() {
		return fmt.Errorf("dispatcher not running")
	}

	// Get active endpoints
	endpoints := d.registry.GetActive()
	if len(endpoints) == 0 {
		log.Printf("No active CLI endpoints to deliver notification %s", notification.ID)
		return nil
	}

	log.Printf("Dispatching notification %s to %d active endpoints", notification.ID, len(endpoints))

	// Create delivery jobs for each endpoint
	for _, endpoint := range endpoints {
		// Check if endpoint should receive this notification
		if d.shouldDeliverToEndpoint(notification, endpoint) {
			job := &DeliveryJob{
				Notification: notification,
				Endpoint:     endpoint,
				Attempt:      1,
				ScheduledAt:  time.Now(),
			}

			select {
			case d.jobQueue <- job:
				// Job queued successfully
			default:
				log.Printf("Job queue full, dropping notification %s for endpoint %s",
					notification.ID, endpoint.ID)
			}
		}
	}

	return nil
}

// DispatchToEndpoint sends a notification to a specific CLI endpoint
func (d *Dispatcher) DispatchToEndpoint(notification *Notification, endpointID string) error {
	if !d.IsRunning() {
		return fmt.Errorf("dispatcher not running")
	}

	endpoint, exists := d.registry.Get(endpointID)
	if !exists {
		return fmt.Errorf("endpoint not found: %s", endpointID)
	}

	if endpoint.Status != StatusActive || !endpoint.Health.IsHealthy {
		return fmt.Errorf("endpoint %s is not active or healthy", endpointID)
	}

	job := &DeliveryJob{
		Notification: notification,
		Endpoint:     endpoint,
		Attempt:      1,
		ScheduledAt:  time.Now(),
	}

	select {
	case d.jobQueue <- job:
		return nil
	default:
		return fmt.Errorf("job queue full")
	}
}

// worker processes delivery jobs
func (d *Dispatcher) worker(id int) {
	defer d.wg.Done()

	log.Printf("Notification worker %d started", id)

	for {
		select {
		case job, ok := <-d.jobQueue:
			if !ok {
				log.Printf("Notification worker %d stopped (queue closed)", id)
				return
			}

			d.processDeliveryJob(job)

		case <-d.ctx.Done():
			log.Printf("Notification worker %d stopped (context cancelled)", id)
			return
		}
	}
}

// processDeliveryJob processes a single delivery job
func (d *Dispatcher) processDeliveryJob(job *DeliveryJob) {
	startTime := time.Now()

	result := &DeliveryResult{
		NotificationID: job.Notification.ID,
		EndpointID:     job.Endpoint.ID,
		Timestamp:      startTime,
		Attempt:        job.Attempt,
	}

	// Prepare notification payload
	payload, err := json.Marshal(job.Notification)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to marshal notification: %v", err)
		d.trackDeliveryResult(result)
		return
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(d.ctx, "POST", job.Endpoint.URL, bytes.NewBuffer(payload))
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		d.trackDeliveryResult(result)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "MCP-Memory-Push-Notifier/1.0")
	req.Header.Set("X-Notification-ID", job.Notification.ID)
	req.Header.Set("X-Notification-Type", job.Notification.Type)

	// Send request
	resp, err := d.httpClient.Do(req)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP request failed: %v", err)
		result.Duration = time.Since(startTime)
		d.trackDeliveryResult(result)
		d.handleDeliveryFailure(job, result)
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(startTime)

	// Read response body
	var response bytes.Buffer
	if _, err := response.ReadFrom(resp.Body); err != nil {
		log.Printf("Failed to read response body: %v", err)
	}
	result.Response = response.String()

	// Check if delivery was successful
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
		log.Printf("Successfully delivered notification %s to endpoint %s (attempt %d, %v)",
			job.Notification.ID, job.Endpoint.ID, job.Attempt, result.Duration)

		// Update endpoint health
		if err := d.registry.UpdateLastSeen(job.Endpoint.ID); err != nil {
			log.Printf("Failed to update last seen for endpoint %s: %v", job.Endpoint.ID, err)
		}
	} else {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, result.Response)
		log.Printf("Failed to deliver notification %s to endpoint %s: %s",
			job.Notification.ID, job.Endpoint.ID, result.Error)

		d.handleDeliveryFailure(job, result)
	}

	d.trackDeliveryResult(result)
}

// shouldDeliverToEndpoint checks if a notification should be delivered to an endpoint
func (d *Dispatcher) shouldDeliverToEndpoint(notification *Notification, endpoint *CLIEndpoint) bool {
	// Check if notification is expired
	if !notification.ExpiresAt.IsZero() && time.Now().After(notification.ExpiresAt) {
		return false
	}

	// Check endpoint preferences
	if endpoint.Preferences == nil {
		return true
	}

	return d.checkEndpointPreferences(notification, endpoint.Preferences)
}

// checkEndpointPreferences validates notification against endpoint preferences
func (d *Dispatcher) checkEndpointPreferences(notification *Notification, preferences *NotificationPreferences) bool {
	if !d.isEventEnabled(notification.Type, preferences) {
		return false
	}

	if d.isEventDisabled(notification.Type, preferences) {
		return false
	}

	return d.passesFilters(notification, preferences)
}

// isEventEnabled checks if the notification type is in the enabled events list
func (d *Dispatcher) isEventEnabled(notificationType string, preferences *NotificationPreferences) bool {
	// If no enabled events are specified, all events are enabled by default
	if len(preferences.EnabledEvents) == 0 {
		return true
	}

	for _, event := range preferences.EnabledEvents {
		if event == notificationType {
			return true
		}
	}
	return false
}

// isEventDisabled checks if the notification type is in the disabled events list
func (d *Dispatcher) isEventDisabled(notificationType string, preferences *NotificationPreferences) bool {
	for _, event := range preferences.DisabledEvents {
		if event == notificationType {
			return true
		}
	}
	return false
}

// passesFilters checks if the notification passes all endpoint filters
func (d *Dispatcher) passesFilters(notification *Notification, preferences *NotificationPreferences) bool {
	if len(preferences.Filters) == 0 {
		return true
	}

	for key, value := range preferences.Filters {
		notificationValue, exists := notification.Metadata[key]
		if !exists || notificationValue != value {
			return false
		}
	}
	return true
}

// handleDeliveryFailure handles failed notification deliveries with retry logic
func (d *Dispatcher) handleDeliveryFailure(job *DeliveryJob, result *DeliveryResult) {
	// Update endpoint health based on failure
	health := &EndpointHealth{
		IsHealthy:           false,
		LastHealthCheck:     time.Now(),
		ConsecutiveFailures: job.Endpoint.Health.ConsecutiveFailures + 1,
		LastError:           result.Error,
		TotalRequests:       job.Endpoint.Health.TotalRequests + 1,
		SuccessfulRequests:  job.Endpoint.Health.SuccessfulRequests,
	}

	// Calculate success rate
	if health.TotalRequests > 0 {
		health.SuccessRate = float64(health.SuccessfulRequests) / float64(health.TotalRequests)
	}

	if err := d.registry.UpdateHealth(job.Endpoint.ID, health); err != nil {
		log.Printf("Failed to update health for endpoint %s: %v", job.Endpoint.ID, err)
	}

	// Check if we should retry
	maxAttempts := job.Notification.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = job.Endpoint.Preferences.MaxRetries
	}

	if job.Attempt < maxAttempts {
		// Calculate retry delay with exponential backoff
		retryDelay := job.Notification.RetryDelay
		if retryDelay == 0 {
			retryDelay = job.Endpoint.Preferences.RetryDelay
		}

		// Exponential backoff: delay * 2^(attempt-1)
		backoffDelay := time.Duration(int64(retryDelay) * (1 << (job.Attempt - 1)))

		// Cap backoff at 5 minutes
		if backoffDelay > 5*time.Minute {
			backoffDelay = 5 * time.Minute
		}

		// Schedule retry
		go func() {
			time.Sleep(backoffDelay)

			retryJob := &DeliveryJob{
				Notification: job.Notification,
				Endpoint:     job.Endpoint,
				Attempt:      job.Attempt + 1,
				ScheduledAt:  time.Now(),
			}

			select {
			case d.jobQueue <- retryJob:
				log.Printf("Scheduled retry %d for notification %s to endpoint %s (delay: %v)",
					retryJob.Attempt, job.Notification.ID, job.Endpoint.ID, backoffDelay)
			default:
				log.Printf("Failed to schedule retry for notification %s to endpoint %s (queue full)",
					job.Notification.ID, job.Endpoint.ID)
			}
		}()
	} else {
		log.Printf("Max attempts reached for notification %s to endpoint %s",
			job.Notification.ID, job.Endpoint.ID)
	}
}

// trackDeliveryResult records delivery results for analytics
func (d *Dispatcher) trackDeliveryResult(result *DeliveryResult) {
	d.tracker.mu.Lock()
	defer d.tracker.mu.Unlock()

	// Store result
	if d.tracker.results[result.NotificationID] == nil {
		d.tracker.results[result.NotificationID] = make([]*DeliveryResult, 0)
	}
	d.tracker.results[result.NotificationID] = append(d.tracker.results[result.NotificationID], result)

	// Update analytics
	d.tracker.analytics.mu.Lock()
	defer d.tracker.analytics.mu.Unlock()

	d.tracker.analytics.TotalDeliveries++
	d.tracker.analytics.LastDelivery = result.Timestamp

	if result.Success {
		d.tracker.analytics.SuccessfulDeliveries++
	} else {
		d.tracker.analytics.FailedDeliveries++
	}

	// Update average latency (simple moving average)
	if d.tracker.analytics.AverageLatency == 0 {
		d.tracker.analytics.AverageLatency = result.Duration
	} else {
		// Weighted average with 90% weight on previous average
		d.tracker.analytics.AverageLatency = time.Duration(
			int64(d.tracker.analytics.AverageLatency)*9/10 + int64(result.Duration)/10,
		)
	}

	// Calculate rates
	if d.tracker.analytics.TotalDeliveries > 0 {
		d.tracker.analytics.ErrorRate = float64(d.tracker.analytics.FailedDeliveries) /
			float64(d.tracker.analytics.TotalDeliveries) * 100

		// Retry rate (approximate)
		d.tracker.analytics.RetryRate = float64(d.tracker.analytics.TotalDeliveries-
			d.tracker.analytics.SuccessfulDeliveries-d.tracker.analytics.FailedDeliveries) /
			float64(d.tracker.analytics.TotalDeliveries) * 100
	}
}

// GetDeliveryResults returns delivery results for a notification
func (d *Dispatcher) GetDeliveryResults(notificationID string) ([]*DeliveryResult, bool) {
	d.tracker.mu.RLock()
	defer d.tracker.mu.RUnlock()

	results, exists := d.tracker.results[notificationID]
	if !exists {
		return nil, false
	}

	// Return a copy
	resultsCopy := make([]*DeliveryResult, len(results))
	copy(resultsCopy, results)
	return resultsCopy, true
}

// GetAnalytics returns delivery analytics
func (d *Dispatcher) GetAnalytics() *DeliveryAnalytics {
	d.tracker.analytics.mu.RLock()
	defer d.tracker.analytics.mu.RUnlock()

	// Return a copy
	return &DeliveryAnalytics{
		TotalDeliveries:      d.tracker.analytics.TotalDeliveries,
		SuccessfulDeliveries: d.tracker.analytics.SuccessfulDeliveries,
		FailedDeliveries:     d.tracker.analytics.FailedDeliveries,
		AverageLatency:       d.tracker.analytics.AverageLatency,
		RetryRate:            d.tracker.analytics.RetryRate,
		ErrorRate:            d.tracker.analytics.ErrorRate,
		LastDelivery:         d.tracker.analytics.LastDelivery,
	}
}

// GetQueueStatus returns the current status of the job queue
func (d *Dispatcher) GetQueueStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":        d.IsRunning(),
		"worker_count":   d.workerCount,
		"queue_length":   len(d.jobQueue),
		"queue_capacity": cap(d.jobQueue),
		"queue_full":     len(d.jobQueue) == cap(d.jobQueue),
	}
}

// CreateNotification creates a new notification with default values
func CreateNotification(notificationType string, payload map[string]interface{}) *Notification {
	return &Notification{
		ID:          fmt.Sprintf("notif_%d", time.Now().UnixNano()),
		Type:        notificationType,
		Payload:     payload,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(10 * time.Minute), // Default 10 minute expiry
		Priority:    PriorityNormal,
		Metadata:    make(map[string]string),
		Attempts:    0,
		MaxAttempts: 3,
		RetryDelay:  time.Second,
	}
}
