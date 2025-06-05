// Package chunking provides intelligent content chunking strategies
// for optimizing memory storage and retrieval in the MCP Memory Server.
package chunking

import (
	"context"
	"fmt"
	"mcp-memory/internal/config"
	"mcp-memory/internal/embeddings"
	"mcp-memory/pkg/types"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Constants for significance levels
const (
	SignificanceHigh   = "high"
	SignificanceMedium = "medium"
	SignificanceLow    = "low"
)

// Service handles the intelligent chunking of conversations
type Service struct {
	config           *config.ChunkingConfig
	embeddingService embeddings.EmbeddingService

	// State tracking for smart chunking
	currentContext *types.ChunkingContext
	contextHistory []types.ChunkingContext
	lastChunkTime  time.Time

	// Content analysis patterns
	problemPatterns  []*regexp.Regexp
	solutionPatterns []*regexp.Regexp
	codePatterns     []*regexp.Regexp

	// Smart detection patterns
	highImpactPatterns    []*regexp.Regexp
	reusablePatterns      []*regexp.Regexp
	gotchaPatterns        []*regexp.Regexp
	architecturalPatterns []*regexp.Regexp
	performancePatterns   []*regexp.Regexp
}

// NewService creates a new chunking service
func NewService(cfg *config.ChunkingConfig, embeddingService embeddings.EmbeddingService) *Service {
	cs := &Service{
		config:           cfg,
		embeddingService: embeddingService,
		currentContext:   &types.ChunkingContext{},
		contextHistory:   []types.ChunkingContext{},
		lastChunkTime:    time.Now(),
	}

	cs.initializePatterns()
	return cs
}

// initializePatterns sets up regex patterns for content analysis
func (cs *Service) initializePatterns() {
	// Problem identification patterns
	cs.problemPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(error|failed|issue|problem|bug|broken)`),
		regexp.MustCompile(`(?i)(not working|doesn't work|can't|unable to)`),
		regexp.MustCompile(`(?i)(exception|stack trace|traceback)`),
		regexp.MustCompile(`(?i)(help.*with|how.*to|need.*to)`),
	}

	// Solution identification patterns
	cs.solutionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(fixed|solved|resolved|implemented)`),
		regexp.MustCompile(`(?i)(here's.*fix|solution.*is|to solve)`),
		regexp.MustCompile(`(?i)(working.*now|successfully|completed)`),
		regexp.MustCompile(`(?i)(let me.*implement|i'll.*create|let's.*add)`),
	}

	// Code change patterns
	cs.codePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(function|class|method|variable)`),
		regexp.MustCompile(`(?i)(import|require|include)`),
		regexp.MustCompile("(?i)(```|`.*`)"), // Code blocks
		regexp.MustCompile(`(?i)(file.*modified|changes.*to|updated.*file)`),
	}

	// High-impact decision patterns
	cs.highImpactPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(decided to|chose|switched from .* to|migrated from)`),
		regexp.MustCompile(`(?i)(architectural decision|design choice|went with)`),
		regexp.MustCompile(`(?i)(breaking change|major refactor|significant update)`),
		regexp.MustCompile(`(?i)(critical|important|significant|major)`),
		regexp.MustCompile(`(?i)(production|deployment|release|launch)`),
		regexp.MustCompile(`(?i)(security|vulnerability|exploit|authentication)`),
	}

	// Reusable pattern detection
	cs.reusablePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(pattern|template|boilerplate|reusable)`),
		regexp.MustCompile(`(?i)(common.*approach|standard.*way|typical.*solution)`),
		regexp.MustCompile(`(?i)(utility|helper|library|framework)`),
		regexp.MustCompile(`(?i)(best practice|recommended|guideline)`),
		regexp.MustCompile(`(?i)(config|configuration|setup|initialization)`),
	}

	// Gotcha and pitfall patterns
	cs.gotchaPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(watch out|careful|gotcha|common mistake|pitfall)`),
		regexp.MustCompile(`(?i)(troubleshooting|debugging tip|lesson learned)`),
		regexp.MustCompile(`(?i)(make sure to|don't forget|important note)`),
		regexp.MustCompile(`(?i)(avoid|never|don't.*do|warning)`),
		regexp.MustCompile(`(?i)(edge case|corner case|exception|special case)`),
		regexp.MustCompile(`(?i)(tricky|subtle|unexpected|surprising)`),
	}

	// Architectural decision patterns
	cs.architecturalPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(architecture|design|structure|organization)`),
		regexp.MustCompile(`(?i)(microservice|monolith|distributed|centralized)`),
		regexp.MustCompile(`(?i)(database|storage|persistence|cache)`),
		regexp.MustCompile(`(?i)(api|interface|contract|protocol)`),
		regexp.MustCompile(`(?i)(scalability|performance|reliability|availability)`),
		regexp.MustCompile(`(?i)(technology.*stack|tech.*choice|framework.*selection)`),
	}

	// Performance-related patterns
	cs.performancePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)(performance|optimization|speed|latency)`),
		regexp.MustCompile(`(?i)(memory.*usage|cpu.*usage|resource.*consumption)`),
		regexp.MustCompile(`(?i)(benchmark|profiling|monitoring|metrics)`),
		regexp.MustCompile(`(?i)(bottleneck|slow.*down|inefficient)`),
		regexp.MustCompile(`(?i)(caching|indexing|query.*optimization)`),
	}
}

