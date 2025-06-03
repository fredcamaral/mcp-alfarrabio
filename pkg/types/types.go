// Package types provides core data structures and type definitions
// for the MCP Memory Server, including conversation chunks and metadata.
package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Priority levels
const (
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// Time periods
const (
	TimeframWeek     = "week"
	TimeframeMonth   = "month"
	TimeframeQuarter = "quarter"
	TimeframeAll     = "all"
)

// Source types
const (
	SourceConversation = "conversation"
	SourceFile         = "file"
	SourceArchive      = "archive"
)

// ChunkType represents the type of conversation chunk
type ChunkType string

const (
	// ChunkTypeProblem represents a problem or issue being discussed
	ChunkTypeProblem ChunkType = "problem"
	// ChunkTypeSolution represents a solution to a problem
	ChunkTypeSolution ChunkType = "solution"
	// ChunkTypeCodeChange represents code modifications or additions
	ChunkTypeCodeChange ChunkType = "code_change"
	// ChunkTypeDiscussion represents general discussion or conversation
	ChunkTypeDiscussion ChunkType = "discussion"
	// ChunkTypeArchitectureDecision represents architectural decisions or design choices
	ChunkTypeArchitectureDecision ChunkType = "architecture_decision"
	// ChunkTypeSessionSummary represents a summary of a work session
	ChunkTypeSessionSummary ChunkType = "session_summary"
	// ChunkTypeAnalysis represents analysis or investigation results
	ChunkTypeAnalysis ChunkType = "analysis"
	// ChunkTypeVerification represents verification or testing activities
	ChunkTypeVerification ChunkType = "verification"
	// ChunkTypeQuestion represents questions or inquiries
	ChunkTypeQuestion ChunkType = "question"
	// ChunkTypeTask represents a task or work item
	ChunkTypeTask ChunkType = "task"
	// ChunkTypeTaskUpdate represents updates to a task
	ChunkTypeTaskUpdate ChunkType = "task_update"
	// ChunkTypeTaskProgress represents progress tracking for a task
	ChunkTypeTaskProgress ChunkType = "task_progress"
)

// Valid returns true if the chunk type is valid
func (ct ChunkType) Valid() bool {
	switch ct {
	case ChunkTypeProblem, ChunkTypeSolution, ChunkTypeCodeChange, ChunkTypeDiscussion, ChunkTypeArchitectureDecision, ChunkTypeSessionSummary, ChunkTypeAnalysis, ChunkTypeVerification, ChunkTypeQuestion, ChunkTypeTask, ChunkTypeTaskUpdate, ChunkTypeTaskProgress:
		return true
	}
	return false
}

// Outcome represents the outcome of a conversation chunk
type Outcome string

const (
	// OutcomeSuccess indicates a successful completion or resolution
	OutcomeSuccess Outcome = "success"
	// OutcomeInProgress indicates work is still ongoing
	OutcomeInProgress Outcome = "in_progress"
	// OutcomeFailed indicates a failed attempt or unsuccessful outcome
	OutcomeFailed Outcome = "failed"
	// OutcomeAbandoned indicates the work was abandoned or cancelled
	OutcomeAbandoned Outcome = "abandoned"
)

// TaskStatus represents the status of a task-oriented chunk
type TaskStatus string

const (
	// TaskStatusTodo indicates a task that needs to be started
	TaskStatusTodo TaskStatus = "todo"
	// TaskStatusInProgress indicates a task currently being worked on
	TaskStatusInProgress TaskStatus = "in_progress"
	// TaskStatusCompleted indicates a finished task
	TaskStatusCompleted TaskStatus = "completed"
	// TaskStatusBlocked indicates a task that cannot proceed due to dependencies
	TaskStatusBlocked TaskStatus = "blocked"
	// TaskStatusCancelled indicates a task that was cancelled
	TaskStatusCancelled TaskStatus = "cancelled"
	// TaskStatusOnHold indicates a task that is temporarily paused
	TaskStatusOnHold TaskStatus = "on_hold"
)

// Valid returns true if the outcome is valid
func (o Outcome) Valid() bool {
	switch o {
	case OutcomeSuccess, OutcomeInProgress, OutcomeFailed, OutcomeAbandoned:
		return true
	}
	return false
}

// Valid returns true if the task status is valid
func (ts TaskStatus) Valid() bool {
	switch ts {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusCompleted, TaskStatusBlocked, TaskStatusCancelled, TaskStatusOnHold:
		return true
	}
	return false
}

// Difficulty represents the difficulty level of a task
type Difficulty string

const (
	// DifficultySimple indicates a simple or straightforward task
	DifficultySimple Difficulty = "simple"
	// DifficultyModerate indicates a task of moderate complexity
	DifficultyModerate Difficulty = "moderate"
	// DifficultyComplex indicates a complex or challenging task
	DifficultyComplex Difficulty = "complex"
)

// Valid returns true if the difficulty is valid
func (d Difficulty) Valid() bool {
	switch d {
	case DifficultySimple, DifficultyModerate, DifficultyComplex:
		return true
	}
	return false
}

// ConversationFlow represents the flow of conversation
type ConversationFlow string

const (
	// FlowProblem indicates conversation is focused on identifying problems
	FlowProblem ConversationFlow = "problem"
	// FlowInvestigation indicates conversation is in investigation or exploration phase
	FlowInvestigation ConversationFlow = "investigation"
	// FlowSolution indicates conversation is focused on finding solutions
	FlowSolution ConversationFlow = "solution"
	// FlowVerification indicates conversation is focused on verification or testing
	FlowVerification ConversationFlow = "verification"
)

// Valid returns true if the conversation flow is valid
func (cf ConversationFlow) Valid() bool {
	switch cf {
	case FlowProblem, FlowInvestigation, FlowSolution, FlowVerification:
		return true
	}
	return false
}

// Recency represents time-based filtering options
type Recency string

const (
	// RecencyRecent filters for recent items
	RecencyRecent Recency = "recent"
	// RecencyAllTime includes all items regardless of age
	RecencyAllTime Recency = "all_time"
	// RecencyLastMonth filters for items from the last month
	RecencyLastMonth Recency = "last_month"
)

// Valid returns true if the recency is valid
func (r Recency) Valid() bool {
	switch r {
	case RecencyRecent, RecencyAllTime, RecencyLastMonth:
		return true
	}
	return false
}

// ChunkMetadata contains metadata about a conversation chunk
type ChunkMetadata struct {
	Repository       string                 `json:"repository,omitempty"`
	Branch           string                 `json:"branch,omitempty"`
	FilesModified    []string               `json:"files_modified"`
	ToolsUsed        []string               `json:"tools_used"`
	Outcome          Outcome                `json:"outcome"`
	Tags             []string               `json:"tags"`
	Difficulty       Difficulty             `json:"difficulty"`
	TimeSpent        *int                   `json:"time_spent,omitempty"` // minutes
	ExtendedMetadata map[string]interface{} `json:"extended_metadata,omitempty"`

	// Enhanced confidence and quality metrics
	Confidence *ConfidenceMetrics `json:"confidence,omitempty"`
	Quality    *QualityMetrics    `json:"quality,omitempty"`

	// Task-specific metadata (only populated for task-oriented chunks)
	TaskStatus       *TaskStatus `json:"task_status,omitempty"`
	TaskPriority     *string     `json:"task_priority,omitempty"` // high, medium, low
	TaskDueDate      *time.Time  `json:"task_due_date,omitempty"`
	TaskAssignee     *string     `json:"task_assignee,omitempty"`
	TaskDependencies []string    `json:"task_dependencies,omitempty"` // IDs of chunks this task depends on
	TaskBlocks       []string    `json:"task_blocks,omitempty"`       // IDs of chunks this task blocks
	TaskEstimate     *int        `json:"task_estimate,omitempty"`     // estimated time in minutes
	TaskProgress     *int        `json:"task_progress,omitempty"`     // percentage 0-100
}

// Validate checks if the metadata is valid
func (cm *ChunkMetadata) Validate() error {
	if !cm.Outcome.Valid() {
		return fmt.Errorf("invalid outcome: %s", cm.Outcome)
	}
	if !cm.Difficulty.Valid() {
		return fmt.Errorf("invalid difficulty: %s", cm.Difficulty)
	}
	if cm.TimeSpent != nil && *cm.TimeSpent < 0 {
		return errors.New("time spent cannot be negative")
	}

	// Task-specific validation
	if cm.TaskStatus != nil && !cm.TaskStatus.Valid() {
		return fmt.Errorf("invalid task status: %s", *cm.TaskStatus)
	}
	if cm.TaskProgress != nil && (*cm.TaskProgress < 0 || *cm.TaskProgress > 100) {
		return errors.New("task progress must be between 0 and 100")
	}
	if cm.TaskEstimate != nil && *cm.TaskEstimate < 0 {
		return errors.New("task estimate cannot be negative")
	}
	if cm.TaskPriority != nil {
		switch *cm.TaskPriority {
		case PriorityHigh, PriorityMedium, PriorityLow:
			// Valid priority
		default:
			return fmt.Errorf("invalid task priority: %s", *cm.TaskPriority)
		}
	}

	return nil
}

// ConversationChunk represents a chunk of conversation with embeddings
type ConversationChunk struct {
	ID            string        `json:"id"`
	SessionID     string        `json:"session_id"`
	Timestamp     time.Time     `json:"timestamp"`
	Type          ChunkType     `json:"type"`
	Content       string        `json:"content"`
	Summary       string        `json:"summary"` // AI-generated summary
	Metadata      ChunkMetadata `json:"metadata"`
	Embeddings    []float64     `json:"embeddings"`
	RelatedChunks []string      `json:"related_chunks,omitempty"`
}

// NewConversationChunk creates a new conversation chunk with defaults
func NewConversationChunk(sessionID, content string, chunkType ChunkType, metadata ChunkMetadata) (*ConversationChunk, error) {
	if sessionID == "" {
		return nil, errors.New("session ID cannot be empty")
	}
	if content == "" {
		return nil, errors.New("content cannot be empty")
	}
	if !chunkType.Valid() {
		return nil, fmt.Errorf("invalid chunk type: %s", chunkType)
	}
	if err := metadata.Validate(); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	return &ConversationChunk{
		ID:            uuid.New().String(),
		SessionID:     sessionID,
		Timestamp:     time.Now().UTC(),
		Type:          chunkType,
		Content:       content,
		Summary:       "", // Will be generated later
		Metadata:      metadata,
		Embeddings:    []float64{},
		RelatedChunks: []string{},
	}, nil
}

// Validate checks if the conversation chunk is valid
func (cc *ConversationChunk) Validate() error {
	if cc.ID == "" {
		return errors.New("ID cannot be empty")
	}
	if cc.SessionID == "" {
		return errors.New("session ID cannot be empty")
	}
	if cc.Content == "" {
		return errors.New("content cannot be empty")
	}
	if !cc.Type.Valid() {
		return fmt.Errorf("invalid chunk type: %s", cc.Type)
	}
	if cc.Timestamp.IsZero() {
		return errors.New("timestamp cannot be zero")
	}
	return cc.Metadata.Validate()
}

// ProjectContext represents context about a project
type ProjectContext struct {
	Repository             string    `json:"repository"`
	LastAccessed           time.Time `json:"last_accessed"`
	TotalSessions          int       `json:"total_sessions"`
	CommonPatterns         []string  `json:"common_patterns"`
	ArchitecturalDecisions []string  `json:"architectural_decisions"`
	TechStack              []string  `json:"tech_stack"`
	TeamPreferences        []string  `json:"team_preferences"`
}

// NewProjectContext creates a new project context
func NewProjectContext(repository string) *ProjectContext {
	return &ProjectContext{
		Repository:             repository,
		LastAccessed:           time.Now().UTC(),
		TotalSessions:          0,
		CommonPatterns:         []string{},
		ArchitecturalDecisions: []string{},
		TechStack:              []string{},
		TeamPreferences:        []string{},
	}
}

// Validate checks if the project context is valid
func (pc *ProjectContext) Validate() error {
	if pc.Repository == "" {
		return errors.New("repository cannot be empty")
	}
	if pc.TotalSessions < 0 {
		return errors.New("total sessions cannot be negative")
	}
	return nil
}

// MemoryQuery represents a query for searching memory
type MemoryQuery struct {
	Query             string      `json:"query"`
	Repository        *string     `json:"repository,omitempty"`
	FileContext       []string    `json:"file_context,omitempty"`
	Recency           Recency     `json:"recency"`
	Types             []ChunkType `json:"types,omitempty"`
	MinRelevanceScore float64     `json:"min_relevance_score"`
	Limit             int         `json:"limit,omitempty"`
}

// NewMemoryQuery creates a new memory query with defaults
// Note: MinRelevanceScore will be overridden by config in progressive search
func NewMemoryQuery(query string) *MemoryQuery {
	return &MemoryQuery{
		Query:             query,
		Recency:           RecencyRecent,
		MinRelevanceScore: 0.5, // Default fallback, config overrides this
		Limit:             10,
	}
}

// Validate checks if the memory query is valid
func (mq *MemoryQuery) Validate() error {
	if mq.Query == "" {
		return errors.New("query cannot be empty")
	}
	if !mq.Recency.Valid() {
		return fmt.Errorf("invalid recency: %s", mq.Recency)
	}
	if mq.MinRelevanceScore < 0 || mq.MinRelevanceScore > 1 {
		return errors.New("min relevance score must be between 0 and 1")
	}
	if mq.Limit < 0 {
		return errors.New("limit cannot be negative")
	}
	for _, chunkType := range mq.Types {
		if !chunkType.Valid() {
			return fmt.Errorf("invalid chunk type: %s", chunkType)
		}
	}
	return nil
}

// TodoItem represents a todo item from Claude's todo system
type TodoItem struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Content string `json:"content"`
}

