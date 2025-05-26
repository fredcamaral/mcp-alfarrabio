package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: openapi <command>")
		fmt.Println("Commands:")
		fmt.Println("  serve    - Serve OpenAPI documentation")
		fmt.Println("  validate - Validate OpenAPI specification")
		fmt.Println("  generate - Generate code from OpenAPI spec")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		serveDocumentation()
	case "validate":
		validateSpec()
	case "generate":
		generateCode()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func serveDocumentation() {
	router := mux.NewRouter()

	// Serve OpenAPI spec
	router.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		spec, err := loadSpec()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(spec)
	})

	// Serve Swagger UI
	router.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>MCP Memory API Documentation</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@4/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@4/swagger-ui-bundle.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@4/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "/openapi.json",
                dom_id: '#swagger-ui',
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                layout: "StandaloneLayout"
            });
        }
    </script>
</body>
</html>
`
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(html))
	})

	// Redirect root to docs
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs", http.StatusTemporaryRedirect)
	})

	port := os.Getenv("OPENAPI_PORT")
	if port == "" {
		port = "8081"
	}

	fmt.Printf("Serving OpenAPI documentation at http://localhost:%s/docs\n", port)
	srv := &http.Server{Addr: ":" + port, Handler: router, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	log.Fatal(srv.ListenAndServe())
}

func validateSpec() {
	doc, err := loadSpec()
	if err != nil {
		fmt.Printf("Error loading spec: %v\n", err)
		os.Exit(1)
	}

	// Validate the spec
	if err := doc.Validate(openapi3.NewLoader().Context); err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ OpenAPI specification is valid")

	// Print statistics
	fmt.Printf("\nAPI Statistics:\n")
	fmt.Printf("- Paths: %d\n", doc.Paths.Len())
	fmt.Printf("- Schemas: %d\n", len(doc.Components.Schemas))
	fmt.Printf("- Operations: %d\n", countOperations(doc))
}

func generateCode() {
	fmt.Println("Code generation from OpenAPI spec")
	fmt.Println("This would generate:")
	fmt.Println("- Client SDKs (Go, TypeScript, Python)")
	fmt.Println("- Server stubs")
	fmt.Println("- Model definitions")
	fmt.Println("- Request/Response validators")
	fmt.Println("\nNote: Actual code generation requires additional tooling like openapi-generator")
}

func loadSpec() (*openapi3.T, error) {
	specPath := "api/openapi.yaml"
	if envPath := os.Getenv("OPENAPI_SPEC_PATH"); envPath != "" {
		specPath = envPath
	}

	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	// Parse YAML to JSON
	var specData interface{}
	if err := yaml.Unmarshal(data, &specData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	jsonData, err := json.Marshal(specData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to JSON: %w", err)
	}

	// Load OpenAPI document
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI document: %w", err)
	}

	return doc, nil
}

func countOperations(doc *openapi3.T) int {
	count := 0
	for _, pathItem := range doc.Paths.Map() {
		if pathItem.Get != nil {
			count++
		}
		if pathItem.Post != nil {
			count++
		}
		if pathItem.Put != nil {
			count++
		}
		if pathItem.Delete != nil {
			count++
		}
		if pathItem.Patch != nil {
			count++
		}
	}
	return count
}