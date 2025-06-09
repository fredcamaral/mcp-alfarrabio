package services

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/repositories"
)

// PatternDetector interface defines pattern detection capabilities
type PatternDetector interface {
	// Sequence pattern detection
	DetectSequencePatterns(ctx context.Context, tasks []*entities.Task, minSupport float64) ([]*entities.TaskPattern, error)

	// Workflow pattern detection
	DetectWorkflowPatterns(ctx context.Context, repository string, timeRange TimeRange) ([]*entities.TaskPattern, error)

	// Temporal pattern detection
	DetectTemporalPatterns(ctx context.Context, sessions []*entities.Session) ([]*entities.TaskPattern, error)

	// Pattern scoring and evaluation
	CalculatePatternScore(pattern *entities.TaskPattern, newSequence []*entities.Task) float64
	UpdatePatternStatistics(ctx context.Context, pattern *entities.TaskPattern, outcome *entities.PatternOutcome) error

	// Pattern management
	GetActivePatterns(ctx context.Context, repository string) ([]*entities.TaskPattern, error)
	RefreshPatterns(ctx context.Context, repository string) error
	GetPatternSuggestions(ctx context.Context, currentTasks []*entities.Task) ([]*entities.TaskPattern, error)
}

// TimeRange represents a time period for analysis
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// PatternDetectorConfig holds configuration for pattern detection
type PatternDetectorConfig struct {
	MinOccurrences      int     // Minimum times pattern must occur
	MinConfidence       float64 // Minimum confidence score
	MinSupport          float64 // Minimum support for sequence patterns
	MaxPatternLength    int     // Maximum length of sequence patterns
	MinPatternLength    int     // Minimum length of sequence patterns
	TemporalWindowHours int     // Hours to consider for temporal patterns
	PatternExpiryDays   int     // Days after which patterns expire
	LearningRate        float64 // Rate at which patterns adapt to new data
}

// DefaultPatternDetectorConfig returns default configuration
func DefaultPatternDetectorConfig() *PatternDetectorConfig {
	return &PatternDetectorConfig{
		MinOccurrences:      3,
		MinConfidence:       0.6,
		MinSupport:          0.1,
		MaxPatternLength:    5,
		MinPatternLength:    2,
		TemporalWindowHours: 24,
		PatternExpiryDays:   30,
		LearningRate:        0.1,
	}
}

// patternDetectorImpl implements the PatternDetector interface
type patternDetectorImpl struct {
	taskRepo    repositories.TaskRepository
	patternRepo PatternRepository
	sessionRepo SessionRepository
	analytics   AnalyticsEngine
	config      *PatternDetectorConfig
	logger      *slog.Logger
}

// NewPatternDetector creates a new pattern detector
func NewPatternDetector(
	taskRepo repositories.TaskRepository,
	patternRepo PatternRepository,
	sessionRepo SessionRepository,
	analytics AnalyticsEngine,
	config *PatternDetectorConfig,
	logger *slog.Logger,
) PatternDetector {
	if config == nil {
		config = DefaultPatternDetectorConfig()
	}

	return &patternDetectorImpl{
		taskRepo:    taskRepo,
		patternRepo: patternRepo,
		sessionRepo: sessionRepo,
		analytics:   analytics,
		config:      config,
		logger:      logger,
	}
}

