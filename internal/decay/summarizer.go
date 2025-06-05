package decay

import (
	"context"
	"errors"
	"fmt"
	"lerian-mcp-memory/pkg/types"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// TaskStatusCompleted represents a completed task status
	TaskStatusCompleted = "completed"
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
	// EventTypeProblemFound represents when a problem is identified
	EventTypeProblemFound EventType = "problem_found"
	// EventTypeSolutionFound represents when a solution is discovered
	EventTypeSolutionFound EventType = "solution_found"
	// EventTypeDecisionMade represents when a decision is made
	EventTypeDecisionMade EventType = "decision_made"
	// EventTypeBreakthroughMade represents when a breakthrough occurs
	EventTypeBreakthroughMade EventType = "breakthrough_made"
	// EventTypeErrorResolved represents when an error is resolved
	EventTypeErrorResolved EventType = "error_resolved"
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
		return "", errors.New("no chunks to summarize")
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
	for i := range chunks {
		typeCounts[chunks[i].Type]++
	}

	// Add type summary
	typeInfo := make([]string, 0, len(typeCounts))
	for chunkType, count := range typeCounts {
		typeInfo = append(typeInfo, strconv.Itoa(count)+" "+string(chunkType))
	}
	summaryParts = append(summaryParts, "Contains: "+strings.Join(typeInfo, ", "))

	// Extract key topics
	topics := s.extractKeyTopics(chunks)
	if len(topics) > 0 {
		summaryParts = append(summaryParts, "Key topics: "+strings.Join(topics, ", "))
	}

	// Extract outcomes
	outcomes := s.extractOutcomes(chunks)
	if len(outcomes) > 0 {
		summaryParts = append(summaryParts, "Outcomes: "+strings.Join(outcomes, "; "))
	}

	// Extract tools used
	tools := s.extractTools(chunks)
	if len(tools) > 0 {
		summaryParts = append(summaryParts, "Tools used: "+strings.Join(tools, ", "))
	}

	return strings.Join(summaryParts, ". "), nil
}

