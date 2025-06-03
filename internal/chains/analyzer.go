package chains

import (
	"context"
	"fmt"
	"math"
	"mcp-memory/internal/embeddings"
	"mcp-memory/pkg/types"
	"strings"
	"time"
)

// DefaultChainAnalyzer implements ChainAnalyzer
type DefaultChainAnalyzer struct {
	embeddingService embeddings.EmbeddingService
}

// NewDefaultChainAnalyzer creates a new default chain analyzer
func NewDefaultChainAnalyzer(embeddingService embeddings.EmbeddingService) *DefaultChainAnalyzer {
	return &DefaultChainAnalyzer{
		embeddingService: embeddingService,
	}
}

// AnalyzeRelationship analyzes the relationship between two chunks
func (a *DefaultChainAnalyzer) AnalyzeRelationship(_ context.Context, chunk1, chunk2 types.ConversationChunk) (ChainType, float64, error) {
	// Calculate various similarity metrics
	semanticSim := a.calculateSemanticSimilarity(chunk1.Embeddings, chunk2.Embeddings)
	temporalProximity := a.calculateTemporalProximity(chunk1.Timestamp, chunk2.Timestamp)
	// Use tags as concepts for now
	conceptOverlap := a.calculateConceptOverlap(chunk1.Metadata.Tags, chunk2.Metadata.Tags)
	entityOverlap := 0.0 // No entities in base type

	// Determine chain type based on patterns
	chainType := a.determineChainType(chunk1, chunk2, semanticSim, temporalProximity, conceptOverlap)

	// Calculate overall strength
	strength := a.calculateOverallStrength(semanticSim, temporalProximity, conceptOverlap, entityOverlap)

	return chainType, strength, nil
}

// SuggestChainName suggests a name and description for a chain
func (a *DefaultChainAnalyzer) SuggestChainName(_ context.Context, chunks []types.ConversationChunk) (string, string, error) {
	if len(chunks) == 0 {
		return "", "", fmt.Errorf("no chunks provided")
	}

	// Find common concepts from tags
	conceptFreq := make(map[string]int)
	for _, chunk := range chunks {
		for _, tag := range chunk.Metadata.Tags {
			conceptFreq[tag]++
		}
	}

	// Find most common concept
	var topConcept string
	maxFreq := 0
	for concept, freq := range conceptFreq {
		if freq > maxFreq {
			maxFreq = freq
			topConcept = concept
		}
	}

	// Determine chain theme
	theme := a.determineChainTheme(chunks)

	// Generate name
	name := a.generateChainName(topConcept, theme, chunks[0].Metadata.Repository)

	// Generate description
	description := a.generateChainDescription(chunks, theme, conceptFreq)

	return name, description, nil
}

