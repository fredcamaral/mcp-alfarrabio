// Package api provides API annotation support for automatic OpenAPI generation.
package api

import (
	"fmt"
	"reflect"
	"strings"
)

// APIAnnotation represents metadata for API endpoints used in OpenAPI generation
type APIAnnotation struct {
	Path        string            `json:"path"`
	Method      string            `json:"method"`
	Summary     string            `json:"summary"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags"`
	Deprecated  bool              `json:"deprecated"`
	Security    []string          `json:"security"`
	Parameters  []ParameterSpec   `json:"parameters"`
	RequestBody *RequestBodySpec  `json:"requestBody,omitempty"`
	Responses   map[string]ResponseSpec `json:"responses"`
}

// ParameterSpec defines a parameter specification for OpenAPI
type ParameterSpec struct {
	Name        string      `json:"name"`
	In          string      `json:"in"` // query, header, path, cookie
	Type        string      `json:"type"`
	Format      string      `json:"format,omitempty"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Example     interface{} `json:"example,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
	MinLength   *int        `json:"minLength,omitempty"`
	MaxLength   *int        `json:"maxLength,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Maximum     *float64    `json:"maximum,omitempty"`
}

// RequestBodySpec defines the request body specification
type RequestBodySpec struct {
	Description string                 `json:"description"`
	Required    bool                   `json:"required"`
	Content     map[string]MediaTypeSpec `json:"content"`
}

// ResponseSpec defines a response specification
type ResponseSpec struct {
	Description string                 `json:"description"`
	Content     map[string]MediaTypeSpec `json:"content,omitempty"`
	Headers     map[string]HeaderSpec  `json:"headers,omitempty"`
}

// MediaTypeSpec defines media type specification
type MediaTypeSpec struct {
	Schema  SchemaSpec  `json:"schema"`
	Example interface{} `json:"example,omitempty"`
}

// SchemaSpec defines JSON schema specification
type SchemaSpec struct {
	Type                 string                 `json:"type,omitempty"`
	Format               string                 `json:"format,omitempty"`
	Description          string                 `json:"description,omitempty"`
	Properties           map[string]SchemaSpec  `json:"properties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Items                *SchemaSpec            `json:"items,omitempty"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty"`
	Enum                 []interface{}          `json:"enum,omitempty"`
	Example              interface{}            `json:"example,omitempty"`
	Default              interface{}            `json:"default,omitempty"`
	Ref                  string                 `json:"$ref,omitempty"`
	OneOf                []SchemaSpec           `json:"oneOf,omitempty"`
	AnyOf                []SchemaSpec           `json:"anyOf,omitempty"`
	AllOf                []SchemaSpec           `json:"allOf,omitempty"`
}

// HeaderSpec defines header specification
type HeaderSpec struct {
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Format      string      `json:"format,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

// EndpointRegistry stores API endpoint annotations
type EndpointRegistry struct {
	endpoints map[string]*APIAnnotation
}

// NewEndpointRegistry creates a new endpoint registry
func NewEndpointRegistry() *EndpointRegistry {
	return &EndpointRegistry{
		endpoints: make(map[string]*APIAnnotation),
	}
}

// Register adds an API annotation to the registry
func (r *EndpointRegistry) Register(annotation *APIAnnotation) {
	key := fmt.Sprintf("%s:%s", annotation.Method, annotation.Path)
	r.endpoints[key] = annotation
}

// GetAll returns all registered API annotations
func (r *EndpointRegistry) GetAll() map[string]*APIAnnotation {
	return r.endpoints
}

// GetByPath returns annotations for a specific path
func (r *EndpointRegistry) GetByPath(path string) []*APIAnnotation {
	var results []*APIAnnotation
	for _, annotation := range r.endpoints {
		if annotation.Path == path {
			results = append(results, annotation)
		}
	}
	return results
}

// GetByTag returns annotations with a specific tag
func (r *EndpointRegistry) GetByTag(tag string) []*APIAnnotation {
	var results []*APIAnnotation
	for _, annotation := range r.endpoints {
		for _, t := range annotation.Tags {
			if t == tag {
				results = append(results, annotation)
				break
			}
		}
	}
	return results
}

// Global endpoint registry
var DefaultRegistry = NewEndpointRegistry()

// RegisterEndpoint is a convenience function to register an endpoint
func RegisterEndpoint(annotation *APIAnnotation) {
	DefaultRegistry.Register(annotation)
}

// APIDoc creates a documentation annotation for a handler
type APIDoc struct {
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
	Security    []string
}

// Param creates a parameter specification
func Param(name, in, paramType, description string, required bool) ParameterSpec {
	return ParameterSpec{
		Name:        name,
		In:          in,
		Type:        paramType,
		Description: description,
		Required:    required,
	}
}

// QueryParam creates a query parameter specification
func QueryParam(name, paramType, description string, required bool) ParameterSpec {
	return Param(name, "query", paramType, description, required)
}

// PathParam creates a path parameter specification
func PathParam(name, paramType, description string) ParameterSpec {
	return Param(name, "path", paramType, description, true)
}

// HeaderParam creates a header parameter specification
func HeaderParam(name, paramType, description string, required bool) ParameterSpec {
	return Param(name, "header", paramType, description, required)
}

// JSONRequest creates a JSON request body specification
func JSONRequest(description string, schema SchemaSpec, required bool) *RequestBodySpec {
	return &RequestBodySpec{
		Description: description,
		Required:    required,
		Content: map[string]MediaTypeSpec{
			"application/json": {
				Schema: schema,
			},
		},
	}
}

// JSONResponse creates a JSON response specification
func JSONResponse(description string, schema SchemaSpec) ResponseSpec {
	return ResponseSpec{
		Description: description,
		Content: map[string]MediaTypeSpec{
			"application/json": {
				Schema: schema,
			},
		},
	}
}

// PlainTextResponse creates a plain text response specification
func PlainTextResponse(description string) ResponseSpec {
	return ResponseSpec{
		Description: description,
		Content: map[string]MediaTypeSpec{
			"text/plain": {
				Schema: SchemaSpec{
					Type: "string",
				},
			},
		},
	}
}

// HTMLResponse creates an HTML response specification
func HTMLResponse(description string) ResponseSpec {
	return ResponseSpec{
		Description: description,
		Content: map[string]MediaTypeSpec{
			"text/html": {
				Schema: SchemaSpec{
					Type: "string",
				},
			},
		},
	}
}

// SchemaFromStruct creates a schema specification from a Go struct
func SchemaFromStruct(v interface{}) SchemaSpec {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := SchemaSpec{
		Type:       "object",
		Properties: make(map[string]SchemaSpec),
	}

	if t.Kind() != reflect.Struct {
		return SchemaSpec{Type: "object"}
	}

	var required []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" { // unexported field
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		tagParts := strings.Split(jsonTag, ",")
		fieldName := tagParts[0]
		
		fieldSchema := schemaFromType(field.Type)
		
		// Check for description in tag
		if desc := field.Tag.Get("description"); desc != "" {
			fieldSchema.Description = desc
		}
		
		// Check if field is required (not omitempty)
		isRequired := true
		for _, part := range tagParts[1:] {
			if part == "omitempty" {
				isRequired = false
				break
			}
		}
		
		if isRequired {
			required = append(required, fieldName)
		}
		
		schema.Properties[fieldName] = fieldSchema
	}

	if len(required) > 0 {
		schema.Required = required
	}

	return schema
}

