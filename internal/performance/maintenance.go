// Package performance provides database maintenance and optimization utilities.
package performance

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"lerian-mcp-memory/internal/config"
)

// Operation status constants
const (
	StatusFailed    = "failed"
	StatusCompleted = "completed"
)

// MaintenanceManager handles database maintenance operations
type MaintenanceManager struct {
	db     *sql.DB
	config *config.DatabaseConfig
	logger Logger
}

// Logger interface for maintenance operations
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// MaintenanceOperation represents a database maintenance operation
type MaintenanceOperation struct {
	Type         string                 `json:"type"`
	Target       string                 `json:"target"`
	Status       string                 `json:"status"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Duration     time.Duration          `json:"duration"`
	RowsAffected int64                  `json:"rows_affected"`
	Details      map[string]interface{} `json:"details"`
	Error        string                 `json:"error,omitempty"`
}

// IndexRecommendation represents a recommended index
type IndexRecommendation struct {
	TableName     string   `json:"table_name"`
	SchemaName    string   `json:"schema_name"`
	Columns       []string `json:"columns"`
	IndexType     string   `json:"index_type"`
	Reason        string   `json:"reason"`
	Priority      string   `json:"priority"`
	EstimatedSize int64    `json:"estimated_size_bytes"`
	Impact        string   `json:"impact"`
	CreateSQL     string   `json:"create_sql"`
}

// NewMaintenanceManager creates a new database maintenance manager
func NewMaintenanceManager(db *sql.DB, cfg *config.DatabaseConfig, logger Logger) *MaintenanceManager {
	return &MaintenanceManager{
		db:     db,
		config: cfg,
		logger: logger,
	}
}

// PerformMaintenance executes comprehensive database maintenance
func (mm *MaintenanceManager) PerformMaintenance(ctx context.Context) ([]*MaintenanceOperation, error) {
	mm.logger.Info("Starting database maintenance operations")

	var operations []*MaintenanceOperation

	// 1. Analyze tables for statistics
	analyzeOps, err := mm.analyzeAllTables(ctx)
	if err != nil {
		mm.logger.Error("Failed to analyze tables: %v", err)
	} else {
		operations = append(operations, analyzeOps...)
	}

	// 2. Vacuum tables to reclaim space
	vacuumOps, err := mm.vacuumTables(ctx)
	if err != nil {
		mm.logger.Error("Failed to vacuum tables: %v", err)
	} else {
		operations = append(operations, vacuumOps...)
	}

	// 3. Reindex heavily fragmented indexes
	reindexOps, err := mm.reindexFragmentedIndexes(ctx)
	if err != nil {
		mm.logger.Error("Failed to reindex: %v", err)
	} else {
		operations = append(operations, reindexOps...)
	}

	// 4. Update table statistics
	statsOps := mm.updateTableStatistics(ctx)
	operations = append(operations, statsOps...)

	mm.logger.Info("Database maintenance completed: %d operations", len(operations))
	return operations, nil
}

// analyzeAllTables runs ANALYZE on all user tables
func (mm *MaintenanceManager) analyzeAllTables(ctx context.Context) ([]*MaintenanceOperation, error) {
	// Get all user tables
	query := `
		SELECT schemaname, tablename 
		FROM pg_tables 
		WHERE schemaname NOT IN ('information_schema', 'pg_catalog')
		ORDER BY schemaname, tablename`

	rows, err := mm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get table list: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var operations []*MaintenanceOperation

	for rows.Next() {
		var schema, table string
		if err := rows.Scan(&schema, &table); err != nil {
			continue
		}

		op := &MaintenanceOperation{
			Type:      "ANALYZE",
			Target:    fmt.Sprintf("%s.%s", schema, table),
			Status:    "running",
			StartTime: time.Now(),
		}

		// Execute ANALYZE
		analyzeSQL := fmt.Sprintf("ANALYZE %s.%s", schema, table)
		result, err := mm.db.ExecContext(ctx, analyzeSQL)

		op.EndTime = timePtr(time.Now())
		op.Duration = op.EndTime.Sub(op.StartTime)

		if err != nil {
			op.Status = StatusFailed
			op.Error = err.Error()
			mm.logger.Warn("Failed to analyze table %s.%s: %v", schema, table, err)
		} else {
			op.Status = StatusCompleted
			if result != nil {
				if affected, err := result.RowsAffected(); err == nil {
					op.RowsAffected = affected
				}
			}
			mm.logger.Info("Analyzed table %s.%s in %v", schema, table, op.Duration)
		}

		operations = append(operations, op)
	}

	return operations, rows.Err()
}

// vacuumTables performs VACUUM on tables that need it
func (mm *MaintenanceManager) vacuumTables(ctx context.Context) ([]*MaintenanceOperation, error) {
	// Find tables with high dead tuple ratio
	query := `
		SELECT 
			schemaname,
			tablename,
			n_dead_tup,
			n_live_tup,
			CASE 
				WHEN n_live_tup > 0 
				THEN round(100.0 * n_dead_tup / (n_live_tup + n_dead_tup), 2)
				ELSE 0 
			END as dead_ratio
		FROM pg_stat_user_tables
		WHERE n_dead_tup > 1000  -- Only tables with significant dead tuples
		  AND CASE 
				WHEN n_live_tup > 0 
				THEN 100.0 * n_dead_tup / (n_live_tup + n_dead_tup)
				ELSE 0 
			  END > 20  -- More than 20% dead tuples
		ORDER BY dead_ratio DESC`

	rows, err := mm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to identify tables for vacuum: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var operations []*MaintenanceOperation

	for rows.Next() {
		var schema, table string
		var deadTuples, liveTuples int64
		var deadRatio float64

		if err := rows.Scan(&schema, &table, &deadTuples, &liveTuples, &deadRatio); err != nil {
			continue
		}

		op := &MaintenanceOperation{
			Type:      "VACUUM",
			Target:    fmt.Sprintf("%s.%s", schema, table),
			Status:    "running",
			StartTime: time.Now(),
			Details: map[string]interface{}{
				"dead_tuples": deadTuples,
				"live_tuples": liveTuples,
				"dead_ratio":  deadRatio,
			},
		}

		// Execute VACUUM
		vacuumSQL := fmt.Sprintf("VACUUM %s.%s", schema, table)
		result, err := mm.db.ExecContext(ctx, vacuumSQL)

		op.EndTime = timePtr(time.Now())
		op.Duration = op.EndTime.Sub(op.StartTime)

		if err != nil {
			op.Status = StatusFailed
			op.Error = err.Error()
			mm.logger.Warn("Failed to vacuum table %s.%s: %v", schema, table, err)
		} else {
			op.Status = StatusCompleted
			if result != nil {
				if affected, err := result.RowsAffected(); err == nil {
					op.RowsAffected = affected
				}
			}
			mm.logger.Info("Vacuumed table %s.%s (%.1f%% dead) in %v",
				schema, table, deadRatio, op.Duration)
		}

		operations = append(operations, op)
	}

	return operations, rows.Err()
}

// reindexFragmentedIndexes rebuilds indexes with high fragmentation
func (mm *MaintenanceManager) reindexFragmentedIndexes(ctx context.Context) ([]*MaintenanceOperation, error) {
	// Find fragmented indexes (simplified - in production, use pg_stat_user_indexes)
	query := `
		SELECT 
			n.nspname as schema_name,
			t.relname as table_name,
			i.relname as index_name,
			pg_relation_size(i.oid) as index_size
		FROM pg_class i
		JOIN pg_index ix ON i.oid = ix.indexrelid
		JOIN pg_class t ON ix.indrelid = t.oid
		JOIN pg_namespace n ON t.relnamespace = n.oid
		WHERE n.nspname NOT IN ('information_schema', 'pg_catalog')
		  AND pg_relation_size(i.oid) > 1048576  -- Only indexes > 1MB
		ORDER BY index_size DESC
		LIMIT 10`

	rows, err := mm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to identify fragmented indexes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var operations []*MaintenanceOperation

	for rows.Next() {
		var schema, table, index string
		var indexSize int64

		if err := rows.Scan(&schema, &table, &index, &indexSize); err != nil {
			continue
		}

		op := &MaintenanceOperation{
			Type:      "REINDEX",
			Target:    fmt.Sprintf("%s.%s.%s", schema, table, index),
			Status:    "running",
			StartTime: time.Now(),
			Details: map[string]interface{}{
				"index_size_bytes": indexSize,
				"table_name":       table,
			},
		}

		// Execute REINDEX
		reindexSQL := fmt.Sprintf("REINDEX INDEX %s.%s", schema, index)
		result, err := mm.db.ExecContext(ctx, reindexSQL)

		op.EndTime = timePtr(time.Now())
		op.Duration = op.EndTime.Sub(op.StartTime)

		if err != nil {
			op.Status = StatusFailed
			op.Error = err.Error()
			mm.logger.Warn("Failed to reindex %s.%s: %v", schema, index, err)
		} else {
			op.Status = StatusCompleted
			if result != nil {
				if affected, err := result.RowsAffected(); err == nil {
					op.RowsAffected = affected
				}
			}
			mm.logger.Info("Reindexed %s.%s (%.1f MB) in %v",
				schema, index, float64(indexSize)/1048576, op.Duration)
		}

		operations = append(operations, op)
	}

	return operations, rows.Err()
}

// updateTableStatistics updates PostgreSQL table statistics
func (mm *MaintenanceManager) updateTableStatistics(ctx context.Context) []*MaintenanceOperation {
	var operations []*MaintenanceOperation

	op := &MaintenanceOperation{
		Type:      "UPDATE_STATS",
		Target:    "pg_stat_statements",
		Status:    "running",
		StartTime: time.Now(),
	}

	// Reset pg_stat_statements to get fresh statistics
	_, err := mm.db.ExecContext(ctx, "SELECT pg_stat_statements_reset()")

	op.EndTime = timePtr(time.Now())
	op.Duration = op.EndTime.Sub(op.StartTime)

	if err != nil {
		op.Status = StatusFailed
		op.Error = err.Error()
		mm.logger.Warn("Failed to reset pg_stat_statements: %v", err)
	} else {
		op.Status = StatusCompleted
		mm.logger.Info("Reset pg_stat_statements in %v", op.Duration)
	}

	operations = append(operations, op)
	return operations
}

// GenerateIndexRecommendations analyzes query patterns and suggests new indexes
func (mm *MaintenanceManager) GenerateIndexRecommendations(ctx context.Context) ([]*IndexRecommendation, error) {
	var recommendations []*IndexRecommendation

	// Find tables with high sequential scan ratios
	tablesQuery := `
		SELECT 
			schemaname,
			tablename,
			seq_scan,
			seq_tup_read,
			idx_scan,
			n_live_tup
		FROM pg_stat_user_tables
		WHERE seq_scan > idx_scan * 2  -- High sequential scan ratio
		  AND seq_tup_read > 10000     -- Significant read activity
		  AND n_live_tup > 1000        -- Non-trivial table size
		ORDER BY seq_tup_read DESC
		LIMIT 20`

	rows, err := mm.db.QueryContext(ctx, tablesQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze tables for index recommendations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var schema, table string
		var seqScan, seqTupRead, idxScan, liveTuples int64

		if err := rows.Scan(&schema, &table, &seqScan, &seqTupRead, &idxScan, &liveTuples); err != nil {
			continue
		}

		// Analyze table structure for potential indexes
		tableRecs, err := mm.analyzeTableForIndexes(ctx, schema, table, seqScan, seqTupRead, liveTuples)
		if err != nil {
			mm.logger.Warn("Failed to analyze table %s.%s for indexes: %v", schema, table, err)
			continue
		}

		recommendations = append(recommendations, tableRecs...)
	}

	return recommendations, nil
}

// analyzeTableForIndexes analyzes a specific table for index opportunities
func (mm *MaintenanceManager) analyzeTableForIndexes(ctx context.Context, schema, table string, seqScan, seqTupRead, liveTuples int64) ([]*IndexRecommendation, error) {
	var recommendations []*IndexRecommendation

	// Get table columns for analysis
	columnsQuery := `
		SELECT 
			column_name,
			data_type,
			is_nullable
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position`

	rows, err := mm.db.QueryContext(ctx, columnsQuery, schema, table)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var columns []string
	var stringColumns []string
	var numericColumns []string

	for rows.Next() {
		var columnName, dataType, isNullable string
		if err := rows.Scan(&columnName, &dataType, &isNullable); err != nil {
			continue
		}

		columns = append(columns, columnName)

		// Categorize columns by type for different index strategies
		if strings.Contains(strings.ToLower(dataType), "char") ||
			strings.Contains(strings.ToLower(dataType), "text") {
			stringColumns = append(stringColumns, columnName)
		} else if strings.Contains(strings.ToLower(dataType), "int") ||
			strings.Contains(strings.ToLower(dataType), "numeric") ||
			strings.Contains(strings.ToLower(dataType), "decimal") {
			numericColumns = append(numericColumns, columnName)
		}
	}

	// Recommend indexes based on common patterns
	ratio := float64(seqScan) / float64(seqScan+1) // Add 1 to avoid division by zero
	estimatedSize := liveTuples * 50               // Rough estimate

	// High priority recommendations for frequently scanned tables
	if ratio > 0.8 && seqTupRead > 100000 {
		// Suggest composite index on first few columns
		if len(columns) >= 2 {
			rec := &IndexRecommendation{
				TableName:     table,
				SchemaName:    schema,
				Columns:       columns[:2],
				IndexType:     "btree",
				Reason:        fmt.Sprintf("High sequential scan ratio (%.1f%%) with %d tuples read", ratio*100, seqTupRead),
				Priority:      "high",
				EstimatedSize: estimatedSize,
				Impact:        "high",
				CreateSQL:     fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s.%s (%s)", table, strings.Join(columns[:2], "_"), schema, table, strings.Join(columns[:2], ", ")),
			}
			recommendations = append(recommendations, rec)
		}

		// Suggest partial indexes for string columns
		for _, col := range stringColumns[:minInt(len(stringColumns), 2)] {
			rec := &IndexRecommendation{
				TableName:     table,
				SchemaName:    schema,
				Columns:       []string{col},
				IndexType:     "btree",
				Reason:        "String column with high scan activity, consider partial index",
				Priority:      "medium",
				EstimatedSize: estimatedSize / 3,
				Impact:        "medium",
				CreateSQL:     fmt.Sprintf("CREATE INDEX idx_%s_%s_partial ON %s.%s (%s) WHERE %s IS NOT NULL", table, col, schema, table, col, col),
			}
			recommendations = append(recommendations, rec)
		}
	}

	// Medium priority for moderately scanned tables
	if ratio > 0.5 && len(numericColumns) > 0 {
		col := numericColumns[0]
		rec := &IndexRecommendation{
			TableName:     table,
			SchemaName:    schema,
			Columns:       []string{col},
			IndexType:     "btree",
			Reason:        "Numeric column with moderate scan activity",
			Priority:      "medium",
			EstimatedSize: estimatedSize / 2,
			Impact:        "medium",
			CreateSQL:     fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s.%s (%s)", table, col, schema, table, col),
		}
		recommendations = append(recommendations, rec)
	}

	return recommendations, nil
}

// CreateRecommendedIndexes creates indexes based on recommendations
func (mm *MaintenanceManager) CreateRecommendedIndexes(ctx context.Context, recommendations []*IndexRecommendation, maxIndexes int) ([]*MaintenanceOperation, error) {
	operations := make([]*MaintenanceOperation, 0, maxIndexes)

	// Sort by priority and impact
	highPriorityRecs := []*IndexRecommendation{}
	for _, rec := range recommendations {
		if rec.Priority == "high" {
			highPriorityRecs = append(highPriorityRecs, rec)
		}
	}

	// Limit to prevent excessive index creation
	toCreate := highPriorityRecs
	if len(toCreate) > maxIndexes {
		toCreate = toCreate[:maxIndexes]
	}

	for _, rec := range toCreate {
		op := &MaintenanceOperation{
			Type:      "CREATE_INDEX",
			Target:    fmt.Sprintf("%s.%s", rec.SchemaName, rec.TableName),
			Status:    "running",
			StartTime: time.Now(),
			Details: map[string]interface{}{
				"index_columns": rec.Columns,
				"index_type":    rec.IndexType,
				"reason":        rec.Reason,
				"sql":           rec.CreateSQL,
			},
		}

		// Execute CREATE INDEX
		result, err := mm.db.ExecContext(ctx, rec.CreateSQL)

		op.EndTime = timePtr(time.Now())
		op.Duration = op.EndTime.Sub(op.StartTime)

		if err != nil {
			op.Status = StatusFailed
			op.Error = err.Error()
			mm.logger.Warn("Failed to create index on %s.%s: %v", rec.SchemaName, rec.TableName, err)
		} else {
			op.Status = StatusCompleted
			if result != nil {
				if affected, err := result.RowsAffected(); err == nil {
					op.RowsAffected = affected
				}
			}
			mm.logger.Info("Created index on %s.%s (%v) in %v",
				rec.SchemaName, rec.TableName, rec.Columns, op.Duration)
		}

		operations = append(operations, op)
	}

	return operations, nil
}

// CleanupUnusedIndexes removes indexes that are never used
func (mm *MaintenanceManager) CleanupUnusedIndexes(ctx context.Context) ([]*MaintenanceOperation, error) {
	// Find indexes with zero scans
	query := `
		SELECT 
			n.nspname as schema_name,
			t.relname as table_name,
			i.relname as index_name,
			pg_relation_size(i.oid) as index_size
		FROM pg_class i
		JOIN pg_index ix ON i.oid = ix.indexrelid
		JOIN pg_class t ON ix.indrelid = t.oid
		JOIN pg_namespace n ON t.relnamespace = n.oid
		LEFT JOIN pg_stat_user_indexes s ON i.oid = s.indexrelid
		WHERE n.nspname NOT IN ('information_schema', 'pg_catalog')
		  AND NOT ix.indisprimary  -- Don't drop primary keys
		  AND NOT ix.indisunique   -- Don't drop unique constraints
		  AND (s.idx_scan = 0 OR s.idx_scan IS NULL)  -- Never used
		  AND pg_relation_size(i.oid) > 0  -- Has size
		ORDER BY index_size DESC`

	rows, err := mm.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to identify unused indexes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var operations []*MaintenanceOperation

	for rows.Next() {
		var schema, table, index string
		var indexSize int64

		if err := rows.Scan(&schema, &table, &index, &indexSize); err != nil {
			continue
		}

		op := &MaintenanceOperation{
			Type:      "DROP_INDEX",
			Target:    fmt.Sprintf("%s.%s", schema, index),
			Status:    "running",
			StartTime: time.Now(),
			Details: map[string]interface{}{
				"table_name":       table,
				"index_size_bytes": indexSize,
				"reason":           "unused index",
			},
		}

		// Execute DROP INDEX
		dropSQL := fmt.Sprintf("DROP INDEX %s.%s", schema, index)
		result, err := mm.db.ExecContext(ctx, dropSQL)

		op.EndTime = timePtr(time.Now())
		op.Duration = op.EndTime.Sub(op.StartTime)

		if err != nil {
			op.Status = StatusFailed
			op.Error = err.Error()
			mm.logger.Warn("Failed to drop unused index %s.%s: %v", schema, index, err)
		} else {
			op.Status = StatusCompleted
			if result != nil {
				if affected, err := result.RowsAffected(); err == nil {
					op.RowsAffected = affected
				}
			}
			mm.logger.Info("Dropped unused index %s.%s (%.1f MB) in %v",
				schema, index, float64(indexSize)/1048576, op.Duration)
		}

		operations = append(operations, op)
	}

	return operations, nil
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	return &t
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DefaultLogger provides a simple logger implementation
type DefaultLogger struct{}

func (dl *DefaultLogger) Info(msg string, args ...interface{}) {
	fmt.Printf("[INFO] "+msg+"\n", args...)
}

func (dl *DefaultLogger) Warn(msg string, args ...interface{}) {
	fmt.Printf("[WARN] "+msg+"\n", args...)
}

func (dl *DefaultLogger) Error(msg string, args ...interface{}) {
	fmt.Printf("[ERROR] "+msg+"\n", args...)
}
