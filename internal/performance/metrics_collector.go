package performance

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// MetricCategory represents different categories of metrics
type MetricCategory string

const (
	CategorySystem      MetricCategory = "system"
	CategoryApplication MetricCategory = "application"
	CategoryDatabase    MetricCategory = "database"
	CategoryCache       MetricCategory = "cache"
	CategoryNetwork     MetricCategory = "network"
	CategorySecurity    MetricCategory = "security"
	CategoryBusiness    MetricCategory = "business"
	CategoryCustom      MetricCategory = "custom"
)

// MetricUnit represents the unit of measurement for metrics
type MetricUnit string

const (
	UnitNone         MetricUnit = ""
	UnitBytes        MetricUnit = "bytes"
	UnitCount        MetricUnit = "count"
	UnitPercent      MetricUnit = "percent"
	UnitSeconds      MetricUnit = "seconds"
	UnitMilliseconds MetricUnit = "milliseconds"
	UnitMicroseconds MetricUnit = "microseconds"
	UnitNanoseconds  MetricUnit = "nanoseconds"
	UnitOpsPerSecond MetricUnit = "ops/sec"
	UnitBytesPerSec  MetricUnit = "bytes/sec"
	UnitRequests     MetricUnit = "requests"
	UnitErrors       MetricUnit = "errors"
	UnitConnections  MetricUnit = "connections"
)

// MetricAggregationType defines how metrics should be aggregated
type MetricAggregationType string

const (
	AggregationSum       MetricAggregationType = "sum"
	AggregationAverage   MetricAggregationType = "average"
	AggregationMin       MetricAggregationType = "min"
	AggregationMax       MetricAggregationType = "max"
	AggregationMedian    MetricAggregationType = "median"
	AggregationP95       MetricAggregationType = "p95"
	AggregationP99       MetricAggregationType = "p99"
	AggregationRate      MetricAggregationType = "rate"
	AggregationGauge     MetricAggregationType = "gauge"
	AggregationHistogram MetricAggregationType = "histogram"
)

// PerformanceMetricV2 represents an enhanced performance metric with rich metadata
type PerformanceMetricV2 struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Category        MetricCategory         `json:"category"`
	Type            MetricAggregationType  `json:"type"`
	Value           float64                `json:"value"`
	Unit            MetricUnit             `json:"unit"`
	Timestamp       time.Time              `json:"timestamp"`
	Tags            map[string]string      `json:"tags"`
	Labels          map[string]string      `json:"labels"`
	Metadata        map[string]interface{} `json:"metadata"`
	Source          string                 `json:"source"`
	Environment     string                 `json:"environment"`
	Component       string                 `json:"component"`
	Threshold       *MetricThreshold       `json:"threshold,omitempty"`
	SampleCount     int64                  `json:"sample_count"`
	WindowDuration  time.Duration          `json:"window_duration"`
	Cardinality     int                    `json:"cardinality"`
	IsAnomaly       bool                   `json:"is_anomaly"`
	AnomalyScore    float64                `json:"anomaly_score"`
	TrendDirection  string                 `json:"trend_direction"` // "up", "down", "stable"
	SeasonalityInfo *SeasonalityInfo       `json:"seasonality_info,omitempty"`
}

// MetricThreshold defines threshold configuration for metrics
type MetricThreshold struct {
	Warning    float64 `json:"warning"`
	Critical   float64 `json:"critical"`
	Operator   string  `json:"operator"` // "gt", "lt", "gte", "lte", "eq", "neq"
	Enabled    bool    `json:"enabled"`
	Hysteresis float64 `json:"hysteresis"` // To prevent flapping
}

// SeasonalityInfo contains information about metric seasonality patterns
type SeasonalityInfo struct {
	Period       time.Duration `json:"period"`
	Amplitude    float64       `json:"amplitude"`
	Phase        float64       `json:"phase"`
	Confidence   float64       `json:"confidence"`
	LastAnalyzed time.Time     `json:"last_analyzed"`
}

// MetricSeries represents a time series of metric values
type MetricSeries struct {
	MetricName      string                 `json:"metric_name"`
	Points          []*MetricDataPoint     `json:"points"`
	Aggregation     MetricAggregationType  `json:"aggregation"`
	Resolution      time.Duration          `json:"resolution"`
	RetentionPolicy *MetricRetentionPolicy `json:"retention_policy"`
	Statistics      *SeriesStatistics      `json:"statistics"`
	mutex           sync.RWMutex
}

