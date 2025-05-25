// Package transport implements MCP transport layers
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"mcp-memory/pkg/mcp/protocol"
)

// RESTConfig contains configuration for REST API compatibility layer
type RESTConfig struct {
	// Base HTTP configuration
	HTTPConfig

	// API prefix (default: "/api/v1")
	APIPrefix string

	// Enable request/response logging
	EnableLogging bool

	// API key for authentication (optional)
	APIKey string

	// Rate limiting (requests per minute per IP)
	RateLimit int

	// Enable OpenAPI documentation
	EnableDocs bool
}

// RESTTransport implements a RESTful API compatibility layer for MCP
type RESTTransport struct {
	*HTTPTransport
	restConfig *RESTConfig
	rateLimiter *rateLimiter
}

// NewRESTTransport creates a new REST transport
func NewRESTTransport(config *RESTConfig) *RESTTransport {
	if config.APIPrefix == "" {
		config.APIPrefix = "/api/v1"
	}

	// Create base HTTP transport
	httpTransport := NewHTTPTransport(&config.HTTPConfig)

	transport := &RESTTransport{
		HTTPTransport: httpTransport,
		restConfig:    config,
	}

	if config.RateLimit > 0 {
		transport.rateLimiter = newRateLimiter(config.RateLimit)
	}

	return transport
}

// Start starts the REST transport
func (t *RESTTransport) Start(ctx context.Context, handler RequestHandler) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("transport already running")
	}

	t.handler = handler

	mux := http.NewServeMux()
	
	// Setup REST routes
	t.setupRoutes(mux)

	// OpenAPI documentation
	if t.restConfig.EnableDocs {
		mux.HandleFunc(t.restConfig.APIPrefix+"/docs", t.handleDocs)
		mux.HandleFunc(t.restConfig.APIPrefix+"/openapi.json", t.handleOpenAPISpec)
	}

	t.server = &http.Server{
		Addr:         t.config.Address,
		Handler:      t.wrapWithRESTMiddleware(mux),
		ReadTimeout:  t.config.ReadTimeout,
		WriteTimeout: t.config.WriteTimeout,
		IdleTimeout:  t.config.IdleTimeout,
		TLSConfig:    t.config.TLSConfig,
	}

	t.running = true

	// Start server
	go func() {
		var err error
		if t.certFile != "" && t.keyFile != "" {
			err = t.server.ListenAndServeTLS(t.certFile, t.keyFile)
		} else {
			err = t.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			t.mu.Lock()
			t.running = false
			t.mu.Unlock()
		}
	}()

	// Wait for context cancellation
	go func() {
		<-ctx.Done()
		t.Stop()
	}()

	return nil
}

// setupRoutes sets up REST API routes
func (t *RESTTransport) setupRoutes(mux *http.ServeMux) {
	prefix := t.restConfig.APIPrefix

	// Tools endpoints
	mux.HandleFunc(prefix+"/tools", t.handleTools)
	mux.HandleFunc(prefix+"/tools/", t.handleToolCall)

	// Resources endpoints
	mux.HandleFunc(prefix+"/resources", t.handleResources)
	mux.HandleFunc(prefix+"/resources/", t.handleResourceAccess)

	// Prompts endpoints
	mux.HandleFunc(prefix+"/prompts", t.handlePrompts)
	mux.HandleFunc(prefix+"/prompts/", t.handlePromptCall)

	// Server info
	mux.HandleFunc(prefix+"/info", t.handleServerInfo)

	// Health check
	mux.HandleFunc(prefix+"/health", t.handleHealth)
}

// handleTools handles GET /tools (list) and POST /tools (not applicable)
func (t *RESTTransport) handleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Convert to JSON-RPC request
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "rest-" + generateID(),
		Method:  "tools/list",
	}

	resp := t.handler.HandleRequest(r.Context(), req)
	t.writeRESTResponse(w, resp)
}

// handleToolCall handles POST /tools/{name}
func (t *RESTTransport) handleToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract tool name from path
	toolName := strings.TrimPrefix(r.URL.Path, t.restConfig.APIPrefix+"/tools/")
	if toolName == "" {
		http.Error(w, "Tool name required", http.StatusBadRequest)
		return
	}

	// Read body
	body, err := io.ReadAll(io.LimitReader(r.Body, t.config.MaxBodySize))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse arguments
	var args map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &args); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}

	// Convert to JSON-RPC request
	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "rest-" + generateID(),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		},
	}

	resp := t.handler.HandleRequest(r.Context(), req)
	t.writeRESTResponse(w, resp)
}

// handleResources handles GET /resources
func (t *RESTTransport) handleResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "rest-" + generateID(),
		Method:  "resources/list",
	}

	resp := t.handler.HandleRequest(r.Context(), req)
	t.writeRESTResponse(w, resp)
}

// handleResourceAccess handles GET /resources/{uri}
func (t *RESTTransport) handleResourceAccess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract resource URI from path
	resourceURI := strings.TrimPrefix(r.URL.Path, t.restConfig.APIPrefix+"/resources/")
	if resourceURI == "" {
		http.Error(w, "Resource URI required", http.StatusBadRequest)
		return
	}

	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "rest-" + generateID(),
		Method:  "resources/read",
		Params: map[string]interface{}{
			"uri": resourceURI,
		},
	}

	resp := t.handler.HandleRequest(r.Context(), req)
	t.writeRESTResponse(w, resp)
}

// handlePrompts handles GET /prompts
func (t *RESTTransport) handlePrompts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "rest-" + generateID(),
		Method:  "prompts/list",
	}

	resp := t.handler.HandleRequest(r.Context(), req)
	t.writeRESTResponse(w, resp)
}

