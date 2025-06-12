// Package security provides comprehensive authentication and authorization
package security

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// AuthManager provides comprehensive authentication and authorization
type AuthManager struct {
	config        *AuthConfig
	tokenManager  *TokenManager
	sessionStore  SessionStore
	userStore     UserStore
	roleManager   *RoleManager
	auditLogger   *AuditLogger
	rateLimiter   *RateLimiter
	mutex         sync.RWMutex
}

// AuthConfig defines authentication configuration
type AuthConfig struct {
	// Token settings
	JWTSecret               string        `json:"jwt_secret"`
	AccessTokenExpiry       time.Duration `json:"access_token_expiry"`
	RefreshTokenExpiry      time.Duration `json:"refresh_token_expiry"`
	TokenRotationEnabled    bool          `json:"token_rotation_enabled"`
	
	// Password settings
	PasswordMinLength       int           `json:"password_min_length"`
	PasswordRequireUpper    bool          `json:"password_require_upper"`
	PasswordRequireLower    bool          `json:"password_require_lower"`
	PasswordRequireDigit    bool          `json:"password_require_digit"`
	PasswordRequireSpecial  bool          `json:"password_require_special"`
	PasswordHashCost        int           `json:"password_hash_cost"`
	
	// Session settings
	SessionTimeout          time.Duration `json:"session_timeout"`
	MaxConcurrentSessions   int           `json:"max_concurrent_sessions"`
	SessionCookieSecure     bool          `json:"session_cookie_secure"`
	SessionCookieHTTPOnly   bool          `json:"session_cookie_http_only"`
	
	// Rate limiting
	EnableRateLimit         bool          `json:"enable_rate_limit"`
	LoginAttempts           int           `json:"login_attempts"`
	LoginWindow             time.Duration `json:"login_window"`
	LockoutDuration         time.Duration `json:"lockout_duration"`
	
	// Audit settings
	EnableAuditLogging      bool          `json:"enable_audit_logging"`
	AuditLogRetention       time.Duration `json:"audit_log_retention"`
	LogFailedAttempts       bool          `json:"log_failed_attempts"`
	LogSuccessfulAuth       bool          `json:"log_successful_auth"`
	
	// MFA settings
	EnableMFA               bool          `json:"enable_mfa"`
	MFARequired             bool          `json:"mfa_required"`
	TOTPIssuer              string        `json:"totp_issuer"`
	BackupCodesCount        int           `json:"backup_codes_count"`
}

// DefaultAuthConfig returns secure default configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		JWTSecret:               generateSecureSecret(),
		AccessTokenExpiry:       15 * time.Minute,
		RefreshTokenExpiry:      7 * 24 * time.Hour, // 7 days
		TokenRotationEnabled:    true,
		PasswordMinLength:       12,
		PasswordRequireUpper:    true,
		PasswordRequireLower:    true,
		PasswordRequireDigit:    true,
		PasswordRequireSpecial:  true,
		PasswordHashCost:        bcrypt.DefaultCost,
		SessionTimeout:          30 * time.Minute,
		MaxConcurrentSessions:   5,
		SessionCookieSecure:     true,
		SessionCookieHTTPOnly:   true,
		EnableRateLimit:         true,
		LoginAttempts:           5,
		LoginWindow:             15 * time.Minute,
		LockoutDuration:         1 * time.Hour,
		EnableAuditLogging:      true,
		AuditLogRetention:       90 * 24 * time.Hour, // 90 days
		LogFailedAttempts:       true,
		LogSuccessfulAuth:       true,
		EnableMFA:               true,
		MFARequired:             false,
		TOTPIssuer:              "MCP Memory Server",
		BackupCodesCount:        10,
	}
}