// MetricDataPoint represents a single data point in a time series
type MetricDataPoint struct {
	Timestamp    time.Time              `json:"timestamp"`
	Value        float64                `json:"value"`
	Tags         map[string]string      `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	Quality      float64                `json:"quality"` // Data quality score 0-1
	Interpolated bool                   `json:"interpolated"`
}

// MetricRetentionPolicy defines how long to keep metrics
type MetricRetentionPolicy struct {
	RawDataRetention        time.Duration `json:"raw_data_retention"`
	AggregatedDataRetention time.Duration `json:"aggregated_data_retention"`
	CompressionThreshold    time.Duration `json:"compression_threshold"`
	CompressionRatio        float64       `json:"compression_ratio"`
}

// SeriesStatistics contains statistical analysis of a metric series
type SeriesStatistics struct {
	Count           int64     `json:"count"`
	Sum             float64   `json:"sum"`
	Mean            float64   `json:"mean"`
	Median          float64   `json:"median"`
	Mode            float64   `json:"mode"`
	Min             float64   `json:"min"`
	Max             float64   `json:"max"`
	Range           float64   `json:"range"`
	Variance        float64   `json:"variance"`
	StdDev          float64   `json:"std_dev"`
	Skewness        float64   `json:"skewness"`
	Kurtosis        float64   `json:"kurtosis"`
	P50             float64   `json:"p50"`
	P90             float64   `json:"p90"`
	P95             float64   `json:"p95"`
	P99             float64   `json:"p99"`
	P999            float64   `json:"p999"`
	LastValue       float64   `json:"last_value"`
	FirstValue      float64   `json:"first_value"`
	Trend           float64   `json:"trend"`       // Linear trend coefficient
	Correlation     float64   `json:"correlation"` // Correlation with time
	AutoCorrelation float64   `json:"auto_correlation"`
	LastUpdated     time.Time `json:"last_updated"`
}

// MetricsCollectorV2 provides advanced metrics collection with real-time analysis
type MetricsCollectorV2 struct {
	// Core storage
	metrics           map[string]*PerformanceMetricV2
	metricSeries      map[string]*MetricSeries
	aggregatedMetrics map[string]map[MetricAggregationType]*PerformanceMetricV2

	// Configuration
	config            *MetricsConfig
	retentionPolicies map[string]*MetricRetentionPolicy
	thresholds        map[string]*MetricThreshold

	// Synchronization
	metricsMutex    sync.RWMutex
	seriesMutex     sync.RWMutex
	aggregatedMutex sync.RWMutex

	// Background processing
	ctx          context.Context
	cancel       context.CancelFunc
	backgroundWG sync.WaitGroup

	// Performance counters
	operationCount int64
	totalLatency   int64
	errorCount     int64
	lastFlush      time.Time

	// Advanced features
	anomalyDetector   *AnomalyDetector
	trendAnalyzer     *TrendAnalyzer
	correlationEngine *CorrelationEngine
	alertingEngine    *AlertingEngine

	// Buffering and batching
	metricBuffer  []*PerformanceMetricV2
	bufferMutex   sync.Mutex
	bufferSize    int
	flushInterval time.Duration
	batchSize     int

	// Export and integration
	exporters      []MetricExporter
	exportInterval time.Duration

	// Health and diagnostics
	healthMetrics map[string]float64
	healthMutex   sync.RWMutex

	enabled bool
}

// MetricsConfig holds configuration for the metrics collector
type MetricsConfig struct {
	CollectionInterval   time.Duration     `json:"collection_interval"`
	RetentionDuration    time.Duration     `json:"retention_duration"`
	MaxMetrics           int               `json:"max_metrics"`
	MaxSeriesLength      int               `json:"max_series_length"`
	CompressionEnabled   bool              `json:"compression_enabled"`
	AnomalyDetection     bool              `json:"anomaly_detection"`
	TrendAnalysis        bool              `json:"trend_analysis"`
	CorrelationAnalysis  bool              `json:"correlation_analysis"`
	ExportEnabled        bool              `json:"export_enabled"`
	BufferSize           int               `json:"buffer_size"`
	FlushInterval        time.Duration     `json:"flush_interval"`
	BatchSize            int               `json:"batch_size"`
	SamplingRate         float64           `json:"sampling_rate"`
	HighCardinalityLimit int               `json:"high_cardinality_limit"`
	DefaultTags          map[string]string `json:"default_tags"`
}

// AnomalyDetector detects anomalies in metric data
type AnomalyDetector struct {
	models           map[string]*AnomalyModel
	threshold        float64
	minDataPoints    int
	learningEnabled  bool
	sensitivityLevel float64
	// mutex           sync.RWMutex // TODO: Reserved for future thread-safe anomaly detection operations
}

// AnomalyModel represents a trained anomaly detection model
type AnomalyModel struct {
	MetricName       string             `json:"metric_name"`
	ModelType        string             `json:"model_type"` // "statistical", "ml", "seasonal"
	Parameters       map[string]float64 `json:"parameters"`
	Threshold        float64            `json:"threshold"`
	Accuracy         float64            `json:"accuracy"`
	LastTrained      time.Time          `json:"last_trained"`
	TrainingDataSize int                `json:"training_data_size"`
}

// TrendAnalyzer analyzes trends in metric data
type TrendAnalyzer struct {
	trendModels      map[string]*TrendModel
	windowSize       time.Duration
	sensitivityLevel float64
	enabled          bool
	mutex            sync.RWMutex
}

// TrendModel represents a trend analysis model
type TrendModel struct {
	MetricName     string    `json:"metric_name"`
	Slope          float64   `json:"slope"`
	Intercept      float64   `json:"intercept"`
	Correlation    float64   `json:"correlation"`
	Confidence     float64   `json:"confidence"`
	Direction      string    `json:"direction"` // "increasing", "decreasing", "stable"
	LastAnalyzed   time.Time `json:"last_analyzed"`
	DataPointsUsed int       `json:"data_points_used"`
}

// CorrelationEngine analyzes correlations between different metrics
type CorrelationEngine struct {
	correlations   map[string]map[string]float64
	minCorrelation float64
	windowSize     time.Duration
	enabled        bool
	mutex          sync.RWMutex
}

// AlertingEngine handles metric-based alerting
type AlertingEngine struct {
	rules           map[string]*AlertRule
	alertHistory    []*Alert
	cooldownPeriods map[string]time.Time
	enabled         bool
	mutex           sync.RWMutex
}

// AlertRule defines an alerting rule
type AlertRule struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	MetricName     string            `json:"metric_name"`
	Condition      string            `json:"condition"`
	Threshold      float64           `json:"threshold"`
	Duration       time.Duration     `json:"duration"`
	Severity       string            `json:"severity"`
	Tags           map[string]string `json:"tags"`
	CooldownPeriod time.Duration     `json:"cooldown_period"`
	Enabled        bool              `json:"enabled"`
	LastTriggered  time.Time         `json:"last_triggered"`
	TriggerCount   int64             `json:"trigger_count"`
}

// Alert represents a triggered alert
type Alert struct {
	ID           string                 `json:"id"`
	RuleID       string                 `json:"rule_id"`
	MetricName   string                 `json:"metric_name"`
	Value        float64                `json:"value"`
	Threshold    float64                `json:"threshold"`
	Severity     string                 `json:"severity"`
	Message      string                 `json:"message"`
	Timestamp    time.Time              `json:"timestamp"`
	Tags         map[string]string      `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	Acknowledged bool                   `json:"acknowledged"`
	Resolved     bool                   `json:"resolved"`
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty"`
}

