package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lerian-mcp-memory-cli/internal/di"
)

// Version information (set by build flags)
var (
	BuildVersion = "dev"
	BuildCommit  = "unknown"
	BuildDate    = "unknown"
)

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Initialize application container
	container, err := di.NewContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		return 1
	}

	// Set version information
	if container.CLI != nil && container.CLI.RootCmd != nil {
		container.CLI.RootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)",
			BuildVersion, BuildCommit, BuildDate)
	}

	// Health check
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer healthCancel()

	if err := container.HealthCheck(healthCtx); err != nil {
		container.Logger.Error("health check failed", "error", err)
		// Continue anyway - storage might still work
	}

	// Channel to receive shutdown result
	shutdownResult := make(chan int, 1)

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		container.Logger.Info("shutdown signal received")

		shutdownCtx, shutdownCancel := context.WithTimeout(
			ctx, 10*time.Second)
		defer shutdownCancel()

		if err := container.Shutdown(shutdownCtx); err != nil {
			container.Logger.Error("shutdown failed", "error", err)
			shutdownResult <- 1
			return
		}

		shutdownResult <- 0
	}()

	// Execute CLI command
	if err := container.CLI.Execute(); err != nil {
		// Error already printed by Cobra
		return 1
	}

	// Wait for shutdown signal or return success
	select {
	case exitCode := <-shutdownResult:
		return exitCode
	default:
		return 0
	}
}
