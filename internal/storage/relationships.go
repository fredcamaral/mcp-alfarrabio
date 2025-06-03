package storage

import (
	"context"
	"fmt"
	"math"
	"mcp-memory/pkg/types"
	"sort"
	"strings"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

const (
	relationshipsCollection = "memory_relationships"
)

// RelationshipStore manages memory relationships in Qdrant
type RelationshipStore struct {
	client         *qdrant.Client
	collectionName string
	metrics        *StorageMetrics
}

// NewRelationshipStore creates a new relationship store
func NewRelationshipStore(client *qdrant.Client) *RelationshipStore {
	return &RelationshipStore{
		client:         client,
		collectionName: relationshipsCollection,
		metrics: &StorageMetrics{
			OperationCounts:  make(map[string]int64),
			AverageLatency:   make(map[string]float64),
			ErrorCounts:      make(map[string]int64),
			ConnectionStatus: "unknown",
		},
	}
}

// Initialize creates the relationships collection if it doesn't exist
func (rs *RelationshipStore) Initialize(ctx context.Context) error {
	start := time.Now()
	defer rs.updateMetrics("initialize", start)

	// Check if collection exists
	collections, err := rs.client.ListCollections(ctx)
	if err != nil {
		rs.metrics.ConnectionStatus = connectionStatusError
		return fmt.Errorf("failed to list collections: %w", err)
	}

	// Check if our collection exists
	collectionExists := false
	for _, collectionName := range collections {
		if collectionName == rs.collectionName {
			collectionExists = true
			break
		}
	}

	// Create collection if it doesn't exist
	if !collectionExists {
		// Relationships don't need vector embeddings, just metadata storage
		err = rs.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: rs.collectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     1, // Minimal vector size since we don't use vectors for relationships
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			rs.metrics.ConnectionStatus = connectionStatusError
			return fmt.Errorf("failed to create relationships collection: %w", err)
		}
	}

	rs.metrics.ConnectionStatus = "connected"
	return nil
}

// Store saves a memory relationship
func (rs *RelationshipStore) Store(ctx context.Context, relationship types.MemoryRelationship) error {
	start := time.Now()
	defer rs.updateMetrics("store", start)

	if err := relationship.Validate(); err != nil {
		return fmt.Errorf("invalid relationship: %w", err)
	}

	// Convert relationship to Qdrant point
	point := rs.relationshipToPoint(relationship)

	// Upsert point to collection
	_, err := rs.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: rs.collectionName,
		Points:         []*qdrant.PointStruct{point},
	})

	if err != nil {
		return fmt.Errorf("failed to store relationship in Qdrant: %w", err)
	}

	return nil
}

// StoreRelationship creates and stores a new relationship
func (rs *RelationshipStore) StoreRelationship(ctx context.Context, sourceID, targetID string, relationType types.RelationType, confidence float64, source types.ConfidenceSource) (*types.MemoryRelationship, error) {
	relationship, err := types.NewMemoryRelationship(sourceID, targetID, relationType, confidence, source)
	if err != nil {
		return nil, fmt.Errorf("failed to create relationship: %w", err)
	}

	if err := rs.Store(ctx, *relationship); err != nil {
		return nil, err
	}

	// Store inverse relationship if symmetric
	if relationType.IsSymmetric() {
		inverseRelationship, err := types.NewMemoryRelationship(targetID, sourceID, relationType, confidence, source)
		if err != nil {
			return nil, fmt.Errorf("failed to create inverse relationship: %w", err)
		}
		if err := rs.Store(ctx, *inverseRelationship); err != nil {
			return nil, fmt.Errorf("failed to store inverse relationship: %w", err)
		}
	}

	return relationship, nil
}

