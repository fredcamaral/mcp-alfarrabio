package intelligence

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// BasicSequenceRecognizer implements the SequenceRecognizer interface
type BasicSequenceRecognizer struct {
	engine *PatternEngine

	// Pattern templates for common sequences
	problemSolutionTemplate []string
	debuggingTemplate       []string
	workflowTemplate        []string
}

// NewSequenceRecognizer creates a new sequence recognizer
func NewSequenceRecognizer(engine *PatternEngine) *BasicSequenceRecognizer {
	return &BasicSequenceRecognizer{
		engine: engine,
		problemSolutionTemplate: []string{
			"report_problem",
			"analyze_problem",
			"propose_solution",
			"implement_solution",
			"verify_solution",
		},
		debuggingTemplate: []string{
			"identify_error",
			"investigate_cause",
			"test_hypothesis",
			"apply_fix",
			"confirm_resolution",
		},
		workflowTemplate: []string{
			"define_objective",
			"plan_approach",
			"execute_steps",
			"review_results",
			"finalize_outcome",
		},
	}
}

// RecognizeSequence identifies patterns in a sequence of conversation chunks
func (bsr *BasicSequenceRecognizer) RecognizeSequence(chunks []types.ConversationChunk) ([]Pattern, error) {
	if len(chunks) < 3 {
		return []Pattern{}, nil
	}

	var recognizedPatterns []Pattern

	// Extract action sequence
	actions := bsr.extractActionSequence(chunks)

	// Try to match against known templates
	if pattern := bsr.matchProblemSolutionSequence(chunks, actions); pattern != nil {
		recognizedPatterns = append(recognizedPatterns, *pattern)
	}

	if pattern := bsr.matchDebuggingSequence(chunks, actions); pattern != nil {
		recognizedPatterns = append(recognizedPatterns, *pattern)
	}

	if pattern := bsr.matchWorkflowSequence(chunks, actions); pattern != nil {
		recognizedPatterns = append(recognizedPatterns, *pattern)
	}

	// Try to identify custom sequences
	if pattern := bsr.identifyCustomSequence(chunks, actions); pattern != nil {
		recognizedPatterns = append(recognizedPatterns, *pattern)
	}

	return recognizedPatterns, nil
}

