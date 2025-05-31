package types

import (
	"encoding/json"
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
	TimeframWeek    = "week"
	TimeframeMonth  = "month"
	TimeframeQuarter = "quarter"
	TimeframeAll    = "all"
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
	ChunkTypeProblem              ChunkType = "problem"
	ChunkTypeSolution             ChunkType = "solution"
	ChunkTypeCodeChange           ChunkType = "code_change"
	ChunkTypeDiscussion           ChunkType = "discussion"
	ChunkTypeArchitectureDecision ChunkType = "architecture_decision"
	ChunkTypeSessionSummary       ChunkType = "session_summary"
	ChunkTypeAnalysis             ChunkType = "analysis"
	ChunkTypeVerification         ChunkType = "verification"
	ChunkTypeQuestion             ChunkType = "question"
)

// Valid returns true if the chunk type is valid
func (ct ChunkType) Valid() bool {
	switch ct {
	case ChunkTypeProblem, ChunkTypeSolution, ChunkTypeCodeChange, ChunkTypeDiscussion, ChunkTypeArchitectureDecision, ChunkTypeSessionSummary, ChunkTypeAnalysis, ChunkTypeVerification, ChunkTypeQuestion:
		return true
	}
	return false
}

// Outcome represents the outcome of a conversation chunk
type Outcome string

const (
	OutcomeSuccess    Outcome = "success"
	OutcomeInProgress Outcome = "in_progress"
	OutcomeFailed     Outcome = "failed"
	OutcomeAbandoned  Outcome = "abandoned"
)

// Valid returns true if the outcome is valid
func (o Outcome) Valid() bool {
	switch o {
	case OutcomeSuccess, OutcomeInProgress, OutcomeFailed, OutcomeAbandoned:
		return true
	}
	return false
}

// Difficulty represents the difficulty level of a task
type Difficulty string

const (
	DifficultySimple   Difficulty = "simple"
	DifficultyModerate Difficulty = "moderate"
	DifficultyComplex  Difficulty = "complex"
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
	FlowProblem       ConversationFlow = "problem"
	FlowInvestigation ConversationFlow = "investigation"
	FlowSolution      ConversationFlow = "solution"
	FlowVerification  ConversationFlow = "verification"
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
	RecencyRecent    Recency = "recent"
	RecencyAllTime   Recency = "all_time"
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
	Confidence       *ConfidenceMetrics     `json:"confidence,omitempty"`
	Quality          *QualityMetrics        `json:"quality,omitempty"`
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
		return fmt.Errorf("time spent cannot be negative")
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
		return nil, fmt.Errorf("session ID cannot be empty")
	}
	if content == "" {
		return nil, fmt.Errorf("content cannot be empty")
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
		return fmt.Errorf("ID cannot be empty")
	}
	if cc.SessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if cc.Content == "" {
		return fmt.Errorf("content cannot be empty")
	}
	if !cc.Type.Valid() {
		return fmt.Errorf("invalid chunk type: %s", cc.Type)
	}
	if cc.Timestamp.IsZero() {
		return fmt.Errorf("timestamp cannot be zero")
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
		return fmt.Errorf("repository cannot be empty")
	}
	if pc.TotalSessions < 0 {
		return fmt.Errorf("total sessions cannot be negative")
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
		return fmt.Errorf("query cannot be empty")
	}
	if !mq.Recency.Valid() {
		return fmt.Errorf("invalid recency: %s", mq.Recency)
	}
	if mq.MinRelevanceScore < 0 || mq.MinRelevanceScore > 1 {
		return fmt.Errorf("min relevance score must be between 0 and 1")
	}
	if mq.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
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
	Score            float64           `json:"score"`             // 0.0 to 1.0
	Source           string            `json:"source"`            // explicit, inferred, derived, auto
	Factors          ConfidenceFactors `json:"factors,omitempty"`
	LastUpdated      *time.Time        `json:"last_updated,omitempty"`
	ValidationCount  int               `json:"validation_count"`
}

// QualityMetrics represents quality metrics for a memory chunk  
type QualityMetrics struct {
	Completeness    float64   `json:"completeness"`     // How complete is this memory (0.0-1.0)
	Clarity         float64   `json:"clarity"`          // How clear/unambiguous (0.0-1.0)
	RelevanceDecay  float64   `json:"relevance_decay"`  // How much relevance has decayed (0.0-1.0)
	FreshnessScore  float64   `json:"freshness_score"`  // How fresh/current (0.0-1.0)
	UsageScore      float64   `json:"usage_score"`      // Based on access patterns (0.0-1.0)
	OverallQuality  float64   `json:"overall_quality"`  // Weighted combination (0.0-1.0)
	LastCalculated  *time.Time `json:"last_calculated,omitempty"`
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
		return fmt.Errorf("time elapsed cannot be negative")
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

func (ct ChunkType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(ct))
}

func (ct *ChunkType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*ct = ChunkType(s)
	return nil
}

func (o Outcome) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(o))
}

func (o *Outcome) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*o = Outcome(s)
	return nil
}

func (d Difficulty) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(d))
}

func (d *Difficulty) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*d = Difficulty(s)
	return nil
}

func (cf ConversationFlow) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(cf))
}

func (cf *ConversationFlow) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*cf = ConversationFlow(s)
	return nil
}

func (r Recency) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(r))
}

func (r *Recency) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*r = Recency(s)
	return nil
}