// ShouldCreateChunk determines if a new chunk should be created based on context
func (cs *Service) ShouldCreateChunk(chunkingContext *types.ChunkingContext) bool {
	// Update current context
	cs.currentContext = chunkingContext

	// Todo completion trigger (highest priority)
	if cs.config.TodoCompletionTrigger && chunkingContext.HasCompletedTodos() {
		return true
	}

	// Significant file changes
	if len(chunkingContext.FileModifications) >= cs.config.FileChangeThreshold {
		return true
	}

	// Time-based chunking
	if chunkingContext.TimeElapsed >= cs.config.TimeThresholdMinutes {
		return true
	}

	// Problem resolution cycle complete
	if chunkingContext.ConversationFlow == types.FlowVerification && chunkingContext.TimeElapsed > 5 {
		return true
	}

	// Context switch detected
	if cs.hasContextSwitch(chunkingContext) {
		return true
	}

	// Content volume threshold
	totalContent := 0
	for _, tool := range chunkingContext.ToolsUsed {
		totalContent += len(tool) * 10 // Rough estimation
	}

	return totalContent > cs.config.MaxContentLength
}

// CreateChunk creates a conversation chunk from the current context
func (cs *Service) CreateChunk(ctx context.Context, sessionID, content string, metadata *types.ChunkMetadata) (*types.ConversationChunk, error) {
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	// Analyze content to determine chunk type
	chunkType := cs.analyzeContentType(content)

	// Enrich metadata with analysis
	enrichedMetadata := cs.enrichMetadata(metadata, content)

	// Create the chunk
	chunk, err := types.NewConversationChunk(sessionID, content, chunkType, enrichedMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunk: %w", err)
	}

	// Generate summary
	summary := cs.generateSummary(ctx, content, chunkType)
	chunk.Summary = summary

	// Generate embeddings
	embedding, err := cs.embeddingService.GenerateEmbedding(ctx, cs.prepareContentForEmbedding(chunk))
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}
	chunk.Embeddings = embedding

	// Update internal state
	cs.lastChunkTime = time.Now()
	if cs.currentContext != nil {
		cs.contextHistory = append(cs.contextHistory, *cs.currentContext)
	}

	// Keep only last 10 contexts for analysis
	if len(cs.contextHistory) > 10 {
		cs.contextHistory = cs.contextHistory[1:]
	}

	return chunk, nil
}

// analyzeContentType determines the type of chunk based on content analysis
func (cs *Service) analyzeContentType(content string) types.ChunkType {
	content = strings.ToLower(content)

	// Check for architecture decisions
	archKeywords := []string{"decision", "architecture", "design", "approach"}
	for _, keyword := range archKeywords {
		if strings.Contains(content, keyword) {
			return types.ChunkTypeArchitectureDecision
		}
	}

	// Check for code changes
	for _, pattern := range cs.codePatterns {
		if pattern.MatchString(content) {
			return types.ChunkTypeCodeChange
		}
	}

	// Check for solutions
	for _, pattern := range cs.solutionPatterns {
		if pattern.MatchString(content) {
			return types.ChunkTypeSolution
		}
	}

	// Check for problems
	for _, pattern := range cs.problemPatterns {
		if pattern.MatchString(content) {
			return types.ChunkTypeProblem
		}
	}

	// Default to discussion
	return types.ChunkTypeDiscussion
}

// enrichMetadata adds analysis-based metadata to the chunk
func (cs *Service) enrichMetadata(metadata *types.ChunkMetadata, content string) types.ChunkMetadata {
	// Add current context tools and files if not already present
	if cs.currentContext != nil {
		if len(metadata.ToolsUsed) == 0 {
			metadata.ToolsUsed = cs.currentContext.ToolsUsed
		}
		if len(metadata.FilesModified) == 0 {
			metadata.FilesModified = cs.currentContext.FileModifications
		}
	}

	// Auto-generate tags based on content
	tags := cs.extractTags(content)
	for _, tag := range tags {
		// Avoid duplicates
		found := false
		for _, existing := range metadata.Tags {
			if existing == tag {
				found = true
				break
			}
		}
		if !found {
			metadata.Tags = append(metadata.Tags, tag)
		}
	}

	// Determine difficulty based on content complexity
	if metadata.Difficulty == "" {
		metadata.Difficulty = cs.assessDifficulty(content)
	}

	// Set outcome based on content analysis (always assess to override default OutcomeInProgress)
	detectedOutcome := cs.assessOutcome(content)
	if detectedOutcome != types.OutcomeInProgress || metadata.Outcome == "" {
		metadata.Outcome = detectedOutcome
	}

	// Smart detection: Add specialized tags for high-impact content
	smartTags := cs.detectSmartTags(content)
	for _, tag := range smartTags {
		// Avoid duplicates
		found := false
		for _, existing := range metadata.Tags {
			if existing == tag {
				found = true
				break
			}
		}
		if !found {
			metadata.Tags = append(metadata.Tags, tag)
		}
	}

	// Calculate impact score and reusability score
	metadata.ExtendedMetadata = cs.buildExtendedMetadata(content, metadata)

	return *metadata
}

