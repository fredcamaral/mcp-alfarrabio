package entities

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-playground/validator/v10"
)

// SuggestionType defines the type of suggestion
type SuggestionType string

const (
	SuggestionTypeNext     SuggestionType = "next_task"
	SuggestionTypeRelated  SuggestionType = "related"
	SuggestionTypeOptimize SuggestionType = "optimization"
	SuggestionTypeTemplate SuggestionType = "template"
	SuggestionTypeLearning SuggestionType = "learning"
	SuggestionTypePattern  SuggestionType = "pattern"
	SuggestionTypeWorkflow SuggestionType = "workflow"
	SuggestionTypeBreak    SuggestionType = "break"
	SuggestionTypePriority SuggestionType = "priority"
)

// TaskSuggestion represents a suggested task with context and reasoning
type TaskSuggestion struct {
	ID             string                 `json:"id" validate:"required,uuid"`
	Type           SuggestionType         `json:"type" validate:"required"`
	Content        string                 `json:"content" validate:"required,min=1,max=1000"`
	Description    string                 `json:"description"`
	Reasoning      string                 `json:"reasoning"` // Why this suggestion was made
	Priority       string                 `json:"priority"`
	TaskType       string                 `json:"task_type,omitempty"`
	EstimatedTime  time.Duration          `json:"estimated_time"`
	Confidence     float64                `json:"confidence"` // 0-1 confidence score
	Relevance      float64                `json:"relevance"`  // 0-1 relevance to current context
	Urgency        float64                `json:"urgency"`    // 0-1 how urgent this suggestion is
	Source         SuggestionSource       `json:"source"`
	PatternID      string                 `json:"pattern_id,omitempty"`
	RelatedTaskIDs []string               `json:"related_task_ids"` // IDs of related tasks
	Prerequisites  []string               `json:"prerequisites"`    // Task IDs that should be completed first
	Keywords       []string               `json:"keywords"`
	Tags           []string               `json:"tags"`
	Repository     string                 `json:"repository"`
	Context        *SuggestionContext     `json:"context"`
	Actions        []SuggestedAction      `json:"actions"` // Specific actions to take
	Metadata       map[string]interface{} `json:"metadata"`
	GeneratedAt    time.Time              `json:"generated_at"`
	ExpiresAt      time.Time              `json:"expires_at"`
	UserFeedback   *SuggestionFeedback    `json:"feedback,omitempty"`
	UsageCount     int                    `json:"usage_count"` // How many times this suggestion was used
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// SuggestionSource provides information about where the suggestion came from
type SuggestionSource struct {
	Type       string                 `json:"type"` // "pattern", "ai", "template", "history", "analytics"
	Name       string                 `json:"name"` // Source identifier
	Version    string                 `json:"version,omitempty"`
	Confidence float64                `json:"confidence"`          // Source-specific confidence
	Algorithm  string                 `json:"algorithm,omitempty"` // Algorithm used
	ModelInfo  *ModelInfo             `json:"model_info,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ModelInfo contains information about AI models used
type ModelInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Provider    string                 `json:"provider"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SuggestionContext provides context about when and why the suggestion was made
type SuggestionContext struct {
	CurrentTasks      []string               `json:"current_tasks"`   // Current task IDs
	RecentTasks       []string               `json:"recent_tasks"`    // Recent task IDs
	ActivePatterns    []string               `json:"active_patterns"` // Active pattern IDs
	TimeOfDay         string                 `json:"time_of_day"`
	DayOfWeek         string                 `json:"day_of_week"`
	ProductivityScore float64                `json:"productivity_score"`
	FocusLevel        float64                `json:"focus_level"`
	EnergyLevel       float64                `json:"energy_level"`
	WorkingHours      bool                   `json:"working_hours"`
	SessionDuration   time.Duration          `json:"session_duration"`
	RecentVelocity    float64                `json:"recent_velocity"` // Recent tasks per hour
	StressLevel       float64                `json:"stress_level"`    // 0-1 estimated stress level
	ContextualFactors map[string]interface{} `json:"contextual_factors"`
}

// SuggestedAction represents a specific action that can be taken
type SuggestedAction struct {
	Type        string                 `json:"type"` // create, update, schedule, break, etc.
	Description string                 `json:"description"`
	Command     string                 `json:"command,omitempty"`  // CLI command to execute
	Shortcut    string                 `json:"shortcut,omitempty"` // Keyboard shortcut
	URL         string                 `json:"url,omitempty"`      // URL to open
	Parameters  map[string]interface{} `json:"parameters"`
	Order       int                    `json:"order"` // Order in which to execute actions
}

// SuggestionFeedback captures user feedback on suggestions
type SuggestionFeedback struct {
	Accepted      bool                   `json:"accepted"`
	Helpful       bool                   `json:"helpful"`
	Relevant      bool                   `json:"relevant"`
	Rating        int                    `json:"rating,omitempty"` // 1-5 stars
	Comment       string                 `json:"comment,omitempty"`
	Reason        string                 `json:"reason,omitempty"`       // Reason for rejection
	Alternatives  []string               `json:"alternatives,omitempty"` // Alternative suggestions user would prefer
	TimeTaken     time.Duration          `json:"time_taken,omitempty"`   // Time taken to complete suggested task
	ActualOutcome string                 `json:"actual_outcome,omitempty"`
	Metadata      map[string]interface{} `json:"metadata"`
	ProvidedAt    time.Time              `json:"provided_at"`
}

// WorkContext represents the current work context for generating suggestions
type WorkContext struct {
	CurrentTasks      []*Task                `json:"current_tasks"`
	RecentTasks       []*Task                `json:"recent_tasks"` // Last 10 completed
	Repository        string                 `json:"repository"`
	ProjectType       string                 `json:"project_type"`
	CurrentSession    *Session               `json:"current_session"`
	TimeOfDay         string                 `json:"time_of_day"` // morning, afternoon, evening, night
	DayOfWeek         string                 `json:"day_of_week"`
	WorkingHours      *WorkingHours          `json:"working_hours"`
	RecentPatterns    []*TaskPattern         `json:"recent_patterns"`
	ActivePatterns    []*TaskPattern         `json:"active_patterns"`    // Currently applicable patterns
	Velocity          float64                `json:"velocity"`           // Tasks/hour
	FocusLevel        float64                `json:"focus_level"`        // 0-1 based on interruptions
	EnergyLevel       float64                `json:"energy_level"`       // 0-1 current energy
	ProductivityScore float64                `json:"productivity_score"` // Recent productivity
	StressIndicators  []StressIndicator      `json:"stress_indicators"`
	Environment       *WorkEnvironment       `json:"environment"`
	Goals             []SessionGoal          `json:"goals"`       // Current session goals
	Constraints       []WorkConstraint       `json:"constraints"` // Current constraints
	Preferences       *UserPreferences       `json:"preferences"`
	RecentFeedback    []*SuggestionFeedback  `json:"recent_feedback"` // Recent feedback for learning
	Metadata          map[string]interface{} `json:"metadata"`
	AnalyzedAt        time.Time              `json:"analyzed_at"`
}

// StressIndicator represents a factor that might indicate stress
type StressIndicator struct {
	Type        string    `json:"type"`     // missed_deadline, overdue_tasks, high_velocity, etc.
	Severity    float64   `json:"severity"` // 0-1
	Description string    `json:"description"`
	Impact      string    `json:"impact"` // productivity, quality, wellbeing
	DetectedAt  time.Time `json:"detected_at"`
}

// WorkConstraint represents a constraint on work
type WorkConstraint struct {
	Type        string                 `json:"type"` // time, resources, dependencies, etc.
	Description string                 `json:"description"`
	Impact      float64                `json:"impact"`             // 0-1 how much this constrains work
	Duration    time.Duration          `json:"duration,omitempty"` // How long constraint lasts
	Metadata    map[string]interface{} `json:"metadata"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// UserPreferences represents user preferences for suggestions
type UserPreferences struct {
	PreferredSuggestionTypes []SuggestionType       `json:"preferred_suggestion_types"`
	MaxSuggestions           int                    `json:"max_suggestions"`
	MinConfidence            float64                `json:"min_confidence"`
	PreferredTaskLength      time.Duration          `json:"preferred_task_length"`
	PreferredTaskTypes       []string               `json:"preferred_task_types"`
	AvoidancePatterns        []string               `json:"avoidance_patterns"` // Patterns to avoid suggesting
	NotificationFrequency    string                 `json:"notification_frequency"`
	LearningRate             float64                `json:"learning_rate"` // How quickly to adapt to feedback
	ExperimentalFeatures     bool                   `json:"experimental_features"`
	Metadata                 map[string]interface{} `json:"metadata"`
}

// SuggestionBatch represents a batch of related suggestions
type SuggestionBatch struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"` // workflow, priority, break, etc.
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Suggestions   []*TaskSuggestion      `json:"suggestions"`
	Context       *WorkContext           `json:"context"`
	Priority      float64                `json:"priority"`       // 0-1 priority of this batch
	Confidence    float64                `json:"confidence"`     // 0-1 confidence in this batch
	EstimatedTime time.Duration          `json:"estimated_time"` // Total estimated time for batch
	Dependencies  []string               `json:"dependencies"`   // Other batch IDs this depends on
	Metadata      map[string]interface{} `json:"metadata"`
	GeneratedAt   time.Time              `json:"generated_at"`
	ExpiresAt     time.Time              `json:"expires_at"`
}

// SuggestionStats tracks statistics about suggestions
type SuggestionStats struct {
	TotalGenerated    int                    `json:"total_generated"`
	TotalAccepted     int                    `json:"total_accepted"`
	TotalRejected     int                    `json:"total_rejected"`
	AcceptanceRate    float64                `json:"acceptance_rate"`
	ByType            map[SuggestionType]int `json:"by_type"`
	BySource          map[string]int         `json:"by_source"`
	AverageRating     float64                `json:"average_rating"`
	AverageConfidence float64                `json:"average_confidence"`
	TopKeywords       []string               `json:"top_keywords"`
	Period            TimeRange              `json:"period"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// TimeRange represents a time period
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NewTaskSuggestion creates a new task suggestion
func NewTaskSuggestion(
	suggestionType SuggestionType,
	content string,
	source SuggestionSource,
	repository string,
) *TaskSuggestion {
	now := time.Now()
	return &TaskSuggestion{
		ID:             generateSuggestionID(),
		Type:           suggestionType,
		Content:        content,
		Source:         source,
		Repository:     repository,
		Keywords:       make([]string, 0),
		Tags:           make([]string, 0),
		RelatedTaskIDs: make([]string, 0),
		Prerequisites:  make([]string, 0),
		Actions:        make([]SuggestedAction, 0),
		Metadata:       make(map[string]interface{}),
		GeneratedAt:    now,
		ExpiresAt:      now.Add(24 * time.Hour), // Default 24 hour expiry
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// IsExpired checks if the suggestion has expired
func (ts *TaskSuggestion) IsExpired() bool {
	return time.Now().After(ts.ExpiresAt)
}

// IsRelevant checks if the suggestion is relevant based on confidence and relevance scores
func (ts *TaskSuggestion) IsRelevant(minConfidence, minRelevance float64) bool {
	return ts.Confidence >= minConfidence && ts.Relevance >= minRelevance
}

// UpdateFeedback updates the suggestion with user feedback
func (ts *TaskSuggestion) UpdateFeedback(feedback *SuggestionFeedback) {
	ts.UserFeedback = feedback
	ts.UpdatedAt = time.Now()

	// Update usage count if accepted
	if feedback.Accepted {
		ts.UsageCount++
	}
}

// CalculateScore calculates a composite score for ranking suggestions
func (ts *TaskSuggestion) CalculateScore() float64 {
	// Weighted combination of confidence, relevance, and urgency
	return (ts.Confidence * 0.4) + (ts.Relevance * 0.4) + (ts.Urgency * 0.2)
}

// AddAction adds a suggested action to the suggestion
func (ts *TaskSuggestion) AddAction(actionType, description string, order int) {
	action := SuggestedAction{
		Type:        actionType,
		Description: description,
		Order:       order,
		Parameters:  make(map[string]interface{}),
	}
	ts.Actions = append(ts.Actions, action)
	ts.UpdatedAt = time.Now()
}

// SetEstimatedTime sets the estimated time and updates reasoning
func (ts *TaskSuggestion) SetEstimatedTime(duration time.Duration, basis string) {
	ts.EstimatedTime = duration
	if ts.Reasoning == "" {
		ts.Reasoning = fmt.Sprintf("Estimated time: %v (based on %s)", duration, basis)
	} else {
		ts.Reasoning += fmt.Sprintf(". Estimated time: %v (based on %s)", duration, basis)
	}
	ts.UpdatedAt = time.Now()
}

// AddKeywords adds keywords to the suggestion
func (ts *TaskSuggestion) AddKeywords(keywords ...string) {
	for _, keyword := range keywords {
		// Avoid duplicates
		exists := false
		for _, existing := range ts.Keywords {
			if existing == keyword {
				exists = true
				break
			}
		}
		if !exists {
			ts.Keywords = append(ts.Keywords, keyword)
		}
	}
	ts.UpdatedAt = time.Now()
}

// AddPrerequisites adds prerequisite task IDs
func (ts *TaskSuggestion) AddPrerequisites(taskIDs ...string) {
	ts.Prerequisites = append(ts.Prerequisites, taskIDs...)
	ts.UpdatedAt = time.Now()
}

// Validate validates the suggestion entity
func (ts *TaskSuggestion) Validate() error {
	validate := validator.New()
	return validate.Struct(ts)
}

// ToJSON converts the suggestion to JSON
func (ts *TaskSuggestion) ToJSON() ([]byte, error) {
	return json.Marshal(ts)
}

// FromJSON creates a suggestion from JSON
func (ts *TaskSuggestion) FromJSON(data []byte) error {
	return json.Unmarshal(data, ts)
}

// Hash generates a hash for the work context for caching
func (wc *WorkContext) Hash() string {
	// Simple hash based on key context factors
	factors := fmt.Sprintf("%s_%s_%s_%d_%d_%.2f_%.2f",
		wc.Repository,
		wc.TimeOfDay,
		wc.DayOfWeek,
		len(wc.CurrentTasks),
		len(wc.RecentTasks),
		wc.Velocity,
		wc.FocusLevel,
	)

	// In a real implementation, you'd use a proper hash function
	return fmt.Sprintf("context_%x", []byte(factors))
}

// GetActiveTaskTypes returns the types of currently active tasks
func (wc *WorkContext) GetActiveTaskTypes() []string {
	typeSet := make(map[string]bool)
	for _, task := range wc.CurrentTasks {
		// Use tags instead of Type field
		for _, tag := range task.Tags {
			typeSet[tag] = true
		}
	}

	types := make([]string, 0, len(typeSet))
	for taskType := range typeSet {
		types = append(types, taskType)
	}
	return types
}

// GetPrimaryPatterns returns the most relevant patterns for current context
func (wc *WorkContext) GetPrimaryPatterns() []*TaskPattern {
	if len(wc.ActivePatterns) <= 3 {
		return wc.ActivePatterns
	}

	// Sort by confidence and return top 3
	patterns := make([]*TaskPattern, len(wc.ActivePatterns))
	copy(patterns, wc.ActivePatterns)

	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Confidence > patterns[j].Confidence
	})

	return patterns[:3]
}

// IsHighStress checks if the current context indicates high stress
func (wc *WorkContext) IsHighStress() bool {
	highStressCount := 0
	for _, indicator := range wc.StressIndicators {
		if indicator.Severity > 0.7 {
			highStressCount++
		}
	}
	return highStressCount >= 2
}

// NewSuggestionBatch creates a new suggestion batch
func NewSuggestionBatch(batchType, title string, suggestions []*TaskSuggestion) *SuggestionBatch {
	now := time.Now()

	// Calculate total estimated time
	var totalTime time.Duration
	var totalConfidence float64
	for _, suggestion := range suggestions {
		totalTime += suggestion.EstimatedTime
		totalConfidence += suggestion.Confidence
	}

	avgConfidence := 0.0
	if len(suggestions) > 0 {
		avgConfidence = totalConfidence / float64(len(suggestions))
	}

	return &SuggestionBatch{
		ID:            generateSuggestionID(),
		Type:          batchType,
		Title:         title,
		Suggestions:   suggestions,
		Confidence:    avgConfidence,
		EstimatedTime: totalTime,
		Dependencies:  make([]string, 0),
		Metadata:      make(map[string]interface{}),
		GeneratedAt:   now,
		ExpiresAt:     now.Add(24 * time.Hour),
	}
}

// Helper functions

func generateSuggestionID() string {
	return fmt.Sprintf("sugg_%d", time.Now().UnixNano())
}

// NewWorkContext creates a new work context
func NewWorkContext(repository string) *WorkContext {
	return &WorkContext{
		Repository:       repository,
		CurrentTasks:     make([]*Task, 0),
		RecentTasks:      make([]*Task, 0),
		RecentPatterns:   make([]*TaskPattern, 0),
		ActivePatterns:   make([]*TaskPattern, 0),
		StressIndicators: make([]StressIndicator, 0),
		Goals:            make([]SessionGoal, 0),
		Constraints:      make([]WorkConstraint, 0),
		RecentFeedback:   make([]*SuggestionFeedback, 0),
		Metadata:         make(map[string]interface{}),
		AnalyzedAt:       time.Now(),
	}
}

// NewUserPreferences creates default user preferences
func NewUserPreferences() *UserPreferences {
	return &UserPreferences{
		PreferredSuggestionTypes: []SuggestionType{
			SuggestionTypeNext,
			SuggestionTypeRelated,
			SuggestionTypePattern,
		},
		MaxSuggestions:        10,
		MinConfidence:         0.6,
		PreferredTaskLength:   time.Hour,
		PreferredTaskTypes:    make([]string, 0),
		AvoidancePatterns:     make([]string, 0),
		NotificationFrequency: "moderate",
		LearningRate:          0.1,
		ExperimentalFeatures:  false,
		Metadata:              make(map[string]interface{}),
	}
}
