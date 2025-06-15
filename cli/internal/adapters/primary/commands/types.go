// Package commands provides primary command interfaces and structures for the CLI.
// This file contains common types and interfaces used across command handlers.
package commands

import (
	"log/slog"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/services"
)

// CommandDeps holds all dependencies needed by command implementations
type CommandDeps struct {
	// Core services
	TaskService *services.TaskService
	Logger      *slog.Logger
	Config      *entities.Config

	// Intelligence services (optional)
	AnalyticsService  services.AnalyticsService
	PatternDetector   services.PatternDetector
	SuggestionService services.SuggestionService
	TemplateService   services.TemplateService
	CrossRepoAnalyzer services.CrossRepoAnalyzer
}
