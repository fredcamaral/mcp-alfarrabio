// Package threading provides memory threading functionality for grouping related conversations
package threading

import (
	"context"
	"fmt"
	"math"
	"mcp-memory/internal/chains"
	"mcp-memory/internal/relationships"
	"mcp-memory/pkg/types"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ThreadType represents different types of memory threads
type ThreadType string

const (
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
	ThreadStatusActive    ThreadStatus = "active"    // Ongoing work
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
	for i, chunk := range sortedChunks {
		chunkIDs[i] = chunk.ID
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
		// logging.Warn("Failed to create thread chain", "thread_id", thread.ID, "error", err)
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
		if len(sessionChunks) >= 2 { // Minimum thread size
			threadType := tm.inferThreadType(sessionChunks)
			thread, err := tm.CreateThread(ctx, sessionChunks, threadType)
			if err != nil {
				continue // Skip failed threads
			}
			thread.Metadata["detection_method"] = "session_grouping"
			thread.Metadata["source_session"] = sessionID
			threads = append(threads, thread)
		}
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
		if len(featureChunks) >= 3 { // Features usually have more chunks
			thread, err := tm.CreateThread(ctx, featureChunks, ThreadTypeFeature)
			if err != nil {
				continue
			}
			thread.Metadata["detection_method"] = "feature_grouping"
			thread.Metadata["feature_name"] = featureName
			threads = append(threads, thread)
		}
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
	for _, chunk := range chunks {
		sessionSet[chunk.SessionID] = true
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
	for _, chunk := range chunks {
		for _, tag := range chunk.Metadata.Tags {
			tagSet[tag] = true
		}
	}

	// Add inferred tags based on content patterns
	for _, chunk := range chunks {
		content := strings.ToLower(chunk.Content + " " + chunk.Summary)

		// Technology tags
		if strings.Contains(content, "docker") || strings.Contains(content, "container") {
			tagSet["docker"] = true
		}
		if strings.Contains(content, "kubernetes") || strings.Contains(content, "k8s") {
			tagSet["kubernetes"] = true
		}
		if strings.Contains(content, "database") || strings.Contains(content, "sql") {
			tagSet["database"] = true
		}
		if strings.Contains(content, "api") || strings.Contains(content, "endpoint") {
			tagSet["api"] = true
		}

		// Process tags
		if strings.Contains(content, "test") || strings.Contains(content, "testing") {
			tagSet["testing"] = true
		}
		if strings.Contains(content, "deploy") || strings.Contains(content, "deployment") {
			tagSet["deployment"] = true
		}
		if strings.Contains(content, "performance") || strings.Contains(content, "optimization") {
			tagSet["performance"] = true
		}
	}

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
		if duration > 24*time.Hour {
			description += fmt.Sprintf(" over %.1f days", duration.Hours()/24)
		} else if duration > time.Hour {
			description += fmt.Sprintf(" over %.1f hours", duration.Hours())
		} else {
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
	for _, chunk := range chunks {
		repoCount[chunk.Metadata.Repository]++
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
		} else {
			return ThreadStatusPaused
		}
	}
}

func (tm *ThreadManager) calculatePriority(chunks []types.ConversationChunk) int {
	if len(chunks) == 0 {
		return 1
	}

	priority := 1 // Base priority

	// Increase priority based on chunk types
	for _, chunk := range chunks {
		switch chunk.Type {
		case types.ChunkTypeArchitectureDecision:
			priority = maxInt(priority, 4) // High priority for architecture
		case types.ChunkTypeProblem:
			priority = maxInt(priority, 3) // Medium-high for problems
		case types.ChunkTypeSolution:
			priority = maxInt(priority, 3) // Medium-high for solutions
		}
	}

	// Increase priority for recent activity
	lastChunk := chunks[len(chunks)-1]
	if time.Since(lastChunk.Timestamp) < 24*time.Hour {
		priority = minInt(priority+1, 5)
	}

	return priority
}

func (tm *ThreadManager) buildThreadMetadata(chunks []types.ConversationChunk) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Count chunk types
	typeCounts := make(map[string]int)
	for _, chunk := range chunks {
		typeCounts[string(chunk.Type)]++
	}
	metadata["chunk_type_counts"] = typeCounts

	// Count outcomes
	outcomeCounts := make(map[string]int)
	for _, chunk := range chunks {
		outcomeCounts[string(chunk.Metadata.Outcome)]++
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

	_, err := tm.chainBuilder.CreateChain(ctx, chainName, chainDescription, chunks)
	return err
}

func (tm *ThreadManager) calculateLinkStrength(chunk1, chunk2 types.ConversationChunk) float64 {
	strength := 0.5 // Base strength

	// Same session = stronger link
	if chunk1.SessionID == chunk2.SessionID {
		strength += 0.3
	}

	// Close in time = stronger link
	timeDiff := chunk2.Timestamp.Sub(chunk1.Timestamp)
	if timeDiff < time.Hour {
		strength += 0.2
	} else if timeDiff < 24*time.Hour {
		strength += 0.1
	}

	// Related types = stronger link
	if tm.areRelatedTypes(chunk1.Type, chunk2.Type) {
		strength += 0.2
	}

	return math.Min(strength, 1.0)
}

func (tm *ThreadManager) areRelatedTypes(type1, type2 types.ChunkType) bool {
	relatedPairs := map[types.ChunkType][]types.ChunkType{
		types.ChunkTypeProblem:    {types.ChunkTypeSolution, types.ChunkTypeAnalysis},
		types.ChunkTypeSolution:   {types.ChunkTypeProblem, types.ChunkTypeCodeChange},
		types.ChunkTypeCodeChange: {types.ChunkTypeSolution, types.ChunkTypeVerification},
	}

	if related, exists := relatedPairs[type1]; exists {
		for _, relatedType := range related {
			if relatedType == type2 {
				return true
			}
		}
	}

	return false
}

// Thread grouping functions

func (tm *ThreadManager) groupBySession(chunks []types.ConversationChunk) map[string][]types.ConversationChunk {
	groups := make(map[string][]types.ConversationChunk)

	for _, chunk := range chunks {
		groups[chunk.SessionID] = append(groups[chunk.SessionID], chunk)
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
	for _, chunk := range chunks {
		if chunk.Type == types.ChunkTypeProblem {
			group := []types.ConversationChunk{chunk}

			// Find related solutions and analysis
			for _, otherChunk := range chunks {
				if otherChunk.ID != chunk.ID &&
					(otherChunk.Type == types.ChunkTypeSolution || otherChunk.Type == types.ChunkTypeAnalysis) &&
					tm.areChunksRelated(chunk, otherChunk) {
					group = append(group, otherChunk)
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
	for _, chunk := range chunks {
		content := strings.ToLower(chunk.Content + " " + chunk.Summary)

		// Look for feature-related keywords
		featureKeywords := []string{"feature", "implement", "add", "create", "build"}
		for _, keyword := range featureKeywords {
			if strings.Contains(content, keyword) {
				// Extract potential feature name (simplified)
				words := strings.Fields(content)
				for i, word := range words {
					if word == keyword && i+1 < len(words) {
						featureName := words[i+1]
						if len(featureName) > 3 { // Avoid short words
							groups[featureName] = append(groups[featureName], chunk)
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

func (tm *ThreadManager) areChunksRelated(chunk1, chunk2 types.ConversationChunk) bool {
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
	for _, chunk := range chunks {
		typeCounts[chunk.Type]++
	}

	// Infer thread type based on chunk composition
	if typeCounts[types.ChunkTypeProblem] > 0 && typeCounts[types.ChunkTypeSolution] > 0 {
		return ThreadTypeProblemSolving
	}

	if typeCounts[types.ChunkTypeArchitectureDecision] > len(chunks)/2 {
		return ThreadTypeArchitecture
	}

	// Check content for debugging indicators
	for _, chunk := range chunks {
		content := strings.ToLower(chunk.Content + " " + chunk.Summary)
		if strings.Contains(content, "debug") || strings.Contains(content, "error") || strings.Contains(content, "bug") {
			return ThreadTypeDebugging
		}
	}

	// Check for feature development
	for _, chunk := range chunks {
		content := strings.ToLower(chunk.Content + " " + chunk.Summary)
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
	for _, chunk := range chunks {
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
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
	for _, chunk := range chunks {
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
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
		steps = append(steps, "Continue working on the current task")
		steps = append(steps, "Update progress with latest findings")
	case types.OutcomeFailed:
		steps = append(steps, "Analyze the failure and identify root cause")
		steps = append(steps, "Consider alternative approaches")
	case types.OutcomeSuccess:
		steps = append(steps, "Document the successful solution")
		steps = append(steps, "Consider if there are related tasks")
	}

	// Thread-type specific suggestions
	switch threadType {
	case ThreadTypeProblemSolving:
		if lastChunk.Type == types.ChunkTypeProblem {
			steps = append(steps, "Research similar problems in memory")
			steps = append(steps, "Break down the problem into smaller parts")
		}
	case ThreadTypeFeature:
		steps = append(steps, "Plan the feature implementation")
		steps = append(steps, "Consider testing strategy")
	case ThreadTypeDebugging:
		steps = append(steps, "Reproduce the issue consistently")
		steps = append(steps, "Check logs and error messages")
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
