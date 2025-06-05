// server is the main MCP Memory Server binary that provides persistent memory capabilities
// for AI assistants through multiple transport protocols (stdio, HTTP, WebSocket, SSE).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"lerian-mcp-memory/internal/config"
	mcpgraphql "lerian-mcp-memory/internal/graphql"
	"lerian-mcp-memory/internal/mcp"
	mcpwebsocket "lerian-mcp-memory/internal/websocket"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/fredcamaral/gomcp-sdk/protocol"
	"github.com/fredcamaral/gomcp-sdk/server"
	"github.com/fredcamaral/gomcp-sdk/transport"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/handler"
)

const (
	// HTTP method constants
	methodOptions = "OPTIONS"
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
		stdioTransport := transport.NewStdioTransport()
		mcpServer.SetTransport(stdioTransport)

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
	// Initialize core components
	wsHub, memoryServer, err := initializeServerComponents(ctx)
	if err != nil {
		return err
	}

	// Setup HTTP routes
	mux := setupHTTPRoutes(ctx, mcpServer, wsHub, memoryServer)

	// Create and start HTTP server
	return startAndRunHTTPServer(ctx, mux, addr)
}

// initializeServerComponents initializes WebSocket hub and memory server
func initializeServerComponents(ctx context.Context) (*mcpwebsocket.Hub, *mcp.MemoryServer, error) {
	// Create WebSocket hub for real-time updates
	wsHub := mcpwebsocket.NewHub()
	go wsHub.Run(ctx)

	// Create memory server instance to access DI container
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config for GraphQL: %w", err)
	}

	memoryServer, err := mcp.NewMemoryServer(cfg) //nolint:contextcheck // Constructor doesn't need context, Start() method does
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create memory server for GraphQL: %w", err)
	}

	// Initialize memory server
	if err := memoryServer.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start memory server for GraphQL: %w", err)
	}

	// Set the WebSocket hub in the memory server for broadcasting
	memoryServer.SetWebSocketHub(wsHub)

	return wsHub, memoryServer, nil
}

// setupHTTPRoutes configures all HTTP routes and handlers
func setupHTTPRoutes(ctx context.Context, mcpServer *server.Server, wsHub *mcpwebsocket.Hub, memoryServer *mcp.MemoryServer) *http.ServeMux {
	mux := http.NewServeMux()

	// Setup GraphQL endpoint
	setupGraphQLHandler(mux, memoryServer)

	// Setup MCP endpoint
	setupMCPHandler(mux, mcpServer)

	// Setup SSE endpoint
	setupSSEHandler(mux, mcpServer)

	// Setup WebSocket endpoint
	setupWebSocketHandler(mux, ctx, wsHub)

	// Setup health check endpoint
	setupHealthHandler(mux)

	return mux
}

// setupGraphQLHandler configures the GraphQL endpoint
func setupGraphQLHandler(mux *http.ServeMux, memoryServer *mcp.MemoryServer) {
	container := memoryServer.GetContainer()
	schema, err := mcpgraphql.NewSchema(container)
	if err != nil {
		log.Printf("Warning: Failed to create GraphQL schema: %v", err)
		// Add a fallback GraphQL endpoint
		mux.HandleFunc("/graphql", createFallbackGraphQLHandler(err))
	} else {
		// Create GraphQL handler with the schema
		graphqlSchema := schema.GetSchema()
		h := handler.New(&handler.Config{
			Schema:     &graphqlSchema,
			Pretty:     true,
			GraphiQL:   true,
			Playground: true,
		})

		// Add GraphQL endpoint with CORS
		mux.HandleFunc("/graphql", createGraphQLHandler(h))
	}
}

// createFallbackGraphQLHandler creates a fallback handler when GraphQL schema creation fails
func createFallbackGraphQLHandler(schemaErr error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, "+methodOptions)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == methodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusServiceUnavailable)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"error":   "GraphQL service unavailable",
			"message": fmt.Sprintf("Schema creation failed: %v", schemaErr),
		}); err != nil {
			log.Printf("Error encoding GraphQL error response: %v", err)
		}
	}
}

// createGraphQLHandler creates the main GraphQL handler
func createGraphQLHandler(h *handler.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, "+methodOptions)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == methodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Delegate to GraphQL handler
		h.ServeHTTP(w, r)
	}
}

// setupMCPHandler configures the MCP-over-HTTP endpoint
func setupMCPHandler(mux *http.ServeMux, mcpServer *server.Server) {
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers for remote access
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, "+methodOptions)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == methodOptions {
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
}

// setupSSEHandler configures the Server-Sent Events endpoint
func setupSSEHandler(mux *http.ServeMux, mcpServer *server.Server) {
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS preflight
		if r.Method == methodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, "+methodOptions)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Cache-Control")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle POST requests for MCP JSON-RPC
		if r.Method == "POST" {
			handleSSEPost(w, r, mcpServer)
			return
		}

		// Handle GET requests for SSE stream
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		handleSSEStream(w, r)
	})
}

// handleSSEPost handles POST requests to the SSE endpoint
func handleSSEPost(w http.ResponseWriter, r *http.Request, mcpServer *server.Server) {
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
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding SSE response: %v", err)
	}
}

// handleSSEStream handles GET requests for SSE streaming
func handleSSEStream(w http.ResponseWriter, r *http.Request) {
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
	_, _ = fmt.Fprintf(w, "data: {\"type\":\"connected\",\"server\":\"lerian-mcp-memory\",\"protocols\":[\"json-rpc\",\"sse\"]}\n\n")
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
}

// setupWebSocketHandler configures the WebSocket endpoint
func setupWebSocketHandler(mux *http.ServeMux, ctx context.Context, wsHub *mcpwebsocket.Hub) {
	// WebSocket upgrader
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool {
			return true // Allow connections from any origin
		},
	}

	// WebSocket endpoint for real-time memory updates
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

		// Get client preferences from query parameters
		repository := r.URL.Query().Get("repository")
		sessionID := r.URL.Query().Get("session_id")

		// Create a new client
		clientID := uuid.New().String()
		client := mcpwebsocket.NewClient(clientID, conn, wsHub, repository, sessionID)

		// Register client with hub
		wsHub.RegisterClient(client)

		// Start goroutines for reading and writing
		go client.WritePump(ctx)
		go client.ReadPump(ctx)

		log.Printf("WebSocket client %s connected from %s", clientID, r.RemoteAddr)
	})
}

// setupHealthHandler configures the health check endpoint
func setupHealthHandler(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status": "healthy", "server": "lerian-mcp-memory", "mode": "development with hot-reload"}`)
	})
}

// startAndRunHTTPServer creates and runs the HTTP server
func startAndRunHTTPServer(ctx context.Context, mux *http.ServeMux, addr string) error {
	httpServer := &http.Server{
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
		log.Printf("🎨 GraphQL endpoint: http://localhost%s/graphql", addr)
		log.Printf("💚 Health check: http://localhost%s/health", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Create a timeout context for shutdown
	// We use context.Background() here because the parent context is already cancelled
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	return httpServer.Shutdown(shutdownCtx) //nolint:contextcheck // Fresh context needed for shutdown when parent is cancelled
}
