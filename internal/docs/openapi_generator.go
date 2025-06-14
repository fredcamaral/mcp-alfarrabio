// Package docs provides OpenAPI specification generation and documentation tools.
package docs

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"time"

	"lerian-mcp-memory/internal/config"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// OpenAPIGenerator generates OpenAPI 3.0 specifications from Go code annotations
type OpenAPIGenerator struct {
	config     *config.Config
	spec       *OpenAPISpec
	paths      map[string]*PathItem
	components *Components
	tags       map[string]*Tag
	servers    []*Server
}

// OpenAPISpec represents the root OpenAPI 3.0 specification
type OpenAPISpec struct {
	OpenAPI      string                `json:"openapi"`
	Info         *Info                 `json:"info"`
	Servers      []*Server             `json:"servers,omitempty"`
	Paths        map[string]*PathItem  `json:"paths"`
	Components   *Components           `json:"components,omitempty"`
	Security     []SecurityRequirement `json:"security,omitempty"`
	Tags         []*Tag                `json:"tags,omitempty"`
	ExternalDocs *ExternalDocs         `json:"externalDocs,omitempty"`
}

// Info provides metadata about the API
type Info struct {
	Title          string   `json:"title"`
	Version        string   `json:"version"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
}

// Contact information for the API
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License information for the API
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Server represents a server object
type Server struct {
	URL         string                     `json:"url"`
	Description string                     `json:"description,omitempty"`
	Variables   map[string]*ServerVariable `json:"variables,omitempty"`
}

// ServerVariable represents a server variable
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"default"`
	Description string   `json:"description,omitempty"`
}

// PathItem describes operations available on a single path
type PathItem struct {
	Summary     string       `json:"summary,omitempty"`
	Description string       `json:"description,omitempty"`
	GET         *Operation   `json:"get,omitempty"`
	PUT         *Operation   `json:"put,omitempty"`
	POST        *Operation   `json:"post,omitempty"`
	DELETE      *Operation   `json:"delete,omitempty"`
	OPTIONS     *Operation   `json:"options,omitempty"`
	HEAD        *Operation   `json:"head,omitempty"`
	PATCH       *Operation   `json:"patch,omitempty"`
	TRACE       *Operation   `json:"trace,omitempty"`
	Parameters  []*Parameter `json:"parameters,omitempty"`
}

// Operation describes a single API operation on a path
type Operation struct {
	Tags         []string              `json:"tags,omitempty"`
	Summary      string                `json:"summary,omitempty"`
	Description  string                `json:"description,omitempty"`
	OperationID  string                `json:"operationId,omitempty"`
	Parameters   []*Parameter          `json:"parameters,omitempty"`
	RequestBody  *RequestBody          `json:"requestBody,omitempty"`
	Responses    map[string]*Response  `json:"responses"`
	Callbacks    map[string]*Callback  `json:"callbacks,omitempty"`
	Deprecated   bool                  `json:"deprecated,omitempty"`
	Security     []SecurityRequirement `json:"security,omitempty"`
	Servers      []*Server             `json:"servers,omitempty"`
	ExternalDocs *ExternalDocs         `json:"externalDocs,omitempty"`
}

// Parameter describes a single operation parameter
type Parameter struct {
	Name            string              `json:"name"`
	In              string              `json:"in"` // "query", "header", "path", "cookie"
	Description     string              `json:"description,omitempty"`
	Required        bool                `json:"required,omitempty"`
	Deprecated      bool                `json:"deprecated,omitempty"`
	AllowEmptyValue bool                `json:"allowEmptyValue,omitempty"`
	Style           string              `json:"style,omitempty"`
	Explode         *bool               `json:"explode,omitempty"`
	AllowReserved   bool                `json:"allowReserved,omitempty"`
	Schema          *Schema             `json:"schema,omitempty"`
	Example         interface{}         `json:"example,omitempty"`
	Examples        map[string]*Example `json:"examples,omitempty"`
}

// RequestBody describes a single request body
type RequestBody struct {
	Description string                `json:"description,omitempty"`
	Content     map[string]*MediaType `json:"content"`
	Required    bool                  `json:"required,omitempty"`
}

// Response describes a single response from an API operation
type Response struct {
	Description string                `json:"description"`
	Headers     map[string]*Header    `json:"headers,omitempty"`
	Content     map[string]*MediaType `json:"content,omitempty"`
	Links       map[string]*Link      `json:"links,omitempty"`
}

