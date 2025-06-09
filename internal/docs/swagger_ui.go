// Package docs provides Swagger UI integration for interactive API documentation.
package docs

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"lerian-mcp-memory/internal/config"
)

//go:embed static/*
var staticFiles embed.FS

// SwaggerUIHandler provides interactive API documentation using Swagger UI
type SwaggerUIHandler struct {
	config    *config.Config
	generator *OpenAPIGenerator
	template  *template.Template
}

// SwaggerUIConfig holds configuration for Swagger UI
type SwaggerUIConfig struct {
	Title         string
	Description   string
	SpecURL       string
	Version       string
	ContactName   string
	ContactURL    string
	ContactEmail  string
	LicenseName   string
	LicenseURL    string
	ServerURL     string
	TryItOutEnabled bool
	DeepLinking   bool
	DisplayOperationId bool
	DefaultModelsExpandDepth int
	DefaultModelExpandDepth  int
	DocExpansion string
	Filter       bool
	ShowExtensions bool
	ShowCommonExtensions bool
	UseUnsafeMarkdown bool
}

// NewSwaggerUIHandler creates a new Swagger UI handler
func NewSwaggerUIHandler(cfg *config.Config, generator *OpenAPIGenerator) *SwaggerUIHandler {
	handler := &SwaggerUIHandler{
		config:    cfg,
		generator: generator,
	}

	// Parse the HTML template for Swagger UI
	handler.template = template.Must(template.New("swagger").Parse(swaggerUITemplate))

	return handler
}

// ServeHTTP serves the Swagger UI interface
func (h *SwaggerUIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/docs")
	
	switch {
	case path == "" || path == "/":
		h.serveSwaggerUI(w, r)
	case path == "/openapi.json":
		h.serveOpenAPIJSON(w, r)
	case path == "/openapi.yaml":
		h.serveOpenAPIYAML(w, r)
	case strings.HasPrefix(path, "/static/"):
		h.serveStaticFile(w, r, path)
	default:
		http.NotFound(w, r)
	}
}

// serveSwaggerUI serves the main Swagger UI HTML page
func (h *SwaggerUIHandler) serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	config := h.getSwaggerUIConfig(r)
	
	var buf bytes.Buffer
	if err := h.template.Execute(&buf, config); err != nil {
		http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	
	w.Write(buf.Bytes())
}

// serveOpenAPIJSON serves the OpenAPI specification in JSON format
func (h *SwaggerUIHandler) serveOpenAPIJSON(w http.ResponseWriter, r *http.Request) {
	spec, err := h.generator.GenerateJSON()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate OpenAPI spec: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	w.Write(spec)
}

// serveOpenAPIYAML serves the OpenAPI specification in YAML format
func (h *SwaggerUIHandler) serveOpenAPIYAML(w http.ResponseWriter, r *http.Request) {
	spec, err := h.generator.GenerateYAML()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate OpenAPI spec: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	
	w.Write(spec)
}

// serveStaticFile serves static Swagger UI assets
func (h *SwaggerUIHandler) serveStaticFile(w http.ResponseWriter, r *http.Request, path string) {
	// Remove /static/ prefix
	filename := strings.TrimPrefix(path, "/static/")
	
	// Read file from embedded filesystem
	content, err := staticFiles.ReadFile(filepath.Join("static", filename))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Set appropriate content type
	contentType := getContentType(filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours
	
	w.Write(content)
}

// getSwaggerUIConfig returns configuration for Swagger UI
func (h *SwaggerUIHandler) getSwaggerUIConfig(r *http.Request) *SwaggerUIConfig {
	baseURL := getBaseURL(r)
	
	return &SwaggerUIConfig{
		Title:         "MCP Memory Server API Documentation",
		Description:   "Interactive API documentation for the Model Context Protocol Memory Server",
		SpecURL:       baseURL + "/docs/openapi.json",
		Version:       "1.0.0",
		ContactName:   "Lerian Studio",
		ContactURL:    "https://github.com/lerianstudio/lerian-mcp-memory",
		ContactEmail:  "support@lerian.studio",
		LicenseName:   "MIT",
		LicenseURL:    "https://opensource.org/licenses/MIT",
		ServerURL:     baseURL,
		TryItOutEnabled: true,
		DeepLinking:   true,
		DisplayOperationId: true,
		DefaultModelsExpandDepth: 1,
		DefaultModelExpandDepth:  1,
		DocExpansion: "list",
		Filter:       true,
		ShowExtensions: false,
		ShowCommonExtensions: false,
		UseUnsafeMarkdown: false,
	}
}

// getBaseURL extracts the base URL from the request
func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	
	host := r.Host
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		host = forwarded
	}
	
	return fmt.Sprintf("%s://%s", scheme, host)
}

