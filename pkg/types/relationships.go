package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RelationType represents the type of relationship between memory chunks
type RelationType string

const (
	// RelationLedTo indicates that one chunk led to another (e.g., problem led to solution)
	RelationLedTo RelationType = "led_to" // Problem led to solution
	// RelationSolvedBy indicates that one chunk was solved by another
	RelationSolvedBy RelationType = "solved_by" // Problem solved by solution

	// RelationDependsOn indicates that one chunk depends on another
	RelationDependsOn RelationType = "depends_on" // Feature depends on another
	// RelationEnables indicates that one chunk enables another
	RelationEnables RelationType = "enables" // Decision enables feature
	// RelationImplements indicates that one chunk implements another
	RelationImplements RelationType = "implements" // Implementation of design/spec

	// RelationConflictsWith indicates that two chunks are in conflict
	RelationConflictsWith RelationType = "conflicts_with" // Decisions conflict
	// RelationSupersedes indicates that one chunk supersedes another
	RelationSupersedes RelationType = "supersedes" // New decision supersedes old

	// RelationRelatedTo indicates a general relationship between chunks
	RelationRelatedTo RelationType = "related_to" // General relation
	// RelationFollowsUp indicates that one chunk follows up on another
	RelationFollowsUp RelationType = "follows_up" // Follow-up discussion
	// RelationPrecedes indicates temporal precedence
	RelationPrecedes RelationType = "precedes" // Temporal precedence

	// RelationLearnedFrom indicates that knowledge was derived from another chunk
	RelationLearnedFrom RelationType = "learned_from" // Knowledge derived from
	// RelationTeaches indicates that one chunk teaches a concept
	RelationTeaches RelationType = "teaches" // Knowledge teaches concept
	// RelationExemplifes indicates that one chunk exemplifies a pattern
	RelationExemplifes RelationType = "exemplifies" // Example of pattern

	// RelationReferencesBy indicates that one chunk is referenced by another
	RelationReferencesBy RelationType = "referenced_by" // Chunk referenced by another
	// RelationReferences indicates that one chunk references another
	RelationReferences RelationType = "references" // Chunk references another
)

// AllValidRelationTypes returns all valid relation types
func AllValidRelationTypes() []RelationType {
	return []RelationType{
		RelationLedTo, RelationSolvedBy, RelationDependsOn, RelationEnables, RelationImplements,
		RelationConflictsWith, RelationSupersedes, RelationRelatedTo, RelationFollowsUp,
		RelationPrecedes, RelationLearnedFrom, RelationTeaches, RelationExemplifes,
		RelationReferencesBy, RelationReferences,
	}
}

// Valid returns true if the relation type is valid
func (rt RelationType) Valid() bool {
	for _, validType := range AllValidRelationTypes() {
		if rt == validType {
			return true
		}
	}
	return false
}

// GetInverse returns the inverse relationship type
func (rt RelationType) GetInverse() RelationType {
	if rt.IsSymmetric() {
		return rt
	}
	
	if inverse, exists := rt.getBidirectionalInverse(); exists {
		return inverse
	}
	
	return rt.getSelfReferencingDefault()
}

// IsSymmetric returns true if the relationship is symmetric
func (rt RelationType) IsSymmetric() bool {
	symmetricRelations := []RelationType{
		RelationConflictsWith,
		RelationRelatedTo,
		RelationImplements,
		RelationSupersedes,
		RelationExemplifes,
	}
	
	for _, symmetric := range symmetricRelations {
		if rt == symmetric {
			return true
		}
	}
	return false
}

// getBidirectionalInverse returns the inverse for bidirectional relationships
func (rt RelationType) getBidirectionalInverse() (RelationType, bool) {
	bidirectionalMap := map[RelationType]RelationType{
		RelationLedTo:         RelationSolvedBy,
		RelationSolvedBy:      RelationLedTo,
		RelationDependsOn:     RelationEnables,
		RelationEnables:       RelationDependsOn,
		RelationFollowsUp:     RelationPrecedes,
		RelationPrecedes:      RelationFollowsUp,
		RelationLearnedFrom:   RelationTeaches,
		RelationTeaches:       RelationLearnedFrom,
		RelationReferencesBy:  RelationReferences,
		RelationReferences:    RelationReferencesBy,
	}
	
	inverse, exists := bidirectionalMap[rt]
	return inverse, exists
}

// getSelfReferencingDefault returns default relation for unknown types
func (rt RelationType) getSelfReferencingDefault() RelationType {
	return RelationRelatedTo
}


// ConfidenceSource represents how the confidence score was determined
type ConfidenceSource string

