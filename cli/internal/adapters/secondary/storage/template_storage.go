package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// FileTemplateStorage implements template storage using file system
type FileTemplateStorage struct {
	basePath string
	logger   *slog.Logger
	mutex    sync.RWMutex
}

// TemplateFile represents the structure of a templates file
type TemplateFile struct {
	Version   string                   `json:"version"`
	Templates []*entities.TaskTemplate `json:"templates"`
	UpdatedAt time.Time                `json:"updated_at"`
}

const (
	TemplateFileVersion = "1.0.0"
	TemplatesFileName   = "templates.json"
	BuiltInFileName     = "built_in_templates.json"
)

// NewFileTemplateStorage creates a new file-based template storage
func NewFileTemplateStorage(logger *slog.Logger) (ports.TemplateStorage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	basePath := filepath.Join(homeDir, ".lmmc", "templates")
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create templates directory: %w", err)
	}

	storage := &FileTemplateStorage{
		basePath: basePath,
		logger:   logger,
	}

	// Initialize built-in templates if they don't exist
	if err := storage.initializeBuiltInTemplates(); err != nil {
		logger.Warn("failed to initialize built-in templates", slog.String("error", err.Error()))
	}

	return storage, nil
}

// Create stores a new template
func (s *FileTemplateStorage) Create(ctx context.Context, template *entities.TaskTemplate) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if template.ID == "" {
		return fmt.Errorf("template ID is required")
	}

	filePath := filepath.Join(s.basePath, TemplatesFileName)
	templates, _ := s.loadTemplatesFromFile(filePath)

	// Check if template already exists
	for _, existing := range templates {
		if existing.ID == template.ID {
			return fmt.Errorf("template with ID %s already exists", template.ID)
		}
	}

	// Validate template structure
	if errors := template.ValidateTemplate(); len(errors) > 0 {
		return fmt.Errorf("template validation failed: %v", errors)
	}

	templates = append(templates, template)

	if err := s.saveTemplatesToFile(filePath, templates); err != nil {
		return fmt.Errorf("failed to save templates: %w", err)
	}

	s.logger.Debug("template created",
		slog.String("id", template.ID),
		slog.String("name", template.Name))
	return nil
}

// Update updates an existing template
func (s *FileTemplateStorage) Update(ctx context.Context, template *entities.TaskTemplate) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if template.ID == "" {
		return fmt.Errorf("template ID is required")
	}

	// Validate template structure
	if errors := template.ValidateTemplate(); len(errors) > 0 {
		return fmt.Errorf("template validation failed: %v", errors)
	}

	filePath := filepath.Join(s.basePath, TemplatesFileName)
	templates, err := s.loadTemplatesFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	found := false
	for i, existing := range templates {
		if existing.ID == template.ID {
			// Don't allow updating built-in templates
			if existing.IsBuiltIn {
				return fmt.Errorf("cannot update built-in template")
			}
			template.UpdatedAt = time.Now()
			templates[i] = template
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("template with ID %s not found", template.ID)
	}

	if err := s.saveTemplatesToFile(filePath, templates); err != nil {
		return fmt.Errorf("failed to save templates: %w", err)
	}

	s.logger.Debug("template updated", slog.String("id", template.ID))
	return nil
}

// Delete removes a template
func (s *FileTemplateStorage) Delete(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	filePath := filepath.Join(s.basePath, TemplatesFileName)
	templates, err := s.loadTemplatesFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	filtered := make([]*entities.TaskTemplate, 0, len(templates))
	found := false

	for _, template := range templates {
		if template.ID == id {
			// Don't allow deleting built-in templates
			if template.IsBuiltIn {
				return fmt.Errorf("cannot delete built-in template")
			}
			found = true
			continue
		}
		filtered = append(filtered, template)
	}

	if !found {
		return fmt.Errorf("template with ID %s not found", id)
	}

	if err := s.saveTemplatesToFile(filePath, filtered); err != nil {
		return fmt.Errorf("failed to save templates: %w", err)
	}

	s.logger.Debug("template deleted", slog.String("id", id))
	return nil
}

// GetByID retrieves a template by ID
func (s *FileTemplateStorage) GetByID(ctx context.Context, id string) (*entities.TaskTemplate, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check user templates first
	filePath := filepath.Join(s.basePath, TemplatesFileName)
	templates, _ := s.loadTemplatesFromFile(filePath)

	for _, template := range templates {
		if template.ID == id {
			return template, nil
		}
	}

	// Check built-in templates
	builtInPath := filepath.Join(s.basePath, BuiltInFileName)
	builtInTemplates, _ := s.loadTemplatesFromFile(builtInPath)

	for _, template := range builtInTemplates {
		if template.ID == id {
			return template, nil
		}
	}

	return nil, fmt.Errorf("template with ID %s not found", id)
}

