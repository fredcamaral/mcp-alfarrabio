package performance

import (
	"context"
	"testing"
	"time"
)

func TestPerformanceOptimizerCreation(t *testing.T) {
	optimizer := NewPerformanceOptimizer()

	if optimizer == nil {
		t.Fatal("Expected performance optimizer to be created")
	}

	if !optimizer.enabled {
		t.Error("Expected optimizer to be enabled by default")
	}

	if optimizer.vectorCache == nil {
		t.Error("Expected vector cache to be initialized")
	}

	if optimizer.patternCache == nil {
		t.Error("Expected pattern cache to be initialized")
	}

	if optimizer.queryCache == nil {
		t.Error("Expected query cache to be initialized")
	}
}

func TestCacheOperations(t *testing.T) {
	config := CacheConfig{
		MaxSize:        3,
		TTL:            1 * time.Second,
		EvictionPolicy: "lru",
		Enabled:        true,
	}

	cache := NewCache(config)

	// Test set and get
	cache.Set("key1", "value1")
	value, exists := cache.Get("key1")
	if !exists {
		t.Error("Expected key1 to exist in cache")
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// Test cache miss
	_, exists = cache.Get("nonexistent")
	if exists {
		t.Error("Expected cache miss for nonexistent key")
	}

	// Test cache statistics
	stats := cache.GetStats()
	if stats["hits"].(int64) != 1 {
		t.Errorf("Expected 1 hit, got %v", stats["hits"])
	}
	if stats["misses"].(int64) != 1 {
		t.Errorf("Expected 1 miss, got %v", stats["misses"])
	}
}

func TestCacheEviction(t *testing.T) {
	config := CacheConfig{
		MaxSize:        2,
		TTL:            1 * time.Hour, // Long TTL for this test
		EvictionPolicy: "lru",
		Enabled:        true,
	}

	cache := NewCache(config)

	// Fill cache to capacity
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	// Access key1 to make it more recently used
	cache.Get("key1")

	// Add key3, should evict key2 (least recently used)
	cache.Set("key3", "value3")

	// key2 should be evicted
	_, exists := cache.Get("key2")
	if exists {
		t.Error("Expected key2 to be evicted")
	}

	// key1 and key3 should still exist
	_, exists = cache.Get("key1")
	if !exists {
		t.Error("Expected key1 to still exist")
	}

	_, exists = cache.Get("key3")
	if !exists {
		t.Error("Expected key3 to exist")
	}
}

func TestCacheTTL(t *testing.T) {
	config := CacheConfig{
		MaxSize:        10,
		TTL:            100 * time.Millisecond,
		EvictionPolicy: "lru",
		Enabled:        true,
	}

	cache := NewCache(config)

	// Set a value
	cache.Set("ttl_key", "ttl_value")

	// Should exist immediately
	value, exists := cache.Get("ttl_key")
	if !exists || value != "ttl_value" {
		t.Error("Expected key to exist immediately after setting")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should be expired now
	_, exists = cache.Get("ttl_key")
	if exists {
		t.Error("Expected key to be expired after TTL")
	}
}

func TestMetricsRecording(t *testing.T) {
	optimizer := NewPerformanceOptimizer()

	metric := PerformanceMetric{
		Name:      "test_latency",
		Type:      MetricTypeLatency,
		Value:     150.0,
		Unit:      "ms",
		Threshold: 200.0,
	}

	optimizer.RecordMetric(&metric)

	metrics := optimizer.GetMetrics()

	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	recorded, exists := metrics["test_latency"]
	if !exists {
		t.Error("Expected test_latency metric to exist")
	}

	if recorded.Value != 150.0 {
		t.Errorf("Expected value 150.0, got %f", recorded.Value)
	}

	if !recorded.IsHealthy {
		t.Error("Expected metric to be healthy (value <= threshold)")
	}
}

func TestOptimizationRules(t *testing.T) {
	optimizer := NewPerformanceOptimizer()

	// Add a test rule
	rule := OptimizationRule{
		ID:         "test_rule",
		Name:       "Test Optimization Rule",
		Condition:  "high_latency",
		Action:     "clear_cache",
		Parameters: map[string]any{"threshold": 100.0},
		Priority:   5,
	}

	optimizer.AddOptimizationRule(&rule)

	// Add a metric that should trigger the rule
	metric := PerformanceMetric{
		Name:      "query_latency",
		Type:      MetricTypeLatency,
		Value:     250.0,
		Unit:      "ms",
		Threshold: 200.0,
	}

	optimizer.RecordMetric(&metric)

	// Apply optimizations
	err := optimizer.ApplyOptimizations(context.Background())
	if err != nil {
		t.Fatalf("Expected no error applying optimizations, got %v", err)
	}

	// Check that optimization was applied (cache should be cleared)
	stats := optimizer.vectorCache.GetStats()
	if stats["size"].(int) != 0 {
		t.Error("Expected cache to be cleared after optimization")
	}
}

func TestBatchProcessor(t *testing.T) {
	processed := make([]BatchItem, 0)

	processor := func(items []BatchItem) error {
		processed = append(processed, items...)
		return nil
	}

	bp := NewBatchProcessor(3, 100*time.Millisecond, processor)

	// Add items one by one
	item1 := BatchItem{ID: "1", Type: "test", Data: "data1"}
	item2 := BatchItem{ID: "2", Type: "test", Data: "data2"}

	err := bp.Add(item1)
	if err != nil {
		t.Fatalf("Expected no error adding item1, got %v", err)
	}

	err = bp.Add(item2)
	if err != nil {
		t.Fatalf("Expected no error adding item2, got %v", err)
	}

	// Should not be processed yet (batch size is 3)
	if len(processed) != 0 {
		t.Errorf("Expected 0 processed items, got %d", len(processed))
	}

	// Add third item to trigger batch processing
	item3 := BatchItem{ID: "3", Type: "test", Data: "data3"}
	err = bp.Add(item3)
	if err != nil {
		t.Fatalf("Expected no error adding item3, got %v", err)
	}

	// Now should be processed
	if len(processed) != 3 {
		t.Errorf("Expected 3 processed items, got %d", len(processed))
	}
}

func TestConnectionPool(t *testing.T) {
	factory := func() (any, error) {
		return "test_connection", nil
	}

	pool := NewConnectionPool(2, factory)

	// Get first connection
	conn1, err := pool.Get()
	if err != nil {
		t.Fatalf("Expected no error getting connection, got %v", err)
	}

	if conn1 == nil {
		t.Error("Expected connection to be returned")
	}

	// Get second connection
	_, err = pool.Get()
	if err != nil {
		t.Fatalf("Expected no error getting second connection, got %v", err)
	}

	// Try to get third connection (should fail - pool exhausted)
	_, err = pool.Get()
	if err == nil {
		t.Error("Expected error when pool is exhausted")
	}

	// Return a connection
	pool.Put(conn1)

	// Should be able to get a connection again
	conn3, err := pool.Get()
	if err != nil {
		t.Fatalf("Expected no error after returning connection, got %v", err)
	}

	if conn3 != conn1 {
		t.Error("Expected to get the same connection back")
	}

	// Check stats
	stats := pool.GetStats()
	if stats["max_size"].(int) != 2 {
		t.Errorf("Expected max_size 2, got %v", stats["max_size"])
	}

	if stats["borrowed"].(int64) != 3 {
		t.Errorf("Expected 3 borrows, got %v", stats["borrowed"])
	}
}

func TestPerformanceReport(t *testing.T) {
	optimizer := NewPerformanceOptimizer()

	// Add some test data
	metric := PerformanceMetric{
		Name:      "test_metric",
		Type:      MetricTypeLatency,
		Value:     100.0,
		Threshold: 200.0,
	}
	optimizer.RecordMetric(&metric)

	rule := OptimizationRule{
		ID:       "test_rule",
		Name:     "Test Rule",
		Enabled:  true,
		IsActive: true,
	}
	optimizer.AddOptimizationRule(&rule)

	// Generate report
	report := optimizer.GetPerformanceReport()

	// Check basic fields
	if !report["enabled"].(bool) {
		t.Error("Expected optimizer to be enabled in report")
	}

	if report["metrics_health_rate"].(float64) != 1.0 {
		t.Errorf("Expected 100%% healthy metrics, got %f", report["metrics_health_rate"])
	}

	if report["active_rules"].(int) != 1 {
		t.Errorf("Expected 1 active rule, got %v", report["active_rules"])
	}

	if report["total_rules"].(int) != 1 {
		t.Errorf("Expected 1 total rule, got %v", report["total_rules"])
	}

	// Check cache stats are included
	cacheStats, ok := report["cache_stats"].(map[string]any)
	if !ok {
		t.Error("Expected cache_stats in report")
	}

	if len(cacheStats) != 3 {
		t.Errorf("Expected 3 cache types in stats, got %d", len(cacheStats))
	}
}

func TestCacheDisabled(t *testing.T) {
	config := CacheConfig{
		MaxSize:        10,
		TTL:            1 * time.Hour,
		EvictionPolicy: "lru",
		Enabled:        false, // Disabled
	}

	cache := NewCache(config)

	// Set should do nothing when disabled
	cache.Set("key1", "value1")

	// Get should return false when disabled
	_, exists := cache.Get("key1")
	if exists {
		t.Error("Expected cache miss when cache is disabled")
	}
}

func TestCacheEvictionPolicies(t *testing.T) {
	// Test LFU eviction
	config := CacheConfig{
		MaxSize:        2,
		TTL:            1 * time.Hour,
		EvictionPolicy: "lfu",
		Enabled:        true,
	}

	cache := NewCache(config)

	// Add two items
	cache.Set("freq1", "value1")
	cache.Set("freq2", "value2")

	// Access freq1 multiple times to make it more frequent
	cache.Get("freq1")
	cache.Get("freq1")
	cache.Get("freq1")

	// Access freq2 only once
	cache.Get("freq2")

	// Add third item - should evict freq2 (less frequent)
	cache.Set("freq3", "value3")

	// freq2 should be evicted
	_, exists := cache.Get("freq2")
	if exists {
		t.Error("Expected freq2 to be evicted (LFU policy)")
	}

	// freq1 should still exist
	_, exists = cache.Get("freq1")
	if !exists {
		t.Error("Expected freq1 to still exist (more frequent)")
	}
}
