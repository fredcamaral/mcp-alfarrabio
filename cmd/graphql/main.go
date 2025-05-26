package main

import (
	"context"
	"fmt"
	"log"
	"mcp-memory/internal/config"
	"mcp-memory/internal/di"
	mcpgraphql "mcp-memory/internal/graphql"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/graphql-go/handler"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create DI container
	container, err := di.NewContainer(cfg)
	if err != nil {
		log.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Shutdown() }()

	// Initialize services
	ctx := context.Background()
	
	// Initialize vector store first
	if err := container.GetVectorStore().Initialize(ctx); err != nil {
		_ = container.Shutdown()
		log.Fatalf("Failed to initialize vector store: %v", err)
	}
	
	// Then do health check
	if err := container.HealthCheck(ctx); err != nil {
		log.Printf("Warning: Health check failed: %v", err)
	}

	// Create GraphQL schema
	schema, err := mcpgraphql.NewSchema(container)
	if err != nil {
		_ = container.Shutdown()
		log.Fatalf("Failed to create GraphQL schema: %v", err)
	}

	// Create GraphQL handler
	graphqlSchema := schema.GetSchema()
	h := handler.New(&handler.Config{
		Schema:     &graphqlSchema,
		Pretty:     true,
		GraphiQL:   true,
		Playground: true,
	})

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.Handle("/graphql", h)
	mux.HandleFunc("/health", healthHandler(container))
	
	// Serve static files for web UI
	staticDir := "./web/static"
	fileServer := http.FileServer(http.Dir(staticDir))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	
	// Serve index.html at root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	})

	// Add CORS middleware
	corsHandler := cors(mux)

	// Start server
	port := os.Getenv("GRAPHQL_PORT")
	if port == "" {
		port = "8082"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      corsHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("GraphQL server starting on http://localhost:%s/graphql", port)
		log.Printf("Web UI available at http://localhost:%s/", port)
		log.Printf("GraphiQL playground available at http://localhost:%s/graphql", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server failed to start: %v", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// healthHandler returns a health check endpoint
func healthHandler(container *di.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if err := container.HealthCheck(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, "Health check failed: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, "OK")
	}
}

// cors adds CORS headers
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}