// User represents an authenticated user
type User struct {
	ID              string                 `json:"id"`
	Username        string                 `json:"username"`
	Email           string                 `json:"email"`
	PasswordHash    string                 `json:"password_hash"`
	Salt            string                 `json:"salt"`
	Roles           []string               `json:"roles"`
	Permissions     []string               `json:"permissions"`
	MFAEnabled      bool                   `json:"mfa_enabled"`
	MFASecret       string                 `json:"mfa_secret,omitempty"`
	BackupCodes     []string               `json:"backup_codes,omitempty"`
	LastLogin       time.Time              `json:"last_login"`
	FailedAttempts  int                    `json:"failed_attempts"`
	LockedUntil     time.Time              `json:"locked_until"`
	Active          bool                   `json:"active"`
	EmailVerified   bool                   `json:"email_verified"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// Session represents a user session
type Session struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	AccessToken   string                 `json:"access_token"`
	RefreshToken  string                 `json:"refresh_token"`
	ExpiresAt     time.Time              `json:"expires_at"`
	RefreshAt     time.Time              `json:"refresh_at"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	Active        bool                   `json:"active"`
	CreatedAt     time.Time              `json:"created_at"`
	LastAccessed  time.Time              `json:"last_accessed"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// AuthRequest represents an authentication request
type AuthRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	MFACode     string `json:"mfa_code,omitempty"`
	RememberMe  bool   `json:"remember_me"`
	IPAddress   string `json:"ip_address"`
	UserAgent   string `json:"user_agent"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Success      bool      `json:"success"`
	AccessToken  string    `json:"access_token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	User         *User     `json:"user,omitempty"`
	Permissions  []string  `json:"permissions,omitempty"`
	MFARequired  bool      `json:"mfa_required"`
	ErrorCode    string    `json:"error_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// UserStore defines interface for user storage
type UserStore interface {
	GetUser(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, userID string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, userID string) error
	ListUsers(ctx context.Context, filters map[string]interface{}) ([]*User, error)
}

// SessionStore defines interface for session storage
type SessionStore interface {
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, sessionID string) error
	DeleteUserSessions(ctx context.Context, userID string) error
	ListActiveSessions(ctx context.Context, userID string) ([]*Session, error)
}

// NewAuthManager creates a new comprehensive authentication manager
func NewAuthManager(config *AuthConfig, userStore UserStore, sessionStore SessionStore) *AuthManager {
	if config == nil {
		config = DefaultAuthConfig()
	}
	
	manager := &AuthManager{
		config:       config,
		tokenManager: NewTokenManager(config),
		sessionStore: sessionStore,
		userStore:    userStore,
		roleManager:  NewRoleManager(),
		auditLogger:  NewAuditLogger(config),
		rateLimiter:  NewRateLimiter(config),
	}
	
	return manager
}

// Authenticate authenticates a user with credentials
func (am *AuthManager) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
	// Check rate limiting
	if am.config.EnableRateLimit {
		if blocked, remaining := am.rateLimiter.CheckLimit(req.IPAddress, req.Username); blocked {
			am.auditLogger.LogEvent(ctx, "auth_rate_limited", map[string]interface{}{
				"username":   req.Username,
				"ip_address": req.IPAddress,
				"remaining":  remaining,
			})
			return &AuthResponse{
				Success:      false,
				ErrorCode:    "RATE_LIMITED",
				ErrorMessage: "Too many login attempts. Please try again later.",
			}, nil
		}
	}
	
	// Get user
	user, err := am.userStore.GetUser(ctx, req.Username)
	if err != nil {
		am.logFailedAuth(ctx, req, "USER_NOT_FOUND", "User not found")
		return &AuthResponse{
			Success:      false,
			ErrorCode:    "INVALID_CREDENTIALS",
			ErrorMessage: "Invalid username or password",
		}, nil
	}
	
	// Check if user is active
	if !user.Active {
		am.logFailedAuth(ctx, req, "USER_INACTIVE", "User account is inactive")
		return &AuthResponse{
			Success:      false,
			ErrorCode:    "ACCOUNT_INACTIVE",
			ErrorMessage: "Account is inactive",
		}, nil
	}
	
	// Check if user is locked
	if time.Now().Before(user.LockedUntil) {
		am.logFailedAuth(ctx, req, "USER_LOCKED", "User account is locked")
		return &AuthResponse{
			Success:      false,
			ErrorCode:    "ACCOUNT_LOCKED",
			ErrorMessage: fmt.Sprintf("Account is locked until %v", user.LockedUntil),
		}, nil
	}
	
	// Verify password
	if !am.verifyPassword(req.Password, user.PasswordHash, user.Salt) {
		am.handleFailedAuth(ctx, user, req)
		return &AuthResponse{
			Success:      false,
			ErrorCode:    "INVALID_CREDENTIALS",
			ErrorMessage: "Invalid username or password",
		}, nil
	}
	
	// Check MFA if enabled
	if user.MFAEnabled || am.config.MFARequired {
		if req.MFACode == "" {
			return &AuthResponse{
				Success:     false,
				MFARequired: true,
				ErrorCode:   "MFA_REQUIRED",
				ErrorMessage: "Multi-factor authentication required",
			}, nil
		}
		
		if !am.verifyMFA(user, req.MFACode) {
			am.handleFailedAuth(ctx, user, req)
			return &AuthResponse{
				Success:      false,
				ErrorCode:    "INVALID_MFA",
				ErrorMessage: "Invalid MFA code",
			}, nil
		}
	}
	
	// Authentication successful
	return am.createSuccessfulAuth(ctx, user, req)
}

