package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check represents a health check function
type Check func(ctx context.Context) *Result

// Result represents the result of a health check
type Result struct {
	Status      Status                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
	Duration    time.Duration          `json:"duration_ms"`
}

// HealthChecker manages health checks for the MCP server
type HealthChecker struct {
	mu           sync.RWMutex
	checks       map[string]Check
	results      map[string]*Result
	checkTimeout time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(checkTimeout time.Duration) *HealthChecker {
	if checkTimeout == 0 {
		checkTimeout = 10 * time.Second
	}

	return &HealthChecker{
		checks:       make(map[string]Check),
		results:      make(map[string]*Result),
		checkTimeout: checkTimeout,
	}
}

// RegisterCheck registers a health check
func (h *HealthChecker) RegisterCheck(name string, check Check) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = check
}

// UnregisterCheck removes a health check
func (h *HealthChecker) UnregisterCheck(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.checks, name)
	delete(h.results, name)
}

// CheckHealth runs all registered health checks
func (h *HealthChecker) CheckHealth(ctx context.Context) map[string]*Result {
	h.mu.RLock()
	checks := make(map[string]Check)
	for name, check := range h.checks {
		checks[name] = check
	}
	h.mu.RUnlock()

	results := make(map[string]*Result)
	var wg sync.WaitGroup

	for name, check := range checks {
		wg.Add(1)
		go func(n string, c Check) {
			defer wg.Done()
			
			checkCtx, cancel := context.WithTimeout(ctx, h.checkTimeout)
			defer cancel()

			start := time.Now()
			result := c(checkCtx)
			result.Duration = time.Since(start)
			result.LastChecked = time.Now()

			h.mu.Lock()
			h.results[n] = result
			h.mu.Unlock()

			results[n] = result
		}(name, check)
	}

	wg.Wait()
	return results
}

// GetOverallStatus returns the overall health status
func (h *HealthChecker) GetOverallStatus() Status {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.results) == 0 {
		return StatusHealthy
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range h.results {
		switch result.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}
	return StatusHealthy
}

// HTTPHandlerLive returns an HTTP handler for liveness checks
func (h *HealthChecker) HTTPHandlerLive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		response := map[string]interface{}{
			"status": "alive",
			"timestamp": time.Now().UTC(),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// HTTPHandlerReady returns an HTTP handler for readiness checks
func (h *HealthChecker) HTTPHandlerReady() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		results := h.CheckHealth(ctx)
		overallStatus := h.GetOverallStatus()

		w.Header().Set("Content-Type", "application/json")

		response := map[string]interface{}{
			"status":    overallStatus,
			"timestamp": time.Now().UTC(),
			"checks":    results,
		}

		statusCode := http.StatusOK
		if overallStatus == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}
}

// HTTPHandlerHealth returns an HTTP handler for detailed health checks
func (h *HealthChecker) HTTPHandlerHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		results := h.CheckHealth(ctx)
		overallStatus := h.GetOverallStatus()

		w.Header().Set("Content-Type", "application/json")

		// Get system info
		systemInfo := getSystemInfo()

		response := map[string]interface{}{
			"status":    overallStatus,
			"timestamp": time.Now().UTC(),
			"system":    systemInfo,
			"checks":    results,
		}

		statusCode := http.StatusOK
		switch overallStatus {
		case StatusDegraded:
			statusCode = http.StatusOK // Still return 200 for degraded
		case StatusUnhealthy:
			statusCode = http.StatusServiceUnavailable
		}

		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}
}

// Standard health checks

// CheckDatabase creates a database health check
func CheckDatabase(db interface{ Ping(context.Context) error }) Check {
	return func(ctx context.Context) *Result {
		start := time.Now()
		err := db.Ping(ctx)
		duration := time.Since(start)

		if err != nil {
			return &Result{
				Status:  StatusUnhealthy,
				Message: fmt.Sprintf("Database ping failed: %v", err),
				Details: map[string]interface{}{
					"error":    err.Error(),
					"duration": duration.Milliseconds(),
				},
			}
		}

		return &Result{
			Status:  StatusHealthy,
			Message: "Database is responsive",
			Details: map[string]interface{}{
				"duration": duration.Milliseconds(),
			},
		}
	}
}

