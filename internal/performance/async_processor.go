// Package performance provides async processing for heavy operations
package performance

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AsyncProcessor handles background processing of heavy operations
type AsyncProcessor struct {
	config      *ProcessorConfig
	workers     []*worker
	jobQueue    chan Job
	resultQueue chan Result
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	metrics     *ProcessorMetrics
}

// ProcessorConfig defines async processor configuration
type ProcessorConfig struct {
	// Worker settings
	WorkerCount int           `json:"worker_count"`
	QueueSize   int           `json:"queue_size"`
	MaxRetries  int           `json:"max_retries"`
	RetryDelay  time.Duration `json:"retry_delay"`

	// Performance settings
	BatchSize      int           `json:"batch_size"`
	ProcessTimeout time.Duration `json:"process_timeout"`
	IdleTimeout    time.Duration `json:"idle_timeout"`

	// Quality settings
	EnableMetrics bool `json:"enable_metrics"`
	EnableTracing bool `json:"enable_tracing"`
}

// DefaultProcessorConfig returns optimized default configuration
func DefaultProcessorConfig() *ProcessorConfig {
	return &ProcessorConfig{
		WorkerCount:    10, // Optimal for I/O-bound operations
		QueueSize:      1000,
		MaxRetries:     3,
		RetryDelay:     1 * time.Second,
		BatchSize:      50,
		ProcessTimeout: 30 * time.Second,
		IdleTimeout:    5 * time.Minute,
		EnableMetrics:  true,
		EnableTracing:  false, // Enable in production for debugging
	}
}

// Job represents a unit of work to be processed asynchronously
type Job struct {
	ID         string                 `json:"id"`
	Type       JobType                `json:"type"`
	Priority   Priority               `json:"priority"`
	Payload    map[string]interface{} `json:"payload"`
	CreatedAt  time.Time              `json:"created_at"`
	Timeout    time.Duration          `json:"timeout"`
	RetryCount int                    `json:"retry_count"`
	Metadata   map[string]string      `json:"metadata"`
}

// Result represents the result of a processed job
type Result struct {
	JobID       string                 `json:"job_id"`
	Success     bool                   `json:"success"`
	Data        map[string]interface{} `json:"data"`
	Error       error                  `json:"error,omitempty"`
	ProcessedAt time.Time              `json:"processed_at"`
	Duration    time.Duration          `json:"duration"`
	WorkerID    string                 `json:"worker_id"`
}

// JobType defines the type of job to be processed
type JobType string

const (
	JobTypeEmbedding        JobType = "embedding_generation"
	JobTypeAnalysis         JobType = "content_analysis"
	JobTypeIndexing         JobType = "search_indexing"
	JobTypeBulkOperation    JobType = "bulk_operation"
	JobTypeQualityCheck     JobType = "quality_check"
	JobTypePatternDetection JobType = "pattern_detection"
)

// Priority defines job processing priority
type Priority int

const (
	PriorityLow      Priority = 1
	PriorityNormal   Priority = 5
	PriorityHigh     Priority = 8
	PriorityCritical Priority = 10
)

// ProcessorMetricsData tracks async processor performance metrics
type ProcessorMetricsData struct {
	JobsQueued       int64         `json:"jobs_queued"`
	JobsProcessed    int64         `json:"jobs_processed"`
	JobsFailed       int64         `json:"jobs_failed"`
	JobsRetried      int64         `json:"jobs_retried"`
	WorkersActive    int           `json:"workers_active"`
	QueueLength      int           `json:"queue_length"`
	AvgProcessTime   time.Duration `json:"avg_process_time"`
	TotalProcessTime time.Duration `json:"total_process_time"`
}

// ProcessorMetrics holds internal metrics with synchronization
type ProcessorMetrics struct {
	mutex sync.RWMutex
	data  ProcessorMetricsData
}

// worker represents a background worker
type worker struct {
	id        string
	processor *AsyncProcessor
	active    bool
	lastJob   time.Time
}

