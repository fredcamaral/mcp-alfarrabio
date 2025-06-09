// Package tasks provides contextual task suggestions based on project state.
package tasks

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// Suggester provides contextual task suggestions
type Suggester struct {
	config SuggesterConfig
}

// SuggesterConfig represents configuration for task suggestions
type SuggesterConfig struct {
	MaxSuggestions            int           `json:"max_suggestions"`
	MinConfidenceScore        float64       `json:"min_confidence_score"`
	EnablePhaseAnalysis       bool          `json:"enable_phase_analysis"`
	EnableBottleneckDetection bool          `json:"enable_bottleneck_detection"`
	EnablePatternMatching     bool          `json:"enable_pattern_matching"`
	SuggestionTimeout         time.Duration `json:"suggestion_timeout"`
}

// DefaultSuggesterConfig returns default suggester configuration
func DefaultSuggesterConfig() SuggesterConfig {
	return SuggesterConfig{
		MaxSuggestions:            20,
		MinConfidenceScore:        0.6,
		EnablePhaseAnalysis:       true,
		EnableBottleneckDetection: true,
		EnablePatternMatching:     true,
		SuggestionTimeout:         30 * time.Second,
	}
}

// NewSuggester creates a new task suggester
func NewSuggester() *Suggester {
	return &Suggester{
		config: DefaultSuggesterConfig(),
	}
}

// NewSuggesterWithConfig creates a new task suggester with custom config
func NewSuggesterWithConfig(config SuggesterConfig) *Suggester {
	return &Suggester{
		config: config,
	}
}

// TaskSuggestion represents a suggested task with context
type TaskSuggestion struct {
	Task           types.Task         `json:"task"`
	Confidence     float64            `json:"confidence"`
	Reasoning      []string           `json:"reasoning"`
	Priority       SuggestionPriority `json:"priority"`
	Category       SuggestionCategory `json:"category"`
	BasedOn        []string           `json:"based_on"`        // Task IDs this suggestion is based on
	Prerequisites  []string           `json:"prerequisites"`   // Task IDs that should be completed first
	EstimatedValue float64            `json:"estimated_value"` // Business/technical value estimate
}

// SuggestionPriority represents the priority of a suggestion
type SuggestionPriority string

const (
	SuggestionPriorityImmediate SuggestionPriority = "immediate"
	SuggestionPriorityHigh      SuggestionPriority = "high"
	SuggestionPriorityMedium    SuggestionPriority = "medium"
	SuggestionPriorityLow       SuggestionPriority = "low"
	SuggestionPriorityFuture    SuggestionPriority = "future"
)

// SuggestionCategory represents the category of a suggestion
type SuggestionCategory string

const (
	SuggestionCategoryPhaseProgression     SuggestionCategory = "phase_progression"
	SuggestionCategoryBottleneckResolution SuggestionCategory = "bottleneck_resolution"
	SuggestionCategoryQualityImprovement   SuggestionCategory = "quality_improvement"
	SuggestionCategoryRiskMitigation       SuggestionCategory = "risk_mitigation"
	SuggestionCategoryEfficiencyGain       SuggestionCategory = "efficiency_gain"
	SuggestionCategoryDependencySetup      SuggestionCategory = "dependency_setup"
	SuggestionCategoryKnowledgeGap         SuggestionCategory = "knowledge_gap"
	SuggestionCategoryArchitectural        SuggestionCategory = "architectural"
	SuggestionCategoryMaintenance          SuggestionCategory = "maintenance"
)