// LearnFromSequence learns from the outcome of a sequence
func (bsr *BasicSequenceRecognizer) LearnFromSequence(chunks []types.ConversationChunk, outcome PatternOutcome) error {
	if len(chunks) < 2 {
		return nil
	}

	// Extract pattern from sequence
	actions := bsr.extractActionSequence(chunks)
	sequenceType := bsr.classifySequence(actions)

	// Create or update pattern based on the sequence
	pattern := Pattern{
		ID:          generatePatternID(),
		Type:        sequenceType,
		Name:        bsr.generateSequenceName(sequenceType, actions),
		Description: bsr.generateSequenceDescription(sequenceType, outcome),
		Confidence:  bsr.calculateSequenceConfidence(chunks, outcome),
		Frequency:   1,
		SuccessRate: bsr.calculateSuccessRateFromOutcome(outcome),
		Keywords:    bsr.extractSequenceKeywords(chunks),
		Triggers:    bsr.extractSequenceTriggers(chunks),
		Outcomes:    bsr.extractSequenceOutcomes(chunks, outcome),
		Steps:       bsr.convertActionsToSteps(actions, chunks),
		Context:     bsr.extractSequenceContext(chunks),
		Examples: []PatternExample{{
			ID:           fmt.Sprintf("%s_example_%d", generatePatternID(), time.Now().Unix()),
			ChunkIDs:     extractChunkIDs(chunks),
			Conversation: chunks,
			Outcome:      outcome,
			Confidence:   1.0,
			Timestamp:    time.Now(),
		}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	// For now, we'll just store this pattern - in a full implementation,
	// this would integrate with the pattern storage system
	_ = pattern

	return nil
}

// Helper methods for sequence recognition

func (bsr *BasicSequenceRecognizer) extractActionSequence(chunks []types.ConversationChunk) []string {
	actions := make([]string, 0, len(chunks))

	for i := range chunks {
		action := bsr.identifyChunkAction(chunks[i])
		actions = append(actions, action)
	}

	return actions
}

//nolint:gocritic // hugeParam: large struct parameter needed for processing
func (bsr *BasicSequenceRecognizer) identifyChunkAction(chunk types.ConversationChunk) string {
	content := strings.ToLower(chunk.Content)

	// Problem identification patterns
	if regexp.MustCompile(`(?i)(error|issue|problem|bug|fail|broken|not working)`).MatchString(content) {
		return "report_problem"
	}

	// Analysis patterns
	if regexp.MustCompile(`(?i)(analyze|investigate|check|look|examine|debug)`).MatchString(content) {
		return "analyze_problem"
	}

	// Solution patterns
	if regexp.MustCompile(`(?i)(fix|solve|solution|resolve|try|attempt)`).MatchString(content) {
		return "propose_solution"
	}

	// Implementation patterns
	if regexp.MustCompile(`(?i)(create|add|modify|update|change|implement)`).MatchString(content) {
		return "implement_solution"
	}

	// Verification patterns
	if regexp.MustCompile(`(?i)(test|verify|check|confirm|validate|run)`).MatchString(content) {
		return "verify_solution"
	}

	// Planning patterns
	if regexp.MustCompile(`(?i)(plan|design|approach|strategy|outline)`).MatchString(content) {
		return "plan_approach"
	}

	// Execution patterns
	if regexp.MustCompile(`(?i)(execute|perform|do|start|begin)`).MatchString(content) {
		return "execute_steps"
	}

	// Review patterns
	if regexp.MustCompile(`(?i)(review|assess|evaluate|summary|done)`).MatchString(content) {
		return "review_results"
	}

	return "general_interaction"
}

func (bsr *BasicSequenceRecognizer) matchProblemSolutionSequence(chunks []types.ConversationChunk, actions []string) *Pattern {
	similarity := bsr.calculateSequenceSimilarity(actions, bsr.problemSolutionTemplate)

	if similarity > 0.6 {
		return &Pattern{
			ID:          generatePatternID(),
			Type:        PatternTypeProblemSolution,
			Name:        "Problem-Solution Pattern",
			Description: "A conversation pattern where a problem is identified and systematically solved",
			Confidence:  similarity,
			Keywords:    bsr.extractSequenceKeywords(chunks),
			Steps:       bsr.convertActionsToSteps(actions, chunks),
			Context:     bsr.extractSequenceContext(chunks),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	return nil
}

func (bsr *BasicSequenceRecognizer) matchDebuggingSequence(chunks []types.ConversationChunk, actions []string) *Pattern {
	similarity := bsr.calculateSequenceSimilarity(actions, bsr.debuggingTemplate)

	if similarity > 0.6 {
		return &Pattern{
			ID:          generatePatternID(),
			Type:        PatternTypeDebugging,
			Name:        "Debugging Pattern",
			Description: "A systematic debugging and error resolution pattern",
			Confidence:  similarity,
			Keywords:    bsr.extractSequenceKeywords(chunks),
			Steps:       bsr.convertActionsToSteps(actions, chunks),
			Context:     bsr.extractSequenceContext(chunks),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	return nil
}

func (bsr *BasicSequenceRecognizer) matchWorkflowSequence(chunks []types.ConversationChunk, actions []string) *Pattern {
	similarity := bsr.calculateSequenceSimilarity(actions, bsr.workflowTemplate)

	if similarity > 0.6 {
		return &Pattern{
			ID:          generatePatternID(),
			Type:        PatternTypeWorkflow,
			Name:        "Workflow Pattern",
			Description: "A structured workflow pattern for completing tasks",
			Confidence:  similarity,
			Keywords:    bsr.extractSequenceKeywords(chunks),
			Steps:       bsr.convertActionsToSteps(actions, chunks),
			Context:     bsr.extractSequenceContext(chunks),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	return nil
}

func (bsr *BasicSequenceRecognizer) identifyCustomSequence(chunks []types.ConversationChunk, actions []string) *Pattern {
	// Only identify custom sequences if they have a clear structure
	if len(actions) < 3 {
		return nil
	}

	// Look for repeated or structured patterns
	if bsr.hasStructuredPattern(actions) {
		return &Pattern{
			ID:          generatePatternID(),
			Type:        PatternTypeWorkflow,
			Name:        "Custom Workflow Pattern",
			Description: "A custom conversation pattern identified from user interactions",
			Confidence:  0.7,
			Keywords:    bsr.extractSequenceKeywords(chunks),
			Steps:       bsr.convertActionsToSteps(actions, chunks),
			Context:     bsr.extractSequenceContext(chunks),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
	}

	return nil
}

func (bsr *BasicSequenceRecognizer) calculateSequenceSimilarity(actions, template []string) float64 {
	if len(actions) == 0 || len(template) == 0 {
		return 0.0
	}

	// Use dynamic programming to find longest common subsequence
	matches := bsr.longestCommonSubsequence(actions, template)

	// Calculate similarity as ratio of matches to template length
	similarity := float64(matches) / float64(len(template))

	return math.Min(similarity, 1.0)
}

func (bsr *BasicSequenceRecognizer) longestCommonSubsequence(seq1, seq2 []string) int {
	m, n := len(seq1), len(seq2)
	dp := make([][]int, m+1)

	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if bsr.actionsMatch(seq1[i-1], seq2[j-1]) {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = int(math.Max(float64(dp[i-1][j]), float64(dp[i][j-1])))
			}
		}
	}

	return dp[m][n]
}

func (bsr *BasicSequenceRecognizer) actionsMatch(action1, action2 string) bool {
	// Direct match
	if action1 == action2 {
		return true
	}

	// Semantic equivalence
	semanticGroups := map[string][]string{
		"problem":      {"report_problem", "identify_error"},
		"analysis":     {"analyze_problem", "investigate_cause", "examine"},
		"solution":     {"propose_solution", "apply_fix", "implement_solution"},
		"verification": {"verify_solution", "confirm_resolution", "test"},
		"planning":     {"plan_approach", "define_objective"},
		"execution":    {"execute_steps", "perform", "implement"},
		"review":       {"review_results", "finalize_outcome", "assess"},
	}

	for _, group := range semanticGroups {
		contains1, contains2 := false, false
		for _, action := range group {
			if action == action1 {
				contains1 = true
			}
			if action == action2 {
				contains2 = true
			}
		}
		if contains1 && contains2 {
			return true
		}
	}

	return false
}

func (bsr *BasicSequenceRecognizer) hasStructuredPattern(actions []string) bool {
	// Check for alternating patterns, repeated sequences, or clear structure
	if len(actions) < 3 {
		return false
	}

	// Look for clear beginning, middle, end structure
	hasBeginning := bsr.isBeginningAction(actions[0])
	hasEnd := bsr.isEndingAction(actions[len(actions)-1])

	return hasBeginning && hasEnd
}

func (bsr *BasicSequenceRecognizer) isBeginningAction(action string) bool {
	beginningActions := []string{"report_problem", "define_objective", "plan_approach", "identify_error"}
	for _, beginAction := range beginningActions {
		if action == beginAction {
			return true
		}
	}
	return false
}

func (bsr *BasicSequenceRecognizer) isEndingAction(action string) bool {
	endingActions := []string{"verify_solution", "review_results", "finalize_outcome", "confirm_resolution"}
	for _, endAction := range endingActions {
		if action == endAction {
			return true
		}
	}
	return false
}

func (bsr *BasicSequenceRecognizer) classifySequence(actions []string) PatternType {
	text := strings.Join(actions, " ")

	if strings.Contains(text, "problem") || strings.Contains(text, "error") {
		return PatternTypeProblemSolution
	}
	if strings.Contains(text, "debug") || strings.Contains(text, "investigate") {
		return PatternTypeDebugging
	}

	return PatternTypeWorkflow
}

func (bsr *BasicSequenceRecognizer) generateSequenceName(sequenceType PatternType, _ []string) string {
	switch sequenceType {
	case PatternTypeProblemSolution:
		return "Problem Resolution Sequence"
	case PatternTypeDebugging:
		return "Debugging Sequence"
	case PatternTypeWorkflow:
		return "Task Workflow Sequence"
	case PatternTypeErrorResolution:
		return "Error Resolution Sequence"
	case PatternTypeCodeEvolution:
		return "Code Evolution Sequence"
	case PatternTypeDecisionMaking:
		return "Decision Making Sequence"
	case PatternTypeArchitectural:
		return "Architectural Design Sequence"
	case PatternTypeConfiguration:
		return "Configuration Sequence"
	case PatternTypeTesting:
		return "Testing Sequence"
	case PatternTypeRefactoring:
		return "Refactoring Sequence"
	default:
		// Unknown pattern types get a generic name
		return "General Sequence Pattern"
	}
}

func (bsr *BasicSequenceRecognizer) generateSequenceDescription(sequenceType PatternType, outcome PatternOutcome) string {
	baseDesc := fmt.Sprintf("A %s sequence", sequenceType)

	switch outcome {
	case OutcomeSuccess:
		return baseDesc + " that completed successfully"
	case OutcomeFailure:
		return baseDesc + " that encountered issues"
	case OutcomePartial:
		return baseDesc + " that was partially completed"
	case OutcomeInterrupted:
		return baseDesc + " that was interrupted"
	case OutcomeUnknown:
		return baseDesc + " with unknown outcome"
	default:
		// Unknown outcomes get a generic description
		return baseDesc + " with unknown outcome"
	}
}

func (bsr *BasicSequenceRecognizer) calculateSequenceConfidence(chunks []types.ConversationChunk, outcome PatternOutcome) float64 {
	baseConfidence := 0.5

	// Longer sequences generally have higher confidence
	if len(chunks) > 5 {
		baseConfidence += 0.2
	}

	// Successful outcomes increase confidence
	switch outcome {
	case OutcomeSuccess:
		baseConfidence += 0.3
	case OutcomeUnknown:
		// No confidence adjustment for unknown outcomes
	case OutcomePartial:
		baseConfidence += 0.1
	case OutcomeFailure:
		baseConfidence -= 0.1
	case OutcomeInterrupted:
		// Interrupted outcomes don't change confidence
	}

	return math.Min(baseConfidence, 1.0)
}

func (bsr *BasicSequenceRecognizer) calculateSuccessRateFromOutcome(outcome PatternOutcome) float64 {
	switch outcome {
	case OutcomeSuccess:
		return 1.0
	case OutcomePartial:
		return 0.6
	case OutcomeFailure:
		return 0.2
	case OutcomeInterrupted:
		return 0.4
	case OutcomeUnknown:
		return 0.5
	default:
		// Unknown outcomes get a neutral success rate
		return 0.5
	}
}

func (bsr *BasicSequenceRecognizer) extractSequenceKeywords(chunks []types.ConversationChunk) []string {
	text := extractText(chunks)
	words := strings.Fields(strings.ToLower(text))

	keywordCount := make(map[string]int)
	for _, word := range words {
		if len(word) > 3 && !isStopWord(word) {
			keywordCount[word]++
		}
	}

	var keywords []string
	for word, count := range keywordCount {
		if count > 1 { // Only include words that appear multiple times
			keywords = append(keywords, word)
		}
	}

	// Limit to top 10 keywords
	if len(keywords) > 10 {
		keywords = keywords[:10]
	}

	return keywords
}

func (bsr *BasicSequenceRecognizer) extractSequenceTriggers(chunks []types.ConversationChunk) []string {
	var triggers []string

	for i := range chunks {
		content := strings.ToLower(chunks[i].Content)

		if regexp.MustCompile(`(?i)(error|issue|problem)`).MatchString(content) {
			triggers = append(triggers, "problem_detected")
		}
		if regexp.MustCompile(`(?i)(help|assist|support)`).MatchString(content) {
			triggers = append(triggers, "help_requested")
		}
		if regexp.MustCompile(`(?i)(create|build|make)`).MatchString(content) {
			triggers = append(triggers, "creation_needed")
		}
	}

	return unique(triggers)
}

func (bsr *BasicSequenceRecognizer) extractSequenceOutcomes(chunks []types.ConversationChunk, outcome PatternOutcome) []string {
	outcomes := []string{string(outcome)}

	for i := range chunks {
		content := strings.ToLower(chunks[i].Content)

		if regexp.MustCompile(`(?i)(complete|done|finished|success)`).MatchString(content) {
			outcomes = append(outcomes, "task_completed")
		}
		if regexp.MustCompile(`(?i)(working|fixed|resolved)`).MatchString(content) {
			outcomes = append(outcomes, "issue_resolved")
		}
	}

	return unique(outcomes)
}

func (bsr *BasicSequenceRecognizer) convertActionsToSteps(actions []string, chunks []types.ConversationChunk) []PatternStep {
	steps := make([]PatternStep, 0, len(actions))

	for i, action := range actions {
		var context map[string]any
		if i < len(chunks) {
			context = map[string]any{
				"type":      string(chunks[i].Type),
				"length":    len(chunks[i].Content),
				"timestamp": chunks[i].Timestamp,
			}
		} else {
			context = make(map[string]any)
		}

		step := PatternStep{
			Order:       i,
			Action:      action,
			Description: bsr.generateActionDescription(action),
			Optional:    false,
			Confidence:  0.8,
			Context:     context,
		}

		steps = append(steps, step)
	}

	return steps
}

func (bsr *BasicSequenceRecognizer) generateActionDescription(action string) string {
	descriptions := map[string]string{
		"report_problem":      "Report or identify a problem",
		"analyze_problem":     "Analyze and investigate the problem",
		"propose_solution":    "Propose a solution or approach",
		"implement_solution":  "Implement the proposed solution",
		"verify_solution":     "Verify and test the solution",
		"plan_approach":       "Plan the approach or strategy",
		"execute_steps":       "Execute the planned steps",
		"review_results":      "Review and assess the results",
		"general_interaction": "General conversation interaction",
	}

	if desc, exists := descriptions[action]; exists {
		return desc
	}

	return "Perform action: " + action
}

func (bsr *BasicSequenceRecognizer) extractSequenceContext(chunks []types.ConversationChunk) map[string]any {
	context := make(map[string]any)

	context["sequence_length"] = len(chunks)
	context["has_code"] = bsr.containsCode(chunks)
	context["has_errors"] = bsr.containsErrors(chunks)
	context["role_distribution"] = bsr.calculateRoleDistribution(chunks)

	if len(chunks) > 0 {
		context["start_time"] = chunks[0].Timestamp
		context["end_time"] = chunks[len(chunks)-1].Timestamp
		context["duration"] = chunks[len(chunks)-1].Timestamp.Sub(chunks[0].Timestamp).Seconds()
	}

	return context
}

func (bsr *BasicSequenceRecognizer) containsCode(chunks []types.ConversationChunk) bool {
	for i := range chunks {
		if strings.Contains(chunks[i].Content, "```") {
			return true
		}
	}
	return false
}

func (bsr *BasicSequenceRecognizer) containsErrors(chunks []types.ConversationChunk) bool {
	for i := range chunks {
		if regexp.MustCompile(`(?i)(error|exception|fail)`).MatchString(chunks[i].Content) {
			return true
		}
	}
	return false
}

func (bsr *BasicSequenceRecognizer) calculateRoleDistribution(chunks []types.ConversationChunk) map[string]float64 {
	typeCount := make(map[string]int)
	total := len(chunks)

	for i := range chunks {
		typeCount[string(chunks[i].Type)]++
	}

	distribution := make(map[string]float64)
	for chunkType, count := range typeCount {
		distribution[chunkType] = float64(count) / float64(total)
	}

	return distribution
}
