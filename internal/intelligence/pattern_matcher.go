package intelligence

import (
	"math"
	"regexp"
	"strings"

	"mcp-memory/pkg/types"
)

// BasicPatternMatcher implements the PatternMatcher interface
type BasicPatternMatcher struct {
	// Compiled regexes for feature extraction
	entityRegex    *regexp.Regexp
	actionRegex    *regexp.Regexp
	intentRegex    *regexp.Regexp
	outcomeRegex   *regexp.Regexp
}

// NewBasicPatternMatcher creates a new basic pattern matcher
func NewBasicPatternMatcher() *BasicPatternMatcher {
	return &BasicPatternMatcher{
		entityRegex:  regexp.MustCompile(`(?i)(file|function|class|variable|api|service|database|config)`),
		actionRegex:  regexp.MustCompile(`(?i)(create|delete|update|fix|install|build|test|deploy|refactor)`),
		intentRegex:  regexp.MustCompile(`(?i)(want to|need to|trying to|going to|should|will|can)`),
		outcomeRegex: regexp.MustCompile(`(?i)(success|fail|error|complete|done|working|broken)`),
	}
}

// MatchPattern calculates how well chunks match a given pattern
func (bpm *BasicPatternMatcher) MatchPattern(chunks []types.ConversationChunk, pattern Pattern) float64 {
	if len(chunks) == 0 {
		return 0.0
	}
	
	features := bpm.ExtractFeatures(chunks)
	sequence := bpm.IdentifySequence(chunks)
	
	// Calculate different matching scores
	keywordScore := bpm.calculateKeywordMatch(features, pattern)
	sequenceScore := bpm.calculateSequenceMatch(sequence, pattern.Steps)
	contextScore := bpm.calculateContextMatch(features, pattern.Context)
	typeScore := bpm.calculateTypeMatch(features, pattern.Type)
	
	// Weighted combination
	overallScore := (keywordScore*0.3 + sequenceScore*0.3 + contextScore*0.2 + typeScore*0.2)
	
	return math.Min(overallScore, 1.0)
}

// ExtractFeatures extracts relevant features from conversation chunks
func (bpm *BasicPatternMatcher) ExtractFeatures(chunks []types.ConversationChunk) map[string]any {
	features := make(map[string]any)
	
	text := extractText(chunks)
	
	// Basic features
	features["chunk_count"] = len(chunks)
	features["total_length"] = len(text)
	features["avg_chunk_length"] = float64(len(text)) / float64(len(chunks))
	features["conversation_duration"] = bpm.calculateDuration(chunks)
	
	// Pattern-specific features
	features["entities"] = bpm.extractEntities(text)
	features["actions"] = bpm.extractActions(text)
	features["intents"] = bpm.extractIntents(text)
	features["outcomes"] = bpm.extractOutcomes(text)
	
	// Technical features
	features["has_code"] = strings.Contains(text, "```")
	features["has_errors"] = regexp.MustCompile(`(?i)(error|exception|fail)`).MatchString(text)
	features["has_commands"] = regexp.MustCompile(`(?i)(run|execute|install)`).MatchString(text)
	features["has_files"] = regexp.MustCompile(`\.(go|js|py|java|cpp|h)(\s|$)`).MatchString(text)
	
	// Sentiment and urgency
	features["urgency"] = bpm.calculateUrgency(text)
	features["complexity"] = bpm.calculateComplexity(text)
	features["question_count"] = strings.Count(text, "?")
	features["exclamation_count"] = strings.Count(text, "!")
	
	// Role distribution
	roleDistribution := bpm.calculateRoleDistribution(chunks)
	features["role_distribution"] = roleDistribution
	
	return features
}

// IdentifySequence identifies the sequence of steps in the conversation
func (bpm *BasicPatternMatcher) IdentifySequence(chunks []types.ConversationChunk) []PatternStep {
	var steps []PatternStep
	
	for i, chunk := range chunks {
		action := bpm.identifyAction(chunk.Content)
		confidence := bpm.calculateStepConfidence(chunk, chunks)
		
		step := PatternStep{
			Order:       i,
			Action:      action,
			Description: bpm.generateStepDescription(action, chunk.Content),
			Optional:    false,
			Confidence:  confidence,
			Context:     bpm.extractStepContext(chunk),
		}
		
		steps = append(steps, step)
	}
	
	return steps
}

// Helper methods for pattern matching