// SuggestTasks generates contextual task suggestions based on project state
func (s *Suggester) SuggestTasks(ctx context.Context, projectState types.ProjectState, existingTasks []types.Task, genContext types.TaskGenerationContext) ([]TaskSuggestion, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, s.config.SuggestionTimeout)
	defer cancel()

	suggestions := []TaskSuggestion{}

	// Analyze current project phase and suggest next steps
	if s.config.EnablePhaseAnalysis {
		phaseSuggestions := s.suggestPhaseTransitionTasks(projectState, existingTasks, genContext)
		suggestions = append(suggestions, phaseSuggestions...)
	}

	// Identify and resolve bottlenecks
	if s.config.EnableBottleneckDetection {
		bottleneckSuggestions := s.suggestBottleneckResolutionTasks(projectState, existingTasks, genContext)
		suggestions = append(suggestions, bottleneckSuggestions...)
	}

	// Suggest quality improvement tasks
	qualitySuggestions := s.suggestQualityImprovementTasks(existingTasks, genContext)
	suggestions = append(suggestions, qualitySuggestions...)

	// Suggest risk mitigation tasks
	riskSuggestions := s.suggestRiskMitigationTasks(projectState, existingTasks, genContext)
	suggestions = append(suggestions, riskSuggestions...)

	// Suggest dependency setup tasks
	dependencySuggestions := s.suggestDependencySetupTasks(existingTasks, genContext)
	suggestions = append(suggestions, dependencySuggestions...)

	// Suggest architectural improvements
	architecturalSuggestions := s.suggestArchitecturalTasks(projectState, existingTasks, genContext)
	suggestions = append(suggestions, architecturalSuggestions...)

	// Filter by confidence score and limit results
	filteredSuggestions := s.filterAndRankSuggestions(suggestions)

	return filteredSuggestions, nil
}

// suggestPhaseTransitionTasks suggests tasks for moving to the next project phase
func (s *Suggester) suggestPhaseTransitionTasks(projectState types.ProjectState, existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	switch projectState.Phase {
	case types.PhaseDiscovery:
		suggestions = append(suggestions, s.suggestDiscoveryToRequirementsTasks(projectState, existingTasks, genContext)...)
	case types.PhaseRequirements:
		suggestions = append(suggestions, s.suggestRequirementsToDesignTasks(projectState, existingTasks, genContext)...)
	case types.PhaseDesign:
		suggestions = append(suggestions, s.suggestDesignToDevelopmentTasks(projectState, existingTasks, genContext)...)
	case types.PhaseDevelopment:
		suggestions = append(suggestions, s.suggestDevelopmentToTestingTasks(projectState, existingTasks, genContext)...)
	case types.PhaseTesting:
		suggestions = append(suggestions, s.suggestTestingToDeploymentTasks(projectState, existingTasks, genContext)...)
	case types.PhaseDeployment:
		suggestions = append(suggestions, s.suggestDeploymentToMaintenanceTasks(projectState, existingTasks, genContext)...)
	}

	return suggestions
}

// suggestDiscoveryToRequirementsTasks suggests tasks for transitioning from discovery to requirements
func (s *Suggester) suggestDiscoveryToRequirementsTasks(projectState types.ProjectState, existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	// Check if user research is complete
	if !s.hasTaskType(existingTasks, types.TaskTypeResearch) {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("user_research"),
				Title:       "Conduct User Research and Stakeholder Interviews",
				Description: "Gather requirements through user interviews, surveys, and stakeholder meetings to understand needs and expectations",
				Type:        types.TaskTypeResearch,
				Priority:    types.TaskPriorityHigh,
				EstimatedEffort: types.EffortEstimate{
					Hours:            16.0,
					Days:             2.0,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"At least 5 user interviews completed",
					"Key stakeholders identified and interviewed",
					"User personas documented",
					"Requirements gathering report created",
				},
				Tags: []string{"research", "requirements", "users", "stakeholders"},
			},
			Confidence: 0.9,
			Priority:   SuggestionPriorityHigh,
			Category:   SuggestionCategoryPhaseProgression,
			Reasoning: []string{
				"Project is in discovery phase",
				"User research is essential before defining requirements",
				"No research tasks found in existing tasks",
			},
			EstimatedValue: 0.9,
		})
	}

	// Suggest market analysis if not done
	if !s.hasKeyword(existingTasks, "market") && !s.hasKeyword(existingTasks, "competitive") {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("market_analysis"),
				Title:       "Conduct Market and Competitive Analysis",
				Description: "Analyze market landscape, competitors, and positioning to inform product strategy and requirements",
				Type:        types.TaskTypeAnalysis,
				Priority:    types.TaskPriorityMedium,
				EstimatedEffort: types.EffortEstimate{
					Hours:            12.0,
					Days:             1.5,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"Competitive landscape documented",
					"Market size and opportunity assessed",
					"Feature gap analysis completed",
					"Positioning strategy defined",
				},
				Tags: []string{"analysis", "market", "competitive", "strategy"},
			},
			Confidence: 0.8,
			Priority:   SuggestionPriorityMedium,
			Category:   SuggestionCategoryPhaseProgression,
			Reasoning: []string{
				"Market analysis helps inform requirements",
				"No competitive analysis tasks found",
				"Essential for product positioning",
			},
			EstimatedValue: 0.7,
		})
	}

	return suggestions
}

