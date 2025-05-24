package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mcp-memory/internal/config"
	"mcp-memory/internal/mcp"
	"mcp-memory/pkg/mcp/protocol"
	"mcp-memory/pkg/mcp/server"
	"mcp-memory/pkg/mcp/transport"
	"net/http"
	"os/signal"
	"syscall"
)

func main() {
	// Parse command line flags
	var (
		mode = flag.String("mode", "stdio", "Server mode: stdio or http")
		addr = flag.String("addr", ":9080", "HTTP server address (when mode=http)")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create memory server
	memoryServer, err := mcp.NewMemoryServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create memory server: %v", err)
	}

	// Initialize the memory server components
	ctx := context.Background()
	if err := memoryServer.Start(ctx); err != nil {
		log.Fatalf("Failed to start memory server: %v", err)
	}

	// Get the underlying MCP server
	mcpServer := memoryServer.GetMCPServer()

	// Set up graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	switch *mode {
	case "stdio":
		log.Printf("🚀 Starting MCP Memory Server in stdio mode")
		// Set up stdio transport for MCP protocol
		transport := transport.NewStdioTransport()
		mcpServer.SetTransport(transport)

		// Start the MCP server
		if err := mcpServer.Start(ctx); err != nil {
			if err != context.Canceled {
				log.Fatalf("MCP server failed: %v", err)
			}
		}

	case "http":
		log.Printf("🚀 Starting MCP Memory Server in HTTP mode on %s", *addr)
		// Set up HTTP server for MCP-over-HTTP
		if err := startHTTPServer(ctx, mcpServer, *addr); err != nil {
			if err != context.Canceled {
				log.Fatalf("HTTP server failed: %v", err)
			}
		}

	default:
		log.Fatalf("Invalid mode: %s. Use 'stdio' or 'http'", *mode)
	}

	// Close resources
	if err := memoryServer.Close(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

func startHTTPServer(ctx context.Context, mcpServer *server.Server, addr string) error {
	// Create HTTP handler that processes MCP requests
	mux := http.NewServeMux()
	
	// Handle MCP-over-HTTP requests
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for remote access
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse the JSON-RPC request
		var req protocol.JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Process the request through MCP server
		resp := mcpServer.HandleRequest(r.Context(), &req)
		
		// Send the response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	})

	// Server-Sent Events endpoint for bidirectional communication
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

		// Keep connection alive
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Send initial connection message
		_, _ = fmt.Fprintf(w, "data: {\"type\":\"connected\",\"server\":\"mcp-memory\"}\n\n")
		flusher.Flush()

		// Keep connection open
		<-r.Context().Done()
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status": "healthy", "server": "mcp-memory"}`)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		log.Printf("✅ MCP Memory Server listening on http://localhost%s", addr)
		log.Printf("🔗 MCP endpoint: http://localhost%s/mcp", addr)
		log.Printf("📡 SSE endpoint: http://localhost%s/sse", addr)
		log.Printf("💚 Health check: http://localhost%s/health", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown server gracefully
	return server.Shutdown(context.Background())
}
