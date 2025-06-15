// Package services provides domain services for the lerian-mcp-memory CLI application
package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"lerian-mcp-memory-cli/internal/adapters/secondary/prompts"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// ReviewService implements the code review functionality
type ReviewService struct {
	aiService    ports.AIService
	mcpClient    ports.MCPClient
	promptLoader *prompts.PromptLoader
	storage      ports.Storage
	mu           sync.Mutex
	sessions     map[string]*entities.ReviewSession
}

// NewReviewService creates a new review service
func NewReviewService(
	aiService ports.AIService,
	mcpClient ports.MCPClient,
	promptLoader *prompts.PromptLoader,
	storage ports.Storage,
) *ReviewService {
	return &ReviewService{
		aiService:    aiService,
		mcpClient:    mcpClient,
		promptLoader: promptLoader,
		storage:      storage,
		sessions:     make(map[string]*entities.ReviewSession),
	}
}

// StartReview initiates a new review session
func (s *ReviewService) StartReview(ctx context.Context, config entities.ReviewConfiguration) (*entities.ReviewSession, error) {
	// Create new session
	session := &entities.ReviewSession{
		ID:         uuid.New().String(),
		Mode:       config.Mode,
		Repository: s.detectRepository(),
		Branch:     s.detectBranch(),
		StartedAt:  time.Now(),
		Status:     entities.ReviewStatusPending,
		Progress: entities.ReviewProgress{
			PhaseProgress: make(map[entities.ReviewPhase]entities.PhaseProgress),
		},
		Findings: []entities.ReviewFinding{},
		Metadata: map[string]interface{}{
			"config": config,
		},
	}

	// Store session
	s.mu.Lock()
	s.sessions[session.ID] = session
	s.mu.Unlock()

	// Start review in background with inherited context
	reviewCtx, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()
		s.executeReview(reviewCtx, session, config)
	}()

	return session, nil
}

// executeReview runs the review process
func (s *ReviewService) executeReview(ctx context.Context, session *entities.ReviewSession, config entities.ReviewConfiguration) {
	// Update status
	s.updateSessionStatus(session.ID, entities.ReviewStatusInProgress)

	// Load prompts based on mode
	prompts := s.promptLoader.GetPromptsForMode(config.Mode)

	// Filter by included/excluded phases if specified
	if len(config.IncludePhases) > 0 || len(config.ExcludePhases) > 0 {
		prompts = s.filterPromptsByPhases(prompts, config.IncludePhases, config.ExcludePhases)
	}

	// Initialize progress
	session.Progress.TotalPrompts = len(prompts)
	s.initializePhaseProgress(session, prompts)

	// Execute prompts
	for _, prompt := range prompts {
		select {
		case <-ctx.Done():
			s.updateSessionStatus(session.ID, entities.ReviewStatusCancelled)
			return
		default:
			// Execute single prompt
			err := s.executePrompt(ctx, session, prompt, config)
			if err != nil {
				// Log error but continue with other prompts
				fmt.Printf("Error executing prompt %s: %v\n", prompt.ID, err)
			}

			// Update progress
			s.updateProgress(session, prompt)
		}
	}

	// Generate summary
	summary := s.generateSummary(session)
	session.Summary = summary

	// Generate todo list if requested
	if config.GenerateTodoList {
		err := s.generateTodoList(ctx, session, config.OutputDirectory)
		if err != nil {
			fmt.Printf("Error generating todo list: %v\n", err)
		}
	}

	// Store in memory if requested
	if config.StoreInMemory {
		s.storeInMemory(ctx, session)
	}

	// Mark as completed
	now := time.Now()
	session.CompletedAt = &now
	s.updateSessionStatus(session.ID, entities.ReviewStatusCompleted)
}

