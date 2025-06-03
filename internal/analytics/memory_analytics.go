// Package analytics provides memory usage tracking, effectiveness scoring,
// and task completion analytics for the MCP Memory Server.
package analytics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mcp-memory/internal/storage"
	"mcp-memory/pkg/types"
)

const (
	// TaskPriorityHigh represents high priority for tasks
	TaskPriorityHigh = "high"
	// TaskStatusCompleted represents a completed task status
	TaskStatusCompleted = "completed"
)

// MemoryAnalytics handles usage tracking and effectiveness scoring for memories
type MemoryAnalytics struct {
	store storage.VectorStore
	mu    sync.RWMutex

	// In-memory cache for access counts to batch updates
	accessCache map[string]*AccessMetrics
	flushTicker *time.Ticker
}

// AccessMetrics tracks access patterns for a memory chunk
type AccessMetrics struct {
	ChunkID        string
	AccessCount    int
	LastAccessed   time.Time
	SuccessfulUses int
	TotalUses      int
	mu             sync.Mutex
}

// NewMemoryAnalytics creates a new analytics tracker
func NewMemoryAnalytics(store storage.VectorStore) *MemoryAnalytics {
	ma := &MemoryAnalytics{
		store:       store,
		accessCache: make(map[string]*AccessMetrics),
		flushTicker: time.NewTicker(30 * time.Second),
	}

	// Start background flush process
	go ma.flushLoop()

	return ma
}

// RecordAccess tracks when a memory chunk is accessed
func (ma *MemoryAnalytics) RecordAccess(_ context.Context, chunkID string) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	metrics, exists := ma.accessCache[chunkID]
	if !exists {
		metrics = &AccessMetrics{
			ChunkID:      chunkID,
			LastAccessed: time.Now(),
		}
		ma.accessCache[chunkID] = metrics
	}

	metrics.mu.Lock()
	metrics.AccessCount++
	metrics.LastAccessed = time.Now()
	metrics.mu.Unlock()

	return nil
}

// RecordUsage tracks when a memory chunk is used with outcome
func (ma *MemoryAnalytics) RecordUsage(_ context.Context, chunkID string, successful bool) error {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	metrics, exists := ma.accessCache[chunkID]
	if !exists {
		metrics = &AccessMetrics{
			ChunkID:      chunkID,
			LastAccessed: time.Now(),
		}
		ma.accessCache[chunkID] = metrics
	}

	metrics.mu.Lock()
	metrics.TotalUses++
	if successful {
		metrics.SuccessfulUses++
	}
	metrics.mu.Unlock()

	return nil
}

// CalculateEffectivenessScore calculates how effective a memory has been
func (ma *MemoryAnalytics) CalculateEffectivenessScore(chunk *types.ConversationChunk) float64 {
	score := 0.0

	// Check if we have extended metadata for historical usage data
	hasExtendedMetadata := chunk.Metadata.ExtendedMetadata != nil

	// Factor 1: Success rate (40% weight)
	if hasExtendedMetadata {
		successRate, hasSuccessRate := chunk.Metadata.ExtendedMetadata[types.EMKeySuccessRate].(float64)
		if hasSuccessRate {
			score += successRate * 0.4
		} else {
			score += 0.2 // Neutral if unknown
		}
	} else {
		// For new chunks without usage history, estimate based on type and attributes
		score += ma.estimateSuccessRateForNewChunk(chunk) * 0.4
	}

	// Factor 2: Access frequency (20% weight)
	if hasExtendedMetadata {
		accessCount, hasAccessCount := chunk.Metadata.ExtendedMetadata[types.EMKeyAccessCount].(int)
		if hasAccessCount {
			// Logarithmic scale for access count
			accessScore := minFloat64(1.0, float64(accessCount)/10.0)
			score += accessScore * 0.2
		} else {
			score += 0.1 // Neutral if unknown
		}
	} else {
		// For new chunks, give moderate access potential based on type
		score += ma.estimateAccessPotentialForNewChunk(chunk) * 0.2
	}

	// Factor 3: Recency (20% weight)
	score += ma.calculateRecencyScore(chunk, hasExtendedMetadata) * 0.2

	// Factor 4: Problem resolution and task completion (20% weight)
	score += ma.calculateTypeEffectivenessScore(chunk)

	return minFloat64(1.0, score)
}

