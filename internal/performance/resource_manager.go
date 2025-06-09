// Package performance provides performance optimization and monitoring utilities.
// It includes caching, metrics collection, query optimization, and resource management.
package performance

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// ResourceType represents different types of managed resources
type ResourceType string

const (
	ResourceTypeDatabase  ResourceType = "database"
	ResourceTypeVector    ResourceType = "vector"
	ResourceTypeHTTP      ResourceType = "http"
	ResourceTypeGRPC      ResourceType = "grpc"
	ResourceTypeRedis     ResourceType = "redis"
	ResourceTypeEmbedding ResourceType = "embedding"
	ResourceTypeFile      ResourceType = "file"
	ResourceTypeMemory    ResourceType = "memory"
	ResourceTypeWorker    ResourceType = "worker"
	ResourceTypeGeneric   ResourceType = "generic"
)

// ResourceStatus represents the status of a resource
type ResourceStatus string

const (
	StatusAvailable   ResourceStatus = "available"
	StatusInUse       ResourceStatus = "in_use"
	StatusMaintenance ResourceStatus = "maintenance"
	StatusError       ResourceStatus = "error"
	StatusRetiring    ResourceStatus = "retiring"
	StatusClosed      ResourceStatus = "closed"
)

// Resource represents a managed resource with lifecycle tracking
type Resource struct {
	ID          string                 `json:"id"`
	Type        ResourceType           `json:"type"`
	Status      ResourceStatus         `json:"status"`
	Connection  interface{}            `json:"-"` // Actual resource connection
	CreatedAt   time.Time              `json:"created_at"`
	LastUsed    time.Time              `json:"last_used"`
	UsageCount  int64                  `json:"usage_count"`
	ErrorCount  int64                  `json:"error_count"`
	MaxUsage    int64                  `json:"max_usage"`
	IdleTimeout time.Duration          `json:"idle_timeout"`
	MaxLifetime time.Duration          `json:"max_lifetime"`
	HealthCheck func() error           `json:"-"`
	Cleanup     func() error           `json:"-"`
	Metadata    map[string]interface{} `json:"metadata"`
	Tags        map[string]string      `json:"tags"`
	Priority    int                    `json:"priority"`
	Weight      float64                `json:"weight"`
	mutex       sync.RWMutex
}

// ResourcePool manages a pool of resources with advanced features
type ResourcePool struct {
	Name                string        `json:"name"`
	Type                ResourceType  `json:"type"`
	MinSize             int           `json:"min_size"`
	MaxSize             int           `json:"max_size"`
	CurrentSize         int           `json:"current_size"`
	IdleTimeout         time.Duration `json:"idle_timeout"`
	MaxLifetime         time.Duration `json:"max_lifetime"`
	AcquisitionTimeout  time.Duration `json:"acquisition_timeout"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`

	// Resource management
	resources []*Resource
	available chan *Resource
	factory   ResourceFactory
	validator ResourceValidator

	// Statistics and monitoring
	statistics *PoolStatistics

	// Configuration
	loadBalancer   LoadBalancer
	retryPolicy    *RetryPolicy
	circuitBreaker *CircuitBreaker

	// Synchronization
	mutex      sync.RWMutex
	statsMutex sync.RWMutex

	// Background operations
	ctx          context.Context
	cancel       context.CancelFunc
	backgroundWG sync.WaitGroup

	// Advanced features
	preWarming      bool
	adaptiveScaling bool
	metricsEnabled  bool
	tracingEnabled  bool

	enabled bool
}

// PoolStatistics holds comprehensive pool statistics
type PoolStatistics struct {
	TotalCreated           int64         `json:"total_created"`
	TotalDestroyed         int64         `json:"total_destroyed"`
	TotalAcquired          int64         `json:"total_acquired"`
	TotalReleased          int64         `json:"total_released"`
	TotalErrors            int64         `json:"total_errors"`
	TotalTimeouts          int64         `json:"total_timeouts"`
	ActiveConnections      int64         `json:"active_connections"`
	IdleConnections        int64         `json:"idle_connections"`
	PeakConnections        int64         `json:"peak_connections"`
	AverageAcquisitionTime time.Duration `json:"average_acquisition_time"`
	AverageUsageTime       time.Duration `json:"average_usage_time"`
	SuccessRate            float64       `json:"success_rate"`
	ErrorRate              float64       `json:"error_rate"`
	UtilizationRate        float64       `json:"utilization_rate"`
	HealthCheckSuccessRate float64       `json:"health_check_success_rate"`
	LastHealthCheck        time.Time     `json:"last_health_check"`
	WindowStart            time.Time     `json:"window_start"`
	WindowDuration         time.Duration `json:"window_duration"`
}

