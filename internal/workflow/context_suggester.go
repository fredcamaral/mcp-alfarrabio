// Package workflow provides proactive context suggestions for Claude
package workflow

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"mcp-memory/pkg/types"
)

// ContextSuggestion represents a proactive suggestion based on historical context
type ContextSuggestion struct {
	ID            string                    `json:"id"`
	Type          SuggestionType            `json:"type"`
	Title         string                    `json:"title"`
	Description   string                    `json:"description"`
	Relevance     float64                   `json:"relevance"` // 0.0 to 1.0
	Source        SuggestionSource          `json:"source"`
	RelatedChunks []types.ConversationChunk `json:"related_chunks"`
	ActionType    ActionType                `json:"action_type"`
	Context       map[string]interface{}    `json:"context"`
	CreatedAt     time.Time                 `json:"created_at"`
}

// SuggestionType represents different types of suggestions
type SuggestionType string

const (
	// SuggestionTypeSimilarProblem represents suggestions based on similar past problems
	SuggestionTypeSimilarProblem SuggestionType = "similar_problem"
	// SuggestionTypeArchitectural represents architectural pattern suggestions
	SuggestionTypeArchitectural     SuggestionType = "architectural_pattern"
	SuggestionTypePastDecision      SuggestionType = "past_decision"
	SuggestionTypeDuplicateWork     SuggestionType = "duplicate_work"
	SuggestionTypeSuccessfulPattern SuggestionType = "successful_pattern"
	SuggestionTypeTechnicalDebt     SuggestionType = "technical_debt"
	SuggestionTypeOptimization      SuggestionType = "optimization"
	SuggestionTypeFlowBased         SuggestionType = "flow_based"
	SuggestionTypeDebuggingContext  SuggestionType = "debugging_context"
	SuggestionTypeImplementContext  SuggestionType = "implement_context"
)

// SuggestionSource indicates where the suggestion came from
type SuggestionSource string

const (
	// SourceVectorSearch represents suggestions from vector similarity search
	SourceVectorSearch SuggestionSource = "vector_search"
	// SourcePatternAnalysis represents suggestions from pattern analysis
	SourcePatternAnalysis SuggestionSource = "pattern_analysis"
	SourceTodoHistory     SuggestionSource = "todo_history"
	SourceDecisionLog     SuggestionSource = "decision_log"
	SourceFlowAnalysis    SuggestionSource = "flow_analysis"
)

// ActionType suggests what action Claude should take
type ActionType string

const (
	// ActionReview suggests reviewing previous decisions or patterns
	ActionReview ActionType = "review"
	// ActionConsider suggests considering alternatives or approaches
	ActionConsider  ActionType = "consider"
	ActionAvoid     ActionType = "avoid"
	ActionImplement ActionType = "implement"
	ActionOptimize  ActionType = "optimize"
)

// SuggestionTrigger represents conditions that trigger suggestions
type SuggestionTrigger struct {
	Keywords       []string                 `json:"keywords"`
	ToolsUsed      []string                 `json:"tools_used"`
	FlowPatterns   []types.ConversationFlow `json:"flow_patterns"`
	FilePatterns   []string                 `json:"file_patterns"`
	ErrorPatterns  []string                 `json:"error_patterns"`
	MinRelevance   float64                  `json:"min_relevance"`
	MaxSuggestions int                      `json:"max_suggestions"`
}

// ContextSuggester provides proactive suggestions based on historical context
type ContextSuggester struct {
	vectorStorage     VectorStorage
	patternAnalyzer   *PatternAnalyzer
	todoTracker       *TodoTracker
	flowDetector      *FlowDetector
	triggers          map[SuggestionType]SuggestionTrigger
	activeSuggestions map[string][]ContextSuggestion
}

// VectorStorage interface for accessing historical chunks
type VectorStorage interface {
	Search(ctx context.Context, query string, filters map[string]interface{}, limit int) ([]types.ConversationChunk, error)
	FindSimilar(ctx context.Context, content string, chunkType *types.ChunkType, limit int) ([]types.ConversationChunk, error)
}

// NewContextSuggester creates a new context suggester
func NewContextSuggester(storage VectorStorage, patternAnalyzer *PatternAnalyzer, todoTracker *TodoTracker, flowDetector *FlowDetector) *ContextSuggester {
	suggester := &ContextSuggester{
		vectorStorage:     storage,
		patternAnalyzer:   patternAnalyzer,
		todoTracker:       todoTracker,
		flowDetector:      flowDetector,
		triggers:          make(map[SuggestionType]SuggestionTrigger),
		activeSuggestions: make(map[string][]ContextSuggestion),
	}

	suggester.initializeTriggers()
	return suggester
}

