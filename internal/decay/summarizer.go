package decay

import (
	"context"
	"fmt"
	"math"
	"mcp-memory/pkg/types"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NarrativeFlow represents the flow of a conversation with phases and transitions
type NarrativeFlow struct {
	Phases      []ConversationPhase `json:"phases"`
	KeyEvents   []KeyEvent          `json:"key_events"`
	Transitions []FlowTransition    `json:"transitions"`
}

// ConversationPhase represents a phase in the conversation flow
type ConversationPhase struct {
	Type      types.ConversationFlow    `json:"type"`
	StartTime time.Time                 `json:"start_time"`
	EndTime   time.Time                 `json:"end_time"`
	Chunks    []types.ConversationChunk `json:"chunks"`
	Summary   string                    `json:"summary"`
}

// KeyEvent represents a significant event in the conversation
type KeyEvent struct {
	Type        EventType               `json:"type"`
	Timestamp   time.Time               `json:"timestamp"`
	Description string                  `json:"description"`
	Chunk       types.ConversationChunk `json:"chunk"`
}

// EventType represents types of key events
type EventType string

const (
	EventTypeProblemFound     EventType = "problem_found"
	EventTypeSolutionFound    EventType = "solution_found"
	EventTypeDecisionMade     EventType = "decision_made"
	EventTypeBreakthroughMade EventType = "breakthrough_made"
	EventTypeErrorResolved    EventType = "error_resolved"
)

// FlowTransition represents a transition between conversation phases
type FlowTransition struct {
	From      types.ConversationFlow `json:"from"`
	To        types.ConversationFlow `json:"to"`
	Timestamp time.Time              `json:"timestamp"`
	Trigger   string                 `json:"trigger"`
}

// CriticalInformation represents the most important information to preserve
type CriticalInformation struct {
	Solutions     []types.ConversationChunk `json:"solutions"`
	Decisions     []types.ConversationChunk `json:"decisions"`
	Learnings     []types.ConversationChunk `json:"learnings"`
	Errors        []types.ConversationChunk `json:"errors"`
	KeyOutcomes   []string                  `json:"key_outcomes"`
	Technologies  []string                  `json:"technologies"`
	Relationships map[string][]string       `json:"relationships"`
}

// DefaultSummarizer implements the Summarizer interface
type DefaultSummarizer struct {
	// In a real implementation, this would use an LLM
	// For now, we'll use rule-based summarization
}

// NewDefaultSummarizer creates a new default summarizer
func NewDefaultSummarizer() *DefaultSummarizer {
	return &DefaultSummarizer{}
}

// Summarize creates a summary of multiple chunks
func (s *DefaultSummarizer) Summarize(ctx context.Context, chunks []types.ConversationChunk) (string, error) {
	if len(chunks) == 0 {
		return "", fmt.Errorf("no chunks to summarize")
	}

	// Sort chunks by timestamp
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Timestamp.Before(chunks[j].Timestamp)
	})

	// Extract key information
	var summaryParts []string

	// Time range
	startTime := chunks[0].Timestamp
	endTime := chunks[len(chunks)-1].Timestamp
	summaryParts = append(summaryParts, fmt.Sprintf("Memory summary from %s to %s",
		startTime.Format("Jan 2, 2006"),
		endTime.Format("Jan 2, 2006")))

	// Count by type
	typeCounts := make(map[types.ChunkType]int)
	for _, chunk := range chunks {
		typeCounts[chunk.Type]++
	}

	// Add type summary
	typeInfo := make([]string, 0, len(typeCounts))
	for chunkType, count := range typeCounts {
		typeInfo = append(typeInfo, fmt.Sprintf("%d %s", count, chunkType))
	}
	summaryParts = append(summaryParts, fmt.Sprintf("Contains: %s", strings.Join(typeInfo, ", ")))

	// Extract key topics
	topics := s.extractKeyTopics(chunks)
	if len(topics) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Key topics: %s", strings.Join(topics, ", ")))
	}

	// Extract outcomes
	outcomes := s.extractOutcomes(chunks)
	if len(outcomes) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Outcomes: %s", strings.Join(outcomes, "; ")))
	}

	// Extract tools used
	tools := s.extractTools(chunks)
	if len(tools) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Tools used: %s", strings.Join(tools, ", ")))
	}

	return strings.Join(summaryParts, ". "), nil
}