// JobProcessor defines the interface for processing specific job types
type JobProcessor interface {
	Process(ctx context.Context, job Job) (*Result, error)
	CanHandle(jobType JobType) bool
	Priority() Priority
}

// NewAsyncProcessor creates a new high-performance async processor
func NewAsyncProcessor(config *ProcessorConfig) *AsyncProcessor {
	if config == nil {
		config = DefaultProcessorConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	processor := &AsyncProcessor{
		config:      config,
		jobQueue:    make(chan Job, config.QueueSize),
		resultQueue: make(chan Result, config.QueueSize),
		ctx:         ctx,
		cancel:      cancel,
		metrics:     &ProcessorMetrics{data: ProcessorMetricsData{}},
	}

	// Start workers
	processor.startWorkers()

	// Start metrics collection if enabled
	if config.EnableMetrics {
		go processor.metricsCollector()
	}

	return processor
}

// SubmitJob submits a job for async processing
func (p *AsyncProcessor) SubmitJob(job *Job) error {
	if job.ID == "" {
		job.ID = generateJobID()
	}

	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}

	if job.Timeout == 0 {
		job.Timeout = p.config.ProcessTimeout
	}

	select {
	case p.jobQueue <- *job:
		p.incrementJobsQueued()
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("processor is shutting down")
	default:
		return fmt.Errorf("job queue is full")
	}
}

// SubmitBatch submits multiple jobs as a batch
func (p *AsyncProcessor) SubmitBatch(jobs []Job) error {
	if len(jobs) == 0 {
		return nil
	}

	// Process in batches to avoid blocking
	batchSize := p.config.BatchSize
	for i := 0; i < len(jobs); i += batchSize {
		end := i + batchSize
		if end > len(jobs) {
			end = len(jobs)
		}

		for j := i; j < end; j++ {
			if err := p.SubmitJob(&jobs[j]); err != nil {
				return fmt.Errorf("failed to submit job %d: %w", j, err)
			}
		}
	}

	return nil
}

// GetResult retrieves a processed result (non-blocking)
func (p *AsyncProcessor) GetResult() (*Result, bool) {
	select {
	case result := <-p.resultQueue:
		return &result, true
	default:
		return nil, false
	}
}

// WaitForResult waits for a specific job result with timeout
func (p *AsyncProcessor) WaitForResult(jobID string, timeout time.Duration) (*Result, error) {
	ctx, cancel := context.WithTimeout(p.ctx, timeout)
	defer cancel()

	for {
		select {
		case result := <-p.resultQueue:
			if result.JobID == jobID {
				return &result, nil
			}
			// Put back result that doesn't match
			select {
			case p.resultQueue <- result:
			default:
				// Queue full, result lost
			}
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for result of job %s", jobID)
		}
	}
}

// GetMetrics returns current processor metrics
func (p *AsyncProcessor) GetMetrics() ProcessorMetricsData {
	p.metrics.mutex.RLock()
	defer p.metrics.mutex.RUnlock()

	// Return a copy of the data without the mutex
	metrics := p.metrics.data
	metrics.QueueLength = len(p.jobQueue)

	// Calculate average process time
	if metrics.JobsProcessed > 0 {
		metrics.AvgProcessTime = metrics.TotalProcessTime / time.Duration(metrics.JobsProcessed)
	}

	return metrics
}

// Shutdown gracefully shuts down the processor
func (p *AsyncProcessor) Shutdown(timeout time.Duration) error {
	// Stop accepting new jobs
	p.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout exceeded")
	}
}

// startWorkers starts the configured number of worker goroutines
func (p *AsyncProcessor) startWorkers() {
	p.workers = make([]*worker, p.config.WorkerCount)

	for i := 0; i < p.config.WorkerCount; i++ {
		worker := &worker{
			id:        fmt.Sprintf("worker-%d", i),
			processor: p,
			active:    false,
		}

		p.workers[i] = worker
		p.wg.Add(1)
		go worker.run()
	}
}