// DetectSequencePatterns detects common task sequences using modified Apriori algorithm
func (pd *patternDetectorImpl) DetectSequencePatterns(
	ctx context.Context,
	tasks []*entities.Task,
	minSupport float64,
) ([]*entities.TaskPattern, error) {
	pd.logger.Info("detecting sequence patterns",
		slog.Int("task_count", len(tasks)),
		slog.Float64("min_support", minSupport))

	// Group tasks into sequences based on completion time
	sequences := pd.extractTaskSequences(tasks)
	pd.logger.Debug("extracted sequences", slog.Int("sequence_count", len(sequences)))

	// Find frequent patterns using modified Apriori algorithm
	patterns := make(map[string]*entities.TaskPattern)

	for _, seq := range sequences {
		subSeqs := pd.generateSubsequences(seq, pd.config.MinPatternLength, pd.config.MaxPatternLength)

		for _, subSeq := range subSeqs {
			key := pd.generatePatternKey(subSeq)

			if pattern, exists := patterns[key]; exists {
				pattern.Occurrences++
				pd.updatePatternStats(pattern, subSeq)
			} else {
				pattern := pd.createPatternFromSequence(subSeq)
				if pattern != nil {
					patterns[key] = pattern
				}
			}
		}
	}

	// Filter by support and confidence
	var results []*entities.TaskPattern
	totalSequences := float64(len(sequences))

	for _, pattern := range patterns {
		support := float64(pattern.Occurrences) / totalSequences
		confidence := pattern.CalculateConfidence()

		if support >= minSupport && confidence >= pd.config.MinConfidence {
			pattern.Frequency = support
			pattern.Confidence = confidence
			results = append(results, pattern)
		}
	}

	// Sort by frequency and confidence
	sort.Slice(results, func(i, j int) bool {
		if math.Abs(results[i].Frequency-results[j].Frequency) < 0.001 {
			return results[i].Confidence > results[j].Confidence
		}
		return results[i].Frequency > results[j].Frequency
	})

	pd.logger.Info("sequence pattern detection completed",
		slog.Int("patterns_found", len(results)))

	return results, nil
}