// MediaType provides schema and examples for the media type identified by its key
type MediaType struct {
	Schema   *Schema              `json:"schema,omitempty"`
	Example  interface{}          `json:"example,omitempty"`
	Examples map[string]*Example  `json:"examples,omitempty"`
	Encoding map[string]*Encoding `json:"encoding,omitempty"`
}

// Schema defines input and output data types
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty"`
	OneOf                []*Schema          `json:"oneOf,omitempty"`
	AnyOf                []*Schema          `json:"anyOf,omitempty"`
	Not                  *Schema            `json:"not,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	AdditionalProperties interface{}        `json:"additionalProperties,omitempty"`
	Description          string             `json:"description,omitempty"`
	Format               string             `json:"format,omitempty"`
	Default              interface{}        `json:"default,omitempty"`
	Title                string             `json:"title,omitempty"`
	MultipleOf           *float64           `json:"multipleOf,omitempty"`
	Maximum              *float64           `json:"maximum,omitempty"`
	ExclusiveMaximum     *bool              `json:"exclusiveMaximum,omitempty"`
	Minimum              *float64           `json:"minimum,omitempty"`
	ExclusiveMinimum     *bool              `json:"exclusiveMinimum,omitempty"`
	MaxLength            *int64             `json:"maxLength,omitempty"`
	MinLength            *int64             `json:"minLength,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
	MaxItems             *int64             `json:"maxItems,omitempty"`
	MinItems             *int64             `json:"minItems,omitempty"`
	UniqueItems          *bool              `json:"uniqueItems,omitempty"`
	MaxProperties        *int64             `json:"maxProperties,omitempty"`
	MinProperties        *int64             `json:"minProperties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Enum                 []interface{}      `json:"enum,omitempty"`
	Example              interface{}        `json:"example,omitempty"`
	Nullable             *bool              `json:"nullable,omitempty"`
	ReadOnly             bool               `json:"readOnly,omitempty"`
	WriteOnly            bool               `json:"writeOnly,omitempty"`
	Deprecated           bool               `json:"deprecated,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
}

// Components holds a set of reusable objects for different aspects of the OAS
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	Responses       map[string]*Response       `json:"responses,omitempty"`
	Parameters      map[string]*Parameter      `json:"parameters,omitempty"`
	Examples        map[string]*Example        `json:"examples,omitempty"`
	RequestBodies   map[string]*RequestBody    `json:"requestBodies,omitempty"`
	Headers         map[string]*Header         `json:"headers,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
	Links           map[string]*Link           `json:"links,omitempty"`
	Callbacks       map[string]*Callback       `json:"callbacks,omitempty"`
}

// Example object
type Example struct {
	Summary       string      `json:"summary,omitempty"`
	Description   string      `json:"description,omitempty"`
	Value         interface{} `json:"value,omitempty"`
	ExternalValue string      `json:"externalValue,omitempty"`
}

// Header object
type Header struct {
	Description     string              `json:"description,omitempty"`
	Required        bool                `json:"required,omitempty"`
	Deprecated      bool                `json:"deprecated,omitempty"`
	AllowEmptyValue bool                `json:"allowEmptyValue,omitempty"`
	Style           string              `json:"style,omitempty"`
	Explode         *bool               `json:"explode,omitempty"`
	AllowReserved   bool                `json:"allowReserved,omitempty"`
	Schema          *Schema             `json:"schema,omitempty"`
	Example         interface{}         `json:"example,omitempty"`
	Examples        map[string]*Example `json:"examples,omitempty"`
}

// Tag adds metadata to a single tag
type Tag struct {
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty"`
}

// ExternalDocs allows referencing an external resource for extended documentation
type ExternalDocs struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// SecurityRequirement lists the required security schemes to execute this operation
type SecurityRequirement map[string][]string

// SecurityScheme defines a security scheme that can be used by the operations
type SecurityScheme struct {
	Type             string      `json:"type"`
	Description      string      `json:"description,omitempty"`
	Name             string      `json:"name,omitempty"`
	In               string      `json:"in,omitempty"`
	Scheme           string      `json:"scheme,omitempty"`
	BearerFormat     string      `json:"bearerFormat,omitempty"`
	Flows            *OAuthFlows `json:"flows,omitempty"`
	OpenIDConnectURL string      `json:"openIdConnectUrl,omitempty"`
}