// ResourceFactory creates new resources
type ResourceFactory interface {
	CreateResource(ctx context.Context, config map[string]interface{}) (*Resource, error)
	ValidateConfig(config map[string]interface{}) error
	GetResourceType() ResourceType
	GetDefaultConfig() map[string]interface{}
}

// ResourceValidator validates resource health and usability
type ResourceValidator interface {
	ValidateResource(resource *Resource) error
	IsHealthy(resource *Resource) bool
	GetValidationTimeout() time.Duration
}

// LoadBalancer distributes load across available resources
type LoadBalancer interface {
	SelectResource(resources []*Resource, criteria map[string]interface{}) (*Resource, error)
	GetAlgorithm() string
	UpdateWeights(resources []*Resource, metrics map[string]float64)
}

// CircuitBreaker protects against cascading failures
type CircuitBreaker struct {
	maxFailures     int
	resetTimeout    time.Duration
	failureCount    int64
	lastFailureTime time.Time
	state           CircuitBreakerState
	mutex           sync.RWMutex
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitClosed   CircuitBreakerState = "closed"
	CircuitOpen     CircuitBreakerState = "open"
	CircuitHalfOpen CircuitBreakerState = "half_open"
)

// ResourceManager coordinates multiple resource pools
type ResourceManager struct {
	pools       map[string]*ResourcePool
	globalStats *GlobalResourceStatistics
	config      *ResourceManagerConfig

	// Monitoring and alerting
	healthMonitor  *ResourceHealthMonitor
	alertingEngine *ResourceAlertingEngine

	// Advanced features
	autoScaler      *ResourceAutoScaler
	failoverManager *FailoverManager

	// Synchronization
	mutex sync.RWMutex

	// Background operations
	ctx          context.Context
	cancel       context.CancelFunc
	backgroundWG sync.WaitGroup

	enabled bool
}

// ResourceManagerConfig holds configuration for the resource manager
type ResourceManagerConfig struct {
	GlobalMaxResources    int           `json:"global_max_resources"`
	GlobalIdleTimeout     time.Duration `json:"global_idle_timeout"`
	GlobalMaxLifetime     time.Duration `json:"global_max_lifetime"`
	HealthCheckInterval   time.Duration `json:"health_check_interval"`
	MetricsInterval       time.Duration `json:"metrics_interval"`
	CleanupInterval       time.Duration `json:"cleanup_interval"`
	AutoScalingEnabled    bool          `json:"auto_scaling_enabled"`
	LoadBalancingEnabled  bool          `json:"load_balancing_enabled"`
	FailoverEnabled       bool          `json:"failover_enabled"`
	CircuitBreakerEnabled bool          `json:"circuit_breaker_enabled"`
	TracingEnabled        bool          `json:"tracing_enabled"`
	MetricsEnabled        bool          `json:"metrics_enabled"`
	DefaultRetryAttempts  int           `json:"default_retry_attempts"`
	DefaultRetryBackoff   time.Duration `json:"default_retry_backoff"`
}

// GlobalResourceStatistics aggregates statistics across all pools
type GlobalResourceStatistics struct {
	TotalPools             int       `json:"total_pools"`
	TotalResources         int       `json:"total_resources"`
	TotalActiveResources   int       `json:"total_active_resources"`
	TotalIdleResources     int       `json:"total_idle_resources"`
	GlobalUtilizationRate  float64   `json:"global_utilization_rate"`
	GlobalSuccessRate      float64   `json:"global_success_rate"`
	GlobalErrorRate        float64   `json:"global_error_rate"`
	AveragePoolUtilization float64   `json:"average_pool_utilization"`
	LastUpdated            time.Time `json:"last_updated"`
}

// ResourceHealthMonitor monitors the health of resources and pools
type ResourceHealthMonitor struct {
	healthChecks    map[string]*HealthCheck
	alertThresholds map[string]*HealthThreshold
	enabled         bool
}

// HealthCheck represents a health check configuration
type HealthCheck struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	Retries     int           `json:"retries"`
	Enabled     bool          `json:"enabled"`
	LastRun     time.Time     `json:"last_run"`
	LastResult  bool          `json:"last_result"`
	SuccessRate float64       `json:"success_rate"`
}

// HealthThreshold defines thresholds for health alerts
type HealthThreshold struct {
	MetricName    string  `json:"metric_name"`
	WarningLevel  float64 `json:"warning_level"`
	CriticalLevel float64 `json:"critical_level"`
	Operator      string  `json:"operator"` // "gt", "lt", "eq"
}

