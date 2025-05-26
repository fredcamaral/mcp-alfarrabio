// Package decay implements memory decay with smart summarization
package decay

import (
	"context"
	"fmt"
	"log"
	"math"
	"mcp-memory/pkg/types"
	"sort"
	"sync"
	"time"
)

// DecayStrategy defines how memories decay over time
type DecayStrategy string

const (
	DecayStrategyExponential DecayStrategy = "exponential"
	DecayStrategyLinear      DecayStrategy = "linear"
	DecayStrategyAdaptive    DecayStrategy = "adaptive"
)

// DecayConfig holds configuration for memory decay
type DecayConfig struct {
	// Strategy for decay calculation
	Strategy DecayStrategy
	
	// BaseDecayRate is the rate at which memories decay (0.0 to 1.0)
	BaseDecayRate float64
	
	// MinRelevanceScore below which memories are candidates for summarization
	MinRelevanceScore float64
	
	// SummarizationThreshold - memories below this score get summarized
	SummarizationThreshold float64
	
	// DeletionThreshold - memories below this score get deleted
	DeletionThreshold float64
	
	// ImportanceBoost - multiplier for important memories
	ImportanceBoost map[string]float64
	
	// DecayInterval - how often to run decay process
	DecayInterval time.Duration
	
	// RetentionPeriod - minimum time to keep memories
	RetentionPeriod time.Duration
}

// DefaultDecayConfig returns default decay configuration
func DefaultDecayConfig() *DecayConfig {
	return &DecayConfig{
		Strategy:               DecayStrategyAdaptive,
		BaseDecayRate:          0.1, // 10% decay per interval
		MinRelevanceScore:      0.7,
		SummarizationThreshold: 0.4,
		DeletionThreshold:      0.1,
		ImportanceBoost: map[string]float64{
			"decision":    2.0,
			"problem":     1.5,
			"solution":    1.8,
			"learning":    1.6,
			"error":       1.7,
		},
		DecayInterval:   24 * time.Hour,
		RetentionPeriod: 7 * 24 * time.Hour,
	}
}

// MemoryDecayManager manages the decay of memories over time
type MemoryDecayManager struct {
	config      *DecayConfig
	store       MemoryStore
	summarizer  Summarizer
	mu          sync.RWMutex
	running     bool
	stopCh      chan struct{}
	lastDecayRun time.Time
}

// MemoryStore interface for accessing memories
type MemoryStore interface {
	GetAllChunks(ctx context.Context, repository string) ([]types.ConversationChunk, error)
	UpdateChunk(ctx context.Context, chunk types.ConversationChunk) error
	DeleteChunk(ctx context.Context, chunkID string) error
	StoreChunk(ctx context.Context, chunk types.ConversationChunk) error
}

// Summarizer interface for creating summaries
type Summarizer interface {
	Summarize(ctx context.Context, chunks []types.ConversationChunk) (string, error)
	SummarizeChain(ctx context.Context, chunks []types.ConversationChunk) (types.ConversationChunk, error)
}

// NewMemoryDecayManager creates a new memory decay manager
func NewMemoryDecayManager(config *DecayConfig, store MemoryStore, summarizer Summarizer) *MemoryDecayManager {
	if config == nil {
		config = DefaultDecayConfig()
	}
	
	return &MemoryDecayManager{
		config:     config,
		store:      store,
		summarizer: summarizer,
		stopCh:     make(chan struct{}),
	}
}

// Start begins the decay process
func (m *MemoryDecayManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("decay manager already running")
	}
	m.running = true
	m.mu.Unlock()
	
	go m.runDecayLoop(ctx)
	return nil
}

// Stop stops the decay process
func (m *MemoryDecayManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.running {
		close(m.stopCh)
		m.running = false
	}
}

