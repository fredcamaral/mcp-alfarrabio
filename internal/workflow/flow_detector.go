// Package workflow provides intelligent conversation flow detection and analysis
package workflow

import (
	"regexp"
	"strings"
	"time"

	"mcp-memory/pkg/types"
)

// ConversationSegment represents a segment of conversation with identified flow type
type ConversationSegment struct {
	ID         string                 `json:"id"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
	Flow       types.ConversationFlow `json:"flow"`
	Content    string                 `json:"content"`
	Confidence float64                `json:"confidence"` // 0.0 to 1.0
	Keywords   []string               `json:"keywords"`
	Entities   []string               `json:"entities"` // File names, error codes, etc.
}

// FlowTransition represents a transition between conversation flows
type FlowTransition struct {
	From      types.ConversationFlow `json:"from"`
	To        types.ConversationFlow `json:"to"`
	Timestamp time.Time              `json:"timestamp"`
	Trigger   string                 `json:"trigger"`
	Context   map[string]interface{} `json:"context"`
}

// ConversationSession represents a complete conversation session with flow analysis
type ConversationSession struct {
	SessionID   string                `json:"session_id"`
	Repository  string                `json:"repository"`
	StartTime   time.Time             `json:"start_time"`
	EndTime     *time.Time            `json:"end_time,omitempty"`
	Segments    []ConversationSegment `json:"segments"`
	Transitions []FlowTransition      `json:"transitions"`
	Summary     SessionSummary        `json:"summary"`
}

// SessionSummary provides high-level analysis of the conversation session
type SessionSummary struct {
	TotalDuration      time.Duration          `json:"total_duration"`
	PrimaryFlow        types.ConversationFlow `json:"primary_flow"`
	ProblemsSolved     int                    `json:"problems_solved"`
	InvestigationTime  time.Duration          `json:"investigation_time"`
	SolutionTime       time.Duration          `json:"solution_time"`
	VerificationTime   time.Duration          `json:"verification_time"`
	SuccessfulOutcomes int                    `json:"successful_outcomes"`
	Technologies       []string               `json:"technologies"`
	KeyDecisions       []string               `json:"key_decisions"`
}

// FlowDetector analyzes conversation content to identify flow patterns
type FlowDetector struct {
	sessions        map[string]*ConversationSession
	currentSegment  *ConversationSegment
	flowPatterns    map[types.ConversationFlow]*FlowPattern
	entityExtractor *EntityExtractor
}

// FlowPattern defines patterns for detecting conversation flows
type FlowPattern struct {
	Keywords  []string         `json:"keywords"`
	Phrases   []string         `json:"phrases"`
	Regex     []*regexp.Regexp `json:"-"`
	ToolUsage []string         `json:"tool_usage"`
	Weight    float64          `json:"weight"`
}

// EntityExtractor identifies entities like file names, error codes, etc.
type EntityExtractor struct {
	filePattern    *regexp.Regexp
	errorPattern   *regexp.Regexp
	commandPattern *regexp.Regexp
	urlPattern     *regexp.Regexp
}

// NewFlowDetector creates a new conversation flow detector
func NewFlowDetector() *FlowDetector {
	detector := &FlowDetector{
		sessions:        make(map[string]*ConversationSession),
		flowPatterns:    make(map[types.ConversationFlow]*FlowPattern),
		entityExtractor: NewEntityExtractor(),
	}

	detector.initializePatterns()
	return detector
}

// NewEntityExtractor creates a new entity extractor
func NewEntityExtractor() *EntityExtractor {
	return &EntityExtractor{
		filePattern:    regexp.MustCompile(`[\w\-_/]+\.\w+`),
		errorPattern:   regexp.MustCompile(`(?i)(error|exception|failed|failure):?\s*([A-Z_][A-Z0-9_]*|\w+)`),
		commandPattern: regexp.MustCompile(`[\$>]\s*([a-zA-Z][a-zA-Z0-9\-_]*(?:\s+[^\n]*)?)`),
		urlPattern:     regexp.MustCompile(`https?://[^\s]+`),
	}
}

