package intelligence

import (
	"context"
	"errors"
	"math"
	"mcp-memory/pkg/types"
	"strings"
	"time"
)

// ConfidenceEngine calculates confidence scores for memory chunks and relationships
type ConfidenceEngine struct {
	storage StorageInterface
}

// StorageInterface defines the interface for intelligence operations
type StorageInterface interface {
	GetByID(ctx context.Context, id string) (*types.ConversationChunk, error)
	GetRelationships(ctx context.Context, query types.RelationshipQuery) ([]types.RelationshipResult, error)
	Search(ctx context.Context, query types.MemoryQuery, embeddings []float64) (*types.SearchResults, error)
	ListByRepository(ctx context.Context, repository string, limit, offset int) ([]types.ConversationChunk, error)
	Update(ctx context.Context, chunk types.ConversationChunk) error
}

// NewConfidenceEngine creates a new confidence scoring engine
func NewConfidenceEngine(storage StorageInterface) *ConfidenceEngine {
	return &ConfidenceEngine{
		storage: storage,
	}
}

// ConfidenceConfig configures confidence calculation parameters
type ConfidenceConfig struct {
	// Weights for different factors (should sum to 1.0)
	UserCertaintyWeight       float64 `json:"user_certainty_weight"`
	ConsistencyWeight         float64 `json:"consistency_weight"`
	CorroborationWeight       float64 `json:"corroboration_weight"`
	SemanticSimilarityWeight  float64 `json:"semantic_similarity_weight"`
	TemporalProximityWeight   float64 `json:"temporal_proximity_weight"`
	ContextualRelevanceWeight float64 `json:"contextual_relevance_weight"`

	// Thresholds
	MinCorroborationCount int           `json:"min_corroboration_count"`
	MaxTemporalDistance   time.Duration `json:"max_temporal_distance"`
	MinSemanticSimilarity float64       `json:"min_semantic_similarity"`
	DecayRate             float64       `json:"decay_rate"` // How much confidence decays over time
}

// DefaultConfidenceConfig returns a sensible default configuration
func DefaultConfidenceConfig() *ConfidenceConfig {
	return &ConfidenceConfig{
		UserCertaintyWeight:       0.30,
		ConsistencyWeight:         0.25,
		CorroborationWeight:       0.20,
		SemanticSimilarityWeight:  0.15,
		TemporalProximityWeight:   0.05,
		ContextualRelevanceWeight: 0.05,

		MinCorroborationCount: 1,
		MaxTemporalDistance:   7 * 24 * time.Hour, // 7 days
		MinSemanticSimilarity: 0.3,
		DecayRate:             0.1, // 10% decay per month
	}
}

// CalculateChunkConfidence calculates confidence score for a memory chunk
func (ce *ConfidenceEngine) CalculateChunkConfidence(ctx context.Context, chunk *types.ConversationChunk, config *ConfidenceConfig) (*types.ConfidenceMetrics, error) {
	if config == nil {
		config = DefaultConfidenceConfig()
	}

	factors := types.ConfidenceFactors{}
	var totalScore float64
	var weightSum float64

	// 1. User certainty (if available from metadata)
	if chunk.Metadata.Confidence != nil && chunk.Metadata.Confidence.Factors.UserCertainty != nil {
		userCertainty := *chunk.Metadata.Confidence.Factors.UserCertainty
		factors.UserCertainty = &userCertainty
		totalScore += userCertainty * config.UserCertaintyWeight
		weightSum += config.UserCertaintyWeight
	}

	// 2. Consistency score (how consistent with similar memories)
	consistencyScore, err := ce.calculateConsistencyScore(ctx, chunk)
	if err == nil {
		factors.ConsistencyScore = &consistencyScore
		totalScore += consistencyScore * config.ConsistencyWeight
		weightSum += config.ConsistencyWeight
	}

	// 3. Corroboration count (how many other memories support this)
	corroborationCount, corroborationScore := ce.calculateCorroborationScore(ctx, chunk, config)
	factors.CorroborationCount = &corroborationCount
	totalScore += corroborationScore * config.CorroborationWeight
	weightSum += config.CorroborationWeight

	// 4. Semantic similarity with related content
	semanticSimilarity, err := ce.calculateSemanticSimilarityScore(ctx, chunk)
	if err == nil {
		factors.SemanticSimilarity = &semanticSimilarity
		totalScore += semanticSimilarity * config.SemanticSimilarityWeight
		weightSum += config.SemanticSimilarityWeight
	}

	// 5. Temporal proximity (recent memories get higher confidence)
	temporalProximity := ce.calculateTemporalProximityScore(chunk, config)
	factors.TemporalProximity = &temporalProximity
	totalScore += temporalProximity * config.TemporalProximityWeight
	weightSum += config.TemporalProximityWeight

	// 6. Contextual relevance (how relevant to the repository/session)
	contextualRelevance := ce.calculateContextualRelevanceScore(ctx, chunk)
	factors.ContextualRelevance = &contextualRelevance
	totalScore += contextualRelevance * config.ContextualRelevanceWeight
	weightSum += config.ContextualRelevanceWeight

	// Calculate final confidence score
	var finalScore float64
	if weightSum > 0 {
		finalScore = totalScore / weightSum
	} else {
		finalScore = 0.5 // Default if no factors available
	}

	// Apply time-based decay
	finalScore = ce.applyTimeDecay(finalScore, chunk.Timestamp, config.DecayRate)

	// Ensure score is within bounds
	finalScore = math.Max(0.0, math.Min(1.0, finalScore))

	confidence := &types.ConfidenceMetrics{
		Score:           finalScore,
		Source:          "calculated",
		Factors:         factors,
		ValidationCount: 0,
	}

	now := time.Now().UTC()
	confidence.LastUpdated = &now

	return confidence, nil
}

