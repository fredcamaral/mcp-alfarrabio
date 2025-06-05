// Package threading provides memory threading functionality for grouping related conversations
package threading

import (
	"context"
	"fmt"
	"math"
	"lerian-mcp-memory/internal/chains"
	"lerian-mcp-memory/internal/relationships"
	"lerian-mcp-memory/pkg/types"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ThreadType represents different types of memory threads
type ThreadType string

const (
	// ThreadTypeConversation represents single conversation threads
	ThreadTypeConversation   ThreadType = "conversation"    // Single conversation thread
	ThreadTypeProblemSolving ThreadType = "problem_solving" // Problem→Investigation→Solution flow
	ThreadTypeFeature        ThreadType = "feature"         // Feature development thread
	ThreadTypeDebugging      ThreadType = "debugging"       // Debugging session thread
	ThreadTypeArchitecture   ThreadType = "architecture"    // Architecture decision thread
	ThreadTypeWorkflow       ThreadType = "workflow"        // General workflow thread
)

// ThreadStatus represents the current status of a thread
type ThreadStatus string

const (
	// ThreadStatusActive represents ongoing work threads
	ThreadStatusActive ThreadStatus = "active" // Ongoing work
	// ThreadStatusComplete represents successfully completed threads
	ThreadStatusComplete  ThreadStatus = "complete"  // Successfully completed
	ThreadStatusPaused    ThreadStatus = "paused"    // Temporarily paused
	ThreadStatusAbandoned ThreadStatus = "abandoned" // Abandoned/cancelled
	ThreadStatusBlocked   ThreadStatus = "blocked"   // Blocked on external dependency
)

// MemoryThread represents a coherent thread of related memory chunks
type MemoryThread struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Type        ThreadType   `json:"type"`
	Status      ThreadStatus `json:"status"`
	Repository  string       `json:"repository"`

	// Chunk organization
	ChunkIDs   []string `json:"chunk_ids"`
	FirstChunk string   `json:"first_chunk"`
	LastChunk  string   `json:"last_chunk"`

	// Temporal information
	StartTime  time.Time  `json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	LastUpdate time.Time  `json:"last_update"`

	// Thread metadata
	SessionIDs []string               `json:"session_ids"`
	Tags       []string               `json:"tags"`
	Priority   int                    `json:"priority"` // 1-5, 5 = highest
	Metadata   map[string]interface{} `json:"metadata,omitempty"`

	// Relationships
	ParentThread   *string  `json:"parent_thread,omitempty"`
	ChildThreads   []string `json:"child_threads,omitempty"`
	RelatedThreads []string `json:"related_threads,omitempty"`
}

// ThreadSummary provides a high-level overview of a thread
type ThreadSummary struct {
	Thread      *MemoryThread `json:"thread"`
	ChunkCount  int           `json:"chunk_count"`
	Duration    time.Duration `json:"duration"`
	Progress    float64       `json:"progress"`     // 0.0 to 1.0
	HealthScore float64       `json:"health_score"` // 0.0 to 1.0
	NextSteps   []string      `json:"next_steps,omitempty"`
}

// ThreadManager handles memory thread operations
type ThreadManager struct {
	chainBuilder    *chains.ChainBuilder
	relationshipMgr *relationships.Manager
	store           ThreadStore
}

// ThreadStore interface for persisting threads
type ThreadStore interface {
	StoreThread(ctx context.Context, thread *MemoryThread) error
	GetThread(ctx context.Context, threadID string) (*MemoryThread, error)
	GetThreadsByRepository(ctx context.Context, repository string) ([]*MemoryThread, error)
	GetActiveThreads(ctx context.Context, repository string) ([]*MemoryThread, error)
	UpdateThreadStatus(ctx context.Context, threadID string, status ThreadStatus) error
	DeleteThread(ctx context.Context, threadID string) error
	ListThreads(ctx context.Context, filters ThreadFilters) ([]*MemoryThread, error)
}

// ThreadFilters for querying threads
type ThreadFilters struct {
	Repository *string       `json:"repository,omitempty"`
	Type       *ThreadType   `json:"type,omitempty"`
	Status     *ThreadStatus `json:"status,omitempty"`
	Tags       []string      `json:"tags,omitempty"`
	SessionID  *string       `json:"session_id,omitempty"`
	Since      *time.Time    `json:"since,omitempty"`
	Until      *time.Time    `json:"until,omitempty"`
}

// NewThreadManager creates a new thread manager
func NewThreadManager(chainBuilder *chains.ChainBuilder, relationshipMgr *relationships.Manager, store ThreadStore) *ThreadManager {
	return &ThreadManager{
		chainBuilder:    chainBuilder,
		relationshipMgr: relationshipMgr,
		store:           store,
	}
}

// CreateThread creates a new memory thread from related chunks
func (tm *ThreadManager) CreateThread(ctx context.Context, chunks []types.ConversationChunk, threadType ThreadType) (*MemoryThread, error) {
	if len(chunks) == 0 {
		return nil, fmt.Errorf("cannot create thread from empty chunks")
	}

	// Sort chunks by timestamp
	sortedChunks := make([]types.ConversationChunk, len(chunks))
	copy(sortedChunks, chunks)
	sort.Slice(sortedChunks, func(i, j int) bool {
		return sortedChunks[i].Timestamp.Before(sortedChunks[j].Timestamp)
	})

	// Extract basic information
	firstChunk := sortedChunks[0]
	lastChunk := sortedChunks[len(sortedChunks)-1]

	// Collect metadata
	sessionIDs := tm.extractSessionIDs(sortedChunks)
	tags := tm.extractThreadTags(sortedChunks)
	chunkIDs := make([]string, len(sortedChunks))
	for i := range sortedChunks {
		chunkIDs[i] = sortedChunks[i].ID
	}

	// Generate thread details
	title := tm.generateThreadTitle(sortedChunks, threadType)
	description := tm.generateThreadDescription(sortedChunks, threadType)
	repository := tm.determineRepository(sortedChunks)

	thread := &MemoryThread{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		Type:        threadType,
		Status:      tm.determineInitialStatus(sortedChunks),
		Repository:  repository,

		ChunkIDs:   chunkIDs,
		FirstChunk: firstChunk.ID,
		LastChunk:  lastChunk.ID,

		StartTime:  firstChunk.Timestamp,
		LastUpdate: lastChunk.Timestamp,

		SessionIDs: sessionIDs,
		Tags:       tags,
		Priority:   tm.calculatePriority(sortedChunks),
		Metadata:   tm.buildThreadMetadata(sortedChunks),
	}

	// Set end time if thread appears complete
	if thread.Status == ThreadStatusComplete {
		thread.EndTime = &lastChunk.Timestamp
	}

	// Store the thread
	if err := tm.store.StoreThread(ctx, thread); err != nil {
		return nil, fmt.Errorf("failed to store thread: %w", err)
	}

	// Create chain relationships
	if err := tm.createThreadChain(ctx, thread, sortedChunks); err != nil {
		// Log error but don't fail thread creation
		// TODO: Enable logging when logger is available
		_ = err // Acknowledge error for linter
	}

	return thread, nil
}

// DetectThreads automatically detects and creates threads from existing chunks
func (tm *ThreadManager) DetectThreads(ctx context.Context, chunks []types.ConversationChunk) ([]*MemoryThread, error) {
	if len(chunks) == 0 {
		return []*MemoryThread{}, nil
	}

	threads := []*MemoryThread{}

	// Group chunks by various criteria for thread detection
	sessionGroups := tm.groupBySession(chunks)
	problemSolutionGroups := tm.groupByProblemSolution(chunks)
	featureGroups := tm.groupByFeature(chunks)

	// Create threads from session groups
	for sessionID, sessionChunks := range sessionGroups {
		if len(sessionChunks) < 2 { // Minimum thread size
			continue
		}
		threadType := tm.inferThreadType(sessionChunks)
		thread, err := tm.CreateThread(ctx, sessionChunks, threadType)
		if err != nil {
			continue // Skip failed threads
		}
		thread.Metadata["detection_method"] = "session_grouping"
		thread.Metadata["source_session"] = sessionID
		threads = append(threads, thread)
	}

	// Create problem-solution threads that span sessions
	for _, problemGroup := range problemSolutionGroups {
		if len(problemGroup) >= 2 {
			thread, err := tm.CreateThread(ctx, problemGroup, ThreadTypeProblemSolving)
			if err != nil {
				continue
			}
			thread.Metadata["detection_method"] = "problem_solution_grouping"
			threads = append(threads, thread)
		}
	}

	// Create feature development threads
	for featureName, featureChunks := range featureGroups {
		if len(featureChunks) < 3 { // Features usually have more chunks
			continue
		}
		thread, err := tm.CreateThread(ctx, featureChunks, ThreadTypeFeature)
		if err != nil {
			continue
		}
		thread.Metadata["detection_method"] = "feature_grouping"
		thread.Metadata["feature_name"] = featureName
		threads = append(threads, thread)
	}

	return threads, nil
}

// GetThreadSummary provides a comprehensive summary of a thread
func (tm *ThreadManager) GetThreadSummary(ctx context.Context, threadID string, chunks []types.ConversationChunk) (*ThreadSummary, error) {
	thread, err := tm.store.GetThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	// Calculate metrics
	duration := time.Duration(0)
	if thread.EndTime != nil {
		duration = thread.EndTime.Sub(thread.StartTime)
	} else {
		duration = thread.LastUpdate.Sub(thread.StartTime)
	}

	progress := tm.calculateProgress(chunks)
	healthScore := tm.calculateHealthScore(chunks)
	nextSteps := tm.generateNextSteps(chunks, thread.Type)

	return &ThreadSummary{
		Thread:      thread,
		ChunkCount:  len(chunks),
		Duration:    duration,
		Progress:    progress,
		HealthScore: healthScore,
		NextSteps:   nextSteps,
	}, nil
}

// Helper functions for thread creation and management

func (tm *ThreadManager) extractSessionIDs(chunks []types.ConversationChunk) []string {
	sessionSet := make(map[string]bool)
	for i := range chunks {
		sessionSet[chunks[i].SessionID] = true
	}

	sessions := make([]string, 0, len(sessionSet))
	for session := range sessionSet {
		sessions = append(sessions, session)
	}
	return sessions
}

func (tm *ThreadManager) extractThreadTags(chunks []types.ConversationChunk) []string {
	tagSet := make(map[string]bool)

	// Extract from chunk metadata tags
	tm.extractMetadataTags(chunks, tagSet)

	// Add inferred tags based on content patterns
	tm.extractInferredTags(chunks, tagSet)

	return tm.convertTagSetToSlice(tagSet)
}

// extractMetadataTags extracts explicit tags from chunk metadata
func (tm *ThreadManager) extractMetadataTags(chunks []types.ConversationChunk, tagSet map[string]bool) {
	for i := range chunks {
		for _, tag := range chunks[i].Metadata.Tags {
			tagSet[tag] = true
		}
	}
}

// extractInferredTags infers tags from content patterns
func (tm *ThreadManager) extractInferredTags(chunks []types.ConversationChunk, tagSet map[string]bool) {
	for i := range chunks {
		content := strings.ToLower(chunks[i].Content + " " + chunks[i].Summary)
		tm.addTechnologyTags(content, tagSet)
		tm.addProcessTags(content, tagSet)
	}
}

// addTechnologyTags adds technology-related tags based on content
func (tm *ThreadManager) addTechnologyTags(content string, tagSet map[string]bool) {
	techPatterns := map[string][]string{
		"docker":     {"docker", "container"},
		"kubernetes": {"kubernetes", "k8s"},
		"database":   {"database", "sql"},
		"api":        {"api", "endpoint"},
	}

	for tag, patterns := range techPatterns {
		for _, pattern := range patterns {
			if strings.Contains(content, pattern) {
				tagSet[tag] = true
				break
			}
		}
	}
}

// addProcessTags adds process-related tags based on content
func (tm *ThreadManager) addProcessTags(content string, tagSet map[string]bool) {
	processPatterns := map[string][]string{
		"testing":     {"test", "testing"},
		"deployment":  {"deploy", "deployment"},
		"performance": {"performance", "optimization"},
	}

	for tag, patterns := range processPatterns {
		for _, pattern := range patterns {
			if strings.Contains(content, pattern) {
				tagSet[tag] = true
				break
			}
		}
	}
}

// convertTagSetToSlice converts tag set to slice
func (tm *ThreadManager) convertTagSetToSlice(tagSet map[string]bool) []string {
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	return tags
}

func (tm *ThreadManager) generateThreadTitle(chunks []types.ConversationChunk, threadType ThreadType) string {
	if len(chunks) == 0 {
		return fmt.Sprintf("Empty %s Thread", threadType)
	}

	// Use first chunk's summary as base, with type prefix
	firstSummary := chunks[0].Summary
	if len(firstSummary) > 50 {
		firstSummary = firstSummary[:47] + "..."
	}

	switch threadType {
	case ThreadTypeProblemSolving:
		return fmt.Sprintf("Problem: %s", firstSummary)
	case ThreadTypeFeature:
		return fmt.Sprintf("Feature: %s", firstSummary)
	case ThreadTypeDebugging:
		return fmt.Sprintf("Debug: %s", firstSummary)
	case ThreadTypeArchitecture:
		return fmt.Sprintf("Architecture: %s", firstSummary)
	case ThreadTypeConversation:
		return fmt.Sprintf("Discussion: %s", firstSummary)
	case ThreadTypeWorkflow:
		return fmt.Sprintf("Workflow: %s", firstSummary)
	default:
		return firstSummary
	}
}

func (tm *ThreadManager) generateThreadDescription(chunks []types.ConversationChunk, threadType ThreadType) string {
	if len(chunks) == 0 {
		return "Empty thread"
	}

	description := fmt.Sprintf("%s thread with %d memory chunks", threadType, len(chunks))

	// Add session info
	sessions := tm.extractSessionIDs(chunks)
	if len(sessions) == 1 {
		description += fmt.Sprintf(" from session %s", sessions[0])
	} else {
		description += fmt.Sprintf(" spanning %d sessions", len(sessions))
	}

	// Add time span
	if len(chunks) > 1 {
		duration := chunks[len(chunks)-1].Timestamp.Sub(chunks[0].Timestamp)
		switch {
		case duration > 24*time.Hour:
			description += fmt.Sprintf(" over %.1f days", duration.Hours()/24)
		case duration > time.Hour:
			description += fmt.Sprintf(" over %.1f hours", duration.Hours())
		default:
			description += fmt.Sprintf(" over %d minutes", int(duration.Minutes()))
		}
	}

	return description
}

func (tm *ThreadManager) determineRepository(chunks []types.ConversationChunk) string {
	if len(chunks) == 0 {
		return ""
	}

	// Use the most common repository
	repoCount := make(map[string]int)
	for i := range chunks {
		repoCount[chunks[i].Metadata.Repository]++
	}

	maxCount := 0
	repository := ""
	for repo, count := range repoCount {
		if count > maxCount {
			maxCount = count
			repository = repo
		}
	}

	return repository
}

func (tm *ThreadManager) determineInitialStatus(chunks []types.ConversationChunk) ThreadStatus {
	if len(chunks) == 0 {
		return ThreadStatusActive
	}

	lastChunk := chunks[len(chunks)-1]

	// Check outcomes of recent chunks
	switch lastChunk.Metadata.Outcome {
	case types.OutcomeSuccess:
		return ThreadStatusComplete
	case types.OutcomeFailed:
		return ThreadStatusBlocked
	case types.OutcomeInProgress:
		return ThreadStatusActive
	case types.OutcomeAbandoned:
		return ThreadStatusAbandoned
	default:
		// If last activity was recent, consider active
		if time.Since(lastChunk.Timestamp) < 24*time.Hour {
			return ThreadStatusActive
		}
		return ThreadStatusPaused
	}
}

func (tm *ThreadManager) calculatePriority(chunks []types.ConversationChunk) int {
	if len(chunks) == 0 {
		return 1
	}

	priority := tm.calculateTypePriority(chunks)
	priority = tm.adjustForRecentActivity(chunks, priority)
	return priority
}

// calculateTypePriority calculates priority based on chunk types
func (tm *ThreadManager) calculateTypePriority(chunks []types.ConversationChunk) int {
	priority := 1 // Base priority

	for i := range chunks {
		chunkPriority := tm.getChunkTypePriority(&chunks[i])
		priority = maxInt(priority, chunkPriority)
	}

	return priority
}

// getChunkTypePriority returns priority for a specific chunk type
func (tm *ThreadManager) getChunkTypePriority(chunk *types.ConversationChunk) int {
	switch chunk.Type {
	case types.ChunkTypeArchitectureDecision:
		return 4 // High priority for architecture
	case types.ChunkTypeProblem, types.ChunkTypeSolution, types.ChunkTypeCodeChange:
		return 3 // Medium-high for critical chunks
	case types.ChunkTypeDiscussion, types.ChunkTypeSessionSummary, types.ChunkTypeAnalysis, types.ChunkTypeVerification:
		return 2 // Medium for analysis chunks
	case types.ChunkTypeQuestion:
		return 1 // Base priority for questions
	case types.ChunkTypeTask:
		return tm.calculateTaskPriority(chunk)
	case types.ChunkTypeTaskUpdate:
		return 3 // Updates are important
	case types.ChunkTypeTaskProgress:
		return 2 // Progress tracking is medium priority
	default:
		return 1 // Default priority
	}
}

// calculateTaskPriority calculates priority for task chunks
func (tm *ThreadManager) calculateTaskPriority(chunk *types.ConversationChunk) int {
	taskPriority := 3 // Default medium-high

	// Adjust based on task priority metadata
	if chunk.Metadata.TaskPriority != nil {
		switch *chunk.Metadata.TaskPriority {
		case "high":
			taskPriority = 4
		case "medium":
			taskPriority = 3
		case "low":
			taskPriority = 2
		}
	}

	// Boost priority for completed tasks
	if chunk.Metadata.TaskStatus != nil && *chunk.Metadata.TaskStatus == "completed" {
		taskPriority = maxInt(taskPriority, 4)
	}

	return taskPriority
}

// adjustForRecentActivity adjusts priority based on recent activity
func (tm *ThreadManager) adjustForRecentActivity(chunks []types.ConversationChunk, priority int) int {
	lastChunk := chunks[len(chunks)-1]
	if time.Since(lastChunk.Timestamp) < 24*time.Hour {
		return minInt(priority+1, 5)
	}
	return priority
}

func (tm *ThreadManager) buildThreadMetadata(chunks []types.ConversationChunk) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Count chunk types
	typeCounts := make(map[string]int)
	for i := range chunks {
		typeCounts[string(chunks[i].Type)]++
	}
	metadata["chunk_type_counts"] = typeCounts

	// Count outcomes
	outcomeCounts := make(map[string]int)
	for i := range chunks {
		outcomeCounts[string(chunks[i].Metadata.Outcome)]++
	}
	metadata["outcome_counts"] = outcomeCounts

	// Add timing information
	if len(chunks) > 1 {
		duration := chunks[len(chunks)-1].Timestamp.Sub(chunks[0].Timestamp)
		metadata["duration_minutes"] = int(duration.Minutes())
	}

	return metadata
}

func (tm *ThreadManager) createThreadChain(ctx context.Context, thread *MemoryThread, chunks []types.ConversationChunk) error {
	// Create a chain for this thread using ChainBuilder
	chainName := "Thread: " + thread.Title
	chainDescription := "Memory chain for thread: " + thread.Description

	// Convert chunks to pointers for the chain builder
	chunkPointers := make([]*types.ConversationChunk, len(chunks))
	for i := range chunks {
		chunkPointers[i] = &chunks[i]
	}

	_, err := tm.chainBuilder.CreateChain(ctx, chainName, chainDescription, chunkPointers)
	return err
}

// Thread grouping functions

func (tm *ThreadManager) groupBySession(chunks []types.ConversationChunk) map[string][]types.ConversationChunk {
	groups := make(map[string][]types.ConversationChunk)

	for i := range chunks {
		groups[chunks[i].SessionID] = append(groups[chunks[i].SessionID], chunks[i])
	}

	// Sort each group by timestamp
	for sessionID := range groups {
		sort.Slice(groups[sessionID], func(i, j int) bool {
			return groups[sessionID][i].Timestamp.Before(groups[sessionID][j].Timestamp)
		})
	}

	return groups
}

func (tm *ThreadManager) groupByProblemSolution(chunks []types.ConversationChunk) [][]types.ConversationChunk {
	groups := [][]types.ConversationChunk{}

	// Find problem chunks and their related solutions
	for i := range chunks {
		if chunks[i].Type == types.ChunkTypeProblem {
			group := []types.ConversationChunk{chunks[i]}

			// Find related solutions and analysis
			for j := range chunks {
				if chunks[j].ID != chunks[i].ID &&
					(chunks[j].Type == types.ChunkTypeSolution || chunks[j].Type == types.ChunkTypeAnalysis) &&
					tm.areChunksRelated(&chunks[i], &chunks[j]) {
					group = append(group, chunks[j])
				}
			}

			if len(group) > 1 {
				groups = append(groups, group)
			}
		}
	}

	return groups
}

func (tm *ThreadManager) groupByFeature(chunks []types.ConversationChunk) map[string][]types.ConversationChunk {
	groups := make(map[string][]types.ConversationChunk)

	// Simple feature detection based on keywords
	for i := range chunks {
		content := strings.ToLower(chunks[i].Content + " " + chunks[i].Summary)

		// Look for feature-related keywords
		featureKeywords := []string{"feature", "implement", "add", "create", "build"}
		for _, keyword := range featureKeywords {
			if strings.Contains(content, keyword) {
				// Extract potential feature name (simplified)
				words := strings.Fields(content)
				for j, word := range words {
					if word == keyword && j+1 < len(words) {
						featureName := words[j+1]
						if len(featureName) > 3 { // Avoid short words
							groups[featureName] = append(groups[featureName], chunks[i])
							break
						}
					}
				}
				break
			}
		}
	}

	return groups
}

func (tm *ThreadManager) areChunksRelated(chunk1, chunk2 *types.ConversationChunk) bool {
	// Check if chunks are related based on various criteria

	// Same session
	if chunk1.SessionID == chunk2.SessionID {
		return true
	}

	// Close in time (within 24 hours)
	timeDiff := chunk2.Timestamp.Sub(chunk1.Timestamp)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff < 24*time.Hour {
		return true
	}

	// Similar content (simple keyword matching)
	content1 := strings.ToLower(chunk1.Summary)
	content2 := strings.ToLower(chunk2.Summary)

	words1 := strings.Fields(content1)
	words2 := strings.Fields(content2)

	commonWords := 0
	for _, word1 := range words1 {
		if len(word1) > 3 { // Only count significant words
			for _, word2 := range words2 {
				if word1 == word2 {
					commonWords++
					break
				}
			}
		}
	}

	// Consider related if they share significant words
	minWords := len(words1)
	if len(words2) < minWords {
		minWords = len(words2)
	}

	if minWords > 0 && float64(commonWords)/float64(minWords) > 0.3 {
		return true
	}

	return false
}

func (tm *ThreadManager) inferThreadType(chunks []types.ConversationChunk) ThreadType {
	if len(chunks) == 0 {
		return ThreadTypeConversation
	}

	// Count different chunk types
	typeCounts := make(map[types.ChunkType]int)
	for i := range chunks {
		typeCounts[chunks[i].Type]++
	}

	// Infer thread type based on chunk composition
	if typeCounts[types.ChunkTypeProblem] > 0 && typeCounts[types.ChunkTypeSolution] > 0 {
		return ThreadTypeProblemSolving
	}

	if typeCounts[types.ChunkTypeArchitectureDecision] > len(chunks)/2 {
		return ThreadTypeArchitecture
	}

	// Check content for debugging indicators
	for i := range chunks {
		content := strings.ToLower(chunks[i].Content + " " + chunks[i].Summary)
		if strings.Contains(content, "debug") || strings.Contains(content, "error") || strings.Contains(content, "bug") {
			return ThreadTypeDebugging
		}
	}

	// Check for feature development
	for i := range chunks {
		content := strings.ToLower(chunks[i].Content + " " + chunks[i].Summary)
		if strings.Contains(content, "feature") || strings.Contains(content, "implement") {
			return ThreadTypeFeature
		}
	}

	return ThreadTypeConversation
}

// Thread analysis functions

func (tm *ThreadManager) calculateProgress(chunks []types.ConversationChunk) float64 {
	if len(chunks) == 0 {
		return 0.0
	}

	completedChunks := 0
	for i := range chunks {
		if chunks[i].Metadata.Outcome == types.OutcomeSuccess {
			completedChunks++
		}
	}

	return float64(completedChunks) / float64(len(chunks))
}

func (tm *ThreadManager) calculateHealthScore(chunks []types.ConversationChunk) float64 {
	if len(chunks) == 0 {
		return 0.0
	}

	score := 0.5 // Base score

	// Factor in success rate
	successCount := 0
	for i := range chunks {
		if chunks[i].Metadata.Outcome == types.OutcomeSuccess {
			successCount++
		}
	}
	successRate := float64(successCount) / float64(len(chunks))
	score = (score * 0.3) + (successRate * 0.7)

	// Bonus for recent activity
	lastChunk := chunks[len(chunks)-1]
	if time.Since(lastChunk.Timestamp) < 24*time.Hour {
		score = math.Min(score+0.1, 1.0)
	}

	// Penalty for old threads without resolution
	if time.Since(lastChunk.Timestamp) > 7*24*time.Hour {
		score = math.Max(score-0.2, 0.0)
	}

	return score
}

func (tm *ThreadManager) generateNextSteps(chunks []types.ConversationChunk, threadType ThreadType) []string {
	if len(chunks) == 0 {
		return []string{}
	}

	steps := []string{}
	lastChunk := chunks[len(chunks)-1]

	// Generic next steps based on last outcome
	switch lastChunk.Metadata.Outcome {
	case types.OutcomeInProgress:
		steps = append(steps, "Continue working on the current task", "Update progress with latest findings")
	case types.OutcomeFailed:
		steps = append(steps, "Analyze the failure and identify root cause", "Consider alternative approaches")
	case types.OutcomeAbandoned:
		steps = append(steps, "Reconsider the abandoned approach", "Evaluate if circumstances have changed")
	case types.OutcomeSuccess:
		steps = append(steps, "Document the successful solution", "Consider if there are related tasks")
	}

	// Thread-type specific suggestions
	switch threadType {
	case ThreadTypeConversation:
		steps = append(steps, "Continue the conversation flow", "Address any open questions")
	case ThreadTypeProblemSolving:
		if lastChunk.Type == types.ChunkTypeProblem {
			steps = append(steps, "Research similar problems in memory", "Break down the problem into smaller parts")
		}
	case ThreadTypeFeature:
		steps = append(steps, "Plan the feature implementation", "Consider testing strategy")
	case ThreadTypeDebugging:
		steps = append(steps, "Reproduce the issue consistently", "Check logs and error messages")
	case ThreadTypeArchitecture:
		steps = append(steps, "Document architectural decisions", "Consider long-term implications")
	case ThreadTypeWorkflow:
		steps = append(steps, "Follow established workflow patterns", "Update process documentation")
	}

	return steps
}

// Utility functions
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