// DetectWorkflowPatterns detects high-level workflow patterns
func (pd *patternDetectorImpl) DetectWorkflowPatterns(
	ctx context.Context,
	repository string,
	timeRange TimeRange,
) ([]*entities.TaskPattern, error) {
	pd.logger.Info("detecting workflow patterns",
		slog.String("repository", repository),
		slog.Time("start", timeRange.Start),
		slog.Time("end", timeRange.End))

	// Get tasks in the time range
	tasks, err := pd.taskRepo.FindByTimeRange(ctx, repository, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Group tasks by project or feature
	projectGroups := pd.groupTasksByProject(tasks)

	var patterns []*entities.TaskPattern

	for projectType, projectTasks := range projectGroups {
		workflowPattern := pd.analyzeWorkflowPattern(projectType, projectTasks)
		if workflowPattern != nil {
			patterns = append(patterns, workflowPattern)
		}
	}

	pd.logger.Info("workflow pattern detection completed",
		slog.Int("patterns_found", len(patterns)))

	return patterns, nil
}

// DetectTemporalPatterns detects time-based patterns from sessions
func (pd *patternDetectorImpl) DetectTemporalPatterns(
	ctx context.Context,
	sessions []*entities.Session,
) ([]*entities.TaskPattern, error) {
	pd.logger.Info("detecting temporal patterns", slog.Int("session_count", len(sessions)))

	// Analyze daily patterns
	dailyPatterns := pd.analyzeDailyPatterns(sessions)

	// Analyze weekly patterns
	weeklyPatterns := pd.analyzeWeeklyPatterns(sessions)

	// Combine patterns
	var allPatterns []*entities.TaskPattern
	allPatterns = append(allPatterns, dailyPatterns...)
	allPatterns = append(allPatterns, weeklyPatterns...)

	// Filter significant patterns
	var significantPatterns []*entities.TaskPattern
	for _, pattern := range allPatterns {
		if pattern.IsSignificant(pd.config.MinOccurrences, pd.config.MinConfidence) {
			significantPatterns = append(significantPatterns, pattern)
		}
	}

	pd.logger.Info("temporal pattern detection completed",
		slog.Int("patterns_found", len(significantPatterns)))

	return significantPatterns, nil
}

// CalculatePatternScore calculates how well a new sequence matches a pattern
func (pd *patternDetectorImpl) CalculatePatternScore(
	pattern *entities.TaskPattern,
	newSequence []*entities.Task,
) float64 {
	if len(pattern.Sequence) == 0 || len(newSequence) == 0 {
		return 0.0
	}

	// Calculate similarity score based on:
	// 1. Task type matches (40%)
	// 2. Keyword similarity (30%)
	// 3. Priority alignment (20%)
	// 4. Duration similarity (10%)

	var typeScore, keywordScore, priorityScore, durationScore float64

	// Type matching
	typeMatches := 0
	for i, step := range pattern.Sequence {
		if i < len(newSequence) {
			if step.TaskType == newSequence[i].Type {
				typeMatches++
			}
		}
	}
	typeScore = float64(typeMatches) / float64(len(pattern.Sequence))

	// Keyword similarity
	patternKeywords := pattern.GetKeywords()
	newKeywords := pd.extractKeywords(newSequence)
	keywordScore = pd.calculateKeywordSimilarity(patternKeywords, newKeywords)

	// Priority alignment
	priorityMatches := 0
	for i, step := range pattern.Sequence {
		if i < len(newSequence) && step.Priority == string(newSequence[i].Priority) {
			priorityMatches++
		}
	}
	priorityScore = float64(priorityMatches) / float64(len(pattern.Sequence))

	// Duration similarity (if available)
	durationScore = pd.calculateDurationSimilarity(pattern, newSequence)

	// Weighted composite score
	score := (typeScore * 0.4) +
		(keywordScore * 0.3) +
		(priorityScore * 0.2) +
		(durationScore * 0.1)

	return score
}

// UpdatePatternStatistics updates pattern statistics with new outcome data
func (pd *patternDetectorImpl) UpdatePatternStatistics(
	ctx context.Context,
	pattern *entities.TaskPattern,
	outcome *entities.PatternOutcome,
) error {
	pd.logger.Debug("updating pattern statistics",
		slog.String("pattern_id", pattern.ID),
		slog.Bool("completed", outcome.Completed))

	// Update pattern with outcome
	pattern.AddOccurrence(outcome)

	// Apply learning rate to confidence adjustment
	if outcome.Completed {
		confidenceAdjustment := pd.config.LearningRate * (1.0 - pattern.Confidence)
		pattern.Confidence += confidenceAdjustment
	} else {
		confidenceAdjustment := pd.config.LearningRate * pattern.Confidence
		pattern.Confidence -= confidenceAdjustment
	}

	// Ensure confidence stays within bounds
	if pattern.Confidence > 1.0 {
		pattern.Confidence = 1.0
	} else if pattern.Confidence < 0.0 {
		pattern.Confidence = 0.0
	}

	// Update pattern in repository
	// TODO: Add Update method to PatternRepository interface
	// if err := pd.patternRepo.Update(ctx, pattern); err != nil {
	//     return fmt.Errorf("failed to update pattern: %w", err)
	// }

	return nil
}

// GetActivePatterns retrieves currently active patterns for a repository
func (pd *patternDetectorImpl) GetActivePatterns(
	ctx context.Context,
	repository string,
) ([]*entities.TaskPattern, error) {
	patterns, err := pd.patternRepo.FindByRepository(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get patterns: %w", err)
	}

	// Filter out expired patterns
	expiryThreshold := time.Duration(pd.config.PatternExpiryDays) * 24 * time.Hour
	var activePatterns []*entities.TaskPattern

	for _, pattern := range patterns {
		if !pattern.IsExpired(expiryThreshold) &&
			pattern.IsSignificant(pd.config.MinOccurrences, pd.config.MinConfidence) {
			activePatterns = append(activePatterns, pattern)
		}
	}

	return activePatterns, nil
}

// RefreshPatterns refreshes patterns for a repository by re-analyzing recent data
func (pd *patternDetectorImpl) RefreshPatterns(ctx context.Context, repository string) error {
	pd.logger.Info("refreshing patterns", slog.String("repository", repository))

	// Get recent tasks (last 30 days)
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30)

	tasks, err := pd.taskRepo.FindByTimeRange(ctx, repository, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to get recent tasks: %w", err)
	}

	// Detect new sequence patterns
	sequencePatterns, err := pd.DetectSequencePatterns(ctx, tasks, pd.config.MinSupport)
	if err != nil {
		return fmt.Errorf("failed to detect sequence patterns: %w", err)
	}

	// Save new patterns
	for _, pattern := range sequencePatterns {
		pattern.Repository = repository
		if err := pd.patternRepo.Create(ctx, pattern); err != nil {
			pd.logger.Warn("failed to save pattern", slog.Any("error", err))
		}
	}

	pd.logger.Info("pattern refresh completed",
		slog.String("repository", repository),
		slog.Int("new_patterns", len(sequencePatterns)))

	return nil
}

