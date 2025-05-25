package middleware

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Common errors
var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
	ErrInvalidAPIKey    = errors.New("invalid API key")
	ErrMissingAuthHeader = errors.New("missing authorization header")
)

// Context keys for authentication
type contextKey string

const (
	// ContextKeyUser stores the authenticated user information
	ContextKeyUser contextKey = "auth:user"
	// ContextKeyAuthMethod stores the authentication method used
	ContextKeyAuthMethod contextKey = "auth:method"
	// ContextKeyTokenClaims stores JWT claims
	ContextKeyTokenClaims contextKey = "auth:claims"
)

// AuthMethod represents the authentication method used
type AuthMethod string

const (
	AuthMethodJWT    AuthMethod = "jwt"
	AuthMethodAPIKey AuthMethod = "api_key"
)

// User represents an authenticated user
type User struct {
	ID       string
	Username string
	Email    string
	Roles    []string
	Metadata map[string]interface{}
}

// AuthConfig contains configuration for authentication middleware
type AuthConfig struct {
	// JWT configuration
	JWTSecret       string
	JWTIssuer       string
	JWTAudience     []string
	JWTExpiration   time.Duration
	JWTClockSkew    time.Duration

	// API Key configuration
	APIKeys         map[string]*User // API key -> User mapping
	APIKeyHeader    string           // Header name for API key (default: X-API-Key)

	// General configuration
	RequireAuth     bool             // Whether authentication is required
	AllowedMethods  []AuthMethod     // Allowed authentication methods
	Logger          *slog.Logger
}

// DefaultAuthConfig returns a default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		JWTExpiration:  24 * time.Hour,
		JWTClockSkew:   5 * time.Minute,
		APIKeyHeader:   "X-API-Key",
		RequireAuth:    true,
		AllowedMethods: []AuthMethod{AuthMethodJWT, AuthMethodAPIKey},
		APIKeys:        make(map[string]*User),
		Logger:         slog.Default(),
	}
}

// AuthMiddleware provides authentication functionality
type AuthMiddleware struct {
	config *AuthConfig
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config *AuthConfig) *AuthMiddleware {
	if config == nil {
		config = DefaultAuthConfig()
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if config.APIKeyHeader == "" {
		config.APIKeyHeader = "X-API-Key"
	}
	if len(config.AllowedMethods) == 0 {
		config.AllowedMethods = []AuthMethod{AuthMethodJWT, AuthMethodAPIKey}
	}
	
	return &AuthMiddleware{
		config: config,
	}
}

// Process implements the Middleware interface
func (m *AuthMiddleware) Process(ctx context.Context, request interface{}, next func(context.Context, interface{}) (interface{}, error)) (interface{}, error) {
	// Extract authentication information from context
	authHeader := m.extractAuthHeader(ctx)
	
	if authHeader == "" && m.config.RequireAuth {
		m.config.Logger.WarnContext(ctx, "missing authentication header")
		return nil, ErrMissingAuthHeader
	}
	
	// Try different authentication methods
	var user *User
	var authMethod AuthMethod
	var err error
	
	// Try JWT authentication
	if m.isMethodAllowed(AuthMethodJWT) && strings.HasPrefix(authHeader, "Bearer ") {
		user, err = m.authenticateJWT(ctx, strings.TrimPrefix(authHeader, "Bearer "))
		if err == nil {
			authMethod = AuthMethodJWT
		}
	}
	
	// Try API key authentication
	if user == nil && m.isMethodAllowed(AuthMethodAPIKey) {
		apiKey := m.extractAPIKey(ctx)
		if apiKey != "" {
			user, err = m.authenticateAPIKey(ctx, apiKey)
			if err == nil {
				authMethod = AuthMethodAPIKey
			}
		}
	}
	
	// Check if authentication was successful
	if user == nil && m.config.RequireAuth {
		m.config.Logger.WarnContext(ctx, "authentication failed", "error", err)
		return nil, ErrUnauthorized
	}
	
	// Add user and auth method to context
	if user != nil {
		ctx = context.WithValue(ctx, ContextKeyUser, user)
		ctx = context.WithValue(ctx, ContextKeyAuthMethod, authMethod)
		
		m.config.Logger.InfoContext(ctx, "authenticated request",
			"user_id", user.ID,
			"method", authMethod)
	}
	
	// Call the next handler
	return next(ctx, request)
}

// authenticateJWT validates a JWT token and returns the user
func (m *AuthMiddleware) authenticateJWT(ctx context.Context, tokenString string) (*User, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.JWTSecret), nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	
	// Check if token is valid
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	
	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}
	
	// Store claims in context
	ctx = context.WithValue(ctx, ContextKeyTokenClaims, claims)
	
	// Validate standard claims
	if err := m.validateClaims(claims); err != nil {
		return nil, err
	}
	
	// Extract user information from claims
	user := &User{
		Metadata: make(map[string]interface{}),
	}
	
	// Extract standard fields
	if sub, ok := claims["sub"].(string); ok {
		user.ID = sub
	}
	if username, ok := claims["username"].(string); ok {
		user.Username = username
	}
	if email, ok := claims["email"].(string); ok {
		user.Email = email
	}
	
	// Extract roles
	if rolesInterface, ok := claims["roles"].([]interface{}); ok {
		roles := make([]string, 0, len(rolesInterface))
		for _, role := range rolesInterface {
			if roleStr, ok := role.(string); ok {
				roles = append(roles, roleStr)
			}
		}
		user.Roles = roles
	}
	
	// Store additional claims as metadata
	for key, value := range claims {
		if key != "sub" && key != "username" && key != "email" && key != "roles" &&
		   key != "iss" && key != "aud" && key != "exp" && key != "nbf" && key != "iat" {
			user.Metadata[key] = value
		}
	}
	
	return user, nil
}

