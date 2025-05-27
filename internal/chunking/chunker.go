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

// ChunkingService handles the intelligent chunking of conversations
type ChunkingService struct {
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
}

// NewChunkingService creates a new chunking service
func NewChunkingService(cfg *config.ChunkingConfig, embeddingService embeddings.EmbeddingService) *ChunkingService {
	cs := &ChunkingService{
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
func (cs *ChunkingService) initializePatterns() {
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
}

// ShouldCreateChunk determines if a new chunk should be created based on context
func (cs *ChunkingService) ShouldCreateChunk(context types.ChunkingContext) bool {
	// Update current context
	cs.currentContext = &context

	// Todo completion trigger (highest priority)
	if cs.config.TodoCompletionTrigger && context.HasCompletedTodos() {
		return true
	}

	// Significant file changes
	if len(context.FileModifications) >= cs.config.FileChangeThreshold {
		return true
	}

	// Time-based chunking
	if context.TimeElapsed >= cs.config.TimeThresholdMinutes {
		return true
	}

	// Problem resolution cycle complete
	if context.ConversationFlow == types.FlowVerification && context.TimeElapsed > 5 {
		return true
	}

	// Context switch detected
	if cs.hasContextSwitch(context) {
		return true
	}

	// Content volume threshold
	totalContent := 0
	for _, tool := range context.ToolsUsed {
		totalContent += len(tool) * 10 // Rough estimation
	}

	return totalContent > cs.config.MaxContentLength
}

// CreateChunk creates a conversation chunk from the current context
func (cs *ChunkingService) CreateChunk(ctx context.Context, sessionID, content string, metadata types.ChunkMetadata) (*types.ConversationChunk, error) {
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
	cs.contextHistory = append(cs.contextHistory, *cs.currentContext)

	// Keep only last 10 contexts for analysis
	if len(cs.contextHistory) > 10 {
		cs.contextHistory = cs.contextHistory[1:]
	}

	return chunk, nil
}

// analyzeContentType determines the type of chunk based on content analysis
func (cs *ChunkingService) analyzeContentType(content string) types.ChunkType {
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
func (cs *ChunkingService) enrichMetadata(metadata types.ChunkMetadata, content string) types.ChunkMetadata {
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

	// Set outcome based on content analysis
	if metadata.Outcome == "" {
		metadata.Outcome = cs.assessOutcome(content)
	}

	return metadata
}

// extractTags extracts relevant tags from content
func (cs *ChunkingService) extractTags(content string) []string {
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
func (cs *ChunkingService) assessDifficulty(content string) types.Difficulty {
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
func (cs *ChunkingService) assessOutcome(content string) types.Outcome {
	content = strings.ToLower(content)

	successIndicators := []string{"completed", "fixed", "solved", "working", "success", "done"}
	failureIndicators := []string{"failed", "error", "broken", "not working", "issue"}
	progressIndicators := []string{"in progress", "working on", "implementing", "developing"}

	for _, indicator := range successIndicators {
		if strings.Contains(content, indicator) {
			return types.OutcomeSuccess
		}
	}

	for _, indicator := range failureIndicators {
		if strings.Contains(content, indicator) {
			return types.OutcomeFailed
		}
	}

	for _, indicator := range progressIndicators {
		if strings.Contains(content, indicator) {
			return types.OutcomeInProgress
		}
	}

	return types.OutcomeInProgress // Default assumption
}

// generateSummary creates an AI-powered summary of the content
func (cs *ChunkingService) generateSummary(_ context.Context, content string, _ types.ChunkType) string {
	// For now, implement a simple extractive summary
	// In a full implementation, this would use an LLM for abstractive summarization
	return cs.generateSimpleSummary(content)
}

// generateSimpleSummary creates a simple extractive summary
func (cs *ChunkingService) generateSimpleSummary(content string) string {
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
func (cs *ChunkingService) prepareContentForEmbedding(chunk *types.ConversationChunk) string {
	parts := []string{}

	// Include chunk type for context
	parts = append(parts, fmt.Sprintf("Type: %s", chunk.Type))

	// Include summary if available
	if chunk.Summary != "" {
		parts = append(parts, fmt.Sprintf("Summary: %s", chunk.Summary))
	}

	// Include main content
	parts = append(parts, fmt.Sprintf("Content: %s", chunk.Content))

	// Include relevant metadata
	if chunk.Metadata.Repository != "" {
		parts = append(parts, fmt.Sprintf("Repository: %s", chunk.Metadata.Repository))
	}

	if len(chunk.Metadata.Tags) > 0 {
		parts = append(parts, fmt.Sprintf("Tags: %s", strings.Join(chunk.Metadata.Tags, ", ")))
	}

	combined := strings.Join(parts, " ")

	// Truncate if too long for embedding model
	maxLength := getEnvInt("MCP_MEMORY_MAX_EMBEDDING_CONTENT_LENGTH", 8000) // Conservative limit for most embedding models
	if len(combined) > maxLength {
		return combined[:maxLength]
	}

	return combined
}

// hasContextSwitch detects if there has been a significant context switch
func (cs *ChunkingService) hasContextSwitch(context types.ChunkingContext) bool {
	if len(cs.contextHistory) == 0 {
		return false
	}

	lastContext := cs.contextHistory[len(cs.contextHistory)-1]

	// Check for conversation flow changes
	if lastContext.ConversationFlow != context.ConversationFlow {
		return true
	}

	// Check for significant tool change
	if len(context.ToolsUsed) > 0 && len(lastContext.ToolsUsed) > 0 {
		commonTools := 0
		for _, tool := range context.ToolsUsed {
			for _, lastTool := range lastContext.ToolsUsed {
				if tool == lastTool {
					commonTools++
					break
				}
			}
		}

		// If less than 30% tools in common, it's a context switch
		if float64(commonTools)/float64(len(context.ToolsUsed)) < 0.3 {
			return true
		}
	}

	// Check for file context changes
	if len(context.FileModifications) > 0 && len(lastContext.FileModifications) > 0 {
		commonFiles := 0
		for _, file := range context.FileModifications {
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
func (cs *ChunkingService) GetCurrentContext() *types.ChunkingContext {
	return cs.currentContext
}

// UpdateContext updates the current context with new information
func (cs *ChunkingService) UpdateContext(updates map[string]interface{}) {
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
func (cs *ChunkingService) ProcessConversation(ctx context.Context, sessionID string, conversation string, baseMetadata types.ChunkMetadata) ([]types.ConversationChunk, error) {
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
func (cs *ChunkingService) splitConversation(conversation string) []string {
	segments := []string{}
	currentSegment := ""
	lines := strings.Split(conversation, "\n")
	
	// Patterns that indicate segment boundaries
	boundaryPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^(Human|Assistant|User|AI|Claude):`),
		regexp.MustCompile(`^###|^---|^===`), // Section markers
		regexp.MustCompile(`^\d+\.\s`),        // Numbered lists
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
func (cs *ChunkingService) createSummaryChunk(ctx context.Context, sessionID string, chunks []types.ConversationChunk, baseMetadata types.ChunkMetadata) *types.ConversationChunk {
	if len(chunks) == 0 {
		return nil
	}
	
	// Aggregate content for summary
	contentParts := []string{"Summary of recent conversation:"}
	for _, chunk := range chunks {
		if chunk.Summary != "" {
			contentParts = append(contentParts, fmt.Sprintf("- %s", chunk.Summary))
		}
	}
	
	summaryContent := strings.Join(contentParts, "\n")
	
	summaryMetadata := baseMetadata
	summaryMetadata.Tags = append(summaryMetadata.Tags, "summary", "aggregated")
	
	summaryChunk, err := cs.CreateChunk(ctx, sessionID, summaryContent, summaryMetadata)
	if err != nil {
		return nil
	}
	
	summaryChunk.Type = types.ChunkTypeSessionSummary
	return summaryChunk
}

// createSessionSummary creates a final summary for the entire session
func (cs *ChunkingService) createSessionSummary(ctx context.Context, sessionID string, chunks []types.ConversationChunk, baseMetadata types.ChunkMetadata) *types.ConversationChunk {
	// Analyze chunk types
	typeCounts := make(map[types.ChunkType]int)
	for _, chunk := range chunks {
		typeCounts[chunk.Type]++
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
	for _, chunk := range chunks {
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
			successCount++
		}
	}
	contentParts = append(contentParts, fmt.Sprintf("Successful outcomes: %d", successCount))
	
	// Add tools and files summary
	toolsUsed := make(map[string]bool)
	filesModified := make(map[string]bool)
	for _, chunk := range chunks {
		for _, tool := range chunk.Metadata.ToolsUsed {
			toolsUsed[tool] = true
		}
		for _, file := range chunk.Metadata.FilesModified {
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
	
	summaryMetadata := baseMetadata
	summaryMetadata.Tags = append(summaryMetadata.Tags, "session-summary", "final")
	
	summaryChunk, err := cs.CreateChunk(ctx, sessionID, summaryContent, summaryMetadata)
	if err != nil {
		return nil
	}
	
	summaryChunk.Type = types.ChunkTypeSessionSummary
	return summaryChunk
}

// Reset resets the chunking service state
func (cs *ChunkingService) Reset() {
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