// GetByProjectType retrieves templates for a project type
func (s *FileTemplateStorage) GetByProjectType(ctx context.Context, projectType entities.ProjectType) ([]*entities.TaskTemplate, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	allTemplates := s.loadAllTemplates()
	filtered := make([]*entities.TaskTemplate, 0)

	for _, template := range allTemplates {
		if template.ProjectType == projectType {
			filtered = append(filtered, template)
		}
	}

	// Sort by success rate and usage count
	sort.Slice(filtered, func(i, j int) bool {
		scoreI := filtered[i].SuccessRate + (float64(filtered[i].UsageCount) * 0.01)
		scoreJ := filtered[j].SuccessRate + (float64(filtered[j].UsageCount) * 0.01)
		return scoreI > scoreJ
	})

	s.logger.Debug("templates retrieved by project type",
		slog.String("type", string(projectType)),
		slog.Int("count", len(filtered)))
	return filtered, nil
}

// List retrieves all templates with pagination
func (s *FileTemplateStorage) List(ctx context.Context, offset, limit int) ([]*entities.TaskTemplate, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	allTemplates := s.loadAllTemplates()

	// Sort by creation date (newest first)
	sort.Slice(allTemplates, func(i, j int) bool {
		return allTemplates[i].CreatedAt.After(allTemplates[j].CreatedAt)
	})

	// Apply pagination
	start := offset
	if start >= len(allTemplates) {
		return []*entities.TaskTemplate{}, nil
	}

	end := start + limit
	if end > len(allTemplates) {
		end = len(allTemplates)
	}

	result := allTemplates[start:end]

	s.logger.Debug("templates listed",
		slog.Int("offset", offset),
		slog.Int("limit", limit),
		slog.Int("returned", len(result)))
	return result, nil
}

// GetBuiltInTemplates retrieves built-in templates
func (s *FileTemplateStorage) GetBuiltInTemplates(ctx context.Context) ([]*entities.TaskTemplate, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	builtInPath := filepath.Join(s.basePath, BuiltInFileName)
	templates, err := s.loadTemplatesFromFile(builtInPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load built-in templates: %w", err)
	}

	// Sort by project type and then by name
	sort.Slice(templates, func(i, j int) bool {
		if templates[i].ProjectType != templates[j].ProjectType {
			return templates[i].ProjectType < templates[j].ProjectType
		}
		return templates[i].Name < templates[j].Name
	})

	s.logger.Debug("built-in templates retrieved", slog.Int("count", len(templates)))
	return templates, nil
}

// Search searches templates by query
func (s *FileTemplateStorage) Search(ctx context.Context, query string) ([]*entities.TaskTemplate, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	queryLower := strings.ToLower(query)
	allTemplates := s.loadAllTemplates()
	results := make([]*entities.TaskTemplate, 0)

	for _, template := range allTemplates {
		// Search in name and description
		if strings.Contains(strings.ToLower(template.Name), queryLower) ||
			strings.Contains(strings.ToLower(template.Description), queryLower) {
			results = append(results, template)
			continue
		}

		// Search in category and author
		if strings.Contains(strings.ToLower(template.Category), queryLower) ||
			strings.Contains(strings.ToLower(template.Author), queryLower) {
			results = append(results, template)
			continue
		}

		// Search in tags
		for _, tag := range template.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				results = append(results, template)
				break
			}
		}
	}

	// Sort by relevance (name matches first, then by success rate)
	sort.Slice(results, func(i, j int) bool {
		iInName := strings.Contains(strings.ToLower(results[i].Name), queryLower)
		jInName := strings.Contains(strings.ToLower(results[j].Name), queryLower)

		if iInName != jInName {
			return iInName
		}

		return results[i].SuccessRate > results[j].SuccessRate
	})

	s.logger.Debug("templates searched",
		slog.String("query", query),
		slog.Int("results", len(results)))
	return results, nil
}

// Helper methods

func (s *FileTemplateStorage) loadAllTemplates() []*entities.TaskTemplate {
	var allTemplates []*entities.TaskTemplate

	// Load user templates
	filePath := filepath.Join(s.basePath, TemplatesFileName)
	if userTemplates, err := s.loadTemplatesFromFile(filePath); err == nil {
		allTemplates = append(allTemplates, userTemplates...)
	}

	// Load built-in templates
	builtInPath := filepath.Join(s.basePath, BuiltInFileName)
	if builtInTemplates, err := s.loadTemplatesFromFile(builtInPath); err == nil {
		allTemplates = append(allTemplates, builtInTemplates...)
	}

	return allTemplates
}

