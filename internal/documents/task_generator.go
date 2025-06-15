// Package documents provides data structures and processing for PRD/TRD document management.
package documents

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"lerian-mcp-memory/internal/logging"

	"github.com/google/uuid"
)

// Task phase constants
const (
	PhaseProduction = "production"
)

// TaskGenerator handles generation of main tasks and sub-tasks
type TaskGenerator struct {
	complexityAnalyzer *ComplexityAnalyzer
	templateManager    *TemplateManager
}

// NewTaskGenerator creates a new task generator
func NewTaskGenerator(logger logging.Logger) *TaskGenerator {
	return &TaskGenerator{
		complexityAnalyzer: NewComplexityAnalyzer(),
		templateManager:    NewTemplateManager(),
	}
}

// GenerateMainTasks generates main tasks from PRD and TRD
func (g *TaskGenerator) GenerateMainTasks(prd *PRDEntity, trd *TRDEntity) ([]*MainTask, error) {
	if prd == nil || trd == nil {
		return nil, errors.New("PRD and TRD are required for main task generation")
	}

	// Analyze complexity
	analysis := g.complexityAnalyzer.AnalyzeProject(prd, trd)

	// Extract key deliverables and phases
	deliverables := g.extractDeliverables(prd, trd)
	phases := g.identifyProjectPhases(deliverables, analysis)

	// Generate main tasks
	mainTasks := []*MainTask{}
	for i := range phases {
		phase := &phases[i]
		task := &MainTask{
			ID:                 uuid.New().String(),
			PRDID:              prd.ID,
			TRDID:              trd.ID,
			TaskID:             GenerateTaskID("MT", i+1),
			Name:               phase.Name,
			Description:        phase.Description,
			Phase:              phase.Type,
			DurationEstimate:   g.estimateTaskDuration(phase.Complexity),
			Dependencies:       phase.Dependencies,
			Deliverables:       phase.Deliverables,
			AcceptanceCriteria: phase.AcceptanceCriteria,
			ComplexityScore:    phase.Complexity,
			Status:             StatusDraft,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
			Repository:         prd.Repository,
		}

		if err := task.Validate(); err != nil {
			return nil, errors.New("invalid main task " + task.TaskID + ": " + err.Error())
		}

		mainTasks = append(mainTasks, task)
	}

	// Set dependencies between tasks
	g.setTaskDependencies(mainTasks)

	return mainTasks, nil
}

// GenerateSubTasks generates sub-tasks for a main task
func (g *TaskGenerator) GenerateSubTasks(mainTask *MainTask, prd *PRDEntity, trd *TRDEntity) ([]*SubTask, error) {
	if mainTask == nil {
		return nil, errors.New("main task is required for sub-task generation")
	}

	// Analyze main task complexity
	taskAnalysis := g.complexityAnalyzer.AnalyzeMainTask(mainTask)

	// Determine sub-task breakdown
	breakdown := g.determineSubTaskBreakdown(mainTask, taskAnalysis)

	// Generate sub-tasks
	subTasks := []*SubTask{}
	for i, component := range breakdown {
		subTask := &SubTask{
			ID:                 uuid.New().String(),
			MainTaskID:         mainTask.ID,
			SubTaskID:          GenerateSubTaskID(mainTask.TaskID, i+1),
			Name:               component.Name,
			Description:        component.Description,
			EstimatedHours:     component.EstimatedHours,
			ImplementationType: component.Type,
			Dependencies:       component.Dependencies,
			AcceptanceCriteria: component.AcceptanceCriteria,
			TechnicalDetails:   component.TechnicalDetails,
			Status:             StatusDraft,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
			Repository:         mainTask.Repository,
		}

		if err := subTask.Validate(); err != nil {
			return nil, errors.New("invalid sub-task " + subTask.SubTaskID + ": " + err.Error())
		}

		subTasks = append(subTasks, subTask)
	}

	return subTasks, nil
}

// ProjectPhase represents a project phase
type ProjectPhase struct {
	Name               string
	Description        string
	Type               string
	Complexity         int
	Dependencies       []string
	Deliverables       []string
	AcceptanceCriteria []string
}

// TaskComponent represents a component of a task
type TaskComponent struct {
	Name               string
	Description        string
	Type               string
	EstimatedHours     int
	Dependencies       []string
	AcceptanceCriteria []string
	TechnicalDetails   map[string]string
}

