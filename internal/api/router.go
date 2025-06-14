// Package api provides the HTTP API layer for the MCP Memory Server.
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq" // PostgreSQL driver

	"lerian-mcp-memory/internal/ai"
	"lerian-mcp-memory/internal/api/handlers"
	"lerian-mcp-memory/internal/api/middleware"
	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/storage"
	"lerian-mcp-memory/internal/tasks"
	"lerian-mcp-memory/pkg/types"
)

// Router represents the main API router
type Router struct {
	config        *config.Config
	mux           *chi.Mux
	version       string
	aiService     *ai.Service
	prdStorage    handlers.PRDStorage
	wsHandler     *handlers.WebSocketHandler
	shutdownFuncs []func(context.Context) error
}

// NewRouter creates a new API router with middleware and routes
//
//nolint:gocritic // ptrToRefParam: API consistency - aiService is consistently used as pointer throughout codebase
func NewRouter(cfg *config.Config, aiService *ai.Service, prdStorage handlers.PRDStorage) *Router {
	r := &Router{
		config:        cfg,
		mux:           chi.NewRouter(),
		version:       "1.0.0",
		aiService:     aiService,
		prdStorage:    prdStorage,
		shutdownFuncs: make([]func(context.Context) error, 0),
	}

	// Create WebSocket handler
	// Use standard logger for now - in production, inject a proper logger
	logger := log.New(os.Stdout, "[WebSocket] ", log.LstdFlags|log.Lshortfile)
	r.wsHandler = handlers.NewWebSocketHandler(cfg, logger)

	// Register shutdown function for WebSocket handler
	r.shutdownFuncs = append(r.shutdownFuncs, r.wsHandler.Shutdown)

	r.setupMiddleware()
	r.setupRoutes()

	return r
}

// NewBasicRouter creates a basic API router without AI services for backward compatibility
func NewBasicRouter(cfg *config.Config) *Router {
	return NewRouter(cfg, nil, nil)
}

// Handler returns the HTTP handler
func (r *Router) Handler() http.Handler {
	return r.mux
}

// setupMiddleware configures the middleware stack
func (r *Router) setupMiddleware() {
	// Recovery middleware (should be first)
	r.mux.Use(chimiddleware.Recoverer)

	// Request timeout middleware - exclude WebSocket endpoints
	r.mux.Use(r.timeoutMiddleware())

	// Logging middleware
	loggingMiddleware := middleware.NewLoggingMiddleware()
	r.mux.Use(loggingMiddleware.Handler())

	// CORS middleware
	corsMiddleware := r.createCORSMiddleware()
	r.mux.Use(corsMiddleware.Handler())

	// Version checking middleware
	versionMiddleware := middleware.NewVersionChecker()
	r.mux.Use(versionMiddleware.Handler())

	// Circuit breaker middleware (for production safety)
	if r.isCircuitBreakerEnabled() {
		circuitBreakerManager := r.createCircuitBreakerMiddleware()
		r.mux.Use(circuitBreakerManager.Middleware("api"))
	}

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

// isCircuitBreakerEnabled checks if circuit breaker middleware should be enabled
func (r *Router) isCircuitBreakerEnabled() bool {
	// Check environment variables for circuit breaker configuration
	return os.Getenv("MCP_MEMORY_CIRCUIT_BREAKER_ENABLED") == "true" ||
		os.Getenv("CIRCUIT_BREAKER_ENABLED") == "true"
}

// createCircuitBreakerMiddleware creates circuit breaker middleware with appropriate configuration
func (r *Router) createCircuitBreakerMiddleware() *middleware.CircuitBreakerManager {
	// Create circuit breaker configuration
	cbConfig := middleware.CircuitBreakerConfig{
		Enabled: true,
		DefaultSettings: middleware.BreakerConfig{
			FailureThreshold:  5,                // Open after 5 failures
			SuccessThreshold:  3,                // Close after 3 successes
			Timeout:           60 * time.Second, // Wait 60 seconds before trying again
			MaxRequests:       100,              // Allow 100 concurrent requests
			ResetTimeout:      30 * time.Second, // Reset after 30 seconds
			BackoffStrategy:   middleware.BackoffConstant,
			BackoffMultiplier: 1.5,
			MaxBackoffTime:    5 * time.Minute,
		},
		ServiceConfigs: map[string]middleware.BreakerConfig{
			"api": {
				FailureThreshold:  10, // More lenient for general API
				SuccessThreshold:  2,
				Timeout:           30 * time.Second,
				MaxRequests:       200,
				ResetTimeout:      15 * time.Second,
				BackoffStrategy:   middleware.BackoffLinear,
				BackoffMultiplier: 1.2,
				MaxBackoffTime:    2 * time.Minute,
			},
			"health": {
				FailureThreshold:  20, // Very lenient for health checks
				SuccessThreshold:  1,
				Timeout:           10 * time.Second,
				MaxRequests:       500,
				ResetTimeout:      5 * time.Second,
				BackoffStrategy:   middleware.BackoffConstant,
				BackoffMultiplier: 1.0,
				MaxBackoffTime:    30 * time.Second,
			},
		},
		MonitorInterval: 10 * time.Second,
		EnableMetrics:   true,
	}

	return middleware.NewCircuitBreakerManager(&cbConfig)
}

// timeoutMiddleware creates a timeout middleware that excludes WebSocket endpoints
func (r *Router) timeoutMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Skip timeout for WebSocket endpoints
			if strings.HasPrefix(req.URL.Path, "/ws") {
				next.ServeHTTP(w, req)
				return
			}

			// Apply timeout for other endpoints
			chimiddleware.Timeout(30*time.Second)(next).ServeHTTP(w, req)
		})
	}
}

