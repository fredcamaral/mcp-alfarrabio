package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message,omitempty"`
	LastCheck   time.Time              `json:"last_check"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// SystemHealth represents the overall system health
type SystemHealth struct {
	Status       HealthStatus   `json:"status"`
	Version      string         `json:"version"`
	Uptime       time.Duration  `json:"uptime"`
	Timestamp    time.Time      `json:"timestamp"`
	Checks       []HealthCheck  `json:"checks"`
	SystemInfo   SystemInfo     `json:"system_info"`
}

// SystemInfo contains system information
type SystemInfo struct {
	OS           string    `json:"os"`
	Architecture string    `json:"architecture"`
	GoVersion    string    `json:"go_version"`
	NumCPU       int       `json:"num_cpu"`
	NumGoroutine int       `json:"num_goroutine"`
	MemStats     MemStats  `json:"mem_stats"`
}

// MemStats contains memory statistics
type MemStats struct {
	Alloc        uint64  `json:"alloc"`
	TotalAlloc   uint64  `json:"total_alloc"`
	Sys          uint64  `json:"sys"`
	NumGC        uint32  `json:"num_gc"`
	HeapAlloc    uint64  `json:"heap_alloc"`
	HeapSys      uint64  `json:"heap_sys"`
	HeapInuse    uint64  `json:"heap_inuse"`
	StackInuse   uint64  `json:"stack_inuse"`
	StackSys     uint64  `json:"stack_sys"`
}

// HealthChecker defines the interface for health checkers
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) HealthCheck
}

// HealthManager manages system health checks
type HealthManager struct {
	checkers    []HealthChecker
	startTime   time.Time
	version     string
	lastChecks  map[string]HealthCheck
	checksMutex sync.RWMutex
}

// DatabaseHealthChecker checks database connectivity
type DatabaseHealthChecker struct {
	name string
	ping func(ctx context.Context) error
}

// VectorStorageHealthChecker checks vector storage connectivity
type VectorStorageHealthChecker struct {
	name string
	ping func(ctx context.Context) error
}

// EmbeddingServiceHealthChecker checks embedding service
type EmbeddingServiceHealthChecker struct {
	name string
	ping func(ctx context.Context) error
}

// MemoryHealthChecker checks memory usage
type MemoryHealthChecker struct {
	name string
	maxMemoryMB uint64
}

// NewHealthManager creates a new health manager
func NewHealthManager(version string) *HealthManager {
	return &HealthManager{
		checkers:   make([]HealthChecker, 0),
		startTime:  time.Now(),
		version:    version,
		lastChecks: make(map[string]HealthCheck),
	}
}

// AddChecker adds a health checker
func (hm *HealthManager) AddChecker(checker HealthChecker) {
	hm.checkers = append(hm.checkers, checker)
}

// CheckHealth performs all health checks and returns system health
func (hm *HealthManager) CheckHealth(ctx context.Context) *SystemHealth {
	start := time.Now()
	checks := make([]HealthCheck, 0, len(hm.checkers))
	
	// Run all health checks
	for _, checker := range hm.checkers {
		check := checker.Check(ctx)
		checks = append(checks, check)
		
		// Cache the check result
		hm.checksMutex.Lock()
		hm.lastChecks[checker.Name()] = check
		hm.checksMutex.Unlock()
	}
	
	// Determine overall status
	overallStatus := HealthStatusHealthy
	for _, check := range checks {
		switch check.Status {
		case HealthStatusUnhealthy:
			overallStatus = HealthStatusUnhealthy
		case HealthStatusDegraded:
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		}
	}
	
	return &SystemHealth{
		Status:     overallStatus,
		Version:    hm.version,
		Uptime:     time.Since(hm.startTime),
		Timestamp:  start,
		Checks:     checks,
		SystemInfo: hm.getSystemInfo(),
	}
}

// GetCachedHealth returns the last health check results
func (hm *HealthManager) GetCachedHealth() *SystemHealth {
	hm.checksMutex.RLock()
	defer hm.checksMutex.RUnlock()
	
	checks := make([]HealthCheck, 0, len(hm.lastChecks))
	overallStatus := HealthStatusHealthy
	
	for _, check := range hm.lastChecks {
		checks = append(checks, check)
		switch check.Status {
		case HealthStatusUnhealthy:
			overallStatus = HealthStatusUnhealthy
		case HealthStatusDegraded:
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		}
	}
	
	return &SystemHealth{
		Status:     overallStatus,
		Version:    hm.version,
		Uptime:     time.Since(hm.startTime),
		Timestamp:  time.Now(),
		Checks:     checks,
		SystemInfo: hm.getSystemInfo(),
	}
}

// StartPeriodicChecks starts periodic health checks
func (hm *HealthManager) StartPeriodicChecks(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hm.CheckHealth(ctx)
		}
	}
}

// HTTPHandler returns an HTTP handler for health checks
func (hm *HealthManager) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		
		health := hm.CheckHealth(ctx)
		
		w.Header().Set("Content-Type", "application/json")
		
		// Set HTTP status based on health
		switch health.Status {
		case HealthStatusHealthy:
			w.WriteHeader(http.StatusOK)
		case HealthStatusDegraded:
			w.WriteHeader(http.StatusOK) // Still OK, but degraded
		case HealthStatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		
		json.NewEncoder(w).Encode(health)
	}
}

// ReadinessHandler returns a readiness check handler
func (hm *HealthManager) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		
		health := hm.CheckHealth(ctx)
		
		w.Header().Set("Content-Type", "application/json")
		
		// Readiness is stricter - only healthy is ready
		if health.Status == HealthStatusHealthy {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"ready","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"not_ready","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
		}
	}
}

// LivenessHandler returns a liveness check handler
func (hm *HealthManager) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Liveness just checks if the service is running
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"alive","timestamp":"%s","uptime":"%s"}`, 
			time.Now().Format(time.RFC3339), time.Since(hm.startTime))
	}
}

