package performance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// QueryType represents different types of queries for optimization
type QueryType string

const (
	QueryTypeVector      QueryType = "vector"
	QueryTypeText        QueryType = "text"
	QueryTypeFilter      QueryType = "filter"
	QueryTypeAggregation QueryType = "aggregation"
	QueryTypeJoin        QueryType = "join"
	QueryTypeHybrid      QueryType = "hybrid"
)

// QueryPlan represents an optimized execution plan for a query
type QueryPlan struct {
	ID              string                 `json:"id"`
	QueryHash       string                 `json:"query_hash"`
	QueryType       QueryType              `json:"query_type"`
	EstimatedCost   float64                `json:"estimated_cost"`
	Steps           []QueryStep            `json:"steps"`
	UseCache        bool                   `json:"use_cache"`
	CacheKey        string                 `json:"cache_key"`
	CacheTTL        time.Duration          `json:"cache_ttl"`
	IndexHints      []string               `json:"index_hints"`
	Parameters      map[string]interface{} `json:"parameters"`
	CreatedAt       time.Time              `json:"created_at"`
	LastUsed        time.Time              `json:"last_used"`
	UsageCount      int64                  `json:"usage_count"`
	SuccessRate     float64                `json:"success_rate"`
	AvgLatency      time.Duration          `json:"avg_latency"`
	OptimizationTag string                 `json:"optimization_tag"`
}

// QueryStep represents a step in the query execution plan
type QueryStep struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Operation   string                 `json:"operation"`
	Cost        float64                `json:"cost"`
	Parallelism int                    `json:"parallelism"`
	Parameters  map[string]interface{} `json:"parameters"`
	DependsOn   []string               `json:"depends_on"`
	CanCache    bool                   `json:"can_cache"`
	CacheKey    string                 `json:"cache_key,omitempty"`
}

// QueryOptimizationRule defines optimization rules for different query patterns
type QueryOptimizationRule struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Pattern        string            `json:"pattern"`
	Condition      string            `json:"condition"`
	Transformation string            `json:"transformation"`
	Priority       int               `json:"priority"`
	Tags           []string          `json:"tags"`
	Metadata       map[string]string `json:"metadata"`
	Enabled        bool              `json:"enabled"`
	SuccessRate    float64           `json:"success_rate"`
	LastApplied    time.Time         `json:"last_applied"`
}

// QueryStatistics holds performance statistics for query optimization
type QueryStatistics struct {
	QueryHash        string        `json:"query_hash"`
	TotalExecutions  int64         `json:"total_executions"`
	SuccessfulRuns   int64         `json:"successful_runs"`
	FailedRuns       int64         `json:"failed_runs"`
	TotalLatency     time.Duration `json:"total_latency"`
	MinLatency       time.Duration `json:"min_latency"`
	MaxLatency       time.Duration `json:"max_latency"`
	AvgLatency       time.Duration `json:"avg_latency"`
	P95Latency       time.Duration `json:"p95_latency"`
	P99Latency       time.Duration `json:"p99_latency"`
	LastExecuted     time.Time     `json:"last_executed"`
	LatencyHistory   []float64     `json:"latency_history"`
	ErrorPatterns    []string      `json:"error_patterns"`
	OptimizationHits int64         `json:"optimization_hits"`
	CacheHits        int64         `json:"cache_hits"`
	CacheMisses      int64         `json:"cache_misses"`
}

// QueryOptimizer provides intelligent query optimization with plan caching
type QueryOptimizer struct {
	planCache         *Cache
	statisticsCache   *Cache
	optimizationRules map[string]*QueryOptimizationRule
	queryStats        map[string]*QueryStatistics
	planStore         map[string]*QueryPlan
	
	// Synchronization
	rulesMutex  sync.RWMutex
	statsMutex  sync.RWMutex
	plansMutex  sync.RWMutex
	
	// Configuration
	enabled                  bool
	maxPlanCacheSize        int
	planCacheTTL            time.Duration
	statisticsRetention     time.Duration
	costThreshold           float64
	parallelismThreshold    int
	optimizationInterval    time.Duration
	adaptiveLearningEnabled bool
	
	// Analytics
	totalOptimizations    int64
	successfulOptimizations int64
	avgOptimizationTime   time.Duration
	lastOptimization      time.Time
}

