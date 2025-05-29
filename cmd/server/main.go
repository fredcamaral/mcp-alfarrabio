package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/fredcamaral/gomcp-sdk/protocol"
	"github.com/fredcamaral/gomcp-sdk/server"
	"github.com/fredcamaral/gomcp-sdk/transport"
	"github.com/gorilla/websocket"
	"log"
	"mcp-memory/internal/config"
	"mcp-memory/internal/mcp"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"
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
			if !errors.Is(err, context.Canceled) {
				cancel()
				log.Printf("MCP server failed: %v", err)
				return
			}
		}

	case "http":
		log.Printf("🚀 Starting MCP Memory Server in HTTP mode on %s", *addr)
		log.Printf("📡 Ready to receive requests from mcp-proxy.js")
		// Set up HTTP server for MCP-over-HTTP
		if err := startHTTPServer(ctx, mcpServer, *addr); err != nil {
			if !errors.Is(err, context.Canceled) {
				cancel()
				log.Printf("HTTP server failed: %v", err)
				return
			}
		}

	default:
		cancel()
		log.Printf("Invalid mode: %s. Use 'stdio' or 'http'", *mode)
		return
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

	// Server-Sent Events endpoint for bidirectional MCP communication
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS preflight
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Cache-Control")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle POST requests for MCP JSON-RPC
		if r.Method == "POST" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "application/json")

			// Parse JSON-RPC request
			var req protocol.JSONRPCRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON-RPC request", http.StatusBadRequest)
				return
			}

			// Process MCP request
			resp := mcpServer.HandleRequest(r.Context(), &req)

			// Send JSON-RPC response
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Handle GET requests for SSE stream
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

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
		_, _ = fmt.Fprintf(w, "data: {\"type\":\"connected\",\"server\":\"mcp-memory\",\"protocols\":[\"json-rpc\",\"sse\"]}\n\n")
		flusher.Flush()

		// Keep connection open and send periodic heartbeats
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Send heartbeat
				_, _ = fmt.Fprintf(w, "data: {\"type\":\"heartbeat\",\"timestamp\":\"%s\"}\n\n", time.Now().UTC().Format(time.RFC3339))
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})

	// WebSocket upgrader
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin
		},
	}

	// WebSocket endpoint for bidirectional MCP communication
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Check if it's a WebSocket upgrade request
		if !strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") ||
			strings.ToLower(r.Header.Get("Upgrade")) != "websocket" {
			http.Error(w, "Expected WebSocket connection", http.StatusBadRequest)
			return
		}

		// Upgrade the HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		log.Printf("WebSocket connection established from %s", r.RemoteAddr)

		// Send initial connection message
		welcomeMsg := map[string]interface{}{
			"type":      "connected",
			"server":    "mcp-memory",
			"protocol":  "websocket",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		if err := conn.WriteJSON(welcomeMsg); err != nil {
			log.Printf("Failed to send welcome message: %v", err)
			return
		}

		// Handle WebSocket messages
		for {
			var req protocol.JSONRPCRequest
			if err := conn.ReadJSON(&req); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}

			log.Printf("Received WebSocket MCP request: %s", req.Method)

			// Process MCP request
			resp := mcpServer.HandleRequest(r.Context(), &req)

			// Send response back via WebSocket
			if err := conn.WriteJSON(resp); err != nil {
				log.Printf("Failed to send WebSocket response: %v", err)
				break
			}
		}

		log.Printf("WebSocket connection closed for %s", r.RemoteAddr)
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status": "healthy", "server": "mcp-memory", "mode": "development with hot-reload"}`)
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("✅ MCP Memory Server listening on http://localhost%s", addr)
		log.Printf("🔗 MCP endpoint: http://localhost%s/mcp", addr)
		log.Printf("📡 SSE endpoint: http://localhost%s/sse", addr)
		log.Printf("🔌 WebSocket endpoint: ws://localhost%s/ws", addr)
		log.Printf("💚 Health check: http://localhost%s/health", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Create a timeout context for shutdown
	// We use context.Background() here because the parent context is already cancelled
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second) //nolint:contextcheck
	defer cancel()

	// Shutdown server gracefully
	return server.Shutdown(shutdownCtx) //nolint:contextcheck // Parent context is already cancelled
}