// initializePatterns sets up the flow detection patterns
func (fd *FlowDetector) initializePatterns() {
	// Problem flow patterns
	fd.flowPatterns[types.FlowProblem] = &FlowPattern{
		Keywords:  []string{"error", "issue", "problem", "bug", "broken", "failed", "failing", "exception", "crash"},
		Phrases:   []string{"I'm getting", "there's an issue", "something's wrong", "not working", "help me with"},
		ToolUsage: []string{"Read", "LS", "Grep"},
		Weight:    1.0,
	}

	// Investigation flow patterns
	fd.flowPatterns[types.FlowInvestigation] = &FlowPattern{
		Keywords:  []string{"let me check", "investigating", "looking at", "examining", "analyzing", "debugging"},
		Phrases:   []string{"let me search", "I'll look for", "checking the", "let me examine", "investigating this"},
		ToolUsage: []string{"Grep", "Read", "Bash", "Glob"},
		Weight:    1.0,
	}

	// Solution flow patterns
	fd.flowPatterns[types.FlowSolution] = &FlowPattern{
		Keywords:  []string{"fix", "solution", "resolve", "implement", "create", "add", "update", "modify"},
		Phrases:   []string{"here's the fix", "I'll implement", "let me create", "the solution is", "to fix this"},
		ToolUsage: []string{"Edit", "Write", "MultiEdit"},
		Weight:    1.0,
	}

	// Verification flow patterns
	fd.flowPatterns[types.FlowVerification] = &FlowPattern{
		Keywords:  []string{"test", "verify", "check", "confirm", "validate", "build", "run", "compile"},
		Phrases:   []string{"let's test", "running the", "checking if", "verifying that", "testing the fix"},
		ToolUsage: []string{"Bash", "Test"},
		Weight:    1.0,
	}

	// Compile regex patterns
	for _, pattern := range fd.flowPatterns {
		for _, phrase := range pattern.Phrases {
			regex, err := regexp.Compile(`(?i)` + regexp.QuoteMeta(phrase))
			if err == nil {
				pattern.Regex = append(pattern.Regex, regex)
			}
		}
	}
}

// StartSession begins tracking a new conversation session
func (fd *FlowDetector) StartSession(sessionID, repository string) {
	session := &ConversationSession{
		SessionID:   sessionID,
		Repository:  repository,
		StartTime:   time.Now(),
		Segments:    make([]ConversationSegment, 0),
		Transitions: make([]FlowTransition, 0),
	}

	fd.sessions[sessionID] = session
}

// ProcessMessage analyzes a conversation message and updates flow state
func (fd *FlowDetector) ProcessMessage(sessionID, content string, toolUsed string, context map[string]interface{}) {
	session := fd.getOrCreateSession(sessionID)

	// Detect flow type for this content
	detectedFlow, confidence := fd.detectFlow(content, toolUsed)

	// Check if we need to create a new segment or continue current one
	if fd.shouldCreateNewSegment(detectedFlow, confidence) {
		fd.finishCurrentSegment(session)
		fd.startNewSegment(session, detectedFlow, content, confidence)
	} else {
		fd.updateCurrentSegment(content, confidence)
	}

	// Extract entities from content
	entities := fd.entityExtractor.Extract(content)
	if fd.currentSegment != nil {
		fd.currentSegment.Entities = append(fd.currentSegment.Entities, entities...)
	}
}

// detectFlow analyzes content to determine conversation flow type
func (fd *FlowDetector) detectFlow(content, toolUsed string) (types.ConversationFlow, float64) {
	contentLower := strings.ToLower(content)
	maxScore := 0.0
	detectedFlow := types.FlowProblem // default

	for flow, pattern := range fd.flowPatterns {
		score := fd.calculateFlowScore(contentLower, toolUsed, pattern)
		if score > maxScore {
			maxScore = score
			detectedFlow = flow
		}
	}

	// Normalize confidence to 0-1 range
	confidence := maxScore / 3.0 // Max possible score is roughly 3 (keyword + phrase + tool)
	if confidence > 1.0 {
		confidence = 1.0
	}

	return detectedFlow, confidence
}

// calculateFlowScore computes a score for how well content matches a flow pattern
func (fd *FlowDetector) calculateFlowScore(content, toolUsed string, pattern *FlowPattern) float64 {
	score := 0.0

	// Check keywords
	for _, keyword := range pattern.Keywords {
		if strings.Contains(content, keyword) {
			score += 0.5
		}
	}

	// Check regex phrases
	for _, regex := range pattern.Regex {
		if regex.MatchString(content) {
			score += 1.0
		}
	}

	// Check tool usage
	for _, tool := range pattern.ToolUsage {
		if tool == toolUsed {
			score += 1.0
			break
		}
	}

	return score * pattern.Weight
}

// shouldCreateNewSegment determines if we should start a new conversation segment
func (fd *FlowDetector) shouldCreateNewSegment(flow types.ConversationFlow, confidence float64) bool {
	// Always create first segment
	if fd.currentSegment == nil {
		return true
	}

	// Create new segment if flow changed and confidence is high
	if fd.currentSegment.Flow != flow && confidence > 0.6 {
		return true
	}

	// Create new segment if current segment is getting too long
	if time.Since(fd.currentSegment.StartTime) > 10*time.Minute {
		return true
	}

	return false
}