// MetricExporter interface for exporting metrics to external systems
type MetricExporter interface {
	Export(metrics []*PerformanceMetricV2) error
	GetName() string
	IsHealthy() bool
	GetExportStats() map[string]interface{}
}

// NewMetricsCollectorV2 creates a new enhanced metrics collector
func NewMetricsCollectorV2(ctx context.Context, config *MetricsConfig) *MetricsCollectorV2 {
	if config == nil {
		config = getDefaultMetricsConfig()
	}

	collectorCtx, cancel := context.WithCancel(ctx)

	mc := &MetricsCollectorV2{
		metrics:           make(map[string]*PerformanceMetricV2),
		metricSeries:      make(map[string]*MetricSeries),
		aggregatedMetrics: make(map[string]map[MetricAggregationType]*PerformanceMetricV2),
		config:            config,
		retentionPolicies: make(map[string]*MetricRetentionPolicy),
		thresholds:        make(map[string]*MetricThreshold),
		ctx:               collectorCtx,
		cancel:            cancel,
		metricBuffer:      make([]*PerformanceMetricV2, 0, config.BufferSize),
		bufferSize:        config.BufferSize,
		flushInterval:     config.FlushInterval,
		batchSize:         config.BatchSize,
		exporters:         make([]MetricExporter, 0),
		exportInterval:    1 * time.Minute,
		healthMetrics:     make(map[string]float64),
		enabled:           true,
		lastFlush:         time.Now(),
	}

	// Initialize advanced features
	if config.AnomalyDetection {
		mc.anomalyDetector = NewAnomalyDetector(0.95, 30)
	}

	if config.TrendAnalysis {
		mc.trendAnalyzer = NewTrendAnalyzer(10*time.Minute, 0.7)
	}

	if config.CorrelationAnalysis {
		mc.correlationEngine = NewCorrelationEngine(0.5, 5*time.Minute)
	}

	mc.alertingEngine = NewAlertingEngine()

	// Start background operations
	mc.startBackgroundOperations()

	return mc
}

// RecordMetric records a new metric with enhanced metadata
func (mc *MetricsCollectorV2) RecordMetric(metric *PerformanceMetricV2) {
	if !mc.enabled {
		return
	}

	start := time.Now()
	defer func() {
		atomic.AddInt64(&mc.totalLatency, int64(time.Since(start)))
		atomic.AddInt64(&mc.operationCount, 1)
	}()

	// Validate and enrich metric
	mc.enrichMetric(metric)

	// Apply sampling if configured
	if mc.config.SamplingRate < 1.0 && mc.shouldSample() {
		return
	}

	// Check for anomalies
	if mc.anomalyDetector != nil {
		isAnomaly, score := mc.anomalyDetector.DetectAnomaly(metric.Name, metric.Value)
		metric.IsAnomaly = isAnomaly
		metric.AnomalyScore = score
	}

	// Add to buffer for batch processing
	mc.bufferMutex.Lock()
	mc.metricBuffer = append(mc.metricBuffer, metric)

	// Flush if buffer is full
	if len(mc.metricBuffer) >= mc.batchSize {
		mc.flushBuffer()
	}
	mc.bufferMutex.Unlock()

	// Update current metrics
	mc.metricsMutex.Lock()
	mc.metrics[metric.Name] = metric
	mc.metricsMutex.Unlock()

	// Add to time series
	mc.addToSeries(metric)

	// Update aggregations
	mc.updateAggregations(metric)

	// Check alert rules
	if mc.alertingEngine != nil {
		mc.alertingEngine.EvaluateMetric(metric)
	}
}

// GetMetric retrieves a specific metric by name
func (mc *MetricsCollectorV2) GetMetric(name string) (*PerformanceMetricV2, bool) {
	mc.metricsMutex.RLock()
	defer mc.metricsMutex.RUnlock()

	metric, exists := mc.metrics[name]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent modification
	metricCopy := *metric
	return &metricCopy, true
}

// GetMetrics returns all current metrics
func (mc *MetricsCollectorV2) GetMetrics() map[string]*PerformanceMetricV2 {
	mc.metricsMutex.RLock()
	defer mc.metricsMutex.RUnlock()

	result := make(map[string]*PerformanceMetricV2)
	for name, metric := range mc.metrics {
		metricCopy := *metric
		result[name] = &metricCopy
	}

	return result
}

