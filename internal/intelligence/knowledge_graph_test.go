package intelligence

import (
	"context"
	"strings"
	"testing"
	"time"

	"mcp-memory/pkg/types"
)

func TestKnowledgeGraphCreation(t *testing.T) {
	graph := NewKnowledgeGraph()

	if graph == nil {
		t.Fatal("Expected knowledge graph to be created")
	}

	if graph.Nodes == nil {
		t.Error("Expected nodes map to be initialized")
	}

	if graph.Relations == nil {
		t.Error("Expected relations map to be initialized")
	}

	if len(graph.Nodes) != 0 {
		t.Error("Expected empty nodes map initially")
	}

	if len(graph.Relations) != 0 {
		t.Error("Expected empty relations map initially")
	}
}

func TestGraphBuilderCreation(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	builder := NewGraphBuilder(patternEngine)

	if builder == nil {
		t.Fatal("Expected graph builder to be created")
	}

	if builder.graph == nil {
		t.Error("Expected graph to be initialized")
	}

	if builder.patternEngine != patternEngine {
		t.Error("Expected pattern engine to be set")
	}

	if builder.minConfidence != 0.6 {
		t.Errorf("Expected minConfidence to be 0.6, got %f", builder.minConfidence)
	}
}

func TestAddNode(t *testing.T) {
	builder := NewGraphBuilder(nil)

	node := &KnowledgeNode{
		ID:          "test_node",
		Type:        NodeTypeConcept,
		Name:        "Test Concept",
		Description: "A test concept",
		Content:     "test content",
		Properties:  map[string]any{"test": true},
		Tags:        []string{"test"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastUsed:    time.Now(),
		UsageCount:  1,
		Confidence:  0.8,
	}

	err := builder.AddNode(node)
	if err != nil {
		t.Fatalf("Expected no error adding node, got %v", err)
	}

	if len(builder.graph.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(builder.graph.Nodes))
	}

	storedNode, exists := builder.graph.Nodes["test_node"]
	if !exists {
		t.Error("Expected node to be stored")
	}

	if storedNode.Name != "Test Concept" {
		t.Errorf("Expected node name 'Test Concept', got '%s'", storedNode.Name)
	}
}

func TestAddRelation(t *testing.T) {
	builder := NewGraphBuilder(nil)

	// Add two nodes first
	node1 := &KnowledgeNode{
		ID:   "node1",
		Type: NodeTypeConcept,
		Name: "Node 1",
	}
	node2 := &KnowledgeNode{
		ID:   "node2",
		Type: NodeTypeConcept,
		Name: "Node 2",
	}

	err := builder.AddNode(node1)
	if err != nil {
		t.Fatalf("Error adding node1: %v", err)
	}

	err = builder.AddNode(node2)
	if err != nil {
		t.Fatalf("Error adding node2: %v", err)
	}

	// Add relation
	relation := &KnowledgeRelation{
		ID:         "test_relation",
		FromNodeID: "node1",
		ToNodeID:   "node2",
		Type:       RelationSimilarTo,
		Weight:     0.8,
		Confidence: 0.8,
		Properties: map[string]any{"test": true},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err = builder.AddRelation(relation)
	if err != nil {
		t.Fatalf("Expected no error adding relation, got %v", err)
	}

	if len(builder.graph.Relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(builder.graph.Relations))
	}

	storedRelation, exists := builder.graph.Relations["test_relation"]
	if !exists {
		t.Error("Expected relation to be stored")
	}

	if storedRelation.Type != RelationSimilarTo {
		t.Errorf("Expected relation type '%s', got '%s'", RelationSimilarTo, storedRelation.Type)
	}
}

func TestBuildFromChunks(t *testing.T) {
	storage := NewMockPatternStorage()
	patternEngine := NewPatternEngine(storage)
	builder := NewGraphBuilder(patternEngine)

	// Create test chunks
	chunks := []types.ConversationChunk{
		{
			ID:        "chunk1",
			SessionID: "session1",
			Content:   "I have an error in my Go code. The function main() is not working.",
			Type:      types.ChunkTypeProblem,
			Timestamp: time.Now(),
		},
		{
			ID:        "chunk2",
			SessionID: "session1",
			Content:   "You need to check your import statements. Try adding 'import fmt' at the top.",
			Type:      types.ChunkTypeSolution,
			Timestamp: time.Now().Add(1 * time.Minute),
		},
	}

	err := builder.BuildFromChunks(context.Background(), chunks)
	if err != nil {
		t.Fatalf("Expected no error building from chunks, got %v", err)
	}

	// Check that nodes were created
	if len(builder.graph.Nodes) == 0 {
		t.Error("Expected nodes to be created from chunks")
	}

	// Check that at least chunk nodes were created
	chunkNodesFound := 0
	for _, node := range builder.graph.Nodes {
		if node.Type == NodeTypeChunk {
			chunkNodesFound++
		}
	}

	if chunkNodesFound != 2 {
		t.Errorf("Expected 2 chunk nodes, got %d", chunkNodesFound)
	}

	// Check that relations were created
	if len(builder.graph.Relations) == 0 {
		t.Error("Expected relations to be created")
	}
}

