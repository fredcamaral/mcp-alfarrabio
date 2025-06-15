package storage

import (
	"context"
	"encoding/json"
	"errors"
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

// FileInsightStorage implements insight storage using file system
type FileInsightStorage struct {
	basePath string
	logger   *slog.Logger
	mutex    sync.RWMutex
}

// InsightFile represents the structure of an insights file
type InsightFile struct {
	Version   string                       `json:"version"`
	Insights  []*entities.CrossRepoInsight `json:"insights"`
	UpdatedAt time.Time                    `json:"updated_at"`
}

const (
	InsightFileVersion = "1.0.0"
	InsightsFileName   = "insights.json"
)

// NewFileInsightStorage creates a new file-based insight storage
func NewFileInsightStorage(logger *slog.Logger) (ports.InsightStorage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	basePath := filepath.Join(homeDir, ".lmmc", "insights")
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create insights directory: %w", err)
	}

	return &FileInsightStorage{
		basePath: basePath,
		logger:   logger,
	}, nil
}

// Create stores a new insight
func (s *FileInsightStorage) Create(ctx context.Context, insight *entities.CrossRepoInsight) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if insight.ID == "" {
		return errors.New("insight ID is required")
	}

	filePath := filepath.Join(s.basePath, InsightsFileName)
	insights, err := s.loadInsightsFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load insights: %w", err)
	}

	// Check if insight already exists
	for _, existing := range insights {
		if existing.ID == insight.ID {
			return fmt.Errorf("insight with ID %s already exists", insight.ID)
		}
	}

	insights = append(insights, insight)

	return s.saveInsightsWithLogging(filePath, insights, "created", insight.ID)
}

// Update updates an existing insight
func (s *FileInsightStorage) Update(ctx context.Context, insight *entities.CrossRepoInsight) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if insight.ID == "" {
		return errors.New("insight ID is required")
	}

	filePath := filepath.Join(s.basePath, InsightsFileName)
	insights, err := s.loadInsightsFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load insights: %w", err)
	}

	found := false
	for i, existing := range insights {
		if existing.ID == insight.ID {
			insights[i] = insight
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("insight with ID %s not found", insight.ID)
	}

	return s.saveInsightsWithLogging(filePath, insights, "updated", insight.ID)
}

// saveInsightsWithLogging saves insights to file with consistent error handling and logging
// This eliminates code duplication between Create and Update methods
func (s *FileInsightStorage) saveInsightsWithLogging(filePath string, insights []*entities.CrossRepoInsight, operation string, insightID string) error {
	if err := s.saveInsightsToFile(filePath, insights); err != nil {
		return fmt.Errorf("failed to save insights: %w", err)
	}

	s.logger.Debug("insight "+operation, slog.String("id", insightID))
	return nil
}

// Delete removes an insight
func (s *FileInsightStorage) Delete(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	filePath := filepath.Join(s.basePath, InsightsFileName)
	insights, err := s.loadInsightsFromFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to load insights: %w", err)
	}

	filtered := make([]*entities.CrossRepoInsight, 0, len(insights))
	found := false
	for _, insight := range insights {
		if insight.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, insight)
	}

	if !found {
		return fmt.Errorf("insight with ID %s not found", id)
	}

	if err := s.saveInsightsToFile(filePath, filtered); err != nil {
		return fmt.Errorf("failed to save insights: %w", err)
	}

	s.logger.Debug("insight deleted", slog.String("id", id))
	return nil
}

// GetByID retrieves an insight by ID
func (s *FileInsightStorage) GetByID(ctx context.Context, id string) (*entities.CrossRepoInsight, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	filePath := filepath.Join(s.basePath, InsightsFileName)
	insights, err := s.loadInsightsFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load insights: %w", err)
	}

	for _, insight := range insights {
		if insight.ID == id {
			return insight, nil
		}
	}

	return nil, fmt.Errorf("insight with ID %s not found", id)
}

// GetByProjectType retrieves insights for a project type
func (s *FileInsightStorage) GetByProjectType(ctx context.Context, projectType entities.ProjectType) ([]*entities.CrossRepoInsight, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	filePath := filepath.Join(s.basePath, InsightsFileName)
	insights, err := s.loadInsightsFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load insights: %w", err)
	}

	filtered := make([]*entities.CrossRepoInsight, 0)
	for _, insight := range insights {
		// Check if this insight applies to the project type
		for _, applicableType := range insight.Applicability {
			if applicableType == string(projectType) {
				filtered = append(filtered, insight)
				break
			}
		}
	}

	// Sort by quality score
	entities.SortInsightsByQuality(filtered)

	s.logger.Debug("insights retrieved by project type",
		slog.String("type", string(projectType)),
		slog.Int("count", len(filtered)))
	return filtered, nil
}

