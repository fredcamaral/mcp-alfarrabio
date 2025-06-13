// Package templates provides MCP tools for template management
package templates

import (
	"context"

	"lerian-mcp-memory/internal/templates"
	"lerian-mcp-memory/pkg/types"

	"github.com/go-viper/mapstructure/v2"
)

// ListTemplatesHandler handles template listing requests
type ListTemplatesHandler struct {
	templateService *templates.TemplateService
}

// NewListTemplatesHandler creates a new list templates handler
func NewListTemplatesHandler(templateService *templates.TemplateService) *ListTemplatesHandler {
	return &ListTemplatesHandler{
		templateService: templateService,
	}
}

// ListTemplatesRequest represents the request parameters for listing templates
type ListTemplatesRequest struct {
	ProjectID   string            `json:"project_id,omitempty" mapstructure:"project_id"`
	SessionID   string            `json:"session_id,omitempty" mapstructure:"session_id"`
	ProjectType types.ProjectType `json:"project_type,omitempty" mapstructure:"project_type"`
	Category    string            `json:"category,omitempty" mapstructure:"category"`
	Tags        []string          `json:"tags,omitempty" mapstructure:"tags"`
	PopularOnly bool              `json:"popular_only,omitempty" mapstructure:"popular_only"`
	Limit       int               `json:"limit,omitempty" mapstructure:"limit"`
}

// ListTemplatesResponse represents the response from listing templates
type ListTemplatesResponse struct {
	Templates []templates.TemplateInfo `json:"templates"`
	Total     int                      `json:"total"`
	Filtered  int                      `json:"filtered"`
	Message   string                   `json:"message"`
}

// Handle processes the list templates request
func (h *ListTemplatesHandler) Handle(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	// Parse request parameters
	var req ListTemplatesRequest
	if err := mapstructure.Decode(arguments, &req); err != nil {
		return ListTemplatesResponse{
			Message: "Invalid request parameters: " + err.Error(),
		}, nil
	}

	// Set default limit if not provided
	if req.Limit == 0 {
		req.Limit = 20
	}

	// Create template service request
	serviceReq := &templates.ListTemplatesRequest{
		ProjectType: req.ProjectType,
		Category:    req.Category,
		Tags:        req.Tags,
		PopularOnly: req.PopularOnly,
		Limit:       req.Limit,
	}

	// Call template service
	result, err := h.templateService.ListTemplates(ctx, serviceReq)
	if err != nil {
		return ListTemplatesResponse{
			Message: "Failed to list templates: " + err.Error(),
		}, nil
	}

	// Build response message
	message := "Available templates retrieved successfully"
	if req.Category != "" {
		message += " for category: " + req.Category
	}
	if req.ProjectType != "" {
		message += " for project type: " + string(req.ProjectType)
	}

	response := ListTemplatesResponse{
		Templates: result.Templates,
		Total:     result.Total,
		Filtered:  result.Filtered,
		Message:   message,
	}

	return response, nil
}

// GetToolDefinition returns the MCP tool definition for list_templates
func (h *ListTemplatesHandler) GetToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        "list_templates",
		"description": "List available task templates with optional filtering by project type, category, or tags",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "Project identifier for context",
				},
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session identifier for scoped operations",
				},
				"project_type": map[string]interface{}{
					"type":        "string",
					"description": "Filter by project type (web, api, backend, frontend, mobile, desktop, library, cli, any)",
					"enum": []string{
						string(types.ProjectTypeWeb),
						string(types.ProjectTypeAPI),
						string(types.ProjectTypeBackend),
						string(types.ProjectTypeFrontend),
						string(types.ProjectTypeMobile),
						string(types.ProjectTypeDesktop),
						string(types.ProjectTypeLibrary),
						string(types.ProjectTypeCLI),
						string(types.ProjectTypeAny),
					},
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "Filter by template category (feature, api, maintenance, testing, etc.)",
					"enum": []string{
						"feature",
						"api",
						"maintenance",
						"testing",
						"documentation",
						"deployment",
						"security",
						"optimization",
						"refactoring",
						"infrastructure",
					},
				},
				"tags": map[string]interface{}{
					"type":        "array",
					"description": "Filter by tags (web, api, bug, etc.)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"popular_only": map[string]interface{}{
					"type":        "boolean",
					"description": "Show only popular templates based on usage statistics",
					"default":     false,
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of templates to return",
					"default":     20,
					"minimum":     1,
					"maximum":     100,
				},
			},
		},
	}
}
