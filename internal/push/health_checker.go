// Package push provides health checking for CLI endpoints
package push

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HealthChecker monitors CLI endpoint health and updates their status
type HealthChecker struct {
	registry       *Registry
	httpClient     *http.Client
	checkInterval  time.Duration
	timeout        time.Duration
	maxConcurrency int
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	running        bool
	mu             sync.RWMutex
	metrics        *HealthMetrics
}

// HealthMetrics tracks health checking performance
type HealthMetrics struct {
	TotalChecks        int64         `json:"total_checks"`
	SuccessfulChecks   int64         `json:"successful_checks"`
	FailedChecks       int64         `json:"failed_checks"`
	AverageCheckTime   time.Duration `json:"average_check_time"`
	LastCheckCycle     time.Time     `json:"last_check_cycle"`
	EndpointsChecked   int           `json:"endpoints_checked"`
	HealthyEndpoints   int           `json:"healthy_endpoints"`
	UnhealthyEndpoints int           `json:"unhealthy_endpoints"`
	ErrorRate          float64       `json:"error_rate"`
	mu                 sync.RWMutex
}

// HealthCheckConfig configures the health checker
type HealthCheckConfig struct {
	CheckInterval  time.Duration `json:"check_interval"`
	Timeout        time.Duration `json:"timeout"`
	MaxConcurrency int           `json:"max_concurrency"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// DefaultHealthCheckConfig returns default health check configuration
func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		CheckInterval:  30 * time.Second,
		Timeout:        5 * time.Second,
		MaxConcurrency: 10,
		RetryAttempts:  2,
		RetryDelay:     time.Second,
	}
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	EndpointID    string        `json:"endpoint_id"`
	URL           string        `json:"url"`
	Success       bool          `json:"success"`
	StatusCode    int           `json:"status_code"`
	ResponseTime  time.Duration `json:"response_time"`
	Error         string        `json:"error"`
	Timestamp     time.Time     `json:"timestamp"`
	Headers       http.Header   `json:"headers"`
	ContentLength int64         `json:"content_length"`
}

// NewHealthChecker creates a new CLI endpoint health checker
func NewHealthChecker(registry *Registry, config *HealthCheckConfig) *HealthChecker {
	if config == nil {
		config = DefaultHealthCheckConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &HealthChecker{
		registry:       registry,
		httpClient:     &http.Client{Timeout: config.Timeout},
		checkInterval:  config.CheckInterval,
		timeout:        config.Timeout,
		maxConcurrency: config.MaxConcurrency,
		ctx:            ctx,
		cancel:         cancel,
		running:        false,
		metrics:        &HealthMetrics{},
	}
}

// Start starts the health checker
func (hc *HealthChecker) Start() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if hc.running {
		return errors.New("health checker already running")
	}

	log.Printf("Starting CLI endpoint health checker (interval: %v)", hc.checkInterval)

	hc.wg.Add(1)
	go hc.healthCheckLoop()

	hc.running = true
	return nil
}

// Stop stops the health checker gracefully
func (hc *HealthChecker) Stop() error {
	hc.mu.Lock()
	if !hc.running {
		hc.mu.Unlock()
		return errors.New("health checker not running")
	}
	hc.running = false
	hc.mu.Unlock()

	log.Println("Stopping CLI endpoint health checker...")

	// Cancel context to signal health check loop to stop
	hc.cancel()

	// Wait for health check loop to finish
	hc.wg.Wait()

	log.Println("CLI endpoint health checker stopped")
	return nil
}

// IsRunning returns whether the health checker is running
func (hc *HealthChecker) IsRunning() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.running
}

// healthCheckLoop runs the periodic health check cycle
func (hc *HealthChecker) healthCheckLoop() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	// Perform initial health check
	hc.performHealthCheckCycle()

	for {
		select {
		case <-ticker.C:
			hc.performHealthCheckCycle()

		case <-hc.ctx.Done():
			log.Println("Health check loop stopped")
			return
		}
	}
}

// performHealthCheckCycle performs a complete health check cycle for all endpoints
func (hc *HealthChecker) performHealthCheckCycle() {
	startTime := time.Now()
	endpoints := hc.registry.GetAll()

	if len(endpoints) == 0 {
		return
	}

	log.Printf("Starting health check cycle for %d endpoints", len(endpoints))

	// Use semaphore to limit concurrency
	semaphore := make(chan struct{}, hc.maxConcurrency)
	var wg sync.WaitGroup
	results := make(chan *HealthCheckResult, len(endpoints))

	// Start health checks for all endpoints
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(ep *CLIEndpoint) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := hc.checkEndpointHealth(ep)
			results <- result
		}(endpoint)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results
	healthyCount := 0
	unhealthyCount := 0
	for result := range results {
		hc.processHealthCheckResult(result)
		if result.Success {
			healthyCount++
		} else {
			unhealthyCount++
		}
	}

	// Update metrics
	hc.updateCycleMetrics(len(endpoints), healthyCount, unhealthyCount, time.Since(startTime))

	log.Printf("Health check cycle completed: %d healthy, %d unhealthy (took %v)",
		healthyCount, unhealthyCount, time.Since(startTime))
}

// checkEndpointHealth performs a health check for a specific endpoint
func (hc *HealthChecker) checkEndpointHealth(endpoint *CLIEndpoint) *HealthCheckResult {
	startTime := time.Now()

	result := &HealthCheckResult{
		EndpointID: endpoint.ID,
		URL:        endpoint.URL,
		Timestamp:  startTime,
	}

	// Determine health check URL
	healthURL := hc.getHealthCheckURL(endpoint)

	// Create request with context for timeout
	req, err := http.NewRequestWithContext(hc.ctx, "GET", healthURL, http.NoBody)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		result.ResponseTime = time.Since(startTime)
		return result
	}

	// Set headers
	req.Header.Set("User-Agent", "MCP-Memory-Health-Checker/1.0")
	req.Header.Set("X-Health-Check", "true")

	// Perform request
	resp, err := hc.httpClient.Do(req)
	result.ResponseTime = time.Since(startTime)

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP request failed: %v", err)
		return result
	}
	defer func() {
		// Ignore close errors in defer for health checks
		_ = resp.Body.Close()
	}()

	result.StatusCode = resp.StatusCode
	result.Headers = resp.Header
	result.ContentLength = resp.ContentLength

	// Determine if endpoint is healthy based on status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
	} else {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

// getHealthCheckURL determines the health check URL for an endpoint
func (hc *HealthChecker) getHealthCheckURL(endpoint *CLIEndpoint) string {
	baseURL := strings.TrimSuffix(endpoint.URL, "/")

	// Check if endpoint has a custom health check path in metadata
	if healthPath, exists := endpoint.Metadata["health_check_path"]; exists {
		return baseURL + "/" + strings.TrimPrefix(healthPath, "/")
	}

	// Try common health check paths
	healthPaths := []string{"/health", "/healthz", "/ping", "/status"}

	// Use the first path or default to /health
	for _, path := range healthPaths {
		if hc.hasCapability(endpoint, "health_check_"+strings.TrimPrefix(path, "/")) {
			return baseURL + path
		}
	}

	// Default to /health
	return baseURL + "/health"
}

// hasCapability checks if an endpoint has a specific capability
func (hc *HealthChecker) hasCapability(endpoint *CLIEndpoint, capability string) bool {
	for _, cap := range endpoint.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// processHealthCheckResult processes a health check result and updates endpoint status
func (hc *HealthChecker) processHealthCheckResult(result *HealthCheckResult) {
	// Update metrics
	hc.updateMetrics(result)

	// Get current endpoint
	endpoint, exists := hc.registry.Get(result.EndpointID)
	if !exists {
		log.Printf("Endpoint %s not found in registry", result.EndpointID)
		return
	}

	// Calculate new health metrics
	newHealth := &EndpointHealth{
		LastHealthCheck: result.Timestamp,
		LastError:       result.Error,
	}

	if result.Success {
		// Endpoint is healthy
		newHealth.IsHealthy = true
		newHealth.ConsecutiveFailures = 0
		newHealth.SuccessfulRequests = endpoint.Health.SuccessfulRequests + 1
		newHealth.TotalRequests = endpoint.Health.TotalRequests + 1

		// Update average response time
		if endpoint.Health.AverageResponseTime == 0 {
			newHealth.AverageResponseTime = result.ResponseTime
		} else {
			// Weighted average with 80% weight on previous average
			newHealth.AverageResponseTime = time.Duration(
				int64(endpoint.Health.AverageResponseTime)*8/10 + int64(result.ResponseTime)*2/10,
			)
		}
	} else {
		// Endpoint is unhealthy
		newHealth.IsHealthy = false
		newHealth.ConsecutiveFailures = endpoint.Health.ConsecutiveFailures + 1
		newHealth.TotalRequests = endpoint.Health.TotalRequests + 1
		newHealth.SuccessfulRequests = endpoint.Health.SuccessfulRequests
		newHealth.AverageResponseTime = endpoint.Health.AverageResponseTime
	}

	// Calculate success rate
	if newHealth.TotalRequests > 0 {
		newHealth.SuccessRate = float64(newHealth.SuccessfulRequests) / float64(newHealth.TotalRequests)
	}

	// Update endpoint health in registry
	if err := hc.registry.UpdateHealth(result.EndpointID, newHealth); err != nil {
		log.Printf("Failed to update health for endpoint %s: %v", result.EndpointID, err)
	}

	// Log health status changes
	if result.Success != endpoint.Health.IsHealthy {
		if result.Success {
			log.Printf("Endpoint %s (%s) is now healthy (response time: %v)",
				result.EndpointID, result.URL, result.ResponseTime)
		} else {
			log.Printf("Endpoint %s (%s) is now unhealthy: %s (consecutive failures: %d)",
				result.EndpointID, result.URL, result.Error, newHealth.ConsecutiveFailures)
		}
	}
}

// updateMetrics updates health check metrics
func (hc *HealthChecker) updateMetrics(result *HealthCheckResult) {
	hc.metrics.mu.Lock()
	defer hc.metrics.mu.Unlock()

	hc.metrics.TotalChecks++

	if result.Success {
		hc.metrics.SuccessfulChecks++
	} else {
		hc.metrics.FailedChecks++
	}

	// Update average check time
	if hc.metrics.AverageCheckTime == 0 {
		hc.metrics.AverageCheckTime = result.ResponseTime
	} else {
		// Weighted average with 90% weight on previous average
		hc.metrics.AverageCheckTime = time.Duration(
			int64(hc.metrics.AverageCheckTime)*9/10 + int64(result.ResponseTime)/10,
		)
	}

	// Calculate error rate
	if hc.metrics.TotalChecks > 0 {
		hc.metrics.ErrorRate = float64(hc.metrics.FailedChecks) / float64(hc.metrics.TotalChecks) * 100
	}
}

// updateCycleMetrics updates metrics after a complete health check cycle
func (hc *HealthChecker) updateCycleMetrics(total, healthy, unhealthy int, duration time.Duration) {
	_ = duration // unused parameter, kept for potential future duration-based metrics
	hc.metrics.mu.Lock()
	defer hc.metrics.mu.Unlock()

	hc.metrics.LastCheckCycle = time.Now()
	hc.metrics.EndpointsChecked = total
	hc.metrics.HealthyEndpoints = healthy
	hc.metrics.UnhealthyEndpoints = unhealthy
}

// CheckEndpoint performs an immediate health check for a specific endpoint
func (hc *HealthChecker) CheckEndpoint(endpointID string) (*HealthCheckResult, error) {
	endpoint, exists := hc.registry.Get(endpointID)
	if !exists {
		return nil, fmt.Errorf("endpoint not found: %s", endpointID)
	}

	result := hc.checkEndpointHealth(endpoint)
	hc.processHealthCheckResult(result)

	return result, nil
}

// GetMetrics returns health check metrics
func (hc *HealthChecker) GetMetrics() *HealthMetrics {
	hc.metrics.mu.RLock()
	defer hc.metrics.mu.RUnlock()

	// Return a copy
	return &HealthMetrics{
		TotalChecks:        hc.metrics.TotalChecks,
		SuccessfulChecks:   hc.metrics.SuccessfulChecks,
		FailedChecks:       hc.metrics.FailedChecks,
		AverageCheckTime:   hc.metrics.AverageCheckTime,
		LastCheckCycle:     hc.metrics.LastCheckCycle,
		EndpointsChecked:   hc.metrics.EndpointsChecked,
		HealthyEndpoints:   hc.metrics.HealthyEndpoints,
		UnhealthyEndpoints: hc.metrics.UnhealthyEndpoints,
		ErrorRate:          hc.metrics.ErrorRate,
	}
}

// GetHealthSummary returns a summary of endpoint health status
func (hc *HealthChecker) GetHealthSummary() map[string]interface{} {
	metrics := hc.GetMetrics()

	return map[string]interface{}{
		"running":             hc.IsRunning(),
		"check_interval":      hc.checkInterval.String(),
		"total_checks":        metrics.TotalChecks,
		"successful_checks":   metrics.SuccessfulChecks,
		"failed_checks":       metrics.FailedChecks,
		"error_rate":          metrics.ErrorRate,
		"average_check_time":  metrics.AverageCheckTime.String(),
		"last_check_cycle":    metrics.LastCheckCycle,
		"endpoints_checked":   metrics.EndpointsChecked,
		"healthy_endpoints":   metrics.HealthyEndpoints,
		"unhealthy_endpoints": metrics.UnhealthyEndpoints,
	}
}

// ForceHealthCheck triggers an immediate health check cycle
func (hc *HealthChecker) ForceHealthCheck() {
	if !hc.IsRunning() {
		return
	}

	go hc.performHealthCheckCycle()
}
