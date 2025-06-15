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
	"strings"
	"sync"
	"time"
)

// Entity defines the minimal interface that entities must implement
// to work with the generic file storage
type Entity interface {
	GetID() string
	GetRepository() string
	Validate() error
}

// FileWrapper represents the structure of a generic file
type FileWrapper[T Entity] struct {
	Version    string    `json:"version"`
	Repository string    `json:"repository"`
	Items      []T       `json:"items"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// FileStorageConfig contains configuration for file storage
type FileStorageConfig struct {
	BasePath    string
	SubDir      string
	FileName    string
	Version     string
	Logger      *slog.Logger
}

// GenericFileStorage provides common CRUD operations for file-based storage
type GenericFileStorage[T Entity] struct {
	config FileStorageConfig
	mutex  sync.RWMutex
}

// NewGenericFileStorage creates a new generic file storage instance
func NewGenericFileStorage[T Entity](config FileStorageConfig) (*GenericFileStorage[T], error) {
	if config.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Use provided base path or default to home/.lmmc
	basePath := config.BasePath
	if basePath == "" {
		basePath = filepath.Join(homeDir, ".lmmc")
	}

	// Add subdirectory
	if config.SubDir != "" {
		basePath = filepath.Join(basePath, config.SubDir)
	}

	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	finalConfig := config
	finalConfig.BasePath = basePath

	return &GenericFileStorage[T]{
		config: finalConfig,
	}, nil
}

// Create stores a new entity
func (s *GenericFileStorage[T]) Create(ctx context.Context, entity T) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := entity.Validate(); err != nil {
		return fmt.Errorf("invalid entity: %w", err)
	}

	repoPath := s.getRepositoryPath(entity.GetRepository())
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	filePath := filepath.Join(repoPath, s.config.FileName)
	items, _ := s.loadItemsFromFile(filePath)

	// Check if entity already exists
	for _, existing := range items {
		if existing.GetID() == entity.GetID() {
			return fmt.Errorf("entity with ID %s already exists", entity.GetID())
		}
	}

	items = append(items, entity)

	if err := s.saveItemsToFile(filePath, entity.GetRepository(), items); err != nil {
		return fmt.Errorf("failed to save items: %w", err)
	}

	s.config.Logger.Debug("entity created",
		slog.String("id", entity.GetID()),
		slog.String("repository", entity.GetRepository()))
	return nil
}

// Update updates an existing entity
func (s *GenericFileStorage[T]) Update(ctx context.Context, entity T) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if err := entity.Validate(); err != nil {
		return fmt.Errorf("invalid entity: %w", err)
	}

	repoPath := s.getRepositoryPath(entity.GetRepository())
	filePath := filepath.Join(repoPath, s.config.FileName)
	items, err := s.loadItemsFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load items: %w", err)
	}

	found := false
	for i, existing := range items {
		if existing.GetID() == entity.GetID() {
			items[i] = entity
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("entity with ID %s not found", entity.GetID())
	}

	if err := s.saveItemsToFile(filePath, entity.GetRepository(), items); err != nil {
		return fmt.Errorf("failed to save items: %w", err)
	}

	s.config.Logger.Debug("entity updated", slog.String("id", entity.GetID()))
	return nil
}

// Delete removes an entity
func (s *GenericFileStorage[T]) Delete(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check all repositories for the entity
	entries, err := os.ReadDir(s.config.BasePath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.config.BasePath, entry.Name(), s.config.FileName)
		items, err := s.loadItemsFromFile(filePath)
		if err != nil {
			continue
		}

		filtered := make([]T, 0, len(items))
		found := false
		repository := ""

		for _, item := range items {
			if item.GetID() == id {
				found = true
				repository = item.GetRepository()
				continue
			}
			filtered = append(filtered, item)
		}

		if found {
			if len(filtered) == 0 {
				// Remove empty file
				if err := os.Remove(filePath); err != nil {
					return fmt.Errorf("failed to remove empty file: %w", err)
				}
			} else {
				if err := s.saveItemsToFile(filePath, repository, filtered); err != nil {
					return fmt.Errorf("failed to save items: %w", err)
				}
			}

			s.config.Logger.Debug("entity deleted", slog.String("id", id))
			return nil
		}
	}

	return fmt.Errorf("entity with ID %s not found", id)
}

// GetByID retrieves an entity by ID
func (s *GenericFileStorage[T]) GetByID(ctx context.Context, id string) (T, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var zero T

	// Check all repositories for the entity
	entries, err := os.ReadDir(s.config.BasePath)
	if err != nil {
		return zero, fmt.Errorf("entity with ID %s not found", id)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.config.BasePath, entry.Name(), s.config.FileName)
		items, err := s.loadItemsFromFile(filePath)
		if err != nil {
			continue
		}

		for _, item := range items {
			if item.GetID() == id {
				return item, nil
			}
		}
	}

	return zero, fmt.Errorf("entity with ID %s not found", id)
}

// GetByRepository retrieves entities for a repository
func (s *GenericFileStorage[T]) GetByRepository(ctx context.Context, repository string) ([]T, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	repoPath := s.getRepositoryPath(repository)
	filePath := filepath.Join(repoPath, s.config.FileName)
	items, err := s.loadItemsFromFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load items: %w", err)
	}

	s.config.Logger.Debug("entities retrieved by repository",
		slog.String("repository", repository),
		slog.Int("count", len(items)))
	return items, nil
}

// GetAllFromAllRepositories retrieves all entities from all repositories
func (s *GenericFileStorage[T]) GetAllFromAllRepositories(ctx context.Context) ([]T, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	allItems := make([]T, 0)

	// Load items from all repositories
	entries, err := os.ReadDir(s.config.BasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		filePath := filepath.Join(s.config.BasePath, entry.Name(), s.config.FileName)
		items, err := s.loadItemsFromFile(filePath)
		if err != nil {
			continue
		}
		allItems = append(allItems, items...)
	}

	return allItems, nil
}

// Helper methods

func (s *GenericFileStorage[T]) getRepositoryPath(repository string) string {
	// Create a safe directory name from repository
	hash := sha256.Sum256([]byte(repository))
	safeName := hex.EncodeToString(hash[:8]) // Use first 8 bytes of hash
	return filepath.Join(s.config.BasePath, safeName)
}

func (s *GenericFileStorage[T]) loadItemsFromFile(filePath string) ([]T, error) {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("path traversal detected: %s", filePath)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []T{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var wrapper FileWrapper[T]
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal items: %w", err)
	}

	return wrapper.Items, nil
}

func (s *GenericFileStorage[T]) saveItemsToFile(filePath, repository string, items []T) error {
	wrapper := FileWrapper[T]{
		Version:    s.config.Version,
		Repository: repository,
		Items:      items,
		UpdatedAt:  time.Now(),
	}

	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
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