// ResourceAlertingEngine handles resource-related alerts
type ResourceAlertingEngine struct {
	rules     map[string]*ResourceAlertRule
	alerts    []*ResourceAlert
	callbacks []func(*ResourceAlert)
	enabled   bool
	mutex     sync.RWMutex
}

// ResourceAlertRule defines an alerting rule for resources
type ResourceAlertRule struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Condition string                 `json:"condition"`
	Threshold float64                `json:"threshold"`
	Duration  time.Duration          `json:"duration"`
	Severity  string                 `json:"severity"`
	Tags      map[string]string      `json:"tags"`
	Metadata  map[string]interface{} `json:"metadata"`
	Enabled   bool                   `json:"enabled"`
	Cooldown  time.Duration          `json:"cooldown"`
	LastFired time.Time              `json:"last_fired"`
}

// ResourceAlert represents a triggered resource alert
type ResourceAlert struct {
	ID         string                 `json:"id"`
	RuleID     string                 `json:"rule_id"`
	ResourceID string                 `json:"resource_id"`
	PoolName   string                 `json:"pool_name"`
	Severity   string                 `json:"severity"`
	Message    string                 `json:"message"`
	Value      float64                `json:"value"`
	Threshold  float64                `json:"threshold"`
	Timestamp  time.Time              `json:"timestamp"`
	Tags       map[string]string      `json:"tags"`
	Metadata   map[string]interface{} `json:"metadata"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// ResourceAutoScaler handles automatic scaling of resource pools
type ResourceAutoScaler struct {
	policies map[string]*ScalingPolicy
	enabled  bool
	mutex    sync.RWMutex
}

// ScalingPolicy defines auto-scaling behavior
type ScalingPolicy struct {
	PoolName          string        `json:"pool_name"`
	MinSize           int           `json:"min_size"`
	MaxSize           int           `json:"max_size"`
	TargetUtilization float64       `json:"target_utilization"`
	ScaleUpCooldown   time.Duration `json:"scale_up_cooldown"`
	ScaleDownCooldown time.Duration `json:"scale_down_cooldown"`
	ScaleUpStep       int           `json:"scale_up_step"`
	ScaleDownStep     int           `json:"scale_down_step"`
	Enabled           bool          `json:"enabled"`
	LastScaleAction   time.Time     `json:"last_scale_action"`
}

// FailoverManager handles failover between resource pools
type FailoverManager struct {
	primaryPools    map[string]string
	backupPools     map[string][]string
	failoverHistory map[string][]*FailoverEvent
	enabled         bool
}

// FailoverEvent represents a failover event
type FailoverEvent struct {
	ID           string        `json:"id"`
	FromPool     string        `json:"from_pool"`
	ToPool       string        `json:"to_pool"`
	Reason       string        `json:"reason"`
	Timestamp    time.Time     `json:"timestamp"`
	Duration     time.Duration `json:"duration"`
	Success      bool          `json:"success"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// NewResourceManager creates a new resource manager
func NewResourceManager(ctx context.Context, config *ResourceManagerConfig) *ResourceManager {
	if config == nil {
		config = getDefaultResourceManagerConfig()
	}

	rmCtx, cancel := context.WithCancel(ctx)

	rm := &ResourceManager{
		pools:       make(map[string]*ResourcePool),
		globalStats: &GlobalResourceStatistics{LastUpdated: time.Now()},
		config:      config,
		ctx:         rmCtx,
		cancel:      cancel,
		enabled:     true,
	}

	// Initialize components
	// Skip metrics collector for now - needs database connection
	// if config.MetricsEnabled {
	//     rm.metricsCollector = NewMetricsCollectorV2(db, dbConfig, metricsConfig)
	// }

	rm.healthMonitor = NewResourceHealthMonitor()
	rm.alertingEngine = NewResourceAlertingEngine()

	if config.AutoScalingEnabled {
		rm.autoScaler = NewResourceAutoScaler()
	}

	if config.FailoverEnabled {
		rm.failoverManager = NewFailoverManager()
	}

	// Start background operations
	rm.startBackgroundOperations()

	return rm
}

// CreatePool creates a new resource pool
func (rm *ResourceManager) CreatePool(name string, config *ResourcePoolConfig, factory ResourceFactory) (*ResourcePool, error) {
	if !rm.enabled {
		return nil, errors.New("resource manager is disabled")
	}

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	if _, exists := rm.pools[name]; exists {
		return nil, errors.New("pool already exists: " + name)
	}

	pool, err := NewResourcePool(rm.ctx, name, config, factory)
	if err != nil {
		return nil, errors.New("failed to create pool: " + err.Error())
	}

	rm.pools[name] = pool
	rm.updateGlobalStats()

	return pool, nil
}

// GetPool retrieves a resource pool by name
func (rm *ResourceManager) GetPool(name string) (*ResourcePool, bool) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	pool, exists := rm.pools[name]
	return pool, exists
}