// executePrompt executes a single review prompt
func (s *ReviewService) executePrompt(ctx context.Context, session *entities.ReviewSession, prompt *entities.ReviewPrompt, config entities.ReviewConfiguration) error {
	execution := &entities.PromptExecution{
		PromptID:  prompt.ID,
		SessionID: session.ID,
		StartedAt: time.Now(),
		Status:    entities.ReviewStatusInProgress,
	}

	// Update current prompt
	session.Progress.CurrentPromptID = prompt.ID
	session.Progress.CurrentPhase = prompt.Phase

	// Prepare context with previous findings
	contextContent := s.buildPromptContext(session, prompt)

	// Execute with AI service
	request := &ports.ContentAnalysisRequest{
		Content: prompt.Content + "\n\n" + contextContent,
		Type:    "code-review",
		Context: map[string]interface{}{
			"prompt_id":   prompt.ID,
			"prompt_name": prompt.Name,
			"phase":       prompt.Phase,
			"repository":  session.Repository,
			"branch":      session.Branch,
		},
	}

	// Set timeout
	timeout := config.TimeoutPerPrompt
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute analysis
	response, err := s.aiService.AnalyzeContent(execCtx, request)
	if err != nil {
		errStr := err.Error()
		execution.Error = &errStr
		execution.Status = entities.ReviewStatusFailed

		// Retry if configured
		if config.RetryCount > 0 {
			for i := 0; i < config.RetryCount; i++ {
				time.Sleep(time.Duration(i+1) * time.Second)
				response, err = s.aiService.AnalyzeContent(execCtx, request)
				if err == nil {
					break
				}
			}
		}

		if err != nil {
			return fmt.Errorf("failed to analyze with prompt %s: %w", prompt.ID, err)
		}
	}

	// Build full response content from analysis
	responseContent := s.buildResponseContent(response)

	// Parse findings from response
	findings := s.parseFindings(responseContent, prompt.ID)

	// Apply severity filter
	if config.SeverityFilter != "" {
		findings = s.filterFindingsBySeverity(findings, config.SeverityFilter)
	}

	// Update execution
	now := time.Now()
	execution.CompletedAt = &now
	execution.Status = entities.ReviewStatusCompleted
	execution.Response = responseContent
	execution.Findings = findings

	// Add findings to session
	s.mu.Lock()
	session.Findings = append(session.Findings, findings...)
	s.mu.Unlock()

	// Write output if directory specified
	if config.OutputDirectory != "" {
		err = s.writePromptOutput(prompt, responseContent, config.OutputDirectory)
		if err != nil {
			fmt.Printf("Error writing output for prompt %s: %v\n", prompt.ID, err)
		}
	}

	return nil
}

// buildResponseContent builds a complete response content from ContentAnalysisResponse
func (s *ReviewService) buildResponseContent(response *ports.ContentAnalysisResponse) string {
	var content strings.Builder

	// Add summary
	if response.Summary != "" {
		content.WriteString(response.Summary)
		content.WriteString("\n\n")
	}

	// Add key features
	if len(response.KeyFeatures) > 0 {
		content.WriteString("## Key Features\n")
		for _, feature := range response.KeyFeatures {
			content.WriteString(fmt.Sprintf("- %s\n", feature))
		}
		content.WriteString("\n")
	}

	// Add technical requirements
	if len(response.TechnicalReqs) > 0 {
		content.WriteString("## Technical Requirements\n")
		for _, req := range response.TechnicalReqs {
			content.WriteString(fmt.Sprintf("- %s\n", req))
		}
		content.WriteString("\n")
	}

	// Add sections
	for _, section := range response.Sections {
		content.WriteString(fmt.Sprintf("## %s\n", section.Title))
		content.WriteString(section.Content)
		content.WriteString("\n\n")
	}

	return content.String()
}