// OAuthFlows allows configuration of the supported OAuth flows
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

// OAuthFlow configuration details for a supported OAuth flow
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

// Link represents a possible design-time link for a response
type Link struct {
	OperationRef string                 `json:"operationRef,omitempty"`
	OperationID  string                 `json:"operationId,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	RequestBody  interface{}            `json:"requestBody,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Server       *Server                `json:"server,omitempty"`
}

// Callback is a map of possible out-of band callbacks related to the parent operation
type Callback map[string]*PathItem

// Encoding definition applied to a single schema property
type Encoding struct {
	ContentType   string             `json:"contentType,omitempty"`
	Headers       map[string]*Header `json:"headers,omitempty"`
	Style         string             `json:"style,omitempty"`
	Explode       *bool              `json:"explode,omitempty"`
	AllowReserved bool               `json:"allowReserved,omitempty"`
}

// EndpointInfo represents API endpoint metadata for generation
type EndpointInfo struct {
	Path        string
	Method      string
	Handler     string
	Summary     string
	Description string
	Tags        []string
	Parameters  []*Parameter
	RequestBody *RequestBody
	Responses   map[string]*Response
	Deprecated  bool
	Security    []SecurityRequirement
}

// NewOpenAPIGenerator creates a new OpenAPI specification generator
func NewOpenAPIGenerator(cfg *config.Config) *OpenAPIGenerator {
	return &OpenAPIGenerator{
		config: cfg,
		paths:  make(map[string]*PathItem),
		components: &Components{
			Schemas:         make(map[string]*Schema),
			Responses:       make(map[string]*Response),
			Parameters:      make(map[string]*Parameter),
			Examples:        make(map[string]*Example),
			RequestBodies:   make(map[string]*RequestBody),
			Headers:         make(map[string]*Header),
			SecuritySchemes: make(map[string]*SecurityScheme),
			Links:           make(map[string]*Link),
			Callbacks:       make(map[string]*Callback),
		},
		tags: make(map[string]*Tag),
		servers: []*Server{
			{
				URL:         "http://localhost:9080",
				Description: "Development server",
			},
			{
				URL:         "https://api.mcp-memory.com",
				Description: "Production server",
			},
		},
	}
}