// QueryExecutionContext provides context for query execution
type QueryExecutionContext struct {
	QueryID        string                 `json:"query_id"`
	UserID         string                 `json:"user_id"`
	SessionID      string                 `json:"session_id"`
	Repository     string                 `json:"repository"`
	StartTime      time.Time              `json:"start_time"`
	Timeout        time.Duration          `json:"timeout"`
	Priority       int                    `json:"priority"`
	Tags           []string               `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata"`
	TracingEnabled bool                   `json:"tracing_enabled"`
}

// NewQueryOptimizer creates a new query optimizer with enhanced capabilities
func NewQueryOptimizer(cacheConfig CacheConfig) *QueryOptimizer {
	return &QueryOptimizer{
		planCache:         NewCache(cacheConfig),
		statisticsCache:   NewCache(CacheConfig{MaxSize: 10000, TTL: 24 * time.Hour, EvictionPolicy: "lru", Enabled: true}),
		optimizationRules: make(map[string]*QueryOptimizationRule),
		queryStats:        make(map[string]*QueryStatistics),
		planStore:         make(map[string]*QueryPlan),
		
		enabled:                  true,
		maxPlanCacheSize:        getEnvInt("MCP_MEMORY_QUERY_PLAN_CACHE_SIZE", 5000),
		planCacheTTL:            getEnvDurationMinutes("MCP_MEMORY_QUERY_PLAN_TTL_MINUTES", 60),
		statisticsRetention:     getEnvDurationMinutes("MCP_MEMORY_QUERY_STATS_RETENTION_MINUTES", 1440), // 24 hours
		costThreshold:           getEnvFloat("MCP_MEMORY_QUERY_COST_THRESHOLD", 100.0),
		parallelismThreshold:    getEnvInt("MCP_MEMORY_QUERY_PARALLELISM_THRESHOLD", 4),
		optimizationInterval:    getEnvDurationMinutes("MCP_MEMORY_OPTIMIZATION_INTERVAL_MINUTES", 10),
		adaptiveLearningEnabled: getEnvBool("MCP_MEMORY_ADAPTIVE_LEARNING_ENABLED", true),
		
		totalOptimizations:      0,
		successfulOptimizations: 0,
		lastOptimization:        time.Now(),
	}
}

// OptimizeQuery creates an optimized execution plan for a query
func (qo *QueryOptimizer) OptimizeQuery(ctx context.Context, query string, queryType QueryType, execCtx QueryExecutionContext) (*QueryPlan, error) {
	if !qo.enabled {
		return qo.createBasicPlan(query, queryType), nil
	}

	queryHash := qo.hashQuery(query)
	
	// Check for cached plan first
	if cachedPlan := qo.getCachedPlan(queryHash); cachedPlan != nil {
		qo.recordPlanUsage(cachedPlan)
		return cachedPlan, nil
	}

	// Generate optimized plan
	plan, err := qo.generateOptimizedPlan(ctx, query, queryType, queryHash, execCtx)
	if err != nil {
		return qo.createBasicPlan(query, queryType), err
	}

	// Cache the plan
	qo.cachePlan(plan)
	
	// Update statistics
	qo.updateOptimizationStats(true)
	
	return plan, nil
}