// parseFindings extracts review findings from AI response
func (s *ReviewService) parseFindings(content string, promptID string) []entities.ReviewFinding {
	findings := []entities.ReviewFinding{}

	// Extract findings from each severity section
	severityPatterns := map[string]entities.ReviewSeverity{
		"CRITICAL|🔴": entities.SeverityCritical,
		"HIGH|🟡":     entities.SeverityHigh,
		"MEDIUM|🟢":   entities.SeverityMedium,
		"LOW|🔵":      entities.SeverityLow,
	}

	for pattern, severity := range severityPatterns {
		re := regexp.MustCompile(fmt.Sprintf(`(?i)(?:^|\n).*(?:%s).*\n((?:- \[.\].*\n?)+)`, pattern))
		matches := re.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) > 1 {
				findingsList := s.parseTaskList(match[1], severity, promptID)
				findings = append(findings, findingsList...)
			}
		}
	}

	// If no structured findings, look for general issues
	if len(findings) == 0 {
		findings = s.parseGeneralFindings(content, promptID)
	}

	return findings
}

// parseTaskList parses a markdown task list into findings
func (s *ReviewService) parseTaskList(taskList string, severity entities.ReviewSeverity, promptID string) []entities.ReviewFinding {
	findings := []entities.ReviewFinding{}

	// Match task items
	taskRe := regexp.MustCompile(`- \[.\] \*\*(.+?)\*\*:?\s*(.+?)(?:\n\s+- \*\*(.+?)\*\*:?\s*(.+?))*`)
	matches := taskRe.FindAllStringSubmatch(taskList, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			finding := entities.ReviewFinding{
				ID:          uuid.New().String(),
				PromptID:    promptID,
				Title:       strings.TrimSpace(match[1]),
				Description: strings.TrimSpace(match[2]),
				Severity:    severity,
				CreatedAt:   time.Now(),
				Details:     make(map[string]interface{}),
			}

			// Extract additional details
			detailRe := regexp.MustCompile(`\*\*(.+?)\*\*:\s*(.+)`)
			detailMatches := detailRe.FindAllStringSubmatch(match[0], -1)

			for _, detail := range detailMatches {
				if len(detail) >= 3 {
					key := strings.ToLower(strings.TrimSpace(detail[1]))
					value := strings.TrimSpace(detail[2])

					switch key {
					case "impact":
						finding.Impact = value
					case "effort":
						finding.Effort = value
					case "files":
						finding.Files = s.parseFileList(value)
					default:
						finding.Details[key] = value
					}
				}
			}

			findings = append(findings, finding)
		}
	}

	return findings
}

// parseGeneralFindings extracts findings from unstructured content
func (s *ReviewService) parseGeneralFindings(content string, promptID string) []entities.ReviewFinding {
	findings := []entities.ReviewFinding{}

	// Look for issue patterns
	issuePatterns := []string{
		`(?i)issue:?\s*(.+)`,
		`(?i)problem:?\s*(.+)`,
		`(?i)vulnerability:?\s*(.+)`,
		`(?i)bug:?\s*(.+)`,
		`(?i)error:?\s*(.+)`,
		`(?i)warning:?\s*(.+)`,
	}

	for _, pattern := range issuePatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) > 1 {
				finding := entities.ReviewFinding{
					ID:          uuid.New().String(),
					PromptID:    promptID,
					Title:       "Issue Found",
					Description: strings.TrimSpace(match[1]),
					Severity:    entities.SeverityMedium,
					CreatedAt:   time.Now(),
				}
				findings = append(findings, finding)
			}
		}
	}

	return findings
}

// parseFileList parses a comma-separated list of files
func (s *ReviewService) parseFileList(fileStr string) []string {
	files := []string{}

	// Remove backticks and split
	fileStr = strings.ReplaceAll(fileStr, "`", "")
	parts := strings.Split(fileStr, ",")

	for _, part := range parts {
		file := strings.TrimSpace(part)
		if file != "" {
			files = append(files, file)
		}
	}

	return files
}

