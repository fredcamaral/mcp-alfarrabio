package analytics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"mcp-memory/internal/storage"
	"mcp-memory/pkg/types"
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
func (ma *MemoryAnalytics) RecordAccess(ctx context.Context, chunkID string) error {
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
func (ma *MemoryAnalytics) RecordUsage(ctx context.Context, chunkID string, successful bool) error {
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

	// Get extended metadata
	if chunk.Metadata.ExtendedMetadata == nil {
		return 0.5 // Default neutral score
	}

	// Factor 1: Success rate (40% weight)
	successRate, hasSuccessRate := chunk.Metadata.ExtendedMetadata[types.EMKeySuccessRate].(float64)
	if hasSuccessRate {
		score += successRate * 0.4
	} else {
		score += 0.2 // Neutral if unknown
	}

	// Factor 2: Access frequency (20% weight)
	accessCount, hasAccessCount := chunk.Metadata.ExtendedMetadata[types.EMKeyAccessCount].(int)
	if hasAccessCount {
		// Logarithmic scale for access count
		accessScore := min(1.0, float64(accessCount)/10.0)
		score += accessScore * 0.2
	} else {
		score += 0.1 // Neutral if unknown
	}

	// Factor 3: Recency (20% weight)
	lastAccessed, hasLastAccessed := chunk.Metadata.ExtendedMetadata[types.EMKeyLastAccessed].(string)
	if hasLastAccessed {
		if lastTime, err := time.Parse(time.RFC3339, lastAccessed); err == nil {
			daysSince := time.Since(lastTime).Hours() / 24
			recencyScore := max(0, 1.0-daysSince/30.0) // Decay over 30 days
			score += recencyScore * 0.2
		}
	} else {
		// Use creation time if no access time
		daysSince := time.Since(chunk.Timestamp).Hours() / 24
		recencyScore := max(0, 1.0-daysSince/30.0)
		score += recencyScore * 0.2
	}

	// Factor 4: Problem resolution (20% weight)
	switch chunk.Type {
	case types.ChunkTypeSolution:
		score += 0.2
	case types.ChunkTypeArchitectureDecision:
		score += 0.15
	case types.ChunkTypeProblem:
		score += 0.05 // Problems are less effective on their own
	case types.ChunkTypeCodeChange:
		score += 0.12 // Code changes are moderately valuable
	case types.ChunkTypeAnalysis:
		score += 0.12 // Analysis is moderately valuable
	case types.ChunkTypeVerification:
		score += 0.15 // Verification shows completion
	case types.ChunkTypeDiscussion, types.ChunkTypeSessionSummary, types.ChunkTypeQuestion:
		score += 0.1 // Neutral for conversational types
	}

	return min(1.0, score)
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
func (ma *MemoryAnalytics) MarkObsolete(ctx context.Context, chunkID string, reason string) error {
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
	for _, chunk := range chunks {
		// Skip obsolete chunks
		if obsolete, ok := chunk.Metadata.ExtendedMetadata[types.EMKeyIsObsolete].(bool); ok && obsolete {
			continue
		}

		score := ma.CalculateEffectivenessScore(&chunk)
		scored = append(scored, scoredChunk{chunk: chunk, score: score})
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
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
