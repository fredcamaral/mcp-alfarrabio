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

// ContextAnalyzer interface defines context analysis capabilities
type ContextAnalyzer interface {
	// Core analysis
	AnalyzeCurrentContext(ctx context.Context, repository string) (*entities.WorkContext, error)

	// Context components
	ExtractKeywords(tasks []*entities.Task) []string
	DetermineProjectType(ctx context.Context, repository string) (string, error)
	CalculateFocusLevel(session *entities.Session) float64
	CalculateStressLevel(ctx context.Context, repository string) (float64, []entities.StressIndicator, error)

	// Predictive analysis
	PredictOptimalTaskType(context *entities.WorkContext) string
	PredictProductivity(context *entities.WorkContext) float64
	DetectWorkPatterns(ctx context.Context, repository string, days int) ([]*entities.TaskPattern, error)

	// Environmental analysis
	AnalyzeWorkingHours(sessions []*entities.Session) *entities.WorkingHours
	DetectEnergyPatterns(sessions []*entities.Session) map[int]float64
	AnalyzeVelocityTrends(ctx context.Context, repository string, days int) (float64, error)

	// Goal and constraint analysis
	AnalyzeCurrentGoals(ctx context.Context, repository string) ([]entities.SessionGoal, error)
	DetectWorkConstraints(ctx context.Context, workContext *entities.WorkContext) ([]entities.WorkConstraint, error)
}

// ContextAnalyzerConfig holds configuration for context analysis
type ContextAnalyzerConfig struct {
	MaxRecentTasks      int     // Maximum recent tasks to consider
	MaxRecentPatterns   int     // Maximum recent patterns to consider
	StressThreshold     float64 // Threshold for stress detection (0-1)
	FocusWindowHours    int     // Hours to look back for focus calculation
	VelocityWindowDays  int     // Days to look back for velocity calculation
	KeywordMinLength    int     // Minimum keyword length
	KeywordMaxCount     int     // Maximum keywords to extract
	PatternConfidence   float64 // Minimum confidence for pattern inclusion
	EnergyLevelsToTrack int     // Number of energy levels to track hourly
}

// DefaultContextAnalyzerConfig returns default configuration
func DefaultContextAnalyzerConfig() *ContextAnalyzerConfig {
	return &ContextAnalyzerConfig{
		MaxRecentTasks:      10,
		MaxRecentPatterns:   5,
		StressThreshold:     0.7,
		FocusWindowHours:    24,
		VelocityWindowDays:  7,
		KeywordMinLength:    3,
		KeywordMaxCount:     20,
		PatternConfidence:   0.6,
		EnergyLevelsToTrack: 24,
	}
}

// contextAnalyzerImpl implements the ContextAnalyzer interface
type contextAnalyzerImpl struct {
	taskRepo    repositories.TaskRepository
	sessionRepo SessionRepository
	patternRepo PatternRepository
	analytics   AnalyticsEngine
	config      *ContextAnalyzerConfig
	logger      *slog.Logger
}

// NewContextAnalyzer creates a new context analyzer
func NewContextAnalyzer(
	taskRepo repositories.TaskRepository,
	sessionRepo SessionRepository,
	patternRepo PatternRepository,
	analytics AnalyticsEngine,
	config *ContextAnalyzerConfig,
	logger *slog.Logger,
) ContextAnalyzer {
	if config == nil {
		config = DefaultContextAnalyzerConfig()
	}

	return &contextAnalyzerImpl{
		taskRepo:    taskRepo,
		sessionRepo: sessionRepo,
		patternRepo: patternRepo,
		analytics:   analytics,
		config:      config,
		logger:      logger,
	}
}

