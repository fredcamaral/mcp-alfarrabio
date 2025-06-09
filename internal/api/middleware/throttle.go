// Package middleware provides HTTP middleware for request throttling and queue management.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"lerian-mcp-memory/internal/api/response"
)

// Throttler manages request throttling with priority queues
type Throttler struct {
	queues  map[Priority]*RequestQueue
	workers int
	timeout time.Duration
	mu      sync.RWMutex
	config  ThrottleConfig
	metrics *ThrottleMetrics
	ctx     context.Context
	cancel  context.CancelFunc
}

// RequestQueue manages queued requests with priority
type RequestQueue struct {
	requests chan *ThrottledRequest
	priority Priority
	maxSize  int
	timeout  time.Duration
	metrics  *QueueMetrics
}

// ThrottledRequest represents a request waiting in the queue
type ThrottledRequest struct {
	Request    *http.Request
	Writer     http.ResponseWriter
	Handler    http.Handler
	Priority   Priority
	StartTime  time.Time
	Timeout    time.Duration
	ResultChan chan *ThrottleResult
	Context    context.Context
}

// Priority defines request priority levels
type Priority int

const (
	ThrottlePriorityLow Priority = iota
	ThrottlePriorityNormal
	ThrottlePriorityHigh
	ThrottlePriorityCritical
)

// ThrottleConfig represents throttler configuration
type ThrottleConfig struct {
	Enabled        bool                `json:"enabled"`
	MaxWorkers     int                 `json:"max_workers"`
	QueueSizes     map[Priority]int    `json:"queue_sizes"`
	RequestTimeout time.Duration       `json:"request_timeout"`
	QueueTimeout   time.Duration       `json:"queue_timeout"`
	PriorityRules  map[string]Priority `json:"priority_rules"`
	EnableMetrics  bool                `json:"enable_metrics"`
	DropPolicy     DropPolicy          `json:"drop_policy"`
}

// DropPolicy defines how to handle queue overflow
type DropPolicy string

const (
	DropOldest DropPolicy = "oldest"
	DropNewest DropPolicy = "newest"
	DropLowest DropPolicy = "lowest_priority"
)

// ThrottleResult represents the result of request processing
type ThrottleResult struct {
	Success   bool          `json:"success"`
	Error     error         `json:"error,omitempty"`
	QueueTime time.Duration `json:"queue_time"`
	WaitTime  time.Duration `json:"wait_time"`
	Priority  Priority      `json:"priority"`
}

// ThrottleMetrics tracks throttling performance
type ThrottleMetrics struct {
	TotalRequests     int64            `json:"total_requests"`
	QueuedRequests    int64            `json:"queued_requests"`
	DroppedRequests   int64            `json:"dropped_requests"`
	ProcessedRequests int64            `json:"processed_requests"`
	AverageQueueTime  time.Duration    `json:"average_queue_time"`
	QueueLengths      map[Priority]int `json:"queue_lengths"`
	WorkerUtilization float64          `json:"worker_utilization"`
	mu                sync.RWMutex
}

// QueueMetrics tracks per-queue metrics
type QueueMetrics struct {
	Enqueued      int64         `json:"enqueued"`
	Dequeued      int64         `json:"dequeued"`
	Dropped       int64         `json:"dropped"`
	Timeouts      int64         `json:"timeouts"`
	AverageWait   time.Duration `json:"average_wait"`
	MaxWait       time.Duration `json:"max_wait"`
	CurrentLength int           `json:"current_length"`
	mu            sync.RWMutex
}

// DefaultThrottleConfig returns default throttling configuration
func DefaultThrottleConfig() ThrottleConfig {
	return ThrottleConfig{
		Enabled:        true,
		MaxWorkers:     10,
		RequestTimeout: 30 * time.Second,
		QueueTimeout:   5 * time.Second,
		EnableMetrics:  true,
		DropPolicy:     DropOldest,
		QueueSizes: map[Priority]int{
			ThrottlePriorityLow:      50,
			ThrottlePriorityNormal:   100,
			ThrottlePriorityHigh:     200,
			ThrottlePriorityCritical: 500,
		},
		PriorityRules: map[string]Priority{
			"/api/v1/health":        ThrottlePriorityCritical,
			"/api/v1/tasks/batch/*": ThrottlePriorityHigh,
			"/api/v1/tasks/search":  ThrottlePriorityNormal,
			"/api/v1/tasks":         ThrottlePriorityNormal,
			"/*":                    ThrottlePriorityLow,
		},
	}
}

