// Package services provides batch synchronization capabilities for the CLI
package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"lerian-mcp-memory-cli/internal/adapters/secondary/api"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// BatchSyncService handles batch synchronization between CLI and server
type BatchSyncService struct {
	mcpClient        ports.MCPClient
	localStorage     ports.Storage
	conflictResolver *ConflictResolver
	syncState        *api.SyncState
	logger           *slog.Logger
	mu               sync.Mutex
	clientID         string
	syncStateFile    string
}

// BatchSyncConfig contains configuration for batch sync operations
type BatchSyncConfig struct {
	SyncInterval     time.Duration `json:"sync_interval"`
	BatchSize        int           `json:"batch_size"`
	MaxRetries       int           `json:"max_retries"`
	ConflictStrategy string        `json:"conflict_strategy"`
	AutoSync         bool          `json:"auto_sync"`
	DeltaSync        bool          `json:"delta_sync"`
}

// SyncResult contains the results of a synchronization operation
type SyncResult struct {
	Success           bool               `json:"success"`
	SyncedTasks       int                `json:"synced_tasks"`
	ConflictsDetected int                `json:"conflicts_detected"`
	ConflictsResolved int                `json:"conflicts_resolved"`
	Errors            []string           `json:"errors"`
	Duration          time.Duration      `json:"duration"`
	Statistics        api.SyncStatistics `json:"statistics"`
	Timestamp         time.Time          `json:"timestamp"`
}

// GetMCPClient returns the MCP client used by the batch sync service
func (s *BatchSyncService) GetMCPClient() ports.MCPClient {
	return s.mcpClient
}

// NewBatchSyncService creates a new batch sync service
func NewBatchSyncService(
	mcpClient ports.MCPClient,
	localStorage ports.Storage,
	logger *slog.Logger,
) *BatchSyncService {
	clientID := generateClientID()

	service := &BatchSyncService{
		mcpClient:        mcpClient,
		localStorage:     localStorage,
		conflictResolver: NewConflictResolver(localStorage, mcpClient, logger),
		logger:           logger,
		clientID:         clientID,
		syncStateFile:    filepath.Join(os.Getenv("HOME"), ".lmmc", "sync_state.json"),
	}

	// Load existing sync state
	service.loadSyncState()

	return service
}

// generateClientID creates a unique client identifier
func generateClientID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("cli_%d", time.Now().Unix())
	}
	return "cli_" + hex.EncodeToString(bytes)[:16]
}

// loadSyncState loads sync state from persistent storage
func (s *BatchSyncService) loadSyncState() {
	// Ensure directory exists
	dir := filepath.Dir(s.syncStateFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		s.logger.Warn("failed to create sync state directory", slog.Any("error", err))
	}

	// Try to load existing state
	data, err := os.ReadFile(s.syncStateFile)
	if err != nil {
		// Create new state
		s.syncState = &api.SyncState{
			ClientID:     s.clientID,
			LastSyncTime: time.Now().Add(-24 * time.Hour), // Start with 24 hours ago
			SyncVersion:  1,
		}
		s.saveSyncState()
		return
	}

	// Parse existing state
	var state api.SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		s.logger.Warn("failed to parse sync state, creating new", slog.Any("error", err))
		s.syncState = &api.SyncState{
			ClientID:     s.clientID,
			LastSyncTime: time.Now().Add(-24 * time.Hour),
			SyncVersion:  1,
		}
		s.saveSyncState()
		return
	}

	s.syncState = &state
	s.logger.Info("loaded sync state",
		slog.String("repository", state.Repository),
		slog.Time("last_sync", state.LastSyncTime),
		slog.Int("total_syncs", state.TotalSyncs))
}

// saveSyncState persists sync state to disk
func (s *BatchSyncService) saveSyncState() {
	data, err := json.MarshalIndent(s.syncState, "", "  ")
	if err != nil {
		s.logger.Error("failed to marshal sync state", slog.Any("error", err))
		return
	}

	if err := os.WriteFile(s.syncStateFile, data, 0600); err != nil {
		s.logger.Error("failed to save sync state", slog.Any("error", err))
	}
}

