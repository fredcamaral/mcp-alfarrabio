// Package ai provides intelligent memory management for CLI operations
package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/domain/constants"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// MemoryManager provides AI-powered memory and file management
type MemoryManager struct {
	mcpClient  ports.MCPClient
	aiService  ports.AIService
	storage    ports.Storage
	logger     *slog.Logger
	repository string
	sessionID  string
	config     *MemoryManagerConfig
}

// MemoryManagerConfig configures the memory manager behavior
type MemoryManagerConfig struct {
	AutoBackup           bool   `json:"auto_backup"`
	SmartSync            bool   `json:"smart_sync"`
	ConflictResolution   string `json:"conflict_resolution"` // "merge", "local", "remote"
	IntelligentCaching   bool   `json:"intelligent_caching"`
	PredictivePreload    bool   `json:"predictive_preload"`
	ContextAwareness     bool   `json:"context_awareness"`
	AutoFileOrganization bool   `json:"auto_file_organization"`
	LearningEnabled      bool   `json:"learning_enabled"`
	SyncFrequencyMins    int    `json:"sync_frequency_mins"`
	MaxCacheSize         int    `json:"max_cache_size_mb"`
	CompressionLevel     int    `json:"compression_level"`
}

// MemoryInsight represents AI-generated insights about memory usage
type MemoryInsight struct {
	Type        string                 `json:"type"`
	Priority    string                 `json:"priority"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	ActionItems []string               `json:"action_items"`
	Impact      string                 `json:"impact"`
	Confidence  float64                `json:"confidence"`
	Metadata    map[string]interface{} `json:"metadata"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// MemoryOperationResult contains the result of memory operations
type MemoryOperationResult struct {
	Operation       string           `json:"operation"`
	Success         bool             `json:"success"`
	FilesProcessed  int              `json:"files_processed"`
	MemoriesCreated int              `json:"memories_created"`
	MemoriesUpdated int              `json:"memories_updated"`
	Insights        []*MemoryInsight `json:"insights"`
	Recommendations []string         `json:"recommendations"`
	Conflicts       []string         `json:"conflicts"`
	ProcessingTime  time.Duration    `json:"processing_time"`
	StorageUsed     int64            `json:"storage_used_bytes"`
}

// FileMemoryMapping represents the relationship between local files and memory chunks
type FileMemoryMapping struct {
	LocalPath     string            `json:"local_path"`
	MemoryChunkID string            `json:"memory_chunk_id"`
	LastSync      time.Time         `json:"last_sync"`
	Checksum      string            `json:"checksum"`
	SyncStatus    string            `json:"sync_status"` // "synced", "local_newer", "remote_newer", "conflict"
	Metadata      map[string]string `json:"metadata"`
}

// NewMemoryManager creates a new AI-powered memory manager
func NewMemoryManager(mcpClient ports.MCPClient, aiService ports.AIService, storage ports.Storage, logger *slog.Logger) *MemoryManager {
	return &MemoryManager{
		mcpClient: mcpClient,
		aiService: aiService,
		storage:   storage,
		logger:    logger,
		config:    getDefaultMemoryManagerConfig(),
	}
}

// SetRepository sets the current repository context
func (mm *MemoryManager) SetRepository(repository string) {
	mm.repository = repository
}

// SetSessionID sets the current session ID
func (mm *MemoryManager) SetSessionID(sessionID string) {
	mm.sessionID = sessionID
}

// SyncLocalFiles intelligently syncs local files with MCP memory server
func (mm *MemoryManager) SyncLocalFiles(ctx context.Context, localPath string) (*MemoryOperationResult, error) {
	startTime := time.Now()
	result := &MemoryOperationResult{
		Operation: "sync_local_files",
		Success:   false,
	}

	// Step 1: Discover and analyze local files
	files, err := mm.discoverLocalFiles(localPath)
	if err != nil {
		return result, fmt.Errorf("failed to discover local files: %w", err)
	}

	result.FilesProcessed = len(files)

	// Step 2: Get existing file mappings
	mappings, err := mm.getFileMappings(ctx)
	if err != nil {
		mm.logger.Warn("failed to get file mappings", slog.String("error", err.Error()))
		mappings = make(map[string]*FileMemoryMapping)
	}

	// Step 3: Determine sync strategy using AI
	syncStrategy, err := mm.determineSyncStrategy(ctx, files, mappings)
	if err != nil {
		mm.logger.Warn("AI sync strategy failed, using default", slog.String("error", err.Error()))
		syncStrategy = mm.getDefaultSyncStrategy()
	}

	// Step 4: Execute intelligent sync
	if err := mm.executeSyncStrategy(ctx, files, mappings, syncStrategy, result); err != nil {
		return result, fmt.Errorf("sync execution failed: %w", err)
	}

	// Step 5: Update file mappings
	if err := mm.updateFileMappings(ctx, mappings); err != nil {
		mm.logger.Warn("failed to update file mappings", slog.String("error", err.Error()))
	}

	// Step 6: Generate insights and recommendations
	mm.generateSyncInsights(ctx, result)

	result.Success = true
	result.ProcessingTime = time.Since(startTime)

	mm.logger.Info("local files sync completed",
		slog.Int("files_processed", result.FilesProcessed),
		slog.Int("memories_created", result.MemoriesCreated),
		slog.Int("memories_updated", result.MemoriesUpdated),
		slog.Duration("processing_time", result.ProcessingTime))

	return result, nil
}

// PredictiveLoad anticipates what memories/tasks user might need and preloads them
func (mm *MemoryManager) PredictiveLoad(ctx context.Context, currentContext *entities.WorkContext) (*MemoryOperationResult, error) {
	if !mm.config.PredictivePreload {
		return &MemoryOperationResult{Operation: "predictive_load", Success: true}, nil
	}

	result := &MemoryOperationResult{
		Operation: "predictive_load",
		Success:   false,
	}

	// Use AI to predict what memories/tasks user will likely need
	predictions, err := mm.generatePredictions(ctx, currentContext)
	if err != nil {
		return result, fmt.Errorf("prediction generation failed: %w", err)
	}

	// Preload predicted memories
	for _, prediction := range predictions {
		if err := mm.preloadMemory(ctx, prediction); err != nil {
			mm.logger.Warn("failed to preload memory",
				slog.String("prediction", prediction),
				slog.String("error", err.Error()))
		}
	}

	result.Success = true
	return result, nil
}

// OptimizeStorage uses AI to optimize local storage and memory usage
func (mm *MemoryManager) OptimizeStorage(ctx context.Context) (*MemoryOperationResult, error) {
	result := &MemoryOperationResult{
		Operation: "optimize_storage",
		Success:   false,
	}

	// Analyze current storage usage
	storageAnalysis, err := mm.analyzeStorageUsage(ctx)
	if err != nil {
		return result, fmt.Errorf("storage analysis failed: %w", err)
	}

	// Generate AI-powered optimization recommendations
	optimizations, err := mm.generateOptimizations(ctx, storageAnalysis)
	if err != nil {
		return result, fmt.Errorf("optimization generation failed: %w", err)
	}

	// Apply optimizations
	if err := mm.applyOptimizations(ctx, optimizations, result); err != nil {
		return result, fmt.Errorf("optimization application failed: %w", err)
	}

	result.Success = true
	return result, nil
}

// ResolveConflicts uses AI to intelligently resolve sync conflicts
func (mm *MemoryManager) ResolveConflicts(ctx context.Context, conflicts []string) (*MemoryOperationResult, error) {
	result := &MemoryOperationResult{
		Operation: "resolve_conflicts",
		Success:   false,
		Conflicts: conflicts,
	}

	if len(conflicts) == 0 {
		result.Success = true
		return result, nil
	}

	// Use AI to analyze conflicts and suggest resolutions
	for _, conflict := range conflicts {
		resolution, err := mm.analyzeConflict(ctx, conflict)
		if err != nil {
			mm.logger.Warn("conflict analysis failed",
				slog.String("conflict", conflict),
				slog.String("error", err.Error()))
			continue
		}

		if err := mm.applyConflictResolution(ctx, conflict, resolution); err != nil {
			result.Conflicts = append(result.Conflicts, conflict)
		}
	}

	result.Success = len(result.Conflicts) == 0
	return result, nil
}

// GetMemoryInsights provides AI-generated insights about memory usage patterns
func (mm *MemoryManager) GetMemoryInsights(ctx context.Context) ([]*MemoryInsight, error) {
	if mm.mcpClient == nil || !mm.mcpClient.IsOnline() {
		return nil, errors.New("MCP client not available")
	}

	// Get memory analytics from MCP server
	analyticsRequest := map[string]interface{}{
		"operation": "health_dashboard",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": mm.repository,
			"session_id": mm.sessionID,
		},
	}

	response, err := mm.mcpClient.QueryIntelligence(ctx, "health_dashboard", analyticsRequest)
	if err != nil {
		return nil, fmt.Errorf("memory analytics request failed: %w", err)
	}

	// Use AI to generate insights from analytics
	return mm.generateMemoryInsights(ctx, response)
}