// runDecayLoop runs the decay process periodically
func (m *MemoryDecayManager) runDecayLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.DecayInterval)
	defer ticker.Stop()
	
	// Run initial decay
	if err := m.RunDecay(ctx, ""); err != nil {
		log.Printf("Initial decay failed: %v", err)
	}
	
	for {
		select {
		case <-ticker.C:
			if err := m.RunDecay(ctx, ""); err != nil {
				log.Printf("Decay process failed: %v", err)
			}
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// RunDecay runs the decay process for a repository
func (m *MemoryDecayManager) RunDecay(ctx context.Context, repository string) error {
	m.mu.Lock()
	m.lastDecayRun = time.Now()
	m.mu.Unlock()
	
	// Get all chunks
	chunks, err := m.store.GetAllChunks(ctx, repository)
	if err != nil {
		return fmt.Errorf("failed to get chunks: %w", err)
	}
	
	// Calculate relevance scores
	scoredChunks := m.calculateRelevanceScores(chunks)
	
	// Group chunks for summarization
	toSummarize := make([]ScoredChunk, 0)
	toDelete := make([]string, 0)
	toUpdate := make([]types.ConversationChunk, 0)
	
	for _, sc := range scoredChunks {
		// Skip recent memories
		if time.Since(sc.Chunk.Timestamp) < m.config.RetentionPeriod {
			continue
		}
		
		if sc.Score < m.config.DeletionThreshold {
			toDelete = append(toDelete, sc.Chunk.ID)
		} else if sc.Score < m.config.SummarizationThreshold {
			toSummarize = append(toSummarize, sc)
		} else if sc.Score < m.config.MinRelevanceScore {
			// Update with decayed score
			chunk := sc.Chunk
			// Store decay info in a separate tracking system since Metadata is structured
			// For now, just add to update list
			toUpdate = append(toUpdate, chunk)
		}
	}
	
	// Summarize chunks
	if len(toSummarize) > 0 {
		if err := m.summarizeChunks(ctx, toSummarize); err != nil {
			log.Printf("Failed to summarize chunks: %v", err)
		}
	}
	
	// Update chunks with new scores
	for _, chunk := range toUpdate {
		if err := m.store.UpdateChunk(ctx, chunk); err != nil {
			log.Printf("Failed to update chunk %s: %v", chunk.ID, err)
		}
	}
	
	// Delete old chunks
	for _, id := range toDelete {
		if err := m.store.DeleteChunk(ctx, id); err != nil {
			log.Printf("Failed to delete chunk %s: %v", id, err)
		}
	}
	
	log.Printf("Decay process completed: %d summarized, %d updated, %d deleted",
		len(toSummarize), len(toUpdate), len(toDelete))
	
	return nil
}

// ScoredChunk holds a chunk with its relevance score
type ScoredChunk struct {
	Chunk types.ConversationChunk
	Score float64
}

// calculateRelevanceScores calculates relevance scores for all chunks
func (m *MemoryDecayManager) calculateRelevanceScores(chunks []types.ConversationChunk) []ScoredChunk {
	scored := make([]ScoredChunk, len(chunks))
	
	for i, chunk := range chunks {
		score := m.calculateChunkScore(chunk)
		scored[i] = ScoredChunk{
			Chunk: chunk,
			Score: score,
		}
	}
	
	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	
	return scored
}

// calculateChunkScore calculates the relevance score for a chunk
func (m *MemoryDecayManager) calculateChunkScore(chunk types.ConversationChunk) float64 {
	// Base score starts at 1.0
	score := 1.0
	
	// Apply time decay
	age := time.Since(chunk.Timestamp)
	score = m.applyTimeDecay(score, age)
	
	// Apply importance boost
	if boost, exists := m.config.ImportanceBoost[string(chunk.Type)]; exists {
		score *= boost
	}
	
	// Since Metadata is structured, we can't store dynamic fields
	// Use TimeSpent as a proxy for importance
	if chunk.Metadata.TimeSpent != nil && *chunk.Metadata.TimeSpent > 0 {
		// Boost score based on time spent (more time = more important)
		timeBoost := 1.0 + float64(*chunk.Metadata.TimeSpent)/60.0 // 1% boost per minute
		score *= math.Min(timeBoost, 2.0) // Cap at 2x boost
	}
	
	// Consider relationships
	if len(chunk.RelatedChunks) > 0 {
		score *= (1.0 + float64(len(chunk.RelatedChunks))/10.0)
	}
	
	// Ensure score stays within bounds
	return math.Max(0.0, math.Min(1.0, score))
}

// applyTimeDecay applies time-based decay to a score
func (m *MemoryDecayManager) applyTimeDecay(score float64, age time.Duration) float64 {
	days := age.Hours() / 24.0
	
	switch m.config.Strategy {
	case DecayStrategyLinear:
		// Linear decay
		decayFactor := 1.0 - (m.config.BaseDecayRate * days / 30.0)
		return score * math.Max(0.0, decayFactor)
		
	case DecayStrategyExponential:
		// Exponential decay with half-life of 30 days
		halfLife := 30.0
		return score * math.Pow(0.5, days/halfLife)
		
	case DecayStrategyAdaptive:
		// Adaptive decay - slower for first week, then accelerates
		if days < 7 {
			// Minimal decay in first week
			return score * (1.0 - m.config.BaseDecayRate*0.1*days/7.0)
		} else if days < 30 {
			// Moderate decay for first month
			return score * (0.9 - m.config.BaseDecayRate*0.3*(days-7)/23.0)
		} else {
			// Accelerated decay after a month
			return score * math.Pow(0.6, (days-30)/30.0)
		}
		
	default:
		return score
	}
}

// summarizeChunks creates summaries for chunks marked for summarization
func (m *MemoryDecayManager) summarizeChunks(ctx context.Context, chunks []ScoredChunk) error {
	// Group related chunks
	groups := m.groupRelatedChunks(chunks)
	
	for _, group := range groups {
		if len(group) < 2 {
			// Don't summarize single chunks
			continue
		}
		
		// Extract the actual chunks
		groupChunks := make([]types.ConversationChunk, len(group))
		for i, sc := range group {
			groupChunks[i] = sc.Chunk
		}
		
		// Create summary
		summary, err := m.summarizer.SummarizeChain(ctx, groupChunks)
		if err != nil {
			log.Printf("Failed to summarize group: %v", err)
			continue
		}
		
		// Store summary
		if err := m.store.StoreChunk(ctx, summary); err != nil {
			log.Printf("Failed to store summary: %v", err)
			continue
		}
		
		// Delete original chunks
		for _, sc := range group {
			if err := m.store.DeleteChunk(ctx, sc.Chunk.ID); err != nil {
				log.Printf("Failed to delete summarized chunk %s: %v", sc.Chunk.ID, err)
			}
		}
	}
	
	return nil
}

// groupRelatedChunks groups chunks that should be summarized together
func (m *MemoryDecayManager) groupRelatedChunks(chunks []ScoredChunk) [][]ScoredChunk {
	// Simple grouping by session and time proximity
	groups := make(map[string][]ScoredChunk)
	
	for _, chunk := range chunks {
		key := chunk.Chunk.SessionID
		groups[key] = append(groups[key], chunk)
	}
	
	// Convert to slice
	result := make([][]ScoredChunk, 0, len(groups))
	for _, group := range groups {
		// Further split by time gaps
		subgroups := m.splitByTimeGaps(group, 4*time.Hour)
		result = append(result, subgroups...)
	}
	
	return result
}

// splitByTimeGaps splits chunks into groups based on time gaps
func (m *MemoryDecayManager) splitByTimeGaps(chunks []ScoredChunk, maxGap time.Duration) [][]ScoredChunk {
	if len(chunks) == 0 {
		return nil
	}
	
	// Sort by timestamp
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Chunk.Timestamp.Before(chunks[j].Chunk.Timestamp)
	})
	
	groups := [][]ScoredChunk{}
	currentGroup := []ScoredChunk{chunks[0]}
	
	for i := 1; i < len(chunks); i++ {
		gap := chunks[i].Chunk.Timestamp.Sub(chunks[i-1].Chunk.Timestamp)
		if gap > maxGap {
			// Start new group
			groups = append(groups, currentGroup)
			currentGroup = []ScoredChunk{chunks[i]}
		} else {
			currentGroup = append(currentGroup, chunks[i])
		}
	}
	
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}
	
	return groups
}

// GetDecayStats returns statistics about the decay process
func (m *MemoryDecayManager) GetDecayStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]interface{}{
		"running":        m.running,
		"last_decay_run": m.lastDecayRun,
		"config":         m.config,
	}
}