// suggestRequirementsToDesignTasks suggests tasks for transitioning to design phase
func (s *Suggester) suggestRequirementsToDesignTasks(projectState types.ProjectState, existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	// Suggest PRD creation if not exists
	if !s.hasKeyword(existingTasks, "prd") && !s.hasKeyword(existingTasks, "requirements document") {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("create_prd"),
				Title:       "Create Product Requirements Document (PRD)",
				Description: "Compile research findings into a comprehensive PRD documenting functional and non-functional requirements",
				Type:        types.TaskTypeDocumentation,
				Priority:    types.TaskPriorityCritical,
				EstimatedEffort: types.EffortEstimate{
					Hours:            24.0,
					Days:             3.0,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"All functional requirements documented",
					"Non-functional requirements specified",
					"User stories created and prioritized",
					"Success metrics defined",
					"PRD reviewed and approved by stakeholders",
				},
				Tags: []string{"prd", "requirements", "documentation", "specifications"},
			},
			Confidence: 0.95,
			Priority:   SuggestionPriorityImmediate,
			Category:   SuggestionCategoryPhaseProgression,
			Reasoning: []string{
				"Requirements phase needs formal documentation",
				"PRD is essential for design phase",
				"No PRD creation task found",
			},
			EstimatedValue: 1.0,
		})
	}

	// Suggest user story creation
	if !s.hasKeyword(existingTasks, "user story") && !s.hasKeyword(existingTasks, "user stories") {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("create_user_stories"),
				Title:       "Create and Prioritize User Stories",
				Description: "Break down requirements into detailed user stories with acceptance criteria and priority rankings",
				Type:        types.TaskTypeAnalysis,
				Priority:    types.TaskPriorityHigh,
				EstimatedEffort: types.EffortEstimate{
					Hours:            16.0,
					Days:             2.0,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"User stories follow standard format (As a...)",
					"Acceptance criteria defined for each story",
					"Stories estimated and prioritized",
					"Epic-level grouping established",
				},
				Tags: []string{"user-stories", "requirements", "agile", "prioritization"},
			},
			Confidence: 0.9,
			Priority:   SuggestionPriorityHigh,
			Category:   SuggestionCategoryPhaseProgression,
			Reasoning: []string{
				"User stories bridge requirements and design",
				"Essential for agile development",
				"No user story creation found",
			},
			EstimatedValue: 0.85,
		})
	}

	return suggestions
}