// SummarizeChain creates a summary chunk from a chain of related chunks
func (s *DefaultSummarizer) SummarizeChain(ctx context.Context, chunks []types.ConversationChunk) (types.ConversationChunk, error) {
	if len(chunks) == 0 {
		return types.ConversationChunk{}, fmt.Errorf("no chunks to summarize")
	}

	// Create summary content
	summaryContent, err := s.Summarize(ctx, chunks)
	if err != nil {
		return types.ConversationChunk{}, err
	}

	// Determine summary metadata
	metadata := s.createSummaryMetadata(chunks)

	// Create summary chunk
	summaryChunk := types.ConversationChunk{
		ID:            uuid.New().String(),
		SessionID:     chunks[0].SessionID, // Use first chunk's session
		Timestamp:     time.Now(),
		Type:          types.ChunkTypeSessionSummary,
		Content:       summaryContent,
		Summary:       fmt.Sprintf("Summary of %d memories", len(chunks)),
		Metadata:      metadata,
		RelatedChunks: extractChunkIDs(chunks),
	}

	return summaryChunk, nil
}

// extractKeyTopics extracts key topics from chunks
func (s *DefaultSummarizer) extractKeyTopics(chunks []types.ConversationChunk) []string {
	// In a real implementation, this would use NLP
	// For now, we'll extract from summaries and content

	topicFreq := make(map[string]int)

	for _, chunk := range chunks {
		// Extract from summary
		words := strings.Fields(strings.ToLower(chunk.Summary))
		for _, word := range words {
			if len(word) > 5 && !isCommonWord(word) {
				topicFreq[word]++
			}
		}

		// Extract from tags
		for _, tag := range chunk.Metadata.Tags {
			topicFreq[strings.ToLower(tag)] += 2 // Tags are more important
		}
	}

	// Get top topics
	return getTopItems(topicFreq, 5)
}

// extractOutcomes extracts outcomes from chunks
func (s *DefaultSummarizer) extractOutcomes(chunks []types.ConversationChunk) []string {
	outcomes := make(map[string]bool)

	for _, chunk := range chunks {
		if chunk.Metadata.Outcome != "" {
			outcomes[string(chunk.Metadata.Outcome)] = true
		}

		// Look for solution chunks
		if chunk.Type == types.ChunkTypeSolution {
			outcomes["Solution implemented"] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(outcomes))
	for outcome := range outcomes {
		result = append(result, outcome)
	}

	return result
}

// extractTools extracts tools used from chunks
func (s *DefaultSummarizer) extractTools(chunks []types.ConversationChunk) []string {
	tools := make(map[string]bool)

	for _, chunk := range chunks {
		for _, tool := range chunk.Metadata.ToolsUsed {
			tools[tool] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(tools))
	for tool := range tools {
		result = append(result, tool)
	}

	sort.Strings(result)
	return result
}

// createSummaryMetadata creates metadata for the summary chunk
func (s *DefaultSummarizer) createSummaryMetadata(chunks []types.ConversationChunk) types.ChunkMetadata {
	metadata := types.ChunkMetadata{
		Tags:       []string{"summary", "decay"},
		Difficulty: types.DifficultyModerate,
		Outcome:    types.OutcomeSuccess,
	}

	// Aggregate metadata from original chunks
	repositories := make(map[string]bool)
	allTags := make(map[string]bool)
	allTools := make(map[string]bool)
	totalTime := 0

	for _, chunk := range chunks {
		if chunk.Metadata.Repository != "" {
			repositories[chunk.Metadata.Repository] = true
		}

		for _, tag := range chunk.Metadata.Tags {
			allTags[tag] = true
		}

		for _, tool := range chunk.Metadata.ToolsUsed {
			allTools[tool] = true
		}

		if chunk.Metadata.TimeSpent != nil {
			totalTime += *chunk.Metadata.TimeSpent
		}
	}

	// Set aggregated values
	if len(repositories) == 1 {
		for repo := range repositories {
			metadata.Repository = repo
			break
		}
	}

	for tag := range allTags {
		metadata.Tags = append(metadata.Tags, tag)
	}

	for tool := range allTools {
		metadata.ToolsUsed = append(metadata.ToolsUsed, tool)
	}

	if totalTime > 0 {
		metadata.TimeSpent = &totalTime
	}

	// Add summarization metadata
	metadata.Tags = append(metadata.Tags, fmt.Sprintf("summarized_%d_chunks", len(chunks)))

	return metadata
}

// Helper functions

func extractChunkIDs(chunks []types.ConversationChunk) []string {
	ids := make([]string, len(chunks))
	for i, chunk := range chunks {
		ids[i] = chunk.ID
	}
	return ids
}

func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true,
		"from": true, "this": true, "that": true, "have": true,
		"been": true, "will": true, "would": true, "could": true,
		"should": true, "about": true, "after": true, "before": true,
	}
	return commonWords[word]
}