// extractDeliverables extracts key deliverables from PRD and TRD
func (g *TaskGenerator) extractDeliverables(prd *PRDEntity, trd *TRDEntity) []string {
	deliverables := []string{}
	seen := make(map[string]bool)

	// Extract from PRD sections
	for _, section := range prd.Sections {
		if strings.Contains(strings.ToLower(section.Title), "deliverable") ||
			strings.Contains(strings.ToLower(section.Title), "output") ||
			strings.Contains(strings.ToLower(section.Title), "feature") {
			items := extractListItems(section.Content)
			for _, item := range items {
				if !seen[item] {
					deliverables = append(deliverables, item)
					seen[item] = true
				}
			}
		}
	}

	// Extract from TRD technical requirements
	for _, section := range trd.Sections {
		if strings.Contains(strings.ToLower(section.Title), "component") ||
			strings.Contains(strings.ToLower(section.Title), "module") ||
			strings.Contains(strings.ToLower(section.Title), "service") {
			items := extractListItems(section.Content)
			for _, item := range items {
				if !seen[item] {
					deliverables = append(deliverables, item)
					seen[item] = true
				}
			}
		}
	}

	// Extract from parsed content
	deliverables = append(deliverables, prd.ParsedContent.Requirements...)

	return deliverables
}

// identifyProjectPhases identifies project phases based on deliverables and analysis
func (g *TaskGenerator) identifyProjectPhases(deliverables []string, analysis *ComplexityAnalysis) []ProjectPhase {
	_ = deliverables // unused parameter, kept for future deliverable-based phase identification
	// Use templates to identify common phases
	template := g.templateManager.GetProjectTemplate(analysis.ProjectType)

	phases := []ProjectPhase{}

	// Foundation phase (always first)
	if template.HasFoundation {
		phases = append(phases, ProjectPhase{
			Name:        "Foundation and Setup",
			Description: "Establish project foundation, architecture, and core infrastructure",
			Type:        "foundation",
			Complexity:  20,
			Deliverables: []string{
				"Project structure and configuration",
				"Core architecture implementation",
				"Development environment setup",
				"Basic CI/CD pipeline",
			},
			AcceptanceCriteria: []string{
				"Project builds successfully",
				"Core architecture is testable",
				"Development workflow established",
			},
		})
	}

	// Core features phase
	if len(analysis.CoreFeatures) > 0 {
		phases = append(phases, ProjectPhase{
			Name:         "Core Feature Implementation",
			Description:  "Implement the primary features and functionality",
			Type:         "core",
			Complexity:   40,
			Deliverables: analysis.CoreFeatures,
			AcceptanceCriteria: []string{
				"All core features implemented",
				"Unit tests passing",
				"Integration tests passing",
			},
		})
	}

	// Integration phase (if needed)
	if analysis.RequiresIntegration {
		phases = append(phases, ProjectPhase{
			Name:        "Integration and APIs",
			Description: "Implement external integrations and API endpoints",
			Type:        "integration",
			Complexity:  25,
			Deliverables: []string{
				"API implementation",
				"External service integrations",
				"Authentication and authorization",
			},
			AcceptanceCriteria: []string{
				"APIs documented and tested",
				"Integrations functional",
				"Security measures implemented",
			},
		})
	}

	// Advanced features phase
	if len(analysis.AdvancedFeatures) > 0 {
		phases = append(phases, ProjectPhase{
			Name:         "Advanced Features",
			Description:  "Implement advanced features and optimizations",
			Type:         "advanced",
			Complexity:   30,
			Deliverables: analysis.AdvancedFeatures,
			AcceptanceCriteria: []string{
				"Advanced features operational",
				"Performance optimized",
				"Edge cases handled",
			},
		})
	}

	// Production readiness phase (always last)
	phases = append(phases, ProjectPhase{
		Name:        "Production Readiness",
		Description: "Prepare system for production deployment",
		Type:        PhaseProduction,
		Complexity:  15,
		Deliverables: []string{
			"Production configuration",
			"Monitoring and logging",
			"Documentation",
			"Deployment automation",
		},
		AcceptanceCriteria: []string{
			"System passes all tests",
			"Documentation complete",
			"Deployment successful",
			"Monitoring operational",
		},
	})

	return phases
}

