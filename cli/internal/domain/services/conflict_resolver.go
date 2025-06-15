// Package services provides conflict resolution logic using Qdrant as source of truth
package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"lerian-mcp-memory-cli/internal/adapters/secondary/api"
	"lerian-mcp-memory-cli/internal/domain/entities"
	"lerian-mcp-memory-cli/internal/domain/ports"
)

// ConflictResolver handles task conflicts using Qdrant as authoritative source
type ConflictResolver struct {
	storage    ports.Storage
	mcpClient  ports.MCPClient
	logger     *slog.Logger
	strategies map[api.ConflictType]ResolutionHandler
}

// ResolutionHandler defines a function that handles specific conflict types
type ResolutionHandler func(ctx context.Context, local, server *api.TaskSyncItem) (*api.ConflictResolution, error)

// ConflictAnalysis provides detailed analysis of a conflict
type ConflictAnalysis struct {
	ConflictTypes       []api.ConflictType     `json:"conflict_types"`
	Severity            ConflictSeverity       `json:"severity"`
	RecommendedStrategy api.ResolutionStrategy `json:"recommended_strategy"`
	Confidence          float64                `json:"confidence"`
	AutoResolvable      bool                   `json:"auto_resolvable"`
	Reasoning           string                 `json:"reasoning"`
}

// ConflictSeverity defines the severity level of a conflict
type ConflictSeverity string

const (
	SeverityLow      ConflictSeverity = "low"
	SeverityMedium   ConflictSeverity = "medium"
	SeverityHigh     ConflictSeverity = "high"
	SeverityCritical ConflictSeverity = "critical"
)

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(storage ports.Storage, mcpClient ports.MCPClient, logger *slog.Logger) *ConflictResolver {
	resolver := &ConflictResolver{
		storage:    storage,
		mcpClient:  mcpClient,
		logger:     logger,
		strategies: make(map[api.ConflictType]ResolutionHandler),
	}

	// Register resolution strategies for different conflict types
	resolver.registerStrategies()

	return resolver
}

// registerStrategies sets up resolution handlers for different conflict types
func (r *ConflictResolver) registerStrategies() {
	r.strategies[api.ConflictTypeContent] = r.resolveContentConflict
	r.strategies[api.ConflictTypeStatus] = r.resolveStatusConflict
	r.strategies[api.ConflictTypePriority] = r.resolvePriorityConflict
	r.strategies[api.ConflictTypeTimestamp] = r.resolveTimestampConflict
	r.strategies[api.ConflictTypeMetadata] = r.resolveMetadataConflict
	r.strategies[api.ConflictTypeStructural] = r.resolveStructuralConflict
}

// DetectConflict identifies and analyzes conflicts between local and server tasks
func (r *ConflictResolver) DetectConflict(ctx context.Context, local, server *api.TaskSyncItem) *api.ConflictItem {
	if local.ID != server.ID {
		r.logger.Warn("cannot compare tasks with different IDs",
			slog.String("local_id", local.ID),
			slog.String("server_id", server.ID))
		return nil
	}

	conflict := &api.ConflictItem{
		TaskID:     local.ID,
		LocalTask:  local,
		ServerTask: server,
	}

	// Determine conflict types
	conflictTypes := r.getConflictTypes(local, server)
	if len(conflictTypes) == 0 {
		return nil // No conflict
	}

	conflict.ConflictType = conflictTypes[0] // Primary conflict type

	// Analyze the conflict
	analysis := r.analyzeConflict(ctx, local, server, conflictTypes)

	// Generate resolution
	resolution := r.generateResolution(ctx, local, server, analysis)

	conflict.Resolution = *resolution
	conflict.Reason = analysis.Reasoning

	r.logger.Info("conflict detected and resolved",
		slog.String("task_id", local.ID),
		slog.String("strategy", string(resolution.Strategy)),
		slog.Float64("confidence", resolution.Confidence))

	return conflict
}

// getConflictTypes identifies all types of conflicts between two tasks
func (r *ConflictResolver) getConflictTypes(local, server *api.TaskSyncItem) []api.ConflictType {
	var types []api.ConflictType

	if local.Content != server.Content {
		types = append(types, api.ConflictTypeContent)
	}

	if local.Status != server.Status {
		types = append(types, api.ConflictTypeStatus)
	}

	if local.Priority != server.Priority {
		types = append(types, api.ConflictTypePriority)
	}

	if !local.UpdatedAt.Equal(server.UpdatedAt) {
		types = append(types, api.ConflictTypeTimestamp)
	}

	if r.hasMetadataConflict(local, server) {
		types = append(types, api.ConflictTypeMetadata)
	}

	return types
}