// extractTags extracts relevant tags from content
func (cs *Service) extractTags(content string) []string {
	tags := []string{}
	content = strings.ToLower(content)

	// Technology tags
	techPatterns := map[string]string{
		"go":         `\bgo\b|\bgolang\b`,
		"typescript": `\btypescript\b|\bts\b`,
		"javascript": `\bjavascript\b|\bjs\b`,
		"python":     `\bpython\b|\bpy\b`,
		"docker":     `\bdocker\b|\bcontainer\b`,
		"git":        `\bgit\b|\bcommit\b|\bbranch\b`,
		"api":        `\bapi\b|\bendpoint\b|\brest\b`,
		"database":   `\bdatabase\b|\bdb\b|\bsql\b`,
		"test":       `\btest\b|\btesting\b|\bspec\b`,
		"bug":        `\bbug\b|\berror\b|\bissue\b`,
		"feature":    `\bfeature\b|\bnew\b|\badd\b`,
		"refactor":   `\brefactor\b|\bcleanup\b|\bimprove\b`,
	}

	for tag, pattern := range techPatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			tags = append(tags, tag)
		}
	}

	// Framework/library detection
	frameworks := []string{"react", "vue", "angular", "express", "fastapi", "django", "flask"}
	for _, framework := range frameworks {
		if strings.Contains(content, framework) {
			tags = append(tags, framework)
		}
	}

	return tags
}

// assessDifficulty determines the difficulty level based on content
func (cs *Service) assessDifficulty(content string) types.Difficulty {
	complexityScore := 0

	// Indicators of complexity
	complexIndicators := []string{
		"complex", "complicated", "challenging", "difficult",
		"architecture", "design pattern", "algorithm",
		"performance", "optimization", "scale",
		"async", "concurrent", "parallel",
		"security", "authentication", "authorization",
	}

	for _, indicator := range complexIndicators {
		if strings.Contains(strings.ToLower(content), indicator) {
			complexityScore++
		}
	}

	// Check content length as complexity indicator
	if len(content) > 2000 {
		complexityScore++
	}

	// Check for multiple tools/files
	if cs.currentContext != nil {
		if len(cs.currentContext.ToolsUsed) > 5 {
			complexityScore++
		}
		if len(cs.currentContext.FileModifications) > 3 {
			complexityScore++
		}
	}

	if complexityScore >= 3 {
		return types.DifficultyComplex
	} else if complexityScore >= 1 {
		return types.DifficultyModerate
	}

	return types.DifficultySimple
}

// assessOutcome determines the outcome based on content
func (cs *Service) assessOutcome(content string) types.Outcome {
	content = strings.ToLower(content)

	// Strong success indicators (weighted higher)
	strongSuccessIndicators := []string{
		"completed successfully", "fixed the issue", "problem solved", "working now",
		"tests pass", "build successful", "merged", "deployed", "resolved",
	}

	// Success indicators
	successIndicators := []string{
		"completed", "fixed", "solved", "working", "success", "done", "implemented",
		"added", "updated", "created", "built", "tested", "verified", "finished",
	}

	// Failure indicators
	failureIndicators := []string{
		"failed", "error", "broken", "not working", "issue", "problem", "bug",
		"crashed", "timeout", "exception", "cannot", "unable", "stuck",
	}

	// Progress indicators
	progressIndicators := []string{
		"in progress", "working on", "implementing", "developing", "testing",
		"investigating", "debugging", "trying", "attempting", "starting",
	}

	// Check for strong success first
	for _, indicator := range strongSuccessIndicators {
		if strings.Contains(content, indicator) {
			return types.OutcomeSuccess
		}
	}

	// Count indicators to make better decisions
	successCount := 0
	failureCount := 0
	progressCount := 0

	for _, indicator := range successIndicators {
		if strings.Contains(content, indicator) {
			successCount++
		}
	}

	for _, indicator := range failureIndicators {
		if strings.Contains(content, indicator) {
			failureCount++
		}
	}

	for _, indicator := range progressIndicators {
		if strings.Contains(content, indicator) {
			progressCount++
		}
	}

	// Decision logic based on counts
	if successCount > failureCount && successCount > progressCount {
		return types.OutcomeSuccess
	}
	if failureCount > 0 && failureCount > progressCount {
		return types.OutcomeFailed
	}
	if progressCount > 0 {
		return types.OutcomeInProgress
	}

	return types.OutcomeInProgress // Default assumption
}

// generateSummary creates an AI-powered summary of the content
func (cs *Service) generateSummary(_ context.Context, content string, _ types.ChunkType) string {
	// For now, implement a simple extractive summary
	// In a full implementation, this would use an LLM for abstractive summarization
	return cs.generateSimpleSummary(content)
}