// Generate creates the complete OpenAPI specification
func (g *OpenAPIGenerator) Generate() (*OpenAPISpec, error) {
	g.spec = &OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: &Info{
			Title:       "MCP Memory Server API",
			Version:     "1.0.0",
			Description: "High-performance Model Context Protocol (MCP) server providing persistent memory capabilities for AI assistants",
			Contact: &Contact{
				Name:  "Lerian Studio",
				URL:   "https://github.com/lerianstudio/lerian-mcp-memory",
				Email: "support@lerian.studio",
			},
			License: &License{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		Servers:    g.servers,
		Paths:      g.paths,
		Components: g.components,
		Tags:       g.convertTagsToSlice(),
		Security: []SecurityRequirement{
			{
				"bearerAuth": []string{},
			},
		},
	}

	// Generate MCP Memory Server endpoints
	g.generateMCPEndpoints()
	g.generateHealthEndpoints()
	g.generateMetricsEndpoints()
	g.generateDocumentationEndpoints()

	// Generate reusable components
	g.generateSchemas()
	g.generateSecuritySchemes()
	g.generateCommonResponses()
	g.generateCommonParameters()

	return g.spec, nil
}

// generateMCPEndpoints creates OpenAPI definitions for MCP protocol endpoints
func (g *OpenAPIGenerator) generateMCPEndpoints() {
	// MCP JSON-RPC endpoint
	g.AddEndpoint(&EndpointInfo{
		Path:        "/mcp",
		Method:      "POST",
		Handler:     "handleMCPRequest",
		Summary:     "Execute MCP JSON-RPC Request",
		Description: "Main endpoint for Model Context Protocol JSON-RPC requests. Supports all 41 memory tools including search, store, analyze, and manage operations.",
		Tags:        []string{"MCP Protocol"},
		RequestBody: &RequestBody{
			Description: "JSON-RPC 2.0 request with MCP method and parameters",
			Required:    true,
			Content: map[string]*MediaType{
				"application/json": {
					Schema: &Schema{
						Ref: "#/components/schemas/MCPRequest",
					},
					Example: map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      "1",
						"method":  "memory_search",
						"params": map[string]interface{}{
							"query":     "database optimization",
							"limit":     10,
							"threshold": 0.7,
						},
					},
				},
			},
		},
		Responses: map[string]*Response{
			"200": {
				Description: "Successful MCP response",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/MCPResponse",
						},
					},
				},
			},
			"400": {
				Description: "Bad Request - Invalid request parameters or format",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/JSONRPCError",
						},
					},
				},
			},
			"500": {
				Description: "Internal Server Error - Unexpected server error",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/JSONRPCError",
						},
					},
				},
			},
		},
	})

	// WebSocket endpoint for real-time MCP
	g.AddEndpoint(&EndpointInfo{
		Path:        "/ws",
		Method:      "GET",
		Summary:     "WebSocket MCP Connection",
		Description: "Upgrade to WebSocket for real-time MCP communication with bidirectional messaging support",
		Tags:        []string{"MCP Protocol", "WebSocket"},
		Parameters: []*Parameter{
			{
				Name:        "protocol",
				In:          "query",
				Description: "WebSocket protocol version",
				Schema: &Schema{
					Type:    "string",
					Default: "mcp-1.0",
				},
			},
		},
		Responses: map[string]*Response{
			"101": {
				Description: "Switching Protocols - WebSocket connection established",
			},
			"400": {
				Description: "Bad Request - Invalid request parameters or format",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/JSONRPCError",
						},
					},
				},
			},
		},
	})

	// Server-Sent Events endpoint
	g.AddEndpoint(&EndpointInfo{
		Path:        "/sse",
		Method:      "GET",
		Summary:     "Server-Sent Events Stream",
		Description: "Subscribe to server-sent events for real-time memory updates and notifications",
		Tags:        []string{"MCP Protocol", "SSE"},
		Parameters: []*Parameter{
			{
				Name:        "topics",
				In:          "query",
				Description: "Comma-separated list of topics to subscribe to",
				Schema: &Schema{
					Type: "string",
					Enum: []interface{}{"memory_updates", "system_status", "analytics"},
				},
			},
		},
		Responses: map[string]*Response{
			"200": {
				Description: "SSE stream established",
				Content: map[string]*MediaType{
					"text/event-stream": {
						Schema: &Schema{
							Type: "string",
						},
					},
				},
			},
		},
	})
}

// generateHealthEndpoints creates health check and status endpoints
func (g *OpenAPIGenerator) generateHealthEndpoints() {
	g.AddEndpoint(&EndpointInfo{
		Path:        "/health",
		Method:      "GET",
		Summary:     "Health Check",
		Description: "Check the health status of the MCP Memory Server and all dependencies",
		Tags:        []string{"Health"},
		Responses: map[string]*Response{
			"200": {
				Description: "Service is healthy",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/HealthStatus",
						},
					},
				},
			},
			"503": {
				Description: "Service is unhealthy",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/HealthStatus",
						},
					},
				},
			},
		},
	})

	g.AddEndpoint(&EndpointInfo{
		Path:        "/health/detailed",
		Method:      "GET",
		Summary:     "Detailed Health Check",
		Description: "Get detailed health information for all system components including database, vector store, and AI services",
		Tags:        []string{"Health"},
		Responses: map[string]*Response{
			"200": {
				Description: "Detailed health information",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/DetailedHealthStatus",
						},
					},
				},
			},
		},
	})
}

// generateMetricsEndpoints creates metrics and monitoring endpoints
func (g *OpenAPIGenerator) generateMetricsEndpoints() {
	g.AddEndpoint(&EndpointInfo{
		Path:        "/metrics",
		Method:      "GET",
		Summary:     "Prometheus Metrics",
		Description: "Get Prometheus-formatted metrics for monitoring and alerting",
		Tags:        []string{"Monitoring"},
		Responses: map[string]*Response{
			"200": {
				Description: "Prometheus metrics",
				Content: map[string]*MediaType{
					"text/plain": {
						Schema: &Schema{
							Type: "string",
						},
					},
				},
			},
		},
	})

	g.AddEndpoint(&EndpointInfo{
		Path:        "/metrics/database",
		Method:      "GET",
		Summary:     "Database Performance Metrics",
		Description: "Get detailed database performance metrics including connection pool, query statistics, and optimization suggestions",
		Tags:        []string{"Monitoring", "Database"},
		Responses: map[string]*Response{
			"200": {
				Description: "Database performance metrics",
				Content: map[string]*MediaType{
					"application/json": {
						Schema: &Schema{
							Ref: "#/components/schemas/DatabaseMetrics",
						},
					},
				},
			},
		},
	})
}