// estimateSuccessRateForNewChunk estimates success rate for chunks without usage history
func (ma *MemoryAnalytics) estimateSuccessRateForNewChunk(chunk *types.ConversationChunk) float64 {
	switch chunk.Type {
	case types.ChunkTypeTask:
		// High priority completed tasks have very high estimated success
		if chunk.Metadata.TaskStatus != nil && *chunk.Metadata.TaskStatus == TaskStatusCompleted {
			if chunk.Metadata.TaskPriority != nil && *chunk.Metadata.TaskPriority == TaskPriorityHigh {
				return 0.9 // Very high success rate for completed high-priority tasks
			}
			return 0.8 // High success rate for completed tasks
		}
		// Active tasks have moderate success potential
		return 0.6
	case types.ChunkTypeTaskUpdate:
		return 0.7 // Updates are generally valuable
	case types.ChunkTypeTaskProgress:
		// High progress indicates likely success
		if chunk.Metadata.TaskProgress != nil && *chunk.Metadata.TaskProgress >= 80 {
			return 0.8
		}
		return 0.6
	case types.ChunkTypeSolution:
		return 0.8 // Solutions are generally successful
	case types.ChunkTypeArchitectureDecision:
		return 0.7 // Architecture decisions are valuable
	case types.ChunkTypeProblem:
		return 0.4 // Problems without solutions have lower success potential
	case types.ChunkTypeCodeChange:
		return 0.6 // Code changes are moderately successful
	case types.ChunkTypeAnalysis:
		return 0.6 // Analysis is moderately successful
	case types.ChunkTypeVerification:
		return 0.7 // Verification shows completion
	case types.ChunkTypeDiscussion, types.ChunkTypeSessionSummary, types.ChunkTypeQuestion:
		return 0.5 // Neutral for conversational types
	default:
		return 0.5 // Neutral default
	}
}

// estimateAccessPotentialForNewChunk estimates how likely a chunk is to be accessed
func (ma *MemoryAnalytics) estimateAccessPotentialForNewChunk(chunk *types.ConversationChunk) float64 {
	switch chunk.Type {
	case types.ChunkTypeTask:
		// High priority tasks are more likely to be referenced
		if chunk.Metadata.TaskPriority != nil && *chunk.Metadata.TaskPriority == TaskPriorityHigh {
			return 0.8
		}
		return 0.6
	case types.ChunkTypeTaskUpdate:
		return 0.7 // Updates are frequently referenced
	case types.ChunkTypeTaskProgress:
		return 0.6 // Progress tracking is moderately accessed
	case types.ChunkTypeSolution:
		return 0.8 // Solutions are frequently accessed
	case types.ChunkTypeArchitectureDecision:
		return 0.7 // Architecture decisions are valuable references
	case types.ChunkTypeProblem:
		return 0.6 // Problems are moderately accessed for context
	case types.ChunkTypeCodeChange:
		return 0.5 // Code changes have neutral access potential
	case types.ChunkTypeAnalysis:
		return 0.6 // Analysis is moderately accessed
	case types.ChunkTypeVerification:
		return 0.5 // Verification has neutral access potential
	case types.ChunkTypeDiscussion, types.ChunkTypeSessionSummary, types.ChunkTypeQuestion:
		return 0.5 // Neutral for conversational types
	default:
		return 0.5 // Neutral default
	}
}

