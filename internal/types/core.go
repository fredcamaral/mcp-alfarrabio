// Package types provides core parameter types for the MCP Memory Server refactor.
// This replaces the fragmented parameter system with clean, consistent types.
package types

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// ProjectID represents a project/tenant identifier for data isolation
// Replaces the confusing "repository" parameter with clear semantics
type ProjectID string

// Validate ensures ProjectID follows consistent format rules
func (p ProjectID) Validate() error {
	if len(p) == 0 {
		return errors.New("project_id cannot be empty")
	}
	if len(p) > 100 {
		return fmt.Errorf("project_id must be 100 characters or less, got %d", len(p))
	}

	// Allow alphanumeric characters, hyphens, underscores, and dots
	// This covers Git URLs, folder names, and custom identifiers
	validFormat := regexp.MustCompile(`^[a-zA-Z0-9\-_./:]+$`)
	if !validFormat.MatchString(string(p)) {
		return errors.New("project_id contains invalid characters, only alphanumeric, hyphens, underscores, dots, colons, and slashes allowed")
	}

	return nil
}

// String returns the string representation
func (p ProjectID) String() string {
	return string(p)
}

// IsEmpty returns true if the ProjectID is empty
func (p ProjectID) IsEmpty() bool {
	return strings.TrimSpace(string(p)) == ""
}

// SessionID represents a user session for scoped operations
// Sessions provide access to both session-specific and project-wide data
type SessionID string

// Validate ensures SessionID follows format rules
func (s SessionID) Validate() error {
	if len(s) == 0 {
		return nil // Empty session ID is valid (read-only access)
	}
	if len(s) > 100 {
		return fmt.Errorf("session_id must be 100 characters or less, got %d", len(s))
	}

	// Allow alphanumeric characters, hyphens, and underscores
	validFormat := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !validFormat.MatchString(string(s)) {
		return errors.New("session_id contains invalid characters, only alphanumeric, hyphens, and underscores allowed")
	}

	return nil
}

// String returns the string representation
func (s SessionID) String() string {
	return string(s)
}

// IsEmpty returns true if the SessionID is empty
func (s SessionID) IsEmpty() bool {
	return strings.TrimSpace(string(s)) == ""
}

// OperationScope defines the access level for operations
// This replaces the backwards session logic with clear semantics
type OperationScope string

const (
	// ScopeSession requires both session_id and project_id
	// Provides full access to session data + project data
	ScopeSession OperationScope = "session"

	// ScopeProject requires project_id only
	// Provides read-only access to project-wide data
	ScopeProject OperationScope = "project"

	// ScopeGlobal requires no parameters
	// Provides access to system-wide operations (health, etc.)
	ScopeGlobal OperationScope = "global"
)

// Valid returns true if the operation scope is valid
func (os OperationScope) Valid() bool {
	switch os {
	case ScopeSession, ScopeProject, ScopeGlobal:
		return true
	default:
		return false
	}
}

// RequiresProjectID returns true if the scope requires a project ID
func (os OperationScope) RequiresProjectID() bool {
	return os == ScopeSession || os == ScopeProject
}

// RequiresSessionID returns true if the scope requires a session ID
func (os OperationScope) RequiresSessionID() bool {
	return os == ScopeSession
}

// StandardParams provides consistent parameter structure across all tools
// This replaces the fragmented parameter validation with unified approach
type StandardParams struct {
	ProjectID ProjectID      `json:"project_id,omitempty"`
	SessionID SessionID      `json:"session_id,omitempty"`
	Scope     OperationScope `json:"scope"`
}

// Validate checks if the parameters are valid for the given scope
func (sp *StandardParams) Validate() error {
	// Validate individual fields
	if err := sp.ProjectID.Validate(); err != nil {
		return fmt.Errorf("invalid project_id: %w", err)
	}
	if err := sp.SessionID.Validate(); err != nil {
		return fmt.Errorf("invalid session_id: %w", err)
	}
	if !sp.Scope.Valid() {
		return fmt.Errorf("invalid scope: %s", sp.Scope)
	}

	// Validate scope requirements
	if sp.Scope.RequiresProjectID() && sp.ProjectID.IsEmpty() {
		return fmt.Errorf("scope %s requires project_id", sp.Scope)
	}
	if sp.Scope.RequiresSessionID() && sp.SessionID.IsEmpty() {
		return fmt.Errorf("scope %s requires session_id", sp.Scope)
	}

	return nil
}

// GetAccessLevel determines the access level based on provided parameters
func (sp *StandardParams) GetAccessLevel() AccessLevel {
	if !sp.SessionID.IsEmpty() {
		return AccessSession // Full access with session
	}
	if !sp.ProjectID.IsEmpty() {
		return AccessProject // Project-wide read access
	}
	return AccessGlobal // System-level access only
}

// AccessLevel represents the level of data access granted
type AccessLevel string

const (
	// AccessGlobal allows access to system-wide operations only
	AccessGlobal AccessLevel = "global"

	// AccessProject allows read-only access to project data
	AccessProject AccessLevel = "project"

	// AccessSession allows full access to session + project data
	AccessSession AccessLevel = "session"
)

// CanWrite returns true if the access level allows write operations
func (al AccessLevel) CanWrite() bool {
	return al == AccessSession
}

// CanReadProject returns true if the access level allows reading project data
func (al AccessLevel) CanReadProject() bool {
	return al == AccessProject || al == AccessSession
}

// CanReadSession returns true if the access level allows reading session data
func (al AccessLevel) CanReadSession() bool {
	return al == AccessSession
}
