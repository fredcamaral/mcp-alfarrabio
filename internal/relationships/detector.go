package relationships

import (
	"context"
	"fmt"
	"mcp-memory/pkg/types"
	"regexp"
	"strings"
	"time"
)

// RelationshipDetector analyzes chunks to automatically detect relationships
type RelationshipDetector struct {
	storage StorageInterface
}

// StorageInterface defines the interface for storage operations needed by the detector
type StorageInterface interface {
	GetByID(ctx context.Context, id string) (*types.ConversationChunk, error)
	StoreRelationship(ctx context.Context, sourceID, targetID string, relationType types.RelationType, confidence float64, source types.ConfidenceSource) (*types.MemoryRelationship, error)
	GetRelationships(ctx context.Context, query types.RelationshipQuery) ([]types.RelationshipResult, error)
}

// NewRelationshipDetector creates a new relationship detector
func NewRelationshipDetector(storage StorageInterface) *RelationshipDetector {
	return &RelationshipDetector{
		storage: storage,
	}
}

// DetectionConfig configures the relationship detection process
type DetectionConfig struct {
	MinConfidence           float64               `json:"min_confidence"`
	MaxTimeDistance         time.Duration         `json:"max_time_distance"`
	SemanticSimilarityThreshold float64           `json:"semantic_similarity_threshold"`
	EnabledDetectors        []string              `json:"enabled_detectors"`
	RelationshipConfidence  map[types.RelationType]float64 `json:"relationship_confidence"`
}

// DefaultDetectionConfig returns a sensible default configuration
func DefaultDetectionConfig() *DetectionConfig {
	return &DetectionConfig{
		MinConfidence:           0.6,
		MaxTimeDistance:         24 * time.Hour,
		SemanticSimilarityThreshold: 0.7,
		EnabledDetectors:        []string{"temporal", "causal", "reference", "problem_solution"},
		RelationshipConfidence: map[types.RelationType]float64{
			types.RelationLedTo:      0.7,
			types.RelationSolvedBy:   0.8,
			types.RelationDependsOn:  0.6,
			types.RelationFollowsUp:  0.7,
			types.RelationRelatedTo:  0.5,
			types.RelationReferences: 0.8,
		},
	}
}

// DetectionResult represents the result of relationship detection
type DetectionResult struct {
	RelationshipsDetected []types.MemoryRelationship `json:"relationships_detected"`
	ConfidenceScores      map[string]float64         `json:"confidence_scores"`
	DetectionMethods      map[string][]string        `json:"detection_methods"`
	ProcessingTime        time.Duration              `json:"processing_time"`
}

// DetectRelationships analyzes a chunk and detects potential relationships
func (rd *RelationshipDetector) DetectRelationships(ctx context.Context, chunk *types.ConversationChunk, config *DetectionConfig) (*DetectionResult, error) {
	start := time.Now()

	if config == nil {
		config = DefaultDetectionConfig()
	}

	result := &DetectionResult{
		RelationshipsDetected: make([]types.MemoryRelationship, 0),
		ConfidenceScores:      make(map[string]float64),
		DetectionMethods:      make(map[string][]string),
	}

	// Find candidate chunks to compare against
	candidates, err := rd.findCandidateChunks(ctx, chunk, config)
	if err != nil {
		return nil, fmt.Errorf("failed to find candidate chunks: %w", err)
	}

	// Run detection algorithms
	for _, detectorType := range config.EnabledDetectors {
		switch detectorType {
		case "temporal":
			rd.detectTemporalRelationships(chunk, candidates, config, result)
		case "causal":
			rd.detectCausalRelationships(chunk, candidates, config, result)
		case "reference":
			rd.detectReferenceRelationships(chunk, candidates, config, result)
		case "problem_solution":
			rd.detectProblemSolutionRelationships(chunk, candidates, config, result)
		}
	}

	// Filter by minimum confidence
	filtered := make([]types.MemoryRelationship, 0)
	for _, rel := range result.RelationshipsDetected {
		if rel.Confidence >= config.MinConfidence {
			filtered = append(filtered, rel)
		}
	}
	result.RelationshipsDetected = filtered

	result.ProcessingTime = time.Since(start)
	return result, nil
}