// PerformSync executes a full batch synchronization with the server
func (s *BatchSyncService) PerformSync(ctx context.Context, repository string) (*SyncResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	startTime := time.Now()
	result := &SyncResult{
		Timestamp: startTime,
		Errors:    make([]string, 0),
	}

	s.logger.Info("starting batch synchronization",
		slog.String("repository", repository),
		slog.String("client_id", s.clientID))

	// Check if server is available
	if s.mcpClient == nil || !s.mcpClient.IsOnline() {
		result.Errors = append(result.Errors, "server not available")
		return result, errors.New("server not available for sync")
	}

	// Get local tasks modified since last sync
	localTasks, err := s.getModifiedLocalTasks(ctx, repository)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to get local tasks: %v", err))
		return result, fmt.Errorf("failed to get local tasks: %w", err)
	}

	// Build sync request
	syncRequest := s.buildSyncRequest(repository, localTasks)

	// Perform server sync
	response, err := s.performServerSync(ctx, syncRequest)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("server sync failed: %v", err))
		return result, fmt.Errorf("server sync failed: %w", err)
	}

	// Process sync response
	syncResult := s.processSyncResponse(ctx, response)

	// Update sync state
	s.updateSyncState(repository, response)

	// Populate result
	result.Success = true
	result.SyncedTasks = len(response.ServerTasks)
	result.ConflictsDetected = len(response.Conflicts)
	result.ConflictsResolved = syncResult.ConflictsResolved
	result.Duration = time.Since(startTime)
	result.Statistics = response.SyncStats

	s.logger.Info("batch synchronization completed",
		slog.String("repository", repository),
		slog.Int("synced_tasks", result.SyncedTasks),
		slog.Int("conflicts", result.ConflictsDetected),
		slog.Duration("duration", result.Duration))

	return result, nil
}

// getModifiedLocalTasks retrieves local tasks that have been modified since last sync
func (s *BatchSyncService) getModifiedLocalTasks(ctx context.Context, repository string) ([]*entities.Task, error) {
	// Get all tasks for the repository
	allTasks, err := s.localStorage.GetTasksByRepository(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository tasks: %w", err)
	}

	// Filter tasks modified since last sync
	var modifiedTasks []*entities.Task
	for _, task := range allTasks {
		if task.UpdatedAt.After(s.syncState.LastSyncTime) {
			modifiedTasks = append(modifiedTasks, task)
		}
	}

	s.logger.Debug("found modified local tasks",
		slog.Int("total_tasks", len(allTasks)),
		slog.Int("modified_tasks", len(modifiedTasks)),
		slog.Time("since", s.syncState.LastSyncTime))

	return modifiedTasks, nil
}

// buildSyncRequest creates a batch sync request from local tasks
func (s *BatchSyncService) buildSyncRequest(repository string, localTasks []*entities.Task) *api.BatchSyncRequest {
	syncItems := make([]api.TaskSyncItem, 0, len(localTasks))

	for _, task := range localTasks {
		item := api.FromTask(task)
		syncItems = append(syncItems, item)
	}

	return &api.BatchSyncRequest{
		LastSyncTime: &s.syncState.LastSyncTime,
		LocalTasks:   syncItems,
		Repository:   repository,
		ClientID:     s.clientID,
		SyncToken:    s.syncState.SyncToken,
	}
}