// calculateTypeEffectivenessScore calculates effectiveness based on chunk type and attributes
func (ma *MemoryAnalytics) calculateTypeEffectivenessScore(chunk *types.ConversationChunk) float64 {
	switch chunk.Type {
	case types.ChunkTypeSolution:
		return 0.2
	case types.ChunkTypeArchitectureDecision:
		return 0.15
	case types.ChunkTypeProblem:
		return 0.05 // Problems are less effective on their own
	case types.ChunkTypeCodeChange:
		return 0.12 // Code changes are moderately valuable
	case types.ChunkTypeAnalysis:
		return 0.12 // Analysis is moderately valuable
	case types.ChunkTypeVerification:
		return 0.15 // Verification shows completion
	case types.ChunkTypeDiscussion, types.ChunkTypeSessionSummary, types.ChunkTypeQuestion:
		return 0.1 // Neutral for conversational types
	// Task-oriented chunk types with enhanced scoring
	case types.ChunkTypeTask:
		baseScore := 0.12 // Increased base score for tasks
		if chunk.Metadata.TaskStatus != nil && *chunk.Metadata.TaskStatus == TaskStatusCompleted {
			baseScore += 0.15 // Higher bonus for completed tasks
			// Additional bonus for high priority completed tasks
			if chunk.Metadata.TaskPriority != nil && *chunk.Metadata.TaskPriority == TaskPriorityHigh {
				baseScore += 0.08
			}
		} else if chunk.Metadata.TaskPriority != nil && *chunk.Metadata.TaskPriority == TaskPriorityHigh {
			baseScore += 0.05 // Bonus for high priority tasks even if not completed
		}
		return baseScore
	case types.ChunkTypeTaskUpdate:
		return 0.15 // Increased score for updates - they show engagement and progress
	case types.ChunkTypeTaskProgress:
		baseScore := 0.12 // Increased base score for progress tracking
		if chunk.Metadata.TaskProgress != nil && *chunk.Metadata.TaskProgress >= 80 {
			baseScore += 0.10 // Higher bonus for high progress
		}
		return baseScore
	default:
		return 0.1 // Default for unknown types
	}
}

// calculateRecencyScore calculates the recency component of effectiveness score
func (ma *MemoryAnalytics) calculateRecencyScore(chunk *types.ConversationChunk, hasExtendedMetadata bool) float64 {
	if !hasExtendedMetadata {
		// For new chunks, use creation time recency
		daysSince := time.Since(chunk.Timestamp).Hours() / 24
		recency := 1.0 - daysSince/30.0
		if recency < 0 {
			return 0
		}
		return recency
	}

	lastAccessed, hasLastAccessed := chunk.Metadata.ExtendedMetadata[types.EMKeyLastAccessed].(string)
	if !hasLastAccessed {
		// Use creation time if no access time
		daysSince := time.Since(chunk.Timestamp).Hours() / 24
		recency := 1.0 - daysSince/30.0
		if recency < 0 {
			return 0
		}
		return recency
	}

	lastTime, err := time.Parse(time.RFC3339, lastAccessed)
	if err != nil {
		// Fallback to creation time on parse error
		daysSince := time.Since(chunk.Timestamp).Hours() / 24
		recency := 1.0 - daysSince/30.0
		if recency < 0 {
			return 0
		}
		return recency
	}

	daysSince := time.Since(lastTime).Hours() / 24
	recency := 1.0 - daysSince/30.0 // Decay over 30 days
	if recency < 0 {
		return 0
	}
	return recency
}

