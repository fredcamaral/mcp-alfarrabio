package intelligence

import "time"

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
	patternSteps := make([]PatternStep, 4)
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
		Frequency:       frequency,
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

// getBuiltInPatterns returns a set of predefined patterns for common conversation flows
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
		{
			ID:          "builtin_debugging_flow",
			Type:        PatternTypeDebugging,
			Name:        "Systematic Debugging Pattern",
			Description: "Systematic approach to debugging: identify error, investigate, test fixes",
			Confidence:  0.8,
			Frequency:   75,
			SuccessRate: 0.78,
			Keywords:    []string{"debug", "error", "stack", "trace", "investigate", "test"},
			Triggers:    []string{"error_with_stack_trace", "debugging_needed"},
			Outcomes:    []string{"root_cause_found", "error_fixed"},
			Steps: []PatternStep{
				{
					Order:       0,
					Action:      "identify_error",
					Description: "Identify the specific error or bug",
					Optional:    false,
					Confidence:  0.9,
					Context:     map[string]any{"includes_stack_trace": true},
				},
				{
					Order:       1,
					Action:      "investigate_cause",
					Description: "Investigate the root cause of the error",
					Optional:    false,
					Confidence:  0.8,
					Context:     map[string]any{"requires_analysis": true},
				},
				{
					Order:       2,
					Action:      "test_hypothesis",
					Description: "Test hypotheses about the cause",
					Optional:    true,
					Confidence:  0.7,
					Context:     map[string]any{"experimental": true},
				},
				{
					Order:       3,
					Action:      "apply_fix",
					Description: "Apply the identified fix",
					Optional:    false,
					Confidence:  0.8,
					Context:     map[string]any{"code_modification": true},
				},
				{
					Order:       4,
					Action:      "confirm_resolution",
					Description: "Confirm the error is resolved",
					Optional:    false,
					Confidence:  0.9,
					Context:     map[string]any{"verification_required": true},
				},
			},
			Context: map[string]any{
				"typical_length":     5,
				"requires_technical": true,
				"complexity":         "high",
				"time_intensive":     true,
			},
			RelatedPatterns: []string{"problem_solution_pattern", "error_resolution_pattern"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
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
		{
			ID:          "builtin_configuration_setup",
			Type:        PatternTypeConfiguration,
			Name:        "Configuration and Setup Pattern",
			Description: "Identify config needs, locate files, make changes, verify settings",
			Confidence:  0.75,
			Frequency:   40,
			SuccessRate: 0.85,
			Keywords:    []string{"config", "configuration", "setup", "settings", "environment"},
			Triggers:    []string{"configuration_needed", "setup_required"},
			Outcomes:    []string{"configuration_complete", "environment_ready"},
			Steps: []PatternStep{
				{
					Order:       0,
					Action:      "identify_config_needs",
					Description: "Identify what needs to be configured",
					Optional:    false,
					Confidence:  0.8,
					Context:     map[string]any{"analysis_required": true},
				},
				{
					Order:       1,
					Action:      "locate_config_files",
					Description: "Locate relevant configuration files",
					Optional:    false,
					Confidence:  0.8,
					Context:     map[string]any{"file_system_navigation": true},
				},
				{
					Order:       2,
					Action:      "modify_configuration",
					Description: "Modify configuration settings",
					Optional:    false,
					Confidence:  0.8,
					Context:     map[string]any{"file_modification": true},
				},
				{
					Order:       3,
					Action:      "verify_settings",
					Description: "Verify configuration is working",
					Optional:    false,
					Confidence:  0.9,
					Context:     map[string]any{"verification_required": true},
				},
			},
			Context: map[string]any{
				"typical_length":   4,
				"involves_files":   true,
				"system_level":     true,
				"environment_setup": true,
			},
			RelatedPatterns: []string{"setup_pattern", "deployment_pattern"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
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