// handlePromptCall handles POST /prompts/{name}
func (t *RESTTransport) handlePromptCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract prompt name from path
	promptName := strings.TrimPrefix(r.URL.Path, t.restConfig.APIPrefix+"/prompts/")
	if promptName == "" {
		http.Error(w, "Prompt name required", http.StatusBadRequest)
		return
	}

	// Read body
	body, err := io.ReadAll(io.LimitReader(r.Body, t.config.MaxBodySize))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse arguments
	var args map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &args); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}

	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "rest-" + generateID(),
		Method:  "prompts/get",
		Params: map[string]interface{}{
			"name":      promptName,
			"arguments": args,
		},
	}

	resp := t.handler.HandleRequest(r.Context(), req)
	t.writeRESTResponse(w, resp)
}

// handleServerInfo handles GET /info
func (t *RESTTransport) handleServerInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := &protocol.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "rest-" + generateID(),
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": protocol.Version,
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "REST Client",
				"version": "1.0.0",
			},
		},
	}

	resp := t.handler.HandleRequest(r.Context(), req)
	t.writeRESTResponse(w, resp)
}

// handleHealth handles GET /health
func (t *RESTTransport) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleDocs serves API documentation
func (t *RESTTransport) handleDocs(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>MCP REST API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
    window.onload = function() {
        SwaggerUIBundle({
            url: "` + t.restConfig.APIPrefix + `/openapi.json",
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIBundle.SwaggerUIStandalonePreset
            ]
        });
    }
    </script>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleOpenAPISpec serves the OpenAPI specification
func (t *RESTTransport) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec := t.generateOpenAPISpec()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spec)
}

// writeRESTResponse converts JSON-RPC response to REST response
func (t *RESTTransport) writeRESTResponse(w http.ResponseWriter, resp *protocol.JSONRPCResponse) {
	w.Header().Set("Content-Type", "application/json")
	
	// Add custom headers
	for k, v := range t.config.CustomHeaders {
		w.Header().Set(k, v)
	}

	if resp.Error != nil {
		// Map JSON-RPC error codes to HTTP status codes
		statusCode := http.StatusInternalServerError
		switch resp.Error.Code {
		case protocol.ParseError:
			statusCode = http.StatusBadRequest
		case protocol.InvalidRequest:
			statusCode = http.StatusBadRequest
		case protocol.MethodNotFound:
			statusCode = http.StatusNotFound
		case protocol.InvalidParams:
			statusCode = http.StatusBadRequest
		}
		
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    resp.Error.Code,
				"message": resp.Error.Message,
				"data":    resp.Error.Data,
			},
		})
	} else {
		json.NewEncoder(w).Encode(resp.Result)
	}
}

// wrapWithRESTMiddleware wraps the handler with REST-specific middleware
func (t *RESTTransport) wrapWithRESTMiddleware(handler http.Handler) http.Handler {
	// Apply base middleware
	handler = t.wrapWithMiddleware(handler)

	// Authentication middleware
	if t.restConfig.APIKey != "" {
		handler = t.authMiddleware(handler)
	}

	// Rate limiting middleware
	if t.rateLimiter != nil {
		handler = t.rateLimitMiddleware(handler)
	}

	// Logging middleware
	if t.restConfig.EnableLogging {
		handler = t.loggingMiddleware(handler)
	}

	return handler
}

// authMiddleware handles API key authentication
func (t *RESTTransport) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check API key
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey != t.restConfig.APIKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// rateLimitMiddleware implements rate limiting
func (t *RESTTransport) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		if !t.rateLimiter.allow(ip) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs requests and responses
func (t *RESTTransport) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		fmt.Printf("[%s] %s %s %d %v\n", 
			time.Now().Format(time.RFC3339),
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
		)
	})
}

// generateOpenAPISpec generates OpenAPI specification
func (t *RESTTransport) generateOpenAPISpec() map[string]interface{} {
	return map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "MCP REST API",
			"version":     "1.0.0",
			"description": "RESTful API for Model Context Protocol",
		},
		"servers": []interface{}{
			map[string]interface{}{
				"url": t.restConfig.APIPrefix,
			},
		},
		"paths": map[string]interface{}{
			"/tools": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List available tools",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of tools",
						},
					},
				},
			},
			"/tools/{name}": map[string]interface{}{
				"post": map[string]interface{}{
					"summary": "Call a tool",
					"parameters": []interface{}{
						map[string]interface{}{
							"name":     "name",
							"in":       "path",
							"required": true,
							"schema":   map[string]interface{}{"type": "string"},
						},
					},
					"requestBody": map[string]interface{}{
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Tool execution result",
						},
					},
				},
			},
			"/resources": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List available resources",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of resources",
						},
					},
				},
			},
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Server health status",
						},
					},
				},
			},
		},
	}
}

// Helper types and functions

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

type rateLimiter struct {
	requests map[string][]time.Time
	limit    int
	mu       sync.Mutex
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	minute := now.Add(-time.Minute)

	// Clean old requests
	if reqs, ok := rl.requests[ip]; ok {
		var valid []time.Time
		for _, t := range reqs {
			if t.After(minute) {
				valid = append(valid, t)
			}
		}
		rl.requests[ip] = valid
	}

	// Check limit
	if len(rl.requests[ip]) >= rl.limit {
		return false
	}

	// Add request
	rl.requests[ip] = append(rl.requests[ip], now)
	return true
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to remote address
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	
	return r.RemoteAddr
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}