// setTaskDependencies sets dependencies between main tasks
func (g *TaskGenerator) setTaskDependencies(tasks []*MainTask) {
	for i, task := range tasks {
		if i > 0 {
			// Each task depends on the previous one by default
			task.Dependencies = append(task.Dependencies, tasks[i-1].TaskID)
		}

		// Special dependencies based on phase type
		switch task.Phase {
		case "integration":
			// Integration depends on core features
			for j := 0; j < i; j++ {
				if tasks[j].Phase == "core" {
					if !contains(task.Dependencies, tasks[j].TaskID) {
						task.Dependencies = append(task.Dependencies, tasks[j].TaskID)
					}
				}
			}
		case PhaseProduction:
			// Production depends on all previous phases
			for j := 0; j < i; j++ {
				if !contains(task.Dependencies, tasks[j].TaskID) {
					task.Dependencies = append(task.Dependencies, tasks[j].TaskID)
				}
			}
		}
	}
}

// estimateTaskDuration estimates duration based on complexity
func (g *TaskGenerator) estimateTaskDuration(complexity int) string {
	weeks := float64(complexity) / 10.0

	switch {
	case weeks < 1:
		days := int(math.Ceil(weeks * 5))
		return strconv.Itoa(days) + " days"
	case weeks < 2:
		return fmt.Sprintf("%.1f weeks", weeks)
	default:
		return strconv.Itoa(int(weeks)) + "-" + strconv.Itoa(int(weeks)+1) + " weeks"
	}
}

// determineSubTaskBreakdown determines how to break down a main task
func (g *TaskGenerator) determineSubTaskBreakdown(mainTask *MainTask, analysis *TaskAnalysis) []TaskComponent {
	components := []TaskComponent{}

	// Standard components based on task phase
	switch mainTask.Phase {
	case "foundation":
		components = append(components, g.getFoundationComponents(mainTask)...)
	case "core":
		components = append(components, g.getCoreComponents(mainTask, analysis)...)
	case "integration":
		components = append(components, g.getIntegrationComponents(mainTask)...)
	case "advanced":
		components = append(components, g.getAdvancedComponents(mainTask)...)
	case PhaseProduction:
		components = append(components, g.getProductionComponents(mainTask)...)
	default:
		components = append(components, g.getGenericComponents(mainTask, analysis)...)
	}

	// Ensure sub-tasks are 2-4 hours each
	components = g.splitLargeTasks(components)

	return components
}

// getStandardPhaseComponents returns standard components for a given phase
func (g *TaskGenerator) getStandardPhaseComponents(phase string) []TaskComponent {
	switch phase {
	case "foundation":
		return []TaskComponent{
			{
				Name:           "Create project structure and configuration",
				Description:    "Set up project directory structure, configuration files, and build system",
				Type:           "setup",
				EstimatedHours: 3,
				TechnicalDetails: map[string]string{
					"structure": "Follow standard project layout",
					"config":    "Environment-based configuration",
					"build":     "Set up build tools and scripts",
				},
				AcceptanceCriteria: []string{
					"Project structure follows conventions",
					"Configuration system works",
					"Build process successful",
				},
			},
			{
				Name:           "Implement core architecture",
				Description:    "Create base architecture patterns and interfaces",
				Type:           "architecture",
				EstimatedHours: 4,
				TechnicalDetails: map[string]string{
					"pattern":    "Implement chosen architecture pattern",
					"interfaces": "Define core interfaces",
					"separation": "Ensure proper separation of concerns",
				},
				AcceptanceCriteria: []string{
					"Architecture pattern implemented",
					"Interfaces defined and documented",
					"Dependencies properly managed",
				},
			},
			{
				Name:           "Set up development environment",
				Description:    "Configure development tools, linters, and local environment",
				Type:           "tooling",
				EstimatedHours: 2,
				TechnicalDetails: map[string]string{
					"tools":   "Install and configure dev tools",
					"linting": "Set up code quality tools",
					"scripts": "Create development scripts",
				},
				AcceptanceCriteria: []string{
					"Development environment reproducible",
					"Linting and formatting configured",
					"Dev scripts functional",
				},
			},
			{
				Name:           "Create initial tests and CI/CD",
				Description:    "Set up testing framework and continuous integration",
				Type:           "testing",
				EstimatedHours: 3,
				TechnicalDetails: map[string]string{
					"testing":  "Configure test framework",
					"ci":       "Set up CI pipeline",
					"coverage": "Configure code coverage",
				},
				AcceptanceCriteria: []string{
					"Test framework operational",
					"CI pipeline running",
					"Coverage reporting working",
				},
			},
		}
	case PhaseProduction:
		return []TaskComponent{
			{
				Name:           "Configure production environment",
				Description:    "Set up production configuration and secrets",
				Type:           "deployment",
				EstimatedHours: 3,
				TechnicalDetails: map[string]string{
					"config":  "Production configuration",
					"secrets": "Secret management",
					"env":     "Environment setup",
				},
				AcceptanceCriteria: []string{
					"Production config complete",
					"Secrets properly managed",
					"Environment validated",
				},
			},
			{
				Name:           "Implement monitoring and logging",
				Description:    "Set up comprehensive monitoring and log aggregation",
				Type:           "observability",
				EstimatedHours: 4,
				TechnicalDetails: map[string]string{
					"monitoring": "Metrics and monitoring",
					"logging":    "Structured logging",
					"alerting":   "Alert configuration",
				},
				AcceptanceCriteria: []string{
					"Monitoring operational",
					"Logs properly structured",
					"Alerts configured",
				},
			},
			{
				Name:           "Create deployment automation",
				Description:    "Automate deployment process with rollback capability",
				Type:           "automation",
				EstimatedHours: 3,
				TechnicalDetails: map[string]string{
					"deployment": "Deployment scripts",
					"rollback":   "Rollback procedures",
					"validation": "Post-deployment checks",
				},
				AcceptanceCriteria: []string{
					"Deployment automated",
					"Rollback tested",
					"Validation working",
				},
			},
			{
				Name:           "Complete documentation",
				Description:    "Write comprehensive user and developer documentation",
				Type:           "documentation",
				EstimatedHours: 4,
				TechnicalDetails: map[string]string{
					"user":      "User documentation",
					"developer": "Developer guides",
					"api":       "API documentation",
				},
				AcceptanceCriteria: []string{
					"User docs complete",
					"Developer docs comprehensive",
					"API docs generated",
				},
			},
		}
	default:
		return []TaskComponent{}
	}
}