// GetMetricSeries returns the time series for a specific metric
func (mc *MetricsCollectorV2) GetMetricSeries(name string, duration time.Duration) (*MetricSeries, bool) {
	mc.seriesMutex.RLock()
	defer mc.seriesMutex.RUnlock()

	series, exists := mc.metricSeries[name]
	if !exists {
		return nil, false
	}

	// Filter points by duration
	cutoff := time.Now().Add(-duration)
	filteredSeries := &MetricSeries{
		MetricName:      series.MetricName,
		Points:          []*MetricDataPoint{},
		Aggregation:     series.Aggregation,
		Resolution:      series.Resolution,
		RetentionPolicy: series.RetentionPolicy,
	}

	for _, point := range series.Points {
		if point.Timestamp.After(cutoff) {
			filteredSeries.Points = append(filteredSeries.Points, point)
		}
	}

	// Calculate statistics for filtered series
	filteredSeries.Statistics = mc.calculateSeriesStatistics(filteredSeries.Points)

	return filteredSeries, true
}

// GetAggregatedMetrics returns aggregated metrics for a specific time window
func (mc *MetricsCollectorV2) GetAggregatedMetrics(metricName string, aggregationType MetricAggregationType, window time.Duration) (*PerformanceMetricV2, error) {
	series, exists := mc.GetMetricSeries(metricName, window)
	if !exists {
		return nil, errors.New("metric series not found: " + metricName)
	}

	if len(series.Points) == 0 {
		return nil, errors.New("no data points in series for metric: " + metricName)
	}

	aggregatedValue, err := mc.aggregateValues(series.Points, aggregationType)
	if err != nil {
		return nil, err
	}

	// Create aggregated metric
	aggregatedMetric := &PerformanceMetricV2{
		ID:             fmt.Sprintf("%s_%s_%d", metricName, aggregationType, time.Now().Unix()),
		Name:           fmt.Sprintf("%s_%s", metricName, aggregationType),
		Category:       CategoryApplication,
		Type:           aggregationType,
		Value:          aggregatedValue,
		Timestamp:      time.Now(),
		WindowDuration: window,
		SampleCount:    int64(len(series.Points)),
		Tags:           map[string]string{"aggregation": string(aggregationType)},
		Metadata:       map[string]interface{}{"window_duration": window.String()},
	}

	return aggregatedMetric, nil
}

// GetPerformanceReport generates a comprehensive performance report
func (mc *MetricsCollectorV2) GetPerformanceReport() map[string]interface{} {
	report := make(map[string]interface{})

	// Basic statistics
	report["total_metrics"] = len(mc.metrics)
	report["total_series"] = len(mc.metricSeries)
	report["operation_count"] = atomic.LoadInt64(&mc.operationCount)
	report["error_count"] = atomic.LoadInt64(&mc.errorCount)
	report["last_flush"] = mc.lastFlush

	// Calculate average latency
	avgLatency := float64(0)
	if opCount := atomic.LoadInt64(&mc.operationCount); opCount > 0 {
		avgLatency = float64(atomic.LoadInt64(&mc.totalLatency)) / float64(opCount)
	}
	report["avg_operation_latency_ns"] = avgLatency

	// Health metrics
	mc.healthMutex.RLock()
	report["health_metrics"] = make(map[string]float64)
	for k, v := range mc.healthMetrics {
		report["health_metrics"].(map[string]float64)[k] = v
	}
	mc.healthMutex.RUnlock()

	// Configuration
	report["config"] = mc.config

	// Anomaly detection stats
	if mc.anomalyDetector != nil {
		report["anomaly_detection"] = mc.anomalyDetector.GetStats()
	}

	// Trend analysis stats
	if mc.trendAnalyzer != nil {
		report["trend_analysis"] = mc.trendAnalyzer.GetStats()
	}

	// Correlation analysis stats
	if mc.correlationEngine != nil {
		report["correlation_analysis"] = mc.correlationEngine.GetStats()
	}

	// Alerting stats
	if mc.alertingEngine != nil {
		report["alerting"] = mc.alertingEngine.GetStats()
	}

	return report
}

// AddThreshold adds a threshold configuration for a metric
func (mc *MetricsCollectorV2) AddThreshold(metricName string, threshold *MetricThreshold) {
	mc.metricsMutex.Lock()
	defer mc.metricsMutex.Unlock()

	mc.thresholds[metricName] = threshold
}

// AddAlertRule adds an alerting rule
func (mc *MetricsCollectorV2) AddAlertRule(rule *AlertRule) {
	if mc.alertingEngine != nil {
		mc.alertingEngine.AddRule(rule)
	}
}

// AddExporter adds a metric exporter
func (mc *MetricsCollectorV2) AddExporter(exporter MetricExporter) {
	mc.exporters = append(mc.exporters, exporter)
}

// GetAnomalies returns detected anomalies in the specified time window
func (mc *MetricsCollectorV2) GetAnomalies(window time.Duration) []*PerformanceMetricV2 {
	anomalies := []*PerformanceMetricV2{}
	cutoff := time.Now().Add(-window)

	mc.metricsMutex.RLock()
	defer mc.metricsMutex.RUnlock()

	for _, metric := range mc.metrics {
		if metric.IsAnomaly && metric.Timestamp.After(cutoff) {
			anomalyCopy := *metric
			anomalies = append(anomalies, &anomalyCopy)
		}
	}

	return anomalies
}