// hasMetadataConflict checks if metadata differs between tasks
func (r *ConflictResolver) hasMetadataConflict(local, server *api.TaskSyncItem) bool {
	if len(local.Metadata) != len(server.Metadata) {
		return true
	}

	for key, localVal := range local.Metadata {
		if serverVal, exists := server.Metadata[key]; !exists || localVal != serverVal {
			return true
		}
	}

	return false
}

// analyzeConflict provides detailed analysis of the conflict
func (r *ConflictResolver) analyzeConflict(ctx context.Context, local, server *api.TaskSyncItem, conflictTypes []api.ConflictType) *ConflictAnalysis {
	analysis := &ConflictAnalysis{
		ConflictTypes: conflictTypes,
	}

	// Determine severity
	analysis.Severity = r.calculateSeverity(conflictTypes, local, server)

	// Determine if auto-resolvable
	analysis.AutoResolvable = r.isAutoResolvable(conflictTypes, local, server)

	// Generate reasoning
	analysis.Reasoning = r.generateReasoning(conflictTypes, local, server)

	// Recommend strategy
	analysis.RecommendedStrategy = r.recommendStrategy(ctx, local, server, conflictTypes)

	// Calculate confidence
	analysis.Confidence = r.calculateConfidence(analysis.RecommendedStrategy, conflictTypes)

	return analysis
}

// calculateSeverity determines how severe the conflict is
func (r *ConflictResolver) calculateSeverity(conflictTypes []api.ConflictType, local, server *api.TaskSyncItem) ConflictSeverity {
	// Multiple conflict types = higher severity
	if len(conflictTypes) >= 3 {
		return SeverityCritical
	}

	// Check for critical conflicts
	for _, conflictType := range conflictTypes {
		switch conflictType {
		case api.ConflictTypeStructural:
			return SeverityCritical
		case api.ConflictTypeContent:
			// Large content differences are high severity
			if r.calculateContentDifference(local.Content, server.Content) > 0.7 {
				return SeverityHigh
			}
		}
	}

	if len(conflictTypes) == 2 {
		return SeverityMedium
	}

	return SeverityLow
}

// calculateContentDifference returns a ratio of how different two content strings are
func (r *ConflictResolver) calculateContentDifference(local, server string) float64 {
	if local == server {
		return 0.0
	}

	if local == "" || server == "" {
		return 1.0
	}

	// Simple difference calculation based on length and common substrings
	longer := len(local)
	if len(server) > longer {
		longer = len(server)
	}

	// Find common substring length
	common := r.longestCommonSubstring(local, server)

	return 1.0 - (float64(common) / float64(longer))
}

// longestCommonSubstring finds the length of the longest common substring
func (r *ConflictResolver) longestCommonSubstring(s1, s2 string) int {
	m, n := len(s1), len(s2)
	if m == 0 || n == 0 {
		return 0
	}

	// Simple implementation - could be optimized
	maxLen := 0
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			length := 0
			for k := 0; i+k < m && j+k < n && s1[i+k] == s2[j+k]; k++ {
				length++
			}
			if length > maxLen {
				maxLen = length
			}
		}
	}

	return maxLen
}

// isAutoResolvable determines if the conflict can be automatically resolved
func (r *ConflictResolver) isAutoResolvable(conflictTypes []api.ConflictType, local, server *api.TaskSyncItem) bool {
	// Never auto-resolve structural conflicts
	for _, conflictType := range conflictTypes {
		if conflictType == api.ConflictTypeStructural {
			return false
		}
	}

	// Don't auto-resolve if there are too many conflicts
	if len(conflictTypes) > 2 {
		return false
	}

	// Check for safe auto-resolvable scenarios
	timeDiff := local.UpdatedAt.Sub(server.UpdatedAt)
	if timeDiff.Abs() > time.Hour {
		return true // Clear time separation
	}

	return false
}

// generateReasoning creates human-readable reasoning for the conflict
func (r *ConflictResolver) generateReasoning(conflictTypes []api.ConflictType, local, server *api.TaskSyncItem) string {
	var reasons []string

	for _, conflictType := range conflictTypes {
		switch conflictType {
		case api.ConflictTypeContent:
			reasons = append(reasons, "content differs between local and server")
		case api.ConflictTypeStatus:
			reasons = append(reasons, fmt.Sprintf("status conflict: local=%s, server=%s", local.Status, server.Status))
		case api.ConflictTypePriority:
			reasons = append(reasons, fmt.Sprintf("priority conflict: local=%s, server=%s", local.Priority, server.Priority))
		case api.ConflictTypeTimestamp:
			reasons = append(reasons, "tasks have different update timestamps")
		case api.ConflictTypeMetadata:
			reasons = append(reasons, "metadata differs between versions")
		case api.ConflictTypeStructural:
			reasons = append(reasons, "structural changes detected")
		}
	}

	return strings.Join(reasons, "; ")
}

