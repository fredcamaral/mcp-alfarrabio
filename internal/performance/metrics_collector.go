// Package performance provides comprehensive database metrics collection and monitoring.
package performance

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"lerian-mcp-memory/internal/config"
)

// MetricsCollector collects and aggregates database performance metrics
type MetricsCollector struct {
	db     *sql.DB
	config *config.DatabaseConfig
	stats  *DatabaseMetrics
	mu     sync.RWMutex
}

// DatabaseMetrics represents comprehensive database performance metrics
type DatabaseMetrics struct {
	// Connection metrics
	ActiveConnections int64 `json:"active_connections"`
	IdleConnections   int64 `json:"idle_connections"`
	MaxConnections    int64 `json:"max_connections"`
	ConnectionWaits   int64 `json:"connection_waits"`

	// Query metrics
	TotalQueries     int64         `json:"total_queries"`
	SlowQueries      int64         `json:"slow_queries"`
	FailedQueries    int64         `json:"failed_queries"`
	AvgQueryDuration time.Duration `json:"avg_query_duration"`
	P95QueryDuration time.Duration `json:"p95_query_duration"`
	P99QueryDuration time.Duration `json:"p99_query_duration"`

	// Cache and index metrics
	CacheHitRatio   float64 `json:"cache_hit_ratio"`
	IndexHitRatio   float64 `json:"index_hit_ratio"`
	BufferHitRatio  float64 `json:"buffer_hit_ratio"`
	TotalIndexScans int64   `json:"total_index_scans"`
	TotalSeqScans   int64   `json:"total_seq_scans"`

	// Transaction metrics
	TotalTransactions int64         `json:"total_transactions"`
	CommittedTxns     int64         `json:"committed_transactions"`
	RolledBackTxns    int64         `json:"rolled_back_transactions"`
	AvgTxnDuration    time.Duration `json:"avg_transaction_duration"`
	DeadlockCount     int64         `json:"deadlock_count"`

	// Storage metrics
	DatabaseSize      int64 `json:"database_size_bytes"`
	TableSize         int64 `json:"table_size_bytes"`
	IndexSize         int64 `json:"index_size_bytes"`
	FreeSpace         int64 `json:"free_space_bytes"`
	VacuumOperations  int64 `json:"vacuum_operations"`
	AnalyzeOperations int64 `json:"analyze_operations"`

	// Lock metrics
	LocksHeld       int64         `json:"locks_held"`
	LockWaits       int64         `json:"lock_waits"`
	LockTimeouts    int64         `json:"lock_timeouts"`
	AvgLockWaitTime time.Duration `json:"avg_lock_wait_time"`

	// Error metrics
	ConnectionErrors     int64 `json:"connection_errors"`
	QueryErrors          int64 `json:"query_errors"`
	ConstraintViolations int64 `json:"constraint_violations"`

	// Timestamp
	CollectedAt time.Time `json:"collected_at"`

	// Query type breakdown
	SelectQueries int64 `json:"select_queries"`
	InsertQueries int64 `json:"insert_queries"`
	UpdateQueries int64 `json:"update_queries"`
	DeleteQueries int64 `json:"delete_queries"`
}

// QueryTypeMetrics represents metrics for specific query types
type QueryTypeMetrics struct {
	QueryType      string        `json:"query_type"`
	Count          int64         `json:"count"`
	TotalDuration  time.Duration `json:"total_duration"`
	AvgDuration    time.Duration `json:"avg_duration"`
	MinDuration    time.Duration `json:"min_duration"`
	MaxDuration    time.Duration `json:"max_duration"`
	ErrorCount     int64         `json:"error_count"`
	SlowQueryCount int64         `json:"slow_query_count"`
	CacheHitRatio  float64       `json:"cache_hit_ratio"`
	IndexScanRatio float64       `json:"index_scan_ratio"`
}

