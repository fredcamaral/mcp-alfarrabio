// Package tasks provides dependency detection and graph generation for tasks.
package tasks

import (
	"sort"
	"strings"

	"lerian-mcp-memory/pkg/types"
)

// DependencyDetector detects dependencies between tasks
type DependencyDetector struct {
	config DependencyConfig
}

// DependencyConfig represents configuration for dependency detection
type DependencyConfig struct {
	MinSimilarityThreshold float64             `json:"min_similarity_threshold"`
	DependencyPatterns     []DependencyPattern `json:"dependency_patterns"`
	EnableSemanticAnalysis bool                `json:"enable_semantic_analysis"`
	MaxDependencyDistance  int                 `json:"max_dependency_distance"`
}

// DependencyPattern represents a pattern for detecting dependencies
type DependencyPattern struct {
	Name         string               `json:"name"`
	FromKeywords []string             `json:"from_keywords"`
	ToKeywords   []string             `json:"to_keywords"`
	Type         types.DependencyType `json:"type"`
	Strength     float64              `json:"strength"`
	Description  string               `json:"description"`
}

// DefaultDependencyConfig returns default dependency configuration
func DefaultDependencyConfig() DependencyConfig {
	return DependencyConfig{
		MinSimilarityThreshold: 0.3,
		EnableSemanticAnalysis: true,
		MaxDependencyDistance:  3,
		DependencyPatterns: []DependencyPattern{
			{
				Name:         "design_before_implementation",
				FromKeywords: []string{"design", "mockup", "wireframe", "prototype", "architecture"},
				ToKeywords:   []string{"implement", "develop", "code", "build"},
				Type:         types.DependencyTypeBlocking,
				Strength:     0.9,
				Description:  "Design tasks must be completed before implementation",
			},
			{
				Name:         "api_before_frontend",
				FromKeywords: []string{"api", "backend", "service", "endpoint"},
				ToKeywords:   []string{"frontend", "ui", "client", "web app"},
				Type:         types.DependencyTypeBlocking,
				Strength:     0.8,
				Description:  "API development should precede frontend implementation",
			},
			{
				Name:         "database_before_backend",
				FromKeywords: []string{"database", "schema", "migration", "model"},
				ToKeywords:   []string{"backend", "api", "service", "business logic"},
				Type:         types.DependencyTypeBlocking,
				Strength:     0.85,
				Description:  "Database setup should precede backend development",
			},
			{
				Name:         "implementation_before_testing",
				FromKeywords: []string{"implement", "develop", "code", "build"},
				ToKeywords:   []string{"test", "qa", "validation", "verify"},
				Type:         types.DependencyTypeBlocking,
				Strength:     0.9,
				Description:  "Implementation must be completed before testing",
			},
			{
				Name:         "authentication_early",
				FromKeywords: []string{"authentication", "auth", "login", "security"},
				ToKeywords:   []string{"feature", "functionality", "business logic"},
				Type:         types.DependencyTypePreferred,
				Strength:     0.7,
				Description:  "Authentication should be implemented early",
			},
			{
				Name:         "setup_before_development",
				FromKeywords: []string{"setup", "configuration", "environment", "infrastructure"},
				ToKeywords:   []string{"develop", "implement", "code", "build"},
				Type:         types.DependencyTypeBlocking,
				Strength:     0.85,
				Description:  "Setup and configuration should precede development",
			},
			{
				Name:         "documentation_after_implementation",
				FromKeywords: []string{"implement", "develop", "code", "build"},
				ToKeywords:   []string{"document", "documentation", "readme", "guide"},
				Type:         types.DependencyTypePreferred,
				Strength:     0.6,
				Description:  "Documentation should follow implementation",
			},
			{
				Name:         "deployment_after_testing",
				FromKeywords: []string{"test", "qa", "validation", "verify"},
				ToKeywords:   []string{"deploy", "deployment", "release", "production"},
				Type:         types.DependencyTypeBlocking,
				Strength:     0.9,
				Description:  "Deployment should follow successful testing",
			},
		},
	}
}

// NewDependencyDetector creates a new dependency detector
func NewDependencyDetector() *DependencyDetector {
	return &DependencyDetector{
		config: DefaultDependencyConfig(),
	}
}