// ConfidenceFactors represents factors that influenced confidence calculation
type ConfidenceFactors struct {
	UserCertainty       *float64 `json:"user_certainty,omitempty"`       // 0.0-1.0
	ConsistencyScore    *float64 `json:"consistency_score,omitempty"`    // 0.0-1.0
	CorroborationCount  *int     `json:"corroboration_count,omitempty"`  // Number of supporting memories
	SemanticSimilarity  *float64 `json:"semantic_similarity,omitempty"`  // 0.0-1.0
	TemporalProximity   *float64 `json:"temporal_proximity,omitempty"`   // 0.0-1.0
	ContextualRelevance *float64 `json:"contextual_relevance,omitempty"` // 0.0-1.0
}

// ConfidenceMetrics represents confidence information for a memory chunk
type ConfidenceMetrics struct {
	Score           float64           `json:"score"`  // 0.0 to 1.0
	Source          string            `json:"source"` // explicit, inferred, derived, auto
	Factors         ConfidenceFactors `json:"factors,omitempty"`
	LastUpdated     *time.Time        `json:"last_updated,omitempty"`
	ValidationCount int               `json:"validation_count"`
}

// QualityMetrics represents quality metrics for a memory chunk
type QualityMetrics struct {
	Completeness   float64    `json:"completeness"`    // How complete is this memory (0.0-1.0)
	Clarity        float64    `json:"clarity"`         // How clear/unambiguous (0.0-1.0)
	RelevanceDecay float64    `json:"relevance_decay"` // How much relevance has decayed (0.0-1.0)
	FreshnessScore float64    `json:"freshness_score"` // How fresh/current (0.0-1.0)
	UsageScore     float64    `json:"usage_score"`     // Based on access patterns (0.0-1.0)
	OverallQuality float64    `json:"overall_quality"` // Weighted combination (0.0-1.0)
	LastCalculated *time.Time `json:"last_calculated,omitempty"`
}