// validateClaims validates JWT claims
func (m *AuthMiddleware) validateClaims(claims jwt.MapClaims) error {
	// Validate issuer
	if m.config.JWTIssuer != "" {
		if iss, ok := claims["iss"].(string); !ok || iss != m.config.JWTIssuer {
			return fmt.Errorf("invalid issuer")
		}
	}
	
	// Validate audience
	if len(m.config.JWTAudience) > 0 {
		audClaim, ok := claims["aud"]
		if !ok {
			return fmt.Errorf("missing audience claim")
		}
		
		// Handle both string and []string audience claims
		var audiences []string
		switch v := audClaim.(type) {
		case string:
			audiences = []string{v}
		case []interface{}:
			for _, aud := range v {
				if audStr, ok := aud.(string); ok {
					audiences = append(audiences, audStr)
				}
			}
		default:
			return fmt.Errorf("invalid audience claim format")
		}
		
		// Check if any configured audience matches
		found := false
		for _, configAud := range m.config.JWTAudience {
			for _, tokenAud := range audiences {
				if configAud == tokenAud {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid audience")
		}
	}
	
	return nil
}

// authenticateAPIKey validates an API key and returns the user
func (m *AuthMiddleware) authenticateAPIKey(ctx context.Context, apiKey string) (*User, error) {
	// Constant time comparison to prevent timing attacks
	for key, user := range m.config.APIKeys {
		if subtle.ConstantTimeCompare([]byte(key), []byte(apiKey)) == 1 {
			return user, nil
		}
	}
	
	return nil, ErrInvalidAPIKey
}

// extractAuthHeader extracts the Authorization header from context
func (m *AuthMiddleware) extractAuthHeader(ctx context.Context) string {
	// This would typically come from transport-specific context
	// For now, we'll check if it's stored in context
	if auth := ctx.Value("Authorization"); auth != nil {
		if authStr, ok := auth.(string); ok {
			return authStr
		}
	}
	return ""
}

// extractAPIKey extracts the API key from context
func (m *AuthMiddleware) extractAPIKey(ctx context.Context) string {
	// Check custom header
	if apiKey := ctx.Value(m.config.APIKeyHeader); apiKey != nil {
		if keyStr, ok := apiKey.(string); ok {
			return keyStr
		}
	}
	
	// Check Authorization header for API key
	auth := m.extractAuthHeader(ctx)
	if strings.HasPrefix(auth, "ApiKey ") {
		return strings.TrimPrefix(auth, "ApiKey ")
	}
	
	return ""
}

// isMethodAllowed checks if an authentication method is allowed
func (m *AuthMiddleware) isMethodAllowed(method AuthMethod) bool {
	for _, allowed := range m.config.AllowedMethods {
		if allowed == method {
			return true
		}
	}
	return false
}

// Helper functions for JWT token generation

// GenerateJWT generates a JWT token for a user
func GenerateJWT(user *User, config *AuthConfig) (string, error) {
	now := time.Now()
	
	// Create claims
	claims := jwt.MapClaims{
		"sub":      user.ID,
		"username": user.Username,
		"email":    user.Email,
		"roles":    user.Roles,
		"iat":      now.Unix(),
		"exp":      now.Add(config.JWTExpiration).Unix(),
	}
	
	// Add issuer if configured
	if config.JWTIssuer != "" {
		claims["iss"] = config.JWTIssuer
	}
	
	// Add audience if configured
	if len(config.JWTAudience) > 0 {
		claims["aud"] = config.JWTAudience
	}
	
	// Add metadata
	for key, value := range user.Metadata {
		claims[key] = value
	}
	
	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	// Sign token
	return token.SignedString([]byte(config.JWTSecret))
}

// GetUser extracts the authenticated user from context
func GetUser(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(ContextKeyUser).(*User)
	return user, ok
}

// GetAuthMethod extracts the authentication method from context
func GetAuthMethod(ctx context.Context) (AuthMethod, bool) {
	method, ok := ctx.Value(ContextKeyAuthMethod).(AuthMethod)
	return method, ok
}

// GetTokenClaims extracts JWT claims from context
func GetTokenClaims(ctx context.Context) (jwt.MapClaims, bool) {
	claims, ok := ctx.Value(ContextKeyTokenClaims).(jwt.MapClaims)
	return claims, ok
}