// finishCurrentSegment completes the current segment and adds it to session
func (fd *FlowDetector) finishCurrentSegment(session *ConversationSession) {
	if fd.currentSegment == nil {
		return
	}

	fd.currentSegment.EndTime = time.Now()
	fd.currentSegment.Keywords = fd.extractKeywords(fd.currentSegment.Content)

	// Record transition if there was a previous segment
	if len(session.Segments) > 0 {
		lastFlow := session.Segments[len(session.Segments)-1].Flow
		if lastFlow != fd.currentSegment.Flow {
			transition := FlowTransition{
				From:      lastFlow,
				To:        fd.currentSegment.Flow,
				Timestamp: fd.currentSegment.StartTime,
				Trigger:   fd.inferTransitionTrigger(lastFlow, fd.currentSegment.Flow),
			}
			session.Transitions = append(session.Transitions, transition)
		}
	}

	session.Segments = append(session.Segments, *fd.currentSegment)
	fd.currentSegment = nil
}

// startNewSegment begins a new conversation segment
func (fd *FlowDetector) startNewSegment(_ *ConversationSession, flow types.ConversationFlow, content string, confidence float64) {
	fd.currentSegment = &ConversationSegment{
		ID:         generateSegmentID(),
		StartTime:  time.Now(),
		Flow:       flow,
		Content:    content,
		Confidence: confidence,
		Keywords:   make([]string, 0),
		Entities:   make([]string, 0),
	}
}

// updateCurrentSegment adds content to the current segment
func (fd *FlowDetector) updateCurrentSegment(content string, confidence float64) {
	if fd.currentSegment == nil {
		return
	}

	fd.currentSegment.Content += "\n" + content
	// Update confidence as weighted average
	totalContent := len(strings.Split(fd.currentSegment.Content, "\n"))
	fd.currentSegment.Confidence = (fd.currentSegment.Confidence*float64(totalContent-1) + confidence) / float64(totalContent)
}

// EndSession completes the conversation session and generates summary
func (fd *FlowDetector) EndSession(sessionID string, outcome types.Outcome) {
	session := fd.sessions[sessionID]
	if session == nil {
		return
	}

	// Finish any current segment
	fd.finishCurrentSegment(session)

	// Mark session as ended
	now := time.Now()
	session.EndTime = &now

	// Generate session summary
	session.Summary = fd.generateSessionSummary(session)

	// Clean up current segment
	fd.currentSegment = nil
}

// generateSessionSummary creates a comprehensive summary of the session
func (fd *FlowDetector) generateSessionSummary(session *ConversationSession) SessionSummary {
	summary := SessionSummary{
		Technologies: make([]string, 0),
		KeyDecisions: make([]string, 0),
	}

	if session.EndTime != nil {
		summary.TotalDuration = session.EndTime.Sub(session.StartTime)
	}

	// Analyze segments for timing and flow distribution
	flowDurations := make(map[types.ConversationFlow]time.Duration)
	flowCounts := make(map[types.ConversationFlow]int)

	for _, segment := range session.Segments {
		duration := segment.EndTime.Sub(segment.StartTime)
		flowDurations[segment.Flow] += duration
		flowCounts[segment.Flow]++
	}

	// Find primary flow (most time spent)
	maxDuration := time.Duration(0)
	for flow, duration := range flowDurations {
		if duration > maxDuration {
			maxDuration = duration
			summary.PrimaryFlow = flow
		}
	}

	// Set specific timing
	summary.InvestigationTime = flowDurations[types.FlowInvestigation]
	summary.SolutionTime = flowDurations[types.FlowSolution]
	summary.VerificationTime = flowDurations[types.FlowVerification]

	// Count problems solved (transitions from problem to solution)
	for _, transition := range session.Transitions {
		if transition.From == types.FlowProblem && transition.To == types.FlowSolution {
			summary.ProblemsSolved++
		}
		if transition.To == types.FlowVerification {
			summary.SuccessfulOutcomes++
		}
	}

	// Extract technologies from entities
	summary.Technologies = fd.extractTechnologies(session)
	summary.KeyDecisions = fd.extractDecisions(session)

	return summary
}

// extractKeywords extracts important keywords from content
func (fd *FlowDetector) extractKeywords(content string) []string {
	// Simple keyword extraction - could be enhanced with NLP
	words := strings.Fields(strings.ToLower(content))
	keywords := make([]string, 0)

	// Tech keywords
	techKeywords := []string{"go", "golang", "docker", "chroma", "vector", "embedding", "mcp", "server", "api", "http", "json", "test", "build", "deploy"}

	for _, word := range words {
		for _, tech := range techKeywords {
			if strings.Contains(word, tech) && len(word) > 2 {
				keywords = append(keywords, word)
				break
			}
		}
	}

	return keywords
}