func getTopItems(freq map[string]int, limit int) []string {
	// Convert to slice for sorting
	type item struct {
		word  string
		count int
	}

	items := make([]item, 0, len(freq))
	for word, count := range freq {
		items = append(items, item{word, count})
	}

	// Sort by frequency
	sort.Slice(items, func(i, j int) bool {
		return items[i].count > items[j].count
	})

	// Get top items
	result := make([]string, 0, limit)
	for i := 0; i < limit && i < len(items); i++ {
		result = append(result, items[i].word)
	}

	return result
}

// Remove duplicate - EmbeddingGenerator is already defined in memory_decay.go

// embeddedChunk represents a chunk with its embedding
type embeddedChunk struct {
	chunk     types.ConversationChunk
	embedding []float32
}

// LLMSummarizer uses an LLM for more intelligent summarization
type LLMSummarizer struct {
	// This would integrate with an LLM API
	// For demonstration, we'll extend DefaultSummarizer
	*DefaultSummarizer
	embeddingGen EmbeddingGenerator
}

// NewLLMSummarizer creates a new LLM-based summarizer
func NewLLMSummarizer(embeddingGen EmbeddingGenerator) *LLMSummarizer {
	return &LLMSummarizer{
		DefaultSummarizer: NewDefaultSummarizer(),
		embeddingGen:      embeddingGen,
	}
}

// Summarize uses LLM to create an intelligent summary
func (l *LLMSummarizer) Summarize(ctx context.Context, chunks []types.ConversationChunk) (string, error) {
	if len(chunks) == 0 {
		return "", fmt.Errorf("no chunks to summarize")
	}

	// Sort chunks chronologically for coherent narrative
	sort.Slice(chunks, func(i, j int) bool {
		return chunks[i].Timestamp.Before(chunks[j].Timestamp)
	})

	// Phase 1: Analyze semantic similarity and group related content
	semanticGroups := l.groupBySemanticSimilarity(ctx, chunks)

	// Phase 2: Extract narrative flow and key patterns
	narrative := l.extractNarrativeFlow(chunks, semanticGroups)

	// Phase 3: Identify critical information to preserve
	criticalInfo := l.extractCriticalInformation(chunks)

	// Phase 4: Generate intelligent summary using narrative structure
	summary := l.generateIntelligentSummary(narrative, criticalInfo)

	return summary, nil
}