// RefreshToken refreshes an access token using a refresh token
func (am *AuthManager) RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error) {
	// Validate refresh token
	claims, err := am.tokenManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return &AuthResponse{
			Success:      false,
			ErrorCode:    "INVALID_TOKEN",
			ErrorMessage: "Invalid refresh token",
		}, nil
	}
	
	// Get session
	session, err := am.sessionStore.GetSession(ctx, claims.SessionID)
	if err != nil || !session.Active {
		return &AuthResponse{
			Success:      false,
			ErrorCode:    "SESSION_INVALID",
			ErrorMessage: "Session not found or inactive",
		}, nil
	}
	
	// Get user
	user, err := am.userStore.GetUserByID(ctx, session.UserID)
	if err != nil || !user.Active {
		return &AuthResponse{
			Success:      false,
			ErrorCode:    "USER_INVALID",
			ErrorMessage: "User not found or inactive",
		}, nil
	}
	
	// Generate new tokens
	accessToken, err := am.tokenManager.GenerateAccessToken(user, session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	
	var newRefreshToken string
	if am.config.TokenRotationEnabled {
		newRefreshToken, err = am.tokenManager.GenerateRefreshToken(user, session.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to generate refresh token: %w", err)
		}
		session.RefreshToken = newRefreshToken
	}
	
	// Update session
	session.AccessToken = accessToken
	session.LastAccessed = time.Now()
	session.ExpiresAt = time.Now().Add(am.config.AccessTokenExpiry)
	
	if err := am.sessionStore.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}
	
	// Get user permissions
	permissions := am.roleManager.GetUserPermissions(user.Roles)
	
	response := &AuthResponse{
		Success:      true,
		AccessToken:  accessToken,
		ExpiresAt:    session.ExpiresAt,
		Permissions:  permissions,
	}
	
	if newRefreshToken != "" {
		response.RefreshToken = newRefreshToken
	}
	
	return response, nil
}

// ValidateToken validates an access token and returns user information
func (am *AuthManager) ValidateToken(ctx context.Context, token string) (*User, error) {
	// Validate token
	claims, err := am.tokenManager.ValidateAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	
	// Get user
	user, err := am.userStore.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	
	if !user.Active {
		return nil, fmt.Errorf("user is inactive")
	}
	
	return user, nil
}

// Logout logs out a user by invalidating their session
func (am *AuthManager) Logout(ctx context.Context, sessionID string) error {
	return am.sessionStore.DeleteSession(ctx, sessionID)
}

// LogoutAll logs out all sessions for a user
func (am *AuthManager) LogoutAll(ctx context.Context, userID string) error {
	return am.sessionStore.DeleteUserSessions(ctx, userID)
}

// CreateUser creates a new user account
func (am *AuthManager) CreateUser(ctx context.Context, username, email, password string, roles []string) (*User, error) {
	// Validate password
	if err := am.validatePassword(password); err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}
	
	// Check if user exists
	if existing, _ := am.userStore.GetUser(ctx, username); existing != nil {
		return nil, fmt.Errorf("user already exists")
	}
	
	// Hash password
	salt := generateSalt()
	passwordHash, err := am.hashPassword(password, salt)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	
	// Create user
	user := &User{
		ID:           generateUserID(),
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Salt:         salt,
		Roles:        roles,
		Active:       true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}
	
	// Generate MFA secret if enabled
	if am.config.EnableMFA {
		user.MFASecret = generateMFASecret()
		user.BackupCodes = generateBackupCodes(am.config.BackupCodesCount)
	}
	
	if err := am.userStore.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	// Log user creation
	am.auditLogger.LogEvent(ctx, "user_created", map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"roles":    user.Roles,
	})
	
	return user, nil
}

