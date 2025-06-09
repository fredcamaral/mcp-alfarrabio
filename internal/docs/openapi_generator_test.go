package docs

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"lerian-mcp-memory/internal/config"
)

func TestOpenAPIGenerator_Generate(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 9080,
			Host: "localhost",
		},
	}

	generator := NewOpenAPIGenerator(cfg)

	spec, err := generator.Generate()
	if err != nil {
		t.Fatalf("Failed to generate OpenAPI spec: %v", err)
	}

	// Validate basic structure
	if spec.OpenAPI != "3.0.3" {
		t.Errorf("Expected OpenAPI version 3.0.3, got %s", spec.OpenAPI)
	}

	if spec.Info == nil {
		t.Fatal("Info object is nil")
	}

	if spec.Info.Title != "MCP Memory Server API" {
		t.Errorf("Expected title 'MCP Memory Server API', got %s", spec.Info.Title)
	}

	if spec.Info.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", spec.Info.Version)
	}

	// Check paths
	if len(spec.Paths) == 0 {
		t.Error("No paths defined in specification")
	}

	// Verify essential endpoints exist
	essentialPaths := []string{"/mcp", "/health", "/docs", "/metrics", "/ws", "/sse"}
	for _, path := range essentialPaths {
		if _, exists := spec.Paths[path]; !exists {
			t.Errorf("Essential path %s not found in specification", path)
		}
	}

	// Check components
	if spec.Components == nil {
		t.Fatal("Components object is nil")
	}

	if len(spec.Components.Schemas) == 0 {
		t.Error("No schemas defined in components")
	}

	// Verify essential schemas exist
	essentialSchemas := []string{"MCPRequest", "MCPResponse", "JSONRPCError", "HealthStatus"}
	for _, schema := range essentialSchemas {
		if _, exists := spec.Components.Schemas[schema]; !exists {
			t.Errorf("Essential schema %s not found in components", schema)
		}
	}
}

func TestOpenAPIGenerator_GenerateJSON(t *testing.T) {
	cfg := &config.Config{}
	generator := NewOpenAPIGenerator(cfg)

	jsonBytes, err := generator.GenerateJSON()
	if err != nil {
		t.Fatalf("Failed to generate JSON: %v", err)
	}

	// Validate JSON is well-formed
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Generated JSON is not valid: %v", err)
	}

	// Check basic structure
	if openapi, ok := parsed["openapi"].(string); !ok || openapi != "3.0.3" {
		t.Error("Invalid or missing OpenAPI version in JSON")
	}

	if info, ok := parsed["info"].(map[string]interface{}); !ok {
		t.Error("Missing info object in JSON")
	} else {
		if title, ok := info["title"].(string); !ok || title == "" {
			t.Error("Missing or empty title in info object")
		}
	}
}

func TestOpenAPIGenerator_GenerateYAML(t *testing.T) {
	cfg := &config.Config{}
	generator := NewOpenAPIGenerator(cfg)

	yamlBytes, err := generator.GenerateYAML()
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	if len(yamlBytes) == 0 {
		t.Error("Generated YAML is empty")
	}

	// Check YAML header
	yamlStr := string(yamlBytes)
	if !strings.Contains(yamlStr, "OpenAPI 3.0 specification for MCP Memory Server") {
		t.Error("YAML does not contain expected header")
	}

	if !strings.Contains(yamlStr, "Generated at:") {
		t.Error("YAML does not contain generation timestamp")
	}
}

func TestOpenAPIGenerator_ValidateSpecification(t *testing.T) {
	cfg := &config.Config{}
	generator := NewOpenAPIGenerator(cfg)

	err := generator.ValidateSpecification()
	if err != nil {
		t.Fatalf("Specification validation failed: %v", err)
	}
}

func TestOpenAPIGenerator_AddEndpoint(t *testing.T) {
	cfg := &config.Config{}
	generator := NewOpenAPIGenerator(cfg)

	endpoint := &EndpointInfo{
		Path:        "/test",
		Method:      "GET",
		Handler:     "testHandler",
		Summary:     "Test endpoint",
		Description: "Test endpoint description",
		Tags:        []string{"Test"},
		Responses: map[string]*Response{
			"200": {
				Description: "Success",
			},
		},
	}

	generator.AddEndpoint(endpoint)

	// Check if endpoint was added
	if pathItem, exists := generator.paths["/test"]; !exists {
		t.Error("Endpoint was not added to paths")
	} else if pathItem.GET == nil {
		t.Error("GET operation was not set")
	} else if pathItem.GET.Summary != "Test endpoint" {
		t.Error("Summary was not set correctly")
	}

	// Check if tag was added
	if _, exists := generator.tags["Test"]; !exists {
		t.Error("Tag was not added to tags map")
	}
}

func TestSchemaFromType(t *testing.T) {
	// Test string type
	stringSchema := SchemaFromType(reflect.TypeOf(""))
	if stringSchema.Type != "string" {
		t.Errorf("Expected string type, got %s", stringSchema.Type)
	}

	// Test integer type
	intSchema := SchemaFromType(reflect.TypeOf(42))
	if intSchema.Type != "integer" {
		t.Errorf("Expected integer type, got %s", intSchema.Type)
	}

	// Test boolean type
	boolSchema := SchemaFromType(reflect.TypeOf(true))
	if boolSchema.Type != "boolean" {
		t.Errorf("Expected boolean type, got %s", boolSchema.Type)
	}

	// Test slice type
	sliceSchema := SchemaFromType(reflect.TypeOf([]string{}))
	if sliceSchema.Type != "array" {
		t.Errorf("Expected array type, got %s", sliceSchema.Type)
	}
	if sliceSchema.Items == nil || sliceSchema.Items.Type != "string" {
		t.Error("Array items schema not set correctly")
	}

	// Test struct type
	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	structSchema := SchemaFromType(reflect.TypeOf(TestStruct{}))
	if structSchema.Type != "object" {
		t.Errorf("Expected object type, got %s", structSchema.Type)
	}
}

func TestGenerateOperationID(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		expected string
	}{
		{"GET", "/health", "getHealth"},
		{"POST", "/mcp", "postMcp"},
		{"GET", "/docs/openapi.yaml", "getDocsOpenapi.Yaml"},
		{"PUT", "/api/v1/users/{id}", "putApiV1Users"},
	}

	for _, test := range tests {
		result := generateOperationID(test.method, test.path)
		if result != test.expected {
			t.Errorf("For %s %s, expected %s, got %s", test.method, test.path, test.expected, result)
		}
	}
}

func TestGenerateTagDescription(t *testing.T) {
	tests := []struct {
		tag      string
		expected string
	}{
		{"MCP Protocol", "Model Context Protocol endpoints for memory operations"},
		{"Health", "Health check and status monitoring endpoints"},
		{"Custom", "Custom related endpoints"},
	}

	for _, test := range tests {
		result := generateTagDescription(test.tag)
		if result != test.expected {
			t.Errorf("For tag %s, expected %s, got %s", test.tag, test.expected, result)
		}
	}
}
