// Package bulk provides bulk operations for memory management,
// including batch importing, exporting, and alias management.
package bulk

import (
	"context"
	"errors"
	"fmt"
	"log"
	"lerian-mcp-memory/internal/storage"
	"lerian-mcp-memory/pkg/types"
	"sync"
	"time"
)

// Operation represents the type of bulk operation
type Operation string

const (
	// OperationStore represents a bulk store operation
	OperationStore Operation = "store"
	// OperationUpdate represents a bulk update operation
	OperationUpdate Operation = "update"
	// OperationDelete represents a bulk delete operation
	OperationDelete Operation = "delete"
)

// OperationStatus represents the status of a bulk operation
type OperationStatus string

const (
	// StatusPending indicates an operation is pending
	StatusPending OperationStatus = "pending"
	// StatusRunning indicates an operation is currently running
	StatusRunning OperationStatus = "running"
	// StatusCompleted indicates an operation has completed successfully
	StatusCompleted OperationStatus = "completed"
	// StatusFailed indicates an operation has failed
	StatusFailed OperationStatus = "failed"
	// StatusCancelled indicates an operation was cancelled
	StatusCancelled OperationStatus = "cancelled"
)

// Request represents a bulk operation request
type Request struct {
	ID          string                    `json:"id"`
	Operation   Operation                 `json:"operation"`
	Chunks      []types.ConversationChunk `json:"chunks,omitempty"`
	IDs         []string                  `json:"ids,omitempty"` // For delete operations
	Options     Options                   `json:"options"`
	CreatedAt   time.Time                 `json:"created_at"`
	StartedAt   *time.Time                `json:"started_at,omitempty"`
	CompletedAt *time.Time                `json:"completed_at,omitempty"`
}

// Options configures bulk operation behavior
type Options struct {
	BatchSize        int            `json:"batch_size"`        // Number of items per batch
	MaxConcurrency   int            `json:"max_concurrency"`   // Max concurrent batches
	ValidateFirst    bool           `json:"validate_first"`    // Validate all items before processing
	ContinueOnError  bool           `json:"continue_on_error"` // Continue processing if individual items fail
	DryRun           bool           `json:"dry_run"`           // Preview operation without executing
	ConflictPolicy   ConflictPolicy `json:"conflict_policy"`   // How to handle conflicts
	ProgressCallback func(Progress) `json:"-"`                 // Progress callback function
}

// ConflictPolicy defines how to handle conflicts during bulk operations
type ConflictPolicy string

const (
	// ConflictPolicySkip skips conflicting items
	ConflictPolicySkip ConflictPolicy = "skip" // Skip conflicting items
	// ConflictPolicyOverwrite overwrites existing items
	ConflictPolicyOverwrite ConflictPolicy = "overwrite" // Overwrite existing items
	// ConflictPolicyMerge merges with existing items
	ConflictPolicyMerge ConflictPolicy = "merge" // Merge with existing items
	// ConflictPolicyFail fails the entire operation on conflict
	ConflictPolicyFail ConflictPolicy = "fail" // Fail the entire operation on conflict
)

// Progress represents the progress of a bulk operation
type Progress struct {
	OperationID      string            `json:"operation_id"`
	Status           OperationStatus   `json:"status"`
	TotalItems       int               `json:"total_items"`
	ProcessedItems   int               `json:"processed_items"`
	SuccessfulItems  int               `json:"successful_items"`
	FailedItems      int               `json:"failed_items"`
	SkippedItems     int               `json:"skipped_items"`
	CurrentBatch     int               `json:"current_batch"`
	TotalBatches     int               `json:"total_batches"`
	StartTime        time.Time         `json:"start_time"`
	ElapsedTime      time.Duration     `json:"elapsed_time"`
	EstimatedTime    time.Duration     `json:"estimated_time"`
	Errors           []Error           `json:"errors,omitempty"`
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
}

// Error represents an error that occurred during bulk processing
type Error struct {
	ItemIndex int       `json:"item_index"`
	ItemID    string    `json:"item_id,omitempty"`
	Error     string    `json:"error"`
	Timestamp time.Time `json:"timestamp"`
}

// ValidationError represents a validation error
type ValidationError struct {
	ItemIndex int    `json:"item_index"`
	Field     string `json:"field"`
	Message   string `json:"message"`
}

// Result represents the final result of a bulk operation
type Result struct {
	Progress     Progress `json:"progress"`
	ProcessedIDs []string `json:"processed_ids,omitempty"`
	Summary      string   `json:"summary"`
}

