// Package security provides access control, encryption, and security
// mechanisms for protecting data in the MCP Memory Server.
package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"
)

// AccessLevel represents different access levels
type AccessLevel string

const (
	AccessLevelNone  AccessLevel = "none"
	AccessLevelRead  AccessLevel = "read"
	AccessLevelWrite AccessLevel = "write"
	AccessLevelAdmin AccessLevel = "admin"
)

// Permission represents a specific permission
type Permission struct {
	Resource  string      `json:"resource"`
	Action    string      `json:"action"`
	Level     AccessLevel `json:"level"`
	ExpiresAt *time.Time  `json:"expires_at,omitempty"`
}

// AccessToken represents an authentication token
type AccessToken struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Scope     []string  `json:"scope"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// User represents a user in the system
type User struct {
	ID           string                 `json:"id"`
	Username     string                 `json:"username"`
	Email        string                 `json:"email"`
	Permissions  []Permission           `json:"permissions"`
	Repositories []string               `json:"repositories"`
	CreatedAt    time.Time              `json:"created_at"`
	LastLogin    *time.Time             `json:"last_login,omitempty"`
	IsActive     bool                   `json:"is_active"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AccessControlManager manages access control and permissions
type AccessControlManager struct {
	users        map[string]*User
	tokens       map[string]*AccessToken
	repositories map[string]*Repository
	policies     []AccessPolicy
	enabled      bool
}

// Repository represents a repository with access controls
type Repository struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Owner        string      `json:"owner"`
	AccessLevel  AccessLevel `json:"access_level"`
	AllowedUsers []string    `json:"allowed_users"`
	IsPublic     bool        `json:"is_public"`
	CreatedAt    time.Time   `json:"created_at"`
}

// AccessPolicy represents an access control policy
type AccessPolicy struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Rules     []AccessRule           `json:"rules"`
	IsActive  bool                   `json:"is_active"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AccessRule represents a single access rule
type AccessRule struct {
	Resource   string      `json:"resource"`
	Action     string      `json:"action"`
	Effect     string      `json:"effect"` // "allow" or "deny"
	Conditions []Condition `json:"conditions"`
	Priority   int         `json:"priority"`
}

// Condition represents a condition for access rules
type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Result    string                 `json:"result"` // "success" or "denied"
	Timestamp time.Time              `json:"timestamp"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewAccessControlManager creates a new access control manager
func NewAccessControlManager() *AccessControlManager {
	return &AccessControlManager{
		users:        make(map[string]*User),
		tokens:       make(map[string]*AccessToken),
		repositories: make(map[string]*Repository),
		policies:     make([]AccessPolicy, 0),
		enabled:      true,
	}
}

// CreateUser creates a new user
func (acm *AccessControlManager) CreateUser(username, email string) (*User, error) {
	if username == "" || email == "" {
		return nil, errors.New("username and email are required")
	}

	// Check if user already exists
	for _, user := range acm.users {
		if user.Username == username || user.Email == email {
			return nil, errors.New("user already exists")
		}
	}

	userID := generateSecureID()
	user := &User{
		ID:           userID,
		Username:     username,
		Email:        email,
		Permissions:  make([]Permission, 0),
		Repositories: make([]string, 0),
		CreatedAt:    time.Now(),
		IsActive:     true,
		Metadata:     make(map[string]interface{}),
	}

	acm.users[userID] = user
	return user, nil
}

// GenerateToken generates an access token for a user
func (acm *AccessControlManager) GenerateToken(userID string, scope []string, duration time.Duration) (*AccessToken, error) {
	user, exists := acm.users[userID]
	if !exists {
		return nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, errors.New("user is not active")
	}

	tokenString := generateSecureToken()
	token := &AccessToken{
		Token:     tokenString,
		UserID:    userID,
		Scope:     scope,
		ExpiresAt: time.Now().Add(duration),
		CreatedAt: time.Now(),
	}

	acm.tokens[tokenString] = token

	// Update user last login
	now := time.Now()
	user.LastLogin = &now

	return token, nil
}

