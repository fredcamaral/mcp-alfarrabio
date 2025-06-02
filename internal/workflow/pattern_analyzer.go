// Package workflow provides intelligent analysis of Claude's tool usage patterns
package workflow

import (
	"fmt"
	"strings"
	"time"

	"mcp-memory/pkg/types"
)

// ToolUsage represents a single tool usage event
type ToolUsage struct {
	Tool      string                 `json:"tool"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context"`
	Success   bool                   `json:"success"`
	Duration  time.Duration          `json:"duration,omitempty"`
}

// ToolSequence represents a sequence of tool usages that led to an outcome
type ToolSequence struct {
	ID          string        `json:"id"`
	SessionID   string        `json:"session_id"`
	Repository  string        `json:"repository"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Tools       []ToolUsage   `json:"tools"`
	Outcome     types.Outcome `json:"outcome"`
	ProblemType string        `json:"problem_type"`
	Solution    string        `json:"solution"`
	Tags        []string      `json:"tags"`
}

// PatternType represents different types of problem-solving patterns
type PatternType string

const (
	PatternInvestigative PatternType = "investigative" // Read → Grep → Read → Edit
	PatternBuildFix      PatternType = "build_fix"     // Build → Error → Edit → Build
	PatternTestDriven    PatternType = "test_driven"   // Test → Edit → Test
	PatternExploration   PatternType = "exploration"   // Glob → Read → Grep → Read
	PatternConfiguration PatternType = "configuration" // Read config → Edit → Test
	PatternDebug         PatternType = "debug"         // Error → Search → Read → Fix

	// Repository constants
	UnknownRepository = "unknown"
)

// SuccessPattern represents a proven successful pattern
type SuccessPattern struct {
	Type        PatternType `json:"type"`
	Tools       []string    `json:"tools"`
	Description string      `json:"description"`
	Frequency   int         `json:"frequency"`
	SuccessRate float64     `json:"success_rate"`
	Examples    []string    `json:"examples"`
}

// PatternAnalyzer analyzes tool usage patterns and identifies successful sequences
type PatternAnalyzer struct {
	sequences       []ToolSequence
	successPatterns []SuccessPattern
	currentSequence *ToolSequence
	contextSwitches []ContextSwitch
}

// ContextSwitch represents when Claude switches between different tasks/repos
type ContextSwitch struct {
	Timestamp   time.Time `json:"timestamp"`
	FromRepo    string    `json:"from_repo"`
	ToRepo      string    `json:"to_repo"`
	TriggerTool string    `json:"trigger_tool"`
	Reason      string    `json:"reason"`
}

// NewPatternAnalyzer creates a new pattern analyzer
func NewPatternAnalyzer() *PatternAnalyzer {
	return &PatternAnalyzer{
		sequences:       make([]ToolSequence, 0),
		successPatterns: make([]SuccessPattern, 0),
		contextSwitches: make([]ContextSwitch, 0),
	}
}

// StartSequence begins tracking a new tool sequence
func (pa *PatternAnalyzer) StartSequence(sessionID, repository, problemType string) {
	pa.currentSequence = &ToolSequence{
		ID:          generateID(),
		SessionID:   sessionID,
		Repository:  repository,
		StartTime:   time.Now(),
		Tools:       make([]ToolUsage, 0),
		ProblemType: problemType,
		Tags:        make([]string, 0),
	}
}

// RecordToolUsage adds a tool usage to the current sequence
func (pa *PatternAnalyzer) RecordToolUsage(tool string, context map[string]interface{}, success bool) {
	if pa.currentSequence == nil {
		// Auto-start sequence if none exists
		pa.StartSequence("auto", UnknownRepository, UnknownRepository)
	}

	usage := ToolUsage{
		Tool:      tool,
		Timestamp: time.Now(),
		Context:   context,
		Success:   success,
	}

	pa.currentSequence.Tools = append(pa.currentSequence.Tools, usage)

	// Check for context switches
	pa.detectContextSwitch(usage)
}

// EndSequence completes the current sequence with an outcome
func (pa *PatternAnalyzer) EndSequence(outcome types.Outcome, solution string) {
	if pa.currentSequence == nil {
		return
	}

	pa.currentSequence.EndTime = time.Now()
	pa.currentSequence.Outcome = outcome
	pa.currentSequence.Solution = solution
	pa.currentSequence.Tags = pa.extractSequenceTags(*pa.currentSequence)

	// Add to completed sequences
	pa.sequences = append(pa.sequences, *pa.currentSequence)

	// Analyze for patterns if successful
	if outcome == types.OutcomeSuccess {
		pa.analyzeSuccessfulSequence(*pa.currentSequence)
	}

	pa.currentSequence = nil
}

// DetectPatternType identifies the type of problem-solving pattern
func (pa *PatternAnalyzer) DetectPatternType(tools []string) PatternType {
	// Define pattern signatures as sequences
	patterns := map[PatternType][][]string{
		PatternInvestigative: {
			{"Read", "Grep", "Read", "Edit"},
			{"Glob", "Read", "Grep", "Edit"},
			{"LS", "Read", "Grep", "Edit"},
		},
		PatternBuildFix: {
			{"Bash", "Read", "Edit", "Bash"},
			{"Build", "Error", "Edit", "Build"},
		},
		PatternTestDriven: {
			{"Test", "Edit", "Test"},
			{"Bash", "Edit", "Bash"}, // test via bash
		},
		PatternExploration: {
			{"Glob", "Read", "Grep", "Read"},
			{"LS", "Read", "LS", "Read"},
		},
		PatternConfiguration: {
			{"Read", "Edit", "Bash"}, // Read config, edit, test
			{"Read", "Edit", "Test"},
		},
		PatternDebug: {
			{"Error", "Grep", "Read", "Edit"},
			{"Bash", "Grep", "Read", "Edit"},
		},
	}

	// Find best matching pattern
	bestMatch := PatternExploration // default
	maxScore := 0

	for patternType, signatures := range patterns {
		for _, signature := range signatures {
			score := pa.calculatePatternMatch(tools, signature)
			if score > maxScore {
				maxScore = score
				bestMatch = patternType
			}
		}
	}

	return bestMatch
}

// calculatePatternMatch calculates how well a tool sequence matches a pattern
func (pa *PatternAnalyzer) calculatePatternMatch(tools, pattern []string) int {
	maxConsecutiveMatches := 0

	// Case 1: tools is longer or equal - look for pattern as subsequence
	if len(tools) >= len(pattern) {
		for i := 0; i <= len(tools)-len(pattern); i++ {
			consecutiveMatches := 0
			for j := 0; j < len(pattern); j++ {
				if tools[i+j] == pattern[j] {
					consecutiveMatches++
				} else {
					break
				}
			}
			if consecutiveMatches > maxConsecutiveMatches {
				maxConsecutiveMatches = consecutiveMatches
			}
		}
	}

	// Case 2: tools is shorter - check if tools is prefix of pattern
	if len(tools) < len(pattern) {
		consecutiveMatches := 0
		for i := 0; i < len(tools); i++ {
			if tools[i] == pattern[i] {
				consecutiveMatches++
			} else {
				break
			}
		}
		if consecutiveMatches > maxConsecutiveMatches {
			maxConsecutiveMatches = consecutiveMatches
		}
	}

	return maxConsecutiveMatches
}

// analyzeSuccessfulSequence extracts patterns from successful sequences
func (pa *PatternAnalyzer) analyzeSuccessfulSequence(sequence ToolSequence) {
	tools := make([]string, len(sequence.Tools))
	for i, tool := range sequence.Tools {
		tools[i] = tool.Tool
	}

	patternType := pa.DetectPatternType(tools)

	// Update or create success pattern
	pa.updateSuccessPattern(patternType, tools, sequence.Solution)
}

// updateSuccessPattern updates success pattern statistics
func (pa *PatternAnalyzer) updateSuccessPattern(patternType PatternType, tools []string, solution string) {
	// Find existing pattern
	for i := range pa.successPatterns {
		if pa.successPatterns[i].Type == patternType {
			pa.successPatterns[i].Frequency++
			if len(pa.successPatterns[i].Examples) < 5 {
				pa.successPatterns[i].Examples = append(pa.successPatterns[i].Examples, solution)
			}
			pa.recalculateSuccessRate(&pa.successPatterns[i])
			return
		}
	}

	// Create new pattern
	pattern := SuccessPattern{
		Type:        patternType,
		Tools:       tools,
		Description: pa.generatePatternDescription(patternType, tools),
		Frequency:   1,
		SuccessRate: 1.0, // Start optimistic
		Examples:    []string{solution},
	}

	pa.successPatterns = append(pa.successPatterns, pattern)
}

// generatePatternDescription creates a human-readable description
func (pa *PatternAnalyzer) generatePatternDescription(patternType PatternType, tools []string) string {
	switch patternType {
	case PatternInvestigative:
		return fmt.Sprintf("Investigative approach: %s", strings.Join(tools, " → "))
	case PatternBuildFix:
		return fmt.Sprintf("Build-fix cycle: %s", strings.Join(tools, " → "))
	case PatternTestDriven:
		return fmt.Sprintf("Test-driven development: %s", strings.Join(tools, " → "))
	case PatternExploration:
		return fmt.Sprintf("Code exploration: %s", strings.Join(tools, " → "))
	case PatternConfiguration:
		return fmt.Sprintf("Configuration management: %s", strings.Join(tools, " → "))
	case PatternDebug:
		return fmt.Sprintf("Debugging workflow: %s", strings.Join(tools, " → "))
	default:
		return fmt.Sprintf("Pattern: %s", strings.Join(tools, " → "))
	}
}

// recalculateSuccessRate updates success rate based on all sequences
func (pa *PatternAnalyzer) recalculateSuccessRate(pattern *SuccessPattern) {
	total := 0
	successful := 0

	for _, sequence := range pa.sequences {
		tools := make([]string, len(sequence.Tools))
		for i, tool := range sequence.Tools {
			tools[i] = tool.Tool
		}

		if pa.DetectPatternType(tools) == pattern.Type {
			total++
			if sequence.Outcome == types.OutcomeSuccess {
				successful++
			}
		}
	}

	if total > 0 {
		pattern.SuccessRate = float64(successful) / float64(total)
	}
}

// detectContextSwitch identifies when Claude switches contexts
func (pa *PatternAnalyzer) detectContextSwitch(usage ToolUsage) {
	if len(pa.sequences) == 0 {
		return
	}

	lastSequence := pa.sequences[len(pa.sequences)-1]
	currentRepo := pa.currentSequence.Repository

	// Check for repository change
	if lastSequence.Repository != currentRepo && currentRepo != UnknownRepository {
		contextSwitch := ContextSwitch{
			Timestamp:   usage.Timestamp,
			FromRepo:    lastSequence.Repository,
			ToRepo:      currentRepo,
			TriggerTool: usage.Tool,
			Reason:      pa.inferSwitchReason(usage),
		}

		pa.contextSwitches = append(pa.contextSwitches, contextSwitch)
	}
}

// inferSwitchReason tries to understand why Claude switched contexts
func (pa *PatternAnalyzer) inferSwitchReason(usage ToolUsage) string {
	switch usage.Tool {
	case "LS", "Glob":
		return "exploring new codebase"
	case "Read":
		if filepath, exists := usage.Context["file_path"]; exists {
			if strings.Contains(fmt.Sprintf("%v", filepath), "README") {
				return "reading project documentation"
			}
		}
		return "investigating files"
	case "Bash":
		if cmd, exists := usage.Context["command"]; exists {
			cmdStr := fmt.Sprintf("%v", cmd)
			if strings.Contains(cmdStr, "cd") {
				return "changing directories"
			}
			if strings.Contains(cmdStr, "git") {
				return "git operations"
			}
		}
		return "running commands"
	default:
		return "task switching"
	}
}

// extractSequenceTags generates relevant tags for a sequence
func (pa *PatternAnalyzer) extractSequenceTags(sequence ToolSequence) []string {
	tags := make([]string, 0)

	// Add outcome tag
	tags = append(tags, string(sequence.Outcome))

	// Add repository tag
	if sequence.Repository != UnknownRepository {
		tags = append(tags, fmt.Sprintf("repo-%s", sequence.Repository))
	}

	// Add pattern type tag
	tools := make([]string, len(sequence.Tools))
	for i, tool := range sequence.Tools {
		tools[i] = tool.Tool
	}
	patternType := pa.DetectPatternType(tools)
	tags = append(tags, string(patternType))

	// Add duration tag
	duration := sequence.EndTime.Sub(sequence.StartTime)
	switch {
	case duration > 30*time.Minute:
		tags = append(tags, "long-session")
	case duration > 10*time.Minute:
		tags = append(tags, "medium-session")
	default:
		tags = append(tags, "short-session")
	}

	// Add tool-specific tags
	toolCounts := make(map[string]int)
	for _, tool := range sequence.Tools {
		toolCounts[tool.Tool]++
	}

	for tool, count := range toolCounts {
		if count > 3 {
			tags = append(tags, fmt.Sprintf("heavy-%s", strings.ToLower(tool)))
		}
	}

	return tags
}

// GetSuccessPatterns returns all identified success patterns
func (pa *PatternAnalyzer) GetSuccessPatterns() []SuccessPattern {
	return pa.successPatterns
}

// GetSequences returns all recorded sequences
func (pa *PatternAnalyzer) GetSequences() []ToolSequence {
	return pa.sequences
}

// GetContextSwitches returns all detected context switches
func (pa *PatternAnalyzer) GetContextSwitches() []ContextSwitch {
	return pa.contextSwitches
}

// GetPatternRecommendations suggests patterns based on current context
func (pa *PatternAnalyzer) GetPatternRecommendations(currentTools []string, problemType string) []SuccessPattern {
	recommendations := make([]SuccessPattern, 0)

	// Find patterns with high success rates
	for _, pattern := range pa.successPatterns {
		if pattern.SuccessRate > 0.7 && pattern.Frequency > 2 {
			// Check if current tools partially match this pattern
			matchScore := pa.calculatePatternMatch(currentTools, pattern.Tools)
			if matchScore > 0 {
				recommendations = append(recommendations, pattern)
			}
		}
	}

	return recommendations
}

// generateID creates a unique ID for sequences
func generateID() string {
	return fmt.Sprintf("seq-%d", time.Now().UnixNano())
}
