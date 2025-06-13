// Package performance provides comprehensive monitoring and observability
package performance

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Monitor provides comprehensive performance monitoring and observability
type Monitor struct {
	config     *MonitorConfig
	metrics    *SystemMetrics
	collectors map[string]MetricCollector
	alerts     *AlertManager
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
}

// MonitorConfig defines monitoring configuration
type MonitorConfig struct {
	// Collection settings
	CollectionInterval  time.Duration `json:"collection_interval"`
	MetricRetention     time.Duration `json:"metric_retention"`
	EnableSystemMetrics bool          `json:"enable_system_metrics"`
	EnableCustomMetrics bool          `json:"enable_custom_metrics"`

	// Performance settings
	MetricBufferSize int           `json:"metric_buffer_size"`
	BatchSize        int           `json:"batch_size"`
	FlushInterval    time.Duration `json:"flush_interval"`

	// Alert settings
	EnableAlerting  bool               `json:"enable_alerting"`
	AlertThresholds map[string]float64 `json:"alert_thresholds"`
	AlertCooldown   time.Duration      `json:"alert_cooldown"`

	// Export settings
	EnableExport      bool   `json:"enable_export"`
	ExportFormat      string `json:"export_format"`
	ExportDestination string `json:"export_destination"`
}

// DefaultMonitorConfig returns optimized default configuration
func DefaultMonitorConfig() *MonitorConfig {
	return &MonitorConfig{
		CollectionInterval:  10 * time.Second,
		MetricRetention:     24 * time.Hour,
		EnableSystemMetrics: true,
		EnableCustomMetrics: true,
		MetricBufferSize:    10000,
		BatchSize:           100,
		FlushInterval:       30 * time.Second,
		EnableAlerting:      true,
		AlertThresholds: map[string]float64{
			"cpu_usage":     80.0,
			"memory_usage":  85.0,
			"response_time": 1000.0, // milliseconds
			"error_rate":    5.0,    // percentage
		},
		AlertCooldown:     5 * time.Minute,
		EnableExport:      false,
		ExportFormat:      "json",
		ExportDestination: "/tmp/metrics",
	}
}

// SystemMetrics represents comprehensive system performance metrics
type SystemMetrics struct {
	mutex     sync.RWMutex
	Timestamp time.Time `json:"timestamp"`

	// System resources
	CPUUsage     float64        `json:"cpu_usage"`
	MemoryUsage  MemoryMetrics  `json:"memory_usage"`
	DiskUsage    DiskMetrics    `json:"disk_usage"`
	NetworkUsage NetworkMetrics `json:"network_usage"`

	// Application metrics
	Goroutines  int             `json:"goroutines"`
	HeapObjects uint64          `json:"heap_objects"`
	GCPauses    []time.Duration `json:"gc_pauses"`

	// Performance metrics
	ResponseTimes     ResponseTimeMetrics `json:"response_times"`
	ThroughputMetrics ThroughputMetrics   `json:"throughput"`
	ErrorMetrics      ErrorMetrics        `json:"errors"`

	// Custom metrics
	CustomMetrics map[string]interface{} `json:"custom_metrics"`
}

// MemoryMetrics represents memory usage statistics
type MemoryMetrics struct {
	TotalAlloc   uint64  `json:"total_alloc"`
	Sys          uint64  `json:"sys"`
	Mallocs      uint64  `json:"mallocs"`
	Frees        uint64  `json:"frees"`
	LiveObjects  uint64  `json:"live_objects"`
	HeapAlloc    uint64  `json:"heap_alloc"`
	HeapSys      uint64  `json:"heap_sys"`
	HeapInuse    uint64  `json:"heap_inuse"`
	StackInuse   uint64  `json:"stack_inuse"`
	UsagePercent float64 `json:"usage_percent"`
}

// DiskMetrics represents disk usage statistics
type DiskMetrics struct {
	TotalSpace   uint64  `json:"total_space"`
	UsedSpace    uint64  `json:"used_space"`
	FreeSpace    uint64  `json:"free_space"`
	UsagePercent float64 `json:"usage_percent"`
	IOOperations uint64  `json:"io_operations"`
	IOBytes      uint64  `json:"io_bytes"`
}

// NetworkMetrics represents network usage statistics
type NetworkMetrics struct {
	BytesReceived   uint64 `json:"bytes_received"`
	BytesSent       uint64 `json:"bytes_sent"`
	PacketsReceived uint64 `json:"packets_received"`
	PacketsSent     uint64 `json:"packets_sent"`
	Connections     int    `json:"connections"`
}

