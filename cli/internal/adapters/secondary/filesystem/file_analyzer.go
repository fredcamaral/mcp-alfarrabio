package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/services"
)

// fileAnalyzerImpl implements the FileAnalyzer interface
type fileAnalyzerImpl struct {
	config *FileAnalyzerConfig
	logger *slog.Logger
}

// FileAnalyzerConfig holds configuration for file analysis
type FileAnalyzerConfig struct {
	MaxDepth           int           `json:"max_depth"`
	MaxFiles           int           `json:"max_files"`
	MaxFileSize        int64         `json:"max_file_size"` // bytes
	IgnorePatterns     []string      `json:"ignore_patterns"`
	ConfigFilePatterns []string      `json:"config_file_patterns"`
	TimeoutDuration    time.Duration `json:"timeout_duration"`
}

// DefaultFileAnalyzerConfig returns default configuration
func DefaultFileAnalyzerConfig() *FileAnalyzerConfig {
	return &FileAnalyzerConfig{
		MaxDepth:        10,
		MaxFiles:        5000,
		MaxFileSize:     10 * 1024 * 1024, // 10MB
		TimeoutDuration: 30 * time.Second,
		IgnorePatterns: []string{
			"node_modules", ".git", ".vscode", ".idea", "target", "dist", "build",
			"vendor", "__pycache__", ".pytest_cache", "coverage", ".coverage",
			"*.log", "*.tmp", "*.cache", ".DS_Store", "Thumbs.db",
		},
		ConfigFilePatterns: []string{
			"package.json", "composer.json", "Gemfile", "requirements.txt", "Pipfile",
			"go.mod", "Cargo.toml", "pom.xml", "build.gradle", "CMakeLists.txt",
			"Dockerfile", "docker-compose.yml", "docker-compose.yaml",
			".github/workflows/*", ".gitlab-ci.yml", "Jenkinsfile",
			"tsconfig.json", "webpack.config.js", "next.config.js", "nuxt.config.js",
			"babel.config.js", "rollup.config.js", "vite.config.js",
			"Makefile", "pyproject.toml", "setup.py", "setup.cfg",
		},
	}
}

// NewFileAnalyzer creates a new file analyzer
func NewFileAnalyzer(config *FileAnalyzerConfig, logger *slog.Logger) services.FileAnalyzer {
	if config == nil {
		config = DefaultFileAnalyzerConfig()
	}

	return &fileAnalyzerImpl{
		config: config,
		logger: logger,
	}
}

// AnalyzeDirectory analyzes a directory structure
func (fa *fileAnalyzerImpl) AnalyzeDirectory(ctx context.Context, path string) (*services.DirectoryAnalysis, error) {
	fa.logger.Debug("analyzing directory", slog.String("path", path))

	start := time.Now()
	analysis := &services.DirectoryAnalysis{
		TotalFiles:     0,
		FilesByExt:     make(map[string]int),
		DirectoryDepth: 0,
		LargestFiles:   []string{},
		ConfigFiles:    []string{},
		Metadata:       make(map[string]interface{}),
	}

	// Use a timeout context
	ctx, cancel := context.WithTimeout(ctx, fa.config.TimeoutDuration)
	defer cancel()

	err := fa.walkDirectory(ctx, path, "", 0, analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze directory: %w", err)
	}

	// Calculate final metrics
	analysis.Metadata["analysis_duration"] = time.Since(start)
	analysis.Metadata["analyzed_at"] = time.Now()
	analysis.Metadata["max_depth_reached"] = analysis.DirectoryDepth

	fa.logger.Debug("directory analysis completed",
		slog.Int("total_files", analysis.TotalFiles),
		slog.Int("directory_depth", analysis.DirectoryDepth),
		slog.Duration("duration", time.Since(start)))

	return analysis, nil
}

// GetFilesByPattern finds files matching specific patterns
func (fa *fileAnalyzerImpl) GetFilesByPattern(ctx context.Context, path string, patterns []string) (map[string][]string, error) {
	result := make(map[string][]string)

	for _, pattern := range patterns {
		matches, err := fa.findFilesByPattern(ctx, path, pattern)
		if err != nil {
			fa.logger.Warn("failed to find files by pattern",
				slog.String("pattern", pattern),
				slog.Any("error", err))
			continue
		}
		if len(matches) > 0 {
			result[pattern] = matches
		}
	}

	return result, nil
}

// ReadConfigFiles reads and parses configuration files
func (fa *fileAnalyzerImpl) ReadConfigFiles(ctx context.Context, path string) (map[string]interface{}, error) {
	configFiles := make(map[string]interface{})

	// Find config files
	patterns := fa.config.ConfigFilePatterns
	filesByPattern, err := fa.GetFilesByPattern(ctx, path, patterns)
	if err != nil {
		return nil, err
	}

	// Read and parse each config file
	for _, files := range filesByPattern {
		for _, file := range files {
			if len(configFiles) >= 20 { // Limit number of config files
				break
			}

			content, err := fa.readConfigFile(file)
			if err != nil {
				fa.logger.Warn("failed to read config file",
					slog.String("file", file),
					slog.Any("error", err))
				continue
			}

			// Use the filename without path as key
			key := filepath.Base(file)
			configFiles[key] = content
		}
	}

	return configFiles, nil
}

// Helper methods