// GetRelationships finds relationships for a chunk
func (rs *RelationshipStore) GetRelationships(ctx context.Context, query types.RelationshipQuery) ([]types.RelationshipResult, error) {
	start := time.Now()
	defer rs.updateMetrics("get_relationships", start)

	if err := query.Validate(); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	// Build filter based on query direction
	filter := rs.buildRelationshipFilter(query)

	// Search for relationships
	points, err := rs.client.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: rs.collectionName,
		Filter:         filter,
		Limit:          rs.calculateLimit(query.Limit),
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get relationships: %w", err)
	}

	// Convert points to relationships
	relationships := make([]types.RelationshipResult, 0, len(points))
	for _, point := range points {
		relationship, err := rs.pointToRelationship(point)
		if err != nil {
			continue // Skip invalid relationships
		}

		// Filter by confidence
		if relationship.Confidence < query.MinConfidence {
			continue
		}

		// Filter by relation types if specified
		if len(query.RelationTypes) > 0 {
			found := false
			for _, rt := range query.RelationTypes {
				if relationship.RelationType == rt {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		result := types.RelationshipResult{
			Relationship: *relationship,
		}
		relationships = append(relationships, result)
	}

	// Sort relationships
	rs.sortRelationships(relationships, query.SortBy, query.SortOrder)

	// Apply limit
	if query.Limit > 0 && len(relationships) > query.Limit {
		relationships = relationships[:query.Limit]
	}

	return relationships, nil
}

// TraverseGraph traverses the knowledge graph starting from a chunk
func (rs *RelationshipStore) TraverseGraph(ctx context.Context, startChunkID string, maxDepth int, relationTypes []types.RelationType) (*types.GraphTraversalResult, error) {
	start := time.Now()
	defer rs.updateMetrics("traverse_graph", start)

	if maxDepth <= 0 {
		maxDepth = 3
	}

	visited := make(map[string]bool)
	paths := make([]types.GraphPath, 0)
	nodes := make(map[string]*types.GraphNode)
	edges := make(map[string]*types.GraphEdge)

	// Recursive traversal function
	var traverse func(chunkID string, currentPath []string, currentScore float64, depth int)
	traverse = func(chunkID string, currentPath []string, currentScore float64, depth int) {
		if depth > maxDepth || visited[chunkID] {
			return
		}

		visited[chunkID] = true
		currentPath = append(currentPath, chunkID)

		// Add node if not exists
		if _, exists := nodes[chunkID]; !exists {
			nodes[chunkID] = &types.GraphNode{
				ChunkID:    chunkID,
				Degree:     0,
				Centrality: 0.0,
			}
		}

		// Get relationships for current chunk
		query := types.NewRelationshipQuery(chunkID)
		query.RelationTypes = relationTypes
		query.MaxDepth = 1 // Only direct relationships
		relationships, err := rs.GetRelationships(ctx, *query)
		if err != nil {
			return
		}

		// Follow each relationship
		for _, rel := range relationships {
			relationship := rel.Relationship
			var targetID string

			// Determine target based on direction
			if relationship.SourceChunkID == chunkID {
				targetID = relationship.TargetChunkID
			} else {
				targetID = relationship.SourceChunkID
			}

			// Add edge
			edgeKey := fmt.Sprintf("%s-%s", relationship.SourceChunkID, relationship.TargetChunkID)
			edges[edgeKey] = &types.GraphEdge{
				Relationship: relationship,
				Weight:       relationship.Confidence,
			}

			// Update node degrees
			nodes[chunkID].Degree++
			if _, exists := nodes[targetID]; !exists {
				nodes[targetID] = &types.GraphNode{
					ChunkID:    targetID,
					Degree:     0,
					Centrality: 0.0,
				}
			}
			nodes[targetID].Degree++

			// Calculate path score (average confidence)
			newScore := (currentScore*float64(len(currentPath)-1) + relationship.Confidence) / float64(len(currentPath))

			// Continue traversal
			traverse(targetID, currentPath, newScore, depth+1)
		}

		// Add path if it has multiple nodes
		if len(currentPath) > 1 {
			pathType := rs.determinePathType(currentPath, relationships)
			paths = append(paths, types.GraphPath{
				ChunkIDs: append([]string{}, currentPath...),
				Score:    currentScore,
				Depth:    len(currentPath) - 1,
				PathType: pathType,
			})
		}
	}

	// Start traversal
	traverse(startChunkID, []string{}, 1.0, 0)

	// Calculate centrality scores
	rs.calculateCentrality(nodes)

	// Convert maps to slices
	nodeSlice := make([]types.GraphNode, 0, len(nodes))
	for _, node := range nodes {
		nodeSlice = append(nodeSlice, *node)
	}

	edgeSlice := make([]types.GraphEdge, 0, len(edges))
	for _, edge := range edges {
		edgeSlice = append(edgeSlice, *edge)
	}

	return &types.GraphTraversalResult{
		Paths: paths,
		Nodes: nodeSlice,
		Edges: edgeSlice,
	}, nil
}

// UpdateRelationship updates an existing relationship
func (rs *RelationshipStore) UpdateRelationship(ctx context.Context, relationshipID string, confidence float64, factors types.ConfidenceFactors) error {
	start := time.Now()
	defer rs.updateMetrics("update_relationship", start)

	// Get existing relationship
	relationship, err := rs.GetByID(ctx, relationshipID)
	if err != nil {
		return fmt.Errorf("relationship not found: %w", err)
	}

	// Update confidence
	if err := relationship.UpdateConfidence(confidence, factors); err != nil {
		return fmt.Errorf("failed to update confidence: %w", err)
	}

	// Store updated relationship
	return rs.Store(ctx, *relationship)
}

// GetByID retrieves a relationship by ID
func (rs *RelationshipStore) GetByID(ctx context.Context, id string) (*types.MemoryRelationship, error) {
	start := time.Now()
	defer rs.updateMetrics("get_by_id", start)

	// Get point by ID
	points, err := rs.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: rs.collectionName,
		Ids:            []*qdrant.PointId{rs.stringToPointID(id)},
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get relationship: %w", err)
	}

	if len(points) == 0 {
		return nil, fmt.Errorf("relationship not found: %s", id)
	}

	return rs.pointToRelationship(points[0])
}

// Delete removes a relationship
func (rs *RelationshipStore) Delete(ctx context.Context, id string) error {
	start := time.Now()
	defer rs.updateMetrics("delete", start)

	_, err := rs.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: rs.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{rs.stringToPointID(id)},
				},
			},
		},
	})

	return err
}

