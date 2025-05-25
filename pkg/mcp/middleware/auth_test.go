package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	config := &AuthConfig{
		JWTSecret:      "test-secret",
		JWTIssuer:      "test-issuer",
		JWTAudience:    []string{"test-audience"},
		JWTExpiration:  1 * time.Hour,
		APIKeys: map[string]*User{
			"test-api-key": {
				ID:       "api-user",
				Username: "apiuser",
				Roles:    []string{"api"},
			},
		},
		RequireAuth:    true,
		AllowedMethods: []AuthMethod{AuthMethodJWT, AuthMethodAPIKey},
	}
	
	t.Run("missing auth header", func(t *testing.T) {
		middleware := NewAuthMiddleware(config)
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			t.Fatal("handler should not be called")
			return nil, nil
		}
		
		_, err := middleware.Process(context.Background(), "request", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrMissingAuthHeader, err)
	})
	
	t.Run("valid JWT authentication", func(t *testing.T) {
		middleware := NewAuthMiddleware(config)
		
		// Create a valid JWT token
		user := &User{
			ID:       "user123",
			Username: "testuser",
			Email:    "test@example.com",
			Roles:    []string{"admin", "user"},
		}
		
		token, err := GenerateJWT(user, config)
		require.NoError(t, err)
		
		// Add token to context
		ctx := context.WithValue(context.Background(), "Authorization", "Bearer "+token)
		
		var capturedUser *User
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			capturedUser, _ = GetUser(ctx)
			return "response", nil
		}
		
		resp, err := middleware.Process(ctx, "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		
		// Verify user was set in context
		require.NotNil(t, capturedUser)
		assert.Equal(t, user.ID, capturedUser.ID)
		assert.Equal(t, user.Username, capturedUser.Username)
		assert.Equal(t, user.Email, capturedUser.Email)
		assert.Equal(t, user.Roles, capturedUser.Roles)
	})
	
	t.Run("invalid JWT token", func(t *testing.T) {
		middleware := NewAuthMiddleware(config)
		
		ctx := context.WithValue(context.Background(), "Authorization", "Bearer invalid-token")
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			t.Fatal("handler should not be called")
			return nil, nil
		}
		
		_, err := middleware.Process(ctx, "request", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})
	
	t.Run("expired JWT token", func(t *testing.T) {
		middleware := NewAuthMiddleware(config)
		
		// Create an expired token
		claims := jwt.MapClaims{
			"sub":      "user123",
			"username": "testuser",
			"exp":      time.Now().Add(-1 * time.Hour).Unix(), // Expired
		}
		
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(config.JWTSecret))
		require.NoError(t, err)
		
		ctx := context.WithValue(context.Background(), "Authorization", "Bearer "+tokenString)
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			t.Fatal("handler should not be called")
			return nil, nil
		}
		
		_, err = middleware.Process(ctx, "request", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})
	
	t.Run("valid API key authentication", func(t *testing.T) {
		middleware := NewAuthMiddleware(config)
		
		ctx := context.WithValue(context.Background(), "X-API-Key", "test-api-key")
		
		var capturedUser *User
		var capturedMethod AuthMethod
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			capturedUser, _ = GetUser(ctx)
			capturedMethod, _ = GetAuthMethod(ctx)
			return "response", nil
		}
		
		resp, err := middleware.Process(ctx, "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		
		// Verify user was set in context
		require.NotNil(t, capturedUser)
		assert.Equal(t, "api-user", capturedUser.ID)
		assert.Equal(t, "apiuser", capturedUser.Username)
		assert.Equal(t, []string{"api"}, capturedUser.Roles)
		assert.Equal(t, AuthMethodAPIKey, capturedMethod)
	})
	
	t.Run("invalid API key", func(t *testing.T) {
		middleware := NewAuthMiddleware(config)
		
		ctx := context.WithValue(context.Background(), "X-API-Key", "invalid-key")
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			t.Fatal("handler should not be called")
			return nil, nil
		}
		
		_, err := middleware.Process(ctx, "request", handler)
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})
	
	t.Run("API key in Authorization header", func(t *testing.T) {
		middleware := NewAuthMiddleware(config)
		
		ctx := context.WithValue(context.Background(), "Authorization", "ApiKey test-api-key")
		
		var capturedUser *User
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			capturedUser, _ = GetUser(ctx)
			return "response", nil
		}
		
		resp, err := middleware.Process(ctx, "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		
		require.NotNil(t, capturedUser)
		assert.Equal(t, "api-user", capturedUser.ID)
	})
	
	t.Run("auth not required", func(t *testing.T) {
		config := &AuthConfig{
			RequireAuth: false,
		}
		middleware := NewAuthMiddleware(config)
		
		var capturedUser *User
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			capturedUser, _ = GetUser(ctx)
			return "response", nil
		}
		
		resp, err := middleware.Process(context.Background(), "request", handler)
		assert.NoError(t, err)
		assert.Equal(t, "response", resp)
		assert.Nil(t, capturedUser)
	})
	
	t.Run("method not allowed", func(t *testing.T) {
		config := &AuthConfig{
			JWTSecret:      "test-secret",
			RequireAuth:    true,
			AllowedMethods: []AuthMethod{AuthMethodJWT}, // Only JWT allowed
			APIKeys: map[string]*User{
				"test-api-key": {ID: "api-user"},
			},
		}
		middleware := NewAuthMiddleware(config)
		
		// Try API key when only JWT is allowed
		ctx := context.WithValue(context.Background(), "X-API-Key", "test-api-key")
		
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			t.Fatal("handler should not be called")
			return nil, nil
		}
		
		_, err := middleware.Process(ctx, "request", handler)
		assert.Error(t, err)
	})
}