// generateSimpleSummary creates a simple extractive summary
func (cs *Service) generateSimpleSummary(content string) string {
	lines := strings.Split(content, "\n")

	// Take first meaningful line as summary
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 20 && len(trimmed) < 150 {
			return trimmed
		}
	}

	// Fallback: truncate content
	if len(content) > 100 {
		return content[:97] + "..."
	}

	return content
}

// prepareContentForEmbedding formats content optimally for embedding generation
func (cs *Service) prepareContentForEmbedding(chunk *types.ConversationChunk) string {
	parts := []string{}

	// Include chunk type for context
	parts = append(parts, fmt.Sprintf("Type: %s", chunk.Type))

	// Include summary if available (cleaned)
	if chunk.Summary != "" {
		cleanSummary := cs.cleanContentForEmbedding(chunk.Summary)
		parts = append(parts, fmt.Sprintf("Summary: %s", cleanSummary))
	}

	// Include main content (cleaned)
	cleanContent := cs.cleanContentForEmbedding(chunk.Content)
	parts = append(parts, fmt.Sprintf("Content: %s", cleanContent))

	// Include relevant metadata
	if chunk.Metadata.Repository != "" {
		parts = append(parts, fmt.Sprintf("Repository: %s", chunk.Metadata.Repository))
	}

	if len(chunk.Metadata.Tags) > 0 {
		// Clean tags too
		cleanTags := make([]string, len(chunk.Metadata.Tags))
		for i, tag := range chunk.Metadata.Tags {
			cleanTags[i] = cs.cleanContentForEmbedding(tag)
		}
		parts = append(parts, fmt.Sprintf("Tags: %s", strings.Join(cleanTags, ", ")))
	}

	combined := strings.Join(parts, " ")

	// Truncate if too long for embedding model
	maxLength := getEnvInt("MCP_MEMORY_MAX_EMBEDDING_CONTENT_LENGTH", 8000) // Conservative limit for most embedding models
	if len(combined) > maxLength {
		return combined[:maxLength]
	}

	return combined
}

// cleanContentForEmbedding removes formatting and emojis that pollute vector similarity
func (cs *Service) cleanContentForEmbedding(content string) string {
	// Remove common emojis and symbols that pollute semantic search
	emojiPattern := `[🚀🔍🔒🔧💡⚡✅❌🎯📊📈📉🛠️🔄🌟⭐💻📝🗂️🎉🔥💪🎨🚨⚠️✨🔎🆕🔵🟢🔴🟡]`
	re := regexp.MustCompile(emojiPattern)
	cleaned := re.ReplaceAllString(content, "")

	// Remove excessive whitespace and normalize
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	cleaned = strings.TrimSpace(cleaned)

	// Remove markdown formatting that doesn't add semantic value
	// Remove bold/italic markers
	cleaned = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(cleaned, "$1")
	cleaned = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(cleaned, "$1")
	cleaned = regexp.MustCompile(`_([^_]+)_`).ReplaceAllString(cleaned, "$1")

	// Remove code blocks but keep inline code content
	cleaned = regexp.MustCompile("```[^`]*```").ReplaceAllString(cleaned, "code_block")
	cleaned = regexp.MustCompile("`([^`]+)`").ReplaceAllString(cleaned, "$1")

	// Remove URLs but keep domain info for context
	cleaned = regexp.MustCompile(`https?://([^/\s]+)\S*`).ReplaceAllString(cleaned, "website_$1")

	// Remove excessive punctuation
	cleaned = regexp.MustCompile(`!{2,}`).ReplaceAllString(cleaned, "!")
	cleaned = regexp.MustCompile(`\?{2,}`).ReplaceAllString(cleaned, "?")
	cleaned = regexp.MustCompile(`-{3,}`).ReplaceAllString(cleaned, "")

	// Normalize common abbreviations for better semantic matching
	replacements := map[string]string{
		"w/":   "with",
		"&":    "and",
		"@":    "at",
		"#":    "number",
		"e.g.": "for example",
		"i.e.": "that is",
		"etc.": "and so on",
	}

	for old, new := range replacements {
		cleaned = strings.ReplaceAll(cleaned, old, new)
	}

	return cleaned
}

// hasContextSwitch detects if there has been a significant context switch
func (cs *Service) hasContextSwitch(chunkingContext *types.ChunkingContext) bool {
	if len(cs.contextHistory) == 0 {
		return false
	}

	lastContext := cs.contextHistory[len(cs.contextHistory)-1]

	// Check for conversation flow changes
	if lastContext.ConversationFlow != chunkingContext.ConversationFlow {
		return true
	}

	// Check for significant tool change
	if len(chunkingContext.ToolsUsed) > 0 && len(lastContext.ToolsUsed) > 0 {
		commonTools := 0
		for _, tool := range chunkingContext.ToolsUsed {
			for _, lastTool := range lastContext.ToolsUsed {
				if tool == lastTool {
					commonTools++
					break
				}
			}
		}

		// If less than 30% tools in common, it's a context switch
		if float64(commonTools)/float64(len(chunkingContext.ToolsUsed)) < 0.3 {
			return true
		}
	}

	// Check for file context changes
	if len(chunkingContext.FileModifications) > 0 && len(lastContext.FileModifications) > 0 {
		commonFiles := 0
		for _, file := range chunkingContext.FileModifications {
			for _, lastFile := range lastContext.FileModifications {
				if file == lastFile {
					commonFiles++
					break
				}
			}
		}

		// If no files in common, it's a context switch
		if commonFiles == 0 {
			return true
		}
	}

	return false
}