// CalculateOverallQuality calculates the overall quality score
func (qm *QualityMetrics) CalculateOverallQuality() {
	// Weighted average of quality factors
	weights := map[string]float64{
		"completeness":    0.25,
		"clarity":         0.25,
		"relevance_decay": 0.20, // Inverted: lower decay = higher quality
		"freshness_score": 0.15,
		"usage_score":     0.15,
	}

	qm.OverallQuality = (weights["completeness"]*qm.Completeness +
		weights["clarity"]*qm.Clarity +
		weights["relevance_decay"]*(1.0-qm.RelevanceDecay) + // Invert decay
		weights["freshness_score"]*qm.FreshnessScore +
		weights["usage_score"]*qm.UsageScore)

	now := time.Now().UTC()
	qm.LastCalculated = &now
}

// ChunkingContext represents context for chunking decisions
type ChunkingContext struct {
	CurrentTodos      []TodoItem       `json:"current_todos"`
	FileModifications []string         `json:"file_modifications"`
	ToolsUsed         []string         `json:"tools_used"`
	TimeElapsed       int              `json:"time_elapsed"` // minutes
	ConversationFlow  ConversationFlow `json:"conversation_flow"`
}

// Validate checks if the chunking context is valid
func (cc *ChunkingContext) Validate() error {
	if cc.TimeElapsed < 0 {
		return errors.New("time elapsed cannot be negative")
	}
	if !cc.ConversationFlow.Valid() {
		return fmt.Errorf("invalid conversation flow: %s", cc.ConversationFlow)
	}
	return nil
}

