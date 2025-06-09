// Package ai provides metrics collection for AI service performance monitoring.
package ai

import (
	"sync"
	"time"
)

// ModelMetrics tracks performance metrics for a specific AI model
type ModelMetrics struct {
	RequestCount    int64         `json:"request_count"`
	SuccessCount    int64         `json:"success_count"`
	ErrorCount      int64         `json:"error_count"`
	TotalLatency    time.Duration `json:"total_latency"`
	AverageLatency  time.Duration `json:"average_latency"`
	MinLatency      time.Duration `json:"min_latency"`
	MaxLatency      time.Duration `json:"max_latency"`
	TotalTokens     int64         `json:"total_tokens"`
	InputTokens     int64         `json:"input_tokens"`
	OutputTokens    int64         `json:"output_tokens"`
	CacheHits       int64         `json:"cache_hits"`
	CacheMisses     int64         `json:"cache_misses"`
	LastRequestTime time.Time     `json:"last_request_time"`
	StartTime       time.Time     `json:"start_time"`
}

// ErrorMetrics tracks error information
type ErrorMetrics struct {
	Type        string    `json:"type"`
	Count       int64     `json:"count"`
	LastOccured time.Time `json:"last_occurred"`
	Message     string    `json:"message"`
}

// Metrics provides comprehensive AI service metrics collection
type Metrics struct {
	models     map[Model]*ModelMetrics
	errors     map[string]*ErrorMetrics
	mutex      sync.RWMutex
	startTime  time.Time
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		models:    make(map[Model]*ModelMetrics),
		errors:    make(map[string]*ErrorMetrics),
		startTime: time.Now(),
	}
}

// RecordRequest records a successful AI request
func (m *Metrics) RecordRequest(model Model, latency time.Duration, tokens TokenUsage) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	modelMetrics := m.getOrCreateModelMetrics(model)
	
	modelMetrics.RequestCount++
	modelMetrics.SuccessCount++
	modelMetrics.TotalLatency += latency
	modelMetrics.AverageLatency = time.Duration(int64(modelMetrics.TotalLatency) / modelMetrics.RequestCount)
	
	// Update min/max latency
	if modelMetrics.MinLatency == 0 || latency < modelMetrics.MinLatency {
		modelMetrics.MinLatency = latency
	}
	if latency > modelMetrics.MaxLatency {
		modelMetrics.MaxLatency = latency
	}
	
	// Update token counts
	modelMetrics.TotalTokens += int64(tokens.Total)
	modelMetrics.InputTokens += int64(tokens.Input)
	modelMetrics.OutputTokens += int64(tokens.Output)
	
	modelMetrics.LastRequestTime = time.Now()
}

// RecordError records an AI request error
func (m *Metrics) RecordError(model Model, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	modelMetrics := m.getOrCreateModelMetrics(model)
	modelMetrics.RequestCount++
	modelMetrics.ErrorCount++
	modelMetrics.LastRequestTime = time.Now()

	// Track error details
	errorType := getErrorType(err)
	errorMetrics := m.getOrCreateErrorMetrics(errorType)
	errorMetrics.Count++
	errorMetrics.LastOccured = time.Now()
	errorMetrics.Message = err.Error()
}

// RecordCacheHit records a cache hit for a model
func (m *Metrics) RecordCacheHit(model Model) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	modelMetrics := m.getOrCreateModelMetrics(model)
	modelMetrics.CacheHits++
}

// RecordCacheMiss records a cache miss for a model
func (m *Metrics) RecordCacheMiss(model Model) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	modelMetrics := m.getOrCreateModelMetrics(model)
	modelMetrics.CacheMisses++
}

// getOrCreateModelMetrics gets or creates metrics for a model
func (m *Metrics) getOrCreateModelMetrics(model Model) *ModelMetrics {
	if metrics, exists := m.models[model]; exists {
		return metrics
	}

	metrics := &ModelMetrics{
		StartTime: time.Now(),
	}
	m.models[model] = metrics
	return metrics
}

// getOrCreateErrorMetrics gets or creates metrics for an error type
func (m *Metrics) getOrCreateErrorMetrics(errorType string) *ErrorMetrics {
	if metrics, exists := m.errors[errorType]; exists {
		return metrics
	}

	metrics := &ErrorMetrics{
		Type: errorType,
	}
	m.errors[errorType] = metrics
	return metrics
}

