package intelligence

import (
	"context"
	"fmt"
	"mcp-memory/pkg/types"
	"sort"
	"strings"
	"time"
)

// SearchExplainer provides detailed explanations for search results
type SearchExplainer struct {
	storage StorageInterface
}

// NewSearchExplainer creates a new search explainer
func NewSearchExplainer(storage StorageInterface) *SearchExplainer {
	return &SearchExplainer{
		storage: storage,
	}
}

// ExplainedSearchResult represents a search result with detailed explanation
type ExplainedSearchResult struct {
	Chunk        types.ConversationChunk `json:"chunk"`
	Score        float64                 `json:"score"`
	Relevance    RelevanceExplanation    `json:"relevance"`
	Context      ContextExplanation      `json:"context"`
	CitationID   string                  `json:"citation_id,omitempty"`
	CitationText string                  `json:"citation_text,omitempty"`
}

// RelevanceExplanation explains why a result is relevant
type RelevanceExplanation struct {
	OverallScore        float64  `json:"overall_score"`
	SemanticSimilarity  float64  `json:"semantic_similarity"`
	KeywordMatches      []string `json:"keyword_matches"`
	RecencyBoost        float64  `json:"recency_boost"`
	UsageFrequencyBoost float64  `json:"usage_frequency_boost"`
	RelationshipBonus   float64  `json:"relationship_bonus"`
	ConfidenceScore     float64  `json:"confidence_score"`
	QualityScore        float64  `json:"quality_score"`
	MatchedConcepts     []string `json:"matched_concepts"`
	Explanation         string   `json:"explanation"`
}

// ContextExplanation provides context about how this result relates to others
type ContextExplanation struct {
	RelatedChunks     []string            `json:"related_chunks"`     // IDs of related memories
	KnowledgePath     []string            `json:"knowledge_path"`     // How we got to this result
	SessionContext    []string            `json:"session_context"`    // Other chunks from same session
	RepositoryContext []string            `json:"repository_context"` // Other chunks from same repo
	TemporalContext   []TemporalContext   `json:"temporal_context"`   // Nearby in time
	ConceptualContext []ConceptualContext `json:"conceptual_context"` // Similar concepts
}

// TemporalContext represents temporal relationship context
type TemporalContext struct {
	ChunkID   string    `json:"chunk_id"`
	Timestamp time.Time `json:"timestamp"`
	Relation  string    `json:"relation"` // "before", "after", "concurrent"
	Distance  string    `json:"distance"` // "hours", "days", "weeks"
}

// ConceptualContext represents conceptual relationship context
type ConceptualContext struct {
	ChunkID    string   `json:"chunk_id"`
	Concepts   []string `json:"concepts"`
	Similarity float64  `json:"similarity"`
}

// ExplainedSearchConfig configures the explanation depth
type ExplainedSearchConfig struct {
	ExplainDepth         string  `json:"explain_depth"` // "basic", "detailed", "debug"
	IncludeRelationships bool    `json:"include_relationships"`
	IncludeCitations     bool    `json:"include_citations"`
	IncludeContext       bool    `json:"include_context"`
	MinExplanationScore  float64 `json:"min_explanation_score"`
	MaxContextItems      int     `json:"max_context_items"`
}

// DefaultExplainedSearchConfig returns default configuration
func DefaultExplainedSearchConfig() *ExplainedSearchConfig {
	return &ExplainedSearchConfig{
		ExplainDepth:         "detailed",
		IncludeRelationships: true,
		IncludeCitations:     true,
		IncludeContext:       true,
		MinExplanationScore:  0.3,
		MaxContextItems:      5,
	}
}

// ExplainedSearchResults contains explained search results
type ExplainedSearchResults struct {
	Results     []ExplainedSearchResult `json:"results"`
	Query       string                  `json:"query"`
	TotalFound  int                     `json:"total_found"`
	QueryTime   time.Duration           `json:"query_time"`
	Explanation SearchQueryExplanation  `json:"explanation"`
	Citations   map[string]string       `json:"citations,omitempty"` // citation_id -> full text
}

// SearchQueryExplanation explains the search process
type SearchQueryExplanation struct {
	QueryTerms       []string      `json:"query_terms"`
	ConceptsDetected []string      `json:"concepts_detected"`
	FiltersApplied   []string      `json:"filters_applied"`
	SearchStrategy   string        `json:"search_strategy"`
	RankingFactors   []string      `json:"ranking_factors"`
	ProcessingTime   time.Duration `json:"processing_time"`
}

