package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"mcp-memory/internal/config"
	"mcp-memory/internal/mcp"
)

func main() {
	fmt.Println("🚀 Claude Vector Memory MCP Server - Demo")
	fmt.Println("==========================================")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		// Use default config for demo
		cfg = &config.Config{
			Chroma: config.ChromaConfig{
				Endpoint:   "http://localhost:9000",
				Collection: "memory",
			},
			OpenAI: config.OpenAIConfig{
				APIKey:         os.Getenv("OPENAI_API_KEY"),
				EmbeddingModel: "text-embedding-ada-002",
				MaxTokens:      8192,
			},
			Chunking: config.ChunkingConfig{
				Strategy:            "adaptive",
				MinContentLength:    100,
				MaxContentLength:    2000,
				SimilarityThreshold: 0.8,
			},
		}
	}

	// Create MCP server
	server, err := mcp.NewMemoryServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Start server
	ctx := context.Background()
	if err := server.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start server components: %v", err)
		fmt.Println("⚠️  Some components failed to start, but demo will continue...")
	}

	// Create tool executor
	executor := mcp.NewMCPToolExecutor(server)

	fmt.Println("\n📋 Available MCP Tools:")
	fmt.Println("=======================")
	tools := executor.ListAvailableTools()
	for i, tool := range tools {
		info := executor.GetToolInfo(tool)
		fmt.Printf("%d. %s\n   📝 %s\n", i+1, tool, info["description"])
	}

	fmt.Println("\n🎯 Running Tool Demonstrations:")
	fmt.Println("================================")

	// Run tool demos
	results := executor.DemoAllTools(ctx)

	for toolName, result := range results {
		fmt.Printf("\n🔧 %s:\n", toolName)
		fmt.Println("---")

		// Pretty print JSON result
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Printf("Error formatting result: %v\n", err)
		} else {
			fmt.Println(string(jsonBytes))
		}
	}

	fmt.Println("\n✅ Demo completed successfully!")
	fmt.Println("\n💡 Next Steps:")
	fmt.Println("- Set up OpenAI API key for full embedding functionality")
	fmt.Println("- Configure Chroma vector database for production use")
	fmt.Println("- Deploy using Docker Compose for full stack")
	fmt.Println("- Integrate with actual Claude MCP protocol")

	// Close server
	if err := server.Close(); err != nil {
		log.Printf("Warning: Error closing server: %v", err)
	}
}
