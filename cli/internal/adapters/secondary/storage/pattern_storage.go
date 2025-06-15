package storage

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// FilePatternStorage implements pattern storage using file system
type FilePatternStorage struct {
	generic *GenericFileStorage[*entities.TaskPattern]
}

const (
	PatternFileVersion = "1.0.0"
	PatternsFileName   = "patterns.json"
)

// NewFilePatternStorage creates a new file-based pattern storage
func NewFilePatternStorage(logger *slog.Logger) (ports.PatternStorage, error) {
	config := FileStorageConfig{
		SubDir:   "patterns",
		FileName: PatternsFileName,
		Version:  PatternFileVersion,
		Logger:   logger,
	}

	generic, err := NewGenericFileStorage[*entities.TaskPattern](config)
	if err != nil {
		return nil, fmt.Errorf("failed to create generic storage: %w", err)
	}

	return &FilePatternStorage{
		generic: generic,
	}, nil
}

// Create stores a new pattern
func (s *FilePatternStorage) Create(ctx context.Context, pattern *entities.TaskPattern) error {
	return s.generic.Create(ctx, pattern)
}

// Update updates an existing pattern
func (s *FilePatternStorage) Update(ctx context.Context, pattern *entities.TaskPattern) error {
	return s.generic.Update(ctx, pattern)
}

// Delete removes a pattern
func (s *FilePatternStorage) Delete(ctx context.Context, id string) error {
	return s.generic.Delete(ctx, id)
}

// GetByID retrieves a pattern by ID
func (s *FilePatternStorage) GetByID(ctx context.Context, id string) (*entities.TaskPattern, error) {
	return s.generic.GetByID(ctx, id)
}

// GetByRepository retrieves patterns for a repository
func (s *FilePatternStorage) GetByRepository(ctx context.Context, repository string) ([]*entities.TaskPattern, error) {
	patterns, err := s.generic.GetByRepository(ctx, repository)
	if err != nil {
		return nil, err
	}

	// Sort by confidence (highest first)
	s.sortPatternsByConfidence(patterns)

	return patterns, nil
}

// GetByType retrieves patterns by type
func (s *FilePatternStorage) GetByType(ctx context.Context, patternType entities.PatternType) ([]*entities.TaskPattern, error) {
	allPatterns, err := s.generic.GetAllFromAllRepositories(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by type
	filtered := make([]*entities.TaskPattern, 0)
	for _, pattern := range allPatterns {
		if pattern.Type == patternType {
			filtered = append(filtered, pattern)
		}
	}

	// Sort by confidence
	s.sortPatternsByConfidence(filtered)

	return filtered, nil
}

// Search searches patterns by query
func (s *FilePatternStorage) Search(ctx context.Context, query string) ([]*entities.TaskPattern, error) {
	allPatterns, err := s.generic.GetAllFromAllRepositories(ctx)
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(query)
	results := make([]*entities.TaskPattern, 0)

	for _, pattern := range allPatterns {
		// Search in name and description
		if strings.Contains(strings.ToLower(pattern.Name), queryLower) ||
			strings.Contains(strings.ToLower(pattern.Description), queryLower) {
			results = append(results, pattern)
			continue
		}

		// Search in keywords
		for _, keyword := range pattern.GetKeywords() {
			if strings.Contains(strings.ToLower(keyword), queryLower) {
				results = append(results, pattern)
				break
			}
		}
	}

	// Sort by relevance (name matches first, then by confidence)
	sort.Slice(results, func(i, j int) bool {
		iInName := strings.Contains(strings.ToLower(results[i].Name), queryLower)
		jInName := strings.Contains(strings.ToLower(results[j].Name), queryLower)

		if iInName != jInName {
			return iInName
		}

		return results[i].Confidence > results[j].Confidence
	})

	return results, nil
}

// GetByProjectType retrieves patterns by project type
func (s *FilePatternStorage) GetByProjectType(ctx context.Context, projectType entities.ProjectType) ([]*entities.TaskPattern, error) {
	allPatterns, err := s.generic.GetAllFromAllRepositories(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]*entities.TaskPattern, 0)
	for _, pattern := range allPatterns {
		if pattern.ProjectType == string(projectType) {
			results = append(results, pattern)
		}
	}

	// Sort by confidence
	s.sortPatternsByConfidence(results)

	return results, nil
}

// sortPatternsByConfidence is a helper function to sort patterns by confidence (highest first)
// This eliminates code duplication across GetByType, GetByRepository, and GetByProjectType
func (s *FilePatternStorage) sortPatternsByConfidence(patterns []*entities.TaskPattern) {
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Confidence > patterns[j].Confidence
	})
}