// ExplainedSearch performs search with detailed explanations
func (se *SearchExplainer) ExplainedSearch(ctx context.Context, query *types.MemoryQuery, embeddings []float64, config *ExplainedSearchConfig) (*ExplainedSearchResults, error) {
	start := time.Now()

	if config == nil {
		config = DefaultExplainedSearchConfig()
	}

	// Perform the base search
	searchResults, err := se.storage.Search(ctx, query, embeddings)
	if err != nil {
		return nil, err
	}

	// Create explained results
	explainedResults := make([]ExplainedSearchResult, 0, len(searchResults.Results))
	citations := make(map[string]string)
	citationCounter := 1

	for i := range searchResults.Results {
		result := &searchResults.Results[i]
		explained := se.explainResult(ctx, result, query, config)

		// Add citation if requested
		if config.IncludeCitations {
			citationID := se.generateCitationID(citationCounter)
			explained.CitationID = citationID
			explained.CitationText = se.generateCitationText(result.Chunk)
			citations[citationID] = explained.CitationText
			citationCounter++
		}

		explainedResults = append(explainedResults, explained)
	}

	// Sort by explained relevance score
	sort.Slice(explainedResults, func(i, j int) bool {
		return explainedResults[i].Relevance.OverallScore > explainedResults[j].Relevance.OverallScore
	})

	// Build query explanation
	queryExplanation := se.explainQuery(*query, embeddings, start)

	return &ExplainedSearchResults{
		Results:     explainedResults,
		Query:       query.Query,
		TotalFound:  len(explainedResults),
		QueryTime:   time.Since(start),
		Explanation: queryExplanation,
		Citations:   citations,
	}, nil
}

// explainResult creates a detailed explanation for a single result
func (se *SearchExplainer) explainResult(ctx context.Context, result *types.SearchResult, query *types.MemoryQuery, config *ExplainedSearchConfig) ExplainedSearchResult {
	chunk := result.Chunk

	// Calculate relevance explanation
	relevance := se.calculateRelevanceExplanation(chunk, query, result.Score)

	// Calculate context explanation
	var contextExplanation ContextExplanation
	if config.IncludeContext {
		contextExplanation = se.calculateContextExplanation(ctx, chunk, query, config)
	}

	return ExplainedSearchResult{
		Chunk:     chunk,
		Score:     result.Score,
		Relevance: relevance,
		Context:   contextExplanation,
	}
}

// calculateRelevanceExplanation explains why a result is relevant
//
//nolint:gocritic // hugeParam: large struct parameters are needed for interface consistency
func (se *SearchExplainer) calculateRelevanceExplanation(chunk types.ConversationChunk, query *types.MemoryQuery, baseScore float64) RelevanceExplanation {
	explanation := RelevanceExplanation{
		OverallScore: baseScore,
	}

	// Extract query terms
	queryTerms := se.extractQueryTerms(query.Query)

	// Calculate semantic similarity (simplified - would use embeddings in real implementation)
	explanation.SemanticSimilarity = se.calculateSemanticSimilarity(chunk.Content, query.Query)

	// Find keyword matches
	explanation.KeywordMatches = se.findKeywordMatches(chunk, queryTerms)

	// Calculate recency boost
	explanation.RecencyBoost = se.calculateRecencyBoost(chunk.Timestamp)

	// Calculate usage frequency boost (simplified)
	explanation.UsageFrequencyBoost = se.calculateUsageBoost(chunk)

	// Calculate confidence score
	if chunk.Metadata.Confidence != nil {
		explanation.ConfidenceScore = chunk.Metadata.Confidence.Score
	} else {
		explanation.ConfidenceScore = 0.5 // Default
	}

	// Calculate quality score
	if chunk.Metadata.Quality != nil {
		explanation.QualityScore = chunk.Metadata.Quality.OverallQuality
	} else {
		explanation.QualityScore = 0.5 // Default
	}

	// Detect matched concepts
	explanation.MatchedConcepts = se.detectMatchedConcepts(chunk, queryTerms)

	// Generate human-readable explanation
	explanation.Explanation = se.generateRelevanceExplanation(&explanation, chunk, query)

	// Recalculate overall score with explanation factors
	explanation.OverallScore = se.calculateWeightedScore(&explanation)

	return explanation
}