// GetCurrentContext returns the current chunking context
func (cs *Service) GetCurrentContext() *types.ChunkingContext {
	return cs.currentContext
}

// UpdateContext updates the current context with new information
func (cs *Service) UpdateContext(updates map[string]interface{}) {
	if cs.currentContext == nil {
		cs.currentContext = &types.ChunkingContext{}
	}

	if todos, ok := updates["todos"].([]types.TodoItem); ok {
		cs.currentContext.CurrentTodos = todos
	}

	if files, ok := updates["files"].([]string); ok {
		cs.currentContext.FileModifications = files
	}

	if tools, ok := updates["tools"].([]string); ok {
		cs.currentContext.ToolsUsed = tools
	}

	if flow, ok := updates["flow"].(types.ConversationFlow); ok {
		cs.currentContext.ConversationFlow = flow
	}

	if elapsed, ok := updates["elapsed"].(int); ok {
		cs.currentContext.TimeElapsed = elapsed
	}
}

// ProcessConversation processes a conversation into multiple chunks intelligently
func (cs *Service) ProcessConversation(ctx context.Context, sessionID, conversation string, baseMetadata *types.ChunkMetadata) ([]types.ConversationChunk, error) {
	if conversation == "" {
		return nil, fmt.Errorf("conversation cannot be empty")
	}

	chunks := []types.ConversationChunk{}

	// Split conversation by natural boundaries
	segments := cs.splitConversation(conversation)

	// Process each segment
	for _, segment := range segments {
		if strings.TrimSpace(segment) == "" {
			continue
		}

		// Create chunk for this segment
		chunk, err := cs.CreateChunk(ctx, sessionID, segment, baseMetadata)
		if err != nil {
			return nil, fmt.Errorf("failed to create chunk: %w", err)
		}

		chunks = append(chunks, *chunk)

		// Check if we should create a summary chunk after multiple segments
		if len(chunks) > 0 && len(chunks)%5 == 0 {
			summaryChunk := cs.createSummaryChunk(ctx, sessionID, chunks[len(chunks)-5:], baseMetadata)
			if summaryChunk != nil {
				chunks = append(chunks, *summaryChunk)
			}
		}
	}

	// Create final session summary if we have multiple chunks
	if len(chunks) > 3 {
		summaryChunk := cs.createSessionSummary(ctx, sessionID, chunks, baseMetadata)
		if summaryChunk != nil {
			chunks = append(chunks, *summaryChunk)
		}
	}

	return chunks, nil
}

// splitConversation splits a conversation into logical segments
func (cs *Service) splitConversation(conversation string) []string {
	segments := []string{}
	currentSegment := ""
	lines := strings.Split(conversation, "\n")

	// Patterns that indicate segment boundaries
	boundaryPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^(Human|Assistant|User|AI|Claude):`),
		regexp.MustCompile(`^###|^---|^===`), // Section markers
		regexp.MustCompile(`^\d+\.\s`),       // Numbered lists
		regexp.MustCompile(`^(Step|Task|Problem|Solution)[\s:]`),
	}

	for i, line := range lines {
		// Check if this line marks a boundary
		isBoundary := false
		for _, pattern := range boundaryPatterns {
			if pattern.MatchString(line) {
				isBoundary = true
				break
			}
		}

		// If boundary and we have content, save segment
		if isBoundary && currentSegment != "" {
			segments = append(segments, strings.TrimSpace(currentSegment))
			currentSegment = line + "\n"
		} else {
			currentSegment += line + "\n"
		}

		// Check for size-based splitting
		if len(currentSegment) > cs.config.MaxContentLength {
			segments = append(segments, strings.TrimSpace(currentSegment))
			currentSegment = ""
		}

		// Check for natural paragraph breaks (multiple newlines)
		if i < len(lines)-1 && line == "" && lines[i+1] == "" && len(currentSegment) > 500 {
			segments = append(segments, strings.TrimSpace(currentSegment))
			currentSegment = ""
		}
	}

	// Add final segment
	if currentSegment != "" {
		segments = append(segments, strings.TrimSpace(currentSegment))
	}

	return segments
}

