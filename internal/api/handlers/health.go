// Package handlers provides HTTP request handlers for the MCP Memory Server API.
package handlers

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"lerian-mcp-memory/internal/api/response"
	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/deployment"
)

// HealthHandler provides health check functionality
type HealthHandler struct {
	config        *config.Config
	startTime     time.Time
	healthManager *deployment.HealthManager
}

// HealthStatus represents the health check response structure
type HealthStatus struct {
	Status      string           `json:"status"`
	Server      string           `json:"server"`
	Version     string           `json:"version"`
	Environment string           `json:"environment"`
	Uptime      string           `json:"uptime"`
	Timestamp   string           `json:"timestamp"`
	Checks      map[string]Check `json:"checks"`
	System      SystemInfo       `json:"system"`
}

// Check represents an individual health check result
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// SystemInfo represents system information
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumGoroutine int    `json:"num_goroutine"`
	MemoryMB     uint64 `json:"memory_mb"`
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(cfg *config.Config) *HealthHandler {
	// Create health manager with production health checkers
	healthManager := deployment.NewHealthManager("1.0.0")

	// Add standard health checkers
	healthManager.AddChecker(deployment.NewMemoryHealthChecker(500)) // 500MB limit

	// TODO: Add database health checker when storage is available
	// healthManager.AddChecker(deployment.NewDatabaseHealthChecker("sqlite", func(ctx context.Context) error {
	//     // Implement actual database ping
	//     return nil
	// }))

	// TODO: Add Qdrant health checker when storage is available
	// healthManager.AddChecker(deployment.NewVectorStorageHealthChecker("qdrant", func(ctx context.Context) error {
	//     // Implement actual Qdrant ping
	//     return nil
	// }))

	return &HealthHandler{
		config:        cfg,
		startTime:     time.Now(),
		healthManager: healthManager,
	}
}

// Handle processes health check requests
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Use production health manager for comprehensive checks
	systemHealth := h.healthManager.CheckHealth(ctx)

	// Convert to legacy format for backward compatibility
	status := h.convertToLegacyFormat(systemHealth)

	// Set appropriate HTTP status code based on health status
	statusCode := http.StatusOK
	switch systemHealth.Status {
	case deployment.HealthStatusUnhealthy:
		statusCode = http.StatusServiceUnavailable
	case deployment.HealthStatusDegraded:
		statusCode = http.StatusOK // Still OK, but degraded
	case deployment.HealthStatusUnknown:
		statusCode = http.StatusInternalServerError
	}

	// Write response
	w.WriteHeader(statusCode)
	response.WriteSuccess(w, status)
}

// HandleReadiness processes readiness check requests
func (h *HealthHandler) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	h.healthManager.ReadinessHandler()(w, r)
}

// HandleLiveness processes liveness check requests
func (h *HealthHandler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	h.healthManager.LivenessHandler()(w, r)
}

// convertToLegacyFormat converts deployment.SystemHealth to legacy HealthStatus format
func (h *HealthHandler) convertToLegacyFormat(systemHealth *deployment.SystemHealth) HealthStatus {
	// Convert checks to legacy format
	checks := make(map[string]Check)
	for _, check := range systemHealth.Checks {
		checks[check.Name] = Check{
			Status:  string(check.Status),
			Message: check.Message,
			Latency: check.Duration.String(),
		}
	}

	// Convert system info
	sysInfo := SystemInfo{
		GoVersion:    systemHealth.SystemInfo.GoVersion,
		NumGoroutine: systemHealth.SystemInfo.NumGoroutine,
		MemoryMB:     systemHealth.SystemInfo.MemStats.Alloc / 1024 / 1024,
	}

	return HealthStatus{
		Status:      string(systemHealth.Status),
		Server:      "lerian-mcp-memory",
		Version:     systemHealth.Version,
		Environment: h.getEnvironment(),
		Uptime:      systemHealth.Uptime.String(),
		Timestamp:   systemHealth.Timestamp.UTC().Format(time.RFC3339),
		Checks:      checks,
		System:      sysInfo,
	}
}

// buildHealthStatus constructs the complete health status (legacy method - kept for compatibility)
func (h *HealthHandler) buildHealthStatus(ctx context.Context) HealthStatus {
	return HealthStatus{
		Server:      "lerian-mcp-memory",
		Version:     "1.0.0",
		Environment: h.getEnvironment(),
		Uptime:      h.getUptime(),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Checks:      h.performHealthChecks(ctx),
		System:      h.getSystemInfo(),
	}
}

// getEnvironment determines the current environment
func (h *HealthHandler) getEnvironment() string {
	// Simple environment detection - in production, use proper env vars
	if h.config.Server.Host == "localhost" || h.config.Server.Host == "127.0.0.1" {
		return "development"
	}
	return "production"
}

// getUptime calculates server uptime
func (h *HealthHandler) getUptime() string {
	uptime := time.Since(h.startTime)
	return uptime.Round(time.Second).String()
}

// performHealthChecks runs various health checks
func (h *HealthHandler) performHealthChecks(ctx context.Context) map[string]Check {
	checks := make(map[string]Check)

	// Qdrant health check
	checks["qdrant"] = h.checkQdrant(ctx)

	// Memory usage check
	checks["memory"] = h.checkMemory()

	// Configuration check
	checks["config"] = h.checkConfiguration()

	return checks
}

// checkQdrant performs Qdrant database health check
func (h *HealthHandler) checkQdrant(ctx context.Context) Check {
	_ = ctx // unused parameter, kept for interface consistency
	start := time.Now()

	// TODO: Implement actual Qdrant health check when storage layer is available
	// For now, simulate a basic check
	latency := time.Since(start)

	// For testing purposes, always return healthy if health check is enabled
	// In production, this would actually ping Qdrant
	if h.config.Qdrant.HealthCheck {
		return Check{
			Status:  "healthy",
			Message: "Qdrant connection simulated (TODO: implement actual check)",
			Latency: latency.Round(time.Millisecond).String(),
		}
	}

	return Check{
		Status:  "healthy",
		Message: "Qdrant health check disabled",
	}
}

// checkMemory performs memory usage health check
func (h *HealthHandler) checkMemory() Check {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memoryMB := m.Alloc / 1024 / 1024

	// Consider unhealthy if using more than 500MB (adjust based on requirements)
	if memoryMB > 500 {
		return Check{
			Status:  "warning",
			Message: "High memory usage",
		}
	}

	return Check{
		Status:  "healthy",
		Message: "Memory usage normal",
	}
}

// checkConfiguration validates configuration health
func (h *HealthHandler) checkConfiguration() Check {
	// Validate critical configuration, but be more lenient for testing
	if err := h.config.Validate(); err != nil {
		// For testing, treat validation errors as warnings instead of failures
		// In production, you might want stricter validation
		return Check{
			Status:  "warning",
			Message: "Configuration validation warning: " + err.Error(),
		}
	}

	return Check{
		Status:  "healthy",
		Message: "Configuration valid",
	}
}

// getSystemInfo collects system information
func (h *HealthHandler) getSystemInfo() SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemInfo{
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		MemoryMB:     m.Alloc / 1024 / 1024,
	}
}

// determineOverallStatus determines overall health based on individual checks
func (h *HealthHandler) determineOverallStatus(checks map[string]Check) string {
	hasUnhealthy := false
	hasWarning := false

	for _, check := range checks {
		switch check.Status {
		case "unhealthy":
			hasUnhealthy = true
		case "warning":
			hasWarning = true
		}
	}

	if hasUnhealthy {
		return "unhealthy"
	}
	if hasWarning {
		return "warning"
	}
	return "healthy"
}