// generateOptimizedPlan creates an optimized query plan using rules and heuristics
func (qo *QueryOptimizer) generateOptimizedPlan(ctx context.Context, query string, queryType QueryType, queryHash string, execCtx QueryExecutionContext) (*QueryPlan, error) {
	startTime := time.Now()
	defer func() {
		qo.avgOptimizationTime = time.Since(startTime)
	}()

	plan := &QueryPlan{
		ID:              generateID(),
		QueryHash:       queryHash,
		QueryType:       queryType,
		EstimatedCost:   0,
		Steps:           []QueryStep{},
		UseCache:        true,
		CacheKey:        qo.generateCacheKey(query, execCtx),
		CacheTTL:        qo.determineCacheTTL(queryType),
		IndexHints:      []string{},
		Parameters:      make(map[string]interface{}),
		CreatedAt:       time.Now(),
		LastUsed:        time.Now(),
		UsageCount:      1,
		SuccessRate:     1.0,
		AvgLatency:      0,
		OptimizationTag: qo.generateOptimizationTag(queryType, execCtx),
	}

	// Apply optimization rules
	qo.applyOptimizationRules(plan, query, execCtx)
	
	// Generate execution steps
	steps := qo.generateExecutionSteps(query, queryType, execCtx)
	plan.Steps = steps
	
	// Calculate estimated cost
	plan.EstimatedCost = qo.calculatePlanCost(plan)
	
	// Determine index hints
	plan.IndexHints = qo.generateIndexHints(query, queryType)
	
	// Set optimization parameters
	plan.Parameters = qo.generateOptimizationParameters(queryType, execCtx)

	return plan, nil
}

// generateExecutionSteps creates optimized execution steps for the query
func (qo *QueryOptimizer) generateExecutionSteps(query string, queryType QueryType, execCtx QueryExecutionContext) []QueryStep {
	steps := []QueryStep{}
	
	switch queryType {
	case QueryTypeVector:
		steps = qo.generateVectorSearchSteps(query, execCtx)
	case QueryTypeText:
		steps = qo.generateTextSearchSteps(query, execCtx)
	case QueryTypeFilter:
		steps = qo.generateFilterSteps(query, execCtx)
	case QueryTypeAggregation:
		steps = qo.generateAggregationSteps(query, execCtx)
	case QueryTypeHybrid:
		steps = qo.generateHybridSearchSteps(query, execCtx)
	case QueryTypeJoin:
		steps = qo.generateJoinSteps(query, execCtx)
	default:
		steps = qo.generateGenericSteps(query, execCtx)
	}
	
	return qo.optimizeStepOrder(steps)
}

// generateVectorSearchSteps creates optimized steps for vector search queries
func (qo *QueryOptimizer) generateVectorSearchSteps(query string, execCtx QueryExecutionContext) []QueryStep {
	steps := []QueryStep{
		{
			ID:          generateID(),
			Type:        "preprocessing",
			Operation:   "validate_vector_input",
			Cost:        1.0,
			Parallelism: 1,
			Parameters:  map[string]interface{}{"timeout": "5s"},
			DependsOn:   []string{},
			CanCache:    false,
		},
		{
			ID:          generateID(),
			Type:        "embedding",
			Operation:   "generate_query_embedding",
			Cost:        10.0,
			Parallelism: 1,
			Parameters:  map[string]interface{}{"model": "text-embedding-ada-002", "cache_enabled": true},
			DependsOn:   []string{},
			CanCache:    true,
			CacheKey:    fmt.Sprintf("embedding:%s", qo.hashQuery(query)),
		},
		{
			ID:          generateID(),
			Type:        "search",
			Operation:   "vector_similarity_search",
			Cost:        25.0,
			Parallelism: qo.determineOptimalParallelism(execCtx),
			Parameters:  map[string]interface{}{"top_k": 50, "ef_search": 128, "use_index": true},
			DependsOn:   []string{},
			CanCache:    true,
			CacheKey:    fmt.Sprintf("vector_search:%s", qo.hashQuery(query)),
		},
		{
			ID:          generateID(),
			Type:        "postprocessing",
			Operation:   "rank_and_filter_results",
			Cost:        5.0,
			Parallelism: 1,
			Parameters:  map[string]interface{}{"rerank": true, "diversity_threshold": 0.8},
			DependsOn:   []string{},
			CanCache:    false,
		},
	}
	
	return steps
}