// ValidateToken validates an access token
func (acm *AccessControlManager) ValidateToken(tokenString string) (*AccessToken, error) {
	token, exists := acm.tokens[tokenString]
	if !exists {
		return nil, errors.New("invalid token")
	}

	if time.Now().After(token.ExpiresAt) {
		delete(acm.tokens, tokenString)
		return nil, errors.New("token expired")
	}

	// Check if user is still active
	user, exists := acm.users[token.UserID]
	if !exists || !user.IsActive {
		delete(acm.tokens, tokenString)
		return nil, errors.New("user not active")
	}

	return token, nil
}

// CheckAccess checks if a user has access to perform an action on a resource
func (acm *AccessControlManager) CheckAccess(ctx context.Context, userID, action, resource string) (bool, error) {
	if !acm.enabled {
		return true, nil
	}

	user, exists := acm.users[userID]
	if !exists {
		return false, errors.New("user not found")
	}

	if !user.IsActive {
		return false, errors.New("user not active")
	}

	// Check user permissions
	hasPermission := false
	for _, permission := range user.Permissions {
		if acm.matchesPermission(permission, action, resource) {
			if permission.ExpiresAt == nil || time.Now().Before(*permission.ExpiresAt) {
				hasPermission = true
				break
			}
		}
	}

	// Check policies
	for _, policy := range acm.policies {
		if !policy.IsActive {
			continue
		}

		for _, rule := range policy.Rules {
			if acm.matchesRule(rule, action, resource, user) {
				if rule.Effect == "deny" {
					// Deny overrides allow
					hasPermission = false
					break
				} else if rule.Effect == "allow" {
					hasPermission = true
				}
			}
		}
	}

	// Log access attempt
	result := "denied"
	if hasPermission {
		result = "success"
	}

	acm.logAccess(userID, action, resource, result, ctx)

	return hasPermission, nil
}

// GrantPermission grants a permission to a user
func (acm *AccessControlManager) GrantPermission(userID string, permission Permission) error {
	user, exists := acm.users[userID]
	if !exists {
		return errors.New("user not found")
	}

	// Check if permission already exists
	for i, existing := range user.Permissions {
		if existing.Resource == permission.Resource && existing.Action == permission.Action {
			// Update existing permission
			user.Permissions[i] = permission
			return nil
		}
	}

	// Add new permission
	user.Permissions = append(user.Permissions, permission)
	return nil
}

// RevokePermission revokes a permission from a user
func (acm *AccessControlManager) RevokePermission(userID, resource, action string) error {
	user, exists := acm.users[userID]
	if !exists {
		return errors.New("user not found")
	}

	// Remove permission
	for i, permission := range user.Permissions {
		if permission.Resource == resource && permission.Action == action {
			user.Permissions = append(user.Permissions[:i], user.Permissions[i+1:]...)
			return nil
		}
	}

	return errors.New("permission not found")
}

// CreateRepository creates a repository with access controls
func (acm *AccessControlManager) CreateRepository(name, owner string, isPublic bool) (*Repository, error) {
	repoID := generateSecureID()
	repo := &Repository{
		ID:           repoID,
		Name:         name,
		Owner:        owner,
		AccessLevel:  AccessLevelRead,
		AllowedUsers: []string{owner},
		IsPublic:     isPublic,
		CreatedAt:    time.Now(),
	}

	acm.repositories[repoID] = repo
	return repo, nil
}

// GrantRepositoryAccess grants access to a repository
func (acm *AccessControlManager) GrantRepositoryAccess(repoID, userID string, level AccessLevel) error {
	repo, exists := acm.repositories[repoID]
	if !exists {
		return errors.New("repository not found")
	}

	// Check if user already has access
	for _, allowedUser := range repo.AllowedUsers {
		if allowedUser == userID {
			return nil // User already has access
		}
	}

	repo.AllowedUsers = append(repo.AllowedUsers, userID)

	// Grant specific permission
	permission := Permission{
		Resource: "repository:" + repoID,
		Action:   "*",
		Level:    level,
	}

	return acm.GrantPermission(userID, permission)
}