// AnalyzeCurrentContext performs comprehensive analysis of current work context
func (ca *contextAnalyzerImpl) AnalyzeCurrentContext(
	ctx context.Context,
	repository string,
) (*entities.WorkContext, error) {
	ca.logger.Info("analyzing current work context", slog.String("repository", repository))

	workContext := entities.NewWorkContext(repository)
	now := time.Now()

	// Get current tasks (in progress or recently created)
	currentTasks, err := ca.getCurrentTasks(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get current tasks: %w", err)
	}
	workContext.CurrentTasks = currentTasks

	// Get recent completed tasks
	recentTasks, err := ca.getRecentCompletedTasks(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent tasks: %w", err)
	}
	workContext.RecentTasks = recentTasks

	// Get current session
	currentSession, err := ca.getCurrentSession(ctx, repository)
	if err != nil {
		ca.logger.Warn("failed to get current session", slog.Any("error", err))
	} else {
		workContext.CurrentSession = currentSession
	}

	// Determine project type
	projectType, err := ca.DetermineProjectType(ctx, repository)
	if err != nil {
		ca.logger.Warn("failed to determine project type", slog.Any("error", err))
		projectType = "general"
	}
	workContext.ProjectType = projectType

	// Analyze time context
	workContext.TimeOfDay = ca.determineTimeOfDay(now)
	workContext.DayOfWeek = now.Weekday().String()

	// Get working hours
	sessions, err := ca.getRecentSessions(ctx, repository, 30) // Last 30 days
	if err == nil {
		workContext.WorkingHours = ca.AnalyzeWorkingHours(sessions)
	}

	// Get recent patterns
	patterns, err := ca.getActivePatterns(ctx, repository)
	if err != nil {
		ca.logger.Warn("failed to get active patterns", slog.Any("error", err))
	} else {
		workContext.ActivePatterns = patterns
	}

	// Calculate velocity
	velocity, err := ca.AnalyzeVelocityTrends(ctx, repository, ca.config.VelocityWindowDays)
	if err != nil {
		ca.logger.Warn("failed to calculate velocity", slog.Any("error", err))
	} else {
		workContext.Velocity = velocity
	}

	// Calculate focus level
	if currentSession != nil {
		workContext.FocusLevel = ca.CalculateFocusLevel(currentSession)
	}

	// Calculate energy level (from current session or default)
	if currentSession != nil && currentSession.EnergyLevel.Overall > 0 {
		workContext.EnergyLevel = currentSession.EnergyLevel.Overall
	} else {
		workContext.EnergyLevel = ca.estimateEnergyLevel(now, sessions)
	}

	// Calculate productivity score (recent average)
	workContext.ProductivityScore = ca.calculateRecentProductivity(sessions)

	// Detect stress indicators
	stressLevel, stressIndicators, err := ca.CalculateStressLevel(ctx, repository)
	if err != nil {
		ca.logger.Warn("failed to calculate stress level", slog.Any("error", err))
	} else {
		workContext.StressIndicators = stressIndicators
	}

	// Get current goals
	goals, err := ca.AnalyzeCurrentGoals(ctx, repository)
	if err != nil {
		ca.logger.Warn("failed to analyze current goals", slog.Any("error", err))
	} else {
		workContext.Goals = goals
	}

	// Detect constraints
	constraints, err := ca.DetectWorkConstraints(ctx, workContext)
	if err != nil {
		ca.logger.Warn("failed to detect work constraints", slog.Any("error", err))
	} else {
		workContext.Constraints = constraints
	}

	// Set environment if available from current session
	if currentSession != nil && currentSession.Environment != nil {
		workContext.Environment = currentSession.Environment
	}

	// Add metadata
	workContext.Metadata["stress_level"] = stressLevel
	workContext.Metadata["analysis_depth"] = "comprehensive"
	workContext.Metadata["context_factors"] = len(currentTasks) + len(recentTasks) + len(patterns)

	ca.logger.Info("context analysis completed",
		slog.Int("current_tasks", len(workContext.CurrentTasks)),
		slog.Int("recent_tasks", len(workContext.RecentTasks)),
		slog.Int("active_patterns", len(workContext.ActivePatterns)),
		slog.Float64("focus_level", workContext.FocusLevel),
		slog.Float64("energy_level", workContext.EnergyLevel))

	return workContext, nil
}

// ExtractKeywords extracts relevant keywords from a list of tasks
func (ca *contextAnalyzerImpl) ExtractKeywords(tasks []*entities.Task) []string {
	if len(tasks) == 0 {
		return []string{}
	}

	keywordFreq := make(map[string]int)

	for _, task := range tasks {
		words := ca.extractWordsFromTask(task)
		for _, word := range words {
			if ca.isValidKeyword(word) {
				keywordFreq[word]++
			}
		}
	}

	// Sort by frequency and return top keywords
	type keywordPair struct {
		word  string
		count int
	}

	var pairs []keywordPair
	for word, count := range keywordFreq {
		pairs = append(pairs, keywordPair{word, count})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})

	// Return top keywords
	maxKeywords := ca.config.KeywordMaxCount
	if len(pairs) < maxKeywords {
		maxKeywords = len(pairs)
	}

	keywords := make([]string, maxKeywords)
	for i := 0; i < maxKeywords; i++ {
		keywords[i] = pairs[i].word
	}

	return keywords
}

// DetermineProjectType determines the type of project based on tasks and patterns
func (ca *contextAnalyzerImpl) DetermineProjectType(
	ctx context.Context,
	repository string,
) (string, error) {
	// Get recent tasks for analysis
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30) // Last 30 days

	tasks, err := ca.taskRepo.FindByTimeRange(ctx, repository, startTime, endTime)
	if err != nil {
		return "", fmt.Errorf("failed to get tasks for project type analysis: %w", err)
	}

	if len(tasks) == 0 {
		return "general", nil
	}

	// Analyze task types and content
	typeFreq := make(map[string]int)
	contentAnalysis := make(map[string]int)

	for _, task := range tasks {
		// Count task types
		if task.Type != "" {
			typeFreq[task.Type]++
		}

		// Analyze content for project indicators
		content := strings.ToLower(task.Content)
		for _, indicator := range ca.getProjectTypeIndicators() {
			if strings.Contains(content, indicator.keyword) {
				contentAnalysis[indicator.projectType]++
			}
		}
	}

	// Determine project type based on frequency
	if projectType := ca.findMostFrequentProjectType(contentAnalysis); projectType != "" {
		return projectType, nil
	}

	if projectType := ca.findMostFrequentTaskType(typeFreq); projectType != "" {
		return projectType, nil
	}

	return "general", nil
}

// CalculateFocusLevel calculates focus level based on session data
func (ca *contextAnalyzerImpl) CalculateFocusLevel(session *entities.Session) float64 {
	if session == nil {
		return 0.5 // Default moderate focus
	}

	// Use the session's calculated focus score
	return session.FocusScore
}