// Helper methods

// relationshipToPoint converts a MemoryRelationship to Qdrant point
func (rs *RelationshipStore) relationshipToPoint(relationship types.MemoryRelationship) *qdrant.PointStruct {
	payload := map[string]*qdrant.Value{
		"source_chunk_id":   rs.stringToValue(relationship.SourceChunkID),
		"target_chunk_id":   rs.stringToValue(relationship.TargetChunkID),
		"relation_type":     rs.stringToValue(string(relationship.RelationType)),
		"confidence":        rs.float64ToValue(relationship.Confidence),
		"confidence_source": rs.stringToValue(string(relationship.ConfidenceSource)),
		"created_at":        rs.int64ToValue(relationship.CreatedAt.Unix()),
		"created_by":        rs.stringToValue(relationship.CreatedBy),
		"validation_count":  rs.int64ToValue(int64(relationship.ValidationCount)),
	}

	// Add confidence factors
	if relationship.ConfidenceFactors.UserCertainty != nil {
		payload["user_certainty"] = rs.float64ToValue(*relationship.ConfidenceFactors.UserCertainty)
	}
	if relationship.ConfidenceFactors.ConsistencyScore != nil {
		payload["consistency_score"] = rs.float64ToValue(*relationship.ConfidenceFactors.ConsistencyScore)
	}
	if relationship.ConfidenceFactors.CorroborationCount != nil {
		payload["corroboration_count"] = rs.int64ToValue(int64(*relationship.ConfidenceFactors.CorroborationCount))
	}

	// Add last validated timestamp
	if relationship.LastValidated != nil {
		payload["last_validated"] = rs.int64ToValue(relationship.LastValidated.Unix())
	}

	// Add metadata
	for key, value := range relationship.Metadata {
		if strValue, ok := value.(string); ok {
			payload["meta_"+key] = rs.stringToValue(strValue)
		}
	}

	return &qdrant.PointStruct{
		Id:      rs.stringToPointID(relationship.ID),
		Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{Data: []float32{0.0}}}}, // Dummy vector
		Payload: payload,
	}
}

