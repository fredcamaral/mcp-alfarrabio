package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"mcp-memory/internal/logging"
)

// MonitoringManager handles application monitoring and metrics
type MonitoringManager struct {
	logger       logging.Logger
	metrics      *MetricsCollector
	alerts       *AlertManager
	healthMgr    *HealthManager
	enabled      bool
	interval     time.Duration
	stopChan     chan struct{}
	doneChan     chan struct{}
	mutex        sync.RWMutex
}

// MetricsCollector collects and stores application metrics
type MetricsCollector struct {
	mutex            sync.RWMutex
	counters         map[string]int64
	gauges           map[string]float64
	histograms       map[string][]float64
	lastReset        time.Time
	retentionPeriod  time.Duration
}

// MetricSnapshot represents a point-in-time view of metrics
type MetricSnapshot struct {
	Timestamp  time.Time         `json:"timestamp"`
	Counters   map[string]int64  `json:"counters"`
	Gauges     map[string]float64 `json:"gauges"`
	Histograms map[string]HistogramStats `json:"histograms"`
	System     SystemMetrics     `json:"system"`
}

// HistogramStats provides statistical analysis of histogram data
type HistogramStats struct {
	Count   int     `json:"count"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Mean    float64 `json:"mean"`
	Median  float64 `json:"median"`
	P95     float64 `json:"p95"`
	P99     float64 `json:"p99"`
}

// SystemMetrics contains system-level metrics
type SystemMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsed    uint64  `json:"memory_used"`
	MemoryTotal   uint64  `json:"memory_total"`
	MemoryPercent float64 `json:"memory_percent"`
	Goroutines    int     `json:"goroutines"`
	GCPauses      float64 `json:"gc_pauses_ms"`
	Uptime        float64 `json:"uptime_seconds"`
}

// AlertManager handles alerting based on metrics thresholds
type AlertManager struct {
	logger    logging.Logger
	rules     []AlertRule
	callbacks []AlertCallback
	mutex     sync.RWMutex
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	MetricName  string        `json:"metric_name"`
	Operator    string        `json:"operator"` // gt, lt, eq, gte, lte
	Threshold   float64       `json:"threshold"`
	Duration    time.Duration `json:"duration"`
	Severity    string        `json:"severity"` // critical, warning, info
	Enabled     bool          `json:"enabled"`
	lastFired   time.Time
	cooldown    time.Duration
}

// AlertCallback defines a function to call when an alert fires
type AlertCallback func(alert Alert)

// Alert represents a triggered alert
type Alert struct {
	Rule        AlertRule `json:"rule"`
	Value       float64   `json:"value"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
}

// NewMonitoringManager creates a new monitoring manager
func NewMonitoringManager(logger logging.Logger, healthMgr *HealthManager, interval time.Duration) *MonitoringManager {
	return &MonitoringManager{
		logger:    logger,
		metrics:   NewMetricsCollector(24 * time.Hour),
		alerts:    NewAlertManager(logger),
		healthMgr: healthMgr,
		enabled:   false,
		interval:  interval,
		stopChan:  make(chan struct{}),
		doneChan:  make(chan struct{}),
	}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(retentionPeriod time.Duration) *MetricsCollector {
	return &MetricsCollector{
		counters:        make(map[string]int64),
		gauges:          make(map[string]float64),
		histograms:      make(map[string][]float64),
		lastReset:       time.Now(),
		retentionPeriod: retentionPeriod,
	}
}

// NewAlertManager creates a new alert manager
func NewAlertManager(logger logging.Logger) *AlertManager {
	return &AlertManager{
		logger:    logger,
		rules:     make([]AlertRule, 0),
		callbacks: make([]AlertCallback, 0),
	}
}

// Start begins monitoring collection
func (mm *MonitoringManager) Start(ctx context.Context) error {
	mm.mutex.Lock()
	if mm.enabled {
		mm.mutex.Unlock()
		return fmt.Errorf("monitoring already started")
	}
	mm.enabled = true
	mm.mutex.Unlock()

	mm.logger.Info("Starting monitoring", "interval", mm.interval)

	go mm.monitoringLoop(ctx)
	return nil
}

// Stop stops monitoring collection
func (mm *MonitoringManager) Stop() error {
	mm.mutex.Lock()
	if !mm.enabled {
		mm.mutex.Unlock()
		return fmt.Errorf("monitoring not started")
	}
	mm.enabled = false
	mm.mutex.Unlock()

	close(mm.stopChan)
	<-mm.doneChan

	mm.logger.Info("Monitoring stopped")
	return nil
}

// monitoringLoop runs the main monitoring collection loop
func (mm *MonitoringManager) monitoringLoop(ctx context.Context) {
	defer close(mm.doneChan)

	ticker := time.NewTicker(mm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-mm.stopChan:
			return
		case <-ticker.C:
			mm.collectMetrics(ctx)
			mm.checkAlerts()
		}
	}
}

