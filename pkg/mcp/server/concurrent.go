// Package server provides concurrent request handling optimizations for MCP
package server

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	
	"mcp-memory/pkg/mcp/protocol"
)

// ConcurrentHandler provides optimized concurrent request handling
type ConcurrentHandler struct {
	// Worker pool configuration
	numWorkers   int
	maxQueueSize int
	
	// Request channels
	requestChan  chan *requestWork
	responseChan chan *responseWork
	
	// Worker management
	workers    []*worker
	workerPool sync.Pool
	wg         sync.WaitGroup
	
	// Metrics
	activeRequests   int64
	totalRequests    int64
	rejectedRequests int64
	
	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
	
	// Request handler
	handler RequestHandler
}

// RequestHandler defines the interface for handling requests
type RequestHandler interface {
	HandleRequest(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error)
}

// requestWork represents a unit of work
type requestWork struct {
	ctx      context.Context
	request  *protocol.JSONRPCRequest
	response chan<- *responseWork
}

// responseWork represents a response
type responseWork struct {
	response *protocol.JSONRPCResponse
	err      error
}

// worker represents a worker goroutine
type worker struct {
	id          int
	handler     *ConcurrentHandler
	activeWork  atomic.Value
}

// ConcurrentOptions configures the concurrent handler
type ConcurrentOptions struct {
	NumWorkers   int
	MaxQueueSize int
}

// DefaultConcurrentOptions returns default options
func DefaultConcurrentOptions() *ConcurrentOptions {
	return &ConcurrentOptions{
		NumWorkers:   runtime.NumCPU() * 2,
		MaxQueueSize: 1000,
	}
}