// ResponseTimeMetrics represents response time statistics
type ResponseTimeMetrics struct {
	Mean   float64 `json:"mean"`
	Median float64 `json:"median"`
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Count  int64   `json:"count"`
}

// ThroughputMetrics represents throughput statistics
type ThroughputMetrics struct {
	RequestsPerSecond   float64 `json:"requests_per_second"`
	OperationsPerSecond float64 `json:"operations_per_second"`
	BytesPerSecond      float64 `json:"bytes_per_second"`
	ActiveConnections   int     `json:"active_connections"`
}

// ErrorMetrics represents error statistics
type ErrorMetrics struct {
	TotalErrors    int64            `json:"total_errors"`
	ErrorRate      float64          `json:"error_rate"`
	ErrorsByType   map[string]int64 `json:"errors_by_type"`
	ErrorsByCode   map[int]int64    `json:"errors_by_code"`
	Last5MinErrors int64            `json:"last_5min_errors"`
}

// MetricCollector defines interface for collecting specific metrics
type MetricCollector interface {
	Name() string
	Collect(ctx context.Context) (map[string]interface{}, error)
	Initialize() error
	Cleanup() error
}

// AlertManager manages performance alerts and notifications
type AlertManager struct {
	config    *MonitorConfig
	alerts    map[string]*Alert
	cooldowns map[string]time.Time
	mutex     sync.RWMutex
	handlers  []AlertHandler
}