// GetResource acquires a resource from the specified pool
func (rm *ResourceManager) GetResource(poolName string, timeout time.Duration) (*Resource, error) {
	pool, exists := rm.GetPool(poolName)
	if !exists {
		return nil, errors.New("pool not found: " + poolName)
	}

	ctx, cancel := context.WithTimeout(rm.ctx, timeout)
	defer cancel()

	return pool.AcquireResource(ctx)
}

// ReleaseResource returns a resource to its pool
func (rm *ResourceManager) ReleaseResource(resource *Resource) error {
	if resource == nil {
		return errors.New("resource is nil")
	}

	// Find the pool that owns this resource
	for _, pool := range rm.pools {
		if pool.OwnsResource(resource) {
			return pool.ReleaseResource(resource)
		}
	}

	return errors.New("no pool found for resource: " + resource.ID)
}

// GetGlobalStatistics returns aggregated statistics across all pools
func (rm *ResourceManager) GetGlobalStatistics() *GlobalResourceStatistics {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	// Create a copy
	stats := *rm.globalStats
	return &stats
}

// GetPoolStatistics returns statistics for all pools
func (rm *ResourceManager) GetPoolStatistics() map[string]*PoolStatistics {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	result := make(map[string]*PoolStatistics)
	for name, pool := range rm.pools {
		result[name] = pool.GetStatistics()
	}

	return result
}

// ResourcePoolConfig holds configuration for a resource pool
type ResourcePoolConfig struct {
	Type                   ResourceType           `json:"type"`
	MinSize                int                    `json:"min_size"`
	MaxSize                int                    `json:"max_size"`
	IdleTimeout            time.Duration          `json:"idle_timeout"`
	MaxLifetime            time.Duration          `json:"max_lifetime"`
	AcquisitionTimeout     time.Duration          `json:"acquisition_timeout"`
	HealthCheckInterval    time.Duration          `json:"health_check_interval"`
	PreWarming             bool                   `json:"pre_warming"`
	AdaptiveScaling        bool                   `json:"adaptive_scaling"`
	MetricsEnabled         bool                   `json:"metrics_enabled"`
	TracingEnabled         bool                   `json:"tracing_enabled"`
	RetryAttempts          int                    `json:"retry_attempts"`
	RetryBackoff           time.Duration          `json:"retry_backoff"`
	LoadBalancingAlgorithm string                 `json:"load_balancing_algorithm"`
	CircuitBreakerEnabled  bool                   `json:"circuit_breaker_enabled"`
	FactoryConfig          map[string]interface{} `json:"factory_config"`
}

// NewResourcePool creates a new resource pool
func NewResourcePool(ctx context.Context, name string, config *ResourcePoolConfig, factory ResourceFactory) (*ResourcePool, error) {
	if config == nil {
		config = getDefaultResourcePoolConfig()
	}

	if factory == nil {
		return nil, errors.New("resource factory is required")
	}

	poolCtx, cancel := context.WithCancel(ctx)

	pool := &ResourcePool{
		Name:                name,
		Type:                config.Type,
		MinSize:             config.MinSize,
		MaxSize:             config.MaxSize,
		IdleTimeout:         config.IdleTimeout,
		MaxLifetime:         config.MaxLifetime,
		AcquisitionTimeout:  config.AcquisitionTimeout,
		HealthCheckInterval: config.HealthCheckInterval,
		resources:           make([]*Resource, 0, config.MaxSize),
		available:           make(chan *Resource, config.MaxSize),
		factory:             factory,
		statistics:          &PoolStatistics{WindowStart: time.Now()},
		ctx:                 poolCtx,
		cancel:              cancel,
		preWarming:          config.PreWarming,
		adaptiveScaling:     config.AdaptiveScaling,
		metricsEnabled:      config.MetricsEnabled,
		tracingEnabled:      config.TracingEnabled,
		enabled:             true,
	}

	// Initialize circuit breaker if enabled
	if config.CircuitBreakerEnabled {
		pool.circuitBreaker = NewCircuitBreaker(5, 30*time.Second)
	}

	// Initialize load balancer
	pool.loadBalancer = NewRoundRobinLoadBalancer()

	// Initialize retry policy
	pool.retryPolicy = &RetryPolicy{
		MaxRetries:    config.RetryAttempts,
		BaseDelay:     config.RetryBackoff,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		JitterEnabled: true,
	}

	// Pre-warm the pool if configured
	if config.PreWarming {
		if err := pool.preWarm(); err != nil {
			return nil, errors.New("failed to pre-warm pool: " + err.Error())
		}
	}

	// Start background operations
	pool.startBackgroundOperations()

	return pool, nil
}