// Helper methods

func (acm *AccessControlManager) matchesPermission(permission Permission, action, resource string) bool {
	// Check resource match
	if permission.Resource != "*" && !strings.Contains(resource, permission.Resource) {
		return false
	}

	// Check action match
	if permission.Action != "*" && permission.Action != action {
		return false
	}

	return true
}

func (acm *AccessControlManager) matchesRule(rule AccessRule, action, resource string, user *User) bool {
	// Check resource pattern
	if rule.Resource != "*" {
		matched, _ := regexp.MatchString(rule.Resource, resource)
		if !matched {
			return false
		}
	}

	// Check action pattern
	if rule.Action != "*" {
		matched, _ := regexp.MatchString(rule.Action, action)
		if !matched {
			return false
		}
	}

	// Check conditions
	for _, condition := range rule.Conditions {
		if !acm.evaluateCondition(condition, user) {
			return false
		}
	}

	return true
}

func (acm *AccessControlManager) evaluateCondition(condition Condition, user *User) bool {
	fieldValue := acm.extractFieldValue(condition.Field, user)
	return acm.evaluateOperator(condition.Operator, fieldValue, condition.Value)
}

// extractFieldValue extracts the field value from the user based on the field name
func (acm *AccessControlManager) extractFieldValue(field string, user *User) interface{} {
	switch field {
	case "user.id":
		return user.ID
	case "user.username":
		return user.Username
	case "user.email":
		return user.Email
	case "user.repositories":
		return user.Repositories
	default:
		if metadata, exists := user.Metadata[field]; exists {
			return metadata
		}
		return nil
	}
}

// evaluateOperator evaluates the condition operator against field and target values
func (acm *AccessControlManager) evaluateOperator(operator string, fieldValue interface{}, targetValue interface{}) bool {
	switch operator {
	case "equals":
		return fieldValue == targetValue
	case "contains":
		return acm.evaluateContains(fieldValue, targetValue)
	case "in":
		return acm.evaluateIn(fieldValue, targetValue)
	default:
		return false
	}
}

// evaluateContains evaluates the "contains" operator for strings and slices
func (acm *AccessControlManager) evaluateContains(fieldValue interface{}, targetValue interface{}) bool {
	if str, ok := fieldValue.(string); ok {
		if target, ok := targetValue.(string); ok {
			return strings.Contains(str, target)
		}
	}
	
	if slice, ok := fieldValue.([]string); ok {
		if target, ok := targetValue.(string); ok {
			return acm.sliceContainsString(slice, target)
		}
	}
	
	return false
}

// evaluateIn evaluates the "in" operator for checking membership in a slice
func (acm *AccessControlManager) evaluateIn(fieldValue interface{}, targetValue interface{}) bool {
	if slice, ok := targetValue.([]interface{}); ok {
		for _, item := range slice {
			if item == fieldValue {
				return true
			}
		}
	}
	return false
}

// sliceContainsString checks if a string slice contains a specific string
func (acm *AccessControlManager) sliceContainsString(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

func (acm *AccessControlManager) logAccess(userID, action, resource, result string, _ context.Context) {
	// In a real implementation, this would write to an audit log
	// For now, we'll just create the log entry structure
	log := AuditLog{
		ID:        generateSecureID(),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Result:    result,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"context": "mcp_memory_access",
		},
	}

	// TODO: Write to persistent audit log storage
	_ = log
}

// Utility functions

func generateSecureID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic("failed to read random bytes: " + err.Error())
	}
	return hex.EncodeToString(bytes)
}

func generateSecureToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic("failed to read random bytes: " + err.Error())
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])
}

// Enable turns on access control for the manager
func (acm *AccessControlManager) Enable() {
	acm.enabled = true
}

func (acm *AccessControlManager) Disable() {
	acm.enabled = false
}

func (acm *AccessControlManager) IsEnabled() bool {
	return acm.enabled
}
