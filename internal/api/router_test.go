package api

import (
	"lerian-mcp-memory/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test constants
const (
	TestAPIKey = "testAPIKey"
)

// Removed unused constant

func TestNewRouter(t *testing.T) {
	cfg := config.DefaultConfig()
	router := NewBasicRouter(cfg)

	assert.NotNil(t, router)
	assert.NotNil(t, router.Handler())
	assert.Equal(t, "1.0.0", router.version)
}

func TestHealthEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	// Set a dummy API key for testing
	cfg.OpenAI.APIKey = TestAPIKey
	router := NewBasicRouter(cfg)

	req := httptest.NewRequest("GET", "/health", http.NoBody)
	w := httptest.NewRecorder()

	router.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestAPIV1HealthEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	// Set a dummy API key for testing
	cfg.OpenAI.APIKey = TestAPIKey
	router := NewBasicRouter(cfg)

	req := httptest.NewRequest("GET", "/api/v1/health", http.NoBody)
	w := httptest.NewRecorder()

	router.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestRootEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	router := NewBasicRouter(cfg)

	req := httptest.NewRequest("GET", "/", http.NoBody)
	w := httptest.NewRecorder()

	router.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestVersionMiddleware(t *testing.T) {
	cfg := config.DefaultConfig()
	// Set a dummy API key for testing
	cfg.OpenAI.APIKey = TestAPIKey
	router := NewBasicRouter(cfg)

	// Test with supported version
	req := httptest.NewRequest("GET", "/api/v1/health", http.NoBody)
	req.Header.Set("X-Client-Version", "1.0.0")
	w := httptest.NewRecorder()

	router.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Server-Version"))
}

func TestCORSMiddleware(t *testing.T) {
	cfg := config.DefaultConfig()
	router := NewBasicRouter(cfg)

	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "/api/v1/health", http.NoBody)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	router.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestNotFoundHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	router := NewBasicRouter(cfg)

	req := httptest.NewRequest("GET", "/nonexistent", http.NoBody)
	w := httptest.NewRecorder()

	router.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestMethodNotAllowed(t *testing.T) {
	cfg := config.DefaultConfig()
	router := NewBasicRouter(cfg)

	req := httptest.NewRequest("PATCH", "/health", http.NoBody)
	w := httptest.NewRecorder()

	router.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}
