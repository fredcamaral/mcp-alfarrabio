// Package storage provides file-based storage implementation
// for the lerian-mcp-memory CLI application.
package storage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// FileStorage implements the Storage interface using local JSON files
type FileStorage struct {
	basePath string
	mutex    sync.RWMutex
}

// TaskFile represents the structure of a tasks file
type TaskFile struct {
	Version    string           `json:"version"`
	Repository string           `json:"repository"`
	Tasks      []*entities.Task `json:"tasks"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

const (
	TaskFileVersion = "1.0.0"
	TasksFileName   = "tasks.json"
	BackupSuffix    = ".backup"
	TempSuffix      = ".tmp"
)

// NewFileStorage creates a new file storage instance
func NewFileStorage() (*FileStorage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	basePath := filepath.Join(homeDir, ".lmmc")
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create .lmmc directory: %w", err)
	}

	fs := &FileStorage{basePath: basePath}

	// Verify write permissions
	if err := fs.HealthCheck(context.Background()); err != nil {
		return nil, fmt.Errorf("storage health check failed: %w", err)
	}

	return fs, nil
}

// NewFileStorageWithPath creates a file storage instance with custom base path
func NewFileStorageWithPath(basePath string) (*FileStorage, error) {
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", basePath, err)
	}

	return &FileStorage{basePath: basePath}, nil
}

// SaveTask saves a task to the appropriate repository file
func (fs *FileStorage) SaveTask(ctx context.Context, task *entities.Task) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if err := task.Validate(); err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	repoPath := fs.getRepositoryPath(task.Repository)
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	filePath := filepath.Join(repoPath, TasksFileName)
	return fs.atomicSaveTask(filePath, task)
}

// GetTask retrieves a task by ID
func (fs *FileStorage) GetTask(ctx context.Context, id string) (*entities.Task, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	// Check all directories in base path for the task
	entries, err := os.ReadDir(fs.basePath)
	if err != nil {
		return nil, fmt.Errorf("task with id %s not found", id)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			repoTasks, loadErr := fs.loadTasksFromDirectoryName(entry.Name())
			if loadErr != nil {
				continue // Skip directories with load errors
			}

			for _, task := range repoTasks {
				if task.ID == id {
					return task, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("task with id %s not found", id)
}

// UpdateTask updates an existing task
func (fs *FileStorage) UpdateTask(ctx context.Context, task *entities.Task) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("invalid task: %w", err)
	}

	// Check if task exists (without holding lock)
	existingTask, err := fs.GetTask(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// If repository changed, remove from old location
	if existingTask.Repository != task.Repository {
		if err := fs.removeTaskFromRepository(existingTask.Repository, task.ID); err != nil {
			return fmt.Errorf("failed to remove task from old repository: %w", err)
		}
	}

	// Save to (possibly new) repository (using internal method to avoid lock)
	repoPath := fs.getRepositoryPath(task.Repository)
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	filePath := filepath.Join(repoPath, TasksFileName)
	return fs.atomicSaveTask(filePath, task)
}

// DeleteTask removes a task by ID
func (fs *FileStorage) DeleteTask(ctx context.Context, id string) error {
	// List repositories without holding lock
	repositories, err := fs.ListRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	for _, repo := range repositories {
		if err := fs.removeTaskFromRepository(repo, id); err == nil {
			return nil // Successfully removed
		}
	}

	return fmt.Errorf("task with id %s not found", id)
}

// ListTasks returns tasks from a repository with optional filtering
func (fs *FileStorage) ListTasks(ctx context.Context, repository string, filters *ports.TaskFilters) ([]*entities.Task, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	tasks, err := fs.loadTasksFromRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks from repository %s: %w", repository, err)
	}

	// Apply filters
	filtered := make([]*entities.Task, 0, len(tasks))
	for _, task := range tasks {
		if fs.matchesFilters(task, filters) {
			filtered = append(filtered, task)
		}
	}

	// Sort by creation date (newest first) by default
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	return filtered, nil
}

// GetTasksByRepository returns all tasks from a specific repository
func (fs *FileStorage) GetTasksByRepository(ctx context.Context, repository string) ([]*entities.Task, error) {
	return fs.ListTasks(ctx, repository, &ports.TaskFilters{})
}

// SearchTasks searches for tasks containing the query string
func (fs *FileStorage) SearchTasks(ctx context.Context, query string, filters *ports.TaskFilters) ([]*entities.Task, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	allTasks, err := fs.loadAllTasks(filters)
	if err != nil {
		return nil, err
	}

	filtered := fs.filterTasksByQuery(allTasks, query, filters)
	fs.sortTasksByRelevance(filtered, strings.ToLower(query))

	return filtered, nil
}

// loadAllTasks loads all tasks from all directories with optional repository filtering
func (fs *FileStorage) loadAllTasks(filters *ports.TaskFilters) ([]*entities.Task, error) {
	entries, err := os.ReadDir(fs.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base directory: %w", err)
	}

	var allTasks []*entities.Task
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		tasks, err := fs.loadTasksFromDirectoryName(entry.Name())
		if err != nil {
			continue // Skip directories with errors
		}

		for _, task := range tasks {
			if fs.shouldIncludeTaskForRepository(task, filters) {
				allTasks = append(allTasks, task)
			}
		}
	}

	return allTasks, nil
}

// shouldIncludeTaskForRepository checks if task should be included based on repository filter
func (fs *FileStorage) shouldIncludeTaskForRepository(task *entities.Task, filters *ports.TaskFilters) bool {
	return filters.Repository == "" || task.Repository == filters.Repository
}

// filterTasksByQuery filters tasks by search query and other criteria
func (fs *FileStorage) filterTasksByQuery(allTasks []*entities.Task, query string, filters *ports.TaskFilters) []*entities.Task {
	queryLower := strings.ToLower(query)
	filtered := make([]*entities.Task, 0)

	for _, task := range allTasks {
		if fs.taskMatchesQuery(task, query, queryLower) && fs.matchesFilters(task, filters) {
			filtered = append(filtered, task)
		}
	}

	return filtered
}

// taskMatchesQuery checks if task matches the search query
func (fs *FileStorage) taskMatchesQuery(task *entities.Task, query, queryLower string) bool {
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(task.Content), queryLower)
}

// sortTasksByRelevance sorts tasks by relevance (exact matches first, then by creation date)
func (fs *FileStorage) sortTasksByRelevance(tasks []*entities.Task, queryLower string) {
	sort.Slice(tasks, func(i, j int) bool {
		iExact := strings.Contains(strings.ToLower(tasks[i].Content), queryLower)
		jExact := strings.Contains(strings.ToLower(tasks[j].Content), queryLower)

		if iExact != jExact {
			return iExact
		}

		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})
}

// SaveTasks saves multiple tasks atomically
func (fs *FileStorage) SaveTasks(ctx context.Context, tasks []*entities.Task) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Group tasks by repository
	tasksByRepo := make(map[string][]*entities.Task)
	for _, task := range tasks {
		if err := task.Validate(); err != nil {
			return fmt.Errorf("invalid task %s: %w", task.ID, err)
		}
		tasksByRepo[task.Repository] = append(tasksByRepo[task.Repository], task)
	}

	// Save each repository group
	for repo, repoTasks := range tasksByRepo {
		repoPath := fs.getRepositoryPath(repo)
		if err := os.MkdirAll(repoPath, 0o750); err != nil {
			return fmt.Errorf("failed to create repository directory %s: %w", repo, err)
		}

		filePath := filepath.Join(repoPath, TasksFileName)
		if err := fs.atomicSaveTasks(filePath, repo, repoTasks); err != nil {
			return fmt.Errorf("failed to save tasks for repository %s: %w", repo, err)
		}
	}

	return nil
}

// DeleteTasks removes multiple tasks by ID
func (fs *FileStorage) DeleteTasks(ctx context.Context, ids []string) error {
	// List repositories without holding lock
	repositories, err := fs.ListRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	for _, id := range ids {
		found := false
		for _, repo := range repositories {
			if err := fs.removeTaskFromRepository(repo, id); err == nil {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("task with id %s not found", id)
		}
	}

	return nil
}

// ListRepositories returns all repositories with tasks
func (fs *FileStorage) ListRepositories(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(fs.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base directory: %w", err)
	}

	var repositories []string
	repoMap := make(map[string]bool) // To avoid duplicates

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		repoName := fs.extractRepositoryName(entry.Name())
		if repoName == "" {
			continue
		}
		
		if !repoMap[repoName] {
			repositories = append(repositories, repoName)
			repoMap[repoName] = true
		}
	}

	sort.Strings(repositories)
	return repositories, nil
}

// extractRepositoryName extracts repository name from directory entry
func (fs *FileStorage) extractRepositoryName(dirName string) string {
	// Check if directory contains tasks.json
	tasksFile := filepath.Join(fs.basePath, dirName, TasksFileName)
	if _, err := os.Stat(tasksFile); err != nil {
		return ""
	}
	
	// Load tasks to get actual repository name
	tasks, err := fs.loadTasksFromDirectoryName(dirName)
	if err != nil || len(tasks) == 0 {
		return ""
	}
	
	return tasks[0].Repository
}

// GetRepositoryStats returns statistics for a repository
func (fs *FileStorage) GetRepositoryStats(ctx context.Context, repository string) (ports.RepositoryStats, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	tasks, err := fs.loadTasksFromRepository(repository)
	if err != nil {
		return ports.RepositoryStats{}, fmt.Errorf("failed to load repository tasks: %w", err)
	}

	stats := ports.RepositoryStats{
		Repository: repository,
		TotalTasks: len(tasks),
	}

	tagSet := make(map[string]bool)
	var lastActivity time.Time

	for _, task := range tasks {
		// Count by status
		switch task.Status {
		case entities.StatusPending:
			stats.PendingTasks++
		case entities.StatusInProgress:
			stats.InProgressTasks++
		case entities.StatusCompleted:
			stats.CompletedTasks++
		case entities.StatusCancelled:
			stats.CancelledTasks++
		}

		// Collect unique tags
		for _, tag := range task.Tags {
			tagSet[tag] = true
		}

		// Find last activity
		if task.UpdatedAt.After(lastActivity) {
			lastActivity = task.UpdatedAt
		}
	}

	stats.TotalTags = len(tagSet)
	if !lastActivity.IsZero() {
		stats.LastActivity = lastActivity.Format(time.RFC3339)
	}

	return stats, nil
}

// HealthCheck verifies storage is accessible and writable
func (fs *FileStorage) HealthCheck(ctx context.Context) error {
	// Check base directory exists and is writable
	testFile := filepath.Join(fs.basePath, ".health_check")
	if err := os.WriteFile(testFile, []byte("test"), 0o600); err != nil {
		return fmt.Errorf("storage not writable: %w", err)
	}

	if err := os.Remove(testFile); err != nil {
		return fmt.Errorf("failed to clean up health check file: %w", err)
	}

	return nil
}

// Backup creates a backup of all task data
func (fs *FileStorage) Backup(ctx context.Context, backupPath string) error {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	if err := os.MkdirAll(filepath.Dir(backupPath), 0o750); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create tar-like structure in JSON
	backup := map[string]interface{}{
		"version":      TaskFileVersion,
		"created_at":   time.Now().Format(time.RFC3339),
		"repositories": make(map[string]TaskFile),
	}

	repositories, err := fs.ListRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to list repositories: %w", err)
	}

	for _, repo := range repositories {
		tasks, err := fs.loadTasksFromRepository(repo)
		if err != nil {
			return fmt.Errorf("failed to load tasks from repository %s: %w", repo, err)
		}

		backup["repositories"].(map[string]TaskFile)[repo] = TaskFile{
			Version:    TaskFileVersion,
			Repository: repo,
			Tasks:      tasks,
			UpdatedAt:  time.Now(),
		}
	}

	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// Restore restores task data from a backup
func (fs *FileStorage) Restore(ctx context.Context, backupPath string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// For backup files, we allow external paths but still validate for basic security
	if err := fs.validateExternalPath(backupPath); err != nil {
		return fmt.Errorf("invalid backup path: %w", err)
	}

	data, err := os.ReadFile(filepath.Clean(backupPath))
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	var backup map[string]interface{}
	if err := json.Unmarshal(data, &backup); err != nil {
		return fmt.Errorf("failed to unmarshal backup data: %w", err)
	}

	repositories, ok := backup["repositories"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid backup format: missing repositories")
	}

	for repoName, repoData := range repositories {
		repoBytes, err := json.Marshal(repoData)
		if err != nil {
			return fmt.Errorf("failed to marshal repository data for %s: %w", repoName, err)
		}

		var taskFile TaskFile
		if err := json.Unmarshal(repoBytes, &taskFile); err != nil {
			return fmt.Errorf("failed to unmarshal repository data for %s: %w", repoName, err)
		}

		// Save restored tasks
		repoPath := fs.getRepositoryPath(repoName)
		if err := os.MkdirAll(repoPath, 0o750); err != nil {
			return fmt.Errorf("failed to create repository directory %s: %w", repoName, err)
		}

		filePath := filepath.Join(repoPath, TasksFileName)
		if err := fs.writeTaskFile(filePath, &taskFile); err != nil {
			return fmt.Errorf("failed to restore repository %s: %w", repoName, err)
		}
	}

	return nil
}

// Helper methods

func (fs *FileStorage) getRepositoryPath(repository string) string {
	// Create a safe directory name from repository
	hash := sha256.Sum256([]byte(repository))
	safeName := fmt.Sprintf("%x", hash[:8]) // Use first 8 bytes of hash
	return filepath.Join(fs.basePath, safeName)
}

func (fs *FileStorage) atomicSaveTask(filePath string, task *entities.Task) error {
	// Load existing tasks
	existingTasks, _ := fs.loadTasksFromFile(filePath)

	// Update or add task
	found := false
	for i, existingTask := range existingTasks {
		if existingTask.ID == task.ID {
			existingTasks[i] = task
			found = true
			break
		}
	}
	if !found {
		existingTasks = append(existingTasks, task)
	}

	// Create task file structure
	taskFile := &TaskFile{
		Version:    TaskFileVersion,
		Repository: task.Repository,
		Tasks:      existingTasks,
		UpdatedAt:  time.Now(),
	}

	return fs.atomicWriteTaskFile(filePath, taskFile)
}

func (fs *FileStorage) atomicSaveTasks(filePath, repository string, tasks []*entities.Task) error {
	// Load existing tasks
	existingTasks, _ := fs.loadTasksFromFile(filePath)

	// Create map for efficient lookup
	existingMap := make(map[string]*entities.Task)
	for _, task := range existingTasks {
		existingMap[task.ID] = task
	}

	// Update or add new tasks
	for _, task := range tasks {
		existingMap[task.ID] = task
	}

	// Convert back to slice
	allTasks := make([]*entities.Task, 0, len(existingMap))
	for _, task := range existingMap {
		allTasks = append(allTasks, task)
	}

	// Sort by creation date
	sort.Slice(allTasks, func(i, j int) bool {
		return allTasks[i].CreatedAt.Before(allTasks[j].CreatedAt)
	})

	taskFile := &TaskFile{
		Version:    TaskFileVersion,
		Repository: repository,
		Tasks:      allTasks,
		UpdatedAt:  time.Now(),
	}

	return fs.atomicWriteTaskFile(filePath, taskFile)
}

func (fs *FileStorage) atomicWriteTaskFile(filePath string, taskFile *TaskFile) error {
	tempFile := filePath + TempSuffix
	backupFile := filePath + BackupSuffix

	// Write to temp file
	if err := fs.writeTaskFile(tempFile, taskFile); err != nil {
		return err
	}

	// Create backup if original exists
	if _, err := os.Stat(filePath); err == nil {
		if err := fs.copyFile(filePath, backupFile); err != nil {
			_ = os.Remove(tempFile)
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Atomic move
	if err := os.Rename(tempFile, filePath); err != nil {
		// Restore backup on failure
		if _, backupErr := os.Stat(backupFile); backupErr == nil {
			_ = os.Rename(backupFile, filePath)
		}
		return fmt.Errorf("failed to move temp file: %w", err)
	}

	// Remove backup on success
	_ = os.Remove(backupFile)
	return nil
}

func (fs *FileStorage) writeTaskFile(filePath string, taskFile *TaskFile) error {
	data, err := json.MarshalIndent(taskFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task file: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (fs *FileStorage) loadTasksFromRepository(repository string) ([]*entities.Task, error) {
	repoPath := fs.getRepositoryPath(repository)
	filePath := filepath.Join(repoPath, TasksFileName)
	return fs.loadTasksFromFile(filePath)
}

func (fs *FileStorage) loadTasksFromDirectoryName(dirName string) ([]*entities.Task, error) {
	filePath := filepath.Join(fs.basePath, dirName, TasksFileName)
	return fs.loadTasksFromFile(filePath)
}

func (fs *FileStorage) loadTasksFromFile(filePath string) ([]*entities.Task, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []*entities.Task{}, nil
	}

	// Validate file path for security
	if err := fs.validatePath(filePath); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var taskFile TaskFile
	if err := json.Unmarshal(data, &taskFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task file: %w", err)
	}

	return taskFile.Tasks, nil
}

func (fs *FileStorage) removeTaskFromRepository(repository, taskID string) error {
	tasks, err := fs.loadTasksFromRepository(repository)
	if err != nil {
		return err
	}

	filtered := make([]*entities.Task, 0, len(tasks))
	found := false
	for _, task := range tasks {
		if task.ID == taskID {
			found = true
			continue
		}
		filtered = append(filtered, task)
	}

	if !found {
		return fmt.Errorf("task not found in repository")
	}

	repoPath := fs.getRepositoryPath(repository)
	filePath := filepath.Join(repoPath, TasksFileName)

	if len(filtered) == 0 {
		// Remove empty file
		return os.Remove(filePath)
	}

	taskFile := &TaskFile{
		Version:    TaskFileVersion,
		Repository: repository,
		Tasks:      filtered,
		UpdatedAt:  time.Now(),
	}

	return fs.atomicWriteTaskFile(filePath, taskFile)
}

func (fs *FileStorage) matchesFilters(task *entities.Task, filters *ports.TaskFilters) bool {
	if filters == nil {
		return true
	}

	if filters.Status != nil && task.Status != *filters.Status {
		return false
	}

	if filters.Priority != nil && task.Priority != *filters.Priority {
		return false
	}

	if filters.ParentID != "" && task.ParentTaskID != filters.ParentID {
		return false
	}

	if filters.SessionID != "" && task.SessionID != filters.SessionID {
		return false
	}

	if len(filters.Tags) > 0 {
		taskTagSet := make(map[string]bool)
		for _, tag := range task.Tags {
			taskTagSet[tag] = true
		}

		for _, filterTag := range filters.Tags {
			if !taskTagSet[filterTag] {
				return false
			}
		}
	}

	return true
}

func (fs *FileStorage) copyFile(src, dst string) error {
	// Validate paths to prevent path traversal
	// Use internal validation for paths within the storage system
	if err := fs.validatePath(src); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}
	if err := fs.validatePath(dst); err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	sourceFile, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	destFile, err := os.Create(filepath.Clean(dst))
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// validatePath ensures the path is safe and within expected bounds
func (fs *FileStorage) validatePath(path string) error {
	cleanPath := filepath.Clean(path)
	
	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}
	
	// Ensure path is within base directory
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	
	absBasePath, err := filepath.Abs(fs.basePath)
	if err != nil {
		return fmt.Errorf("invalid base path: %w", err)
	}
	
	if !strings.HasPrefix(absPath, absBasePath) {
		return fmt.Errorf("path outside allowed directory")
	}
	
	return nil
}

// validateExternalPath performs basic security checks for external paths (like backups)
func (fs *FileStorage) validateExternalPath(path string) error {
	cleanPath := filepath.Clean(path)
	
	// Check for path traversal attempts using relative paths
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}
	
	// Additional validation: ensure it's not trying to access sensitive system files
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	
	// Block access to sensitive system directories
	prohibitedPaths := []string{"/etc", "/proc", "/sys", "/dev", "/boot"}
	for _, prohibited := range prohibitedPaths {
		if strings.HasPrefix(absPath, prohibited) {
			return fmt.Errorf("access to system directories not allowed")
		}
	}
	
	return nil
}