// getFoundationComponents returns standard foundation phase components
func (g *TaskGenerator) getFoundationComponents(mainTask *MainTask) []TaskComponent {
	_ = mainTask // unused parameter, kept for potential future task-specific components
	return g.getStandardPhaseComponents("foundation")
}

// getCoreComponents returns core feature implementation components
func (g *TaskGenerator) getCoreComponents(mainTask *MainTask, analysis *TaskAnalysis) []TaskComponent {
	_ = analysis // unused parameter, kept for future analysis-based component generation
	components := []TaskComponent{}

	// Create components for each deliverable
	for _, deliverable := range mainTask.Deliverables {
		component := TaskComponent{
			Name:           "Implement " + deliverable,
			Description:    "Complete implementation of " + deliverable + " feature",
			Type:           "feature",
			EstimatedHours: 4, // Will be split if needed
			TechnicalDetails: map[string]string{
				"feature":  deliverable,
				"approach": "Follow established patterns",
				"testing":  "Include unit tests",
			},
			AcceptanceCriteria: []string{
				deliverable + " fully functional",
				"Unit tests passing",
				"Code reviewed and documented",
			},
		}
		components = append(components, component)
	}

	return components
}

// getIntegrationComponents returns integration phase components
func (g *TaskGenerator) getIntegrationComponents(mainTask *MainTask) []TaskComponent {
	_ = mainTask // unused parameter, kept for potential future task-specific integration
	return []TaskComponent{
		{
			Name:           "Design and implement API endpoints",
			Description:    "Create RESTful API endpoints with proper validation",
			Type:           "api",
			EstimatedHours: 4,
			TechnicalDetails: map[string]string{
				"design":     "RESTful API design",
				"validation": "Input validation and sanitization",
				"docs":       "API documentation",
			},
			AcceptanceCriteria: []string{
				"APIs follow REST conventions",
				"Validation comprehensive",
				"Documentation complete",
			},
		},
		{
			Name:           "Implement authentication and authorization",
			Description:    "Set up secure authentication and role-based access",
			Type:           "security",
			EstimatedHours: 4,
			TechnicalDetails: map[string]string{
				"auth":     "Authentication mechanism",
				"authz":    "Authorization rules",
				"security": "Security best practices",
			},
			AcceptanceCriteria: []string{
				"Authentication working",
				"Authorization enforced",
				"Security measures in place",
			},
		},
		{
			Name:           "Integrate external services",
			Description:    "Connect to required external APIs and services",
			Type:           "integration",
			EstimatedHours: 3,
			TechnicalDetails: map[string]string{
				"services":       "External service connections",
				"error_handling": "Robust error handling",
				"retry":          "Retry mechanisms",
			},
			AcceptanceCriteria: []string{
				"External services connected",
				"Error handling comprehensive",
				"Retry logic working",
			},
		},
	}
}

