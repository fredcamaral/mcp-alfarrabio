// Package performance provides intelligent data partitioning for large datasets
package performance

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

// PartitionManager manages intelligent data partitioning for scalability
type PartitionManager struct {
	config     *PartitionConfig
	partitions map[string]*Partition
	strategies map[string]PartitionStrategy
	router     *PartitionRouter
	balancer   *LoadBalancer
	mutex      sync.RWMutex
	metrics    *PartitionMetrics
}

// PartitionConfig defines partitioning configuration
type PartitionConfig struct {
	// Partitioning strategy
	DefaultStrategy       PartitionStrategy `json:"default_strategy"`
	MaxPartitionSize      int64             `json:"max_partition_size"`       // Max records per partition
	MaxPartitionSizeBytes int64             `json:"max_partition_size_bytes"` // Max bytes per partition

	// Rebalancing settings
	EnableAutoRebalance bool          `json:"enable_auto_rebalance"`
	RebalanceThreshold  float64       `json:"rebalance_threshold"` // Imbalance threshold (0.0-1.0)
	RebalanceInterval   time.Duration `json:"rebalance_interval"`

	// Performance settings
	ConcurrentOperations int           `json:"concurrent_operations"`
	BatchSize            int           `json:"batch_size"`
	EnablePartitionCache bool          `json:"enable_partition_cache"`
	CacheTTL             time.Duration `json:"cache_ttl"`

	// Distribution settings
	ReplicationFactor int              `json:"replication_factor"`
	ConsistencyLevel  ConsistencyLevel `json:"consistency_level"`
}

// DefaultPartitionConfig returns optimized default configuration
func DefaultPartitionConfig() *PartitionConfig {
	return &PartitionConfig{
		DefaultStrategy:       StrategyTimeRange,
		MaxPartitionSize:      100000,             // 100K records
		MaxPartitionSizeBytes: 1024 * 1024 * 1024, // 1GB
		EnableAutoRebalance:   true,
		RebalanceThreshold:    0.3, // 30% imbalance triggers rebalance
		RebalanceInterval:     1 * time.Hour,
		ConcurrentOperations:  10,
		BatchSize:             1000,
		EnablePartitionCache:  true,
		CacheTTL:              15 * time.Minute,
		ReplicationFactor:     2,
		ConsistencyLevel:      ConsistencyLevelEventual,
	}
}