// Alert represents a performance alert
type Alert struct {
	ID           string                 `json:"id"`
	Type         AlertType              `json:"type"`
	Severity     AlertSeverity          `json:"severity"`
	Message      string                 `json:"message"`
	Metric       string                 `json:"metric"`
	Threshold    float64                `json:"threshold"`
	CurrentValue float64                `json:"current_value"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// AlertType defines the type of alert
type AlertType string

const (
	AlertTypeThreshold    AlertType = "threshold"
	AlertTypeAnomaly      AlertType = "anomaly"
	AlertTypeTrend        AlertType = "trend"
	AlertTypeAvailability AlertType = "availability"
)

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertHandler defines interface for handling alerts
type AlertHandler interface {
	HandleAlert(alert *Alert) error
	Name() string
}

// NewMonitor creates a new comprehensive performance monitor
func NewMonitor(config *MonitorConfig) *Monitor {
	if config == nil {
		config = DefaultMonitorConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	monitor := &Monitor{
		config:     config,
		metrics:    &SystemMetrics{CustomMetrics: make(map[string]interface{})},
		collectors: make(map[string]MetricCollector),
		alerts:     NewAlertManager(config),
		ctx:        ctx,
		cancel:     cancel,
		running:    false,
	}

	// Initialize built-in collectors
	monitor.initializeCollectors()

	return monitor
}

// Start begins monitoring and metric collection
func (m *Monitor) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return fmt.Errorf("monitor already running")
	}

	// Initialize all collectors
	for name, collector := range m.collectors {
		if err := collector.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize collector %s: %w", name, err)
		}
	}

	m.running = true

	// Start collection goroutine
	go m.collectionLoop()

	// Start alert processing if enabled
	if m.config.EnableAlerting {
		go m.alertLoop()
	}

	// Start export if enabled
	if m.config.EnableExport {
		go m.exportLoop()
	}

	return nil
}

// Stop halts monitoring and cleanup resources
func (m *Monitor) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return nil
	}

	m.running = false
	m.cancel()

	// Cleanup all collectors
	for _, collector := range m.collectors {
		collector.Cleanup()
	}

	return nil
}

// GetMetrics returns current system metrics
func (m *Monitor) GetMetrics() SystemMetrics {
	m.metrics.mutex.RLock()
	defer m.metrics.mutex.RUnlock()

	// Return a copy without the mutex to avoid lock value copying
	return SystemMetrics{
		Timestamp:         m.metrics.Timestamp,
		CPUUsage:          m.metrics.CPUUsage,
		MemoryUsage:       m.metrics.MemoryUsage,
		DiskUsage:         m.metrics.DiskUsage,
		NetworkUsage:      m.metrics.NetworkUsage,
		Goroutines:        m.metrics.Goroutines,
		HeapObjects:       m.metrics.HeapObjects,
		GCPauses:          m.metrics.GCPauses,
		ResponseTimes:     m.metrics.ResponseTimes,
		ThroughputMetrics: m.metrics.ThroughputMetrics,
		ErrorMetrics:      m.metrics.ErrorMetrics,
		CustomMetrics:     m.metrics.CustomMetrics,
	}
}

// RecordCustomMetric records a custom application metric
func (m *Monitor) RecordCustomMetric(name string, value interface{}) {
	m.metrics.mutex.Lock()
	defer m.metrics.mutex.Unlock()

	m.metrics.CustomMetrics[name] = value
}

// AddCollector adds a custom metric collector
func (m *Monitor) AddCollector(collector MetricCollector) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.collectors[collector.Name()] = collector
}

// AddAlertHandler adds a custom alert handler
func (m *Monitor) AddAlertHandler(handler AlertHandler) {
	m.alerts.AddHandler(handler)
}

// initializeCollectors sets up built-in metric collectors
func (m *Monitor) initializeCollectors() {
	// System metrics collector
	if m.config.EnableSystemMetrics {
		m.collectors["system"] = &SystemCollector{}
	}

	// Memory metrics collector
	m.collectors["memory"] = &MemoryCollector{}

	// Performance metrics collector
	m.collectors["performance"] = &PerformanceCollector{}
}

// collectionLoop runs the main metric collection loop
func (m *Monitor) collectionLoop() {
	ticker := time.NewTicker(m.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.collectMetrics()
		case <-m.ctx.Done():
			return
		}
	}
}

// collectMetrics collects metrics from all registered collectors
func (m *Monitor) collectMetrics() {
	m.metrics.mutex.Lock()
	defer m.metrics.mutex.Unlock()

	m.metrics.Timestamp = time.Now()

	// Collect from all registered collectors
	for name, collector := range m.collectors {
		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
		metrics, err := collector.Collect(ctx)
		cancel()

		if err != nil {
			// Log error in production
			continue
		}

		// Merge metrics based on collector type
		m.mergeMetrics(name, metrics)
	}
}

// mergeMetrics merges collected metrics into system metrics
func (m *Monitor) mergeMetrics(collectorName string, metrics map[string]interface{}) {
	switch collectorName {
	case "system":
		if cpu, ok := metrics["cpu_usage"].(float64); ok {
			m.metrics.CPUUsage = cpu
		}
	case "memory":
		if memData, ok := metrics["memory"].(MemoryMetrics); ok {
			m.metrics.MemoryUsage = memData
		}
	case "performance":
		if respTime, ok := metrics["response_times"].(ResponseTimeMetrics); ok {
			m.metrics.ResponseTimes = respTime
		}
	default:
		// Store as custom metrics
		for key, value := range metrics {
			m.metrics.CustomMetrics[fmt.Sprintf("%s.%s", collectorName, key)] = value
		}
	}
}

// alertLoop processes alerts based on collected metrics
func (m *Monitor) alertLoop() {
	ticker := time.NewTicker(m.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.processAlerts()
		case <-m.ctx.Done():
			return
		}
	}
}

// processAlerts checks metrics against thresholds and generates alerts
func (m *Monitor) processAlerts() {
	metrics := m.GetMetrics()

	// Check CPU usage
	if threshold, exists := m.config.AlertThresholds["cpu_usage"]; exists {
		if metrics.CPUUsage > threshold {
			alert := &Alert{
				ID:           generateAlertID(),
				Type:         AlertTypeThreshold,
				Severity:     AlertSeverityWarning,
				Message:      fmt.Sprintf("CPU usage high: %.2f%% > %.2f%%", metrics.CPUUsage, threshold),
				Metric:       "cpu_usage",
				Threshold:    threshold,
				CurrentValue: metrics.CPUUsage,
				Timestamp:    time.Now(),
			}
			m.alerts.TriggerAlert(alert)
		}
	}

	// Check memory usage
	if threshold, exists := m.config.AlertThresholds["memory_usage"]; exists {
		if metrics.MemoryUsage.UsagePercent > threshold {
			alert := &Alert{
				ID:           generateAlertID(),
				Type:         AlertTypeThreshold,
				Severity:     AlertSeverityError,
				Message:      fmt.Sprintf("Memory usage high: %.2f%% > %.2f%%", metrics.MemoryUsage.UsagePercent, threshold),
				Metric:       "memory_usage",
				Threshold:    threshold,
				CurrentValue: metrics.MemoryUsage.UsagePercent,
				Timestamp:    time.Now(),
			}
			m.alerts.TriggerAlert(alert)
		}
	}

	// Check response time
	if threshold, exists := m.config.AlertThresholds["response_time"]; exists {
		if metrics.ResponseTimes.P95 > threshold {
			alert := &Alert{
				ID:           generateAlertID(),
				Type:         AlertTypeThreshold,
				Severity:     AlertSeverityWarning,
				Message:      fmt.Sprintf("Response time high: %.2fms > %.2fms", metrics.ResponseTimes.P95, threshold),
				Metric:       "response_time",
				Threshold:    threshold,
				CurrentValue: metrics.ResponseTimes.P95,
				Timestamp:    time.Now(),
			}
			m.alerts.TriggerAlert(alert)
		}
	}

	// Check error rate
	if threshold, exists := m.config.AlertThresholds["error_rate"]; exists {
		if metrics.ErrorMetrics.ErrorRate > threshold {
			alert := &Alert{
				ID:           generateAlertID(),
				Type:         AlertTypeThreshold,
				Severity:     AlertSeverityCritical,
				Message:      fmt.Sprintf("Error rate high: %.2f%% > %.2f%%", metrics.ErrorMetrics.ErrorRate, threshold),
				Metric:       "error_rate",
				Threshold:    threshold,
				CurrentValue: metrics.ErrorMetrics.ErrorRate,
				Timestamp:    time.Now(),
			}
			m.alerts.TriggerAlert(alert)
		}
	}
}

// exportLoop handles metric export if enabled
func (m *Monitor) exportLoop() {
	ticker := time.NewTicker(m.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.exportMetrics()
		case <-m.ctx.Done():
			return
		}
	}
}

// exportMetrics exports metrics to configured destination
func (m *Monitor) exportMetrics() {
	metrics := m.GetMetrics()

	switch m.config.ExportFormat {
	case "json":
		data, err := json.Marshal(metrics)
		if err != nil {
			return
		}

		// In production, would write to configured destination
		_ = data
	}
}

// Built-in collectors

// SystemCollector collects system-level metrics
type SystemCollector struct{}

func (c *SystemCollector) Name() string { return "system" }

func (c *SystemCollector) Initialize() error { return nil }

func (c *SystemCollector) Cleanup() error { return nil }

func (c *SystemCollector) Collect(ctx context.Context) (map[string]interface{}, error) {
	// Simplified system metrics collection
	return map[string]interface{}{
		"cpu_usage": getCPUUsage(),
	}, nil
}

// MemoryCollector collects memory metrics
type MemoryCollector struct{}

func (c *MemoryCollector) Name() string { return "memory" }

func (c *MemoryCollector) Initialize() error { return nil }

func (c *MemoryCollector) Cleanup() error { return nil }

func (c *MemoryCollector) Collect(ctx context.Context) (map[string]interface{}, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memMetrics := MemoryMetrics{
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		Mallocs:      m.Mallocs,
		Frees:        m.Frees,
		LiveObjects:  m.Mallocs - m.Frees,
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapInuse:    m.HeapInuse,
		StackInuse:   m.StackInuse,
		UsagePercent: float64(m.HeapInuse) / float64(m.HeapSys) * 100,
	}

	return map[string]interface{}{
		"memory": memMetrics,
	}, nil
}

// PerformanceCollector collects performance metrics
type PerformanceCollector struct{}

func (c *PerformanceCollector) Name() string { return "performance" }

func (c *PerformanceCollector) Initialize() error { return nil }

func (c *PerformanceCollector) Cleanup() error { return nil }

func (c *PerformanceCollector) Collect(ctx context.Context) (map[string]interface{}, error) {
	// Simplified performance metrics
	respTimeMetrics := ResponseTimeMetrics{
		Mean:   50.0,
		Median: 45.0,
		P95:    120.0,
		P99:    200.0,
		Min:    10.0,
		Max:    500.0,
		Count:  1000,
	}

	return map[string]interface{}{
		"response_times": respTimeMetrics,
	}, nil
}

// Alert Manager implementation

// NewAlertManager creates a new alert manager
func NewAlertManager(config *MonitorConfig) *AlertManager {
	return &AlertManager{
		config:    config,
		alerts:    make(map[string]*Alert),
		cooldowns: make(map[string]time.Time),
		handlers:  []AlertHandler{},
	}
}

// TriggerAlert triggers an alert if not in cooldown
func (am *AlertManager) TriggerAlert(alert *Alert) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check cooldown
	if lastAlert, exists := am.cooldowns[alert.Metric]; exists {
		if time.Since(lastAlert) < am.config.AlertCooldown {
			return // Still in cooldown
		}
	}

	// Store alert
	am.alerts[alert.ID] = alert
	am.cooldowns[alert.Metric] = time.Now()

	// Send to all handlers
	for _, handler := range am.handlers {
		go handler.HandleAlert(alert)
	}
}

// AddHandler adds an alert handler
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.handlers = append(am.handlers, handler)
}

// GetAlerts returns recent alerts
func (am *AlertManager) GetAlerts() []*Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	alerts := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		alerts = append(alerts, alert)
	}

	return alerts
}

// Utility functions
func getCPUUsage() float64 {
	// Simplified CPU usage calculation
	return 25.5 // Placeholder
}

func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}