// initializeTriggers sets up suggestion triggers
func (cs *ContextSuggester) initializeTriggers() {
	// Similar problem triggers
	cs.triggers[SuggestionTypeSimilarProblem] = SuggestionTrigger{
		Keywords:       []string{"error", "issue", "problem", "bug", "failed", "exception"},
		ToolsUsed:      []string{"Read", "Grep", "LS"},
		FlowPatterns:   []types.ConversationFlow{types.FlowProblem, types.FlowInvestigation},
		ErrorPatterns:  []string{`(?i)error:`, `(?i)exception:`, `(?i)failed to`},
		MinRelevance:   0.7,
		MaxSuggestions: 3,
	}

	// Architectural pattern triggers
	cs.triggers[SuggestionTypeArchitectural] = SuggestionTrigger{
		Keywords:       []string{"architecture", "design", "structure", "pattern", "implement"},
		ToolsUsed:      []string{"Write", "Edit", "MultiEdit"},
		FlowPatterns:   []types.ConversationFlow{types.FlowSolution},
		FilePatterns:   []string{`\.go$`, `\.js$`, `\.py$`, `\.rs$`},
		MinRelevance:   0.6,
		MaxSuggestions: 2,
	}

	// Past decision triggers
	cs.triggers[SuggestionTypePastDecision] = SuggestionTrigger{
		Keywords:       []string{"decide", "choose", "option", "alternative", "approach"},
		FlowPatterns:   []types.ConversationFlow{types.FlowSolution},
		MinRelevance:   0.6,
		MaxSuggestions: 2,
	}

	// Duplicate work triggers
	cs.triggers[SuggestionTypeDuplicateWork] = SuggestionTrigger{
		Keywords:       []string{"implement", "create", "add", "build"},
		ToolsUsed:      []string{"Write", "Edit"},
		FlowPatterns:   []types.ConversationFlow{types.FlowSolution},
		MinRelevance:   0.8,
		MaxSuggestions: 2,
	}

	// Successful pattern triggers
	cs.triggers[SuggestionTypeSuccessfulPattern] = SuggestionTrigger{
		ToolsUsed:      []string{"Edit", "Write", "Bash"},
		FlowPatterns:   []types.ConversationFlow{types.FlowSolution, types.FlowVerification},
		MinRelevance:   0.5,
		MaxSuggestions: 1,
	}

	// Flow-based contextual suggestions
	cs.triggers[SuggestionTypeFlowBased] = SuggestionTrigger{
		FlowPatterns:   []types.ConversationFlow{types.FlowProblem, types.FlowInvestigation, types.FlowSolution, types.FlowVerification},
		MinRelevance:   0.6,
		MaxSuggestions: 3,
	}

	// Debugging context suggestions - triggered when debugging flow is detected
	cs.triggers[SuggestionTypeDebuggingContext] = SuggestionTrigger{
		Keywords:       []string{"debug", "trace", "log", "investigate", "error", "exception", "stacktrace"},
		ToolsUsed:      []string{"Read", "Grep", "Bash"},
		FlowPatterns:   []types.ConversationFlow{types.FlowProblem, types.FlowInvestigation},
		ErrorPatterns:  []string{`(?i)(error|exception|panic|fatal)`, `(?i)(debug|trace|log)`},
		MinRelevance:   0.7,
		MaxSuggestions: 4,
	}

	// Implementation context suggestions - triggered when implementing solutions
	cs.triggers[SuggestionTypeImplementContext] = SuggestionTrigger{
		Keywords:       []string{"implement", "create", "build", "develop", "code", "function", "method", "class"},
		ToolsUsed:      []string{"Edit", "Write", "MultiEdit"},
		FlowPatterns:   []types.ConversationFlow{types.FlowSolution},
		FilePatterns:   []string{`\.go$`, `\.js$`, `\.ts$`, `\.py$`, `\.rs$`, `\.java$`, `\.cpp$`},
		MinRelevance:   0.6,
		MaxSuggestions: 3,
	}
}