// CalculateQualityMetrics calculates quality metrics for a memory chunk
func (ce *ConfidenceEngine) CalculateQualityMetrics(ctx context.Context, chunk *types.ConversationChunk) (*types.QualityMetrics, error) {
	quality := &types.QualityMetrics{}

	// 1. Completeness - based on content length and structure
	quality.Completeness = ce.calculateCompletenessScore(chunk)

	// 2. Clarity - based on content structure and readability
	quality.Clarity = ce.calculateClarityScore(chunk)

	// 3. Relevance decay - how much relevance has decayed over time
	quality.RelevanceDecay = ce.calculateRelevanceDecay(chunk)

	// 4. Freshness score - how recent and up-to-date
	quality.FreshnessScore = ce.calculateFreshnessScore(chunk)

	// 5. Usage score - based on access patterns (would need tracking)
	quality.UsageScore = ce.calculateUsageScore(chunk)

	// Calculate overall quality
	quality.CalculateOverallQuality()

	return quality, nil
}

// Helper methods for confidence calculation

// calculateConsistencyScore measures how consistent this chunk is with similar memories
func (ce *ConfidenceEngine) calculateConsistencyScore(_ context.Context, chunk *types.ConversationChunk) (float64, error) {
	// Validate input
	if chunk == nil {
		return 0, errors.New("chunk cannot be nil")
	}
	if len(chunk.Content) == 0 {
		return 0, errors.New("chunk content cannot be empty")
	}

	// Find similar chunks to compare against
	query := types.NewMemoryQuery(chunk.Content[:minInt(100, len(chunk.Content))])
	query.Repository = &chunk.Metadata.Repository
	query.Types = []types.ChunkType{chunk.Type}
	query.Limit = 10

	// Note: This would need embeddings service integration
	// For now, return a default score based on chunk type and outcome
	switch chunk.Metadata.Outcome {
	case types.OutcomeSuccess:
		return 0.8, nil
	case types.OutcomeInProgress:
		return 0.6, nil
	case types.OutcomeFailed:
		return 0.4, nil
	case types.OutcomeAbandoned:
		return 0.3, nil
	default:
		return 0.5, nil
	}
}

// calculateCorroborationScore counts supporting memories and converts to score
func (ce *ConfidenceEngine) calculateCorroborationScore(_ context.Context, chunk *types.ConversationChunk, config *ConfidenceConfig) (int, float64) {
	// Count related chunks in the same session or repository
	count := 0

	// Count by tags overlap
	if len(chunk.Metadata.Tags) > 0 {
		count += len(chunk.Metadata.Tags) // Simple heuristic
	}

	// Count by session context
	if chunk.SessionID != "" {
		count += 2 // Assume some session context
	}

	// Convert count to score (0.0-1.0)
	score := float64(count) / float64(config.MinCorroborationCount+3)
	return count, math.Min(1.0, score)
}

// calculateSemanticSimilarityScore measures semantic similarity with related content
func (ce *ConfidenceEngine) calculateSemanticSimilarityScore(_ context.Context, chunk *types.ConversationChunk) (float64, error) {
	// Validate input
	if chunk == nil {
		return 0, errors.New("chunk cannot be nil")
	}
	if len(chunk.Content) == 0 {
		return 0, errors.New("chunk content cannot be empty")
	}

	// This would need embeddings service integration
	// For now, return a score based on content characteristics

	score := 0.5 // Base score

	// Boost for detailed content
	if len(chunk.Content) > 200 {
		score += 0.2
	}

	// Boost for structured content (code, error messages, etc.)
	if strings.Contains(chunk.Content, "```") || strings.Contains(chunk.Content, "error:") {
		score += 0.2
	}

	// Boost for having a summary
	if chunk.Summary != "" && len(chunk.Summary) > 20 {
		score += 0.1
	}

	return math.Min(1.0, score), nil
}

// calculateTemporalProximityScore gives higher scores to recent memories
func (ce *ConfidenceEngine) calculateTemporalProximityScore(chunk *types.ConversationChunk, config *ConfidenceConfig) float64 {
	timeSince := time.Since(chunk.Timestamp)

	if timeSince < 0 {
		return 1.0 // Future timestamp, give full score
	}

	if timeSince > config.MaxTemporalDistance {
		return 0.1 // Very old, minimal score
	}

	// Linear decay within the max distance
	ratio := float64(timeSince) / float64(config.MaxTemporalDistance)
	return 1.0 - ratio
}