// HasCompletedTodos returns true if there are completed todos
func (cc *ChunkingContext) HasCompletedTodos() bool {
	for _, todo := range cc.CurrentTodos {
		if todo.Status == "completed" {
			return true
		}
	}
	return false
}

// SearchResult represents a search result with relevance score
type SearchResult struct {
	Chunk ConversationChunk `json:"chunk"`
	Score float64           `json:"score"`
}

// SearchResults represents a collection of search results
type SearchResults struct {
	Results   []SearchResult `json:"results"`
	Total     int            `json:"total"`
	QueryTime time.Duration  `json:"query_time"`
}

// JSON marshaling helpers for custom types

// MarshalJSON implements json.Marshaler for ChunkType
func (ct ChunkType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(ct))
}

// UnmarshalJSON implements json.Unmarshaler for ChunkType
func (ct *ChunkType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*ct = ChunkType(s)
	return nil
}

// MarshalJSON implements json.Marshaler for Outcome
func (o Outcome) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(o))
}

// UnmarshalJSON implements json.Unmarshaler for Outcome
func (o *Outcome) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*o = Outcome(s)
	return nil
}

// MarshalJSON implements json.Marshaler for Difficulty
func (d Difficulty) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(d))
}

// UnmarshalJSON implements json.Unmarshaler for Difficulty
func (d *Difficulty) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*d = Difficulty(s)
	return nil
}

// MarshalJSON implements json.Marshaler for ConversationFlow
func (cf ConversationFlow) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(cf))
}

// UnmarshalJSON implements json.Unmarshaler for ConversationFlow
func (cf *ConversationFlow) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*cf = ConversationFlow(s)
	return nil
}

// MarshalJSON implements json.Marshaler for Recency
func (r Recency) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

// UnmarshalJSON implements json.Unmarshaler for Recency
func (r *Recency) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*r = Recency(s)
	return nil
}
