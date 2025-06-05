package intelligence

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// ConflictType represents different types of conflicts that can be detected
type ConflictType string

const (
	ConflictTypeArchitectural ConflictType = "architectural"
	ConflictTypeTechnical     ConflictType = "technical"
	ConflictTypeTemporal      ConflictType = "temporal"
	ConflictTypeMethodology   ConflictType = "methodology"
	ConflictTypeOutcome       ConflictType = "outcome"
	ConflictTypePattern       ConflictType = "pattern"
	ConflictTypeDecision      ConflictType = "decision"
)

// ConflictResolutionType represents different types of conflict resolution strategies
type ConflictResolutionType string

const (
	ResolutionAcceptLatest  ConflictResolutionType = "accept_latest"
	ResolutionAcceptHighest ConflictResolutionType = "accept_highest_confidence"
	ResolutionMerge         ConflictResolutionType = "merge"
	ResolutionManualReview  ConflictResolutionType = "manual_review"
	ResolutionContextual    ConflictResolutionType = "contextual"
	ResolutionEvolutionary  ConflictResolutionType = "evolutionary"
	ResolutionDomain        ConflictResolutionType = "domain_specific"
)

// ConflictSeverity represents the severity level of a conflict
type ConflictSeverity string

const (
	SeverityCritical ConflictSeverity = "critical"
	SeverityHigh     ConflictSeverity = "high"
	SeverityMedium   ConflictSeverity = "medium"
	SeverityLow      ConflictSeverity = "low"
	SeverityInfo     ConflictSeverity = "info"
)

