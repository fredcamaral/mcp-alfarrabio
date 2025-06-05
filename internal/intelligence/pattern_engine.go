// Package intelligence provides AI-powered pattern recognition, learning engines,
// conflict detection, and knowledge graph capabilities for the MCP Memory Server.
package intelligence

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"mcp-memory/pkg/types"
)

// PatternType represents different types of patterns we can recognize
type PatternType string

const (
	PatternTypeProblemSolution PatternType = "problem_solution"
	PatternTypeWorkflow        PatternType = "workflow"
	PatternTypeDecisionMaking  PatternType = "decision_making"
	PatternTypeErrorResolution PatternType = "error_resolution"
	PatternTypeCodeEvolution   PatternType = "code_evolution"
	PatternTypeArchitectural   PatternType = "architectural"
	PatternTypeConfiguration   PatternType = "configuration"
	PatternTypeDebugging       PatternType = "debugging"
	PatternTypeTesting         PatternType = "testing"
	PatternTypeRefactoring     PatternType = "refactoring"
)

// Pattern represents a recognized conversation pattern
type Pattern struct {
	ID              string           `json:"id"`
	Type            PatternType      `json:"type"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Confidence      float64          `json:"confidence"`
	Frequency       int              `json:"frequency"`
	SuccessRate     float64          `json:"success_rate"`
	Keywords        []string         `json:"keywords"`
	Triggers        []string         `json:"triggers"`
	Outcomes        []string         `json:"outcomes"`
	Steps           []PatternStep    `json:"steps"`
	Context         map[string]any   `json:"context"`
	RelatedPatterns []string         `json:"related_patterns"`
	Examples        []PatternExample `json:"examples"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	LastUsed        time.Time        `json:"last_used"`
}

// PatternStep represents a step in a pattern sequence
type PatternStep struct {
	Order       int            `json:"order"`
	Action      string         `json:"action"`
	Description string         `json:"description"`
	Optional    bool           `json:"optional"`
	Confidence  float64        `json:"confidence"`
	Context     map[string]any `json:"context"`
}

// PatternExample represents an example instance of a pattern
type PatternExample struct {
	ID           string                    `json:"id"`
	ChunkIDs     []string                  `json:"chunk_ids"`
	Conversation []types.ConversationChunk `json:"conversation"`
	Outcome      PatternOutcome            `json:"outcome"`
	Confidence   float64                   `json:"confidence"`
	Timestamp    time.Time                 `json:"timestamp"`
}

// PatternOutcome represents the result of applying a pattern
type PatternOutcome string

const (
	OutcomeSuccess     PatternOutcome = "success"
	OutcomePartial     PatternOutcome = "partial"
	OutcomeFailure     PatternOutcome = "failure"
	OutcomeInterrupted PatternOutcome = "interrupted"
	OutcomeUnknown     PatternOutcome = "unknown"
)

// PatternMatcher defines the interface for pattern matching algorithms
type PatternMatcher interface {
	MatchPattern(chunks []types.ConversationChunk, pattern *Pattern) float64
	ExtractFeatures(chunks []types.ConversationChunk) map[string]any
	IdentifySequence(chunks []types.ConversationChunk) []PatternStep
}

// SequenceRecognizer recognizes sequential patterns in conversations
type SequenceRecognizer interface {
	RecognizeSequence(chunks []types.ConversationChunk) ([]Pattern, error)
	LearnFromSequence(chunks []types.ConversationChunk, outcome PatternOutcome) error
}

// PatternStorage interface for storing and retrieving patterns
type PatternStorage interface {
	StorePattern(ctx context.Context, pattern *Pattern) error
	GetPattern(ctx context.Context, id string) (*Pattern, error)
	ListPatterns(ctx context.Context, patternType *PatternType) ([]Pattern, error)
	UpdatePattern(ctx context.Context, pattern *Pattern) error
	DeletePattern(ctx context.Context, id string) error
	SearchPatterns(ctx context.Context, query string, limit int) ([]Pattern, error)
}

// PatternEngine is the main engine for pattern recognition and learning
type PatternEngine struct {
	storage    PatternStorage
	matcher    PatternMatcher
	recognizer SequenceRecognizer

	// Configuration
	minConfidence   float64
	maxPatterns     int
	learningEnabled bool

	// Built-in pattern definitions
	builtInPatterns []Pattern

	// Pattern matching regexes
	problemRegex  *regexp.Regexp
	solutionRegex *regexp.Regexp
	errorRegex    *regexp.Regexp
	commandRegex  *regexp.Regexp
	codeRegex     *regexp.Regexp
}