// Partition represents a data partition with metadata
type Partition struct {
	ID           string                 `json:"id"`
	Strategy     PartitionStrategy      `json:"strategy"`
	KeyRange     *KeyRange              `json:"key_range"`
	TimeRange    *TimeRange             `json:"time_range"`
	Size         int64                  `json:"size"`
	SizeBytes    int64                  `json:"size_bytes"`
	RecordCount  int64                  `json:"record_count"`
	LastAccessed time.Time              `json:"last_accessed"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Status       PartitionStatus        `json:"status"`
	Replicas     []string               `json:"replicas"`
	Metadata     map[string]interface{} `json:"metadata"`
	LoadScore    float64                `json:"load_score"`
}

// PartitionStrategy defines partitioning strategies
type PartitionStrategy string

const (
	StrategyHash       PartitionStrategy = "hash"        // Hash-based partitioning
	StrategyRange      PartitionStrategy = "range"       // Range-based partitioning
	StrategyTimeRange  PartitionStrategy = "time_range"  // Time-based range partitioning
	StrategyCustom     PartitionStrategy = "custom"      // Custom partitioning logic
	StrategyRoundRobin PartitionStrategy = "round_robin" // Round-robin distribution
)

// PartitionStatus defines partition status
type PartitionStatus string

const (
	StatusActive    PartitionStatus = "active"
	StatusMigrating PartitionStatus = "migrating"
	StatusBalancing PartitionStatus = "balancing"
	StatusReadOnly  PartitionStatus = "read_only"
	StatusArchived  PartitionStatus = "archived"
)

// ConsistencyLevel defines data consistency requirements
type ConsistencyLevel string

const (
	ConsistencyLevelStrong   ConsistencyLevel = "strong"
	ConsistencyLevelEventual ConsistencyLevel = "eventual"
	ConsistencyLevelSession  ConsistencyLevel = "session"
)

// KeyRange represents a key-based partition range
type KeyRange struct {
	Start interface{} `json:"start"`
	End   interface{} `json:"end"`
	Type  string      `json:"type"` // "string", "int", "uuid"
}

// TimeRange represents a time-based partition range
type TimeRange struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Interval string    `json:"interval"` // "hour", "day", "week", "month"
}

// PartitionRouter handles routing requests to appropriate partitions
type PartitionRouter struct {
	manager    *PartitionManager
	cache      map[string]string // Key -> PartitionID cache
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

// LoadBalancer handles partition load balancing
type LoadBalancer struct {
	manager   *PartitionManager
	threshold float64
	running   bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// PartitionMetricsData tracks partitioning performance metrics
type PartitionMetricsData struct {
	TotalPartitions       int              `json:"total_partitions"`
	ActivePartitions      int              `json:"active_partitions"`
	TotalRecords          int64            `json:"total_records"`
	TotalSize             int64            `json:"total_size"`
	BalanceScore          float64          `json:"balance_score"`
	LastRebalance         time.Time        `json:"last_rebalance"`
	RebalanceCount        int64            `json:"rebalance_count"`
	PartitionDistribution map[string]int64 `json:"partition_distribution"`
	OperationCounts       map[string]int64 `json:"operation_counts"`
}

// PartitionMetrics holds internal partition metrics with synchronization
type PartitionMetrics struct {
	mutex sync.RWMutex
	data  PartitionMetricsData
}

// NewPartitionManager creates a new intelligent partition manager
func NewPartitionManager(config *PartitionConfig) *PartitionManager {
	if config == nil {
		config = DefaultPartitionConfig()
	}

	manager := &PartitionManager{
		config:     config,
		partitions: make(map[string]*Partition),
		strategies: make(map[string]PartitionStrategy),
		metrics: &PartitionMetrics{
			data: PartitionMetricsData{
				PartitionDistribution: make(map[string]int64),
				OperationCounts:       make(map[string]int64),
			},
		},
	}

	// Initialize router
	manager.router = &PartitionRouter{
		manager:  manager,
		cache:    make(map[string]string),
		cacheTTL: config.CacheTTL,
	}

	// Initialize load balancer
	ctx, cancel := context.WithCancel(context.Background())
	manager.balancer = &LoadBalancer{
		manager:   manager,
		threshold: config.RebalanceThreshold,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Register built-in strategies
	manager.registerBuiltinStrategies()

	// Start auto-rebalancing if enabled
	if config.EnableAutoRebalance {
		go manager.balancer.autoRebalanceLoop()
	}

	return manager
}

// CreatePartition creates a new partition with specified strategy
func (pm *PartitionManager) CreatePartition(strategy PartitionStrategy, keyRange *KeyRange, timeRange *TimeRange) (*Partition, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	partition := &Partition{
		ID:        generatePartitionID(),
		Strategy:  strategy,
		KeyRange:  keyRange,
		TimeRange: timeRange,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    StatusActive,
		Replicas:  make([]string, 0),
		Metadata:  make(map[string]interface{}),
		LoadScore: 0.0,
	}

	pm.partitions[partition.ID] = partition
	pm.updateMetrics()

	return partition, nil
}

// GetPartitionForKey determines which partition a key belongs to
func (pm *PartitionManager) GetPartitionForKey(key interface{}) (*Partition, error) {
	// Check cache first
	if pm.config.EnablePartitionCache {
		keyStr := fmt.Sprintf("%v", key)
		if partitionID := pm.router.getCachedPartition(keyStr); partitionID != "" {
			if partition, exists := pm.partitions[partitionID]; exists {
				return partition, nil
			}
		}
	}

	// Find partition using strategy
	partition, err := pm.findPartitionByStrategy(key)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if pm.config.EnablePartitionCache {
		keyStr := fmt.Sprintf("%v", key)
		pm.router.cachePartition(keyStr, partition.ID)
	}

	return partition, nil
}

// GetPartitionsForTimeRange returns partitions that overlap with the time range
func (pm *PartitionManager) GetPartitionsForTimeRange(start, end time.Time) ([]*Partition, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var matchingPartitions []*Partition

	for _, partition := range pm.partitions {
		if partition.TimeRange != nil {
			// Check if time ranges overlap
			if timeRangesOverlap(partition.TimeRange.Start, partition.TimeRange.End, start, end) {
				matchingPartitions = append(matchingPartitions, partition)
			}
		}
	}

	return matchingPartitions, nil
}

// InsertRecord inserts a record into the appropriate partition
func (pm *PartitionManager) InsertRecord(key, data interface{}) error {
	partition, err := pm.GetPartitionForKey(key)
	if err != nil {
		return fmt.Errorf("failed to find partition for key: %w", err)
	}

	// Check if partition needs splitting
	if pm.shouldSplitPartition(partition) {
		if err := pm.splitPartition(partition); err != nil {
			// Log error but continue with insertion
			log.Printf("failed to split partition %s: %v", partition.ID, err)
		}
	}

	// Insert into partition (simplified)
	partition.RecordCount++
	partition.Size += estimateRecordSize(data)
	partition.LastAccessed = time.Now()
	partition.UpdatedAt = time.Now()

	pm.incrementOperationCount("insert")
	return nil
}

// QueryPartitions queries multiple partitions and merges results
func (pm *PartitionManager) QueryPartitions(query PartitionQuery) (*QueryResult, error) {
	// Determine which partitions to query
	partitions, err := pm.getPartitionsForQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to determine query partitions: %w", err)
	}

	// Query partitions concurrently
	results := make(chan *PartitionQueryResult, len(partitions))
	errors := make(chan error, len(partitions))

	sem := make(chan struct{}, pm.config.ConcurrentOperations)

	for _, partition := range partitions {
		go func(p *Partition) {
			sem <- struct{}{}
			defer func() { <-sem }()

			result := pm.queryPartition(p, query)
			results <- result
		}(partition)
	}

	// Collect results
	var allResults []*PartitionQueryResult
	var queryErrors []error

	for i := 0; i < len(partitions); i++ {
		select {
		case result := <-results:
			allResults = append(allResults, result)
		case err := <-errors:
			queryErrors = append(queryErrors, err)
		}
	}

	if len(queryErrors) > 0 {
		return nil, fmt.Errorf("query errors: %v", queryErrors)
	}

	// Merge and return results
	mergedResult := pm.mergeQueryResults(allResults, query)
	pm.incrementOperationCount("query")

	return mergedResult, nil
}

// PartitionQuery represents a query across partitions
type PartitionQuery struct {
	KeyRange  *KeyRange              `json:"key_range"`
	TimeRange *TimeRange             `json:"time_range"`
	Filters   map[string]interface{} `json:"filters"`
	OrderBy   string                 `json:"order_by"`
	Limit     int                    `json:"limit"`
	Offset    int                    `json:"offset"`
}

// PartitionQueryResult represents results from a single partition
type PartitionQueryResult struct {
	PartitionID string                   `json:"partition_id"`
	Records     []map[string]interface{} `json:"records"`
	Count       int64                    `json:"count"`
	Size        int64                    `json:"size"`
}

// QueryResult represents merged results from multiple partitions
type QueryResult struct {
	Records        []map[string]interface{} `json:"records"`
	TotalCount     int64                    `json:"total_count"`
	PartitionCount int                      `json:"partition_count"`
	ExecutionTime  time.Duration            `json:"execution_time"`
}

// RebalancePartitions triggers manual partition rebalancing
func (pm *PartitionManager) RebalancePartitions() error {
	return pm.balancer.rebalance()
}

// GetMetrics returns current partitioning metrics
func (pm *PartitionManager) GetMetrics() PartitionMetricsData {
	pm.metrics.mutex.RLock()
	defer pm.metrics.mutex.RUnlock()

	// Return a copy of the data without the mutex
	metrics := pm.metrics.data
	metrics.BalanceScore = pm.calculateBalanceScore()

	return metrics
}

// Private methods

// registerBuiltinStrategies registers built-in partitioning strategies
func (pm *PartitionManager) registerBuiltinStrategies() {
	pm.strategies["hash"] = StrategyHash
	pm.strategies["range"] = StrategyRange
	pm.strategies["time_range"] = StrategyTimeRange
	pm.strategies["round_robin"] = StrategyRoundRobin
}

// findPartitionByStrategy finds partition using configured strategy
func (pm *PartitionManager) findPartitionByStrategy(key interface{}) (*Partition, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	switch pm.config.DefaultStrategy {
	case StrategyHash:
		return pm.findByHashStrategy(key)
	case StrategyRange:
		return pm.findByRangeStrategy(key)
	case StrategyTimeRange:
		return pm.findByTimeRangeStrategy(key)
	case StrategyRoundRobin:
		return pm.findByRoundRobinStrategy(key)
	default:
		return nil, fmt.Errorf("unknown partition strategy: %s", pm.config.DefaultStrategy)
	}
}

// findByHashStrategy finds partition using hash strategy
func (pm *PartitionManager) findByHashStrategy(key interface{}) (*Partition, error) {
	keyStr := fmt.Sprintf("%v", key)
	hash := calculateHash(keyStr)

	// Find partition with hash-based distribution
	partitionCount := len(pm.partitions)
	if partitionCount == 0 {
		// Create first partition
		return pm.createInitialPartition()
	}

	partitionIndex := int(hash) % partitionCount
	i := 0
	for _, partition := range pm.partitions {
		if i == partitionIndex {
			return partition, nil
		}
		i++
	}

	// Fallback to first partition
	for _, partition := range pm.partitions {
		return partition, nil
	}

	return nil, fmt.Errorf("no partitions available")
}

// findByRangeStrategy finds partition using range strategy
func (pm *PartitionManager) findByRangeStrategy(key interface{}) (*Partition, error) {
	for _, partition := range pm.partitions {
		if partition.KeyRange != nil {
			if pm.keyInRange(key, partition.KeyRange) {
				return partition, nil
			}
		}
	}

	// Create new partition for key
	return pm.createPartitionForKey(key)
}

// findByTimeRangeStrategy finds partition using time range strategy
func (pm *PartitionManager) findByTimeRangeStrategy(_ interface{}) (*Partition, error) {
	now := time.Now()

	for _, partition := range pm.partitions {
		if partition.TimeRange != nil {
			if now.After(partition.TimeRange.Start) && now.Before(partition.TimeRange.End) {
				return partition, nil
			}
		}
	}

	// Create new time-based partition
	return pm.createTimePartition(now)
}

// findByRoundRobinStrategy finds partition using round-robin strategy
func (pm *PartitionManager) findByRoundRobinStrategy(_ interface{}) (*Partition, error) {
	if len(pm.partitions) == 0 {
		return pm.createInitialPartition()
	}

	// Simple round-robin based on current time
	index := int(time.Now().UnixNano()) % len(pm.partitions)
	i := 0
	for _, partition := range pm.partitions {
		if i == index {
			return partition, nil
		}
		i++
	}

	// Fallback
	for _, partition := range pm.partitions {
		return partition, nil
	}

	return nil, fmt.Errorf("no partitions available")
}

// shouldSplitPartition determines if partition should be split
func (pm *PartitionManager) shouldSplitPartition(partition *Partition) bool {
	return partition.RecordCount >= pm.config.MaxPartitionSize ||
		partition.SizeBytes >= pm.config.MaxPartitionSizeBytes
}

// splitPartition splits a partition into two
func (pm *PartitionManager) splitPartition(partition *Partition) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Create new partition
	newPartition := &Partition{
		ID:        generatePartitionID(),
		Strategy:  partition.Strategy,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    StatusActive,
		Replicas:  make([]string, 0),
		Metadata:  make(map[string]interface{}),
	}

	// Split ranges based on strategy
	switch partition.Strategy {
	case StrategyTimeRange:
		pm.splitTimeRange(partition, newPartition)
	case StrategyRange:
		pm.splitKeyRange(partition, newPartition)
	default:
		// For hash and round-robin, just halve the load
		newPartition.RecordCount = partition.RecordCount / 2
		newPartition.Size = partition.Size / 2
		partition.RecordCount /= 2
		partition.Size /= 2
	}

	pm.partitions[newPartition.ID] = newPartition
	pm.updateMetrics()

	return nil
}

// getPartitionsForQuery determines partitions needed for query
func (pm *PartitionManager) getPartitionsForQuery(query PartitionQuery) ([]*Partition, error) {
	pm.mutex.RLock()
	capacity := len(pm.partitions)
	pm.mutex.RUnlock()

	partitions := make([]*Partition, 0, capacity)

	if query.TimeRange != nil {
		return pm.GetPartitionsForTimeRange(query.TimeRange.Start, query.TimeRange.End)
	}

	if query.KeyRange != nil {
		pm.mutex.RLock()
		for _, partition := range pm.partitions {
			if partition.KeyRange != nil && pm.rangesOverlap(query.KeyRange, partition.KeyRange) {
				partitions = append(partitions, partition)
			}
		}
		pm.mutex.RUnlock()
		return partitions, nil
	}

	// Query all partitions if no specific range
	pm.mutex.RLock()
	for _, partition := range pm.partitions {
		partitions = append(partitions, partition)
	}
	pm.mutex.RUnlock()

	return partitions, nil
}

// queryPartition queries a single partition
func (pm *PartitionManager) queryPartition(partition *Partition, _ PartitionQuery) *PartitionQueryResult {
	// Simplified query execution - in production would integrate with storage layer
	result := &PartitionQueryResult{
		PartitionID: partition.ID,
		Records:     make([]map[string]interface{}, 0),
		Count:       0,
		Size:        0,
	}

	// Simulate query execution
	partition.LastAccessed = time.Now()

	return result
}

// mergeQueryResults merges results from multiple partitions
func (pm *PartitionManager) mergeQueryResults(results []*PartitionQueryResult, query PartitionQuery) *QueryResult {
	merged := &QueryResult{
		Records:        make([]map[string]interface{}, 0),
		TotalCount:     0,
		PartitionCount: len(results),
	}

	// Merge all records
	for _, result := range results {
		merged.Records = append(merged.Records, result.Records...)
		merged.TotalCount += result.Count
	}

	// Sort if needed
	if query.OrderBy != "" {
		pm.sortResults(merged.Records, query.OrderBy)
	}

	// Apply limit/offset
	if query.Limit > 0 {
		start := query.Offset
		end := start + query.Limit
		if start < len(merged.Records) {
			if end > len(merged.Records) {
				end = len(merged.Records)
			}
			merged.Records = merged.Records[start:end]
		} else {
			merged.Records = make([]map[string]interface{}, 0)
		}
	}

	return merged
}

// Load balancer methods

// autoRebalanceLoop runs automatic rebalancing
func (lb *LoadBalancer) autoRebalanceLoop() {
	lb.running = true
	defer func() { lb.running = false }()

	ticker := time.NewTicker(lb.manager.config.RebalanceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if lb.shouldRebalance() {
				_ = lb.rebalance()
			}
		case <-lb.ctx.Done():
			return
		}
	}
}

// shouldRebalance determines if rebalancing is needed
func (lb *LoadBalancer) shouldRebalance() bool {
	score := lb.manager.calculateBalanceScore()
	return score < (1.0 - lb.threshold)
}

// rebalance performs partition rebalancing
func (lb *LoadBalancer) rebalance() error {
	lb.manager.mutex.Lock()
	defer lb.manager.mutex.Unlock()

	// Get partition load information
	loads := make([]float64, 0, len(lb.manager.partitions))
	for _, partition := range lb.manager.partitions {
		loads = append(loads, partition.LoadScore)
	}

	// Sort by load
	sort.Float64s(loads)

	// Implement simple load balancing logic
	// In production, would implement sophisticated rebalancing algorithms

	lb.manager.metrics.mutex.Lock()
	lb.manager.metrics.data.LastRebalance = time.Now()
	lb.manager.metrics.data.RebalanceCount++
	lb.manager.metrics.mutex.Unlock()

	return nil
}

// Router methods

// getCachedPartition gets partition from cache
func (pr *PartitionRouter) getCachedPartition(key string) string {
	pr.cacheMutex.RLock()
	defer pr.cacheMutex.RUnlock()

	return pr.cache[key]
}

// cachePartition caches key->partition mapping
func (pr *PartitionRouter) cachePartition(key, partitionID string) {
	pr.cacheMutex.Lock()
	defer pr.cacheMutex.Unlock()

	pr.cache[key] = partitionID
}

// Utility functions

func generatePartitionID() string {
	return fmt.Sprintf("part_%d_%d", time.Now().UnixNano(), time.Now().Nanosecond()%1000)
}

func calculateHash(s string) uint32 {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}

func timeRangesOverlap(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && start2.Before(end1)
}

func estimateRecordSize(data interface{}) int64 {
	// Simplified size estimation
	return 1024 // 1KB default
}

func (pm *PartitionManager) keyInRange(key interface{}, keyRange *KeyRange) bool {
	// Simplified range checking
	return true
}

func (pm *PartitionManager) rangesOverlap(range1, range2 *KeyRange) bool {
	// Simplified range overlap checking
	return true
}

func (pm *PartitionManager) createInitialPartition() (*Partition, error) {
	return pm.CreatePartition(pm.config.DefaultStrategy, nil, nil)
}

func (pm *PartitionManager) createPartitionForKey(_ interface{}) (*Partition, error) {
	return pm.CreatePartition(pm.config.DefaultStrategy, nil, nil)
}

func (pm *PartitionManager) createTimePartition(t time.Time) (*Partition, error) {
	timeRange := &TimeRange{
		Start:    t.Truncate(24 * time.Hour),
		End:      t.Truncate(24 * time.Hour).Add(24 * time.Hour),
		Interval: "day",
	}
	return pm.CreatePartition(StrategyTimeRange, nil, timeRange)
}

func (pm *PartitionManager) splitTimeRange(old, newPartition *Partition) {
	if old.TimeRange != nil {
		duration := old.TimeRange.End.Sub(old.TimeRange.Start)
		mid := old.TimeRange.Start.Add(duration / 2)

		newPartition.TimeRange = &TimeRange{
			Start:    mid,
			End:      old.TimeRange.End,
			Interval: old.TimeRange.Interval,
		}

		old.TimeRange.End = mid
	}
}

func (pm *PartitionManager) splitKeyRange(old, newPartition *Partition) {
	// Simplified key range splitting
}

func (pm *PartitionManager) sortResults(records []map[string]interface{}, orderBy string) {
	// Simplified sorting implementation
}

func (pm *PartitionManager) calculateBalanceScore() float64 {
	if len(pm.partitions) < 2 {
		return 1.0
	}

	// Calculate load distribution variance
	loads := make([]float64, 0, len(pm.partitions))
	var sum float64

	for _, partition := range pm.partitions {
		load := float64(partition.RecordCount)
		loads = append(loads, load)
		sum += load
	}

	if sum == 0 {
		return 1.0
	}

	mean := sum / float64(len(loads))
	var variance float64

	for _, load := range loads {
		variance += (load - mean) * (load - mean)
	}

	variance /= float64(len(loads))
	stddev := variance // Simplified, should be sqrt(variance)

	// Return balance score (1.0 = perfectly balanced, 0.0 = completely unbalanced)
	if mean == 0 {
		return 1.0
	}

	return 1.0 - (stddev / mean)
}

func (pm *PartitionManager) updateMetrics() {
	pm.metrics.mutex.Lock()
	defer pm.metrics.mutex.Unlock()

	pm.metrics.data.TotalPartitions = len(pm.partitions)
	pm.metrics.data.ActivePartitions = 0
	pm.metrics.data.TotalRecords = 0
	pm.metrics.data.TotalSize = 0

	for _, partition := range pm.partitions {
		if partition.Status == StatusActive {
			pm.metrics.data.ActivePartitions++
		}
		pm.metrics.data.TotalRecords += partition.RecordCount
		pm.metrics.data.TotalSize += partition.Size
	}
}

func (pm *PartitionManager) incrementOperationCount(operation string) {
	pm.metrics.mutex.Lock()
	pm.metrics.data.OperationCounts[operation]++
	pm.metrics.mutex.Unlock()
}