// CalculateStressLevel calculates current stress level and identifies stress indicators
func (ca *contextAnalyzerImpl) CalculateStressLevel(
	ctx context.Context,
	repository string,
) (float64, []entities.StressIndicator, error) {
	var indicators []entities.StressIndicator
	var stressFactors []float64

	// Check for overdue tasks
	overdueTasks, err := ca.getOverdueTasks(ctx, repository)
	if err == nil && len(overdueTasks) > 0 {
		severity := math.Min(1.0, float64(len(overdueTasks))/10.0) // Cap at 10 overdue tasks
		indicators = append(indicators, entities.StressIndicator{
			Type:        "overdue_tasks",
			Severity:    severity,
			Description: fmt.Sprintf("%d overdue tasks", len(overdueTasks)),
			Impact:      "productivity",
			DetectedAt:  time.Now(),
		})
		stressFactors = append(stressFactors, severity)
	}

	// Check velocity trends
	velocity, err := ca.AnalyzeVelocityTrends(ctx, repository, 7)
	if err == nil {
		avgVelocity, err := ca.getAverageVelocity(ctx, repository, 30)
		if err == nil && avgVelocity > 0 {
			velocityRatio := velocity / avgVelocity
			if velocityRatio > 1.5 { // Working 50% faster than normal
				severity := math.Min(1.0, (velocityRatio-1.0)/1.0) // Normalize
				indicators = append(indicators, entities.StressIndicator{
					Type:        "high_velocity",
					Severity:    severity,
					Description: fmt.Sprintf("Working %.0f%% faster than normal", (velocityRatio-1)*100),
					Impact:      "wellbeing",
					DetectedAt:  time.Now(),
				})
				stressFactors = append(stressFactors, severity*0.7) // Weight slightly lower
			}
		}
	}

	// Check for long work periods without breaks
	currentSession, err := ca.getCurrentSession(ctx, repository)
	if err == nil && currentSession != nil {
		if currentSession.Duration > 4*time.Hour && len(currentSession.WorkPeriods) == 0 {
			severity := math.Min(1.0, currentSession.Duration.Hours()/8.0) // Cap at 8 hours
			indicators = append(indicators, entities.StressIndicator{
				Type:        "long_work_period",
				Severity:    severity,
				Description: fmt.Sprintf("Working for %.1f hours without recorded breaks", currentSession.Duration.Hours()),
				Impact:      "wellbeing",
				DetectedAt:  time.Now(),
			})
			stressFactors = append(stressFactors, severity)
		}
	}

	// Check for high interruption rate
	if currentSession != nil && len(currentSession.Interruptions) > 0 {
		interruptionRate := float64(len(currentSession.Interruptions)) / currentSession.Duration.Hours()
		if interruptionRate > 2.0 { // More than 2 interruptions per hour
			severity := math.Min(1.0, interruptionRate/5.0) // Cap at 5 per hour
			indicators = append(indicators, entities.StressIndicator{
				Type:        "high_interruptions",
				Severity:    severity,
				Description: fmt.Sprintf("%.1f interruptions per hour", interruptionRate),
				Impact:      "productivity",
				DetectedAt:  time.Now(),
			})
			stressFactors = append(stressFactors, severity*0.8)
		}
	}

	// Calculate overall stress level
	stressLevel := 0.0
	if len(stressFactors) > 0 {
		for _, factor := range stressFactors {
			stressLevel += factor
		}
		stressLevel = stressLevel / float64(len(stressFactors))

		// Apply non-linear scaling to emphasize high stress
		stressLevel = math.Pow(stressLevel, 1.5)

		// Cap at 1.0
		if stressLevel > 1.0 {
			stressLevel = 1.0
		}
	}

	return stressLevel, indicators, nil
}

// PredictOptimalTaskType predicts the best task type for current context
func (ca *contextAnalyzerImpl) PredictOptimalTaskType(context *entities.WorkContext) string {
	if context == nil {
		return "general"
	}

	// Score different task types based on context
	scores := make(map[string]float64)

	// Consider time of day
	timeScores := ca.getTimeOfDayScores(context.TimeOfDay)
	for taskType, score := range timeScores {
		scores[taskType] = score * 0.3 // 30% weight
	}

	// Consider energy level
	energyScores := ca.getEnergyBasedScores(context.EnergyLevel)
	for taskType, score := range energyScores {
		scores[taskType] += score * 0.25 // 25% weight
	}

	// Consider focus level
	focusScores := ca.getFocusBasedScores(context.FocusLevel)
	for taskType, score := range focusScores {
		scores[taskType] += score * 0.25 // 25% weight
	}

	// Consider recent patterns
	patternScores := ca.getPatternBasedScores(context.ActivePatterns)
	for taskType, score := range patternScores {
		scores[taskType] += score * 0.2 // 20% weight
	}

	// Find highest scoring task type
	bestType := "general"
	bestScore := 0.0

	for taskType, score := range scores {
		if score > bestScore {
			bestScore = score
			bestType = taskType
		}
	}

	return bestType
}

