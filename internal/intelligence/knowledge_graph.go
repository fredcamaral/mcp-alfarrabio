package intelligence

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"mcp-memory/pkg/types"
)

// NodeType represents different types of nodes in the knowledge graph
type NodeType string

const (
	NodeTypeChunk      NodeType = "chunk"
	NodeTypeConcept    NodeType = "concept"
	NodeTypeFile       NodeType = "file"
	NodeTypeFunction   NodeType = "function"
	NodeTypePattern    NodeType = "pattern"
	NodeTypeProblem    NodeType = "problem"
	NodeTypeSolution   NodeType = "solution"
	NodeTypeDecision   NodeType = "decision"
	NodeTypeRepository NodeType = "repository"
	NodeTypeWorkflow   NodeType = "workflow"
)

// RelationType represents different types of relationships
type RelationType string

const (
	RelationFollows       RelationType = "follows"
	RelationSolves        RelationType = "solves"
	RelationReferences    RelationType = "references"
	RelationSimilarTo     RelationType = "similar_to"
	RelationCauses        RelationType = "causes"
	RelationDependsOn     RelationType = "depends_on"
	RelationImplements    RelationType = "implements"
	RelationModifies      RelationType = "modifies"
	RelationUsedWith      RelationType = "used_with"
	RelationContains      RelationType = "contains"
	RelationEvolvesFrom   RelationType = "evolves_from"
	RelationConflictsWith RelationType = "conflicts_with"
)

// KnowledgeNode represents a node in the knowledge graph
type KnowledgeNode struct {
	ID          string         `json:"id"`
	Type        NodeType       `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Content     string         `json:"content"`
	Properties  map[string]any `json:"properties"`
	Tags        []string       `json:"tags"`
	ChunkID     *string        `json:"chunk_id,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	LastUsed    time.Time      `json:"last_used"`
	UsageCount  int            `json:"usage_count"`
	Confidence  float64        `json:"confidence"`
}

