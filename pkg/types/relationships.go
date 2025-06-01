package types

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RelationType represents the type of relationship between memory chunks
type RelationType string

const (
	// Causal relationships
	RelationLedTo    RelationType = "led_to"    // Problem led to solution
	RelationSolvedBy RelationType = "solved_by" // Problem solved by solution

	// Dependency relationships
	RelationDependsOn RelationType = "depends_on" // Feature depends on another
	RelationEnables   RelationType = "enables"    // Decision enables feature

	// Conflict relationships
	RelationConflictsWith RelationType = "conflicts_with" // Decisions conflict
	RelationSupersedes    RelationType = "supersedes"     // New decision supersedes old

	// Contextual relationships
	RelationRelatedTo RelationType = "related_to" // General relation
	RelationFollowsUp RelationType = "follows_up" // Follow-up discussion
	RelationPrecedes  RelationType = "precedes"   // Temporal precedence

	// Learning relationships
	RelationLearnedFrom RelationType = "learned_from" // Knowledge derived from
	RelationTeaches     RelationType = "teaches"      // Knowledge teaches concept
	RelationExemplifes  RelationType = "exemplifies"  // Example of pattern

	// Reference relationships
	RelationReferencesBy RelationType = "referenced_by" // Chunk referenced by another
	RelationReferences   RelationType = "references"    // Chunk references another
)

// Valid returns true if the relation type is valid
func (rt RelationType) Valid() bool {
	switch rt {
	case RelationLedTo, RelationSolvedBy, RelationDependsOn, RelationEnables,
		RelationConflictsWith, RelationSupersedes, RelationRelatedTo, RelationFollowsUp,
		RelationPrecedes, RelationLearnedFrom, RelationTeaches, RelationExemplifes,
		RelationReferencesBy, RelationReferences:
		return true
	}
	return false
}

// GetInverse returns the inverse relationship type
func (rt RelationType) GetInverse() RelationType {
	switch rt {
	case RelationLedTo:
		return RelationSolvedBy
	case RelationSolvedBy:
		return RelationLedTo
	case RelationDependsOn:
		return RelationEnables
	case RelationEnables:
		return RelationDependsOn
	case RelationConflictsWith:
		return RelationConflictsWith // Symmetric
	case RelationSupersedes:
		return RelationSupersedes // No direct inverse
	case RelationRelatedTo:
		return RelationRelatedTo // Symmetric
	case RelationFollowsUp:
		return RelationPrecedes
	case RelationPrecedes:
		return RelationFollowsUp
	case RelationLearnedFrom:
		return RelationTeaches
	case RelationTeaches:
		return RelationLearnedFrom
	case RelationExemplifes:
		return RelationExemplifes // No direct inverse
	case RelationReferencesBy:
		return RelationReferences
	case RelationReferences:
		return RelationReferencesBy
	default:
		return RelationRelatedTo
	}
}

// IsSymmetric returns true if the relationship is symmetric
func (rt RelationType) IsSymmetric() bool {
	switch rt {
	case RelationConflictsWith, RelationRelatedTo:
		return true
	case RelationLedTo, RelationSolvedBy, RelationDependsOn, RelationEnables,
		RelationSupersedes, RelationFollowsUp, RelationPrecedes, RelationLearnedFrom,
		RelationTeaches, RelationExemplifes, RelationReferencesBy, RelationReferences:
		return false
	}
	return false
}

// ConfidenceSource represents how the confidence score was determined
type ConfidenceSource string

const (
	ConfidenceExplicit ConfidenceSource = "explicit" // User explicitly stated
	ConfidenceInferred ConfidenceSource = "inferred" // AI inferred from context
	ConfidenceDerived  ConfidenceSource = "derived"  // Calculated from other data
	ConfidenceAuto     ConfidenceSource = "auto"     // Automatically detected
)

