// Package tasks provides task workflow and status transition management.
package tasks

import (
	"fmt"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// WorkflowManager handles task status transitions and workflow validation
type WorkflowManager struct {
	transitions map[types.TaskStatus][]types.TaskStatus
	rules       map[string]WorkflowRule
	config      WorkflowConfig
}

// WorkflowConfig represents workflow configuration
type WorkflowConfig struct {
	StrictTransitions    bool `json:"strict_transitions"`
	RequireComments      bool `json:"require_comments"`
	AutoAssignment       bool `json:"auto_assignment"`
	NotifyOnTransition   bool `json:"notify_on_transition"`
	MaxTransitionsPerDay int  `json:"max_transitions_per_day"`
}

// WorkflowRule represents a rule for workflow transitions
type WorkflowRule struct {
	Name         string              `json:"name"`
	FromStatus   types.TaskStatus    `json:"from_status"`
	ToStatus     types.TaskStatus    `json:"to_status"`
	RequiredRole []string            `json:"required_role"`
	Conditions   []WorkflowCondition `json:"conditions"`
	Actions      []WorkflowAction    `json:"actions"`
}

// WorkflowCondition represents a condition for workflow transitions
type WorkflowCondition struct {
	Type  string      `json:"type"`
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

// WorkflowAction represents an action to perform on workflow transition
type WorkflowAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
}

// TransitionResult represents the result of a workflow transition
type TransitionResult struct {
	Allowed    bool     `json:"allowed"`
	Reason     string   `json:"reason,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	ActionsRun []string `json:"actions_run,omitempty"`
}

// DefaultWorkflowConfig returns default workflow configuration
func DefaultWorkflowConfig() WorkflowConfig {
	return WorkflowConfig{
		StrictTransitions:    true,
		RequireComments:      false,
		AutoAssignment:       false,
		NotifyOnTransition:   true,
		MaxTransitionsPerDay: 50,
	}
}

// NewWorkflowManager creates a new workflow manager
func NewWorkflowManager() *WorkflowManager {
	wm := &WorkflowManager{
		transitions: make(map[types.TaskStatus][]types.TaskStatus),
		rules:       make(map[string]WorkflowRule),
		config:      DefaultWorkflowConfig(),
	}

	wm.setupDefaultTransitions()
	wm.setupDefaultRules()

	return wm
}

// NewWorkflowManagerWithConfig creates a workflow manager with custom config
func NewWorkflowManagerWithConfig(config WorkflowConfig) *WorkflowManager {
	wm := &WorkflowManager{
		transitions: make(map[types.TaskStatus][]types.TaskStatus),
		rules:       make(map[string]WorkflowRule),
		config:      config,
	}

	wm.setupDefaultTransitions()
	wm.setupDefaultRules()

	return wm
}

// ValidateTransition validates if a status transition is allowed
func (wm *WorkflowManager) ValidateTransition(fromStatus, toStatus types.TaskStatus, userID string) error {
	// Allow creation (empty fromStatus)
	if fromStatus == "" {
		if toStatus == types.TaskStatusTodo || toStatus == types.TaskStatusInProgress {
			return nil
		}
		return fmt.Errorf("new tasks can only be created as 'todo' or 'in_progress', got '%s'", toStatus)
	}

	// Same status is always allowed
	if fromStatus == toStatus {
		return nil
	}

	// Check if transition is allowed
	if !wm.config.StrictTransitions {
		return nil // Allow all transitions if not strict
	}

	allowedTransitions, exists := wm.transitions[fromStatus]
	if !exists {
		return fmt.Errorf("no transitions defined for status '%s'", fromStatus)
	}

	// Check if target status is in allowed transitions
	for _, allowed := range allowedTransitions {
		if allowed == toStatus {
			// Check workflow rules
			ruleKey := fmt.Sprintf("%s_to_%s", fromStatus, toStatus)
			if rule, hasRule := wm.rules[ruleKey]; hasRule {
				return wm.validateRule(rule, userID)
			}
			return nil // Transition allowed
		}
	}

	return fmt.Errorf("transition from '%s' to '%s' is not allowed", fromStatus, toStatus)
}

// GetAllowedTransitions returns allowed transitions for a given status
func (wm *WorkflowManager) GetAllowedTransitions(status types.TaskStatus) []types.TaskStatus {
	transitions, exists := wm.transitions[status]
	if !exists {
		return []types.TaskStatus{}
	}

	// Return a copy to prevent modification
	result := make([]types.TaskStatus, len(transitions))
	copy(result, transitions)
	return result
}

// AddTransition adds a new transition rule
func (wm *WorkflowManager) AddTransition(from, to types.TaskStatus) {
	if wm.transitions[from] == nil {
		wm.transitions[from] = make([]types.TaskStatus, 0)
	}

	// Check if transition already exists
	for _, existing := range wm.transitions[from] {
		if existing == to {
			return // Already exists
		}
	}

	wm.transitions[from] = append(wm.transitions[from], to)
}

// RemoveTransition removes a transition rule
func (wm *WorkflowManager) RemoveTransition(from, to types.TaskStatus) {
	transitions, exists := wm.transitions[from]
	if !exists {
		return
	}

	for i, transition := range transitions {
		if transition == to {
			wm.transitions[from] = append(transitions[:i], transitions[i+1:]...)
			break
		}
	}
}

// AddRule adds a workflow rule
func (wm *WorkflowManager) AddRule(rule WorkflowRule) {
	ruleKey := fmt.Sprintf("%s_to_%s", rule.FromStatus, rule.ToStatus)
	wm.rules[ruleKey] = rule
}

// ProcessTransition processes a workflow transition with all checks and actions
func (wm *WorkflowManager) ProcessTransition(fromStatus, toStatus types.TaskStatus, userID string, task *types.Task) (*TransitionResult, error) {
	result := &TransitionResult{
		Allowed:    false,
		ActionsRun: make([]string, 0),
		Warnings:   make([]string, 0),
	}

	// Validate transition
	if err := wm.ValidateTransition(fromStatus, toStatus, userID); err != nil {
		result.Reason = err.Error()
		return result, nil
	}

	// Check and execute workflow rules
	ruleKey := fmt.Sprintf("%s_to_%s", fromStatus, toStatus)
	if rule, hasRule := wm.rules[ruleKey]; hasRule {
		if err := wm.executeRule(rule, userID, task, result); err != nil {
			result.Reason = fmt.Sprintf("rule execution failed: %v", err)
			return result, nil
		}
	}

	// Add any automatic actions
	wm.executeAutomaticActions(fromStatus, toStatus, task, result)

	result.Allowed = true
	return result, nil
}

// GetWorkflowInfo returns comprehensive workflow information
func (wm *WorkflowManager) GetWorkflowInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// Transition map
	transitions := make(map[string][]string)
	for from, toList := range wm.transitions {
		toStrings := make([]string, len(toList))
		for i, to := range toList {
			toStrings[i] = string(to)
		}
		transitions[string(from)] = toStrings
	}

	info["transitions"] = transitions
	info["config"] = wm.config
	info["rules_count"] = len(wm.rules)

	return info
}

// Private methods

func (wm *WorkflowManager) setupDefaultTransitions() {
	// Todo transitions
	wm.AddTransition(types.TaskStatusTodo, types.TaskStatusInProgress)
	wm.AddTransition(types.TaskStatusTodo, types.TaskStatusBlocked)
	wm.AddTransition(types.TaskStatusTodo, types.TaskStatusCancelled)

	// In Progress transitions
	wm.AddTransition(types.TaskStatusInProgress, types.TaskStatusCompleted)
	wm.AddTransition(types.TaskStatusInProgress, types.TaskStatusBlocked)
	wm.AddTransition(types.TaskStatusInProgress, types.TaskStatusTodo)
	wm.AddTransition(types.TaskStatusInProgress, types.TaskStatusCancelled)

	// Blocked transitions
	wm.AddTransition(types.TaskStatusBlocked, types.TaskStatusTodo)
	wm.AddTransition(types.TaskStatusBlocked, types.TaskStatusInProgress)
	wm.AddTransition(types.TaskStatusBlocked, types.TaskStatusCancelled)

	// Completed transitions (limited)
	wm.AddTransition(types.TaskStatusCompleted, types.TaskStatusTodo) // Reopen

	// Cancelled transitions (limited)
	wm.AddTransition(types.TaskStatusCancelled, types.TaskStatusTodo) // Reopen
}

func (wm *WorkflowManager) setupDefaultRules() {
	// Rule: Starting work requires assignment
	wm.AddRule(WorkflowRule{
		Name:       "start_work_requires_assignment",
		FromStatus: types.TaskStatusTodo,
		ToStatus:   types.TaskStatusInProgress,
		Conditions: []WorkflowCondition{
			{
				Type:  "field_not_empty",
				Field: "assignee",
				Value: nil,
			},
		},
		Actions: []WorkflowAction{
			{
				Type: "set_started_timestamp",
				Parameters: map[string]interface{}{
					"field": "started_at",
				},
			},
		},
	})

	// Rule: Completing work requires acceptance criteria validation
	wm.AddRule(WorkflowRule{
		Name:       "complete_requires_acceptance_criteria",
		FromStatus: types.TaskStatusInProgress,
		ToStatus:   types.TaskStatusCompleted,
		Conditions: []WorkflowCondition{
			{
				Type:  "field_not_empty",
				Field: "acceptance_criteria",
				Value: nil,
			},
		},
		Actions: []WorkflowAction{
			{
				Type: "set_completed_timestamp",
				Parameters: map[string]interface{}{
					"field": "completed_at",
				},
			},
			{
				Type: "calculate_completion_time",
				Parameters: map[string]interface{}{
					"start_field": "started_at",
					"end_field":   "completed_at",
				},
			},
		},
	})
}

func (wm *WorkflowManager) validateRule(rule WorkflowRule, userID string) error {
	// Check required roles
	if len(rule.RequiredRole) > 0 {
		hasRole := false
		for _, role := range rule.RequiredRole {
			if wm.userHasRole(userID, role) {
				hasRole = true
				break
			}
		}
		if !hasRole {
			return fmt.Errorf("user %s does not have required role for this transition", userID)
		}
	}

	return nil
}

func (wm *WorkflowManager) executeRule(rule WorkflowRule, userID string, task *types.Task, result *TransitionResult) error {
	// Validate conditions
	for _, condition := range rule.Conditions {
		if err := wm.validateCondition(condition, task, result); err != nil {
			return err
		}
	}

	// Execute actions
	for _, action := range rule.Actions {
		if err := wm.executeAction(action, task); err != nil {
			return fmt.Errorf("failed to execute action %s: %w", action.Type, err)
		}
		result.ActionsRun = append(result.ActionsRun, action.Type)
	}

	return nil
}

func (wm *WorkflowManager) validateCondition(condition WorkflowCondition, task *types.Task, result *TransitionResult) error {
	switch condition.Type {
	case "field_not_empty":
		value := wm.getFieldValue(task, condition.Field)
		if value == "" {
			return fmt.Errorf("field '%s' cannot be empty", condition.Field)
		}
	case "field_equals":
		value := wm.getFieldValue(task, condition.Field)
		if value != condition.Value {
			return fmt.Errorf("field '%s' must equal '%v'", condition.Field, condition.Value)
		}
	case "has_acceptance_criteria":
		if len(task.AcceptanceCriteria) == 0 {
			result.Warnings = append(result.Warnings, "Task has no acceptance criteria defined")
		}
	}
	return nil
}

func (wm *WorkflowManager) executeAction(action WorkflowAction, task *types.Task) error {
	now := time.Now()

	switch action.Type {
	case "set_started_timestamp":
		task.Timestamps.Started = &now
	case "set_completed_timestamp":
		task.Timestamps.Completed = &now
	case "calculate_completion_time":
		if task.Timestamps.Started != nil {
			// Could calculate and store completion time here
			// For now, just ensure completed timestamp is set
			if task.Timestamps.Completed == nil {
				task.Timestamps.Completed = &now
			}
		}
	case "auto_assign":
		if assignee, ok := action.Parameters["assignee"].(string); ok && task.Assignee == "" {
			task.Assignee = assignee
		}
	case "add_tag":
		if tag, ok := action.Parameters["tag"].(string); ok {
			// Add tag if not already present
			for _, existingTag := range task.Tags {
				if existingTag == tag {
					return nil // Tag already exists
				}
			}
			task.Tags = append(task.Tags, tag)
		}
	}
	return nil
}

func (wm *WorkflowManager) executeAutomaticActions(fromStatus, toStatus types.TaskStatus, task *types.Task, result *TransitionResult) {
	// Add automatic timestamps
	now := time.Now()

	switch toStatus {
	case types.TaskStatusInProgress:
		if task.Timestamps.Started == nil {
			task.Timestamps.Started = &now
			result.ActionsRun = append(result.ActionsRun, "auto_set_started_timestamp")
		}
	case types.TaskStatusCompleted:
		if task.Timestamps.Completed == nil {
			task.Timestamps.Completed = &now
			result.ActionsRun = append(result.ActionsRun, "auto_set_completed_timestamp")
		}
	}

	// Update the updated timestamp
	task.Timestamps.Updated = now
}

func (wm *WorkflowManager) getFieldValue(task *types.Task, field string) string {
	switch field {
	case "assignee":
		return task.Assignee
	case "title":
		return task.Title
	case "description":
		return task.Description
	case "status":
		return string(task.Status)
	case "priority":
		return string(task.Priority)
	case "type":
		return string(task.Type)
	default:
		return ""
	}
}

func (wm *WorkflowManager) userHasRole(userID, role string) bool {
	// Simplified role check - in a real system this would check against a role system
	switch role {
	case "admin":
		return userID == "admin"
	case "developer":
		return userID != ""
	case "manager":
		return userID == "admin" || userID == "manager"
	default:
		return false
	}
}