// GetTrends returns trend analysis for all metrics
func (mc *MetricsCollectorV2) GetTrends() map[string]*TrendModel {
	if mc.trendAnalyzer == nil {
		return nil
	}

	return mc.trendAnalyzer.GetTrends()
}

// GetCorrelations returns correlation analysis between metrics
func (mc *MetricsCollectorV2) GetCorrelations() map[string]map[string]float64 {
	if mc.correlationEngine == nil {
		return nil
	}

	return mc.correlationEngine.GetCorrelations()
}

// GetActiveAlerts returns currently active alerts
func (mc *MetricsCollectorV2) GetActiveAlerts() []*Alert {
	if mc.alertingEngine == nil {
		return nil
	}

	return mc.alertingEngine.GetActiveAlerts()
}

// Helper methods

func (mc *MetricsCollectorV2) enrichMetric(metric *PerformanceMetricV2) {
	if metric.ID == "" {
		metric.ID = fmt.Sprintf("%s_%d", metric.Name, time.Now().UnixNano())
	}

	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	if metric.Tags == nil {
		metric.Tags = make(map[string]string)
	}

	if metric.Labels == nil {
		metric.Labels = make(map[string]string)
	}

	if metric.Metadata == nil {
		metric.Metadata = make(map[string]interface{})
	}

	// Add default tags
	for k, v := range mc.config.DefaultTags {
		if _, exists := metric.Tags[k]; !exists {
			metric.Tags[k] = v
		}
	}

	// Set environment and component if not specified
	if metric.Environment == "" {
		metric.Environment = getEnvString("MCP_ENVIRONMENT", "development")
	}

	if metric.Component == "" {
		metric.Component = "mcp-memory"
	}

	// Calculate cardinality
	metric.Cardinality = len(metric.Tags) + len(metric.Labels)
}

func (mc *MetricsCollectorV2) shouldSample() bool {
	// Simple random sampling
	return false // Placeholder - implement actual sampling logic
}

func (mc *MetricsCollectorV2) addToSeries(metric *PerformanceMetricV2) {
	mc.seriesMutex.Lock()
	defer mc.seriesMutex.Unlock()

	series, exists := mc.metricSeries[metric.Name]
	if !exists {
		series = &MetricSeries{
			MetricName:  metric.Name,
			Points:      []*MetricDataPoint{},
			Aggregation: AggregationGauge,
			Resolution:  1 * time.Minute,
			RetentionPolicy: &MetricRetentionPolicy{
				RawDataRetention:        24 * time.Hour,
				AggregatedDataRetention: 7 * 24 * time.Hour,
				CompressionThreshold:    1 * time.Hour,
				CompressionRatio:        0.5,
			},
		}
		mc.metricSeries[metric.Name] = series
	}

	// Add new data point
	dataPoint := &MetricDataPoint{
		Timestamp: metric.Timestamp,
		Value:     metric.Value,
		Tags:      metric.Tags,
		Metadata:  metric.Metadata,
		Quality:   1.0, // Perfect quality by default
	}

	series.mutex.Lock()
	series.Points = append(series.Points, dataPoint)

	// Limit series length
	if len(series.Points) > mc.config.MaxSeriesLength {
		series.Points = series.Points[len(series.Points)-mc.config.MaxSeriesLength:]
	}

	// Update statistics
	series.Statistics = mc.calculateSeriesStatistics(series.Points)
	series.mutex.Unlock()
}

func (mc *MetricsCollectorV2) updateAggregations(metric *PerformanceMetricV2) {
	mc.aggregatedMutex.Lock()
	defer mc.aggregatedMutex.Unlock()

	if mc.aggregatedMetrics[metric.Name] == nil {
		mc.aggregatedMetrics[metric.Name] = make(map[MetricAggregationType]*PerformanceMetricV2)
	}

	aggregations := []MetricAggregationType{AggregationSum, AggregationAverage, AggregationMin, AggregationMax}

	for _, aggType := range aggregations {
		mc.updateSingleAggregation(metric, aggType)
	}
}

// updateSingleAggregation updates a single aggregation type for a metric
func (mc *MetricsCollectorV2) updateSingleAggregation(metric *PerformanceMetricV2, aggType MetricAggregationType) {
	existing := mc.aggregatedMetrics[metric.Name][aggType]
	if existing == nil {
		existing = mc.createNewAggregation(metric, aggType)
	} else {
		mc.updateExistingAggregation(existing, metric, aggType)
	}

	mc.aggregatedMetrics[metric.Name][aggType] = existing
}

// createNewAggregation creates a new aggregation entry
func (mc *MetricsCollectorV2) createNewAggregation(metric *PerformanceMetricV2, aggType MetricAggregationType) *PerformanceMetricV2 {
	return &PerformanceMetricV2{
		Name:        fmt.Sprintf("%s_%s", metric.Name, aggType),
		Category:    metric.Category,
		Type:        aggType,
		Value:       metric.Value,
		Unit:        metric.Unit,
		Timestamp:   metric.Timestamp,
		SampleCount: 1,
	}
}

