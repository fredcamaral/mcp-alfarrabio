package main

import (
	"fmt"
	"os"

	"lerian-mcp-memory-cli/internal/di"
)

func main() {
	// Initialize DI container
	container, err := di.NewContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize container: %v\n", err)
		os.Exit(1)
	}
	defer container.Close()

	// Run CLI application
	if err := container.CLI.Execute(); err != nil {
		// Error already formatted by CLI
		os.Exit(1)
	}
}