// GetModelMetrics returns metrics for a specific model
func (m *Metrics) GetModelMetrics(model Model) *ModelMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if metrics, exists := m.models[model]; exists {
		// Return a copy to prevent data races
		metricsCopy := *metrics
		return &metricsCopy
	}
	return nil
}

// GetAllModelMetrics returns metrics for all models
func (m *Metrics) GetAllModelMetrics() map[Model]*ModelMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[Model]*ModelMetrics)
	for model, metrics := range m.models {
		metricsCopy := *metrics
		result[model] = &metricsCopy
	}
	return result
}

// GetErrorMetrics returns all error metrics
func (m *Metrics) GetErrorMetrics() map[string]*ErrorMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*ErrorMetrics)
	for errorType, metrics := range m.errors {
		metricsCopy := *metrics
		result[errorType] = &metricsCopy
	}
	return result
}

// GetSuccessRate calculates success rate for a model
func (m *Metrics) GetSuccessRate(model Model) float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if metrics, exists := m.models[model]; exists {
		if metrics.RequestCount == 0 {
			return 0.0
		}
		return float64(metrics.SuccessCount) / float64(metrics.RequestCount)
	}
	return 0.0
}

// GetCacheHitRate calculates cache hit rate for a model
func (m *Metrics) GetCacheHitRate(model Model) float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if metrics, exists := m.models[model]; exists {
		total := metrics.CacheHits + metrics.CacheMisses
		if total == 0 {
			return 0.0
		}
		return float64(metrics.CacheHits) / float64(total)
	}
	return 0.0
}

// GetOverallStats returns overall service statistics
func (m *Metrics) GetOverallStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var totalRequests, totalSuccess, totalErrors int64
	var totalLatency time.Duration
	var totalTokens int64

	for _, metrics := range m.models {
		totalRequests += metrics.RequestCount
		totalSuccess += metrics.SuccessCount
		totalErrors += metrics.ErrorCount
		totalLatency += metrics.TotalLatency
		totalTokens += metrics.TotalTokens
	}

	var averageLatency time.Duration
	if totalRequests > 0 {
		averageLatency = time.Duration(int64(totalLatency) / totalRequests)
	}

	var successRate float64
	if totalRequests > 0 {
		successRate = float64(totalSuccess) / float64(totalRequests)
	}

	uptime := time.Since(m.startTime)

	return map[string]interface{}{
		"total_requests":    totalRequests,
		"total_success":     totalSuccess,
		"total_errors":      totalErrors,
		"success_rate":      successRate,
		"average_latency":   averageLatency,
		"total_tokens":      totalTokens,
		"uptime":           uptime,
		"models_count":     len(m.models),
		"error_types":      len(m.errors),
	}
}

// Reset clears all metrics
func (m *Metrics) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.models = make(map[Model]*ModelMetrics)
	m.errors = make(map[string]*ErrorMetrics)
	m.startTime = time.Now()
}

// getErrorType categorizes errors by type
func getErrorType(err error) string {
	if err == nil {
		return "unknown"
	}

	errMsg := err.Error()
	
	// Categorize common error types
	switch {
	case containsAny(errMsg, []string{"rate limit", "too many requests", "quota"}):
		return "rate_limit"
	case containsAny(errMsg, []string{"timeout", "deadline", "context canceled"}):
		return "timeout"
	case containsAny(errMsg, []string{"authentication", "unauthorized", "api key"}):
		return "auth"
	case containsAny(errMsg, []string{"network", "connection", "dns"}):
		return "network"
	case containsAny(errMsg, []string{"invalid", "malformed", "bad request"}):
		return "validation"
	case containsAny(errMsg, []string{"server error", "internal error", "503", "502", "500"}):
		return "server_error"
	default:
		return "other"
	}
}

// containsAny checks if a string contains any of the given substrings
func containsAny(str string, substrings []string) bool {
	for _, substr := range substrings {
		if len(str) >= len(substr) {
			for i := 0; i <= len(str)-len(substr); i++ {
				if str[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}