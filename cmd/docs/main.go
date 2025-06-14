// docs is a command-line tool for generating OpenAPI documentation and specifications
// for the MCP Memory Server API endpoints.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/docs"
)

const (
	docsVersion = "1.0.0"
)

func main() {
	var (
		outputDir        = flag.String("output", "./docs/api", "Output directory for generated documentation")
		format           = flag.String("format", "json", "Output format: json, yaml, or both")
		validate         = flag.Bool("validate", true, "Validate the generated OpenAPI specification")
		serve            = flag.Bool("serve", false, "Start a local server to serve the documentation")
		port             = flag.Int("port", 8081, "Port for the documentation server")
		generateExamples = flag.Bool("examples", true, "Generate API examples")
		verbose          = flag.Bool("verbose", false, "Enable verbose logging")
		configFile       = flag.String("config", "", "Path to configuration file")
	)
	flag.Parse()

	// Setup logging
	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	log.Printf("MCP Memory Server Documentation Generator v%s", docsVersion)

	// Load configuration
	cfg, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create OpenAPI generator
	generator := docs.NewOpenAPIGenerator(cfg)

	// Generate documentation based on flags
	if *serve {
		err = serveDocumentation(generator, *port)
	} else {
		err = generateDocumentation(generator, *outputDir, *format, *validate, *generateExamples, *verbose)
	}

	if err != nil {
		log.Fatalf("Documentation generation failed: %v", err)
	}

	log.Println("Documentation generation completed successfully")
}

// loadConfig loads the application configuration
func loadConfig(configFile string) (*config.Config, error) {
	if configFile != "" {
		// Load from specific file
		return config.LoadConfig()
	}

	// Load default configuration
	return config.LoadConfig()
}