// TableMetrics represents metrics for individual database tables
type TableMetrics struct {
	SchemaName      string     `json:"schema_name"`
	TableName       string     `json:"table_name"`
	RowCount        int64      `json:"row_count"`
	TableSize       int64      `json:"table_size_bytes"`
	IndexSize       int64      `json:"index_size_bytes"`
	SeqScans        int64      `json:"sequential_scans"`
	SeqTupRead      int64      `json:"sequential_tuples_read"`
	IndexScans      int64      `json:"index_scans"`
	IndexTupFetch   int64      `json:"index_tuples_fetched"`
	Inserts         int64      `json:"inserts"`
	Updates         int64      `json:"updates"`
	Deletes         int64      `json:"deletes"`
	HotUpdates      int64      `json:"hot_updates"`
	LiveTuples      int64      `json:"live_tuples"`
	DeadTuples      int64      `json:"dead_tuples"`
	LastVacuum      *time.Time `json:"last_vacuum,omitempty"`
	LastAutoVacuum  *time.Time `json:"last_autovacuum,omitempty"`
	LastAnalyze     *time.Time `json:"last_analyze,omitempty"`
	LastAutoAnalyze *time.Time `json:"last_autoanalyze,omitempty"`
}

// IndexMetrics represents metrics for database indexes
type IndexMetrics struct {
	SchemaName    string `json:"schema_name"`
	TableName     string `json:"table_name"`
	IndexName     string `json:"index_name"`
	IndexSize     int64  `json:"index_size_bytes"`
	IndexScans    int64  `json:"index_scans"`
	TuplesRead    int64  `json:"tuples_read"`
	TuplesFetched int64  `json:"tuples_fetched"`
	IsUnique      bool   `json:"is_unique"`
	IsPrimary     bool   `json:"is_primary"`
	IndexDef      string `json:"index_definition"`
}

// NewMetricsCollector creates a new database metrics collector
func NewMetricsCollector(db *sql.DB, cfg *config.DatabaseConfig) *MetricsCollector {
	return &MetricsCollector{
		db:     db,
		config: cfg,
		stats:  &DatabaseMetrics{},
	}
}

// CollectMetrics collects all database performance metrics
func (mc *MetricsCollector) CollectMetrics(ctx context.Context) (*DatabaseMetrics, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics := &DatabaseMetrics{
		CollectedAt: time.Now(),
	}

	// Collect connection metrics
	if err := mc.collectConnectionMetrics(ctx, metrics); err != nil {
		return nil, fmt.Errorf("failed to collect connection metrics: %w", err)
	}

	// Collect query statistics
	if err := mc.collectQueryStatistics(ctx, metrics); err != nil {
		return nil, fmt.Errorf("failed to collect query statistics: %w", err)
	}

	// Collect cache and buffer metrics
	if err := mc.collectCacheMetrics(ctx, metrics); err != nil {
		return nil, fmt.Errorf("failed to collect cache metrics: %w", err)
	}

	// Collect transaction metrics
	if err := mc.collectTransactionMetrics(ctx, metrics); err != nil {
		return nil, fmt.Errorf("failed to collect transaction metrics: %w", err)
	}

	// Collect storage metrics
	if err := mc.collectStorageMetrics(ctx, metrics); err != nil {
		return nil, fmt.Errorf("failed to collect storage metrics: %w", err)
	}

	// Collect lock metrics
	if err := mc.collectLockMetrics(ctx, metrics); err != nil {
		return nil, fmt.Errorf("failed to collect lock metrics: %w", err)
	}

	mc.stats = metrics
	return metrics, nil
}

// collectConnectionMetrics collects database connection pool metrics
func (mc *MetricsCollector) collectConnectionMetrics(ctx context.Context, metrics *DatabaseMetrics) error {
	// Get connection pool stats from Go database/sql
	stats := mc.db.Stats()
	metrics.ActiveConnections = int64(stats.InUse)
	metrics.IdleConnections = int64(stats.Idle)
	metrics.MaxConnections = int64(stats.MaxOpenConnections)
	metrics.ConnectionWaits = stats.WaitCount

	return nil
}

