package entities

import (
	"encoding/json"
	"time"

	"github.com/go-playground/validator/v10"
)

// Session represents a work session with productivity metrics
type Session struct {
	ID                string                 `json:"id" validate:"required,uuid"`
	Repository        string                 `json:"repository" validate:"required"`
	StartTime         time.Time              `json:"start_time"`
	EndTime           *time.Time             `json:"end_time,omitempty"`
	Duration          time.Duration          `json:"duration"`
	TasksStarted      int                    `json:"tasks_started"`
	TasksCompleted    int                    `json:"tasks_completed"`
	TasksInProgress   int                    `json:"tasks_in_progress"`
	TasksByPriority   map[string]int         `json:"tasks_by_priority"` // Priority -> count
	TasksByType       map[string]int         `json:"tasks_by_type"`     // Type -> count
	TaskDurations     []TaskDurationEntry    `json:"task_durations"`
	WorkPeriods       []WorkPeriod           `json:"work_periods"`
	Interruptions     []Interruption         `json:"interruptions"`
	FocusScore        float64                `json:"focus_score"`        // 0-1 based on interruptions
	ProductivityScore float64                `json:"productivity_score"` // 0-1 overall productivity
	VelocityScore     float64                `json:"velocity_score"`     // Tasks per hour
	QualityScore      float64                `json:"quality_score"`      // Based on task completion quality
	EnergyLevel       EnergyLevel            `json:"energy_level"`
	MoodRating        int                    `json:"mood_rating"` // 1-5 user-reported mood
	Environment       *WorkEnvironment       `json:"environment,omitempty"`
	Goals             []SessionGoal          `json:"goals"`
	Achievements      []Achievement          `json:"achievements"`
	Notes             string                 `json:"notes,omitempty"`
	Metadata          map[string]interface{} `json:"metadata"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// TaskDurationEntry tracks how long tasks took to complete
type TaskDurationEntry struct {
	TaskID      string        `json:"task_id"`
	TaskType    string        `json:"task_type"`
	Priority    string        `json:"priority"`
	Estimated   time.Duration `json:"estimated"`
	Actual      time.Duration `json:"actual"`
	Efficiency  float64       `json:"efficiency"` // actual/estimated
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
	Quality     int           `json:"quality"` // 1-5 rating
}

// WorkPeriod represents a continuous period of focused work
type WorkPeriod struct {
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	Duration      time.Duration `json:"duration"`
	TasksFocused  []string      `json:"tasks_focused"`        // Task IDs worked on
	Intensity     float64       `json:"intensity"`            // 0-1 work intensity
	BreakType     string        `json:"break_type,omitempty"` // Type of break after period
	BreakDuration time.Duration `json:"break_duration,omitempty"`
}

// Interruption tracks interruptions during work
type Interruption struct {
	Timestamp time.Time     `json:"timestamp"`
	Type      string        `json:"type"` // notification, meeting, break, etc.
	Duration  time.Duration `json:"duration"`
	Source    string        `json:"source,omitempty"`  // What caused the interruption
	Impact    float64       `json:"impact"`            // 0-1 how much it affected focus
	Recovery  time.Duration `json:"recovery"`          // Time to regain focus
	TaskID    string        `json:"task_id,omitempty"` // Task being worked on
}

// EnergyLevel represents the user's energy state
type EnergyLevel struct {
	Physical  int       `json:"physical"` // 1-5 physical energy
	Mental    int       `json:"mental"`   // 1-5 mental energy
	Overall   float64   `json:"overall"`  // 0-1 calculated overall energy
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkEnvironment captures the work environment context
type WorkEnvironment struct {
	Location     string                 `json:"location"`    // office, home, cafe, etc.
	TimeOfDay    string                 `json:"time_of_day"` // morning, afternoon, evening
	DayOfWeek    string                 `json:"day_of_week"`
	Weather      string                 `json:"weather,omitempty"`
	Noise        int                    `json:"noise"`        // 1-5 noise level
	Comfort      int                    `json:"comfort"`      // 1-5 comfort level
	Tools        []string               `json:"tools"`        // Tools/apps used
	Distractions []string               `json:"distractions"` // Types of distractions present
	Metadata     map[string]interface{} `json:"metadata"`
}

// SessionGoal represents a goal for the session
type SessionGoal struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Type        string     `json:"type"`     // task, learning, exploration
	Target      int        `json:"target"`   // Target number (tasks, hours, etc.)
	Achieved    int        `json:"achieved"` // Actual achievement
	Completed   bool       `json:"completed"`
	Priority    string     `json:"priority"`
	SetAt       time.Time  `json:"set_at"`
	DueAt       *time.Time `json:"due_at,omitempty"`
}

// Achievement represents an accomplishment during the session
type Achievement struct {
	Type        string                 `json:"type"` // milestone, streak, efficiency, etc.
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Value       float64                `json:"value"` // Numeric value associated
	Metadata    map[string]interface{} `json:"metadata"`
	EarnedAt    time.Time              `json:"earned_at"`
}

// WorkingHours represents typical working hours and preferences
type WorkingHours struct {
	StartTime     string             `json:"start_time"`     // e.g., "09:00"
	EndTime       string             `json:"end_time"`       // e.g., "17:00"
	BreakDuration time.Duration      `json:"break_duration"` // Lunch break duration
	TimeZone      string             `json:"timezone"`
	WeekDays      []string           `json:"week_days"`      // Working days
	PeakHours     []string           `json:"peak_hours"`     // Most productive hours
	EnergyPattern map[string]float64 `json:"energy_pattern"` // Hour -> energy level
	Preferences   WorkPreferences    `json:"preferences"`
}

// WorkPreferences represents user work preferences
type WorkPreferences struct {
	PreferredTaskLength time.Duration `json:"preferred_task_length"`
	MaxFocusTime        time.Duration `json:"max_focus_time"`
	PreferredBreakType  string        `json:"preferred_break_type"`
	NotificationFreq    string        `json:"notification_freq"`
	WorkStyle           string        `json:"work_style"` // focused, multitask, flexible
	PreferredTaskTypes  []string      `json:"preferred_task_types"`
}

// SessionProductivityMetrics aggregates productivity data for a session
type SessionProductivityMetrics struct {
	CompletionRate   float64   `json:"completion_rate"`   // Tasks completed / started
	EfficiencyScore  float64   `json:"efficiency_score"`  // Actual vs estimated time
	FocusScore       float64   `json:"focus_score"`       // Based on interruptions
	ConsistencyScore float64   `json:"consistency_score"` // Regularity of work patterns
	QualityScore     float64   `json:"quality_score"`     // Based on task quality ratings
	VelocityTrend    float64   `json:"velocity_trend"`    // Change in tasks/hour over time
	BurnoutRisk      float64   `json:"burnout_risk"`      // 0-1 risk assessment
	OptimalHours     []string  `json:"optimal_hours"`     // Best performing time slots
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	CalculatedAt     time.Time `json:"calculated_at"`
}

// NewSession creates a new work session
func NewSession(repository string) *Session {
	now := time.Now()
	return &Session{
		Repository:      repository,
		StartTime:       now,
		TasksByPriority: make(map[string]int),
		TasksByType:     make(map[string]int),
		TaskDurations:   make([]TaskDurationEntry, 0),
		WorkPeriods:     make([]WorkPeriod, 0),
		Interruptions:   make([]Interruption, 0),
		Goals:           make([]SessionGoal, 0),
		Achievements:    make([]Achievement, 0),
		Metadata:        make(map[string]interface{}),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// EndSession marks the session as ended and calculates final metrics
func (s *Session) EndSession() {
	now := time.Now()
	s.EndTime = &now
	s.Duration = now.Sub(s.StartTime)
	s.UpdatedAt = now

	// Calculate final scores
	s.calculateFinalScores()
}

// AddTask records that a task was started in this session
func (s *Session) AddTask(task *Task) {
	s.TasksStarted++

	// Update counters
	if task.Priority != "" {
		s.TasksByPriority[string(task.Priority)]++
	}
	// Since Task doesn't have a Type field, we'll use tags instead
	if len(task.Tags) > 0 {
		for _, tag := range task.Tags {
			s.TasksByType[tag]++
		}
	}

	s.UpdatedAt = time.Now()
}

// CompleteTask records that a task was completed in this session
func (s *Session) CompleteTask(taskID string, duration time.Duration, quality int) {
	s.TasksCompleted++

	// Add duration entry
	entry := TaskDurationEntry{
		TaskID:      taskID,
		Actual:      duration,
		Quality:     quality,
		CompletedAt: time.Now(),
	}
	s.TaskDurations = append(s.TaskDurations, entry)

	s.UpdatedAt = time.Now()
}

// AddInterruption records an interruption during the session
func (s *Session) AddInterruption(interruption Interruption) {
	s.Interruptions = append(s.Interruptions, interruption)
	s.UpdatedAt = time.Now()
}

// StartWorkPeriod begins a new focused work period
func (s *Session) StartWorkPeriod() *WorkPeriod {
	period := WorkPeriod{
		StartTime:    time.Now(),
		TasksFocused: make([]string, 0),
	}
	return &period
}

// EndWorkPeriod completes a work period and adds it to the session
func (s *Session) EndWorkPeriod(period *WorkPeriod, intensity float64) {
	period.EndTime = time.Now()
	period.Duration = period.EndTime.Sub(period.StartTime)
	period.Intensity = intensity

	s.WorkPeriods = append(s.WorkPeriods, *period)
	s.UpdatedAt = time.Now()
}

// AddGoal adds a goal to the session
func (s *Session) AddGoal(description, goalType string, target int, priority string) {
	goal := SessionGoal{
		ID:          generateID(),
		Description: description,
		Type:        goalType,
		Target:      target,
		Priority:    priority,
		SetAt:       time.Now(),
	}
	s.Goals = append(s.Goals, goal)
	s.UpdatedAt = time.Now()
}

// UpdateGoalProgress updates progress toward a goal
func (s *Session) UpdateGoalProgress(goalID string, achieved int) {
	for i := range s.Goals {
		if s.Goals[i].ID == goalID {
			s.Goals[i].Achieved = achieved
			s.Goals[i].Completed = achieved >= s.Goals[i].Target
			break
		}
	}
	s.UpdatedAt = time.Now()
}

// AddAchievement records an achievement during the session
func (s *Session) AddAchievement(achievementType, title, description string, value float64) {
	achievement := Achievement{
		Type:        achievementType,
		Title:       title,
		Description: description,
		Value:       value,
		Metadata:    make(map[string]interface{}),
		EarnedAt:    time.Now(),
	}
	s.Achievements = append(s.Achievements, achievement)
	s.UpdatedAt = time.Now()
}

// CalculateProductivityScore calculates overall productivity for the session
func (s *Session) CalculateProductivityScore() float64 {
	if s.TasksStarted == 0 {
		return 0.0
	}

	// Completion rate (30%)
	completionRate := float64(s.TasksCompleted) / float64(s.TasksStarted)

	// Efficiency score (25%) - based on task duration estimates
	efficiencyScore := s.calculateEfficiencyScore()

	// Focus score (25%) - based on interruptions
	focusScore := s.calculateFocusScore()

	// Quality score (20%) - based on task quality ratings
	qualityScore := s.calculateQualityScore()

	score := (completionRate * 0.30) +
		(efficiencyScore * 0.25) +
		(focusScore * 0.25) +
		(qualityScore * 0.20)

	s.ProductivityScore = score
	return score
}

// calculateFinalScores calculates all final scores for the session
func (s *Session) calculateFinalScores() {
	s.ProductivityScore = s.CalculateProductivityScore()
	s.FocusScore = s.calculateFocusScore()
	s.QualityScore = s.calculateQualityScore()
	s.VelocityScore = s.calculateVelocityScore()
}

// calculateEfficiencyScore calculates efficiency based on estimated vs actual time
func (s *Session) calculateEfficiencyScore() float64 {
	if len(s.TaskDurations) == 0 {
		return 1.0 // No data, assume perfect efficiency
	}

	var totalEfficiency float64
	validEntries := 0

	for _, entry := range s.TaskDurations {
		if entry.Estimated > 0 {
			efficiency := float64(entry.Estimated) / float64(entry.Actual)
			if efficiency > 2.0 {
				efficiency = 2.0 // Cap at 2x efficiency
			}
			totalEfficiency += efficiency
			validEntries++
		}
	}

	if validEntries == 0 {
		return 1.0
	}

	avgEfficiency := totalEfficiency / float64(validEntries)
	if avgEfficiency > 1.0 {
		avgEfficiency = 1.0 // Cap at 100% efficiency
	}

	return avgEfficiency
}

// calculateFocusScore calculates focus based on interruptions and work periods
func (s *Session) calculateFocusScore() float64 {
	if s.Duration == 0 {
		return 1.0
	}

	// Calculate total interruption time and impact
	var totalInterruptionTime time.Duration
	var totalImpact float64

	for _, interruption := range s.Interruptions {
		totalInterruptionTime += interruption.Duration + interruption.Recovery
		totalImpact += interruption.Impact
	}

	// Calculate focus score based on:
	// 1. Time lost to interruptions (60%)
	// 2. Number and impact of interruptions (40%)

	timeScore := 1.0 - (float64(totalInterruptionTime) / float64(s.Duration))
	if timeScore < 0 {
		timeScore = 0
	}

	interruptionScore := 1.0
	if len(s.Interruptions) > 0 {
		avgImpact := totalImpact / float64(len(s.Interruptions))
		interruptionPenalty := float64(len(s.Interruptions)) * avgImpact * 0.1
		interruptionScore = 1.0 - interruptionPenalty
		if interruptionScore < 0 {
			interruptionScore = 0
		}
	}

	return (timeScore * 0.6) + (interruptionScore * 0.4)
}

// calculateQualityScore calculates quality based on task quality ratings
func (s *Session) calculateQualityScore() float64 {
	if len(s.TaskDurations) == 0 {
		return 1.0
	}

	var totalQuality int
	validEntries := 0

	for _, entry := range s.TaskDurations {
		if entry.Quality > 0 {
			totalQuality += entry.Quality
			validEntries++
		}
	}

	if validEntries == 0 {
		return 1.0
	}

	avgQuality := float64(totalQuality) / float64(validEntries)
	return avgQuality / 5.0 // Normalize to 0-1 (assuming 1-5 scale)
}

// calculateVelocityScore calculates velocity (tasks per hour)
func (s *Session) calculateVelocityScore() float64 {
	if s.Duration == 0 {
		return 0.0
	}

	hours := s.Duration.Hours()
	if hours == 0 {
		return 0.0
	}

	return float64(s.TasksCompleted) / hours
}

// GetActiveTime returns the total active work time (excluding breaks and interruptions)
func (s *Session) GetActiveTime() time.Duration {
	var activeTime time.Duration
	for _, period := range s.WorkPeriods {
		activeTime += period.Duration
	}
	return activeTime
}

// GetGoalCompletion returns the percentage of goals completed
func (s *Session) GetGoalCompletion() float64 {
	if len(s.Goals) == 0 {
		return 1.0
	}

	completed := 0
	for _, goal := range s.Goals {
		if goal.Completed {
			completed++
		}
	}

	return float64(completed) / float64(len(s.Goals))
}

// Validate validates the session entity
func (s *Session) Validate() error {
	validate := validator.New()
	return validate.Struct(s)
}

// ToJSON converts the session to JSON
func (s *Session) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON creates a session from JSON
func (s *Session) FromJSON(data []byte) error {
	return json.Unmarshal(data, s)
}

// Helper function to generate IDs (simplified for this example)
func generateID() string {
	return time.Now().Format("20060102150405") + "000"
}