// calculateSemanticSimilarity calculates cosine similarity between embeddings
func (a *DefaultChainAnalyzer) calculateSemanticSimilarity(emb1, emb2 []float64) float64 {
	if len(emb1) != len(emb2) || len(emb1) == 0 {
		return 0.0
	}

	var dotProduct, norm1, norm2 float64
	for i := range emb1 {
		dotProduct += emb1[i] * emb2[i]
		norm1 += emb1[i] * emb1[i]
		norm2 += emb2[i] * emb2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// calculateTemporalProximity calculates how close in time two chunks are
func (a *DefaultChainAnalyzer) calculateTemporalProximity(t1, t2 time.Time) float64 {
	diff := math.Abs(t1.Sub(t2).Hours())

	// Use exponential decay for temporal proximity
	// Chunks within 1 hour have high proximity, decays over days
	return math.Exp(-diff / 24.0) // 24 hours decay constant
}

// calculateConceptOverlap calculates Jaccard similarity for concepts
func (a *DefaultChainAnalyzer) calculateConceptOverlap(concepts1, concepts2 []string) float64 {
	if len(concepts1) == 0 && len(concepts2) == 0 {
		return 0.0
	}

	set1 := make(map[string]bool)
	for _, c := range concepts1 {
		set1[strings.ToLower(c)] = true
	}

	set2 := make(map[string]bool)
	for _, c := range concepts2 {
		set2[strings.ToLower(c)] = true
	}

	// Calculate intersection and union
	intersection := 0
	for c := range set1 {
		if set2[c] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// determineChainType determines the type of relationship
func (a *DefaultChainAnalyzer) determineChainType(chunk1, chunk2 types.ConversationChunk, semanticSim, temporalProx, conceptOverlap float64) ChainType {
	// Check for solution pattern
	if (chunk1.Type == "problem" || chunk1.Type == "bug") && chunk2.Type == "solution" {
		return ChainTypeSolution
	}

	// Check for continuation (high temporal proximity and semantic similarity)
	if temporalProx > 0.8 && semanticSim > 0.7 {
		return ChainTypeContinuation
	}

	// Check for evolution (moderate temporal distance but high concept overlap)
	if temporalProx < 0.3 && conceptOverlap > 0.6 {
		return ChainTypeEvolution
	}

	// Check for conflict (similar concepts but different outcomes)
	if conceptOverlap > 0.5 && chunk1.Metadata.Outcome != "" && chunk2.Metadata.Outcome != "" &&
		chunk1.Metadata.Outcome != chunk2.Metadata.Outcome {
		return ChainTypeConflict
	}

	// Check for support (one supports the other)
	if semanticSim > 0.6 && (chunk1.Type == "reference" || chunk2.Type == "reference") {
		return ChainTypeSupport
	}

	// Default to reference
	return ChainTypeReference
}

// calculateOverallStrength calculates the overall link strength
func (a *DefaultChainAnalyzer) calculateOverallStrength(semanticSim, temporalProx, conceptOverlap, entityOverlap float64) float64 {
	// Weighted average of different factors
	weights := map[string]float64{
		"semantic": 0.4,
		"temporal": 0.2,
		"concept":  0.25,
		"entity":   0.15,
	}

	strength := semanticSim*weights["semantic"] +
		temporalProx*weights["temporal"] +
		conceptOverlap*weights["concept"] +
		entityOverlap*weights["entity"]

	// Ensure strength is between 0 and 1
	return math.Min(math.Max(strength, 0.0), 1.0)
}

// determineChainTheme determines the overall theme of a chain
func (a *DefaultChainAnalyzer) determineChainTheme(chunks []types.ConversationChunk) string {
	// Count types
	typeCount := make(map[string]int)
	for _, chunk := range chunks {
		typeCount[string(chunk.Type)]++
	}

	// Find dominant type
	var dominantType string
	maxCount := 0
	for t, count := range typeCount {
		if count > maxCount {
			maxCount = count
			dominantType = t
		}
	}

	// Map to theme
	themeMap := map[string]string{
		"problem":       "Problem Solving",
		"solution":      "Solution Implementation",
		"decision":      "Decision Making",
		"conversation":  "Discussion",
		"code":          "Code Development",
		"documentation": "Documentation",
		"learning":      "Learning Journey",
	}

	if theme, exists := themeMap[dominantType]; exists {
		return theme
	}

	return "General Development"
}

// generateChainName generates a descriptive name for the chain
func (a *DefaultChainAnalyzer) generateChainName(topConcept, theme, repository string) string {
	if topConcept != "" {
		// Simple title case - capitalize first letter
		titled := topConcept
		if len(topConcept) > 0 {
			titled = strings.ToUpper(topConcept[:1]) + topConcept[1:]
		}
		return fmt.Sprintf("%s: %s", theme, titled)
	}

	if repository != "" && repository != "_global" {
		return fmt.Sprintf("%s in %s", theme, repository)
	}

	return fmt.Sprintf("%s Chain", theme)
}

// generateChainDescription generates a description for the chain
func (a *DefaultChainAnalyzer) generateChainDescription(chunks []types.ConversationChunk, theme string, conceptFreq map[string]int) string {
	// Get time range
	var minTime, maxTime time.Time
	for i, chunk := range chunks {
		if i == 0 || chunk.Timestamp.Before(minTime) {
			minTime = chunk.Timestamp
		}
		if i == 0 || chunk.Timestamp.After(maxTime) {
			maxTime = chunk.Timestamp
		}
	}

	// Get top concepts
	topConcepts := a.getTopConcepts(conceptFreq, 3)

	// Build description
	desc := fmt.Sprintf("%s chain spanning from %s to %s. ",
		theme,
		minTime.Format("Jan 2, 2006"),
		maxTime.Format("Jan 2, 2006"))

	if len(topConcepts) > 0 {
		desc += fmt.Sprintf("Key concepts: %s. ", strings.Join(topConcepts, ", "))
	}

	desc += fmt.Sprintf("Contains %d related memories.", len(chunks))

	return desc
}

// getTopConcepts gets the top N concepts by frequency
func (a *DefaultChainAnalyzer) getTopConcepts(conceptFreq map[string]int, n int) []string {
	// Convert to slice for sorting
	type conceptCount struct {
		concept string
		count   int
	}

	counts := make([]conceptCount, 0, len(conceptFreq))
	for concept, count := range conceptFreq {
		counts = append(counts, conceptCount{concept, count})
	}

	// Sort by count
	for i := 0; i < len(counts)-1; i++ {
		for j := i + 1; j < len(counts); j++ {
			if counts[j].count > counts[i].count {
				counts[i], counts[j] = counts[j], counts[i]
			}
		}
	}

	// Get top N
	result := make([]string, 0, n)
	for i := 0; i < n && i < len(counts); i++ {
		result = append(result, counts[i].concept)
	}

	return result
}
