package intelligence

import "time"

// StepDefinition represents a pattern step definition
type StepDefinition struct {
	Action      string
	Description string
	Context     map[string]any
	Optional    bool
	Confidence  float64
}

// createPattern creates a Pattern with the given steps
func createPattern(
	id string,
	patternType PatternType,
	name, description string,
	keywords, triggers, outcomes []string,
	steps []StepDefinition,
	context map[string]any,
	relatedPatterns []string,
	confidence, frequency float64,
	successRate float64,
) Pattern {
	patternSteps := make([]PatternStep, len(steps))
	for i, step := range steps {
		patternSteps[i] = PatternStep{
			Order:       i,
			Action:      step.Action,
			Description: step.Description,
			Optional:    step.Optional,
			Confidence:  step.Confidence,
			Context:     step.Context,
		}
	}
	
	return Pattern{
		ID:              id,
		Type:            patternType,
		Name:            name,
		Description:     description,
		Confidence:      confidence,
		Frequency:       int(frequency),
		SuccessRate:     successRate,
		Keywords:        keywords,
		Triggers:        triggers,
		Outcomes:        outcomes,
		Steps:           patternSteps,
		Context:         context,
		RelatedPatterns: relatedPatterns,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// createFourStepPattern creates a pattern with a standard four-step workflow
func createFourStepPattern(
	id string,
	patternType PatternType,
	name, description string,
	keywords, triggers, outcomes []string,
	steps [4]struct {
		Action      string
		Description string
		Context     map[string]any
		Optional    bool
		Confidence  float64
	},
	context map[string]any,
	relatedPatterns []string,
	confidence, frequency float64,
	successRate float64,
) Pattern {
	stepDefs := make([]StepDefinition, 4)
	for i, step := range steps {
		stepDefs[i] = StepDefinition{
			Action:      step.Action,
			Description: step.Description,
			Context:     step.Context,
			Optional:    step.Optional,
			Confidence:  step.Confidence,
		}
	}
	
	return createPattern(
		id, patternType, name, description,
		keywords, triggers, outcomes,
		stepDefs, context, relatedPatterns,
		confidence, frequency, successRate,
	)
}

// getBuiltInPatterns returns a set of predefined patterns for common conversation flows
//
//nolint:dupl // Pattern definitions have similar structure but unique parameters
func getBuiltInPatterns() []Pattern {
	return []Pattern{
		{
			ID:          "builtin_problem_solution",
			Type:        PatternTypeProblemSolution,
			Name:        "Basic Problem-Solution Pattern",
			Description: "User reports a problem, assistant provides solution, user confirms it works",
			Confidence:  0.8,
			Frequency:   100,
			SuccessRate: 0.85,
			Keywords:    []string{"problem", "error", "issue", "fix", "solution", "works"},
			Triggers:    []string{"error_reported", "help_requested"},
			Outcomes:    []string{"problem_resolved", "solution_implemented"},
			Steps: []PatternStep{
				{
					Order:       0,
					Action:      "report_problem",
					Description: "User reports a problem or error",
					Optional:    false,
					Confidence:  0.9,
					Context:     map[string]any{"typical_type": "problem"},
				},
				{
					Order:       1,
					Action:      "analyze_problem", 
					Description: "Assistant analyzes the reported problem",
					Optional:    true,
					Confidence:  0.8,
					Context:     map[string]any{"typical_type": "solution"},
				},
				{
					Order:       2,
					Action:      "propose_solution",
					Description: "Assistant proposes a solution",
					Optional:    false,
					Confidence:  0.9,
					Context:     map[string]any{"typical_type": "solution"},
				},
				{
					Order:       3,
					Action:      "verify_solution",
					Description: "User tests and verifies the solution",
					Optional:    false,
					Confidence:  0.8,
					Context:     map[string]any{"typical_type": "verification"},
				},
			},
			Context: map[string]any{
				"typical_length":     4,
				"requires_technical": true,
				"success_indicators": []string{"works", "fixed", "resolved"},
			},
			RelatedPatterns: []string{"debugging_pattern", "error_resolution_pattern"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		createPattern(
			"builtin_debugging_flow",
			PatternTypeDebugging,
			"Systematic Debugging Pattern",
			"Systematic approach to debugging: identify error, investigate, test fixes",
			[]string{"debug", "error", "stack", "trace", "investigate", "test"},
			[]string{"error_with_stack_trace", "debugging_needed"},
			[]string{"root_cause_found", "error_fixed"},
			[]StepDefinition{
				{
					Action:      "identify_error",
					Description: "Identify the specific error or bug",
					Context:     map[string]any{"includes_stack_trace": true},
					Optional:    false,
					Confidence:  0.9,
				},
				{
					Action:      "investigate_cause",
					Description: "Investigate the root cause of the error",
					Context:     map[string]any{"requires_analysis": true},
					Optional:    false,
					Confidence:  0.8,
				},
				{
					Action:      "test_hypothesis",
					Description: "Test hypotheses about the cause",
					Context:     map[string]any{"experimental": true},
					Optional:    true,
					Confidence:  0.7,
				},
				{
					Action:      "apply_fix",
					Description: "Apply the identified fix",
					Context:     map[string]any{"code_modification": true},
					Optional:    false,
					Confidence:  0.8,
				},
				{
					Action:      "confirm_resolution",
					Description: "Confirm the error is resolved",
					Context:     map[string]any{"verification_required": true},
					Optional:    false,
					Confidence:  0.9,
				},
			},
			map[string]any{
				"typical_length":     5,
				"requires_technical": true,
				"complexity":         "high",
				"time_intensive":     true,
			},
			[]string{"problem_solution_pattern", "error_resolution_pattern"},
			0.8, 75, 0.78,
		),
		createFourStepPattern(
			"builtin_code_review",
			PatternTypeCodeEvolution,
			"Code Review and Improvement Pattern",
			"Review code, identify improvements, implement changes, verify quality",
			[]string{"code", "review", "improve", "refactor", "optimize", "quality"},
			[]string{"code_review_requested", "improvement_needed"},
			[]string{"code_improved", "quality_enhanced"},
			[4]struct {
				Action      string
				Description string
				Context     map[string]any
				Optional    bool
				Confidence  float64
			}{
				{
					Action:      "review_code",
					Description: "Review existing code structure and quality",
					Context:     map[string]any{"analysis_required": true},
					Optional:    false,
					Confidence:  0.8,
				},
				{
					Action:      "identify_improvements",
					Description: "Identify areas for improvement",
					Context:     map[string]any{"expertise_required": true},
					Optional:    false,
					Confidence:  0.8,
				},
				{
					Action:      "implement_changes",
					Description: "Implement the suggested improvements",
					Context:     map[string]any{"code_modification": true},
					Optional:    false,
					Confidence:  0.7,
				},
				{
					Action:      "verify_quality",
					Description: "Verify code quality and functionality",
					Context:     map[string]any{"testing_required": true, "quality_check": "comprehensive"},
					Optional:    false,
					Confidence:  0.8,
				},
			},
			map[string]any{
				"typical_length":   4,
				"involves_code":    true,
				"quality_focused":  true,
				"iterative":        true,
				"review_type":     "comprehensive",
				"improvement_goal": "code_quality",
			},
			[]string{"refactoring_pattern", "optimization_pattern"},
			0.75, 50, 0.82,
		),
		createFourStepPattern(
			"builtin_feature_development",
			PatternTypeWorkflow,
			"Feature Development Workflow",
			"Plan feature, implement code, test functionality, deploy/integrate",
			[]string{"feature", "implement", "develop", "test", "deploy", "integration"},
			[]string{"feature_requested", "new_functionality_needed"},
			[]string{"feature_completed", "functionality_added"},
			[4]struct {
				Action      string
				Description string
				Context     map[string]any
				Optional    bool
				Confidence  float64
			}{
				{
					Action:      "plan_feature",
					Description: "Plan and design the new feature",
					Context:     map[string]any{"design_phase": true},
					Optional:    false,
					Confidence:  0.8,
				},
				{
					Action:      "implement_code",
					Description: "Implement the feature code",
					Context:     map[string]any{"development_phase": true},
					Optional:    false,
					Confidence:  0.8,
				},
				{
					Action:      "test_functionality",
					Description: "Test the new functionality",
					Context:     map[string]any{"testing_phase": true},
					Optional:    false,
					Confidence:  0.9,
				},
				{
					Action:      "integrate_deploy",
					Description: "Integrate and deploy the feature",
					Context:     map[string]any{"deployment_phase": true, "feature_type": "new"},
					Optional:    true,
					Confidence:  0.7,
				},
			},
			map[string]any{
				"typical_length":   4,
				"involves_code":    true,
				"end_to_end":      true,
				"deliverable":     true,
				"development_type": "feature",
				"lifecycle_phase": "complete",
			},
			[]string{"development_workflow", "testing_pattern"},
			0.8, 60, 0.80,
		),
		createPattern(
			"builtin_configuration_setup",
			PatternTypeConfiguration,
			"Configuration and Setup Pattern",
			"Identify config needs, locate files, make changes, verify settings",
			[]string{"config", "configuration", "setup", "settings", "environment"},
			[]string{"configuration_needed", "setup_required"},
			[]string{"configuration_complete", "environment_ready"},
			[]StepDefinition{
				{
					Action:      "identify_config_needs",
					Description: "Identify what needs to be configured",
					Context:     map[string]any{"analysis_required": true},
					Optional:    false,
					Confidence:  0.8,
				},
				{
					Action:      "locate_config_files",
					Description: "Locate relevant configuration files",
					Context:     map[string]any{"file_system_navigation": true},
					Optional:    false,
					Confidence:  0.8,
				},
				{
					Action:      "modify_configuration",
					Description: "Modify configuration settings",
					Context:     map[string]any{"file_modification": true},
					Optional:    false,
					Confidence:  0.8,
				},
				{
					Action:      "verify_settings",
					Description: "Verify configuration is working",
					Context:     map[string]any{"verification_required": true},
					Optional:    false,
					Confidence:  0.9,
				},
			},
			map[string]any{
				"typical_length":   4,
				"involves_files":   true,
				"system_level":     true,
				"environment_setup": true,
			},
			[]string{"setup_pattern", "deployment_pattern"},
			0.75, 40, 0.85,
		),
		createFourStepPattern(
			"builtin_learning_exploration",
			PatternTypeWorkflow,
			"Learning and Exploration Pattern",
			"Ask questions, explore concepts, get explanations, apply knowledge",
			[]string{"learn", "understand", "explain", "concept", "how", "why"},
			[]string{"learning_question", "explanation_requested"},
			[]string{"understanding_gained", "knowledge_applied"},
			[4]struct {
				Action      string
				Description string
				Context     map[string]any
				Optional    bool
				Confidence  float64
			}{
				{
					Action:      "ask_question",
					Description: "Ask about a concept or topic",
					Context:     map[string]any{"information_seeking": true},
					Optional:    false,
					Confidence:  0.9,
				},
				{
					Action:      "provide_explanation",
					Description: "Provide detailed explanation",
					Context:     map[string]any{"educational": true},
					Optional:    false,
					Confidence:  0.8,
				},
				{
					Action:      "explore_examples",
					Description: "Explore practical examples",
					Context:     map[string]any{"practical_application": true},
					Optional:    true,
					Confidence:  0.7,
				},
				{
					Action:      "apply_knowledge",
					Description: "Apply the newly learned concepts",
					Context:     map[string]any{"hands_on_practice": true, "learning_stage": "application"},
					Optional:    true,
					Confidence:  0.7,
				},
			},
			map[string]any{
				"typical_length":   3,
				"educational":      true,
				"knowledge_transfer": true,
				"interactive":      true,
				"learning_pattern": "exploratory",
				"knowledge_depth":  "conceptual",
			},
			[]string{"explanation_pattern", "tutorial_pattern"},
			0.7, 30, 0.88,
		),
	}
}