// collectMetrics collects current system and application metrics
func (mm *MonitoringManager) collectMetrics(ctx context.Context) {
	// Collect system metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	systemMetrics := SystemMetrics{
		MemoryUsed:    memStats.Alloc,
		MemoryTotal:   memStats.Sys,
		MemoryPercent: float64(memStats.Alloc) / float64(memStats.Sys) * 100,
		Goroutines:    runtime.NumGoroutine(),
		GCPauses:      float64(memStats.PauseTotalNs) / 1e6, // Convert to milliseconds
	}

	// Update gauges with system metrics
	mm.metrics.SetGauge("system.memory.used", float64(systemMetrics.MemoryUsed))
	mm.metrics.SetGauge("system.memory.total", float64(systemMetrics.MemoryTotal))
	mm.metrics.SetGauge("system.memory.percent", systemMetrics.MemoryPercent)
	mm.metrics.SetGauge("system.goroutines", float64(systemMetrics.Goroutines))
	mm.metrics.SetGauge("system.gc.pauses", systemMetrics.GCPauses)

	// Collect health metrics if health manager is available
	if mm.healthMgr != nil {
		health := mm.healthMgr.CheckHealth(ctx)
		
		// Convert health status to numeric score
		var healthScore float64
		switch health.Status {
		case HealthStatusHealthy:
			healthScore = 1.0
		case HealthStatusDegraded:
			healthScore = 0.5
		case HealthStatusUnhealthy:
			healthScore = 0.0
		case HealthStatusUnknown:
			// Unknown status gets a low score
			healthScore = 0.0
		default:
			// This should not happen given our HealthStatus enum
			healthScore = 0.0
		}
		mm.metrics.SetGauge("health.overall.score", healthScore)
		
		for _, check := range health.Checks {
			var checkScore float64
			switch check.Status {
			case HealthStatusHealthy:
				checkScore = 1.0
			case HealthStatusDegraded:
				checkScore = 0.5
			case HealthStatusUnhealthy:
				checkScore = 0.0
			case HealthStatusUnknown:
				// Unknown status gets a low score
				checkScore = 0.0
			default:
				// This should not happen given our HealthStatus enum
				checkScore = 0.0
			}
			mm.metrics.SetGauge(fmt.Sprintf("health.%s.score", check.Name), checkScore)
			mm.metrics.SetGauge(fmt.Sprintf("health.%s.response_time", check.Name), check.Duration.Seconds())
		}
	}
}

// checkAlerts evaluates alert rules against current metrics
func (mm *MonitoringManager) checkAlerts() {
	snapshot := mm.GetSnapshot()
	mm.alerts.EvaluateRules(snapshot)
}

// Counter methods
func (mc *MetricsCollector) IncrementCounter(name string) {
	mc.AddToCounter(name, 1)
}

func (mc *MetricsCollector) AddToCounter(name string, value int64) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.counters[name] += value
}

func (mc *MetricsCollector) GetCounter(name string) int64 {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.counters[name]
}

// Gauge methods
func (mc *MetricsCollector) SetGauge(name string, value float64) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.gauges[name] = value
}

func (mc *MetricsCollector) GetGauge(name string) float64 {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.gauges[name]
}

// Histogram methods
func (mc *MetricsCollector) RecordHistogram(name string, value float64) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	if mc.histograms[name] == nil {
		mc.histograms[name] = make([]float64, 0)
	}
	
	mc.histograms[name] = append(mc.histograms[name], value)
	
	// Limit histogram size to prevent memory issues
	if len(mc.histograms[name]) > 10000 {
		mc.histograms[name] = mc.histograms[name][len(mc.histograms[name])-5000:]
	}
}

