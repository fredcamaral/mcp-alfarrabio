// Package events provides event persistence for reliability and replay
package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// EventStore provides event persistence and retrieval
type EventStore struct {
	db              *sql.DB
	config          *PersistenceConfig
	writeBuffer     chan *Event
	batchBuffer     []*Event
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	running         bool
	metrics         *PersistenceMetrics
	replicationChan chan *Event
}

// PersistenceConfig configures event persistence
type PersistenceConfig struct {
	DatabasePath      string        `json:"database_path"`
	BufferSize        int           `json:"buffer_size"`
	BatchSize         int           `json:"batch_size"`
	FlushInterval     time.Duration `json:"flush_interval"`
	RetentionPeriod   time.Duration `json:"retention_period"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	EnableCompression bool          `json:"enable_compression"`
	EnableReplication bool          `json:"enable_replication"`
	MaxDiskUsage      int64         `json:"max_disk_usage_bytes"`
	EnableBackup      bool          `json:"enable_backup"`
	BackupInterval    time.Duration `json:"backup_interval"`
	VerifyIntegrity   bool          `json:"verify_integrity"`
}

// PersistenceMetrics tracks persistence performance
type PersistenceMetrics struct {
	EventsStored      int64         `json:"events_stored"`
	EventsRetrieved   int64         `json:"events_retrieved"`
	EventsDeleted     int64         `json:"events_deleted"`
	BatchesProcessed  int64         `json:"batches_processed"`
	AverageWriteTime  time.Duration `json:"average_write_time"`
	AverageReadTime   time.Duration `json:"average_read_time"`
	TotalDiskUsage    int64         `json:"total_disk_usage_bytes"`
	LastCleanup       time.Time     `json:"last_cleanup"`
	LastBackup        time.Time     `json:"last_backup"`
	WriteThroughput   float64       `json:"write_throughput_per_sec"`
	ReadThroughput    float64       `json:"read_throughput_per_sec"`
	BufferUtilization float64       `json:"buffer_utilization_percent"`
	mu                sync.RWMutex
}

// EventQuery represents a query for retrieving events
type EventQuery struct {
	Types          []EventType `json:"types,omitempty"`
	Actions        []string    `json:"actions,omitempty"`
	Sources        []string    `json:"sources,omitempty"`
	Repositories   []string    `json:"repositories,omitempty"`
	SessionIDs     []string    `json:"session_ids,omitempty"`
	UserIDs        []string    `json:"user_ids,omitempty"`
	ClientIDs      []string    `json:"client_ids,omitempty"`
	Tags           []string    `json:"tags,omitempty"`
	After          *time.Time  `json:"after,omitempty"`
	Before         *time.Time  `json:"before,omitempty"`
	Limit          int         `json:"limit,omitempty"`
	Offset         int         `json:"offset,omitempty"`
	OrderBy        string      `json:"order_by,omitempty"`
	OrderDirection string      `json:"order_direction,omitempty"`
}

// EventReplay provides event replay functionality
type EventReplay struct {
	ID           string       `json:"id"`
	Query        *EventQuery  `json:"query"`
	StartTime    time.Time    `json:"start_time"`
	EndTime      *time.Time   `json:"end_time,omitempty"`
	EventCount   int64        `json:"event_count"`
	ReplaySpeed  float64      `json:"replay_speed"`
	Status       ReplayStatus `json:"status"`
	Progress     float64      `json:"progress"`
	CurrentEvent int64        `json:"current_event"`
	ErrorMessage string       `json:"error_message,omitempty"`
}

// ReplayStatus represents the status of an event replay
type ReplayStatus string

const (
	ReplayStatusPending   ReplayStatus = "pending"
	ReplayStatusRunning   ReplayStatus = "running"
	ReplayStatusCompleted ReplayStatus = "completed"
	ReplayStatusFailed    ReplayStatus = "failed"
	ReplayStatusCancelled ReplayStatus = "cancelled"
)

// DefaultPersistenceConfig returns default persistence configuration
func DefaultPersistenceConfig() *PersistenceConfig {
	return &PersistenceConfig{
		DatabasePath:      "events.db",
		BufferSize:        10000,
		BatchSize:         100,
		FlushInterval:     5 * time.Second,
		RetentionPeriod:   30 * 24 * time.Hour, // 30 days
		CleanupInterval:   time.Hour,
		EnableCompression: true,
		EnableReplication: false,
		MaxDiskUsage:      1024 * 1024 * 1024, // 1GB
		EnableBackup:      true,
		BackupInterval:    24 * time.Hour,
		VerifyIntegrity:   true,
	}
}

// NewEventStore creates a new event store
func NewEventStore(config *PersistenceConfig) (*EventStore, error) {
	if config == nil {
		config = DefaultPersistenceConfig()
	}

	// Open database
	db, err := sql.Open("sqlite3", config.DatabasePath+"?_journal_mode=WAL&_sync=NORMAL&_cache_size=10000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure database connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithCancel(context.Background())

	store := &EventStore{
		db:              db,
		config:          config,
		writeBuffer:     make(chan *Event, config.BufferSize),
		batchBuffer:     make([]*Event, 0, config.BatchSize),
		ctx:             ctx,
		cancel:          cancel,
		running:         false,
		metrics:         &PersistenceMetrics{},
		replicationChan: make(chan *Event, 1000),
	}

	// Initialize database schema
	if err := store.initDatabase(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return store, nil
}

// Start starts the event store
func (es *EventStore) Start() error {
	es.mu.Lock()
	defer es.mu.Unlock()

	if es.running {
		return errors.New("event store already running")
	}

	log.Printf("Starting event store with database: %s", es.config.DatabasePath)

	// Start write processor
	es.wg.Add(1)
	go es.writeProcessor()

	// Start cleanup routine
	es.wg.Add(1)
	go es.cleanupRoutine()

	// Start backup routine if enabled
	if es.config.EnableBackup {
		es.wg.Add(1)
		go es.backupRoutine()
	}

	// Start replication processor if enabled
	if es.config.EnableReplication {
		es.wg.Add(1)
		go es.replicationProcessor()
	}

	es.running = true
	log.Println("Event store started successfully")

	return nil
}

// Stop stops the event store gracefully
func (es *EventStore) Stop() error {
	es.mu.Lock()
	if !es.running {
		es.mu.Unlock()
		return errors.New("event store not running")
	}
	es.running = false
	es.mu.Unlock()

	log.Println("Stopping event store...")

	// Cancel context to signal routines to stop
	es.cancel()

	// Close write buffer
	close(es.writeBuffer)

	// Wait for all routines to finish
	es.wg.Wait()

	// Flush any remaining events
	if len(es.batchBuffer) > 0 {
		if err := es.flushBatch(); err != nil {
			log.Printf("Error flushing final batch: %v", err)
		}
	}

	// Close database
	if err := es.db.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Println("Event store stopped")
	return nil
}

// IsRunning returns whether the event store is running
func (es *EventStore) IsRunning() bool {
	es.mu.RLock()
	defer es.mu.RUnlock()
	return es.running
}

// Store stores an event
func (es *EventStore) Store(event *Event) error {
	if !es.IsRunning() {
		return errors.New("event store not running")
	}

	if event == nil {
		return errors.New("event cannot be nil")
	}

	select {
	case es.writeBuffer <- event:
		return nil
	default:
		// Buffer is full, drop event
		es.updateMetrics(func(m *PersistenceMetrics) {
			// Could track dropped events here
		})
		return errors.New("write buffer full, event dropped")
	}
}

// StoreBatch stores multiple events
func (es *EventStore) StoreBatch(events []*Event) error {
	if !es.IsRunning() {
		return errors.New("event store not running")
	}

	for _, event := range events {
		if err := es.Store(event); err != nil {
			return err
		}
	}

	return nil
}

// Retrieve retrieves events matching a query
func (es *EventStore) Retrieve(query *EventQuery) ([]*Event, error) {
	if !es.IsRunning() {
		return nil, errors.New("event store not running")
	}

	startTime := time.Now()

	sqlQuery, args := es.buildQuery(query)
	rows, err := es.db.QueryContext(es.ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	events := make([]*Event, 0)
	for rows.Next() {
		event, err := es.scanEvent(rows)
		if err != nil {
			log.Printf("Error scanning event: %v", err)
			continue
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Update metrics
	es.updateMetrics(func(m *PersistenceMetrics) {
		m.EventsRetrieved += int64(len(events))

		readTime := time.Since(startTime)
		if m.AverageReadTime == 0 {
			m.AverageReadTime = readTime
		} else {
			m.AverageReadTime = time.Duration(
				int64(m.AverageReadTime)*9/10 + int64(readTime)/10,
			)
		}
	})

	return events, nil
}

// GetEvent retrieves a single event by ID
func (es *EventStore) GetEvent(eventID string) (*Event, error) {
	if !es.IsRunning() {
		return nil, errors.New("event store not running")
	}

	sqlQuery := `SELECT id, type, action, version, timestamp, source, repository, session_id, 
		user_id, client_id, tags, correlation_id, causation_id, parent_id, payload, metadata, 
		sequence_number, ttl, expires_at, processed_at, delivered_at, acknowledged_at 
		FROM events WHERE id = ? LIMIT 1`

	row := es.db.QueryRowContext(es.ctx, sqlQuery, eventID)
	event, err := es.scanEventRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("event not found: %s", eventID)
		}
		return nil, fmt.Errorf("failed to retrieve event: %w", err)
	}

	es.updateMetrics(func(m *PersistenceMetrics) {
		m.EventsRetrieved++
	})

	return event, nil
}

// Delete deletes events matching a query
func (es *EventStore) Delete(query *EventQuery) (int64, error) {
	if !es.IsRunning() {
		return 0, errors.New("event store not running")
	}

	// Build delete query
	whereClause, args := es.buildWhereClause(query)
	sqlQuery := "DELETE FROM events"
	if whereClause != "" {
		sqlQuery += " WHERE " + whereClause
	}

	result, err := es.db.ExecContext(es.ctx, sqlQuery, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete events: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	es.updateMetrics(func(m *PersistenceMetrics) {
		m.EventsDeleted += rowsAffected
	})

	return rowsAffected, nil
}

// StartReplay starts event replay based on a query
func (es *EventStore) StartReplay(query *EventQuery, replaySpeed float64, callback func(*Event) error) (*EventReplay, error) {
	if !es.IsRunning() {
		return nil, errors.New("event store not running")
	}

	replay := &EventReplay{
		ID:          generateReplayID(),
		Query:       query,
		StartTime:   time.Now(),
		ReplaySpeed: replaySpeed,
		Status:      ReplayStatusPending,
		Progress:    0.0,
	}

	// Start replay in background
	go es.performReplay(replay, callback)

	return replay, nil
}

// GetMetrics returns persistence metrics
func (es *EventStore) GetMetrics() *PersistenceMetrics {
	es.metrics.mu.RLock()
	defer es.metrics.mu.RUnlock()

	// Update buffer utilization
	bufferUtilization := float64(len(es.writeBuffer)) / float64(cap(es.writeBuffer)) * 100

	return &PersistenceMetrics{
		EventsStored:      es.metrics.EventsStored,
		EventsRetrieved:   es.metrics.EventsRetrieved,
		EventsDeleted:     es.metrics.EventsDeleted,
		BatchesProcessed:  es.metrics.BatchesProcessed,
		AverageWriteTime:  es.metrics.AverageWriteTime,
		AverageReadTime:   es.metrics.AverageReadTime,
		TotalDiskUsage:    es.metrics.TotalDiskUsage,
		LastCleanup:       es.metrics.LastCleanup,
		LastBackup:        es.metrics.LastBackup,
		WriteThroughput:   es.metrics.WriteThroughput,
		ReadThroughput:    es.metrics.ReadThroughput,
		BufferUtilization: bufferUtilization,
	}
}

// initDatabase initializes the database schema
func (es *EventStore) initDatabase() error {
	schema := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		action TEXT NOT NULL,
		version TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		source TEXT NOT NULL,
		repository TEXT,
		session_id TEXT,
		user_id TEXT,
		client_id TEXT,
		tags TEXT,
		correlation_id TEXT,
		causation_id TEXT,
		parent_id TEXT,
		payload TEXT,
		metadata TEXT,
		sequence_number INTEGER,
		ttl INTEGER,
		expires_at DATETIME,
		processed_at DATETIME,
		delivered_at DATETIME,
		acknowledged_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
	CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
	CREATE INDEX IF NOT EXISTS idx_events_source ON events(source);
	CREATE INDEX IF NOT EXISTS idx_events_repository ON events(repository);
	CREATE INDEX IF NOT EXISTS idx_events_session_id ON events(session_id);
	CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id);
	CREATE INDEX IF NOT EXISTS idx_events_created_at ON events(created_at);
	CREATE INDEX IF NOT EXISTS idx_events_expires_at ON events(expires_at);
	`

	_, err := es.db.Exec(schema)
	return err
}