// extractTechnologies identifies technologies used in the session
func (fd *FlowDetector) extractTechnologies(session *ConversationSession) []string {
	techMap := make(map[string]bool)

	for _, segment := range session.Segments {
		for _, entity := range segment.Entities {
			// Check file extensions
			switch {
			case strings.Contains(entity, ".go"):
				techMap["Go"] = true
			case strings.Contains(entity, ".js") || strings.Contains(entity, ".ts"):
				techMap["JavaScript/TypeScript"] = true
			case strings.Contains(entity, ".py"):
				techMap["Python"] = true
			case strings.Contains(entity, "docker") || strings.Contains(entity, "Dockerfile"):
				techMap["Docker"] = true
			}
		}

		// Check keywords
		for _, keyword := range segment.Keywords {
			switch {
			case strings.Contains(keyword, "chroma"):
				techMap["Chroma"] = true
			case strings.Contains(keyword, "vector"):
				techMap["Vector Database"] = true
			case strings.Contains(keyword, "mcp"):
				techMap["MCP"] = true
			}
		}
	}

	technologies := make([]string, 0, len(techMap))
	for tech := range techMap {
		technologies = append(technologies, tech)
	}

	return technologies
}

// extractDecisions identifies key decisions made during the session
func (fd *FlowDetector) extractDecisions(session *ConversationSession) []string {
	decisions := make([]string, 0)

	for _, segment := range session.Segments {
		if segment.Flow != types.FlowSolution {
			continue
		}

		newDecisions := fd.extractDecisionsFromSegment(&segment)
		decisions = append(decisions, newDecisions...)

		if len(decisions) >= 5 { // Limit to 5 key decisions
			return decisions[:5]
		}
	}

	return decisions
}

func (fd *FlowDetector) extractDecisionsFromSegment(segment *ConversationSegment) []string {
	content := strings.ToLower(segment.Content)
	if !fd.containsDecisionLanguage(content) {
		return nil
	}

	return fd.extractDecisionLines(segment.Content)
}

func (fd *FlowDetector) containsDecisionLanguage(content string) bool {
	decisionPhrases := []string{"let's use", "I'll implement", "we should"}
	for _, phrase := range decisionPhrases {
		if strings.Contains(content, phrase) {
			return true
		}
	}
	return false
}

func (fd *FlowDetector) extractDecisionLines(content string) []string {
	var decisions []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if fd.isValidDecisionLine(trimmedLine) {
			decisions = append(decisions, trimmedLine)
			if len(decisions) >= 5 {
				break
			}
		}
	}

	return decisions
}

func (fd *FlowDetector) isValidDecisionLine(line string) bool {
	return len(line) > 20 && len(line) < 100
}

// inferTransitionTrigger determines what caused a flow transition
func (fd *FlowDetector) inferTransitionTrigger(from, to types.ConversationFlow) string {
	switch {
	case from == types.FlowProblem && to == types.FlowInvestigation:
		return "began investigation"
	case from == types.FlowInvestigation && to == types.FlowSolution:
		return "found solution"
	case from == types.FlowSolution && to == types.FlowVerification:
		return "testing implementation"
	case from == types.FlowVerification && to == types.FlowProblem:
		return "discovered new issue"
	default:
		return "context change"
	}
}

// getOrCreateSession gets an existing session or creates a new one
func (fd *FlowDetector) getOrCreateSession(sessionID string) *ConversationSession {
	if session, exists := fd.sessions[sessionID]; exists {
		return session
	}

	fd.StartSession(sessionID, "unknown")
	return fd.sessions[sessionID]
}

// GetSession returns a completed session
func (fd *FlowDetector) GetSession(sessionID string) (*ConversationSession, bool) {
	session, exists := fd.sessions[sessionID]
	return session, exists
}

// GetActiveSessions returns all currently active sessions
func (fd *FlowDetector) GetActiveSessions() map[string]*ConversationSession {
	active := make(map[string]*ConversationSession)
	for id, session := range fd.sessions {
		if session.EndTime == nil {
			active[id] = session
		}
	}
	return active
}

// Extract extracts entities from text content
func (ee *EntityExtractor) Extract(content string) []string {
	entities := make([]string, 0)

	// Extract file paths
	files := ee.filePattern.FindAllString(content, -1)
	entities = append(entities, files...)

	// Extract error codes/messages
	errors := ee.errorPattern.FindAllString(content, -1)
	entities = append(entities, errors...)

	// Extract commands
	commands := ee.commandPattern.FindAllString(content, -1)
	entities = append(entities, commands...)

	// Extract URLs
	urls := ee.urlPattern.FindAllString(content, -1)
	entities = append(entities, urls...)

	return entities
}

// generateSegmentID creates a unique ID for conversation segments
func generateSegmentID() string {
	return "seg-" + generateID()[4:] // Remove "seq-" prefix and add "seg-"
}
