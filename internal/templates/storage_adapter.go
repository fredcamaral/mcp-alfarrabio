// Package templates provides storage adapter for template data persistence
package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"lerian-mcp-memory/internal/storage"
	itypes "lerian-mcp-memory/internal/types"
)

// Template represents a template stored in the system
type Template struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	ProjectType string                 `json:"project_type"`
	Complexity  string                 `json:"complexity"`
	Content     string                 `json:"content"`
	Variables   []TemplateVariable     `json:"variables,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Task represents a generated task from template
type Task struct {
	ID           string                 `json:"id"`
	TemplateID   string                 `json:"template_id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Type         string                 `json:"type"`
	Priority     string                 `json:"priority"`
	Complexity   string                 `json:"complexity"`
	Status       string                 `json:"status"`
	Tags         []string               `json:"tags,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// TaskGeneration represents a task generation session
type TaskGeneration struct {
	ID         string                 `json:"id"`
	TemplateID string                 `json:"template_id"`
	ProjectID  string                 `json:"project_id"`
	Tasks      []*Task                `json:"tasks"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// GetBuiltinTemplatesAsTemplates returns built-in templates converted to Template type
func GetBuiltinTemplatesAsTemplates() []*Template {
	builtins := GetBuiltinTemplates()
	templates := make([]*Template, len(builtins))

	for i := range builtins {
		templates[i] = &Template{
			ID:          builtins[i].ID,
			Name:        builtins[i].Name,
			Description: builtins[i].Description,
			Category:    builtins[i].Category,
			ProjectType: string(builtins[i].ProjectType),
			Complexity:  "medium", // Default complexity
			Content:     "Built-in template: " + builtins[i].Description,
			Variables:   builtins[i].Variables,
			Tags:        builtins[i].Tags,
			Metadata:    builtins[i].Metadata,
			CreatedAt:   builtins[i].CreatedAt,
			UpdatedAt:   builtins[i].CreatedAt,
		}
	}

	return templates
}

// CleanStorageAdapter implements TemplateStorage using the existing storage interfaces
// This is a clean implementation that avoids interface conflicts
type CleanStorageAdapter struct {
	vectorStore  storage.VectorStore
	contentStore storage.ContentStore
}

// NewCleanStorageAdapter creates a new clean storage adapter for templates
func NewCleanStorageAdapter(vectorStore storage.VectorStore, contentStore storage.ContentStore) *CleanStorageAdapter {
	return &CleanStorageAdapter{
		vectorStore:  vectorStore,
		contentStore: contentStore,
	}
}

