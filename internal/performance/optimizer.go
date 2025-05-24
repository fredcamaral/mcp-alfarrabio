package performance

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MetricType represents different types of performance metrics
type MetricType string

const (
	MetricTypeLatency    MetricType = "latency"
	MetricTypeThroughput MetricType = "throughput"
	MetricTypeMemory     MetricType = "memory"
	MetricTypeCPU        MetricType = "cpu"
	MetricTypeCache      MetricType = "cache"
	MetricTypeDatabase   MetricType = "database"
	MetricTypeVector     MetricType = "vector"
)

// PerformanceMetric represents a performance measurement
type PerformanceMetric struct {
	Name        string                 `json:"name"`
	Type        MetricType             `json:"type"`
	Value       float64                `json:"value"`
	Unit        string                 `json:"unit"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     map[string]any         `json:"context"`
	Threshold   float64                `json:"threshold"`
	IsHealthy   bool                   `json:"is_healthy"`
}

// OptimizationRule represents a rule for optimization
type OptimizationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Condition   string                 `json:"condition"`
	Action      string                 `json:"action"`
	Parameters  map[string]any         `json:"parameters"`
	Priority    int                    `json:"priority"`
	IsActive    bool                   `json:"is_active"`
	Enabled     bool                   `json:"enabled"`
	LastApplied time.Time              `json:"last_applied"`
	SuccessRate float64                `json:"success_rate"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	MaxSize        int           `json:"max_size"`
	TTL            time.Duration `json:"ttl"`
	EvictionPolicy string        `json:"eviction_policy"` // "lru", "lfu", "fifo"
	Enabled        bool          `json:"enabled"`
}

// Cache represents a simple in-memory cache
type Cache struct {
	data       map[string]*CacheItem
	mutex      sync.RWMutex
	config     CacheConfig
	hits       int64
	misses     int64
	evictions  int64
}

// CacheItem represents an item in the cache
type CacheItem struct {
	Key        string      `json:"key"`
	Value      any         `json:"value"`
	CreatedAt  time.Time   `json:"created_at"`
	AccessedAt time.Time   `json:"accessed_at"`
	AccessCount int64      `json:"access_count"`
	Size       int         `json:"size"`
}

// PerformanceOptimizer manages performance optimizations
type PerformanceOptimizer struct {
	// Caches
	vectorCache   *Cache
	patternCache  *Cache
	queryCache    *Cache
	
	// Metrics
	metrics       map[string]*PerformanceMetric
	metricsMutex  sync.RWMutex
	
	// Optimization rules
	rules         map[string]*OptimizationRule
	rulesMutex    sync.RWMutex
	
	// Configuration
	enabled           bool
	metricsInterval   time.Duration
	optimizeInterval  time.Duration
	
	// State
	lastOptimization  time.Time
	optimizationCount int64
}

// BatchProcessor handles batch operations for better performance
type BatchProcessor struct {
	batchSize    int
	flushInterval time.Duration
	queue        []BatchItem
	queueMutex   sync.Mutex
	processor    func([]BatchItem) error
	lastFlush    time.Time
}