// AutoDetectAndStore detects relationships for a chunk and stores them
func (rd *RelationshipDetector) AutoDetectAndStore(ctx context.Context, chunk *types.ConversationChunk, config *DetectionConfig) error {
	result, err := rd.DetectRelationships(ctx, chunk, config)
	if err != nil {
		return err
	}

	// Store detected relationships
	for _, relationship := range result.RelationshipsDetected {
		_, err := rd.storage.StoreRelationship(ctx, 
			relationship.SourceChunkID, 
			relationship.TargetChunkID, 
			relationship.RelationType, 
			relationship.Confidence, 
			types.ConfidenceAuto,
		)
		if err != nil {
			return fmt.Errorf("failed to store relationship: %w", err)
		}
	}

	return nil
}

// findCandidateChunks finds chunks that could potentially have relationships with the given chunk
func (rd *RelationshipDetector) findCandidateChunks(ctx context.Context, chunk *types.ConversationChunk, config *DetectionConfig) ([]*types.ConversationChunk, error) {
	candidates := make([]*types.ConversationChunk, 0)

	// Get chunks from the same session
	if chunk.SessionID != "" {
		sessionCandidates, err := rd.getSessionChunks(ctx, chunk.SessionID, chunk.ID)
		if err == nil {
			candidates = append(candidates, sessionCandidates...)
		}
	}

	// Get recent chunks from the same repository
	if chunk.Metadata.Repository != "" {
		repoCandidates, err := rd.getRecentRepositoryChunks(ctx, chunk.Metadata.Repository, chunk.ID, config.MaxTimeDistance)
		if err == nil {
			candidates = append(candidates, repoCandidates...)
		}
	}

	// Remove duplicates and the chunk itself
	seen := make(map[string]bool)
	seen[chunk.ID] = true
	
	uniqueCandidates := make([]*types.ConversationChunk, 0)
	for _, candidate := range candidates {
		if !seen[candidate.ID] {
			seen[candidate.ID] = true
			uniqueCandidates = append(uniqueCandidates, candidate)
		}
	}

	return uniqueCandidates, nil
}

// getSessionChunks retrieves other chunks from the same session
func (rd *RelationshipDetector) getSessionChunks(ctx context.Context, sessionID, excludeID string) ([]*types.ConversationChunk, error) {
	// This would need to be implemented based on the storage interface
	// For now, return empty slice
	return []*types.ConversationChunk{}, nil
}

// getRecentRepositoryChunks retrieves recent chunks from the same repository
func (rd *RelationshipDetector) getRecentRepositoryChunks(ctx context.Context, repository, excludeID string, maxAge time.Duration) ([]*types.ConversationChunk, error) {
	// This would need to be implemented based on the storage interface
	// For now, return empty slice
	return []*types.ConversationChunk{}, nil
}

// detectTemporalRelationships detects relationships based on time proximity
func (rd *RelationshipDetector) detectTemporalRelationships(chunk *types.ConversationChunk, candidates []*types.ConversationChunk, config *DetectionConfig, result *DetectionResult) {
	for _, candidate := range candidates {
		timeDiff := chunk.Timestamp.Sub(candidate.Timestamp)
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		// If chunks are close in time and in the same session
		if timeDiff <= config.MaxTimeDistance && chunk.SessionID == candidate.SessionID {
			confidence := rd.calculateTemporalConfidence(timeDiff, config.MaxTimeDistance)
			
			var relationType types.RelationType
			if chunk.Timestamp.After(candidate.Timestamp) {
				relationType = types.RelationFollowsUp
			} else {
				relationType = types.RelationPrecedes
			}

			if baseConfidence, exists := config.RelationshipConfidence[relationType]; exists {
				confidence *= baseConfidence
			}

			if confidence >= config.MinConfidence {
				rel := types.MemoryRelationship{
					SourceChunkID:    candidate.ID,
					TargetChunkID:    chunk.ID,
					RelationType:     relationType,
					Confidence:       confidence,
					ConfidenceSource: types.ConfidenceAuto,
					ConfidenceFactors: types.ConfidenceFactors{
						TemporalProximity: &confidence,
					},
					CreatedAt: time.Now().UTC(),
				}

				result.RelationshipsDetected = append(result.RelationshipsDetected, rel)
				result.DetectionMethods[rel.ID] = append(result.DetectionMethods[rel.ID], "temporal")
			}
		}
	}
}

