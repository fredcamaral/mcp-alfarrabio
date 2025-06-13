// Package templates provides MCP tools for template management
package templates

import (
	"context"

	"lerian-mcp-memory/internal/templates"

	"github.com/go-viper/mapstructure/v2"
)

// GetTemplateHandler handles template retrieval requests
type GetTemplateHandler struct {
	templateService *templates.TemplateService
}

// NewGetTemplateHandler creates a new get template handler
func NewGetTemplateHandler(templateService *templates.TemplateService) *GetTemplateHandler {
	return &GetTemplateHandler{
		templateService: templateService,
	}
}

// GetTemplateRequest represents the request parameters for getting a specific template
type GetTemplateRequest struct {
	ProjectID  string `json:"project_id,omitempty" mapstructure:"project_id"`
	SessionID  string `json:"session_id,omitempty" mapstructure:"session_id"`
	TemplateID string `json:"template_id" mapstructure:"template_id"`
}

// GetTemplateResponse represents the response from getting a template
type GetTemplateResponse struct {
	Template *templates.TemplateInfo `json:"template,omitempty"`
	Message  string                  `json:"message"`
}

// Handle processes the get template request
func (h *GetTemplateHandler) Handle(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	// Parse request parameters
	var req GetTemplateRequest
	if err := mapstructure.Decode(arguments, &req); err != nil {
		return GetTemplateResponse{
			Message: "Invalid request parameters: " + err.Error(),
		}, nil
	}

	// Validate required parameters
	if req.TemplateID == "" {
		return GetTemplateResponse{
			Message: "template_id is required",
		}, nil
	}

	// Call template service
	template, err := h.templateService.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		return GetTemplateResponse{
			Message: "Failed to get template: " + err.Error(),
		}, nil
	}

	response := GetTemplateResponse{
		Template: template,
		Message:  "Template retrieved successfully",
	}

	return response, nil
}

// GetToolDefinition returns the MCP tool definition for get_template
func (h *GetTemplateHandler) GetToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        "get_template",
		"description": "Get detailed information about a specific template including variables and tasks",
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
				"template_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the template to retrieve",
				},
			},
			"required": []string{"template_id"},
		},
	}
}