// NewPatternEngine creates a new pattern recognition engine
func NewPatternEngine(storage PatternStorage) *PatternEngine {
	engine := &PatternEngine{
		storage:         storage,
		minConfidence:   0.6,
		maxPatterns:     1000,
		learningEnabled: true,
		builtInPatterns: getBuiltInPatterns(),
	}

	// Compile regexes for pattern matching
	engine.problemRegex = regexp.MustCompile(`(?i)(error|issue|problem|bug|fail|broken|not working|doesn't work)`)
	engine.solutionRegex = regexp.MustCompile(`(?i)(fix|solve|resolve|solution|fixed|resolved|working)`)
	engine.errorRegex = regexp.MustCompile(`(?i)(error:|exception:|fatal:|panic:|warning:)`)
	engine.commandRegex = regexp.MustCompile(`(?i)(run|execute|install|build|test|deploy)`)
	engine.codeRegex = regexp.MustCompile("```[\\s\\S]*?```")

	// Initialize built-in pattern matcher and recognizer
	engine.matcher = NewBasicPatternMatcher()
	engine.recognizer = NewSequenceRecognizer(engine)

	return engine
}

// RecognizePatterns analyzes chunks and identifies patterns
func (pe *PatternEngine) RecognizePatterns(ctx context.Context, chunks []types.ConversationChunk) ([]Pattern, error) {
	if len(chunks) == 0 {
		return []Pattern{}, nil
	}

	var recognizedPatterns []Pattern

	// Try sequence recognition first
	sequencePatterns, err := pe.recognizer.RecognizeSequence(chunks)
	if err == nil {
		recognizedPatterns = append(recognizedPatterns, sequencePatterns...)
	}

	// Match against stored patterns
	storedPatterns, err := pe.storage.ListPatterns(ctx, nil)
	if err == nil {
		for i := range storedPatterns {
			pattern := &storedPatterns[i]
			confidence := pe.matcher.MatchPattern(chunks, pattern)
			if confidence >= pe.minConfidence {
				pattern.Confidence = confidence
				recognizedPatterns = append(recognizedPatterns, *pattern)
			}
		}
	}

	// Match against built-in patterns
	for i := range pe.builtInPatterns {
		pattern := &pe.builtInPatterns[i]
		confidence := pe.matcher.MatchPattern(chunks, pattern)
		if confidence >= pe.minConfidence {
			pattern.Confidence = confidence
			recognizedPatterns = append(recognizedPatterns, *pattern)
		}
	}

	// Sort by confidence and limit results
	sort.Slice(recognizedPatterns, func(i, j int) bool {
		return recognizedPatterns[i].Confidence > recognizedPatterns[j].Confidence
	})

	if len(recognizedPatterns) > 10 {
		recognizedPatterns = recognizedPatterns[:10]
	}

	return recognizedPatterns, nil
}

