// Package performance provides query optimization and database performance monitoring.
package performance

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"lerian-mcp-memory/internal/config"
)

// QueryOptimizer provides query analysis and optimization suggestions
type QueryOptimizer struct {
	db     *sql.DB
	config *config.DatabaseConfig
	cache  *QueryPlanCache
	stats  *QueryStats
	mu     sync.RWMutex
}

// QueryPlan represents an analyzed query execution plan
type QueryPlan struct {
	Query        string                   `json:"query"`
	Plan         string                   `json:"plan"`
	Cost         float64                  `json:"cost"`
	Duration     time.Duration            `json:"duration"`
	RowsEstimate int64                    `json:"rows_estimate"`
	RowsActual   int64                    `json:"rows_actual"`
	Operations   []string                 `json:"operations"`
	Indexes      []string                 `json:"indexes"`
	Suggestions  []OptimizationSuggestion `json:"suggestions"`
	CreatedAt    time.Time                `json:"created_at"`
}

// OptimizationSuggestion represents a query optimization recommendation
type OptimizationSuggestion struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion"`
	ImpactLevel string `json:"impact_level"`
}

// QueryPlanCache caches query execution plans
type QueryPlanCache struct {
	plans  map[string]*QueryPlan
	maxAge time.Duration
	mu     sync.RWMutex
}

// QueryStats tracks query performance statistics
type QueryStats struct {
	TotalQueries     int64         `json:"total_queries"`
	SlowQueries      int64         `json:"slow_queries"`
	OptimizedCount   int64         `json:"optimized_count"`
	AvgDuration      time.Duration `json:"avg_duration"`
	MaxDuration      time.Duration `json:"max_duration"`
	MinDuration      time.Duration `json:"min_duration"`
	P95Duration      time.Duration `json:"p95_duration"`
	IndexHitRatio    float64       `json:"index_hit_ratio"`
	TableScans       int64         `json:"table_scans"`
	IndexScans       int64         `json:"index_scans"`
	CacheHitRatio    float64       `json:"cache_hit_ratio"`
	AvgAnalysisTime  time.Duration `json:"avg_analysis_time"`
	IndexSuggestions int64         `json:"index_suggestions"`
	QueryRewrites    int64         `json:"query_rewrites"`
}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer(db *sql.DB, cfg *config.DatabaseConfig) *QueryOptimizer {
	return &QueryOptimizer{
		db:     db,
		config: cfg,
		cache: &QueryPlanCache{
			plans:  make(map[string]*QueryPlan),
			maxAge: time.Hour,
		},
		stats: &QueryStats{},
	}
}

// AnalyzeQuery analyzes a query and returns optimization suggestions
func (qo *QueryOptimizer) AnalyzeQuery(ctx context.Context, query string, args ...interface{}) (*QueryPlan, error) {
	// Check cache first
	if plan := qo.cache.Get(query); plan != nil {
		return plan, nil
	}

	start := time.Now()

	// Get query execution plan
	plan, err := qo.getExecutionPlan(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution plan: %w", err)
	}

	duration := time.Since(start)

	// Analyze the plan
	suggestions := qo.analyzePlan(plan, query)

	queryPlan := &QueryPlan{
		Query:       query,
		Plan:        plan,
		Duration:    duration,
		Operations:  qo.extractOperations(plan),
		Indexes:     qo.extractIndexes(plan),
		Suggestions: suggestions,
		CreatedAt:   time.Now(),
	}

	// Parse cost and row estimates from plan
	qo.parsePlanMetrics(queryPlan, plan)

	// Cache the plan
	qo.cache.Set(query, queryPlan)

	// Update statistics
	qo.updateStats(queryPlan)

	return queryPlan, nil
}

// getExecutionPlan retrieves the PostgreSQL execution plan
func (qo *QueryOptimizer) getExecutionPlan(ctx context.Context, query string, args ...interface{}) (string, error) {
	// Use EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) for detailed analysis
	explainQuery := fmt.Sprintf("EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) %s", query)

	rows, err := qo.db.QueryContext(ctx, explainQuery, args...)
	if err != nil {
		return "", err
	}
	defer func() { _ = rows.Close() }()

	var planJSON string
	for rows.Next() {
		if err := rows.Scan(&planJSON); err != nil {
			return "", err
		}
	}

	return planJSON, rows.Err()
}