// calculateContextExplanation builds context information
//
//nolint:gocritic // hugeParam: large struct parameters are needed for interface consistency
func (se *SearchExplainer) calculateContextExplanation(ctx context.Context, chunk types.ConversationChunk, query *types.MemoryQuery, config *ExplainedSearchConfig) ContextExplanation {
	contextExpl := ContextExplanation{
		RelatedChunks:     make([]string, 0),
		KnowledgePath:     make([]string, 0),
		SessionContext:    make([]string, 0),
		RepositoryContext: make([]string, 0),
		TemporalContext:   make([]TemporalContext, 0),
		ConceptualContext: make([]ConceptualContext, 0),
	}

	// Get related chunks through relationships
	if config.IncludeRelationships {
		relQuery := types.NewRelationshipQuery(chunk.ID)
		relQuery.Limit = config.MaxContextItems
		if relationships, err := se.storage.GetRelationships(ctx, relQuery); err == nil {
			for i := range relationships {
				// Add the other chunk in the relationship
				if relationships[i].Relationship.SourceChunkID == chunk.ID {
					contextExpl.RelatedChunks = append(contextExpl.RelatedChunks, relationships[i].Relationship.TargetChunkID)
				} else {
					contextExpl.RelatedChunks = append(contextExpl.RelatedChunks, relationships[i].Relationship.SourceChunkID)
				}
			}
		}
	}

	// Use query to enhance session context detection
	if chunk.SessionID != "" {
		sessionDesc := chunk.SessionID
		// If query mentions specific terms, add them to session context
		queryTerms := se.extractQueryTerms(query.Query)
		for _, term := range queryTerms {
			if strings.Contains(strings.ToLower(chunk.Content), term) {
				sessionDesc += fmt.Sprintf(" (matches: %s)", term)
				break
			}
		}
		contextExpl.SessionContext = append(contextExpl.SessionContext, sessionDesc)
	}

	// Enhanced repository context using query filters
	if chunk.Metadata.Repository != "" {
		repoDesc := chunk.Metadata.Repository
		if query.Repository != nil && *query.Repository == chunk.Metadata.Repository {
			repoDesc += " (query-filtered)"
		}
		contextExpl.RepositoryContext = append(contextExpl.RepositoryContext, repoDesc)
	}

	// Add conceptual context based on query
	queryConcepts := se.detectQueryConcepts(query.Query)
	for i := range queryConcepts {
		contextExpl.ConceptualContext = append(contextExpl.ConceptualContext, ConceptualContext{
			ChunkID:    chunk.ID,
			Concepts:   []string{queryConcepts[i]},
			Similarity: 0.8, // Estimated based on query detection
		})
	}

	// Add temporal context (simplified)
	contextExpl.TemporalContext = se.findTemporalContext(chunk, config.MaxContextItems)

	return contextExpl
}

// Helper methods for explanation calculation

func (se *SearchExplainer) extractQueryTerms(query string) []string {
	// Simple tokenization - in production would use proper NLP
	words := strings.Fields(strings.ToLower(query))
	terms := make([]string, 0)

	for _, word := range words {
		// Remove punctuation and filter stop words
		word = strings.Trim(word, ".,!?;:")
		if len(word) > 2 && !se.isStopWord(word) {
			terms = append(terms, word)
		}
	}

	return terms
}

func (se *SearchExplainer) calculateSemanticSimilarity(content, query string) float64 {
	// Simplified semantic similarity - would use embeddings in real implementation
	contentWords := strings.Fields(strings.ToLower(content))
	queryWords := strings.Fields(strings.ToLower(query))

	matches := 0
	for _, qWord := range queryWords {
		for _, cWord := range contentWords {
			if qWord == cWord {
				matches++
				break
			}
		}
	}

	if len(queryWords) == 0 {
		return 0.0
	}

	return float64(matches) / float64(len(queryWords))
}

//nolint:gocritic // hugeParam: large struct parameter needed for processing
func (se *SearchExplainer) findKeywordMatches(chunk types.ConversationChunk, queryTerms []string) []string {
	matches := make([]string, 0)
	contentLower := strings.ToLower(chunk.Content)
	summaryLower := strings.ToLower(chunk.Summary)

	for _, term := range queryTerms {
		if strings.Contains(contentLower, term) || strings.Contains(summaryLower, term) {
			matches = append(matches, term)
		}
	}

	return matches
}