// detectCausalRelationships detects cause-effect relationships
func (rd *RelationshipDetector) detectCausalRelationships(chunk *types.ConversationChunk, candidates []*types.ConversationChunk, config *DetectionConfig, result *DetectionResult) {
	causalPatterns := []struct {
		pattern    *regexp.Regexp
		relation   types.RelationType
		confidence float64
	}{
		{regexp.MustCompile(`(?i)\b(led to|caused|resulted in|triggered)\b`), types.RelationLedTo, 0.8},
		{regexp.MustCompile(`(?i)\b(solved by|fixed by|resolved by)\b`), types.RelationSolvedBy, 0.8},
		{regexp.MustCompile(`(?i)\b(depends on|requires|needs)\b`), types.RelationDependsOn, 0.7},
		{regexp.MustCompile(`(?i)\b(because of|due to|as a result of)\b`), types.RelationLedTo, 0.7},
	}

	for _, candidate := range candidates {
		for _, pattern := range causalPatterns {
			// Check if chunk content references the candidate
			if pattern.pattern.MatchString(chunk.Content) && rd.contentReferences(chunk.Content, candidate) {
				confidence := pattern.confidence
				
				if baseConfidence, exists := config.RelationshipConfidence[pattern.relation]; exists {
					confidence *= baseConfidence
				}

				if confidence >= config.MinConfidence {
					rel := types.MemoryRelationship{
						SourceChunkID:    candidate.ID,
						TargetChunkID:    chunk.ID,
						RelationType:     pattern.relation,
						Confidence:       confidence,
						ConfidenceSource: types.ConfidenceAuto,
						ConfidenceFactors: types.ConfidenceFactors{
							ConsistencyScore: &confidence,
						},
						CreatedAt: time.Now().UTC(),
					}

					result.RelationshipsDetected = append(result.RelationshipsDetected, rel)
					result.DetectionMethods[rel.ID] = append(result.DetectionMethods[rel.ID], "causal")
				}
			}
		}
	}
}