func TestJWTValidation(t *testing.T) {
	config := &AuthConfig{
		JWTSecret:   "test-secret",
		JWTIssuer:   "test-issuer",
		JWTAudience: []string{"aud1", "aud2"},
	}
	
	middleware := NewAuthMiddleware(config)
	
	t.Run("valid issuer", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"iss": "test-issuer",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		
		err := middleware.validateClaims(claims)
		assert.NoError(t, err)
	})
	
	t.Run("invalid issuer", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"iss": "wrong-issuer",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		
		err := middleware.validateClaims(claims)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid issuer")
	})
	
	t.Run("valid audience (string)", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"aud": "aud1",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		
		err := middleware.validateClaims(claims)
		assert.NoError(t, err)
	})
	
	t.Run("valid audience (array)", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"aud": []interface{}{"aud1", "aud3"},
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		
		err := middleware.validateClaims(claims)
		assert.NoError(t, err)
	})
	
	t.Run("invalid audience", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"aud": "wrong-audience",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		
		err := middleware.validateClaims(claims)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid audience")
	})
	
	t.Run("missing audience", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(1 * time.Hour).Unix(),
		}
		
		err := middleware.validateClaims(claims)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing audience")
	})
}

func TestGenerateJWT(t *testing.T) {
	config := &AuthConfig{
		JWTSecret:     "test-secret",
		JWTIssuer:     "test-issuer",
		JWTAudience:   []string{"test-audience"},
		JWTExpiration: 1 * time.Hour,
	}
	
	user := &User{
		ID:       "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"admin", "user"},
		Metadata: map[string]interface{}{
			"custom": "value",
		},
	}
	
	tokenString, err := GenerateJWT(user, config)
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)
	
	// Parse and verify the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWTSecret), nil
	})
	require.NoError(t, err)
	require.True(t, token.Valid)
	
	// Verify claims
	claims, ok := token.Claims.(jwt.MapClaims)
	require.True(t, ok)
	
	assert.Equal(t, user.ID, claims["sub"])
	assert.Equal(t, user.Username, claims["username"])
	assert.Equal(t, user.Email, claims["email"])
	assert.Equal(t, config.JWTIssuer, claims["iss"])
	assert.Equal(t, config.JWTAudience, claims["aud"])
	assert.Equal(t, "value", claims["custom"])
	
	// Verify expiration
	exp, ok := claims["exp"].(float64)
	require.True(t, ok)
	expTime := time.Unix(int64(exp), 0)
	assert.True(t, expTime.After(time.Now()))
}

func TestContextHelpers(t *testing.T) {
	t.Run("GetUser", func(t *testing.T) {
		user := &User{ID: "123", Username: "test"}
		ctx := context.WithValue(context.Background(), ContextKeyUser, user)
		
		retrieved, ok := GetUser(ctx)
		assert.True(t, ok)
		assert.Equal(t, user, retrieved)
		
		// Test missing user
		retrieved, ok = GetUser(context.Background())
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})
	
	t.Run("GetAuthMethod", func(t *testing.T) {
		method := AuthMethodJWT
		ctx := context.WithValue(context.Background(), ContextKeyAuthMethod, method)
		
		retrieved, ok := GetAuthMethod(ctx)
		assert.True(t, ok)
		assert.Equal(t, method, retrieved)
		
		// Test missing method
		retrieved, ok = GetAuthMethod(context.Background())
		assert.False(t, ok)
		assert.Empty(t, retrieved)
	})
	
	t.Run("GetTokenClaims", func(t *testing.T) {
		claims := jwt.MapClaims{"sub": "123"}
		ctx := context.WithValue(context.Background(), ContextKeyTokenClaims, claims)
		
		retrieved, ok := GetTokenClaims(ctx)
		assert.True(t, ok)
		assert.Equal(t, claims, retrieved)
		
		// Test missing claims
		retrieved, ok = GetTokenClaims(context.Background())
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})
}

func BenchmarkAuthMiddleware(b *testing.B) {
	config := &AuthConfig{
		JWTSecret:      "test-secret",
		RequireAuth:    true,
		AllowedMethods: []AuthMethod{AuthMethodJWT},
	}
	
	middleware := NewAuthMiddleware(config)
	
	// Generate a valid token
	user := &User{ID: "user123", Username: "testuser"}
	token, _ := GenerateJWT(user, config)
	ctx := context.WithValue(context.Background(), "Authorization", "Bearer "+token)
	
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		middleware.Process(ctx, "request", handler)
	}
}

func BenchmarkGenerateJWT(b *testing.B) {
	config := &AuthConfig{
		JWTSecret:     "test-secret",
		JWTExpiration: 1 * time.Hour,
	}
	
	user := &User{
		ID:       "user123",
		Username: "testuser",
		Email:    "test@example.com",
		Roles:    []string{"admin", "user"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateJWT(user, config)
	}
}