// BatchItem represents an item in a batch
type BatchItem struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Data      any         `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// ConnectionPool manages database connections efficiently
type ConnectionPool struct {
	connections   []any
	available     chan any
	mutex         sync.Mutex
	maxSize       int
	currentSize   int
	created       int64
	borrowed      int64
	returned      int64
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer() *PerformanceOptimizer {
	return &PerformanceOptimizer{
		vectorCache:       NewCache(CacheConfig{MaxSize: 1000, TTL: 30 * time.Minute, EvictionPolicy: "lru", Enabled: true}),
		patternCache:      NewCache(CacheConfig{MaxSize: 500, TTL: 1 * time.Hour, EvictionPolicy: "lfu", Enabled: true}),
		queryCache:        NewCache(CacheConfig{MaxSize: 200, TTL: 15 * time.Minute, EvictionPolicy: "lru", Enabled: true}),
		metrics:           make(map[string]*PerformanceMetric),
		rules:             make(map[string]*OptimizationRule),
		enabled:           true,
		metricsInterval:   30 * time.Second,
		optimizeInterval:  5 * time.Minute,
		lastOptimization:  time.Now(),
		optimizationCount: 0,
	}
}

// NewCache creates a new cache with the given configuration
func NewCache(config CacheConfig) *Cache {
	return &Cache{
		data:   make(map[string]*CacheItem),
		config: config,
		hits:   0,
		misses: 0,
		evictions: 0,
	}
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) (any, bool) {
	if !c.config.Enabled {
		return nil, false
	}
	
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	item, exists := c.data[key]
	if !exists {
		c.misses++
		return nil, false
	}
	
	// Check TTL
	if time.Since(item.CreatedAt) > c.config.TTL {
		c.mutex.RUnlock()
		c.mutex.Lock()
		delete(c.data, key)
		c.mutex.Unlock()
		c.mutex.RLock()
		c.misses++
		return nil, false
	}
	
	// Update access information
	item.AccessedAt = time.Now()
	item.AccessCount++
	c.hits++
	
	return item.Value, true
}

// Set stores an item in the cache
func (c *Cache) Set(key string, value any) {
	if !c.config.Enabled {
		return
	}
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Check if we need to evict items
	if len(c.data) >= c.config.MaxSize {
		c.evict()
	}
	
	c.data[key] = &CacheItem{
		Key:         key,
		Value:       value,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
		Size:        c.estimateSize(value),
	}
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	delete(c.data, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.data = make(map[string]*CacheItem)
	c.hits = 0
	c.misses = 0
	c.evictions = 0
}

// GetStats returns cache statistics
func (c *Cache) GetStats() map[string]any {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	total := c.hits + c.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}
	
	return map[string]any{
		"size":       len(c.data),
		"max_size":   c.config.MaxSize,
		"hits":       c.hits,
		"misses":     c.misses,
		"hit_rate":   hitRate,
		"evictions":  c.evictions,
		"enabled":    c.config.Enabled,
	}
}

// evict removes items based on the eviction policy
func (c *Cache) evict() {
	if len(c.data) == 0 {
		return
	}
	
	switch c.config.EvictionPolicy {
	case "lru":
		c.evictLRU()
	case "lfu":
		c.evictLFU()
	case "fifo":
		c.evictFIFO()
	default:
		c.evictLRU()
	}
	
	c.evictions++
}

func (c *Cache) evictLRU() {
	var oldestKey string
	oldestTime := time.Now()
	
	for key, item := range c.data {
		if item.AccessedAt.Before(oldestTime) {
			oldestTime = item.AccessedAt
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(c.data, oldestKey)
	}
}

func (c *Cache) evictLFU() {
	var lfuKey string
	var minAccess int64 = -1
	
	for key, item := range c.data {
		if minAccess == -1 || item.AccessCount < minAccess {
			minAccess = item.AccessCount
			lfuKey = key
		}
	}
	
	if lfuKey != "" {
		delete(c.data, lfuKey)
	}
}

func (c *Cache) evictFIFO() {
	var oldestKey string
	oldestTime := time.Now()
	
	for key, item := range c.data {
		if item.CreatedAt.Before(oldestTime) {
			oldestTime = item.CreatedAt
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		delete(c.data, oldestKey)
	}
}

func (c *Cache) estimateSize(value any) int {
	// Simple size estimation - in production, use more sophisticated methods
	switch v := value.(type) {
	case string:
		return len(v)
	case []byte:
		return len(v)
	default:
		return 64 // Default estimate
	}
}

// RecordMetric records a performance metric
func (po *PerformanceOptimizer) RecordMetric(metric PerformanceMetric) {
	if !po.enabled {
		return
	}
	
	metric.Timestamp = time.Now()
	metric.IsHealthy = metric.Value <= metric.Threshold
	
	po.metricsMutex.Lock()
	po.metrics[metric.Name] = &metric
	po.metricsMutex.Unlock()
}

// GetMetrics returns all current metrics
func (po *PerformanceOptimizer) GetMetrics() map[string]*PerformanceMetric {
	po.metricsMutex.RLock()
	defer po.metricsMutex.RUnlock()
	
	result := make(map[string]*PerformanceMetric)
	for k, v := range po.metrics {
		result[k] = v
	}
	
	return result
}

// AddOptimizationRule adds a new optimization rule
func (po *PerformanceOptimizer) AddOptimizationRule(rule OptimizationRule) {
	po.rulesMutex.Lock()
	defer po.rulesMutex.Unlock()
	
	rule.Enabled = true
	rule.IsActive = true
	po.rules[rule.ID] = &rule
}

// ApplyOptimizations applies optimization rules based on current metrics
func (po *PerformanceOptimizer) ApplyOptimizations(ctx context.Context) error {
	if !po.enabled || time.Since(po.lastOptimization) < po.optimizeInterval {
		return nil
	}
	
	po.rulesMutex.RLock()
	var applicableRules []*OptimizationRule
	for _, rule := range po.rules {
		if rule.Enabled && rule.IsActive && po.ruleApplies(rule) {
			applicableRules = append(applicableRules, rule)
		}
	}
	po.rulesMutex.RUnlock()
	
	// Sort by priority
	sort.Slice(applicableRules, func(i, j int) bool {
		return applicableRules[i].Priority > applicableRules[j].Priority
	})
	
	// Apply rules
	for _, rule := range applicableRules {
		err := po.applyRule(ctx, rule)
		if err != nil {
			continue // Log error in production
		}
		
		rule.LastApplied = time.Now()
		po.optimizationCount++
	}
	
	po.lastOptimization = time.Now()
	return nil
}

// GetCacheStats returns statistics for all caches
func (po *PerformanceOptimizer) GetCacheStats() map[string]any {
	return map[string]any{
		"vector_cache":  po.vectorCache.GetStats(),
		"pattern_cache": po.patternCache.GetStats(),
		"query_cache":   po.queryCache.GetStats(),
	}
}

// GetVectorCache returns the vector cache for external use
func (po *PerformanceOptimizer) GetVectorCache() *Cache {
	return po.vectorCache
}

// GetPatternCache returns the pattern cache for external use
func (po *PerformanceOptimizer) GetPatternCache() *Cache {
	return po.patternCache
}

// GetQueryCache returns the query cache for external use
func (po *PerformanceOptimizer) GetQueryCache() *Cache {
	return po.queryCache
}

// OptimizeQueries optimizes query performance
func (po *PerformanceOptimizer) OptimizeQueries(ctx context.Context) error {
	// Clear old cache entries
	po.cleanupCaches()
	
	// Optimize vector searches
	return po.optimizeVectorSearches(ctx)
}

// Helper methods

func (po *PerformanceOptimizer) ruleApplies(rule *OptimizationRule) bool {
	po.metricsMutex.RLock()
	defer po.metricsMutex.RUnlock()
	
	// Simple rule evaluation - in production, use a more sophisticated engine
	switch rule.Condition {
	case "high_latency":
		if metric, exists := po.metrics["query_latency"]; exists {
			return metric.Value > metric.Threshold
		}
	case "low_cache_hit_rate":
		stats := po.vectorCache.GetStats()
		if hitRate, ok := stats["hit_rate"].(float64); ok {
			return hitRate < 0.7
		}
	case "high_memory_usage":
		if metric, exists := po.metrics["memory_usage"]; exists {
			return metric.Value > metric.Threshold
		}
	}
	
	return false
}

func (po *PerformanceOptimizer) applyRule(ctx context.Context, rule *OptimizationRule) error {
	switch rule.Action {
	case "increase_cache_size":
		if size, ok := rule.Parameters["size"].(int); ok {
			po.vectorCache.config.MaxSize = size
			return nil
		}
	case "clear_cache":
		po.vectorCache.Clear()
		return nil
	case "reduce_batch_size":
		// This would be implemented with actual batch processors
		return nil
	case "optimize_queries":
		return po.OptimizeQueries(ctx)
	}
	
	return fmt.Errorf("unknown optimization action: %s", rule.Action)
}

func (po *PerformanceOptimizer) cleanupCaches() {
	// Clean up expired entries in all caches
	caches := []*Cache{po.vectorCache, po.patternCache, po.queryCache}
	
	for _, cache := range caches {
		cache.mutex.Lock()
		for key, item := range cache.data {
			if time.Since(item.CreatedAt) > cache.config.TTL {
				delete(cache.data, key)
			}
		}
		cache.mutex.Unlock()
	}
}

func (po *PerformanceOptimizer) optimizeVectorSearches(_ context.Context) error {
	// Implementation would include:
	// - Index optimization
	// - Query plan optimization
	// - Parallel search optimization
	// For now, this is a placeholder
	return nil
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int, flushInterval time.Duration, processor func([]BatchItem) error) *BatchProcessor {
	bp := &BatchProcessor{
		batchSize:     batchSize,
		flushInterval: flushInterval,
		queue:         make([]BatchItem, 0, batchSize),
		processor:     processor,
		lastFlush:     time.Now(),
	}
	
	// Start background flusher
	go bp.backgroundFlush()
	
	return bp
}

// Add adds an item to the batch queue
func (bp *BatchProcessor) Add(item BatchItem) error {
	bp.queueMutex.Lock()
	defer bp.queueMutex.Unlock()
	
	bp.queue = append(bp.queue, item)
	
	// Flush if batch is full
	if len(bp.queue) >= bp.batchSize {
		return bp.flush()
	}
	
	return nil
}

// Flush processes all queued items
func (bp *BatchProcessor) Flush() error {
	bp.queueMutex.Lock()
	defer bp.queueMutex.Unlock()
	
	return bp.flush()
}

func (bp *BatchProcessor) flush() error {
	if len(bp.queue) == 0 {
		return nil
	}
	
	items := make([]BatchItem, len(bp.queue))
	copy(items, bp.queue)
	bp.queue = bp.queue[:0] // Clear queue
	bp.lastFlush = time.Now()
	
	return bp.processor(items)
}

func (bp *BatchProcessor) backgroundFlush() {
	ticker := time.NewTicker(bp.flushInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		bp.queueMutex.Lock()
		if len(bp.queue) > 0 && time.Since(bp.lastFlush) >= bp.flushInterval {
			if err := bp.flush(); err != nil {
				// Log error but continue processing
				_ = err
			}
		}
		bp.queueMutex.Unlock()
	}
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int, factory func() (any, error)) *ConnectionPool {
	return &ConnectionPool{
		connections: make([]any, 0, maxSize),
		available:   make(chan any, maxSize),
		maxSize:     maxSize,
		currentSize: 0,
	}
}

// Get borrows a connection from the pool
func (cp *ConnectionPool) Get() (any, error) {
	select {
	case conn := <-cp.available:
		cp.borrowed++
		return conn, nil
	default:
		cp.mutex.Lock()
		defer cp.mutex.Unlock()
		
		if cp.currentSize < cp.maxSize {
			// Create new connection (placeholder)
			conn := fmt.Sprintf("connection_%d", cp.created)
			cp.connections = append(cp.connections, conn)
			cp.currentSize++
			cp.created++
			cp.borrowed++
			return conn, nil
		}
		
		return nil, fmt.Errorf("connection pool exhausted")
	}
}

// Put returns a connection to the pool
func (cp *ConnectionPool) Put(conn any) {
	select {
	case cp.available <- conn:
		cp.returned++
	default:
		// Pool is full, discard connection
	}
}

// GetStats returns connection pool statistics
func (cp *ConnectionPool) GetStats() map[string]any {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	return map[string]any{
		"max_size":     cp.maxSize,
		"current_size": cp.currentSize,
		"available":    len(cp.available),
		"created":      cp.created,
		"borrowed":     cp.borrowed,
		"returned":     cp.returned,
	}
}

// GetPerformanceReport generates a comprehensive performance report
func (po *PerformanceOptimizer) GetPerformanceReport() map[string]any {
	report := make(map[string]any)
	
	// Basic stats
	report["enabled"] = po.enabled
	report["last_optimization"] = po.lastOptimization
	report["optimization_count"] = po.optimizationCount
	
	// Metrics summary
	po.metricsMutex.RLock()
	healthyMetrics := 0
	totalMetrics := len(po.metrics)
	for _, metric := range po.metrics {
		if metric.IsHealthy {
			healthyMetrics++
		}
	}
	po.metricsMutex.RUnlock()
	
	report["metrics_health_rate"] = 0.0
	if totalMetrics > 0 {
		report["metrics_health_rate"] = float64(healthyMetrics) / float64(totalMetrics)
	}
	
	// Cache performance
	report["cache_stats"] = po.GetCacheStats()
	
	// Active rules
	po.rulesMutex.RLock()
	activeRules := 0
	for _, rule := range po.rules {
		if rule.Enabled && rule.IsActive {
			activeRules++
		}
	}
	po.rulesMutex.RUnlock()
	
	report["active_rules"] = activeRules
	report["total_rules"] = len(po.rules)
	
	return report
}