func (se *SearchExplainer) calculateRecencyBoost(timestamp time.Time) float64 {
	daysSince := time.Since(timestamp).Hours() / 24

	// Boost for recent memories
	switch {
	case daysSince < 1:
		return 0.3 // Strong boost for very recent
	case daysSince < 7:
		return 0.2 // Good boost for recent
	case daysSince < 30:
		return 0.1 // Small boost for somewhat recent
	}

	return 0.0 // No boost for old memories
}

//nolint:gocritic // hugeParam: large struct parameter needed for processing
func (se *SearchExplainer) calculateUsageBoost(chunk types.ConversationChunk) float64 {
	// Simplified usage boost based on outcome and type
	boost := 0.0

	// Boost successful outcomes
	if chunk.Metadata.Outcome == types.OutcomeSuccess {
		boost += 0.2
	}

	// Boost solutions and decisions (likely to be referenced)
	if chunk.Type == types.ChunkTypeSolution || chunk.Type == types.ChunkTypeArchitectureDecision {
		boost += 0.1
	}

	return boost
}

//nolint:gocritic // hugeParam: large struct parameter needed for processing
func (se *SearchExplainer) detectMatchedConcepts(chunk types.ConversationChunk, queryTerms []string) []string {
	concepts := make([]string, 0)

	// Extract concepts from tags
	for _, tag := range chunk.Metadata.Tags {
		for _, term := range queryTerms {
			if strings.Contains(strings.ToLower(tag), term) {
				concepts = append(concepts, tag)
			}
		}
	}

	// Extract concepts from content (simplified)
	content := strings.ToLower(chunk.Content)
	if strings.Contains(content, "error") || strings.Contains(content, "bug") {
		concepts = append(concepts, "debugging")
	}
	if strings.Contains(content, "performance") || strings.Contains(content, "slow") {
		concepts = append(concepts, "performance")
	}
	if strings.Contains(content, "security") || strings.Contains(content, "vulnerability") {
		concepts = append(concepts, "security")
	}

	return concepts
}

//nolint:gocritic // hugeParam: large struct parameters needed for interface consistency
func (se *SearchExplainer) generateRelevanceExplanation(rel *RelevanceExplanation, chunk types.ConversationChunk, query *types.MemoryQuery) string {
	explanation := "This result is relevant because: "

	factors := make([]string, 0)

	if len(rel.KeywordMatches) > 0 {
		factors = append(factors, "it contains matching keywords: "+strings.Join(rel.KeywordMatches, ", "))
	}

	if rel.SemanticSimilarity > 0.3 {
		factors = append(factors, "it has high semantic similarity to your query")
	}

	if rel.RecencyBoost > 0 {
		factors = append(factors, "it's a recent memory")
	}

	if rel.ConfidenceScore > 0.7 {
		factors = append(factors, "it has high confidence")
	}

	if chunk.Metadata.Outcome == types.OutcomeSuccess {
		factors = append(factors, "it represents a successful outcome")
	}

	// Use query parameters to enhance explanation
	if query.Repository != nil && chunk.Metadata.Repository == *query.Repository {
		factors = append(factors, fmt.Sprintf("it's from the requested repository: %s", *query.Repository))
	}

	if len(query.Types) > 0 {
		for _, queryType := range query.Types {
			if chunk.Type == queryType {
				factors = append(factors, fmt.Sprintf("it matches the requested type: %s", queryType))
				break
			}
		}
	}

	// Check if query mentions specific problem/solution terms
	queryLower := strings.ToLower(query.Query)
	if (strings.Contains(queryLower, "error") || strings.Contains(queryLower, "problem")) &&
		chunk.Metadata.Outcome == types.OutcomeSuccess {
		factors = append(factors, "it provides a solution to a problem similar to your query")
	}

	if len(factors) == 0 {
		explanation += "it has general similarity to your query."
	} else {
		explanation += strings.Join(factors, "; ") + "."
	}

	return explanation
}

func (se *SearchExplainer) calculateWeightedScore(rel *RelevanceExplanation) float64 {
	// Weighted combination of factors
	weights := map[string]float64{
		"semantic":   0.40,
		"keywords":   0.20,
		"recency":    0.15,
		"usage":      0.10,
		"confidence": 0.10,
		"quality":    0.05,
	}

	score := 0.0
	score += weights["semantic"] * rel.SemanticSimilarity
	score += weights["keywords"] * (float64(len(rel.KeywordMatches)) / 5.0) // Normalize to max 5 keywords
	score += weights["recency"] * rel.RecencyBoost
	score += weights["usage"] * rel.UsageFrequencyBoost
	score += weights["confidence"] * rel.ConfidenceScore
	score += weights["quality"] * rel.QualityScore

	return score
}