// groupBySemanticSimilarity groups chunks by semantic similarity using embeddings
func (l *LLMSummarizer) groupBySemanticSimilarity(ctx context.Context, chunks []types.ConversationChunk) [][]types.ConversationChunk {
	if l.embeddingGen == nil || len(chunks) < 2 {
		// Fallback to simple grouping
		return [][]types.ConversationChunk{chunks}
	}

	// Generate embeddings for each chunk

	embedded := make([]embeddedChunk, 0, len(chunks))
	for _, chunk := range chunks {
		// Use summary if available, otherwise content
		text := chunk.Summary
		if text == "" {
			text = chunk.Content
		}
		if len(text) > 1000 {
			text = text[:1000] // Truncate for embedding
		}

		if embedding, err := l.embeddingGen.GenerateEmbedding(ctx, text); err == nil {
			embedded = append(embedded, embeddedChunk{chunk: chunk, embedding: embedding})
		}
	}

	// Group by similarity threshold
	return l.clusterBySimilarity(embedded, 0.8) // 0.8 similarity threshold
}

// clusterBySimilarity clusters chunks based on embedding similarity
func (l *LLMSummarizer) clusterBySimilarity(embedded []embeddedChunk, threshold float64) [][]types.ConversationChunk {
	if len(embedded) == 0 {
		return nil
	}

	groups := make([][]types.ConversationChunk, 0)
	used := make(map[int]bool)

	for i, chunk1 := range embedded {
		if used[i] {
			continue
		}

		group := []types.ConversationChunk{chunk1.chunk}
		used[i] = true

		// Find similar chunks
		for j, chunk2 := range embedded {
			if used[j] || i == j {
				continue
			}

			similarity := l.cosineSimilarity(chunk1.embedding, chunk2.embedding)
			if similarity >= threshold {
				group = append(group, chunk2.chunk)
				used[j] = true
			}
		}

		groups = append(groups, group)
	}

	return groups
}

// cosineSimilarity calculates cosine similarity between two vectors
func (l *LLMSummarizer) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// extractNarrativeFlow analyzes the conversation flow and creates a narrative structure
func (l *LLMSummarizer) extractNarrativeFlow(chunks []types.ConversationChunk, groups [][]types.ConversationChunk) NarrativeFlow {
	flow := NarrativeFlow{
		Phases:      make([]ConversationPhase, 0),
		KeyEvents:   make([]KeyEvent, 0),
		Transitions: make([]FlowTransition, 0),
	}

	// Analyze conversation phases
	currentPhase := l.detectPhase(chunks[0])
	phaseStart := chunks[0].Timestamp
	phaseChunks := []types.ConversationChunk{chunks[0]}

	for i := 1; i < len(chunks); i++ {
		chunk := chunks[i]
		detectedPhase := l.detectPhase(chunk)

		if detectedPhase != currentPhase {
			// Phase transition detected
			flow.Phases = append(flow.Phases, ConversationPhase{
				Type:      currentPhase,
				StartTime: phaseStart,
				EndTime:   chunk.Timestamp,
				Chunks:    phaseChunks,
				Summary:   l.summarizePhase(phaseChunks),
			})

			flow.Transitions = append(flow.Transitions, FlowTransition{
				From:      currentPhase,
				To:        detectedPhase,
				Timestamp: chunk.Timestamp,
				Trigger:   l.inferTransitionReason(currentPhase, detectedPhase),
			})

			// Start new phase
			currentPhase = detectedPhase
			phaseStart = chunk.Timestamp
			phaseChunks = []types.ConversationChunk{chunk}
		} else {
			phaseChunks = append(phaseChunks, chunk)
		}

		// Detect key events
		if l.isKeyEvent(chunk) {
			flow.KeyEvents = append(flow.KeyEvents, KeyEvent{
				Type:        l.classifyEvent(chunk),
				Timestamp:   chunk.Timestamp,
				Description: chunk.Summary,
				Chunk:       chunk,
			})
		}
	}

	// Add final phase
	if len(phaseChunks) > 0 {
		flow.Phases = append(flow.Phases, ConversationPhase{
			Type:      currentPhase,
			StartTime: phaseStart,
			EndTime:   chunks[len(chunks)-1].Timestamp,
			Chunks:    phaseChunks,
			Summary:   l.summarizePhase(phaseChunks),
		})
	}

	return flow
}

