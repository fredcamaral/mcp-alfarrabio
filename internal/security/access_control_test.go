package security

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAccessControlManager(t *testing.T) {
	acm := NewAccessControlManager()
	assert.NotNil(t, acm)
	assert.True(t, acm.IsEnabled())
	assert.Empty(t, acm.users)
	assert.Empty(t, acm.tokens)
	assert.Empty(t, acm.repositories)
	assert.Empty(t, acm.policies)
}

func TestAccessControlManager_CreateUser(t *testing.T) {
	acm := NewAccessControlManager()

	tests := []struct {
		name     string
		username string
		email    string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Valid user",
			username: "testuser",
			email:    "test@example.com",
			wantErr:  false,
		},
		{
			name:     "Empty username",
			username: "",
			email:    "test@example.com",
			wantErr:  true,
			errMsg:   "username and email are required",
		},
		{
			name:     "Empty email",
			username: "testuser",
			email:    "",
			wantErr:  true,
			errMsg:   "username and email are required",
		},
		{
			name:     "Duplicate username",
			username: "testuser",
			email:    "another@example.com",
			wantErr:  true,
			errMsg:   "user already exists",
		},
		{
			name:     "Duplicate email",
			username: "anotheruser",
			email:    "test@example.com",
			wantErr:  true,
			errMsg:   "user already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := acm.CreateUser(tt.username, tt.email)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, user)
				assert.NotEmpty(t, user.ID)
				assert.Equal(t, tt.username, user.Username)
				assert.Equal(t, tt.email, user.Email)
				assert.True(t, user.IsActive)
				assert.Empty(t, user.Permissions)
				assert.Empty(t, user.Repositories)
			}
		})
	}
}

func TestAccessControlManager_GenerateToken(t *testing.T) {
	acm := NewAccessControlManager()

	// Create a user
	user, err := acm.CreateUser("testuser", "test@example.com")
	require.NoError(t, err)

	tests := []struct {
		name     string
		userID   string
		scope    []string
		duration time.Duration
		wantErr  bool
		errMsg   string
		setup    func()
	}{
		{
			name:     "Valid token",
			userID:   user.ID,
			scope:    []string{"read", "write"},
			duration: time.Hour,
			wantErr:  false,
		},
		{
			name:     "Non-existent user",
			userID:   "invalid-user-id",
			scope:    []string{"read"},
			duration: time.Hour,
			wantErr:  true,
			errMsg:   "user not found",
		},
		{
			name:     "Inactive user",
			userID:   user.ID,
			scope:    []string{"read"},
			duration: time.Hour,
			wantErr:  true,
			errMsg:   "user is not active",
			setup: func() {
				user.IsActive = false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			token, err := acm.GenerateToken(tt.userID, tt.scope, tt.duration)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, token)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, token)
				assert.NotEmpty(t, token.Token)
				assert.Equal(t, tt.userID, token.UserID)
				assert.Equal(t, tt.scope, token.Scope)
				assert.True(t, token.ExpiresAt.After(time.Now()))
			}

			// Reset user state
			user.IsActive = true
		})
	}
}

func TestAccessControlManager_ValidateToken(t *testing.T) {
	acm := NewAccessControlManager()

	// Create a user and token
	user, err := acm.CreateUser("testuser", "test@example.com")
	require.NoError(t, err)

	validToken, err := acm.GenerateToken(user.ID, []string{"read"}, time.Hour)
	require.NoError(t, err)

	// Create expired token
	expiredToken, err := acm.GenerateToken(user.ID, []string{"read"}, -time.Hour) // Negative duration for expired token
	require.NoError(t, err)

	tests := []struct {
		name        string
		tokenString string
		wantErr     bool
		errMsg      string
		setup       func()
		cleanup     func()
	}{
		{
			name:        "Valid token",
			tokenString: validToken.Token,
			wantErr:     false,
		},
		{
			name:        "Invalid token",
			tokenString: "invalid-token",
			wantErr:     true,
			errMsg:      "invalid token",
		},
		{
			name:        "Expired token",
			tokenString: expiredToken.Token,
			wantErr:     true,
			errMsg:      "token expired",
		},
		{
			name:        "Token for inactive user",
			tokenString: validToken.Token,
			wantErr:     true,
			errMsg:      "user not active",
			setup: func() {
				user.IsActive = false
			},
			cleanup: func() {
				user.IsActive = true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			token, err := acm.ValidateToken(tt.tokenString)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, token)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, tt.tokenString, token.Token)
			}

			if tt.cleanup != nil {
				tt.cleanup()
			}
		})
	}
}