// generateTextSearchSteps creates optimized steps for text search queries
func (qo *QueryOptimizer) generateTextSearchSteps(query string, execCtx QueryExecutionContext) []QueryStep {
	steps := []QueryStep{
		{
			ID:          generateID(),
			Type:        "preprocessing",
			Operation:   "analyze_text_query",
			Cost:        2.0,
			Parallelism: 1,
			Parameters:  map[string]interface{}{"tokenize": true, "stemming": true, "stop_words": true},
			DependsOn:   []string{},
			CanCache:    true,
			CacheKey:    fmt.Sprintf("text_analysis:%s", qo.hashQuery(query)),
		},
		{
			ID:          generateID(),
			Type:        "search",
			Operation:   "full_text_search",
			Cost:        15.0,
			Parallelism: qo.determineOptimalParallelism(execCtx),
			Parameters:  map[string]interface{}{"fuzzy": true, "boost_fields": []string{"title", "tags"}},
			DependsOn:   []string{},
			CanCache:    true,
			CacheKey:    fmt.Sprintf("text_search:%s", qo.hashQuery(query)),
		},
		{
			ID:          generateID(),
			Type:        "postprocessing",
			Operation:   "score_and_rank",
			Cost:        3.0,
			Parallelism: 1,
			Parameters:  map[string]interface{}{"tf_idf": true, "semantic_boost": 0.3},
			DependsOn:   []string{},
			CanCache:    false,
		},
	}
	
	return steps
}

// generateFilterSteps creates optimized steps for filter queries
func (qo *QueryOptimizer) generateFilterSteps(query string, execCtx QueryExecutionContext) []QueryStep {
	steps := []QueryStep{
		{
			ID:          generateID(),
			Type:        "preprocessing",
			Operation:   "parse_filter_conditions",
			Cost:        1.0,
			Parallelism: 1,
			Parameters:  map[string]interface{}{"validate": true},
			DependsOn:   []string{},
			CanCache:    true,
			CacheKey:    fmt.Sprintf("filter_parse:%s", qo.hashQuery(query)),
		},
		{
			ID:          generateID(),
			Type:        "optimization",
			Operation:   "optimize_filter_order",
			Cost:        2.0,
			Parallelism: 1,
			Parameters:  map[string]interface{}{"selectivity_based": true},
			DependsOn:   []string{},
			CanCache:    false,
		},
		{
			ID:          generateID(),
			Type:        "execution",
			Operation:   "apply_filters",
			Cost:        8.0,
			Parallelism: qo.determineOptimalParallelism(execCtx),
			Parameters:  map[string]interface{}{"batch_size": 1000, "use_index": true},
			DependsOn:   []string{},
			CanCache:    true,
			CacheKey:    fmt.Sprintf("filter_result:%s", qo.hashQuery(query)),
		},
	}
	
	return steps
}

// generateAggregationSteps creates optimized steps for aggregation queries
func (qo *QueryOptimizer) generateAggregationSteps(query string, execCtx QueryExecutionContext) []QueryStep {
	steps := []QueryStep{
		{
			ID:          generateID(),
			Type:        "preprocessing",
			Operation:   "parse_aggregation_query",
			Cost:        2.0,
			Parallelism: 1,
			Parameters:  map[string]interface{}{"validate_fields": true},
			DependsOn:   []string{},
			CanCache:    true,
			CacheKey:    fmt.Sprintf("agg_parse:%s", qo.hashQuery(query)),
		},
		{
			ID:          generateID(),
			Type:        "data_access",
			Operation:   "scan_collection",
			Cost:        20.0,
			Parallelism: qo.determineOptimalParallelism(execCtx),
			Parameters:  map[string]interface{}{"streaming": true, "batch_size": 5000},
			DependsOn:   []string{},
			CanCache:    false,
		},
		{
			ID:          generateID(),
			Type:        "computation",
			Operation:   "compute_aggregates",
			Cost:        10.0,
			Parallelism: qo.parallelismThreshold,
			Parameters:  map[string]interface{}{"in_memory": true, "spill_to_disk": false},
			DependsOn:   []string{},
			CanCache:    true,
			CacheKey:    fmt.Sprintf("agg_result:%s", qo.hashQuery(query)),
		},
	}
	
	return steps
}