// AcquireResource acquires a resource from the pool
func (rp *ResourcePool) AcquireResource(ctx context.Context) (*Resource, error) {
	if !rp.enabled {
		return nil, errors.New("resource pool is disabled")
	}

	start := time.Now()
	defer func() {
		rp.updateAcquisitionTime(time.Since(start))
	}()

	// Check circuit breaker
	if rp.circuitBreaker != nil && !rp.circuitBreaker.CanExecute() {
		atomic.AddInt64(&rp.statistics.TotalErrors, 1)
		return nil, errors.New("circuit breaker is open")
	}

	// Try to get an available resource
	select {
	case resource := <-rp.available:
		if rp.validator != nil && !rp.validator.IsHealthy(resource) {
			// Resource is unhealthy, try to create a new one
			go rp.destroyResource(resource)
			return rp.createNewResource(ctx)
		}

		resource.mutex.Lock()
		resource.Status = StatusInUse
		resource.LastUsed = time.Now()
		resource.UsageCount++
		resource.mutex.Unlock()

		atomic.AddInt64(&rp.statistics.TotalAcquired, 1)
		rp.updateUtilization()

		return resource, nil

	case <-time.After(rp.AcquisitionTimeout):
		atomic.AddInt64(&rp.statistics.TotalTimeouts, 1)
		return nil, errors.New("timeout acquiring resource from pool: " + rp.Name)

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReleaseResource returns a resource to the pool
func (rp *ResourcePool) ReleaseResource(resource *Resource) error {
	if resource == nil {
		return errors.New("resource is nil")
	}

	resource.mutex.Lock()
	defer resource.mutex.Unlock()

	if resource.Status != StatusInUse {
		return errors.New("resource is not in use: " + resource.ID)
	}

	// Check if resource has exceeded its lifetime
	if time.Since(resource.CreatedAt) > rp.MaxLifetime {
		go rp.destroyResource(resource)
		return nil
	}

	// Check if resource has too many errors
	if resource.ErrorCount > 10 {
		go rp.destroyResource(resource)
		return nil
	}

	resource.Status = StatusAvailable
	resource.LastUsed = time.Now()

	select {
	case rp.available <- resource:
		atomic.AddInt64(&rp.statistics.TotalReleased, 1)
		rp.updateUtilization()
		return nil
	default:
		// Pool is full, destroy the resource
		go rp.destroyResource(resource)
		return nil
	}
}

// GetStatistics returns pool statistics
func (rp *ResourcePool) GetStatistics() *PoolStatistics {
	rp.statsMutex.RLock()
	defer rp.statsMutex.RUnlock()

	// Create a copy
	stats := *rp.statistics
	return &stats
}

// OwnsResource checks if the pool owns the given resource
func (rp *ResourcePool) OwnsResource(resource *Resource) bool {
	rp.mutex.RLock()
	defer rp.mutex.RUnlock()

	for _, r := range rp.resources {
		if r.ID == resource.ID {
			return true
		}
	}

	return false
}

// Helper methods

func (rp *ResourcePool) createNewResource(ctx context.Context) (*Resource, error) {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	if rp.CurrentSize >= rp.MaxSize {
		return nil, errors.New("pool has reached maximum size: " + strconv.Itoa(rp.MaxSize))
	}

	resource, err := rp.factory.CreateResource(ctx, map[string]interface{}{
		"pool_name": rp.Name,
		"pool_type": rp.Type,
	})
	if err != nil {
		atomic.AddInt64(&rp.statistics.TotalErrors, 1)
		if rp.circuitBreaker != nil {
			rp.circuitBreaker.RecordFailure()
		}
		return nil, err
	}

	resource.Status = StatusInUse
	resource.LastUsed = time.Now()
	resource.UsageCount = 1

	rp.resources = append(rp.resources, resource)
	rp.CurrentSize++

	atomic.AddInt64(&rp.statistics.TotalCreated, 1)
	atomic.AddInt64(&rp.statistics.TotalAcquired, 1)

	if int64(rp.CurrentSize) > rp.statistics.PeakConnections {
		rp.statistics.PeakConnections = int64(rp.CurrentSize)
	}

	rp.updateUtilization()

	if rp.circuitBreaker != nil {
		rp.circuitBreaker.RecordSuccess()
	}

	return resource, nil
}

func (rp *ResourcePool) destroyResource(resource *Resource) {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	// Remove from resources slice
	for i, r := range rp.resources {
		if r.ID == resource.ID {
			rp.resources = append(rp.resources[:i], rp.resources[i+1:]...)
			break
		}
	}

	// Cleanup the resource
	if resource.Cleanup != nil {
		if err := resource.Cleanup(); err != nil {
			// Log error but continue - future implementation will add proper logging
			_ = err
		}
	}

	rp.CurrentSize--
	atomic.AddInt64(&rp.statistics.TotalDestroyed, 1)
	rp.updateUtilization()
}

func (rp *ResourcePool) preWarm() error {
	for i := 0; i < rp.MinSize; i++ {
		resource, err := rp.factory.CreateResource(rp.ctx, map[string]interface{}{
			"pool_name": rp.Name,
			"pool_type": rp.Type,
		})
		if err != nil {
			return err
		}

		resource.Status = StatusAvailable
		rp.resources = append(rp.resources, resource)
		rp.available <- resource
		rp.CurrentSize++

		atomic.AddInt64(&rp.statistics.TotalCreated, 1)
	}

	return nil
}

func (rp *ResourcePool) updateAcquisitionTime(duration time.Duration) {
	rp.statsMutex.Lock()
	defer rp.statsMutex.Unlock()

	// Simple moving average
	rp.statistics.AverageAcquisitionTime = (rp.statistics.AverageAcquisitionTime + duration) / 2
}

func (rp *ResourcePool) updateUtilization() {
	rp.statsMutex.Lock()
	defer rp.statsMutex.Unlock()

	activeCount := rp.CurrentSize - len(rp.available)
	rp.statistics.ActiveConnections = int64(activeCount)
	rp.statistics.IdleConnections = int64(len(rp.available))

	if rp.MaxSize > 0 {
		rp.statistics.UtilizationRate = float64(activeCount) / float64(rp.MaxSize)
	}
}

func (rp *ResourcePool) startBackgroundOperations() {
	// Health check loop
	rp.backgroundWG.Add(1)
	go rp.healthCheckLoop()

	// Cleanup loop
	rp.backgroundWG.Add(1)
	go rp.cleanupLoop()

	// Metrics collection loop
	if rp.metricsEnabled {
		rp.backgroundWG.Add(1)
		go rp.metricsLoop()
	}
}

func (rp *ResourcePool) healthCheckLoop() {
	defer rp.backgroundWG.Done()

	ticker := time.NewTicker(rp.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rp.ctx.Done():
			return
		case <-ticker.C:
			rp.performHealthCheck()
		}
	}
}

func (rp *ResourcePool) cleanupLoop() {
	defer rp.backgroundWG.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rp.ctx.Done():
			return
		case <-ticker.C:
			rp.cleanupIdleResources()
		}
	}
}