// pointToRelationship converts a Qdrant point to MemoryRelationship
func (rs *RelationshipStore) pointToRelationship(point *qdrant.RetrievedPoint) (*types.MemoryRelationship, error) {
	payload := point.GetPayload()
	id := rs.pointIDToString(point.GetId())

	// Parse required fields
	sourceChunkID := rs.getStringFromPayload(payload, "source_chunk_id")
	targetChunkID := rs.getStringFromPayload(payload, "target_chunk_id")
	relationTypeStr := rs.getStringFromPayload(payload, "relation_type")
	confidence := rs.getFloat64FromPayload(payload, "confidence")
	confidenceSourceStr := rs.getStringFromPayload(payload, "confidence_source")
	createdAtValue := rs.getInt64FromPayload(payload, "created_at")
	validationCount := int(rs.getInt64FromPayload(payload, "validation_count"))

	// Validate required fields
	if sourceChunkID == "" || targetChunkID == "" || relationTypeStr == "" {
		return nil, fmt.Errorf("missing required fields in relationship")
	}

	relationType := types.RelationType(relationTypeStr)
	confidenceSource := types.ConfidenceSource(confidenceSourceStr)

	// Build confidence factors
	factors := types.ConfidenceFactors{}
	if userCertainty := rs.getFloat64FromPayload(payload, "user_certainty"); userCertainty != 0 {
		factors.UserCertainty = &userCertainty
	}
	if consistencyScore := rs.getFloat64FromPayload(payload, "consistency_score"); consistencyScore != 0 {
		factors.ConsistencyScore = &consistencyScore
	}
	if corroborationCount := rs.getInt64FromPayload(payload, "corroboration_count"); corroborationCount != 0 {
		count := int(corroborationCount)
		factors.CorroborationCount = &count
	}

	// Build metadata
	metadata := make(map[string]interface{})
	for key, value := range payload {
		if strings.HasPrefix(key, "meta_") {
			metaKey := strings.TrimPrefix(key, "meta_")
			metadata[metaKey] = value.GetStringValue()
		}
	}

	// Parse timestamps
	createdAt := time.Unix(createdAtValue, 0)
	var lastValidated *time.Time
	if lastValidatedValue := rs.getInt64FromPayload(payload, "last_validated"); lastValidatedValue != 0 {
		lv := time.Unix(lastValidatedValue, 0)
		lastValidated = &lv
	}

	relationship := &types.MemoryRelationship{
		ID:                id,
		SourceChunkID:     sourceChunkID,
		TargetChunkID:     targetChunkID,
		RelationType:      relationType,
		Confidence:        confidence,
		ConfidenceSource:  confidenceSource,
		ConfidenceFactors: factors,
		Metadata:          metadata,
		CreatedAt:         createdAt,
		CreatedBy:         rs.getStringFromPayload(payload, "created_by"),
		LastValidated:     lastValidated,
		ValidationCount:   validationCount,
	}

	return relationship, nil
}

// buildRelationshipFilter creates a filter for relationship queries
func (rs *RelationshipStore) buildRelationshipFilter(query types.RelationshipQuery) *qdrant.Filter {
	conditions := make([]*qdrant.Condition, 0)

	// Direction-based filtering
	switch query.Direction {
	case "outgoing":
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key: "source_chunk_id",
					Match: &qdrant.Match{
						MatchValue: &qdrant.Match_Keyword{Keyword: query.ChunkID},
					},
				},
			},
		})
	case "incoming":
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key: "target_chunk_id",
					Match: &qdrant.Match{
						MatchValue: &qdrant.Match_Keyword{Keyword: query.ChunkID},
					},
				},
			},
		})
	case "both":
		// OR condition for both source and target
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Filter{
				Filter: &qdrant.Filter{
					Should: []*qdrant.Condition{
						{
							ConditionOneOf: &qdrant.Condition_Field{
								Field: &qdrant.FieldCondition{
									Key: "source_chunk_id",
									Match: &qdrant.Match{
										MatchValue: &qdrant.Match_Keyword{Keyword: query.ChunkID},
									},
								},
							},
						},
						{
							ConditionOneOf: &qdrant.Condition_Field{
								Field: &qdrant.FieldCondition{
									Key: "target_chunk_id",
									Match: &qdrant.Match{
										MatchValue: &qdrant.Match_Keyword{Keyword: query.ChunkID},
									},
								},
							},
						},
					},
				},
			},
		})
	}

	if len(conditions) == 0 {
		return nil
	}

	return &qdrant.Filter{Must: conditions}
}

// Utility methods
func (rs *RelationshipStore) stringToValue(s string) *qdrant.Value {
	return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: s}}
}

func (rs *RelationshipStore) float64ToValue(f float64) *qdrant.Value {
	return &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: f}}
}