// GetPatternSuggestions gets patterns that might apply to current tasks
func (pd *patternDetectorImpl) GetPatternSuggestions(
	ctx context.Context,
	currentTasks []*entities.Task,
) ([]*entities.TaskPattern, error) {
	if len(currentTasks) == 0 {
		return nil, nil
	}

	repository := currentTasks[0].Repository
	activePatterns, err := pd.GetActivePatterns(ctx, repository)
	if err != nil {
		return nil, err
	}

	// Score patterns against current tasks
	type patternScore struct {
		pattern *entities.TaskPattern
		score   float64
	}

	var scoredPatterns []patternScore
	for _, pattern := range activePatterns {
		score := pd.CalculatePatternScore(pattern, currentTasks)
		if score > 0.3 { // Minimum threshold for suggestions
			scoredPatterns = append(scoredPatterns, patternScore{
				pattern: pattern,
				score:   score,
			})
		}
	}

	// Sort by score
	sort.Slice(scoredPatterns, func(i, j int) bool {
		return scoredPatterns[i].score > scoredPatterns[j].score
	})

	// Return top patterns
	var suggestions []*entities.TaskPattern
	for i, sp := range scoredPatterns {
		if i >= 5 { // Limit to top 5
			break
		}
		suggestions = append(suggestions, sp.pattern)
	}

	return suggestions, nil
}

// Helper methods

// extractTaskSequences groups tasks into sequences based on completion time
func (pd *patternDetectorImpl) extractTaskSequences(tasks []*entities.Task) [][]*entities.Task {
	// Sort tasks by completion time
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].UpdatedAt.Before(tasks[j].UpdatedAt)
	})

	var sequences [][]*entities.Task
	var currentSeq []*entities.Task

	for i, task := range tasks {
		if task.Status != "completed" {
			continue
		}

		// Start new sequence if gap is too large (more than 4 hours)
		if len(currentSeq) > 0 {
			timeDiff := task.UpdatedAt.Sub(tasks[i-1].UpdatedAt)
			if timeDiff > 4*time.Hour {
				if len(currentSeq) >= pd.config.MinPatternLength {
					sequences = append(sequences, currentSeq)
				}
				currentSeq = []*entities.Task{task}
				continue
			}
		}

		currentSeq = append(currentSeq, task)
	}

	// Add the last sequence
	if len(currentSeq) >= pd.config.MinPatternLength {
		sequences = append(sequences, currentSeq)
	}

	return sequences
}

// generateSubsequences generates all subsequences of specified length
func (pd *patternDetectorImpl) generateSubsequences(
	sequence []*entities.Task,
	minLen, maxLen int,
) [][]*entities.Task {
	var subsequences [][]*entities.Task

	for length := minLen; length <= maxLen && length <= len(sequence); length++ {
		for start := 0; start <= len(sequence)-length; start++ {
			subSeq := make([]*entities.Task, length)
			copy(subSeq, sequence[start:start+length])
			subsequences = append(subsequences, subSeq)
		}
	}

	return subsequences
}