// analyzePlan analyzes the execution plan and generates suggestions
func (qo *QueryOptimizer) analyzePlan(plan, query string) []OptimizationSuggestion {
	var suggestions []OptimizationSuggestion

	// Check for sequential scans on large tables
	if strings.Contains(plan, "Seq Scan") {
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "index",
			Severity:    "high",
			Message:     "Sequential scan detected on table",
			Suggestion:  "Consider adding appropriate indexes for WHERE clauses",
			ImpactLevel: "high",
		})
	}

	// Check for nested loops with high cost
	if strings.Contains(plan, "Nested Loop") && strings.Contains(plan, "cost=") {
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "join",
			Severity:    "medium",
			Message:     "Nested loop join detected",
			Suggestion:  "Consider rewriting query to use hash or merge joins",
			ImpactLevel: "medium",
		})
	}

	// Check for missing ORDER BY optimization
	if strings.Contains(strings.ToUpper(query), "ORDER BY") && strings.Contains(plan, "Sort") {
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "index",
			Severity:    "medium",
			Message:     "External sort operation required",
			Suggestion:  "Add index on ORDER BY columns to avoid sorting",
			ImpactLevel: "medium",
		})
	}

	// Check for LIMIT without proper optimization
	if strings.Contains(strings.ToUpper(query), "LIMIT") && !strings.Contains(plan, "Limit") {
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "query",
			Severity:    "low",
			Message:     "LIMIT not pushed down effectively",
			Suggestion:  "Ensure indexes support early result termination",
			ImpactLevel: "low",
		})
	}

	// Check for full table scans on joins
	if strings.Contains(plan, "Hash Join") && strings.Contains(plan, "Seq Scan") {
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "index",
			Severity:    "high",
			Message:     "Hash join with sequential scan",
			Suggestion:  "Add indexes on join columns for better performance",
			ImpactLevel: "high",
		})
	}

	// Check for subquery optimization opportunities
	if strings.Contains(strings.ToUpper(query), "IN (SELECT") {
		suggestions = append(suggestions, OptimizationSuggestion{
			Type:        "query",
			Severity:    "medium",
			Message:     "IN subquery detected",
			Suggestion:  "Consider rewriting as EXISTS or JOIN for better performance",
			ImpactLevel: "medium",
		})
	}

	return suggestions
}

// extractOperations extracts operations from the execution plan
func (qo *QueryOptimizer) extractOperations(plan string) []string {
	operations := []string{}

	// Common PostgreSQL operations
	opPatterns := []string{
		"Seq Scan", "Index Scan", "Index Only Scan", "Bitmap Heap Scan",
		"Nested Loop", "Hash Join", "Merge Join", "Sort", "Hash", "Aggregate",
		"Limit", "Subquery Scan", "CTE Scan", "Function Scan",
	}

	for _, op := range opPatterns {
		if strings.Contains(plan, op) {
			operations = append(operations, op)
		}
	}

	return operations
}

// extractIndexes extracts index names from the execution plan
func (qo *QueryOptimizer) extractIndexes(plan string) []string {
	indexes := []string{}

	// Use regex to find index names in the plan
	indexRegex := regexp.MustCompile(`Index.*?"([^"]+)"`)
	matches := indexRegex.FindAllStringSubmatch(plan, -1)

	for _, match := range matches {
		if len(match) > 1 {
			indexes = append(indexes, match[1])
		}
	}

	return indexes
}

// parsePlanMetrics extracts cost and row estimates from the plan
func (qo *QueryOptimizer) parsePlanMetrics(queryPlan *QueryPlan, plan string) {
	// Parse cost (cost=start..total)
	costRegex := regexp.MustCompile(`cost=[\d.]+\.\.([\d.]+)`)
	if matches := costRegex.FindStringSubmatch(plan); len(matches) > 1 {
		if _, err := fmt.Sscanf(matches[1], "%f", &queryPlan.Cost); err != nil {
			log.Printf("Warning: failed to parse cost from plan: %v", err)
		}
	}

	// Parse row estimates (rows=N)
	rowsRegex := regexp.MustCompile(`rows=(\d+)`)
	if matches := rowsRegex.FindStringSubmatch(plan); len(matches) > 1 {
		if _, err := fmt.Sscanf(matches[1], "%d", &queryPlan.RowsEstimate); err != nil {
			log.Printf("Warning: failed to parse row estimate from plan: %v", err)
		}
	}

	// Parse actual rows (actual.*rows=N)
	actualRowsRegex := regexp.MustCompile(`actual.*rows=(\d+)`)
	if matches := actualRowsRegex.FindStringSubmatch(plan); len(matches) > 1 {
		if _, err := fmt.Sscanf(matches[1], "%d", &queryPlan.RowsActual); err != nil {
			log.Printf("Warning: failed to parse actual rows from plan: %v", err)
		}
	}
}