// getContentType returns the appropriate content type for a file
func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	
	contentTypes := map[string]string{
		".html": "text/html; charset=utf-8",
		".css":  "text/css; charset=utf-8",
		".js":   "application/javascript; charset=utf-8",
		".json": "application/json",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".woff": "font/woff",
		".woff2": "font/woff2",
		".ttf":  "font/ttf",
		".eot":  "application/vnd.ms-fontobject",
	}
	
	if contentType, exists := contentTypes[ext]; exists {
		return contentType
	}
	
	return "application/octet-stream"
}

// DocumentationMiddleware adds documentation-related headers and CORS
func DocumentationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers for API documentation
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		w.Header().Set("Access-Control-Max-Age", "86400")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		
		// Add documentation discovery headers
		w.Header().Set("Link", `</docs/openapi.json>; rel="service-desc"`)
		w.Header().Set("X-API-Version", "1.0.0")
		w.Header().Set("X-Documentation-URL", "/docs")
		
		next.ServeHTTP(w, r)
	})
}

// HealthCheckHandler provides a documentation-aware health check
type HealthCheckHandler struct {
	generator *OpenAPIGenerator
	startTime time.Time
}

// NewHealthCheckHandler creates a new health check handler
func NewHealthCheckHandler(generator *OpenAPIGenerator) *HealthCheckHandler {
	return &HealthCheckHandler{
		generator: generator,
		startTime: time.Now(),
	}
}

// ServeHTTP serves health check information with API documentation links
func (h *HealthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)
	
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
		"uptime":    uptime.String(),
		"documentation": map[string]string{
			"interactive": "/docs",
			"openapi_json": "/docs/openapi.json",
			"openapi_yaml": "/docs/openapi.yaml",
		},
		"endpoints": map[string]string{
			"mcp":     "/mcp",
			"health":  "/health",
			"metrics": "/metrics",
			"websocket": "/ws",
			"sse":     "/sse",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	
	// Validate OpenAPI spec as part of health check
	if err := h.generator.ValidateSpecification(); err != nil {
		health["status"] = "degraded"
		health["warnings"] = []string{"OpenAPI specification validation failed: " + err.Error()}
		w.WriteHeader(http.StatusOK) // Still return 200 but mark as degraded
	}
	
	fmt.Fprintf(w, `{
  "status": "%s",
  "timestamp": "%s",
  "version": "%s",
  "uptime": "%s",
  "documentation": {
    "interactive": "/docs",
    "openapi_json": "/docs/openapi.json",
    "openapi_yaml": "/docs/openapi.yaml"
  },
  "endpoints": {
    "mcp": "/mcp",
    "health": "/health",
    "metrics": "/metrics",
    "websocket": "/ws",
    "sse": "/sse"
  }
}`, health["status"], health["timestamp"], health["version"], health["uptime"])
}

// APIExampleGenerator generates example requests and responses for API endpoints
type APIExampleGenerator struct {
	generator *OpenAPIGenerator
}

// NewAPIExampleGenerator creates a new API example generator
func NewAPIExampleGenerator(generator *OpenAPIGenerator) *APIExampleGenerator {
	return &APIExampleGenerator{
		generator: generator,
	}
}