func (rp *ResourcePool) metricsLoop() {
	defer rp.backgroundWG.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rp.ctx.Done():
			return
		case <-ticker.C:
			rp.collectMetrics()
		}
	}
}

func (rp *ResourcePool) performHealthCheck() {
	rp.mutex.RLock()
	resources := make([]*Resource, len(rp.resources))
	copy(resources, rp.resources)
	rp.mutex.RUnlock()

	for _, resource := range resources {
		if resource.HealthCheck != nil {
			if err := resource.HealthCheck(); err != nil {
				resource.mutex.Lock()
				resource.ErrorCount++
				resource.mutex.Unlock()

				if resource.ErrorCount > 5 {
					go rp.destroyResource(resource)
				}
			}
		}
	}
}

func (rp *ResourcePool) cleanupIdleResources() {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	now := time.Now()
	resourcesToDestroy := []*Resource{}

	for _, resource := range rp.resources {
		resource.mutex.RLock()
		isIdle := resource.Status == StatusAvailable && now.Sub(resource.LastUsed) > rp.IdleTimeout
		isExpired := now.Sub(resource.CreatedAt) > rp.MaxLifetime
		resource.mutex.RUnlock()

		if (isIdle || isExpired) && rp.CurrentSize > rp.MinSize {
			resourcesToDestroy = append(resourcesToDestroy, resource)
		}
	}

	for _, resource := range resourcesToDestroy {
		go rp.destroyResource(resource)
	}
}

func (rp *ResourcePool) collectMetrics() {
	stats := rp.GetStatistics()

	// Emit metrics (placeholder - integrate with actual metrics system)
	_ = stats
}

// Resource Manager background operations

func (rm *ResourceManager) startBackgroundOperations() {
	// Global statistics update
	rm.backgroundWG.Add(1)
	go rm.globalStatsLoop()

	// Health monitoring
	if rm.healthMonitor != nil {
		rm.backgroundWG.Add(1)
		go rm.healthMonitoringLoop()
	}

	// Auto scaling
	if rm.autoScaler != nil {
		rm.backgroundWG.Add(1)
		go rm.autoScalingLoop()
	}
}

