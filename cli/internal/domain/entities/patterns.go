package entities

import (
	"encoding/json"
	"time"

	"github.com/go-playground/validator/v10"
)

// PatternType defines the type of pattern detected
type PatternType string

const (
	PatternTypeSequence PatternType = "sequence"
	PatternTypeWorkflow PatternType = "workflow"
	PatternTypeTemporal PatternType = "temporal"
	PatternTypeProject  PatternType = "project"
)

// TaskPattern represents a detected pattern in task completion
type TaskPattern struct {
	ID          string                 `json:"id" validate:"required,uuid"`
	Type        PatternType            `json:"type" validate:"required"`
	Name        string                 `json:"name" validate:"required,min=1,max=200"`
	Description string                 `json:"description"`
	Sequence    []PatternStep          `json:"sequence"`
	Frequency   float64                `json:"frequency"`    // How often this pattern occurs (0-1)
	Confidence  float64                `json:"confidence"`   // Confidence score 0-1
	SuccessRate float64                `json:"success_rate"` // Completion success rate 0-1
	Repository  string                 `json:"repository"`
	ProjectType string                 `json:"project_type,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
	Occurrences int                    `json:"occurrences"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PatternStep represents a step in a task pattern
type PatternStep struct {
	Order       int                    `json:"order"`
	TaskType    string                 `json:"task_type"` // Type of task (feature, bug, refactor)
	Keywords    []string               `json:"keywords"`  // Common keywords found
	Duration    *DurationStats         `json:"duration"`  // Time statistics for this step
	Priority    string                 `json:"priority,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Probability float64                `json:"probability"` // Probability of this step appearing
}

// DurationStats holds statistical information about task durations
type DurationStats struct {
	Average   time.Duration `json:"average"`
	Min       time.Duration `json:"min"`
	Max       time.Duration `json:"max"`
	StdDev    time.Duration `json:"std_dev"`
	Median    time.Duration `json:"median"`
	Samples   int           `json:"samples"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// PatternOutcome represents the result of following a pattern
type PatternOutcome struct {
	PatternID      string        `json:"pattern_id"`
	Completed      bool          `json:"completed"`
	CompletionRate float64       `json:"completion_rate"` // 0-1 for partial completion
	Duration       time.Duration `json:"duration"`
	Deviation      float64       `json:"deviation"`    // How much it deviated from predicted
	Satisfaction   float64       `json:"satisfaction"` // User satisfaction 0-1
	Notes          string        `json:"notes,omitempty"`
	RecordedAt     time.Time     `json:"recorded_at"`
}

// WorkflowPattern represents a high-level workflow pattern
type WorkflowPattern struct {
	ID            string                 `json:"id" validate:"required,uuid"`
	Name          string                 `json:"name" validate:"required"`
	Description   string                 `json:"description"`
	Phases        []WorkflowPhase        `json:"phases"`
	TotalTasks    int                    `json:"total_tasks"`
	AvgDuration   time.Duration          `json:"avg_duration"`
	SuccessRate   float64                `json:"success_rate"`
	Repository    string                 `json:"repository"`
	ProjectType   string                 `json:"project_type"`
	Prerequisites []string               `json:"prerequisites"` // Required conditions
	Outcomes      []string               `json:"outcomes"`      // Expected results
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// WorkflowPhase represents a phase in a workflow
type WorkflowPhase struct {
	Name         string        `json:"name"`
	Order        int           `json:"order"`
	Description  string        `json:"description"`
	TaskPatterns []string      `json:"task_patterns"` // Pattern IDs used in this phase
	AvgDuration  time.Duration `json:"avg_duration"`
	Required     bool          `json:"required"`
	Parallel     bool          `json:"parallel"` // Can be done in parallel with other phases
}

// SequencePattern represents a sequence of tasks that commonly occur together
type SequencePattern struct {
	ID          string                 `json:"id" validate:"required,uuid"`
	Tasks       []SequenceTask         `json:"tasks"`
	Support     float64                `json:"support"`    // How frequently this sequence appears
	Confidence  float64                `json:"confidence"` // Likelihood of completion
	Lift        float64                `json:"lift"`       // How much more likely than random
	Repository  string                 `json:"repository"`
	MinGap      time.Duration          `json:"min_gap"` // Minimum time between tasks
	MaxGap      time.Duration          `json:"max_gap"` // Maximum time between tasks
	Occurrences int                    `json:"occurrences"`
	Metadata    map[string]interface{} `json:"metadata"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
}

// SequenceTask represents a task in a sequence pattern
type SequenceTask struct {
	Order        int           `json:"order"`
	ContentMatch string        `json:"content_match"` // Pattern to match task content
	Priority     string        `json:"priority,omitempty"`
	Tags         []string      `json:"tags,omitempty"`
	AvgDuration  time.Duration `json:"avg_duration"`
	Required     bool          `json:"required"` // Whether this task is always present
}

// TemporalPattern represents time-based patterns in task completion
type TemporalPattern struct {
	ID           string                 `json:"id" validate:"required,uuid"`
	Type         string                 `json:"type"` // daily, weekly, monthly
	Name         string                 `json:"name"`
	TimeSlots    []TimeSlot             `json:"time_slots"`
	Repository   string                 `json:"repository"`
	TaskTypes    []string               `json:"task_types"`   // Types of tasks in this pattern
	Productivity float64                `json:"productivity"` // Relative productivity score
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TimeSlot represents a time period with associated task data
type TimeSlot struct {
	StartTime      string         `json:"start_time"` // e.g., "09:00" or "Monday 09:00"
	EndTime        string         `json:"end_time"`
	TaskCount      int            `json:"task_count"`
	CompletionRate float64        `json:"completion_rate"`
	AvgDuration    time.Duration  `json:"avg_duration"`
	TaskTypes      map[string]int `json:"task_types"` // Count by task type
}

// Validate validates the pattern entity
func (p *TaskPattern) Validate() error {
	validate := validator.New()
	return validate.Struct(p)
}

// GetID returns the pattern ID (implements Entity interface)
func (p *TaskPattern) GetID() string {
	return p.ID
}

// GetRepository returns the pattern repository (implements Entity interface)
func (p *TaskPattern) GetRepository() string {
	return p.Repository
}

// AddOccurrence updates pattern statistics with a new occurrence
func (p *TaskPattern) AddOccurrence(outcome *PatternOutcome) {
	p.Occurrences++
	p.LastSeen = time.Now()

	// Update success rate
	if outcome.Completed {
		completedCount := float64(p.Occurrences) * p.SuccessRate
		completedCount += outcome.CompletionRate
		p.SuccessRate = completedCount / float64(p.Occurrences)
	}

	p.UpdatedAt = time.Now()
}

// CalculateConfidence calculates confidence based on occurrences and success rate
func (p *TaskPattern) CalculateConfidence() float64 {
	// Base confidence on number of occurrences and success rate
	occurrenceScore := float64(p.Occurrences) / (float64(p.Occurrences) + 10.0) // Diminishing returns
	return (occurrenceScore * 0.6) + (p.SuccessRate * 0.4)
}

// IsExpired checks if the pattern should be considered outdated
func (p *TaskPattern) IsExpired(maxAge time.Duration) bool {
	return time.Since(p.LastSeen) > maxAge
}

// ToJSON converts the pattern to JSON
func (p *TaskPattern) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// FromJSON creates a pattern from JSON
func (p *TaskPattern) FromJSON(data []byte) error {
	return json.Unmarshal(data, p)
}

// UpdateDurationStats updates duration statistics with new data
func (d *DurationStats) UpdateDurationStats(duration time.Duration) {
	if d.Samples == 0 {
		// First sample
		d.Average = duration
		d.Min = duration
		d.Max = duration
		d.StdDev = 0
		d.Median = duration
		d.Samples = 1
	} else {
		// Update statistics
		oldAvg := d.Average
		d.Samples++

		// Update average
		d.Average = time.Duration(
			(int64(oldAvg)*int64(d.Samples-1) + int64(duration)) / int64(d.Samples),
		)

		// Update min/max
		if duration < d.Min {
			d.Min = duration
		}
		if duration > d.Max {
			d.Max = duration
		}

		// Estimate standard deviation (simplified calculation)
		diff := duration - d.Average
		d.StdDev = time.Duration(
			(int64(d.StdDev)*int64(d.Samples-1) + int64(diff)*int64(diff)) / int64(d.Samples),
		)
	}

	d.UpdatedAt = time.Now()
}

// IsSignificant checks if the pattern has enough data to be considered significant
func (p *TaskPattern) IsSignificant(minOccurrences int, minConfidence float64) bool {
	return p.Occurrences >= minOccurrences && p.CalculateConfidence() >= minConfidence
}

// GetEstimatedDuration returns the estimated duration for this pattern
func (p *TaskPattern) GetEstimatedDuration() time.Duration {
	var totalDuration time.Duration
	for _, step := range p.Sequence {
		if step.Duration != nil {
			totalDuration += step.Duration.Average
		}
	}
	return totalDuration
}

// GetKeywords extracts all keywords from pattern steps
func (p *TaskPattern) GetKeywords() []string {
	keywordSet := make(map[string]bool)
	var keywords []string

	for _, step := range p.Sequence {
		for _, keyword := range step.Keywords {
			if !keywordSet[keyword] {
				keywordSet[keyword] = true
				keywords = append(keywords, keyword)
			}
		}
	}

	return keywords
}