// getAdvancedComponents returns advanced feature components
func (g *TaskGenerator) getAdvancedComponents(mainTask *MainTask) []TaskComponent {
	components := []TaskComponent{
		{
			Name:           "Implement caching layer",
			Description:    "Add caching for performance optimization",
			Type:           "performance",
			EstimatedHours: 3,
			TechnicalDetails: map[string]string{
				"strategy":     "Caching strategy",
				"invalidation": "Cache invalidation",
				"monitoring":   "Cache metrics",
			},
			AcceptanceCriteria: []string{
				"Caching functional",
				"Performance improved",
				"Metrics available",
			},
		},
		{
			Name:           "Add advanced error handling",
			Description:    "Implement comprehensive error handling and recovery",
			Type:           "reliability",
			EstimatedHours: 3,
			TechnicalDetails: map[string]string{
				"handling": "Error handling patterns",
				"recovery": "Recovery mechanisms",
				"logging":  "Error logging",
			},
			AcceptanceCriteria: []string{
				"Errors handled gracefully",
				"Recovery mechanisms work",
				"Logging comprehensive",
			},
		},
	}

	// Add specific advanced features from deliverables
	for _, deliverable := range mainTask.Deliverables {
		if !strings.Contains(strings.ToLower(deliverable), "caching") &&
			!strings.Contains(strings.ToLower(deliverable), "error") {
			components = append(components, TaskComponent{
				Name:           "Implement " + deliverable,
				Description:    "Advanced implementation of " + deliverable,
				Type:           "advanced",
				EstimatedHours: 4,
				TechnicalDetails: map[string]string{
					"feature": deliverable,
					"level":   "advanced",
				},
				AcceptanceCriteria: []string{
					deliverable + " implemented",
					"Performance optimized",
					"Edge cases handled",
				},
			})
		}
	}

	return components
}

// getProductionComponents returns production readiness components
func (g *TaskGenerator) getProductionComponents(mainTask *MainTask) []TaskComponent {
	_ = mainTask // unused parameter, kept for potential future task-specific production config
	return g.getStandardPhaseComponents(PhaseProduction)
}

// getGenericComponents returns generic task components
func (g *TaskGenerator) getGenericComponents(mainTask *MainTask, analysis *TaskAnalysis) []TaskComponent {
	components := []TaskComponent{}

	// Break down deliverables into components
	for _, deliverable := range mainTask.Deliverables {
		hours := 4
		if analysis.DeliverableComplexity[deliverable] > 50 {
			hours = 6
		}

		component := TaskComponent{
			Name:           "Implement " + deliverable,
			Description:    "Complete implementation of " + deliverable,
			Type:           "implementation",
			EstimatedHours: hours,
			TechnicalDetails: map[string]string{
				"deliverable": deliverable,
				"complexity":  strconv.Itoa(analysis.DeliverableComplexity[deliverable]),
			},
			AcceptanceCriteria: []string{
				deliverable + " complete",
				"Tests passing",
				"Documentation updated",
			},
		}
		components = append(components, component)
	}

	return components
}

// splitLargeTasks splits tasks larger than 4 hours into smaller sub-tasks
func (g *TaskGenerator) splitLargeTasks(components []TaskComponent) []TaskComponent {
	result := []TaskComponent{}

	for _, component := range components {
		if component.EstimatedHours <= 4 {
			result = append(result, component)
		} else {
			// Split into smaller tasks
			parts := int(math.Ceil(float64(component.EstimatedHours) / 4.0))
			hoursPerPart := component.EstimatedHours / parts

			for i := 0; i < parts; i++ {
				part := TaskComponent{
					Name:               component.Name + " (Part " + strconv.Itoa(i+1) + "/" + strconv.Itoa(parts) + ")",
					Description:        component.Description + " - Part " + strconv.Itoa(i+1) + " of " + strconv.Itoa(parts),
					Type:               component.Type,
					EstimatedHours:     hoursPerPart,
					Dependencies:       component.Dependencies,
					AcceptanceCriteria: []string{"Part " + strconv.Itoa(i+1) + " complete"},
					TechnicalDetails:   component.TechnicalDetails,
				}

				// Add dependency on previous part
				if i > 0 {
					prevPartName := component.Name + " (Part " + strconv.Itoa(i) + "/" + strconv.Itoa(parts) + ")"
					part.Dependencies = append(part.Dependencies, prevPartName)
				}

				// Last part gets the original acceptance criteria
				if i == parts-1 {
					part.AcceptanceCriteria = component.AcceptanceCriteria
				}

				result = append(result, part)
			}
		}
	}

	return result
}

