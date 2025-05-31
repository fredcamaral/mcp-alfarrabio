package bulk

import (
	"context"
	"fmt"
	"log"
	"mcp-memory/internal/storage"
	"mcp-memory/pkg/types"
	"sync"
	"time"
)

// Operation represents the type of bulk operation
type Operation string

const (
	OperationStore  Operation = "store"
	OperationUpdate Operation = "update"
	OperationDelete Operation = "delete"
)

// OperationStatus represents the status of a bulk operation
type OperationStatus string

const (
	StatusPending    OperationStatus = "pending"
	StatusRunning    OperationStatus = "running"
	StatusCompleted  OperationStatus = "completed"
	StatusFailed     OperationStatus = "failed"
	StatusCancelled  OperationStatus = "cancelled"
)

// BulkRequest represents a bulk operation request
type BulkRequest struct {
	ID          string                     `json:"id"`
	Operation   Operation                  `json:"operation"`
	Chunks      []types.ConversationChunk  `json:"chunks,omitempty"`
	IDs         []string                   `json:"ids,omitempty"` // For delete operations
	Options     BulkOptions                `json:"options"`
	CreatedAt   time.Time                  `json:"created_at"`
	StartedAt   *time.Time                 `json:"started_at,omitempty"`
	CompletedAt *time.Time                 `json:"completed_at,omitempty"`
}

// BulkOptions configures bulk operation behavior
type BulkOptions struct {
	BatchSize        int               `json:"batch_size"`         // Number of items per batch
	MaxConcurrency   int               `json:"max_concurrency"`    // Max concurrent batches
	ValidateFirst    bool              `json:"validate_first"`     // Validate all items before processing
	ContinueOnError  bool              `json:"continue_on_error"`  // Continue processing if individual items fail
	DryRun           bool              `json:"dry_run"`            // Preview operation without executing
	ConflictPolicy   ConflictPolicy    `json:"conflict_policy"`    // How to handle conflicts
	ProgressCallback func(BulkProgress) `json:"-"`                 // Progress callback function
}

// ConflictPolicy defines how to handle conflicts during bulk operations
type ConflictPolicy string

const (
	ConflictPolicySkip      ConflictPolicy = "skip"       // Skip conflicting items
	ConflictPolicyOverwrite ConflictPolicy = "overwrite"  // Overwrite existing items
	ConflictPolicyMerge     ConflictPolicy = "merge"      // Merge with existing items
	ConflictPolicyFail      ConflictPolicy = "fail"       // Fail the entire operation on conflict
)

// BulkProgress represents the progress of a bulk operation
type BulkProgress struct {
	OperationID      string          `json:"operation_id"`
	Status           OperationStatus `json:"status"`
	TotalItems       int             `json:"total_items"`
	ProcessedItems   int             `json:"processed_items"`
	SuccessfulItems  int             `json:"successful_items"`
	FailedItems      int             `json:"failed_items"`
	SkippedItems     int             `json:"skipped_items"`
	CurrentBatch     int             `json:"current_batch"`
	TotalBatches     int             `json:"total_batches"`
	StartTime        time.Time       `json:"start_time"`
	ElapsedTime      time.Duration   `json:"elapsed_time"`
	EstimatedTime    time.Duration   `json:"estimated_time"`
	Errors           []BulkError     `json:"errors,omitempty"`
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
}