// Valid returns true if the confidence source is valid
func (cs ConfidenceSource) Valid() bool {
	switch cs {
	case ConfidenceExplicit, ConfidenceInferred, ConfidenceDerived, ConfidenceAuto:
		return true
	}
	return false
}

// Note: ConfidenceFactors is now defined in types.go for reuse across packages

// MemoryRelationship represents a relationship between memory chunks
type MemoryRelationship struct {
	ID                string                 `json:"id"`
	SourceChunkID     string                 `json:"source_chunk_id"`
	TargetChunkID     string                 `json:"target_chunk_id"`
	RelationType      RelationType           `json:"relation_type"`
	Confidence        float64                `json:"confidence"` // 0.0-1.0
	ConfidenceSource  ConfidenceSource       `json:"confidence_source"`
	ConfidenceFactors ConfidenceFactors      `json:"confidence_factors,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	CreatedBy         string                 `json:"created_by,omitempty"` // User/system that created
	LastValidated     *time.Time             `json:"last_validated,omitempty"`
	ValidationCount   int                    `json:"validation_count"`
}

// NewMemoryRelationship creates a new memory relationship
func NewMemoryRelationship(sourceID, targetID string, relationType RelationType, confidence float64, source ConfidenceSource) (*MemoryRelationship, error) {
	if sourceID == "" {
		return nil, fmt.Errorf("source chunk ID cannot be empty")
	}
	if targetID == "" {
		return nil, fmt.Errorf("target chunk ID cannot be empty")
	}
	if sourceID == targetID {
		return nil, fmt.Errorf("source and target chunk IDs cannot be the same")
	}
	if !relationType.Valid() {
		return nil, fmt.Errorf("invalid relation type: %s", relationType)
	}
	if confidence < 0 || confidence > 1 {
		return nil, fmt.Errorf("confidence must be between 0 and 1")
	}
	if !source.Valid() {
		return nil, fmt.Errorf("invalid confidence source: %s", source)
	}

	return &MemoryRelationship{
		ID:                uuid.New().String(),
		SourceChunkID:     sourceID,
		TargetChunkID:     targetID,
		RelationType:      relationType,
		Confidence:        confidence,
		ConfidenceSource:  source,
		ConfidenceFactors: ConfidenceFactors{},
		Metadata:          make(map[string]interface{}),
		CreatedAt:         time.Now().UTC(),
		ValidationCount:   0,
	}, nil
}

// Validate checks if the memory relationship is valid
func (mr *MemoryRelationship) Validate() error {
	if mr.ID == "" {
		return fmt.Errorf("ID cannot be empty")
	}
	if mr.SourceChunkID == "" {
		return fmt.Errorf("source chunk ID cannot be empty")
	}
	if mr.TargetChunkID == "" {
		return fmt.Errorf("target chunk ID cannot be empty")
	}
	if mr.SourceChunkID == mr.TargetChunkID {
		return fmt.Errorf("source and target chunk IDs cannot be the same")
	}
	if !mr.RelationType.Valid() {
		return fmt.Errorf("invalid relation type: %s", mr.RelationType)
	}
	if mr.Confidence < 0 || mr.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	if !mr.ConfidenceSource.Valid() {
		return fmt.Errorf("invalid confidence source: %s", mr.ConfidenceSource)
	}
	if mr.ValidationCount < 0 {
		return fmt.Errorf("validation count cannot be negative")
	}
	if mr.CreatedAt.IsZero() {
		return fmt.Errorf("created at cannot be zero")
	}
	return nil
}

// UpdateConfidence updates the confidence score and factors
func (mr *MemoryRelationship) UpdateConfidence(newConfidence float64, factors ConfidenceFactors) error {
	if newConfidence < 0 || newConfidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	mr.Confidence = newConfidence
	mr.ConfidenceFactors = factors
	mr.ValidationCount++
	now := time.Now().UTC()
	mr.LastValidated = &now
	return nil
}

// Note: QualityMetrics is now defined in types.go for reuse across packages

// RelationshipQuery represents a query for finding relationships
type RelationshipQuery struct {
	ChunkID       string         `json:"chunk_id"`
	RelationTypes []RelationType `json:"relation_types,omitempty"`
	Direction     string         `json:"direction"`      // "incoming", "outgoing", "both"
	MinConfidence float64        `json:"min_confidence"` // 0.0-1.0
	MaxDepth      int            `json:"max_depth"`      // For graph traversal
	IncludeChunks bool           `json:"include_chunks"` // Include full chunk data
	SortBy        string         `json:"sort_by"`        // "confidence", "created_at", "validation_count"
	SortOrder     string         `json:"sort_order"`     // "asc", "desc"
	Limit         int            `json:"limit,omitempty"`
}

// NewRelationshipQuery creates a new relationship query with defaults
func NewRelationshipQuery(chunkID string) *RelationshipQuery {
	return &RelationshipQuery{
		ChunkID:       chunkID,
		Direction:     "both",
		MinConfidence: 0.5,
		MaxDepth:      3,
		IncludeChunks: true,
		SortBy:        "confidence",
		SortOrder:     "desc",
		Limit:         50,
	}
}

// Validate checks if the relationship query is valid
func (rq *RelationshipQuery) Validate() error {
	if rq.ChunkID == "" {
		return fmt.Errorf("chunk ID cannot be empty")
	}
	if rq.Direction != "incoming" && rq.Direction != "outgoing" && rq.Direction != "both" {
		return fmt.Errorf("direction must be 'incoming', 'outgoing', or 'both'")
	}
	if rq.MinConfidence < 0 || rq.MinConfidence > 1 {
		return fmt.Errorf("min confidence must be between 0 and 1")
	}
	if rq.MaxDepth < 1 {
		return fmt.Errorf("max depth must be at least 1")
	}
	if rq.SortBy != "" && rq.SortBy != "confidence" && rq.SortBy != "created_at" && rq.SortBy != "validation_count" {
		return fmt.Errorf("sort by must be 'confidence', 'created_at', or 'validation_count'")
	}
	if rq.SortOrder != "" && rq.SortOrder != "asc" && rq.SortOrder != "desc" {
		return fmt.Errorf("sort order must be 'asc' or 'desc'")
	}
	if rq.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	for _, relType := range rq.RelationTypes {
		if !relType.Valid() {
			return fmt.Errorf("invalid relation type: %s", relType)
		}
	}
	return nil
}

// RelationshipResult represents a relationship with optional chunk data
type RelationshipResult struct {
	Relationship MemoryRelationship `json:"relationship"`
	SourceChunk  *ConversationChunk `json:"source_chunk,omitempty"`
	TargetChunk  *ConversationChunk `json:"target_chunk,omitempty"`
}

// GraphTraversalResult represents the result of graph traversal
type GraphTraversalResult struct {
	Paths []GraphPath `json:"paths"`
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphPath represents a path through the knowledge graph
type GraphPath struct {
	ChunkIDs []string `json:"chunk_ids"`
	Score    float64  `json:"score"` // Combined confidence score
	Depth    int      `json:"depth"`
	PathType string   `json:"path_type"` // e.g., "problem_to_solution"
}

// GraphNode represents a node in the knowledge graph
type GraphNode struct {
	ChunkID    string             `json:"chunk_id"`
	Chunk      *ConversationChunk `json:"chunk,omitempty"`
	Degree     int                `json:"degree"`     // Number of connections
	Centrality float64            `json:"centrality"` // Importance in graph
}

// GraphEdge represents an edge in the knowledge graph
type GraphEdge struct {
	Relationship MemoryRelationship `json:"relationship"`
	Weight       float64            `json:"weight"` // Based on confidence
}

// JSON marshaling for custom types
func (rt RelationType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(rt))
}

func (rt *RelationType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*rt = RelationType(s)
	return nil
}

func (cs ConfidenceSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(cs))
}

func (cs *ConfidenceSource) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*cs = ConfidenceSource(s)
	return nil
}