// KnowledgeRelation represents a relationship between nodes
type KnowledgeRelation struct {
	ID         string         `json:"id"`
	FromNodeID string         `json:"from_node_id"`
	ToNodeID   string         `json:"to_node_id"`
	Type       RelationType   `json:"type"`
	Weight     float64        `json:"weight"`
	Confidence float64        `json:"confidence"`
	Properties map[string]any `json:"properties"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Evidence   []string       `json:"evidence"`
}

// KnowledgeGraph represents the main knowledge graph structure
type KnowledgeGraph struct {
	Nodes     map[string]*KnowledgeNode     `json:"nodes"`
	Relations map[string]*KnowledgeRelation `json:"relations"`
	NodeIndex map[NodeType][]string         `json:"node_index"`
	CreatedAt time.Time                     `json:"created_at"`
	UpdatedAt time.Time                     `json:"updated_at"`
}

// GraphBuilder builds and maintains the knowledge graph
type GraphBuilder struct {
	graph         *KnowledgeGraph
	patternEngine *PatternEngine

	// Configuration
	minConfidence     float64
	maxNodes          int
	relationThreshold float64

	// Entity extractors
	conceptExtractor ConceptExtractor
	entityExtractor  EntityExtractor
}

// ConceptExtractor interface for extracting concepts from text
type ConceptExtractor interface {
	ExtractConcepts(text string) ([]Concept, error)
	IdentifyTechnicalTerms(text string) []string
	ExtractKeyPhrases(text string) []string
}

// EntityExtractor interface for extracting entities
type EntityExtractor interface {
	ExtractFiles(text string) []string
	ExtractFunctions(text string) []string
	ExtractVariables(text string) []string
	ExtractCommands(text string) []string
}

// Concept represents an extracted concept
type Concept struct {
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Confidence  float64        `json:"confidence"`
	Context     map[string]any `json:"context"`
}

// GraphQuery represents a query for the knowledge graph
type GraphQuery struct {
	NodeTypes     []NodeType     `json:"node_types,omitempty"`
	RelationTypes []RelationType `json:"relation_types,omitempty"`
	Keywords      []string       `json:"keywords,omitempty"`
	MinConfidence float64        `json:"min_confidence"`
	MaxDepth      int            `json:"max_depth"`
	Limit         int            `json:"limit"`
}

// NewKnowledgeGraph creates a new knowledge graph
func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{
		Nodes:     make(map[string]*KnowledgeNode),
		Relations: make(map[string]*KnowledgeRelation),
		NodeIndex: make(map[NodeType][]string),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// NewGraphBuilder creates a new graph builder
func NewGraphBuilder(patternEngine *PatternEngine) *GraphBuilder {
	return &GraphBuilder{
		graph:             NewKnowledgeGraph(),
		patternEngine:     patternEngine,
		minConfidence:     0.6,
		maxNodes:          10000,
		relationThreshold: 0.5,
		conceptExtractor:  NewBasicConceptExtractor(),
		entityExtractor:   NewBasicEntityExtractor(),
	}
}

// BuildFromChunks builds knowledge graph from conversation chunks
func (gb *GraphBuilder) BuildFromChunks(ctx context.Context, chunks []types.ConversationChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	// First pass: create nodes for chunks and extract entities
	for _, chunk := range chunks {
		err := gb.addChunkNode(chunk)
		if err != nil {
			return fmt.Errorf("failed to add chunk node: %w", err)
		}

		// Extract and add concept nodes
		err = gb.extractAndAddConcepts(chunk)
		if err != nil {
			return fmt.Errorf("failed to extract concepts: %w", err)
		}

		// Extract and add entity nodes
		err = gb.extractAndAddEntities(chunk)
		if err != nil {
			return fmt.Errorf("failed to extract entities: %w", err)
		}
	}

	// Second pass: identify and create relations
	err := gb.identifyRelations(chunks)
	if err != nil {
		return fmt.Errorf("failed to identify relations: %w", err)
	}

	// Third pass: infer additional relationships
	err = gb.inferRelationships()
	if err != nil {
		return fmt.Errorf("failed to infer relationships: %w", err)
	}

	gb.graph.UpdatedAt = time.Now()
	return nil
}

// AddNode adds a node to the knowledge graph
func (gb *GraphBuilder) AddNode(node *KnowledgeNode) error {
	if node.ID == "" {
		return errors.New("node ID cannot be empty")
	}

	if len(gb.graph.Nodes) >= gb.maxNodes {
		return errors.New("maximum number of nodes reached")
	}

	gb.graph.Nodes[node.ID] = node
	gb.graph.NodeIndex[node.Type] = append(gb.graph.NodeIndex[node.Type], node.ID)
	gb.graph.UpdatedAt = time.Now()

	return nil
}

// AddRelation adds a relation to the knowledge graph
func (gb *GraphBuilder) AddRelation(relation *KnowledgeRelation) error {
	if relation.ID == "" {
		return errors.New("relation ID cannot be empty")
	}

	// Verify nodes exist
	if _, exists := gb.graph.Nodes[relation.FromNodeID]; !exists {
		return fmt.Errorf("from node %s does not exist", relation.FromNodeID)
	}
	if _, exists := gb.graph.Nodes[relation.ToNodeID]; !exists {
		return fmt.Errorf("to node %s does not exist", relation.ToNodeID)
	}

	gb.graph.Relations[relation.ID] = relation
	gb.graph.UpdatedAt = time.Now()

	return nil
}

// QueryGraph queries the knowledge graph
func (gb *GraphBuilder) QueryGraph(query GraphQuery) ([]*KnowledgeNode, error) {
	var results []*KnowledgeNode

	// Filter nodes by type
	candidateNodes := make([]*KnowledgeNode, 0)

	if len(query.NodeTypes) == 0 {
		// Include all nodes
		for _, node := range gb.graph.Nodes {
			candidateNodes = append(candidateNodes, node)
		}
	} else {
		// Include only specified types
		for _, nodeType := range query.NodeTypes {
			if nodeIDs, exists := gb.graph.NodeIndex[nodeType]; exists {
				for _, nodeID := range nodeIDs {
					if node, exists := gb.graph.Nodes[nodeID]; exists {
						candidateNodes = append(candidateNodes, node)
					}
				}
			}
		}
	}

	// Filter by confidence
	for _, node := range candidateNodes {
		if node.Confidence >= query.MinConfidence {
			// Check keyword match
			if len(query.Keywords) == 0 || gb.nodeMatchesKeywords(node, query.Keywords) {
				results = append(results, node)
			}
		}
	}

	// Sort by relevance (confidence * usage)
	sort.Slice(results, func(i, j int) bool {
		scoreI := results[i].Confidence * float64(results[i].UsageCount+1)
		scoreJ := results[j].Confidence * float64(results[j].UsageCount+1)
		return scoreI > scoreJ
	})

	// Apply limit
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	return results, nil
}

// GetRelatedNodes finds nodes related to a given node
func (gb *GraphBuilder) GetRelatedNodes(nodeID string, maxDepth int) ([]*KnowledgeNode, error) {
	if maxDepth <= 0 {
		return nil, errors.New("maxDepth must be positive")
	}

	visited := make(map[string]bool)
	var result []*KnowledgeNode

	gb.traverseRelations(nodeID, maxDepth, 0, visited, &result)

	return result, nil
}

// Helper methods

func (gb *GraphBuilder) addChunkNode(chunk types.ConversationChunk) error {
	node := &KnowledgeNode{
		ID:          fmt.Sprintf("chunk_%s", chunk.ID),
		Type:        NodeTypeChunk,
		Name:        fmt.Sprintf("Chunk %s", chunk.ID),
		Description: gb.generateChunkDescription(chunk),
		Content:     chunk.Content,
		Properties: map[string]any{
			"chunk_type": string(chunk.Type),
			"session_id": chunk.SessionID,
			"timestamp":  chunk.Timestamp,
			"has_code":   strings.Contains(chunk.Content, "```"),
			"word_count": len(strings.Fields(chunk.Content)),
		},
		Tags:       gb.extractTags(chunk),
		ChunkID:    &chunk.ID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		LastUsed:   time.Now(),
		UsageCount: 1,
		Confidence: 1.0,
	}

	return gb.AddNode(node)
}