// run is the main worker loop
func (w *worker) run() {
	defer w.processor.wg.Done()

	for {
		select {
		case job := <-w.processor.jobQueue:
			w.processJob(&job)
		case <-w.processor.ctx.Done():
			return
		case <-time.After(w.processor.config.IdleTimeout):
			// Worker idle timeout - could implement worker scaling here
			continue
		}
	}
}

// processJob processes a single job
func (w *worker) processJob(job *Job) {
	w.active = true
	w.lastJob = time.Now()
	defer func() {
		w.active = false
	}()

	start := time.Now()

	// Create job context with timeout
	ctx, cancel := context.WithTimeout(w.processor.ctx, job.Timeout)
	defer cancel()

	// Process the job
	result := w.executeJob(ctx, job)
	result.WorkerID = w.id
	result.Duration = time.Since(start)
	result.ProcessedAt = time.Now()

	// Track metrics
	w.processor.updateMetrics(result)

	// Send result (non-blocking)
	select {
	case w.processor.resultQueue <- *result:
	default:
		// Result queue full - in production, would log this
	}
}

// executeJob executes the actual job processing with retry logic
func (w *worker) executeJob(ctx context.Context, job *Job) *Result {
	var lastErr error

	for attempt := 0; attempt <= w.processor.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-time.After(w.processor.config.RetryDelay):
			case <-ctx.Done():
				return &Result{
					JobID:   job.ID,
					Success: false,
					Error:   fmt.Errorf("job cancelled during retry: %w", ctx.Err()),
				}
			}

			w.processor.incrementJobsRetried()
		}

		// Execute job based on type
		result, err := w.processJobByType(ctx, job)
		if err == nil {
			w.processor.incrementJobsProcessed()
			return result
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			break
		}
	}

	w.processor.incrementJobsFailed()
	return &Result{
		JobID:   job.ID,
		Success: false,
		Error:   fmt.Errorf("job failed after %d attempts: %w", w.processor.config.MaxRetries+1, lastErr),
	}
}

// processJobByType processes job based on its type
func (w *worker) processJobByType(ctx context.Context, job *Job) (*Result, error) {
	switch job.Type {
	case JobTypeEmbedding:
		return w.processEmbeddingJob(ctx, job)
	case JobTypeAnalysis:
		return w.processAnalysisJob(ctx, job)
	case JobTypeIndexing:
		return w.processIndexingJob(ctx, job)
	case JobTypeBulkOperation:
		return w.processBulkOperationJob(ctx, job)
	case JobTypeQualityCheck:
		return w.processQualityCheckJob(ctx, job)
	case JobTypePatternDetection:
		return w.processPatternDetectionJob(ctx, job)
	default:
		return nil, fmt.Errorf("unknown job type: %s", job.Type)
	}
}

// Job-specific processors (simplified implementations)
func (w *worker) processEmbeddingJob(ctx context.Context, job *Job) (*Result, error) {
	// Simulate embedding generation
	content, ok := job.Payload["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid content in embedding job")
	}

	// Simulate processing time
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &Result{
		JobID:   job.ID,
		Success: true,
		Data: map[string]interface{}{
			"embedding_id":   generateJobID(),
			"content_length": len(content),
			"vector_size":    1536,
		},
	}, nil
}

func (w *worker) processAnalysisJob(ctx context.Context, job *Job) (*Result, error) {
	// Simulate content analysis
	content, ok := job.Payload["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid content in analysis job")
	}

	// Simulate processing time
	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &Result{
		JobID:   job.ID,
		Success: true,
		Data: map[string]interface{}{
			"quality_score": 0.85,
			"word_count":    len(content),
			"sentiment":     "neutral",
		},
	}, nil
}

