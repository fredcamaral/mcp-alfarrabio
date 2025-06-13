package ai

import (
	"sync"
	"time"
)

// Metrics tracks AI service performance metrics
type Metrics struct {
	mu       sync.RWMutex
	requests map[Model]*modelMetrics
}

type modelMetrics struct {
	totalRequests   int64
	totalErrors     int64
	totalTokens     int64
	totalLatency    time.Duration
	cacheHits       int64
	cacheMisses     int64
	lastRequestTime time.Time
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		requests: make(map[Model]*modelMetrics),
	}
}

// RecordRequest records a successful request
func (m *Metrics) RecordRequest(model Model, latency time.Duration, tokens TokenUsage) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.requests[model]; !exists {
		m.requests[model] = &modelMetrics{}
	}

	metrics := m.requests[model]
	metrics.totalRequests++
	metrics.totalTokens += int64(tokens.TotalTokens)
	metrics.totalLatency += latency
	metrics.lastRequestTime = time.Now()
}

// RecordError records a failed request
func (m *Metrics) RecordError(model Model, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.requests[model]; !exists {
		m.requests[model] = &modelMetrics{}
	}

	m.requests[model].totalErrors++
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(model Model) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.requests[model]; !exists {
		m.requests[model] = &modelMetrics{}
	}

	m.requests[model].cacheHits++
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(model Model) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.requests[model]; !exists {
		m.requests[model] = &modelMetrics{}
	}

	m.requests[model].cacheMisses++
}

// GetMetrics returns a copy of current metrics
func (m *Metrics) GetMetrics() map[Model]ModelStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[Model]ModelStats)
	for model, metrics := range m.requests {
		avgLatency := time.Duration(0)
		if metrics.totalRequests > 0 {
			avgLatency = metrics.totalLatency / time.Duration(metrics.totalRequests)
		}

		stats[model] = ModelStats{
			TotalRequests:   metrics.totalRequests,
			TotalErrors:     metrics.totalErrors,
			TotalTokens:     metrics.totalTokens,
			AverageLatency:  avgLatency,
			CacheHitRate:    calculateHitRate(metrics.cacheHits, metrics.cacheMisses),
			LastRequestTime: metrics.lastRequestTime,
		}
	}

	return stats
}

// ModelStats represents statistics for a single model
type ModelStats struct {
	TotalRequests   int64         `json:"total_requests"`
	TotalErrors     int64         `json:"total_errors"`
	TotalTokens     int64         `json:"total_tokens"`
	AverageLatency  time.Duration `json:"average_latency"`
	CacheHitRate    float64       `json:"cache_hit_rate"`
	LastRequestTime time.Time     `json:"last_request_time"`
}

func calculateHitRate(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total)
}