// createSummaryChunk creates a summary chunk for a group of chunks
func (cs *Service) createSummaryChunk(ctx context.Context, sessionID string, chunks []types.ConversationChunk, baseMetadata *types.ChunkMetadata) *types.ConversationChunk {
	if len(chunks) == 0 {
		return nil
	}

	// Aggregate content for summary
	contentParts := []string{"Summary of recent conversation:"}
	for i := range chunks {
		if chunks[i].Summary != "" {
			contentParts = append(contentParts, fmt.Sprintf("- %s", chunks[i].Summary))
		}
	}

	summaryContent := strings.Join(contentParts, "\n")

	summaryMetadata := *baseMetadata
	summaryMetadata.Tags = append(summaryMetadata.Tags, "summary", "aggregated")

	summaryChunk, err := cs.CreateChunk(ctx, sessionID, summaryContent, &summaryMetadata)
	if err != nil {
		return nil
	}

	summaryChunk.Type = types.ChunkTypeSessionSummary
	return summaryChunk
}

// createSessionSummary creates a final summary for the entire session
func (cs *Service) createSessionSummary(ctx context.Context, sessionID string, chunks []types.ConversationChunk, baseMetadata *types.ChunkMetadata) *types.ConversationChunk {
	// Analyze chunk types
	typeCounts := make(map[types.ChunkType]int)
	for i := range chunks {
		typeCounts[chunks[i].Type]++
	}

	// Build summary content
	contentParts := []string{"Session Summary:"}
	contentParts = append(contentParts, fmt.Sprintf("Total chunks: %d", len(chunks)))

	// Add type breakdown
	for chunkType, count := range typeCounts {
		contentParts = append(contentParts, fmt.Sprintf("- %s: %d", chunkType, count))
	}

	// Add key outcomes
	successCount := 0
	for i := range chunks {
		if chunks[i].Metadata.Outcome == types.OutcomeSuccess {
			successCount++
		}
	}
	contentParts = append(contentParts, fmt.Sprintf("Successful outcomes: %d", successCount))

	// Add tools and files summary
	toolsUsed := make(map[string]bool)
	filesModified := make(map[string]bool)
	for i := range chunks {
		for _, tool := range chunks[i].Metadata.ToolsUsed {
			toolsUsed[tool] = true
		}
		for _, file := range chunks[i].Metadata.FilesModified {
			filesModified[file] = true
		}
	}

	if len(toolsUsed) > 0 {
		tools := []string{}
		for tool := range toolsUsed {
			tools = append(tools, tool)
		}
		contentParts = append(contentParts, fmt.Sprintf("Tools used: %s", strings.Join(tools, ", ")))
	}

	if len(filesModified) > 0 {
		contentParts = append(contentParts, fmt.Sprintf("Files modified: %d", len(filesModified)))
	}

	summaryContent := strings.Join(contentParts, "\n")

	summaryMetadata := *baseMetadata
	summaryMetadata.Tags = append(summaryMetadata.Tags, "session-summary", "final")

	summaryChunk, err := cs.CreateChunk(ctx, sessionID, summaryContent, &summaryMetadata)
	if err != nil {
		return nil
	}

	summaryChunk.Type = types.ChunkTypeSessionSummary
	return summaryChunk
}

// Reset resets the chunking service state
func (cs *Service) Reset() {
	cs.currentContext = &types.ChunkingContext{}
	cs.contextHistory = []types.ChunkingContext{}
	cs.lastChunkTime = time.Now()
}

// getEnvInt gets an integer from environment variable with a default
func getEnvInt(key string, defaultValue int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultValue
}

// Smart Detection Functions

// detectSmartTags identifies specialized tags based on content patterns
func (cs *Service) detectSmartTags(content string) []string {
	var tags []string

	// Detect pattern-based tags
	tags = append(tags, cs.detectPatternBasedTags(content)...)
	
	// Detect content-based tags
	tags = append(tags, cs.detectContentBasedTags(content)...)

	return tags
}

// detectPatternBasedTags detects tags using regex patterns
func (cs *Service) detectPatternBasedTags(content string) []string {
	var tags []string

	// High-impact decision detection
	if cs.detectHighImpactPattern(content) {
		tags = append(tags, "high-impact")
	}

	// Reusable pattern detection
	if cs.detectReusablePattern(content) {
		tags = append(tags, "reusable-pattern")
	}

	// Gotcha detection
	if cs.detectGotchaPattern(content) {
		tags = append(tags, "gotcha")
	}

	// Architectural decision detection
	if cs.detectArchitecturalPattern(content) {
		tags = append(tags, "architecture")
	}

	// Performance-related detection
	if cs.detectPerformancePattern(content) {
		tags = append(tags, "performance")
	}

	return tags
}

// detectContentBasedTags detects tags using string content analysis
func (cs *Service) detectContentBasedTags(content string) []string {
	var tags []string
	contentLower := strings.ToLower(content)

	// Testing and quality
	if cs.detectTestingKeywords(contentLower) {
		tags = append(tags, "testing")
	}

	// Documentation
	if cs.detectDocumentationKeywords(contentLower) {
		tags = append(tags, "documentation")
	}

	// Security
	if cs.detectSecurityKeywords(contentLower) {
		tags = append(tags, "security")
	}

	// DevOps and deployment
	if cs.detectDevOpsKeywords(contentLower) {
		tags = append(tags, "devops")
	}

	// API and integration
	if cs.detectAPIKeywords(contentLower) {
		tags = append(tags, "api")
	}

	return tags
}