// extractCriticalInformation identifies the most important information to preserve
func (l *LLMSummarizer) extractCriticalInformation(chunks []types.ConversationChunk) CriticalInformation {
	info := CriticalInformation{
		Solutions:     make([]types.ConversationChunk, 0),
		Decisions:     make([]types.ConversationChunk, 0),
		Learnings:     make([]types.ConversationChunk, 0),
		Errors:        make([]types.ConversationChunk, 0),
		KeyOutcomes:   make([]string, 0),
		Technologies:  make([]string, 0),
		Relationships: make(map[string][]string),
	}

	techMap := make(map[string]bool)
	outcomeMap := make(map[string]bool)

	for _, chunk := range chunks {
		switch chunk.Type {
		case types.ChunkTypeSolution:
			if chunk.Metadata.Outcome == types.OutcomeSuccess {
				info.Solutions = append(info.Solutions, chunk)
			}
		case types.ChunkTypeArchitectureDecision:
			info.Decisions = append(info.Decisions, chunk)
		case types.ChunkTypeAnalysis:
			if l.containsLearning(chunk) {
				info.Learnings = append(info.Learnings, chunk)
			}
		case types.ChunkTypeProblem:
			if chunk.Metadata.Difficulty == types.DifficultyComplex {
				info.Errors = append(info.Errors, chunk)
			}
		}

		// Extract technologies
		for _, tag := range chunk.Metadata.Tags {
			if l.isTechnologyTag(tag) {
				techMap[tag] = true
			}
		}

		// Extract outcomes
		if chunk.Metadata.Outcome != "" {
			outcomeMap[string(chunk.Metadata.Outcome)] = true
		}

		// Extract relationships
		if len(chunk.RelatedChunks) > 0 {
			info.Relationships[chunk.ID] = chunk.RelatedChunks
		}
	}

	// Convert maps to slices
	for tech := range techMap {
		info.Technologies = append(info.Technologies, tech)
	}
	for outcome := range outcomeMap {
		info.KeyOutcomes = append(info.KeyOutcomes, outcome)
	}

	return info
}

// generateIntelligentSummary creates the final intelligent summary
func (l *LLMSummarizer) generateIntelligentSummary(narrative NarrativeFlow, critical CriticalInformation) string {
	var parts []string

	// Start with time context
	if len(narrative.Phases) > 0 {
		startTime := narrative.Phases[0].StartTime
		endTime := narrative.Phases[len(narrative.Phases)-1].EndTime
		duration := endTime.Sub(startTime)
		parts = append(parts, fmt.Sprintf("Conversation spanning %s (%s to %s)",
			formatDuration(duration),
			startTime.Format("Jan 2, 15:04"),
			endTime.Format("Jan 2, 15:04")))
	}

	// Add narrative summary
	if len(narrative.Phases) > 1 {
		phaseDesc := make([]string, 0, len(narrative.Phases))
		for _, phase := range narrative.Phases {
			phaseDuration := phase.EndTime.Sub(phase.StartTime)
			phaseDesc = append(phaseDesc, fmt.Sprintf("%s (%s)",
				string(phase.Type), formatDuration(phaseDuration)))
		}
		parts = append(parts, fmt.Sprintf("Flow: %s", strings.Join(phaseDesc, " → ")))
	}

	// Add critical outcomes
	if len(critical.Solutions) > 0 {
		parts = append(parts, fmt.Sprintf("Implemented %d successful solutions", len(critical.Solutions)))
	}

	if len(critical.Decisions) > 0 {
		decisionDesc := make([]string, 0, len(critical.Decisions))
		for _, decision := range critical.Decisions {
			if len(decisionDesc) < 3 { // Limit to 3 most important
				decisionDesc = append(decisionDesc, decision.Summary)
			}
		}
		parts = append(parts, fmt.Sprintf("Key decisions: %s", strings.Join(decisionDesc, "; ")))
	}

	// Add key learnings
	if len(critical.Learnings) > 0 {
		learningDesc := make([]string, 0, len(critical.Learnings))
		for _, learning := range critical.Learnings {
			if len(learningDesc) < 2 { // Limit to 2 most important
				learningDesc = append(learningDesc, learning.Summary)
			}
		}
		parts = append(parts, fmt.Sprintf("Learnings: %s", strings.Join(learningDesc, "; ")))
	}

	// Add technology context
	if len(critical.Technologies) > 0 {
		parts = append(parts, fmt.Sprintf("Technologies: %s", strings.Join(critical.Technologies, ", ")))
	}

	// Add key events if significant
	if len(narrative.KeyEvents) > 0 {
		eventCount := make(map[string]int)
		for _, event := range narrative.KeyEvents {
			eventCount[string(event.Type)]++
		}
		eventDesc := make([]string, 0, len(eventCount))
		for eventType, count := range eventCount {
			eventDesc = append(eventDesc, fmt.Sprintf("%d %s", count, eventType))
		}
		parts = append(parts, fmt.Sprintf("Events: %s", strings.Join(eventDesc, ", ")))
	}

	return strings.Join(parts, ". ")
}