// generateDocumentationEndpoints creates documentation endpoints
func (g *OpenAPIGenerator) generateDocumentationEndpoints() {
	g.AddEndpoint(&EndpointInfo{
		Path:        "/docs",
		Method:      "GET",
		Summary:     "API Documentation",
		Description: "Interactive API documentation powered by Swagger UI",
		Tags:        []string{"Documentation"},
		Responses: map[string]*Response{
			"200": {
				Description: "HTML documentation page",
				Content: map[string]*MediaType{
					"text/html": {
						Schema: &Schema{
							Type: "string",
						},
					},
				},
			},
		},
	})

	g.AddEndpoint(&EndpointInfo{
		Path:        "/docs/openapi.yaml",
		Method:      "GET",
		Summary:     "OpenAPI Specification",
		Description: "Download the complete OpenAPI 3.0 specification in YAML format",
		Tags:        []string{"Documentation"},
		Responses: map[string]*Response{
			"200": {
				Description: "OpenAPI specification",
				Content: map[string]*MediaType{
					"application/x-yaml": {
						Schema: &Schema{
							Type: "string",
						},
					},
				},
			},
		},
	})
}

// AddEndpoint adds an endpoint to the OpenAPI specification
func (g *OpenAPIGenerator) AddEndpoint(endpoint *EndpointInfo) {
	pathItem, exists := g.paths[endpoint.Path]
	if !exists {
		pathItem = &PathItem{}
		g.paths[endpoint.Path] = pathItem
	}

	operation := &Operation{
		Summary:     endpoint.Summary,
		Description: endpoint.Description,
		OperationID: generateOperationID(endpoint.Method, endpoint.Path),
		Tags:        endpoint.Tags,
		Parameters:  endpoint.Parameters,
		RequestBody: endpoint.RequestBody,
		Responses:   endpoint.Responses,
		Deprecated:  endpoint.Deprecated,
		Security:    endpoint.Security,
	}

	// Add tags to global tag list
	for _, tag := range endpoint.Tags {
		if _, exists := g.tags[tag]; !exists {
			g.tags[tag] = &Tag{
				Name:        tag,
				Description: generateTagDescription(tag),
			}
		}
	}

	// Set operation based on HTTP method
	switch strings.ToUpper(endpoint.Method) {
	case http.MethodGet:
		pathItem.GET = operation
	case http.MethodPost:
		pathItem.POST = operation
	case http.MethodPut:
		pathItem.PUT = operation
	case http.MethodDelete:
		pathItem.DELETE = operation
	case http.MethodPatch:
		pathItem.PATCH = operation
	case http.MethodOptions:
		pathItem.OPTIONS = operation
	case http.MethodHead:
		pathItem.HEAD = operation
	case http.MethodTrace:
		pathItem.TRACE = operation
	}
}