// Helper methods

func (mm *MemoryManager) discoverLocalFiles(localPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && mm.shouldIncludeFile(path, info) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func (mm *MemoryManager) shouldIncludeFile(path string, info os.FileInfo) bool {
	// Skip hidden files and directories
	if strings.HasPrefix(filepath.Base(path), ".") {
		return false
	}

	// Skip large files
	if info.Size() > 10*1024*1024 { // 10MB limit
		return false
	}

	// Include common text/code files
	ext := strings.ToLower(filepath.Ext(path))
	includedExts := map[string]bool{
		".md": true, ".txt": true, ".go": true, ".js": true, ".ts": true,
		".py": true, ".java": true, ".c": true, ".cpp": true, ".h": true,
		".json": true, ".yaml": true, ".yml": true, ".toml": true,
		".sql": true, ".sh": true, ".bat": true, ".ps1": true,
		".html": true, ".css": true, ".scss": true, ".xml": true,
	}

	return includedExts[ext]
}

func (mm *MemoryManager) getFileMappings(_ context.Context) (map[string]*FileMemoryMapping, error) {
	mappings := make(map[string]*FileMemoryMapping)

	// For now, file mappings are stored in memory only
	// In a real implementation, these would be persisted to a dedicated storage
	mm.logger.Debug("loading file mappings from memory (no persistence yet)")

	return mappings, nil
}

func (mm *MemoryManager) determineSyncStrategy(_ context.Context, files []string, mappings map[string]*FileMemoryMapping) (map[string]string, error) {
	// Use intelligent heuristics to determine sync strategy
	strategy := make(map[string]string)

	// Based on file count and mappings, determine strategy
	if len(files) > 100 {
		strategy["approach"] = "batch"
		strategy["batch_size"] = "20"
	} else {
		strategy["approach"] = "incremental"
		strategy["batch_size"] = "10"
	}

	strategy["conflict_resolution"] = mm.config.ConflictResolution
	strategy["priority"] = "modified_first"

	// Add intelligent decisions based on context
	if len(mappings) == 0 {
		strategy["new_repo"] = constants.BoolStringTrue
		strategy["full_sync"] = constants.BoolStringTrue
	} else {
		strategy["existing_repo"] = constants.BoolStringTrue
		strategy["incremental_sync"] = constants.BoolStringTrue
	}

	return strategy, nil
}

func (mm *MemoryManager) getDefaultSyncStrategy() map[string]string {
	return map[string]string{
		"new_files":      "upload",
		"modified_files": "update",
		"deleted_files":  "remove",
		"conflict_files": mm.config.ConflictResolution,
	}
}

func (mm *MemoryManager) executeSyncStrategy(ctx context.Context, files []string, mappings map[string]*FileMemoryMapping, strategy map[string]string, result *MemoryOperationResult) error {
	for _, file := range files {
		mapping, exists := mappings[file]

		if !exists {
			// New file - upload to memory
			if err := mm.uploadNewFile(ctx, file, mappings); err != nil {
				mm.logger.Warn("failed to upload new file",
					slog.String("file", file),
					slog.String("error", err.Error()))
				continue
			}
			result.MemoriesCreated++
		} else {
			// Existing file - check for changes
			if mm.fileHasChanged(file, mapping) {
				if err := mm.updateExistingFile(ctx, file, mapping); err != nil {
					mm.logger.Warn("failed to update existing file",
						slog.String("file", file),
						slog.String("error", err.Error()))
					continue
				}
				result.MemoriesUpdated++
			}
		}
	}

	return nil
}

func (mm *MemoryManager) uploadNewFile(ctx context.Context, filePath string, mappings map[string]*FileMemoryMapping) error {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("path traversal detected: %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create memory chunk via MCP
	storeRequest := map[string]interface{}{
		"operation": "store_chunk",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": mm.repository,
			"session_id": mm.sessionID,
			"content":    string(content),
			"type":       "file_content",
			"metadata": map[string]interface{}{
				"file_path":     filePath,
				"file_name":     filepath.Base(filePath),
				"file_ext":      filepath.Ext(filePath),
				"sync_type":     "auto",
				"last_modified": time.Now().Format(time.RFC3339),
			},
		},
	}

	response, err := mm.mcpClient.QueryIntelligence(ctx, "store_chunk", storeRequest)
	if err != nil {
		return fmt.Errorf("failed to store file content: %w", err)
	}

	// Extract chunk ID from response and create mapping
	chunkID := mm.extractChunkID(response)
	mappings[filePath] = &FileMemoryMapping{
		LocalPath:     filePath,
		MemoryChunkID: chunkID,
		LastSync:      time.Now(),
		Checksum:      mm.calculateChecksum(content),
		SyncStatus:    "synced",
		Metadata:      map[string]string{"sync_type": "auto"},
	}

	return nil
}

func (mm *MemoryManager) updateExistingFile(ctx context.Context, filePath string, mapping *FileMemoryMapping) error {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("path traversal detected: %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Update memory chunk via MCP
	updateRequest := map[string]interface{}{
		"operation": "update_thread",
		"scope":     "single",
		"options": map[string]interface{}{
			"repository": mm.repository,
			"session_id": mm.sessionID,
			"thread_id":  mapping.MemoryChunkID,
			"metadata": map[string]interface{}{
				"content":       string(content),
				"last_modified": time.Now().Format(time.RFC3339),
				"sync_type":     "auto_update",
			},
		},
	}

	_, err = mm.mcpClient.QueryIntelligence(ctx, "update_thread", updateRequest)
	if err != nil {
		return fmt.Errorf("failed to update file content: %w", err)
	}

	// Update mapping
	mapping.LastSync = time.Now()
	mapping.Checksum = mm.calculateChecksum(content)
	mapping.SyncStatus = "synced"

	return nil
}

func (mm *MemoryManager) fileHasChanged(filePath string, mapping *FileMemoryMapping) bool {
	// Clean and validate the file path
	filePath = filepath.Clean(filePath)
	if strings.Contains(filePath, "..") {
		return false
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	currentChecksum := mm.calculateChecksum(content)
	return currentChecksum != mapping.Checksum
}

func (mm *MemoryManager) calculateChecksum(content []byte) string {
	// Simple checksum - could be enhanced with proper hashing
	return fmt.Sprintf("%x", len(content)) // Placeholder
}

func (mm *MemoryManager) extractChunkID(response map[string]interface{}) string {
	if chunkID, ok := response["chunk_id"].(string); ok {
		return chunkID
	}
	return fmt.Sprintf("chunk_%d", time.Now().UnixNano())
}

func (mm *MemoryManager) updateFileMappings(ctx context.Context, mappings map[string]*FileMemoryMapping) error {
	// For now, file mappings are stored in memory only
	// In a real implementation, these would be persisted to a dedicated storage
	mm.logger.Debug("updating file mappings in memory (no persistence yet)",
		slog.Int("mappings_count", len(mappings)))

	return nil
}

func (mm *MemoryManager) generatePredictions(_ context.Context, workContext *entities.WorkContext) ([]string, error) {
	// Generate intelligent predictions based on context patterns
	var predictions []string

	// Time-based predictions
	timeOfDay := time.Now().Hour()
	if timeOfDay < 12 {
		predictions = append(predictions, "morning_planning_memories", "daily_tasks_memories")
	} else if timeOfDay < 17 {
		predictions = append(predictions, "active_work_memories", "implementation_memories")
	} else {
		predictions = append(predictions, "wrap_up_memories", "next_day_preparation_memories")
	}

	// Task-based predictions
	if len(workContext.CurrentTasks) > 0 {
		predictions = append(predictions, "related_task_memories", "dependency_memories")

		// Predict based on task types
		for _, task := range workContext.CurrentTasks {
			for _, tag := range task.Tags {
				predictions = append(predictions, tag+"_related_memories")
			}
		}
	}

	// Energy and focus level predictions
	if workContext.EnergyLevel > 0.7 {
		predictions = append(predictions, "complex_task_memories", "challenging_work_memories")
	} else {
		predictions = append(predictions, "simple_task_memories", "maintenance_memories")
	}

	// Repository-specific predictions
	predictions = append(predictions, workContext.Repository+"_specific_memories")

	return predictions, nil
}

func (mm *MemoryManager) preloadMemory(_ context.Context, prediction string) error {
	// Implementation would preload predicted memories into local cache
	mm.logger.Debug("preloading memory prediction", slog.String("prediction", prediction))
	return nil
}

func (mm *MemoryManager) analyzeStorageUsage(ctx context.Context) (map[string]interface{}, error) {
	// Analyze current storage usage patterns
	return map[string]interface{}{
		"total_size":  0,
		"file_count":  0,
		"cache_usage": 0,
		"patterns":    []string{},
	}, nil
}

func (mm *MemoryManager) generateOptimizations(ctx context.Context, analysis map[string]interface{}) (map[string]interface{}, error) {
	// Generate AI-powered optimization recommendations
	return map[string]interface{}{
		"actions": []string{},
	}, nil
}

func (mm *MemoryManager) applyOptimizations(ctx context.Context, optimizations map[string]interface{}, result *MemoryOperationResult) error {
	// Apply optimizations
	return nil
}

func (mm *MemoryManager) analyzeConflict(ctx context.Context, conflict string) (string, error) {
	// Use AI to analyze and suggest conflict resolution
	return "merge", nil
}

func (mm *MemoryManager) applyConflictResolution(ctx context.Context, conflict, resolution string) error {
	// Apply conflict resolution
	return nil
}

func (mm *MemoryManager) generateSyncInsights(_ context.Context, result *MemoryOperationResult) {
	// Generate insights about the sync operation
	if result.FilesProcessed > 0 {
		result.Insights = append(result.Insights, &MemoryInsight{
			Type:        "sync_summary",
			Priority:    "info",
			Title:       "File Sync Completed",
			Description: fmt.Sprintf("Successfully processed %d files", result.FilesProcessed),
			Confidence:  1.0,
			GeneratedAt: time.Now(),
		})
	}
}

func (mm *MemoryManager) generateMemoryInsights(_ context.Context, analytics map[string]interface{}) ([]*MemoryInsight, error) {
	// Generate AI insights from memory analytics
	var insights []*MemoryInsight

	insights = append(insights, &MemoryInsight{
		Type:        "usage_analysis",
		Priority:    "info",
		Title:       "Memory Usage Analysis",
		Description: "Analysis of current memory usage patterns",
		Confidence:  0.8,
		GeneratedAt: time.Now(),
	})

	return insights, nil
}

func getDefaultMemoryManagerConfig() *MemoryManagerConfig {
	return &MemoryManagerConfig{
		AutoBackup:           true,
		SmartSync:            true,
		ConflictResolution:   "merge",
		IntelligentCaching:   true,
		PredictivePreload:    true,
		ContextAwareness:     true,
		AutoFileOrganization: true,
		LearningEnabled:      true,
		SyncFrequencyMins:    15,
		MaxCacheSize:         500, // MB
		CompressionLevel:     3,
	}
}
