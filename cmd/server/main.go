package main

import (
	"fmt"
	"log"
	"mcp-memory/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Claude Memory MCP Server Configuration Loaded Successfully!\n")
	fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Chroma: %s\n", cfg.Chroma.Endpoint)
	fmt.Printf("Collection: %s\n", cfg.Chroma.Collection)
	fmt.Printf("Embedding Model: %s\n", cfg.OpenAI.EmbeddingModel)
	fmt.Println("\nMCP Server implementation temporarily disabled for testing phase.")
	fmt.Println("Run tests with: make test-coverage")
}