func (rs *RelationshipStore) int64ToValue(i int64) *qdrant.Value {
	return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: i}}
}

func (rs *RelationshipStore) stringToPointID(s string) *qdrant.PointId {
	return &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: s}}
}

func (rs *RelationshipStore) pointIDToString(id *qdrant.PointId) string {
	return id.GetUuid()
}

func (rs *RelationshipStore) getStringFromPayload(payload map[string]*qdrant.Value, key string) string {
	if value, ok := payload[key]; ok {
		return value.GetStringValue()
	}
	return ""
}

func (rs *RelationshipStore) getFloat64FromPayload(payload map[string]*qdrant.Value, key string) float64 {
	if value, ok := payload[key]; ok {
		return value.GetDoubleValue()
	}
	return 0.0
}

func (rs *RelationshipStore) getInt64FromPayload(payload map[string]*qdrant.Value, key string) int64 {
	if value, ok := payload[key]; ok {
		return value.GetIntegerValue()
	}
	return 0
}

func (rs *RelationshipStore) calculateLimit(limit int) *uint32 {
	if limit <= 0 {
		return qdrant.PtrOf(uint32(100)) // Default limit
	}
	// Prevent integer overflow by capping limit
	if limit > math.MaxUint32 {
		return qdrant.PtrOf(uint32(math.MaxUint32))
	}
	return qdrant.PtrOf(uint32(limit))
}

func (rs *RelationshipStore) sortRelationships(relationships []types.RelationshipResult, sortBy, sortOrder string) {
	if sortBy == "" {
		sortBy = "confidence"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	sort.Slice(relationships, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "confidence":
			less = relationships[i].Relationship.Confidence < relationships[j].Relationship.Confidence
		case "created_at":
			less = relationships[i].Relationship.CreatedAt.Before(relationships[j].Relationship.CreatedAt)
		case "validation_count":
			less = relationships[i].Relationship.ValidationCount < relationships[j].Relationship.ValidationCount
		default:
			less = relationships[i].Relationship.Confidence < relationships[j].Relationship.Confidence
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

func (rs *RelationshipStore) determinePathType(chunkIDs []string, relationships []types.RelationshipResult) string {
	if len(relationships) == 0 {
		return "unknown"
	}

	// Check path length characteristics based on chunk count
	pathLength := len(chunkIDs)
	if pathLength <= 1 {
		return "single_node"
	}

	// Analyze relationship types to determine path type
	relationTypes := make(map[types.RelationType]int)
	for _, rel := range relationships {
		relationTypes[rel.Relationship.RelationType]++
	}

	// Enhanced heuristics considering both relationships and path characteristics
	if relationTypes[types.RelationLedTo] > 0 || relationTypes[types.RelationSolvedBy] > 0 {
		if pathLength >= 3 {
			return "complex_problem_to_solution"
		}
		return "problem_to_solution"
	}
	if relationTypes[types.RelationDependsOn] > 0 || relationTypes[types.RelationEnables] > 0 {
		if pathLength >= 4 {
			return "deep_dependency_chain"
		}
		return "dependency_chain"
	}
	if relationTypes[types.RelationFollowsUp] > 0 || relationTypes[types.RelationPrecedes] > 0 {
		if pathLength >= 5 {
			return "long_temporal_sequence"
		}
		return "temporal_sequence"
	}

	// For general paths, classify by length
	if pathLength >= 4 {
		return "complex_general"
	}
	return "general"
}

func (rs *RelationshipStore) calculateCentrality(nodes map[string]*types.GraphNode) {
	totalDegree := 0
	for _, node := range nodes {
		totalDegree += node.Degree
	}

	// Simple degree centrality calculation
	for _, node := range nodes {
		if totalDegree > 0 {
			node.Centrality = float64(node.Degree) / float64(totalDegree)
		}
	}
}

func (rs *RelationshipStore) updateMetrics(operation string, start time.Time) {
	duration := time.Since(start)

	rs.metrics.OperationCounts[operation]++

	// Update average latency
	currentAvg := rs.metrics.AverageLatency[operation]
	count := float64(rs.metrics.OperationCounts[operation])
	newLatency := float64(duration.Milliseconds())
	rs.metrics.AverageLatency[operation] = (currentAvg*(count-1) + newLatency) / count

	rs.metrics.LastOperation = &operation
}