// schemaFromType converts a Go type to a JSON schema specification
func schemaFromType(t reflect.Type) SchemaSpec {
	switch t.Kind() {
	case reflect.String:
		return SchemaSpec{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return SchemaSpec{Type: "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return SchemaSpec{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return SchemaSpec{Type: "number"}
	case reflect.Bool:
		return SchemaSpec{Type: "boolean"}
	case reflect.Slice, reflect.Array:
		items := schemaFromType(t.Elem())
		return SchemaSpec{
			Type:  "array",
			Items: &items,
		}
	case reflect.Map:
		if t.Key().Kind() == reflect.String {
			additionalProps := schemaFromType(t.Elem())
			return SchemaSpec{
				Type:                 "object",
				AdditionalProperties: additionalProps,
			}
		}
		return SchemaSpec{Type: "object"}
	case reflect.Struct:
		// For complex structs, we should ideally create a reference
		// For now, return a generic object
		return SchemaSpec{Type: "object"}
	case reflect.Ptr:
		schema := schemaFromType(t.Elem())
		// In OpenAPI 3.0, nullable is handled differently
		return schema
	case reflect.Interface:
		return SchemaSpec{} // Any type
	default:
		return SchemaSpec{Type: "object"}
	}
}

// Predefined common schemas for MCP protocol
var (
	// MCPRequestSchema represents a JSON-RPC 2.0 request
	MCPRequestSchema = SchemaSpec{
		Type:        "object",
		Description: "JSON-RPC 2.0 request for MCP protocol",
		Required:    []string{"jsonrpc", "method"},
		Properties: map[string]SchemaSpec{
			"jsonrpc": {
				Type:        "string",
				Description: "JSON-RPC protocol version",
				Enum:        []interface{}{"2.0"},
			},
			"id": {
				Description: "Request identifier",
				OneOf: []SchemaSpec{
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

	// MCPResponseSchema represents a JSON-RPC 2.0 response
	MCPResponseSchema = SchemaSpec{
		Type:        "object",
		Description: "JSON-RPC 2.0 response from MCP protocol",
		Required:    []string{"jsonrpc"},
		Properties: map[string]SchemaSpec{
			"jsonrpc": {
				Type: "string",
				Enum: []interface{}{"2.0"},
			},
			"id": {
				OneOf: []SchemaSpec{
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

	// ErrorSchema represents a standard error response
	ErrorSchema = SchemaSpec{
		Type:        "object",
		Description: "JSON-RPC 2.0 error object",
		Required:    []string{"code", "message"},
		Properties: map[string]SchemaSpec{
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
)

// Helper function to create float64 pointer
func float64Ptr(f float64) *float64 {
	return &f
}

// InitializeMCPEndpoints registers all MCP Memory Server endpoints
func InitializeMCPEndpoints() {
	// Main MCP JSON-RPC endpoint
	RegisterEndpoint(&APIAnnotation{
		Path:        "/mcp",
		Method:      "POST",
		Summary:     "Execute MCP JSON-RPC Request",
		Description: "Main endpoint for Model Context Protocol JSON-RPC requests. Supports all 41 memory tools including search, store, analyze, and manage operations.",
		Tags:        []string{"MCP Protocol"},
		RequestBody: JSONRequest("JSON-RPC 2.0 request with MCP method and parameters", MCPRequestSchema, true),
		Responses: map[string]ResponseSpec{
			"200": JSONResponse("Successful MCP response", MCPResponseSchema),
			"400": JSONResponse("Bad Request - Invalid request parameters", ErrorSchema),
			"500": JSONResponse("Internal Server Error", ErrorSchema),
		},
	})

	// WebSocket endpoint
	RegisterEndpoint(&APIAnnotation{
		Path:        "/ws",
		Method:      "GET",
		Summary:     "WebSocket MCP Connection",
		Description: "Upgrade to WebSocket for real-time MCP communication with bidirectional messaging support",
		Tags:        []string{"MCP Protocol", "WebSocket"},
		Parameters: []ParameterSpec{
			QueryParam("protocol", "string", "WebSocket protocol version", false),
		},
		Responses: map[string]ResponseSpec{
			"101": {Description: "Switching Protocols - WebSocket connection established"},
			"400": JSONResponse("Bad Request", ErrorSchema),
		},
	})

	// Server-Sent Events endpoint
	RegisterEndpoint(&APIAnnotation{
		Path:        "/sse",
		Method:      "GET",
		Summary:     "Server-Sent Events Stream",
		Description: "Subscribe to server-sent events for real-time memory updates and notifications",
		Tags:        []string{"MCP Protocol", "SSE"},
		Parameters: []ParameterSpec{
			QueryParam("topics", "string", "Comma-separated list of topics to subscribe to", false),
		},
		Responses: map[string]ResponseSpec{
			"200": {
				Description: "SSE stream established",
				Content: map[string]MediaTypeSpec{
					"text/event-stream": {
						Schema: SchemaSpec{Type: "string"},
					},
				},
			},
		},
	})

	// Health check endpoint
	RegisterEndpoint(&APIAnnotation{
		Path:        "/health",
		Method:      "GET",
		Summary:     "Health Check",
		Description: "Check the health status of the MCP Memory Server and all dependencies",
		Tags:        []string{"Health"},
		Responses: map[string]ResponseSpec{
			"200": JSONResponse("Service is healthy", SchemaSpec{
				Type: "object",
				Properties: map[string]SchemaSpec{
					"status":    {Type: "string", Enum: []interface{}{"healthy", "degraded", "unhealthy"}},
					"timestamp": {Type: "string", Format: "date-time"},
					"version":   {Type: "string"},
					"uptime":    {Type: "string"},
				},
			}),
			"503": JSONResponse("Service is unhealthy", ErrorSchema),
		},
	})

	// Metrics endpoint
	RegisterEndpoint(&APIAnnotation{
		Path:        "/metrics",
		Method:      "GET",
		Summary:     "Prometheus Metrics",
		Description: "Get Prometheus-formatted metrics for monitoring and alerting",
		Tags:        []string{"Monitoring"},
		Responses: map[string]ResponseSpec{
			"200": PlainTextResponse("Prometheus metrics"),
		},
	})

	// Documentation endpoint
	RegisterEndpoint(&APIAnnotation{
		Path:        "/docs",
		Method:      "GET",
		Summary:     "API Documentation",
		Description: "Interactive API documentation powered by Swagger UI",
		Tags:        []string{"Documentation"},
		Responses: map[string]ResponseSpec{
			"200": HTMLResponse("HTML documentation page"),
		},
	})
}