// PredictProductivity predicts productivity based on current context
func (ca *contextAnalyzerImpl) PredictProductivity(context *entities.WorkContext) float64 {
	if context == nil {
		return 0.5
	}

	// Base productivity on various factors
	factors := []float64{
		context.EnergyLevel * 0.3,                                  // 30% energy
		context.FocusLevel * 0.25,                                  // 25% focus
		(1.0 - ca.getStressImpact(context.StressIndicators)) * 0.2, // 20% stress (inverted)
		ca.getTimeOptimality(context.TimeOfDay) * 0.15,             // 15% time optimality
		ca.getPatternProductivity(context.ActivePatterns) * 0.1,    // 10% pattern history
	}

	productivity := 0.0
	for _, factor := range factors {
		productivity += factor
	}

	// Ensure between 0 and 1
	if productivity > 1.0 {
		productivity = 1.0
	} else if productivity < 0.0 {
		productivity = 0.0
	}

	return productivity
}

// DetectWorkPatterns detects patterns in work behavior
func (ca *contextAnalyzerImpl) DetectWorkPatterns(
	ctx context.Context,
	repository string,
	days int,
) ([]*entities.TaskPattern, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	sessions, err := ca.sessionRepo.FindByTimeRange(ctx, repository, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions for pattern detection: %w", err)
	}

	patterns := make([]*entities.TaskPattern, 0)

	// Detect temporal patterns
	temporalPattern := ca.detectTemporalWorkPattern(sessions)
	if temporalPattern != nil {
		patterns = append(patterns, temporalPattern)
	}

	// Detect productivity patterns
	productivityPattern := ca.detectProductivityPattern(sessions)
	if productivityPattern != nil {
		patterns = append(patterns, productivityPattern)
	}

	return patterns, nil
}

// AnalyzeWorkingHours analyzes working hours patterns from sessions
func (ca *contextAnalyzerImpl) AnalyzeWorkingHours(sessions []*entities.Session) *entities.WorkingHours {
	if len(sessions) == 0 {
		return ca.getDefaultWorkingHours()
	}

	// Analyze start and end times
	var startTimes, endTimes []time.Time
	energyPattern := make(map[string]float64)
	weekDays := make(map[time.Weekday]bool)

	for _, session := range sessions {
		startTimes = append(startTimes, session.StartTime)
		if session.EndTime != nil {
			endTimes = append(endTimes, *session.EndTime)
		}

		// Track working days
		weekDays[session.StartTime.Weekday()] = true

		// Track energy patterns
		if session.EnergyLevel.Overall > 0 {
			hour := fmt.Sprintf("%02d:00", session.StartTime.Hour())
			energyPattern[hour] = session.EnergyLevel.Overall
		}
	}

	// Calculate typical start and end times
	avgStartTime := ca.calculateAverageTime(startTimes)
	avgEndTime := ca.calculateAverageTime(endTimes)

	// Determine working days
	var workingDays []string
	for day := range weekDays {
		if day != time.Saturday && day != time.Sunday { // Exclude weekends by default
			workingDays = append(workingDays, day.String())
		}
	}

	// Find peak hours
	peakHours := ca.findPeakHours(energyPattern)

	return &entities.WorkingHours{
		StartTime:     avgStartTime,
		EndTime:       avgEndTime,
		BreakDuration: time.Hour, // Default 1 hour lunch break
		TimeZone:      "Local",
		WeekDays:      workingDays,
		PeakHours:     peakHours,
		EnergyPattern: ca.normalizeEnergyPattern(energyPattern),
		Preferences: entities.WorkPreferences{
			PreferredTaskLength: time.Hour,
			MaxFocusTime:        2 * time.Hour,
			PreferredBreakType:  "short",
			WorkStyle:           "focused",
		},
	}
}

// DetectEnergyPatterns detects energy patterns throughout the day
func (ca *contextAnalyzerImpl) DetectEnergyPatterns(sessions []*entities.Session) map[int]float64 {
	hourlyEnergy := make(map[int][]float64)

	for _, session := range sessions {
		if session.EnergyLevel.Overall > 0 {
			hour := session.StartTime.Hour()
			hourlyEnergy[hour] = append(hourlyEnergy[hour], session.EnergyLevel.Overall)
		}
	}

	// Calculate average energy for each hour
	patterns := make(map[int]float64)
	for hour, energyLevels := range hourlyEnergy {
		if len(energyLevels) > 0 {
			sum := 0.0
			for _, energy := range energyLevels {
				sum += energy
			}
			patterns[hour] = sum / float64(len(energyLevels))
		}
	}

	return patterns
}

// AnalyzeVelocityTrends analyzes velocity trends over time
func (ca *contextAnalyzerImpl) AnalyzeVelocityTrends(
	ctx context.Context,
	repository string,
	days int,
) (float64, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	tasks, err := ca.taskRepo.FindByTimeRange(ctx, repository, startTime, endTime)
	if err != nil {
		return 0, fmt.Errorf("failed to get tasks for velocity analysis: %w", err)
	}

	// Count completed tasks
	completedTasks := 0
	for _, task := range tasks {
		if task.Status == "completed" {
			completedTasks++
		}
	}

	// Calculate velocity (tasks per day)
	if days > 0 {
		return float64(completedTasks) / float64(days), nil
	}

	return 0, nil
}