// buildPromptContext builds context for a prompt execution
func (s *ReviewService) buildPromptContext(session *entities.ReviewSession, prompt *entities.ReviewPrompt) string {
	var contextParts []string

	// Add repository context
	contextParts = append(contextParts, "Repository: "+session.Repository)
	if session.Branch != "" {
		contextParts = append(contextParts, "Branch: "+session.Branch)
	}

	// Add dependent prompt results
	if len(prompt.DependsOn) > 0 {
		contextParts = append(contextParts, "\n## Context from Previous Analyses:")

		for _, depID := range prompt.DependsOn {
			// Find findings from dependent prompt
			depFindings := s.getFindingsByPromptID(session, depID)
			if len(depFindings) > 0 {
				contextParts = append(contextParts, fmt.Sprintf("\n### From %s:", depID))
				for _, finding := range depFindings {
					contextParts = append(contextParts, fmt.Sprintf("- %s: %s", finding.Severity, finding.Title))
				}
			}
		}
	}

	// Add phase context
	if prompt.Phase != entities.PhaseFoundation {
		phaseFindings := s.getFindingsByPhase(session, prompt.Phase)
		if len(phaseFindings) > 0 {
			contextParts = append(contextParts, fmt.Sprintf("\n## Current Phase (%s) Findings:", prompt.Phase))
			contextParts = append(contextParts, fmt.Sprintf("Total findings so far: %d", len(phaseFindings)))
		}
	}

	return strings.Join(contextParts, "\n")
}

// Helper methods

func (s *ReviewService) detectRepository() string {
	// Try to detect git repository
	if gitRemote, err := s.getGitRemoteURL(); err == nil {
		return gitRemote
	}

	// Fallback to current directory
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Base(cwd)
	}

	return "unknown"
}

func (s *ReviewService) detectBranch() string {
	// Try to detect git branch
	// This would use git commands or a git library
	return "main"
}

func (s *ReviewService) getGitRemoteURL() (string, error) {
	// Implementation would use git commands
	return "", errors.New("not implemented")
}

func (s *ReviewService) updateSessionStatus(sessionID string, status entities.ReviewStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, exists := s.sessions[sessionID]; exists {
		session.Status = status
	}
}

func (s *ReviewService) updateProgress(session *entities.ReviewSession, prompt *entities.ReviewPrompt) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session.Progress.CompletedPrompts++

	// Update phase progress
	if phaseProgress, exists := session.Progress.PhaseProgress[prompt.Phase]; exists {
		phaseProgress.CompletedPrompts++
		if phaseProgress.CompletedPrompts >= phaseProgress.TotalPrompts {
			now := time.Now()
			phaseProgress.CompletedAt = &now
			phaseProgress.Status = entities.ReviewStatusCompleted
		}
		session.Progress.PhaseProgress[prompt.Phase] = phaseProgress
	}
}

func (s *ReviewService) initializePhaseProgress(session *entities.ReviewSession, prompts []*entities.ReviewPrompt) {
	// Count prompts per phase
	phaseCounts := make(map[entities.ReviewPhase]int)
	for _, prompt := range prompts {
		phaseCounts[prompt.Phase]++
	}

	// Initialize progress for each phase
	for phase, count := range phaseCounts {
		session.Progress.PhaseProgress[phase] = entities.PhaseProgress{
			Status:           entities.ReviewStatusPending,
			TotalPrompts:     count,
			CompletedPrompts: 0,
		}
	}
}

func (s *ReviewService) filterPromptsByPhases(prompts []*entities.ReviewPrompt, include, exclude []entities.ReviewPhase) []*entities.ReviewPrompt {
	filtered := []*entities.ReviewPrompt{}

	includeMap := make(map[entities.ReviewPhase]bool)
	excludeMap := make(map[entities.ReviewPhase]bool)

	for _, phase := range include {
		includeMap[phase] = true
	}
	for _, phase := range exclude {
		excludeMap[phase] = true
	}

	for _, prompt := range prompts {
		// If include list specified, prompt must be in it
		if len(includeMap) > 0 && !includeMap[prompt.Phase] {
			continue
		}

		// If exclude list specified, prompt must not be in it
		if excludeMap[prompt.Phase] {
			continue
		}

		filtered = append(filtered, prompt)
	}

	return filtered
}