// detectHighImpactPattern checks for high-impact decision patterns
func (cs *Service) detectHighImpactPattern(content string) bool {
	for _, pattern := range cs.highImpactPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// detectReusablePattern checks for reusable pattern indicators
func (cs *Service) detectReusablePattern(content string) bool {
	for _, pattern := range cs.reusablePatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// detectGotchaPattern checks for gotcha/pitfall patterns
func (cs *Service) detectGotchaPattern(content string) bool {
	for _, pattern := range cs.gotchaPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// detectArchitecturalPattern checks for architectural decision patterns
func (cs *Service) detectArchitecturalPattern(content string) bool {
	for _, pattern := range cs.architecturalPatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// detectPerformancePattern checks for performance-related patterns
func (cs *Service) detectPerformancePattern(content string) bool {
	for _, pattern := range cs.performancePatterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

// detectTestingKeywords checks for testing-related keywords
func (cs *Service) detectTestingKeywords(contentLower string) bool {
	return strings.Contains(contentLower, "test") || 
		   strings.Contains(contentLower, "testing")
}

// detectDocumentationKeywords checks for documentation-related keywords
func (cs *Service) detectDocumentationKeywords(contentLower string) bool {
	return strings.Contains(contentLower, "document") || 
		   strings.Contains(contentLower, "readme")
}

// detectSecurityKeywords checks for security-related keywords
func (cs *Service) detectSecurityKeywords(contentLower string) bool {
	return strings.Contains(contentLower, "security") || 
		   strings.Contains(contentLower, "auth")
}

// detectDevOpsKeywords checks for DevOps-related keywords
func (cs *Service) detectDevOpsKeywords(contentLower string) bool {
	return strings.Contains(contentLower, "deploy") || 
		   strings.Contains(contentLower, "ci/cd")
}

// detectAPIKeywords checks for API-related keywords
func (cs *Service) detectAPIKeywords(contentLower string) bool {
	return strings.Contains(contentLower, "api") || 
		   strings.Contains(contentLower, "endpoint")
}

// buildExtendedMetadata creates rich metadata for smart analysis
func (cs *Service) buildExtendedMetadata(content string, metadata *types.ChunkMetadata) map[string]interface{} {
	extended := make(map[string]interface{})

	// Calculate impact score (0.0 to 1.0)
	impactScore := cs.calculateImpactScore(content, metadata)
	extended["impact_score"] = impactScore

	// Calculate reusability score (0.0 to 1.0)
	reusabilityScore := cs.calculateReusabilityScore(content)
	extended["reusability_score"] = reusabilityScore

	// Determine significance level
	significanceLevel := cs.determineSignificanceLevel(impactScore, reusabilityScore)
	extended["significance_level"] = significanceLevel

	// Extract technical concepts
	concepts := cs.extractTechnicalConcepts(content)
	if len(concepts) > 0 {
		extended["technical_concepts"] = concepts
	}

	// Analyze complexity indicators
	complexity := cs.analyzeComplexity(content, metadata)
	extended["complexity_indicators"] = complexity

	// Time investment estimation
	timeInvestment := cs.estimateTimeInvestment(content, metadata)
	extended["time_investment_minutes"] = timeInvestment

	// Learning value assessment
	learningValue := cs.assessLearningValue(content, impactScore)
	extended["learning_value"] = learningValue

	return extended
}

// calculateImpactScore determines the impact level of the content
func (cs *Service) calculateImpactScore(content string, metadata *types.ChunkMetadata) float64 {
	score := 0.0

	// Base score from chunk type
	switch metadata.Outcome {
	case types.OutcomeSuccess:
		score += 0.3
	case types.OutcomeFailed:
		score += 0.1
	case types.OutcomeInProgress:
		score += 0.15
	case types.OutcomeAbandoned:
		score += 0.05
	default:
		score += 0.2
	}

	// High-impact pattern bonus
	for _, pattern := range cs.highImpactPatterns {
		if pattern.MatchString(content) {
			score += 0.3
			break
		}
	}

	// Architectural decisions get high impact
	for _, pattern := range cs.architecturalPatterns {
		if pattern.MatchString(content) {
			score += 0.2
			break
		}
	}

	// Tools used indicator
	if len(metadata.ToolsUsed) > 3 {
		score += 0.1
	}

	// Files modified indicator
	if len(metadata.FilesModified) > 2 {
		score += 0.1
	}

	// Content length and depth
	if len(content) > 500 {
		score += 0.1
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// calculateReusabilityScore determines how reusable the content is
func (cs *Service) calculateReusabilityScore(content string) float64 {
	score := 0.0

	// Reusable pattern bonus
	for _, pattern := range cs.reusablePatterns {
		if pattern.MatchString(content) {
			score += 0.4
			break
		}
	}

	// Configuration and setup patterns
	contentLower := strings.ToLower(content)
	if strings.Contains(contentLower, "config") || strings.Contains(contentLower, "setup") {
		score += 0.2
	}

	// Utility and helper patterns
	if strings.Contains(contentLower, "utility") || strings.Contains(contentLower, "helper") {
		score += 0.3
	}

	// Best practices
	if strings.Contains(contentLower, "best practice") || strings.Contains(contentLower, "recommended") {
		score += 0.2
	}

	// Code examples and templates
	if strings.Contains(content, "```") || strings.Contains(contentLower, "template") {
		score += 0.1
	}

	// Documentation value
	if strings.Contains(contentLower, "document") || strings.Contains(contentLower, "guide") {
		score += 0.1
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// determineSignificanceLevel categorizes the overall significance
func (cs *Service) determineSignificanceLevel(impactScore, reusabilityScore float64) string {
	combinedScore := (impactScore + reusabilityScore) / 2

	switch {
	case combinedScore >= 0.8:
		return "critical"
	case combinedScore >= 0.6:
		return SignificanceHigh
	case combinedScore >= 0.4:
		return SignificanceMedium
	default:
		return SignificanceLow
	}
}

// extractTechnicalConcepts identifies key technical concepts
func (cs *Service) extractTechnicalConcepts(content string) []string {
	concepts := []string{}

	// Technology patterns
	techPatterns := map[string]*regexp.Regexp{
		"docker":      regexp.MustCompile(`(?i)\b(docker|container|image|dockerfile)\b`),
		"kubernetes":  regexp.MustCompile(`(?i)\b(kubernetes|k8s|pod|service|deployment)\b`),
		"database":    regexp.MustCompile(`(?i)\b(database|sql|nosql|postgres|mysql|mongodb)\b`),
		"api":         regexp.MustCompile(`(?i)\b(api|rest|graphql|endpoint|http)\b`),
		"security":    regexp.MustCompile(`(?i)\b(security|auth|jwt|oauth|ssl|tls)\b`),
		"testing":     regexp.MustCompile(`(?i)\b(test|testing|unittest|integration|e2e)\b`),
		"performance": regexp.MustCompile(`(?i)\b(performance|optimization|cache|memory|cpu)\b`),
		"monitoring":  regexp.MustCompile(`(?i)\b(monitoring|logging|metrics|observability)\b`),
	}

	for concept, pattern := range techPatterns {
		if pattern.MatchString(content) {
			concepts = append(concepts, concept)
		}
	}

	return concepts
}

// analyzeComplexity provides complexity indicators
func (cs *Service) analyzeComplexity(content string, metadata *types.ChunkMetadata) map[string]interface{} {
	complexity := make(map[string]interface{})

	// Content length indicator
	complexity["content_length"] = len(content)

	// Tools complexity
	complexity["tools_count"] = len(metadata.ToolsUsed)

	// Files complexity
	complexity["files_count"] = len(metadata.FilesModified)

	// Code blocks
	codeBlocks := strings.Count(content, "```")
	complexity["code_blocks"] = codeBlocks

	// Technical terms density
	technicalTerms := 0
	techWords := []string{"function", "class", "method", "api", "database", "server", "client", "config", "deploy", "test"}
	contentLower := strings.ToLower(content)

	for _, term := range techWords {
		if strings.Contains(contentLower, term) {
			technicalTerms++
		}
	}
	complexity["technical_density"] = technicalTerms

	return complexity
}

// estimateTimeInvestment estimates time spent based on content and metadata
func (cs *Service) estimateTimeInvestment(content string, metadata *types.ChunkMetadata) int {
	// Base time estimation
	baseTime := 5 // minutes

	// Content length factor
	if len(content) > 1000 {
		baseTime += 10
	} else if len(content) > 500 {
		baseTime += 5
	}

	// Tools used factor
	baseTime += len(metadata.ToolsUsed) * 2

	// Files modified factor
	baseTime += len(metadata.FilesModified) * 3

	// Complexity factor
	switch metadata.Difficulty {
	case types.DifficultyComplex:
		baseTime += 15
	case types.DifficultyModerate:
		baseTime += 7
	case types.DifficultySimple:
		baseTime += 2
	}

	// Problem resolution factor
	contentLower := strings.ToLower(content)
	if strings.Contains(contentLower, "debug") || strings.Contains(contentLower, "troubleshoot") {
		baseTime += 10
	}

	return baseTime
}

// assessLearningValue determines the educational value of the content
func (cs *Service) assessLearningValue(content string, impactScore float64) string {
	// High impact usually means high learning value
	if impactScore >= 0.7 {
		return "high"
	}

	// Gotcha content is valuable for learning
	for _, pattern := range cs.gotchaPatterns {
		if pattern.MatchString(content) {
			return "high"
		}
	}

	// Best practices are medium learning value
	contentLower := strings.ToLower(content)
	if strings.Contains(contentLower, "best practice") || strings.Contains(contentLower, "lesson") {
		return "medium"
	}

	// Architectural content is valuable
	for _, pattern := range cs.architecturalPatterns {
		if pattern.MatchString(content) {
			return "medium"
		}
	}

	return "low"
}