// performServerSync sends the sync request to the server
func (s *BatchSyncService) performServerSync(ctx context.Context, request *api.BatchSyncRequest) (*api.BatchSyncResponse, error) {
	// This would normally call the MCP server's batch sync endpoint
	// For now, we'll simulate the response based on the existing MCP client

	// Get server tasks
	serverTasks, err := s.mcpClient.GetTasks(ctx, request.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get server tasks: %w", err)
	}

	// Build response
	response := &api.BatchSyncResponse{
		ServerTasks: make([]api.TaskSyncItem, 0, len(serverTasks)),
		Conflicts:   make([]api.ConflictItem, 0),
		ToCreate:    make([]string, 0),
		ToUpdate:    make([]string, 0),
		ToDelete:    make([]string, 0),
		ServerTime:  time.Now(),
		SyncToken:   s.generateSyncToken(),
		SyncStats:   api.SyncStatistics{},
	}

	// Convert server tasks
	serverTaskMap := make(map[string]*entities.Task)
	for _, task := range serverTasks {
		serverTaskMap[task.ID] = task
		response.ServerTasks = append(response.ServerTasks, api.FromTask(task))
	}

	// Create local task map
	localTaskMap := make(map[string]api.TaskSyncItem)
	for i := range request.LocalTasks {
		task := &request.LocalTasks[i]
		localTaskMap[task.ID] = *task
	}

	// Detect conflicts and operations
	for i := range request.LocalTasks {
		localTask := &request.LocalTasks[i]
		serverTask, exists := serverTaskMap[localTask.ID]

		if !exists {
			// Local task doesn't exist on server
			response.ToCreate = append(response.ToCreate, localTask.ID)
		} else {
			// Check for conflicts
			serverItem := api.FromTask(serverTask)
			if localTask.HasConflictWith(&serverItem) {
				conflict := s.conflictResolver.DetectConflict(ctx, localTask, &serverItem)
				if conflict != nil {
					response.Conflicts = append(response.Conflicts, *conflict)
				}
			} else if localTask.IsNewer(&serverItem) {
				// Local is newer, update server
				response.ToUpdate = append(response.ToUpdate, localTask.ID)
			}
		}
	}

	// Check for server tasks not in local
	for _, serverTask := range serverTasks {
		if _, exists := localTaskMap[serverTask.ID]; !exists {
			response.ToDelete = append(response.ToDelete, serverTask.ID)
		}
	}

	// Update statistics
	response.SyncStats.TotalTasks = len(request.LocalTasks) + len(serverTasks)
	response.SyncStats.ConflictsDetected = len(response.Conflicts)
	response.SyncStats.TasksCreated = len(response.ToCreate)
	response.SyncStats.TasksUpdated = len(response.ToUpdate)
	response.SyncStats.TasksDeleted = len(response.ToDelete)

	return response, nil
}

// processSyncResponse applies the sync response to local storage
func (s *BatchSyncService) processSyncResponse(ctx context.Context, response *api.BatchSyncResponse) *SyncResult {
	result := &SyncResult{
		Timestamp: time.Now(),
		Errors:    make([]string, 0),
	}

	// Handle conflicts first
	conflictsResolved := 0
	for _, conflict := range response.Conflicts {
		if conflict.Resolution.AutoApply && conflict.Resolution.ResolvedTask != nil {
			task := conflict.Resolution.ResolvedTask.ToTask()
			if err := s.localStorage.UpdateTask(ctx, task); err != nil {
				result.Errors = append(result.Errors,
					fmt.Sprintf("failed to apply conflict resolution for task %s: %v", task.ID, err))
				s.logger.Error("failed to apply conflict resolution",
					slog.String("task_id", task.ID),
					slog.Any("error", err))
			} else {
				conflictsResolved++
				s.logger.Info("conflict resolved automatically",
					slog.String("task_id", task.ID),
					slog.String("strategy", string(conflict.Resolution.Strategy)))
			}
		} else {
			s.logger.Warn("conflict requires manual resolution",
				slog.String("task_id", conflict.TaskID),
				slog.String("reason", conflict.Reason))
		}
	}

	// Apply server changes for tasks that don't have conflicts
	conflictTaskIDs := make(map[string]bool)
	for i := range response.Conflicts {
		conflict := &response.Conflicts[i]
		conflictTaskIDs[conflict.TaskID] = true
	}

	for _, serverTask := range response.ServerTasks {
		if conflictTaskIDs[serverTask.ID] {
			continue // Skip conflicted tasks
		}

		task := serverTask.ToTask()

		// Check if we need to create or update
		existing, err := s.localStorage.GetTask(ctx, task.ID)
		if err != nil {
			// Task doesn't exist locally, create it
			if err := s.localStorage.SaveTask(ctx, task); err != nil {
				result.Errors = append(result.Errors,
					fmt.Sprintf("failed to create task %s from server: %v", task.ID, err))
				s.logger.Error("failed to create task from server",
					slog.String("task_id", task.ID),
					slog.Any("error", err))
			}
		} else if task.UpdatedAt.After(existing.UpdatedAt) {
			// Server version is newer, update local
			if err := s.localStorage.UpdateTask(ctx, task); err != nil {
				result.Errors = append(result.Errors,
					fmt.Sprintf("failed to update task %s from server: %v", task.ID, err))
				s.logger.Error("failed to update task from server",
					slog.String("task_id", task.ID),
					slog.Any("error", err))
			}
		}
	}

	// Handle deletions (tasks that exist locally but not on server)
	for _, taskID := range response.ToDelete {
		if conflictTaskIDs[taskID] {
			continue // Don't delete conflicted tasks
		}

		if err := s.localStorage.DeleteTask(ctx, taskID); err != nil {
			result.Errors = append(result.Errors,
				fmt.Sprintf("failed to delete local task %s: %v", taskID, err))
			s.logger.Error("failed to delete local task",
				slog.String("task_id", taskID),
				slog.Any("error", err))
		}
	}

	result.ConflictsResolved = conflictsResolved
	result.Success = len(result.Errors) == 0

	return result
}

