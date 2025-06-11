// Package sync provides conflict resolution for real-time synchronization.
package sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"lerian-mcp-memory-cli/internal/domain/entities"
)

// SimpleConflictResolver provides basic conflict resolution strategies
type SimpleConflictResolver struct {
	logger   *slog.Logger
	strategy ConflictStrategy
}

// ConflictStrategy defines how conflicts should be resolved
type ConflictStrategy string

const (
	// StrategyLastWriteWins uses timestamp to determine the winner
	StrategyLastWriteWins ConflictStrategy = "last_write_wins"
	// StrategyLocalWins always favors the local version
	StrategyLocalWins ConflictStrategy = "local_wins"
	// StrategyRemoteWins always favors the remote version
	StrategyRemoteWins ConflictStrategy = "remote_wins"
	// StrategyMerge attempts to merge non-conflicting fields
	StrategyMerge ConflictStrategy = "merge"
)

// NewSimpleConflictResolver creates a new conflict resolver with the specified strategy
func NewSimpleConflictResolver(strategy ConflictStrategy, logger *slog.Logger) *SimpleConflictResolver {
	return &SimpleConflictResolver{
		logger:   logger,
		strategy: strategy,
	}
}

// DetectConflict determines if there's a conflict between local and remote tasks
func (r *SimpleConflictResolver) DetectConflict(localTask, remoteTask *entities.Task) bool {
	if localTask == nil || remoteTask == nil {
		return false
	}

	// Consider it a conflict if both have been updated and have different content
	if localTask.UpdatedAt.After(remoteTask.CreatedAt) &&
		remoteTask.UpdatedAt.After(localTask.CreatedAt) {

		// Check if key fields differ
		if localTask.Content != remoteTask.Content ||
			localTask.Type != remoteTask.Type ||
			localTask.Status != remoteTask.Status ||
			localTask.Priority != remoteTask.Priority {
			return true
		}

		// Check if tags differ
		if !r.equalStringSlices(localTask.Tags, remoteTask.Tags) {
			return true
		}

		// Check if metadata differs (simplified check)
		if len(localTask.Metadata) != len(remoteTask.Metadata) {
			return true
		}
	}

	return false
}

// ResolveConflict resolves conflicts between local and remote tasks
func (r *SimpleConflictResolver) ResolveConflict(ctx context.Context, localTask, remoteTask *entities.Task) (*entities.Task, error) {
	r.logger.Info("Resolving task conflict",
		"strategy", string(r.strategy),
		"task_id", localTask.ID,
		"local_updated", localTask.UpdatedAt,
		"remote_updated", remoteTask.UpdatedAt,
	)

	switch r.strategy {
	case StrategyLastWriteWins:
		return r.resolveLastWriteWins(localTask, remoteTask), nil
	case StrategyLocalWins:
		return localTask, nil
	case StrategyRemoteWins:
		return remoteTask, nil
	case StrategyMerge:
		return r.resolveMerge(localTask, remoteTask), nil
	default:
		return nil, fmt.Errorf("unknown conflict strategy: %s", r.strategy)
	}
}

// resolveLastWriteWins returns the task that was updated most recently
func (r *SimpleConflictResolver) resolveLastWriteWins(localTask, remoteTask *entities.Task) *entities.Task {
	if remoteTask.UpdatedAt.After(localTask.UpdatedAt) {
		r.logger.Info("Conflict resolved: remote wins (newer)",
			"task_id", localTask.ID,
			"remote_updated", remoteTask.UpdatedAt,
			"local_updated", localTask.UpdatedAt,
		)
		return remoteTask
	} else {
		r.logger.Info("Conflict resolved: local wins (newer)",
			"task_id", localTask.ID,
			"local_updated", localTask.UpdatedAt,
			"remote_updated", remoteTask.UpdatedAt,
		)
		return localTask
	}
}

// resolveMerge attempts to merge non-conflicting fields intelligently
func (r *SimpleConflictResolver) resolveMerge(localTask, remoteTask *entities.Task) *entities.Task {
	merged := &entities.Task{
		ID:         localTask.ID,
		Repository: localTask.Repository,
		CreatedAt:  localTask.CreatedAt,
		UpdatedAt:  time.Now(), // Set to current time since we're creating a new version
	}

	// Use the most recent update time for UpdatedAt
	if remoteTask.UpdatedAt.After(localTask.UpdatedAt) {
		merged.UpdatedAt = remoteTask.UpdatedAt
	} else {
		merged.UpdatedAt = localTask.UpdatedAt
	}

	// For key fields, use last-write-wins approach
	if remoteTask.UpdatedAt.After(localTask.UpdatedAt) {
		merged.Content = remoteTask.Content
		merged.Type = remoteTask.Type
		merged.Status = remoteTask.Status
		merged.Priority = remoteTask.Priority
		merged.EstimatedMins = remoteTask.EstimatedMins
	} else {
		merged.Content = localTask.Content
		merged.Type = localTask.Type
		merged.Status = localTask.Status
		merged.Priority = localTask.Priority
		merged.EstimatedMins = localTask.EstimatedMins
	}

	// For arrays, merge uniquely
	merged.Tags = r.mergeStringSlices(localTask.Tags, remoteTask.Tags)

	// Use the higher actual minutes (if any)
	if remoteTask.ActualMins > localTask.ActualMins {
		merged.ActualMins = remoteTask.ActualMins
	} else {
		merged.ActualMins = localTask.ActualMins
	}

	// Merge metadata (prefer local, add remote keys that don't exist locally)
	merged.Metadata = make(map[string]interface{})
	if localTask.Metadata != nil {
		for k, v := range localTask.Metadata {
			merged.Metadata[k] = v
		}
	}
	if remoteTask.Metadata != nil {
		for k, v := range remoteTask.Metadata {
			if _, exists := merged.Metadata[k]; !exists {
				merged.Metadata[k] = v
			}
		}
	}

	// Merge session IDs (prefer local)
	if localTask.SessionID != "" {
		merged.SessionID = localTask.SessionID
	} else {
		merged.SessionID = remoteTask.SessionID
	}

	r.logger.Info("Conflict resolved: merged",
		"task_id", merged.ID,
		"merged_content", merged.Content,
		"merged_status", string(merged.Status),
		"merged_tags_count", len(merged.Tags),
	)

	return merged
}

// equalStringSlices checks if two string slices are equal (order doesn't matter)
func (r *SimpleConflictResolver) equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Create a map to count occurrences
	counts := make(map[string]int)
	for _, s := range a {
		counts[s]++
	}

	for _, s := range b {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}

	for _, count := range counts {
		if count != 0 {
			return false
		}
	}

	return true
}

// mergeStringSlices merges two string slices, removing duplicates
func (r *SimpleConflictResolver) mergeStringSlices(a, b []string) []string {
	seen := make(map[string]bool)
	var result []string

	// Add all items from both slices, avoiding duplicates
	for _, s := range a {
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	for _, s := range b {
		if s != "" && !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}

// SetStrategy changes the conflict resolution strategy
func (r *SimpleConflictResolver) SetStrategy(strategy ConflictStrategy) {
	r.strategy = strategy
	r.logger.Info("Conflict resolution strategy changed", "strategy", string(strategy))
}