// generateHybridSearchSteps creates optimized steps for hybrid search queries
func (qo *QueryOptimizer) generateHybridSearchSteps(query string, execCtx QueryExecutionContext) []QueryStep {
	vectorSteps := qo.generateVectorSearchSteps(query, execCtx)
	textSteps := qo.generateTextSearchSteps(query, execCtx)
	
	// Add fusion step
	fusionStep := QueryStep{
		ID:          generateID(),
		Type:        "fusion",
		Operation:   "hybrid_score_fusion",
		Cost:        8.0,
		Parallelism: 1,
		Parameters: map[string]interface{}{
			"vector_weight": 0.6,
			"text_weight":   0.4,
			"fusion_method": "rrf", // Reciprocal Rank Fusion
		},
		DependsOn: []string{},
		CanCache:  false,
	}
	
	steps := append(vectorSteps, textSteps...)
	steps = append(steps, fusionStep)
	
	return steps
}

// generateJoinSteps creates optimized steps for join queries
func (qo *QueryOptimizer) generateJoinSteps(query string, execCtx QueryExecutionContext) []QueryStep {
	steps := []QueryStep{
		{
			ID:          generateID(),
			Type:        "preprocessing",
			Operation:   "parse_join_conditions",
			Cost:        3.0,
			Parallelism: 1,
			Parameters:  map[string]interface{}{"validate_keys": true},
			DependsOn:   []string{},
			CanCache:    true,
			CacheKey:    fmt.Sprintf("join_parse:%s", qo.hashQuery(query)),
		},
		{
			ID:          generateID(),
			Type:        "optimization",
			Operation:   "optimize_join_order",
			Cost:        5.0,
			Parallelism: 1,
			Parameters:  map[string]interface{}{"cost_based": true},
			DependsOn:   []string{},
			CanCache:    false,
		},
		{
			ID:          generateID(),
			Type:        "execution",
			Operation:   "execute_join",
			Cost:        25.0,
			Parallelism: qo.determineOptimalParallelism(execCtx),
			Parameters:  map[string]interface{}{"algorithm": "hash_join", "batch_size": 1000},
			DependsOn:   []string{},
			CanCache:    true,
			CacheKey:    fmt.Sprintf("join_result:%s", qo.hashQuery(query)),
		},
	}
	
	return steps
}

// generateGenericSteps creates basic steps for unknown query types
func (qo *QueryOptimizer) generateGenericSteps(query string, execCtx QueryExecutionContext) []QueryStep {
	return []QueryStep{
		{
			ID:          generateID(),
			Type:        "execution",
			Operation:   "generic_query_execution",
			Cost:        10.0,
			Parallelism: 1,
			Parameters:  map[string]interface{}{"timeout": "30s"},
			DependsOn:   []string{},
			CanCache:    false,
		},
	}
}

// optimizeStepOrder reorders steps for optimal execution
func (qo *QueryOptimizer) optimizeStepOrder(steps []QueryStep) []QueryStep {
	// Sort by cost (execute cheaper operations first)
	sort.Slice(steps, func(i, j int) bool {
		// If one step depends on another, respect that ordering
		for _, dep := range steps[j].DependsOn {
			if dep == steps[i].ID {
				return true
			}
		}
		for _, dep := range steps[i].DependsOn {
			if dep == steps[j].ID {
				return false
			}
		}
		
		// Otherwise, sort by cost
		return steps[i].Cost < steps[j].Cost
	})
	
	return steps
}

// applyOptimizationRules applies registered optimization rules to the plan
func (qo *QueryOptimizer) applyOptimizationRules(plan *QueryPlan, query string, execCtx QueryExecutionContext) {
	qo.rulesMutex.RLock()
	defer qo.rulesMutex.RUnlock()
	
	for _, rule := range qo.optimizationRules {
		if !rule.Enabled {
			continue
		}
		
		if qo.ruleMatches(rule, query, plan.QueryType, execCtx) {
			qo.applyRule(plan, rule)
			rule.LastApplied = time.Now()
		}
	}
}

