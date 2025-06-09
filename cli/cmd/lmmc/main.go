package main

import (
	"fmt"
	"log/slog"
	"os"

	"lerian-mcp-memory-cli/internal/adapters/primary/cli"
	"lerian-mcp-memory-cli/internal/adapters/secondary/config"
	"lerian-mcp-memory-cli/internal/adapters/secondary/repository"
	"lerian-mcp-memory-cli/internal/adapters/secondary/storage"
	"lerian-mcp-memory-cli/internal/domain/services"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Default to warn, can be changed via config
	}))

	// Initialize configuration manager
	configMgr, err := config.NewViperConfigManager(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize storage
	fileStorage, err := storage.NewFileStorage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize storage: %v\n", err)
		os.Exit(1)
	}

	// Initialize repository detector
	repoDetector := repository.NewGitDetector()

	// Initialize task service
	taskService := services.NewTaskService(fileStorage, repoDetector, logger)

	// Create and run CLI
	cliApp := cli.NewCLI(taskService, configMgr, logger)
	if err := cliApp.Execute(); err != nil {
		// Error already formatted by CLI
		os.Exit(1)
	}
}
