package decay

import (
	"context"
	"fmt"
	"mcp-memory/pkg/types"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
		ID:        uuid.New().String(),
		SessionID: chunks[0].SessionID, // Use first chunk's session
		Timestamp: time.Now(),
		Type:      types.ChunkTypeSessionSummary,
		Content:   summaryContent,
		Summary:   fmt.Sprintf("Summary of %d memories", len(chunks)),
		Metadata:  metadata,
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

// LLMSummarizer uses an LLM for more intelligent summarization
type LLMSummarizer struct {
	// This would integrate with an LLM API
	// For demonstration, we'll extend DefaultSummarizer
	*DefaultSummarizer
}

// NewLLMSummarizer creates a new LLM-based summarizer
func NewLLMSummarizer() *LLMSummarizer {
	return &LLMSummarizer{
		DefaultSummarizer: NewDefaultSummarizer(),
	}
}

// Summarize uses LLM to create an intelligent summary
func (l *LLMSummarizer) Summarize(ctx context.Context, chunks []types.ConversationChunk) (string, error) {
	// In a real implementation, this would:
	// 1. Prepare a prompt with chunk contents
	// 2. Send to LLM API (e.g., OpenAI)
	// 3. Return the generated summary
	
	// For now, we'll create a more detailed summary
	var parts []string
	
	// Group by type
	byType := make(map[types.ChunkType][]types.ConversationChunk)
	for _, chunk := range chunks {
		byType[chunk.Type] = append(byType[chunk.Type], chunk)
	}
	
	// Create type-specific summaries
	if problems := byType[types.ChunkTypeProblem]; len(problems) > 0 {
		parts = append(parts, fmt.Sprintf("Encountered %d problems", len(problems)))
	}
	
	if solutions := byType[types.ChunkTypeSolution]; len(solutions) > 0 {
		parts = append(parts, fmt.Sprintf("Implemented %d solutions", len(solutions)))
	}
	
	if decisions := byType[types.ChunkTypeArchitectureDecision]; len(decisions) > 0 {
		parts = append(parts, fmt.Sprintf("Made %d architectural decisions", len(decisions)))
	}
	
	// Add time context
	if len(chunks) > 0 {
		duration := chunks[len(chunks)-1].Timestamp.Sub(chunks[0].Timestamp)
		parts = append(parts, fmt.Sprintf("over %s", formatDuration(duration)))
	}
	
	// Add key learnings
	learnings := l.extractLearnings(chunks)
	if len(learnings) > 0 {
		parts = append(parts, fmt.Sprintf("Key learnings: %s", strings.Join(learnings, "; ")))
	}
	
	return strings.Join(parts, ". "), nil
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