// NewThrottler creates a new request throttler
func NewThrottler(config ThrottleConfig) *Throttler {
	ctx, cancel := context.WithCancel(context.Background())

	t := &Throttler{
		queues:  make(map[Priority]*RequestQueue),
		workers: config.MaxWorkers,
		timeout: config.RequestTimeout,
		config:  config,
		metrics: NewThrottleMetrics(),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Initialize priority queues
	for priority, size := range config.QueueSizes {
		t.queues[priority] = NewRequestQueue(priority, size, config.QueueTimeout)
	}

	// Start worker pool
	for i := 0; i < config.MaxWorkers; i++ {
		go t.worker(i)
	}

	// Start metrics updater
	if config.EnableMetrics {
		go t.updateMetrics()
	}

	return t
}

// Middleware returns HTTP middleware for request throttling
func (t *Throttler) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !t.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Determine request priority
			priority := t.determinePriority(r)

			// Create throttled request
			throttledReq := &ThrottledRequest{
				Request:    r,
				Writer:     w,
				Handler:    next,
				Priority:   priority,
				StartTime:  time.Now(),
				Timeout:    t.config.RequestTimeout,
				ResultChan: make(chan *ThrottleResult, 1),
				Context:    r.Context(),
			}

			// Try to enqueue request
			if !t.enqueue(throttledReq) {
				// Queue is full, handle based on drop policy
				response.WriteError(w, http.StatusServiceUnavailable,
					"Service temporarily unavailable",
					"Request queue is full. Please try again later.")
				return
			}

			// Wait for processing result
			select {
			case result := <-throttledReq.ResultChan:
				if !result.Success && result.Error != nil {
					response.WriteError(w, http.StatusRequestTimeout,
						"Request timeout", result.Error.Error())
				}
			case <-r.Context().Done():
				// Request was cancelled
				response.WriteError(w, http.StatusRequestTimeout,
					"Request cancelled", "Request was cancelled by client")
			case <-time.After(t.config.QueueTimeout):
				// Queue timeout
				response.WriteError(w, http.StatusServiceUnavailable,
					"Queue timeout", "Request timed out in queue")
			}
		})
	}
}

// Stop gracefully stops the throttler
func (t *Throttler) Stop() {
	t.cancel()

	// Close all queues
	for _, queue := range t.queues {
		close(queue.requests)
	}
}

// GetMetrics returns current throttling metrics
func (t *Throttler) GetMetrics() *ThrottleMetrics {
	t.metrics.mu.RLock()
	defer t.metrics.mu.RUnlock()

	// Create new instance with zero values (fresh mutex)
	metrics := NewThrottleMetrics()

	// Copy data fields explicitly (not the mutex)
	metrics.TotalRequests = t.metrics.TotalRequests
	metrics.QueuedRequests = t.metrics.QueuedRequests
	metrics.DroppedRequests = t.metrics.DroppedRequests
	metrics.ProcessedRequests = t.metrics.ProcessedRequests
	metrics.AverageQueueTime = t.metrics.AverageQueueTime
	metrics.WorkerUtilization = t.metrics.WorkerUtilization

	// Copy queue lengths
	for priority, queue := range t.queues {
		metrics.QueueLengths[priority] = len(queue.requests)
	}

	return metrics
}

// Helper methods

func (t *Throttler) determinePriority(r *http.Request) Priority {
	path := r.URL.Path

	// Check priority rules in order of specificity
	var matchedPriority Priority = ThrottlePriorityLow

	for pattern, priority := range t.config.PriorityRules {
		if t.matchesPattern(path, pattern) {
			if priority > matchedPriority {
				matchedPriority = priority
			}
		}
	}

	return matchedPriority
}

func (t *Throttler) matchesPattern(path, pattern string) bool {
	if pattern == "/*" {
		return true
	}

	if pattern == path {
		return true
	}

	if len(pattern) > 2 && pattern[len(pattern)-2:] == "/*" {
		prefix := pattern[:len(pattern)-2]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}

	return false
}

func (t *Throttler) enqueue(req *ThrottledRequest) bool {
	queue, exists := t.queues[req.Priority]
	if !exists {
		return false
	}

	select {
	case queue.requests <- req:
		// Successfully enqueued
		t.metrics.mu.Lock()
		t.metrics.QueuedRequests++
		t.metrics.TotalRequests++
		t.metrics.mu.Unlock()

		queue.metrics.mu.Lock()
		queue.metrics.Enqueued++
		queue.metrics.CurrentLength++
		queue.metrics.mu.Unlock()

		return true
	default:
		// Queue is full, apply drop policy
		return t.handleQueueOverflow(queue, req)
	}
}