// AnalyzeCurrentGoals analyzes current goals and objectives
func (ca *contextAnalyzerImpl) AnalyzeCurrentGoals(
	ctx context.Context,
	repository string,
) ([]entities.SessionGoal, error) {
	// Get current session to check for active goals
	currentSession, err := ca.getCurrentSession(ctx, repository)
	if err != nil || currentSession == nil {
		return []entities.SessionGoal{}, nil
	}

	// Return goals from current session
	return currentSession.Goals, nil
}

// DetectWorkConstraints detects current work constraints
func (ca *contextAnalyzerImpl) DetectWorkConstraints(
	ctx context.Context,
	workContext *entities.WorkContext,
) ([]entities.WorkConstraint, error) {
	var constraints []entities.WorkConstraint

	// Time constraints based on working hours
	if workContext.WorkingHours != nil {
		if ca.isOutsideWorkingHours(workContext.WorkingHours) {
			constraints = append(constraints, entities.WorkConstraint{
				Type:        "time",
				Description: "Outside normal working hours",
				Impact:      0.3,
				Metadata:    make(map[string]interface{}),
			})
		}
	}

	// Energy constraints
	if workContext.EnergyLevel < 0.3 {
		constraints = append(constraints, entities.WorkConstraint{
			Type:        "energy",
			Description: "Low energy level",
			Impact:      1.0 - workContext.EnergyLevel,
			Metadata:    make(map[string]interface{}),
		})
	}

	// Focus constraints
	if workContext.FocusLevel < 0.4 {
		constraints = append(constraints, entities.WorkConstraint{
			Type:        "focus",
			Description: "Low focus level",
			Impact:      1.0 - workContext.FocusLevel,
			Metadata:    make(map[string]interface{}),
		})
	}

	// Stress constraints
	if workContext.IsHighStress() {
		constraints = append(constraints, entities.WorkConstraint{
			Type:        "stress",
			Description: "High stress indicators detected",
			Impact:      0.8,
			Duration:    2 * time.Hour, // Estimated recovery time
			Metadata:    make(map[string]interface{}),
		})
	}

	return constraints, nil
}

// Helper methods

func (ca *contextAnalyzerImpl) getCurrentTasks(ctx context.Context, repository string) ([]*entities.Task, error) {
	// Get tasks that are in progress or created recently
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -1) // Last 24 hours

	allTasks, err := ca.taskRepo.FindByTimeRange(ctx, repository, startTime, endTime)
	if err != nil {
		return nil, err
	}

	var currentTasks []*entities.Task
	for _, task := range allTasks {
		if task.Status == "in_progress" ||
			(task.Status == "pending" && time.Since(task.CreatedAt) < 24*time.Hour) {
			currentTasks = append(currentTasks, task)
		}
	}

	return currentTasks, nil
}

func (ca *contextAnalyzerImpl) getRecentCompletedTasks(ctx context.Context, repository string) ([]*entities.Task, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7) // Last week

	tasks, err := ca.taskRepo.FindByTimeRange(ctx, repository, startTime, endTime)
	if err != nil {
		return nil, err
	}

	var completedTasks []*entities.Task
	for _, task := range tasks {
		if task.Status == "completed" {
			completedTasks = append(completedTasks, task)
		}
	}

	// Sort by completion time and return most recent
	sort.Slice(completedTasks, func(i, j int) bool {
		return completedTasks[i].UpdatedAt.After(completedTasks[j].UpdatedAt)
	})

	maxTasks := ca.config.MaxRecentTasks
	if len(completedTasks) < maxTasks {
		maxTasks = len(completedTasks)
	}

	return completedTasks[:maxTasks], nil
}

func (ca *contextAnalyzerImpl) getCurrentSession(ctx context.Context, repository string) (*entities.Session, error) {
	// Get the most recent session for the repository
	sessions, err := ca.sessionRepo.FindByRepository(ctx, repository)
	if err != nil {
		return nil, err
	}

	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found")
	}

	session := sessions[0]

	// Check if session is still active (no end time or ended recently)
	if session.EndTime == nil || time.Since(*session.EndTime) < 4*time.Hour {
		return session, nil
	}

	return nil, fmt.Errorf("no active session found")
}

func (ca *contextAnalyzerImpl) getRecentSessions(ctx context.Context, repository string, days int) ([]*entities.Session, error) {
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	return ca.sessionRepo.FindByTimeRange(ctx, repository, startTime, endTime)
}

func (ca *contextAnalyzerImpl) getActivePatterns(ctx context.Context, repository string) ([]*entities.TaskPattern, error) {
	patterns, err := ca.patternRepo.FindByRepository(ctx, repository)
	if err != nil {
		return nil, err
	}

	// Filter for active, confident patterns
	var activePatterns []*entities.TaskPattern
	for _, pattern := range patterns {
		if pattern.Confidence >= ca.config.PatternConfidence &&
			!pattern.IsExpired(30*24*time.Hour) { // Not expired in last 30 days
			activePatterns = append(activePatterns, pattern)
		}
	}

	// Sort by confidence and return top patterns
	sort.Slice(activePatterns, func(i, j int) bool {
		return activePatterns[i].Confidence > activePatterns[j].Confidence
	})

	maxPatterns := ca.config.MaxRecentPatterns
	if len(activePatterns) < maxPatterns {
		maxPatterns = len(activePatterns)
	}

	return activePatterns[:maxPatterns], nil
}