const (
	// ConfidenceExplicit indicates confidence was explicitly stated by user
	ConfidenceExplicit ConfidenceSource = "explicit" // User explicitly stated
	// ConfidenceInferred indicates confidence was inferred by AI from context
	ConfidenceInferred ConfidenceSource = "inferred" // AI inferred from context
	// ConfidenceDerived indicates confidence was calculated from other data
	ConfidenceDerived ConfidenceSource = "derived" // Calculated from other data
	// ConfidenceAuto indicates confidence was automatically detected
	ConfidenceAuto ConfidenceSource = "auto" // Automatically detected
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
		return nil, errors.New("source chunk ID cannot be empty")
	}
	if targetID == "" {
		return nil, errors.New("target chunk ID cannot be empty")
	}
	if sourceID == targetID {
		return nil, errors.New("source and target chunk IDs cannot be the same")
	}
	if !relationType.Valid() {
		return nil, fmt.Errorf("invalid relation type: %s", relationType)
	}
	if confidence < 0 || confidence > 1 {
		return nil, errors.New("confidence must be between 0 and 1")
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
		return errors.New("ID cannot be empty")
	}
	if mr.SourceChunkID == "" {
		return errors.New("source chunk ID cannot be empty")
	}
	if mr.TargetChunkID == "" {
		return errors.New("target chunk ID cannot be empty")
	}
	if mr.SourceChunkID == mr.TargetChunkID {
		return errors.New("source and target chunk IDs cannot be the same")
	}
	if !mr.RelationType.Valid() {
		return fmt.Errorf("invalid relation type: %s", mr.RelationType)
	}
	if mr.Confidence < 0 || mr.Confidence > 1 {
		return errors.New("confidence must be between 0 and 1")
	}
	if !mr.ConfidenceSource.Valid() {
		return fmt.Errorf("invalid confidence source: %s", mr.ConfidenceSource)
	}
	if mr.ValidationCount < 0 {
		return errors.New("validation count cannot be negative")
	}
	if mr.CreatedAt.IsZero() {
		return errors.New("created at cannot be zero")
	}
	return nil
}

// UpdateConfidence updates the confidence score and factors
func (mr *MemoryRelationship) UpdateConfidence(newConfidence float64, factors ConfidenceFactors) error {
	if newConfidence < 0 || newConfidence > 1 {
		return errors.New("confidence must be between 0 and 1")
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
	if err := rq.validateBasicFields(); err != nil {
		return err
	}
	
	if err := rq.validateRangeFields(); err != nil {
		return err
	}
	
	if err := rq.validateSortFields(); err != nil {
		return err
	}
	
	return rq.validateRelationTypes()
}

// validateBasicFields validates basic required fields
func (rq *RelationshipQuery) validateBasicFields() error {
	if rq.ChunkID == "" {
		return errors.New("chunk ID cannot be empty")
	}
	
	validDirections := []string{"incoming", "outgoing", "both"}
	if !rq.isValidDirection(validDirections) {
		return errors.New("direction must be 'incoming', 'outgoing', or 'both'")
	}
	
	return nil
}

// validateRangeFields validates numeric range fields
func (rq *RelationshipQuery) validateRangeFields() error {
	if rq.MinConfidence < 0 || rq.MinConfidence > 1 {
		return errors.New("min confidence must be between 0 and 1")
	}
	
	if rq.MaxDepth < 1 {
		return errors.New("max depth must be at least 1")
	}
	
	if rq.Limit < 0 {
		return errors.New("limit cannot be negative")
	}
	
	return nil
}

// validateSortFields validates sorting related fields
func (rq *RelationshipQuery) validateSortFields() error {
	validSortFields := []string{"", "confidence", "created_at", "validation_count"}
	if !rq.isValidSortBy(validSortFields) {
		return errors.New("sort by must be 'confidence', 'created_at', or 'validation_count'")
	}
	
	validSortOrders := []string{"", "asc", "desc"}
	if !rq.isValidSortOrder(validSortOrders) {
		return errors.New("sort order must be 'asc' or 'desc'")
	}
	
	return nil
}

// validateRelationTypes validates all relation types in the query
func (rq *RelationshipQuery) validateRelationTypes() error {
	for _, relType := range rq.RelationTypes {
		if !relType.Valid() {
			return fmt.Errorf("invalid relation type: %s", relType)
		}
	}
	return nil
}

// isValidDirection checks if direction is valid
func (rq *RelationshipQuery) isValidDirection(validDirections []string) bool {
	for _, valid := range validDirections {
		if rq.Direction == valid {
			return true
		}
	}
	return false
}

// isValidSortBy checks if sort by field is valid
func (rq *RelationshipQuery) isValidSortBy(validFields []string) bool {
	for _, valid := range validFields {
		if rq.SortBy == valid {
			return true
		}
	}
	return false
}

// isValidSortOrder checks if sort order is valid
func (rq *RelationshipQuery) isValidSortOrder(validOrders []string) bool {
	for _, valid := range validOrders {
		if rq.SortOrder == valid {
			return true
		}
	}
	return false
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

// MarshalJSON implements json.Marshaler for RelationType
func (rt RelationType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(rt))
}

// UnmarshalJSON implements json.Unmarshaler for RelationType
func (rt *RelationType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*rt = RelationType(s)
	return nil
}

// MarshalJSON implements json.Marshaler for ConfidenceSource
func (cs ConfidenceSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(cs))
}

// UnmarshalJSON implements json.Unmarshaler for ConfidenceSource
func (cs *ConfidenceSource) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*cs = ConfidenceSource(s)
	return nil
}