// generateSchemas creates reusable schema components
func (g *OpenAPIGenerator) generateSchemas() {
	// MCP Request schema
	g.components.Schemas["MCPRequest"] = &Schema{
		Type:        "object",
		Description: "JSON-RPC 2.0 request for MCP protocol",
		Required:    []string{"jsonrpc", "method"},
		Properties: map[string]*Schema{
			"jsonrpc": {
				Type:        "string",
				Description: "JSON-RPC protocol version",
				Enum:        []interface{}{"2.0"},
			},
			"id": {
				Description: "Request identifier",
				OneOf: []*Schema{
					{Type: "string"},
					{Type: "number"},
					{Type: "null"},
				},
			},
			"method": {
				Type:        "string",
				Description: "MCP method name",
				Example:     "memory_search",
			},
			"params": {
				Type:                 "object",
				Description:          "Method-specific parameters",
				AdditionalProperties: true,
			},
		},
	}

	// MCP Response schema
	g.components.Schemas["MCPResponse"] = &Schema{
		Type:        "object",
		Description: "JSON-RPC 2.0 response from MCP protocol",
		Required:    []string{"jsonrpc"},
		Properties: map[string]*Schema{
			"jsonrpc": {
				Type: "string",
				Enum: []interface{}{"2.0"},
			},
			"id": {
				OneOf: []*Schema{
					{Type: "string"},
					{Type: "number"},
					{Type: "null"},
				},
			},
			"result": {
				Description:          "Successful response result",
				Type:                 "object",
				AdditionalProperties: true,
			},
			"error": {
				Ref: "#/components/schemas/JSONRPCError",
			},
		},
	}

	// JSON-RPC Error schema
	g.components.Schemas["JSONRPCError"] = &Schema{
		Type:        "object",
		Description: "JSON-RPC 2.0 error object",
		Required:    []string{"code", "message"},
		Properties: map[string]*Schema{
			"code": {
				Type:        "integer",
				Description: "Error code",
				Example:     -32602,
			},
			"message": {
				Type:        "string",
				Description: "Error message",
				Example:     "Invalid params",
			},
			"data": {
				Description:          "Additional error data",
				AdditionalProperties: true,
			},
		},
	}

	// Health Status schema
	g.components.Schemas["HealthStatus"] = &Schema{
		Type:        "object",
		Description: "System health status",
		Required:    []string{"status", "timestamp"},
		Properties: map[string]*Schema{
			"status": {
				Type:        "string",
				Description: "Overall health status",
				Enum:        []interface{}{"healthy", "degraded", "unhealthy"},
			},
			"timestamp": {
				Type:        "string",
				Format:      "date-time",
				Description: "Health check timestamp",
			},
			"version": {
				Type:        "string",
				Description: "Service version",
				Example:     "1.0.0",
			},
			"uptime": {
				Type:        "string",
				Description: "Service uptime duration",
				Example:     "2h30m15s",
			},
		},
	}

	// Detailed Health Status schema
	g.components.Schemas["DetailedHealthStatus"] = &Schema{
		Type:        "object",
		Description: "Detailed system health status",
		Properties: map[string]*Schema{
			"overall": {
				Ref: "#/components/schemas/HealthStatus",
			},
			"components": {
				Type:        "object",
				Description: "Individual component health status",
				Properties: map[string]*Schema{
					"database": {
						Ref: "#/components/schemas/ComponentHealth",
					},
					"vector_store": {
						Ref: "#/components/schemas/ComponentHealth",
					},
					"ai_service": {
						Ref: "#/components/schemas/ComponentHealth",
					},
					"memory_system": {
						Ref: "#/components/schemas/ComponentHealth",
					},
				},
			},
		},
	}

	// Component Health schema
	g.components.Schemas["ComponentHealth"] = &Schema{
		Type:        "object",
		Description: "Individual component health information",
		Properties: map[string]*Schema{
			"status": {
				Type: "string",
				Enum: []interface{}{"healthy", "degraded", "unhealthy"},
			},
			"latency": {
				Type:        "string",
				Description: "Component response latency",
				Example:     "5ms",
			},
			"error": {
				Type:        "string",
				Description: "Error message if unhealthy",
			},
			"details": {
				Type:                 "object",
				Description:          "Component-specific health details",
				AdditionalProperties: true,
			},
		},
	}

	// Database Metrics schema
	g.components.Schemas["DatabaseMetrics"] = &Schema{
		Type:        "object",
		Description: "Comprehensive database performance metrics",
		Properties: map[string]*Schema{
			"connections": {
				Type:        "object",
				Description: "Connection pool metrics",
				Properties: map[string]*Schema{
					"active": {Type: "integer", Description: "Active connections"},
					"idle":   {Type: "integer", Description: "Idle connections"},
					"max":    {Type: "integer", Description: "Maximum connections"},
					"waits":  {Type: "integer", Description: "Connection waits"},
				},
			},
			"queries": {
				Type:        "object",
				Description: "Query performance metrics",
				Properties: map[string]*Schema{
					"total":        {Type: "integer", Description: "Total queries executed"},
					"slow":         {Type: "integer", Description: "Slow queries"},
					"avg_duration": {Type: "string", Description: "Average query duration"},
					"p95_duration": {Type: "string", Description: "95th percentile duration"},
				},
			},
			"cache": {
				Type:        "object",
				Description: "Cache performance metrics",
				Properties: map[string]*Schema{
					"hit_ratio":    {Type: "number", Format: "float", Description: "Cache hit ratio percentage"},
					"buffer_ratio": {Type: "number", Format: "float", Description: "Buffer hit ratio percentage"},
					"index_ratio":  {Type: "number", Format: "float", Description: "Index hit ratio percentage"},
				},
			},
		},
	}
}