// ruleMatches checks if an optimization rule applies to the current query
func (qo *QueryOptimizer) ruleMatches(rule *QueryOptimizationRule, query string, queryType QueryType, execCtx QueryExecutionContext) bool {
	// Simple pattern matching - in production, use more sophisticated matching
	if rule.Pattern != "" && !strings.Contains(query, rule.Pattern) {
		return false
	}
	
	// Check conditions
	switch rule.Condition {
	case "vector_query":
		return queryType == QueryTypeVector
	case "high_priority":
		return execCtx.Priority > 5
	case "has_repository":
		return execCtx.Repository != ""
	default:
		return true
	}
}

// applyRule applies a specific optimization rule to the plan
func (qo *QueryOptimizer) applyRule(plan *QueryPlan, rule *QueryOptimizationRule) {
	switch rule.Transformation {
	case "enable_caching":
		plan.UseCache = true
		plan.CacheTTL = 30 * time.Minute
	case "increase_parallelism":
		for i := range plan.Steps {
			if plan.Steps[i].Parallelism < qo.parallelismThreshold {
				plan.Steps[i].Parallelism = qo.parallelismThreshold
			}
		}
	case "optimize_for_latency":
		plan.CacheTTL = 5 * time.Minute
		for i := range plan.Steps {
			if plan.Steps[i].Type == "preprocessing" {
				plan.Steps[i].Parameters["fast_mode"] = true
			}
		}
	case "add_index_hint":
		if hint, exists := rule.Metadata["index_name"]; exists {
			plan.IndexHints = append(plan.IndexHints, hint)
		}
	}
}

// RecordQueryExecution records statistics for a completed query execution
func (qo *QueryOptimizer) RecordQueryExecution(queryHash string, duration time.Duration, success bool, errorMsg string) {
	qo.statsMutex.Lock()
	defer qo.statsMutex.Unlock()
	
	stats, exists := qo.queryStats[queryHash]
	if !exists {
		stats = &QueryStatistics{
			QueryHash:       queryHash,
			LatencyHistory:  make([]float64, 0),
			ErrorPatterns:   make([]string, 0),
			MinLatency:      duration,
			MaxLatency:      duration,
		}
		qo.queryStats[queryHash] = stats
	}
	
	stats.TotalExecutions++
	stats.LastExecuted = time.Now()
	stats.TotalLatency += duration
	
	if success {
		stats.SuccessfulRuns++
	} else {
		stats.FailedRuns++
		if errorMsg != "" {
			stats.ErrorPatterns = append(stats.ErrorPatterns, errorMsg)
		}
	}
	
	// Update latency statistics
	latencyMs := float64(duration) / float64(time.Millisecond)
	stats.LatencyHistory = append(stats.LatencyHistory, latencyMs)
	
	// Keep only recent history
	if len(stats.LatencyHistory) > 1000 {
		stats.LatencyHistory = stats.LatencyHistory[len(stats.LatencyHistory)-500:]
	}
	
	// Update min/max/avg latencies
	if duration < stats.MinLatency {
		stats.MinLatency = duration
	}
	if duration > stats.MaxLatency {
		stats.MaxLatency = duration
	}
	if stats.TotalExecutions > 0 {
		stats.AvgLatency = stats.TotalLatency / time.Duration(stats.TotalExecutions)
	}
	
	// Calculate percentiles
	qo.updatePercentiles(stats)
}

// updatePercentiles calculates P95 and P99 latencies from history
func (qo *QueryOptimizer) updatePercentiles(stats *QueryStatistics) {
	if len(stats.LatencyHistory) == 0 {
		return
	}
	
	// Sort latency history
	sorted := make([]float64, len(stats.LatencyHistory))
	copy(sorted, stats.LatencyHistory)
	sort.Float64s(sorted)
	
	// Calculate percentiles
	p95Index := int(0.95 * float64(len(sorted)))
	p99Index := int(0.99 * float64(len(sorted)))
	
	if p95Index < len(sorted) {
		stats.P95Latency = time.Duration(sorted[p95Index]) * time.Millisecond
	}
	if p99Index < len(sorted) {
		stats.P99Latency = time.Duration(sorted[p99Index]) * time.Millisecond
	}
}