// suggestDesignToDevelopmentTasks suggests tasks for transitioning to development
func (s *Suggester) suggestDesignToDevelopmentTasks(projectState types.ProjectState, existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	// Suggest technical architecture if not done
	if !s.hasTaskType(existingTasks, types.TaskTypeArchitecture) {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("technical_architecture"),
				Title:       "Design Technical Architecture and System Design",
				Description: "Create comprehensive technical architecture including system components, data flow, and technology stack decisions",
				Type:        types.TaskTypeArchitecture,
				Priority:    types.TaskPriorityCritical,
				EstimatedEffort: types.EffortEstimate{
					Hours:            32.0,
					Days:             4.0,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"System architecture diagram created",
					"Technology stack finalized",
					"Data flow and storage design completed",
					"API design and contracts defined",
					"Security architecture documented",
				},
				Tags: []string{"architecture", "system-design", "technical", "planning"},
			},
			Confidence: 0.95,
			Priority:   SuggestionPriorityImmediate,
			Category:   SuggestionCategoryPhaseProgression,
			Reasoning: []string{
				"Technical architecture required before development",
				"No architecture tasks found",
				"Critical for development success",
			},
			EstimatedValue: 1.0,
		})
	}

	// Suggest development environment setup
	if !s.hasKeyword(existingTasks, "environment") && !s.hasKeyword(existingTasks, "setup") {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("dev_environment"),
				Title:       "Setup Development Environment and CI/CD Pipeline",
				Description: "Configure development environment, version control, and continuous integration/deployment pipeline",
				Type:        types.TaskTypeImplementation,
				Priority:    types.TaskPriorityHigh,
				EstimatedEffort: types.EffortEstimate{
					Hours:            20.0,
					Days:             2.5,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"Development environment configured for all team members",
					"Version control repository setup with branching strategy",
					"CI/CD pipeline configured and tested",
					"Code quality tools integrated",
					"Deployment processes documented",
				},
				Tags: []string{"devops", "environment", "ci-cd", "setup"},
			},
			Confidence: 0.9,
			Priority:   SuggestionPriorityHigh,
			Category:   SuggestionCategoryDependencySetup,
			Reasoning: []string{
				"Development environment needed before coding",
				"CI/CD critical for team collaboration",
				"No environment setup tasks found",
			},
			EstimatedValue: 0.8,
		})
	}

	return suggestions
}

// suggestDevelopmentToTestingTasks suggests tasks for transitioning to testing
func (s *Suggester) suggestDevelopmentToTestingTasks(projectState types.ProjectState, existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	// Suggest test strategy if not defined
	if !s.hasTaskType(existingTasks, types.TaskTypeTesting) {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("test_strategy"),
				Title:       "Define Testing Strategy and Test Plans",
				Description: "Create comprehensive testing strategy covering unit, integration, and end-to-end testing approaches",
				Type:        types.TaskTypeTesting,
				Priority:    types.TaskPriorityHigh,
				EstimatedEffort: types.EffortEstimate{
					Hours:            16.0,
					Days:             2.0,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"Test strategy document created",
					"Test coverage requirements defined",
					"Testing tools and frameworks selected",
					"Test data management strategy established",
					"Quality gates defined",
				},
				Tags: []string{"testing", "strategy", "quality", "planning"},
			},
			Confidence: 0.9,
			Priority:   SuggestionPriorityHigh,
			Category:   SuggestionCategoryQualityImprovement,
			Reasoning: []string{
				"Testing strategy needed before QA phase",
				"No testing tasks found",
				"Critical for quality assurance",
			},
			EstimatedValue: 0.85,
		})
	}

	return suggestions
}

// suggestTestingToDeploymentTasks suggests tasks for transitioning to deployment
func (s *Suggester) suggestTestingToDeploymentTasks(projectState types.ProjectState, existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	// Suggest deployment strategy
	if !s.hasTaskType(existingTasks, types.TaskTypeDeployment) {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("deployment_strategy"),
				Title:       "Define Deployment Strategy and Release Process",
				Description: "Plan deployment approach, rollback procedures, and release management processes",
				Type:        types.TaskTypeDeployment,
				Priority:    types.TaskPriorityHigh,
				EstimatedEffort: types.EffortEstimate{
					Hours:            12.0,
					Days:             1.5,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"Deployment strategy documented",
					"Rollback procedures defined",
					"Release checklist created",
					"Monitoring and alerting setup",
					"Post-deployment verification plan",
				},
				Tags: []string{"deployment", "release", "devops", "strategy"},
			},
			Confidence: 0.85,
			Priority:   SuggestionPriorityHigh,
			Category:   SuggestionCategoryPhaseProgression,
			Reasoning: []string{
				"Deployment strategy needed before release",
				"No deployment tasks found",
				"Risk mitigation for production release",
			},
			EstimatedValue: 0.8,
		})
	}

	return suggestions
}

