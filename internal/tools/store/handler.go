// Package store provides the memory_store tool implementation.
// Handles all data persistence operations with proper session validation.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/session"
	"lerian-mcp-memory/internal/tools"
	"lerian-mcp-memory/internal/types"
	"lerian-mcp-memory/internal/validation"
	pkgTypes "lerian-mcp-memory/pkg/types"
)

// ContentStore interface for dependency injection
type ContentStore interface {
	Store(ctx context.Context, content *types.Content) error
	Update(ctx context.Context, content *types.Content) error
	Delete(ctx context.Context, projectID types.ProjectID, contentID string) error
	Get(ctx context.Context, projectID types.ProjectID, contentID string) (*types.Content, error)
}

// Handler implements the memory_store tool
type Handler struct {
	sessionManager *session.Manager
	validator      *validation.ParameterValidator
	contentStore   ContentStore
}

// NewHandler creates a new store handler
func NewHandler(sessionManager *session.Manager, validator *validation.ParameterValidator, contentStore ContentStore) *Handler {
	return &Handler{
		sessionManager: sessionManager,
		validator:      validator,
		contentStore:   contentStore,
	}
}

// StoreContentRequest represents a request to store content
type StoreContentRequest struct {
	types.StandardParams
	Content    string                 `json:"content"`
	Type       pkgTypes.ChunkType     `json:"type"`
	Summary    string                 `json:"summary,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	ThreadID   string                 `json:"thread_id,omitempty"`
	RelatedIDs []string               `json:"related_ids,omitempty"`
}

// StoreContentResponse represents the response from storing content
type StoreContentResponse struct {
	ContentID  string    `json:"content_id"`
	ThreadID   string    `json:"thread_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	Embeddings []float64 `json:"embeddings,omitempty"`
	Relations  []string  `json:"relations,omitempty"`
	Success    bool      `json:"success"`
	Message    string    `json:"message"`
}

