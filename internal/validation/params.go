// Package validation provides parameter validation for MCP operations.
// This replaces the scattered validation logic with centralized, clear validation.
package validation

import (
	"errors"
	"fmt"

	"lerian-mcp-memory/internal/types"
)

// OperationRequirements defines what parameters an operation requires
type OperationRequirements struct {
	Scope              types.OperationScope `json:"scope"`
	RequiresProjectID  bool                 `json:"requires_project_id"`
	RequiresSessionID  bool                 `json:"requires_session_id"`
	AllowsEmptySession bool                 `json:"allows_empty_session"`
	Description        string               `json:"description"`
}

// Validate checks if the provided parameters meet the operation requirements
func (or *OperationRequirements) Validate(params *types.StandardParams) error {
	if params == nil {
		return errors.New("parameters cannot be nil")
	}

	// Validate the parameters themselves
	if err := params.Validate(); err != nil {
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	// Check scope matches
	if params.Scope != or.Scope {
		return fmt.Errorf("operation requires scope %s, got %s", or.Scope, params.Scope)
	}

	// Check project ID requirement
	if or.RequiresProjectID && params.ProjectID.IsEmpty() {
		return errors.New("operation requires project_id")
	}

	// Check session ID requirement
	if or.RequiresSessionID && params.SessionID.IsEmpty() {
		return errors.New("operation requires session_id")
	}

	// If operation doesn't allow empty session but we have project scope,
	// ensure we're not trying to do writes without session
	if !or.AllowsEmptySession && params.Scope == types.ScopeProject && params.SessionID.IsEmpty() {
		return errors.New("operation requires session_id for write access")
	}

	return nil
}

// ParameterValidator provides validation for MCP operations
type ParameterValidator struct {
	requirements map[string]*OperationRequirements
}

// NewParameterValidator creates a new parameter validator with operation requirements
func NewParameterValidator() *ParameterValidator {
	pv := &ParameterValidator{
		requirements: make(map[string]*OperationRequirements),
	}

	// Initialize with standard operation requirements
	pv.initializeStandardRequirements()

	return pv
}

// initializeStandardRequirements sets up validation rules for all operations
func (pv *ParameterValidator) initializeStandardRequirements() {
	// Store operations - require session for writes
	pv.requirements["store_content"] = &OperationRequirements{
		Scope:              types.ScopeSession,
		RequiresProjectID:  true,
		RequiresSessionID:  true,
		AllowsEmptySession: false,
		Description:        "Store content requires session access for write operations",
	}

	pv.requirements["store_decision"] = &OperationRequirements{
		Scope:              types.ScopeSession,
		RequiresProjectID:  true,
		RequiresSessionID:  true,
		AllowsEmptySession: false,
		Description:        "Store decision requires session access for write operations",
	}

	pv.requirements["update_content"] = &OperationRequirements{
		Scope:              types.ScopeSession,
		RequiresProjectID:  true,
		RequiresSessionID:  true,
		AllowsEmptySession: false,
		Description:        "Update content requires session access for write operations",
	}

	pv.requirements["delete_content"] = &OperationRequirements{
		Scope:              types.ScopeSession,
		RequiresProjectID:  true,
		RequiresSessionID:  true,
		AllowsEmptySession: false,
		Description:        "Delete content requires session access for write operations",
	}

	// Retrieve operations - can work with project scope (read-only)
	pv.requirements["search"] = &OperationRequirements{
		Scope:              types.ScopeProject,
		RequiresProjectID:  true,
		RequiresSessionID:  false,
		AllowsEmptySession: true,
		Description:        "Search allows project-level read access",
	}

	pv.requirements["get_content"] = &OperationRequirements{
		Scope:              types.ScopeProject,
		RequiresProjectID:  true,
		RequiresSessionID:  false,
		AllowsEmptySession: true,
		Description:        "Get content allows project-level read access",
	}

	pv.requirements["find_similar"] = &OperationRequirements{
		Scope:              types.ScopeProject,
		RequiresProjectID:  true,
		RequiresSessionID:  false,
		AllowsEmptySession: true,
		Description:        "Find similar allows project-level read access",
	}

	// Analyze operations - read-only, work with project scope
	pv.requirements["detect_patterns"] = &OperationRequirements{
		Scope:              types.ScopeProject,
		RequiresProjectID:  true,
		RequiresSessionID:  false,
		AllowsEmptySession: true,
		Description:        "Pattern detection allows project-level read access",
	}

	pv.requirements["suggest_related"] = &OperationRequirements{
		Scope:              types.ScopeProject,
		RequiresProjectID:  true,
		RequiresSessionID:  false,
		AllowsEmptySession: true,
		Description:        "Suggest related allows project-level read access",
	}

	pv.requirements["analyze_quality"] = &OperationRequirements{
		Scope:              types.ScopeProject,
		RequiresProjectID:  true,
		RequiresSessionID:  false,
		AllowsEmptySession: true,
		Description:        "Quality analysis allows project-level read access",
	}

	// System operations - global scope, no project/session required
	pv.requirements["health"] = &OperationRequirements{
		Scope:              types.ScopeGlobal,
		RequiresProjectID:  false,
		RequiresSessionID:  false,
		AllowsEmptySession: true,
		Description:        "Health check is a global operation",
	}

	pv.requirements["export_project"] = &OperationRequirements{
		Scope:              types.ScopeProject,
		RequiresProjectID:  true,
		RequiresSessionID:  false,
		AllowsEmptySession: true,
		Description:        "Export project allows project-level read access",
	}

	pv.requirements["import_project"] = &OperationRequirements{
		Scope:              types.ScopeSession,
		RequiresProjectID:  true,
		RequiresSessionID:  true,
		AllowsEmptySession: false,
		Description:        "Import project requires session access for write operations",
	}
}

// ValidateOperation validates parameters for a specific operation
func (pv *ParameterValidator) ValidateOperation(operation string, params *types.StandardParams) error {
	requirements, exists := pv.requirements[operation]
	if !exists {
		return fmt.Errorf("unknown operation: %s", operation)
	}

	return requirements.Validate(params)
}

// GetOperationRequirements returns the requirements for an operation
func (pv *ParameterValidator) GetOperationRequirements(operation string) (*OperationRequirements, error) {
	requirements, exists := pv.requirements[operation]
	if !exists {
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	return requirements, nil
}

// AddOperationRequirement adds or updates requirements for an operation
func (pv *ParameterValidator) AddOperationRequirement(operation string, requirements *OperationRequirements) {
	pv.requirements[operation] = requirements
}

// ListOperations returns all known operations
func (pv *ParameterValidator) ListOperations() []string {
	operations := make([]string, 0, len(pv.requirements))
	for operation := range pv.requirements {
		operations = append(operations, operation)
	}
	return operations
}

// CreateErrorMessage creates a helpful error message for parameter validation failures
func CreateErrorMessage(operation string, err error, params *types.StandardParams) string {
	baseMsg := fmt.Sprintf("Operation '%s' failed parameter validation: %s", operation, err.Error())

	if params != nil {
		baseMsg += fmt.Sprintf("\nProvided parameters: project_id='%s', session_id='%s', scope='%s'",
			params.ProjectID, params.SessionID, params.Scope)
	}

	// Add helpful suggestions based on common errors
	if params != nil {
		if params.ProjectID.IsEmpty() {
			baseMsg += "\nSuggestion: Provide a project_id to identify which project/repository you're working with"
		}
		if params.SessionID.IsEmpty() && (operation == "store_content" || operation == "update_content" || operation == "delete_content") {
			baseMsg += "\nSuggestion: Provide a session_id for write operations to ensure proper data isolation"
		}
	}

	return baseMsg
}