func (w *worker) processIndexingJob(ctx context.Context, job *Job) (*Result, error) {
	// Simulate search indexing
	contentID, ok := job.Payload["content_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid content_id in indexing job")
	}

	// Simulate processing time
	select {
	case <-time.After(50 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &Result{
		JobID:   job.ID,
		Success: true,
		Data: map[string]interface{}{
			"content_id":    contentID,
			"indexed":       true,
			"index_version": "v2.1",
		},
	}, nil
}

func (w *worker) processBulkOperationJob(ctx context.Context, job *Job) (*Result, error) {
	// Simulate bulk operation
	items, ok := job.Payload["items"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid items in bulk operation job")
	}

	// Simulate processing time based on item count
	processingTime := time.Duration(len(items)) * 10 * time.Millisecond
	select {
	case <-time.After(processingTime):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &Result{
		JobID:   job.ID,
		Success: true,
		Data: map[string]interface{}{
			"items_processed": len(items),
			"success_count":   len(items),
			"failed_count":    0,
		},
	}, nil
}

func (w *worker) processQualityCheckJob(ctx context.Context, job *Job) (*Result, error) {
	// Simulate quality check
	contentID, ok := job.Payload["content_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid content_id in quality check job")
	}

	// Simulate processing time
	select {
	case <-time.After(150 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &Result{
		JobID:   job.ID,
		Success: true,
		Data: map[string]interface{}{
			"content_id":    contentID,
			"quality_score": 0.92,
			"issues_found":  2,
			"passed":        true,
		},
	}, nil
}

func (w *worker) processPatternDetectionJob(ctx context.Context, job *Job) (*Result, error) {
	// Simulate pattern detection
	projectID, ok := job.Payload["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid project_id in pattern detection job")
	}

	// Simulate processing time
	select {
	case <-time.After(300 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return &Result{
		JobID:   job.ID,
		Success: true,
		Data: map[string]interface{}{
			"project_id":     projectID,
			"patterns_found": 5,
			"confidence":     0.88,
		},
	}, nil
}

// metricsCollector runs background metrics collection
func (p *AsyncProcessor) metricsCollector() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.collectMetrics()
		case <-p.ctx.Done():
			return
		}
	}
}

// collectMetrics updates processor metrics
func (p *AsyncProcessor) collectMetrics() {
	p.metrics.mutex.Lock()
	defer p.metrics.mutex.Unlock()

	// Count active workers
	activeWorkers := 0
	for _, worker := range p.workers {
		if worker.active {
			activeWorkers++
		}
	}

	p.metrics.data.WorkersActive = activeWorkers
	p.metrics.data.QueueLength = len(p.jobQueue)
}

// Metric update methods
func (p *AsyncProcessor) incrementJobsQueued() {
	p.metrics.mutex.Lock()
	p.metrics.data.JobsQueued++
	p.metrics.mutex.Unlock()
}

func (p *AsyncProcessor) incrementJobsProcessed() {
	p.metrics.mutex.Lock()
	p.metrics.data.JobsProcessed++
	p.metrics.mutex.Unlock()
}

func (p *AsyncProcessor) incrementJobsFailed() {
	p.metrics.mutex.Lock()
	p.metrics.data.JobsFailed++
	p.metrics.mutex.Unlock()
}

func (p *AsyncProcessor) incrementJobsRetried() {
	p.metrics.mutex.Lock()
	p.metrics.data.JobsRetried++
	p.metrics.mutex.Unlock()
}

func (p *AsyncProcessor) updateMetrics(result *Result) {
	p.metrics.mutex.Lock()
	defer p.metrics.mutex.Unlock()

	p.metrics.data.TotalProcessTime += result.Duration
}

// Utility functions
func generateJobID() string {
	return fmt.Sprintf("job_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

func isRetryableError(err error) bool {
	// Simple retry logic - in production, would be more sophisticated
	if err == nil {
		return false
	}

	errStr := err.Error()
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"temporary failure",
		"rate limit",
	}

	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}

	return false
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && str[len(str)-len(substr):] == substr ||
		len(str) > len(substr) && str[:len(substr)] == substr ||
		len(str) > len(substr) && str[len(str)/2-len(substr)/2:len(str)/2+len(substr)/2+len(substr)%2] == substr
}