func (gb *GraphBuilder) extractAndAddConcepts(chunk types.ConversationChunk) error {
	concepts, err := gb.conceptExtractor.ExtractConcepts(chunk.Content)
	if err != nil {
		return err
	}

	for _, concept := range concepts {
		if err := gb.processConcept(concept, chunk); err != nil {
			return err
		}
	}

	return nil
}

func (gb *GraphBuilder) processConcept(concept Concept, chunk types.ConversationChunk) error {
	if concept.Confidence < gb.minConfidence {
		return nil
	}

	// Check if concept already exists
	existingNode := gb.findNodeByName(concept.Name, NodeTypeConcept)
	if existingNode != nil {
		return gb.updateExistingConcept(existingNode, concept)
	}

	// Create new concept node
	return gb.createConceptNode(concept, chunk)
}

func (gb *GraphBuilder) updateExistingConcept(node *KnowledgeNode, concept Concept) error {
	node.UsageCount++
	node.LastUsed = time.Now()
	node.Confidence = math.Max(node.Confidence, concept.Confidence)
	return nil
}

func (gb *GraphBuilder) createConceptNode(concept Concept, chunk types.ConversationChunk) error {
	node := &KnowledgeNode{
		ID:          fmt.Sprintf("concept_%s", gb.generateNodeID(concept.Name)),
		Type:        NodeTypeConcept,
		Name:        concept.Name,
		Description: concept.Description,
		Content:     concept.Name,
		Properties:  concept.Context,
		Tags:        []string{concept.Type},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastUsed:    time.Now(),
		UsageCount:  1,
		Confidence:  concept.Confidence,
	}

	if err := gb.AddNode(node); err != nil {
		return err
	}

	// Add relation from chunk to concept
	return gb.addConceptRelation(chunk, node, concept)
}