// LearnPattern creates or updates a pattern based on conversation examples
func (pe *PatternEngine) LearnPattern(ctx context.Context, chunks []types.ConversationChunk, outcome PatternOutcome) error {
	if !pe.learningEnabled || len(chunks) < 2 {
		return nil
	}

	// Extract pattern features
	features := pe.matcher.ExtractFeatures(chunks)
	steps := pe.matcher.IdentifySequence(chunks)

	// Create pattern from conversation
	pattern := Pattern{
		ID:          generatePatternID(),
		Type:        pe.inferPatternType(chunks, features),
		Name:        pe.generatePatternName(chunks, features),
		Description: pe.generatePatternDescription(chunks, features),
		Confidence:  pe.calculatePatternConfidence(chunks, features),
		Frequency:   1,
		SuccessRate: pe.calculateSuccessRate(outcome),
		Keywords:    pe.extractKeywords(chunks),
		Triggers:    pe.extractTriggers(chunks),
		Outcomes:    pe.extractOutcomes(chunks),
		Steps:       steps,
		Context:     features,
		Examples: []PatternExample{{
			ID:           fmt.Sprintf("%s_example_1", generatePatternID()),
			ChunkIDs:     extractChunkIDs(chunks),
			Conversation: chunks,
			Outcome:      outcome,
			Confidence:   1.0,
			Timestamp:    time.Now(),
		}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	// Try to find existing similar pattern
	existingPatterns, err := pe.storage.SearchPatterns(ctx, pattern.Name, 5)
	if err == nil && len(existingPatterns) > 0 {
		// Update existing pattern
		existing := existingPatterns[0]
		if pe.matcher.MatchPattern(chunks, &existing) > 0.8 {
			existing.Frequency++
			existing.SuccessRate = (existing.SuccessRate*float64(existing.Frequency-1) + pe.calculateSuccessRate(outcome)) / float64(existing.Frequency)
			existing.Examples = append(existing.Examples, pattern.Examples[0])
			existing.UpdatedAt = time.Now()
			existing.LastUsed = time.Now()

			return pe.storage.UpdatePattern(ctx, &existing)
		}
	}

	// Store new pattern
	return pe.storage.StorePattern(ctx, &pattern)
}

// GetPatternSuggestions returns patterns that might be relevant to current context
func (pe *PatternEngine) GetPatternSuggestions(ctx context.Context, currentChunks []types.ConversationChunk, limit int) ([]Pattern, error) {
	if len(currentChunks) == 0 {
		return []Pattern{}, nil
	}

	// Get all patterns
	allPatterns, err := pe.storage.ListPatterns(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Calculate relevance for each pattern
	type patternScore struct {
		pattern   Pattern
		relevance float64
	}

	var scoredPatterns []patternScore

	for i := range allPatterns {
		pattern := &allPatterns[i]
		relevance := pe.calculateRelevance(currentChunks, pattern)
		if relevance > 0.3 {
			scoredPatterns = append(scoredPatterns, patternScore{
				pattern:   *pattern,
				relevance: relevance,
			})
		}
	}

	// Sort by relevance
	sort.Slice(scoredPatterns, func(i, j int) bool {
		return scoredPatterns[i].relevance > scoredPatterns[j].relevance
	})

	// Return top patterns
	result := make([]Pattern, 0, limit)
	for i := range scoredPatterns {
		if i >= limit {
			break
		}
		scored := &scoredPatterns[i]
		scored.pattern.Confidence = scored.relevance
		result = append(result, scored.pattern)
	}

	return result, nil
}

// Helper methods

func (pe *PatternEngine) inferPatternType(chunks []types.ConversationChunk, _ map[string]any) PatternType {
	text := extractText(chunks)

	if pe.problemRegex.MatchString(text) && pe.solutionRegex.MatchString(text) {
		return PatternTypeProblemSolution
	}
	if pe.errorRegex.MatchString(text) {
		return PatternTypeErrorResolution
	}
	if pe.codeRegex.MatchString(text) {
		return PatternTypeCodeEvolution
	}
	if pe.commandRegex.MatchString(text) {
		return PatternTypeWorkflow
	}

	return PatternTypeWorkflow
}

func (pe *PatternEngine) generatePatternName(chunks []types.ConversationChunk, _ map[string]any) string {
	keywords := pe.extractKeywords(chunks)
	if len(keywords) > 0 {
		return fmt.Sprintf("%s Pattern", strings.ToUpper(keywords[0][:1])+keywords[0][1:])
	}
	return "Generic Pattern"
}

func (pe *PatternEngine) generatePatternDescription(chunks []types.ConversationChunk, features map[string]any) string {
	patternType := pe.inferPatternType(chunks, features)
	return fmt.Sprintf("Automatically learned %s pattern from conversation", patternType)
}

func (pe *PatternEngine) calculatePatternConfidence(chunks []types.ConversationChunk, features map[string]any) float64 {
	// Base confidence on conversation length and coherence
	baseConfidence := math.Min(float64(len(chunks))/10.0, 1.0)

	// Adjust based on features
	if len(features) > 3 {
		baseConfidence += 0.1
	}

	return math.Min(baseConfidence, 1.0)
}

func (pe *PatternEngine) calculateSuccessRate(outcome PatternOutcome) float64 {
	switch outcome {
	case OutcomeSuccess:
		return 1.0
	case OutcomePartial:
		return 0.6
	case OutcomeFailure:
		return 0.2
	case OutcomeInterrupted:
		return 0.4
	case OutcomeUnknown:
		return 0.5
	default:
		return 0.5
	}
}

func (pe *PatternEngine) extractKeywords(chunks []types.ConversationChunk) []string {
	text := extractText(chunks)
	words := strings.Fields(strings.ToLower(text))

	keywordCount := make(map[string]int)
	for _, word := range words {
		if len(word) > 3 && !isStopWord(word) {
			keywordCount[word]++
		}
	}

	type wordCount struct {
		word  string
		count int
	}

	wordCounts := make([]wordCount, 0, len(keywordCount))
	for word, count := range keywordCount {
		wordCounts = append(wordCounts, wordCount{word, count})
	}

	sort.Slice(wordCounts, func(i, j int) bool {
		return wordCounts[i].count > wordCounts[j].count
	})

	// Pre-allocate for max 10 keywords
	maxKeywords := 10
	if len(wordCounts) < maxKeywords {
		maxKeywords = len(wordCounts)
	}
	keywords := make([]string, 0, maxKeywords)
	for i, wc := range wordCounts {
		if i >= 10 {
			break
		}
		keywords = append(keywords, wc.word)
	}

	return keywords
}

func (pe *PatternEngine) extractTriggers(chunks []types.ConversationChunk) []string {
	var triggers []string

	for i := range chunks {
		chunk := &chunks[i]
		if pe.problemRegex.MatchString(chunk.Content) {
			triggers = append(triggers, "problem_identified")
		}
		if pe.errorRegex.MatchString(chunk.Content) {
			triggers = append(triggers, "error_occurred")
		}
		if pe.commandRegex.MatchString(chunk.Content) {
			triggers = append(triggers, "command_execution")
		}
	}

	return unique(triggers)
}

func (pe *PatternEngine) extractOutcomes(chunks []types.ConversationChunk) []string {
	var outcomes []string

	for i := range chunks {
		chunk := &chunks[i]
		if pe.solutionRegex.MatchString(chunk.Content) {
			outcomes = append(outcomes, "solution_found")
		}
		if strings.Contains(strings.ToLower(chunk.Content), "complete") {
			outcomes = append(outcomes, "task_completed")
		}
	}

	return unique(outcomes)
}

func (pe *PatternEngine) calculateRelevance(currentChunks []types.ConversationChunk, pattern *Pattern) float64 {
	// Calculate keyword overlap
	currentKeywords := pe.extractKeywords(currentChunks)
	keywordOverlap := calculateOverlap(currentKeywords, pattern.Keywords)

	// Calculate trigger relevance
	currentTriggers := pe.extractTriggers(currentChunks)
	triggerOverlap := calculateOverlap(currentTriggers, pattern.Triggers)

	// Weight the relevance
	relevance := (keywordOverlap*0.6 + triggerOverlap*0.4) * pattern.SuccessRate

	return math.Min(relevance, 1.0)
}

// Utility functions

func extractText(chunks []types.ConversationChunk) string {
	texts := make([]string, 0, len(chunks))
	for i := range chunks {
		chunk := &chunks[i]
		texts = append(texts, chunk.Content)
	}
	return strings.Join(texts, " ")
}

func extractChunkIDs(chunks []types.ConversationChunk) []string {
	ids := make([]string, 0, len(chunks))
	for i := range chunks {
		chunk := &chunks[i]
		ids = append(ids, chunk.ID)
	}
	return ids
}

func generatePatternID() string {
	return fmt.Sprintf("pattern_%d", time.Now().UnixNano())
}

func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "had": true,
		"her": true, "was": true, "one": true, "our": true, "out": true,
		"day": true, "get": true, "has": true, "him": true, "how": true,
		"man": true, "new": true, "now": true, "old": true, "see": true,
		"two": true, "way": true, "who": true, "boy": true, "did": true,
		"its": true, "let": true, "put": true, "say": true, "she": true,
		"too": true, "use": true,
	}
	return stopWords[word]
}

func unique(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

func calculateOverlap(slice1, slice2 []string) float64 {
	if len(slice1) == 0 || len(slice2) == 0 {
		return 0.0
	}

	set1 := make(map[string]bool)
	for _, item := range slice1 {
		set1[item] = true
	}

	overlap := 0
	for _, item := range slice2 {
		if set1[item] {
			overlap++
		}
	}

	return float64(overlap) / math.Max(float64(len(slice1)), float64(len(slice2)))
}