// suggestDeploymentToMaintenanceTasks suggests tasks for transitioning to maintenance
func (s *Suggester) suggestDeploymentToMaintenanceTasks(projectState types.ProjectState, existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	// Suggest monitoring and observability
	if !s.hasKeyword(existingTasks, "monitoring") && !s.hasKeyword(existingTasks, "observability") {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("monitoring_setup"),
				Title:       "Setup Production Monitoring and Observability",
				Description: "Implement comprehensive monitoring, logging, and alerting for production systems",
				Type:        types.TaskTypeImplementation,
				Priority:    types.TaskPriorityCritical,
				EstimatedEffort: types.EffortEstimate{
					Hours:            24.0,
					Days:             3.0,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"Application performance monitoring implemented",
					"Infrastructure monitoring configured",
					"Log aggregation and analysis setup",
					"Alert thresholds and escalation defined",
					"Dashboard for key metrics created",
				},
				Tags: []string{"monitoring", "observability", "production", "alerts"},
			},
			Confidence: 0.9,
			Priority:   SuggestionPriorityImmediate,
			Category:   SuggestionCategoryRiskMitigation,
			Reasoning: []string{
				"Production monitoring critical for maintenance",
				"No monitoring tasks found",
				"Early detection of production issues",
			},
			EstimatedValue: 0.9,
		})
	}

	return suggestions
}

// suggestBottleneckResolutionTasks suggests tasks to resolve project bottlenecks
func (s *Suggester) suggestBottleneckResolutionTasks(projectState types.ProjectState, existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	// Analyze bottlenecks from project state
	for _, bottleneck := range projectState.CurrentBottlenecks {
		bottleneckLower := strings.ToLower(bottleneck)

		if strings.Contains(bottleneckLower, "testing") {
			suggestions = append(suggestions, TaskSuggestion{
				Task: types.Task{
					ID:          s.generateTaskID("testing_bottleneck"),
					Title:       "Resolve Testing Bottleneck",
					Description: fmt.Sprintf("Address testing bottleneck: %s", bottleneck),
					Type:        types.TaskTypeTesting,
					Priority:    types.TaskPriorityHigh,
					EstimatedEffort: types.EffortEstimate{
						Hours:            8.0,
						Days:             1.0,
						EstimationMethod: "contextual_suggestion",
					},
					AcceptanceCriteria: []string{
						"Bottleneck cause identified",
						"Resolution plan implemented",
						"Testing process improved",
						"Future prevention measures in place",
					},
					Tags: []string{"bottleneck", "testing", "process-improvement"},
				},
				Confidence: 0.8,
				Priority:   SuggestionPriorityHigh,
				Category:   SuggestionCategoryBottleneckResolution,
				Reasoning: []string{
					"Testing bottleneck identified in project state",
					"Immediate resolution needed for progress",
				},
				EstimatedValue: 0.9,
			})
		}

		if strings.Contains(bottleneckLower, "review") {
			suggestions = append(suggestions, TaskSuggestion{
				Task: types.Task{
					ID:          s.generateTaskID("review_bottleneck"),
					Title:       "Streamline Code Review Process",
					Description: fmt.Sprintf("Address review bottleneck: %s", bottleneck),
					Type:        types.TaskTypeReview,
					Priority:    types.TaskPriorityMedium,
					EstimatedEffort: types.EffortEstimate{
						Hours:            4.0,
						Days:             0.5,
						EstimationMethod: "contextual_suggestion",
					},
					AcceptanceCriteria: []string{
						"Review process analyzed",
						"Process improvements implemented",
						"Review turnaround time reduced",
						"Team guidelines updated",
					},
					Tags: []string{"bottleneck", "review", "process"},
				},
				Confidence: 0.75,
				Priority:   SuggestionPriorityMedium,
				Category:   SuggestionCategoryBottleneckResolution,
				Reasoning: []string{
					"Code review bottleneck affecting development velocity",
					"Process improvement needed",
				},
				EstimatedValue: 0.7,
			})
		}
	}

	return suggestions
}