//nolint:gocritic // hugeParam: large struct parameter needed for processing
func (se *SearchExplainer) findTemporalContext(chunk types.ConversationChunk, maxItems int) []TemporalContext {
	// Simplified temporal context - would query database in real implementation
	return []TemporalContext{}
}

//nolint:gocritic // hugeParam: large struct parameter needed for interface consistency
func (se *SearchExplainer) explainQuery(query types.MemoryQuery, embeddings []float64, start time.Time) SearchQueryExplanation {
	// Use embeddings to enhance search strategy explanation
	searchStrategy := "semantic_vector_search"
	if len(embeddings) > 0 {
		searchStrategy = fmt.Sprintf("semantic_vector_search (dim=%d)", len(embeddings))
	}

	// Calculate embedding quality if available
	rankingFactors := []string{"semantic_similarity", "keyword_matches", "recency", "confidence", "quality"}
	if len(embeddings) > 0 {
		// Check if embeddings have good distribution (not all zeros or too uniform)
		nonZeroCount := 0
		for _, val := range embeddings {
			if val != 0 {
				nonZeroCount++
			}
		}
		embeddingQuality := float64(nonZeroCount) / float64(len(embeddings))
		if embeddingQuality > 0.1 {
			rankingFactors = append(rankingFactors, "vector_quality")
		}
	}

	return SearchQueryExplanation{
		QueryTerms:       se.extractQueryTerms(query.Query),
		ConceptsDetected: se.detectQueryConcepts(query.Query),
		FiltersApplied:   se.getAppliedFilters(query),
		SearchStrategy:   searchStrategy,
		RankingFactors:   rankingFactors,
		ProcessingTime:   time.Since(start),
	}
}

func (se *SearchExplainer) detectQueryConcepts(query string) []string {
	concepts := make([]string, 0)
	queryLower := strings.ToLower(query)

	conceptKeywords := map[string][]string{
		"debugging":    {"error", "bug", "issue", "problem", "failed"},
		"performance":  {"slow", "performance", "optimization", "speed"},
		"security":     {"security", "vulnerability", "attack", "breach"},
		"architecture": {"design", "architecture", "pattern", "structure"},
		"deployment":   {"deploy", "deployment", "release", "production"},
	}

	for concept, keywords := range conceptKeywords {
		for _, keyword := range keywords {
			if strings.Contains(queryLower, keyword) {
				concepts = append(concepts, concept)
				break
			}
		}
	}

	return concepts
}

//nolint:gocritic // hugeParam: large struct parameter needed for processing
func (se *SearchExplainer) getAppliedFilters(query types.MemoryQuery) []string {
	filters := make([]string, 0)

	if query.Repository != nil && *query.Repository != "" {
		filters = append(filters, "repository:"+*query.Repository)
	}

	if len(query.Types) > 0 {
		typeStrs := make([]string, len(query.Types))
		for i, t := range query.Types {
			typeStrs[i] = string(t)
		}
		filters = append(filters, "types:"+strings.Join(typeStrs, ","))
	}

	if query.Recency != types.RecencyAllTime {
		filters = append(filters, "recency:"+string(query.Recency))
	}

	return filters
}

func (se *SearchExplainer) generateCitationID(counter int) string {
	return "[" + string(rune('A'+counter-1)) + "]"
}

//nolint:gocritic // hugeParam: large struct parameter needed for processing
func (se *SearchExplainer) generateCitationText(chunk types.ConversationChunk) string {
	citation := ""

	// Add type and timestamp
	citation += string(chunk.Type) + " from " + chunk.Timestamp.Format("2006-01-02")

	// Add repository if available
	if chunk.Metadata.Repository != "" {
		citation += " (" + chunk.Metadata.Repository + ")"
	}

	// Add session if available
	if chunk.SessionID != "" {
		citation += " [Session: " + chunk.SessionID + "]"
	}

	return citation
}

func (se *SearchExplainer) isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "is": true,
		"are": true, "was": true, "were": true, "be": true, "been": true,
		"have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
	}

	return stopWords[word]
}