// GetQueryStatistics returns performance statistics for all queries
func (qo *QueryOptimizer) GetQueryStatistics() map[string]*QueryStatistics {
	qo.statsMutex.RLock()
	defer qo.statsMutex.RUnlock()
	
	result := make(map[string]*QueryStatistics)
	for k, v := range qo.queryStats {
		result[k] = v
	}
	
	return result
}

// GetOptimizationSuggestions provides optimization suggestions based on query patterns
func (qo *QueryOptimizer) GetOptimizationSuggestions() []map[string]interface{} {
	suggestions := []map[string]interface{}{}
	
	qo.statsMutex.RLock()
	defer qo.statsMutex.RUnlock()
	
	for queryHash, stats := range qo.queryStats {
		if stats.TotalExecutions < 10 {
			continue // Need more data for meaningful suggestions
		}
		
		// High latency queries
		if stats.AvgLatency > 5*time.Second {
			suggestions = append(suggestions, map[string]interface{}{
				"type":        "high_latency",
				"query_hash":  queryHash,
				"avg_latency": stats.AvgLatency.String(),
				"suggestion":  "Consider adding caching or optimizing query structure",
				"priority":    "high",
			})
		}
		
		// Low success rate queries
		successRate := float64(stats.SuccessfulRuns) / float64(stats.TotalExecutions)
		if successRate < 0.9 {
			suggestions = append(suggestions, map[string]interface{}{
				"type":         "low_success_rate",
				"query_hash":   queryHash,
				"success_rate": fmt.Sprintf("%.2f%%", successRate*100),
				"suggestion":   "Review error patterns and add error handling",
				"priority":     "medium",
			})
		}
		
		// Frequently executed queries that could benefit from caching
		if stats.TotalExecutions > 100 && stats.CacheHits == 0 {
			suggestions = append(suggestions, map[string]interface{}{
				"type":        "cache_candidate",
				"query_hash":  queryHash,
				"executions":  stats.TotalExecutions,
				"suggestion":  "Enable caching for this frequently executed query",
				"priority":    "medium",
			})
		}
	}
	
	return suggestions
}

// Helper methods

func (qo *QueryOptimizer) hashQuery(query string) string {
	hash := sha256.Sum256([]byte(query))
	return hex.EncodeToString(hash[:])
}

func (qo *QueryOptimizer) getCachedPlan(queryHash string) *QueryPlan {
	if plan, exists := qo.planCache.Get(queryHash); exists {
		if p, ok := plan.(*QueryPlan); ok {
			return p
		}
	}
	return nil
}

func (qo *QueryOptimizer) cachePlan(plan *QueryPlan) {
	qo.planCache.Set(plan.QueryHash, plan)
	
	qo.plansMutex.Lock()
	qo.planStore[plan.QueryHash] = plan
	qo.plansMutex.Unlock()
}

func (qo *QueryOptimizer) recordPlanUsage(plan *QueryPlan) {
	plan.LastUsed = time.Now()
	plan.UsageCount++
}

func (qo *QueryOptimizer) createBasicPlan(query string, queryType QueryType) *QueryPlan {
	return &QueryPlan{
		ID:        generateID(),
		QueryHash: qo.hashQuery(query),
		QueryType: queryType,
		Steps: []QueryStep{
			{
				ID:          generateID(),
				Type:        "execution",
				Operation:   "basic_execution",
				Cost:        10.0,
				Parallelism: 1,
				Parameters:  make(map[string]interface{}),
				DependsOn:   []string{},
				CanCache:    false,
			},
		},
		UseCache:        false,
		CreatedAt:       time.Now(),
		LastUsed:        time.Now(),
		UsageCount:      1,
		SuccessRate:     1.0,
		OptimizationTag: "basic",
	}
}

func (qo *QueryOptimizer) generateCacheKey(query string, execCtx QueryExecutionContext) string {
	return fmt.Sprintf("query:%s:%s:%s", qo.hashQuery(query), execCtx.Repository, execCtx.UserID)
}