// suggestQualityImprovementTasks suggests tasks to improve overall quality
func (s *Suggester) suggestQualityImprovementTasks(existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	// Count task types to identify gaps
	typeCount := make(map[types.TaskType]int)
	for _, task := range existingTasks {
		typeCount[task.Type]++
	}

	totalTasks := len(existingTasks)
	if totalTasks == 0 {
		return suggestions
	}

	// Suggest testing if underrepresented
	testingRatio := float64(typeCount[types.TaskTypeTesting]) / float64(totalTasks)
	if testingRatio < 0.2 {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("add_testing"),
				Title:       "Increase Test Coverage",
				Description: "Add comprehensive testing to improve code quality and reduce bugs",
				Type:        types.TaskTypeTesting,
				Priority:    types.TaskPriorityMedium,
				EstimatedEffort: types.EffortEstimate{
					Hours:            16.0,
					Days:             2.0,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"Unit test coverage increased to 80%+",
					"Integration tests added for key workflows",
					"Test automation implemented",
					"Quality gates enforced in CI/CD",
				},
				Tags: []string{"testing", "quality", "coverage", "automation"},
			},
			Confidence: 0.8,
			Priority:   SuggestionPriorityMedium,
			Category:   SuggestionCategoryQualityImprovement,
			Reasoning: []string{
				fmt.Sprintf("Testing represents only %.1f%% of tasks", testingRatio*100),
				"Insufficient test coverage detected",
				"Quality improvement opportunity",
			},
			EstimatedValue: 0.75,
		})
	}

	// Suggest documentation if underrepresented
	docRatio := float64(typeCount[types.TaskTypeDocumentation]) / float64(totalTasks)
	if docRatio < 0.1 {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("add_documentation"),
				Title:       "Improve Project Documentation",
				Description: "Create comprehensive documentation to improve maintainability and onboarding",
				Type:        types.TaskTypeDocumentation,
				Priority:    types.TaskPriorityLow,
				EstimatedEffort: types.EffortEstimate{
					Hours:            8.0,
					Days:             1.0,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"API documentation updated",
					"Developer setup guide created",
					"Architecture documentation completed",
					"Code comments improved",
				},
				Tags: []string{"documentation", "maintainability", "onboarding"},
			},
			Confidence: 0.7,
			Priority:   SuggestionPriorityLow,
			Category:   SuggestionCategoryQualityImprovement,
			Reasoning: []string{
				fmt.Sprintf("Documentation represents only %.1f%% of tasks", docRatio*100),
				"Documentation gap affects maintainability",
			},
			EstimatedValue: 0.6,
		})
	}

	return suggestions
}

