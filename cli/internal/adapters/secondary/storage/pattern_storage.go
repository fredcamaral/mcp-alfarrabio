package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// FilePatternStorage implements pattern storage using file system
type FilePatternStorage struct {
	basePath string
	logger   *slog.Logger
	mutex    sync.RWMutex
}

// PatternFile represents the structure of a patterns file
type PatternFile struct {
	Version    string                  `json:"version"`
	Repository string                  `json:"repository"`
	Patterns   []*entities.TaskPattern `json:"patterns"`
	UpdatedAt  time.Time               `json:"updated_at"`
}

const (
	PatternFileVersion = "1.0.0"
	PatternsFileName   = "patterns.json"
)

// NewFilePatternStorage creates a new file-based pattern storage
func NewFilePatternStorage(logger *slog.Logger) (ports.PatternStorage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	basePath := filepath.Join(homeDir, ".lmmc", "patterns")
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create patterns directory: %w", err)
	}

	return &FilePatternStorage{
		basePath: basePath,
		logger:   logger,
	}, nil
}

// Create stores a new pattern
func (s *FilePatternStorage) Create(ctx context.Context, pattern *entities.TaskPattern) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := pattern.Validate(); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	repoPath := s.getRepositoryPath(pattern.Repository)
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	filePath := filepath.Join(repoPath, PatternsFileName)
	patterns, _ := s.loadPatternsFromFile(filePath)

	// Check if pattern already exists
	for _, existing := range patterns {
		if existing.ID == pattern.ID {
			return fmt.Errorf("pattern with ID %s already exists", pattern.ID)
		}
	}

	patterns = append(patterns, pattern)

	if err := s.savePatternsToFile(filePath, pattern.Repository, patterns); err != nil {
		return fmt.Errorf("failed to save patterns: %w", err)
	}

	s.logger.Debug("pattern created",
		slog.String("id", pattern.ID),
		slog.String("repository", pattern.Repository))
	return nil
}

// Update updates an existing pattern
func (s *FilePatternStorage) Update(ctx context.Context, pattern *entities.TaskPattern) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := pattern.Validate(); err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}

	repoPath := s.getRepositoryPath(pattern.Repository)
	filePath := filepath.Join(repoPath, PatternsFileName)
	patterns, err := s.loadPatternsFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load patterns: %w", err)
	}

	found := false
	for i, existing := range patterns {
		if existing.ID == pattern.ID {
			patterns[i] = pattern
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("pattern with ID %s not found", pattern.ID)
	}

	if err := s.savePatternsToFile(filePath, pattern.Repository, patterns); err != nil {
		return fmt.Errorf("failed to save patterns: %w", err)
	}

	s.logger.Debug("pattern updated", slog.String("id", pattern.ID))
	return nil
}

// Delete removes a pattern
func (s *FilePatternStorage) Delete(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check all repositories for the pattern
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return fmt.Errorf("failed to read patterns directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.basePath, entry.Name(), PatternsFileName)
		patterns, err := s.loadPatternsFromFile(filePath)
		if err != nil {
			continue
		}

		filtered := make([]*entities.TaskPattern, 0, len(patterns))
		found := false
		repository := ""

		for _, pattern := range patterns {
			if pattern.ID == id {
				found = true
				repository = pattern.Repository
				continue
			}
			filtered = append(filtered, pattern)
		}

		if found {
			if len(filtered) == 0 {
				// Remove empty file
				if err := os.Remove(filePath); err != nil {
					return fmt.Errorf("failed to remove empty patterns file: %w", err)
				}
			} else {
				if err := s.savePatternsToFile(filePath, repository, filtered); err != nil {
					return fmt.Errorf("failed to save patterns: %w", err)
				}
			}

			s.logger.Debug("pattern deleted", slog.String("id", id))
			return nil
		}
	}

	return fmt.Errorf("pattern with ID %s not found", id)
}

// GetByID retrieves a pattern by ID
func (s *FilePatternStorage) GetByID(ctx context.Context, id string) (*entities.TaskPattern, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check all repositories for the pattern
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("pattern with ID %s not found", id)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.basePath, entry.Name(), PatternsFileName)
		patterns, err := s.loadPatternsFromFile(filePath)
		if err != nil {
			continue
		}

		for _, pattern := range patterns {
			if pattern.ID == id {
				return pattern, nil
			}
		}
	}

	return nil, fmt.Errorf("pattern with ID %s not found", id)
}