// SummarizeChain creates a summary chunk from a chain of related chunks
func (s *DefaultSummarizer) SummarizeChain(ctx context.Context, chunks []types.ConversationChunk) (types.ConversationChunk, error) {
	if len(chunks) == 0 {
		return types.ConversationChunk{}, errors.New("no chunks to summarize")
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

	for i := range chunks {
		// Extract from summary
		words := strings.Fields(strings.ToLower(chunks[i].Summary))
		for _, word := range words {
			if len(word) > 5 && !isCommonWord(word) {
				topicFreq[word]++
			}
		}

		// Extract from tags
		for _, tag := range chunks[i].Metadata.Tags {
			topicFreq[strings.ToLower(tag)] += 2 // Tags are more important
		}
	}

	// Get top topics
	return getTopItems(topicFreq, 5)
}

// extractOutcomes extracts outcomes from chunks
func (s *DefaultSummarizer) extractOutcomes(chunks []types.ConversationChunk) []string {
	outcomes := make(map[string]bool)

	for i := range chunks {
		if chunks[i].Metadata.Outcome != "" {
			outcomes[string(chunks[i].Metadata.Outcome)] = true
		}

		// Look for solution chunks
		if chunks[i].Type == types.ChunkTypeSolution {
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

	for i := range chunks {
		for _, tool := range chunks[i].Metadata.ToolsUsed {
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

	for i := range chunks {
		if chunks[i].Metadata.Repository != "" {
			repositories[chunks[i].Metadata.Repository] = true
		}

		for _, tag := range chunks[i].Metadata.Tags {
			allTags[tag] = true
		}

		for _, tool := range chunks[i].Metadata.ToolsUsed {
			allTools[tool] = true
		}

		if chunks[i].Metadata.TimeSpent != nil {
			totalTime += *chunks[i].Metadata.TimeSpent
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
	for i := range chunks {
		ids[i] = chunks[i].ID
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
		return "", errors.New("no chunks to summarize")
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
	summary := l.generateIntelligentSummary(narrative, &criticalInfo)

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
	for i := range chunks {
		// Use summary if available, otherwise content
		text := chunks[i].Summary
		if text == "" {
			text = chunks[i].Content
		}
		if len(text) > 1000 {
			text = text[:1000] // Truncate for embedding
		}

		if embedding, err := l.embeddingGen.GenerateEmbedding(ctx, text); err == nil {
			embedded = append(embedded, embeddedChunk{chunk: chunks[i], embedding: embedding})
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

	for i := range embedded {
		if used[i] {
			continue
		}

		group := []types.ConversationChunk{embedded[i].chunk}
		used[i] = true

		// Find similar chunks
		for j := range embedded {
			if used[j] || i == j {
				continue
			}

			similarity := l.cosineSimilarity(embedded[i].embedding, embedded[j].embedding)
			if similarity >= threshold {
				group = append(group, embedded[j].chunk)
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
func (l *LLMSummarizer) extractNarrativeFlow(chunks []types.ConversationChunk, _ [][]types.ConversationChunk) NarrativeFlow {
	flow := NarrativeFlow{
		Phases:      make([]ConversationPhase, 0),
		KeyEvents:   make([]KeyEvent, 0),
		Transitions: make([]FlowTransition, 0),
	}

	// Analyze conversation phases
	currentPhase := l.detectPhase(&chunks[0])
	phaseStart := chunks[0].Timestamp
	phaseChunks := []types.ConversationChunk{chunks[0]}

	for i := 1; i < len(chunks); i++ {
		chunk := chunks[i]
		detectedPhase := l.detectPhase(&chunk)

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
		if l.isKeyEvent(&chunk) {
			flow.KeyEvents = append(flow.KeyEvents, KeyEvent{
				Type:        l.classifyEvent(&chunk),
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

	for i := range chunks {
		l.processChunkByCategoryType(&chunks[i], &info)
		l.extractTechnologiesFromChunk(&chunks[i], techMap)
		l.extractOutcomesFromChunk(&chunks[i], outcomeMap)
		l.extractRelationshipsFromChunk(&chunks[i], &info)
	}

	l.convertMapsToSlices(techMap, outcomeMap, &info)
	return info
}

// processChunkByCategoryType processes a chunk based on its category type
func (l *LLMSummarizer) processChunkByCategoryType(chunk *types.ConversationChunk, info *CriticalInformation) {
	switch chunk.Type {
	case types.ChunkTypeSolution:
		l.processSolutionChunk(chunk, info)
	case types.ChunkTypeArchitectureDecision:
		l.processArchitectureDecisionChunk(chunk, info)
	case types.ChunkTypeAnalysis:
		l.processAnalysisChunk(chunk, info)
	case types.ChunkTypeProblem:
		l.processProblemChunk(chunk, info)
	case types.ChunkTypeCodeChange:
		l.processCodeChangeChunk(chunk, info)
	case types.ChunkTypeDiscussion:
		l.processDiscussionChunk(chunk, info)
	case types.ChunkTypeTask:
		l.processTaskChunk(chunk, info)
	case types.ChunkTypeTaskUpdate, types.ChunkTypeTaskProgress:
		l.processTaskUpdateChunk(chunk, info)
	case types.ChunkTypeVerification:
		l.processVerificationChunk(chunk, info)
	case types.ChunkTypeQuestion:
		l.processQuestionChunk(chunk, info)
	case types.ChunkTypeSessionSummary:
		// Session summaries already contain aggregated info - no processing needed
	}
}

// processSolutionChunk handles solution-type chunks
func (l *LLMSummarizer) processSolutionChunk(chunk *types.ConversationChunk, info *CriticalInformation) {
	if chunk.Metadata.Outcome == types.OutcomeSuccess {
		info.Solutions = append(info.Solutions, *chunk)
	}
}

// processArchitectureDecisionChunk handles architecture decision chunks
func (l *LLMSummarizer) processArchitectureDecisionChunk(chunk *types.ConversationChunk, info *CriticalInformation) {
	info.Decisions = append(info.Decisions, *chunk)
}

// processAnalysisChunk handles analysis-type chunks
func (l *LLMSummarizer) processAnalysisChunk(chunk *types.ConversationChunk, info *CriticalInformation) {
	if l.containsLearning(chunk) {
		info.Learnings = append(info.Learnings, *chunk)
	}
}

// processProblemChunk handles problem-type chunks
func (l *LLMSummarizer) processProblemChunk(chunk *types.ConversationChunk, info *CriticalInformation) {
	if chunk.Metadata.Difficulty == types.DifficultyComplex {
		info.Errors = append(info.Errors, *chunk)
	}
}

// processCodeChangeChunk handles code change chunks
func (l *LLMSummarizer) processCodeChangeChunk(chunk *types.ConversationChunk, info *CriticalInformation) {
	// Code changes can be solutions if successful
	if chunk.Metadata.Outcome == types.OutcomeSuccess {
		info.Solutions = append(info.Solutions, *chunk)
	}
}

// processDiscussionChunk handles discussion-type chunks
func (l *LLMSummarizer) processDiscussionChunk(chunk *types.ConversationChunk, info *CriticalInformation) {
	// Discussions can contain learnings
	if l.containsLearning(chunk) {
		info.Learnings = append(info.Learnings, *chunk)
	}
}

// processTaskChunk handles task-type chunks
func (l *LLMSummarizer) processTaskChunk(chunk *types.ConversationChunk, info *CriticalInformation) {
	// Completed tasks are valuable solutions
	if chunk.Metadata.TaskStatus != nil && *chunk.Metadata.TaskStatus == TaskStatusCompleted {
		info.Solutions = append(info.Solutions, *chunk)
		return
	}
	// High priority tasks are important even if not completed
	if chunk.Metadata.TaskPriority != nil && *chunk.Metadata.TaskPriority == "high" {
		info.Decisions = append(info.Decisions, *chunk)
	}
}

// processTaskUpdateChunk handles task update and progress chunks
func (l *LLMSummarizer) processTaskUpdateChunk(chunk *types.ConversationChunk, info *CriticalInformation) {
	// Task updates and progress can contain learnings about process
	if l.containsLearning(chunk) {
		info.Learnings = append(info.Learnings, *chunk)
	}
}

// processVerificationChunk handles verification-type chunks
func (l *LLMSummarizer) processVerificationChunk(chunk *types.ConversationChunk, info *CriticalInformation) {
	// Verifications can be solutions if successful
	if chunk.Metadata.Outcome == types.OutcomeSuccess {
		info.Solutions = append(info.Solutions, *chunk)
	}
}

// processQuestionChunk handles question-type chunks
func (l *LLMSummarizer) processQuestionChunk(chunk *types.ConversationChunk, info *CriticalInformation) {
	// Questions that lead to insights are learnings
	if l.containsLearning(chunk) {
		info.Learnings = append(info.Learnings, *chunk)
	}
}

// extractTechnologiesFromChunk extracts technology tags from a chunk
func (l *LLMSummarizer) extractTechnologiesFromChunk(chunk *types.ConversationChunk, techMap map[string]bool) {
	for _, tag := range chunk.Metadata.Tags {
		if l.isTechnologyTag(tag) {
			techMap[tag] = true
		}
	}
}

// extractOutcomesFromChunk extracts outcomes from a chunk
func (l *LLMSummarizer) extractOutcomesFromChunk(chunk *types.ConversationChunk, outcomeMap map[string]bool) {
	if chunk.Metadata.Outcome != "" {
		outcomeMap[string(chunk.Metadata.Outcome)] = true
	}
}

// extractRelationshipsFromChunk extracts relationships from a chunk
func (l *LLMSummarizer) extractRelationshipsFromChunk(chunk *types.ConversationChunk, info *CriticalInformation) {
	if len(chunk.RelatedChunks) > 0 {
		info.Relationships[chunk.ID] = chunk.RelatedChunks
	}
}

// convertMapsToSlices converts technology and outcome maps to slices
func (l *LLMSummarizer) convertMapsToSlices(techMap, outcomeMap map[string]bool, info *CriticalInformation) {
	for tech := range techMap {
		info.Technologies = append(info.Technologies, tech)
	}
	for outcome := range outcomeMap {
		info.KeyOutcomes = append(info.KeyOutcomes, outcome)
	}
}

// generateIntelligentSummary creates the final intelligent summary
func (l *LLMSummarizer) generateIntelligentSummary(narrative NarrativeFlow, critical *CriticalInformation) string {
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
		parts = append(parts, "Flow: "+strings.Join(phaseDesc, " → "))
	}

	// Add critical outcomes
	if len(critical.Solutions) > 0 {
		parts = append(parts, "Implemented "+strconv.Itoa(len(critical.Solutions))+" successful solutions")
	}

	if len(critical.Decisions) > 0 {
		decisionDesc := make([]string, 0, len(critical.Decisions))
		for i := range critical.Decisions {
			if len(decisionDesc) < 3 { // Limit to 3 most important
				decisionDesc = append(decisionDesc, critical.Decisions[i].Summary)
			}
		}
		parts = append(parts, "Key decisions: "+strings.Join(decisionDesc, "; "))
	}

	// Add key learnings
	if len(critical.Learnings) > 0 {
		learningDesc := make([]string, 0, len(critical.Learnings))
		for i := range critical.Learnings {
			if len(learningDesc) < 2 { // Limit to 2 most important
				learningDesc = append(learningDesc, critical.Learnings[i].Summary)
			}
		}
		parts = append(parts, "Learnings: "+strings.Join(learningDesc, "; "))
	}

	// Add technology context
	if len(critical.Technologies) > 0 {
		parts = append(parts, "Technologies: "+strings.Join(critical.Technologies, ", "))
	}

	// Add key events if significant
	if len(narrative.KeyEvents) > 0 {
		eventCount := make(map[string]int)
		for i := range narrative.KeyEvents {
			eventCount[string(narrative.KeyEvents[i].Type)]++
		}
		eventDesc := make([]string, 0, len(eventCount))
		for eventType, count := range eventCount {
			eventDesc = append(eventDesc, fmt.Sprintf("%d %s", count, eventType))
		}
		parts = append(parts, "Events: "+strings.Join(eventDesc, ", "))
	}

	return strings.Join(parts, ". ")
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
func (l *LLMSummarizer) detectPhase(chunk *types.ConversationChunk) types.ConversationFlow {
	// First try content-based detection
	if phase := l.detectPhaseFromContent(chunk); phase != "" {
		return phase
	}

	// Fall back to type-based detection
	return l.detectPhaseFromType(chunk)
}

// detectPhaseFromContent analyzes chunk content to determine conversation phase
func (l *LLMSummarizer) detectPhaseFromContent(chunk *types.ConversationChunk) types.ConversationFlow {
	content := strings.ToLower(chunk.Content + " " + chunk.Summary)

	if l.containsProblemIndicators(content) {
		return types.FlowProblem
	}
	if l.containsInvestigationIndicators(content) {
		return types.FlowInvestigation
	}
	if l.containsSolutionIndicators(content) {
		return types.FlowSolution
	}
	if l.containsVerificationIndicators(content) {
		return types.FlowVerification
	}

	return "" // No content-based detection
}

// containsProblemIndicators checks if content contains problem-related keywords
func (l *LLMSummarizer) containsProblemIndicators(content string) bool {
	problemWords := []string{"error", "issue", "problem", "bug", "failed", "exception"}
	return l.containsAnyWord(content, problemWords)
}

// containsInvestigationIndicators checks if content contains investigation-related keywords
func (l *LLMSummarizer) containsInvestigationIndicators(content string) bool {
	investigationWords := []string{"investigating", "looking", "checking", "analyzing", "debugging"}
	return l.containsAnyWord(content, investigationWords)
}

// containsSolutionIndicators checks if content contains solution-related keywords
func (l *LLMSummarizer) containsSolutionIndicators(content string) bool {
	solutionWords := []string{"fix", "solution", "implement", "create", "resolve"}
	return l.containsAnyWord(content, solutionWords)
}

// containsVerificationIndicators checks if content contains verification-related keywords
func (l *LLMSummarizer) containsVerificationIndicators(content string) bool {
	verificationWords := []string{"test", "verify", "check", "confirm", "validate"}
	return l.containsAnyWord(content, verificationWords)
}

// containsAnyWord checks if content contains any of the specified words
func (l *LLMSummarizer) containsAnyWord(content string, words []string) bool {
	for _, word := range words {
		if strings.Contains(content, word) {
			return true
		}
	}
	return false
}

// detectPhaseFromType determines conversation phase based on chunk type
func (l *LLMSummarizer) detectPhaseFromType(chunk *types.ConversationChunk) types.ConversationFlow {
	switch chunk.Type {
	case types.ChunkTypeProblem, types.ChunkTypeQuestion, types.ChunkTypeTask:
		return types.FlowProblem
	case types.ChunkTypeSolution, types.ChunkTypeCodeChange, types.ChunkTypeArchitectureDecision:
		return types.FlowSolution
	case types.ChunkTypeVerification, types.ChunkTypeTaskProgress:
		return types.FlowVerification
	case types.ChunkTypeDiscussion, types.ChunkTypeSessionSummary, types.ChunkTypeAnalysis, types.ChunkTypeTaskUpdate:
		return types.FlowInvestigation
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
	for i := range chunks {
		if chunks[i].Summary != "" {
			summaries = append(summaries, chunks[i].Summary)
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
func (l *LLMSummarizer) isKeyEvent(chunk *types.ConversationChunk) bool {
	// Check type-based key events first
	if l.isTypeBasedKeyEvent(chunk) {
		return true
	}

	// Check task-related key events
	if l.isTaskRelatedKeyEvent(chunk) {
		return true
	}

	// Check content-based breakthrough events
	return l.isContentBasedKeyEvent(chunk)
}

// isTypeBasedKeyEvent checks for key events based on chunk type and metadata
func (l *LLMSummarizer) isTypeBasedKeyEvent(chunk *types.ConversationChunk) bool {
	switch chunk.Type {
	case types.ChunkTypeArchitectureDecision:
		return true
	case types.ChunkTypeSolution, types.ChunkTypeVerification, types.ChunkTypeCodeChange:
		return chunk.Metadata.Outcome == types.OutcomeSuccess
	case types.ChunkTypeProblem:
		return chunk.Metadata.Difficulty == types.DifficultyComplex
	case types.ChunkTypeAnalysis:
		return l.containsLearning(chunk)
	case types.ChunkTypeDiscussion, types.ChunkTypeSessionSummary, types.ChunkTypeQuestion:
		return false
	}
	return false
}

// isTaskRelatedKeyEvent checks for key events in task-oriented chunk types
func (l *LLMSummarizer) isTaskRelatedKeyEvent(chunk *types.ConversationChunk) bool {
	switch chunk.Type {
	case types.ChunkTypeTask:
		return l.isHighPriorityTask(chunk) || l.isCompletedTask(chunk)
	case types.ChunkTypeTaskUpdate:
		return chunk.Metadata.TaskStatus != nil
	case types.ChunkTypeTaskProgress:
		return l.isHighProgressMilestone(chunk)
	}
	return false
}

// isHighPriorityTask checks if a task has high priority
func (l *LLMSummarizer) isHighPriorityTask(chunk *types.ConversationChunk) bool {
	return chunk.Metadata.TaskPriority != nil && *chunk.Metadata.TaskPriority == "high"
}

// isCompletedTask checks if a task is completed
func (l *LLMSummarizer) isCompletedTask(chunk *types.ConversationChunk) bool {
	return chunk.Metadata.TaskStatus != nil && *chunk.Metadata.TaskStatus == TaskStatusCompleted
}

// isHighProgressMilestone checks if progress indicates a significant milestone
func (l *LLMSummarizer) isHighProgressMilestone(chunk *types.ConversationChunk) bool {
	return chunk.Metadata.TaskProgress != nil && *chunk.Metadata.TaskProgress >= 80
}

// isContentBasedKeyEvent checks for breakthrough language in content
func (l *LLMSummarizer) isContentBasedKeyEvent(chunk *types.ConversationChunk) bool {
	content := strings.ToLower(chunk.Content + " " + chunk.Summary)
	breakthroughWords := []string{"breakthrough", "discovered", "realized", "found the issue"}
	return l.containsAnyWord(content, breakthroughWords)
}

// classifyBasicChunkTypes handles simple chunk type to event type mappings
func (l *LLMSummarizer) classifyBasicChunkTypes(chunkType types.ChunkType) (EventType, bool) {
	switch chunkType {
	case types.ChunkTypeProblem:
		return EventTypeProblemFound, true
	case types.ChunkTypeArchitectureDecision:
		return EventTypeDecisionMade, true
	case types.ChunkTypeVerification:
		return EventTypeErrorResolved, true
	case types.ChunkTypeDiscussion:
		return EventTypeBreakthroughMade, true // Discussions often lead to insights
	case types.ChunkTypeTaskProgress:
		return EventTypeBreakthroughMade, true // Progress tracking shows advancement
	case types.ChunkTypeSessionSummary:
		return EventTypeSolutionFound, true // Summaries capture completed work
	case types.ChunkTypeAnalysis:
		return EventTypeBreakthroughMade, true // Analysis often yields insights
	case types.ChunkTypeQuestion:
		return EventTypeProblemFound, true // Questions identify issues
	default:
		return EventTypeProblemFound, false // Not handled by basic classification
	}
}

// classifyOutcomeBasedChunkTypes handles chunk types that depend on outcome metadata
func (l *LLMSummarizer) classifyOutcomeBasedChunkTypes(chunk *types.ConversationChunk) (EventType, bool) {
	switch chunk.Type {
	case types.ChunkTypeSolution:
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
			return EventTypeSolutionFound, true
		}
		return EventTypeErrorResolved, true
	case types.ChunkTypeCodeChange:
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
			return EventTypeSolutionFound, true
		}
		return EventTypeErrorResolved, true
	default:
		return EventTypeProblemFound, false // Not an outcome-based chunk type
	}
}

// classifyTaskBasedChunkTypes handles task-oriented chunk types
func (l *LLMSummarizer) classifyTaskBasedChunkTypes(chunk *types.ConversationChunk) (EventType, bool) {
	switch chunk.Type {
	case types.ChunkTypeTask:
		if chunk.Metadata.TaskStatus != nil && *chunk.Metadata.TaskStatus == TaskStatusCompleted {
			return EventTypeSolutionFound, true // Completed tasks are solutions
		}
		return EventTypeProblemFound, true // New tasks are problems to solve
	case types.ChunkTypeTaskUpdate:
		if chunk.Metadata.TaskStatus != nil && *chunk.Metadata.TaskStatus == TaskStatusCompleted {
			return EventTypeSolutionFound, true
		}
		return EventTypeBreakthroughMade, true // Progress/updates are breakthroughs
	default:
		return EventTypeProblemFound, false // Not a task-based chunk type
	}
}

// classifyByContent determines event type based on content analysis
func (l *LLMSummarizer) classifyByContent(chunk *types.ConversationChunk) EventType {
	content := strings.ToLower(chunk.Content + " " + chunk.Summary)
	if strings.Contains(content, "breakthrough") || strings.Contains(content, "discovered") {
		return EventTypeBreakthroughMade
	}
	return EventTypeProblemFound
}

// classifyEvent determines the type of key event
func (l *LLMSummarizer) classifyEvent(chunk *types.ConversationChunk) EventType {
	// Try basic chunk type classification first
	if eventType, handled := l.classifyBasicChunkTypes(chunk.Type); handled {
		return eventType
	}

	// Try outcome-based classification
	if eventType, handled := l.classifyOutcomeBasedChunkTypes(chunk); handled {
		return eventType
	}

	// Try task-based classification
	if eventType, handled := l.classifyTaskBasedChunkTypes(chunk); handled {
		return eventType
	}

	// Fall back to content-based classification
	return l.classifyByContent(chunk)
}

// containsLearning checks if a chunk contains learning content
func (l *LLMSummarizer) containsLearning(chunk *types.ConversationChunk) bool {
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