// suggestRiskMitigationTasks suggests tasks to mitigate project risks
func (s *Suggester) suggestRiskMitigationTasks(projectState types.ProjectState, existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	// Analyze technical challenges and suggest mitigation
	for _, challenge := range projectState.TechnicalChallenges {
		challengeLower := strings.ToLower(challenge)

		if strings.Contains(challengeLower, "performance") {
			suggestions = append(suggestions, TaskSuggestion{
				Task: types.Task{
					ID:          s.generateTaskID("performance_risk"),
					Title:       "Address Performance Risk",
					Description: fmt.Sprintf("Mitigate performance challenge: %s", challenge),
					Type:        types.TaskTypeAnalysis,
					Priority:    types.TaskPriorityHigh,
					EstimatedEffort: types.EffortEstimate{
						Hours:            12.0,
						Days:             1.5,
						EstimationMethod: "contextual_suggestion",
					},
					AcceptanceCriteria: []string{
						"Performance bottlenecks identified",
						"Optimization strategy defined",
						"Performance benchmarks established",
						"Monitoring implemented",
					},
					Tags: []string{"performance", "risk", "optimization"},
				},
				Confidence: 0.8,
				Priority:   SuggestionPriorityHigh,
				Category:   SuggestionCategoryRiskMitigation,
				Reasoning: []string{
					"Performance challenge identified as technical risk",
					"Early mitigation reduces project risk",
				},
				EstimatedValue: 0.8,
			})
		}

		if strings.Contains(challengeLower, "security") {
			suggestions = append(suggestions, TaskSuggestion{
				Task: types.Task{
					ID:          s.generateTaskID("security_risk"),
					Title:       "Address Security Risk",
					Description: fmt.Sprintf("Mitigate security challenge: %s", challenge),
					Type:        types.TaskTypeAnalysis,
					Priority:    types.TaskPriorityCritical,
					EstimatedEffort: types.EffortEstimate{
						Hours:            16.0,
						Days:             2.0,
						EstimationMethod: "contextual_suggestion",
					},
					AcceptanceCriteria: []string{
						"Security threats assessed",
						"Security controls implemented",
						"Security testing performed",
						"Security documentation updated",
					},
					Tags: []string{"security", "risk", "compliance"},
				},
				Confidence: 0.9,
				Priority:   SuggestionPriorityImmediate,
				Category:   SuggestionCategoryRiskMitigation,
				Reasoning: []string{
					"Security challenge poses significant risk",
					"Critical for system safety and compliance",
				},
				EstimatedValue: 0.95,
			})
		}
	}

	return suggestions
}

// suggestDependencySetupTasks suggests tasks to setup dependencies
func (s *Suggester) suggestDependencySetupTasks(existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	// Analyze dependencies across tasks
	dependencyMap := make(map[string]int)
	for _, task := range existingTasks {
		for _, dep := range task.Dependencies {
			dependencyMap[dep]++
		}
	}

	// Find tasks that are heavily depended upon but might need prerequisites
	for _, task := range existingTasks {
		if dependencyMap[task.ID] > 2 { // Task is a dependency for multiple other tasks
			if task.Type == types.TaskTypeImplementation && !s.hasRelatedTask(existingTasks, task.ID, types.TaskTypeDesign) {
				suggestions = append(suggestions, TaskSuggestion{
					Task: types.Task{
						ID:          s.generateTaskID("design_prerequisite"),
						Title:       fmt.Sprintf("Design for %s", task.Title),
						Description: fmt.Sprintf("Create design and specifications before implementing %s", task.Title),
						Type:        types.TaskTypeDesign,
						Priority:    types.TaskPriorityHigh,
						EstimatedEffort: types.EffortEstimate{
							Hours:            6.0,
							Days:             0.75,
							EstimationMethod: "contextual_suggestion",
						},
						AcceptanceCriteria: []string{
							"Design specifications created",
							"Technical approach documented",
							"Interface contracts defined",
							"Design review completed",
						},
						Tags: []string{"design", "prerequisite", "planning"},
					},
					Confidence: 0.75,
					Priority:   SuggestionPriorityHigh,
					Category:   SuggestionCategoryDependencySetup,
					Reasoning: []string{
						"Implementation task has multiple dependencies",
						"Design phase missing for critical component",
						"Proper design reduces implementation risk",
					},
					Prerequisites:  []string{},
					BasedOn:        []string{task.ID},
					EstimatedValue: 0.7,
				})
			}
		}
	}

	return suggestions
}