func (bpm *BasicPatternMatcher) calculateKeywordMatch(features map[string]any, pattern Pattern) float64 {
	// Extract keywords from features
	currentKeywords := make([]string, 0)
	
	if entities, ok := features["entities"].([]string); ok {
		currentKeywords = append(currentKeywords, entities...)
	}
	if actions, ok := features["actions"].([]string); ok {
		currentKeywords = append(currentKeywords, actions...)
	}
	
	return calculateOverlap(currentKeywords, pattern.Keywords)
}

func (bpm *BasicPatternMatcher) calculateSequenceMatch(currentSequence []PatternStep, patternSteps []PatternStep) float64 {
	if len(currentSequence) == 0 || len(patternSteps) == 0 {
		return 0.0
	}
	
	// Calculate similarity between action sequences
	matches := 0
	minLength := int(math.Min(float64(len(currentSequence)), float64(len(patternSteps))))
	
	for i := 0; i < minLength; i++ {
			if bpm.actionsMatch(currentSequence[i].Action, patternSteps[i].Action) {
			matches++
		}
	}
	
	return float64(matches) / float64(math.Max(float64(len(currentSequence)), float64(len(patternSteps))))
}

func (bpm *BasicPatternMatcher) calculateContextMatch(currentFeatures map[string]any, patternContext map[string]any) float64 {
	if len(patternContext) == 0 {
		return 0.5 // Neutral score for patterns without context
	}
	
	matches := 0
	total := len(patternContext)
	
	for key, patternValue := range patternContext {
		if currentValue, exists := currentFeatures[key]; exists {
				if bpm.valuesMatch(currentValue, patternValue) {
				matches++
			}
		}
	}
	
	return float64(matches) / float64(total)
}

func (bpm *BasicPatternMatcher) calculateTypeMatch(features map[string]any, patternType PatternType) float64 {
	hasCode := features["has_code"].(bool)
	hasErrors := features["has_errors"].(bool)
	hasCommands := features["has_commands"].(bool)
	
	switch patternType {
	case PatternTypeProblemSolution:
		if hasErrors {
			return 0.8
		}
		return 0.4
	case PatternTypeErrorResolution:
		if hasErrors {
			return 0.9
		}
		return 0.2
	case PatternTypeCodeEvolution:
		if hasCode {
			return 0.8
		}
		return 0.3
	case PatternTypeWorkflow:
		if hasCommands {
			return 0.7
		}
		return 0.5
	default:
		return 0.5
	}
}

// Feature extraction methods

func (bpm *BasicPatternMatcher) extractEntities(text string) []string {
	matches := bpm.entityRegex.FindAllString(text, -1)
	return unique(matches)
}

func (bpm *BasicPatternMatcher) extractActions(text string) []string {
	matches := bpm.actionRegex.FindAllString(text, -1)
	return unique(matches)
}

func (bpm *BasicPatternMatcher) extractIntents(text string) []string {
	matches := bpm.intentRegex.FindAllString(text, -1)
	return unique(matches)
}

func (bpm *BasicPatternMatcher) extractOutcomes(text string) []string {
	matches := bpm.outcomeRegex.FindAllString(text, -1)
	return unique(matches)
}

func (bpm *BasicPatternMatcher) calculateDuration(chunks []types.ConversationChunk) float64 {
	if len(chunks) < 2 {
		return 0.0
	}
	
	start := chunks[0].Timestamp
	end := chunks[len(chunks)-1].Timestamp
	
	return end.Sub(start).Seconds()
}

func (bpm *BasicPatternMatcher) calculateUrgency(text string) float64 {
	urgentWords := []string{"urgent", "emergency", "asap", "immediately", "critical", "blocking"}
	urgencyScore := 0.0
	
	lowerText := strings.ToLower(text)
	for _, word := range urgentWords {
		if strings.Contains(lowerText, word) {
			urgencyScore += 0.2
		}
	}
	
	// Account for multiple exclamation marks
	exclamations := strings.Count(text, "!")
	urgencyScore += math.Min(float64(exclamations)*0.1, 0.3)
	
	return math.Min(urgencyScore, 1.0)
}

func (bpm *BasicPatternMatcher) calculateComplexity(text string) float64 {
	// Simple complexity heuristics
	complexity := 0.0
	
	// Length contributes to complexity
	complexity += math.Min(float64(len(text))/1000.0, 0.5)
	
	// Code blocks increase complexity
	codeBlocks := strings.Count(text, "```")
	complexity += math.Min(float64(codeBlocks)*0.2, 0.3)
	
	// Technical terms increase complexity
	techTerms := []string{"api", "database", "algorithm", "architecture", "framework"}
	lowerText := strings.ToLower(text)
	for _, term := range techTerms {
		if strings.Contains(lowerText, term) {
			complexity += 0.1
		}
	}
	
	return math.Min(complexity, 1.0)
}