// updateExistingAggregation updates an existing aggregation with a new metric value
func (mc *MetricsCollectorV2) updateExistingAggregation(existing, metric *PerformanceMetricV2, aggType MetricAggregationType) {
	switch aggType {
	case AggregationSum:
		existing.Value += metric.Value
	case AggregationAverage:
		existing.Value = (existing.Value*float64(existing.SampleCount) + metric.Value) / float64(existing.SampleCount+1)
	case AggregationMin:
		if metric.Value < existing.Value {
			existing.Value = metric.Value
		}
	case AggregationMax:
		if metric.Value > existing.Value {
			existing.Value = metric.Value
		}
	default:
		mc.updateAdvancedAggregation(existing, metric, aggType)
	}
	existing.SampleCount++
	existing.Timestamp = metric.Timestamp
}

// updateAdvancedAggregation handles complex aggregation types
func (mc *MetricsCollectorV2) updateAdvancedAggregation(existing, metric *PerformanceMetricV2, aggType MetricAggregationType) {
	switch aggType {
	case AggregationMedian:
		// For median, we would need to store all values - simplified for now
		existing.Value = metric.Value
	case AggregationP95:
		// For P95, we would need to store all values - simplified for now
		existing.Value = metric.Value
	case AggregationP99:
		// For P99, we would need to store all values - simplified for now
		existing.Value = metric.Value
	case AggregationRate:
		// Rate calculation would need time series data - simplified for now
		existing.Value = metric.Value
	case AggregationGauge:
		// Gauge just takes the latest value
		existing.Value = metric.Value
	case AggregationHistogram:
		// Histogram would need bucket data - simplified for now
		existing.Value = metric.Value
	}
}

func (mc *MetricsCollectorV2) calculateSeriesStatistics(points []*MetricDataPoint) *SeriesStatistics {
	if len(points) == 0 {
		return &SeriesStatistics{LastUpdated: time.Now()}
	}

	values := make([]float64, len(points))
	for i, point := range points {
		values[i] = point.Value
	}

	sort.Float64s(values)

	stats := &SeriesStatistics{
		Count:       int64(len(values)),
		Min:         values[0],
		Max:         values[len(values)-1],
		Range:       values[len(values)-1] - values[0],
		FirstValue:  points[0].Value,
		LastValue:   points[len(points)-1].Value,
		LastUpdated: time.Now(),
	}

	// Calculate sum and mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	stats.Sum = sum
	stats.Mean = sum / float64(len(values))

	// Calculate median
	if len(values)%2 == 0 {
		stats.Median = (values[len(values)/2-1] + values[len(values)/2]) / 2
	} else {
		stats.Median = values[len(values)/2]
	}

	// Calculate percentiles
	stats.P50 = percentileValue(values, 0.5)
	stats.P90 = percentileValue(values, 0.9)
	stats.P95 = percentileValue(values, 0.95)
	stats.P99 = percentileValue(values, 0.99)
	stats.P999 = percentileValue(values, 0.999)

	// Calculate variance and standard deviation
	variance := 0.0
	for _, v := range values {
		diff := v - stats.Mean
		variance += diff * diff
	}
	stats.Variance = variance / float64(len(values))
	stats.StdDev = math.Sqrt(stats.Variance)

	// Calculate skewness and kurtosis (simplified)
	stats.Skewness = calculateSkewness(values, stats.Mean, stats.StdDev)
	stats.Kurtosis = calculateKurtosis(values, stats.Mean, stats.StdDev)

	return stats
}

func (mc *MetricsCollectorV2) aggregateValues(points []*MetricDataPoint, aggregationType MetricAggregationType) (float64, error) {
	if len(points) == 0 {
		return 0, errors.New("no data points to aggregate")
	}

	values := mc.extractValues(points)
	return mc.computeAggregation(values, aggregationType)
}

// extractValues extracts numeric values from metric data points
func (mc *MetricsCollectorV2) extractValues(points []*MetricDataPoint) []float64 {
	values := make([]float64, len(points))
	for i, point := range points {
		values[i] = point.Value
	}
	return values
}

// computeAggregation computes the aggregated value based on type
func (mc *MetricsCollectorV2) computeAggregation(values []float64, aggregationType MetricAggregationType) (float64, error) {
	switch aggregationType {
	case AggregationSum, AggregationAverage:
		return mc.computeSumOrAverage(values, aggregationType)
	case AggregationMin, AggregationMax:
		return mc.computeMinOrMax(values, aggregationType)
	case AggregationMedian:
		return mc.computeMedian(values)
	case AggregationP95:
		return percentileValue(values, 0.95), nil
	case AggregationP99:
		return percentileValue(values, 0.99), nil
	case AggregationRate:
		return mc.computeRate(values)
	case AggregationGauge:
		return values[len(values)-1], nil
	case AggregationHistogram:
		return float64(len(values)), nil
	default:
		return 0, errors.New("unsupported aggregation type: " + string(aggregationType))
	}
}

// computeSumOrAverage calculates sum or average
func (mc *MetricsCollectorV2) computeSumOrAverage(values []float64, aggregationType MetricAggregationType) (float64, error) {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	if aggregationType == AggregationSum {
		return sum, nil
	}
	return sum / float64(len(values)), nil
}

// computeMinOrMax finds minimum or maximum value
func (mc *MetricsCollectorV2) computeMinOrMax(values []float64, aggregationType MetricAggregationType) (float64, error) {
	result := values[0]
	for _, v := range values[1:] {
		if aggregationType == AggregationMin && v < result {
			result = v
		} else if aggregationType == AggregationMax && v > result {
			result = v
		}
	}
	return result, nil
}