// recommendStrategy suggests the best resolution strategy
func (r *ConflictResolver) recommendStrategy(ctx context.Context, local, server *api.TaskSyncItem, conflictTypes []api.ConflictType) api.ResolutionStrategy {
	// Try Qdrant-based resolution first
	if r.mcpClient != nil {
		if strategy := r.tryQdrantResolution(ctx, local, server); strategy != "" {
			return strategy
		}
	}

	// Fallback to timestamp-based resolution
	if local.UpdatedAt.After(server.UpdatedAt) {
		return api.StrategyLocalWinsNewer
	} else if server.UpdatedAt.After(local.UpdatedAt) {
		return api.StrategyServerWinsNewer
	}

	// For same timestamp, check conflict types
	for _, conflictType := range conflictTypes {
		switch conflictType {
		case api.ConflictTypeStatus:
			// Prefer status progression
			if r.isStatusProgression(server.Status, local.Status) {
				return api.StrategyLocalWins
			}
			return api.StrategyServerWins

		case api.ConflictTypePriority:
			// Prefer higher priority
			if r.isPriorityHigher(local.Priority, server.Priority) {
				return api.StrategyLocalWins
			}
			return api.StrategyServerWins

		case api.ConflictTypeContent:
			// Try merge for content conflicts
			return api.StrategyMerge
		}
	}

	// Default to server wins
	return api.StrategyServerWins
}

// tryQdrantResolution attempts to resolve conflict using Qdrant as source of truth
func (r *ConflictResolver) tryQdrantResolution(ctx context.Context, local, server *api.TaskSyncItem) api.ResolutionStrategy {
	// This would query Qdrant to find the authoritative version
	// For now, simulate by checking if we can find the task in storage
	task, err := r.storage.GetTask(ctx, local.ID)
	if err != nil {
		r.logger.Debug("could not find task in storage for Qdrant resolution",
			slog.String("task_id", local.ID),
			slog.Any("error", err))
		return ""
	}

	// Compare checksums to determine truth
	localChecksum := local.GenerateChecksum()
	serverChecksum := server.GenerateChecksum()

	// Create checksum for storage task
	storageItem := api.FromTask(task)
	storageChecksum := storageItem.GenerateChecksum()

	switch storageChecksum {
	case serverChecksum:
		return api.StrategyQdrantTruth // Server matches authoritative source
	case localChecksum:
		return api.StrategyLocalWins // Local matches authoritative source
	}

	return "" // No clear match, use other strategies
}

// calculateConfidence calculates confidence level for a strategy
func (r *ConflictResolver) calculateConfidence(strategy api.ResolutionStrategy, conflictTypes []api.ConflictType) float64 {
	baseConfidence := 0.5

	switch strategy {
	case api.StrategyQdrantTruth:
		baseConfidence = 0.95
	case api.StrategyServerWinsNewer, api.StrategyLocalWinsNewer:
		baseConfidence = 0.85
	case api.StrategyMerge:
		baseConfidence = 0.70
	case api.StrategyServerWins, api.StrategyLocalWins:
		baseConfidence = 0.60
	case api.StrategyManual:
		baseConfidence = 0.0
	}

	// Reduce confidence based on conflict complexity
	complexityPenalty := float64(len(conflictTypes)) * 0.05
	confidence := baseConfidence - complexityPenalty

	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}

// generateResolution creates the final conflict resolution
func (r *ConflictResolver) generateResolution(_ context.Context, local, server *api.TaskSyncItem, analysis *ConflictAnalysis) *api.ConflictResolution {
	resolution := &api.ConflictResolution{
		Strategy:   analysis.RecommendedStrategy,
		Confidence: analysis.Confidence,
		AutoApply:  analysis.AutoResolvable && analysis.Confidence >= 0.8,
	}

	// Generate the resolved task based on strategy
	switch analysis.RecommendedStrategy {
	case api.StrategyServerWins, api.StrategyServerWinsNewer, api.StrategyQdrantTruth:
		resolution.ResolvedTask = server

	case api.StrategyLocalWins, api.StrategyLocalWinsNewer:
		resolution.ResolvedTask = local

	case api.StrategyMerge:
		merged := r.mergeTask(local, server)
		resolution.ResolvedTask = merged

	case api.StrategyManual:
		// For manual resolution, provide both options
		resolution.ResolvedTask = server // Default to server, but mark for manual review
		resolution.AutoApply = false

	default:
		resolution.ResolvedTask = server
	}

	return resolution
}