// AnalyzeContext analyzes current context and generates suggestions
func (cs *ContextSuggester) AnalyzeContext(ctx context.Context, sessionID, repository, currentContent, toolUsed string, currentFlow types.ConversationFlow) ([]ContextSuggestion, error) {
	suggestions := make([]ContextSuggestion, 0)

	// Generate suggestions for each trigger type
	for suggestionType := range cs.triggers {
		trigger := cs.triggers[suggestionType]
		if cs.shouldTrigger(currentContent, toolUsed, currentFlow, &trigger) {
			typeSuggestions, err := cs.generateSuggestions(ctx, suggestionType, &trigger, repository, currentContent)
			if err != nil {
				continue // Log error but don't fail completely
			}
			suggestions = append(suggestions, typeSuggestions...)
		}
	}

	// Sort by relevance and limit total suggestions
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Relevance > suggestions[j].Relevance
	})

	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	// Store active suggestions for this session
	cs.activeSuggestions[sessionID] = suggestions

	return suggestions, nil
}

// shouldTrigger determines if a trigger condition is met
func (cs *ContextSuggester) shouldTrigger(content, toolUsed string, flow types.ConversationFlow, trigger *SuggestionTrigger) bool {
	contentLower := strings.ToLower(content)

	// Check keywords
	keywordMatch := len(trigger.Keywords) == 0
	for _, keyword := range trigger.Keywords {
		if strings.Contains(contentLower, keyword) {
			keywordMatch = true
			break
		}
	}

	// Check tools used
	toolMatch := len(trigger.ToolsUsed) == 0
	for _, tool := range trigger.ToolsUsed {
		if tool == toolUsed {
			toolMatch = true
			break
		}
	}

	// Check flow patterns
	flowMatch := len(trigger.FlowPatterns) == 0
	for _, flowPattern := range trigger.FlowPatterns {
		if flowPattern == flow {
			flowMatch = true
			break
		}
	}

	return keywordMatch && toolMatch && flowMatch
}

// generateSuggestions creates suggestions for a specific type
func (cs *ContextSuggester) generateSuggestions(ctx context.Context, suggestionType SuggestionType, trigger *SuggestionTrigger, repository, content string) ([]ContextSuggestion, error) {
	switch suggestionType {
	case SuggestionTypeSimilarProblem:
		return cs.generateSimilarProblemSuggestions(ctx, trigger, repository, content)
	case SuggestionTypeArchitectural:
		return cs.generateArchitecturalSuggestions(ctx, trigger, repository, content)
	case SuggestionTypePastDecision:
		return cs.generatePastDecisionSuggestions(ctx, trigger, repository, content)
	case SuggestionTypeDuplicateWork:
		return cs.generateDuplicateWorkSuggestions(ctx, trigger, repository, content)
	case SuggestionTypeSuccessfulPattern:
		return cs.generateSuccessfulPatternSuggestions(ctx, trigger, repository, content)
	case SuggestionTypeFlowBased:
		return cs.generateFlowBasedSuggestions(ctx, trigger, repository, content)
	case SuggestionTypeDebuggingContext:
		return cs.generateDebuggingContextSuggestions(ctx, trigger, repository, content)
	case SuggestionTypeImplementContext:
		return cs.generateImplementationContextSuggestions(ctx, trigger, repository, content)
	case SuggestionTypeTechnicalDebt, SuggestionTypeOptimization:
		// These suggestion types are not yet implemented
		return []ContextSuggestion{}, nil
	default:
		return []ContextSuggestion{}, nil
	}
}

// generateSimilarProblemSuggestions finds similar past problems and solutions
func (cs *ContextSuggester) generateSimilarProblemSuggestions(ctx context.Context, trigger *SuggestionTrigger, _ /* repository */, content string) ([]ContextSuggestion, error) {
	// Search for similar problems
	problemType := types.ChunkTypeProblem
	similarChunks, err := cs.vectorStorage.FindSimilar(ctx, content, &problemType, trigger.MaxSuggestions*2)
	if err != nil {
		return nil, err
	}

	suggestions := make([]ContextSuggestion, 0)

	for i := range similarChunks {
		chunk := similarChunks[i]
		// Look for associated solution chunks
		solutionChunks, err := cs.findAssociatedSolutions(ctx, &chunk)
		if err != nil {
			continue
		}

		if len(solutionChunks) > 0 {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          SuggestionTypeSimilarProblem,
				Title:         "Similar problem resolved previously",
				Description:   cs.buildSimilarProblemDescription(&chunk, solutionChunks),
				Relevance:     cs.calculateRelevance(content, chunk.Content),
				Source:        SourceVectorSearch,
				RelatedChunks: append([]types.ConversationChunk{chunk}, solutionChunks...),
				ActionType:    ActionReview,
				Context:       map[string]interface{}{"original_problem": chunk.ID, "solutions": len(solutionChunks)},
				CreatedAt:     time.Now(),
			}

			if suggestion.Relevance >= trigger.MinRelevance {
				suggestions = append(suggestions, suggestion)
			}
		}

		if len(suggestions) >= trigger.MaxSuggestions {
			break
		}
	}

	return suggestions, nil
}