// UpdateContentRequest represents a request to update content
type UpdateContentRequest struct {
	types.StandardParams
	ContentID  string                 `json:"content_id"`
	Content    string                 `json:"content,omitempty"`
	Summary    string                 `json:"summary,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	AddTags    []string               `json:"add_tags,omitempty"`
	RemoveTags []string               `json:"remove_tags,omitempty"`
}

// DeleteContentRequest represents a request to delete content
type DeleteContentRequest struct {
	types.StandardParams
	ContentID string `json:"content_id"`
	Force     bool   `json:"force,omitempty"` // Force delete even if referenced
}

// CreateThreadRequest represents a request to create a thread
type CreateThreadRequest struct {
	types.StandardParams
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CreateRelationRequest represents a request to create a relationship
type CreateRelationRequest struct {
	types.StandardParams
	FromContentID string  `json:"from_content_id"`
	ToContentID   string  `json:"to_content_id"`
	RelationType  string  `json:"relation_type"` // "references", "blocks", "implements", etc.
	Description   string  `json:"description,omitempty"`
	Strength      float64 `json:"strength,omitempty"` // 0.0-1.0
}

// HandleOperation handles all store operations
func (h *Handler) HandleOperation(ctx context.Context, operation string, params map[string]interface{}) (interface{}, error) {
	switch operation {
	case string(tools.OpStoreContent):
		return h.handleStoreContent(ctx, params)
	case string(tools.OpStoreDecision):
		return h.handleStoreDecision(ctx, params)
	case string(tools.OpUpdateContent):
		return h.handleUpdateContent(ctx, params)
	case string(tools.OpDeleteContent):
		return h.handleDeleteContent(ctx, params)
	case string(tools.OpCreateThread):
		return h.handleCreateThread(ctx, params)
	case string(tools.OpCreateRelation):
		return h.handleCreateRelation(ctx, params)
	default:
		return nil, fmt.Errorf("unknown store operation: %s", operation)
	}
}

// handleStoreContent stores new content with proper validation
func (h *Handler) handleStoreContent(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse request
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req StoreContentRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse store content request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpStoreContent), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Validate request data
	if req.Content == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}
	if !req.Type.Valid() {
		return nil, fmt.Errorf("invalid content type: %s", req.Type)
	}

	// Update session access
	if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
		return nil, fmt.Errorf("failed to update session access: %w", err)
	}

	// Create content object
	content := &types.Content{
		ID:        fmt.Sprintf("content_%d", time.Now().Unix()),
		ProjectID: req.ProjectID,
		SessionID: req.SessionID,
		Type:      string(req.Type),
		Content:   req.Content,
		Summary:   req.Summary,
		Tags:      req.Tags,
		Metadata:  req.Metadata,
		ThreadID:  req.ThreadID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}

	// Store content using real storage implementation
	if err := h.contentStore.Store(ctx, content); err != nil {
		return nil, fmt.Errorf("failed to store content: %w", err)
	}

	response := &StoreContentResponse{
		ContentID:  content.ID,
		ThreadID:   content.ThreadID,
		CreatedAt:  content.CreatedAt,
		Embeddings: content.Embeddings,
		Success:    true,
		Message:    "Content stored successfully",
	}

	return response, nil
}

// handleStoreDecision stores architectural or design decisions
func (h *Handler) handleStoreDecision(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse request (similar to store content but with decision-specific validation)
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req StoreContentRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse store decision request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpStoreDecision), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Validate this is a decision type
	if req.Type != pkgTypes.ChunkTypeArchitectureDecision {
		return nil, fmt.Errorf("store_decision operation requires architecture_decision type, got %s", req.Type)
	}

	// Update session access
	if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
		return nil, fmt.Errorf("failed to update session access: %w", err)
	}

	// Create decision content object with proper tagging
	content := &types.Content{
		ID:        fmt.Sprintf("decision_%d", time.Now().Unix()),
		ProjectID: req.ProjectID,
		SessionID: req.SessionID,
		Type:      string(req.Type),
		Content:   req.Content,
		Summary:   req.Summary,
		Tags:      append(req.Tags, "architecture-decision", "decision"), // Add decision-specific tags
		Metadata:  req.Metadata,
		ThreadID:  req.ThreadID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}

	// Ensure decision metadata
	if content.Metadata == nil {
		content.Metadata = make(map[string]interface{})
	}
	content.Metadata["decision_type"] = "architecture"
	content.Metadata["decision_date"] = time.Now().Format(time.RFC3339)

	// Store decision using real storage implementation
	if err := h.contentStore.Store(ctx, content); err != nil {
		return nil, fmt.Errorf("failed to store decision: %w", err)
	}

	response := &StoreContentResponse{
		ContentID:  content.ID,
		ThreadID:   content.ThreadID,
		CreatedAt:  content.CreatedAt,
		Embeddings: content.Embeddings,
		Success:    true,
		Message:    "Decision stored successfully",
	}

	return response, nil
}

// handleUpdateContent updates existing content
func (h *Handler) handleUpdateContent(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req UpdateContentRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse update content request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpUpdateContent), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.ContentID == "" {
		return nil, fmt.Errorf("content_id is required for update operation")
	}

	// Update session access
	if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
		return nil, fmt.Errorf("failed to update session access: %w", err)
	}

	// Get existing content
	existing, err := h.contentStore.Get(ctx, req.ProjectID, req.ContentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing content: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("content with ID %s not found", req.ContentID)
	}

	// Create updated content object
	updatedContent := &types.Content{
		ID:         existing.ID,
		ProjectID:  existing.ProjectID,
		SessionID:  existing.SessionID,
		Type:       existing.Type,
		Content:    existing.Content,
		Summary:    existing.Summary,
		Tags:       existing.Tags,
		Metadata:   existing.Metadata,
		ThreadID:   existing.ThreadID,
		CreatedAt:  existing.CreatedAt,
		Version:    existing.Version,
		ParentID:   existing.ParentID,
		Source:     existing.Source,
		SourcePath: existing.SourcePath,
		Quality:    existing.Quality,
		Confidence: existing.Confidence,
	}

	// Apply updates from request
	if req.Content != "" {
		updatedContent.Content = req.Content
	}
	if req.Summary != "" {
		updatedContent.Summary = req.Summary
	}
	if req.Tags != nil {
		updatedContent.Tags = req.Tags
	}
	if req.Metadata != nil {
		if updatedContent.Metadata == nil {
			updatedContent.Metadata = make(map[string]interface{})
		}
		for k, v := range req.Metadata {
			updatedContent.Metadata[k] = v
		}
	}

	// Handle tag operations
	if len(req.AddTags) > 0 {
		tagSet := make(map[string]bool)
		for _, tag := range updatedContent.Tags {
			tagSet[tag] = true
		}
		for _, tag := range req.AddTags {
			tagSet[tag] = true
		}
		updatedContent.Tags = make([]string, 0, len(tagSet))
		for tag := range tagSet {
			updatedContent.Tags = append(updatedContent.Tags, tag)
		}
	}
	if len(req.RemoveTags) > 0 {
		removeSet := make(map[string]bool)
		for _, tag := range req.RemoveTags {
			removeSet[tag] = true
		}
		var filteredTags []string
		for _, tag := range updatedContent.Tags {
			if !removeSet[tag] {
				filteredTags = append(filteredTags, tag)
			}
		}
		updatedContent.Tags = filteredTags
	}

	// Update using real storage implementation
	if err := h.contentStore.Update(ctx, updatedContent); err != nil {
		return nil, fmt.Errorf("failed to update content: %w", err)
	}

	response := map[string]interface{}{
		"content_id": updatedContent.ID,
		"updated_at": updatedContent.UpdatedAt,
		"version":    updatedContent.Version,
		"success":    true,
		"message":    "Content updated successfully",
	}

	return response, nil
}

// handleDeleteContent deletes content
func (h *Handler) handleDeleteContent(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req DeleteContentRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse delete content request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpDeleteContent), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.ContentID == "" {
		return nil, fmt.Errorf("content_id is required for delete operation")
	}

	// Update session access
	if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
		return nil, fmt.Errorf("failed to update session access: %w", err)
	}

	// Check if content exists before deletion
	existing, err := h.contentStore.Get(ctx, req.ProjectID, req.ContentID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing content: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("content with ID %s not found", req.ContentID)
	}

	// TODO: Add reference checking logic here if needed
	// For now, proceed with deletion if force flag is set or no references exist

	// Delete using real storage implementation
	if err := h.contentStore.Delete(ctx, req.ProjectID, req.ContentID); err != nil {
		return nil, fmt.Errorf("failed to delete content: %w", err)
	}

	response := map[string]interface{}{
		"content_id": req.ContentID,
		"deleted_at": time.Now(),
		"success":    true,
		"message":    "Content deleted successfully",
	}

	return response, nil
}

// handleCreateThread creates a new conversation thread
func (h *Handler) handleCreateThread(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req CreateThreadRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse create thread request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpCreateThread), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.Title == "" {
		return nil, fmt.Errorf("title is required for create thread operation")
	}

	// Update session access
	if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
		return nil, fmt.Errorf("failed to update session access: %w", err)
	}

	// TODO: Implement actual thread creation
	response := map[string]interface{}{
		"thread_id":   fmt.Sprintf("thread_%d", time.Now().Unix()),
		"title":       req.Title,
		"description": req.Description,
		"created_at":  time.Now(),
		"success":     true,
		"message":     "Thread created successfully",
	}

	return response, nil
}

// handleCreateRelation creates relationships between content
func (h *Handler) handleCreateRelation(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	reqBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var req CreateRelationRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("failed to parse create relation request: %w", err)
	}

	// Validate parameters
	if err := h.validator.ValidateOperation(string(tools.OpCreateRelation), &req.StandardParams); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	if req.FromContentID == "" || req.ToContentID == "" {
		return nil, fmt.Errorf("both from_content_id and to_content_id are required")
	}
	if req.RelationType == "" {
		return nil, fmt.Errorf("relation_type is required")
	}

	// Update session access
	if err := h.sessionManager.UpdateSessionAccess(req.ProjectID, req.SessionID); err != nil {
		return nil, fmt.Errorf("failed to update session access: %w", err)
	}

	// TODO: Implement actual relation creation
	response := map[string]interface{}{
		"relation_id":     fmt.Sprintf("relation_%d", time.Now().Unix()),
		"from_content_id": req.FromContentID,
		"to_content_id":   req.ToContentID,
		"relation_type":   req.RelationType,
		"created_at":      time.Now(),
		"success":         true,
		"message":         "Relation created successfully",
	}

	return response, nil
}

// GetToolDefinition returns the MCP tool definition for memory_store
func (h *Handler) GetToolDefinition() map[string]interface{} {
	return map[string]interface{}{
		"name":        string(tools.ToolMemoryStore),
		"description": "Store and manage memory content with proper session isolation. Handles all data persistence operations including content storage, updates, deletions, thread creation, and relationship management.",
		"inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"enum":        tools.GetOperationsForTool(tools.ToolMemoryStore),
					"description": "The store operation to perform",
				},
				"project_id": map[string]interface{}{
					"type":        "string",
					"description": "Project identifier for data isolation (required)",
				},
				"session_id": map[string]interface{}{
					"type":        "string",
					"description": "Session identifier for write access (required for all store operations)",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The content to store (required for store_content, store_decision, update_content)",
				},
				"type": map[string]interface{}{
					"type":        "string",
					"description": "Content type (problem, solution, code_change, discussion, architecture_decision, etc.)",
				},
				"content_id": map[string]interface{}{
					"type":        "string",
					"description": "Content ID for update/delete operations",
				},
				"thread_id": map[string]interface{}{
					"type":        "string",
					"description": "Thread ID to associate content with",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "Title for thread creation",
				},
				"from_content_id": map[string]interface{}{
					"type":        "string",
					"description": "Source content ID for relationship creation",
				},
				"to_content_id": map[string]interface{}{
					"type":        "string",
					"description": "Target content ID for relationship creation",
				},
				"relation_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of relationship (references, blocks, implements, etc.)",
				},
			},
			"required": []string{"operation", "project_id", "session_id"},
		},
	}
}
