package main

import (
	"context"
	"fmt"
	"log"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/roots"
	"mcp-memory/pkg/mcp/server"
	"mcp-memory/pkg/mcp/transport"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Create an extended MCP server (with sampling and roots)
	srv := server.NewExtendedServer("MCP Full Featured Demo", "1.0.0")
	
	// Add some demo tools
	srv.AddTool(
		protocol.Tool{
			Name:        "echo",
			Description: "Echoes back the input message",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Message to echo",
					},
				},
				"required": []string{"message"},
			},
		},
		protocol.ToolHandlerFunc(func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			message, ok := params["message"].(string)
			if !ok {
				return nil, fmt.Errorf("message parameter required")
			}
			return protocol.NewToolCallResult(protocol.NewContent("Echo: " + message)), nil
		}),
	)
	
	// Add a demo resource
	srv.AddResource(
		protocol.Resource{
			URI:         "demo://test.txt",
			Name:        "Test Resource",
			Description: "A demo resource for testing",
			MimeType:    "text/plain",
		},
		&demoResourceHandler{},
	)
	
	// Add a demo prompt
	srv.AddPrompt(
		protocol.Prompt{
			Name:        "greeting",
			Description: "Generate a greeting message",
			Arguments: []protocol.PromptArgument{
				{
					Name:        "name",
					Description: "Name of the person to greet",
					Required:    true,
				},
				{
					Name:        "style",
					Description: "Style of greeting (formal/casual)",
					Required:    false,
				},
			},
		},
		&demoPromptHandler{},
	)
	
	// Add custom roots
	srv.AddRoot(roots.Root{
		URI:         "demo://workspace",
		Name:        "Demo Workspace",
		Description: "Virtual workspace for demo purposes",
	})
	
	// Set up transport based on environment
	var t transport.Transport
	
	if os.Getenv("MCP_TRANSPORT") == "stdio" {
		log.Println("Starting MCP server with stdio transport...")
		t = transport.NewStdioTransport()
	} else {
		// Default to HTTP for testing
		port := os.Getenv("MCP_PORT")
		if port == "" {
			port = "3000"
		}
		
		log.Printf("Starting MCP server on http://localhost:%s", port)
		log.Println("Available endpoints:")
		log.Println("  POST /rpc - JSON-RPC endpoint")
		log.Println("  GET /sse - Server-sent events endpoint")
		log.Println("  WS /ws - WebSocket endpoint")
		
		httpConfig := &transport.HTTPConfig{
			Address:        ":" + port,
			Path:           "/rpc",
			EnableCORS:     true,
			AllowedOrigins: []string{"*"},
		}
		httpTransport := transport.NewHTTPTransport(httpConfig)
		
		t = httpTransport
	}
	
	srv.SetTransport(t)
	
	// Start server
	ctx := context.Background()
	
	// Start discovery watcher if plugin path is set
	if pluginPath := os.Getenv("MCP_PLUGIN_PATH"); pluginPath != "" {
		log.Printf("Watching for plugins in: %s", pluginPath)
		// This would be integrated into the full-featured server
		// For now, we'll skip the actual implementation
	}
	
	// Run server
	log.Println("Starting server...")
	if err := srv.Start(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	
	// Keep server running (wait for interrupt)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	
	log.Println("Shutting down...")
}

// demoResourceHandler implements server.ResourceHandler
type demoResourceHandler struct{}

func (h *demoResourceHandler) Handle(ctx context.Context, uri string) ([]protocol.Content, error) {
	content := fmt.Sprintf("This is demo content for resource: %s\nAccessed at: %s", uri, time.Now().Format(time.RFC3339))
	return []protocol.Content{protocol.NewContent(content)}, nil
}

// demoPromptHandler implements server.PromptHandler
type demoPromptHandler struct{}

func (h *demoPromptHandler) Handle(ctx context.Context, args map[string]interface{}) ([]protocol.Content, error) {
	name, _ := args["name"].(string)
	style, _ := args["style"].(string)
	
	if name == "" {
		name = "Friend"
	}
	
	var greeting string
	switch style {
	case "formal":
		greeting = fmt.Sprintf("Good day, %s. I hope this message finds you well.", name)
	case "casual":
		greeting = fmt.Sprintf("Hey %s! What's up?", name)
	default:
		greeting = fmt.Sprintf("Hello, %s!", name)
	}
	
	return []protocol.Content{protocol.NewContent(greeting)}, nil
}