// generateDocumentation generates OpenAPI documentation files
func generateDocumentation(generator *docs.OpenAPIGenerator, outputDir, format string, validate, generateExamples, verbose bool) error {
	cleanOutputDir, err := validateOutputDirectory(outputDir)
	if err != nil {
		return err
	}

	log.Printf("Generating OpenAPI documentation to: %s", cleanOutputDir)

	if err := os.MkdirAll(cleanOutputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := validateSpecificationIfRequested(generator, validate); err != nil {
		return err
	}

	if err := generateSpecificationFiles(generator, cleanOutputDir, format, verbose); err != nil {
		return err
	}

	if err := generateAdditionalFiles(generator, cleanOutputDir, format, generateExamples, verbose); err != nil {
		return err
	}

	return nil
}

// validateOutputDirectory validates and cleans the output directory path
func validateOutputDirectory(outputDir string) (string, error) {
	cleanOutputDir := filepath.Clean(outputDir)

	if !filepath.IsAbs(cleanOutputDir) {
		return cleanOutputDir, nil
	}

	systemDirs := []string{"/etc/", "/usr/", "/bin/", "/sbin/", "/sys/", "/proc/", "/dev/"}
	for _, sysDir := range systemDirs {
		if strings.HasPrefix(cleanOutputDir, sysDir) {
			return "", errors.New("invalid output directory: access to system directory not allowed")
		}
	}

	return cleanOutputDir, nil
}

// validateSpecificationIfRequested validates the OpenAPI specification if requested
func validateSpecificationIfRequested(generator *docs.OpenAPIGenerator, validate bool) error {
	if !validate {
		return nil
	}

	log.Println("Validating OpenAPI specification...")
	if err := generator.ValidateSpecification(); err != nil {
		return fmt.Errorf("OpenAPI specification validation failed: %w", err)
	}
	log.Println("✓ OpenAPI specification validation passed")
	return nil
}

// generateSpecificationFiles generates the OpenAPI specification files in requested formats
func generateSpecificationFiles(generator *docs.OpenAPIGenerator, outputDir, format string, verbose bool) error {
	if format == "json" || format == "both" {
		if err := generateJSONSpec(generator, outputDir, verbose); err != nil {
			return fmt.Errorf("failed to generate JSON specification: %w", err)
		}
	}

	if format == "yaml" || format == "both" {
		if err := generateYAMLSpec(generator, outputDir, verbose); err != nil {
			return fmt.Errorf("failed to generate YAML specification: %w", err)
		}
	}

	return nil
}

// generateAdditionalFiles generates additional documentation files
func generateAdditionalFiles(generator *docs.OpenAPIGenerator, outputDir, format string, generateExamples, verbose bool) error {
	if generateExamples {
		if err := generateAPIExamples(generator, outputDir, verbose); err != nil {
			return fmt.Errorf("failed to generate API examples: %w", err)
		}
	}

	if err := generateStaticHTML(generator, outputDir, verbose); err != nil {
		return fmt.Errorf("failed to generate static HTML: %w", err)
	}

	if err := generateDocumentationReadme(outputDir, format); err != nil {
		return fmt.Errorf("failed to generate documentation README: %w", err)
	}

	return nil
}

// generateJSONSpec generates the OpenAPI specification in JSON format
func generateJSONSpec(generator *docs.OpenAPIGenerator, outputDir string, verbose bool) error {
	if verbose {
		log.Println("Generating OpenAPI JSON specification...")
	}

	jsonSpec, err := generator.GenerateJSON()
	if err != nil {
		return err
	}

	jsonPath := filepath.Join(outputDir, "openapi.json")
	if err := os.WriteFile(jsonPath, jsonSpec, 0o600); err != nil { // #nosec G304 -- Path is constructed safely using filepath.Join
		return fmt.Errorf("failed to write JSON specification: %w", err)
	}

	log.Printf("✓ Generated JSON specification: %s (%d bytes)", jsonPath, len(jsonSpec))
	return nil
}

// generateYAMLSpec generates the OpenAPI specification in YAML format
func generateYAMLSpec(generator *docs.OpenAPIGenerator, outputDir string, verbose bool) error {
	if verbose {
		log.Println("Generating OpenAPI YAML specification...")
	}

	yamlSpec, err := generator.GenerateYAML()
	if err != nil {
		return err
	}

	yamlPath := filepath.Join(outputDir, "openapi.yaml")
	if err := os.WriteFile(yamlPath, yamlSpec, 0o600); err != nil { // #nosec G304 -- Path is constructed safely using filepath.Join
		return fmt.Errorf("failed to write YAML specification: %w", err)
	}

	log.Printf("✓ Generated YAML specification: %s (%d bytes)", yamlPath, len(yamlSpec))
	return nil
}

// generateAPIExamples generates API request/response examples
func generateAPIExamples(generator *docs.OpenAPIGenerator, outputDir string, verbose bool) error {
	if verbose {
		log.Println("Generating API examples...")
	}

	exampleGenerator := docs.NewAPIExampleGenerator(generator)
	examples := exampleGenerator.GenerateExamples()

	// Convert examples to JSON
	jsonBytes, err := marshalIndent(examples, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal examples: %w", err)
	}

	examplesPath := filepath.Join(outputDir, "examples.json")
	if err := os.WriteFile(examplesPath, jsonBytes, 0o600); err != nil { // #nosec G304 -- Path is constructed safely using filepath.Join
		return fmt.Errorf("failed to write examples: %w", err)
	}

	log.Printf("✓ Generated API examples: %s", examplesPath)
	return nil
}

// generateStaticHTML generates a static HTML version of the documentation
func generateStaticHTML(generator *docs.OpenAPIGenerator, outputDir string, verbose bool) error {
	// Explicitly ignore generator parameter as it's not used in static HTML generation
	_ = generator
	if verbose {
		log.Println("Generating static HTML documentation...")
	}

	// Create a basic HTML file that loads Swagger UI
	htmlContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>MCP Memory Server API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui.css" />
    <style>
        body { margin: 0; background: #fafafa; }
        .swagger-ui .topbar { background-color: #1f2937; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5.10.5/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: './openapi.json',
            dom_id: '#swagger-ui',
            deepLinking: true,
            presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.presets.standalone],
            layout: "StandaloneLayout"
        });
    </script>
</body>
</html>`

	htmlPath := filepath.Join(outputDir, "index.html")
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0o600); err != nil { // #nosec G304 -- Path is constructed safely using filepath.Join
		return fmt.Errorf("failed to write HTML documentation: %w", err)
	}

	log.Printf("✓ Generated static HTML documentation: %s", htmlPath)
	return nil
}

// generateDocumentationReadme generates a README file for the documentation
func generateDocumentationReadme(outputDir, format string) error {
	readmeContent := `# MCP Memory Server API Documentation

This directory contains the automatically generated API documentation for the MCP Memory Server.

## Files

- **openapi.json** - OpenAPI 3.0 specification in JSON format
- **openapi.yaml** - OpenAPI 3.0 specification in YAML format  
- **examples.json** - API request/response examples
- **index.html** - Static HTML documentation (Swagger UI)

## Usage

### Viewing Documentation

1. **Static HTML**: Open ` + "`index.html`" + ` in a web browser
2. **JSON Spec**: Import ` + "`openapi.json`" + ` into your API client
3. **YAML Spec**: Use ` + "`openapi.yaml`" + ` with OpenAPI tools

### API Overview

The MCP Memory Server provides 41 memory tools for AI assistants:

- **Memory Search**: Search stored memories with similarity matching
- **Memory Storage**: Store conversation chunks and context
- **Memory Analysis**: Analyze patterns and relationships
- **Memory Management**: Organize and maintain memory data

### Authentication

The API supports multiple authentication methods:
- Bearer token (JWT)
- API key authentication

### Transport Protocols

- **HTTP**: Standard REST API endpoints
- **WebSocket**: Real-time bidirectional communication  
- **SSE**: Server-sent events for live updates

### Examples

See ` + "`examples.json`" + ` for comprehensive request/response examples for all endpoints.

## Generation

This documentation was generated using:
` + "```bash" + `
go run cmd/docs/main.go -output ./docs/api -format ` + format + `
` + "```" + `

For the latest documentation, run the generator again or visit the live API documentation at ` + "`/docs`" + `.

## Support

- GitHub: https://github.com/lerianstudio/lerian-mcp-memory
- Issues: https://github.com/lerianstudio/lerian-mcp-memory/issues
- Email: support@lerian.studio
`

	readmePath := filepath.Join(outputDir, "README.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0o600); err != nil { // #nosec G304 -- Path is constructed safely using filepath.Join
		return fmt.Errorf("failed to write README: %w", err)
	}

	log.Printf("✓ Generated documentation README: %s", readmePath)
	return nil
}

// serveDocumentation starts a local server to serve the documentation
func serveDocumentation(generator *docs.OpenAPIGenerator, port int) error {
	log.Printf("Starting documentation server on port %d...", port)
	log.Printf("Documentation will be available at: http://localhost:%d/docs", port)

	// This is a simplified implementation
	// In a real implementation, you would set up proper HTTP routes
	return errors.New("documentation server not implemented - use the main server with /docs endpoint")
}

// marshalIndent marshals the value with proper indentation
func marshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}