// NewDependencyDetectorWithConfig creates a new dependency detector with custom config
func NewDependencyDetectorWithConfig(config DependencyConfig) *DependencyDetector {
	return &DependencyDetector{
		config: config,
	}
}

// GenerateDependencyGraph generates a dependency graph for the given tasks
func (dd *DependencyDetector) GenerateDependencyGraph(tasks []types.Task) types.DependencyGraph {
	if len(tasks) == 0 {
		return types.DependencyGraph{Nodes: []types.DependencyNode{}, Edges: []types.DependencyEdge{}}
	}

	// Create nodes
	nodes := make([]types.DependencyNode, 0, len(tasks))
	for _, task := range tasks {
		nodes = append(nodes, types.DependencyNode{
			TaskID:     task.ID,
			Title:      task.Title,
			Type:       task.Type,
			Priority:   task.Priority,
			Complexity: task.Complexity.Level,
		})
	}

	// Detect dependencies
	edges := dd.detectDependencies(tasks)

	// Sort edges by strength (strongest first)
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].Strength > edges[j].Strength
	})

	return types.DependencyGraph{
		Nodes: nodes,
		Edges: edges,
	}
}

// detectDependencies detects dependencies between tasks
func (dd *DependencyDetector) detectDependencies(tasks []types.Task) []types.DependencyEdge {
	edges := make([]types.DependencyEdge, 0)

	// Check explicit dependencies first
	for _, task := range tasks {
		for _, depID := range task.Dependencies {
			if dd.taskExists(tasks, depID) {
				edges = append(edges, types.DependencyEdge{
					FromTaskID:  depID,
					ToTaskID:    task.ID,
					Type:        types.DependencyTypeBlocking,
					Strength:    1.0,
					Description: "Explicit dependency",
				})
			}
		}
	}

	// Detect implicit dependencies using patterns
	for i, fromTask := range tasks {
		for j, toTask := range tasks {
			if i == j {
				continue // Skip self-dependencies
			}

			// Check if there's already an explicit dependency
			if dd.hasExplicitDependency(fromTask, toTask) {
				continue
			}

			// Apply dependency patterns
			for _, pattern := range dd.config.DependencyPatterns {
				if dd.matchesPattern(fromTask, toTask, pattern) {
					edge := types.DependencyEdge{
						FromTaskID:  fromTask.ID,
						ToTaskID:    toTask.ID,
						Type:        pattern.Type,
						Strength:    pattern.Strength,
						Description: pattern.Description,
					}

					// Adjust strength based on content similarity
					similarity := dd.calculateContentSimilarity(fromTask, toTask)
					edge.Strength *= (0.5 + 0.5*similarity) // Boost strength for similar tasks

					// Only add if strength meets threshold
					if edge.Strength >= dd.config.MinSimilarityThreshold {
						edges = append(edges, edge)
					}
				}
			}
		}
	}

	// Detect conflicting tasks
	for i, task1 := range tasks {
		for j, task2 := range tasks {
			if i >= j {
				continue // Avoid duplicates and self-comparison
			}

			if dd.areTasksConflicting(task1, task2) {
				edges = append(edges, types.DependencyEdge{
					FromTaskID:  task1.ID,
					ToTaskID:    task2.ID,
					Type:        types.DependencyTypeConflicting,
					Strength:    0.8,
					Description: "Tasks may conflict with each other",
				})
			}
		}
	}

	// Detect related tasks
	for i, task1 := range tasks {
		for j, task2 := range tasks {
			if i >= j {
				continue
			}

			similarity := dd.calculateContentSimilarity(task1, task2)
			if similarity > 0.6 && !dd.hasAnyDependency(edges, task1.ID, task2.ID) {
				edges = append(edges, types.DependencyEdge{
					FromTaskID:  task1.ID,
					ToTaskID:    task2.ID,
					Type:        types.DependencyTypeRelated,
					Strength:    similarity,
					Description: "Tasks are related in content or scope",
				})
			}
		}
	}

	return dd.removeCycles(edges)
}