// collectQueryStatistics collects query performance statistics
func (mc *MetricsCollector) collectQueryStatistics(ctx context.Context, metrics *DatabaseMetrics) error {
	query := `
		SELECT 
			sum(calls) as total_queries,
			sum(total_time) as total_duration_ms,
			avg(mean_time) as avg_duration_ms,
			percentile_cont(0.95) WITHIN GROUP (ORDER BY mean_time) as p95_duration_ms,
			percentile_cont(0.99) WITHIN GROUP (ORDER BY mean_time) as p99_duration_ms
		FROM pg_stat_statements
		WHERE query NOT LIKE '%pg_stat_statements%'`

	row := mc.db.QueryRowContext(ctx, query)

	var totalQueries int64
	var totalDurationMs, avgDurationMs, p95DurationMs, p99DurationMs float64

	err := row.Scan(&totalQueries, &totalDurationMs, &avgDurationMs, &p95DurationMs, &p99DurationMs)
	if err != nil {
		// If pg_stat_statements is not available, use basic metrics
		return mc.collectBasicQueryMetrics(ctx, metrics)
	}

	metrics.TotalQueries = totalQueries
	metrics.AvgQueryDuration = time.Duration(avgDurationMs) * time.Millisecond
	metrics.P95QueryDuration = time.Duration(p95DurationMs) * time.Millisecond
	metrics.P99QueryDuration = time.Duration(p99DurationMs) * time.Millisecond

	// Count slow queries
	slowQuery := `
		SELECT count(*) 
		FROM pg_stat_statements 
		WHERE mean_time > $1 AND query NOT LIKE '%pg_stat_statements%'`

	slowThresholdMs := float64(mc.config.SlowQueryThreshold.Milliseconds())
	row = mc.db.QueryRowContext(ctx, slowQuery, slowThresholdMs)
	row.Scan(&metrics.SlowQueries)

	return nil
}

// collectBasicQueryMetrics collects basic query metrics when pg_stat_statements is unavailable
func (mc *MetricsCollector) collectBasicQueryMetrics(ctx context.Context, metrics *DatabaseMetrics) error {
	// Get basic database statistics
	query := `
		SELECT 
			sum(tup_returned + tup_fetched + tup_inserted + tup_updated + tup_deleted) as total_operations,
			sum(tup_returned + tup_fetched) as select_ops,
			sum(tup_inserted) as insert_ops,
			sum(tup_updated) as update_ops,
			sum(tup_deleted) as delete_ops
		FROM pg_stat_database 
		WHERE datname = current_database()`

	row := mc.db.QueryRowContext(ctx, query)
	var totalOps, selectOps, insertOps, updateOps, deleteOps int64

	err := row.Scan(&totalOps, &selectOps, &insertOps, &updateOps, &deleteOps)
	if err != nil {
		return err
	}

	metrics.TotalQueries = totalOps
	metrics.SelectQueries = selectOps
	metrics.InsertQueries = insertOps
	metrics.UpdateQueries = updateOps
	metrics.DeleteQueries = deleteOps

	return nil
}

// collectCacheMetrics collects cache hit ratio and buffer statistics
func (mc *MetricsCollector) collectCacheMetrics(ctx context.Context, metrics *DatabaseMetrics) error {
	// Buffer cache hit ratio
	query := `
		SELECT 
			round(100.0 * sum(blks_hit) / (sum(blks_hit) + sum(blks_read)), 2) as buffer_hit_ratio
		FROM pg_stat_database 
		WHERE datname = current_database()`

	row := mc.db.QueryRowContext(ctx, query)
	row.Scan(&metrics.BufferHitRatio)

	// Index usage statistics
	indexQuery := `
		SELECT 
			sum(idx_scan) as total_index_scans,
			sum(seq_scan) as total_seq_scans
		FROM pg_stat_user_tables`

	row = mc.db.QueryRowContext(ctx, indexQuery)
	row.Scan(&metrics.TotalIndexScans, &metrics.TotalSeqScans)

	// Calculate index hit ratio
	if metrics.TotalIndexScans+metrics.TotalSeqScans > 0 {
		metrics.IndexHitRatio = float64(metrics.TotalIndexScans) / float64(metrics.TotalIndexScans+metrics.TotalSeqScans) * 100
	}

	return nil
}