// extractLearnings extracts key learnings from chunks
func (l *LLMSummarizer) extractLearnings(chunks []types.ConversationChunk) []string {
	learnings := make([]string, 0)

	for _, chunk := range chunks {
		// Look for learning patterns in content
		content := strings.ToLower(chunk.Content)
		if strings.Contains(content, "learned") ||
			strings.Contains(content, "discovered") ||
			strings.Contains(content, "realized") ||
			chunk.Type == types.ChunkTypeAnalysis {
			// Extract the learning (simplified)
			if chunk.Summary != "" {
				learnings = append(learnings, chunk.Summary)
			}
		}
	}

	// Deduplicate and limit
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, learning := range learnings {
		if !seen[learning] && len(unique) < 3 {
			seen[learning] = true
			unique = append(unique, learning)
		}
	}

	return unique
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	case d < 24*time.Hour:
		return fmt.Sprintf("%.1f hours", d.Hours())
	default:
		return fmt.Sprintf("%.1f days", d.Hours()/24)
	}
}

// Helper methods for intelligent summarization

// detectPhase determines the conversation phase for a chunk
func (l *LLMSummarizer) detectPhase(chunk types.ConversationChunk) types.ConversationFlow {
	content := strings.ToLower(chunk.Content + " " + chunk.Summary)

	// Check for problem indicators
	problemWords := []string{"error", "issue", "problem", "bug", "failed", "exception"}
	for _, word := range problemWords {
		if strings.Contains(content, word) {
			return types.FlowProblem
		}
	}

	// Check for investigation indicators
	investigationWords := []string{"investigating", "looking", "checking", "analyzing", "debugging"}
	for _, word := range investigationWords {
		if strings.Contains(content, word) {
			return types.FlowInvestigation
		}
	}

	// Check for solution indicators
	solutionWords := []string{"fix", "solution", "implement", "create", "resolve"}
	for _, word := range solutionWords {
		if strings.Contains(content, word) {
			return types.FlowSolution
		}
	}

	// Check for verification indicators
	verificationWords := []string{"test", "verify", "check", "confirm", "validate"}
	for _, word := range verificationWords {
		if strings.Contains(content, word) {
			return types.FlowVerification
		}
	}

	// Default based on chunk type
	switch chunk.Type {
	case types.ChunkTypeProblem:
		return types.FlowProblem
	case types.ChunkTypeSolution:
		return types.FlowSolution
	case types.ChunkTypeVerification:
		return types.FlowVerification
	default:
		return types.FlowInvestigation
	}
}