// Conflict represents a detected conflict between chunks
type Conflict struct {
	ID          string           `json:"id"`
	Type        ConflictType     `json:"type"`
	Severity    ConflictSeverity `json:"severity"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Confidence  float64          `json:"confidence"`

	// Conflicting chunks
	PrimaryChunk  types.ConversationChunk   `json:"primary_chunk"`
	ConflictChunk types.ConversationChunk   `json:"conflict_chunk"`
	RelatedChunks []types.ConversationChunk `json:"related_chunks,omitempty"`

	// Conflict details
	ConflictPoints []ConflictPoint `json:"conflict_points"`
	Evidence       []string        `json:"evidence"`
	Context        map[string]any  `json:"context"`

	// Temporal information
	TimeDifference time.Duration `json:"time_difference"`
	DetectedAt     time.Time     `json:"detected_at"`

	// Resolution information
	Resolved       bool                    `json:"resolved"`
	ResolutionType *ConflictResolutionType `json:"resolution_type,omitempty"`
	ResolutionNote *string                 `json:"resolution_note,omitempty"`
}

// ConflictPoint represents a specific point of conflict
type ConflictPoint struct {
	Aspect      string  `json:"aspect"`
	Primary     string  `json:"primary"`
	Conflicting string  `json:"conflicting"`
	Confidence  float64 `json:"confidence"`
}

// ConflictDetectionResult contains the results of conflict detection
type ConflictDetectionResult struct {
	Repository     string     `json:"repository"`
	Timeframe      string     `json:"timeframe"`
	TotalChunks    int        `json:"total_chunks"`
	ConflictsFound int        `json:"conflicts_found"`
	Conflicts      []Conflict `json:"conflicts"`
	AnalysisTime   time.Time  `json:"analysis_time"`
	ProcessingTime string     `json:"processing_time"`
}

// ConflictDetector is the main conflict detection engine
type ConflictDetector struct {
	// Configuration
	minConfidence          float64
	maxTimeDifferencedays  int
	enableTemporalAnalysis bool
	enablePatternAnalysis  bool

	// Keyword sets for different conflict domains
	architecturalKeywords []string
	technicalKeywords     []string
	methodologyKeywords   []string
	outcomeKeywords       []string
}

// NewConflictDetector creates a new conflict detection engine
func NewConflictDetector() *ConflictDetector {
	return &ConflictDetector{
		minConfidence:          0.6,
		maxTimeDifferencedays:  180, // 6 months
		enableTemporalAnalysis: true,
		enablePatternAnalysis:  true,

		architecturalKeywords: []string{
			"architecture", "design", "pattern", "structure", "component",
			"microservice", "monolith", "database", "storage", "cache",
			"api", "interface", "protocol", "framework", "library",
		},
		technicalKeywords: []string{
			"implementation", "algorithm", "performance", "optimization",
			"security", "authentication", "authorization", "encryption",
			"testing", "deployment", "monitoring", "logging", "metrics",
		},
		methodologyKeywords: []string{
			"approach", "methodology", "process", "workflow", "strategy",
			"best practice", "convention", "standard", "guideline",
		},
		outcomeKeywords: []string{
			"success", "failure", "works", "doesn't work", "broken", "fixed",
			"resolved", "solved", "failed", "error", "issue", "bug",
		},
	}
}

// DetectConflicts analyzes chunks and detects conflicts
func (cd *ConflictDetector) DetectConflicts(ctx context.Context, chunks []types.ConversationChunk) (*ConflictDetectionResult, error) {
	startTime := time.Now()

	result := &ConflictDetectionResult{
		TotalChunks:  len(chunks),
		Conflicts:    []Conflict{},
		AnalysisTime: startTime,
	}

	if len(chunks) < 2 {
		result.ProcessingTime = time.Since(startTime).String()
		return result, nil
	}

	// Detect different types of conflicts
	conflicts := []Conflict{}

	// 1. Architectural conflicts
	archConflicts := cd.detectArchitecturalConflicts(chunks)
	conflicts = append(conflicts, archConflicts...)

	// 2. Technical conflicts
	techConflicts := cd.detectTechnicalConflicts(chunks)
	conflicts = append(conflicts, techConflicts...)

	// 3. Temporal conflicts
	if cd.enableTemporalAnalysis {
		temporalConflicts := cd.detectTemporalConflicts(chunks)
		conflicts = append(conflicts, temporalConflicts...)
	}

	// 4. Outcome conflicts
	outcomeConflicts := cd.detectOutcomeConflicts(chunks)
	conflicts = append(conflicts, outcomeConflicts...)

	// 5. Decision conflicts
	decisionConflicts := cd.detectDecisionConflicts(chunks)
	conflicts = append(conflicts, decisionConflicts...)

	// 6. Methodology conflicts
	methodConflicts := cd.detectMethodologyConflicts(chunks)
	conflicts = append(conflicts, methodConflicts...)

	// Filter conflicts by confidence threshold
	filteredConflicts := []Conflict{}
	for i := range conflicts {
		if conflicts[i].Confidence >= cd.minConfidence {
			filteredConflicts = append(filteredConflicts, conflicts[i])
		}
	}

	// Sort by severity and confidence
	sort.Slice(filteredConflicts, func(i, j int) bool {
		if filteredConflicts[i].Severity != filteredConflicts[j].Severity {
			return cd.getSeverityWeight(filteredConflicts[i].Severity) > cd.getSeverityWeight(filteredConflicts[j].Severity)
		}
		return filteredConflicts[i].Confidence > filteredConflicts[j].Confidence
	})

	result.Conflicts = filteredConflicts
	result.ConflictsFound = len(filteredConflicts)
	result.ProcessingTime = time.Since(startTime).String()

	return result, nil
}

// detectArchitecturalConflicts finds conflicts in architectural decisions
func (cd *ConflictDetector) detectArchitecturalConflicts(chunks []types.ConversationChunk) []Conflict {
	var conflicts []Conflict

	// Group architectural decision chunks
	archChunks := []types.ConversationChunk{}
	for i := range chunks {
		if chunks[i].Type == types.ChunkTypeArchitectureDecision ||
			cd.containsKeywords(chunks[i].Content, cd.architecturalKeywords) {
			archChunks = append(archChunks, chunks[i])
		}
	}

	// Compare architectural chunks for conflicts
	for i := range archChunks {
		for j := range archChunks {
			if i >= j || archChunks[i].SessionID == archChunks[j].SessionID {
				continue
			}

			if conflict := cd.analyzeArchitecturalConflict(archChunks[i], archChunks[j]); conflict != nil {
				conflicts = append(conflicts, *conflict)
			}
		}
	}

	return conflicts
}

// detectTechnicalConflicts finds conflicts in technical implementations
func (cd *ConflictDetector) detectTechnicalConflicts(chunks []types.ConversationChunk) []Conflict {
	var conflicts []Conflict

	// Group technical chunks
	techChunks := []types.ConversationChunk{}
	for i := range chunks {
		if chunks[i].Type == types.ChunkTypeCodeChange || chunks[i].Type == types.ChunkTypeSolution ||
			cd.containsKeywords(chunks[i].Content, cd.technicalKeywords) {
			techChunks = append(techChunks, chunks[i])
		}
	}

	// Compare technical approaches
	for i := range techChunks {
		for j := range techChunks {
			if i >= j || techChunks[i].SessionID == techChunks[j].SessionID {
				continue
			}

			if conflict := cd.analyzeTechnicalConflict(techChunks[i], techChunks[j]); conflict != nil {
				conflicts = append(conflicts, *conflict)
			}
		}
	}

	return conflicts
}

// detectTemporalConflicts finds conflicts based on temporal patterns
func (cd *ConflictDetector) detectTemporalConflicts(chunks []types.ConversationChunk) []Conflict {
	var conflicts []Conflict

	// Sort chunks by timestamp
	sortedChunks := make([]types.ConversationChunk, len(chunks))
	copy(sortedChunks, chunks)
	sort.Slice(sortedChunks, func(i, j int) bool {
		return sortedChunks[i].Timestamp.Before(sortedChunks[j].Timestamp)
	})

	// Look for contradictory patterns over time
	for i := 0; i < len(sortedChunks)-1; i++ {
		for j := i + 1; j < len(sortedChunks); j++ {
			chunk1 := sortedChunks[i]
			chunk2 := sortedChunks[j]

			timeDiff := chunk2.Timestamp.Sub(chunk1.Timestamp)
			if timeDiff.Hours() > float64(cd.maxTimeDifferencedays*24) {
				break // Too far apart
			}

			if conflict := cd.analyzeTemporalConflict(chunk1, chunk2, timeDiff); conflict != nil {
				conflicts = append(conflicts, *conflict)
			}
		}
	}

	return conflicts
}

// detectOutcomeConflicts finds conflicts in reported outcomes
func (cd *ConflictDetector) detectOutcomeConflicts(chunks []types.ConversationChunk) []Conflict {
	var conflicts []Conflict

	// Group chunks with similar content but different outcomes
	chunkGroups := cd.groupSimilarChunks(chunks)

	for _, group := range chunkGroups {
		if len(group) < 2 {
			continue
		}

		// Check for outcome conflicts within the group
		for i := range group {
			for j := range group {
				if i >= j {
					continue
				}

				if conflict := cd.analyzeOutcomeConflict(group[i], group[j]); conflict != nil {
					conflicts = append(conflicts, *conflict)
				}
			}
		}
	}

	return conflicts
}

// detectDecisionConflicts finds conflicts in decision-making
func (cd *ConflictDetector) detectDecisionConflicts(chunks []types.ConversationChunk) []Conflict {
	var conflicts []Conflict

	// Find decision-related chunks
	decisionChunks := []types.ConversationChunk{}
	for i := range chunks {
		if chunks[i].Type == types.ChunkTypeArchitectureDecision ||
			cd.containsDecisionLanguage(chunks[i].Content) {
			decisionChunks = append(decisionChunks, chunks[i])
		}
	}

	// Analyze decision conflicts
	for i := range decisionChunks {
		for j := range decisionChunks {
			if i >= j || decisionChunks[i].SessionID == decisionChunks[j].SessionID {
				continue
			}

			if conflict := cd.analyzeDecisionConflict(decisionChunks[i], decisionChunks[j]); conflict != nil {
				conflicts = append(conflicts, *conflict)
			}
		}
	}

	return conflicts
}

// detectMethodologyConflicts finds conflicts in methodological approaches
func (cd *ConflictDetector) detectMethodologyConflicts(chunks []types.ConversationChunk) []Conflict {
	var conflicts []Conflict

	// Find methodology-related chunks
	methodChunks := []types.ConversationChunk{}
	for i := range chunks {
		if cd.containsKeywords(chunks[i].Content, cd.methodologyKeywords) {
			methodChunks = append(methodChunks, chunks[i])
		}
	}

	// Analyze methodology conflicts
	for i := range methodChunks {
		for j := range methodChunks {
			if i >= j || methodChunks[i].SessionID == methodChunks[j].SessionID {
				continue
			}

			if conflict := cd.analyzeMethodologyConflict(methodChunks[i], methodChunks[j]); conflict != nil {
				conflicts = append(conflicts, *conflict)
			}
		}
	}

	return conflicts
}

// Analysis methods for specific conflict types

// analyzeGenericConflict provides common conflict analysis logic
//
//nolint:gocritic // hugeParam: large struct parameters required for interface consistency
func (cd *ConflictDetector) analyzeGenericConflict(chunk1, chunk2 types.ConversationChunk, conflictType ConflictType,
	findConflictPoints func(string, string) []ConflictPoint,
	determineSeverity func([]ConflictPoint) ConflictSeverity,
	generateDescription func([]ConflictPoint) string,
	title string) *Conflict {
	similarity := cd.calculateContentSimilarity(chunk1.Content, chunk2.Content)
	if similarity < 0.3 {
		return nil
	}

	conflictPoints := findConflictPoints(chunk1.Content, chunk2.Content)
	if len(conflictPoints) == 0 {
		return nil
	}

	confidence := cd.calculateConflictConfidence(similarity, conflictPoints)
	severity := determineSeverity(conflictPoints)

	return &Conflict{
		ID:             cd.generateConflictID(),
		Type:           conflictType,
		Severity:       severity,
		Title:          title,
		Description:    generateDescription(conflictPoints),
		Confidence:     confidence,
		PrimaryChunk:   chunk1,
		ConflictChunk:  chunk2,
		ConflictPoints: conflictPoints,
		Evidence:       cd.extractEvidence(chunk1.Content, chunk2.Content),
		Context:        map[string]any{"similarity": similarity},
		TimeDifference: chunk2.Timestamp.Sub(chunk1.Timestamp),
		DetectedAt:     time.Now(),
	}
}

//nolint:gocritic // hugeParam: large struct parameters required for interface consistency
func (cd *ConflictDetector) analyzeArchitecturalConflict(chunk1, chunk2 types.ConversationChunk) *Conflict {
	return cd.analyzeGenericConflict(chunk1, chunk2, ConflictTypeArchitectural,
		cd.findArchitecturalConflictPoints,
		cd.determineArchitecturalSeverity,
		cd.generateArchitecturalDescription,
		"Architectural Decision Conflict")
}

//nolint:gocritic // hugeParam: large struct parameters required for interface consistency
func (cd *ConflictDetector) analyzeTechnicalConflict(chunk1, chunk2 types.ConversationChunk) *Conflict {
	return cd.analyzeGenericConflict(chunk1, chunk2, ConflictTypeTechnical,
		cd.findTechnicalConflictPoints,
		cd.determineTechnicalSeverity,
		cd.generateTechnicalDescription,
		"Technical Implementation Conflict")
}

//nolint:gocritic // hugeParam: large struct parameters required for interface consistency
func (cd *ConflictDetector) analyzeTemporalConflict(chunk1, chunk2 types.ConversationChunk, timeDiff time.Duration) *Conflict {
	similarity := cd.calculateContentSimilarity(chunk1.Content, chunk2.Content)
	if similarity < 0.4 {
		return nil
	}

	// Check if there's a temporal progression that seems contradictory
	if !cd.hasTemporalContradiction(&chunk1, &chunk2) {
		return nil
	}

	conflictPoints := cd.findTemporalConflictPoints(chunk1.Content, chunk2.Content)
	confidence := cd.calculateConflictConfidence(similarity, conflictPoints)
	severity := cd.determineTemporalSeverity(timeDiff, conflictPoints)

	return &Conflict{
		ID:             cd.generateConflictID(),
		Type:           ConflictTypeTemporal,
		Severity:       severity,
		Title:          "Temporal Contradiction",
		Description:    cd.generateTemporalDescription(timeDiff, conflictPoints),
		Confidence:     confidence,
		PrimaryChunk:   chunk1,
		ConflictChunk:  chunk2,
		ConflictPoints: conflictPoints,
		Evidence:       cd.extractEvidence(chunk1.Content, chunk2.Content),
		Context:        map[string]any{"time_difference_days": timeDiff.Hours() / 24},
		TimeDifference: timeDiff,
		DetectedAt:     time.Now(),
	}
}

//nolint:gocritic // hugeParam: large struct parameters required for interface consistency
func (cd *ConflictDetector) analyzeOutcomeConflict(chunk1, chunk2 types.ConversationChunk) *Conflict {
	if chunk1.Metadata.Outcome == chunk2.Metadata.Outcome {
		return nil // Same outcome, no conflict
	}

	similarity := cd.calculateContentSimilarity(chunk1.Content, chunk2.Content)
	if similarity < 0.5 {
		return nil // Not similar enough
	}

	conflictPoints := cd.findOutcomeConflictPoints(&chunk1, &chunk2)
	confidence := cd.calculateConflictConfidence(similarity, conflictPoints)
	severity := cd.determineOutcomeSeverity(chunk1.Metadata.Outcome, chunk2.Metadata.Outcome)

	return &Conflict{
		ID:             cd.generateConflictID(),
		Type:           ConflictTypeOutcome,
		Severity:       severity,
		Title:          "Outcome Conflict",
		Description:    cd.generateOutcomeDescription(chunk1.Metadata.Outcome, chunk2.Metadata.Outcome),
		Confidence:     confidence,
		PrimaryChunk:   chunk1,
		ConflictChunk:  chunk2,
		ConflictPoints: conflictPoints,
		Evidence:       cd.extractEvidence(chunk1.Content, chunk2.Content),
		Context:        map[string]any{"similarity": similarity},
		TimeDifference: chunk2.Timestamp.Sub(chunk1.Timestamp),
		DetectedAt:     time.Now(),
	}
}

//nolint:gocritic // hugeParam: large struct parameters required for interface consistency
func (cd *ConflictDetector) analyzeDecisionConflict(chunk1, chunk2 types.ConversationChunk) *Conflict {
	return cd.analyzeGenericConflict(chunk1, chunk2, ConflictTypeDecision,
		cd.findDecisionConflictPoints,
		cd.determineDecisionSeverity,
		cd.generateDecisionDescription,
		"Decision Conflict")
}

//nolint:gocritic // hugeParam: large struct parameters required for interface consistency
func (cd *ConflictDetector) analyzeMethodologyConflict(chunk1, chunk2 types.ConversationChunk) *Conflict {
	return cd.analyzeGenericConflict(chunk1, chunk2, ConflictTypeMethodology,
		cd.findMethodologyConflictPoints,
		cd.determineMethodologySeverity,
		cd.generateMethodologyDescription,
		"Methodology Conflict")
}

// Helper methods

func (cd *ConflictDetector) containsKeywords(content string, keywords []string) bool {
	contentLower := strings.ToLower(content)
	for _, keyword := range keywords {
		if strings.Contains(contentLower, keyword) {
			return true
		}
	}
	return false
}

func (cd *ConflictDetector) containsDecisionLanguage(content string) bool {
	decisionWords := []string{
		"decided", "choose", "chose", "decision", "option", "alternative",
		"recommend", "suggest", "prefer", "adopt", "implement", "use",
		"go with", "select", "pick", "opt for",
	}
	return cd.containsKeywords(content, decisionWords)
}

func (cd *ConflictDetector) calculateContentSimilarity(content1, content2 string) float64 {
	words1 := strings.Fields(strings.ToLower(content1))
	words2 := strings.Fields(strings.ToLower(content2))

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Simple Jaccard similarity
	set1 := make(map[string]bool)
	for _, word := range words1 {
		set1[word] = true
	}

	intersection := 0
	set2 := make(map[string]bool)
	for _, word := range words2 {
		set2[word] = true
		if set1[word] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

func (cd *ConflictDetector) groupSimilarChunks(chunks []types.ConversationChunk) [][]types.ConversationChunk {
	// Pre-allocate with estimated capacity (worst case: each chunk is its own group)
	groups := make([][]types.ConversationChunk, 0, len(chunks))
	used := make(map[int]bool)

	for i := range chunks {
		if used[i] {
			continue
		}

		group := []types.ConversationChunk{chunks[i]}
		used[i] = true

		for j := range chunks {
			if i == j || used[j] {
				continue
			}

			if cd.calculateContentSimilarity(chunks[i].Content, chunks[j].Content) > 0.4 {
				group = append(group, chunks[j])
				used[j] = true
			}
		}

		groups = append(groups, group)
	}

	return groups
}

func (cd *ConflictDetector) hasTemporalContradiction(chunk1, chunk2 *types.ConversationChunk) bool {
	// Check if later chunk contradicts earlier chunk
	if chunk1.Metadata.Outcome == types.OutcomeSuccess && chunk2.Metadata.Outcome == types.OutcomeFailed {
		return true
	}

	// Check for contradictory statements
	contradictions := [][]string{
		{"works", "doesn't work"},
		{"successful", "failed"},
		{"resolved", "still broken"},
		{"fixed", "still having issues"},
		{"stable", "unstable"},
		{"fast", "slow"},
		{"secure", "vulnerable"},
	}

	content1Lower := strings.ToLower(chunk1.Content)
	content2Lower := strings.ToLower(chunk2.Content)

	for _, pair := range contradictions {
		if strings.Contains(content1Lower, pair[0]) && strings.Contains(content2Lower, pair[1]) {
			return true
		}
	}

	return false
}

func (cd *ConflictDetector) calculateConflictConfidence(similarity float64, conflictPoints []ConflictPoint) float64 {
	if len(conflictPoints) == 0 {
		return 0.0
	}

	avgPointConfidence := 0.0
	for _, point := range conflictPoints {
		avgPointConfidence += point.Confidence
	}
	avgPointConfidence /= float64(len(conflictPoints))

	// Combine similarity and conflict point confidence
	return (similarity*0.3 + avgPointConfidence*0.7)
}

func (cd *ConflictDetector) getSeverityWeight(severity ConflictSeverity) int {
	switch severity {
	case SeverityCritical:
		return 5
	case SeverityHigh:
		return 4
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

func (cd *ConflictDetector) generateConflictID() string {
	return fmt.Sprintf("conflict_%d", time.Now().UnixNano())
}

// Placeholder methods for specific conflict point detection
// These would be implemented with more sophisticated NLP and domain-specific logic

func (cd *ConflictDetector) findArchitecturalConflictPoints(content1, content2 string) []ConflictPoint {
	// Implementation would analyze architectural aspects
	return []ConflictPoint{}
}

func (cd *ConflictDetector) findTechnicalConflictPoints(content1, content2 string) []ConflictPoint {
	// Implementation would analyze technical aspects
	return []ConflictPoint{}
}

func (cd *ConflictDetector) findTemporalConflictPoints(content1, content2 string) []ConflictPoint {
	// Implementation would analyze temporal aspects
	return []ConflictPoint{}
}

func (cd *ConflictDetector) findOutcomeConflictPoints(chunk1, chunk2 *types.ConversationChunk) []ConflictPoint {
	// Implementation would analyze outcome differences
	return []ConflictPoint{}
}

func (cd *ConflictDetector) findDecisionConflictPoints(content1, content2 string) []ConflictPoint {
	// Implementation would analyze decision differences
	return []ConflictPoint{}
}

func (cd *ConflictDetector) findMethodologyConflictPoints(content1, content2 string) []ConflictPoint {
	// Implementation would analyze methodology differences
	return []ConflictPoint{}
}

// Severity determination methods

func (cd *ConflictDetector) determineArchitecturalSeverity(conflictPoints []ConflictPoint) ConflictSeverity {
	if len(conflictPoints) > 3 {
		return SeverityCritical
	}
	if len(conflictPoints) > 1 {
		return SeverityHigh
	}
	return SeverityMedium
}

func (cd *ConflictDetector) determineTechnicalSeverity(conflictPoints []ConflictPoint) ConflictSeverity {
	if len(conflictPoints) > 2 {
		return SeverityHigh
	}
	return SeverityMedium
}

func (cd *ConflictDetector) determineTemporalSeverity(timeDiff time.Duration, _ []ConflictPoint) ConflictSeverity {
	days := timeDiff.Hours() / 24
	if days < 7 {
		return SeverityHigh // Recent contradiction
	}
	if days < 30 {
		return SeverityMedium
	}
	return SeverityLow
}

func (cd *ConflictDetector) determineOutcomeSeverity(outcome1, outcome2 types.Outcome) ConflictSeverity {
	if (outcome1 == types.OutcomeSuccess && outcome2 == types.OutcomeFailed) ||
		(outcome1 == types.OutcomeFailed && outcome2 == types.OutcomeSuccess) {
		return SeverityHigh
	}
	return SeverityMedium
}

func (cd *ConflictDetector) determineDecisionSeverity(conflictPoints []ConflictPoint) ConflictSeverity {
	if len(conflictPoints) > 2 {
		return SeverityHigh
	}
	return SeverityMedium
}

func (cd *ConflictDetector) determineMethodologySeverity(conflictPoints []ConflictPoint) ConflictSeverity {
	if len(conflictPoints) > 1 {
		return SeverityMedium
	}
	return SeverityLow
}

// Description generation methods

func (cd *ConflictDetector) generateArchitecturalDescription(conflictPoints []ConflictPoint) string {
	return fmt.Sprintf("Detected %d architectural conflicts", len(conflictPoints))
}

func (cd *ConflictDetector) generateTechnicalDescription(conflictPoints []ConflictPoint) string {
	return fmt.Sprintf("Detected %d technical implementation conflicts", len(conflictPoints))
}

func (cd *ConflictDetector) generateTemporalDescription(timeDiff time.Duration, _ []ConflictPoint) string {
	days := int(timeDiff.Hours() / 24)
	return fmt.Sprintf("Temporal contradiction detected %d days apart", days)
}

func (cd *ConflictDetector) generateOutcomeDescription(outcome1, outcome2 types.Outcome) string {
	return fmt.Sprintf("Conflicting outcomes: %s vs %s", outcome1, outcome2)
}

func (cd *ConflictDetector) generateDecisionDescription(conflictPoints []ConflictPoint) string {
	return fmt.Sprintf("Detected %d decision conflicts", len(conflictPoints))
}

func (cd *ConflictDetector) generateMethodologyDescription(conflictPoints []ConflictPoint) string {
	return fmt.Sprintf("Detected %d methodology conflicts", len(conflictPoints))
}

func (cd *ConflictDetector) extractEvidence(content1, content2 string) []string {
	// Simple evidence extraction - could be enhanced with NLP
	evidence := []string{}

	// Extract conflicting statements
	sentences1 := strings.Split(content1, ".")
	sentences2 := strings.Split(content2, ".")

	for _, s1 := range sentences1 {
		for _, s2 := range sentences2 {
			if cd.areSentencesConflicting(s1, s2) {
				evidence = append(evidence, fmt.Sprintf("'%s' vs '%s'", strings.TrimSpace(s1), strings.TrimSpace(s2)))
			}
		}
	}

	return evidence
}

func (cd *ConflictDetector) areSentencesConflicting(s1, s2 string) bool {
	// Simple conflicting sentence detection
	contradictions := [][]string{
		{"should", "shouldn't"},
		{"use", "avoid"},
		{"recommended", "not recommended"},
		{"works", "doesn't work"},
		{"stable", "unstable"},
		{"fast", "slow"},
		{"secure", "insecure"},
	}

	s1Lower := strings.ToLower(s1)
	s2Lower := strings.ToLower(s2)

	for _, pair := range contradictions {
		if strings.Contains(s1Lower, pair[0]) && strings.Contains(s2Lower, pair[1]) {
			return true
		}
		if strings.Contains(s1Lower, pair[1]) && strings.Contains(s2Lower, pair[0]) {
			return true
		}
	}

	return false
}
