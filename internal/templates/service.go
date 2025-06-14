// Package templates provides template service integration for MCP Memory Server
package templates

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"lerian-mcp-memory/pkg/types"
)

// TemplateService provides template management and instantiation capabilities
type TemplateService struct {
	engine  *TemplateEngine
	storage TemplateStorage
	logger  *slog.Logger
}

// TemplateStorage interface for persisting template data
type TemplateStorage interface {
	StoreTemplateUsage(ctx context.Context, templateID, projectID string, success bool, metadata map[string]interface{}) error
	GetTemplateUsage(ctx context.Context, templateID string) (*TemplateUsageStats, error)
	StoreGeneratedTasks(ctx context.Context, result *TemplateInstantiationResult) error
	GetGeneratedTasks(ctx context.Context, projectID, templateID string) ([]GeneratedTask, error)
}

// TemplateUsageStats represents usage statistics for a template
type TemplateUsageStats struct {
	TemplateID      string    `json:"template_id"`
	UsageCount      int       `json:"usage_count"`
	SuccessCount    int       `json:"success_count"`
	FailureCount    int       `json:"failure_count"`
	SuccessRate     float64   `json:"success_rate"`
	LastUsed        time.Time `json:"last_used"`
	AverageTime     string    `json:"average_time"`
	PopularityScore float64   `json:"popularity_score"`
}

// NewTemplateService creates a new template service
func NewTemplateService(storage TemplateStorage, logger *slog.Logger) *TemplateService {
	if logger == nil {
		logger = slog.Default()
	}

	return &TemplateService{
		engine:  NewTemplateEngine(),
		storage: storage,
		logger:  logger,
	}
}