// ChangePassword changes a user's password
func (am *AuthManager) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	// Get user
	user, err := am.userStore.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	
	// Verify old password
	if !am.verifyPassword(oldPassword, user.PasswordHash, user.Salt) {
		return fmt.Errorf("invalid current password")
	}
	
	// Validate new password
	if err := am.validatePassword(newPassword); err != nil {
		return fmt.Errorf("invalid new password: %w", err)
	}
	
	// Hash new password
	salt := generateSalt()
	passwordHash, err := am.hashPassword(newPassword, salt)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	
	// Update user
	user.PasswordHash = passwordHash
	user.Salt = salt
	user.UpdatedAt = time.Now()
	
	if err := am.userStore.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	
	// Log password change
	am.auditLogger.LogEvent(ctx, "password_changed", map[string]interface{}{
		"user_id": user.ID,
	})
	
	// Invalidate all sessions
	am.sessionStore.DeleteUserSessions(ctx, userID)
	
	return nil
}

// EnableMFA enables multi-factor authentication for a user
func (am *AuthManager) EnableMFA(ctx context.Context, userID string) (*MFASetup, error) {
	user, err := am.userStore.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	
	if user.MFAEnabled {
		return nil, fmt.Errorf("MFA already enabled")
	}
	
	// Generate MFA secret and backup codes
	secret := generateMFASecret()
	backupCodes := generateBackupCodes(am.config.BackupCodesCount)
	
	setup := &MFASetup{
		Secret:      secret,
		QRCode:      generateQRCode(user.Username, secret, am.config.TOTPIssuer),
		BackupCodes: backupCodes,
	}
	
	// Store temporarily until verified
	user.MFASecret = secret
	user.BackupCodes = backupCodes
	user.UpdatedAt = time.Now()
	
	if err := am.userStore.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	
	return setup, nil
}

// VerifyMFASetup verifies MFA setup and enables it
func (am *AuthManager) VerifyMFASetup(ctx context.Context, userID, code string) error {
	user, err := am.userStore.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	
	if user.MFAEnabled {
		return fmt.Errorf("MFA already enabled")
	}
	
	if !am.verifyTOTP(user.MFASecret, code) {
		return fmt.Errorf("invalid verification code")
	}
	
	// Enable MFA
	user.MFAEnabled = true
	user.UpdatedAt = time.Now()
	
	if err := am.userStore.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to enable MFA: %w", err)
	}
	
	// Log MFA enabled
	am.auditLogger.LogEvent(ctx, "mfa_enabled", map[string]interface{}{
		"user_id": user.ID,
	})
	
	return nil
}

// MFASetup represents MFA setup information
type MFASetup struct {
	Secret      string   `json:"secret"`
	QRCode      string   `json:"qr_code"`
	BackupCodes []string `json:"backup_codes"`
}

// Private methods

func (am *AuthManager) verifyPassword(password, hash, salt string) bool {
	expectedHash, err := am.hashPassword(password, salt)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(hash), []byte(expectedHash)) == 1
}

func (am *AuthManager) hashPassword(password, salt string) (string, error) {
	// Use Argon2id for password hashing
	hash := argon2.IDKey([]byte(password), []byte(salt), 1, 64*1024, 4, 32)
	return base64.StdEncoding.EncodeToString(hash), nil
}

func (am *AuthManager) validatePassword(password string) error {
	if len(password) < am.config.PasswordMinLength {
		return fmt.Errorf("password must be at least %d characters", am.config.PasswordMinLength)
	}
	
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", r):
			hasSpecial = true
		}
	}
	
	if am.config.PasswordRequireUpper && !hasUpper {
		return fmt.Errorf("password must contain uppercase letters")
	}
	if am.config.PasswordRequireLower && !hasLower {
		return fmt.Errorf("password must contain lowercase letters")
	}
	if am.config.PasswordRequireDigit && !hasDigit {
		return fmt.Errorf("password must contain digits")
	}
	if am.config.PasswordRequireSpecial && !hasSpecial {
		return fmt.Errorf("password must contain special characters")
	}
	
	return nil
}

