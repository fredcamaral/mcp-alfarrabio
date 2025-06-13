// Package performance provides performance optimization capabilities
package performance

import (
	"time"
)

// PerformanceOptimizer provides optimization services across the system
type PerformanceOptimizer struct {
	enabled      bool
	vectorCache  *Cache
	patternCache *Cache
	queryCache   *Cache
}

// NewPerformanceOptimizer creates a new performance optimizer with default configuration
func NewPerformanceOptimizer() *PerformanceOptimizer {
	// Create default cache configs for different cache types
	vectorConfig := DefaultCacheConfig()
	vectorConfig.MaxItems = 5000

	patternConfig := DefaultCacheConfig()
	patternConfig.MaxItems = 1000
	patternConfig.DefaultTTL = 30 * time.Minute

	queryConfig := DefaultCacheConfig()
	queryConfig.MaxItems = 2000
	queryConfig.DefaultTTL = 10 * time.Minute

	return &PerformanceOptimizer{
		enabled:      true,
		vectorCache:  NewCache(vectorConfig),
		patternCache: NewCache(patternConfig),
		queryCache:   NewCache(queryConfig),
	}
}

// GetStats returns performance statistics for the optimizer
func (p *PerformanceOptimizer) GetStats() *OptimizerStats {
	return &OptimizerStats{
		Enabled:          p.enabled,
		VectorCacheHits:  p.vectorCache.stats.Hits,
		PatternCacheHits: p.patternCache.stats.Hits,
		QueryCacheHits:   p.queryCache.stats.Hits,
	}
}

// OptimizerStats provides statistics for the performance optimizer
type OptimizerStats struct {
	Enabled          bool  `json:"enabled"`
	VectorCacheHits  int64 `json:"vector_cache_hits"`
	PatternCacheHits int64 `json:"pattern_cache_hits"`
	QueryCacheHits   int64 `json:"query_cache_hits"`
}