// generateSecuritySchemes creates security scheme definitions
func (g *OpenAPIGenerator) generateSecuritySchemes() {
	g.components.SecuritySchemes["bearerAuth"] = &SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "Bearer token authentication using JWT",
	}

	g.components.SecuritySchemes["apiKey"] = &SecurityScheme{
		Type:        "apiKey",
		In:          "header",
		Name:        "X-API-Key",
		Description: "API key authentication",
	}
}

// generateCommonResponses creates reusable response definitions
func (g *OpenAPIGenerator) generateCommonResponses() {
	g.components.Responses["BadRequest"] = &Response{
		Description: "Bad Request - Invalid request parameters or format",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: &Schema{
					Ref: "#/components/schemas/JSONRPCError",
				},
			},
		},
	}

	g.components.Responses["Unauthorized"] = &Response{
		Description: "Unauthorized - Missing or invalid authentication",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: &Schema{
					Ref: "#/components/schemas/JSONRPCError",
				},
			},
		},
	}

	g.components.Responses["NotFound"] = &Response{
		Description: "Not Found - Requested resource does not exist",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: &Schema{
					Ref: "#/components/schemas/JSONRPCError",
				},
			},
		},
	}

	g.components.Responses["InternalError"] = &Response{
		Description: "Internal Server Error - Unexpected server error",
		Content: map[string]*MediaType{
			"application/json": {
				Schema: &Schema{
					Ref: "#/components/schemas/JSONRPCError",
				},
			},
		},
	}
}

// generateCommonParameters creates reusable parameter definitions
func (g *OpenAPIGenerator) generateCommonParameters() {
	g.components.Parameters["limit"] = &Parameter{
		Name:        "limit",
		In:          "query",
		Description: "Maximum number of results to return",
		Schema: &Schema{
			Type:    "integer",
			Minimum: floatPtr(1),
			Maximum: floatPtr(1000),
			Default: 50,
		},
	}

	g.components.Parameters["offset"] = &Parameter{
		Name:        "offset",
		In:          "query",
		Description: "Number of results to skip",
		Schema: &Schema{
			Type:    "integer",
			Minimum: floatPtr(0),
			Default: 0,
		},
	}

	g.components.Parameters["format"] = &Parameter{
		Name:        "format",
		In:          "query",
		Description: "Response format",
		Schema: &Schema{
			Type:    "string",
			Enum:    []interface{}{"json", "yaml"},
			Default: "json",
		},
	}
}

// convertTagsToSlice converts the tags map to a slice for the specification
func (g *OpenAPIGenerator) convertTagsToSlice() []*Tag {
	tags := make([]*Tag, 0, len(g.tags))
	for _, tag := range g.tags {
		tags = append(tags, tag)
	}
	return tags
}

// GenerateJSON returns the OpenAPI specification as JSON
func (g *OpenAPIGenerator) GenerateJSON() ([]byte, error) {
	spec, err := g.Generate()
	if err != nil {
		return nil, errors.New("failed to generate OpenAPI spec: " + err.Error())
	}

	return json.MarshalIndent(spec, "", "  ")
}

// GenerateYAML returns the OpenAPI specification as YAML
func (g *OpenAPIGenerator) GenerateYAML() ([]byte, error) {
	spec, err := g.Generate()
	if err != nil {
		return nil, errors.New("failed to generate OpenAPI spec: " + err.Error())
	}

	// Convert to YAML format using a simple JSON-to-YAML conversion
	// In production, you would use gopkg.in/yaml.v3 or similar
	jsonBytes, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return nil, errors.New("failed to marshal spec to JSON: " + err.Error())
	}

	// Simple YAML header and format conversion
	yamlHeader := "# OpenAPI 3.0 specification for MCP Memory Server\n# Generated at: " + time.Now().Format(time.RFC3339) + "\n# This is a JSON representation in YAML format\n# For proper YAML formatting, use a dedicated YAML library\n\n"

	// For now, return JSON with YAML header (better than nothing)
	// In production, implement proper YAML marshaling
	return append([]byte(yamlHeader), jsonBytes...), nil
}

