package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"lerian-mcp-memory/internal/config"
)

func TestHealthHandler_Handle(t *testing.T) {
	// Create minimal config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 9080,
		},
	}

	// Create health handler
	handler := NewHealthHandler(cfg)

	// Create test request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.Handle(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got '%s'", contentType)
	}

	// Check that body is not empty
	if w.Body.Len() == 0 {
		t.Error("Expected non-empty response body")
	}
}

func TestHealthHandler_HandleReadiness(t *testing.T) {
	// Create minimal config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 9080,
		},
	}

	// Create health handler
	handler := NewHealthHandler(cfg)

	// Create test request
	req := httptest.NewRequest("GET", "/readiness", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.HandleReadiness(w, req)

	// Check response - readiness should return 200 for healthy service
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got '%s'", contentType)
	}
}

func TestHealthHandler_HandleLiveness(t *testing.T) {
	// Create minimal config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 9080,
		},
	}

	// Create health handler
	handler := NewHealthHandler(cfg)

	// Create test request
	req := httptest.NewRequest("GET", "/liveness", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.HandleLiveness(w, req)

	// Check response - liveness should always return 200 if service is running
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check content type
	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected content type 'application/json', got '%s'", contentType)
	}
}