func (qo *QueryOptimizer) determineCacheTTL(queryType QueryType) time.Duration {
	switch queryType {
	case QueryTypeVector:
		return 30 * time.Minute
	case QueryTypeText:
		return 15 * time.Minute
	case QueryTypeFilter:
		return 60 * time.Minute
	case QueryTypeAggregation:
		return 5 * time.Minute
	case QueryTypeJoin:
		return 20 * time.Minute
	case QueryTypeHybrid:
		return 25 * time.Minute
	default:
		return 10 * time.Minute
	}
}

func (qo *QueryOptimizer) determineOptimalParallelism(execCtx QueryExecutionContext) int {
	if execCtx.Priority > 8 {
		return qo.parallelismThreshold * 2
	} else if execCtx.Priority > 5 {
		return qo.parallelismThreshold
	}
	return 1
}

func (qo *QueryOptimizer) calculatePlanCost(plan *QueryPlan) float64 {
	totalCost := 0.0
	for _, step := range plan.Steps {
		totalCost += step.Cost
	}
	return totalCost
}

func (qo *QueryOptimizer) generateIndexHints(query string, queryType QueryType) []string {
	hints := []string{}
	
	switch queryType {
	case QueryTypeVector:
		hints = append(hints, "use_vector_index", "hnsw_ef_search_128")
	case QueryTypeText:
		hints = append(hints, "use_text_index", "enable_fuzzy_search")
	case QueryTypeFilter:
		hints = append(hints, "use_filter_index", "optimize_range_queries")
	case QueryTypeJoin:
		hints = append(hints, "use_join_index", "optimize_hash_join")
	case QueryTypeHybrid:
		hints = append(hints, "use_hybrid_index", "enable_parallel_search")
	case QueryTypeAggregation:
		hints = append(hints, "use_aggregation_index", "enable_columnar_scan")
	}
	
	return hints
}

func (qo *QueryOptimizer) generateOptimizationParameters(queryType QueryType, execCtx QueryExecutionContext) map[string]interface{} {
	params := make(map[string]interface{})
	
	params["query_type"] = string(queryType)
	params["priority"] = execCtx.Priority
	params["timeout"] = execCtx.Timeout.String()
	params["parallelism_enabled"] = execCtx.Priority > 5
	params["caching_enabled"] = true
	
	return params
}

func (qo *QueryOptimizer) generateOptimizationTag(queryType QueryType, execCtx QueryExecutionContext) string {
	tag := string(queryType)
	
	if execCtx.Priority > 8 {
		tag += "_high_priority"
	}
	if execCtx.Repository != "" {
		tag += "_repo_scoped"
	}
	
	return tag
}

func (qo *QueryOptimizer) updateOptimizationStats(success bool) {
	qo.totalOptimizations++
	if success {
		qo.successfulOptimizations++
	}
	qo.lastOptimization = time.Now()
}

func generateID() string {
	return fmt.Sprintf("opt_%d", time.Now().UnixNano())
}

func getEnvBool(key string, defaultValue bool) bool {
	// Simple implementation - in production, parse environment variable
	return defaultValue
}

// AddOptimizationRule adds a new optimization rule
func (qo *QueryOptimizer) AddOptimizationRule(rule QueryOptimizationRule) {
	qo.rulesMutex.Lock()
	defer qo.rulesMutex.Unlock()
	
	rule.Enabled = true
	qo.optimizationRules[rule.ID] = &rule
}

// GetOptimizationReport provides a comprehensive optimization report
func (qo *QueryOptimizer) GetOptimizationReport() map[string]interface{} {
	qo.statsMutex.RLock()
	defer qo.statsMutex.RUnlock()
	
	report := map[string]interface{}{
		"total_optimizations":      qo.totalOptimizations,
		"successful_optimizations": qo.successfulOptimizations,
		"success_rate":             0.0,
		"avg_optimization_time":    qo.avgOptimizationTime.String(),
		"last_optimization":        qo.lastOptimization,
		"plan_cache_stats":         qo.planCache.GetStats(),
		"total_queries_tracked":    len(qo.queryStats),
		"optimization_suggestions": qo.GetOptimizationSuggestions(),
	}
	
	if qo.totalOptimizations > 0 {
		report["success_rate"] = float64(qo.successfulOptimizations) / float64(qo.totalOptimizations)
	}
	
	return report
}