// NewConcurrentHandler creates a new concurrent handler
func NewConcurrentHandler(handler RequestHandler, opts *ConcurrentOptions) *ConcurrentHandler {
	if opts == nil {
		opts = DefaultConcurrentOptions()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	ch := &ConcurrentHandler{
		numWorkers:   opts.NumWorkers,
		maxQueueSize: opts.MaxQueueSize,
		requestChan:  make(chan *requestWork, opts.MaxQueueSize),
		responseChan: make(chan *responseWork, opts.MaxQueueSize),
		workers:      make([]*worker, opts.NumWorkers),
		handler:      handler,
		ctx:          ctx,
		cancel:       cancel,
	}
	
	// Initialize worker pool
	ch.workerPool = sync.Pool{
		New: func() interface{} {
			return &worker{
				handler: ch,
			}
		},
	}
	
	// Start workers
	for i := 0; i < ch.numWorkers; i++ {
		w := &worker{
			id:      i,
			handler: ch,
		}
		ch.workers[i] = w
		ch.wg.Add(1)
		go w.run()
	}
	
	return ch
}

// HandleRequest processes a request concurrently
func (ch *ConcurrentHandler) HandleRequest(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
	// Increment metrics
	atomic.AddInt64(&ch.totalRequests, 1)
	
	// Check if we're at capacity
	if atomic.LoadInt64(&ch.activeRequests) >= int64(ch.maxQueueSize) {
		atomic.AddInt64(&ch.rejectedRequests, 1)
		return nil, protocol.NewJSONRPCError(protocol.InternalError, "server at capacity", nil)
	}
	
	// Increment active requests
	atomic.AddInt64(&ch.activeRequests, 1)
	defer atomic.AddInt64(&ch.activeRequests, -1)
	
	// Create response channel
	respChan := make(chan *responseWork, 1)
	
	// Create work item
	work := &requestWork{
		ctx:      ctx,
		request:  req,
		response: respChan,
	}
	
	// Submit work
	select {
	case ch.requestChan <- work:
		// Work submitted
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ch.ctx.Done():
		return nil, context.Canceled
	}
	
	// Wait for response
	select {
	case resp := <-respChan:
		return resp.response, resp.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ch.ctx.Done():
		return nil, context.Canceled
	}
}

// Shutdown gracefully shuts down the concurrent handler
func (ch *ConcurrentHandler) Shutdown(timeout time.Duration) error {
	// Signal shutdown
	ch.cancel()
	
	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		ch.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}

// Metrics returns current metrics
func (ch *ConcurrentHandler) Metrics() map[string]int64 {
	return map[string]int64{
		"active_requests":   atomic.LoadInt64(&ch.activeRequests),
		"total_requests":    atomic.LoadInt64(&ch.totalRequests),
		"rejected_requests": atomic.LoadInt64(&ch.rejectedRequests),
		"num_workers":       int64(ch.numWorkers),
		"max_queue_size":    int64(ch.maxQueueSize),
	}
}

// run is the worker loop
func (w *worker) run() {
	defer w.handler.wg.Done()
	
	for {
		select {
		case work := <-w.handler.requestChan:
			w.processWork(work)
		case <-w.handler.ctx.Done():
			return
		}
	}
}

// processWork handles a single work item
func (w *worker) processWork(work *requestWork) {
	// Store active work for monitoring
	w.activeWork.Store(work)
	defer w.activeWork.Store(nil)
	
	// Handle the request
	resp, err := w.handler.handler.HandleRequest(work.ctx, work.request)
	
	// Send response
	select {
	case work.response <- &responseWork{response: resp, err: err}:
		// Response sent
	case <-work.ctx.Done():
		// Context cancelled
	case <-w.handler.ctx.Done():
		// Handler shutting down
	}
}

// BatchProcessor handles multiple requests in batches for improved throughput
type BatchProcessor struct {
	handler      RequestHandler
	batchSize    int
	batchTimeout time.Duration
	
	mu       sync.Mutex
	batch    []*batchItem
	timer    *time.Timer
	resultCh chan *batchResult
}

// batchItem represents an item in a batch
type batchItem struct {
	ctx      context.Context
	request  *protocol.JSONRPCRequest
	resultCh chan<- *batchResult
}

// batchResult represents a batch result
type batchResult struct {
	response *protocol.JSONRPCResponse
	err      error
}

// BatchOptions configures batch processing
type BatchOptions struct {
	BatchSize    int
	BatchTimeout time.Duration
}

// DefaultBatchOptions returns default batch options
func DefaultBatchOptions() *BatchOptions {
	return &BatchOptions{
		BatchSize:    50,
		BatchTimeout: 10 * time.Millisecond,
	}
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(handler RequestHandler, opts *BatchOptions) *BatchProcessor {
	if opts == nil {
		opts = DefaultBatchOptions()
	}
	
	bp := &BatchProcessor{
		handler:      handler,
		batchSize:    opts.BatchSize,
		batchTimeout: opts.BatchTimeout,
		batch:        make([]*batchItem, 0, opts.BatchSize),
		resultCh:     make(chan *batchResult, opts.BatchSize),
	}
	
	return bp
}

// HandleRequest adds a request to the batch
func (bp *BatchProcessor) HandleRequest(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
	resultCh := make(chan *batchResult, 1)
	
	bp.mu.Lock()
	
	// Add to batch
	bp.batch = append(bp.batch, &batchItem{
		ctx:      ctx,
		request:  req,
		resultCh: resultCh,
	})
	
	// Check if batch is full
	if len(bp.batch) >= bp.batchSize {
		bp.processBatchLocked()
	} else if bp.timer == nil {
		// Start timer for batch timeout
		bp.timer = time.AfterFunc(bp.batchTimeout, func() {
			bp.mu.Lock()
			defer bp.mu.Unlock()
			bp.processBatchLocked()
		})
	}
	
	bp.mu.Unlock()
	
	// Wait for result
	select {
	case result := <-resultCh:
		return result.response, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// processBatchLocked processes the current batch
func (bp *BatchProcessor) processBatchLocked() {
	if len(bp.batch) == 0 {
		return
	}
	
	// Stop timer if running
	if bp.timer != nil {
		bp.timer.Stop()
		bp.timer = nil
	}
	
	// Process batch in parallel
	batch := bp.batch
	bp.batch = make([]*batchItem, 0, bp.batchSize)
	
	// Process each item in the batch concurrently
	var wg sync.WaitGroup
	for _, item := range batch {
		wg.Add(1)
		go func(item *batchItem) {
			defer wg.Done()
			
			resp, err := bp.handler.HandleRequest(item.ctx, item.request)
			
			select {
			case item.resultCh <- &batchResult{response: resp, err: err}:
				// Result sent
			case <-item.ctx.Done():
				// Context cancelled
			}
		}(item)
	}
	
	// Wait for all items to complete
	wg.Wait()
}

// PipelineHandler provides request pipelining for improved latency
type PipelineHandler struct {
	handler  RequestHandler
	pipeline chan *pipelineWork
	workers  int
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// pipelineWork represents work in the pipeline
type pipelineWork struct {
	ctx      context.Context
	request  *protocol.JSONRPCRequest
	resultCh chan<- *pipelineResult
}

// pipelineResult represents a pipeline result
type pipelineResult struct {
	response *protocol.JSONRPCResponse
	err      error
}

// NewPipelineHandler creates a new pipeline handler
func NewPipelineHandler(handler RequestHandler, workers int) *PipelineHandler {
	ctx, cancel := context.WithCancel(context.Background())
	
	ph := &PipelineHandler{
		handler:  handler,
		pipeline: make(chan *pipelineWork, workers*2),
		workers:  workers,
		ctx:      ctx,
		cancel:   cancel,
	}
	
	// Start pipeline workers
	for i := 0; i < workers; i++ {
		ph.wg.Add(1)
		go ph.pipelineWorker()
	}
	
	return ph
}

// HandleRequest processes a request through the pipeline
func (ph *PipelineHandler) HandleRequest(ctx context.Context, req *protocol.JSONRPCRequest) (*protocol.JSONRPCResponse, error) {
	resultCh := make(chan *pipelineResult, 1)
	
	work := &pipelineWork{
		ctx:      ctx,
		request:  req,
		resultCh: resultCh,
	}
	
	select {
	case ph.pipeline <- work:
		// Work submitted
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ph.ctx.Done():
		return nil, context.Canceled
	}
	
	select {
	case result := <-resultCh:
		return result.response, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ph.ctx.Done():
		return nil, context.Canceled
	}
}

// pipelineWorker processes work from the pipeline
func (ph *PipelineHandler) pipelineWorker() {
	defer ph.wg.Done()
	
	for {
		select {
		case work := <-ph.pipeline:
			resp, err := ph.handler.HandleRequest(work.ctx, work.request)
			
			select {
			case work.resultCh <- &pipelineResult{response: resp, err: err}:
				// Result sent
			case <-work.ctx.Done():
				// Context cancelled
			}
		case <-ph.ctx.Done():
			return
		}
	}
}

// Shutdown gracefully shuts down the pipeline handler
func (ph *PipelineHandler) Shutdown() {
	ph.cancel()
	ph.wg.Wait()
}