// GetSnapshot returns a snapshot of current metrics
func (mm *MonitoringManager) GetSnapshot() MetricSnapshot {
	return mm.metrics.GetSnapshot()
}

func (mc *MetricsCollector) GetSnapshot() MetricSnapshot {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	// Copy current metrics
	counters := make(map[string]int64)
	for k, v := range mc.counters {
		counters[k] = v
	}

	gauges := make(map[string]float64)
	for k, v := range mc.gauges {
		gauges[k] = v
	}

	histograms := make(map[string]HistogramStats)
	for k, v := range mc.histograms {
		histograms[k] = calculateHistogramStats(v)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return MetricSnapshot{
		Timestamp:  time.Now(),
		Counters:   counters,
		Gauges:     gauges,
		Histograms: histograms,
		System: SystemMetrics{
			MemoryUsed:    memStats.Alloc,
			MemoryTotal:   memStats.Sys,
			MemoryPercent: float64(memStats.Alloc) / float64(memStats.Sys) * 100,
			Goroutines:    runtime.NumGoroutine(),
			GCPauses:      float64(memStats.PauseTotalNs) / 1e6,
		},
	}
}

// calculateHistogramStats computes statistical measures for histogram data
func calculateHistogramStats(values []float64) HistogramStats {
	if len(values) == 0 {
		return HistogramStats{}
	}

	// Sort values for percentile calculations
	sorted := make([]float64, len(values))
	copy(sorted, values)
	
	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted); i++ {
		for j := 0; j < len(sorted)-1-i; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	stats := HistogramStats{
		Count: len(values),
		Min:   sorted[0],
		Max:   sorted[len(sorted)-1],
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	stats.Mean = sum / float64(len(values))

	// Calculate percentiles
	stats.Median = percentile(sorted, 0.5)
	stats.P95 = percentile(sorted, 0.95)
	stats.P99 = percentile(sorted, 0.99)

	return stats
}

// percentile calculates the given percentile from sorted data
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	
	index := p * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1
	
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// HTTPHandler returns an HTTP handler for metrics endpoint
func (mm *MonitoringManager) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot := mm.GetSnapshot()
		
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(snapshot); err != nil {
			http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
			return
		}
	}
}

// Alert management methods
func (am *AlertManager) AddRule(rule AlertRule) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	
	rule.cooldown = 5 * time.Minute // Default cooldown
	am.rules = append(am.rules, rule)
	am.logger.Info("Added alert rule", "name", rule.Name)
}

func (am *AlertManager) AddCallback(callback AlertCallback) {
	am.mutex.Lock()
	defer am.mutex.Unlock()
	am.callbacks = append(am.callbacks, callback)
}

func (am *AlertManager) EvaluateRules(snapshot MetricSnapshot) {
	am.mutex.RLock()
	rules := make([]AlertRule, len(am.rules))
	copy(rules, am.rules)
	callbacks := make([]AlertCallback, len(am.callbacks))
	copy(callbacks, am.callbacks)
	am.mutex.RUnlock()

	for i, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// Check cooldown
		if time.Since(rule.lastFired) < rule.cooldown {
			continue
		}

		// Get metric value
		var value float64
		var found bool

		if val, ok := snapshot.Counters[rule.MetricName]; ok {
			value = float64(val)
			found = true
		} else if val, ok := snapshot.Gauges[rule.MetricName]; ok {
			value = val
			found = true
		}

		if !found {
			continue
		}

		// Evaluate condition
		triggered := false
		switch rule.Operator {
		case "gt":
			triggered = value > rule.Threshold
		case "gte":
			triggered = value >= rule.Threshold
		case "lt":
			triggered = value < rule.Threshold
		case "lte":
			triggered = value <= rule.Threshold
		case "eq":
			triggered = value == rule.Threshold
		}

		if triggered {
			alert := Alert{
				Rule:        rule,
				Value:       value,
				Timestamp:   snapshot.Timestamp,
				Description: fmt.Sprintf("Metric %s is %f, threshold is %f", rule.MetricName, value, rule.Threshold),
			}

			am.logger.Warn("Alert triggered", "rule", rule.Name, "value", value, "threshold", rule.Threshold)

			// Update last fired time
			am.mutex.Lock()
			am.rules[i].lastFired = time.Now()
			am.mutex.Unlock()

			// Execute callbacks
			for _, callback := range callbacks {
				go callback(alert)
			}
		}
	}
}