// CheckHTTPEndpoint creates an HTTP endpoint health check
func CheckHTTPEndpoint(name, url string, timeout time.Duration) Check {
	return func(ctx context.Context) *Result {
		client := &http.Client{
			Timeout: timeout,
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return &Result{
				Status:  StatusUnhealthy,
				Message: fmt.Sprintf("Failed to create request: %v", err),
			}
		}

		start := time.Now()
		resp, err := client.Do(req)
		duration := time.Since(start)

		if err != nil {
			return &Result{
				Status:  StatusUnhealthy,
				Message: fmt.Sprintf("HTTP request failed: %v", err),
				Details: map[string]interface{}{
					"endpoint": url,
					"error":    err.Error(),
					"duration": duration.Milliseconds(),
				},
			}
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return &Result{
				Status:  StatusHealthy,
				Message: fmt.Sprintf("%s is healthy", name),
				Details: map[string]interface{}{
					"endpoint":    url,
					"status_code": resp.StatusCode,
					"duration":    duration.Milliseconds(),
				},
			}
		}

		return &Result{
			Status:  StatusDegraded,
			Message: fmt.Sprintf("%s returned non-2xx status", name),
			Details: map[string]interface{}{
				"endpoint":    url,
				"status_code": resp.StatusCode,
				"duration":    duration.Milliseconds(),
			},
		}
	}
}

// CheckDiskSpace creates a disk space health check
func CheckDiskSpace(path string, minFreeBytes uint64) Check {
	return func(ctx context.Context) *Result {
		usage, err := getDiskUsage(path)
		if err != nil {
			return &Result{
				Status:  StatusUnhealthy,
				Message: fmt.Sprintf("Failed to get disk usage: %v", err),
			}
		}

		freeBytes := usage.Total - usage.Used
		freePercent := float64(freeBytes) / float64(usage.Total) * 100

		details := map[string]interface{}{
			"path":         path,
			"total_bytes":  usage.Total,
			"used_bytes":   usage.Used,
			"free_bytes":   freeBytes,
			"free_percent": fmt.Sprintf("%.2f%%", freePercent),
		}

		if freeBytes < minFreeBytes {
			return &Result{
				Status:  StatusUnhealthy,
				Message: "Disk space critically low",
				Details: details,
			}
		}

		if freePercent < 20 {
			return &Result{
				Status:  StatusDegraded,
				Message: "Disk space low",
				Details: details,
			}
		}

		return &Result{
			Status:  StatusHealthy,
			Message: "Disk space adequate",
			Details: details,
		}
	}
}

// CheckMemory creates a memory usage health check
func CheckMemory(maxUsagePercent float64) Check {
	return func(ctx context.Context) *Result {
		memInfo, err := getMemoryInfo()
		if err != nil {
			return &Result{
				Status:  StatusUnhealthy,
				Message: fmt.Sprintf("Failed to get memory info: %v", err),
			}
		}

		usagePercent := memInfo.UsedPercent

		details := map[string]interface{}{
			"total_bytes":   memInfo.Total,
			"used_bytes":    memInfo.Used,
			"free_bytes":    memInfo.Free,
			"used_percent":  fmt.Sprintf("%.2f%%", usagePercent),
		}

		if usagePercent > maxUsagePercent {
			return &Result{
				Status:  StatusUnhealthy,
				Message: "Memory usage too high",
				Details: details,
			}
		}

		if usagePercent > maxUsagePercent*0.9 {
			return &Result{
				Status:  StatusDegraded,
				Message: "Memory usage elevated",
				Details: details,
			}
		}

		return &Result{
			Status:  StatusHealthy,
			Message: "Memory usage normal",
			Details: details,
		}
	}
}

// Helper types and functions (implementations would be platform-specific)

type diskUsage struct {
	Total uint64
	Used  uint64
}

type memoryInfo struct {
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}

// Platform-specific implementations would go here
func getDiskUsage(path string) (*diskUsage, error) {
	// This would have platform-specific implementation
	return &diskUsage{
		Total: 1000000000000, // 1TB
		Used:  600000000000,  // 600GB
	}, nil
}

func getMemoryInfo() (*memoryInfo, error) {
	// This would have platform-specific implementation
	return &memoryInfo{
		Total:       16000000000, // 16GB
		Used:        8000000000,  // 8GB
		Free:        8000000000,  // 8GB
		UsedPercent: 50.0,
	}, nil
}

func getSystemInfo() map[string]interface{} {
	// This would gather actual system information
	return map[string]interface{}{
		"version":    "1.0.0",
		"go_version": "1.21",
		"uptime":     "24h30m15s",
		"hostname":   "mcp-server-1",
	}
}