func (ca *contextAnalyzerImpl) determineTimeOfDay(now time.Time) string {
	hour := now.Hour()

	switch {
	case hour >= 5 && hour < 12:
		return "morning"
	case hour >= 12 && hour < 17:
		return "afternoon"
	case hour >= 17 && hour < 21:
		return "evening"
	default:
		return "night"
	}
}

func (ca *contextAnalyzerImpl) estimateEnergyLevel(now time.Time, sessions []*entities.Session) float64 {
	// Default energy pattern based on time of day
	hour := now.Hour()

	// Typical energy curve
	energyCurve := map[int]float64{
		6: 0.6, 7: 0.7, 8: 0.8, 9: 0.9, 10: 1.0, 11: 0.9,
		12: 0.7, 13: 0.6, 14: 0.8, 15: 0.9, 16: 0.8, 17: 0.7,
		18: 0.6, 19: 0.5, 20: 0.4, 21: 0.3, 22: 0.2, 23: 0.1,
		0: 0.1, 1: 0.1, 2: 0.1, 3: 0.1, 4: 0.2, 5: 0.4,
	}

	baseEnergy := energyCurve[hour]

	// Adjust based on historical data if available
	if len(sessions) > 0 {
		energyPattern := ca.DetectEnergyPatterns(sessions)
		if historicalEnergy, exists := energyPattern[hour]; exists {
			// Blend historical data with default curve
			baseEnergy = (baseEnergy + historicalEnergy) / 2
		}
	}

	return baseEnergy
}

func (ca *contextAnalyzerImpl) calculateRecentProductivity(sessions []*entities.Session) float64 {
	if len(sessions) == 0 {
		return 0.5 // Default moderate productivity
	}

	// Calculate average productivity from recent sessions
	totalProductivity := 0.0
	count := 0

	for _, session := range sessions {
		if session.ProductivityScore > 0 {
			totalProductivity += session.ProductivityScore
			count++
		}
	}

	if count == 0 {
		return 0.5
	}

	return totalProductivity / float64(count)
}

func (ca *contextAnalyzerImpl) extractWordsFromTask(task *entities.Task) []string {
	text := task.Content
	// Task entity only has Content field
	// if task.Description != "" {
	//	text += " " + task.Description
	// }

	// Simple word extraction
	words := strings.Fields(strings.ToLower(text))

	// Clean words
	var cleanWords []string
	for _, word := range words {
		// Remove punctuation and validate
		cleaned := strings.Trim(word, ".,!?;:()[]{}\"'")
		if ca.isValidKeyword(cleaned) {
			cleanWords = append(cleanWords, cleaned)
		}
	}

	return cleanWords
}

func (ca *contextAnalyzerImpl) isValidKeyword(word string) bool {
	if len(word) < ca.config.KeywordMinLength {
		return false
	}

	// Check if it's a stop word
	stopWords := map[string]bool{
		"the": true, "and": true, "or": true, "but": true, "in": true,
		"on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "this": true, "that": true, "these": true,
		"those": true, "is": true, "are": true, "was": true, "were": true,
		"will": true, "would": true, "could": true, "should": true,
	}

	return !stopWords[word]
}

type projectTypeIndicator struct {
	keyword     string
	projectType string
}

func (ca *contextAnalyzerImpl) getProjectTypeIndicators() []projectTypeIndicator {
	return []projectTypeIndicator{
		{"bug", "bugfix"}, {"fix", "bugfix"}, {"error", "bugfix"},
		{"feature", "feature"}, {"add", "feature"}, {"implement", "feature"},
		{"refactor", "refactor"}, {"improve", "refactor"}, {"optimize", "refactor"},
		{"test", "testing"}, {"spec", "testing"}, {"verify", "testing"},
		{"doc", "documentation"}, {"readme", "documentation"}, {"guide", "documentation"},
		{"deploy", "deployment"}, {"release", "deployment"}, {"publish", "deployment"},
		{"research", "research"}, {"investigate", "research"}, {"explore", "research"},
		{"meeting", "planning"}, {"plan", "planning"}, {"design", "planning"},
	}
}

func (ca *contextAnalyzerImpl) findMostFrequentProjectType(contentAnalysis map[string]int) string {
	maxCount := 0
	mostFrequent := ""

	for projectType, count := range contentAnalysis {
		if count > maxCount {
			maxCount = count
			mostFrequent = projectType
		}
	}

	if maxCount >= 3 { // Minimum threshold
		return mostFrequent
	}

	return ""
}

func (ca *contextAnalyzerImpl) findMostFrequentTaskType(typeFreq map[string]int) string {
	maxCount := 0
	mostFrequent := ""

	for taskType, count := range typeFreq {
		if count > maxCount {
			maxCount = count
			mostFrequent = taskType
		}
	}

	if maxCount >= 2 { // Minimum threshold
		return mostFrequent
	}

	return ""
}

