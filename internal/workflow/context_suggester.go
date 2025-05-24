// Package workflow provides proactive context suggestions for Claude
package workflow

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"mcp-memory/pkg/types"
)

// ContextSuggestion represents a proactive suggestion based on historical context
type ContextSuggestion struct {
	ID          string                 `json:"id"`
	Type        SuggestionType         `json:"type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Relevance   float64               `json:"relevance"` // 0.0 to 1.0
	Source      SuggestionSource       `json:"source"`
	RelatedChunks []types.ConversationChunk `json:"related_chunks"`
	ActionType   ActionType            `json:"action_type"`
	Context     map[string]interface{} `json:"context"`
	CreatedAt   time.Time             `json:"created_at"`
}

// SuggestionType represents different types of suggestions
type SuggestionType string

const (
	SuggestionTypeSimilarProblem    SuggestionType = "similar_problem"
	SuggestionTypeArchitectural     SuggestionType = "architectural_pattern"
	SuggestionTypePastDecision      SuggestionType = "past_decision"
	SuggestionTypeDuplicateWork     SuggestionType = "duplicate_work"
	SuggestionTypeSuccessfulPattern SuggestionType = "successful_pattern"
	SuggestionTypeTechnicalDebt     SuggestionType = "technical_debt"
	SuggestionTypeOptimization      SuggestionType = "optimization"
)

// SuggestionSource indicates where the suggestion came from
type SuggestionSource string

const (
	SourceVectorSearch    SuggestionSource = "vector_search"
	SourcePatternAnalysis SuggestionSource = "pattern_analysis"
	SourceTodoHistory     SuggestionSource = "todo_history"
	SourceDecisionLog     SuggestionSource = "decision_log"
	SourceFlowAnalysis    SuggestionSource = "flow_analysis"
)

// ActionType suggests what action Claude should take
type ActionType string

const (
	ActionReview     ActionType = "review"
	ActionConsider   ActionType = "consider"
	ActionAvoid      ActionType = "avoid"
	ActionImplement  ActionType = "implement"
	ActionOptimize   ActionType = "optimize"
)

// SuggestionTrigger represents conditions that trigger suggestions
type SuggestionTrigger struct {
	Keywords       []string                    `json:"keywords"`
	ToolsUsed      []string                    `json:"tools_used"`
	FlowPatterns   []types.ConversationFlow    `json:"flow_patterns"`
	FilePatterns   []string                    `json:"file_patterns"`
	ErrorPatterns  []string                    `json:"error_patterns"`
	MinRelevance   float64                     `json:"min_relevance"`
	MaxSuggestions int                         `json:"max_suggestions"`
}

// ContextSuggester provides proactive suggestions based on historical context
type ContextSuggester struct {
	vectorStorage   VectorStorage
	patternAnalyzer *PatternAnalyzer
	todoTracker     *TodoTracker
	flowDetector    *FlowDetector
	triggers        map[SuggestionType]SuggestionTrigger
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
}

// AnalyzeContext analyzes current context and generates suggestions
func (cs *ContextSuggester) AnalyzeContext(ctx context.Context, sessionID, repository, currentContent string, toolUsed string, currentFlow types.ConversationFlow) ([]ContextSuggestion, error) {
	suggestions := make([]ContextSuggestion, 0)
	
	// Generate suggestions for each trigger type
	for suggestionType, trigger := range cs.triggers {
		if cs.shouldTrigger(currentContent, toolUsed, currentFlow, trigger) {
			typeSuggestions, err := cs.generateSuggestions(ctx, suggestionType, trigger, repository, currentContent)
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
func (cs *ContextSuggester) shouldTrigger(content, toolUsed string, flow types.ConversationFlow, trigger SuggestionTrigger) bool {
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
func (cs *ContextSuggester) generateSuggestions(ctx context.Context, suggestionType SuggestionType, trigger SuggestionTrigger, repository, content string) ([]ContextSuggestion, error) {
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
	default:
		return []ContextSuggestion{}, nil
	}
}

// generateSimilarProblemSuggestions finds similar past problems and solutions
func (cs *ContextSuggester) generateSimilarProblemSuggestions(ctx context.Context, trigger SuggestionTrigger, _ /* repository */, content string) ([]ContextSuggestion, error) {
	// Search for similar problems
	problemType := types.ChunkTypeProblem
	similarChunks, err := cs.vectorStorage.FindSimilar(ctx, content, &problemType, trigger.MaxSuggestions*2)
	if err != nil {
		return nil, err
	}
	
	suggestions := make([]ContextSuggestion, 0)
	
	for _, chunk := range similarChunks {
		// Look for associated solution chunks
		solutionChunks, err := cs.findAssociatedSolutions(ctx, chunk)
		if err != nil {
			continue
		}
		
		if len(solutionChunks) > 0 {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          SuggestionTypeSimilarProblem,
				Title:         "Similar problem resolved previously",
				Description:   cs.buildSimilarProblemDescription(chunk, solutionChunks),
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

// generateArchitecturalSuggestions suggests relevant architectural patterns
func (cs *ContextSuggester) generateArchitecturalSuggestions(ctx context.Context, trigger SuggestionTrigger, _ /* repository */, content string) ([]ContextSuggestion, error) {
	// Search for architectural decisions
	decisionType := types.ChunkTypeArchitectureDecision
	decisionChunks, err := cs.vectorStorage.FindSimilar(ctx, content, &decisionType, trigger.MaxSuggestions)
	if err != nil {
		return nil, err
	}
	
	suggestions := make([]ContextSuggestion, 0)
	
	for _, chunk := range decisionChunks {
		relevance := cs.calculateRelevance(content, chunk.Content)
		
		if relevance >= trigger.MinRelevance {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          SuggestionTypeArchitectural,
				Title:         "Relevant architectural pattern",
				Description:   cs.buildArchitecturalDescription(chunk),
				Relevance:     relevance,
				Source:        SourceDecisionLog,
				RelatedChunks: []types.ConversationChunk{chunk},
				ActionType:    ActionConsider,
				Context:       map[string]interface{}{"decision_id": chunk.ID, "repository": chunk.Metadata.Repository},
				CreatedAt:     time.Now(),
			}
			suggestions = append(suggestions, suggestion)
		}
	}
	
	return suggestions, nil
}

// generatePastDecisionSuggestions reminds about relevant past decisions
func (cs *ContextSuggester) generatePastDecisionSuggestions(ctx context.Context, trigger SuggestionTrigger, repository, content string) ([]ContextSuggestion, error) {
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
	
	for _, chunk := range chunks {
		relevance := cs.calculateRelevance(content, chunk.Content)
		
		if relevance >= trigger.MinRelevance {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          SuggestionTypePastDecision,
				Title:         "Past decision may be relevant",
				Description:   cs.buildPastDecisionDescription(chunk),
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
func (cs *ContextSuggester) generateDuplicateWorkSuggestions(ctx context.Context, trigger SuggestionTrigger, _ /* repository */, content string) ([]ContextSuggestion, error) {
	// Search for similar implementations
	solutionType := types.ChunkTypeSolution
	existingWork, err := cs.vectorStorage.FindSimilar(ctx, content, &solutionType, trigger.MaxSuggestions)
	if err != nil {
		return nil, err
	}
	
	suggestions := make([]ContextSuggestion, 0)
	
	for _, chunk := range existingWork {
		relevance := cs.calculateRelevance(content, chunk.Content)
		
		if relevance >= trigger.MinRelevance {
			suggestion := ContextSuggestion{
				ID:            generateSuggestionID(),
				Type:          SuggestionTypeDuplicateWork,
				Title:         "Similar work already exists",
				Description:   cs.buildDuplicateWorkDescription(chunk),
				Relevance:     relevance,
				Source:        SourceVectorSearch,
				RelatedChunks: []types.ConversationChunk{chunk},
				ActionType:    ActionReview,
				Context:       map[string]interface{}{"existing_work": chunk.ID, "outcome": chunk.Metadata.Outcome},
				CreatedAt:     time.Now(),
			}
			suggestions = append(suggestions, suggestion)
		}
	}
	
	return suggestions, nil
}

// generateSuccessfulPatternSuggestions suggests proven successful patterns
func (cs *ContextSuggester) generateSuccessfulPatternSuggestions(_ /* ctx */ context.Context, trigger SuggestionTrigger, _ /* repository */, _ /* content */ string) ([]ContextSuggestion, error) {
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
				Title:         fmt.Sprintf("Successful %s pattern available", pattern.Type),
				Description:   cs.buildSuccessfulPatternDescription(pattern),
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

func (cs *ContextSuggester) buildSimilarProblemDescription(problem types.ConversationChunk, solutions []types.ConversationChunk) string {
	if len(solutions) == 0 {
		return fmt.Sprintf("Found similar problem from %s", problem.Timestamp.Format("Jan 2"))
	}
	return fmt.Sprintf("Found similar problem from %s with %d solution(s). Success rate: %s", 
		problem.Timestamp.Format("Jan 2"), len(solutions), problem.Metadata.Outcome)
}

func (cs *ContextSuggester) buildArchitecturalDescription(chunk types.ConversationChunk) string {
	return fmt.Sprintf("Architectural decision from %s: %s", 
		chunk.Timestamp.Format("Jan 2"), truncateString(chunk.Summary, 100))
}

func (cs *ContextSuggester) buildPastDecisionDescription(chunk types.ConversationChunk) string {
	return fmt.Sprintf("Decision from %s may apply to current situation: %s", 
		chunk.Timestamp.Format("Jan 2"), truncateString(chunk.Summary, 100))
}

func (cs *ContextSuggester) buildDuplicateWorkDescription(chunk types.ConversationChunk) string {
	return fmt.Sprintf("Similar implementation from %s (outcome: %s): %s", 
		chunk.Timestamp.Format("Jan 2"), chunk.Metadata.Outcome, truncateString(chunk.Summary, 80))
}

func (cs *ContextSuggester) buildSuccessfulPatternDescription(pattern SuccessPattern) string {
	return fmt.Sprintf("%s (%.1f%% success rate, used %d times)", 
		pattern.Description, pattern.SuccessRate*100, pattern.Frequency)
}

// findAssociatedSolutions finds solution chunks related to a problem
func (cs *ContextSuggester) findAssociatedSolutions(ctx context.Context, problemChunk types.ConversationChunk) ([]types.ConversationChunk, error) {
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
	return fmt.Sprintf("sug-%d", time.Now().UnixNano())
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}