// searchAndCreateSuggestions is a helper to reduce duplication in suggestion generation
func (cs *ContextSuggester) searchAndCreateSuggestions(
	ctx context.Context,
	content string,
	chunkType types.ChunkType,
	trigger *SuggestionTrigger,
	suggestionType SuggestionType,
	title string,
	source SuggestionSource,
	actionType ActionType,
	buildDescription func(*types.ConversationChunk) string,
	buildContext func(*types.ConversationChunk) map[string]interface{},
) ([]ContextSuggestion, error) {
	chunks, err := cs.vectorStorage.FindSimilar(ctx, content, &chunkType, trigger.MaxSuggestions)
	if err != nil {
		return nil, err
	}

	suggestions := make([]ContextSuggestion, 0)

	for i := range chunks {
		relevance := cs.calculateRelevance(content, chunks[i].Content)

		if relevance >= trigger.MinRelevance {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          suggestionType,
				Title:         title,
				Description:   buildDescription(&chunks[i]),
				Relevance:     relevance,
				Source:        source,
				RelatedChunks: []types.ConversationChunk{chunks[i]},
				ActionType:    actionType,
				Context:       buildContext(&chunks[i]),
				CreatedAt:     time.Now(),
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions, nil
}

// generateArchitecturalSuggestions suggests relevant architectural patterns
func (cs *ContextSuggester) generateArchitecturalSuggestions(ctx context.Context, trigger *SuggestionTrigger, _ /* repository */, content string) ([]ContextSuggestion, error) {
	return cs.searchAndCreateSuggestions(
		ctx,
		content,
		types.ChunkTypeArchitectureDecision,
		trigger,
		SuggestionTypeArchitectural,
		"Relevant architectural pattern",
		SourceDecisionLog,
		ActionConsider,
		cs.buildArchitecturalDescription,
		func(chunk *types.ConversationChunk) map[string]interface{} {
			return map[string]interface{}{
				"decision_id": chunk.ID,
				"repository":  chunk.Metadata.Repository,
			}
		},
	)
}

// generatePastDecisionSuggestions reminds about relevant past decisions
func (cs *ContextSuggester) generatePastDecisionSuggestions(ctx context.Context, trigger *SuggestionTrigger, repository, content string) ([]ContextSuggestion, error) {
	// Search within same repository for past decisions
	filters := map[string]interface{}{
		"repository": repository,
		"type":       types.ChunkTypeArchitectureDecision,
	}

	chunks, err := cs.vectorStorage.Search(ctx, content, filters, trigger.MaxSuggestions)
	if err != nil {
		return nil, err
	}

	suggestions := make([]ContextSuggestion, 0)

	for i := range chunks {
		chunk := chunks[i]
		relevance := cs.calculateRelevance(content, chunk.Content)

		if relevance >= trigger.MinRelevance {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          SuggestionTypePastDecision,
				Title:         "Past decision may be relevant",
				Description:   cs.buildPastDecisionDescription(&chunk),
				Relevance:     relevance,
				Source:        SourceDecisionLog,
				RelatedChunks: []types.ConversationChunk{chunk},
				ActionType:    ActionReview,
				Context:       map[string]interface{}{"decision_date": chunk.Timestamp, "repository": repository},
				CreatedAt:     time.Now(),
			}
			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions, nil
}

// generateDuplicateWorkSuggestions alerts to potential duplicate work
func (cs *ContextSuggester) generateDuplicateWorkSuggestions(ctx context.Context, trigger *SuggestionTrigger, _ /* repository */, content string) ([]ContextSuggestion, error) {
	return cs.searchAndCreateSuggestions(
		ctx,
		content,
		types.ChunkTypeSolution,
		trigger,
		SuggestionTypeDuplicateWork,
		"Similar work already exists",
		SourceVectorSearch,
		ActionReview,
		cs.buildDuplicateWorkDescription,
		func(chunk *types.ConversationChunk) map[string]interface{} {
			return map[string]interface{}{
				"existing_work": chunk.ID,
				"outcome":       chunk.Metadata.Outcome,
			}
		},
	)
}

// generateSuccessfulPatternSuggestions suggests proven successful patterns
func (cs *ContextSuggester) generateSuccessfulPatternSuggestions(_ /* ctx */ context.Context, trigger *SuggestionTrigger, _ /* repository */, _ /* content */ string) ([]ContextSuggestion, error) {
	// Get successful patterns from pattern analyzer
	if cs.patternAnalyzer == nil {
		return []ContextSuggestion{}, nil
	}

	patterns := cs.patternAnalyzer.GetSuccessPatterns()
	suggestions := make([]ContextSuggestion, 0)

	for _, pattern := range patterns {
		if pattern.SuccessRate > 0.7 && pattern.Frequency > 2 {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          SuggestionTypeSuccessfulPattern,
				Title:         "Successful " + string(pattern.Type) + " pattern available",
				Description:   cs.buildSuccessfulPatternDescription(&pattern),
				Relevance:     pattern.SuccessRate * 0.8, // Weight success rate
				Source:        SourcePatternAnalysis,
				RelatedChunks: []types.ConversationChunk{}, // Pattern-based, no specific chunks
				ActionType:    ActionConsider,
				Context:       map[string]interface{}{"pattern_type": pattern.Type, "success_rate": pattern.SuccessRate, "frequency": pattern.Frequency},
				CreatedAt:     time.Now(),
			}
			suggestions = append(suggestions, suggestion)
		}

		if len(suggestions) >= trigger.MaxSuggestions {
			break
		}
	}

	return suggestions, nil
}

// Helper methods for building descriptions

func (cs *ContextSuggester) buildSimilarProblemDescription(problem *types.ConversationChunk, solutions []types.ConversationChunk) string {
	if len(solutions) == 0 {
		return "Found similar problem from " + problem.Timestamp.Format("Jan 2")
	}
	return fmt.Sprintf("Found similar problem from %s with %d solution(s). Success rate: %s",
		problem.Timestamp.Format("Jan 2"), len(solutions), problem.Metadata.Outcome)
}

func (cs *ContextSuggester) buildArchitecturalDescription(chunk *types.ConversationChunk) string {
	return fmt.Sprintf("Architectural decision from %s: %s",
		chunk.Timestamp.Format("Jan 2"), truncateString(chunk.Summary, 100))
}

func (cs *ContextSuggester) buildPastDecisionDescription(chunk *types.ConversationChunk) string {
	return fmt.Sprintf("Decision from %s may apply to current situation: %s",
		chunk.Timestamp.Format("Jan 2"), truncateString(chunk.Summary, 100))
}

func (cs *ContextSuggester) buildDuplicateWorkDescription(chunk *types.ConversationChunk) string {
	return fmt.Sprintf("Similar implementation from %s (outcome: %s): %s",
		chunk.Timestamp.Format("Jan 2"), chunk.Metadata.Outcome, truncateString(chunk.Summary, 80))
}

func (cs *ContextSuggester) buildSuccessfulPatternDescription(pattern *SuccessPattern) string {
	return fmt.Sprintf("%s (%.1f%% success rate, used %d times)",
		pattern.Description, pattern.SuccessRate*100, pattern.Frequency)
}

// findAssociatedSolutions finds solution chunks related to a problem
func (cs *ContextSuggester) findAssociatedSolutions(ctx context.Context, problemChunk *types.ConversationChunk) ([]types.ConversationChunk, error) {
	// Search for solutions in the same session
	filters := map[string]interface{}{
		"session_id": problemChunk.SessionID,
		"type":       types.ChunkTypeSolution,
	}

	return cs.vectorStorage.Search(ctx, problemChunk.Content, filters, 3)
}

// calculateRelevance computes relevance score between two pieces of content
func (cs *ContextSuggester) calculateRelevance(current, historical string) float64 {
	// Simple relevance calculation based on common words
	currentWords := strings.Fields(strings.ToLower(current))
	historicalWords := strings.Fields(strings.ToLower(historical))

	currentWordSet := make(map[string]bool)
	for _, word := range currentWords {
		if len(word) > 3 { // Ignore short words
			currentWordSet[word] = true
		}
	}

	matches := 0
	historicalWordCount := 0
	for _, word := range historicalWords {
		if len(word) > 3 {
			historicalWordCount++
			if currentWordSet[word] {
				matches++
			}
		}
	}

	if historicalWordCount == 0 {
		return 0.0
	}

	return float64(matches) / float64(historicalWordCount)
}

// GetActiveSuggestions returns current suggestions for a session
func (cs *ContextSuggester) GetActiveSuggestions(sessionID string) []ContextSuggestion {
	if suggestions, exists := cs.activeSuggestions[sessionID]; exists {
		return suggestions
	}
	return []ContextSuggestion{}
}

// ClearSuggestions removes suggestions for a session
func (cs *ContextSuggester) ClearSuggestions(sessionID string) {
	delete(cs.activeSuggestions, sessionID)
}

// generateSuggestionID creates unique IDs for suggestions
func generateSuggestionID() string {
	return "sug-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Enhanced Flow-Based Suggestion Methods

// generateFlowBasedSuggestions provides contextual suggestions based on current conversation flow
func (cs *ContextSuggester) generateFlowBasedSuggestions(ctx context.Context, trigger *SuggestionTrigger, repository, content string) ([]ContextSuggestion, error) {
	suggestions := make([]ContextSuggestion, 0)

	// Detect current flow from content if not provided by calling context
	var currentFlow types.ConversationFlow
	var confidence float64
	if cs.flowDetector != nil {
		currentFlow, confidence = cs.flowDetector.detectFlow(content, "")
	} else {
		// Default to unknown flow if detector is not available
		currentFlow = types.FlowProblem
		confidence = 0.0
	}

	// Get flow-specific suggestions based on detected conversation flow
	switch currentFlow {
	case types.FlowProblem:
		flowSuggestions, err := cs.generateProblemFlowSuggestions(ctx, trigger, repository, content)
		if err == nil {
			suggestions = append(suggestions, flowSuggestions...)
		}
	case types.FlowInvestigation:
		flowSuggestions, err := cs.generateInvestigationFlowSuggestions(ctx, trigger, repository, content)
		if err == nil {
			suggestions = append(suggestions, flowSuggestions...)
		}
	case types.FlowSolution:
		flowSuggestions, err := cs.generateSolutionFlowSuggestions(ctx, trigger, repository, content)
		if err == nil {
			suggestions = append(suggestions, flowSuggestions...)
		}
	case types.FlowVerification:
		flowSuggestions, err := cs.generateVerificationFlowSuggestions(ctx, trigger, repository, content)
		if err == nil {
			suggestions = append(suggestions, flowSuggestions...)
		}
	}

	// Filter by confidence and relevance
	filtered := make([]ContextSuggestion, 0)
	for i := range suggestions {
		suggestion := suggestions[i]
		if confidence > 0.5 && suggestion.Relevance >= trigger.MinRelevance {
			suggestion.Context["flow_confidence"] = confidence
			suggestion.Context["detected_flow"] = string(currentFlow)
			filtered = append(filtered, suggestion)
		}
	}

	return filtered, nil
}

// generateProblemFlowSuggestions suggests relevant memories when problems are detected
func (cs *ContextSuggester) generateProblemFlowSuggestions(ctx context.Context, trigger *SuggestionTrigger, _, content string) ([]ContextSuggestion, error) {
	// Find similar problems that were successfully resolved
	problemType := types.ChunkTypeProblem
	similarProblems, err := cs.vectorStorage.FindSimilar(ctx, content, &problemType, trigger.MaxSuggestions*2)
	if err != nil {
		return nil, err
	}

	suggestions := make([]ContextSuggestion, 0)
	for i := range similarProblems {
		problemChunk := similarProblems[i]
		// Look for associated solutions
		solutions, err := cs.findAssociatedSolutions(ctx, &problemChunk)
		if err != nil || len(solutions) == 0 {
			continue
		}

		suggestion := ContextSuggestion{
			ID:            generateSuggestionID(),
			Type:          SuggestionTypeFlowBased,
			Title:         "🔍 Similar problem with solution found",
			Description:   "Found a similar problem that was resolved: " + truncateString(problemChunk.Summary, 100),
			Relevance:     cs.calculateRelevance(content, problemChunk.Content),
			Source:        SourceFlowAnalysis,
			RelatedChunks: append([]types.ConversationChunk{problemChunk}, solutions...),
			ActionType:    ActionReview,
			Context: map[string]interface{}{
				"flow_stage":     "problem_detected",
				"solution_count": len(solutions),
				"problem_id":     problemChunk.ID,
			},
			CreatedAt: time.Now(),
		}

		suggestions = append(suggestions, suggestion)
		if len(suggestions) >= trigger.MaxSuggestions {
			break
		}
	}

	return suggestions, nil
}

// generateInvestigationFlowSuggestions suggests debugging approaches and investigation patterns
func (cs *ContextSuggester) generateInvestigationFlowSuggestions(ctx context.Context, trigger *SuggestionTrigger, _, content string) ([]ContextSuggestion, error) {
	// Find successful investigation and debugging patterns
	searchTerms := content + " investigation debugging analysis"
	chunks, err := cs.vectorStorage.Search(ctx, searchTerms, map[string]interface{}{
		"tags": []string{"debugging", "investigation", "analysis"},
	}, trigger.MaxSuggestions*2)
	if err != nil {
		return nil, err
	}

	suggestions := make([]ContextSuggestion, 0)
	for i := range chunks {
		chunk := chunks[i]
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          SuggestionTypeFlowBased,
				Title:         "🔬 Successful investigation approach found",
				Description:   "Similar investigation was successful: " + truncateString(chunk.Summary, 100),
				Relevance:     cs.calculateRelevance(content, chunk.Content),
				Source:        SourceFlowAnalysis,
				RelatedChunks: []types.ConversationChunk{chunk},
				ActionType:    ActionConsider,
				Context: map[string]interface{}{
					"flow_stage":       "investigation",
					"investigation_id": chunk.ID,
					"tools_used":       chunk.Metadata.ToolsUsed,
				},
				CreatedAt: time.Now(),
			}

			suggestions = append(suggestions, suggestion)
			if len(suggestions) >= trigger.MaxSuggestions {
				break
			}
		}
	}

	return suggestions, nil
}

// generateSolutionFlowSuggestions suggests implementation patterns and architectural decisions
func (cs *ContextSuggester) generateSolutionFlowSuggestions(ctx context.Context, trigger *SuggestionTrigger, _, content string) ([]ContextSuggestion, error) {
	// Find similar implementation patterns and successful solutions
	solutionType := types.ChunkTypeSolution
	solutions, err := cs.vectorStorage.FindSimilar(ctx, content, &solutionType, trigger.MaxSuggestions*2)
	if err != nil {
		return nil, err
	}

	suggestions := make([]ContextSuggestion, 0)
	for i := range solutions {
		solution := solutions[i]
		if solution.Metadata.Outcome == types.OutcomeSuccess {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          SuggestionTypeFlowBased,
				Title:         "💡 Similar implementation pattern found",
				Description:   "Successful implementation approach: " + truncateString(solution.Summary, 100),
				Relevance:     cs.calculateRelevance(content, solution.Content),
				Source:        SourceFlowAnalysis,
				RelatedChunks: []types.ConversationChunk{solution},
				ActionType:    ActionConsider,
				Context: map[string]interface{}{
					"flow_stage":    "solution",
					"solution_id":   solution.ID,
					"files_changed": solution.Metadata.FilesModified,
					"tags":          solution.Metadata.Tags,
				},
				CreatedAt: time.Now(),
			}

			suggestions = append(suggestions, suggestion)
			if len(suggestions) >= trigger.MaxSuggestions {
				break
			}
		}
	}

	return suggestions, nil
}