// BulkError represents an error that occurred during bulk processing
type BulkError struct {
	ItemIndex int    `json:"item_index"`
	ItemID    string `json:"item_id,omitempty"`
	Error     string `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

// ValidationError represents a validation error
type ValidationError struct {
	ItemIndex int    `json:"item_index"`
	Field     string `json:"field"`
	Message   string `json:"message"`
}

// BulkResult represents the final result of a bulk operation
type BulkResult struct {
	Progress     BulkProgress `json:"progress"`
	ProcessedIDs []string     `json:"processed_ids,omitempty"`
	Summary      string       `json:"summary"`
}

// Manager handles bulk operations on memory chunks
type Manager struct {
	storage       storage.VectorStore
	operations    map[string]*BulkRequest
	operationsMux sync.RWMutex
	logger        *log.Logger
}

// NewManager creates a new bulk operations manager
func NewManager(storage storage.VectorStore, logger *log.Logger) *Manager {
	if logger == nil {
		logger = log.New(log.Writer(), "[BulkManager] ", log.LstdFlags)
	}

	return &Manager{
		storage:    storage,
		operations: make(map[string]*BulkRequest),
		logger:     logger,
	}
}

// SubmitOperation submits a new bulk operation
func (m *Manager) SubmitOperation(ctx context.Context, req BulkRequest) (*BulkProgress, error) {
	// Set defaults
	if req.Options.BatchSize <= 0 {
		req.Options.BatchSize = 50
	}
	if req.Options.MaxConcurrency <= 0 {
		req.Options.MaxConcurrency = 3
	}

	// Generate ID if not provided
	if req.ID == "" {
		req.ID = generateOperationID()
	}

	req.CreatedAt = time.Now().UTC()

	// Store operation
	m.operationsMux.Lock()
	m.operations[req.ID] = &req
	m.operationsMux.Unlock()

	// Start processing asynchronously
	go func() {
		if err := m.processOperation(ctx, req.ID); err != nil {
			m.logger.Printf("Error processing bulk operation %s: %v", req.ID, err)
		}
	}()

	// Return initial progress
	return &BulkProgress{
		OperationID:    req.ID,
		Status:         StatusPending,
		TotalItems:     m.getTotalItems(req),
		StartTime:      req.CreatedAt,
		TotalBatches:   m.calculateTotalBatches(req),
	}, nil
}

// GetProgress returns the current progress of an operation
func (m *Manager) GetProgress(operationID string) (*BulkProgress, error) {
	m.operationsMux.RLock()
	op, exists := m.operations[operationID]
	m.operationsMux.RUnlock()

	if !exists {
		return nil, fmt.Errorf("operation %s not found", operationID)
	}

	return m.buildProgress(op), nil
}

// CancelOperation cancels a running operation
func (m *Manager) CancelOperation(operationID string) error {
	m.operationsMux.Lock()
	defer m.operationsMux.Unlock()

	op, exists := m.operations[operationID]
	if !exists {
		return fmt.Errorf("operation %s not found", operationID)
	}

	// Mark as cancelled (actual cancellation would require context cancellation)
	op.CompletedAt = timePtr(time.Now().UTC())
	return nil
}

// ListOperations returns a list of all operations with optional filtering
func (m *Manager) ListOperations(status *OperationStatus, limit int) ([]*BulkProgress, error) {
	m.operationsMux.RLock()
	defer m.operationsMux.RUnlock()

	var results []*BulkProgress
	for _, op := range m.operations {
		progress := m.buildProgress(op)
		
		if status != nil && progress.Status != *status {
			continue
		}

		results = append(results, progress)
		
		if limit > 0 && len(results) >= limit {
			break
		}
	}

	return results, nil
}

// processOperation processes a bulk operation
func (m *Manager) processOperation(ctx context.Context, operationID string) error {
	m.operationsMux.Lock()
	op := m.operations[operationID]
	now := time.Now().UTC()
	op.StartedAt = &now
	m.operationsMux.Unlock()

	defer func() {
		m.operationsMux.Lock()
		completedAt := time.Now().UTC()
		op.CompletedAt = &completedAt
		m.operationsMux.Unlock()
	}()

	progress := m.buildProgress(op)
	progress.Status = StatusRunning

	// Validation phase
	if op.Options.ValidateFirst {
		if err := m.validateOperation(op, progress); err != nil {
			progress.Status = StatusFailed
			m.notifyProgress(op, progress)
			return err
		}
	}

	// Dry run
	if op.Options.DryRun {
		return m.performDryRun(op, progress)
	}

	// Execute operation
	switch op.Operation {
	case OperationStore:
		return m.executeBulkStore(ctx, op, progress)
	case OperationUpdate:
		return m.executeBulkUpdate(ctx, op, progress)
	case OperationDelete:
		return m.executeBulkDelete(ctx, op, progress)
	default:
		return fmt.Errorf("unsupported operation: %s", op.Operation)
	}
}

// validateOperation validates all items in the operation
func (m *Manager) validateOperation(op *BulkRequest, progress *BulkProgress) error {
	for i, chunk := range op.Chunks {
		if err := chunk.Validate(); err != nil {
			progress.ValidationErrors = append(progress.ValidationErrors, ValidationError{
				ItemIndex: i,
				Field:     "chunk",
				Message:   err.Error(),
			})
		}
	}

	if len(progress.ValidationErrors) > 0 && !op.Options.ContinueOnError {
		return fmt.Errorf("validation failed with %d errors", len(progress.ValidationErrors))
	}

	return nil
}

// performDryRun simulates the operation without executing it
func (m *Manager) performDryRun(op *BulkRequest, progress *BulkProgress) error {
	m.logger.Printf("Performing dry run for operation %s", op.ID)
	
	progress.Status = StatusCompleted
	progress.ProcessedItems = progress.TotalItems
	progress.SuccessfulItems = progress.TotalItems
	
	m.notifyProgress(op, progress)
	return nil
}

// executeBulkStore executes a bulk store operation
func (m *Manager) executeBulkStore(ctx context.Context, op *BulkRequest, progress *BulkProgress) error {
	batches := m.createBatches(op.Chunks, op.Options.BatchSize)
	progress.TotalBatches = len(batches)

	// Process batches with concurrency control
	semaphore := make(chan struct{}, op.Options.MaxConcurrency)
	// Use generic batch processor
	err := m.processBatchesStore(ctx, batches, op, progress, semaphore)
	if err != nil {
		return err
	}
	return nil
}

// executeBulkUpdate executes a bulk update operation
func (m *Manager) executeBulkUpdate(ctx context.Context, op *BulkRequest, progress *BulkProgress) error {
	// Similar to store but uses Update method
	for i, chunk := range op.Chunks {
		if err := m.storage.Update(ctx, chunk); err != nil {
			progress.Errors = append(progress.Errors, BulkError{
				ItemIndex: i,
				ItemID:    chunk.ID,
				Error:     err.Error(),
				Timestamp: time.Now().UTC(),
			})
			
			if !op.Options.ContinueOnError {
				progress.Status = StatusFailed
				return err
			}
			progress.FailedItems++
		} else {
			progress.SuccessfulItems++
		}
		
		progress.ProcessedItems++
		progress.ElapsedTime = time.Since(progress.StartTime)
		progress.EstimatedTime = m.estimateRemainingTime(progress)
		
		m.notifyProgress(op, progress)
	}

	progress.Status = StatusCompleted
	m.notifyProgress(op, progress)
	return nil
}

// executeBulkDelete executes a bulk delete operation
func (m *Manager) executeBulkDelete(ctx context.Context, op *BulkRequest, progress *BulkProgress) error {
	batches := m.createIDBatches(op.IDs, op.Options.BatchSize)
	progress.TotalBatches = len(batches)

	// Process batches with concurrency control
	semaphore := make(chan struct{}, op.Options.MaxConcurrency)
	
	// Use generic batch processor
	err := m.processBatchesDelete(ctx, batches, op, progress, semaphore)
	if err != nil {
		return err
	}
	return nil
}

// Helper methods

func (m *Manager) getTotalItems(req BulkRequest) int {
	switch req.Operation {
	case OperationStore, OperationUpdate:
		return len(req.Chunks)
	case OperationDelete:
		return len(req.IDs)
	default:
		return 0
	}
}

func (m *Manager) calculateTotalBatches(req BulkRequest) int {
	totalItems := m.getTotalItems(req)
	batchSize := req.Options.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}
	return (totalItems + batchSize - 1) / batchSize
}

func (m *Manager) createBatches(chunks []types.ConversationChunk, batchSize int) [][]types.ConversationChunk {
	var batches [][]types.ConversationChunk
	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		batches = append(batches, chunks[i:end])
	}
	return batches
}

func (m *Manager) createIDBatches(ids []string, batchSize int) [][]string {
	var batches [][]string
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batches = append(batches, ids[i:end])
	}
	return batches
}

func (m *Manager) buildProgress(op *BulkRequest) *BulkProgress {
	status := StatusPending
	if op.StartedAt != nil {
		status = StatusRunning
	}
	if op.CompletedAt != nil {
		status = StatusCompleted
	}

	progress := &BulkProgress{
		OperationID:  op.ID,
		Status:       status,
		TotalItems:   m.getTotalItems(*op),
		TotalBatches: m.calculateTotalBatches(*op),
		StartTime:    op.CreatedAt,
	}

	if op.StartedAt != nil {
		progress.ElapsedTime = time.Since(*op.StartedAt)
	}

	return progress
}

func (m *Manager) estimateRemainingTime(progress *BulkProgress) time.Duration {
	if progress.ProcessedItems == 0 {
		return 0
	}

	avgTimePerItem := progress.ElapsedTime / time.Duration(progress.ProcessedItems)
	remainingItems := progress.TotalItems - progress.ProcessedItems
	return avgTimePerItem * time.Duration(remainingItems)
}

func (m *Manager) notifyProgress(op *BulkRequest, progress *BulkProgress) {
	if op.Options.ProgressCallback != nil {
		op.Options.ProgressCallback(*progress)
	}
}

// Utility functions

func generateOperationID() string {
	return fmt.Sprintf("bulk_%d", time.Now().UnixNano())
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// BulkOperationRequest represents an external API request for bulk operations
type BulkOperationRequest struct {
	Operation      string                     `json:"operation"`
	Chunks         []types.ConversationChunk  `json:"chunks,omitempty"`
	IDs            []string                   `json:"ids,omitempty"`
	BatchSize      int                        `json:"batch_size,omitempty"`
	MaxConcurrency int                        `json:"max_concurrency,omitempty"`
	ValidateFirst  bool                       `json:"validate_first,omitempty"`
	ContinueOnError bool                      `json:"continue_on_error,omitempty"`
	DryRun         bool                       `json:"dry_run,omitempty"`
	ConflictPolicy string                     `json:"conflict_policy,omitempty"`
}

// ToBulkRequest converts an external request to internal BulkRequest
func (req *BulkOperationRequest) ToBulkRequest() (BulkRequest, error) {
	operation := Operation(req.Operation)
	switch operation {
	case OperationStore, OperationUpdate, OperationDelete:
		// Valid operation
	default:
		return BulkRequest{}, fmt.Errorf("invalid operation: %s", req.Operation)
	}

	conflictPolicy := ConflictPolicySkip
	if req.ConflictPolicy != "" {
		conflictPolicy = ConflictPolicy(req.ConflictPolicy)
	}

	return BulkRequest{
		Operation: operation,
		Chunks:    req.Chunks,
		IDs:       req.IDs,
		Options: BulkOptions{
			BatchSize:       req.BatchSize,
			MaxConcurrency:  req.MaxConcurrency,
			ValidateFirst:   req.ValidateFirst,
			ContinueOnError: req.ContinueOnError,
			DryRun:          req.DryRun,
			ConflictPolicy:  conflictPolicy,
		},
	}, nil
}

// processBatchesStore processes batches for store operations with concurrency control
func (m *Manager) processBatchesStore(ctx context.Context, batches [][]types.ConversationChunk, op *BulkRequest, progress *BulkProgress, semaphore chan struct{}) error {
	var wg sync.WaitGroup
	
	for i, batch := range batches {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case semaphore <- struct{}{}: // Acquire semaphore
		}
		
		wg.Add(1)
		go func(batchIndex int, chunks []types.ConversationChunk) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
			}()
			
			for j, chunk := range chunks {
				if err := m.storage.Store(ctx, chunk); err != nil {
					progress.Errors = append(progress.Errors, BulkError{
						ItemIndex: batchIndex*op.Options.BatchSize + j,
						ItemID:    chunk.ID,
						Error:     err.Error(),
						Timestamp: time.Now().UTC(),
					})
					
					if !op.Options.ContinueOnError {
						progress.Status = StatusFailed
						return
					}
					progress.FailedItems++
				} else {
					progress.SuccessfulItems++
				}
				
				progress.ProcessedItems++
				progress.ElapsedTime = time.Since(progress.StartTime)
				progress.EstimatedTime = m.estimateRemainingTime(progress)
				progress.CurrentBatch = batchIndex + 1
				
				m.notifyProgress(op, progress)
			}
		}(i, batch)
	}
	
	wg.Wait()
	
	progress.Status = StatusCompleted
	m.notifyProgress(op, progress)
	return nil
}

// processBatchesDelete processes batches for delete operations with concurrency control
func (m *Manager) processBatchesDelete(ctx context.Context, batches [][]string, op *BulkRequest, progress *BulkProgress, semaphore chan struct{}) error {
	var wg sync.WaitGroup
	
	for i, batch := range batches {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case semaphore <- struct{}{}: // Acquire semaphore
		}
		
		wg.Add(1)
		go func(batchIndex int, ids []string) {
			defer func() {
				<-semaphore // Release semaphore
				wg.Done()
			}()
			
			for j, id := range ids {
				if err := m.storage.Delete(ctx, id); err != nil {
					progress.Errors = append(progress.Errors, BulkError{
						ItemIndex: batchIndex*op.Options.BatchSize + j,
						ItemID:    id,
						Error:     err.Error(),
						Timestamp: time.Now().UTC(),
					})
					
					if !op.Options.ContinueOnError {
						progress.Status = StatusFailed
						return
					}
					progress.FailedItems++
				} else {
					progress.SuccessfulItems++
				}
				
				progress.ProcessedItems++
				progress.ElapsedTime = time.Since(progress.StartTime)
				progress.EstimatedTime = m.estimateRemainingTime(progress)
				progress.CurrentBatch = batchIndex + 1
				
				m.notifyProgress(op, progress)
			}
		}(i, batch)
	}
	
	wg.Wait()
	
	progress.Status = StatusCompleted
	m.notifyProgress(op, progress)
	return nil
}