// collectTransactionMetrics collects transaction statistics
func (mc *MetricsCollector) collectTransactionMetrics(ctx context.Context, metrics *DatabaseMetrics) error {
	query := `
		SELECT 
			xact_commit as committed_txns,
			xact_rollback as rolled_back_txns,
			deadlocks as deadlock_count
		FROM pg_stat_database 
		WHERE datname = current_database()`

	row := mc.db.QueryRowContext(ctx, query)
	err := row.Scan(&metrics.CommittedTxns, &metrics.RolledBackTxns, &metrics.DeadlockCount)
	if err != nil {
		return err
	}

	metrics.TotalTransactions = metrics.CommittedTxns + metrics.RolledBackTxns
	return nil
}

// collectStorageMetrics collects database storage statistics
func (mc *MetricsCollector) collectStorageMetrics(ctx context.Context, metrics *DatabaseMetrics) error {
	// Database size
	sizeQuery := `SELECT pg_database_size(current_database())`
	row := mc.db.QueryRowContext(ctx, sizeQuery)
	row.Scan(&metrics.DatabaseSize)

	// Total table and index sizes
	tablesQuery := `
		SELECT 
			sum(pg_total_relation_size(c.oid)) as total_size,
			sum(pg_relation_size(c.oid)) as table_size,
			sum(pg_total_relation_size(c.oid) - pg_relation_size(c.oid)) as index_size
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'r' AND n.nspname NOT IN ('information_schema', 'pg_catalog')`

	row = mc.db.QueryRowContext(ctx, tablesQuery)
	var totalSize int64
	row.Scan(&totalSize, &metrics.TableSize, &metrics.IndexSize)

	// Vacuum and analyze statistics
	vacuumQuery := `
		SELECT 
			sum(CASE WHEN last_vacuum IS NOT NULL THEN 1 ELSE 0 END) as manual_vacuums,
			sum(CASE WHEN last_autovacuum IS NOT NULL THEN 1 ELSE 0 END) as auto_vacuums,
			sum(CASE WHEN last_analyze IS NOT NULL THEN 1 ELSE 0 END) as manual_analyzes,
			sum(CASE WHEN last_autoanalyze IS NOT NULL THEN 1 ELSE 0 END) as auto_analyzes
		FROM pg_stat_user_tables`

	row = mc.db.QueryRowContext(ctx, vacuumQuery)
	var manualVacuums, autoVacuums, manualAnalyzes, autoAnalyzes int64
	row.Scan(&manualVacuums, &autoVacuums, &manualAnalyzes, &autoAnalyzes)

	metrics.VacuumOperations = manualVacuums + autoVacuums
	metrics.AnalyzeOperations = manualAnalyzes + autoAnalyzes

	return nil
}

// collectLockMetrics collects database lock statistics
func (mc *MetricsCollector) collectLockMetrics(ctx context.Context, metrics *DatabaseMetrics) error {
	// Current locks
	lockQuery := `
		SELECT 
			count(*) as locks_held
		FROM pg_locks 
		WHERE granted = true`

	row := mc.db.QueryRowContext(ctx, lockQuery)
	row.Scan(&metrics.LocksHeld)

	// Lock waits from pg_stat_database
	waitQuery := `
		SELECT 
			blk_read_time + blk_write_time as io_wait_time
		FROM pg_stat_database 
		WHERE datname = current_database()`

	row = mc.db.QueryRowContext(ctx, waitQuery)
	var ioWaitTime float64
	row.Scan(&ioWaitTime)
	metrics.AvgLockWaitTime = time.Duration(ioWaitTime) * time.Millisecond

	return nil
}