// ComplexityAnalyzer analyzes project and task complexity
type ComplexityAnalyzer struct {
	weights map[string]float64
}

// NewComplexityAnalyzer creates a new complexity analyzer
func NewComplexityAnalyzer() *ComplexityAnalyzer {
	return &ComplexityAnalyzer{
		weights: map[string]float64{
			"features":        1.0,
			"integrations":    1.5,
			"security":        2.0,
			"performance":     1.8,
			"scalability":     2.2,
			"data_complexity": 1.6,
			"ui_complexity":   1.4,
			"testing":         1.2,
		},
	}
}

// ComplexityAnalysis represents the result of complexity analysis
type ComplexityAnalysis struct {
	TotalComplexity       int
	FeatureComplexity     map[string]int
	TechnicalComplexity   int
	IntegrationComplexity int
	ProjectType           string
	CoreFeatures          []string
	AdvancedFeatures      []string
	RequiresIntegration   bool
}

// TaskAnalysis represents analysis of a main task
type TaskAnalysis struct {
	Complexity            int
	DeliverableComplexity map[string]int
	RequiredSkills        []string
	RiskFactors           []string
}

// AnalyzeProject analyzes overall project complexity
func (a *ComplexityAnalyzer) AnalyzeProject(prd *PRDEntity, trd *TRDEntity) *ComplexityAnalysis {
	analysis := &ComplexityAnalysis{
		FeatureComplexity: make(map[string]int),
		CoreFeatures:      []string{},
		AdvancedFeatures:  []string{},
	}

	// Analyze PRD complexity
	prdComplexity := prd.ComplexityScore

	// Analyze technical complexity from TRD
	techComplexity := a.analyzeTechnicalComplexity(trd)

	// Analyze feature complexity
	for _, req := range prd.ParsedContent.Requirements {
		complexity := a.analyzeFeatureComplexity(req)
		analysis.FeatureComplexity[req] = complexity

		if complexity > 50 {
			analysis.AdvancedFeatures = append(analysis.AdvancedFeatures, req)
		} else {
			analysis.CoreFeatures = append(analysis.CoreFeatures, req)
		}
	}

	// Determine project type
	analysis.ProjectType = a.determineProjectType(prd, trd)

	// Check for integrations
	analysis.RequiresIntegration = a.checkForIntegrations(prd, trd)
	if analysis.RequiresIntegration {
		analysis.IntegrationComplexity = 30
	}

	// Calculate total complexity
	analysis.TotalComplexity = (prdComplexity + techComplexity + analysis.IntegrationComplexity) / 3

	return analysis
}

// AnalyzeMainTask analyzes a main task's complexity
func (a *ComplexityAnalyzer) AnalyzeMainTask(task *MainTask) *TaskAnalysis {
	analysis := &TaskAnalysis{
		Complexity:            task.ComplexityScore,
		DeliverableComplexity: make(map[string]int),
		RequiredSkills:        []string{},
		RiskFactors:           []string{},
	}

	// Analyze each deliverable
	for _, deliverable := range task.Deliverables {
		analysis.DeliverableComplexity[deliverable] = a.analyzeFeatureComplexity(deliverable)
	}

	// Determine required skills
	analysis.RequiredSkills = a.extractRequiredSkills(task)

	// Identify risk factors
	analysis.RiskFactors = a.identifyRiskFactors(task)

	return analysis
}

// analyzeTechnicalComplexity analyzes technical complexity from TRD
func (a *ComplexityAnalyzer) analyzeTechnicalComplexity(trd *TRDEntity) int {
	complexity := 0

	// Stack complexity
	complexity += len(trd.TechnicalStack) * 5

	// Architecture complexity
	archPatterns := map[string]int{
		"microservices": 30,
		"monolithic":    10,
		"serverless":    25,
		"event-driven":  20,
		"hexagonal":     15,
	}

	for pattern, score := range archPatterns {
		if strings.Contains(strings.ToLower(trd.Architecture), pattern) {
			complexity += score
			break
		}
	}

	// Dependencies complexity
	complexity += len(trd.Dependencies) * 3

	return complexity
}