// summarizePhase creates a summary for a conversation phase
func (l *LLMSummarizer) summarizePhase(chunks []types.ConversationChunk) string {
	if len(chunks) == 0 {
		return ""
	}

	if len(chunks) == 1 {
		return chunks[0].Summary
	}

	// Combine the most important summaries
	summaries := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.Summary != "" {
			summaries = append(summaries, chunk.Summary)
		}
	}

	if len(summaries) == 0 {
		return fmt.Sprintf("%d activities", len(chunks))
	}

	// Take first 2-3 summaries to avoid overly long phase summaries
	limit := 3
	if len(summaries) < limit {
		limit = len(summaries)
	}

	return strings.Join(summaries[:limit], "; ")
}

// inferTransitionReason determines why a phase transition occurred
func (l *LLMSummarizer) inferTransitionReason(from, to types.ConversationFlow) string {
	switch {
	case from == types.FlowProblem && to == types.FlowInvestigation:
		return "began investigation"
	case from == types.FlowInvestigation && to == types.FlowSolution:
		return "found solution approach"
	case from == types.FlowSolution && to == types.FlowVerification:
		return "implemented solution"
	case from == types.FlowVerification && to == types.FlowProblem:
		return "discovered new issue"
	case to == types.FlowProblem:
		return "encountered problem"
	case to == types.FlowSolution:
		return "moved to implementation"
	default:
		return "context change"
	}
}

// isKeyEvent determines if a chunk represents a key event
func (l *LLMSummarizer) isKeyEvent(chunk types.ConversationChunk) bool {
	// Key events are significant outcomes, decisions, or breakthroughs
	switch chunk.Type {
	case types.ChunkTypeSolution:
		return chunk.Metadata.Outcome == types.OutcomeSuccess
	case types.ChunkTypeArchitectureDecision:
		return true
	case types.ChunkTypeProblem:
		return chunk.Metadata.Difficulty == types.DifficultyComplex
	case types.ChunkTypeVerification:
		return chunk.Metadata.Outcome == types.OutcomeSuccess
	default:
		// Check for breakthrough language
		content := strings.ToLower(chunk.Content + " " + chunk.Summary)
		breakthroughWords := []string{"breakthrough", "discovered", "realized", "found the issue"}
		for _, word := range breakthroughWords {
			if strings.Contains(content, word) {
				return true
			}
		}
		return false
	}
}

// classifyEvent determines the type of key event
func (l *LLMSummarizer) classifyEvent(chunk types.ConversationChunk) EventType {
	switch chunk.Type {
	case types.ChunkTypeProblem:
		return EventTypeProblemFound
	case types.ChunkTypeSolution:
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
			return EventTypeSolutionFound
		}
		return EventTypeErrorResolved
	case types.ChunkTypeArchitectureDecision:
		return EventTypeDecisionMade
	case types.ChunkTypeVerification:
		return EventTypeErrorResolved
	default:
		// Check content for breakthrough indicators
		content := strings.ToLower(chunk.Content + " " + chunk.Summary)
		if strings.Contains(content, "breakthrough") || strings.Contains(content, "discovered") {
			return EventTypeBreakthroughMade
		}
		return EventTypeProblemFound
	}
}

// containsLearning checks if a chunk contains learning content
func (l *LLMSummarizer) containsLearning(chunk types.ConversationChunk) bool {
	content := strings.ToLower(chunk.Content + " " + chunk.Summary)
	learningWords := []string{"learned", "discovered", "realized", "understanding", "insight"}
	for _, word := range learningWords {
		if strings.Contains(content, word) {
			return true
		}
	}
	return false
}

// isTechnologyTag checks if a tag represents a technology
func (l *LLMSummarizer) isTechnologyTag(tag string) bool {
	techTags := map[string]bool{
		"go": true, "golang": true, "javascript": true, "typescript": true, "python": true,
		"docker": true, "kubernetes": true, "aws": true, "gcp": true, "azure": true,
		"postgres": true, "mysql": true, "redis": true, "mongodb": true,
		"react": true, "vue": true, "angular": true, "node": true,
		"chroma": true, "vector": true, "embedding": true, "mcp": true,
		"api": true, "rest": true, "graphql": true, "grpc": true,
	}
	return techTags[strings.ToLower(tag)]
}