// generateVerificationFlowSuggestions suggests testing approaches and verification patterns
func (cs *ContextSuggester) generateVerificationFlowSuggestions(ctx context.Context, trigger *SuggestionTrigger, _, content string) ([]ContextSuggestion, error) {
	// Find successful verification and testing patterns
	searchTerms := content + " testing verification validation"
	chunks, err := cs.vectorStorage.Search(ctx, searchTerms, map[string]interface{}{
		"tags": []string{"testing", "verification", "validation"},
	}, trigger.MaxSuggestions)
	if err != nil {
		return nil, err
	}

	suggestions := make([]ContextSuggestion, 0)
	for i := range chunks {
		chunk := chunks[i]
		suggestion := ContextSuggestion{
			ID:            generateSuggestionID(),
			Type:          SuggestionTypeFlowBased,
			Title:         "✅ Testing approach found",
			Description:   "Verification method: " + truncateString(chunk.Summary, 100),
			Relevance:     cs.calculateRelevance(content, chunk.Content),
			Source:        SourceFlowAnalysis,
			RelatedChunks: []types.ConversationChunk{chunk},
			ActionType:    ActionImplement,
			Context: map[string]interface{}{
				"flow_stage":      "verification",
				"verification_id": chunk.ID,
				"test_approach":   chunk.Metadata.Tags,
			},
			CreatedAt: time.Now(),
		}

		suggestions = append(suggestions, suggestion)
		if len(suggestions) >= trigger.MaxSuggestions {
			break
		}
	}

	return suggestions, nil
}