// updateStats updates query performance statistics
func (qo *QueryOptimizer) updateStats(plan *QueryPlan) {
	qo.mu.Lock()
	defer qo.mu.Unlock()

	qo.stats.TotalQueries++

	// Update duration statistics
	if qo.stats.TotalQueries == 1 {
		qo.stats.MinDuration = plan.Duration
		qo.stats.MaxDuration = plan.Duration
		qo.stats.AvgDuration = plan.Duration
	} else {
		if plan.Duration < qo.stats.MinDuration {
			qo.stats.MinDuration = plan.Duration
		}
		if plan.Duration > qo.stats.MaxDuration {
			qo.stats.MaxDuration = plan.Duration
		}

		// Update rolling average
		qo.stats.AvgDuration = time.Duration(
			(int64(qo.stats.AvgDuration)*(qo.stats.TotalQueries-1) + int64(plan.Duration)) / qo.stats.TotalQueries,
		)
	}

	// Check if query is slow
	if plan.Duration > qo.config.SlowQueryThreshold {
		qo.stats.SlowQueries++
	}

	// Update scan statistics
	for _, op := range plan.Operations {
		switch op {
		case "Seq Scan":
			qo.stats.TableScans++
		case "Index Scan", "Index Only Scan", "Bitmap Heap Scan":
			qo.stats.IndexScans++
		}
	}

	// Calculate index hit ratio
	if qo.stats.TableScans+qo.stats.IndexScans > 0 {
		qo.stats.IndexHitRatio = float64(qo.stats.IndexScans) / float64(qo.stats.TableScans+qo.stats.IndexScans)
	}
}

// GetStats returns current query performance statistics
func (qo *QueryOptimizer) GetStats() *QueryStats {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	// Return a copy to avoid race conditions
	stats := *qo.stats
	return &stats
}

// SuggestIndexes analyzes query patterns and suggests new indexes
func (qo *QueryOptimizer) SuggestIndexes(ctx context.Context) ([]IndexSuggestion, error) {
	suggestions := []IndexSuggestion{}

	// Analyze missing indexes from pg_stat_user_tables
	query := `
		SELECT 
			schemaname,
			tablename,
			seq_scan,
			seq_tup_read,
			idx_scan,
			idx_tup_fetch,
			n_tup_ins + n_tup_upd + n_tup_del as total_writes
		FROM pg_stat_user_tables
		WHERE seq_scan > idx_scan * 2  -- Tables with high sequential scan ratio
		ORDER BY seq_tup_read DESC
		LIMIT 10`

	rows, err := qo.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze table statistics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var schema, table string
		var seqScan, seqTupRead, idxScan, idxTupFetch, totalWrites int64

		err := rows.Scan(&schema, &table, &seqScan, &seqTupRead, &idxScan, &idxTupFetch, &totalWrites)
		if err != nil {
			continue
		}

		suggestion := IndexSuggestion{
			Table:        fmt.Sprintf("%s.%s", schema, table),
			Reason:       "High sequential scan ratio",
			SeqScanRatio: float64(seqScan) / float64(seqScan+idxScan),
			Impact:       "high",
			Suggestion:   fmt.Sprintf("Analyze queries on %s.%s and add appropriate indexes", schema, table),
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, rows.Err()
}

// IndexSuggestion represents an index optimization recommendation
type IndexSuggestion struct {
	Table        string   `json:"table"`
	Columns      []string `json:"columns,omitempty"`
	Reason       string   `json:"reason"`
	SeqScanRatio float64  `json:"seq_scan_ratio"`
	Impact       string   `json:"impact"`
	Suggestion   string   `json:"suggestion"`
}

// Get retrieves a cached query plan
func (qpc *QueryPlanCache) Get(query string) *QueryPlan {
	qpc.mu.RLock()
	defer qpc.mu.RUnlock()

	plan, exists := qpc.plans[query]
	if !exists {
		return nil
	}

	// Check if plan is still valid
	if time.Since(plan.CreatedAt) > qpc.maxAge {
		delete(qpc.plans, query)
		return nil
	}

	return plan
}

// Set stores a query plan in cache
func (qpc *QueryPlanCache) Set(query string, plan *QueryPlan) {
	qpc.mu.Lock()
	defer qpc.mu.Unlock()

	qpc.plans[query] = plan

	// Cleanup old entries periodically
	if len(qpc.plans) > 1000 {
		qpc.cleanup()
	}
}

// cleanup removes expired plans from cache
func (qpc *QueryPlanCache) cleanup() {
	now := time.Now()
	for query, plan := range qpc.plans {
		if now.Sub(plan.CreatedAt) > qpc.maxAge {
			delete(qpc.plans, query)
		}
	}
}

// GetOptimizationReport returns a comprehensive optimization report
func (qo *QueryOptimizer) GetOptimizationReport() map[string]interface{} {
	qo.mu.RLock()
	defer qo.mu.RUnlock()

	return map[string]interface{}{
		"total_queries_analyzed": qo.stats.TotalQueries,
		"slow_queries_detected":  qo.stats.SlowQueries,
		"cache_hit_rate":         qo.stats.CacheHitRatio,
		"average_analysis_time":  qo.stats.AvgAnalysisTime,
		"cached_plans":           len(qo.cache.plans),
		"optimization_suggestions": map[string]interface{}{
			"index_suggestions": qo.stats.IndexSuggestions,
			"query_rewrites":    qo.stats.QueryRewrites,
		},
	}
}

// getEnvBool gets a boolean environment variable with a default value
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return boolValue
}
