// server is the main MCP Memory Server binary that provides persistent memory capabilities
// for AI assistants through multiple transport protocols (stdio, HTTP, WebSocket).
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"lerian-mcp-memory/internal/api"
	"lerian-mcp-memory/internal/config"
	"lerian-mcp-memory/internal/mcp"
	mcpwebsocket "lerian-mcp-memory/internal/websocket"
	"log"
	"net/http"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/fredcamaral/gomcp-sdk/protocol"
	"github.com/fredcamaral/gomcp-sdk/server"
	"github.com/fredcamaral/gomcp-sdk/transport"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// HTTP method constants
	methodOptions = "OPTIONS"

	// Default origins for CORS
	defaultLocalOrigin = "http://localhost:2001"
	defaultDevOrigin   = "http://localhost:3000"
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
	wsHub, _, err := initializeServerComponents(ctx)
	if err != nil {
		return err
	}

	// Setup HTTP routes
	handler := setupHTTPRoutes(ctx, mcpServer, wsHub)

	// Create and start HTTP server
	return startAndRunHTTPServer(ctx, handler, addr)
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

	memoryServer, err := mcp.NewMemoryServer(cfg) //nolint:contextcheck // NewMemoryServer doesn't require context parameter
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create memory server: %w", err)
	}

	// Initialize memory server
	if err := memoryServer.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start memory server: %w", err)
	}

	// Set the WebSocket hub in the memory server for broadcasting
	memoryServer.SetWebSocketHub(wsHub)

	return wsHub, memoryServer, nil
}

// setupHTTPRoutes configures all HTTP routes and handlers
func setupHTTPRoutes(ctx context.Context, mcpServer *server.Server, wsHub *mcpwebsocket.Hub) http.Handler {
	// Load configuration for the API router
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load config for API router: %v", err)
		// Fallback to legacy setup
		return setupLegacyHTTPRoutes(ctx, mcpServer, wsHub)
	}

	// Create new API router with Chi and middleware
	apiRouter := api.NewBasicRouter(cfg) //nolint:contextcheck // NewBasicRouter doesn't require context parameter

	// Create a new mux that combines the API router with legacy endpoints
	mux := http.NewServeMux()

	// Register specific endpoints first (before catch-all patterns)
	setupMCPHandler(mux, mcpServer)
	setupWebSocketHandler(mux, ctx, wsHub)

	// Mount the new API routes (with catch-all patterns last)
	mux.Handle("/api/", apiRouter.Handler())
	mux.Handle("/health", apiRouter.Handler())
	mux.Handle("/", apiRouter.Handler())

	return mux
}

// setupLegacyHTTPRoutes configures HTTP routes using the legacy approach
func setupLegacyHTTPRoutes(ctx context.Context, mcpServer *server.Server, wsHub *mcpwebsocket.Hub) *http.ServeMux {
	mux := http.NewServeMux()

	// Setup MCP endpoint
	setupMCPHandler(mux, mcpServer)

	// Setup WebSocket endpoint
	setupWebSocketHandler(mux, ctx, wsHub)

	// Setup health check endpoint
	setupHealthHandler(mux)

	return mux
}

// setupMCPHandler configures the MCP-over-HTTP endpoint
func setupMCPHandler(mux *http.ServeMux, mcpServer *server.Server) {
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		// Recover from panics to prevent server crashes
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic in MCP handler: %v", err)
				log.Printf("Stack trace: %s", debug.Stack())

				// Return error response
				errorResp := protocol.JSONRPCResponse{
					JSONRPC: "2.0",
					Error: &protocol.JSONRPCError{
						Code:    -32603,
						Message: "Internal server error",
						Data:    fmt.Sprintf("Server panic: %v", err),
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				if err := json.NewEncoder(w).Encode(errorResp); err != nil {
					log.Printf("Failed to encode error response: %v", err)
				}
			}
		}()

		// Debug: Log authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			log.Printf("MCP request with Authorization: %s", authHeader)
		} else {
			log.Printf("MCP request without Authorization header")
		}

		// Set CORS headers with specific origin to allow credentials
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = defaultLocalOrigin
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, "+methodOptions)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
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

// setupWebSocketHandler configures the WebSocket endpoint
func setupWebSocketHandler(mux *http.ServeMux, ctx context.Context, wsHub *mcpwebsocket.Hub) {
	// WebSocket upgrader with specific origin check
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Allow localhost connections for development
			return origin == defaultLocalOrigin || origin == defaultDevOrigin || origin == ""
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
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := fmt.Fprintf(w, `{"status": "healthy", "server": "lerian-mcp-memory", "mode": "development with hot-reload"}`); err != nil {
			log.Printf("Failed to write health check response: %v", err)
		}
	})
}

// startAndRunHTTPServer creates and runs the HTTP server
func startAndRunHTTPServer(ctx context.Context, handler http.Handler, addr string) error {
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0, // Disable write timeout for WebSocket compatibility
		IdleTimeout:       120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("----- WELCOME TO LERIAN MCP MEMORY SERVER -----")
		log.Printf("✅ MCP Memory Server listening on http://localhost%s", addr)
		log.Printf("🔗 MCP endpoint: http://localhost%s/mcp", addr)
		log.Printf("🔌 WebSocket endpoint: ws://localhost%s/ws", addr)
		log.Printf("💚 Health check: http://localhost%s/health", addr)
		log.Printf("📋 Use mcp-proxy.js for MCP stdio-to-HTTP communication")
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
