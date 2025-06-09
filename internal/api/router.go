// Package api provides the HTTP API layer for the MCP Memory Server.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"lerian-mcp-memory/internal/api/handlers"
	"lerian-mcp-memory/internal/api/middleware"
	"lerian-mcp-memory/internal/config"
)

// Router represents the main API router
type Router struct {
	config  *config.Config
	mux     *chi.Mux
	version string
}

// NewRouter creates a new API router with middleware and routes
func NewRouter(cfg *config.Config) *Router {
	r := &Router{
		config:  cfg,
		mux:     chi.NewRouter(),
		version: "1.0.0",
	}

	r.setupMiddleware()
	r.setupRoutes()

	return r
}

// Handler returns the HTTP handler
func (r *Router) Handler() http.Handler {
	return r.mux
}

// setupMiddleware configures the middleware stack
func (r *Router) setupMiddleware() {
	// Recovery middleware (should be first)
	r.mux.Use(chimiddleware.Recoverer)

	// Request timeout middleware
	r.mux.Use(chimiddleware.Timeout(30 * time.Second))

	// Logging middleware
	loggingMiddleware := middleware.NewLoggingMiddleware()
	r.mux.Use(loggingMiddleware.Handler())

	// CORS middleware
	corsMiddleware := r.createCORSMiddleware()
	r.mux.Use(corsMiddleware.Handler())

	// Version checking middleware
	versionMiddleware := middleware.NewVersionChecker()
	r.mux.Use(versionMiddleware.Handler())

	// Request size limit (10MB)
	r.mux.Use(chimiddleware.RequestSize(10 * 1024 * 1024))

	// Heartbeat for load balancer health checks
	r.mux.Use(chimiddleware.Heartbeat("/ping"))
}

// createCORSMiddleware creates appropriate CORS middleware based on environment
func (r *Router) createCORSMiddleware() *middleware.CORSMiddleware {
	// In development, use permissive CORS
	if r.isDevEnvironment() {
		return middleware.NewDefaultCORSMiddleware()
	}

	// In production, use strict CORS (should be configured via environment)
	allowedOrigins := []string{
		"https://app.lerian.ai",
		"https://lerian.ai",
	}
	
	return middleware.NewProductionCORSMiddleware(allowedOrigins)
}

// isDevEnvironment checks if running in development environment
func (r *Router) isDevEnvironment() bool {
	return r.config.Server.Host == "localhost" || r.config.Server.Host == "127.0.0.1"
}

// setupRoutes configures API routes
func (r *Router) setupRoutes() {
	// Health check endpoint (no version prefix for load balancers)
	healthHandler := handlers.NewHealthHandler(r.config)
	r.mux.Get("/health", healthHandler.Handle)

	// API v1 routes
	r.mux.Route("/api/v1", func(rtr chi.Router) {
		// Health check with version prefix
		rtr.Get("/health", healthHandler.Handle)

		// Future API endpoints will be added here in subsequent sub-tasks
		// rtr.Post("/prd/import", ...) // ST-006-03
		// rtr.Post("/tasks/suggest", ...) // ST-006-04
		// rtr.Route("/tasks", ...) // ST-006-05
	})

	// Root endpoint with server info
	r.mux.Get("/", r.handleRoot)

	// 404 handler
	r.mux.NotFound(r.handleNotFound)

	// 405 handler
	r.mux.MethodNotAllowed(r.handleMethodNotAllowed)
}

// handleRoot handles requests to the root endpoint
func (r *Router) handleRoot(w http.ResponseWriter, req *http.Request) {
	serverInfo := map[string]interface{}{
		"server":      "lerian-mcp-memory",
		"version":     r.version,
		"api_version": "v1",
		"endpoints": map[string]string{
			"health":    "/health",
			"api":       "/api/v1",
			"docs":      "/docs",
			"openapi":   "/api/v1/openapi.json",
		},
		"protocols": []string{"HTTP", "WebSocket"},
		"status":    "running",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	// Use the response package for consistent formatting
	// Import would be added: "lerian-mcp-memory/internal/api/response"
	// For now, using a simple approach to avoid circular imports
	if err := writeJSON(w, serverInfo); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleNotFound handles 404 errors
func (r *Router) handleNotFound(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	
	errorResp := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    "NOT_FOUND",
			"message": "Endpoint not found",
			"details": "The requested resource does not exist",
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	writeJSON(w, errorResp)
}

// handleMethodNotAllowed handles 405 errors
func (r *Router) handleMethodNotAllowed(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	
	errorResp := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    "METHOD_NOT_ALLOWED",
			"message": "Method not allowed",
			"details": "The HTTP method is not supported for this endpoint",
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	
	writeJSON(w, errorResp)
}

// writeJSON writes JSON response
func writeJSON(w http.ResponseWriter, data interface{}) error {
	return json.NewEncoder(w).Encode(data)
}

// GetServerConfig returns the server configuration for external access
func (r *Router) GetServerConfig() *config.Config {
	return r.config
}

// WithContext adds context to the router (useful for dependency injection)
func (r *Router) WithContext(ctx context.Context) *Router {
	// Future enhancement: add context for request-scoped dependencies
	return r
}