// writeProcessor processes events from the write buffer
func (es *EventStore) writeProcessor() {
	defer es.wg.Done()

	ticker := time.NewTicker(es.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-es.writeBuffer:
			if !ok {
				// Channel closed, flush remaining events and exit
				if len(es.batchBuffer) > 0 {
					es.flushBatch()
				}
				return
			}

			es.batchBuffer = append(es.batchBuffer, event)

			// Flush if batch is full
			if len(es.batchBuffer) >= es.config.BatchSize {
				es.flushBatch()
			}

		case <-ticker.C:
			// Flush on timer
			if len(es.batchBuffer) > 0 {
				es.flushBatch()
			}

		case <-es.ctx.Done():
			return
		}
	}
}

// flushBatch writes the current batch to the database
func (es *EventStore) flushBatch() error {
	if len(es.batchBuffer) == 0 {
		return nil
	}

	startTime := time.Now()

	tx, err := es.db.BeginTx(es.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.PrepareContext(es.ctx, `
		INSERT INTO events (id, type, action, version, timestamp, source, repository, 
			session_id, user_id, client_id, tags, correlation_id, causation_id, parent_id, 
			payload, metadata, sequence_number, ttl, expires_at, processed_at, delivered_at, acknowledged_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, event := range es.batchBuffer {
		if err := es.insertEvent(stmt, event); err != nil {
			log.Printf("Failed to insert event %s: %v", event.ID, err)
			continue
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update metrics
	batchSize := len(es.batchBuffer)
	writeTime := time.Since(startTime)

	es.updateMetrics(func(m *PersistenceMetrics) {
		m.EventsStored += int64(batchSize)
		m.BatchesProcessed++

		if m.AverageWriteTime == 0 {
			m.AverageWriteTime = writeTime
		} else {
			m.AverageWriteTime = time.Duration(
				int64(m.AverageWriteTime)*9/10 + int64(writeTime)/10,
			)
		}
	})

	log.Printf("Flushed batch of %d events to database (took %v)", batchSize, writeTime)

	// Clear batch buffer
	es.batchBuffer = es.batchBuffer[:0]

	return nil
}

// insertEvent inserts a single event using a prepared statement
func (es *EventStore) insertEvent(stmt *sql.Stmt, event *Event) error {
	// Convert complex fields to JSON
	tagsJSON, _ := json.Marshal(event.Tags)
	payloadJSON, _ := json.Marshal(event.Payload)
	metadataJSON, _ := json.Marshal(event.Metadata)

	var ttl *int64
	if event.TTL > 0 {
		ttlValue := int64(event.TTL)
		ttl = &ttlValue
	}

	_, err := stmt.ExecContext(es.ctx,
		event.ID, event.Type, event.Action, event.Version, event.Timestamp, event.Source,
		event.Repository, event.SessionID, event.UserID, event.ClientID,
		string(tagsJSON), event.CorrelationID, event.CausationID, event.ParentID,
		string(payloadJSON), string(metadataJSON), event.SequenceNumber,
		ttl, event.ExpiresAt, event.ProcessedAt, event.DeliveredAt, event.AcknowledgedAt)

	return err
}

// buildQuery builds a SQL query from an EventQuery
func (es *EventStore) buildQuery(query *EventQuery) (string, []interface{}) {
	sqlQuery := `SELECT id, type, action, version, timestamp, source, repository, session_id, 
		user_id, client_id, tags, correlation_id, causation_id, parent_id, payload, metadata, 
		sequence_number, ttl, expires_at, processed_at, delivered_at, acknowledged_at 
		FROM events`

	whereClause, args := es.buildWhereClause(query)
	if whereClause != "" {
		sqlQuery += " WHERE " + whereClause
	}

	// Add ordering
	orderBy := "timestamp"
	if query.OrderBy != "" {
		orderBy = query.OrderBy
	}

	orderDirection := "ASC"
	if query.OrderDirection != "" {
		orderDirection = query.OrderDirection
	}

	sqlQuery += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDirection)

	// Add limit and offset
	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT %d", query.Limit)
	}
	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	return sqlQuery, args
}

// buildWhereClause builds the WHERE clause for a query
func (es *EventStore) buildWhereClause(query *EventQuery) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if len(query.Types) > 0 {
		placeholders := make([]string, len(query.Types))
		for i, eventType := range query.Types {
			placeholders[i] = "?"
			args = append(args, eventType)
		}
		conditions = append(conditions, fmt.Sprintf("type IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(query.Actions) > 0 {
		placeholders := make([]string, len(query.Actions))
		for i, action := range query.Actions {
			placeholders[i] = "?"
			args = append(args, action)
		}
		conditions = append(conditions, fmt.Sprintf("action IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(query.Sources) > 0 {
		placeholders := make([]string, len(query.Sources))
		for i, source := range query.Sources {
			placeholders[i] = "?"
			args = append(args, source)
		}
		conditions = append(conditions, fmt.Sprintf("source IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(query.Repositories) > 0 {
		placeholders := make([]string, len(query.Repositories))
		for i, repo := range query.Repositories {
			placeholders[i] = "?"
			args = append(args, repo)
		}
		conditions = append(conditions, fmt.Sprintf("repository IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(query.SessionIDs) > 0 {
		placeholders := make([]string, len(query.SessionIDs))
		for i, sessionID := range query.SessionIDs {
			placeholders[i] = "?"
			args = append(args, sessionID)
		}
		conditions = append(conditions, fmt.Sprintf("session_id IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(query.UserIDs) > 0 {
		placeholders := make([]string, len(query.UserIDs))
		for i, userID := range query.UserIDs {
			placeholders[i] = "?"
			args = append(args, userID)
		}
		conditions = append(conditions, fmt.Sprintf("user_id IN (%s)", strings.Join(placeholders, ",")))
	}

	if query.After != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *query.After)
	}

	if query.Before != nil {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, *query.Before)
	}

	return strings.Join(conditions, " AND "), args
}

// scanEvent scans a database row into an Event
func (es *EventStore) scanEvent(rows *sql.Rows) (*Event, error) {
	var event Event
	var tagsJSON, payloadJSON, metadataJSON string
	var ttl *int64

	err := rows.Scan(
		&event.ID, &event.Type, &event.Action, &event.Version, &event.Timestamp, &event.Source,
		&event.Repository, &event.SessionID, &event.UserID, &event.ClientID,
		&tagsJSON, &event.CorrelationID, &event.CausationID, &event.ParentID,
		&payloadJSON, &metadataJSON, &event.SequenceNumber,
		&ttl, &event.ExpiresAt, &event.ProcessedAt, &event.DeliveredAt, &event.AcknowledgedAt)
	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	json.Unmarshal([]byte(tagsJSON), &event.Tags)
	json.Unmarshal([]byte(payloadJSON), &event.Payload)
	json.Unmarshal([]byte(metadataJSON), &event.Metadata)

	if ttl != nil {
		event.TTL = time.Duration(*ttl)
	}

	return &event, nil
}

// scanEventRow scans a single database row into an Event
func (es *EventStore) scanEventRow(row *sql.Row) (*Event, error) {
	var event Event
	var tagsJSON, payloadJSON, metadataJSON string
	var ttl *int64

	err := row.Scan(
		&event.ID, &event.Type, &event.Action, &event.Version, &event.Timestamp, &event.Source,
		&event.Repository, &event.SessionID, &event.UserID, &event.ClientID,
		&tagsJSON, &event.CorrelationID, &event.CausationID, &event.ParentID,
		&payloadJSON, &metadataJSON, &event.SequenceNumber,
		&ttl, &event.ExpiresAt, &event.ProcessedAt, &event.DeliveredAt, &event.AcknowledgedAt)
	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	json.Unmarshal([]byte(tagsJSON), &event.Tags)
	json.Unmarshal([]byte(payloadJSON), &event.Payload)
	json.Unmarshal([]byte(metadataJSON), &event.Metadata)

	if ttl != nil {
		event.TTL = time.Duration(*ttl)
	}

	return &event, nil
}

// cleanupRoutine performs periodic cleanup of expired events
func (es *EventStore) cleanupRoutine() {
	defer es.wg.Done()

	ticker := time.NewTicker(es.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			es.performCleanup()
		case <-es.ctx.Done():
			return
		}
	}
}

// performCleanup removes expired events and maintains disk usage limits
func (es *EventStore) performCleanup() {
	startTime := time.Now()

	// Delete expired events
	cutoff := time.Now().Add(-es.config.RetentionPeriod)
	result, err := es.db.ExecContext(es.ctx, "DELETE FROM events WHERE created_at < ?", cutoff)
	if err != nil {
		log.Printf("Error during cleanup: %v", err)
		return
	}

	rowsDeleted, _ := result.RowsAffected()

	// Vacuum database to reclaim space
	if rowsDeleted > 0 {
		_, err := es.db.ExecContext(es.ctx, "VACUUM")
		if err != nil {
			log.Printf("Error during vacuum: %v", err)
		}
	}

	es.updateMetrics(func(m *PersistenceMetrics) {
		m.EventsDeleted += rowsDeleted
		m.LastCleanup = time.Now()
	})

	if rowsDeleted > 0 {
		log.Printf("Cleanup completed: deleted %d expired events (took %v)", rowsDeleted, time.Since(startTime))
	}
}

// backupRoutine performs periodic database backups
func (es *EventStore) backupRoutine() {
	defer es.wg.Done()

	ticker := time.NewTicker(es.config.BackupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			es.performBackup()
		case <-es.ctx.Done():
			return
		}
	}
}

// performBackup creates a backup of the database
func (es *EventStore) performBackup() {
	startTime := time.Now()
	backupPath := fmt.Sprintf("%s.backup.%d", es.config.DatabasePath, time.Now().Unix())

	// Simple file copy backup (in production, consider using SQLite backup API)
	_, err := es.db.ExecContext(es.ctx, fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		log.Printf("Error creating backup: %v", err)
		return
	}

	es.updateMetrics(func(m *PersistenceMetrics) {
		m.LastBackup = time.Now()
	})

	log.Printf("Database backup created: %s (took %v)", backupPath, time.Since(startTime))
}

// replicationProcessor handles event replication to secondary stores
func (es *EventStore) replicationProcessor() {
	defer es.wg.Done()

	for {
		select {
		case event, ok := <-es.replicationChan:
			if !ok {
				return
			}

			// Here you would implement replication logic
			// For example, send to remote database, message queue, etc.
			log.Printf("Replicating event %s", event.ID)

		case <-es.ctx.Done():
			return
		}
	}
}

// performReplay performs event replay
func (es *EventStore) performReplay(replay *EventReplay, callback func(*Event) error) {
	replay.Status = ReplayStatusRunning

	events, err := es.Retrieve(replay.Query)
	if err != nil {
		replay.Status = ReplayStatusFailed
		replay.ErrorMessage = err.Error()
		return
	}

	replay.EventCount = int64(len(events))

	for i, event := range events {
		replay.CurrentEvent = int64(i + 1)
		replay.Progress = float64(replay.CurrentEvent) / float64(replay.EventCount) * 100

		if err := callback(event); err != nil {
			replay.Status = ReplayStatusFailed
			replay.ErrorMessage = err.Error()
			return
		}

		// Apply replay speed (1.0 = real-time, 2.0 = 2x speed, 0.5 = half speed)
		if replay.ReplaySpeed > 0 && replay.ReplaySpeed != 1.0 {
			time.Sleep(time.Duration(float64(time.Millisecond) / replay.ReplaySpeed))
		}
	}

	replay.Status = ReplayStatusCompleted
	replay.EndTime = &[]time.Time{time.Now()}[0]
}

// updateMetrics updates persistence metrics safely
func (es *EventStore) updateMetrics(updateFunc func(*PersistenceMetrics)) {
	es.metrics.mu.Lock()
	defer es.metrics.mu.Unlock()
	updateFunc(es.metrics)
}

// generateReplayID generates a unique replay ID
func generateReplayID() string {
	return fmt.Sprintf("replay_%d", time.Now().UnixNano())
}