// GetTableMetrics retrieves detailed metrics for all user tables
func (mc *MetricsCollector) GetTableMetrics(ctx context.Context) ([]TableMetrics, error) {
	query := `
		SELECT 
			schemaname,
			tablename,
			n_tup_ins as inserts,
			n_tup_upd as updates,
			n_tup_del as deletes,
			n_tup_hot_upd as hot_updates,
			n_live_tup as live_tuples,
			n_dead_tup as dead_tuples,
			seq_scan,
			seq_tup_read,
			idx_scan,
			idx_tup_fetch,
			last_vacuum,
			last_autovacuum,
			last_analyze,
			last_autoanalyze
		FROM pg_stat_user_tables
		ORDER BY schemaname, tablename`

	rows, err := mc.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableMetrics
	for rows.Next() {
		var table TableMetrics
		var lastVacuum, lastAutoVacuum, lastAnalyze, lastAutoAnalyze sql.NullTime

		err := rows.Scan(
			&table.SchemaName, &table.TableName,
			&table.Inserts, &table.Updates, &table.Deletes, &table.HotUpdates,
			&table.LiveTuples, &table.DeadTuples,
			&table.SeqScans, &table.SeqTupRead,
			&table.IndexScans, &table.IndexTupFetch,
			&lastVacuum, &lastAutoVacuum, &lastAnalyze, &lastAutoAnalyze,
		)
		if err != nil {
			continue
		}

		// Get table and index sizes
		sizeQuery := `
			SELECT 
				pg_relation_size($1) as table_size,
				pg_total_relation_size($1) - pg_relation_size($1) as index_size,
				pg_stat_get_live_tuples($2) as row_count
			FROM pg_class 
			WHERE relname = $3 AND relnamespace = (SELECT oid FROM pg_namespace WHERE nspname = $4)`

		var tableOid int64
		oidQuery := `SELECT c.oid FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.relname = $1 AND n.nspname = $2`
		mc.db.QueryRowContext(ctx, oidQuery, table.TableName, table.SchemaName).Scan(&tableOid)

		mc.db.QueryRowContext(ctx, sizeQuery, tableOid, tableOid, table.TableName, table.SchemaName).Scan(
			&table.TableSize, &table.IndexSize, &table.RowCount)

		// Set nullable timestamps
		if lastVacuum.Valid {
			table.LastVacuum = &lastVacuum.Time
		}
		if lastAutoVacuum.Valid {
			table.LastAutoVacuum = &lastAutoVacuum.Time
		}
		if lastAnalyze.Valid {
			table.LastAnalyze = &lastAnalyze.Time
		}
		if lastAutoAnalyze.Valid {
			table.LastAutoAnalyze = &lastAutoAnalyze.Time
		}

		tables = append(tables, table)
	}

	return tables, rows.Err()
}

// GetIndexMetrics retrieves detailed metrics for all indexes
func (mc *MetricsCollector) GetIndexMetrics(ctx context.Context) ([]IndexMetrics, error) {
	query := `
		SELECT 
			n.nspname as schema_name,
			t.relname as table_name,
			i.relname as index_name,
			pg_relation_size(i.oid) as index_size,
			s.idx_scan,
			s.idx_tup_read,
			s.idx_tup_fetch,
			ix.indisunique as is_unique,
			ix.indisprimary as is_primary,
			pg_get_indexdef(i.oid) as index_def
		FROM pg_class i
		JOIN pg_index ix ON i.oid = ix.indexrelid
		JOIN pg_class t ON ix.indrelid = t.oid
		JOIN pg_namespace n ON t.relnamespace = n.oid
		LEFT JOIN pg_stat_user_indexes s ON i.oid = s.indexrelid
		WHERE n.nspname NOT IN ('information_schema', 'pg_catalog')
		ORDER BY schema_name, table_name, index_name`

	rows, err := mc.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []IndexMetrics
	for rows.Next() {
		var index IndexMetrics
		err := rows.Scan(
			&index.SchemaName, &index.TableName, &index.IndexName,
			&index.IndexSize, &index.IndexScans, &index.TuplesRead, &index.TuplesFetched,
			&index.IsUnique, &index.IsPrimary, &index.IndexDef,
		)
		if err != nil {
			continue
		}
		indexes = append(indexes, index)
	}

	return indexes, rows.Err()
}

// GetCurrentMetrics returns the most recently collected metrics
func (mc *MetricsCollector) GetCurrentMetrics() *DatabaseMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy to avoid race conditions
	if mc.stats == nil {
		return &DatabaseMetrics{}
	}

	metrics := *mc.stats
	return &metrics
}

// StartPeriodicCollection starts automatic metrics collection at specified interval
func (mc *MetricsCollector) StartPeriodicCollection(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := mc.CollectMetrics(ctx); err != nil {
				// Log error but continue collection
				fmt.Printf("Failed to collect metrics: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