func (ca *contextAnalyzerImpl) getOverdueTasks(ctx context.Context, repository string) ([]*entities.Task, error) {
	// Get all pending and in-progress tasks
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -90) // Last 90 days

	tasks, err := ca.taskRepo.FindByTimeRange(ctx, repository, startTime, endTime)
	if err != nil {
		return nil, err
	}

	var overdueTasks []*entities.Task
	for _, task := range tasks {
		// TODO: Task entity doesn't have DueDate field yet
		// For now, we'll use a simpler approach based on creation time and estimated duration
		if (task.Status == entities.StatusPending || task.Status == entities.StatusInProgress) &&
			task.EstimatedMins > 0 {
			// Consider a task overdue if it's been around longer than its estimated time
			estimatedDuration := time.Duration(task.EstimatedMins) * time.Minute
			if time.Since(task.CreatedAt) > estimatedDuration {
				overdueTasks = append(overdueTasks, task)
			}
		}
	}

	return overdueTasks, nil
}

func (ca *contextAnalyzerImpl) getAverageVelocity(ctx context.Context, repository string, days int) (float64, error) {
	return ca.AnalyzeVelocityTrends(ctx, repository, days)
}

func (ca *contextAnalyzerImpl) getTimeOfDayScores(timeOfDay string) map[string]float64 {
	scores := make(map[string]float64)

	switch timeOfDay {
	case "morning":
		scores["planning"] = 0.9
		scores["learning"] = 0.8
		scores["creative"] = 0.8
		scores["analytical"] = 0.9
		scores["routine"] = 0.6
	case "afternoon":
		scores["implementation"] = 0.9
		scores["collaborative"] = 0.8
		scores["routine"] = 0.8
		scores["testing"] = 0.7
		scores["communication"] = 0.9
	case "evening":
		scores["review"] = 0.8
		scores["documentation"] = 0.7
		scores["planning"] = 0.6
		scores["routine"] = 0.9
		scores["reflection"] = 0.8
	case "night":
		scores["routine"] = 0.5
		scores["creative"] = 0.6
		scores["research"] = 0.7
		scores["learning"] = 0.5
	}

	return scores
}

func (ca *contextAnalyzerImpl) getEnergyBasedScores(energyLevel float64) map[string]float64 {
	scores := make(map[string]float64)

	if energyLevel > 0.8 {
		scores["complex"] = 0.9
		scores["creative"] = 0.9
		scores["challenging"] = 0.9
		scores["learning"] = 0.8
	} else if energyLevel > 0.6 {
		scores["implementation"] = 0.8
		scores["routine"] = 0.9
		scores["testing"] = 0.8
		scores["communication"] = 0.7
	} else if energyLevel > 0.4 {
		scores["routine"] = 0.9
		scores["documentation"] = 0.8
		scores["review"] = 0.8
		scores["organizing"] = 0.9
	} else {
		scores["routine"] = 0.8
		scores["passive"] = 0.9
		scores["reading"] = 0.7
		scores["organizing"] = 0.6
	}

	return scores
}

func (ca *contextAnalyzerImpl) getFocusBasedScores(focusLevel float64) map[string]float64 {
	scores := make(map[string]float64)

	if focusLevel > 0.8 {
		scores["deep_work"] = 0.9
		scores["complex"] = 0.9
		scores["analytical"] = 0.9
		scores["creative"] = 0.8
	} else if focusLevel > 0.6 {
		scores["implementation"] = 0.8
		scores["testing"] = 0.8
		scores["learning"] = 0.7
	} else if focusLevel > 0.4 {
		scores["routine"] = 0.8
		scores["communication"] = 0.9
		scores["collaborative"] = 0.8
	} else {
		scores["routine"] = 0.7
		scores["organizing"] = 0.8
		scores["communication"] = 0.6
		scores["break"] = 0.9
	}

	return scores
}

func (ca *contextAnalyzerImpl) getPatternBasedScores(patterns []*entities.TaskPattern) map[string]float64 {
	scores := make(map[string]float64)

	for _, pattern := range patterns {
		for _, step := range pattern.Sequence {
			scores[step.TaskType] += pattern.Confidence * 0.1
		}
	}

	return scores
}

func (ca *contextAnalyzerImpl) getStressImpact(indicators []entities.StressIndicator) float64 {
	if len(indicators) == 0 {
		return 0.0
	}

	totalImpact := 0.0
	for _, indicator := range indicators {
		totalImpact += indicator.Severity
	}

	avgImpact := totalImpact / float64(len(indicators))
	return math.Min(1.0, avgImpact)
}

func (ca *contextAnalyzerImpl) getTimeOptimality(timeOfDay string) float64 {
	// Optimal time scores based on general productivity research
	switch timeOfDay {
	case "morning":
		return 0.9
	case "afternoon":
		return 0.8
	case "evening":
		return 0.6
	case "night":
		return 0.3
	default:
		return 0.5
	}
}

func (ca *contextAnalyzerImpl) getPatternProductivity(patterns []*entities.TaskPattern) float64 {
	if len(patterns) == 0 {
		return 0.5
	}

	totalSuccessRate := 0.0
	for _, pattern := range patterns {
		totalSuccessRate += pattern.SuccessRate
	}

	return totalSuccessRate / float64(len(patterns))
}