// updateSyncState updates the sync state after successful sync
func (s *BatchSyncService) updateSyncState(repository string, response *api.BatchSyncResponse) {
	s.syncState.Repository = repository
	s.syncState.LastSyncTime = response.ServerTime
	s.syncState.SyncToken = response.SyncToken
	s.syncState.LastConflictCount = len(response.Conflicts)
	s.syncState.TotalSyncs++
	s.syncState.SyncVersion++

	s.saveSyncState()

	s.logger.Debug("sync state updated",
		slog.String("repository", repository),
		slog.Time("last_sync", s.syncState.LastSyncTime),
		slog.String("sync_token", s.syncState.SyncToken))
}

// generateSyncToken creates a unique token for this sync operation
func (s *BatchSyncService) generateSyncToken() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("sync_%d", time.Now().Unix())
	}
	return "sync_" + hex.EncodeToString(bytes)
}

// PerformDeltaSync performs an optimized delta-only sync
func (s *BatchSyncService) PerformDeltaSync(ctx context.Context, repository string) (*SyncResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("starting delta synchronization",
		slog.String("repository", repository),
		slog.Time("since", s.syncState.LastSyncTime))

	// This would call a delta sync endpoint on the server
	// For now, fall back to full sync
	return s.PerformSync(ctx, repository)
}

// GetSyncStatus returns the current synchronization status
func (s *BatchSyncService) GetSyncStatus() *api.SyncState {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a copy to avoid concurrent access issues
	status := *s.syncState
	return &status
}

// ClearSyncState resets the sync state (useful for testing or re-initialization)
func (s *BatchSyncService) ClearSyncState() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.syncState = &api.SyncState{
		ClientID:     s.clientID,
		LastSyncTime: time.Now().Add(-24 * time.Hour),
		SyncVersion:  1,
	}

	s.saveSyncState()
	s.logger.Info("sync state cleared")
}

// ForceSync performs a full sync regardless of timestamps
func (s *BatchSyncService) ForceSync(ctx context.Context, repository string) (*SyncResult, error) {
	s.mu.Lock()
	// Temporarily reset last sync time to force full sync
	originalTime := s.syncState.LastSyncTime
	s.syncState.LastSyncTime = time.Time{}
	s.mu.Unlock()

	result, err := s.PerformSync(ctx, repository)

	// Restore original time if sync failed
	if err != nil {
		s.mu.Lock()
		s.syncState.LastSyncTime = originalTime
		s.mu.Unlock()
	}

	return result, err
}

// GetPendingChanges returns the number of local changes that need to be synced
func (s *BatchSyncService) GetPendingChanges(ctx context.Context, repository string) (int, error) {
	modifiedTasks, err := s.getModifiedLocalTasks(ctx, repository)
	if err != nil {
		return 0, err
	}
	return len(modifiedTasks), nil
}

// ScheduleAutoSync starts automatic synchronization at regular intervals
func (s *BatchSyncService) ScheduleAutoSync(ctx context.Context, repository string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Info("automatic sync scheduled",
		slog.String("repository", repository),
		slog.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("automatic sync stopped")
			return
		case <-ticker.C:
			if result, err := s.PerformSync(ctx, repository); err != nil {
				s.logger.Warn("automatic sync failed",
					slog.String("repository", repository),
					slog.Any("error", err))
			} else {
				s.logger.Debug("automatic sync completed",
					slog.String("repository", repository),
					slog.Int("synced_tasks", result.SyncedTasks),
					slog.Int("conflicts", result.ConflictsDetected))
			}
		}
	}
}