// generateDebuggingContextSuggestions provides debugging-specific contextual suggestions
func (cs *ContextSuggester) generateDebuggingContextSuggestions(ctx context.Context, trigger *SuggestionTrigger, _, content string) ([]ContextSuggestion, error) {
	// Extract error patterns and look for similar debugging sessions
	searchTerms := content + " debugging error exception trace"
	chunks, err := cs.vectorStorage.Search(ctx, searchTerms, map[string]interface{}{
		"tags": []string{"debugging", "error-resolution", "troubleshooting"},
		"type": []string{string(types.ChunkTypeProblem), string(types.ChunkTypeSolution)},
	}, trigger.MaxSuggestions*2)
	if err != nil {
		return nil, err
	}

	suggestions := make([]ContextSuggestion, 0)
	for i := range chunks {
		chunk := chunks[i]
		if chunk.Metadata.Outcome == types.OutcomeSuccess {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          SuggestionTypeDebuggingContext,
				Title:         "🐛 Similar debugging success found",
				Description:   "Debugging approach that worked: " + truncateString(chunk.Summary, 100),
				Relevance:     cs.calculateRelevance(content, chunk.Content),
				Source:        SourceFlowAnalysis,
				RelatedChunks: []types.ConversationChunk{chunk},
				ActionType:    ActionReview,
				Context: map[string]interface{}{
					"debugging_type": "error_resolution",
					"chunk_id":       chunk.ID,
					"tools_used":     chunk.Metadata.ToolsUsed,
					"difficulty":     chunk.Metadata.Difficulty,
				},
				CreatedAt: time.Now(),
			}

			suggestions = append(suggestions, suggestion)
			if len(suggestions) >= trigger.MaxSuggestions {
				break
			}
		}
	}

	return suggestions, nil
}