// analyzeFeatureComplexity analyzes complexity of a single feature
func (a *ComplexityAnalyzer) analyzeFeatureComplexity(feature string) int {
	complexity := 20 // Base complexity

	featureLower := strings.ToLower(feature)

	// Apply weights for different aspects
	for aspect, weight := range a.weights {
		if strings.Contains(featureLower, aspect) {
			complexity = int(float64(complexity) * weight)
		}
	}

	// Check for complexity indicators
	complexityIndicators := []string{
		"complex", "advanced", "sophisticated", "enterprise",
		"large-scale", "distributed", "real-time", "machine learning",
		"ai", "blockchain", "quantum",
	}

	for _, indicator := range complexityIndicators {
		if strings.Contains(featureLower, indicator) {
			complexity += 15
		}
	}

	return complexity
}

// determineProjectType determines the type of project
func (a *ComplexityAnalyzer) determineProjectType(prd *PRDEntity, trd *TRDEntity) string {
	content := strings.ToLower(prd.Content + " " + trd.Content)

	projectTypes := map[string][]string{
		"web-application": {"web", "frontend", "backend", "ui", "ux"},
		"api-service":     {"api", "rest", "graphql", "service", "endpoint"},
		"cli-tool":        {"cli", "command", "terminal", "console"},
		"mobile-app":      {"mobile", "ios", "android", "app"},
		"data-pipeline":   {"data", "etl", "pipeline", "analytics", "warehouse"},
		"ml-system":       {"machine learning", "ml", "ai", "model", "training"},
		"infrastructure":  {"infrastructure", "devops", "deployment", "kubernetes"},
	}

	scores := make(map[string]int)
	for pType, keywords := range projectTypes {
		for _, keyword := range keywords {
			if strings.Contains(content, keyword) {
				scores[pType]++
			}
		}
	}

	// Find type with highest score
	maxScore := 0
	projectType := "general"
	for pType, score := range scores {
		if score > maxScore {
			maxScore = score
			projectType = pType
		}
	}

	return projectType
}