func (rm *ResourceManager) globalStatsLoop() {
	defer rm.backgroundWG.Done()

	ticker := time.NewTicker(rm.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.updateGlobalStats()
		}
	}
}

func (rm *ResourceManager) healthMonitoringLoop() {
	defer rm.backgroundWG.Done()

	ticker := time.NewTicker(rm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.performGlobalHealthCheck()
		}
	}
}

func (rm *ResourceManager) autoScalingLoop() {
	defer rm.backgroundWG.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.performAutoScaling()
		}
	}
}

func (rm *ResourceManager) updateGlobalStats() {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	totalResources := 0
	totalActive := 0
	totalIdle := 0
	totalSuccess := int64(0)
	totalOperations := int64(0)

	for _, pool := range rm.pools {
		stats := pool.GetStatistics()
		totalResources += pool.CurrentSize
		totalActive += int(stats.ActiveConnections)
		totalIdle += int(stats.IdleConnections)
		totalSuccess += stats.TotalAcquired - stats.TotalErrors
		totalOperations += stats.TotalAcquired
	}

	rm.globalStats.TotalPools = len(rm.pools)
	rm.globalStats.TotalResources = totalResources
	rm.globalStats.TotalActiveResources = totalActive
	rm.globalStats.TotalIdleResources = totalIdle

	if totalResources > 0 {
		rm.globalStats.GlobalUtilizationRate = float64(totalActive) / float64(totalResources)
	}

	if totalOperations > 0 {
		rm.globalStats.GlobalSuccessRate = float64(totalSuccess) / float64(totalOperations)
		rm.globalStats.GlobalErrorRate = 1.0 - rm.globalStats.GlobalSuccessRate
	}

	rm.globalStats.LastUpdated = time.Now()
}

func (rm *ResourceManager) performGlobalHealthCheck() {
	// Check health of all pools and trigger alerts if needed
	for name, pool := range rm.pools {
		stats := pool.GetStatistics()

		// Check utilization
		if stats.UtilizationRate > 0.9 {
			rm.triggerAlert(&ResourceAlert{
				ID:        generateAlertID(),
				PoolName:  name,
				Severity:  "warning",
				Message:   fmt.Sprintf("High utilization: %.2f%%", stats.UtilizationRate*100),
				Value:     stats.UtilizationRate,
				Threshold: 0.9,
				Timestamp: time.Now(),
			})
		}

		// Check error rate
		if stats.ErrorRate > 0.1 {
			rm.triggerAlert(&ResourceAlert{
				ID:        generateAlertID(),
				PoolName:  name,
				Severity:  "critical",
				Message:   fmt.Sprintf("High error rate: %.2f%%", stats.ErrorRate*100),
				Value:     stats.ErrorRate,
				Threshold: 0.1,
				Timestamp: time.Now(),
			})
		}
	}
}

func (rm *ResourceManager) performAutoScaling() {
	if rm.autoScaler == nil {
		return
	}

	for name, pool := range rm.pools {
		policy := rm.autoScaler.GetPolicy(name)
		if policy == nil || !policy.Enabled {
			continue
		}

		stats := pool.GetStatistics()

		// Scale up if utilization is high
		if stats.UtilizationRate > policy.TargetUtilization && pool.CurrentSize < policy.MaxSize {
			if time.Since(policy.LastScaleAction) > policy.ScaleUpCooldown {
				_ = rm.scalePool(name, pool.CurrentSize+policy.ScaleUpStep)
				policy.LastScaleAction = time.Now()
			}
		}

		// Scale down if utilization is low
		if stats.UtilizationRate < policy.TargetUtilization*0.5 && pool.CurrentSize > policy.MinSize {
			if time.Since(policy.LastScaleAction) > policy.ScaleDownCooldown {
				newSize := pool.CurrentSize - policy.ScaleDownStep
				if newSize < policy.MinSize {
					newSize = policy.MinSize
				}
				_ = rm.scalePool(name, newSize)
				policy.LastScaleAction = time.Now()
			}
		}
	}
}

func (rm *ResourceManager) scalePool(poolName string, targetSize int) error {
	// Placeholder implementation
	return nil
}

func (rm *ResourceManager) triggerAlert(alert *ResourceAlert) {
	if rm.alertingEngine != nil {
		rm.alertingEngine.TriggerAlert(alert)
	}
}

// Shutdown gracefully shuts down the resource manager
func (rm *ResourceManager) Shutdown() error {
	rm.cancel()
	rm.backgroundWG.Wait()

	// Shutdown all pools
	for _, pool := range rm.pools {
		_ = pool.Shutdown()
	}

	return nil
}