// mergeTask attempts to intelligently merge two conflicting tasks
func (r *ConflictResolver) mergeTask(local, server *api.TaskSyncItem) *api.TaskSyncItem {
	merged := *server // Start with server as base

	// Status: prefer progression
	if r.isStatusProgression(server.Status, local.Status) {
		merged.Status = local.Status
	}

	// Priority: prefer higher priority
	if r.isPriorityHigher(local.Priority, server.Priority) {
		merged.Priority = local.Priority
	}

	// Content: if local has additional content, use it
	if len(local.Content) > len(server.Content) &&
		strings.Contains(local.Content, server.Content) {
		merged.Content = local.Content
	}

	// Tags: merge unique tags
	tagSet := make(map[string]bool)
	for _, tag := range server.Tags {
		tagSet[tag] = true
	}
	for _, tag := range local.Tags {
		tagSet[tag] = true
	}

	merged.Tags = make([]string, 0, len(tagSet))
	for tag := range tagSet {
		merged.Tags = append(merged.Tags, tag)
	}

	// Metadata: merge non-conflicting entries
	if merged.Metadata == nil {
		merged.Metadata = make(map[string]interface{})
	}
	for key, value := range local.Metadata {
		if _, exists := merged.Metadata[key]; !exists {
			merged.Metadata[key] = value
		}
	}

	// Update timestamp and checksum
	merged.UpdatedAt = time.Now()
	merged.UpdateChecksum()

	return &merged
}

// isStatusProgression checks if the transition from old to new status is a progression
func (r *ConflictResolver) isStatusProgression(oldStatus, newStatus entities.Status) bool {
	progressions := map[entities.Status][]entities.Status{
		entities.StatusPending:    {entities.StatusInProgress, entities.StatusCompleted},
		entities.StatusInProgress: {entities.StatusCompleted, entities.StatusCancelled},
	}

	validNext, exists := progressions[oldStatus]
	if !exists {
		return false
	}

	for _, status := range validNext {
		if status == newStatus {
			return true
		}
	}

	return false
}

// isPriorityHigher checks if priority1 is higher than priority2
func (r *ConflictResolver) isPriorityHigher(priority1, priority2 entities.Priority) bool {
	priorityOrder := map[entities.Priority]int{
		entities.PriorityLow:    1,
		entities.PriorityMedium: 2,
		entities.PriorityHigh:   3,
	}

	return priorityOrder[priority1] > priorityOrder[priority2]
}

// Individual conflict type handlers

func (r *ConflictResolver) resolveContentConflict(ctx context.Context, local, server *api.TaskSyncItem) (*api.ConflictResolution, error) {
	// Try intelligent content merge
	merged := r.mergeTask(local, server)

	return &api.ConflictResolution{
		Strategy:     api.StrategyMerge,
		ResolvedTask: merged,
		Confidence:   0.75,
	}, nil
}

func (r *ConflictResolver) resolveStatusConflict(ctx context.Context, local, server *api.TaskSyncItem) (*api.ConflictResolution, error) {
	if r.isStatusProgression(server.Status, local.Status) {
		return &api.ConflictResolution{
			Strategy:     api.StrategyLocalWins,
			ResolvedTask: local,
			Confidence:   0.9,
		}, nil
	}

	return &api.ConflictResolution{
		Strategy:     api.StrategyServerWins,
		ResolvedTask: server,
		Confidence:   0.8,
	}, nil
}

func (r *ConflictResolver) resolvePriorityConflict(ctx context.Context, local, server *api.TaskSyncItem) (*api.ConflictResolution, error) {
	if r.isPriorityHigher(local.Priority, server.Priority) {
		return &api.ConflictResolution{
			Strategy:     api.StrategyLocalWins,
			ResolvedTask: local,
			Confidence:   0.85,
		}, nil
	}

	return &api.ConflictResolution{
		Strategy:     api.StrategyServerWins,
		ResolvedTask: server,
		Confidence:   0.85,
	}, nil
}

func (r *ConflictResolver) resolveTimestampConflict(ctx context.Context, local, server *api.TaskSyncItem) (*api.ConflictResolution, error) {
	if local.UpdatedAt.After(server.UpdatedAt) {
		return &api.ConflictResolution{
			Strategy:     api.StrategyLocalWinsNewer,
			ResolvedTask: local,
			Confidence:   0.9,
		}, nil
	}

	return &api.ConflictResolution{
		Strategy:     api.StrategyServerWinsNewer,
		ResolvedTask: server,
		Confidence:   0.9,
	}, nil
}

func (r *ConflictResolver) resolveMetadataConflict(ctx context.Context, local, server *api.TaskSyncItem) (*api.ConflictResolution, error) {
	merged := r.mergeTask(local, server)

	return &api.ConflictResolution{
		Strategy:     api.StrategyMerge,
		ResolvedTask: merged,
		Confidence:   0.8,
	}, nil
}

func (r *ConflictResolver) resolveStructuralConflict(ctx context.Context, local, server *api.TaskSyncItem) (*api.ConflictResolution, error) {
	// Structural conflicts require manual resolution
	return &api.ConflictResolution{
		Strategy:     api.StrategyManual,
		ResolvedTask: server, // Provide server as default
		Confidence:   0.0,
		AutoApply:    false,
	}, nil
}