func (bpm *BasicPatternMatcher) calculateRoleDistribution(chunks []types.ConversationChunk) map[string]float64 {
	typeCount := make(map[string]int)
	total := len(chunks)
	
	for _, chunk := range chunks {
		typeCount[string(chunk.Type)]++
	}
	
	distribution := make(map[string]float64)
	for chunkType, count := range typeCount {
		distribution[chunkType] = float64(count) / float64(total)
	}
	
	return distribution
}

func (bpm *BasicPatternMatcher) identifyAction(content string) string {
	content = strings.ToLower(content)
	
	if strings.Contains(content, "error") || strings.Contains(content, "fail") {
		return "report_problem"
	}
	if strings.Contains(content, "fix") || strings.Contains(content, "solve") {
		return "provide_solution"
	}
	if strings.Contains(content, "create") || strings.Contains(content, "add") {
		return "create_resource"
	}
	if strings.Contains(content, "update") || strings.Contains(content, "modify") {
		return "modify_resource"
	}
	if strings.Contains(content, "test") || strings.Contains(content, "verify") {
		return "verify_solution"
	}
	if strings.Contains(content, "run") || strings.Contains(content, "execute") {
		return "execute_command"
	}
	
	return "general_interaction"
}

func (bpm *BasicPatternMatcher) calculateStepConfidence(chunk types.ConversationChunk, _ []types.ConversationChunk) float64 {
	// Base confidence on chunk properties
	baseConfidence := 0.5
	
	// Longer chunks generally have higher confidence
	if len(chunk.Content) > 100 {
		baseConfidence += 0.2
	}
	
	// Chunks with specific patterns have higher confidence
	if regexp.MustCompile(`(?i)(step|then|next|after)`).MatchString(chunk.Content) {
		baseConfidence += 0.2
	}
	
	// Code or technical content has higher confidence
	if strings.Contains(chunk.Content, "```") {
		baseConfidence += 0.1
	}
	
	return math.Min(baseConfidence, 1.0)
}

func (bpm *BasicPatternMatcher) generateStepDescription(action, _ string) string {
	switch action {
	case "report_problem":
		return "User reports an issue or problem"
	case "provide_solution":
		return "Assistant provides a solution or fix"
	case "create_resource":
		return "Create or add a new resource"
	case "modify_resource":
		return "Modify or update existing resource"
	case "verify_solution":
		return "Test or verify the proposed solution"
	case "execute_command":
		return "Execute a command or operation"
	default:
		return "General conversation interaction"
	}
}

func (bpm *BasicPatternMatcher) extractStepContext(chunk types.ConversationChunk) map[string]any {
	context := make(map[string]any)
	
	context["type"] = string(chunk.Type)
	context["length"] = len(chunk.Content)
	context["has_code"] = strings.Contains(chunk.Content, "```")
	context["timestamp"] = chunk.Timestamp
	
	return context
}

func (bpm *BasicPatternMatcher) actionsMatch(action1, action2 string) bool {
	// Exact match
	if action1 == action2 {
		return true
	}
	
	// Semantic similarity for actions
	semanticGroups := map[string][]string{
		"problem": {"report_problem", "identify_issue"},
		"solution": {"provide_solution", "fix_issue", "resolve_problem"},
		"creation": {"create_resource", "add_resource", "build_resource"},
		"modification": {"modify_resource", "update_resource", "change_resource"},
		"verification": {"verify_solution", "test_solution", "validate_solution"},
		"execution": {"execute_command", "run_command", "perform_action"},
	}
	
	for _, group := range semanticGroups {
		contains1 := false
		contains2 := false
		
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

func (bpm *BasicPatternMatcher) valuesMatch(value1, value2 any) bool {
	// Type-specific matching logic
	switch v1 := value1.(type) {
	case string:
		if v2, ok := value2.(string); ok {
			return strings.EqualFold(v1, v2)
		}
	case bool:
		if v2, ok := value2.(bool); ok {
			return v1 == v2
		}
	case float64:
		if v2, ok := value2.(float64); ok {
			return math.Abs(v1-v2) < 0.1 // Allow small differences
		}
	case int:
		if v2, ok := value2.(int); ok {
			return v1 == v2
		}
	}
	
	return false
}