func (s *FileTemplateStorage) loadTemplatesFromFile(filePath string) ([]*entities.TaskTemplate, error) {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("path traversal detected: %s", filePath)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []*entities.TaskTemplate{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var templateFile TemplateFile
	if err := json.Unmarshal(data, &templateFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal templates: %w", err)
	}

	return templateFile.Templates, nil
}

func (s *FileTemplateStorage) saveTemplatesToFile(filePath string, templates []*entities.TaskTemplate) error {
	templateFile := TemplateFile{
		Version:   TemplateFileVersion,
		Templates: templates,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(templateFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal templates: %w", err)
	}

	// Write atomically
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to move temp file: %w", err)
	}

	return nil
}

func (s *FileTemplateStorage) initializeBuiltInTemplates() error {
	builtInPath := filepath.Join(s.basePath, BuiltInFileName)

	// Only initialize if file doesn't exist
	if _, err := os.Stat(builtInPath); err == nil {
		return nil
	}

	// Create basic built-in templates
	builtInTemplates := s.createBuiltInTemplates()

	return s.saveTemplatesToFile(builtInPath, builtInTemplates)
}

func (s *FileTemplateStorage) createBuiltInTemplates() []*entities.TaskTemplate {
	now := time.Now()

	return []*entities.TaskTemplate{
		{
			ID:          "builtin-webapp-init",
			Name:        "Web Application Initialization",
			Description: "Basic setup tasks for a new web application",
			ProjectType: entities.ProjectTypeWebApp,
			Category:    "initialization",
			Version:     "1.0.0",
			Author:      "Lerian MCP Memory",
			Tasks: []entities.TemplateTask{
				{Order: 1, Content: "Set up project structure", Priority: "high", EstimatedHours: 1.0},
				{Order: 2, Content: "Initialize package.json and dependencies", Priority: "high", EstimatedHours: 0.5},
				{Order: 3, Content: "Set up development environment", Priority: "high", EstimatedHours: 1.0},
				{Order: 4, Content: "Create basic HTML/CSS structure", Priority: "medium", EstimatedHours: 2.0},
				{Order: 5, Content: "Set up build tools", Priority: "medium", EstimatedHours: 1.5},
			},
			Tags:        []string{"initialization", "web", "frontend"},
			IsBuiltIn:   true,
			IsPublic:    true,
			SuccessRate: 0.85,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "builtin-cli-init",
			Name:        "CLI Application Initialization",
			Description: "Basic setup tasks for a new CLI application",
			ProjectType: entities.ProjectTypeCLI,
			Category:    "initialization",
			Version:     "1.0.0",
			Author:      "Lerian MCP Memory",
			Tasks: []entities.TemplateTask{
				{Order: 1, Content: "Set up Go module", Priority: "high", EstimatedHours: 0.5},
				{Order: 2, Content: "Create main.go entry point", Priority: "high", EstimatedHours: 0.5},
				{Order: 3, Content: "Set up CLI framework (Cobra)", Priority: "high", EstimatedHours: 1.0},
				{Order: 4, Content: "Implement basic commands", Priority: "medium", EstimatedHours: 2.0},
				{Order: 5, Content: "Add configuration management", Priority: "medium", EstimatedHours: 1.5},
				{Order: 6, Content: "Set up testing", Priority: "medium", EstimatedHours: 1.0},
			},
			Tags:        []string{"initialization", "cli", "go"},
			IsBuiltIn:   true,
			IsPublic:    true,
			SuccessRate: 0.90,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "builtin-api-init",
			Name:        "API Service Initialization",
			Description: "Basic setup tasks for a new API service",
			ProjectType: entities.ProjectTypeAPI,
			Category:    "initialization",
			Version:     "1.0.0",
			Author:      "Lerian MCP Memory",
			Tasks: []entities.TemplateTask{
				{Order: 1, Content: "Set up project structure", Priority: "high", EstimatedHours: 1.0},
				{Order: 2, Content: "Initialize web framework", Priority: "high", EstimatedHours: 1.0},
				{Order: 3, Content: "Set up database connection", Priority: "high", EstimatedHours: 1.5},
				{Order: 4, Content: "Create basic API routes", Priority: "medium", EstimatedHours: 2.0},
				{Order: 5, Content: "Add middleware (auth, logging, CORS)", Priority: "medium", EstimatedHours: 2.0},
				{Order: 6, Content: "Set up API documentation", Priority: "medium", EstimatedHours: 1.0},
				{Order: 7, Content: "Add testing framework", Priority: "medium", EstimatedHours: 1.5},
			},
			Tags:        []string{"initialization", "api", "backend"},
			IsBuiltIn:   true,
			IsPublic:    true,
			SuccessRate: 0.88,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}