func (gb *GraphBuilder) addConceptRelation(chunk types.ConversationChunk, node *KnowledgeNode, concept Concept) error {
	relation := &KnowledgeRelation{
		ID:         fmt.Sprintf("rel_%s_%s", chunk.ID, node.ID),
		FromNodeID: fmt.Sprintf("chunk_%s", chunk.ID),
		ToNodeID:   node.ID,
		Type:       RelationContains,
		Weight:     concept.Confidence,
		Confidence: concept.Confidence,
		Properties: map[string]any{
			"extraction_method": "concept_extractor",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return gb.AddRelation(relation)
}

func (gb *GraphBuilder) extractAndAddEntities(chunk types.ConversationChunk) error {
	// Extract files
	files := gb.entityExtractor.ExtractFiles(chunk.Content)
	for _, file := range files {
		err := gb.addEntityNode(file, NodeTypeFile, chunk.ID)
		if err != nil {
			return err
		}
	}

	// Extract functions
	functions := gb.entityExtractor.ExtractFunctions(chunk.Content)
	for _, function := range functions {
		err := gb.addEntityNode(function, NodeTypeFunction, chunk.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (gb *GraphBuilder) addEntityNode(entityName string, nodeType NodeType, chunkID string) error {
	nodeID := fmt.Sprintf("%s_%s", nodeType, gb.generateNodeID(entityName))

	// Check if entity already exists
	if existingNode := gb.findNodeByName(entityName, nodeType); existingNode != nil {
		existingNode.UsageCount++
		existingNode.LastUsed = time.Now()
		return nil
	}

	node := &KnowledgeNode{
		ID:          nodeID,
		Type:        nodeType,
		Name:        entityName,
		Description: fmt.Sprintf("%s entity: %s", nodeType, entityName),
		Content:     entityName,
		Properties: map[string]any{
			"entity_type": string(nodeType),
		},
		Tags:       []string{string(nodeType)},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		LastUsed:   time.Now(),
		UsageCount: 1,
		Confidence: 0.8,
	}

	err := gb.AddNode(node)
	if err != nil {
		return err
	}

	// Add relation from chunk to entity
	relation := &KnowledgeRelation{
		ID:         fmt.Sprintf("rel_%s_%s", chunkID, nodeID),
		FromNodeID: fmt.Sprintf("chunk_%s", chunkID),
		ToNodeID:   nodeID,
		Type:       RelationReferences,
		Weight:     0.8,
		Confidence: 0.8,
		Properties: map[string]any{
			"extraction_method": "entity_extractor",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return gb.AddRelation(relation)
}

func (gb *GraphBuilder) identifyRelations(chunks []types.ConversationChunk) error {
	// Identify temporal relationships (follows)
	for i := 0; i < len(chunks)-1; i++ {
		relation := &KnowledgeRelation{
			ID:         fmt.Sprintf("follows_%s_%s", chunks[i].ID, chunks[i+1].ID),
			FromNodeID: fmt.Sprintf("chunk_%s", chunks[i].ID),
			ToNodeID:   fmt.Sprintf("chunk_%s", chunks[i+1].ID),
			Type:       RelationFollows,
			Weight:     1.0,
			Confidence: 1.0,
			Properties: map[string]any{
				"temporal_order": i,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := gb.AddRelation(relation)
		if err != nil {
			return err
		}
	}

	// Identify problem-solution relationships
	for i, chunk := range chunks {
		if chunk.Type == types.ChunkTypeProblem {
			// Look for solutions in subsequent chunks
			for j := i + 1; j < len(chunks) && j < i+5; j++ {
				if chunks[j].Type == types.ChunkTypeSolution {
					relation := &KnowledgeRelation{
						ID:         fmt.Sprintf("solves_%s_%s", chunks[j].ID, chunk.ID),
						FromNodeID: fmt.Sprintf("chunk_%s", chunks[j].ID),
						ToNodeID:   fmt.Sprintf("chunk_%s", chunk.ID),
						Type:       RelationSolves,
						Weight:     gb.calculateSolutionRelevance(chunk, chunks[j]),
						Confidence: 0.8,
						Properties: map[string]any{
							"distance": j - i,
						},
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}

					err := gb.AddRelation(relation)
					if err != nil {
						return err
					}
					break // Only link to first solution found
				}
			}
		}
	}

	return nil
}

func (gb *GraphBuilder) inferRelationships() error {
	// Infer similarity relationships based on content
	nodes := make([]*KnowledgeNode, 0, len(gb.graph.Nodes))
	for _, node := range gb.graph.Nodes {
		nodes = append(nodes, node)
	}

	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			similarity := gb.calculateNodeSimilarity(nodes[i], nodes[j])
			if similarity > gb.relationThreshold {
				relation := &KnowledgeRelation{
					ID:         fmt.Sprintf("similar_%s_%s", nodes[i].ID, nodes[j].ID),
					FromNodeID: nodes[i].ID,
					ToNodeID:   nodes[j].ID,
					Type:       RelationSimilarTo,
					Weight:     similarity,
					Confidence: similarity,
					Properties: map[string]any{
						"similarity_score": similarity,
						"inferred":         true,
					},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}

				err := gb.AddRelation(relation)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (gb *GraphBuilder) traverseRelations(nodeID string, maxDepth, currentDepth int, visited map[string]bool, result *[]*KnowledgeNode) {
	if currentDepth >= maxDepth || visited[nodeID] {
		return
	}

	visited[nodeID] = true

	if node, exists := gb.graph.Nodes[nodeID]; exists {
		*result = append(*result, node)
	}

	// Follow outgoing relations
	for _, relation := range gb.graph.Relations {
		if relation.FromNodeID == nodeID {
			gb.traverseRelations(relation.ToNodeID, maxDepth, currentDepth+1, visited, result)
		}
		if relation.ToNodeID == nodeID {
			gb.traverseRelations(relation.FromNodeID, maxDepth, currentDepth+1, visited, result)
		}
	}
}

// Utility methods

func (gb *GraphBuilder) generateNodeID(name string) string {
	// Simple hash-based ID generation
	return fmt.Sprintf("%x", strings.ToLower(strings.ReplaceAll(name, " ", "_")))
}

func (gb *GraphBuilder) generateChunkDescription(chunk types.ConversationChunk) string {
	summary := chunk.Summary
	if summary == "" {
		words := strings.Fields(chunk.Content)
		if len(words) > 20 {
			summary = strings.Join(words[:20], " ") + "..."
		} else {
			summary = chunk.Content
		}
	}
	return fmt.Sprintf("%s chunk: %s", chunk.Type, summary)
}

func (gb *GraphBuilder) extractTags(chunk types.ConversationChunk) []string {
	var tags []string

	tags = append(tags, string(chunk.Type))

	if strings.Contains(chunk.Content, "```") {
		tags = append(tags, "code")
	}
	if strings.Contains(strings.ToLower(chunk.Content), "error") {
		tags = append(tags, "error")
	}
	if strings.Contains(strings.ToLower(chunk.Content), "fix") {
		tags = append(tags, "fix")
	}

	return tags
}

func (gb *GraphBuilder) findNodeByName(name string, nodeType NodeType) *KnowledgeNode {
	for _, node := range gb.graph.Nodes {
		if node.Type == nodeType && strings.EqualFold(node.Name, name) {
			return node
		}
	}
	return nil
}

func (gb *GraphBuilder) nodeMatchesKeywords(node *KnowledgeNode, keywords []string) bool {
	nodeText := strings.ToLower(node.Name + " " + node.Description + " " + node.Content)

	for _, keyword := range keywords {
		if strings.Contains(nodeText, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

func (gb *GraphBuilder) calculateSolutionRelevance(problem, solution types.ConversationChunk) float64 {
	// Simple keyword overlap calculation
	problemWords := strings.Fields(strings.ToLower(problem.Content))
	solutionWords := strings.Fields(strings.ToLower(solution.Content))

	overlap := 0
	for _, pw := range problemWords {
		for _, sw := range solutionWords {
			if pw == sw && len(pw) > 3 {
				overlap++
			}
		}
	}

	maxWords := math.Max(float64(len(problemWords)), float64(len(solutionWords)))
	return float64(overlap) / maxWords
}

func (gb *GraphBuilder) calculateNodeSimilarity(node1, node2 *KnowledgeNode) float64 {
	if node1.Type != node2.Type {
		return 0.0
	}

	// Calculate content similarity
	text1 := strings.ToLower(node1.Name + " " + node1.Content)
	text2 := strings.ToLower(node2.Name + " " + node2.Content)

	words1 := strings.Fields(text1)
	words2 := strings.Fields(text2)

	overlap := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 && len(w1) > 3 {
				overlap++
			}
		}
	}

	maxWords := math.Max(float64(len(words1)), float64(len(words2)))
	if maxWords == 0 {
		return 0.0
	}

	return float64(overlap) / maxWords
}

// GetGraph returns the current knowledge graph
func (gb *GraphBuilder) GetGraph() *KnowledgeGraph {
	return gb.graph
}

// GetStats returns statistics about the knowledge graph
func (gb *GraphBuilder) GetStats() map[string]any {
	stats := make(map[string]any)

	stats["total_nodes"] = len(gb.graph.Nodes)
	stats["total_relations"] = len(gb.graph.Relations)

	// Node type distribution
	nodeTypes := make(map[string]int)
	for _, node := range gb.graph.Nodes {
		nodeTypes[string(node.Type)]++
	}
	stats["node_types"] = nodeTypes

	// Relation type distribution
	relationTypes := make(map[string]int)
	for _, relation := range gb.graph.Relations {
		relationTypes[string(relation.Type)]++
	}
	stats["relation_types"] = relationTypes

	stats["created_at"] = gb.graph.CreatedAt
	stats["updated_at"] = gb.graph.UpdatedAt

	return stats
}