func (rp *ResourcePool) Shutdown() error {
	rp.cancel()
	rp.backgroundWG.Wait()

	// Close all resources
	for _, resource := range rp.resources {
		if resource.Cleanup != nil {
			_ = resource.Cleanup()
		}
	}

	return nil
}

// Placeholder implementations for supporting types

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
	}
}

func (cb *CircuitBreaker) CanExecute() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	if cb.state == CircuitClosed {
		return true
	}

	if cb.state == CircuitOpen && time.Since(cb.lastFailureTime) > cb.resetTimeout {
		cb.state = CircuitHalfOpen
		return true
	}

	return cb.state == CircuitHalfOpen
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount = 0
	cb.state = CircuitClosed
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= int64(cb.maxFailures) {
		cb.state = CircuitOpen
	}
}

type RoundRobinLoadBalancer struct {
	current int64
}

func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{}
}

func (rrb *RoundRobinLoadBalancer) SelectResource(resources []*Resource, criteria map[string]interface{}) (*Resource, error) {
	if len(resources) == 0 {
		return nil, errors.New("no resources available")
	}

	index := atomic.AddInt64(&rrb.current, 1) % int64(len(resources))
	return resources[index], nil
}

func (rrb *RoundRobinLoadBalancer) GetAlgorithm() string {
	return "round_robin"
}

func (rrb *RoundRobinLoadBalancer) UpdateWeights(resources []*Resource, metrics map[string]float64) {
	// Round robin doesn't use weights
}

func NewResourceHealthMonitor() *ResourceHealthMonitor {
	return &ResourceHealthMonitor{
		healthChecks:    make(map[string]*HealthCheck),
		alertThresholds: make(map[string]*HealthThreshold),
		enabled:         true,
	}
}

func NewResourceAlertingEngine() *ResourceAlertingEngine {
	return &ResourceAlertingEngine{
		rules:     make(map[string]*ResourceAlertRule),
		alerts:    make([]*ResourceAlert, 0),
		callbacks: make([]func(*ResourceAlert), 0),
		enabled:   true,
	}
}

func (rae *ResourceAlertingEngine) TriggerAlert(alert *ResourceAlert) {
	rae.mutex.Lock()
	defer rae.mutex.Unlock()

	rae.alerts = append(rae.alerts, alert)

	for _, callback := range rae.callbacks {
		go callback(alert)
	}
}

func NewResourceAutoScaler() *ResourceAutoScaler {
	return &ResourceAutoScaler{
		policies: make(map[string]*ScalingPolicy),
		enabled:  true,
	}
}

func (ras *ResourceAutoScaler) GetPolicy(poolName string) *ScalingPolicy {
	ras.mutex.RLock()
	defer ras.mutex.RUnlock()

	return ras.policies[poolName]
}

func NewFailoverManager() *FailoverManager {
	return &FailoverManager{
		primaryPools:    make(map[string]string),
		backupPools:     make(map[string][]string),
		failoverHistory: make(map[string][]*FailoverEvent),
		enabled:         true,
	}
}

func getDefaultResourceManagerConfig() *ResourceManagerConfig {
	return &ResourceManagerConfig{
		GlobalMaxResources:    1000,
		GlobalIdleTimeout:     5 * time.Minute,
		GlobalMaxLifetime:     1 * time.Hour,
		HealthCheckInterval:   30 * time.Second,
		MetricsInterval:       10 * time.Second,
		CleanupInterval:       1 * time.Minute,
		AutoScalingEnabled:    true,
		LoadBalancingEnabled:  true,
		FailoverEnabled:       true,
		CircuitBreakerEnabled: true,
		TracingEnabled:        false,
		MetricsEnabled:        true,
		DefaultRetryAttempts:  3,
		DefaultRetryBackoff:   1 * time.Second,
	}
}

func getDefaultResourcePoolConfig() *ResourcePoolConfig {
	return &ResourcePoolConfig{
		Type:                   ResourceTypeGeneric,
		MinSize:                2,
		MaxSize:                10,
		IdleTimeout:            5 * time.Minute,
		MaxLifetime:            1 * time.Hour,
		AcquisitionTimeout:     30 * time.Second,
		HealthCheckInterval:    30 * time.Second,
		PreWarming:             true,
		AdaptiveScaling:        true,
		MetricsEnabled:         true,
		TracingEnabled:         false,
		RetryAttempts:          3,
		RetryBackoff:           1 * time.Second,
		LoadBalancingAlgorithm: "round_robin",
		CircuitBreakerEnabled:  true,
		FactoryConfig:          make(map[string]interface{}),
	}
}

func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}