// UpdateChunkAnalytics updates analytics metadata for a chunk
func (ma *MemoryAnalytics) UpdateChunkAnalytics(ctx context.Context, chunkID string) error {
	chunk, err := ma.store.GetByID(ctx, chunkID)
	if err != nil {
		return fmt.Errorf("failed to get chunk: %w", err)
	}

	// Initialize extended metadata if needed
	if chunk.Metadata.ExtendedMetadata == nil {
		chunk.Metadata.ExtendedMetadata = make(map[string]interface{})
	}

	// Get cached metrics
	ma.mu.RLock()
	metrics, exists := ma.accessCache[chunkID]
	ma.mu.RUnlock()

	if exists {
		metrics.mu.Lock()

		// Update access count
		currentCount, _ := chunk.Metadata.ExtendedMetadata[types.EMKeyAccessCount].(int)
		chunk.Metadata.ExtendedMetadata[types.EMKeyAccessCount] = currentCount + metrics.AccessCount

		// Update last accessed
		chunk.Metadata.ExtendedMetadata[types.EMKeyLastAccessed] = metrics.LastAccessed.Format(time.RFC3339)

		// Update success rate
		if metrics.TotalUses > 0 {
			successRate := float64(metrics.SuccessfulUses) / float64(metrics.TotalUses)
			chunk.Metadata.ExtendedMetadata[types.EMKeySuccessRate] = successRate
		}

		metrics.mu.Unlock()
	}

	// Calculate and update effectiveness score
	effectivenessScore := ma.CalculateEffectivenessScore(chunk)
	chunk.Metadata.ExtendedMetadata[types.EMKeyEffectivenessScore] = effectivenessScore

	// Update the chunk in storage
	return ma.store.Update(ctx, *chunk)
}

// MarkObsolete marks a memory as obsolete
func (ma *MemoryAnalytics) MarkObsolete(ctx context.Context, chunkID, reason string) error {
	chunk, err := ma.store.GetByID(ctx, chunkID)
	if err != nil {
		return fmt.Errorf("failed to get chunk: %w", err)
	}

	if chunk.Metadata.ExtendedMetadata == nil {
		chunk.Metadata.ExtendedMetadata = make(map[string]interface{})
	}

	chunk.Metadata.ExtendedMetadata[types.EMKeyIsObsolete] = true
	chunk.Metadata.ExtendedMetadata[types.EMKeyArchivedAt] = time.Now().Format(time.RFC3339)
	chunk.Metadata.ExtendedMetadata["obsolete_reason"] = reason

	return ma.store.Update(ctx, *chunk)
}

// GetTopMemories returns the most effective memories
func (ma *MemoryAnalytics) GetTopMemories(ctx context.Context, repository string, limit int) ([]types.ConversationChunk, error) {
	// Search for chunks in the repository
	chunks, err := ma.store.ListByRepository(ctx, repository, limit*2, 0) // Get extra to filter
	if err != nil {
		return nil, fmt.Errorf("failed to search chunks: %w", err)
	}

	// Calculate scores and sort
	type scoredChunk struct {
		chunk types.ConversationChunk
		score float64
	}

	scored := make([]scoredChunk, 0, len(chunks))
	for i := range chunks {
		// Skip obsolete chunks
		if obsolete, ok := chunks[i].Metadata.ExtendedMetadata[types.EMKeyIsObsolete].(bool); ok && obsolete {
			continue
		}

		score := ma.CalculateEffectivenessScore(&chunks[i])
		scored = append(scored, scoredChunk{chunk: chunks[i], score: score})
	}

	// Sort by score descending
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Return top N
	result := make([]types.ConversationChunk, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		result = append(result, scored[i].chunk)
	}

	return result, nil
}

// flushLoop periodically flushes cached metrics to storage
func (ma *MemoryAnalytics) flushLoop() {
	for range ma.flushTicker.C {
		ma.flush()
	}
}

// flush writes cached metrics to storage
func (ma *MemoryAnalytics) flush() {
	ctx := context.Background()

	ma.mu.Lock()
	// Copy and clear cache
	toFlush := make(map[string]*AccessMetrics)
	for k, v := range ma.accessCache {
		toFlush[k] = v
		delete(ma.accessCache, k)
	}
	ma.mu.Unlock()

	// Update each chunk
	for chunkID := range toFlush {
		if err := ma.UpdateChunkAnalytics(ctx, chunkID); err != nil {
			// Log error but continue with other chunks
			fmt.Printf("Failed to update analytics for chunk %s: %v\n", chunkID, err)
		}
	}
}

// Stop stops the analytics tracker
func (ma *MemoryAnalytics) Stop() {
	ma.flushTicker.Stop()
	ma.flush() // Final flush
}

// Helper functions
func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