// detectReferenceRelationships detects explicit references between chunks
func (rd *RelationshipDetector) detectReferenceRelationships(chunk *types.ConversationChunk, candidates []*types.ConversationChunk, config *DetectionConfig, result *DetectionResult) {
	referencePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(as mentioned|as discussed|as we saw)\b`),
		regexp.MustCompile(`(?i)\b(earlier|previously|before)\b`),
		regexp.MustCompile(`(?i)\b(similar to|like the|same as)\b`),
		regexp.MustCompile(`(?i)\b(refer to|see|check)\b`),
	}

	for _, candidate := range candidates {
		referenceScore := 0.0
		
		for _, pattern := range referencePatterns {
			if pattern.MatchString(chunk.Content) && rd.contentReferences(chunk.Content, candidate) {
				referenceScore += 0.2
			}
		}

		// Also check for explicit ID references
		if strings.Contains(chunk.Content, candidate.ID) {
			referenceScore += 0.5
		}

		// Check for content similarity
		if rd.calculateContentSimilarity(chunk.Content, candidate.Content) > 0.3 {
			referenceScore += 0.3
		}

		if baseConfidence, exists := config.RelationshipConfidence[types.RelationReferences]; exists {
			referenceScore *= baseConfidence
		}

		if referenceScore >= config.MinConfidence {
			rel := types.MemoryRelationship{
				SourceChunkID:    chunk.ID,
				TargetChunkID:    candidate.ID,
				RelationType:     types.RelationReferences,
				Confidence:       referenceScore,
				ConfidenceSource: types.ConfidenceAuto,
				ConfidenceFactors: types.ConfidenceFactors{
					ContextualRelevance: &referenceScore,
				},
				CreatedAt: time.Now().UTC(),
			}

			result.RelationshipsDetected = append(result.RelationshipsDetected, rel)
			result.DetectionMethods[rel.ID] = append(result.DetectionMethods[rel.ID], "reference")
		}
	}
}

// detectProblemSolutionRelationships detects problem-solution pairs
func (rd *RelationshipDetector) detectProblemSolutionRelationships(chunk *types.ConversationChunk, candidates []*types.ConversationChunk, config *DetectionConfig, result *DetectionResult) {
	if chunk.Type == types.ChunkTypeSolution {
		// Look for related problems
		for _, candidate := range candidates {
			if candidate.Type == types.ChunkTypeProblem && 
			   chunk.SessionID == candidate.SessionID &&
			   chunk.Timestamp.After(candidate.Timestamp) {
				
				// Calculate relevance based on content similarity and time proximity
				contentSim := rd.calculateContentSimilarity(chunk.Content, candidate.Content)
				timeDiff := chunk.Timestamp.Sub(candidate.Timestamp)
				timeScore := rd.calculateTemporalConfidence(timeDiff, config.MaxTimeDistance)
				
				confidence := (contentSim * 0.7) + (timeScore * 0.3)
				
				if baseConfidence, exists := config.RelationshipConfidence[types.RelationSolvedBy]; exists {
					confidence *= baseConfidence
				}

				if confidence >= config.MinConfidence {
					rel := types.MemoryRelationship{
						SourceChunkID:    candidate.ID,
						TargetChunkID:    chunk.ID,
						RelationType:     types.RelationSolvedBy,
						Confidence:       confidence,
						ConfidenceSource: types.ConfidenceAuto,
						ConfidenceFactors: types.ConfidenceFactors{
							SemanticSimilarity:  &contentSim,
							TemporalProximity:   &timeScore,
						},
						CreatedAt: time.Now().UTC(),
					}

					result.RelationshipsDetected = append(result.RelationshipsDetected, rel)
					result.DetectionMethods[rel.ID] = append(result.DetectionMethods[rel.ID], "problem_solution")
				}
			}
		}
	}
}

// Helper methods

// calculateTemporalConfidence calculates confidence based on time proximity
func (rd *RelationshipDetector) calculateTemporalConfidence(timeDiff, maxTime time.Duration) float64 {
	if timeDiff > maxTime {
		return 0.0
	}
	// Linear decay: closer in time = higher confidence
	return 1.0 - (float64(timeDiff) / float64(maxTime))
}

// contentReferences checks if content references another chunk
func (rd *RelationshipDetector) contentReferences(content string, candidate *types.ConversationChunk) bool {
	content = strings.ToLower(content)
	
	// Check for ID reference
	if strings.Contains(content, strings.ToLower(candidate.ID)) {
		return true
	}

	// Check for summary reference
	if candidate.Summary != "" {
		summaryWords := strings.Fields(strings.ToLower(candidate.Summary))
		for _, word := range summaryWords {
			if len(word) > 4 && strings.Contains(content, word) {
				return true
			}
		}
	}

	// Check for content overlap
	return rd.calculateContentSimilarity(content, candidate.Content) > 0.3
}

// calculateContentSimilarity calculates simple similarity between two pieces of content
func (rd *RelationshipDetector) calculateContentSimilarity(content1, content2 string) float64 {
	words1 := rd.extractWords(strings.ToLower(content1))
	words2 := rd.extractWords(strings.ToLower(content2))

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Simple Jaccard similarity
	intersection := 0
	word2Set := make(map[string]bool)
	for _, word := range words2 {
		word2Set[word] = true
	}

	for _, word := range words1 {
		if word2Set[word] {
			intersection++
		}
	}

	union := len(words1) + len(words2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// extractWords extracts meaningful words from content
func (rd *RelationshipDetector) extractWords(content string) []string {
	// Simple word extraction, excluding common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "can": true, "this": true,
		"that": true, "these": true, "those": true, "i": true, "you": true,
		"he": true, "she": true, "it": true, "we": true, "they": true,
	}

	wordRegex := regexp.MustCompile(`\b[a-zA-Z]{3,}\b`)
	matches := wordRegex.FindAllString(content, -1)
	
	var words []string
	for _, word := range matches {
		word = strings.ToLower(word)
		if !stopWords[word] {
			words = append(words, word)
		}
	}
	
	return words
}