// StoreTemplateUsage stores template usage statistics
func (sa *CleanStorageAdapter) StoreTemplateUsage(ctx context.Context, templateID, projectID string, success bool, metadata map[string]interface{}) error {
	// Create usage record
	usageRecord := map[string]interface{}{
		"template_id": templateID,
		"project_id":  projectID,
		"success":     success,
		"timestamp":   time.Now(),
		"metadata":    metadata,
	}

	// Serialize to JSON
	content, err := json.Marshal(usageRecord)
	if err != nil {
		return fmt.Errorf("failed to serialize usage record: %w", err)
	}

	// Store as content
	contentData := &itypes.Content{
		ID:        fmt.Sprintf("template_usage_%s_%s_%d", templateID, projectID, time.Now().Unix()),
		ProjectID: itypes.ProjectID(projectID),
		Type:      "template_usage",
		Content:   string(content),
		Metadata: map[string]interface{}{
			"template_id": templateID,
			"category":    "template_usage",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return sa.contentStore.Store(ctx, contentData)
}

// GetTemplateUsage retrieves usage statistics for a template
func (sa *CleanStorageAdapter) GetTemplateUsage(ctx context.Context, templateID string) (*TemplateUsageStats, error) {
	// For now, return basic statistics since we don't have a search interface
	// In a full implementation, this would query stored usage records
	return &TemplateUsageStats{
		TemplateID:      templateID,
		UsageCount:      0,
		SuccessCount:    0,
		FailureCount:    0,
		SuccessRate:     0.0,
		PopularityScore: 0.0,
		LastUsed:        time.Time{},
		AverageTime:     "0m",
	}, nil
}

// StoreTemplate stores a template definition
func (sa *CleanStorageAdapter) StoreTemplate(ctx context.Context, template *Template) error {
	// Serialize template to JSON
	templateData, err := json.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to serialize template: %w", err)
	}

	// Store as content
	contentData := &itypes.Content{
		ID:        "template_" + template.ID,
		ProjectID: itypes.ProjectID("global"), // Templates are global
		Type:      "template_definition",
		Content:   string(templateData),
		Metadata: map[string]interface{}{
			"template_id":   template.ID,
			"project_type":  template.ProjectType,
			"category":      template.Category,
			"complexity":    template.Complexity,
			"template_name": template.Name,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return sa.contentStore.Store(ctx, contentData)
}

// GetTemplate retrieves a template by ID
func (sa *CleanStorageAdapter) GetTemplate(ctx context.Context, templateID string) (*Template, error) {
	// Try to get template from content store
	content, err := sa.contentStore.Get(ctx, itypes.ProjectID("global"), "template_"+templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template %s: %w", templateID, err)
	}

	// Deserialize template
	var template Template
	if err := json.Unmarshal([]byte(content.Content), &template); err != nil {
		return nil, fmt.Errorf("failed to deserialize template %s: %w", templateID, err)
	}

	return &template, nil
}

// ListTemplates returns available templates with filtering
func (sa *CleanStorageAdapter) ListTemplates(ctx context.Context, filters map[string]interface{}) ([]*Template, error) {
	// For now, return built-in templates since we don't have a search interface
	// In a full implementation, this would query stored templates
	return GetBuiltinTemplatesAsTemplates(), nil
}

// DeleteTemplate removes a template
func (sa *CleanStorageAdapter) DeleteTemplate(ctx context.Context, templateID string) error {
	return sa.contentStore.Delete(ctx, itypes.ProjectID("global"), "template_"+templateID)
}

// StoreTaskGeneration stores task generation results
func (sa *CleanStorageAdapter) StoreTaskGeneration(ctx context.Context, templateID, projectID string, tasks []*Task, metadata map[string]interface{}) error {
	// Create task generation record
	generationRecord := map[string]interface{}{
		"template_id": templateID,
		"project_id":  projectID,
		"tasks":       tasks,
		"timestamp":   time.Now(),
		"metadata":    metadata,
	}

	// Serialize to JSON
	content, err := json.Marshal(generationRecord)
	if err != nil {
		return fmt.Errorf("failed to serialize task generation: %w", err)
	}

	// Store as content
	contentData := &itypes.Content{
		ID:        fmt.Sprintf("task_generation_%s_%s_%d", templateID, projectID, time.Now().Unix()),
		ProjectID: itypes.ProjectID(projectID),
		Type:      "task_generation",
		Content:   string(content),
		Metadata: map[string]interface{}{
			"template_id": templateID,
			"task_count":  len(tasks),
			"category":    "task_generation",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store the generation record
	if err := sa.contentStore.Store(ctx, contentData); err != nil {
		return fmt.Errorf("failed to store task generation record: %w", err)
	}

	// Store individual tasks
	for i, task := range tasks {
		taskContent, err := json.Marshal(task)
		if err != nil {
			continue // Skip failed serialization
		}

		taskContentData := &itypes.Content{
			ID:        fmt.Sprintf("generated_task_%s_%s_%d_%d", templateID, projectID, time.Now().Unix(), i),
			ProjectID: itypes.ProjectID(projectID),
			Type:      "generated_task",
			Content:   string(taskContent),
			Metadata: map[string]interface{}{
				"template_id": templateID,
				"task_title":  task.Title,
				"complexity":  task.Complexity,
				"category":    "generated_task",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Store individual task (ignore errors for individual tasks)
		_ = sa.contentStore.Store(ctx, taskContentData)
	}

	return nil
}

// GetTaskGenerations retrieves task generation history
func (sa *CleanStorageAdapter) GetTaskGenerations(ctx context.Context, projectID string, limit int) ([]*TaskGeneration, error) {
	// For now, return empty list since we don't have search capability
	// In a full implementation, this would query stored generation records
	return []*TaskGeneration{}, nil
}

// GetPopularTemplates returns most popular templates
func (sa *CleanStorageAdapter) GetPopularTemplates(ctx context.Context, limit int) ([]*Template, error) {
	// Return built-in templates as "popular" templates
	templates := GetBuiltinTemplatesAsTemplates()
	if len(templates) > limit {
		return templates[:limit], nil
	}
	return templates, nil
}

// GetTemplateRecommendations returns template recommendations
func (sa *CleanStorageAdapter) GetTemplateRecommendations(ctx context.Context, projectID string, contextMap map[string]interface{}) ([]*Template, error) {
	// Return built-in templates as recommendations
	return GetBuiltinTemplatesAsTemplates(), nil
}

// StoreGeneratedTasks stores the result of template instantiation (implements TemplateStorage interface)
func (sa *CleanStorageAdapter) StoreGeneratedTasks(ctx context.Context, result *TemplateInstantiationResult) error {
	// Serialize instantiation result to JSON
	resultData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to serialize instantiation result: %w", err)
	}

	// Store as content
	contentData := &itypes.Content{
		ID:        fmt.Sprintf("template_instantiation_%s_%s_%d", result.TemplateID, result.ProjectID, result.GeneratedAt.Unix()),
		ProjectID: itypes.ProjectID(result.ProjectID),
		SessionID: itypes.SessionID(result.SessionID),
		Type:      "template_instantiation",
		Content:   string(resultData),
		Metadata: map[string]interface{}{
			"template_id":    result.TemplateID,
			"template_name":  result.TemplateName,
			"task_count":     result.TaskCount,
			"estimated_time": result.EstimatedTime,
			"category":       "template_instantiation",
		},
		CreatedAt: result.GeneratedAt,
		UpdatedAt: result.GeneratedAt,
	}

	if err := sa.contentStore.Store(ctx, contentData); err != nil {
		return fmt.Errorf("failed to store instantiation result: %w", err)
	}

	// Store individual tasks as separate content items for better searchability
	for i := range result.Tasks {
		taskContent, err := json.Marshal(result.Tasks[i])
		if err != nil {
			continue // Skip this task if serialization fails
		}

		taskContentData := &itypes.Content{
			ID:        result.Tasks[i].ID,
			ProjectID: itypes.ProjectID(result.Tasks[i].ProjectID),
			SessionID: itypes.SessionID(result.Tasks[i].SessionID),
			Type:      "generated_task",
			Content:   string(taskContent),
			Metadata: map[string]interface{}{
				"template_id":      result.Tasks[i].TemplateID,
				"task_type":        result.Tasks[i].Type,
				"task_priority":    result.Tasks[i].Priority,
				"estimated_time":   result.Tasks[i].EstimatedTime,
				"dependency_count": len(result.Tasks[i].Dependencies),
				"category":         "generated_task",
			},
			CreatedAt: result.Tasks[i].CreatedAt,
			UpdatedAt: result.Tasks[i].CreatedAt,
		}

		// Store task content (ignore errors for individual tasks)
		_ = sa.contentStore.Store(ctx, taskContentData)
	}

	return nil
}

// GetGeneratedTasks retrieves generated tasks for a project and template (implements TemplateStorage interface)
func (sa *CleanStorageAdapter) GetGeneratedTasks(ctx context.Context, projectID, templateID string) ([]GeneratedTask, error) {
	// For now, return empty list since we don't have search capability in the content store
	// In a full implementation, this would query stored task records and deserialize them
	return []GeneratedTask{}, nil
}
