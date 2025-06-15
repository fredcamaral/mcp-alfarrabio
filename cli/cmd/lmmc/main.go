package main

import (
	"fmt"
	"os"

	"lerian-mcp-memory-cli/internal/di"
)

func main() {
	os.Exit(run())
}

func run() int {
	// Initialize DI container
	container, err := di.NewContainer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize container: %v\n", err)
		return 1
	}
	defer func() {
		if err := container.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close container: %v\n", err)
		}
	}()

	// Run CLI application
	if err := container.CLI.Execute(); err != nil {
		// Error already formatted by CLI
		return 1
	}
	return 0
}