// generateImplementationContextSuggestions provides implementation-specific contextual suggestions
func (cs *ContextSuggester) generateImplementationContextSuggestions(ctx context.Context, trigger *SuggestionTrigger, _, content string) ([]ContextSuggestion, error) {
	// Find similar implementation patterns and code changes
	searchTerms := content + " implementation code function class method"
	chunks, err := cs.vectorStorage.Search(ctx, searchTerms, map[string]interface{}{
		"tags": []string{"implementation", "code-change", "feature"},
		"type": []string{string(types.ChunkTypeCodeChange), string(types.ChunkTypeSolution)},
	}, trigger.MaxSuggestions*2)
	if err != nil {
		return nil, err
	}

	suggestions := make([]ContextSuggestion, 0)
	for i := range chunks {
		chunk := chunks[i]
		if chunk.Metadata.Outcome == types.OutcomeSuccess && len(chunk.Metadata.FilesModified) > 0 {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          SuggestionTypeImplementContext,
				Title:         "⚙️ Similar implementation found",
				Description:   "Implementation pattern: " + truncateString(chunk.Summary, 100),
				Relevance:     cs.calculateRelevance(content, chunk.Content),
				Source:        SourceFlowAnalysis,
				RelatedChunks: []types.ConversationChunk{chunk},
				ActionType:    ActionConsider,
				Context: map[string]interface{}{
					"implementation_type": "code_change",
					"chunk_id":            chunk.ID,
					"files_modified":      chunk.Metadata.FilesModified,
					"complexity":          chunk.Metadata.Difficulty,
					"tags":                chunk.Metadata.Tags,
				},
				CreatedAt: time.Now(),
			}

			suggestions = append(suggestions, suggestion)
			if len(suggestions) >= trigger.MaxSuggestions {
				break
			}
		}
	}

	return suggestions, nil
}
