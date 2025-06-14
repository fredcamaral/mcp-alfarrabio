// Package session provides logical session management for the MCP Memory Server.
// This replaces the backwards session logic with intuitive semantics.
package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"lerian-mcp-memory/internal/types"
)

// Manager handles session management with logical access semantics
// Key principle: Including session_id provides MORE access, not less
type Manager struct {
	sessions map[string]*SessionInfo
	mutex    sync.RWMutex
}

// SessionInfo contains information about an active session
type SessionInfo struct {
	SessionID   types.SessionID   `json:"session_id"`
	ProjectID   types.ProjectID   `json:"project_id"`
	CreatedAt   time.Time         `json:"created_at"`
	LastAccess  time.Time         `json:"last_access"`
	AccessCount int64             `json:"access_count"`
	IsActive    bool              `json:"is_active"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// AccessLevel represents the level of data access granted
// This replaces the confusing backwards logic with clear semantics
type AccessLevel string

const (
	// AccessReadOnly: No session_id provided - limited project data access
	// User can read basic project information but not session-specific data
	AccessReadOnly AccessLevel = "read_only"

	// AccessSession: With session_id - full session + project data access
	// User can read/write both session-specific and project-wide data
	AccessSession AccessLevel = "session"

	// AccessProject: Project scope - all project data but no session writes
	// User can read all project data but cannot create session-specific content
	AccessProject AccessLevel = "project"
)

// CanWrite returns true if the access level allows write operations
func (al AccessLevel) CanWrite() bool {
	return al == AccessSession
}

// CanReadProject returns true if access level allows reading project data
func (al AccessLevel) CanReadProject() bool {
	return al == AccessProject || al == AccessSession || al == AccessReadOnly
}

// CanReadSession returns true if access level allows reading session data
func (al AccessLevel) CanReadSession() bool {
	return al == AccessSession
}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*SessionInfo),
	}
}

// GetAccessLevel determines what data user can access based on parameters
// This implements the logical session semantics:
// - No session_id: Limited read-only access to project data
// - With session_id: Full access to session + project data
func (m *Manager) GetAccessLevel(projectID types.ProjectID, sessionID types.SessionID) AccessLevel {
	if sessionID.IsEmpty() {
		if projectID.IsEmpty() {
			return AccessReadOnly // Very limited access
		}
		return AccessReadOnly // Project data but no session access
	}

	// With session ID, user gets full access to both session and project data
	return AccessSession
}

// CreateSession creates a new session for a project
func (m *Manager) CreateSession(ctx context.Context, projectID types.ProjectID, sessionID types.SessionID) (*SessionInfo, error) {
	if projectID.IsEmpty() {
		return nil, errors.New("project_id is required to create a session")
	}
	if sessionID.IsEmpty() {
		return nil, errors.New("session_id is required to create a session")
	}

	if err := projectID.Validate(); err != nil {
		return nil, fmt.Errorf("invalid project_id: %w", err)
	}
	if err := sessionID.Validate(); err != nil {
		return nil, fmt.Errorf("invalid session_id: %w", err)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	sessionKey := m.getSessionKey(projectID, sessionID)

	// Check if session already exists
	if existing, exists := m.sessions[sessionKey]; exists {
		existing.LastAccess = time.Now()
		existing.AccessCount++
		return existing, nil
	}

	// Create new session
	sessionInfo := &SessionInfo{
		SessionID:   sessionID,
		ProjectID:   projectID,
		CreatedAt:   time.Now(),
		LastAccess:  time.Now(),
		AccessCount: 1,
		IsActive:    true,
		Metadata:    make(map[string]string),
	}

	m.sessions[sessionKey] = sessionInfo

	return sessionInfo, nil
}

// GetSession retrieves session information
func (m *Manager) GetSession(projectID types.ProjectID, sessionID types.SessionID) (*SessionInfo, error) {
	if projectID.IsEmpty() || sessionID.IsEmpty() {
		return nil, errors.New("both project_id and session_id are required")
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sessionKey := m.getSessionKey(projectID, sessionID)
	sessionInfo, exists := m.sessions[sessionKey]
	if !exists {
		return nil, errors.New("session not found")
	}

	if !sessionInfo.IsActive {
		return nil, errors.New("session is not active")
	}

	return sessionInfo, nil
}

// UpdateSessionAccess updates the last access time for a session
func (m *Manager) UpdateSessionAccess(projectID types.ProjectID, sessionID types.SessionID) error {
	if projectID.IsEmpty() || sessionID.IsEmpty() {
		return nil // No session to update
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	sessionKey := m.getSessionKey(projectID, sessionID)
	sessionInfo, exists := m.sessions[sessionKey]
	if !exists {
		// Create session on first access
		sessionInfo = &SessionInfo{
			SessionID:   sessionID,
			ProjectID:   projectID,
			CreatedAt:   time.Now(),
			LastAccess:  time.Now(),
			AccessCount: 1,
			IsActive:    true,
			Metadata:    make(map[string]string),
		}
		m.sessions[sessionKey] = sessionInfo
		return nil
	}

	sessionInfo.LastAccess = time.Now()
	sessionInfo.AccessCount++

	return nil
}

// ValidateAccess validates if the requested operation is allowed with current access level
func (m *Manager) ValidateAccess(projectID types.ProjectID, sessionID types.SessionID, operation string, requiresWrite bool) error {
	accessLevel := m.GetAccessLevel(projectID, sessionID)

	// Check write access
	if requiresWrite && !accessLevel.CanWrite() {
		return fmt.Errorf("operation '%s' requires session access for write operations, but only %s access provided", operation, accessLevel)
	}

	// Check project access
	if !accessLevel.CanReadProject() {
		return fmt.Errorf("operation '%s' requires project access", operation)
	}

	// For session-specific operations, ensure session access
	if (operation == "get_session_data" || operation == "store_session_data") && !accessLevel.CanReadSession() {
		return fmt.Errorf("operation '%s' requires session access", operation)
	}

	return nil
}

// GetProjectSessions returns all sessions for a project
func (m *Manager) GetProjectSessions(projectID types.ProjectID) ([]*SessionInfo, error) {
	if projectID.IsEmpty() {
		return nil, errors.New("project_id is required")
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var sessions []*SessionInfo
	for _, sessionInfo := range m.sessions {
		if sessionInfo.ProjectID == projectID && sessionInfo.IsActive {
			sessions = append(sessions, sessionInfo)
		}
	}

	return sessions, nil
}

// DeactivateSession marks a session as inactive
func (m *Manager) DeactivateSession(projectID types.ProjectID, sessionID types.SessionID) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	sessionKey := m.getSessionKey(projectID, sessionID)
	sessionInfo, exists := m.sessions[sessionKey]
	if !exists {
		return errors.New("session not found")
	}

	sessionInfo.IsActive = false

	return nil
}

// CleanupExpiredSessions removes sessions that haven't been accessed recently
func (m *Manager) CleanupExpiredSessions(maxAge time.Duration) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for key, sessionInfo := range m.sessions {
		if sessionInfo.LastAccess.Before(cutoff) {
			delete(m.sessions, key)
			removed++
		}
	}

	return removed
}

// GetSessionStats returns statistics about active sessions
func (m *Manager) GetSessionStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	totalSessions := len(m.sessions)
	activeSessions := 0
	projectCounts := make(map[types.ProjectID]int)

	for _, sessionInfo := range m.sessions {
		if sessionInfo.IsActive {
			activeSessions++
		}
		projectCounts[sessionInfo.ProjectID]++
	}

	return map[string]interface{}{
		"total_sessions":  totalSessions,
		"active_sessions": activeSessions,
		"project_counts":  projectCounts,
	}
}

// getSessionKey creates a unique key for a session
func (m *Manager) getSessionKey(projectID types.ProjectID, sessionID types.SessionID) string {
	return fmt.Sprintf("%s:%s", projectID, sessionID)
}

// IsWriteOperation returns true if the operation requires write access
func IsWriteOperation(operation string) bool {
	writeOperations := map[string]bool{
		"store_content":       true,
		"store_decision":      true,
		"update_content":      true,
		"delete_content":      true,
		"create_thread":       true,
		"create_relationship": true,
		"import_project":      true,
	}

	return writeOperations[operation]
}

// IsSessionOperation returns true if the operation requires session-specific access
func IsSessionOperation(operation string) bool {
	sessionOperations := map[string]bool{
		"get_session_data":    true,
		"store_session_data":  true,
		"get_session_history": true,
	}

	return sessionOperations[operation]
}