// generatePatternKey creates a unique key for a task sequence
func (pd *patternDetectorImpl) generatePatternKey(sequence []*entities.Task) string {
	var keyParts []string
	for _, task := range sequence {
		keyParts = append(keyParts, fmt.Sprintf("%s:%s", task.Type, task.Priority))
	}
	return strings.Join(keyParts, "->")
}

// createPatternFromSequence creates a TaskPattern from a task sequence
func (pd *patternDetectorImpl) createPatternFromSequence(sequence []*entities.Task) *entities.TaskPattern {
	if len(sequence) == 0 {
		return nil
	}

	pattern := &entities.TaskPattern{
		ID:          generatePatternID(),
		Type:        entities.PatternTypeSequence,
		Name:        fmt.Sprintf("Sequence: %s", pd.generatePatternName(sequence)),
		Description: pd.generatePatternDescription(sequence),
		Sequence:    make([]entities.PatternStep, 0, len(sequence)),
		Repository:  sequence[0].Repository,
		Occurrences: 1,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create pattern steps
	for i, task := range sequence {
		taskType := "default"
		if len(task.Tags) > 0 {
			taskType = task.Tags[0]
		}
		step := entities.PatternStep{
			Order:       i + 1,
			TaskType:    taskType,
			Keywords:    pd.extractTaskKeywords(task),
			Priority:    string(task.Priority),
			Tags:        task.Tags,
			Metadata:    make(map[string]interface{}),
			Probability: 1.0, // Initial probability
		}

		pattern.Sequence = append(pattern.Sequence, step)
	}

	return pattern
}

// updatePatternStats updates pattern statistics with new occurrence
func (pd *patternDetectorImpl) updatePatternStats(pattern *entities.TaskPattern, sequence []*entities.Task) {
	pattern.LastSeen = time.Now()
	pattern.UpdatedAt = time.Now()

	// Update duration statistics for each step
	for i, _ := range sequence {
		if i < len(pattern.Sequence) && pattern.Sequence[i].Duration != nil {
			// In a real implementation, you'd track actual task durations
			// For now, we'll use a placeholder duration
			duration := time.Hour // Placeholder
			pattern.Sequence[i].Duration.UpdateDurationStats(duration)
		}
	}
}

// extractKeywords extracts keywords from a list of tasks
func (pd *patternDetectorImpl) extractKeywords(tasks []*entities.Task) []string {
	keywordSet := make(map[string]bool)
	var keywords []string

	for _, task := range tasks {
		taskKeywords := pd.extractTaskKeywords(task)
		for _, keyword := range taskKeywords {
			if !keywordSet[keyword] {
				keywordSet[keyword] = true
				keywords = append(keywords, keyword)
			}
		}
	}

	return keywords
}

// extractTaskKeywords extracts keywords from a single task
func (pd *patternDetectorImpl) extractTaskKeywords(task *entities.Task) []string {
	// Simple keyword extraction from content
	content := strings.ToLower(task.Content)
	words := strings.Fields(content)

	var keywords []string
	for _, word := range words {
		// Filter out common words and keep meaningful terms
		if len(word) > 3 && !pd.isStopWord(word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// isStopWord checks if a word is a common stop word
func (pd *patternDetectorImpl) isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true,
		"this": true, "that": true, "these": true, "those": true,
		"is": true, "are": true, "was": true, "were": true,
		"will": true, "would": true, "could": true, "should": true,
	}

	return stopWords[word]
}

// calculateKeywordSimilarity calculates similarity between two keyword sets
func (pd *patternDetectorImpl) calculateKeywordSimilarity(keywords1, keywords2 []string) float64 {
	if len(keywords1) == 0 && len(keywords2) == 0 {
		return 1.0
	}
	if len(keywords1) == 0 || len(keywords2) == 0 {
		return 0.0
	}

	set1 := make(map[string]bool)
	for _, keyword := range keywords1 {
		set1[keyword] = true
	}

	intersection := 0
	for _, keyword := range keywords2 {
		if set1[keyword] {
			intersection++
		}
	}

	union := len(keywords1) + len(keywords2) - intersection
	if union == 0 {
		return 1.0
	}

	return float64(intersection) / float64(union) // Jaccard similarity
}

// calculateDurationSimilarity calculates duration similarity
func (pd *patternDetectorImpl) calculateDurationSimilarity(
	pattern *entities.TaskPattern,
	newSequence []*entities.Task,
) float64 {
	// Simplified implementation - in practice, you'd compare actual durations
	return 0.5 // Placeholder
}

// groupTasksByProject groups tasks by project type
func (pd *patternDetectorImpl) groupTasksByProject(tasks []*entities.Task) map[string][]*entities.Task {
	groups := make(map[string][]*entities.Task)

	for _, task := range tasks {
		projectType := pd.inferProjectType(task)
		groups[projectType] = append(groups[projectType], task)
	}

	return groups
}

// inferProjectType infers project type from task
func (pd *patternDetectorImpl) inferProjectType(task *entities.Task) string {
	// Simple heuristic based on task content and tags
	content := strings.ToLower(task.Content)

	if strings.Contains(content, "bug") || strings.Contains(content, "fix") {
		return "bugfix"
	}
	if strings.Contains(content, "feature") || strings.Contains(content, "add") {
		return "feature"
	}
	if strings.Contains(content, "refactor") || strings.Contains(content, "improve") {
		return "refactor"
	}
	if strings.Contains(content, "test") {
		return "testing"
	}

	return "general"
}

// analyzeWorkflowPattern analyzes workflow pattern for a project type
func (pd *patternDetectorImpl) analyzeWorkflowPattern(
	projectType string,
	tasks []*entities.Task,
) *entities.TaskPattern {
	if len(tasks) < pd.config.MinOccurrences {
		return nil
	}

	pattern := &entities.TaskPattern{
		ID:          generatePatternID(),
		Type:        entities.PatternTypeWorkflow,
		Name:        fmt.Sprintf("Workflow: %s", projectType),
		Description: fmt.Sprintf("Common workflow pattern for %s projects", projectType),
		Repository:  tasks[0].Repository,
		ProjectType: projectType,
		Occurrences: len(tasks),
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Analyze common phases in the workflow
	phases := pd.identifyWorkflowPhases(tasks)

	// Convert phases to pattern steps
	for i, phase := range phases {
		step := entities.PatternStep{
			Order:       i + 1,
			TaskType:    phase,
			Keywords:    []string{},
			Metadata:    make(map[string]interface{}),
			Probability: 1.0,
		}
		pattern.Sequence = append(pattern.Sequence, step)
	}

	return pattern
}

// identifyWorkflowPhases identifies common phases in a workflow
func (pd *patternDetectorImpl) identifyWorkflowPhases(tasks []*entities.Task) []string {
	// Simplified implementation - identify common task types in order
	phaseMap := make(map[string]int)

	for _, task := range tasks {
		taskType := task.Type
		if taskType == "" {
			taskType = pd.inferTaskType(task)
		}
		phaseMap[taskType]++
	}

	// Convert to ordered list
	var phases []string
	for phase := range phaseMap {
		phases = append(phases, phase)
	}

	return phases
}

// inferTaskType infers task type from content
func (pd *patternDetectorImpl) inferTaskType(task *entities.Task) string {
	content := strings.ToLower(task.Content)

	if strings.Contains(content, "plan") || strings.Contains(content, "design") {
		return "planning"
	}
	if strings.Contains(content, "implement") || strings.Contains(content, "code") {
		return "implementation"
	}
	if strings.Contains(content, "test") {
		return "testing"
	}
	if strings.Contains(content, "review") {
		return "review"
	}
	if strings.Contains(content, "deploy") || strings.Contains(content, "release") {
		return "deployment"
	}

	return "general"
}

// analyzeDailyPatterns analyzes daily productivity patterns
func (pd *patternDetectorImpl) analyzeDailyPatterns(sessions []*entities.Session) []*entities.TaskPattern {
	// Group sessions by hour of day
	hourlyData := make(map[int][]float64)

	for _, session := range sessions {
		hour := session.StartTime.Hour()
		hourlyData[hour] = append(hourlyData[hour], session.ProductivityScore)
	}

	// Find most productive hours
	var bestHours []int
	bestScore := 0.0

	for hour, scores := range hourlyData {
		if len(scores) >= pd.config.MinOccurrences {
			avgScore := pd.calculateAverage(scores)
			if avgScore > bestScore {
				bestScore = avgScore
			}
			if avgScore > 0.7 { // High productivity threshold
				bestHours = append(bestHours, hour)
			}
		}
	}

	if len(bestHours) == 0 {
		return nil
	}

	// Create temporal pattern
	pattern := &entities.TaskPattern{
		ID:          generatePatternID(),
		Type:        entities.PatternTypeTemporal,
		Name:        "Daily Productivity Pattern",
		Description: fmt.Sprintf("High productivity hours: %v", bestHours),
		Frequency:   bestScore,
		Confidence:  bestScore,
		Metadata: map[string]interface{}{
			"pattern_type":     "daily",
			"productive_hours": bestHours,
			"avg_score":        bestScore,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return []*entities.TaskPattern{pattern}
}

// analyzeWeeklyPatterns analyzes weekly productivity patterns
func (pd *patternDetectorImpl) analyzeWeeklyPatterns(sessions []*entities.Session) []*entities.TaskPattern {
	// Group sessions by day of week
	weeklyData := make(map[time.Weekday][]float64)

	for _, session := range sessions {
		day := session.StartTime.Weekday()
		weeklyData[day] = append(weeklyData[day], session.ProductivityScore)
	}

	// Find most productive days
	var bestDays []time.Weekday
	bestScore := 0.0

	for day, scores := range weeklyData {
		if len(scores) >= pd.config.MinOccurrences {
			avgScore := pd.calculateAverage(scores)
			if avgScore > bestScore {
				bestScore = avgScore
			}
			if avgScore > 0.7 { // High productivity threshold
				bestDays = append(bestDays, day)
			}
		}
	}

	if len(bestDays) == 0 {
		return nil
	}

	// Create temporal pattern
	pattern := &entities.TaskPattern{
		ID:          generatePatternID(),
		Type:        entities.PatternTypeTemporal,
		Name:        "Weekly Productivity Pattern",
		Description: fmt.Sprintf("High productivity days: %v", bestDays),
		Frequency:   bestScore,
		Confidence:  bestScore,
		Metadata: map[string]interface{}{
			"pattern_type":    "weekly",
			"productive_days": bestDays,
			"avg_score":       bestScore,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return []*entities.TaskPattern{pattern}
}

// calculateAverage calculates the average of a slice of float64
func (pd *patternDetectorImpl) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

// generatePatternID generates a unique pattern ID
func generatePatternID() string {
	return fmt.Sprintf("pattern_%d", time.Now().UnixNano())
}

// generatePatternName generates a descriptive name for a pattern
func (pd *patternDetectorImpl) generatePatternName(sequence []*entities.Task) string {
	if len(sequence) == 0 {
		return "Empty Sequence"
	}

	var types []string
	for _, task := range sequence {
		if task.Type != "" {
			types = append(types, task.Type)
		}
	}

	if len(types) == 0 {
		return fmt.Sprintf("%d-step sequence", len(sequence))
	}

	return strings.Join(types, " → ")
}

// generatePatternDescription generates a description for a pattern
func (pd *patternDetectorImpl) generatePatternDescription(sequence []*entities.Task) string {
	return fmt.Sprintf("Common sequence of %d tasks typically completed together", len(sequence))
}