func (t *Throttler) handleQueueOverflow(queue *RequestQueue, newReq *ThrottledRequest) bool {
	switch t.config.DropPolicy {
	case DropNewest:
		// Drop the new request
		t.recordDrop(queue)
		return false

	case DropOldest:
		// Try to drop oldest request and enqueue new one
		select {
		case oldReq := <-queue.requests:
			// Notify dropped request
			select {
			case oldReq.ResultChan <- &ThrottleResult{
				Success: false,
				Error:   fmt.Errorf("request dropped due to queue overflow"),
			}:
			default:
			}

			// Enqueue new request
			select {
			case queue.requests <- newReq:
				return true
			default:
				t.recordDrop(queue)
				return false
			}
		default:
			t.recordDrop(queue)
			return false
		}

	case DropLowest:
		// This would require more complex queue management
		// For now, just drop the new request
		t.recordDrop(queue)
		return false

	default:
		t.recordDrop(queue)
		return false
	}
}

func (t *Throttler) recordDrop(queue *RequestQueue) {
	t.metrics.mu.Lock()
	t.metrics.DroppedRequests++
	t.metrics.mu.Unlock()

	queue.metrics.mu.Lock()
	queue.metrics.Dropped++
	queue.metrics.mu.Unlock()
}

func (t *Throttler) worker(id int) {
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			// Process requests by priority
			req := t.getNextRequest()
			if req == nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}

			t.processRequest(req)
		}
	}
}

func (t *Throttler) getNextRequest() *ThrottledRequest {
	// Process in priority order
	priorities := []Priority{ThrottlePriorityCritical, ThrottlePriorityHigh, ThrottlePriorityNormal, ThrottlePriorityLow}

	for _, priority := range priorities {
		queue, exists := t.queues[priority]
		if !exists {
			continue
		}

		select {
		case req := <-queue.requests:
			queue.metrics.mu.Lock()
			queue.metrics.Dequeued++
			queue.metrics.CurrentLength--

			queueTime := time.Since(req.StartTime)
			if queueTime > queue.metrics.MaxWait {
				queue.metrics.MaxWait = queueTime
			}
			queue.metrics.mu.Unlock()

			return req
		default:
			continue
		}
	}

	return nil
}

func (t *Throttler) processRequest(req *ThrottledRequest) {
	defer func() {
		if r := recover(); r != nil {
			// Handle panic in request processing
			select {
			case req.ResultChan <- &ThrottleResult{
				Success: false,
				Error:   fmt.Errorf("request processing panicked: %v", r),
			}:
			default:
			}
		}
	}()

	// Check if request has timed out
	if time.Since(req.StartTime) > req.Timeout {
		select {
		case req.ResultChan <- &ThrottleResult{
			Success: false,
			Error:   fmt.Errorf("request timed out"),
		}:
		default:
		}
		return
	}

	// Check if context is cancelled
	select {
	case <-req.Context.Done():
		select {
		case req.ResultChan <- &ThrottleResult{
			Success: false,
			Error:   req.Context.Err(),
		}:
		default:
		}
		return
	default:
	}

	// Process request
	queueTime := time.Since(req.StartTime)
	processStart := time.Now()

	// Execute the handler
	req.Handler.ServeHTTP(req.Writer, req.Request)

	waitTime := time.Since(processStart)

	// Record successful processing
	t.metrics.mu.Lock()
	t.metrics.ProcessedRequests++
	t.metrics.mu.Unlock()

	// Send result
	select {
	case req.ResultChan <- &ThrottleResult{
		Success:   true,
		QueueTime: queueTime,
		WaitTime:  waitTime,
		Priority:  req.Priority,
	}:
	default:
	}
}

func (t *Throttler) updateMetrics() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.calculateMetrics()
		}
	}
}

func (t *Throttler) calculateMetrics() {
	t.metrics.mu.Lock()
	defer t.metrics.mu.Unlock()

	// Calculate worker utilization
	totalCapacity := float64(t.workers)
	if totalCapacity > 0 {
		// This is a simplified calculation
		// In practice, you'd track active workers
		t.metrics.WorkerUtilization = float64(t.metrics.ProcessedRequests) / totalCapacity
	}

	// Update queue lengths
	for priority, queue := range t.queues {
		t.metrics.QueueLengths[priority] = len(queue.requests)
	}
}

// RequestQueue implementation

func NewRequestQueue(priority Priority, maxSize int, timeout time.Duration) *RequestQueue {
	return &RequestQueue{
		requests: make(chan *ThrottledRequest, maxSize),
		priority: priority,
		maxSize:  maxSize,
		timeout:  timeout,
		metrics:  NewQueueMetrics(),
	}
}

// Metrics constructors

func NewThrottleMetrics() *ThrottleMetrics {
	return &ThrottleMetrics{
		QueueLengths: make(map[Priority]int),
	}
}

func NewQueueMetrics() *QueueMetrics {
	return &QueueMetrics{}
}

// Priority string methods

func (p Priority) String() string {
	switch p {
	case ThrottlePriorityLow:
		return "low"
	case ThrottlePriorityNormal:
		return "normal"
	case ThrottlePriorityHigh:
		return "high"
	case ThrottlePriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}
