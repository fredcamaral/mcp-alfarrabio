package storage

import (
	"context"
	"log/slog"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// FileTemplateStorage implements template storage using file system
type FileTemplateStorage struct {
	logger *slog.Logger
}

// NewFileTemplateStorage creates a new file-based template storage
func NewFileTemplateStorage(logger *slog.Logger) ports.TemplateStorage {
	return &FileTemplateStorage{
		logger: logger,
	}
}

// Create stores a new template
func (s *FileTemplateStorage) Create(ctx context.Context, template *entities.TaskTemplate) error {
	// TODO: Implement file-based template storage
	s.logger.Debug("template create called", slog.String("id", template.ID))
	return nil
}

// Update updates an existing template
func (s *FileTemplateStorage) Update(ctx context.Context, template *entities.TaskTemplate) error {
	// TODO: Implement file-based template storage
	s.logger.Debug("template update called", slog.String("id", template.ID))
	return nil
}

// Delete removes a template
func (s *FileTemplateStorage) Delete(ctx context.Context, id string) error {
	// TODO: Implement file-based template storage
	s.logger.Debug("template delete called", slog.String("id", id))
	return nil
}

// GetByID retrieves a template by ID
func (s *FileTemplateStorage) GetByID(ctx context.Context, id string) (*entities.TaskTemplate, error) {
	// TODO: Implement file-based template storage
	s.logger.Debug("template get by id called", slog.String("id", id))
	return nil, nil
}

// GetByProjectType retrieves templates for a project type
func (s *FileTemplateStorage) GetByProjectType(ctx context.Context, projectType entities.ProjectType) ([]*entities.TaskTemplate, error) {
	// TODO: Implement file-based template storage
	s.logger.Debug("template get by project type called", slog.String("type", string(projectType)))
	return []*entities.TaskTemplate{}, nil
}

// List retrieves all templates with pagination
func (s *FileTemplateStorage) List(ctx context.Context, offset, limit int) ([]*entities.TaskTemplate, error) {
	// TODO: Implement file-based template storage
	s.logger.Debug("template list called", slog.Int("offset", offset), slog.Int("limit", limit))
	return []*entities.TaskTemplate{}, nil
}

// GetBuiltInTemplates retrieves built-in templates
func (s *FileTemplateStorage) GetBuiltInTemplates(ctx context.Context) ([]*entities.TaskTemplate, error) {
	// TODO: Implement file-based template storage
	s.logger.Debug("get built-in templates called")
	return []*entities.TaskTemplate{}, nil
}

// Search searches templates by query
func (s *FileTemplateStorage) Search(ctx context.Context, query string) ([]*entities.TaskTemplate, error) {
	// TODO: Implement file-based template storage
	s.logger.Debug("template search called", slog.String("query", query))
	return []*entities.TaskTemplate{}, nil
}