// getSystemInfo collects system information
func (hm *HealthManager) getSystemInfo() SystemInfo {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
		MemStats: MemStats{
			Alloc:        memStats.Alloc,
			TotalAlloc:   memStats.TotalAlloc,
			Sys:          memStats.Sys,
			NumGC:        memStats.NumGC,
			HeapAlloc:    memStats.HeapAlloc,
			HeapSys:      memStats.HeapSys,
			HeapInuse:    memStats.HeapInuse,
			StackInuse:   memStats.StackInuse,
			StackSys:     memStats.StackSys,
		},
	}
}

// Health checker implementations

// NewDatabaseHealthChecker creates a database health checker
func NewDatabaseHealthChecker(name string, pingFunc func(ctx context.Context) error) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		name: name,
		ping: pingFunc,
	}
}

func (dhc *DatabaseHealthChecker) Name() string {
	return dhc.name
}

func (dhc *DatabaseHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()
	
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	err := dhc.ping(checkCtx)
	duration := time.Since(start)
	
	if err != nil {
		return HealthCheck{
			Name:      dhc.name,
			Status:    HealthStatusUnhealthy,
			Message:   fmt.Sprintf("Database ping failed: %v", err),
			LastCheck: start,
			Duration:  duration,
		}
	}
	
	status := HealthStatusHealthy
	message := "Database connection is healthy"
	
	// Check response time
	if duration > 1*time.Second {
		status = HealthStatusDegraded
		message = fmt.Sprintf("Database response time is slow: %v", duration)
	}
	
	return HealthCheck{
		Name:      dhc.name,
		Status:    status,
		Message:   message,
		LastCheck: start,
		Duration:  duration,
		Metadata: map[string]interface{}{
			"response_time_ms": duration.Milliseconds(),
		},
	}
}

// NewVectorStorageHealthChecker creates a vector storage health checker
func NewVectorStorageHealthChecker(name string, pingFunc func(ctx context.Context) error) *VectorStorageHealthChecker {
	return &VectorStorageHealthChecker{
		name: name,
		ping: pingFunc,
	}
}

func (vshc *VectorStorageHealthChecker) Name() string {
	return vshc.name
}

func (vshc *VectorStorageHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()
	
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	err := vshc.ping(checkCtx)
	duration := time.Since(start)
	
	if err != nil {
		return HealthCheck{
			Name:      vshc.name,
			Status:    HealthStatusUnhealthy,
			Message:   fmt.Sprintf("Vector storage ping failed: %v", err),
			LastCheck: start,
			Duration:  duration,
		}
	}
	
	status := HealthStatusHealthy
	message := "Vector storage is healthy"
	
	// Check response time
	if duration > 2*time.Second {
		status = HealthStatusDegraded
		message = fmt.Sprintf("Vector storage response time is slow: %v", duration)
	}
	
	return HealthCheck{
		Name:      vshc.name,
		Status:    status,
		Message:   message,
		LastCheck: start,
		Duration:  duration,
		Metadata: map[string]interface{}{
			"response_time_ms": duration.Milliseconds(),
		},
	}
}

// NewMemoryHealthChecker creates a memory health checker
func NewMemoryHealthChecker(maxMemoryMB uint64) *MemoryHealthChecker {
	return &MemoryHealthChecker{
		name:        "memory",
		maxMemoryMB: maxMemoryMB,
	}
}

func (mhc *MemoryHealthChecker) Name() string {
	return mhc.name
}

func (mhc *MemoryHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()
	
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	allocMB := memStats.Alloc / 1024 / 1024
	sysMB := memStats.Sys / 1024 / 1024
	
	status := HealthStatusHealthy
	message := fmt.Sprintf("Memory usage: %d MB allocated, %d MB system", allocMB, sysMB)
	
	if mhc.maxMemoryMB > 0 {
		if allocMB > mhc.maxMemoryMB {
			status = HealthStatusUnhealthy
			message = fmt.Sprintf("Memory usage exceeded limit: %d MB > %d MB", allocMB, mhc.maxMemoryMB)
		} else if allocMB > mhc.maxMemoryMB*80/100 {
			status = HealthStatusDegraded
			message = fmt.Sprintf("Memory usage is high: %d MB (limit: %d MB)", allocMB, mhc.maxMemoryMB)
		}
	}
	
	return HealthCheck{
		Name:      mhc.name,
		Status:    status,
		Message:   message,
		LastCheck: start,
		Duration:  time.Since(start),
		Metadata: map[string]interface{}{
			"alloc_mb":     allocMB,
			"sys_mb":       sysMB,
			"heap_mb":      memStats.HeapAlloc / 1024 / 1024,
			"num_gc":       memStats.NumGC,
			"goroutines":   runtime.NumGoroutine(),
		},
	}
}