func (s *ReviewService) filterFindingsBySeverity(findings []entities.ReviewFinding, minSeverity entities.ReviewSeverity) []entities.ReviewFinding {
	severityOrder := map[entities.ReviewSeverity]int{
		entities.SeverityCritical: 4,
		entities.SeverityHigh:     3,
		entities.SeverityMedium:   2,
		entities.SeverityLow:      1,
	}

	minOrder := severityOrder[minSeverity]
	filtered := []entities.ReviewFinding{}

	for _, finding := range findings {
		if severityOrder[finding.Severity] >= minOrder {
			filtered = append(filtered, finding)
		}
	}

	return filtered
}

func (s *ReviewService) getFindingsByPromptID(session *entities.ReviewSession, promptID string) []entities.ReviewFinding {
	findings := []entities.ReviewFinding{}

	for _, finding := range session.Findings {
		if finding.PromptID == promptID {
			findings = append(findings, finding)
		}
	}

	return findings
}

func (s *ReviewService) getFindingsByPhase(session *entities.ReviewSession, phase entities.ReviewPhase) []entities.ReviewFinding {
	findings := []entities.ReviewFinding{}

	// Need to map prompt IDs to phases
	for _, finding := range session.Findings {
		if prompt, err := s.promptLoader.GetPrompt(finding.PromptID); err == nil {
			if prompt.Phase == phase {
				findings = append(findings, finding)
			}
		}
	}

	return findings
}

func (s *ReviewService) generateSummary(session *entities.ReviewSession) *entities.ReviewSummary {
	summary := &entities.ReviewSummary{
		TotalFindings:      len(session.Findings),
		FindingsBySeverity: make(map[entities.ReviewSeverity]int),
		FindingsByPhase:    make(map[entities.ReviewPhase]int),
		CriticalIssues:     []string{},
		ImmediateActions:   []string{},
		GeneratedAt:        time.Now(),
	}

	// Count by severity and collect critical issues
	for _, finding := range session.Findings {
		summary.FindingsBySeverity[finding.Severity]++

		if finding.Severity == entities.SeverityCritical {
			summary.CriticalIssues = append(summary.CriticalIssues, finding.Title)
			summary.ImmediateActions = append(summary.ImmediateActions, finding.Description)
		}
	}

	// Count by phase
	for _, finding := range session.Findings {
		if prompt, err := s.promptLoader.GetPrompt(finding.PromptID); err == nil {
			summary.FindingsByPhase[prompt.Phase]++
		}
	}

	// Estimate effort
	totalEffort := 0
	for _, finding := range session.Findings {
		if finding.Effort != "" {
			// Parse effort (e.g., "2 hours", "1 day")
			// This is simplified
			if strings.Contains(finding.Effort, "hour") {
				totalEffort += 1
			} else if strings.Contains(finding.Effort, "day") {
				totalEffort += 8
			}
		}
	}
	summary.EstimatedEffort = fmt.Sprintf("%d hours", totalEffort)

	// Production readiness
	if summary.FindingsBySeverity[entities.SeverityCritical] > 0 {
		summary.ProductionReadiness = "Not Ready - Critical issues found"
	} else if summary.FindingsBySeverity[entities.SeverityHigh] > 5 {
		summary.ProductionReadiness = "Not Ready - Multiple high priority issues"
	} else if summary.FindingsBySeverity[entities.SeverityHigh] > 0 {
		summary.ProductionReadiness = "Nearly Ready - Address high priority issues"
	} else {
		summary.ProductionReadiness = "Ready - Only minor issues remain"
	}

	return summary
}

func (s *ReviewService) writePromptOutput(prompt *entities.ReviewPrompt, content string, outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("%d-%s.md", prompt.Order, strings.ToUpper(strings.ReplaceAll(prompt.Name, " ", "_")))
	filepath := filepath.Join(outputDir, filename)

	// Write content
	return os.WriteFile(filepath, []byte(content), 0600)
}