func TestAccessControlManager_GrantRevokePermission(t *testing.T) {
	acm := NewAccessControlManager()

	// Create a user
	user, err := acm.CreateUser("testuser", "test@example.com")
	require.NoError(t, err)

	permission := Permission{
		Resource: "repository:test-repo",
		Action:   "read",
		Level:    AccessLevelRead,
	}

	// Grant permission
	err = acm.GrantPermission(user.ID, permission)
	require.NoError(t, err)
	assert.Len(t, user.Permissions, 1)
	assert.Equal(t, permission, user.Permissions[0])

	// Grant same permission again (should update)
	permission.Level = AccessLevelWrite
	err = acm.GrantPermission(user.ID, permission)
	require.NoError(t, err)
	assert.Len(t, user.Permissions, 1)
	assert.Equal(t, AccessLevelWrite, user.Permissions[0].Level)

	// Grant different permission
	permission2 := Permission{
		Resource: "repository:another-repo",
		Action:   "write",
		Level:    AccessLevelWrite,
	}
	err = acm.GrantPermission(user.ID, permission2)
	require.NoError(t, err)
	assert.Len(t, user.Permissions, 2)

	// Revoke permission
	err = acm.RevokePermission(user.ID, "repository:test-repo", "read")
	require.NoError(t, err)
	assert.Len(t, user.Permissions, 1)

	// Try to revoke non-existent permission
	err = acm.RevokePermission(user.ID, "non-existent", "read")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission not found")

	// Try to grant permission to non-existent user
	err = acm.GrantPermission("invalid-user", permission)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestAccessControlManager_CheckAccess(t *testing.T) {
	acm := NewAccessControlManager()
	ctx := context.Background()

	// Create users
	user1, err := acm.CreateUser("user1", "user1@example.com")
	require.NoError(t, err)

	user2, err := acm.CreateUser("user2", "user2@example.com")
	require.NoError(t, err)

	// Grant permissions
	err = acm.GrantPermission(user1.ID, Permission{
		Resource: "repository:repo1",
		Action:   "read",
		Level:    AccessLevelRead,
	})
	require.NoError(t, err)

	err = acm.GrantPermission(user1.ID, Permission{
		Resource: "repository:repo1",
		Action:   "write",
		Level:    AccessLevelWrite,
	})
	require.NoError(t, err)

	// Add wildcard permission
	err = acm.GrantPermission(user2.ID, Permission{
		Resource: "*",
		Action:   "*",
		Level:    AccessLevelAdmin,
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		userID   string
		action   string
		resource string
		want     bool
		wantErr  bool
		setup    func()
		cleanup  func()
	}{
		{
			name:     "User has specific permission",
			userID:   user1.ID,
			action:   "read",
			resource: "repository:repo1",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "User lacks permission",
			userID:   user1.ID,
			action:   "delete",
			resource: "repository:repo1",
			want:     false,
			wantErr:  false,
		},
		{
			name:     "User has wildcard permission",
			userID:   user2.ID,
			action:   "delete",
			resource: "repository:any-repo",
			want:     true,
			wantErr:  false,
		},
		{
			name:     "Non-existent user",
			userID:   "invalid-user",
			action:   "read",
			resource: "repository:repo1",
			want:     false,
			wantErr:  true,
		},
		{
			name:     "Inactive user",
			userID:   user1.ID,
			action:   "read",
			resource: "repository:repo1",
			want:     false,
			wantErr:  true,
			setup: func() {
				user1.IsActive = false
			},
			cleanup: func() {
				user1.IsActive = true
			},
		},
		{
			name:     "Access control disabled",
			userID:   user1.ID,
			action:   "anything",
			resource: "anything",
			want:     true,
			wantErr:  false,
			setup: func() {
				acm.Disable()
			},
			cleanup: func() {
				acm.Enable()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			got, err := acm.CheckAccess(ctx, tt.userID, tt.action, tt.resource)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)

			if tt.cleanup != nil {
				tt.cleanup()
			}
		})
	}
}

func TestAccessControlManager_Repository(t *testing.T) {
	acm := NewAccessControlManager()

	// Create users
	owner, err := acm.CreateUser("owner", "owner@example.com")
	require.NoError(t, err)

	user, err := acm.CreateUser("user", "user@example.com")
	require.NoError(t, err)

	// Create repository
	repo, err := acm.CreateRepository("test-repo", owner.ID, false)
	require.NoError(t, err)
	assert.NotNil(t, repo)
	assert.NotEmpty(t, repo.ID)
	assert.Equal(t, "test-repo", repo.Name)
	assert.Equal(t, owner.ID, repo.Owner)
	assert.False(t, repo.IsPublic)
	assert.Contains(t, repo.AllowedUsers, owner.ID)

	// Grant repository access
	err = acm.GrantRepositoryAccess(repo.ID, user.ID, AccessLevelRead)
	require.NoError(t, err)
	assert.Contains(t, repo.AllowedUsers, user.ID)

	// Check that permission was granted
	assert.Len(t, user.Permissions, 1)
	assert.Equal(t, fmt.Sprintf("repository:%s", repo.ID), user.Permissions[0].Resource)
	assert.Equal(t, "*", user.Permissions[0].Action)
	assert.Equal(t, AccessLevelRead, user.Permissions[0].Level)

	// Grant access to non-existent repository
	err = acm.GrantRepositoryAccess("invalid-repo", user.ID, AccessLevelRead)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository not found")
}

func TestAccessControlManager_Policies(t *testing.T) {
	acm := NewAccessControlManager()
	ctx := context.Background()

	// Create a user
	user, err := acm.CreateUser("testuser", "test@example.com")
	require.NoError(t, err)

	// Add a deny policy
	denyPolicy := AccessPolicy{
		ID:       "deny-delete",
		Name:     "Deny Delete Operations",
		IsActive: true,
		Rules: []AccessRule{
			{
				Resource: "repository:.*",
				Action:   "delete",
				Effect:   "deny",
				Priority: 1,
			},
		},
		CreatedAt: time.Now(),
	}
	acm.policies = append(acm.policies, denyPolicy)

	// Grant permission that should be overridden by deny policy
	err = acm.GrantPermission(user.ID, Permission{
		Resource: "repository:test",
		Action:   "*",
		Level:    AccessLevelAdmin,
	})
	require.NoError(t, err)

	// Check access - should be denied due to policy
	hasAccess, err := acm.CheckAccess(ctx, user.ID, "delete", "repository:test")
	require.NoError(t, err)
	assert.False(t, hasAccess)

	// Check other actions - should be allowed
	hasAccess, err = acm.CheckAccess(ctx, user.ID, "read", "repository:test")
	require.NoError(t, err)
	assert.True(t, hasAccess)
}

func TestAccessControlManager_PermissionExpiration(t *testing.T) {
	acm := NewAccessControlManager()
	ctx := context.Background()

	// Create a user
	user, err := acm.CreateUser("testuser", "test@example.com")
	require.NoError(t, err)

	// Grant permission with expiration
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	// Valid permission
	err = acm.GrantPermission(user.ID, Permission{
		Resource:  "repository:valid",
		Action:    "read",
		Level:     AccessLevelRead,
		ExpiresAt: &future,
	})
	require.NoError(t, err)

	// Expired permission
	err = acm.GrantPermission(user.ID, Permission{
		Resource:  "repository:expired",
		Action:    "read",
		Level:     AccessLevelRead,
		ExpiresAt: &past,
	})
	require.NoError(t, err)

	// Check access
	hasAccess, err := acm.CheckAccess(ctx, user.ID, "read", "repository:valid")
	require.NoError(t, err)
	assert.True(t, hasAccess)

	hasAccess, err = acm.CheckAccess(ctx, user.ID, "read", "repository:expired")
	require.NoError(t, err)
	assert.False(t, hasAccess)
}

func TestAccessControlManager_Conditions(t *testing.T) {
	acm := NewAccessControlManager()
	ctx := context.Background()

	// Create users
	user1, err := acm.CreateUser("user1", "user1@example.com")
	require.NoError(t, err)
	user1.Metadata["department"] = "engineering"
	user1.Repositories = []string{"repo1", "repo2"}

	user2, err := acm.CreateUser("user2", "user2@example.com")
	require.NoError(t, err)
	user2.Metadata["department"] = "marketing"

	// Add policy with conditions
	policy := AccessPolicy{
		ID:       "conditional-access",
		Name:     "Conditional Access Policy",
		IsActive: true,
		Rules: []AccessRule{
			{
				Resource: "repository:engineering-.*",
				Action:   ".*",
				Effect:   "allow",
				Priority: 1,
				Conditions: []Condition{
					{
						Field:    "department",
						Operator: "equals",
						Value:    "engineering",
					},
				},
			},
			{
				Resource: "repository:.*",
				Action:   "read",
				Effect:   "allow",
				Priority: 2,
				Conditions: []Condition{
					{
						Field:    "user.repositories",
						Operator: "contains",
						Value:    "repo1",
					},
				},
			},
		},
		CreatedAt: time.Now(),
	}
	acm.policies = append(acm.policies, policy)

	// Test conditions
	tests := []struct {
		name     string
		userID   string
		action   string
		resource string
		want     bool
	}{
		{
			name:     "Engineering user can access engineering repos",
			userID:   user1.ID,
			action:   "write",
			resource: "repository:engineering-api",
			want:     true,
		},
		{
			name:     "Non-engineering user cannot access engineering repos",
			userID:   user2.ID,
			action:   "write",
			resource: "repository:engineering-api",
			want:     false,
		},
		{
			name:     "User with repo1 can read any repo",
			userID:   user1.ID,
			action:   "read",
			resource: "repository:any-repo",
			want:     true,
		},
		{
			name:     "User without repo1 cannot read",
			userID:   user2.ID,
			action:   "read",
			resource: "repository:any-repo",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := acm.CheckAccess(ctx, tt.userID, tt.action, tt.resource)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateSecureID(t *testing.T) {
	// Test uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateSecureID()
		assert.NotEmpty(t, id)
		assert.False(t, ids[id], "ID should be unique")
		ids[id] = true
	}
}

func TestGenerateSecureToken(t *testing.T) {
	// Test uniqueness and length
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token := generateSecureToken()
		assert.NotEmpty(t, token)
		assert.Len(t, token, 64) // SHA256 produces 32 bytes = 64 hex chars
		assert.False(t, tokens[token], "Token should be unique")
		tokens[token] = true
	}
}

// Benchmark tests
func BenchmarkCheckAccess(b *testing.B) {
	acm := NewAccessControlManager()
	ctx := context.Background()

	// Create user with permissions
	user, _ := acm.CreateUser("benchuser", "bench@example.com")
	for i := 0; i < 10; i++ {
		_ = acm.GrantPermission(user.ID, Permission{
			Resource: fmt.Sprintf("repository:repo%d", i),
			Action:   "read",
			Level:    AccessLevelRead,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = acm.CheckAccess(ctx, user.ID, "read", "repository:repo5")
	}
}

func BenchmarkGenerateToken(b *testing.B) {
	acm := NewAccessControlManager()
	user, _ := acm.CreateUser("benchuser", "bench@example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = acm.GenerateToken(user.ID, []string{"read"}, time.Hour)
	}
}
