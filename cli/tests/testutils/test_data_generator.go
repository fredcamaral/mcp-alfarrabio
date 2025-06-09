package testutils

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct {
	random *rand.Rand
}

// NewTestDataGenerator creates a new test data generator
func NewTestDataGenerator() *TestDataGenerator {
	return &TestDataGenerator{
		// #nosec G404 - Using math/rand for test data generation is acceptable
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// PatternTemplate defines a template for generating patterned tasks
type PatternTemplate struct {
	Name     string
	Sequence []string
	Count    int
	Variance float64 // 0.0 = no variance, 1.0 = high variance
}

// WorkflowTemplate defines a template for generating workflow tasks
type WorkflowTemplate struct {
	Name   string
	Phases []WorkflowPhase
	Count  int
}

// WorkflowPhase represents a phase in a workflow
type WorkflowPhase struct {
	Name         string
	Tasks        []string
	Parallelism  int
	Dependencies []string
}

// TemporalTemplate defines a template for generating temporal patterns
type TemporalTemplate struct {
	Name      string
	TaskName  string
	Frequency time.Duration
	Count     int
	StartTime time.Time
}

// Task generation methods

// CreateTask creates a single task with specified parameters
func (g *TestDataGenerator) CreateTask(content, priority, status string, metadata map[string]interface{}) *entities.Task {
	// Extract task type from metadata and use as tag
	tags := []string{}
	if taskType, ok := metadata["type"].(string); ok {
		tags = append(tags, taskType)
	} else {
		tags = append(tags, g.inferTaskType(content))
	}

	return &entities.Task{
		ID:         uuid.New().String(),
		Content:    content,
		Priority:   entities.Priority(priority),
		Status:     entities.Status(status),
		Repository: "test-repo",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Tags:       tags,
	}
}

// CreateTaskAt creates a task at a specific time
func (g *TestDataGenerator) CreateTaskAt(content, priority, status string, createdAt time.Time, metadata map[string]interface{}) *entities.Task {
	task := g.CreateTask(content, priority, status, metadata)
	task.CreatedAt = createdAt
	task.UpdatedAt = createdAt
	return task
}

// GenerateRandomTasks generates a specified number of random tasks
func (g *TestDataGenerator) GenerateRandomTasks(repository string, count int) []*entities.Task {
	var tasks []*entities.Task

	taskTypes := []string{"feature", "bug-fix", "refactor", "documentation", "testing"}
	priorities := []string{"high", "medium", "low"}
	statuses := []string{"pending", "in_progress", "completed", "cancelled"}

	for i := 0; i < count; i++ {
		taskType := taskTypes[g.random.Intn(len(taskTypes))]
		task := &entities.Task{
			ID:         uuid.New().String(),
			Content:    fmt.Sprintf("Random task %d", i+1),
			Priority:   entities.Priority(priorities[g.random.Intn(len(priorities))]),
			Status:     entities.Status(statuses[g.random.Intn(len(statuses))]),
			Repository: repository,
			CreatedAt:  time.Now().Add(-time.Duration(g.random.Intn(720)) * time.Hour), // Random time within 30 days
			UpdatedAt:  time.Now().Add(-time.Duration(g.random.Intn(24)) * time.Hour),  // Random time within 1 day
			Tags:       []string{taskType},
		}

		tasks = append(tasks, task)
	}

	return tasks
}

// GeneratePatternedTasks generates tasks following specific patterns
func (g *TestDataGenerator) GeneratePatternedTasks(repository string, patterns []PatternTemplate) []*entities.Task {
	var tasks []*entities.Task

	for _, pattern := range patterns {
		for i := 0; i < pattern.Count; i++ {
			baseTime := time.Now().AddDate(0, 0, -30+i) // Spread over 30 days

			for j, taskType := range pattern.Sequence {
				// Add variance if specified
				delay := time.Duration(j) * time.Hour
				if pattern.Variance > 0 {
					varianceHours := int(24 * pattern.Variance)
					delay += time.Duration(g.random.Intn(varianceHours)) * time.Hour
				}

				task := &entities.Task{
					ID:         uuid.New().String(),
					Content:    fmt.Sprintf("%s task %d-%d", taskType, i+1, j+1),
					Priority:   entities.PriorityMedium,
					Status:     entities.StatusCompleted,
					Repository: repository,
					CreatedAt:  baseTime.Add(delay),
					UpdatedAt:  baseTime.Add(delay + time.Hour),
					Tags:       []string{taskType, pattern.Name},
				}

				// Add some failed tasks for variance
				if pattern.Variance > 0.5 && g.random.Float64() < 0.1 {
					task.Status = entities.StatusCancelled
				}

				tasks = append(tasks, task)
			}
		}
	}

	// Shuffle to simulate real-world disorder
	g.random.Shuffle(len(tasks), func(i, j int) {
		tasks[i], tasks[j] = tasks[j], tasks[i]
	})

	return tasks
}

// GenerateWorkflowTasks generates tasks following workflow patterns
func (g *TestDataGenerator) GenerateWorkflowTasks(repository string, workflows []WorkflowTemplate) []*entities.Task {
	var tasks []*entities.Task

	for _, workflow := range workflows {
		for i := 0; i < workflow.Count; i++ {
			baseTime := time.Now().AddDate(0, 0, -60+i*7) // Weekly cycles

			for phaseIdx, phase := range workflow.Phases {
				phaseStartTime := baseTime.Add(time.Duration(phaseIdx*24) * time.Hour)

				// Create tasks for this phase
				for taskIdx, taskName := range phase.Tasks {
					// Simulate parallelism by starting tasks at similar times
					taskStartTime := phaseStartTime
					if phase.Parallelism > 1 {
						// Spread start times within the phase
						maxSpread := 4 * time.Hour
						spread := time.Duration(taskIdx * int(maxSpread) / len(phase.Tasks))
						taskStartTime = taskStartTime.Add(spread)
					}

					task := &entities.Task{
						ID:         uuid.New().String(),
						Content:    fmt.Sprintf("%s - %s %d", phase.Name, taskName, i+1),
						Priority:   entities.PriorityMedium,
						Status:     entities.StatusCompleted,
						Repository: repository,
						CreatedAt:  taskStartTime,
						UpdatedAt:  taskStartTime.Add(4 * time.Hour),
						Tags:       []string{g.inferTaskType(taskName), workflow.Name, phase.Name},
					}

					tasks = append(tasks, task)
				}
			}
		}
	}

	return tasks
}

// GenerateTemporalTasks generates tasks with temporal patterns
func (g *TestDataGenerator) GenerateTemporalTasks(repository string, templates []TemporalTemplate) []*entities.Task {
	var tasks []*entities.Task

	for _, template := range templates {
		for i := 0; i < template.Count; i++ {
			// Calculate time for this occurrence
			taskTime := template.StartTime.Add(time.Duration(i) * template.Frequency)

			// Add some jitter (±10% of frequency)
			jitter := time.Duration(float64(template.Frequency) * 0.1 * (g.random.Float64() - 0.5))
			taskTime = taskTime.Add(jitter)

			task := &entities.Task{
				ID:         uuid.New().String(),
				Content:    fmt.Sprintf("%s %d", template.TaskName, i+1),
				Priority:   entities.PriorityMedium,
				Status:     entities.StatusCompleted,
				Repository: repository,
				CreatedAt:  taskTime,
				UpdatedAt:  taskTime.Add(30 * time.Minute),
				Tags:       []string{g.inferTaskType(template.TaskName), template.Name, "temporal"},
			}

			tasks = append(tasks, task)
		}
	}

	return tasks
}

// Session generation methods

// GenerateRandomSessions generates random work sessions
func (g *TestDataGenerator) GenerateRandomSessions(repository string, count int) []*entities.Session {
	var sessions []*entities.Session

	for i := 0; i < count; i++ {
		duration := time.Duration(1+g.random.Intn(8)) * time.Hour // 1-8 hours
		startTime := time.Now().Add(-time.Duration(g.random.Intn(720)) * time.Hour)

		session := &entities.Session{
			ID:         uuid.New().String(),
			Repository: repository,
			Duration:   duration,
			CreatedAt:  startTime,
			UpdatedAt:  startTime.Add(duration),
			Metadata: map[string]interface{}{
				"session_type": g.random.Intn(3), // 0=morning, 1=afternoon, 2=evening
				"productivity": g.random.Float64(),
			},
		}

		sessions = append(sessions, session)
	}

	return sessions
}

// Pattern generation methods

// GenerateRandomPatterns generates random task patterns
func (g *TestDataGenerator) GenerateRandomPatterns(repository string, count int) []*entities.TaskPattern {
	var patterns []*entities.TaskPattern

	patternTypes := []entities.PatternType{
		entities.PatternTypeSequence,
		entities.PatternTypeWorkflow,
		entities.PatternTypeTemporal,
	}

	for i := 0; i < count; i++ {
		patternType := patternTypes[g.random.Intn(len(patternTypes))]

		pattern := &entities.TaskPattern{
			ID:          uuid.New().String(),
			Type:        patternType,
			Name:        fmt.Sprintf("Random pattern %d", i+1),
			Repository:  repository,
			Confidence:  0.5 + g.random.Float64()*0.5,       // 0.5-1.0
			SuccessRate: 0.6 + g.random.Float64()*0.4,       // 0.6-1.0
			Frequency:   float64(g.random.Intn(20) + 5),     // 5-25
			ProjectType: string(entities.ProjectTypeWebApp), // Default
			Metadata:    map[string]interface{}{"keywords": g.generateRandomKeywords()},
			CreatedAt:   time.Now().Add(-time.Duration(g.random.Intn(30)) * 24 * time.Hour),
			UpdatedAt:   time.Now(),
		}

		// Add type-specific data
		switch patternType {
		case entities.PatternTypeSequence:
			pattern.Sequence = g.generateRandomSequence()
		case entities.PatternTypeWorkflow:
			pattern.Metadata["workflow"] = g.generateRandomWorkflowPhases()
		case entities.PatternTypeTemporal:
			pattern.Frequency = float64(g.random.Intn(168) + 1) // 1-168 hours
		}

		patterns = append(patterns, pattern)
	}

	return patterns
}

// Project creation methods

// CreateTempProject creates a temporary project directory with specified files
func (g *TestDataGenerator) CreateTempProject(files map[string]string) string {
	tempDir, err := os.MkdirTemp("", "test-project-*")
	if err != nil {
		panic(fmt.Sprintf("Failed to create temp dir: %v", err))
	}

	for filePath, content := range files {
		fullPath := filepath.Join(tempDir, filePath)

		// Create directory if needed
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(fmt.Sprintf("Failed to create dir %s: %v", dir, err))
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			panic(fmt.Sprintf("Failed to write file %s: %v", fullPath, err))
		}
	}

	return tempDir
}

// Helper methods

func (g *TestDataGenerator) inferTaskType(content string) string {
	content = strings.ToLower(content)

	if strings.Contains(content, "test") {
		return "testing"
	} else if strings.Contains(content, "fix") || strings.Contains(content, "bug") {
		return "bug-fix"
	} else if strings.Contains(content, "design") || strings.Contains(content, "ui") {
		return "design"
	} else if strings.Contains(content, "implement") || strings.Contains(content, "code") {
		return "implementation"
	} else if strings.Contains(content, "doc") {
		return "documentation"
	} else if strings.Contains(content, "refactor") {
		return "refactor"
	} else if strings.Contains(content, "deploy") {
		return "deployment"
	} else if strings.Contains(content, "setup") || strings.Contains(content, "config") {
		return "setup"
	}

	return "feature"
}

func (g *TestDataGenerator) generateRandomDescription() string {
	descriptions := []string{
		"Implement new functionality for the application",
		"Fix critical issue affecting user experience",
		"Refactor existing code to improve maintainability",
		"Add comprehensive tests for better coverage",
		"Update documentation for better clarity",
		"Optimize performance in key areas",
		"Enhance security measures",
		"Improve user interface design",
		"Configure deployment pipeline",
		"Research new technology options",
	}

	return descriptions[g.random.Intn(len(descriptions))]
}

func (g *TestDataGenerator) generateRandomKeywords() []string {
	allKeywords := []string{
		"api", "frontend", "backend", "database", "testing", "security",
		"performance", "ui", "ux", "documentation", "deployment", "ci",
		"cd", "monitoring", "logging", "authentication", "authorization",
		"validation", "optimization", "refactoring", "bug-fix", "feature",
	}

	count := 3 + g.random.Intn(5) // 3-7 keywords
	keywords := make([]string, 0, count)

	// Randomly select keywords without duplicates
	selected := make(map[int]bool)
	for len(keywords) < count {
		idx := g.random.Intn(len(allKeywords))
		if !selected[idx] {
			keywords = append(keywords, allKeywords[idx])
			selected[idx] = true
		}
	}

	return keywords
}

func (g *TestDataGenerator) generateRandomSequence() []entities.PatternStep {
	stepTypes := []string{"design", "implement", "test", "review", "deploy", "document"}
	sequenceLength := 3 + g.random.Intn(4) // 3-6 steps

	sequence := make([]entities.PatternStep, 0, sequenceLength)

	for i := 0; i < sequenceLength; i++ {
		step := entities.PatternStep{
			Order:    i + 1,
			TaskType: stepTypes[g.random.Intn(len(stepTypes))],
			Keywords: g.generateRandomKeywords()[:2], // 2 keywords per step
		}
		sequence = append(sequence, step)
	}

	return sequence
}

func (g *TestDataGenerator) generateRandomWorkflowPhases() []entities.WorkflowPhase {
	phaseNames := []string{"planning", "development", "testing", "deployment", "monitoring"}
	phaseCount := 2 + g.random.Intn(3) // 2-4 phases

	phases := make([]entities.WorkflowPhase, 0, phaseCount)

	for i := 0; i < phaseCount; i++ {
		phase := entities.WorkflowPhase{
			Name:         phaseNames[i],
			Order:        i + 1,
			TaskPatterns: []string{fmt.Sprintf("task-%d-1", i), fmt.Sprintf("task-%d-2", i)},
		}
		phases = append(phases, phase)
	}

	return phases
}
