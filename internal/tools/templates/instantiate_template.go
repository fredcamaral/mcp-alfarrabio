// Package templates provides MCP tools for template management
package templates

import (
	"context"
	"strconv"

	"lerian-mcp-memory/internal/templates"

	"github.com/go-viper/mapstructure/v2"
)

// InstantiateTemplateHandler handles template instantiation requests
type InstantiateTemplateHandler struct {
	templateService *templates.TemplateService
}

// NewInstantiateTemplateHandler creates a new instantiate template handler
func NewInstantiateTemplateHandler(templateService *templates.TemplateService) *InstantiateTemplateHandler {
	return &InstantiateTemplateHandler{
		templateService: templateService,
	}
}

// InstantiateTemplateRequest represents the request parameters for instantiating a template
type InstantiateTemplateRequest struct {
	ProjectID  string                 `json:"project_id" mapstructure:"project_id"`
	SessionID  string                 `json:"session_id,omitempty" mapstructure:"session_id"`
	TemplateID string                 `json:"template_id" mapstructure:"template_id"`
	Variables  map[string]interface{} `json:"variables,omitempty" mapstructure:"variables"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" mapstructure:"metadata"`
	Prefix     string                 `json:"prefix,omitempty" mapstructure:"prefix"`
}

// InstantiateTemplateResponse represents the response from instantiating a template
type InstantiateTemplateResponse struct {
	Result  *templates.TemplateInstantiationResult `json:"result,omitempty"`
	Message string                                 `json:"message"`
}

// Handle processes the instantiate template request
func (h *InstantiateTemplateHandler) Handle(ctx context.Context, arguments map[string]interface{}) (interface{}, error) {
	// Parse request parameters
	var req InstantiateTemplateRequest
	if err := mapstructure.Decode(arguments, &req); err != nil {
		return InstantiateTemplateResponse{
			Message: "Invalid request parameters: " + err.Error(),
		}, nil
	}

	// Validate required parameters
	if req.TemplateID == "" {
		return InstantiateTemplateResponse{
			Message: "template_id is required",
		}, nil
	}

	if req.ProjectID == "" {
		return InstantiateTemplateResponse{
			Message: "project_id is required",
		}, nil
	}

	// Initialize variables if nil
	if req.Variables == nil {
		req.Variables = make(map[string]interface{})
	}

	// Create template service request
	serviceReq := &templates.TemplateInstantiationRequest{
		TemplateID: req.TemplateID,
		ProjectID:  req.ProjectID,
		SessionID:  req.SessionID,
		Variables:  req.Variables,
		Metadata:   req.Metadata,
		Prefix:     req.Prefix,
	}

	// Call template service
	result, err := h.templateService.InstantiateTemplate(ctx, serviceReq)
	if err != nil {
		return InstantiateTemplateResponse{
			Message: "Failed to instantiate template: " + err.Error(),
		}, nil
	}

	// Build success message
	message := "Template instantiated successfully"
	if result.TaskCount > 0 {
		message += " with " + strconv.Itoa(result.TaskCount) + " tasks"
	}
	if result.EstimatedTime != "" {
		message += " (estimated time: " + result.EstimatedTime + ")"
	}
	if len(result.Warnings) > 0 {
		message += " with " + strconv.Itoa(len(result.Warnings)) + " warnings"
	}

	response := InstantiateTemplateResponse{
		Result:  result,
		Message: message,
	}

	return response, nil
}

// GetToolDefinition returns the MCP tool definition for instantiate_template
func (h *InstantiateTemplateHandler) GetToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        "instantiate_template",
		"description": "Instantiate a template to generate tasks with provided variables",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "Project identifier where tasks will be created",
				},
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session identifier for scoped operations",
				},
				"template_id": map[string]interface{}{
					"type":        "string",
					"description": "ID of the template to instantiate",
				},
				"variables": map[string]interface{}{
					"type":                 "object",
					"description":          "Variables to substitute in the template",
					"additionalProperties": true,
				},
				"metadata": map[string]interface{}{
					"type":                 "object",
					"description":          "Additional metadata to attach to generated tasks",
					"additionalProperties": true,
				},
				"prefix": map[string]interface{}{
					"type":        "string",
					"description": "Optional prefix to add to task names",
				},
			},
			"required": []string{"project_id", "template_id"},
		},
	}
}