// Search searches insights by query
func (s *FileInsightStorage) Search(ctx context.Context, query string) ([]*entities.CrossRepoInsight, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	filePath := filepath.Join(s.basePath, InsightsFileName)
	insights, err := s.loadInsightsFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load insights: %w", err)
	}

	queryLower := strings.ToLower(query)
	filtered := make([]*entities.CrossRepoInsight, 0)

	for _, insight := range insights {
		// Search in title and description
		if strings.Contains(strings.ToLower(insight.Title), queryLower) ||
			strings.Contains(strings.ToLower(insight.Description), queryLower) {
			filtered = append(filtered, insight)
			continue
		}

		// Search in recommendations
		for _, rec := range insight.Recommendations {
			if strings.Contains(strings.ToLower(rec), queryLower) {
				filtered = append(filtered, insight)
				break
			}
		}

		// Search in tags
		for _, tag := range insight.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				filtered = append(filtered, insight)
				break
			}
		}
	}

	// Sort by relevance (those with query in title first)
	sort.Slice(filtered, func(i, j int) bool {
		iInTitle := strings.Contains(strings.ToLower(filtered[i].Title), queryLower)
		jInTitle := strings.Contains(strings.ToLower(filtered[j].Title), queryLower)

		if iInTitle != jInTitle {
			return iInTitle
		}

		// Otherwise sort by quality score
		return filtered[i].GetQualityScore() > filtered[j].GetQualityScore()
	})

	s.logger.Debug("insights searched",
		slog.String("query", query),
		slog.Int("results", len(filtered)))
	return filtered, nil
}

// GetShared retrieves shared insights
func (s *FileInsightStorage) GetShared(ctx context.Context) ([]*entities.CrossRepoInsight, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	filePath := filepath.Join(s.basePath, InsightsFileName)
	insights, err := s.loadInsightsFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load insights: %w", err)
	}

	// Filter for insights that are marked as shareable (high confidence, multiple sources)
	filtered := make([]*entities.CrossRepoInsight, 0)
	for _, insight := range insights {
		// Consider insights with high confidence and multiple sources as shareable
		if insight.Confidence >= 0.7 && insight.SourceCount >= 3 {
			filtered = append(filtered, insight)
		}
	}

	// Sort by quality score
	entities.SortInsightsByQuality(filtered)

	s.logger.Debug("shared insights retrieved", slog.Int("count", len(filtered)))
	return filtered, nil
}

// List retrieves all insights with pagination
func (s *FileInsightStorage) List(ctx context.Context, offset, limit int) ([]*entities.CrossRepoInsight, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	filePath := filepath.Join(s.basePath, InsightsFileName)
	insights, err := s.loadInsightsFromFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load insights: %w", err)
	}

	// Sort by generation date (newest first)
	sort.Slice(insights, func(i, j int) bool {
		return insights[i].GeneratedAt.After(insights[j].GeneratedAt)
	})

	// Apply pagination
	start := offset
	if start >= len(insights) {
		return []*entities.CrossRepoInsight{}, nil
	}

	end := start + limit
	if end > len(insights) {
		end = len(insights)
	}

	result := insights[start:end]

	s.logger.Debug("insights listed",
		slog.Int("offset", offset),
		slog.Int("limit", limit),
		slog.Int("returned", len(result)))
	return result, nil
}

// Helper methods

func (s *FileInsightStorage) loadInsightsFromFile(filePath string) ([]*entities.CrossRepoInsight, error) {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("path traversal detected: %s", filePath)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []*entities.CrossRepoInsight{}, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var insightFile InsightFile
	if err := json.Unmarshal(data, &insightFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal insights: %w", err)
	}

	return insightFile.Insights, nil
}

func (s *FileInsightStorage) saveInsightsToFile(filePath string, insights []*entities.CrossRepoInsight) error {
	insightFile := InsightFile{
		Version:   InsightFileVersion,
		Insights:  insights,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(insightFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal insights: %w", err)
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