// GetByRepository retrieves patterns for a repository
func (s *FilePatternStorage) GetByRepository(ctx context.Context, repository string) ([]*entities.TaskPattern, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	repoPath := s.getRepositoryPath(repository)
	filePath := filepath.Join(repoPath, PatternsFileName)
	patterns, err := s.loadPatternsFromFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load patterns: %w", err)
	}

	// Sort by confidence (highest first)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Confidence > patterns[j].Confidence
	})

	s.logger.Debug("patterns retrieved by repository",
		slog.String("repository", repository),
		slog.Int("count", len(patterns)))
	return patterns, nil
}

// GetByType retrieves patterns by type
func (s *FilePatternStorage) GetByType(ctx context.Context, patternType entities.PatternType) ([]*entities.TaskPattern, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	allPatterns := make([]*entities.TaskPattern, 0)

	// Load patterns from all repositories
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read patterns directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.basePath, entry.Name(), PatternsFileName)
		patterns, err := s.loadPatternsFromFile(filePath)
		if err != nil {
			continue
		}

		for _, pattern := range patterns {
			if pattern.Type == patternType {
				allPatterns = append(allPatterns, pattern)
			}
		}
	}

	// Sort by confidence
	sort.Slice(allPatterns, func(i, j int) bool {
		return allPatterns[i].Confidence > allPatterns[j].Confidence
	})

	s.logger.Debug("patterns retrieved by type",
		slog.String("type", string(patternType)),
		slog.Int("count", len(allPatterns)))
	return allPatterns, nil
}

// Search searches patterns by query
func (s *FilePatternStorage) Search(ctx context.Context, query string) ([]*entities.TaskPattern, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	queryLower := strings.ToLower(query)
	results := make([]*entities.TaskPattern, 0)

	// Search across all repositories
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read patterns directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.basePath, entry.Name(), PatternsFileName)
		patterns, err := s.loadPatternsFromFile(filePath)
		if err != nil {
			continue
		}

		for _, pattern := range patterns {
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

	s.logger.Debug("patterns searched",
		slog.String("query", query),
		slog.Int("results", len(results)))
	return results, nil
}

// GetByProjectType retrieves patterns by project type
func (s *FilePatternStorage) GetByProjectType(ctx context.Context, projectType entities.ProjectType) ([]*entities.TaskPattern, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	results := make([]*entities.TaskPattern, 0)

	// Load patterns from all repositories
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read patterns directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.basePath, entry.Name(), PatternsFileName)
		patterns, err := s.loadPatternsFromFile(filePath)
		if err != nil {
			continue
		}

		for _, pattern := range patterns {
			if pattern.ProjectType == string(projectType) {
				results = append(results, pattern)
			}
		}
	}

	// Sort by confidence
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	s.logger.Debug("patterns retrieved by project type",
		slog.String("type", string(projectType)),
		slog.Int("count", len(results)))
	return results, nil
}

// Helper methods

func (s *FilePatternStorage) getRepositoryPath(repository string) string {
	// Create a safe directory name from repository
	hash := sha256.Sum256([]byte(repository))
	safeName := hex.EncodeToString(hash[:8]) // Use first 8 bytes of hash
	return filepath.Join(s.basePath, safeName)
}

func (s *FilePatternStorage) loadPatternsFromFile(filePath string) ([]*entities.TaskPattern, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []*entities.TaskPattern{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var patternFile PatternFile
	if err := json.Unmarshal(data, &patternFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal patterns: %w", err)
	}

	return patternFile.Patterns, nil
}

func (s *FilePatternStorage) savePatternsToFile(filePath, repository string, patterns []*entities.TaskPattern) error {
	patternFile := PatternFile{
		Version:    PatternFileVersion,
		Repository: repository,
		Patterns:   patterns,
		UpdatedAt:  time.Now(),
	}

	data, err := json.MarshalIndent(patternFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal patterns: %w", err)
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
