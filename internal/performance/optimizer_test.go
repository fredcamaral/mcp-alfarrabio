package performance

import (
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
	config := &CacheConfig{
		MaxItems:       3,
		DefaultTTL:     1 * time.Hour, // Use longer TTL for test stability
		MaxTTL:         2 * time.Hour,
		EvictionPolicy: "lru",
		TrackStats:     true,
	}

	cache := NewCache(config)

	// Test set and get
	if err := cache.Set("key1", "value1"); err != nil {
		t.Fatalf("Failed to set cache value: %v", err)
	}
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
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %v", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %v", stats.Misses)
	}
}

func TestOptimizerStats(t *testing.T) {
	optimizer := NewPerformanceOptimizer()
	stats := optimizer.GetStats()

	if !stats.Enabled {
		t.Error("Expected optimizer to be enabled")
	}

	// Test that stats are initialized to zero
	if stats.VectorCacheHits != 0 {
		t.Errorf("Expected 0 vector cache hits, got %v", stats.VectorCacheHits)
	}
}