// suggestArchitecturalTasks suggests architectural improvement tasks
func (s *Suggester) suggestArchitecturalTasks(projectState types.ProjectState, existingTasks []types.Task, genContext types.TaskGenerationContext) []TaskSuggestion {
	suggestions := []TaskSuggestion{}

	// Check if architecture documentation exists
	if !s.hasTaskType(existingTasks, types.TaskTypeArchitecture) && len(existingTasks) > 5 {
		suggestions = append(suggestions, TaskSuggestion{
			Task: types.Task{
				ID:          s.generateTaskID("architecture_documentation"),
				Title:       "Document System Architecture",
				Description: "Create comprehensive architectural documentation for the system",
				Type:        types.TaskTypeArchitecture,
				Priority:    types.TaskPriorityMedium,
				EstimatedEffort: types.EffortEstimate{
					Hours:            12.0,
					Days:             1.5,
					EstimationMethod: "contextual_suggestion",
				},
				AcceptanceCriteria: []string{
					"Architecture overview diagram created",
					"Component interactions documented",
					"Technology decisions recorded",
					"Scalability considerations addressed",
				},
				Tags: []string{"architecture", "documentation", "system-design"},
			},
			Confidence: 0.7,
			Priority:   SuggestionPriorityMedium,
			Category:   SuggestionCategoryArchitectural,
			Reasoning: []string{
				"Project has multiple implementation tasks but no architecture documentation",
				"Architecture documentation improves maintainability",
			},
			EstimatedValue: 0.6,
		})
	}

	return suggestions
}

// Helper functions

// hasTaskType checks if any existing task has the specified type
func (s *Suggester) hasTaskType(tasks []types.Task, taskType types.TaskType) bool {
	for _, task := range tasks {
		if task.Type == taskType {
			return true
		}
	}
	return false
}

// hasKeyword checks if any task title or description contains the keyword
func (s *Suggester) hasKeyword(tasks []types.Task, keyword string) bool {
	keywordLower := strings.ToLower(keyword)
	for _, task := range tasks {
		content := strings.ToLower(task.Title + " " + task.Description)
		if strings.Contains(content, keywordLower) {
			return true
		}
	}
	return false
}

// hasRelatedTask checks if there's a task of given type related to the specified task ID
func (s *Suggester) hasRelatedTask(tasks []types.Task, taskID string, taskType types.TaskType) bool {
	for _, task := range tasks {
		if task.Type == taskType {
			// Check if task references the given task ID
			for _, dep := range task.Dependencies {
				if dep == taskID {
					return true
				}
			}
			// Check if given task depends on this task
			for _, otherTask := range tasks {
				if otherTask.ID == taskID {
					for _, dep := range otherTask.Dependencies {
						if dep == task.ID {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// generateTaskID generates a unique task ID
func (s *Suggester) generateTaskID(prefix string) string {
	return fmt.Sprintf("suggestion_%s_%d", prefix, time.Now().UnixNano())
}

// filterAndRankSuggestions filters and ranks suggestions by confidence and priority
func (s *Suggester) filterAndRankSuggestions(suggestions []TaskSuggestion) []TaskSuggestion {
	// Filter by confidence threshold
	filtered := []TaskSuggestion{}
	for _, suggestion := range suggestions {
		if suggestion.Confidence >= s.config.MinConfidenceScore {
			filtered = append(filtered, suggestion)
		}
	}

	// Sort by priority and confidence
	sort.Slice(filtered, func(i, j int) bool {
		// Primary sort by priority
		priorityOrder := map[SuggestionPriority]int{
			SuggestionPriorityImmediate: 5,
			SuggestionPriorityHigh:      4,
			SuggestionPriorityMedium:    3,
			SuggestionPriorityLow:       2,
			SuggestionPriorityFuture:    1,
		}

		iPriority := priorityOrder[filtered[i].Priority]
		jPriority := priorityOrder[filtered[j].Priority]

		if iPriority != jPriority {
			return iPriority > jPriority
		}

		// Secondary sort by confidence
		if filtered[i].Confidence != filtered[j].Confidence {
			return filtered[i].Confidence > filtered[j].Confidence
		}

		// Tertiary sort by estimated value
		return filtered[i].EstimatedValue > filtered[j].EstimatedValue
	})

	// Limit results
	if len(filtered) > s.config.MaxSuggestions {
		filtered = filtered[:s.config.MaxSuggestions]
	}

	return filtered
}