// checkForIntegrations checks if project requires external integrations
func (a *ComplexityAnalyzer) checkForIntegrations(prd *PRDEntity, trd *TRDEntity) bool {
	integrationKeywords := []string{
		"integration", "api", "external", "third-party", "webhook",
		"oauth", "saml", "ldap", "payment", "email", "sms",
		"cloud", "aws", "azure", "gcp", "database", "message queue",
	}

	content := strings.ToLower(prd.Content + " " + trd.Content)

	for _, keyword := range integrationKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

// extractRequiredSkills extracts required skills for a task
func (a *ComplexityAnalyzer) extractRequiredSkills(task *MainTask) []string {
	skills := []string{}
	skillMap := make(map[string]bool)

	// Extract from task description and deliverables
	content := strings.ToLower(task.Description + " " + strings.Join(task.Deliverables, " "))

	skillKeywords := map[string]string{
		"frontend":     "Frontend Development",
		"backend":      "Backend Development",
		"database":     "Database Design",
		"api":          "API Development",
		"security":     "Security Engineering",
		"devops":       "DevOps",
		"testing":      "Testing/QA",
		"ui":           "UI Design",
		"ux":           "UX Design",
		"architecture": "Software Architecture",
		"cloud":        "Cloud Infrastructure",
		"mobile":       "Mobile Development",
		"data":         "Data Engineering",
		"ml":           "Machine Learning",
		"performance":  "Performance Optimization",
	}

	for keyword, skill := range skillKeywords {
		if strings.Contains(content, keyword) && !skillMap[skill] {
			skills = append(skills, skill)
			skillMap[skill] = true
		}
	}

	return skills
}

// identifyRiskFactors identifies risk factors for a task
func (a *ComplexityAnalyzer) identifyRiskFactors(task *MainTask) []string {
	risks := []string{}

	// High complexity risk
	if task.ComplexityScore > 70 {
		risks = append(risks, "High complexity may lead to delays")
	}

	// Many dependencies risk
	if len(task.Dependencies) > 3 {
		risks = append(risks, "Multiple dependencies could cause bottlenecks")
	}

	// Check for specific risk keywords
	riskKeywords := map[string]string{
		"security":    "Security implementation requires careful review",
		"performance": "Performance requirements need thorough testing",
		"integration": "External integrations may have compatibility issues",
		"migration":   "Data migration carries risk of data loss",
		"real-time":   "Real-time requirements add complexity",
		"scale":       "Scalability requirements need architecture review",
	}

	content := strings.ToLower(task.Description)
	for keyword, risk := range riskKeywords {
		if strings.Contains(content, keyword) {
			risks = append(risks, risk)
		}
	}

	return risks
}

// TemplateManager manages task generation templates
type TemplateManager struct {
	projectTemplates map[string]*ProjectTemplate
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	tm := &TemplateManager{
		projectTemplates: make(map[string]*ProjectTemplate),
	}
	tm.initializeTemplates()
	return tm
}

// ProjectTemplate represents a project template
type ProjectTemplate struct {
	Type                string
	HasFoundation       bool
	RequiresIntegration bool
	StandardPhases      []string
}

// initializeTemplates initializes standard project templates
func (tm *TemplateManager) initializeTemplates() {
	tm.projectTemplates["web-application"] = &ProjectTemplate{
		Type:                "web-application",
		HasFoundation:       true,
		RequiresIntegration: true,
		StandardPhases:      []string{"foundation", "core", "integration", "advanced", PhaseProduction},
	}

	tm.projectTemplates["api-service"] = &ProjectTemplate{
		Type:                "api-service",
		HasFoundation:       true,
		RequiresIntegration: true,
		StandardPhases:      []string{"foundation", "core", "integration", PhaseProduction},
	}

	tm.projectTemplates["cli-tool"] = &ProjectTemplate{
		Type:                "cli-tool",
		HasFoundation:       true,
		RequiresIntegration: false,
		StandardPhases:      []string{"foundation", "core", "advanced", PhaseProduction},
	}

	tm.projectTemplates["general"] = &ProjectTemplate{
		Type:                "general",
		HasFoundation:       true,
		RequiresIntegration: true,
		StandardPhases:      []string{"foundation", "core", "integration", "advanced", PhaseProduction},
	}
}

// GetProjectTemplate returns a project template
func (tm *TemplateManager) GetProjectTemplate(projectType string) *ProjectTemplate {
	if template, ok := tm.projectTemplates[projectType]; ok {
		return template
	}
	return tm.projectTemplates["general"]
}

// EstimateProjectTimeline estimates the overall project timeline
func EstimateProjectTimeline(mainTasks []*MainTask) string {
	totalWeeks := 0.0

	// Calculate critical path (simplified - assumes sequential)
	for _, task := range mainTasks {
		// Parse duration estimate
		if strings.Contains(task.DurationEstimate, "week") {
			var weeks float64
			if _, err := fmt.Sscanf(task.DurationEstimate, "%f", &weeks); err != nil {
				// Unable to parse duration, skip this task
				continue
			}
			totalWeeks += weeks
		} else if strings.Contains(task.DurationEstimate, "day") {
			var days int
			if _, err := fmt.Sscanf(task.DurationEstimate, "%d", &days); err != nil {
				// Unable to parse duration, skip this task
				continue
			}
			totalWeeks += float64(days) / 5.0
		}
	}

	// Add buffer for integration and testing
	totalWeeks *= 1.2

	switch {
	case totalWeeks < 4:
		return strconv.Itoa(int(totalWeeks)) + "-" + strconv.Itoa(int(totalWeeks)+1) + " weeks"
	case totalWeeks < 12:
		months := totalWeeks / 4.0
		return fmt.Sprintf("%.1f months", months)
	default:
		months := totalWeeks / 4.0
		return strconv.Itoa(int(months)) + "-" + strconv.Itoa(int(months)+1) + " months"
	}
}

// GenerateTaskDependencyGraph generates a dependency graph for tasks
func GenerateTaskDependencyGraph(mainTasks []*MainTask) string {
	var graph strings.Builder

	graph.WriteString("Task Dependency Graph:\n")
	graph.WriteString("```mermaid\n")
	graph.WriteString("graph TD\n")

	// Add nodes
	for _, task := range mainTasks {
		graph.WriteString("    " + task.TaskID + "[\"" + task.TaskID + ": " + task.Name + "\"]\n")
	}

	// Add edges
	for _, task := range mainTasks {
		for _, dep := range task.Dependencies {
			graph.WriteString("    " + dep + " --> " + task.TaskID + "\n")
		}
	}

	graph.WriteString("```\n")

	return graph.String()
}