// Manager handles bulk operations on memory chunks
type Manager struct {
	storage       storage.VectorStore
	operations    map[string]*Request
	operationsMux sync.RWMutex
	logger        *log.Logger
}

// NewManager creates a new bulk operations manager
func NewManager(vectorStore storage.VectorStore, logger *log.Logger) *Manager {
	if logger == nil {
		logger = log.New(log.Writer(), "[BulkManager] ", log.LstdFlags)
	}

	return &Manager{
		storage:    vectorStore,
		operations: make(map[string]*Request),
		logger:     logger,
	}
}

// SubmitOperation submits a new bulk operation
func (m *Manager) SubmitOperation(ctx context.Context, req *Request) (*Progress, error) {
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
	m.operations[req.ID] = req
	m.operationsMux.Unlock()

	// Return initial progress before starting async processing to avoid races
	initialProgress := &Progress{
		OperationID:  req.ID,
		Status:       StatusPending,
		TotalItems:   m.getTotalItems(req),
		StartTime:    req.CreatedAt,
		TotalBatches: m.calculateTotalBatches(req),
	}

	// Start processing asynchronously
	go func() {
		if err := m.processOperation(ctx, req.ID); err != nil {
			m.logger.Printf("Error processing bulk operation %s: %v", req.ID, err)
		}
	}()

	return initialProgress, nil
}

// GetProgress returns the current progress of an operation
func (m *Manager) GetProgress(operationID string) (*Progress, error) {
	m.operationsMux.RLock()
	op, exists := m.operations[operationID]
	if !exists {
		m.operationsMux.RUnlock()
		return nil, errors.New("operation " + operationID + " not found")
	}

	// Build progress while holding the read lock to avoid races
	progress := m.buildProgress(op)
	m.operationsMux.RUnlock()

	return progress, nil
}

// CancelOperation cancels a running operation
func (m *Manager) CancelOperation(operationID string) error {
	m.operationsMux.Lock()
	defer m.operationsMux.Unlock()

	op, exists := m.operations[operationID]
	if !exists {
		return errors.New("operation " + operationID + " not found")
	}

	// Mark as cancelled (actual cancellation would require context cancellation)
	op.CompletedAt = timePtr(time.Now().UTC())
	return nil
}

// ListOperations returns a list of all operations with optional filtering
func (m *Manager) ListOperations(status *OperationStatus, limit int) ([]*Progress, error) {
	m.operationsMux.RLock()
	defer m.operationsMux.RUnlock()

	capacity := len(m.operations)
	if limit > 0 && limit < capacity {
		capacity = limit
	}
	results := make([]*Progress, 0, capacity)
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
	op, exists := m.operations[operationID]
	if !exists {
		m.operationsMux.Unlock()
		return errors.New("operation " + operationID + " not found")
	}
	now := time.Now().UTC()
	op.StartedAt = &now
	m.operationsMux.Unlock()

	defer func() {
		m.operationsMux.Lock()
		defer m.operationsMux.Unlock()
		if op, exists := m.operations[operationID]; exists {
			completedAt := time.Now().UTC()
			op.CompletedAt = &completedAt
		}
	}()

	// Build progress under lock to avoid race conditions
	m.operationsMux.Lock()
	progress := m.buildProgress(op)
	progress.Status = StatusRunning
	m.operationsMux.Unlock()

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
		return errors.New("unsupported operation: " + string(op.Operation))
	}
}

// validateOperation validates all items in the operation
func (m *Manager) validateOperation(op *Request, progress *Progress) error {
	for i := range op.Chunks {
		chunk := &op.Chunks[i]
		if err := chunk.Validate(); err != nil {
			progress.ValidationErrors = append(progress.ValidationErrors, ValidationError{
				ItemIndex: i,
				Field:     "chunk",
				Message:   err.Error(),
			})
		}
	}

	if len(progress.ValidationErrors) > 0 && !op.Options.ContinueOnError {
		return errors.New("validation failed with " + fmt.Sprint(len(progress.ValidationErrors)) + " errors")
	}

	return nil
}