// ListTemplatesRequest represents a request to list templates
type ListTemplatesRequest struct {
	ProjectType types.ProjectType `json:"project_type,omitempty"`
	Category    string            `json:"category,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	PopularOnly bool              `json:"popular_only,omitempty"`
	Limit       int               `json:"limit,omitempty"`
}

// ListTemplatesResponse represents response from listing templates
type ListTemplatesResponse struct {
	Templates []TemplateInfo `json:"templates"`
	Total     int            `json:"total"`
	Filtered  int            `json:"filtered"`
}

// TemplateInfo represents template information with usage stats
type TemplateInfo struct {
	BuiltinTemplate
	UsageStats *TemplateUsageStats `json:"usage_stats,omitempty"`
}

// ListTemplates returns available templates with optional filtering
func (ts *TemplateService) ListTemplates(ctx context.Context, req *ListTemplatesRequest) (*ListTemplatesResponse, error) {
	ts.logger.Debug("listing templates",
		slog.String("project_type", string(req.ProjectType)),
		slog.String("category", req.Category))

	// Get all templates from engine
	allTemplates := ts.engine.ListTemplates()

	// Apply filters
	var filteredTemplates []BuiltinTemplate
	for i := range allTemplates {
		if ts.matchesFilters(&allTemplates[i], req) {
			filteredTemplates = append(filteredTemplates, allTemplates[i])
		}
	}

	// Convert to TemplateInfo with usage stats
	templateInfos := make([]TemplateInfo, 0, len(filteredTemplates))
	for i := range filteredTemplates {
		info := TemplateInfo{BuiltinTemplate: filteredTemplates[i]}

		// Get usage stats if storage is available
		if ts.storage != nil {
			if stats, err := ts.storage.GetTemplateUsage(ctx, filteredTemplates[i].ID); err == nil {
				info.UsageStats = stats
			}
		}

		templateInfos = append(templateInfos, info)
	}

	// Apply limit
	if req.Limit > 0 && len(templateInfos) > req.Limit {
		templateInfos = templateInfos[:req.Limit]
	}

	response := &ListTemplatesResponse{
		Templates: templateInfos,
		Total:     len(allTemplates),
		Filtered:  len(templateInfos),
	}

	ts.logger.Debug("templates listed",
		slog.Int("total", response.Total),
		slog.Int("filtered", response.Filtered))

	return response, nil
}

// matchesFilters checks if a template matches the request filters
func (ts *TemplateService) matchesFilters(tmpl *BuiltinTemplate, req *ListTemplatesRequest) bool {
	// Project type filter
	if req.ProjectType != "" && req.ProjectType != types.ProjectTypeAny {
		if tmpl.ProjectType != req.ProjectType && tmpl.ProjectType != types.ProjectTypeAny {
			return false
		}
	}

	// Category filter
	if req.Category != "" && tmpl.Category != req.Category {
		return false
	}

	// Tags filter
	if len(req.Tags) > 0 {
		hasMatchingTag := false
		for _, reqTag := range req.Tags {
			for _, tmplTag := range tmpl.Tags {
				if tmplTag == reqTag {
					hasMatchingTag = true
					break
				}
			}
			if hasMatchingTag {
				break
			}
		}
		if !hasMatchingTag {
			return false
		}
	}

	return true
}

// GetTemplate returns a specific template by ID
func (ts *TemplateService) GetTemplate(ctx context.Context, templateID string) (*TemplateInfo, error) {
	ts.logger.Debug("getting template", slog.String("template_id", templateID))

	tmpl, err := ts.engine.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	info := &TemplateInfo{BuiltinTemplate: *tmpl}

	// Get usage stats if storage is available
	if ts.storage != nil {
		if stats, err := ts.storage.GetTemplateUsage(ctx, templateID); err == nil {
			info.UsageStats = stats
		}
	}

	return info, nil
}

// InstantiateTemplate creates tasks from a template
func (ts *TemplateService) InstantiateTemplate(ctx context.Context, req *TemplateInstantiationRequest) (*TemplateInstantiationResult, error) {
	ts.logger.Info("instantiating template",
		slog.String("template_id", req.TemplateID),
		slog.String("project_id", req.ProjectID),
		slog.String("session_id", req.SessionID))

	// Record start time for usage tracking
	startTime := time.Now()

	// Instantiate template using engine
	result, err := ts.engine.InstantiateTemplate(req)
	if err != nil {
		// Record failure
		if ts.storage != nil {
			metadata := map[string]interface{}{
				"error":          err.Error(),
				"duration_ms":    time.Since(startTime).Milliseconds(),
				"variable_count": len(req.Variables),
			}
			_ = ts.storage.StoreTemplateUsage(ctx, req.TemplateID, req.ProjectID, false, metadata)
		}
		return nil, fmt.Errorf("template instantiation failed: %w", err)
	}

	// Record success
	if ts.storage != nil {
		metadata := map[string]interface{}{
			"task_count":     result.TaskCount,
			"estimated_time": result.EstimatedTime,
			"duration_ms":    time.Since(startTime).Milliseconds(),
			"variable_count": len(req.Variables),
			"warnings":       len(result.Warnings),
			"session_id":     req.SessionID,
		}

		// Store usage stats
		if err := ts.storage.StoreTemplateUsage(ctx, req.TemplateID, req.ProjectID, true, metadata); err != nil {
			ts.logger.Warn("failed to store template usage", slog.String("error", err.Error()))
		}

		// Store generated tasks
		if err := ts.storage.StoreGeneratedTasks(ctx, result); err != nil {
			ts.logger.Warn("failed to store generated tasks", slog.String("error", err.Error()))
		}
	}

	ts.logger.Info("template instantiated successfully",
		slog.String("template_id", req.TemplateID),
		slog.String("project_id", req.ProjectID),
		slog.Int("task_count", result.TaskCount),
		slog.String("estimated_time", result.EstimatedTime))

	return result, nil
}

// ValidateTemplateVariables validates variables for a template
func (ts *TemplateService) ValidateTemplateVariables(ctx context.Context, templateID string, variables map[string]interface{}) error {
	ts.logger.Debug("validating template variables",
		slog.String("template_id", templateID),
		slog.Int("variable_count", len(variables)))

	tmpl, err := ts.engine.GetTemplate(templateID)
	if err != nil {
		return err
	}

	return ts.engine.validateVariables(tmpl, variables)
}

// GetTemplateVariables returns the variables required by a template
func (ts *TemplateService) GetTemplateVariables(ctx context.Context, templateID string) ([]TemplateVariable, error) {
	ts.logger.Debug("getting template variables", slog.String("template_id", templateID))

	tmpl, err := ts.engine.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	return tmpl.Variables, nil
}

// SuggestTemplates suggests templates based on project characteristics
func (ts *TemplateService) SuggestTemplates(ctx context.Context, projectID string, projectType types.ProjectType, keywords []string) ([]TemplateInfo, error) {
	ts.logger.Debug("suggesting templates",
		slog.String("project_id", projectID),
		slog.String("project_type", string(projectType)),
		slog.Int("keyword_count", len(keywords)))

	// Get templates matching project type
	templates := ts.engine.ListTemplatesByProjectType(projectType)

	// Score templates based on keywords and usage
	var suggestions []TemplateInfo
	for i := range templates {
		score := ts.calculateRelevanceScore(&templates[i], keywords)
		if score > 0.3 { // Minimum relevance threshold
			info := TemplateInfo{BuiltinTemplate: templates[i]}

			// Get usage stats for popularity scoring
			if ts.storage != nil {
				if stats, err := ts.storage.GetTemplateUsage(ctx, templates[i].ID); err == nil {
					info.UsageStats = stats
					// Boost score based on popularity
					score += stats.PopularityScore * 0.2
				}
			}

			// Store score in metadata for sorting
			info.Metadata = make(map[string]interface{})
			info.Metadata["relevance_score"] = score

			suggestions = append(suggestions, info)
		}
	}

	// Sort by relevance score (highest first)
	for i := 0; i < len(suggestions); i++ {
		for j := i + 1; j < len(suggestions); j++ {
			scoreI := suggestions[i].Metadata["relevance_score"].(float64)
			scoreJ := suggestions[j].Metadata["relevance_score"].(float64)
			if scoreJ > scoreI {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}

	// Limit to top 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	ts.logger.Debug("templates suggested",
		slog.String("project_id", projectID),
		slog.Int("suggestion_count", len(suggestions)))

	return suggestions, nil
}

// calculateRelevanceScore calculates how relevant a template is based on keywords
func (ts *TemplateService) calculateRelevanceScore(tmpl *BuiltinTemplate, keywords []string) float64 {
	if len(keywords) == 0 {
		return 0.5 // Default score for no keywords
	}

	score := 0.0
	totalPossible := 0.0

	// Check name and description
	searchText := tmpl.Name + " " + tmpl.Description
	for _, keyword := range keywords {
		totalPossible += 1.0
		if containsIgnoreCase(searchText, keyword) {
			score += 1.0
		}
	}

	// Check tags
	for _, keyword := range keywords {
		for _, tag := range tmpl.Tags {
			if containsIgnoreCase(tag, keyword) {
				score += 0.5
				break
			}
		}
	}

	// Check category
	for _, keyword := range keywords {
		if containsIgnoreCase(tmpl.Category, keyword) {
			score += 0.3
		}
	}

	// Normalize score
	if totalPossible > 0 {
		score /= totalPossible
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// containsIgnoreCase checks if text contains substring case-insensitively
func containsIgnoreCase(text, substring string) bool {
	return len(text) >= len(substring) &&
		substring != ""
	// TODO: Implement proper case-insensitive contains
}

// GetGeneratedTasks returns tasks generated from templates for a project
func (ts *TemplateService) GetGeneratedTasks(ctx context.Context, projectID, templateID string) ([]GeneratedTask, error) {
	ts.logger.Debug("getting generated tasks",
		slog.String("project_id", projectID),
		slog.String("template_id", templateID))

	if ts.storage == nil {
		return []GeneratedTask{}, nil
	}

	return ts.storage.GetGeneratedTasks(ctx, projectID, templateID)
}

// GetTemplateUsageStats returns usage statistics for a template
func (ts *TemplateService) GetTemplateUsageStats(ctx context.Context, templateID string) (*TemplateUsageStats, error) {
	ts.logger.Debug("getting template usage stats", slog.String("template_id", templateID))

	if ts.storage == nil {
		return nil, errors.New("storage not available")
	}

	return ts.storage.GetTemplateUsage(ctx, templateID)
}

// ValidateTemplate validates a template structure
func (ts *TemplateService) ValidateTemplate(tmpl *BuiltinTemplate) []string {
	return ts.engine.ValidateTemplate(tmpl)
}

// AddCustomTemplate adds a custom template (for future extensibility)
func (ts *TemplateService) AddCustomTemplate(tmpl *BuiltinTemplate) error {
	ts.logger.Info("adding custom template",
		slog.String("template_id", tmpl.ID),
		slog.String("template_name", tmpl.Name))

	return ts.engine.AddCustomTemplate(tmpl)
}

// GetTemplateCategories returns all available template categories
func (ts *TemplateService) GetTemplateCategories() []string {
	templates := ts.engine.ListTemplates()
	categoryMap := make(map[string]bool)

	for i := range templates {
		if templates[i].Category != "" {
			categoryMap[templates[i].Category] = true
		}
	}

	categories := make([]string, 0, len(categoryMap))
	for category := range categoryMap {
		categories = append(categories, category)
	}

	return categories
}

// GetProjectTypes returns all supported project types
func (ts *TemplateService) GetProjectTypes() []types.ProjectType {
	return []types.ProjectType{
		types.ProjectTypeWeb,
		types.ProjectTypeAPI,
		types.ProjectTypeBackend,
		types.ProjectTypeFrontend,
		types.ProjectTypeMobile,
		types.ProjectTypeDesktop,
		types.ProjectTypeLibrary,
		types.ProjectTypeCLI,
		types.ProjectTypeAny,
	}
}
