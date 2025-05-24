package main

import (
	"context"
	"fmt"
	"log"
	"mcp-memory/internal/config"
	"mcp-memory/internal/mcp"
	"os"
	"os/signal"
	"syscall"
)

func main() {
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

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the server
	if err := memoryServer.Start(ctx); err != nil {
		log.Fatalf("Failed to start memory server: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start MCP server in a goroutine
	go func() {
		log.Printf("Starting Claude Memory MCP Server on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := memoryServer.GetServer().Serve(); err != nil {
			log.Printf("MCP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Received shutdown signal, gracefully shutting down...")

	// Cancel context to stop operations
	cancel()

	// Close connections
	if err := memoryServer.Close(); err != nil {
		log.Printf("Error closing memory server: %v", err)
	}

	log.Println("Server shutdown complete")
}