func (fa *fileAnalyzerImpl) walkDirectory(ctx context.Context, basePath, relativePath string, depth int, analysis *services.DirectoryAnalysis) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Check depth limit
	if depth > fa.config.MaxDepth {
		return nil
	}

	// Update max depth
	if depth > analysis.DirectoryDepth {
		analysis.DirectoryDepth = depth
	}

	currentPath := filepath.Join(basePath, relativePath)
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Check file limit
		if analysis.TotalFiles >= fa.config.MaxFiles {
			break
		}

		entryPath := filepath.Join(currentPath, entry.Name())
		relativeEntryPath := filepath.Join(relativePath, entry.Name())

		// Skip ignored patterns
		if fa.shouldIgnore(entry.Name()) {
			continue
		}

		if entry.IsDir() {
			// Recurse into subdirectory
			err := fa.walkDirectory(ctx, basePath, relativeEntryPath, depth+1, analysis)
			if err != nil {
				return err
			}
		} else {
			// Analyze file
			fa.analyzeFile(entryPath, relativeEntryPath, entry, analysis)
		}
	}

	return nil
}

func (fa *fileAnalyzerImpl) analyzeFile(_, relativePath string, entry fs.DirEntry, analysis *services.DirectoryAnalysis) {
	analysis.TotalFiles++

	// Get file extension
	ext := strings.ToLower(filepath.Ext(entry.Name()))
	if ext == "" {
		ext = "no_extension"
	}
	analysis.FilesByExt[ext]++

	// Check if it's a config file
	if fa.isConfigFile(entry.Name()) {
		analysis.ConfigFiles = append(analysis.ConfigFiles, relativePath)
	}

	// Get file info for size analysis
	info, err := entry.Info()
	if err == nil {
		// Track largest files (keep top 10)
		if info.Size() > 1024*1024 { // > 1MB
			analysis.LargestFiles = append(analysis.LargestFiles, fmt.Sprintf("%s (%s)", relativePath, fa.formatFileSize(info.Size())))

			// Keep only top 10 largest files
			if len(analysis.LargestFiles) > 10 {
				// Sort by size (extract from string) and keep top 10
				sort.Slice(analysis.LargestFiles, func(i, j int) bool {
					// Simple size comparison (could be improved)
					return len(analysis.LargestFiles[i]) > len(analysis.LargestFiles[j])
				})
				analysis.LargestFiles = analysis.LargestFiles[:10]
			}
		}
	}
}

func (fa *fileAnalyzerImpl) shouldIgnore(name string) bool {
	name = strings.ToLower(name)

	for _, pattern := range fa.config.IgnorePatterns {
		pattern = strings.ToLower(pattern)

		// Simple pattern matching (could be improved with glob)
		if strings.Contains(name, pattern) {
			return true
		}

		// Handle wildcard patterns
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(name, prefix) {
				return true
			}
		}

		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(name, suffix) {
				return true
			}
		}
	}

	return false
}

func (fa *fileAnalyzerImpl) isConfigFile(name string) bool {
	name = strings.ToLower(name)

	for _, pattern := range fa.config.ConfigFilePatterns {
		pattern = strings.ToLower(pattern)

		// Exact match
		if name == pattern {
			return true
		}

		// Pattern matching
		if strings.Contains(pattern, "*") {
			if strings.HasSuffix(pattern, "*") {
				prefix := strings.TrimSuffix(pattern, "*")
				if strings.HasPrefix(name, prefix) {
					return true
				}
			}
			if strings.HasPrefix(pattern, "*") {
				suffix := strings.TrimPrefix(pattern, "*")
				if strings.HasSuffix(name, suffix) {
					return true
				}
			}
		}

		// Check if it's in a specific directory pattern
		if strings.Contains(pattern, "/") && strings.Contains(name, strings.ToLower(filepath.Base(pattern))) {
			return true
		}
	}

	// Common config file patterns
	configIndicators := []string{"config", "conf", ".env", "settings", "rc", "yaml", "yml", "toml", "ini"}
	for _, indicator := range configIndicators {
		if strings.Contains(name, indicator) {
			return true
		}
	}

	return false
}

func (fa *fileAnalyzerImpl) findFilesByPattern(ctx context.Context, basePath, pattern string) ([]string, error) {
	var matches []string

	err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip ignored directories
		if d.IsDir() && fa.shouldIgnore(d.Name()) {
			return filepath.SkipDir
		}

		if !d.IsDir() {
			name := strings.ToLower(d.Name())
			pattern = strings.ToLower(pattern)

			// Simple pattern matching
			if name == pattern || strings.Contains(name, pattern) {
				relativePath, _ := filepath.Rel(basePath, path)
				matches = append(matches, relativePath)
			}
		}

		return nil
	})

	return matches, err
}

func (fa *fileAnalyzerImpl) readConfigFile(filePath string) (interface{}, error) {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return nil, fmt.Errorf("path traversal detected: %s", filePath)
	}

	// Check file size
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	if info.Size() > fa.config.MaxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes", info.Size())
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Try to parse as JSON first
	ext := strings.ToLower(filepath.Ext(filePath))
	filename := strings.ToLower(filepath.Base(filePath))

	switch {
	case ext == ".json" || strings.Contains(filename, "package.json"):
		var jsonContent interface{}
		if err := json.Unmarshal(content, &jsonContent); err == nil {
			return jsonContent, nil
		}
		fallthrough
	default:
		// Return as string for other file types
		return string(content), nil
	}
}

func (fa *fileAnalyzerImpl) formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