func (am *AuthManager) verifyMFA(user *User, code string) bool {
	// Try TOTP first
	if am.verifyTOTP(user.MFASecret, code) {
		return true
	}
	
	// Try backup codes
	for i, backupCode := range user.BackupCodes {
		if subtle.ConstantTimeCompare([]byte(code), []byte(backupCode)) == 1 {
			// Remove used backup code
			user.BackupCodes = append(user.BackupCodes[:i], user.BackupCodes[i+1:]...)
			am.userStore.UpdateUser(context.Background(), user)
			return true
		}
	}
	
	return false
}

func (am *AuthManager) verifyTOTP(secret, code string) bool {
	// Simplified TOTP verification - in production would use proper TOTP library
	return len(code) == 6 && code != ""
}

func (am *AuthManager) handleFailedAuth(ctx context.Context, user *User, req *AuthRequest) {
	user.FailedAttempts++
	
	if user.FailedAttempts >= am.config.LoginAttempts {
		user.LockedUntil = time.Now().Add(am.config.LockoutDuration)
	}
	
	user.UpdatedAt = time.Now()
	am.userStore.UpdateUser(ctx, user)
	
	am.logFailedAuth(ctx, req, "INVALID_CREDENTIALS", "Invalid credentials")
}

func (am *AuthManager) logFailedAuth(ctx context.Context, req *AuthRequest, errorCode, errorMessage string) {
	if am.config.LogFailedAttempts {
		am.auditLogger.LogEvent(ctx, "auth_failed", map[string]interface{}{
			"username":      req.Username,
			"ip_address":    req.IPAddress,
			"user_agent":    req.UserAgent,
			"error_code":    errorCode,
			"error_message": errorMessage,
		})
	}
}

func (am *AuthManager) createSuccessfulAuth(ctx context.Context, user *User, req *AuthRequest) (*AuthResponse, error) {
	// Reset failed attempts
	user.FailedAttempts = 0
	user.LockedUntil = time.Time{}
	user.LastLogin = time.Now()
	user.UpdatedAt = time.Now()
	
	if err := am.userStore.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	
	// Create session
	session := &Session{
		ID:        generateSessionID(),
		UserID:    user.ID,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		Active:    true,
		CreatedAt: time.Now(),
		LastAccessed: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	// Set expiry based on remember me
	if req.RememberMe {
		session.ExpiresAt = time.Now().Add(am.config.RefreshTokenExpiry)
	} else {
		session.ExpiresAt = time.Now().Add(am.config.SessionTimeout)
	}
	
	// Generate tokens
	accessToken, err := am.tokenManager.GenerateAccessToken(user, session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	
	refreshToken, err := am.tokenManager.GenerateRefreshToken(user, session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	
	session.AccessToken = accessToken
	session.RefreshToken = refreshToken
	
	// Check concurrent sessions limit
	if am.config.MaxConcurrentSessions > 0 {
		sessions, _ := am.sessionStore.ListActiveSessions(ctx, user.ID)
		if len(sessions) >= am.config.MaxConcurrentSessions {
			// Remove oldest session
			oldestSession := sessions[0]
			for _, s := range sessions {
				if s.CreatedAt.Before(oldestSession.CreatedAt) {
					oldestSession = s
				}
			}
			am.sessionStore.DeleteSession(ctx, oldestSession.ID)
		}
	}
	
	if err := am.sessionStore.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	
	// Get permissions
	permissions := am.roleManager.GetUserPermissions(user.Roles)
	
	// Log successful authentication
	if am.config.LogSuccessfulAuth {
		am.auditLogger.LogEvent(ctx, "auth_successful", map[string]interface{}{
			"user_id":    user.ID,
			"username":   user.Username,
			"ip_address": req.IPAddress,
			"user_agent": req.UserAgent,
		})
	}
	
	return &AuthResponse{
		Success:      true,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    session.ExpiresAt,
		User:         user,
		Permissions:  permissions,
	}, nil
}

// Utility functions

func generateSecureSecret() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func generateSalt() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

func generateUserID() string {
	return fmt.Sprintf("user_%d", time.Now().UnixNano())
}

func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

func generateMFASecret() string {
	bytes := make([]byte, 20)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

func generateBackupCodes(count int) []string {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		bytes := make([]byte, 6)
		rand.Read(bytes)
		codes[i] = fmt.Sprintf("%X", bytes)[:8]
	}
	return codes
}

func generateQRCode(username, secret, issuer string) string {
	// Simplified QR code generation - in production would generate actual QR code
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", issuer, username, secret, issuer)
}