// performDryRun simulates the operation without executing it
func (m *Manager) performDryRun(op *Request, progress *Progress) error {
	m.logger.Printf("Performing dry run for operation %s", op.ID)

	// Basic validation during dry run
	switch op.Operation {
	case OperationStore, OperationUpdate:
		if len(op.Chunks) == 0 {
			progress.Status = StatusFailed
			m.notifyProgress(op, progress)
			return errors.New("no chunks provided for " + string(op.Operation) + " operation")
		}
	case OperationDelete:
		if len(op.IDs) == 0 {
			progress.Status = StatusFailed
			m.notifyProgress(op, progress)
			return errors.New("no IDs provided for delete operation")
		}
	default:
		progress.Status = StatusFailed
		m.notifyProgress(op, progress)
		return errors.New("unknown operation: " + string(op.Operation))
	}

	progress.Status = StatusCompleted
	progress.ProcessedItems = progress.TotalItems
	progress.SuccessfulItems = progress.TotalItems

	m.notifyProgress(op, progress)
	return nil
}

// executeBulkStore executes a bulk store operation
func (m *Manager) executeBulkStore(ctx context.Context, op *Request, progress *Progress) error {
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
func (m *Manager) executeBulkUpdate(ctx context.Context, op *Request, progress *Progress) error {
	// Similar to store but uses Update method
	for i := range op.Chunks {
		chunk := &op.Chunks[i]
		if err := m.storage.Update(ctx, chunk); err != nil {
			progress.Errors = append(progress.Errors, Error{
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
func (m *Manager) executeBulkDelete(ctx context.Context, op *Request, progress *Progress) error {
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

func (m *Manager) getTotalItems(req *Request) int {
	switch req.Operation {
	case OperationStore, OperationUpdate:
		return len(req.Chunks)
	case OperationDelete:
		return len(req.IDs)
	default:
		return 0
	}
}

func (m *Manager) calculateTotalBatches(req *Request) int {
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

func (m *Manager) buildProgress(op *Request) *Progress {
	status := StatusPending
	if op.StartedAt != nil {
		status = StatusRunning
	}
	if op.CompletedAt != nil {
		status = StatusCompleted
	}

	progress := &Progress{
		OperationID:  op.ID,
		Status:       status,
		TotalItems:   m.getTotalItems(op),
		TotalBatches: m.calculateTotalBatches(op),
		StartTime:    op.CreatedAt,
	}

	if op.StartedAt != nil {
		progress.ElapsedTime = time.Since(*op.StartedAt)
	}

	return progress
}

func (m *Manager) estimateRemainingTime(progress *Progress) time.Duration {
	if progress.ProcessedItems == 0 {
		return 0
	}

	avgTimePerItem := progress.ElapsedTime / time.Duration(progress.ProcessedItems)
	remainingItems := progress.TotalItems - progress.ProcessedItems
	return avgTimePerItem * time.Duration(remainingItems)
}

func (m *Manager) notifyProgress(op *Request, progress *Progress) {
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

// OperationRequest represents an external API request for bulk operations
type OperationRequest struct {
	Operation       string                    `json:"operation"`
	Chunks          []types.ConversationChunk `json:"chunks,omitempty"`
	IDs             []string                  `json:"ids,omitempty"`
	BatchSize       int                       `json:"batch_size,omitempty"`
	MaxConcurrency  int                       `json:"max_concurrency,omitempty"`
	ValidateFirst   bool                      `json:"validate_first,omitempty"`
	ContinueOnError bool                      `json:"continue_on_error,omitempty"`
	DryRun          bool                      `json:"dry_run,omitempty"`
	ConflictPolicy  string                    `json:"conflict_policy,omitempty"`
}

// ToRequest converts an external request to internal Request
func (req *OperationRequest) ToRequest() (Request, error) {
	operation := Operation(req.Operation)
	switch operation {
	case OperationStore, OperationUpdate, OperationDelete:
		// Valid operation
	default:
		return Request{}, errors.New("invalid operation: " + req.Operation)
	}

	conflictPolicy := ConflictPolicySkip
	if req.ConflictPolicy != "" {
		conflictPolicy = ConflictPolicy(req.ConflictPolicy)
	}

	return Request{
		Operation: operation,
		Chunks:    req.Chunks,
		IDs:       req.IDs,
		Options: Options{
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
func (m *Manager) processBatchesStore(ctx context.Context, batches [][]types.ConversationChunk, op *Request, progress *Progress, semaphore chan struct{}) error {
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

			for j := range chunks {
				chunk := &chunks[j]
				if err := m.storage.Store(ctx, chunk); err != nil {
					progress.Errors = append(progress.Errors, Error{
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
func (m *Manager) processBatchesDelete(ctx context.Context, batches [][]string, op *Request, progress *Progress, semaphore chan struct{}) error {
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
					progress.Errors = append(progress.Errors, Error{
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