// computeMedian calculates the median value
func (mc *MetricsCollectorV2) computeMedian(values []float64) (float64, error) {
	sort.Float64s(values)
	if len(values)%2 == 0 {
		return (values[len(values)/2-1] + values[len(values)/2]) / 2, nil
	}
	return values[len(values)/2], nil
}

// computeRate calculates the rate of change
func (mc *MetricsCollectorV2) computeRate(values []float64) (float64, error) {
	if len(values) < 2 {
		return 0, nil
	}
	return (values[len(values)-1] - values[0]) / float64(len(values)-1), nil
}

func (mc *MetricsCollectorV2) flushBuffer() {
	if len(mc.metricBuffer) == 0 {
		return
	}

	// Export metrics if exporters are configured
	for _, exporter := range mc.exporters {
		if exporter.IsHealthy() {
			go func(exp MetricExporter, metrics []*PerformanceMetricV2) {
				if err := exp.Export(metrics); err != nil {
					atomic.AddInt64(&mc.errorCount, 1)
				}
			}(exporter, mc.metricBuffer)
		}
	}

	// Clear buffer
	mc.metricBuffer = mc.metricBuffer[:0]
	mc.lastFlush = time.Now()
}

func (mc *MetricsCollectorV2) startBackgroundOperations() {
	// Periodic buffer flush
	mc.backgroundWG.Add(1)
	go mc.bufferFlushLoop()

	// Cleanup old data
	mc.backgroundWG.Add(1)
	go mc.cleanupLoop()

	// Update health metrics
	mc.backgroundWG.Add(1)
	go mc.healthMetricsLoop()

	// Analysis loops
	if mc.anomalyDetector != nil {
		mc.backgroundWG.Add(1)
		go mc.anomalyDetectionLoop()
	}

	if mc.trendAnalyzer != nil {
		mc.backgroundWG.Add(1)
		go mc.trendAnalysisLoop()
	}

	if mc.correlationEngine != nil {
		mc.backgroundWG.Add(1)
		go mc.correlationAnalysisLoop()
	}
}

func (mc *MetricsCollectorV2) bufferFlushLoop() {
	defer mc.backgroundWG.Done()

	ticker := time.NewTicker(mc.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.bufferMutex.Lock()
			mc.flushBuffer()
			mc.bufferMutex.Unlock()
		}
	}
}

func (mc *MetricsCollectorV2) cleanupLoop() {
	defer mc.backgroundWG.Done()

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.cleanupOldData()
		}
	}
}

func (mc *MetricsCollectorV2) healthMetricsLoop() {
	defer mc.backgroundWG.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.updateHealthMetrics()
		}
	}
}

func (mc *MetricsCollectorV2) anomalyDetectionLoop() {
	defer mc.backgroundWG.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.anomalyDetector.AnalyzeAll(mc.metricSeries)
		}
	}
}

func (mc *MetricsCollectorV2) trendAnalysisLoop() {
	defer mc.backgroundWG.Done()

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.trendAnalyzer.AnalyzeAll(mc.metricSeries)
		}
	}
}

func (mc *MetricsCollectorV2) correlationAnalysisLoop() {
	defer mc.backgroundWG.Done()

	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mc.ctx.Done():
			return
		case <-ticker.C:
			mc.correlationEngine.AnalyzeAll(mc.metricSeries)
		}
	}
}

func (mc *MetricsCollectorV2) cleanupOldData() {
	cutoff := time.Now().Add(-mc.config.RetentionDuration)

	// Cleanup metrics
	mc.metricsMutex.Lock()
	for name, metric := range mc.metrics {
		if metric.Timestamp.Before(cutoff) {
			delete(mc.metrics, name)
		}
	}
	mc.metricsMutex.Unlock()

	// Cleanup series data
	mc.seriesMutex.Lock()
	for _, series := range mc.metricSeries {
		series.mutex.Lock()
		filteredPoints := []*MetricDataPoint{}
		for _, point := range series.Points {
			if point.Timestamp.After(cutoff) {
				filteredPoints = append(filteredPoints, point)
			}
		}
		series.Points = filteredPoints
		series.mutex.Unlock()
	}
	mc.seriesMutex.Unlock()
}

func (mc *MetricsCollectorV2) updateHealthMetrics() {
	mc.healthMutex.Lock()
	defer mc.healthMutex.Unlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	mc.healthMetrics["memory_used_bytes"] = float64(memStats.Alloc)
	mc.healthMetrics["memory_total_bytes"] = float64(memStats.Sys)
	mc.healthMetrics["goroutines"] = float64(runtime.NumGoroutine())
	mc.healthMetrics["gc_pauses_ms"] = float64(memStats.PauseTotalNs) / 1e6
	mc.healthMetrics["metrics_count"] = float64(len(mc.metrics))
	mc.healthMetrics["series_count"] = float64(len(mc.metricSeries))
	mc.healthMetrics["operations_per_sec"] = float64(atomic.LoadInt64(&mc.operationCount)) / time.Since(mc.lastFlush).Seconds()
}

// Shutdown gracefully shuts down the metrics collector
func (mc *MetricsCollectorV2) Shutdown() error {
	mc.cancel()
	mc.backgroundWG.Wait()

	// Final flush
	mc.bufferMutex.Lock()
	mc.flushBuffer()
	mc.bufferMutex.Unlock()

	return nil
}

// Utility functions

func getDefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		CollectionInterval:   30 * time.Second,
		RetentionDuration:    24 * time.Hour,
		MaxMetrics:           10000,
		MaxSeriesLength:      1000,
		CompressionEnabled:   true,
		AnomalyDetection:     true,
		TrendAnalysis:        true,
		CorrelationAnalysis:  true,
		ExportEnabled:        true,
		BufferSize:           1000,
		FlushInterval:        10 * time.Second,
		BatchSize:            100,
		SamplingRate:         1.0,
		HighCardinalityLimit: 1000,
		DefaultTags:          map[string]string{"service": "mcp-memory"},
	}
}

func percentileValue(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}

	index := percentile * float64(len(sortedValues)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sortedValues) {
		return sortedValues[len(sortedValues)-1]
	}

	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

func calculateSkewness(values []float64, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		normalized := (v - mean) / stdDev
		sum += normalized * normalized * normalized
	}

	return sum / float64(len(values))
}

func calculateKurtosis(values []float64, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		normalized := (v - mean) / stdDev
		square := normalized * normalized
		sum += square * square
	}

	return (sum / float64(len(values))) - 3 // Excess kurtosis
}

func getEnvString(key, defaultValue string) string {
	// Placeholder - in production, read from environment
	return defaultValue
}

// Placeholder implementations for analysis engines

func NewAnomalyDetector(threshold float64, minDataPoints int) *AnomalyDetector {
	return &AnomalyDetector{
		models:           make(map[string]*AnomalyModel),
		threshold:        threshold,
		minDataPoints:    minDataPoints,
		learningEnabled:  true,
		sensitivityLevel: 0.8,
	}
}

func (ad *AnomalyDetector) DetectAnomaly(metricName string, value float64) (isAnomaly bool, anomalyScore float64) {
	// Placeholder implementation
	return false, 0.0
}

func (ad *AnomalyDetector) AnalyzeAll(series map[string]*MetricSeries) {
	// Placeholder implementation
}

func (ad *AnomalyDetector) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"models_count": len(ad.models),
		"threshold":    ad.threshold,
		"enabled":      ad.learningEnabled,
	}
}

func NewTrendAnalyzer(windowSize time.Duration, sensitivityLevel float64) *TrendAnalyzer {
	return &TrendAnalyzer{
		trendModels:      make(map[string]*TrendModel),
		windowSize:       windowSize,
		sensitivityLevel: sensitivityLevel,
		enabled:          true,
	}
}

func (ta *TrendAnalyzer) AnalyzeAll(series map[string]*MetricSeries) {
	// Placeholder implementation
}

func (ta *TrendAnalyzer) GetTrends() map[string]*TrendModel {
	ta.mutex.RLock()
	defer ta.mutex.RUnlock()

	result := make(map[string]*TrendModel)
	for k, v := range ta.trendModels {
		result[k] = v
	}

	return result
}

func (ta *TrendAnalyzer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"models_count":      len(ta.trendModels),
		"window_size":       ta.windowSize.String(),
		"sensitivity_level": ta.sensitivityLevel,
		"enabled":           ta.enabled,
	}
}

func NewCorrelationEngine(minCorrelation float64, windowSize time.Duration) *CorrelationEngine {
	return &CorrelationEngine{
		correlations:   make(map[string]map[string]float64),
		minCorrelation: minCorrelation,
		windowSize:     windowSize,
		enabled:        true,
	}
}

func (ce *CorrelationEngine) AnalyzeAll(series map[string]*MetricSeries) {
	// Placeholder implementation
}

func (ce *CorrelationEngine) GetCorrelations() map[string]map[string]float64 {
	ce.mutex.RLock()
	defer ce.mutex.RUnlock()

	result := make(map[string]map[string]float64)
	for k, v := range ce.correlations {
		result[k] = make(map[string]float64)
		for k2, v2 := range v {
			result[k][k2] = v2
		}
	}

	return result
}

func (ce *CorrelationEngine) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"correlations_count": len(ce.correlations),
		"min_correlation":    ce.minCorrelation,
		"window_size":        ce.windowSize.String(),
		"enabled":            ce.enabled,
	}
}

func NewAlertingEngine() *AlertingEngine {
	return &AlertingEngine{
		rules:           make(map[string]*AlertRule),
		alertHistory:    make([]*Alert, 0),
		cooldownPeriods: make(map[string]time.Time),
		enabled:         true,
	}
}

func (ae *AlertingEngine) AddRule(rule *AlertRule) {
	ae.mutex.Lock()
	defer ae.mutex.Unlock()

	ae.rules[rule.ID] = rule
}

func (ae *AlertingEngine) EvaluateMetric(metric *PerformanceMetricV2) {
	// Placeholder implementation
}

func (ae *AlertingEngine) GetActiveAlerts() []*Alert {
	ae.mutex.RLock()
	defer ae.mutex.RUnlock()

	activeAlerts := []*Alert{}
	for _, alert := range ae.alertHistory {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	return activeAlerts
}

func (ae *AlertingEngine) GetStats() map[string]interface{} {
	ae.mutex.RLock()
	defer ae.mutex.RUnlock()

	activeCount := 0
	for _, alert := range ae.alertHistory {
		if !alert.Resolved {
			activeCount++
		}
	}

	return map[string]interface{}{
		"rules_count":   len(ae.rules),
		"total_alerts":  len(ae.alertHistory),
		"active_alerts": activeCount,
		"enabled":       ae.enabled,
	}
}