// calculateContextualRelevanceScore measures relevance to current context
func (ce *ConfidenceEngine) calculateContextualRelevanceScore(_ context.Context, chunk *types.ConversationChunk) float64 {
	score := 0.5 // Base score

	// Boost for having repository context
	if chunk.Metadata.Repository != "" && chunk.Metadata.Repository != "_global" {
		score += 0.3
	}

	// Boost for having file modifications
	if len(chunk.Metadata.FilesModified) > 0 {
		score += 0.1
	}

	// Boost for having tools used
	if len(chunk.Metadata.ToolsUsed) > 0 {
		score += 0.1
	}

	return math.Min(1.0, score)
}

// applyTimeDecay applies exponential decay based on age
func (ce *ConfidenceEngine) applyTimeDecay(score float64, timestamp time.Time, decayRate float64) float64 {
	monthsOld := time.Since(timestamp).Hours() / (24 * 30) // Approximate months
	decayFactor := math.Exp(-decayRate * monthsOld)
	return score * decayFactor
}

// Helper methods for quality calculation

// calculateCompletenessScore measures how complete the memory is
func (ce *ConfidenceEngine) calculateCompletenessScore(chunk *types.ConversationChunk) float64 {
	score := 0.0

	// Content length score
	if len(chunk.Content) > 100 {
		score += 0.3
	}
	if len(chunk.Content) > 500 {
		score += 0.2
	}

	// Has summary
	if chunk.Summary != "" {
		score += 0.2
	}

	// Has metadata
	if len(chunk.Metadata.Tags) > 0 {
		score += 0.1
	}
	if len(chunk.Metadata.FilesModified) > 0 {
		score += 0.1
	}
	if len(chunk.Metadata.ToolsUsed) > 0 {
		score += 0.1
	}

	return math.Min(1.0, score)
}

// calculateClarityScore measures how clear and unambiguous the memory is
func (ce *ConfidenceEngine) calculateClarityScore(chunk *types.ConversationChunk) float64 {
	score := 0.5 // Base score

	// Boost for structured content
	if strings.Contains(chunk.Content, "```") {
		score += 0.2
	}

	// Boost for clear problem/solution structure
	lowerContent := strings.ToLower(chunk.Content)
	if strings.Contains(lowerContent, "problem:") || strings.Contains(lowerContent, "solution:") {
		score += 0.2
	}

	// Boost for having outcome
	if chunk.Metadata.Outcome == types.OutcomeSuccess {
		score += 0.1
	}

	return math.Min(1.0, score)
}

// calculateRelevanceDecay measures how much relevance has decayed
func (ce *ConfidenceEngine) calculateRelevanceDecay(chunk *types.ConversationChunk) float64 {
	daysSince := time.Since(chunk.Timestamp).Hours() / 24

	// Technology-specific decay rates
	techDecayRate := 0.001 // Default: 0.1% per day

	// Faster decay for rapidly changing tech
	lowerContent := strings.ToLower(chunk.Content)
	if strings.Contains(lowerContent, "npm") || strings.Contains(lowerContent, "node") {
		techDecayRate = 0.003 // 0.3% per day for Node.js ecosystem
	}
	if strings.Contains(lowerContent, "kubernetes") || strings.Contains(lowerContent, "docker") {
		techDecayRate = 0.002 // 0.2% per day for container tech
	}

	decay := daysSince * techDecayRate
	return math.Min(1.0, decay)
}

// calculateFreshnessScore measures how fresh and current the memory is
func (ce *ConfidenceEngine) calculateFreshnessScore(chunk *types.ConversationChunk) float64 {
	daysSince := time.Since(chunk.Timestamp).Hours() / 24

	if daysSince < 1 {
		return 1.0 // Very fresh
	}
	if daysSince < 7 {
		return 0.9 // Fresh
	}
	if daysSince < 30 {
		return 0.7 // Recent
	}
	if daysSince < 90 {
		return 0.5 // Moderately old
	}
	if daysSince < 365 {
		return 0.3 // Old
	}

	return 0.1 // Very old
}

// calculateUsageScore measures usage patterns (would need access tracking)
func (ce *ConfidenceEngine) calculateUsageScore(chunk *types.ConversationChunk) float64 {
	// This would require usage tracking in the storage layer
	// For now, return a default based on chunk characteristics

	score := 0.5 // Base score

	// Boost for successful outcomes (likely to be referenced)
	if chunk.Metadata.Outcome == types.OutcomeSuccess {
		score += 0.3
	}

	// Boost for problem/solution chunks (highly reusable)
	if chunk.Type == types.ChunkTypeSolution || chunk.Type == types.ChunkTypeProblem {
		score += 0.2
	}

	return math.Min(1.0, score)
}

// Utility functions
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