func (ca *contextAnalyzerImpl) detectTemporalWorkPattern(sessions []*entities.Session) *entities.TaskPattern {
	if len(sessions) < 5 {
		return nil
	}

	// Analyze work start times
	startHours := make(map[int]int)
	for _, session := range sessions {
		hour := session.StartTime.Hour()
		startHours[hour]++
	}

	// Find most common start time
	maxCount := 0
	commonHour := 9 // Default
	for hour, count := range startHours {
		if count > maxCount {
			maxCount = count
			commonHour = hour
		}
	}

	if maxCount < 3 {
		return nil
	}

	pattern := &entities.TaskPattern{
		ID:          fmt.Sprintf("temporal_%d", time.Now().Unix()),
		Type:        entities.PatternTypeTemporal,
		Name:        fmt.Sprintf("Work Start Pattern - %d:00", commonHour),
		Description: fmt.Sprintf("Typically starts work around %d:00", commonHour),
		Confidence:  float64(maxCount) / float64(len(sessions)),
		Frequency:   float64(maxCount) / float64(len(sessions)),
		Occurrences: maxCount,
		FirstSeen:   time.Now().AddDate(0, 0, -len(sessions)),
		LastSeen:    time.Now(),
		Metadata: map[string]interface{}{
			"pattern_type": "work_start_time",
			"peak_hour":    commonHour,
			"sessions":     len(sessions),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return pattern
}

func (ca *contextAnalyzerImpl) detectProductivityPattern(sessions []*entities.Session) *entities.TaskPattern {
	if len(sessions) < 7 {
		return nil
	}

	// Analyze productivity by day of week
	dayProductivity := make(map[time.Weekday][]float64)
	for _, session := range sessions {
		day := session.StartTime.Weekday()
		dayProductivity[day] = append(dayProductivity[day], session.ProductivityScore)
	}

	// Find most productive day
	bestDay := time.Monday
	bestScore := 0.0

	for day, scores := range dayProductivity {
		if len(scores) > 0 {
			avg := 0.0
			for _, score := range scores {
				avg += score
			}
			avg /= float64(len(scores))

			if avg > bestScore {
				bestScore = avg
				bestDay = day
			}
		}
	}

	if bestScore < 0.6 {
		return nil
	}

	pattern := &entities.TaskPattern{
		ID:          fmt.Sprintf("productivity_%d", time.Now().Unix()),
		Type:        entities.PatternTypeTemporal,
		Name:        fmt.Sprintf("High Productivity - %s", bestDay.String()),
		Description: fmt.Sprintf("Typically most productive on %s (%.1f%% average)", bestDay.String(), bestScore*100),
		Confidence:  bestScore,
		Frequency:   1.0 / 7.0, // Once per week
		Occurrences: len(dayProductivity[bestDay]),
		FirstSeen:   time.Now().AddDate(0, 0, -len(sessions)),
		LastSeen:    time.Now(),
		Metadata: map[string]interface{}{
			"pattern_type":     "weekly_productivity",
			"best_day":         bestDay.String(),
			"avg_productivity": bestScore,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return pattern
}

func (ca *contextAnalyzerImpl) getDefaultWorkingHours() *entities.WorkingHours {
	return &entities.WorkingHours{
		StartTime:     "09:00",
		EndTime:       "17:00",
		BreakDuration: time.Hour,
		TimeZone:      "Local",
		WeekDays:      []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
		PeakHours:     []string{"10:00", "11:00", "14:00", "15:00"},
		EnergyPattern: map[string]float64{
			"09:00": 0.8, "10:00": 0.9, "11:00": 0.9, "12:00": 0.7,
			"13:00": 0.6, "14:00": 0.8, "15:00": 0.8, "16:00": 0.7,
		},
		Preferences: entities.WorkPreferences{
			PreferredTaskLength: time.Hour,
			MaxFocusTime:        2 * time.Hour,
			PreferredBreakType:  "short",
			WorkStyle:           "focused",
		},
	}
}

func (ca *contextAnalyzerImpl) calculateAverageTime(times []time.Time) string {
	if len(times) == 0 {
		return "09:00"
	}

	totalMinutes := 0
	for _, t := range times {
		totalMinutes += t.Hour()*60 + t.Minute()
	}

	avgMinutes := totalMinutes / len(times)
	hours := avgMinutes / 60
	minutes := avgMinutes % 60

	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

func (ca *contextAnalyzerImpl) findPeakHours(energyPattern map[string]float64) []string {
	type hourEnergy struct {
		hour   string
		energy float64
	}

	var pairs []hourEnergy
	for hour, energy := range energyPattern {
		pairs = append(pairs, hourEnergy{hour, energy})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].energy > pairs[j].energy
	})

	// Return top 4 hours
	var peakHours []string
	maxHours := 4
	if len(pairs) < maxHours {
		maxHours = len(pairs)
	}

	for i := 0; i < maxHours; i++ {
		peakHours = append(peakHours, pairs[i].hour)
	}

	return peakHours
}

func (ca *contextAnalyzerImpl) normalizeEnergyPattern(energyPattern map[string]float64) map[string]float64 {
	// Convert to hour-based map
	normalized := make(map[string]float64)
	for hour, energy := range energyPattern {
		normalized[hour] = energy
	}
	return normalized
}

func (ca *contextAnalyzerImpl) isOutsideWorkingHours(workingHours *entities.WorkingHours) bool {
	now := time.Now()
	currentHour := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())

	// Simple comparison (could be improved with proper time parsing)
	return currentHour < workingHours.StartTime || currentHour > workingHours.EndTime
}