// GenerateExamples generates realistic examples for all API endpoints
func (g *APIExampleGenerator) GenerateExamples() map[string]interface{} {
	examples := map[string]interface{}{
		"mcp_memory_search": map[string]interface{}{
			"request": map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "1",
				"method":  "memory_search",
				"params": map[string]interface{}{
					"query":     "database optimization techniques",
					"limit":     10,
					"threshold": 0.7,
					"repository": "lerian-mcp-memory",
				},
			},
			"response": map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "1",
				"result": map[string]interface{}{
					"chunks": []map[string]interface{}{
						{
							"id":         "chunk_001",
							"content":    "Database indexing strategies for optimal query performance...",
							"relevance":  0.92,
							"created_at": "2024-01-15T10:30:00Z",
							"metadata": map[string]interface{}{
								"source": "performance_guide.md",
								"type":   "documentation",
							},
						},
					},
					"total": 5,
					"query_time": "15ms",
				},
			},
		},
		"mcp_memory_store": map[string]interface{}{
			"request": map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "2",
				"method":  "memory_store_chunk",
				"params": map[string]interface{}{
					"content": "Implemented connection pooling with health monitoring for PostgreSQL database",
					"type":    "code_change",
					"metadata": map[string]interface{}{
						"file":   "internal/storage/pool/connection_pool.go",
						"author": "developer",
						"commit": "abc123def456",
					},
					"repository": "lerian-mcp-memory",
					"tags":       []string{"database", "performance", "pooling"},
				},
			},
			"response": map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "2",
				"result": map[string]interface{}{
					"chunk_id":   "chunk_002",
					"stored_at":  "2024-01-15T11:45:00Z",
					"embedding":  true,
					"indexed":    true,
					"similar_chunks": 3,
				},
			},
		},
		"health_check": map[string]interface{}{
			"response": map[string]interface{}{
				"status":    "healthy",
				"timestamp": "2024-01-15T12:00:00Z",
				"version":   "1.0.0",
				"uptime":    "24h30m15s",
				"components": map[string]interface{}{
					"database": map[string]interface{}{
						"status":  "healthy",
						"latency": "5ms",
					},
					"vector_store": map[string]interface{}{
						"status":  "healthy",
						"latency": "12ms",
					},
					"ai_service": map[string]interface{}{
						"status":  "healthy",
						"latency": "150ms",
					},
				},
			},
		},
		"database_metrics": map[string]interface{}{
			"response": map[string]interface{}{
				"connections": map[string]interface{}{
					"active": 8,
					"idle":   17,
					"max":    25,
					"waits":  0,
				},
				"queries": map[string]interface{}{
					"total":        150423,
					"slow":         12,
					"avg_duration": "8ms",
					"p95_duration": "25ms",
				},
				"cache": map[string]interface{}{
					"hit_ratio":    98.5,
					"buffer_ratio": 99.2,
					"index_ratio":  94.8,
				},
				"collected_at": "2024-01-15T12:00:00Z",
			},
		},
	}
	
	return examples
}