// matchesPattern checks if two tasks match a dependency pattern
func (dd *DependencyDetector) matchesPattern(fromTask, toTask types.Task, pattern DependencyPattern) bool {
	fromContent := strings.ToLower(fromTask.Title + " " + fromTask.Description)
	toContent := strings.ToLower(toTask.Title + " " + toTask.Description)

	// Check if fromTask matches fromKeywords
	fromMatches := false
	for _, keyword := range pattern.FromKeywords {
		if strings.Contains(fromContent, keyword) {
			fromMatches = true
			break
		}
	}

	if !fromMatches {
		return false
	}

	// Check if toTask matches toKeywords
	toMatches := false
	for _, keyword := range pattern.ToKeywords {
		if strings.Contains(toContent, keyword) {
			toMatches = true
			break
		}
	}

	return toMatches
}

// calculateContentSimilarity calculates similarity between two tasks
func (dd *DependencyDetector) calculateContentSimilarity(task1, task2 types.Task) float64 {
	// Simple keyword-based similarity
	content1 := strings.ToLower(task1.Title + " " + task1.Description)
	content2 := strings.ToLower(task2.Title + " " + task2.Description)

	words1 := strings.Fields(content1)
	words2 := strings.Fields(content2)

	// Create word frequency maps
	freq1 := make(map[string]int)
	freq2 := make(map[string]int)

	for _, word := range words1 {
		if len(word) > 3 { // Skip short words
			freq1[word]++
		}
	}

	for _, word := range words2 {
		if len(word) > 3 {
			freq2[word]++
		}
	}

	// Calculate Jaccard similarity
	intersection := 0
	union := len(freq1)

	for word := range freq2 {
		if _, exists := freq1[word]; exists {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	similarity := float64(intersection) / float64(union)

	// Boost similarity for same task type
	if task1.Type == task2.Type {
		similarity += 0.1
	}

	// Boost similarity for same priority
	if task1.Priority == task2.Priority {
		similarity += 0.05
	}

	// Boost similarity for overlapping tags
	commonTags := dd.countCommonTags(task1.Tags, task2.Tags)
	if commonTags > 0 {
		similarity += float64(commonTags) * 0.05
	}

	return similarity
}

// areTasksConflicting determines if two tasks are conflicting
func (dd *DependencyDetector) areTasksConflicting(task1, task2 types.Task) bool {
	// Tasks with same file modifications might conflict
	if len(task1.Metadata.ExtendedData) > 0 && len(task2.Metadata.ExtendedData) > 0 {
		files1, ok1 := task1.Metadata.ExtendedData["files"].([]string)
		files2, ok2 := task2.Metadata.ExtendedData["files"].([]string)

		if ok1 && ok2 {
			commonFiles := dd.countCommonStrings(files1, files2)
			if commonFiles > 0 {
				return true
			}
		}
	}

	// Conflicting keywords
	conflictPatterns := [][]string{
		{"refactor", "new feature"}, // Refactoring might conflict with new features
		{"remove", "add"},           // Removing and adding similar functionality
		{"migrate", "upgrade"},      // Different approaches to updating
		{"redesign", "enhance"},     // Different levels of changes
		{"replace", "extend"},       // Different modification strategies
	}

	content1 := strings.ToLower(task1.Title + " " + task1.Description)
	content2 := strings.ToLower(task2.Title + " " + task2.Description)

	for _, pattern := range conflictPatterns {
		if strings.Contains(content1, pattern[0]) && strings.Contains(content2, pattern[1]) {
			return true
		}
		if strings.Contains(content1, pattern[1]) && strings.Contains(content2, pattern[0]) {
			return true
		}
	}

	return false
}

// hasExplicitDependency checks if there's an explicit dependency between tasks
func (dd *DependencyDetector) hasExplicitDependency(fromTask, toTask types.Task) bool {
	for _, depID := range toTask.Dependencies {
		if depID == fromTask.ID {
			return true
		}
	}
	for _, depID := range fromTask.Dependencies {
		if depID == toTask.ID {
			return true
		}
	}
	return false
}

// hasAnyDependency checks if there's any dependency between two tasks in the edges
func (dd *DependencyDetector) hasAnyDependency(edges []types.DependencyEdge, task1ID, task2ID string) bool {
	for _, edge := range edges {
		if (edge.FromTaskID == task1ID && edge.ToTaskID == task2ID) ||
			(edge.FromTaskID == task2ID && edge.ToTaskID == task1ID) {
			return true
		}
	}
	return false
}

// taskExists checks if a task with the given ID exists
func (dd *DependencyDetector) taskExists(tasks []types.Task, taskID string) bool {
	for _, task := range tasks {
		if task.ID == taskID {
			return true
		}
	}
	return false
}

// countCommonTags counts common tags between two task tag lists
func (dd *DependencyDetector) countCommonTags(tags1, tags2 []string) int {
	tagMap := make(map[string]bool)
	for _, tag := range tags1 {
		tagMap[tag] = true
	}

	count := 0
	for _, tag := range tags2 {
		if tagMap[tag] {
			count++
		}
	}
	return count
}

// countCommonStrings counts common strings between two slices
func (dd *DependencyDetector) countCommonStrings(slice1, slice2 []string) int {
	stringMap := make(map[string]bool)
	for _, str := range slice1 {
		stringMap[str] = true
	}

	count := 0
	for _, str := range slice2 {
		if stringMap[str] {
			count++
		}
	}
	return count
}

// removeCycles removes cyclic dependencies to create a DAG
func (dd *DependencyDetector) removeCycles(edges []types.DependencyEdge) []types.DependencyEdge {
	// Build adjacency list
	graph := make(map[string][]string)
	edgeMap := make(map[string]types.DependencyEdge)

	for _, edge := range edges {
		key := edge.FromTaskID + "->" + edge.ToTaskID
		graph[edge.FromTaskID] = append(graph[edge.FromTaskID], edge.ToTaskID)
		edgeMap[key] = edge
	}

	// Find strongly connected components using DFS
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	result := []types.DependencyEdge{}

	var dfs func(string) bool
	dfs = func(node string) bool {
		if inStack[node] {
			return true // Cycle detected
		}
		if visited[node] {
			return false
		}

		visited[node] = true
		inStack[node] = true

		for _, neighbor := range graph[node] {
			if dfs(neighbor) {
				// Remove this edge to break the cycle
				key := node + "->" + neighbor
				delete(edgeMap, key)
			}
		}

		inStack[node] = false
		return false
	}

	// Run DFS from all unvisited nodes
	for node := range graph {
		if !visited[node] {
			dfs(node)
		}
	}

	// Rebuild edges list without cycles
	for _, edge := range edgeMap {
		result = append(result, edge)
	}

	return result
}

// DetectTaskDependencies detects dependencies for a single task against existing tasks
func (dd *DependencyDetector) DetectTaskDependencies(task *types.Task, existingTasks []types.Task) []string {
	dependencies := []string{}

	for _, existingTask := range existingTasks {
		// Apply dependency patterns
		for _, pattern := range dd.config.DependencyPatterns {
			if pattern.Type == types.DependencyTypeBlocking &&
				dd.matchesPattern(existingTask, *task, pattern) {
				// Check if dependency already exists
				found := false
				for _, dep := range dependencies {
					if dep == existingTask.ID {
						found = true
						break
					}
				}
				if !found {
					dependencies = append(dependencies, existingTask.ID)
				}
			}
		}
	}

	return dependencies
}

// GetDependencyStrength calculates the strength of dependency between two tasks
func (dd *DependencyDetector) GetDependencyStrength(fromTask, toTask types.Task) float64 {
	maxStrength := 0.0

	for _, pattern := range dd.config.DependencyPatterns {
		if dd.matchesPattern(fromTask, toTask, pattern) {
			similarity := dd.calculateContentSimilarity(fromTask, toTask)
			strength := pattern.Strength * (0.5 + 0.5*similarity)
			if strength > maxStrength {
				maxStrength = strength
			}
		}
	}

	return maxStrength
}

// GetDependencyReasons returns reasons why tasks are dependent
func (dd *DependencyDetector) GetDependencyReasons(fromTask, toTask types.Task) []string {
	reasons := []string{}

	for _, pattern := range dd.config.DependencyPatterns {
		if dd.matchesPattern(fromTask, toTask, pattern) {
			reasons = append(reasons, pattern.Description)
		}
	}

	return reasons
}