func (s *ReviewService) generateTodoList(_ context.Context, session *entities.ReviewSession, outputDir string) error {
	// Group findings by severity
	findingsBySeverity := make(map[entities.ReviewSeverity][]entities.ReviewFinding)

	for _, finding := range session.Findings {
		findingsBySeverity[finding.Severity] = append(findingsBySeverity[finding.Severity], finding)
	}

	// Build todo list content
	var content strings.Builder
	content.WriteString("# Code Review Todo List\n\n")
	content.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	content.WriteString(fmt.Sprintf("Repository: %s\n", session.Repository))
	content.WriteString(fmt.Sprintf("Total Findings: %d\n\n", len(session.Findings)))

	// Write findings by severity
	severityOrder := []entities.ReviewSeverity{
		entities.SeverityCritical,
		entities.SeverityHigh,
		entities.SeverityMedium,
		entities.SeverityLow,
	}

	severityEmojis := map[entities.ReviewSeverity]string{
		entities.SeverityCritical: "🔴",
		entities.SeverityHigh:     "🟡",
		entities.SeverityMedium:   "🟢",
		entities.SeverityLow:      "🔵",
	}

	for _, severity := range severityOrder {
		findings := findingsBySeverity[severity]
		if len(findings) == 0 {
			continue
		}

		content.WriteString(fmt.Sprintf("## %s %s (%d items)\n\n", severityEmojis[severity], strings.ToUpper(string(severity)), len(findings)))

		for _, finding := range findings {
			content.WriteString(fmt.Sprintf("- [ ] **%s**: %s\n", finding.Title, finding.Description))
			if finding.Impact != "" {
				content.WriteString(fmt.Sprintf("  - **Impact**: %s\n", finding.Impact))
			}
			if finding.Effort != "" {
				content.WriteString(fmt.Sprintf("  - **Effort**: %s\n", finding.Effort))
			}
			if len(finding.Files) > 0 {
				content.WriteString(fmt.Sprintf("  - **Files**: `%s`\n", strings.Join(finding.Files, "`, `")))
			}
			if len(finding.Tags) > 0 {
				content.WriteString(fmt.Sprintf("  - **Tags**: #%s\n", strings.Join(finding.Tags, " #")))
			}
			content.WriteString("\n")
		}
	}

	// Write file
	filename := "code-review-todo-list.md"
	if outputDir != "" {
		filename = filepath.Join(outputDir, filename)
	}

	return os.WriteFile(filename, []byte(content.String()), 0600)
}

func (s *ReviewService) storeInMemory(_ context.Context, session *entities.ReviewSession) {
	// TODO: Implement memory storage when MCP client supports it
	// For now, we'll just log that we would store it

	summaryContent := fmt.Sprintf("Code Review Session: %s\nRepository: %s\nMode: %s\nTotal Findings: %d\nCritical: %d, High: %d, Medium: %d, Low: %d",
		session.ID,
		session.Repository,
		session.Mode,
		len(session.Findings),
		session.Summary.FindingsBySeverity[entities.SeverityCritical],
		session.Summary.FindingsBySeverity[entities.SeverityHigh],
		session.Summary.FindingsBySeverity[entities.SeverityMedium],
		session.Summary.FindingsBySeverity[entities.SeverityLow],
	)

	// Log what we would store
	fmt.Printf("Would store in memory: %s\n", summaryContent)

	// Store critical findings count
	criticalCount := 0
	for _, finding := range session.Findings {
		if finding.Severity == entities.SeverityCritical || finding.Severity == entities.SeverityHigh {
			criticalCount++
		}
	}

	if criticalCount > 0 {
		fmt.Printf("Would store %d critical/high findings in memory\n", criticalCount)
	}
}

// GetSession retrieves a review session by ID
func (s *ReviewService) GetSession(sessionID string) (*entities.ReviewSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// ListSessions returns all review sessions
func (s *ReviewService) ListSessions() []*entities.ReviewSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions := make([]*entities.ReviewSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// CancelReview cancels an in-progress review
func (s *ReviewService) CancelReview(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Status != entities.ReviewStatusInProgress {
		return errors.New("session is not in progress")
	}

	session.Status = entities.ReviewStatusCancelled
	now := time.Now()
	session.CompletedAt = &now

	return nil
}