func TestConceptExtraction(t *testing.T) {
	extractor := NewBasicConceptExtractor()

	text := "I need to create a REST API using Go and PostgreSQL database for user authentication"

	concepts, err := extractor.ExtractConcepts(text)
	if err != nil {
		t.Fatalf("Expected no error extracting concepts, got %v", err)
	}

	if len(concepts) == 0 {
		t.Error("Expected to extract some concepts")
	}

	// Check for specific technical terms
	foundGo := false
	foundAPI := false
	foundDatabase := false

	for _, concept := range concepts {
		switch strings.ToLower(concept.Name) {
		case "go":
			foundGo = true
		case "api":
			foundAPI = true
		case "database":
			foundDatabase = true
		}
	}

	if !foundGo {
		t.Error("Expected to find 'Go' as a technical term")
	}
	if !foundAPI {
		t.Error("Expected to find 'API' as a technical term")
	}
	if !foundDatabase {
		t.Error("Expected to find 'database' as a technical term")
	}
}

func TestEntityExtraction(t *testing.T) {
	extractor := NewBasicEntityExtractor()

	text := "Check the main.go file and look at the handleRequest() function. Also run 'go build' command."

	// Test file extraction
	files := extractor.ExtractFiles(text)
	if len(files) == 0 {
		t.Error("Expected to extract file references")
	}

	foundMainGo := false
	for _, file := range files {
		if file == "main.go" {
			foundMainGo = true
		}
	}
	if !foundMainGo {
		t.Error("Expected to find 'main.go' file reference")
	}

	// Test function extraction
	functions := extractor.ExtractFunctions(text)
	if len(functions) == 0 {
		t.Error("Expected to extract function references")
	}

	foundHandleRequest := false
	for _, function := range functions {
		if strings.Contains(function, "handleRequest") {
			foundHandleRequest = true
		}
	}
	if !foundHandleRequest {
		t.Error("Expected to find 'handleRequest' function reference")
	}
}

func TestQueryGraph(t *testing.T) {
	builder := NewGraphBuilder(nil)

	// Add test nodes
	conceptNode := &KnowledgeNode{
		ID:          "concept1",
		Type:        NodeTypeConcept,
		Name:        "Go Programming",
		Description: "Programming language Go",
		Content:     "go language programming",
		Confidence:  0.9,
		UsageCount:  5,
	}

	fileNode := &KnowledgeNode{
		ID:          "file1",
		Type:        NodeTypeFile,
		Name:        "main.go",
		Description: "Main Go file",
		Content:     "main.go",
		Confidence:  0.8,
		UsageCount:  3,
	}

	_ = builder.AddNode(conceptNode) // test - ignore error
	_ = builder.AddNode(fileNode)    // test - ignore error

	// Query for concept nodes
	query := GraphQuery{
		NodeTypes:     []NodeType{NodeTypeConcept},
		MinConfidence: 0.5,
		Limit:         10,
	}

	results, err := builder.QueryGraph(&query)
	if err != nil {
		t.Fatalf("Expected no error querying graph, got %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 concept node, got %d", len(results))
	}

	if results[0].Type != NodeTypeConcept {
		t.Errorf("Expected concept node, got %s", results[0].Type)
	}

	// Query with keywords
	keywordQuery := GraphQuery{
		Keywords:      []string{"go"},
		MinConfidence: 0.5,
		Limit:         10,
	}

	keywordResults, err := builder.QueryGraph(&keywordQuery)
	if err != nil {
		t.Fatalf("Expected no error querying with keywords, got %v", err)
	}

	if len(keywordResults) == 0 {
		t.Error("Expected to find nodes matching 'go' keyword")
	}
}

func TestGetStats(t *testing.T) {
	builder := NewGraphBuilder(nil)

	// Add some test data
	node1 := &KnowledgeNode{ID: "node1", Type: NodeTypeConcept, Name: "Concept1"}
	node2 := &KnowledgeNode{ID: "node2", Type: NodeTypeFile, Name: "File1"}

	_ = builder.AddNode(node1) // test - ignore error
	_ = builder.AddNode(node2) // test - ignore error

	relation := &KnowledgeRelation{
		ID:         "rel1",
		FromNodeID: "node1",
		ToNodeID:   "node2",
		Type:       RelationReferences,
	}
	_ = builder.AddRelation(relation) // test - ignore error

	stats := builder.GetStats()

	if stats["total_nodes"] != 2 {
		t.Errorf("Expected 2 total nodes, got %v", stats["total_nodes"])
	}

	if stats["total_relations"] != 1 {
		t.Errorf("Expected 1 total relation, got %v", stats["total_relations"])
	}

	nodeTypes, ok := stats["node_types"].(map[string]int)
	if !ok {
		t.Error("Expected node_types to be map[string]int")
	} else {
		switch {
		case nodeTypes["concept"] != 1:
			t.Errorf("Expected 1 concept node, got %d", nodeTypes["concept"])
		case nodeTypes["file"] != 1:
			t.Errorf("Expected 1 file node, got %d", nodeTypes["file"])
		}
	}
}