// setupRoutes configures API routes
func (r *Router) setupRoutes() {
	// OAuth endpoints that provide working auth flow but don't require it for MCP operations
	r.mux.Get("/.well-known/oauth-authorization-server", r.handleNoAuthDiscovery)
	r.mux.Post("/register", r.handleNoAuthRegistration)
	r.mux.Post("/token", r.handleNoAuthToken)
	r.mux.Get("/authorize", r.handleNoAuthAuthorize)
	r.mux.Post("/authorize", r.handleNoAuthAuthorize)

	// Health check endpoints (no version prefix for load balancers)
	healthHandler := handlers.NewHealthHandler(r.config)
	r.mux.Get("/health", healthHandler.Handle)
	r.mux.Get("/readiness", healthHandler.HandleReadiness)
	r.mux.Get("/liveness", healthHandler.HandleLiveness)

	// API v1 routes
	r.mux.Route("/api/v1", func(rtr chi.Router) {
		// Health check endpoints with version prefix
		rtr.Get("/health", healthHandler.Handle)
		rtr.Get("/readiness", healthHandler.HandleReadiness)
		rtr.Get("/liveness", healthHandler.HandleLiveness)

		// AI endpoints for document generation - REMOVED
		// These HTTP endpoints are no longer needed as CLI now uses shared AI package directly
		// This eliminates the HTTP dependency and improves performance
		// Server AI functionality is still available through MCP protocol
		/* REMOVED: AI HTTP endpoints
		if r.aiService != nil {
			aiHandler := handlers.NewAIHandler(r.aiService)
			rtr.Route("/ai", func(aiRouter chi.Router) {
				aiRouter.Post("/generate/prd", aiHandler.GeneratePRD)
				aiRouter.Post("/generate/trd", aiHandler.GenerateTRD)
				aiRouter.Post("/generate/main-tasks", aiHandler.GenerateMainTasks)
				aiRouter.Post("/generate/sub-tasks", aiHandler.GenerateSubTasks)
				aiRouter.Post("/analyze/content", aiHandler.AnalyzeContent)
				aiRouter.Post("/analyze/complexity", aiHandler.EstimateComplexity)
				aiRouter.Post("/session/start", aiHandler.StartInteractiveSession)
				aiRouter.Post("/session/{id}/continue", aiHandler.ContinueSession)
				aiRouter.Post("/session/{id}/end", aiHandler.EndSession)
				aiRouter.Get("/models", aiHandler.GetAvailableModels)
			})
		}
		*/

		// PRD endpoints (from ST-006-03)
		if r.prdStorage != nil {
			prdHandler := handlers.NewPRDHandler(r.prdStorage, handlers.DefaultPRDHandlerConfig())
			rtr.Route("/prd", func(prdRouter chi.Router) {
				prdRouter.Post("/import", prdHandler.ImportPRD)
				prdRouter.Get("/", prdHandler.ListPRDs)
				prdRouter.Get("/{id}", prdHandler.GetPRD)
				prdRouter.Put("/{id}", prdHandler.UpdatePRD)
				prdRouter.Delete("/{id}", prdHandler.DeletePRD)
			})
		}

		// Task endpoints (ST-006-04 & ST-006-05) - temporarily disabled due to AI interface conflicts
		// if r.aiService != nil && r.prdStorage != nil {
		//	// AI-powered task generation endpoints (ST-006-04)
		//	taskHandler := handlers.NewTaskHandler(r.aiService, r.prdStorage, handlers.DefaultTaskHandlerConfig())
		//	rtr.Route("/tasks", func(taskRouter chi.Router) {
		//		// AI generation endpoints
		//		taskRouter.Post("/suggest", taskHandler.SuggestTasks)
		//		taskRouter.Post("/generate-from-prd", taskHandler.GenerateFromPRD)
		//		taskRouter.Post("/contextual-suggestions", taskHandler.GetTaskSuggestions)
		//		taskRouter.Post("/validate", taskHandler.ValidateTask)
		//		taskRouter.Post("/score", taskHandler.ScoreTask)
		//	})
		// }

		// Enhanced task management endpoints (ST-006-05)
		// Note: In production, you'd inject a real task service with database connection
		if true { // Always enable CRUD operations
			// Create real task service with database connection
			// Using context.Background() for service initialization is appropriate here
			taskService, err := createRealTaskService(context.Background(), r.config)
			if err != nil {
				// Fall back to mock service if database connection fails
				taskService = createMockTaskService()
			}

			// CRUD operations
			crudHandler := handlers.NewTaskCRUDHandler(taskService, handlers.DefaultTaskCRUDConfig())
			rtr.Route("/tasks", func(taskRouter chi.Router) {
				taskRouter.Get("/", crudHandler.ListTasks)
				taskRouter.Post("/", crudHandler.CreateTask)
				taskRouter.Get("/{id}", crudHandler.GetTask)
				taskRouter.Put("/{id}", crudHandler.UpdateTask)
				taskRouter.Delete("/{id}", crudHandler.DeleteTask)
				taskRouter.Get("/metrics", crudHandler.GetTaskMetrics)
			})

			// Search operations
			searchHandler := handlers.NewTaskSearchHandler(taskService, handlers.DefaultTaskSearchConfig())
			rtr.Route("/tasks/search", func(searchRouter chi.Router) {
				searchRouter.Get("/", searchHandler.SearchTasks)
				searchRouter.Post("/advanced", searchHandler.AdvancedSearch)
				searchRouter.Get("/suggestions", searchHandler.GetSearchSuggestions)
				searchRouter.Get("/history", searchHandler.GetSearchHistory)
			})

			// Batch operations
			batchHandler := handlers.NewTaskBatchHandler(taskService, handlers.DefaultTaskBatchConfig())
			rtr.Route("/tasks/batch", func(batchRouter chi.Router) {
				batchRouter.Post("/update", batchHandler.BatchUpdate)
				batchRouter.Post("/create", batchHandler.BatchCreate)
				batchRouter.Post("/delete", batchHandler.BatchDelete)
				batchRouter.Post("/status-transition", batchHandler.BatchStatusTransition)
				batchRouter.Get("/status", batchHandler.GetBatchOperationStatus)
			})
		}
	})

	// WebSocket routes
	r.mux.Route("/ws", func(wsRouter chi.Router) {
		// WebSocket upgrade endpoint - must be at /ws root
		wsRouter.HandleFunc("/", r.wsHandler.HandleUpgrade)

		// WebSocket management endpoints
		wsRouter.Get("/status", r.wsHandler.HandleStatus)
		wsRouter.Get("/metrics", r.wsHandler.HandleMetrics)
		wsRouter.Post("/broadcast", r.wsHandler.HandleBroadcast)
		wsRouter.Get("/health", r.wsHandler.HandleHealthCheck)
		wsRouter.Get("/connections", r.wsHandler.HandleConnectionInfo)
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
	endpoints := map[string]string{
		"health":    "/health",
		"readiness": "/readiness",
		"liveness":  "/liveness",
		"api":       "/api/v1",
		"docs":      "/docs",
		"openapi":   "/api/v1/openapi.json",
		"websocket": "/ws",
	}

	// Add available endpoints based on configured services
	if r.prdStorage != nil {
		endpoints["prd_import"] = "/api/v1/prd/import"
		endpoints["prd_list"] = "/api/v1/prd"
	}

	if r.aiService != nil && r.prdStorage != nil {
		endpoints["task_suggest"] = "/api/v1/tasks/suggest"
		endpoints["task_generate"] = "/api/v1/tasks/generate-from-prd"
		endpoints["task_validate"] = "/api/v1/tasks/validate"
		endpoints["task_score"] = "/api/v1/tasks/score"
	}

	serverInfo := map[string]interface{}{
		"server":      "lerian-mcp-memory",
		"version":     r.version,
		"api_version": "v1",
		"endpoints":   endpoints,
		"protocols":   []string{"HTTP", "WebSocket"},
		"status":      "running",
		"features": map[string]bool{
			"ai_task_generation": r.aiService != nil,
			"prd_processing":     r.prdStorage != nil,
			"task_suggestions":   r.aiService != nil && r.prdStorage != nil,
			"websocket":          r.wsHandler != nil,
			"ai_http_endpoints":  false, // HTTP AI endpoints removed - use shared AI package
		},
		"websocket": map[string]interface{}{
			"available": r.wsHandler != nil,
			"endpoints": map[string]string{
				"upgrade":     "/ws",
				"status":      "/ws/status",
				"metrics":     "/ws/metrics",
				"broadcast":   "/ws/broadcast",
				"health":      "/ws/health",
				"connections": "/ws/connections",
			},
		},
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

	if err := writeJSON(w, errorResp); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
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

	if err := writeJSON(w, errorResp); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
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

// Stop gracefully shuts down all router components
func (r *Router) Stop(ctx context.Context) error {
	var errs []error

	// Execute all shutdown functions
	for _, shutdownFunc := range r.shutdownFuncs {
		if err := shutdownFunc(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	// Return combined error if any
	if len(errs) > 0 {
		return fmt.Errorf("router shutdown errors: %v", errs)
	}

	return nil
}

// GetWebSocketHandler returns the WebSocket handler for external access
func (r *Router) GetWebSocketHandler() *handlers.WebSocketHandler {
	return r.wsHandler
}

// createMockTaskService creates a mock task service for development/testing
// In production, this would be replaced with a real service with database connection
func createMockTaskService() *tasks.Service {
	// Create a mock repository that implements TaskRepository interface
	mockRepo := &MockTaskRepository{}

	// Create service with mock repository
	serviceConfig := tasks.DefaultServiceConfig()
	service := tasks.NewService(mockRepo, serviceConfig)

	return service
}

// MockTaskRepository provides a mock implementation of TaskRepository for development
type MockTaskRepository struct {
	tasks map[string]*types.Task
}

// Create implements TaskRepository.Create
func (m *MockTaskRepository) Create(ctx context.Context, task *types.Task) error {
	if m.tasks == nil {
		m.tasks = make(map[string]*types.Task)
	}

	// Generate a simple ID if not set
	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d", len(m.tasks)+1)
	}

	m.tasks[task.ID] = task
	return nil
}

// GetByID implements TaskRepository.GetByID
func (m *MockTaskRepository) GetByID(ctx context.Context, id string) (*types.Task, error) {
	if m.tasks == nil {
		return nil, errors.New("task not found")
	}

	task, exists := m.tasks[id]
	if !exists {
		return nil, errors.New("task not found")
	}

	return task, nil
}

// Update implements TaskRepository.Update
func (m *MockTaskRepository) Update(ctx context.Context, task *types.Task) error {
	if m.tasks == nil {
		return errors.New("task not found")
	}

	if _, exists := m.tasks[task.ID]; !exists {
		return errors.New("task not found")
	}

	m.tasks[task.ID] = task
	return nil
}

// Delete implements TaskRepository.Delete
func (m *MockTaskRepository) Delete(ctx context.Context, id string) error {
	if m.tasks == nil {
		return errors.New("task not found")
	}

	if _, exists := m.tasks[id]; !exists {
		return errors.New("task not found")
	}

	delete(m.tasks, id)
	return nil
}

// List implements TaskRepository.List
func (m *MockTaskRepository) List(ctx context.Context, filters *tasks.TaskFilters) ([]types.Task, error) {
	if m.tasks == nil {
		return []types.Task{}, nil
	}

	result := m.filterTasks(filters)
	result = m.applyPagination(result, filters)

	return result, nil
}

// filterTasks applies filtering logic to tasks
func (m *MockTaskRepository) filterTasks(filters *tasks.TaskFilters) []types.Task {
	result := make([]types.Task, 0)
	for _, task := range m.tasks {
		if !m.matchesFilters(task, filters) {
			continue
		}
		result = append(result, *task)
	}
	return result
}

// matchesFilters checks if a task matches the given filters
func (m *MockTaskRepository) matchesFilters(task *types.Task, filters *tasks.TaskFilters) bool {
	return m.matchesStatusFilter(task, filters) &&
		m.matchesTypeFilter(task, filters) &&
		m.matchesAssigneeFilter(task, filters)
}

// matchesStatusFilter checks status filter
func (m *MockTaskRepository) matchesStatusFilter(task *types.Task, filters *tasks.TaskFilters) bool {
	if len(filters.Status) == 0 {
		return true
	}
	for _, status := range filters.Status {
		if task.Status == status {
			return true
		}
	}
	return false
}

// matchesTypeFilter checks type filter
func (m *MockTaskRepository) matchesTypeFilter(task *types.Task, filters *tasks.TaskFilters) bool {
	if len(filters.Type) == 0 {
		return true
	}
	for _, taskType := range filters.Type {
		if task.Type == taskType {
			return true
		}
	}
	return false
}

// matchesAssigneeFilter checks assignee filter
func (m *MockTaskRepository) matchesAssigneeFilter(task *types.Task, filters *tasks.TaskFilters) bool {
	return filters.Assignee == "" || task.Assignee == filters.Assignee
}

// applyPagination applies limit and offset to results
func (m *MockTaskRepository) applyPagination(result []types.Task, filters *tasks.TaskFilters) []types.Task {
	if filters.Offset > 0 && filters.Offset < len(result) {
		result = result[filters.Offset:]
	}
	if filters.Limit > 0 && filters.Limit < len(result) {
		result = result[:filters.Limit]
	}
	return result
}

// Search implements TaskRepository.Search
func (m *MockTaskRepository) Search(ctx context.Context, query *tasks.SearchQuery) (*tasks.SearchResults, error) {
	// Simple search implementation for mock
	allTasks, err := m.List(ctx, &query.Filters)
	if err != nil {
		return nil, err
	}

	// Filter by text query if provided
	var filteredTasks []types.Task
	if query.Query != "" {
		queryLower := strings.ToLower(query.Query)
		for i := range allTasks {
			task := &allTasks[i]
			if strings.Contains(strings.ToLower(task.Title), queryLower) ||
				strings.Contains(strings.ToLower(task.Description), queryLower) {
				filteredTasks = append(filteredTasks, *task)
			}
		}
	} else {
		filteredTasks = allTasks
	}

	// Apply max results limit
	if query.Options.MaxResults > 0 && len(filteredTasks) > query.Options.MaxResults {
		filteredTasks = filteredTasks[:query.Options.MaxResults]
	}

	return &tasks.SearchResults{
		Tasks:        filteredTasks,
		TotalResults: len(filteredTasks),
		SearchTime:   time.Millisecond * 10, // Mock search time
	}, nil
}

// BatchUpdate implements TaskRepository.BatchUpdate
func (m *MockTaskRepository) BatchUpdate(ctx context.Context, updates []tasks.BatchUpdate) error {
	for _, update := range updates {
		task, err := m.GetByID(ctx, update.TaskID)
		if err != nil {
			return fmt.Errorf("failed to get task %s: %w", update.TaskID, err)
		}

		// Apply updates
		if update.Status != nil {
			task.Status = *update.Status
		}
		if update.Priority != nil {
			task.Priority = *update.Priority
		}
		if update.Assignee != nil {
			task.Assignee = *update.Assignee
		}
		if update.DueDate != nil {
			task.DueDate = update.DueDate
		}
		if len(update.Tags) > 0 {
			task.Tags = update.Tags
		}

		task.Timestamps.Updated = update.UpdatedAt

		if err := m.Update(ctx, task); err != nil {
			return fmt.Errorf("failed to update task %s: %w", update.TaskID, err)
		}
	}

	return nil
}

// GetByIDs implements TaskRepository.GetByIDs
func (m *MockTaskRepository) GetByIDs(ctx context.Context, ids []string) ([]types.Task, error) {
	if m.tasks == nil {
		return []types.Task{}, nil
	}

	result := make([]types.Task, 0)
	for _, id := range ids {
		if task, exists := m.tasks[id]; exists {
			result = append(result, *task)
		}
	}

	return result, nil
}

// createRealTaskService creates a real task service with database connection
func createRealTaskService(ctx context.Context, cfg *config.Config) (*tasks.Service, error) {
	// Connect to PostgreSQL database
	db, err := connectToDatabase(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create real task repository with database connection
	taskRepo := storage.NewTaskRepository(db)

	// Create service with real repository
	serviceConfig := tasks.DefaultServiceConfig()
	service := tasks.NewService(taskRepo, serviceConfig)

	return service, nil
}

// connectToDatabase establishes a connection to the PostgreSQL database
func connectToDatabase(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)

	// Test connection
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// handleNoAuthDiscovery signals to MCP clients that authentication is not required
func (r *Router) handleNoAuthDiscovery(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Use the actual server host and port - default to localhost:9080 for development
	baseURL := "http://localhost:9080"
	if r.config.Server.Host != "" && r.config.Server.Port > 0 {
		baseURL = fmt.Sprintf("http://%s:%d", r.config.Server.Host, r.config.Server.Port)
	}

	// Complete OAuth discovery response that includes required fields but signals optional auth
	metadata := map[string]interface{}{
		"issuer":                                baseURL,
		"authorization_endpoint":                baseURL + "/authorize",
		"token_endpoint":                        baseURL + "/token",
		"registration_endpoint":                 baseURL + "/register",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "client_credentials"},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_basic"},
		"scopes_supported":                      []string{"mcp", "memory:read", "memory:write"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
		"pkce_required":                         false, // PKCE is optional
		"authorization_required":                false, // Custom field: auth not required
		"authentication_required":               false, // Custom field: auth not required
		"service_documentation":                 "https://github.com/lerianstudio/lerian-mcp-memory",
	}

	if err := writeJSON(w, metadata); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleNoAuthRegistration provides working OAuth client registration
func (r *Router) handleNoAuthRegistration(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Return a valid OAuth registration response
	clientResponse := map[string]interface{}{
		"client_id":                  "mcp-client-" + time.Now().Format("20060102150405"),
		"client_secret":              "mcp-secret-not-required",
		"client_id_issued_at":        time.Now().Unix(),
		"client_secret_expires_at":   0, // Never expires
		"redirect_uris":              []string{},
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "client_credentials"},
		"response_types":             []string{"code"},
		"scope":                      "mcp memory:read memory:write",
		"authentication_required":    false, // Custom field
		"access_token_required":      false, // Custom field
	}

	w.WriteHeader(http.StatusCreated)
	if err := writeJSON(w, clientResponse); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleNoAuthToken provides working OAuth token endpoint
func (r *Router) handleNoAuthToken(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Return a valid OAuth token response
	tokenResponse := map[string]interface{}{
		"access_token":            "mcp-token-not-required-" + time.Now().Format("20060102150405"),
		"token_type":              "Bearer",
		"expires_in":              3600, // 1 hour
		"scope":                   "mcp memory:read memory:write",
		"refresh_token":           "mcp-refresh-not-required",
		"authentication_required": false, // Custom field
	}

	w.WriteHeader(http.StatusOK)
	if err := writeJSON(w, tokenResponse); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleNoAuthAuthorize provides working OAuth authorization endpoint
func (r *Router) handleNoAuthAuthorize(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse query parameters
	responseType := req.URL.Query().Get("response_type")
	clientID := req.URL.Query().Get("client_id")
	redirectURI := req.URL.Query().Get("redirect_uri")
	scope := req.URL.Query().Get("scope")
	state := req.URL.Query().Get("state")
	codeChallenge := req.URL.Query().Get("code_challenge")
	codeChallengeMethod := req.URL.Query().Get("code_challenge_method")

	// Log for debugging
	_ = clientID
	_ = scope
	if codeChallenge != "" {
		if codeChallengeMethod == "" {
			codeChallengeMethod = "S256"
		}
		log.Printf("PKCE Challenge: %s, Method: %s", codeChallenge, codeChallengeMethod)
	}

	// Auto-approve all authorization requests
	if responseType == "code" {
		authCode := "mcp-auth-code-" + time.Now().Format("20060102150405")

		// If redirect URI provided, redirect with the code
		if redirectURI != "" {
			redirectURL := redirectURI + "?code=" + authCode
			if state != "" {
				redirectURL += "&state=" + state
			}
			http.Redirect(w, req, redirectURL, http.StatusFound)
			return
		}

		// Otherwise return code directly
		response := map[string]interface{}{
			"code":  authCode,
			"state": state,
		}

		w.WriteHeader(http.StatusOK)
		if err := writeJSON(w, response); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Unsupported response type
	http.Error(w, "Unsupported response type", http.StatusBadRequest)
}