// Swagger UI HTML template
const swaggerUITemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{.Title}}</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui.css" />
    <link rel="icon" type="image/png" href="https://unpkg.com/swagger-ui-dist@5.10.5/favicon-32x32.png" sizes="32x32" />
    <link rel="icon" type="image/png" href="https://unpkg.com/swagger-ui-dist@5.10.5/favicon-16x16.png" sizes="16x16" />
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        
        *, *:before, *:after {
            box-sizing: inherit;
        }
        
        body {
            margin: 0;
            background: #fafafa;
        }
        
        .swagger-ui .topbar {
            background-color: #1f2937;
        }
        
        .swagger-ui .topbar .topbar-wrapper {
            padding: 10px 20px;
        }
        
        .swagger-ui .topbar .topbar-wrapper .link {
            color: #ffffff;
            font-weight: bold;
            text-decoration: none;
        }
        
        .custom-header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            margin-bottom: 20px;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        
        .custom-header h1 {
            margin: 0 0 10px 0;
            font-size: 2em;
        }
        
        .custom-header p {
            margin: 0;
            opacity: 0.9;
            font-size: 1.1em;
        }
        
        .api-info {
            background: white;
            padding: 20px;
            margin-bottom: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
        }
        
        .api-info h3 {
            margin-top: 0;
            color: #1f2937;
        }
        
        .quick-links {
            display: flex;
            gap: 15px;
            flex-wrap: wrap;
        }
        
        .quick-link {
            background: #3b82f6;
            color: white;
            padding: 8px 16px;
            border-radius: 4px;
            text-decoration: none;
            transition: background-color 0.2s;
        }
        
        .quick-link:hover {
            background: #2563eb;
        }
        
        .endpoint-count {
            background: #10b981;
            color: white;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 0.9em;
            margin-left: 10px;
        }
    </style>
</head>

<body>
    <div class="custom-header">
        <h1>{{.Title}}</h1>
        <p>{{.Description}}</p>
    </div>
    
    <div class="api-info">
        <h3>MCP Memory Server API <span class="endpoint-count">25+ Endpoints</span></h3>
        <p>
            This API provides 41 memory tools for AI assistants including search, storage, analysis, and management operations.
            The server supports multiple transport protocols (HTTP, WebSocket, SSE) and provides comprehensive memory capabilities.
        </p>
        
        <div class="quick-links">
            <a href="{{.SpecURL}}" class="quick-link" target="_blank">📄 Download JSON</a>
            <a href="/docs/openapi.yaml" class="quick-link" target="_blank">📄 Download YAML</a>
            <a href="/health" class="quick-link" target="_blank">❤️ Health Check</a>
            <a href="/metrics" class="quick-link" target="_blank">📊 Metrics</a>
            <a href="{{.ContactURL}}" class="quick-link" target="_blank">🔗 GitHub</a>
        </div>
    </div>

    <div id="swagger-ui"></div>

    <script src="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '{{.SpecURL}}',
                dom_id: '#swagger-ui',
                deepLinking: {{.DeepLinking}},
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                tryItOutEnabled: {{.TryItOutEnabled}},
                displayOperationId: {{.DisplayOperationId}},
                defaultModelsExpandDepth: {{.DefaultModelsExpandDepth}},
                defaultModelExpandDepth: {{.DefaultModelExpandDepth}},
                docExpansion: "{{.DocExpansion}}",
                filter: {{.Filter}},
                showExtensions: {{.ShowExtensions}},
                showCommonExtensions: {{.ShowCommonExtensions}},
                useUnsafeMarkdown: {{.UseUnsafeMarkdown}},
                validatorUrl: null,
                supportedSubmitMethods: ['get', 'post', 'put', 'delete', 'patch'],
                onComplete: function() {
                    console.log('Swagger UI loaded successfully');
                },
                requestInterceptor: function(request) {
                    // Add custom headers or modify requests here
                    request.headers['X-API-Client'] = 'SwaggerUI';
                    return request;
                },
                responseInterceptor: function(response) {
                    // Process responses here
                    return response;
                }
            });

            // Add custom styling or behavior after UI loads
            setTimeout(function() {
                // Add version info to topbar
                const topbar = document.querySelector('.swagger-ui .topbar');
                if (topbar) {
                    const versionInfo = document.createElement('div');
                    versionInfo.style.cssText = 'position: absolute; right: 20px; top: 50%; transform: translateY(-50%); color: #ffffff; font-size: 14px;';
                    versionInfo.innerHTML = 'v{{.Version}} | <a href="{{.ContactURL}}" style="color: #ffffff;">Support</a>';
                    topbar.style.position = 'relative';
                    topbar.appendChild(versionInfo);
                }
            }, 1000);
        };
    </script>
</body>
</html>`