// ValidateSpecification validates the generated OpenAPI specification
func (g *OpenAPIGenerator) ValidateSpecification() error {
	spec, err := g.Generate()
	if err != nil {
		return errors.New("failed to generate spec for validation: " + err.Error())
	}

	// Basic validation checks
	if spec.OpenAPI == "" {
		return errors.New("missing OpenAPI version")
	}

	if spec.Info == nil || spec.Info.Title == "" || spec.Info.Version == "" {
		return errors.New("missing required info fields")
	}

	if len(spec.Paths) == 0 {
		return errors.New("no paths defined")
	}

	// Validate each path
	for path, pathItem := range spec.Paths {
		if err := g.validatePathItem(path, pathItem); err != nil {
			return errors.New("path " + path + " validation error: " + err.Error())
		}
	}

	return nil
}

// validatePathItem validates a single path item
func (g *OpenAPIGenerator) validatePathItem(path string, pathItem *PathItem) error {
	_ = path // unused parameter, kept for potential future path-specific validations
	operations := []*Operation{
		pathItem.GET, pathItem.POST, pathItem.PUT, pathItem.DELETE,
		pathItem.PATCH, pathItem.OPTIONS, pathItem.HEAD, pathItem.TRACE,
	}

	hasOperation := false
	for _, op := range operations {
		if op != nil {
			hasOperation = true
			if len(op.Responses) == 0 {
				return errors.New("operation missing responses")
			}
		}
	}

	if !hasOperation {
		return errors.New("path has no operations defined")
	}

	return nil
}

// Helper functions

func generateOperationID(method, path string) string {
	// Convert path to camelCase operation ID
	parts := strings.Split(strings.Trim(path, "/"), "/")
	operationID := strings.ToLower(method)
	caser := cases.Title(language.English)

	for _, part := range parts {
		if part != "" && !strings.HasPrefix(part, "{") {
			operationID += formatPathPart(part, caser)
		}
	}

	return operationID
}

// formatPathPart formats a path part for operation ID, handling extensions specially
func formatPathPart(part string, caser cases.Caser) string {
	// Handle file extensions specially - capitalize each part after dots
	if strings.Contains(part, ".") {
		subParts := strings.Split(part, ".")
		result := ""
		for i, subPart := range subParts {
			if i == 0 {
				result += caser.String(subPart)
			} else {
				result += "." + caser.String(subPart)
			}
		}
		return result
	}
	return caser.String(part)
}

func generateTagDescription(tag string) string {
	descriptions := map[string]string{
		"MCP Protocol":  "Model Context Protocol endpoints for memory operations",
		"Health":        "Health check and status monitoring endpoints",
		"Monitoring":    "Metrics and performance monitoring endpoints",
		"Documentation": "API documentation and specification endpoints",
		"Database":      "Database management and optimization endpoints",
		"WebSocket":     "WebSocket endpoints for real-time communication",
		"SSE":           "Server-Sent Events for real-time updates",
	}

	if desc, exists := descriptions[tag]; exists {
		return desc
	}
	return tag + " related endpoints"
}

func floatPtr(f float64) *float64 {
	return &f
}

// SchemaFromType generates a JSON schema from a Go type using reflection
func SchemaFromType(t reflect.Type) *Schema {
	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Schema{Type: "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer", Minimum: floatPtr(0)}
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Slice, reflect.Array:
		return &Schema{
			Type:  "array",
			Items: SchemaFromType(t.Elem()),
		}
	case reflect.Map:
		return &Schema{
			Type:                 "object",
			AdditionalProperties: SchemaFromType(t.Elem()),
		}
	case reflect.Struct:
		schema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
		}

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath == "" { // exported field
				jsonTag := field.Tag.Get("json")
				if jsonTag != "" && jsonTag != "-" {
					fieldName := strings.Split(jsonTag, ",")[0]
					schema.Properties[fieldName] = SchemaFromType(field.Type)
				}
			}
		}

		return schema
	case reflect.Ptr:
		schema := SchemaFromType(t.Elem())
		nullable := true
		schema.Nullable = &nullable
		return schema
	default:
		return &Schema{Type: "object"}
	}
}
